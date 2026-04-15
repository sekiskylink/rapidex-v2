# API Token Settings UI

The authenticated Settings page now exposes a shared API Access section in both web and desktop. The section lets a user persist local auth mode and API token settings, create a new backend API token when they have `api_tokens.write`, and copy the plaintext token immediately after creation.

Regular backend requests now prefer the saved API token when API-token mode is selected. Auth/session endpoints still use the JWT flow, so login, refresh, and bootstrap behavior remain unchanged. The default seeded admin user does not receive a token automatically; they create one from Settings after signing in.
