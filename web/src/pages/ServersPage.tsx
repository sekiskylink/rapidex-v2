import React from 'react'
import {
  Alert,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  MenuItem,
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

interface ServerRow {
  id: number
  uid: string
  name: string
  code: string
  systemType: string
  baseUrl: string
  endpointType: string
  httpMethod: string
  useAsync: boolean
  parseResponses: boolean
  responseBodyPersistence: string
  headers: Record<string, string>
  urlParams: Record<string, string>
  suspended: boolean
  createdAt: string
  updatedAt: string
}

interface ServerFormState {
  name: string
  code: string
  systemType: string
  baseUrl: string
  endpointType: string
  httpMethod: string
  useAsync: boolean
  parseResponses: boolean
  responseBodyPersistence: string
  suspended: boolean
  headersText: string
  urlParamsText: string
}

type ServerFormErrors = Partial<Record<keyof ServerFormState | 'headers' | 'urlParams', string>>

const defaultForm: ServerFormState = {
  name: '',
  code: '',
  systemType: 'dhis2',
  baseUrl: '',
  endpointType: 'http',
  httpMethod: 'POST',
  useAsync: false,
  parseResponses: true,
  responseBodyPersistence: 'filter',
  suspended: false,
  headersText: '{}',
  urlParamsText: '{}',
}

function formatDate(value: string) {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return value
  }
  return parsed.toLocaleString()
}

function mapValidationFieldErrors(details?: Record<string, string[]>): ServerFormErrors {
  const first = (key: string) => details?.[key]?.[0] ?? ''
  return {
    name: first('name'),
    code: first('code'),
    systemType: first('systemType'),
    baseUrl: first('baseUrl'),
    endpointType: first('endpointType'),
    httpMethod: first('httpMethod'),
    headers: first('headers'),
    urlParams: first('urlParams'),
  }
}

function toPrettyJSON(value: Record<string, string>) {
  return JSON.stringify(value ?? {}, null, 2)
}

function toFormState(row: ServerRow): ServerFormState {
  return {
    name: row.name,
    code: row.code,
    systemType: row.systemType,
    baseUrl: row.baseUrl,
    endpointType: row.endpointType,
    httpMethod: row.httpMethod,
    useAsync: row.useAsync,
    parseResponses: row.parseResponses,
    responseBodyPersistence: row.responseBodyPersistence || 'filter',
    suspended: row.suspended,
    headersText: toPrettyJSON(row.headers),
    urlParamsText: toPrettyJSON(row.urlParams),
  }
}

function parseStringMap(value: string, field: 'headers' | 'urlParams'): { parsed?: Record<string, string>; error?: string } {
  try {
    const raw = JSON.parse(value || '{}') as Record<string, unknown>
    if (!raw || Array.isArray(raw) || typeof raw !== 'object') {
      return { error: `${field === 'headers' ? 'Headers' : 'URL parameters'} must be a JSON object.` }
    }
    const parsed: Record<string, string> = {}
    for (const [key, entry] of Object.entries(raw)) {
      parsed[key] = typeof entry === 'string' ? entry : JSON.stringify(entry)
    }
    return { parsed }
  } catch {
    return { error: `${field === 'headers' ? 'Headers' : 'URL parameters'} must be valid JSON.` }
  }
}

function validateClientForm(form: ServerFormState): ServerFormErrors {
  const errors: ServerFormErrors = {}
  if (!form.name.trim()) {
    errors.name = 'Name is required.'
  }
  if (!form.code.trim()) {
    errors.code = 'Code is required.'
  }
  if (!form.baseUrl.trim()) {
    errors.baseUrl = 'Base URL is required.'
  }
  if (!form.systemType.trim()) {
    errors.systemType = 'System type is required.'
  }
  if (!form.endpointType.trim()) {
    errors.endpointType = 'Endpoint type is required.'
  }
  if (!form.httpMethod.trim()) {
    errors.httpMethod = 'HTTP method is required.'
  }

  const headers = parseStringMap(form.headersText, 'headers')
  if (headers.error) {
    errors.headers = headers.error
  }
  const urlParams = parseStringMap(form.urlParamsText, 'urlParams')
  if (urlParams.error) {
    errors.urlParams = urlParams.error
  }

  return errors
}

function toRequestPayload(form: ServerFormState) {
  return {
    name: form.name.trim(),
    code: form.code.trim(),
    systemType: form.systemType.trim(),
    baseUrl: form.baseUrl.trim(),
    endpointType: form.endpointType.trim(),
    httpMethod: form.httpMethod.trim(),
    useAsync: form.useAsync,
    parseResponses: form.parseResponses,
    responseBodyPersistence: form.responseBodyPersistence,
    suspended: form.suspended,
    headers: parseStringMap(form.headersText, 'headers').parsed ?? {},
    urlParams: parseStringMap(form.urlParamsText, 'urlParams').parsed ?? {},
  }
}

