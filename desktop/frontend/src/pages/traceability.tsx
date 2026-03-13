import React from 'react'
import { Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, Stack, Typography } from '@mui/material'

export interface EventRecord {
  id: number
  uid: string
  requestId?: number | null
  requestUid: string
  deliveryAttemptId?: number | null
  deliveryUid: string
  asyncTaskId?: number | null
  asyncTaskUid: string
  workerRunId?: number | null
  workerRunUid: string
  eventType: string
  eventLevel: string
  eventData?: Record<string, unknown>
  eventDataPreview: string
  message: string
  correlationId: string
  actorType: string
  actorUserId?: number | null
  actorName: string
  sourceComponent: string
  createdAt: string
}

export interface TraceReference {
  id: number
  uid: string
  createdAt: string
}

export interface TraceResult {
  correlationId: string
  summary: {
    requests: TraceReference[]
    deliveries: TraceReference[]
    jobs: TraceReference[]
    workers: TraceReference[]
  }
  events: EventRecord[]
}

export function formatTraceDate(value?: string | null) {
  if (!value) {
    return '-'
  }
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return value
  }
  return parsed.toLocaleString()
}

export function traceLevelColor(level: string): 'default' | 'info' | 'warning' | 'error' {
  switch (level) {
    case 'error':
      return 'error'
    case 'warning':
      return 'warning'
    case 'info':
      return 'info'
    default:
      return 'default'
  }
}

export function EventTimeline({ events, emptyMessage = 'No events recorded yet.' }: { events: EventRecord[]; emptyMessage?: string }) {
  return (
    <Stack spacing={1}>
      {events.length === 0 ? <Typography variant="body2">{emptyMessage}</Typography> : null}
      {events.map((event) => (
        <Box key={event.id} sx={{ p: 1.5, borderRadius: 2, border: '1px solid', borderColor: 'divider' }}>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5} alignItems={{ md: 'center' }}>
            <Typography variant="subtitle2">{event.eventType}</Typography>
            <Chip size="small" label={event.eventLevel} color={traceLevelColor(event.eventLevel)} />
            <Typography variant="caption" color="text.secondary">
              {formatTraceDate(event.createdAt)}
            </Typography>
          </Stack>
          <Typography variant="body2" sx={{ mt: 1 }}>
            {event.message || event.eventDataPreview || 'No message'}
          </Typography>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} sx={{ mt: 1 }}>
            <Typography variant="caption" color="text.secondary">
              Source: {event.sourceComponent || '-'}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Actor: {event.actorName || event.actorType || '-'}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Correlation: {event.correlationId || '-'}
            </Typography>
          </Stack>
        </Box>
      ))}
    </Stack>
  )
}

export function EventDetailDialog({
  open,
  event,
  onClose,
}: {
  open: boolean
  event: EventRecord | null
  onClose: () => void
}) {
  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>Event Detail</DialogTitle>
      <DialogContent>
        {event ? (
          <Stack spacing={2} sx={{ pt: 1 }}>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5} alignItems={{ md: 'center' }}>
              <Typography variant="h6">{event.eventType}</Typography>
              <Chip size="small" label={event.eventLevel} color={traceLevelColor(event.eventLevel)} />
            </Stack>
            <Typography variant="body2">{event.message || 'No message'}</Typography>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              <Typography variant="caption" color="text.secondary">
                Correlation ID: {event.correlationId || '-'}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Source: {event.sourceComponent || '-'}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Created: {formatTraceDate(event.createdAt)}
              </Typography>
            </Stack>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              <Typography variant="caption" color="text.secondary">
                Request: {event.requestUid || '-'}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Delivery: {event.deliveryUid || '-'}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Job: {event.asyncTaskUid || '-'}
              </Typography>
            </Stack>
            <Box component="pre" sx={{ m: 0, p: 2, borderRadius: 2, bgcolor: 'background.default', overflowX: 'auto' }}>
              {JSON.stringify(event.eventData ?? {}, null, 2)}
            </Box>
          </Stack>
        ) : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  )
}
