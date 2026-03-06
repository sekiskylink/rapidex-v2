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
    expect(screen.getByRole('button', { name: 'Users' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Roles' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Permissions' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Audit Log' })).not.toBeInTheDocument()
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
