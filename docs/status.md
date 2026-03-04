# Status

## Milestone 1 — Repo Bootstrap + Baseline Build (Complete)

### What changed
- Bootstrapped repository structure for `backend/`, `desktop/` (generated via `wails init`), `web/`, and `docs/`.
- Added root project files: `.gitignore`, `Makefile`, and `docker-compose.yml`.
- Added prompt traceability folder `docs/prompts/` and stored the Milestone 1 prompt copy (kept ignored from git).
- Bootstrapped backend Go API with Gin under `/api/v1`.
- Implemented `GET /api/v1/health` returning `{ "status": "ok" }`.
- Implemented backend graceful shutdown with `signal.NotifyContext` + `http.Server.Shutdown` timeout.
- Added backend route test for `/api/v1/health`.
- Bootstrapped desktop app using `wails init` (React + TypeScript template).
- Added required frontend dependencies: MUI, MUI Icons, Emotion, TanStack Router, TanStack Query, and MUI X Data Grid.
- Implemented TanStack Router root/outlet + `/` route + configured NotFound route.
- Implemented minimal MUI centered card UI:
  - Title: `Skeleton App Ready`
  - Subtitle: `Wails + React + MUI + TanStack Router`
- Added frontend route tests:
  - `/` renders expected content
  - unknown route renders NotFound

### How to run
- Backend API: `make backend-run`
- Desktop app (dev): `make desktop-dev`

### How to test
- Backend tests: `make backend-test`
- Desktop route tests: `make desktop-test`

### Verification summary
- Backend builds: PASS (`go build ./...` in `backend/`)
- Backend tests pass: PASS (`go test ./...` in `backend/`)
- Desktop builds: PASS (`wails build -compiler /usr/local/go/bin/go -skipbindings` in `desktop/`)
- Desktop route tests pass: PASS (`npm test` in `desktop/frontend`)

### Known follow-ups
- `wails build` in this environment needs explicit Go toolchain env (`GOROOT=/usr/local/go`) and `-skipbindings` due local Wails binding-generation issues with the shell Go configuration and module scanning.

## Milestone 2 — Backend Foundation (Complete)

### What changed
- Added `backend/internal/config` with typed config struct and Viper loading using precedence: config file, environment variables, then CLI flag overrides.
- Enabled Viper hot reload via `WatchConfig` and `OnConfigChange`.
- Added atomic runtime snapshot (`atomic.Value`) with `config.Get()` so runtime code reads typed config without direct Viper access.
- Added reload validation logic: invalid config reloads are rejected and previous snapshot remains active.
- Added safe write helper (`SafeWriteFile`) that writes temp file then atomic rename.
- Added `backend/internal/db` with SQLX-based DB initialization, pool sizing, and startup ping.
- Added migration SQL files under `backend/migrations` for:
  - users
  - roles
  - permissions
  - role_permissions
  - user_roles
  - api_tokens
  - refresh_tokens
  - audit_logs
- Added migration runner commands in `backend/cmd/migrate` and shared helpers in `backend/internal/migrateutil`.
- Added Makefile migration targets:
  - `make migrate-up`
  - `make migrate-down`
  - `make migrate-create name=<migration_name>`
- Hardened API startup/shutdown:
  - `signal.NotifyContext` in `cmd/api/main.go`
  - shutdown timeout sourced from config
  - `http.Server.Shutdown(...)` on cancellation
  - clean DB close on process exit
  - startup server lifecycle helper with unit test coverage
- Updated API routes to include:
  - `GET /api/v1/health` returning `{ "status": "ok", "version": "...", "db": "up" }` when DB is healthy
  - `GET /api/v1/version` returning backend version
- Added tests:
  - config hot reload atomic swap + invalid reload retention test
  - health route test with DB ping mock
  - server graceful startup/shutdown lifecycle test
  - integration-style DB open test that skips when `BASEPRO_TEST_DSN` is not set

### How config reload works
- Config is loaded via Viper from `backend/config/config.yaml` (or `--config` path), env vars (`BASEPRO_*`), and CLI overrides.
- Hot reload watches config file changes.
- On each change, config is decoded + validated.
- If valid, new config is atomically swapped into `config.Get()`.
- If invalid, reload is logged and ignored; previous config remains active.

### How to run backend
- Ensure PostgreSQL is running (example: `docker compose up -d postgres`).
- From repo root run:
  - `make migrate-up`
  - `make backend-run`

### How to run migrations
- Apply all up migrations: `make migrate-up`
- Roll back one migration: `make migrate-down`
- Create migration pair: `make migrate-create name=<migration_name>`

### How to test
- Backend tests: `make backend-test`
- Frontend route/smoke tests: `make desktop-test`

### Verification summary
- Backend builds/tests (`go test ./...` in `backend/`): PASS
- Health route test: PASS
- Config reload test: PASS
- Frontend route tests: PASS

### Known follow-ups
- Auto-migration is available via `database.auto_migrate` config / `--database-auto-migrate`; keep disabled in production.
- Authentication/RBAC remains intentionally unimplemented for Milestone 2.

## Milestone 3 — Backend Auth (JWT + Refresh Rotation + Typed Errors + Audit) (Complete)

### What changed
- Added auth-specific migration `000009_auth_token_chain_and_audit_indexes` to extend existing tables without editing prior migrations.
- `refresh_tokens` now supports rotation/reuse chain tracking fields:
  - `issued_at`
  - `replaced_by_token_id`
  - `updated_at`
- `audit_logs` now supports `timestamp` for ordered event querying.
- Added required indexes:
  - `refresh_tokens(user_id)`
  - `refresh_tokens(token_hash)` unique index
  - `audit_logs(timestamp DESC)`
- Extended backend config with auth settings:
  - `auth.access_token_ttl_seconds`
  - `auth.refresh_token_ttl_seconds`
  - `auth.jwt_signing_key`
  - `auth.password_hash_cost`
- Added config validation to fail fast when JWT signing key is empty.
- Added `internal/apperror` for standardized typed error responses.
- Added `internal/audit` (SQLX repository + service).
- Added `internal/auth`:
  - SQLX repository for users/refresh tokens
  - password hashing/verify helpers (bcrypt)
  - JWT manager
  - auth service implementing login, refresh rotation, refresh reuse detection, logout, and `me`
  - HTTP handlers for `/api/v1/auth/*`
