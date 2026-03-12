import { getAuthSnapshot, type AuthUser } from './auth/state'
import { authenticatedNavigationRegistry, canAccessNavigationPath } from './registry/navigation'

export function buildNavigation(user: AuthUser | null | undefined = getAuthSnapshot().user) {
  const topLevel: Array<{ key: string; label: string; path: string; visible: boolean }> = []
  const administrationChildren: Array<{ key: string; label: string; path: string; visible: boolean }> = []
  const sukumadChildren: Array<{ key: string; label: string; path: string; visible: boolean }> = []

  for (const item of authenticatedNavigationRegistry) {
    if (item.id === 'administration') {
      for (const child of item.children ?? []) {
        if (!child.path) {
          continue
        }
        const visible = canAccessNavigationPath(child.path, user)
        if (!visible) {
          continue
        }
        administrationChildren.push({
          key: child.id,
          label: child.label,
          path: child.path,
          visible: true,
        })
      }
      continue
    }
    if (item.id === 'sukumad') {
      for (const child of item.children ?? []) {
        if (!child.path) {
          continue
        }
        const visible = canAccessNavigationPath(child.path, user)
        if (!visible) {
          continue
        }
        sukumadChildren.push({
          key: child.id,
          label: child.label,
          path: child.path,
          visible: true,
        })
      }
      continue
    }
    if (!item.path) {
      continue
    }
    const visible = canAccessNavigationPath(item.path, user)
    if (!visible) {
      continue
    }
    topLevel.push({
      key: item.id,
      label: item.label,
      path: item.path,
      visible: true,
    })
  }

  const administration = {
    key: 'administration',
    label: 'Administration',
    children: administrationChildren,
    visible: administrationChildren.length > 0,
  }
  const sukumad = {
    key: 'sukumad',
    label: 'Sukumad',
    children: sukumadChildren,
    visible: sukumadChildren.length > 0,
  }

  return {
    topLevel,
    administration,
    sukumad,
  }
}

export function canAccessRoute(pathname: string, user: AuthUser | null | undefined = getAuthSnapshot().user) {
  return canAccessNavigationPath(pathname, user)
}
