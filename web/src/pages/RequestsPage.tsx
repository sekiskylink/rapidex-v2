import React from 'react'
import { Alert, Box, Button, Chip, Stack, TextField, Typography } from '@mui/material'
import type { GridColDef, GridRenderCellParams } from '@mui/x-data-grid'
import { getAuthSnapshot } from '../auth/state'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { buildAdminListRequestQuery, useAdminListSearch } from '../components/admin/listSearch'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { handleAppError } from '../errors/handleAppError'
import { apiRequest } from '../lib/api'
import type { PaginatedResponse } from '../lib/pagination'
import { useAppNotify } from '../notifications/facade'
import { RequestDetailPage, type RequestDetailRecord } from './RequestDetailPage'
import { RequestForm, type RequestFormErrors, type RequestFormState, type RequestServerOption } from './RequestForm'
import type { EventRecord } from './traceability'
import { JsonMetadataDialog } from '../components/admin/JsonMetadataDialog'

interface RequestRow extends RequestDetailRecord {}

const defaultForm: RequestFormState = {
  destinationServerId: '',
  destinationServerIdsText: '',
  dependencyRequestIdsText: '',
  sourceSystem: '',
  correlationId: '',
  batchId: '',
  idempotencyKey: '',
  payloadFormat: 'json',
  submissionBinding: 'body',
  urlSuffix: '',
  payloadText: '{\n  \n}',
  metadataText: '{}',
}

function formatDate(value: string) {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return value
  }
  return parsed.toLocaleString()
}

function statusColor(status: string): 'default' | 'warning' | 'success' | 'error' | 'info' {
  switch (status) {
    case 'pending':
      return 'warning'
    case 'blocked':
      return 'warning'
    case 'processing':
      return 'info'
    case 'completed':
      return 'success'
    case 'failed':
      return 'error'
    default:
      return 'default'
  }
}

function parseJSONValue(value: string): { parsed?: unknown; error?: string } {
  try {
    return { parsed: JSON.parse(value) }
  } catch {
    return { error: 'Payload must be valid JSON.' }
  }
}

function parseTextValue(value: string): { parsed?: string; error?: string } {
  const trimmed = value.trim()
  if (!trimmed) {
    return { error: 'Payload is required.' }
  }
  return { parsed: trimmed }
}

function parseJSONObject(value: string): { parsed?: Record<string, unknown>; error?: string } {
  try {
    const parsed = JSON.parse(value || '{}') as Record<string, unknown>
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
      return { error: 'Metadata must be a JSON object.' }
    }
    return { parsed }
  } catch {
    return { error: 'Metadata must be valid JSON.' }
  }
}

function mapValidationFieldErrors(details?: Record<string, string[]>): RequestFormErrors {
  const first = (key: string) => details?.[key]?.[0] ?? ''
  return {
    destinationServerId: first('destinationServerId'),
    destinationServerIdsText: first('destinationServerIds'),
    dependencyRequestIdsText: first('dependencyRequestIds'),
    sourceSystem: first('sourceSystem'),
    correlationId: first('correlationId'),
    batchId: first('batchId'),
    idempotencyKey: first('idempotencyKey'),
    payloadFormat: first('payloadFormat'),
    submissionBinding: first('submissionBinding'),
    urlSuffix: first('urlSuffix'),
    payload: first('payload'),
    metadata: first('metadata'),
  }
}

function parseIDList(value: string): { parsed?: number[]; error?: string } {
  const trimmed = value.trim()
  if (!trimmed) {
    return { parsed: [] }
  }
  const items = trimmed
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)

  const parsed: number[] = []
  for (const item of items) {
    const next = Number(item)
    if (!Number.isInteger(next) || next <= 0) {
      return { error: 'Values must be comma-separated positive integers.' }
    }
    parsed.push(next)
  }
  return { parsed }
}

function validateForm(form: RequestFormState): RequestFormErrors {
  const errors: RequestFormErrors = {}
  if (!form.destinationServerId.trim()) {
    errors.destinationServerId = 'Destination server is required.'
  }
  const destinationIDs = parseIDList(form.destinationServerIdsText)
  if (destinationIDs.error) {
    errors.destinationServerIdsText = destinationIDs.error
  }
  const dependencyIDs = parseIDList(form.dependencyRequestIdsText)
  if (dependencyIDs.error) {
    errors.dependencyRequestIdsText = dependencyIDs.error
  }
  if (!['json', 'text'].includes(form.payloadFormat)) {
    errors.payloadFormat = 'Payload format is required.'
  }
  if (!['body', 'query'].includes(form.submissionBinding)) {
    errors.submissionBinding = 'Send As is required.'
  }
  const payload = form.payloadFormat === 'text' ? parseTextValue(form.payloadText) : parseJSONValue(form.payloadText)
  if (payload.error) {
    errors.payload = payload.error
  }
  if (form.payloadFormat === 'json' && form.submissionBinding === 'query' && payload.parsed) {
    if (!payload.parsed || Array.isArray(payload.parsed) || typeof payload.parsed !== 'object') {
      errors.payload = 'Query param JSON payload must be an object.'
    }
  }
  const metadata = parseJSONObject(form.metadataText)
  if (metadata.error) {
    errors.metadata = metadata.error
  }
  return errors
}

