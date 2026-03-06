import React from 'react'
import {
  Autocomplete,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import type { GridColDef, GridRenderCellParams } from '@mui/x-data-grid'
import { ApiError } from '../api/client'
import { buildServerQuery, type PaginatedResponse } from '../api/pagination'
import { useApiClient } from '../api/useApiClient'
import { useSessionPrincipal } from '../auth/hooks'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { notify } from '../notifications/store'

interface RoleRow {
  id: number
  name: string
  permissionCount: number
  userCount: number
  createdAt: string
  updatedAt: string
}

interface PermissionRow {
  id: number
  name: string
  moduleScope?: string | null
  createdAt: string
}

interface RoleUserRow {
  id: number
  username: string
  isActive: boolean
}

interface RoleDetail {
  id: number
  name: string
  permissions: PermissionRow[]
  users?: RoleUserRow[]
  createdAt: string
  updatedAt: string
}

function toErrorMessage(error: unknown, fallback: string) {
  if (error instanceof ApiError) {
    return error.message || fallback
  }
  return fallback
}

function permissionLabel(permission: PermissionRow) {
  if (!permission.moduleScope) {
    return permission.name
  }
  return `${permission.name} (${permission.moduleScope})`
}

function formatDate(value: string) {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return value
  }
  return parsed.toLocaleString()
}

