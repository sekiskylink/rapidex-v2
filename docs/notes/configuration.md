# Backend Configuration

This note documents the backend runtime configuration accepted by SukumadPro as represented by `backend/config/config.yaml`.

The backend config loader uses Viper in `backend/internal/config`. Values are read from in-code defaults, then YAML, then environment variables, then selected API CLI flags. The active config is validated and stored as an atomic runtime snapshot.

## Loading And Overrides

Default config file lookup:

- `./config/config.yaml`
- `./config.yaml`

Explicit config file flags:

- API server: `--config <path>`
- Worker: `--config <path>`
- Migrate command: `-config <path>`
- Demo seeder: `--config <path>`

Environment override format:

- Prefix: `BASEPRO_`
- Convert dot-separated keys to uppercase snake case.
- Example: `database.dsn` becomes `BASEPRO_DATABASE_DSN`.
- Example: `sukumad.workers.send.batch_size` becomes `BASEPRO_SUKUMAD_WORKERS_SEND_BATCH_SIZE`.

The API server also exposes selected CLI overrides. See the "API CLI Flags" section.

## Logging

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `logging.level` | string | `info` | Log level. Must be `debug`, `info`, `warn`, or `error`. |
| `logging.format` | string | `console` | Log format. Must be `console` or `json`. |

## Server

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `server.port` | string | `:8080` | HTTP listen address for the API server. |
| `server.shutdown_timeout_seconds` | int | `10` | Graceful shutdown timeout. Must be greater than `0`. |
| `server.cors_allowed_origins` | string array | `http://localhost:5173`, `http://127.0.0.1:5173`, `wails://wails.localhost`, `wails://wails.localhost:*` | Legacy/default CORS origin list. Values must not be empty. |

## Database

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `database.dsn` | string | sample: `postgres://postgres:postgres@127.0.0.1:5432/sukumadpro?sslmode=disable` | PostgreSQL DSN. Required. Do not commit production credentials. |
| `database.max_open_conns` | int | `10` | Maximum open DB connections. Must be greater than `0`. |
| `database.max_idle_conns` | int | `5` | Maximum idle DB connections. Must be greater than `0`. |
| `database.auto_migrate` | bool | `false` | Runs migrations on API/worker startup when enabled. |
| `database.auto_migrate_lock_timeout_seconds` | int | `30` | Advisory lock timeout for startup migrations. Must be greater than `0`. |

## Auth

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `auth.access_token_ttl_seconds` | int | `900` | JWT access token TTL. Must be greater than `0`. |
| `auth.refresh_token_ttl_seconds` | int | `604800` | Refresh token TTL. Must be greater than `0`. |
| `auth.jwt_signing_key` | string | sample: `dev-insecure-signing-key` | JWT signing secret. Required. Replace for non-development environments. |
| `auth.password_hash_cost` | int | `12` | bcrypt password hashing cost. Must be between `4` and `31`. |
| `auth.api_token_enabled` | bool | `true` | Enables API-token authentication support. |
| `auth.api_token_header_name` | string | `X-API-Token` | Header name for API tokens. Required. |
| `auth.api_token_ttl_seconds` | int | `2592000` | Default API token TTL. Must be greater than `0`. |
| `auth.api_token_allow_bearer` | bool | `false` | Allows API tokens through `Authorization: Bearer <token>` when enabled. |

## Security

### Rate Limit

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `security.rate_limit.enabled` | bool | `false` | Enables auth endpoint rate limiting. |
| `security.rate_limit.requests_per_second` | float | `5` | Allowed request rate when enabled. Must be greater than `0` when enabled. |
| `security.rate_limit.burst` | int | `10` | Token-bucket burst size when enabled. Must be greater than `0` when enabled. |

### CORS

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `security.cors.enabled` | bool | config sample: `true`, code default: `false` | Enables CORS middleware. |
| `security.cors.allowed_origins` | string array | config sample includes Wails and localhost dev origins; code default is empty | Required and non-empty when CORS is enabled. |
| `security.cors.allowed_methods` | string array | `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS` | Allowed CORS methods. Required and non-empty when CORS is enabled. |
| `security.cors.allowed_headers` | string array | `Authorization`, `Content-Type`, `X-API-Token`, `X-Request-Id` | Allowed CORS headers. Required and non-empty when CORS is enabled. |
| `security.cors.allow_credentials` | bool | `false` | Allows credentialed CORS requests. Cannot be `true` if any allowed origin contains `*`. |

## Seed

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `seed.enable_dev_bootstrap` | bool | `false` | Seeds base RBAC and a development admin user on API startup. Development-only. |

Related environment variables used during dev seeding:

| Variable | Default | Description |
| --- | --- | --- |
| `BASEPRO_DEV_ADMIN_USERNAME` | `admin` | Dev admin username. |
| `BASEPRO_DEV_ADMIN_PASSWORD` | `admin123!` | Dev admin password. Do not use this default outside local development. |

