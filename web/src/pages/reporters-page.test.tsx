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

function buildReporter() {
  return {
    id: 11,
    uid: 'rep-11',
    name: 'Alice Reporter',
    telephone: '+256700000001',
    whatsapp: '+256700000001',
    telegram: '@alice',
    orgUnitId: 2,
    reportingLocation: 'Kampala Health Centre',
    districtId: 9,
    totalReports: 12,
    lastReportingDate: '2026-04-10T09:00:00Z',
    smsCode: '1234',
    smsCodeExpiresAt: '2026-04-11T09:00:00Z',
    mtuuid: 'mt-uuid-11',
    synced: true,
    rapidProUuid: 'rapidpro-11',
    isActive: true,
    createdAt: '2026-04-01T08:00:00Z',
    updatedAt: '2026-04-12T08:00:00Z',
    lastLoginAt: '2026-04-13T08:00:00Z',
    groups: ['Lead'],
  }
}

describe('reporters page', () => {
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

  it('renders reporters grid rows from mocked API', async () => {
    authenticate(['reporters.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path.includes('/reporters?')) {
        return { items: [buildReporter()], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path.includes('/orgunits?')) {
        return { items: [{ id: 2, name: 'Kampala Health Centre' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      return {}
    })

    renderRoute('/reporters')

    expect(await screen.findByRole('heading', { name: 'Reporters', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('Alice Reporter')).toBeInTheDocument()
    expect(screen.getByText('Kampala Health Centre')).toBeInTheDocument()
  })

  it('create reporter omits backend-managed payload fields and hides backend-managed inputs', async () => {
    authenticate(['reporters.read', 'reporters.write'])
    let createPayload: Record<string, unknown> | null = null
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/reporters?')) {
        return { items: [], totalCount: 0, page: 1, pageSize: 25 }
      }
      if (path.includes('/orgunits?')) {
        return { items: [{ id: 2, name: 'Kampala Health Centre' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/reporters' && init?.method === 'POST') {
        createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { id: 22 }
      }
      return {}
    })

    renderRoute('/reporters')

    fireEvent.click(await screen.findByRole('button', { name: 'New Reporter' }))
    const dialog = await screen.findByRole('dialog', { name: 'New Reporter' })
    expect(within(dialog).queryByLabelText('SMS Code')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('MT UUID')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Last Reporting Date')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Created At')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Updated At')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Last Login At')).not.toBeInTheDocument()

    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Name' }), { target: { value: 'Alice Reporter' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Telephone' }), { target: { value: '+256700000001' } })
    fireEvent.mouseDown(within(dialog).getByRole('combobox', { name: 'Facility' }))
    fireEvent.click(await screen.findByRole('option', { name: 'Kampala Health Centre' }))
    fireEvent.click(within(dialog).getByRole('button', { name: 'Create' }))

    await waitFor(() => expect(createPayload).not.toBeNull())
    expect(createPayload).toMatchObject({
      name: 'Alice Reporter',
      telephone: '+256700000001',
      orgUnitId: 2,
    })
    expect(createPayload).not.toHaveProperty('smsCode')
    expect(createPayload).not.toHaveProperty('mtuuid')
    expect(createPayload).not.toHaveProperty('rapidProUuid')
  })

  it('edit reporter keeps backend-managed RapidPro fields read-only', async () => {
    authenticate(['reporters.read', 'reporters.write'])
    let updatePayload: Record<string, unknown> | null = null
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/reporters?')) {
        return { items: [buildReporter()], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path.includes('/orgunits?')) {
        return { items: [{ id: 2, name: 'Kampala Health Centre' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/reporters/11' && init?.method === 'PUT') {
        updatePayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { id: 11 }
      }
      return {}
    })

    renderRoute('/reporters')

    fireEvent.click(await screen.findByLabelText('Actions for Alice Reporter'))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Edit' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit Reporter' })
    expect(within(dialog).queryByLabelText('SMS Code')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('MT UUID')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Last Reporting Date')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Created At')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Updated At')).not.toBeInTheDocument()
    expect(within(dialog).queryByLabelText('Last Login At')).not.toBeInTheDocument()

    expect(within(dialog).getByDisplayValue('rapidpro-11')).toBeInTheDocument()
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Name' }), { target: { value: 'Alice Reporter Updated' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(updatePayload).not.toBeNull())
    expect(updatePayload).toMatchObject({
      uid: 'rep-11',
      name: 'Alice Reporter Updated',
      totalReports: 12,
      lastReportingDate: '2026-04-10T09:00:00Z',
      lastLoginAt: '2026-04-13T08:00:00Z',
    })
    expect(updatePayload).not.toHaveProperty('smsCode')
    expect(updatePayload).not.toHaveProperty('mtuuid')
    expect(updatePayload).not.toHaveProperty('rapidProUuid')
  })

  it('supports row sync and bulk broadcast actions', async () => {
    authenticate(['reporters.read', 'reporters.write'])
    let syncPayload: Record<string, unknown> | null = null
    let broadcastPayload: Record<string, unknown> | null = null
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/reporters?')) {
        return { items: [buildReporter()], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path.includes('/orgunits?')) {
        return { items: [{ id: 2, name: 'Kampala Health Centre' }], totalCount: 1, page: 1, pageSize: 25 }
      }
      if (path === '/reporters/11/sync' && init?.method === 'POST') {
        syncPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { reporter: buildReporter(), operation: 'updated' }
      }
      if (path === '/reporters/bulk/broadcast' && init?.method === 'POST') {
        broadcastPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { reporterIds: [11], message: 'Test broadcast' }
      }
      return {}
    })

    renderRoute('/reporters')

    fireEvent.click(await screen.findByLabelText('Actions for Alice Reporter'))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Sync to RapidPro' }))
    await waitFor(() => expect(syncPayload).toEqual({}))

    fireEvent.click(screen.getByRole('checkbox', { name: 'Select reporter Alice Reporter' }))
    fireEvent.click(screen.getByRole('button', { name: 'Broadcast to Selected' }))
    const dialog = await screen.findByRole('dialog', { name: 'Broadcast to Selected Reporters' })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Message' }), { target: { value: 'Test broadcast' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Send Broadcast' }))

    await waitFor(() => expect(broadcastPayload).not.toBeNull())
    expect(broadcastPayload).toMatchObject({
      reporterIds: [11],
      text: 'Test broadcast',
    })
  })
})
