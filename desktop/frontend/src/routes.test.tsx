import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
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

  it('hides Audit menu item when user lacks audit.read permission', async () => {
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
    expect(screen.getByRole('button', { name: 'Users' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Audit' })).not.toBeInTheDocument()
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

  it('creates a user and refreshes the users grid', async () => {
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
                  isActive: true,
                  roles: ['Admin'],
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
        if (url.endsWith('/api/v1/users') && init?.method === 'POST') {
          createCalls += 1
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
    fireEvent.change(screen.getByLabelText('Username'), { target: { value: 'new-user' } })
    fireEvent.change(screen.getByLabelText('Temp Password'), { target: { value: 'TempPass123!' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create' }))

    await waitFor(() => {
      expect(createCalls).toBe(1)
    })
    await waitFor(() => {
      expect(getUsersCalls).toBeGreaterThanOrEqual(2)
    }, { timeout: 10_000 })
  }, 15_000)

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

    expect(await screen.findByRole('heading', { name: 'Audit', level: 1 })).toBeInTheDocument()
    expect(await screen.findByText('users.create')).toBeInTheDocument()
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
              permissions: ['users.read', 'users.write', 'audit.read'],
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
})
