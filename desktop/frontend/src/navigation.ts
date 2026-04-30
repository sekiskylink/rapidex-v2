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

export interface NavigationSearchEntry {
  label: string
  key: string
  path: string
  visible: boolean
  breadcrumb: string
  searchText: string
}

interface RankedNavigationSearchEntry extends NavigationSearchEntry {
  rank: number
}

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

export function getNavigationSearchEntries(
  principal: SessionPrincipal | null | undefined,
  options: NavigationOptions = {},
) {
  const entries: NavigationSearchEntry[] = []

  const collectEntries = (items: readonly NavigationDefinition[], parentLabels: string[] = []) => {
    for (const item of items) {
      const itemLabel = resolveLabel(item.id, item.label, options.labels)
      const nextParentLabels = item.path ? parentLabels : [...parentLabels, itemLabel]

      if (item.path && canAccessNavigationPath(principal, item.path)) {
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

export function searchNavigationEntries(
  entries: readonly NavigationSearchEntry[],
  currentPath: string,
  query: string,
) {
  const normalizedQuery = query.trim().toLowerCase()

  const ranked = entries
    .map((entry): RankedNavigationSearchEntry | null => {
      let rank = 0

      if (normalizedQuery) {
        const label = entry.label.toLowerCase()
        const breadcrumb = entry.breadcrumb.toLowerCase()
        const path = entry.path.toLowerCase()

        if (label === normalizedQuery) {
          rank = 400
        } else if (label.startsWith(normalizedQuery)) {
          rank = 300
        } else if (breadcrumb.startsWith(normalizedQuery) || path.startsWith(normalizedQuery)) {
          rank = 220
        } else if (entry.searchText.includes(normalizedQuery)) {
          rank = 120
        } else {
          return null
        }
      }

      if (entry.path === currentPath) {
        rank += normalizedQuery ? 20 : 40
      }

      return {
        ...entry,
        rank,
      }
    })
    .filter((entry): entry is RankedNavigationSearchEntry => entry !== null)

  ranked.sort((left, right) => {
    if (right.rank !== left.rank) {
      return right.rank - left.rank
    }
    return left.label.localeCompare(right.label)
  })

  return ranked
}
