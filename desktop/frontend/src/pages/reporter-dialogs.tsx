import React from 'react'
import { Alert, Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, Divider, Stack, Typography } from '@mui/material'
import { alpha } from '@mui/material/styles'

interface ReporterLike {
  id: number
  uid: string
  name: string
  telephone: string
  whatsapp: string
  telegram: string
  reportingLocation: string
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

interface OrgUnitLike {
  id: number
  uid: string
  code: string
  name: string
  shortName: string
  description: string
  hierarchyLevel: number
  path: string
  displayPath?: string
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

export interface RapidProContactDetailsResponse {
  reporter: ReporterLike
  found: boolean
  contact?: {
    uuid: string
    name: string
    status: string
    language: string
    urns: string[]
    groups: Array<{ uuid: string; name: string }>
    fields: Record<string, string>
    flow?: { uuid: string; name: string } | null
    createdOn: string
    modifiedOn: string
    lastSeenOn: string
  } | null
}

export interface RapidProMessageHistoryResponse {
  reporter: ReporterLike
  found: boolean
  items: Array<{
    id: number
    broadcastId?: number | null
    direction: string
    type: string
    status: string
    visibility: string
    text: string
    urn: string
    channel?: { uuid: string; name: string } | null
    flow?: { uuid: string; name: string } | null
    createdOn: string
    sentOn: string
    modifiedOn: string
  }>
  next?: string
}

export interface ReporterReportsResponse {
  reporter: ReporterLike
  items: Array<{
    id: number
    uid: string
    status: string
    createdAt: string
    payloadBody: string
    payloadPreview: string
  }>
}

function formatDateTime(value?: string | null) {
  if (!value) {
    return '-'
  }
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return value
  }
  return parsed.toLocaleString()
}

function formatPrettyJSON(value?: string | null) {
  if (!value) {
    return ''
  }
  try {
    return JSON.stringify(JSON.parse(value), null, 2)
  } catch {
    return value
  }
}

function chipColor(status: string): 'default' | 'success' | 'warning' | 'error' | 'info' {
  switch (status.toLowerCase()) {
    case 'active':
    case 'sent':
    case 'delivered':
    case 'read':
    case 'synced':
      return 'success'
    case 'pending':
    case 'queued':
    case 'wired':
    case 'handled':
      return 'info'
    case 'failed':
    case 'errored':
    case 'stopped':
    case 'archived':
      return 'error'
    default:
      return 'default'
  }
}

function renderDetail(label: string, value: React.ReactNode) {
  const isPlainValue = typeof value === 'string' || typeof value === 'number'
  return (
    <Stack spacing={0.5} sx={{ minWidth: 0 }}>
      <Typography variant="caption" color="text.secondary">
        {label}
      </Typography>
      {isPlainValue ? (
        <Typography variant="body2" sx={{ wordBreak: 'break-word' }}>
          {value || '-'}
        </Typography>
      ) : (
        <Box sx={{ minWidth: 0 }}>{value || '-'}</Box>
      )}
    </Stack>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Stack spacing={1.5}>
      <Typography variant="subtitle2">{title}</Typography>
      {children}
    </Stack>
  )
}

function renderHistoryBadge(label: string) {
  return (
    <Box
      component="span"
      sx={{
        display: 'inline-flex',
        alignItems: 'center',
        px: 1.1,
        py: 0.4,
        borderRadius: 999,
        bgcolor: 'rgba(15, 23, 42, 0.08)',
        color: '#334155',
        fontSize: '0.75rem',
        fontWeight: 600,
        lineHeight: 1.2,
      }}
    >
      {label}
    </Box>
  )
}

interface ReporterDetailsDialogProps {
  open: boolean
  reporter: ReporterLike | null
  facilityName: string
  districtName: string
  onClose: () => void
}

export function ReporterDetailsDialog({ open, reporter, facilityName, districtName, onClose }: ReporterDetailsDialogProps) {
  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <DialogTitle>Reporter Details</DialogTitle>
      <DialogContent>
        {reporter ? (
          <Stack spacing={2} sx={{ pt: 1 }}>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={1} alignItems={{ xs: 'flex-start', md: 'center' }}>
              <Typography variant="h6">{reporter.name}</Typography>
              <Chip label={reporter.synced && reporter.rapidProUuid ? 'RapidPro synced' : 'Sync pending'} color={reporter.synced && reporter.rapidProUuid ? 'success' : 'default'} size="small" />
              <Chip label={reporter.isActive ? 'Active' : 'Inactive'} color={reporter.isActive ? 'success' : 'default'} size="small" variant="outlined" />
              {facilityName ? <Chip label={facilityName} size="small" variant="outlined" /> : null}
            </Stack>
            <Section title="Identity">
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderDetail('Reporter UID', reporter.uid)}
                {renderDetail('RapidPro UUID', reporter.rapidProUuid || 'Not synced yet')}
                {renderDetail('Reporter Groups', reporter.groups.length > 0 ? reporter.groups.join(', ') : '-')}
              </Stack>
            </Section>
            <Divider />
            <Section title="Contact Channels">
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderDetail('Telephone', reporter.telephone)}
                {renderDetail('WhatsApp', reporter.whatsapp || '-')}
                {renderDetail('Telegram', reporter.telegram || '-')}
              </Stack>
            </Section>
            <Divider />
            <Section title="Reporting Context">
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderDetail('Facility', facilityName || '-')}
                {renderDetail('District', districtName || '-')}
                {renderDetail('Reporting Location', reporter.reportingLocation || '-')}
              </Stack>
            </Section>
            <Divider />
            <Section title="Activity">
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderDetail('Total Reports', String(reporter.totalReports ?? 0))}
                {renderDetail('Last Reporting Date', formatDateTime(reporter.lastReportingDate))}
                {renderDetail('Last Login', formatDateTime(reporter.lastLoginAt))}
              </Stack>
            </Section>
            <Divider />
            <Section title="System Metadata">
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderDetail('SMS Code', reporter.smsCode || '-')}
                {renderDetail('SMS Code Expires', formatDateTime(reporter.smsCodeExpiresAt))}
                {renderDetail('MT UUID', reporter.mtuuid || '-')}
              </Stack>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderDetail('Created', formatDateTime(reporter.createdAt))}
                {renderDetail('Updated', formatDateTime(reporter.updatedAt))}
              </Stack>
            </Section>
          </Stack>
        ) : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  )
}

