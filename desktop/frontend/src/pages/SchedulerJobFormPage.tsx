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
  { value: 'url_call', label: 'URL Call' },
  { value: 'request_exchange', label: 'Request Exchange' },
  { value: 'rapidpro_reporter_sync', label: 'RapidPro Reporter Sync' },
  { value: 'dhis2_org_unit_refresh', label: 'DHIS2 Org Unit Refresh' },
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

interface URLCallConfigState {
  destinationServerUid: string
  urlSuffix: string
  payloadFormat: string
  submissionBinding: string
  responseBodyPersistence: string
  payloadText: string
}

interface RequestExchangeConfigState extends URLCallConfigState {
  sourceSystem: string
  destinationServerUids: string
  batchId: string
  correlationId: string
  idempotencyKeyPrefix: string
  metadataText: string
}

interface RapidProReporterSyncConfigState {
  dryRun: boolean
  batchSize: string
  onlyActive: boolean
  lookbackMinutes: string
}

interface DHIS2OrgUnitRefreshConfigState {
  serverUid: string
  serverCode: string
  fullRefresh: boolean
  dryRun: boolean
  districtLevelName: string
  districtLevelCode: string
}

const defaultMaintenanceConfigs: Record<string, MaintenanceConfigState> = {
  archive_old_requests: { dryRun: false, batchSize: '100', maxAgeDays: '30', staleCutoffMinutes: '', staleCutoffHours: '' },
  purge_old_logs: { dryRun: false, batchSize: '500', maxAgeDays: '30', staleCutoffMinutes: '', staleCutoffHours: '' },
  mark_stuck_requests: { dryRun: false, batchSize: '100', maxAgeDays: '', staleCutoffMinutes: '30', staleCutoffHours: '' },
  cleanup_orphaned_records: { dryRun: false, batchSize: '100', maxAgeDays: '14', staleCutoffMinutes: '', staleCutoffHours: '' },
}

const defaultURLCallConfig: URLCallConfigState = {
  destinationServerUid: '',
  urlSuffix: '',
  payloadFormat: 'json',
  submissionBinding: 'body',
  responseBodyPersistence: '',
  payloadText: '{}',
}

const defaultRequestExchangeConfig: RequestExchangeConfigState = {
  ...defaultURLCallConfig,
  sourceSystem: 'scheduler',
  destinationServerUids: '',
  batchId: '',
  correlationId: '',
  idempotencyKeyPrefix: '',
  metadataText: '{}',
}

const defaultRapidProReporterSyncConfig: RapidProReporterSyncConfigState = {
  dryRun: false,
  batchSize: '100',
  onlyActive: true,
  lookbackMinutes: '1',
}

const defaultDHIS2OrgUnitRefreshConfig: DHIS2OrgUnitRefreshConfigState = {
  serverUid: '',
  serverCode: 'dhis2',
  fullRefresh: true,
  dryRun: false,
  districtLevelName: 'District',
  districtLevelCode: '',
}

function isMaintenanceJobType(jobType: string) {
  return maintenanceJobTypes.has(jobType)
}

function isURLCallJobType(jobType: string) {
  return jobType === 'url_call'
}

function isRequestExchangeJobType(jobType: string) {
  return jobType === 'request_exchange'
}

function isRapidProReporterSyncJobType(jobType: string) {
  return jobType === 'rapidpro_reporter_sync'
}

