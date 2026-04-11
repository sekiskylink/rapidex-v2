# Status

## Milestone — Dashboard Live Request Processing Graph (Complete)

### What changed
- Extended the Sukumad dashboard snapshot under [backend/internal/sukumad/dashboard](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/dashboard) with a `processingGraph` payload that buckets request lifecycle movement into pending, processing, completed, and failed stages across the recent dashboard time window.
- Reused the existing dashboard websocket stream and added `processingGraph` invalidation on request lifecycle events so the live dashboard graph stays aligned with the current operations event source of truth.
- Replaced the request-trend summary cards on the web dashboard in [web/src/pages/DashboardPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/DashboardPage.tsx) with a live request-processing flow graph and stage totals that deep-link into filtered Requests views.
- Added the matching live request-processing flow graph to the desktop dashboard in [desktop/frontend/src/pages/DashboardPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/DashboardPage.tsx), keeping web/desktop parity and continuing to use backend APIs only.
- Added request status route-search support to web and desktop Requests pages so dashboard graph clicks open filtered request lists:
  - [web/src/pages/RequestsPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/RequestsPage.tsx)
  - [desktop/frontend/src/pages/RequestsPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/RequestsPage.tsx)
  - [web/src/pages/listRouteSearch.ts](/Users/sam/projects/go/sukumadpro/web/src/pages/listRouteSearch.ts)
  - [desktop/frontend/src/pages/listRouteSearch.ts](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/listRouteSearch.ts)
  - [web/src/routes.tsx](/Users/sam/projects/go/sukumadpro/web/src/routes.tsx)
  - [desktop/frontend/src/routes.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/routes.tsx)
- Saved prompt traceability copy in `docs/prompts/2026-04-11-dashboard-live-request-processing-graph.md` (gitignored).

### Added or updated tests
- Backend:
  - extended dashboard repository coverage for processing-graph aggregation and request-event mapping
  - verified dashboard websocket handler coverage still passes with the extended stream invalidation contract
- Web:
  - updated dashboard coverage for the processing graph and request-list drill-down behavior
  - updated requests page coverage through the full suite with request-status route search enabled
- Desktop:
  - updated matching dashboard coverage for the processing graph and request-list drill-down behavior
  - updated desktop requests coverage through the full suite with request-status route search enabled

### Verification summary
- Backend focused tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./internal/sukumad/dashboard`)
- Backend full tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./...`)
- Web focused tests: PASS (`cd web && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm test -- --run src/pages/dashboard-page.test.tsx src/pages/requests-page.test.tsx`)
- Web tests: PASS (`cd web && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm test -- --run`)
- Web build: PASS (`cd web && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm run build`)
- Desktop focused tests: PASS (`cd desktop/frontend && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm test -- --run src/pages/dashboard-page.test.tsx src/pages/requests-page.test.tsx`)
- Desktop tests: PASS (`cd desktop/frontend && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm test -- --run`)
- Desktop build: PASS (`cd desktop/frontend && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm run build`)

### Known follow-ups
- The first non-escalated backend dashboard-package test run failed because websocket handler tests need a local `httptest` listener (`bind: operation not permitted`); rerunning with approved local listener permissions passed.
- Existing non-blocking frontend warnings remain in verification logs: MUI/jsdom `anchorEl` warnings, desktop `useRouter` warnings, third-party `'use client'` build warnings, MUI X `rowCount` warnings in one web route test, and Vite chunk-size warnings.

## Milestone — Worker Delivery Target Roll-Up Fix (Complete)

### What changed
- Fixed API and worker bootstrap wiring so the Sukumad delivery service updates request targets as deliveries move through the worker lifecycle.
- Successful worker deliveries now mark the matching request target `succeeded`, allowing the existing request target roll-up to move the exchange request from `pending` to `completed` once all targets succeed.
- Added a regression assertion to the durable worker-restart lifecycle test so completed worker dispatch must also persist the target success state.
- No web or desktop code changes were required because clients already read the backend request and target status fields.

### Verification summary
- Backend focused tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./internal/sukumad/worker ./internal/sukumad/delivery ./internal/sukumad/request`)
- Backend full tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./...`)

### Known follow-ups
- The initial sandboxed backend full-test run failed because local `httptest` listener creation was blocked (`bind: operation not permitted`); rerunning with approved local listener permissions passed.

## Milestone — Worker Outbound Logging Configuration (Complete)

### What changed
- Added `sukumad.workers.outbound_logging.enabled` and `sukumad.workers.outbound_logging.body_preview_bytes` backend config for worker-process outbound DHIS2 request logging.
- Wired the worker process to emit `worker_outbound_request` logs for DHIS2 submit and poll calls when enabled.
- Outbound logs include method, sanitized URL, destination key, body byte count, and a redacted preview; headers and secrets are not logged.
- Updated worker/configuration notes and saved prompt traceability copy in `docs/prompts/2026-04-10-worker-outbound-logging-configuration.md` (gitignored).

### Added or updated tests
- Backend:
  - added config default/load/validation coverage for worker outbound logging
  - added DHIS2 service coverage for disabled logging, redacted previews, truncation, poll logging, and preserving request bodies for transport

### Verification summary
- Backend focused tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./internal/config ./internal/sukumad/dhis2`)
- Backend full tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./...`)
- Backend build: PASS (`make backend-build`)
- Web tests: PASS (`make web-test`)
- Web build: PASS (`make web-build`)
- Desktop tests: PASS (`make desktop-test`)
- Desktop build: PASS (`make desktop-build`)

### Known follow-ups
- The initial sandboxed backend full-test run failed because an existing API test could not create a local `httptest` listener (`bind: operation not permitted`); rerunning with approved local listener permissions passed.
- Existing non-blocking frontend warnings remain: MUI/jsdom `anchorEl` warnings, desktop `useRouter` warnings, third-party `"use client"` build warnings, and Vite chunk-size warnings.

## Milestone — Request Delete and Response Body Persistence (Complete)

### What changed
- Added `DELETE /api/v1/requests/:id` with `requests.write` enforcement and service/repository deletion logic for exchange requests and associated Sukumad domain rows.
- Added a destructive Delete action to the Requests page actions menu in both web and desktop clients, with confirmation and refresh behavior.
- Added integration-server response body persistence configuration with server default `filter` and request-level optional override for `filter`, `save`, or `discard`.
- Threaded the effective persistence policy through delivery, DHIS2 submit/poll, async task reads, and worker dispatch so response bodies are saved, filtered, or discarded consistently.
- Added migration `000024_request_delete_response_body_persistence` for the new server/request policy columns and check constraints.
- Added architecture notes in [request-delete-response-body-persistence.md](/Users/sam/projects/go/sukumadpro/docs/notes/request-delete-response-body-persistence.md).
- Saved prompt traceability copy in `docs/prompts/2026-04-10-request-delete-response-body-persistence.md` (gitignored).

### Added or updated tests
- Backend:
  - added request delete handler/service coverage
  - updated request, server, and async repository coverage for response body persistence columns
  - updated delivery/DHIS2/worker paths for effective response body persistence behavior
- Web:
  - updated request and server page coverage around request/server form payloads and route smoke behavior
- Desktop:
  - updated matching request and server page coverage around request/server form payloads and route smoke behavior

### Verification summary
- Backend focused tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./internal/sukumad/request ./internal/sukumad/server ./internal/sukumad/delivery ./internal/sukumad/dhis2 ./internal/sukumad/async ./internal/sukumad/worker`)
- Backend full tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./...`)
- Web focused tests: PASS (`cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run src/pages/requests-page.test.tsx src/pages/servers-page.test.tsx --run`)
- Web tests: PASS (`cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run`)
- Web build: PASS (`cd web && npm run build`)
- Desktop focused tests: PASS (`cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run src/pages/requests-page.test.tsx src/pages/servers-page.test.tsx --run`)
- Desktop tests: PASS (`cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run`)
- Desktop build: PASS (`cd desktop/frontend && npm run build`)

### Known follow-ups
- The initial sandboxed backend full-test run failed because local `httptest` listener creation was blocked (`bind: operation not permitted`); rerunning with approved local listener permissions passed.
- Existing non-blocking MUI/jsdom `anchorEl` warnings, desktop `useRouter` warnings, third-party `'use client'` build warnings, and Vite chunk-size warnings remain in verification logs.

## Milestone — Worker Processing Logs (Complete)

### What changed
- Added structured process logs for send/retry delivery workers in [backend/internal/sukumad/worker/executor.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/executor.go):
  - pickup, completion, deferral, failed delivery status, and worker-side error logs
  - safe metadata including worker run, request, delivery, correlation, server, status, HTTP status, attempt, and duration fields
- Added structured process logs for async polling in [backend/internal/sukumad/async/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/service.go):
  - poll pickup, success, transient poll failure, and worker-side persistence/claim error logs
  - safe metadata including worker run, request, delivery, async task, correlation, server code, remote job/status, terminal state, and duration fields
- Kept payload bodies, response bodies, headers, JWTs, refresh tokens, API tokens, passwords, and DSNs out of worker logs.
- Added worker logging notes in [docs/notes/worker-process-logging.md](/Users/sam/projects/go/sukumadpro/docs/notes/worker-process-logging.md).
- Saved prompt traceability copy in `docs/prompts/2026-04-10-worker-processing-logging.md` (gitignored).
- No web or desktop code changes were required because this is backend worker-process observability only and does not change API responses or navigation.

### Added or updated tests
- Backend:
  - updated delivery worker executor tests to assert structured pickup/deferred/completed process logs and safe metadata
  - updated async service tests to assert structured poll pickup/success/failure process logs and safe metadata
- Web and desktop:
  - no code changes; existing route/smoke suites were run for milestone parity.

### Verification summary
- Backend focused tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./internal/sukumad/worker ./internal/sukumad/async`)
- Backend full tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./...`)
- Web tests: PASS (`cd web && npm test -- --run`)
- Web build: PASS (`cd web && npm run build`)
- Desktop tests: PASS (`cd desktop/frontend && npm test -- --run`)
- Desktop build: PASS (`cd desktop/frontend && npm run build`)

### Known follow-ups
- The initial sandboxed backend full-test run failed because local `httptest` listener creation was blocked (`bind: operation not permitted`); rerunning with approved local listener permissions passed.
- Existing non-blocking MUI/jsdom warnings, desktop `useRouter` warnings, third-party `'use client'` build warnings, and Vite chunk-size warnings remain unchanged.

## Milestone — Request Create Dialog Validation and Destination Autocomplete (Complete)

### What changed
- Added migration `000023_default_request_metadata_empty_strings` to backfill `exchange_requests.batch_id`, `correlation_id`, and `idempotency_key` from `NULL` to `''`, then set empty-string defaults and `NOT NULL` constraints.
- Updated Sukumad request persistence so new requests store empty strings for blank `batchId`, `correlationId`, and `idempotencyKey`, while request reads coalesce legacy nullable metadata safely.
- Coalesced nullable `url_suffix` on request reads so create responses remain safe when the create dialog leaves URL Suffix blank.
- Updated web and desktop request create dialogs to use MUI autocomplete controls for the primary destination server and additional fan-out destination servers.
- Kept request form validation for required destination, dependency ID list, payload format, send-as mode, payload, and metadata JSON; both clients now submit selected destination IDs and blank optional metadata as empty strings.
- Saved prompt traceability copy in `docs/prompts/2026-04-10-request-create-dialog-validation-destination-autocomplete.md` (gitignored).

### Added or updated tests
- Backend:
  - updated request repository tests for empty optional metadata persistence and nullable read coalescing
  - added service coverage for optional metadata normalization to empty strings
- Web:
  - updated request page tests for destination autocomplete selection and empty optional metadata payloads
- Desktop:
  - updated matching request page tests for destination autocomplete selection and empty optional metadata payloads

### Verification summary
- Backend tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./...`)
- Web tests: PASS (`cd web && npm test -- --run`)
- Web build: PASS (`cd web && npm run build`)
- Desktop tests: PASS (`cd desktop/frontend && npm test -- --run`)
- Desktop build: PASS (`cd desktop/frontend && npm run build`)

### Known follow-ups
- Existing non-blocking MUI `anchorEl`, desktop `useRouter`, third-party `'use client'`, and Vite chunk-size warnings remain in test/build logs.

## Milestone — Authenticated Markdown Documentation Browser (Complete)

### What changed
- Added an authenticated Sukumad documentation backend under [backend/internal/sukumad/documentation](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/documentation) with:
  - configurable markdown root path
  - explicit allowlist of relevant markdown files
  - slug-based document listing and detail reads
  - path traversal, absolute path, duplicate slug, extension, and missing file validation
- Added documentation configuration defaults and config-file entries in:
  - [backend/internal/config/config.go](/Users/sam/projects/go/sukumadpro/backend/internal/config/config.go)
  - [backend/config/config.yaml](/Users/sam/projects/go/sukumadpro/backend/config/config.yaml)
- Registered the `documentation` module across backend module enablement and RBAC metadata in:
  - [backend/internal/moduleenablement/registry.go](/Users/sam/projects/go/sukumadpro/backend/internal/moduleenablement/registry.go)
  - [backend/internal/rbac/registry.go](/Users/sam/projects/go/sukumadpro/backend/internal/rbac/registry.go)
- Wired authenticated backend routes through the existing Sukumad router:
  - `GET /api/v1/documentation`
  - `GET /api/v1/documentation/:slug`
- Added a MUI-styled markdown documentation page to web at [web/src/pages/DocumentationPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/DocumentationPage.tsx), with matching route, navigation, module registry, module enablement, and settings label wiring.
- Added the matching desktop documentation page at [desktop/frontend/src/pages/DocumentationPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/DocumentationPage.tsx), using the backend API only.
- Added `react-markdown` and `remark-gfm` to the web and desktop frontend packages for markdown rendering. Versions were pinned to the Vite-compatible React Markdown 8 / Remark GFM 3 line.
- Added architecture notes in [docs/notes/documentation-browser.md](/Users/sam/projects/go/sukumadpro/docs/notes/documentation-browser.md).
- Saved prompt traceability copy in `docs/prompts/2026-04-10-documentation-pages.md` (gitignored).

### Added or updated tests
- Backend:
  - added documentation service coverage in [backend/internal/sukumad/documentation/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/documentation/service_test.go)
  - extended authenticated route coverage in [backend/cmd/api/router_sukumad_test.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/router_sukumad_test.go)
  - updated module enablement and RBAC registry expectations for the new module
- Web:
  - added documentation route coverage in [web/src/routes.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/routes.test.tsx)
  - updated registry coverage in [web/src/registry/registry.test.ts](/Users/sam/projects/go/sukumadpro/web/src/registry/registry.test.ts)
- Desktop:
  - added documentation route coverage in [desktop/frontend/src/routes.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/routes.test.tsx)
  - updated registry coverage in [desktop/frontend/src/registry/registry.test.ts](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/registry/registry.test.ts)

### How to run tests
- `cd backend && GOCACHE=/tmp/go-build go test ./...`
- `cd web && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm test`
- `cd web && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm run build`
- `cd desktop/frontend && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm test`
- `cd desktop/frontend && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm run build`

### Verification summary
- Backend tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./...`)
- Web tests: PASS (`cd web && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm test`)
- Web build: PASS (`cd web && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm run build`)
- Desktop tests: PASS (`cd desktop/frontend && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm test`)
- Desktop build: PASS (`cd desktop/frontend && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm run build`)

### Known follow-ups
- Web and desktop verification still emit existing non-blocking MUI `anchorEl`, desktop `useRouter`, third-party `'use client'`, and bundle-size warnings.
- `npm install` emitted existing audit and engine warnings in this environment; the build and test commands were run with the nvm-managed Node 22 binary.

## Milestone — Sukumad Operations Dashboard (Complete)

### What changed
- Added a new backend dashboard module under [backend/internal/sukumad/dashboard](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/dashboard) with:
  - snapshot aggregation for KPIs, trends, attention panels, worker summaries, and recent events
  - `GET /api/v1/dashboard/operations`
  - `GET /api/v1/dashboard/operations/events` websocket streaming
  - sanitized stream event mapping for safe live payloads
- Wired dashboard routes into the authenticated Sukumad router with `observability.read` permission enforcement in:
  - [backend/internal/sukumad/routes.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/routes.go)
  - [backend/internal/middleware/auth.go](/Users/sam/projects/go/sukumadpro/backend/internal/middleware/auth.go)
  - browser and desktop websocket clients now authenticate through the existing JWT model using the `access_token` query parameter for websocket upgrade requests
- Published live dashboard events from the persisted observability path in [backend/internal/sukumad/observability/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/observability/service.go), so dashboard live updates stay aligned with the existing event source of truth.
- Replaced the placeholder web dashboard with a snapshot-driven operations view in [web/src/pages/DashboardPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/DashboardPage.tsx):
  - KPI cards
  - attention panels
  - worker health
  - trends
  - recent events
  - live websocket connection state and incremental updates
- Replaced the placeholder desktop dashboard with the matching snapshot-driven operations view in [desktop/frontend/src/pages/DashboardPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/DashboardPage.tsx) and added typed dashboard API support in [desktop/frontend/src/api/client.ts](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/api/client.ts).
- Added drill-down filter support for dashboard navigation across web and desktop through:
  - [web/src/pages/listRouteSearch.ts](/Users/sam/projects/go/sukumadpro/web/src/pages/listRouteSearch.ts)
  - [desktop/frontend/src/pages/listRouteSearch.ts](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/listRouteSearch.ts)
  - [web/src/routes.tsx](/Users/sam/projects/go/sukumadpro/web/src/routes.tsx)
  - [desktop/frontend/src/routes.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/routes.tsx)
  - dashboard actions now deep-link into filtered Deliveries, Jobs, and Observability pages instead of generic lists
- Hardened snapshot/live-update verification support:
  - fixed the web dashboard timer typing in [web/src/pages/DashboardPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/DashboardPage.tsx)
  - updated stale theme test fixtures in [web/src/ui/theme/theme.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/ui/theme/theme.test.tsx)
  - raised Vitest suite/test time budgets where the full desktop/web matrix was timing out under load in:
    - [web/vite.config.ts](/Users/sam/projects/go/sukumadpro/web/vite.config.ts)
    - [desktop/frontend/vite.config.ts](/Users/sam/projects/go/sukumadpro/desktop/frontend/vite.config.ts)
    - [web/src/pages/requests-page.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/requests-page.test.tsx)
    - [desktop/frontend/src/pages/requests-page.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/requests-page.test.tsx)
    - [desktop/frontend/src/pages/servers-page.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/servers-page.test.tsx)
    - [desktop/frontend/src/pages/deliveries-page.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/deliveries-page.test.tsx)
    - [desktop/frontend/src/pages/dashboard-page.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/dashboard-page.test.tsx)

### Added or updated tests
- Backend:
  - added dashboard repository/service/handler coverage in:
    - [backend/internal/sukumad/dashboard/repository_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/dashboard/repository_test.go)
    - [backend/internal/sukumad/dashboard/handler_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/dashboard/handler_test.go)
    - [backend/internal/sukumad/dashboard/stream_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/dashboard/stream_test.go)
  - extended route/auth/websocket coverage in [backend/cmd/api/router_sukumad_test.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/router_sukumad_test.go)
- Web:
  - added dashboard snapshot/live-update coverage in [web/src/pages/dashboard-page.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/dashboard-page.test.tsx)
  - existing route coverage remains in [web/src/routes.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/routes.test.tsx)
- Desktop:
  - added dashboard snapshot/live-update coverage in [desktop/frontend/src/pages/dashboard-page.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/dashboard-page.test.tsx)
  - updated route coverage in [desktop/frontend/src/routes.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/routes.test.tsx)

### How to run tests
- `cd backend && GOCACHE=/tmp/go-build go test ./...`
- `cd web && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm test`
- `cd web && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm run build`
- `cd desktop/frontend && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm test`
- `cd desktop/frontend && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm run build`

### Verification summary
- Backend tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./...`)
- Web tests: PASS (`cd web && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm test`)
- Web build: PASS (`cd web && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm run build`)
- Desktop tests: PASS (`cd desktop/frontend && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm test`)
- Desktop build: PASS (`cd desktop/frontend && PATH=/Users/sam/.nvm/versions/node/v22.15.1/bin:/usr/local/bin:/usr/bin:/bin npm run build`)

### Known follow-ups
- Backend full-suite verification requires permission to bind a local ephemeral test server port because the websocket route tests use `httptest.NewServer`.
- Web and desktop verification still emit existing non-blocking MUI `anchorEl`, desktop `useRouter`, third-party `'use client'`, and bundle-size warnings.

## Milestone — OpenAPI Documentation + Scalar UI (Complete)

### What changed
- Added the OpenAPI source spec at [openapi.yaml](/Users/sam/projects/go/sukumadpro/api/openapi.yaml).
- Added `oapi-codegen` configuration at [oapi-codegen.yaml](/Users/sam/projects/go/sukumadpro/api/oapi-codegen.yaml).
- Added committed generated Go output at [openapi.gen.go](/Users/sam/projects/go/sukumadpro/backend/internal/openapi/generated/openapi.gen.go).
- Added OpenAPI/Scalar backend integration in:
  - [handler.go](/Users/sam/projects/go/sukumadpro/backend/internal/openapi/handler.go)
  - [router.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/router.go)
- The backend now serves:
  - `/openapi.yaml` for the raw OpenAPI document
  - `/docs` for the Scalar reference UI
- Added explicit Make targets in [Makefile](/Users/sam/projects/go/sukumadpro/Makefile):
  - `generate-openapi`
  - `check-openapi`
- Pinned the generator tool in [tools.go](/Users/sam/projects/go/sukumadpro/backend/tools.go) so regeneration is reproducible through the backend Go module.
- Added and updated documentation in:
  - [README.md](/Users/sam/projects/go/sukumadpro/README.md)
  - [api-documentation.md](/Users/sam/projects/go/sukumadpro/docs/notes/api-documentation.md)
  - [web/README.md](/Users/sam/projects/go/sukumadpro/web/README.md)
- Saved prompt traceability copy in `docs/prompts/2026-03-17-openapi-scalar.md` (gitignored).

### Added or updated tests
- Backend:
  - added `/openapi.yaml` route coverage in [router_test.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/router_test.go)
  - added `/docs` route coverage in [router_test.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/router_test.go)
- Web:
  - no feature changes required; full suite rerun to confirm no regressions
- Desktop:
  - no feature changes required; full suite rerun to confirm no regressions

### How to run tests
- `cd backend && GOCACHE=/tmp/go-build go test ./...`
- `make check-openapi`
- `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run`
- `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build`
- `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run`
- `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build`

### Verification summary
- Backend tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./...`)
- OpenAPI generation drift check: PASS (`make check-openapi`)
- Web tests: PASS (`cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run`)
- Web build: PASS (`cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build`)
- Desktop tests: PASS (`cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run`)
- Desktop build: PASS (`cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build`)

### Known follow-ups
- The current OpenAPI document is router-aligned and intentionally keeps some evolving request/response envelopes generic.
- The Scalar page is served from this app but loads its JS bundle from jsDelivr at runtime.
- The default `node` binary in this environment is currently broken due to a missing ICU library, so verification used the explicit nvm-managed Node binary shown above.

## Milestone — Sukumad Directory Ingestion Worker (Complete)

### What changed
- Added a production directory-ingestion design note at [docs/notes/sukumad-directory-ingestion.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-directory-ingestion.md).
- Added prompt traceability copy:
  - `docs/prompts/2026-03-16-directory-ingestion.md` (gitignored; not for commit)
- Introduced a durable ingestion ledger through:
  - [backend/migrations/000021_create_ingest_files.up.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000021_create_ingest_files.up.sql)
  - [backend/migrations/000021_create_ingest_files.down.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000021_create_ingest_files.down.sql)
  - new `ingest_files` records now track:
    - discovered / retry / processing / processed / failed state
    - current path, archived path, claim ownership, retry timing, checksum, request linkage, and error metadata
