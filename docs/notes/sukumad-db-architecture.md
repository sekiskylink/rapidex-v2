# Sukumad Database Architecture

## 1. Purpose

This document defines the database architecture for the Sukumad integration middleware when implemented within the BasePro platform.

The database must support:

- integration server configuration
- request intake and traceability
- delivery orchestration
- asynchronous task monitoring
- rate limiting
- retry and restart workflows
- operational observability

The schema must integrate cleanly with BasePro and must **not duplicate BasePro platform tables such as users, roles, and permissions**.

---

# 2. Platform Tables (Provided by BasePro)

These tables are owned by BasePro and must **not be reimplemented by Sukumad**.

Examples include:

- users
- roles
- permissions
- user_roles
- refresh_tokens
- audit_logs

Sukumad may reference platform user IDs when needed.

---

# 3. Sukumad Domain Tables

The Sukumad middleware requires the following domain tables:

integration_servers  
integration_server_credentials  
exchange_requests  
request_targets  
delivery_attempts  
async_tasks  
async_task_polls  
rate_limit_policies  
request_events  
worker_runs

These tables support the full lifecycle of integration operations.

---

# 4. integration_servers

## Purpose

Defines external systems that Sukumad communicates with.

Examples:

- DHIS2
- RapidPro
- Other APIs

## Columns

| Column | Type | Description |
|------|------|-------------|
| id | bigserial | primary key |
| uid | uuid | public identifier |
| name | text | server name |
| code | text | unique short identifier |
| system_type | text | dhis2, generic, etc |
| base_url | text | base API URL |
| endpoint_type | text | http, graphql |
| http_method | text | GET, POST |
| use_async | boolean | whether remote system supports async |
| parse_responses | boolean | whether responses should be parsed |
| suspended | boolean | disable server |
| headers | jsonb | custom HTTP headers |
| url_params | jsonb | default query parameters |
| created_at | timestamptz | creation timestamp |
| updated_at | timestamptz | last update |
| created_by | bigint | reference to BasePro user |

---

# 5. integration_server_credentials

## Purpose

Stores credentials used when communicating with external systems.

Separating credentials from server metadata allows secure rotation and better management.

## Columns

| Column | Type | Description |
|------|------|-------------|
| id | bigserial | primary key |
| server_id | bigint | FK to integration_servers |
| credential_type | text | basic, bearer, api_key |
| username | text | optional |
| password_encrypted | text | encrypted password |
| token_encrypted | text | encrypted token |
| api_key_encrypted | text | encrypted API key |
| valid_from | timestamptz | credential start |
| valid_to | timestamptz | credential expiry |
| is_active | boolean | active credential |
| created_at | timestamptz | creation timestamp |
| updated_at | timestamptz | last update |

---

# 6. rate_limit_policies

## Purpose

Defines rate limiting policies for outbound integration traffic.

Rate limits protect downstream systems from overload.

## Columns

| Column | Type | Description |
|------|------|-------------|
| id | bigserial | primary key |
| name | text | policy name |
| scope_type | text | global, server |
| scope_ref | text | server code or global |
| rps | integer | requests per second |
| burst | integer | burst capacity |
| max_concurrency | integer | worker concurrency |
| timeout_ms | integer | request timeout |
| is_active | boolean | policy enabled |
| created_at | timestamptz | creation timestamp |

---

# 7. exchange_requests

## Purpose

Represents a logical integration request submitted into Sukumad.

A request may generate one or more delivery attempts.

## Columns

| Column | Type | Description |
|------|------|-------------|
| id | bigserial | primary key |
| uid | uuid | request identifier |
| source_system | text | originating system |
| destination_server_id | bigint | FK to integration_servers |
| batch_id | text | optional batch grouping |
| correlation_id | text | tracing identifier |
| idempotency_key | text | prevents duplicate processing |
| payload_body | text | request body |
| payload_format | text | json, xml |
| url_suffix | text | optional endpoint suffix |
| status | text | pending, processing, completed |
| extras | jsonb | additional metadata |
| created_at | timestamptz | creation timestamp |
| updated_at | timestamptz | last update |
| created_by | bigint | BasePro user |

---

# 8. request_targets

## Purpose

Represents expanded destinations for a request.

