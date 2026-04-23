# RapidEx Org-Unit Jurisdiction Scope

Date: 2026-04-23

## Decisions

- `user_org_units` assignments are now an enforced jurisdiction boundary for RapidEx org-unit-tagged modules.
- The first enforced modules are:
  - `orgunits`
  - `reporters`
- `Admin` users remain unrestricted.
- Non-admin users with no assigned org units are treated as restricted to an empty scope.

## Enforcement Model

- A user assignment is treated as a root jurisdiction node.
- Jurisdiction includes the assigned org unit and all descendants.
- Descendant checks use `org_units.path` as the canonical hierarchy source.
- Scoped list endpoints filter by allowed org-unit path prefixes.
- Scoped write and action endpoints validate that the affected record and target org unit remain inside scope.

## Interface Changes

- `GET /api/v1/auth/me` now includes:
  - `assignedOrgUnitIds`
  - `isOrgUnitScopeRestricted`
- `GET /api/v1/user-org-units/:userId` now returns:
  - `orgUnitIds`
  - `items` with assignment display details for admin UIs

## UI Notes

- Web and desktop Users pages now expose org-unit assignment management in-place.
- Facilities and Reporters pages show an explicit empty-state when a restricted user has no assigned org units.
- Facility selection/search continues using the existing endpoints and now benefits from backend-enforced scope automatically.
