import { Box, Paper, Typography } from '@mui/material'

export function PermissionsPage() {
  return (
    <Box>
      <Paper sx={{ p: 3 }}>
        <Typography variant="h5" component="h1" gutterBottom>
          Permissions
        </Typography>
        <Typography color="text.secondary">
          Permissions administration UI foundation is ready. Backend permission-management endpoints will be connected in the next RBAC administration milestone.
        </Typography>
      </Paper>
    </Box>
  )
}
