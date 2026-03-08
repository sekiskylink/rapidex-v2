import { beforeEach, describe, expect, it } from 'vitest'
import { isPathModuleEnabled, resetEffectiveModuleEnablement } from '../registry/moduleEnablement'
import { applyBootstrap, clearBootstrap, getBootstrapSnapshot, hydrateBootstrapFromCache, type BootstrapPayload } from './state'

const STORAGE_KEY = 'basepro.desktop.bootstrap.v1'

function payload(): BootstrapPayload {
  return {
    version: 1,
    generatedAt: '2026-03-08T00:00:00Z',
    app: {
      version: '1.0.0',
      commit: 'abc123',
      buildDate: '2026-03-08T00:00:00Z',
    },
    branding: {
      applicationDisplayName: 'Acme Desktop',
      loginImageUrl: null,
    },
    modules: [
      {
        moduleId: 'dashboard',
        flagKey: 'modules.dashboard.enabled',
        enabled: true,
        enabledByDefault: true,
        source: 'default',
      },
      {
        moduleId: 'administration',
        flagKey: 'modules.administration.enabled',
        enabled: false,
        enabledByDefault: true,
        source: 'config',
      },
      {
        moduleId: 'settings',
        flagKey: 'modules.settings.enabled',
        enabled: true,
        enabledByDefault: true,
        source: 'default',
      },
    ],
    capabilities: {
      canAccessSettings: true,
      settings: {
        canRead: true,
        canWrite: true,
      },
    },
    cache: {
      maxStaleSeconds: 300,
      schemaVersion: 1,
      cacheable: true,
      offlineSafePayload: true,
      containsSecrets: false,
    },
    principal: {
      type: 'user',
      userId: 1,
      username: 'admin',
      roles: ['Admin'],
      permissions: ['settings.write'],
    },
  }
}

describe('desktop bootstrap state', () => {
  beforeEach(() => {
    window.localStorage.clear()
    clearBootstrap()
    resetEffectiveModuleEnablement()
  })

  it('applies live bootstrap and persists it for future startup', () => {
    applyBootstrap(payload(), 'live')

    expect(getBootstrapSnapshot().source).toBe('live')
    expect(getBootstrapSnapshot().payload?.branding.applicationDisplayName).toBe('Acme Desktop')
    expect(isPathModuleEnabled('/users')).toBe(false)
    expect(window.localStorage.getItem(STORAGE_KEY)).toBeTruthy()
  })

  it('hydrates cached bootstrap and marks stale cache', () => {
    const cached = payload()
    cached.cache.maxStaleSeconds = 1

    window.localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        cachedAt: Date.now() - 10_000,
        payload: cached,
      }),
    )

    expect(hydrateBootstrapFromCache()).toBe(true)
    expect(getBootstrapSnapshot().source).toBe('cache')
    expect(getBootstrapSnapshot().stale).toBe(true)
    expect(isPathModuleEnabled('/users')).toBe(false)
  })

  it('clears bootstrap cache and restores default module enablement', () => {
    applyBootstrap(payload(), 'live')
    clearBootstrap()

    expect(getBootstrapSnapshot().payload).toBeNull()
    expect(getBootstrapSnapshot().source).toBe('none')
    expect(window.localStorage.getItem(STORAGE_KEY)).toBeNull()
    expect(isPathModuleEnabled('/users')).toBe(true)
  })
})
