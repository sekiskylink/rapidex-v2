import type { SessionPrincipal } from './auth/session'
import { hasAnyPermission, hasPermission } from './rbac/permissions'

export interface NavigationLeaf {
  label: string
  path: string
  key: string
  visible: boolean
}

export interface NavigationGroup {
  label: string
  key: string
  children: NavigationLeaf[]
  visible: boolean
}

function canAccessUsers(principal: SessionPrincipal | null | undefined) {
  return hasAnyPermission(principal, ['users.read', 'users.write'])
}

function canAccessRoles(principal: SessionPrincipal | null | undefined) {
  return hasAnyPermission(principal, ['users.read', 'users.write'])
}

function canAccessPermissions(principal: SessionPrincipal | null | undefined) {
  return hasAnyPermission(principal, ['users.read', 'users.write'])
}

function canAccessAudit(principal: SessionPrincipal | null | undefined) {
  return hasPermission(principal, 'audit.read')
}

export function buildNavigation(principal: SessionPrincipal | null | undefined) {
  const dashboard: NavigationLeaf = {
    key: 'dashboard',
    label: 'Dashboard',
    path: '/dashboard',
    visible: true,
  }

  const settings: NavigationLeaf = {
    key: 'settings',
    label: 'Settings',
    path: '/settings',
    visible: hasAnyPermission(principal, ['settings.read', 'settings.write']),
  }

  const administrationChildren: NavigationLeaf[] = [
    { key: 'users', label: 'Users', path: '/users', visible: canAccessUsers(principal) },
    { key: 'roles', label: 'Roles', path: '/roles', visible: canAccessRoles(principal) },
    { key: 'permissions', label: 'Permissions', path: '/permissions', visible: canAccessPermissions(principal) },
    { key: 'audit', label: 'Audit Log', path: '/audit', visible: canAccessAudit(principal) },
  ]

  const administration: NavigationGroup = {
    key: 'administration',
    label: 'Administration',
    children: administrationChildren.filter((item) => item.visible),
    visible: administrationChildren.some((item) => item.visible),
  }

  return {
    topLevel: [dashboard, settings].filter((item) => item.visible),
    administration,
  }
}

export function canAccessRoute(principal: SessionPrincipal | null | undefined, pathname: string) {
  if (pathname.startsWith('/users')) {
    return canAccessUsers(principal)
  }
  if (pathname.startsWith('/roles')) {
    return canAccessRoles(principal)
  }
  if (pathname.startsWith('/permissions')) {
    return canAccessPermissions(principal)
  }
  if (pathname.startsWith('/audit')) {
    return canAccessAudit(principal)
  }
  if (pathname.startsWith('/settings')) {
    return hasAnyPermission(principal, ['settings.read', 'settings.write'])
  }
  return true
}
