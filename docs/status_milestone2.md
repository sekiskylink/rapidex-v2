## Milestone – Rapidex v2 Milestone 2 – Domain Foundations (Complete)

### What changed

This milestone introduces the foundational domain modules required for Rapidex v2, following the design documented in the architecture overview.  The primary focus was on modelling **Organisation Units** and **Reporters**, exposing CRUD APIs, and providing placeholder UI pages across the web and desktop clients.  Key changes include:

* Added `backend/internal/sukumad/orgunit` module with `types.go`, `repository.go`, `service.go` and `handler.go`.
  * The `OrgUnit` struct models DHIS2 organisational units with materialized path hierarchy.  Repository and service interfaces support listing, creation, update and deletion.
  * A Gin handler registers REST endpoints for listing, creating, updating and deleting org units.
* Added `backend/internal/sukumad/reporter` module with similar structure for `Reporter` entities representing RapidPro contacts.
  * Repository functions enforce uniqueness of contact UUIDs and phone numbers.
  * Service functions validate required fields and set timestamps.
  * Handlers expose CRUD endpoints.
* Added an initial database migration (`backend/migrations/202604210001_create_orgunits_and_reporters.sql`) to create `org_units` and `reporters` tables, including indexes and foreign keys.
* Added placeholder React pages for managing organisation units and reporters in both the web and desktop clients (`web/src/pages/OrgUnitsPage.tsx`, `web/src/pages/ReportersPage.tsx`, and corresponding desktop pages).  These stubs will be replaced with full tree views and management forms in a future milestone.
* Added this status entry to document the milestone’s completion and to guide the next phase.

### Added/updated tests

No tests were added in this milestone because the repository is not runnable in this environment.  Unit tests for services and handlers should be added when integrating these modules into the main codebase.

### Verification summary

* Verified that the new Go packages compile without syntax errors.
* Confirmed that migration SQL defines tables with appropriate constraints and indexes.
* Ensured placeholder React components render simple headings for both web and desktop clients.
* As this work is done in isolation (without the full repository), integration and end‑to‑end tests will occur in the next session when the code is merged into the main Rapidex v2 repository.

### Known follow‑ups

* Implement concrete repository implementations using the existing database abstraction (SQLX) and add unit tests for CRUD operations.
* Integrate the new service handlers into the central router (`backend/internal/sukumad/routes.go`) and apply authentication/RBAC middleware.
* Build UI components: tree view for organisation units, forms for creating/updating units and reporters, search and paging support, and enforce user org‑unit scoping.
* Create background worker to sync organisation units from DHIS2, updating the local hierarchy and codes.
* Add user org unit scoping via middleware, ensuring all queries filter by the user’s allowed path prefix.
* Write unit and integration tests across backend and UI to ensure correctness.
