# Desktop Skeleton App (Wails UI + Gin API Backend)
## Authoritative Requirements Specification
## Phase S1 – Skeleton/Foundation

Last Updated: 2026-03-07

---

# 0. Execution Rules (Global)
These rules apply to every milestone and every change.

1) **Milestone gating**
- No milestone may start until the previous milestone is fully complete.
- “Complete” means: all required work is done, **all tests pass (including route tests)**, and `docs/status.md` is updated.

2) **Always read requirements first**
- Before implementing any milestone, the agent must read this `docs/requirements.md`.
- The agent must treat it as the single source of truth.

3) **Status + prompts hygiene**
- At the end of each successful milestone, update `docs/status.md` with what changed and what is now complete.
- Keep a copy of milestone prompts under `docs/prompts/` for traceability **but do not commit prompts**.
  - (Recommended: add `docs/prompts/` to `.gitignore`.)

4) **Commit discipline**
- At the end of every successful milestone, the agent must propose an appropriate **git commit message** and prompt the user to commit.
- Do not suggest committing if tests are failing.

---

# 1. Vision
Build a professional desktop application shell (Wails) that never connects to the database directly.
All data access must happen through a Gin HTTP API backend.

This skeleton must be reusable for multiple future business apps, not only HR.

---

# 2. Target Architecture

## 2.1 Components

### A) Desktop Client (Wails)
- Wails v2 + Go 1.22+
- Frontend: React + TypeScript + MUI v5+
- TanStack Router + TanStack Query
- MUI X Data Grid (advanced features required; see section 3.7)
- Responsibilities:
  - UI shell, navigation, theming
  - login + refresh token flow
  - local settings storage (API base URL and app preferences)
  - HTTP client wrapper for calling the backend API
  - file dialogs (Save As / Open) via Wails bindings if needed later

### B) Backend API (Gin)
- Go 1.22+
- Gin REST API (/api/v1)
- SQLX for DB access
- golang-migrate with up/down SQL files
- Viper-based config file loading with hot reload (see section 4.7)
- PostgreSQL 13+
- Responsibilities:
  - Authentication (JWT access + refresh rotation)
  - Optional API-token authentication (service/machine use)
  - RBAC: roles + permissions + scoped permissions
  - User management
  - Audit logging (writes to DB)
  - Future domain modules (employees, payroll, etc.)

### C) Optional Web Frontend (React)
Because Wails apps are not a web deployment target, the system must allow a **separate web frontend**
that talks to the same Gin API.
- Reuse as much UI code as practical (shared packages allowed later).
- Web frontend must support the same auth flow and theming behavior, as feasible.

---

# 3. Desktop Client Requirements (Wails)

