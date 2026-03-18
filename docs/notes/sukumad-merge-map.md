# Sukumad → SukumadPro Merge Map

## 1. Purpose

This document defines how the existing **Sukumad** repository will be merged into **SukumadPro**, which is built on top of the **BasePro platform architecture**.

The goal is to preserve Sukumad’s **integration middleware capabilities** while reusing BasePro’s platform infrastructure.

SukumadPro must follow these principles:

- BasePro remains the **platform foundation**
- Sukumad becomes a **domain module**
- BasePro authentication and RBAC remain authoritative
- BasePro UI shell and routing remain authoritative

Sukumad must **not exist as a second application architecture** inside the repository.

---

# 2. BasePro Platform Architecture (Authoritative)

The BasePro backend currently contains platform modules under:
backend/internal/
Examples:
backend/internal/
auth/
user/
rbac/
audit/
config/
middleware/


These modules provide:

- authentication
- user management
- role-based access control
- configuration
- middleware
- audit logging

These modules **must not be replaced or duplicated** by Sukumad.

---

# 3. Sukumad Domain Module Placement

All Sukumad functionality will be placed under a single domain namespace:

backend/internal/sukumad


Example structure:


backend/internal/sukumad/

server/
request/
delivery/
async/
ratelimit/
worker/
dhis2/
observability/

This ensures the architecture remains consistent with BasePro.

---

# 4. Backend Module Structure

Each Sukumad module must follow the BasePro layering pattern:


handler.go
service.go
repository.go
types.go


Responsibilities:

Handler  
HTTP layer

Service  
business logic

Repository  
database access

---

# 5. Revised Merge Table

| Sukumad Source | Purpose | SukumadPro Target | Action | Notes |
|---|---|---|---|---|
| `cmd/api/main.go` | API bootstrap | `backend/cmd/server/main.go` | Adapt | Use BasePro bootstrap pattern |
| `config/` | runtime config | `backend/internal/config` | Adapt | Integrate worker and rate-limit settings |
| `db/migrations` | database schema | `backend/db/migrations` | Rewrite | Remove Sukumad auth tables |
| `internal/models` | domain models | `internal/sukumad/*/types.go` | Adapt | Split models by domain |
| `internal/repo/server_repo.go` | server persistence | `internal/sukumad/server/repository.go` | Adapt | Preserve logic |
| `internal/repo/request_repo.go` | request persistence | `internal/sukumad/request/repository.go` | Adapt | Align with BasePro DB patterns |
| `internal/repo/delivery_repo.go` | delivery queue | `internal/sukumad/delivery/repository.go` | Adapt | Possibly split async responsibilities |
| `internal/repo/refresh_repo.go` | auth refresh tokens | `internal/auth` | Delete | BasePro handles tokens |
| `internal/repo/permission_repo.go` | permission storage | `internal/rbac` | Delete | BasePro RBAC authoritative |
| `internal/handlers/server_handler.go` | server API | `internal/sukumad/server/handler.go` | Adapt | Integrate with BasePro router |
| `internal/handlers/request_handler.go` | request API | `internal/sukumad/request/handler.go` | Adapt | Align with BasePro API conventions |
| `internal/handlers/delivery_handler.go` | delivery operations | `internal/sukumad/delivery/handler.go` | Adapt | Add restart/retry operations |
| `internal/handlers/auth_handler.go` | login endpoints | `internal/auth` | Delete | BasePro auth authoritative |
| `internal/handlers/permission_handler.go` | permission API | `internal/rbac` | Delete | Duplicate functionality |
| `internal/services/*` | business logic | `internal/sukumad/*/service.go` | Adapt | Preserve domain behavior |
| `internal/workers/send_worker.go` | delivery worker | `internal/sukumad/worker/send.go` | Adapt | Must support graceful shutdown |
| `internal/workers/poll_worker.go` | async polling | `internal/sukumad/worker/poll.go` | Adapt | Poll async tasks |
| `internal/workers/retry_worker.go` | retry orchestration | `internal/sukumad/worker/retry.go` | Adapt | Delivery retry logic |
| `internal/dhis2/*` | DHIS2 integration | `internal/sukumad/dhis2/` | Keep/Adapt | Valuable integration logic |
| `internal/ratelimit/*` | rate limiting | `internal/sukumad/ratelimit/` | Adapt | Integrate with worker scheduler |
| Sukumad request lifecycle logic | request orchestration | `internal/sukumad/delivery/service.go` | Adapt | Preserve domain flow |
| Sukumad async status tracking | async monitoring | `internal/sukumad/async/` | Adapt | Manage polling and final status |
| Sukumad logging/tracing | observability | `internal/sukumad/observability/` | Adapt | Provide operational visibility |
| Sukumad user models | user management | `internal/user` | Delete | BasePro user module authoritative |
| Sukumad role models | RBAC | `internal/rbac` | Delete | BasePro RBAC authoritative |

