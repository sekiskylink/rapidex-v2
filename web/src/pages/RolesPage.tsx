import React from 'react'
import {
  Alert,
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
import { getAuthSnapshot } from '../auth/state'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { buildAdminListRequestQuery, useAdminListSearch } from '../components/admin/listSearch'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { handleAppError } from '../errors/handleAppError'
import { apiRequest } from '../lib/api'
import type { PaginatedResponse } from '../lib/pagination'
import { useAppNotify } from '../notifications/facade'

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

function formatDate(value: string) {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return value
  }
  return parsed.toLocaleString()
}

function permissionLabel(permission: PermissionRow) {
  if (!permission.moduleScope) {
    return permission.name
  }
  return `${permission.name} (${permission.moduleScope})`
}

export function RolesPage() {
  const notify = useAppNotify()
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
  const canWrite = React.useMemo(() => {
    const user = getAuthSnapshot().user
    return Boolean(user?.permissions.some((permission) => permission.trim().toLowerCase() === 'users.write'))
  }, [])
  const { searchInput, setSearchInput, search } = useAdminListSearch()

  const [reloadToken, setReloadToken] = React.useState(0)
  const [permissionOptions, setPermissionOptions] = React.useState<PermissionRow[]>([])
  const [loadingPermissions, setLoadingPermissions] = React.useState(false)

  const [createOpen, setCreateOpen] = React.useState(false)
  const [createName, setCreateName] = React.useState('')
  const [createPermissions, setCreatePermissions] = React.useState<PermissionRow[]>([])
  const [createErrorMessage, setCreateErrorMessage] = React.useState('')

  const [editOpen, setEditOpen] = React.useState(false)
  const [editRoleId, setEditRoleId] = React.useState<number | null>(null)
  const [editName, setEditName] = React.useState('')
  const [editPermissions, setEditPermissions] = React.useState<PermissionRow[]>([])
  const [editErrorMessage, setEditErrorMessage] = React.useState('')

  const [detailsOpen, setDetailsOpen] = React.useState(false)
  const [details, setDetails] = React.useState<RoleDetail | null>(null)

  const [submitting, setSubmitting] = React.useState(false)

  const refreshGrid = React.useCallback(() => setReloadToken((value) => value + 1), [])

  const loadPermissions = React.useCallback(async () => {
    setLoadingPermissions(true)
    try {
      const payload = await apiRequest<PaginatedResponse<PermissionRow>>('/admin/permissions?page=1&pageSize=200&sort=name:asc')
      setPermissionOptions(Array.isArray(payload.items) ? payload.items : [])
    } catch (error) {
      setPermissionOptions([])
      await handleAppError(error, {
        fallbackMessage: 'Unable to load permissions.',
        notifier: notify,
      })
    } finally {
      setLoadingPermissions(false)
    }
  }, [notify])

  React.useEffect(() => {
    if (!canRead) {
      return
    }
    void loadPermissions()
  }, [canRead, loadPermissions])

  const fetchRoles = React.useCallback(async (params: AppDataGridFetchParams) => {
    const query = buildAdminListRequestQuery(params, { search })
    const payload = await apiRequest<PaginatedResponse<RoleRow>>(`/admin/roles?${query}`)
    return {
      rows: Array.isArray(payload.items) ? payload.items : [],
      total: typeof payload.totalCount === 'number' ? payload.totalCount : 0,
    }
  }, [search])

  const loadRoleDetail = React.useCallback(async (roleId: number, includeUsers: boolean) => {
    const includeQuery = includeUsers ? 'true' : 'false'
    return apiRequest<RoleDetail>(`/admin/roles/${roleId}?includeUsers=${includeQuery}`)
  }, [])

  const onCreateRole = async () => {
    setSubmitting(true)
    setCreateErrorMessage('')
    try {
      await apiRequest('/admin/roles', {
        method: 'POST',
        body: JSON.stringify({
          name: createName.trim(),
          permissions: createPermissions.map((permission) => permission.name),
        }),
      })
      notify.success('Role created.')
      setCreateOpen(false)
      setCreateName('')
      setCreatePermissions([])
      refreshGrid()
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to create role.',
        notifier: notify,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setCreateErrorMessage(`${normalized.message}${requestId}`)
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
        setEditErrorMessage('')
        setEditOpen(true)
      } catch (error) {
        await handleAppError(error, {
          fallbackMessage: 'Unable to load role details.',
          notifier: notify,
        })
      }
    },
    [loadRoleDetail, notify],
  )

  const onSaveEdit = async () => {
    if (!editRoleId) {
      return
    }
    setSubmitting(true)
    setEditErrorMessage('')
    try {
      await apiRequest(`/admin/roles/${editRoleId}`, {
        method: 'PATCH',
        body: JSON.stringify({
          name: editName.trim(),
          permissions: editPermissions.map((permission) => permission.name),
        }),
      })
      notify.success('Role updated.')
      setEditOpen(false)
      setEditRoleId(null)
      setEditName('')
      setEditPermissions([])
      refreshGrid()
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to update role.',
        notifier: notify,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setEditErrorMessage(`${normalized.message}${requestId}`)
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
        await handleAppError(error, {
          fallbackMessage: 'Unable to load role details.',
          notifier: notify,
        })
      }
    },
    [loadRoleDetail, notify],
  )

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
          <Typography color="text.secondary">Manage RBAC roles, review assignments, and control role permissions.</Typography>
        </Box>
        <Button
          variant="contained"
          disabled={!canWrite}
          onClick={() => {
            setCreateName('')
            setCreatePermissions([])
            setCreateErrorMessage('')
            setCreateOpen(true)
          }}
        >
          Create Role
        </Button>
      </Box>

      <TextField
        label="Search"
        placeholder="Search roles by name"
        value={searchInput}
        onChange={(event) => setSearchInput(event.target.value)}
        sx={{ maxWidth: 360 }}
      />

      <Alert severity="info">Permission assignment options exclude permissions from disabled modules.</Alert>

      <Box sx={{ height: 620, width: '100%', minWidth: 0, overflow: 'hidden' }}>
        <AppDataGrid
          columns={columns}
          fetchData={fetchRoles}
          storageKey="roles-table"
          reloadToken={reloadToken}
          externalQueryKey={search}
          enablePinnedColumns
          stickyRightFields={['actions']}
        />
      </Box>

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Create Role</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            {createErrorMessage ? <Alert severity="error">{createErrorMessage}</Alert> : null}
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
            {editErrorMessage ? <Alert severity="error">{editErrorMessage}</Alert> : null}
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
                  details?.permissions.map((permission) => <Chip key={permission.id} label={permissionLabel(permission)} size="small" />)
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
