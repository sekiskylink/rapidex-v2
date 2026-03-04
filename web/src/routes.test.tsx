import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearAuthSnapshot, persistRefreshToken, setAuthSnapshot } from './auth/state'
import { API_BASE_URL_OVERRIDE_STORAGE_KEY } from './lib/apiBaseUrl'
import { apiRequest } from './lib/api'
import { createAppRouter } from './routes'
import { SnackbarProvider } from './ui/snackbar'
import { UI_PREFERENCES_STORAGE_KEY } from './ui/preferences'

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
  vi.stubEnv('VITE_API_BASE_URL', 'http://localhost:8080/api/v1')
  vi.stubGlobal('fetch', vi.fn())
})

afterEach(() => {
  cleanup()
  window.localStorage.clear()
  clearAuthSnapshot()
  vi.unstubAllEnvs()
  vi.unstubAllGlobals()
})

describe('web auth routes', () => {
  it('login success redirects to /dashboard', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            accessToken: 'access-token',
            refreshToken: 'refresh-token',
            expiresIn: 300,
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 1,
            username: 'admin',
            roles: ['Admin'],
            permissions: ['users.read', 'settings.read'],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        ),
      )

    renderWithRouter('/login')

    fireEvent.change(await screen.findByRole('textbox', { name: /username/i }), { target: { value: 'admin' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: 'Login' }))

    expect(await screen.findByRole('heading', { name: 'Dashboard', level: 1 })).toBeInTheDocument()
  })

  it('login failure shows backend message and request ID', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(
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
      ),
    )

    renderWithRouter('/login')

    fireEvent.change(await screen.findByRole('textbox', { name: /username/i }), { target: { value: 'bad-user' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'bad-pass' } })
    fireEvent.click(screen.getByRole('button', { name: 'Login' }))

    expect(await screen.findByText('Invalid credentials Request ID: req-401')).toBeInTheDocument()
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
})

describe('web RBAC navigation', () => {
  it('role Admin sees Payroll module', async () => {
    setAuthSnapshot({
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      user: {
        id: 1,
        username: 'admin',
        roles: ['Admin'],
        permissions: ['settings.read'],
      },
    })

    renderWithRouter('/settings')

    expect(await screen.findByRole('heading', { name: 'Settings', level: 1 })).toBeInTheDocument()
    expect(screen.getByText('Payroll')).toBeInTheDocument()
  })

  it('role Staff does not see Payroll module', async () => {
    setAuthSnapshot({
      isAuthenticated: true,
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      user: {
        id: 2,
        username: 'staff-user',
        roles: ['Staff'],
        permissions: ['settings.read'],
      },
    })

    renderWithRouter('/settings')

    expect(await screen.findByRole('heading', { name: 'Settings', level: 1 })).toBeInTheDocument()
    expect(screen.queryByText('Payroll')).not.toBeInTheDocument()
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

    renderWithRouter('/users')

    expect(await screen.findByRole('heading', { name: 'Not Authorized', level: 1 })).toBeInTheDocument()
    expect(screen.getByText('You do not have permission to access this page.')).toBeInTheDocument()
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

    fireEvent.click(screen.getByRole('radio', { name: 'Dark' }))
    fireEvent.click(screen.getByLabelText('Select Forest preset'))
    fireEvent.click(screen.getByRole('switch', { name: 'Collapse side navigation by default' }))

    const rawPrefs = window.localStorage.getItem(UI_PREFERENCES_STORAGE_KEY)
    expect(rawPrefs).toBeTruthy()
    expect(JSON.parse(rawPrefs ?? '{}')).toEqual(
      expect.objectContaining({
        mode: 'dark',
        preset: 'forest',
        collapseNavByDefault: true,
      }),
    )
  })

  it('changing mode persists after reload', async () => {
    authenticateForSettings()
    const firstRender = renderWithRouter('/settings')

    await screen.findByRole('heading', { name: 'Settings', level: 1 })
    fireEvent.click(screen.getByRole('radio', { name: 'Light' }))

    firstRender.unmount()
    renderWithRouter('/settings')

    expect(await screen.findByRole('radio', { name: 'Light' })).toBeChecked()
  })

  it('changing preset persists after reload', async () => {
    authenticateForSettings()
    const firstRender = renderWithRouter('/settings')

    await screen.findByRole('heading', { name: 'Settings', level: 1 })
    fireEvent.click(screen.getByLabelText('Select Graphite preset'))

    firstRender.unmount()
    renderWithRouter('/settings')

    expect(await screen.findByLabelText('Select Graphite preset')).toBeInTheDocument()
    expect(screen.getByText('Graphite')).toBeInTheDocument()

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
})
