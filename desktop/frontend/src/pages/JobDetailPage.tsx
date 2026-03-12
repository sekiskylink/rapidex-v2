import React from 'react'
import {
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Stack,
  Typography,
} from '@mui/material'

export interface JobPollRecord {
  id: number
  asyncTaskId: number
  polledAt: string
  statusCode?: number | null
  remoteStatus: string
  responseBody: string
  errorMessage: string
  durationMs?: number | null
}

export interface JobDetailRecord {
  id: number
  uid: string
  deliveryAttemptId: number
  deliveryUid: string
  requestId: number
  requestUid: string
  remoteJobId: string
  pollUrl: string
  remoteStatus: string
  terminalState: string
  currentState: string
  nextPollAt?: string | null
  completedAt?: string | null
  remoteResponse: Record<string, unknown>
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

function stateColor(state: string): 'default' | 'warning' | 'success' | 'error' | 'info' {
  switch (state) {
    case 'pending':
      return 'warning'
    case 'polling':
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

interface JobDetailPageProps {
  open: boolean
  job: JobDetailRecord | null
  polls: JobPollRecord[]
  onClose: () => void
}

export function JobDetailPage({ open, job, polls, onClose }: JobDetailPageProps) {
  return (
    <Dialog open={open} onClose={onClose} maxWidth="lg" fullWidth>
      <DialogTitle>Job Detail</DialogTitle>
      <DialogContent>
        {job ? (
          <Stack spacing={2} sx={{ pt: 1 }}>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems={{ xs: 'flex-start', md: 'center' }}>
              <Typography variant="h6">{job.uid}</Typography>
              <Chip label={job.currentState} color={stateColor(job.currentState)} size="small" />
            </Stack>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              {renderMetadata('Delivery Reference', job.deliveryUid)}
              {renderMetadata('Request Reference', job.requestUid)}
              {renderMetadata('Remote Job ID', job.remoteJobId || '-')}
            </Stack>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              {renderMetadata('Remote Status', job.remoteStatus || '-')}
              {renderMetadata('Terminal State', job.terminalState || '-')}
              {renderMetadata('Poll URL', job.pollUrl || '-')}
            </Stack>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              {renderMetadata('Next Poll', formatDate(job.nextPollAt))}
              {renderMetadata('Completed', formatDate(job.completedAt))}
              {renderMetadata('Created', formatDate(job.createdAt))}
            </Stack>
            <Divider />
            <Box>
              <Typography variant="subtitle2" gutterBottom>
                Latest Remote Response
              </Typography>
              <Box component="pre" sx={{ m: 0, p: 2, borderRadius: 2, bgcolor: 'background.default', overflowX: 'auto' }}>
                {JSON.stringify(job.remoteResponse ?? {}, null, 2)}
              </Box>
            </Box>
            <Box>
              <Typography variant="subtitle2" gutterBottom>
                Poll History
              </Typography>
              <Stack spacing={1}>
                {polls.length === 0 ? <Typography variant="body2">No poll history recorded yet.</Typography> : null}
                {polls.map((poll) => (
                  <Box key={poll.id} sx={{ p: 1.5, borderRadius: 2, border: '1px solid', borderColor: 'divider' }}>
                    <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                      {renderMetadata('Timestamp', formatDate(poll.polledAt))}
                      {renderMetadata('Status Code', poll.statusCode ?? '-')}
                      {renderMetadata('Remote Status', poll.remoteStatus || '-')}
                      {renderMetadata('Duration (ms)', poll.durationMs ?? '-')}
                    </Stack>
                    <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} sx={{ mt: 1 }}>
                      {renderMetadata('Error', poll.errorMessage || '-')}
                    </Stack>
                  </Box>
                ))}
              </Stack>
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
