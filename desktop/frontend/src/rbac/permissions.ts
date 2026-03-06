import type { SessionPrincipal } from '../auth/session'

function normalize(value: string) {
  return value.trim().toLowerCase()
}

export function hasPermission(principal: SessionPrincipal | null | undefined, permission: string) {
  const target = normalize(permission)
  if (!target || !principal) {
    return false
  }
  return principal.permissions.some((candidate) => normalize(candidate) === target)
}

export function hasAnyPermission(principal: SessionPrincipal | null | undefined, permissions: readonly string[]) {
  return permissions.some((permission) => hasPermission(principal, permission))
}
