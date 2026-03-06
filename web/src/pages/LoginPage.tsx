import React from 'react'
import { Alert, Button, Link, Stack, TextField } from '@mui/material'
import { useNavigate } from '@tanstack/react-router'
import { isApiError, useAuth } from '../auth/AuthProvider'
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
      await navigate({ to: '/dashboard', replace: true })
    } catch (error) {
      if (isApiError(error)) {
        const requestId = error.requestId ? ` Request ID: ${error.requestId}` : ''
        setErrorMessage(`${error.message}${requestId}`)
      } else {
        setErrorMessage('Unable to sign in right now. Please try again.')
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
        {!apiBaseUrl && <Alert severity="warning">VITE_API_BASE_URL is not configured.</Alert>}
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
