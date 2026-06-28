# Render Deployment

This repo is configured for Render with a two-service Blueprint:

- `abi-pipeline-api`: Go web service for ingesting, processing, and serving the REST API.
- `abi-pipeline-web`: Next.js web service for the dashboard.

## Files Added

- `render.yaml`: Render Blueprint for both services.
- `scripts/render-backend-start.sh`: backend start script that runs initial ingest only when the SQLite DB is missing.
- `package.json`: `start` now binds Next.js to Render's `$PORT`.

## Deploy With Blueprint

1. Push this repo to GitHub.
2. In Render, choose **New > Blueprint**.
3. Connect this repository.
4. Render will create:
   - `abi-pipeline-api`
   - `abi-pipeline-web`
5. Wait for the backend first deploy to complete. On first boot it will:
   - create `/var/data/pcc.db`
   - start the REST API immediately
   - fetch PCC data in the background
   - write eligibility rows as each patient is ready
   - run a final extraction/routing reconciliation
6. Open the frontend service URL.

## Important Render Notes

- The Go backend uses `PORT` automatically through `internal/config/config.go`.
- The Next app uses `next start -p ${PORT:-3000}`.
- `GO_API_BASE_URL` is populated from the backend service's `RENDER_EXTERNAL_URL`.
- The backend stores SQLite at `/var/data/pcc.db` on a persistent disk.
- Persistent disks require a paid Render web service, so the backend is set to `plan: starter`.
- The frontend is set to `plan: free`.

## Manual Deployment Values

If you do not use the Blueprint, create two Render web services manually.

### Backend

- Runtime: Go
- Build command:

```bash
go build -o pipeline ./cmd/pipeline
```

- Start command:

```bash
bash scripts/render-backend-start.sh
```

- Health check path:

```text
/health
```

- Environment variables:

```text
DB_PATH=/var/data/pcc.db
CORS_ORIGIN=*
RATE_PER_SECOND=8
RATE_BURST=2
MAX_RETRIES=8
```

- Persistent disk:

```text
mountPath=/var/data
sizeGB=1
```

### Frontend

- Runtime: Node
- Build command:

```bash
npm install && npm run build
```

- Start command:

```bash
npm run start
```

- Environment variables:

```text
NODE_VERSION=22
NODE_ENV=production
GO_API_BASE_URL=https://your-backend-service.onrender.com
```
