import type { NavigationGroupKey } from './navigation'
import type { PermissionKey } from './permissions'

export interface ModuleDefinition {
  id: string
  label: string
  navGroup: NavigationGroupKey
  basePath: string
  permissions: readonly PermissionKey[]
  navItems: readonly string[]
  flags?: {
    hiddenFromNavigation?: boolean
  }
  metadata?: {
    description?: string
  }
}

export const moduleRegistry = [
  {
    id: 'dashboard',
    label: 'Dashboard',
    navGroup: 'dashboard',
    basePath: '/dashboard',
    permissions: [],
    navItems: ['dashboard'],
    metadata: {
      description: 'Authenticated landing page.',
    },
  },
  {
    id: 'administration',
    label: 'Administration',
    navGroup: 'administration',
    basePath: '/users',
    permissions: ['users.read', 'users.write', 'audit.read'],
    navItems: ['users', 'roles', 'permissions', 'audit'],
    metadata: {
      description: 'Core RBAC and audit administration pages.',
    },
  },
  {
    id: 'settings',
    label: 'Settings',
    navGroup: 'settings',
    basePath: '/settings',
    permissions: ['settings.read', 'settings.write'],
    navItems: ['settings'],
    metadata: {
      description: 'System and branding configuration.',
    },
  },
] as const satisfies readonly ModuleDefinition[]

