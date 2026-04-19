import React from 'react'
import { Alert, Box, Button, Chip, Dialog, DialogContent, DialogTitle, MenuItem, Stack, TextField, Typography } from '@mui/material'
import type { GridColDef, GridRenderCellParams } from '@mui/x-data-grid'
import { useNavigate, useParams } from '@tanstack/react-router'
import type { PaginatedResponse } from '../api/pagination'
import { useApiClient } from '../api/useApiClient'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { buildAdminListRequestQuery } from '../components/admin/listSearch'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { handleAppError } from '../errors/handleAppError'

interface ScheduledJobRecord {
  id: number
  uid: string
  code: string
  name: string
}

interface RunRecord {
  id: number
  uid: string
  scheduledJobId: number
  scheduledJobUid: string
  scheduledJobCode: string
  scheduledJobName: string
  triggerMode: string
  scheduledFor: string
  startedAt?: string | null
  finishedAt?: string | null
  status: string
  workerId?: number | null
  errorMessage: string
  resultSummary: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

const statusOptions = ['', 'pending', 'running', 'succeeded', 'failed', 'cancelled', 'skipped'] as const

function formatDate(value?: string | null) {
  if (!value) {
    return '-'
  }
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return value
  }
  return parsed.toLocaleString()
}

function renderStatusChip(status: string) {
  const normalized = status.trim().toLowerCase()
  const color =
    normalized === 'succeeded'
      ? 'success'
      : normalized === 'failed' || normalized === 'cancelled'
        ? 'error'
        : normalized === 'running'
          ? 'warning'
          : normalized === 'pending'
            ? 'info'
            : 'default'
  return <Chip label={status || 'Unknown'} size="small" color={color} />
}

function getSummaryMetric(summary: Record<string, unknown>, key: string) {
  const value = summary[key]
  if (typeof value === 'number') {
    return value
  }
  return 0
}

