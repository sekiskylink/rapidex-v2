import React from 'react'
import { Alert, Box, Chip, MenuItem, Stack, TextField, Typography } from '@mui/material'
import type { GridColDef, GridRenderCellParams } from '@mui/x-data-grid'
import { useNavigate, useSearch } from '@tanstack/react-router'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { buildAdminListRequestQuery, useAdminListSearch } from '../components/admin/listSearch'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { handleAppError } from '../errors/handleAppError'
import { apiRequest } from '../lib/api'
import type { PaginatedResponse } from '../lib/pagination'
import { useAppNotify } from '../notifications/facade'
import { JobDetailPage, type JobDetailRecord, type JobPollRecord } from './JobDetailPage'
import type { JobsRouteSearch } from './listRouteSearch'
import type { EventRecord } from './traceability'

interface JobRow extends JobDetailRecord {}

const statusOptions = ['', 'pending', 'polling', 'succeeded', 'failed'] as const

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

function stateColor(state: string): 'default' | 'warning' | 'success' | 'error' | 'info' {
  switch (state) {
    case 'pending':
      return 'warning'
    case 'polling':
      return 'info'
    case 'succeeded':
      return 'success'
    case 'failed':
      return 'error'
    default:
      return 'default'
  }
}

export function JobsPage() {
  const notify = useAppNotify()
  const navigate = useNavigate()
  const routeSearch = useSearch({ strict: false }) as JobsRouteSearch
  const [reloadToken, setReloadToken] = React.useState(0)
  const { searchInput, setSearchInput, search } = useAdminListSearch(
    routeSearch.q ?? '',
  )
  const [statusFilter, setStatusFilter] = React.useState(routeSearch.status ?? '')
  const [detailOpen, setDetailOpen] = React.useState(false)
  const [detailJob, setDetailJob] = React.useState<JobDetailRecord | null>(null)
  const [polls, setPolls] = React.useState<JobPollRecord[]>([])
  const [events, setEvents] = React.useState<EventRecord[]>([])
  const [detailError, setDetailError] = React.useState('')

  React.useEffect(() => {
    setSearchInput(routeSearch.q ?? '')
  }, [routeSearch.q, setSearchInput])

  React.useEffect(() => {
    setStatusFilter(routeSearch.status ?? '')
  }, [routeSearch.status])

  React.useEffect(() => {
    const nextSearch: JobsRouteSearch = {
      q: search || undefined,
      status: statusFilter || undefined,
    }
    if (
      (routeSearch.q ?? '') === (nextSearch.q ?? '') &&
      (routeSearch.status ?? '') === (nextSearch.status ?? '')
    ) {
      return
    }
    void navigate({ to: '/jobs', search: nextSearch, replace: true })
  }, [navigate, routeSearch, search, statusFilter])

  const fetchJobs = React.useCallback(
    async (params: AppDataGridFetchParams) => {
      const query = buildAdminListRequestQuery(params, {
        search,
        extra: {
          status: statusFilter,
        },
      })
      const response = await apiRequest<PaginatedResponse<JobRow>>(`/jobs?${query}`)
      return {
        rows: response.items ?? [],
        total: response.totalCount,
      }
    },
    [search, statusFilter],
  )

  const openDetailDialog = async (row: JobRow) => {
    setDetailError('')
    try {
      const [detail, pollResponse, eventResponse] = await Promise.all([
        apiRequest<JobDetailRecord>(`/jobs/${row.id}`),
        apiRequest<PaginatedResponse<JobPollRecord>>(`/jobs/${row.id}/polls?page=1&pageSize=50`),
        apiRequest<PaginatedResponse<EventRecord>>(`/jobs/${row.id}/events?page=1&pageSize=50&sort=createdAt:asc`),
      ])
      setDetailJob(detail)
      setPolls(pollResponse.items ?? [])
      setEvents(eventResponse.items ?? [])
      setDetailOpen(true)
    } catch (error) {
      setDetailJob(null)
      setPolls([])
      setEvents([])
      setDetailOpen(false)
      setDetailError('Unable to load job detail.')
      await handleAppError(error, {
        fallbackMessage: 'Unable to load job detail.',
        notifier: notify,
      })
    }
  }

  const columns = React.useMemo<GridColDef<JobRow>[]>(
    () => [
      { field: 'uid', headerName: 'Job UID', minWidth: 220, flex: 1.1 },
      { field: 'deliveryUid', headerName: 'Delivery UID', minWidth: 180, flex: 1 },
      { field: 'requestUid', headerName: 'Request UID', minWidth: 180, flex: 1 },
      { field: 'remoteJobId', headerName: 'Remote Job ID', minWidth: 180, flex: 1 },
      {
        field: 'remoteStatus',
        headerName: 'Remote Status',
        minWidth: 150,
      },
      {
        field: 'currentState',
        headerName: 'State',
        minWidth: 140,
        renderCell: (params: GridRenderCellParams<JobRow, string>) => (
          <Chip label={params.value ?? 'unknown'} size="small" color={stateColor(params.value ?? '')} />
        ),
      },
      {
        field: 'nextPollAt',
        headerName: 'Next Poll',
        minWidth: 180,
        flex: 1,
        valueGetter: (value) => formatDate(String(value ?? '')),
      },
      {
        field: 'completedAt',
        headerName: 'Completed',
        minWidth: 180,
        flex: 1,
        valueGetter: (value) => formatDate(String(value ?? '')),
      },
      {
        field: 'createdAt',
        headerName: 'Created',
        minWidth: 180,
        flex: 1,
        valueGetter: (value) => formatDate(String(value ?? '')),
      },
      {
        field: 'actions',
        headerName: 'Actions',
        minWidth: 96,
        sortable: false,
        filterable: false,
        renderCell: (params: GridRenderCellParams<JobRow>) => (
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
            ]}
          />
        ),
      },
    ],
    [],
  )

  return (
    <Stack spacing={3}>
      <Box>
        <Typography variant="h4" component="h1">
          Jobs
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Track async tasks, linked deliveries, and poll history.
        </Typography>
      </Box>

      {detailError ? <Alert severity="error">{detailError}</Alert> : null}

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
        <TextField
          label="Search jobs"
          placeholder="Search by job, delivery, request, or remote job ID"
          value={searchInput}
          onChange={(event) => setSearchInput(event.target.value)}
          fullWidth
        />
        <TextField
          select
          label="Status"
          value={statusFilter}
          onChange={(event) => setStatusFilter(event.target.value)}
          sx={{ minWidth: { md: 180 } }}
        >
          {statusOptions.map((option) => (
            <MenuItem key={option || 'all'} value={option}>
              {option ? option[0].toUpperCase() + option.slice(1) : 'All statuses'}
            </MenuItem>
          ))}
        </TextField>
      </Stack>

      <AppDataGrid
        columns={columns}
        fetchData={fetchJobs}
        storageKey="sukumad-jobs-grid"
        reloadToken={reloadToken}
        externalQueryKey={[search, statusFilter].join('|')}
        pinActionsToRight
      />

      <JobDetailPage
        open={detailOpen}
        job={detailJob}
        polls={polls}
        events={events}
        onClose={() => {
          setDetailOpen(false)
          setDetailJob(null)
          setPolls([])
          setEvents([])
          setReloadToken((value) => value + 1)
        }}
      />
    </Stack>
  )
}
