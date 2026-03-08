import React from 'react'
import { Alert, Paper, Stack, Typography } from '@mui/material'

interface ModuleDisabledPageProps {
  moduleLabel?: string
}

export function ModuleDisabledPage({ moduleLabel }: ModuleDisabledPageProps) {
  return (
    <Paper elevation={1} sx={{ p: 3 }}>
      <Stack spacing={2}>
        <Typography variant="h4" component="h1">
          Module Disabled
        </Typography>
        <Alert severity="info">
          {moduleLabel ? `${moduleLabel} is currently disabled by platform configuration.` : 'This module is currently disabled by platform configuration.'}
        </Alert>
      </Stack>
    </Paper>
  )
}
