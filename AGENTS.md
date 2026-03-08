# AGENTS.md (Major Constraints / Contract)
# BasePro Agent Guide

BasePro is both:

1. A **reusable application skeleton**
2. A **foundation for real applications built on top of it**

Agents must recognise when work concerns:

- platform capabilities (skeleton improvements)
- domain modules (application features)

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

## 11) Platform Skeleton vs Application Projects

This repository is a **reusable platform skeleton** first, and later may become a concrete application.

The skeleton phase focuses on shared platform capabilities:

* authentication
* RBAC (roles and permissions)
* audit logging
* settings management
* shared application shell
* backend API contracts
* desktop/web clients consuming the same backend

During skeleton work:

* prioritize platform features over domain features
* avoid domain placeholders that are not defined by requirements
* keep default navigation focused on:
  * Dashboard
  * Administration
  * Settings

When a project intentionally transitions to domain modules:

1. Define domain scope in `docs/requirements.md`.
2. Update navigation structure intentionally.
3. Record the transition and scope in `docs/status.md`.

Agents must allow domain expansion when requirements explicitly define it.

---

## 12) Navigation Architecture

Navigation must be grouped and scalable.

Baseline skeleton grouping:

* Dashboard
* Administration
  * Users
  * Roles
  * Permissions
  * Audit Log
* Settings

Applications may add groups (for example `Operations`, `Reporting`, `Finance`) only when defined in requirements.

Rules:

* keep navigation grouped, not flat
* keep platform/system management under `Administration` by default
* apply RBAC visibility rules to navigation items
* desktop and web may differ in responsive layout, but grouping and permission intent must remain consistent

---

## 13) Cross-Client Parity Contract

BasePro has three core layers:

* Gin backend API
* Wails desktop client
* React/TypeScript web client

Shared platform capabilities are expected in both desktop and web.
Temporary parity gaps are allowed only when explicitly documented in `docs/status.md` with follow-up scope.

When adding shared platform capabilities:

* define/update backend API contract first
* implement client consumption in desktop and web
* keep behavior aligned unless a documented temporary gap exists

For shared authentication-entry capabilities:

* keep login, forgot-password, and reset-password views behaviorally aligned across desktop and web
* keep configurable login branding settings (display name, login image, and related presentation settings) aligned across clients
* any temporary parity gap for authentication flows must be explicitly documented in `docs/status.md` with follow-up scope

---

## 14) Shared Administration UX Patterns

Administration UX should remain consistent and reusable across clients.

Preferred patterns:

* DataGrid list pages
* shared actions column behavior
* confirmation flows for destructive actions
* dialogs/drawers for detailed views
* consistent validation display and submission behavior

Prefer extending shared components/utilities for:

* DataGrid wrappers
* row actions renderers
* confirmation dialogs
* metadata/JSON viewers
* shared form helpers and validation adapters

Avoid one-off patterns unless there is a clear requirement gap.

---

## 15) DataGrid Contract

DataGrid usage must follow a common baseline:

* support horizontal and vertical scrolling
* keep action columns pinned right where supported
* degrade gracefully when pinning is unavailable
* honor global UI preferences when configured (for example density, radius, action pinning)
* do not silently ignore global preferences at table level

Grid containers must remain scroll-safe and must not clip:

* menus
* dialogs
* pinned columns
* scrollbars

---

## 16) RBAC UI / Backend Coupling

Frontend permission checks are UX-level only.
Backend authorization is mandatory and authoritative.

Rules:

* enforce permissions server-side for all protected actions
* use UI permission checks only to hide/disable affordances
* validate role/permission assignments server-side
* return typed validation errors for invalid identifiers
* surface validation errors clearly in clients

---

## 17) Audit UX Rules

Audit metadata can contain structured JSON and may be large.

Rules:

* show compact/truncated metadata previews in tables
* provide full metadata in a details dialog/drawer
* metadata viewer must support:
  * pretty print
  * scroll
  * copy
* avoid rendering full raw JSON directly in grid cells

---

## 18) Extension Rule

When extending the skeleton, prefer additive changes over risky core rewrites.

Prefer:

* adding modules in clear directories
* extending navigation via shared patterns/configuration
* adding backend modules without breaking existing API contracts

Treat these as core platform modules:

* authentication
* RBAC
* audit logging
* settings
* application shell

Changes to core modules should be intentional, minimal, and documented.

---

## 19) Agent Behavior

Agents must:

* read `docs/requirements.md` and `docs/status.md` before milestone work
* obey milestone sequencing and completion gates
* preserve backend/desktop/web contract alignment
* avoid breaking API changes without coordinated client updates
* keep changes incremental unless requirements demand broader refactor
* ensure required tests pass before claiming milestone completion

---

## 20) Prompt Scope Discipline

Milestone prompts must explicitly state impacted layers:

* backend
* desktop
* web
* or all three

Prompts should also explicitly require updates to:

* tests
* `docs/status.md`
* relevant documentation

Do not replace working behavior with temporary placeholders that reduce parity, quality, or architectural consistency unless requirements explicitly permit it.

---

## 21) New Module Contract

When introducing a new module, agents must treat it as a cross-layer feature unless `docs/requirements.md` explicitly limits scope.

A new module should be defined intentionally, not scattered across unrelated files.

### 21.1 Before implementing a new module
Agents must first confirm the module is defined in `docs/requirements.md` or explicitly requested by the user.

Before implementation, identify and document:

- module key/id
- display label
- base route/path
- navigation group
- permission keys
- backend API scope
- whether the module applies to:
  - backend only
  - desktop + backend
  - web + backend
  - backend + desktop + web

If the module changes platform structure, update `docs/status.md` accordingly.

### 21.2 Registry-first rule
New modules must be added through the project registries/configuration first, not by hardcoding scattered entries.

At minimum, review whether the module needs entries in:

- module registry
- navigation registry
- permission registry
- route definitions
- settings or feature flags
- dashboard shortcuts/widgets if applicable

Agents should prefer extending registry/config-driven structures over duplicating one-off wiring.
When laying initial registry foundations, use existing Administration pages/routes as first consumers where practical before introducing new modules.

### 21.2.1 Route organization rule
Route organization for modules must remain registry-aligned.

Rules:

- define canonical module route/base path in module registry metadata
- keep navigation and route wiring consistent with the same registry definitions
- avoid scattering module route constants across unrelated files without registry linkage

### 21.3 Permission naming rule
Permissions for new modules must follow a consistent naming convention.

Preferred pattern:

- `<module>.read`
- `<module>.write`
- `<module>.delete`
- `<module>.admin`

If more granular permissions are needed, they must remain predictable and documented.

Examples:

- `inventory.read`
- `inventory.write`
- `inventory.adjust`
- `reports.read`

Do not invent inconsistent naming styles across modules.

### 21.4 Cross-client parity for modules
If a new module is intended to exist in both desktop and web, agents must maintain parity unless `docs/status.md` records a temporary intentional gap.

Parity applies to:

- route existence
- navigation visibility
- permission checks
- major CRUD capabilities
- shared backend contract usage

### 21.5 Backend-first module rule
If a module requires new business data or actions:

- define/update backend API contract first
- then implement desktop/web consumers against that contract

Desktop must never access the database directly.

### 21.6 Shared UX patterns for modules
New modules must reuse shared platform patterns where applicable:

- app shell and navigation
- DataGrid wrappers
- actions column patterns
- dialogs/drawers
- notifications and error handling
- auth/session handling
- settings conventions

Avoid inventing module-specific UX patterns unless requirements justify them.

### 21.7 Module documentation rule
Every new module milestone must update `docs/status.md` with:

- module name
- layers affected
- routes/pages added
- permissions added
- backend endpoints added or changed
- tests added/run
- any known parity gaps

If the module introduces a reusable pattern, add a short `docs/notes/` entry when helpful.
If work is architecture/documentation-only (registry foundation), status must still include a planned/completed milestone entry with scope and follow-up implementation targets.

### 21.8 Keep the skeleton extensible
Agents must not implement modules in ways that make future extension harder.

Prefer:

- typed registries
- predictable route structure
- predictable permission naming
- reusable form/list/detail patterns
- minimal coupling between unrelated modules

Do not over-engineer plugin systems unless explicitly required.

---

## 22) Feature Flags / Module Enablement Contract

BasePro must support selectively enabling or disabling modules without deleting code.