## 3.1 First-Run Setup (API Base URL)
Before login is possible, the app must allow configuration of:
- API Base URL (e.g., http://127.0.0.1:8080 or https://api.example.com)
- Optional: “API Token” (for environments using token auth instead of username/password)
- Optional: request timeout (seconds)

Storage:
- Persist locally on the machine in an app config file under OS app data dir.
- Do not store secrets in logs.
- MVP: local file with restrictive permissions (best-effort).

## 3.2 Authentication (JWT)
- Login UI exactly as previously defined in the HR requirements (small centered card; no shell visible pre-login).
- Login options:
  - Username + Password (standard)
  - (Optional mode) API Token login (if configured)
- JWT access token stored in memory.
- Refresh token stored in a secure-ish local store (MVP: local file or localStorage; choose one consistent approach).
- Auto-refresh:
  - If access token expires, attempt refresh.
  - If refresh token is invalid/expired/reused, force logout and show:
    "Session expired. Please log in again."

## 3.3 App Shell UI (MUI)
Must look like a sleek MUI admin template:
- Left side navigation (Drawer):
  - supports collapse (mini variant) and mobile overlay behavior
- Top AppBar:
  - user avatar/menu
  - current section title
  - quick access to Appearance and Settings
- Main content area (Outlet)
- Footer (always visible on authenticated pages)

## 3.4 Theme System + Admin Palette
- Theme mode: light | dark | system
- Accent/palette selection:
  - Provide a UI to choose from multiple presets (admin-dashboard feel)
  - Persist locally
- Must be smooth and consistent across pages.

## 3.5 Routes (Skeleton)
Unauthenticated:
- /setup (API base URL + optional API token)
- /login

Authenticated:
- /dashboard (placeholder content)
- /settings (local + server-backed settings later)
- /users (scaffold; only visible if allowed)
- /audit (scaffold; only visible if allowed)

Unknown routes must show a NotFound component (no router warnings).

## 3.6 Settings Page
Settings must include at minimum:
- API Base URL (editable, with Test Connection)
- Auth mode selector (Username/Password vs API Token)
- Appearance: mode + accent preset (can link to the Appearance dialog)
- Placeholder “About” section (version/build)

## 3.7 MUI Data Grid (Advanced)
The app must use **MUI X Data Grid** with advanced capabilities (where applicable):
- Server-side pagination
- Column filtering and sorting (server-driven where needed)
- Column visibility, reordering
- Column pinning (left/right) if available
- Export (CSV at minimum; advanced export when available)
- Density selector
- Persist user table preferences locally (per table)

Note:
- If the project uses DataGridPro/Premium features, ensure the team handles licensing appropriately.
- Skeleton must be structured so tables can “upgrade” to Pro/Premium without refactor.

---

# 4. Backend API Requirements (Gin)

## 4.1 Migrations
- Use golang-migrate with up/down SQL files.
- Use SQLX DB pool.
- Provide a clear migration runner command (Makefile required; see section 7).

## 4.2 Authentication
### 4.2.1 Username/Password + JWT
Endpoints (suggested):
- POST /api/v1/auth/login
- POST /api/v1/auth/refresh
- POST /api/v1/auth/logout
- GET  /api/v1/auth/me

Rules:
- Access token short-lived.
- Refresh token rotation:
  - refresh exchanges old refresh token for a new pair
  - refresh reuse detection must invalidate the session and return a typed error

Error codes (standardized JSON):
- AUTH_UNAUTHORIZED
- AUTH_EXPIRED
- AUTH_REFRESH_REUSED
- AUTH_REFRESH_INVALID

### 4.2.2 API Token Authentication (Machine/Integration)
Backend must also support authenticating requests via an API token:
- Header: X-API-Token: <token>
or:
- Authorization: Bearer <token> (token-type distinguishable server-side)

Use cases:
- automation scripts
- other services calling the API

Token storage:
- Store hashed token in DB.
- Support token rotation (create new, revoke old).
- Tokens may be scoped by permissions.

## 4.3 Authorization: Roles + Permissions + Scoping
Roles (initial):
- Admin
- Manager
- Staff
- Viewer

Permissions:
- Defined as strings, e.g.:
  - users.read, users.write
  - audit.read
  - settings.read, settings.write

Scoping requirement:
- permissions can optionally be scoped to a module (e.g. module=hr, module=payroll)
- MVP: implement “module scope” as an optional column; enforce later as needed

## 4.4 User Management (Server)
Admin-only endpoints:
- GET /api/v1/users
- POST /api/v1/users
- PATCH /api/v1/users/:id (role, active flag)
- POST /api/v1/users/:id/reset-password

## 4.5 Audit Logging
- audit_logs table records:
  - id, timestamp, actor_user_id, action, entity_type, entity_id, metadata_json
- Must record at least:
  - auth.login.success / auth.login.failure
  - auth.refresh
  - auth.logout
  - users.create / users.update / users.reset_password / users.set_active
- Admin-only endpoint:
  - GET /api/v1/audit (with pagination + filters)

## 4.6 Health / Version
- GET /api/v1/health (db connectivity + version)
- GET /api/v1/version

## 4.7 Backend Configuration (Viper + Hot Reload)
The backend must use a config file, loaded via **Viper**, supporting hot reload.

Requirements:
- Config sources:
  1) config file (YAML)
  2) environment variables (override)
  3) flags (override)
- Hot reload:
  - Use fsnotify/Viper WatchConfig.
  - On file change, load and validate the new config.
  - Swap the active config atomically for readers (use `sync/atomic` via `atomic.Value`).
- Read/write caution:
  - Runtime-generated or mutable settings (e.g., secrets generated at runtime) must not “fight” with hot reload.
  - If the app writes any config to disk, implement a safe write strategy (write temp file + atomic rename) and prevent reload loops.
  - Keep “backend runtime state” separate from “configuration”.

---

# 5. Graceful Shutdown + Context
Both desktop and backend processes must:
- Use `context.Context` properly to propagate cancellation.
- Handle SIGTERM and Ctrl+C (SIGINT) using `signal.NotifyContext`.
- Ensure HTTP server shuts down gracefully:
  - stop accepting new requests
  - allow in-flight requests to complete within a timeout
- Ensure background goroutines exit on context cancellation.

---

# 6. Non-Functional Requirements
- Clean architecture:
  - handlers -> services -> repositories
- Parameterized SQL only
- Consistent error response shape
- Tests:
  - backend: auth + RBAC + audit + config reload behavior
  - frontend: routing + auth flow smoke tests (including NotFound route behavior)

---

# 7. Makefiles (Required)
A Makefile must exist (at least at repo root), to simplify routine commands, for example:
- `make backend-run`
- `make backend-test`
- `make migrate-up` / `make migrate-down`
- `make desktop-dev`
- `make desktop-test`
- `make web-dev` (if web frontend exists)

---

# 8. Repository Structure (Recommended)

