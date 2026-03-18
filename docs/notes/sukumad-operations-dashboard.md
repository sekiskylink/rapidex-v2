# Sukumad Operations Dashboard

## Purpose

This note defines the implementation plan for adding a real operations dashboard to Sukumad across:

- backend API
- web client
- desktop client

The dashboard must reuse the existing BasePro shell, RBAC, module enablement, and Sukumad domain model.

It must not introduce:

- a second app shell
- a parallel router hierarchy
- a separate auth model
- direct desktop-to-database access

The dashboard should combine:

- snapshot APIs for initial page load
- websocket updates for fast-changing operational state

Relevant code areas:

- `backend/internal/sukumad/request`
- `backend/internal/sukumad/delivery`
- `backend/internal/sukumad/async`
- `backend/internal/sukumad/worker`
- `backend/internal/sukumad/ingest`
- `backend/internal/audit`
- `backend/internal/rbac`
- `backend/internal/middleware`
- `web/src/pages`
- `desktop/frontend/src/pages`

## Outcome

The resulting dashboard should be the primary operator landing page for Sukumad.

It should answer four questions immediately:

1. What is happening now?
2. What is failing now?
3. Are workers and ingest flows healthy?
4. Where should the operator drill down next?

## Dashboard shape

The dashboard should contain these sections.

### 1. KPI row

- requests received today
- pending requests
- pending deliveries
- running deliveries
- failed deliveries in the last hour
- polling jobs
- ingest backlog
- healthy workers / unhealthy workers

### 2. Live event feed

Most recent operational events, such as:

- request accepted
- delivery started
- delivery succeeded
- delivery failed
- retry scheduled
- job polling started
- job completed
- worker stopped or failed
- ingest file failed or retried

### 3. Attention panels

- failed deliveries needing retry
- stale running deliveries
- stuck polling jobs
- recent ingest failures
- unhealthy workers

### 4. Trend widgets

- requests by hour
- deliveries by status over time
- jobs by state over time
- failures by server

### 5. Drill-down actions

Each widget should link into existing pages rather than duplicating full workflows:

- `/requests`
- `/deliveries`
- `/jobs`
- `/observability`

## Architectural approach

The dashboard should follow a hybrid delivery model.

### Snapshot APIs

Snapshot APIs remain the source of truth for:

- initial page load
- reconnect recovery
- periodic consistency refresh
- chart/trend data
- attention-panel table contents

### Websocket stream

The websocket stream should be used for:

- live KPI adjustments
- live event feed updates
- worker heartbeat/status changes
- notification-worthy state changes
- invalidation hints for affected widgets

The websocket stream should not attempt to replace the existing paginated HTTP list endpoints.

That means:

- requests, deliveries, jobs, and observability grids stay HTTP-driven
- websocket events update summary widgets or trigger targeted refetch

## Backend plan

### Module placement

Create a new Sukumad module under:

- `backend/internal/sukumad/dashboard`

Expected files:

- `handler.go`
- `service.go`
- `repository.go`
- `types.go`

If websocket hub logic is substantial, keep it inside the same module instead of creating a new top-level `internal/` package.

Possible internal helper files:

- `stream.go`
- `stream_test.go`
- `mapper.go`

### Endpoints

Add these endpoints under `/api/v1`:

- `GET /dashboard/operations`
- `GET /dashboard/operations/events`

Recommended endpoint behavior:

- `GET /dashboard/operations`
  - returns the full initial dashboard snapshot
  - includes KPI values, trend series, attention lists, and worker summary
- `GET /dashboard/operations/events`
  - upgrades to websocket
  - sends authenticated operational events
  - may also send lightweight counter patches or invalidation events

If the snapshot becomes too large later, it can be split into dedicated endpoints, but milestone 1 should favor a single snapshot contract for simplicity.

### Suggested snapshot contract

The snapshot payload should be shaped around widgets, not raw tables.

Suggested top-level sections:

- `generatedAt`
- `health`
- `kpis`
- `trends`
- `attention`
- `workers`
- `recentEvents`

Suggested `kpis` fields:

- `requestsToday`
- `pendingRequests`
- `pendingDeliveries`
- `runningDeliveries`
- `failedDeliveriesLastHour`
- `pollingJobs`
- `ingestBacklog`
- `healthyWorkers`
- `unhealthyWorkers`

Suggested `attention` sections:

