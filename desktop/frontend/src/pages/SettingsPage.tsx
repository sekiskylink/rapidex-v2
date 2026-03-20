import React from 'react'
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
  Select,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import { useNavigate, useRouter } from '@tanstack/react-router'
import { createApiClient } from '../api/client'
import { useSessionPrincipal } from '../auth/hooks'
import { handleAppError } from '../errors/handleAppError'
import { notify } from '../notifications/facade'
import { hasPermission } from '../rbac/permissions'
import type { ModuleEffectiveConfig } from '../registry/moduleEnablement'
import { moduleRegistry } from '../registry/modules'
import { THEME_MODES, type AppSettings, type ThemeMode } from '../settings/types'
import { PalettePresetPicker } from '../ui/PalettePresetPicker'
import { useThemePreferences } from '../ui/theme'

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
  { id: 'observability', label: 'Observability link' },
] as const

export function SettingsPage() {
  const router = useRouter()
  const navigate = useNavigate()
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
  }, [apiClient, settings?.apiBaseUrl])

  React.useEffect(() => {
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
  }, [apiClient])

  React.useEffect(() => {
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
  }, [apiClient, canReadModuleEnablement])

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
        <Typography variant="h5" component="h1" gutterBottom>
          Settings
        </Typography>
        <Typography color="text.secondary">Manage local desktop connection and appearance preferences.</Typography>
      </Box>

      <Card>
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
            <Box>
              <Typography variant="subtitle2">Auth mode</Typography>
              <Typography color="text.secondary" sx={{ mb: 1 }}>
                {settings.authMode === 'api_token' ? 'API Token' : 'Username / Password'}
              </Typography>
              <Button variant="text" onClick={() => void navigate({ to: '/setup' })}>
                Change in Setup
              </Button>
            </Box>
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

      <Card>
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

      <Card>
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
              ) : (
                runtimeToggleModules.length === 0 ? (
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
              )
            ) : null}
            {canReadModuleEnablement && !canWriteBranding ? (
              <Alert severity="info">You need settings.write permission to change runtime-manageable module flags.</Alert>
            ) : null}
          </Stack>
        </CardContent>
      </Card>

      <Card>
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

      <Card>
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

      <PalettePresetPicker open={appearanceOpen} onClose={() => setAppearanceOpen(false)} />
    </Stack>
  )
}
