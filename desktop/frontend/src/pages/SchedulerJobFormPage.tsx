import React from 'react'
import {
  Alert,
  Box,
  Button,
  Chip,
  FormControlLabel,
  MenuItem,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useApiClient } from '../api/useApiClient'
import { handleAppError } from '../errors/handleAppError'

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
  latestRunStatus?: string | null
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

const maintenanceJobTypes = new Set([
  'archive_old_requests',
  'purge_old_logs',
  'mark_stuck_requests',
  'cleanup_orphaned_records',
])

interface MaintenanceConfigState {
  dryRun: boolean
  batchSize: string
  maxAgeDays: string
  staleCutoffMinutes: string
  staleCutoffHours: string
}

const defaultMaintenanceConfigs: Record<string, MaintenanceConfigState> = {
  archive_old_requests: { dryRun: false, batchSize: '100', maxAgeDays: '30', staleCutoffMinutes: '', staleCutoffHours: '' },
  purge_old_logs: { dryRun: false, batchSize: '500', maxAgeDays: '30', staleCutoffMinutes: '', staleCutoffHours: '' },
  mark_stuck_requests: { dryRun: false, batchSize: '100', maxAgeDays: '', staleCutoffMinutes: '30', staleCutoffHours: '' },
  cleanup_orphaned_records: { dryRun: false, batchSize: '100', maxAgeDays: '14', staleCutoffMinutes: '', staleCutoffHours: '' },
}

function isMaintenanceJobType(jobType: string) {
  return maintenanceJobTypes.has(jobType)
}

function toStringNumber(value: unknown, fallback = '') {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value)
  }
  if (typeof value === 'string' && value.trim() !== '') {
    return value
  }
  return fallback
}

function getMaintenanceConfigState(jobType: string, config: Record<string, unknown>): MaintenanceConfigState {
  const defaults = defaultMaintenanceConfigs[jobType] ?? defaultMaintenanceConfigs.archive_old_requests
  return {
    dryRun: Boolean(config.dryRun ?? defaults.dryRun),
    batchSize: toStringNumber(config.batchSize, defaults.batchSize),
    maxAgeDays: toStringNumber(config.maxAgeDays, defaults.maxAgeDays),
    staleCutoffMinutes: toStringNumber(config.staleCutoffMinutes, defaults.staleCutoffMinutes),
    staleCutoffHours: toStringNumber(config.staleCutoffHours, defaults.staleCutoffHours),
  }
}

function parseOptionalInt(value: string) {
  const trimmed = value.trim()
  if (!trimmed) {
    return undefined
  }
  return Number.parseInt(trimmed, 10)
}

function buildMaintenanceConfig(jobType: string, config: MaintenanceConfigState) {
  const payload: Record<string, unknown> = {
    dryRun: config.dryRun,
    batchSize: parseOptionalInt(config.batchSize) ?? 0,
  }
  if (jobType === 'mark_stuck_requests') {
    if (config.staleCutoffMinutes.trim()) {
      payload.staleCutoffMinutes = parseOptionalInt(config.staleCutoffMinutes) ?? 0
    }
    if (config.staleCutoffHours.trim()) {
      payload.staleCutoffHours = parseOptionalInt(config.staleCutoffHours) ?? 0
    }
    return payload
  }
  payload.maxAgeDays = parseOptionalInt(config.maxAgeDays) ?? 0
  return payload
}

