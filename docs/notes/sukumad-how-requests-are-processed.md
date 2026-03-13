# Sukumad Request Processing

## Purpose

This note explains the current Sukumad request lifecycle in the backend:

- how a request is created
- how it is expanded into destination targets
- how delivery attempts are created and submitted
- how async polling works
- how retries are scheduled today
- what is implemented now versus what is scaffolded for later

The description below is based on the current code in:

- `backend/internal/sukumad/request`
- `backend/internal/sukumad/delivery`
- `backend/internal/sukumad/async`
- `backend/internal/sukumad/dhis2`
- `backend/internal/sukumad/worker`

## Main records involved

The request flow is spread across four persistent record types.

### 1. `exchange_requests`

This is the business-level request record.

It stores the original payload, destination metadata, correlation fields, roll-up status, and dependency information.

Request statuses are:

- `pending`
- `blocked`
- `processing`
- `completed`
- `failed`

### 2. `request_targets`

Each request can fan out to multiple destinations.

The primary destination comes from `destinationServerId`. Additional destinations can be supplied through `destinationServerIds`.

Each target tracks one destination server’s state inside the overall request:

- `pending`
- `blocked`
- `processing`
- `succeeded`
- `failed`

The request status is derived from these target states.

### 3. `delivery_attempts`

Each concrete submission attempt to a destination server is stored as a delivery attempt.

Important delivery states are:

- `pending`
- `running`
- `succeeded`
- `failed`
- `retrying`

Retries create a new delivery attempt. Failed attempts remain as history.

### 4. `async_tasks` and `async_task_polls`

If the destination system accepts work asynchronously, Sukumad creates an async task linked to the delivery attempt.

That task stores:

- remote job id
- poll URL
- current remote state
- next poll time
- terminal state when known

Each poll attempt is persisted in `async_task_polls`.

## End-to-end flow

## 1. Request creation starts at the API

New requests are created through:

- `POST /api/v1/requests`

The handler delegates to `request.Service.CreateRequest(...)`.

This is the orchestration entry point for the whole initial lifecycle.

## 2. Input is normalized and the request row is created

`CreateRequest(...)` does the following first:

- validates the payload is non-empty valid JSON
- normalizes the destination server list
- validates dependency request ids
- inserts the `exchange_requests` row with initial status `pending`
- writes audit and trace events such as `request.created`

At this point the business request exists durably even before any outbound call succeeds.

## 3. Destination targets are created

After the request row is created, the service creates one `request_targets` row per destination server.

The first target is marked `primary`. Any extra destinations are marked `cc`.

All targets start in `pending`.

This is the point where the request becomes a multi-destination request internally, even though users still think of it as a single request.

## 4. Dependencies are evaluated before sending

A request can depend on other requests.

The request service loads dependency statuses immediately after creation.

There are three possible outcomes:

- If any dependency has already failed, the new request is failed immediately.
- If any dependency is still incomplete, the request is created but outbound submission is blocked.
- If all dependencies are completed, submission continues immediately.

When blocked by dependency:

- the delivery row is still created
- the target is moved to `blocked`
- the request roll-up becomes `blocked`
- no outbound submission is attempted yet

## 5. A pending delivery attempt is created per target

For each target server, the request service calls `delivery.Service.CreatePendingDelivery(...)`.

That creates a `delivery_attempts` row with:

- `attempt_number = 1`
- `status = pending`

The delivery row exists before the destination system is contacted. That gives the system a durable attempt history and something to reconcile later if the request becomes async.

## 6. The first delivery remains pending for later execution

If the target is not dependency-blocked, the request service stops after creating the initial `pending` delivery attempt.

This means initial request intake is now accept-and-persist only. The first send is no longer attempted inline from the API path and is reserved for the worker-owned execution path.

## What `SubmitDHIS2Delivery(...)` does

The delivery service is the core state machine for outbound submission.

Its flow is:

1. Evaluate submission-window policy for the destination.
2. If allowed, move the delivery from `pending` or `retrying` to `running`.
3. Call the configured dispatcher.
4. Interpret the dispatcher result.
5. Update delivery, target, and request state.
6. Create an async task if the destination accepted async processing.

The dispatcher is currently the DHIS2 integration service:

- `backend/internal/sukumad/dhis2/service.go`

That module builds the outbound HTTP request from the stored server configuration and returns a normalized `DispatchResult`.

## Submission window behavior

Before any outbound call, delivery checks whether the destination is inside its allowed submission window.

If the window is closed:

- the delivery remains `pending`
- `submission_hold_reason` is set to `window_closed`
- `next_eligible_at` is populated
- the target becomes `blocked`
- the request roll-up becomes `blocked`

This is a defer, not a failure.

Important current limitation:

- there is no active send worker implementation that later resumes these `pending` deliveries automatically

The worker type exists, but `backend/cmd/api/main.go` currently wires `worker.NewSendDefinition(nil)`.

## Synchronous destination outcomes

If the destination returns a terminal answer immediately, the delivery finishes in the same request path.

### Success path

When the dispatcher returns `Terminal=true` and `Succeeded=true`:

- delivery becomes `succeeded`
- response metadata is stored
- target becomes `succeeded`
- request roll-up is recalculated

If all targets have succeeded, the request becomes `completed`.

### Failure path

When the dispatcher returns a terminal failure, or the outbound call errors:

