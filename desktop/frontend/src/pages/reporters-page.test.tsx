import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../api/client'
import { clearSession, configureSessionStorage, setSession } from '../auth/session'
import { createAppRouter } from '../routes'
import { AppThemeProvider } from '../ui/theme'
import { defaultSettings, type AppSettings, type SaveSettingsPatch, type SettingsStore } from '../settings/types'

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

function createMockSettingsStore(seed: AppSettings): SettingsStore {
  let state = {
    ...seed,
    uiPrefs: {
      ...seed.uiPrefs,
    },
  }

  return {
    loadSettings: async () => state,
    saveSettings: async (patch: SaveSettingsPatch) => {
      state = {
        ...state,
        ...patch,
        uiPrefs: {
          ...state.uiPrefs,
          ...(patch.uiPrefs ?? {}),
        },
        tablePrefs: patch.tablePrefs ?? state.tablePrefs,
      }
      return state
    },
    resetSettings: async () => state,
  }
}

function renderRoute(initialPath: string, store: SettingsStore) {
  const router = createAppRouter([initialPath], store)
  const queryClient = new QueryClient()
  return render(
    <QueryClientProvider client={queryClient}>
      <AppThemeProvider store={store}>
        <RouterProvider router={router} />
      </AppThemeProvider>
    </QueryClientProvider>,
  )
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

describe('desktop reporters page', () => {
  beforeEach(async () => {
    await clearSession()
  })

  afterEach(async () => {
    cleanup()
    await clearSession()
  })

  it('renders reporters grid rows from mocked API', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
      refreshToken: 'refresh-token',
    })
    configureSessionStorage(store)
    await setSession({
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      expiresAt: Date.now() + 60_000,
    })

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/api/v1/auth/me')) {
          return new Response(
            JSON.stringify({
              id: 5,
              username: 'alice',
              roles: ['Staff'],
              permissions: ['reporters.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/reporters?')) {
          return new Response(JSON.stringify({ items: [buildReporter()], totalCount: 1, page: 1, pageSize: 25 }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        if (url.endsWith('/api/v1/reporter-groups/options')) {
          return new Response(JSON.stringify({ items: [{ id: 1, name: 'Lead' }] }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        if (url.includes('/api/v1/orgunits?')) {
          return new Response(JSON.stringify({ items: [{ id: 2, name: 'Kampala Health Centre' }], totalCount: 1, page: 1, pageSize: 25 }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        if (url.includes('/api/v1/bootstrap')) {
          return new Response(JSON.stringify({}), { status: 200, headers: { 'Content-Type': 'application/json' } })
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderRoute('/reporters', store)

    expect(await screen.findByRole('heading', { name: 'Reporters', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('Alice Reporter')).toBeInTheDocument()
    expect(screen.getByText('Kampala Health Centre')).toBeInTheDocument()
  })

  it('create, edit, and messaging flows use the RapidPro actions workflow', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
      refreshToken: 'refresh-token',
    })
    configureSessionStorage(store)
    await setSession({
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      expiresAt: Date.now() + 60_000,
    })

    let createPayload: Record<string, unknown> | null = null
    let updatePayload: Record<string, unknown> | null = null
    let syncPayload: Record<string, unknown> | null = null
    let broadcastPayload: Record<string, unknown> | null = null
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('/api/v1/auth/me')) {
        return new Response(
          JSON.stringify({
            id: 5,
            username: 'alice',
            roles: ['Staff'],
            permissions: ['reporters.read', 'reporters.write'],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.includes('/api/v1/reporters?')) {
        return new Response(JSON.stringify({ items: [buildReporter()], totalCount: 1, page: 1, pageSize: 25 }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      if (url.endsWith('/api/v1/reporter-groups/options')) {
        return new Response(JSON.stringify({ items: [{ id: 1, name: 'Lead' }] }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      if (url.includes('/api/v1/orgunits?')) {
        return new Response(JSON.stringify({ items: [{ id: 2, name: 'Kampala Health Centre' }], totalCount: 1, page: 1, pageSize: 25 }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      if (url.endsWith('/api/v1/reporters') && init?.method === 'POST') {
        createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return new Response(JSON.stringify({ id: 22 }), { status: 201, headers: { 'Content-Type': 'application/json' } })
      }
      if (url.endsWith('/api/v1/reporters/11') && init?.method === 'PUT') {
        updatePayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return new Response(JSON.stringify({ id: 11 }), { status: 200, headers: { 'Content-Type': 'application/json' } })
      }
      if (url.endsWith('/api/v1/reporters/11/sync') && init?.method === 'POST') {
        syncPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return new Response(JSON.stringify({ reporter: buildReporter(), operation: 'updated' }), { status: 200, headers: { 'Content-Type': 'application/json' } })
      }
      if (url.endsWith('/api/v1/reporters/bulk/broadcast') && init?.method === 'POST') {
        broadcastPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return new Response(JSON.stringify({ reporterIds: [11], message: 'Test broadcast' }), { status: 200, headers: { 'Content-Type': 'application/json' } })
      }
      if (url.includes('/api/v1/bootstrap')) {
        return new Response(JSON.stringify({}), { status: 200, headers: { 'Content-Type': 'application/json' } })
      }
      return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderRoute('/reporters', store)

    fireEvent.click(await screen.findByRole('button', { name: 'New Reporter' }))
    const createDialog = await screen.findByRole('dialog', { name: 'New Reporter' })
    expect(within(createDialog).queryByLabelText('SMS Code')).not.toBeInTheDocument()
    expect(within(createDialog).queryByLabelText('MT UUID')).not.toBeInTheDocument()
    expect(within(createDialog).queryByLabelText('Last Reporting Date')).not.toBeInTheDocument()
    expect(within(createDialog).queryByLabelText('Created At')).not.toBeInTheDocument()
    expect(within(createDialog).queryByLabelText('Updated At')).not.toBeInTheDocument()
    expect(within(createDialog).queryByLabelText('Last Login At')).not.toBeInTheDocument()
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Name' }), { target: { value: 'Alice Reporter' } })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Telephone' }), { target: { value: '+256700000001' } })
    fireEvent.mouseDown(within(createDialog).getByRole('combobox', { name: 'Facility' }))
    fireEvent.click(await screen.findByRole('option', { name: 'Kampala Health Centre' }))
    fireEvent.click(within(createDialog).getByRole('button', { name: 'Create' }))

    await waitFor(() => expect(createPayload).not.toBeNull())
    expect(createPayload).toMatchObject({
      name: 'Alice Reporter',
      telephone: '+256700000001',
      orgUnitId: 2,
    })
    expect(createPayload).not.toHaveProperty('smsCode')
    expect(createPayload).not.toHaveProperty('mtuuid')
    expect(createPayload).not.toHaveProperty('rapidProUuid')

    fireEvent.click(await screen.findByLabelText('Actions for Alice Reporter'))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Edit' }))
    const editDialog = await screen.findByRole('dialog', { name: 'Edit Reporter' })
    expect(within(editDialog).queryByLabelText('SMS Code')).not.toBeInTheDocument()
    expect(within(editDialog).queryByLabelText('MT UUID')).not.toBeInTheDocument()
    expect(within(editDialog).queryByLabelText('Last Reporting Date')).not.toBeInTheDocument()
    expect(within(editDialog).queryByLabelText('Created At')).not.toBeInTheDocument()
    expect(within(editDialog).queryByLabelText('Updated At')).not.toBeInTheDocument()
    expect(within(editDialog).queryByLabelText('Last Login At')).not.toBeInTheDocument()
    expect(within(editDialog).getByDisplayValue('rapidpro-11')).toBeInTheDocument()
    fireEvent.change(within(editDialog).getByRole('textbox', { name: 'Name' }), { target: { value: 'Alice Reporter Updated' } })
    fireEvent.click(within(editDialog).getByRole('button', { name: 'Save' }))

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

    fireEvent.click(await screen.findByLabelText('Actions for Alice Reporter'))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Sync to RapidPro' }))
    await waitFor(() => expect(syncPayload).toEqual({}))

    fireEvent.click(await screen.findByRole('checkbox', { name: 'Select reporter Alice Reporter' }))
    fireEvent.click(screen.getByRole('button', { name: 'Broadcast to Selected' }))
    const messageDialog = await screen.findByRole('dialog', { name: 'Broadcast to Selected Reporters' })
    fireEvent.change(within(messageDialog).getByRole('textbox', { name: 'Message' }), { target: { value: 'Test broadcast' } })
    fireEvent.click(within(messageDialog).getByRole('button', { name: 'Send Broadcast' }))

    await waitFor(() => expect(broadcastPayload).not.toBeNull())
    expect(broadcastPayload).toMatchObject({
      reporterIds: [11],
      text: 'Test broadcast',
    })
  }, 20000)

  it('shows actionable sync validation detail in the error banner', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
      refreshToken: 'refresh-token',
    })
    configureSessionStorage(store)
    await setSession({
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      expiresAt: Date.now() + 60_000,
    })

    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('/api/v1/auth/me')) {
        return new Response(
          JSON.stringify({
            id: 5,
            username: 'alice',
            roles: ['Staff'],
            permissions: ['reporters.read', 'reporters.write'],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.includes('/api/v1/reporters?')) {
        return new Response(JSON.stringify({ items: [buildReporter()], totalCount: 1, page: 1, pageSize: 25 }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      if (url.endsWith('/api/v1/reporter-groups/options')) {
        return new Response(JSON.stringify({ items: [{ id: 1, name: 'Lead' }] }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      if (url.includes('/api/v1/orgunits?')) {
        return new Response(JSON.stringify({ items: [{ id: 2, name: 'Kampala Health Centre' }], totalCount: 1, page: 1, pageSize: 25 }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      if (url.endsWith('/api/v1/reporters/11/sync') && init?.method === 'POST') {
        throw new ApiError(
          400,
          'validation failed',
          'VALIDATION_ERROR',
          { telephone: ['must resolve to a RapidPro tel: URN'] },
          'req-desktop-reporters-422',
        )
      }
      if (url.includes('/api/v1/bootstrap')) {
        return new Response(JSON.stringify({}), { status: 200, headers: { 'Content-Type': 'application/json' } })
      }
      return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderRoute('/reporters', store)

    fireEvent.click(await screen.findByLabelText('Actions for Alice Reporter'))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Sync to RapidPro' }))

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Unable to sync reporter.')
    expect(alert).toHaveTextContent('must resolve to a RapidPro tel: URN')
    expect(alert).toHaveTextContent('req-desktop-reporters-422')
  })
})
