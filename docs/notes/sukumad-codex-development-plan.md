# Sukumad Codex Development Plan

## 1. Purpose

This document defines how Codex should implement SukumadPro in controlled milestones.

The plan must ensure:

- BasePro architecture remains authoritative
- Sukumad is implemented under `backend/internal/sukumad`
- every milestone includes backend, web, and desktop work
- tests must pass before the next milestone begins
- no milestone is considered complete until documentation is updated

This plan is intended for iterative Codex CLI execution.

---

## 2. Global Rules for Every Milestone

Before doing any work, Codex must:

1. Read `AGENTS.md`
2. Read `docs/requirements.md`
3. Read `docs/status.md`
4. Read all relevant notes under `docs/notes/`

Hard rules:

- Do not begin a new milestone until the previous one is complete
- Every milestone must include:
  - backend implementation
  - web implementation
  - desktop implementation
  - tests
  - documentation updates
- Do not claim completion unless all applicable tests pass
- Update `docs/status.md` after each successful milestone
- Save a copy of the prompt under `docs/prompts/` but do not commit it
- Reuse existing BasePro routing, navigation, AppShell, RBAC, and API conventions
- Do not introduce a second auth, user, or permission system
- Desktop must only use backend APIs
- New routes/pages must be added to both web and desktop in the same milestone unless explicitly impossible

Definition of done for every milestone:

- backend builds
- web builds
- desktop builds
- backend tests pass
- web tests pass
- desktop tests pass where applicable
- route guards and permission visibility are verified
- `docs/status.md` updated

---

## 3. Delivery Principle

SukumadPro must be built vertically, not in isolated layers.

That means each milestone should deliver a thin but complete slice across:

- database
- backend API
- web UI
- desktop UI
- permissions
- tests
- docs

Do not finish backend-only milestones first and postpone all UI work.
Do not finish web-only milestones first and postpone desktop.
Do not build desktop features that web does not have.
The two clients must evolve together.

---

## 4. Milestone Sequence

## Milestone 0 — Discovery and Mapping

### Goal
Create the planning artifacts needed for safe migration.

### Scope
- no production feature code yet
- architecture and mapping only

### Deliverables
- `docs/notes/sukumad-overview.md`
- `docs/notes/sukumad-architecture.md`
- `docs/notes/sukumad-db-architecture.md`
- `docs/notes/sukumad-merge-map.md`
- `docs/notes/sukumad-target-repo-structure.md`
- proposed navigation additions for both web and desktop
- merge inventory of Sukumad packages into BasePro targets

### Backend
- none beyond discovery notes

### Web
- none beyond navigation planning

### Desktop
- none beyond navigation planning

### Tests
- existing tests must still pass after any doc-only changes

### Completion gate
- planning docs committed
- `docs/status.md` updated

---

## Milestone 1 — Bootstrap SukumadPro Structure

### Goal
Create the initial SukumadPro scaffolding in the BasePro codebase.

### Backend
Create:

- `backend/internal/sukumad/server/`
- `backend/internal/sukumad/request/`
- `backend/internal/sukumad/delivery/`
- `backend/internal/sukumad/async/`
- `backend/internal/sukumad/ratelimit/`
- `backend/internal/sukumad/worker/`
- `backend/internal/sukumad/dhis2/`
- `backend/internal/sukumad/observability/`

Each module should have initial placeholders where appropriate.

Add:
- route registration placeholders
- permission constants/placeholders
- initial config placeholders if needed

### Web
Add placeholder pages under `web/src/pages/`:

- `ServersPage.tsx`
- `RequestsPage.tsx`
- `DeliveriesPage.tsx`
- `JobsPage.tsx`
- `ObservabilityPage.tsx`

Add matching routes into the existing authenticated route tree.

Add navigation entries using BasePro navigation conventions.

Guard pages with permissions.

### Desktop
Add placeholder pages under `desktop/frontend/src/pages/`:

- `ServersPage.tsx`
- `RequestsPage.tsx`
- `DeliveriesPage.tsx`
- `JobsPage.tsx`
- `ObservabilityPage.tsx`

Add matching routes into the existing authenticated desktop route tree.

