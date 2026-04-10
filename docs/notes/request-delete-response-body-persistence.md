# Request Delete and Response Body Persistence

## Summary

Exchange requests can be deleted from the authenticated Requests page. Deletion is implemented through the Sukumad request service and repository layers and removes the request plus dependent Sukumad domain rows in one database transaction.

The response body persistence policy is configured on integration servers and can be overridden per exchange request:

- `filter`: persist response bodies only when the existing response filter allows the content type.
- `save`: persist the response body regardless of response filter content-type rules.
- `discard`: do not persist the response body.

Requests use the server policy by default. A request override is optional and is stored as an empty string when the request should inherit the integration server setting.

## Deletion Scope

`DELETE /api/v1/requests/:id` requires `requests.write` and deletes:

- async task polls for deliveries tied to the request
- async tasks for deliveries tied to the request
- request events
- delivery attempts
- request targets
- request dependency edges where the request is the source or dependency
- the exchange request row

The delete service writes an audit event with request identity and routing metadata only. It does not place request payloads, response bodies, headers, or credentials into audit metadata.

## Persistence Policy Flow

Integration servers store `response_body_persistence` with a default of `filter`. Request creation can send `responseBodyPersistence` as `filter`, `save`, `discard`, or blank for the server default.

Delivery resolution applies the effective policy when saving DHIS2 submission and polling responses:

- request override when present
- otherwise the integration server default
- fallback to `filter` for legacy rows

`discard` intentionally clears the saved response body and marks it filtered. `filter` retains the prior response filter behavior. `save` stores the response body and marks it unfiltered.