repo/
  backend/
    cmd/api/
    internal/
      config/
      db/
      auth/
      rbac/
      users/
      audit/
      middleware/
    migrations/
  desktop/
    frontend/
    internal/ (wails bindings: config store, api client, file dialogs)
    wails.json
  web/ (optional, React app targeting browsers)
  docs/
    requirements.md
    status.md
    prompts/   (not committed)
    notes/
  Makefile

---

# Phase S2 — Platform Hardening & Release Readiness

## 9. Packaging & Versioning

### 9.1 Backend Versioning
- Build-time version injection (commit, tag, build date)
- /api/v1/version returns:
  - version
  - commit
  - build date

### 9.2 Desktop Version Display
- About page shows:
  - Desktop version
  - Backend version (if reachable)

### 9.3 CI Requirements
- Backend tests must pass
- Frontend tests must pass
- Frontend build must succeed
- CI fails on any test failure

## 10. Observability & Error Handling (Milestone 12 Part A)

### 10.1 Structured Logging
- Backend must use structured logging (single logger implementation).
- Log level must be configurable (debug/info/warn/error).
- Logging format must support JSON for production.
- Console format may be used for development.
- Logs must never include:
  - passwords
  - JWTs
  - refresh tokens
  - API tokens
  - secrets

### 10.2 Request Correlation
- Every HTTP request must have a Request ID.
- If not provided, backend must generate one.
- Request ID must:
  - be accessible in request context
  - be included in response headers
  - be included in all logs for that request

### 10.3 Access Logging
- Backend must log per-request:
  - method
  - path
  - status code
  - duration
  - request_id
- Authorization headers and token values must not be logged.

### 10.4 Centralized Error Handling
- All errors must follow a consistent JSON shape:

  ```json
  {
    "error": {
      "code": "<CODE>",
      "message": "<MESSAGE>",
      "details": {}
    }
  }
  ```

---

## 11. Security & Operational Baseline (Milestone 12 Part B)

This section defines the minimum security and operational standards required
before the system is considered production-ready.

All items in this section must be fully implemented and tested
before Milestone 12 is marked complete.

---

### 11.1 Rate Limiting for Authentication

The backend must implement rate limiting for sensitive authentication endpoints:

- POST `/api/v1/auth/login`
- POST `/api/v1/auth/refresh`

Requirements:

- Rate limiting must be configurable via application configuration.
- Default behavior: disabled unless explicitly enabled.
- Rate limiting must support:
  - configurable request rate
  - configurable burst size
- Rate-limited responses must:
  - return HTTP 429
  - return error code `RATE_LIMITED`
- Rate limiting must not log secrets.
- Rate limiting must not degrade overall system stability.

---

### 11.2 CORS Support for Optional Web Frontend

The backend must support configurable Cross-Origin Resource Sharing (CORS)
to enable an optional web frontend.

Requirements:

- CORS must be disabled by default.
- Configuration must allow specifying:
  - allowed origins
  - allowed methods
  - allowed headers
  - allow credentials (boolean)
- When disabled, no CORS headers should be emitted.
- CORS configuration must not allow wildcard origins when credentials are enabled.

---

### 11.3 Configuration Validation

The backend must validate configuration at startup.

Requirements:

- Invalid configuration must prevent server startup.
- Validation must include:
  - server port
  - database DSN
  - JWT signing key (must not be empty outside test environments)
  - rate limit parameters (if enabled)
  - CORS origin formats (if enabled)
- On hot reload:
  - invalid configuration must be rejected
  - previously valid configuration must remain active
  - rejection must be logged without leaking secrets

---

### 11.4 Safe Defaults

The application must enforce secure defaults.

Requirements:

- Auto-migrate must default to `false` in production environments.
- JWT signing key must not be empty outside test environments.
- Security-sensitive features must default to secure configurations.
- Debug logging must not be enabled by default in production.

---

### 11.5 Health Endpoint (Non-sensitive)

The `/api/v1/health` endpoint must:

- Not expose secrets.
- Not expose environment variables.
- Not expose full configuration.
- May include:
  - service status
  - version
  - database connectivity status
  - optional uptime

Health endpoint must remain safe to expose to infrastructure monitoring systems.

---

### 11.6 Mandatory Test Coverage

Milestone 12 Part B is complete only when:

- Rate limiting behavior is tested.
- CORS enabled/disabled behavior is tested.
- Configuration validation is tested.
- Invalid hot reload behavior is tested.
- Health endpoint does not expose sensitive data.
- All backend tests pass.
- Existing frontend tests continue to pass.

---

# Phase S3 — Web Frontend (Optional but Supported)

Because Wails is **not** a web deployment target, the system must support an optional **web frontend** that consumes the **same backend API contract** used by the desktop app.

## 12. Web Frontend (Optional)

### 12.1 Goals
- Provide a browser-based UI for the same system features supported by the desktop app.
- Reuse the same backend endpoints, auth contract, RBAC rules, and error formats.
- Keep desktop and web implementations independent, while optionally sharing UI components/utilities.

