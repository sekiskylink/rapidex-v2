import React from 'react'
import {
  Alert,
  Autocomplete,
  Box,
  Button,
  Chip,
  CircularProgress,
  Divider,
  Drawer,
  InputAdornment,
  List,
  ListItemButton,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import type { PaginatedResponse } from '../lib/pagination'
import { apiRequest } from '../lib/api'
import { handleAppError } from '../errors/handleAppError'
import { useAppNotify } from '../notifications/facade'
import { CloseIcon, ReceiptLongRoundedIcon, SearchRoundedIcon } from '../ui/icons'
import { RequestDetailPage, type RequestDetailRecord } from '../pages/RequestDetailPage'
import type { EventRecord } from '../pages/traceability'

interface OrgUnitOption {
  id: number
  uid: string
  name: string
  displayPath?: string
  path?: string
}

interface RequestSummaryRow {
  id: number
  uid: string
  destinationServerName: string
  sourceSystem: string
  correlationId: string
  status: string
  createdAt: string
  payload?: unknown
  payloadBody: string
  extras: Record<string, unknown>
}

interface GlobalRequestFinderDrawerProps {
  open: boolean
  canReadRequests: boolean
  onClose: () => void
}

function useDebouncedValue(value: string, delayMs: number) {
  const [debounced, setDebounced] = React.useState(value)

  React.useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delayMs)
    return () => window.clearTimeout(timer)
  }, [delayMs, value])

  return debounced
}

function toDateTimeLocalValue(date: Date) {
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}

function defaultRange() {
  const to = new Date()
  const from = new Date(to.getTime() - 7 * 24 * 60 * 60 * 1000)
  return {
    from: toDateTimeLocalValue(from),
    to: toDateTimeLocalValue(to),
  }
}

function parseLocalDateTime(value: string) {
  if (!value) {
    return undefined
  }
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return undefined
  }
  return parsed.toISOString()
}

function formatDate(value: string) {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return value
  }
  return parsed.toLocaleString()
}

function formatPayloadPreview(row: RequestSummaryRow) {
  const raw = typeof row.payload === 'string' ? row.payload : row.payloadBody
  if (!raw) {
    return 'No payload preview'
  }
  const compact = String(raw).replace(/\s+/g, ' ').trim()
  if (compact.length <= 96) {
    return compact
  }
  return `${compact.slice(0, 93)}...`
}

function formatOrgUnitLabel(option: OrgUnitOption | null) {
  if (!option) {
    return ''
  }
  const path = option.displayPath || option.path
  return path ? `${option.name} (${path})` : option.name
}

function statusColor(status: string): 'default' | 'warning' | 'success' | 'error' | 'info' {
  switch (status) {
    case 'pending':
      return 'warning'
    case 'blocked':
      return 'warning'
    case 'processing':
      return 'info'
    case 'completed':
      return 'success'
    case 'failed':
      return 'error'
    default:
      return 'default'
  }
}

