import React from 'react'
import { Alert, Button, Link, Stack, TextField } from '@mui/material'
import { useNavigate, useRouter } from '@tanstack/react-router'
import { ApiError, createApiClient } from '../api/client'
import { AuthSplitLayout } from './auth/AuthSplitLayout'
import { useAuthBranding } from './auth/useAuthBranding'

export function ResetPasswordPage() {
  const navigate = useNavigate()
  const router = useRouter()
  const settingsStore = router.options.context.settingsStore
  const branding = useAuthBranding(settingsStore)
  const apiClient = React.useMemo(
    () =>
      createApiClient({
        getSettings: () => settingsStore.loadSettings(),
      }),
    [settingsStore],
  )

  const tokenFromUrl = React.useMemo(() => {
    if (typeof window === 'undefined') {
      return ''
    }
    return new URLSearchParams(window.location.search).get('token')?.trim() ?? ''
  }, [])

  const [token, setToken] = React.useState(tokenFromUrl)
  const [password, setPassword] = React.useState('')
  const [confirmPassword, setConfirmPassword] = React.useState('')
  const [submitting, setSubmitting] = React.useState(false)
  const [success, setSuccess] = React.useState(false)
  const [errorMessage, setErrorMessage] = React.useState('')

  const onSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    setErrorMessage('')

    if (password !== confirmPassword) {
      setErrorMessage('Passwords do not match.')
      return
    }

    setSubmitting(true)
    try {
      await apiClient.resetPassword({ token: token.trim(), password })
      setSuccess(true)
    } catch (error) {
      if (error instanceof ApiError) {
        setErrorMessage(error.message)
      } else {
        setErrorMessage('Unable to reset password right now. Please try again.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthSplitLayout
      branding={branding}
      panelTitle="Reset password"
      panelSubtitle="Create a new password for your account using the reset token."
    >
      <Stack spacing={2.25} component="form" onSubmit={onSubmit}>
        <TextField
          label="Reset Token"
          value={token}
          onChange={(event) => setToken(event.target.value)}
          required
          fullWidth
          InputProps={{ sx: { minHeight: 56 } }}
        />

        <TextField
          label="New Password"
          type="password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          required
          fullWidth
          InputProps={{ sx: { minHeight: 56 } }}
        />

        <TextField
          label="Confirm New Password"
          type="password"
          value={confirmPassword}
          onChange={(event) => setConfirmPassword(event.target.value)}
          required
          fullWidth
          InputProps={{ sx: { minHeight: 56 } }}
        />

        {success ? (
          <Alert severity="success">Password reset successful. You can now sign in with your new password.</Alert>
        ) : null}
        {errorMessage ? <Alert severity="error">{errorMessage}</Alert> : null}

        <Button type="submit" variant="contained" disabled={submitting} size="large" sx={{ minHeight: 52 }}>
          {submitting ? 'Resetting...' : 'Reset Password'}
        </Button>

        <Link component="button" type="button" underline="hover" onClick={() => void navigate({ to: '/login' })}>
          Return to login
        </Link>
      </Stack>
    </AuthSplitLayout>
  )
}
