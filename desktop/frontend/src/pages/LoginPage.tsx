import React from 'react'
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Container,
  Link,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { useNavigate, useRouter } from '@tanstack/react-router'
import { ApiError, createApiClient } from '../api/client'
import { setSession } from '../auth/session'

export function LoginPage() {
  const router = useRouter()
  const navigate = useNavigate()
  const settingsStore = router.options.context.settingsStore

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
      await navigate({ to: '/dashboard', replace: true })
    } catch (error) {
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
      <Card sx={{ width: '100%', maxWidth: 420 }}>
        <CardContent>
          <Stack spacing={2.5} component="form" onSubmit={onLogin}>
            <Box>
              <Typography variant="h5" component="h1" gutterBottom>
                BasePro Desktop
              </Typography>
              <Typography color="text.secondary">Sign in to continue to the dashboard.</Typography>
            </Box>

            <TextField
              label="Username"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              autoComplete="username"
              required
              fullWidth
            />

            <TextField
              label="Password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoComplete="current-password"
              required
              fullWidth
            />

            {errorMessage ? <Alert severity="error">{errorMessage}</Alert> : null}

            <Button type="submit" variant="contained" disabled={submitting}>
              {submitting ? 'Signing in...' : 'Login'}
            </Button>

            <Link
              component="button"
              type="button"
              underline="hover"
              onClick={() => void navigate({ to: '/setup' })}
              sx={{ alignSelf: 'flex-start' }}
            >
              Change API Settings
            </Link>
          </Stack>
        </CardContent>
      </Card>
    </Container>
  )
}
