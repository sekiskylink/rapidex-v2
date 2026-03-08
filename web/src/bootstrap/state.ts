import { useSyncExternalStore } from 'react'
import { applyEffectiveModuleEnablement, resetEffectiveModuleEnablement, type ModuleEffectiveConfig } from '../registry/moduleEnablement'

const BOOTSTRAP_CACHE_STORAGE_KEY = 'basepro.web.bootstrap.v1'

interface CachedBootstrapEnvelope {
  cachedAt: number
  payload: BootstrapPayload
}

export interface BootstrapAppMetadata {
  version: string
  commit: string
  buildDate: string
}

export interface BootstrapBranding {
  applicationDisplayName?: string
  loginImageUrl?: string | null
  loginImageAssetPath?: string | null
  imageConfigured?: boolean
}

export interface BootstrapSettingsCapabilities {
  canRead?: boolean
  canWrite?: boolean
  writeRequiresAny?: string[]
  enforcedByApi?: boolean
  authorization?: string
  moduleRequired?: string
}

export interface BootstrapCapabilitySummary {
  canAccessSettings?: boolean
  settings?: BootstrapSettingsCapabilities
}

export interface BootstrapPrincipalSummary {
  type: string
  userId?: number
  username?: string
  roles?: string[]
  permissions?: string[]
}

export interface BootstrapCacheHints {
  maxStaleSeconds?: number
  schemaVersion?: number
  cacheable?: boolean
  offlineSafePayload?: boolean
  containsSecrets?: boolean
}

export interface BootstrapPayload {
  version: number
  generatedAt: string
  app: BootstrapAppMetadata
  branding: BootstrapBranding
  modules: ModuleEffectiveConfig[]
  capabilities: BootstrapCapabilitySummary
  cache: BootstrapCacheHints
  principal?: BootstrapPrincipalSummary
}

export interface BootstrapSnapshot {
  payload: BootstrapPayload | null
  source: 'none' | 'cache' | 'live'
  stale: boolean
}

const listeners = new Set<() => void>()

let snapshot: BootstrapSnapshot = {
  payload: null,
  source: 'none',
  stale: false,
}

function notify() {
  for (const listener of listeners) {
    listener()
  }
}

function isBootstrapPayload(value: unknown): value is BootstrapPayload {
  if (!value || typeof value !== 'object') {
    return false
  }
  const candidate = value as Partial<BootstrapPayload>
  return typeof candidate.version === 'number' && Array.isArray(candidate.modules)
}

function readCachedEnvelope(): CachedBootstrapEnvelope | null {
  if (typeof window === 'undefined') {
    return null
  }

  const raw = window.localStorage.getItem(BOOTSTRAP_CACHE_STORAGE_KEY)
  if (!raw) {
    return null
  }

  try {
    const parsed = JSON.parse(raw) as Partial<CachedBootstrapEnvelope>
    if (!parsed || typeof parsed !== 'object') {
      return null
    }
    if (typeof parsed.cachedAt !== 'number' || !isBootstrapPayload(parsed.payload)) {
      return null
    }
    return {
      cachedAt: parsed.cachedAt,
      payload: parsed.payload,
    }
  } catch {
    return null
  }
}

function writeCachedEnvelope(payload: BootstrapPayload) {
  if (typeof window === 'undefined') {
    return
  }
  const envelope: CachedBootstrapEnvelope = {
    cachedAt: Date.now(),
    payload,
  }
  window.localStorage.setItem(BOOTSTRAP_CACHE_STORAGE_KEY, JSON.stringify(envelope))
}

function clearCachedEnvelope() {
  if (typeof window === 'undefined') {
    return
  }
  window.localStorage.removeItem(BOOTSTRAP_CACHE_STORAGE_KEY)
}

function isCacheStale(envelope: CachedBootstrapEnvelope) {
  const maxStaleSeconds = Math.max(0, envelope.payload.cache.maxStaleSeconds ?? 0)
  if (!maxStaleSeconds) {
    return false
  }
  return Date.now() - envelope.cachedAt > maxStaleSeconds * 1000
}

function applySnapshot(next: BootstrapSnapshot) {
  snapshot = next
  if (next.payload) {
    applyEffectiveModuleEnablement({ modules: next.payload.modules })
  } else {
    resetEffectiveModuleEnablement()
  }
  notify()
}

export function hydrateBootstrapFromCache() {
  const envelope = readCachedEnvelope()
  if (!envelope) {
    return false
  }
  applySnapshot({
    payload: envelope.payload,
    source: 'cache',
    stale: isCacheStale(envelope),
  })
  return true
}

export function applyBootstrap(payload: BootstrapPayload, source: 'cache' | 'live' = 'live') {
  applySnapshot({
    payload,
    source,
    stale: false,
  })
  if (source === 'live') {
    writeCachedEnvelope(payload)
  }
}

export function clearBootstrap() {
  clearCachedEnvelope()
  applySnapshot({
    payload: null,
    source: 'none',
    stale: false,
  })
}

export function getBootstrapSnapshot() {
  return snapshot
}

export function subscribeBootstrap(listener: () => void) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function useBootstrapSnapshot() {
  return useSyncExternalStore(subscribeBootstrap, getBootstrapSnapshot, getBootstrapSnapshot)
}