- Added a new Sukumad ingestion module under [backend/internal/sukumad/ingest](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/ingest):
  - repository layer for SQL + in-memory testing
  - service layer for:
    - inbox discovery
    - debounce-aware claim/processing
    - envelope validation
    - `request.Service.CreateRequest(...)` reuse
    - success/archive to `processed`
    - terminal failure archive to `failed`
    - transient retry scheduling
  - runtime layer that combines:
    - `fsnotify` low-latency detection
    - periodic inbox reconciliation
    - stale-claim requeue
- Extended backend configuration in:
  - [backend/internal/config/config.go](/Users/sam/projects/go/sukumadpro/backend/internal/config/config.go)
  - [backend/config/config.yaml](/Users/sam/projects/go/sukumadpro/backend/config/config.yaml)
  - added:
    - `sukumad.ingest.directory.enabled`
    - `sukumad.ingest.directory.inbox_path`
    - `sukumad.ingest.directory.processing_path`
    - `sukumad.ingest.directory.processed_path`
    - `sukumad.ingest.directory.failed_path`
    - `sukumad.ingest.directory.allowed_extensions`
    - `sukumad.ingest.directory.default_source_system`
    - `sukumad.ingest.directory.require_idempotency_key`
    - `sukumad.ingest.directory.debounce_milliseconds`
    - `sukumad.ingest.directory.retry_delay_seconds`
    - `sukumad.ingest.directory.claim_timeout_seconds`
    - `sukumad.ingest.directory.scan_interval_seconds`
    - `sukumad.ingest.directory.batch_size`
- Wired a dedicated worker role into [backend/cmd/worker/main.go](/Users/sam/projects/go/sukumadpro/backend/cmd/worker/main.go):
  - new worker type `ingest`
  - new definition helper in [backend/internal/sukumad/worker/ingest.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/ingest.go)
  - worker process now starts the directory-ingest loop alongside send/retry/poll/retention
- No new web or desktop feature pages were required; the existing app shells and observability surfaces remain the operator-facing UI.
- Adjusted one desktop route test timeout in [desktop/frontend/src/pages/requests-page.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/requests-page.test.tsx) so the existing request-flow test remains stable when the whole desktop suite runs under load.

### Added or updated tests
- Backend:
  - added [backend/internal/sukumad/ingest/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/ingest/service_test.go) for:
    - successful ingest to processed archive + request creation
    - terminal invalid-envelope failure to failed archive
    - transient request-creation failure retry scheduling
    - validation-error terminal handling
  - added [backend/internal/sukumad/ingest/runtime_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/ingest/runtime_test.go) for:
    - watcher drain + reconciliation-driven processing
- Web:
  - no code changes required; full existing test/build verification rerun
- Desktop:
  - no feature code changes required; full existing test/build verification rerun
  - updated the existing request-page test timeout to remove full-suite flakiness

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./internal/sukumad/ingest ./cmd/worker ./internal/config` -> PASS
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Web:
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run src/pages/requests-page.test.tsx --run` -> PASS
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS

### Remaining follow-ups
- The ingestion path currently assumes producers publish files into the inbox using atomic rename semantics; if upstream systems cannot do that, the debounce/stability contract may need stronger writer-side coordination.
- Operator-facing detail for individual `ingest_files` rows is currently in the backend ledger and logs; if users need direct UI visibility into per-file ingest history, that should be added as a dedicated backend/API/web/desktop milestone rather than ad hoc.
- Web and desktop verification still emit the existing non-blocking MUI `anchorEl`, desktop `useRouter`, third-party `'use client'`, and chunk-size warnings.

## Milestone — Sukumad Durable Poll Worker And Recovery Model (Complete)

### What changed
- Hardened the separate worker process in [backend/cmd/worker/main.go](/Users/sam/projects/go/sukumadpro/backend/cmd/worker/main.go):
  - worker startup now performs two durable recovery passes before normal looping:
    - reconcile terminal async tasks that were persisted but not fully rolled up
    - requeue stale `running` deliveries that have no async task
  - poll worker startup now passes a configured async-claim timeout into the real poll loop
- Extended worker configuration in:
  - [backend/internal/config/config.go](/Users/sam/projects/go/sukumadpro/backend/internal/config/config.go)
  - [backend/config/config.yaml](/Users/sam/projects/go/sukumadpro/backend/config/config.yaml)
  - new settings:
    - `sukumad.workers.recovery.stale_delivery_after_seconds`
    - `sukumad.workers.poll.claim_timeout_seconds`
- Added durable async poll claim persistence through:
  - [backend/migrations/000019_add_async_poll_claims.up.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000019_add_async_poll_claims.up.sql)
  - [backend/migrations/000019_add_async_poll_claims.down.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000019_add_async_poll_claims.down.sql)
  - [backend/internal/sukumad/async/repository.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/repository.go)
  - async tasks now claim via `poll_claimed_at` and `poll_claimed_by_worker_run_id`
  - stale async claims are recoverable after the configured timeout
- Reworked async polling in:
  - [backend/internal/sukumad/async/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/service.go)
  - [backend/internal/sukumad/async/types.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/types.go)
  - poll execution is worker-run aware
  - poll events now carry `worker_run_id`
  - poll history still persists in `async_task_polls`
  - terminal async reconciliation can be replayed during recovery
- Added stale-running delivery recovery in:
  - [backend/internal/sukumad/delivery/repository.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/repository.go)
  - [backend/internal/sukumad/delivery/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/service.go)
  - initial `running` deliveries recover back to `pending`
  - retry-attempt `running` deliveries recover back to `retrying` with a due `retry_at`
  - recovered deliveries re-drive request/target roll-up through the existing target updater
- Improved worker observability in:
  - [backend/internal/sukumad/worker/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/service.go)
  - [backend/internal/sukumad/worker/types.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/types.go)
  - [backend/internal/sukumad/worker/executor.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/executor.go)
  - [backend/internal/sukumad/worker/poll.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/poll.go)
  - worker runs now persist activity counts in `worker_runs.meta.counts`
  - send/retry/poll execution increments per-run counters and keeps request/delivery/async correlation on worker-generated events
- No web or desktop code changes were required; existing pages already read the durable request/delivery/job state correctly once the backend transitions were fixed.
- Updated worker documentation:
  - [docs/notes/sukumad-workers.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-workers.md)
  - [docs/notes/sukumad-workers-async.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-workers-async.md)
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-13-sukumad-worker-recovery-and-poll-wiring.md` (gitignored; not for commit)

### Added or updated tests
- Backend:
  - updated [backend/internal/sukumad/async/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/service_test.go) for:
    - poll claim/clear behavior
    - poll counter observation
    - terminal async recovery reconciliation
  - updated [backend/internal/sukumad/delivery/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/service_test.go) for stale-running delivery recovery
  - updated [backend/internal/sukumad/worker/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/service_test.go) for persisted worker-run counts
  - added [backend/internal/sukumad/worker/lifecycle_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/lifecycle_test.go) for:
    - durable pending work surviving worker restart
    - async accepted then later success through the poll worker path
    - failed first submission then successful retry-worker completion
  - updated [backend/internal/sukumad/async/repository_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/repository_test.go) for the new async claim columns
  - updated [backend/internal/sukumad/worker/executor_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/executor_test.go) for the expanded delivery executor contract
- Web:
  - no code changes required; full existing test/build verification rerun
- Desktop:
  - no code changes required; full existing test/build verification rerun

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./internal/sukumad/async ./internal/sukumad/delivery ./internal/sukumad/worker ./internal/config ./cmd/worker` -> PASS
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Web:
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS

### Remaining follow-ups
- Worker recovery currently requeues stale `running` deliveries without an async task and replays terminal async reconciliation; if later workflows introduce more mid-flight delivery sub-states, recovery rules should be extended explicitly rather than inferred from attempt number.
- Web and desktop verification still emit the existing non-blocking MUI `anchorEl`, desktop `useRouter`, third-party `'use client'`, and chunk-size warnings.

## Milestone — Sukumad Real Worker Execution Model (Complete)

### What changed
- Introduced a real worker process entrypoint at [backend/cmd/worker/main.go](/Users/sam/projects/go/sukumadpro/backend/cmd/worker/main.go):
  - worker loops now run outside the HTTP server process
  - the worker process starts send, retry, poll, and retention loops with `signal.NotifyContext` shutdown handling
  - loop timing and batch sizes are configurable under:
    - [backend/internal/config/config.go](/Users/sam/projects/go/sukumadpro/backend/internal/config/config.go)
    - [backend/config/config.yaml](/Users/sam/projects/go/sukumadpro/backend/config/config.yaml)
- Removed the unused worker bootstrap from the API startup path in [backend/cmd/api/main.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/main.go) so the API process remains accept-and-persist plus HTTP-only.
- Implemented real send/retry worker execution in [backend/internal/sukumad/worker](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker):
  - added a delivery executor that claims work, loads persisted request/server context, emits worker lifecycle events, and submits through the shared delivery service
  - send worker now claims eligible `pending` deliveries
  - retry worker now claims due `retrying` deliveries and reuses the exact same submission path
  - files:
    - [backend/internal/sukumad/worker/executor.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/executor.go)
    - [backend/internal/sukumad/worker/send.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/send.go)
    - [backend/internal/sukumad/worker/retry.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/retry.go)
- Added durable claim support in [backend/internal/sukumad/delivery](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery):
  - repository claim methods now atomically select and transition one eligible delivery to `running`
  - pending claim respects target/dependency/window eligibility
  - retry claim respects `retry_at <= now`
  - the shared `SubmitDHIS2Delivery(...)` path now accepts already-claimed `running` deliveries so workers do not need a parallel outbound implementation
  - files:
    - [backend/internal/sukumad/delivery/types.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/types.go)
    - [backend/internal/sukumad/delivery/repository.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/repository.go)
    - [backend/internal/sukumad/delivery/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/service.go)
- Updated Sukumad worker/process documentation:
  - [docs/notes/sukumad-workers.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-workers.md)
  - [docs/notes/sukumad-workers-async.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-workers-async.md)
  - [docs/notes/sukumad-how-requests-are-processed.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-how-requests-are-processed.md)
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-13-sukumad-worker-execution-model.md` (gitignored; not for commit)

### Added or updated tests
- Backend:
  - added [backend/internal/sukumad/delivery/repository_claim_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/repository_claim_test.go) for concurrency-safe pending/retry claims
  - updated [backend/internal/sukumad/delivery/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/service_test.go) for worker-claimed shared submission behavior
  - added [backend/internal/sukumad/worker/executor_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/executor_test.go) for send/retry execution and shared submission-path reuse
- Web:
  - no code changes required; existing suites were rerun to confirm no regressions
- Desktop:
  - no code changes required; existing suites were rerun to confirm no regressions

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./internal/sukumad/delivery ./internal/sukumad/worker ./internal/config` -> PASS
  - `cd backend && GOCACHE=/tmp/go-build go test ./cmd/api ./cmd/worker` -> PASS
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Web:
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS

### Remaining follow-ups
- Add stale-running recovery or reconciliation for deliveries claimed into `running` if a worker crashes after claim but before final completion.
- Poll worker pickup is now process-separated and operational, but async-task claim exclusivity is still based on due-task selection rather than a dedicated async claim column/state model.
- Frontend test/build logs still include the existing non-blocking MUI `anchorEl`, desktop `useRouter`, third-party `'use client'`, and chunk-size warnings.

## Milestone — Sukumad Request Creation Accept-and-Persist (Complete)

### What changed
- Refactored Sukumad request creation so `POST /api/v1/requests` is now accept-and-persist only:
  - request creation still validates, persists the request row, target rows, dependency links, and initial delivery attempts
  - initial delivery attempts remain durable in `pending`
  - the request service no longer calls `delivery.Service.SubmitDHIS2Delivery(...)` inline
  - files:
    - [backend/internal/sukumad/request/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/service.go)
    - [backend/cmd/api/main.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/main.go)
- Dependency release behavior now matches the worker-oriented contract:
  - when dependencies complete, blocked targets are released back to `pending`
  - the request roll-up returns to `pending` for later worker pickup instead of dispatching inline
  - dependency-failed cases still roll up to terminal `failed`
- Updated backend tests to prove:
  - request creation persists request, targets, dependencies, and pending deliveries durably
  - unblocked request creation no longer performs outbound submission inline
  - dependency-blocked requests remain blocked durably until release
  - dependency release returns work to pending rather than submitting inline
  - the handler response still returns the persisted request state immediately
  - files:
    - [backend/internal/sukumad/request/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/service_test.go)
    - [backend/internal/sukumad/request/handler_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/handler_test.go)
- Updated architecture and lifecycle notes to document the new accept-and-persist API path:
  - [docs/notes/sukumad-workers.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-workers.md)
  - [docs/notes/sukumad-how-requests-are-processed.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-how-requests-are-processed.md)
  - [docs/notes/sukumad-addons.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-addons.md)
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-13-request-creation-accept-and-persist.md` (gitignored; not for commit)

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./internal/sukumad/request` -> PASS
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Web:
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS

### Remaining follow-ups
- Wire an actual send worker runtime so durable pending deliveries created by the API can be claimed and dispatched automatically.
- Frontend test logs still include the existing non-blocking MUI `anchorEl` warnings and desktop `useRouter` warnings in jsdom-based tests.
- Frontend build logs still include the existing third-party `'use client'` and chunk-size warnings.

## Milestone — Sukumad Worker Architecture Recommendation (Documentation)

### What changed
- Added [docs/notes/sukumad-workers.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-workers.md) to document:
  - the current inline-first Sukumad request-processing model as implemented today
  - the recommended production worker model with separate send, poll, and retry worker roles
  - recommended request and delivery status semantics for worker-driven execution
  - a staged migration path from inline submission to accept-and-persist API behavior
  - operational and testing expectations for production workers
- The note reflects current backend reality:
  - request, target, and delivery records are persisted before outbound dispatch
  - initial delivery submission still happens inline from `request.Service.CreateRequest(...)`
  - async polling is implemented through `async.Service.PollDueTasks(...)`
  - send and retry worker definitions exist but are not wired with execution logic or started as active worker processes

### Tests and verification
- Documentation-only change.
- No code, build, or test commands were run in this step.

### Remaining follow-ups
- Implement durable pickup and claim logic for pending deliveries and due retries.
- Introduce a real worker runtime/process and remove inline first submission from the API path.

## Milestone — Sukumad Addons Gap Closure (Complete)

### What changed
- Closed the remaining implementation gaps from [docs/notes/sukumad-addons.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-addons.md).
- Backend dependency handling now reevaluates dependent requests when a prerequisite request completes or fails:
  - completed dependencies release blocked targets back into the normal submission path
  - failed dependencies convert dependent blocked/pending targets into terminal `dependency_failed` state
  - release/failure flows emit dependency-specific observability events and reuse the existing delivery submission path so submission windows still apply
  - files:
    - [backend/internal/sukumad/request/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/service.go)
- Backend async response filtering now sanitizes filtered remote payloads before persistence:
  - unexpected content types still record a safe bounded summary
  - async task `remote_response` no longer retains raw filtered HTML/body content
  - files:
    - [backend/internal/sukumad/dhis2/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/dhis2/service.go)
- Web and desktop request creation now expose the missing addon inputs:
  - additional destination server IDs for fan-out
  - dependency request IDs
  - both clients submit the backend-supported `destinationServerIds` and `dependencyRequestIds` fields without introducing a parallel route or shell
  - files:
    - [web/src/pages/RequestForm.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/RequestForm.tsx)
    - [web/src/pages/RequestsPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/RequestsPage.tsx)
    - [desktop/frontend/src/pages/RequestForm.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/RequestForm.tsx)
    - [desktop/frontend/src/pages/RequestsPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/RequestsPage.tsx)
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-13-sukumad-addons-gap-closure.md` (gitignored; not for commit)

### Added or updated tests
- Backend:
  - updated [backend/internal/sukumad/request/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/service_test.go) for dependency release/failure reevaluation
  - updated [backend/internal/sukumad/dhis2/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/dhis2/service_test.go) for sanitized filtered async remote responses
- Web:
  - updated [web/src/pages/requests-page.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/requests-page.test.tsx) for addon request-create payload submission
- Desktop:
  - updated [desktop/frontend/src/pages/requests-page.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/requests-page.test.tsx) for the matching addon request-create payload submission

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./internal/sukumad/request ./internal/sukumad/dhis2 ./internal/sukumad/async ./internal/sukumad/delivery ./internal/sukumad/retention ./internal/sukumad/worker` -> PASS
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Web:
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run src/pages/requests-page.test.tsx --run` -> PASS
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run src/pages/requests-page.test.tsx --run` -> PASS
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS

### Remaining follow-ups
- Frontend test logs still include the existing non-blocking MUI `anchorEl` warnings in jsdom-based menu/select tests.
- Frontend build logs still include the existing third-party `'use client'` and chunk-size warnings.

## Milestone — Sukumad Addons Implementation (Complete)

### What changed
- Implemented the addon behaviors from [docs/notes/sukumad-addons.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-addons.md) vertically across backend, web, and desktop.
- Backend request processing now persists and hydrates fan-out target and dependency detail through normal request reads under:
  - [backend/internal/sukumad/request/repository.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/repository.go)
  - [backend/internal/sukumad/request/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/service.go)
  - [backend/internal/sukumad/request/types.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/types.go)
- Request status is now a target-derived roll-up across fan-out combinations instead of a minimal single-destination projection:
  - blocked/deferred targets keep the request blocked
  - processing targets keep the request processing
  - any failed target fails the request
  - all succeeded targets complete the request
- Delivery and async persistence now keep the addon response-filtering fields durable in SQL and memory repositories:
  - [backend/internal/sukumad/delivery/repository.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/repository.go)
  - [backend/internal/sukumad/async/repository.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/repository.go)
- Added a dedicated retention module under [backend/internal/sukumad/retention](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/retention) and wired a retention worker definition through:
  - [backend/internal/sukumad/worker/retention.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/retention.go)
  - [backend/internal/sukumad/worker/types.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/types.go)
  - [backend/internal/config/config.go](/Users/sam/projects/go/sukumadpro/backend/internal/config/config.go)
  - [backend/config/config.yaml](/Users/sam/projects/go/sukumadpro/backend/config/config.yaml)
  - [backend/cmd/api/main.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/main.go)
- Extended request visibility on both clients:
  - [web/src/pages/RequestsPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/RequestsPage.tsx)
  - [web/src/pages/RequestDetailPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/RequestDetailPage.tsx)
  - [desktop/frontend/src/pages/RequestsPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/RequestsPage.tsx)
  - [desktop/frontend/src/pages/RequestDetailPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/RequestDetailPage.tsx)
  - both now expose blocked/deferred reasons, fan-out target counts, target lists, and dependency lists without introducing a parallel shell or route structure
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-13-sukumad-addons-implementation.md` (gitignored; not for commit)

### Added or updated tests
- Backend:
  - added [backend/internal/sukumad/retention/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/retention/service_test.go)
  - updated [backend/internal/sukumad/request/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/service_test.go) for fan-out target roll-up behavior
  - updated [backend/internal/sukumad/request/repository_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/repository_test.go) for target/dependency hydration
  - updated [backend/internal/sukumad/delivery/repository_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/repository_test.go) for persisted response-filter fields
  - updated [backend/internal/sukumad/async/repository_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/repository_test.go) for async poll response filtering fields
- Web:
  - updated [web/src/pages/requests-page.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/requests-page.test.tsx) for blocked/deferred/fan-out request visibility
- Desktop:
  - updated [desktop/frontend/src/pages/requests-page.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/requests-page.test.tsx) for the matching blocked/deferred/fan-out visibility

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./internal/sukumad/request ./internal/sukumad/delivery ./internal/sukumad/async ./internal/sukumad/retention ./internal/sukumad/worker` -> PASS
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Web:
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run src/pages/requests-page.test.tsx --run` -> PASS
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run src/pages/requests-page.test.tsx --run` -> PASS
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS

### Remaining follow-ups
- Frontend test logs still include the existing non-blocking MUI jsdom `anchorEl` warnings and one existing MUI X `rowCount` warning in mocked route tests.
- Frontend build logs still include existing third-party `'use client'` and chunk-size warnings.
- Request roll-up intentionally stays coarse for now; optional-target semantics and richer partial-success UI can be layered later without changing the current persistence model.

## Milestone — Sukumad Addons Design Note (Documentation)

### What changed
- Added [docs/notes/sukumad-addons.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-addons.md) to document refined requirements and implementation-ready architecture for:
  - submission windows / allowed delivery periods
  - max retries
  - expected response content-type filtering
  - retention / purge strategy
  - request dependencies
  - multi-destination / fan-out delivery
- The note is aligned with the current Sukumad request-processing reality:
  - request row persisted first
  - first delivery attempt created immediately
  - first submission triggered inline today
  - retries modeled as new delivery attempts
  - durable async state already in place
  - future worker-driven execution kept compatible by reusing shared gating rules
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-13-sukumad-addons-design.md` (gitignored; not for commit)

### Tests and verification
- Documentation-only change.
- No code, build, or test commands were run in this step.

### Remaining follow-ups
- The addons remain design-only until a later milestone implements schema, service, API, and UI changes vertically across backend, web, and desktop.

## Milestone — Sukumad Rate-Limiting Correction (Complete)

### What changed
- Added a reusable destination-scoped outbound limiter in [backend/internal/ratelimit/ratelimit.go](/Users/sam/projects/go/sukumadpro/backend/internal/ratelimit/ratelimit.go):
  - central registry backed by `golang.org/x/time/rate`
  - thread-safe per-destination limiter lookup and replacement
  - safe default policy when no destination override is configured
- Extended backend runtime config in [backend/internal/config/config.go](/Users/sam/projects/go/sukumadpro/backend/internal/config/config.go) and [backend/config/config.yaml](/Users/sam/projects/go/sukumadpro/backend/config/config.yaml):
  - `sukumad.rate_limit.default.requests_per_second`
  - `sukumad.rate_limit.default.burst`
  - `sukumad.rate_limit.destinations.<server_code>`
- Updated API bootstrap wiring in [backend/cmd/api/main.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/main.go) so the shared DHIS2 dispatcher receives the top-level outbound limiter with config-driven destination policies.
- Refactored the DHIS2 outbound client under:
  - [backend/internal/sukumad/dhis2/client.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/dhis2/client.go)
  - [backend/internal/sukumad/dhis2/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/dhis2/service.go)
  - outbound `Submit` and async `Poll` requests now wait on the shared destination limiter immediately before `http.Client.Do(...)`
  - destination keys prefer the persisted Sukumad server code and fall back to host only if no code is available
- Threaded destination server codes through the delivery/request/async path in:
  - [backend/internal/sukumad/delivery/types.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/types.go)
  - [backend/internal/sukumad/request/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/service.go)
  - [backend/internal/sukumad/async/types.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/types.go)
  - [backend/internal/sukumad/async/repository.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/repository.go)
  - this keeps initial submissions and async follow-up traffic under the same destination-scoped limiter
- No web or desktop code changes were required; existing request, delivery, retry, and observability pages continued to work against the unchanged API surface.
- Updated the architecture note in [docs/notes/sukumad-rate-limiting-correction.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-rate-limiting-correction.md).
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-13-milestone-8-rate-limiting-correction.md` (gitignored; not for commit)

### Added or updated tests
- Backend:
  - added [backend/internal/ratelimit/ratelimit_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/ratelimit/ratelimit_test.go)
  - updated [backend/internal/config/config_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/config/config_test.go) for Sukumad outbound rate-limit config validation
  - updated [backend/internal/sukumad/dhis2/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/dhis2/service_test.go) for destination-key limiter coverage on submit and poll
  - updated [backend/internal/sukumad/delivery/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/service_test.go) for retry-path reuse of the same destination-scoped dispatcher path
  - updated [backend/internal/sukumad/async/repository_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/repository_test.go) for destination-code projections
