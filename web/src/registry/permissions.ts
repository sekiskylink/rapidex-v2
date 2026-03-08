import { isModuleEnabled, type ModuleId } from './moduleEnablement'

export interface PermissionDefinition {
  key: string
  label: string
  description: string
  module: string
  category?: string
  moduleEnablementId?: ModuleId
}

export const permissionRegistry: readonly PermissionDefinition[] = [
  {
    key: 'users.read',
    label: 'Users: Read',
    description: 'View users and administration listings that depend on user read access.',
    module: 'users',
    category: 'Administration',
    moduleEnablementId: 'administration',
  },
  {
    key: 'users.write',
    label: 'Users: Write',
    description: 'Create and update users, roles, and role-permission mappings.',
    module: 'users',
    category: 'Administration',
    moduleEnablementId: 'administration',
  },
  {
    key: 'audit.read',
    label: 'Audit: Read',
    description: 'View audit log entries and related metadata.',
    module: 'audit',
    category: 'Administration',
    moduleEnablementId: 'administration',
  },
  {
    key: 'settings.read',
    label: 'Settings: Read',
    description: 'View platform settings such as login branding and configuration.',
    module: 'settings',
    category: 'Settings',
    moduleEnablementId: 'settings',
  },
  {
    key: 'settings.write',
    label: 'Settings: Write',
    description: 'Update platform settings such as login branding and configuration.',
    module: 'settings',
    category: 'Settings',
    moduleEnablementId: 'settings',
  },
  {
    key: 'api_tokens.read',
    label: 'API Tokens: Read',
    description: 'View API token records.',
    module: 'api_tokens',
    category: 'Administration',
    moduleEnablementId: 'administration',
  },
  {
    key: 'api_tokens.write',
    label: 'API Tokens: Write',
    description: 'Create and revoke API tokens.',
    module: 'api_tokens',
    category: 'Administration',
    moduleEnablementId: 'administration',
  },
]

export type PermissionKey = (typeof permissionRegistry)[number]['key']

const permissionMap = new Map<string, PermissionDefinition>(permissionRegistry.map((definition) => [definition.key, definition]))

export function getPermissionDefinition(key: string) {
  return permissionMap.get(key)
}

export function isPermissionDefinitionEnabled(key: string) {
  const definition = getPermissionDefinition(key)
  if (!definition?.moduleEnablementId) {
    return true
  }
  return isModuleEnabled(definition.moduleEnablementId)
}
