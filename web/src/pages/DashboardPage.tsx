import React from 'react'
import { Button, Paper, Stack, Typography } from '@mui/material'
import { useNavigate } from '@tanstack/react-router'
import { hasPermission } from '../rbac/permissions'
import { canAccessRoute } from '../navigation'

export function DashboardPage() {
  const navigate = useNavigate()

  const moduleActions = [
    {
      label: 'Users',
      path: '/users',
      enabled: (hasPermission('users.read') || hasPermission('users.write')) && canAccessRoute('/users'),
    },
    {
      label: 'Roles',
      path: '/roles',
      enabled: (hasPermission('users.read') || hasPermission('users.write')) && canAccessRoute('/roles'),
    },
    {
      label: 'Permissions',
      path: '/permissions',
      enabled: (hasPermission('users.read') || hasPermission('users.write')) && canAccessRoute('/permissions'),
    },
    {
      label: 'Audit Log',
      path: '/audit',
      enabled: hasPermission('audit.read') && canAccessRoute('/audit'),
    },
    {
      label: 'Settings',
      path: '/settings',
      enabled: (hasPermission('settings.read') || hasPermission('settings.write')) && canAccessRoute('/settings'),
    },
  ].filter((action) => action.enabled)

  return (
    <Paper elevation={1} sx={{ p: 3 }}>
      <Typography variant="h4" component="h1" gutterBottom>
        Dashboard
      </Typography>
      <Typography color="text.secondary" sx={{ mb: 3 }}>
        Role and module enablement access is enforced by the backend. UI visibility is supplemental.
      </Typography>
      <Stack direction="row" spacing={1.5} useFlexGap flexWrap="wrap">
        {moduleActions.map((item) => (
          <Button
            key={item.label}
            variant="contained"
            onClick={() => void navigate({ to: item.path })}
            disabled={!item.enabled}
          >
            {item.label}
          </Button>
        ))}
      </Stack>
    </Paper>
  )
}
