import React from 'react'
import { Box, Button, Chip, Stack, TextField, Typography } from '@mui/material'
import type { GridColDef, GridRenderCellParams } from '@mui/x-data-grid'
import type { PaginatedResponse } from '../api/pagination'
import { useApiClient } from '../api/useApiClient'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { EventDetailDialog, formatTraceDate, traceLevelColor, type EventRecord, type TraceResult } from './traceability'

interface WorkerRow {
  id: number
  uid: string
  workerType: string
  workerName: string
  status: string
  startedAt: string
  lastHeartbeatAt?: string | null
}

interface RateLimitRow {
  id: number
  uid: string
  name: string
  scopeType: string
  scopeRef: string
  rps: number
  burst: number
  maxConcurrency: number
  timeoutMs: number
  isActive: boolean
}

function statusColor(status: string): 'default' | 'warning' | 'success' | 'error' | 'info' {
  switch (status) {
    case 'starting':
      return 'warning'
    case 'running':
      return 'success'
    case 'failed':
      return 'error'
    case 'stopped':
      return 'default'
    default:
      return 'info'
  }
}

export function ObservabilityPage() {
  const apiClient = useApiClient()
  const [eventType, setEventType] = React.useState('')
  const [level, setLevel] = React.useState('')
  const [correlationId, setCorrelationId] = React.useState('')
  const [from, setFrom] = React.useState('')
  const [to, setTo] = React.useState('')
  const [selectedEvent, setSelectedEvent] = React.useState<EventRecord | null>(null)
  const [traceResult, setTraceResult] = React.useState<TraceResult | null>(null)

  const fetchEvents = React.useCallback(
    async (params: AppDataGridFetchParams) => {
      const query = new URLSearchParams({
        page: String(params.page + 1),
        pageSize: String(params.pageSize),
        sort: 'createdAt:desc',
      })
      if (eventType.trim()) {
        query.set('eventType', eventType.trim())
      }
      if (level.trim()) {
        query.set('level', level.trim())
      }
      if (correlationId.trim()) {
        query.set('correlationId', correlationId.trim())
      }
      if (from) {
        query.set('from', new Date(from).toISOString())
      }
      if (to) {
        query.set('to', new Date(to).toISOString())
      }
      const response = await apiClient.request<PaginatedResponse<EventRecord>>(`/api/v1/observability/events?${query.toString()}`)
      return { rows: response.items ?? [], total: response.totalCount }
    },
    [apiClient, correlationId, eventType, from, level, to],
  )

  const fetchWorkers = React.useCallback(
    async (params: AppDataGridFetchParams) => {
      const query = new URLSearchParams({
        page: String(params.page + 1),
        pageSize: String(params.pageSize),
      })
      const response = await apiClient.request<PaginatedResponse<WorkerRow>>(`/api/v1/observability/workers?${query.toString()}`)
      return { rows: response.items ?? [], total: response.totalCount }
    },
    [apiClient],
  )

  const fetchRateLimits = React.useCallback(
    async (params: AppDataGridFetchParams) => {
      const query = new URLSearchParams({
        page: String(params.page + 1),
        pageSize: String(params.pageSize),
      })
      const response = await apiClient.request<PaginatedResponse<RateLimitRow>>(`/api/v1/observability/rate-limits?${query.toString()}`)
      return { rows: response.items ?? [], total: response.totalCount }
    },
    [apiClient],
  )

  const loadEventDetail = async (id: number) => {
    const detail = await apiClient.request<EventRecord>(`/api/v1/observability/events/${id}`)
    setSelectedEvent(detail)
  }

  const loadTrace = async () => {
    if (!correlationId.trim()) {
      setTraceResult(null)
      return
    }
    const trace = await apiClient.request<TraceResult>(`/api/v1/observability/trace?correlationId=${encodeURIComponent(correlationId.trim())}`)
    setTraceResult(trace)
  }

  const eventColumns = React.useMemo<GridColDef<EventRecord>[]>(
    () => [
      { field: 'createdAt', headerName: 'Timestamp', minWidth: 190, flex: 1, valueGetter: (value) => formatTraceDate(String(value ?? '')) },
      { field: 'eventType', headerName: 'Event Type', minWidth: 180, flex: 1 },
      {
        field: 'eventLevel',
        headerName: 'Level',
        minWidth: 120,
        renderCell: (params: GridRenderCellParams<EventRecord, string>) => (
          <Chip size="small" label={params.value ?? 'info'} color={traceLevelColor(params.value ?? '')} />
        ),
      },
      { field: 'message', headerName: 'Message', minWidth: 240, flex: 1.5 },
      { field: 'correlationId', headerName: 'Correlation ID', minWidth: 180, flex: 1 },
      { field: 'requestUid', headerName: 'Request', minWidth: 160, flex: 1 },
      { field: 'deliveryUid', headerName: 'Delivery', minWidth: 160, flex: 1 },
      { field: 'asyncTaskUid', headerName: 'Job', minWidth: 160, flex: 1 },
      { field: 'sourceComponent', headerName: 'Source', minWidth: 160, flex: 1 },
      {
        field: 'actions',
        headerName: 'Actions',
        minWidth: 96,
        sortable: false,
        filterable: false,
        renderCell: (params: GridRenderCellParams<EventRecord>) => (
          <AdminRowActions
            rowLabel={params.row.eventType}
            actions={[
              {
                id: 'view',
                label: 'View',
                icon: 'view',
                onClick: () => {
                  void loadEventDetail(params.row.id)
                },
              },
            ]}
          />
        ),
      },
    ],
    [],
  )

  const workerColumns = React.useMemo<GridColDef<WorkerRow>[]>(
    () => [
      { field: 'workerName', headerName: 'Worker Name', minWidth: 180, flex: 1 },
      { field: 'workerType', headerName: 'Worker Type', minWidth: 140, flex: 1 },
      {
        field: 'status',
        headerName: 'Status',
        minWidth: 130,
        renderCell: (params: GridRenderCellParams<WorkerRow, string>) => (
          <Chip label={params.value ?? 'unknown'} size="small" color={statusColor(params.value ?? '')} />
        ),
      },
      { field: 'startedAt', headerName: 'Started', minWidth: 190, flex: 1, valueGetter: (value) => formatTraceDate(String(value ?? '')) },
      {
        field: 'lastHeartbeatAt',
        headerName: 'Last Heartbeat',
        minWidth: 190,
        flex: 1,
        valueGetter: (value) => formatTraceDate(String(value ?? '')),
      },
    ],
    [],
  )

  const rateLimitColumns = React.useMemo<GridColDef<RateLimitRow>[]>(
    () => [
      { field: 'name', headerName: 'Policy', minWidth: 180, flex: 1 },
      { field: 'scopeType', headerName: 'Scope Type', minWidth: 140, flex: 1 },
      { field: 'scopeRef', headerName: 'Scope Ref', minWidth: 160, flex: 1 },
      { field: 'rps', headerName: 'RPS', minWidth: 90, type: 'number' },
      { field: 'burst', headerName: 'Burst', minWidth: 90, type: 'number' },
      { field: 'maxConcurrency', headerName: 'Max Concurrency', minWidth: 150, type: 'number' },
      {
        field: 'isActive',
        headerName: 'Active',
        minWidth: 120,
        renderCell: (params: GridRenderCellParams<RateLimitRow, boolean>) => (
          <Chip label={params.value ? 'Active' : 'Inactive'} size="small" color={params.value ? 'success' : 'default'} />
        ),
      },
    ],
    [],
  )

  return (
    <Stack spacing={3}>
      <Box>
        <Typography variant="h4" component="h1">
          Observability
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Inspect Sukumad event history, correlation traces, workers, and active rate-limit policies.
        </Typography>
      </Box>

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
        <TextField label="Event type" value={eventType} onChange={(event) => setEventType(event.target.value)} />
        <TextField label="Level" value={level} onChange={(event) => setLevel(event.target.value)} placeholder="info | warning | error" />
        <TextField label="Correlation ID" value={correlationId} onChange={(event) => setCorrelationId(event.target.value)} />
        <TextField label="From" type="datetime-local" value={from} onChange={(event) => setFrom(event.target.value)} InputLabelProps={{ shrink: true }} />
        <TextField label="To" type="datetime-local" value={to} onChange={(event) => setTo(event.target.value)} InputLabelProps={{ shrink: true }} />
        <Button variant="outlined" onClick={() => void loadTrace()}>
          Trace
        </Button>
      </Stack>

      {traceResult ? (
        <Box sx={{ p: 2, borderRadius: 2, border: '1px solid', borderColor: 'divider' }}>
          <Typography variant="subtitle2">Trace Summary</Typography>
          <Typography variant="body2" color="text.secondary">
            {traceResult.correlationId}
          </Typography>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} sx={{ mt: 1 }}>
            <Typography variant="caption">Requests: {traceResult.summary.requests.map((item) => item.uid).join(', ') || '-'}</Typography>
            <Typography variant="caption">Deliveries: {traceResult.summary.deliveries.map((item) => item.uid).join(', ') || '-'}</Typography>
            <Typography variant="caption">Jobs: {traceResult.summary.jobs.map((item) => item.uid).join(', ') || '-'}</Typography>
          </Stack>
        </Box>
      ) : null}

      <Box>
        <Typography variant="h6" gutterBottom>
          Event History
        </Typography>
        <AppDataGrid
          columns={eventColumns}
          fetchData={fetchEvents}
          storageKey="observability-events-grid"
          externalQueryKey={[eventType, level, correlationId, from, to].join('|')}
          pinActionsToRight
        />
      </Box>

      <Box>
        <Typography variant="h6" gutterBottom>
          Worker Status
        </Typography>
        <AppDataGrid columns={workerColumns} fetchData={fetchWorkers} storageKey="workers-grid" />
      </Box>

      <Box>
        <Typography variant="h6" gutterBottom>
          Rate Limits
        </Typography>
        <AppDataGrid columns={rateLimitColumns} fetchData={fetchRateLimits} storageKey="rate-limits-grid" />
      </Box>

      <EventDetailDialog open={Boolean(selectedEvent)} event={selectedEvent} onClose={() => setSelectedEvent(null)} />
    </Stack>
  )
}
