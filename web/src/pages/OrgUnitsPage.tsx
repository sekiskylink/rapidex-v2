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
import { getAuthSnapshot } from '../auth/state'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { apiRequest } from '../lib/api'
import { AddCircleRoundedIcon } from '../ui/icons'

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

interface SyncState {
  lastStartedAt?: string | null
  lastCompletedAt?: string | null
  lastSyncedAt?: string | null
  lastStatus?: string | null
  lastError?: string | null
  sourceServerCode?: string | null
  districtLevelName?: string | null
}

interface SyncResult {
  status: string
  serverCode: string
  dryRun: boolean
  fullRefresh: boolean
  districtLevelName: string
  orgUnitsCount: number
  deletedReporters: number
  deletedAssignments: number
  orphanedReporters: number
  remappedReporters: number
  errorMessage?: string | null
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

const defaultSyncForm = {
  serverCode: 'dhis2',
  serverUid: '',
  districtLevelName: 'District',
  districtLevelCode: '',
  dryRun: true,
  fullRefresh: true,
  initialSync: false,
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
  const currentUser = getAuthSnapshot().user
  const [items, setItems] = React.useState<OrgUnit[]>([])
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState('')
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [syncDialogOpen, setSyncDialogOpen] = React.useState(false)
  const [viewing, setViewing] = React.useState<OrgUnit | null>(null)
  const [editing, setEditing] = React.useState<OrgUnit | null>(null)
  const [form, setForm] = React.useState<OrgUnitFormState>(emptyForm)
  const [submitting, setSubmitting] = React.useState(false)
  const [syncing, setSyncing] = React.useState(false)
  const [syncState, setSyncState] = React.useState<SyncState | null>(null)
  const [syncResult, setSyncResult] = React.useState<SyncResult | null>(null)
  const [syncForm, setSyncForm] = React.useState(defaultSyncForm)

  const load = React.useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const response = await apiRequest<OrgUnitListResponse>('/orgunits?page=0&pageSize=200')
      setItems(response.items ?? [])
      const state = await apiRequest<SyncState>('/orgunits/sync-state')
      setSyncState(state)
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

  function closeSyncDialog() {
    if (syncing) {
      return
    }
    setSyncDialogOpen(false)
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

  async function runSync() {
    setSyncing(true)
    setError('')
    setSyncResult(null)
    try {
      const response = await apiRequest<SyncResult>('/orgunits/sync', {
        method: 'POST',
        body: JSON.stringify({
          serverCode: syncForm.serverCode.trim(),
          serverUid: syncForm.serverUid.trim(),
          districtLevelName: syncForm.districtLevelName.trim(),
          districtLevelCode: syncForm.districtLevelCode.trim(),
          dryRun: syncForm.dryRun,
          fullRefresh: syncForm.fullRefresh,
          initialSync: syncForm.initialSync,
        }),
      })
      setSyncResult(response)
      await load()
      if (!response.dryRun) {
        setSyncDialogOpen(false)
      }
    } catch (syncError) {
      setError(syncError instanceof Error ? syncError.message : 'Unable to run DHIS2 hierarchy sync.')
    } finally {
      setSyncing(false)
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
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
          {currentUser?.permissions.includes('orgunits.write') ? (
            <Button variant="outlined" size="small" onClick={() => setSyncDialogOpen(true)}>
              Sync DHIS2 Hierarchy
            </Button>
          ) : null}
          <Button variant="contained" size="small" startIcon={<AddCircleRoundedIcon />} onClick={() => openDialog()}>
            New Facility
          </Button>
        </Stack>
      </Stack>
      {Boolean(currentUser?.isOrgUnitScopeRestricted && (currentUser.assignedOrgUnitIds?.length ?? 0) === 0) ? (
        <Alert severity="info" sx={{ mb: 2 }}>
          No org units are assigned to your account yet, so the facility hierarchy is currently empty.
        </Alert>
      ) : null}

      {error ? <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert> : null}
      {syncState?.lastStatus ? (
        <Alert severity={syncState.lastStatus === 'failed' ? 'error' : syncState.lastStatus === 'succeeded' ? 'success' : 'info'} sx={{ mb: 2 }}>
          Last hierarchy sync: {syncState.lastStatus}
          {syncState.sourceServerCode ? ` via ${syncState.sourceServerCode}` : ''}
          {syncState.lastCompletedAt ? ` at ${new Date(syncState.lastCompletedAt).toLocaleString()}` : ''}
          {syncState.districtLevelName ? ` using district level ${syncState.districtLevelName}` : ''}
          {syncState.lastError ? ` (${syncState.lastError})` : ''}
        </Alert>
      ) : null}
      {syncResult ? (
        <Alert severity={syncResult.status === 'failed' ? 'error' : syncResult.dryRun ? 'info' : 'success'} sx={{ mb: 2 }}>
          {syncResult.dryRun ? 'Dry run complete.' : 'Hierarchy refresh complete.'} Imported {syncResult.orgUnitsCount} org units.
          {!syncResult.dryRun
            ? syncForm.initialSync
              ? ` Cleared ${syncResult.deletedReporters} reporters and ${syncResult.deletedAssignments} user assignments.`
              : ` Remapped ${syncResult.remappedReporters} reporters, orphaned ${syncResult.orphanedReporters}, and rebuilt ${syncResult.deletedAssignments} user assignments.`
            : ''}
        </Alert>
      ) : null}

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

      <Dialog open={syncDialogOpen} onClose={closeSyncDialog} fullWidth maxWidth="sm">
        <DialogTitle>Sync DHIS2 Hierarchy</DialogTitle>
        <DialogContent dividers>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <Alert severity="warning">
              Initial sync deletes local reporters and user assignments. Reconcile mode preserves reporters, remaps them by DHIS2 UID, and orphans only unmatched reporters.
            </Alert>
            <TextField label="Server Code" value={syncForm.serverCode} onChange={(event) => setSyncForm((current) => ({ ...current, serverCode: event.target.value }))} />
            <TextField label="Server UID" value={syncForm.serverUid} onChange={(event) => setSyncForm((current) => ({ ...current, serverUid: event.target.value }))} helperText="Optional override when you prefer UID lookup." />
            <TextField label="District Level Name" value={syncForm.districtLevelName} onChange={(event) => setSyncForm((current) => ({ ...current, districtLevelName: event.target.value }))} />
            <TextField label="District Level Code" value={syncForm.districtLevelCode} onChange={(event) => setSyncForm((current) => ({ ...current, districtLevelCode: event.target.value }))} />
            <TextField
              select
              label="Refresh Mode"
              value={syncForm.initialSync ? 'initial' : 'reconcile'}
              onChange={(event) => setSyncForm((current) => ({ ...current, initialSync: event.target.value === 'initial' }))}
            >
              <MenuItem value="reconcile">Reconcile Existing Reporters</MenuItem>
              <MenuItem value="initial">Initial Sync (Destructive)</MenuItem>
            </TextField>
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
              <Button variant={syncForm.dryRun ? 'contained' : 'outlined'} onClick={() => setSyncForm((current) => ({ ...current, dryRun: !current.dryRun }))}>
                {syncForm.dryRun ? 'Dry Run Enabled' : 'Enable Dry Run'}
              </Button>
              <Button variant={syncForm.fullRefresh ? 'contained' : 'outlined'} color="warning" onClick={() => setSyncForm((current) => ({ ...current, fullRefresh: !current.fullRefresh }))}>
                {syncForm.fullRefresh ? 'Full Refresh Enabled' : 'Enable Full Refresh'}
              </Button>
            </Stack>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeSyncDialog} disabled={syncing}>
            Cancel
          </Button>
          <Button onClick={() => void runSync()} variant="contained" color={syncForm.dryRun ? 'primary' : 'warning'} disabled={syncing}>
            {syncing ? 'Running…' : syncForm.dryRun ? 'Run Dry Sync' : 'Run Full Refresh'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
