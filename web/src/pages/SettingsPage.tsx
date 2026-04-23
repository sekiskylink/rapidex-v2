import React from 'react'
import {
  Alert,
  Avatar,
  Box,
  Button,
  Chip,
  Divider,
  FormControl,
  FormControlLabel,
  MenuItem,
  Paper,
  Radio,
  RadioGroup,
  Select,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import { useNavigate } from '@tanstack/react-router'
import { useAuth } from '../auth/AuthProvider'
import { handleAppError } from '../errors/handleAppError'
import { apiRequest } from '../lib/api'
import { appName } from '../lib/env'
import { getApiBaseUrlOverride, setApiBaseUrlOverride } from '../lib/apiBaseUrl'
import { loadAuthSettings, saveAuthSettings, type AuthMode } from '../lib/authSettings'
import { useAppNotify } from '../notifications/facade'
import { hasPermission } from '../rbac/permissions'
import type { ModuleEffectiveConfig } from '../registry/moduleEnablement'
import { moduleRegistry } from '../registry/modules'
import { type UiThemeMode } from '../ui/preferences'
import {
  BrushRoundedIcon,
  HubRoundedIcon,
  InfoRoundedIcon,
  PaletteRoundedIcon,
  TuneRoundedIcon,
  WidgetsRoundedIcon,
} from '../ui/icons'
import { PalettePresetPicker } from '../ui/theme/PalettePresetPicker'
import { palettePresets } from '../ui/theme/presets'
import { useUiPreferences } from '../ui/theme/UiPreferencesProvider'

interface HealthResponse {
  status?: string
  version?: string
}

interface RuntimeConfigResponse {
  config: Record<string, unknown>
}

interface RapidProContactField {
  key: string
  label: string
  valueType?: string
}

interface RapidProReporterFieldMapping {
  sourceKey: string
  sourceLabel?: string
  rapidProFieldKey: string
}

interface RapidProReporterSyncValidation {
  isValid: boolean
  errors?: string[]
}

interface RapidProReporterSyncSettingsResponse {
  rapidProServerCode: string
  availableFields: RapidProContactField[]
  mappings: RapidProReporterFieldMapping[]
  lastFetchedAt?: string | null
  validation: RapidProReporterSyncValidation
}

interface CreateAPITokenResponse {
  id: number
  name: string
  prefix: string
  token: string
  expiresAt?: string | null
  permissions: string[]
}

type RuntimeConfigFormat = 'json' | 'yaml'

const navigationLabelFields = [
  { id: 'dashboard', label: 'Dashboard link' },
  { id: 'settings', label: 'Settings link' },
  { id: 'administration', label: 'Administration menu' },
  { id: 'users', label: 'Users link' },
  { id: 'roles', label: 'Roles link' },
  { id: 'permissions', label: 'Permissions link' },
  { id: 'audit', label: 'Audit link' },
  { id: 'sukumad', label: 'Sukumad menu' },
  { id: 'servers', label: 'Servers link' },
  { id: 'requests', label: 'Requests link' },
  { id: 'deliveries', label: 'Deliveries link' },
  { id: 'jobs', label: 'Jobs link' },
  { id: 'scheduler', label: 'Scheduler link' },
  { id: 'observability', label: 'Observability link' },
  { id: 'documentation', label: 'Documentation link' },
] as const

const rapidProReporterSourceOptions = [
  { key: 'name', label: 'Reporter Name' },
  { key: 'telephone', label: 'Telephone' },
  { key: 'whatsapp', label: 'WhatsApp' },
  { key: 'telegram', label: 'Telegram' },
  { key: 'reportingLocation', label: 'Reporting Location' },
  { key: 'facilityName', label: 'Facility Name' },
  { key: 'facilityUID', label: 'Facility UID' },
] as const

function isRapidProBuiltInField(field: RapidProContactField) {
  return (field.valueType ?? '').trim().toLowerCase() === 'builtin'
}

function getRapidProFieldOptionLabel(field: RapidProContactField) {
  return isRapidProBuiltInField(field) ? `${field.label} (Built-in)` : `${field.label} (Custom field)`
}

export type SettingsSection = 'general' | 'branding' | 'modules' | 'integrations' | 'about'

const settingsSections: Array<{ key: SettingsSection; label: string; description: string; path: string; anchorId: string }> = [
  {
    key: 'general',
    label: 'General',
    description: 'Manage local appearance and connection preferences for this browser profile.',
    path: '/settings/general',
    anchorId: 'settings-appearance',
  },
  {
    key: 'branding',
    label: 'Branding',
    description: 'Configure login branding used across the authenticated experience.',
    path: '/settings/branding',
    anchorId: 'settings-branding',
  },
  {
    key: 'modules',
    label: 'Modules',
    description: 'Review runtime-manageable modules and feature visibility.',
    path: '/settings/modules',
    anchorId: 'settings-modules',
  },
  {
    key: 'integrations',
    label: 'Integrations',
    description: 'Manage API token access and RapidPro sync settings.',
    path: '/settings/integrations',
    anchorId: 'settings-api-access',
  },
  {
    key: 'about',
    label: 'About',
    description: 'Application build details and static metadata.',
    path: '/settings/about',
    anchorId: 'settings-about',
  },
]

const settingsSectionIcons: Record<SettingsSection, React.ReactNode> = {
  general: <TuneRoundedIcon />,
  branding: <BrushRoundedIcon />,
  modules: <WidgetsRoundedIcon />,
  integrations: <HubRoundedIcon />,
  about: <InfoRoundedIcon />,
}

export function SettingsPage({ section = 'general' }: { section?: SettingsSection }) {
  const navigate = useNavigate()
  const auth = useAuth()
  const notify = useAppNotify()
  const {
    prefs,
    resolvedMode,
    setMode,
    setPreset,
    setCollapseNavByDefault,
    setShowFooter,
    setShowSukumadMenu,
    setShowAdministrationMenu,
    setPinActionsColumnRight,
    setDataGridBorderRadius,
    setNavLabel,
  } = useUiPreferences()
  const [apiBaseUrlOverride, setApiBaseUrlOverrideValue] = React.useState(() => getApiBaseUrlOverride())
  const [testingConnection, setTestingConnection] = React.useState(false)
  const [appearanceOpen, setAppearanceOpen] = React.useState(false)
  const [brandingDisplayName, setBrandingDisplayName] = React.useState(appName)
  const [brandingImageUrl, setBrandingImageUrl] = React.useState('')
  const [brandingLoading, setBrandingLoading] = React.useState(true)
  const [brandingSaving, setBrandingSaving] = React.useState(false)
  const [brandingPreviewBroken, setBrandingPreviewBroken] = React.useState(false)
  const [brandingErrorMessage, setBrandingErrorMessage] = React.useState('')
  const [authMode, setAuthMode] = React.useState<AuthMode>(() => loadAuthSettings().authMode)
  const [apiToken, setApiToken] = React.useState(() => loadAuthSettings().apiToken)
  const [tokenName, setTokenName] = React.useState('Web API token')
  const [createdApiToken, setCreatedApiToken] = React.useState('')
  const [apiAccessSaving, setApiAccessSaving] = React.useState(false)
  const [apiAccessError, setApiAccessError] = React.useState('')
  const [apiTokenCreating, setApiTokenCreating] = React.useState(false)
  const [moduleEnablement, setModuleEnablement] = React.useState<ModuleEffectiveConfig[]>([])
  const [moduleEnablementLoading, setModuleEnablementLoading] = React.useState(true)
  const [moduleEnablementSaving, setModuleEnablementSaving] = React.useState(false)
  const [moduleEnablementError, setModuleEnablementError] = React.useState('')
  const [runtimeConfig, setRuntimeConfig] = React.useState<Record<string, unknown> | null>(null)
  const [runtimeConfigLoading, setRuntimeConfigLoading] = React.useState(false)
  const [runtimeConfigError, setRuntimeConfigError] = React.useState('')
  const [runtimeConfigFormat, setRuntimeConfigFormat] = React.useState<RuntimeConfigFormat>('json')
  const [rapidProServerCode, setRapidProServerCode] = React.useState('rapidpro')
  const [rapidProFields, setRapidProFields] = React.useState<RapidProContactField[]>([])
  const [rapidProMappings, setRapidProMappings] = React.useState<RapidProReporterFieldMapping[]>([])
  const [rapidProLastFetchedAt, setRapidProLastFetchedAt] = React.useState<string | null>(null)
  const [rapidProValidation, setRapidProValidation] = React.useState<RapidProReporterSyncValidation>({ isValid: true })
  const [rapidProSyncLoading, setRapidProSyncLoading] = React.useState(true)
  const [rapidProSyncSaving, setRapidProSyncSaving] = React.useState(false)
  const [rapidProSyncRefreshing, setRapidProSyncRefreshing] = React.useState(false)
  const [rapidProSyncError, setRapidProSyncError] = React.useState('')

  const runtimeToggleModules = React.useMemo(() => {
    const definitionsById = new Map(moduleRegistry.map((module) => [module.id, module]))

    return moduleEnablement
      .filter((module): module is ModuleEffectiveConfig & { moduleId: (typeof moduleRegistry)[number]['id'] } => {
        return definitionsById.has(module.moduleId) && module.editable === true
      })
      .sort(
        (left, right) =>
          moduleRegistry.findIndex((module) => module.id === left.moduleId) -
          moduleRegistry.findIndex((module) => module.id === right.moduleId),
      )
      .map((module) => {
        const definition = definitionsById.get(module.moduleId)
        return {
          ...module,
          label: definition?.label ?? module.moduleId,
          description: module.description ?? definition?.metadata?.description ?? 'No description provided.',
        }
      })
  }, [moduleEnablement])

  const canWriteBranding = React.useMemo(
    () => (auth.user?.permissions ?? []).some((permission) => permission.trim().toLowerCase() === 'settings.write'),
    [auth.user?.permissions],
  )
  const canReadModuleEnablement = React.useMemo(
    () =>
      canWriteBranding ||
      (auth.user?.permissions ?? []).some((permission) => permission.trim().toLowerCase() === 'settings.read'),
    [auth.user?.permissions, canWriteBranding],
  )
  const canManageApiTokens = React.useMemo(
    () => hasPermission('api_tokens.write'),
    [],
  )
  const currentPermissions = React.useMemo(
    () =>
      Array.from(
        new Set((auth.user?.permissions ?? []).map((permission) => permission.trim()).filter((permission) => permission)),
      ),
    [auth.user?.permissions],
  )
  const runtimeConfigJson = React.useMemo(
    () => (runtimeConfig ? JSON.stringify(runtimeConfig, null, 2) : ''),
    [runtimeConfig],
  )
  const runtimeConfigYaml = React.useMemo(
    () => (runtimeConfig ? toYaml(runtimeConfig) : ''),
    [runtimeConfig],
  )
  const runtimeConfigText = runtimeConfigFormat === 'yaml' ? runtimeConfigYaml : runtimeConfigJson
  const currentSection = settingsSections.find((item) => item.key === section) ?? settingsSections[0]
  const isGeneralSection = currentSection.key === 'general'
  const isBrandingSection = currentSection.key === 'branding'
  const isModulesSection = currentSection.key === 'modules'
  const isIntegrationsSection = currentSection.key === 'integrations'
  const isAboutSection = currentSection.key === 'about'

  React.useEffect(() => {
    if (!isBrandingSection) {
      setBrandingLoading(false)
      return
    }

    let active = true
    setBrandingLoading(true)
    apiRequest<{ appDisplayName?: string; applicationDisplayName?: string; loginImageUrl?: string | null }>(
      '/settings/login-branding',
      { method: 'GET' },
      { retryOnUnauthorized: true },
    )
      .then((payload) => {
        if (!active) {
          return
        }
        setBrandingDisplayName((payload.appDisplayName ?? payload.applicationDisplayName ?? '').trim() || appName)
        setBrandingImageUrl(typeof payload.loginImageUrl === 'string' ? payload.loginImageUrl.trim() : '')
        setBrandingPreviewBroken(false)
      })
      .catch((error) => {
        if (active) {
          setBrandingDisplayName(appName)
          setBrandingImageUrl('')
          void handleAppError(error, {
            fallbackMessage: 'Unable to load login branding settings.',
            notifier: notify,
          })
        }
      })
      .finally(() => {
        if (active) {
          setBrandingLoading(false)
        }
      })

    return () => {
      active = false
    }
  }, [isBrandingSection, notify])

  React.useEffect(() => {
    if (!isModulesSection) {
      setModuleEnablement([])
      setModuleEnablementLoading(false)
      return
    }
    if (!canReadModuleEnablement) {
      setModuleEnablement([])
      setModuleEnablementLoading(false)
      return
    }
    let active = true
    setModuleEnablementLoading(true)
    apiRequest<{ modules: ModuleEffectiveConfig[] }>('/settings/module-enablement', { method: 'GET' })
      .then((payload) => {
        if (!active) {
          return
        }
        setModuleEnablement(payload.modules ?? [])
      })
      .catch((error) => {
        if (active) {
          void handleAppError(error, {
            fallbackMessage: 'Unable to load module enablement settings.',
            notifier: notify,
          })
        }
      })
      .finally(() => {
        if (active) {
          setModuleEnablementLoading(false)
        }
      })
    return () => {
      active = false
    }
  }, [canReadModuleEnablement, isModulesSection, notify])

  React.useEffect(() => {
    if (!isIntegrationsSection) {
      setRapidProSyncLoading(false)
      setRapidProFields([])
      setRapidProMappings([])
      setRapidProValidation({ isValid: true })
      return
    }
    if (!canReadModuleEnablement) {
      setRapidProSyncLoading(false)
      setRapidProFields([])
      setRapidProMappings([])
      setRapidProValidation({ isValid: true })
      return
    }
    let active = true
    setRapidProSyncLoading(true)
    apiRequest<RapidProReporterSyncSettingsResponse>('/settings/rapidpro-reporter-sync', { method: 'GET' })
      .then((payload) => {
        if (!active) {
          return
        }
        applyRapidProSyncPayload(payload)
      })
      .catch((error) => {
        if (active) {
          void handleAppError(error, {
            fallbackMessage: 'Unable to load RapidPro reporter sync settings.',
            notifier: notify,
          })
        }
      })
      .finally(() => {
        if (active) {
          setRapidProSyncLoading(false)
        }
      })
    return () => {
      active = false
    }
  }, [canReadModuleEnablement, isIntegrationsSection, notify])

  const brandingUrlValidationError = React.useMemo(() => {
    if (!brandingImageUrl.trim()) {
      return ''
    }
    try {
      const parsed = new URL(brandingImageUrl)
      if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
        return 'Image URL must use http or https.'
      }
      return ''
    } catch {
      return 'Image URL must be a valid absolute URL.'
    }
  }, [brandingImageUrl])

  const handleBaseUrlSave = () => {
    setApiBaseUrlOverride(apiBaseUrlOverride)
    notify.success(
      apiBaseUrlOverride.trim()
        ? 'API base URL override saved.'
        : 'API base URL override cleared. Using default environment URL.',
    )
  }

  const handleSaveApiAccess = () => {
    setApiAccessSaving(true)
    setApiAccessError('')
    const saved = saveAuthSettings({
      authMode,
      apiToken: authMode === 'api_token' ? apiToken : '',
    })
    setAuthMode(saved.authMode)
    setApiToken(saved.apiToken)
    setApiAccessSaving(false)
    notify.success('API access settings saved.')
  }

  const handleTestConnection = async () => {
    setTestingConnection(true)
    try {
      const health = await apiRequest<HealthResponse>('/health', { method: 'GET' }, { withAuth: false, retryOnUnauthorized: false })
      notify.success(`Connection successful (${health.status ?? 'ok'})`)
    } catch (error) {
      await handleAppError(error, {
        fallbackMessage: 'Connection failed. Please verify the API base URL and try again.',
        notifier: notify,
      })
    } finally {
      setTestingConnection(false)
    }
  }

  const handleSaveBranding = async () => {
    if (!canWriteBranding) {
      return
    }
    if (brandingUrlValidationError) {
      setBrandingErrorMessage(brandingUrlValidationError)
      return
    }

    setBrandingSaving(true)
    setBrandingErrorMessage('')
    try {
      const payload = await apiRequest<{ appDisplayName?: string; applicationDisplayName?: string; loginImageUrl?: string | null }>(
        '/settings/login-branding',
        {
          method: 'PUT',
          body: JSON.stringify({
            applicationDisplayName: brandingDisplayName.trim(),
            loginImageUrl: brandingImageUrl.trim() || null,
          }),
        },
      )
      setBrandingDisplayName((payload.appDisplayName ?? payload.applicationDisplayName ?? '').trim() || appName)
      setBrandingImageUrl(typeof payload.loginImageUrl === 'string' ? payload.loginImageUrl.trim() : '')
      setBrandingPreviewBroken(false)
      notify.success('Login branding saved.')
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to save login branding.',
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setBrandingErrorMessage(`${normalized.message}${requestId}`)
    } finally {
      setBrandingSaving(false)
    }
  }

  const applyRapidProSyncPayload = React.useCallback((payload: RapidProReporterSyncSettingsResponse) => {
    setRapidProServerCode((payload.rapidProServerCode ?? '').trim() || 'rapidpro')
    setRapidProFields(payload.availableFields ?? [])
    setRapidProMappings(payload.mappings ?? [])
    setRapidProLastFetchedAt(payload.lastFetchedAt ?? null)
    setRapidProValidation(payload.validation ?? { isValid: true })
  }, [])

  const handleRapidProFieldMappingChange = React.useCallback((sourceKey: string, rapidProFieldKey: string) => {
    setRapidProMappings((current) => {
      const filtered = current.filter((item) => item.sourceKey !== sourceKey)
      if (!rapidProFieldKey) {
        return filtered
      }
      const sourceLabel = rapidProReporterSourceOptions.find((item) => item.key === sourceKey)?.label ?? sourceKey
      return [...filtered, { sourceKey, sourceLabel, rapidProFieldKey }]
    })
  }, [])

  const handleRefreshRapidProFields = async () => {
    if (!canWriteBranding) {
      return
    }
    setRapidProSyncRefreshing(true)
    setRapidProSyncError('')
    try {
      const payload = await apiRequest<RapidProReporterSyncSettingsResponse>(
        '/settings/rapidpro-reporter-sync/refresh-fields',
        { method: 'POST' },
      )
      applyRapidProSyncPayload(payload)
      notify.success('RapidPro fields refreshed.')
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to refresh RapidPro fields.',
        notifier: notify,
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setRapidProSyncError(`${normalized.message}${requestId}`)
    } finally {
      setRapidProSyncRefreshing(false)
    }
  }

  const handleSaveRapidProSync = async () => {
    if (!canWriteBranding) {
      return
    }
    setRapidProSyncSaving(true)
    setRapidProSyncError('')
    try {
      const payload = await apiRequest<RapidProReporterSyncSettingsResponse>(
        '/settings/rapidpro-reporter-sync',
        {
          method: 'PUT',
          body: JSON.stringify({
            rapidProServerCode,
            mappings: rapidProMappings.map((item) => ({
              sourceKey: item.sourceKey,
              rapidProFieldKey: item.rapidProFieldKey,
            })),
          }),
        },
      )
      applyRapidProSyncPayload(payload)
      notify.success('RapidPro reporter sync settings saved.')
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to save RapidPro reporter sync settings.',
        notifier: notify,
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setRapidProSyncError(`${normalized.message}${requestId}`)
    } finally {
      setRapidProSyncSaving(false)
    }
  }

  const handleToggleModuleEnablement = async (moduleId: string, enabled: boolean) => {
    if (!canWriteBranding) {
      return
    }
    setModuleEnablementSaving(true)
    setModuleEnablementError('')
    try {
      const payload = await apiRequest<{ modules: ModuleEffectiveConfig[] }>('/settings/module-enablement', {
        method: 'PUT',
        body: JSON.stringify({
          modules: [{ moduleId, enabled }],
        }),
      })
      setModuleEnablement(payload.modules ?? [])
      notify.success('Module enablement updated.')
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to update module enablement.',
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setModuleEnablementError(`${normalized.message}${requestId}`)
    } finally {
      setModuleEnablementSaving(false)
    }
  }

  const handleCreateApiToken = async () => {
    if (!canManageApiTokens || !tokenName.trim()) {
      return
    }

    setApiTokenCreating(true)
    setApiAccessError('')
    try {
      const token = await apiRequest<CreateAPITokenResponse>('/admin/api-tokens', {
        method: 'POST',
        body: JSON.stringify({
          name: tokenName.trim(),
          permissions: currentPermissions,
        }),
      })
      setCreatedApiToken(token.token)
      const saved = saveAuthSettings({
        authMode: 'api_token',
        apiToken: token.token,
      })
      setAuthMode(saved.authMode)
      setApiToken(saved.apiToken)
      notify.success('API token created. Copy it now.')
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to create API token.',
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setApiAccessError(`${normalized.message}${requestId}`)
    } finally {
      setApiTokenCreating(false)
    }
  }

  const handleCopyCreatedToken = async () => {
    if (!createdApiToken.trim()) {
      return
    }
    try {
      await navigator.clipboard.writeText(createdApiToken)
      notify.success('API token copied.')
    } catch {
      notify.error('Unable to copy API token.')
    }
  }

  const loadRuntimeConfig = React.useCallback(async () => {
    if (!canReadModuleEnablement) {
      setRuntimeConfig(null)
      return
    }

    setRuntimeConfigLoading(true)
    setRuntimeConfigError('')
    try {
      const payload = await apiRequest<RuntimeConfigResponse>('/settings/runtime-config', { method: 'GET' })
      setRuntimeConfig(payload.config ?? null)
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to load runtime configuration.',
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setRuntimeConfigError(`${normalized.message}${requestId}`)
    } finally {
      setRuntimeConfigLoading(false)
    }
  }, [canReadModuleEnablement])

  React.useEffect(() => {
    if (!isModulesSection || !canReadModuleEnablement) {
      setRuntimeConfig(null)
      setRuntimeConfigLoading(false)
      return
    }
    void loadRuntimeConfig()
  }, [canReadModuleEnablement, isModulesSection, loadRuntimeConfig])

  const handleCopyRuntimeConfig = async () => {
    if (!runtimeConfigText) {
      return
    }
    try {
      await navigator.clipboard.writeText(runtimeConfigText)
      notify.success(`Runtime config copied as ${runtimeConfigFormat.toUpperCase()}.`)
    } catch {
      notify.error('Unable to copy runtime config.')
    }
  }

  return (
    <Stack spacing={3}>
      <Paper elevation={1} sx={{ p: 3 }}>
        <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 1 }}>
          <Avatar
            variant="rounded"
            sx={{
              width: 48,
              height: 48,
              bgcolor: 'primary.main',
              color: 'primary.contrastText',
              boxShadow: (theme) => `0 10px 24px ${theme.palette.primary.main}33`,
            }}
          >
            {settingsSectionIcons[currentSection.key]}
          </Avatar>
          <Box>
            <Typography variant="h4" component="h1">
              Settings
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {currentSection.label}
            </Typography>
          </Box>
        </Stack>
        <Typography color="text.secondary">{currentSection.description}</Typography>
        <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" sx={{ mt: 2 }}>
          {settingsSections.map((item) => (
            <Button
              key={item.key}
              variant={item.key === currentSection.key ? 'contained' : 'outlined'}
              startIcon={settingsSectionIcons[item.key]}
              onClick={() => void navigate({ to: item.path })}
            >
              {item.label}
            </Button>
          ))}
        </Stack>
      </Paper>

      {isGeneralSection ? (
        <>
          <Paper id="settings-appearance" elevation={1} sx={{ p: 3 }}>
            <Stack spacing={2.5}>
              <Typography variant="h6" component="h2">
                Appearance
              </Typography>
              <Divider />

              <FormControl size="small" sx={{ maxWidth: 220 }}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>
                  Theme mode
                </Typography>
                <Select
                  inputProps={{ 'aria-label': 'Theme mode' }}
                  value={prefs.mode}
                  onChange={(event) => setMode(event.target.value as UiThemeMode)}
                >
                  <MenuItem value="light">Light</MenuItem>
                  <MenuItem value="dark">Dark</MenuItem>
                  <MenuItem value="system">System</MenuItem>
                </Select>
                <Typography variant="body2" color="text.secondary">
                  Active mode: {resolvedMode}
                </Typography>
              </FormControl>

              <Box>
                <Typography variant="subtitle2">Palette preset</Typography>
                <Typography color="text.secondary" sx={{ mb: 1.5 }}>
                  Active preset: {palettePresets.find((preset) => preset.id === prefs.preset)?.name ?? 'Custom'}
                </Typography>
                <Stack direction="row" spacing={1.25} alignItems="center" useFlexGap flexWrap="wrap">
                  {palettePresets.slice(0, 4).map((preset) => (
                    <Button
                      key={preset.id}
                      size="small"
                      variant={preset.id === prefs.preset ? 'contained' : 'outlined'}
                      onClick={() => setPreset(preset.id)}
                    >
                      {preset.name}
                    </Button>
                  ))}
                  <Button variant="text" startIcon={<PaletteRoundedIcon />} onClick={() => setAppearanceOpen(true)}>
                    Browse all presets
                  </Button>
                </Stack>
              </Box>

              <Paper variant="outlined" sx={{ p: 2 }}>
                <Typography variant="subtitle2" gutterBottom>
                  Preview
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>
                  Current palette and mode preview using common components.
                </Typography>
                <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
                  <Button variant="contained" size="small">
                    Primary Action
                  </Button>
                  <Button variant="outlined" size="small" color="secondary">
                    Secondary
                  </Button>
                  <Chip label="Theme Chip" color="primary" />
                </Stack>
              </Paper>
            </Stack>
          </Paper>

          <Paper id="settings-navigation" elevation={1} sx={{ p: 3 }}>
            <Stack spacing={2}>
              <Typography variant="h6" component="h2">
                Navigation
              </Typography>
              <Divider />
              <FormControlLabel
                control={
                  <Switch
                    checked={prefs.collapseNavByDefault}
                    onChange={(event) => setCollapseNavByDefault(event.target.checked)}
                  />
                }
                label="Start with side navigation collapsed"
              />
              <FormControlLabel
                control={<Switch checked={prefs.showFooter} onChange={(event) => setShowFooter(event.target.checked)} />}
                label="Show footer on authenticated pages"
              />
              <FormControlLabel
                control={<Switch checked={prefs.showSukumadMenu} onChange={(event) => setShowSukumadMenu(event.target.checked)} />}
                label="Show Sukumad menu group"
              />
              <FormControlLabel
                control={
                  <Switch
                    checked={prefs.showAdministrationMenu}
                    onChange={(event) => setShowAdministrationMenu(event.target.checked)}
                  />
                }
                label="Show Administration menu group"
              />
              <Divider />
              <Typography variant="subtitle2">Navigation labels</Typography>
              <Typography color="text.secondary">
                Rename drawer links for this browser profile. Leaving a field blank restores the default label.
              </Typography>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5} useFlexGap flexWrap="wrap">
                {navigationLabelFields.map((field) => (
                  <TextField
                    key={field.id}
                    label={field.label}
                    value={prefs.navLabels[field.id] ?? ''}
                    onChange={(event) => setNavLabel(field.id, event.target.value)}
                    placeholder="Use default label"
                    size="small"
                    sx={{ minWidth: { xs: '100%', md: 240 } }}
                  />
                ))}
              </Stack>
            </Stack>
          </Paper>

          <Paper id="settings-data-grid" elevation={1} sx={{ p: 3 }}>
            <Stack spacing={2}>
              <Typography variant="h6" component="h2">
                Data Grid Defaults
              </Typography>
              <Divider />
              <FormControlLabel
                control={
                  <Switch
                    checked={prefs.pinActionsColumnRight}
                    onChange={(event) => setPinActionsColumnRight(event.target.checked)}
                  />
                }
                label="Pin actions column to the right"
              />
              <TextField
                label="DataGrid border radius"
                type="number"
                value={prefs.dataGridBorderRadius}
                onChange={(event) => setDataGridBorderRadius(Number(event.target.value))}
                inputProps={{ min: 4, max: 32 }}
                sx={{ maxWidth: 220 }}
              />
            </Stack>
          </Paper>

          <Paper id="settings-connection" elevation={1} sx={{ p: 3 }}>
            <Stack spacing={2}>
              <Typography variant="h6" component="h2">
                Connection
              </Typography>
              <Divider />
              <Alert severity="info">Local override applies only in this browser profile.</Alert>
              <TextField
                label="API Base URL Override"
                placeholder="http://127.0.0.1:8080/api/v1"
                value={apiBaseUrlOverride}
                onChange={(event) => setApiBaseUrlOverrideValue(event.target.value)}
                fullWidth
              />
              <Stack direction="row" spacing={1}>
                <Button variant="contained" onClick={handleBaseUrlSave}>
                  Save Override
                </Button>
                <Button variant="outlined" onClick={handleTestConnection} disabled={testingConnection}>
                  {testingConnection ? 'Testing...' : 'Test Connection'}
                </Button>
              </Stack>
            </Stack>
          </Paper>
        </>
      ) : null}

      {isModulesSection ? (
        <>
          <Paper id="settings-modules" elevation={1} sx={{ p: 3 }}>
            <Stack spacing={2}>
              <Typography variant="h6" component="h2">
                Module Enablement
              </Typography>
              <Divider />
              <Typography color="text.secondary">
                Turn runtime-manageable application modules on or off. Static modules are not listed here.
              </Typography>
              {!canReadModuleEnablement ? <Alert severity="info">You need settings.read permission to view module flags.</Alert> : null}
              {moduleEnablementError ? <Alert severity="error">{moduleEnablementError}</Alert> : null}
              {canReadModuleEnablement ? (
                moduleEnablementLoading ? (
                  <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
                    <Typography color="text.secondary">Loading module flags...</Typography>
                  </Box>
                ) : runtimeToggleModules.length === 0 ? (
                  <Alert severity="info">No runtime-manageable modules are available for toggle control.</Alert>
                ) : (
                  <Stack spacing={1.5}>
                    {runtimeToggleModules.map((module) => {
                      const isEditable = canWriteBranding
                      return (
                        <Box
                          key={module.moduleId}
                          data-testid={`module-flag-${module.moduleId}`}
                          sx={{
                            border: '1px solid',
                            borderColor: 'divider',
                            borderRadius: 2,
                            p: 1.5,
                          }}
                        >
                          <Stack spacing={1}>
                            <Stack direction="row" alignItems="center" justifyContent="space-between" useFlexGap flexWrap="wrap">
                              <Typography variant="subtitle1">{module.label}</Typography>
                              <FormControlLabel
                                control={
                                  <Switch
                                    checked={module.enabled}
                                    disabled={!isEditable || moduleEnablementSaving}
                                    inputProps={{ 'aria-label': `Toggle ${module.label} module` }}
                                    onChange={(event) => void handleToggleModuleEnablement(module.moduleId, event.target.checked)}
                                  />
                                }
                                label={module.enabled ? 'Enabled' : 'Disabled'}
                              />
                            </Stack>
                            <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
                              <Chip size="small" label={module.enabled ? 'Enabled' : 'Disabled'} color={module.enabled ? 'success' : 'default'} />
                              <Chip size="small" label={`Source: ${module.source}`} variant="outlined" />
                              {module.experimental ? <Chip size="small" label="Experimental" color="warning" /> : null}
                            </Stack>
                            <Typography color="text.secondary">{module.description}</Typography>
                          </Stack>
                        </Box>
                      )
                    })}
                  </Stack>
                )
              ) : null}
              {canReadModuleEnablement && !canWriteBranding ? (
                <Alert severity="info">You need settings.write permission to change runtime-manageable module flags.</Alert>
              ) : null}
            </Stack>
          </Paper>

          <Paper id="settings-runtime-config" elevation={1} sx={{ p: 3 }}>
            <Stack spacing={2}>
              <Typography variant="h6" component="h2">
                Runtime Config
              </Typography>
              <Divider />
              <Typography color="text.secondary">
                Read the active backend configuration snapshot with sensitive fields masked before display.
              </Typography>
              {!canReadModuleEnablement ? <Alert severity="info">You need settings.read permission to view runtime configuration.</Alert> : null}
              {runtimeConfigError ? <Alert severity="error">{runtimeConfigError}</Alert> : null}
              {canReadModuleEnablement ? (
                <>
                  <Stack direction="row" spacing={1}>
                    <Button variant="outlined" onClick={() => void loadRuntimeConfig()} disabled={runtimeConfigLoading}>
                      {runtimeConfigLoading ? 'Refreshing...' : runtimeConfig ? 'Refresh' : 'View Running Config'}
                    </Button>
                    <Button
                      variant={runtimeConfigFormat === 'json' ? 'contained' : 'outlined'}
                      onClick={() => setRuntimeConfigFormat('json')}
                      disabled={!runtimeConfig}
                    >
                      JSON
                    </Button>
                    <Button
                      variant={runtimeConfigFormat === 'yaml' ? 'contained' : 'outlined'}
                      onClick={() => setRuntimeConfigFormat('yaml')}
                      disabled={!runtimeConfig}
                    >
                      YAML
                    </Button>
                    <Button variant="text" onClick={() => void handleCopyRuntimeConfig()} disabled={!runtimeConfigText}>
                      {runtimeConfigFormat === 'yaml' ? 'Copy YAML' : 'Copy JSON'}
                    </Button>
                  </Stack>
                  <TextField
                    label={`Sanitized runtime config (${runtimeConfigFormat.toUpperCase()})`}
                    value={runtimeConfigText}
                    multiline
                    minRows={14}
                    InputProps={{ readOnly: true, sx: { fontFamily: 'monospace' } }}
                    placeholder={runtimeConfigLoading ? 'Loading runtime configuration...' : 'No runtime configuration loaded.'}
                    fullWidth
                  />
                </>
              ) : null}
            </Stack>
          </Paper>
        </>
      ) : null}

      {isIntegrationsSection ? (
        <>
          <Paper id="settings-api-access" elevation={1} sx={{ p: 3 }}>
            <Stack spacing={2}>
              <Typography variant="h6" component="h2">
                API Access
              </Typography>
              <Divider />
              <Typography color="text.secondary">
                Choose whether this browser profile uses the current session or a saved API token for backend requests.
              </Typography>
              <FormControl>
                <RadioGroup
                  value={authMode}
                  onChange={(event) => setAuthMode(event.target.value as AuthMode)}
                >
                  <FormControlLabel value="password" control={<Radio />} label="Username / Password" />
                  <FormControlLabel value="api_token" control={<Radio />} label="API Token" />
                </RadioGroup>
              </FormControl>
              <TextField
                label="API Token"
                type="password"
                value={apiToken}
                onChange={(event) => setApiToken(event.target.value)}
                helperText="Stored locally and used for backend API requests when API token mode is selected."
                fullWidth
              />
              <Stack direction="row" spacing={1} justifyContent="flex-end">
                <Button variant="contained" onClick={handleSaveApiAccess} disabled={apiAccessSaving}>
                  {apiAccessSaving ? 'Saving...' : 'Save API Access'}
                </Button>
              </Stack>
              {canManageApiTokens ? (
                <>
                  <Divider />
                  <Typography variant="subtitle2">Create API Token</Typography>
                  <Typography color="text.secondary">
                    Tokens inherit the current permissions and are shown once after creation.
                  </Typography>
                  <TextField
                    label="Token Name"
                    value={tokenName}
                    onChange={(event) => setTokenName(event.target.value)}
                    fullWidth
                  />
                  <Stack direction="row" spacing={1} justifyContent="flex-end">
                    <Button variant="contained" onClick={() => void handleCreateApiToken()} disabled={apiTokenCreating || !tokenName.trim()}>
                      {apiTokenCreating ? 'Creating...' : 'Create Token'}
                    </Button>
                  </Stack>
                </>
              ) : (
                <Alert severity="info">You need api_tokens.write permission to create API tokens here.</Alert>
              )}
              {apiAccessError ? <Alert severity="error">{apiAccessError}</Alert> : null}
              {createdApiToken ? (
                <Box
                  sx={{
                    border: '1px solid',
                    borderColor: 'divider',
                    borderRadius: 2,
                    p: 2,
                  }}
                >
                  <Stack spacing={1.25}>
                    <Typography variant="subtitle2">Plaintext token</Typography>
                    <TextField
                      value={createdApiToken}
                      multiline
                      minRows={2}
                      InputProps={{ readOnly: true, sx: { fontFamily: 'monospace' } }}
                      fullWidth
                    />
                    <Stack direction="row" justifyContent="flex-end">
                      <Button variant="outlined" onClick={() => void handleCopyCreatedToken()}>
                        Copy Token
                      </Button>
                    </Stack>
                  </Stack>
                </Box>
              ) : null}
            </Stack>
          </Paper>

          <Paper id="settings-rapidpro" elevation={1} sx={{ p: 3 }}>
            <Stack spacing={2}>
              <Typography variant="h6" component="h2">
                RapidPro Reporter Sync
              </Typography>
              <Divider />
              <Typography color="text.secondary">
                Fetch RapidPro custom contact fields once, then map reporter values to RapidPro built-in targets like Contact Name and URNs or to custom fields.
              </Typography>
              {!canReadModuleEnablement ? <Alert severity="info">You need settings.read permission to view RapidPro sync settings.</Alert> : null}
              {rapidProSyncError ? <Alert severity="error">{rapidProSyncError}</Alert> : null}
              {canReadModuleEnablement ? (
                rapidProSyncLoading ? (
                  <Typography color="text.secondary">Loading RapidPro sync settings...</Typography>
                ) : (
                  <>
                    <TextField
                      label="RapidPro Server Code"
                      value={rapidProServerCode}
                      onChange={(event) => setRapidProServerCode(event.target.value)}
                      disabled={!canWriteBranding || rapidProSyncSaving || rapidProSyncRefreshing}
                      helperText="Defaults to the existing RapidPro integration server code."
                      sx={{ maxWidth: 280 }}
                    />
                    <Stack direction="row" spacing={1} alignItems="center" useFlexGap flexWrap="wrap">
                      <Button variant="outlined" onClick={() => void handleRefreshRapidProFields()} disabled={!canWriteBranding || rapidProSyncRefreshing}>
                        {rapidProSyncRefreshing ? 'Refreshing...' : 'Refresh RapidPro Fields'}
                      </Button>
                      <Typography color="text.secondary">
                        {rapidProLastFetchedAt ? `Last fetched: ${new Date(rapidProLastFetchedAt).toLocaleString()}` : 'Fields have not been fetched yet.'}
                      </Typography>
                    </Stack>
                    {rapidProFields.length === 0 ? (
                      <Alert severity="info">Refresh RapidPro fields to populate the available mapping targets.</Alert>
                    ) : (
                      <Stack spacing={1.5}>
                        <Alert severity="info">
                          Built-in targets are always available. Custom field targets are fetched from RapidPro <code>/fields.json</code>.
                        </Alert>
                        {rapidProReporterSourceOptions.map((option) => {
                          const selected = rapidProMappings.find((item) => item.sourceKey === option.key)?.rapidProFieldKey ?? ''
                          return (
                            <FormControl key={option.key} fullWidth>
                              <Select
                                displayEmpty
                                value={selected}
                                inputProps={{ 'aria-label': option.label }}
                                disabled={!canWriteBranding || rapidProSyncSaving}
                                onChange={(event) => handleRapidProFieldMappingChange(option.key, event.target.value)}
                              >
                                <MenuItem value="">
                                  <em>Do not sync</em>
                                </MenuItem>
                                {rapidProFields.map((field) => (
                                  <MenuItem key={field.key} value={field.key}>
                                    {getRapidProFieldOptionLabel(field)}
                                  </MenuItem>
                                ))}
                              </Select>
                              <Typography variant="body2" color="text.secondary">
                                {option.label}
                              </Typography>
                            </FormControl>
                          )
                        })}
                      </Stack>
                    )}
                    {!rapidProValidation.isValid ? (
                      <Alert severity="warning">{(rapidProValidation.errors ?? []).join(' ')}</Alert>
                    ) : (
                      <Alert severity="success">Saved RapidPro mappings are valid and ready for sync.</Alert>
                    )}
                    {!canWriteBranding ? <Alert severity="info">You need settings.write permission to change RapidPro sync settings.</Alert> : null}
                    <Stack direction="row" justifyContent="flex-end">
                      <Button variant="contained" onClick={() => void handleSaveRapidProSync()} disabled={!canWriteBranding || rapidProSyncSaving}>
                        {rapidProSyncSaving ? 'Saving...' : 'Save RapidPro Sync Settings'}
                      </Button>
                    </Stack>
                  </>
                )
              ) : null}
            </Stack>
          </Paper>
        </>
      ) : null}

      {isBrandingSection ? (
        <Paper id="settings-branding" elevation={1} sx={{ p: 3 }}>
          <Stack spacing={2}>
            <Typography variant="h6" component="h2">
              Login Branding
            </Typography>
            <Divider />
            <Typography color="text.secondary">
              Configure authentication screen branding for both desktop and web clients.
            </Typography>
            <TextField
              label="Application Display Name"
              value={brandingDisplayName}
              onChange={(event) => setBrandingDisplayName(event.target.value)}
              disabled={brandingLoading || brandingSaving || !canWriteBranding}
              fullWidth
            />
            <TextField
              label="Login Image URL"
              value={brandingImageUrl}
              onChange={(event) => {
                setBrandingImageUrl(event.target.value)
                setBrandingPreviewBroken(false)
              }}
              error={Boolean(brandingUrlValidationError)}
              helperText={brandingUrlValidationError || 'Optional absolute http(s) URL.'}
              disabled={brandingLoading || brandingSaving || !canWriteBranding}
              fullWidth
            />
            <Box
              sx={{
                border: '1px solid',
                borderColor: 'divider',
                borderRadius: 2,
                p: 2,
                minHeight: 140,
                display: 'grid',
                placeItems: 'center',
              }}
            >
              {brandingImageUrl.trim() && !brandingPreviewBroken && !brandingUrlValidationError ? (
                <Box
                  component="img"
                  src={brandingImageUrl}
                  alt="Login branding preview"
                  onError={() => setBrandingPreviewBroken(true)}
                  sx={{ maxWidth: '100%', maxHeight: 150, objectFit: 'contain' }}
                />
              ) : (
                <Stack spacing={1} alignItems="center">
                  <Avatar sx={{ bgcolor: 'primary.main' }}>{brandingDisplayName.slice(0, 1).toUpperCase()}</Avatar>
                  <Typography>{brandingDisplayName || appName}</Typography>
                </Stack>
              )}
            </Box>
            {!canWriteBranding ? <Alert severity="info">You need settings.write permission to update branding.</Alert> : null}
            {brandingErrorMessage ? <Alert severity="error">{brandingErrorMessage}</Alert> : null}
            <Stack direction="row" justifyContent="flex-end">
              <Button
                variant="contained"
                onClick={handleSaveBranding}
                disabled={
                  brandingLoading ||
                  brandingSaving ||
                  !canWriteBranding ||
                  !brandingDisplayName.trim() ||
                  Boolean(brandingUrlValidationError)
                }
              >
                {brandingSaving ? 'Saving...' : 'Save Branding'}
              </Button>
            </Stack>
          </Stack>
        </Paper>
      ) : null}

      {isAboutSection ? (
        <Paper id="settings-about" elevation={1} sx={{ p: 3 }}>
          <Stack spacing={1}>
            <Typography variant="h6" component="h2">
              About
            </Typography>
            <Divider />
            <Typography>
              {appName} <Typography component="span" color="text.secondary">v0.1.0 (web)</Typography>
            </Typography>
            <Typography color="text.secondary">Build: local-dev</Typography>
          </Stack>
        </Paper>
      ) : null}
      <PalettePresetPicker open={appearanceOpen} onClose={() => setAppearanceOpen(false)} />
    </Stack>
  )
}