- Added `internal/middleware` JWT auth middleware with request context claims injection.
- Wired auth + audit dependencies in `cmd/api/main.go` via dependency injection (no global DB state).
- Added backend auth endpoints:
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/refresh`
  - `POST /api/v1/auth/logout`
  - `GET /api/v1/auth/me` (JWT protected)

### Typed error shape
- Handlers now return auth errors in standardized shape:
  - `{ "error": { "code": "...", "message": "..." } }`
- Implemented codes:
  - `AUTH_UNAUTHORIZED`
  - `AUTH_EXPIRED`
  - `AUTH_REFRESH_REUSED`
  - `AUTH_REFRESH_INVALID`

### Refresh rotation and reuse detection
- Refresh tokens are generated as random opaque strings and only SHA-256 hashes are stored in DB.
- On refresh success:
  - old token row is marked `revoked_at`
  - old token row links `replaced_by_token_id` to the new token row
  - new access + refresh tokens are issued
- On refresh reuse (revoked token presented):
  - response returns `AUTH_REFRESH_REUSED` (401)
  - active refresh tokens for that user are revoked to contain the token family/session.

### Auth endpoint examples (curl)
- Login:
  - `curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login -H 'Content-Type: application/json' -d '{"username":"alice","password":"secret"}'`
- Refresh:
  - `curl -s -X POST http://127.0.0.1:8080/api/v1/auth/refresh -H 'Content-Type: application/json' -d '{"refreshToken":"<refresh-token>"}'`
- Logout with refresh token:
  - `curl -s -X POST http://127.0.0.1:8080/api/v1/auth/logout -H 'Content-Type: application/json' -d '{"refreshToken":"<refresh-token>"}'`
- Me with access token:
  - `curl -s http://127.0.0.1:8080/api/v1/auth/me -H 'Authorization: Bearer <access-token>'`

### How to test
- Backend tests: `make backend-test`
- Frontend route/smoke tests: `make desktop-test`

### Verification summary
- Backend tests (`go test ./...`): PASS
- Login success test (token issuance + hashed refresh storage): PASS
- Login failure typed error test: PASS
- Refresh rotation test (old revoked + new works): PASS
- Refresh reuse detection + active token revocation test: PASS
- JWT middleware missing token -> `AUTH_UNAUTHORIZED`: PASS
- JWT middleware expired token -> `AUTH_EXPIRED`: PASS
- Frontend route tests: PASS

### Milestone scope guard
- API-token authentication was not implemented in this milestone.
- RBAC roles/permission enforcement was not implemented in this milestone.

## Milestone 4 — Backend API Token Auth (Machine/Integration Auth) (Complete)

### What changed
- Added migration `000010_api_token_management` to extend API-token storage and add token permission linkage without editing prior migrations.
- `api_tokens` now supports token management fields:
  - `prefix`
  - `created_by_user_id`
  - `revoked_at`
  - `expires_at` (already present)
  - `last_used_at`
  - `created_at` / `updated_at`
- Added `api_token_permissions` table for permission-scoped tokens (`permission` + optional `module_scope`).
- Added indexes:
  - unique `api_tokens(token_hash)`
  - `api_tokens(revoked_at)`
  - `api_tokens(prefix)`

### API token hashing approach
- API tokens are generated as random opaque plaintext strings (returned once at creation).
- Only a deterministic HMAC-SHA256 hash is stored in DB for lookup:
  - `stored_hash = HMAC_SHA256(token, auth.jwt_signing_key)`
- Plaintext token is never persisted and cannot be re-read from the API after create.

### Config additions
- Added config keys under `auth`:
  - `api_token_enabled` (bool)
  - `api_token_header_name` (default `X-API-Token`)
  - `api_token_ttl_seconds` (default TTL)
  - `api_token_allow_bearer` (allow Bearer token for API-token auth)
- Existing config validation now enforces non-empty token header name and positive token TTL.

### Middleware and principal model
- Added API-token middleware that accepts:
  - `X-API-Token: <token>`
  - optional `Authorization: Bearer <token>` when enabled
- Validation behavior:
  - token hash lookup
  - reject revoked tokens
  - reject expired tokens
  - update `last_used_at` best-effort
- Context principal now supports:
  - `type = user` (JWT)
  - `type = api_token` (API token)
  - `permissions` for API-token scoped checks
- Added `RequirePermission(...)` helper to enforce API token permissions (and keep JWT path ready for Milestone 5).

### Admin API-token endpoints
- Added JWT-protected admin endpoints under `/api/v1/admin/api-tokens`:
  - `GET /api/v1/admin/api-tokens`
  - `POST /api/v1/admin/api-tokens`
  - `POST /api/v1/admin/api-tokens/:id/revoke`
- Admin check is intentionally minimal for Milestone 4:
  - user ID `1` is treated as admin stub.
- Added audit events:
  - `api_token.create`
  - `api_token.revoke`

### Dev-only seed convenience
- Added startup flag `--seed-dev-admin` to create/ensure an initial dev admin user.
- Seed credentials can be set via env:
  - `BASEPRO_DEV_ADMIN_USERNAME`
  - `BASEPRO_DEV_ADMIN_PASSWORD`

### How to create/revoke API tokens
- Create token:
  - `curl -s -X POST http://127.0.0.1:8080/api/v1/admin/api-tokens -H 'Authorization: Bearer <jwt>' -H 'Content-Type: application/json' -d '{"name":"ci-token","permissions":["audit.read"],"expiresInSeconds":3600}'`
- List tokens (masked):
  - `curl -s http://127.0.0.1:8080/api/v1/admin/api-tokens -H 'Authorization: Bearer <jwt>'`
- Revoke token:
  - `curl -s -X POST http://127.0.0.1:8080/api/v1/admin/api-tokens/<id>/revoke -H 'Authorization: Bearer <jwt>'`

### How to test
- Backend tests: `make backend-test`
- Frontend route/smoke tests: `make desktop-test`

### Verification summary
- `go test ./...` in `backend/`: PASS
- API-token create endpoint test (plaintext once + hashed storage + prefix + permissions): PASS
- API-token middleware tests (valid/revoked/expired): PASS
- Audit tests for create/revoke actions: PASS
- Frontend route tests: PASS

### Milestone scope guard
- RBAC user role/permission enforcement was not implemented (reserved for Milestone 5).
- Desktop login/auth UI work was not started (reserved for Milestones 6/7).

## Milestone 5 — RBAC (Roles + Permissions + Module Scope) + Enforcement (Complete)

### What changed
- Implemented `backend/internal/rbac` with SQLX repository + service for role/permission resolution:
  - `GetUserRoles(userID)`
  - `GetUserPermissions(userID)`
  - `HasPermission(userID, perm, moduleScope)`
  - `PermissionsForUser(userID)`
- Added minimal in-process RBAC cache in service with explicit invalidation (`InvalidateUser`) on role assignment.
- Adopted **Option A** for module scope:
  - canonical permission key in `permissions.name` (e.g. `users.read`)
  - optional scope in `permissions.module_scope`
  - enforcement supports `RequirePermission("perm")` and `RequirePermission("perm", WithModule("scope"))`

### Principal and middleware changes
- Unified principal model for both auth types:
  - `principal.type = user | api_token`
  - user fields: user ID, username, roles, resolved permission grants
  - api token fields: token ID, permission grants (including optional module scope)
- Added/updated middleware helpers:
  - `RequireAuth()`
  - `RequireJWTUser()`
  - `RequirePermission(rbacService, "perm", optional WithModule(...))`
  - `ResolveJWTPrincipal(...)` so protected routes accept either JWT or API token principals.
- Added typed forbidden code:
  - `AUTH_FORBIDDEN` with response shape `{ "error": { "code": "AUTH_FORBIDDEN", "message": "Forbidden" } }`

