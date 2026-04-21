import React from 'react'
import {
  Alert,
  Autocomplete,
  Box,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  MenuItem,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import type { GridColDef } from '@mui/x-data-grid'
import { DataGrid } from '@mui/x-data-grid'
import { useApiClient } from '../api/useApiClient'

interface Reporter {
  id: number
  uid: string
  name: string
  telephone: string
  whatsapp: string
  telegram: string
  orgUnitId: number
  reportingLocation: string
  districtId?: number | null
  totalReports: number
  lastReportingDate?: string | null
  smsCode: string
  smsCodeExpiresAt?: string | null
  mtuuid: string
  synced: boolean
  rapidProUuid: string
  isActive: boolean
  createdAt: string
  updatedAt: string
  lastLoginAt?: string | null
  groups: string[]
}

interface OrgUnit {
  id: number
  name: string
}

interface ListResponse<T> {
  items: T[]
  totalCount: number
}

type ReporterFormState = {
  name: string
  telephone: string
  whatsapp: string
  telegram: string
  orgUnitId: string
  smsCode: string
  mtuuid: string
  rapidProUuid: string
  isActive: boolean
  groups: string[]
}

const emptyForm: ReporterFormState = {
  name: '',
  telephone: '',
  whatsapp: '',
  telegram: '',
  orgUnitId: '',
  smsCode: '',
  mtuuid: '',
  rapidProUuid: '',
  isActive: true,
  groups: [],
}

function toForm(reporter?: Reporter | null): ReporterFormState {
  if (!reporter) {
    return emptyForm
  }
  return {
    name: reporter.name ?? '',
    telephone: reporter.telephone ?? '',
    whatsapp: reporter.whatsapp ?? '',
    telegram: reporter.telegram ?? '',
    orgUnitId: reporter.orgUnitId ? String(reporter.orgUnitId) : '',
    smsCode: reporter.smsCode ?? '',
    mtuuid: reporter.mtuuid ?? '',
    rapidProUuid: reporter.rapidProUuid ?? '',
    isActive: reporter.isActive,
    groups: reporter.groups ?? [],
  }
}

export function ReportersPage() {
  const apiClient = useApiClient()
  const [reporters, setReporters] = React.useState<Reporter[]>([])
  const [orgUnits, setOrgUnits] = React.useState<OrgUnit[]>([])
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState('')
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [editing, setEditing] = React.useState<Reporter | null>(null)
  const [form, setForm] = React.useState<ReporterFormState>(emptyForm)
  const [submitting, setSubmitting] = React.useState(false)

  const load = React.useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [reporterResponse, orgUnitResponse] = await Promise.all([
        apiClient.request<ListResponse<Reporter>>('/api/v1/reporters?page=0&pageSize=200'),
        apiClient.request<ListResponse<OrgUnit>>('/api/v1/orgunits?page=0&pageSize=200'),
      ])
      setReporters(reporterResponse.items ?? [])
      setOrgUnits(orgUnitResponse.items ?? [])
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Unable to load reporters.')
    } finally {
      setLoading(false)
    }
  }, [apiClient])

  React.useEffect(() => {
    void load()
  }, [load])

  const columns = React.useMemo<GridColDef<Reporter>[]>(
    () => [
      { field: 'name', headerName: 'Reporter', flex: 1, minWidth: 180 },
      { field: 'telephone', headerName: 'Telephone', width: 150 },
      { field: 'rapidProUuid', headerName: 'RapidPro UUID', flex: 1, minWidth: 180 },
      {
        field: 'orgUnitId',
        headerName: 'Facility',
        flex: 1,
        minWidth: 180,
        valueGetter: (_value, row) => orgUnits.find((unit) => unit.id === row.orgUnitId)?.name ?? '',
      },
      { field: 'groups', headerName: 'Groups', flex: 1, minWidth: 160, valueGetter: (_value, row) => row.groups.join(', ') },
      { field: 'isActive', headerName: 'Active', width: 100, type: 'boolean' },
      {
        field: 'actions',
        headerName: 'Actions',
        width: 170,
        sortable: false,
        filterable: false,
        renderCell: ({ row }) => (
          <Stack direction="row" spacing={1}>
            <Button size="small" onClick={() => openDialog(row)}>
              Edit
            </Button>
            <Button size="small" color="error" onClick={() => void deleteReporter(row)}>
              Delete
            </Button>
          </Stack>
        ),
      },
    ],
    [orgUnits],
  )

  function openDialog(reporter?: Reporter) {
    setEditing(reporter ?? null)
    setForm(toForm(reporter ?? null))
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

  async function submitReporter() {
    setSubmitting(true)
    setError('')
    try {
      await apiClient.request<Reporter>(editing ? `/api/v1/reporters/${editing.id}` : '/api/v1/reporters', {
        method: editing ? 'PUT' : 'POST',
        body: JSON.stringify({
          uid: editing?.uid ?? '',
          name: form.name.trim(),
          telephone: form.telephone.trim(),
          whatsapp: form.whatsapp.trim(),
          telegram: form.telegram.trim(),
          orgUnitId: Number(form.orgUnitId),
          smsCode: form.smsCode.trim(),
          mtuuid: form.mtuuid.trim(),
          rapidProUuid: form.rapidProUuid.trim(),
          isActive: form.isActive,
          groups: form.groups,
          synced: editing?.synced ?? false,
          totalReports: editing?.totalReports ?? 0,
          lastReportingDate: editing?.lastReportingDate ?? null,
          lastLoginAt: editing?.lastLoginAt ?? null,
        }),
      })
      closeDialog()
      await load()
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : 'Unable to save reporter.')
    } finally {
      setSubmitting(false)
    }
  }

  async function deleteReporter(reporter: Reporter) {
    if (!window.confirm(`Delete reporter "${reporter.name}"?`)) {
      return
    }
    setError('')
    try {
      await apiClient.request<void>(`/api/v1/reporters/${reporter.id}`, { method: 'DELETE' })
      await load()
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : 'Unable to delete reporter.')
    }
  }

  return (
    <Box>
      <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" spacing={2} sx={{ mb: 2 }}>
        <Box>
          <Typography variant="h4" component="h1">
            Reporters
          </Typography>
          <Typography color="text.secondary">RapidPro contacts mapped to reporting facilities.</Typography>
        </Box>
        <Button variant="contained" onClick={() => openDialog()}>
          New Reporter
        </Button>
      </Stack>

      {error ? <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert> : null}

      <DataGrid autoHeight rows={reporters} columns={columns} loading={loading} disableRowSelectionOnClick pageSizeOptions={[25, 50, 100]} />

      <Dialog open={dialogOpen} onClose={closeDialog} fullWidth maxWidth="lg">
        <DialogTitle>{editing ? 'Edit Reporter' : 'New Reporter'}</DialogTitle>
        <DialogContent>
          <Box
            sx={{
              pt: 1,
              display: 'grid',
              gap: 2,
              gridTemplateColumns: { xs: '1fr', md: 'repeat(2, minmax(0, 1fr))' },
            }}
          >
            <TextField label="Name" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required />
            <TextField label="Telephone" value={form.telephone} onChange={(event) => setForm({ ...form, telephone: event.target.value })} required />
            <TextField label="WhatsApp" value={form.whatsapp} onChange={(event) => setForm({ ...form, whatsapp: event.target.value })} />
            <TextField label="Telegram" value={form.telegram} onChange={(event) => setForm({ ...form, telegram: event.target.value })} />
            <TextField select label="Facility" value={form.orgUnitId} onChange={(event) => setForm({ ...form, orgUnitId: event.target.value })} required>
              <MenuItem value="">Select facility</MenuItem>
              {orgUnits.map((unit) => (
                <MenuItem key={unit.id} value={String(unit.id)}>
                  {unit.name}
                </MenuItem>
              ))}
            </TextField>
            <TextField label="RapidPro UUID" value={form.rapidProUuid} onChange={(event) => setForm({ ...form, rapidProUuid: event.target.value })} />
            <TextField label="SMS Code" value={form.smsCode} onChange={(event) => setForm({ ...form, smsCode: event.target.value })} />
            <TextField label="MT UUID" value={form.mtuuid} onChange={(event) => setForm({ ...form, mtuuid: event.target.value })} />
            <Autocomplete
              multiple
              freeSolo
              options={[]}
              value={form.groups}
              onChange={(_event, value) => setForm({ ...form, groups: value.map((item) => item.trim()).filter(Boolean) })}
              renderInput={(params) => <TextField {...params} label="Reporter Groups" placeholder="Type and press Enter" />}
              sx={{ gridColumn: '1 / -1' }}
            />
            <FormControlLabel
              control={<Checkbox checked={form.isActive} onChange={(event) => setForm({ ...form, isActive: event.target.checked })} />}
              label="Reporter is active"
            />
            <Box />
            <TextField label="UID" value={editing?.uid ?? 'Generated on save'} InputProps={{ readOnly: true }} />
            <TextField label="Reporting Location" value={editing?.reportingLocation ?? 'Derived from facility'} InputProps={{ readOnly: true }} />
            <TextField
              label="District"
              value={editing?.districtId ? orgUnits.find((unit) => unit.id === editing.districtId)?.name ?? String(editing.districtId) : 'Derived from hierarchy'}
              InputProps={{ readOnly: true }}
            />
            <TextField label="Total Reports" value={editing?.totalReports ?? 0} InputProps={{ readOnly: true }} />
            <TextField label="Last Reporting Date" value={editing?.lastReportingDate ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="Synced" value={editing ? String(editing.synced) : 'Derived later'} InputProps={{ readOnly: true }} />
            <TextField label="Created At" value={editing?.createdAt ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="Updated At" value={editing?.updatedAt ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="Last Login At" value={editing?.lastLoginAt ?? ''} InputProps={{ readOnly: true }} />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeDialog} disabled={submitting}>Cancel</Button>
          <Button onClick={() => void submitReporter()} disabled={submitting || !form.name.trim() || !form.telephone.trim() || !form.orgUnitId} variant="contained">
            {editing ? 'Save' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
