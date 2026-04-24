import React from 'react'
import AddCircleRoundedIcon from '@mui/icons-material/AddCircleRounded'
import { Alert, Box, Button, Chip, MenuItem, Stack, TextField, Typography } from '@mui/material'
import type { GridColDef, GridRenderCellParams } from '@mui/x-data-grid'
import { useNavigate, useSearch } from '@tanstack/react-router'
import { useApiClient } from '../api/useApiClient'
import type { PaginatedResponse } from '../api/pagination'
import { useSessionPrincipal } from '../auth/hooks'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { buildAdminListRequestQuery, useAdminListSearch } from '../components/admin/listSearch'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { handleAppError } from '../errors/handleAppError'
import type { SchedulerRouteSearch } from './listRouteSearch'

interface ScheduledJobRecord {
  id: number
  uid: string
  code: string
  name: string
  description: string
  jobCategory: string
  jobType: string
  scheduleType: string
  scheduleExpr: string
  timezone: string
  enabled: boolean
  allowConcurrentRuns: boolean
  config: Record<string, unknown>
  lastRunAt?: string | null
  nextRunAt?: string | null
  lastSuccessAt?: string | null
  lastFailureAt?: string | null
  latestRunStatus?: string | null
  createdAt: string
  updatedAt: string
}

const categoryOptions = ['', 'integration', 'maintenance'] as const

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

function renderStatusChip(value?: string | null) {
  const status = (value ?? '').trim().toLowerCase()
  const color =
    status === 'succeeded'
      ? 'success'
      : status === 'failed' || status === 'cancelled'
        ? 'error'
        : status === 'running'
          ? 'warning'
          : status === 'pending'
            ? 'info'
            : 'default'
  return <Chip label={value || 'No runs'} size="small" color={color} />
}