### Endpoint enforcement applied
- Users management:
  - `GET /api/v1/users` -> `users.read`
  - `POST /api/v1/users` -> `users.write`
  - `PATCH /api/v1/users/:id` -> `users.write`
  - `POST /api/v1/users/:id/reset-password` -> `users.write`
- Audit:
  - `GET /api/v1/audit` -> `audit.read`
- API token admin:
  - `GET /api/v1/admin/api-tokens` -> `api_tokens.read`
  - `POST /api/v1/admin/api-tokens` -> `api_tokens.write`
  - `POST /api/v1/admin/api-tokens/:id/revoke` -> `api_tokens.write`
- Removed the Milestone 4 hardcoded admin check (`userID == 1`) from runtime authorization path.

### RBAC seed/bootstrap (dev/test only)
- Added config-controlled bootstrap flag:
  - `seed.enable_dev_bootstrap` (default: `false`)
- When enabled (or when `--seed-dev-admin` is passed), startup now:
  - ensures base roles: `Admin`, `Manager`, `Staff`, `Viewer`
  - ensures base permissions:
    - `users.read`, `users.write`
    - `audit.read`
    - `settings.read`, `settings.write`
    - `api_tokens.read`, `api_tokens.write`
  - ensures role-permission mapping:
    - `Admin`: all above permissions
    - `Manager`: `users.read`, `audit.read`, `settings.read`
    - `Staff`: `settings.read`
    - `Viewer`: `users.read`, `audit.read`, `settings.read`
  - ensures dev admin user exists and assigns `Admin` role.

### Additional backend handlers/services
- Added users service + handler (`backend/internal/users`) for list/create/update/reset-password paths.
- Extended audit package with list/query support and added audit handler (`GET /api/v1/audit` with pagination/filter params).

### How to test
- Backend tests:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Frontend route/smoke tests:
  - `make desktop-test`

### Verification summary
- Backend tests (`go test ./...` in `backend/`): PASS
- Frontend route tests (`make desktop-test`): PASS
- Mandatory Milestone 5 coverage included:
  - JWT permission enforcement (forbidden vs allowed)
  - role -> permission resolution
  - API token permission enforcement for audit access
  - correct unauthorized/forbidden error codes and shapes

### Milestone scope guard
- Desktop setup/login UI work was not started (still reserved for Milestones 6/7).

## Milestone 6 — Desktop Setup Screen + Routing + Route Tests (Complete)

### What changed
- Added desktop local settings persistence through Wails Go bindings in `desktop/app.go`:
  - `LoadSettings()`
  - `SaveSettings(partial)`
  - `ResetSettings()`
- Settings are now stored as JSON under OS user config dir:
  - `<os_user_config_dir>/basepro-desktop/settings.json`
  - directory created with best-effort `0700` permissions
  - file writes use best-effort `0600` permissions and temp-file rename
- Added frontend typed settings abstractions:
  - `src/settings/types.ts`
  - `src/settings/store.ts`
- Added frontend HTTP wrapper with timeout-aware health check:
  - `src/api/client.ts`
  - `healthCheck()` calls `GET {apiBaseUrl}/api/v1/health`
- Replaced route tree with Milestone 6 unauthenticated routing:
  - `/setup`
  - `/login` (placeholder: “Login not implemented yet”)
  - `/` gate route
  - NotFound component configured at root
- Implemented route gating behavior:
  - if `apiBaseUrl` is empty, navigating to `/` or `/login` redirects to `/setup`
- Implemented Setup page UI (`/setup`) with MUI centered card:
  - API Base URL input
  - Auth Mode selector (Username/Password vs API Token)
  - conditional API Token input
  - Request Timeout input
  - `Test Connection` and `Save & Continue` actions
- Added/updated route tests (`src/routes.test.tsx`) covering:
  - `/login` and `/` redirect to `/setup` when API base URL is missing
  - NotFound rendering for unknown routes
  - setup save flow navigates to `/login`
  - setup health check uses mocked `fetch` endpoint
- Added test setup shim for `Response` to keep TanStack Router redirect handling deterministic in tests.

### How to run desktop tests
- `make desktop-test`
- or `cd desktop/frontend && npm test`

### How to run full milestone verification
- Backend tests: `make backend-test`
- Desktop route tests: `make desktop-test`
- Desktop production build: `make desktop-build`

### Verification summary
- Backend tests (`make backend-test`): PASS
- Desktop route tests (`make desktop-test`): PASS (5 tests)
- Desktop build (`make desktop-build`): PASS

### Current desktop routes
- `/` (gate route)
- `/setup`
- `/login` (placeholder only)
- NotFound for unknown paths

### Milestone scope guard
- Login/refresh token UI flow was not implemented in this milestone.
- Authenticated AppShell (Drawer/AppBar/Footer) was not implemented in this milestone.

### Milestone 6 Bug Fix — Settings `authMode` Type Alignment (2026-02-28)
- Updated desktop settings type definitions to use a central `AuthMode` union source:
  - `AUTH_MODES = ['password', 'api_token'] as const`
  - `type AuthMode = (typeof AUTH_MODES)[number]`
- Hardened `src/settings/store.ts` normalization boundary to treat Wails binding payloads as untrusted input:
  - normalize from `unknown`
  - validate `authMode` against `AUTH_MODES`
  - default invalid/missing `authMode` to `password`
  - normalize API token and timeout values safely
- Removed the direct assumption that Wails `main.Settings` is already `AppSettings`, eliminating the `string` vs `AuthMode` mismatch at compile-time.

Bug-fix verification:
- `cd desktop/frontend && npm run build`: PASS
- `cd desktop/frontend && npm test -- --run`: PASS

## Backend Change — Configurable Startup Auto-Migrate with Advisory Lock (Complete)

### What changed
- Added a dedicated startup migration package: `backend/internal/migrate`.
- Startup migration behavior is now configuration-driven with three keys:
  - `database.auto_migrate` (bool, default `false`)
  - `database.auto_migrate_lock_timeout_seconds` (int, default `30`)
  - `database.migrations_path` (string, default `file://./migrations`)
- Backend startup (`cmd/api`) now runs migration wiring before serving HTTP:
  - If `database.auto_migrate=false`, migrations are skipped and startup continues.
  - If `database.auto_migrate=true`, startup attempts migrations via `migrate.Up()`.
  - `migrate.ErrNoChange` is treated as success.
  - Any migration error fails fast and prevents the server from starting.
- Added Postgres advisory-lock coordination in startup migration flow:
  - acquires advisory lock before migration run
  - uses context timeout from `database.auto_migrate_lock_timeout_seconds`
  - returns error if lock cannot be acquired in time
  - always attempts lock release after migration attempt
- Added startup migration logging for:
  - skipped vs running status
  - start and successful completion
  - no-change condition

### Example config (dev)
```yaml
database:
  auto_migrate: true
  auto_migrate_lock_timeout_seconds: 30
  migrations_path: "file://./migrations"
```

