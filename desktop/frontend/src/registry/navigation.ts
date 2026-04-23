import type { SessionPrincipal } from '../auth/session'
import { hasAdminRole, hasAnyPermission, hasPermission } from '../rbac/permissions'
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
  | 'scheduler'
  | 'observability'
  | 'documentation'
  | 'orgunits'
  | 'reporters'
export type NavigationGroupKey = 'dashboard' | 'administration' | 'sukumad' | 'settings'

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
      {
        id: 'settings',
        label: 'Settings',
        icon: 'settings',
        group: 'administration',
        visibleWhen: ({ principal }) => hasAdminRole(principal) || hasPermission(principal, 'settings.write'),
        children: [
          {
            id: 'settings-general',
            label: 'General',
            icon: 'settings',
            path: '/settings/general',
            group: 'administration',
            requiredPermissions: ['settings.read', 'settings.write'],
          },
          {
            id: 'settings-branding',
            label: 'Branding',
            icon: 'settings',
            path: '/settings/branding',
            group: 'administration',
            requiredPermissions: ['settings.read', 'settings.write'],
          },
          {
            id: 'settings-modules',
            label: 'Modules',
            icon: 'settings',
            path: '/settings/modules',
            group: 'administration',
            requiredPermissions: ['settings.read', 'settings.write'],
          },
          {
            id: 'settings-integrations',
            label: 'Integrations',
            icon: 'settings',
            path: '/settings/integrations',
            group: 'administration',
            requiredPermissions: ['settings.read', 'settings.write'],
          },
          {
            id: 'settings-about',
            label: 'About',
            icon: 'settings',
            path: '/settings/about',
            group: 'administration',
            requiredPermissions: ['settings.read', 'settings.write'],
          },
        ],
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
        id: 'scheduler',
        label: 'Scheduler',
        icon: 'scheduler',
        path: '/scheduler',
        group: 'sukumad',
        requiredPermissions: ['scheduler.read', 'scheduler.write'],
      },
      {
        id: 'observability',
        label: 'Observability',
        icon: 'observability',
        path: '/observability',
        group: 'sukumad',
        requiredPermissions: ['observability.read'],
      },
      {
        id: 'documentation',
        label: 'Documentation',
        icon: 'documentation',
        path: '/documentation',
        group: 'sukumad',
      },
      {
        id: 'orgunits',
        label: 'Facilities',
        icon: 'orgunits',
        path: '/orgunits',
        group: 'sukumad',
        requiredPermissions: ['orgunits.read', 'orgunits.write'],
      },
      {
        id: 'reporters',
        label: 'Reporters',
        icon: 'reporters',
        path: '/reporters',
        group: 'sukumad',
        requiredPermissions: ['reporters.read', 'reporters.write'],
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
    if (!hasAnyPermission(ctx.principal, definition.requiredPermissions)) {
      return false
    }
  }
  if (definition.visibleWhen) {
    return definition.visibleWhen(ctx)
  }
  return true
}

function collectPathMatches(definition: NavigationDefinition, pathname: string, matches: string[]) {
  if (definition.path && pathname.startsWith(definition.path)) {
    matches.push(definition.label)
  }
  for (const child of definition.children ?? []) {
    collectPathMatches(child, pathname, matches)
  }
}

function findMatchingDefinition(
  definitions: readonly NavigationDefinition[],
  pathname: string,
): NavigationDefinition | null {
  for (const definition of definitions) {
    if (definition.path && pathname.startsWith(definition.path)) {
      return definition
    }
    if (definition.children) {
      const match = findMatchingDefinition(definition.children, pathname)
      if (match) {
        return match
      }
    }
  }
  return null
}

function canAccessDefinitionPath(
  definitions: readonly NavigationDefinition[],
  pathname: string,
  ctx: NavigationVisibilityContext,
  ancestorsVisible = true,
): boolean | null {
  for (const definition of definitions) {
    const currentVisible = ancestorsVisible && isVisible(definition, ctx)
    if (definition.path && pathname.startsWith(definition.path)) {
      return currentVisible
    }
    if (definition.children) {
      const childMatch = canAccessDefinitionPath(definition.children, pathname, ctx, currentVisible)
      if (childMatch !== null) {
        return childMatch
      }
    }
  }
  return null
}

export function getRouteLabel(pathname: string) {
  const matches: string[] = []
  for (const item of authenticatedNavigationRegistry) {
    collectPathMatches(item, pathname, matches)
  }
  return matches[0] ?? 'RapidEx'
}

export function canAccessNavigationPath(principal: SessionPrincipal | null | undefined, pathname: string) {
  if (!isPathModuleEnabled(pathname)) {
    return false
  }
  const ctx: NavigationVisibilityContext = { principal }
  const match = canAccessDefinitionPath(authenticatedNavigationRegistry, pathname, ctx)
  if (match === null) {
    return true
  }
  return match
}

export type RouteAccessState = 'allowed' | 'forbidden' | 'module-disabled'

export function getRouteAccessState(principal: SessionPrincipal | null | undefined, pathname: string): RouteAccessState {
  if (!isPathModuleEnabled(pathname)) {
    return 'module-disabled'
  }
  return canAccessNavigationPath(principal, pathname) ? 'allowed' : 'forbidden'
}
