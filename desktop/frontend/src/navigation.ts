import type { SessionPrincipal } from './auth/session'
import { authenticatedNavigationRegistry, canAccessNavigationPath } from './registry/navigation'

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

export function buildNavigation(principal: SessionPrincipal | null | undefined) {
  const topLevel: NavigationLeaf[] = []
  const administrationChildren: NavigationLeaf[] = []
  const sukumadChildren: NavigationLeaf[] = []

  for (const item of authenticatedNavigationRegistry) {
    if (item.id === 'administration') {
      for (const child of item.children ?? []) {
        if (!child.path) {
          continue
        }
        const visible = canAccessNavigationPath(principal, child.path)
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
        const visible = canAccessNavigationPath(principal, child.path)
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

    const visible = canAccessNavigationPath(principal, item.path)
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

  const administration: NavigationGroup = {
    key: 'administration',
    label: 'Administration',
    children: administrationChildren,
    visible: administrationChildren.length > 0,
  }
  const sukumad: NavigationGroup = {
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

export function canAccessRoute(principal: SessionPrincipal | null | undefined, pathname: string) {
  return canAccessNavigationPath(principal, pathname)
}
