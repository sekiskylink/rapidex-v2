# Sukumad Worker Execution Model

## Purpose

This note is the implementation contract for Sukumad background execution.

It now describes the real worker model in the codebase:

- the API process accepts and persists requests only
- a separate worker process runs send, retry, poll, and retention loops
- send and retry both reuse the same shared delivery submission path
- delivery pickup is concurrency-safe through durable claim semantics
- async polling is also concurrency-safe through durable task claims
- worker startup performs durable recovery before normal loop execution

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

Before the normal loops start, the worker process now performs two recovery passes:

- reconcile terminal async tasks that were persisted but not fully rolled up into delivery/request state
- requeue stale `running` deliveries that have no async task and were likely abandoned after a worker crash

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
- `RequeueStaleRunningDeliveries(...)`

The async repository now exposes:

- `ClaimNextDueTask(...)`
- `ListTerminalTasksForRecovery(...)`

### SQL claim behavior

For PostgreSQL-backed execution, claims use an atomic SQL pattern:

- select one eligible candidate with `FOR UPDATE SKIP LOCKED`
- update that row to `running` in the same statement
- clear stale response/hold/retry fields
- set `started_at`
- return the claimed delivery row

This prevents two worker loops from submitting the same delivery attempt concurrently.

For async tasks, PostgreSQL claim uses the same shape:

- select one due task with `FOR UPDATE SKIP LOCKED`
- require either no current poll claim or a stale poll claim older than the configured claim timeout
- set `poll_claimed_at` and `poll_claimed_by_worker_run_id`
- return the claimed async task row to the poll worker

### Memory/test claim behavior

The in-memory repository applies the same semantics under a mutex so service tests can assert the same concurrency assumptions.

## 7. Recovery semantics

The durable model now defines these restart behaviors:

- `pending` deliveries remain in durable `pending` and are picked by the send worker after restart
- `retrying` deliveries remain in durable `retrying`; once `retry_at <= now`, the retry worker picks them after restart
- due async tasks remain durable in `async_tasks`; the poll worker claims them after restart
- stale poll claims are recovered implicitly once `poll_claimed_at` is older than `sukumad.workers.poll.claim_timeout_seconds`
- stale `running` deliveries without an async task are requeued at worker startup:
  - initial attempts return to `pending`
  - retry attempts return to `retrying` with `retry_at = now`
- terminal async tasks whose poll result was persisted but whose delivery/request reconciliation was interrupted are reconciled at worker startup before the normal loops begin

`running` deliveries that already have a live non-terminal async task are not requeued; they remain owned by the poll-worker path.

## 8. Poll worker

The poll worker now runs as a real started worker in the separate worker process and works like this:

1. claim one due async task durably
2. emit poll-start worker-linked events
3. call the downstream poller through the shared rate-limited DHIS2 client
4. persist poll history in `async_task_polls`
5. update async task state and clear the claim
6. reconcile terminal success/failure back into delivery and request status

The poll worker also contributes per-run activity counts through `worker_runs.meta.counts`, including:

- `polls_picked`
- `polls_completed`
- `polls_failed`

## 9. Observability

Worker execution now emits worker-specific request events in addition to the existing delivery/request/async events, including:

- `delivery.worker.picked`
- `delivery.worker.claimed`
- `delivery.submission.started`
- `delivery.submission.deferred`
- `delivery.submission.completed`
- `delivery.retry.executed`
- `delivery.retry.deferred`
- `delivery.recovered.stale_running`
- `async.recovery.reconciled`

Existing worker lifecycle events remain in place:

- `worker.started`
- `worker.heartbeat`
- `worker.stopped`
- `worker.error`

Worker runs now also persist cumulative activity counters and last-activity metadata in `worker_runs.meta`, so operators can correlate:

- worker run
- request
- delivery attempt
- async task
- poll history

No secrets or tokens are written into event payloads.

## 10. Configuration and startup

Worker loop timing lives under `sukumad.workers` in backend config:

```yaml
sukumad:
  workers:
    heartbeat_seconds: 10
    recovery:
      stale_delivery_after_seconds: 300
    send:
      interval_seconds: 5
      batch_size: 10
    retry:
      interval_seconds: 5
      batch_size: 10
    poll:
      interval_seconds: 5
      batch_size: 10
      claim_timeout_seconds: 60
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

## 11. Status flow summary

The intended worker-driven lifecycle is now:

- first submission success: `pending -> running -> succeeded`, target `pending -> processing -> succeeded`, request `pending -> processing/completed -> completed`
- first submission sync failure: `pending -> running -> failed`, target becomes `failed`, request becomes `failed`
- async accepted then later success: delivery stays `running`, async task becomes durable, request/target become `processing`, poll worker later reconciles delivery to `succeeded` and request to `completed`
- async accepted then later failure: same durable async handoff, but poll reconciliation drives delivery to `failed` and request to `failed`
- retry scheduled then executed: failed attempt remains history, new attempt is created in `retrying`, retry worker claims it when due, and the same submission path decides final success/failure
- blocked/deferred work becoming executable later: dependency release returns blocked targets to `pending`; submission-window deferrals return the same delivery to `pending` with `next_eligible_at` until the send worker can claim it again

## 12. Testing expectations

Backend coverage for this worker model must include:

- pending delivery pickup by the send worker
- due retry pickup by the retry worker
- concurrency-safe delivery and async-task claim behavior
- reuse of the same submission path for send and retry
- deferred window handling without corrupting delivery/request state
- async poll reconciliation
- stale-running recovery
- worker restart against durable pending/async work
- worker process bootstrap/build validity

Current tests cover these areas in:

- `backend/internal/sukumad/delivery/repository_claim_test.go`
- `backend/internal/sukumad/delivery/service_test.go`
- `backend/internal/sukumad/async/service_test.go`
- `backend/internal/sukumad/worker/executor_test.go`
- `backend/internal/sukumad/worker/lifecycle_test.go`
- `backend/internal/sukumad/worker/service_test.go`

## 13. Summary

Sukumad now runs with the intended production split:

- API accepts and persists
- worker process executes background work
- send worker performs first submissions
- retry worker performs due retries
- poll worker reconciles async tasks
- poll worker pickup is claim-based and restart-safe
- send and retry share one delivery submission path
- delivery and async claim/pickup are safe under concurrent workers
- durable recovery handles stale running deliveries and missed async terminal reconciliation
