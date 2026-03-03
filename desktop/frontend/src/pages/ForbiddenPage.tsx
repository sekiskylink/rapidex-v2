import React from 'react'
import { Box, Paper, Typography } from '@mui/material'

export function ForbiddenPage() {
  return (
    <Box sx={{ display: 'flex', justifyContent: 'center', py: { xs: 6, md: 10 } }}>
      <Paper sx={{ p: 4, maxWidth: 520, width: '100%' }}>
        <Typography variant="h4" component="h1" gutterBottom>
          403
        </Typography>
        <Typography variant="h6" gutterBottom>
          Forbidden
        </Typography>
        <Typography color="text.secondary">
          You do not have permission to access this page.
        </Typography>
      </Paper>
    </Box>
  )
}