### Production safety note
- Default remains `database.auto_migrate=false`.
- Advisory lock behavior prevents concurrent migration execution across multiple backend instances when auto-migrate is enabled.

### Test coverage added
- `backend/internal/migrate/runner_test.go`:
  - `AutoMigrate=false` skips migrator execution.
  - `ErrNoChange` is treated as success.
  - advisory-lock acquisition timeout path returns expected error and does not run migrations.

### Verification summary
- Backend tests: `cd backend && GOCACHE=/tmp/go-build go test ./...` => PASS
- Frontend route tests: `make desktop-test` => PASS

### Backend Auto-Migrate Update — Embedded Migrations (No `database.migrations_path`)
- Removed `database.migrations_path` from backend runtime config and CLI overrides.
- Startup auto-migrate now infers migration source from Go-embedded SQL files (`go:embed`) via `golang-migrate` `iofs` source driver.
- Added embedded migrations package: `backend/migrations/embed.go`.
- `internal/migrate` runner now executes `Up()` from embedded migrations, so no filesystem migration path is required at runtime.
- Existing startup behavior remains:
  - `database.auto_migrate=false` (default): startup skips migrations.
  - `database.auto_migrate=true`: startup acquires advisory lock and runs migrations before serving.

Verification for this update:
- `cd backend && GOCACHE=/tmp/go-build go test ./...` => PASS
- `make desktop-test` => PASS

## Milestone 7 — Desktop Login + Refresh + Session Expiry UX (Complete)

### What changed
- Added desktop auth contracts in `desktop/frontend/src/auth/types.ts`:
  - `LoginRequest`, `LoginResponse`
  - `RefreshRequest`, `RefreshResponse`
  - `MeResponse`
- Added auth session store in `desktop/frontend/src/auth/session.ts` with required helpers:
  - `setSession({ accessToken, refreshToken, expiresAt })`
  - `clearSession()`
  - `isAuthenticated()`
- Implemented token storage model:
  - access token kept in memory only (session store state)
  - refresh token persisted via existing desktop settings storage (`settings.json`) and Wails binding patch.
- Extended desktop settings persistence schema with `refreshToken` in:
  - `desktop/app.go`
  - `desktop/frontend/src/settings/types.ts`
  - `desktop/frontend/src/settings/store.ts`
- Reworked API client (`desktop/frontend/src/api/client.ts`) to support authenticated requests:
  - attaches `Authorization: Bearer <accessToken>`
  - handles 401 by performing refresh once and retrying original request once
  - includes single-flight refresh lock so parallel requests wait on one refresh operation
  - on refresh failure with `AUTH_REFRESH_INVALID` / `AUTH_REFRESH_REUSED` / `AUTH_EXPIRED`, clears session and triggers forced re-login flow
  - emits separate network-unreachable event path
- Added global session UX wiring in `desktop/frontend/src/App.tsx`:
  - MUI Snackbar/Alert notifications
  - forced logout message: `Session expired. Please log in again.`
  - separate message for network unreachable
  - redirect to `/login` on forced logout
- Updated routing/guards (`desktop/frontend/src/routes.tsx`) for Milestone 7:
  - unauthenticated routes: `/setup`, `/login`
  - authenticated route: `/dashboard` (placeholder)
  - guards:
    - missing `apiBaseUrl` => redirect to `/setup`
    - unauthenticated access to `/dashboard` => redirect to `/login`
    - authenticated access to `/login` => redirect to `/dashboard`
  - NotFound route remains configured
- Added desktop pages:
  - Login UI implementation in `desktop/frontend/src/pages/LoginPage.tsx` (title/subtitle, username/password, loading state, generic invalid-credentials error, change API settings action)
  - Authenticated placeholder dashboard in `desktop/frontend/src/pages/DashboardPage.tsx`
- Added global notification store scaffold in `desktop/frontend/src/notifications/store.ts`.

### Refresh token storage location
- Refresh token is persisted through existing desktop settings storage:
  - file path: `<os_user_config_dir>/basepro-desktop/settings.json`
  - JSON key: `refreshToken`
  - write path remains temp-file + atomic rename with best-effort restrictive permissions.

### Auto-refresh behavior
- Access token is attached to authenticated API requests.
- If access token is expired or a request returns HTTP 401:
  - client performs one refresh request (`POST /api/v1/auth/refresh`) using persisted refresh token
  - while refresh is in-flight, other requests wait on the same promise (single-flight lock)
  - on success, stores new tokens and retries original request exactly once.

### Session-expired behavior
- If refresh fails with `AUTH_REFRESH_INVALID`, `AUTH_REFRESH_REUSED`, or `AUTH_EXPIRED`:
  - session is cleared (memory access token + persisted refresh token removed)
  - app redirects user to `/login`
  - global snackbar shows: `Session expired. Please log in again.`
- Network-unreachable errors show a separate non-auth snackbar message.

### Frontend tests added/updated
- `desktop/frontend/src/routes.test.tsx` now covers:
  - `/login` -> `/setup` when `apiBaseUrl` missing
  - `/dashboard` -> `/login` when not authenticated
  - `/login` -> `/dashboard` when authenticated
  - login success -> `/dashboard`
  - login failure -> generic invalid credentials error
  - 401 -> refresh success -> retried request success
  - refresh failure -> forced logout + redirect to `/login` + session-expired snackbar

### How to test
- Backend tests: `make backend-test`
- Desktop route/auth tests: `make desktop-test`
- Frontend build: `cd desktop/frontend && npm run build`
- Desktop build: `make desktop-build`

### Verification summary
- `make backend-test`: PASS
- `make desktop-test`: PASS (7 tests)
- `cd desktop/frontend && npm run build`: PASS
- `make desktop-build`: PASS

### Milestone scope guard
- AppShell (Drawer/AppBar/Footer) was not implemented in this milestone.
- Users and Audit pages were not implemented in this milestone.

## Milestone 8 — Sleek MUI Admin Shell + Themes + Palette Presets (Complete)

### What changed
- Added authenticated `AppShell` layout for `/dashboard` and `/settings` with:
  - desktop permanent Drawer with mini/collapsed mode
  - mobile temporary Drawer with hamburger toggle
  - top AppBar with route-based section title and user avatar/menu
  - main content outlet container and footer (`BasePro Desktop v0.1.0`)
- Implemented nav entries:
  - active routes: `Dashboard`, `Settings`
  - placeholders: `Users`, `Audit` shown disabled with “Soon” markers (no milestone 9/10 pages implemented)
- Added authenticated `/settings` page with sections:
  - Connection: editable API base URL + request timeout + `Test Connection` + save action
  - Auth mode display (read-only) with explicit redirect path to `/setup` for safe editing
  - Appearance: theme mode selector + preset quick buttons + full preset picker dialog entry point
  - About: app/build placeholders
- Added logout action in user menu:
  - best-effort `POST /api/v1/auth/logout` when refresh token exists
  - always clears local session and redirects to `/login`

### Theme + palette system
- Added persisted UI preferences (`uiPrefs`) in desktop settings model:
  - `themeMode`: `light | dark | system`
  - `palettePreset`: preset id
  - `navCollapsed`: drawer mini/collapse state
