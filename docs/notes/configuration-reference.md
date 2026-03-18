# Configuration Reference

This note documents the configuration surfaces currently present in the repository as of 2026-03-17.

## Overview

The project currently has four main configuration layers:

| Layer | Where it lives | Used by | Hot reload | Notes |
| --- | --- | --- | --- | --- |
| Backend runtime config | `backend/config/config.yaml` plus `BASEPRO_*` env vars and selected CLI flags | `backend/cmd/api`, `backend/cmd/worker`, `backend/cmd/migrate`, `backend/cmd/seed-sukumad-demo` | Yes for `api` and `worker` when loaded with `Watch: true` | Viper-backed, validated, stored as an atomic snapshot |
| Desktop persisted settings | OS user config dir: `basepro-desktop/settings.json` | Wails backend and desktop React frontend | No | Saved atomically by the desktop app |
| Web local preferences | Browser `localStorage` | React web app | No | Used for theme, API URL override, auth refresh token, bootstrap cache, and data grid prefs |
| Build/tooling config | `desktop/wails.json`, Vite env vars, Make targets | Build and dev workflows | No | Not part of business runtime behavior, but still relevant operationally |

## Backend Config Loading

Backend config is loaded by `backend/internal/config`.

### Resolution order

From lowest to highest precedence:

1. In-code defaults
2. `backend/config/config.yaml` or `./config.yaml`
3. Environment variables with `BASEPRO_` prefix
4. Explicit CLI overrides from `backend/cmd/api`

Examples:

- `database.dsn` can be overridden by `BASEPRO_DATABASE_DSN`
- `auth.jwt_signing_key` can be overridden by `BASEPRO_AUTH_JWT_SIGNING_KEY`
- `server.port` can be overridden by `--server-port`

### Config file lookup

| Entry point | Default lookup | Explicit file flag |
| --- | --- | --- |
| API server | `./config/config.yaml`, then `./config.yaml` | `--config` |
| Worker | `./config/config.yaml`, then `./config.yaml` | `--config` |
| Migrate | `./config/config.yaml`, then `./config.yaml` | `-config` |
| Seed demo | `./config/config.yaml`, then `./config.yaml` | `--config` |

### Backend environment variables in active use

| Variable | Default | Used by | Purpose |
| --- | --- | --- | --- |
| `BASEPRO_*` | per key | backend config loader | Generic Viper override for YAML keys |
| `BASEPRO_ENV` | empty | bootstrap metadata | Exposes current environment name in bootstrap payload |
| `BASEPRO_DEV_ADMIN_USERNAME` | `admin` | API server when `--seed-dev-admin` or `seed.enable_dev_bootstrap=true` | Dev admin seed username |
| `BASEPRO_DEV_ADMIN_PASSWORD` | `admin123!` | API server when seeding dev admin | Dev admin seed password |
| `BASEPRO_TEST_DSN` | none | backend DB tests | Integration test DSN |

## Backend YAML Keys

Defaults come from `backend/internal/config/config.go`. The sample file `backend/config/config.yaml` currently mirrors almost all defaults and also supplies development values for required secrets.

### Logging

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `logging.level` | string | `info` | Must be one of `debug`, `info`, `warn`, `error` |
| `logging.format` | string | `console` | Must be `json` or `console` |

### Server

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `server.port` | string | `:8080` | HTTP listen address |
| `server.shutdown_timeout_seconds` | int | `10` | Must be `> 0` |
| `server.cors_allowed_origins` | `[]string` | `["http://localhost:5173","http://127.0.0.1:5173","wails://wails.localhost","wails://wails.localhost:*"]` | Legacy/default origin list; values must be non-empty |

### Database

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `database.dsn` | string | none in code; sample file sets `postgres://postgres:postgres@127.0.0.1:5432/sukumadpro?sslmode=disable` | Required |
| `database.max_open_conns` | int | `10` | Must be `> 0` |
| `database.max_idle_conns` | int | `5` | Must be `> 0` |
| `database.auto_migrate` | bool | `false` | Runs migrations at startup for API and worker |
| `database.auto_migrate_lock_timeout_seconds` | int | `30` | Must be `> 0` |

### Auth

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `auth.access_token_ttl_seconds` | int | `900` | 15 minutes |
| `auth.refresh_token_ttl_seconds` | int | `604800` | 7 days |
| `auth.jwt_signing_key` | string | none in code; sample file sets `dev-insecure-signing-key` | Required |
| `auth.password_hash_cost` | int | `12` | Must be between `4` and `31` |
| `auth.api_token_enabled` | bool | `true` | Enables API token auth path |
| `auth.api_token_header_name` | string | `X-API-Token` | Required non-empty |
| `auth.api_token_ttl_seconds` | int | `2592000` | 30 days |
| `auth.api_token_allow_bearer` | bool | `false` | If `true`, bearer token auth may accept API tokens |

