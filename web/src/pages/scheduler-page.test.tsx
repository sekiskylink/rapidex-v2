import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
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
      id: 9,
      username: 'scheduler-operator',
      roles: ['Staff'],
      permissions,
    },
  })
}

describe('scheduler pages', () => {
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

  it('renders scheduler jobs grid rows from mocked API', async () => {
    authenticate(['scheduler.read', 'scheduler.write'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path.includes('/scheduler/jobs?')) {
        return {
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
        }
      }
      return {}
    })

    renderRoute('/scheduler')

    expect(await screen.findByRole('heading', { name: 'Scheduler', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('nightly-sync')).toBeInTheDocument()
    expect(screen.getByText('Nightly Sync')).toBeInTheDocument()
    expect(screen.getByText('succeeded')).toBeInTheDocument()
  })

  it('submits create scheduled job form through API', async () => {
    authenticate(['scheduler.read', 'scheduler.write'])
    let createPayload: Record<string, unknown> | null = null
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path === '/scheduler/jobs' && init?.method === 'POST') {
        createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return {
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
          config: { dryRun: false, batchSize: 500, maxAgeDays: 30 },
        }
      }
      if (path === '/scheduler/jobs/3') {
        return {
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
          config: { dryRun: false, batchSize: 500, maxAgeDays: 30 },
          nextRunAt: '2026-04-19T02:00:00Z',
        }
      }
      return {}
    })

    renderRoute('/scheduler/new')

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
    fireEvent.change(screen.getByLabelText('Batch Size'), { target: { value: '250' } })
    fireEvent.change(screen.getByLabelText('Max Age Days'), { target: { value: '45' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create Scheduled Job' }))

    await waitFor(() => expect(createPayload).not.toBeNull())
    expect(createPayload).toMatchObject({
      code: 'cleanup',
      name: 'Cleanup',
      jobCategory: 'maintenance',
      jobType: 'purge_old_logs',
      scheduleType: 'cron',
      scheduleExpr: '0 2 * * *',
      config: { dryRun: false, batchSize: 250, maxAgeDays: 45 },
    })
  })

  it('submits URL call scheduled job config through API', async () => {
    authenticate(['scheduler.read', 'scheduler.write'])
    let createPayload: Record<string, unknown> | null = null
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path === '/scheduler/jobs' && init?.method === 'POST') {
        createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return {
          id: 4,
          uid: 'sch-4',
          code: 'url-call',
          name: 'URL Call',
          description: '',
          jobCategory: 'integration',
          jobType: 'url_call',
          scheduleType: 'interval',
          scheduleExpr: '15m',
          timezone: 'UTC',
          enabled: true,
          allowConcurrentRuns: false,
          config: {},
        }
      }
      if (path === '/scheduler/jobs/4') {
        return { id: 4, uid: 'sch-4', code: 'url-call', name: 'URL Call', config: {}, latestRunStatus: '' }
      }
      return {}
    })

    renderRoute('/scheduler/new')

    fireEvent.change(await screen.findByLabelText('Code'), { target: { value: 'url-call' } })
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'URL Call' } })
    fireEvent.mouseDown(screen.getByLabelText('Job Type'))
    fireEvent.click(await screen.findByRole('option', { name: 'URL Call' }))
    fireEvent.change(screen.getByLabelText('Destination Server UID'), { target: { value: 'srv-uid' } })
    fireEvent.change(screen.getByLabelText('URL Suffix'), { target: { value: '/api/ping' } })
    fireEvent.change(screen.getByLabelText('Payload JSON'), { target: { value: '{"ping":true}' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create Scheduled Job' }))

    await waitFor(() => expect(createPayload).not.toBeNull())
    expect(createPayload).toMatchObject({
      code: 'url-call',
      jobCategory: 'integration',
      jobType: 'url_call',
      config: {
        destinationServerUid: 'srv-uid',
        urlSuffix: '/api/ping',
        payloadFormat: 'json',
        submissionBinding: 'body',
        payload: { ping: true },
      },
    })
  })

  it('submits request exchange scheduled job config through API', async () => {
    authenticate(['scheduler.read', 'scheduler.write'])
    let createPayload: Record<string, unknown> | null = null
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path === '/scheduler/jobs' && init?.method === 'POST') {
        createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return {
          id: 5,
          uid: 'sch-5',
          code: 'exchange',
          name: 'Exchange',
          description: '',
          jobCategory: 'integration',
          jobType: 'request_exchange',
          scheduleType: 'interval',
          scheduleExpr: '15m',
          timezone: 'UTC',
          enabled: true,
          allowConcurrentRuns: false,
          config: {},
        }
      }
      if (path === '/scheduler/jobs/5') {
        return { id: 5, uid: 'sch-5', code: 'exchange', name: 'Exchange', config: {}, latestRunStatus: '' }
      }
      return {}
    })

    renderRoute('/scheduler/new')

    fireEvent.change(await screen.findByLabelText('Code'), { target: { value: 'exchange' } })
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'Exchange' } })
    fireEvent.mouseDown(screen.getByLabelText('Job Type'))
    fireEvent.click(await screen.findByRole('option', { name: 'Request Exchange' }))
    fireEvent.change(screen.getByLabelText('Destination Server UID'), { target: { value: 'srv-uid' } })
    fireEvent.change(screen.getByLabelText('Additional Destination UIDs'), { target: { value: 'srv-cc-1, srv-cc-2' } })
    fireEvent.change(screen.getByLabelText('Idempotency Key Prefix'), { target: { value: 'daily' } })
    fireEvent.change(screen.getByLabelText('Payload JSON'), { target: { value: '{"event":"daily"}' } })
    fireEvent.change(screen.getByLabelText('Metadata JSON'), { target: { value: '{"owner":"scheduler"}' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create Scheduled Job' }))

    await waitFor(() => expect(createPayload).not.toBeNull())
    expect(createPayload).toMatchObject({
      code: 'exchange',
      jobType: 'request_exchange',
      config: {
        sourceSystem: 'scheduler',
        destinationServerUid: 'srv-uid',
        destinationServerUids: ['srv-cc-1', 'srv-cc-2'],
        idempotencyKeyPrefix: 'daily',
        payload: { event: 'daily' },
        metadata: { owner: 'scheduler' },
      },
    })
  })

  it('submits RapidPro reporter sync scheduled job config through API', async () => {
    authenticate(['scheduler.read', 'scheduler.write'])
    let createPayload: Record<string, unknown> | null = null
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path === '/scheduler/jobs' && init?.method === 'POST') {
        createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return {
          id: 6,
          uid: 'sch-6',
          code: 'rapidpro-reporter-sync',
          name: 'RapidPro Reporter Sync',
          description: '',
          jobCategory: 'integration',
          jobType: 'rapidpro_reporter_sync',
          scheduleType: 'interval',
          scheduleExpr: '30m',
          timezone: 'UTC',
          enabled: true,
          allowConcurrentRuns: false,
          config: {},
        }
      }
      if (path === '/scheduler/jobs/6') {
        return { id: 6, uid: 'sch-6', code: 'rapidpro-reporter-sync', name: 'RapidPro Reporter Sync', config: {}, latestRunStatus: '' }
      }
      return {}
    })

    renderRoute('/scheduler/new')

    fireEvent.change(await screen.findByLabelText('Code'), { target: { value: 'rapidpro-reporter-sync' } })
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'RapidPro Reporter Sync' } })
    fireEvent.mouseDown(screen.getByLabelText('Job Type'))
    fireEvent.click(await screen.findByRole('option', { name: 'RapidPro Reporter Sync' }))
    fireEvent.change(screen.getByLabelText('Batch Size'), { target: { value: '150' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create Scheduled Job' }))

    await waitFor(() => expect(createPayload).not.toBeNull())
    expect(createPayload).toMatchObject({
      code: 'rapidpro-reporter-sync',
      jobType: 'rapidpro_reporter_sync',
      config: {
        dryRun: false,
        batchSize: 150,
        onlyActive: true,
      },
    })
  })
})
