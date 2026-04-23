# Rapidex RapidPro Reporter Sync

## Summary
- Added a RapidPro client under `backend/internal/sukumad/rapidex/rapidpro` for contacts, groups, direct messages, and broadcasts.
- Extended Rapidex reporters with explicit RapidPro sync and messaging flows, plus a scheduler job for incremental synchronization.
- Kept the integration inside existing BasePro/Sukumad seams:
  - RapidPro connection details come from the existing integration servers module using reserved server code `rapidpro`
  - reporter routes continue using existing auth and RBAC
  - scheduler watermarking reuses scheduled-job `last_success_at`

## Reporter sync behavior
- Reporters sync to RapidPro by local `rapidpro_uuid` first when present.
- If a stored local UUID no longer exists remotely, the sync falls back to phone-based lookup instead of failing immediately.
- If no local UUID is present, the sync attempts RapidPro contact lookup by normalized `tel:` URN.
- If a contact exists remotely, Rapidex updates it and persists the returned UUID locally.
- If no contact exists, Rapidex creates it and stores the created UUID in `reporters.rapidpro_uuid`.
- Successful syncs mark the reporter as `synced=true`.
- Reporter group names are normalized and mapped to existing RapidPro groups.
- Sync now fails with actionable validation detail when a reporter references a group name that does not already exist in RapidPro.
- The contact upsert flow no longer mixes URL URN lookup mode with body `urns`, which RapidPro rejects with HTTP 400.

## Reporter field mapping settings
- Added a server-backed RapidPro reporter sync settings payload under the existing Settings module.
- Operators can refresh available RapidPro contact fields from the configured integration server and save reusable mappings once.
- Settings now expose a per-reporter RapidPro sync preview so operators can inspect the exact outbound request body, query mode, and resolved groups before syncing.
- Saved mappings are applied uniformly to:
  - row sync
  - bulk sync
  - scheduled `rapidpro_reporter_sync` jobs
- The current default suggestions are:
  - `Facility` <- linked org unit `name`
  - `FacilityCode` <- linked org unit `uid`
- Sync now fails early with validation detail when:
  - a saved RapidPro field mapping points to a field that no longer exists remotely
  - a mapped reporter/facility value is missing for the current reporter
  - a mapped reporter group does not exist remotely in RapidPro
- RapidPro contact upsert now includes the mapped `fields` payload in addition to the existing contact name, URNs, and groups.

## Messaging behavior
- Single-reporter SMS uses the RapidPro messages endpoint.
- Multi-reporter sends use the RapidPro broadcasts endpoint.
- Reporters page actions were added for:
  - row sync
  - row SMS send
  - bulk sync
  - bulk broadcast

## Scheduler behavior
- Added scheduler job type `rapidpro_reporter_sync`.
- Job config supports:
  - `batchSize`
  - `dryRun`
  - `onlyActive`
- First successful run performs a full sync when no prior `last_success_at` exists.
- Subsequent runs only scan reporters with `updated_at > last_success_at`.
- Result summaries include watermark, scanned count, synced count, created/updated counts, failed count, and dry-run state.
- Startup now ensures a default integration scheduler job exists for RapidPro reporter sync when scheduler seeding is enabled.

## UI updates
- Web and desktop navigation now use distinct icons for every visible Rapidex link, including the parent Sukumad section and the new org-unit/reporter links.
- Web and desktop Reporters pages now expose row actions and bulk actions for RapidPro operations while keeping `rapidProUuid` read-only in the edit form.
- Web and desktop scheduler forms now support creating and editing `rapidpro_reporter_sync` jobs.

## Reporter details and chat history
- Added backend reporter read endpoints for RapidPro-linked detail views:
  - `GET /api/v1/reporters/:id/rapidpro-contact`
  - `GET /api/v1/reporters/:id/chat-history`
- Reporter contact detail lookup reuses the existing sync identity rules:
  - prefer stored `rapidpro_uuid` when present and valid
  - fall back to normalized `tel:` URN lookup when the UUID is missing or stale
- RapidPro contact snapshots now expose richer read fields for UI inspection, including:
  - status
  - language
  - custom fields
  - current flow
  - groups
  - created/modified/last-seen timestamps
- Reporter chat history reads recent RapidPro messages and normalizes them into a UI-safe conversation timeline with:
  - direction
  - status
  - text
  - channel
  - flow
  - created/sent/modified timestamps
- The current implementation filters the recent RapidPro message page to the reporter conversation and does not yet provide message pagination.
- Web and desktop Reporters pages now:
  - use an informational detail dialog for `View details`
  - expose a dedicated `View RapidPro Details` action
  - open a conversation-style chat history dialog from the `Telephone` column
