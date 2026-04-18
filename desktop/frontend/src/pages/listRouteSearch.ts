export interface RequestsRouteSearch {
  q?: string
  status?: string
}

export interface DeliveriesRouteSearch {
  q?: string
  status?: string
  server?: string
  date?: string
}

export interface JobsRouteSearch {
  q?: string
  status?: string
}

export interface SchedulerRouteSearch {
  q?: string
  category?: string
}

export interface ObservabilityRouteSearch {
  eventType?: string
  level?: string
  correlationId?: string
  from?: string
  to?: string
  requestId?: string
  deliveryId?: string
  jobId?: string
  workerId?: string
}

function readString(search: Record<string, unknown>, key: string) {
  const value = search[key]
  return typeof value === 'string' ? value.trim() : ''
}

function toOptional(value: string) {
  return value || undefined
}

export function normalizeDeliveriesRouteSearch(search: Record<string, unknown>): DeliveriesRouteSearch {
  return {
    q: toOptional(readString(search, 'q')),
    status: toOptional(readString(search, 'status')),
    server: toOptional(readString(search, 'server')),
    date: toOptional(readString(search, 'date')),
  }
}

export function normalizeRequestsRouteSearch(search: Record<string, unknown>): RequestsRouteSearch {
  return {
    q: toOptional(readString(search, 'q')),
    status: toOptional(readString(search, 'status')),
  }
}

export function normalizeJobsRouteSearch(search: Record<string, unknown>): JobsRouteSearch {
  return {
    q: toOptional(readString(search, 'q')),
    status: toOptional(readString(search, 'status')),
  }
}

export function normalizeSchedulerRouteSearch(search: Record<string, unknown>): SchedulerRouteSearch {
  return {
    q: toOptional(readString(search, 'q')),
    category: toOptional(readString(search, 'category')),
  }
}

export function normalizeObservabilityRouteSearch(search: Record<string, unknown>): ObservabilityRouteSearch {
  return {
    eventType: toOptional(readString(search, 'eventType')),
    level: toOptional(readString(search, 'level')),
    correlationId: toOptional(readString(search, 'correlationId')),
    from: toOptional(readString(search, 'from')),
    to: toOptional(readString(search, 'to')),
    requestId: toOptional(readString(search, 'requestId')),
    deliveryId: toOptional(readString(search, 'deliveryId')),
    jobId: toOptional(readString(search, 'jobId')),
    workerId: toOptional(readString(search, 'workerId')),
  }
}
