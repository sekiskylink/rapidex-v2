# Sukumad Request Payload Transport

## Purpose

This note defines how Sukumad request payloads are stored and translated into outbound HTTP submissions.

The system must support both traditional request bodies and payloads that are converted into query parameters for the destination URL.

## Request contract

Each request now carries two transport fields:

- `payloadFormat`
  - `json`
  - `text`
- `submissionBinding`
  - `body`
  - `query`

These fields are request-level metadata.

They do not replace server-level defaults such as integration server headers or default URL params.

## Supported combinations

### 1. `payloadFormat=json`, `submissionBinding=body`

Current default behavior.

- payload must be valid JSON
- outbound request body is the compact JSON
- default `Content-Type` is `application/json`

### 2. `payloadFormat=text`, `submissionBinding=body`

Plain text body submission.

- payload must be a non-empty string
- outbound request body is the trimmed text
- default `Content-Type` is `text/plain`

### 3. `payloadFormat=json`, `submissionBinding=query`

JSON object converted into URL query params.

- payload must be a JSON object
- values may be strings, numbers, booleans, null, or arrays of those values
- no outbound request body is sent
- default `Content-Type` is not added unless configured explicitly in server headers

### 4. `payloadFormat=text`, `submissionBinding=query`

Raw querystring text converted into URL query params.

- payload must be a non-empty valid querystring
- no outbound request body is sent
- default `Content-Type` is not added unless configured explicitly in server headers

## Query param precedence

Outbound URL construction applies params in this order:

1. existing query params already present in the integration server base URL
2. server default `urlParams`
3. request-derived query params

Request-derived query params win on key collisions.

## Persistence

The request record persists:

- `payload_body`
- `payload_format`
- `submission_binding`

This keeps the original operator input durable for retries, worker pickup, and auditability.

## Worker behavior

The worker-owned delivery path must pass the persisted transport fields into the dispatcher.

This ensures retries and deferred deliveries behave the same way as the original request submission contract.

## UI behavior

Both web and desktop request forms must expose:

- payload format selector
- send-as selector

The payload editor label and helper text should adapt to the chosen combination.
