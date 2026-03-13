# Sukumad Worker Architecture Recommendation

## Purpose

This note documents:

- the current Sukumad request-processing model as it exists in the codebase today
- the recommended production worker model for long-running outbound processing
- a practical migration path from the current accept-and-persist API flow to fully worker-driven execution

The description below is based on the current code in:

- `backend/internal/sukumad/request`
- `backend/internal/sukumad/delivery`
- `backend/internal/sukumad/async`
- `backend/internal/sukumad/dhis2`
- `backend/internal/sukumad/worker`
- `backend/cmd/api/main.go`

## 1. Current model

### Request creation via API

New requests are created through `POST /api/v1/requests`.

`request.Service.CreateRequest(...)` now performs accept-and-persist orchestration only:

1. validates and normalizes payload, destination IDs, and dependency IDs
2. persists the `exchange_requests` row with initial request status `pending`
3. persists `request_targets` rows for the primary destination and any fan-out destinations
4. persists dependency links
5. evaluates dependency state
6. creates one `delivery_attempts` row per target with `attempt_number = 1` and delivery status `pending`
7. leaves each first attempt in durable `pending` for later worker pickup, or marks dependency-blocked targets and the request as `blocked`

This means request acceptance and first outbound submission are now decoupled in the API path.

### Persistence of request records

The persistence model is already durable-first:

- `exchange_requests` is written before outbound submission
- `request_targets` is written before outbound submission
- `delivery_attempts` is written before outbound submission
- `async_tasks` and `async_task_polls` are written durably once async work begins

That part of the shape is already compatible with a worker model.

### Current outbound execution behavior

Outbound submission still lives in `delivery.Service.SubmitDHIS2Delivery(...)`, but request creation no longer calls it directly.

`delivery.Service.SubmitDHIS2Delivery(...)` currently:

1. evaluates the submission-window policy
2. if the destination is closed, leaves the delivery in `pending`, stores `submission_hold_reason` and `next_eligible_at`, and blocks the target/request roll-up
3. if allowed, marks the delivery `running`
4. calls the shared dispatcher, currently the DHIS2 service
5. finalizes the attempt as sync success, sync failure, or async waiting

Important consequence: API latency no longer includes the first outbound dispatch attempt. Pending deliveries remain durable and eligible for later worker pickup.

### Delivery attempt creation

Every concrete submission attempt is represented by a `delivery_attempts` row.

Current delivery statuses are:

- `pending`
- `running`
- `succeeded`
- `failed`
- `retrying`

First attempts are created as `pending`. Retries create a new row rather than mutating history.

### Retry behavior

Retry creation already exists, but retry execution is not worker-driven yet.

`delivery.Service.RetryDelivery(...)` currently:

- only accepts a failed delivery
- enforces max retries
- creates a new delivery row with status `retrying`
- sets `retry_at` about 5 minutes in the future

What is missing today:

- there is no active retry worker that scans due `retrying` deliveries and resubmits them
- there is no delivery pickup repository path for due retries or pending deliveries

So retries are durably represented, but automated retry execution is still incomplete.

### Async polling behavior

Async polling is the most worker-shaped flow already present.

When a destination accepts async processing:

- the delivery remains `running`
- an `async_tasks` row is created
- the request roll-up becomes `processing`
- the first `next_poll_at` is scheduled about 15 seconds later

`async.Service.PollDueTasks(...)` then:

1. loads due tasks from `async.Repository.ListDueTasks(...)`
2. polls the remote system through the shared DHIS2 poller
3. persists poll history in `async_task_polls`
4. updates async task state
5. reconciles terminal success/failure back into delivery and request state

This is durable and production-shaped, but it is not yet actually running as an independently started worker process.

### Partial or incomplete worker wiring

The worker module currently provides:

- worker run persistence in `worker_runs`
- lifecycle tracking (`starting`, `running`, `stopped`, `failed`)
- a generic manager loop
- definitions for send, poll, retry, and retention workers

Current bootstrap reality in `backend/cmd/api/main.go`:

- `worker.NewPollDefinition(...)` is constructed with a real `PollDueTasks(...)` path
- `worker.NewSendDefinition(nil)` is wired with no execution logic
- `worker.NewRetryDefinition(nil)` is wired with no execution logic
- `worker.NewBootstrap(...)` is created, but `Manager.Start(...)` is not called

So today:

- no send worker is running
- no retry worker is running
- no poll worker process is running automatically
- request creation stops after durable persistence, so pending deliveries wait for worker/runtime execution wiring

## 2. Recommended production model

### Top-level split

Recommended long-term responsibility split:

- API process handles validation, authorization, idempotency checks, persistence, and request acceptance
- worker process handles outbound execution, retries, async polling, and state reconciliation

