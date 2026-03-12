import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { afterEach, beforeEach, describe, expect, it, vi, type MockInstance } from 'vitest'
import { clearAuthSnapshot, setAuthSnapshot } from '../auth/state'
import { API_BASE_URL_OVERRIDE_STORAGE_KEY } from '../lib/apiBaseUrl'
import * as api from '../lib/api'
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
  DataGrid: (props: Record<string, any>) => {
    const columns = Array.isArray(props.columns) ? props.columns : []
    const rows = Array.isArray(props.rows) ? props.rows : []
    return (
      <div>
        <div>
          {columns.map((column: Record<string, any>) => (
            <span key={column.field}>{column.headerName}</span>
          ))}
        </div>
        {rows.map((row: Record<string, any>) => (
          <div key={String(row.id)}>
            {columns.map((column: Record<string, any>) => (
              <div key={column.field}>{renderCellValue(column, row)}</div>
            ))}
          </div>
        ))}
      </div>
    )
  },
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

function authenticate(permissions: string[]) {
  setAuthSnapshot({
    isAuthenticated: true,
    accessToken: 'access-token',
    refreshToken: 'refresh-token',
    user: {
      id: 7,
      username: 'operator',
      roles: ['Staff'],
      permissions,
    },
  })
}

describe('servers page', () => {
  let apiRequestSpy: MockInstance

  beforeEach(() => {
    window.localStorage.clear()
    clearAuthSnapshot()
    vi.stubEnv('VITE_API_BASE_URL', 'http://localhost:8080/api/v1')
    window.localStorage.setItem(API_BASE_URL_OVERRIDE_STORAGE_KEY, 'http://localhost:8080/api/v1')
    apiRequestSpy = vi.spyOn(api, 'apiRequest')
  })

  afterEach(() => {
    cleanup()
    clearAuthSnapshot()
    vi.unstubAllEnvs()
    apiRequestSpy.mockRestore()
  })

  it('renders servers grid rows from mocked API', async () => {
    authenticate(['servers.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path.includes('/servers?')) {
        return {
          items: [
            {
              id: 11,
              uid: 'uid-11',
              name: 'DHIS2 Uganda',
              code: 'dhis2-ug',
              systemType: 'dhis2',
              baseUrl: 'https://dhis.example.com',
              endpointType: 'http',
              httpMethod: 'POST',
              useAsync: true,
              parseResponses: true,
              headers: {},
              urlParams: {},
              suspended: false,
              createdAt: '2026-03-10T09:00:00Z',
              updatedAt: '2026-03-10T10:00:00Z',
            },
          ],
          totalCount: 1,
          page: 1,
          pageSize: 25,
        }
      }
      return {}
    })

    renderRoute('/servers')

    expect(await screen.findByRole('heading', { name: 'Servers', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('DHIS2 Uganda')).toBeInTheDocument()
    expect(screen.getByText('dhis2-ug')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Toggle Sukumad menu' })).toBeInTheDocument()
  })

  it('create server submits payload through API', async () => {
    authenticate(['servers.read', 'servers.write'])
    let createPayload: Record<string, unknown> | null = null
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/servers?')) {
        return { items: [], totalCount: 0, page: 1, pageSize: 25 }
      }
      if (path === '/servers' && init?.method === 'POST') {
        createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { id: 22 }
      }
      return {}
    })

    renderRoute('/servers')

    fireEvent.click(await screen.findByRole('button', { name: 'Create Server' }))
    const dialog = await screen.findByRole('dialog', { name: 'Create Server' })
    expect(within(dialog).getByTestId('web-server-create-form-grid')).toBeInTheDocument()
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Server Name' }), { target: { value: 'RapidPro' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Code' }), { target: { value: 'rapidpro' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Base URL' }), { target: { value: 'https://rapidpro.example.com' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Create' }))

    await waitFor(() => expect(createPayload).not.toBeNull())
    expect(createPayload).toMatchObject({
      name: 'RapidPro',
      code: 'rapidpro',
      baseUrl: 'https://rapidpro.example.com',
      systemType: 'dhis2',
      endpointType: 'http',
      httpMethod: 'POST',
    })
  })

  it('edit server loads detail and submits updated payload', async () => {
    authenticate(['servers.read', 'servers.write'])
    let updatePayload: Record<string, unknown> | null = null
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/servers?')) {
        return {
          items: [
            {
              id: 5,
              uid: 'uid-5',
              name: 'OpenHIM',
              code: 'openhim',
              systemType: 'api',
              baseUrl: 'https://openhim.example.com',
              endpointType: 'http',
              httpMethod: 'POST',
              useAsync: false,
              parseResponses: true,
              headers: {},
              urlParams: {},
              suspended: false,
              createdAt: '2026-03-10T09:00:00Z',
              updatedAt: '2026-03-10T10:00:00Z',
            },
          ],
          totalCount: 1,
          page: 1,
          pageSize: 25,
        }
      }
      if (path === '/servers/5' && (!init?.method || init.method === 'GET')) {
        return {
          id: 5,
          uid: 'uid-5',
          name: 'OpenHIM',
          code: 'openhim',
          systemType: 'api',
          baseUrl: 'https://openhim.example.com',
          endpointType: 'http',
          httpMethod: 'POST',
          useAsync: false,
          parseResponses: true,
          headers: {},
          urlParams: {},
          suspended: false,
          createdAt: '2026-03-10T09:00:00Z',
          updatedAt: '2026-03-10T10:00:00Z',
        }
      }
      if (path === '/servers/5' && init?.method === 'PUT') {
        updatePayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { id: 5 }
      }
      return {}
    })

    renderRoute('/servers')

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for openhim' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Edit' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit Server' })
    expect(within(dialog).getByTestId('web-server-edit-form-grid')).toBeInTheDocument()
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Base URL' }), { target: { value: 'https://openhim.example.com/v2' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(updatePayload).not.toBeNull())
    expect(updatePayload).toMatchObject({
      code: 'openhim',
      baseUrl: 'https://openhim.example.com/v2',
    })
  })
})