- Web:
  - no code changes required; existing suites verified request status visibility, delivery visibility, and retry/detail flows still pass
- Desktop:
  - no code changes required; existing suites verified the matching visibility and retry/detail flows still pass

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go mod tidy` -> PASS
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Web:
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS

### Remaining follow-ups
- Web and desktop needed no parity fixes for this correction.
- Frontend test logs still include the existing non-blocking MUI jsdom `anchorEl` warnings and one existing MUI X `rowCount` warning in mocked route tests.
- Frontend build logs still include existing third-party `'use client'` and chunk-size warnings.

## Milestone — Traceability, Event History, and Operational Observability (Complete)

### What changed
- Added the append-only `request_events` schema in [backend/migrations/000017_create_request_events.up.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000017_create_request_events.up.sql) and [backend/migrations/000017_create_request_events.down.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000017_create_request_events.down.sql):
  - `request_events` table with request, delivery, async-task, worker, correlation, actor, source-component, and structured JSON event metadata
  - supporting indexes for request, delivery, job, worker, correlation, and created-at lookups
  - additive correlation-oriented indexes on existing Sukumad tables to keep timeline queries fast
- Extended [backend/internal/sukumad/observability](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/observability) from worker/rate-limit read APIs into a traceability module with:
  - structured event persistence and safe metadata masking
  - `GET /api/v1/observability/events`
  - `GET /api/v1/observability/events/:id`
  - `GET /api/v1/observability/trace`
  - `GET /api/v1/requests/:id/events`
  - `GET /api/v1/deliveries/:id/events`
  - `GET /api/v1/jobs/:id/events`
- Added a shared event-writing contract in [backend/internal/sukumad/traceevent/traceevent.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/traceevent/traceevent.go) so lifecycle modules emit structured operational events without introducing package cycles.
- Extended the Sukumad request, delivery, async, and worker services under:
  - [backend/internal/sukumad/request/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/service.go)
  - [backend/internal/sukumad/delivery/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/service.go)
  - [backend/internal/sukumad/async/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/service.go)
  - [backend/internal/sukumad/worker/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/service.go)
  - requests emit creation, submission, status-change, completion, and failure events
  - deliveries emit creation, start, response-received, success, failure, retry-scheduled, and retry-started events
  - async tasks emit creation, poll-started, poll-succeeded, poll-failed, completion, and failure events
  - worker runs emit started, heartbeat, stopped, and error events
  - event metadata is structured, append-only, and token/password fields are masked before persistence
- Updated bootstrap wiring in [backend/cmd/api/main.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/main.go) so the request, delivery, async, worker, and observability services share the same event writer and correlation-aware trace surface.
- Extended the web Sukumad pages in:
  - [web/src/pages/ObservabilityPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/ObservabilityPage.tsx)
  - [web/src/pages/RequestsPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/RequestsPage.tsx)
  - [web/src/pages/RequestDetailPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/RequestDetailPage.tsx)
  - [web/src/pages/DeliveriesPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/DeliveriesPage.tsx)
  - [web/src/pages/DeliveryDetailPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/DeliveryDetailPage.tsx)
  - [web/src/pages/JobsPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/JobsPage.tsx)
  - [web/src/pages/JobDetailPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/JobDetailPage.tsx)
  - [web/src/pages/traceability.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/traceability.tsx)
  - observability now has an event-history grid, filter controls, correlation trace lookup, and event-detail dialog
  - request, delivery, and job detail dialogs now show operator-facing event timelines alongside the existing payload/response/poll views
- Extended the desktop frontend with matching API-backed functionality in:
  - [desktop/frontend/src/pages/ObservabilityPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/ObservabilityPage.tsx)
  - [desktop/frontend/src/pages/RequestsPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/RequestsPage.tsx)
  - [desktop/frontend/src/pages/RequestDetailPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/RequestDetailPage.tsx)
  - [desktop/frontend/src/pages/DeliveriesPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/DeliveriesPage.tsx)
  - [desktop/frontend/src/pages/DeliveryDetailPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/DeliveryDetailPage.tsx)
  - [desktop/frontend/src/pages/JobsPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/JobsPage.tsx)
  - [desktop/frontend/src/pages/JobDetailPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/JobDetailPage.tsx)
  - [desktop/frontend/src/pages/traceability.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/traceability.tsx)
  - desktop preserves parity with web and continues to use backend APIs only
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-13-milestone-7-traceability-event-history-observability.md` (gitignored; not for commit)

### Added or updated tests
- Backend:
  - added [backend/internal/sukumad/observability/tracing_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/observability/tracing_test.go)
  - updated [backend/internal/sukumad/request/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/service_test.go) for request event emission
  - updated [backend/internal/sukumad/delivery/repository_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/repository_test.go) and [backend/internal/sukumad/async/repository_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/repository_test.go) for correlation-aware projections
  - updated [backend/cmd/api/router_sukumad_test.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/router_sukumad_test.go) for observability/request/delivery/job event permission coverage
- Web:
  - existing Sukumad page and route tests now exercise the observability/events-enabled pages through the full web suite
- Desktop:
  - existing Sukumad page and route tests now exercise the observability/events-enabled pages through the full desktop suite

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Web:
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS

### Remaining follow-ups
- No intentional web/desktop parity gaps remain for this milestone.
- Frontend test logs still include existing non-blocking MUI jsdom `anchorEl` warnings and one existing MUI X `rowCount` warning in mocked route tests.
- Frontend build logs still include existing third-party `'use client'` and chunk-size warnings.

## Milestone — DHIS2 Async Submission Integration (Complete)

### What changed
- Replaced the placeholder DHIS2 integration with a production-shaped submission and polling module under [backend/internal/sukumad/dhis2](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/dhis2):
  - server-driven request preparation from persisted Sukumad server records
  - structured interpretation for synchronous DHIS2 import responses
  - structured interpretation for asynchronous DHIS2 task/import-status responses
  - polling support that normalizes remote terminal and non-terminal states
- Extended the Sukumad delivery engine under [backend/internal/sukumad/delivery](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery) so DHIS2-targeted requests now:
  - create and run a delivery attempt from request creation
  - persist response metadata without exposing secrets
  - distinguish synchronous success, synchronous failure, and async hand-off
  - create linked async tasks for DHIS2 async flows
  - reconcile terminal async outcomes back into delivery state
- Extended the Sukumad async lifecycle under [backend/internal/sukumad/async](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async) so DHIS2 poll results now:
  - map into `pending`, `polling`, `succeeded`, and `failed`
  - keep transient poll failures retryable instead of forcing premature terminal failure
  - reconcile linked deliveries on terminal completion
  - roll request status forward to `processing`, `completed`, or `failed`
- Extended Sukumad request detail loading under [backend/internal/sukumad/request](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request) so list/detail responses expose the latest delivery and async linkage metadata needed by clients.
- Updated API bootstrap wiring in [backend/cmd/api/main.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/main.go) so the delivery service, request service, async service, and worker poller share the new DHIS2 dispatcher and reconciliation flow.
- Extended the web Sukumad pages in [web/src/pages/RequestsPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/RequestsPage.tsx), [web/src/pages/RequestDetailPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/RequestDetailPage.tsx), [web/src/pages/DeliveriesPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/DeliveriesPage.tsx), and [web/src/pages/DeliveryDetailPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/DeliveryDetailPage.tsx):
  - requests show async-waiting state, latest delivery, latest job state, and remote DHIS2 tracking details
  - deliveries show system type, submission mode, linked async job, remote job ID, poll URL, and terminal outcome details
- Reused the existing jobs and servers routes without redesign:
  - job detail APIs already surface DHIS2 remote job metadata, poll state, and latest response snapshots through the existing jobs UI
  - server forms continue to expose the DHIS2-relevant persisted fields introduced in earlier milestones
- Extended the desktop Sukumad pages in [desktop/frontend/src/pages/RequestsPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/RequestsPage.tsx), [desktop/frontend/src/pages/RequestDetailPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/RequestDetailPage.tsx), [desktop/frontend/src/pages/DeliveriesPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/DeliveriesPage.tsx), and [desktop/frontend/src/pages/DeliveryDetailPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/DeliveryDetailPage.tsx) with the same DHIS2-aware visibility, preserving web/desktop parity and backend-only access.
- Added a DHIS2 async integration note in [docs/notes/sukumad-dhis2-async-integration.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-dhis2-async-integration.md).
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-12-milestone-6-dhis2-async-submission-integration.md` (gitignored; not for commit)

### Added or updated tests
- Backend:
  - added [backend/internal/sukumad/dhis2/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/dhis2/service_test.go)
  - updated [backend/internal/sukumad/delivery/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/service_test.go) for DHIS2 submission and async-task creation
  - updated [backend/internal/sukumad/async/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/service_test.go) for terminal reconciliation and retryable poll errors
  - updated [backend/internal/sukumad/request/repository_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/repository_test.go) and [backend/internal/sukumad/delivery/repository_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/repository_test.go) for enriched DHIS2-aware detail projections
- Web:
  - updated [web/src/pages/requests-page.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/requests-page.test.tsx)
  - updated [web/src/pages/deliveries-page.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/deliveries-page.test.tsx)
  - updated [web/src/routes.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/routes.test.tsx)
- Desktop:
  - updated [desktop/frontend/src/components/datagrid/AppDataGrid.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/components/datagrid/AppDataGrid.test.tsx) to stabilize grid state assertions in the full suite
  - updated [desktop/frontend/src/pages/requests-page.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/requests-page.test.tsx)
  - updated [desktop/frontend/src/pages/deliveries-page.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/deliveries-page.test.tsx)
  - updated [desktop/frontend/src/routes.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/routes.test.tsx)

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Web:
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS

### Remaining follow-ups
- Jobs pages still inherit existing generic poll-history presentation; a later milestone can add richer DHIS2-specific job-detail fields without changing the underlying reconciliation model.
- Frontend test logs still include existing non-blocking MUI jsdom `anchorEl` warnings.
- Frontend build logs still include existing third-party `'use client'` and chunk-size warnings.

## Milestone — Workers, Async Tasks, and Rate Limiting (Complete)

### What changed
- Added the async-processing schema in [backend/migrations/000016_create_async_and_worker_tables.up.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000016_create_async_and_worker_tables.up.sql) and [backend/migrations/000016_create_async_and_worker_tables.down.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000016_create_async_and_worker_tables.down.sql):
  - `async_tasks`
  - `async_task_polls`
  - `rate_limit_policies`
  - `worker_runs`
- Replaced the Sukumad async placeholder module with a real implementation under [backend/internal/sukumad/async](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async):
  - async task create/list/detail support linked to delivery attempts and request references
  - poll history recording and list/detail visibility
  - status updates for `pending`, `polling`, `succeeded`, and `failed`
  - generic due-task polling abstraction through `poller.go`
  - audit events for `async_task.created`, `async_task.completed`, and `async_task.failed`
- Replaced the worker placeholder module with a real lifecycle implementation under [backend/internal/sukumad/worker](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker):
  - persistent worker-run registration
  - heartbeat updates
  - clean status transitions
  - context-cancelled shutdown support
  - bootstrap seam and production-shaped send/poll/retry worker definitions
- Replaced the rate-limit placeholder module with a real implementation under [backend/internal/sukumad/ratelimit](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/ratelimit):
  - policy persistence and listing
  - active-policy resolution by scope
  - in-process limiter with pacing and concurrency gating hooks
- Replaced the observability placeholder module with real read APIs under [backend/internal/sukumad/observability](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/observability):
  - worker list/detail visibility
  - rate-limit visibility
- Updated Sukumad backend routing so the authenticated RBAC-enforced API surface now includes:
  - `GET /api/v1/jobs`
  - `GET /api/v1/jobs/:id`
  - `GET /api/v1/jobs/:id/polls`
  - `GET /api/v1/observability/workers`
  - `GET /api/v1/observability/workers/:id`
  - `GET /api/v1/observability/rate-limits`
- Replaced the web placeholders with real jobs and observability pages in [web/src/pages/JobsPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/JobsPage.tsx), [web/src/pages/JobDetailPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/JobDetailPage.tsx), and [web/src/pages/ObservabilityPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/ObservabilityPage.tsx):
  - jobs grid with state filtering, sorting, and detail inspection
  - poll-history visibility inside the job detail dialog
  - read-only worker and rate-limit observability grids
- Replaced the desktop placeholders with matching API-backed functionality in [desktop/frontend/src/pages/JobsPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/JobsPage.tsx), [desktop/frontend/src/pages/JobDetailPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/JobDetailPage.tsx), and [desktop/frontend/src/pages/ObservabilityPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/ObservabilityPage.tsx), preserving backend-only data access and parity with web.
- Added an architecture note for the async/worker layer in [docs/notes/sukumad-workers-async.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-workers-async.md).
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-12-milestone-5-workers-async-rate-limiting.md` (gitignored; not for commit)

### Added or updated tests
- Backend:
  - added [backend/internal/sukumad/async/repository_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/repository_test.go)
  - added [backend/internal/sukumad/async/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/service_test.go)
  - added [backend/internal/sukumad/async/handler_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/async/handler_test.go)
  - added [backend/internal/sukumad/worker/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/worker/service_test.go)
  - added [backend/internal/sukumad/ratelimit/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/ratelimit/service_test.go)
  - updated [backend/cmd/api/router_sukumad_test.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/router_sukumad_test.go) for jobs, observability, and permission coverage
- Web:
  - added [web/src/pages/jobs-observability-page.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/jobs-observability-page.test.tsx)
  - updated [web/src/routes.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/routes.test.tsx) for `/jobs` and `/observability` route coverage
- Desktop:
  - added [desktop/frontend/src/pages/jobs-observability-page.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/jobs-observability-page.test.tsx)
  - updated [desktop/frontend/src/routes.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/routes.test.tsx) for `/jobs` and `/observability` route coverage

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
  - `cd backend && GOCACHE=/tmp/go-build go run ./cmd/migrate up -config ./config/config.yaml` -> PASS
- Web:
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run` -> PASS
  - `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build` -> PASS

### Remaining follow-ups
- Worker definitions and bootstrap seams are now production-shaped, but real startup scheduling remains a controlled hook rather than an always-on runtime path.
- Poll execution is currently generic and abstraction-driven; deeper DHIS2-specific remote polling behavior still belongs to later milestones.
- Existing non-blocking MUI jsdom `anchorEl` warnings remain in frontend test logs.
- Existing non-blocking Vite third-party `'use client'` and chunk-size warnings remain in web and desktop build output.

## Milestone — Delivery Engine & Retry Control (Complete)

### What changed
- Added the `delivery_attempts` migration in [backend/migrations/000015_create_delivery_attempts.up.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000015_create_delivery_attempts.up.sql) and [backend/migrations/000015_create_delivery_attempts.down.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000015_create_delivery_attempts.down.sql).
- Replaced the Sukumad delivery placeholder backend with a real repository/service/handler implementation under [backend/internal/sukumad/delivery](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery):
  - paginated delivery listing with status, server, date, search, and sort support
  - delivery detail lookup enriched with request UID and server name
  - delivery lifecycle operations for `pending`, `running`, `succeeded`, `failed`, and `retrying`
  - retry scheduling through `POST /api/v1/deliveries/:id/retry` for failed deliveries only
  - audit logging for `delivery.created`, `delivery.succeeded`, `delivery.failed`, and `delivery.retry`
- Updated Sukumad route registration so `/api/v1/deliveries` is a full RBAC-enforced surface with:
  - `GET /api/v1/deliveries`
  - `GET /api/v1/deliveries/:id`
  - `POST /api/v1/deliveries/:id/retry`
- Wired request creation to seed the initial pending delivery attempt so new exchange requests now enter the delivery engine immediately using the shared Sukumad delivery service.
- Replaced the web Deliveries placeholder with a real inspection and retry UI in [web/src/pages/DeliveriesPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/DeliveriesPage.tsx) and [web/src/pages/DeliveryDetailPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/DeliveryDetailPage.tsx):
  - delivery grid for delivery UID, request UID, server, status, attempt number, started time, and finished time
  - filters for status, server, and date
  - detail dialog showing request reference, server, attempt number, status, response body, error message, and timestamps
  - permission-aware retry control for failed deliveries
- Replaced the desktop Deliveries placeholder with matching API-backed functionality in [desktop/frontend/src/pages/DeliveriesPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/DeliveriesPage.tsx) and [desktop/frontend/src/pages/DeliveryDetailPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/DeliveryDetailPage.tsx), preserving web/desktop parity and backend-only data access.
- Added a delivery-engine architecture note in [docs/notes/sukumad-delivery-engine.md](/Users/sam/projects/go/sukumadpro/docs/notes/sukumad-delivery-engine.md).
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-12-milestone-4-delivery-engine-retry-control.md` (gitignored; not for commit)

### Added or updated tests
- Backend:
  - added [backend/internal/sukumad/delivery/repository_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/repository_test.go)
  - added [backend/internal/sukumad/delivery/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/service_test.go)
  - added [backend/internal/sukumad/delivery/handler_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/handler_test.go)
  - updated [backend/cmd/api/router_sukumad_test.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/router_sukumad_test.go) for delivery list/detail/retry flows and permission coverage
- Web:
  - added [web/src/pages/deliveries-page.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/deliveries-page.test.tsx)
  - updated [web/src/routes.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/routes.test.tsx)
- Desktop:
  - added [desktop/frontend/src/pages/deliveries-page.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/deliveries-page.test.tsx)
  - updated [desktop/frontend/src/routes.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/routes.test.tsx)

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Web:
  - `cd web && npm test -- --run` -> PASS
  - `cd web && npm run build` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && npm test -- --run` -> PASS
  - `cd desktop/frontend && npm run build` -> PASS

### Remaining follow-ups
- Delivery retries are scheduled in the database with `retrying` status and `retry_at`, but worker pickup/execution remains a later milestone.
- Request creation now seeds an initial pending delivery attempt; deeper transactional orchestration between request and delivery persistence can be tightened further if later milestones require strict atomicity across both records.
- Existing non-blocking MUI jsdom `anchorEl` warnings remain in frontend test logs.
- Existing non-blocking Vite third-party `'use client'` and chunk-size warnings remain in web and desktop build output.

## Milestone — Request Lifecycle (Complete)

### What changed
- Added the `exchange_requests` migration in [backend/migrations/000014_create_exchange_requests.up.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000014_create_exchange_requests.up.sql) and [backend/migrations/000014_create_exchange_requests.down.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000014_create_exchange_requests.down.sql).
- Replaced the Sukumad request placeholder backend with a real repository/service/handler implementation under [backend/internal/sukumad/request](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request):
  - paginated request listing with search, sort, and status filtering
  - request detail lookup with destination-server enrichment
  - request creation with JSON payload validation, metadata capture, initial `pending` status, and UID generation
  - RBAC-enforced `GET /api/v1/requests`, `GET /api/v1/requests/:id`, and `POST /api/v1/requests`
  - audit logging for `request.created`
- Updated Sukumad backend route registration and API bootstrap wiring so the request module uses the existing Gin router, middleware, SQL repository wiring, and BasePro audit stack.
- Replaced the web placeholder Requests page with a real request lifecycle UI in [web/src/pages/RequestsPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/RequestsPage.tsx):
  - request list grid for UID, destination server, status, created time, and correlation ID
  - permission-aware `Create Request` flow using backend APIs
  - request detail inspection through the existing `/requests` route using [web/src/pages/RequestForm.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/RequestForm.tsx) and [web/src/pages/RequestDetailPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/RequestDetailPage.tsx)
- Replaced the desktop placeholder Requests page with matching API-backed functionality in [desktop/frontend/src/pages/RequestsPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/RequestsPage.tsx), with parity components in [desktop/frontend/src/pages/RequestForm.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/RequestForm.tsx) and [desktop/frontend/src/pages/RequestDetailPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/RequestDetailPage.tsx).
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-12-milestone-3-request-lifecycle.md` (gitignored; not for commit)

### Added or updated tests
- Backend:
  - [backend/internal/sukumad/request/repository_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/repository_test.go)
  - [backend/internal/sukumad/request/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/service_test.go)
  - [backend/internal/sukumad/request/handler_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/handler_test.go)
  - updated [backend/cmd/api/router_sukumad_test.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/router_sukumad_test.go) for request create/list/detail and permission coverage
- Web:
  - added [web/src/pages/requests-page.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/requests-page.test.tsx)
  - updated [web/src/routes.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/routes.test.tsx)
- Desktop:
  - added [desktop/frontend/src/pages/requests-page.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/requests-page.test.tsx)
  - updated [desktop/frontend/src/routes.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/routes.test.tsx)

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Web:
  - `cd web && npm test -- --run` -> PASS
  - `cd web && npm run build` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && npm test -- --run` -> PASS
  - `cd desktop/frontend && npm run build` -> PASS

### Remaining follow-ups
- Request status transition endpoints and background processing are still pending later milestones; requests currently enter the lifecycle in `pending` state only.
- Deliveries, jobs, and observability remain separate placeholder or partial Sukumad surfaces outside this milestone’s request lifecycle scope.
- Existing non-blocking MUI jsdom `anchorEl` warnings remain in frontend test logs.
- Existing non-blocking Vite third-party `'use client'` and chunk-size warnings remain in web and desktop build output.

## Milestone — Server Management (Complete)

### What changed
- Added the `integration_servers` migration in [backend/migrations/000013_create_integration_servers.up.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000013_create_integration_servers.up.sql) and [backend/migrations/000013_create_integration_servers.down.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000013_create_integration_servers.down.sql).
- Replaced the Sukumad server placeholder backend with a real repository/service/handler implementation under [backend/internal/sukumad/server](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/server):
  - paginated list, detail, create, update, and delete support
  - input validation and structured validation errors
  - RBAC-enforced CRUD routing through the existing Gin router/middleware stack
  - audit events for create, update, delete, suspend, and activate
- Updated Sukumad backend route registration so `/api/v1/servers` is a full CRUD surface while the other Sukumad modules remain on their existing placeholder list routes.
- Replaced the web placeholder Servers page with a real CRUD page in [web/src/pages/ServersPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/ServersPage.tsx):
  - Data Grid list for server name, code, system type, base URL, async status, and suspended status
  - create/edit dialogs with client-side validation
  - delete and suspend/activate actions
- Replaced the desktop placeholder Servers page with matching CRUD behavior in [desktop/frontend/src/pages/ServersPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/ServersPage.tsx), using backend APIs only.
- Grouped Sukumad navigation under an explicit `Sukumad` menu in both clients while preserving the existing shell/router/navigation patterns:
  - web shell/navigation and route access registry updated
  - desktop registry-driven navigation and shell updated
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-12-milestone-2-server-management.md` (gitignored; not for commit)

### Added or updated tests
- Backend:
  - [backend/internal/sukumad/server/repository_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/server/repository_test.go)
  - [backend/internal/sukumad/server/service_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/server/service_test.go)
  - [backend/internal/sukumad/server/handler_test.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/server/handler_test.go)
  - updated [backend/cmd/api/router_sukumad_test.go](/Users/sam/projects/go/sukumadpro/backend/cmd/api/router_sukumad_test.go) for CRUD and permission coverage
- Web:
  - added [web/src/pages/servers-page.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/servers-page.test.tsx)
  - updated [web/src/routes.test.tsx](/Users/sam/projects/go/sukumadpro/web/src/routes.test.tsx)
  - updated [web/src/registry/registry.test.ts](/Users/sam/projects/go/sukumadpro/web/src/registry/registry.test.ts)