---

# 6. Sukumad Backend Modules

## server

Manages downstream integration systems.


internal/sukumad/server/


Responsibilities:

- register servers
- manage credentials
- store rate limits
- test connectivity

---

## request

Represents integration requests.


internal/sukumad/request/


Responsibilities:

- create requests
- track lifecycle
- associate metadata

---

## delivery

Handles sending requests to servers.


internal/sukumad/delivery/


Responsibilities:

- manage delivery attempts
- schedule retries
- store responses

---

## async

Handles asynchronous workflows.


internal/sukumad/async/


Responsibilities:

- track async tasks
- schedule polling
- update final status

---

## ratelimit

Controls outbound traffic.


internal/sukumad/ratelimit/


Responsibilities:

- enforce RPS limits
- manage concurrency
- protect downstream systems

---

## worker

Background workers.


internal/sukumad/worker/


Worker types:

- send worker
- poll worker
- retry worker

---

## dhis2

DHIS2-specific integration logic.


internal/sukumad/dhis2/


Responsibilities:

- submission
- async import monitoring
- error interpretation

---

## observability

Operational visibility.


internal/sukumad/observability/


Responsibilities:

- request tracing
- event logs
- worker monitoring

---

# 7. Frontend Integration

## Web

New pages under:


web/src/pages/


Pages:


ServersPage.tsx
RequestsPage.tsx
DeliveriesPage.tsx
JobsPage.tsx
ObservabilityPage.tsx


These must integrate into the **existing BasePro AppShell**.

---

## Desktop

New pages under:


desktop/frontend/src/pages/


Pages mirror the web UI.

Desktop must **only communicate with backend APIs**.

---

# 8. Permissions

Permissions must be managed through BasePro RBAC.

Suggested permissions:


servers.read
servers.write
requests.read
requests.write
deliveries.read
deliveries.write
jobs.read
jobs.write
observability.read


Navigation visibility must follow permission checks.

---

# 9. Database Migration Rules

The Sukumad database schema must be adapted.

Changes required:

- remove Sukumad authentication tables
- reuse BasePro platform tables
- implement Sukumad domain tables

Recommended domain entities:


integration_servers
exchange_requests
request_targets
delivery_attempts
async_tasks
async_task_polls
request_events
rate_limit_policies
worker_runs


---

# 10. Migration Milestones

### Milestone 0 — Discovery

Produce:

- architecture note
- DB architecture
- merge map

---

### Milestone 1 — Bootstrap

Create:


internal/sukumad


Add placeholder modules.

---

### Milestone 2 — Server Management

Implement:

- migrations
- server CRUD
- UI pages

---

### Milestone 3 — Requests and Deliveries

Implement:

- request lifecycle
- delivery attempts
- retry functionality

---

### Milestone 4 — Async Processing

Implement:

- async tasks
- polling workers
- status reconciliation

---

### Milestone 5 — Observability

Implement:

- event logs
- worker monitoring
- request tracing

---

### Milestone 6 — Legacy Removal

Remove remaining Sukumad authentication and permission code.

Ensure full alignment with BasePro platform modules.

---

# 11. Architecture Principle

SukumadPro must follow this architecture:


BasePro Platform
auth
user
rbac
config
middleware
audit

Sukumad Domain
integration middleware


Sukumad must **extend the platform, not replace it**.
