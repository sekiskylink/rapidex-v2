import React from 'react'
import {
  Alert,
  Box,
  Button,
  Chip,
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
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { apiRequest } from '../lib/api'

interface OrgUnit {
  id: number
  uid: string
  code: string
  name: string
  shortName: string
  description: string
  parentId?: number | null
  hierarchyLevel: number
  path: string
  address: string
  email: string
  url: string
  phoneNumber: string
  extras: Record<string, unknown>
  attributeValues: Record<string, unknown>
  openingDate?: string | null
  deleted: boolean
  lastSyncDate?: string | null
}

interface OrgUnitListResponse {
  items: OrgUnit[]
  totalCount: number
}

type OrgUnitFormState = {
  code: string
  name: string
  shortName: string
  description: string
  parentId: string
  address: string
  email: string
  url: string
  phoneNumber: string
  openingDate: string
  extrasText: string
  attributeValuesText: string
}

const emptyForm: OrgUnitFormState = {
  code: '',
  name: '',
  shortName: '',
  description: '',
  parentId: '',
  address: '',
  email: '',
  url: '',
  phoneNumber: '',
  openingDate: '',
  extrasText: '{}',
  attributeValuesText: '{}',
}

const dataGridSx = {
  '& .MuiDataGrid-columnHeaderTitle': {
    fontWeight: 700,
  },
}

function parseJSONField(value: string, label: string) {
  const trimmed = value.trim()
  if (trimmed === '') {
    return {}
  }
  const parsed = JSON.parse(trimmed)
  if (parsed === null || Array.isArray(parsed) || typeof parsed !== 'object') {
    throw new Error(`${label} must be a JSON object.`)
  }
  return parsed
}

function toDateInput(value?: string | null) {
  if (!value) {
    return ''
  }
  return value.slice(0, 10)
}

function toForm(unit?: OrgUnit | null): OrgUnitFormState {
  if (!unit) {
    return emptyForm
  }
  return {
    code: unit.code ?? '',
    name: unit.name ?? '',
    shortName: unit.shortName ?? '',
    description: unit.description ?? '',
    parentId: unit.parentId ? String(unit.parentId) : '',
    address: unit.address ?? '',
    email: unit.email ?? '',
    url: unit.url ?? '',
    phoneNumber: unit.phoneNumber ?? '',
    openingDate: toDateInput(unit.openingDate),
    extrasText: JSON.stringify(unit.extras ?? {}, null, 2),
    attributeValuesText: JSON.stringify(unit.attributeValues ?? {}, null, 2),
  }
}

export function OrgUnitsPage() {
  const [items, setItems] = React.useState<OrgUnit[]>([])
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState('')
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [viewing, setViewing] = React.useState<OrgUnit | null>(null)
  const [editing, setEditing] = React.useState<OrgUnit | null>(null)
  const [form, setForm] = React.useState<OrgUnitFormState>(emptyForm)
  const [submitting, setSubmitting] = React.useState(false)

  const load = React.useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const response = await apiRequest<OrgUnitListResponse>('/orgunits?page=0&pageSize=200')
      setItems(response.items ?? [])
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Unable to load facilities.')
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    void load()
  }, [load])

  const columns = React.useMemo<GridColDef<OrgUnit>[]>(
    () => [
      { field: 'name', headerName: 'Facility', flex: 1.1, minWidth: 180 },
      { field: 'shortName', headerName: 'Short Name', width: 160 },
      { field: 'code', headerName: 'Code', width: 150 },
      { field: 'uid', headerName: 'UID', width: 140 },
      { field: 'hierarchyLevel', headerName: 'Level', width: 90 },
      {
        field: 'parentId',
        headerName: 'Parent',
        width: 180,
        valueGetter: (_value, row) => items.find((item) => item.id === row.parentId)?.name ?? '',
      },
      { field: 'phoneNumber', headerName: 'Phone', width: 150 },
      { field: 'path', headerName: 'UID Path', flex: 1.1, minWidth: 220 },
      {
        field: 'actions',
        headerName: 'Actions',
        width: 110,
        sortable: false,
        filterable: false,
        renderCell: ({ row }) => (
          <AdminRowActions
            rowLabel={row.name}
            actions={[
              { id: 'view', label: 'View details', icon: 'view', onClick: () => setViewing(row) },
              { id: 'edit', label: 'Edit', icon: 'edit', onClick: () => openDialog(row) },
              {
                id: 'delete',
                label: 'Delete',
                icon: 'delete',
                destructive: true,
                confirmTitle: 'Delete facility',
                confirmMessage: `Delete ${row.name}? This cannot be undone.`,
                onClick: () => void deleteOrgUnit(row),
              },
            ]}
          />
        ),
      },
    ],
    [items],
  )

  function openDialog(unit?: OrgUnit) {
    setEditing(unit ?? null)
    setForm(toForm(unit ?? null))
    setDialogOpen(true)
    setError('')
  }

  function closeDialog() {
    if (submitting) {
      return
    }
    setDialogOpen(false)
    setEditing(null)
    setForm(emptyForm)
  }

  async function submitOrgUnit() {
    setSubmitting(true)
    setError('')
    try {
      const payload = {
        uid: editing?.uid ?? '',
        code: form.code.trim(),
        name: form.name.trim(),
        shortName: form.shortName.trim(),
        description: form.description.trim(),
        parentId: form.parentId ? Number(form.parentId) : null,
        address: form.address.trim(),
        email: form.email.trim(),
        url: form.url.trim(),
        phoneNumber: form.phoneNumber.trim(),
        openingDate: form.openingDate ? new Date(`${form.openingDate}T00:00:00Z`).toISOString() : null,
        extras: parseJSONField(form.extrasText, 'Extras'),
        attributeValues: parseJSONField(form.attributeValuesText, 'Attribute values'),
        deleted: editing?.deleted ?? false,
        lastSyncDate: editing?.lastSyncDate ?? null,
      }
      await apiRequest<OrgUnit>(editing ? `/orgunits/${editing.id}` : '/orgunits', {
        method: editing ? 'PUT' : 'POST',
        body: JSON.stringify(payload),
      })
      closeDialog()
      await load()
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : 'Unable to save facility.')
    } finally {
      setSubmitting(false)
    }
  }

  async function deleteOrgUnit(unit: OrgUnit) {
    setError('')
    try {
      await apiRequest<void>(`/orgunits/${unit.id}`, { method: 'DELETE' })
      await load()
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : 'Unable to delete facility.')
    }
  }

  const parentOptions = React.useMemo(
    () => items.filter((item) => item.id !== editing?.id),
    [editing?.id, items],
  )

  return (
    <Box>
      <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" spacing={2} sx={{ mb: 2 }}>
        <Box>
          <Typography variant="h4" component="h1">
            Facilities
          </Typography>
          <Typography color="text.secondary">Rapidex organisation units and facility hierarchy.</Typography>
        </Box>
        <Button variant="contained" onClick={() => openDialog()}>
          New Facility
        </Button>
      </Stack>

      {error ? <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert> : null}

      <DataGrid
        autoHeight
        rows={items}
        columns={columns}
        loading={loading}
        disableRowSelectionOnClick
        initialState={{ pagination: { paginationModel: { page: 0, pageSize: 25 } } }}
        pageSizeOptions={[25, 50, 100]}
        showToolbar
        slotProps={{
          toolbar: {
            csvOptions: {
              fileName: 'facilities-grid',
            },
          },
        }}
        sx={dataGridSx}
      />

      <Dialog open={Boolean(viewing)} onClose={() => setViewing(null)} fullWidth maxWidth="md">
        <DialogTitle>Facility Details</DialogTitle>
        <DialogContent>
          <Box
            sx={{
              pt: 1,
              display: 'grid',
              gap: 2,
              gridTemplateColumns: { xs: '1fr', md: 'repeat(2, minmax(0, 1fr))' },
            }}
          >
            <TextField label="Name" value={viewing?.name ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="Short Name" value={viewing?.shortName ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="Code" value={viewing?.code ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="UID" value={viewing?.uid ?? ''} InputProps={{ readOnly: true }} />
            <TextField
              label="Parent"
              value={viewing?.parentId ? items.find((item) => item.id === viewing.parentId)?.name ?? String(viewing.parentId) : 'None'}
              InputProps={{ readOnly: true }}
            />
            <TextField label="Hierarchy Level" value={viewing?.hierarchyLevel ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="Phone Number" value={viewing?.phoneNumber ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="Email" value={viewing?.email ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="Website URL" value={viewing?.url ?? ''} InputProps={{ readOnly: true }} />
            <Box sx={{ display: 'flex', alignItems: 'center' }}>
              <Chip label={viewing?.deleted ? 'Deleted' : 'Active'} color={viewing?.deleted ? 'warning' : 'success'} size="small" />
            </Box>
            <TextField label="Address" value={viewing?.address ?? ''} InputProps={{ readOnly: true }} sx={{ gridColumn: '1 / -1' }} />
            <TextField label="Description" value={viewing?.description ?? ''} InputProps={{ readOnly: true }} multiline minRows={2} sx={{ gridColumn: '1 / -1' }} />
            <TextField label="UID Path" value={viewing?.path ?? ''} InputProps={{ readOnly: true }} sx={{ gridColumn: '1 / -1' }} />
            <TextField label="Opening Date" value={toDateInput(viewing?.openingDate)} InputProps={{ readOnly: true }} />
            <TextField label="Last Sync Date" value={viewing?.lastSyncDate ? new Date(viewing.lastSyncDate).toLocaleString() : ''} InputProps={{ readOnly: true }} />
            <TextField
              label="Extras JSON"
              value={JSON.stringify(viewing?.extras ?? {}, null, 2)}
              InputProps={{ readOnly: true }}
              multiline
              minRows={4}
            />
            <TextField
              label="Attribute Values JSON"
              value={JSON.stringify(viewing?.attributeValues ?? {}, null, 2)}
              InputProps={{ readOnly: true }}
              multiline
              minRows={4}
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setViewing(null)}>Close</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={dialogOpen} onClose={closeDialog} fullWidth maxWidth="lg">
        <DialogTitle>{editing ? 'Edit Facility' : 'New Facility'}</DialogTitle>
        <DialogContent>
          <Box
            sx={{
              pt: 1,
              display: 'grid',
              gap: 2,
              gridTemplateColumns: { xs: '1fr', md: 'repeat(2, minmax(0, 1fr))' },
            }}
          >
            <TextField label="Code" value={form.code} onChange={(event) => setForm({ ...form, code: event.target.value })} required />
            <TextField label="Name" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required />
            <TextField label="Short Name" value={form.shortName} onChange={(event) => setForm({ ...form, shortName: event.target.value })} />
            <TextField select label="Parent" value={form.parentId} onChange={(event) => setForm({ ...form, parentId: event.target.value })}>
              <MenuItem value="">No parent</MenuItem>
              {parentOptions.map((item) => (
                <MenuItem key={item.id} value={String(item.id)}>
                  {item.name}
                </MenuItem>
              ))}
            </TextField>
            <TextField label="Phone Number" value={form.phoneNumber} onChange={(event) => setForm({ ...form, phoneNumber: event.target.value })} />
            <TextField label="Email" value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} />
            <TextField label="Website URL" value={form.url} onChange={(event) => setForm({ ...form, url: event.target.value })} />
            <TextField
              label="Opening Date"
              type="date"
              value={form.openingDate}
              onChange={(event) => setForm({ ...form, openingDate: event.target.value })}
              InputLabelProps={{ shrink: true }}
            />
            <TextField
              label="Address"
              value={form.address}
              onChange={(event) => setForm({ ...form, address: event.target.value })}
              sx={{ gridColumn: '1 / -1' }}
            />
            <TextField
              label="Description"
              value={form.description}
              onChange={(event) => setForm({ ...form, description: event.target.value })}
              multiline
              minRows={2}
              sx={{ gridColumn: '1 / -1' }}
            />
            <TextField label="UID" value={editing?.uid ?? 'Generated on save'} InputProps={{ readOnly: true }} />
            <TextField label="Hierarchy Level" value={editing?.hierarchyLevel ?? 'Derived on save'} InputProps={{ readOnly: true }} />
            <TextField label="UID Path" value={editing?.path ?? 'Derived on save'} InputProps={{ readOnly: true }} sx={{ gridColumn: '1 / -1' }} />
            <TextField
              label="Extras JSON"
              value={form.extrasText}
              onChange={(event) => setForm({ ...form, extrasText: event.target.value })}
              multiline
              minRows={4}
              sx={{ gridColumn: { xs: '1 / -1', md: 'span 1' } }}
            />
            <TextField
              label="Attribute Values JSON"
              value={form.attributeValuesText}
              onChange={(event) => setForm({ ...form, attributeValuesText: event.target.value })}
              multiline
              minRows={4}
              sx={{ gridColumn: { xs: '1 / -1', md: 'span 1' } }}
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeDialog} disabled={submitting}>Cancel</Button>
          <Button onClick={() => void submitOrgUnit()} disabled={submitting || !form.code.trim() || !form.name.trim()} variant="contained">
            {editing ? 'Save' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
