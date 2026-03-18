# API Documentation

This project serves API documentation directly from the backend using:

- `oapi-codegen` for Go code generation from OpenAPI
- Scalar for the browser documentation UI

## Source of truth

The OpenAPI source document lives at [openapi.yaml](/Users/sam/projects/go/sukumadpro/api/openapi.yaml).

Supporting generator configuration lives at [oapi-codegen.yaml](/Users/sam/projects/go/sukumadpro/api/oapi-codegen.yaml).

Generated Go output is committed at [openapi.gen.go](/Users/sam/projects/go/sukumadpro/backend/internal/openapi/generated/openapi.gen.go).

## How it works

1. `api/openapi.yaml` is edited by hand.
2. `make generate-openapi` runs `oapi-codegen` against that spec.
3. Generated code is written to `backend/internal/openapi/generated/openapi.gen.go`.
4. The backend serves:
   - `/openapi.yaml` by loading the generated embedded spec and rendering YAML
   - `/docs` by returning a small Scalar HTML page that points to `/openapi.yaml`

This keeps documentation aligned with the checked-in spec while avoiding fragile runtime filesystem lookups.

## Make targets

From the repository root:

```bash
make generate-openapi
make check-openapi
```

`generate-openapi`
- Regenerates the Go artifacts from the OpenAPI spec.

`check-openapi`
- Regenerates the Go artifacts and fails if [openapi.gen.go](/Users/sam/projects/go/sukumadpro/backend/internal/openapi/generated/openapi.gen.go) is out of date relative to the checked-in version.

## Safe update workflow

When changing the HTTP API contract:

1. Update [openapi.yaml](/Users/sam/projects/go/sukumadpro/api/openapi.yaml).
2. Run `make generate-openapi`.
3. Review the generated diff in [openapi.gen.go](/Users/sam/projects/go/sukumadpro/backend/internal/openapi/generated/openapi.gen.go).
4. Run backend tests, at minimum:
   - `cd backend && GOCACHE=/tmp/go-build go test ./cmd/api`
5. Open the docs locally:
   - `http://127.0.0.1:8080/openapi.yaml`
   - `http://127.0.0.1:8080/docs`
6. If the change affects request or response shapes, update any affected route tests and client code.

## Notes

- The current spec is router-aligned and intentionally uses some generic object schemas for endpoints whose JSON envelopes are still evolving.
- The docs UI depends on the app serving `/openapi.yaml`; Scalar is configured to read from that local endpoint rather than from an external hosted spec.
