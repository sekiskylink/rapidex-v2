# Sukumad Addons Design

## 1. Context

Current Sukumad request processing is not queue-first.

Today the request path:

1. persists the `exchange_requests` row
2. creates the first `delivery_attempts` row immediately
3. calls `delivery.Service.SubmitDHIS2Delivery(...)` inline from `request.Service.CreateRequest(...)`
4. creates durable `async_tasks` state when the downstream system accepts async work
5. models retries as new `delivery_attempts` rows
6. includes worker definitions for send, poll, and retry, but only the poll path is materially wired today

That matters for addon design because any new gate must work in two execution modes:

- current mode: request intake and first submission happen in the same API flow
- future mode: workers become the primary execution boundary for initial send, retry send, and deferred replay

The design below keeps one rule for both modes: request and delivery state remain durable first, and eligibility to submit is evaluated immediately before outbound dispatch. The inline flow and future workers must call the same eligibility checks instead of implementing separate logic.

## 2. Refined Requirements

### 2.1 Submission Windows / Allowed Delivery Periods

#### Problem statement

Some downstream systems should only receive traffic during defined hours. The current inline request path would otherwise submit immediately after request creation, even when the destination should be closed.

#### Requirement

Support:

- one global submission window default
- per-destination submission window overrides
- per-destination precedence over the global default

Initial configuration supports:

- `start_hour`
- `end_hour`

The model must be extensible to later support:

- blocked dates
- maintenance windows
- allowed weekdays
- blackout periods

#### Operational behavior

- Window evaluation applies to the first send, retries, manual resubmissions, replays, and fan-out target deliveries.
- If the current time is outside the allowed window, Sukumad must not submit.
- The request and target delivery state must already exist durably before the gate is evaluated.
- The system must store the reason for deferral and, where possible, compute `next_eligible_at`.
- When the next eligible time arrives, the same delivery can be submitted by the future worker path without recreating the request.
- Overnight windows such as `22 -> 6` must be supported.
- `start_hour == end_hour` should be rejected as invalid rather than interpreted ambiguously.

#### Precedence rules

1. Per-destination override
2. Global default
3. No window configured means always allowed

#### Edge cases

- Requests created exactly at `start_hour` are allowed.
- Requests created exactly at `end_hour` are deferred.
- If no `next_eligible_at` can be computed because future rules become more complex later, the state still persists with a defer reason and can be reevaluated by the scheduler.
- Manual retry during a blocked window does not bypass the window; it only creates or reuses durable state.

#### Observability implications

Record structured events such as:

- `delivery.deferred.window`
- `delivery.eligible.window`
- `delivery.submission_released`

Recommended event data:

- request UID
- delivery UID
- destination server code
- matched policy source (`global` or `destination`)
- evaluated hour window
- computed `next_eligible_at`
- defer reason

#### Compatibility considerations

- Current inline flow: place the gate after the initial delivery row is created and before `SubmitDHIS2Delivery(...)` performs any outbound call.
- Future workers: the send worker and retry worker must call the same eligibility service before transitioning a delivery to `running`.
- Async poll traffic should normally ignore submission windows unless a later policy explicitly says poll traffic must also be windowed; initial scope should apply to outbound submissions, retries, replays, and manual resubmissions.

### 2.2 Max Retries

#### Problem statement

Retries are currently modeled as new delivery attempts. Without a retry ceiling, operators or future automation can create unlimited delivery history against a failing destination.

#### Requirement

Support:

- one global default max retry limit
- per-destination override

Per-destination takes precedence.

`max_retries` should mean additional attempts after the initial attempt. Example: `max_retries = 2` allows attempt numbers `1`, `2`, and `3`.

#### Operational behavior

- Retry limits are enforced when deciding whether a new retry attempt may be created.
- Existing failed attempts remain immutable history.
- When the limit is reached, the system must not create a new retry delivery.
- The original failed delivery remains the latest terminal attempt unless a future operator action changes policy and retries again.
- Fan-out targets each track retry limits independently.

#### Precedence rules

1. Per-destination override
2. Global default

#### Edge cases

