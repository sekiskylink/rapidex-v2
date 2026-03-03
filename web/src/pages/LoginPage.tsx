import React from 'react'
import { Alert, Box, Button, Card, CardContent, Stack, TextField, Typography } from '@mui/material'
import { apiBaseUrl, appName } from '../lib/env'

export function LoginPage() {
  return (
    <Box sx={{ minHeight: '100vh', display: 'grid', placeItems: 'center', px: 2 }}>
      <Card sx={{ width: '100%', maxWidth: 420 }}>
        <CardContent>
          <Stack spacing={2}>
            <Typography variant="h5" component="h1">
              {appName}
            </Typography>
            {!apiBaseUrl && <Alert severity="warning">VITE_API_BASE_URL is not configured.</Alert>}
            <TextField label="Username" autoComplete="username" />
            <TextField label="Password" type="password" autoComplete="current-password" />
            <Button variant="contained" disabled>
              Login (coming soon)
            </Button>
          </Stack>
        </CardContent>
      </Card>
    </Box>
  )
}
