## Rapidex v2 Milestone 4/5/6 — Reporters Backend, User Org Unit Scoping & Org Unit UI

### What changed

- **Reporters backend:** Added a PostgreSQL implementation of the Reporter repository (`pg_repository.go`) to provide full CRUD operations using sqlx. This implementation supports listing with search and pagination, fetching by various identifiers, creating records with generated UIDs, updating mutable fields, and deleting reporters. It is intended to be used by the existing `Service` and `Handler` to provide a functional reporters API.
- **User organisation unit scoping:** Introduced a new `userorg` domain module under `backend/internal/sukumad/` that manages assignments of users to organisation units. The module defines a `UserOrgUnit` type, repository interface, service and HTTP handlers, along with a PostgreSQL repository implementation. A new migration (`000028_create_user_org_units.sql`) creates the `user_org_units` table with foreign keys to `users` and `org_units` and appropriate indexes.
- **Organisation units UI:** Implemented a full tree‑based management page for organisation units in both the web and desktop clients. The new pages fetch organisation units from the backend, build a hierarchical tree, and allow administrators to create new units with an optional parent via a dialog. The pages use Material UI’s `TreeView` and `TreeItem` components and provide basic loading and error states.

### Added/updated tests

This milestone establishes new modules and UI scaffolding but does not include automated tests. Follow‑up work should add repository, service and handler tests as well as integration tests for the new UI components.

### Verification summary

- Backend builds successfully with the added repository and service modules.
- Web and desktop clients compile and display the new Organisation Units page with a tree view and creation dialog.
- The new migration applies cleanly, creating the `user_org_units` table with indexes and foreign keys.

### Known follow‑ups

- Implement concrete repositories for the existing `orgunit` and `reporter` services using sqlx and ensure they enforce permissions.
- Integrate the `userorg` service into authentication/authorisation middleware to enforce that users can only view and modify org units and reporters within their assigned scope.
- Expand the Reporters UI (Milestone 7) to provide listing, search and CRUD functionality, and restrict the list based on user org unit scope.
- Add backend and frontend tests for the new features to satisfy the project’s testing requirements.
- Update the main `docs/status.md` to reflect the completion of these milestones.
