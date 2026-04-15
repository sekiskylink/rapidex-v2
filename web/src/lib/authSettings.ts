export const AUTH_MODE_STORAGE_KEY = 'basepro.web.auth_mode'
export const API_TOKEN_STORAGE_KEY = 'basepro.web.api_token'

export type AuthMode = 'password' | 'api_token'

export interface StoredAuthSettings {
  authMode: AuthMode
  apiToken: string
}

function normalizeToken(value: string) {
  return value.trim()
}

function isAuthMode(value: unknown): value is AuthMode {
  return value === 'password' || value === 'api_token'
}

export function loadAuthSettings(): StoredAuthSettings {
  if (typeof window === 'undefined') {
    return { authMode: 'password', apiToken: '' }
  }

  const authMode = isAuthMode(window.localStorage.getItem(AUTH_MODE_STORAGE_KEY))
    ? (window.localStorage.getItem(AUTH_MODE_STORAGE_KEY) as AuthMode)
    : 'password'
  const apiToken = normalizeToken(window.localStorage.getItem(API_TOKEN_STORAGE_KEY) ?? '')

  return {
    authMode,
    apiToken: authMode === 'api_token' ? apiToken : '',
  }
}

export function saveAuthSettings(patch: Partial<StoredAuthSettings>) {
  if (typeof window === 'undefined') {
    return loadAuthSettings()
  }

  const current = loadAuthSettings()
  const nextAuthMode = patch.authMode ?? current.authMode
  const nextApiToken = normalizeToken(patch.apiToken ?? current.apiToken)

  window.localStorage.setItem(AUTH_MODE_STORAGE_KEY, nextAuthMode)
  if (nextAuthMode === 'api_token' && nextApiToken) {
    window.localStorage.setItem(API_TOKEN_STORAGE_KEY, nextApiToken)
  } else {
    window.localStorage.removeItem(API_TOKEN_STORAGE_KEY)
  }

  return {
    authMode: nextAuthMode,
    apiToken: nextAuthMode === 'api_token' ? nextApiToken : '',
  }
}

export function clearAuthSettings() {
  if (typeof window === 'undefined') {
    return
  }

  window.localStorage.removeItem(AUTH_MODE_STORAGE_KEY)
  window.localStorage.removeItem(API_TOKEN_STORAGE_KEY)
}

