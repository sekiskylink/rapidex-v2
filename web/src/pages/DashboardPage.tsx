import React from 'react'
import { Box, Container, Paper, Typography } from '@mui/material'

export function DashboardPage() {
  return (
    <Container sx={{ py: 6 }}>
      <Paper elevation={1} sx={{ p: 3 }}>
        <Typography variant="h4" component="h1" gutterBottom>
          Dashboard
        </Typography>
        <Typography color="text.secondary">Placeholder web dashboard. Auth and RBAC are added in a later step.</Typography>
      </Paper>
    </Container>
  )
}
