# Scheduler Maintenance Jobs

## Purpose

Sukumad scheduler maintenance jobs provide built-in operational cleanup and recovery tasks that can run on a schedule or on demand through the Scheduler UI and API.

These jobs are intentionally bounded:

- they use typed config payloads
- they return structured `resultSummary` metrics
- they can run in `dryRun` mode before making data changes

## Supported Job Types

### `archive_old_requests`

Marks old terminal requests as archived in `exchange_requests.extras.maintenance`.

Behavior:

- scans `completed` and `failed` requests older than `maxAgeDays`
- skips requests that are already marked as archived
- records scheduler archive metadata without deleting the request row

Supported config:

- `dryRun`
- `batchSize`
- `maxAgeDays`

### `purge_old_logs`

Deletes old operational log rows in batches.

Current purge targets:

- `audit_logs`
- `request_events`
- `async_task_polls`
- terminal `worker_runs`

Supported config:

- `dryRun`
- `batchSize`
- `maxAgeDays`

### `mark_stuck_requests`

Finds stale pending or processing requests and blocks them for operator attention.

Behavior:

- scans requests with active `pending` or `processing` targets whose request row has not changed within the configured stale cutoff
- marks matching targets as `blocked`
- marks the parent request as `blocked` with reason `stuck_request_timeout`

Supported config:

- `dryRun`
- `batchSize`
- `staleCutoffMinutes`
- `staleCutoffHours`

### `cleanup_orphaned_records`

Removes orphaned ingest-file rows after their linked requests have disappeared or never existed.

Current cleanup target:

- `ingest_files` rows with `request_id IS NULL` in terminal states (`processed`, `failed`) older than `maxAgeDays`

Supported config:

- `dryRun`
- `batchSize`
- `maxAgeDays`

## Result Summary Contract

Maintenance job runs return a structured `resultSummary` with these common fields:

- `jobType`
- `runUid`
- `dryRun`
- `batchSize`
- `scanned_count`
- `affected_count`
- `archived_count`
- `deleted_count`
- `skipped_count`

Job-specific fields may also appear, for example:

- `cutoff`
- `maxAgeDays`
- `staleCutoffMinutes`
- `deletedByTable`

## Seeded Default Jobs

When `seed.ensure_scheduler_jobs=true`, the API process ensures these jobs exist if they are missing:

- `archive-old-requests`
  - schedule: daily cron `0 1 * * *`
- `purge-old-logs`
  - schedule: weekly cron `0 2 * * 0`
- `mark-stuck-requests`
  - schedule: interval `10m`

Seed behavior is additive only:

- existing jobs are left unchanged
- seeded jobs remain editable and disable-able through the existing Scheduler UI and API

## UI Expectations

Web and desktop Scheduler forms expose typed maintenance config editors instead of requiring raw JSON for maintenance jobs.

Run details surfaces show:

- status badge
- dry-run indicator
- structured summary counts
- raw summary JSON for full inspection
