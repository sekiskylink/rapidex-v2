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
  FormControlLabel,
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
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { notify } from '../notifications/store'

interface UserRow {
  id: number
  username: string
  email?: string
  language: string
  firstName?: string
  lastName?: string
  displayName?: string
  phoneNumber?: string
  whatsappNumber?: string
  telegramHandle?: string
  isActive: boolean
  roles: string[]
  updatedAt: string
  createdAt: string
}

interface RoleOption {
  id: number
  name: string
}

interface UserFormState {
  username: string
  password: string
  email: string
  language: string
  firstName: string
  lastName: string
  displayName: string
  phoneNumber: string
  whatsappNumber: string
  telegramHandle: string
  isActive: boolean
}

type UserFormErrors = Partial<Record<keyof UserFormState, string>>

const defaultCreateForm: UserFormState = {
  username: '',
  password: '',
  email: '',
  language: 'English',
  firstName: '',
  lastName: '',
  displayName: '',
  phoneNumber: '',
  whatsappNumber: '',
  telegramHandle: '',
  isActive: true,
}

function toErrorMessage(error: unknown, fallback: string) {
  if (error instanceof ApiError) {
    return error.message || fallback
  }
  return fallback
}

function toFieldError(value: unknown) {
  if (typeof value === 'string') {
    return value
  }
  if (Array.isArray(value)) {
    const first = value.find((entry) => typeof entry === 'string')
    return typeof first === 'string' ? first : ''
  }
  return ''
}

function validationFieldErrors(error: unknown): UserFormErrors {
  if (!(error instanceof ApiError) || error.code !== 'VALIDATION_ERROR' || !error.details) {
    return {}
  }

  const details = error.details
  return {
    username: toFieldError(details.username),
    password: toFieldError(details.password),
    email: toFieldError(details.email),
    language: toFieldError(details.language),
    firstName: toFieldError(details.firstName),
    lastName: toFieldError(details.lastName),
    displayName: toFieldError(details.displayName),
    phoneNumber: toFieldError(details.phoneNumber),
    whatsappNumber: toFieldError(details.whatsappNumber),
    telegramHandle: toFieldError(details.telegramHandle),
    isActive: '',
  }
}

function displayNameForRow(row: UserRow) {
  if (row.displayName && row.displayName.trim()) {
    return row.displayName
  }
  const derived = [row.firstName, row.lastName].filter((value) => value && value.trim()).join(' ').trim()
  return derived || row.username
}

function toText(value?: string) {
  return value ?? ''
}

function toUserForm(row: UserRow): UserFormState {
  return {
    username: row.username,
    password: '',
    email: toText(row.email),
    language: toText(row.language),
    firstName: toText(row.firstName),
    lastName: toText(row.lastName),
    displayName: toText(row.displayName),
    phoneNumber: toText(row.phoneNumber),
    whatsappNumber: toText(row.whatsappNumber),
    telegramHandle: toText(row.telegramHandle),
    isActive: row.isActive,
  }
}

function formatDate(value: string) {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return value
  }
  return parsed.toLocaleString()
}

function renderRoleChips(roles: string[]) {
  if (!roles.length) {
    return <Typography color="text.secondary">No roles assigned</Typography>
  }
  return (
    <Stack direction="row" spacing={0.5} useFlexGap flexWrap="wrap" sx={{ py: 0.25 }}>
      {roles.map((role) => (
        <Chip key={role} label={role} size="small" variant="outlined" />
      ))}
    </Stack>
  )
}