interface OrgUnitDetailsDialogProps {
  open: boolean
  orgUnit: OrgUnitLike | null
  parentName: string
  onClose: () => void
}

export function OrgUnitDetailsDialog({ open, orgUnit, parentName, onClose }: OrgUnitDetailsDialogProps) {
  const formattedDisplayPath = orgUnit?.displayPath || '-'
  const formattedPath = orgUnit?.path || '-'
  const extras = formatPrettyJSON(JSON.stringify(orgUnit?.extras ?? {}, null, 2)) || '{}'
  const attributeValues = formatPrettyJSON(JSON.stringify(orgUnit?.attributeValues ?? {}, null, 2)) || '{}'

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <DialogTitle>Facility Details</DialogTitle>
      <DialogContent>
        {orgUnit ? (
          <Stack spacing={2} sx={{ pt: 1 }}>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={1} alignItems={{ xs: 'flex-start', md: 'center' }}>
              <Typography variant="h6">{orgUnit.name}</Typography>
              <Chip label={orgUnit.deleted ? 'Deleted' : 'Active'} color={orgUnit.deleted ? 'warning' : 'success'} size="small" />
              <Chip label={`Level ${orgUnit.hierarchyLevel}`} size="small" variant="outlined" />
              {parentName ? <Chip label={`Parent: ${parentName}`} size="small" variant="outlined" /> : null}
            </Stack>
            <Section title="Overview">
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderDetail('Name', orgUnit.name)}
                {renderDetail('Short Name', orgUnit.shortName || '-')}
                {renderDetail('Code', orgUnit.code || '-')}
              </Stack>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderDetail('UID', orgUnit.uid)}
                {renderDetail('Opening Date', formatDateTime(orgUnit.openingDate))}
                {renderDetail('Last Sync Date', formatDateTime(orgUnit.lastSyncDate))}
              </Stack>
            </Section>
            <Divider />
            <Section title="Hierarchy">
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderDetail('Parent', parentName || '-')}
                {renderDetail('Hierarchy Level', String(orgUnit.hierarchyLevel))}
                {renderDetail('Display Path', formattedDisplayPath)}
              </Stack>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderDetail('UID Path', formattedPath)}
              </Stack>
            </Section>
            <Divider />
            <Section title="Contacts">
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderDetail('Phone Number', orgUnit.phoneNumber || '-')}
                {renderDetail('Email', orgUnit.email || '-')}
                {renderDetail('Website URL', orgUnit.url || '-')}
              </Stack>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderDetail('Address', orgUnit.address || '-')}
              </Stack>
            </Section>
            <Divider />
            <Section title="Metadata">
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderDetail('Description', orgUnit.description || '-')}
              </Stack>
            </Section>
            <Divider />
            <Section title="Extended Data">
              <Stack spacing={1}>
                {renderDetail(
                  'Extras JSON',
                  <Box
                    component="pre"
                    sx={{
                      m: 0,
                      p: 1.5,
                      borderRadius: 1.5,
                      bgcolor: alpha('#0f172a', 0.04),
                      overflowX: 'auto',
                      whiteSpace: 'pre-wrap',
                      fontFamily: 'monospace',
                      fontSize: '0.8rem',
                    }}
                  >
                    {extras}
                  </Box>,
                )}
                {renderDetail(
                  'Attribute Values JSON',
                  <Box
                    component="pre"
                    sx={{
                      m: 0,
                      p: 1.5,
                      borderRadius: 1.5,
                      bgcolor: alpha('#0f172a', 0.04),
                      overflowX: 'auto',
                      whiteSpace: 'pre-wrap',
                      fontFamily: 'monospace',
                      fontSize: '0.8rem',
                    }}
                  >
                    {attributeValues}
                  </Box>,
                )}
              </Stack>
            </Section>
          </Stack>
        ) : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  )
}

