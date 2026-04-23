import React from 'react'
import { Alert, Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, Divider, Stack, Typography } from '@mui/material'

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
  return (
    <Stack spacing={0.5} sx={{ minWidth: 0 }}>
      <Typography variant="caption" color="text.secondary">
        {label}
      </Typography>
      <Typography variant="body2" sx={{ wordBreak: 'break-word' }}>
        {value || '-'}
      </Typography>
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
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <DialogTitle>Reporter Chat History</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
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
            <Stack spacing={1.5}>
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
                        maxWidth: '82%',
                        minWidth: { xs: '100%', sm: '55%' },
                        px: 2,
                        py: 1.5,
                        borderRadius: 3,
                        bgcolor: outgoing ? 'primary.main' : 'background.default',
                        color: outgoing ? 'primary.contrastText' : 'text.primary',
                        boxShadow: 1,
                      }}
                    >
                      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} justifyContent="space-between" alignItems={{ xs: 'flex-start', sm: 'center' }}>
                        <Stack direction="row" spacing={1} alignItems="center">
                          <Chip
                            label={outgoing ? 'Outgoing' : 'Incoming'}
                            size="small"
                            color={outgoing ? 'primary' : 'default'}
                            variant={outgoing ? 'filled' : 'outlined'}
                          />
                          <Chip label={item.status || 'unknown'} size="small" color={chipColor(item.status || '')} variant="outlined" />
                        </Stack>
                        <Typography variant="caption" color={outgoing ? 'inherit' : 'text.secondary'}>
                          {formatDateTime(item.sentOn || item.createdOn || item.modifiedOn)}
                        </Typography>
                      </Stack>
                      <Typography sx={{ mt: 1.25, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                        {item.text || 'No message text'}
                      </Typography>
                      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} sx={{ mt: 1.5 }}>
                        {item.urn ? <Chip label={item.urn} size="small" variant="outlined" /> : null}
                        {item.channel?.name ? <Chip label={item.channel.name} size="small" variant="outlined" /> : null}
                        {item.flow?.name ? <Chip label={item.flow.name} size="small" variant="outlined" /> : null}
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