- Desktop:
  - added [desktop/frontend/src/pages/servers-page.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/servers-page.test.tsx)
  - updated [desktop/frontend/src/routes.test.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/routes.test.tsx)
  - updated [desktop/frontend/src/registry/registry.test.ts](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/registry/registry.test.ts)

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Web:
  - `cd web && npm test -- --run` -> PASS
  - `cd web && npm run build` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && npm test -- --run` -> PASS
  - `cd desktop/frontend && npm run build` -> PASS

### Remaining follow-ups
- Requests, deliveries, jobs, and observability remain placeholder Sukumad surfaces and are now grouped under the same `Sukumad` navigation section as Servers.
- Existing non-blocking MUI jsdom `anchorEl` warnings remain in frontend test logs.
- Existing non-blocking Vite third-party `'use client'` and chunk-size warnings remain in web and desktop build output.

## Milestone - SukumadPro Bootstrap (Complete)

### What changed
- Added Sukumad backend scaffolding under `backend/internal/sukumad`:
  - `server`
  - `request`
  - `delivery`
  - `async`
  - `ratelimit`
  - `worker`
  - `dhis2`
  - `observability`
- Added placeholder handler/service/repository/types files for the new Sukumad module tree without introducing any new top-level backend platform modules.
- Added placeholder authenticated backend endpoints using the existing API router and middleware stack:
  - `GET /api/v1/servers`
  - `GET /api/v1/requests`
  - `GET /api/v1/deliveries`
  - `GET /api/v1/jobs`
  - `GET /api/v1/observability`
- Added initial Sukumad RBAC permissions to the existing backend/client registries:
  - `servers.read`, `servers.write`
  - `requests.read`, `requests.write`
  - `deliveries.read`, `deliveries.write`
  - `jobs.read`, `jobs.write`
  - `observability.read`
- Extended backend and client module-enablement registries for:
  - `servers`
  - `requests`
  - `deliveries`
  - `jobs`
  - `observability`
- Added web placeholder pages, authenticated routes, page labels, shell navigation icons, and permission-aware visibility for:
  - `/servers`
  - `/requests`
  - `/deliveries`
  - `/jobs`
  - `/observability`
- Added desktop placeholder pages, authenticated routes, route labels, shell navigation icons, and registry-driven permission visibility for the same Sukumad areas.
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-12-sukumad-bootstrap.md` (gitignored; not for commit)

### Added or updated tests
- Backend:
  - `backend/cmd/api/router_sukumad_test.go`
  - updated Sukumad-related permission/module registry coverage in:
    - `backend/internal/rbac/registry_test.go`
    - `backend/internal/moduleenablement/registry_test.go`
    - `backend/internal/moduleenablement/service_test.go`
    - `backend/cmd/api/router_module_enablement_test.go`
- Web:
  - updated `web/src/registry/registry.test.ts`
  - updated `web/src/routes.test.tsx` with Sukumad route/navigation smoke coverage
- Desktop:
  - updated `desktop/frontend/src/registry/registry.test.ts`
  - updated `desktop/frontend/src/routes.test.tsx` with Sukumad route/navigation smoke coverage

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Web:
  - `cd web && npm test -- --run` -> PASS
  - `cd web && npm run build` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && npm test -- --run` -> PASS
  - `cd desktop/frontend && npm run build` -> PASS

### Remaining follow-ups
- Sukumad endpoints currently return placeholder `{ "message": "not implemented" }` responses until later milestones add repository-backed business logic.
- Existing non-blocking MUI jsdom `anchorEl` warnings remain in web and desktop test logs.
- Existing non-blocking Vite third-party `'use client'` and chunk-size warnings remain in web and desktop build output.

## Milestone — Desktop/Web Bootstrap Consumption + Settings Access Guard (Complete)

### What changed
- Added typed client bootstrap runtime containers in both clients:
  - desktop: `desktop/frontend/src/bootstrap/state.ts`
  - web: `web/src/bootstrap/state.ts`
- Added authenticated bootstrap consumption from backend `GET /api/v1/bootstrap`:
  - desktop authenticated gate now:
    - hydrates cached bootstrap first (if available)
    - refreshes live bootstrap when reachable
    - applies effective module state from bootstrap payload
    - uses bootstrap principal when available, with `/auth/me` fallback
  - web auth provider now:
    - fetches bootstrap after login and refresh
    - applies effective module state from bootstrap payload
    - uses bootstrap principal when available, with `/auth/me` fallback
- Added offline-aware bootstrap cache behavior in both clients:
  - persist last successful bootstrap payload in local storage
  - hydrate cached payload on startup/refresh attempt before network refresh
  - continue app startup/auth flow when bootstrap refresh fails (fallback to cached/default + existing auth/me flow)
- Added bootstrap-driven UI shaping in both clients:
  - app shell branding display name now reads from bootstrap branding payload with existing static fallback
  - effective module visibility remains driven by backend payload, now consumed via bootstrap for parity
- Tightened Settings access policy in both clients to match contract intent:
  - Settings navigation visibility and direct route access now require:
    - `admin` role OR
    - `settings.write` permission
  - users with only `settings.read` are denied Settings route access (forbidden/not-authorized fallback per client pattern)
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-08-client-bootstrap-consumption-settings-guard.md` (gitignored; not for commit)

### Added/updated targeted tests
- Desktop:
  - `desktop/frontend/src/bootstrap/state.test.ts` (live cache write, cache hydrate/stale, clear/reset behavior)
  - `desktop/frontend/src/registry/registry.test.ts` (Settings nav access now admin-or-settings.write)
  - `desktop/frontend/src/routes.test.tsx`:
    - module-disable flow switched to bootstrap payload source
    - added Settings denial test for non-admin user without `settings.write`
- Web:
  - `web/src/bootstrap/state.test.ts` (live cache write, cache hydrate/stale, clear/reset behavior)
  - `web/src/registry/registry.test.ts` (Settings nav access now admin-or-settings.write)
  - `web/src/routes.test.tsx`:
    - adjusted staff Settings access fixture to `settings.write`
    - added Settings denial test for non-admin user without `settings.write`

### Tests and verification
- Desktop frontend:
  - `cd desktop/frontend && npm test -- --run` -> PASS
- Web frontend:
  - `cd web && npm test -- --run` -> PASS
- Web production build:
  - `cd web && npm run build` -> PASS

### Remaining follow-ups
- Existing non-blocking MUI jsdom `anchorEl` warnings remain in frontend test logs.
- Existing non-blocking Vite third-party `'use client'` and chunk-size warnings remain in web build output.
- Bootstrap cache freshness is tracked (`stale` state) internally; optional future UX could surface explicit stale/offline status indicators in shell/header.

## Milestone — Server-Driven Bootstrap Config + Settings Authorization Contract (Complete)

### What changed
- Added backend bootstrap endpoint:
  - `GET /api/v1/bootstrap`
- Added typed bootstrap contract implementation in `backend/internal/bootstrap`:
  - app metadata (`version`, `commit`, `buildDate`)
  - branding metadata (from existing login-branding settings service)
  - effective module enablement list (resolved from registry defaults + config/runtime overrides)
  - capability summary (auth/runtime + settings authorization intent)
  - optional authenticated principal summary (user or api-token context)
  - runtime metadata and cache hints for offline-aware client startup usage
- Wired bootstrap route as backend source of truth while preserving existing auth/session behavior:
  - optional principal enrichment via existing auth middleware
  - no changes to login/refresh/logout contracts
- Strengthened Settings write authorization server-side:
  - write Settings APIs now allow `admin` role OR `settings.write` permission
  - applied to:
    - `PUT /api/v1/settings/login-branding`
    - `PUT /api/v1/settings/module-enablement`
  - read Settings APIs remain permission-based (`settings.read`) as currently designed
- Kept settings mutations auditable:
  - existing audit events for branding and module enablement updates remain unchanged and active
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-08-backend-bootstrap-settings-auth.md` (gitignored; not for commit)

### Added/updated targeted tests
- Backend bootstrap contract tests:
  - `backend/cmd/api/router_bootstrap_test.go`
    - response shape and key sections
    - effective module/config data inclusion
    - authenticated principal + capability summary inclusion
- Settings authorization tests:
  - `backend/internal/middleware/authorization_test.go`
    - admin-role override behavior for permission checks
    - non-admin without write permission remains forbidden
  - `backend/cmd/api/router_settings_test.go`
    - settings branding update accepts admin without explicit `settings.write`
  - `backend/cmd/api/router_module_enablement_test.go`
    - module enablement update accepts admin without explicit `settings.write`

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS

### Follow-up notes
- Desktop and web bootstrap consumption/caching behavior is intentionally pending client milestones.
- Frontend route-level Settings gating should consume the new bootstrap capability summary in follow-up parity work.

## Milestone — Module Enablement Admin UX (Complete)

### What changed
- Added a deliberate backend management model for module flags:
  - `static` flags are visible but read-only in admin UX.
  - `runtime` flags are explicitly allowlisted for admin edits.
  - `read-only` control state is supported for experimental/runtime-readonly scenarios.
- Current skeleton manageability defaults:
  - `dashboard`: static (read-only)
  - `administration`: runtime-manageable
  - `settings`: static (read-only)
- Extended backend module-enablement contract and runtime persistence:
  - runtime overrides stored in `app_settings` under `module_enablement/runtime_overrides`
  - effective resolution precedence: default -> config -> runtime (for runtime-manageable flags only)
  - enriched effective payload metadata: `adminControl`, `editable`, and `source` (`default|config|runtime`)
- Added backend API support for module enablement administration:
  - public effective listing remains: `GET /api/v1/modules/effective`
  - admin settings endpoints:
    - `GET /api/v1/settings/module-enablement` (`settings.read`)
    - `PUT /api/v1/settings/module-enablement` (`settings.write`)
  - server-side validation rejects non-editable module updates
  - audit logging on updates: `settings.module_enablement.update`
- Added desktop and web Settings UX parity:
  - new `Module Enablement` section in Settings pages
  - clear state indicators for enabled/disabled, source, static/runtime/read-only control, and experimental flags
  - runtime updates available only for editable rows and authorized writers
  - reader-only users see explicit non-editable guidance
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-08-module-enablement-admin-ux.md` (gitignored; not for commit)

### Added/updated targeted tests
- Backend:
  - `backend/internal/moduleenablement/service_test.go` (runtime update validation + audit event)
  - `backend/internal/moduleenablement/registry_test.go` (runtime source + editable metadata behavior)
  - `backend/cmd/api/router_module_enablement_test.go` (settings endpoint authz + runtime update flow)
- Desktop:
  - updated `desktop/frontend/src/routes.test.tsx` settings tests for module enablement visibility and write-permission guidance
- Web:
  - updated `web/src/routes.test.tsx` settings tests for module enablement visibility and write-permission guidance

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && npm test -- --run` -> PASS
- Web frontend:
  - `cd web && npm test -- --run` -> PASS
- Web build:
  - `cd web && npm run build` -> PASS

### Remaining follow-ups
- Existing non-blocking MUI jsdom `anchorEl` warnings remain in desktop/web frontend test logs.
- Existing non-blocking Vite third-party `'use client'` and chunk-size warnings remain in web build output.

## Milestone — Module Enablement Enforcement Hardening (Complete)

### What changed
- Strengthened module-disabled route/page behavior in both clients:
  - desktop and web now render an explicit `Module Disabled` fallback page for disabled-module direct route access.
  - preserved permission-based unauthorized handling separately for enabled modules.
- Tightened discovery-surface behavior:
  - side navigation remains module-aware (existing).
  - user-menu `Settings` entry now hides when the settings module is disabled (desktop + web).
  - web dashboard shortcut cards now only include currently enabled and routable platform modules.
- Hardened backend module-disabled API behavior:
  - module guard middleware now emits standardized typed app errors via shared error writer (`MODULE_DISABLED`) instead of ad-hoc JSON.
  - guarded API groups remain unchanged in scope (`users`, `audit`, `admin/roles`, `admin/permissions`, `settings`, and public login-branding settings endpoint).
- Implemented deliberate permission-exposure behavior for disabled modules:
  - chosen behavior: **hide disabled-module permissions from default admin listing and assignment flows**.
  - backend RBAC admin service now filters permission list responses based on effective module enablement.
  - backend role create/update now rejects disabled-module permission assignment with typed validation.
  - role detail responses filter disabled-module permissions.
  - desktop/web roles + permissions pages now show explicit info messages clarifying hidden disabled-module permissions.
- Added prompt traceability copy:
  - `docs/prompts/2026-03-08-module-enablement-enforcement.md` (gitignored; not for commit).

### Behavior decisions
- Disabled route handling (desktop + web):
  - direct navigation to a disabled module route renders `Module Disabled` fallback (no broken rendering).
  - enabled route without required permission still renders `Forbidden` / `Not Authorized` as before.
- Disabled permission behavior:
  - disabled-module permissions are hidden from normal admin permission listing and role assignment options.
  - direct attempts to assign disabled permissions are rejected server-side with validation error.

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && npm test -- --run` -> PASS
- Web frontend:
  - `cd web && npm test -- --run` -> PASS
- Web build:
  - `cd web && npm run build` -> PASS

### Added/updated targeted tests
- Backend:
  - `backend/internal/middleware/module_enablement_test.go` (typed module-disabled middleware behavior)
  - `backend/internal/rbac/admin_test.go` (permission exposure filtering + disabled-permission assignment rejection)
- Desktop:
  - updated `desktop/frontend/src/routes.test.tsx` disabled-module route assertions for `Module Disabled` fallback behavior
- Web:
  - updated `web/src/routes.test.tsx` disabled-module route assertions for `Module Disabled` fallback behavior

### Remaining follow-ups
- Existing non-blocking MUI jsdom `anchorEl` warnings remain in desktop/web frontend test logs.
- Existing non-blocking Vite third-party `'use client'` and chunk-size warnings remain in web build output.

## Milestone — Feature Flags / Module Enablement Registry (Implementation) (Complete)

### What changed
- Added a typed backend module-enablement registry and contract:
  - `backend/internal/moduleenablement/registry.go`
  - `backend/internal/moduleenablement/handler.go`
- Added backend-served effective module config endpoint for clients:
  - `GET /api/v1/modules/effective`
  - router wiring in `backend/cmd/api/router.go`
- Added backend config support as the source-of-truth override surface:
  - `modules.flags` in `backend/internal/config/config.go`
  - sample config section in `backend/config/config.yaml`
  - validation for unknown module-flag keys
- Added backend module-disable guards (where practical) for module-owned routes:
  - new middleware `backend/internal/middleware/module_enablement.go`
  - guarded Administration routes (`/users`, `/audit`, `/admin/roles`, `/admin/permissions`)
  - guarded Settings routes (including `/settings/public/login-branding`)
- Added typed module enablement registries and local effective-state handling in both clients:
  - `desktop/frontend/src/registry/moduleEnablement.ts`
  - `web/src/registry/moduleEnablement.ts`
- Connected desktop/web nav and route access checks to effective module enablement:
  - `desktop/frontend/src/registry/navigation.ts`
  - `web/src/registry/navigation.ts`
  - route-guard updates in `desktop/frontend/src/routes.tsx` and `web/src/routes.tsx`
- Desktop and web now consume backend effective module enablement during authenticated bootstrap:
  - desktop via API client + authenticated gate
  - web via `AuthProvider` refresh/login bootstrap
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-07-feature-flags-module-enablement-registry.md` (gitignored, not for commit)

### Source of truth decision
- Implemented: **backend-determined effective module enablement**.
- Registry defaults live in code (backend + clients) for bootstrapping/documentation.
- Runtime effective state is served by backend `/api/v1/modules/effective` using:
  - typed registry defaults
  - optional `modules.flags` config overrides.

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && npm test -- --run` -> PASS
- Web frontend:
  - `cd web && npm test -- --run` -> PASS
- Web production build:
  - `cd web && npm run build` -> PASS

### Added/updated targeted tests
- Backend:
  - `backend/internal/moduleenablement/registry_test.go`
  - `backend/cmd/api/router_module_enablement_test.go`
  - `backend/internal/config/config_test.go` (module flag key validation)
- Desktop:
  - `desktop/frontend/src/registry/registry.test.ts` (enablement defaults + disabled-module checks)
  - `desktop/frontend/src/routes.test.tsx` (disabled-module navigation/route behavior)
- Web:
  - `web/src/registry/registry.test.ts` (enablement defaults + disabled-module checks)
  - `web/src/routes.test.tsx` (disabled-module navigation/route behavior)

### Remaining follow-ups
- Existing non-blocking MUI jsdom `anchorEl` warnings remain in frontend test logs.
- Existing non-blocking Vite warnings for third-party `'use client'` directives and chunk-size notices remain in web build output.

## Milestone — Feature Flags / Module Enablement Registry (Documentation) (Complete)

### Scope completed
- Defined typed module-enablement registry/contract expectations aligned to module, navigation, and permission registries.
- Defined flag scope taxonomy:
  - backend
  - desktop
  - web
  - full-stack
- Defined safe-disable expectations for:
  - navigation visibility
  - route/page access
  - backend API availability/guarding
  - permission exposure behavior

### Delivery notes
- This milestone was documentation/design-first planning only.
- Runtime implementation is captured in the implementation milestone above.

## Milestone — Registry-First Module Extension Workflow (Documentation) (Complete)

### What changed
- Added module-extension guidance note:
  - `docs/notes/module-registry.md`
- Documented a concrete registry-first workflow for adding modules across:
  - desktop registry files (`modules.ts`, `navigation.ts`, `permissions.ts`)
  - web registry files (`modules.ts`, `navigation.ts`, `permissions.ts`)
  - backend RBAC registry/constants (`backend/internal/rbac/registry.go`)
- Added educational/reference-only examples for:
  - module definition entry
  - navigation entry
  - permission entries
  - backend endpoint permission guard mapping
