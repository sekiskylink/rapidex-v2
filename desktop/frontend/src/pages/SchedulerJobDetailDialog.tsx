import React from 'react'
import { Alert, Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, Divider, Stack, Typography } from '@mui/material'

export interface SchedulerJobDetailRecord {
  id: number
  uid: string
  code: string
  name: string
  description: string
  jobCategory: string
  jobType: string
  scheduleType: string
  scheduleExpr: string
  timezone: string
  enabled: boolean
  allowConcurrentRuns: boolean
  config: Record<string, unknown>
  lastRunAt?: string | null
  nextRunAt?: string | null
  lastSuccessAt?: string | null
  lastFailureAt?: string | null
  latestRunStatus?: string | null
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

function formatJSON(value: unknown) {
  try {
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

function statusColor(status?: string | null): 'default' | 'warning' | 'success' | 'error' | 'info' {
  switch ((status ?? '').trim().toLowerCase()) {
    case 'pending':
      return 'info'
    case 'running':
      return 'warning'
    case 'succeeded':
      return 'success'
    case 'failed':
    case 'cancelled':
      return 'error'
    default:
      return 'default'
  }
}

interface SchedulerJobDetailDialogProps {
  open: boolean
  job: SchedulerJobDetailRecord | null
  loading: boolean
  errorMessage: string
  onClose: () => void
  onEdit: () => void
}

export function SchedulerJobDetailDialog({ open, job, loading, errorMessage, onClose, onEdit }: SchedulerJobDetailDialogProps) {
  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>Scheduled Job Detail</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          {loading ? <Typography color="text.secondary">Loading scheduled job details...</Typography> : null}
          {errorMessage ? <Alert severity="error">{errorMessage}</Alert> : null}
          {job && !loading ? (
            <>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems={{ xs: 'flex-start', md: 'center' }}>
                <Typography variant="h6">{job.name}</Typography>
                <Chip label={job.enabled ? 'Enabled' : 'Disabled'} color={job.enabled ? 'success' : 'default'} size="small" />
                <Chip label={job.latestRunStatus || 'No runs'} color={statusColor(job.latestRunStatus)} size="small" />
              </Stack>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderMetadata('UID', job.uid)}
                {renderMetadata('Code', job.code)}
                {renderMetadata('Description', job.description || '-')}
              </Stack>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderMetadata('Category', job.jobCategory)}
                {renderMetadata('Job Type', job.jobType)}
                {renderMetadata('Concurrent Runs', job.allowConcurrentRuns ? 'Allowed' : 'Single active run')}
              </Stack>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderMetadata('Schedule Type', job.scheduleType)}
                {renderMetadata('Expression', job.scheduleExpr)}
                {renderMetadata('Timezone', job.timezone)}
              </Stack>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderMetadata('Next Run', formatDate(job.nextRunAt))}
                {renderMetadata('Last Run', formatDate(job.lastRunAt))}
                {renderMetadata('Last Success', formatDate(job.lastSuccessAt))}
              </Stack>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                {renderMetadata('Last Failure', formatDate(job.lastFailureAt))}
                {renderMetadata('Created', formatDate(job.createdAt))}
                {renderMetadata('Updated', formatDate(job.updatedAt))}
              </Stack>
              <Divider />
              <Box>
                <Typography variant="subtitle2" gutterBottom>
                  Configuration
                </Typography>
                <Box component="pre" sx={{ m: 0, p: 2, borderRadius: 2, bgcolor: 'background.default', overflowX: 'auto' }}>
                  {formatJSON(job.config)}
                </Box>
              </Box>
            </>
          ) : null}
        </Stack>
      </DialogContent>
      <DialogActions>
        {job ? (
          <Button variant="outlined" onClick={onEdit}>
            Edit
          </Button>
        ) : null}
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  )
}
