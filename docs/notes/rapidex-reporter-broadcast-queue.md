# RapidEx Reporter Broadcast Queue

## Summary

RapidEx reporter broadcasts can now be queued by jurisdiction scope instead of only by explicit reporter selection. The queue-backed flow allows large matching sets to be processed in the background while preventing near-duplicate submissions from the same operator.

## Backend shape

- Endpoint: `POST /api/v1/reporters/broadcasts`
- Request:
  - `orgUnitIds: number[]`
  - `reporterGroup: string`
  - `text: string`
- Response states:
  - `queued`
  - `duplicate_pending`

## Scope rules

- Selected organisation units are treated as a union.
- Non-leaf organisation units include descendant facilities/reporters by path.
- Final recipients are intersected with the current user’s allowed org-unit scope.
- Reporter group filtering is applied after org-unit scope resolution.
- Matching reporters are de-duplicated before RapidPro send.

## Queue and duplicate suppression

- Queue state is stored in `reporter_broadcasts`.
- A broadcast dedupe key is derived from:
  - requesting user
  - sorted org-unit IDs
  - normalized reporter group
  - normalized message hash
- If an identical request is already queued or running within 15 minutes, the API returns `duplicate_pending` and does not enqueue another broadcast.

## Worker behavior

- A dedicated Sukumad worker claims queued reporter broadcasts.
- The worker resolves recipients, syncs eligible reporters to RapidPro, and sends the resulting broadcast asynchronously.
- Broadcast records capture matched/sent/failed counts and terminal status for operator visibility and troubleshooting.
