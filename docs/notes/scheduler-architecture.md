# Scheduler Architecture

## Purpose

The scheduler v1 slice introduces a first-class Sukumad scheduler module for two supported job categories:

- scheduled integration jobs
- maintenance jobs

This milestone is intentionally narrow. It establishes the data model, API surface, and client management shells without adding a general-purpose orchestration engine.

## Scope

Scheduler v1 includes:

- persisted scheduled job definitions
- persisted scheduled job run history
- schedule validation for `cron` and `interval`
- next-run calculation using the declared timezone
- manual `run now` queueing through the API
- web and desktop management pages for list, edit, and run history

Scheduler v1 does not include:

- retries
- async polling
- delivery-window scheduling
- arbitrary scripts
- DAG dependencies

Scheduler runtime v1.1 adds:

- a dispatcher loop in the worker process
- durable pending scheduler-run queueing through `scheduled_job_runs`
- worker-side execution of queued scheduler runs
- latest-run status surfaced back to list/detail APIs

## Module Placement

All backend scheduler logic lives under `backend/internal/sukumad/scheduler`.

The module follows the standard BasePro layering contract:

- `handler.go`
- `service.go`
- `repository.go`
- `types.go`

## Data Model

### `scheduled_jobs`

Stores durable scheduler definitions:

- identity and metadata
- job classification
- schedule definition
- execution controls
- runtime JSON configuration
- run summary timestamps
- audit timestamps

### `scheduled_job_runs`

Stores scheduler run history:

- identity and linkage to the scheduled job
- trigger metadata
- execution timestamps
- lifecycle status
- optional worker linkage
- error and summary payloads
- audit timestamps

## Scheduling Policy

The schedule engine validates schedule expressions on create/update and calculates `next_run_at`.

Supported schedule types:

- `interval`
- `cron`

Missed-run policy for v1:

- compute the next future due time only
- do not backfill or replay every missed execution

This keeps v1 deterministic and avoids implicit catch-up bursts before worker orchestration policies are designed.

## Runtime Execution

The scheduler runtime lives in the worker process and has two cooperating loops:

- a dispatcher loop that wakes on a configured interval, selects due enabled jobs, and enqueues pending `scheduled_job_runs`
- a scheduler-run worker loop that claims pending runs, marks them running, executes the registered job handler, and finalizes the run status

Duplicate dispatch protection relies on database row locking with `FOR UPDATE SKIP LOCKED` around due-job dispatch. This allows multiple worker instances to share the same database without queueing the same due occurrence twice.

`allow_concurrent_runs` is enforced in two places:

- dispatch skips creating a new pending run when the job already has a pending or running run
- run claiming prevents a second pending run for the same job from being promoted to `running` while another run is active

The durable queue remains `scheduled_job_runs`; no second scheduler-specific task table is introduced.

## Handler Registry

Scheduler job execution uses a handler registry keyed by `job_type`.

Registered job types:

- `url_call`
- `request_exchange`
- `metadata_sync`
- `export_pending_requests`
- `reconciliation_pull`
- `scheduled_backfill`
- `archive_old_requests`
- `purge_old_logs`
- `mark_stuck_requests`
- `cleanup_orphaned_records`

Handlers decode typed config payloads from the stored JSON config and return structured result summaries. Placeholder handlers remain explicit by returning either `succeeded` or `skipped` with a structured summary instead of silently no-oping.

### Integration Handlers

`url_call` executes an outbound call through an existing Integration Server record. It reuses the configured server base URL, HTTP method, headers, URL params, async flag, rate limiting, and response body persistence policy. The scheduler run stores a structured summary with destination server identity, HTTP status, response filtering state, and response summary metadata.

`request_exchange` creates a normal Sukumad exchange request through the existing request service. It accepts server UIDs, source system, payload, submission binding, response policy, URL suffix, and metadata. If no correlation ID is provided, the handler uses `scheduler:<jobCode>:<runUid>`; if an idempotency key prefix is provided, the handler uses `<prefix>:<runUid>`. Existing delivery workers remain responsible for sending the resulting request deliveries.

## API Surface

The versioned scheduler API lives under `/api/v1/scheduler`.

Endpoints:

- `GET /scheduler/jobs`
- `POST /scheduler/jobs`
- `GET /scheduler/jobs/:id`
- `PUT /scheduler/jobs/:id`
- `POST /scheduler/jobs/:id/enable`
- `POST /scheduler/jobs/:id/disable`
- `POST /scheduler/jobs/:id/run-now`
- `GET /scheduler/jobs/:id/runs`
- `GET /scheduler/runs/:id`

RBAC is enforced with dedicated scheduler permissions:

- `scheduler.read`
- `scheduler.write`

## Client Surfaces

Both clients expose:

- Scheduler list page
- Create/Edit Scheduled Job form shell
- Job Runs history shell

These pages use the existing authenticated app shell and Sukumad navigation group. They remain API-only; desktop does not access the database directly.

## Follow-Up Directions

Expected later milestones can extend the scheduler with:

- worker pickup/execution loops
- richer run lifecycle transitions
- retry and backoff policies
- delivery-window aware scheduling
- dependency orchestration