function toRequestPayload(form: RequestFormState) {
  const payload = form.payloadFormat === 'text' ? parseTextValue(form.payloadText).parsed : parseJSONValue(form.payloadText).parsed
  return {
    destinationServerId: Number(form.destinationServerId),
    destinationServerIds: parseIDList(form.destinationServerIdsText).parsed ?? [],
    dependencyRequestIds: parseIDList(form.dependencyRequestIdsText).parsed ?? [],
    sourceSystem: form.sourceSystem.trim(),
    correlationId: form.correlationId.trim(),
    batchId: form.batchId.trim(),
    idempotencyKey: form.idempotencyKey.trim(),
    payloadFormat: form.payloadFormat,
    submissionBinding: form.submissionBinding,
    urlSuffix: form.urlSuffix.trim(),
    payload,
    metadata: parseJSONObject(form.metadataText).parsed ?? {},
  }
}

export function RequestsPage() {
  const notify = useAppNotify()
  const permissions = getAuthSnapshot().user?.permissions ?? []
  const canWrite = permissions.includes('requests.write')

  const [reloadToken, setReloadToken] = React.useState(0)
  const { searchInput, setSearchInput, search } = useAdminListSearch()
  const [createOpen, setCreateOpen] = React.useState(false)
  const [createForm, setCreateForm] = React.useState<RequestFormState>(defaultForm)
  const [createErrors, setCreateErrors] = React.useState<RequestFormErrors>({})
  const [createErrorMessage, setCreateErrorMessage] = React.useState('')
  const [submitting, setSubmitting] = React.useState(false)
  const [servers, setServers] = React.useState<RequestServerOption[]>([])
  const [loadingServers, setLoadingServers] = React.useState(false)
  const [detailOpen, setDetailOpen] = React.useState(false)
  const [detailRequest, setDetailRequest] = React.useState<RequestDetailRecord | null>(null)
  const [detailEvents, setDetailEvents] = React.useState<EventRecord[]>([])
  const [detailError, setDetailError] = React.useState('')
  const [bodyDialog, setBodyDialog] = React.useState<{ open: boolean; title: string; body: unknown }>({
    open: false,
    title: 'Request Body',
    body: null,
  })

  const refreshGrid = () => setReloadToken((value) => value + 1)

  const loadServers = React.useCallback(async () => {
    setLoadingServers(true)
    try {
      const response = await apiRequest<PaginatedResponse<{ id: number; name: string; code: string }>>(
        '/servers?page=1&pageSize=200&sort=name:asc',
      )
      setServers(response.items.map((item) => ({ id: item.id, name: item.name, code: item.code })))
    } catch (error) {
      await handleAppError(error, {
        fallbackMessage: 'Unable to load servers.',
        notifier: notify,
      })
    } finally {
      setLoadingServers(false)
    }
  }, [notify])

  React.useEffect(() => {
    if (!canWrite) {
      return
    }
    void loadServers()
  }, [canWrite, loadServers])

  const fetchRequests = React.useCallback(
    async (params: AppDataGridFetchParams) => {
      const query = buildAdminListRequestQuery(params, { search })
      const response = await apiRequest<PaginatedResponse<RequestRow>>(`/requests?${query}`)
      return {
        rows: response.items ?? [],
        total: response.totalCount,
      }
    },
    [search],
  )

  const openCreateDialog = () => {
    setCreateErrors({})
    setCreateErrorMessage('')
    setCreateForm(defaultForm)
    setCreateOpen(true)
  }

  const openDetailDialog = async (row: RequestRow) => {
    setDetailError('')
    try {
      const [detail, events] = await Promise.all([
        apiRequest<RequestDetailRecord>(`/requests/${row.id}`),
        apiRequest<PaginatedResponse<EventRecord>>(`/requests/${row.id}/events?page=1&pageSize=50&sort=createdAt:asc`),
      ])
      setDetailRequest(detail)
      setDetailEvents(events.items ?? [])
      setDetailOpen(true)
    } catch (error) {
      setDetailRequest(null)
      setDetailEvents([])
      setDetailOpen(false)
      setDetailError('Unable to load request detail.')
      await handleAppError(error, {
        fallbackMessage: 'Unable to load request detail.',
        notifier: notify,
      })
    }
  }

  const handleCreate = async () => {
    const errors = validateForm(createForm)
    setCreateErrors(errors)
    if (Object.keys(errors).length > 0) {
      return
    }

    setSubmitting(true)
    setCreateErrorMessage('')
    try {
      await apiRequest('/requests', {
        method: 'POST',
        body: JSON.stringify(toRequestPayload(createForm)),
      })
      setCreateOpen(false)
      refreshGrid()
      notify.success('Request created.')
    } catch (error) {
      if (typeof error === 'object' && error && 'details' in error) {
        setCreateErrors(mapValidationFieldErrors((error as { details?: Record<string, string[]> }).details))
      }
      setCreateErrorMessage('Unable to create request.')
      await handleAppError(error, {
        fallbackMessage: 'Unable to create request.',
        notifier: notify,
      })
    } finally {
      setSubmitting(false)
    }
  }

  const columns = React.useMemo<GridColDef<RequestRow>[]>(
    () => [
      { field: 'uid', headerName: 'Request UID', minWidth: 220, flex: 1.2 },
      { field: 'destinationServerName', headerName: 'Destination Server', minWidth: 180, flex: 1 },
      {
        field: 'status',
        headerName: 'Status',
        minWidth: 140,
        renderCell: (params: GridRenderCellParams<RequestRow, string>) => (
          <Chip label={params.value ?? 'unknown'} size="small" color={statusColor(params.value ?? '')} />
        ),
      },
      {
        field: 'targets',
        headerName: 'Targets',
        minWidth: 140,
        valueGetter: (_value, row) => (Array.isArray(row.targets) && row.targets.length > 0 ? String(row.targets.length) : '1'),
      },
      {
        field: 'statusReason',
        headerName: 'Blocked / Reason',
        minWidth: 220,
        flex: 1,
        valueGetter: (_value, row) => {
          const deferred = row.deferredUntil ? ` until ${formatDate(row.deferredUntil)}` : ''
          return row.statusReason ? `${row.statusReason}${deferred}` : row.deferredUntil ? formatDate(row.deferredUntil) : '-'
        },
      },
      {
        field: 'latestAsyncState',
        headerName: 'Async',
        minWidth: 140,
        valueGetter: (_value, row) => (row.awaitingAsync ? row.latestAsyncState || 'awaiting' : row.latestAsyncState || '-'),
      },
      { field: 'correlationId', headerName: 'Correlation ID', minWidth: 180, flex: 1 },
      {
        field: 'createdAt',
        headerName: 'Created',
        minWidth: 200,
        flex: 1,
        valueGetter: (_value, row) => formatDate(row.createdAt),
      },
      {
        field: 'actions',
        headerName: 'Actions',
        minWidth: 96,
        sortable: false,
        filterable: false,
        renderCell: (params: GridRenderCellParams<RequestRow>) => (
          <AdminRowActions
            rowLabel={params.row.uid}
            actions={[
              {
                id: 'view',
                label: 'View',
                icon: 'view',
                onClick: () => {
                  void openDetailDialog(params.row)
                },
              },
              {
                id: 'view-body',
                label: 'View Body',
                icon: 'view',
                onClick: () => {
                  setBodyDialog({
                    open: true,
                    title: `Request Body: ${params.row.uid}`,
                    body: params.row.payload ?? params.row.payloadBody,
                  })
                },
              },
            ]}
          />
        ),
      },
    ],
    [],
  )

  return (
    <Stack spacing={3}>
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} justifyContent="space-between" alignItems={{ md: 'center' }}>
        <Box>
          <Typography variant="h4" component="h1">
            Requests
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Create, inspect, and trace Sukumad exchange requests.
          </Typography>
        </Box>
        <Stack direction="row" spacing={1}>
          <Button variant="outlined" onClick={refreshGrid}>
            Refresh
          </Button>
          {canWrite ? (
            <Button variant="contained" onClick={openCreateDialog}>
              Create Request
            </Button>
          ) : null}
        </Stack>
      </Stack>

      {detailError ? <Alert severity="error">{detailError}</Alert> : null}

      <TextField
        label="Search requests"
        placeholder="Search by UID, server, source, correlation, or batch"
        value={searchInput}
        onChange={(event) => setSearchInput(event.target.value)}
        fullWidth
      />

      <AppDataGrid
        columns={columns}
        fetchData={fetchRequests}
        storageKey="sukumad-requests-grid"
        reloadToken={reloadToken}
        externalQueryKey={search}
        pinActionsToRight
      />

      <RequestForm
        open={createOpen}
        title="Create Request"
        form={createForm}
        errors={createErrors}
        servers={servers}
        submitting={submitting}
        loadingServers={loadingServers}
        errorMessage={createErrorMessage}
        testId="web-request-create-form-grid"
        submitLabel="Create"
        onClose={() => setCreateOpen(false)}
        onSubmit={() => {
          void handleCreate()
        }}
        onChange={(patch) => setCreateForm((current) => ({ ...current, ...patch }))}
      />

      <RequestDetailPage
        open={detailOpen}
        request={detailRequest}
        events={detailEvents}
        onClose={() => {
          setDetailOpen(false)
          setDetailRequest(null)
          setDetailEvents([])
        }}
      />

      <JsonMetadataDialog
        open={bodyDialog.open}
        title={bodyDialog.title}
        metadata={bodyDialog.body}
        emptyMessage="No request body is available."
        invalidMessage="The stored request body is not valid JSON. Showing the raw body."
        onClose={() => setBodyDialog({ open: false, title: 'Request Body', body: null })}
        onCopied={() => notify.success('Request body copied.')}
      />
    </Stack>
  )
}
