import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
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

describe('desktop requests page', () => {
  beforeEach(async () => {
    await clearSession()
  })

  afterEach(async () => {
    cleanup()
    await clearSession()
  })

  it('renders requests route and rows from backend API', async () => {
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
              permissions: ['requests.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/requests?')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 1,
                  uid: 'req-1',
                  sourceSystem: 'emr',
                  destinationServerId: 3,
                  destinationServerName: 'DHIS2 Uganda',
                  batchId: 'batch-1',
                  correlationId: 'corr-1',
                  idempotencyKey: 'idem-1',
                  payloadBody: '{"trackedEntity":"123"}',
                  payloadFormat: 'json',
                  submissionBinding: 'body',
                  payload: { trackedEntity: '123' },
                  urlSuffix: '/api/data',
                  status: 'pending',
                  statusReason: 'window_closed',
                  deferredUntil: '2026-03-10T12:00:00Z',
                  extras: {},
                  createdAt: '2026-03-10T09:00:00Z',
                  updatedAt: '2026-03-10T10:00:00Z',
                  latestDeliveryUid: 'del-1',
                  latestDeliveryStatus: 'pending',
                  latestAsyncTaskUid: '',
                  latestAsyncState: '',
                  latestAsyncRemoteJobId: '',
                  latestAsyncPollUrl: '',
                  awaitingAsync: false,
                  targets: [{ id: 1, uid: 'target-1', serverId: 3, serverName: 'DHIS2 Uganda', serverCode: 'dhis2-ug', targetKind: 'primary', status: 'blocked', blockedReason: 'window_closed', deferredUntil: '2026-03-10T12:00:00Z', latestDeliveryUid: 'del-1', latestDeliveryStatus: 'pending', latestAsyncTaskUid: '', latestAsyncState: '', awaitingAsync: false }],
                  dependencies: [],
                },
              ],
              totalCount: 1,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/bootstrap')) {
          return new Response(JSON.stringify({}), { status: 200, headers: { 'Content-Type': 'application/json' } })
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderRoute('/requests', store)

    expect(await screen.findByRole('heading', { name: 'Requests', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('req-1')).toBeInTheDocument()
    expect(screen.getByText('DHIS2 Uganda')).toBeInTheDocument()
    expect(screen.getByText(/window_closed/)).toBeInTheDocument()
  })

  it('create and detail request flows call backend APIs', async () => {
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
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('/api/v1/auth/me')) {
        return new Response(
          JSON.stringify({
            id: 5,
            username: 'alice',
            roles: ['Staff'],
            permissions: ['requests.read', 'requests.write'],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.includes('/api/v1/requests?')) {
        return new Response(
          JSON.stringify({
            items: [
              {
                id: 4,
                uid: 'req-4',
                sourceSystem: 'emr',
                destinationServerId: 3,
                destinationServerName: 'DHIS2 Uganda',
                batchId: 'batch-4',
                correlationId: 'corr-4',
                idempotencyKey: 'idem-4',
                payloadBody: '{"trackedEntity":"123"}',
                payloadFormat: 'json',
                submissionBinding: 'body',
                payload: { trackedEntity: '123' },
                urlSuffix: '/api/data',
                status: 'pending',
                statusReason: 'dependency_blocked',
                deferredUntil: null,
                extras: {},
                createdAt: '2026-03-10T09:00:00Z',
                updatedAt: '2026-03-10T10:00:00Z',
                latestDeliveryUid: 'del-4',
                latestDeliveryStatus: 'pending',
                latestAsyncTaskUid: '',
                latestAsyncState: '',
                latestAsyncRemoteJobId: '',
                latestAsyncPollUrl: '',
                awaitingAsync: false,
                targets: [
                  { id: 1, uid: 'target-4a', serverId: 3, serverName: 'DHIS2 Uganda', serverCode: 'dhis2-ug', targetKind: 'primary', status: 'blocked', blockedReason: 'dependency_blocked', deferredUntil: null, latestDeliveryUid: 'del-4a', latestDeliveryStatus: 'pending', latestAsyncTaskUid: '', latestAsyncState: '', awaitingAsync: false },
                  { id: 2, uid: 'target-4b', serverId: 9, serverName: 'Mirror', serverCode: 'mirror', targetKind: 'cc', status: 'succeeded', blockedReason: '', deferredUntil: null, latestDeliveryUid: 'del-4b', latestDeliveryStatus: 'succeeded', latestAsyncTaskUid: '', latestAsyncState: '', awaitingAsync: false },
                ],
                dependencies: [{ requestId: 4, dependsOnRequestId: 1, dependsOnUid: 'req-1', status: 'failed', statusReason: 'dependency_failed', deferredUntil: null, dependsOnDestinationServerName: 'DHIS2 Uganda' }],
              },
            ],
            totalCount: 1,
            page: 1,
            pageSize: 25,
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.includes('/api/v1/servers?')) {
        return new Response(
          JSON.stringify({
            items: [{ id: 3, name: 'DHIS2 Uganda', code: 'dhis2-ug' }],
            totalCount: 1,
            page: 1,
            pageSize: 200,
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.endsWith('/api/v1/requests/4') && (!init?.method || init.method === 'GET')) {
        return new Response(
          JSON.stringify({
            id: 4,
            uid: 'req-4',
            sourceSystem: 'emr',
            destinationServerId: 3,
            destinationServerName: 'DHIS2 Uganda',
            batchId: 'batch-4',
            correlationId: 'corr-4',
            idempotencyKey: 'idem-4',
            payloadBody: '{"trackedEntity":"123"}',
            payloadFormat: 'json',
            submissionBinding: 'body',
            payload: { trackedEntity: '123' },
            urlSuffix: '/api/data',
            status: 'pending',
            statusReason: 'dependency_blocked',
            deferredUntil: null,
            extras: { priority: 'high' },
            createdAt: '2026-03-10T09:00:00Z',
            updatedAt: '2026-03-10T10:00:00Z',
            latestDeliveryUid: 'del-4',
            latestDeliveryStatus: 'pending',
            latestAsyncTaskUid: '',
            latestAsyncState: '',
            latestAsyncRemoteJobId: '',
            latestAsyncPollUrl: '',
            awaitingAsync: false,
            targets: [
              { id: 1, uid: 'target-4a', serverId: 3, serverName: 'DHIS2 Uganda', serverCode: 'dhis2-ug', targetKind: 'primary', status: 'blocked', blockedReason: 'dependency_blocked', deferredUntil: null, latestDeliveryUid: 'del-4a', latestDeliveryStatus: 'pending', latestAsyncTaskUid: '', latestAsyncState: '', awaitingAsync: false },
              { id: 2, uid: 'target-4b', serverId: 9, serverName: 'Mirror', serverCode: 'mirror', targetKind: 'cc', status: 'succeeded', blockedReason: '', deferredUntil: null, latestDeliveryUid: 'del-4b', latestDeliveryStatus: 'succeeded', latestAsyncTaskUid: '', latestAsyncState: '', awaitingAsync: false },
            ],
            dependencies: [{ requestId: 4, dependsOnRequestId: 1, dependsOnUid: 'req-1', status: 'failed', statusReason: 'dependency_failed', deferredUntil: null, dependsOnDestinationServerName: 'DHIS2 Uganda' }],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.endsWith('/api/v1/requests') && init?.method === 'POST') {
        createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return new Response(JSON.stringify({ id: 9, uid: 'req-9' }), { status: 201, headers: { 'Content-Type': 'application/json' } })
      }
      if (url.includes('/api/v1/bootstrap')) {
        return new Response(JSON.stringify({}), { status: 200, headers: { 'Content-Type': 'application/json' } })
      }
      return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderRoute('/requests', store)

    fireEvent.click(await screen.findByRole('button', { name: 'Create Request' }))
    const dialog = await screen.findByRole('dialog', { name: 'Create Request' })
    expect(within(dialog).getByTestId('desktop-request-create-form-grid')).toBeInTheDocument()
    fireEvent.mouseDown(within(dialog).getByLabelText('Destination Server'))
    fireEvent.click(await screen.findByRole('option', { name: 'DHIS2 Uganda (dhis2-ug)' }))
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Additional Destination Server IDs' }), { target: { value: '9, 12' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Dependency Request IDs' }), { target: { value: '7, 8' } })
    fireEvent.mouseDown(within(dialog).getByLabelText('Payload Format'))
    fireEvent.click(await screen.findByRole('option', { name: 'Text' }))
    fireEvent.mouseDown(within(dialog).getByLabelText('Send As'))
    fireEvent.click(await screen.findByRole('option', { name: 'Query Params' }))
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Payload Text' }), {
      target: { value: 'trackedEntity=xyz&orgUnit=ou-7' },
    })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Create' }))

    await waitFor(() => expect(createPayload).not.toBeNull())
    expect(createPayload).toMatchObject({
      destinationServerId: 3,
      destinationServerIds: [9, 12],
      dependencyRequestIds: [7, 8],
      payloadFormat: 'text',
      submissionBinding: 'query',
      payload: 'trackedEntity=xyz&orgUnit=ou-7',
    })

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for req-4' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View' }))

    const detailDialog = await screen.findByRole('dialog', { name: 'Request Detail' })
    expect(within(detailDialog).getByText('req-4')).toBeInTheDocument()
    expect(within(detailDialog).getByText('2 targets')).toBeInTheDocument()
    expect(within(detailDialog).getAllByText('dependency_blocked').length).toBeGreaterThan(0)
    expect(within(detailDialog).getByText('req-1')).toBeInTheDocument()
    expect(within(detailDialog).getByText(/trackedEntity/)).toBeInTheDocument()
  }, 40000)

  it('hides create button without write permission', async () => {
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
              permissions: ['requests.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/requests?')) {
          return new Response(JSON.stringify({ items: [], totalCount: 0, page: 1, pageSize: 25 }), {
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

    renderRoute('/requests', store)

    expect(await screen.findByRole('heading', { name: 'Requests', level: 1 })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create Request' })).not.toBeInTheDocument()
  })

  it('opens request body in a formatted dialog from row actions', async () => {
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
              permissions: ['requests.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/requests?')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 10,
                  uid: 'req-10',
                  sourceSystem: 'emr',
                  destinationServerId: 3,
                  destinationServerName: 'DHIS2 Uganda',
                  batchId: 'batch-10',
                  correlationId: 'corr-10',
                  idempotencyKey: 'idem-10',
                  payloadBody: '{"trackedEntity":"body-10","program":"beta"}',
                  payloadFormat: 'json',
                  submissionBinding: 'body',
                  payload: { trackedEntity: 'body-10', program: 'beta' },
                  urlSuffix: '/api/data',
                  status: 'pending',
                  statusReason: '',
                  deferredUntil: null,
                  extras: {},
                  createdAt: '2026-03-10T09:00:00Z',
                  updatedAt: '2026-03-10T10:00:00Z',
                  latestDeliveryUid: 'del-10',
                  latestDeliveryStatus: 'pending',
                  latestAsyncTaskUid: '',
                  latestAsyncState: '',
                  latestAsyncRemoteJobId: '',
                  latestAsyncPollUrl: '',
                  awaitingAsync: false,
                  targets: [],
                  dependencies: [],
                },
              ],
              totalCount: 1,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/bootstrap')) {
          return new Response(JSON.stringify({}), { status: 200, headers: { 'Content-Type': 'application/json' } })
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderRoute('/requests', store)

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for req-10' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View Body' }))

    const dialog = await screen.findByRole('dialog', { name: 'Request Body: req-10' })
    expect(within(dialog).getByText('Formatted JSON')).toBeInTheDocument()
    expect(within(dialog).getByText(/trackedEntity/)).toBeInTheDocument()
    expect(within(dialog).getByText(/program/)).toBeInTheDocument()
  })
})
