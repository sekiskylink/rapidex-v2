import React from 'react'
import {
  Box,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import type { GridColDef } from '@mui/x-data-grid'
import { ApiError } from '../api/client'
import { buildServerQuery, type PaginatedResponse } from '../api/pagination'
import { useApiClient } from '../api/useApiClient'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { JsonMetadataDialog } from '../components/admin/JsonMetadataDialog'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { notify } from '../notifications/store'

interface AuditRow {
  id: number
  timestamp: string
  actorUserId?: number
  action: string
  entityType?: string
  entityId?: string
  metadata?: unknown
}

const ACTION_FILTER_OPTIONS = [
  '',
  'users.create',
  'users.update',
  'users.reset_password',
  'users.set_active',
  'auth.login.success',
  'auth.login.failure',
]

function compactMetadata(metadata: unknown) {
  if (metadata == null) {
    return 'No metadata'
  }
  if (typeof metadata === 'string') {
    return metadata.length > 72 ? `${metadata.slice(0, 72)}...` : metadata
  }
  try {
    const value = JSON.stringify(metadata)
    return value.length > 72 ? `${value.slice(0, 72)}...` : value
  } catch {
    return String(metadata)
  }
}

export function AuditPage() {
  const apiClient = useApiClient()

  const [action, setAction] = React.useState('')
  const [actorUserId, setActorUserId] = React.useState('')
  const [dateFrom, setDateFrom] = React.useState('')
  const [dateTo, setDateTo] = React.useState('')
  const [reloadToken, setReloadToken] = React.useState(0)
  const [metadataDialogOpen, setMetadataDialogOpen] = React.useState(false)
  const [selectedMetadata, setSelectedMetadata] = React.useState<unknown>(null)

  const columns = React.useMemo<GridColDef<AuditRow>[]>(
    () => [
      {
        field: 'timestamp',
        headerName: 'Timestamp',
        width: 210,
        valueGetter: (_value, row) => new Date(row.timestamp).toLocaleString(),
      },
      { field: 'actorUserId', headerName: 'Actor', width: 110 },
      { field: 'action', headerName: 'Action', flex: 1, minWidth: 200 },
      { field: 'entityType', headerName: 'Entity Type', width: 140 },
      { field: 'entityId', headerName: 'Entity ID', width: 120 },
      {
        field: 'metadata',
        headerName: 'Metadata',
        flex: 1,
        minWidth: 260,
        sortable: false,
        valueGetter: (_value, row) => compactMetadata(row.metadata),
      },
      {
        field: 'actions',
        headerName: 'Actions',
        sortable: false,
        filterable: false,
        width: 96,
        renderCell: (params) => (
          <AdminRowActions
            rowLabel={params.row.action}
            actions={[
              {
                id: 'view-metadata',
                label: 'View Metadata',
                icon: 'view',
                onClick: () => {
                  setSelectedMetadata(params.row.metadata)
                  setMetadataDialogOpen(true)
                },
              },
            ]}
          />
        ),
      },
    ],
    [],
  )

  const fetchAudit = React.useCallback(
    async (params: AppDataGridFetchParams) => {
      const query = new URLSearchParams(buildServerQuery(params))
      if (action.trim()) {
        query.set('action', action.trim())
      }
      if (actorUserId.trim()) {
        query.set('actor_user_id', actorUserId.trim())
      }
      if (dateFrom.trim()) {
        query.set('date_from', `${dateFrom.trim()}T00:00:00Z`)
      }
      if (dateTo.trim()) {
        query.set('date_to', `${dateTo.trim()}T23:59:59Z`)
      }

      try {
        const payload = await apiClient.request<PaginatedResponse<AuditRow>>(`/api/v1/audit?${query.toString()}`)
        return {
          rows: payload.items,
          total: payload.totalCount,
        }
      } catch (error) {
        const message = error instanceof ApiError ? error.message : 'Unable to load audit logs.'
        notify({ severity: 'error', message })
        throw error
      }
    },
    [action, actorUserId, apiClient, dateFrom, dateTo],
  )

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box>
        <Typography variant="h5" component="h1" gutterBottom>
          Audit Log
        </Typography>
        <Typography color="text.secondary">View audit events with server-side filtering and pagination.</Typography>
      </Box>

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5}>
        <FormControl sx={{ minWidth: 220 }}>
          <InputLabel id="audit-action-label">Action</InputLabel>
          <Select
            labelId="audit-action-label"
            value={action}
            label="Action"
            onChange={(event) => {
              setAction(event.target.value)
              setReloadToken((value) => value + 1)
            }}
          >
            <MenuItem value="">
              <em>All actions</em>
            </MenuItem>
            {ACTION_FILTER_OPTIONS.filter(Boolean).map((option) => (
              <MenuItem key={option} value={option}>
                {option}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <TextField
          label="Actor User ID"
          value={actorUserId}
          onChange={(event) => setActorUserId(event.target.value)}
          onBlur={() => setReloadToken((value) => value + 1)}
          sx={{ minWidth: 180 }}
        />

        <TextField
          label="Date From"
          type="date"
          value={dateFrom}
          onChange={(event) => {
            setDateFrom(event.target.value)
            setReloadToken((value) => value + 1)
          }}
          InputLabelProps={{ shrink: true }}
        />

        <TextField
          label="Date To"
          type="date"
          value={dateTo}
          onChange={(event) => {
            setDateTo(event.target.value)
            setReloadToken((value) => value + 1)
          }}
          InputLabelProps={{ shrink: true }}
        />
      </Stack>

      <Box sx={{ height: 620, width: '100%', minWidth: 0, overflow: 'hidden' }}>
        <AppDataGrid
          columns={columns}
          fetchData={fetchAudit}
          storageKey="audit-table"
          reloadToken={reloadToken}
          stickyRightFields={['actions']}
        />
      </Box>
      <JsonMetadataDialog
        open={metadataDialogOpen}
        metadata={selectedMetadata}
        onClose={() => setMetadataDialogOpen(false)}
        onCopied={() => notify({ severity: 'success', message: 'Metadata copied.' })}
      />
    </Box>
  )
}
