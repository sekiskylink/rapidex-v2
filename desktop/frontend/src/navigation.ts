import type { SessionPrincipal } from './auth/session'
import { isNavigationItemEnabled } from './registry/moduleEnablement'
import {
  authenticatedNavigationRegistry,
  canAccessNavigationPath,
  type NavigationDefinition,
} from './registry/navigation'

export interface NavigationLeaf {
  label: string
  path: string
  key: string
  visible: boolean
}

export interface NavigationBranch {
  label: string
  key: string
  children: NavigationNode[]
  visible: boolean
  path?: string
}

export type NavigationNode = NavigationLeaf | NavigationBranch

export interface NavigationGroup {
  label: string
  key: string
  children: NavigationNode[]
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
  const administrationChildren: NavigationNode[] = []
  const sukumadChildren: NavigationNode[] = []

  const mapDefinition = (item: NavigationDefinition): NavigationNode | null => {
    if (item.children && item.children.length > 0) {
      const children = item.children
        .map((child) => mapDefinition(child))
        .filter((child): child is NavigationNode => child !== null)

      if (children.length === 0) {
        return null
      }

      return {
        key: item.id,
        label: resolveLabel(item.id, item.label, options.labels),
        path: item.path,
        visible: true,
        children,
      }
    }

    if (!item.path) {
      return null
    }

    const visible = canAccessNavigationPath(principal, item.path)
    if (!visible) {
      return null
    }

    return {
      key: item.id,
      label: resolveLabel(item.id, item.label, options.labels),
      path: item.path,
      visible: true,
    }
  }

  for (const item of authenticatedNavigationRegistry) {
    if (item.id === 'administration') {
      for (const child of item.children ?? []) {
        const next = mapDefinition(child)
        if (next) {
          administrationChildren.push(next)
        }
      }
      continue
    }
    if (item.id === 'sukumad') {
      for (const child of item.children ?? []) {
        const next = mapDefinition(child)
        if (next) {
          sukumadChildren.push(next)
        }
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
