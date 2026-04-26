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
  type RapidProReporterOption,
  type RapidProReporterSyncPreviewResponse,
  type RapidProReporterSyncSettingsResponse,
  type RapidProReporterSyncValidation,
  type RapidexWebhookDataValueMapping,
  type RapidexWebhookMetadataResponse,
  type RapidexWebhookMetadataSnapshot,
  type RapidexWebhookMappingConfig,
  type RapidexWebhookMappingsExportResponse,
  type RapidexWebhookMappingsSettingsResponse,
  type RapidexWebhookMappingsValidation,
  type RapidexDhis2AttributeOptionComboOption,
  type RapidexDhis2CategoryOptionComboOption,
  type RapidexDhis2DataElementOption,
  type RapidexDhis2DatasetOption,
  type RapidexIntegrationServerOption,
  type RapidexRapidProFlowOption,
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

interface ReporterGroupRecord {
  id: number
  name: string
  isActive: boolean
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
  return isRapidProBuiltInField(field) ? `${field.label} (Built-in target)` : `${field.label} (Custom field)`
}

function formatRapidProPreviewJSON(preview: RapidProReporterSyncPreviewResponse | null) {
  if (!preview) {
    return ''
  }
  return JSON.stringify(preview.requestBody ?? {}, null, 2)
}

function createEmptyRapidexDataValueMapping(): RapidexWebhookDataValueMapping {
  return {
    field: '',
    dataElement: '',
    categoryOptionCombo: '',
    attributeOptionCombo: '',
  }
}

function createEmptyRapidexWebhookMapping(): RapidexWebhookMappingConfig {
  return {
    flowUuid: '',
    flowName: '',
    dataset: '',
    orgUnitVar: '',
    periodVar: '',
    payloadAoc: '',
    mappings: [createEmptyRapidexDataValueMapping()],
  }
}