### 12.2 Non-Goals
- The web frontend must **not** introduce a second backend.
- The web frontend must **not** bypass backend authorization checks (RBAC remains server-enforced).
- No requirement to implement offline/sync behavior for web unless explicitly added later.

### 12.3 Repository Layout
- Create a dedicated web app under `web/`:
  - `web/` (React + TypeScript)
  - `backend/` (existing Gin API)
  - optional `packages/` for shared UI/utilities (see 12.9)
- The web app must be buildable independently (no dependency on Wails build).

### 12.4 Web App Setup (Web Variant)
- Stack:
  - React + TypeScript
  - Material UI (same design language as desktop)
  - TanStack Router (file-based routing if preferred)
  - TanStack Query for data fetching/caching
- Environment configuration (at minimum):
  - `VITE_API_BASE_URL` (e.g., `http://localhost:8080/api/v1`)
  - `VITE_APP_NAME` (optional)
  - Optional: `VITE_ENABLE_DEVTOOLS`
- Provide:
  - `web/README.md` with local dev steps
  - `.env.example` in `web/` showing required variables

### 12.5 API Contract Compatibility
- Web must use the same endpoints, request/response shapes, and pagination conventions as desktop.
- Web must fully respect standardized error shape:
  ```json
  { "error": { "code": "<CODE>", "message": "<MESSAGE>", "details": {} } }
  ```

---

## 13. Upcoming Milestone — Shared Administration UX + Parity (Reusable Skeleton)

This milestone defines the next reusable-skeleton step for administration UX and cross-client consistency.
The scope is platform administration only and must remain domain-agnostic.

### 13.1 Administration Navigation Architecture
- Navigation must use grouped sections with `Administration` as the primary location for platform management.
- At minimum, the grouped structure must include:
  - `Dashboard`
  - `Administration`:
    - `Users`
    - `Roles`
    - `Permissions`
    - `Audit Log`
  - `Settings`
- Visibility of administration links must respect RBAC permissions.
- `Administration` must be an expandable/collapsible navigation group (not a static section label).
- If no administration child route is visible to the current principal, hide the entire `Administration` group.
- Desktop and web may differ in responsive presentation, but group semantics must remain aligned.

### 13.2 RBAC Administration UI
- Add reusable administration pages for:
  - roles management
  - permissions management
- Pages must use shared table/list and form patterns already used by users/audit administration.
- UI checks are convenience only; backend endpoints remain the authorization source of truth.

### 13.3 User Create/Update with Multi-Role Assignment
- User create/edit flows must support assigning multiple roles in one operation.
- Role assignments must be validated server-side.
- Invalid role identifiers must return typed validation errors and be surfaced in the UI.
- Role assignment behavior must be consistent between desktop and web clients.

### 13.4 Shared Admin Form/Dialog Patterns
- Administration pages should reuse shared patterns for:
  - create/edit dialogs (or drawers)
  - confirmation dialogs for destructive actions
  - validation error presentation
- User administration create/edit forms should use responsive multi-column layout:
  - 1 column on narrow/mobile overlays
  - 2 columns on medium widths
  - 3 columns on wide dialogs where space allows
- Avoid one-off page patterns when shared components can be extended.
- Keep all labels and copy generic so the skeleton can be reused across domains.

### 13.5 DataGrid Global UX Standards
- DataGrid behavior must follow a global baseline:
  - horizontal and vertical scrolling
  - server-side pagination/sort/filter where applicable
  - consistent density and spacing behavior
  - stable behavior for menus/dialogs in scroll containers
- Global UI preferences must be able to control DataGrid defaults (for example density, border radius, action-column pinning).
- Tables may extend defaults, but must not silently ignore global preferences.

### 13.6 Standard Actions Column
- Administration DataGrids must use a shared actions-column pattern.
- Actions column should be pinned right when supported; if unsupported, degrade gracefully without breaking usability.
- Shared table wrappers should default actions-column pinning when a standard `actions` field is present and pinning support is enabled.
- Actions-column pinning behavior should avoid excessive right-pinned width that causes usability/layout regressions.
- Action rendering should be centralized so row actions remain consistent across users, roles, permissions, and audit pages.

### 13.7 Audit Metadata Details UX
- Audit metadata cells may show a truncated preview in the grid.
- Full metadata must be viewable in a dedicated details dialog/drawer.
- Metadata viewer requirements:
  - pretty-printed JSON
  - scrollable container
  - copyable content
- Large JSON payloads must not be rendered inline in table cells.

### 13.8 Cross-Client Parity Rules
- Shared administration capabilities must be available in both desktop and web unless an explicit temporary parity gap is recorded in `docs/status.md`.
- Backend API updates for shared administration features must be consumed by both clients against the same API contract.
- Any intentional parity gap must include owner, scope, and follow-up milestone target in `docs/status.md`.

