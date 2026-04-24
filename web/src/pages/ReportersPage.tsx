import React from 'react'
import {
  Alert,
  Autocomplete,
  Box,
  Button,
  Checkbox,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContentText,
  DialogContent,
  DialogTitle,
  Divider,
  FormControlLabel,
  List,
  ListItemButton,
  ListItemText,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import type { GridColDef, GridPaginationModel } from '@mui/x-data-grid'
import { DataGrid } from '@mui/x-data-grid'
import { getAuthSnapshot } from '../auth/state'
import { AdminRowActions } from '../components/admin/AdminRowActions'
import { handleAppError } from '../errors/handleAppError'
import { apiRequest } from '../lib/api'
import { useAppNotify } from '../notifications/facade'
import { ArticleRoundedIcon, CampaignRoundedIcon, MessageRoundedIcon, PersonAddRoundedIcon, SyncRoundedIcon } from '../ui/icons'
import {
  ChatHistoryDialog,
  RapidProDetailsDialog,
  ReporterReportsDialog,
  ReporterDetailsDialog,
  type RapidProContactDetailsResponse,
  type RapidProMessageHistoryResponse,
  type ReporterReportsResponse,
} from './reporter-dialogs'

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
  uid?: string
  name: string
  parentId?: number | null
  hierarchyLevel?: number
  path?: string
  displayPath?: string
  hasChildren?: boolean
}

interface ReporterGroupOption {
  id: number
  name: string
}

interface ListResponse<T> {
  items: T[]
  totalCount: number
}

interface JurisdictionBroadcastQueueResponse {
  status: 'queued' | 'duplicate_pending'
  message: string
  broadcast: {
    id: number
    matchedCount: number
    status: string
  }
}

interface BroadcastHistoryItem {
  id: number
  uid: string
  requestedByUserId: number
  orgUnitIds: number[]
  reporterGroup: string
  messageText: string
  matchedCount: number
  sentCount: number
  failedCount: number
  status: string
  lastError: string
  requestedAt: string
  startedAt?: string | null
  finishedAt?: string | null
  claimedByWorkerRunId?: number | null
}