A request may be delivered to multiple targets.

Example:

Primary DHIS2 server  
Secondary CC servers

## Columns

| Column | Type | Description |
|------|------|-------------|
| id | bigserial | primary key |
| request_id | bigint | FK to exchange_requests |
| server_id | bigint | FK to integration_servers |
| target_kind | text | primary or cc |
| priority | integer | delivery order |
| status | text | pending, processing, done |
| created_at | timestamptz | creation timestamp |

---

# 9. delivery_attempts

## Purpose

Represents attempts to send data to a server.

A request target may have multiple attempts due to retries.

## Columns

| Column | Type | Description |
|------|------|-------------|
| id | bigserial | primary key |
| request_target_id | bigint | FK to request_targets |
| attempt_number | integer | retry count |
| status | text | processing, failed, success |
| scheduled_at | timestamptz | when attempt scheduled |
| started_at | timestamptz | when execution started |
| finished_at | timestamptz | completion time |
| next_run_at | timestamptz | next retry time |
| status_code | integer | HTTP status |
| response_body | text | server response |
| error_message | text | error details |
| worker_id | text | worker processing attempt |
| created_at | timestamptz | creation timestamp |

---

# 10. async_tasks

## Purpose

Represents asynchronous jobs started in external systems.

Example:

DHIS2 async import tasks.

## Columns

| Column | Type | Description |
|------|------|-------------|
| id | bigserial | primary key |
| delivery_attempt_id | bigint | FK to delivery_attempts |
| remote_job_id | text | external job identifier |
| poll_url | text | endpoint for polling status |
| remote_status | text | current remote status |
| terminal_state | text | success, failure |
| remote_response | jsonb | final response |
| next_poll_at | timestamptz | next poll time |
| completed_at | timestamptz | final completion |
| created_at | timestamptz | creation timestamp |

---

# 11. async_task_polls

## Purpose

Stores polling history for async tasks.

This table provides traceability and debugging capability.

## Columns

| Column | Type | Description |
|------|------|-------------|
| id | bigserial | primary key |
| async_task_id | bigint | FK to async_tasks |
| polled_at | timestamptz | poll timestamp |
| status_code | integer | HTTP status |
| remote_status | text | returned status |
| response_body | text | poll response |
| error_message | text | polling error |
| duration_ms | integer | poll duration |

---

# 12. request_events

## Purpose

Provides an append-only event log for requests.

Events improve observability and debugging.

Example events:

request.created  
delivery.claimed  
delivery.failed  
async.started  
async.completed

## Columns

| Column | Type | Description |
|------|------|-------------|
| id | bigserial | primary key |
| request_id | bigint | FK to exchange_requests |
| delivery_attempt_id | bigint | optional |
| async_task_id | bigint | optional |
| event_type | text | event name |
| event_data | jsonb | event metadata |
| actor_user_id | bigint | optional user |
| actor_type | text | system, worker, user |
| created_at | timestamptz | event timestamp |

---

# 13. worker_runs

## Purpose

Tracks worker activity for operational monitoring.

## Columns

| Column | Type | Description |
|------|------|-------------|
| id | bigserial | primary key |
| worker_type | text | send, poll, retry |
| worker_name | text | worker instance |
| status | text | running, stopped |
| started_at | timestamptz | start time |
| stopped_at | timestamptz | stop time |
| last_heartbeat_at | timestamptz | health check |
| meta | jsonb | worker metadata |

---

# 14. Relationships

Core relationships:

Server  
→ many ExchangeRequests

ExchangeRequest  
→ many RequestTargets

RequestTarget  
→ many DeliveryAttempts

DeliveryAttempt  
→ optional AsyncTask

AsyncTask  
→ many AsyncTaskPolls

ExchangeRequest  
→ many RequestEvents

---

# 15. Design Principles

The database design must emphasize:

Traceability  
Recoverability  
Operational transparency  
System protection

Every request must be fully observable from creation to final completion.

---

# 16. Migration Strategy

When integrating Sukumad into BasePro:

1. Retire Sukumad authentication tables.
2. Reuse BasePro platform tables.
3. Implement Sukumad domain tables via new migrations.
4. Maintain backwards compatibility where necessary.

All migrations must be written using BasePro migration conventions.