This is intended for:
- downstream projects built from the skeleton
- phased rollouts
- deployments where some modules are not yet active
- hiding incomplete or not-licensed modules safely

### 22.1 Registry-first rule for module enablement
Module enablement must be configuration-driven, not hardcoded ad hoc in pages or handlers.

When a module can be toggled, review whether it needs entries in:
- module registry
- navigation registry
- permission registry
- feature-flag / enablement registry
- settings/config surfaces
- backend route guards or service guards

Do not scatter unrelated boolean checks across the codebase.

### 22.2 Flag scope must be explicit
Each flag must declare its intended scope.

Examples:
- backend-only
- desktop-only
- web-only
- cross-client UI visibility
- full-stack module enablement

Agents must document whether a flag controls:
- navigation visibility only
- route/page visibility
- API availability
- permissions exposure
- settings visibility
- all of the above

### 22.3 Safe disable behavior
Disabling a module must fail safely.

At minimum, review:
- hide navigation entries
- block direct route/page access
- prevent actions against disabled backend APIs where applicable
- avoid exposing disabled-module permissions in normal assignment flows unless intentionally retained for compatibility

Do not rely on hiding navigation alone.

### 22.4 Default behavior
Flags must have well-defined defaults.

Rules:
- default values must be documented
- defaults must be consistent across backend, desktop, and web
- if a module is marked enabled-by-default, that must be clear in the registry/config
- disabling a module must not break unrelated modules

### 22.5 Source-of-truth discipline
The source of truth for feature/module enablement must be documented.

Prefer a predictable approach such as:
- static defaults in registry/config
- backend-served effective enablement config for runtime awareness
- local client overrides only for development/testing if explicitly allowed

Agents must not invent multiple conflicting sources of truth.

### 22.6 Parity rule
If a flag controls a shared module, agents must maintain parity across:
- backend behavior
- desktop visibility/access
- web visibility/access

Any temporary mismatch must be recorded in `docs/status.md`.

### 22.7 New module rule
When creating a new module, agents must decide whether the module is:
- always enabled
- enabled by default but configurable
- experimental / disabled by default

That decision must be reflected in the registries and documented.

### 22.8 Testing rule
Feature/module enablement changes must include tests for:
- enabled state
- disabled state
- navigation behavior
- route guarding
- backend guarding if applicable

Do not consider a flag complete without testing both sides of the toggle.


---

## 23) Server-Driven Bootstrap + Settings Security Contract

This contract governs runtime bootstrap behavior and Settings authorization for the reusable skeleton.

### 23.1 Server-driven bootstrap contract
Agents must treat bootstrap runtime config as backend-served contract data.

Rules:
- define/update backend bootstrap contract first
- keep payload typed, additive, and version-aware
- keep desktop/web bootstrap consumption aligned to the same backend contract
- avoid introducing a large dynamic configuration platform for this baseline

### 23.2 Offline-aware bootstrap rule
Bootstrap consumption must support offline-aware startup behavior.

Rules:
- attempt fresh bootstrap from backend first
- when backend is unreachable, allow using last known valid cached bootstrap
- surface stale/offline-aware runtime state clearly to users
- do not fabricate new bootstrap values when neither backend nor cache is available

### 23.3 Settings security contract
Settings is a privileged administrative surface.

Rules:
- Settings route/page access intent in clients must require `admin` or `settings.write`
- backend authorization for Settings APIs is mandatory and authoritative
- UI checks are UX-only and must not be treated as security enforcement
- unauthorized settings actions must return typed authorization errors
- never log secret/sensitive settings values

### 23.4 Bootstrap + registry discipline
Bootstrap and registries must remain coherent.

Rules:
- registries remain canonical static definitions (modules/navigation/permissions/defaults)
- bootstrap runtime payload must be derived from registry-aligned keys plus allowed overrides
- avoid scattered hardcoded constants that diverge from registry/bootstrap contract
- document temporary parity gaps in `docs/status.md` with follow-up scope

### 23.5 Runtime config rule
Agents must distinguish runtime bootstrap config from persisted/static config.

Rules:
- treat bootstrap cache as non-secret runtime state only
- never cache tokens, passwords, or secret material in bootstrap cache
- keep cache schema versioned and invalidate incompatible payload versions
- runtime bootstrap state must not bypass backend authorization decisions

## END
