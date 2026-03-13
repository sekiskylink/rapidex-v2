import React from 'react'
import { Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, Divider, Stack, Typography } from '@mui/material'
import { EventTimeline, type EventRecord } from './traceability'

export interface RequestDetailRecord {
  id: number
  uid: string
  sourceSystem: string
  destinationServerId: number
  destinationServerName: string
  batchId: string
  correlationId: string
  idempotencyKey: string
  payloadBody: string
  payloadFormat: string
  payload: unknown
  urlSuffix: string
  status: string
  extras: Record<string, unknown>
  createdAt: string
  updatedAt: string
  createdBy?: number | null
  latestDeliveryId?: number | null
  latestDeliveryUid: string
  latestDeliveryStatus: string
  latestAsyncTaskId?: number | null
  latestAsyncTaskUid: string
  latestAsyncState: string
  latestAsyncRemoteJobId: string
  latestAsyncPollUrl: string
  awaitingAsync: boolean
}

function formatDate(value: string) {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return value
  }
  return parsed.toLocaleString()
}

function formatJSON(value: unknown) {
  try {
    if (typeof value === 'string') {
      return JSON.stringify(JSON.parse(value), null, 2)
    }
    return JSON.stringify(value ?? {}, null, 2)
  } catch {
    return String(value ?? '')
  }
}

function renderMetadata(label: string, value: React.ReactNode) {
  return (
    <Stack spacing={0.5}>
      <Typography variant="caption" color="text.secondary">
        {label}
      </Typography>
      <Typography variant="body2">{value || '-'}</Typography>
    </Stack>
  )
}

function statusColor(status: string): 'default' | 'warning' | 'success' | 'error' | 'info' {
  switch (status) {
    case 'pending':
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

interface RequestDetailPageProps {
  open: boolean
  request: RequestDetailRecord | null
  events: EventRecord[]
  onClose: () => void
}

export function RequestDetailPage({ open, request, events, onClose }: RequestDetailPageProps) {
  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>Request Detail</DialogTitle>
      <DialogContent>
        {request ? (
          <Stack spacing={2} sx={{ pt: 1 }}>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems={{ xs: 'flex-start', md: 'center' }}>
              <Typography variant="h6">{request.uid}</Typography>
              <Chip label={request.status} color={statusColor(request.status)} size="small" />
              {request.awaitingAsync ? <Chip label="Awaiting async" color="info" size="small" /> : null}
            </Stack>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              {renderMetadata('Destination Server', request.destinationServerName)}
              {renderMetadata('Source System', request.sourceSystem)}
              {renderMetadata('Correlation ID', request.correlationId)}
            </Stack>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              {renderMetadata('Batch ID', request.batchId)}
              {renderMetadata('Idempotency Key', request.idempotencyKey)}
              {renderMetadata('URL Suffix', request.urlSuffix)}
            </Stack>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              {renderMetadata('Created', formatDate(request.createdAt))}
              {renderMetadata('Updated', formatDate(request.updatedAt))}
              {renderMetadata('Payload Format', request.payloadFormat)}
            </Stack>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              {renderMetadata('Latest Delivery', request.latestDeliveryUid)}
              {renderMetadata('Delivery Status', request.latestDeliveryStatus)}
              {renderMetadata('Latest Job', request.latestAsyncTaskUid || request.latestAsyncRemoteJobId)}
            </Stack>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              {renderMetadata('Job State', request.latestAsyncState)}
              {renderMetadata('Remote Job ID', request.latestAsyncRemoteJobId)}
              {renderMetadata('Poll URL', request.latestAsyncPollUrl)}
            </Stack>
            <Divider />
            <Box>
              <Typography variant="subtitle2" gutterBottom>
                Payload
              </Typography>
              <Box component="pre" sx={{ m: 0, p: 2, borderRadius: 2, bgcolor: 'background.default', overflowX: 'auto' }}>
                {formatJSON(request.payload ?? request.payloadBody)}
              </Box>
            </Box>
            <Box>
              <Typography variant="subtitle2" gutterBottom>
                Metadata
              </Typography>
              <Box component="pre" sx={{ m: 0, p: 2, borderRadius: 2, bgcolor: 'background.default', overflowX: 'auto' }}>
                {formatJSON(request.extras)}
              </Box>
            </Box>
            <Divider />
            <Box>
              <Typography variant="subtitle2" gutterBottom>
                Event Timeline
              </Typography>
              <EventTimeline events={events} emptyMessage="No request events recorded yet." />
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
