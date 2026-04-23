import React from 'react'
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Divider,
  List,
  ListItem,
  ListItemText,
  Paper,
  Stack,
  Typography,
} from '@mui/material'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useRouter } from '@tanstack/react-router'
import {
  createApiClient,
  type ApiError,
  type DashboardEventSummary,
  type DashboardKpis,
  type DashboardProcessingGraph,
  type DashboardProcessingGraphPoint,
  type DashboardProcessingGraphStages,
  type DashboardSnapshot,
} from '../api/client'
import { getAccessToken } from '../auth/session'
import { useSessionPrincipal } from '../auth/hooks'
import { canAccessRoute } from '../navigation'
import { hasPermission, hasRole } from '../rbac/permissions'

type QuickAction = {
  label: string
  path: string
  enabled: boolean
}

type AttentionPanelProps = {
  title: string
  subtitle: string
  total: number
  emptyMessage: string
  actionLabel: string
  actionPath: string
  actionSearch?: Record<string, string | undefined>
  actionEnabled: boolean
  children: React.ReactNode
}

type TrendPanelProps = {
  title: string
  subtitle: string
  emptyMessage: string
  children: React.ReactNode
}

type ProcessingStageKey = keyof DashboardProcessingGraphStages

const processingStageConfig: Array<{
  key: ProcessingStageKey
  label: string
  color: string
  requestStatus?: string
}> = [
  { key: 'pending', label: 'Pending', color: '#d97706', requestStatus: 'pending' },
  { key: 'processing', label: 'Processing', color: '#2563eb', requestStatus: 'processing' },
  { key: 'completed', label: 'Completed', color: '#059669', requestStatus: 'completed' },
  { key: 'failed', label: 'Failed', color: '#dc2626', requestStatus: 'failed' },
]

interface DashboardStreamEvent {
  type: string
  timestamp: string
  severity: string
  entityType: string
  entityId?: number
  entityUid?: string
  summary: string
  correlationId?: string
  requestId?: number | null
  deliveryId?: number | null
  jobId?: number | null
  serverId?: number | null
  workerId?: number | null
  patch?: DashboardStreamPatch
  invalidations?: string[]
  payload?: Record<string, unknown>
}

interface DashboardStreamPatch {
  kpi: keyof DashboardKpis
  op: 'increment' | 'decrement'
  value: number
}

type LiveStatus = 'idle' | 'connecting' | 'connected' | 'reconnecting'

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

function formatRelativeWindow(seconds: number) {
  if (seconds % 60 === 0) {
    const minutes = seconds / 60
    return `${minutes} minute${minutes === 1 ? '' : 's'}`
  }
  return `${seconds} seconds`
}

function severityColor(
  severity: string,
): 'default' | 'success' | 'warning' | 'error' | 'info' {
  switch (severity.toLowerCase()) {
    case 'success':
      return 'success'
    case 'warning':
      return 'warning'
    case 'error':
      return 'error'
    case 'info':
      return 'info'
    default:
      return 'default'
  }
}

function healthSeverity(status: string): 'success' | 'warning' | 'error' | 'info' {
  switch (status.toLowerCase()) {
    case 'ok':
      return 'success'
    case 'degraded':
      return 'warning'
    case 'failed':
      return 'error'
    default:
      return 'info'
  }
}

function sumCounts(points: Array<{ count: number }>) {
  return points.reduce((total, point) => total + point.count, 0)
}

function liveStatusColor(status: LiveStatus): 'default' | 'success' | 'warning' | 'info' {
  switch (status) {
    case 'connected':
      return 'success'
    case 'reconnecting':
      return 'warning'
    case 'connecting':
      return 'info'
    default:
      return 'default'
  }
}

function normalizeBaseUrl(baseUrl: string) {
  const trimmed = baseUrl.trim().replace(/\/+$/, '')
  return trimmed.endsWith('/api/v1') ? trimmed.slice(0, -'/api/v1'.length) : trimmed
}

function buildDashboardEventsUrl(baseUrl: string, accessToken: string) {
  const normalized = normalizeBaseUrl(baseUrl)
  if (!normalized) {
    return ''
  }
  const url = new URL(`${normalized}/api/v1/dashboard/operations/events`)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  url.searchParams.set('access_token', accessToken)
  return url.toString()
}