### 13.9 Required Tests for Milestone Completion
- Backend:
  - role/permission administration API tests
  - user multi-role assignment validation tests
  - authorization enforcement tests
- Desktop and web:
  - route-level smoke tests for new admin pages
  - permission-gated navigation tests
  - DataGrid actions column behavior tests
  - audit metadata details dialog tests
- Milestone completion still requires:
  - backend tests passing (`go test ./...`)
  - desktop/frontend route tests passing
  - web/frontend route tests passing
  - `docs/status.md` updated with completion evidence

### 13.10 Admin DataGrid Search Contract
- Administrative list endpoints and clients must use a consistent search/query contract:
  - `page` and `pageSize` for pagination
  - `sort=<field>:<asc|desc>` for server sorting
  - `filter=<field>:<value>` for DataGrid column filter passthrough
  - `q=<text>` for page-level quick search
- Backend should apply `q` server-side and preserve backward compatibility for legacy filter-only requests where practical.
- Desktop and web admin pages should debounce quick search input, compose `q` with pagination/sort/filter, and reset page to 1 when search text changes.

---

## 14. Upcoming Milestone — Shared Authentication UX, Branding, and Password Reset

This milestone defines reusable authentication-entry improvements for the BasePro skeleton and must remain domain-agnostic.

### 14.1 Polished Login Experience
- Improve login page UX for both desktop and web while keeping a shared interaction model.
- Login view must provide clear loading, validation, and authentication-failure states.
- Login copy and layout must remain reusable for future applications built on the skeleton.

### 14.2 Login Form Field Baseline
- Username/email and password inputs must use a minimum interactive height of `48px`.
- The minimum height baseline must remain consistent between desktop and web login views.
- Spacing, error-state visibility, and keyboard submit behavior must be consistent across clients.

### 14.3 Configurable Login Branding
- Login screen branding must be configurable from platform settings rather than hard-coded in client code.
- Branding configuration must support at minimum:
  - application display name
  - optional login subtitle/tagline
  - optional login hero/side image reference
- Branding values must be consumable by both desktop and web clients through the backend contract.

### 14.4 Login-Page App Image Support
- Login page must support an optional configurable image asset for product/app presentation.
- Image configuration must support graceful fallback when not configured or unavailable.
- Image rendering behavior should remain responsive and not block login form usability on small screens.

### 14.5 Configurable Application Display Name
- Application display name shown on authentication pages must come from shared branding settings.
- Display name updates must be reflected consistently across desktop and web authentication views.
- Display name configuration must be reusable across future projects without domain-specific language.

### 14.6 Settings Management for Login Branding
- Provide settings-page support for managing login branding values through authorized administration UX.
- Backend must validate branding setting payloads and return typed validation errors for invalid values.
- Any persisted branding settings must be served by backend APIs and consumed by both desktop and web clients.

### 14.7 Forgot-Password Flow
- Provide a forgot-password flow initiated from the login view.
- Flow must accept username/email input and return a non-enumerating response (same outward response regardless of account existence).
- Delivery mechanism details may remain configurable by deployment (for example email provider integration), but API contract and UX states must be defined in the skeleton.

### 14.8 Reset-Password Flow
- Provide a reset-password view that accepts reset token plus new password input.
- Reset flow must include password confirmation and server-side validation feedback.
- On successful password reset, previous active sessions for that account should be invalidated per backend security policy.

### 14.9 Password Reset Token Security Rules
- Reset tokens must be cryptographically random, single-use, and time-limited.
- Store only hashed token values server-side; never persist plaintext reset tokens.
- Tokens must be invalidated after successful use or when superseded by a newer reset request.
- Reset endpoints must enforce abuse protections (rate limiting/attempt controls) and must never leak whether an account exists.
- Logs must not include raw reset tokens, passwords, or other secret material.

### 14.10 Backend/Desktop/Web Parity Expectations
- Backend authentication contract changes for branding and password reset must be implemented before client consumption.
- Desktop and web must both implement:
  - polished login UI updates
  - shared branding display behavior
  - forgot-password and reset-password views and error handling
- Any temporary parity gap must be explicitly documented in `docs/status.md` with owner and follow-up milestone.

### 14.11 Required Tests for Milestone Completion
- Backend tests must cover:
  - forgot-password initiation behavior (including non-enumerating response pattern)
  - reset-password token validation, expiry, single-use enforcement, and invalidation behavior
  - branding settings validation and retrieval contract
  - authorization around settings update endpoints
- Desktop frontend tests must cover:
  - login UX states (loading, validation, failure)
  - forgot-password and reset-password route smoke tests
  - branding rendering on authentication views with fallback behavior
- Web frontend tests must cover:
  - login UX states (loading, validation, failure)
  - forgot-password and reset-password route smoke tests
  - branding rendering on authentication views with fallback behavior