## Modules

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `modules.flags` | map string to bool | `{}` | Static module enablement overrides. Keys are module IDs and values force enabled/disabled state. |

Example:

```yaml
modules:
  flags:
    sukumad: true
```

## Sukumad

### Requests

Configured request metadata columns project selected `exchange_requests.extras` keys into the Requests table in both web and desktop.

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `sukumad.requests.metadata_columns` | array | `[]` | Optional list of metadata projections for request list tables and quick search. |

Entry shape:

```yaml
sukumad:
  requests:
    metadata_columns:
      - key: "patientId"
        label: "Patient ID"
        type: "string"
        searchable: true
        visible_by_default: true
      - key: "submittedAt"
        label: "Submitted At"
        type: "datetime"
        searchable: false
        visible_by_default: false
```

Allowed `type` values:

- `string`
- `number`
- `boolean`
- `datetime`

Notes:

- `key` maps to a top-level key in `exchange_requests.extras`.
- `searchable: true` includes that metadata key in the Requests page quick search (`q`).
- `visible_by_default: false` keeps the column available in the grid column chooser but hidden on first load.

### Submission Window

Submission windows control when deliveries may be sent. If both `start_hour` and `end_hour` are `0`, the window is disabled.

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `sukumad.submission_window.default.start_hour` | int | `0` | Default allowed-window start hour in UTC. Must be `0..23` when enabled. |
| `sukumad.submission_window.default.end_hour` | int | `0` | Default allowed-window end hour in UTC. Must be `0..23` when enabled and must differ from `start_hour`. |
| `sukumad.submission_window.destinations` | map | `{}` | Per-destination overrides keyed by integration server code/destination key. |

Destination entry shape:

```yaml
sukumad:
  submission_window:
    destinations:
      dhis2_uganda:
        start_hour: 6
        end_hour: 18
```

### Retry

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `sukumad.retry.default.max_retries` | int | `2` | Default max retry count. Must be `>= 0`. |
| `sukumad.retry.destinations` | map | `{}` | Per-destination retry overrides keyed by integration server code/destination key. |

Destination entry shape:

```yaml
sukumad:
  retry:
    destinations:
      dhis2_uganda:
        max_retries: 5
```

### Response Filter

The response filter controls whether delivery response bodies are retained. If a response content type is not allowed, the raw response body is cleared, `response_body_filtered` is set, and a safe summary is stored.

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `sukumad.response_filter.default.allowed_content_types` | string array | `application/json`, `application/*+json`, `application/xml`, `text/xml` | Allowed response MIME types for saved delivery response bodies. Supports exact matches and wildcard forms used by the code. |
| `sukumad.response_filter.default.allow_unknown` | bool | `false` | Allows saving responses with missing/unknown content type. |
| `sukumad.response_filter.destinations` | map | `{}` | Per-destination response-filter overrides keyed by integration server code/destination key. |

Destination entry shape:

```yaml
sukumad:
  response_filter:
    destinations:
      dhis2_uganda:
        allowed_content_types:
          - "application/json"
          - "text/plain"
        allow_unknown: false
```

### Retention

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `sukumad.retention.enabled` | bool | `false` | Enables the retention worker. |
| `sukumad.retention.dry_run` | bool | `true` | When true, retention scans and reports without deleting. |
| `sukumad.retention.terminal_age_days` | int | `30` | Terminal requests older than this are eligible for retention. Must be greater than `0`. |
| `sukumad.retention.batch_size` | int | `100` | Max retention candidates per run. Must be greater than `0`. |

### Outbound Rate Limit

Outbound rate limits throttle delivery calls per destination.

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `sukumad.rate_limit.default.requests_per_second` | float | `2` | Default outbound request rate. Must be greater than `0`. |
| `sukumad.rate_limit.default.burst` | int | `2` | Default outbound burst size. Must be greater than `0`. |
| `sukumad.rate_limit.destinations` | map | `{}` | Per-destination outbound rate limits keyed by integration server code/destination key. |

Destination entry shape:

```yaml
sukumad:
  rate_limit:
    destinations:
      dhis2_uganda:
        requests_per_second: 1
        burst: 1
```