function parseDashboardStreamEvent(input: string): DashboardStreamEvent | null {
  try {
    const parsed = JSON.parse(input) as Partial<DashboardStreamEvent>
    if (!parsed || typeof parsed !== 'object') {
      return null
    }
    if (
      typeof parsed.type !== 'string' ||
      typeof parsed.summary !== 'string' ||
      typeof parsed.timestamp !== 'string'
    ) {
      return null
    }
    return {
      type: parsed.type,
      timestamp: parsed.timestamp,
      severity: typeof parsed.severity === 'string' ? parsed.severity : 'info',
      entityType:
        typeof parsed.entityType === 'string' ? parsed.entityType : 'system',
      entityId: typeof parsed.entityId === 'number' ? parsed.entityId : undefined,
      entityUid:
        typeof parsed.entityUid === 'string' ? parsed.entityUid : undefined,
      summary: parsed.summary,
      correlationId:
        typeof parsed.correlationId === 'string'
          ? parsed.correlationId
          : undefined,
      requestId: typeof parsed.requestId === 'number' ? parsed.requestId : null,
      deliveryId:
        typeof parsed.deliveryId === 'number' ? parsed.deliveryId : null,
      jobId: typeof parsed.jobId === 'number' ? parsed.jobId : null,
      serverId: typeof parsed.serverId === 'number' ? parsed.serverId : null,
      workerId: typeof parsed.workerId === 'number' ? parsed.workerId : null,
      patch: parsed.patch,
      invalidations: Array.isArray(parsed.invalidations)
        ? parsed.invalidations.filter(
            (value): value is string => typeof value === 'string',
          )
        : [],
      payload:
        parsed.payload && typeof parsed.payload === 'object'
          ? parsed.payload
          : undefined,
    }
  } catch {
    return null
  }
}

function applyKpiPatch(kpis: DashboardKpis, patch: DashboardStreamPatch): DashboardKpis {
  const current = Number(kpis[patch.kpi] ?? 0)
  const delta = Math.max(0, patch.value)
  const nextValue =
    patch.op === 'decrement' ? Math.max(0, current - delta) : current + delta
  return {
    ...kpis,
    [patch.kpi]: nextValue,
  }
}

function toRecentEventSummary(event: DashboardStreamEvent): DashboardEventSummary {
  return {
    type: event.type,
    timestamp: event.timestamp,
    severity: event.severity,
    entityType: event.entityType,
    entityId: event.entityId,
    entityUid: event.entityUid,
    summary: event.summary,
    correlationId: event.correlationId,
    requestId: event.requestId,
    deliveryId: event.deliveryId,
    jobId: event.jobId,
    workerId: event.workerId,
  }
}

function eventKey(event: DashboardEventSummary) {
  return [
    event.type,
    event.timestamp,
    event.entityId ?? '',
    event.entityUid ?? '',
    event.summary,
  ].join(':')
}

function totalProcessingStage(
  series: DashboardProcessingGraphPoint[],
  key: ProcessingStageKey,
) {
  return series.reduce((total, point) => total + Number(point.stages[key] ?? 0), 0)
}

function formatBucketLabel(value: string) {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.valueOf())) {
    return value
  }
  return parsed.toLocaleTimeString([], { hour: 'numeric' })
}

function applyDashboardStreamEvent(
  snapshot: DashboardSnapshot,
  event: DashboardStreamEvent,
): DashboardSnapshot {
  const nextEvent = toRecentEventSummary(event)
  const recentEvents = [
    nextEvent,
    ...snapshot.recentEvents.filter(
      (item) => eventKey(item) !== eventKey(nextEvent),
    ),
  ].slice(0, Math.max(snapshot.recentEvents.length, 10))

  return {
    ...snapshot,
    generatedAt: event.timestamp,
    kpis: event.patch ? applyKpiPatch(snapshot.kpis, event.patch) : snapshot.kpis,
    recentEvents,
  }
}

function toObservabilitySearch(event: DashboardEventSummary) {
  return {
    eventType: event.type,
    correlationId: event.correlationId || undefined,
    requestId: event.requestId ? String(event.requestId) : undefined,
    deliveryId: event.deliveryId ? String(event.deliveryId) : undefined,
    jobId: event.jobId ? String(event.jobId) : undefined,
    workerId: event.workerId ? String(event.workerId) : undefined,
  }
}

function DashboardSection({
  title,
  subtitle,
  action,
  children,
}: {
  title: string
  subtitle?: string
  action?: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <Paper elevation={1} sx={{ p: 3 }}>
      <Stack
        direction={{ xs: 'column', md: 'row' }}
        spacing={1.5}
        justifyContent="space-between"
        alignItems={{ md: 'center' }}
      >
        <Box>
          <Typography variant="h6" component="h2">
            {title}
          </Typography>
          {subtitle ? (
            <Typography variant="body2" color="text.secondary">
              {subtitle}
            </Typography>
          ) : null}
        </Box>
        {action}
      </Stack>
      <Box sx={{ mt: 2.5 }}>{children}</Box>
    </Paper>
  )
}

function KpiCard({
  label,
  value,
  helper,
}: {
  label: string
  value: number
  helper?: string
}) {
  return (
    <Paper variant="outlined" sx={{ p: 2.25, minHeight: 132 }}>
      <Stack spacing={1}>
        <Typography variant="body2" color="text.secondary">
          {label}
        </Typography>
        <Typography variant="h4" component="div">
          {value.toLocaleString()}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {helper ?? 'Current snapshot'}
        </Typography>
      </Stack>
    </Paper>
  )
}

