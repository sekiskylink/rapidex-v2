import type { AuthUser } from '../auth/state'
import { hasAdminRoleForUser, hasAnyPermissionForUser, hasPermissionForUser } from '../rbac/permissions'
import { isNavigationItemEnabled, isPathModuleEnabled } from './moduleEnablement'
import type { PermissionKey } from './permissions'

export type NavigationIconKey =
  | 'dashboard'
  | 'settings'
  | 'administration'
  | 'sukumad'
  | 'users'
  | 'roles'
  | 'permissions'
  | 'audit'
  | 'servers'
  | 'requests'
  | 'deliveries'
  | 'jobs'
  | 'observability'
export type NavigationGroupKey = 'dashboard' | 'administration' | 'sukumad' | 'settings'

export interface NavigationVisibilityContext {
  user: AuthUser | null | undefined
}

export interface NavigationDefinition {
  id: string
  label: string
  icon: NavigationIconKey
  path?: string
  group: NavigationGroupKey
  requiredPermissions?: readonly PermissionKey[]
  children?: readonly NavigationDefinition[]
  visibleWhen?: (ctx: NavigationVisibilityContext) => boolean
}

export const authenticatedNavigationRegistry: readonly NavigationDefinition[] = [
  {
    id: 'dashboard',
    label: 'Dashboard',
    icon: 'dashboard',
    path: '/dashboard',
    group: 'dashboard',
  },
  {
    id: 'administration',
    label: 'Administration',
    icon: 'administration',
    group: 'administration',
    children: [
      {
        id: 'users',
        label: 'Users',
        icon: 'users',
        path: '/users',
        group: 'administration',
        requiredPermissions: ['users.read', 'users.write'],
      },
      {
        id: 'roles',
        label: 'Roles',
        icon: 'roles',
        path: '/roles',
        group: 'administration',
        requiredPermissions: ['users.read', 'users.write'],
      },
      {
        id: 'permissions',
        label: 'Permissions',
        icon: 'permissions',
        path: '/permissions',
        group: 'administration',
        requiredPermissions: ['users.read', 'users.write'],
      },
      {
        id: 'audit',
        label: 'Audit Log',
        icon: 'audit',
        path: '/audit',
        group: 'administration',
        requiredPermissions: ['audit.read'],
      },
      {
        id: 'settings',
        label: 'Settings',
        icon: 'settings',
        path: '/settings',
        group: 'administration',
        visibleWhen: ({ user }) => hasAdminRoleForUser(user) || hasPermissionForUser(user, 'settings.write'),
      },
    ],
  },
  {
    id: 'sukumad',
    label: 'Sukumad',
    icon: 'sukumad',
    group: 'sukumad',
    children: [
      {
        id: 'servers',
        label: 'Servers',
        icon: 'servers',
        path: '/servers',
        group: 'sukumad',
        requiredPermissions: ['servers.read', 'servers.write'],
      },
      {
        id: 'requests',
        label: 'Requests',
        icon: 'requests',
        path: '/requests',
        group: 'sukumad',
        requiredPermissions: ['requests.read'],
      },
      {
        id: 'deliveries',
        label: 'Deliveries',
        icon: 'deliveries',
        path: '/deliveries',
        group: 'sukumad',
        requiredPermissions: ['deliveries.read', 'deliveries.write'],
      },
      {
        id: 'jobs',
        label: 'Jobs',
        icon: 'jobs',
        path: '/jobs',
        group: 'sukumad',
        requiredPermissions: ['jobs.read', 'jobs.write'],
      },
      {
        id: 'observability',
        label: 'Observability',
        icon: 'observability',
        path: '/observability',
        group: 'sukumad',
        requiredPermissions: ['observability.read'],
      },
    ],
  },
]

function isVisible(definition: NavigationDefinition, ctx: NavigationVisibilityContext) {
  if (!isNavigationItemEnabled(definition.id)) {
    return false
  }
  if (definition.path && !isPathModuleEnabled(definition.path)) {
    return false
  }
  if (definition.requiredPermissions && definition.requiredPermissions.length > 0) {
    if (!hasAnyPermissionForUser(ctx.user, definition.requiredPermissions)) {
      return false
    }
  }
  if (definition.visibleWhen) {
    return definition.visibleWhen(ctx)
  }
  return true
}

export function getRouteLabel(pathname: string) {
  const matches: string[] = []
  for (const item of authenticatedNavigationRegistry) {
    if (item.path && pathname.startsWith(item.path)) {
      matches.push(item.label)
    }
    if (item.children) {
      for (const child of item.children) {
        if (child.path && pathname.startsWith(child.path)) {
          matches.push(child.label)
        }
      }
    }
  }
  return matches[0] ?? 'BasePro'
}

export function canAccessNavigationPath(pathname: string, user: AuthUser | null | undefined) {
  if (!isPathModuleEnabled(pathname)) {
    return false
  }
  const ctx: NavigationVisibilityContext = { user }
  for (const item of authenticatedNavigationRegistry) {
    if (item.path && pathname.startsWith(item.path)) {
      return isVisible(item, ctx)
    }
    if (item.children) {
      for (const child of item.children) {
        if (child.path && pathname.startsWith(child.path)) {
          return isVisible(child, ctx)
        }
      }
    }
  }
  return true
}

export type RouteAccessState = 'allowed' | 'forbidden' | 'module-disabled'

export function getRouteAccessState(pathname: string, user: AuthUser | null | undefined): RouteAccessState {
  if (!isPathModuleEnabled(pathname)) {
    return 'module-disabled'
  }
  return canAccessNavigationPath(pathname, user) ? 'allowed' : 'forbidden'
}
