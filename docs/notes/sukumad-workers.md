# Sukumad Worker Execution Model

## Purpose

This note is the implementation contract for Sukumad background execution.

It now describes the real worker model in the codebase:

- the API process accepts and persists requests only
- a separate worker process runs send, retry, poll, and retention loops
- send and retry both reuse the same shared delivery submission path
- delivery pickup is concurrency-safe through durable claim semantics

Relevant code:

- `backend/cmd/api/main.go`
- `backend/cmd/worker/main.go`
- `backend/internal/sukumad/delivery`
- `backend/internal/sukumad/request`
- `backend/internal/sukumad/async`
- `backend/internal/sukumad/worker`

## 1. Process split

### API process

`backend/cmd/api/main.go` starts the Gin HTTP server only.

Its responsibilities remain:

- validation
- authorization
- persistence
- request acceptance
- API visibility for deliveries, jobs, observability, and related domain state

The API process does not start send, retry, poll, or retention loops.

### Worker process

`backend/cmd/worker/main.go` is the operational worker entrypoint.

It starts a dedicated worker manager with these loops:

- send worker
- retry worker
- poll worker
- retention worker

This process uses `signal.NotifyContext`, the shared DB/config bootstrap, and the existing `worker_runs` lifecycle table for heartbeats and run visibility.

## 2. Request intake model

`request.Service.CreateRequest(...)` remains accept-and-persist only.

The API path:

1. validates request input
2. persists `exchange_requests`
3. persists `request_targets`
4. persists dependency links
5. creates initial `delivery_attempts` with status `pending`
6. returns the persisted request immediately

No outbound submission happens inline during API request creation.

## 3. Shared submission path

`delivery.Service.SubmitDHIS2Delivery(...)` remains the single outbound submission path for delivery execution.

That shared path is used by:

- send worker first submissions
- retry worker due retries
- any future controlled replay or resubmit flow

The shared path continues to own:

- submission-window evaluation
- transition to terminal success or failure
- async task creation and handoff
- request/target roll-up updates
- destination-scoped rate limiting through the shared DHIS2 client
- response content-type filtering
- trace/audit event emission

The submission path now accepts a delivery that was already claimed into `running` by a worker, so worker claim and outbound dispatch stay unified without duplicating delivery logic.

## 4. Send worker

The send worker runs in the separate worker process and:

1. claims one eligible `delivery_attempts` row in `pending`
2. loads the persisted request and destination server context
3. emits worker pickup/claim/submission events
4. calls `SubmitDHIS2Delivery(...)`
5. lets the shared delivery service finalize the result

Eligible pending deliveries are:

- `delivery_attempts.status = 'pending'`
- linked target is `pending`
- or linked target is `blocked` for `window_closed` and the delivery is now due by `next_eligible_at`

Dependency-blocked targets are not claimable by the send worker until the dependency release flow returns the target to `pending`.

Possible send outcomes:

- `succeeded`
- `failed`
- `running` with durable `async_tasks` for async downstream work
- back to `pending` plus blocked request/target state when the submission window still prevents dispatch

## 5. Retry worker

The retry worker runs in the separate worker process and:

1. claims one `delivery_attempts` row in `retrying`
2. requires `retry_at <= now`
3. loads the same persisted request and server context
4. emits retry pickup/claim/executed events
5. calls the exact same `SubmitDHIS2Delivery(...)` path used by the send worker

Retry execution does not implement a separate outbound code path.

That preserves identical behavior for:

- rate limiting
- submission windows
- response filtering
- async creation
- success/failure reconciliation

## 6. Claim and pickup model

Claiming is durable and exclusive.

The delivery repository now exposes:

- `ClaimNextPendingDelivery(...)`
- `ClaimNextRetryDelivery(...)`

### SQL claim behavior

For PostgreSQL-backed execution, claims use an atomic SQL pattern:

- select one eligible candidate with `FOR UPDATE SKIP LOCKED`
- update that row to `running` in the same statement
- clear stale response/hold/retry fields
- set `started_at`
- return the claimed delivery row

This prevents two worker loops from submitting the same delivery attempt concurrently.

### Memory/test claim behavior

The in-memory repository applies the same semantics under a mutex so service tests can assert the same concurrency assumptions.

### Current recovery note

The current durable claim model protects against duplicate live pickup. It does not yet add stale-running recovery after a crash between claim and completion. That remains a follow-up concern, but pending and retrying rows remain durable and resumable across worker restarts.

## 7. Observability

Worker execution now emits worker-specific request events in addition to the existing delivery/request/async events, including:

- `delivery.worker.picked`
- `delivery.worker.claimed`
- `delivery.submission.started`
- `delivery.submission.deferred`
- `delivery.submission.completed`
- `delivery.retry.executed`
- `delivery.retry.deferred`

Existing worker lifecycle events remain in place:

- `worker.started`
- `worker.heartbeat`
- `worker.stopped`
- `worker.error`

No secrets or tokens are written into event payloads.

## 8. Configuration and startup

Worker loop timing lives under `sukumad.workers` in backend config:

```yaml
sukumad:
  workers:
    heartbeat_seconds: 10
    send:
      interval_seconds: 5
      batch_size: 10
    retry:
      interval_seconds: 5
      batch_size: 10
    poll:
      interval_seconds: 5
      batch_size: 10
    retention:
      interval_seconds: 300
```

Start the worker process separately from the API process, for example:

```bash
cd backend
go run ./cmd/worker -config ./config/config.yaml
```

Start the API separately:

```bash
cd backend
go run ./cmd/api -config ./config/config.yaml
```

## 9. Testing expectations

Backend coverage for this worker model must include:

- pending delivery pickup by the send worker
- due retry pickup by the retry worker
- concurrency-safe claim behavior
- reuse of the same submission path for send and retry
- deferred window handling without corrupting delivery/request state
- worker process bootstrap/build validity

Current tests cover these areas in:

- `backend/internal/sukumad/delivery/repository_claim_test.go`
- `backend/internal/sukumad/delivery/service_test.go`
- `backend/internal/sukumad/worker/executor_test.go`

## 10. Summary

Sukumad now runs with the intended production split:

- API accepts and persists
- worker process executes background work
- send worker performs first submissions
- retry worker performs due retries
- poll worker reconciles async tasks
- send and retry share one delivery submission path
- delivery claim/pickup is safe under concurrent workers
