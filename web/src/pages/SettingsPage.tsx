import React from 'react'
import {
  Alert,
  Box,
  Button,
  Card,
  CardActionArea,
  CardContent,
  Chip,
  Divider,
  FormControlLabel,
  FormLabel,
  Paper,
  Radio,
  RadioGroup,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import { isApiError } from '../auth/AuthProvider'
import { apiRequest } from '../lib/api'
import { appName } from '../lib/env'
import { getApiBaseUrlOverride, setApiBaseUrlOverride } from '../lib/apiBaseUrl'
import { useSnackbar } from '../ui/snackbar'
import { type UiThemeMode } from '../ui/preferences'
import { palettePresets } from '../ui/theme/presets'
import { useUiPreferences } from '../ui/theme/UiPreferencesProvider'

interface HealthResponse {
  status?: string
  version?: string
}

function paletteColorMain(presetId: string, mode: 'light' | 'dark', key: 'primary' | 'secondary') {
  const preset = palettePresets.find((item) => item.id === presetId) ?? palettePresets[0]
  const palette = mode === 'dark' ? preset.palettes.dark : preset.palettes.light
  const color = palette[key]
  if (color && typeof color === 'object' && 'main' in color && typeof color.main === 'string') {
    return color.main
  }
  return 'transparent'
}

function paletteBackgroundDefault(presetId: string, mode: 'light' | 'dark') {
  const preset = palettePresets.find((item) => item.id === presetId) ?? palettePresets[0]
  const palette = mode === 'dark' ? preset.palettes.dark : preset.palettes.light
  return palette.background?.default ?? 'transparent'
}

export function SettingsPage() {
  const { showSnackbar } = useSnackbar()
  const { prefs, resolvedMode, setMode, setPreset, setCollapseNavByDefault } = useUiPreferences()
  const [apiBaseUrlOverride, setApiBaseUrlOverrideValue] = React.useState(() => getApiBaseUrlOverride())
  const [testingConnection, setTestingConnection] = React.useState(false)

  const handleModeChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setMode(event.target.value as UiThemeMode)
  }

  const handleBaseUrlSave = () => {
    setApiBaseUrlOverride(apiBaseUrlOverride)
    showSnackbar({
      severity: 'success',
      message: apiBaseUrlOverride.trim()
        ? 'API base URL override saved.'
        : 'API base URL override cleared. Using default environment URL.',
    })
  }

  const handleTestConnection = async () => {
    setTestingConnection(true)
    try {
      const health = await apiRequest<HealthResponse>('/health', { method: 'GET' }, { withAuth: false, retryOnUnauthorized: false })
      showSnackbar({
        severity: 'success',
        message: `Connection successful (${health.status ?? 'ok'})`,
      })
    } catch (error) {
      if (isApiError(error)) {
        const requestId = error.requestId ? ` Request ID: ${error.requestId}` : ''
        showSnackbar({
          severity: 'error',
          message: `Connection failed: ${error.message}${requestId}`,
          autoHideDuration: 7000,
        })
      } else {
        showSnackbar({
          severity: 'error',
          message: 'Connection failed. Please verify the API base URL and try again.',
          autoHideDuration: 7000,
        })
      }
    } finally {
      setTestingConnection(false)
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

          <Box>
            <FormLabel id="theme-mode-label">Theme Mode</FormLabel>
            <RadioGroup
              row
              aria-labelledby="theme-mode-label"
              name="theme-mode"
              value={prefs.mode}
              onChange={handleModeChange}
            >
              <FormControlLabel value="light" control={<Radio />} label="Light" />
              <FormControlLabel value="dark" control={<Radio />} label="Dark" />
              <FormControlLabel value="system" control={<Radio />} label="System" />
            </RadioGroup>
            <Typography variant="body2" color="text.secondary">
              Active mode: {resolvedMode}
            </Typography>
          </Box>

          <Box>
            <Typography variant="subtitle1" gutterBottom>
              Palette Presets
            </Typography>
            <Stack direction="row" useFlexGap flexWrap="wrap" gap={1.5}>
              {palettePresets.map((preset) => {
                const selected = prefs.preset === preset.id
                const primary = paletteColorMain(preset.id, resolvedMode, 'primary')
                const secondary = paletteColorMain(preset.id, resolvedMode, 'secondary')
                const backgroundDefault = paletteBackgroundDefault(preset.id, resolvedMode)

                return (
                  <Card
                    key={preset.id}
                    variant={selected ? 'elevation' : 'outlined'}
                    elevation={selected ? 4 : 0}
                    sx={{ width: 148 }}
                  >
                    <CardActionArea onClick={() => setPreset(preset.id)} aria-label={`Select ${preset.name} preset`}>
                      <CardContent sx={{ p: 1.25 }}>
                        <Stack spacing={1}>
                          <Box
                            sx={{
                              borderRadius: 1,
                              border: 1,
                              borderColor: 'divider',
                              overflow: 'hidden',
                              display: 'grid',
                              gridTemplateColumns: '1fr 1fr 1fr',
                              height: 26,
                            }}
                          >
                            <Box sx={{ bgcolor: backgroundDefault }} />
                            <Box sx={{ bgcolor: primary }} />
                            <Box sx={{ bgcolor: secondary }} />
                          </Box>
                          <Stack direction="row" alignItems="center" justifyContent="space-between">
                            <Typography variant="body2">{preset.name}</Typography>
                            {selected ? <Chip size="small" label="Active" color="primary" /> : null}
                          </Stack>
                        </Stack>
                      </CardContent>
                    </CardActionArea>
                  </Card>
                )
              })}
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
            label="Collapse side navigation by default"
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
    </Stack>
  )
}
