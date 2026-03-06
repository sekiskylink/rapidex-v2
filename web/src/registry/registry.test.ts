import { describe, expect, it } from 'vitest'
import type { AuthUser } from '../auth/state'
import { canAccessNavigationPath, authenticatedNavigationRegistry } from './navigation'
import { moduleRegistry } from './modules'
import { permissionRegistry } from './permissions'

function userWith(permissions: string[]): AuthUser {
  return {
    id: 1,
    username: 'tester',
    roles: ['Admin'],
    permissions,
  }
}

describe('web registries', () => {
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
    const settingsOnly = userWith(['settings.read'])
    expect(canAccessNavigationPath('/settings', settingsOnly)).toBe(true)
    expect(canAccessNavigationPath('/users', settingsOnly)).toBe(false)

    const usersReader = userWith(['users.read'])
    expect(canAccessNavigationPath('/users', usersReader)).toBe(true)
    expect(canAccessNavigationPath('/audit', usersReader)).toBe(false)
  })
})

