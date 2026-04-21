# Rapidex v2 Port Plan

## 1. Overview

Rapidex v2 extends the existing Sukumadpro system into a domain-aware integration platform that:

- Processes RapidPro flow results
- Maps them into DHIS2 aggregate payloads
- Submits them through Sukumadpro’s request exchange engine
- Introduces domain entities:
  - Organisation Units (health facilities)
  - Reporters
  - User Organisation Unit scoping

This work MUST reuse Sukumadpro’s architecture, request lifecycle, workers, UI patterns, and API conventions.

---

## 2. Architectural Principles

1. Do NOT rewrite Sukumadpro core request lifecycle.
2. Rapidex logic must be added as a domain layer.
3. Mapping engine must remain independent from transport logic.
4. Sukumadpro remains responsible for:
   - request execution
   - retries
   - delivery attempts
5. Rapidex is responsible for:
   - transforming data
   - creating exchange requests
6. Desktop and Web UIs MUST remain in sync.

---

## 3. Target Flow

Rapidex v2 MUST implement the following pipeline:

RapidPro webhook
→ Rapidex mapping engine
→ DHIS2 aggregate payload
→ create exchange_request
→ Sukumadpro workers
→ DHIS2

---

## 4. Domain Additions

### 4.1 Organisation Units

- Implement using Materialized Path
- Represent health system hierarchy
- Used as:
  - facilities
  - permission scope
  - reporter affiliation

Must support:
- CRUD
- tree traversal
- DHIS2 sync

---

### 4.2 Reporters

Represents field data submitters.

Fields:
- id
- name
- phone
- organisation_unit_id
- active

Rules:
- A reporter belongs to one organisation unit
- Used to resolve facility during submissions

---

### 4.3 User Organisation Unit Scope

Users MUST be scoped to organisation units.

Rules:
- A user can only:
  - view assigned org units + descendants
  - create/manage reporters within those org units

---

### 4.4 Rapidex Mapping Engine

- Port from Rapidex v1
- Must:
  - map RapidPro payload → DHIS2 aggregate payload
  - resolve orgUnit via reporter or mapping config
- Must NOT depend on Sukumad transport layer

---

### 4.5 Delivery Modes

Support:
- sukumad (default)
- direct (optional)
- asynq (optional legacy)

---

## 5. Milestones

### Milestone 1 — Discovery & Architecture Alignment
- Inspect Sukumadpro structure
- Identify integration points
- Produce:
  - rapidex-v2-architecture.md
  - confirm layering strategy

### Milestone 2 — Organisation Units Backend
- Add schema + migrations
- Implement materialized path logic
- Add repository/service/API
- Add tests

### Milestone 3 — DHIS2 Org Unit Sync
- Add sync service
- Fetch and persist DHIS2 org units
- Rebuild hierarchy
- Add tests

### Milestone 4 — Reporters Backend
- Add schema + migrations
- CRUD APIs
- Link to org units
- Add validation + tests

### Milestone 5 — User Org Unit Scoping
- Add user_org_units
- Enforce permissions
- Add tests

### Milestone 6 — Organisation Units UI
- Desktop + Web pages
- Tree + CRUD
- Follow existing UI patterns

### Milestone 7 — Reporters UI
- Desktop + Web pages
- DataGrid + forms
- Scoped org unit selection

### Milestone 8 — Rapidex Engine Port
- Port mapping logic
- Add config loading
- Add transformation pipeline
- Add tests

### Milestone 9 — Sukumad Integration
- Create exchange_requests from mapping output
- Reuse request lifecycle
- Add delivery mode config

### Milestone 10 — End-to-End Completion
- Add integration tests
- Validate full pipeline
- Finalize docs
- Ensure builds pass (desktop + web + backend)

---

## 6. Rules of Execution

For every milestone:

1. Read AGENTS.md
2. Read requirements.md
3. Read this document
4. Inspect existing implementation
5. Implement ONLY the current milestone
6. Keep repo buildable
7. Update docs/status
8. Maintain desktop + web parity

---

## 7. Success Criteria

Rapidex v2 is complete when:

- RapidPro flows are processed and mapped correctly
- Submissions are executed via Sukumad
- Organisation units are implemented and synced
- Reporters are functional and scoped
- Users are restricted by org units
- Desktop and web UIs support all features
- Tests and builds pass
