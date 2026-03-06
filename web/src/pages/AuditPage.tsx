import React from 'react'
import { Box, Typography } from '@mui/material'
import type { GridColDef } from '@mui/x-data-grid'
import { JsonMetadataDialog } from '../components/admin/JsonMetadataDialog'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { AppDataGrid, type AppDataGridFetchParams } from '../components/datagrid/AppDataGrid'
import { apiRequest } from '../lib/api'
import { buildListQuery, type PaginatedResponse } from '../lib/pagination'
import { useSnackbar } from '../ui/snackbar'

interface AuditRow {
  id: number
  timestamp: string
  actorUserId?: number
  action: string
  entityType?: string
  entityId?: string
  metadata?: unknown
}

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
  const { showSnackbar } = useSnackbar()
  const [metadataDialogOpen, setMetadataDialogOpen] = React.useState(false)
  const [selectedMetadata, setSelectedMetadata] = React.useState<unknown>(null)

  const columns = React.useMemo<GridColDef<AuditRow>[]>(
    () => [
      { field: 'id', headerName: 'ID', width: 90 },
      {
        field: 'timestamp',
        headerName: 'Timestamp',
        minWidth: 220,
        valueGetter: (_value, row) => new Date(row.timestamp).toLocaleString(),
      },
      { field: 'actorUserId', headerName: 'Actor User', width: 120 },
      { field: 'action', headerName: 'Action', minWidth: 220, flex: 1 },
      { field: 'entityType', headerName: 'Entity Type', width: 140 },
      { field: 'entityId', headerName: 'Entity ID', width: 120 },
      {
        field: 'metadata',
        headerName: 'Metadata',
        minWidth: 260,
        flex: 1,
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

  const fetchAudit = React.useCallback(async (params: AppDataGridFetchParams) => {
    const query = buildListQuery(params)
    const response = await apiRequest<PaginatedResponse<AuditRow>>(`/audit?${query}`)
    return {
      rows: response.items,
      total: response.totalCount,
    }
  }, [])

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box>
        <Typography variant="h5" component="h1" gutterBottom>
          Audit Log
        </Typography>
        <Typography color="text.secondary">Server-side pagination, sorting, filtering, and metadata details for audit events.</Typography>
      </Box>
      <Box sx={{ height: 620, width: '100%', minWidth: 0, overflow: 'auto' }}>
        <AppDataGrid
          columns={columns}
          fetchData={fetchAudit}
          storageKey="audit-table"
          stickyRightFields={['actions']}
          enablePinnedColumns
        />
      </Box>
      <JsonMetadataDialog
        open={metadataDialogOpen}
        metadata={selectedMetadata}
        onClose={() => setMetadataDialogOpen(false)}
        onCopied={() => showSnackbar({ severity: 'success', message: 'Metadata copied.' })}
      />
    </Box>
  )
}