function isDHIS2OrgUnitRefreshJobType(jobType: string) {
  return jobType === 'dhis2_org_unit_refresh'
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

function getURLCallConfigState(config: Record<string, unknown>): URLCallConfigState {
  const payloadFormat = typeof config.payloadFormat === 'string' && config.payloadFormat ? config.payloadFormat : 'json'
  return {
    destinationServerUid: typeof config.destinationServerUid === 'string' ? config.destinationServerUid : '',
    urlSuffix: typeof config.urlSuffix === 'string' ? config.urlSuffix : '',
    payloadFormat,
    submissionBinding: typeof config.submissionBinding === 'string' && config.submissionBinding ? config.submissionBinding : 'body',
    responseBodyPersistence: typeof config.responseBodyPersistence === 'string' ? config.responseBodyPersistence : '',
    payloadText:
      payloadFormat === 'text'
        ? typeof config.payload === 'string'
          ? config.payload
          : ''
        : JSON.stringify(config.payload ?? {}, null, 2),
  }
}

function getRequestExchangeConfigState(config: Record<string, unknown>): RequestExchangeConfigState {
  const base = getURLCallConfigState(config)
  return {
    ...base,
    sourceSystem: typeof config.sourceSystem === 'string' && config.sourceSystem ? config.sourceSystem : 'scheduler',
    destinationServerUids: Array.isArray(config.destinationServerUids) ? config.destinationServerUids.join(', ') : '',
    batchId: typeof config.batchId === 'string' ? config.batchId : '',
    correlationId: typeof config.correlationId === 'string' ? config.correlationId : '',
    idempotencyKeyPrefix: typeof config.idempotencyKeyPrefix === 'string' ? config.idempotencyKeyPrefix : '',
    metadataText: JSON.stringify(config.metadata ?? {}, null, 2),
  }
}

function getRapidProReporterSyncConfigState(config: Record<string, unknown>): RapidProReporterSyncConfigState {
  return {
    dryRun: Boolean(config.dryRun ?? defaultRapidProReporterSyncConfig.dryRun),
    batchSize: toStringNumber(config.batchSize, defaultRapidProReporterSyncConfig.batchSize),
    onlyActive: Boolean(config.onlyActive ?? defaultRapidProReporterSyncConfig.onlyActive),
    lookbackMinutes: toStringNumber(config.lookbackMinutes, defaultRapidProReporterSyncConfig.lookbackMinutes),
  }
}

function getDHIS2OrgUnitRefreshConfigState(config: Record<string, unknown>): DHIS2OrgUnitRefreshConfigState {
  return {
    serverUid: typeof config.serverUid === 'string' ? config.serverUid : '',
    serverCode: typeof config.serverCode === 'string' && config.serverCode ? config.serverCode : defaultDHIS2OrgUnitRefreshConfig.serverCode,
    fullRefresh: config.fullRefresh !== false,
    dryRun: Boolean(config.dryRun ?? defaultDHIS2OrgUnitRefreshConfig.dryRun),
    districtLevelName: typeof config.districtLevelName === 'string' && config.districtLevelName ? config.districtLevelName : defaultDHIS2OrgUnitRefreshConfig.districtLevelName,
    districtLevelCode: typeof config.districtLevelCode === 'string' ? config.districtLevelCode : '',
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

function parsePayloadConfig(payloadFormat: string, payloadText: string) {
  if (payloadFormat === 'text') {
    return payloadText
  }
  return JSON.parse(payloadText || '{}') as unknown
}

function splitCSV(value: string) {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function buildRapidProReporterSyncConfig(config: RapidProReporterSyncConfigState) {
  return {
    dryRun: config.dryRun,
    batchSize: parseOptionalInt(config.batchSize) ?? 0,
    onlyActive: config.onlyActive,
    lookbackMinutes: parseOptionalInt(config.lookbackMinutes) ?? 1,
  }
}

function buildDHIS2OrgUnitRefreshConfig(config: DHIS2OrgUnitRefreshConfigState) {
  return {
    serverUid: config.serverUid.trim(),
    serverCode: config.serverCode.trim(),
    fullRefresh: config.fullRefresh,
    dryRun: config.dryRun,
    districtLevelName: config.districtLevelName.trim(),
    districtLevelCode: config.districtLevelCode.trim(),
  }
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
  const [urlCallConfig, setURLCallConfig] = React.useState<URLCallConfigState>(defaultURLCallConfig)
  const [requestExchangeConfig, setRequestExchangeConfig] = React.useState<RequestExchangeConfigState>(defaultRequestExchangeConfig)
  const [rapidProReporterSyncConfig, setRapidProReporterSyncConfig] = React.useState<RapidProReporterSyncConfigState>(defaultRapidProReporterSyncConfig)
  const [dhis2OrgUnitRefreshConfig, setDHIS2OrgUnitRefreshConfig] = React.useState<DHIS2OrgUnitRefreshConfigState>(defaultDHIS2OrgUnitRefreshConfig)

  React.useEffect(() => {
    if (!isEdit || !jobId) {
      setLoading(false)
      setRecord(null)
      setForm(defaultFormState)
      setURLCallConfig(defaultURLCallConfig)
      setRequestExchangeConfig(defaultRequestExchangeConfig)
      setRapidProReporterSyncConfig(defaultRapidProReporterSyncConfig)
      setDHIS2OrgUnitRefreshConfig(defaultDHIS2OrgUnitRefreshConfig)
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
        setURLCallConfig(getURLCallConfigState(response.config ?? {}))
        setRequestExchangeConfig(getRequestExchangeConfigState(response.config ?? {}))
        setRapidProReporterSyncConfig(getRapidProReporterSyncConfigState(response.config ?? {}))
        setDHIS2OrgUnitRefreshConfig(getDHIS2OrgUnitRefreshConfigState(response.config ?? {}))
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

  const updateURLCallField = <K extends keyof URLCallConfigState>(field: K, value: URLCallConfigState[K]) => {
    setURLCallConfig((current) => ({ ...current, [field]: value }))
  }

  const updateRequestExchangeField = <K extends keyof RequestExchangeConfigState>(field: K, value: RequestExchangeConfigState[K]) => {
    setRequestExchangeConfig((current) => ({ ...current, [field]: value }))
  }

  const applyJobCategory = (jobCategory: string) => {
    const nextJobType = jobCategory === 'maintenance'
      ? (isMaintenanceJobType(form.jobType) ? form.jobType : 'archive_old_requests')
      : (isMaintenanceJobType(form.jobType) ? 'metadata_sync' : form.jobType)
    updateField('jobCategory', jobCategory)
    updateField('jobType', nextJobType)
    if (isMaintenanceJobType(nextJobType)) {
      setMaintenanceConfig(getMaintenanceConfigState(nextJobType, {}))
    } else if (isURLCallJobType(nextJobType)) {
      setURLCallConfig(defaultURLCallConfig)
    } else if (isRequestExchangeJobType(nextJobType)) {
      setRequestExchangeConfig(defaultRequestExchangeConfig)
    } else if (isRapidProReporterSyncJobType(nextJobType)) {
      setRapidProReporterSyncConfig(defaultRapidProReporterSyncConfig)
    } else if (isDHIS2OrgUnitRefreshJobType(nextJobType)) {
      setDHIS2OrgUnitRefreshConfig(defaultDHIS2OrgUnitRefreshConfig)
    }
  }

  const applyJobType = (jobType: string) => {
    updateField('jobType', jobType)
    updateField('jobCategory', isMaintenanceJobType(jobType) ? 'maintenance' : 'integration')
    if (isMaintenanceJobType(jobType)) {
      setMaintenanceConfig(getMaintenanceConfigState(jobType, record?.config ?? {}))
    } else if (isURLCallJobType(jobType)) {
      setURLCallConfig(getURLCallConfigState(record?.config ?? {}))
    } else if (isRequestExchangeJobType(jobType)) {
      setRequestExchangeConfig(getRequestExchangeConfigState(record?.config ?? {}))
    } else if (isRapidProReporterSyncJobType(jobType)) {
      setRapidProReporterSyncConfig(getRapidProReporterSyncConfigState(record?.config ?? {}))
    } else if (isDHIS2OrgUnitRefreshJobType(jobType)) {
      setDHIS2OrgUnitRefreshConfig(getDHIS2OrgUnitRefreshConfigState(record?.config ?? {}))
    }
  }

  const handleSubmit = async () => {
    setSaving(true)
    setErrorMessage('')

    let configValue: Record<string, unknown> = {}
    if (isMaintenanceJobType(form.jobType)) {
      configValue = buildMaintenanceConfig(form.jobType, maintenanceConfig)
    } else if (isURLCallJobType(form.jobType)) {
      try {
        configValue = {
          destinationServerUid: urlCallConfig.destinationServerUid,
          urlSuffix: urlCallConfig.urlSuffix,
          payloadFormat: urlCallConfig.payloadFormat,
          submissionBinding: urlCallConfig.submissionBinding,
          responseBodyPersistence: urlCallConfig.responseBodyPersistence,
          payload: parsePayloadConfig(urlCallConfig.payloadFormat, urlCallConfig.payloadText),
        }
      } catch {
        setSaving(false)
        setErrorMessage('Payload JSON must be valid.')
        return
      }
    } else if (isRequestExchangeJobType(form.jobType)) {
      try {
        configValue = {
          sourceSystem: requestExchangeConfig.sourceSystem,
          destinationServerUid: requestExchangeConfig.destinationServerUid,
          destinationServerUids: splitCSV(requestExchangeConfig.destinationServerUids),
          batchId: requestExchangeConfig.batchId,
          correlationId: requestExchangeConfig.correlationId,
          idempotencyKeyPrefix: requestExchangeConfig.idempotencyKeyPrefix,
          urlSuffix: requestExchangeConfig.urlSuffix,
          payloadFormat: requestExchangeConfig.payloadFormat,
          submissionBinding: requestExchangeConfig.submissionBinding,
          responseBodyPersistence: requestExchangeConfig.responseBodyPersistence,
          payload: parsePayloadConfig(requestExchangeConfig.payloadFormat, requestExchangeConfig.payloadText),
          metadata: JSON.parse(requestExchangeConfig.metadataText || '{}') as Record<string, unknown>,
        }
      } catch {
        setSaving(false)
        setErrorMessage('Payload and metadata JSON must be valid.')
        return
      }
    } else if (isRapidProReporterSyncJobType(form.jobType)) {
      configValue = buildRapidProReporterSyncConfig(rapidProReporterSyncConfig)
    } else if (isDHIS2OrgUnitRefreshJobType(form.jobType)) {
      configValue = buildDHIS2OrgUnitRefreshConfig(dhis2OrgUnitRefreshConfig)
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
          ) : isURLCallJobType(form.jobType) ? (
            <Stack spacing={2}>
              <Typography variant="subtitle2">URL Call Configuration</Typography>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                <TextField label="Destination Server UID" value={urlCallConfig.destinationServerUid} onChange={(event) => updateURLCallField('destinationServerUid', event.target.value)} fullWidth disabled={loading || saving} />
                <TextField label="URL Suffix" value={urlCallConfig.urlSuffix} onChange={(event) => updateURLCallField('urlSuffix', event.target.value)} fullWidth disabled={loading || saving} />
              </Stack>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                <TextField select label="Payload Format" value={urlCallConfig.payloadFormat} onChange={(event) => updateURLCallField('payloadFormat', event.target.value)} fullWidth disabled={loading || saving}>
                  <MenuItem value="json">JSON</MenuItem>
                  <MenuItem value="text">Text</MenuItem>
                </TextField>
                <TextField select label="Submission Binding" value={urlCallConfig.submissionBinding} onChange={(event) => updateURLCallField('submissionBinding', event.target.value)} fullWidth disabled={loading || saving}>
                  <MenuItem value="body">Body</MenuItem>
                  <MenuItem value="query">Query</MenuItem>
                </TextField>
                <TextField select label="Response Body Persistence" value={urlCallConfig.responseBodyPersistence} onChange={(event) => updateURLCallField('responseBodyPersistence', event.target.value)} fullWidth disabled={loading || saving}>
                  <MenuItem value="">Server default</MenuItem>
                  <MenuItem value="filter">Filter</MenuItem>
                  <MenuItem value="save">Save</MenuItem>
                  <MenuItem value="discard">Discard</MenuItem>
                </TextField>
              </Stack>
              <TextField label={urlCallConfig.payloadFormat === 'text' ? 'Payload Text' : 'Payload JSON'} value={urlCallConfig.payloadText} onChange={(event) => updateURLCallField('payloadText', event.target.value)} multiline minRows={8} disabled={loading || saving} />
            </Stack>
          ) : isRequestExchangeJobType(form.jobType) ? (
            <Stack spacing={2}>
              <Typography variant="subtitle2">Request Exchange Configuration</Typography>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                <TextField label="Source System" value={requestExchangeConfig.sourceSystem} onChange={(event) => updateRequestExchangeField('sourceSystem', event.target.value)} fullWidth disabled={loading || saving} />
                <TextField label="Destination Server UID" value={requestExchangeConfig.destinationServerUid} onChange={(event) => updateRequestExchangeField('destinationServerUid', event.target.value)} fullWidth disabled={loading || saving} />
                <TextField label="Additional Destination UIDs" value={requestExchangeConfig.destinationServerUids} onChange={(event) => updateRequestExchangeField('destinationServerUids', event.target.value)} fullWidth disabled={loading || saving} helperText="Comma-separated optional CC destinations." />
              </Stack>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                <TextField label="Batch ID" value={requestExchangeConfig.batchId} onChange={(event) => updateRequestExchangeField('batchId', event.target.value)} fullWidth disabled={loading || saving} />
                <TextField label="Correlation ID" value={requestExchangeConfig.correlationId} onChange={(event) => updateRequestExchangeField('correlationId', event.target.value)} fullWidth disabled={loading || saving} helperText="Blank uses scheduler job/run IDs." />
                <TextField label="Idempotency Key Prefix" value={requestExchangeConfig.idempotencyKeyPrefix} onChange={(event) => updateRequestExchangeField('idempotencyKeyPrefix', event.target.value)} fullWidth disabled={loading || saving} />
              </Stack>
              <TextField label="URL Suffix" value={requestExchangeConfig.urlSuffix} onChange={(event) => updateRequestExchangeField('urlSuffix', event.target.value)} fullWidth disabled={loading || saving} />
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                <TextField select label="Payload Format" value={requestExchangeConfig.payloadFormat} onChange={(event) => updateRequestExchangeField('payloadFormat', event.target.value)} fullWidth disabled={loading || saving}>
                  <MenuItem value="json">JSON</MenuItem>
                  <MenuItem value="text">Text</MenuItem>
                </TextField>
                <TextField select label="Submission Binding" value={requestExchangeConfig.submissionBinding} onChange={(event) => updateRequestExchangeField('submissionBinding', event.target.value)} fullWidth disabled={loading || saving}>
                  <MenuItem value="body">Body</MenuItem>
                  <MenuItem value="query">Query</MenuItem>
                </TextField>
                <TextField select label="Response Body Persistence" value={requestExchangeConfig.responseBodyPersistence} onChange={(event) => updateRequestExchangeField('responseBodyPersistence', event.target.value)} fullWidth disabled={loading || saving}>
                  <MenuItem value="">Server default</MenuItem>
                  <MenuItem value="filter">Filter</MenuItem>
                  <MenuItem value="save">Save</MenuItem>
                  <MenuItem value="discard">Discard</MenuItem>
                </TextField>
              </Stack>
              <TextField label={requestExchangeConfig.payloadFormat === 'text' ? 'Payload Text' : 'Payload JSON'} value={requestExchangeConfig.payloadText} onChange={(event) => updateRequestExchangeField('payloadText', event.target.value)} multiline minRows={8} disabled={loading || saving} />
              <TextField label="Metadata JSON" value={requestExchangeConfig.metadataText} onChange={(event) => updateRequestExchangeField('metadataText', event.target.value)} multiline minRows={4} disabled={loading || saving} />
            </Stack>
          ) : isRapidProReporterSyncJobType(form.jobType) ? (
            <Stack spacing={2}>
              <Typography variant="subtitle2">RapidPro Reporter Sync Configuration</Typography>
              <TextField
                label="Batch Size"
                value={rapidProReporterSyncConfig.batchSize}
                onChange={(event) => setRapidProReporterSyncConfig((current) => ({ ...current, batchSize: event.target.value }))}
                disabled={loading || saving}
              />
              <TextField
                label="Lookback Minutes"
                value={rapidProReporterSyncConfig.lookbackMinutes}
                onChange={(event) => setRapidProReporterSyncConfig((current) => ({ ...current, lookbackMinutes: event.target.value }))}
                disabled={loading || saving}
                helperText="Subtract this overlap window from the last successful sync watermark."
              />
              <FormControlLabel
                control={
                  <Switch
                    checked={rapidProReporterSyncConfig.onlyActive}
                    onChange={(event) => setRapidProReporterSyncConfig((current) => ({ ...current, onlyActive: event.target.checked }))}
                    disabled={loading || saving}
                  />
                }
                label="Only Active Reporters"
              />
              <FormControlLabel
                control={
                  <Switch
                    checked={rapidProReporterSyncConfig.dryRun}
                    onChange={(event) => setRapidProReporterSyncConfig((current) => ({ ...current, dryRun: event.target.checked }))}
                    disabled={loading || saving}
                  />
                }
                label="Dry Run"
              />
            </Stack>
          ) : isDHIS2OrgUnitRefreshJobType(form.jobType) ? (
            <Stack spacing={2}>
              <Typography variant="subtitle2">DHIS2 Org Unit Refresh Configuration</Typography>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                <TextField
                  label="Server Code"
                  value={dhis2OrgUnitRefreshConfig.serverCode}
                  onChange={(event) => setDHIS2OrgUnitRefreshConfig((current) => ({ ...current, serverCode: event.target.value }))}
                  fullWidth
                  disabled={loading || saving}
                  helperText="Recommended when the DHIS2 integration server has a stable code."
                />
                <TextField
                  label="Server UID"
                  value={dhis2OrgUnitRefreshConfig.serverUid}
                  onChange={(event) => setDHIS2OrgUnitRefreshConfig((current) => ({ ...current, serverUid: event.target.value }))}
                  fullWidth
                  disabled={loading || saving}
                  helperText="Optional override when you prefer UID lookup."
                />
              </Stack>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                <TextField
                  label="District Level Name"
                  value={dhis2OrgUnitRefreshConfig.districtLevelName}
                  onChange={(event) => setDHIS2OrgUnitRefreshConfig((current) => ({ ...current, districtLevelName: event.target.value }))}
                  fullWidth
                  disabled={loading || saving}
                />
                <TextField
                  label="District Level Code"
                  value={dhis2OrgUnitRefreshConfig.districtLevelCode}
                  onChange={(event) => setDHIS2OrgUnitRefreshConfig((current) => ({ ...current, districtLevelCode: event.target.value }))}
                  fullWidth
                  disabled={loading || saving}
                  helperText="Optional fallback if level names vary."
                />
              </Stack>
              <Alert severity="warning">
                Full refresh deletes local reporters and user-facility assignments before rebuilding the hierarchy.
              </Alert>
              <FormControlLabel
                control={
                  <Switch
                    checked={dhis2OrgUnitRefreshConfig.fullRefresh}
                    onChange={(event) => setDHIS2OrgUnitRefreshConfig((current) => ({ ...current, fullRefresh: event.target.checked }))}
                    disabled={loading || saving}
                  />
                }
                label="Full Refresh"
              />
              <FormControlLabel
                control={
                  <Switch
                    checked={dhis2OrgUnitRefreshConfig.dryRun}
                    onChange={(event) => setDHIS2OrgUnitRefreshConfig((current) => ({ ...current, dryRun: event.target.checked }))}
                    disabled={loading || saving}
                  />
                }
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