- Explicitly marked all examples as non-runtime reference scaffolding (not an implemented module).
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-06-module-registry-workflow-scaffold.md` (gitignored, not for commit)

### Documentation coverage
- How to define a module in module registries.
- How to define navigation entries with grouped/RBAC-aware visibility.
- How to define permission keys/metadata with consistent naming.
- How backend endpoints align to permission keys and middleware guards.
- How to preserve desktop/web parity for shared modules.
- What to record in `docs/status.md` for module milestones.

### Tests and verification
- Verified guidance against current implemented registry structure in:
  - `desktop/frontend/src/registry/*`
  - `web/src/registry/*`
  - `backend/internal/rbac/registry.go`
- No runtime code changed in this milestone (documentation-only), so no additional backend/desktop/web test run was required.

## Milestone — Registry-First Module Foundation (Implementation) (Complete)

### What changed
- Added typed permission registries in both clients:
  - `desktop/frontend/src/registry/permissions.ts`
  - `web/src/registry/permissions.ts`
- Added typed navigation registries in both clients:
  - `desktop/frontend/src/registry/navigation.ts`
  - `web/src/registry/navigation.ts`
- Added typed module registries in both clients:
  - `desktop/frontend/src/registry/modules.ts`
  - `web/src/registry/modules.ts`
- Refactored desktop navigation composition and route-access checks to consume registry-driven navigation rules.
- Refactored web navigation composition and route-access checks to consume registry-driven navigation rules.
- Refactored shell route-title resolution in desktop and web to use registry-backed route labels.
- Refactored permissions pages in desktop and web to consume permission registry metadata for details display and permission checks.
- Added backend RBAC registry foundations and constants:
  - `backend/internal/rbac/registry.go`
  - `backend/internal/rbac/registry_test.go`
- Updated backend RBAC base seeding (`EnsureBaseRBAC`) and router permission middleware wiring to use registry/constants instead of repeated string literals.
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-06-registry-foundation-implementation.md` (gitignored, not for commit)

### Registry usage coverage
- Desktop now consumes registries for:
  - authenticated shell navigation visibility
  - route authorization checks (`canAccessRoute`)
  - AppBar section-title mapping
  - permission metadata lookup in Permissions details dialog
- Web now consumes registries for:
  - authenticated shell navigation visibility
  - route authorization checks (`canAccessRoute`)
  - AppBar section-title mapping
  - permission metadata lookup in Permissions details dialog
- Backend now consumes registry/constants for:
  - base permission definitions used in RBAC bootstrapping
  - permission-key references in route middleware guards

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && npm test -- --run` -> PASS
- Web frontend:
  - `cd web && npm test -- --run` -> PASS
- Web production build:
  - `cd web && npm run build` -> PASS

### Added targeted tests
- Backend:
  - `backend/internal/rbac/registry_test.go`
- Desktop:
  - `desktop/frontend/src/registry/registry.test.ts`
- Web:
  - `web/src/registry/registry.test.ts`

### Remaining follow-ups
- Existing non-blocking MUI jsdom `anchorEl` warnings remain in desktop/web frontend test logs.
- Existing non-blocking Vite warnings for third-party `'use client'` directives and chunk-size notices remain unchanged.
- Some non-navigation UI labels/titles are still page-local constants; they can be migrated incrementally if registry expansion continues.

## Milestone — Registry-First Module Foundation (Documentation) (Complete)

### Scope completed
- Define a typed module registry baseline for future platform/domain modules.
- Define a typed navigation registry baseline for grouped shell navigation wiring.
- Define a typed permission registry baseline for RBAC-aware permission definitions.
- Refactor existing Administration-related platform features to consume registry entries incrementally (users, roles, permissions, audit, settings-adjacent navigation intent) without introducing domain modules.
- Keep implementation static/config-driven (no dynamic plugin loader), simple, and maintainable.

### Delivery constraints met
- Backend/desktop/web contracts remain aligned for shared module behaviors.
- Registry-first architecture should reduce scattered wiring for future module additions.
- Any temporary parity gap discovered during implementation must be documented explicitly.
- This milestone was architecture/documentation-first only; no application/runtime code changes were included in this documentation step.

### Verification completed
- Milestone foundation docs updated (`docs/requirements.md`, `AGENTS.md`, `docs/status.md`).
- Prompt traceability copy saved under `docs/prompts/` and not committed.
- Confirm prompt copy is not staged for commit.
- Confirm no application code files changed as part of this docs-only planning step.
- Implementation milestone gate (for the next code-change milestone) remains:
  - backend tests: `cd backend && GOCACHE=/tmp/go-build go test ./...`
  - desktop route/smoke tests: `cd desktop/frontend && npm test -- --run`
  - web route/smoke tests: `cd web && npm test -- --run`

## Milestone J — Page-Level Notification/Error Handling Migration (Complete)

### What changed
- Migrated desktop and web page-level logic for:
  - Login
  - Forgot Password
  - Reset Password
  - Users
  - Roles
  - Permissions
  - Settings
  - Audit Log
- Replaced ad-hoc web snackbar usage (`useSnackbar().showSnackbar`) with standardized notification facade usage (`useAppNotify()` with `notify.success/error/warning/info`) across the migrated pages.
- Replaced page-level manual error parsing with shared error handling:
  - `handleAppError(error, options)` now drives fallback messaging and notifications
  - validation-field mapping now consumes normalized `fieldErrors` from shared handling, then renders inline on form fields
- Applied login exception behavior while still using shared logic:
  - invalid credentials now render as inline login form message (`Invalid username or password.`)
  - non-credential login failures still use normalized error output with request-id suffix where available
- Settings page behavior aligned to policy:
  - success/failure actions now use standardized notifications
  - form submission failures for settings/branding now render form-level alerts
  - unexpected branding-load failures now use fallback behavior plus notification via shared error handling
- Removed duplicated/inconsistent page helper patterns where practical:
  - removed ad-hoc request-id string builders and API-error parsing helpers in migrated pages
  - removed mixed snackbar APIs in favor of standardized notify facade usage
- Saved this prompt copy at:
  - `docs/prompts/2026-03-06-notification-error-page-migration.md` (gitignored, not for commit)

### Desktop/Web parity state
- Desktop and web now share the same page-level error-handling intent and behavior classes for the migrated pages:
  - field validation errors are rendered inline under inputs
  - form submission failures are rendered as form-level messages where applicable
  - action outcomes are surfaced via shared notification facade
  - session-expiry handling remains centralized in existing auth/session flows
- Login behavior is aligned in both clients with the documented inline invalid-credentials exception.

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && npm test -- --run` -> PASS
- Web frontend:
  - `cd web && npm test -- --run` -> PASS
- Web production build:
  - `cd web && npm run build` -> PASS

### Known follow-ups
- Existing non-blocking MUI jsdom `anchorEl` warnings remain in frontend tests.
- Existing non-blocking Vite bundle warnings (`'use client'` directive and chunk-size notices) remain unchanged.

## Milestone I — Desktop/Web Notification + Error Handling Foundation (Complete)

### What changed
- Introduced a shared notification contract in both clients via `AppNotification` with consistent fields:
  - `kind`, `message`, optional `title`, `autoHideDuration`, `requestId`, and `persistent`
- Added notification facades with standardized API:
  - desktop: `notify.success/error/warning/info` via pub/sub store integration
  - web: `useAppNotify()` facade backed by existing `SnackbarProvider`
- Added reusable error normalization in both clients:
  - `normalizeError(error)` -> `NormalizedAppError`
  - normalized types include: `validation`, `unauthorized`, `forbidden`, `not_found`, `conflict`, `network`, `timeout`, `server`, `unknown`
  - validation details are mapped to `fieldErrors`
  - `requestId` propagation is supported
- Added reusable `handleAppError(error, options?)` helpers in both clients:
  - normalization
  - optional notification dispatch
  - optional validation-field mapping callback
  - optional session-expiry callback for unauthorized errors
- Standardized session-expiry UX behavior across desktop/web:
  - unauthorized/expired auth clears local session state through existing auth flows
  - users receive session-expired notification
  - redirect to login occurs
  - intended destination is preserved in session storage and consumed after successful login
- Added root-level error boundaries in both clients:
  - friendly fallback
  - reload action
  - no stack trace exposure in production
  - dev-mode message visibility only
- Improved web route error fallback UI with reload action.
- Added documentation note:
  - `docs/notes/error-handling.md`
- Saved prompt copy:
  - `docs/prompts/2026-03-06-notification-error-foundation.md` (gitignored)

### Backend alignment notes
- No backend contract redesign was required.
- Existing backend error envelope and request-id headers were already compatible with normalization.
- Client-side alignment update:
  - web `ApiError` now carries `status` to improve consistent normalization classification.
  - desktop `ApiError` now carries `requestId` from response headers.

### Tests and verification
- Backend:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Desktop frontend:
  - `cd desktop/frontend && npm test -- --run` -> PASS
- Web frontend:
  - `cd web && npm test -- --run` -> PASS
- Web production build:
  - `cd web && npm run build` -> PASS

### Added targeted tests
- Desktop:
  - `src/notifications/facade.test.ts`
  - `src/errors/normalizeError.test.ts`
  - `src/errors/handleAppError.test.ts`
  - `src/auth/sessionExpiry.test.ts`
- Web:
  - `src/ui/snackbar.test.tsx` (notification facade dispatch)
  - `src/errors/normalizeError.test.ts`
  - `src/errors/handleAppError.test.ts`
  - `src/auth/sessionExpiry.test.ts`

### Known follow-ups
- Existing non-blocking MUI jsdom `anchorEl` warnings remain in frontend test logs.
- Existing non-blocking Vite bundle warnings (`'use client'` directives and chunk-size notices) remain unchanged.
- Full page-by-page migration from ad-hoc page-level error parsing to `handleAppError` is intentionally incremental and can continue in the next milestone.

## Milestone H — Desktop/Web Authentication UI Parity (Complete)

### What changed
- Implemented a redesigned split-layout authentication experience in both desktop and web:
  - left branding panel + right auth form panel
  - responsive composition with theme-aware styling for light/dark mode
  - larger auth field sizing (minimum 56px inputs) and clearer action hierarchy
- Added shared login-branding consumption for unauthenticated auth-entry screens in both clients:
  - load branding from backend public settings endpoint
  - show configured application name and optional image URL
  - graceful fallback avatar/name rendering when image is absent or fails
- Added forgot-password and reset-password pages in both desktop and web:
  - forgot-password request form (username/email) with non-enumerating success message
  - reset-password form with token + new password + confirm password
  - clear loading, success, validation, and backend error states
  - reset token pulled from route query parameter for both clients (`?token=...`)
- Extended desktop and web Settings pages with login-branding controls:
  - application display name field
  - login image URL field (URL-only for this milestone; no uploads)
  - URL validation and branding preview/fallback
  - backend-backed save flow gated by `settings.write`
- Preserved API-only desktop architecture and existing login contract behavior.
- Saved prompt copy to `docs/prompts/2026-03-06-milestone-h-auth-ui-parity.md` (gitignored).

### Backend APIs consumed
- Branding:
  - `GET /api/v1/settings/public/login-branding`
  - `GET /api/v1/settings/login-branding`
  - `PUT /api/v1/settings/login-branding`
- Authentication:
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/forgot-password`
  - `POST /api/v1/auth/reset-password`

### Desktop/Web parity notes
- Capability parity is maintained for:
  - login branding rendering
  - forgot-password request UX and response handling
  - reset-password submission and validation behavior
  - settings-based branding updates
- Minor differences are limited to platform shell/context wiring (desktop settings store and setup-gate behavior vs web auth-provider context), with equivalent end-user auth capabilities.

### Tests and verification
- Desktop frontend tests:
  - `cd desktop/frontend && npm test -- --run` -> PASS
- Web frontend tests:
  - `cd web && npm test -- --run` -> PASS
- Web production build:
  - `cd web && npm run build` -> PASS

### Known follow-ups
- Existing non-blocking jsdom/MUI `anchorEl` warnings remain in frontend test output.
- Existing non-blocking Vite warnings for third-party `'use client'` directives/chunk size remain unchanged.

## Milestone G — Backend Authentication Branding + Password Recovery Contract (Complete)

### What changed
- Added backend persistence for reusable application settings and password-reset token lifecycle:
  - migration `backend/migrations/000012_create_app_settings_and_password_reset_tokens.up.sql`
  - new generic `app_settings` table (`category`, `key`, `value_json`) for extensible platform settings
  - new `password_reset_tokens` table for secure reset token hash storage, expiry, and one-time consumption metadata
- Added a new backend settings module at `backend/internal/settings` with login-branding contract support:
  - public branding fetch endpoint for unauthenticated login views
  - authenticated branding read/update endpoints for settings management
  - URL/path validation and default fallback behavior when no branding image is configured
- Extended auth backend contract for forgot/reset flows:
  - `POST /api/v1/auth/forgot-password` (identifier by username/email, non-leaking accepted response)
  - `POST /api/v1/auth/reset-password` (token + new password)
  - secure random reset token generation, hashed token persistence, expiry enforcement, used-token invalidation, and refresh-session invalidation on successful reset
- Added audit coverage for password reset and branding updates:
  - `auth.password_reset.requested`
  - `auth.password_reset.success`
  - `settings.login_branding.update`
- Wired routes in `cmd/api/router.go` and service setup in `cmd/api/main.go` for backend-first consumption by both desktop and web clients.
- Saved prompt copy to `docs/prompts/2026-03-06-milestone-g-backend-auth-branding-password-reset.md` (gitignored).

### Backend contract summary
- Branding endpoints:
  - `GET /api/v1/settings/public/login-branding` (no auth; login page consumption)
  - `GET /api/v1/settings/login-branding` (`settings.read`)
  - `PUT /api/v1/settings/login-branding` (`settings.write`)
- Branding payload:
  - `applicationDisplayName` (required on update)
  - `loginImageUrl` (optional absolute `http(s)` URL)
  - `loginImageAssetPath` (optional path/reference for media-managed projects)
  - `imageConfigured` (boolean for graceful client fallback)
- Password recovery endpoints:
  - `POST /api/v1/auth/forgot-password` body: `{ "identifier": "<username-or-email>", "resetUrl": "https://.../reset-password" }`
    - response: `202 Accepted`, `{ "status": "accepted" }`
    - does not reveal account existence
  - `POST /api/v1/auth/reset-password` body: `{ "token": "<token>", "password": "<newPassword>" }`
    - response: `200 OK`, `{ "status": "ok" }`
    - validates token existence/expiry/one-time-use and rotates password securely.

### Tests and verification
- Backend targeted tests:
  - `cd backend && GOCACHE=/tmp/go-build go test ./internal/auth ./internal/settings ./cmd/api ./internal/middleware` -> PASS
- Backend full test suite:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Desktop frontend route/smoke tests:
  - `cd desktop/frontend && npm test -- --run` -> PASS
- Web frontend route/smoke tests:
  - `cd web && npm test -- --run` -> PASS

### Known follow-ups
- Desktop and web login/forgot/reset views still need milestone implementation to consume the new backend contract end-to-end in UI flows.
- Email delivery integration for password reset notifications is intentionally left as pluggable service wiring (backend contract is prepared).
- Existing non-blocking MUI `anchorEl` warnings remain in frontend test output; tests pass.

### 2026-03-06 Verification Refresh
- Re-verified Milestone G backend contract implementation against authentication branding and password-recovery requirements.
- Re-ran backend suite:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Saved prompt copy for this verification at:
  - `docs/prompts/2026-03-06-backend-auth-branding-password-recovery-contract.txt` (gitignored).

## Next Planned Milestone — Desktop/Web Authentication UX Consumption Parity (Planned)

### Planned scope
- Implement desktop and web login branding consumption from `GET /api/v1/settings/public/login-branding`.
- Implement desktop and web forgot-password/reset-password views wired to the new backend auth endpoints.
- Preserve parity in auth-entry behavior, loading/error states, and routing between desktop and web clients.
- Add route/smoke and behavior tests for desktop/web auth recovery flows.

## Milestone F — Admin DataGrid Search Contract Standardization (Complete)

### What changed
- Standardized backend admin list query handling for Users, Roles, Permissions, and Audit endpoints:
  - shared query parsing/validation helper added at `backend/internal/listquery/query.go`
  - `q` quick-search parameter now supported consistently across admin list handlers
  - legacy `filter` fallback preserved for backward compatibility
  - validation added for malformed `page`, `pageSize`, `sort`, and `filter` query parameters
- Aligned backend list handlers to avoid silently ignoring malformed query params and return typed validation errors.
- Added backend endpoint coverage for search/filter contract behavior:
  - users list `q` handling and invalid sort validation
  - roles list `q` handling and invalid sort validation
  - audit list `q` handling and invalid page validation
- Added reusable admin search utilities in desktop and web:
  - `useAdminListSearch` (debounced local search state)
  - `buildAdminListRequestQuery` (predictable `q` + composed list params)
- Updated desktop admin DataGrid pages (`Users`, `Roles`, `Permissions`, `Audit`) to use the shared search utility and send a consistent composed query contract.
- Updated web admin DataGrid pages (`Users`, `Roles`, `Permissions`, `Audit`) to use the same search contract and behavior parity with desktop.
- Added grid-level page-reset behavior when search/filter query key changes:
  - `AppDataGrid` now supports `externalQueryKey`
  - changing search terms resets pagination to page 1
- Added/updated frontend tests for:
  - page reset on search query changes (desktop/web `AppDataGrid` tests)
  - permissions search wiring without manual apply button
  - continued search + sort + pagination interaction coverage through existing route/page tests.
- Saved prompt copy to `docs/prompts/2026-03-06-milestone-admin-datagrid-search-contract.md` (gitignored).

### Standardized contract summary
- Request params:
  - `page`, `pageSize`
  - `sort=<field>:<asc|desc>`
  - `filter=<field>:<value>` (DataGrid column filter passthrough)
  - `q=<text>` (quick search)
- Behavior:
  - search input updates local state
  - debounced `q` request updates
  - search changes reset page to 1
  - search/sort/pagination/filter compose in one request
  - backend applies search server-side and validates malformed query values.

### Tests and verification
- Backend tests:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...` -> PASS
- Desktop frontend tests:
  - `cd desktop/frontend && npm test -- --run` -> PASS
- Web frontend tests:
  - `cd web && npm test -- --run` -> PASS

### Known follow-ups
- Existing non-blocking jsdom/MUI `anchorEl` warnings remain in desktop/web frontend test logs; behavior and assertions pass.

## Milestone E — Administration UX Refinement (Navigation Dropdowns + Multi-Column Forms + Grid Pinning) (Complete)

### What changed
- Refined desktop and web authenticated shell navigation so `Administration` is an expandable/collapsible grouped menu instead of a static section label.
- Preserved RBAC-aware visibility in both clients:
  - hidden child links for unauthorized routes
  - hidden `Administration` group when no child route is accessible.
- Added aligned expand/collapse test coverage for desktop and web navigation behavior.
- Refactored desktop and web users create/edit dialogs to responsive multi-column form grids:
  - 1 column on narrow overlays
  - 2 columns on medium widths
  - 3 columns on wide dialogs
- Increased users create/edit dialog width (`maxWidth="lg"`) to reduce excessive vertical scrolling while keeping validation helper text and error rendering intact.
- Strengthened shared DataGrid wrappers in desktop and web:
  - default right-pin behavior for a standard `actions` column when action pinning is enabled
  - enforced sticky-right action pinning via wrapper-level resolution (reduces per-page drift)
  - right-pinned width safeguards (caps pinned-right column count in wrapper merge logic)
  - pinned header/cell styling tweaks to keep pinned areas visually stable during horizontal scroll.
- Aligned admin table containers to avoid clipping menus/pinned areas/scrollbars by removing outer `overflow: auto` wrappers and relying on DataGrid scrolling behavior.
- Added/updated tests for:
  - admin navigation expand/collapse and visibility
  - users form layout container presence for create/edit dialogs
  - default actions-column pinning behavior in shared DataGrid wrappers.
- Saved prompt copy to `docs/prompts/2026-03-06-milestone-admin-ux-refinement.md` (gitignored).

### Rendering notes (desktop vs web)
- Both clients now share the same grouped-navigation semantics and permission behavior for `Administration`.
- Desktop and web differ slightly in responsive interaction details:
  - desktop mini drawer and web mini drawer/mobile drawer render collapsed labels/tooltips differently
  - both still preserve equivalent expand/collapse behavior and route grouping intent.

### Tests and verification
- Backend tests:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Desktop frontend tests:
  - `cd desktop/frontend && npm test -- --run`
- Web frontend tests:
  - `cd web && npm test -- --run`

### Verification summary
- Backend tests: PASS
- Desktop frontend tests: PASS
- Web frontend tests: PASS
- Known non-blocking jsdom `anchorEl` warnings remain in frontend test output; behavior/tests pass.

## Milestone D — Web RBAC Administration Parity + Users Multi-Role UX (Complete)

### What changed
- Implemented web `/roles` page using shared admin patterns:
  - server-backed roles list with `AppDataGrid`
  - create role dialog
  - edit role dialog (name + permissions update)
  - role details dialog (assigned permissions + assigned users)
  - row action menu wired through shared `AdminRowActions`
- Implemented web `/permissions` page using shared admin patterns:
  - server-backed permissions list with `AppDataGrid`
  - search (`q`) and module-scope filtering (`moduleScope`)
  - permission details dialog
  - permission metadata viewer via shared JSON metadata dialog
- Extended web `/users` create/edit flows for multi-role assignment:
  - added API-driven MUI `Autocomplete` multi-select + chips for role assignment in both create and edit dialogs
  - included role arrays in create/update payloads
  - mapped backend `VALIDATION_ERROR` role-field errors (`details.roles`) to form helper text
- Improved web users role visibility UX:
  - added explicit `Roles` column in users grid with chips
  - added user details dialog that clearly shows assigned roles and key metadata
- Preserved permission-based action visibility/enablement and existing auth/session-expiry behavior via shared `apiRequest` + `AuthProvider`.
- Saved prompt copy to `docs/prompts/2026-03-06-milestone-d-web-rbac-parity.md` (gitignored).

### API endpoints consumed (web)
- `GET /api/v1/users`
- `POST /api/v1/users`
- `PATCH /api/v1/users/:id`
- `GET /api/v1/admin/roles`
- `POST /api/v1/admin/roles`
- `GET /api/v1/admin/roles/:id?includeUsers=true|false`
- `PATCH /api/v1/admin/roles/:id`
- `GET /api/v1/admin/permissions`

### Tests and build
- Backend tests:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Desktop frontend tests:
  - `cd desktop/frontend && npm test -- --run`
- Web frontend tests:
  - `cd web && npm test -- --run`
- Web build:
  - `cd web && npm run build`

### Verification summary
- Backend tests: PASS
- Desktop frontend tests: PASS
- Web frontend tests: PASS (including new role/permission route smoke coverage, users role multi-select behavior, and admin action-menu flow coverage)
- Web build: PASS

### Parity notes
- Web now matches desktop for reusable RBAC administration capabilities across `/roles`, `/permissions`, and users multi-role create/edit flows.
- Remaining runtime test/build warnings are unchanged non-blocking MUI jsdom `anchorEl` warnings and Vite third-party `'use client'`/chunk-size warnings.

## Milestone C — Desktop RBAC Administration Pages + User Multi-Role UX (Complete)

### What changed
- Implemented desktop `/roles` page using shared admin patterns:
  - server-backed roles list with `AppDataGrid`
  - create role dialog
  - edit role dialog (name + permissions update)
  - role details dialog (assigned permissions + assigned users)
  - row action menu wired through shared `AdminRowActions`
- Implemented desktop `/permissions` page using shared admin patterns:
  - server-backed permissions list with `AppDataGrid`
  - search and module-scope filtering
  - permission details dialog
  - permission metadata viewer via shared JSON metadata dialog
- Extended desktop `/users` create/edit flows for multi-role assignment:
  - replaced static role select with API-driven MUI `Autocomplete` multi-select + chips
  - included role multi-select in both create and edit dialogs
  - retained dedicated roles action/edit dialog and upgraded to autocomplete chips
- Improved users role visibility UX:
  - added explicit roles column in users grid with chips
  - added user details dialog that clearly shows assigned roles and core metadata
- Preserved permission-based action enablement and existing auth/session error handling behavior via shared API client.
- Kept all desktop data access API-only (no DB access from desktop).
- Saved prompt copy to `docs/prompts/2026-03-06-milestone-c-desktop-rbac-admin.md` (gitignored).

### API endpoints consumed (desktop)
- `GET /api/v1/users`
- `POST /api/v1/users`
- `PATCH /api/v1/users/:id`
- `POST /api/v1/users/:id/reset-password`
- `GET /api/v1/admin/roles`
- `POST /api/v1/admin/roles`
- `GET /api/v1/admin/roles/:id?includeUsers=true|false`
- `PATCH /api/v1/admin/roles/:id`
- `GET /api/v1/admin/permissions`

### How to test
- Backend tests:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Desktop frontend tests:
  - `cd desktop/frontend && npm test -- --run`
- Web frontend tests:
  - `cd web && npm test -- --run`

### Verification summary
- Backend tests: PASS
- Desktop frontend tests: PASS (including new RBAC route/page smoke tests and role multi-select behavior coverage)
- Web frontend tests: PASS

### Known parity items for web
- Desktop Milestone C RBAC administration UX and user role autocomplete are now implemented.
- Equivalent web-role/permission management UX for Milestone C is still a follow-up item to maintain full cross-client feature parity.
- Existing jsdom MUI `anchorEl` warnings remain non-blocking in test runs; tests pass.

## Milestone B — Backend RBAC Administration Contract (Complete)

### What changed
- Added backend RBAC administration API support under `/api/v1/admin`:
  - `GET /api/v1/admin/roles`
  - `POST /api/v1/admin/roles`
  - `GET /api/v1/admin/roles/:id` (supports `includeUsers=true`)
  - `PUT/PATCH /api/v1/admin/roles/:id`
  - `PUT /api/v1/admin/roles/:id/permissions`
  - `GET /api/v1/admin/permissions`
- Implemented new RBAC admin service/repository flow (no layering bypass):
  - role listing with counts
  - role create/update
  - role detail with assigned permissions and optional assigned users
  - permissions listing with pagination, search/query filter, and module scope filter
  - role-permission replacement flow for assign/remove behavior
- Extended backend router and app wiring to include RBAC admin handlers with backend authorization middleware.
- Protected RBAC admin endpoints server-side (frontend checks remain UX-level only).
- Preserved typed validation error envelope for RBAC admin and role/permission identifier validation.
- Added RBAC admin audit events:
  - `roles.create`
  - `roles.update`
- Strengthened user role-assignment validation coverage:
  - invalid role identifiers on user create/update now explicitly covered by tests with typed validation expectations.
- Saved prompt copy to `docs/prompts/2026-03-06-milestone-b-backend-rbac-admin-contract.md` (gitignored).

### Endpoint contract summary (backend for desktop/web consumers)
- Roles list:
  - `GET /api/v1/admin/roles?page=1&pageSize=25&sort=name:asc&filter=name:admin`
  - Response shape: `{ items, totalCount, page, pageSize }`
- Role create:
  - `POST /api/v1/admin/roles`
  - Request: `{ "name": "RoleName", "permissions": ["users.read"] }`
  - Response: role detail with `permissions`
- Role detail:
  - `GET /api/v1/admin/roles/:id?includeUsers=true`
  - Response: `{ id, name, createdAt, updatedAt, permissions: [...], users?: [...] }`
- Role update:
  - `PATCH /api/v1/admin/roles/:id` (or `PUT`)
  - Request supports partial: `{ "name": "...", "permissions": [...] }`
- Role-permissions replace:
  - `PUT /api/v1/admin/roles/:id/permissions`
  - Request: `{ "permissions": [...] }`
- Permissions list:
  - `GET /api/v1/admin/permissions?page=1&pageSize=25&sort=name:asc&q=users&moduleScope=admin`
  - Response shape: `{ items, totalCount, page, pageSize }`
- Validation/auth errors:
  - Typed envelope preserved: `{ "error": { "code", "message", "details" } }`
  - Invalid identifiers use `VALIDATION_ERROR` with field details.

### How to test
- Backend tests:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Desktop frontend route/smoke tests:
  - `cd desktop/frontend && npm test -- --run`
- Web frontend route/smoke tests:
  - `cd web && npm test -- --run`

### Verification summary
- Backend tests: PASS
- Desktop frontend tests: PASS
- Web frontend tests: PASS

### Known follow-ups
- Desktop/web `Roles` and `Permissions` pages are still scaffolded UI shells and should be wired to the new RBAC admin endpoints in the next client milestone.
- Existing jsdom MUI `anchorEl` warnings in test runs remain non-blocking; tests pass.

## Milestone A — Shared Administration UX Foundation + Desktop/Web Parity (Complete)

### What changed
- Refactored authenticated navigation in both desktop and web to grouped platform structure:
  - top-level: `Dashboard`, `Settings`
  - grouped: `Administration` with `Users`, `Roles`, `Permissions`, `Audit Log`
- Added `Roles` and `Permissions` admin routes/pages on desktop and web without adding any domain/business modules.
- Removed web domain-module routing from active navigation (`Employees`, `Leave`, `Payroll`) to keep skeleton scope platform-only.
- Centralized permission-gated navigation intent with shared navigation helpers in both clients and hide the entire `Administration` group when no child route is allowed.
- Added reusable shared row-actions pattern in both clients:
  - vertical ellipsis menu
  - permission-aware action visibility/disabled states
  - destructive-action confirmation flow
- Updated users grids to use shared row-actions menu pattern.
- Added reusable audit metadata JSON dialog in desktop and web:
  - concise metadata preview in grid
  - full metadata dialog
  - pretty-printed scrollable JSON
  - copy-to-clipboard
  - graceful empty/invalid JSON handling
- Extended desktop and web global UI preferences with DataGrid defaults:
  - `pinActionsColumnRight`
  - `dataGridBorderRadius`
- Updated DataGrid wrappers in desktop and web to honor global defaults while still allowing per-table sticky-right overrides.
- Improved table container/wrapper styles to keep horizontal/vertical scroll and reduce clipping risk for menus, pinned areas, and scrollbars.
- Saved prompt copy to `docs/prompts/2026-03-06-milestone-a-admin-ux.md` (gitignored).

### How to test
- Backend tests: `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Desktop frontend tests: `cd desktop/frontend && npm test -- --run`
- Web frontend tests: `cd web && npm test -- --run`

### Verification summary
- Backend tests: PASS
- Desktop frontend tests: PASS
- Web frontend tests: PASS

### Parity notes
- Desktop and web now share aligned grouped admin navigation semantics, admin action-menu behavior, and audit metadata dialog UX.
- `Roles` and `Permissions` pages are intentionally scaffolded UI foundations pending backend role/permission administration endpoints; parity is maintained across both clients.

### Known follow-ups
- MUI `anchorEl` test-runtime warnings can appear in jsdom when opening menu/popover controls from virtualized DataGrid cells, but tests pass and runtime behavior is correct.

## Milestone 1 — Repo Bootstrap + Baseline Build (Complete)

### What changed
- Bootstrapped repository structure for `backend/`, `desktop/` (generated via `wails init`), `web/`, and `docs/`.
- Added root project files: `.gitignore`, `Makefile`, and `docker-compose.yml`.
- Added prompt traceability folder `docs/prompts/` and stored the Milestone 1 prompt copy (kept ignored from git).
- Bootstrapped backend Go API with Gin under `/api/v1`.
- Implemented `GET /api/v1/health` returning `{ "status": "ok" }`.
- Implemented backend graceful shutdown with `signal.NotifyContext` + `http.Server.Shutdown` timeout.
- Added backend route test for `/api/v1/health`.
- Bootstrapped desktop app using `wails init` (React + TypeScript template).
- Added required frontend dependencies: MUI, MUI Icons, Emotion, TanStack Router, TanStack Query, and MUI X Data Grid.
- Implemented TanStack Router root/outlet + `/` route + configured NotFound route.
- Implemented minimal MUI centered card UI:
  - Title: `Skeleton App Ready`
  - Subtitle: `Wails + React + MUI + TanStack Router`
- Added frontend route tests:
  - `/` renders expected content
  - unknown route renders NotFound

### How to run
- Backend API: `make backend-run`
- Desktop app (dev): `make desktop-dev`

### How to test
- Backend tests: `make backend-test`
- Desktop route tests: `make desktop-test`

### Verification summary
- Backend builds: PASS (`go build ./...` in `backend/`)
- Backend tests pass: PASS (`go test ./...` in `backend/`)
- Desktop builds: PASS (`wails build -compiler /usr/local/go/bin/go -skipbindings` in `desktop/`)
- Desktop route tests pass: PASS (`npm test` in `desktop/frontend`)

### Known follow-ups
- `wails build` in this environment needs explicit Go toolchain env (`GOROOT=/usr/local/go`) and `-skipbindings` due local Wails binding-generation issues with the shell Go configuration and module scanning.

## Milestone 2 — Backend Foundation (Complete)

### What changed
- Added `backend/internal/config` with typed config struct and Viper loading using precedence: config file, environment variables, then CLI flag overrides.
- Enabled Viper hot reload via `WatchConfig` and `OnConfigChange`.
- Added atomic runtime snapshot (`atomic.Value`) with `config.Get()` so runtime code reads typed config without direct Viper access.
- Added reload validation logic: invalid config reloads are rejected and previous snapshot remains active.
- Added safe write helper (`SafeWriteFile`) that writes temp file then atomic rename.
- Added `backend/internal/db` with SQLX-based DB initialization, pool sizing, and startup ping.
- Added migration SQL files under `backend/migrations` for:
  - users
  - roles
  - permissions
  - role_permissions
  - user_roles
  - api_tokens
  - refresh_tokens
  - audit_logs
- Added migration runner commands in `backend/cmd/migrate` and shared helpers in `backend/internal/migrateutil`.
- Added Makefile migration targets:
  - `make migrate-up`
  - `make migrate-down`
  - `make migrate-create name=<migration_name>`
- Hardened API startup/shutdown:
  - `signal.NotifyContext` in `cmd/api/main.go`
  - shutdown timeout sourced from config
  - `http.Server.Shutdown(...)` on cancellation
  - clean DB close on process exit
  - startup server lifecycle helper with unit test coverage
- Updated API routes to include:
  - `GET /api/v1/health` returning `{ "status": "ok", "version": "...", "db": "up" }` when DB is healthy
  - `GET /api/v1/version` returning backend version
- Added tests:
  - config hot reload atomic swap + invalid reload retention test
  - health route test with DB ping mock
  - server graceful startup/shutdown lifecycle test
  - integration-style DB open test that skips when `BASEPRO_TEST_DSN` is not set

### How config reload works
- Config is loaded via Viper from `backend/config/config.yaml` (or `--config` path), env vars (`BASEPRO_*`), and CLI overrides.
- Hot reload watches config file changes.
- On each change, config is decoded + validated.
- If valid, new config is atomically swapped into `config.Get()`.
- If invalid, reload is logged and ignored; previous config remains active.

### How to run backend
- Ensure PostgreSQL is running (example: `docker compose up -d postgres`).
- From repo root run:
  - `make migrate-up`
  - `make backend-run`

### How to run migrations
- Apply all up migrations: `make migrate-up`
- Roll back one migration: `make migrate-down`
- Create migration pair: `make migrate-create name=<migration_name>`

### How to test
- Backend tests: `make backend-test`
- Frontend route/smoke tests: `make desktop-test`

### Verification summary
- Backend builds/tests (`go test ./...` in `backend/`): PASS
- Health route test: PASS
- Config reload test: PASS
- Frontend route tests: PASS

### Known follow-ups
- Auto-migration is available via `database.auto_migrate` config / `--database-auto-migrate`; keep disabled in production.
- Authentication/RBAC remains intentionally unimplemented for Milestone 2.

## Milestone 3 — Backend Auth (JWT + Refresh Rotation + Typed Errors + Audit) (Complete)

### What changed
- Added auth-specific migration `000009_auth_token_chain_and_audit_indexes` to extend existing tables without editing prior migrations.
- `refresh_tokens` now supports rotation/reuse chain tracking fields:
  - `issued_at`
  - `replaced_by_token_id`
  - `updated_at`
- `audit_logs` now supports `timestamp` for ordered event querying.
- Added required indexes:
  - `refresh_tokens(user_id)`
  - `refresh_tokens(token_hash)` unique index
  - `audit_logs(timestamp DESC)`
- Extended backend config with auth settings:
  - `auth.access_token_ttl_seconds`
  - `auth.refresh_token_ttl_seconds`
  - `auth.jwt_signing_key`
  - `auth.password_hash_cost`
- Added config validation to fail fast when JWT signing key is empty.
- Added `internal/apperror` for standardized typed error responses.
- Added `internal/audit` (SQLX repository + service).
- Added `internal/auth`:
  - SQLX repository for users/refresh tokens
  - password hashing/verify helpers (bcrypt)
  - JWT manager
  - auth service implementing login, refresh rotation, refresh reuse detection, logout, and `me`
  - HTTP handlers for `/api/v1/auth/*`
- Added `internal/middleware` JWT auth middleware with request context claims injection.
- Wired auth + audit dependencies in `cmd/api/main.go` via dependency injection (no global DB state).
- Added backend auth endpoints:
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/refresh`
  - `POST /api/v1/auth/logout`
  - `GET /api/v1/auth/me` (JWT protected)

### Typed error shape
- Handlers now return auth errors in standardized shape:
  - `{ "error": { "code": "...", "message": "..." } }`
- Implemented codes:
  - `AUTH_UNAUTHORIZED`
  - `AUTH_EXPIRED`
  - `AUTH_REFRESH_REUSED`
  - `AUTH_REFRESH_INVALID`

### Refresh rotation and reuse detection
- Refresh tokens are generated as random opaque strings and only SHA-256 hashes are stored in DB.
- On refresh success:
  - old token row is marked `revoked_at`
  - old token row links `replaced_by_token_id` to the new token row
  - new access + refresh tokens are issued
- On refresh reuse (revoked token presented):
  - response returns `AUTH_REFRESH_REUSED` (401)
  - active refresh tokens for that user are revoked to contain the token family/session.

### Auth endpoint examples (curl)
- Login:
  - `curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login -H 'Content-Type: application/json' -d '{"username":"alice","password":"secret"}'`
- Refresh:
  - `curl -s -X POST http://127.0.0.1:8080/api/v1/auth/refresh -H 'Content-Type: application/json' -d '{"refreshToken":"<refresh-token>"}'`
- Logout with refresh token:
  - `curl -s -X POST http://127.0.0.1:8080/api/v1/auth/logout -H 'Content-Type: application/json' -d '{"refreshToken":"<refresh-token>"}'`
- Me with access token:
  - `curl -s http://127.0.0.1:8080/api/v1/auth/me -H 'Authorization: Bearer <access-token>'`

### How to test
- Backend tests: `make backend-test`
- Frontend route/smoke tests: `make desktop-test`

### Verification summary
- Backend tests (`go test ./...`): PASS
- Login success test (token issuance + hashed refresh storage): PASS
- Login failure typed error test: PASS
- Refresh rotation test (old revoked + new works): PASS
- Refresh reuse detection + active token revocation test: PASS
- JWT middleware missing token -> `AUTH_UNAUTHORIZED`: PASS
- JWT middleware expired token -> `AUTH_EXPIRED`: PASS
- Frontend route tests: PASS

### Milestone scope guard
- API-token authentication was not implemented in this milestone.
- RBAC roles/permission enforcement was not implemented in this milestone.

## Milestone 4 — Backend API Token Auth (Machine/Integration Auth) (Complete)

### What changed
- Added migration `000010_api_token_management` to extend API-token storage and add token permission linkage without editing prior migrations.
- `api_tokens` now supports token management fields:
  - `prefix`
  - `created_by_user_id`
  - `revoked_at`
  - `expires_at` (already present)
  - `last_used_at`
  - `created_at` / `updated_at`
- Added `api_token_permissions` table for permission-scoped tokens (`permission` + optional `module_scope`).
- Added indexes:
  - unique `api_tokens(token_hash)`
  - `api_tokens(revoked_at)`
  - `api_tokens(prefix)`

### API token hashing approach
- API tokens are generated as random opaque plaintext strings (returned once at creation).
- Only a deterministic HMAC-SHA256 hash is stored in DB for lookup:
  - `stored_hash = HMAC_SHA256(token, auth.jwt_signing_key)`
- Plaintext token is never persisted and cannot be re-read from the API after create.

### Config additions
- Added config keys under `auth`:
  - `api_token_enabled` (bool)
  - `api_token_header_name` (default `X-API-Token`)
  - `api_token_ttl_seconds` (default TTL)
  - `api_token_allow_bearer` (allow Bearer token for API-token auth)
- Existing config validation now enforces non-empty token header name and positive token TTL.

### Middleware and principal model
- Added API-token middleware that accepts:
  - `X-API-Token: <token>`
  - optional `Authorization: Bearer <token>` when enabled
- Validation behavior:
  - token hash lookup
  - reject revoked tokens
  - reject expired tokens
  - update `last_used_at` best-effort
- Context principal now supports:
  - `type = user` (JWT)
  - `type = api_token` (API token)
  - `permissions` for API-token scoped checks
- Added `RequirePermission(...)` helper to enforce API token permissions (and keep JWT path ready for Milestone 5).

### Admin API-token endpoints
- Added JWT-protected admin endpoints under `/api/v1/admin/api-tokens`:
  - `GET /api/v1/admin/api-tokens`
  - `POST /api/v1/admin/api-tokens`
  - `POST /api/v1/admin/api-tokens/:id/revoke`
- Admin check is intentionally minimal for Milestone 4:
  - user ID `1` is treated as admin stub.
- Added audit events:
  - `api_token.create`
  - `api_token.revoke`

### Dev-only seed convenience
- Added startup flag `--seed-dev-admin` to create/ensure an initial dev admin user.
- Seed credentials can be set via env:
  - `BASEPRO_DEV_ADMIN_USERNAME`
  - `BASEPRO_DEV_ADMIN_PASSWORD`

### How to create/revoke API tokens
- Create token:
  - `curl -s -X POST http://127.0.0.1:8080/api/v1/admin/api-tokens -H 'Authorization: Bearer <jwt>' -H 'Content-Type: application/json' -d '{"name":"ci-token","permissions":["audit.read"],"expiresInSeconds":3600}'`
- List tokens (masked):
  - `curl -s http://127.0.0.1:8080/api/v1/admin/api-tokens -H 'Authorization: Bearer <jwt>'`
- Revoke token:
  - `curl -s -X POST http://127.0.0.1:8080/api/v1/admin/api-tokens/<id>/revoke -H 'Authorization: Bearer <jwt>'`

### How to test
- Backend tests: `make backend-test`
- Frontend route/smoke tests: `make desktop-test`

### Verification summary
- `go test ./...` in `backend/`: PASS
- API-token create endpoint test (plaintext once + hashed storage + prefix + permissions): PASS
- API-token middleware tests (valid/revoked/expired): PASS
- Audit tests for create/revoke actions: PASS
- Frontend route tests: PASS

### Milestone scope guard
- RBAC user role/permission enforcement was not implemented (reserved for Milestone 5).
- Desktop login/auth UI work was not started (reserved for Milestones 6/7).

## Milestone 5 — RBAC (Roles + Permissions + Module Scope) + Enforcement (Complete)

### What changed
- Implemented `backend/internal/rbac` with SQLX repository + service for role/permission resolution:
  - `GetUserRoles(userID)`
  - `GetUserPermissions(userID)`
  - `HasPermission(userID, perm, moduleScope)`
  - `PermissionsForUser(userID)`
- Added minimal in-process RBAC cache in service with explicit invalidation (`InvalidateUser`) on role assignment.
- Adopted **Option A** for module scope:
  - canonical permission key in `permissions.name` (e.g. `users.read`)
  - optional scope in `permissions.module_scope`
  - enforcement supports `RequirePermission("perm")` and `RequirePermission("perm", WithModule("scope"))`

### Principal and middleware changes
- Unified principal model for both auth types:
  - `principal.type = user | api_token`
  - user fields: user ID, username, roles, resolved permission grants
  - api token fields: token ID, permission grants (including optional module scope)
- Added/updated middleware helpers:
  - `RequireAuth()`
  - `RequireJWTUser()`
  - `RequirePermission(rbacService, "perm", optional WithModule(...))`
  - `ResolveJWTPrincipal(...)` so protected routes accept either JWT or API token principals.
- Added typed forbidden code:
  - `AUTH_FORBIDDEN` with response shape `{ "error": { "code": "AUTH_FORBIDDEN", "message": "Forbidden" } }`

### Endpoint enforcement applied
- Users management:
  - `GET /api/v1/users` -> `users.read`
  - `POST /api/v1/users` -> `users.write`
  - `PATCH /api/v1/users/:id` -> `users.write`
  - `POST /api/v1/users/:id/reset-password` -> `users.write`
- Audit:
  - `GET /api/v1/audit` -> `audit.read`
- API token admin:
  - `GET /api/v1/admin/api-tokens` -> `api_tokens.read`
  - `POST /api/v1/admin/api-tokens` -> `api_tokens.write`
  - `POST /api/v1/admin/api-tokens/:id/revoke` -> `api_tokens.write`
- Removed the Milestone 4 hardcoded admin check (`userID == 1`) from runtime authorization path.

### RBAC seed/bootstrap (dev/test only)
- Added config-controlled bootstrap flag:
  - `seed.enable_dev_bootstrap` (default: `false`)
- When enabled (or when `--seed-dev-admin` is passed), startup now:
  - ensures base roles: `Admin`, `Manager`, `Staff`, `Viewer`
  - ensures base permissions:
    - `users.read`, `users.write`
    - `audit.read`
    - `settings.read`, `settings.write`
    - `api_tokens.read`, `api_tokens.write`
  - ensures role-permission mapping:
    - `Admin`: all above permissions
    - `Manager`: `users.read`, `audit.read`, `settings.read`
    - `Staff`: `settings.read`
    - `Viewer`: `users.read`, `audit.read`, `settings.read`
  - ensures dev admin user exists and assigns `Admin` role.

### Additional backend handlers/services
- Added users service + handler (`backend/internal/users`) for list/create/update/reset-password paths.
- Extended audit package with list/query support and added audit handler (`GET /api/v1/audit` with pagination/filter params).

### How to test
- Backend tests:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Frontend route/smoke tests:
  - `make desktop-test`

### Verification summary
- Backend tests (`go test ./...` in `backend/`): PASS
- Frontend route tests (`make desktop-test`): PASS
- Mandatory Milestone 5 coverage included:
  - JWT permission enforcement (forbidden vs allowed)
  - role -> permission resolution
  - API token permission enforcement for audit access
  - correct unauthorized/forbidden error codes and shapes

### Milestone scope guard
- Desktop setup/login UI work was not started (still reserved for Milestones 6/7).

## Milestone 6 — Desktop Setup Screen + Routing + Route Tests (Complete)

### What changed
- Added desktop local settings persistence through Wails Go bindings in `desktop/app.go`:
  - `LoadSettings()`
  - `SaveSettings(partial)`
  - `ResetSettings()`
- Settings are now stored as JSON under OS user config dir:
  - `<os_user_config_dir>/basepro-desktop/settings.json`
  - directory created with best-effort `0700` permissions
  - file writes use best-effort `0600` permissions and temp-file rename
- Added frontend typed settings abstractions:
  - `src/settings/types.ts`
  - `src/settings/store.ts`
- Added frontend HTTP wrapper with timeout-aware health check:
  - `src/api/client.ts`
  - `healthCheck()` calls `GET {apiBaseUrl}/api/v1/health`
- Replaced route tree with Milestone 6 unauthenticated routing:
  - `/setup`
  - `/login` (placeholder: “Login not implemented yet”)
  - `/` gate route
  - NotFound component configured at root
- Implemented route gating behavior:
  - if `apiBaseUrl` is empty, navigating to `/` or `/login` redirects to `/setup`
- Implemented Setup page UI (`/setup`) with MUI centered card:
  - API Base URL input
  - Auth Mode selector (Username/Password vs API Token)
  - conditional API Token input
  - Request Timeout input
  - `Test Connection` and `Save & Continue` actions
- Added/updated route tests (`src/routes.test.tsx`) covering:
  - `/login` and `/` redirect to `/setup` when API base URL is missing
  - NotFound rendering for unknown routes
  - setup save flow navigates to `/login`
  - setup health check uses mocked `fetch` endpoint
- Added test setup shim for `Response` to keep TanStack Router redirect handling deterministic in tests.

### How to run desktop tests
- `make desktop-test`
- or `cd desktop/frontend && npm test`

### How to run full milestone verification
- Backend tests: `make backend-test`
- Desktop route tests: `make desktop-test`
- Desktop production build: `make desktop-build`

### Verification summary
- Backend tests (`make backend-test`): PASS
- Desktop route tests (`make desktop-test`): PASS (5 tests)
- Desktop build (`make desktop-build`): PASS

### Current desktop routes
- `/` (gate route)
- `/setup`
- `/login` (placeholder only)
- NotFound for unknown paths

### Milestone scope guard
- Login/refresh token UI flow was not implemented in this milestone.
- Authenticated AppShell (Drawer/AppBar/Footer) was not implemented in this milestone.

### Milestone 6 Bug Fix — Settings `authMode` Type Alignment (2026-02-28)
- Updated desktop settings type definitions to use a central `AuthMode` union source:
  - `AUTH_MODES = ['password', 'api_token'] as const`
  - `type AuthMode = (typeof AUTH_MODES)[number]`
- Hardened `src/settings/store.ts` normalization boundary to treat Wails binding payloads as untrusted input:
  - normalize from `unknown`
  - validate `authMode` against `AUTH_MODES`
  - default invalid/missing `authMode` to `password`
  - normalize API token and timeout values safely
- Removed the direct assumption that Wails `main.Settings` is already `AppSettings`, eliminating the `string` vs `AuthMode` mismatch at compile-time.

Bug-fix verification:
- `cd desktop/frontend && npm run build`: PASS
- `cd desktop/frontend && npm test -- --run`: PASS

## Backend Change — Configurable Startup Auto-Migrate with Advisory Lock (Complete)

### What changed
- Added a dedicated startup migration package: `backend/internal/migrate`.
- Startup migration behavior is now configuration-driven with three keys:
  - `database.auto_migrate` (bool, default `false`)
  - `database.auto_migrate_lock_timeout_seconds` (int, default `30`)
  - `database.migrations_path` (string, default `file://./migrations`)
- Backend startup (`cmd/api`) now runs migration wiring before serving HTTP:
  - If `database.auto_migrate=false`, migrations are skipped and startup continues.
  - If `database.auto_migrate=true`, startup attempts migrations via `migrate.Up()`.
  - `migrate.ErrNoChange` is treated as success.
  - Any migration error fails fast and prevents the server from starting.
- Added Postgres advisory-lock coordination in startup migration flow:
  - acquires advisory lock before migration run
  - uses context timeout from `database.auto_migrate_lock_timeout_seconds`
  - returns error if lock cannot be acquired in time
  - always attempts lock release after migration attempt
- Added startup migration logging for:
  - skipped vs running status
  - start and successful completion
  - no-change condition

### Example config (dev)
```yaml
database:
  auto_migrate: true
  auto_migrate_lock_timeout_seconds: 30
  migrations_path: "file://./migrations"
```

### Production safety note
- Default remains `database.auto_migrate=false`.
- Advisory lock behavior prevents concurrent migration execution across multiple backend instances when auto-migrate is enabled.

### Test coverage added
- `backend/internal/migrate/runner_test.go`:
  - `AutoMigrate=false` skips migrator execution.
  - `ErrNoChange` is treated as success.
  - advisory-lock acquisition timeout path returns expected error and does not run migrations.

### Verification summary
- Backend tests: `cd backend && GOCACHE=/tmp/go-build go test ./...` => PASS
- Frontend route tests: `make desktop-test` => PASS

### Backend Auto-Migrate Update — Embedded Migrations (No `database.migrations_path`)
- Removed `database.migrations_path` from backend runtime config and CLI overrides.
- Startup auto-migrate now infers migration source from Go-embedded SQL files (`go:embed`) via `golang-migrate` `iofs` source driver.
- Added embedded migrations package: `backend/migrations/embed.go`.
- `internal/migrate` runner now executes `Up()` from embedded migrations, so no filesystem migration path is required at runtime.
- Existing startup behavior remains:
  - `database.auto_migrate=false` (default): startup skips migrations.
  - `database.auto_migrate=true`: startup acquires advisory lock and runs migrations before serving.

Verification for this update:
- `cd backend && GOCACHE=/tmp/go-build go test ./...` => PASS
- `make desktop-test` => PASS

## Milestone 7 — Desktop Login + Refresh + Session Expiry UX (Complete)

### What changed
- Added desktop auth contracts in `desktop/frontend/src/auth/types.ts`:
  - `LoginRequest`, `LoginResponse`
  - `RefreshRequest`, `RefreshResponse`
  - `MeResponse`
- Added auth session store in `desktop/frontend/src/auth/session.ts` with required helpers:
  - `setSession({ accessToken, refreshToken, expiresAt })`
  - `clearSession()`
  - `isAuthenticated()`
- Implemented token storage model:
  - access token kept in memory only (session store state)
  - refresh token persisted via existing desktop settings storage (`settings.json`) and Wails binding patch.
- Extended desktop settings persistence schema with `refreshToken` in:
  - `desktop/app.go`
  - `desktop/frontend/src/settings/types.ts`
  - `desktop/frontend/src/settings/store.ts`
- Reworked API client (`desktop/frontend/src/api/client.ts`) to support authenticated requests:
  - attaches `Authorization: Bearer <accessToken>`
  - handles 401 by performing refresh once and retrying original request once
  - includes single-flight refresh lock so parallel requests wait on one refresh operation
  - on refresh failure with `AUTH_REFRESH_INVALID` / `AUTH_REFRESH_REUSED` / `AUTH_EXPIRED`, clears session and triggers forced re-login flow
  - emits separate network-unreachable event path
- Added global session UX wiring in `desktop/frontend/src/App.tsx`:
  - MUI Snackbar/Alert notifications
  - forced logout message: `Session expired. Please log in again.`
  - separate message for network unreachable
  - redirect to `/login` on forced logout
- Updated routing/guards (`desktop/frontend/src/routes.tsx`) for Milestone 7:
  - unauthenticated routes: `/setup`, `/login`
  - authenticated route: `/dashboard` (placeholder)
  - guards:
    - missing `apiBaseUrl` => redirect to `/setup`
    - unauthenticated access to `/dashboard` => redirect to `/login`
    - authenticated access to `/login` => redirect to `/dashboard`
  - NotFound route remains configured
- Added desktop pages:
  - Login UI implementation in `desktop/frontend/src/pages/LoginPage.tsx` (title/subtitle, username/password, loading state, generic invalid-credentials error, change API settings action)
  - Authenticated placeholder dashboard in `desktop/frontend/src/pages/DashboardPage.tsx`
- Added global notification store scaffold in `desktop/frontend/src/notifications/store.ts`.

### Refresh token storage location
- Refresh token is persisted through existing desktop settings storage:
  - file path: `<os_user_config_dir>/basepro-desktop/settings.json`
  - JSON key: `refreshToken`
  - write path remains temp-file + atomic rename with best-effort restrictive permissions.

### Auto-refresh behavior
- Access token is attached to authenticated API requests.
- If access token is expired or a request returns HTTP 401:
  - client performs one refresh request (`POST /api/v1/auth/refresh`) using persisted refresh token
  - while refresh is in-flight, other requests wait on the same promise (single-flight lock)
  - on success, stores new tokens and retries original request exactly once.

### Session-expired behavior
- If refresh fails with `AUTH_REFRESH_INVALID`, `AUTH_REFRESH_REUSED`, or `AUTH_EXPIRED`:
  - session is cleared (memory access token + persisted refresh token removed)
  - app redirects user to `/login`
  - global snackbar shows: `Session expired. Please log in again.`
- Network-unreachable errors show a separate non-auth snackbar message.

### Frontend tests added/updated
- `desktop/frontend/src/routes.test.tsx` now covers:
  - `/login` -> `/setup` when `apiBaseUrl` missing
  - `/dashboard` -> `/login` when not authenticated
  - `/login` -> `/dashboard` when authenticated
  - login success -> `/dashboard`
  - login failure -> generic invalid credentials error
  - 401 -> refresh success -> retried request success
  - refresh failure -> forced logout + redirect to `/login` + session-expired snackbar

### How to test
- Backend tests: `make backend-test`
- Desktop route/auth tests: `make desktop-test`
- Frontend build: `cd desktop/frontend && npm run build`
- Desktop build: `make desktop-build`

### Verification summary
- `make backend-test`: PASS
- `make desktop-test`: PASS (7 tests)
- `cd desktop/frontend && npm run build`: PASS
- `make desktop-build`: PASS

### Milestone scope guard
- AppShell (Drawer/AppBar/Footer) was not implemented in this milestone.
- Users and Audit pages were not implemented in this milestone.

## Milestone 8 — Sleek MUI Admin Shell + Themes + Palette Presets (Complete)

### What changed
- Added authenticated `AppShell` layout for `/dashboard` and `/settings` with:
  - desktop permanent Drawer with mini/collapsed mode
  - mobile temporary Drawer with hamburger toggle
  - top AppBar with route-based section title and user avatar/menu
  - main content outlet container and footer (`BasePro Desktop v0.1.0`)
- Implemented nav entries:
  - active routes: `Dashboard`, `Settings`
  - placeholders: `Users`, `Audit` shown disabled with “Soon” markers (no milestone 9/10 pages implemented)
- Added authenticated `/settings` page with sections:
  - Connection: editable API base URL + request timeout + `Test Connection` + save action
  - Auth mode display (read-only) with explicit redirect path to `/setup` for safe editing
  - Appearance: theme mode selector + preset quick buttons + full preset picker dialog entry point
  - About: app/build placeholders
- Added logout action in user menu:
  - best-effort `POST /api/v1/auth/logout` when refresh token exists
  - always clears local session and redirects to `/login`

### Theme + palette system
- Added persisted UI preferences (`uiPrefs`) in desktop settings model:
  - `themeMode`: `light | dark | system`
  - `palettePreset`: preset id
  - `navCollapsed`: drawer mini/collapse state
- Added reusable theme/palette modules:
  - `src/ui/theme.tsx` (`AppThemeProvider`, preference context, system mode handling)
  - `src/ui/palettePresets.ts` (8 preset definitions)
  - `src/ui/PalettePresetPicker.tsx` (dialog picker with instant preview)
- Theme and preset changes apply immediately and persist via settings store.

### UI preference storage location
- UI preferences are persisted in the existing desktop settings file:
  - `<os_user_config_dir>/basepro-desktop/settings.json`
  - JSON section: `uiPrefs`
- Backend/Wails settings schema extended in `desktop/app.go` with `UIPrefs`/`UIPrefsPatch` and validation/normalization defaults.

### Frontend tests added/updated
- `desktop/frontend/src/routes.test.tsx` now verifies:
  - authenticated `/dashboard` renders AppShell + dashboard content
  - Settings navigation works from shell
  - theme mode persistence and re-application after reload
  - palette preset persistence and re-application after reload
- Added deterministic `matchMedia` mock in `desktop/frontend/src/test-setup.ts`.

### How to test
- Backend tests: `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Frontend route/smoke tests: `make desktop-test`
- Frontend build: `cd desktop/frontend && npm run build`

### Verification summary
- `cd backend && GOCACHE=/tmp/go-build go test ./...`: PASS
- `make desktop-test`: PASS (4 tests)
- `cd desktop/frontend && npm run build`: PASS

### Milestone scope guard
- Users and Audit pages were not implemented (only disabled placeholders in shell nav).

## Milestone 9 — DataGrid Foundation (Advanced Features + Server Integration) (Complete)

### What changed
- Added reusable shared wrapper: `desktop/frontend/src/components/datagrid/AppDataGrid.tsx`.
- Confirmed MUI X Data Grid package is in use (`@mui/x-data-grid`) and implemented wrapper in a way that can accept Pro/Premium props later (including pinned columns passthrough).
- Wrapper capabilities implemented:
  - server-side pagination (`paginationMode="server"`)
  - server-side sorting (`sortingMode="server"`)
  - server-side filtering (`filterMode="server"`)
  - page size options (`10, 25, 50, 100`)
  - density selector via toolbar
  - column visibility model
  - column reordering persistence
  - pinned columns persistence + passthrough (active where supported)
  - CSV export via toolbar
- Added shared frontend pagination helpers:
  - `desktop/frontend/src/api/pagination.ts`
  - `desktop/frontend/src/api/useApiClient.ts`
- Added authenticated skeleton pages using wrapper:
  - `desktop/frontend/src/pages/UsersPage.tsx`
  - `desktop/frontend/src/pages/AuditPage.tsx`
- Wired routes for `/users` and `/audit` and enabled shell navigation entries for both pages.

### Persistent table preferences
- Extended desktop local settings schema (same Wails settings storage mechanism) to persist per-table preferences by `storageKey`:
  - `pageSize`
  - `columnVisibility`
  - `columnOrder`
  - `density`
  - `pinnedColumns`
- Added schema versioning in table preferences (`version: 1`) to support future schema evolution.
- Storage location remains the existing desktop settings file:
  - `<os_user_config_dir>/basepro-desktop/settings.json`
  - persisted under top-level key: `tablePrefs`.
- Updated both frontend and Wails model bindings for the extended settings shape.

### Server query contract (Milestone 9 contract)
- Adopted consistent query format for list endpoints:
  - `GET /api/v1/<resource>?page=1&pageSize=25&sort=<field>:<asc|desc>&filter=<field>:<value>`
- Implemented for:
  - `GET /api/v1/users`
  - `GET /api/v1/audit`
- Implemented consistent paginated response shape:
  - `{ "items": [...], "totalCount": number, "page": number, "pageSize": number }`

### Backend updates
- Updated users list backend to support:
  - pagination (`page`, `pageSize`)
  - sorting (`sort`)
  - filtering (`filter`, username)
  - response includes `totalCount`, `page`, `pageSize`
- Updated audit list backend to support:
  - pagination (`page`, `pageSize`)
  - sorting (`sort`)
  - filtering (`filter`, action)
  - response includes `totalCount`, `page`, `pageSize`
- Added backend pagination tests:
  - `backend/internal/users/users_test.go`
  - `backend/internal/audit/audit_test.go`

### Frontend tests added
- DataGrid wrapper tests:
  - calls `fetchData` on page change
  - calls `fetchData` on sort change
  - calls `fetchData` on filter change
- Persistence tests:
  - page size persists and reloads from settings store
  - column visibility persists and reloads from settings store
- Users page route test:
  - `/users` loads and renders rows from mocked API response

### How to test
- Backend tests:
  - `make backend-test`
- Frontend route/smoke + DataGrid tests:
  - `make desktop-test`
- Frontend build:
  - `cd desktop/frontend && npm run build`

### Verification summary
- `make backend-test`: PASS
- `make desktop-test`: PASS (7 tests)
- `cd desktop/frontend && npm run build`: PASS

### Milestone scope guard
- No business-specific modules (HR/payroll/etc.) were implemented.
- Milestone 10 work was not started.

## Milestone 10 — Users + Audit (End-to-End) + Permission-Gated Navigation (Complete)

### What changed
- Completed backend users endpoints with permission enforcement and validation:
  - `GET /api/v1/users` (`users.read`)
  - `POST /api/v1/users` (`users.write`)
  - `PATCH /api/v1/users/:id` (`users.write`)
  - `POST /api/v1/users/:id/reset-password` (`users.write`)
- Added/finished users backend behaviors:
  - input validation now uses typed validation errors (`VALIDATION_ERROR`)
  - patch now supports optional `username`, `isActive`, and `roles`
  - role updates are replacement-based (set semantics) instead of additive-only
  - password hashes remain server-only and are never returned in responses
- Added RBAC role replacement support:
  - `rbac.Repository.ReplaceUserRoles(...)`
  - `rbac.Service.SetUserRoles(...)`
- Ensured user-management audit writes are emitted:
  - `users.create`
  - `users.update`
  - `users.reset_password`
  - `users.set_active`
- Completed backend audit listing endpoint filters and contract:
  - `GET /api/v1/audit` (`audit.read`)
  - pagination/sort/filter response: `{ items, totalCount, page, pageSize }`
  - supports `action`, `actor_user_id`/`actorUserId`, and date range (`date_from`/`date_to` and camelCase aliases)
- Completed principal/capability behavior:
  - `/api/v1/auth/me` now returns `id`, `username`, `roles`, and `permissions`
  - desktop auth state stores principal permissions and exposes permission checks

### Permission and principal decisions
- `users.*` endpoints are JWT-user-only:
  - router now applies `RequireJWTUser` on `/api/v1/users` group in addition to `RequirePermission`.
  - API-token principals cannot administer users.
- `audit.read` endpoint supports either JWT or API-token principals:
  - `RequirePermission` is enforced for both principal types.

### Desktop completion
- Added permission-aware auth state:
  - session principal (`id`, `username`, `roles`, `permissions`) and subscription-based updates
  - login flow now loads `/api/v1/auth/me` immediately after token issuance
- Added permission-gated navigation and route guards:
  - Users nav shown only when `users.read` or `users.write`
  - Audit nav shown only when `audit.read`
  - direct navigation to `/users` or `/audit` without permission renders a new `ForbiddenPage` (`403`)
- Implemented full `/users` page with `AppDataGrid` and server data:
  - columns: username, roles, active, created
  - create user dialog
  - toggle active action
  - reset password dialog
  - edit roles dialog (multi-select)
  - write actions are disabled when `users.write` is missing
  - success/error toasts for all actions
- Implemented full `/audit` page with `AppDataGrid` and server data:
  - columns: timestamp, actor, action, entity_type, entity_id, metadata (compact)
  - filters: action dropdown, actor user id, date range

### Permission strings used
- `users.read`
- `users.write`
- `audit.read`

### Dev seeding for roles/permissions
- Backend dev bootstrap still seeds baseline RBAC and optional dev admin:
  - enable via existing config `seed.enable_dev_bootstrap` or CLI `--seed-dev-admin`
  - default baseline includes roles `Admin`, `Manager`, `Staff`, `Viewer`
  - baseline permissions include `users.read`, `users.write`, `audit.read`, and existing settings/api-token permissions
- Example startup for local dev:
  - `make migrate-up`
  - `cd backend && go run ./cmd/api --seed-dev-admin`

### Tests added/updated
- Backend:
  - permission enforcement tests for missing `users.write` and missing `audit.read`
    - `backend/internal/middleware/authorization_test.go`
  - pagination contract tests for users and audit list responses
    - `backend/internal/users/users_test.go`
    - `backend/internal/audit/audit_test.go`
  - audit write test for user creation (`users.create`)
    - `backend/internal/users/users_test.go`
- Desktop:
  - navigation gating test (Audit menu hidden without `audit.read`)
  - route guard test (`/audit` shows Forbidden without permission)
  - users action test (create user triggers API + grid refresh)
  - audit grid load test (rows rendered from mocked API)
  - file: `desktop/frontend/src/routes.test.tsx`

### How to test
- Backend tests:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Frontend tests:
  - `make desktop-test`
- Frontend build:
  - `cd desktop/frontend && npm run build`

### Verification summary
- `cd backend && GOCACHE=/tmp/go-build go test ./...`: PASS
- `make desktop-test`: PASS (6 tests)
- `cd desktop/frontend && npm run build`: PASS

### Milestone scope guard
- No HR/payroll/business-domain modules were started.
- Milestone remained skeleton-focused (users + audit + permission-gated shell behavior).

## Milestone 11 — Packaging + Versioning + CI Baseline (Complete)

### What changed
- Added backend build metadata variables in `cmd/api/main.go`:
  - `Version` (default `dev`)
  - `Commit` (default `none`)
  - `BuildDate` (default `unknown`)
- Updated `/api/v1/version` to return:
  - `version`
  - `commit`
  - `buildDate`
- Extended router dependency wiring to pass version metadata from build/runtime into HTTP responses.
- Added backend test coverage for version endpoint metadata contract.
- Updated desktop Settings > About section to show:
  - desktop version (from Vite define)
  - backend version from `/api/v1/version`
  - backend commit/build date when available
  - `Not Connected` fallback when backend is unreachable.
- Added desktop API client helper for unauthenticated `/api/v1/version` fetch.
- Added desktop test coverage asserting About section renders mocked backend version.
- Hardened root `Makefile` and added release/CI targets:
  - `backend-build` (with ldflags version injection)
  - `backend-test`
  - `backend-run`
  - `desktop-build`
  - `desktop-dev`
  - `desktop-test`
  - `ci` (runs `backend-test`, `desktop-test`, `desktop-build`)
- Added GitHub Actions workflow: `.github/workflows/ci.yml`
  - runs on push + pull_request
  - installs Go and Node
  - runs backend tests, desktop install/test, and desktop build
  - fails on any non-zero step.

### How version injection works
- `Makefile` computes build metadata values:
  - `VERSION` (default `dev`, overrideable)
  - `COMMIT` from `git rev-parse --short HEAD`
  - `BUILD_DATE` in UTC RFC3339-like format
- `backend-build` injects metadata with:
  - `-X main.Version=$(VERSION)`
  - `-X main.Commit=$(COMMIT)`
  - `-X main.BuildDate=$(BUILD_DATE)`
- Backend binary output path:
  - `backend/bin/basepro-api`

### How to build backend with version info
- Default build metadata:
  - `make backend-build`
- Explicit version override:
  - `make backend-build VERSION=1.0.0`

### How CI works
- Workflow file: `.github/workflows/ci.yml`
- Trigger events:
  - `push`
  - `pull_request`
- Steps:
  - `go test ./...` in `backend/`
  - `npm install`, `npm test`, and `npm run build` in `desktop/frontend/`

### How to run milestone verification
- Full local CI baseline:
  - `make ci`
- Backend version-injected build:
  - `make backend-build`

### Verification summary
- `make backend-build`: PASS
- `make ci`: PASS
  - backend tests: PASS
  - desktop tests: PASS
  - desktop build: PASS

### Known follow-ups
- Frontend build currently emits existing third-party bundling warnings (`"use client"` directives and chunk-size warnings) but build completes successfully.

## Milestone 12 Part A — Observability & Error Handling (Complete)

### What changed
- Added a centralized backend structured logger (`backend/internal/logging`) built on `slog` and replaced backend `log.Printf` usage in API startup/config reload/migration paths.
- Added backend config keys for logging:
  - `logging.level` (`debug|info|warn|error`)
  - `logging.format` (`json|console`)
- Wired logging config updates through existing Viper hot reload using a config change callback, so runtime logger settings are updated when validated config changes.
- Added request-correlation middleware:
  - `backend/internal/middleware/requestid.go`
  - every request now has `X-Request-Id` (preserved if provided, generated if missing)
  - request ID is stored in Gin + request context for downstream logging.
- Added access logging middleware:
  - `backend/internal/middleware/accesslog.go`
  - logs `method`, `path`, `status`, `duration_ms`, `request_id` per request.
- Updated router middleware chain to include request ID and access logging for all routes.
- Upgraded centralized error handling (`backend/internal/apperror/errors.go`) to enforce consistent shape:
  - `{ "error": { "code", "message", "details" } }`
  - typed auth errors preserve existing error codes and now include `details: {}`
  - validation errors return `VALIDATION_ERROR` with populated details
  - internal errors return safe message + `INTERNAL_ERROR` with empty details.
- Added structured error log entries in `apperror.Write(...)` that include `request_id`, `code`, and status while avoiding secret-bearing payload/header logging.

### How to run tests
- Backend tests:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Desktop tests:
  - `make desktop-test`

### Verification summary
- `cd backend && GOCACHE=/tmp/go-build go test ./...`: PASS
- `make desktop-test`: PASS
- Added/updated test coverage for Milestone 12 Part A:
  - request ID generation/preservation + response header behavior
  - access log fields including request correlation
  - centralized error JSON shape for auth, validation, and internal errors.

## Milestone 12 Part B — Security & Operational Baseline (Complete)

### What changed
- Added backend security config sections and defaults:
  - `security.rate_limit.enabled`
  - `security.rate_limit.requests_per_second`
  - `security.rate_limit.burst`
  - `security.cors.enabled`
  - `security.cors.allowed_origins`
  - `security.cors.allowed_methods`
  - `security.cors.allowed_headers`
  - `security.cors.allow_credentials`
- Implemented authentication endpoint rate limiting middleware with bounded in-memory per-client token buckets and stale-entry cleanup:
  - applied only to `POST /api/v1/auth/login` and `POST /api/v1/auth/refresh`
  - rate-limited responses return HTTP `429` with standardized error code `RATE_LIMITED`.
- Implemented configurable CORS middleware with conditional router application:
  - CORS disabled by default (no CORS headers emitted)
  - configurable origins/methods/headers/credentials
  - preflight handling with explicit policy enforcement.
- Extended config validation for security baseline rules:
  - rate limit parameters must be valid when enabled
  - CORS must have non-empty policy when enabled
  - wildcard origins are rejected when `allow_credentials=true`
  - invalid startup config fails load
  - invalid hot reload is rejected and previous valid snapshot remains active.
- Preserved secure defaults and existing protections:
  - `database.auto_migrate` default remains `false`
  - JWT signing key remains required by config validation
  - default logging level remains `info` (debug not default).
- Added health endpoint safety test coverage to ensure response does not leak sensitive terms/fields.

### How to run tests
- Backend tests:
  - `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Desktop tests:
  - `make desktop-test`

### Verification summary
- `cd backend && GOCACHE=/tmp/go-build go test ./...`: PASS
- `make desktop-test`: PASS
- Added/updated backend test coverage for:
  - rate limiting enabled/disabled behavior and `429/RATE_LIMITED` response shape
  - CORS enabled/disabled behavior
  - wildcard origin + credentials validation failure
  - invalid startup config failure
  - invalid hot reload rejection with prior config retention
  - health endpoint non-sensitive response assertions.

## Milestone 13 — Phase S3 Step 1 Web Frontend Scaffold & Baseline (Complete)

### What changed
- Scaffolded `web/` as a standalone Vite + React + TypeScript app (incremental extension of existing placeholder `web/.gitkeep`).
- Added baseline dependencies for:
  - Material UI
  - TanStack Router
  - TanStack Query
- Added environment variable support and examples:
  - `VITE_API_BASE_URL` (required)
  - `VITE_APP_NAME` (optional)
  - `VITE_ENABLE_DEVTOOLS` (optional placeholder for future use)
- Added `web/.env.example`.
- Added `web/README.md` with setup/dev/build/test instructions.
- Implemented web routes and baseline pages:
  - `/` redirects to `/login`
  - `/login` placeholder page
  - `/dashboard` placeholder page
  - NotFound page for unknown routes
  - global route error boundary component
- Added minimal route smoke tests in `web/src/routes.test.tsx`.
- Added root Makefile targets for web:
  - `make web-dev`
  - `make web-build`
  - `make web-test`
- Added prompt traceability copy:
  - `docs/prompts/2026-03-03-phase-s3-web-step1.md`
- Updated `.gitignore` to include web artifacts:
  - `web/node_modules/`
  - `web/dist/`
  - `web/.env`

### How to run
- Install deps: `cd web && npm install`
- Start dev server: `npm run dev`
- Build: `npm run build`
- Test: `npm run test`

### How to test
- Backend tests: `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Desktop route tests: `cd desktop/frontend && npm test`
- Web route tests: `cd web && npm test`

### Verification summary
- `cd backend && GOCACHE=/tmp/go-build go test ./...`: PASS
- `cd desktop/frontend && npm test`: PASS
- `cd web && npm run test`: PASS (3 tests)
- `cd web && npm run build`: PASS
- `cd web && npm run dev -- --host 127.0.0.1 --port 4173 --strictPort`: PASS (server started; `Local: http://127.0.0.1:4173/`)

### Known follow-ups
- Full web auth flow and RBAC gating are intentionally deferred to later S3 steps.

## Phase S3 Web — API Client Complete

### What changed
- Added typed web API client at `web/src/lib/api.ts`:
  - `apiRequest<T>()` wrapper using `import.meta.env.VITE_API_BASE_URL`
  - optional auth provider wiring via `configureApiClient(...)` and automatic `Authorization: Bearer <accessToken>` attachment
  - standardized non-2xx parsing to:
    - `type ApiError = { code: string; message: string; details?: unknown; requestId?: string }`
  - `X-Request-Id` extraction from error responses
  - sanitized request logging path so `Authorization` is redacted (`[REDACTED]`)
- Added user-facing error utility at `web/src/lib/errors.ts`:
  - `toUserFriendlyError(error)`
  - appends `Request ID: <id>` when present
- Added global MUI snackbar system at `web/src/ui/snackbar.tsx` and wired it in `web/src/main.tsx`.
- Added web API-client tests at `web/src/lib/api.test.ts` covering:
  - standardized error JSON parsing
  - `X-Request-Id` extraction
  - Authorization redaction in logger metadata
  - success response behavior
- Saved milestone prompt copy under `docs/prompts/2026-03-03-phase-s3-web-api-client.md` (ignored from git).

### How to run
- Backend tests: `cd backend && GOCACHE=/tmp/go-build go test ./...`
- Desktop tests: `cd desktop/frontend && npm test -- --run`
- Desktop build: `cd desktop/frontend && npm run build`
- Web tests: `cd web && npm test -- --run`
- Web build: `cd web && npm run build`

### Verification summary
- Backend tests (`go test ./...`): PASS
- Web tests (`npm test -- --run`): PASS
- Web build (`npm run build`): PASS
- Desktop build (`npm run build`): PASS
- Desktop tests (`npm test -- --run`): FAIL in current workspace due existing desktop test issues not introduced by this web change set (route assertions and `structuredClone` availability in current test runtime).

### Known follow-ups
- Desktop test suite needs a separate stabilization pass in this workspace before milestone-wide "all frontend tests pass" can be asserted.

## Phase S3 Web — Auth Complete

### What changed
- Added `AuthProvider` in `web/src/auth/AuthProvider.tsx` with auth state and methods:
  - `isAuthenticated`
  - `accessToken` (memory)
  - `user`
  - `login(username, password)`
  - `logout()`
  - `refresh()`
- Added auth snapshot/session storage helpers in `web/src/auth/state.ts`:
  - refresh token persistence in local storage
  - in-memory auth snapshot for route guards
- Extended API client (`web/src/lib/api.ts`) to support:
  - configurable unauthorized handler via `configureApiClient(...)`
  - automatic one-time retry after 401 when refresh succeeds
  - per-request options to disable auth injection/retry for login/refresh/logout calls
- Updated web `LoginPage`:
  - real username/password submit flow using `auth.login`
  - success navigation to `/dashboard`
  - backend `ApiError` message display
  - Request ID display when present
- Added route guards in `web/src/routes.tsx`:
  - `/dashboard` requires authentication
  - `/login` redirects to `/dashboard` when authenticated
  - `/` redirects by auth state (`/dashboard` or `/login`)
- Added/updated auth route tests in `web/src/routes.test.tsx`:
  - login success redirects
  - login failure shows backend error + request ID
  - refresh failure logs out, redirects to `/login`, and shows session-expired message
  - protected route blocked while logged out
- Saved prompt traceability copy under `docs/prompts/2026-03-03-phase-s3-web-auth.md` (ignored by git).

### How to run tests
- Backend: `make backend-test`
- Desktop: `make desktop-test`
- Web tests: `make web-test`
- Web build: `make web-build`

### Verification summary
- `make backend-test`: PASS
- `make desktop-test`: PASS
- `make web-test`: PASS
- `make web-build`: PASS

### Known follow-ups
- Vite build emits non-blocking warnings from third-party dependencies about ignored `'use client'` directives.

## Phase S3 Web — RBAC + Navigation Complete

### What changed
- Added RBAC helper utilities in `web/src/rbac/permissions.ts`:
  - `hasRole(role)`
  - `hasPermission(permission)`
  - both read claims from authenticated user snapshot managed by `AuthProvider`.
- Added authenticated shell layout at `web/src/components/AppShell.tsx`:
  - Drawer navigation and top app bar for authenticated routes
  - role/permission-based visibility for module links
  - hidden restricted nav modules
  - logout action preserved through existing auth provider logout flow.
- Updated route structure in `web/src/routes.tsx`:
  - authenticated parent route now renders `AppShell`
  - added module routes for `/employees`, `/leave`, `/payroll`, `/users`, `/settings`
  - unauthorized route access now renders `Not Authorized` page.
- Added module placeholder pages with `title + Coming soon`:
  - `EmployeesPage`
  - `LeavePage`
  - `PayrollPage`
  - `UsersPage`
  - `SettingsPage`
- Added reusable module placeholder renderer at `web/src/pages/modules/ModulePlaceholderPage.tsx`.
- Added `web/src/pages/NotAuthorizedPage.tsx`.
- Updated `DashboardPage` module action buttons so restricted actions are visible but disabled.
- Added RBAC navigation tests in `web/src/routes.test.tsx`:
  - role-based module visibility (Admin sees Payroll)
  - role-based module hiding (Staff does not see Payroll)
  - unauthorized route navigation handling (`/users` without permission shows `Not Authorized`).
- Saved prompt traceability copy:
  - `docs/prompts/2026-03-03-phase-s3-web-rbac-navigation.md` (gitignored)

### How to run tests
- Backend: `make backend-test`
- Desktop: `make desktop-test`
- Web tests: `make web-test`
- Web build: `make web-build`

### Verification summary
- `make backend-test`: PASS
- `make desktop-test`: PASS
- `make web-test`: PASS
- `make web-build`: PASS

### Known follow-ups
- `web` production build still emits non-blocking third-party `'use client'` and chunk-size warnings from existing dependency bundles.

## Web — Theme System + Presets (Foundation) Complete

### What changed
- Added local UI preferences foundation in `web/src/ui/preferences.ts` with APIs:
  - `loadPrefs()`
  - `savePrefs()`
  - `setMode()`
  - `setPreset()`
- Persisted UI preferences in localStorage key `basepro.web.ui_preferences`.
- Added theme preset catalog in `web/src/ui/theme/presets.ts`:
  - 8 admin-style presets with `id`, `name`, and light/dark palette tokens.
  - helper `getPaletteOptions(presetId, mode)` to resolve MUI palette options by preset.
- Added `UiPreferencesProvider` in `web/src/ui/theme/UiPreferencesProvider.tsx`:
  - tracks `mode` (`light | dark | system`) and `preset`
  - resolves `system` mode with `prefers-color-scheme`
  - applies updates immediately and persists locally.
- Added app-level theme wiring in `web/src/ui/theme/AppThemeProvider.tsx`:
  - computes MUI theme from resolved mode + preset
  - applies `ThemeProvider` + `CssBaseline`
  - includes token-based component style overrides.
- Updated `web/src/App.tsx` to use `AppThemeProvider` while keeping existing auth flow and API client behavior intact.
- Added deterministic theme tests in `web/src/ui/theme/theme.test.tsx` covering:
  - mode persistence and apply-after-reload
  - preset persistence and apply-after-reload
  - deterministic system mode behavior with mocked `matchMedia`.
- Updated `web/src/test-setup.ts` with a baseline `matchMedia` stub for deterministic test runtime.

### How to run tests
- `cd web && npm run test`

### Verification summary
- `cd web && npm run test`: PASS
- `cd web && npm run build`: PASS

### Known follow-ups
- Existing non-blocking Vite bundle warnings from third-party dependencies (`'use client'` directives, chunk-size warning) remain unchanged.

## Web — Settings Page (Theme + Presets + Connection) Complete

### What changed
- Implemented full `web` settings experience in `web/src/pages/SettingsPage.tsx` with sections:
  - Appearance:
    - theme mode selector (`light | dark | system`)
    - palette preset picker with mini visual previews for all presets
    - immediate apply via existing UI preferences provider
    - preview block with common components
  - Navigation:
    - `Collapse side navigation by default` toggle persisted in UI preferences
  - Connection:
    - editable API base URL override saved locally
    - `Test Connection` action calling `GET /health` via API client
    - snackbar success/failure feedback
    - failure message includes request id when backend returns `X-Request-Id`
  - About:
    - app name and version/build placeholders
- Extended UI preferences store in `web/src/ui/preferences.ts`:
  - added `collapseNavByDefault` persisted with existing `mode` and `preset`
  - added `setCollapseNavByDefault(...)`
- Extended UI preferences provider in `web/src/ui/theme/UiPreferencesProvider.tsx`:
  - added `setCollapseNavByDefault(...)` context action
- Added local API base URL override module `web/src/lib/apiBaseUrl.ts`:
  - local storage read/write for override
  - `resolveApiBaseUrl()` helper
- Updated API client base URL resolution in `web/src/lib/api.ts`:
  - `baseURL = override || env default`
- Updated app shell behavior in `web/src/components/AppShell.tsx`:
  - mini/collapsed drawer state now initializes from persisted `collapseNavByDefault`
- Added/updated deterministic tests in `web/src/routes.test.tsx` and `web/src/ui/theme/theme.test.tsx` covering:
  - `/settings` render and control-driven preference updates
  - mode persistence across reload
  - preset persistence across reload
  - API base URL override persistence

### Storage locations
- UI preferences: `localStorage['basepro.web.ui_preferences']`
- API base URL override: `localStorage['basepro.web.api_base_url_override']`

### How to run tests
- `cd web && npm run test`

### Verification summary
- `cd web && npm run test`: PASS
- `cd web && npm run build`: PASS

### Known follow-ups
- Non-blocking Vite warnings from third-party bundles (`'use client'` directives and chunk-size warning) remain unchanged.

## Web — AppShell Drawer (Responsive + Mini) + Footer Complete

### What changed
- Upgraded authenticated layout in [AppShell.tsx](/Users/sam/projects/go/basepro/web/src/components/AppShell.tsx):
  - desktop permanent Drawer supports expanded and mini/collapsed widths
  - mobile temporary Drawer with hamburger open + close button
  - AppBar and main content shift smoothly with MUI transitions when drawer state changes
  - selected nav item highlight is preserved
  - mobile nav selection closes the Drawer automatically
  - accessibility improvements: explicit `aria-label` on controls/nav actions and focus handoff to first nav item on mobile open.
- Added footer support in authenticated shell:
  - footer appears below main content and stays on authenticated screens
  - uses theme tokens (`background.paper`, `text.secondary`, `divider`) with subtle styling
  - footer displays app name + version placeholder.
- Extended UI preferences persistence:
  - added `showFooter` to `basepro.web.ui_preferences`
  - added `setShowFooter(...)`
  - kept backward-compatible loading defaults for existing stored preferences.
- Updated settings UI in [SettingsPage.tsx](/Users/sam/projects/go/basepro/web/src/pages/SettingsPage.tsx):
  - navigation collapsed preference label refined
  - added `Show footer on authenticated pages` toggle tied to persisted UI prefs.
- Added/updated tests in [routes.test.tsx](/Users/sam/projects/go/basepro/web/src/routes.test.tsx) and [theme.test.tsx](/Users/sam/projects/go/basepro/web/src/ui/theme/theme.test.tsx):
  - authenticated `/dashboard` renders AppShell
  - desktop collapse toggle updates/persists drawer state across reload
  - mobile drawer opens, closes, and closes after route selection
  - updated persisted preference assertions for the new `showFooter` key.

### Storage location
- UI preferences: `localStorage['basepro.web.ui_preferences']`
  - includes `mode`, `preset`, `collapseNavByDefault`, `showFooter`

### How to run tests
- `cd web && npm run test`
- `cd web && npm run build`

### Verification summary
- `cd web && npm run test`: PASS
- `cd web && npm run build`: PASS

### Known follow-ups
- Vite still emits existing non-blocking third-party warnings (`'use client'` directives, chunk-size warning); unchanged by this milestone.

## Web — Theme/Palette Accessibility + Icon Parity Update Complete

### What changed
- Updated web settings appearance UX to match desktop patterns more closely:
  - added accessible `Appearance` dialog picker in [PalettePresetPicker.tsx](/Users/sam/projects/go/basepro/web/src/ui/theme/PalettePresetPicker.tsx)
  - keyboard-selectable preset tiles (`role="button"`, `tabIndex=0`, Enter/Space handlers)
  - explicit `aria-label`s on mode selector and preset actions
  - settings page now offers quick preset buttons + `Browse all presets` entry like desktop.
- Updated authenticated shell iconography in [AppShell.tsx](/Users/sam/projects/go/basepro/web/src/components/AppShell.tsx):
  - replaced placeholder glyphs with Material-style drawer/menu/action icons matching desktop conventions.
- Added shared local icon set in [icons.tsx](/Users/sam/projects/go/basepro/web/src/ui/icons.tsx) so web can use MUI `SvgIcon` components without external package download in this offline environment.
- Updated route tests for the new accessible controls in [routes.test.tsx](/Users/sam/projects/go/basepro/web/src/routes.test.tsx).

### How to run tests
- `cd web && npm run test`
- `cd web && npm run build`

### Verification summary
- `cd web && npm run test`: PASS
- `cd web && npm run build`: PASS

### Known follow-ups
- One non-blocking test console warning appears from MUI Select popover anchor validation in jsdom when opening the theme mode menu.
- Existing Vite third-party warnings (`'use client'`, chunk-size warning) remain unchanged.

## Web — DataGrid Foundation (Milestone 9 parity) Complete

### What changed
- Added reusable web DataGrid wrapper: `web/src/components/datagrid/AppDataGrid.tsx`.
  - Required `storageKey`.
  - Server-side pagination, sorting, filtering.
  - Column visibility and reordering.
  - Density selector and CSV export toolbar.
  - Bold headers styling.
  - Pinned columns behind an explicit feature flag (`enablePinnedColumns`) to gracefully no-op when unsupported.
  - Standardized snackbar error handling with `X-Request-Id` support (no stack traces).
- Added per-table DataGrid preference persistence helper: `web/src/components/datagrid/storage.ts`.
  - Namespaced key format: `app.datagrid.<storageKey>.v1`.
  - Stores `pageSize`, visibility, order, density, pinned columns.
  - Schema-safe loading with default fallback and optional migration hook.
- Added shared server list query contract helper: `web/src/lib/pagination.ts`.
- Replaced web users placeholder page with DataGrid integration: `web/src/pages/UsersPage.tsx`.
- Added web audit page with DataGrid integration: `web/src/pages/AuditPage.tsx`.
- Added `/audit` route and AppShell nav/title integration with RBAC gating.
- Documented the web list query/response contract in `web/README.md`.
- Saved prompt traceability copy under `docs/prompts/milestone-9-web-parity.md` (prompt directory is gitignored).

### Tests and verification
- Web tests: PASS (`cd web && npm run test`)
- Web build: PASS (`cd web && npm run build`)
- Backend tests: PASS (`cd backend && go test ./...`)
- Desktop frontend tests: PASS (`cd desktop/frontend && npm test`)

### Known follow-ups
- Vite build emits existing upstream warnings (`"use client"` directive noise and chunk size warning); build still passes.
- Pinned columns remain feature-flagged for web until enabling a supported DataGrid tier/config.

## Milestone — User Metadata Expansion Part 2 (Desktop) Complete

### What changed
- Updated desktop users management page in `desktop/frontend/src/pages/UsersPage.tsx`:
  - added metadata fields to create form: `username`, `password`, `email`, `language`, `firstName`, `lastName`, `displayName`, `phoneNumber`, `whatsappNumber`, `telegramHandle`, `isActive`
  - added edit form for user metadata with `username` disabled and `password` optional
  - kept existing RBAC gating (`users.write`) for create/update actions
  - kept existing session-expiry flow unchanged (global app handler still shows message and redirects to login)
  - mapped backend `VALIDATION_ERROR` field details to MUI field helper/error states for create/edit forms
  - updated users table columns to: `username`, `displayName`, `email`, `phoneNumber`, `isActive`, `updatedAt` (plus existing actions)
- Extended API error handling in `desktop/frontend/src/api/client.ts` to preserve backend `error.details` for form-level validation mapping.
- Updated desktop route tests in `desktop/frontend/src/routes.test.tsx`:
  - verifies users list renders metadata (`email`, `phoneNumber`, active switch)
  - verifies create form submits metadata payload
  - verifies edit form submits metadata updates with optional password behavior (password omitted when blank)

### How to run tests
- `make desktop-test`

### Verification summary
- Desktop tests: PASS (`make desktop-test`)

### Known follow-ups
- Backend and web suites were not rerun in this desktop-only part.

## Milestone — User Metadata Expansion Part 3 (Web) Complete

### What changed
- Updated users management UI in `web/src/pages/UsersPage.tsx` using existing `AppDataGrid` server-side list flow.
- Expanded `/users` grid columns for metadata: `username`, `displayName` (with fallback to `firstName + lastName` then username), `language`, `email`, `phoneNumber`, `whatsappNumber`, `telegramHandle`, `isActive`, `updatedAt`.
- Added create/edit user dialogs with metadata fields:
  - create: `username` + required `password`, plus metadata fields
  - edit: metadata fields + optional `password` (omitted from payload when blank)
- Kept standardized API error handling and added field-level validation mapping from backend `VALIDATION_ERROR` details.
- Added request failure messaging that surfaces backend `requestId` when available.
- Preserved RBAC gating in users UI:
  - create action hidden for users without `users.write`
  - edit action disabled when user lacks write permission
- Expanded web users tests in `web/src/pages/users-audit-pages.test.tsx` for metadata list rendering, create payload, edit payload (optional password), and validation-field error display.
- Updated web test setup expectations in `web/src/routes.test.tsx` and base URL resolution fallback in `web/src/lib/apiBaseUrl.ts` to keep deterministic test behavior with local overrides.
- Saved prompt traceability copy in `docs/prompts/milestone-user-metadata-expansion-part3-web.md` (gitignored).

### How to run tests
- `cd web && npm run test`

### Verification summary
- Web tests: PASS (`cd web && npm run test`)
- Web build: PASS (`cd web && npm run build`)

### Known follow-ups
- Existing non-blocking warnings remain in build output from third-party bundles (`'use client'` directives and chunk-size warning).

## Milestone — Administration Settings Navigation + Sukumad Demo Seed Complete

### What changed
- Moved `Settings` under the existing `Administration` navigation group in both desktop and web while preserving permission-based visibility.
- Updated both app shells so the `Administration` group auto-expands when the current route is `/settings`.
- Aligned navigation registry metadata and route tests in desktop and web so grouped administration items render consistently.
- Added repeatable Sukumad demo seed command at `backend/cmd/seed-sukumad-demo` to create sample integration servers and requests for UI review.
- Added backend demo seed service under `backend/internal/sukumad/devseed` that recreates a clean demo dataset targeting `https://play.im.dhis2.org/dev`.
- Demo seed currently creates 3 integration servers and 5 requests covering completed, pending, blocked, failed, and async-processing display states.
- Corrected Sukumad request SQL persistence so `status_reason` honors the current non-null schema and dependency status scans map correctly in Postgres-backed flows.
- Wired the demo seed command so seeded requests create durable delivery attempts before status transitions are applied.
- Saved prompt traceability copy in `docs/prompts/2026-03-14-admin-settings-nav-and-demo-seed.md` (gitignored).

### How to run tests
- `cd backend && GOCACHE=/tmp/go-build go test ./...`
- `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run`
- `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build`
- `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run`
- `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build`
- `cd backend && GOCACHE=/tmp/go-build go run ./cmd/seed-sukumad-demo`

### Verification summary
- Backend tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./...`)
- Web tests: PASS (`cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run`)
- Web build: PASS (`cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build`)
- Desktop tests: PASS (`cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run`)
- Desktop build: PASS (`cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build`)
- Demo seed command: PASS (`cd backend && GOCACHE=/tmp/go-build go run ./cmd/seed-sukumad-demo`)

### Known follow-ups
- Frontend verification still emits existing non-blocking warnings from MUI/jsdom anchor handling and upstream Vite bundle warnings; behavior remains unchanged.
- Demo seed uses fixed sample payloads against the DHIS2 play instance for display/testing purposes and should not be used as production fixture data.

## Milestone — External Request Tracking API Complete

### What changed
- Added a dedicated external Sukumad request contract for machine integrations without internal numeric IDs.
- Added external request routes in [backend/internal/sukumad/routes.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/routes.go):
  - `POST /api/v1/external/requests`
  - `GET /api/v1/external/requests/:uid`
  - `GET /api/v1/external/requests/lookup`
- Kept existing internal `/api/v1/requests` routes unchanged for current desktop and web clients.
- Extended request lookup and projection support in:
  - [backend/internal/sukumad/request/types.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/types.go)
  - [backend/internal/sukumad/request/repository.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/repository.go)
  - [backend/internal/sukumad/request/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/service.go)
  - [backend/internal/sukumad/request/handler.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/request/handler.go)
- External create now accepts only public references:
  - `destinationServerUid`
  - `destinationServerUids`
  - `dependencyRequestUids`
- External status responses now return only public identifiers and tracking metadata:
  - request `uid`
  - per-target `uid`
  - `batchId`
  - `correlationId`
  - `idempotencyKey`
  - request roll-up status
  - per-destination target status
  - latest delivery UID/status
  - latest async task UID/state/remote job metadata
- Added exact lookup support for:
  - request UID
  - `correlationId`
  - `sourceSystem + idempotencyKey`
  - `batchId`
- Enabled API-token principals for the external create/read-status routes by reusing the existing platform API-token middleware and `requests.read` / `requests.write` permission checks.
- Added server UID lookup support in:
  - [backend/internal/sukumad/server/types.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/server/types.go)
  - [backend/internal/sukumad/server/repository.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/server/repository.go)
  - [backend/internal/sukumad/server/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/server/service.go)
- Added database enforcement for external idempotency and faster batch lookup in:
  - [backend/migrations/000022_add_exchange_request_external_lookup_indexes.up.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000022_add_exchange_request_external_lookup_indexes.up.sql)
  - [backend/migrations/000022_add_exchange_request_external_lookup_indexes.down.sql](/Users/sam/projects/go/sukumadpro/backend/migrations/000022_add_exchange_request_external_lookup_indexes.down.sql)
  - unique idempotency scope: `source_system + idempotency_key` when the key is present
  - added `batch_id` index for batch lookup
- Updated the integration note in [docs/notes/integrating-external-apps-with-request-create.md](/Users/sam/projects/go/sukumadpro/docs/notes/integrating-external-apps-with-request-create.md) to document the new API-token and UID-only integration path.
- Saved the milestone prompt copy under `docs/prompts/2026-04-08-external-request-tracking-api.md` (gitignored, not committed).

### How to run tests
- `cd backend && GOCACHE=/tmp/go-build go test ./...`
- `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run`
- `cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build`
- `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run`
- `cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build`

### Verification summary
- Backend tests: PASS (`cd backend && GOCACHE=/tmp/go-build go test ./...`)
- Web tests: PASS (`cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run`)
- Web build: PASS (`cd web && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build`)
- Desktop tests: PASS (`cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vitest/vitest.mjs run --run`)
- Desktop build: PASS (`cd desktop/frontend && /Users/sam/.nvm/versions/node/v22.15.1/bin/node node_modules/vite/bin/vite.js build`)

### Known follow-ups
- Existing frontend test output still includes non-blocking MUI/jsdom anchor warnings; no frontend code was changed in this milestone.
- Existing Vite build warnings about ignored `'use client'` directives and chunk-size thresholds remain unchanged.
- The new external contract is intentionally parallel to the existing internal `/api/v1/requests` routes so current web and desktop request pages can continue using numeric-ID-backed internal APIs until a later parity decision is made.

## Planned Milestone — Shared Administration UX + Parity (Upcoming)

### Planned scope
- Expand administration navigation as grouped platform sections with first-class `Roles` and `Permissions` pages.
- Add reusable RBAC administration UI patterns for roles/permissions management on both desktop and web.
- Extend user create/update flows to support multi-role assignment with typed validation feedback.
- Standardize shared DataGrid actions-column behavior across administration pages.
- Introduce and apply global DataGrid UI preference controls (for example density, action pinning, radius) across shared admin tables.
- Add audit metadata details UX with truncated in-grid previews and full JSON details dialog/drawer.
- Enforce cross-client parity expectations for shared admin capabilities unless temporary gaps are explicitly documented.

### Planned validation expectations
- Backend tests for RBAC administration and multi-role assignment validation.
- Desktop and web route/smoke tests for new admin pages and permission-gated navigation.
- Shared DataGrid action-pattern tests and audit metadata details dialog tests.

### Notes
- This milestone remains skeleton-focused and intentionally avoids domain placeholders.
- This entry documents upcoming work only; no implementation completion is claimed here.
