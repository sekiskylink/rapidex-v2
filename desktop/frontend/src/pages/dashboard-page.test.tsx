import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearSession, configureSessionStorage, setSession } from '../auth/session'
import { createAppRouter } from '../routes'
import { AppThemeProvider } from '../ui/theme'
import {
  defaultSettings,
  type AppSettings,
  type SaveSettingsPatch,
  type SettingsStore,
} from '../settings/types'

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

describe('desktop dashboard page', () => {
  beforeEach(async () => {
    vi.restoreAllMocks()
    MockWebSocket.instances = []
    vi.stubGlobal('WebSocket', MockWebSocket)
    await clearSession()
  })

  afterEach(async () => {
    cleanup()
    await clearSession()
  })

  it('renders snapshot-backed widgets for observability readers', async () => {
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
              id: 42,
              username: 'operator',
              roles: ['Staff'],
              permissions: [
                'requests.read',
                'deliveries.read',
                'jobs.read',
                'observability.read',
              ],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/dashboard/operations')) {
          return new Response(JSON.stringify(buildSnapshot()), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        if (url.includes('/api/v1/bootstrap')) {
          return new Response(JSON.stringify({}), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        return new Response('{}', {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }),
    )

    renderRoute('/dashboard', store)

    expect(await screen.findByRole('heading', { name: 'Dashboard', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('Requests Today', {}, { timeout: 5000 })).toBeInTheDocument()
    expect(screen.getByText('124')).toBeInTheDocument()
    expect(screen.getByText('Failed Deliveries')).toBeInTheDocument()
    expect(screen.getByText('del-9 • DHIS2 Uganda')).toBeInTheDocument()
    expect(screen.getByText('delivery-worker')).toBeInTheDocument()
    expect(screen.getByText('Delivery to DHIS2 Uganda failed')).toBeInTheDocument()

    await waitFor(() => {
      expect(vi.mocked(fetch)).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/dashboard/operations'),
        expect.any(Object),
      )
    })
  })

  it('applies live websocket updates on top of the snapshot', async () => {
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

    const fetchSpy = vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('/api/v1/auth/me')) {
        return new Response(
          JSON.stringify({
            id: 42,
            username: 'operator',
            roles: ['Staff'],
            permissions: [
              'requests.read',
              'deliveries.read',
              'jobs.read',
              'observability.read',
            ],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.includes('/api/v1/dashboard/operations')) {
        return new Response(JSON.stringify(buildSnapshot()), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      if (url.includes('/api/v1/bootstrap')) {
        return new Response(JSON.stringify({}), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      return new Response('{}', {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    })
    vi.stubGlobal('fetch', fetchSpy)

    renderRoute('/dashboard', store)

    expect(await screen.findByText('Requests Today', {}, { timeout: 5000 })).toBeInTheDocument()

    await waitFor(() => {
      expect(MockWebSocket.instances).toHaveLength(1)
    })
    expect(MockWebSocket.instances[0]?.url).toBe(
      'ws://127.0.0.1:8080/api/v1/dashboard/operations/events?access_token=access-token',
    )

    MockWebSocket.instances[0]?.open()

    expect(await screen.findByText('Live: connected', {}, { timeout: 5000 })).toBeInTheDocument()

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

    expect(await screen.findByText('Request accepted from EMR', {}, { timeout: 5000 })).toBeInTheDocument()
    expect(
      fetchSpy.mock.calls.filter(([input]) =>
        String(input).includes('/api/v1/dashboard/operations'),
      ),
    ).toHaveLength(1)

    MockWebSocket.instances[0]?.close()

    await waitFor(() => {
      expect(MockWebSocket.instances).toHaveLength(2)
    }, { timeout: 3000 })

    MockWebSocket.instances[1]?.open()

    await waitFor(() => {
      expect(
        fetchSpy.mock.calls.filter(([input]) =>
          String(input).includes('/api/v1/dashboard/operations'),
        ),
      ).toHaveLength(2)
    })
  })

  it('drills down to filtered deliveries', async () => {
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

    const fetchSpy = vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('/api/v1/auth/me')) {
        return new Response(
          JSON.stringify({
            id: 42,
            username: 'operator',
            roles: ['Staff'],
            permissions: [
              'requests.read',
              'deliveries.read',
              'jobs.read',
              'observability.read',
            ],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.includes('/api/v1/dashboard/operations')) {
        return new Response(JSON.stringify(buildSnapshot()), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      if (url.includes('/api/v1/deliveries?')) {
        return new Response(
          JSON.stringify({ items: [], totalCount: 0, page: 1, pageSize: 25 }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.includes('/api/v1/bootstrap')) {
        return new Response(JSON.stringify({}), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      return new Response('{}', {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    })
    vi.stubGlobal('fetch', fetchSpy)

    renderRoute('/dashboard', store)

    expect(await screen.findByText('Requests Today', {}, { timeout: 5000 })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Open Failed Deliveries' }))

    await waitFor(() => {
      expect(
        fetchSpy.mock.calls.some(
          ([input]) =>
            String(input).includes('/api/v1/deliveries?') &&
            String(input).includes('status=failed'),
        ),
      ).toBe(true)
    })
  })

  it('drills down to observability trace filters from recent events', async () => {
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

    const fetchSpy = vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('/api/v1/auth/me')) {
        return new Response(
          JSON.stringify({
            id: 42,
            username: 'operator',
            roles: ['Staff'],
            permissions: [
              'requests.read',
              'deliveries.read',
              'jobs.read',
              'observability.read',
            ],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.includes('/api/v1/dashboard/operations')) {
        return new Response(JSON.stringify(buildSnapshot()), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      if (url.includes('/api/v1/observability/events?')) {
        return new Response(
          JSON.stringify({ items: [], totalCount: 0, page: 1, pageSize: 25 }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.includes('/api/v1/observability/workers?')) {
        return new Response(
          JSON.stringify({ items: [], totalCount: 0, page: 1, pageSize: 25 }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.includes('/api/v1/observability/rate-limits?')) {
        return new Response(
          JSON.stringify({ items: [], totalCount: 0, page: 1, pageSize: 25 }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        )
      }
      if (url.includes('/api/v1/bootstrap')) {
        return new Response(JSON.stringify({}), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      return new Response('{}', {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    })
    vi.stubGlobal('fetch', fetchSpy)

    renderRoute('/dashboard', store)

    expect(await screen.findByText('Delivery to DHIS2 Uganda failed', {}, { timeout: 5000 })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Trace' }))

    await waitFor(() => {
      expect(
        fetchSpy.mock.calls.some(
          ([input]) =>
            String(input).includes('/api/v1/observability/events?') &&
            String(input).includes('correlationId=corr-9') &&
            String(input).includes('deliveryId=9'),
        ),
      ).toBe(true)
    })
  })

  it('does not fetch the snapshot when the user lacks observability access', async () => {
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
              id: 42,
              username: 'operator',
              roles: ['Staff'],
              permissions: ['requests.read', 'deliveries.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/bootstrap')) {
          return new Response(JSON.stringify({}), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        return new Response('{}', {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }),
    )

    renderRoute('/dashboard', store)

    expect(await screen.findByRole('heading', { name: 'Dashboard', level: 1 })).toBeInTheDocument()
    expect(screen.getByText(/Operations widgets require/i)).toBeInTheDocument()
    expect(vi.mocked(fetch)).not.toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/dashboard/operations'),
      expect.anything(),
    )
    expect(MockWebSocket.instances).toHaveLength(0)
  })
})
