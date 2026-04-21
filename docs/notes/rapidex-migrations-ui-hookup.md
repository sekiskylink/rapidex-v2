# Rapidex Migration and UI Hookup

Date: 2026-04-21

Rapidex organisation units/facilities and reporters are exposed as Sukumad-domain extensions while reusing the existing BasePro/Sukumad app shell, RBAC, module enablement, and API conventions.

## Decisions

- Rapidex migration files use the existing `backend/migrations` directory and golang-migrate `.up.sql` / `.down.sql` pairing.
- The 2026 migration versions remain in place and sort after the existing numbered BasePro/Sukumad migrations.
- API routes are mounted directly under `/api/v1`, matching existing Sukumad resources:
  - `/api/v1/orgunits`
  - `/api/v1/reporters`
  - `/api/v1/user-org-units`
- UI labels use `Facilities` for organisation units and `Reporters` for reporter management.
- Rapidex engine execution remains deferred until after the schema, route, RBAC, and web/desktop navigation foundation is stable.

## Follow-ups

- Enforce descendant org-unit scoping inside list/create/update flows.
- Expand facility and reporter pages beyond first-pass create/list surfaces.
- Validate the Rapidex mapping engine and Sukumad exchange-request creation in the next milestone.
