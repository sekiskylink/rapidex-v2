import { describe, expect, it } from 'vitest'
import type { SessionPrincipal } from '../auth/session'
import { canAccessNavigationPath, authenticatedNavigationRegistry } from './navigation'
import { moduleRegistry } from './modules'
import { permissionRegistry } from './permissions'

function principalWith(permissions: string[]): SessionPrincipal {
  return {
    id: 1,
    username: 'tester',
    roles: ['Admin'],
    permissions,
  }
}

describe('desktop registries', () => {
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
    const settingsOnly = principalWith(['settings.read'])
    expect(canAccessNavigationPath(settingsOnly, '/settings')).toBe(true)
    expect(canAccessNavigationPath(settingsOnly, '/users')).toBe(false)

    const usersReader = principalWith(['users.read'])
    expect(canAccessNavigationPath(usersReader, '/users')).toBe(true)
    expect(canAccessNavigationPath(usersReader, '/audit')).toBe(false)
  })
})

