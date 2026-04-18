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
