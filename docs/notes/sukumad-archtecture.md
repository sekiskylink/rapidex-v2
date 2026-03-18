# Sukumad Architecture

## 1. Overview

Sukumad is an integration middleware platform implemented inside the BasePro application architecture.

Its role is to provide **controlled, observable, and reliable data exchange** between upstream systems and downstream systems such as DHIS2.

Sukumad provides:

- request intake
- delivery orchestration
- rate limiting
- asynchronous workflow support
- retry mechanisms
- observability

Sukumad must operate within the **BasePro platform architecture**, reusing its infrastructure:

- authentication
- role-based access control
- API infrastructure
- configuration management
- audit logging
- web UI shell
- desktop UI shell

Sukumad must **not introduce duplicate implementations** of these platform capabilities.

---

# 2. High-Level Architecture

The architecture consists of four main layers:

Client Layer  
Application Layer  
Worker Layer  
Integration Layer
Clients
(Web UI / Desktop UI / External APIs)

↓

Application Layer
(API handlers + services)

↓

Worker Layer
(Delivery + Polling + Retry)

↓

Integration Layer
(DHIS2 and other external systems)


---

# 3. BasePro Platform Layer

BasePro provides the foundational platform services used by Sukumad.

These include:

Authentication  
User Management  
Role-Based Access Control  
Audit Logging  
Configuration Management  
HTTP Middleware  
API Routing  
Desktop/Web Application Shell

Sukumad must integrate into these platform capabilities rather than reimplement them.

---

# 4. Backend Architecture

Backend modules follow BasePro conventions.

Each module uses a layered structure:

Handlers  
Services  
Repositories
Handler → Service → Repository → Database


Responsibilities:

Handlers  
HTTP request handling and validation

Services  
Business logic

Repositories  
Database access

---

# 5. Backend Module Structure

The following modules are expected inside `backend/internal`.

## servers

Manages downstream system definitions.

Responsibilities:

- store server configuration
- manage server activation
- store credentials
- define rate limit policies

Entities:

Server  
ServerCredential  
ServerRateLimit

---

## requests

Represents integration requests entering the system.

Responsibilities:

- create request records
- track request status
- associate payloads and metadata

Entities:

ExchangeRequest

Key fields:

- request_id
- server_id
- payload
- status
- correlation_id
- created_at

---

## deliveries

Represents attempts to send requests to servers.

Responsibilities:

- create delivery attempts
- store response status
- track retries
- record response payloads

Entities:

DeliveryAttempt

Key fields:

- request_id
- attempt_number
- response_status
- error_message
- created_at

---

## polling

Handles asynchronous status polling for external systems.

Responsibilities:

- schedule polling events
- call remote status endpoints
- update request state

Entities:

AsyncTask  
PollEvent

---

## ratelimits

Controls outbound traffic to downstream systems.

Responsibilities:

- enforce global rate limits
- enforce per-server limits
- manage burst capacity
- control worker concurrency

Mechanisms may include:

- token buckets
- worker queues
- concurrency gates

---

## dhis2

Implements DHIS2-specific integration logic.

Responsibilities:

- submission endpoints
- async import monitoring
- response parsing
- error interpretation

DHIS2 integration must support:

- async import tasks
- polling of task endpoints
- final state reconciliation

---

## observability

Provides traceability and operational visibility.

Responsibilities:

- request tracing
- job tracking
- system health monitoring

Entities:

RequestLog  
WorkerRun  
FailureRecord

---

# 6. Worker Architecture

Workers are responsible for asynchronous processing.

Worker types:

Delivery Workers  
Polling Workers  
Retry Workers

Workers must support:

- configurable concurrency
- graceful shutdown
- visibility in observability
- configurable backoff

Workers should run inside the backend process or as separate worker processes depending on deployment.

---

# 7. Request Lifecycle

The lifecycle of an integration request is:

Request Created  
↓  
Delivery Attempt  
↓  
External System Processing  
↓  
Async Task Created (optional)  
↓  
Polling  
↓  
Final Status

Possible final states:

Success  
Failed  
Abandoned

---

# 8. Rate Limiting Architecture

Rate limiting ensures downstream systems are protected.

Rate limits may include:

Global limits  
Per-server limits  
Burst limits

Implementation options include:

Token bucket  
Leaky bucket  
Worker concurrency limits

Rate limiting must apply before delivery attempts.

---

# 9. Observability Architecture

Observability ensures operational transparency.

Capabilities must include:

Request traceability  
Delivery history  
Async task monitoring  
Failure diagnostics

Operators must be able to:

- inspect request history
- restart failed deliveries
- identify system bottlenecks

---

# 10. Database Architecture

The system database stores operational state.

Primary entities:

Servers  
ExchangeRequests  
DeliveryAttempts  
AsyncTasks  
PollEvents  
RateLimitPolicies

Relationships:

Server  
→ ExchangeRequests

ExchangeRequest  
→ DeliveryAttempts

DeliveryAttempt  
→ AsyncTask

AsyncTask  
→ PollEvents

---

# 11. UI Architecture

Sukumad user interfaces must integrate into the existing BasePro application shell.

Both Web and Desktop must reuse:

AppShell  
Navigation Drawer  
Route Guards  
Permission Checks

Sukumad must not introduce an independent UI framework.

---

# 12. UI Modules

Expected user interface pages:

Servers  
Requests  
Deliveries  
Jobs / Async Tasks  
Observability

Pages should follow BasePro conventions:

- MUI components
- DataGrid list views
- detail panels
- consistent navigation

---

# 13. Permissions Architecture

Permissions must integrate with BasePro RBAC.

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

Navigation visibility must follow permission checks.

---

# 14. Configuration Architecture

Configuration must follow BasePro configuration patterns.

Configuration may include:

worker concurrency  
rate limits  
poll intervals  
retry policies  
server definitions

Configuration must support environment overrides.

---

# 15. Deployment Architecture

The system may run in several deployment modes.

Single-process mode:

API server + workers

Multi-process mode:

API server  
Delivery workers  
Polling workers

The architecture must allow both modes.

---

# 16. Integration Targets

Initial supported system:

DHIS2

Future integrations may include:

other health systems  
financial systems  
messaging platforms

Architecture must remain system-agnostic.

---

# 17. Error Handling

Error handling must include:

structured errors  
retryable vs terminal failures  
error classification

Errors must never expose secrets.

---

# 18. Logging

Logging must include:

request IDs  
server IDs  
delivery attempts  
worker actions

Sensitive credentials must never be logged.

---

# 19. Security

Security requirements include:

secure credential storage  
audit logging  
permission enforcement  
API authentication via BasePro

Sukumad must not bypass BasePro security mechanisms.

---

# 20. Architectural Principles

Sukumad must prioritize:

Reliability  
Traceability  
Recoverability  
System Protection

The system must ensure that integration workloads are controlled, observable, and recoverable.
