import React from 'react'
import PersonAddRoundedIcon from '@mui/icons-material/PersonAddRounded'
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
  FormControlLabel,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import type { GridColDef, GridRenderCellParams } from '@mui/x-data-grid'
import type { PaginatedResponse } from '../api/pagination'
import { useApiClient } from '../api/useApiClient'
import { useSessionPrincipal } from '../auth/hooks'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { buildAdminListRequestQuery, useAdminListSearch } from '../components/admin/listSearch'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { handleAppError } from '../errors/handleAppError'
import { notify } from '../notifications/facade'

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

interface OrgUnitOption {
  id: number
  name: string
  displayPath?: string
}

interface UserOrgUnitAssignment {
  orgUnitId: number
  orgUnitName: string
  displayPath?: string
}

interface UserOrgUnitAssignmentsResponse {
  orgUnitIds: number[]
  items: UserOrgUnitAssignment[]
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

function mapValidationFieldErrors(details?: Record<string, string[]>): UserFormErrors {
  const first = (key: string) => details?.[key]?.[0] ?? ''
  return {
    username: first('username'),
    password: first('password'),
    email: first('email'),
    language: first('language'),
    firstName: first('firstName'),
    lastName: first('lastName'),
    displayName: first('displayName'),
    phoneNumber: first('phoneNumber'),
    whatsappNumber: first('whatsappNumber'),
    telegramHandle: first('telegramHandle'),
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

function mergeOrgUnitOptions(...collections: OrgUnitOption[][]) {
  const merged = new Map<number, OrgUnitOption>()
  for (const collection of collections) {
    for (const item of collection) {
      merged.set(item.id, item)
    }
  }
  return Array.from(merged.values()).sort((left, right) => left.name.localeCompare(right.name))
}

function formatOrgUnitPath(option: OrgUnitOption) {
  return option.displayPath?.trim() ?? ''
}

function formatOrgUnitChipLabel(option: OrgUnitOption) {
  const path = formatOrgUnitPath(option)
  if (!path) {
    return option.name
  }
  return `${option.name} (${path})`
}

function renderOrgUnitOption(props: React.HTMLAttributes<HTMLLIElement>, option: OrgUnitOption) {
  const path = formatOrgUnitPath(option)
  return (
    <Box component="li" {...props}>
      <Stack spacing={0.25}>
        <Typography variant="body2">{option.name}</Typography>
        {path ? (
          <Typography variant="caption" color="text.secondary">
            {path}
          </Typography>
        ) : null}
      </Stack>
    </Box>
  )
}

export function UsersPage() {
  const apiClient = useApiClient()
  const principal = useSessionPrincipal()
  const canWrite = Boolean(principal?.permissions.includes('users.write'))
  const canReadRoles = Boolean(principal?.permissions.includes('users.read'))

  const [reloadToken, setReloadToken] = React.useState(0)
  const { searchInput, setSearchInput, search } = useAdminListSearch()

  const [roleOptions, setRoleOptions] = React.useState<RoleOption[]>([])
  const [loadingRoleOptions, setLoadingRoleOptions] = React.useState(false)

  const [createOpen, setCreateOpen] = React.useState(false)
  const [createForm, setCreateForm] = React.useState<UserFormState>(defaultCreateForm)
  const [createRoles, setCreateRoles] = React.useState<string[]>([])
  const [createAssignmentOptions, setCreateAssignmentOptions] = React.useState<OrgUnitOption[]>([])
  const [createAssignmentSelection, setCreateAssignmentSelection] = React.useState<OrgUnitOption[]>([])
  const [createAssignmentSearch, setCreateAssignmentSearch] = React.useState('')
  const [createAssignmentLoading, setCreateAssignmentLoading] = React.useState(false)
  const [createErrors, setCreateErrors] = React.useState<UserFormErrors>({})

  const [editOpen, setEditOpen] = React.useState(false)
  const [editUser, setEditUser] = React.useState<UserRow | null>(null)
  const [editForm, setEditForm] = React.useState<UserFormState>(defaultCreateForm)
  const [editRoles, setEditRoles] = React.useState<string[]>([])
  const [editAssignmentOptions, setEditAssignmentOptions] = React.useState<OrgUnitOption[]>([])
  const [editAssignmentSelection, setEditAssignmentSelection] = React.useState<OrgUnitOption[]>([])
  const [editAssignmentInitialIDs, setEditAssignmentInitialIDs] = React.useState<number[]>([])
  const [editAssignmentSearch, setEditAssignmentSearch] = React.useState('')
  const [editAssignmentLoading, setEditAssignmentLoading] = React.useState(false)
  const [editErrors, setEditErrors] = React.useState<UserFormErrors>({})

  const [detailsOpen, setDetailsOpen] = React.useState(false)
  const [detailsUser, setDetailsUser] = React.useState<UserRow | null>(null)

  const [resetOpen, setResetOpen] = React.useState(false)
  const [resetUser, setResetUser] = React.useState<UserRow | null>(null)
  const [resetPassword, setResetPassword] = React.useState('')

  const [rolesOpen, setRolesOpen] = React.useState(false)
  const [rolesUser, setRolesUser] = React.useState<UserRow | null>(null)
  const [rolesSelection, setRolesSelection] = React.useState<string[]>([])
  const [assignmentsOpen, setAssignmentsOpen] = React.useState(false)
  const [assignmentsUser, setAssignmentsUser] = React.useState<UserRow | null>(null)
  const [assignmentOptions, setAssignmentOptions] = React.useState<OrgUnitOption[]>([])
  const [assignmentSelection, setAssignmentSelection] = React.useState<OrgUnitOption[]>([])
  const [assignmentInitialIDs, setAssignmentInitialIDs] = React.useState<number[]>([])
  const [assignmentSearch, setAssignmentSearch] = React.useState('')
  const [assignmentLoading, setAssignmentLoading] = React.useState(false)
  const [assignmentSaving, setAssignmentSaving] = React.useState(false)
  const [assignmentError, setAssignmentError] = React.useState('')

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
    } catch (error) {
      setRoleOptions([])
      await handleAppError(error, { fallbackMessage: 'Unable to load roles.' })
    } finally {
      setLoadingRoleOptions(false)
    }
  }, [apiClient, canReadRoles])