- Added reusable theme/palette modules:
  - `src/ui/theme.tsx` (`AppThemeProvider`, preference context, system mode handling)
  - `src/ui/palettePresets.ts` (8 preset definitions)
  - `src/ui/PalettePresetPicker.tsx` (dialog picker with instant preview)
- Theme and preset changes apply immediately and persist via settings store.

### UI preference storage location
- UI preferences are persisted in the existing desktop settings file:
  - `<os_user_config_dir>/basepro-desktop/settings.json`
  - JSON section: `uiPrefs`
- Backend/Wails settings schema extended in `desktop/app.go` with `UIPrefs`/`UIPrefsPatch` and validation/normalization defaults.

### Frontend tests added/updated
- `desktop/frontend/src/routes.test.tsx` now verifies:
  - authenticated `/dashboard` renders AppShell + dashboard content
  - Settings navigation works from shell
  - theme mode persistence and re-application after reload
  - palette preset persistence and re-application after reload
- Added deterministic `matchMedia` mock in `desktop/frontend/src/test-setup.ts`.

### How to test
- Backend tests: `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Frontend route/smoke tests: `make desktop-test`
- Frontend build: `cd desktop/frontend && npm run build`

### Verification summary
- `cd backend && GOCACHE=/tmp/go-build go test ./...`: PASS
- `make desktop-test`: PASS (4 tests)
- `cd desktop/frontend && npm run build`: PASS

### Milestone scope guard
- Users and Audit pages were not implemented (only disabled placeholders in shell nav).

## Milestone 9 — DataGrid Foundation (Advanced Features + Server Integration) (Complete)

### What changed
- Added reusable shared wrapper: `desktop/frontend/src/components/datagrid/AppDataGrid.tsx`.
- Confirmed MUI X Data Grid package is in use (`@mui/x-data-grid`) and implemented wrapper in a way that can accept Pro/Premium props later (including pinned columns passthrough).
- Wrapper capabilities implemented:
  - server-side pagination (`paginationMode="server"`)
  - server-side sorting (`sortingMode="server"`)
  - server-side filtering (`filterMode="server"`)
  - page size options (`10, 25, 50, 100`)
  - density selector via toolbar
  - column visibility model
  - column reordering persistence
  - pinned columns persistence + passthrough (active where supported)
  - CSV export via toolbar
- Added shared frontend pagination helpers:
  - `desktop/frontend/src/api/pagination.ts`
  - `desktop/frontend/src/api/useApiClient.ts`
- Added authenticated skeleton pages using wrapper:
  - `desktop/frontend/src/pages/UsersPage.tsx`
  - `desktop/frontend/src/pages/AuditPage.tsx`
- Wired routes for `/users` and `/audit` and enabled shell navigation entries for both pages.

### Persistent table preferences
- Extended desktop local settings schema (same Wails settings storage mechanism) to persist per-table preferences by `storageKey`:
  - `pageSize`
  - `columnVisibility`
  - `columnOrder`
  - `density`
  - `pinnedColumns`
- Added schema versioning in table preferences (`version: 1`) to support future schema evolution.
- Storage location remains the existing desktop settings file:
  - `<os_user_config_dir>/basepro-desktop/settings.json`
  - persisted under top-level key: `tablePrefs`.
- Updated both frontend and Wails model bindings for the extended settings shape.

### Server query contract (Milestone 9 contract)
- Adopted consistent query format for list endpoints:
  - `GET /api/v1/<resource>?page=1&pageSize=25&sort=<field>:<asc|desc>&filter=<field>:<value>`
- Implemented for:
  - `GET /api/v1/users`
  - `GET /api/v1/audit`
- Implemented consistent paginated response shape:
  - `{ "items": [...], "totalCount": number, "page": number, "pageSize": number }`

### Backend updates
- Updated users list backend to support:
  - pagination (`page`, `pageSize`)
  - sorting (`sort`)
  - filtering (`filter`, username)
  - response includes `totalCount`, `page`, `pageSize`
- Updated audit list backend to support:
  - pagination (`page`, `pageSize`)
  - sorting (`sort`)
  - filtering (`filter`, action)
  - response includes `totalCount`, `page`, `pageSize`
- Added backend pagination tests:
  - `backend/internal/users/users_test.go`
  - `backend/internal/audit/audit_test.go`

### Frontend tests added
- DataGrid wrapper tests:
  - calls `fetchData` on page change
  - calls `fetchData` on sort change
  - calls `fetchData` on filter change
- Persistence tests:
  - page size persists and reloads from settings store
  - column visibility persists and reloads from settings store
- Users page route test:
  - `/users` loads and renders rows from mocked API response

### How to test
- Backend tests:
  - `make backend-test`
- Frontend route/smoke + DataGrid tests:
  - `make desktop-test`
- Frontend build:
  - `cd desktop/frontend && npm run build`

### Verification summary
- `make backend-test`: PASS
- `make desktop-test`: PASS (7 tests)
- `cd desktop/frontend && npm run build`: PASS

### Milestone scope guard
- No business-specific modules (HR/payroll/etc.) were implemented.
- Milestone 10 work was not started.

## Milestone 10 — Users + Audit (End-to-End) + Permission-Gated Navigation (Complete)

### What changed
- Completed backend users endpoints with permission enforcement and validation:
  - `GET /api/v1/users` (`users.read`)
  - `POST /api/v1/users` (`users.write`)
  - `PATCH /api/v1/users/:id` (`users.write`)
  - `POST /api/v1/users/:id/reset-password` (`users.write`)
- Added/finished users backend behaviors:
  - input validation now uses typed validation errors (`VALIDATION_ERROR`)
  - patch now supports optional `username`, `isActive`, and `roles`
  - role updates are replacement-based (set semantics) instead of additive-only
  - password hashes remain server-only and are never returned in responses
- Added RBAC role replacement support:
  - `rbac.Repository.ReplaceUserRoles(...)`
  - `rbac.Service.SetUserRoles(...)`
- Ensured user-management audit writes are emitted:
  - `users.create`
  - `users.update`
  - `users.reset_password`
  - `users.set_active`
- Completed backend audit listing endpoint filters and contract:
  - `GET /api/v1/audit` (`audit.read`)
  - pagination/sort/filter response: `{ items, totalCount, page, pageSize }`
  - supports `action`, `actor_user_id`/`actorUserId`, and date range (`date_from`/`date_to` and camelCase aliases)
- Completed principal/capability behavior:
  - `/api/v1/auth/me` now returns `id`, `username`, `roles`, and `permissions`
  - desktop auth state stores principal permissions and exposes permission checks

### Permission and principal decisions
- `users.*` endpoints are JWT-user-only:
  - router now applies `RequireJWTUser` on `/api/v1/users` group in addition to `RequirePermission`.
  - API-token principals cannot administer users.
- `audit.read` endpoint supports either JWT or API-token principals:
  - `RequirePermission` is enforced for both principal types.

### Desktop completion
- Added permission-aware auth state:
  - session principal (`id`, `username`, `roles`, `permissions`) and subscription-based updates
  - login flow now loads `/api/v1/auth/me` immediately after token issuance
- Added permission-gated navigation and route guards:
  - Users nav shown only when `users.read` or `users.write`
  - Audit nav shown only when `audit.read`
  - direct navigation to `/users` or `/audit` without permission renders a new `ForbiddenPage` (`403`)
- Implemented full `/users` page with `AppDataGrid` and server data:
  - columns: username, roles, active, created
  - create user dialog
  - toggle active action
  - reset password dialog
  - edit roles dialog (multi-select)
  - write actions are disabled when `users.write` is missing
  - success/error toasts for all actions
- Implemented full `/audit` page with `AppDataGrid` and server data:
  - columns: timestamp, actor, action, entity_type, entity_id, metadata (compact)
  - filters: action dropdown, actor user id, date range

### Permission strings used
- `users.read`
- `users.write`
- `audit.read`

### Dev seeding for roles/permissions
- Backend dev bootstrap still seeds baseline RBAC and optional dev admin:
  - enable via existing config `seed.enable_dev_bootstrap` or CLI `--seed-dev-admin`
  - default baseline includes roles `Admin`, `Manager`, `Staff`, `Viewer`
  - baseline permissions include `users.read`, `users.write`, `audit.read`, and existing settings/api-token permissions
- Example startup for local dev:
  - `make migrate-up`
  - `cd backend && go run ./cmd/api --seed-dev-admin`

### Tests added/updated
- Backend:
  - permission enforcement tests for missing `users.write` and missing `audit.read`
    - `backend/internal/middleware/authorization_test.go`
  - pagination contract tests for users and audit list responses
    - `backend/internal/users/users_test.go`
    - `backend/internal/audit/audit_test.go`
  - audit write test for user creation (`users.create`)
    - `backend/internal/users/users_test.go`
- Desktop:
  - navigation gating test (Audit menu hidden without `audit.read`)
  - route guard test (`/audit` shows Forbidden without permission)
  - users action test (create user triggers API + grid refresh)
  - audit grid load test (rows rendered from mocked API)
  - file: `desktop/frontend/src/routes.test.tsx`

### How to test
- Backend tests:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Frontend tests:
  - `make desktop-test`
- Frontend build:
  - `cd desktop/frontend && npm run build`

### Verification summary
- `cd backend && GOCACHE=/tmp/go-build go test ./...`: PASS
- `make desktop-test`: PASS (6 tests)
- `cd desktop/frontend && npm run build`: PASS

### Milestone scope guard
- No HR/payroll/business-domain modules were started.
- Milestone remained skeleton-focused (users + audit + permission-gated shell behavior).

## Milestone 11 — Packaging + Versioning + CI Baseline (Complete)

### What changed
- Added backend build metadata variables in `cmd/api/main.go`:
  - `Version` (default `dev`)
  - `Commit` (default `none`)
  - `BuildDate` (default `unknown`)
- Updated `/api/v1/version` to return:
  - `version`
  - `commit`
  - `buildDate`
- Extended router dependency wiring to pass version metadata from build/runtime into HTTP responses.
- Added backend test coverage for version endpoint metadata contract.
- Updated desktop Settings > About section to show:
  - desktop version (from Vite define)
  - backend version from `/api/v1/version`
  - backend commit/build date when available
  - `Not Connected` fallback when backend is unreachable.
- Added desktop API client helper for unauthenticated `/api/v1/version` fetch.
- Added desktop test coverage asserting About section renders mocked backend version.
- Hardened root `Makefile` and added release/CI targets:
  - `backend-build` (with ldflags version injection)
  - `backend-test`
  - `backend-run`
  - `desktop-build`
  - `desktop-dev`
  - `desktop-test`
  - `ci` (runs `backend-test`, `desktop-test`, `desktop-build`)
- Added GitHub Actions workflow: `.github/workflows/ci.yml`
  - runs on push + pull_request
  - installs Go and Node
  - runs backend tests, desktop install/test, and desktop build
  - fails on any non-zero step.

### How version injection works
- `Makefile` computes build metadata values:
  - `VERSION` (default `dev`, overrideable)
  - `COMMIT` from `git rev-parse --short HEAD`
  - `BUILD_DATE` in UTC RFC3339-like format
- `backend-build` injects metadata with:
  - `-X main.Version=$(VERSION)`
  - `-X main.Commit=$(COMMIT)`
  - `-X main.BuildDate=$(BUILD_DATE)`
- Backend binary output path:
  - `backend/bin/basepro-api`

### How to build backend with version info
- Default build metadata:
  - `make backend-build`
- Explicit version override:
  - `make backend-build VERSION=1.0.0`

### How CI works
- Workflow file: `.github/workflows/ci.yml`
- Trigger events:
  - `push`
  - `pull_request`
- Steps:
  - `go test ./...` in `backend/`
  - `npm install`, `npm test`, and `npm run build` in `desktop/frontend/`

### How to run milestone verification
- Full local CI baseline:
  - `make ci`
- Backend version-injected build:
  - `make backend-build`

### Verification summary
- `make backend-build`: PASS
- `make ci`: PASS
  - backend tests: PASS
  - desktop tests: PASS
  - desktop build: PASS

### Known follow-ups
- Frontend build currently emits existing third-party bundling warnings (`"use client"` directives and chunk-size warnings) but build completes successfully.

## Milestone 12 Part A — Observability & Error Handling (Complete)

### What changed
- Added a centralized backend structured logger (`backend/internal/logging`) built on `slog` and replaced backend `log.Printf` usage in API startup/config reload/migration paths.
- Added backend config keys for logging:
  - `logging.level` (`debug|info|warn|error`)
  - `logging.format` (`json|console`)
- Wired logging config updates through existing Viper hot reload using a config change callback, so runtime logger settings are updated when validated config changes.
- Added request-correlation middleware:
  - `backend/internal/middleware/requestid.go`
  - every request now has `X-Request-Id` (preserved if provided, generated if missing)
  - request ID is stored in Gin + request context for downstream logging.
- Added access logging middleware:
  - `backend/internal/middleware/accesslog.go`
  - logs `method`, `path`, `status`, `duration_ms`, `request_id` per request.
- Updated router middleware chain to include request ID and access logging for all routes.
- Upgraded centralized error handling (`backend/internal/apperror/errors.go`) to enforce consistent shape:
  - `{ "error": { "code", "message", "details" } }`
  - typed auth errors preserve existing error codes and now include `details: {}`
  - validation errors return `VALIDATION_ERROR` with populated details
  - internal errors return safe message + `INTERNAL_ERROR` with empty details.
- Added structured error log entries in `apperror.Write(...)` that include `request_id`, `code`, and status while avoiding secret-bearing payload/header logging.

### How to run tests
- Backend tests:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Desktop tests:
  - `make desktop-test`

### Verification summary
- `cd backend && GOCACHE=/tmp/go-build go test ./...`: PASS
- `make desktop-test`: PASS
- Added/updated test coverage for Milestone 12 Part A:
  - request ID generation/preservation + response header behavior
  - access log fields including request correlation
  - centralized error JSON shape for auth, validation, and internal errors.

## Milestone 12 Part B — Security & Operational Baseline (Complete)

### What changed
- Added backend security config sections and defaults:
  - `security.rate_limit.enabled`
  - `security.rate_limit.requests_per_second`
  - `security.rate_limit.burst`
  - `security.cors.enabled`
  - `security.cors.allowed_origins`
  - `security.cors.allowed_methods`
  - `security.cors.allowed_headers`
  - `security.cors.allow_credentials`
- Implemented authentication endpoint rate limiting middleware with bounded in-memory per-client token buckets and stale-entry cleanup:
  - applied only to `POST /api/v1/auth/login` and `POST /api/v1/auth/refresh`
  - rate-limited responses return HTTP `429` with standardized error code `RATE_LIMITED`.
- Implemented configurable CORS middleware with conditional router application:
  - CORS disabled by default (no CORS headers emitted)
  - configurable origins/methods/headers/credentials
  - preflight handling with explicit policy enforcement.
- Extended config validation for security baseline rules:
  - rate limit parameters must be valid when enabled
  - CORS must have non-empty policy when enabled
  - wildcard origins are rejected when `allow_credentials=true`
  - invalid startup config fails load
  - invalid hot reload is rejected and previous valid snapshot remains active.
- Preserved secure defaults and existing protections:
  - `database.auto_migrate` default remains `false`
  - JWT signing key remains required by config validation
  - default logging level remains `info` (debug not default).
- Added health endpoint safety test coverage to ensure response does not leak sensitive terms/fields.

### How to run tests
- Backend tests:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Desktop tests:
  - `make desktop-test`

### Verification summary
- `cd backend && GOCACHE=/tmp/go-build go test ./...`: PASS
- `make desktop-test`: PASS
- Added/updated backend test coverage for:
  - rate limiting enabled/disabled behavior and `429/RATE_LIMITED` response shape
  - CORS enabled/disabled behavior
  - wildcard origin + credentials validation failure
  - invalid startup config failure
  - invalid hot reload rejection with prior config retention
  - health endpoint non-sensitive response assertions.

## Milestone 13 — Phase S3 Step 1 Web Frontend Scaffold & Baseline (Complete)

### What changed
- Scaffolded `web/` as a standalone Vite + React + TypeScript app (incremental extension of existing placeholder `web/.gitkeep`).
- Added baseline dependencies for:
  - Material UI
  - TanStack Router
  - TanStack Query
- Added environment variable support and examples:
  - `VITE_API_BASE_URL` (required)
  - `VITE_APP_NAME` (optional)
  - `VITE_ENABLE_DEVTOOLS` (optional placeholder for future use)
- Added `web/.env.example`.
- Added `web/README.md` with setup/dev/build/test instructions.
- Implemented web routes and baseline pages:
  - `/` redirects to `/login`
  - `/login` placeholder page
  - `/dashboard` placeholder page
  - NotFound page for unknown routes
  - global route error boundary component
- Added minimal route smoke tests in `web/src/routes.test.tsx`.
- Added root Makefile targets for web:
  - `make web-dev`
  - `make web-build`
  - `make web-test`
- Added prompt traceability copy:
  - `docs/prompts/2026-03-03-phase-s3-web-step1.md`
- Updated `.gitignore` to include web artifacts:
  - `web/node_modules/`
  - `web/dist/`
  - `web/.env`

### How to run
- Install deps: `cd web && npm install`
- Start dev server: `npm run dev`
- Build: `npm run build`
- Test: `npm run test`

### How to test
- Backend tests: `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Desktop route tests: `cd desktop/frontend && npm test`
- Web route tests: `cd web && npm test`

