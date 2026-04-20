import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearSession, configureSessionStorage, setSession } from './auth/session'
import { createAppRouter } from './routes'
import { AppThemeProvider } from './ui/theme'
import {
  defaultSettings,
  type AppSettings,
  type SaveSettingsPatch,
  type SettingsStore,
} from './settings/types'

function createMockSettingsStore(seed: AppSettings): SettingsStore & {
  loadSettingsMock: ReturnType<typeof vi.fn>
  saveSettingsMock: ReturnType<typeof vi.fn>
  resetSettingsMock: ReturnType<typeof vi.fn>
} {
  let state = {
    ...seed,
    uiPrefs: {
      ...seed.uiPrefs,
    },
  }

  const loadSettingsMock = vi.fn(async () => state)
  const saveSettingsMock = vi.fn(async (patch: SaveSettingsPatch) => {
    const nextAuthMode = patch.authMode ?? state.authMode
    state = {
      ...state,
      ...patch,
      authMode: nextAuthMode,
      apiToken:
        nextAuthMode === 'password'
          ? undefined
          : patch.apiToken !== undefined
            ? patch.apiToken || undefined
            : state.apiToken,
      refreshToken:
        patch.refreshToken !== undefined ? patch.refreshToken || undefined : state.refreshToken,
      uiPrefs: {
        ...state.uiPrefs,
        ...(patch.uiPrefs ?? {}),
      },
      tablePrefs: patch.tablePrefs ?? state.tablePrefs,
    }
    return state
  })
  const resetSettingsMock = vi.fn(async () => {
    state = { ...defaultSettings, uiPrefs: { ...defaultSettings.uiPrefs } }
    return state
  })

  return {
    loadSettings: loadSettingsMock,
    saveSettings: saveSettingsMock,
    resetSettings: resetSettingsMock,
    loadSettingsMock,
    saveSettingsMock,
    resetSettingsMock,
  }
}