The API should stop after accept-and-persist. Workers should own all outbound work after that point.

### Separate worker process or binary

Workers should run as a separate process or binary, not inside the main HTTP server loop.

Recommended deployment shape:

- `api` binary/process for Gin HTTP traffic
- `worker` binary/process for background execution

It is acceptable for one worker process to host multiple worker loops, but it should still be operationally separate from the HTTP server.

### Send worker

The send worker should:

1. pick delivery attempts eligible for first submission
2. claim them durably
3. mark them `running`
4. execute outbound submission through the same shared submission path used today
5. finalize each attempt to:
   - `succeeded`
   - `failed`
   - async waiting state (`delivery running` plus durable `async_tasks`)
   - deferred `pending` if submission windows still block execution

Eligibility should include:

- delivery status `pending`
- dependency gate satisfied
- submission window eligible now, or reevaluated immediately before dispatch
- no conflicting active claim by another worker

### Poll worker

The poll worker should:

1. pick due async tasks
2. claim them safely
3. poll downstream systems
4. persist poll history
5. update async task state
6. reconcile terminal async outcome back into delivery and request state

This should keep using the existing async service and remote poller abstraction. The poll path already matches the desired long-term ownership model; it mainly needs real worker deployment and safe pickup semantics.

### Retry worker

The retry worker should:

1. pick deliveries in `retrying` state whose `retry_at` is due
2. claim them safely
3. resubmit them through the same shared submission path used by the send worker
4. reuse the same rate limiting, submission-window, and response-handling logic as first sends

Retry logic should not be a separate outbound implementation. It should reuse the exact same dispatch and reconciliation path as normal sends.

## 3. Why this is preferred

This model is preferred in production because it gives:

- predictable API latency: request acceptance is no longer tied to downstream network calls
- durable request creation: the request, targets, and deliveries exist before workers act
- background processing independent of HTTP timeouts and reverse-proxy timeouts
- independent scaling: worker capacity can scale separately from API capacity
- safer restarts and deploys: API restarts do not interrupt in-flight orchestration design
- clearer operations: sends, retries, and polls become explicit background responsibilities
- cleaner retry and async semantics: durable state, pickup, and reconciliation all live in one operational model

## 4. Status semantics

### Recommended request semantics

Recommended worker-driven request semantics:

- `pending` = accepted and persisted, awaiting worker pickup
- `processing` = at least one target is actively submitting or awaiting async completion
- `completed` = all required targets reached terminal success
- `failed` = terminal failure based on target roll-up

The current codebase also uses `blocked`, and that should remain meaningful:

- `blocked` = accepted and persisted, but currently ineligible to proceed because of dependency gating or a submission window

In practice, `blocked` is useful and already encoded in the current roll-up logic, so the recommended production model should keep it even if the simplified lifecycle is described as pending/processing/completed/failed.

When dependencies complete, blocked targets should return to `pending` so execution resumes through the worker path rather than an inline API shortcut.

### Recommended delivery semantics

Delivery-level expectations:

- `pending` = durable attempt exists but no worker has started outbound execution yet
- `running` = worker claimed the attempt and is actively submitting, or the attempt is awaiting async completion
- `succeeded` = terminal success for that attempt
- `failed` = terminal failure for that attempt
- `retrying` = a follow-up attempt has been created and is scheduled for later worker pickup

Additional expectations:

- `submission_hold_reason` and `next_eligible_at` explain why a delivery is not yet executable
- async work should continue to link back to the same delivery attempt rather than inventing a parallel state machine
- request roll-up should continue deriving from target state, not from ad hoc worker memory

## 5. Refactor path

Recommended staged migration with minimal disruption:

### Stage 1. Preserve the current persistence model

Keep the existing durable-first shape:

- request row first
- target rows first
- delivery attempt rows first
- async task rows durable when needed

This part is already in place and should not be redesigned.

### Stage 2. Add durable worker pickup for pending deliveries

Add repository support for worker pickup of eligible deliveries, for example:

- pending first attempts ready to send
- retrying attempts whose `retry_at` is due

Pickup should be safe for concurrent workers and should use claim semantics rather than optimistic duplicate scanning.

### Stage 3. Share one submission path

Keep `delivery.Service.SubmitDHIS2Delivery(...)` as the shared submission path.

Execution paths that do send work should all converge here:

- future send worker path
- future retry worker path
- any controlled replay/resubmit path

### Stage 4. Preserve accept-and-persist request intake

This stage is now implemented:

- `request.Service.CreateRequest(...)` no longer calls `SubmitDHIS2Delivery(...)` directly
- dependency release no longer submits inline
- released targets return to worker-eligible `pending` state

