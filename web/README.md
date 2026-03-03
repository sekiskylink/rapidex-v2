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

## Scripts
- Development server: `npm run dev`
- Production build: `npm run build`
- Tests: `npm run test`
- Preview build: `npm run preview`