export function UsersPage() {
  const apiClient = useApiClient()
  const principal = useSessionPrincipal()
  const canWrite = Boolean(principal?.permissions.includes('users.write'))
  const canReadRoles = Boolean(principal?.permissions.includes('users.read'))

  const [reloadToken, setReloadToken] = React.useState(0)

  const [roleOptions, setRoleOptions] = React.useState<RoleOption[]>([])
  const [loadingRoleOptions, setLoadingRoleOptions] = React.useState(false)

  const [createOpen, setCreateOpen] = React.useState(false)
  const [createForm, setCreateForm] = React.useState<UserFormState>(defaultCreateForm)
  const [createRoles, setCreateRoles] = React.useState<string[]>([])
  const [createErrors, setCreateErrors] = React.useState<UserFormErrors>({})

  const [editOpen, setEditOpen] = React.useState(false)
  const [editUser, setEditUser] = React.useState<UserRow | null>(null)
  const [editForm, setEditForm] = React.useState<UserFormState>(defaultCreateForm)
  const [editRoles, setEditRoles] = React.useState<string[]>([])
  const [editErrors, setEditErrors] = React.useState<UserFormErrors>({})

  const [detailsOpen, setDetailsOpen] = React.useState(false)
  const [detailsUser, setDetailsUser] = React.useState<UserRow | null>(null)

  const [resetOpen, setResetOpen] = React.useState(false)
  const [resetUser, setResetUser] = React.useState<UserRow | null>(null)
  const [resetPassword, setResetPassword] = React.useState('')

  const [rolesOpen, setRolesOpen] = React.useState(false)
  const [rolesUser, setRolesUser] = React.useState<UserRow | null>(null)
  const [rolesSelection, setRolesSelection] = React.useState<string[]>([])

  const [submitting, setSubmitting] = React.useState(false)

  const refreshGrid = React.useCallback(() => setReloadToken((value) => value + 1), [])

  const loadRoleOptions = React.useCallback(async () => {
    if (!canReadRoles) {
      return
    }
    setLoadingRoleOptions(true)
    try {
      const payload = await apiClient.request<PaginatedResponse<RoleOption>>('/api/v1/admin/roles?page=1&pageSize=200&sort=name:asc')
      setRoleOptions(Array.isArray(payload.items) ? payload.items : [])
    } catch {
      setRoleOptions([])
    } finally {
      setLoadingRoleOptions(false)
    }
  }, [apiClient, canReadRoles])

  React.useEffect(() => {
    void loadRoleOptions()
  }, [loadRoleOptions])

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
    setCreateErrors({})
    try {
      await apiClient.request('/api/v1/users', {
        method: 'POST',
        body: JSON.stringify({
          username: createForm.username.trim(),
          password: createForm.password,
          email: createForm.email,
          language: createForm.language,
          firstName: createForm.firstName,
          lastName: createForm.lastName,
          displayName: createForm.displayName,
          phoneNumber: createForm.phoneNumber,
          whatsappNumber: createForm.whatsappNumber,
          telegramHandle: createForm.telegramHandle,
          isActive: createForm.isActive,
          roles: createRoles,
        }),
      })
      notify({ severity: 'success', message: 'User created.' })
      setCreateOpen(false)
      setCreateForm(defaultCreateForm)
      setCreateRoles([])
      refreshGrid()
    } catch (error) {
      const fieldErrors = validationFieldErrors(error)
      if (Object.values(fieldErrors).some((message) => Boolean(message))) {
        setCreateErrors(fieldErrors)
      }
      notify({ severity: 'error', message: toErrorMessage(error, 'Unable to create user.') })
    } finally {
      setSubmitting(false)
    }
  }

  const onSaveEdit = async () => {
    if (!editUser) {
      return
    }

    setSubmitting(true)
    setEditErrors({})

    const payload: Record<string, unknown> = {
      username: editForm.username.trim(),
      email: editForm.email,
      language: editForm.language,
      firstName: editForm.firstName,
      lastName: editForm.lastName,
      displayName: editForm.displayName,
      phoneNumber: editForm.phoneNumber,
      whatsappNumber: editForm.whatsappNumber,
      telegramHandle: editForm.telegramHandle,
      isActive: editForm.isActive,
      roles: editRoles,
    }
    if (editForm.password.trim()) {
      payload.password = editForm.password
    }

    try {
      await apiClient.request(`/api/v1/users/${editUser.id}`, {
        method: 'PATCH',
        body: JSON.stringify(payload),
      })
      notify({ severity: 'success', message: 'User updated.' })
      setEditOpen(false)
      setEditUser(null)
      setEditForm(defaultCreateForm)
      setEditRoles([])
      refreshGrid()
    } catch (error) {
      const fieldErrors = validationFieldErrors(error)
      if (Object.values(fieldErrors).some((message) => Boolean(message))) {
        setEditErrors(fieldErrors)
      }
      notify({ severity: 'error', message: toErrorMessage(error, 'Unable to update user.') })
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

  const roleNames = React.useMemo(() => roleOptions.map((role) => role.name), [roleOptions])

  const columns = React.useMemo<GridColDef<UserRow>[]>(
    () => [
      { field: 'username', headerName: 'Username', flex: 1, minWidth: 160 },
      {
        field: 'displayName',
        headerName: 'Display Name',
        flex: 1,
        minWidth: 200,
        valueGetter: (_value, row) => displayNameForRow(row),
      },
      {
        field: 'roles',
        headerName: 'Roles',
        flex: 1,
        minWidth: 220,
        sortable: false,
        renderCell: (params: GridRenderCellParams<UserRow>) => renderRoleChips(params.row.roles ?? []),
      },
      {
        field: 'email',
        headerName: 'Email',
        flex: 1,
        minWidth: 220,
        valueGetter: (_value, row) => row.email ?? '',
      },
      {
        field: 'phoneNumber',
        headerName: 'Phone Number',
        flex: 1,
        minWidth: 170,
        valueGetter: (_value, row) => row.phoneNumber ?? '',
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
        field: 'updatedAt',
        headerName: 'Updated',
        width: 190,
        valueGetter: (_value, row) => formatDate(row.updatedAt),
      },
      {
        field: 'actions',
        headerName: 'Actions',
        width: 100,
        sortable: false,
        filterable: false,
        renderCell: (params: GridRenderCellParams<UserRow>) => (
          <AdminRowActions
            rowLabel={params.row.username}
            actions={[
              {
                id: 'view',
                label: 'View Details',
                icon: 'view',
                onClick: () => {
                  setDetailsUser(params.row)
                  setDetailsOpen(true)
                },
              },
              {
                id: 'edit',
                label: 'Edit',
                icon: 'edit',
                disabled: !canWrite,
                onClick: () => {
                  setEditUser(params.row)
                  setEditForm(toUserForm(params.row))
                  setEditRoles(params.row.roles ?? [])
                  setEditErrors({})
                  setEditOpen(true)
                },
              },
              {
                id: 'roles',
                label: 'Roles',
                icon: 'view',
                disabled: !canWrite,
                onClick: () => {
                  setRolesUser(params.row)
                  setRolesSelection(params.row.roles ?? [])
                  setRolesOpen(true)
                },
              },
              {
                id: 'reset-password',
                label: 'Reset Password',
                icon: 'delete',
                disabled: !canWrite,
                destructive: true,
                confirmTitle: 'Reset password',
                confirmMessage: `Reset password for ${params.row.username}?`,
                onClick: () => {
                  setResetUser(params.row)
                  setResetPassword('')
                  setResetOpen(true)
                },
              },
            ]}
          />
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
          <Typography color="text.secondary">Manage users, profile metadata, roles, and password resets.</Typography>
        </Box>
        <Button
          variant="contained"
          onClick={() => {
            setCreateForm(defaultCreateForm)
            setCreateRoles([])
            setCreateErrors({})
            setCreateOpen(true)
          }}
          disabled={!canWrite}
        >
          Create User
        </Button>
      </Box>

      <Box sx={{ height: 620, width: '100%', minWidth: 0, overflow: 'hidden' }}>
        <AppDataGrid
          columns={columns}
          fetchData={fetchUsers}
          storageKey="users-table"
          reloadToken={reloadToken}
          stickyRightFields={['actions']}
        />
      </Box>

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} fullWidth maxWidth="lg">
        <DialogTitle>Create User</DialogTitle>
        <DialogContent>
          <Box
            data-testid="desktop-user-create-form-grid"
            sx={{
              mt: 1,
              display: 'grid',
              gap: 2,
              gridTemplateColumns: {
                xs: '1fr',
                md: 'repeat(2, minmax(0, 1fr))',
                xl: 'repeat(3, minmax(0, 1fr))',
              },
            }}
          >
            <TextField
              label="Username"
              value={createForm.username}
              onChange={(event) => setCreateForm((current) => ({ ...current, username: event.target.value }))}
              fullWidth
              required
              error={Boolean(createErrors.username)}
              helperText={createErrors.username}
            />
            <TextField
              label="Password"
              type="password"
              value={createForm.password}
              onChange={(event) => setCreateForm((current) => ({ ...current, password: event.target.value }))}
              fullWidth
              required
              error={Boolean(createErrors.password)}
              helperText={createErrors.password}
            />
            <TextField
              label="Email"
              value={createForm.email}
              onChange={(event) => setCreateForm((current) => ({ ...current, email: event.target.value }))}
              fullWidth
              error={Boolean(createErrors.email)}
              helperText={createErrors.email}
            />
            <TextField
              label="Language"
              value={createForm.language}
              onChange={(event) => setCreateForm((current) => ({ ...current, language: event.target.value }))}
              fullWidth
              error={Boolean(createErrors.language)}
              helperText={createErrors.language}
            />
            <TextField
              label="First Name"
              value={createForm.firstName}
              onChange={(event) => setCreateForm((current) => ({ ...current, firstName: event.target.value }))}
              fullWidth
              error={Boolean(createErrors.firstName)}
              helperText={createErrors.firstName}
            />
            <TextField
              label="Last Name"
              value={createForm.lastName}
              onChange={(event) => setCreateForm((current) => ({ ...current, lastName: event.target.value }))}
              fullWidth
              error={Boolean(createErrors.lastName)}
              helperText={createErrors.lastName}
            />
            <TextField
              label="Display Name"
              value={createForm.displayName}
              onChange={(event) => setCreateForm((current) => ({ ...current, displayName: event.target.value }))}
              fullWidth
              error={Boolean(createErrors.displayName)}
              helperText={createErrors.displayName}
            />
            <TextField
              label="Phone Number"
              value={createForm.phoneNumber}
              onChange={(event) => setCreateForm((current) => ({ ...current, phoneNumber: event.target.value }))}
              fullWidth
              error={Boolean(createErrors.phoneNumber)}
              helperText={createErrors.phoneNumber}
            />
            <TextField
              label="WhatsApp Number"
              value={createForm.whatsappNumber}
              onChange={(event) => setCreateForm((current) => ({ ...current, whatsappNumber: event.target.value }))}
              fullWidth
              error={Boolean(createErrors.whatsappNumber)}
              helperText={createErrors.whatsappNumber}
            />
            <TextField
              label="Telegram Handle"
              value={createForm.telegramHandle}
              onChange={(event) => setCreateForm((current) => ({ ...current, telegramHandle: event.target.value }))}
              fullWidth
              error={Boolean(createErrors.telegramHandle)}
              helperText={createErrors.telegramHandle}
            />
            <Autocomplete
              multiple
              options={roleNames}
              loading={loadingRoleOptions}
              value={createRoles}
              onChange={(_event, value) => setCreateRoles(value)}
              sx={{ gridColumn: '1 / -1' }}
              renderTags={(value, getTagProps) =>
                value.map((option, index) => <Chip label={option} {...getTagProps({ index })} key={option} size="small" />)
              }
              renderInput={(params) => <TextField {...params} label="Roles" placeholder="Assign roles" />}
            />
            <FormControlLabel
              sx={{ gridColumn: '1 / -1' }}
              control={
                <Switch
                  checked={createForm.isActive}
                  onChange={(event) => setCreateForm((current) => ({ ...current, isActive: event.target.checked }))}
                />
              }
              label="Active"
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateOpen(false)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={() => void onCreateUser()}
            disabled={submitting || !createForm.username.trim() || !createForm.password.trim()}
          >
            Create
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={editOpen} onClose={() => setEditOpen(false)} fullWidth maxWidth="lg">
        <DialogTitle>Edit User</DialogTitle>
        <DialogContent>
          <Box
            data-testid="desktop-user-edit-form-grid"
            sx={{
              mt: 1,
              display: 'grid',
              gap: 2,
              gridTemplateColumns: {
                xs: '1fr',
                md: 'repeat(2, minmax(0, 1fr))',
                xl: 'repeat(3, minmax(0, 1fr))',
              },
            }}
          >
            <TextField
              label="Username"
              value={editForm.username}
              onChange={(event) => setEditForm((current) => ({ ...current, username: event.target.value }))}
              fullWidth
              required
              disabled
              error={Boolean(editErrors.username)}
              helperText={editErrors.username}
            />
            <TextField
              label="Password"
              type="password"
              value={editForm.password}
              onChange={(event) => setEditForm((current) => ({ ...current, password: event.target.value }))}
              fullWidth
              error={Boolean(editErrors.password)}
              helperText={editErrors.password || 'Optional. Leave blank to keep current password.'}
            />
            <TextField
              label="Email"
              value={editForm.email}
              onChange={(event) => setEditForm((current) => ({ ...current, email: event.target.value }))}
              fullWidth
              error={Boolean(editErrors.email)}
              helperText={editErrors.email}
            />
            <TextField
              label="Language"
              value={editForm.language}
              onChange={(event) => setEditForm((current) => ({ ...current, language: event.target.value }))}
              fullWidth
              error={Boolean(editErrors.language)}
              helperText={editErrors.language}
            />
            <TextField
              label="First Name"
              value={editForm.firstName}
              onChange={(event) => setEditForm((current) => ({ ...current, firstName: event.target.value }))}
              fullWidth
              error={Boolean(editErrors.firstName)}
              helperText={editErrors.firstName}
            />
            <TextField
              label="Last Name"
              value={editForm.lastName}
              onChange={(event) => setEditForm((current) => ({ ...current, lastName: event.target.value }))}
              fullWidth
              error={Boolean(editErrors.lastName)}
              helperText={editErrors.lastName}
            />
            <TextField
              label="Display Name"
              value={editForm.displayName}
              onChange={(event) => setEditForm((current) => ({ ...current, displayName: event.target.value }))}
              fullWidth
              error={Boolean(editErrors.displayName)}
              helperText={editErrors.displayName}
            />
            <TextField
              label="Phone Number"
              value={editForm.phoneNumber}
              onChange={(event) => setEditForm((current) => ({ ...current, phoneNumber: event.target.value }))}
              fullWidth
              error={Boolean(editErrors.phoneNumber)}
              helperText={editErrors.phoneNumber}
            />
            <TextField
              label="WhatsApp Number"
              value={editForm.whatsappNumber}
              onChange={(event) => setEditForm((current) => ({ ...current, whatsappNumber: event.target.value }))}
              fullWidth
              error={Boolean(editErrors.whatsappNumber)}
              helperText={editErrors.whatsappNumber}
            />
            <TextField
              label="Telegram Handle"
              value={editForm.telegramHandle}
              onChange={(event) => setEditForm((current) => ({ ...current, telegramHandle: event.target.value }))}
              fullWidth
              error={Boolean(editErrors.telegramHandle)}
              helperText={editErrors.telegramHandle}
            />
            <Autocomplete
              multiple
              options={roleNames}
              loading={loadingRoleOptions}
              value={editRoles}
              onChange={(_event, value) => setEditRoles(value)}
              sx={{ gridColumn: '1 / -1' }}
              renderTags={(value, getTagProps) =>
                value.map((option, index) => <Chip label={option} {...getTagProps({ index })} key={option} size="small" />)
              }
              renderInput={(params) => <TextField {...params} label="Roles" placeholder="Assign roles" />}
            />
            <FormControlLabel
              sx={{ gridColumn: '1 / -1' }}
              control={
                <Switch
                  checked={editForm.isActive}
                  onChange={(event) => setEditForm((current) => ({ ...current, isActive: event.target.checked }))}
                />
              }
              label="Active"
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={() => void onSaveEdit()} disabled={submitting || !editUser}>
            Save
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={detailsOpen} onClose={() => setDetailsOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>User Details</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <Box>
              <Typography variant="subtitle2" color="text.secondary">
                Username
              </Typography>
              <Typography>{detailsUser?.username ?? '-'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" color="text.secondary">
                Display Name
              </Typography>
              <Typography>{detailsUser ? displayNameForRow(detailsUser) : '-'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                Roles
              </Typography>
              {renderRoleChips(detailsUser?.roles ?? [])}
            </Box>
            <Box>
              <Typography variant="subtitle2" color="text.secondary">
                Status
              </Typography>
              <Typography>{detailsUser?.isActive ? 'Active' : 'Inactive'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" color="text.secondary">
                Updated
              </Typography>
              <Typography>{detailsUser ? formatDate(detailsUser.updatedAt) : '-'}</Typography>
            </Box>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDetailsOpen(false)}>Close</Button>
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
            <Autocomplete
              multiple
              options={roleNames}
              loading={loadingRoleOptions}
              value={rolesSelection}
              onChange={(_event, value) => setRolesSelection(value)}
              renderTags={(value, getTagProps) =>
                value.map((option, index) => <Chip label={option} {...getTagProps({ index })} key={option} size="small" />)
              }
              renderInput={(params) => <TextField {...params} label="Roles" placeholder="Assign roles" />}
            />
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
