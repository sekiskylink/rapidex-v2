# Global Request Finder

## Summary

RapidEx now exposes a global request finder from the authenticated application shell in both web and desktop clients.

The finder is intended for fast operational lookup without navigating to the Requests page first.

## UI contract

- Entry point:
  - AppBar button labeled by icon only
  - keyboard shortcut `Cmd/Ctrl+K`
- Existing app launcher remains available with `Cmd/Ctrl+Shift+K`
- The finder opens as a right-side drawer and renders compact request matches directly in the drawer.
- Selecting a match opens the existing request detail dialog.

## Backend request-list contract

The existing `GET /api/v1/requests` endpoint now accepts additional optional query params:

- `msisdn`
- `orgUnitUid`
- `from`
- `to`

Behavior:

- `msisdn` matches `exchange_requests.extras.msisdn` exactly.
- `orgUnitUid` matches descendant-aware org-unit UIDs using this precedence:
  - `extras.mappedOrgUnit`
  - `extras.orgUnit`
  - payload JSON field `orgUnit`
- `from` and `to` filter `exchange_requests.created_at` inclusively.
- `from` and `to` accept RFC3339 timestamps and `YYYY-MM-DD`.

## Notes

- The finder defaults to the last 7 days in the client.
- Build/test runs still emit existing non-blocking frontend warnings around MUI/jsdom `anchorEl`, Vite `'use client'`, and large chunk sizes.
