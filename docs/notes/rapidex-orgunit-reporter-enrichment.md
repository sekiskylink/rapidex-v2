# Rapidex Org Unit and Reporter Enrichment

Date: 2026-04-21

## Decisions

- Kept the existing Rapidex tables and endpoints:
  - `org_units`
  - `reporters`
  - `/api/v1/orgunits`
  - `/api/v1/reporters`
- Enriched those existing contracts instead of creating a second `organisationunit` schema branch.
- Organisation-unit `uid` values are now treated as DHIS2-style identifiers:
  - 11 characters
  - first character must be a letter
  - remaining characters are alphanumeric
- Organisation-unit materialized paths now use UID segments, for example:
  - `/ImspTQPwCqd/O6uvpzGd5pu/`
- Reporter groups are stored in `reporter_groups` and edited inline in the CRUD dialogs as free-form chips.
- Derived or system-owned fields stay backend-managed in normal CRUD:
  - org-unit `path`, `hierarchyLevel`
  - reporter `reportingLocation`, `districtId`, `totalReports`, `lastReportingDate`, `synced`

## Implementation Notes

- Org-unit create/update now recomputes hierarchy level and path from ancestry, and descendant paths are cascaded when a node is reparented.
- Reporter create/update now derives district and reporting location from the selected organisation unit hierarchy.
- Backend reporter JSON accepts legacy aliases for compatibility during the transition:
  - `displayName` -> `name`
  - `phoneNumber` -> `telephone`
  - `contactUuid` -> `rapidProUuid`
- Web and desktop Facilities/Reporters pages were upgraded from create-only screens to CRUD screens with:
  - row-level edit/delete actions
  - responsive multi-column dialogs
  - inline reporter-group editing
  - read-only derived/system metadata displays

## Follow-ups

- Backfill existing data more richly if legacy production records need organization-specific district derivation beyond the current level-based heuristic.
- Consider extracting the shared Facilities/Reporters dialog logic into reusable web/desktop components if these pages continue to grow.
