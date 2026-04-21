import React from 'react'
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  MenuItem,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import type { GridColDef } from '@mui/x-data-grid'
import { DataGrid } from '@mui/x-data-grid'
import { useApiClient } from '../api/useApiClient'

interface OrgUnit {
  id: number
  uid: string
  code: string
  name: string
  description: string
  parentId?: number | null
  path: string
}

interface OrgUnitListResponse {
  items: OrgUnit[]
  totalCount: number
}

const emptyForm = {
  code: '',
  name: '',
  description: '',
  parentId: '',
}

export function OrgUnitsPage() {
  const apiClient = useApiClient()
  const [items, setItems] = React.useState<OrgUnit[]>([])
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState('')
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [form, setForm] = React.useState(emptyForm)
  const [submitting, setSubmitting] = React.useState(false)

  const load = React.useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const response = await apiClient.request<OrgUnitListResponse>('/api/v1/orgunits?page=0&pageSize=200')
      setItems(response.items ?? [])
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Unable to load facilities.')
    } finally {
      setLoading(false)
    }
  }, [apiClient])

  React.useEffect(() => {
    void load()
  }, [load])

  const columns = React.useMemo<GridColDef<OrgUnit>[]>(
    () => [
      { field: 'name', headerName: 'Facility', flex: 1, minWidth: 180 },
      { field: 'code', headerName: 'Code', width: 160 },
      {
        field: 'parentId',
        headerName: 'Parent',
        width: 180,
        valueGetter: (_value, row) => items.find((item) => item.id === row.parentId)?.name ?? '',
      },
      { field: 'path', headerName: 'Path', flex: 1, minWidth: 180 },
    ],
    [items],
  )

  async function createOrgUnit() {
    setSubmitting(true)
    setError('')
    try {
      await apiClient.request<OrgUnit>('/api/v1/orgunits', {
        method: 'POST',
        body: JSON.stringify({
          code: form.code.trim(),
          name: form.name.trim(),
          description: form.description.trim(),
          parentId: form.parentId ? Number(form.parentId) : null,
        }),
      })
      setDialogOpen(false)
      setForm(emptyForm)
      await load()
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : 'Unable to create facility.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Box>
      <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" spacing={2} sx={{ mb: 2 }}>
        <Box>
          <Typography variant="h4" component="h1">
            Facilities
          </Typography>
          <Typography color="text.secondary">Rapidex organisation units and facility hierarchy.</Typography>
        </Box>
        <Button variant="contained" onClick={() => setDialogOpen(true)}>
          New Facility
        </Button>
      </Stack>

      {error ? <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert> : null}

      <DataGrid autoHeight rows={items} columns={columns} loading={loading} disableRowSelectionOnClick pageSizeOptions={[25, 50, 100]} />

      <Dialog open={dialogOpen} onClose={submitting ? undefined : () => setDialogOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>New Facility</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            <TextField label="Code" value={form.code} onChange={(event) => setForm({ ...form, code: event.target.value })} required />
            <TextField label="Name" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required />
            <TextField label="Description" value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} multiline minRows={2} />
            <TextField select label="Parent" value={form.parentId} onChange={(event) => setForm({ ...form, parentId: event.target.value })}>
              <MenuItem value="">No parent</MenuItem>
              {items.map((item) => (
                <MenuItem key={item.id} value={String(item.id)}>
                  {item.name}
                </MenuItem>
              ))}
            </TextField>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)} disabled={submitting}>Cancel</Button>
          <Button onClick={createOrgUnit} disabled={submitting || !form.code.trim() || !form.name.trim()} variant="contained">Create</Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