  React.useEffect(() => {
    void loadRoleOptions()
  }, [loadRoleOptions])

  const loadOrgUnitOptions = React.useCallback(async (search: string) => {
    const payload = await apiClient.request<PaginatedResponse<OrgUnitOption>>(`/api/v1/orgunits?page=0&pageSize=50&search=${encodeURIComponent(search.trim())}`)
    return Array.isArray(payload.items) ? payload.items : []
  }, [apiClient])

  const fetchUsers = React.useCallback(
    async (params: AppDataGridFetchParams) => {
      const query = buildAdminListRequestQuery(params, { search })
      const payload = await apiClient.request<PaginatedResponse<UserRow>>(`/api/v1/users?${query}`)
      return {
        rows: payload.items,
        total: payload.totalCount,
      }
    },
    [apiClient, search],
  )

  const saveOrgUnitAssignments = React.useCallback(async (userID: number, initialIDs: number[], selection: OrgUnitOption[]) => {
    const nextIDs = selection.map((item) => item.id)
    const toRemove = initialIDs.filter((id) => !nextIDs.includes(id))
    const toAdd = nextIDs.filter((id) => !initialIDs.includes(id))
    for (const orgUnitID of toRemove) {
      await apiClient.request(`/api/v1/user-org-units/${userID}/${orgUnitID}`, { method: 'DELETE' })
    }
    for (const orgUnitID of toAdd) {
      await apiClient.request('/api/v1/user-org-units', {
        method: 'POST',
        body: JSON.stringify({ userId: userID, orgUnitId: orgUnitID }),
      })
    }
    return nextIDs
  }, [apiClient])