- `max_retries = 0` means the initial attempt is the only allowed attempt.
- Manual retries and worker-driven retries are subject to the same limit.
- Policy changes do not retroactively delete existing attempts; they only affect future retry creation.

#### Observability implications

Record structured events such as:

- `delivery.retry.scheduled`
- `delivery.retry.rejected.max_retries`
- `delivery.retry.limit_applied`

Recommended event data:

- request UID
- delivery UID
- target server code
- current attempt number
- configured max retries
- remaining retries at evaluation time

#### Compatibility considerations

- Current inline/manual model: enforce in `delivery.Service.RetryDelivery(...)` before inserting the next attempt.
- Future worker model: any automatic retry scheduler must call the same policy evaluation before creating a retry attempt.

### 2.3 Expected Response Content Type Filtering

#### Problem statement

Downstream proxies and upstream failures often return HTML or other unexpected content. Treating those bodies as normal trusted metadata pollutes request history and can mislead operators.

#### Requirement

Support:

- one global default content-type policy
- per-destination override

If the actual content type is unexpected:

- do not store the raw body as trusted normal response metadata
- store a safe summary only
- record observability details

The policy should support simple allow lists first, for example JSON and XML families.

#### Operational behavior

- Apply the policy to synchronous submission responses and async poll responses.
- Determine content type from `Content-Type`, normalized without parameters where possible.
- If the response type is allowed, persist the response normally.
- If the response type is unexpected, persist:
  - HTTP status
  - normalized content type
  - a bounded safe summary
  - a flag that raw body was filtered
- Safe summary should include small, sanitized facts only, for example:
  - content type
  - body length
  - a short text snippet after HTML stripping or binary-safe truncation
  - whether the body looked like an HTML error page

#### Precedence rules

1. Per-destination override
2. Global default

#### Edge cases

- Missing `Content-Type` is treated as unexpected unless the destination policy explicitly allows unknown types.
- Very large bodies should not be persisted in full even when allowed; existing response size controls may still truncate.
- Binary responses are always summarized only in the initial implementation.
- A content-type mismatch does not itself decide success or failure; transport and integration logic still decide the delivery outcome.

#### Observability implications

Record structured events such as:

- `delivery.response.filtered_content_type`
- `async.poll.filtered_content_type`

Recommended event data:

- expected policy
- actual content type
- HTTP status
- filtered flag
- safe summary metadata

#### Compatibility considerations

- Current inline flow: apply filtering inside the DHIS2 client/service response mapping before persistence is updated by delivery or async services.
- Future workers: the same filter is reused because workers still submit and poll through the same integration service.

### 2.4 Retention / Purge Strategy

#### Problem statement

Request, delivery, async, and event history will grow continually. Ad hoc deletes would be unsafe, hard to observe, and hostile to future archival requirements.

#### Requirement

Provide a deliberate retention service/job for at least:

- `exchange_requests`
- `delivery_attempts`
- `async_tasks`
- `async_task_polls`
- `request_events`

The strategy must support:

- configurable retention age
- terminal-state-only purging
- referentially safe cleanup order
- future archival/export compatibility

#### Operational behavior

- Purge only records whose request lifecycle is terminal and older than the configured cutoff.
- Requests still `pending` or `processing`, or with any non-terminal target later, are never purged.
- Purge runs should process bounded batches and write summary events/audit entries.
- A dry-run mode should be supported for verification before destructive execution.
- Retention should be age-based and table-aware, but the top-level purge unit should remain the business request so related records are removed coherently.

#### Precedence rules

- The request terminal-state gate overrides age. Old non-terminal records are retained.
- Child-table retention must not delete records needed by a still-retained parent request.

#### Edge cases

- Async tasks in polling state block purge even if the original request is old.
- Failed requests retained for manual investigation remain until the configured age is reached.
- If archival/export is later enabled, purge must be able to mark an export checkpoint before delete.

#### Observability implications

Record structured events such as:

- `retention.run.started`
- `retention.run.completed`
- `retention.run.failed`
- `retention.request.purged`

Recommended event data:

