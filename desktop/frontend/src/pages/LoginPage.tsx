import React from 'react'
import { Card, CardContent, Container, Typography } from '@mui/material'

export function LoginPage() {
  return (
    <Container
      maxWidth="sm"
      sx={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <Card sx={{ width: '100%' }}>
        <CardContent>
          <Typography variant="h5" component="h1" gutterBottom>
            Login
          </Typography>
          <Typography color="text.secondary">Login not implemented yet</Typography>
        </CardContent>
      </Card>
    </Container>
  )
}
