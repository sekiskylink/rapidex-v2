import React from 'react'
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Container,
  FormControl,
  FormControlLabel,
  Radio,
  RadioGroup,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { useNavigate, useRouter } from '@tanstack/react-router'
import { createApiClient } from '../api/client'
import { defaultSettings, type AppSettings } from '../settings/types'

export function SetupPage() {
  const router = useRouter()
  const navigate = useNavigate()
  const settingsStore = router.options.context.settingsStore

  const [settings, setSettings] = React.useState<AppSettings>(defaultSettings)
  const [loading, setLoading] = React.useState(true)
  const [saving, setSaving] = React.useState(false)
  const [testing, setTesting] = React.useState(false)
  const [connectionStatus, setConnectionStatus] = React.useState<{
    severity: 'success' | 'error'
    message: string
  } | null>(null)

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
        getSettings: async () => settings,
      }),
    [settings],
  )

  const onTestConnection = async () => {
    setTesting(true)
    setConnectionStatus(null)
    try {
      await apiClient.healthCheck()
      setConnectionStatus({ severity: 'success', message: 'Connection succeeded.' })
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Connection failed.'
      setConnectionStatus({ severity: 'error', message })
    } finally {
      setTesting(false)
    }
  }

  const onSaveAndContinue = async () => {
    setSaving(true)
    try {
      await settingsStore.saveSettings({
        apiBaseUrl: settings.apiBaseUrl,
        authMode: settings.authMode,
        apiToken: settings.authMode === 'api_token' ? settings.apiToken : '',
        requestTimeoutSeconds: settings.requestTimeoutSeconds,
      })
      await navigate({ to: '/login' })
    } finally {
      setSaving(false)
    }
  }

  return (
    <Container
      maxWidth="sm"
      sx={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        py: 4,
      }}
    >
      <Card sx={{ width: '100%' }}>
        <CardContent>
          <Stack spacing={2.5}>
            <Box>
              <Typography variant="h5" component="h1" gutterBottom>
                Connect to API
              </Typography>
              <Typography color="text.secondary">
                Configure your desktop client before continuing to login.
              </Typography>
            </Box>

            {loading ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
                <CircularProgress size={24} />
              </Box>
            ) : (
              <Stack spacing={2}>
                <TextField
                  label="API Base URL"
                  value={settings.apiBaseUrl}
                  onChange={(event) =>
                    setSettings((prev) => ({ ...prev, apiBaseUrl: event.target.value }))
                  }
                  placeholder="http://127.0.0.1:8080"
                  fullWidth
                />

                <FormControl>
                  <Typography variant="subtitle2" sx={{ mb: 1 }}>
                    Auth Mode
                  </Typography>
                  <RadioGroup
                    value={settings.authMode}
                    onChange={(event) =>
                      setSettings((prev) => ({
                        ...prev,
                        authMode: event.target.value as AppSettings['authMode'],
                      }))
                    }
                  >
                    <FormControlLabel
                      value="password"
                      control={<Radio />}
                      label="Username / Password"
                    />
                    <FormControlLabel value="api_token" control={<Radio />} label="API Token" />
                  </RadioGroup>
                </FormControl>

                {settings.authMode === 'api_token' ? (
                  <TextField
                    label="API Token"
                    type="password"
                    value={settings.apiToken ?? ''}
                    onChange={(event) =>
                      setSettings((prev) => ({ ...prev, apiToken: event.target.value }))
                    }
                    fullWidth
                  />
                ) : null}

                <TextField
                  label="Request Timeout (seconds)"
                  type="number"
                  value={settings.requestTimeoutSeconds}
                  onChange={(event) =>
                    setSettings((prev) => ({
                      ...prev,
                      requestTimeoutSeconds: Number(event.target.value),
                    }))
                  }
                  inputProps={{ min: 1, max: 300 }}
                  fullWidth
                />

                {connectionStatus ? (
                  <Alert severity={connectionStatus.severity}>{connectionStatus.message}</Alert>
                ) : null}

                <Stack direction="row" spacing={1.5} justifyContent="flex-end">
                  <Button variant="outlined" onClick={onTestConnection} disabled={testing || saving}>
                    {testing ? 'Testing...' : 'Test Connection'}
                  </Button>
                  <Button variant="contained" onClick={onSaveAndContinue} disabled={saving || loading}>
                    {saving ? 'Saving...' : 'Save & Continue'}
                  </Button>
                </Stack>
              </Stack>
            )}
          </Stack>
        </CardContent>
      </Card>
    </Container>
  )
}