### Directory Ingest

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `sukumad.ingest.directory.enabled` | bool | `false` | Enables filesystem directory ingestion. |
| `sukumad.ingest.directory.inbox_path` | string | `""` | Directory watched for new files. Required when directory ingest is enabled. |
| `sukumad.ingest.directory.processing_path` | string | `""` | Directory for claimed/in-progress files. Required when enabled. |
| `sukumad.ingest.directory.processed_path` | string | `""` | Directory for successfully processed files. Required when enabled. |
| `sukumad.ingest.directory.failed_path` | string | `""` | Directory for failed files. Required when enabled. |
| `sukumad.ingest.directory.allowed_extensions` | string array | `.json` | Allowed file extensions. Values must not be empty. |
| `sukumad.ingest.directory.default_source_system` | string | `directory` | Source system used when an ingested payload does not provide one. |
| `sukumad.ingest.directory.require_idempotency_key` | bool | `true` | Rejects files without an idempotency key when enabled. |
| `sukumad.ingest.directory.debounce_milliseconds` | int | `1000` | Delay before processing detected files. Must be greater than `0`. |
| `sukumad.ingest.directory.retry_delay_seconds` | int | `30` | Delay before retrying failed ingest work. Must be greater than `0`. |
| `sukumad.ingest.directory.claim_timeout_seconds` | int | `300` | Timeout for claimed ingest work. Must be greater than `0`. |
| `sukumad.ingest.directory.scan_interval_seconds` | int | `30` | Directory scan interval. Must be greater than `0`. |
| `sukumad.ingest.directory.batch_size` | int | `10` | Max files processed per ingest batch. Must be greater than `0`. |

### Workers

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `sukumad.workers.heartbeat_seconds` | int | `10` | Worker heartbeat interval. Must be greater than `0`. |
| `sukumad.workers.outbound_logging.enabled` | bool | `false` | Enables worker-process outbound DHIS2 request logs. |
| `sukumad.workers.outbound_logging.body_preview_bytes` | int | `256` | Max outbound request body preview size when worker outbound logging is enabled. Must be greater than `0` when enabled. |
| `sukumad.workers.recovery.stale_delivery_after_seconds` | int | `300` | Age after which running deliveries are considered stale for recovery. Must be greater than `0`. |
| `sukumad.workers.send.interval_seconds` | int | `5` | Send worker interval. Must be greater than `0`. |
| `sukumad.workers.send.batch_size` | int | `10` | Send worker batch size. Must be greater than `0`. |
| `sukumad.workers.retry.interval_seconds` | int | `5` | Retry worker interval. Must be greater than `0`. |
| `sukumad.workers.retry.batch_size` | int | `10` | Retry worker batch size. Must be greater than `0`. |
| `sukumad.workers.poll.interval_seconds` | int | `5` | Async poll worker interval. Must be greater than `0`. |
| `sukumad.workers.poll.batch_size` | int | `10` | Async poll worker batch size. Must be greater than `0`. |
| `sukumad.workers.poll.claim_timeout_seconds` | int | `60` | Async poll claim timeout. Must be greater than `0`. |
| `sukumad.workers.retention.interval_seconds` | int | `300` | Retention worker interval. Must be greater than `0`. |

Worker outbound logging is off by default. When enabled, the worker logs `worker_outbound_request` before DHIS2 submit and poll calls with method, sanitized URL, destination key, body size, and a small redacted body preview. It does not log headers, API tokens, authorization values, passwords, or full request bodies.

## API CLI Flags

Only the API server exposes these config overrides as command-line flags:

| Flag | Config key |
| --- | --- |
| `--logging-level` | `logging.level` |
| `--logging-format` | `logging.format` |
| `--server-port` | `server.port` |
| `--shutdown-timeout` | `server.shutdown_timeout_seconds` |
| `--database-dsn` | `database.dsn` |
| `--database-max-open-conns` | `database.max_open_conns` |
| `--database-max-idle-conns` | `database.max_idle_conns` |
| `--database-auto-migrate` | `database.auto_migrate` |
| `--database-auto-migrate-lock-timeout` | `database.auto_migrate_lock_timeout_seconds` |
| `--auth-access-ttl` | `auth.access_token_ttl_seconds` |
| `--auth-refresh-ttl` | `auth.refresh_token_ttl_seconds` |
| `--auth-jwt-signing-key` | `auth.jwt_signing_key` |
| `--auth-password-hash-cost` | `auth.password_hash_cost` |
| `--auth-api-token-enabled` | `auth.api_token_enabled` |
| `--auth-api-token-header` | `auth.api_token_header_name` |
| `--auth-api-token-ttl` | `auth.api_token_ttl_seconds` |
| `--auth-api-token-allow-bearer` | `auth.api_token_allow_bearer` |
| `--security-rate-limit-enabled` | `security.rate_limit.enabled` |
| `--security-rate-limit-rps` | `security.rate_limit.requests_per_second` |
| `--security-rate-limit-burst` | `security.rate_limit.burst` |
| `--security-cors-enabled` | `security.cors.enabled` |
| `--security-cors-allow-credentials` | `security.cors.allow_credentials` |

`--seed-dev-admin` is an API flag that affects startup seeding behavior, but it is not a YAML config key. It complements `seed.enable_dev_bootstrap`.

## Secrets

Treat these values as sensitive in production:

- `database.dsn`
- `auth.jwt_signing_key`
- any credentials embedded in environment variables or deployment config

The sample `backend/config/config.yaml` contains local development defaults and must not be used unchanged in production.