  const loadUserOrgUnitAssignments = React.useCallback(async (userID: number) => {
    const [currentAssignments, availableOptions] = await Promise.all([
      apiClient.request<UserOrgUnitAssignmentsResponse>(`/api/v1/user-org-units/${userID}`),
      loadOrgUnitOptions(''),
    ])
    const selected = (currentAssignments.items ?? []).map((item) => ({
      id: item.orgUnitId,
      name: item.orgUnitName,
      displayPath: item.displayPath,
    }))
    return {
      options: mergeOrgUnitOptions(availableOptions, selected),
      selection: selected,
      initialIDs: currentAssignments.orgUnitIds ?? selected.map((item) => item.id),
    }
  }, [apiClient, loadOrgUnitOptions])

  const onCreateAssignmentSearchChange = React.useCallback(async (value: string) => {
    setCreateAssignmentSearch(value)
    try {
      const options = await loadOrgUnitOptions(value)
      setCreateAssignmentOptions((current) => mergeOrgUnitOptions(current, options, createAssignmentSelection))
    } catch {
    }
  }, [createAssignmentSelection, loadOrgUnitOptions])

  const resetCreateDialog = React.useCallback(() => {
    setCreateForm(defaultCreateForm)
    setCreateRoles([])
    setCreateAssignmentOptions([])
    setCreateAssignmentSelection([])
    setCreateAssignmentSearch('')
    setCreateAssignmentLoading(false)
    setCreateErrors({})
  }, [])

  const resetEditDialog = React.useCallback(() => {
    setEditUser(null)
    setEditForm(defaultCreateForm)
    setEditRoles([])
    setEditAssignmentOptions([])
    setEditAssignmentSelection([])
    setEditAssignmentInitialIDs([])
    setEditAssignmentSearch('')
    setEditAssignmentLoading(false)
    setEditErrors({})
  }, [])

  const openCreateDialog = React.useCallback(async () => {
    resetCreateDialog()
    setCreateOpen(true)
    setCreateAssignmentLoading(true)
    try {
      const options = await loadOrgUnitOptions('')
      setCreateAssignmentOptions(options)
    } catch {
      setCreateAssignmentOptions([])
    } finally {
      setCreateAssignmentLoading(false)
    }
  }, [loadOrgUnitOptions, resetCreateDialog])

  const onEditAssignmentSearchChange = React.useCallback(async (value: string) => {
    setEditAssignmentSearch(value)
    try {
      const options = await loadOrgUnitOptions(value)
      setEditAssignmentOptions((current) => mergeOrgUnitOptions(current, options, editAssignmentSelection))
    } catch {
    }
  }, [editAssignmentSelection, loadOrgUnitOptions])

  const openEditDialog = React.useCallback(async (row: UserRow) => {
    resetEditDialog()
    setEditUser(row)
    setEditForm(toUserForm(row))
    setEditRoles(row.roles ?? [])
    setEditOpen(true)
    setEditAssignmentLoading(true)
    try {
      const assignmentState = await loadUserOrgUnitAssignments(row.id)
      setEditAssignmentOptions(assignmentState.options)
      setEditAssignmentSelection(assignmentState.selection)
      setEditAssignmentInitialIDs(assignmentState.initialIDs)
    } catch (error) {
      await handleAppError(error, { fallbackMessage: 'Unable to load org unit assignments.' })
      setEditAssignmentOptions([])
      setEditAssignmentSelection([])
      setEditAssignmentInitialIDs([])
    } finally {
      setEditAssignmentLoading(false)
    }
  }, [loadUserOrgUnitAssignments, resetEditDialog])

