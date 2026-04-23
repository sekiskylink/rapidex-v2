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
  Switch,
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

type UserFormErrors = Partial<Record<keyof UserFormState | 'roles', string>>

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

function formatDate(value: string) {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return value
  }
  return parsed.toLocaleString()
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
    roles: first('roles'),
    isActive: '',
  }
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

export function UsersPage() {
  const notify = useAppNotify()
  const canWrite = React.useMemo(() => {
    const user = getAuthSnapshot().user
    return Boolean(user?.permissions.some((permission) => permission.trim().toLowerCase() === 'users.write'))
  }, [])
  const canReadRoles = React.useMemo(() => {
    const user = getAuthSnapshot().user
    if (!user) {
      return false
    }
    return user.permissions.some((permission) => {
      const normalized = permission.trim().toLowerCase()
      return normalized === 'users.read' || normalized === 'users.write'
    })
  }, [])

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
  const [createErrorMessage, setCreateErrorMessage] = React.useState('')

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
  const [editErrorMessage, setEditErrorMessage] = React.useState('')

  const [detailsOpen, setDetailsOpen] = React.useState(false)
  const [detailsUser, setDetailsUser] = React.useState<UserRow | null>(null)
  const [assignmentsOpen, setAssignmentsOpen] = React.useState(false)
  const [assignmentsUser, setAssignmentsUser] = React.useState<UserRow | null>(null)
  const [assignmentOptions, setAssignmentOptions] = React.useState<OrgUnitOption[]>([])
  const [assignmentSelection, setAssignmentSelection] = React.useState<OrgUnitOption[]>([])
  const [assignmentInitialIDs, setAssignmentInitialIDs] = React.useState<number[]>([])
  const [assignmentSearch, setAssignmentSearch] = React.useState('')
  const [assignmentLoading, setAssignmentLoading] = React.useState(false)
  const [assignmentSaving, setAssignmentSaving] = React.useState(false)
  const [assignmentErrorMessage, setAssignmentErrorMessage] = React.useState('')

  const [submitting, setSubmitting] = React.useState(false)

  const refreshGrid = React.useCallback(() => setReloadToken((value) => value + 1), [])

  const loadRoleOptions = React.useCallback(async () => {
    if (!canReadRoles) {
      return
    }
    setLoadingRoleOptions(true)
    try {
      const payload = await apiRequest<PaginatedResponse<RoleOption>>('/admin/roles?page=1&pageSize=200&sort=name:asc')
      setRoleOptions(Array.isArray(payload.items) ? payload.items : [])
    } catch (error) {
      setRoleOptions([])
      await handleAppError(error, {
        fallbackMessage: 'Unable to load roles.',
        notifier: notify,
      })
    } finally {
      setLoadingRoleOptions(false)
    }
  }, [canReadRoles, notify])

  React.useEffect(() => {
    void loadRoleOptions()
  }, [loadRoleOptions])

  const loadOrgUnitOptions = React.useCallback(async (search: string) => {
    const response = await apiRequest<PaginatedResponse<OrgUnitOption>>(`/orgunits?page=0&pageSize=50&search=${encodeURIComponent(search.trim())}`)
    return Array.isArray(response.items) ? response.items : []
  }, [])

  const fetchUsers = React.useCallback(async (params: AppDataGridFetchParams) => {
    const query = buildAdminListRequestQuery(params, { search })
    const response = await apiRequest<PaginatedResponse<UserRow>>(`/users?${query}`)
    return {
      rows: response.items,
      total: response.totalCount,
    }
  }, [search])

  const saveOrgUnitAssignments = React.useCallback(async (userID: number, initialIDs: number[], selection: OrgUnitOption[]) => {
    const nextIDs = selection.map((item) => item.id)
    const toRemove = initialIDs.filter((id) => !nextIDs.includes(id))
    const toAdd = nextIDs.filter((id) => !initialIDs.includes(id))
    for (const orgUnitID of toRemove) {
      await apiRequest(`/user-org-units/${userID}/${orgUnitID}`, { method: 'DELETE' })
    }
    for (const orgUnitID of toAdd) {
      await apiRequest('/user-org-units', {
        method: 'POST',
        body: JSON.stringify({ userId: userID, orgUnitId: orgUnitID }),
      })
    }
    return nextIDs
  }, [])

  const loadUserOrgUnitAssignments = React.useCallback(async (userID: number) => {
    const [currentAssignments, availableOptions] = await Promise.all([
      apiRequest<UserOrgUnitAssignmentsResponse>(`/user-org-units/${userID}`),
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
  }, [loadOrgUnitOptions])

  const onCreateAssignmentSearchChange = React.useCallback(async (value: string) => {
    setCreateAssignmentSearch(value)
    try {
      const options = await loadOrgUnitOptions(value)
      setCreateAssignmentOptions((current) => mergeOrgUnitOptions(current, options, createAssignmentSelection))
    } catch {
      // Keep the current options if incremental search fails.
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
    setCreateErrorMessage('')
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
    setEditErrorMessage('')
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
      // Keep the current options if incremental search fails.
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
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to load org unit assignments.',
        notifier: notify,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setEditErrorMessage(`Unable to load org unit assignments.${requestId}`)
      setEditAssignmentOptions([])
      setEditAssignmentSelection([])
      setEditAssignmentInitialIDs([])
    } finally {
      setEditAssignmentLoading(false)
    }
  }, [loadUserOrgUnitAssignments, notify, resetEditDialog])

  const onCreateUser = async () => {
    setSubmitting(true)
    setCreateErrors({})
    setCreateErrorMessage('')
    try {
      const created = await apiRequest<UserRow>('/users', {
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
          const { error: normalized } = await handleAppError(error, {
            fallbackMessage: 'User created, but org unit assignments could not be saved.',
            notifier: notify,
          })
          const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
          notify.error(`User created, but org unit assignments could not be saved.${requestId ? requestId : ''} Update Org Unit Scope from the user actions menu.`)
        }
      }
      notify.success('User created.')
      setCreateOpen(false)
      resetCreateDialog()
      refreshGrid()
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to create user.',
        notifier: notify,
        onValidationError: (fieldErrors) => setCreateErrors(mapValidationFieldErrors(fieldErrors)),
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setCreateErrorMessage(`${normalized.message}${requestId}`)
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
      roles: editRoles,
    }
    if (editForm.password.trim()) {
      payload.password = editForm.password
    }

    try {
      await apiRequest(`/users/${editUser.id}`, {
        method: 'PATCH',
        body: JSON.stringify(payload),
      })
      await saveOrgUnitAssignments(editUser.id, editAssignmentInitialIDs, editAssignmentSelection)
      notify.success('User updated.')
      setEditOpen(false)
      resetEditDialog()
      refreshGrid()
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to update user.',
        notifier: notify,
        onValidationError: (fieldErrors) => setEditErrors(mapValidationFieldErrors(fieldErrors)),
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setEditErrorMessage(`${normalized.message}${requestId}`)
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
        notify.success(`User ${nextActive ? 'activated' : 'deactivated'}.`)
        refreshGrid()
      } catch (error) {
        await handleAppError(error, {
          fallbackMessage: 'Unable to update active status.',
          notifier: notify,
        })
      }
    },
    [notify, refreshGrid],
  )

  const openAssignmentsDialog = React.useCallback(async (row: UserRow) => {
    setAssignmentsUser(row)
    setAssignmentsOpen(true)
    setAssignmentLoading(true)
    setAssignmentSaving(false)
    setAssignmentErrorMessage('')
    setAssignmentSearch('')
    try {
      const assignmentState = await loadUserOrgUnitAssignments(row.id)
      setAssignmentSelection(assignmentState.selection)
      setAssignmentInitialIDs(assignmentState.initialIDs)
      setAssignmentOptions(assignmentState.options)
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to load org unit assignments.',
        notifier: notify,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setAssignmentErrorMessage(`${normalized.message}${requestId}`)
      setAssignmentSelection([])
      setAssignmentInitialIDs([])
      setAssignmentOptions([])
    } finally {
      setAssignmentLoading(false)
    }
  }, [loadUserOrgUnitAssignments, notify])

  const onAssignmentSearchChange = React.useCallback(async (value: string) => {
    setAssignmentSearch(value)
    if (!assignmentsOpen) {
      return
    }
    try {
      const options = await loadOrgUnitOptions(value)
      setAssignmentOptions((current) => mergeOrgUnitOptions(current, options, assignmentSelection))
    } catch {
      // Keep the current options if incremental search fails.
    }
  }, [assignmentSelection, assignmentsOpen, loadOrgUnitOptions])

  const onSaveAssignments = React.useCallback(async () => {
    if (!assignmentsUser) {
      return
    }
    setAssignmentSaving(true)
    setAssignmentErrorMessage('')
    try {
      const nextIDs = await saveOrgUnitAssignments(assignmentsUser.id, assignmentInitialIDs, assignmentSelection)
      notify.success('Org unit assignments updated.')
      setAssignmentsOpen(false)
      setAssignmentsUser(null)
      setAssignmentInitialIDs(nextIDs)
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to save org unit assignments.',
        notifier: notify,
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setAssignmentErrorMessage(`${normalized.message}${requestId}`)
    } finally {
      setAssignmentSaving(false)
    }
  }, [assignmentInitialIDs, assignmentSelection, assignmentsUser, notify, saveOrgUnitAssignments])

  const roleNames = React.useMemo(() => roleOptions.map((role) => role.name), [roleOptions])

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
      {
        field: 'roles',
        headerName: 'Roles',
        minWidth: 220,
        flex: 1,
        sortable: false,
        renderCell: (params: GridRenderCellParams<UserRow>) => renderRoleChips(params.row.roles ?? []),
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
        valueGetter: (_value, row) => formatDate(row.updatedAt),
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
                id: 'jurisdiction',
                label: 'Org Unit Scope',
                icon: 'view',
                disabled: !canWrite,
                onClick: () => {
                  void openAssignmentsDialog(params.row)
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
    [canWrite, onToggleActive, openEditDialog],
  )

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 2 }}>
        <Box>
          <Typography variant="h5" component="h1" gutterBottom>
            Users
          </Typography>
          <Typography color="text.secondary">Manage users, profile metadata, and multi-role assignments.</Typography>
        </Box>
        {canWrite ? (
          <Button
            variant="contained"
            onClick={() => void openCreateDialog()}
          >
            Create User
          </Button>
        ) : null}
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
          enablePinnedColumns
          stickyRightFields={['actions']}
        />
      </Box>

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} fullWidth maxWidth="lg">
        <DialogTitle>Create User</DialogTitle>
        <DialogContent>
          <Box
            data-testid="web-user-create-form-grid"
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
            {createErrorMessage ? <Alert severity="error" sx={{ gridColumn: '1 / -1' }}>{createErrorMessage}</Alert> : null}
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
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Roles"
                  placeholder="Assign roles"
                  error={Boolean(createErrors.roles)}
                  helperText={createErrors.roles}
                />
              )}
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
              sx={{ gridColumn: '1 / -1' }}
              renderTags={(value, getTagProps) =>
                value.map((option, index) => (
                  <Chip label={option.displayPath ? `${option.name} (${option.displayPath})` : option.name} {...getTagProps({ index })} key={option.id} size="small" />
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
            <Stack direction="row" alignItems="center" spacing={1} sx={{ gridColumn: '1 / -1' }}>
              <Typography variant="body2">Active</Typography>
              <Switch
                checked={createForm.isActive}
                onChange={(event) => setCreateForm((current) => ({ ...current, isActive: event.target.checked }))}
                inputProps={{ 'aria-label': 'Active' }}
              />
            </Stack>
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
            data-testid="web-user-edit-form-grid"
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
            {editErrorMessage ? <Alert severity="error" sx={{ gridColumn: '1 / -1' }}>{editErrorMessage}</Alert> : null}
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
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Roles"
                  placeholder="Assign roles"
                  error={Boolean(editErrors.roles)}
                  helperText={editErrors.roles}
                />
              )}
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
              sx={{ gridColumn: '1 / -1' }}
              renderTags={(value, getTagProps) =>
                value.map((option, index) => (
                  <Chip label={option.displayPath ? `${option.name} (${option.displayPath})` : option.name} {...getTagProps({ index })} key={option.id} size="small" />
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
            <Stack direction="row" alignItems="center" spacing={1} sx={{ gridColumn: '1 / -1' }}>
              <Typography variant="body2">Active</Typography>
              <Switch
                checked={editForm.isActive}
                onChange={(event) => setEditForm((current) => ({ ...current, isActive: event.target.checked }))}
                inputProps={{ 'aria-label': 'Active' }}
              />
            </Stack>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={() => void onSaveEdit()} disabled={submitting || !editUser}>
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
            {assignmentErrorMessage ? <Alert severity="error">{assignmentErrorMessage}</Alert> : null}
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
              renderTags={(value, getTagProps) =>
                value.map((option, index) => (
                  <Chip label={option.displayPath ? `${option.name} (${option.displayPath})` : option.name} {...getTagProps({ index })} key={option.id} size="small" />
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
    </Box>
  )
}
