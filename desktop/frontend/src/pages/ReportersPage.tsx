import React from 'react'
import {
  Alert,
  Autocomplete,
  Box,
  Button,
  Checkbox,
  Chip,
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
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { handleAppError } from '../errors/handleAppError'

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
  isActive: boolean
  groups: string[]
}

type MessageDialogState = {
  mode: 'single' | 'bulk'
  reporter?: Reporter | null
}

const emptyForm: ReporterFormState = {
  name: '',
  telephone: '',
  whatsapp: '',
  telegram: '',
  orgUnitId: '',
  isActive: true,
  groups: [],
}

const dataGridSx = {
  '& .MuiDataGrid-columnHeaderTitle': {
    fontWeight: 700,
  },
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
    isActive: reporter.isActive,
    groups: reporter.groups ?? [],
  }
}

function formatActionError(prefix: string, normalized: { message: string; fieldErrors?: Record<string, string[]>; requestId?: string }) {
  const detail = Object.values(normalized.fieldErrors ?? {}).flat()[0]
  const requestId = normalized.requestId ? ` Request ID: ${normalized.requestId}` : ''
  if (!detail || detail === normalized.message) {
    return `${prefix}${requestId}`
  }
  return `${prefix} ${detail}${requestId}`
}

export function ReportersPage() {
  const apiClient = useApiClient()
  const [reporters, setReporters] = React.useState<Reporter[]>([])
  const [orgUnits, setOrgUnits] = React.useState<OrgUnit[]>([])
  const [selectedIds, setSelectedIds] = React.useState<number[]>([])
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState('')
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [viewing, setViewing] = React.useState<Reporter | null>(null)
  const [editing, setEditing] = React.useState<Reporter | null>(null)
  const [form, setForm] = React.useState<ReporterFormState>(emptyForm)
  const [submitting, setSubmitting] = React.useState(false)
  const [messageDialog, setMessageDialog] = React.useState<MessageDialogState | null>(null)
  const [messageText, setMessageText] = React.useState('')
  const [messageSending, setMessageSending] = React.useState(false)

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
      setSelectedIds((current) => current.filter((id) => (reporterResponse.items ?? []).some((reporter) => reporter.id === id)))
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Unable to load reporters.')
    } finally {
      setLoading(false)
    }
  }, [apiClient])

  React.useEffect(() => {
    void load()
  }, [load])

  const selectedCount = selectedIds.length
  const selectedReporters = reporters.filter((reporter) => selectedIds.includes(reporter.id))

  const columns = React.useMemo<GridColDef<Reporter>[]>(
    () => [
      {
        field: 'selected',
        headerName: '',
        width: 64,
        sortable: false,
        filterable: false,
        renderCell: ({ row }) => (
          <Checkbox
            checked={selectedIds.includes(row.id)}
            inputProps={{ 'aria-label': `Select reporter ${row.name}` }}
            onChange={() => {
              setSelectedIds((current) =>
                current.includes(row.id) ? current.filter((id) => id !== row.id) : [...current, row.id],
              )
            }}
          />
        ),
      },
      { field: 'name', headerName: 'Reporter', flex: 1, minWidth: 180 },
      { field: 'telephone', headerName: 'Telephone', width: 150 },
      {
        field: 'syncStatus',
        headerName: 'Sync Status',
        width: 150,
        sortable: false,
        renderCell: ({ row }) => (
          <Chip
            label={row.synced && row.rapidProUuid ? 'Synced' : 'Pending'}
            color={row.synced && row.rapidProUuid ? 'success' : 'default'}
            size="small"
          />
        ),
      },
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
        width: 120,
        sortable: false,
        filterable: false,
        renderCell: ({ row }) => (
          <AdminRowActions
            rowLabel={row.name}
            actions={[
              { id: 'view', label: 'View details', icon: 'view', onClick: () => setViewing(row) },
              { id: 'edit', label: 'Edit', icon: 'edit', onClick: () => openDialog(row) },
              { id: 'sync', label: 'Sync to RapidPro', onClick: () => void syncReporter(row.id) },
              { id: 'send', label: 'Send SMS', onClick: () => openMessageDialog('single', row) },
              {
                id: 'delete',
                label: 'Delete',
                icon: 'delete',
                destructive: true,
                confirmTitle: 'Delete reporter',
                confirmMessage: `Delete ${row.name}? This cannot be undone.`,
                onClick: () => void deleteReporter(row),
              },
            ]}
          />
        ),
      },
    ],
    [orgUnits, selectedIds],
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

  function openMessageDialog(mode: 'single' | 'bulk', reporter?: Reporter | null) {
    setMessageDialog({ mode, reporter: reporter ?? null })
    setMessageText('')
    setError('')
  }

  function closeMessageDialog() {
    if (messageSending) {
      return
    }
    setMessageDialog(null)
    setMessageText('')
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
    setError('')
    try {
      await apiClient.request<void>(`/api/v1/reporters/${reporter.id}`, { method: 'DELETE' })
      await load()
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : 'Unable to delete reporter.')
    }
  }

  async function syncReporter(id: number) {
    setError('')
    try {
      await apiClient.request(`/api/v1/reporters/${id}/sync`, { method: 'POST', body: '{}' })
      await load()
    } catch (syncError) {
      const { error: normalized } = await handleAppError(syncError, {
        fallbackMessage: 'Unable to sync reporter.',
        notifyUser: false,
      })
      setError(formatActionError('Unable to sync reporter.', normalized))
    }
  }

  async function syncSelected() {
    setError('')
    try {
      await apiClient.request('/api/v1/reporters/bulk/sync', {
        method: 'POST',
        body: JSON.stringify({ reporterIds: selectedIds }),
      })
      await load()
    } catch (syncError) {
      const { error: normalized } = await handleAppError(syncError, {
        fallbackMessage: 'Unable to sync selected reporters.',
        notifyUser: false,
      })
      setError(formatActionError('Unable to sync selected reporters.', normalized))
    }
  }

  async function submitMessage() {
    if (!messageDialog) {
      return
    }
    setMessageSending(true)
    setError('')
    try {
      if (messageDialog.mode === 'single' && messageDialog.reporter) {
        await apiClient.request(`/api/v1/reporters/${messageDialog.reporter.id}/send-message`, {
          method: 'POST',
          body: JSON.stringify({ text: messageText.trim() }),
        })
      } else {
        await apiClient.request('/api/v1/reporters/bulk/broadcast', {
          method: 'POST',
          body: JSON.stringify({ reporterIds: selectedIds, text: messageText.trim() }),
        })
      }
      closeMessageDialog()
      await load()
    } catch (messageError) {
      setError(messageError instanceof Error ? messageError.message : 'Unable to send message.')
    } finally {
      setMessageSending(false)
    }
  }

  return (
    <Box>
      <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" spacing={2} sx={{ mb: 2 }}>
        <Box>
          <Typography variant="h4" component="h1">
            Reporters
          </Typography>
          <Typography color="text.secondary">Manage local reporters, RapidPro contact sync, and outbound SMS.</Typography>
        </Box>
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
          <Button variant="outlined" onClick={() => void syncSelected()} disabled={selectedCount === 0}>
            Sync Selected
          </Button>
          <Button variant="outlined" onClick={() => openMessageDialog('bulk')} disabled={selectedCount === 0}>
            Broadcast to Selected
          </Button>
          <Button variant="contained" onClick={() => openDialog()}>
            New Reporter
          </Button>
        </Stack>
      </Stack>

      {selectedCount > 0 ? (
        <Alert severity="info" sx={{ mb: 2 }}>
          {selectedCount} reporter{selectedCount === 1 ? '' : 's'} selected.
        </Alert>
      ) : null}

      {error ? <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert> : null}

      <DataGrid
        autoHeight
        rows={reporters}
        columns={columns}
        loading={loading}
        disableRowSelectionOnClick
        initialState={{ pagination: { paginationModel: { page: 0, pageSize: 25 } } }}
        pageSizeOptions={[25, 50, 100]}
        showToolbar
        slotProps={{
          toolbar: {
            csvOptions: {
              fileName: 'reporters-grid',
            },
          },
        }}
        sx={dataGridSx}
      />

      <Dialog open={Boolean(viewing)} onClose={() => setViewing(null)} fullWidth maxWidth="md">
        <DialogTitle>Reporter Details</DialogTitle>
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
            <TextField label="Telephone" value={viewing?.telephone ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="WhatsApp" value={viewing?.whatsapp ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="Telegram" value={viewing?.telegram ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="UID" value={viewing?.uid ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="RapidPro UUID" value={viewing?.rapidProUuid ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="Facility" value={viewing ? orgUnits.find((unit) => unit.id === viewing.orgUnitId)?.name ?? String(viewing.orgUnitId) : ''} InputProps={{ readOnly: true }} />
            <TextField label="Reporting Location" value={viewing?.reportingLocation ?? ''} InputProps={{ readOnly: true }} />
            <TextField
              label="District"
              value={viewing?.districtId ? orgUnits.find((unit) => unit.id === viewing.districtId)?.name ?? String(viewing.districtId) : ''}
              InputProps={{ readOnly: true }}
            />
            <TextField label="Last Login" value={viewing?.lastLoginAt ? new Date(viewing.lastLoginAt).toLocaleString() : ''} InputProps={{ readOnly: true }} />
            <TextField label="Total Reports" value={String(viewing?.totalReports ?? 0)} InputProps={{ readOnly: true }} />
            <TextField label="Last Reporting Date" value={viewing?.lastReportingDate ? new Date(viewing.lastReportingDate).toLocaleString() : ''} InputProps={{ readOnly: true }} />
            <TextField label="SMS Code" value={viewing?.smsCode ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="SMS Code Expires At" value={viewing?.smsCodeExpiresAt ? new Date(viewing.smsCodeExpiresAt).toLocaleString() : ''} InputProps={{ readOnly: true }} />
            <TextField label="MT UUID" value={viewing?.mtuuid ?? ''} InputProps={{ readOnly: true }} />
            <TextField label="Created At" value={viewing?.createdAt ? new Date(viewing.createdAt).toLocaleString() : ''} InputProps={{ readOnly: true }} />
            <TextField label="Updated At" value={viewing?.updatedAt ? new Date(viewing.updatedAt).toLocaleString() : ''} InputProps={{ readOnly: true }} />
            <TextField label="Reporter Groups" value={viewing?.groups?.join(', ') ?? ''} InputProps={{ readOnly: true }} sx={{ gridColumn: '1 / -1' }} />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setViewing(null)}>Close</Button>
        </DialogActions>
      </Dialog>

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
            <TextField label="RapidPro UUID" value={editing?.rapidProUuid ?? 'Generated by sync'} InputProps={{ readOnly: true }} />
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
            <TextField label="Synced" value={editing ? String(editing.synced) : 'Pending'} InputProps={{ readOnly: true }} />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeDialog} disabled={submitting}>Cancel</Button>
          <Button onClick={() => void submitReporter()} disabled={submitting || !form.name.trim() || !form.telephone.trim() || !form.orgUnitId} variant="contained">
            {editing ? 'Save' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={Boolean(messageDialog)} onClose={closeMessageDialog} fullWidth maxWidth="sm">
        <DialogTitle>{messageDialog?.mode === 'single' ? 'Send SMS' : 'Broadcast to Selected Reporters'}</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            <Typography color="text.secondary">
              {messageDialog?.mode === 'single'
                ? `Send a message to ${messageDialog.reporter?.name ?? 'the selected reporter'}.`
                : `Send a broadcast to ${selectedReporters.length} selected reporter${selectedReporters.length === 1 ? '' : 's'}.`}
            </Typography>
            <TextField
              label="Message"
              multiline
              minRows={4}
              value={messageText}
              onChange={(event) => setMessageText(event.target.value)}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeMessageDialog} disabled={messageSending}>Cancel</Button>
          <Button onClick={() => void submitMessage()} disabled={messageSending || !messageText.trim()} variant="contained">
            {messageDialog?.mode === 'single' ? 'Send SMS' : 'Send Broadcast'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
