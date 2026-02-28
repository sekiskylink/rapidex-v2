import React from 'react'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { CssBaseline, ThemeProvider, createTheme } from '@mui/material'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createAppRouter } from './routes'
import { defaultSettings, type AppSettings, type SaveSettingsPatch, type SettingsStore } from './settings/types'

function createMockSettingsStore(seed: AppSettings): SettingsStore & {
  loadSettingsMock: ReturnType<typeof vi.fn>
  saveSettingsMock: ReturnType<typeof vi.fn>
  resetSettingsMock: ReturnType<typeof vi.fn>
} {
  let state = { ...seed }

  const loadSettingsMock = vi.fn(async () => state)
  const saveSettingsMock = vi.fn(async (patch: SaveSettingsPatch) => {
    state = {
      ...state,
      ...patch,
      apiToken:
        patch.authMode === 'password' || (patch.authMode === undefined && state.authMode === 'password')
          ? undefined
          : patch.apiToken ?? state.apiToken,
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

describe('app routes', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })
  afterEach(() => {
    cleanup()
  })

  it('redirects /login to /setup when api base url is missing', async () => {
    const store = createMockSettingsStore({ ...defaultSettings, apiBaseUrl: '' })

    renderWithRouter('/login', store)

    expect(await screen.findByRole('heading', { name: 'Connect to API' })).toBeInTheDocument()
  })

  it('redirects / to /setup when api base url is missing', async () => {
    const store = createMockSettingsStore({ ...defaultSettings, apiBaseUrl: '' })

    renderWithRouter('/', store)

    expect(await screen.findByRole('heading', { name: 'Connect to API' })).toBeInTheDocument()
  })

  it('renders not found for unknown route', async () => {
    const store = createMockSettingsStore({ ...defaultSettings, apiBaseUrl: '' })

    renderWithRouter('/missing-route', store)

    expect(await screen.findByRole('heading', { name: 'Not Found' })).toBeInTheDocument()
  })

  it('saves setup and continues to login when api base url is configured', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
      requestTimeoutSeconds: 20,
    })

    renderWithRouter('/setup', store)

    expect(await screen.findByRole('heading', { name: 'Connect to API' })).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('API Base URL'), {
      target: { value: 'http://localhost:8080' },
    })

    fireEvent.click(screen.getByRole('button', { name: 'Save & Continue' }))

    expect(await screen.findByRole('heading', { name: 'Login' })).toBeInTheDocument()
    expect(store.saveSettingsMock).toHaveBeenCalledWith(
      expect.objectContaining({ apiBaseUrl: 'http://localhost:8080' }),
    )
  })

  it('tests backend connection from setup page using health endpoint', async () => {
    const store = createMockSettingsStore({
      ...defaultSettings,
      apiBaseUrl: 'http://127.0.0.1:8080',
      requestTimeoutSeconds: 10,
    })
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200 })
    vi.stubGlobal('fetch', fetchMock)

    renderWithRouter('/setup', store)

    await screen.findByRole('heading', { name: 'Connect to API' })
    fireEvent.click(screen.getByRole('button', { name: 'Test Connection' }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('http://127.0.0.1:8080/api/v1/health', {
        method: 'GET',
        signal: expect.any(AbortSignal),
      })
    })

    expect(await screen.findByText('Connection succeeded.')).toBeInTheDocument()
  })
})
