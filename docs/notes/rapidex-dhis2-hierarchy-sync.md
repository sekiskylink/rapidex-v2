# RapidEx DHIS2 Hierarchy Refresh

Date: 2026-04-24

## Decisions

- `org_units` remains the canonical RapidEx organisation-unit table.
- DHIS2 hierarchy metadata is stored alongside it in dedicated tables:
  - `org_unit_levels`
  - `org_unit_groups`
  - `org_unit_attributes`
  - `org_unit_group_members`
- District is no longer treated as a fixed local hierarchy level.
- District derivation now uses configured DHIS2 level metadata recorded on the most recent successful or attempted hierarchy sync.
- The first rollout mode is a full destructive refresh instead of an incremental merge.

## Sync Model

- Hierarchy refresh resolves its source from the existing Sukumad integration-server records for DHIS2.
- Each run fetches DHIS2 metadata for:
  - organisation unit levels
  - organisation unit groups
  - organisation unit attributes flagged for organisation units
  - organisation units
- The sync service validates the fetched hierarchy before writing live data:
  - parent references must resolve
  - DHIS2 paths are normalized to the local UID-path format
  - the configured district level must exist by level name or fallback code
- `dryRun` performs the fetch and validation path without deleting or replacing live data.

## Refresh Semantics

- The initial full refresh deliberately clears:
  - `user_org_units`
  - `reporters`
  - existing hierarchy metadata
  - existing `org_units`
- This matches the current operational requirement to rebuild the local facility hierarchy cleanly from DHIS2 before re-establishing downstream assignments.
- Sync-state persistence records run timing, status, errors, source server, district-level settings, and row counts so operators can inspect the most recent run from the UI.

## Scheduler and UI

- Hierarchy refresh is exposed as a scheduler integration job type:
  - `dhis2_org_unit_refresh`
- The default seeded scheduler job is intentionally disabled and uses:
  - `serverCode: "dhis2"`
  - `districtLevelName: "District"`
  - `fullRefresh: true`
  - `dryRun: false`
- Both web and desktop Facilities pages now expose a manual sync dialog and the latest sync-state summary.

## Follow-ups

- If production payload size makes single-transaction replacement too heavy, move the refresh path to staging tables plus a final swap transaction.
- Add automatic reassignment/remapping only after the rebuilt hierarchy and district-level semantics are stable in real data.
