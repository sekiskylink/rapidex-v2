import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearAuthSnapshot, persistRefreshToken, setAuthSnapshot } from './auth/state'
import { API_BASE_URL_OVERRIDE_STORAGE_KEY } from './lib/apiBaseUrl'
import { apiRequest } from './lib/api'
import { applyEffectiveModuleEnablement, resetEffectiveModuleEnablement } from './registry/moduleEnablement'
import { createAppRouter } from './routes'
import { SnackbarProvider } from './ui/snackbar'
import { UI_PREFERENCES_STORAGE_KEY } from './ui/preferences'

function mockViewport(isMobile: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => {
      const isColorSchemeQuery = query === '(prefers-color-scheme: dark)'
      const isMobileWidthQuery = query.includes('max-width:599.95px')
      return {
        matches: isColorSchemeQuery ? false : isMobileWidthQuery ? isMobile : false,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      }
    }),
  })
}

function renderWithRouter(initialPath: string) {
  const router = createAppRouter([initialPath])
  const queryClient = new QueryClient()

  return render(
    <QueryClientProvider client={queryClient}>
      <SnackbarProvider>
        <RouterProvider router={router} />
      </SnackbarProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  window.localStorage.clear()
  clearAuthSnapshot()
  resetEffectiveModuleEnablement()
  mockViewport(false)
  vi.stubEnv('VITE_API_BASE_URL', 'http://localhost:8080/api/v1')
  window.localStorage.setItem(API_BASE_URL_OVERRIDE_STORAGE_KEY, 'http://localhost:8080/api/v1')
  vi.stubGlobal(
    'fetch',
    vi.fn(async () => new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })),
  )
})

afterEach(() => {
  cleanup()
  window.localStorage.clear()
  clearAuthSnapshot()
  resetEffectiveModuleEnablement()
  vi.unstubAllEnvs()
  vi.unstubAllGlobals()
})