function AttentionPanel({
  title,
  subtitle,
  total,
  emptyMessage,
  actionLabel,
  actionPath,
  actionSearch,
  actionEnabled,
  children,
}: AttentionPanelProps) {
  const navigate = useNavigate()

  return (
    <DashboardSection
      title={title}
      subtitle={subtitle}
      action={
        <Button
          variant="outlined"
          onClick={() =>
            void navigate({
              to: actionPath,
              search: actionSearch ?? {},
            })
          }
          disabled={!actionEnabled}
        >
          {actionLabel}
        </Button>
      }
    >
      <Stack spacing={2}>
        <Chip
          label={`${total} needing attention`}
          color={total > 0 ? 'warning' : 'default'}
          sx={{ width: 'fit-content' }}
        />
        {total === 0 ? (
          <Typography variant="body2" color="text.secondary">
            {emptyMessage}
          </Typography>
        ) : (
          children
        )}
      </Stack>
    </DashboardSection>
  )
}

function TrendPanel({ title, subtitle, emptyMessage, children }: TrendPanelProps) {
  return (
    <DashboardSection title={title} subtitle={subtitle}>
      <Box>
        {React.Children.count(children) === 0 ? (
          <Typography variant="body2" color="text.secondary">
            {emptyMessage}
          </Typography>
        ) : (
          children
        )}
      </Box>
    </DashboardSection>
  )
}

function ProcessingGraphPanel({
  graph,
  actionEnabled,
  onStageClick,
}: {
  graph: DashboardProcessingGraph
  actionEnabled: boolean
  onStageClick: (stage: ProcessingStageKey) => void
}) {
  const visibleSeries = graph.series.slice(-12)
  const stageTotals = processingStageConfig.map((stage) => ({
    ...stage,
    total: totalProcessingStage(graph.series, stage.key),
  }))

  return (
    <TrendPanel
      title="Request Processing Flow"
      subtitle="Live request pipeline activity over the recent dashboard window."
      emptyMessage="No request processing data is available yet."
    >
      <Stack spacing={2}>
        <Typography variant="body2" color="text.secondary">
          {graph.windowHours} hour window in {graph.bucketSizeMinutes}-minute buckets. Click a stage to open the matching request list.
        </Typography>
        <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
          {stageTotals.map((stage) => (
            <Button
              key={stage.key}
              variant="outlined"
              onClick={() => onStageClick(stage.key)}
              disabled={!actionEnabled || !stage.requestStatus}
              sx={{ borderColor: stage.color, color: stage.color }}
            >
              {stage.label}: {stage.total.toLocaleString()}
            </Button>
          ))}
        </Stack>
        <Box
          sx={{
            display: 'grid',
            gridTemplateColumns: `repeat(${Math.max(visibleSeries.length, 1)}, minmax(0, 1fr))`,
            gap: 1,
            alignItems: 'end',
            minHeight: 220,
          }}
        >
          {visibleSeries.map((point) => {
            const total = processingStageConfig.reduce(
              (sum, stage) => sum + Number(point.stages[stage.key] ?? 0),
              0,
            )
            const denominator = Math.max(total, 1)
            return (
              <Stack key={point.bucketStart} spacing={1} alignItems="center">
                <Stack
                  direction="column-reverse"
                  sx={{
                    width: '100%',
                    maxWidth: 36,
                    height: 160,
                    borderRadius: 999,
                    overflow: 'hidden',
                    bgcolor: 'action.hover',
                  }}
                >
                  {processingStageConfig.map((stage) => {
                    const value = Number(point.stages[stage.key] ?? 0)
                    if (value <= 0) {
                      return null
                    }
                    return (
                      <Box
                        key={stage.key}
                        title={`${stage.label}: ${value}`}
                        sx={{
                          height: `${(value / denominator) * 100}%`,
                          bgcolor: stage.color,
                        }}
                      />
                    )
                  })}
                </Stack>
                <Typography variant="caption" color="text.secondary">
                  {formatBucketLabel(point.bucketStart)}
                </Typography>
                <Typography variant="caption" fontWeight={600}>
                  {total.toLocaleString()}
                </Typography>
              </Stack>
            )
          })}
        </Box>
      </Stack>
    </TrendPanel>
  )
}

