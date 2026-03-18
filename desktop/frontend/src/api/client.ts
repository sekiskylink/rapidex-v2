import {
  getAccessToken,
  getAccessTokenExpiresAt,
  getPersistedRefreshToken,
  handleNetworkUnreachable,
  handleSessionExpiry,
  setSession,
} from '../auth/session'
import type { LoginRequest, LoginResponse, MeResponse, RefreshRequest, RefreshResponse } from '../auth/types'
import type { BootstrapPayload } from '../bootstrap/state'
import type { ModuleEnablementApiResponse } from '../registry/moduleEnablement'
import type { AppSettings } from '../settings/types'

interface ApiClientDeps {
  getSettings: () => Promise<AppSettings>
}

interface VersionResponse {
  version: string
  commit: string
  buildDate: string
}

export interface DashboardSnapshot {
  generatedAt: string
  health: DashboardHealth
  kpis: DashboardKpis
  trends: DashboardTrends
  attention: DashboardAttention
  workers: DashboardWorkersSummary
  recentEvents: DashboardEventSummary[]
}

export interface DashboardHealth {
  status: string
  signals: string[]
}

export interface DashboardKpis {
  requestsToday: number
  pendingRequests: number
  pendingDeliveries: number
  runningDeliveries: number
  failedDeliveriesLastHour: number
  pollingJobs: number
  ingestBacklog: number
  healthyWorkers: number
  unhealthyWorkers: number
}

export interface DashboardTrends {
  requestsByHour: DashboardTimeCountPoint[]
  deliveriesByStatus: DashboardStatusCountPoint[]
  jobsByState: DashboardStatusCountPoint[]
  failuresByServer: DashboardServerCountPoint[]
}

export interface DashboardTimeCountPoint {
  bucketStart: string
  count: number
}

export interface DashboardStatusCountPoint {
  bucketStart: string
  status: string
  count: number
}

export interface DashboardServerCountPoint {
  serverId: number
  serverName: string
  count: number
}

export interface DashboardAttention {
  failedDeliveries: DashboardDeliveryAttentionList
  staleRunningDeliveries: DashboardDeliveryAttentionList
  stuckJobs: DashboardJobAttentionList
  recentIngestFailures: DashboardIngestAttentionList
  unhealthyWorkers: DashboardWorkerAttentionList
}

export interface DashboardDeliveryAttentionList {
  total: number
  items: DashboardDeliveryAttentionItem[]
}

export interface DashboardJobAttentionList {
  total: number
  items: DashboardJobAttentionItem[]
}

export interface DashboardIngestAttentionList {
  total: number
  items: DashboardIngestAttentionItem[]
}

export interface DashboardWorkerAttentionList {
  total: number
  items: DashboardWorkerAttentionItem[]
}

export interface DashboardDeliveryAttentionItem {
  id: number
  uid: string
  requestId: number
  requestUid: string
  serverId: number
  serverName: string
  correlationId: string
  status: string
  errorMessage: string
  startedAt?: string | null
  finishedAt?: string | null
  nextEligibleAt?: string | null
  updatedAt: string
}

export interface DashboardJobAttentionItem {
  id: number
  uid: string
  deliveryId: number
  deliveryUid: string
  requestId: number
  requestUid: string
  correlationId: string
  remoteJobId: string
  remoteStatus: string
  currentState: string
  nextPollAt?: string | null
  updatedAt: string
}

export interface DashboardIngestAttentionItem {
  id: number
  uid: string
  originalName: string
  currentPath: string
  status: string
  lastErrorCode: string
  lastErrorMessage: string
  requestId?: number | null
  failedAt?: string | null
  updatedAt: string
}

export interface DashboardWorkerAttentionItem {
  id: number
  uid: string
  workerType: string
  workerName: string
  status: string
  lastHeartbeatAt?: string | null
  startedAt: string
  updatedAt: string
}

export interface DashboardWorkersSummary {
  heartbeatFreshnessSeconds: number
  items: DashboardWorkerAttentionItem[]
}

export interface DashboardEventSummary {
  type: string
  timestamp: string
  severity: string
  entityType: string
  entityId?: number
  entityUid?: string
  summary: string
  correlationId?: string
  requestId?: number | null
  deliveryId?: number | null
  jobId?: number | null
  workerId?: number | null
}

export interface LoginBrandingResponse {
  appDisplayName?: string
  applicationDisplayName?: string
  loginImageUrl?: string | null
}

export interface ForgotPasswordRequest {
  identifier: string
  resetUrl?: string
}

export interface ForgotPasswordResponse {
  status: string
}

export interface ResetPasswordRequest {
  token: string
  password: string
}

export interface ResetPasswordResponse {
  status: string
}

export interface LoginBrandingUpdateRequest {
  applicationDisplayName: string
  loginImageUrl?: string | null
}

