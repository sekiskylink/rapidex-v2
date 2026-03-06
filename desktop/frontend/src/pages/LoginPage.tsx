import React from 'react'
import {
  Alert,
  Button,
  Link,
  Stack,
  TextField,
} from '@mui/material'
import { useNavigate, useRouter } from '@tanstack/react-router'
import { ApiError, createApiClient } from '../api/client'
import { clearSession, setSession, setSessionPrincipal } from '../auth/session'
import { AuthSplitLayout } from './auth/AuthSplitLayout'
import { useAuthBranding } from './auth/useAuthBranding'

export function LoginPage() {
  const router = useRouter()
  const navigate = useNavigate()
  const settingsStore = router.options.context.settingsStore
  const branding = useAuthBranding(settingsStore)

  const [username, setUsername] = React.useState('')
  const [password, setPassword] = React.useState('')
  const [submitting, setSubmitting] = React.useState(false)
  const [errorMessage, setErrorMessage] = React.useState('')

  const apiClient = React.useMemo(
    () =>
      createApiClient({
        getSettings: () => settingsStore.loadSettings(),
      }),
    [settingsStore],
  )

  const onLogin = async (event: React.FormEvent) => {
    event.preventDefault()
    setSubmitting(true)
    setErrorMessage('')

    try {
      const response = await apiClient.login({ username: username.trim(), password })
      await setSession({
        accessToken: response.accessToken,
        refreshToken: response.refreshToken,
        expiresAt: Date.now() + response.expiresIn * 1000,
      })
      const me = await apiClient.me()
      setSessionPrincipal({
        id: me.id,
        username: me.username,
        roles: me.roles ?? [],
        permissions: me.permissions ?? [],
      })
      await navigate({ to: '/dashboard', replace: true })
    } catch (error) {
      await clearSession()
      if (error instanceof ApiError && error.status === 401) {
        setErrorMessage('Invalid username or password.')
      } else {
        setErrorMessage('Unable to sign in right now. Please try again.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthSplitLayout
      branding={branding}
      panelTitle="Welcome back"
      panelSubtitle="Sign in with your platform account to continue."
    >
      <Stack spacing={2.25} component="form" onSubmit={onLogin}>
        <TextField
          label="Username or Email"
          value={username}
          onChange={(event) => setUsername(event.target.value)}
          autoComplete="username"
          required
          fullWidth
          InputProps={{ sx: { minHeight: 56 } }}
        />

        <TextField
          label="Password"
          type="password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          autoComplete="current-password"
          required
          fullWidth
          InputProps={{ sx: { minHeight: 56 } }}
        />

        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Link component="button" type="button" underline="hover" onClick={() => void navigate({ to: '/forgot-password' })}>
            Forgot password?
          </Link>
          <Link component="button" type="button" underline="hover" onClick={() => void navigate({ to: '/setup' })}>
            API settings
          </Link>
        </Stack>

        {errorMessage ? <Alert severity="error">{errorMessage}</Alert> : null}

        <Button type="submit" variant="contained" disabled={submitting} size="large" sx={{ minHeight: 52 }}>
          {submitting ? 'Signing in...' : 'Sign In'}
        </Button>
      </Stack>
    </AuthSplitLayout>
  )
}