Add matching navigation entries.

Guard pages with permissions.

### Tests
- backend build/tests pass
- web route/smoke tests pass
- desktop route/smoke tests pass
- navigation renders correctly in both clients

### Completion gate
- both clients show the new Sukumad navigation/pages
- all tests pass
- `docs/status.md` updated

---

## Milestone 2 — Server Management

### Goal
Implement server management end to end.

### Backend
Implement:
- migrations for integration servers and credentials
- repository/service/handler for server CRUD
- validation
- permission enforcement
- audit logging for create/update/activate/deactivate
- list/detail APIs

### Web
Implement:
- server list page
- create/edit server form
- view/details page if applicable
- filtering, sorting, pagination where consistent with BasePro
- permission-aware actions

### Desktop
Implement:
- server list page
- create/edit server form
- view/details page if applicable
- same capabilities as web
- permission-aware actions

### Tests
- backend service/repository/handler tests
- web page tests
- desktop page tests
- permission visibility tests
- form validation tests where practical

### Completion gate
- a user can manage servers from both web and desktop
- tests pass in backend, web, and desktop
- `docs/status.md` updated

---

## Milestone 3 — Request Lifecycle

### Goal
Implement request creation, listing, and detail visibility.

### Backend
Implement:
- migrations for exchange requests
- request repository/service/handler
- list/detail endpoints
- request status model
- correlation/idempotency fields
- permission enforcement
- audit logging where relevant

### Web
Implement:
- requests list page
- request detail page
- filters and status display
- permission-aware actions
- traceability fields display

### Desktop
Implement:
- requests list page
- request detail page
- same filters and status display
- permission-aware actions
- traceability fields display

### Tests
- backend tests
- web page tests
- desktop page tests
- route guard tests
- serialization/status tests if needed

### Completion gate
- request lifecycle visible in both clients
- tests pass
- `docs/status.md` updated

---

## Milestone 4 — Delivery Attempts and Retry Controls

### Goal
Implement delivery attempt tracking and retry controls.

### Backend
Implement:
- migrations for request targets and/or delivery attempts
- delivery repository/service/handler
- retry scheduling model
- restart/retry endpoints
- delivery status transitions
- permission enforcement
- audit logging for manual retry/restart operations

### Web
Implement:
- deliveries list page
- delivery detail page
- retry/restart controls
- status history presentation

### Desktop
Implement:
- deliveries list page
- delivery detail page
- retry/restart controls
- status history presentation

### Tests
- backend retry/status tests
- web interaction tests
- desktop interaction tests
- permission tests around retry/restart actions

### Completion gate
- delivery lifecycle manageable in both clients
- tests pass
- `docs/status.md` updated

---

## Milestone 5 — Async Tasks and Polling Visibility

### Goal
Implement first-class async task support.

### Backend
Implement:
- migrations for async tasks and async task polls
- async repository/service
- polling orchestration
- APIs for task list/detail/status
- worker-safe polling state updates
- graceful shutdown support

### Web
Implement:
- jobs page
- async task list
- async task detail
- poll status display
- refresh/status UI

### Desktop
Implement:
- jobs page
- async task list
- async task detail
- poll status display
- refresh/status UI

### Tests
- backend async/polling tests
- worker shutdown/context tests
- web page tests
- desktop page tests

### Completion gate
- async tasks visible in both clients
- polling behavior tested
- tests pass
- `docs/status.md` updated

---

## Milestone 6 — Rate Limiting and Worker Governance

### Goal
Implement configurable worker and throttling control.

### Backend
Implement:
- rate-limit policy model
- worker state tracking
- concurrency controls
- APIs for worker status and rate-limit visibility
- config integration for worker counts and throttling

### Web
Implement:
- worker status views
- rate-limit visibility
- read-only control displays initially unless write actions are approved
- operational status cards/tables

### Desktop
Implement:
- worker status views
- rate-limit visibility
- same operational displays as web

### Tests
- backend policy tests
- worker state tests
- web page tests
- desktop page tests

### Completion gate
- operators can inspect worker/rate-limit state from both clients
- tests pass
- `docs/status.md` updated

---

## Milestone 7 — Observability and Traceability