- `failedDeliveries`
- `staleRunningDeliveries`
- `stuckJobs`
- `recentIngestFailures`
- `unhealthyWorkers`

All attention lists should be intentionally short:

- default 5 to 10 rows
- enough to guide the operator into the detailed pages

### Suggested websocket event envelope

Use a compact typed envelope that can be consumed by both web and desktop:

```json
{
  "type": "delivery.failed",
  "timestamp": "2026-03-17T10:15:00Z",
  "severity": "error",
  "entityType": "delivery",
  "entityId": 42,
  "entityUid": "del_123",
  "summary": "Delivery to DHIS2 Uganda failed",
  "correlationId": "corr_456",
  "requestId": 12,
  "deliveryId": 42,
  "jobId": 8,
  "serverId": 3,
  "patch": {
    "kpi": "failedDeliveriesLastHour",
    "op": "increment",
    "value": 1
  },
  "payload": {
    "status": "failed",
    "serverName": "DHIS2 Uganda",
    "errorCode": "REMOTE_TIMEOUT"
  }
}
```

Recommended event types:

- `request.accepted`
- `request.completed`
- `request.failed`
- `delivery.started`
- `delivery.succeeded`
- `delivery.failed`
- `delivery.retry_scheduled`
- `job.pending`
- `job.polling`
- `job.succeeded`
- `job.failed`
- `worker.started`
- `worker.heartbeat`
- `worker.failed`
- `worker.stopped`
- `ingest.discovered`
- `ingest.processed`
- `ingest.failed`
- `ingest.retry_scheduled`
- `dashboard.invalidate`

The `dashboard.invalidate` event is useful when an exact patch is awkward and the client should refetch one widget or the whole snapshot.

### Event source strategy

Do not build a second event model from scratch.

Instead, publish dashboard events from state transitions that already exist in:

- request service
- delivery service
- async service
- worker service
- ingest service
- observability/event recording flow where appropriate

The goal is:

- existing services remain the source of state changes
- the dashboard stream observes those state changes

### Websocket auth and RBAC

Authentication and authorization must match the existing platform model.

Recommended rule for milestone 1:

- require authenticated user
- require `observability.read` to consume the websocket stream
- require at least read access to the underlying Sukumad surfaces for the snapshot

Practical options:

- strict option: require `observability.read` for both snapshot and stream
- broader option: allow snapshot with combined `requests.read` or `deliveries.read` or `jobs.read`, but reserve the stream for `observability.read`

The strict option is simpler and should be preferred unless product requirements clearly need wider access.

### Graceful shutdown

Any websocket hub or broadcaster must:

- stop on context cancellation
- avoid leaking goroutines
- detach dead subscribers

This should follow the existing project contract:

- `signal.NotifyContext`
- clean server shutdown
- background goroutines exit on cancellation

## Frontend plan

### Navigation

Reuse the existing authenticated shell and existing `/dashboard` route in both clients.

Do not add a second dashboard route hierarchy.

If Sukumad needs clearer navigation later, that should still be implemented within the existing drawer model.

### Shared widget model

Both web and desktop should implement the same dashboard sections:

- KPI cards
- live event feed
- worker health panel
- attention lists
- trend charts

Differences should be cosmetic only.

### Client loading model

Each client should:

1. load the dashboard snapshot over HTTP
2. render all widgets from the snapshot
3. open the websocket stream after initial load
4. merge live events into:
   - event feed
   - KPI patches where safe
   - targeted widget invalidation where needed
5. reconnect automatically if the websocket drops
6. refetch the full snapshot after reconnect

### Widget-by-widget implementation order

Implement in this order to keep risk controlled.

#### Phase 1 widgets

- KPI cards
- live event feed
- worker health summary
- failed deliveries attention panel

These provide immediate operational value with the smallest surface area.

#### Phase 2 widgets

- stuck jobs panel
- ingest failures panel
- stale running deliveries panel
- requests by hour chart
- deliveries by status chart

#### Phase 3 widgets

- failures by server chart
- richer worker detail
- live notifications/toasts from important events
- per-widget refetch optimization

### Reuse of existing pages

Attention widgets should link into existing filtered pages rather than duplicating full detail views.

Examples:

- failed deliveries panel -> `/deliveries?status=failed`
- stuck jobs panel -> `/jobs?status=polling`
- failures by server -> `/deliveries?server=<id>&status=failed`
- traceable event -> `/observability` with correlation filter