- run UID
- batch size
- cutoff timestamp
- deleted counts by table
- dry-run flag
- failure reason

#### Compatibility considerations

- Current state: this is a background concern and can be introduced without changing request intake or delivery APIs.
- Future worker architecture: retention should run as its own scheduled worker definition and honor graceful shutdown and context cancellation.

### 2.5 Request Dependencies

#### Problem statement

Some requests should not be submitted until one or more prerequisite requests complete successfully. The current inline create flow would otherwise submit immediately after persistence.

#### Requirement

Support:

- one request depending on one or many other requests
- blocking submission until dependencies complete
- self-dependency prevention
- obvious cycle prevention
- explicit blocked-state behavior
- explicit handling when a dependency fails terminally

#### Operational behavior

- Dependency relationships are persisted when the request is created or updated.
- The business request row is still created immediately.
- The first delivery row is still created immediately to preserve current durable-attempt history.
- Before the first submission, the dependency gate evaluates whether all required dependencies are complete.
- If dependencies are unresolved, submission is blocked and no outbound dispatch occurs.
- If every dependency completes successfully, the delivery becomes eligible for submission.
- If any dependency fails terminally, the dependent request should move to terminal failure with reason `dependency_failed`, and its blocked delivery should never be sent.

#### Precedence rules

- Dependency blocks take precedence over submission-window eligibility.
- Once dependencies are satisfied, normal submission-window checks still apply.

#### Edge cases

- A request cannot depend on itself.
- Duplicate dependency links should be rejected.
- Obvious cycles must be rejected at write time by graph traversal over the existing dependency chain.
- A dependency on a missing or already purged request should be rejected.
- If a dependency completes before the dependent request is created, the dependent request can proceed immediately after normal gate evaluation.

#### Observability implications

Record structured events such as:

- `request.blocked.dependency`
- `request.unblocked.dependency`
- `request.failed.dependency`

Recommended event data:

- request UID
- blocked dependency count
- dependency request UIDs
- failed dependency UID
- gate evaluation result

#### Compatibility considerations

- Current inline flow: the dependency gate must sit between `CreatePendingDelivery(...)` and `SubmitDHIS2Delivery(...)` inside `request.Service.CreateRequest(...)`.
- Future worker flow: the same dependency evaluator must be called by the send worker before dispatching any pending delivery.

### 2.6 Multi-Destination / Fan-Out Delivery

#### Problem statement

Sukumad currently models one request against one `destination_server_id`. Real workflows may need one business request delivered to multiple integration servers, each with independent retries, async tasks, and observability.

#### Requirement

Support one business request being sent to multiple integration servers using a normalized relational design:

- one request record
- destination-specific target records
- destination-specific delivery attempts
- destination-specific async handling
- request-level roll-up across targets

Do not use a single JSONB response blob as the primary model.

#### Operational behavior

- A request can expand to one or more targets.
- Each target has its own lifecycle, eligibility checks, retries, async task linkage, and observability events.
- Each target creates its own initial delivery attempt.
- Request-level status is a roll-up of target outcomes.
- Initial implementation can treat every target as required.
- Later support can mark targets as required or optional without redesigning the core relational model.

#### Precedence rules

- Eligibility, retry, and content-type rules are evaluated per target destination.
- Request-level roll-up is derived from target state, not from a single destination field.

#### Edge cases

- Duplicate target server assignments on the same request should be rejected unless a future use case explicitly permits repeated targets with different roles.
- Partial success across multiple required targets does not complete the request.
- Optional target failures should not fail the request once optional semantics are introduced later.

#### Observability implications

Record structured events such as:

- `request.target.created`
- `request.target.completed`
- `request.target.failed`
- `request.rollup.updated`

Recommended event data:

- request UID
- target UID
- server code
- target role (`primary`, `cc`, later `optional`)
- target status
- request roll-up status

#### Compatibility considerations

- Current inline flow: create all target rows and their first delivery attempts during request creation, then evaluate/send each eligible target without bypassing durability.
- Future worker flow: targets become the natural send-unit for worker scheduling.
- Single-destination requests remain supported as the degenerate case of one target.

