# Sukumad Delivery Engine

## Purpose

This note records the Milestone 4 delivery-engine design inside SukumadPro.

Deliveries represent concrete attempts to send an exchange request to an integration server.

## Data model

Deliveries are stored in `delivery_attempts`.

Each delivery attempt links to:

- one `exchange_requests` row
- one `integration_servers` row

The public inspection fields exposed to clients are:

- delivery UID
- request UID
- server name
- attempt number
- lifecycle status
- response body
- error message
- execution timestamps
- retry timestamp

## Lifecycle

Supported statuses:

- `pending`
- `running`
- `succeeded`
- `failed`
- `retrying`

Allowed transitions implemented in the service:

- `pending` -> `running`
- `retrying` -> `running`
- `running` -> `succeeded`
- `running` -> `failed`
- `failed` -> new delivery attempt with `retrying`

Retries create a new delivery attempt rather than mutating the failed historical attempt.

## Retry behavior

`POST /api/v1/deliveries/:id/retry` only accepts failed deliveries.

The retry operation:

- reads the failed attempt
- creates a new attempt with `attempt_number + 1`
- sets status to `retrying`
- schedules the retry using `retry_at`

This milestone stores the retry schedule and exposes it in the UI.
Worker execution of scheduled retries remains a later milestone.

## Request integration

Request creation now seeds the first pending delivery attempt automatically.

That keeps request intake and delivery tracking vertically connected without adding a second orchestration surface.

## Permissions and audit

RBAC:

- `deliveries.read` for list/detail
- `deliveries.write` for retry

Audit events:

- `delivery.created`
- `delivery.succeeded`
- `delivery.failed`
- `delivery.retry`