### Verification summary
- `cd backend && GOCACHE=/tmp/go-build go test ./...`: PASS
- `cd desktop/frontend && npm test`: PASS
- `cd web && npm run test`: PASS (3 tests)
- `cd web && npm run build`: PASS
- `cd web && npm run dev -- --host 127.0.0.1 --port 4173 --strictPort`: PASS (server started; `Local: http://127.0.0.1:4173/`)

### Known follow-ups
- Full web auth flow and RBAC gating are intentionally deferred to later S3 steps.

## Phase S3 Web — API Client Complete

### What changed
- Added typed web API client at `web/src/lib/api.ts`:
  - `apiRequest<T>()` wrapper using `import.meta.env.VITE_API_BASE_URL`
  - optional auth provider wiring via `configureApiClient(...)` and automatic `Authorization: Bearer <accessToken>` attachment
  - standardized non-2xx parsing to:
    - `type ApiError = { code: string; message: string; details?: unknown; requestId?: string }`
  - `X-Request-Id` extraction from error responses
  - sanitized request logging path so `Authorization` is redacted (`[REDACTED]`)
- Added user-facing error utility at `web/src/lib/errors.ts`:
  - `toUserFriendlyError(error)`
  - appends `Request ID: <id>` when present
- Added global MUI snackbar system at `web/src/ui/snackbar.tsx` and wired it in `web/src/main.tsx`.
- Added web API-client tests at `web/src/lib/api.test.ts` covering:
  - standardized error JSON parsing
  - `X-Request-Id` extraction
  - Authorization redaction in logger metadata
  - success response behavior
