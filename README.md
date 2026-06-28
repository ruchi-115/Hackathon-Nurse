# ABI Wound Billing Triage

A Next.js end-to-end application for the ABI Frameworks hackathon challenge. The app syncs synthetic PointClickCare records, extracts wound documentation from structured assessments and clinical notes, evaluates Medicare Part B eligibility, and presents a biller-facing triage worklist.

## What This Builds

The challenge asks for a pipeline that helps a post-acute care company identify which patients qualify for wound-care billing under Medicare Part B.

This implementation turns that into a practical workflow:

1. Pull patients from all mock PCC facilities.
2. Fetch diagnoses, coverage, progress notes, and wound assessments for each patient.
3. Handle API rate limiting with retry logic and `Retry-After` support.
4. Persist a local raw sync snapshot in `data/sync-snapshot.json`.
5. Extract wound type, stage, location, dimensions, and drainage.
6. Evaluate each patient for active Medicare Part B coverage and complete wound documentation.
7. Materialize a derived eligibility cache in `data/eligibility-results.json`.
8. Route each patient to `auto_accept`, `flag_for_review`, or `reject`.
9. Display paginated results in a dashboard designed for a non-technical biller.

## App Experience

The first screen is the billing worklist.

It includes:

- a summary of total patients, auto-accepts, review items, rejections, and Medicare B patients
- a `Run sync` action that fetches the mock PCC API and rebuilds local caches
- server-side search, filters, and pagination
- rows-per-page control for large worklists
- cookie-backed UI preferences for facility, decision, and page size
- a patient detail panel that loads only after a row is selected, with:
  - routing decision
  - plain-English reason
  - coverage evidence
  - extracted wound fields
  - evidence snippet
  - source observations

The dashboard is meant to answer a biller's core question quickly:

> Who can I act on, and why?

## Tech Stack

- **Next.js** for the full-stack app
- **React** for the worklist UI
- **TypeScript** for typed pipeline models
- **Local JSON storage** for raw and derived caches
- **Rules and regex extraction** for explainable wound parsing
- **Server route handlers** for API sync and result generation

No external database is required for the hackathon version. The local files keep the app easy to run and inspect while avoiding a large frontend hydration payload.

## Setup

```bash
npm install
npm run dev
```

Then open:

```text
http://localhost:3000
```

Run the first sync from the dashboard. The sync can take a little time because the mock API intentionally returns HTTP `429` responses around 30% of the time.

## Environment

The app defaults to the hackathon API:

```text
https://hackathon.prod.pulsefoundry.ai
```

To override it:

```bash
PCC_BASE_URL=https://your-api.example.com npm run dev
```

## Project Structure

```text
app/
  api/
    health/route.ts     health and local snapshot status
    results/route.ts    paginated eligibility rows
    results/[patientId] lazy patient detail and evidence
    summary/route.ts    cached summary metrics
    sync/route.ts       fetches PCC data and rebuilds caches
  globals.css           dashboard styling
  layout.tsx            app shell metadata
  page.tsx              biller-facing worklist UI

src/lib/
  eligibility.ts        patient-level routing rules
  extraction.ts         structured and free-text wound extraction
  pccClient.ts          API client with retry and backoff
  storage.ts            local cache persistence and server-side querying
  types.ts              shared TypeScript models

data/
  sync-snapshot.json        generated raw PCC snapshot
  eligibility-results.json  generated derived eligibility cache

API.md                 hackathon API reference
README.md             this handoff document
```

## Pipeline Design

### 1. Facility Sync

The app queries all facilities:

```text
101 - Facility A
102 - Facility B
103 - Facility C
```

The pipeline does not hardcode patient counts. It asks the API for patients per facility and processes whatever is returned.

### 2. Identifier Handling

The mock PCC API uses two patient identifiers:

- string `patient_id`, such as `FA-001`, for diagnoses and coverage
- integer `id`, such as `1`, for notes and assessments

The sync route keeps both identifiers from the patient endpoint and uses the correct one for each downstream endpoint.

### 3. Rate Limit Handling

Every PCC request goes through `PccClient`.

The client:

- retries HTTP `429`
- reads the `Retry-After` header
- adds small jitter to avoid synchronized retries
- retries transient `5xx` responses
- fails individual endpoint calls without crashing the full sync

Endpoint-level failures are stored in the snapshot and returned to the UI through the results payload.

### 4. Local Caches

After sync, the app writes:

```text
data/sync-snapshot.json
data/eligibility-results.json
```

The raw snapshot contains:

- generation timestamp
- facilities synced
- patient bundles
- diagnoses
- coverage
- notes
- assessments
- endpoint errors

The derived eligibility cache contains:

- one current eligibility result per patient
- summary counts
- routing reasons
- extracted wound fields
- detail evidence and observations

This keeps the pipeline inspectable and repeatable while letting dashboard APIs return small payloads.

## Extraction Strategy

The extraction layer favors reliable structured data before free text.

### Source Priority

1. **Structured assessments**
   - parse `raw_json`
   - highest confidence
   - best source for dimensions and drainage

2. **Structured progress notes**
   - parse labels such as `Location:`, `Wound Type:`, `Length:`, `Drainage:`
   - high confidence when all required fields are present

3. **Prose notes**
   - parse shorthand such as `4.2x3.1x1.5cm`
   - parse drainage phrases such as `moderate drainage`
   - lower confidence than structured notes

