import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { CssBaseline, ThemeProvider, createTheme } from '@mui/material'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearSession, configureSessionStorage, setSession } from './auth/session'
import { createAppRouter } from './routes'
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
  let state = { ...seed }

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
    }
    return state
  })
  const resetSettingsMock = vi.fn(async () => {
    state = { ...defaultSettings }
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
      <ThemeProvider theme={createTheme()}>
        <CssBaseline />
        <RouterProvider router={router} />
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

describe('app routes and auth flow', () => {
  beforeEach(async () => {
    vi.restoreAllMocks()
    await clearSession()
  })

  afterEach(async () => {
    cleanup()
    await clearSession()
  })

  it('redirects /login to /setup when api base url is missing', async () => {
    const store = createMockSettingsStore({ ...defaultSettings, apiBaseUrl: '' })

    renderWithRouter('/login', store)

    expect(await screen.findByRole('heading', { name: 'Connect to API' })).toBeInTheDocument()
  })

  it('redirects /dashboard to /login when api base url is set but user is not authenticated', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
    })

    renderWithRouter('/dashboard', store)

    expect(await screen.findByRole('heading', { name: 'BasePro Desktop' })).toBeInTheDocument()
  })

  it('redirects /login to /dashboard when user is authenticated', async () => {
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

    renderWithRouter('/login', store)

    expect(await screen.findByRole('heading', { name: 'Dashboard' })).toBeInTheDocument()
  })

  it('logs in successfully and navigates to /dashboard', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
    })

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          accessToken: 'access-token',
          refreshToken: 'refresh-token',
          expiresIn: 300,
        }),
      }),
    )

    renderWithRouter('/login', store)

    fireEvent.change(await screen.findByRole('textbox', { name: /username/i }), {
      target: { value: 'alice' },
    })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: 'Login' }))

    expect(await screen.findByRole('heading', { name: 'Dashboard' })).toBeInTheDocument()
    expect(store.saveSettingsMock).toHaveBeenCalledWith(
      expect.objectContaining({ refreshToken: 'refresh-token' }),
    )
  })

  it('shows generic error for invalid credentials', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
    })

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        json: async () => ({
          error: {
            code: 'AUTH_UNAUTHORIZED',
            message: 'Invalid credentials',
          },
        }),
      }),
    )

    renderWithRouter('/login', store)

    fireEvent.change(await screen.findByRole('textbox', { name: /username/i }), {
      target: { value: 'alice' },
    })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'wrong' } })
    fireEvent.click(screen.getByRole('button', { name: 'Login' }))

    expect(await screen.findByText('Invalid username or password.')).toBeInTheDocument()
  })

  it('retries protected request after 401 when refresh succeeds', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
      refreshToken: 'refresh-token',
    })
    configureSessionStorage(store)
    await setSession({
      accessToken: 'old-access',
      refreshToken: 'refresh-token',
      expiresAt: Date.now() + 60_000,
    })

    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const authHeader = new Headers(init?.headers).get('Authorization')

      if (url.endsWith('/api/v1/auth/me') && authHeader === 'Bearer old-access') {
        return {
          ok: false,
          status: 401,
          json: async () => ({
            error: { code: 'AUTH_EXPIRED', message: 'expired' },
          }),
        }
      }

      if (url.endsWith('/api/v1/auth/refresh')) {
        return {
          ok: true,
          status: 200,
          json: async () => ({
            accessToken: 'new-access',
            refreshToken: 'new-refresh',
            expiresIn: 300,
          }),
        }
      }

      if (url.endsWith('/api/v1/auth/me') && authHeader === 'Bearer new-access') {
        return {
          ok: true,
          status: 200,
          json: async () => ({
            id: 1,
            username: 'alice',
            roles: [],
          }),
        }
      }

      throw new Error(`Unexpected request: ${url} (${authHeader ?? 'no auth'})`)
    })

    vi.stubGlobal('fetch', fetchMock)

    renderWithRouter('/dashboard', store)

    fireEvent.click(await screen.findByRole('button', { name: 'Load Profile' }))

    expect(await screen.findByText('Signed in as alice')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('http://127.0.0.1:8080/api/v1/auth/refresh', expect.any(Object))
    expect(store.saveSettingsMock).toHaveBeenCalledWith(
      expect.objectContaining({ refreshToken: 'new-refresh' }),
    )
  })

  it('forces logout, redirects to /login, and shows session-expired message when refresh fails', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
      refreshToken: 'refresh-token',
    })
    configureSessionStorage(store)
    await setSession({
      accessToken: 'old-access',
      refreshToken: 'refresh-token',
      expiresAt: Date.now() + 60_000,
    })

    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const authHeader = new Headers(init?.headers).get('Authorization')

      if (url.endsWith('/api/v1/auth/me') && authHeader === 'Bearer old-access') {
        return {
          ok: false,
          status: 401,
          json: async () => ({ error: { code: 'AUTH_EXPIRED', message: 'expired' } }),
        }
      }

      if (url.endsWith('/api/v1/auth/refresh')) {
        return {
          ok: false,
          status: 401,
          json: async () => ({ error: { code: 'AUTH_REFRESH_INVALID', message: 'invalid' } }),
        }
      }

      throw new Error(`Unexpected request: ${url}`)
    })

    vi.stubGlobal('fetch', fetchMock)

    renderWithRouter('/dashboard', store)

    fireEvent.click(await screen.findByRole('button', { name: 'Load Profile' }))

    expect(await screen.findByRole('heading', { name: 'BasePro Desktop' })).toBeInTheDocument()
    expect(await screen.findByText('Session expired. Please log in again.')).toBeInTheDocument()
    await waitFor(() => {
      expect(store.saveSettingsMock).toHaveBeenCalledWith(expect.objectContaining({ refreshToken: '' }))
    })
  })
})