type FacilityBrowserEntry = {
  unit?: OrgUnit
  label: string
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

type JurisdictionBroadcastFormState = {
  orgUnits: OrgUnit[]
  reporterGroup: string
  text: string
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

const emptyJurisdictionBroadcastForm: JurisdictionBroadcastFormState = {
  orgUnits: [],
  reporterGroup: '',
  text: '',
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

function uniqOrgUnits(items: OrgUnit[]) {
  const seen = new Set<number>()
  return items.filter((item) => {
    if (seen.has(item.id)) {
      return false
    }
    seen.add(item.id)
    return true
  })
}

function formatFacilityPath(unit: OrgUnit | null) {
  if (unit?.displayPath) {
    return unit.displayPath
  }
  if (!unit?.path) {
    return ''
  }
  return unit.path
    .split('/')
    .map((part) => part.trim())
    .filter(Boolean)
    .join(' / ')
}

function formatTimestamp(value?: string | null) {
  if (!value) {
    return '-'
  }
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return value
  }
  return parsed.toLocaleString()
}

function broadcastStatusColor(status: string): 'default' | 'info' | 'warning' | 'success' | 'error' {
  switch (status) {
    case 'queued':
      return 'warning'
    case 'running':
      return 'info'
    case 'completed':
      return 'success'
    case 'failed':
      return 'error'
    default:
      return 'default'
  }
}

export function ReportersPage() {
  const currentUser = getAuthSnapshot().user
  const notify = useAppNotify()
  const [reporters, setReporters] = React.useState<Reporter[]>([])
  const [orgUnits, setOrgUnits] = React.useState<OrgUnit[]>([])
  const [reporterGroupOptions, setReporterGroupOptions] = React.useState<ReporterGroupOption[]>([])
  const [broadcastHistory, setBroadcastHistory] = React.useState<BroadcastHistoryItem[]>([])
  const [broadcastHistoryLoading, setBroadcastHistoryLoading] = React.useState(true)
  const [broadcastHistoryError, setBroadcastHistoryError] = React.useState('')
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
  const [jurisdictionDialogOpen, setJurisdictionDialogOpen] = React.useState(false)
  const [jurisdictionForm, setJurisdictionForm] = React.useState<JurisdictionBroadcastFormState>(emptyJurisdictionBroadcastForm)
  const [jurisdictionOrgUnitOptions, setJurisdictionOrgUnitOptions] = React.useState<OrgUnit[]>([])
  const [jurisdictionOrgUnitSearch, setJurisdictionOrgUnitSearch] = React.useState('')
  const [jurisdictionOrgUnitLoading, setJurisdictionOrgUnitLoading] = React.useState(false)
  const [jurisdictionSubmitting, setJurisdictionSubmitting] = React.useState(false)
  const [rapidProReporter, setRapidProReporter] = React.useState<Reporter | null>(null)
  const [rapidProDetails, setRapidProDetails] = React.useState<RapidProContactDetailsResponse | null>(null)
  const [rapidProDetailsLoading, setRapidProDetailsLoading] = React.useState(false)
  const [rapidProDetailsError, setRapidProDetailsError] = React.useState('')
  const [chatHistoryReporter, setChatHistoryReporter] = React.useState<Reporter | null>(null)
  const [chatHistory, setChatHistory] = React.useState<RapidProMessageHistoryResponse | null>(null)
  const [chatHistoryLoading, setChatHistoryLoading] = React.useState(false)
  const [chatHistoryError, setChatHistoryError] = React.useState('')
  const [reportsReporter, setReportsReporter] = React.useState<Reporter | null>(null)
  const [recentReports, setRecentReports] = React.useState<ReporterReportsResponse | null>(null)
  const [recentReportsLoading, setRecentReportsLoading] = React.useState(false)
  const [recentReportsError, setRecentReportsError] = React.useState('')
  const [selectedReportId, setSelectedReportId] = React.useState<number | null>(null)
  const [paginationModel, setPaginationModel] = React.useState<GridPaginationModel>({ page: 0, pageSize: 25 })
  const [selectedFacility, setSelectedFacility] = React.useState<OrgUnit | null>(null)
  const [facilitySearchInput, setFacilitySearchInput] = React.useState('')
  const [facilityOptions, setFacilityOptions] = React.useState<OrgUnit[]>([])
  const [facilitySearchLoading, setFacilitySearchLoading] = React.useState(false)
  const [facilityBrowserOpen, setFacilityBrowserOpen] = React.useState(false)
  const [facilityBrowserTrail, setFacilityBrowserTrail] = React.useState<FacilityBrowserEntry[]>([])
  const [facilityBrowserItems, setFacilityBrowserItems] = React.useState<OrgUnit[]>([])
  const [facilityBrowserLoading, setFacilityBrowserLoading] = React.useState(false)

  const load = React.useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [reporterResponse, orgUnitResponse, reporterGroupResponse] = await Promise.all([
        apiRequest<ListResponse<Reporter>>('/reporters?page=0&pageSize=200'),
        apiRequest<ListResponse<OrgUnit>>('/orgunits?page=0&pageSize=200'),
        apiRequest<{ items: ReporterGroupOption[] }>('/reporter-groups/options'),
      ])
      setReporters(reporterResponse.items ?? [])
      setOrgUnits(orgUnitResponse.items ?? [])
      setReporterGroupOptions(reporterGroupResponse.items ?? [])
      setSelectedIds((current) => current.filter((id) => (reporterResponse.items ?? []).some((reporter) => reporter.id === id)))
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Unable to load reporters.')
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    void load()
  }, [load])

  const loadBroadcastHistory = React.useCallback(async () => {
    setBroadcastHistoryLoading(true)
    setBroadcastHistoryError('')
    try {
      const response = await apiRequest<ListResponse<BroadcastHistoryItem>>('/reporters/broadcasts?page=0&pageSize=10')
      setBroadcastHistory(response.items ?? [])
    } catch (historyError) {
      setBroadcastHistoryError(historyError instanceof Error ? historyError.message : 'Unable to load broadcast history.')
    } finally {
      setBroadcastHistoryLoading(false)
    }
  }, [])

  React.useEffect(() => {
    void loadBroadcastHistory()
  }, [loadBroadcastHistory])

  const selectedCount = selectedIds.length
  const selectedReporters = reporters.filter((reporter) => selectedIds.includes(reporter.id))
  const visibleReporterIds = React.useMemo(() => {
    const start = paginationModel.page * paginationModel.pageSize
    return reporters.slice(start, start + paginationModel.pageSize).map((reporter) => reporter.id)
  }, [paginationModel.page, paginationModel.pageSize, reporters])
  const allVisibleSelected = visibleReporterIds.length > 0 && visibleReporterIds.every((id) => selectedIds.includes(id))
  const someVisibleSelected = visibleReporterIds.some((id) => selectedIds.includes(id))
  const reporterGroupNames = React.useMemo(
    () => Array.from(new Set([...reporterGroupOptions.map((item) => item.name), ...form.groups])).sort((left, right) => left.localeCompare(right)),
    [form.groups, reporterGroupOptions],
  )

  const getOrgUnitName = React.useCallback((id?: number | null) => {
    if (!id) {
      return ''
    }
    return orgUnits.find((unit) => unit.id === id)?.name ?? String(id)
  }, [orgUnits])

  const fetchFacilityOptions = React.useCallback(async (search: string) => {
    const response = await apiRequest<ListResponse<OrgUnit>>(`/orgunits?page=0&pageSize=20&leafOnly=true&search=${encodeURIComponent(search)}`)
    return response.items ?? []
  }, [])

  const fetchJurisdictionOrgUnitOptions = React.useCallback(async (search: string) => {
    const response = await apiRequest<ListResponse<OrgUnit>>(`/orgunits?page=0&pageSize=20&search=${encodeURIComponent(search)}`)
    return response.items ?? []
  }, [])

  const loadFacilityBrowserLevel = React.useCallback(async (trail: FacilityBrowserEntry[]) => {
    setFacilityBrowserLoading(true)
    try {
      const current = trail[trail.length - 1]?.unit
      const query = current ? `parentId=${current.id}` : 'rootsOnly=true'
      const response = await apiRequest<ListResponse<OrgUnit>>(`/orgunits?page=0&pageSize=200&${query}`)
      setFacilityBrowserTrail(trail)
      setFacilityBrowserItems(response.items ?? [])
    } catch (browserError) {
      setError(browserError instanceof Error ? browserError.message : 'Unable to load facility hierarchy.')
    } finally {
      setFacilityBrowserLoading(false)
    }
  }, [])

  const applyFacilitySelection = React.useCallback((unit: OrgUnit | null) => {
    setSelectedFacility(unit)
    setForm((current) => ({
      ...current,
      orgUnitId: unit ? String(unit.id) : '',
    }))
    setFacilitySearchInput(unit?.name ?? '')
    setFacilityOptions(unit ? [unit] : [])
  }, [])

  React.useEffect(() => {
    if (!dialogOpen) {
      return
    }
    const search = facilitySearchInput.trim()
    if (search === '') {
      setFacilityOptions(selectedFacility ? [selectedFacility] : [])
      setFacilitySearchLoading(false)
      return
    }

    let active = true
    setFacilitySearchLoading(true)
    const timeoutId = window.setTimeout(() => {
      void fetchFacilityOptions(search)
        .then((items) => {
          if (!active) {
            return
          }
          setFacilityOptions(uniqOrgUnits([...(selectedFacility ? [selectedFacility] : []), ...items]))
        })
        .catch((searchError) => {
          if (!active) {
            return
          }
          setError(searchError instanceof Error ? searchError.message : 'Unable to search facilities.')
        })
        .finally(() => {
          if (active) {
            setFacilitySearchLoading(false)
          }
        })
    }, 250)

    return () => {
      active = false
      window.clearTimeout(timeoutId)
    }
  }, [dialogOpen, facilitySearchInput, fetchFacilityOptions, selectedFacility])

  React.useEffect(() => {
    if (!jurisdictionDialogOpen) {
      return
    }
    const search = jurisdictionOrgUnitSearch.trim()
    if (search === '') {
      setJurisdictionOrgUnitOptions(uniqOrgUnits([...jurisdictionForm.orgUnits, ...orgUnits.slice(0, 20)]))
      setJurisdictionOrgUnitLoading(false)
      return
    }
    let active = true
    setJurisdictionOrgUnitLoading(true)
    const timeoutId = window.setTimeout(() => {
      void fetchJurisdictionOrgUnitOptions(search)
        .then((items) => {
          if (!active) {
            return
          }
          setJurisdictionOrgUnitOptions(uniqOrgUnits([...jurisdictionForm.orgUnits, ...items]))
        })
        .catch((searchError) => {
          if (!active) {
            return
          }
          setError(searchError instanceof Error ? searchError.message : 'Unable to search organisation units.')
        })
        .finally(() => {
          if (active) {
            setJurisdictionOrgUnitLoading(false)
          }
        })
    }, 250)
    return () => {
      active = false
      window.clearTimeout(timeoutId)
    }
  }, [fetchJurisdictionOrgUnitOptions, jurisdictionDialogOpen, jurisdictionForm.orgUnits, jurisdictionOrgUnitSearch, orgUnits])

  const openRapidProDetails = React.useCallback(async (reporter: Reporter) => {
    setRapidProReporter(reporter)
    setRapidProDetails(null)
    setRapidProDetailsError('')
    setRapidProDetailsLoading(true)
    try {
      const response = await apiRequest<RapidProContactDetailsResponse>(`/reporters/${reporter.id}/rapidpro-contact`)
      setRapidProDetails(response)
    } catch (detailError) {
      setRapidProDetailsError(detailError instanceof Error ? detailError.message : 'Unable to load RapidPro contact details.')
    } finally {
      setRapidProDetailsLoading(false)
    }
  }, [])

  const openChatHistoryDialog = React.useCallback(async (reporter: Reporter) => {
    setChatHistoryReporter(reporter)
    setChatHistory(null)
    setChatHistoryError('')
    setChatHistoryLoading(true)
    try {
      const response = await apiRequest<RapidProMessageHistoryResponse>(`/reporters/${reporter.id}/chat-history`)
      setChatHistory(response)
    } catch (historyError) {
      setChatHistoryError(historyError instanceof Error ? historyError.message : 'Unable to load chat history.')
    } finally {
      setChatHistoryLoading(false)
    }
  }, [])

  const openReportsDialog = React.useCallback(async (reporter: Reporter) => {
    setReportsReporter(reporter)
    setRecentReports(null)
    setRecentReportsError('')
    setRecentReportsLoading(true)
    setSelectedReportId(null)
    try {
      const response = await apiRequest<ReporterReportsResponse>(`/reporters/${reporter.id}/reports`)
      setRecentReports(response)
      setSelectedReportId(response.items?.[0]?.id ?? null)
    } catch (reportsError) {
      setRecentReportsError(reportsError instanceof Error ? reportsError.message : 'Unable to load recent reports.')
    } finally {
      setRecentReportsLoading(false)
    }
  }, [])

  const columns = React.useMemo<GridColDef<Reporter>[]>(
    () => [
      {
        field: 'selected',
        headerName: '',
        width: 72,
        sortable: false,
        filterable: false,
        disableColumnMenu: true,
        renderHeader: () => (
          <Checkbox
            checked={allVisibleSelected}
            indeterminate={!allVisibleSelected && someVisibleSelected}
            inputProps={{ 'aria-label': 'Select all rows' }}
            onChange={(event) => {
              const checked = event.target.checked
              setSelectedIds((current) => {
                const next = new Set(current)
                for (const id of visibleReporterIds) {
                  if (checked) {
                    next.add(id)
                  } else {
                    next.delete(id)
                  }
                }
                return Array.from(next)
              })
            }}
          />
        ),
        renderCell: ({ row }) => (
          <Checkbox
            checked={selectedIds.includes(row.id)}
            inputProps={{ 'aria-label': `Select reporter ${row.name}` }}
            onChange={(event) => {
              const checked = event.target.checked
              setSelectedIds((current) => {
                if (checked) {
                  return current.includes(row.id) ? current : [...current, row.id]
                }
                return current.filter((id) => id !== row.id)
              })
            }}
          />
        ),
      },
      { field: 'name', headerName: 'Reporter', flex: 1, minWidth: 180 },
      {
        field: 'telephone',
        headerName: 'Telephone',
        width: 170,
        renderCell: ({ row }) =>
          row.telephone ? (
            <Button
              size="small"
              variant="text"
              sx={{ p: 0, minWidth: 0, justifyContent: 'flex-start', textTransform: 'none' }}
              onClick={() => void openChatHistoryDialog(row)}
            >
              {row.telephone}
            </Button>
          ) : (
            '-'
          ),
      },
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
        valueGetter: (_value, row) => getOrgUnitName(row.orgUnitId),
      },
      { field: 'groups', headerName: 'Groups', flex: 1, minWidth: 160, valueGetter: (_value, row) => row.groups.join(', ') },
      { field: 'isActive', headerName: 'Active', width: 100, type: 'boolean' },
      {
        field: 'actions',
        headerName: 'Actions',
        width: 140,
        sortable: false,
        filterable: false,
        renderCell: ({ row }) => (
          <AdminRowActions
            rowLabel={row.name}
            actions={[
              { id: 'view', label: 'View details', icon: 'view', onClick: () => setViewing(row) },
              { id: 'reports', label: 'Reports', icon: <ArticleRoundedIcon fontSize="small" />, onClick: () => void openReportsDialog(row) },
              { id: 'rapidpro', label: 'RapidPro Details', icon: 'rapidpro', onClick: () => void openRapidProDetails(row) },
              { id: 'edit', label: 'Edit', icon: 'edit', onClick: () => openDialog(row) },
              { id: 'sync', label: 'Sync to RapidPro', icon: 'sync', onClick: () => void syncReporter(row.id) },
              { id: 'send', label: 'Send Message', icon: 'message', onClick: () => openMessageDialog('single', row) },
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
    [allVisibleSelected, getOrgUnitName, openChatHistoryDialog, openRapidProDetails, openReportsDialog, selectedIds, someVisibleSelected, visibleReporterIds],
  )

  const broadcastColumns = React.useMemo<GridColDef<BroadcastHistoryItem>[]>(
    () => [
      {
        field: 'requestedAt',
        headerName: 'Requested',
        minWidth: 180,
        flex: 1,
        valueGetter: (value) => formatTimestamp(String(value ?? '')),
      },
      {
        field: 'status',
        headerName: 'Status',
        minWidth: 120,
        renderCell: ({ value }) => <Chip size="small" label={String(value ?? 'unknown')} color={broadcastStatusColor(String(value ?? ''))} />,
      },
      { field: 'reporterGroup', headerName: 'Reporter Group', minWidth: 150, flex: 1 },
      { field: 'matchedCount', headerName: 'Matched', minWidth: 90, type: 'number' },
      { field: 'sentCount', headerName: 'Sent', minWidth: 90, type: 'number' },
      { field: 'failedCount', headerName: 'Failed', minWidth: 90, type: 'number' },
      {
        field: 'startedAt',
        headerName: 'Started',
        minWidth: 180,
        flex: 1,
        valueGetter: (value) => formatTimestamp(String(value ?? '')),
      },
      {
        field: 'finishedAt',
        headerName: 'Finished',
        minWidth: 180,
        flex: 1,
        valueGetter: (value) => formatTimestamp(String(value ?? '')),
      },
      { field: 'messageText', headerName: 'Message', minWidth: 260, flex: 1.5 },
      {
        field: 'lastError',
        headerName: 'Last Error',
        minWidth: 240,
        flex: 1.25,
        valueGetter: (value) => String(value ?? '').trim() || '-',
      },
    ],
    [],
  )

  function openDialog(reporter?: Reporter) {
    setEditing(reporter ?? null)
    setForm(toForm(reporter ?? null))
    const initialFacility =
      orgUnits.find((unit) => unit.id === reporter?.orgUnitId) ??
      (reporter?.orgUnitId
        ? {
            id: reporter.orgUnitId,
            name: getOrgUnitName(reporter.orgUnitId),
            parentId: reporter.districtId ?? null,
            hasChildren: false,
          }
        : null)
    setSelectedFacility(initialFacility)
    setFacilitySearchInput(initialFacility?.name ?? '')
    setFacilityOptions(initialFacility ? [initialFacility] : [])
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
    setSelectedFacility(null)
    setFacilitySearchInput('')
    setFacilityOptions([])
    setFacilityBrowserOpen(false)
    setFacilityBrowserTrail([])
    setFacilityBrowserItems([])
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

  function openJurisdictionDialog() {
    setJurisdictionDialogOpen(true)
    setJurisdictionForm(emptyJurisdictionBroadcastForm)
    setJurisdictionOrgUnitOptions(orgUnits.slice(0, 20))
    setJurisdictionOrgUnitSearch('')
    setError('')
  }

  function closeJurisdictionDialog() {
    if (jurisdictionSubmitting) {
      return
    }
    setJurisdictionDialogOpen(false)
    setJurisdictionForm(emptyJurisdictionBroadcastForm)
    setJurisdictionOrgUnitOptions([])
    setJurisdictionOrgUnitSearch('')
  }

  function openFacilityBrowser() {
    setFacilityBrowserOpen(true)
    void loadFacilityBrowserLevel([])
  }

  function handleFacilityBrowserNavigate(unit: OrgUnit) {
    const nextTrail = [...facilityBrowserTrail, { unit, label: unit.name }]
    void loadFacilityBrowserLevel(nextTrail)
  }

  function handleFacilityBrowserCrumb(index: number) {
    const nextTrail = facilityBrowserTrail.slice(0, index + 1)
    void loadFacilityBrowserLevel(nextTrail)
  }

  async function submitReporter() {
    setSubmitting(true)
    setError('')
    try {
      await apiRequest<Reporter>(editing ? `/reporters/${editing.id}` : '/reporters', {
        method: editing ? 'PUT' : 'POST',
        body: JSON.stringify({
          name: form.name.trim(),
          telephone: form.telephone.trim(),
          whatsapp: form.whatsapp.trim(),
          telegram: form.telegram.trim(),
          orgUnitId: Number(form.orgUnitId),
          isActive: form.isActive,
          groups: form.groups,
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
      await apiRequest<void>(`/reporters/${reporter.id}`, { method: 'DELETE' })
      await load()
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : 'Unable to delete reporter.')
    }
  }

  async function syncReporter(id: number) {
    setError('')
    try {
      await apiRequest(`/reporters/${id}/sync`, { method: 'POST', body: '{}' })
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
      await apiRequest('/reporters/bulk/sync', {
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
        await apiRequest(`/reporters/${messageDialog.reporter.id}/send-message`, {
          method: 'POST',
          body: JSON.stringify({ text: messageText.trim() }),
        })
      } else {
        await apiRequest('/reporters/bulk/broadcast', {
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

  async function submitJurisdictionBroadcast() {
    setJurisdictionSubmitting(true)
    setError('')
    try {
      const result = await apiRequest<JurisdictionBroadcastQueueResponse>('/reporters/broadcasts', {
        method: 'POST',
        body: JSON.stringify({
          orgUnitIds: jurisdictionForm.orgUnits.map((item) => item.id),
          reporterGroup: jurisdictionForm.reporterGroup,
          text: jurisdictionForm.text.trim(),
        }),
      })
      closeJurisdictionDialog()
      await loadBroadcastHistory()
      if (result.status === 'duplicate_pending') {
        notify.info(result.message)
        return
      }
      notify.success(`${result.message} ${result.broadcast.matchedCount} reporter${result.broadcast.matchedCount === 1 ? '' : 's'} matched.`)
    } catch (broadcastError) {
      const { error: normalized } = await handleAppError(broadcastError, {
        fallbackMessage: 'Unable to queue reporter broadcast.',
        notifyUser: false,
      })
      setError(formatActionError('Unable to queue reporter broadcast.', normalized))
    } finally {
      setJurisdictionSubmitting(false)
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
          <Button variant="outlined" size="small" startIcon={<SyncRoundedIcon />} onClick={() => void syncSelected()} disabled={selectedCount === 0}>
            Sync Selected
          </Button>
          <Button variant="outlined" size="small" startIcon={<CampaignRoundedIcon />} onClick={() => openMessageDialog('bulk')} disabled={selectedCount === 0}>
            Broadcast to Selected
          </Button>
          <Button variant="outlined" size="small" startIcon={<MessageRoundedIcon />} onClick={openJurisdictionDialog}>
            Send Message
          </Button>
          <Button variant="contained" size="small" startIcon={<PersonAddRoundedIcon />} onClick={() => openDialog()}>
            New Reporter
          </Button>
        </Stack>
      </Stack>
      {Boolean(currentUser?.isOrgUnitScopeRestricted && (currentUser.assignedOrgUnitIds?.length ?? 0) === 0) ? (
        <Alert severity="info" sx={{ mb: 2 }}>
          No org units are assigned to your account yet, so reporter search and creation are currently unavailable.
        </Alert>
      ) : null}

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
        pagination
        paginationModel={paginationModel}
        onPaginationModelChange={setPaginationModel}
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

      <Box sx={{ mt: 3 }}>
        <Stack spacing={0.75} sx={{ mb: 1.5 }}>
          <Typography variant="h5">Broadcast History</Typography>
          <Typography color="text.secondary">Recent queued background broadcasts from the top-level Send Message action.</Typography>
        </Stack>
        {broadcastHistoryError ? <Alert severity="error" sx={{ mb: 2 }}>{broadcastHistoryError}</Alert> : null}
        <DataGrid
          autoHeight
          rows={broadcastHistory}
          columns={broadcastColumns}
          loading={broadcastHistoryLoading}
          disableRowSelectionOnClick
          hideFooter
          sx={dataGridSx}
          slots={{
            noRowsOverlay: () => (
              <Stack alignItems="center" justifyContent="center" sx={{ p: 3, minHeight: 120 }}>
                <Typography color="text.secondary">No queued broadcasts yet.</Typography>
              </Stack>
            ),
          }}
        />
      </Box>

      <ReporterDetailsDialog
        open={Boolean(viewing)}
        reporter={viewing}
        facilityName={getOrgUnitName(viewing?.orgUnitId)}
        districtName={getOrgUnitName(viewing?.districtId)}
        onClose={() => setViewing(null)}
      />

      <RapidProDetailsDialog
        open={Boolean(rapidProReporter)}
        reporter={rapidProReporter}
        loading={rapidProDetailsLoading}
        error={rapidProDetailsError}
        details={rapidProDetails}
        onClose={() => {
          setRapidProReporter(null)
          setRapidProDetails(null)
          setRapidProDetailsError('')
          setRapidProDetailsLoading(false)
        }}
      />

      <ChatHistoryDialog
        open={Boolean(chatHistoryReporter)}
        reporter={chatHistoryReporter}
        loading={chatHistoryLoading}
        error={chatHistoryError}
        history={chatHistory}
        onClose={() => {
          setChatHistoryReporter(null)
          setChatHistory(null)
          setChatHistoryError('')
          setChatHistoryLoading(false)
        }}
      />

      <ReporterReportsDialog
        open={Boolean(reportsReporter)}
        reporter={reportsReporter}
        loading={recentReportsLoading}
        error={recentReportsError}
        reports={recentReports}
        selectedReportId={selectedReportId}
        onSelectReport={setSelectedReportId}
        onClose={() => {
          setReportsReporter(null)
          setRecentReports(null)
          setRecentReportsError('')
          setRecentReportsLoading(false)
          setSelectedReportId(null)
        }}
      />

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
            <Stack spacing={1.25} sx={{ gridColumn: '1 / -1' }}>
              <Autocomplete
                options={facilityOptions}
                value={selectedFacility}
                inputValue={facilitySearchInput}
                loading={facilitySearchLoading}
                filterOptions={(options) => options}
                onInputChange={(_event, value, reason) => {
                  if (reason === 'reset' && selectedFacility) {
                    setFacilitySearchInput(selectedFacility.name)
                    return
                  }
                  setFacilitySearchInput(value)
                  if (value.trim() === '') {
                    applyFacilitySelection(null)
                  }
                }}
                onChange={(_event, value) => applyFacilitySelection(value)}
                isOptionEqualToValue={(option, value) => option.id === value.id}
                getOptionLabel={(option) => option.name ?? ''}
                renderOption={(props, option) => (
                  <Box component="li" {...props}>
                    <Stack spacing={0.25}>
                      <Typography variant="body2">{option.name}</Typography>
                      {formatFacilityPath(option) ? (
                        <Typography variant="caption" color="text.secondary">
                          {formatFacilityPath(option)}
                        </Typography>
                      ) : null}
                    </Stack>
                  </Box>
                )}
                renderInput={(params) => (
                  <TextField
                    {...params}
                    label="Facility"
                    required
                    placeholder="Search facilities"
                    helperText="Search for a facility or browse the hierarchy."
                    InputProps={{
                      ...params.InputProps,
                      endAdornment: (
                        <>
                          {facilitySearchLoading ? <CircularProgress color="inherit" size={18} /> : null}
                          {params.InputProps.endAdornment}
                        </>
                      ),
                    }}
                  />
                )}
              />
              <Stack direction="row" spacing={1} alignItems="center">
                <Button variant="outlined" onClick={openFacilityBrowser}>
                  Browse hierarchy
                </Button>
                {selectedFacility ? (
                  <Typography variant="body2" color="text.secondary">
                    Selected: {selectedFacility.name}
                  </Typography>
                ) : (
                  <Typography variant="body2" color="text.secondary">
                    No facility selected.
                  </Typography>
                )}
              </Stack>
            </Stack>
            <Autocomplete
              multiple
              options={reporterGroupNames}
              value={form.groups}
              onChange={(_event, value) => setForm({ ...form, groups: value.map((item) => item.trim()).filter(Boolean) })}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Reporter Groups"
                  placeholder={reporterGroupNames.length > 0 ? 'Select predefined groups' : 'Create groups in Settings first'}
                  helperText={reporterGroupNames.length > 0 ? 'Reporter groups are managed from Settings.' : 'No active reporter groups available yet.'}
                />
              )}
              sx={{ gridColumn: '1 / -1' }}
            />
            <FormControlLabel
              control={<Checkbox checked={form.isActive} onChange={(event) => setForm({ ...form, isActive: event.target.checked })} />}
              label="Reporter is active"
            />
            {editing ? (
              <Box sx={{ gridColumn: '1 / -1', border: (theme) => `1px solid ${theme.palette.divider}`, borderRadius: 2, p: 2 }}>
                <Stack spacing={0.75}>
                  <Typography variant="subtitle2">Facility summary</Typography>
                  <Typography variant="body2">Facility: {selectedFacility?.name || getOrgUnitName(editing.orgUnitId) || '-'}</Typography>
                  <Typography variant="body2">District: {editing.districtId ? getOrgUnitName(editing.districtId) : 'Derived from hierarchy'}</Typography>
                  <Typography variant="body2">Reporting location: {editing.reportingLocation || 'Derived from facility'}</Typography>
                </Stack>
              </Box>
            ) : null}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeDialog} disabled={submitting}>Cancel</Button>
          <Button onClick={() => void submitReporter()} disabled={submitting || !form.name.trim() || !form.telephone.trim() || !form.orgUnitId} variant="contained">
            {editing ? 'Save' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={facilityBrowserOpen} onClose={() => setFacilityBrowserOpen(false)} fullWidth maxWidth="md">
        <DialogTitle>Browse Facility Hierarchy</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            <DialogContentText>Select a facility from the hierarchy. Parent nodes drill down; leaf nodes select the facility.</DialogContentText>
            <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
              <Button size="small" variant={facilityBrowserTrail.length === 0 ? 'contained' : 'outlined'} onClick={() => void loadFacilityBrowserLevel([])}>
                Root
              </Button>
              {facilityBrowserTrail.map((entry, index) => (
                <Button key={`${entry.label}-${index}`} size="small" variant={index === facilityBrowserTrail.length - 1 ? 'contained' : 'outlined'} onClick={() => handleFacilityBrowserCrumb(index)}>
                  {entry.label}
                </Button>
              ))}
            </Stack>
            <Divider />
            {facilityBrowserLoading ? <CircularProgress size={24} /> : null}
            {!facilityBrowserLoading ? (
              <List dense sx={{ py: 0 }}>
                {facilityBrowserItems.map((unit) => (
                  <ListItemButton
                    key={unit.id}
                    aria-label={`${unit.name} ${unit.hasChildren ? 'Browse children' : 'Select facility'}`}
                    onClick={() => {
                      if (unit.hasChildren) {
                        handleFacilityBrowserNavigate(unit)
                        return
                      }
                      applyFacilitySelection(unit)
                      setFacilityBrowserOpen(false)
                    }}
                  >
                    <ListItemText
                      primary={unit.name}
                      secondary={unit.hasChildren ? 'Browse children' : formatFacilityPath(unit) || 'Select facility'}
                    />
                  </ListItemButton>
                ))}
                {facilityBrowserItems.length === 0 ? <Typography color="text.secondary">No facilities found at this level.</Typography> : null}
              </List>
            ) : null}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setFacilityBrowserOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={Boolean(messageDialog)} onClose={closeMessageDialog} fullWidth maxWidth="sm">
        <DialogTitle>{messageDialog?.mode === 'single' ? 'Send Message' : 'Broadcast to Selected Reporters'}</DialogTitle>
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
            {messageDialog?.mode === 'single' ? 'Send Message' : 'Send Broadcast'}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={jurisdictionDialogOpen} onClose={closeJurisdictionDialog} fullWidth maxWidth="md">
        <DialogTitle>Send Message</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            <DialogContentText>
              Queue a background broadcast for reporters in the selected organisation units and reporter group. Selecting a non-leaf organisation unit includes all reporters below it.
            </DialogContentText>
            <Autocomplete
              multiple
              autoHighlight
              openOnFocus
              disablePortal
              options={jurisdictionOrgUnitOptions}
              value={jurisdictionForm.orgUnits}
              inputValue={jurisdictionOrgUnitSearch}
              loading={jurisdictionOrgUnitLoading}
              filterOptions={(options) => options}
              onInputChange={(_event, value) => setJurisdictionOrgUnitSearch(value)}
              onChange={(_event, value) => {
                setJurisdictionForm((current) => ({ ...current, orgUnits: uniqOrgUnits(value) }))
                setJurisdictionOrgUnitOptions((current) => uniqOrgUnits([...value, ...current]))
              }}
              isOptionEqualToValue={(option, value) => option.id === value.id}
              getOptionLabel={(option) => option.name ?? ''}
              renderOption={(props, option) => (
                <Box component="li" {...props}>
                  <Stack spacing={0.25}>
                    <Typography variant="body2">{option.name}</Typography>
                    {formatFacilityPath(option) ? (
                      <Typography variant="caption" color="text.secondary">
                        {formatFacilityPath(option)}
                      </Typography>
                    ) : null}
                  </Stack>
                </Box>
              )}
              renderInput={(params) => <TextField {...params} label="Organisation Units" placeholder="Search organisation units" required />}
            />
            <Autocomplete
              autoHighlight
              openOnFocus
              disablePortal
              options={reporterGroupOptions}
              value={reporterGroupOptions.find((item) => item.name === jurisdictionForm.reporterGroup) ?? null}
              onChange={(_event, value) => setJurisdictionForm((current) => ({ ...current, reporterGroup: value?.name ?? '' }))}
              isOptionEqualToValue={(option, value) => option.id === value.id}
              getOptionLabel={(option) => option.name ?? ''}
              renderInput={(params) => <TextField {...params} label="Reporter Group" required />}
            />
            <TextField
              label="Message"
              multiline
              minRows={4}
              value={jurisdictionForm.text}
              onChange={(event) => setJurisdictionForm((current) => ({ ...current, text: event.target.value }))}
              required
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeJurisdictionDialog} disabled={jurisdictionSubmitting}>
            Cancel
          </Button>
          <Button variant="contained" onClick={() => void submitJurisdictionBroadcast()} disabled={jurisdictionSubmitting}>
            {jurisdictionSubmitting ? 'Queueing...' : 'Queue Broadcast'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