function createEmptyRapidexMetadataSnapshot(): RapidexWebhookMetadataSnapshot {
  return {
    rapidProServerCode: 'rapidpro',
    dhis2ServerCode: 'dhis2',
    lastRefreshedAt: null,
    rapidProFlows: [],
    rapidProContactFields: [],
    dhis2Datasets: [],
    dhis2DataElements: [],
    dhis2CategoryOptionCombos: [],
    dhis2AttributeOptionCombos: [],
  }
}

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
  const [brandingDisplayName, setBrandingDisplayName] = React.useState('RapidEx')
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
  const [rapidProPreviewReporters, setRapidProPreviewReporters] = React.useState<RapidProReporterOption[]>([])
  const [rapidProPreviewReporterId, setRapidProPreviewReporterId] = React.useState('')
  const [rapidProPreview, setRapidProPreview] = React.useState<RapidProReporterSyncPreviewResponse | null>(null)
  const [rapidProPreviewLoading, setRapidProPreviewLoading] = React.useState(false)
  const [rapidProPreviewError, setRapidProPreviewError] = React.useState('')
  const [rapidexMappings, setRapidexMappings] = React.useState<RapidexWebhookMappingConfig[]>([])
  const [rapidexValidation, setRapidexValidation] = React.useState<RapidexWebhookMappingsValidation>({ isValid: true })
  const [rapidexLoading, setRapidexLoading] = React.useState(true)
  const [rapidexSaving, setRapidexSaving] = React.useState(false)
  const [rapidexImporting, setRapidexImporting] = React.useState(false)
  const [rapidexExporting, setRapidexExporting] = React.useState(false)
  const [rapidexError, setRapidexError] = React.useState('')
  const [rapidexYamlText, setRapidexYamlText] = React.useState('')
  const [rapidexMetadata, setRapidexMetadata] = React.useState<RapidexWebhookMetadataSnapshot>(createEmptyRapidexMetadataSnapshot())
  const [rapidexMetadataWarnings, setRapidexMetadataWarnings] = React.useState<string[]>([])
  const [rapidexRapidProServers, setRapidexRapidProServers] = React.useState<RapidexIntegrationServerOption[]>([])
  const [rapidexDhis2Servers, setRapidexDhis2Servers] = React.useState<RapidexIntegrationServerOption[]>([])
  const [rapidexRapidProServerCode, setRapidexRapidProServerCode] = React.useState('rapidpro')
  const [rapidexDhis2ServerCode, setRapidexDhis2ServerCode] = React.useState('dhis2')
  const [rapidexMetadataRefreshing, setRapidexMetadataRefreshing] = React.useState(false)
  const [reporterGroups, setReporterGroups] = React.useState<ReporterGroupRecord[]>([])
  const [reporterGroupsLoading, setReporterGroupsLoading] = React.useState(true)
  const [reporterGroupsSavingId, setReporterGroupsSavingId] = React.useState<number | null>(null)
  const [reporterGroupsCreating, setReporterGroupsCreating] = React.useState(false)
  const [reporterGroupsError, setReporterGroupsError] = React.useState('')
  const [newReporterGroupName, setNewReporterGroupName] = React.useState('')
  const [newReporterGroupActive, setNewReporterGroupActive] = React.useState(true)
  const rapidProBuiltInFields = React.useMemo(() => rapidProFields.filter((field) => isRapidProBuiltInField(field)), [rapidProFields])
  const rapidProCustomFields = React.useMemo(() => rapidProFields.filter((field) => !isRapidProBuiltInField(field)), [rapidProFields])
  const rapidProPreviewJSON = React.useMemo(() => formatRapidProPreviewJSON(rapidProPreview), [rapidProPreview])
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

  const applyRapidexPayload = React.useCallback((payload: RapidexWebhookMappingsSettingsResponse) => {
    setRapidexRapidProServerCode((payload.rapidProServerCode ?? '').trim() || 'rapidpro')
    setRapidexDhis2ServerCode((payload.dhis2ServerCode ?? '').trim() || 'dhis2')
    setRapidexMappings(payload.mappings ?? [])
    setRapidexValidation(payload.validation ?? { isValid: true })
  }, [])
  const applyRapidexMetadataPayload = React.useCallback((payload: RapidexWebhookMetadataResponse) => {
    setRapidexRapidProServerCode((payload.rapidProServerCode ?? '').trim() || 'rapidpro')
    setRapidexDhis2ServerCode((payload.dhis2ServerCode ?? '').trim() || 'dhis2')
    setRapidexRapidProServers(payload.rapidProServers ?? [])
    setRapidexDhis2Servers(payload.dhis2Servers ?? [])
    setRapidexMetadata(payload.snapshot ?? createEmptyRapidexMetadataSnapshot())
    setRapidexMetadataWarnings(payload.warnings ?? [])
  }, [])
  const rapidexFlowOptions = React.useMemo(() => rapidexMetadata.rapidProFlows ?? [], [rapidexMetadata])
  const rapidexDatasetOptions = React.useMemo(() => rapidexMetadata.dhis2Datasets ?? [], [rapidexMetadata])
  const rapidexDataElementOptions = React.useMemo(() => rapidexMetadata.dhis2DataElements ?? [], [rapidexMetadata])
  const rapidexCOCOptions = React.useMemo(() => rapidexMetadata.dhis2CategoryOptionCombos ?? [], [rapidexMetadata])
  const rapidexAOCOptions = React.useMemo(() => rapidexMetadata.dhis2AttributeOptionCombos ?? [], [rapidexMetadata])

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
        const displayName = (payload.appDisplayName ?? payload.applicationDisplayName ?? '').trim() || 'RapidEx'
        const loginImageUrl = typeof payload.loginImageUrl === 'string' ? payload.loginImageUrl.trim() : ''
        setBrandingDisplayName(displayName)
        setBrandingImageUrl(loginImageUrl)
        setBrandingPreviewBroken(false)
      })
      .catch((error) => {
        if (!active) {
          return
        }
        setBrandingDisplayName('RapidEx')
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
      setRapidexLoading(false)
      setReporterGroupsLoading(false)
      setRapidProFields([])
      setRapidProMappings([])
      setRapidProValidation({ isValid: true })
      setRapidProPreviewReporters([])
      setRapidProPreviewReporterId('')
      setRapidProPreview(null)
      setRapidProPreviewError('')
      setRapidexMappings([])
      setRapidexValidation({ isValid: true })
      setRapidexYamlText('')
      setRapidexMetadata(createEmptyRapidexMetadataSnapshot())
      setRapidexMetadataWarnings([])
      setRapidexRapidProServers([])
      setRapidexDhis2Servers([])
      setReporterGroups([])
      setReporterGroupsError('')
      return
    }
    if (!canReadModuleEnablement) {
      setRapidProSyncLoading(false)
      setRapidexLoading(false)
      setReporterGroupsLoading(false)
      setRapidProFields([])
      setRapidProMappings([])
      setRapidProValidation({ isValid: true })
      setRapidProPreviewReporters([])
      setRapidProPreviewReporterId('')
      setRapidProPreview(null)
      setRapidProPreviewError('')
      setRapidexMappings([])
      setRapidexValidation({ isValid: true })
      setRapidexYamlText('')
      setRapidexMetadata(createEmptyRapidexMetadataSnapshot())
      setRapidexMetadataWarnings([])
      setRapidexRapidProServers([])
      setRapidexDhis2Servers([])
      setReporterGroups([])
      setReporterGroupsError('')
      return
    }
    let active = true
    setRapidProSyncLoading(true)
    setRapidexLoading(true)
    setReporterGroupsLoading(true)
    Promise.all([
      apiClient.getRapidProReporterSyncSettings(),
      apiClient.listRapidProReporterSyncPreviewReporters(),
      apiClient.getRapidexWebhookMappingsSettings(),
      apiClient.getRapidexWebhookMetadata(),
      apiClient.request<{ items: ReporterGroupRecord[] }>('/api/v1/reporter-groups?page=0&pageSize=200'),
    ])
      .then(([payload, reporterPayload, rapidexPayload, rapidexMetadataPayload, groupPayload]) => {
        if (active) {
          applyRapidProSyncPayload(payload)
          applyRapidexPayload(rapidexPayload)
          applyRapidexMetadataPayload(rapidexMetadataPayload)
          const items = reporterPayload.items ?? []
          setReporterGroups(groupPayload.items ?? [])
          setRapidProPreviewReporters(items)
          setRapidProPreviewReporterId((current) => {
            if (current && items.some((item) => String(item.id) === current)) {
              return current
            }
            return items.length > 0 ? String(items[0].id) : ''
          })
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
          setRapidexLoading(false)
          setReporterGroupsLoading(false)
        }
      })
    return () => {
      active = false
    }
  }, [apiClient, applyRapidProSyncPayload, applyRapidexMetadataPayload, applyRapidexPayload, canReadModuleEnablement, isIntegrationsSection])

  React.useEffect(() => {
    if (!isIntegrationsSection || !canReadModuleEnablement || !rapidProPreviewReporterId) {
      setRapidProPreview(null)
      setRapidProPreviewLoading(false)
      return
    }
    let active = true
    setRapidProPreviewLoading(true)
    setRapidProPreviewError('')
    apiClient
      .getRapidProReporterSyncPreview(Number(rapidProPreviewReporterId))
      .then((payload) => {
        if (active) {
          setRapidProPreview(payload)
        }
      })
      .catch(async (error) => {
        if (!active) {
          return
        }
        setRapidProPreview(null)
        const { error: normalized } = await handleAppError(error, {
          fallbackMessage: 'Unable to load RapidPro sync preview.',
          notifyUser: false,
        })
        const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
        setRapidProPreviewError(`${normalized.message}${requestId}`)
      })
      .finally(() => {
        if (active) {
          setRapidProPreviewLoading(false)
        }
      })
    return () => {
      active = false
    }
  }, [apiClient, canReadModuleEnablement, isIntegrationsSection, rapidProPreviewReporterId])

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
      setBrandingDisplayName((saved.appDisplayName ?? saved.applicationDisplayName ?? '').trim() || 'RapidEx')
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

  const onRapidexMappingChange = React.useCallback(
    (mappingIndex: number, field: keyof RapidexWebhookMappingConfig, value: string) => {
      setRapidexMappings((current) =>
        current.map((item, index) => (index === mappingIndex ? { ...item, [field]: value } : item)),
      )
    },
    [],
  )

  const onRapidexFlowSelection = React.useCallback(
    (mappingIndex: number, flowUUID: string) => {
      const selectedFlow = rapidexFlowOptions.find((item) => item.uuid === flowUUID)
      if (!selectedFlow) {
        return
      }
      setRapidexMappings((current) =>
        current.map((item, index) =>
          index === mappingIndex
            ? {
                ...item,
                flowUuid: selectedFlow.uuid,
                flowName: selectedFlow.name,
              }
            : item,
        ),
      )
    },
    [rapidexFlowOptions],
  )

  const rapidexSourceSuggestionsForMapping = React.useCallback(
    (mapping: RapidexWebhookMappingConfig) => {
      const selectedFlow = rapidexFlowOptions.find((item) => item.uuid === mapping.flowUuid)
      const resultKeys = (selectedFlow?.results ?? []).map((item) => item.key)
      const contactFieldKeys = (rapidexMetadata.rapidProContactFields ?? []).map((item) => item.key)
      return Array.from(new Set([...resultKeys, ...contactFieldKeys].filter((item) => item.trim()))).sort((left, right) =>
        left.localeCompare(right),
      )
    },
    [rapidexFlowOptions, rapidexMetadata.rapidProContactFields],
  )

  const rapidexDataElementSuggestionsForDataset = React.useCallback(
    (datasetID: string) => {
      const selectedDataset = rapidexDatasetOptions.find((item) => item.id === datasetID)
      if (!selectedDataset || selectedDataset.dataElements.length === 0) {
        return rapidexDataElementOptions
      }
      const allowed = new Set(selectedDataset.dataElements.map((item) => item.id))
      return rapidexDataElementOptions.filter((item) => allowed.has(item.id))
    },
    [rapidexDataElementOptions, rapidexDatasetOptions],
  )

  const onRapidexDataValueChange = React.useCallback(
    (mappingIndex: number, rowIndex: number, field: keyof RapidexWebhookDataValueMapping, value: string) => {
      setRapidexMappings((current) =>
        current.map((item, index) =>
          index === mappingIndex
            ? {
                ...item,
                mappings: item.mappings.map((row, currentRowIndex) =>
                  currentRowIndex === rowIndex ? { ...row, [field]: value } : row,
                ),
              }
            : item,
        ),
      )
    },
    [],
  )

  const onAddRapidexMapping = React.useCallback(() => {
    setRapidexMappings((current) => [...current, createEmptyRapidexWebhookMapping()])
  }, [])

  const onRemoveRapidexMapping = React.useCallback((mappingIndex: number) => {
    setRapidexMappings((current) => current.filter((_, index) => index !== mappingIndex))
  }, [])

  const onAddRapidexDataValueRow = React.useCallback((mappingIndex: number) => {
    setRapidexMappings((current) =>
      current.map((item, index) =>
        index === mappingIndex ? { ...item, mappings: [...item.mappings, createEmptyRapidexDataValueMapping()] } : item,
      ),
    )
  }, [])

  const onRemoveRapidexDataValueRow = React.useCallback((mappingIndex: number, rowIndex: number) => {
    setRapidexMappings((current) =>
      current.map((item, index) =>
        index === mappingIndex
          ? { ...item, mappings: item.mappings.filter((_, currentRowIndex) => currentRowIndex !== rowIndex) }
          : item,
      ),
    )
  }, [])

  const onSaveRapidexMappings = async () => {
    if (!canWriteBranding) {
      return
    }
    setRapidexSaving(true)
    setRapidexError('')
    try {
      const payload = await apiClient.updateRapidexWebhookMappingsSettings({
        rapidProServerCode: rapidexRapidProServerCode,
        dhis2ServerCode: rapidexDhis2ServerCode,
        mappings: rapidexMappings,
      })
      applyRapidexPayload(payload)
      notify.success('RapidEx webhook mappings saved.')
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to save RapidEx webhook mappings.',
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setRapidexError(`${normalized.message}${requestId}`)
    } finally {
      setRapidexSaving(false)
    }
  }

  const onRefreshRapidexMetadata = async () => {
    if (!canWriteBranding) {
      return
    }
    setRapidexMetadataRefreshing(true)
    setRapidexError('')
    try {
      const payload = await apiClient.refreshRapidexWebhookMetadata({
        rapidProServerCode: rapidexRapidProServerCode,
        dhis2ServerCode: rapidexDhis2ServerCode,
      })
      applyRapidexMetadataPayload(payload)
      notify.success('RapidEx mapping metadata refreshed.')
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to refresh RapidEx mapping metadata.',
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setRapidexError(`${normalized.message}${requestId}`)
    } finally {
      setRapidexMetadataRefreshing(false)
    }
  }

  const onImportRapidexYaml = async () => {
    if (!canWriteBranding) {
      return
    }
    setRapidexImporting(true)
    setRapidexError('')
    try {
      const payload = await apiClient.importRapidexWebhookMappingsYAML({
        yaml: rapidexYamlText,
      })
      applyRapidexPayload(payload)
      notify.success('RapidEx webhook mappings imported.')
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to import RapidEx webhook mappings YAML.',
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setRapidexError(`${normalized.message}${requestId}`)
    } finally {
      setRapidexImporting(false)
    }
  }

  const onExportRapidexYaml = async () => {
    setRapidexExporting(true)
    setRapidexError('')
    try {
      const payload: RapidexWebhookMappingsExportResponse = await apiClient.exportRapidexWebhookMappingsYAML()
      setRapidexYamlText(payload.yaml ?? '')
      notify.success('RapidEx webhook mappings YAML exported.')
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to export RapidEx webhook mappings YAML.',
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setRapidexError(`${normalized.message}${requestId}`)
    } finally {
      setRapidexExporting(false)
    }
  }

  const onCreateReporterGroup = async () => {
    if (!canWriteBranding) {
      return
    }
    setReporterGroupsCreating(true)
    setReporterGroupsError('')
    try {
      const payload = await apiClient.request<ReporterGroupRecord>('/api/v1/reporter-groups', {
        method: 'POST',
        body: JSON.stringify({
          name: newReporterGroupName.trim(),
          isActive: newReporterGroupActive,
        }),
      })
      setReporterGroups((current) => [...current, payload].sort((left, right) => left.name.localeCompare(right.name)))
      setNewReporterGroupName('')
      setNewReporterGroupActive(true)
      notify.success('Reporter group created.')
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to create reporter group.',
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setReporterGroupsError(`${normalized.message}${requestId}`)
    } finally {
      setReporterGroupsCreating(false)
    }
  }

  const onSaveReporterGroup = async (group: ReporterGroupRecord) => {
    if (!canWriteBranding) {
      return
    }
    setReporterGroupsSavingId(group.id)
    setReporterGroupsError('')
    try {
      const payload = await apiClient.request<ReporterGroupRecord>(`/api/v1/reporter-groups/${group.id}`, {
        method: 'PUT',
        body: JSON.stringify({
          name: group.name.trim(),
          isActive: group.isActive,
        }),
      })
      setReporterGroups((current) =>
        current.map((item) => (item.id === payload.id ? payload : item)).sort((left, right) => left.name.localeCompare(right.name)),
      )
      notify.success('Reporter group updated.')
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to save reporter group.',
        notifyUser: false,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setReporterGroupsError(`${normalized.message}${requestId}`)
    } finally {
      setReporterGroupsSavingId(null)
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
                  Map reporter values to RapidPro built-in targets like Contact Name and URNs, or to fetched custom contact fields.
                </Typography>
                {!canReadModuleEnablement ? <Alert severity="info">You need settings.read permission to view RapidPro sync settings.</Alert> : null}
                {rapidProSyncError ? <Alert severity="error">{rapidProSyncError}</Alert> : null}
                {rapidProPreviewError ? <Alert severity="error">{rapidProPreviewError}</Alert> : null}
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
                          <Alert severity="info">
                            Built-in targets are local mapping options. Only custom contact fields are fetched from RapidPro <code>/fields.json</code>.
                          </Alert>
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
                                  {rapidProBuiltInFields.map((field) => (
                                    <MenuItem key={field.key} value={field.key}>
                                      {getRapidProFieldOptionLabel(field)}
                                    </MenuItem>
                                  ))}
                                  {rapidProCustomFields.map((field) => (
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
                      <Divider />
                      <Stack spacing={1.5}>
                        <Typography variant="subtitle1">RapidPro Sync Preview</Typography>
                        {rapidProPreviewReporters.length === 0 ? (
                          <Alert severity="info">No reporters are available yet for preview.</Alert>
                        ) : (
                          <>
                            <TextField
                              select
                              label="Preview Reporter"
                              value={rapidProPreviewReporterId}
                              onChange={(event) => setRapidProPreviewReporterId(event.target.value)}
                              sx={{ maxWidth: 360 }}
                            >
                              {rapidProPreviewReporters.map((reporter) => (
                                <MenuItem key={reporter.id} value={String(reporter.id)}>
                                  {reporter.name}
                                </MenuItem>
                              ))}
                            </TextField>
                            {rapidProPreviewLoading ? (
                              <Typography color="text.secondary">Loading RapidPro sync preview...</Typography>
                            ) : rapidProPreview ? (
                              <Stack spacing={1.5}>
                                <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
                                  <Chip label={`Mode: ${rapidProPreview.reporter.syncOperation}`} size="small" />
                                  <Chip label={`Path: ${rapidProPreview.requestPath}`} size="small" />
                                  <Chip
                                    label={
                                      Object.keys(rapidProPreview.requestQuery ?? {}).length > 0
                                        ? `Query: ${new URLSearchParams(rapidProPreview.requestQuery).toString()}`
                                        : 'Query: none'
                                    }
                                    size="small"
                                  />
                                </Stack>
                                <Typography color="text.secondary">
                                  Previewing {rapidProPreview.reporter.name} ({rapidProPreview.reporter.telephone})
                                </Typography>
                                <TextField
                                  label="Resolved RapidPro Groups"
                                  value={
                                    (rapidProPreview.resolvedGroups ?? []).length > 0
                                      ? (rapidProPreview.resolvedGroups ?? []).map((group) => `${group.name} (${group.uuid})`).join(', ')
                                      : 'None'
                                  }
                                  InputProps={{ readOnly: true }}
                                  fullWidth
                                />
                                <TextField
                                  label="RapidPro Sync Preview JSON"
                                  value={rapidProPreviewJSON}
                                  InputProps={{ readOnly: true, sx: { fontFamily: 'monospace' } }}
                                  multiline
                                  minRows={10}
                                  fullWidth
                                />
                              </Stack>
                            ) : null}
                          </>
                        )}
                      </Stack>
                      <Divider />
                      <Stack spacing={1.5}>
                        <Typography variant="subtitle1">Reporter Groups</Typography>
                        <Typography color="text.secondary">
                          RapidEx manages the canonical reporter group catalog. Active groups are selectable on reporter forms and are created in RapidPro automatically.
                        </Typography>
                        {reporterGroupsError ? <Alert severity="error">{reporterGroupsError}</Alert> : null}
                        {reporterGroupsLoading ? (
                          <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
                            <CircularProgress size={24} />
                          </Box>
                        ) : (
                          <Stack spacing={1.25}>
                            {reporterGroups.length === 0 ? <Alert severity="info">No reporter groups have been created yet.</Alert> : null}
                            {reporterGroups.map((group) => (
                              <Stack key={group.id} direction={{ xs: 'column', md: 'row' }} spacing={1.25} alignItems={{ md: 'center' }}>
                                <TextField
                                  label={`Reporter Group ${group.id}`}
                                  value={group.name}
                                  onChange={(event) =>
                                    setReporterGroups((current) =>
                                      current.map((item) => (item.id === group.id ? { ...item, name: event.target.value } : item)),
                                    )
                                  }
                                  fullWidth
                                />
                                <FormControlLabel
                                  control={
                                    <Switch
                                      checked={group.isActive}
                                      onChange={(event) =>
                                        setReporterGroups((current) =>
                                          current.map((item) => (item.id === group.id ? { ...item, isActive: event.target.checked } : item)),
                                        )
                                      }
                                    />
                                  }
                                  label={group.isActive ? 'Active' : 'Inactive'}
                                />
                                <Button
                                  variant="outlined"
                                  onClick={() => void onSaveReporterGroup(group)}
                                  disabled={!canWriteBranding || reporterGroupsSavingId === group.id || !group.name.trim()}
                                >
                                  {reporterGroupsSavingId === group.id ? 'Saving...' : 'Save Group'}
                                </Button>
                              </Stack>
                            ))}
                            <Divider />
                            <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.25} alignItems={{ md: 'center' }}>
                              <TextField
                                label="New Reporter Group"
                                value={newReporterGroupName}
                                onChange={(event) => setNewReporterGroupName(event.target.value)}
                                fullWidth
                              />
                              <FormControlLabel
                                control={<Switch checked={newReporterGroupActive} onChange={(event) => setNewReporterGroupActive(event.target.checked)} />}
                                label={newReporterGroupActive ? 'Active' : 'Inactive'}
                              />
                              <Button
                                variant="contained"
                                onClick={() => void onCreateReporterGroup()}
                                disabled={!canWriteBranding || reporterGroupsCreating || !newReporterGroupName.trim()}
                              >
                                {reporterGroupsCreating ? 'Creating...' : 'Create Group'}
                              </Button>
                            </Stack>
                          </Stack>
                        )}
                      </Stack>
                      <Divider />
                      <Stack spacing={1.5}>
                        <Typography variant="subtitle1">RapidEx Webhook Mappings</Typography>
                        <Typography color="text.secondary">
                          Manage RapidPro flow-to-DHIS2 aggregate mappings from the desktop UI. The database is the live source of truth.
                        </Typography>
                        {rapidexError ? <Alert severity="error">{rapidexError}</Alert> : null}
                        {rapidexLoading ? (
                          <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
                            <CircularProgress size={24} />
                          </Box>
                        ) : (
                          <>
                            <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.25}>
                              <TextField
                                select
                                label="RapidPro Server"
                                value={rapidexRapidProServerCode}
                                onChange={(event) => setRapidexRapidProServerCode(event.target.value)}
                                fullWidth
                              >
                                {(rapidexRapidProServers.length > 0 ? rapidexRapidProServers : [{ code: rapidexRapidProServerCode, name: rapidexRapidProServerCode, systemType: 'rapidpro', suspended: false }]).map((item) => (
                                  <MenuItem key={item.code} value={item.code}>
                                    {item.name} ({item.code}){item.suspended ? ' [Suspended]' : ''}
                                  </MenuItem>
                                ))}
                              </TextField>
                              <TextField
                                select
                                label="DHIS2 Server"
                                value={rapidexDhis2ServerCode}
                                onChange={(event) => setRapidexDhis2ServerCode(event.target.value)}
                                fullWidth
                              >
                                {(rapidexDhis2Servers.length > 0 ? rapidexDhis2Servers : [{ code: rapidexDhis2ServerCode, name: rapidexDhis2ServerCode, systemType: 'dhis2', suspended: false }]).map((item) => (
                                  <MenuItem key={item.code} value={item.code}>
                                    {item.name} ({item.code}){item.suspended ? ' [Suspended]' : ''}
                                  </MenuItem>
                                ))}
                              </TextField>
                            </Stack>
                            <Stack direction="row" spacing={1.25} useFlexGap flexWrap="wrap" alignItems="center">
                              <Button variant="outlined" onClick={() => void onRefreshRapidexMetadata()} disabled={!canWriteBranding || rapidexMetadataRefreshing}>
                                {rapidexMetadataRefreshing ? 'Refreshing...' : 'Refresh Metadata'}
                              </Button>
                              <Typography color="text.secondary" variant="body2">
                                Last refreshed: {rapidexMetadata.lastRefreshedAt ? new Date(rapidexMetadata.lastRefreshedAt).toLocaleString() : 'Not yet refreshed'}
                              </Typography>
                            </Stack>
                            {rapidexMetadataWarnings.length > 0 ? <Alert severity="warning">{rapidexMetadataWarnings.join(' ')}</Alert> : null}
                            <TextField
                              label="RapidEx Mapping YAML"
                              value={rapidexYamlText}
                              onChange={(event) => setRapidexYamlText(event.target.value)}
                              multiline
                              minRows={8}
                              fullWidth
                              InputProps={{ sx: { fontFamily: 'monospace' } }}
                            />
                            <Stack direction="row" spacing={1.25} useFlexGap flexWrap="wrap">
                              <Button variant="outlined" onClick={() => void onExportRapidexYaml()} disabled={rapidexExporting}>
                                {rapidexExporting ? 'Exporting...' : 'Export YAML'}
                              </Button>
                              <Button
                                variant="outlined"
                                onClick={() => void onImportRapidexYaml()}
                                disabled={!canWriteBranding || rapidexImporting || !rapidexYamlText.trim()}
                              >
                                {rapidexImporting ? 'Importing...' : 'Import YAML'}
                              </Button>
                              <Button variant="text" onClick={onAddRapidexMapping} disabled={!canWriteBranding}>
                                Add Flow Mapping
                              </Button>
                            </Stack>
                            {rapidexMappings.length === 0 ? (
                              <Alert severity="info">No RapidEx webhook mappings have been saved yet.</Alert>
                            ) : (
                              <Stack spacing={2}>
                                {rapidexMappings.map((mapping, mappingIndex) => (
                                  <Card key={`${mapping.flowUuid || 'new'}-${mappingIndex}`} variant="outlined">
                                    <CardContent>
                                      <Stack spacing={1.25}>
                                        <Stack direction="row" justifyContent="space-between" alignItems="center" useFlexGap flexWrap="wrap">
                                          <Typography variant="subtitle2">Flow Mapping {mappingIndex + 1}</Typography>
                                          <Button
                                            variant="text"
                                            color="error"
                                            onClick={() => onRemoveRapidexMapping(mappingIndex)}
                                            disabled={!canWriteBranding}
                                          >
                                            Remove Flow
                                          </Button>
                                        </Stack>
                                        <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.25}>
                                          <TextField
                                            select
                                            label="Discovered Flow"
                                            value={mapping.flowUuid}
                                            onChange={(event) => onRapidexFlowSelection(mappingIndex, event.target.value)}
                                            fullWidth
                                          >
                                            <MenuItem value="">Select a discovered flow</MenuItem>
                                            {rapidexFlowOptions.map((item) => (
                                              <MenuItem key={item.uuid} value={item.uuid}>
                                                {item.name} ({item.uuid})
                                              </MenuItem>
                                            ))}
                                          </TextField>
                                        </Stack>
                                        <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.25}>
                                          <TextField
                                            label="Flow UUID"
                                            value={mapping.flowUuid}
                                            onChange={(event) => onRapidexMappingChange(mappingIndex, 'flowUuid', event.target.value)}
                                            fullWidth
                                          />
                                          <TextField
                                            label="Flow Name"
                                            value={mapping.flowName ?? ''}
                                            onChange={(event) => onRapidexMappingChange(mappingIndex, 'flowName', event.target.value)}
                                            fullWidth
                                          />
                                        </Stack>
                                        <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.25}>
                                          <TextField
                                            label="Dataset"
                                            value={mapping.dataset}
                                            onChange={(event) => onRapidexMappingChange(mappingIndex, 'dataset', event.target.value)}
                                            inputProps={{ list: `desktop-rapidex-datasets-${mappingIndex}` }}
                                            fullWidth
                                          />
                                          <datalist id={`desktop-rapidex-datasets-${mappingIndex}`}>
                                            {rapidexDatasetOptions.map((item) => (
                                              <option key={item.id} value={item.id}>{item.name}</option>
                                            ))}
                                          </datalist>
                                          <TextField
                                            label="Org Unit Variable"
                                            value={mapping.orgUnitVar}
                                            onChange={(event) => onRapidexMappingChange(mappingIndex, 'orgUnitVar', event.target.value)}
                                            inputProps={{ list: `desktop-rapidex-source-fields-org-${mappingIndex}` }}
                                            fullWidth
                                          />
                                          <TextField
                                            label="Period Variable"
                                            value={mapping.periodVar}
                                            onChange={(event) => onRapidexMappingChange(mappingIndex, 'periodVar', event.target.value)}
                                            inputProps={{ list: `desktop-rapidex-source-fields-period-${mappingIndex}` }}
                                            fullWidth
                                          />
                                          <TextField
                                            label="Payload AOC"
                                            value={mapping.payloadAoc ?? ''}
                                            onChange={(event) => onRapidexMappingChange(mappingIndex, 'payloadAoc', event.target.value)}
                                            inputProps={{ list: `desktop-rapidex-aoc-${mappingIndex}` }}
                                            fullWidth
                                          />
                                          <datalist id={`desktop-rapidex-source-fields-org-${mappingIndex}`}>
                                            {rapidexSourceSuggestionsForMapping(mapping).map((item) => (
                                              <option key={item} value={item} />
                                            ))}
                                          </datalist>
                                          <datalist id={`desktop-rapidex-source-fields-period-${mappingIndex}`}>
                                            {rapidexSourceSuggestionsForMapping(mapping).map((item) => (
                                              <option key={item} value={item} />
                                            ))}
                                          </datalist>
                                          <datalist id={`desktop-rapidex-aoc-${mappingIndex}`}>
                                            {rapidexAOCOptions.map((item) => (
                                              <option key={item.id} value={item.id}>{item.name}</option>
                                            ))}
                                          </datalist>
                                        </Stack>
                                        <Divider />
                                        <Typography variant="body2" color="text.secondary">
                                          Data value mappings
                                        </Typography>
                                        <Stack spacing={1.25}>
                                          {mapping.mappings.map((item, rowIndex) => (
                                            <Stack key={`${mappingIndex}-${rowIndex}`} spacing={1}>
                                              <Stack direction={{ xs: 'column', md: 'row' }} spacing={1}>
                                                <TextField
                                                  label="Field"
                                                  value={item.field}
                                                  onChange={(event) => onRapidexDataValueChange(mappingIndex, rowIndex, 'field', event.target.value)}
                                                  inputProps={{ list: `desktop-rapidex-source-fields-row-${mappingIndex}-${rowIndex}` }}
                                                  fullWidth
                                                />
                                                <TextField
                                                  label="Data Element"
                                                  value={item.dataElement}
                                                  onChange={(event) =>
                                                    onRapidexDataValueChange(mappingIndex, rowIndex, 'dataElement', event.target.value)
                                                  }
                                                  inputProps={{ list: `desktop-rapidex-data-elements-${mappingIndex}-${rowIndex}` }}
                                                  fullWidth
                                                />
                                                <TextField
                                                  label="Category Option Combo"
                                                  value={item.categoryOptionCombo ?? ''}
                                                  onChange={(event) =>
                                                    onRapidexDataValueChange(mappingIndex, rowIndex, 'categoryOptionCombo', event.target.value)
                                                  }
                                                  inputProps={{ list: `desktop-rapidex-coc-${mappingIndex}-${rowIndex}` }}
                                                  fullWidth
                                                />
                                                <TextField
                                                  label="Attribute Option Combo"
                                                  value={item.attributeOptionCombo ?? ''}
                                                  onChange={(event) =>
                                                    onRapidexDataValueChange(mappingIndex, rowIndex, 'attributeOptionCombo', event.target.value)
                                                  }
                                                  inputProps={{ list: `desktop-rapidex-aoc-row-${mappingIndex}-${rowIndex}` }}
                                                  fullWidth
                                                />
                                                <datalist id={`desktop-rapidex-source-fields-row-${mappingIndex}-${rowIndex}`}>
                                                  {rapidexSourceSuggestionsForMapping(mapping).map((value) => (
                                                    <option key={value} value={value} />
                                                  ))}
                                                </datalist>
                                                <datalist id={`desktop-rapidex-data-elements-${mappingIndex}-${rowIndex}`}>
                                                  {rapidexDataElementSuggestionsForDataset(mapping.dataset).map((value) => (
                                                    <option key={value.id} value={value.id}>{value.name}</option>
                                                  ))}
                                                </datalist>
                                                <datalist id={`desktop-rapidex-coc-${mappingIndex}-${rowIndex}`}>
                                                  {rapidexCOCOptions.map((value) => (
                                                    <option key={value.id} value={value.id}>{value.name}</option>
                                                  ))}
                                                </datalist>
                                                <datalist id={`desktop-rapidex-aoc-row-${mappingIndex}-${rowIndex}`}>
                                                  {rapidexAOCOptions.map((value) => (
                                                    <option key={value.id} value={value.id}>{value.name}</option>
                                                  ))}
                                                </datalist>
                                              </Stack>
                                              <Stack direction="row" justifyContent="flex-end">
                                                <Button
                                                  variant="text"
                                                  color="error"
                                                  onClick={() => onRemoveRapidexDataValueRow(mappingIndex, rowIndex)}
                                                  disabled={!canWriteBranding || mapping.mappings.length <= 1}
                                                >
                                                  Remove Row
                                                </Button>
                                              </Stack>
                                            </Stack>
                                          ))}
                                          <Stack direction="row" justifyContent="flex-end">
                                            <Button variant="text" onClick={() => onAddRapidexDataValueRow(mappingIndex)} disabled={!canWriteBranding}>
                                              Add Data Value Row
                                            </Button>
                                          </Stack>
                                        </Stack>
                                      </Stack>
                                    </CardContent>
                                  </Card>
                                ))}
                              </Stack>
                            )}
                          </>
                        )}
                      </Stack>
                      {!rapidProValidation.isValid ? (
                        <Alert severity="warning">{(rapidProValidation.errors ?? []).join(' ')}</Alert>
                      ) : (
                        <Alert severity="success">Saved RapidPro mappings are valid and ready for sync.</Alert>
                      )}
                      {!rapidexValidation.isValid ? (
                        <Alert severity="warning">{(rapidexValidation.errors ?? []).join(' ')}</Alert>
                      ) : (
                        <Alert severity="success">Saved RapidEx webhook mappings are valid and ready for webhook processing.</Alert>
                      )}
                      {!canWriteBranding ? <Alert severity="info">You need settings.write permission to change RapidPro sync settings.</Alert> : null}
                      <Stack direction="row" spacing={1.25} justifyContent="flex-end">
                        <Button variant="outlined" onClick={() => void onSaveRapidProSync()} disabled={!canWriteBranding || rapidProSyncSaving}>
                          {rapidProSyncSaving ? 'Saving...' : 'Save RapidPro Sync Settings'}
                        </Button>
                        <Button variant="contained" onClick={() => void onSaveRapidexMappings()} disabled={!canWriteBranding || rapidexSaving}>
                          {rapidexSaving ? 'Saving...' : 'Save RapidEx Webhook Mappings'}
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
                    <Typography>{brandingDisplayName || 'RapidEx'}</Typography>
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
              <Typography>App: RapidEx</Typography>
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
