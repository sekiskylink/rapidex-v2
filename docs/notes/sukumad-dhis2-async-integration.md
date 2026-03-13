# Sukumad DHIS2 Async Integration

## Scope

Milestone 6 adds the first real outbound integration target for SukumadPro. The implementation keeps the target-specific behavior inside `backend/internal/sukumad/dhis2` and lets the existing request, delivery, async-task, and worker modules remain the orchestration layers.

## Flow

1. A request is created against a persisted Sukumad server.
2. The request service creates the initial delivery attempt.
3. The delivery service loads the server snapshot and dispatches through the DHIS2 service.
4. The DHIS2 service submits the HTTP request using server-configured URL, headers, params, and method.
5. A synchronous DHIS2 response is mapped directly to terminal delivery success or failure.
6. An asynchronous DHIS2 response creates a linked async task with remote job metadata and keeps the delivery awaiting polling.
7. The async worker polls due tasks through the DHIS2 service.
8. Terminal poll results reconcile back into the linked delivery and then the request roll-up status.

## Responsibilities

- `dhis2.Service`
  - Knows how to submit to DHIS2 and interpret DHIS2 response shapes.
  - Returns normalized dispatch and poll results instead of mutating persistence directly.
- `delivery.Service`
  - Owns delivery lifecycle transitions and async-task creation.
  - Emits audit events for submission start, success, failure, and async task creation.
- `async.Service`
  - Owns polling state transitions, poll history, retry scheduling, and terminal reconciliation.
- `request.Service`
  - Owns request roll-up status transitions.

## Design Constraints

- No DHIS2 logic was added to handlers.
- No new top-level Sukumad modules were introduced outside `backend/internal/sukumad`.
- No credentials are returned in API payloads or written to logs.
- Desktop and web continue to use the shared backend APIs only.

## Current Output Shape

Request and delivery detail responses now expose latest async linkage fields such as:

- latest delivery UID and status
- latest async task UID and state
- remote DHIS2 job ID
- internal poll URL when present
- whether the entity is awaiting async completion

This keeps the UI target-aware without teaching the clients DHIS2 response semantics directly.
