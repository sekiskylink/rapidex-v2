import React from 'react'
import { Alert, Button, Link, Stack, TextField } from '@mui/material'
import { useNavigate } from '@tanstack/react-router'
import { isApiError } from '../auth/AuthProvider'
import { apiRequest } from '../lib/api'
import { AuthSplitLayout } from './auth/AuthSplitLayout'
import { useAuthBranding } from './auth/useAuthBranding'

export function ResetPasswordPage() {
  const navigate = useNavigate()
  const branding = useAuthBranding()
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
      await apiRequest(
        '/auth/reset-password',
        {
          method: 'POST',
          body: JSON.stringify({ token: token.trim(), password }),
        },
        { withAuth: false, retryOnUnauthorized: false },
      )
      setSuccess(true)
    } catch (error) {
      if (isApiError(error)) {
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
