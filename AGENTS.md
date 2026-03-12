# AGENTS.md (Major Constraints / Contract)
## Desktop Skeleton App (Wails + Gin + React/MUI)

This file defines the project contract for automation agents (Codex CLI) and human contributors.

---

## 1) Source of Truth
- `docs/requirements.md` is the authoritative specification.
- Before starting any milestone, the agent **must read** `docs/requirements.md`.

---

## 2) Milestone Contract (Hard Rule)
- Work is milestone-based.
- **No milestone may begin until the previous milestone is complete.**
- A milestone is only complete when:
  - All required implementation tasks for that milestone are done,
  - **All tests pass**, including:
    - backend tests (`go test ./...`)
    - frontend route tests / smoke tests
  - `docs/status.md` is updated to reflect completion.

If any tests fail, the agent must fix them before claiming completion.

---

## 3) Prompts Hygiene
- A copy of each milestone prompt must be saved under `docs/prompts/` for traceability.
- **Do not commit** the prompt copies.
- Ensure `.gitignore` includes:
  - `docs/prompts/`

---

## 4) Status Updates
After a successful milestone:
- Update `docs/status.md`:
  - milestone name
  - what changed (high-level)
  - how to run tests / what passed
  - any known follow-ups

---

## 5) Commit Discipline
After a successful milestone (tests passing + status updated):
- The agent must propose a commit message in conventional style, e.g.:
  - `feat(backend): add jwt login + refresh rotation`
  - `feat(desktop): add setup screen + api base url persistence`
  - `test(frontend): add route smoke tests`
- The agent must explicitly prompt the user to commit (but must not claim the commit happened).

Do not propose commits when tests are failing.

---

## 6) Logging / Secrets
- Never log secrets (JWTs, refresh tokens, API tokens, passwords).
- Mask tokens if they must appear in debug output.
- Config files should not be committed if they contain secrets.

---

## 7) Graceful Shutdown
- Backend must use `signal.NotifyContext` and `http.Server.Shutdown(...)`.
- Any background goroutine must exit on context cancellation.

---

## 8) Config Hot Reload (Backend)
- Backend config is Viper-based and supports hot reload.
- Runtime readers must use an atomic snapshot (`atomic.Value`).
- If writing config to disk:
  - write to a temp file then atomic rename
  - avoid reload loops
  - keep “runtime state” separate from “config”

---

## 9) UI Constraints
- MUI-based admin dashboard layout:
  - Drawer + AppBar + content + footer
- Use MUI X Data Grid with advanced features.
- Themes: light/dark/system + accent palette presets persisted locally.

---

## 10) API-Only Desktop
- Desktop must never access DB directly.
- All domain data must go through Gin APIs.

---

# SukumadPro Development Contract

## Purpose

This contract defines how Sukumad functionality must be implemented within the BasePro architecture.

SukumadPro is built on top of BasePro. BasePro remains the **platform layer**, while Sukumad is implemented as a **domain module**.

The agent must follow these rules strictly when implementing new functionality.

---

# 1. Architectural Authority

BasePro architecture is authoritative.

Existing platform modules must not be replaced or duplicated.

Platform modules include:


backend/internal/auth
backend/internal/user
backend/internal/rbac
backend/internal/audit
backend/internal/config
backend/internal/middleware


These modules provide:

- authentication
- user management
- role-based access control
- audit logging
- configuration
- request middleware

Sukumad must reuse these capabilities.

The agent must **never introduce a second authentication system, RBAC system, or user management system**.

---

# 2. Sukumad Module Placement

All Sukumad functionality must be placed under:


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


The agent must **not create new top-level modules under `internal/` for Sukumad features**.

---

# 3. Backend Module Pattern

Each backend module must follow the BasePro layering pattern.

Expected files:


handler.go
service.go
repository.go
types.go


Responsibilities:

Handler  
HTTP request handling

Service  
business logic

Repository  
database operations

---

# 4. Vertical Feature Development Rule

Every feature must be implemented **vertically across the entire system**.

Each milestone must include:

- backend implementation
- web client implementation
- desktop client implementation
- tests
- documentation updates

The agent must not implement backend-only milestones unless explicitly instructed.

---

# 5. Web and Desktop Parity

Web and desktop clients must evolve together.

Whenever a feature introduces:

- new entities
- new management pages
- new operational views

The agent must implement the corresponding functionality in:


web/
desktop/


Both clients must expose the same capabilities unless explicitly documented.

Desktop must **never access the database directly** and must communicate with the backend using APIs.

---

# 6. Routing and Navigation Rules

The existing BasePro application shell must be reused.

The agent must:

- reuse the existing AppShell
- reuse the existing authenticated routing structure
- reuse the existing navigation drawer

The agent must not introduce:

- a second application shell
- a second router hierarchy
- a parallel navigation system

Sukumad pages must be added to the existing authenticated navigation.

---

# 7. Permission Enforcement

All new functionality must integrate with BasePro RBAC.

Example permissions:


servers.read
servers.write
requests.read
requests.write
deliveries.read
deliveries.write
jobs.read
jobs.write
observability.read


The agent must ensure:

- backend endpoints enforce permissions
- web navigation respects permissions
- desktop navigation respects permissions

---

# 8. Database Rules

Database migrations must be placed under:


backend/db/migrations


Platform tables owned by BasePro must not be duplicated.

These include:


users
roles
permissions
refresh_tokens
audit_logs


Sukumad domain tables may include:


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

# 9. Worker and Async Rules

Background processing must support:

- graceful shutdown
- context cancellation
- configurable concurrency

Workers must be implemented inside:


backend/internal/sukumad/worker


Worker logic must not bypass application configuration.

---

# 10. Testing Requirements

Every milestone must include tests.

Minimum requirements:

### Backend

- service tests
- repository tests where practical
- API handler tests where practical

### Web

- route rendering tests
- page interaction tests

### Desktop

- route rendering tests
- page interaction tests

The agent must **not claim milestone completion unless tests pass**.

---

# 11. Documentation Requirements

At the completion of each milestone the agent must:

- update `docs/status.md`
- maintain architecture notes under `docs/notes/`
- save the prompt used for the milestone under `docs/prompts/` (not committed)

---

# 12. Milestone Completion Criteria

A milestone is complete only when:

- backend builds successfully
- web builds successfully
- desktop builds successfully
- all tests pass
- navigation and permissions behave correctly
- documentation is updated

If any of these conditions are not met, the milestone must be considered incomplete.

---

# 13. Architectural Principle

The final architecture must always look like this:


BasePro Platform
auth
user
rbac
config
middleware
audit

Sukumad Domain
integration middleware


Sukumad extends the platform but must never replace it.

## END