## 3. Proposed Architecture

### 3.1 Shared rule: durable state first, gate second, dispatch third

All addons should follow the same orchestration sequence:

1. persist request and target state
2. persist the first delivery attempt for each target
3. evaluate dependency and submission-window eligibility
4. dispatch only if eligible
5. record defer/block/failure reasons as structured state and events

This preserves current behavior where history exists before network calls while keeping the same contract for future workers.

### 3.2 Module placement

The design stays inside existing BasePro-consistent Sukumad modules:

- `backend/internal/sukumad/request`
  - request creation
  - dependency orchestration
  - request-level roll-up
- `backend/internal/sukumad/delivery`
  - delivery eligibility evaluation
  - retry ceiling enforcement
  - submission deferral metadata
- `backend/internal/sukumad/async`
  - content-type filtering for poll responses
  - request/target reconciliation from async terminal outcomes
- `backend/internal/sukumad/server`
  - destination-scoped policy source for per-server overrides
- `backend/internal/sukumad/observability`
  - event readout and retention-run visibility
- `backend/internal/sukumad/worker`
  - future execution path for due sends, retries, dependency-release rechecks, and retention

If a shared evaluator is needed during implementation, it should remain under `backend/internal/sukumad` and be consumed by request, delivery, and worker services rather than creating separate orchestration paths in handlers.

### 3.3 Submission-window architecture

- Add a delivery eligibility evaluator that resolves the active destination policy from runtime config.
- `request.Service.CreateRequest(...)` and future send/retry workers call the evaluator before dispatch.
- If blocked by window, the delivery remains non-running and stores:
  - `submission_hold_reason = window_closed`
  - `next_eligible_at`
  - matched policy source
- A future due-send worker should select non-terminal deliveries whose `next_eligible_at <= now`.

### 3.4 Max-retry architecture

- Retry policy resolution belongs in the delivery service because retries are represented as new delivery attempts.
- `RetryDelivery(...)` must count existing attempts for the relevant target and reject creation when the configured max is exhausted.
- Future automatic retry creation must reuse the same service method or policy helper.

### 3.5 Content-type filter architecture

- Content-type filtering belongs at the integration boundary, before response persistence.
- `dhis2.Service.Submit(...)` and `dhis2.Service.Poll(...)` should return normalized response metadata that includes:
  - actual content type
  - whether the body was filtered
  - safe summary
- Delivery and async services persist that normalized metadata without trusting filtered bodies as canonical response content.

### 3.6 Retention architecture

- Implement retention as a dedicated Sukumad service/job, not ad hoc deletes inside repositories.
- The service should:
  - resolve a cutoff from config
  - list purge-eligible requests in batches
  - optionally export later
  - delete related data in a safe order inside transactions where practical
  - emit run-level observability events
- The worker manager can later host a `retention` definition alongside send, poll, and retry.

### 3.7 Dependency architecture

- Add dependency persistence and evaluation in the request module because dependency state is business-request scoped.
- The dependency gate should be checked immediately before the first inline submit and before future worker-driven sends.
- Dependency completion and failure should trigger request-level reevaluation:
  - all succeeded -> release blocked delivery to normal window evaluation
  - any failed -> fail the dependent request without dispatching

### 3.8 Fan-out architecture

- Introduce `request_targets` as the normalized expansion layer between requests and delivery attempts.
- Request creation becomes:
  - create `exchange_requests`
  - create one or more `request_targets`
  - create one initial delivery attempt per target
  - evaluate/send each target independently
- Async tasks link to delivery attempts as they do today.
- Request roll-up becomes derived from target statuses instead of a single destination field.

## 4. Proposed Schema / Model Changes

### 4.1 Config additions

Suggested additive runtime config:

