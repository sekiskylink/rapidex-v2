import React from 'react'
import { Alert, Box, Button, Card, CardContent, Container, Stack, Typography } from '@mui/material'
import { useRouter } from '@tanstack/react-router'
import { createApiClient } from '../api/client'

export function DashboardPage() {
  const router = useRouter()
  const [status, setStatus] = React.useState<string>('')
  const [loading, setLoading] = React.useState(false)

  const apiClient = React.useMemo(
    () =>
      createApiClient({
        getSettings: () => router.options.context.settingsStore.loadSettings(),
      }),
    [router.options.context.settingsStore],
  )

  const onLoadProfile = async () => {
    setLoading(true)
    setStatus('')
    try {
      const me = await apiClient.me()
      setStatus(`Signed in as ${me.username}`)
    } catch {
      setStatus('Request failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Container
      maxWidth="md"
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
          <Stack spacing={2}>
            <Box>
              <Typography variant="h5" component="h1" gutterBottom>
                Dashboard
              </Typography>
              <Typography color="text.secondary">Authenticated placeholder content.</Typography>
            </Box>
            <Box>
              <Button variant="outlined" onClick={onLoadProfile} disabled={loading}>
                {loading ? 'Loading...' : 'Load Profile'}
              </Button>
            </Box>
            {status ? <Alert severity={status.startsWith('Signed in as ') ? 'success' : 'error'}>{status}</Alert> : null}
          </Stack>
        </CardContent>
      </Card>
    </Container>
  )
}
