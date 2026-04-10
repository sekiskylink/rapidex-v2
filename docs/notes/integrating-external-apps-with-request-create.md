# Creating and Tracking Sukumad Exchanges from Another App

This document describes the external integration contract for creating and polling Sukumad exchange requests without using internal numeric IDs.

Use the external request endpoints when integrating another system.

- create: `POST /api/v1/external/requests`
- get status by request UID: `GET /api/v1/external/requests/:uid`
- lookup by correlation or idempotency or batch: `GET /api/v1/external/requests/lookup`

These endpoints are designed for service-to-service use and accept API tokens.

## Authentication

External callers can authenticate with either:

- `X-API-Token: <token>`
- `Authorization: Bearer <api-token>`

Required permissions:

- create requests: `requests.write`
- read status: `requests.read`

Recommended pattern:

- create a dedicated API token for the integrating system
- scope it only to the request permissions it needs
- do not reuse a human operator login for machine integration

## External Create Contract

### Request

```http
POST /api/v1/external/requests
X-API-Token: <token>
Content-Type: application/json

{
  "sourceSystem": "emr",
  "destinationServerUid": "srv-primary",
  "destinationServerUids": ["srv-copy-1", "srv-copy-2"],
  "dependencyRequestUids": ["req-dependency-1", "req-dependency-2"],
  "batchId": "batch-2026-04-08-001",
  "correlationId": "emr-sync-2026-04-08-001",
  "idempotencyKey": "emr-sync-12345",
  "payloadFormat": "json",
  "submissionBinding": "body",
  "urlSuffix": "/api/data",
  "payload": {
    "trackedEntity": "123",
    "orgUnit": "OU_1"
  },
  "metadata": {
    "priority": "high",
    "sourceApp": "emr"
  }
}
```

### Required fields

- `sourceSystem`
  - stable identifier for the upstream caller, for example `emr`
- `destinationServerUid`
  - public Sukumad server UID, not the internal numeric ID
- `payload`
  - the outbound payload

### Optional fields

- `destinationServerUids`
  - additional destination server UIDs for fan-out delivery
- `dependencyRequestUids`
  - public request UIDs that must complete before this request can proceed
- `batchId`
  - grouping key for a set of related requests
- `correlationId`
  - tracing key shared with the upstream system
- `idempotencyKey`
  - caller-supplied deduplication key
- `payloadFormat`
  - `json` or `text`
- `submissionBinding`
  - `body` or `query`
- `urlSuffix`
  - optional downstream path suffix
- `metadata`
  - caller-defined JSON metadata

## Idempotency Semantics

Idempotency is enforced by:

- `sourceSystem`
- `idempotencyKey`

That pair maps to one logical exchange request.

Behavior:

- first create returns `201 Created`
- replay of the same `sourceSystem + idempotencyKey` returns `200 OK`
- the replay response returns the existing request, not a new duplicate

If you want deterministic reconciliation from another system, always send both:

- `sourceSystem`
- `idempotencyKey`

## Payload Rules

### When `payloadFormat = "json"`

- `payload` must be valid JSON
- if `submissionBinding = "query"`, the payload must be a JSON object
- query payload values may be strings, numbers, booleans, null, or arrays of those scalar values

### When `payloadFormat = "text"`

- `payload` must be a non-empty string
- if `submissionBinding = "query"`, the string must be a valid query string

## What Happens After Create

When the API accepts the request, Sukumad:

1. creates one `exchange_requests` record
2. creates one `request_targets` row per destination server
3. creates one initial pending delivery attempt per target
4. returns the created request immediately

Request creation is accept-and-persist first. Outbound submission is performed later by the worker flow.

## Create Response

The external response exposes only public identifiers.

```json
{
  "uid": "0f2c7f8d-8f13-4d27-b2d0-0ac4a1675be2",
  "sourceSystem": "emr",
  "destinationServerUid": "srv-primary",
  "destinationServerCode": "dhis2-prod",
  "destinationServerName": "DHIS2 Production",
  "batchId": "batch-2026-04-08-001",
  "correlationId": "emr-sync-2026-04-08-001",
  "idempotencyKey": "emr-sync-12345",
  "payloadFormat": "json",
  "submissionBinding": "body",
  "urlSuffix": "/api/data",
  "status": "pending",
  "statusReason": "",
  "metadata": {
    "priority": "high",
    "sourceApp": "emr"
  },
  "awaitingAsync": false,
  "targets": [
    {
      "uid": "50d10f74-602b-4af4-9cb6-58e23f7db8d1",
      "destinationServerUid": "srv-primary",
      "destinationServerCode": "dhis2-prod",
      "destinationServerName": "DHIS2 Production",
      "targetKind": "primary",
      "priority": 1,
      "status": "pending",
      "blockedReason": "",
      "latestDelivery": {
        "uid": "80bb2d84-ff71-4fb0-b0ce-1ad21367a83d",
        "status": "pending"
      },
      "latestAsyncTask": {
        "uid": "",
        "state": "",
        "remoteJobId": "",
        "pollUrl": ""
      },
      "awaitingAsync": false
    }
  ],
  "dependencies": [],
  "createdAt": "2026-04-08T10:15:00Z",
  "updatedAt": "2026-04-08T10:15:00Z"
}
```

