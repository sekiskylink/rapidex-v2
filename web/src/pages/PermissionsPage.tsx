import React from 'react'
import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, Stack, TextField, Typography } from '@mui/material'
import type { GridColDef, GridRenderCellParams } from '@mui/x-data-grid'
import { getAuthSnapshot } from '../auth/state'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { JsonMetadataDialog } from '../components/admin/JsonMetadataDialog'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { apiRequest } from '../lib/api'
import { buildListQuery, type PaginatedResponse } from '../lib/pagination'
import { useSnackbar } from '../ui/snackbar'

interface PermissionRow {
  id: number
  name: string
  moduleScope?: string | null
  createdAt: string
}

function formatDate(value: string) {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return value
  }
  return parsed.toLocaleString()
}

export function PermissionsPage() {
  const { showSnackbar } = useSnackbar()
  const canRead = React.useMemo(() => {
    const user = getAuthSnapshot().user
    if (!user) {
      return false
    }
    return user.permissions.some((permission) => {
      const normalized = permission.trim().toLowerCase()
      return normalized === 'users.read' || normalized === 'users.write'
    })
  }, [])

  const [queryText, setQueryText] = React.useState('')
  const [moduleScope, setModuleScope] = React.useState('')
  const [reloadToken, setReloadToken] = React.useState(0)

  const [metadataOpen, setMetadataOpen] = React.useState(false)
  const [selectedMetadata, setSelectedMetadata] = React.useState<unknown>(null)

  const [detailsOpen, setDetailsOpen] = React.useState(false)
  const [selectedPermission, setSelectedPermission] = React.useState<PermissionRow | null>(null)

  const fetchPermissions = React.useCallback(
    async (params: AppDataGridFetchParams) => {
      const query = new URLSearchParams(buildListQuery(params))
      if (queryText.trim()) {
        query.set('q', queryText.trim())
      }
      if (moduleScope.trim()) {
        query.set('moduleScope', moduleScope.trim())
      }

      const payload = await apiRequest<PaginatedResponse<PermissionRow>>(`/admin/permissions?${query.toString()}`)
      return {
        rows: Array.isArray(payload.items) ? payload.items : [],
        total: typeof payload.totalCount === 'number' ? payload.totalCount : 0,
      }
    },
    [moduleScope, queryText],
  )

  const columns = React.useMemo<GridColDef<PermissionRow>[]>(
    () => [
      { field: 'name', headerName: 'Permission', flex: 1, minWidth: 260 },
      {
        field: 'moduleScope',
        headerName: 'Module Scope',
        width: 170,
        valueGetter: (_value, row) => row.moduleScope ?? '-',
      },
      {
        field: 'createdAt',
        headerName: 'Created',
        width: 190,
        valueGetter: (_value, row) => formatDate(row.createdAt),
      },
      {
        field: 'actions',
        headerName: 'Actions',
        sortable: false,
        filterable: false,
        width: 96,
        renderCell: (params: GridRenderCellParams<PermissionRow>) => (
          <AdminRowActions
            rowLabel={params.row.name}
            actions={[
              {
                id: 'view-details',
                label: 'View Details',
                icon: 'view',
                disabled: !canRead,
                onClick: () => {
                  setSelectedPermission(params.row)
                  setDetailsOpen(true)
                },
              },
              {
                id: 'view-metadata',
                label: 'View Metadata',
                icon: 'view',
                disabled: !canRead,
                onClick: () => {
                  setSelectedMetadata(params.row)
                  setMetadataOpen(true)
                },
              },
            ]}
          />
        ),
      },
    ],
    [canRead],
  )

  const applyFilters = React.useCallback(() => {
    setReloadToken((value) => value + 1)
  }, [])

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box>
        <Typography variant="h5" component="h1" gutterBottom>
          Permissions
        </Typography>
        <Typography color="text.secondary">Browse system permissions, filter by scope, and inspect permission metadata.</Typography>
      </Box>

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5}>
        <TextField
          label="Search"
          placeholder="e.g. users.read"
          value={queryText}
          onChange={(event) => setQueryText(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              applyFilters()
            }
          }}
          onBlur={applyFilters}
          sx={{ minWidth: 280 }}
        />
        <TextField
          label="Module Scope"
          placeholder="e.g. admin"
          value={moduleScope}
          onChange={(event) => setModuleScope(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              applyFilters()
            }
          }}
          onBlur={applyFilters}
          sx={{ minWidth: 220 }}
        />
        <Button variant="outlined" onClick={applyFilters}>
          Apply
        </Button>
      </Stack>

      <Box sx={{ height: 620, width: '100%', minWidth: 0, overflow: 'hidden' }}>
        <AppDataGrid
          columns={columns}
          fetchData={fetchPermissions}
          storageKey="permissions-table"
          reloadToken={reloadToken}
          enablePinnedColumns
          stickyRightFields={['actions']}
        />
      </Box>

      <Dialog open={detailsOpen} onClose={() => setDetailsOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Permission Details</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <Box>
              <Typography variant="subtitle2" color="text.secondary">
                Name
              </Typography>
              <Typography>{selectedPermission?.name ?? '-'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" color="text.secondary">
                Module Scope
              </Typography>
              <Typography>{selectedPermission?.moduleScope ?? '-'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" color="text.secondary">
                Created
              </Typography>
              <Typography>{selectedPermission ? formatDate(selectedPermission.createdAt) : '-'}</Typography>
            </Box>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDetailsOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      <JsonMetadataDialog
        open={metadataOpen}
        metadata={selectedMetadata}
        onClose={() => setMetadataOpen(false)}
        onCopied={() => showSnackbar({ severity: 'success', message: 'Permission metadata copied.' })}
      />
    </Box>
  )
}
