import { getAuthSnapshot, type AuthUser } from './auth/state'
import { authenticatedNavigationRegistry, canAccessNavigationPath } from './registry/navigation'

interface NavigationOptions {
  labels?: Record<string, string>
  showAdministration?: boolean
  showSukumad?: boolean
}

function resolveLabel(id: string, fallback: string, labels?: Record<string, string>) {
  const next = labels?.[id]?.trim()
  return next || fallback
}

export function buildNavigation(
  user: AuthUser | null | undefined = getAuthSnapshot().user,
  options: NavigationOptions = {},
) {
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
          label: resolveLabel(child.id, child.label, options.labels),
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
          label: resolveLabel(child.id, child.label, options.labels),
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
      label: resolveLabel(item.id, item.label, options.labels),
      path: item.path,
      visible: true,
    })
  }

  const administration = {
    key: 'administration',
    label: resolveLabel('administration', 'Administration', options.labels),
    children: administrationChildren,
    visible: administrationChildren.length > 0 && options.showAdministration !== false,
  }
  const sukumad = {
    key: 'sukumad',
    label: resolveLabel('sukumad', 'Sukumad', options.labels),
    children: sukumadChildren,
    visible: sukumadChildren.length > 0 && options.showSukumad !== false,
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