Notes:

- internal numeric IDs are intentionally omitted
- request body `metadata` is returned as `metadata`
- per-target status is included so the upstream system can track destination fan-out

## Polling Status

### Get by request UID

```http
GET /api/v1/external/requests/0f2c7f8d-8f13-4d27-b2d0-0ac4a1675be2
X-API-Token: <token>
```

Use this when the caller already has the request UID from create.

### Lookup by correlation ID

```http
GET /api/v1/external/requests/lookup?correlationId=emr-sync-2026-04-08-001
X-API-Token: <token>
```

Use this when the upstream system correlates by a shared trace key.

### Lookup by idempotency key

```http
GET /api/v1/external/requests/lookup?sourceSystem=emr&idempotencyKey=emr-sync-12345
X-API-Token: <token>
```

Use this for deterministic deduplication reconciliation.

### Lookup by batch ID

```http
GET /api/v1/external/requests/lookup?batchId=batch-2026-04-08-001
X-API-Token: <token>
```

Use this to fetch all requests that belong to the same upstream batch.

## Status Semantics

Request roll-up statuses:

- `pending`
- `blocked`
- `processing`
- `completed`
- `failed`

Target-level destination statuses:

- `pending`
- `blocked`
- `processing`
- `succeeded`
- `failed`

Delivery status is exposed via `targets[].latestDelivery.status`.

Async progress is exposed via:

- `targets[].latestAsyncTask.uid`
- `targets[].latestAsyncTask.state`
- `targets[].latestAsyncTask.remoteJobId`
- `targets[].latestAsyncTask.pollUrl`

This lets external callers track:

- overall exchange status
- status per destination server
- whether a destination is waiting on asynchronous completion

## Example cURL

### Create

```bash
curl -sS \
  -X POST "http://127.0.0.1:8080/api/v1/external/requests" \
  -H "X-API-Token: <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "sourceSystem": "emr",
    "destinationServerUid": "srv-primary",
    "correlationId": "emr-sync-2026-04-08-001",
    "idempotencyKey": "emr-sync-12345",
    "payloadFormat": "json",
    "submissionBinding": "body",
    "payload": {
      "trackedEntity": "123",
      "orgUnit": "OU_1"
    },
    "metadata": {
      "sourceApp": "emr"
    }
  }'
```

### Poll by request UID

```bash
curl -sS \
  "http://127.0.0.1:8080/api/v1/external/requests/<request-uid>" \
  -H "X-API-Token: <token>"
```

### Lookup by idempotency

```bash
curl -sS \
  "http://127.0.0.1:8080/api/v1/external/requests/lookup?sourceSystem=emr&idempotencyKey=emr-sync-12345" \
  -H "X-API-Token: <token>"
```

## Example JavaScript

```js
async function createExchange(baseUrl, apiToken, payload) {
  const response = await fetch(`${baseUrl}/external/requests`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Token': apiToken,
    },
    body: JSON.stringify(payload),
  })

  if (!response.ok) {
    throw new Error(`Create request failed: ${response.status} ${await response.text()}`)
  }

  return response.json()
}

async function getExchange(baseUrl, apiToken, requestUid) {
  const response = await fetch(`${baseUrl}/external/requests/${requestUid}`, {
    headers: {
      'X-API-Token': apiToken,
    },
  })

  if (!response.ok) {
    throw new Error(`Get request failed: ${response.status} ${await response.text()}`)
  }

  return response.json()
}
```

## Error Handling

Common validation errors include:

- unknown `destinationServerUid`
- unknown `destinationServerUids`
- unknown `dependencyRequestUids`
- invalid payload for the selected `payloadFormat` and `submissionBinding`
- missing `sourceSystem`

If an idempotent replay occurs, the API returns the existing request rather than a validation error.
