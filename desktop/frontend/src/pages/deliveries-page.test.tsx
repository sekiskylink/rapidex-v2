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

describe('desktop deliveries page', () => {
  beforeEach(async () => {
    await clearSession()
  })

  afterEach(async () => {
    cleanup()
    await clearSession()
  })

  it('renders deliveries route and rows from backend API', async () => {
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
              permissions: ['deliveries.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/deliveries?')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 1,
                  uid: 'del-1',
                  requestId: 3,
                  requestUid: 'req-3',
                  serverId: 4,
                  serverName: 'DHIS2 Uganda',
                  attemptNumber: 1,
                  status: 'failed',
                  httpStatus: 504,
                  responseBody: '',
                  errorMessage: 'timeout',
                  startedAt: '2026-03-10T09:00:00Z',
                  finishedAt: '2026-03-10T09:05:00Z',
                  retryAt: null,
                  createdAt: '2026-03-10T09:00:00Z',
                  updatedAt: '2026-03-10T09:05:00Z',
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

    renderRoute('/deliveries', store)

    expect(await screen.findByRole('heading', { name: 'Deliveries', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('del-1')).toBeInTheDocument()
    expect(screen.getByText('req-3')).toBeInTheDocument()
  })

  it('delivery detail and retry flow call backend APIs', async () => {
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

    let retryCalls = 0
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/api/v1/auth/me')) {
          return new Response(
            JSON.stringify({
              id: 5,
              username: 'alice',
              roles: ['Staff'],
              permissions: ['deliveries.read', 'deliveries.write'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/deliveries?')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 4,
                  uid: 'del-4',
                  requestId: 2,
                  requestUid: 'req-2',
                  serverId: 3,
                  serverName: 'DHIS2 Uganda',
                  attemptNumber: 1,
                  status: 'failed',
                  httpStatus: 500,
                  responseBody: '',
                  errorMessage: 'upstream failed',
                  startedAt: '2026-03-10T09:00:00Z',
                  finishedAt: '2026-03-10T09:03:00Z',
                  retryAt: null,
                  createdAt: '2026-03-10T09:00:00Z',
                  updatedAt: '2026-03-10T09:03:00Z',
                },
              ],
              totalCount: 1,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/deliveries/4') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              id: 4,
              uid: 'del-4',
              requestId: 2,
              requestUid: 'req-2',
              serverId: 3,
              serverName: 'DHIS2 Uganda',
              attemptNumber: 1,
              status: 'failed',
              httpStatus: 500,
              responseBody: '',
              errorMessage: 'upstream failed',
              startedAt: '2026-03-10T09:00:00Z',
              finishedAt: '2026-03-10T09:03:00Z',
              retryAt: null,
              createdAt: '2026-03-10T09:00:00Z',
              updatedAt: '2026-03-10T09:03:00Z',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/deliveries/4/retry') && init?.method === 'POST') {
          retryCalls += 1
          return new Response(
            JSON.stringify({
              id: 5,
              uid: 'del-5',
              requestId: 2,
              requestUid: 'req-2',
              serverId: 3,
              serverName: 'DHIS2 Uganda',
              attemptNumber: 2,
              status: 'retrying',
              httpStatus: null,
              responseBody: '',
              errorMessage: '',
              startedAt: null,
              finishedAt: null,
              retryAt: '2026-03-10T09:10:00Z',
              createdAt: '2026-03-10T09:04:00Z',
              updatedAt: '2026-03-10T09:04:00Z',
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

    renderRoute('/deliveries', store)

    expect(await screen.findByText('del-4', {}, { timeout: 5000 })).toBeInTheDocument()
    fireEvent.click(await screen.findByRole('button', { name: 'Actions for del-4' }, { timeout: 5000 }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View' }))

    const dialog = await screen.findByRole('dialog', { name: 'Delivery Detail' })
    expect(within(dialog).getByText('req-2')).toBeInTheDocument()
    expect(within(dialog).getByText(/upstream failed/)).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Retry' }))

    await waitFor(() => expect(retryCalls).toBe(1))
    expect(within(dialog).getByText('del-5')).toBeInTheDocument()
    expect(within(dialog).getByText('retrying')).toBeInTheDocument()
  })

  it('hides retry button without write permission', async () => {
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
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/api/v1/auth/me')) {
          return new Response(
            JSON.stringify({
              id: 5,
              username: 'alice',
              roles: ['Staff'],
              permissions: ['deliveries.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/deliveries?')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 8,
                  uid: 'del-8',
                  requestId: 3,
                  requestUid: 'req-3',
                  serverId: 4,
                  serverName: 'OpenHIM',
                  attemptNumber: 1,
                  status: 'failed',
                  httpStatus: 500,
                  responseBody: '',
                  errorMessage: 'error',
                  startedAt: '2026-03-10T09:00:00Z',
                  finishedAt: '2026-03-10T09:03:00Z',
                  retryAt: null,
                  createdAt: '2026-03-10T09:00:00Z',
                  updatedAt: '2026-03-10T09:03:00Z',
                },
              ],
              totalCount: 1,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/deliveries/8') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              id: 8,
              uid: 'del-8',
              requestId: 3,
              requestUid: 'req-3',
              serverId: 4,
              serverName: 'OpenHIM',
              attemptNumber: 1,
              status: 'failed',
              httpStatus: 500,
              responseBody: '',
              errorMessage: 'error',
              startedAt: '2026-03-10T09:00:00Z',
              finishedAt: '2026-03-10T09:03:00Z',
              retryAt: null,
              createdAt: '2026-03-10T09:00:00Z',
              updatedAt: '2026-03-10T09:03:00Z',
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

    renderRoute('/deliveries', store)

    expect(await screen.findByText('del-8', {}, { timeout: 5000 })).toBeInTheDocument()
    fireEvent.click(await screen.findByRole('button', { name: 'Actions for del-8' }, { timeout: 5000 }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View' }))

    const dialog = await screen.findByRole('dialog', { name: 'Delivery Detail' })
    expect(within(dialog).queryByRole('button', { name: 'Retry' })).not.toBeInTheDocument()
  })
})
