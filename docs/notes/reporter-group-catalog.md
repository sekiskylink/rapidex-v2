# Reporter Group Catalog

## Summary

RapidEx now owns the canonical reporter-group catalog. Reporter records can only reference active groups from this catalog, and RapidPro is treated as a synchronized downstream dependency rather than the source of truth.

## Key Decisions

- The catalog lives in RapidEx under `reporter_group_catalog`.
- Existing distinct `reporter_groups.group_name` values are backfilled into the catalog during migration.
- Reporter create/update validates selected groups against active catalog entries.
- Active group creation or rename provisions the corresponding RapidPro group immediately.
- Reporter sync defensively creates missing RapidPro groups for active catalog entries before syncing the contact.
- Reporter forms use a predefined multi-select autocomplete sourced from `GET /reporter-groups/options`.

## Operational Notes

- Deactivating a group prevents new reporter assignments, but existing reporter records still retain the stored group name until edited.
- Group rename provisions the new RapidPro group name but does not attempt to rename or remove the old RapidPro group remotely.
- Current management is intentionally simple: create, rename, and activate/deactivate from Settings.