- Saved milestone prompt copy under `docs/prompts/2026-03-03-phase-s3-web-api-client.md` (ignored from git).

### How to run
- Backend tests: `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Desktop tests: `cd desktop/frontend && npm test -- --run`
- Desktop build: `cd desktop/frontend && npm run build`
- Web tests: `cd web && npm test -- --run`
- Web build: `cd web && npm run build`

### Verification summary
- Backend tests (`go test ./...`): PASS
- Web tests (`npm test -- --run`): PASS
- Web build (`npm run build`): PASS
- Desktop build (`npm run build`): PASS
- Desktop tests (`npm test -- --run`): FAIL in current workspace due existing desktop test issues not introduced by this web change set (route assertions and `structuredClone` availability in current test runtime).

### Known follow-ups
- Desktop test suite needs a separate stabilization pass in this workspace before milestone-wide "all frontend tests pass" can be asserted.

## Phase S3 Web — Auth Complete

### What changed
- Added `AuthProvider` in `web/src/auth/AuthProvider.tsx` with auth state and methods:
  - `isAuthenticated`
  - `accessToken` (memory)
  - `user`
  - `login(username, password)`
  - `logout()`
  - `refresh()`
- Added auth snapshot/session storage helpers in `web/src/auth/state.ts`:
  - refresh token persistence in local storage
  - in-memory auth snapshot for route guards
- Extended API client (`web/src/lib/api.ts`) to support:
  - configurable unauthorized handler via `configureApiClient(...)`
  - automatic one-time retry after 401 when refresh succeeds
  - per-request options to disable auth injection/retry for login/refresh/logout calls
- Updated web `LoginPage`:
  - real username/password submit flow using `auth.login`
  - success navigation to `/dashboard`
  - backend `ApiError` message display
  - Request ID display when present
- Added route guards in `web/src/routes.tsx`:
  - `/dashboard` requires authentication
  - `/login` redirects to `/dashboard` when authenticated
  - `/` redirects by auth state (`/dashboard` or `/login`)
- Added/updated auth route tests in `web/src/routes.test.tsx`:
  - login success redirects
  - login failure shows backend error + request ID
  - refresh failure logs out, redirects to `/login`, and shows session-expired message
  - protected route blocked while logged out
- Saved prompt traceability copy under `docs/prompts/2026-03-03-phase-s3-web-auth.md` (ignored by git).

### How to run tests
- Backend: `make backend-test`
- Desktop: `make desktop-test`
- Web tests: `make web-test`
- Web build: `make web-build`

### Verification summary
- `make backend-test`: PASS
- `make desktop-test`: PASS
- `make web-test`: PASS
- `make web-build`: PASS

### Known follow-ups
- Vite build emits non-blocking warnings from third-party dependencies about ignored `'use client'` directives.

## Phase S3 Web — RBAC + Navigation Complete

### What changed
- Added RBAC helper utilities in `web/src/rbac/permissions.ts`:
  - `hasRole(role)`
  - `hasPermission(permission)`
  - both read claims from authenticated user snapshot managed by `AuthProvider`.
- Added authenticated shell layout at `web/src/components/AppShell.tsx`:
  - Drawer navigation and top app bar for authenticated routes
  - role/permission-based visibility for module links
  - hidden restricted nav modules
  - logout action preserved through existing auth provider logout flow.
- Updated route structure in `web/src/routes.tsx`:
  - authenticated parent route now renders `AppShell`
  - added module routes for `/employees`, `/leave`, `/payroll`, `/users`, `/settings`
  - unauthorized route access now renders `Not Authorized` page.
- Added module placeholder pages with `title + Coming soon`:
  - `EmployeesPage`
  - `LeavePage`
  - `PayrollPage`
  - `UsersPage`
  - `SettingsPage`
- Added reusable module placeholder renderer at `web/src/pages/modules/ModulePlaceholderPage.tsx`.
- Added `web/src/pages/NotAuthorizedPage.tsx`.
- Updated `DashboardPage` module action buttons so restricted actions are visible but disabled.
- Added RBAC navigation tests in `web/src/routes.test.tsx`:
  - role-based module visibility (Admin sees Payroll)
  - role-based module hiding (Staff does not see Payroll)
  - unauthorized route navigation handling (`/users` without permission shows `Not Authorized`).
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-03-phase-s3-web-rbac-navigation.md` (gitignored)