  const onCreateUser = async () => {
    setSubmitting(true)
    setCreateErrors({})
    try {
      const created = await apiClient.request<UserRow>('/api/v1/users', {
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
      if (createAssignmentSelection.length) {
        try {
          await saveOrgUnitAssignments(created.id, [], createAssignmentSelection)
        } catch (error) {
          await handleAppError(error, { fallbackMessage: 'User created, but org unit assignments could not be saved.' })
          notify.error('User created, but org unit assignments could not be saved. Update Org Unit Scope from the user actions menu.')
        }
      }
      notify.success('User created.')
      setCreateOpen(false)
      resetCreateDialog()
      refreshGrid()
    } catch (error) {
      await handleAppError(error, {
        fallbackMessage: 'Unable to create user.',
        onValidationError: (fieldErrors) => setCreateErrors(mapValidationFieldErrors(fieldErrors)),
      })
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
      await saveOrgUnitAssignments(editUser.id, editAssignmentInitialIDs, editAssignmentSelection)
      notify.success('User updated.')
      setEditOpen(false)
      resetEditDialog()
      refreshGrid()
    } catch (error) {
      await handleAppError(error, {
        fallbackMessage: 'Unable to update user.',
        onValidationError: (fieldErrors) => setEditErrors(mapValidationFieldErrors(fieldErrors)),
      })
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
      notify.success(`User ${nextActive ? 'activated' : 'deactivated'}.`)
      refreshGrid()
    } catch (error) {
      await handleAppError(error, { fallbackMessage: 'Unable to update active status.' })
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
      notify.success('Password reset.')
      setResetOpen(false)
      setResetUser(null)
      setResetPassword('')
      refreshGrid()
    } catch (error) {
      await handleAppError(error, { fallbackMessage: 'Unable to reset password.' })
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
      notify.success('Roles updated.')
      setRolesOpen(false)
      setRolesUser(null)
      setRolesSelection([])
      refreshGrid()
    } catch (error) {
      await handleAppError(error, { fallbackMessage: 'Unable to update roles.' })
    } finally {
      setSubmitting(false)
    }
  }

  const openAssignmentsDialog = React.useCallback(async (row: UserRow) => {
    setAssignmentsUser(row)
    setAssignmentsOpen(true)
    setAssignmentLoading(true)
    setAssignmentSaving(false)
    setAssignmentError('')
    setAssignmentSearch('')
    try {
      const assignmentState = await loadUserOrgUnitAssignments(row.id)
      setAssignmentSelection(assignmentState.selection)
      setAssignmentInitialIDs(assignmentState.initialIDs)
      setAssignmentOptions(assignmentState.options)
    } catch (error) {
      setAssignmentSelection([])
      setAssignmentInitialIDs([])
      setAssignmentOptions([])
      await handleAppError(error, { fallbackMessage: 'Unable to load org unit assignments.' })
      setAssignmentError('Unable to load org unit assignments.')
    } finally {
      setAssignmentLoading(false)
    }
  }, [loadUserOrgUnitAssignments])

  const onAssignmentSearchChange = React.useCallback(async (value: string) => {
    setAssignmentSearch(value)
    if (!assignmentsOpen) {
      return
    }
    try {
      const options = await loadOrgUnitOptions(value)
      setAssignmentOptions((current) => mergeOrgUnitOptions(current, options, assignmentSelection))
    } catch {
    }
  }, [assignmentSelection, assignmentsOpen, loadOrgUnitOptions])

  const onSaveAssignments = React.useCallback(async () => {
    if (!assignmentsUser) {
      return
    }
    setAssignmentSaving(true)
    setAssignmentError('')
    try {
      const nextIDs = await saveOrgUnitAssignments(assignmentsUser.id, assignmentInitialIDs, assignmentSelection)
      notify.success('Org unit assignments updated.')
      setAssignmentsOpen(false)
      setAssignmentsUser(null)
      setAssignmentInitialIDs(nextIDs)
    } catch (error) {
      await handleAppError(error, { fallbackMessage: 'Unable to save org unit assignments.' })
      setAssignmentError('Unable to save org unit assignments.')
    } finally {
      setAssignmentSaving(false)
    }
  }, [assignmentInitialIDs, assignmentSelection, assignmentsUser, saveOrgUnitAssignments])

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
                  void openEditDialog(params.row)
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
                id: 'jurisdiction',
                label: 'Org Unit Scope',
                icon: 'view',
                disabled: !canWrite,
                onClick: () => {
                  void openAssignmentsDialog(params.row)
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
    [canWrite, openAssignmentsDialog, openEditDialog],
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
          startIcon={<PersonAddRoundedIcon />}
          onClick={() => void openCreateDialog()}
          disabled={!canWrite}
        >
          Create User
        </Button>
      </Box>

      <TextField
        label="Search"
        placeholder="Search username, email, or display name"
        value={searchInput}
        onChange={(event) => setSearchInput(event.target.value)}
        sx={{ maxWidth: 420 }}
      />

      <Box sx={{ height: 620, width: '100%', minWidth: 0, overflow: 'hidden' }}>
        <AppDataGrid
          columns={columns}
          fetchData={fetchUsers}
          storageKey="users-table"
          reloadToken={reloadToken}
          externalQueryKey={search}
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
            <Autocomplete
              multiple
              options={createAssignmentOptions}
              loading={createAssignmentLoading}
              value={createAssignmentSelection}
              inputValue={createAssignmentSearch}
              onInputChange={(_event, value) => void onCreateAssignmentSearchChange(value)}
              onChange={(_event, value) => {
                setCreateAssignmentSelection(value)
                setCreateAssignmentOptions((current) => mergeOrgUnitOptions(current, value))
              }}
              isOptionEqualToValue={(option, value) => option.id === value.id}
              getOptionLabel={(option) => option.name}
              renderOption={renderOrgUnitOption}
              sx={{ gridColumn: '1 / -1' }}
              renderTags={(value, getTagProps) =>
                value.map((option, index) => (
                  <Chip label={formatOrgUnitChipLabel(option)} {...getTagProps({ index })} key={option.id} size="small" />
                ))
              }
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Assigned Org Units"
                  placeholder="Search facilities or jurisdiction roots"
                  helperText="Users can access any descendants of the selected org units."
                />
              )}
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
            <Autocomplete
              multiple
              options={editAssignmentOptions}
              loading={editAssignmentLoading}
              value={editAssignmentSelection}
              inputValue={editAssignmentSearch}
              onInputChange={(_event, value) => void onEditAssignmentSearchChange(value)}
              onChange={(_event, value) => {
                setEditAssignmentSelection(value)
                setEditAssignmentOptions((current) => mergeOrgUnitOptions(current, value))
              }}
              isOptionEqualToValue={(option, value) => option.id === value.id}
              getOptionLabel={(option) => option.name}
              renderOption={renderOrgUnitOption}
              sx={{ gridColumn: '1 / -1' }}
              renderTags={(value, getTagProps) =>
                value.map((option, index) => (
                  <Chip label={formatOrgUnitChipLabel(option)} {...getTagProps({ index })} key={option.id} size="small" />
                ))
              }
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Assigned Org Units"
                  placeholder="Search facilities or jurisdiction roots"
                  helperText="Users can access any descendants of the selected org units."
                />
              )}
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

      <Dialog open={assignmentsOpen} onClose={() => !assignmentSaving && setAssignmentsOpen(false)} fullWidth maxWidth="md">
        <DialogTitle>Org Unit Scope</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <Typography color="text.secondary">
              {assignmentsUser ? `Assign jurisdiction roots for ${assignmentsUser.username}. Descendants inherit automatically.` : 'Assign jurisdiction roots.'}
            </Typography>
            {assignmentError ? <Alert severity="error">{assignmentError}</Alert> : null}
            <Autocomplete
              multiple
              options={assignmentOptions}
              loading={assignmentLoading}
              value={assignmentSelection}
              inputValue={assignmentSearch}
              onInputChange={(_event, value) => void onAssignmentSearchChange(value)}
              onChange={(_event, value) => {
                setAssignmentSelection(value)
                setAssignmentOptions((current) => mergeOrgUnitOptions(current, value))
              }}
              isOptionEqualToValue={(option, value) => option.id === value.id}
              getOptionLabel={(option) => option.name}
              renderOption={renderOrgUnitOption}
              renderTags={(value, getTagProps) =>
                value.map((option, index) => (
                  <Chip label={formatOrgUnitChipLabel(option)} {...getTagProps({ index })} key={option.id} size="small" />
                ))
              }
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Assigned Org Units"
                  placeholder="Search facilities or jurisdiction roots"
                  helperText="Users can access any descendants of the selected org units."
                />
              )}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAssignmentsOpen(false)} disabled={assignmentSaving}>Cancel</Button>
          <Button onClick={() => void onSaveAssignments()} variant="contained" disabled={assignmentSaving || assignmentLoading}>
            Save Scope
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
