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

describe('deliveries page', () => {
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

  it('renders deliveries grid rows from mocked API', async () => {
    authenticate(['deliveries.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path.includes('/deliveries?')) {
        return {
          items: [
            {
              id: 11,
              uid: 'del-11',
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
        }
      }
      return {}
    })

    renderRoute('/deliveries')

    expect(await screen.findByRole('heading', { name: 'Deliveries', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('del-11')).toBeInTheDocument()
    expect(screen.getByText('req-3')).toBeInTheDocument()
    expect(screen.getByText('DHIS2 Uganda')).toBeInTheDocument()
  })

  it('delivery detail renders and retry action calls API', async () => {
    authenticate(['deliveries.read', 'deliveries.write'])
    let retryCalls = 0
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/deliveries?')) {
        return {
          items: [
            {
              id: 5,
              uid: 'del-5',
              requestId: 2,
              requestUid: 'req-2',
              serverId: 9,
              serverName: 'DHIS2 Uganda',
              attemptNumber: 1,
              status: 'failed',
              httpStatus: 500,
              responseBody: '',
              errorMessage: 'upstream failed',
              startedAt: '2026-03-10T09:00:00Z',
              finishedAt: '2026-03-10T09:04:00Z',
              retryAt: null,
              createdAt: '2026-03-10T09:00:00Z',
              updatedAt: '2026-03-10T09:04:00Z',
            },
          ],
          totalCount: 1,
          page: 1,
          pageSize: 25,
        }
      }
      if (path === '/deliveries/5') {
        return {
          id: 5,
          uid: 'del-5',
          requestId: 2,
          requestUid: 'req-2',
          serverId: 9,
          serverName: 'DHIS2 Uganda',
          attemptNumber: 1,
          status: 'failed',
          httpStatus: 500,
          responseBody: '',
          errorMessage: 'upstream failed',
          startedAt: '2026-03-10T09:00:00Z',
          finishedAt: '2026-03-10T09:04:00Z',
          retryAt: null,
          createdAt: '2026-03-10T09:00:00Z',
          updatedAt: '2026-03-10T09:04:00Z',
        }
      }
      if (path === '/deliveries/5/retry' && init?.method === 'POST') {
        retryCalls += 1
        return {
          id: 6,
          uid: 'del-6',
          requestId: 2,
          requestUid: 'req-2',
          serverId: 9,
          serverName: 'DHIS2 Uganda',
          attemptNumber: 2,
          status: 'retrying',
          httpStatus: null,
          responseBody: '',
          errorMessage: '',
          startedAt: null,
          finishedAt: null,
          retryAt: '2026-03-10T09:10:00Z',
          createdAt: '2026-03-10T09:05:00Z',
          updatedAt: '2026-03-10T09:05:00Z',
        }
      }
      return {}
    })

    renderRoute('/deliveries')

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for del-5' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View' }))

    const dialog = await screen.findByRole('dialog', { name: 'Delivery Detail' })
    expect(within(dialog).getByText('req-2')).toBeInTheDocument()
    expect(within(dialog).getByText(/upstream failed/)).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Retry' }))

    await waitFor(() => expect(retryCalls).toBe(1))
    expect(within(dialog).getByText('del-6')).toBeInTheDocument()
    expect(within(dialog).getByText('retrying')).toBeInTheDocument()
  })

  it('hides retry action without write permission', async () => {
    authenticate(['deliveries.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path.includes('/deliveries?')) {
        return {
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
        }
      }
      if (path === '/deliveries/8') {
        return {
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
        }
      }
      return {}
    })

    renderRoute('/deliveries')

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for del-8' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View' }))

    const dialog = await screen.findByRole('dialog', { name: 'Delivery Detail' })
    expect(within(dialog).queryByRole('button', { name: 'Retry' })).not.toBeInTheDocument()
  })
})
