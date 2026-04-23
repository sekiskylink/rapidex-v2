export interface AuthUser {
  id: number
  username: string
  roles: string[]
  permissions: string[]
  assignedOrgUnitIds?: number[]
  isOrgUnitScopeRestricted?: boolean
}

export interface AuthSnapshot {
  isAuthenticated: boolean
  accessToken: string | null
  refreshToken: string | null
  user: AuthUser | null
}

const REFRESH_TOKEN_STORAGE_KEY = 'basepro.web.refresh_token'

let snapshot: AuthSnapshot = {
  isAuthenticated: false,
  accessToken: null,
  refreshToken: null,
  user: null,
}

const listeners = new Set<() => void>()

function notify() {
  for (const listener of listeners) {
    listener()
  }
}

export function getAuthSnapshot() {
  return snapshot
}

export function setAuthSnapshot(next: AuthSnapshot) {
  snapshot = next
  notify()
}

export function subscribeAuthSnapshot(listener: () => void) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function getPersistedRefreshToken() {
  if (typeof window === 'undefined') {
    return ''
  }
  return window.localStorage.getItem(REFRESH_TOKEN_STORAGE_KEY) ?? ''
}

export function persistRefreshToken(token: string) {
  if (typeof window === 'undefined') {
    return
  }
  if (!token.trim()) {
    window.localStorage.removeItem(REFRESH_TOKEN_STORAGE_KEY)
    return
  }
  window.localStorage.setItem(REFRESH_TOKEN_STORAGE_KEY, token)
}

export function clearPersistedRefreshToken() {
  if (typeof window === 'undefined') {
    return
  }
  window.localStorage.removeItem(REFRESH_TOKEN_STORAGE_KEY)
}

export function clearAuthSnapshot() {
  clearPersistedRefreshToken()
  setAuthSnapshot({
    isAuthenticated: false,
    accessToken: null,
    refreshToken: null,
    user: null,
  })
}
