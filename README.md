# SukumadPro

SukumadPro is a BasePro-derived application with:

- a Gin backend under `backend/`
- a Wails desktop client under `desktop/`
- an optional React web client under `web/`

## API Documentation

OpenAPI-based API documentation is now part of the backend.

- Spec source: [api/openapi.yaml](/Users/sam/projects/go/sukumadpro/api/openapi.yaml)
- Generator config: [api/oapi-codegen.yaml](/Users/sam/projects/go/sukumadpro/api/oapi-codegen.yaml)
- Generated Go code: [openapi.gen.go](/Users/sam/projects/go/sukumadpro/backend/internal/openapi/generated/openapi.gen.go)
- Notes: [api-documentation.md](/Users/sam/projects/go/sukumadpro/docs/notes/api-documentation.md)

## Regeneration

From the repository root:

```bash
make generate-openapi
make check-openapi
```

## Docs URLs

When the backend is running:

- OpenAPI document: `/openapi.yaml`
- Scalar UI: `/docs`

Example local URLs:

- `http://127.0.0.1:8080/openapi.yaml`
- `http://127.0.0.1:8080/docs`
