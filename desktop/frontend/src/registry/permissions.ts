export interface PermissionDefinition {
  key: string
  label: string
  description: string
  module: string
  category?: string
}

export const permissionRegistry: readonly PermissionDefinition[] = [
  {
    key: 'users.read',
    label: 'Users: Read',
    description: 'View users and administration listings that depend on user read access.',
    module: 'users',
    category: 'Administration',
  },
  {
    key: 'users.write',
    label: 'Users: Write',
    description: 'Create and update users, roles, and role-permission mappings.',
    module: 'users',
    category: 'Administration',
  },
  {
    key: 'audit.read',
    label: 'Audit: Read',
    description: 'View audit log entries and related metadata.',
    module: 'audit',
    category: 'Administration',
  },
  {
    key: 'settings.read',
    label: 'Settings: Read',
    description: 'View platform settings such as login branding and configuration.',
    module: 'settings',
    category: 'Settings',
  },
  {
    key: 'settings.write',
    label: 'Settings: Write',
    description: 'Update platform settings such as login branding and configuration.',
    module: 'settings',
    category: 'Settings',
  },
  {
    key: 'api_tokens.read',
    label: 'API Tokens: Read',
    description: 'View API token records.',
    module: 'api_tokens',
    category: 'Administration',
  },
  {
    key: 'api_tokens.write',
    label: 'API Tokens: Write',
    description: 'Create and revoke API tokens.',
    module: 'api_tokens',
    category: 'Administration',
  },
]

export type PermissionKey = (typeof permissionRegistry)[number]['key']

const permissionMap = new Map<string, PermissionDefinition>(permissionRegistry.map((definition) => [definition.key, definition]))

export function getPermissionDefinition(key: string) {
  return permissionMap.get(key)
}