describe('web auth routes', () => {
  it('login success redirects to /dashboard', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/settings/public/login-branding')) {
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'BasePro Web',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/auth/login') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({
              accessToken: 'access-token',
              refreshToken: 'refresh-token',
              expiresIn: 300,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/auth/me')) {
          return new Response(
            JSON.stringify({
              id: 1,
              username: 'admin',
              roles: ['Admin'],
              permissions: ['users.read', 'settings.read'],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/login')

    fireEvent.change(await screen.findByRole('textbox', { name: /username/i }), { target: { value: 'admin' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: 'Sign In' }))

    expect(await screen.findByRole('heading', { name: 'Dashboard', level: 1 })).toBeInTheDocument()
  })

  it('login failure shows inline invalid-credentials message', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/settings/public/login-branding')) {
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'BasePro Web',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/auth/login') && init?.method === 'POST') {
          return new Response(
            JSON.stringify({
              error: {
                code: 'AUTH_UNAUTHORIZED',
                message: 'Invalid credentials',
              },
            }),
            {
              status: 401,
              headers: {
                'Content-Type': 'application/json',
                'X-Request-Id': 'req-401',
              },
            },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/login')

    fireEvent.change(await screen.findByRole('textbox', { name: /username/i }), { target: { value: 'bad-user' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'bad-pass' } })
    fireEvent.click(screen.getByRole('button', { name: 'Sign In' }))

    expect(await screen.findByText('Invalid username or password.')).toBeInTheDocument()
  })

  it('refresh failure logs out and redirects to /login', async () => {
    setAuthSnapshot({
      isAuthenticated: true,
      accessToken: 'stale-access-token',
      refreshToken: 'refresh-token',
      user: {
        id: 1,
        username: 'admin',
        roles: ['Admin'],
        permissions: ['settings.read'],
      },
    })
    persistRefreshToken('refresh-token')

    vi.mocked(fetch)
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            error: {
              code: 'AUTH_EXPIRED',
              message: 'Access token expired',
            },
          }),
          { status: 401, headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            error: {
              code: 'AUTH_REFRESH_INVALID',
              message: 'Refresh invalid',
            },
          }),
          { status: 401, headers: { 'Content-Type': 'application/json' } },
        ),
      )

    renderWithRouter('/dashboard')
    expect(await screen.findByRole('heading', { name: 'Dashboard', level: 1 })).toBeInTheDocument()

    await expect(apiRequest('/users')).rejects.toMatchObject({
      code: 'AUTH_EXPIRED',
    })

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'BasePro Web', level: 1 })).toBeInTheDocument()
    })
    expect(screen.getByText('Session expired. Please log in again.')).toBeInTheDocument()
  })

  it('protected route is blocked when logged out', async () => {
    renderWithRouter('/dashboard')
    expect(await screen.findByRole('heading', { name: 'BasePro Web', level: 1 })).toBeInTheDocument()
  })

  it('renders branding display name from backend on login page', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/settings/public/login-branding')) {
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'Acme Cloud',
              loginImageUrl: '',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/login')

    expect(await screen.findByRole('heading', { name: 'Acme Cloud', level: 1 })).toBeInTheDocument()
  })

  it('forgot-password flow shows non-enumerating success response', async () => {
    let forgotCalls = 0
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/settings/public/login-branding')) {
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'BasePro Web',
              loginImageUrl: '',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/auth/forgot-password') && init?.method === 'POST') {
          forgotCalls += 1
          return new Response(JSON.stringify({ status: 'accepted' }), {
            status: 202,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/forgot-password')

    fireEvent.change(await screen.findByRole('textbox', { name: 'Username or Email' }), {
      target: { value: 'alice@example.com' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Send Reset Instructions' }))

    await waitFor(() => {
      expect(forgotCalls).toBe(1)
    })
    expect(await screen.findByText('If the account exists, password reset instructions have been sent.')).toBeInTheDocument()
  })

  it('reset-password form validates password confirmation', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.includes('/settings/public/login-branding')) {
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'BasePro Web',
              loginImageUrl: '',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/reset-password?token=abc123')

    await screen.findByRole('button', { name: 'Reset Password' })
    fireEvent.change(screen.getByLabelText(/Reset Token/i), { target: { value: 'abc123' } })
    fireEvent.change(screen.getByLabelText(/^New Password/i), { target: { value: 'PasswordOne!' } })
    fireEvent.change(screen.getByLabelText(/Confirm New Password/i), { target: { value: 'PasswordTwo!' } })
    fireEvent.click(screen.getByRole('button', { name: 'Reset Password' }))

    expect(await screen.findByText('Passwords do not match.')).toBeInTheDocument()
  })
})

describe('web RBAC navigation', () => {
  it('shows Administration group with allowed children', async () => {
    setAuthSnapshot({
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      user: {
        id: 1,
        username: 'admin',
        roles: ['Admin'],
        permissions: ['settings.read', 'users.read'],
      },
    })

    renderWithRouter('/settings')

    expect(await screen.findByRole('heading', { name: 'Settings', level: 1 })).toBeInTheDocument()
    expect(screen.getByText('Administration')).toBeInTheDocument()
    expect(screen.queryByText('Users')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Toggle Administration menu' }))
    expect(screen.getByText('Users')).toBeInTheDocument()
    expect(screen.getByText('Roles')).toBeInTheDocument()
    expect(screen.getByText('Permissions')).toBeInTheDocument()
    expect(screen.queryByText('Audit Log')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Toggle Administration menu' }))
    await waitFor(() => {
      expect(screen.queryByText('Users')).not.toBeInTheDocument()
      expect(screen.queryByText('Roles')).not.toBeInTheDocument()
    })

    fireEvent.click(screen.getByRole('button', { name: 'Toggle Administration menu' }))
    await waitFor(() => {
      expect(screen.getByText('Users')).toBeInTheDocument()
    })
  })

  it('hides Administration group when no admin route is allowed', async () => {
    setAuthSnapshot({
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      user: {
        id: 2,
        username: 'staff-user',
        roles: ['Staff'],
        permissions: ['settings.write'],
      },
    })

    renderWithRouter('/settings')

    expect(await screen.findByRole('heading', { name: 'Settings', level: 1 })).toBeInTheDocument()
    expect(screen.queryByText('Administration')).not.toBeInTheDocument()
    expect(screen.queryByText('Users')).not.toBeInTheDocument()
    expect(screen.queryByText('Roles')).not.toBeInTheDocument()
    expect(screen.queryByText('Permissions')).not.toBeInTheDocument()
  })

  it('denies /settings for non-admin users without settings.write', async () => {
    setAuthSnapshot({
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      user: {
        id: 22,
        username: 'read-only-staff',
        roles: ['Staff'],
        permissions: ['settings.read'],
      },
    })

    renderWithRouter('/settings')

    expect(await screen.findByRole('heading', { name: 'Not Authorized', level: 1 })).toBeInTheDocument()
    expect(screen.getByText('You do not have permission to access this page.')).toBeInTheDocument()
  })

  it('unauthorized route navigation shows Not Authorized page', async () => {
    setAuthSnapshot({
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      user: {
        id: 3,
        username: 'staff-user',
        roles: ['Staff'],
        permissions: ['settings.read'],
      },
    })

    renderWithRouter('/roles')

    expect(await screen.findByRole('heading', { name: 'Not Authorized', level: 1 })).toBeInTheDocument()
    expect(screen.getByText('You do not have permission to access this page.')).toBeInTheDocument()
  })

  it('hides administration navigation and shows module-disabled state when administration module is disabled', async () => {
    applyEffectiveModuleEnablement({
      modules: [
        {
          moduleId: 'dashboard',
          flagKey: 'modules.dashboard.enabled',
          enabled: true,
          enabledByDefault: true,
          source: 'default',
        },
        {
          moduleId: 'administration',
          flagKey: 'modules.administration.enabled',
          enabled: false,
          enabledByDefault: true,
          source: 'config',
        },
        {
          moduleId: 'settings',
          flagKey: 'modules.settings.enabled',
          enabled: true,
          enabledByDefault: true,
          source: 'default',
        },
      ],
    })

    setAuthSnapshot({
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      user: {
        id: 4,
        username: 'admin',
        roles: ['Admin'],
        permissions: ['settings.read', 'users.read', 'users.write'],
      },
    })

    renderWithRouter('/users')

    expect(await screen.findByRole('heading', { name: 'Module Disabled', level: 1 })).toBeInTheDocument()
    expect(screen.getByText('Administration is currently disabled by platform configuration.')).toBeInTheDocument()
    expect(screen.queryByText('Administration')).not.toBeInTheDocument()
  })
})

describe('web settings page', () => {
  function authenticateForSettings() {
    setAuthSnapshot({
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      user: {
        id: 99,
        username: 'settings-user',
        roles: ['Admin'],
        permissions: ['settings.read'],
      },
    })
  }

  it('/settings renders and controls update persisted preferences', async () => {
    authenticateForSettings()
    renderWithRouter('/settings')

    expect(await screen.findByRole('heading', { name: 'Settings', level: 1 })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Forest' }))
    fireEvent.click(screen.getByRole('switch', { name: 'Start with side navigation collapsed' }))
    fireEvent.click(screen.getByRole('switch', { name: 'Show footer on authenticated pages' }))

    const rawPrefs = window.localStorage.getItem(UI_PREFERENCES_STORAGE_KEY)
    expect(rawPrefs).toBeTruthy()
    expect(JSON.parse(rawPrefs ?? '{}')).toEqual(
      expect.objectContaining({
        preset: 'forest',
        collapseNavByDefault: true,
        showFooter: false,
      }),
    )
  })

  it('changing mode persists after reload', async () => {
    authenticateForSettings()
    const firstRender = renderWithRouter('/settings')

    await screen.findByRole('heading', { name: 'Settings', level: 1 })
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Theme mode' }))
    fireEvent.click(await screen.findByRole('option', { name: 'Light' }))

    firstRender.unmount()
    renderWithRouter('/settings')

    expect(window.localStorage.getItem(UI_PREFERENCES_STORAGE_KEY)).toContain('"mode":"light"')
  })

  it('changing preset persists after reload', async () => {
    authenticateForSettings()
    const firstRender = renderWithRouter('/settings')

    await screen.findByRole('heading', { name: 'Settings', level: 1 })
    fireEvent.click(screen.getByRole('button', { name: 'Browse all presets' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Select Graphite preset' }))

    firstRender.unmount()
    renderWithRouter('/settings')

    expect(await screen.findByText('Active preset: Graphite')).toBeInTheDocument()

    const rawPrefs = window.localStorage.getItem(UI_PREFERENCES_STORAGE_KEY)
    expect(rawPrefs).toBeTruthy()
    expect(JSON.parse(rawPrefs ?? '{}')).toEqual(
      expect.objectContaining({
        preset: 'graphite',
      }),
    )
  })

  it('api base URL override persists after save', async () => {
    authenticateForSettings()
    renderWithRouter('/settings')

    await screen.findByRole('heading', { name: 'Settings', level: 1 })

    fireEvent.change(screen.getByLabelText('API Base URL Override'), {
      target: { value: 'http://127.0.0.1:8080/api/v1/' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Save Override' }))

    expect(window.localStorage.getItem(API_BASE_URL_OVERRIDE_STORAGE_KEY)).toBe('http://127.0.0.1:8080/api/v1')
  })

  it('saves login branding settings through backend API', async () => {
    setAuthSnapshot({
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      user: {
        id: 99,
        username: 'settings-user',
        roles: ['Admin'],
        permissions: ['settings.read', 'settings.write'],
      },
    })

    let brandingPutPayload: Record<string, unknown> | null = null
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.endsWith('/settings/login-branding') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'BasePro Web',
              loginImageUrl: 'https://cdn.example.com/old.png',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/settings/login-branding') && init?.method === 'PUT') {
          brandingPutPayload = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'Platform Pro',
              loginImageUrl: 'https://cdn.example.com/new.png',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/settings/module-enablement') && (!init?.method || init.method === 'GET')) {
          return new Response(JSON.stringify({ modules: [] }), { status: 200, headers: { 'Content-Type': 'application/json' } })
        }
        if (url.endsWith('/health')) {
          return new Response(JSON.stringify({ status: 'ok' }), { status: 200, headers: { 'Content-Type': 'application/json' } })
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/settings')
    await screen.findByRole('heading', { name: 'Settings', level: 1 })

    fireEvent.change(await screen.findByLabelText('Application Display Name'), { target: { value: 'Platform Pro' } })
    fireEvent.change(screen.getByLabelText('Login Image URL'), { target: { value: 'https://cdn.example.com/new.png' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save Branding' }))

    await waitFor(() => {
      expect(brandingPutPayload).toEqual({
        applicationDisplayName: 'Platform Pro',
        loginImageUrl: 'https://cdn.example.com/new.png',
      })
    })
  })

  it('shows module enablement list and write-permission guidance', async () => {
    setAuthSnapshot({
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      user: {
        id: 99,
        username: 'settings-user',
        roles: ['Admin'],
        permissions: ['settings.read'],
      },
    })

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === 'string' ? input : input.toString()
        if (url.endsWith('/settings/module-enablement') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              modules: [
                {
                  moduleId: 'administration',
                  flagKey: 'modules.administration.enabled',
                  enabled: true,
                  enabledByDefault: true,
                  description: 'Administration surfaces',
                  source: 'default',
                  adminControl: 'runtime',
                  editable: true,
                },
              ],
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/settings/login-branding') && (!init?.method || init.method === 'GET')) {
          return new Response(
            JSON.stringify({
              applicationDisplayName: 'BasePro Web',
              loginImageUrl: '',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          )
        }
        if (url.endsWith('/health')) {
          return new Response(JSON.stringify({ status: 'ok' }), { status: 200, headers: { 'Content-Type': 'application/json' } })
        }
        return new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } })
      }),
    )

    renderWithRouter('/settings')
    expect(await screen.findByText('Administration surfaces')).toBeInTheDocument()
    expect(
      screen.getByText('You need settings.write permission to change runtime-manageable module flags.'),
    ).toBeInTheDocument()
  })
})

describe('web AppShell layout behavior', () => {
  function authenticateForAppShell() {
    setAuthSnapshot({
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      user: {
        id: 50,
        username: 'app-shell-user',
        roles: ['Admin'],
        permissions: ['settings.read', 'users.read'],
      },
    })
  }

  it('authenticated /dashboard renders AppShell', async () => {
    authenticateForAppShell()
    renderWithRouter('/dashboard')

    expect(await screen.findByTestId('app-shell')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Dashboard', level: 1 })).toBeInTheDocument()
  })

  it('shows username and desktop-style user menu actions', async () => {
    authenticateForAppShell()
    renderWithRouter('/dashboard')

    await screen.findByRole('heading', { name: 'Dashboard', level: 1 })
    expect(screen.getByText('app-shell-user')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Open user menu' }))
    expect(await screen.findByRole('menuitem', { name: 'Settings' })).toBeInTheDocument()
    expect(screen.getByRole('menuitem', { name: 'Appearance' })).toBeInTheDocument()
    expect(screen.getByRole('menuitem', { name: 'Logout' })).toBeInTheDocument()
  })

  it('collapse toggle changes Drawer state and persists after reload', async () => {
    authenticateForAppShell()
    const firstRender = renderWithRouter('/dashboard')

    await screen.findByRole('heading', { name: 'Dashboard', level: 1 })
    expect(screen.getAllByRole('button', { name: 'Collapse navigation' }).length).toBeGreaterThan(0)

    fireEvent.click(screen.getAllByRole('button', { name: 'Collapse navigation' })[0])
    expect(screen.getAllByRole('button', { name: 'Expand navigation' }).length).toBeGreaterThan(0)

    const rawPrefs = window.localStorage.getItem(UI_PREFERENCES_STORAGE_KEY)
    expect(rawPrefs).toBeTruthy()
    expect(JSON.parse(rawPrefs ?? '{}')).toEqual(
      expect.objectContaining({
        collapseNavByDefault: true,
      }),
    )

    firstRender.unmount()
    renderWithRouter('/dashboard')

    expect((await screen.findAllByRole('button', { name: 'Expand navigation' })).length).toBeGreaterThan(0)
  })

  it('mobile drawer opens, closes, and closes on navigation selection', async () => {
    mockViewport(true)
    authenticateForAppShell()
    renderWithRouter('/dashboard')

    await screen.findByRole('heading', { name: 'Dashboard', level: 1 })
    fireEvent.click(screen.getByRole('button', { name: 'Open navigation menu' }))

    expect(await screen.findByRole('button', { name: 'Close navigation menu' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Settings' }))

    expect(await screen.findByRole('heading', { name: 'Settings', level: 1 })).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.queryByRole('button', { name: 'Close navigation menu' })).not.toBeInTheDocument()
    })
  })
})
