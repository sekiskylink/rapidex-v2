import React from 'react'
import { Container, Typography } from '@mui/material'

export function NotFoundPage() {
  return (
    <Container sx={{ py: 8, textAlign: 'center' }}>
      <Typography variant="h4" component="h1" gutterBottom>
        Not Found
      </Typography>
      <Typography color="text.secondary">The page you requested does not exist.</Typography>
    </Container>
  )
}
