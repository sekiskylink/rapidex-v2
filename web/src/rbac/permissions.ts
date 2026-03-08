import { getAuthSnapshot, type AuthUser } from '../auth/state'

function normalize(value: string) {
  return value.trim().toLowerCase()
}

export function hasRole(role: string) {
  return hasRoleForUser(getAuthSnapshot().user, role)
}

export function hasRoleForUser(user: AuthUser | null | undefined, role: string) {
  const target = normalize(role)
  if (!target) {
    return false
  }

  if (!user) {
    return false
  }

  return user.roles.some((candidate) => normalize(candidate) === target)
}

export function hasAdminRoleForUser(user: AuthUser | null | undefined) {
  return hasRoleForUser(user, 'admin')
}

export function hasPermission(permission: string) {
  return hasPermissionForUser(getAuthSnapshot().user, permission)
}

export function hasPermissionForUser(user: AuthUser | null | undefined, permission: string) {
  const target = normalize(permission)
  if (!target) {
    return false
  }

  if (!user) {
    return false
  }

  return user.permissions.some((candidate) => normalize(candidate) === target)
}

export function hasAnyPermissionForUser(user: AuthUser | null | undefined, permissions: readonly string[]) {
  return permissions.some((permission) => hasPermissionForUser(user, permission))
}
