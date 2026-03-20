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
  Select,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import { useAuth } from '../auth/AuthProvider'
import { handleAppError } from '../errors/handleAppError'
import { apiRequest } from '../lib/api'
import { appName } from '../lib/env'
import { getApiBaseUrlOverride, setApiBaseUrlOverride } from '../lib/apiBaseUrl'
import { useAppNotify } from '../notifications/facade'
import type { ModuleEffectiveConfig } from '../registry/moduleEnablement'
import { moduleRegistry } from '../registry/modules'
import { type UiThemeMode } from '../ui/preferences'
import { PaletteRoundedIcon } from '../ui/icons'
import { PalettePresetPicker } from '../ui/theme/PalettePresetPicker'
import { palettePresets } from '../ui/theme/presets'
import { useUiPreferences } from '../ui/theme/UiPreferencesProvider'

interface HealthResponse {
  status?: string
  version?: string
}

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

  React.useEffect(() => {
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
  }, [notify])

  React.useEffect(() => {
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
  }, [canReadModuleEnablement, notify])

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

  return (
    <Stack spacing={3}>
      <Paper elevation={1} sx={{ p: 3 }}>
        <Typography variant="h4" component="h1" gutterBottom>
          Settings
        </Typography>
        <Typography color="text.secondary">Manage local appearance and connection preferences for this browser profile.</Typography>
      </Paper>

      <Paper elevation={1} sx={{ p: 3 }}>
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

      <Paper elevation={1} sx={{ p: 3 }}>
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
            )
          ) : null}
          {canReadModuleEnablement && !canWriteBranding ? (
            <Alert severity="info">You need settings.write permission to change runtime-manageable module flags.</Alert>
          ) : null}
        </Stack>
      </Paper>

      <Paper elevation={1} sx={{ p: 3 }}>
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

      <Paper elevation={1} sx={{ p: 3 }}>
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

      <Paper elevation={1} sx={{ p: 3 }}>
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

      <Paper elevation={1} sx={{ p: 3 }}>
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

      <Paper elevation={1} sx={{ p: 3 }}>
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
      <PalettePresetPicker open={appearanceOpen} onClose={() => setAppearanceOpen(false)} />
    </Stack>
  )
}