export function DashboardPage() {
  const principal = useSessionPrincipal()
  const navigate = useNavigate()
  const router = useRouter()
  const queryClient = useQueryClient()
  const [liveStatus, setLiveStatus] = React.useState<LiveStatus>('idle')

  const apiClient = React.useMemo(
    () =>
      createApiClient({
        getSettings: () => router.options.context.settingsStore.loadSettings(),
      }),
    [router.options.context.settingsStore],
  )

  const canReadOperations =
    hasPermission(principal, 'observability.read') &&
    canAccessRoute(principal, '/observability')

  const quickActions = React.useMemo<QuickAction[]>(
    () =>
      [
        {
          label: 'Requests',
          path: '/requests',
          enabled:
            (hasPermission(principal, 'requests.read') ||
              hasPermission(principal, 'requests.write')) &&
            canAccessRoute(principal, '/requests'),
        },
        {
          label: 'Deliveries',
          path: '/deliveries',
          enabled:
            (hasPermission(principal, 'deliveries.read') ||
              hasPermission(principal, 'deliveries.write')) &&
            canAccessRoute(principal, '/deliveries'),
        },
        {
          label: 'Jobs',
          path: '/jobs',
          enabled:
            (hasPermission(principal, 'jobs.read') ||
              hasPermission(principal, 'jobs.write')) &&
            canAccessRoute(principal, '/jobs'),
        },
        {
          label: 'Observability',
          path: '/observability',
          enabled:
            hasPermission(principal, 'observability.read') &&
            canAccessRoute(principal, '/observability'),
        },
        {
          label: 'Users',
          path: '/users',
          enabled:
            (hasPermission(principal, 'users.read') ||
              hasPermission(principal, 'users.write')) &&
            canAccessRoute(principal, '/users'),
        },
        {
          label: 'Roles',
          path: '/roles',
          enabled:
            (hasPermission(principal, 'users.read') ||
              hasPermission(principal, 'users.write')) &&
            canAccessRoute(principal, '/roles'),
        },
        {
          label: 'Permissions',
          path: '/permissions',
          enabled:
            (hasPermission(principal, 'users.read') ||
              hasPermission(principal, 'users.write')) &&
            canAccessRoute(principal, '/permissions'),
        },
        {
          label: 'Audit Log',
          path: '/audit',
          enabled:
            hasPermission(principal, 'audit.read') &&
            canAccessRoute(principal, '/audit'),
        },
        {
          label: 'Settings',
          path: '/settings',
          enabled:
            (hasRole(principal, 'admin') ||
              hasPermission(principal, 'settings.write')) &&
            canAccessRoute(principal, '/settings/general'),
        },
      ].filter((action) => action.enabled),
    [principal],
  )

  const snapshotQuery = useQuery<DashboardSnapshot, ApiError>({
    queryKey: ['dashboard', 'operations'],
    queryFn: () => apiClient.getOperationsDashboard(),
    enabled: canReadOperations,
    retry: false,
  })

  React.useEffect(() => {
    if (!canReadOperations || !snapshotQuery.isSuccess) {
      setLiveStatus('idle')
      return
    }

    let reconnectTimer: number | null = null
    let refetchTimer: number | null = null
    let closedByCleanup = false
    let socket: WebSocket | null = null

    const scheduleRefetch = () => {
      if (refetchTimer !== null) {
        window.clearTimeout(refetchTimer)
      }
      refetchTimer = window.setTimeout(() => {
        void queryClient.invalidateQueries({
          queryKey: ['dashboard', 'operations'],
        })
      }, 1000)
    }

    const connect = async (isReconnect: boolean) => {
      const accessToken = getAccessToken()
      if (!accessToken) {
        setLiveStatus('idle')
        return
      }

      const settings = await router.options.context.settingsStore.loadSettings()
      const url = buildDashboardEventsUrl(settings.apiBaseUrl, accessToken)
      if (!url) {
        setLiveStatus('idle')
        return
      }

      setLiveStatus(isReconnect ? 'reconnecting' : 'connecting')
      socket = new WebSocket(url)

      socket.onopen = () => {
        setLiveStatus('connected')
        if (isReconnect) {
          void queryClient.invalidateQueries({
            queryKey: ['dashboard', 'operations'],
          })
        }
      }

      socket.onmessage = (message) => {
        const event = parseDashboardStreamEvent(String(message.data ?? ''))
        if (!event) {
          return
        }

        queryClient.setQueryData<DashboardSnapshot>(
          ['dashboard', 'operations'],
          (current) =>
            current ? applyDashboardStreamEvent(current, event) : current,
        )

        const needsSnapshotRefresh = (event.invalidations ?? []).some(
          (value) => value !== 'recentEvents' && value !== 'kpis',
        )
        if (needsSnapshotRefresh) {
          scheduleRefetch()
        }
      }

      socket.onclose = () => {
        if (closedByCleanup) {
          return
        }
        setLiveStatus('reconnecting')
        reconnectTimer = window.setTimeout(() => {
          void connect(true)
        }, 2000)
      }

      socket.onerror = () => {
        socket?.close()
      }
    }

    void connect(false)

    return () => {
      closedByCleanup = true
      if (reconnectTimer !== null) {
        window.clearTimeout(reconnectTimer)
      }
      if (refetchTimer !== null) {
        window.clearTimeout(refetchTimer)
      }
      socket?.close()
    }
  }, [
    canReadOperations,
    queryClient,
    router.options.context.settingsStore,
    snapshotQuery.isSuccess,
  ])

  const snapshot = snapshotQuery.data
  const requestsTotal = snapshot ? sumCounts(snapshot.trends.requestsByHour) : 0
  const deliveryTrendStatuses = snapshot
    ? Array.from(
        new Set(snapshot.trends.deliveriesByStatus.map((point) => point.status)),
      )
    : []
  const jobTrendStates = snapshot
    ? Array.from(new Set(snapshot.trends.jobsByState.map((point) => point.status)))
    : []

  return (
    <Stack spacing={3}>
      <Paper elevation={1} sx={{ p: 3 }}>
        <Stack spacing={2}>
          <Stack
            direction={{ xs: 'column', md: 'row' }}
            justifyContent="space-between"
            spacing={2}
          >
            <Box>
              <Typography variant="h5" component="h1" gutterBottom>
                Dashboard
              </Typography>
              <Typography color="text.secondary">
                Snapshot-driven operations view for Sukumad requests, deliveries,
                jobs, ingest, and worker health.
              </Typography>
            </Box>
            {snapshot ? (
              <Stack spacing={1} alignItems={{ md: 'flex-end' }}>
                <Chip
                  label={`Health: ${snapshot.health.status}`}
                  color={healthSeverity(snapshot.health.status)}
                  sx={{ textTransform: 'capitalize' }}
                />
                <Typography variant="body2" color="text.secondary">
                  Generated {formatDateTime(snapshot.generatedAt)}
                </Typography>
                {canReadOperations ? (
                  <Chip
                    label={`Live: ${liveStatus}`}
                    color={liveStatusColor(liveStatus)}
                    variant="outlined"
                    sx={{ textTransform: 'capitalize' }}
                  />
                ) : null}
              </Stack>
            ) : null}
          </Stack>

          <Stack direction="row" spacing={1.5} useFlexGap flexWrap="wrap">
            {quickActions.map((item) => (
              <Button
                key={item.label}
                variant="contained"
                onClick={() => void navigate({ to: item.path })}
              >
                {item.label}
              </Button>
            ))}
          </Stack>

          {!canReadOperations ? (
            <Alert severity="info">
              Operations widgets require <strong>observability.read</strong>. You
              can still use the available pages from the existing navigation.
            </Alert>
          ) : null}

          {snapshotQuery.isLoading ? (
            <Stack direction="row" spacing={1.5} alignItems="center">
              <CircularProgress size={24} />
              <Typography color="text.secondary">
                Loading operations snapshot...
              </Typography>
            </Stack>
          ) : null}

          {snapshotQuery.isError ? (
            <Alert
              severity="error"
              action={
                <Button
                  color="inherit"
                  size="small"
                  onClick={() => void snapshotQuery.refetch()}
                >
                  Retry
                </Button>
              }
            >
              {snapshotQuery.error.message ||
                'Unable to load operations dashboard snapshot.'}
            </Alert>
          ) : null}

          {snapshot && snapshot.health.signals.length > 0 ? (
            <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
              {snapshot.health.signals.map((signal) => (
                <Chip
                  key={signal}
                  label={signal}
                  variant="outlined"
                  color={healthSeverity(snapshot.health.status)}
                />
              ))}
            </Stack>
          ) : null}
        </Stack>
      </Paper>

      {snapshot ? (
        <>
          <Box
            sx={{
              display: 'grid',
              gap: 2,
              gridTemplateColumns: {
                xs: '1fr',
                sm: 'repeat(2, minmax(0, 1fr))',
                lg: 'repeat(3, minmax(0, 1fr))',
                xl: 'repeat(4, minmax(0, 1fr))',
              },
            }}
          >
            <KpiCard
              label="Requests Today"
              value={snapshot.kpis.requestsToday}
              helper={`${requestsTotal} requests across the hourly trend window`}
            />
            <KpiCard
              label="Pending Requests"
              value={snapshot.kpis.pendingRequests}
            />
            <KpiCard
              label="Pending Deliveries"
              value={snapshot.kpis.pendingDeliveries}
            />
            <KpiCard
              label="Running Deliveries"
              value={snapshot.kpis.runningDeliveries}
            />
            <KpiCard
              label="Failed Deliveries Last Hour"
              value={snapshot.kpis.failedDeliveriesLastHour}
              helper="Fast failure pressure indicator"
            />
            <KpiCard label="Polling Jobs" value={snapshot.kpis.pollingJobs} />
            <KpiCard label="Ingest Backlog" value={snapshot.kpis.ingestBacklog} />
            <KpiCard
              label="Workers"
              value={snapshot.kpis.healthyWorkers}
              helper={`${snapshot.kpis.unhealthyWorkers.toLocaleString()} unhealthy workers in the current snapshot`}
            />
          </Box>

          <Box
            sx={{
              display: 'grid',
              gap: 2,
              gridTemplateColumns: {
                xs: '1fr',
                lg: 'minmax(0, 1.2fr) minmax(0, 0.8fr)',
              },
            }}
          >
            <DashboardSection
              title="Failed Deliveries"
              subtitle="Most recent delivery failures that should send the operator into the deliveries workflow."
              action={
                <Button
                  variant="outlined"
                  onClick={() =>
                    void navigate({
                      to: '/deliveries',
                      search: { status: 'failed' },
                    })
                  }
                  disabled={!canAccessRoute(principal, '/deliveries')}
                >
                  Open Failed Deliveries
                </Button>
              }
            >
              <Stack spacing={2}>
                <Chip
                  label={`${snapshot.attention.failedDeliveries.total} open failures`}
                  color={
                    snapshot.attention.failedDeliveries.total > 0
                      ? 'error'
                      : 'default'
                  }
                  sx={{ width: 'fit-content' }}
                />
                {snapshot.attention.failedDeliveries.items.length === 0 ? (
                  <Typography variant="body2" color="text.secondary">
                    No failed deliveries are waiting for action.
                  </Typography>
                ) : (
                  <List disablePadding>
                    {snapshot.attention.failedDeliveries.items.map((item, index) => (
                      <React.Fragment key={item.id}>
                        <ListItem disableGutters alignItems="flex-start">
                          <ListItemText
                            primary={`${item.uid} • ${item.serverName}`}
                            secondary={
                              <>
                                <Typography
                                  component="span"
                                  variant="body2"
                                  color="text.primary"
                                >
                                  {item.errorMessage || item.status}
                                </Typography>
                                {` • Request ${item.requestUid} • Updated ${formatDateTime(item.updatedAt)}`}
                              </>
                            }
                          />
                        </ListItem>
                        {index < snapshot.attention.failedDeliveries.items.length - 1 ? (
                          <Divider component="li" />
                        ) : null}
                      </React.Fragment>
                    ))}
                  </List>
                )}
              </Stack>
            </DashboardSection>

            <DashboardSection
              title="Worker Health"
              subtitle={`Heartbeat freshness window: ${formatRelativeWindow(snapshot.workers.heartbeatFreshnessSeconds)}.`}
              action={
                <Button
                  variant="outlined"
                  onClick={() => void navigate({ to: '/observability' })}
                  disabled={!canAccessRoute(principal, '/observability')}
                >
                  Open Observability
                </Button>
              }
            >
              <Stack spacing={1.5}>
                {snapshot.workers.items.length === 0 ? (
                  <Typography variant="body2" color="text.secondary">
                    No worker activity is currently reported.
                  </Typography>
                ) : (
                  snapshot.workers.items.map((item) => (
                    <Paper key={item.id} variant="outlined" sx={{ p: 1.5 }}>
                      <Stack
                        direction={{ xs: 'column', sm: 'row' }}
                        spacing={1.5}
                        justifyContent="space-between"
                      >
                        <Box>
                          <Typography variant="subtitle2">
                            {item.workerName}
                          </Typography>
                          <Typography variant="body2" color="text.secondary">
                            {item.workerType} • {item.uid}
                          </Typography>
                        </Box>
                        <Stack alignItems={{ sm: 'flex-end' }} spacing={0.5}>
                          <Chip
                            label={item.status}
                            size="small"
                            color={
                              item.status === 'running' ? 'success' : 'warning'
                            }
                          />
                          <Typography variant="caption" color="text.secondary">
                            Last heartbeat {formatDateTime(item.lastHeartbeatAt)}
                          </Typography>
                        </Stack>
                      </Stack>
                    </Paper>
                  ))
                )}
              </Stack>
            </DashboardSection>
          </Box>

          <Box
            sx={{
              display: 'grid',
              gap: 2,
              gridTemplateColumns: {
                xs: '1fr',
                xl: 'repeat(2, minmax(0, 1fr))',
              },
            }}
          >
            <AttentionPanel
              title="Stale Running Deliveries"
              subtitle="Deliveries that have been running longer than expected."
              total={snapshot.attention.staleRunningDeliveries.total}
              emptyMessage="No stale running deliveries were detected."
              actionLabel="Open Deliveries"
              actionPath="/deliveries"
              actionSearch={{ status: 'running' }}
              actionEnabled={canAccessRoute(principal, '/deliveries')}
            >
              <List disablePadding>
                {snapshot.attention.staleRunningDeliveries.items.map(
                  (item, index) => (
                    <React.Fragment key={item.id}>
                      <ListItem disableGutters>
                        <ListItemText
                          primary={`${item.uid} • ${item.serverName}`}
                          secondary={`Started ${formatDateTime(item.startedAt)} • Updated ${formatDateTime(item.updatedAt)}`}
                        />
                      </ListItem>
                      {index < snapshot.attention.staleRunningDeliveries.items.length - 1 ? (
                        <Divider component="li" />
                      ) : null}
                    </React.Fragment>
                  ),
                )}
              </List>
            </AttentionPanel>

            <AttentionPanel
              title="Stuck Jobs"
              subtitle="Polling jobs that have not progressed on schedule."
              total={snapshot.attention.stuckJobs.total}
              emptyMessage="No stuck jobs were detected."
              actionLabel="Open Jobs"
              actionPath="/jobs"
              actionSearch={{ status: 'polling' }}
              actionEnabled={canAccessRoute(principal, '/jobs')}
            >
              <List disablePadding>
                {snapshot.attention.stuckJobs.items.map((item, index) => (
                  <React.Fragment key={item.id}>
                    <ListItem disableGutters>
                      <ListItemText
                        primary={`${item.uid} • ${item.currentState}`}
                        secondary={`Remote ${item.remoteStatus || 'unknown'} • Next poll ${formatDateTime(item.nextPollAt)}`}
                      />
                    </ListItem>
                    {index < snapshot.attention.stuckJobs.items.length - 1 ? (
                      <Divider component="li" />
                    ) : null}
                  </React.Fragment>
                ))}
              </List>
            </AttentionPanel>

            <AttentionPanel
              title="Recent Ingest Failures"
              subtitle="Failed file ingest attempts from the recent backlog window."
              total={snapshot.attention.recentIngestFailures.total}
              emptyMessage="No recent ingest failures were reported."
              actionLabel="Open Observability"
              actionPath="/observability"
              actionSearch={{ eventType: 'ingest.failed', level: 'error' }}
              actionEnabled={canAccessRoute(principal, '/observability')}
            >
              <List disablePadding>
                {snapshot.attention.recentIngestFailures.items.map(
                  (item, index) => (
                    <React.Fragment key={item.id}>
                      <ListItem disableGutters>
                        <ListItemText
                          primary={`${item.originalName} • ${item.status}`}
                          secondary={`${item.lastErrorCode || item.lastErrorMessage || 'Unknown error'} • Updated ${formatDateTime(item.updatedAt)}`}
                        />
                      </ListItem>
                      {index < snapshot.attention.recentIngestFailures.items.length - 1 ? (
                        <Divider component="li" />
                      ) : null}
                    </React.Fragment>
                  ),
                )}
              </List>
            </AttentionPanel>

            <AttentionPanel
              title="Unhealthy Workers"
              subtitle="Workers that need restart, investigation, or closer observation."
              total={snapshot.attention.unhealthyWorkers.total}
              emptyMessage="All reported workers are healthy."
              actionLabel="Open Observability"
              actionPath="/observability"
              actionSearch={{ level: 'error' }}
              actionEnabled={canAccessRoute(principal, '/observability')}
            >
              <List disablePadding>
                {snapshot.attention.unhealthyWorkers.items.map((item, index) => (
                  <React.Fragment key={item.id}>
                    <ListItem disableGutters>
                      <ListItemText
                        primary={`${item.workerName} • ${item.status}`}
                        secondary={`Heartbeat ${formatDateTime(item.lastHeartbeatAt)} • Updated ${formatDateTime(item.updatedAt)}`}
                      />
                    </ListItem>
                    {index < snapshot.attention.unhealthyWorkers.items.length - 1 ? (
                      <Divider component="li" />
                    ) : null}
                  </React.Fragment>
                ))}
              </List>
            </AttentionPanel>
          </Box>

          <Box
            sx={{
              display: 'grid',
              gap: 2,
              gridTemplateColumns: {
                xs: '1fr',
                xl: 'repeat(2, minmax(0, 1fr))',
              },
            }}
          >
            <ProcessingGraphPanel
              graph={snapshot.processingGraph}
              actionEnabled={canAccessRoute(principal, '/requests')}
              onStageClick={(stage) => {
                const requestStatus = processingStageConfig.find(
                  (item) => item.key === stage,
                )?.requestStatus
                if (!requestStatus) {
                  return
                }
                void navigate({
                  to: '/requests',
                  search: { status: requestStatus },
                })
              }}
            />

            <TrendPanel
              title="Deliveries by Status"
              subtitle="Status mix across the recent delivery trend window."
              emptyMessage="No delivery trend data is available yet."
            >
              <Stack spacing={1.25}>
                <Typography variant="body2" color="text.secondary">
                  {deliveryTrendStatuses.length} statuses represented in the
                  snapshot.
                </Typography>
                {deliveryTrendStatuses.map((status) => (
                  <Paper key={status} variant="outlined" sx={{ p: 1.25 }}>
                    <Stack
                      direction="row"
                      justifyContent="space-between"
                      spacing={2}
                    >
                      <Typography
                        variant="body2"
                        sx={{ textTransform: 'capitalize' }}
                      >
                        {status}
                      </Typography>
                      <Typography variant="subtitle2">
                        {snapshot.trends.deliveriesByStatus
                          .filter((point) => point.status === status)
                          .reduce((total, point) => total + point.count, 0)
                          .toLocaleString()}
                      </Typography>
                    </Stack>
                  </Paper>
                ))}
              </Stack>
            </TrendPanel>

            <TrendPanel
              title="Jobs by State"
              subtitle="Current polling and terminal state mix from the trend snapshot."
              emptyMessage="No job state trend data is available yet."
            >
              <Stack spacing={1.25}>
                <Typography variant="body2" color="text.secondary">
                  {jobTrendStates.length} job states represented in the snapshot.
                </Typography>
                {jobTrendStates.map((status) => (
                  <Paper key={status} variant="outlined" sx={{ p: 1.25 }}>
                    <Stack
                      direction="row"
                      justifyContent="space-between"
                      spacing={2}
                    >
                      <Typography
                        variant="body2"
                        sx={{ textTransform: 'capitalize' }}
                      >
                        {status}
                      </Typography>
                      <Typography variant="subtitle2">
                        {snapshot.trends.jobsByState
                          .filter((point) => point.status === status)
                          .reduce((total, point) => total + point.count, 0)
                          .toLocaleString()}
                      </Typography>
                    </Stack>
                  </Paper>
                ))}
              </Stack>
            </TrendPanel>

            <TrendPanel
              title="Failures by Server"
              subtitle="Servers generating the highest recent failure counts."
              emptyMessage="No server-specific failures are in the current snapshot."
            >
              <Stack spacing={1.25}>
                {snapshot.trends.failuresByServer.map((point) => (
                  <Paper
                    key={`${point.serverId}-${point.serverName}`}
                    variant="outlined"
                    sx={{ p: 1.25 }}
                  >
                    <Stack
                      direction="row"
                      justifyContent="space-between"
                      spacing={2}
                    >
                      <Typography variant="body2">{point.serverName}</Typography>
                      <Typography variant="subtitle2">
                        {point.count.toLocaleString()}
                      </Typography>
                    </Stack>
                  </Paper>
                ))}
              </Stack>
            </TrendPanel>
          </Box>

          <DashboardSection
            title="Recent Events"
            subtitle="Latest operational events from the snapshot feed."
          >
            {snapshot.recentEvents.length === 0 ? (
              <Typography variant="body2" color="text.secondary">
                No recent operational events are available.
              </Typography>
            ) : (
              <List disablePadding>
                {snapshot.recentEvents.map((event, index) => (
                  <React.Fragment
                    key={`${event.type}-${event.timestamp}-${index}`}
                  >
                    <ListItem disableGutters alignItems="flex-start">
                      <Stack spacing={1} sx={{ width: '100%' }}>
                        <Stack
                          direction={{ xs: 'column', sm: 'row' }}
                          spacing={1}
                          justifyContent="space-between"
                        >
                          <Stack
                            direction="row"
                            spacing={1}
                            alignItems="center"
                            useFlexGap
                            flexWrap="wrap"
                          >
                            <Chip
                              label={event.severity}
                              size="small"
                              color={severityColor(event.severity)}
                            />
                            <Typography variant="subtitle2">
                              {event.summary}
                            </Typography>
                          </Stack>
                          <Stack
                            direction={{ xs: 'column', sm: 'row' }}
                            spacing={1}
                            alignItems={{ sm: 'center' }}
                          >
                            <Typography variant="caption" color="text.secondary">
                              {formatDateTime(event.timestamp)}
                            </Typography>
                            {(event.correlationId ||
                              event.requestId ||
                              event.deliveryId ||
                              event.jobId ||
                              event.workerId) &&
                            canAccessRoute(principal, '/observability') ? (
                              <Button
                                size="small"
                                onClick={() =>
                                  void navigate({
                                    to: '/observability',
                                    search: toObservabilitySearch(event),
                                  })
                                }
                              >
                                Trace
                              </Button>
                            ) : null}
                          </Stack>
                        </Stack>
                        <Typography variant="body2" color="text.secondary">
                          {event.type}
                          {event.entityUid ? ` • ${event.entityUid}` : ''}
                          {event.correlationId
                            ? ` • Correlation ${event.correlationId}`
                            : ''}
                        </Typography>
                      </Stack>
                    </ListItem>
                    {index < snapshot.recentEvents.length - 1 ? (
                      <Divider component="li" />
                    ) : null}
                  </React.Fragment>
                ))}
              </List>
            )}
          </DashboardSection>
        </>
      ) : null}
    </Stack>
  )
}