interface ServerFormDialogProps {
  open: boolean
  title: string
  form: ServerFormState
  errors: ServerFormErrors
  submitting: boolean
  errorMessage: string
  testId: string
  submitLabel: string
  onClose: () => void
  onSubmit: () => void
  onChange: (patch: Partial<ServerFormState>) => void
}

function ServerFormDialog({
  open,
  title,
  form,
  errors,
  submitting,
  errorMessage,
  testId,
  submitLabel,
  onClose,
  onSubmit,
  onChange,
}: ServerFormDialogProps) {
  return (
    <Dialog open={open} onClose={submitting ? undefined : onClose} maxWidth="md" fullWidth>
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }} data-testid={testId}>
          {errorMessage ? <Alert severity="error">{errorMessage}</Alert> : null}
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField
              label="Server Name"
              value={form.name}
              onChange={(event) => onChange({ name: event.target.value })}
              error={Boolean(errors.name)}
              helperText={errors.name}
              fullWidth
            />
            <TextField
              label="Code"
              value={form.code}
              onChange={(event) => onChange({ code: event.target.value })}
              error={Boolean(errors.code)}
              helperText={errors.code}
              fullWidth
            />
          </Stack>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField
              label="System Type"
              value={form.systemType}
              onChange={(event) => onChange({ systemType: event.target.value })}
              error={Boolean(errors.systemType)}
              helperText={errors.systemType}
              fullWidth
            />
            <TextField
              label="Endpoint Type"
              value={form.endpointType}
              onChange={(event) => onChange({ endpointType: event.target.value })}
              error={Boolean(errors.endpointType)}
              helperText={errors.endpointType}
              fullWidth
            />
            <TextField
              label="HTTP Method"
              value={form.httpMethod}
              onChange={(event) => onChange({ httpMethod: event.target.value.toUpperCase() })}
              error={Boolean(errors.httpMethod)}
              helperText={errors.httpMethod}
              fullWidth
            />
          </Stack>
          <TextField
            label="Base URL"
            value={form.baseUrl}
            onChange={(event) => onChange({ baseUrl: event.target.value })}
            error={Boolean(errors.baseUrl)}
            helperText={errors.baseUrl}
            fullWidth
          />
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField
              label="Headers JSON"
              value={form.headersText}
              onChange={(event) => onChange({ headersText: event.target.value })}
              error={Boolean(errors.headers)}
              helperText={errors.headers || 'JSON object of default request headers.'}
              fullWidth
              minRows={6}
              multiline
            />
            <TextField
              label="URL Params JSON"
              value={form.urlParamsText}
              onChange={(event) => onChange({ urlParamsText: event.target.value })}
              error={Boolean(errors.urlParams)}
              helperText={errors.urlParams || 'JSON object of default query parameters.'}
              fullWidth
              minRows={6}
              multiline
            />
          </Stack>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField
              select
              label="Response Body"
              value={form.responseBodyPersistence}
              onChange={(event) => onChange({ responseBodyPersistence: event.target.value })}
              error={Boolean(errors.responseBodyPersistence)}
              helperText={errors.responseBodyPersistence || 'Default policy for saving destination response bodies.'}
              fullWidth
            >
              <MenuItem value="filter">Use response filter</MenuItem>
              <MenuItem value="save">Always save</MenuItem>
              <MenuItem value="discard">Never save</MenuItem>
            </TextField>
            <FormControlLabel
              control={<Switch checked={form.useAsync} onChange={(event) => onChange({ useAsync: event.target.checked })} />}
              label="Use async processing"
            />
            <FormControlLabel
              control={
                <Switch checked={form.parseResponses} onChange={(event) => onChange({ parseResponses: event.target.checked })} />
              }
              label="Parse responses"
            />
            <FormControlLabel
              control={<Switch checked={form.suspended} onChange={(event) => onChange({ suspended: event.target.checked })} />}
              label="Suspended"
            />
          </Stack>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={submitting}>
          Cancel
        </Button>
        <Button onClick={onSubmit} disabled={submitting} variant="contained">
          {submitLabel}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export function ServersPage() {
  const notify = useAppNotify()
  const permissions = React.useMemo(() => getAuthSnapshot().user?.permissions ?? [], [])
  const canWrite = permissions.includes('servers.write')

  const [reloadToken, setReloadToken] = React.useState(0)
  const { searchInput, setSearchInput, search } = useAdminListSearch()

  const [createOpen, setCreateOpen] = React.useState(false)
  const [createForm, setCreateForm] = React.useState<ServerFormState>(defaultForm)
  const [createErrors, setCreateErrors] = React.useState<ServerFormErrors>({})
  const [createErrorMessage, setCreateErrorMessage] = React.useState('')

  const [editOpen, setEditOpen] = React.useState(false)
  const [editServer, setEditServer] = React.useState<ServerRow | null>(null)
  const [editForm, setEditForm] = React.useState<ServerFormState>(defaultForm)
  const [editErrors, setEditErrors] = React.useState<ServerFormErrors>({})
  const [editErrorMessage, setEditErrorMessage] = React.useState('')
  const [loadingDetails, setLoadingDetails] = React.useState(false)

  const [submitting, setSubmitting] = React.useState(false)

  const refreshGrid = React.useCallback(() => setReloadToken((value) => value + 1), [])

  const fetchServers = React.useCallback(async (params: AppDataGridFetchParams) => {
    const query = buildAdminListRequestQuery(params, { search })
    const response = await apiRequest<PaginatedResponse<ServerRow>>(`/servers?${query}`)
    return {
      rows: response.items,
      total: response.totalCount,
    }
  }, [search])

  const openCreateDialog = () => {
    setCreateErrors({})
    setCreateErrorMessage('')
    setCreateForm(defaultForm)
    setCreateOpen(true)
  }

  const openEditDialog = async (row: ServerRow) => {
    setLoadingDetails(true)
    setEditErrors({})
    setEditErrorMessage('')
    try {
      const detail = await apiRequest<ServerRow>(`/servers/${row.id}`)
      setEditServer(detail)
      setEditForm(toFormState(detail))
      setEditOpen(true)
    } catch (error) {
      await handleAppError(error, {
        fallbackMessage: 'Unable to load server details.',
        notifier: notify,
      })
    } finally {
      setLoadingDetails(false)
    }
  }

  const submitCreate = async () => {
    const validation = validateClientForm(createForm)
    if (Object.keys(validation).length > 0) {
      setCreateErrors(validation)
      return
    }

    setSubmitting(true)
    setCreateErrors({})
    setCreateErrorMessage('')
    try {
      await apiRequest('/servers', {
        method: 'POST',
        body: JSON.stringify(toRequestPayload(createForm)),
      })
      notify.success('Server created.')
      setCreateOpen(false)
      setCreateForm(defaultForm)
      refreshGrid()
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to create server.',
        notifier: notify,
        onValidationError: (fieldErrors) => setCreateErrors(mapValidationFieldErrors(fieldErrors)),
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setCreateErrorMessage(`${normalized.message}${requestId}`)
    } finally {
      setSubmitting(false)
    }
  }

  const submitEdit = async () => {
    if (!editServer) {
      return
    }
    const validation = validateClientForm(editForm)
    if (Object.keys(validation).length > 0) {
      setEditErrors(validation)
      return
    }

    setSubmitting(true)
    setEditErrors({})
    setEditErrorMessage('')
    try {
      await apiRequest(`/servers/${editServer.id}`, {
        method: 'PUT',
        body: JSON.stringify(toRequestPayload(editForm)),
      })
      notify.success('Server updated.')
      setEditOpen(false)
      setEditServer(null)
      refreshGrid()
    } catch (error) {
      const { error: normalized } = await handleAppError(error, {
        fallbackMessage: 'Unable to update server.',
        notifier: notify,
        onValidationError: (fieldErrors) => setEditErrors(mapValidationFieldErrors(fieldErrors)),
      })
      const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
      setEditErrorMessage(`${normalized.message}${requestId}`)
    } finally {
      setSubmitting(false)
    }
  }

  const toggleSuspended = async (row: ServerRow) => {
    try {
      await apiRequest(`/servers/${row.id}`, {
        method: 'PUT',
        body: JSON.stringify({
          ...row,
          suspended: !row.suspended,
        }),
      })
      notify.success(row.suspended ? 'Server activated.' : 'Server suspended.')
      refreshGrid()
    } catch (error) {
      await handleAppError(error, {
        fallbackMessage: 'Unable to update server status.',
        notifier: notify,
      })
    }
  }

  const deleteServer = async (row: ServerRow) => {
    try {
      await apiRequest(`/servers/${row.id}`, {
        method: 'DELETE',
      })
      notify.success('Server deleted.')
      refreshGrid()
    } catch (error) {
      await handleAppError(error, {
        fallbackMessage: 'Unable to delete server.',
        notifier: notify,
      })
    }
  }

  const columns = React.useMemo<GridColDef<ServerRow>[]>(() => [
    { field: 'name', headerName: 'Server Name', flex: 1.1, minWidth: 180 },
    { field: 'code', headerName: 'Code', flex: 0.8, minWidth: 120 },
    { field: 'systemType', headerName: 'System Type', flex: 0.8, minWidth: 130 },
    { field: 'baseUrl', headerName: 'Base URL', flex: 1.4, minWidth: 220 },
    {
      field: 'useAsync',
      headerName: 'Async',
      minWidth: 110,
      renderCell: (params: GridRenderCellParams<ServerRow, boolean>) => (
        <Chip size="small" color={params.value ? 'primary' : 'default'} label={params.value ? 'Enabled' : 'Disabled'} />
      ),
    },
    {
      field: 'suspended',
      headerName: 'Status',
      minWidth: 120,
      renderCell: (params: GridRenderCellParams<ServerRow, boolean>) => (
        <Chip size="small" color={params.value ? 'warning' : 'success'} label={params.value ? 'Suspended' : 'Active'} />
      ),
    },
    {
      field: 'updatedAt',
      headerName: 'Updated',
      minWidth: 180,
      valueGetter: (_value, row) => formatDate(row.updatedAt),
    },
    {
      field: 'actions',
      headerName: 'Actions',
      sortable: false,
      filterable: false,
      minWidth: 110,
      renderCell: (params: GridRenderCellParams<ServerRow>) => (
        <AdminRowActions
          rowLabel={params.row.code}
          actions={[
            {
              id: 'edit',
              label: 'Edit',
              icon: 'edit',
              visible: canWrite,
              disabled: loadingDetails,
              onClick: () => {
                void openEditDialog(params.row)
              },
            },
            {
              id: 'toggle-suspended',
              label: params.row.suspended ? 'Activate' : 'Suspend',
              icon: 'edit',
              visible: canWrite,
              onClick: () => {
                void toggleSuspended(params.row)
              },
              confirmTitle: params.row.suspended ? 'Activate server' : 'Suspend server',
              confirmMessage: params.row.suspended
                ? `Activate ${params.row.name}?`
                : `Suspend ${params.row.name}?`,
            },
            {
              id: 'delete',
              label: 'Delete',
              icon: 'delete',
              visible: canWrite,
              destructive: true,
              confirmTitle: 'Delete server',
              confirmMessage: `Delete ${params.row.name}? This cannot be undone.`,
              onClick: () => {
                void deleteServer(params.row)
              },
            },
          ]}
        />
      ),
    },
  ], [canWrite, loadingDetails])

  return (
    <Stack spacing={2.5}>
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} justifyContent="space-between" alignItems={{ md: 'center' }}>
        <Box>
          <Typography variant="h4" component="h1" gutterBottom>
            Servers
          </Typography>
          <Typography color="text.secondary">
            Manage DHIS2 instances and other downstream integration endpoints.
          </Typography>
        </Box>
        {canWrite ? (
          <Button variant="contained" onClick={openCreateDialog}>
            Create Server
          </Button>
        ) : null}
      </Stack>

      <TextField
        label="Search servers"
        placeholder="Search by name, code, system type, or URL"
        value={searchInput}
        onChange={(event) => setSearchInput(event.target.value)}
        fullWidth
      />

      <AppDataGrid
        columns={columns}
        fetchData={fetchServers}
        storageKey="servers-grid"
        reloadToken={reloadToken}
        externalQueryKey={search}
        pinActionsToRight
      />

      <ServerFormDialog
        open={createOpen}
        title="Create Server"
        form={createForm}
        errors={createErrors}
        submitting={submitting}
        errorMessage={createErrorMessage}
        testId="web-server-create-form-grid"
        submitLabel="Create"
        onClose={() => setCreateOpen(false)}
        onSubmit={() => {
          void submitCreate()
        }}
        onChange={(patch) => setCreateForm((current) => ({ ...current, ...patch }))}
      />

      <ServerFormDialog
        open={editOpen}
        title="Edit Server"
        form={editForm}
        errors={editErrors}
        submitting={submitting}
        errorMessage={editErrorMessage}
        testId="web-server-edit-form-grid"
        submitLabel="Save"
        onClose={() => {
          setEditOpen(false)
          setEditServer(null)
        }}
        onSubmit={() => {
          void submitEdit()
        }}
        onChange={(patch) => setEditForm((current) => ({ ...current, ...patch }))}
      />
    </Stack>
  )
}