function toYaml(value: unknown) {
  return stringifyYaml(value, 0).trimEnd()
}

function stringifyYaml(value: unknown, indentLevel: number): string {
  const indent = '  '.repeat(indentLevel)

  if (Array.isArray(value)) {
    if (value.length === 0) {
      return '[]\n'
    }
    return value
      .map((item) => {
        if (isScalar(item)) {
          return `${indent}- ${formatYamlScalar(item)}\n`
        }
        const nested = stringifyYaml(item, indentLevel + 1)
        return `${indent}-\n${nested}`
      })
      .join('')
  }

  if (value && typeof value === 'object') {
    const entries = Object.entries(value as Record<string, unknown>)
    if (entries.length === 0) {
      return '{}\n'
    }
    return entries
      .map(([key, item]) => {
        if (isScalar(item)) {
          return `${indent}${key}: ${formatYamlScalar(item)}\n`
        }
        const nested = stringifyYaml(item, indentLevel + 1)
        return `${indent}${key}:\n${nested}`
      })
      .join('')
  }

  return `${indent}${formatYamlScalar(value)}\n`
}

function isScalar(value: unknown) {
  return value === null || ['string', 'number', 'boolean'].includes(typeof value)
}

function formatYamlScalar(value: unknown) {
  if (value === null) {
    return 'null'
  }
  if (typeof value === 'string') {
    if (value === '' || /[:#[\]{}&,*?|\-<>=!%@`]/.test(value) || /^\s|\s$/.test(value)) {
      return JSON.stringify(value)
    }
    return value
  }
  return String(value)
}