export interface ModuleEnablementUpdateRequest {
  modules: Array<{
    moduleId: string
    enabled: boolean
  }>
}

interface ApiErrorPayload {
  error?: {
    code?: string
    message?: string
    details?: Record<string, unknown>
  }
}

class ApiError extends Error {
  status: number
  code?: string
  details?: Record<string, unknown>
  requestId?: string

  constructor(status: number, message: string, code?: string, details?: Record<string, unknown>, requestId?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = details
    this.requestId = requestId
  }
}

let refreshPromise: Promise<boolean> | null = null

function normalizeBaseUrl(baseUrl: string) {
  const trimmed = baseUrl.trim().replace(/\/+$/, '')
  return trimmed.endsWith('/api/v1') ? trimmed.slice(0, -'/api/v1'.length) : trimmed
}

function isSessionInvalidCode(code?: string) {
  return code === 'AUTH_REFRESH_INVALID' || code === 'AUTH_REFRESH_REUSED' || code === 'AUTH_EXPIRED'
}

function isNetworkError(error: unknown) {
  return error instanceof TypeError || (error instanceof DOMException && error.name === 'AbortError')
}

async function parseApiError(response: Response) {
  let code: string | undefined
  let message = `Request failed: HTTP ${response.status}`
  let details: Record<string, unknown> | undefined
  const requestId = response.headers.get('X-Request-Id') ?? response.headers.get('x-request-id') ?? undefined

  try {
    const payload = (await response.json()) as ApiErrorPayload
    if (payload.error?.message) {
      message = payload.error.message
    }
    code = payload.error?.code
    details = payload.error?.details
  } catch {
    // Keep fallback message when body is not JSON.
  }

  return new ApiError(response.status, message, code, details, requestId)
}

