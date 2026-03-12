import React from 'react'
import { Box, Chip, Stack, Typography } from '@mui/material'
import type { GridColDef, GridRenderCellParams } from '@mui/x-data-grid'
import type { PaginatedResponse } from '../api/pagination'
import { useApiClient } from '../api/useApiClient'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'

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

  const fetchWorkers = React.useCallback(
    async (params: AppDataGridFetchParams) => {
      const query = new URLSearchParams({
        page: String(params.page + 1),
        pageSize: String(params.pageSize),
      })
      const response = await apiClient.request<PaginatedResponse<WorkerRow>>(`/api/v1/observability/workers?${query.toString()}`)
      return {
        rows: response.items,
        total: response.totalCount,
      }
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
      return {
        rows: response.items,
        total: response.totalCount,
      }
    },
    [apiClient],
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
      {
        field: 'startedAt',
        headerName: 'Started',
        minWidth: 190,
        flex: 1,
        valueGetter: (value) => formatDate(String(value ?? '')),
      },
      {
        field: 'lastHeartbeatAt',
        headerName: 'Last Heartbeat',
        minWidth: 190,
        flex: 1,
        valueGetter: (value) => formatDate(String(value ?? '')),
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
          Inspect worker activity and active rate-limit policies.
        </Typography>
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
    </Stack>
  )
}
