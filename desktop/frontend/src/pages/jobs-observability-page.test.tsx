import React from 'react'
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
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

describe('desktop jobs and observability pages', () => {
  beforeEach(async () => {
    await clearSession()
  })

  afterEach(async () => {
    cleanup()
    await clearSession()
  })

  it('renders jobs route and job detail poll history', async () => {
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
              permissions: ['jobs.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/jobs?')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 1,
                  uid: 'job-1',
                  deliveryAttemptId: 7,
                  deliveryUid: 'del-7',
                  requestId: 3,
                  requestUid: 'req-3',
                  remoteJobId: 'remote-1',
                  pollUrl: 'https://remote/jobs/1',
                  remoteStatus: 'processing',
                  terminalState: '',
                  currentState: 'polling',
                  nextPollAt: '2026-03-12T09:05:00Z',
                  completedAt: null,
                  remoteResponse: { state: 'processing' },
                  createdAt: '2026-03-12T09:00:00Z',
                  updatedAt: '2026-03-12T09:01:00Z',
                },
              ],
              totalCount: 1,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/jobs/1')) {
          return new Response(
            JSON.stringify({
              id: 1,
              uid: 'job-1',
              deliveryAttemptId: 7,
              deliveryUid: 'del-7',
              requestId: 3,
              requestUid: 'req-3',
              remoteJobId: 'remote-1',
              pollUrl: 'https://remote/jobs/1',
              remoteStatus: 'processing',
              terminalState: '',
              currentState: 'polling',
              nextPollAt: '2026-03-12T09:05:00Z',
              completedAt: null,
              remoteResponse: { state: 'processing' },
              createdAt: '2026-03-12T09:00:00Z',
              updatedAt: '2026-03-12T09:01:00Z',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/jobs/1/polls')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 4,
                  asyncTaskId: 1,
                  polledAt: '2026-03-12T09:01:00Z',
                  statusCode: 202,
                  remoteStatus: 'processing',
                  responseBody: '{"state":"processing"}',
                  errorMessage: '',
                  durationMs: 120,
                },
              ],
              totalCount: 1,
              page: 1,
              pageSize: 50,
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

    renderRoute('/jobs', store)

    expect(await screen.findByRole('heading', { name: 'Jobs', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('job-1')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Actions for job-1' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View' }))

    const dialog = await screen.findByRole('dialog', { name: 'Job Detail' })
    expect(within(dialog).getByText('req-3')).toBeInTheDocument()
    expect(within(dialog).getByText('https://remote/jobs/1')).toBeInTheDocument()
  })

  it('renders observability route and rows', async () => {
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
              permissions: ['observability.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/observability/workers?')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 2,
                  uid: 'worker-2',
                  workerType: 'poll',
                  workerName: 'poll-worker',
                  status: 'running',
                  startedAt: '2026-03-12T09:00:00Z',
                  lastHeartbeatAt: '2026-03-12T09:01:00Z',
                },
              ],
              totalCount: 1,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/observability/rate-limits?')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 3,
                  uid: 'policy-3',
                  name: 'Global policy',
                  scopeType: 'global',
                  scopeRef: '',
                  rps: 10,
                  burst: 20,
                  maxConcurrency: 3,
                  timeoutMs: 500,
                  isActive: true,
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

    renderRoute('/observability', store)

    expect(await screen.findByRole('heading', { name: 'Observability', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('poll-worker')).toBeInTheDocument()
    expect(screen.getByText('Global policy')).toBeInTheDocument()
  })
})
