# BasePro Web Frontend

Optional browser frontend for BasePro using React + TypeScript, Material UI, TanStack Router, and TanStack Query.

## Prerequisites
- Node.js 18+
- npm 9+

## Setup
```bash
cd web
npm install
cp .env.example .env
```

## Environment Variables
- `VITE_API_BASE_URL` (required): Backend API base URL, for example `http://127.0.0.1:8080/api/v1`
- `VITE_APP_NAME` (optional): App display name
- `VITE_ENABLE_DEVTOOLS` (optional): Enables query devtools integration when implemented

## Backend API Docs
- Raw OpenAPI document: `http://127.0.0.1:8080/openapi.yaml`
- Scalar UI: `http://127.0.0.1:8080/docs`
- Spec source: `../api/openapi.yaml`
- Regenerate backend OpenAPI artifacts from the repo root with `make generate-openapi`

## Scripts
- Development server: `npm run dev`
- Production build: `npm run build`
- Tests: `npm run test`
- Preview build: `npm run preview`

## DataGrid Server Contract
List pages use a shared server query contract (1-based pagination):
- Query params:
  - `page` (1-based integer)
  - `pageSize` (integer)
  - `sort` (`<field>:<asc|desc>`)
  - `filter` (`<field>:<value>`, first active filter item)
- Response JSON:
  - `{ items, totalCount, page, pageSize }`

Current web list pages using this contract:
- `/users` -> `GET /users`
- `/audit` -> `GET /audit`
