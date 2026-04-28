# RapidEx Webhook Request Queueing

## Summary

RapidEx webhook processing now queues normal Sukumad exchange requests instead of logging a placeholder payload.

The execution path is:

RapidPro webhook
-> RapidEx settings-backed mapping lookup
-> DHIS2 aggregate payload mapping
-> request queueing through `request.Service.CreateExternalRequest`
-> existing delivery workers
-> DHIS2 `/api/dataValueSets`

## Queueing contract

- Endpoint: `POST /api/v1/rapidex/webhook`
- Auth: existing API-token auth plus `requests.write`
- Source system: `rapidex-webhook`
- Payload format: `json`
- Submission binding: `body`
- URL suffix: `/api/dataValueSets`
- Destination server: resolved from the saved RapidEx `dhis2ServerCode`

## Idempotency and metadata

- Webhook queueing derives a deterministic idempotency key from the webhook body so duplicate deliveries dedupe in the request layer.
- Correlation IDs use `rapidex:<flow_uuid>:<contact_uuid>` when the contact UUID is present, otherwise they fall back to a hashed webhook suffix.
- Request extras include:
  - `flowUuid`
  - `flowName`
  - `contactUuid`
  - `rapidProServer`
  - `dhis2Server`
  - `mappedDataset`
  - `mappedOrgUnit`
  - `mappedPeriod`
  - `mappedValueCount`
  - `msisdn` when a `tel:` URN is present

`msisdn` is stored as the international phone number without the `tel:` prefix, for example `+256782820208`.

## Current scope boundary

- Reporter lookup is not part of this milestone.
- The mapped `orgUnit` is queued exactly as resolved from the saved RapidEx mapping.
- Unknown or incomplete mappings fail validation before queueing.