### Security

#### `security.rate_limit`

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `security.rate_limit.enabled` | bool | `false` | Enables auth endpoint rate limiting |
| `security.rate_limit.requests_per_second` | float | `5` | Must be `> 0` when enabled |
| `security.rate_limit.burst` | int | `10` | Must be `> 0` when enabled |

#### `security.cors`

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `security.cors.enabled` | bool | `false` | Sample file currently sets this to `true` |
| `security.cors.allowed_origins` | `[]string` | `[]` | Required non-empty when enabled |
| `security.cors.allowed_methods` | `[]string` | `["GET","POST","PUT","PATCH","DELETE","OPTIONS"]` | Required non-empty when enabled |
| `security.cors.allowed_headers` | `[]string` | `["Authorization","Content-Type","X-API-Token","X-Request-Id"]` | Required non-empty when enabled |
| `security.cors.allow_credentials` | bool | `false` | Cannot be `true` if any allowed origin contains `*` |

### Seed

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `seed.enable_dev_bootstrap` | bool | `false` | Seeds base RBAC and a dev admin at API startup |

### Modules

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `modules.flags` | `map[string]bool` | `{}` | Static module enablement overrides |

### Sukumad Submission Windows

#### `sukumad.submission_window.default`

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `sukumad.submission_window.default.start_hour` | int | `0` | Window disabled when both start/end are `0`; otherwise hours must be valid 0-23 |
| `sukumad.submission_window.default.end_hour` | int | `0` | Same validation as above |

#### `sukumad.submission_window.destinations`

Map keyed by destination key. Each entry supports:

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `start_hour` | int | none | Optional per-destination override |
| `end_hour` | int | none | Optional per-destination override |

### Sukumad Retry

#### `sukumad.retry.default`

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `sukumad.retry.default.max_retries` | int | `2` | Must be `>= 0` |

#### `sukumad.retry.destinations`

Map keyed by destination key:

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `max_retries` | int | none | Must be `>= 0` |

### Sukumad Response Filter

#### `sukumad.response_filter.default`

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `sukumad.response_filter.default.allowed_content_types` | `[]string` | `["application/json","application/*+json","application/xml","text/xml"]` | Allowed response MIME types |
| `sukumad.response_filter.default.allow_unknown` | bool | `false` | Accept responses with unknown content type |

#### `sukumad.response_filter.destinations`

Map keyed by destination key:

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `allowed_content_types` | `[]string` | none | Destination override |
| `allow_unknown` | bool | none | Destination override |

### Sukumad Retention

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `sukumad.retention.enabled` | bool | `false` | Enables retention worker behavior |
| `sukumad.retention.dry_run` | bool | `true` | When true, retention computes but does not delete |
| `sukumad.retention.terminal_age_days` | int | `30` | Must be `> 0` |
| `sukumad.retention.batch_size` | int | `100` | Must be `> 0` |

### Sukumad Outbound Rate Limit

#### `sukumad.rate_limit.default`

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `sukumad.rate_limit.default.requests_per_second` | float | `2` | Must be `> 0` |
| `sukumad.rate_limit.default.burst` | int | `2` | Must be `> 0` |

#### `sukumad.rate_limit.destinations`

Map keyed by destination key:

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `requests_per_second` | float | none | Destination override |
| `burst` | int | none | Destination override |

### Sukumad Directory Ingest

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `sukumad.ingest.directory.enabled` | bool | `false` | Enables directory ingest runtime |
| `sukumad.ingest.directory.inbox_path` | string | `""` | Inbox directory |
| `sukumad.ingest.directory.processing_path` | string | `""` | Working directory for claimed files |
| `sukumad.ingest.directory.processed_path` | string | `""` | Success archive directory |
| `sukumad.ingest.directory.failed_path` | string | `""` | Failure archive directory |
| `sukumad.ingest.directory.allowed_extensions` | `[]string` | `[".json"]` | Allowed file extensions |
| `sukumad.ingest.directory.default_source_system` | string | `directory` | Used when payload does not specify a source system |
| `sukumad.ingest.directory.require_idempotency_key` | bool | `true` | Rejects files without idempotency key when enabled |
| `sukumad.ingest.directory.debounce_milliseconds` | int | `1000` | Debounce interval before ingest |
| `sukumad.ingest.directory.retry_delay_seconds` | int | `30` | Retry delay after a failed claim/process attempt |
| `sukumad.ingest.directory.claim_timeout_seconds` | int | `300` | Claim timeout for in-flight ingest work |
| `sukumad.ingest.directory.scan_interval_seconds` | int | `30` | Directory scan cadence |
| `sukumad.ingest.directory.batch_size` | int | `10` | Max files processed per scan batch |

