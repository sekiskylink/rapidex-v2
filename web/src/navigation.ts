import { getAuthSnapshot, type AuthUser } from './auth/state'

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

function normalize(value: string) {
  return value.trim().toLowerCase()
}

function hasPermission(user: AuthUser | null | undefined, permission: string) {
  const target = normalize(permission)
  if (!target || !user) {
    return false
  }
  return user.permissions.some((candidate) => normalize(candidate) === target)
}

function hasAnyPermission(user: AuthUser | null | undefined, permissions: string[]) {
  return permissions.some((permission) => hasPermission(user, permission))
}

function canAccessUsers(user: AuthUser | null | undefined) {
  return hasAnyPermission(user, ['users.read', 'users.write'])
}

function canAccessRoles(user: AuthUser | null | undefined) {
  return hasAnyPermission(user, ['users.read', 'users.write'])
}

function canAccessPermissions(user: AuthUser | null | undefined) {
  return hasAnyPermission(user, ['users.read', 'users.write'])
}

function canAccessAudit(user: AuthUser | null | undefined) {
  return hasPermission(user, 'audit.read')
}

export function buildNavigation(user: AuthUser | null | undefined = getAuthSnapshot().user) {
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
    visible: hasAnyPermission(user, ['settings.read', 'settings.write']),
  }

  const administrationChildren: NavigationLeaf[] = [
    { key: 'users', label: 'Users', path: '/users', visible: canAccessUsers(user) },
    { key: 'roles', label: 'Roles', path: '/roles', visible: canAccessRoles(user) },
    { key: 'permissions', label: 'Permissions', path: '/permissions', visible: canAccessPermissions(user) },
    { key: 'audit', label: 'Audit Log', path: '/audit', visible: canAccessAudit(user) },
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

export function canAccessRoute(pathname: string, user: AuthUser | null | undefined = getAuthSnapshot().user) {
  if (pathname.startsWith('/users')) {
    return canAccessUsers(user)
  }
  if (pathname.startsWith('/roles')) {
    return canAccessRoles(user)
  }
  if (pathname.startsWith('/permissions')) {
    return canAccessPermissions(user)
  }
  if (pathname.startsWith('/audit')) {
    return canAccessAudit(user)
  }
  if (pathname.startsWith('/settings')) {
    return hasAnyPermission(user, ['settings.read', 'settings.write'])
  }
  return true
}
