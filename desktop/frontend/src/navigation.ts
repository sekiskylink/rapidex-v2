import type { SessionPrincipal } from './auth/session'
import { isNavigationItemEnabled } from './registry/moduleEnablement'
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

interface NavigationOptions {
  labels?: Record<string, string>
  showAdministration?: boolean
  showSukumad?: boolean
}

function resolveLabel(id: string, fallback: string, labels?: Record<string, string>) {
  const next = labels?.[id]?.trim()
  return next || fallback
}

export function buildNavigation(principal: SessionPrincipal | null | undefined, options: NavigationOptions = {}) {
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
        const visible = canAccessNavigationPath(principal, child.path)
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

    const visible = canAccessNavigationPath(principal, item.path)
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

  const administration: NavigationGroup = {
    key: 'administration',
    label: resolveLabel('administration', 'Administration', options.labels),
    children: administrationChildren,
    visible:
      administrationChildren.length > 0 &&
      options.showAdministration !== false &&
      isNavigationItemEnabled('administration'),
  }
  const sukumad: NavigationGroup = {
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

export function canAccessRoute(principal: SessionPrincipal | null | undefined, pathname: string) {
  return canAccessNavigationPath(principal, pathname)
}
