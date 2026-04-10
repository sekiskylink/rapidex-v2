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

describe('requests page', () => {
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

  it('renders requests grid rows from mocked API', async () => {
    authenticate(['requests.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path.includes('/requests?')) {
        return {
          items: [
            {
              id: 11,
              uid: 'req-11',
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
              extras: { priority: 'high' },
              createdAt: '2026-03-10T09:00:00Z',
              updatedAt: '2026-03-10T10:00:00Z',
              latestDeliveryUid: 'del-11',
              latestDeliveryStatus: 'pending',
              latestAsyncTaskUid: '',
              latestAsyncState: '',
              latestAsyncRemoteJobId: '',
              latestAsyncPollUrl: '',
              awaitingAsync: false,
              targets: [{ id: 1, uid: 'target-1', serverId: 3, serverName: 'DHIS2 Uganda', serverCode: 'dhis2-ug', targetKind: 'primary', status: 'blocked', blockedReason: 'window_closed', deferredUntil: '2026-03-10T12:00:00Z', latestDeliveryUid: 'del-11', latestDeliveryStatus: 'pending', latestAsyncTaskUid: '', latestAsyncState: '', awaitingAsync: false }],
              dependencies: [],
            },
          ],
          totalCount: 1,
          page: 1,
          pageSize: 25,
        }
      }
      return {}
    })

    renderRoute('/requests')

    expect(await screen.findByRole('heading', { name: 'Requests', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('req-11')).toBeInTheDocument()
    expect(screen.getByText('DHIS2 Uganda')).toBeInTheDocument()
    expect(screen.getByText(/window_closed/)).toBeInTheDocument()
  })

  it('create request submits payload through API', async () => {
    authenticate(['requests.read', 'requests.write'])
    let createPayload: Record<string, unknown> | null = null
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/requests?')) {
        return { items: [], totalCount: 0, page: 1, pageSize: 25 }
      }
      if (path.includes('/servers?')) {
        return {
          items: [
            { id: 4, name: 'DHIS2 Uganda', code: 'dhis2-ug' },
            { id: 9, name: 'DHIS2 Mirror A', code: 'mirror-a' },
            { id: 12, name: 'DHIS2 Mirror B', code: 'mirror-b' },
          ],
          totalCount: 3,
          page: 1,
          pageSize: 200,
        }
      }
      if (path === '/requests' && init?.method === 'POST') {
        createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
        return { id: 22, uid: 'req-22', status: 'pending' }
      }
      return {}
    })

    renderRoute('/requests')

    fireEvent.click(await screen.findByRole('button', { name: 'Create Request' }))
    const dialog = await screen.findByRole('dialog', { name: 'Create Request' })
    expect(within(dialog).getByTestId('web-request-create-form-grid')).toBeInTheDocument()
    fireEvent.mouseDown(within(dialog).getByLabelText('Destination Server'))
    fireEvent.click(await screen.findByRole('option', { name: 'DHIS2 Uganda (dhis2-ug)' }))
    fireEvent.mouseDown(within(dialog).getByLabelText('Additional Destination Servers'))
    fireEvent.click(await screen.findByRole('option', { name: 'DHIS2 Mirror A (mirror-a)' }))
    fireEvent.mouseDown(within(dialog).getByLabelText('Additional Destination Servers'))
    fireEvent.click(await screen.findByRole('option', { name: 'DHIS2 Mirror B (mirror-b)' }))
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Dependency Request IDs' }), { target: { value: '7, 8' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Source System' }), { target: { value: 'emr' } })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Correlation ID' }), { target: { value: 'corr-22' } })
    fireEvent.mouseDown(within(dialog).getByLabelText('Send As'))
    fireEvent.click(await screen.findByRole('option', { name: 'Query Params' }))
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Payload JSON' }), {
      target: { value: '{"trackedEntity":"abc","orgUnit":"ou-1"}' },
    })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Create' }))

    await waitFor(() => expect(createPayload).not.toBeNull())
    expect(createPayload).toMatchObject({
      destinationServerId: 4,
      destinationServerIds: [9, 12],
      dependencyRequestIds: [7, 8],
      sourceSystem: 'emr',
      correlationId: 'corr-22',
      batchId: '',
      idempotencyKey: '',
      payloadFormat: 'json',
      submissionBinding: 'query',
      payload: { trackedEntity: 'abc', orgUnit: 'ou-1' },
    })
  }, 20000)

  it('request detail renders from backend API', async () => {
    authenticate(['requests.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path.includes('/requests?')) {
        return {
          items: [
            {
              id: 5,
              uid: 'req-5',
              sourceSystem: 'emr',
              destinationServerId: 4,
              destinationServerName: 'DHIS2 Uganda',
              batchId: 'batch-5',
              correlationId: 'corr-5',
              idempotencyKey: 'idem-5',
              payloadBody: '{"trackedEntity":"abc"}',
              payloadFormat: 'json',
              submissionBinding: 'body',
              payload: { trackedEntity: 'abc' },
              urlSuffix: '/api/data',
              status: 'completed',
              statusReason: '',
              deferredUntil: null,
              extras: { priority: 'high' },
              createdAt: '2026-03-10T09:00:00Z',
              updatedAt: '2026-03-10T10:00:00Z',
              latestDeliveryUid: 'del-5',
              latestDeliveryStatus: 'succeeded',
              latestAsyncTaskUid: '',
              latestAsyncState: '',
              latestAsyncRemoteJobId: '',
              latestAsyncPollUrl: '',
              awaitingAsync: false,
              targets: [
                { id: 1, uid: 'target-5a', serverId: 4, serverName: 'DHIS2 Uganda', serverCode: 'dhis2-ug', targetKind: 'primary', status: 'succeeded', blockedReason: '', deferredUntil: null, latestDeliveryUid: 'del-5a', latestDeliveryStatus: 'succeeded', latestAsyncTaskUid: '', latestAsyncState: '', awaitingAsync: false },
                { id: 2, uid: 'target-5b', serverId: 6, serverName: 'Mirror', serverCode: 'mirror', targetKind: 'cc', status: 'succeeded', blockedReason: '', deferredUntil: null, latestDeliveryUid: 'del-5b', latestDeliveryStatus: 'succeeded', latestAsyncTaskUid: '', latestAsyncState: '', awaitingAsync: false },
              ],
              dependencies: [{ requestId: 5, dependsOnRequestId: 2, dependsOnUid: 'req-2', status: 'completed', statusReason: '', deferredUntil: null, dependsOnDestinationServerName: 'Bootstrap' }],
            },
          ],
          totalCount: 1,
          page: 1,
          pageSize: 25,
        }
      }
      if (path === '/requests/5') {
        return {
          id: 5,
          uid: 'req-5',
          sourceSystem: 'emr',
          destinationServerId: 4,
          destinationServerName: 'DHIS2 Uganda',
          batchId: 'batch-5',
          correlationId: 'corr-5',
          idempotencyKey: 'idem-5',
          payloadBody: '{"trackedEntity":"abc"}',
          payloadFormat: 'json',
          submissionBinding: 'body',
          payload: { trackedEntity: 'abc' },
          urlSuffix: '/api/data',
          status: 'completed',
          statusReason: '',
          deferredUntil: null,
          extras: { priority: 'high' },
          createdAt: '2026-03-10T09:00:00Z',
          updatedAt: '2026-03-10T10:00:00Z',
          latestDeliveryUid: 'del-5',
          latestDeliveryStatus: 'succeeded',
          latestAsyncTaskUid: '',
          latestAsyncState: '',
          latestAsyncRemoteJobId: '',
          latestAsyncPollUrl: '',
          awaitingAsync: false,
          targets: [
            { id: 1, uid: 'target-5a', serverId: 4, serverName: 'DHIS2 Uganda', serverCode: 'dhis2-ug', targetKind: 'primary', status: 'succeeded', blockedReason: '', deferredUntil: null, latestDeliveryUid: 'del-5a', latestDeliveryStatus: 'succeeded', latestAsyncTaskUid: '', latestAsyncState: '', awaitingAsync: false },
            { id: 2, uid: 'target-5b', serverId: 6, serverName: 'Mirror', serverCode: 'mirror', targetKind: 'cc', status: 'blocked', blockedReason: 'dependency_blocked', deferredUntil: null, latestDeliveryUid: 'del-5b', latestDeliveryStatus: 'pending', latestAsyncTaskUid: '', latestAsyncState: '', awaitingAsync: false },
          ],
          dependencies: [{ requestId: 5, dependsOnRequestId: 2, dependsOnUid: 'req-2', status: 'failed', statusReason: 'dependency_failed', deferredUntil: null, dependsOnDestinationServerName: 'Bootstrap' }],
        }
      }
      return {}
    })

    renderRoute('/requests')

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for req-5' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View' }))

    const dialog = await screen.findByRole('dialog', { name: 'Request Detail' })
    expect(within(dialog).getByText('req-5')).toBeInTheDocument()
    expect(within(dialog).getByText('2 targets')).toBeInTheDocument()
    expect(within(dialog).getByText('dependency_blocked')).toBeInTheDocument()
    expect(within(dialog).getByText('req-2')).toBeInTheDocument()
    expect(within(dialog).getByText(/trackedEntity/)).toBeInTheDocument()
    expect(within(dialog).getByText(/priority/)).toBeInTheDocument()
  })

  it('opens request body in a formatted dialog from row actions', async () => {
    authenticate(['requests.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path.includes('/requests?')) {
        return {
          items: [
            {
              id: 8,
              uid: 'req-8',
              sourceSystem: 'emr',
              destinationServerId: 4,
              destinationServerName: 'DHIS2 Uganda',
              batchId: 'batch-8',
              correlationId: 'corr-8',
              idempotencyKey: 'idem-8',
              payloadBody: '{"trackedEntity":"body-8","program":"alpha"}',
              payloadFormat: 'json',
              submissionBinding: 'body',
              payload: { trackedEntity: 'body-8', program: 'alpha' },
              urlSuffix: '/api/data',
              status: 'pending',
              statusReason: '',
              deferredUntil: null,
              extras: {},
              createdAt: '2026-03-10T09:00:00Z',
              updatedAt: '2026-03-10T10:00:00Z',
              latestDeliveryUid: 'del-8',
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
        }
      }
      return {}
    })

    renderRoute('/requests')

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for req-8' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View Body' }))

    const dialog = await screen.findByRole('dialog', { name: 'Request Body: req-8' })
    expect(within(dialog).getByText('Formatted JSON')).toBeInTheDocument()
    expect(within(dialog).getByText(/trackedEntity/)).toBeInTheDocument()
    expect(within(dialog).getByText(/program/)).toBeInTheDocument()
  })

  it('deletes a request from row actions after confirmation', async () => {
    authenticate(['requests.read', 'requests.write'])
    let deletePath = ''
    let deleteMethod = ''
    apiRequestSpy.mockImplementation(async (path: string, init?: RequestInit) => {
      if (path.includes('/requests?')) {
        return {
          items: [
            {
              id: 15,
              uid: 'req-15',
              sourceSystem: 'emr',
              destinationServerId: 4,
              destinationServerName: 'DHIS2 Uganda',
              batchId: 'batch-15',
              correlationId: 'corr-15',
              idempotencyKey: 'idem-15',
              payloadBody: '{"trackedEntity":"delete-me"}',
              payloadFormat: 'json',
              submissionBinding: 'body',
              payload: { trackedEntity: 'delete-me' },
              urlSuffix: '/api/data',
              status: 'pending',
              statusReason: '',
              deferredUntil: null,
              extras: {},
              createdAt: '2026-03-10T09:00:00Z',
              updatedAt: '2026-03-10T10:00:00Z',
              latestDeliveryUid: 'del-15',
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
        }
      }
      if (path === '/requests/15' && init?.method === 'DELETE') {
        deletePath = path
        deleteMethod = init.method
        return {}
      }
      return {}
    })

    renderRoute('/requests')

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for req-15' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Delete' }))
    const dialog = await screen.findByRole('dialog', { name: 'Delete request' })
    expect(within(dialog).getByText(/related deliveries/)).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(deletePath).toBe('/requests/15'))
    expect(deleteMethod).toBe('DELETE')
  })

  it('hides create button without write permission', async () => {
    authenticate(['requests.read'])
    apiRequestSpy.mockResolvedValue({ items: [], totalCount: 0, page: 1, pageSize: 25 })

    renderRoute('/requests')

    expect(await screen.findByRole('heading', { name: 'Requests', level: 1 })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create Request' })).not.toBeInTheDocument()
  })
})