- Completion gate remains unchanged:
  - backend tests passing (`go test ./...`)
  - desktop frontend route/smoke tests passing
  - web frontend route/smoke tests passing
  - `docs/status.md` updated with completion evidence

---

## 15. Upcoming Milestone — Registry-First Module Foundation

This milestone establishes a reusable architecture baseline for adding future modules without scattered one-off wiring.
The milestone is architecture and documentation first and must remain domain-agnostic.

### 15.1 Module Registry Concept
- Define a typed module registry as the canonical inventory of platform/application modules.
- Each module entry must declare at minimum:
  - module key/id
  - display label
  - base route/path
  - navigation group
  - default enabled/visibility intent
- Registry entries should be additive and predictable; avoid implicit behavior that depends on file location conventions alone.

### 15.2 Navigation Registry Concept
- Define a typed navigation registry derived from module definitions and platform shell requirements.
- Navigation registry must support grouped navigation semantics (for example `Dashboard`, `Administration`, `Settings`) and child ordering.
- Route organization should remain registry-aligned so module routes can be discovered and wired predictably from shared definitions instead of scattered one-off declarations.
- Navigation visibility remains RBAC-aware:
  - frontend visibility checks are UX-level
  - backend authorization is authoritative
- Existing `Administration` routes/pages should be the first concrete consumers of the navigation registry baseline.

### 15.3 Permission Registry Concept
- Define a typed permission registry as the canonical source for permission identifiers and optional module scope metadata.
- Permission registry must be reusable by:
  - backend authorization wiring
  - role/permission administration APIs
  - client-side permission-aware UI affordances
- Existing administration permissions should be the first concrete consumers of the permission registry baseline.

### 15.4 Registry-First Module Extension Workflow
- New module extension flow must be registry-first:
  1) add/update module registry entry
  2) add/update navigation registry mapping
  3) add/update route organization wiring from registry definitions
  4) add/update permission registry definitions
  5) add/update backend API contract (when module introduces data/actions)
  6) add/update desktop/web route wiring and UI consumption
  7) update milestone status documentation
- Do not introduce a dynamic plugin loader; keep registration static, typed, and maintainable.
- Avoid broad refactors in one step; prefer incremental migration of existing platform features to registry consumers.

### 15.5 Permission Naming Conventions
- Permission names must follow predictable module-based naming:
  - `<module>.read`
  - `<module>.write`
  - `<module>.delete`
  - `<module>.admin`
- Granular permissions are allowed when justified, but must stay consistent and documented (for example `inventory.adjust`).
- Do not mix naming styles for the same module.

### 15.6 Desktop/Web Parity Expectations for Modules
- If a module is defined as shared, route organization, navigation visibility intent, and major CRUD capabilities must be aligned across desktop and web.
- Temporary parity gaps are allowed only when explicitly documented in `docs/status.md` with scope and follow-up milestone.
- Backend API contracts for shared modules must be defined/updated before client consumption.

### 15.7 Required Tests for Milestone Completion
- For the registry-foundation milestone:
  - documentation updates are required in `docs/requirements.md`, `AGENTS.md`, and `docs/status.md`
  - prompt traceability copy must be saved under `docs/prompts/` and remain uncommitted
  - verify no application/runtime code changes are introduced
- For subsequent implementation milestones:
  - backend tests (`go test ./...`) must pass
  - desktop/frontend route/smoke tests must pass
  - web/frontend route/smoke tests must pass
  - targeted tests should cover registry wiring usage (navigation visibility and permission mapping behavior)
  - `docs/status.md` must record completion evidence and parity notes

---

## 16. Upcoming Milestone — Feature Flags / Module Enablement Contract

This milestone defines a typed, registry-first module enablement model so downstream projects can turn modules on/off without deleting code.
This milestone is documentation/design first and must not introduce domain modules.

### 16.1 Goals
- Support phased module rollout in reusable skeleton deployments.
- Keep enablement predictable across backend, desktop, and web.
- Prevent scattered ad-hoc feature checks.
- Keep behavior safe when modules are disabled.

### 16.2 Module Enablement Registry Concept
- Introduce a typed module-enablement registry/config that complements:
  - module registry
  - navigation registry
  - permission registry
- Each enablement entry should define at minimum:
  - module key/id
  - enabled default
  - explicit scope
  - rollout state (`always_on`, `default_on`, `default_off`, `experimental`)
- Keep registry configuration static and maintainable; do not introduce plugin loading.

### 16.3 Flag Scopes
Each module flag must declare scope explicitly:
- `backend` (API/service availability)
- `desktop` (desktop route/nav/page visibility)
- `web` (web route/nav/page visibility)
- `full_stack` (backend + desktop + web)

Flags may also declare behavior intent:
- navigation visibility
- route/page access
- API availability
- permission exposure

