## Rapidex v2 Milestone 7–10 — Reporters UI, Mapping Engine Port, Sukumad Integration & End‑to‑End

### What changed

- **Reporters UI (Milestone 7):** Implemented full management pages for reporters in both the web and desktop clients.  The pages fetch reporters from the backend, display them in a data grid, and allow administrators to create new reporters by entering a contact UUID, phone number, display name and selecting an organisation unit.  Deletion is supported via an action column.  The UI reuses Material UI’s data grid and dialog components and integrates organisation unit lists for selection.
- **Mapping engine port (Milestone 8):** Added a new `rapidex` domain module with types representing mapping configurations (`MappingConfig`, `DataValueMapping`), aggregate payloads (`AggregatePayload`, `DataValue`) and RapidPro webhooks.  Implemented a `Manager` that loads and validates YAML mapping definitions from a `configs/mappings` directory and provides access by flow UUID.  Ported the core mapping logic from Rapidex v1, including a helper `extractField` and a `MapToAggregate` function that transforms RapidPro webhook events into DHIS2 aggregate payloads.  Added a validator to ensure mapping files include mandatory fields and do not contain duplicate mappings.  Included a sample mapping file under `configs/mappings/sample.yaml`.
- **Sukumad integration (Milestone 9):** Created an `IntegrationService` skeleton that coordinates webhook processing.  It retrieves the appropriate mapping configuration, performs the mapping to an aggregate payload and prepares to hand off to the Sukumad request engine.  A placeholder is provided for reporter lookup and exchange request creation.  Added a Gin HTTP handler that exposes a `POST /rapidex/webhook` endpoint for RapidPro to deliver flow results.
- **Documentation and examples:** Added a milestone status document summarising these changes and outlining follow‑up tasks.

### Added/updated tests

No automated tests were added for these milestones.  Future work should include unit tests for the mapping functions, manager reload behaviour, reporter UI interactions and the integration service.

### Verification summary

- The web and desktop clients compile and display the new Reporters page with list and creation functionality.
- Mapping configurations can be loaded from YAML files; the validator catches missing fields and duplicates.
- Calling the `/rapidex/webhook` endpoint with a valid webhook JSON triggers the integration service, which maps the event and logs the would‑be aggregate payload.

### Known follow‑ups

- Complete the integration between the `IntegrationService` and Sukumad’s request engine so that aggregate payloads result in `exchange_requests` being created and delivered via the existing request lifecycle.
- Implement reporter lookup by contact UUID and phone number in the integration service and override the mapped org unit accordingly.  Ensure new reporters are created when unknown contacts submit data.
- Enforce user‑org‑unit scoping in reporters UI and API so that users only see and manage reporters within their assigned scope.
- Add update/edit functionality to the reporters UI, including toggling active status and changing organisation assignments.
- Add comprehensive tests covering mapping validation, transformation logic, reporter UI interactions and end‑to‑end webhook processing.
- Incorporate asynchronous delivery modes (direct, asynq) as configuration options for Rapidex submissions.
- Once all follow‑ups are completed, run integration tests to validate the full pipeline from RapidPro webhook to DHIS2 submission and update `docs/status.md` accordingly.