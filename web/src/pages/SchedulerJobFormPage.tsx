import React from 'react'
import {
  Alert,
  Box,
  Button,
  FormControlLabel,
  MenuItem,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import { useNavigate, useParams } from '@tanstack/react-router'
import { handleAppError } from '../errors/handleAppError'
import { apiRequest } from '../lib/api'
import { useAppNotify } from '../notifications/facade'

interface ScheduledJobRecord {
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
  nextRunAt?: string | null
}

interface SchedulerJobFormState {
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
  configText: string
}

const defaultFormState: SchedulerJobFormState = {
  code: '',
  name: '',
  description: '',
  jobCategory: 'integration',
  jobType: 'metadata_sync',
  scheduleType: 'interval',
  scheduleExpr: '15m',
  timezone: 'UTC',
  enabled: true,
  allowConcurrentRuns: false,
  configText: '{}',
}

const jobTypeOptions = [
  { value: 'metadata_sync', label: 'Metadata Sync' },
  { value: 'export_pending_requests', label: 'Export Pending Requests' },
  { value: 'reconciliation_pull', label: 'Reconciliation Pull' },
  { value: 'scheduled_backfill', label: 'Scheduled Backfill' },
  { value: 'archive_old_requests', label: 'Archive Old Requests' },
  { value: 'purge_old_logs', label: 'Purge Old Logs' },
  { value: 'mark_stuck_requests', label: 'Mark Stuck Requests' },
  { value: 'cleanup_orphaned_records', label: 'Cleanup Orphaned Records' },
] as const

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

export function SchedulerJobFormPage() {
  const notify = useAppNotify()
  const navigate = useNavigate()
  const params = useParams({ strict: false }) as { jobId?: string }
  const jobId = params.jobId ? Number(params.jobId) : null
  const isEdit = Number.isFinite(jobId)
  const [loading, setLoading] = React.useState(Boolean(isEdit))
  const [saving, setSaving] = React.useState(false)
  const [errorMessage, setErrorMessage] = React.useState('')
  const [record, setRecord] = React.useState<ScheduledJobRecord | null>(null)
  const [form, setForm] = React.useState<SchedulerJobFormState>(defaultFormState)

  React.useEffect(() => {
    if (!isEdit || !jobId) {
      setLoading(false)
      setRecord(null)
      setForm(defaultFormState)
      return
    }

    let active = true
    setLoading(true)
    setErrorMessage('')
    apiRequest<ScheduledJobRecord>(`/scheduler/jobs/${jobId}`)
      .then((response) => {
        if (!active) {
          return
        }
        setRecord(response)
        setForm({
          code: response.code,
          name: response.name,
          description: response.description,
          jobCategory: response.jobCategory,
          jobType: response.jobType,
          scheduleType: response.scheduleType,
          scheduleExpr: response.scheduleExpr,
          timezone: response.timezone,
          enabled: response.enabled,
          allowConcurrentRuns: response.allowConcurrentRuns,
          configText: JSON.stringify(response.config ?? {}, null, 2),
        })
      })
      .catch(async (error) => {
        if (!active) {
          return
        }
        setErrorMessage('Unable to load scheduled job.')
        await handleAppError(error, {
          fallbackMessage: 'Unable to load scheduled job.',
          notifier: notify,
        })
      })
      .finally(() => {
        if (active) {
          setLoading(false)
        }
      })

    return () => {
      active = false
    }
  }, [isEdit, jobId, notify])

  const updateField = <K extends keyof SchedulerJobFormState>(field: K, value: SchedulerJobFormState[K]) => {
    setForm((current) => ({ ...current, [field]: value }))
  }

  const handleSubmit = async () => {
    setSaving(true)
    setErrorMessage('')

    let configValue: Record<string, unknown> = {}
    try {
      configValue = JSON.parse(form.configText || '{}') as Record<string, unknown>
    } catch {
      setSaving(false)
      setErrorMessage('Config JSON must be valid.')
      return
    }

    try {
      const payload = {
        code: form.code,
        name: form.name,
        description: form.description,
        jobCategory: form.jobCategory,
        jobType: form.jobType,
        scheduleType: form.scheduleType,
        scheduleExpr: form.scheduleExpr,
        timezone: form.timezone,
        enabled: form.enabled,
        allowConcurrentRuns: form.allowConcurrentRuns,
        config: configValue,
      }
      const response = await apiRequest<ScheduledJobRecord>(isEdit && jobId ? `/scheduler/jobs/${jobId}` : '/scheduler/jobs', {
        method: isEdit ? 'PUT' : 'POST',
        body: JSON.stringify(payload),
      })
      notify.success(isEdit ? 'Scheduled job updated.' : 'Scheduled job created.')
      void navigate({ to: '/scheduler/$jobId', params: { jobId: String(response.id) }, replace: true })
    } catch (error) {
      setErrorMessage(isEdit ? 'Unable to update scheduled job.' : 'Unable to create scheduled job.')
      await handleAppError(error, {
        fallbackMessage: isEdit ? 'Unable to update scheduled job.' : 'Unable to create scheduled job.',
        notifier: notify,
      })
    } finally {
      setSaving(false)
    }
  }

  const handleRunNow = async () => {
    if (!jobId) {
      return
    }
    try {
      await apiRequest(`/scheduler/jobs/${jobId}/run-now`, { method: 'POST' })
      notify.success('Scheduled job run queued.')
    } catch (error) {
      await handleAppError(error, {
        fallbackMessage: 'Unable to queue scheduled job run.',
        notifier: notify,
      })
    }
  }

  return (
    <Stack spacing={3}>
      <Box display="flex" justifyContent="space-between" gap={2} flexDirection={{ xs: 'column', md: 'row' }}>
        <Box>
          <Typography variant="h4" component="h1">
            {isEdit ? 'Edit Scheduled Job' : 'Create Scheduled Job'}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Define job category, schedule, and runtime configuration for the scheduler v1 slice.
          </Typography>
        </Box>
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5}>
          <Button variant="outlined" onClick={() => void navigate({ to: '/scheduler' })}>
            Back to Scheduler
          </Button>
          {isEdit && jobId ? (
            <Button variant="outlined" onClick={handleRunNow}>
              Run Now
            </Button>
          ) : null}
          {isEdit && jobId ? (
            <Button variant="outlined" onClick={() => void navigate({ to: '/scheduler/$jobId/runs', params: { jobId: String(jobId) } })}>
              View Runs
            </Button>
          ) : null}
        </Stack>
      </Box>

      {errorMessage ? <Alert severity="error">{errorMessage}</Alert> : null}

      {record ? (
        <Alert severity="info">
          Job UID: {record.uid} | Next run: {formatDate(record.nextRunAt)}
        </Alert>
      ) : null}

      <Box
        sx={{
          p: 3,
          borderRadius: 2,
          border: (theme) => `1px solid ${theme.palette.divider}`,
          bgcolor: 'background.paper',
        }}
      >
        <Stack spacing={2.5}>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField label="Code" value={form.code} onChange={(event) => updateField('code', event.target.value)} fullWidth disabled={loading || saving} />
            <TextField label="Name" value={form.name} onChange={(event) => updateField('name', event.target.value)} fullWidth disabled={loading || saving} />
          </Stack>

          <TextField
            label="Description"
            value={form.description}
            onChange={(event) => updateField('description', event.target.value)}
            multiline
            minRows={2}
            disabled={loading || saving}
          />

          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField select label="Job Category" value={form.jobCategory} onChange={(event) => updateField('jobCategory', event.target.value)} fullWidth disabled={loading || saving}>
              <MenuItem value="integration">Integration</MenuItem>
              <MenuItem value="maintenance">Maintenance</MenuItem>
            </TextField>
            <TextField
              select
              label="Job Type"
              value={form.jobType}
              onChange={(event) => updateField('jobType', event.target.value)}
              fullWidth
              disabled={loading || saving}
              helperText="Scheduler runtime supports the registered scheduler job types."
            >
              {jobTypeOptions.map((option) => (
                <MenuItem key={option.value} value={option.value}>
                  {option.label}
                </MenuItem>
              ))}
            </TextField>
          </Stack>

          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField select label="Schedule Type" value={form.scheduleType} onChange={(event) => updateField('scheduleType', event.target.value)} fullWidth disabled={loading || saving}>
              <MenuItem value="interval">Interval</MenuItem>
              <MenuItem value="cron">Cron</MenuItem>
            </TextField>
            <TextField label="Schedule Expression" value={form.scheduleExpr} onChange={(event) => updateField('scheduleExpr', event.target.value)} fullWidth disabled={loading || saving} />
            <TextField label="Timezone" value={form.timezone} onChange={(event) => updateField('timezone', event.target.value)} fullWidth disabled={loading || saving} />
          </Stack>

          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <FormControlLabel
              control={<Switch checked={form.enabled} onChange={(event) => updateField('enabled', event.target.checked)} disabled={loading || saving} />}
              label="Enabled"
            />
            <FormControlLabel
              control={
                <Switch
                  checked={form.allowConcurrentRuns}
                  onChange={(event) => updateField('allowConcurrentRuns', event.target.checked)}
                  disabled={loading || saving}
                />
              }
              label="Allow Concurrent Runs"
            />
          </Stack>

          <TextField
            label="Config JSON"
            value={form.configText}
            onChange={(event) => updateField('configText', event.target.value)}
            multiline
            minRows={10}
            disabled={loading || saving}
          />

          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} justifyContent="flex-end">
            <Button variant="outlined" onClick={() => void navigate({ to: '/scheduler' })} disabled={saving}>
              Cancel
            </Button>
            <Button variant="contained" onClick={handleSubmit} disabled={loading || saving}>
              {isEdit ? 'Save Changes' : 'Create Scheduled Job'}
            </Button>
          </Stack>
        </Stack>
      </Box>
    </Stack>
  )
}