### 16.4 Source of Truth Expectations
- Source of truth must be explicitly documented and singular per environment.
- Preferred baseline:
  - typed static defaults in registry/config
  - backend computes and serves effective module-enablement config
  - desktop/web consume backend effective config for runtime gating
- Local client-only overrides are allowed only for development/testing and must not become production truth.
- Avoid conflicting sources of truth across clients.

### 16.5 Safe Disable Behavior
Disabling a module must be safe by default:
- hide module navigation entries
- block direct route access in desktop/web
- block or return typed disabled-module error for backend APIs when applicable
- prevent normal assignment/exposure of disabled-module permissions unless intentionally retained for compatibility
- avoid runtime crashes when a disabled module still has code present

Navigation hiding alone is insufficient.

### 16.6 Defaults and Rollout Strategy
- Every module must declare a documented default enablement state.
- Rollout states must be explicit and reviewable:
  - `always_on`: core platform capability
  - `default_on`: enabled unless explicitly disabled
  - `default_off`: present but disabled until enabled
  - `experimental`: disabled by default with explicit opt-in
- Changing defaults must be recorded in `docs/status.md` with migration/compatibility notes where needed.

### 16.7 Parity Expectations (Backend/Desktop/Web)
- For shared modules, enablement intent must remain aligned across backend, desktop, and web.
- If backend disables a module, clients must not present enabled UX that implies availability.
- Temporary parity gaps are allowed only when documented in `docs/status.md` with follow-up scope/owner.

### 16.8 Required Tests for Enablement Milestones
- Backend tests:
  - enabled/disabled API behavior
  - effective enablement config generation/serving
  - authorization behavior for disabled modules
- Desktop and web tests:
  - navigation visibility for enabled and disabled states
  - direct-route guard behavior for enabled and disabled states
  - permission-aware affordance behavior under enablement toggles
- Milestone completion still requires:
  - backend tests passing (`go test ./...`)
  - desktop/frontend route/smoke tests passing
  - web/frontend route/smoke tests passing
  - `docs/status.md` updated with parity notes and verification

---

## 17. Upcoming Milestone — Server-Driven Bootstrap Config + Settings Authorization Contract

This milestone defines a small, typed runtime bootstrap contract served by backend and consumed by desktop/web.
The goal is to provide consistent startup behavior and permission-safe Settings access without introducing a dynamic configuration platform.

### 17.1 Purpose of Server-Driven Bootstrap Config
- Provide a single backend endpoint that returns runtime UI/API context required at app startup.
- Ensure desktop and web derive effective startup behavior from backend contract rather than ad-hoc client defaults.
- Keep bootstrap contract additive, versioned, and typed so clients can safely evolve.

### 17.2 Representative Bootstrap Payload Structure
Bootstrap payload should remain compact and predictable. A representative shape:

```json
{
  "version": "1",
  "generatedAt": "2026-03-08T00:00:00Z",
  "modules": {
    "dashboard": { "enabled": true },
    "administration": { "enabled": true },
    "settings": { "enabled": true }
  },
  "navigation": {
    "groups": [
      { "key": "dashboard", "enabled": true },
      { "key": "administration", "enabled": true },
      { "key": "settings", "enabled": true }
    ]
  },
  "authz": {
    "settingsAccess": {
      "allowed": true,
      "requiresAny": ["admin", "settings.write"]
    }
  },
  "ui": {
    "theme": {
      "defaultMode": "system",
      "defaultPreset": "base"
    }
  }
}
```

This example is representative only; implementation may add fields but must keep typed compatibility and backward-safe evolution.

### 17.3 Relationship Between Registries and Bootstrap Runtime Config
- Registries remain the canonical static definitions for modules, navigation, permissions, and defaults.
- Bootstrap payload represents backend-resolved runtime state derived from registries plus runtime/config overrides.
- Clients should use registries for local typing/fallback and bootstrap payload for effective runtime behavior.
- Registry keys and bootstrap keys must stay aligned; avoid disconnected constants.

### 17.4 Offline-Aware Bootstrap Caching Behavior
- Desktop and web should cache the last valid bootstrap payload with timestamp metadata.
- On startup:
  - attempt fresh bootstrap fetch first
  - if fetch fails due to network/unreachable backend, use cached bootstrap when available
  - mark runtime state as degraded/offline-aware so UI can communicate stale config state
- Cache usage must be deterministic and safe; do not silently fabricate new bootstrap values when both network and cache are unavailable.

### 17.5 Bootstrap Cache Boundaries
- Bootstrap cache must include only non-secret runtime configuration needed for startup gating/visibility.
- Bootstrap cache must not include:
  - passwords
  - JWT access tokens
  - refresh tokens
  - API tokens
  - sensitive secret material
- Cache should be versioned and invalidated when payload schema version is incompatible.
- Bootstrap cache is a startup/runtime aid and must not replace backend authorization.

