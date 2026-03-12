import { describe, expect, it } from 'vitest'
import type { AuthUser } from '../auth/state'
import { applyEffectiveModuleEnablement, isPathModuleEnabled, moduleEnablementRegistry, resetEffectiveModuleEnablement } from './moduleEnablement'
import { canAccessNavigationPath, authenticatedNavigationRegistry } from './navigation'
import { moduleRegistry } from './modules'
import { permissionRegistry } from './permissions'

function userWith(permissions: string[], roles: string[] = ['Admin']): AuthUser {
  return {
    id: 1,
    username: 'tester',
    roles,
    permissions,
  }
}

describe('web registries', () => {
  it('defines module enablement defaults for baseline platform and Sukumad modules', () => {
    expect(moduleEnablementRegistry.map((item) => item.moduleId)).toEqual([
      'dashboard',
      'administration',
      'settings',
      'servers',
      'requests',
      'deliveries',
      'jobs',
      'observability',
    ])
    expect(moduleEnablementRegistry.every((item) => item.enabledByDefault)).toBe(true)
  })

  it('defines unique permission keys and includes platform and Sukumad permissions', () => {
    const keys = permissionRegistry.map((item) => item.key)
    expect(new Set(keys).size).toBe(keys.length)
    expect(keys).toEqual(
      expect.arrayContaining([
        'users.read',
        'users.write',
        'audit.read',
        'settings.read',
        'settings.write',
        'servers.read',
        'requests.read',
        'deliveries.read',
        'jobs.read',
        'observability.read',
      ]),
    )
  })

  it('defines baseline modules for platform and Sukumad placeholders', () => {
    const moduleIds = moduleRegistry.map((item) => item.id)
    expect(moduleIds).toEqual([
      'dashboard',
      'administration',
      'settings',
      'servers',
      'requests',
      'deliveries',
      'jobs',
      'observability',
    ])
    expect(moduleRegistry.find((item) => item.id === 'administration')?.navItems).toEqual([
      'users',
      'roles',
      'permissions',
      'audit',
    ])
  })

  it('defines grouped navigation with administration and Sukumad children', () => {
    const admin = authenticatedNavigationRegistry.find((item) => item.id === 'administration')
    expect(admin?.children?.map((item) => item.id)).toEqual(['users', 'roles', 'permissions', 'audit'])
    const sukumad = authenticatedNavigationRegistry.find((item) => item.id === 'sukumad')
    expect(sukumad?.children?.map((item) => item.id)).toEqual(['servers', 'requests', 'deliveries', 'jobs', 'observability'])
  })

  it('enforces navigation access from required permissions', () => {
    const settingsOnly = userWith(['settings.read'], ['Staff'])
    expect(canAccessNavigationPath('/settings', settingsOnly)).toBe(false)
    const settingsWriter = userWith(['settings.write'], ['Staff'])
    expect(canAccessNavigationPath('/settings', settingsWriter)).toBe(true)
    expect(canAccessNavigationPath('/users', settingsOnly)).toBe(false)

    const usersReader = userWith(['users.read'])
    expect(canAccessNavigationPath('/users', usersReader)).toBe(true)
    expect(canAccessNavigationPath('/audit', usersReader)).toBe(false)

    const serversReader = userWith(['servers.read'], ['Staff'])
    expect(canAccessNavigationPath('/servers', serversReader)).toBe(true)
    expect(canAccessNavigationPath('/observability', serversReader)).toBe(false)
  })

  it('blocks module paths when effective config disables a module', () => {
    resetEffectiveModuleEnablement()
    applyEffectiveModuleEnablement({
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
    })

    expect(isPathModuleEnabled('/users')).toBe(false)
    expect(canAccessNavigationPath('/users', userWith(['users.read']))).toBe(false)
    expect(isPathModuleEnabled('/servers')).toBe(true)
    resetEffectiveModuleEnablement()
  })
})
