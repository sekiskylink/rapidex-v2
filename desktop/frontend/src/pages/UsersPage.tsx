import React from 'react'
import {
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  FormControlLabel,
  InputLabel,
  MenuItem,
  OutlinedInput,
  Select,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import type { GridColDef, GridRenderCellParams } from '@mui/x-data-grid'
import { ApiError } from '../api/client'
import { buildServerQuery, type PaginatedResponse } from '../api/pagination'
import { useApiClient } from '../api/useApiClient'
import { useSessionPrincipal } from '../auth/hooks'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { notify } from '../notifications/store'

interface UserRow {
  id: number
  username: string
  isActive: boolean
  roles: string[]
  createdAt: string
}

const ROLE_OPTIONS = ['Admin', 'Manager', 'Staff', 'Viewer']

function toErrorMessage(error: unknown, fallback: string) {
  if (error instanceof ApiError) {
    return error.message || fallback
  }
  return fallback
}

export function UsersPage() {
  const apiClient = useApiClient()
  const principal = useSessionPrincipal()
  const canWrite = Boolean(principal?.permissions.includes('users.write'))

  const [reloadToken, setReloadToken] = React.useState(0)

  const [createOpen, setCreateOpen] = React.useState(false)
  const [createUsername, setCreateUsername] = React.useState('')
  const [createPassword, setCreatePassword] = React.useState('')
  const [createRoles, setCreateRoles] = React.useState<string[]>([])
  const [createActive, setCreateActive] = React.useState(true)

  const [resetOpen, setResetOpen] = React.useState(false)
  const [resetUser, setResetUser] = React.useState<UserRow | null>(null)
  const [resetPassword, setResetPassword] = React.useState('')

  const [rolesOpen, setRolesOpen] = React.useState(false)
  const [rolesUser, setRolesUser] = React.useState<UserRow | null>(null)
  const [rolesSelection, setRolesSelection] = React.useState<string[]>([])

  const [submitting, setSubmitting] = React.useState(false)

  const refreshGrid = React.useCallback(() => setReloadToken((value) => value + 1), [])

  const fetchUsers = React.useCallback(
    async (params: AppDataGridFetchParams) => {
      const query = buildServerQuery(params)
      const payload = await apiClient.request<PaginatedResponse<UserRow>>(`/api/v1/users?${query}`)
      return {
        rows: payload.items,
        total: payload.totalCount,
      }
    },
    [apiClient],
  )

  const onCreateUser = async () => {
    setSubmitting(true)
    try {
      await apiClient.request('/api/v1/users', {
        method: 'POST',
        body: JSON.stringify({
          username: createUsername.trim(),
          password: createPassword,
          isActive: createActive,
          roles: createRoles,
        }),
      })
      notify({ severity: 'success', message: 'User created.' })
      setCreateOpen(false)
      setCreateUsername('')
      setCreatePassword('')
      setCreateRoles([])
      setCreateActive(true)
      refreshGrid()
    } catch (error) {
      notify({ severity: 'error', message: toErrorMessage(error, 'Unable to create user.') })
    } finally {
      setSubmitting(false)
    }
  }

  const onToggleActive = async (row: UserRow, nextActive: boolean) => {
    try {
      await apiClient.request(`/api/v1/users/${row.id}`, {
        method: 'PATCH',
        body: JSON.stringify({ isActive: nextActive }),
      })
      notify({ severity: 'success', message: `User ${nextActive ? 'activated' : 'deactivated'}.` })
      refreshGrid()
    } catch (error) {
      notify({ severity: 'error', message: toErrorMessage(error, 'Unable to update active status.') })
    }
  }

  const onResetPassword = async () => {
    if (!resetUser) {
      return
    }
    setSubmitting(true)
    try {
      await apiClient.request(`/api/v1/users/${resetUser.id}/reset-password`, {
        method: 'POST',
        body: JSON.stringify({ password: resetPassword }),
      })
      notify({ severity: 'success', message: 'Password reset.' })
      setResetOpen(false)
      setResetUser(null)
      setResetPassword('')
      refreshGrid()
    } catch (error) {
      notify({ severity: 'error', message: toErrorMessage(error, 'Unable to reset password.') })
    } finally {
      setSubmitting(false)
    }
  }

  const onSaveRoles = async () => {
    if (!rolesUser) {
      return
    }
    setSubmitting(true)
    try {
      await apiClient.request(`/api/v1/users/${rolesUser.id}`, {
        method: 'PATCH',
        body: JSON.stringify({ roles: rolesSelection }),
      })
      notify({ severity: 'success', message: 'Roles updated.' })
      setRolesOpen(false)
      setRolesUser(null)
      setRolesSelection([])
      refreshGrid()
    } catch (error) {
      notify({ severity: 'error', message: toErrorMessage(error, 'Unable to update roles.') })
    } finally {
      setSubmitting(false)
    }
  }

  const columns = React.useMemo<GridColDef<UserRow>[]>(
    () => [
      { field: 'username', headerName: 'Username', flex: 1, minWidth: 180 },
      {
        field: 'roles',
        headerName: 'Roles',
        flex: 1,
        minWidth: 220,
        sortable: false,
        renderCell: (params: GridRenderCellParams<UserRow, string[]>) => (
          <Stack direction="row" spacing={0.5} sx={{ overflow: 'hidden' }}>
            {(params.row.roles || []).map((role) => (
              <Chip key={`${params.row.id}-${role}`} size="small" label={role} />
            ))}
          </Stack>
        ),
      },
      {
        field: 'isActive',
        headerName: 'Active',
        width: 120,
        sortable: false,
        renderCell: (params: GridRenderCellParams<UserRow, boolean>) => (
          <Switch
            checked={Boolean(params.row.isActive)}
            onChange={(event) => void onToggleActive(params.row, event.target.checked)}
            disabled={!canWrite}
            inputProps={{ 'aria-label': `Toggle active ${params.row.username}` }}
          />
        ),
      },
      {
        field: 'createdAt',
        headerName: 'Created',
        width: 190,
        valueGetter: (_value, row) => new Date(row.createdAt).toLocaleString(),
      },
      {
        field: 'actions',
        headerName: 'Actions',
        width: 240,
        sortable: false,
        filterable: false,
        renderCell: (params: GridRenderCellParams<UserRow>) => (
          <Stack direction="row" spacing={1}>
            <Button
              size="small"
              onClick={() => {
                setRolesUser(params.row)
                setRolesSelection(params.row.roles ?? [])
                setRolesOpen(true)
              }}
              disabled={!canWrite}
            >
              Roles
            </Button>
            <Button
              size="small"
              onClick={() => {
                setResetUser(params.row)
                setResetPassword('')
                setResetOpen(true)
              }}
              disabled={!canWrite}
            >
              Reset Password
            </Button>
          </Stack>
        ),
      },
    ],
    [canWrite],
  )

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 2 }}>
        <Box>
          <Typography variant="h5" component="h1" gutterBottom>
            Users
          </Typography>
          <Typography color="text.secondary">Manage users, roles, active state, and password resets.</Typography>
        </Box>
        <Button variant="contained" onClick={() => setCreateOpen(true)} disabled={!canWrite}>
          Create User
        </Button>
      </Box>

      <Box sx={{ height: 620, width: '100%' }}>
        <AppDataGrid
          columns={columns}
          fetchData={fetchUsers}
          storageKey="users-table"
          reloadToken={reloadToken}
        />
      </Box>

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Create User</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <TextField label="Username" value={createUsername} onChange={(event) => setCreateUsername(event.target.value)} fullWidth />
            <TextField
              label="Temp Password"
              type="password"
              value={createPassword}
              onChange={(event) => setCreatePassword(event.target.value)}
              fullWidth
            />
            <FormControl fullWidth>
              <InputLabel id="create-roles-label">Roles</InputLabel>
              <Select
                labelId="create-roles-label"
                multiple
                value={createRoles}
                onChange={(event) => setCreateRoles(event.target.value as string[])}
                input={<OutlinedInput label="Roles" />}
              >
                {ROLE_OPTIONS.map((role) => (
                  <MenuItem key={role} value={role}>
                    {role}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            <FormControlLabel
              control={<Switch checked={createActive} onChange={(event) => setCreateActive(event.target.checked)} />}
              label="Active"
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateOpen(false)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={() => void onCreateUser()}
            disabled={submitting || !createUsername.trim() || !createPassword.trim()}
          >
            Create
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={resetOpen} onClose={() => setResetOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Reset Password</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <Typography color="text.secondary">
              Set a new temporary password for {resetUser?.username ?? 'this user'}.
            </Typography>
            <TextField
              label="New Password"
              type="password"
              value={resetPassword}
              onChange={(event) => setResetPassword(event.target.value)}
              fullWidth
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setResetOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={() => void onResetPassword()} disabled={submitting || !resetPassword.trim()}>
            Reset
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={rolesOpen} onClose={() => setRolesOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Edit Roles</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <Typography color="text.secondary">Update role assignments for {rolesUser?.username ?? 'this user'}.</Typography>
            <FormControl fullWidth>
              <InputLabel id="edit-roles-label">Roles</InputLabel>
              <Select
                labelId="edit-roles-label"
                multiple
                value={rolesSelection}
                onChange={(event) => setRolesSelection(event.target.value as string[])}
                input={<OutlinedInput label="Roles" />}
              >
                {ROLE_OPTIONS.map((role) => (
                  <MenuItem key={role} value={role}>
                    {role}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRolesOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={() => void onSaveRoles()} disabled={submitting}>
            Save
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
