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
    window.localStorage.clear()
    clearAuthSnapshot()
    vi.unstubAllEnvs()
    apiRequestSpy.mockRestore()
  })

  it('/users renders metadata columns and assigned roles from mocked API', async () => {
    authenticate(['users.read', 'audit.read', 'settings.read', 'users.write'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path.includes('/admin/roles?')) {
        return {
          items: [
            { id: 1, name: 'Admin' },
            { id: 2, name: 'Viewer' },
          ],
          totalCount: 2,
          page: 1,
          pageSize: 200,
        }
      }
      if (path.includes('/users?')) {
        return {
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
              roles: ['Admin', 'Viewer'],
              updatedAt: '2026-03-01T12:00:00Z',
              createdAt: '2026-03-01T10:00:00Z',
            },
          ],
          totalCount: 1,
          page: 1,
          pageSize: 25,
        }
      }
      return {}
    })

    renderRoute('/users')

    expect(await screen.findByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument()
    expect(screen.getAllByText('Roles').length).toBeGreaterThan(0)
    expect(await screen.findByText('Admin')).toBeInTheDocument()
    expect(await screen.findByText('Viewer')).toBeInTheDocument()
    expect(await screen.findByText('Alice Johnson')).toBeInTheDocument()
    expect(screen.getByText('alice@example.com')).toBeInTheDocument()
    await waitFor(() => expect(apiRequestSpy).toHaveBeenCalledWith(expect.stringContaining('/users?')))
  })

  it('create user supports role and org-unit multi-select and submits selected assignments', async () => {
    authenticate(['users.read', 'users.write', 'audit.read'])
    let createPayload: Record<string, unknown> | null = null
    const assignmentPayloads: Array<Record<string, unknown>> = []
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/admin/roles?')) {
        return {
          items: [
            { id: 1, name: 'Admin' },
            { id: 2, name: 'Viewer' },
          ],
          totalCount: 2,
          page: 1,
          pageSize: 200,
        }
      }
      if (path.includes('/orgunits?')) {
        return {
          items: [
            { id: 11, name: 'Kampala', displayPath: 'Uganda / Kampala' },
            { id: 12, name: 'Wakiso', displayPath: 'Uganda / Wakiso' },
          ],
          totalCount: 2,
          page: 0,
          pageSize: 50,
        }
      }
      if (path.includes('/users?') && (!init?.method || init.method === 'GET')) {
        return {
          items: [],
          totalCount: 0,
          page: 1,
          pageSize: 25,
        }
      }
      if (path.includes('/users') && init?.method === 'POST') {
        createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { id: 99, username: 'new-user' }
      }
      if (path.endsWith('/user-org-units') && init?.method === 'POST') {
        assignmentPayloads.push(JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>)
        return {}
      }
      return {}
    })

    renderRoute('/users')
    expect(await screen.findByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Create User' }))

    const dialog = await screen.findByRole('dialog', { name: 'Create User' })
    expect(within(dialog).getByTestId('web-user-create-form-grid')).toBeInTheDocument()
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Username' }), { target: { value: 'new-user' } })
    const passwordInput = dialog.querySelector('input[type="password"]')
    expect(passwordInput).not.toBeNull()
    fireEvent.change(passwordInput as Element, { target: { value: 'TempPass123!' } })

    const rolesInput = within(dialog).getByRole('combobox', { name: 'Roles' })
    fireEvent.mouseDown(rolesInput)
    fireEvent.change(rolesInput, { target: { value: 'Admin' } })
    fireEvent.click(await screen.findByRole('option', { name: 'Admin' }))

    const orgUnitsInput = within(dialog).getByRole('combobox', { name: 'Assigned Org Units' })
    fireEvent.mouseDown(orgUnitsInput)
    fireEvent.change(orgUnitsInput, { target: { value: 'Kam' } })
    fireEvent.click(await screen.findByRole('option', { name: /Kampala\s+Uganda \/ Kampala/ }))
    fireEvent.mouseDown(orgUnitsInput)
    fireEvent.change(orgUnitsInput, { target: { value: 'Wak' } })
    fireEvent.click(await screen.findByRole('option', { name: /Wakiso\s+Uganda \/ Wakiso/ }))

    fireEvent.click(within(dialog).getByRole('button', { name: 'Create' }))

    await waitFor(() => expect(createPayload).not.toBeNull())
    await waitFor(() => expect(assignmentPayloads).toHaveLength(2))
    expect(await screen.findByText('User created.')).toBeInTheDocument()
    expect(createPayload).toMatchObject({
      username: 'new-user',
      password: 'TempPass123!',
      roles: ['Admin'],
      isActive: true,
    })
    expect(assignmentPayloads).toEqual([
      { userId: 99, orgUnitId: 11 },
      { userId: 99, orgUnitId: 12 },
    ])
  })

  it('edit user supports role and org-unit multi-select and updates selected assignments', async () => {
    authenticate(['users.read', 'users.write', 'audit.read'])
    let patchPayload: Record<string, unknown> | null = null
    const assignmentPayloads: Array<Record<string, unknown>> = []
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/admin/roles?')) {
        return {
          items: [
            { id: 1, name: 'Admin' },
            { id: 2, name: 'Viewer' },
            { id: 3, name: 'Manager' },
          ],
          totalCount: 3,
          page: 1,
          pageSize: 200,
        }
      }
      if (path.includes('/orgunits?')) {
        return {
          items: [
            { id: 11, name: 'Kampala', displayPath: 'Uganda / Kampala' },
            { id: 12, name: 'Wakiso', displayPath: 'Uganda / Wakiso' },
          ],
          totalCount: 2,
          page: 0,
          pageSize: 50,
        }
      }
      if (path.includes('/users?') && (!init?.method || init.method === 'GET')) {
        return {
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
              roles: ['Viewer'],
              updatedAt: '2026-03-01T12:00:00Z',
              createdAt: '2026-03-01T10:00:00Z',
            },
          ],
          totalCount: 1,
          page: 1,
          pageSize: 25,
        }
      }
      if (path.endsWith('/user-org-units/7') && (!init?.method || init.method === 'GET')) {
        return {
          orgUnitIds: [11],
          items: [{ orgUnitId: 11, orgUnitName: 'Kampala', displayPath: 'Uganda / Kampala' }],
        }
      }
      if (path.endsWith('/users/7') && init?.method === 'PATCH') {
        patchPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { id: 7, username: 'jane' }
      }
      if (path.endsWith('/user-org-units') && init?.method === 'POST') {
        assignmentPayloads.push(JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>)
        return {}
      }
      return {}
    })

    renderRoute('/users')
    expect(await screen.findByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument()
    fireEvent.click(await screen.findByRole('button', { name: 'Actions for jane' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Edit' }))

    const dialog = await screen.findByRole('dialog', { name: 'Edit User' })
    expect(within(dialog).getByTestId('web-user-edit-form-grid')).toBeInTheDocument()
    const rolesInput = within(dialog).getByRole('combobox', { name: 'Roles' })
    fireEvent.mouseDown(rolesInput)
    fireEvent.change(rolesInput, { target: { value: 'Admin' } })
    fireEvent.click(await screen.findByRole('option', { name: 'Admin' }))

    const orgUnitsInput = within(dialog).getByRole('combobox', { name: 'Assigned Org Units' })
    fireEvent.mouseDown(orgUnitsInput)
    fireEvent.change(orgUnitsInput, { target: { value: 'Wak' } })
    fireEvent.click(await screen.findByRole('option', { name: /Wakiso\s+Uganda \/ Wakiso/ }))
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(patchPayload).not.toBeNull())
    await waitFor(() => expect(assignmentPayloads).toHaveLength(1))
    expect(patchPayload).toMatchObject({
      username: 'jane',
      roles: ['Viewer', 'Admin'],
    })
    expect(assignmentPayloads).toEqual([{ userId: 7, orgUnitId: 12 }])
  })

  it('users action menu supports details view and deactivate confirmation flow', async () => {
    authenticate(['users.read', 'users.write'])
    let deactivatePayload: Record<string, unknown> | null = null
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/admin/roles?')) {
        return { items: [], totalCount: 0, page: 1, pageSize: 200 }
      }
      if (path.includes('/users?') && (!init?.method || init.method === 'GET')) {
        return {
          items: [
            {
              id: 11,
              username: 'mark',
              language: 'English',
              email: 'mark@example.com',
              isActive: true,
              roles: ['Admin'],
              updatedAt: '2026-03-01T12:00:00Z',
              createdAt: '2026-03-01T10:00:00Z',
            },
          ],
          totalCount: 1,
          page: 1,
          pageSize: 25,
        }
      }
      if (path.endsWith('/users/11') && init?.method === 'PATCH') {
        deactivatePayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { id: 11, username: 'mark' }
      }
      return {}
    })

    renderRoute('/users')
    expect(await screen.findByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument()

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for mark' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View Details' }))
    const detailsDialog = await screen.findByRole('dialog', { name: 'User Details' })
    expect(within(detailsDialog).getAllByText('mark').length).toBeGreaterThan(0)
    expect(within(detailsDialog).getByText('Admin')).toBeInTheDocument()
    fireEvent.click(within(detailsDialog).getByRole('button', { name: 'Close' }))

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for mark' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Deactivate' }))
    expect(await screen.findByRole('dialog', { name: 'Deactivate user' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(deactivatePayload).not.toBeNull())
    expect(deactivatePayload).toEqual({ isActive: false })
  })

  it('validation error displays field messages and request ID', async () => {
    authenticate(['users.read', 'users.write'])
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/admin/roles?')) {
        return {
          items: [{ id: 1, name: 'Admin' }],
          totalCount: 1,
          page: 1,
          pageSize: 200,
        }
      }
      if (path.includes('/users?') && (!init?.method || init.method === 'GET')) {
        return {
          items: [],
          totalCount: 0,
          page: 1,
          pageSize: 25,
        }
      }
      if (path.includes('/users') && init?.method === 'POST') {
        const error = new Error('validation failed') as Error & {
          code: string
          message: string
          details: Record<string, string[]>
          requestId: string
        }
        error.code = 'VALIDATION_ERROR'
        error.message = 'validation failed'
        error.details = {
          email: ['must be a valid email address'],
          phoneNumber: ['must be E.164 format, e.g. +15551234567'],
          roles: ['contains unknown role identifier'],
        }
        error.requestId = 'req-users-422'
        throw error
      }
      return {}
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
    expect(within(dialog).getByText('contains unknown role identifier')).toBeInTheDocument()
    expect(within(dialog).getByText('validation failed Request ID: req-users-422')).toBeInTheDocument()
  })

  it('/roles supports list/create/edit/detail flows', async () => {
    authenticate(['users.read', 'users.write'])
    let createPayload: Record<string, unknown> | null = null
    let updatePayload: Record<string, unknown> | null = null

    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/admin/permissions?')) {
        return {
          items: [
            { id: 1, name: 'users.read', moduleScope: 'admin', createdAt: '2026-03-01T00:00:00Z' },
            { id: 2, name: 'users.write', moduleScope: 'admin', createdAt: '2026-03-01T00:00:00Z' },
          ],
          totalCount: 2,
          page: 1,
          pageSize: 200,
        }
      }
      if (path.includes('/admin/roles/2?includeUsers=false')) {
        return {
          id: 2,
          name: 'Manager',
          permissions: [{ id: 1, name: 'users.read', moduleScope: 'admin', createdAt: '2026-03-01T00:00:00Z' }],
          createdAt: '2026-03-01T00:00:00Z',
          updatedAt: '2026-03-01T00:00:00Z',
        }
      }
      if (path.includes('/admin/roles/2?includeUsers=true')) {
        return {
          id: 2,
          name: 'Manager',
          permissions: [{ id: 1, name: 'users.read', moduleScope: 'admin', createdAt: '2026-03-01T00:00:00Z' }],
          users: [{ id: 42, username: 'alice', isActive: true }],
          createdAt: '2026-03-01T00:00:00Z',
          updatedAt: '2026-03-01T00:00:00Z',
        }
      }
      if (path.includes('/admin/roles?') && (!init?.method || init.method === 'GET')) {
        return {
          items: [
            {
              id: 2,
              name: 'Manager',
              permissionCount: 1,
              userCount: 1,
              createdAt: '2026-03-01T00:00:00Z',
              updatedAt: '2026-03-02T00:00:00Z',
            },
          ],
          totalCount: 1,
          page: 1,
          pageSize: 25,
        }
      }
      if (path.endsWith('/admin/roles') && init?.method === 'POST') {
        createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { id: 3, name: 'Support' }
      }
      if (path.endsWith('/admin/roles/2') && init?.method === 'PATCH') {
        updatePayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { id: 2, name: 'Manager Updated' }
      }
      return {}
    })

    renderRoute('/roles')

    expect(await screen.findByRole('heading', { name: 'Roles', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('Manager')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Create Role' }))
    const createDialog = await screen.findByRole('dialog', { name: 'Create Role' })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Role Name' }), { target: { value: 'Support' } })
    fireEvent.click(within(createDialog).getByRole('button', { name: 'Create' }))

    await waitFor(() => expect(createPayload).not.toBeNull())
    expect(createPayload).toEqual({ name: 'Support', permissions: [] })

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for Manager' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Edit Role' }))
    const editDialog = await screen.findByRole('dialog', { name: 'Edit Role' })
    fireEvent.change(within(editDialog).getByRole('textbox', { name: 'Role Name' }), { target: { value: 'Manager Updated' } })
    fireEvent.click(within(editDialog).getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(updatePayload).not.toBeNull())
    expect(updatePayload).toMatchObject({ name: 'Manager Updated' })

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for Manager' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View Details' }))
    expect(await screen.findByRole('dialog', { name: 'Role Details' })).toBeInTheDocument()
    expect(screen.getByText('alice')).toBeInTheDocument()
    await waitFor(() => {
      expect(apiRequestSpy).toHaveBeenCalledWith(expect.stringContaining('/admin/roles/2?includeUsers=true'))
    })
  })

  it('/permissions supports list/filter/details flows', async () => {
    authenticate(['users.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path.includes('/admin/permissions?')) {
        return {
          items: [
            {
              id: 4,
              name: 'users.read',
              moduleScope: 'admin',
              createdAt: '2026-03-01T00:00:00Z',
            },
          ],
          totalCount: 1,
          page: 1,
          pageSize: 25,
        }
      }
      return {}
    })

    renderRoute('/permissions')

    expect(await screen.findByRole('heading', { name: 'Permissions', level: 1 })).toBeInTheDocument()
    fireEvent.change(screen.getByPlaceholderText('e.g. users.read'), { target: { value: 'users' } })
    fireEvent.change(screen.getByPlaceholderText('e.g. admin'), { target: { value: 'admin' } })

    await waitFor(() => {
      expect(apiRequestSpy).toHaveBeenCalledWith(expect.stringContaining('/admin/permissions?'))
      const calls = apiRequestSpy.mock.calls.map((call) => String(call[0]))
      expect(calls.some((call) => call.includes('q=users'))).toBe(true)
      expect(calls.some((call) => call.includes('moduleScope=admin'))).toBe(true)
    })

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for users.read' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View Details' }))
    expect(await screen.findByRole('dialog', { name: 'Permission Details' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for users.read' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View Metadata' }))
    expect(await screen.findByRole('dialog', { name: 'Audit Metadata' })).toBeInTheDocument()
  })

  it('/audit renders mocked API rows', async () => {
    authenticate()
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path.includes('/audit?')) {
        return {
          items: [{ id: 20, timestamp: new Date().toISOString(), action: 'auth.login.success', metadata: { ip: '127.0.0.1' } }],
          totalCount: 1,
          page: 1,
          pageSize: 25,
        }
      }
      return {}
    })

    renderRoute('/audit')

    expect(await screen.findByRole('heading', { name: 'Audit Log', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('auth.login.success')).toBeInTheDocument()
    fireEvent.click(await screen.findByRole('button', { name: 'Actions for auth.login.success' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View Metadata' }))
    expect(await screen.findByRole('dialog', { name: 'Audit Metadata' })).toBeInTheDocument()
    expect(screen.getByText(/"ip": "127.0.0.1"/)).toBeInTheDocument()
    await waitFor(() => expect(apiRequestSpy).toHaveBeenCalledWith(expect.stringContaining('/audit?')))
  })
})