4. **Diagnoses**
   - used as wound evidence only
   - not enough for `auto_accept` because diagnoses do not include measurements or drainage

### Extracted Fields

For each wound observation, the app attempts to extract:

- wound type
- pressure ulcer stage
- location
- length in cm
- width in cm
- depth in cm
- drainage amount
- source
- source date
- evidence text
- confidence score

## Eligibility Logic

The output is one patient-level row.

### `auto_accept`

Assigned when:

- active Medicare Part B coverage exists
- wound evidence exists
- wound type is present
- length, width, and depth are present
- drainage amount is present
- extraction confidence is high enough

This means the patient is ready for billing review.

### `flag_for_review`

Assigned when:

- active Medicare Part B coverage exists
- wound evidence exists
- but required wound fields are incomplete, ambiguous, or lower confidence

This is the queue for a clinician or biller to inspect.

### `reject`

Assigned when:

- active Medicare Part B coverage is missing, or
- no active wound evidence is found

This prevents non-billable patients from entering the billing work queue.

## Why This Is Scalable

The app now assumes the dataset can be much larger than the hackathon sample, such as 10,000 patients.

The important scalability choices are:

- **No full frontend hydration:** the browser does not receive every patient result.
- **Materialized eligibility cache:** extraction and routing run during sync, not every page load.
- **Paginated table API:** `/api/results` returns only the requested page.
- **Lazy detail fetch:** evidence snippets and observations load from `/api/results/:patientId` only when a patient is selected.
- **Server-side filtering/search:** facility, decision, and search filters run before pagination.
- **Cookie-backed UI preferences:** facility, decision, and page-size preferences persist without storing clinical data in the browser.
- **Bounded page size:** the API caps page size so the UI cannot accidentally request thousands of rows.

The hackathon version still uses local JSON storage, but the code is split so each layer can be swapped cleanly.

Recommended production upgrades:

- Replace local cache files with Postgres.
- Store raw API responses in append-only tables.
- Add incremental sync using the API's `since` parameter.
- Add durable job processing with a queue.
- Stream sync progress from a job table instead of holding one long web request open.
- Use Redis for hot dashboard caches like summary metrics and common filter pages.
- Track sync runs, endpoint latency, retries, and failed patients.
- Add authenticated biller accounts and audit history.
- Add an LLM fallback only for low-confidence narrative notes.
- Save human review outcomes to improve future routing.

The current app already separates the important concerns:

- API client
- storage
- extraction
- eligibility rules
- presentation

That makes it straightforward to evolve without rewriting the whole system.

## Frontend Hydration Strategy

The dashboard deliberately hydrates a small amount of data:

```text
Initial load:
  GET /api/summary
  GET /api/results?page=1&pageSize=50
  GET /api/results/:patientId for the selected row

Filter/search change:
  GET /api/results with server-side filters

Row selection:
  GET /api/results/:patientId
```

For a 10,000-patient dataset, this means the frontend normally holds 25-200 row objects plus one detailed patient record, not 10,000 detailed records.

The cookie stores only UI preferences:

```text
decisionFilter
facilityFilter
pageSize
```

It does not store PHI, wound evidence, or patient records.

## Demo Walkthrough

For a 10-minute presentation:

1. Start on the dashboard and run a sync.
2. Explain that the app pulls from all PCC facilities and handles rate limits automatically.
3. Show the summary metrics.
4. Filter to `Auto accept`.
5. Open one patient and read the reason aloud like a biller would.
6. Filter to `Review`.
7. Show a patient with missing measurements or drainage.
8. Filter to `Reject`.
9. Show that non-Medicare-B patients are excluded from billing action.
10. Close with the architecture: sync, extract, route, present.

## Design Choices

### Why rules first?

Rules are explainable and deterministic. Structured assessments and labeled notes should not require an LLM to parse reliably.

### Why include confidence?

Confidence helps separate clean extraction from ambiguous narrative notes. It also gives a future human-review workflow a useful sorting signal.

### Why preserve evidence snippets?

A biller needs trust. The app does not only say `auto_accept`; it shows the source evidence behind the decision.

### Why local JSON instead of a database?

For the hackathon, local JSON is easier to inspect and avoids setup overhead. The code is structured so the storage layer can be replaced with SQLite or Postgres later.

## API Endpoints

### `POST /api/sync`

Fetches PCC data, writes `data/sync-snapshot.json`, builds `data/eligibility-results.json`, and returns sync metadata plus summary counts.

The endpoint does not return every patient result.

### `GET /api/summary`

Returns cached dashboard metrics:

```text
generatedAt
summary
errorCount
```

### `GET /api/results`

Returns a paginated worklist page from the derived eligibility cache.

Supported query parameters:

```text
page
pageSize
facility
decision
q
```

The response includes lightweight rows without heavy evidence/observation arrays.

### `GET /api/results/:patientId`

Returns the full patient-level eligibility result, including evidence and observations, for one selected patient.

### `GET /api/health`

Returns whether the app has a raw snapshot, a derived results cache, and where both files are stored.

## Known Tradeoffs

- Multi-wound notes are handled by ranking extracted observations, but a production system should preserve explicit wound identity across time.
- Narrative extraction is intentionally conservative. Ambiguous notes should route to review.
- The app currently runs sync from a web request. A production system should move sync to a background job with progress polling.
- The local snapshot is a convenient hackathon store, not a multi-user database.

## Challenge Reference

The mock API reference is in [API.md](./API.md).