function renderWithRouter(initialPath: string, store: SettingsStore) {
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

describe('app shell routes', () => {
  beforeEach(async () => {
    vi.restoreAllMocks()
    await clearSession()
  })

  afterEach(async () => {
    cleanup()
    await clearSession()
  })

  it('shows Administration group only with allowed children', async () => {
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
              roles: ['Manager'],
              permissions: ['users.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/dashboard', store)

    expect(await screen.findByRole('heading', { name: 'Dashboard', level: 1 })).toBeInTheDocument()
    expect(screen.getAllByText('Administration').length).toBeGreaterThan(0)
    fireEvent.click(screen.getByRole('button', { name: 'Toggle Administration menu' }))
    expect(screen.getAllByRole('button', { name: 'Users' }).length).toBeGreaterThan(0)
    expect(screen.getAllByRole('button', { name: 'Roles' }).length).toBeGreaterThan(0)
    expect(screen.getAllByRole('button', { name: 'Permissions' }).length).toBeGreaterThan(0)
    expect(screen.queryByRole('button', { name: 'Audit Log' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Settings' })).not.toBeInTheDocument()
  })

  it('shows Forbidden when navigating to /audit without audit.read permission', async () => {
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
              roles: ['Manager'],
              permissions: ['users.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/audit', store)

    expect(await screen.findByRole('heading', { name: '403', level: 1 })).toBeInTheDocument()
    expect(screen.getByText('Forbidden')).toBeInTheDocument()
  })

  it('applies local navigation visibility and label preferences', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
      refreshToken: 'refresh-token',
      uiPrefs: {
        ...defaultSettings.uiPrefs,
        showSukumadMenu: false,
        showAdministrationMenu: false,
        navLabels: {
          dashboard: 'Home',
        },
      },
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
              roles: ['Admin'],
              permissions: ['users.read', 'settings.write', 'requests.read'],
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

    renderWithRouter('/dashboard', store)

    expect(await screen.findByRole('heading', { name: 'Dashboard', level: 1 })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Home' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Toggle Sukumad menu' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Toggle Administration menu' })).not.toBeInTheDocument()
  })

  it('renders Sukumad servers route and navigation when permission is granted', async () => {
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
              id: 15,
              username: 'operator',
              roles: ['Staff'],
              permissions: ['servers.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/servers?')) {
          return new Response(
            JSON.stringify({
              items: [],
              totalCount: 0,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/servers', store)

    expect(await screen.findByRole('heading', { name: 'Servers', level: 1 })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Toggle Sukumad menu' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Toggle Sukumad menu' }))
    expect(screen.getByRole('button', { name: 'Servers' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Requests' })).not.toBeInTheDocument()
  })

  it('renders Sukumad requests route and navigation when permission is granted', async () => {
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
              id: 16,
              username: 'request-operator',
              roles: ['Staff'],
              permissions: ['requests.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/requests?')) {
          return new Response(
            JSON.stringify({
              items: [],
              totalCount: 0,
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

    renderWithRouter('/requests', store)

    expect(await screen.findByRole('heading', { name: 'Requests', level: 1 })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Toggle Sukumad menu' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Toggle Sukumad menu' }))
    expect(screen.getByRole('button', { name: 'Requests' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Servers' })).not.toBeInTheDocument()
  })

  it('renders Sukumad deliveries route and navigation when permission is granted', async () => {
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
              id: 14,
              username: 'delivery-operator',
              roles: ['Staff'],
              permissions: ['deliveries.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/deliveries?')) {
          return new Response(
            JSON.stringify({
              items: [],
              totalCount: 0,
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

    renderWithRouter('/deliveries', store)

    expect(await screen.findByRole('heading', { name: 'Deliveries', level: 1 })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Toggle Sukumad menu' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Toggle Sukumad menu' }))
    expect(screen.getByRole('button', { name: 'Deliveries' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Servers' })).not.toBeInTheDocument()
  })

  it('renders Sukumad jobs route and navigation when permission is granted', async () => {
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
              id: 17,
              username: 'jobs-operator',
              roles: ['Staff'],
              permissions: ['jobs.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/jobs?')) {
          return new Response(
            JSON.stringify({
              items: [],
              totalCount: 0,
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

    renderWithRouter('/jobs', store)

    expect(await screen.findByRole('heading', { name: 'Jobs', level: 1 })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Toggle Sukumad menu' }))
    expect(screen.getByRole('button', { name: 'Jobs' })).toBeInTheDocument()
  })

  it('renders Sukumad scheduler route and navigation when permission is granted', async () => {
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
              id: 171,
              username: 'scheduler-operator',
              roles: ['Staff'],
              permissions: ['scheduler.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/scheduler/jobs?')) {
          return new Response(
            JSON.stringify({
              items: [],
              totalCount: 0,
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

    renderWithRouter('/scheduler', store)

    expect(await screen.findByRole('heading', { name: 'Scheduler', level: 1 })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Toggle Sukumad menu' }))
    expect(screen.getByRole('button', { name: 'Scheduler' })).toBeInTheDocument()
  })

  it('renders Sukumad observability route and navigation when permission is granted', async () => {
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
              id: 18,
              username: 'ops-reader',
              roles: ['Staff'],
              permissions: ['observability.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/observability/workers?')) {
          return new Response(
            JSON.stringify({
              items: [],
              totalCount: 0,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/observability/rate-limits?')) {
          return new Response(
            JSON.stringify({
              items: [],
              totalCount: 0,
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

    renderWithRouter('/observability', store)

    expect(await screen.findByRole('heading', { name: 'Observability', level: 1 })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Toggle Sukumad menu' }))
    expect(screen.getByRole('button', { name: 'Observability' })).toBeInTheDocument()
  })

  it('renders documentation route and navigation for authenticated users', async () => {
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
              id: 19,
              username: 'docs-reader',
              roles: ['Staff'],
              permissions: [],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/bootstrap')) {
          return new Response(JSON.stringify({}), { status: 200, headers: { 'Content-Type': 'application/json' } })
        }
        if (url.endsWith('/api/v1/documentation')) {
          return new Response(
            JSON.stringify({
              items: [
                { slug: 'overview', title: 'Overview', sourcePath: 'overview.md' },
                { slug: 'operations', title: 'Operations', sourcePath: 'operations.md' },
              ],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/documentation/overview')) {
          return new Response(
            JSON.stringify({
              slug: 'overview',
              title: 'Overview',
              sourcePath: 'overview.md',
              content: '# Overview\n\nDocumentation body',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/documentation/operations')) {
          return new Response(
            JSON.stringify({
              slug: 'operations',
              title: 'Operations',
              sourcePath: 'operations.md',
              content: '# Operations\n\nOperational handoff',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/documentation', store)

    expect(await screen.findByRole('heading', { name: 'Documentation', level: 1 })).toBeInTheDocument()
    expect(await screen.findByRole('heading', { name: 'Overview', level: 1 })).toBeInTheDocument()
    expect(screen.getByText('Documentation body')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /operations/i }))
    expect(await screen.findByRole('heading', { name: 'Operations', level: 1 })).toBeInTheDocument()
    expect(screen.getByText('Operational handoff')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Toggle Sukumad menu' }))
    expect(screen.getByRole('button', { name: 'Documentation' })).toBeInTheDocument()
  })

  it('hides Administration group when no admin route permission is granted', async () => {
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
              id: 8,
              username: 'viewer',
              roles: ['Viewer'],
              permissions: ['settings.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/dashboard', store)

    expect(await screen.findByRole('heading', { name: 'Dashboard', level: 1 })).toBeInTheDocument()
    expect(screen.queryByText('Administration')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Users' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Roles' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Permissions' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Audit Log' })).not.toBeInTheDocument()
  })

  it('shows Settings inside Administration when settings.write is granted', async () => {
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
              id: 11,
              username: 'settings-writer',
              roles: ['Staff'],
              permissions: ['settings.write'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/settings', store)

    expect(await screen.findByRole('heading', { name: 'Settings', level: 1 })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Settings' })).toBeInTheDocument()
    expect(screen.getAllByText('Administration').length).toBeGreaterThan(0)
    expect(screen.queryByRole('button', { name: 'Users' })).not.toBeInTheDocument()
  })

  it('hides administration navigation and shows module-disabled state when administration module is disabled', async () => {
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
              roles: ['Admin'],
              permissions: ['users.read', 'users.write', 'audit.read', 'settings.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/bootstrap')) {
          return new Response(
            JSON.stringify({
              version: 1,
              generatedAt: '2026-03-08T00:00:00Z',
              app: { version: '1.0.0', commit: 'abc123', buildDate: '2026-03-08T00:00:00Z' },
              branding: { applicationDisplayName: 'BasePro Desktop', loginImageUrl: null },
              modules: [
                { moduleId: 'dashboard', flagKey: 'modules.dashboard.enabled', enabled: true, enabledByDefault: true, source: 'default' },
                { moduleId: 'administration', flagKey: 'modules.administration.enabled', enabled: false, enabledByDefault: true, source: 'config' },
                { moduleId: 'settings', flagKey: 'modules.settings.enabled', enabled: true, enabledByDefault: true, source: 'default' },
              ],
              capabilities: { canAccessSettings: true, settings: { canRead: true, canWrite: true } },
              cache: { maxStaleSeconds: 300, schemaVersion: 1, cacheable: true, offlineSafePayload: true, containsSecrets: false },
              principal: { type: 'user', userId: 5, username: 'alice', roles: ['Admin'], permissions: ['users.read', 'users.write', 'audit.read', 'settings.read'] },
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/users', store)

    expect(await screen.findByRole('heading', { name: 'Module Disabled', level: 1 })).toBeInTheDocument()
    expect(screen.getByText('Administration is unavailable')).toBeInTheDocument()
    expect(screen.queryByText('Administration')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Users' })).not.toBeInTheDocument()
  })

  it('denies /settings for non-admin users without settings.write', async () => {
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
              id: 9,
              username: 'viewer',
              roles: ['Viewer'],
              permissions: ['settings.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/settings', store)

    expect(await screen.findByRole('heading', { name: '403', level: 1 })).toBeInTheDocument()
    expect(screen.getByText('Forbidden')).toBeInTheDocument()
  })

  it('renders users list metadata columns and values', async () => {
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

    let getUsersCalls = 0
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/api/v1/auth/me')) {
          return new Response(
            JSON.stringify({
              id: 1,
              username: 'admin',
              roles: ['Admin'],
              permissions: ['users.read', 'users.write'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/users') && (init?.method === undefined || init.method === 'GET')) {
          getUsersCalls += 1
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 7,
                  username: 'jane',
                  displayName: 'Jane Doe',
                  firstName: 'Jane',
                  lastName: 'Doe',
                  email: 'jane@example.com',
                  language: 'English',
                  phoneNumber: '+15551234567',
                  isActive: true,
                  roles: ['Admin'],
                  updatedAt: '2026-03-02T12:00:00Z',
                  createdAt: '2026-02-28T00:00:00Z',
                },
              ],
              totalCount: 1,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/users', store)

    expect(await screen.findByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument()
    await waitFor(() => {
      expect(getUsersCalls).toBeGreaterThanOrEqual(1)
    })
    expect(await screen.findByText('jane@example.com')).toBeInTheDocument()
    expect(screen.getByText('+15551234567')).toBeInTheDocument()
    expect(screen.getByRole('switch')).toBeChecked()
  })

  it('creates a user with metadata payload and refreshes the users grid', async () => {
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

    let getUsersCalls = 0
    let createCalls = 0
    let createPayload: Record<string, unknown> | null = null

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/api/v1/auth/me')) {
          return new Response(
            JSON.stringify({
              id: 1,
              username: 'admin',
              roles: ['Admin'],
              permissions: ['users.read', 'users.write'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/users') && (init?.method === undefined || init.method === 'GET')) {
          getUsersCalls += 1
          return new Response(
            JSON.stringify({
              items: [],
              totalCount: 0,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/users') && init?.method === 'POST') {
          createCalls += 1
          createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
          return new Response(
            JSON.stringify({ id: 8, username: 'new-user', isActive: true, roles: ['Viewer'] }),
            { status: 201, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/users', store)

    expect(await screen.findByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument()
    await waitFor(() => {
      expect(getUsersCalls).toBeGreaterThanOrEqual(1)
    })

    fireEvent.click(screen.getByRole('button', { name: 'Create User' }))
    const createDialog = await screen.findByRole('dialog', { name: 'Create User' })
    expect(within(createDialog).getByTestId('desktop-user-create-form-grid')).toBeInTheDocument()
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Username' }), { target: { value: 'new-user' } })
    const createPasswordInput = createDialog.querySelector('input[type=\"password\"]')
    expect(createPasswordInput).not.toBeNull()
    fireEvent.change(createPasswordInput as Element, { target: { value: 'TempPass123!' } })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Email' }), { target: { value: 'new-user@example.com' } })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Language' }), { target: { value: 'French' } })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'First Name' }), { target: { value: 'New' } })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Last Name' }), { target: { value: 'User' } })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Display Name' }), { target: { value: 'New User' } })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Phone Number' }), { target: { value: '+15550000001' } })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'WhatsApp Number' }), { target: { value: '+15550000002' } })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Telegram Handle' }), { target: { value: '@new_user' } })
    fireEvent.click(within(createDialog).getByRole('button', { name: 'Create' }))

    await waitFor(() => {
      expect(createCalls).toBe(1)
    })
    expect(await screen.findByText('User created.')).toBeInTheDocument()
    expect(createPayload).toMatchObject({
      username: 'new-user',
      password: 'TempPass123!',
      email: 'new-user@example.com',
      language: 'French',
      firstName: 'New',
      lastName: 'User',
      displayName: 'New User',
      phoneNumber: '+15550000001',
      whatsappNumber: '+15550000002',
      telegramHandle: '@new_user',
      isActive: true,
    })
    await waitFor(() => {
      expect(getUsersCalls).toBeGreaterThanOrEqual(2)
    }, { timeout: 10_000 })
  }, 15_000)

  it('shows inline validation errors when create user fails validation', async () => {
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
              id: 1,
              username: 'admin',
              roles: ['Admin'],
              permissions: ['users.read', 'users.write'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/admin/roles')) {
          return new Response(
            JSON.stringify({
              items: [{ id: 1, name: 'Admin' }],
              totalCount: 1,
              page: 1,
              pageSize: 200,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/users') && (init?.method === undefined || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              items: [],
              totalCount: 0,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/users') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({
              error: {
                code: 'VALIDATION_ERROR',
                message: 'validation failed',
                details: {
                  email: ['must be a valid email address'],
                  phoneNumber: ['must be E.164 format, e.g. +15551234567'],
                },
              },
            }),
            {
              status: 422,
              headers: {
                'Content-Type': 'application/json',
                'X-Request-Id': 'req-desktop-users-422',
              },
            },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/users', store)

    expect(await screen.findByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Create User' }))
    const createDialog = await screen.findByRole('dialog', { name: 'Create User' })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Username' }), { target: { value: 'bad-user' } })
    const createPasswordInput = createDialog.querySelector('input[type=\"password\"]')
    expect(createPasswordInput).not.toBeNull()
    fireEvent.change(createPasswordInput as Element, { target: { value: 'TempPass123!' } })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Email' }), { target: { value: 'not-an-email' } })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Phone Number' }), { target: { value: '123' } })
    fireEvent.click(within(createDialog).getByRole('button', { name: 'Create' }))

    expect(await within(createDialog).findByText('must be a valid email address')).toBeInTheDocument()
    expect(within(createDialog).getByText('must be E.164 format, e.g. +15551234567')).toBeInTheDocument()
  })

  it('edits user metadata and allows optional password on edit', async () => {
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

    let patchCalls = 0
    let patchPayload: Record<string, unknown> | null = null

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/api/v1/auth/me')) {
          return new Response(
            JSON.stringify({
              id: 1,
              username: 'admin',
              roles: ['Admin'],
              permissions: ['users.read', 'users.write'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/users') && (init?.method === undefined || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 7,
                  username: 'jane',
                  displayName: 'Jane Doe',
                  firstName: 'Jane',
                  lastName: 'Doe',
                  email: 'jane@example.com',
                  language: 'English',
                  phoneNumber: '+15551234567',
                  whatsappNumber: '+15551234568',
                  telegramHandle: '@janedoe',
                  isActive: true,
                  roles: ['Admin'],
                  updatedAt: '2026-03-02T12:00:00Z',
                  createdAt: '2026-02-28T00:00:00Z',
                },
              ],
              totalCount: 1,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/users/7') && init?.method === 'PATCH') {
          patchCalls += 1
          patchPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
          return new Response(
            JSON.stringify({ id: 7, username: 'jane', isActive: true }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/users', store)

    expect(await screen.findByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument()
    fireEvent.click(await screen.findByRole('button', { name: 'Actions for jane' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Edit' }))

    const editDialog = await screen.findByRole('dialog', { name: 'Edit User' })
    expect(within(editDialog).getByTestId('desktop-user-edit-form-grid')).toBeInTheDocument()
    expect(within(editDialog).getByDisplayValue('jane')).toBeDisabled()
    fireEvent.change(within(editDialog).getByRole('textbox', { name: 'Email' }), { target: { value: 'jane-updated@example.com' } })
    fireEvent.change(within(editDialog).getByRole('textbox', { name: 'Phone Number' }), { target: { value: '+15559876543' } })
    fireEvent.change(within(editDialog).getByRole('textbox', { name: 'Display Name' }), { target: { value: 'Jane Updated' } })
    fireEvent.click(within(editDialog).getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(patchCalls).toBe(1)
    })
    expect(patchPayload).toMatchObject({
      username: 'jane',
      email: 'jane-updated@example.com',
      language: 'English',
      firstName: 'Jane',
      lastName: 'Doe',
      displayName: 'Jane Updated',
      phoneNumber: '+15559876543',
      whatsappNumber: '+15551234568',
      telegramHandle: '@janedoe',
      isActive: true,
    })
    expect(patchPayload).not.toHaveProperty('password')
  })

  it('loads audit grid rows on /audit', async () => {
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
              id: 1,
              username: 'admin',
              roles: ['Admin'],
              permissions: ['audit.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/audit')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 12,
                  timestamp: '2026-03-01T01:00:00Z',
                  actorUserId: 1,
                  action: 'users.create',
                  entityType: 'user',
                  entityId: '12',
                  metadata: { username: 'new-user' },
                },
              ],
              totalCount: 1,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/audit', store)

    expect(await screen.findByRole('heading', { name: 'Audit Log', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('users.create')).toBeInTheDocument()
    fireEvent.click(await screen.findByRole('button', { name: 'Actions for users.create' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View Metadata' }))
    expect(await screen.findByRole('dialog', { name: 'Audit Metadata' })).toBeInTheDocument()
    expect(screen.getByText(/\"username\": \"new-user\"/)).toBeInTheDocument()
  })

  it('renders backend version in Settings About section', async () => {
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
              id: 1,
              username: 'admin',
              roles: ['Admin'],
              permissions: ['users.read', 'users.write', 'audit.read', 'settings.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/version')) {
          return new Response(
            JSON.stringify({
              version: '1.2.3',
              commit: 'abc1234',
              buildDate: '2026-03-03T00:00:00Z',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/settings', store)

    expect(await screen.findByRole('heading', { name: 'Settings', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText(/Backend version:\s*1\.2\.3/i)).toBeInTheDocument()
  })

  it('supports users role multi-select in create flow', async () => {
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
              id: 1,
              username: 'admin',
              roles: ['Admin'],
              permissions: ['users.read', 'users.write'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/admin/roles')) {
          return new Response(
            JSON.stringify({
              items: [
                { id: 1, name: 'Admin' },
                { id: 2, name: 'Viewer' },
              ],
              totalCount: 2,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/users') && (init?.method === undefined || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              items: [],
              totalCount: 0,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/users') && init?.method === 'POST') {
          createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
          return new Response(JSON.stringify({ id: 10, username: 'ops-user' }), {
            status: 201,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/users', store)

    expect(await screen.findByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Create User' }))
    const createDialog = await screen.findByRole('dialog', { name: 'Create User' })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Username' }), { target: { value: 'ops-user' } })
    const passwordInput = createDialog.querySelector('input[type=\"password\"]')
    expect(passwordInput).not.toBeNull()
    fireEvent.change(passwordInput as Element, { target: { value: 'TempPass123!' } })
    const rolesInput = within(createDialog).getByRole('combobox', { name: 'Roles' })
    fireEvent.mouseDown(rolesInput)
    fireEvent.change(rolesInput, { target: { value: 'Admin' } })
    fireEvent.click(await screen.findByText('Admin'))
    fireEvent.click(within(createDialog).getByRole('button', { name: 'Create' }))

    await waitFor(() => {
      expect(createPayload).not.toBeNull()
    })
    expect(createPayload).toMatchObject({
      username: 'ops-user',
      roles: ['Admin'],
    })
  })

  it('supports roles list/create/edit/detail flows', async () => {
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

    const requestedUrls: string[] = []
    let createPayload: Record<string, unknown> | null = null
    let patchPayload: Record<string, unknown> | null = null
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString()
        requestedUrls.push(url)
        if (url.includes('/api/v1/auth/me')) {
          return new Response(
            JSON.stringify({
              id: 1,
              username: 'admin',
              roles: ['Admin'],
              permissions: ['users.read', 'users.write'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/admin/permissions')) {
          return new Response(
            JSON.stringify({
              items: [
                { id: 1, name: 'users.read', moduleScope: 'admin', createdAt: '2026-03-01T00:00:00Z' },
                { id: 2, name: 'users.write', moduleScope: 'admin', createdAt: '2026-03-01T00:00:00Z' },
              ],
              totalCount: 2,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/admin/roles/2?includeUsers=false')) {
          return new Response(
            JSON.stringify({
              id: 2,
              name: 'Manager',
              permissions: [{ id: 1, name: 'users.read', moduleScope: 'admin', createdAt: '2026-03-01T00:00:00Z' }],
              createdAt: '2026-03-01T00:00:00Z',
              updatedAt: '2026-03-01T00:00:00Z',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/admin/roles/2?includeUsers=true')) {
          return new Response(
            JSON.stringify({
              id: 2,
              name: 'Manager',
              permissions: [{ id: 1, name: 'users.read', moduleScope: 'admin', createdAt: '2026-03-01T00:00:00Z' }],
              users: [{ id: 10, username: 'jane', isActive: true }],
              createdAt: '2026-03-01T00:00:00Z',
              updatedAt: '2026-03-01T00:00:00Z',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/admin/roles') && (init?.method === undefined || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 2,
                  name: 'Manager',
                  permissionCount: 1,
                  userCount: 1,
                  createdAt: '2026-03-01T00:00:00Z',
                  updatedAt: '2026-03-02T00:00:00Z',
                },
              ],
              totalCount: 1,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/admin/roles') && init?.method === 'POST') {
          createPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
          return new Response(JSON.stringify({ id: 3, name: 'Ops' }), {
            status: 201,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        if (url.endsWith('/api/v1/admin/roles/2') && init?.method === 'PATCH') {
          patchPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
          return new Response(JSON.stringify({ id: 2, name: 'Manager Updated' }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/roles', store)

    expect(await screen.findByRole('heading', { name: 'Roles', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('Manager')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Create Role' }))
    const createDialog = await screen.findByRole('dialog', { name: 'Create Role' })
    fireEvent.change(within(createDialog).getByRole('textbox', { name: 'Role Name' }), { target: { value: 'Ops' } })
    fireEvent.click(within(createDialog).getByRole('button', { name: 'Create' }))
    await waitFor(() => {
      expect(createPayload).not.toBeNull()
    })
    expect(createPayload).toMatchObject({ name: 'Ops' })

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for Manager' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Edit Role' }))
    const editDialog = await screen.findByRole('dialog', { name: 'Edit Role' })
    fireEvent.change(within(editDialog).getByRole('textbox', { name: 'Role Name' }), {
      target: { value: 'Manager Updated' },
    })
    fireEvent.click(within(editDialog).getByRole('button', { name: 'Save' }))
    await waitFor(() => {
      expect(patchPayload).not.toBeNull()
    })
    expect(patchPayload).toMatchObject({ name: 'Manager Updated' })

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for Manager' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View Details' }))
    expect(await screen.findByRole('dialog', { name: 'Role Details' })).toBeInTheDocument()
    expect(screen.getByText('jane')).toBeInTheDocument()
    expect(requestedUrls.some((url) => url.includes('/api/v1/admin/roles/2?includeUsers=true'))).toBe(true)
  })

  it('supports permissions list/filter/details flows', async () => {
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

    const permissionUrls: string[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/api/v1/auth/me')) {
          return new Response(
            JSON.stringify({
              id: 1,
              username: 'admin',
              roles: ['Admin'],
              permissions: ['users.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/admin/permissions')) {
          permissionUrls.push(url)
          return new Response(
            JSON.stringify({
              items: [{ id: 1, name: 'users.read', moduleScope: 'admin', createdAt: '2026-03-01T00:00:00Z' }],
              totalCount: 1,
              page: 1,
              pageSize: 25,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/permissions', store)

    expect(await screen.findByRole('heading', { name: 'Permissions', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('users.read')).toBeInTheDocument()

    fireEvent.change(screen.getByPlaceholderText('e.g. users.read'), { target: { value: 'users' } })
    fireEvent.change(screen.getByPlaceholderText('e.g. admin'), { target: { value: 'admin' } })

    await waitFor(() => {
      expect(permissionUrls.some((url) => url.includes('q=users'))).toBe(true)
      expect(permissionUrls.some((url) => url.includes('moduleScope=admin'))).toBe(true)
    })

    fireEvent.click(await screen.findByRole('button', { name: 'Actions for users.read' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'View Details' }))
    expect(await screen.findByRole('dialog', { name: 'Permission Details' })).toBeInTheDocument()
  })

  it('renders login branding fallback with backend display name', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
    })

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/api/v1/settings/public/login-branding')) {
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'Acme Platform',
              loginImageUrl: '',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/login', store)

    expect(await screen.findByRole('heading', { name: 'Acme Platform', level: 1 })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Sign In' })).toBeInTheDocument()
  })

  it('updates Scheduler navigation label from settings page', async () => {
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
              roles: ['Admin'],
              permissions: ['settings.read', 'settings.write'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.includes('/api/v1/settings/login-branding')) {
          return new Response(JSON.stringify({ applicationDisplayName: 'BasePro', loginImageUrl: '' }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        if (url.includes('/api/v1/settings/module-enablement')) {
          return new Response(JSON.stringify({ modules: [] }), { status: 200, headers: { 'Content-Type': 'application/json' } })
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/settings', store)

    expect(await screen.findByRole('heading', { name: 'Settings', level: 1 })).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Scheduler link'), { target: { value: 'Timed Calls' } })

    await waitFor(() => {
      expect(store.saveSettingsMock).toHaveBeenCalledWith(
        expect.objectContaining({
          uiPrefs: expect.objectContaining({
            navLabels: expect.objectContaining({ scheduler: 'Timed Calls' }),
          }),
        }),
      )
    })
  })

  it('submits forgot-password request and shows non-enumerating success message', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
    })

    let forgotCallCount = 0
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/api/v1/settings/public/login-branding')) {
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'BasePro',
              loginImageUrl: '',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/auth/forgot-password') && init?.method === 'POST') {
          forgotCallCount += 1
          return new Response(JSON.stringify({ status: 'accepted' }), {
            status: 202,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/forgot-password', store)

    fireEvent.change(await screen.findByRole('textbox', { name: 'Username or Email' }), {
      target: { value: 'alice@example.com' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Send Reset Instructions' }))

    await waitFor(() => {
      expect(forgotCallCount).toBe(1)
    })
    expect(await screen.findByText('If the account exists, password reset instructions have been sent.')).toBeInTheDocument()
  })

  it('validates reset-password confirmation before submitting', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
    })

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/api/v1/settings/public/login-branding')) {
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'BasePro',
              loginImageUrl: '',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/reset-password?token=abc123', store)

    await screen.findByRole('button', { name: 'Reset Password' })
    fireEvent.change(screen.getByLabelText(/Reset Token/i), { target: { value: 'abc123' } })
    fireEvent.change(screen.getByLabelText(/^New Password/i), { target: { value: 'PasswordOne!' } })
    fireEvent.change(screen.getByLabelText(/Confirm New Password/i), { target: { value: 'PasswordTwo!' } })
    fireEvent.click(screen.getByRole('button', { name: 'Reset Password' }))

    expect(await screen.findByText('Passwords do not match.')).toBeInTheDocument()
  })

  it('updates login branding from settings page', async () => {
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

    let brandingUpdatePayload: Record<string, unknown> | null = null
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/api/v1/auth/me')) {
          return new Response(
            JSON.stringify({
              id: 1,
              username: 'admin',
              roles: ['Admin'],
              permissions: ['settings.read', 'settings.write'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/version')) {
          return new Response(
            JSON.stringify({ version: '1.0.0', commit: 'abc123', buildDate: '2026-03-06T00:00:00Z' }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/settings/login-branding') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'BasePro',
              loginImageUrl: 'https://cdn.example.com/old.png',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/settings/login-branding') && init?.method === 'PUT') {
          brandingUpdatePayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'Platform Pro',
              loginImageUrl: 'https://cdn.example.com/new.png',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/settings/module-enablement') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              modules: [],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/settings', store)

    expect(await screen.findByRole('heading', { name: 'Settings', level: 1 })).toBeInTheDocument()
    fireEvent.change(await screen.findByLabelText('Application Display Name'), { target: { value: 'Platform Pro' } })
    fireEvent.change(screen.getByLabelText('Login Image URL'), { target: { value: 'https://cdn.example.com/new.png' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save Branding' }))

    await waitFor(() => {
      expect(brandingUpdatePayload).toEqual({
        applicationDisplayName: 'Platform Pro',
        loginImageUrl: 'https://cdn.example.com/new.png',
      })
    })
  })

  it('creates and reveals an API token from settings page', async () => {
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
              id: 1,
              username: 'admin',
              roles: ['Admin'],
              permissions: ['settings.read', 'settings.write', 'api_tokens.write'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/admin/api-tokens') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({
              id: 12,
              name: 'Desktop API token',
              prefix: 'bpt_xyz',
              token: 'plaintext-api-token',
              permissions: ['settings.read'],
            }),
            { status: 201, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/version')) {
          return new Response(
            JSON.stringify({ version: '1.0.0', commit: 'abc123', buildDate: '2026-03-06T00:00:00Z' }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/settings/login-branding') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'BasePro',
              loginImageUrl: 'https://cdn.example.com/old.png',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/settings/module-enablement') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              modules: [],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/settings/runtime-config') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              config: {},
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/settings', store)

    await screen.findByRole('heading', { name: 'Settings', level: 1 })
    fireEvent.change(await screen.findByLabelText('Token Name'), { target: { value: 'Desktop API token' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create Token' }))

    expect(await screen.findByText('plaintext-api-token')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Copy Token' })).toBeInTheDocument()
  })

  it('shows module enablement list and write-permission guidance on settings page', async () => {
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
              id: 1,
              username: 'admin',
              roles: ['Admin'],
              permissions: ['settings.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/version')) {
          return new Response(
            JSON.stringify({ version: '1.0.0', commit: 'abc123', buildDate: '2026-03-06T00:00:00Z' }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/settings/module-enablement') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              modules: [
                {
                  moduleId: 'settings',
                  flagKey: 'modules.settings.enabled',
                  enabled: true,
                  enabledByDefault: true,
                  description: 'Settings module',
                  source: 'default',
                  adminControl: 'static',
                  editable: false,
                },
                {
                  moduleId: 'administration',
                  flagKey: 'modules.administration.enabled',
                  enabled: true,
                  enabledByDefault: true,
                  description: 'Admin module',
                  source: 'default',
                  adminControl: 'runtime',
                  editable: true,
                },
              ],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/settings/login-branding') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'BasePro',
              loginImageUrl: '',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/settings', store)
    expect(await screen.findByTestId('module-flag-administration')).toHaveTextContent('Administration')
    expect(await screen.findByText('Admin module')).toBeInTheDocument()
    expect(screen.queryByText('Settings module')).not.toBeInTheDocument()
    expect(
      screen.getByText('You need settings.write permission to change runtime-manageable module flags.'),
    ).toBeInTheDocument()
  })

  it('renders sanitized runtime config on the settings page', async () => {
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
              id: 1,
              username: 'admin',
              roles: ['Admin'],
              permissions: ['settings.read', 'settings.write'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/version')) {
          return new Response(
            JSON.stringify({ version: '1.0.0', commit: 'abc123', buildDate: '2026-03-06T00:00:00Z' }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/settings/module-enablement') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              modules: [],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/settings/login-branding') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'BasePro',
              loginImageUrl: '',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/api/v1/settings/runtime-config') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              config: {
                database: {
                  dsn: 'postgres://dbuser:[masked]@db.example.com:5432/basepro?sslmode=disable',
                },
                auth: {
                  jwt_signing_key: '[masked]',
                },
              },
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

    renderWithRouter('/settings', store)

    expect(await screen.findByRole('heading', { name: 'Settings', level: 1 })).toBeInTheDocument()
    expect(await screen.findByRole('heading', { name: 'Runtime Config', level: 6 })).toBeInTheDocument()
    expect(await screen.findByDisplayValue(/dbuser:\[masked\]@db\.example\.com/)).toBeInTheDocument()
    expect(screen.getByDisplayValue(/"jwt_signing_key": "\[masked\]"/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'YAML' }))
    expect(await screen.findByDisplayValue(/database:/)).toBeInTheDocument()
    expect(screen.getByDisplayValue(/dsn: "postgres:\/\/dbuser:\[masked\]@db\.example\.com:5432\/basepro\?sslmode=disable"/)).toBeInTheDocument()
    expect(screen.getByDisplayValue(/jwt_signing_key: "\[masked\]"/)).toBeInTheDocument()
  })
})