interface RapidProDetailsDialogProps {
  open: boolean
  reporter: ReporterLike | null
  loading: boolean
  error: string
  details: RapidProContactDetailsResponse | null
  onClose: () => void
}

export function RapidProDetailsDialog({ open, reporter, loading, error, details, onClose }: RapidProDetailsDialogProps) {
  const contact = details?.contact ?? null
  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <DialogTitle>RapidPro Contact Details</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          {reporter ? (
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={1} alignItems={{ xs: 'flex-start', md: 'center' }}>
              <Typography variant="h6">{reporter.name}</Typography>
              <Chip label={reporter.rapidProUuid ? reporter.rapidProUuid : 'No local RapidPro UUID'} size="small" variant="outlined" />
            </Stack>
          ) : null}
          {loading ? <Typography color="text.secondary">Loading RapidPro contact details...</Typography> : null}
          {error ? <Alert severity="error">{error}</Alert> : null}
          {!loading && !error && details && !details.found ? (
            <Alert severity="info">No synced RapidPro contact was found for this reporter.</Alert>
          ) : null}
          {!loading && !error && contact ? (
            <Stack spacing={2}>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={1} alignItems={{ xs: 'flex-start', md: 'center' }}>
                <Typography variant="subtitle1">{contact.name || 'Unnamed contact'}</Typography>
                <Chip label={contact.status || 'unknown'} color={chipColor(contact.status || '')} size="small" />
                {contact.language ? <Chip label={`Language: ${contact.language}`} size="small" variant="outlined" /> : null}
              </Stack>
              <Section title="Contact">
                <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                  {renderDetail('UUID', contact.uuid)}
                  {renderDetail('URNs', contact.urns.length > 0 ? contact.urns.join(', ') : '-')}
                  {renderDetail('Groups', contact.groups.length > 0 ? contact.groups.map((group) => group.name).join(', ') : '-')}
                </Stack>
              </Section>
              <Divider />
              <Section title="Flow State">
                <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                  {renderDetail('Current Flow', contact.flow ? `${contact.flow.name} (${contact.flow.uuid})` : 'No active flow')}
                  {renderDetail('Last Seen', formatDateTime(contact.lastSeenOn))}
                </Stack>
              </Section>
              <Divider />
              <Section title="Custom Fields">
                {Object.keys(contact.fields ?? {}).length > 0 ? (
                  <Box
                    sx={{
                      display: 'grid',
                      gap: 2,
                      gridTemplateColumns: { xs: '1fr', md: 'repeat(2, minmax(0, 1fr))' },
                    }}
                  >
                    {Object.entries(contact.fields)
                      .sort(([left], [right]) => left.localeCompare(right))
                      .map(([key, value]) => (
                        <Box key={key}>{renderDetail(key, value || '-')}</Box>
                      ))}
                  </Box>
                ) : (
                  <Typography color="text.secondary">This contact has no custom fields.</Typography>
                )}
              </Section>
              <Divider />
              <Section title="Lifecycle">
                <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                  {renderDetail('Created', formatDateTime(contact.createdOn))}
                  {renderDetail('Modified', formatDateTime(contact.modifiedOn))}
                </Stack>
              </Section>
            </Stack>
          ) : null}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  )
}

interface ChatHistoryDialogProps {
  open: boolean
  reporter: ReporterLike | null
  loading: boolean
  error: string
  history: RapidProMessageHistoryResponse | null
  onClose: () => void
}

