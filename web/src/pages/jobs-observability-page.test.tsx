import React from 'react'
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
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

describe('jobs and observability pages', () => {
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

  it('renders jobs grid and job detail poll history', async () => {
    authenticate(['jobs.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path.includes('/jobs?')) {
        return {
          items: [
            {
              id: 4,
              uid: 'job-4',
              deliveryAttemptId: 8,
              deliveryUid: 'del-8',
              requestId: 3,
              requestUid: 'req-3',
              remoteJobId: 'remote-4',
              pollUrl: 'https://remote/jobs/4',
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
        }
      }
      if (path === '/jobs/4') {
        return {
          id: 4,
          uid: 'job-4',
          deliveryAttemptId: 8,
          deliveryUid: 'del-8',
          requestId: 3,
          requestUid: 'req-3',
          remoteJobId: 'remote-4',
          pollUrl: 'https://remote/jobs/4',
          remoteStatus: 'processing',
          terminalState: '',
          currentState: 'polling',
          nextPollAt: '2026-03-12T09:05:00Z',
          completedAt: null,
          remoteResponse: { state: 'processing' },
          createdAt: '2026-03-12T09:00:00Z',
          updatedAt: '2026-03-12T09:01:00Z',
        }
      }
      if (path.includes('/jobs/4/polls')) {
        return {
          items: [
            {
              id: 1,
              asyncTaskId: 4,
              polledAt: '2026-03-12T09:01:00Z',
              statusCode: 202,
              remoteStatus: 'processing',
              responseBody: '{"state":"processing"}',
              errorMessage: '',
              durationMs: 125,
            },
          ],
          totalCount: 1,
          page: 1,
          pageSize: 50,
        }
      }
      return {}
    })

    renderRoute('/jobs')

    expect(await screen.findByRole('heading', { name: 'Jobs', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('job-4')).toBeInTheDocument()
    expect(screen.getByText('del-8')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Actions for job-4' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View' }))

    const dialog = await screen.findByRole('dialog', { name: 'Job Detail' })
    expect(within(dialog).getByText('req-3')).toBeInTheDocument()
    expect(within(dialog).getByText('https://remote/jobs/4')).toBeInTheDocument()
    expect(within(dialog).getAllByText(/processing/).length).toBeGreaterThan(0)
  })

  it('renders observability workers and rate limits', async () => {
    authenticate(['observability.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path.includes('/observability/workers?')) {
        return {
          items: [
            {
              id: 1,
              uid: 'worker-1',
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
        }
      }
      if (path.includes('/observability/rate-limits?')) {
        return {
          items: [
            {
              id: 2,
              uid: 'policy-2',
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
        }
      }
      return {}
    })

    renderRoute('/observability')

    expect(await screen.findByRole('heading', { name: 'Observability', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('poll-worker')).toBeInTheDocument()
    expect(screen.getByText('Global policy')).toBeInTheDocument()
    expect(screen.getByText('running')).toBeInTheDocument()
  })
})