### How to run tests
- Backend: `make backend-test`
- Desktop: `make desktop-test`
- Web tests: `make web-test`
- Web build: `make web-build`

### Verification summary
- `make backend-test`: PASS
- `make desktop-test`: PASS
- `make web-test`: PASS
- `make web-build`: PASS

### Known follow-ups
- `web` production build still emits non-blocking third-party `'use client'` and chunk-size warnings from existing dependency bundles.

## Web — Theme System + Presets (Foundation) Complete

### What changed
- Added local UI preferences foundation in `web/src/ui/preferences.ts` with APIs:
  - `loadPrefs()`
  - `savePrefs()`
  - `setMode()`
  - `setPreset()`
- Persisted UI preferences in localStorage key `basepro.web.ui_preferences`.
- Added theme preset catalog in `web/src/ui/theme/presets.ts`:
  - 8 admin-style presets with `id`, `name`, and light/dark palette tokens.
  - helper `getPaletteOptions(presetId, mode)` to resolve MUI palette options by preset.
- Added `UiPreferencesProvider` in `web/src/ui/theme/UiPreferencesProvider.tsx`:
  - tracks `mode` (`light | dark | system`) and `preset`
  - resolves `system` mode with `prefers-color-scheme`
  - applies updates immediately and persists locally.
- Added app-level theme wiring in `web/src/ui/theme/AppThemeProvider.tsx`:
  - computes MUI theme from resolved mode + preset
  - applies `ThemeProvider` + `CssBaseline`
  - includes token-based component style overrides.
- Updated `web/src/App.tsx` to use `AppThemeProvider` while keeping existing auth flow and API client behavior intact.
- Added deterministic theme tests in `web/src/ui/theme/theme.test.tsx` covering:
  - mode persistence and apply-after-reload
  - preset persistence and apply-after-reload
  - deterministic system mode behavior with mocked `matchMedia`.
- Updated `web/src/test-setup.ts` with a baseline `matchMedia` stub for deterministic test runtime.

### How to run tests
- `cd web && npm run test`

### Verification summary
- `cd web && npm run test`: PASS
- `cd web && npm run build`: PASS

### Known follow-ups
- Existing non-blocking Vite bundle warnings from third-party dependencies (`'use client'` directives, chunk-size warning) remain unchanged.

## Web — Settings Page (Theme + Presets + Connection) Complete

### What changed
- Implemented full `web` settings experience in `web/src/pages/SettingsPage.tsx` with sections:
  - Appearance:
    - theme mode selector (`light | dark | system`)
    - palette preset picker with mini visual previews for all presets
    - immediate apply via existing UI preferences provider
    - preview block with common components
  - Navigation:
    - `Collapse side navigation by default` toggle persisted in UI preferences
  - Connection:
    - editable API base URL override saved locally
    - `Test Connection` action calling `GET /health` via API client
    - snackbar success/failure feedback
    - failure message includes request id when backend returns `X-Request-Id`
  - About:
    - app name and version/build placeholders
- Extended UI preferences store in `web/src/ui/preferences.ts`:
  - added `collapseNavByDefault` persisted with existing `mode` and `preset`
  - added `setCollapseNavByDefault(...)`
- Extended UI preferences provider in `web/src/ui/theme/UiPreferencesProvider.tsx`:
  - added `setCollapseNavByDefault(...)` context action
- Added local API base URL override module `web/src/lib/apiBaseUrl.ts`:
  - local storage read/write for override
  - `resolveApiBaseUrl()` helper
- Updated API client base URL resolution in `web/src/lib/api.ts`:
  - `baseURL = override || env default`
- Updated app shell behavior in `web/src/components/AppShell.tsx`:
  - mini/collapsed drawer state now initializes from persisted `collapseNavByDefault`
- Added/updated deterministic tests in `web/src/routes.test.tsx` and `web/src/ui/theme/theme.test.tsx` covering:
  - `/settings` render and control-driven preference updates
  - mode persistence across reload
  - preset persistence across reload
  - API base URL override persistence

### Storage locations
- UI preferences: `localStorage['basepro.web.ui_preferences']`
- API base URL override: `localStorage['basepro.web.api_base_url_override']`

### How to run tests
- `cd web && npm run test`

### Verification summary
- `cd web && npm run test`: PASS
- `cd web && npm run build`: PASS

### Known follow-ups
- Non-blocking Vite warnings from third-party bundles (`'use client'` directives and chunk-size warning) remain unchanged.
