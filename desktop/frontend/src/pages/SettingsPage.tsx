import React from 'react'
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Divider,
  FormControl,
  MenuItem,
  Select,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { useNavigate, useRouter } from '@tanstack/react-router'
import { createApiClient } from '../api/client'
import { THEME_MODES, type AppSettings, type ThemeMode } from '../settings/types'
import { PalettePresetPicker } from '../ui/PalettePresetPicker'
import { useThemePreferences } from '../ui/theme'

export function SettingsPage() {
  const router = useRouter()
  const navigate = useNavigate()
  const settingsStore = router.options.context.settingsStore
  const { prefs, setThemeMode, setPalettePreset, presets } = useThemePreferences()

  const [loading, setLoading] = React.useState(true)
  const [saving, setSaving] = React.useState(false)
  const [testing, setTesting] = React.useState(false)
  const [appearanceOpen, setAppearanceOpen] = React.useState(false)
  const [settings, setSettings] = React.useState<AppSettings | null>(null)
  const [backendVersion, setBackendVersion] = React.useState<{ version: string; commit: string; buildDate: string } | null>(
    null,
  )
  const [backendVersionLoading, setBackendVersionLoading] = React.useState(true)
  const [status, setStatus] = React.useState<{ severity: 'success' | 'error'; message: string } | null>(null)

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

  const onSaveConnection = async () => {
    if (!settings) {
      return
    }

    setSaving(true)
    setStatus(null)
    try {
      const saved = await settingsStore.saveSettings({
        apiBaseUrl: settings.apiBaseUrl,
        requestTimeoutSeconds: settings.requestTimeoutSeconds,
      })
      setSettings(saved)
      setBackendVersionLoading(true)
      setStatus({ severity: 'success', message: 'Connection settings saved.' })
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to save settings.'
      setStatus({ severity: 'error', message })
    } finally {
      setSaving(false)
    }
  }

  const onTestConnection = async () => {
    setTesting(true)
    setStatus(null)
    try {
      await apiClient.healthCheck()
      setStatus({ severity: 'success', message: 'Connection succeeded.' })
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Connection failed.'
      setStatus({ severity: 'error', message })
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

      {status ? <Alert severity={status.severity}>{status.message}</Alert> : null}

      <PalettePresetPicker open={appearanceOpen} onClose={() => setAppearanceOpen(false)} />
    </Stack>
  )
}