function statusChip(status: string) {
  const normalized = status.trim().toLowerCase()
  const color =
    normalized === 'succeeded'
      ? 'success'
      : normalized === 'failed' || normalized === 'cancelled'
        ? 'error'
        : normalized === 'running'
          ? 'warning'
          : normalized === 'pending'
            ? 'info'
            : 'default'
  return <Chip label={status || 'No runs yet'} size="small" color={color} />
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

export function SchedulerJobFormPage() {
  const apiClient = useApiClient()
  const navigate = useNavigate()
  const params = useParams({ strict: false }) as { jobId?: string }
  const jobId = params.jobId ? Number(params.jobId) : null
  const isEdit = Number.isFinite(jobId)
  const [loading, setLoading] = React.useState(Boolean(isEdit))
  const [saving, setSaving] = React.useState(false)
  const [errorMessage, setErrorMessage] = React.useState('')
  const [record, setRecord] = React.useState<ScheduledJobRecord | null>(null)
  const [form, setForm] = React.useState<SchedulerJobFormState>(defaultFormState)
  const [maintenanceConfig, setMaintenanceConfig] = React.useState<MaintenanceConfigState>(defaultMaintenanceConfigs.archive_old_requests)

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
    apiClient
      .request<ScheduledJobRecord>(`/api/v1/scheduler/jobs/${jobId}`)
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
        setMaintenanceConfig(getMaintenanceConfigState(response.jobType, response.config ?? {}))
      })
      .catch(async (error) => {
        if (!active) {
          return
        }
        setErrorMessage('Unable to load scheduled job.')
        await handleAppError(error, { fallbackMessage: 'Unable to load scheduled job.' })
      })
      .finally(() => {
        if (active) {
          setLoading(false)
        }
      })

    return () => {
      active = false
    }
  }, [apiClient, isEdit, jobId])

  const updateField = <K extends keyof SchedulerJobFormState>(field: K, value: SchedulerJobFormState[K]) => {
    setForm((current) => ({ ...current, [field]: value }))
  }

  const updateMaintenanceField = <K extends keyof MaintenanceConfigState>(field: K, value: MaintenanceConfigState[K]) => {
    setMaintenanceConfig((current) => ({ ...current, [field]: value }))
  }

  const applyJobCategory = (jobCategory: string) => {
    const nextJobType = jobCategory === 'maintenance'
      ? (isMaintenanceJobType(form.jobType) ? form.jobType : 'archive_old_requests')
      : (isMaintenanceJobType(form.jobType) ? 'metadata_sync' : form.jobType)
    updateField('jobCategory', jobCategory)
    updateField('jobType', nextJobType)
    if (isMaintenanceJobType(nextJobType)) {
      setMaintenanceConfig(getMaintenanceConfigState(nextJobType, {}))
    }
  }

  const applyJobType = (jobType: string) => {
    updateField('jobType', jobType)
    updateField('jobCategory', isMaintenanceJobType(jobType) ? 'maintenance' : 'integration')
    if (isMaintenanceJobType(jobType)) {
      setMaintenanceConfig(getMaintenanceConfigState(jobType, record?.config ?? {}))
    }
  }

  const handleSubmit = async () => {
    setSaving(true)
    setErrorMessage('')

    let configValue: Record<string, unknown> = {}
    if (isMaintenanceJobType(form.jobType)) {
      configValue = buildMaintenanceConfig(form.jobType, maintenanceConfig)
    } else {
      try {
        configValue = JSON.parse(form.configText || '{}') as Record<string, unknown>
      } catch {
        setSaving(false)
        setErrorMessage('Config JSON must be valid.')
        return
      }
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
      const response = await apiClient.request<ScheduledJobRecord>(isEdit && jobId ? `/api/v1/scheduler/jobs/${jobId}` : '/api/v1/scheduler/jobs', {
        method: isEdit ? 'PUT' : 'POST',
        body: JSON.stringify(payload),
      })
      void navigate({ to: '/scheduler/$jobId', params: { jobId: String(response.id) }, replace: true })
    } catch (error) {
      setErrorMessage(isEdit ? 'Unable to update scheduled job.' : 'Unable to create scheduled job.')
      await handleAppError(error, {
        fallbackMessage: isEdit ? 'Unable to update scheduled job.' : 'Unable to create scheduled job.',
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
      await apiClient.request(`/api/v1/scheduler/jobs/${jobId}/run-now`, { method: 'POST' })
    } catch (error) {
      await handleAppError(error, { fallbackMessage: 'Unable to queue scheduled job run.' })
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
          Job UID: {record.uid} | Next run: {formatDate(record.nextRunAt)} | Latest status: {statusChip(record.latestRunStatus ?? '')}
        </Alert>
      ) : null}

      {isMaintenanceJobType(form.jobType) && maintenanceConfig.dryRun ? (
        <Alert severity="warning">Dry run is enabled. This maintenance job will scan candidates and record a summary without changing data.</Alert>
      ) : null}

      <Box sx={{ p: 3, borderRadius: 2, border: (theme) => `1px solid ${theme.palette.divider}`, bgcolor: 'background.paper' }}>
        <Stack spacing={2.5}>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField label="Code" value={form.code} onChange={(event) => updateField('code', event.target.value)} fullWidth disabled={loading || saving} />
            <TextField label="Name" value={form.name} onChange={(event) => updateField('name', event.target.value)} fullWidth disabled={loading || saving} />
          </Stack>
          <TextField label="Description" value={form.description} onChange={(event) => updateField('description', event.target.value)} multiline minRows={2} disabled={loading || saving} />
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField select label="Job Category" value={form.jobCategory} onChange={(event) => applyJobCategory(event.target.value)} fullWidth disabled={loading || saving}>
              <MenuItem value="integration">Integration</MenuItem>
              <MenuItem value="maintenance">Maintenance</MenuItem>
            </TextField>
            <TextField
              select
              label="Job Type"
              value={form.jobType}
              onChange={(event) => applyJobType(event.target.value)}
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
            <FormControlLabel control={<Switch checked={form.enabled} onChange={(event) => updateField('enabled', event.target.checked)} disabled={loading || saving} />} label="Enabled" />
            <FormControlLabel
              control={<Switch checked={form.allowConcurrentRuns} onChange={(event) => updateField('allowConcurrentRuns', event.target.checked)} disabled={loading || saving} />}
              label="Allow Concurrent Runs"
            />
          </Stack>
          {isMaintenanceJobType(form.jobType) ? (
            <Stack spacing={2}>
              <Typography variant="subtitle2">Maintenance Configuration</Typography>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                <TextField
                  label="Batch Size"
                  value={maintenanceConfig.batchSize}
                  onChange={(event) => updateMaintenanceField('batchSize', event.target.value)}
                  fullWidth
                  disabled={loading || saving}
                />
                {form.jobType === 'mark_stuck_requests' ? (
                  <>
                    <TextField
                      label="Stale Cutoff Minutes"
                      value={maintenanceConfig.staleCutoffMinutes}
                      onChange={(event) => updateMaintenanceField('staleCutoffMinutes', event.target.value)}
                      fullWidth
                      disabled={loading || saving}
                      helperText="Preferred for short scheduler intervals."
                    />
                    <TextField
                      label="Stale Cutoff Hours"
                      value={maintenanceConfig.staleCutoffHours}
                      onChange={(event) => updateMaintenanceField('staleCutoffHours', event.target.value)}
                      fullWidth
                      disabled={loading || saving}
                      helperText="Optional alternative to minutes."
                    />
                  </>
                ) : (
                  <TextField
                    label="Max Age Days"
                    value={maintenanceConfig.maxAgeDays}
                    onChange={(event) => updateMaintenanceField('maxAgeDays', event.target.value)}
                    fullWidth
                    disabled={loading || saving}
                  />
                )}
              </Stack>
              <FormControlLabel
                control={<Switch checked={maintenanceConfig.dryRun} onChange={(event) => updateMaintenanceField('dryRun', event.target.checked)} disabled={loading || saving} />}
                label="Dry Run"
              />
            </Stack>
          ) : (
            <TextField label="Config JSON" value={form.configText} onChange={(event) => updateField('configText', event.target.value)} multiline minRows={10} disabled={loading || saving} />
          )}
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
