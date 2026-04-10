# Documentation Browser

This note describes the authenticated markdown documentation browser for SukumadPro.

## Purpose

The app now exposes curated project documentation inside the existing BasePro shell instead of sending operators to repository files. Documentation content remains markdown on disk, while the backend controls which files are visible through a configurable root path and allowlist.

## Backend source and API

The backend implementation lives under [backend/internal/sukumad/documentation](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/documentation).

The service is configured from the Viper-backed runtime config:

- `documentation.root_path`
- `documentation.files[].slug`
- `documentation.files[].title`
- `documentation.files[].path`
- `documentation.files[].order`

The default configuration points at `../docs/notes` from the backend working directory and allowlists the current notes in [docs/notes](/Users/sam/projects/go/sukumadpro/docs/notes).

The backend serves two authenticated endpoints:

- `GET /api/v1/documentation`
- `GET /api/v1/documentation/:slug`

The service validates the configured root, rejects absolute or traversal paths, requires markdown files, and only serves files present in the allowlist. Missing or unknown documents return a typed not-found response instead of leaking filesystem details.

## Module and navigation

Documentation is registered as a BasePro module with module id `documentation` and module flag `modules.documentation.enabled`.

The browser is intentionally authenticated but does not introduce a new RBAC permission. Any authenticated user with the module enabled can read the curated docs. This keeps operator help content broadly available while still avoiding unauthenticated filesystem exposure.

Web and desktop navigation add the same Documentation entry under the existing Sukumad group, reusing the current AppShell and route access helpers.

## Client rendering

The web page lives at [web/src/pages/DocumentationPage.tsx](/Users/sam/projects/go/sukumadpro/web/src/pages/DocumentationPage.tsx).

The desktop page lives at [desktop/frontend/src/pages/DocumentationPage.tsx](/Users/sam/projects/go/sukumadpro/desktop/frontend/src/pages/DocumentationPage.tsx).

Both clients:

- fetch the document list and selected document body through the backend API
- render markdown with `react-markdown` and `remark-gfm`
- style markdown elements with MUI components and `sx`
- avoid raw HTML rendering
- keep route, module registry, navigation registry, and settings label parity

## Updating the documentation set

To add a document:

1. Add the markdown file under the configured documentation root.
2. Add an entry to `documentation.files` in backend config with a stable slug, title, relative path, and order.
3. If it should be part of the default app config, update the defaults in [backend/internal/config/config.go](/Users/sam/projects/go/sukumadpro/backend/internal/config/config.go) and [backend/config/config.yaml](/Users/sam/projects/go/sukumadpro/backend/config/config.yaml).
4. Run backend route/service tests and web/desktop route tests.

This design keeps the file root configurable while avoiding broad directory browsing or serving arbitrary markdown files.
