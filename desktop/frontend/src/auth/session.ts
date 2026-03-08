import type { SettingsStore } from '../settings/types'
import { resetEffectiveModuleEnablement } from '../registry/moduleEnablement'
import { clearBootstrap } from '../bootstrap/state'

interface SessionData {
  accessToken: string
  refreshToken: string
  expiresAt: number
}

interface SessionState {
  accessToken?: string
  expiresAt?: number
  principal?: SessionPrincipal
}

export interface SessionPrincipal {
  id: number
  username: string
  roles: string[]
  permissions: string[]
}

type SessionExpiredReason = 'expired' | 'network'

const state: SessionState = {}

let settingsStoreRef: SettingsStore | null = null
let sessionExpiryHandler: ((reason: SessionExpiredReason) => void) | null = null
const authStateListeners = new Set<() => void>()

function notifyAuthStateChange() {
  for (const listener of authStateListeners) {
    listener()
  }
}

export function configureSessionStorage(settingsStore: SettingsStore) {
  settingsStoreRef = settingsStore
}

export function onSessionExpired(handler: (reason: SessionExpiredReason) => void) {
  sessionExpiryHandler = handler
  return () => {
    if (sessionExpiryHandler === handler) {
      sessionExpiryHandler = null
    }
  }
}

export async function setSession(session: SessionData) {
  state.accessToken = session.accessToken
  state.expiresAt = session.expiresAt
  if (settingsStoreRef) {
    await settingsStoreRef.saveSettings({ refreshToken: session.refreshToken })
  }
  notifyAuthStateChange()
}

export async function clearSession() {
  state.accessToken = undefined
  state.expiresAt = undefined
  state.principal = undefined
  resetEffectiveModuleEnablement()
  clearBootstrap()
  if (settingsStoreRef) {
    await settingsStoreRef.saveSettings({ refreshToken: '' })
  }
  notifyAuthStateChange()
}

export function isAuthenticated() {
  return Boolean(state.accessToken)
}

export function getAccessToken() {
  return state.accessToken
}

export function getAccessTokenExpiresAt() {
  return state.expiresAt
}

export function setSessionPrincipal(principal: SessionPrincipal) {
  state.principal = {
    id: principal.id,
    username: principal.username,
    roles: [...principal.roles],
    permissions: [...principal.permissions],
  }
  notifyAuthStateChange()
}

export function getSessionPrincipal() {
  return state.principal
}

export function can(permission: string) {
  const principal = state.principal
  if (!principal) {
    return false
  }
  return principal.permissions.includes(permission)
}

export function subscribeAuthState(listener: () => void) {
  authStateListeners.add(listener)
  return () => {
    authStateListeners.delete(listener)
  }
}

export async function getPersistedRefreshToken() {
  if (!settingsStoreRef) {
    return ''
  }
  const settings = await settingsStoreRef.loadSettings()
  return settings.refreshToken?.trim() ?? ''
}

export async function handleSessionExpiry() {
  await clearSession()
  sessionExpiryHandler?.('expired')
}

export function handleNetworkUnreachable() {
  sessionExpiryHandler?.('network')
}
