# SukumadPro Target Repository Structure

## 1. Purpose

This document defines the **intended repository structure for SukumadPro**, which is built on top of the BasePro platform.

It serves two purposes:

1. Provide architectural clarity for developers.
2. Prevent automation agents (such as Codex) from inventing inconsistent folder structures.

SukumadPro must preserve the BasePro platform architecture and extend it with Sukumad integration capabilities.

---

# 2. Architectural Principle

The repository must follow this architecture:

BasePro Platform  
+  
Sukumad Domain Module

BasePro provides the platform:

- authentication
- users
- role-based access control
- configuration
- middleware
- audit logging
- application shells

Sukumad provides the integration middleware domain.

Sukumad **must not replace or duplicate BasePro platform modules**.

---

# 3. Repository Root


sukumadpro/

backend/
web/
desktop/

docs/
scripts/



---

# 4. Backend Structure

Backend code lives under:


backend/


### Backend layout


backend/

cmd/
server/

internal/

auth/
user/
rbac/
audit/
config/
middleware/

sukumad/

    server/
    request/
    delivery/
    async/
    ratelimit/
    worker/
    dhis2/
    observability/

db/

migrations/

pkg/


---

# 5. Backend Platform Modules (BasePro)

These modules already exist in BasePro and must remain authoritative.


backend/internal/auth
backend/internal/user
backend/internal/rbac
backend/internal/audit
backend/internal/config
backend/internal/middleware


Responsibilities:

### auth

- login
- JWT handling
- refresh tokens
- session validation

### user

- user management
- user profiles

### rbac

- roles
- permissions
- permission checks

### config

- environment configuration
- runtime configuration

### middleware

- request logging
- authentication middleware
- request IDs
- error handling

### audit

- audit events
- user activity logging

---

# 6. Sukumad Domain Modules

All Sukumad features live under:


backend/internal/sukumad


Modules:


server/
request/
delivery/
async/
ratelimit/
worker/
dhis2/
observability/


---

## 6.1 server


internal/sukumad/server/


Responsibilities:

- manage integration servers
- store credentials
- manage connectivity
- configure rate limits

Files:


handler.go
service.go
repository.go
types.go


---

## 6.2 request


internal/sukumad/request/


Responsibilities:

- create integration requests
- track lifecycle
- manage request metadata

Files:


handler.go
service.go
repository.go
types.go


---

## 6.3 delivery


internal/sukumad/delivery/


Responsibilities:

- manage delivery attempts
- schedule retries
- store responses
- handle delivery status

Files:


handler.go
service.go
repository.go
types.go


---

## 6.4 async


internal/sukumad/async/


Responsibilities:

- manage asynchronous tasks
- poll remote systems
- update async status

Files:


service.go
repository.go
poller.go


---

## 6.5 ratelimit


internal/sukumad/ratelimit/


Responsibilities:

- enforce request rate limits
- manage concurrency
- protect downstream systems

Files:


limiter.go
policy.go


---

## 6.6 worker


internal/sukumad/worker/


Responsibilities:

- background processing
- delivery workers
- polling workers
- retry workers

Files:


send.go
poll.go
retry.go


---

## 6.7 dhis2


internal/sukumad/dhis2/


Responsibilities:

- DHIS2 submission logic
- async import monitoring
- DHIS2 response interpretation

Files:


client.go
submission.go
async.go


---

## 6.8 observability


internal/sukumad/observability/


Responsibilities:

- request tracing
- event logs
- system monitoring
- worker visibility

Files:


events.go
tracing.go
metrics.go


---

# 7. Database Structure

Database migrations live under:


backend/db/migrations


Primary domain tables:


integration_servers
exchange_requests
request_targets
delivery_attempts
async_tasks
async_task_polls
request_events
rate_limit_policies
worker_runs


Platform tables from BasePro must remain:


users
roles
permissions
refresh_tokens
audit_logs


---

# 8. Web Frontend

Web UI lives under:


web/


Key directories:


web/src/

auth/
components/
lib/
pages/
rbac/
ui/


Sukumad pages must be added under:


web/src/pages


Pages:


ServersPage.tsx
RequestsPage.tsx
DeliveriesPage.tsx
JobsPage.tsx
ObservabilityPage.tsx


These pages must integrate into the existing **BasePro AppShell**.

---

# 9. Desktop Application

Desktop UI lives under:


desktop/


Key structure:


desktop/frontend/src/

api/
auth/
components/
notifications/
pages/
settings/
ui/


Sukumad pages should be added under:


desktop/frontend/src/pages


Pages mirror the web interface.

Desktop must **only communicate with backend APIs**.

---

# 10. Documentation

Documentation lives under:


docs/


Important directories:


docs/notes/
docs/prompts/
docs/status.md
docs/requirements.md


Sukumad architecture documentation:


docs/notes/sukumad-overview.md
docs/notes/sukumad-architecture.md
docs/notes/sukumad-db-architecture.md
docs/notes/sukumad-merge-map.md
docs/notes/sukumad-target-repo-structure.md


---

# 11. Scripts

Operational scripts live under:


scripts/


Examples:


scripts/dev.sh
scripts/migrate.sh
scripts/test.sh


---

# 12. Migration Strategy

Sukumad integration should proceed in stages.

### Stage 1

Bootstrap Sukumad module.

Create:


internal/sukumad


---

### Stage 2

Implement server management.

---

### Stage 3

Implement request lifecycle.

---

### Stage 4

Implement delivery orchestration.

---

### Stage 5

Implement async polling.

---

### Stage 6

Implement observability.

---

# 13. Architectural Constraints

The repository must obey the following rules:

- BasePro platform modules remain authoritative.
- Sukumad is implemented only inside `internal/sukumad`.
- Desktop must never access the database directly.
- UI must reuse BasePro shells and navigation.
- Permissions must use BasePro RBAC.

---

# 14. Final Architecture


BasePro Platform
auth
user
rbac
config
middleware
audit

Sukumad Domain
integration middleware
