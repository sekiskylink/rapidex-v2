import React from 'react'
import { Alert, Button, Link, Stack, TextField } from '@mui/material'
import { useNavigate } from '@tanstack/react-router'
import { isApiError } from '../auth/AuthProvider'
import { consumeIntendedDestination } from '../auth/sessionExpiry'
import { useAuth } from '../auth/AuthProvider'
import { handleAppError } from '../errors/handleAppError'
import { apiBaseUrl, appName } from '../lib/env'
import { AuthSplitLayout } from './auth/AuthSplitLayout'
import { useAuthBranding } from './auth/useAuthBranding'

export function LoginPage() {
  const navigate = useNavigate()
  const auth = useAuth()
  const branding = useAuthBranding()
  const [username, setUsername] = React.useState('')
  const [password, setPassword] = React.useState('')
  const [submitting, setSubmitting] = React.useState(false)
  const [errorMessage, setErrorMessage] = React.useState('')

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSubmitting(true)
    setErrorMessage('')

    try {
      await auth.login(username, password)
      await navigate({ to: consumeIntendedDestination('/dashboard'), replace: true })
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to sign in right now. Please try again.',
        notifyUser: false,
      })
      if (isApiError(error) && (error.status === 401 || normalized.type === 'unauthorized')) {
        setErrorMessage('Invalid username or password.')
      } else {
        const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
        setErrorMessage(`${normalized.message}${requestId}`)
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthSplitLayout
      branding={{ ...branding, appDisplayName: branding.appDisplayName || appName }}
      panelTitle="Welcome back"
      panelSubtitle="Sign in with your platform account to continue."
    >
      <Stack spacing={2.25} component="form" onSubmit={handleSubmit}>
        {!apiBaseUrl && (
          <Alert severity="warning">
            VITE_API_BASE_URL is not configured. Set it in `web/.env`, for example `http://127.0.0.1:8080/api/v1`. If you have already signed in before, you can also change it in Settings &gt; General &gt; API Base URL Override for this browser.
          </Alert>
        )}
        <TextField
          label="Username or Email"
          autoComplete="username"
          value={username}
          onChange={(event) => setUsername(event.target.value)}
          required
          InputProps={{ sx: { minHeight: 56 } }}
        />
        <TextField
          label="Password"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          required
          InputProps={{ sx: { minHeight: 56 } }}
        />
        <Link component="button" type="button" underline="hover" onClick={() => void navigate({ to: '/forgot-password' })}>
          Forgot password?
        </Link>
        {errorMessage ? <Alert severity="error">{errorMessage}</Alert> : null}
        <Button type="submit" variant="contained" disabled={submitting} size="large" sx={{ minHeight: 52 }}>
          {submitting ? 'Signing in...' : 'Sign In'}
        </Button>
      </Stack>
    </AuthSplitLayout>
  )
}
