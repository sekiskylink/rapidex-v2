import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearAuthSnapshot, setAuthSnapshot } from '../auth/state'
import { API_BASE_URL_OVERRIDE_STORAGE_KEY } from '../lib/apiBaseUrl'
import { createAppRouter } from '../routes'
import { SnackbarProvider } from '../ui/snackbar'

function renderCellValue(column: Record<string, any>, row: Record<string, any>) {
  if (typeof column.renderCell === 'function') {
    return column.renderCell({
      row,
      field: column.field,
      value: row[column.field],
      colDef: column,
      id: row.id,
    })
  }
  if (typeof column.valueGetter === 'function') {
    return column.valueGetter(row[column.field], row, column, null)
  }
  const value = row[column.field]
  return value === undefined || value === null ? '' : String(value)
}

vi.mock('@mui/x-data-grid', () => ({
  DataGrid: (props: Record<string, any>) => (
    <div>
      <div>
        {props.columns.map((column: Record<string, any>) => (
          <span key={column.field}>{column.headerName}</span>
        ))}
      </div>
      {props.rows.map((row: Record<string, any>) => (
        <div key={String(row.id)}>
          {props.columns.map((column: Record<string, any>) => (
            <div key={column.field}>{renderCellValue(column, row)}</div>
          ))}
        </div>
      ))}
    </div>
  ),
}))

function renderRoute(path: string) {
  const router = createAppRouter([path])
  const queryClient = new QueryClient()
  return render(
    <QueryClientProvider client={queryClient}>
      <SnackbarProvider>
        <RouterProvider router={router} />
      </SnackbarProvider>
    </QueryClientProvider>,
  )
}

function authenticate(permissions: string[] = ['users.read', 'audit.read', 'settings.read']) {
  setAuthSnapshot({
    isAuthenticated: true,
    accessToken: 'access-token',
    refreshToken: 'refresh-token',
    user: {
      id: 1,
      username: 'admin',
      roles: ['Admin'],
      permissions,
    },
  })
}