```yaml
sukumad:
  submission_window:
    default:
      start_hour: 6
      end_hour: 22
    destinations:
      dhis2-ug:
        start_hour: 7
        end_hour: 19
  retries:
    default:
      max_retries: 2
    destinations:
      dhis2-ug:
        max_retries: 1
  response_policy:
    default:
      allowed_content_types:
        - application/json
        - application/*+json
    destinations:
      dhis2-ug:
        allowed_content_types:
          - application/json
          - application/*+json
  retention:
    enabled: true
    dry_run: true
    batch_size: 100
    requests:
      max_age_days: 90
    deliveries:
      max_age_days: 90
    async_tasks:
      max_age_days: 90
    async_task_polls:
      max_age_days: 30
    request_events:
      max_age_days: 30
```

The retention values can later collapse into request-centric classes if separate table ages prove unnecessary.

### 4.2 `exchange_requests`

Keep current fields for compatibility and add only additive state:

- `terminal_reason` nullable text
- `blocked_reason` nullable text
- `next_eligible_at` nullable timestamptz

For fan-out compatibility:

- keep `destination_server_id` during transition as the primary-target convenience field
- derive it from the primary `request_targets` row for new records later

Status guidance:

- keep `pending`, `processing`, `completed`, `failed`
- use `pending` plus explicit block/defer reason for initial introduction
- avoid introducing a new request status unless later UI needs prove it necessary

### 4.3 `request_targets`

Add or formalize:

- `id`
- `uid`
- `request_id`
- `server_id`
- `target_role` text (`primary`, `cc`)
- `is_required` boolean default `true`
- `status` text (`pending`, `processing`, `completed`, `failed`)
- `blocked_reason` nullable text
- `next_eligible_at` nullable timestamptz
- `terminal_reason` nullable text
- `created_at`
- `updated_at`

Constraints:

- unique `(request_id, server_id)` for initial implementation

### 4.4 `delivery_attempts`

Evolve toward target-specific attempts:

- add `request_target_id` nullable first, then backfill, then make not null in a later migration
- keep `server_id` as denormalized read/query support if useful
- replace uniqueness from `(request_id, attempt_number)` to `(request_target_id, attempt_number)` once fan-out lands

Add additive columns:

- `submission_hold_reason` nullable text
- `next_eligible_at` nullable timestamptz
- `response_content_type` nullable text
- `response_body_trusted` boolean not null default `true`
- `response_body_summary` text not null default `''`

Roll-up behavior:

- delivery remains the execution record
- target roll-up is computed from latest attempt state
- request roll-up is computed from target states

### 4.5 `async_tasks` and `async_task_polls`

Add additive response-policy fields where needed:

- `response_content_type` nullable text
- `response_body_trusted` boolean not null default `true`
- `response_body_summary` text not null default `''`

Async tasks continue linking to `delivery_attempts`; no separate fan-out async model is needed.

### 4.6 `request_dependencies`

Add a new relational table:

- `id`
- `request_id` FK to `exchange_requests`
- `depends_on_request_id` FK to `exchange_requests`
- `created_at`
- unique `(request_id, depends_on_request_id)`

Rules:

- reject `request_id == depends_on_request_id`
- enforce cycle prevention in the service layer before insert

### 4.7 `request_events`

No major schema redesign is required, but event payloads should consistently carry:

- `targetUid`
- `serverCode`
- `policySource`
- `blockedReason`
- `nextEligibleAt`
- `responseContentType`
- `responseBodyTrusted`

### 4.8 Purge order

Recommended logical cleanup order for a request-centered purge:

1. `async_task_polls`
2. `async_tasks`
3. `request_events` scoped to the request
4. `delivery_attempts`
5. `request_targets`
6. `request_dependencies` rows where the request is parent or dependency
7. `exchange_requests`

Where FK cascades already exist, implementation can rely on them, but the retention service should still think and report in this order.

## 5. Test Strategy

### 5.1 Submission windows

Must test:

- global window allowed and denied cases
- per-destination override precedence
- boundary hours and overnight windows
- durable defer state written before no-submit return
- `next_eligible_at` computation
- retry/manual resubmit/fan-out target reuse of the same gate
- inline create path does not dispatch when window is closed
- future worker selection of due deferred deliveries

### 5.2 Max retries

Must test:

- global default application
- per-destination override precedence
- `max_retries = 0`
- manual retry rejection after limit
- worker-driven retry creation rejection after limit
- retry counting per fan-out target rather than per whole request

