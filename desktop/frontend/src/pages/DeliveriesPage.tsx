import React from 'react'
import { Alert, Box, Chip, MenuItem, Stack, TextField, Typography } from '@mui/material'
import type { GridColDef, GridRenderCellParams } from '@mui/x-data-grid'
import type { PaginatedResponse } from '../api/pagination'
import { useApiClient } from '../api/useApiClient'
import { useSessionPrincipal } from '../auth/hooks'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { buildAdminListRequestQuery, useAdminListSearch } from '../components/admin/listSearch'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { handleAppError } from '../errors/handleAppError'
import { notify } from '../notifications/facade'
import { DeliveryDetailPage, type DeliveryDetailRecord } from './DeliveryDetailPage'

interface DeliveryRow extends DeliveryDetailRecord {}

const statusOptions = ['', 'pending', 'running', 'succeeded', 'failed', 'retrying'] as const

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
    case 'pending':
    case 'retrying':
      return 'warning'
    case 'running':
      return 'info'
    case 'succeeded':
      return 'success'
    case 'failed':
      return 'error'
    default:
      return 'default'
  }
}

export function DeliveriesPage() {
  const apiClient = useApiClient()
  const principal = useSessionPrincipal()
  const canWrite = Boolean(principal?.permissions.includes('deliveries.write'))

  const [reloadToken, setReloadToken] = React.useState(0)
  const { searchInput, setSearchInput, search } = useAdminListSearch()
  const [statusFilter, setStatusFilter] = React.useState('')
  const [serverFilter, setServerFilter] = React.useState('')
  const [dateFilter, setDateFilter] = React.useState('')
  const [detailOpen, setDetailOpen] = React.useState(false)
  const [detailDelivery, setDetailDelivery] = React.useState<DeliveryDetailRecord | null>(null)
  const [detailError, setDetailError] = React.useState('')
  const [retrying, setRetrying] = React.useState(false)

  const refreshGrid = () => setReloadToken((value) => value + 1)

  const fetchDeliveries = React.useCallback(
    async (params: AppDataGridFetchParams) => {
      const query = buildAdminListRequestQuery(params, {
        search,
        extra: {
          status: statusFilter,
          server: serverFilter,
          date: dateFilter,
        },
      })
      const response = await apiClient.request<PaginatedResponse<DeliveryRow>>(`/api/v1/deliveries?${query}`)
      return {
        rows: response.items,
        total: response.totalCount,
      }
    },
    [apiClient, dateFilter, search, serverFilter, statusFilter],
  )

  const openDetailDialog = async (row: DeliveryRow) => {
    setDetailError('')
    try {
      const detail = await apiClient.request<DeliveryDetailRecord>(`/api/v1/deliveries/${row.id}`)
      setDetailDelivery(detail)
      setDetailOpen(true)
    } catch (error) {
      setDetailDelivery(null)
      setDetailOpen(false)
      setDetailError('Unable to load delivery detail.')
      await handleAppError(error, { fallbackMessage: 'Unable to load delivery detail.' })
    }
  }

  const retryDelivery = async (deliveryId: number) => {
    setRetrying(true)
    try {
      const retried = await apiClient.request<DeliveryDetailRecord>(`/api/v1/deliveries/${deliveryId}/retry`, {
        method: 'POST',
      })
      setDetailDelivery(retried)
      setDetailOpen(true)
      refreshGrid()
      notify.success('Delivery retry scheduled.')
    } catch (error) {
      await handleAppError(error, { fallbackMessage: 'Unable to retry delivery.' })
    } finally {
      setRetrying(false)
    }
  }

  const columns = React.useMemo<GridColDef<DeliveryRow>[]>(
    () => [
      { field: 'uid', headerName: 'Delivery UID', minWidth: 220, flex: 1.1 },
      { field: 'requestUid', headerName: 'Request UID', minWidth: 220, flex: 1 },
      { field: 'serverName', headerName: 'Server', minWidth: 180, flex: 1 },
      {
        field: 'status',
        headerName: 'Status',
        minWidth: 140,
        renderCell: (params: GridRenderCellParams<DeliveryRow, string>) => (
          <Chip label={params.value ?? 'unknown'} size="small" color={statusColor(params.value ?? '')} />
        ),
      },
      { field: 'attemptNumber', headerName: 'Attempt', minWidth: 110, type: 'number' },
      {
        field: 'startedAt',
        headerName: 'Started',
        minWidth: 190,
        flex: 1,
        valueGetter: (value) => formatDate(String(value ?? '')),
      },
      {
        field: 'finishedAt',
        headerName: 'Finished',
        minWidth: 190,
        flex: 1,
        valueGetter: (value) => formatDate(String(value ?? '')),
      },
      {
        field: 'actions',
        headerName: 'Actions',
        minWidth: 96,
        sortable: false,
        filterable: false,
        renderCell: (params: GridRenderCellParams<DeliveryRow>) => (
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
                id: 'retry',
                label: 'Retry',
                onClick: () => {
                  void retryDelivery(params.row.id)
                },
                visible: canWrite && params.row.status === 'failed',
              },
            ]}
          />
        ),
      },
    ],
    [canWrite],
  )

  return (
    <Stack spacing={3}>
      <Box>
        <Typography variant="h4" component="h1">
          Deliveries
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Inspect delivery attempts, responses, and retry scheduling.
        </Typography>
      </Box>

      {detailError ? <Alert severity="error">{detailError}</Alert> : null}

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
        <TextField
          label="Search deliveries"
          placeholder="Search by delivery UID, request UID, server, or error"
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
        <TextField
          label="Server"
          placeholder="Filter by server"
          value={serverFilter}
          onChange={(event) => setServerFilter(event.target.value)}
          sx={{ minWidth: { md: 220 } }}
        />
        <TextField
          label="Date"
          type="date"
          value={dateFilter}
          onChange={(event) => setDateFilter(event.target.value)}
          InputLabelProps={{ shrink: true }}
          sx={{ minWidth: { md: 180 } }}
        />
      </Stack>

      <AppDataGrid
        columns={columns}
        fetchData={fetchDeliveries}
        storageKey="deliveries-grid"
        reloadToken={reloadToken}
        externalQueryKey={[search, statusFilter, serverFilter, dateFilter].join('|')}
        pinActionsToRight
      />

      <DeliveryDetailPage
        open={detailOpen}
        delivery={detailDelivery}
        canRetry={canWrite && detailDelivery?.status === 'failed'}
        retrying={retrying}
        onRetry={() => {
          if (!detailDelivery) {
            return
          }
          void retryDelivery(detailDelivery.id)
        }}
        onClose={() => {
          setDetailOpen(false)
          setDetailDelivery(null)
        }}
      />
    </Stack>
  )
}
