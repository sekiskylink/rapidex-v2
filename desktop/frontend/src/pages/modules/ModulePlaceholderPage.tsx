import React from 'react'
import { Card, CardContent, Stack, Typography } from '@mui/material'

interface ModulePlaceholderPageProps {
  title: string
}

export function ModulePlaceholderPage({ title }: ModulePlaceholderPageProps) {
  return (
    <Card>
      <CardContent>
        <Stack spacing={1}>
          <Typography variant="h5" component="h1">
            {title}
          </Typography>
          <Typography color="text.secondary">Coming soon</Typography>
        </Stack>
      </CardContent>
    </Card>
  )
}