export function SchedulerJobRunsPage() {
  const apiClient = useApiClient()
  const navigate = useNavigate()
  const params = useParams({ strict: false }) as { jobId?: string }
  const jobId = Number(params.jobId)
  const [job, setJob] = React.useState<ScheduledJobRecord | null>(null)
  const [statusFilter, setStatusFilter] = React.useState('')
  const [errorMessage, setErrorMessage] = React.useState('')
  const [selectedRun, setSelectedRun] = React.useState<RunRecord | null>(null)

  React.useEffect(() => {
    if (!Number.isFinite(jobId)) {
      return
    }
    let active = true
    apiClient
      .request<ScheduledJobRecord>(`/api/v1/scheduler/jobs/${jobId}`)
      .then((response) => {
        if (active) {
          setJob(response)
        }
      })
      .catch(async (error) => {
        if (!active) {
          return
        }
        setErrorMessage('Unable to load scheduled job.')
        await handleAppError(error, { fallbackMessage: 'Unable to load scheduled job.' })
      })
    return () => {
      active = false
    }
  }, [apiClient, jobId])

  const fetchRuns = React.useCallback(
    async (params: AppDataGridFetchParams) => {
      const query = buildAdminListRequestQuery(params, {
        extra: { status: statusFilter },
      })
      const response = await apiClient.request<PaginatedResponse<RunRecord>>(`/api/v1/scheduler/jobs/${jobId}/runs?${query}`)
      return {
        rows: response.items ?? [],
        total: response.totalCount,
      }
    },
    [apiClient, jobId, statusFilter],
  )

  const openRunDetail = async (runId: number) => {
    try {
      const response = await apiClient.request<RunRecord>(`/api/v1/scheduler/runs/${runId}`)
      setSelectedRun(response)
    } catch (error) {
      await handleAppError(error, { fallbackMessage: 'Unable to load run details.' })
    }
  }

  const queueRunNow = async () => {
    try {
      await apiClient.request(`/api/v1/scheduler/jobs/${jobId}/run-now`, { method: 'POST' })
      setErrorMessage('')
      setSelectedRun(null)
    } catch (error) {
      await handleAppError(error, { fallbackMessage: 'Unable to queue scheduled job run.' })
    }
  }

  const columns = React.useMemo<GridColDef<RunRecord>[]>(
    () => [
      { field: 'uid', headerName: 'Run UID', minWidth: 220, flex: 1.1 },
      { field: 'triggerMode', headerName: 'Trigger', minWidth: 120 },
      { field: 'status', headerName: 'Status', minWidth: 130, renderCell: (params: GridRenderCellParams<RunRecord, string>) => renderStatusChip(params.value ?? '') },
      { field: 'scheduledFor', headerName: 'Scheduled For', minWidth: 180, flex: 1, valueGetter: (value) => formatDate(String(value ?? '')) },
      { field: 'startedAt', headerName: 'Started', minWidth: 180, flex: 1, valueGetter: (value) => formatDate(String(value ?? '')) },
      { field: 'finishedAt', headerName: 'Finished', minWidth: 180, flex: 1, valueGetter: (value) => formatDate(String(value ?? '')) },
      {
        field: 'actions',
        headerName: 'Actions',
        minWidth: 96,
        sortable: false,
        filterable: false,
        renderCell: (params: GridRenderCellParams<RunRecord>) => (
          <AdminRowActions
            rowLabel={params.row.uid}
            actions={[
              {
                id: 'view',
                label: 'View',
                icon: 'view',
                onClick: () => {
                  void openRunDetail(params.row.id)
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
      <Box display="flex" justifyContent="space-between" gap={2} flexDirection={{ xs: 'column', md: 'row' }}>
        <Box>
          <Typography variant="h4" component="h1">
            Scheduler Runs
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {job ? `Run history for ${job.name} (${job.code}).` : 'Run history for the selected scheduled job.'}
          </Typography>
        </Box>
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5}>
          {Number.isFinite(jobId) ? (
            <Button variant="outlined" onClick={() => void navigate({ to: '/scheduler/$jobId', params: { jobId: String(jobId) } })}>
              Edit Job
            </Button>
          ) : null}
          {Number.isFinite(jobId) ? (
            <Button variant="outlined" onClick={() => void queueRunNow()}>
              Run Now
            </Button>
          ) : null}
          <Button variant="outlined" onClick={() => void navigate({ to: '/scheduler' })}>
            Back to Scheduler
          </Button>
        </Stack>
      </Box>

      {errorMessage ? <Alert severity="error">{errorMessage}</Alert> : null}

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
        <TextField
          select
          label="Status"
          value={statusFilter}
          onChange={(event) => setStatusFilter(event.target.value)}
          sx={{ minWidth: { md: 220 } }}
        >
          {statusOptions.map((option) => (
            <MenuItem key={option || 'all'} value={option}>
              {option ? option[0].toUpperCase() + option.slice(1) : 'All statuses'}
            </MenuItem>
          ))}
        </TextField>
      </Stack>

      <AppDataGrid columns={columns} fetchData={fetchRuns} storageKey="scheduler-runs-grid" externalQueryKey={statusFilter} pinActionsToRight />

      <Dialog open={Boolean(selectedRun)} onClose={() => setSelectedRun(null)} fullWidth maxWidth="md">
        <DialogTitle>Run Details</DialogTitle>
        <DialogContent>
          {selectedRun ? (
            <Stack spacing={1.5} sx={{ py: 1 }}>
              <Typography variant="body2">Run UID: {selectedRun.uid}</Typography>
              <Stack direction="row" spacing={1} alignItems="center">
                <Typography variant="body2">Status:</Typography>
                {renderStatusChip(selectedRun.status)}
                {selectedRun.resultSummary?.dryRun ? <Chip label="Dry Run" size="small" color="warning" variant="outlined" /> : null}
              </Stack>
              <Typography variant="body2">Scheduled For: {formatDate(selectedRun.scheduledFor)}</Typography>
              <Typography variant="body2">Started: {formatDate(selectedRun.startedAt)}</Typography>
              <Typography variant="body2">Finished: {formatDate(selectedRun.finishedAt)}</Typography>
              {selectedRun.errorMessage ? <Alert severity="error">{selectedRun.errorMessage}</Alert> : null}
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={1}>
                <Chip label={`Scanned ${getSummaryMetric(selectedRun.resultSummary, 'scanned_count')}`} size="small" />
                <Chip label={`Affected ${getSummaryMetric(selectedRun.resultSummary, 'affected_count')}`} size="small" />
                <Chip label={`Archived ${getSummaryMetric(selectedRun.resultSummary, 'archived_count')}`} size="small" />
                <Chip label={`Deleted ${getSummaryMetric(selectedRun.resultSummary, 'deleted_count')}`} size="small" />
                <Chip label={`Skipped ${getSummaryMetric(selectedRun.resultSummary, 'skipped_count')}`} size="small" />
              </Stack>
              <TextField label="Result Summary" value={JSON.stringify(selectedRun.resultSummary ?? {}, null, 2)} multiline minRows={8} fullWidth InputProps={{ readOnly: true }} />
            </Stack>
          ) : null}
        </DialogContent>
      </Dialog>
    </Stack>
  )
}
