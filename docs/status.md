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
