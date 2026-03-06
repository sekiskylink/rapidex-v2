import React from 'react'
import { Alert, Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, Stack, Switch, TextField, Typography } from '@mui/material'
import type { GridColDef, GridRenderCellParams } from '@mui/x-data-grid'
import { getAuthSnapshot } from '../auth/state'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { apiRequest } from '../lib/api'
import { isApiError } from '../auth/AuthProvider'
import { buildListQuery, type PaginatedResponse } from '../lib/pagination'
import { useSnackbar } from '../ui/snackbar'

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

function toText(value?: string) {
  return value ?? ''
}

function displayNameForRow(row: UserRow) {
  if (row.displayName && row.displayName.trim()) {
    return row.displayName
  }
  const derived = [row.firstName, row.lastName].filter((value) => value && value.trim()).join(' ').trim()
  return derived || row.username
}

function toUserForm(row: UserRow): UserFormState {
  return {
    username: row.username,
    password: '',
    email: toText(row.email),
    language: toText(row.language) || 'English',
    firstName: toText(row.firstName),
    lastName: toText(row.lastName),
    displayName: toText(row.displayName),
    phoneNumber: toText(row.phoneNumber),
    whatsappNumber: toText(row.whatsappNumber),
    telegramHandle: toText(row.telegramHandle),
    isActive: row.isActive,
  }
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
  if (!isApiError(error) || error.code !== 'VALIDATION_ERROR' || !error.details || typeof error.details !== 'object') {
    return {}
  }

  const details = error.details as Record<string, unknown>
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

function toRequestErrorMessage(error: unknown, fallback: string) {
  if (!isApiError(error)) {
    return fallback
  }
  return error.requestId ? `${error.message} Request ID: ${error.requestId}` : error.message
}

export function UsersPage() {
  const { showSnackbar } = useSnackbar()
  const canWrite = React.useMemo(() => {
    const user = getAuthSnapshot().user
    return Boolean(user?.permissions.some((permission) => permission.trim().toLowerCase() === 'users.write'))
  }, [])

  const [reloadToken, setReloadToken] = React.useState(0)

  const [createOpen, setCreateOpen] = React.useState(false)
  const [createForm, setCreateForm] = React.useState<UserFormState>(defaultCreateForm)
  const [createErrors, setCreateErrors] = React.useState<UserFormErrors>({})
  const [createErrorMessage, setCreateErrorMessage] = React.useState('')

  const [editOpen, setEditOpen] = React.useState(false)
  const [editUser, setEditUser] = React.useState<UserRow | null>(null)
  const [editForm, setEditForm] = React.useState<UserFormState>(defaultCreateForm)
  const [editErrors, setEditErrors] = React.useState<UserFormErrors>({})
  const [editErrorMessage, setEditErrorMessage] = React.useState('')
  const [submitting, setSubmitting] = React.useState(false)

  const refreshGrid = React.useCallback(() => setReloadToken((value) => value + 1), [])

  const fetchUsers = React.useCallback(async (params: AppDataGridFetchParams) => {
    const query = buildListQuery(params)
    const response = await apiRequest<PaginatedResponse<UserRow>>(`/users?${query}`)
    return {
      rows: response.items,
      total: response.totalCount,
    }
  }, [])

  const onCreateUser = async () => {
    setSubmitting(true)
    setCreateErrors({})
    setCreateErrorMessage('')
    try {
      await apiRequest('/users', {
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
        }),
      })
      showSnackbar({ severity: 'success', message: 'User created.' })
      setCreateOpen(false)
      setCreateForm(defaultCreateForm)
      refreshGrid()
    } catch (error) {
      const fieldErrors = validationFieldErrors(error)
      if (Object.values(fieldErrors).some((message) => Boolean(message))) {
        setCreateErrors(fieldErrors)
      }
      setCreateErrorMessage(toRequestErrorMessage(error, 'Unable to create user.'))
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
    setEditErrorMessage('')

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
      await apiRequest(`/users/${editUser.id}`, {
        method: 'PATCH',
        body: JSON.stringify(payload),
      })
      showSnackbar({ severity: 'success', message: 'User updated.' })
      setEditOpen(false)
      setEditUser(null)
      setEditForm(defaultCreateForm)
      refreshGrid()
    } catch (error) {
      const fieldErrors = validationFieldErrors(error)
      if (Object.values(fieldErrors).some((message) => Boolean(message))) {
        setEditErrors(fieldErrors)
      }
      setEditErrorMessage(toRequestErrorMessage(error, 'Unable to update user.'))
    } finally {
      setSubmitting(false)
    }
  }

  const onToggleActive = React.useCallback(
    async (row: UserRow, nextActive: boolean) => {
      try {
        await apiRequest(`/users/${row.id}`, {
          method: 'PATCH',
          body: JSON.stringify({ isActive: nextActive }),
        })
        showSnackbar({ severity: 'success', message: `User ${nextActive ? 'activated' : 'deactivated'}.` })
        refreshGrid()
      } catch (error) {
        showSnackbar({ severity: 'error', message: toRequestErrorMessage(error, 'Unable to update active status.') })
      }
    },
    [refreshGrid, showSnackbar],
  )

  const columns = React.useMemo<GridColDef<UserRow>[]>(
    () => [
      { field: 'username', headerName: 'Username', minWidth: 150, flex: 1 },
      {
        field: 'displayName',
        headerName: 'Display Name',
        minWidth: 190,
        flex: 1,
        valueGetter: (_value, row) => displayNameForRow(row),
      },
      { field: 'language', headerName: 'Language', minWidth: 140, valueGetter: (_value, row) => row.language ?? 'English' },
      { field: 'email', headerName: 'Email', minWidth: 220, flex: 1, valueGetter: (_value, row) => row.email ?? '' },
      { field: 'phoneNumber', headerName: 'Phone Number', minWidth: 160, valueGetter: (_value, row) => row.phoneNumber ?? '' },
      { field: 'whatsappNumber', headerName: 'WhatsApp Number', minWidth: 170, valueGetter: (_value, row) => row.whatsappNumber ?? '' },
      { field: 'telegramHandle', headerName: 'Telegram Handle', minWidth: 170, valueGetter: (_value, row) => row.telegramHandle ?? '' },
      { field: 'isActive', headerName: 'Active', width: 110, type: 'boolean' },
      {
        field: 'updatedAt',
        headerName: 'Updated',
        minWidth: 185,
        valueGetter: (_value, row) => new Date(row.updatedAt).toLocaleString(),
      },
      {
        field: 'actions',
        headerName: 'Actions',
        sortable: false,
        filterable: false,
        width: 100,
        renderCell: (params: GridRenderCellParams<UserRow>) => (
          <AdminRowActions
            rowLabel={params.row.username}
            actions={[
              {
                id: 'view',
                label: 'View',
                icon: 'view',
                onClick: () => {
                  setEditUser(params.row)
                  setEditForm(toUserForm(params.row))
                  setEditErrors({})
                  setEditErrorMessage('')
                  setEditOpen(true)
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
                  setEditErrors({})
                  setEditErrorMessage('')
                  setEditOpen(true)
                },
              },
              {
                id: 'toggle-active',
                label: params.row.isActive ? 'Deactivate' : 'Activate',
                icon: 'delete',
                disabled: !canWrite,
                destructive: params.row.isActive,
                confirmTitle: params.row.isActive ? 'Deactivate user' : 'Activate user',
                confirmMessage: `${params.row.isActive ? 'Deactivate' : 'Activate'} ${params.row.username}?`,
                onClick: () => void onToggleActive(params.row, !params.row.isActive),
              },
            ]}
          />
        ),
      },
    ],
    [canWrite, onToggleActive],
  )

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 2 }}>
        <Box>
          <Typography variant="h5" component="h1" gutterBottom>
            Users
          </Typography>
          <Typography color="text.secondary">Manage users and profile metadata.</Typography>
        </Box>
        {canWrite ? (
          <Button
            variant="contained"
            onClick={() => {
              setCreateForm(defaultCreateForm)
              setCreateErrors({})
              setCreateErrorMessage('')
              setCreateOpen(true)
            }}
          >
            Create User
          </Button>
        ) : null}
      </Box>
      <Box sx={{ height: 620, width: '100%', minWidth: 0, overflow: 'auto' }}>
        <AppDataGrid
          columns={columns}
          fetchData={fetchUsers}
          storageKey="users-table"
          reloadToken={reloadToken}
          enablePinnedColumns
          stickyRightFields={['actions']}
        />
      </Box>

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Create User</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            {createErrorMessage ? <Alert severity="error">{createErrorMessage}</Alert> : null}
            <TextField
              label="Username"
              required
              value={createForm.username}
              onChange={(event) => setCreateForm((current) => ({ ...current, username: event.target.value }))}
              error={Boolean(createErrors.username)}
              helperText={createErrors.username}
              fullWidth
            />
            <TextField
              label="Password"
              type="password"
              required
              value={createForm.password}
              onChange={(event) => setCreateForm((current) => ({ ...current, password: event.target.value }))}
              error={Boolean(createErrors.password)}
              helperText={createErrors.password}
              fullWidth
            />
            <TextField
              label="Email"
              value={createForm.email}
              onChange={(event) => setCreateForm((current) => ({ ...current, email: event.target.value }))}
              error={Boolean(createErrors.email)}
              helperText={createErrors.email}
              fullWidth
            />
            <TextField
              label="Language"
              value={createForm.language}
              onChange={(event) => setCreateForm((current) => ({ ...current, language: event.target.value }))}
              error={Boolean(createErrors.language)}
              helperText={createErrors.language}
              fullWidth
            />
            <TextField
              label="First Name"
              value={createForm.firstName}
              onChange={(event) => setCreateForm((current) => ({ ...current, firstName: event.target.value }))}
              error={Boolean(createErrors.firstName)}
              helperText={createErrors.firstName}
              fullWidth
            />
            <TextField
              label="Last Name"
              value={createForm.lastName}
              onChange={(event) => setCreateForm((current) => ({ ...current, lastName: event.target.value }))}
              error={Boolean(createErrors.lastName)}
              helperText={createErrors.lastName}
              fullWidth
            />
            <TextField
              label="Display Name"
              value={createForm.displayName}
              onChange={(event) => setCreateForm((current) => ({ ...current, displayName: event.target.value }))}
              error={Boolean(createErrors.displayName)}
              helperText={createErrors.displayName}
              fullWidth
            />
            <TextField
              label="Phone Number"
              value={createForm.phoneNumber}
              onChange={(event) => setCreateForm((current) => ({ ...current, phoneNumber: event.target.value }))}
              error={Boolean(createErrors.phoneNumber)}
              helperText={createErrors.phoneNumber}
              fullWidth
            />
            <TextField
              label="WhatsApp Number"
              value={createForm.whatsappNumber}
              onChange={(event) => setCreateForm((current) => ({ ...current, whatsappNumber: event.target.value }))}
              error={Boolean(createErrors.whatsappNumber)}
              helperText={createErrors.whatsappNumber}
              fullWidth
            />
            <TextField
              label="Telegram Handle"
              value={createForm.telegramHandle}
              onChange={(event) => setCreateForm((current) => ({ ...current, telegramHandle: event.target.value }))}
              error={Boolean(createErrors.telegramHandle)}
              helperText={createErrors.telegramHandle}
              fullWidth
            />
            <Stack direction="row" alignItems="center" spacing={1}>
              <Typography variant="body2">Active</Typography>
              <Switch
                checked={createForm.isActive}
                onChange={(event) => setCreateForm((current) => ({ ...current, isActive: event.target.checked }))}
                inputProps={{ 'aria-label': 'Active' }}
              />
            </Stack>
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
            {editErrorMessage ? <Alert severity="error">{editErrorMessage}</Alert> : null}
            <TextField
              label="Username"
              value={editForm.username}
              onChange={(event) => setEditForm((current) => ({ ...current, username: event.target.value }))}
              error={Boolean(editErrors.username)}
              helperText={editErrors.username}
              disabled
              fullWidth
            />
            <TextField
              label="Password"
              type="password"
              value={editForm.password}
              onChange={(event) => setEditForm((current) => ({ ...current, password: event.target.value }))}
              error={Boolean(editErrors.password)}
              helperText={editErrors.password || 'Optional. Leave blank to keep current password.'}
              fullWidth
            />
            <TextField
              label="Email"
              value={editForm.email}
              onChange={(event) => setEditForm((current) => ({ ...current, email: event.target.value }))}
              error={Boolean(editErrors.email)}
              helperText={editErrors.email}
              fullWidth
            />
            <TextField
              label="Language"
              value={editForm.language}
              onChange={(event) => setEditForm((current) => ({ ...current, language: event.target.value }))}
              error={Boolean(editErrors.language)}
              helperText={editErrors.language}
              fullWidth
            />
            <TextField
              label="First Name"
              value={editForm.firstName}
              onChange={(event) => setEditForm((current) => ({ ...current, firstName: event.target.value }))}
              error={Boolean(editErrors.firstName)}
              helperText={editErrors.firstName}
              fullWidth
            />
            <TextField
              label="Last Name"
              value={editForm.lastName}
              onChange={(event) => setEditForm((current) => ({ ...current, lastName: event.target.value }))}
              error={Boolean(editErrors.lastName)}
              helperText={editErrors.lastName}
              fullWidth
            />
            <TextField
              label="Display Name"
              value={editForm.displayName}
              onChange={(event) => setEditForm((current) => ({ ...current, displayName: event.target.value }))}
              error={Boolean(editErrors.displayName)}
              helperText={editErrors.displayName}
              fullWidth
            />
            <TextField
              label="Phone Number"
              value={editForm.phoneNumber}
              onChange={(event) => setEditForm((current) => ({ ...current, phoneNumber: event.target.value }))}
              error={Boolean(editErrors.phoneNumber)}
              helperText={editErrors.phoneNumber}
              fullWidth
            />
            <TextField
              label="WhatsApp Number"
              value={editForm.whatsappNumber}
              onChange={(event) => setEditForm((current) => ({ ...current, whatsappNumber: event.target.value }))}
              error={Boolean(editErrors.whatsappNumber)}
              helperText={editErrors.whatsappNumber}
              fullWidth
            />
            <TextField
              label="Telegram Handle"
              value={editForm.telegramHandle}
              onChange={(event) => setEditForm((current) => ({ ...current, telegramHandle: event.target.value }))}
              error={Boolean(editErrors.telegramHandle)}
              helperText={editErrors.telegramHandle}
              fullWidth
            />
            <Stack direction="row" alignItems="center" spacing={1}>
              <Typography variant="body2">Active</Typography>
              <Switch
                checked={editForm.isActive}
                onChange={(event) => setEditForm((current) => ({ ...current, isActive: event.target.checked }))}
                inputProps={{ 'aria-label': 'Active' }}
              />
            </Stack>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={() => void onSaveEdit()} disabled={submitting || !editUser}>
            Save
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
