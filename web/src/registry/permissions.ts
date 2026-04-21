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
  {
    key: 'servers.read',
    label: 'Servers: Read',
    description: 'View Sukumad integration servers.',
    module: 'servers',
    category: 'Sukumad',
    moduleEnablementId: 'servers',
  },
  {
    key: 'servers.write',
    label: 'Servers: Write',
    description: 'Create and update Sukumad integration servers.',
    module: 'servers',
    category: 'Sukumad',
    moduleEnablementId: 'servers',
  },
  {
    key: 'requests.read',
    label: 'Requests: Read',
    description: 'View Sukumad exchange requests.',
    module: 'requests',
    category: 'Sukumad',
    moduleEnablementId: 'requests',
  },
  {
    key: 'requests.write',
    label: 'Requests: Write',
    description: 'Create and update Sukumad exchange requests.',
    module: 'requests',
    category: 'Sukumad',
    moduleEnablementId: 'requests',
  },
  {
    key: 'deliveries.read',
    label: 'Deliveries: Read',
    description: 'View Sukumad delivery attempts.',
    module: 'deliveries',
    category: 'Sukumad',
    moduleEnablementId: 'deliveries',
  },
  {
    key: 'deliveries.write',
    label: 'Deliveries: Write',
    description: 'Manage Sukumad delivery attempts.',
    module: 'deliveries',
    category: 'Sukumad',
    moduleEnablementId: 'deliveries',
  },
  {
    key: 'jobs.read',
    label: 'Jobs: Read',
    description: 'View Sukumad worker jobs and queue state.',
    module: 'jobs',
    category: 'Sukumad',
    moduleEnablementId: 'jobs',
  },
  {
    key: 'jobs.write',
    label: 'Jobs: Write',
    description: 'Manage Sukumad worker jobs and queue state.',
    module: 'jobs',
    category: 'Sukumad',
    moduleEnablementId: 'jobs',
  },
  {
    key: 'scheduler.read',
    label: 'Scheduler: Read',
    description: 'View Sukumad scheduled jobs and run histories.',
    module: 'scheduler',
    category: 'Sukumad',
    moduleEnablementId: 'scheduler',
  },
  {
    key: 'scheduler.write',
    label: 'Scheduler: Write',
    description: 'Create, update, and trigger Sukumad scheduled jobs.',
    module: 'scheduler',
    category: 'Sukumad',
    moduleEnablementId: 'scheduler',
  },
  {
    key: 'observability.read',
    label: 'Observability: Read',
    description: 'View Sukumad observability dashboards and traces.',
    module: 'observability',
    category: 'Sukumad',
    moduleEnablementId: 'observability',
  },
  {
    key: 'orgunits.read',
    label: 'Facilities: Read',
    description: 'View Rapidex organisation units and facility hierarchy.',
    module: 'orgunits',
    category: 'Rapidex',
    moduleEnablementId: 'orgunits',
  },
  {
    key: 'orgunits.write',
    label: 'Facilities: Write',
    description: 'Create and update Rapidex organisation units and facilities.',
    module: 'orgunits',
    category: 'Rapidex',
    moduleEnablementId: 'orgunits',
  },
  {
    key: 'reporters.read',
    label: 'Reporters: Read',
    description: 'View Rapidex reporters and facility assignments.',
    module: 'reporters',
    category: 'Rapidex',
    moduleEnablementId: 'reporters',
  },
  {
    key: 'reporters.write',
    label: 'Reporters: Write',
    description: 'Create and update Rapidex reporters and facility assignments.',
    module: 'reporters',
    category: 'Rapidex',
    moduleEnablementId: 'reporters',
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