export function ChatHistoryDialog({ open, reporter, loading, error, history, onClose }: ChatHistoryDialogProps) {
  const items = history?.items ?? []
  return (
    <Dialog
      open={open}
      onClose={onClose}
      fullWidth
      maxWidth="lg"
      PaperProps={{
        sx: {
          width: { xs: 'calc(100vw - 16px)', sm: 'min(1100px, calc(100vw - 40px))' },
          maxWidth: 'none',
          height: { xs: 'calc(100vh - 16px)', sm: 'min(88vh, 940px)' },
        },
      }}
    >
      <DialogTitle>Reporter Chat History</DialogTitle>
      <DialogContent
        dividers
        sx={{
          display: 'flex',
          flexDirection: 'column',
          minHeight: 0,
          px: { xs: 2, sm: 3 },
          py: 2,
        }}
      >
        <Stack spacing={2} sx={{ flex: 1, minHeight: 0 }}>
          {reporter ? (
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={1} alignItems={{ xs: 'flex-start', md: 'center' }}>
              <Typography variant="h6">{reporter.name}</Typography>
              <Chip label={reporter.telephone || 'No telephone'} size="small" variant="outlined" />
              {history?.found ? <Chip label="RapidPro conversation found" size="small" color="success" /> : null}
            </Stack>
          ) : null}
          {loading ? <Typography color="text.secondary">Loading SMS history...</Typography> : null}
          {error ? <Alert severity="error">{error}</Alert> : null}
          {!loading && !error && history && !history.found ? (
            <Alert severity="info">This reporter is not currently linked to a RapidPro contact, so no conversation history is available.</Alert>
          ) : null}
          {!loading && !error && history?.found && items.length === 0 ? (
            <Alert severity="info">No SMS history was returned for this reporter.</Alert>
          ) : null}
          {!loading && !error && items.length > 0 ? (
            <Stack
              spacing={1.75}
              sx={{
                flex: 1,
                minHeight: 0,
                overflowY: 'auto',
                pr: { xs: 0.5, sm: 1 },
                pb: 0.5,
              }}
            >
              {items.map((item) => {
                const outgoing = item.direction.toLowerCase().startsWith('out')
                return (
                  <Box
                    key={item.id}
                    sx={{
                      display: 'flex',
                      justifyContent: outgoing ? 'flex-end' : 'flex-start',
                    }}
                  >
                    <Box
                      sx={{
                        width: { xs: '100%', sm: 'min(78%, 760px)' },
                        px: { xs: 1.75, sm: 2.25 },
                        py: 1.75,
                        borderRadius: 3,
                        bgcolor: outgoing ? '#e8f1ff' : '#f8fafc',
                        color: '#122033',
                        border: '1px solid',
                        borderColor: outgoing ? '#c7dafc' : '#dbe2ea',
                        boxShadow: outgoing ? `0 10px 24px ${alpha('#8fb4f6', 0.18)}` : `0 8px 22px ${alpha('#0f172a', 0.08)}`,
                      }}
                    >
                      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} justifyContent="space-between" alignItems={{ xs: 'flex-start', sm: 'center' }}>
                        <Stack direction="row" spacing={0.75} alignItems="center" useFlexGap flexWrap="wrap">
                          {renderHistoryBadge(outgoing ? 'Outgoing' : 'Incoming')}
                          {renderHistoryBadge(item.status || 'unknown')}
                        </Stack>
                        <Typography variant="caption" sx={{ color: '#475569' }}>
                          {formatDateTime(item.sentOn || item.createdOn || item.modifiedOn)}
                        </Typography>
                      </Stack>
                      <Typography sx={{ mt: 1.25, whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontSize: '0.98rem', lineHeight: 1.6 }}>
                        {item.text || 'No message text'}
                      </Typography>
                      <Stack direction="row" spacing={0.75} useFlexGap flexWrap="wrap" sx={{ mt: 1.5 }}>
                        {item.urn ? renderHistoryBadge(item.urn) : null}
                        {item.channel?.name ? renderHistoryBadge(item.channel.name) : null}
                        {item.flow?.name ? renderHistoryBadge(item.flow.name) : null}
                      </Stack>
                    </Box>
                  </Box>
                )
              })}
              {history?.next ? (
                <Typography variant="caption" color="text.secondary">
                  More RapidPro history is available but not loaded in this dialog yet.
                </Typography>
              ) : null}
            </Stack>
          ) : null}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  )
}

