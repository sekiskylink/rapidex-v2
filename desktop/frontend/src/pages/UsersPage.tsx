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

const ROLE_OPTIONS = ['Admin', 'Manager', 'Staff', 'Viewer']

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

export function UsersPage() {
  const apiClient = useApiClient()
  const principal = useSessionPrincipal()
  const canWrite = Boolean(principal?.permissions.includes('users.write'))

  const [reloadToken, setReloadToken] = React.useState(0)

  const [createOpen, setCreateOpen] = React.useState(false)
  const [createForm, setCreateForm] = React.useState<UserFormState>(defaultCreateForm)
  const [createRoles, setCreateRoles] = React.useState<string[]>([])
  const [createErrors, setCreateErrors] = React.useState<UserFormErrors>({})

  const [editOpen, setEditOpen] = React.useState(false)
  const [editUser, setEditUser] = React.useState<UserRow | null>(null)
  const [editForm, setEditForm] = React.useState<UserFormState>(defaultCreateForm)
  const [editErrors, setEditErrors] = React.useState<UserFormErrors>({})

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
        valueGetter: (_value, row) => new Date(row.updatedAt).toLocaleString(),
      },
      {
        field: 'actions',
        headerName: 'Actions',
        width: 320,
        sortable: false,
        filterable: false,
        renderCell: (params: GridRenderCellParams<UserRow>) => (
          <Stack direction="row" spacing={1}>
            <Button
              size="small"
              onClick={() => {
                setEditUser(params.row)
                setEditForm(toUserForm(params.row))
                setEditErrors({})
                setEditOpen(true)
              }}
              disabled={!canWrite}
            >
              Edit
            </Button>
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
              control={
                <Switch
                  checked={createForm.isActive}
                  onChange={(event) => setCreateForm((current) => ({ ...current, isActive: event.target.checked }))}
                />
              }
              label="Active"
            />
          </Stack>
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

      <Dialog open={editOpen} onClose={() => setEditOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Edit User</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
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
            <FormControlLabel
              control={
                <Switch
                  checked={editForm.isActive}
                  onChange={(event) => setEditForm((current) => ({ ...current, isActive: event.target.checked }))}
                />
              }
              label="Active"
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={() => void onSaveEdit()} disabled={submitting || !editUser}>
            Save
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