### Goal
Implement event history and traceability views.

### Backend
Implement:
- request events model
- observability repository/service/handler
- trace/event APIs
- correlation-aware query support

### Web
Implement:
- observability page
- request event timeline or history view
- filters/search
- trace detail display

### Desktop
Implement:
- observability page
- request event timeline or history view
- filters/search
- trace detail display

### Tests
- backend observability tests
- web page tests
- desktop page tests

### Completion gate
- request traceability is visible in both clients
- tests pass
- `docs/status.md` updated

---

## Milestone 8 — DHIS2 Integration Layer

### Goal
Implement DHIS2-specific delivery and async handling.

### Backend
Implement:
- DHIS2 client logic
- submission service
- response parsing
- async import interpretation
- terminal state reconciliation

### Web
Implement:
- DHIS2-specific status details where useful
- request/delivery detail enrichments
- server configuration options relevant to DHIS2

### Desktop
Implement:
- matching DHIS2-specific status details
- matching server configuration options
- parity with web

### Tests
- backend DHIS2 integration unit tests
- response parsing tests
- web UI tests
- desktop UI tests

### Completion gate
- DHIS2 workflows are visible and manageable in both clients
- tests pass
- `docs/status.md` updated

---

## Milestone 9 — Legacy Sukumad Removal and Hardening

### Goal
Remove any remaining duplicated Sukumad platform concerns and harden the integrated system.

### Backend
- remove old Sukumad auth/user/permission remnants
- verify all Sukumad permissions use BasePro RBAC
- clean up duplicate config paths
- tighten error handling/logging
- review migrations and indexes

### Web
- remove temporary placeholders
- polish navigation and permission-driven visibility
- ensure consistent page behavior

### Desktop
- remove temporary placeholders
- polish navigation and permission-driven visibility
- ensure consistent page behavior

### Tests
- full backend test suite
- full web test suite
- full desktop test suite
- smoke pass across all major flows

### Completion gate
- no active duplicate auth/user/permission system remains
- all tests pass
- `docs/status.md` updated

---

## 5. Prompt Template for Every Milestone

Use this structure whenever prompting Codex:

### Standard milestone prompt template

Before doing anything:

1. Read `AGENTS.md`
2. Read `docs/requirements.md`
3. Read `docs/status.md`
4. Read the relevant files in `docs/notes/`

Rules:
- Follow BasePro architecture exactly
- Keep Sukumad under `backend/internal/sukumad`
- Implement the milestone across backend, web, and desktop in the same iteration
- Do not leave one client behind unless explicitly documented as impossible
- Do not claim completion unless tests pass
- Update `docs/status.md`
- Save this prompt under `docs/prompts/` but do not commit it

Task:
[describe the milestone]

Required deliverables:
- backend code
- web code
- desktop code
- tests
- docs/status update

Completion requirements:
- backend tests pass
- web tests pass
- desktop tests pass
- builds succeed
- route guards and permission visibility are verified

---

## 6. UI Parity Rule

Web and desktop do not need pixel-perfect duplication, but they must have functional parity.

That means for each milestone:

- both clients must expose the same primary capabilities
- both clients must respect the same permissions
- both clients must use the same backend APIs
- both clients must make the same entities visible and manageable

If a feature exists in web but not desktop, or desktop but not web, the milestone is incomplete unless the gap is documented and explicitly accepted.

---

## 7. Testing Rule

Every milestone must include tests appropriate to each layer.

### Backend
At minimum:
- unit tests for services
- repository tests where practical
- handler/API tests where practical

### Web
At minimum:
- route rendering or smoke tests
- page interaction tests for new forms and lists
- permission visibility tests where practical

### Desktop
At minimum:
- route rendering or smoke tests
- page interaction tests for new forms and lists
- permission visibility tests where practical

If a test type is not currently available in the project, Codex should add the minimum sustainable testing structure instead of skipping tests silently.

---

## 8. Final Principle

SukumadPro must be built as:

BasePro platform  
plus  
Sukumad domain module

Every iteration must keep the system coherent across:

- backend
- web
- desktop
- permissions
- tests
- docs

This is mandatory.
