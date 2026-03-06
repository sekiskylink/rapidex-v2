import { Box, Paper, Typography } from '@mui/material'

export function RolesPage() {
  return (
    <Box>
      <Paper sx={{ p: 3 }}>
        <Typography variant="h5" component="h1" gutterBottom>
          Roles
        </Typography>
        <Typography color="text.secondary">
          Roles administration UI foundation is ready. Backend role-management endpoints will be connected in the next RBAC administration milestone.
        </Typography>
      </Paper>
    </Box>
  )
}
