import React from 'react'
import { useNavigate } from '@tanstack/react-router'
import { apiRequest, configureApiClient, type ApiError } from '../lib/api'
import { useAppNotify } from '../notifications/facade'
import { applyBootstrap, clearBootstrap, hydrateBootstrapFromCache, type BootstrapPayload } from '../bootstrap/state'
import { rememberIntendedDestination } from './sessionExpiry'
import {
  clearAuthSnapshot,
  getAuthSnapshot,
  getPersistedRefreshToken,
  persistRefreshToken,
  setAuthSnapshot,
  subscribeAuthSnapshot,
  type AuthUser,
} from './state'

interface AuthTokensResponse {
  accessToken: string
  refreshToken: string
  expiresIn: number
}

interface MeResponse {
  id: number
  username: string
  roles?: string[]
  permissions?: string[]
}

interface AuthContextValue {
  isAuthenticated: boolean
  accessToken: string | null
  user: AuthUser | null
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
  refresh: () => Promise<boolean>
}

const AuthContext = React.createContext<AuthContextValue | undefined>(undefined)

function toAuthUser(payload: MeResponse): AuthUser {
  return {
    id: payload.id,
    username: payload.username,
    roles: payload.roles ?? [],
    permissions: payload.permissions ?? [],
  }
}

function toAuthUserFromBootstrap(payload: BootstrapPayload | null | undefined): AuthUser | null {
  if (!payload?.principal || payload.principal.type !== 'user' || !payload.principal.userId || !payload.principal.username) {
    return null
  }
  return {
    id: payload.principal.userId,
    username: payload.principal.username,
    roles: payload.principal.roles ?? [],
    permissions: payload.principal.permissions ?? [],
  }
}

export function AuthProvider({ children }: React.PropsWithChildren) {
  const navigate = useNavigate()
  const notify = useAppNotify()
  const [snapshot, setSnapshot] = React.useState(getAuthSnapshot())
  const refreshPromiseRef = React.useRef<Promise<boolean> | null>(null)

  React.useEffect(() => {
    return subscribeAuthSnapshot(() => {
      setSnapshot(getAuthSnapshot())
    })
  }, [])

  const applySession = React.useCallback((tokens: AuthTokensResponse, user: AuthUser | null) => {
    persistRefreshToken(tokens.refreshToken)
    setAuthSnapshot({
      isAuthenticated: true,
      accessToken: tokens.accessToken,
      refreshToken: tokens.refreshToken,
      user,
    })
  }, [])

  const clearSession = React.useCallback(() => {
    clearAuthSnapshot()
    clearBootstrap()
  }, [])

  const refresh = React.useCallback(async (): Promise<boolean> => {
    if (refreshPromiseRef.current) {
      return refreshPromiseRef.current
    }

    refreshPromiseRef.current = (async () => {
      const refreshToken = getPersistedRefreshToken().trim()
      if (!refreshToken) {
        clearSession()
        return false
      }

      try {
        const tokens = await apiRequest<AuthTokensResponse>(
          '/auth/refresh',
          {
            method: 'POST',
            body: JSON.stringify({ refreshToken }),
          },
          { withAuth: false, retryOnUnauthorized: false, preferApiToken: false },
        )

        applySession(tokens, getAuthSnapshot().user)
        let user: AuthUser | null = null
        const bootstrap = await apiRequest<BootstrapPayload>(
          '/bootstrap',
          { method: 'GET' },
          { retryOnUnauthorized: false, preferApiToken: false },
        ).catch(() => null)
        if (bootstrap) {
          applyBootstrap(bootstrap, 'live')
          user = toAuthUserFromBootstrap(bootstrap)
        } else {
          hydrateBootstrapFromCache()
        }
        if (!user) {
          const me = await apiRequest<MeResponse>(
            '/auth/me',
            { method: 'GET' },
            { retryOnUnauthorized: false, preferApiToken: false },
          )
          user = toAuthUser(me)
        }
        applySession(tokens, user)
        return true
      } catch {
        clearSession()
        return false
      }
    })().finally(() => {
      refreshPromiseRef.current = null
    })

    return refreshPromiseRef.current
  }, [applySession, clearSession])

  const handleSessionExpired = React.useCallback(async () => {
    const currentPath = `${window.location.pathname}${window.location.search}${window.location.hash}`
    rememberIntendedDestination(currentPath)
    clearSession()
    notify.warning('Session expired. Please log in again.', { persistent: true })
    await navigate({ to: '/login', replace: true })
  }, [clearSession, navigate, notify])

  React.useEffect(() => {
    configureApiClient({
      getAccessToken: () => getAuthSnapshot().accessToken,
      onUnauthorized: async () => {
        const refreshed = await refresh()
        if (!refreshed) {
          await handleSessionExpired()
        }
        return refreshed
      },
    })
  }, [handleSessionExpired, refresh])

  const login = React.useCallback(
    async (username: string, password: string) => {
      const tokens = await apiRequest<AuthTokensResponse>(
        '/auth/login',
        {
          method: 'POST',
          body: JSON.stringify({
            username: username.trim(),
            password,
          }),
        },
        { withAuth: false, retryOnUnauthorized: false, preferApiToken: false },
      )

      applySession(tokens, null)

      try {
        let user: AuthUser | null = null
        const bootstrap = await apiRequest<BootstrapPayload>(
          '/bootstrap',
          { method: 'GET' },
          { retryOnUnauthorized: false, preferApiToken: false },
        ).catch(() => null)
        if (bootstrap) {
          applyBootstrap(bootstrap, 'live')
          user = toAuthUserFromBootstrap(bootstrap)
        }
        if (!user) {
          const me = await apiRequest<MeResponse>('/auth/me', { method: 'GET' }, { preferApiToken: false })
          user = toAuthUser(me)
        }
        setAuthSnapshot({
          ...getAuthSnapshot(),
          user,
          isAuthenticated: true,
        })
      } catch (error) {
        clearSession()
        throw error
      }
    },
    [applySession, clearSession],
  )

  const logout = React.useCallback(async () => {
    const refreshToken = getPersistedRefreshToken().trim()
    if (refreshToken) {
      try {
        await apiRequest(
          '/auth/logout',
          {
            method: 'POST',
            body: JSON.stringify({ refreshToken }),
          },
          { withAuth: false, retryOnUnauthorized: false },
        )
      } catch {
        // Ignore logout API failures and clear local session regardless.
      }
    }
    clearSession()
    await navigate({ to: '/login', replace: true })
  }, [clearSession, navigate])

  React.useEffect(() => {
    if (snapshot.isAuthenticated) {
      return
    }

    const persistedRefreshToken = getPersistedRefreshToken().trim()
    if (!persistedRefreshToken) {
      return
    }

    hydrateBootstrapFromCache()
    void refresh()
  }, [refresh, snapshot.isAuthenticated])

  const value = React.useMemo<AuthContextValue>(
    () => ({
      isAuthenticated: snapshot.isAuthenticated,
      accessToken: snapshot.accessToken,
      user: snapshot.user,
      login,
      logout,
      refresh,
    }),
    [login, logout, refresh, snapshot.accessToken, snapshot.isAuthenticated, snapshot.user],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const context = React.useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used inside AuthProvider')
  }
  return context
}

export function isApiError(value: unknown): value is ApiError {
  if (!value || typeof value !== 'object') {
    return false
  }
  const candidate = value as Partial<ApiError>
  return typeof candidate.code === 'string' && typeof candidate.message === 'string'
}
