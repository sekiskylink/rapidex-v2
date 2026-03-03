import React from 'react'
import { Alert, Container, Stack, Typography } from '@mui/material'

export function RouteErrorPage() {
  return (
    <Container sx={{ py: 8 }}>
      <Stack spacing={2}>
        <Typography variant="h4" component="h1">
          Something went wrong
        </Typography>
        <Alert severity="error">An unexpected routing error occurred.</Alert>
      </Stack>
    </Container>
  )
}