export function createApiClient(deps: ApiClientDeps) {
  async function fetchWithTimeout(url: string, init: RequestInit = {}) {
    const settings = await deps.getSettings()
    const controller = new AbortController()
    const timeoutMs = Math.max(1, settings.requestTimeoutSeconds) * 1000
    const timeout = window.setTimeout(() => controller.abort(), timeoutMs)

    try {
      return await fetch(url, {
        ...init,
        signal: controller.signal,
      })
    } catch (error) {
      if (isNetworkError(error)) {
        handleNetworkUnreachable()
      }
      throw error
    } finally {
      window.clearTimeout(timeout)
    }
  }

  async function runRefresh(settings: AppSettings) {
    const refreshToken = await getPersistedRefreshToken()
    if (!refreshToken) {
      await handleSessionExpiry()
      return false
    }

    let response: Response
    try {
      response = await fetchWithTimeout(`${normalizeBaseUrl(settings.apiBaseUrl)}/api/v1/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refreshToken } satisfies RefreshRequest),
      })
    } catch (error) {
      if (isNetworkError(error)) {
        return false
      }
      throw error
    }

    if (!response.ok) {
      const apiError = await parseApiError(response)
      if (apiError.status === 401 && isSessionInvalidCode(apiError.code)) {
        await handleSessionExpiry()
        return false
      }
      throw apiError
    }

    const refresh = (await response.json()) as RefreshResponse
    await setSession({
      accessToken: refresh.accessToken,
      refreshToken: refresh.refreshToken,
      expiresAt: Date.now() + refresh.expiresIn * 1000,
    })
    return true
  }

  async function refreshSession(settings: AppSettings) {
    if (!refreshPromise) {
      refreshPromise = runRefresh(settings).finally(() => {
        refreshPromise = null
      })
    }
    return refreshPromise
  }

  async function authorizedRequest<T>(path: string, init: RequestInit = {}, allowRetry = true): Promise<T> {
    const settings = await deps.getSettings()
    const baseUrl = normalizeBaseUrl(settings.apiBaseUrl)
    if (!baseUrl) {
      throw new Error('API base URL is required')
    }

    if (refreshPromise) {
      await refreshPromise
    }

    const accessToken = getAccessToken()
    const expiresAt = getAccessTokenExpiresAt() ?? 0
    if (accessToken && expiresAt > 0 && expiresAt <= Date.now()) {
      await refreshSession(settings)
    }

    const token = getAccessToken()
    const headers = new Headers(init.headers)
    if (!headers.has('Content-Type') && init.body) {
      headers.set('Content-Type', 'application/json')
    }
    if (token) {
      headers.set('Authorization', `Bearer ${token}`)
    }

    let response: Response
    try {
      response = await fetchWithTimeout(`${baseUrl}${path}`, {
        ...init,
        headers,
      })
    } catch (error) {
      throw error
    }

    if (response.status === 401 && allowRetry) {
      const refreshed = await refreshSession(settings)
      if (refreshed) {
        return authorizedRequest<T>(path, init, false)
      }
      throw new ApiError(401, 'Session expired', 'AUTH_EXPIRED')
    }

    if (!response.ok) {
      throw await parseApiError(response)
    }

    if (response.status === 204) {
      return undefined as T
    }

    return (await response.json()) as T
  }

  return {
    async healthCheck() {
      const settings = await deps.getSettings()
      const baseUrl = normalizeBaseUrl(settings.apiBaseUrl)
      if (!baseUrl) {
        throw new Error('API base URL is required')
      }

      const response = await fetchWithTimeout(`${baseUrl}/api/v1/health`, {
        method: 'GET',
      })
      if (!response.ok) {
        throw new Error(`Health check failed: HTTP ${response.status}`)
      }
    },

    async login(payload: LoginRequest) {
      const settings = await deps.getSettings()
      const baseUrl = normalizeBaseUrl(settings.apiBaseUrl)
      if (!baseUrl) {
        throw new Error('API base URL is required')
      }

      const response = await fetchWithTimeout(`${baseUrl}/api/v1/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })

      if (!response.ok) {
        throw await parseApiError(response)
      }

      return (await response.json()) as LoginResponse
    },

    async me() {
      return authorizedRequest<MeResponse>('/api/v1/auth/me', {
        method: 'GET',
      })
    },

    async version() {
      const settings = await deps.getSettings()
      const baseUrl = normalizeBaseUrl(settings.apiBaseUrl)
      if (!baseUrl) {
        throw new Error('API base URL is required')
      }

      const response = await fetchWithTimeout(`${baseUrl}/api/v1/version`, {
        method: 'GET',
      })
      if (!response.ok) {
        throw new Error(`Version check failed: HTTP ${response.status}`)
      }
      return (await response.json()) as VersionResponse
    },

    async getEffectiveModuleEnablement() {
      const settings = await deps.getSettings()
      const baseUrl = normalizeBaseUrl(settings.apiBaseUrl)
      if (!baseUrl) {
        throw new Error('API base URL is required')
      }

      const response = await fetchWithTimeout(`${baseUrl}/api/v1/modules/effective`, {
        method: 'GET',
      })
      if (!response.ok) {
        throw await parseApiError(response)
      }

      return (await response.json()) as ModuleEnablementApiResponse
    },

    async getBootstrap() {
      return authorizedRequest<BootstrapPayload>('/api/v1/bootstrap', {
        method: 'GET',
      })
    },

    async getOperationsDashboard() {
      return authorizedRequest<DashboardSnapshot>('/api/v1/dashboard/operations', {
        method: 'GET',
      })
    },

    async getPublicLoginBranding() {
      const settings = await deps.getSettings()
      const baseUrl = normalizeBaseUrl(settings.apiBaseUrl)
      if (!baseUrl) {
        throw new Error('API base URL is required')
      }

      const response = await fetchWithTimeout(`${baseUrl}/api/v1/settings/public/login-branding`, {
        method: 'GET',
      })
      if (!response.ok) {
        throw await parseApiError(response)
      }
      return (await response.json()) as LoginBrandingResponse
    },

    async forgotPassword(payload: ForgotPasswordRequest) {
      const settings = await deps.getSettings()
      const baseUrl = normalizeBaseUrl(settings.apiBaseUrl)
      if (!baseUrl) {
        throw new Error('API base URL is required')
      }

      const response = await fetchWithTimeout(`${baseUrl}/api/v1/auth/forgot-password`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!response.ok) {
        throw await parseApiError(response)
      }
      return (await response.json()) as ForgotPasswordResponse
    },

    async resetPassword(payload: ResetPasswordRequest) {
      const settings = await deps.getSettings()
      const baseUrl = normalizeBaseUrl(settings.apiBaseUrl)
      if (!baseUrl) {
        throw new Error('API base URL is required')
      }

      const response = await fetchWithTimeout(`${baseUrl}/api/v1/auth/reset-password`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!response.ok) {
        throw await parseApiError(response)
      }
      return (await response.json()) as ResetPasswordResponse
    },

    async getLoginBranding() {
      return authorizedRequest<LoginBrandingResponse>('/api/v1/settings/login-branding', {
        method: 'GET',
      })
    },

    async updateLoginBranding(payload: LoginBrandingUpdateRequest) {
      return authorizedRequest<LoginBrandingResponse>('/api/v1/settings/login-branding', {
        method: 'PUT',
        body: JSON.stringify(payload),
      })
    },

    async getModuleEnablementSettings() {
      return authorizedRequest<ModuleEnablementApiResponse>('/api/v1/settings/module-enablement', {
        method: 'GET',
      })
    },

    async updateModuleEnablementSettings(payload: ModuleEnablementUpdateRequest) {
      return authorizedRequest<ModuleEnablementApiResponse>('/api/v1/settings/module-enablement', {
        method: 'PUT',
        body: JSON.stringify(payload),
      })
    },

    request<T>(path: string, init: RequestInit = {}) {
      return authorizedRequest<T>(path, init)
    },
  }
}

export { ApiError }