export function SchedulerJobsPage() {
  const apiClient = useApiClient()
  const principal = useSessionPrincipal()
  const navigate = useNavigate()
  const routeSearch = useSearch({ strict: false }) as SchedulerRouteSearch
  const canWrite = Boolean(principal?.permissions.includes('scheduler.write'))
  const [reloadToken, setReloadToken] = React.useState(0)
  const [errorMessage, setErrorMessage] = React.useState('')
  const { searchInput, setSearchInput, search } = useAdminListSearch(routeSearch.q ?? '')
  const [categoryFilter, setCategoryFilter] = React.useState(routeSearch.category ?? '')

  React.useEffect(() => {
    setSearchInput(routeSearch.q ?? '')
  }, [routeSearch.q, setSearchInput])

  React.useEffect(() => {
    setCategoryFilter(routeSearch.category ?? '')
  }, [routeSearch.category])

  React.useEffect(() => {
    const nextSearch: SchedulerRouteSearch = {
      q: search || undefined,
      category: categoryFilter || undefined,
    }
    if ((routeSearch.q ?? '') === (nextSearch.q ?? '') && (routeSearch.category ?? '') === (nextSearch.category ?? '')) {
      return
    }
    void navigate({ to: '/scheduler', search: nextSearch, replace: true })
  }, [categoryFilter, navigate, routeSearch, search])

  const fetchJobs = React.useCallback(
    async (params: AppDataGridFetchParams) => {
      const query = buildAdminListRequestQuery(params, {
        search,
        extra: { category: categoryFilter },
      })
      const response = await apiClient.request<PaginatedResponse<ScheduledJobRecord>>(`/api/v1/scheduler/jobs?${query}`)
      return {
        rows: response.items ?? [],
        total: response.totalCount,
      }
    },
    [apiClient, categoryFilter, search],
  )

  const performMutation = async (path: string, failureMessage: string) => {
    setErrorMessage('')
    try {
      await apiClient.request(path, { method: 'POST' })
      setReloadToken((value) => value + 1)
    } catch (error) {
      setErrorMessage(failureMessage)
      await handleAppError(error, { fallbackMessage: failureMessage })
    }
  }

  const columns = React.useMemo<GridColDef<ScheduledJobRecord>[]>(
    () => [
      { field: 'code', headerName: 'Code', minWidth: 170, flex: 0.9 },
      { field: 'name', headerName: 'Name', minWidth: 220, flex: 1.2 },
      {
        field: 'jobCategory',
        headerName: 'Category',
        minWidth: 140,
        renderCell: (params: GridRenderCellParams<ScheduledJobRecord, string>) => (
          <Chip label={params.value ?? '-'} size="small" color={params.value === 'maintenance' ? 'warning' : 'info'} />
        ),
      },
      { field: 'jobType', headerName: 'Job Type', minWidth: 180, flex: 1 },
      { field: 'scheduleType', headerName: 'Schedule', minWidth: 120 },
      { field: 'scheduleExpr', headerName: 'Expression', minWidth: 160, flex: 0.9 },
      { field: 'timezone', headerName: 'Timezone', minWidth: 140 },
      {
        field: 'enabled',
        headerName: 'Enabled',
        minWidth: 110,
        renderCell: (params: GridRenderCellParams<ScheduledJobRecord, boolean>) => (
          <Chip label={params.value ? 'Enabled' : 'Disabled'} size="small" color={params.value ? 'success' : 'default'} />
        ),
      },
      {
        field: 'latestRunStatus',
        headerName: 'Latest Status',
        minWidth: 140,
        renderCell: (params: GridRenderCellParams<ScheduledJobRecord, string | null | undefined>) => renderStatusChip(params.value),
      },
      { field: 'nextRunAt', headerName: 'Next Run', minWidth: 180, flex: 1, valueGetter: (value) => formatDate(String(value ?? '')) },
      { field: 'lastRunAt', headerName: 'Last Run', minWidth: 180, flex: 1, valueGetter: (value) => formatDate(String(value ?? '')) },
      {
        field: 'actions',
        headerName: 'Actions',
        minWidth: 140,
        sortable: false,
        filterable: false,
        renderCell: (params: GridRenderCellParams<ScheduledJobRecord>) => (
          <AdminRowActions
            rowLabel={params.row.code}
            actions={[
              {
                id: 'edit',
                label: 'Edit',
                icon: 'edit',
                onClick: () => {
                  void navigate({ to: '/scheduler/$jobId', params: { jobId: String(params.row.id) } })
                },
              },
              {
                id: 'runs',
                label: 'Runs',
                icon: 'view',
                onClick: () => {
                  void navigate({ to: '/scheduler/$jobId/runs', params: { jobId: String(params.row.id) } })
                },
              },
              ...(canWrite
                ? [
                    {
                      id: params.row.enabled ? 'disable' : 'enable',
                      label: params.row.enabled ? 'Disable' : 'Enable',
                      icon: 'edit' as const,
                      onClick: () => {
                        void performMutation(
                          `/api/v1/scheduler/jobs/${params.row.id}/${params.row.enabled ? 'disable' : 'enable'}`,
                          `Unable to ${params.row.enabled ? 'disable' : 'enable'} scheduled job.`,
                        )
                      },
                    },
                    {
                      id: 'run-now',
                      label: 'Run Now',
                      icon: 'view' as const,
                      onClick: () => {
                        void performMutation(`/api/v1/scheduler/jobs/${params.row.id}/run-now`, 'Unable to queue scheduled job run.')
                      },
                    },
                  ]
                : []),
            ]}
          />
        ),
      },
    ],
    [canWrite, navigate],
  )

  return (
    <Stack spacing={3}>
      <Box display="flex" justifyContent="space-between" gap={2} alignItems={{ xs: 'stretch', md: 'center' }} flexDirection={{ xs: 'column', md: 'row' }}>
        <Box>
          <Typography variant="h4" component="h1">
            Scheduler
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Manage scheduled integration and maintenance jobs.
          </Typography>
        </Box>
        {canWrite ? (
          <Button variant="contained" startIcon={<AddCircleRoundedIcon />} onClick={() => void navigate({ to: '/scheduler/new' })}>
            Create Scheduled Job
          </Button>
        ) : null}
      </Box>

      {errorMessage ? <Alert severity="error">{errorMessage}</Alert> : null}

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
        <TextField
          label="Search scheduled jobs"
          placeholder="Search by code, name, description, or job type"
          value={searchInput}
          onChange={(event) => setSearchInput(event.target.value)}
          fullWidth
        />
        <TextField
          select
          label="Category"
          value={categoryFilter}
          onChange={(event) => setCategoryFilter(event.target.value)}
          sx={{ minWidth: { md: 180 } }}
        >
          {categoryOptions.map((option) => (
            <MenuItem key={option || 'all'} value={option}>
              {option ? option[0].toUpperCase() + option.slice(1) : 'All categories'}
            </MenuItem>
          ))}
        </TextField>
      </Stack>

      <AppDataGrid
        columns={columns}
        fetchData={fetchJobs}
        storageKey="scheduler-grid"
        reloadToken={reloadToken}
        externalQueryKey={[search, categoryFilter].join('|')}
        pinActionsToRight
      />
    </Stack>
  )
}
