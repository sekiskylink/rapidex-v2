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
- Hierarchy refresh now has two explicit modes:
  - `initialSync=true`
  - `initialSync=false`

## Sync Model

- Hierarchy refresh resolves its source from the existing Sukumad integration-server records for DHIS2.
- The DHIS2 integration server `baseUrl` must be the DHIS2 instance root, including any deployment context path:
  - valid examples: `https://play.im.dhis2.org/dev`, `https://dhis.example.org`, `https://dhis.example.org/dhis`
  - do not include `/api` or a specific metadata endpoint such as `/api/organisationUnits.json`
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

- `initialSync=true` is the destructive bootstrap mode.
- It deliberately clears:
  - `user_org_units`
  - `reporters`
  - existing hierarchy metadata
  - existing `org_units`
- `initialSync=false` is the normal steady-state refresh mode.
- In steady-state refresh:
  - existing org units are rebuilt from DHIS2
  - existing reporters are matched back to facilities by DHIS2 org-unit UID
  - matched reporters are remapped to the rebuilt local `org_units.id`
  - unmatched reporters are preserved and marked orphaned instead of being deleted
  - `user_org_units` assignments are rebuilt by DHIS2 org-unit UID where possible
  - unmatched user-org assignments are dropped because they cannot be represented without a target org unit
- Reporter orphaning is stored directly on the reporter record:
  - nullable `org_unit_id`
  - `orphaned_at`
  - `orphan_reason`
  - `last_known_org_unit_uid`
  - `last_known_org_unit_name`
- Sync-state persistence records run timing, status, errors, source server, district-level settings, and row counts so operators can inspect the most recent run from the UI.

## Reporter Sync Fallout

- Reporter facility changes now trigger an automatic RapidPro contact refresh attempt after the local update succeeds.
- The local reporter update remains authoritative:
  - the reporter is first marked `synced=false`
  - RapidPro refresh is attempted afterward
  - a failed RapidPro refresh does not roll back the local facility update
- Hierarchy reconcile refresh also marks remapped or orphaned reporters unsynced so downstream RapidPro synchronization can observe the changed facility context.

## Scheduler and UI

- Hierarchy refresh is exposed as a scheduler integration job type:
  - `dhis2_org_unit_refresh`
- The default seeded scheduler job is intentionally disabled and uses:
  - `serverCode: "dhis2"`
  - `districtLevelName: "District"`
  - `fullRefresh: true`
  - `dryRun: false`
- Both web and desktop Facilities pages now expose a manual sync dialog and the latest sync-state summary.
- Both web and desktop scheduler forms now expose the `initialSync` choice explicitly.

## Follow-ups

- If production payload size makes single-transaction replacement too heavy, move the refresh path to staging tables plus a final swap transaction.
- Consider a dedicated queue-backed reporter RapidPro resync worker if facility-change sync volume becomes high enough that detached best-effort sync is no longer sufficient.
