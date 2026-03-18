# Sukumad Overview

## 1. Purpose

Sukumad is a **data exchange middleware platform** designed primarily for integrating systems with **DHIS2**, but with a general architecture that can support other external systems.

The platform acts as an **intermediary layer** between upstream data sources and downstream systems such as DHIS2.

Key goals:

- Reliable data delivery
- Rate limiting and system protection
- Support for asynchronous submission workflows
- Observability and traceability of integration requests
- Retry and restart capabilities
- Support for multiple downstream servers

The system is designed to **protect DHIS2 from overload** while ensuring that integrations remain reliable and recoverable.

---

# 2. Primary Use Case

Typical workflow:

External System  
↓  
Sukumad API  
↓  
Queue / Delivery Workers  
↓  
DHIS2 API  
↓  
Async Task Polling  
↓  
Final Status Recorded

Sukumad handles:

- submission throttling
- asynchronous processing
- retry policies
- task status polling
- request tracking
- observability

---

# 3. Core Concepts

## 3.1 Servers

Servers represent **external systems or destinations**.

Examples:

- DHIS2 instance
- another API
- internal service endpoint

Each server may contain:

- base URL
- credentials
- authentication configuration
- rate limits
- activation status

Servers allow Sukumad to support **multiple integration targets simultaneously**.

---

## 3.2 Requests

Requests represent **integration submissions**.

A request is created when data must be delivered to a downstream system.

Requests contain:

- payload
- destination server
- metadata
- status
- timestamps
- correlation identifiers

Requests must support full traceability.

---

## 3.3 Deliveries

Deliveries represent **attempts to send a request to a server**.

A request may have multiple deliveries.

Example lifecycle:

Request  
→ Delivery attempt  
→ Failure  
→ Retry delivery  
→ Success

Delivery records allow:

- retry history
- auditability
- debugging

---

## 3.4 Async Tasks

Some systems (such as DHIS2) process submissions asynchronously.

Example:

Submit data → receive task ID → poll for completion.

Sukumad must support:

- recording async task IDs
- polling remote status endpoints
- resolving final success or failure states

---

## 3.5 Polling

Polling workers check remote systems for async task status.

Polling must support:

- configurable intervals
- backoff strategies
- termination conditions

---

## 3.6 Rate Limiting

Rate limiting protects downstream systems.

Rate limits may include:

- global request limits
- per-server limits
- burst limits
- worker concurrency limits

Rate limiting must prevent overload of systems like DHIS2.

---

## 3.7 Workers

Background workers handle:

- request processing
- delivery attempts
- polling for async tasks
- retries

Workers must support:

- graceful shutdown
- concurrency control
- visibility in observability dashboards

---

# 4. Observability

Sukumad requires strong observability features.

The system must support:

- request tracing
- correlation IDs
- request status history
- delivery history
- async task monitoring
- retry tracking
- failure analysis

Operators must be able to:

- inspect request details
- restart failed deliveries
- monitor system health

---

# 5. Configuration

Sukumad configuration may include:

- Redis or queue configuration
- worker counts
- rate limits
- polling intervals
- server definitions
- authentication credentials

Configuration must support environment-based overrides.

---

# 6. Data Model (Conceptual)

Primary entities:

Servers  
Requests  
Deliveries  
AsyncTasks  
PollEvents  
RateLimitPolicies

Relationships:

Server  
→ many Requests

Request  
→ many Deliveries

Delivery  
→ optional AsyncTask

AsyncTask  
→ many PollEvents

---

# 7. Integration Targets

Initial integration target:

DHIS2

DHIS2 requires:

- REST API submission
- asynchronous import tasks
- polling for completion status

Future integrations may include:

- other health systems
- financial systems
- messaging systems

The architecture must remain **system-agnostic**.

---

# 8. Key Architectural Requirements

Sukumad must support:

Reliable Delivery  
Retry Mechanisms  
Async Task Polling  
Rate Limiting  
Traceability  
Observability  
Multiple Target Servers

The system must scale to large integration workloads.

---

# 9. Integration with BasePro

In the new architecture, Sukumad will be implemented **inside BasePro**.

BasePro provides:

- authentication
- user management
- roles and permissions
- API infrastructure
- web UI shell
- desktop UI shell
- audit logging
- configuration management

Sukumad must **reuse these features** rather than implementing duplicates.

Specifically:

Do not implement:

- a separate login system
- a separate permissions system
- a separate UI shell

Instead Sukumad must add domain modules inside BasePro.

---

# 10. Sukumad Modules (Target Architecture)

The following backend modules are expected:

servers  
requests  
deliveries  
polling  
ratelimits  
dhis2  
observability

Each module should follow BasePro layering:

handlers  
services  
repositories

---

# 11. UI Modules

The web and desktop interfaces should include pages for:

Servers  
Requests  
Deliveries  
Jobs / Async Tasks  
Observability

These pages must integrate into the **existing BasePro navigation shell**.

---

# 12. Initial Navigation

Suggested navigation additions:

Dashboard

Integrations
- Servers

Exchange
- Requests
- Deliveries
- Jobs

Observability
- Request Logs
- System Health

Administration
- Users
- Audit
- Settings

---

# 13. Migration Strategy

Sukumad code should **not be copied blindly**.

Instead:

1. Map Sukumad components
2. Identify reusable logic
3. Adapt into BasePro modules
4. Replace Sukumad authentication and permissions with BasePro equivalents

A merge map must be produced before code migration.

---

# 14. Design Principles

Sukumad must emphasize:

Reliability  
Transparency  
Recoverability  
System Protection

The system must ensure that **integration failures are visible, recoverable, and auditable**.

---

# 15. Long-Term Vision

Sukumad should become a **general integration middleware platform** capable of:

- DHIS2 integration
- ETL pipelines
- messaging bridges
- external system orchestration

The architecture should remain modular and extensible.
