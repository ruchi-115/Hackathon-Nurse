# Vercel Deployment

Use Vercel for the Next.js dashboard. Keep the Go pipeline/API on Render or another long-running backend host.

## Why Frontend-Only on Vercel

The Go backend is a long-running API plus background ingestion pipeline that writes to SQLite. Vercel can run Go serverless functions, but this project needs:

- a continuously running REST server
- background ingestion that can take minutes on first sync
- durable SQLite storage

That fits Render/Fly/Railway-style web services better than Vercel Functions. The frontend is already built to call a remote backend through `GO_API_BASE_URL`.

## Prerequisite

Deploy the Go backend first. For Render, use `render.yaml` or follow [DEPLOYMENT.md](./DEPLOYMENT.md).

After the backend is deployed, copy its public URL, for example:

```text
https://abi-pipeline-api.onrender.com
```

## Deploy Through Vercel Dashboard

1. Push this repo to GitHub.
2. Go to Vercel.
3. Choose **Add New > Project**.
4. Import the GitHub repo.
5. Set the framework preset to **Next.js**.
6. Use these settings:

```text
Install Command: npm install
Build Command: npm run build
Output Directory: .next
```

7. Add this environment variable for Production, Preview, and Development:

```text
GO_API_BASE_URL=https://your-backend-service.onrender.com
```

8. Deploy.

## Deploy With CLI

If you prefer CLI:

```bash
npx vercel login
npx vercel
```

When prompted, link this project. Then set the backend URL:

```bash
npx vercel env add GO_API_BASE_URL production
npx vercel env add GO_API_BASE_URL preview
npx vercel env add GO_API_BASE_URL development
```

Deploy production:

```bash
npx vercel --prod
```

## Smoke Test

After deployment, open:

```text
https://your-vercel-app.vercel.app/api/stats
```

You should see JSON from the Go backend with `"source":"backend"`.

Then open:

```text
https://your-vercel-app.vercel.app
```

## Fully Free Demo Path

For a zero-cost demo:

1. Deploy the Next.js frontend on Vercel Hobby.
2. Deploy the Go backend as a Render Free web service.
3. Use `DB_PATH=/tmp/pcc.db` on Render.
4. Do not attach a persistent disk.

This works for demos because the dashboard progressively loads rows while the backend reingests. The tradeoff is that the SQLite database is not durable: Render Free web services can spin down and have an ephemeral filesystem, so the backend may need to reingest after restarts, redeploys, or idle spin-downs.

Use [render.free.yaml](./render.free.yaml) as a reference for the free backend settings. If Render only detects `render.yaml`, create the free backend manually in the Render dashboard with the same values.
