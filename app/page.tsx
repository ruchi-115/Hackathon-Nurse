"use client";

import { useEffect, useMemo, useState } from "react";
import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  Download,
  Filter,
  RefreshCw,
  Search,
  ShieldCheck,
  XCircle
} from "lucide-react";
import type {
  EligibilityListRow,
  EligibilityResult,
  PaginatedResultsPayload,
  ResultsSummary,
  RoutingDecision
} from "@/src/lib/types";

type SyncState = "idle" | "loading" | "syncing" | "error";
type DetailState = "idle" | "loading" | "error";

type SummaryPayload = {
  generatedAt: string | null;
  summary: ResultsSummary;
  errorCount: number;
};

const emptySummary: SummaryPayload = {
  generatedAt: null,
  errorCount: 0,
  summary: {
    totalPatients: 0,
    autoAccept: 0,
    flagForReview: 0,
    reject: 0,
    medicareB: 0,
    withWoundEvidence: 0
  }
};

const emptyPage: PaginatedResultsPayload = {
  generatedAt: null,
  rows: [],
  page: 1,
  pageSize: 50,
  totalRows: 0,
  totalPages: 0,
  errorCount: 0
};

export default function Home() {
  const [summary, setSummary] = useState<SummaryPayload>(emptySummary);
  const [pageData, setPageData] = useState<PaginatedResultsPayload>(emptyPage);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [selectedDetail, setSelectedDetail] = useState<EligibilityResult | null>(null);
  const [decisionFilter, setDecisionFilter] = useState<RoutingDecision | "all">("all");
  const [facilityFilter, setFacilityFilter] = useState<string>("all");
  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [syncState, setSyncState] = useState<SyncState>("loading");
  const [detailState, setDetailState] = useState<DetailState>("idle");
  const [error, setError] = useState<string | null>(null);
  const [refreshToken, setRefreshToken] = useState(0);

  useEffect(() => {
    const preferences = readPreferencesCookie();
    if (preferences) {
      setDecisionFilter(preferences.decisionFilter);
      setFacilityFilter(preferences.facilityFilter);
      setPageSize(preferences.pageSize);
    }
    void loadSummary();
  }, []);

  useEffect(() => {
    const timeout = window.setTimeout(() => {
      setDebouncedQuery(query);
      setPage(1);
    }, 300);

    return () => window.clearTimeout(timeout);
  }, [query]);

  useEffect(() => {
    writePreferencesCookie({ decisionFilter, facilityFilter, pageSize });
  }, [decisionFilter, facilityFilter, pageSize]);

  useEffect(() => {
    void loadRows();
  }, [decisionFilter, facilityFilter, debouncedQuery, page, pageSize, refreshToken]);

  useEffect(() => {
    if (!selectedId) {
      setSelectedDetail(null);
      return;
    }

    const controller = new AbortController();
    void loadPatientDetail(selectedId, controller.signal);
    return () => controller.abort();
  }, [selectedId, refreshToken]);

  const rangeLabel = useMemo(() => {
    if (pageData.totalRows === 0) return "0 patients";
    const start = (pageData.page - 1) * pageData.pageSize + 1;
    const end = Math.min(pageData.totalRows, pageData.page * pageData.pageSize);
    return `${start}-${end} of ${pageData.totalRows}`;
  }, [pageData]);

  async function loadSummary() {
    setError(null);

    try {
      const response = await fetch("/api/summary", { cache: "no-store" });
      const data = (await response.json()) as SummaryPayload;
      setSummary(data);
      setSyncState("idle");
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "Unable to load summary");
      setSyncState("error");
    }
  }

  async function loadRows() {
    setSyncState((current) => (current === "syncing" ? current : "loading"));
    setError(null);

    const params = new URLSearchParams({
      page: String(page),
      pageSize: String(pageSize),
      decision: decisionFilter,
      facility: facilityFilter
    });

    if (debouncedQuery.trim()) {
      params.set("q", debouncedQuery.trim());
    }

    try {
      const response = await fetch(`/api/results?${params.toString()}`, { cache: "no-store" });
      const data = (await response.json()) as PaginatedResultsPayload;
      setPageData(data);
      setSelectedId((current) => {
        if (data.rows.length === 0) return null;
        if (current && data.rows.some((row) => row.patientId === current)) return current;
        return data.rows[0].patientId;
      });
      setSyncState("idle");
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "Unable to load patients");
      setSyncState("error");
    }
  }

  async function loadPatientDetail(patientId: string, signal: AbortSignal) {
    setDetailState("loading");

    try {
      const response = await fetch(`/api/results/${encodeURIComponent(patientId)}`, {
        cache: "no-store",
        signal
      });

      if (!response.ok) {
        throw new Error(`Detail request failed with ${response.status}`);
      }

      const data = (await response.json()) as EligibilityResult;
      setSelectedDetail(data);
      setDetailState("idle");
    } catch (detailError) {
      if ((detailError as Error).name === "AbortError") return;
      setSelectedDetail(null);
      setDetailState("error");
    }
  }

  async function runSync() {
    setSyncState("syncing");
    setError(null);

    try {
      const response = await fetch("/api/sync", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({})
      });

      if (!response.ok) {
        throw new Error(`Sync failed with ${response.status}`);
      }

      await response.json();
      await loadSummary();
      setPage(1);
      setRefreshToken((value) => value + 1);
      setSyncState("idle");
    } catch (syncError) {
      setError(syncError instanceof Error ? syncError.message : "Unable to complete sync");
      setSyncState("error");
    }
  }

  function updateDecisionFilter(value: RoutingDecision | "all") {
    setDecisionFilter(value);
    setPage(1);
  }

  function updateFacilityFilter(value: string) {
    setFacilityFilter(value);
    setPage(1);
  }

  function updatePageSize(value: number) {
    setPageSize(value);
    setPage(1);
  }

  const isBusy = syncState === "loading" || syncState === "syncing";

  return (
    <main className="shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Medicare Part B</p>
          <h1>Wound Billing Triage</h1>
        </div>

        <div className="topbarActions">
          <div className="snapshot">
            <Activity size={16} aria-hidden="true" />
            <span>{summary.generatedAt ? `Synced ${formatDateTime(summary.generatedAt)}` : "No sync yet"}</span>
          </div>
          <button className="primaryButton" onClick={runSync} disabled={isBusy} title="Run PCC sync">
            <RefreshCw size={17} className={isBusy ? "spin" : ""} aria-hidden="true" />
            <span>{syncState === "syncing" ? "Syncing" : "Run sync"}</span>
          </button>
        </div>
      </header>

      {error ? (
        <div className="errorBanner" role="alert">
          <AlertTriangle size={18} aria-hidden="true" />
          <span>{error}</span>
        </div>
      ) : null}

      <section className="metrics" aria-label="Eligibility summary">
        <Metric label="Total patients" value={summary.summary.totalPatients} icon={<Activity size={18} />} />
        <Metric label="Auto accept" value={summary.summary.autoAccept} icon={<CheckCircle2 size={18} />} tone="good" />
        <Metric label="Review" value={summary.summary.flagForReview} icon={<AlertTriangle size={18} />} tone="warn" />
        <Metric label="Rejected" value={summary.summary.reject} icon={<XCircle size={18} />} tone="bad" />
        <Metric label="Medicare B" value={summary.summary.medicareB} icon={<ShieldCheck size={18} />} />
      </section>

      <section className="workspace">
        <div className="worklist">
          <div className="controls">
            <label className="searchBox">
              <Search size={17} aria-hidden="true" />
              <input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Search patient, ID, wound"
              />
            </label>

            <label className="selectBox">
              <Filter size={16} aria-hidden="true" />
              <select
                value={decisionFilter}
                onChange={(event) => updateDecisionFilter(event.target.value as RoutingDecision | "all")}
              >
                <option value="all">All decisions</option>
                <option value="auto_accept">Auto accept</option>
                <option value="flag_for_review">Review</option>
                <option value="reject">Reject</option>
              </select>
            </label>

            <label className="selectBox">
              <Filter size={16} aria-hidden="true" />
              <select value={facilityFilter} onChange={(event) => updateFacilityFilter(event.target.value)}>
                <option value="all">All facilities</option>
                <option value="101">Facility A</option>
                <option value="102">Facility B</option>
                <option value="103">Facility C</option>
              </select>
            </label>

            <label className="selectBox pageSizeControl">
              <span>Rows</span>
              <select value={pageSize} onChange={(event) => updatePageSize(Number(event.target.value))}>
                <option value={25}>25</option>
                <option value={50}>50</option>
                <option value={100}>100</option>
                <option value={200}>200</option>
              </select>
            </label>
          </div>

          <div className="tableWrap">
            <table>
              <thead>
                <tr>
                  <th>Patient</th>
                  <th>Decision</th>
                  <th>Wound</th>
                  <th>Measurements</th>
                  <th>Coverage</th>
                </tr>
              </thead>
              <tbody>
                {pageData.rows.map((result) => (
                  <tr
                    key={result.patientId}
                    className={selectedId === result.patientId ? "selectedRow" : ""}
                    onClick={() => setSelectedId(result.patientId)}
                  >
                    <td>
                      <strong>{result.patientName}</strong>
                      <span>{result.patientId} | Facility {result.facilityId}</span>
                    </td>
                    <td>
                      <DecisionBadge decision={result.routingDecision} />
                    </td>
                    <td>{formatWound(result)}</td>
                    <td>{formatMeasurements(result)}</td>
                    <td>{result.hasActiveMedicareB ? "Medicare B" : result.primaryPayerCode ?? "None"}</td>
                  </tr>
                ))}
              </tbody>
            </table>

            {!isBusy && pageData.rows.length === 0 ? (
              <div className="emptyState">
                <Search size={20} aria-hidden="true" />
                <span>No patients match the current filters.</span>
              </div>
            ) : null}

            {isBusy ? (
              <div className="emptyState">
                <RefreshCw size={20} className="spin" aria-hidden="true" />
                <span>{syncState === "syncing" ? "Fetching PCC records and rebuilding caches." : "Loading a page of patients."}</span>
              </div>
            ) : null}
          </div>

          <div className="pagination">
            <span>{rangeLabel}</span>
            <div className="paginationButtons">
              <button onClick={() => setPage((value) => Math.max(1, value - 1))} disabled={pageData.page <= 1 || isBusy}>
                <ChevronLeft size={17} aria-hidden="true" />
                <span>Prev</span>
              </button>
              <strong>
                Page {pageData.totalPages === 0 ? 0 : pageData.page} of {pageData.totalPages}
              </strong>
              <button
                onClick={() => setPage((value) => Math.min(pageData.totalPages, value + 1))}
                disabled={pageData.totalPages === 0 || pageData.page >= pageData.totalPages || isBusy}
              >
                <span>Next</span>
                <ChevronRight size={17} aria-hidden="true" />
              </button>
            </div>
          </div>
        </div>

        <PatientDetail result={selectedDetail} state={detailState} />
      </section>
    </main>
  );
}

