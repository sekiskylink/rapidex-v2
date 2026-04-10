# Worker Process Logging

Worker pickup and processing logs use the existing backend structured logger. They are process logs intended for stdout/stderr collection and do not change the API contract or client UI behavior.

## Logged Events

- `delivery_worker_picked`: a send or retry worker claimed a delivery attempt for processing.
- `delivery_worker_completed`: delivery submission finished without a terminal failure or deferral.
- `delivery_worker_deferred`: delivery submission was held for a policy reason, such as a closed submission window.
- `delivery_worker_failed`: delivery submission completed with a failed delivery status.
- `delivery_worker_error`: worker-side claim, load, or submit plumbing failed before normal status handling completed.
- `async_poll_picked`: the poll worker claimed an async task.
- `async_poll_succeeded`: an async poll completed and task state was updated.
- `async_poll_failed`: the remote poller returned a transient poll error and the task stayed eligible for later polling.
- `async_poll_error`: worker-side async claim, poll history, or task status persistence failed.

## Safe Metadata

Worker logs include operational identifiers and routing/status fields only:

- worker run ID and worker type
- request, delivery, and async task IDs/UIDs
- correlation ID
- server ID/code where available
- attempt number
- delivery or async status
- HTTP status and remote status where available
- remote job ID for async polling
- duration fields where available
- sanitized error messages

Worker logs must not include request payload bodies, response bodies, headers, JWTs, refresh tokens, API tokens, passwords, database DSNs, or other secrets.
