import React from 'react'
import { Box, Paper, Typography } from '@mui/material'

interface ModuleDisabledPageProps {
  moduleLabel?: string
}

export function ModuleDisabledPage({ moduleLabel }: ModuleDisabledPageProps) {
  return (
    <Box sx={{ display: 'flex', justifyContent: 'center', py: { xs: 6, md: 10 } }}>
      <Paper sx={{ p: 4, maxWidth: 560, width: '100%' }}>
        <Typography variant="h4" component="h1" gutterBottom>
          Module Disabled
        </Typography>
        <Typography variant="h6" gutterBottom>
          {moduleLabel ? `${moduleLabel} is unavailable` : 'This module is unavailable'}
        </Typography>
        <Typography color="text.secondary">
          This module is currently disabled by platform configuration.
        </Typography>
      </Paper>
    </Box>
  )
}
