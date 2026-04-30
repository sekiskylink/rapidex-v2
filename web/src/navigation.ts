import { getAuthSnapshot, type AuthUser } from './auth/state'
import { isNavigationItemEnabled } from './registry/moduleEnablement'
import {
  authenticatedNavigationRegistry,
  canAccessNavigationPath,
  type NavigationDefinition,
} from './registry/navigation'

export interface NavigationLeaf {
  key: string
  label: string
  path: string
  visible: boolean
}

export interface NavigationBranch {
  key: string
  label: string
  visible: boolean
  path?: string
  children: NavigationNode[]
}

export type NavigationNode = NavigationLeaf | NavigationBranch

export interface NavigationSearchEntry {
  key: string
  label: string
  path: string
  visible: boolean
  breadcrumb: string
  searchText: string
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

export function buildNavigation(
  user: AuthUser | null | undefined = getAuthSnapshot().user,
  options: NavigationOptions = {},
) {
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
    if (!canAccessNavigationPath(item.path, user)) {
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
    visible:
      administrationChildren.length > 0 &&
      options.showAdministration !== false &&
      isNavigationItemEnabled('administration'),
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

export function getNavigationSearchEntries(
  user: AuthUser | null | undefined = getAuthSnapshot().user,
  options: NavigationOptions = {},
) {
  const entries: NavigationSearchEntry[] = []

  const collectEntries = (items: readonly NavigationDefinition[], parentLabels: string[] = []) => {
    for (const item of items) {
      const itemLabel = resolveLabel(item.id, item.label, options.labels)
      const nextParentLabels = item.path ? parentLabels : [...parentLabels, itemLabel]

      if (item.path && canAccessNavigationPath(item.path, user)) {
        const breadcrumb = parentLabels.join(' / ')
        const searchParts = [itemLabel, breadcrumb, item.path, item.id]
        entries.push({
          key: item.id,
          label: itemLabel,
          path: item.path,
          visible: true,
          breadcrumb,
          searchText: searchParts.filter(Boolean).join(' ').toLowerCase(),
        })
      }

      if (item.children) {
        collectEntries(item.children, nextParentLabels)
      }
    }
  }

  collectEntries(authenticatedNavigationRegistry)
  return entries
}
