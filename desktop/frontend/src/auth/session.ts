import type { SettingsStore } from '../settings/types'

interface SessionData {
  accessToken: string
  refreshToken: string
  expiresAt: number
}

interface SessionState {
  accessToken?: string
  expiresAt?: number
}

type SessionExpiredReason = 'expired' | 'network'

const state: SessionState = {}

let settingsStoreRef: SettingsStore | null = null
let sessionExpiryHandler: ((reason: SessionExpiredReason) => void) | null = null

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
}

export async function clearSession() {
  state.accessToken = undefined
  state.expiresAt = undefined
  if (settingsStoreRef) {
    await settingsStoreRef.saveSettings({ refreshToken: '' })
  }
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