### Sukumad Workers

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `sukumad.workers.heartbeat_seconds` | int | `10` | Shared heartbeat interval |
| `sukumad.workers.recovery.stale_delivery_after_seconds` | int | `300` | Must be `> 0` |
| `sukumad.workers.send.interval_seconds` | int | `5` | Must be `> 0` |
| `sukumad.workers.send.batch_size` | int | `10` | Must be `> 0` |
| `sukumad.workers.retry.interval_seconds` | int | `5` | Must be `> 0` |
| `sukumad.workers.retry.batch_size` | int | `10` | Must be `> 0` |
| `sukumad.workers.poll.interval_seconds` | int | `5` | Must be `> 0` |
| `sukumad.workers.poll.batch_size` | int | `10` | Must be `> 0` |
| `sukumad.workers.poll.claim_timeout_seconds` | int | `60` | Must be `> 0` |
| `sukumad.workers.retention.interval_seconds` | int | `300` | Must be `> 0` |

## API CLI Flags

`backend/cmd/api` exposes a subset of config as CLI overrides.

| Flag | Maps to | Default |
| --- | --- | --- |
| `--config` | config file path | empty |
| `--logging-level` | `logging.level` | no override |
| `--logging-format` | `logging.format` | no override |
| `--server-port` | `server.port` | no override |
| `--shutdown-timeout` | `server.shutdown_timeout_seconds` | no override |
| `--database-dsn` | `database.dsn` | no override |
| `--database-max-open-conns` | `database.max_open_conns` | no override |
| `--database-max-idle-conns` | `database.max_idle_conns` | no override |
| `--database-auto-migrate` | `database.auto_migrate` | `false` unless explicitly passed |
| `--database-auto-migrate-lock-timeout` | `database.auto_migrate_lock_timeout_seconds` | no override |
| `--auth-access-ttl` | `auth.access_token_ttl_seconds` | no override |
| `--auth-refresh-ttl` | `auth.refresh_token_ttl_seconds` | no override |
| `--auth-jwt-signing-key` | `auth.jwt_signing_key` | no override |
| `--auth-password-hash-cost` | `auth.password_hash_cost` | no override |
| `--auth-api-token-enabled` | `auth.api_token_enabled` | `false` unless explicitly passed |
| `--auth-api-token-header` | `auth.api_token_header_name` | no override |
| `--auth-api-token-ttl` | `auth.api_token_ttl_seconds` | no override |
| `--auth-api-token-allow-bearer` | `auth.api_token_allow_bearer` | `false` unless explicitly passed |
| `--security-rate-limit-enabled` | `security.rate_limit.enabled` | `false` unless explicitly passed |
| `--security-rate-limit-rps` | `security.rate_limit.requests_per_second` | no override |
| `--security-rate-limit-burst` | `security.rate_limit.burst` | no override |
| `--security-cors-enabled` | `security.cors.enabled` | `false` unless explicitly passed |
| `--security-cors-allow-credentials` | `security.cors.allow_credentials` | `false` unless explicitly passed |
| `--seed-dev-admin` | seed behavior only | `false` |

`backend/cmd/worker` supports only `--config`.

`backend/cmd/migrate` supports `up` and `down` with `-config`, and `create -name <migration_name>`.

## Desktop Persisted Settings

Desktop settings are stored in the user config directory at:

```text
<os-user-config-dir>/basepro-desktop/settings.json
```

Examples:

- macOS: `~/Library/Application Support/basepro-desktop/settings.json`
- Linux: `~/.config/basepro-desktop/settings.json`
- Windows: `%AppData%\basepro-desktop\settings.json`

The file is written with `0600` permissions after saving to `settings.json.tmp` and renaming atomically.

### Desktop settings schema

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `apiBaseUrl` | string | `""` | Trimmed before save |
| `authMode` | string | `password` | Must be `password` or `api_token` |
| `apiToken` | string | omitted | Only retained when `authMode=api_token` |
| `refreshToken` | string | omitted | Trimmed; cleared if empty |
| `requestTimeoutSeconds` | int | `15` | Valid range `1..300` |
| `uiPrefs` | object | see below | Desktop UI preferences |
| `tablePrefs` | object | `{}` | Per-grid saved preferences keyed by storage key |

### `uiPrefs`

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `themeMode` | string | `system` | Must be `light`, `dark`, or `system` |
| `palettePreset` | string | `ocean` | Required non-empty |
| `navCollapsed` | bool | `false` | Initial drawer collapse state |
| `showSukumadMenu` | bool | `true` | Local visibility preference |
| `showAdministrationMenu` | bool | `true` | Local visibility preference |
| `pinActionsColumnRight` | bool | `true` | Grid default |
| `dataGridBorderRadius` | int | `12` | Valid range `4..32` |
| `navLabels` | `map[string]string` | `{}` | Empty or whitespace-only keys/values are removed |

