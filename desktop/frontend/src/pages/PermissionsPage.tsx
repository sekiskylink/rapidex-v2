import React from 'react'
import { Alert, Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, Stack, TextField, Typography } from '@mui/material'
import type { GridColDef, GridRenderCellParams } from '@mui/x-data-grid'
import type { PaginatedResponse } from '../api/pagination'
import { useApiClient } from '../api/useApiClient'
import { useSessionPrincipal } from '../auth/hooks'
import { hasAnyPermission } from '../rbac/permissions'
import { getPermissionDefinition } from '../registry/permissions'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { JsonMetadataDialog } from '../components/admin/JsonMetadataDialog'
import { buildAdminListRequestQuery, useAdminListSearch } from '../components/admin/listSearch'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { notify } from '../notifications/facade'

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
  const apiClient = useApiClient()
  const principal = useSessionPrincipal()
  const canRead = hasAnyPermission(principal, ['users.read', 'users.write'])

  const { searchInput, setSearchInput, search } = useAdminListSearch()
  const {
    searchInput: moduleScopeInput,
    setSearchInput: setModuleScopeInput,
    search: moduleScope,
  } = useAdminListSearch()

  const [metadataOpen, setMetadataOpen] = React.useState(false)
  const [selectedMetadata, setSelectedMetadata] = React.useState<unknown>(null)

  const [detailsOpen, setDetailsOpen] = React.useState(false)
  const [selectedPermission, setSelectedPermission] = React.useState<PermissionRow | null>(null)
  const selectedPermissionDefinition = React.useMemo(
    () => (selectedPermission ? getPermissionDefinition(selectedPermission.name) : undefined),
    [selectedPermission],
  )

  const fetchPermissions = React.useCallback(
    async (params: AppDataGridFetchParams) => {
      const query = buildAdminListRequestQuery(params, {
        search,
        extra: {
          moduleScope,
        },
      })
      const payload = await apiClient.request<PaginatedResponse<PermissionRow>>(`/api/v1/admin/permissions?${query}`)
      return {
        rows: Array.isArray(payload.items) ? payload.items : [],
        total: typeof payload.totalCount === 'number' ? payload.totalCount : 0,
      }
    },
    [apiClient, moduleScope, search],
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

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box>
        <Typography variant="h5" component="h1" gutterBottom>
          Permissions
        </Typography>
        <Typography color="text.secondary">
          Browse system permissions, filter by scope, and inspect permission metadata.
        </Typography>
      </Box>

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5}>
        <TextField
          label="Search"
          placeholder="e.g. users.read"
          value={searchInput}
          onChange={(event) => setSearchInput(event.target.value)}
          sx={{ minWidth: 280 }}
        />
        <TextField
          label="Module Scope"
          placeholder="e.g. admin"
          value={moduleScopeInput}
          onChange={(event) => setModuleScopeInput(event.target.value)}
          sx={{ minWidth: 220 }}
        />
      </Stack>

      <Alert severity="info">Permissions for disabled modules are hidden from default administration listings.</Alert>

      <Box sx={{ height: 620, width: '100%', minWidth: 0, overflow: 'hidden' }}>
        <AppDataGrid
          columns={columns}
          fetchData={fetchPermissions}
          storageKey="permissions-table"
          externalQueryKey={`${search}|${moduleScope}`}
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
                Label
              </Typography>
              <Typography>{selectedPermissionDefinition?.label ?? '-'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" color="text.secondary">
                Description
              </Typography>
              <Typography>{selectedPermissionDefinition?.description ?? '-'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" color="text.secondary">
                Module Scope
              </Typography>
              <Typography>{selectedPermission?.moduleScope ?? '-'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" color="text.secondary">
                Registry Module
              </Typography>
              <Typography>{selectedPermissionDefinition?.module ?? '-'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" color="text.secondary">
                Category
              </Typography>
              <Typography>{selectedPermissionDefinition?.category ?? '-'}</Typography>
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
        onCopied={() => notify.success('Permission metadata copied.')}
      />
    </Box>
  )
}