function Metric({
  label,
  value,
  icon,
  tone = "neutral"
}: {
  label: string;
  value: number;
  icon: React.ReactNode;
  tone?: "neutral" | "good" | "warn" | "bad";
}) {
  return (
    <div className={`metric ${tone}`}>
      <div className="metricIcon">{icon}</div>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function PatientDetail({ result, state }: { result: EligibilityResult | null; state: DetailState }) {
  if (state === "loading") {
    return (
      <aside className="detail">
        <div className="detailEmpty">
          <RefreshCw size={22} className="spin" aria-hidden="true" />
          <span>Loading patient evidence.</span>
        </div>
      </aside>
    );
  }

  if (!result) {
    return (
      <aside className="detail">
        <div className="detailEmpty">
          <Download size={22} aria-hidden="true" />
          <span>Run sync or select a patient to see evidence.</span>
        </div>
      </aside>
    );
  }

  return (
    <aside className="detail">
      <div className="detailHeader">
        <div>
          <p>{result.patientId}</p>
          <h2>{result.patientName}</h2>
        </div>
        <DecisionBadge decision={result.routingDecision} />
      </div>

      <div className="reasonBox">
        <span>Reason</span>
        <p>{result.reason}</p>
      </div>

      <dl className="facts">
        <div>
          <dt>Coverage</dt>
          <dd>{result.activeCoverageLabel ?? "No active Medicare Part B"}</dd>
        </div>
        <div>
          <dt>Wound</dt>
          <dd>{formatWound(result)}</dd>
        </div>
        <div>
          <dt>Location</dt>
          <dd>{result.location ?? "Missing"}</dd>
        </div>
        <div>
          <dt>Measurements</dt>
          <dd>{formatMeasurements(result)}</dd>
        </div>
        <div>
          <dt>Drainage</dt>
          <dd>{result.drainageAmount ?? "Missing"}</dd>
        </div>
        <div>
          <dt>Source</dt>
          <dd>{result.source ?? "None"}</dd>
        </div>
      </dl>

      <div className="evidence">
        <div className="sectionTitle">
          <span>Evidence</span>
          <strong>{Math.round(result.confidence * 100)}%</strong>
        </div>
        <p>{result.evidence}</p>
      </div>

      <div className="observations">
        <div className="sectionTitle">
          <span>Observations</span>
          <strong>{result.observations.length}</strong>
        </div>
        {result.observations.slice(0, 4).map((observation, index) => (
          <div className="observation" key={`${observation.sourceLabel}-${index}`}>
            <span>{observation.sourceLabel}</span>
            <p>{observation.evidence}</p>
          </div>
        ))}
      </div>
    </aside>
  );
}

function DecisionBadge({ decision }: { decision: RoutingDecision }) {
  const labels: Record<RoutingDecision, string> = {
    auto_accept: "Auto accept",
    flag_for_review: "Review",
    reject: "Reject"
  };

  return <span className={`badge ${decision}`}>{labels[decision]}</span>;
}

function formatWound(result: EligibilityListRow | EligibilityResult) {
  const pieces = [titleCase(result.woundType), result.stage ? `stage ${result.stage}` : null].filter(Boolean);
  return pieces.join(", ") || "None found";
}

function formatMeasurements(result: EligibilityListRow | EligibilityResult) {
  if (result.lengthCm === null || result.widthCm === null || result.depthCm === null) {
    return "Missing";
  }

  return `${result.lengthCm} x ${result.widthCm} x ${result.depthCm} cm`;
}

function formatDateTime(value: string) {
  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit"
  }).format(new Date(value));
}

function titleCase(value: string | null) {
  if (!value) return null;
  return value.replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function readPreferencesCookie() {
  if (typeof document === "undefined") return null;

  const match = document.cookie
    .split("; ")
    .find((cookie) => cookie.startsWith("abi_triage_preferences="));

  if (!match) return null;

  try {
    const parsed = JSON.parse(decodeURIComponent(match.split("=")[1])) as {
      decisionFilter?: RoutingDecision | "all";
      facilityFilter?: string;
      pageSize?: number;
    };

    return {
      decisionFilter: parsed.decisionFilter ?? "all",
      facilityFilter: parsed.facilityFilter ?? "all",
      pageSize: parsed.pageSize && parsed.pageSize >= 10 ? parsed.pageSize : 50
    };
  } catch {
    return null;
  }
}

function writePreferencesCookie(preferences: {
  decisionFilter: RoutingDecision | "all";
  facilityFilter: string;
  pageSize: number;
}) {
  if (typeof document === "undefined") return;

  document.cookie = `abi_triage_preferences=${encodeURIComponent(
    JSON.stringify(preferences)
  )}; Max-Age=2592000; Path=/; SameSite=Lax`;
}
