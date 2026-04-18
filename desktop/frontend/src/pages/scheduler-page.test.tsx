import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
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

describe('desktop scheduler pages', () => {
  beforeEach(async () => {
    await clearSession()
  })

  afterEach(async () => {
    cleanup()
    await clearSession()
  })

  it('renders scheduler jobs route and rows from backend API', async () => {
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
              permissions: ['scheduler.read', 'scheduler.write'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/scheduler/jobs?')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 1,
                  uid: 'sch-1',
                  code: 'nightly-sync',
                  name: 'Nightly Sync',
                  description: 'Nightly integration sync',
                  jobCategory: 'integration',
                  jobType: 'dhis2.sync',
                  scheduleType: 'interval',
                  scheduleExpr: '15m',
                  timezone: 'UTC',
                  enabled: true,
                  allowConcurrentRuns: false,
                  config: { serverCode: 'dhis2' },
                  nextRunAt: '2026-04-18T21:15:00Z',
                  latestRunStatus: 'succeeded',
                  createdAt: '2026-04-18T21:00:00Z',
                  updatedAt: '2026-04-18T21:00:00Z',
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

    renderRoute('/scheduler', store)

    expect(await screen.findByRole('heading', { name: 'Scheduler', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('nightly-sync')).toBeInTheDocument()
    expect(screen.getByText('Nightly Sync')).toBeInTheDocument()
    expect(screen.getByText('succeeded')).toBeInTheDocument()
  })

  it('submits create scheduled job form through backend API', async () => {
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
              permissions: ['scheduler.read', 'scheduler.write'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/scheduler/jobs') && init?.method === 'POST') {
          createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
          return new Response(
            JSON.stringify({
              id: 3,
              uid: 'sch-3',
              code: 'cleanup',
              name: 'Cleanup',
              description: 'Cleanup job',
              jobCategory: 'maintenance',
              jobType: 'purge_old_logs',
              scheduleType: 'cron',
              scheduleExpr: '0 2 * * *',
              timezone: 'UTC',
              enabled: true,
              allowConcurrentRuns: true,
              config: { retainDays: 30 },
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/scheduler/jobs/3')) {
          return new Response(
            JSON.stringify({
              id: 3,
              uid: 'sch-3',
              code: 'cleanup',
              name: 'Cleanup',
              description: 'Cleanup job',
              jobCategory: 'maintenance',
              jobType: 'purge_old_logs',
              scheduleType: 'cron',
              scheduleExpr: '0 2 * * *',
              timezone: 'UTC',
              enabled: true,
              allowConcurrentRuns: true,
              config: { retainDays: 30 },
              nextRunAt: '2026-04-19T02:00:00Z',
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

    renderRoute('/scheduler/new', store)

    fireEvent.change(await screen.findByLabelText('Code'), { target: { value: 'cleanup' } })
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'Cleanup' } })
    fireEvent.change(screen.getByLabelText('Description'), { target: { value: 'Cleanup job' } })
    fireEvent.mouseDown(screen.getByLabelText('Job Category'))
    fireEvent.click(await screen.findByRole('option', { name: 'Maintenance' }))
    fireEvent.mouseDown(screen.getByLabelText('Job Type'))
    fireEvent.click(await screen.findByRole('option', { name: 'Purge Old Logs' }))
    fireEvent.mouseDown(screen.getByLabelText('Schedule Type'))
    fireEvent.click(await screen.findByRole('option', { name: 'Cron' }))
    fireEvent.change(screen.getByLabelText('Schedule Expression'), { target: { value: '0 2 * * *' } })
    fireEvent.change(screen.getByLabelText('Config JSON'), { target: { value: '{"retainDays":30}' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create Scheduled Job' }))

    await waitFor(() => expect(createPayload).not.toBeNull())
    expect(createPayload).toMatchObject({
      code: 'cleanup',
      name: 'Cleanup',
      jobCategory: 'maintenance',
      jobType: 'purge_old_logs',
      scheduleType: 'cron',
      scheduleExpr: '0 2 * * *',
      config: { retainDays: 30 },
    })
  })
})
