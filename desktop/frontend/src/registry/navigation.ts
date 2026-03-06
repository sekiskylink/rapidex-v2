import type { SessionPrincipal } from '../auth/session'
import { hasAnyPermission } from '../rbac/permissions'
import type { PermissionKey } from './permissions'

export type NavigationIconKey = 'dashboard' | 'settings' | 'administration' | 'users' | 'roles' | 'permissions' | 'audit'
export type NavigationGroupKey = 'dashboard' | 'administration' | 'settings'

export interface NavigationVisibilityContext {
  principal: SessionPrincipal | null | undefined
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
    ],
  },
  {
    id: 'settings',
    label: 'Settings',
    icon: 'settings',
    path: '/settings',
    group: 'settings',
    requiredPermissions: ['settings.read', 'settings.write'],
  },
]

function isVisible(definition: NavigationDefinition, ctx: NavigationVisibilityContext) {
  if (definition.requiredPermissions && definition.requiredPermissions.length > 0) {
    if (!hasAnyPermission(ctx.principal, definition.requiredPermissions)) {
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

export function canAccessNavigationPath(principal: SessionPrincipal | null | undefined, pathname: string) {
  const ctx: NavigationVisibilityContext = { principal }
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