export function GlobalRequestFinderDrawer({ open, canReadRequests, onClose }: GlobalRequestFinderDrawerProps) {
  const notify = useAppNotify()
  const [msisdn, setMSISDN] = React.useState('')
  const [range, setRange] = React.useState(defaultRange)
  const [orgUnitQuery, setOrgUnitQuery] = React.useState('')
  const [orgUnitOptions, setOrgUnitOptions] = React.useState<OrgUnitOption[]>([])
  const [selectedOrgUnit, setSelectedOrgUnit] = React.useState<OrgUnitOption | null>(null)
  const [results, setResults] = React.useState<RequestSummaryRow[]>([])
  const [total, setTotal] = React.useState(0)
  const [loading, setLoading] = React.useState(false)
  const [loadingOrgUnits, setLoadingOrgUnits] = React.useState(false)
  const [errorMessage, setErrorMessage] = React.useState('')
  const [detailOpen, setDetailOpen] = React.useState(false)
  const [detailRequest, setDetailRequest] = React.useState<RequestDetailRecord | null>(null)
  const [detailEvents, setDetailEvents] = React.useState<EventRecord[]>([])
  const [detailError, setDetailError] = React.useState('')
  const requestIdRef = React.useRef(0)

  const debouncedMsisdn = useDebouncedValue(msisdn.trim(), 300)
  const debouncedOrgUnitQuery = useDebouncedValue(orgUnitQuery.trim(), 250)

  React.useEffect(() => {
    if (!open) {
      return
    }
    setDetailError('')
  }, [open])

  React.useEffect(() => {
    if (!open || !canReadRequests) {
      return
    }
    const requestId = ++requestIdRef.current
    const query = new URLSearchParams({
      page: '1',
      pageSize: '20',
      sort: 'createdAt:desc',
    })
    const from = parseLocalDateTime(range.from)
    const to = parseLocalDateTime(range.to)
    if (debouncedMsisdn) {
      query.set('msisdn', debouncedMsisdn)
    }
    if (selectedOrgUnit?.uid) {
      query.set('orgUnitUid', selectedOrgUnit.uid)
    }
    if (from) {
      query.set('from', from)
    }
    if (to) {
      query.set('to', to)
    }

    setLoading(true)
    setErrorMessage('')
    void apiRequest<PaginatedResponse<RequestSummaryRow>>(`/requests?${query.toString()}`)
      .then((response) => {
        if (requestId !== requestIdRef.current) {
          return
        }
        setResults(response.items ?? [])
        setTotal(response.totalCount ?? 0)
      })
      .catch(async (error) => {
        if (requestId !== requestIdRef.current) {
          return
        }
        setResults([])
        setTotal(0)
        setErrorMessage('Unable to load request matches.')
        await handleAppError(error, {
          fallbackMessage: 'Unable to load request matches.',
          notifier: notify,
        })
      })
      .finally(() => {
        if (requestId === requestIdRef.current) {
          setLoading(false)
        }
      })
  }, [canReadRequests, debouncedMsisdn, notify, open, range.from, range.to, selectedOrgUnit?.uid])

  React.useEffect(() => {
    if (!open || !canReadRequests) {
      return
    }
    const query = debouncedOrgUnitQuery
    setLoadingOrgUnits(true)
    void apiRequest<PaginatedResponse<OrgUnitOption>>(`/orgunits?page=0&pageSize=20&search=${encodeURIComponent(query)}`)
      .then((response) => {
        const next = response.items ?? []
        setOrgUnitOptions((current) => {
          const selected = selectedOrgUnit ? [selectedOrgUnit] : []
          const deduped = [...selected, ...next].filter(
            (item, index, items) => items.findIndex((candidate) => candidate.id === item.id) === index,
          )
          return deduped
        })
      })
      .catch(async (error) => {
        await handleAppError(error, {
          fallbackMessage: 'Unable to search org units.',
          notifier: notify,
        })
      })
      .finally(() => setLoadingOrgUnits(false))
  }, [canReadRequests, debouncedOrgUnitQuery, notify, open, selectedOrgUnit])

  const resetFilters = () => {
    setMSISDN('')
    setSelectedOrgUnit(null)
    setOrgUnitQuery('')
    setRange(defaultRange())
  }

  const openDetail = async (row: RequestSummaryRow) => {
    setDetailError('')
    try {
      const [detail, events] = await Promise.all([
        apiRequest<RequestDetailRecord>(`/requests/${row.id}`),
        apiRequest<PaginatedResponse<EventRecord>>(`/requests/${row.id}/events?page=1&pageSize=50&sort=createdAt:asc`),
      ])
      setDetailRequest(detail)
      setDetailEvents(events.items ?? [])
      setDetailOpen(true)
    } catch (error) {
      setDetailOpen(false)
      setDetailRequest(null)
      setDetailEvents([])
      setDetailError('Unable to load request detail.')
      await handleAppError(error, {
        fallbackMessage: 'Unable to load request detail.',
        notifier: notify,
      })
    }
  }

  if (!canReadRequests) {
    return null
  }

  return (
    <>
      <Drawer
        anchor="right"
        open={open}
        onClose={onClose}
        PaperProps={{
          sx: {
            width: { xs: '100%', sm: 460 },
          },
        }}
      >
        <Stack sx={{ height: '100%' }}>
          <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ px: 2, py: 1.5 }}>
            <Stack direction="row" spacing={1.25} alignItems="center">
              <ReceiptLongRoundedIcon fontSize="small" />
              <Box>
                <Typography variant="h6">Request Finder</Typography>
                <Typography variant="body2" color="text.secondary">
                  Search exchanges by MSISDN, org unit, and time range.
                </Typography>
              </Box>
            </Stack>
            <Button onClick={onClose} startIcon={<CloseIcon />}>
              Close
            </Button>
          </Stack>

          <Divider />

          <Stack spacing={2} sx={{ p: 2 }}>
            <TextField
              label="MSISDN"
              placeholder="+256700000001"
              value={msisdn}
              onChange={(event) => setMSISDN(event.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchRoundedIcon fontSize="small" />
                  </InputAdornment>
                ),
              }}
            />
            <Autocomplete
              options={orgUnitOptions}
              value={selectedOrgUnit}
              inputValue={orgUnitQuery}
              onInputChange={(_, value, reason) => {
                if (reason === 'reset') {
                  return
                }
                setOrgUnitQuery(value)
              }}
              onChange={(_, value) => setSelectedOrgUnit(value)}
              getOptionLabel={(option) => formatOrgUnitLabel(option)}
              isOptionEqualToValue={(option, value) => option.id === value.id}
              loading={loadingOrgUnits}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Org Unit"
                  placeholder="Search facilities or parent org units"
                  helperText="Matches the selected org unit and its descendants."
                  InputProps={{
                    ...params.InputProps,
                    endAdornment: (
                      <>
                        {loadingOrgUnits ? <CircularProgress size={18} color="inherit" /> : null}
                        {params.InputProps.endAdornment}
                      </>
                    ),
                  }}
                />
              )}
            />
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5}>
              <TextField
                label="From"
                type="datetime-local"
                value={range.from}
                onChange={(event) => setRange((current) => ({ ...current, from: event.target.value }))}
                InputLabelProps={{ shrink: true }}
                fullWidth
              />
              <TextField
                label="To"
                type="datetime-local"
                value={range.to}
                onChange={(event) => setRange((current) => ({ ...current, to: event.target.value }))}
                InputLabelProps={{ shrink: true }}
                fullWidth
              />
            </Stack>
            <Stack direction="row" justifyContent="space-between" alignItems="center">
              <Typography variant="caption" color="text.secondary">
                Default window: last 7 days
              </Typography>
              <Button size="small" onClick={resetFilters}>
                Reset
              </Button>
            </Stack>
          </Stack>

          <Divider />

          <Box sx={{ flex: 1, overflowY: 'auto' }}>
            <Stack spacing={1} sx={{ p: 2 }}>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Typography variant="subtitle2">Matches</Typography>
                <Chip label={loading ? 'Loading...' : `${total} found`} size="small" variant="outlined" />
              </Stack>
              {errorMessage ? <Alert severity="error">{errorMessage}</Alert> : null}
              {detailError ? <Alert severity="error">{detailError}</Alert> : null}
              {!loading && results.length === 0 ? (
                <Alert severity="info">No matching requests were found in the selected time window.</Alert>
              ) : null}
            </Stack>
            <List disablePadding>
              {results.map((row) => (
                <React.Fragment key={row.id}>
                  <ListItemButton alignItems="flex-start" onClick={() => void openDetail(row)} sx={{ px: 2, py: 1.5 }}>
                    <Stack spacing={1} sx={{ width: '100%' }}>
                      <Stack direction="row" justifyContent="space-between" spacing={1} alignItems="center">
                        <Typography variant="body2" sx={{ fontWeight: 700 }}>
                          {row.uid}
                        </Typography>
                        <Chip label={row.status} color={statusColor(row.status)} size="small" />
                      </Stack>
                      <Typography variant="body2" color="text.secondary">
                        {row.destinationServerName || row.sourceSystem || 'Unknown source'}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {formatDate(row.createdAt)}
                      </Typography>
                      <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
                        {row.extras?.msisdn ? <Chip label={String(row.extras.msisdn)} size="small" variant="outlined" /> : null}
                        {row.extras?.mappedOrgUnit ? (
                          <Chip label={`Org Unit ${String(row.extras.mappedOrgUnit)}`} size="small" variant="outlined" />
                        ) : row.extras?.orgUnit ? (
                          <Chip label={`Org Unit ${String(row.extras.orgUnit)}`} size="small" variant="outlined" />
                        ) : null}
                      </Stack>
                      <Typography variant="body2">{formatPayloadPreview(row)}</Typography>
                    </Stack>
                  </ListItemButton>
                  <Divider component="li" />
                </React.Fragment>
              ))}
            </List>
          </Box>
        </Stack>
      </Drawer>

      <RequestDetailPage
        open={detailOpen}
        request={detailRequest}
        events={detailEvents}
        onClose={() => {
          setDetailOpen(false)
          setDetailRequest(null)
          setDetailEvents([])
        }}
      />
    </>
  )
}
