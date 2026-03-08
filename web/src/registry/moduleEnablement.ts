import { useSyncExternalStore } from 'react'
import { moduleRegistry } from './modules'

export type ModuleId = (typeof moduleRegistry)[number]['id']
export type ModuleEnablementScope = 'backend' | 'desktop' | 'web' | 'full-stack'

export interface ModuleEnablementDefinition {
  moduleId: ModuleId
  flagKey: string
  enabledByDefault: boolean
  description?: string
  scope?: ModuleEnablementScope
  experimental?: boolean
}

export interface ModuleEffectiveConfig {
  moduleId: ModuleId
  flagKey: string
  enabled: boolean
  enabledByDefault: boolean
  description?: string
  scope?: ModuleEnablementScope
  experimental?: boolean
  source: 'default' | 'config'
}

export interface ModuleEnablementApiResponse {
  modules: ModuleEffectiveConfig[]
}

export const moduleEnablementRegistry = [
  {
    moduleId: 'dashboard',
    flagKey: 'modules.dashboard.enabled',
    enabledByDefault: true,
    description: 'Authenticated dashboard shell entry.',
    scope: 'full-stack',
  },
  {
    moduleId: 'administration',
    flagKey: 'modules.administration.enabled',
    enabledByDefault: true,
    description: 'RBAC and audit administration surfaces.',
    scope: 'full-stack',
  },
  {
    moduleId: 'settings',
    flagKey: 'modules.settings.enabled',
    enabledByDefault: true,
    description: 'System configuration and branding surfaces.',
    scope: 'full-stack',
  },
] as const satisfies readonly ModuleEnablementDefinition[]

interface ModuleEnablementSnapshot {
  byModuleId: Record<ModuleId, boolean>
}

const listeners = new Set<() => void>()

function defaultByModule(): Record<ModuleId, boolean> {
  const next = {} as Record<ModuleId, boolean>
  for (const definition of moduleEnablementRegistry) {
    next[definition.moduleId] = definition.enabledByDefault
  }
  return next
}

let snapshot: ModuleEnablementSnapshot = {
  byModuleId: defaultByModule(),
}

export function resetEffectiveModuleEnablement() {
  snapshot = {
    byModuleId: defaultByModule(),
  }
  for (const listener of listeners) {
    listener()
  }
}

export function applyEffectiveModuleEnablement(payload: ModuleEnablementApiResponse | null | undefined) {
  if (!payload?.modules || payload.modules.length === 0) {
    resetEffectiveModuleEnablement()
    return
  }

  const byModuleId = defaultByModule()
  for (const item of payload.modules) {
    if (item.moduleId in byModuleId) {
      byModuleId[item.moduleId as ModuleId] = item.enabled
    }
  }

  snapshot = { byModuleId }
  for (const listener of listeners) {
    listener()
  }
}

export function isModuleEnabled(moduleId: ModuleId, moduleSnapshot: ModuleEnablementSnapshot = snapshot) {
  return moduleSnapshot.byModuleId[moduleId] ?? true
}

export function findModuleForPath(pathname: string): ModuleId | null {
  const normalizedPath = pathname.trim()
  if (!normalizedPath) {
    return null
  }

  let winner: ModuleId | null = null
  let bestLength = -1

  for (const module of moduleRegistry) {
    if (normalizedPath.startsWith(module.basePath) && module.basePath.length > bestLength) {
      winner = module.id
      bestLength = module.basePath.length
    }
  }

  return winner
}

export function isPathModuleEnabled(pathname: string, moduleSnapshot: ModuleEnablementSnapshot = snapshot) {
  const moduleId = findModuleForPath(pathname)
  if (!moduleId) {
    return true
  }
  return isModuleEnabled(moduleId, moduleSnapshot)
}

export function isNavigationItemEnabled(navItemId: string, moduleSnapshot: ModuleEnablementSnapshot = snapshot) {
  for (const module of moduleRegistry) {
    if ((module.navItems as readonly string[]).includes(navItemId)) {
      return isModuleEnabled(module.id, moduleSnapshot)
    }
  }
  return true
}

export function getModuleEnablementSnapshot() {
  return snapshot
}

export function subscribeModuleEnablement(listener: () => void) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function useModuleEnablementSnapshot() {
  return useSyncExternalStore(subscribeModuleEnablement, getModuleEnablementSnapshot, getModuleEnablementSnapshot)
}
