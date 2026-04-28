# API Token Bound User Scope

## Summary

API-token authentication remains globally available, but route support is now intentional instead of accidental.

- Interactive admin/session endpoints stay JWT-user only.
- Operational Sukumad APIs can accept API tokens when permission checks are sufficient.
- User-scoped Sukumad APIs can accept API tokens only when the token is bound to an active user.

## Bound user model

API tokens now support an optional `boundUserId`.

- The token stores the bound user reference in `api_tokens.bound_user_id`.
- Token creation rejects missing or inactive bound users.
- On successful token authentication, the principal remains `api_token` but carries an effective user context derived from the bound user.

This preserves the distinction between a machine principal and a human session while still allowing user-scoped services to reuse existing org-unit and reporter access rules.

## Route policy

### JWT-only

These routes remain JWT-user only because they represent interactive session/admin behavior rather than machine-safe integration access:

- `/api/v1/auth/me`
- `/api/v1/admin/api-tokens`
- `/api/v1/users`
- `/api/v1/admin/roles`
- `/api/v1/admin/permissions`
- `/api/v1/settings`
- `/api/v1/dashboard`

### API token enabled

These route families now accept API tokens when the token has the required permission:

- `/api/v1/servers`
- `/api/v1/external/servers`
- `/api/v1/requests`
- `/api/v1/external/requests`
- `/api/v1/jobs`
- `/api/v1/deliveries`
- `/api/v1/scheduler`
- `/api/v1/observability`
- `/api/v1/documentation`
- `/api/v1/rapidex/webhook`
- `/api/v1/audit`

### API token enabled with bound user

These route families require a bound user because their handlers/services depend on user-scoped access:

- `/api/v1/orgunits`
- `/api/v1/reporters`
- `/api/v1/reporter-groups`
- `/api/v1/user-org-units`

## Implications

- A token without `boundUserId` is still valid for operational endpoints that do not require user scope.
- A token targeting user-scoped endpoints must be created with `boundUserId`.
- Audit and actor attribution on token-enabled operational writes now use the effective bound user when available.
