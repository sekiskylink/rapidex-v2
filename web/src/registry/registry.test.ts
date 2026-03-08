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
  it('defines module enablement defaults for baseline platform modules', () => {
    expect(moduleEnablementRegistry.map((item) => item.moduleId)).toEqual(['dashboard', 'administration', 'settings'])
    expect(moduleEnablementRegistry.every((item) => item.enabledByDefault)).toBe(true)
  })

  it('defines unique permission keys and includes baseline platform permissions', () => {
    const keys = permissionRegistry.map((item) => item.key)
    expect(new Set(keys).size).toBe(keys.length)
    expect(keys).toEqual(
      expect.arrayContaining(['users.read', 'users.write', 'audit.read', 'settings.read', 'settings.write']),
    )
  })

  it('defines baseline modules for dashboard, administration, and settings', () => {
    const moduleIds = moduleRegistry.map((item) => item.id)
    expect(moduleIds).toEqual(['dashboard', 'administration', 'settings'])
    expect(moduleRegistry.find((item) => item.id === 'administration')?.navItems).toEqual([
      'users',
      'roles',
      'permissions',
      'audit',
    ])
  })

  it('defines grouped navigation with administration children', () => {
    const admin = authenticatedNavigationRegistry.find((item) => item.id === 'administration')
    expect(admin?.children?.map((item) => item.id)).toEqual(['users', 'roles', 'permissions', 'audit'])
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
    resetEffectiveModuleEnablement()
  })
})
