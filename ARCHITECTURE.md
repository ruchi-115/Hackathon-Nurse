# End-to-End Architecture

This project ingests synthetic PCC patient data, extracts wound-care billing fields, routes each patient to a biller-facing decision, and displays the result in a Next.js dashboard.

```mermaid
flowchart LR
    biller["Biller or reviewer"]
    operator["Pipeline operator"]

    subgraph frontend ["Next.js frontend"]
        dashboard["Dashboard UI: filters, routing table, patient drill-down"]
        nextApi["Next API routes: /api/stats, /api/eligibility, /api/patients/:id"]
    end

    subgraph backend ["Go backend"]
        cli["cmd/pipeline: ingest, process, serve, run"]
        pccClient["pccclient: retry, backoff, rate limit"]
        ingestor["ingest.Ingestor: facility scan and patient fan-out"]
        processor["ingest.Process: per-patient extraction and routing"]
        extractor["extract.Extract: assessments, note rules, optional LLM"]
        router["routing.Decide: auto_accept, flag_for_review, reject"]
        restApi["chi REST API: /health, /stats, /eligibility, /patients/{patientID}"]
    end

    subgraph datastore ["Local datastore"]
        sqlite["SQLite pcc.db: patient, diagnosis, coverage, note, assessment, eligibility"]
    end

    subgraph external ["External systems"]
        pcc["Mock PCC API: patients, diagnoses, coverage, notes, assessments"]
        llm["Anthropic API optional: narrative extraction fallback"]
    end

    operator -->|"Runs go pipeline command"| cli
    cli -->|"Creates client"| pccClient
    cli -->|"Starts ingest"| ingestor
    pccClient -.->|"Fetches with retry-after handling"| pcc
    ingestor -->|"Stores raw PCC records"| sqlite
    cli -->|"Starts process"| processor
    processor -->|"Reads raw records"| sqlite
    processor -->|"Extracts wound fields"| extractor
    extractor -.->|"Optional fallback"| llm
    processor -->|"Checks coverage and diagnoses"| router
    router -->|"Writes one row per patient"| sqlite
    cli -->|"Starts serve mode"| restApi
    restApi -->|"Queries eligibility and detail records"| sqlite
    biller -->|"Uses dashboard"| dashboard
    dashboard -->|"Fetches JSON"| nextApi
    nextApi -->|"Proxies to Go backend only"| restApi
```

## Runtime Flow

```mermaid
sequenceDiagram
    participant Operator
    participant Pipeline as Go cmd/pipeline
    participant PCC as Mock PCC API
    participant DB as SQLite pcc.db
    participant API as Go REST API
    participant Next as Next.js API routes
    participant UI as Dashboard
    participant Biller

    Operator->>Pipeline: go run ./cmd/pipeline run
    Pipeline->>API: Serve /health, /stats, /eligibility, /patients/{patientID}
    Pipeline->>PCC: Background GET facilities and patient records
    PCC-->>Pipeline: Patients, diagnoses, coverage, notes, assessments
    Pipeline->>DB: Upsert raw records
    Pipeline->>DB: Read per-patient record bundle
    Pipeline->>Pipeline: Extract wound fields and route decision
    Pipeline->>DB: Upsert eligibility row as each patient is ready
    Biller->>UI: Open dashboard
    UI->>Next: Request stats, table rows, patient detail
    Next->>API: Proxy backend requests
    API->>DB: Query eligibility and drill-down records
    DB-->>API: Result rows
    API-->>Next: JSON response
    Next-->>UI: JSON response
    UI-->>Biller: Routing summary and patient evidence
    UI->>Next: Poll while processed count is below 300
```

## Key Data Products

- Raw PCC tables: `patient`, `diagnosis`, `coverage`, `note`, `assessment`
- Derived output table: `eligibility`
- Dashboard-facing API:
  - `GET /stats`
  - `GET /eligibility`
  - `GET /patients/{patientID}`

## Operational Modes

- `go run ./cmd/pipeline ingest`: fetch raw PCC data, then process it
- `go run ./cmd/pipeline process`: re-run extraction and routing on existing SQLite data
- `go run ./cmd/pipeline serve`: serve the REST API from existing SQLite data
- `go run ./cmd/pipeline run`: ingest, process, then serve