- delivery becomes `failed`
- error details are recorded
- target becomes `failed`
- request roll-up is recalculated

If any target fails, the request becomes `failed`.

## Asynchronous destination outcomes

If the destination accepts the submission asynchronously:

- the delivery stays `running`
- initial response metadata is stored on the delivery
- an `async_tasks` row is created
- the request is moved to `processing`
- the target remains `processing`

The async task stores the remote identifiers Sukumad needs to keep following the job after the original API request is gone.

For async submissions, the initial polling schedule is:

- first poll at about 15 seconds after task creation

That timestamp is stored in `next_poll_at`.

## Polling flow

Polling is owned by:

- `async.Service.PollDueTasks(...)`

The poller currently used is the DHIS2 integration poller.

### How due tasks are selected

The async repository returns tasks whose `next_poll_at` is due.

For each due task, the service:

1. writes an `async.poll_started` trace event
2. calls the remote poller
3. records a poll history row
4. updates async task state
5. reconciles terminal states back into delivery and request records

### If a poll call fails

When the poll HTTP call itself fails:

- a poll history row is still recorded
- the task is updated back to a polling state
- `next_poll_at` is pushed forward
- an `async.poll_failed` event is written

The current retry delay for failed poll attempts is:

- 30 seconds

### If a poll returns non-terminal progress

When the remote system responds successfully but is still in progress:

- the poll history row is recorded
- the task remains non-terminal
- `next_poll_at` is updated from the poll result
- the request stays `processing`

### If a poll returns terminal success

When the poll result resolves successfully:

- async task terminal state becomes `succeeded`
- `delivery.CompleteFromAsyncSuccess(...)` marks the delivery `succeeded`
- the target becomes `succeeded`
- the request roll-up is recalculated

If all targets are now successful, the request becomes `completed`.

### If a poll returns terminal failure

When the poll result resolves as failed:

- async task terminal state becomes `failed`
- `delivery.CompleteFromAsyncFailure(...)` marks the delivery `failed`
- the target becomes `failed`
- the request roll-up is recalculated

If any target is failed, the request becomes `failed`.

## How request roll-up is derived

The request record is not treated as an independent state machine once targets exist. Instead, the request service derives the request roll-up from target states.

The current roll-up rules are:

- if any target is `failed`, the request is `failed`
- else if any target is `processing`, the request is `processing`
- else if all targets are `succeeded`, the request is `completed`
- else if any target is `blocked`, or any target carries a block reason/deferred time, the request is `blocked`
- otherwise the request is `pending`

This is why a multi-target request can remain `processing` even if some targets have already completed, and why one failed target fails the whole request.

## Dependency release behavior

A request that was blocked by dependencies is not stuck forever.

When a dependency request reaches a terminal state, the request service reevaluates dependents.

### If a dependency failed

All still-open blocked targets on the dependent request are marked failed with reason `dependency_failed`.

That causes the dependent request to roll up to `failed`.

### If all dependencies completed

Blocked targets with reason `dependency_blocked` are released:

- target status is moved back to `pending`
- `last_released_at` is set
- the request roll-up returns to `pending`

So dependency release no longer resumes work inline from the request service. It returns the request to worker-eligible durable state instead.

## Retry behavior

Manual retries are exposed through:

- `POST /api/v1/deliveries/:id/retry`

`delivery.Service.RetryDelivery(...)` only accepts deliveries in `failed`.

When a retry is requested:

- the failed delivery is loaded
- max retries are checked using destination policy
- a new delivery attempt is created
- the new attempt gets `status = retrying`
- `retry_at` is set to about 5 minutes in the future

Important current behavior:

- creating the retry attempt does not itself submit it
- the original failed attempt remains unchanged
- the retry is durable and visible through the API/UI

Important current limitation:

- retry execution is not automatically wired yet
- `backend/cmd/api/main.go` currently wires `worker.NewRetryDefinition(nil)`

So retries are scheduled in data, but no active retry worker is consuming them yet.

## Worker involvement today

The worker framework exists and worker runs are persisted in `worker_runs`.

Current runtime wiring is:

- send worker definition exists but has no run function
- poll worker is active and calls `async.PollDueTasks(...)`
- retry worker definition exists but has no run function
- retention worker is active separately for cleanup

In practical terms:

- request creation performs the first submission inline
- dependency release performs resumed submission inline
- async polling is background-worker driven
- deferred sends and scheduled retries are not yet background-worker driven

## Observability and audit trail

The flow writes both audit events and Sukumad trace events.

Examples include:

- `request.created`
- `delivery.created`
- `delivery.started`
- `delivery.succeeded`
- `delivery.failed`
- `async_task.created`
- `async.poll_started`
- `async.poll_failed`
- `async.poll_succeeded`
- `request.rollup.updated`

That means operators can inspect the lifecycle from:

- requests
- deliveries
- async jobs
- request, delivery, and job event streams
- worker run status

## Practical summary

Today, a request moves through the system like this:

1. A request is created and persisted.
2. One or more destination targets are created.
3. A pending delivery attempt is created for each target.
4. If dependencies allow it, submission is attempted immediately.
5. The destination either finishes synchronously or returns an async job.
6. Async jobs are polled by the poll worker until terminal.
7. Terminal delivery outcomes update target state.
8. Target states roll up into the final request state.

The important implementation caveat is that polling is automated, but deferred sends and scheduled retries are still only persisted scaffolding until send/retry worker execution is wired in.
