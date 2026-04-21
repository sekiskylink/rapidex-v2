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
import { apiRequest } from '../lib/api'

interface Reporter {
  id: number
  uid: string
  contactUuid: string
  phoneNumber: string
  displayName: string
  orgUnitId: number
  isActive: boolean
}

interface OrgUnit {
  id: number
  name: string
}

interface ListResponse<T> {
  items: T[]
  totalCount: number
}

const emptyForm = {
  contactUuid: '',
  phoneNumber: '',
  displayName: '',
  orgUnitId: '',
}

export function ReportersPage() {
  const [reporters, setReporters] = React.useState<Reporter[]>([])
  const [orgUnits, setOrgUnits] = React.useState<OrgUnit[]>([])
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState('')
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [form, setForm] = React.useState(emptyForm)
  const [submitting, setSubmitting] = React.useState(false)

  const load = React.useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [reporterResponse, orgUnitResponse] = await Promise.all([
        apiRequest<ListResponse<Reporter>>('/reporters?page=0&pageSize=200'),
        apiRequest<ListResponse<OrgUnit>>('/orgunits?page=0&pageSize=200'),
      ])
      setReporters(reporterResponse.items ?? [])
      setOrgUnits(orgUnitResponse.items ?? [])
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Unable to load reporters.')
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    void load()
  }, [load])

  const columns = React.useMemo<GridColDef<Reporter>[]>(
    () => [
      { field: 'displayName', headerName: 'Reporter', flex: 1, minWidth: 180 },
      { field: 'phoneNumber', headerName: 'Phone', width: 160 },
      { field: 'contactUuid', headerName: 'RapidPro Contact', flex: 1, minWidth: 180 },
      {
        field: 'orgUnitId',
        headerName: 'Facility',
        flex: 1,
        minWidth: 180,
        valueGetter: (_value, row) => orgUnits.find((unit) => unit.id === row.orgUnitId)?.name ?? '',
      },
      { field: 'isActive', headerName: 'Active', width: 100, type: 'boolean' },
    ],
    [orgUnits],
  )

  async function createReporter() {
    setSubmitting(true)
    setError('')
    try {
      await apiRequest<Reporter>('/reporters', {
        method: 'POST',
        body: JSON.stringify({
          contactUuid: form.contactUuid.trim(),
          phoneNumber: form.phoneNumber.trim(),
          displayName: form.displayName.trim(),
          orgUnitId: Number(form.orgUnitId),
        }),
      })
      setDialogOpen(false)
      setForm(emptyForm)
      await load()
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : 'Unable to create reporter.')
    } finally {
      setSubmitting(false)
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
        <Button variant="contained" onClick={() => setDialogOpen(true)}>
          New Reporter
        </Button>
      </Stack>

      {error ? <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert> : null}

      <DataGrid
        autoHeight
        rows={reporters}
        columns={columns}
        loading={loading}
        disableRowSelectionOnClick
        initialState={{ pagination: { paginationModel: { page: 0, pageSize: 25 } } }}
        pageSizeOptions={[25, 50, 100]}
      />

      <Dialog open={dialogOpen} onClose={submitting ? undefined : () => setDialogOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>New Reporter</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            <TextField label="Display Name" value={form.displayName} onChange={(event) => setForm({ ...form, displayName: event.target.value })} required />
            <TextField label="Phone Number" value={form.phoneNumber} onChange={(event) => setForm({ ...form, phoneNumber: event.target.value })} required />
            <TextField label="RapidPro Contact UUID" value={form.contactUuid} onChange={(event) => setForm({ ...form, contactUuid: event.target.value })} required />
            <TextField select label="Facility" value={form.orgUnitId} onChange={(event) => setForm({ ...form, orgUnitId: event.target.value })} required>
              <MenuItem value="">Select facility</MenuItem>
              {orgUnits.map((unit) => (
                <MenuItem key={unit.id} value={String(unit.id)}>
                  {unit.name}
                </MenuItem>
              ))}
            </TextField>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)} disabled={submitting}>Cancel</Button>
          <Button onClick={createReporter} disabled={submitting || !form.displayName.trim() || !form.phoneNumber.trim() || !form.contactUuid.trim() || !form.orgUnitId} variant="contained">Create</Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