### 5.3 Response content type filtering

Must test:

- allowed JSON/XML content types persist normally
- unexpected HTML response stores only safe summary
- missing content type handling
- very large response summary truncation
- async poll response filtering
- observability events include expected vs actual content type

### 5.4 Retention / purge

Must test:

- only terminal requests are purge candidates
- non-terminal requests block purge
- dry-run reports without deleting
- batched delete counts
- safe cleanup order
- request-linked events and async records are removed with the request
- archival/export hook can be inserted before delete later
- worker cancellation stops a long-running purge safely

### 5.5 Request dependencies

Must test:

- single and multiple dependencies
- self-dependency rejection
- duplicate dependency rejection
- obvious cycle rejection
- blocked request remains durable but unsubmitted
- dependent request releases when prerequisites complete
- dependent request fails terminally when a prerequisite fails
- inline create path places the gate before outbound submit

### 5.6 Fan-out delivery

Must test:

- one request expands to multiple targets
- one initial delivery per target
- per-target retries and async tasks remain isolated
- request roll-up across required targets
- partial success behavior
- observability timelines contain target identifiers
- backward-compatible single-target requests still behave correctly

## 6. Rollout / Compatibility Notes

Recommended rollout order:

1. Add submission-window, max-retry, and content-type policy evaluation as additive behavior on the existing single-destination model.
2. Add retention as an independent background service/job.
3. Add request dependencies using the existing single-destination flow, with the gate between delivery creation and dispatch.
4. Introduce `request_targets` and migrate delivery attempts to target-scoped execution for fan-out.

Compatibility recommendations:

- Keep current request statuses initially and represent new blocked/deferred causes with additive fields plus events.
- Reuse one eligibility evaluator from both inline orchestration and future workers.
- Keep current `destination_server_id` readable during the fan-out transition so existing APIs and UIs do not break immediately.
- Make all schema changes additive first, backfill, then tighten constraints in later milestones.
- Do not enable destructive retention by default; start with `dry_run`.
- Treat optional targets as a later additive feature by storing `is_required` from the start.

## 7. Major Recommendations

- Use `request_targets` as the core expansion seam for fan-out instead of trying to overload `exchange_requests` or store multi-server outcomes in JSON.
- Keep request roll-up coarse and stable; add reason fields rather than proliferating statuses immediately.
- Put dependency and window gating before outbound dispatch but after durable request and delivery persistence.
- Enforce retry ceilings at retry-attempt creation time, because retries are new rows in the current model.
- Filter unexpected response content at the integration boundary so HTML/proxy failures do not become trusted delivery metadata.
- Implement retention as a dedicated request-centered service/job with dry-run support and observable batch reporting.

## 8. Implementation Status

Status as of 2026-03-13:

- Landed the additive schema and service wiring for submission windows, retry limits, response content-type filtering, request dependencies, and fan-out request targets.
- Landed persisted SQL hydration for request targets and dependencies across request list/detail reads, replacing the earlier minimal single-target projection.
- Landed request roll-up derived from target state so fan-out requests now reconcile `pending`, `blocked`, `processing`, `completed`, and `failed` from the target set rather than a single destination field.
- Landed a dedicated request-centered retention service and worker definition with dry-run support, observable retention events, audit logging, safe purge ordering, and terminal-state eligibility checks.
- Landed minimal web and desktop visibility for blocked/deferred/fan-out state by exposing target counts, blocked reasons, deferred timestamps, target lists, and dependency lists on the request pages.
- Added focused addon test coverage for:
  - request SQL hydration of targets and dependencies
  - request roll-up across fan-out target combinations
  - retention candidate selection, dry-run, purge ordering, and cancellation-safe execution
  - response-content filtering persistence for delivery and async poll reads
  - web and desktop request-page visibility for blocked/deferred/fan-out details

Remaining follow-ups:

- Request roll-up remains intentionally coarse; optional-target semantics and richer partial-success presentation can be layered later without changing the target model.
- The retention worker definition is now bootstrapped and config-driven, but broader scheduler orchestration can still evolve independently of this milestone.
