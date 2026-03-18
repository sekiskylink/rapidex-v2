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

class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3
  static instances: MockWebSocket[] = []

  url: string
  readyState = MockWebSocket.CONNECTING
  onopen: ((event: Event) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null

  constructor(url: string | URL) {
    this.url = String(url)
    MockWebSocket.instances.push(this)
  }

  open() {
    this.readyState = MockWebSocket.OPEN
    this.onopen?.({} as Event)
  }

  emitMessage(payload: unknown) {
    this.onmessage?.({
      data: typeof payload === 'string' ? payload : JSON.stringify(payload),
    } as MessageEvent)
  }

  close() {
    this.readyState = MockWebSocket.CLOSED
    this.onclose?.({} as CloseEvent)
  }

  send() {}
}

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
      id: 42,
      username: 'operator',
      roles: ['Staff'],
      permissions,
    },
  })
}

function buildSnapshot() {
  return {
    generatedAt: '2026-03-18T10:00:00Z',
    health: {
      status: 'degraded',
      signals: ['failed_deliveries', 'unhealthy_workers'],
    },
    kpis: {
      requestsToday: 124,
      pendingRequests: 8,
      pendingDeliveries: 5,
      runningDeliveries: 3,
      failedDeliveriesLastHour: 2,
      pollingJobs: 6,
      ingestBacklog: 4,
      healthyWorkers: 5,
      unhealthyWorkers: 1,
    },
    trends: {
      requestsByHour: [
        { bucketStart: '2026-03-18T08:00:00Z', count: 55 },
        { bucketStart: '2026-03-18T09:00:00Z', count: 69 },
      ],
      deliveriesByStatus: [
        { bucketStart: '2026-03-18T09:00:00Z', status: 'succeeded', count: 12 },
        { bucketStart: '2026-03-18T09:00:00Z', status: 'failed', count: 2 },
      ],
      jobsByState: [
        { bucketStart: '2026-03-18T09:00:00Z', status: 'polling', count: 6 },
        { bucketStart: '2026-03-18T09:00:00Z', status: 'failed', count: 1 },
      ],
      failuresByServer: [{ serverId: 3, serverName: 'DHIS2 Uganda', count: 2 }],
    },
    attention: {
      failedDeliveries: {
        total: 2,
        items: [
          {
            id: 9,
            uid: 'del-9',
            requestId: 4,
            requestUid: 'req-4',
            serverId: 3,
            serverName: 'DHIS2 Uganda',
            correlationId: 'corr-9',
            status: 'failed',
            errorMessage: 'REMOTE_TIMEOUT',
            startedAt: '2026-03-18T09:02:00Z',
            finishedAt: '2026-03-18T09:05:00Z',
            nextEligibleAt: '2026-03-18T09:20:00Z',
            updatedAt: '2026-03-18T09:05:00Z',
          },
        ],
      },
      staleRunningDeliveries: {
        total: 1,
        items: [
          {
            id: 10,
            uid: 'del-10',
            requestId: 5,
            requestUid: 'req-5',
            serverId: 3,
            serverName: 'DHIS2 Uganda',
            correlationId: 'corr-10',
            status: 'running',
            errorMessage: '',
            startedAt: '2026-03-18T08:20:00Z',
            finishedAt: null,
            nextEligibleAt: null,
            updatedAt: '2026-03-18T09:01:00Z',
          },
        ],
      },
      stuckJobs: {
        total: 1,
        items: [
          {
            id: 11,
            uid: 'job-11',
            deliveryId: 9,
            deliveryUid: 'del-9',
            requestId: 4,
            requestUid: 'req-4',
            correlationId: 'corr-9',
            remoteJobId: 'remote-11',
            remoteStatus: 'processing',
            currentState: 'polling',
            nextPollAt: '2026-03-18T09:06:00Z',
            updatedAt: '2026-03-18T09:07:00Z',
          },
        ],
      },
      recentIngestFailures: {
        total: 1,
        items: [
          {
            id: 12,
            uid: 'ingest-12',
            originalName: 'payload.csv',
            currentPath: '/tmp/payload.csv',
            status: 'failed',
            lastErrorCode: 'CSV_PARSE',
            lastErrorMessage: 'Invalid column',
            requestId: null,
            failedAt: '2026-03-18T09:04:00Z',
            updatedAt: '2026-03-18T09:04:00Z',
          },
        ],
      },
      unhealthyWorkers: {
        total: 1,
        items: [
          {
            id: 13,
            uid: 'worker-13',
            workerType: 'poll',
            workerName: 'poll-worker',
            status: 'stopped',
            lastHeartbeatAt: '2026-03-18T08:55:00Z',
            startedAt: '2026-03-18T08:00:00Z',
            updatedAt: '2026-03-18T09:05:00Z',
          },
        ],
      },
    },
    workers: {
      heartbeatFreshnessSeconds: 120,
      items: [
        {
          id: 14,
          uid: 'worker-14',
          workerType: 'delivery',
          workerName: 'delivery-worker',
          status: 'running',
          lastHeartbeatAt: '2026-03-18T09:09:30Z',
          startedAt: '2026-03-18T08:00:00Z',
          updatedAt: '2026-03-18T09:09:30Z',
        },
      ],
    },
    recentEvents: [
      {
        type: 'delivery.failed',
        timestamp: '2026-03-18T09:05:00Z',
        severity: 'error',
        entityType: 'delivery',
        entityId: 9,
        entityUid: 'del-9',
        summary: 'Delivery to DHIS2 Uganda failed',
        correlationId: 'corr-9',
        requestId: 4,
        deliveryId: 9,
        jobId: 11,
        workerId: null,
      },
    ],
  }
}

