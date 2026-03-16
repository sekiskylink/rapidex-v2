# Sukumad Directory Ingestion

## Purpose

This note describes the production directory-ingestion model for Sukumad request intake.

The feature is intentionally backend-only at the execution layer:

- desktop and web do not read watched directories
- desktop and web continue to create requests through the same backend APIs
- the backend worker process can additionally ingest request envelopes from a configured directory

The implementation reuses the existing request-creation path and worker runtime instead of introducing a parallel request model.

Relevant code:

- `backend/cmd/worker/main.go`
- `backend/internal/sukumad/ingest`
- `backend/internal/sukumad/request`
- `backend/internal/sukumad/worker`

## Architecture

Directory ingestion is implemented as a dedicated Sukumad worker role.

It combines two inputs:

- `fsnotify` for low-latency detection when new files arrive
- periodic directory reconciliation so correctness does not depend on watcher delivery

Detected files are tracked in a durable `ingest_files` ledger table before they are processed.

That ledger is the source of truth for:

- deduplication
- retry scheduling
- claim ownership
- crash recovery
- request linkage
- operator visibility through logs and worker run metadata

## File lifecycle

Configured directories:

- inbox
- processed
- failed

Expected flow:

1. A producer writes a request envelope into the inbox directory.
2. The ingest worker observes the file through `fsnotify` and/or periodic scan.
3. The worker upserts a ledger record in `ingest_files`.
4. After the file is stable for the configured debounce period, the worker claims the ledger row.
5. The worker reads and validates the file envelope.
6. The worker calls `request.Service.CreateRequest(...)`.
7. On success, the file is atomically moved to the processed directory and the ledger row is marked `processed`.
8. On validation or business-rule failure, the file is moved to the failed directory and the ledger row is marked `failed`.
9. On transient failures, the ledger row is scheduled for retry and the file remains in the inbox.

## Durability rules

`fsnotify` is not treated as a queue.

The worker must always perform periodic reconciliation of the inbox directory because:

- watcher events can be coalesced
- events can be dropped
- the process may restart while files are already present

The database ledger must support:

- claim timeout recovery
- retry backoff
- active-row uniqueness per source path
- historical records for processed and failed files

## Envelope contract

Each file contains one request envelope.

The envelope maps directly onto the existing request-creation API contract:

- `sourceSystem`
- `destinationServerId`
- `destinationServerIds`
- `dependencyRequestIds`
- `batchId`
- `correlationId`
- `idempotencyKey`
- `payload`
- `payloadFormat`
- `submissionBinding`
- `urlSuffix`
- `extras`

The worker enriches `extras` with ingest metadata such as:

- ingest source = `directory`
- original file name
- checksum
- observed timestamp

## Idempotency

Production ingestion must not rely on filenames alone.

The preferred contract is:

- producers send a stable `idempotencyKey`

If a file omits an idempotency key, the worker may derive one from the file checksum as a fallback, but an explicit producer-supplied key remains the safer operational contract.

## Failure handling

Failures are separated into two classes.

### Terminal failures

Examples:

- malformed JSON
- missing required fields
- unsupported payload contract
- request creation rejected by business validation

These move the file to `failed/` and mark the ledger row `failed`.

### Retryable failures

Examples:

- temporary file read failure
- transient database outage
- transient request-service dependency failure

These keep the file in the inbox, clear the active claim, and set a retry time on the ledger row.

## Operational notes

- Writers should publish files into the inbox via atomic rename.
- The worker should avoid logging payload bodies or secrets.
- Worker runs should expose counters such as files detected, claimed, processed, retried, and failed.
- Existing web and desktop observability views can continue to read generic worker status without adding a parallel shell or navigation system.