### 17.6 Settings Page Access Control
- Settings route/page visibility in clients must require either:
  - `admin` role
  - or `settings.write` permission
- Backend remains authoritative:
  - all protected Settings APIs must enforce authorization server-side
  - client-side checks are UX-level only and must not be treated as security controls
- Unauthorized Settings API access must return typed authorization errors.

### 17.7 Settings Scope and Security Expectations
- Settings surface is an administrative capability and must be treated as privileged.
- Separate read vs write behavior clearly:
  - `settings.read` for viewing administrative settings where applicable
  - `settings.write` for modifying settings
- Security-sensitive values should be masked/redacted in logs and responses where needed.
- Settings updates must be auditable with clear actor/action metadata.

### 17.8 Desktop/Web Parity Expectations
- Desktop and web must consume the same bootstrap API contract and apply equivalent Settings route protection logic.
- Differences in rendering/responsiveness are allowed, but runtime decisions (module visibility, Settings access intent) must align.
- Any temporary parity gap must be documented in `docs/status.md` with follow-up scope.

### 17.9 Required Tests for Milestone Completion
- Backend:
  - bootstrap endpoint contract/shape tests
  - bootstrap derivation tests from registry/default + override sources
  - Settings API authorization tests (`admin` or `settings.write` rules)
  - typed error tests for unauthorized access
- Desktop and web:
  - bootstrap fetch + startup consumption tests
  - offline cache fallback tests (fresh vs cached behavior)
  - Settings route-guard tests for authorized and unauthorized principals
  - navigation visibility tests consistent with bootstrap effective state
- Milestone completion still requires:
  - backend tests passing (`go test ./...`)
  - desktop/frontend route/smoke tests passing
  - web/frontend route/smoke tests passing
  - `docs/status.md` updated with verification and parity notes

---

## 18. Upcoming Milestone — Scheduler Vertical Slice

This milestone introduces the first Scheduler slice for SukumadPro and must be implemented vertically across backend, web, and desktop.

### 18.1 V1 Scope
- Support scheduled integration jobs.
- Support maintenance jobs.
- Store scheduled job definitions and scheduled job run history in the backend.
- Expose versioned API endpoints for create, update, list, get, enable/disable, run-now, and run-history retrieval.
- Add matching Scheduler navigation and placeholder-but-working management pages in both web and desktop clients.

### 18.2 V1 Non-Goals
- retries
- async polling
- delivery-window scheduling
- arbitrary scripts
- DAG dependencies

### 18.3 Scheduled Job Model
- Scheduled jobs must include:
  - identity and metadata (`id`, `uid`, `code`, `name`, `description`)
  - job classification (`job_category`, `job_type`)
  - schedule definition (`schedule_type`, `schedule_expr`, `timezone`)
  - execution controls (`enabled`, `allow_concurrent_runs`)
  - JSON configuration payload (`config`)
  - runtime summary fields (`last_run_at`, `next_run_at`, `last_success_at`, `last_failure_at`)
  - audit timestamps (`created_at`, `updated_at`)

### 18.4 Scheduled Job Run Model
- Scheduled job runs must include:
  - identity and linkage (`id`, `uid`, `scheduled_job_id`)
  - trigger metadata (`trigger_mode`, `scheduled_for`)
  - execution timestamps (`started_at`, `finished_at`)
  - lifecycle status (`pending`, `running`, `succeeded`, `failed`, `cancelled`, `skipped`)
  - optional worker linkage (`worker_id`)
  - failure and summary payloads (`error_message`, `result_summary`)
  - audit timestamps (`created_at`, `updated_at`)

### 18.5 Schedule Engine Rules
- Scheduler v1 must validate schedule definitions at write time.
- Supported schedule types:
  - `cron`
  - `interval`
- Next-run calculation must use the declared timezone and compute the next future due time only.
- Missed-run policy for v1:
  - do not replay all missed executions
  - compute only the next future due time from the current reference time

### 18.6 Authorization and Navigation
- Scheduler endpoints must integrate with platform authentication, RBAC, and audit logging.
- Scheduler navigation must be added to the existing Sukumad navigation group in both web and desktop clients.
- Scheduler route visibility must respect scheduler permissions client-side, while backend authorization remains authoritative.

### 18.7 Required Client Surfaces
- Both web and desktop must include:
  - Scheduled Jobs list
  - Create/Edit Scheduled Job form shell
  - Job Runs history shell
- These pages must load data from the shared Gin API and reuse existing admin shell/layout patterns.

### 18.8 Required Tests for Milestone Completion
- Backend tests must cover:
  - migration shape for scheduler tables
  - schedule validation and next-run calculation
  - repository and API route coverage for scheduler endpoints
- Web and desktop tests must cover:
  - scheduler route/page smoke coverage
  - API-backed scheduler list/form interactions at a basic level

# END (Authoritative)
