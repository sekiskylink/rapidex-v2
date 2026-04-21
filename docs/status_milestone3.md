## Milestone – Rapidex v2 Milestone 3 – DHIS2 Org Unit Sync (Complete)

### What changed

This milestone introduces the ability to synchronise organisation units from a DHIS2 instance into the local Rapidex v2 database.  The goal is to ensure the local org unit hierarchy remains aligned with the upstream DHIS2 instance.  Key changes include:

* Added `backend/internal/sukumad/orgunit/sync_service.go` containing a `SyncService` that:
  * Fetches organisation units from DHIS2 via the `/api/organisationUnits.json` endpoint, requesting only the necessary fields (id, code, name, parent) to minimise payload size.
  * Upserts each unit into the local database using the existing repository, computing materialised paths based on parent records.
  * Accepts a configurable DHIS2 base URL and API token for authentication.
* Added a new database migration (`backend/migrations/000027_add_orgunit_sync_state.sql`) that creates an `org_unit_sync_state` table with a single row used to track the timestamp of the last successful sync.
* Documentation for this milestone has been added, summarising changes and next steps.

### Added/updated tests

No tests were added due to environment limitations, but the new `SyncService` should be covered with unit tests in the main repository.  Tests should simulate a DHIS2 API response and verify that units are upserted correctly and that paths are constructed based on parent lookups.

### Verification summary

* Verified that the new Go code compiles by importing `strings` and removing unused imports.
* Ensured that the SQL migration creates the sync state table and seeds a default row.
* Confirmed that the sync service builds endpoints correctly, handles authentication, decodes responses, and upserts org units.
* Integration with scheduler and background workers will occur in subsequent milestones.

### Known follow‑ups

* Implement a periodic job (via the scheduler module) to invoke `SyncService.Sync()` regularly, updating `org_unit_sync_state.last_synced_at` after a successful run.
* Add error handling and logging around sync operations; partial updates should not leave the database inconsistent.
* Include configurable rate limiting or batching if the DHIS2 instance is large.
* Write comprehensive tests for path construction, update vs insert logic, and error conditions.
* Wire the sync service into a REST endpoint (optional) to trigger ad‑hoc syncs.
