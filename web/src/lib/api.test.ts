import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { apiRequest, configureApiClient, type ApiError } from './api'

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

describe('apiRequest', () => {
  beforeEach(() => {
    vi.stubEnv('VITE_API_BASE_URL', 'http://localhost:8080/api/v1')
    vi.stubGlobal('fetch', vi.fn())
    window.localStorage.clear()
    configureApiClient({})
  })

  afterEach(() => {
    window.localStorage.clear()
    vi.unstubAllEnvs()
    vi.unstubAllGlobals()
  })

  it('parses standardized error JSON into ApiError', async () => {
    const errorResponse = {
      error: {
        code: 'AUTH_UNAUTHORIZED',
        message: 'Authentication required',
        details: { field: 'token' },
      },
    }

    vi.mocked(fetch).mockResolvedValueOnce(mockJsonResponse(401, errorResponse))

    let caught: ApiError | undefined
    try {
      await apiRequest('/auth/me')
    } catch (error) {
      caught = error as ApiError
    }

    expect(caught).toEqual({
      status: 401,
      code: 'AUTH_UNAUTHORIZED',
      message: 'Authentication required',
      details: { field: 'token' },
      requestId: undefined,
    })
  })

  it('extracts X-Request-Id from non-2xx responses', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      mockJsonResponse(
        401,
        {
          error: {
            code: 'AUTH_EXPIRED',
            message: 'Session expired',
          },
        },
        {
          'Content-Type': 'application/json',
          'X-Request-Id': 'req-123',
        },
      ),
    )

    await expect(apiRequest('/auth/me')).rejects.toMatchObject({
      code: 'AUTH_EXPIRED',
      requestId: 'req-123',
    })
  })

  it('does not log Authorization header values', async () => {
    const logger = vi.fn()
    configureApiClient({
      getAccessToken: () => 'super-secret-token',
      logger,
    })

    vi.mocked(fetch).mockResolvedValueOnce(mockJsonResponse(200, { ok: true }))

    await apiRequest<{ ok: boolean }>('/health')

    expect(logger).toHaveBeenCalledTimes(1)
    const metadata = logger.mock.calls[0]?.[1] as { headers?: Record<string, string> }
    const loggedAuthHeader = metadata.headers?.Authorization ?? metadata.headers?.authorization
    expect(loggedAuthHeader).toBe('[REDACTED]')
    expect(JSON.stringify(metadata)).not.toContain('super-secret-token')
  })

  it('returns parsed response body on success', async () => {
    configureApiClient({
      getAccessToken: () => 'access-token',
    })

    vi.mocked(fetch).mockResolvedValueOnce(mockJsonResponse(200, { id: 1, name: 'alice' }))

    const result = await apiRequest<{ id: number; name: string }>('/users/1', {
      method: 'GET',
    })

    expect(result).toEqual({ id: 1, name: 'alice' })
    expect(fetch).toHaveBeenCalledWith('http://localhost:8080/api/v1/users/1', {
      method: 'GET',
      headers: expect.any(Headers),
    })

    const requestInit = vi.mocked(fetch).mock.calls[0]?.[1] as RequestInit
    const headers = requestInit.headers as Headers
    expect(headers.get('Authorization')).toBe('Bearer access-token')
  })

  it('prefers a saved API token when auth mode is api_token', async () => {
    window.localStorage.setItem('basepro.web.auth_mode', 'api_token')
    window.localStorage.setItem('basepro.web.api_token', 'saved-api-token')
    configureApiClient({
      getAccessToken: () => 'session-token',
    })

    vi.mocked(fetch).mockResolvedValueOnce(mockJsonResponse(200, { ok: true }))

    await apiRequest<{ ok: boolean }>('/users/1')

    const requestInit = vi.mocked(fetch).mock.calls[0]?.[1] as RequestInit
    const headers = requestInit.headers as Headers
    expect(headers.get('Authorization')).toBe('Bearer saved-api-token')
  })
})
