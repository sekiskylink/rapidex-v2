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
  {
    id: 'servers',
    label: 'Servers',
    navGroup: 'dashboard',
    basePath: '/servers',
    permissions: ['servers.read', 'servers.write'],
    navItems: ['servers'],
    metadata: {
      description: 'Sukumad integration server placeholders.',
    },
  },
  {
    id: 'requests',
    label: 'Requests',
    navGroup: 'dashboard',
    basePath: '/requests',
    permissions: ['requests.read', 'requests.write'],
    navItems: ['requests'],
    metadata: {
      description: 'Sukumad request placeholders.',
    },
  },
  {
    id: 'deliveries',
    label: 'Deliveries',
    navGroup: 'dashboard',
    basePath: '/deliveries',
    permissions: ['deliveries.read', 'deliveries.write'],
    navItems: ['deliveries'],
    metadata: {
      description: 'Sukumad delivery placeholders.',
    },
  },
  {
    id: 'jobs',
    label: 'Jobs',
    navGroup: 'dashboard',
    basePath: '/jobs',
    permissions: ['jobs.read', 'jobs.write'],
    navItems: ['jobs'],
    metadata: {
      description: 'Sukumad worker job placeholders.',
    },
  },
  {
    id: 'observability',
    label: 'Observability',
    navGroup: 'dashboard',
    basePath: '/observability',
    permissions: ['observability.read'],
    navItems: ['observability'],
    metadata: {
      description: 'Sukumad observability placeholders.',
    },
  },
] as const satisfies readonly ModuleDefinition[]
