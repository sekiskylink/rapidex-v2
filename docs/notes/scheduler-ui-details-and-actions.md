# Scheduler UI Details and Actions

Date: 2026-04-29

## Decisions

- Keep `/scheduler/:jobId` as the edit route.
- Add schedule viewing as a read-only dialog launched from the scheduler listing.
- Use the existing scheduler `GET /jobs/:id` payload for the detail dialog instead of reusing the form.
- Add hard delete for scheduled jobs through the existing `scheduler.write` permission.
- Use distinct action icons for view, edit, runs, enable/disable, run now, and delete so scheduler actions are easier to scan.

## Rationale

- The scheduler form route was already established and is still the clearest place for editing.
- A detail dialog matches the exchange-request pattern and avoids duplicating read-only values inside disabled form controls.
- Deleting schedules belongs in the same scheduler management surface and should remove related run history alongside the schedule.