### Stage 5. Make request creation purely accept-and-persist

Final API behavior:

1. validate
2. authorize
3. persist request, targets, dependencies, and initial deliveries
4. return accepted response

After this stage, all outbound execution belongs to workers only.

## 6. Operational considerations

### Worker deployment model

Recommended default:

- one dedicated worker process running send, poll, retry, and optionally retention loops

Alternative later:

- separate binaries or separately scaled deployments for send-heavy versus poll-heavy workloads

The key requirement is process separation from the HTTP server.

### One process vs separate binaries

One worker process with multiple loops is simpler first.

Separate binaries become attractive when:

- send throughput and poll throughput need different scaling
- retry traffic must be isolated
- operators want fault isolation between worker roles

The architecture should support both without changing persistence or service contracts.

### Idempotency and safe pickup

Workers must be safe under concurrency and restarts.

Recommended expectations:

- request idempotency remains enforced at API acceptance time
- worker pickup claims should be durable and exclusive
- delivery submission should tolerate duplicate worker wakeups without double-submitting the same attempt
- claim records or atomic state transitions should prevent two workers from executing one delivery simultaneously

### Crash recovery

Crash recovery should be based on durable state, not in-memory queues.

Recommended rules:

- `pending` deliveries remain eligible for later pickup after worker restarts
- `retrying` deliveries remain eligible once `retry_at` is due
- async tasks remain pollable based on `next_poll_at`
- stale `running` deliveries may need a recovery or reconciliation rule if a worker crashes after claim but before terminal update

### Concurrency controls

Concurrency should be explicit and configurable per worker role.

Recommended controls:

- max concurrent send submissions
- max concurrent poll calls
- batch size for pickup scans
- per-destination fairness where practical

### Interaction with rate limiting

Workers must continue using the existing shared destination-scoped outbound limiter.

That means:

- send worker submissions pass through the same DHIS2 client limiter path
- retry submissions pass through the same limiter path
- poll traffic keeps using the same limiter path it already uses today

Rate limiting should stay at the outbound client boundary, not be reimplemented separately inside worker loops.

### Interaction with retries

Retry creation and retry execution are separate concerns:

- delivery service can continue creating retry attempts
- retry worker should own execution of due retries

Retry policy checks should remain shared so manual retry and worker retry do not drift.

### Interaction with submission windows

Submission windows should remain an execution-time gate.

Recommended behavior:

- a worker can pick a candidate delivery
- eligibility is reevaluated immediately before dispatch
- if still blocked, the delivery remains or returns to `pending` with `submission_hold_reason` and `next_eligible_at`
- workers later re-check when the delivery becomes due again

### Observability expectations

Production worker operation should expose:

- worker run lifecycle and heartbeats
- counts of picked, started, deferred, succeeded, failed, and retried deliveries
- async poll success/failure counts
- queue depth indicators for pending deliveries, due retries, and due polls
- correlation between request, delivery, async task, and worker run events

Existing request, delivery, async, and worker event streams provide a useful base and should be extended rather than replaced.

## 7. Testing expectations

Worker-driven Sukumad architecture should be tested at several levels.

### Backend service tests

Test:

- API request creation persists request, targets, dependencies, and deliveries without outbound submission
- send worker picks eligible pending deliveries and calls the shared submission path
- retry worker picks due retrying deliveries and reuses the same submission path
- poll worker picks due async tasks and reconciles terminal outcomes correctly

### Repository tests

Test:

- safe pickup queries for pending deliveries
- safe pickup queries for due retries
- due async polling selection
- idempotent claim/update behavior under concurrent access assumptions

### Handler and API tests

Test:

- request creation returns accepted persisted state
- delivery and request status surfaces reflect worker-driven transitions
- observability endpoints show worker runs and lifecycle data correctly

### Failure and recovery tests

Test:

- worker restart with pending work still in the database
- worker crash after claim but before completion
- downstream timeout or transport failure
- async poll failures that reschedule rather than terminate immediately
- blocked-by-window and blocked-by-dependency transitions back to executable state

### End-to-end expectations

Test end-to-end flows for:

- sync success
- sync failure
- async success
- async failure
- retry success after an initial failure
- fan-out requests with mixed target outcomes
- dependency-blocked requests released later by dependency completion

## Recommended conclusion

Sukumad already has the correct durable persistence shape for worker-driven processing, but it still executes the first send inline in the API path and does not start real send or retry workers.

The recommended production model is:

- API accepts and persists
- send worker executes first submissions
- poll worker reconciles async work
- retry worker executes due retries
- workers run outside the HTTP server as a separate operational process

That approach keeps the current data model, reuses the existing submission and polling services, and removes the main production weakness of the current inline-first design.
