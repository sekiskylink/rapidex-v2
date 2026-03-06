import React from 'react'
import { Alert, Button, Link, Stack, TextField } from '@mui/material'
import { useNavigate, useRouter } from '@tanstack/react-router'
import { ApiError, createApiClient } from '../api/client'
import { AuthSplitLayout } from './auth/AuthSplitLayout'
import { useAuthBranding } from './auth/useAuthBranding'

const successMessage = 'If the account exists, password reset instructions have been sent.'

export function ForgotPasswordPage() {
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

  const [identifier, setIdentifier] = React.useState('')
  const [submitting, setSubmitting] = React.useState(false)
  const [success, setSuccess] = React.useState(false)
  const [errorMessage, setErrorMessage] = React.useState('')

  const onSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    setSubmitting(true)
    setErrorMessage('')

    try {
      await apiClient.forgotPassword({ identifier: identifier.trim() })
      setSuccess(true)
    } catch (error) {
      if (error instanceof ApiError) {
        setErrorMessage(error.message)
      } else {
        setErrorMessage('Unable to request reset right now. Please try again.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthSplitLayout
      branding={branding}
      panelTitle="Forgot password"
      panelSubtitle="Enter your username or email and we will start the password reset flow."
    >
      <Stack spacing={2.25} component="form" onSubmit={onSubmit}>
        <TextField
          label="Username or Email"
          value={identifier}
          onChange={(event) => setIdentifier(event.target.value)}
          required
          fullWidth
          InputProps={{ sx: { minHeight: 56 } }}
        />

        {success ? <Alert severity="success">{successMessage}</Alert> : null}
        {errorMessage ? <Alert severity="error">{errorMessage}</Alert> : null}

        <Button type="submit" variant="contained" disabled={submitting} size="large" sx={{ minHeight: 52 }}>
          {submitting ? 'Submitting...' : 'Send Reset Instructions'}
        </Button>

        <Link component="button" type="button" underline="hover" onClick={() => void navigate({ to: '/login' })}>
          Back to login
        </Link>
      </Stack>
    </AuthSplitLayout>
  )
}