export function RolesPage() {
  const apiClient = useApiClient()
  const principal = useSessionPrincipal()
  const canRead = Boolean(principal?.permissions.includes('users.read'))
  const canWrite = Boolean(principal?.permissions.includes('users.write'))

  const [reloadToken, setReloadToken] = React.useState(0)
  const [permissionOptions, setPermissionOptions] = React.useState<PermissionRow[]>([])
  const [loadingPermissions, setLoadingPermissions] = React.useState(false)

  const [createOpen, setCreateOpen] = React.useState(false)
  const [createName, setCreateName] = React.useState('')
  const [createPermissions, setCreatePermissions] = React.useState<PermissionRow[]>([])

  const [editOpen, setEditOpen] = React.useState(false)
  const [editRoleId, setEditRoleId] = React.useState<number | null>(null)
  const [editName, setEditName] = React.useState('')
  const [editPermissions, setEditPermissions] = React.useState<PermissionRow[]>([])

  const [detailsOpen, setDetailsOpen] = React.useState(false)
  const [details, setDetails] = React.useState<RoleDetail | null>(null)

  const [submitting, setSubmitting] = React.useState(false)

  const refreshGrid = React.useCallback(() => setReloadToken((value) => value + 1), [])

  const loadPermissions = React.useCallback(async () => {
    setLoadingPermissions(true)
    try {
      const payload = await apiClient.request<PaginatedResponse<PermissionRow>>(
        '/api/v1/admin/permissions?page=1&pageSize=200&sort=name:asc',
      )
      setPermissionOptions(Array.isArray(payload.items) ? payload.items : [])
    } catch (error) {
      setPermissionOptions([])
      notify({ severity: 'error', message: toErrorMessage(error, 'Unable to load permissions.') })
    } finally {
      setLoadingPermissions(false)
    }
  }, [apiClient])

  const fetchRoles = React.useCallback(
    async (params: AppDataGridFetchParams) => {
      const query = buildServerQuery(params)
      const payload = await apiClient.request<PaginatedResponse<RoleRow>>(`/api/v1/admin/roles?${query}`)
      return {
        rows: Array.isArray(payload.items) ? payload.items : [],
        total: typeof payload.totalCount === 'number' ? payload.totalCount : 0,
      }
    },
    [apiClient],
  )

  const loadRoleDetail = React.useCallback(
    async (roleId: number, includeUsers: boolean) => {
      const includeQuery = includeUsers ? 'true' : 'false'
      return apiClient.request<RoleDetail>(`/api/v1/admin/roles/${roleId}?includeUsers=${includeQuery}`)
    },
    [apiClient],
  )

  const onCreateRole = async () => {
    setSubmitting(true)
    try {
      await apiClient.request('/api/v1/admin/roles', {
        method: 'POST',
        body: JSON.stringify({
          name: createName.trim(),
          permissions: createPermissions.map((permission) => permission.name),
        }),
      })
      notify({ severity: 'success', message: 'Role created.' })
      setCreateOpen(false)
      setCreateName('')
      setCreatePermissions([])
      refreshGrid()
    } catch (error) {
      notify({ severity: 'error', message: toErrorMessage(error, 'Unable to create role.') })
    } finally {
      setSubmitting(false)
    }
  }

  const openEditDialog = React.useCallback(
    async (row: RoleRow) => {
      try {
        const detail = await loadRoleDetail(row.id, false)
        setEditRoleId(detail.id)
        setEditName(detail.name)
        setEditPermissions(detail.permissions)
        setEditOpen(true)
      } catch (error) {
        notify({ severity: 'error', message: toErrorMessage(error, 'Unable to load role details.') })
      }
    },
    [loadRoleDetail],
  )

  const onSaveEdit = async () => {
    if (!editRoleId) {
      return
    }
    setSubmitting(true)
    try {
      await apiClient.request(`/api/v1/admin/roles/${editRoleId}`, {
        method: 'PATCH',
        body: JSON.stringify({
          name: editName.trim(),
          permissions: editPermissions.map((permission) => permission.name),
        }),
      })
      notify({ severity: 'success', message: 'Role updated.' })
      setEditOpen(false)
      setEditRoleId(null)
      setEditName('')
      setEditPermissions([])
      refreshGrid()
    } catch (error) {
      notify({ severity: 'error', message: toErrorMessage(error, 'Unable to update role.') })
    } finally {
      setSubmitting(false)
    }
  }

  const openDetailsDialog = React.useCallback(
    async (row: RoleRow) => {
      try {
        const detail = await loadRoleDetail(row.id, true)
        setDetails(detail)
        setDetailsOpen(true)
      } catch (error) {
        notify({ severity: 'error', message: toErrorMessage(error, 'Unable to load role details.') })
      }
    },
    [loadRoleDetail],
  )

  React.useEffect(() => {
    if (!canRead) {
      return
    }
    void loadPermissions()
  }, [canRead, loadPermissions])

  const columns = React.useMemo<GridColDef<RoleRow>[]>(
    () => [
      { field: 'name', headerName: 'Role', flex: 1, minWidth: 220 },
      { field: 'permissionCount', headerName: 'Permissions', width: 140 },
      { field: 'userCount', headerName: 'Users', width: 120 },
      {
        field: 'updatedAt',
        headerName: 'Updated',
        width: 190,
        valueGetter: (_value, row) => formatDate(row.updatedAt),
      },
      {
        field: 'actions',
        headerName: 'Actions',
        sortable: false,
        filterable: false,
        width: 96,
        renderCell: (params: GridRenderCellParams<RoleRow>) => (
          <AdminRowActions
            rowLabel={params.row.name}
            actions={[
              {
                id: 'view',
                label: 'View Details',
                icon: 'view',
                disabled: !canRead,
                onClick: () => {
                  void openDetailsDialog(params.row)
                },
              },
              {
                id: 'edit',
                label: 'Edit Role',
                icon: 'edit',
                disabled: !canWrite,
                onClick: () => {
                  void openEditDialog(params.row)
                },
              },
            ]}
          />
        ),
      },
    ],
    [canRead, canWrite, openDetailsDialog, openEditDialog],
  )

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 2 }}>
        <Box>
          <Typography variant="h5" component="h1" gutterBottom>
            Roles
          </Typography>
          <Typography color="text.secondary">
            Manage RBAC roles, review assignments, and control role permissions.
          </Typography>
        </Box>
        <Button
          variant="contained"
          disabled={!canWrite}
          onClick={() => {
            setCreateName('')
            setCreatePermissions([])
            setCreateOpen(true)
          }}
        >
          Create Role
        </Button>
      </Box>

      <Box sx={{ height: 620, width: '100%', minWidth: 0, overflow: 'auto' }}>
        <AppDataGrid
          columns={columns}
          fetchData={fetchRoles}
          storageKey="roles-table"
          reloadToken={reloadToken}
          stickyRightFields={['actions']}
        />
      </Box>

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Create Role</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <TextField
              label="Role Name"
              value={createName}
              onChange={(event) => setCreateName(event.target.value)}
              fullWidth
              required
            />
            <Autocomplete
              multiple
              options={permissionOptions}
              loading={loadingPermissions}
              value={createPermissions}
              onChange={(_event, value) => setCreatePermissions(value)}
              getOptionLabel={permissionLabel}
              isOptionEqualToValue={(option, value) => option.id === value.id}
              renderTags={(value, getTagProps) =>
                value.map((option, index) => <Chip label={option.name} {...getTagProps({ index })} key={option.id} size="small" />)
              }
              renderInput={(params) => <TextField {...params} label="Permissions" placeholder="Assign permissions" />}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={() => void onCreateRole()} disabled={submitting || !createName.trim()}>
            Create
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={editOpen} onClose={() => setEditOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Edit Role</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <TextField
              label="Role Name"
              value={editName}
              onChange={(event) => setEditName(event.target.value)}
              fullWidth
              required
            />
            <Autocomplete
              multiple
              options={permissionOptions}
              loading={loadingPermissions}
              value={editPermissions}
              onChange={(_event, value) => setEditPermissions(value)}
              getOptionLabel={permissionLabel}
              isOptionEqualToValue={(option, value) => option.id === value.id}
              renderTags={(value, getTagProps) =>
                value.map((option, index) => <Chip label={option.name} {...getTagProps({ index })} key={option.id} size="small" />)
              }
              renderInput={(params) => <TextField {...params} label="Permissions" placeholder="Assign permissions" />}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={() => void onSaveEdit()} disabled={submitting || !editRoleId || !editName.trim()}>
            Save
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={detailsOpen} onClose={() => setDetailsOpen(false)} fullWidth maxWidth="md">
        <DialogTitle>Role Details</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <Box>
              <Typography variant="subtitle2" color="text.secondary">
                Role
              </Typography>
              <Typography variant="body1">{details?.name ?? '-'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                Assigned Permissions
              </Typography>
              <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
                {(details?.permissions ?? []).length === 0 ? (
                  <Typography color="text.secondary">No permissions assigned.</Typography>
                ) : (
                  details?.permissions.map((permission) => (
                    <Chip key={permission.id} label={permissionLabel(permission)} size="small" />
                  ))
                )}
              </Stack>
            </Box>
            <Box>
              <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                Assigned Users
              </Typography>
              <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
                {(details?.users ?? []).length === 0 ? (
                  <Typography color="text.secondary">No users assigned.</Typography>
                ) : (
                  details?.users?.map((user) => (
                    <Chip
                      key={user.id}
                      label={user.isActive ? user.username : `${user.username} (inactive)`}
                      size="small"
                      variant={user.isActive ? 'filled' : 'outlined'}
                    />
                  ))
                )}
              </Stack>
            </Box>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDetailsOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
