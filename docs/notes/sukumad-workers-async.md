# Sukumad Workers, Async Tasks, and Rate Limiting

## Purpose

This note describes the first production-shaped async processing layer added in Milestone 5.

## Backend shape

The milestone introduces four persistent Sukumad runtime tables under the existing migration path:

- `async_tasks`
- `async_task_polls`
- `rate_limit_policies`
- `worker_runs`

Async task records are linked to `delivery_attempts`, and poll history is linked to async tasks.
Worker state is tracked independently in `worker_runs`.
Rate-limit policies are modeled separately so workers can resolve policy state without coupling policy data to server records.

## Module responsibilities

### `backend/internal/sukumad/async`

- owns async task list/detail/query operations
- creates async tasks linked to delivery attempts
- durably claims due async tasks for poll workers
- records poll history
- updates remote state, terminal state, next poll, completion, and response snapshots
- exposes a generic `PollDueTasks` workflow through a `RemotePoller` abstraction
- exposes recovery reconciliation for terminal async tasks that need delivery/request roll-up replay

### `backend/internal/sukumad/worker`

- owns worker run persistence and lifecycle transitions
- starts worker runs in `starting` then `running`
- writes periodic heartbeats to `worker_runs`
- persists per-run activity counts in `worker_runs.meta`
- stops workers cleanly on context cancellation
- provides production-shaped definitions for send, poll, and retry workers
- exposes a bootstrap seam without forcing startup auto-execution

### `backend/internal/sukumad/ratelimit`

- owns rate-limit policy persistence and lookup
- resolves active policies by scope
- provides an in-process limiter with request pacing and concurrency gates
- keeps the contract simple so future worker scheduling can plug into the same policy abstraction

### `backend/internal/sukumad/observability`

- exposes worker status and rate-limit visibility over the API
- reuses worker and ratelimit services instead of duplicating state management

## API surface

### Jobs

- `GET /api/v1/jobs`
- `GET /api/v1/jobs/:id`
- `GET /api/v1/jobs/:id/polls`

These endpoints are backed by the async module and require `jobs.read`.

### Observability

- `GET /api/v1/observability/workers`
- `GET /api/v1/observability/workers/:id`
- `GET /api/v1/observability/rate-limits`

These endpoints are read-only for this milestone and require `observability.read`.

## Worker startup strategy

Worker execution now starts from the dedicated `backend/cmd/worker` entrypoint.

The worker process is separate from the HTTP API process and starts:

- send worker
- retry worker
- poll worker
- retention worker

The async polling path still uses the same `async.Service.PollDueTasks(...)` and remote poller abstraction, but it now runs under the real worker manager and `worker_runs` lifecycle tracking rather than an unused bootstrap seam.

Poll pickup is now durable:

- due tasks are claimed through `async_tasks.poll_claimed_at` and `poll_claimed_by_worker_run_id`
- stale claims are recoverable after the configured claim timeout
- each poll still writes `async_task_polls` history and then clears the claim as part of the async-task status update

Worker startup also performs recovery work before the normal loops:

- terminal async tasks still attached to `running` deliveries are reconciled
- stale `running` deliveries with no async task are requeued for send/retry pickup

## Client shape

Web and desktop both reuse the existing Sukumad routes and navigation:

- `/jobs` shows async task visibility and job detail with poll history
- `/observability` shows worker status and rate-limit visibility

No parallel shell, router, or navigation model was introduced.