describe('dashboard page', () => {
  let apiRequestSpy: MockInstance

  beforeEach(() => {
    window.localStorage.clear()
    clearAuthSnapshot()
    vi.stubEnv('VITE_API_BASE_URL', 'http://localhost:8080/api/v1')
    window.localStorage.setItem(API_BASE_URL_OVERRIDE_STORAGE_KEY, 'http://localhost:8080/api/v1')
    apiRequestSpy = vi.spyOn(api, 'apiRequest')
    MockWebSocket.instances = []
    vi.stubGlobal('WebSocket', MockWebSocket)
  })

  afterEach(() => {
    cleanup()
    clearAuthSnapshot()
    vi.unstubAllEnvs()
    apiRequestSpy.mockRestore()
  })

  it('renders snapshot-backed widgets for observability readers', async () => {
    authenticate(['requests.read', 'deliveries.read', 'jobs.read', 'observability.read'])
    apiRequestSpy.mockResolvedValue(buildSnapshot())

    renderRoute('/dashboard')

    expect(await screen.findByRole('heading', { name: 'Dashboard', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('Requests Today')).toBeInTheDocument()
    expect(screen.getByText('124')).toBeInTheDocument()
    expect(screen.getByText('Failed Deliveries')).toBeInTheDocument()
    expect(screen.getByText('del-9 • DHIS2 Uganda')).toBeInTheDocument()
    expect(screen.getByText('delivery-worker')).toBeInTheDocument()
    expect(screen.getByText('Delivery to DHIS2 Uganda failed')).toBeInTheDocument()

    await waitFor(() => {
      expect(apiRequestSpy).toHaveBeenCalledWith('/dashboard/operations')
    })
  })

  it('applies live websocket updates on top of the snapshot', async () => {
    authenticate(['requests.read', 'deliveries.read', 'jobs.read', 'observability.read'])
    apiRequestSpy.mockResolvedValue(buildSnapshot())

    renderRoute('/dashboard')

    expect(await screen.findByText('Requests Today')).toBeInTheDocument()

    await waitFor(() => {
      expect(MockWebSocket.instances).toHaveLength(1)
    })
    expect(MockWebSocket.instances[0]?.url).toContain('/dashboard/operations/events?access_token=access-token')

    MockWebSocket.instances[0]?.open()

    expect(await screen.findByText('Live: connected')).toBeInTheDocument()

    MockWebSocket.instances[0]?.emitMessage({
      type: 'request.created',
      timestamp: '2026-03-18T10:05:00Z',
      severity: 'info',
      entityType: 'request',
      entityId: 15,
      entityUid: 'req-15',
      summary: 'Request accepted from EMR',
      correlationId: 'corr-15',
      requestId: 15,
      patch: {
        kpi: 'pendingRequests',
        op: 'increment',
        value: 1,
      },
      invalidations: ['kpis', 'recentEvents'],
    })

    expect(await screen.findByText('Request accepted from EMR')).toBeInTheDocument()
    expect(apiRequestSpy).toHaveBeenCalledTimes(1)

    MockWebSocket.instances[0]?.close()

    await waitFor(() => {
      expect(MockWebSocket.instances).toHaveLength(2)
    }, { timeout: 3000 })

    MockWebSocket.instances[1]?.open()

    await waitFor(() => {
      expect(apiRequestSpy).toHaveBeenCalledTimes(2)
    })
  })

  it('drills down to filtered deliveries', async () => {
    authenticate(['requests.read', 'deliveries.read', 'jobs.read', 'observability.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path === '/dashboard/operations') {
        return buildSnapshot()
      }
      if (path.startsWith('/deliveries?')) {
        return { items: [], totalCount: 0, page: 1, pageSize: 25 }
      }
      return {}
    })

    renderRoute('/dashboard')

    expect(await screen.findByText('Requests Today')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Open Failed Deliveries' }))

    await waitFor(() => {
      expect(apiRequestSpy).toHaveBeenCalledWith(expect.stringContaining('/deliveries?'))
    })
    expect(
      apiRequestSpy.mock.calls.some(
        ([path]) =>
          typeof path === 'string' &&
          path.includes('/deliveries?') &&
          path.includes('status=failed'),
      ),
    ).toBe(true)
  })

  it('drills down to observability trace filters from recent events', async () => {
    authenticate(['requests.read', 'deliveries.read', 'jobs.read', 'observability.read'])
    apiRequestSpy.mockImplementation(async (path: string) => {
      if (path === '/dashboard/operations') {
        return buildSnapshot()
      }
      if (path.startsWith('/observability/events?')) {
        return { items: [], totalCount: 0, page: 1, pageSize: 25 }
      }
      if (path.startsWith('/observability/workers?')) {
        return { items: [], totalCount: 0, page: 1, pageSize: 25 }
      }
      if (path.startsWith('/observability/rate-limits?')) {
        return { items: [], totalCount: 0, page: 1, pageSize: 25 }
      }
      return {}
    })

    renderRoute('/dashboard')
    expect(await screen.findByText('Delivery to DHIS2 Uganda failed')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Trace' }))

    await waitFor(() => {
      expect(
        apiRequestSpy.mock.calls.some(
          ([path]) =>
            typeof path === 'string' &&
            path.includes('/observability/events?') &&
            path.includes('correlationId=corr-9') &&
            path.includes('deliveryId=9'),
        ),
      ).toBe(true)
    })
  })

  it('does not fetch the snapshot when the user lacks observability access', async () => {
    authenticate(['requests.read', 'deliveries.read'])

    renderRoute('/dashboard')

    expect(await screen.findByRole('heading', { name: 'Dashboard', level: 1 })).toBeInTheDocument()
    expect(
      screen.getByText(/Operations widgets require/i),
    ).toBeInTheDocument()
    expect(apiRequestSpy).not.toHaveBeenCalled()
    expect(MockWebSocket.instances).toHaveLength(0)
  })
})
