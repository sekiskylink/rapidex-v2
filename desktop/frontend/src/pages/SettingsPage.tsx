import React from 'react'
import BrushRoundedIcon from '@mui/icons-material/BrushRounded'
import HubRoundedIcon from '@mui/icons-material/HubRounded'
import InfoRoundedIcon from '@mui/icons-material/InfoRounded'
import TuneRoundedIcon from '@mui/icons-material/TuneRounded'
import WidgetsRoundedIcon from '@mui/icons-material/WidgetsRounded'
import {
  Alert,
  Avatar,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Divider,
  FormControl,
  FormControlLabel,
  MenuItem,
  Radio,
  RadioGroup,
  Select,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import { useRouter } from '@tanstack/react-router'
import {
  createApiClient,
  type RapidProContactField,
  type RapidProReporterFieldMapping,
  type RapidProReporterSyncSettingsResponse,
  type RapidProReporterSyncValidation,
} from '../api/client'
import { useSessionPrincipal } from '../auth/hooks'
import { handleAppError } from '../errors/handleAppError'
import { notify } from '../notifications/facade'
import { hasPermission } from '../rbac/permissions'
import type { ModuleEffectiveConfig } from '../registry/moduleEnablement'
import { moduleRegistry } from '../registry/modules'
import { THEME_MODES, type AppSettings, type ThemeMode } from '../settings/types'
import { PalettePresetPicker } from '../ui/PalettePresetPicker'
import { useThemePreferences } from '../ui/theme'

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

export type SettingsSection = 'general' | 'branding' | 'modules' | 'integrations' | 'about'

const settingsSections: Array<{ key: SettingsSection; label: string; description: string; path: string; anchorId: string }> = [
  {
    key: 'general',
    label: 'General',
    description: 'Manage local desktop connection and appearance preferences.',
    path: '/settings/general',
    anchorId: 'settings-connection',
  },
  {
    key: 'branding',
    label: 'Branding',
    description: 'Configure authentication screen branding used by desktop and web clients.',
    path: '/settings/branding',
    anchorId: 'settings-branding',
  },
  {
    key: 'modules',
    label: 'Modules',
    description: 'Review runtime-manageable modules and module-level flags.',
    path: '/settings/modules',
    anchorId: 'settings-modules',
  },
  {
    key: 'integrations',
    label: 'Integrations',
    description: 'Manage API token access and RapidPro reporter sync mappings.',
    path: '/settings/integrations',
    anchorId: 'settings-rapidpro',
  },
  {
    key: 'about',
    label: 'About',
    description: 'Desktop and backend build information.',
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
  const router = useRouter()
  const settingsStore = router.options.context.settingsStore
  const {
    prefs,
    setThemeMode,
    setPalettePreset,
    setPinActionsColumnRight,
    setDataGridBorderRadius,
    setShowSukumadMenu,
    setShowAdministrationMenu,
    setNavLabel,
    presets,
  } = useThemePreferences()
  const principal = useSessionPrincipal()
  const canWriteBranding = hasPermission(principal, 'settings.write')
  const canReadModuleEnablement = hasPermission(principal, 'settings.read') || canWriteBranding
  const canManageApiTokens = hasPermission(principal, 'api_tokens.write')
  const currentPermissions = React.useMemo(
    () =>
      Array.from(
        new Set((principal?.permissions ?? []).map((permission) => permission.trim()).filter((permission) => permission)),
      ),
    [principal?.permissions],
  )
  const currentSection = settingsSections.find((item) => item.key === section) ?? settingsSections[0]
  const isGeneralSection = currentSection.key === 'general'
  const isBrandingSection = currentSection.key === 'branding'
  const isModulesSection = currentSection.key === 'modules'
  const isIntegrationsSection = currentSection.key === 'integrations'
  const isAboutSection = currentSection.key === 'about'

  const [loading, setLoading] = React.useState(true)
  const [saving, setSaving] = React.useState(false)
  const [testing, setTesting] = React.useState(false)
  const [appearanceOpen, setAppearanceOpen] = React.useState(false)
  const [settings, setSettings] = React.useState<AppSettings | null>(null)
  const [backendVersion, setBackendVersion] = React.useState<{ version: string; commit: string; buildDate: string } | null>(
    null,
  )
  const [backendVersionLoading, setBackendVersionLoading] = React.useState(true)
  const [connectionErrorMessage, setConnectionErrorMessage] = React.useState('')
  const [brandingDisplayName, setBrandingDisplayName] = React.useState('BasePro')
  const [brandingImageUrl, setBrandingImageUrl] = React.useState('')
  const [brandingLoading, setBrandingLoading] = React.useState(true)
  const [brandingSaving, setBrandingSaving] = React.useState(false)
  const [brandingErrorMessage, setBrandingErrorMessage] = React.useState('')
  const [brandingPreviewBroken, setBrandingPreviewBroken] = React.useState(false)
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
  const [tokenName, setTokenName] = React.useState('Desktop API token')
  const [createdApiToken, setCreatedApiToken] = React.useState('')
  const [apiAccessError, setApiAccessError] = React.useState('')
  const [apiTokenCreating, setApiTokenCreating] = React.useState(false)

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

  React.useEffect(() => {
    let active = true

    settingsStore
      .loadSettings()
      .then((loaded) => {
        if (active) {
          setSettings(loaded)
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false)
        }
      })

    return () => {
      active = false
    }
  }, [settingsStore])

  const apiClient = React.useMemo(
    () =>
      createApiClient({
        getSettings: async () => settings ?? (await settingsStore.loadSettings()),
      }),
    [settings, settingsStore],
  )

  React.useEffect(() => {
    if (!isAboutSection) {
      setBackendVersion(null)
      setBackendVersionLoading(false)
      return
    }

    let active = true

    const loadBackendVersion = async () => {
      if (!settings?.apiBaseUrl.trim()) {
        if (!active) {
          return
        }
        setBackendVersion(null)
        setBackendVersionLoading(false)
        return
      }

      setBackendVersionLoading(true)
      try {
        const versionInfo = await apiClient.version()
        if (!active) {
          return
        }
        setBackendVersion(versionInfo)
      } catch {
        if (!active) {
          return
        }
        setBackendVersion(null)
      } finally {
        if (active) {
          setBackendVersionLoading(false)
        }
      }
    }

    void loadBackendVersion()

    return () => {
      active = false
    }
  }, [apiClient, isAboutSection, settings?.apiBaseUrl])

  React.useEffect(() => {
    if (!isBrandingSection) {
      setBrandingLoading(false)
      return
    }

    let active = true
    setBrandingLoading(true)
    apiClient
      .getLoginBranding()
      .then((payload) => {
        if (!active) {
          return
        }
        const displayName = (payload.appDisplayName ?? payload.applicationDisplayName ?? '').trim() || 'BasePro'
        const loginImageUrl = typeof payload.loginImageUrl === 'string' ? payload.loginImageUrl.trim() : ''
        setBrandingDisplayName(displayName)
        setBrandingImageUrl(loginImageUrl)
        setBrandingPreviewBroken(false)
      })
      .catch((error) => {
        if (!active) {
          return
        }
        setBrandingDisplayName('BasePro')
        setBrandingImageUrl('')
        void handleAppError(error, { fallbackMessage: 'Unable to load login branding settings.' })
      })
      .finally(() => {
        if (active) {
          setBrandingLoading(false)
        }
      })

    return () => {
      active = false
    }
  }, [apiClient, isBrandingSection])

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
    apiClient
      .getModuleEnablementSettings()
      .then((payload) => {
        if (!active) {
          return
        }
        setModuleEnablement(payload.modules ?? [])
      })
      .catch((error) => {
        if (!active) {
          return
        }
        void handleAppError(error, { fallbackMessage: 'Unable to load module enablement settings.' })
      })
      .finally(() => {
        if (active) {
          setModuleEnablementLoading(false)
        }
      })

    return () => {
      active = false
    }
  }, [apiClient, canReadModuleEnablement, isModulesSection])

  const applyRapidProSyncPayload = React.useCallback((payload: RapidProReporterSyncSettingsResponse) => {
    setRapidProServerCode((payload.rapidProServerCode ?? '').trim() || 'rapidpro')
    setRapidProFields(payload.availableFields ?? [])
    setRapidProMappings(payload.mappings ?? [])
    setRapidProLastFetchedAt(payload.lastFetchedAt ?? null)
    setRapidProValidation(payload.validation ?? { isValid: true })
  }, [])

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
    apiClient
      .getRapidProReporterSyncSettings()
      .then((payload) => {
        if (active) {
          applyRapidProSyncPayload(payload)
        }
      })
      .catch((error) => {
        if (!active) {
          return
        }
        void handleAppError(error, { fallbackMessage: 'Unable to load RapidPro reporter sync settings.' })
      })
      .finally(() => {
        if (active) {
          setRapidProSyncLoading(false)
        }
      })
    return () => {
      active = false
    }
  }, [apiClient, applyRapidProSyncPayload, canReadModuleEnablement, isIntegrationsSection])

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

  const runtimeConfigJson = React.useMemo(
    () => (runtimeConfig ? JSON.stringify(runtimeConfig, null, 2) : ''),
    [runtimeConfig],
  )
  const runtimeConfigYaml = React.useMemo(
    () => (runtimeConfig ? toYaml(runtimeConfig) : ''),
    [runtimeConfig],
  )
  const runtimeConfigText = runtimeConfigFormat === 'yaml' ? runtimeConfigYaml : runtimeConfigJson

  const onSaveBranding = async () => {
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
      const saved = await apiClient.updateLoginBranding({
        applicationDisplayName: brandingDisplayName.trim(),
        loginImageUrl: brandingImageUrl.trim() || null,
      })
      setBrandingDisplayName((saved.appDisplayName ?? saved.applicationDisplayName ?? '').trim() || 'BasePro')
      setBrandingImageUrl(typeof saved.loginImageUrl === 'string' ? saved.loginImageUrl : '')
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

  const onToggleModuleEnablement = async (moduleId: string, enabled: boolean) => {
    if (!canWriteBranding) {
      return
    }
    setModuleEnablementSaving(true)
    setModuleEnablementError('')
    try {
      const payload = await apiClient.updateModuleEnablementSettings({
        modules: [{ moduleId, enabled }],
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

  const onSaveConnection = async () => {
    if (!settings) {
      return
    }

    setSaving(true)
    setConnectionErrorMessage('')
    try {
      const saved = await settingsStore.saveSettings({
        apiBaseUrl: settings.apiBaseUrl,
        requestTimeoutSeconds: settings.requestTimeoutSeconds,
      })
      setSettings(saved)
      setBackendVersionLoading(true)
      notify.success('Connection settings saved.')
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to save settings.',
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setConnectionErrorMessage(`${normalized.message}${requestId}`)
    } finally {
      setSaving(false)
    }
  }

  const onSaveApiAccess = async () => {
    if (!settings) {
      return
    }

    setSaving(true)
    setApiAccessError('')
    try {
      const saved = await settingsStore.saveSettings({
        authMode: settings.authMode,
        apiToken: settings.authMode === 'api_token' ? settings.apiToken ?? '' : '',
      })
      setSettings(saved)
      notify.success('API access settings saved.')
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to save API access settings.',
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setApiAccessError(`${normalized.message}${requestId}`)
    } finally {
      setSaving(false)
    }
  }

  const onCreateApiToken = async () => {
    if (!settings || !canManageApiTokens || !tokenName.trim()) {
      return
    }

    setApiTokenCreating(true)
    setApiAccessError('')
    try {
      const token = await apiClient.createApiToken({
        name: tokenName.trim(),
        permissions: currentPermissions,
      })
      setCreatedApiToken(token.token)
      const saved = await settingsStore.saveSettings({
        authMode: 'api_token',
        apiToken: token.token,
      })
      setSettings(saved)
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

  const onCopyCreatedToken = async () => {
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

  const onTestConnection = async () => {
    setTesting(true)
    setConnectionErrorMessage('')
    try {
      await apiClient.healthCheck()
      notify.success('Connection succeeded.')
    } catch (error) {
      await handleAppError(error, { fallbackMessage: 'Connection failed.' })
    } finally {
      setTesting(false)
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
      const payload = await apiClient.getRuntimeConfig()
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
  }, [apiClient, canReadModuleEnablement])

  React.useEffect(() => {
    if (!isModulesSection || !canReadModuleEnablement) {
      setRuntimeConfig(null)
      setRuntimeConfigLoading(false)
      return
    }
    void loadRuntimeConfig()
  }, [canReadModuleEnablement, isModulesSection, loadRuntimeConfig])

  const onCopyRuntimeConfig = async () => {
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

  const onRapidProFieldMappingChange = React.useCallback((sourceKey: string, rapidProFieldKey: string) => {
    setRapidProMappings((current) => {
      const filtered = current.filter((item) => item.sourceKey !== sourceKey)
      if (!rapidProFieldKey) {
        return filtered
      }
      const sourceLabel = rapidProReporterSourceOptions.find((item) => item.key === sourceKey)?.label ?? sourceKey
      return [...filtered, { sourceKey, sourceLabel, rapidProFieldKey }]
    })
  }, [])

  const onRefreshRapidProFields = async () => {
    if (!canWriteBranding) {
      return
    }
    setRapidProSyncRefreshing(true)
    setRapidProSyncError('')
    try {
      const payload = await apiClient.refreshRapidProReporterSyncFields()
      applyRapidProSyncPayload(payload)
      notify.success('RapidPro fields refreshed.')
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to refresh RapidPro fields.',
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setRapidProSyncError(`${normalized.message}${requestId}`)
    } finally {
      setRapidProSyncRefreshing(false)
    }
  }

  const onSaveRapidProSync = async () => {
    if (!canWriteBranding) {
      return
    }
    setRapidProSyncSaving(true)
    setRapidProSyncError('')
    try {
      const payload = await apiClient.updateRapidProReporterSyncSettings({
        rapidProServerCode,
        mappings: rapidProMappings.map((item) => ({
          sourceKey: item.sourceKey,
          rapidProFieldKey: item.rapidProFieldKey,
        })),
      })
      applyRapidProSyncPayload(payload)
      notify.success('RapidPro reporter sync settings saved.')
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to save RapidPro reporter sync settings.',
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setRapidProSyncError(`${normalized.message}${requestId}`)
    } finally {
      setRapidProSyncSaving(false)
    }
  }

  if (loading || !settings) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
        <CircularProgress size={28} />
      </Box>
    )
  }

  return (
    <Stack spacing={2.5}>
      <Box>
        <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 1 }}>
          <Avatar
            variant="rounded"
            sx={{
              width: 46,
              height: 46,
              bgcolor: 'primary.main',
              color: 'primary.contrastText',
              boxShadow: (theme) => `0 10px 24px ${theme.palette.primary.main}33`,
            }}
          >
            {settingsSectionIcons[currentSection.key]}
          </Avatar>
          <Box>
            <Typography variant="h5" component="h1">
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
              onClick={() => void router.navigate({ to: item.path })}
            >
              {item.label}
            </Button>
          ))}
        </Stack>
      </Box>

      {isGeneralSection ? (
        <>
          <Card id="settings-connection">
            <CardContent>
              <Stack spacing={2}>
                <Typography variant="h6">Connection</Typography>
                <TextField
                  label="API Base URL"
                  value={settings.apiBaseUrl}
                  onChange={(event) => setSettings((prev) => (prev ? { ...prev, apiBaseUrl: event.target.value } : prev))}
                  placeholder="http://127.0.0.1:8080"
                  fullWidth
                />
                <TextField
                  label="Request Timeout (seconds)"
                  type="number"
                  value={settings.requestTimeoutSeconds}
                  onChange={(event) =>
                    setSettings((prev) =>
                      prev
                        ? {
                            ...prev,
                            requestTimeoutSeconds: Number(event.target.value),
                          }
                        : prev,
                    )
                  }
                  inputProps={{ min: 1, max: 300 }}
                  fullWidth
                />
                <Stack direction="row" spacing={1.25} justifyContent="flex-end">
                  <Button variant="outlined" onClick={onTestConnection} disabled={testing || saving}>
                    {testing ? 'Testing...' : 'Test Connection'}
                  </Button>
                  <Button variant="contained" onClick={onSaveConnection} disabled={testing || saving}>
                    {saving ? 'Saving...' : 'Save Connection'}
                  </Button>
                </Stack>
                {connectionErrorMessage ? <Alert severity="error">{connectionErrorMessage}</Alert> : null}
              </Stack>
            </CardContent>
          </Card>

          <Card id="settings-appearance">
            <CardContent>
              <Stack spacing={2}>
                <Typography variant="h6">Appearance</Typography>
                <FormControl size="small" sx={{ maxWidth: 220 }}>
                  <Typography variant="subtitle2" sx={{ mb: 1 }}>
                    Theme mode
                  </Typography>
                  <Select
                    inputProps={{ 'aria-label': 'Theme mode' }}
                    value={prefs.themeMode}
                    onChange={(event) => void setThemeMode(event.target.value as ThemeMode)}
                  >
                    {THEME_MODES.map((mode) => (
                      <MenuItem key={mode} value={mode}>
                        {mode[0].toUpperCase() + mode.slice(1)}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
                <Box>
                  <Typography variant="subtitle2">Palette preset</Typography>
                  <Typography color="text.secondary" sx={{ mb: 1.5 }}>
                    Active preset: {presets.find((preset) => preset.id === prefs.palettePreset)?.label ?? 'Custom'}
                  </Typography>
                  <Stack direction="row" spacing={1.25} alignItems="center" useFlexGap flexWrap="wrap">
                    {presets.slice(0, 4).map((preset) => (
                      <Button
                        key={preset.id}
                        size="small"
                        variant={preset.id === prefs.palettePreset ? 'contained' : 'outlined'}
                        onClick={() => void setPalettePreset(preset.id)}
                      >
                        {preset.label}
                      </Button>
                    ))}
                    <Button variant="text" onClick={() => setAppearanceOpen(true)}>
                      Browse all presets
                    </Button>
                  </Stack>
                </Box>
                <Divider />
                <Typography variant="subtitle2">Navigation</Typography>
                <FormControlLabel
                  control={<Switch checked={prefs.showSukumadMenu} onChange={(event) => void setShowSukumadMenu(event.target.checked)} />}
                  label="Show Sukumad menu group"
                />
                <FormControlLabel
                  control={
                    <Switch
                      checked={prefs.showAdministrationMenu}
                      onChange={(event) => void setShowAdministrationMenu(event.target.checked)}
                    />
                  }
                  label="Show Administration menu group"
                />
                <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5} useFlexGap flexWrap="wrap">
                  {navigationLabelFields.map((field) => (
                    <TextField
                      key={field.id}
                      label={field.label}
                      value={prefs.navLabels[field.id] ?? ''}
                      onChange={(event) => void setNavLabel(field.id, event.target.value)}
                      placeholder="Use default label"
                      size="small"
                      sx={{ minWidth: { xs: '100%', md: 240 } }}
                    />
                  ))}
                </Stack>
                <Divider />
                <Typography variant="subtitle2">Data Grid defaults</Typography>
                <FormControlLabel
                  control={
                    <Switch
                      checked={prefs.pinActionsColumnRight}
                      onChange={(event) => void setPinActionsColumnRight(event.target.checked)}
                    />
                  }
                  label="Pin actions column to the right"
                />
                <TextField
                  label="DataGrid border radius"
                  type="number"
                  value={prefs.dataGridBorderRadius}
                  onChange={(event) => void setDataGridBorderRadius(Number(event.target.value))}
                  inputProps={{ min: 4, max: 32 }}
                  sx={{ maxWidth: 220 }}
                />
              </Stack>
            </CardContent>
          </Card>
        </>
      ) : null}

      {isModulesSection ? (
        <>
          <Card id="settings-modules">
            <CardContent>
              <Stack spacing={2}>
                <Typography variant="h6">Module Enablement</Typography>
                <Typography color="text.secondary">
                  Turn runtime-manageable application modules on or off. Static modules are not listed here.
                </Typography>
                {!canReadModuleEnablement ? <Alert severity="info">You need settings.read permission to view module flags.</Alert> : null}
                {moduleEnablementError ? <Alert severity="error">{moduleEnablementError}</Alert> : null}
                {canReadModuleEnablement ? (
                  moduleEnablementLoading ? (
                    <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
                      <CircularProgress size={24} />
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
                                      onChange={(event) => void onToggleModuleEnablement(module.moduleId, event.target.checked)}
                                    />
                                  }
                                  label={module.enabled ? 'Enabled' : 'Disabled'}
                                />
                              </Stack>
                              <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
                                <Chip
                                  size="small"
                                  label={module.enabled ? 'Enabled' : 'Disabled'}
                                  color={module.enabled ? 'success' : 'default'}
                                />
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
            </CardContent>
          </Card>

          <Card id="settings-runtime-config">
            <CardContent>
              <Stack spacing={2}>
                <Typography variant="h6">Runtime Config</Typography>
                <Typography color="text.secondary">
                  Read the active backend configuration snapshot with sensitive values masked before display.
                </Typography>
                {!canReadModuleEnablement ? <Alert severity="info">You need settings.read permission to view runtime configuration.</Alert> : null}
                {runtimeConfigError ? <Alert severity="error">{runtimeConfigError}</Alert> : null}
                {canReadModuleEnablement ? (
                  <>
                    <Stack direction="row" spacing={1.25} justifyContent="flex-end">
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
                      <Button variant="text" onClick={() => void onCopyRuntimeConfig()} disabled={!runtimeConfigText}>
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
            </CardContent>
          </Card>
        </>
      ) : null}

      {isIntegrationsSection ? (
        <>
          <Card id="settings-api-access">
            <CardContent>
              <Stack spacing={2}>
                <Typography variant="h6">API Access</Typography>
                <Typography color="text.secondary">
                  Choose whether this profile uses the current session or an API token for backend requests.
                </Typography>
                <FormControl>
                  <RadioGroup
                    value={settings.authMode}
                    onChange={(event) =>
                      setSettings((prev) =>
                        prev
                          ? {
                              ...prev,
                              authMode: event.target.value as AppSettings['authMode'],
                            }
                          : prev,
                      )
                    }
                  >
                    <FormControlLabel value="password" control={<Radio />} label="Username / Password" />
                    <FormControlLabel value="api_token" control={<Radio />} label="API Token" />
                  </RadioGroup>
                </FormControl>
                <TextField
                  label="API Token"
                  type="password"
                  value={settings.apiToken ?? ''}
                  onChange={(event) =>
                    setSettings((prev) => (prev ? { ...prev, apiToken: event.target.value } : prev))
                  }
                  helperText="Stored locally for backend API requests."
                  fullWidth
                />
                <Stack direction="row" spacing={1.25} justifyContent="flex-end">
                  <Button variant="contained" onClick={onSaveApiAccess} disabled={saving}>
                    {saving ? 'Saving...' : 'Save API Access'}
                  </Button>
                </Stack>
                {canManageApiTokens ? (
                  <>
                    <Divider />
                    <Typography variant="subtitle2">Create API Token</Typography>
                    <Typography color="text.secondary">
                      The token inherits the current permissions and is shown once after creation.
                    </Typography>
                    <TextField
                      label="Token Name"
                      value={tokenName}
                      onChange={(event) => setTokenName(event.target.value)}
                      fullWidth
                    />
                    <Stack direction="row" spacing={1.25} justifyContent="flex-end">
                      <Button
                        variant="contained"
                        onClick={() => void onCreateApiToken()}
                        disabled={apiTokenCreating || !tokenName.trim()}
                      >
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
                        <Button variant="outlined" onClick={() => void onCopyCreatedToken()}>
                          Copy Token
                        </Button>
                      </Stack>
                    </Stack>
                  </Box>
                ) : null}
              </Stack>
            </CardContent>
          </Card>

          <Card id="settings-rapidpro">
            <CardContent>
              <Stack spacing={2}>
                <Typography variant="h6">RapidPro Reporter Sync</Typography>
                <Typography color="text.secondary">
                  Fetch RapidPro contact fields once, review suggested reporter mappings, and reuse them across manual and scheduled syncs.
                </Typography>
                {!canReadModuleEnablement ? <Alert severity="info">You need settings.read permission to view RapidPro sync settings.</Alert> : null}
                {rapidProSyncError ? <Alert severity="error">{rapidProSyncError}</Alert> : null}
                {canReadModuleEnablement ? (
                  rapidProSyncLoading ? (
                    <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
                      <CircularProgress size={24} />
                    </Box>
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
                      <Stack direction="row" spacing={1.25} alignItems="center" useFlexGap flexWrap="wrap">
                        <Button variant="outlined" onClick={() => void onRefreshRapidProFields()} disabled={!canWriteBranding || rapidProSyncRefreshing}>
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
                          {rapidProReporterSourceOptions.map((option) => {
                            const selected = rapidProMappings.find((item) => item.sourceKey === option.key)?.rapidProFieldKey ?? ''
                            return (
                              <FormControl key={option.key} fullWidth>
                                <Select
                                  displayEmpty
                                  inputProps={{ 'aria-label': option.label }}
                                  value={selected}
                                  disabled={!canWriteBranding || rapidProSyncSaving}
                                  onChange={(event) => onRapidProFieldMappingChange(option.key, event.target.value)}
                                >
                                  <MenuItem value="">
                                    <em>Do not sync</em>
                                  </MenuItem>
                                  {rapidProFields.map((field) => (
                                    <MenuItem key={field.key} value={field.key}>
                                      {field.label}
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
                        <Button variant="contained" onClick={() => void onSaveRapidProSync()} disabled={!canWriteBranding || rapidProSyncSaving}>
                          {rapidProSyncSaving ? 'Saving...' : 'Save RapidPro Sync Settings'}
                        </Button>
                      </Stack>
                    </>
                  )
                ) : null}
              </Stack>
            </CardContent>
          </Card>
        </>
      ) : null}

      {isBrandingSection ? (
        <Card id="settings-branding">
          <CardContent>
            <Stack spacing={2}>
              <Typography variant="h6">Login Branding</Typography>
              <Typography color="text.secondary">
                Configure authentication screen branding used by desktop and web clients.
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
                    <Typography>{brandingDisplayName || 'BasePro'}</Typography>
                  </Stack>
                )}
              </Box>
              {!canWriteBranding ? <Alert severity="info">You need settings.write permission to update branding.</Alert> : null}
              {brandingErrorMessage ? <Alert severity="error">{brandingErrorMessage}</Alert> : null}
              <Stack direction="row" justifyContent="flex-end">
                <Button
                  variant="contained"
                  onClick={onSaveBranding}
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
          </CardContent>
        </Card>
      ) : null}

      {isAboutSection ? (
        <Card id="settings-about">
          <CardContent>
            <Stack spacing={1}>
              <Typography variant="h6">About</Typography>
              <Divider />
              <Typography>App: BasePro Desktop</Typography>
              <Typography color="text.secondary">Desktop version: {import.meta.env.VITE_APP_VERSION ?? '0.0.0-dev'}</Typography>
              <Typography color="text.secondary">
                Backend version:{' '}
                {backendVersionLoading ? 'Checking...' : backendVersion ? backendVersion.version : 'Not Connected'}
              </Typography>
              {backendVersion ? (
                <Typography color="text.secondary">
                  Backend commit/build: {backendVersion.commit} ({backendVersion.buildDate})
                </Typography>
              ) : null}
            </Stack>
          </CardContent>
        </Card>
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
