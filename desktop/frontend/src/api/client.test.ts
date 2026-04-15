import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { createApiClient } from './client'
import { clearSession, setSession } from '../auth/session'
import { defaultSettings, type AppSettings } from '../settings/types'

function mockJsonResponse(
  status: number,
  body: unknown,
  headers: Record<string, string> = { 'Content-Type': 'application/json' },
) {
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: new Headers(headers),
    json: vi.fn(async () => body),
    text: vi.fn(async () => JSON.stringify(body)),
  } as unknown as Response
}

describe('createApiClient', () => {
  const settings: AppSettings = {
    ...defaultSettings,
    apiBaseUrl: 'http://localhost:8080/api/v1',
  }

  beforeEach(async () => {
    vi.stubGlobal('fetch', vi.fn())
    await clearSession()
  })

  afterEach(async () => {
    await clearSession()
    vi.unstubAllGlobals()
  })

  it('prefers the saved API token for API requests when enabled', async () => {
    const client = createApiClient({
      getSettings: async () => ({
        ...settings,
        authMode: 'api_token',
        apiToken: 'saved-api-token',
      }),
    })

    vi.mocked(fetch).mockResolvedValueOnce(mockJsonResponse(200, { ok: true }))

    await client.request<{ ok: boolean }>('/users/1')

    const requestInit = vi.mocked(fetch).mock.calls[0]?.[1] as RequestInit
    const headers = requestInit.headers as Headers
    expect(headers.get('Authorization')).toBe('Bearer saved-api-token')
  })

  it('keeps me() on the current JWT session', async () => {
    await setSession({
      accessToken: 'session-access-token',
      refreshToken: 'refresh-token',
      expiresAt: Date.now() + 60_000,
    })

    const client = createApiClient({
      getSettings: async () => ({
        ...settings,
        authMode: 'api_token',
        apiToken: 'saved-api-token',
      }),
    })

    vi.mocked(fetch).mockResolvedValueOnce(mockJsonResponse(200, { id: 1, username: 'alice' }))

    await client.me()

    const requestInit = vi.mocked(fetch).mock.calls[0]?.[1] as RequestInit
    const headers = requestInit.headers as Headers
    expect(headers.get('Authorization')).toBe('Bearer session-access-token')
  })

  it('creates API tokens with the active request auth', async () => {
    const client = createApiClient({
      getSettings: async () => ({
        ...settings,
        authMode: 'api_token',
        apiToken: 'saved-api-token',
      }),
    })

    vi.mocked(fetch).mockResolvedValueOnce(
      mockJsonResponse(201, {
        id: 9,
        name: 'Desktop API token',
        prefix: 'bpt_123',
        token: 'plaintext-token',
        permissions: ['settings.read'],
      }),
    )

    const result = await client.createApiToken({
      name: 'Desktop API token',
      permissions: ['settings.read'],
    })

    expect(result.token).toBe('plaintext-token')
    const requestInit = vi.mocked(fetch).mock.calls[0]?.[1] as RequestInit
    const headers = requestInit.headers as Headers
    expect(headers.get('Authorization')).toBe('Bearer saved-api-token')
  })
})