describe('users and audit pages', () => {
  beforeEach(() => {
    window.localStorage.clear()
    clearAuthSnapshot()
    vi.stubEnv('VITE_API_BASE_URL', 'http://localhost:8080/api/v1')
    window.localStorage.setItem(API_BASE_URL_OVERRIDE_STORAGE_KEY, 'http://localhost:8080/api/v1')
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })),
    )
  })

  afterEach(() => {
    cleanup()
    window.localStorage.clear()
    clearAuthSnapshot()
    vi.unstubAllEnvs()
    vi.unstubAllGlobals()
  })

  it('/users renders metadata columns and values from mocked API', async () => {
    authenticate(['users.read', 'audit.read', 'settings.read', 'users.write'])
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          items: [
            {
              id: 10,
              username: 'alice',
              firstName: 'Alice',
              lastName: 'Johnson',
              displayName: '',
              language: 'English',
              email: 'alice@example.com',
              phoneNumber: '+15551234567',
              whatsappNumber: '+15557654321',
              telegramHandle: '@alice',
              isActive: true,
              updatedAt: '2026-03-01T12:00:00Z',
              createdAt: '2026-03-01T10:00:00Z',
            },
          ],
          totalCount: 1,
          page: 1,
          pageSize: 25,
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    )

    renderRoute('/users')

    expect(await screen.findByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument()
    expect(screen.getByText('Display Name')).toBeInTheDocument()
    expect(screen.getByText('Language')).toBeInTheDocument()
    expect(screen.getByText('Email')).toBeInTheDocument()
    expect(screen.getByText('Phone Number')).toBeInTheDocument()
    expect(screen.getByText('WhatsApp Number')).toBeInTheDocument()
    expect(screen.getByText('Telegram Handle')).toBeInTheDocument()
    expect(screen.getByText('Updated')).toBeInTheDocument()
    expect(await screen.findByText('Alice Johnson')).toBeInTheDocument()
    expect(screen.getByText('alice@example.com')).toBeInTheDocument()
    expect(screen.getByText('+15551234567')).toBeInTheDocument()
    expect(await screen.findByText('alice')).toBeInTheDocument()
    await waitFor(() => expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/users?'), expect.anything()))
  })

  it('create user submits expected metadata payload', async () => {
    authenticate(['users.read', 'users.write', 'audit.read'])
    let createPayload: Record<string, unknown> | null = null
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('/users?') && (!init?.method || init.method === 'GET')) {
        return new Response(
          JSON.stringify({
            items: [],
            totalCount: 0,
            page: 1,
            pageSize: 25,
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.endsWith('/users') && init?.method === 'POST') {
        createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return new Response(JSON.stringify({ id: 99, username: 'new-user' }), {
          status: 201,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
    })

    renderRoute('/users')
    expect(await screen.findByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Create User' }))

    const dialog = await screen.findByRole('dialog', { name: 'Create User' })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Username' }), { target: { value: 'new-user' } })
    const passwordInput = dialog.querySelector('input[type="password"]')
    expect(passwordInput).not.toBeNull()
    fireEvent.change(passwordInput as Element, { target: { value: 'TempPass123!' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Email' }), { target: { value: 'new-user@example.com' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Language' }), { target: { value: 'French' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'First Name' }), { target: { value: 'New' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Last Name' }), { target: { value: 'User' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Display Name' }), { target: { value: 'New User' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Phone Number' }), { target: { value: '+15550000001' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'WhatsApp Number' }), { target: { value: '+15550000002' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Telegram Handle' }), { target: { value: '@new_user' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Create' }))

    await waitFor(() => expect(createPayload).not.toBeNull())
    expect(createPayload).toMatchObject({
      username: 'new-user',
      password: 'TempPass123!',
      email: 'new-user@example.com',
      language: 'French',
      firstName: 'New',
      lastName: 'User',
      displayName: 'New User',
      phoneNumber: '+15550000001',
      whatsappNumber: '+15550000002',
      telegramHandle: '@new_user',
      isActive: true,
    })
  })

  it('edit user submits metadata payload and omits password when empty', async () => {
    authenticate(['users.read', 'users.write', 'audit.read'])
    let patchPayload: Record<string, unknown> | null = null
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('/users?') && (!init?.method || init.method === 'GET')) {
        return new Response(
          JSON.stringify({
            items: [
              {
                id: 7,
                username: 'jane',
                firstName: 'Jane',
                lastName: 'Doe',
                displayName: 'Jane Doe',
                language: 'English',
                email: 'jane@example.com',
                phoneNumber: '+15551234567',
                whatsappNumber: '+15551234568',
                telegramHandle: '@janedoe',
                isActive: true,
                updatedAt: '2026-03-01T12:00:00Z',
                createdAt: '2026-03-01T10:00:00Z',
              },
            ],
            totalCount: 1,
            page: 1,
            pageSize: 25,
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.endsWith('/users/7') && init?.method === 'PATCH') {
        patchPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return new Response(JSON.stringify({ id: 7, username: 'jane' }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
    })

    renderRoute('/users')
    expect(await screen.findByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument()
    fireEvent.click(await screen.findByRole('button', { name: 'Edit' }))

    const dialog = await screen.findByRole('dialog', { name: 'Edit User' })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Email' }), { target: { value: 'jane-updated@example.com' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Display Name' }), { target: { value: 'Jane Updated' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Phone Number' }), { target: { value: '+15559876543' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(patchPayload).not.toBeNull())
    expect(patchPayload).toMatchObject({
      username: 'jane',
      email: 'jane-updated@example.com',
      language: 'English',
      firstName: 'Jane',
      lastName: 'Doe',
      displayName: 'Jane Updated',
      phoneNumber: '+15559876543',
      whatsappNumber: '+15551234568',
      telegramHandle: '@janedoe',
      isActive: true,
    })
    expect(patchPayload).not.toHaveProperty('password')
  })

  it('validation error displays field messages and request ID', async () => {
    authenticate(['users.read', 'users.write'])
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('/users?') && (!init?.method || init.method === 'GET')) {
        return new Response(
          JSON.stringify({
            items: [],
            totalCount: 0,
            page: 1,
            pageSize: 25,
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.endsWith('/users') && init?.method === 'POST') {
        return new Response(
          JSON.stringify({
            error: {
              code: 'VALIDATION_ERROR',
              message: 'validation failed',
              details: {
                email: ['must be a valid email address'],
                phoneNumber: ['must be E.164 format, e.g. +15551234567'],
              },
            },
          }),
          {
            status: 400,
            headers: {
              'Content-Type': 'application/json',
              'X-Request-Id': 'req-users-422',
            },
          },
        )
      }
      return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
    })

    renderRoute('/users')
    expect(await screen.findByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Create User' }))

    const dialog = await screen.findByRole('dialog', { name: 'Create User' })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Username' }), { target: { value: 'bad-user' } })
    const passwordInput = dialog.querySelector('input[type="password"]')
    expect(passwordInput).not.toBeNull()
    fireEvent.change(passwordInput as Element, { target: { value: 'TempPass123!' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Email' }), { target: { value: 'not-an-email' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Phone Number' }), { target: { value: '123' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Create' }))

    expect(await within(dialog).findByText('must be a valid email address')).toBeInTheDocument()
    expect(within(dialog).getByText('must be E.164 format, e.g. +15551234567')).toBeInTheDocument()
    expect(within(dialog).getByText('validation failed Request ID: req-users-422')).toBeInTheDocument()
  })

  it('/audit renders mocked API rows', async () => {
    authenticate()
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          items: [{ id: 20, timestamp: new Date().toISOString(), action: 'auth.login.success' }],
          totalCount: 1,
          page: 1,
          pageSize: 25,
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    )

    renderRoute('/audit')

    expect(await screen.findByRole('heading', { name: 'Audit', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('auth.login.success')).toBeInTheDocument()
    await waitFor(() => expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/audit?'), expect.anything()))
  })
})
