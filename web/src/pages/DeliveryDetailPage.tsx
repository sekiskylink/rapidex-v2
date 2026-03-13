import React from 'react'
import { Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, Divider, Stack, Typography } from '@mui/material'

export interface DeliveryDetailRecord {
  id: number
  uid: string
  requestId: number
  requestUid: string
  serverId: number
  serverName: string
  attemptNumber: number
  status: string
  httpStatus?: number | null
  responseBody: string
  errorMessage: string
  systemType: string
  submissionMode: string
  asyncTaskId?: number | null
  asyncTaskUid: string
  asyncCurrentState: string
  asyncRemoteJobId: string
  asyncPollUrl: string
  awaitingAsync: boolean
  startedAt?: string | null
  finishedAt?: string | null
  retryAt?: string | null
  createdAt: string
  updatedAt: string
}

function formatDate(value?: string | null) {
  if (!value) {
    return '-'
  }
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return value
  }
  return parsed.toLocaleString()
}

function statusColor(status: string): 'default' | 'warning' | 'success' | 'error' | 'info' {
  switch (status) {
    case 'pending':
    case 'retrying':
      return 'warning'
    case 'running':
      return 'info'
    case 'succeeded':
      return 'success'
    case 'failed':
      return 'error'
    default:
      return 'default'
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

interface DeliveryDetailPageProps {
  open: boolean
  delivery: DeliveryDetailRecord | null
  canRetry: boolean
  retrying: boolean
  onRetry: () => void
  onClose: () => void
}

export function DeliveryDetailPage({ open, delivery, canRetry, retrying, onRetry, onClose }: DeliveryDetailPageProps) {
  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>Delivery Detail</DialogTitle>
      <DialogContent>
        {delivery ? (
          <Stack spacing={2} sx={{ pt: 1 }}>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems={{ xs: 'flex-start', md: 'center' }}>
              <Typography variant="h6">{delivery.uid}</Typography>
              <Chip label={delivery.status} color={statusColor(delivery.status)} size="small" />
              {delivery.awaitingAsync ? <Chip label="Awaiting async" color="info" size="small" /> : null}
            </Stack>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              {renderMetadata('Request Reference', delivery.requestUid)}
              {renderMetadata('Server', delivery.serverName)}
              {renderMetadata('Attempt Number', delivery.attemptNumber)}
            </Stack>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              {renderMetadata('HTTP Status', delivery.httpStatus ?? '-')}
              {renderMetadata('System Type', delivery.systemType)}
              {renderMetadata('Submission Mode', delivery.submissionMode)}
            </Stack>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              {renderMetadata('Linked Job', delivery.asyncTaskUid || '-')}
              {renderMetadata('Async State', delivery.asyncCurrentState || '-')}
              {renderMetadata('Remote Job ID', delivery.asyncRemoteJobId || '-')}
            </Stack>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              {renderMetadata('Poll URL', delivery.asyncPollUrl || '-')}
              {renderMetadata('Started', formatDate(delivery.startedAt))}
              {renderMetadata('Finished', formatDate(delivery.finishedAt))}
            </Stack>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              {renderMetadata('Retry At', formatDate(delivery.retryAt))}
              {renderMetadata('Created', formatDate(delivery.createdAt))}
              {renderMetadata('Updated', formatDate(delivery.updatedAt))}
            </Stack>
            <Divider />
            <Box>
              <Typography variant="subtitle2" gutterBottom>
                Response Body
              </Typography>
              <Box component="pre" sx={{ m: 0, p: 2, borderRadius: 2, bgcolor: 'background.default', overflowX: 'auto' }}>
                {delivery.responseBody || '-'}
              </Box>
            </Box>
            <Box>
              <Typography variant="subtitle2" gutterBottom>
                Error Message
              </Typography>
              <Box component="pre" sx={{ m: 0, p: 2, borderRadius: 2, bgcolor: 'background.default', overflowX: 'auto' }}>
                {delivery.errorMessage || '-'}
              </Box>
            </Box>
          </Stack>
        ) : null}
      </DialogContent>
      <DialogActions>
        {canRetry ? (
          <Button onClick={onRetry} variant="contained" disabled={retrying}>
            Retry
          </Button>
        ) : null}
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  )
}