If current routes do not yet support some filters through URL state, add that as part of the milestone rather than duplicating detail grids on the dashboard.

## Phased implementation sequence

### Step 1. Backend snapshot aggregation

Implement the dashboard repository/service/handler for `GET /dashboard/operations`.

This step should aggregate from existing domain tables and services:

- requests
- deliveries
- async tasks
- worker runs
- ingest files
- recent observability events

Deliverables:

- dashboard types
- repository aggregation queries
- service mapping
- handler response
- backend tests

### Step 2. Web dashboard replacement

Replace the current placeholder web dashboard with the snapshot-driven operations view.

Deliverables:

- new widget components or local page sections
- dashboard data loader
- permission-gated rendering
- route tests

### Step 3. Desktop dashboard replacement

Replace the current placeholder desktop dashboard with the same operational view.

Deliverables:

- matching page structure
- desktop API integration
- route tests

### Step 4. Backend websocket stream

Add the authenticated websocket endpoint and server-side fanout.

Deliverables:

- websocket handler
- hub/broadcast implementation
- auth and permission enforcement
- publish hooks from existing Sukumad state transitions
- graceful shutdown handling
- backend tests for auth, subscription lifecycle, and delivery

### Step 5. Web live updates

Connect the web dashboard to the websocket stream.

Deliverables:

- stream client
- reconnect logic
- snapshot refetch on reconnect
- live event feed updates
- safe KPI patching or invalidation handling
- tests for live update behavior

### Step 6. Desktop live updates

Connect the desktop dashboard to the websocket stream.

Deliverables:

- stream client
- reconnect logic
- snapshot refetch on reconnect
- live event feed updates
- tests for live update behavior

### Step 7. Drill-down and URL filters

Make sure dashboard actions link into existing pages with useful filters already applied.

Deliverables:

- query-string filter support where missing
- tests confirming filtered routes render as expected

### Step 8. Final hardening

Before milestone completion:

- verify no secrets are emitted in dashboard payloads or events
- verify permission gating in backend and both clients
- verify websocket goroutines exit cleanly
- verify reconnect behavior and full resync
- update docs and status after all tests pass

## Testing plan

### Backend

Minimum backend test coverage should include:

- repository aggregation tests where practical
- service mapping tests
- snapshot handler tests
- websocket auth tests
- websocket permission tests
- websocket subscriber lifecycle tests
- event fanout tests
- graceful shutdown tests for the stream hub if introduced

### Web

Minimum web test coverage should include:

- dashboard route render test
- KPI cards render from snapshot
- attention panel render from snapshot
- websocket-driven live event feed update
- websocket-driven KPI change or invalidation handling
- permission-gated rendering or route handling

### Desktop

Minimum desktop test coverage should include:

- dashboard route render test
- KPI cards render from snapshot
- attention panel render from snapshot
- websocket-driven live event feed update
- websocket-driven KPI change or invalidation handling
- permission-gated rendering or route handling

### Verification gate

This milestone is not complete until all of these pass:

- `cd backend && GOCACHE=/tmp/go-build go test ./...`
- web tests
- web build
- desktop frontend tests
- desktop frontend build

`docs/status.md` must be updated only after the full milestone passes.

## Risks and controls

### Risk: too much websocket complexity too early

Control:

- start with one stream endpoint
- stream event envelopes, not full grids
- prefer snapshot refetch for complex recalculation

### Risk: duplicated event logic

Control:

- emit stream events from existing service transitions
- avoid a separate parallel lifecycle model

### Risk: permission drift between stream and UI

Control:

- enforce permissions in backend endpoint
- reuse the same route-access/navigation checks in web and desktop

### Risk: stale dashboard state after reconnect

Control:

- refetch the full snapshot after reconnect
- treat websocket patches as temporary optimization, not exclusive truth

### Risk: noisy event feed

Control:

- keep only operator-relevant event types in milestone 1
- cap the feed length client-side
- use severity and concise summaries

## Recommended milestone boundary

This work should be treated as one vertical milestone:

- backend snapshot + stream
- web dashboard
- desktop dashboard
- tests
- documentation

Do not treat backend-only streaming as complete on its own.

## Suggested milestone title

`Milestone — Sukumad Operations Dashboard + Live Event Stream`

## Suggested commit message after successful completion

`feat(sukumad): add operations dashboard with live websocket updates`