interface ReporterReportsDialogProps {
  open: boolean
  reporter: ReporterLike | null
  loading: boolean
  error: string
  reports: ReporterReportsResponse | null
  selectedReportId: number | null
  onSelectReport: (reportId: number) => void
  onClose: () => void
}

export function ReporterReportsDialog({
  open,
  reporter,
  loading,
  error,
  reports,
  selectedReportId,
  onSelectReport,
  onClose,
}: ReporterReportsDialogProps) {
  const items = reports?.items ?? []
  const selectedReport = items.find((item) => item.id === selectedReportId) ?? items[0] ?? null

  return (
    <Dialog
      open={open}
      onClose={onClose}
      fullWidth
      maxWidth="lg"
      PaperProps={{
        sx: {
          width: { xs: 'calc(100vw - 16px)', sm: 'min(1100px, calc(100vw - 40px))' },
          maxWidth: 'none',
          height: { xs: 'calc(100vh - 16px)', sm: 'min(84vh, 860px)' },
        },
      }}
    >
      <DialogTitle>Reporter Reports</DialogTitle>
      <DialogContent
        dividers
        sx={{
          display: 'flex',
          flexDirection: 'column',
          minHeight: 0,
          px: { xs: 2, sm: 3 },
          py: 2,
        }}
      >
        <Stack spacing={2} sx={{ flex: 1, minHeight: 0 }}>
          {reporter ? (
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={1} alignItems={{ xs: 'flex-start', md: 'center' }}>
              <Typography variant="h6">{reporter.name}</Typography>
              <Chip label={reporter.telephone || 'No telephone'} size="small" variant="outlined" />
            </Stack>
          ) : null}
          {loading ? <Typography color="text.secondary">Loading recent reports...</Typography> : null}
          {error ? <Alert severity="error">{error}</Alert> : null}
          {!loading && !error && items.length === 0 ? <Alert severity="info">No recent reports found for this reporter.</Alert> : null}
          {!loading && !error && items.length > 0 ? (
            <Box
              sx={{
                display: 'grid',
                gridTemplateColumns: { xs: '1fr', md: 'minmax(320px, 0.95fr) minmax(0, 1.35fr)' },
                gap: 2,
                flex: 1,
                minHeight: 0,
              }}
            >
              <Stack spacing={1.25} sx={{ minHeight: 0, overflowY: 'auto', pr: { md: 1 } }}>
                {items.map((item) => {
                  const active = item.id === selectedReport?.id
                  return (
                    <Box
                      key={item.id}
                      sx={{
                        p: 1.5,
                        borderRadius: 2,
                        border: '1px solid',
                        borderColor: active ? 'primary.main' : 'divider',
                        bgcolor: active ? alpha('#2563eb', 0.06) : 'background.paper',
                      }}
                    >
                      <Stack spacing={1}>
                        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} justifyContent="space-between" alignItems={{ xs: 'flex-start', sm: 'center' }}>
                          <Typography variant="body2" sx={{ fontWeight: 600 }}>
                            {formatDateTime(item.createdAt)}
                          </Typography>
                          <Chip label={item.status || 'unknown'} color={chipColor(item.status || '')} size="small" />
                        </Stack>
                        <Button
                          variant="text"
                          size="small"
                          sx={{ p: 0, minWidth: 0, justifyContent: 'flex-start', textTransform: 'none', fontWeight: 600 }}
                          onClick={() => onSelectReport(item.id)}
                        >
                          {item.payloadPreview || '(empty payload)'}
                        </Button>
                      </Stack>
                    </Box>
                  )
                })}
              </Stack>
              <Box
                sx={{
                  minHeight: 0,
                  display: 'flex',
                  flexDirection: 'column',
                  borderRadius: 2,
                  border: '1px solid',
                  borderColor: 'divider',
                  bgcolor: 'background.default',
                }}
              >
                <Box sx={{ px: 2, py: 1.5, borderBottom: '1px solid', borderColor: 'divider' }}>
                  <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} justifyContent="space-between">
                    <Typography variant="subtitle2">Payload</Typography>
                    {selectedReport ? <Typography variant="caption" color="text.secondary">{selectedReport.uid}</Typography> : null}
                  </Stack>
                </Box>
                <Box component="pre" sx={{ m: 0, p: 2, flex: 1, minHeight: 0, overflow: 'auto', whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                  {selectedReport ? formatPrettyJSON(selectedReport.payloadBody) : 'Select a report to inspect its payload.'}
                </Box>
              </Box>
            </Box>
          ) : null}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  )
}