### `tablePrefs[<storageKey>]`

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `version` | int | `1` | Normalized to `1` |
| `pageSize` | int | `25` | Must be positive |
| `density` | string | `standard` | Allowed: `compact`, `standard`, `comfortable` |
| `columnVisibility` | `map[string]bool` | `{}` | Saved hidden/visible columns |
| `columnOrder` | `[]string` | `[]` | Empty values are removed |
| `pinnedColumns.left` | `[]string` | `[]` | Empty values are removed |
| `pinnedColumns.right` | `[]string` | `[]` | Empty values are removed |

## Web Configuration

The web app uses both Vite env vars and browser `localStorage`.

### Vite environment variables

| Variable | Default | Purpose |
| --- | --- | --- |
| `VITE_API_BASE_URL` | empty | Primary API base URL used when no local override is stored |
| `VITE_APP_NAME` | `BasePro Web` | App display name |
| `VITE_ENABLE_DEVTOOLS` | empty / unused in current code | Reserved by README for devtools enablement |
| `VITE_APP_VERSION` | injected from `npm_package_version`, fallback `0.0.0` | Desktop frontend build-time version label |

Notes:

- `web/src/lib/api.ts` throws if no API base URL can be resolved.
- `desktop/frontend/vite.config.ts` injects `VITE_APP_VERSION`; this is not used by the web frontend.

### Web localStorage keys

| Key | Default | Purpose |
| --- | --- | --- |
| `basepro.web.api_base_url_override` | absent | Optional API base URL override; trailing slashes are trimmed |
| `basepro.web.ui_preferences` | absent | Theme and UI preferences |
| `basepro.web.refresh_token` | absent | Persisted refresh token |
| `basepro.web.bootstrap.v1` | absent | Cached bootstrap payload plus `cachedAt` timestamp |
| `app.datagrid.<storageKey>.v1` | absent | Per-grid preferences |

### `basepro.web.ui_preferences`

| Field | Type | Default | Notes |
| --- | --- | --- | --- |
| `mode` | string | `system` | Must be `light`, `dark`, or `system` |
| `preset` | string | `oceanic` | Trimmed; falls back if empty |
| `collapseNavByDefault` | bool | `false` | App shell drawer default |
| `showFooter` | bool | `true` | Footer visibility |
| `showSukumadMenu` | bool | `true` | Local visibility preference |
| `showAdministrationMenu` | bool | `true` | Local visibility preference |
| `pinActionsColumnRight` | bool | `true` | Grid default |
| `dataGridBorderRadius` | int | `12` | Clamped to `4..32` |
| `navLabels` | `map[string]string` | `{}` | Blank keys/values removed |

### `app.datagrid.<storageKey>.v1`

| Field | Type | Default |
| --- | --- | --- |
| `version` | int | `1` |
| `pageSize` | int | `25` |
| `columnVisibility` | object | `{}` |
| `columnOrder` | `[]string` | `[]` |
| `density` | string | `standard` |
| `pinnedColumns.left` | `[]string` | `[]` |
| `pinnedColumns.right` | `[]string` | `[]` |

### `basepro.web.bootstrap.v1`

Stored shape:

| Field | Type | Notes |
| --- | --- | --- |
| `cachedAt` | number | Unix epoch in milliseconds |
| `payload.version` | number | Bootstrap schema version |
| `payload.generatedAt` | string | Server-generated timestamp |
| `payload.app.version` | string | App version |
| `payload.app.commit` | string | Build commit |
| `payload.app.buildDate` | string | Build timestamp |
| `payload.branding` | object | Branding metadata from API |
| `payload.modules` | array | Effective module config |
| `payload.capabilities` | object | Settings capability summary |
| `payload.cache` | object | Cache hints including `maxStaleSeconds` |
| `payload.principal` | object | Optional principal summary |

## Wails Project Config

Current `desktop/wails.json` values:

| Key | Value |
| --- | --- |
| `name` | `desktop` |
| `outputfilename` | `desktop` |
| `frontend:install` | `npm install` |
| `frontend:build` | `npm run build` |
| `frontend:dev:watcher` | `npm run dev` |
| `frontend:dev:serverUrl` | `auto` |
| `author.name` | `Samuel Sekiwere` |
| `author.email` | `sekiskylink@gmail.com` |

## Operational Notes

- Backend config hot reload applies only when the reloaded file still validates. Invalid reloads are rejected and the prior snapshot stays active.
- The API server and worker both read the latest config snapshot at runtime for policies that are expected to change while running.
- The sample backend config currently includes development credentials and a development JWT signing key; that file should not be treated as a production-ready config.
