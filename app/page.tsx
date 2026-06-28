"use client";

import { useEffect, useMemo, useState } from "react";
import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  Download,
  Filter,
  RefreshCw,
  Search,
  ShieldCheck,
  XCircle
} from "lucide-react";
import type { EligibilityResult, ResultsPayload, RoutingDecision } from "@/src/lib/types";

type SyncState = "idle" | "loading" | "syncing" | "error";

const emptyPayload: ResultsPayload = {
  generatedAt: null,
  summary: {
    totalPatients: 0,
    autoAccept: 0,
    flagForReview: 0,
    reject: 0,
    medicareB: 0,
    withWoundEvidence: 0
  },
  results: [],
  errors: []
};

export default function Home() {
  const [payload, setPayload] = useState<ResultsPayload>(emptyPayload);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [decisionFilter, setDecisionFilter] = useState<RoutingDecision | "all">("all");
  const [facilityFilter, setFacilityFilter] = useState<string>("all");
  const [query, setQuery] = useState("");
  const [syncState, setSyncState] = useState<SyncState>("loading");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void loadResults();
  }, []);

  const filteredResults = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();

    return payload.results.filter((result) => {
      const matchesDecision = decisionFilter === "all" || result.routingDecision === decisionFilter;
      const matchesFacility = facilityFilter === "all" || String(result.facilityId) === facilityFilter;
      const matchesQuery =
        !normalizedQuery ||
        result.patientName.toLowerCase().includes(normalizedQuery) ||
        result.patientId.toLowerCase().includes(normalizedQuery) ||
        (result.woundType ?? "").toLowerCase().includes(normalizedQuery);

      return matchesDecision && matchesFacility && matchesQuery;
    });
  }, [decisionFilter, facilityFilter, payload.results, query]);

  const selected = useMemo(() => {
    return (
      filteredResults.find((result) => result.patientId === selectedId) ??
      filteredResults[0] ??
      null
    );
  }, [filteredResults, selectedId]);

  async function loadResults() {
    setSyncState("loading");
    setError(null);

    try {
      const response = await fetch("/api/results", { cache: "no-store" });
      const data = (await response.json()) as ResultsPayload;
      setPayload(data);
      setSelectedId(data.results[0]?.patientId ?? null);
      setSyncState("idle");
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "Unable to load results");
      setSyncState("error");
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

      const data = (await response.json()) as { results: ResultsPayload };
      setPayload(data.results);
      setSelectedId(data.results.results[0]?.patientId ?? null);
      setSyncState("idle");
    } catch (syncError) {
      setError(syncError instanceof Error ? syncError.message : "Unable to complete sync");
      setSyncState("error");
    }
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
            <span>{payload.generatedAt ? `Synced ${formatDateTime(payload.generatedAt)}` : "No sync yet"}</span>
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
        <Metric label="Total patients" value={payload.summary.totalPatients} icon={<Activity size={18} />} />
        <Metric label="Auto accept" value={payload.summary.autoAccept} icon={<CheckCircle2 size={18} />} tone="good" />
        <Metric label="Review" value={payload.summary.flagForReview} icon={<AlertTriangle size={18} />} tone="warn" />
        <Metric label="Rejected" value={payload.summary.reject} icon={<XCircle size={18} />} tone="bad" />
        <Metric label="Medicare B" value={payload.summary.medicareB} icon={<ShieldCheck size={18} />} />
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
                onChange={(event) => setDecisionFilter(event.target.value as RoutingDecision | "all")}
              >
                <option value="all">All decisions</option>
                <option value="auto_accept">Auto accept</option>
                <option value="flag_for_review">Review</option>
                <option value="reject">Reject</option>
              </select>
            </label>

            <label className="selectBox">
              <Filter size={16} aria-hidden="true" />
              <select value={facilityFilter} onChange={(event) => setFacilityFilter(event.target.value)}>
                <option value="all">All facilities</option>
                <option value="101">Facility A</option>
                <option value="102">Facility B</option>
                <option value="103">Facility C</option>
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
                {filteredResults.map((result) => (
                  <tr
                    key={result.patientId}
                    className={selected?.patientId === result.patientId ? "selectedRow" : ""}
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

            {!isBusy && filteredResults.length === 0 ? (
              <div className="emptyState">
                <Search size={20} aria-hidden="true" />
                <span>No patients match the current filters.</span>
              </div>
            ) : null}

            {isBusy ? (
              <div className="emptyState">
                <RefreshCw size={20} className="spin" aria-hidden="true" />
                <span>{syncState === "syncing" ? "Fetching PCC records and extracting wounds." : "Loading cached results."}</span>
              </div>
            ) : null}
          </div>
        </div>

        <PatientDetail result={selected} />
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

function PatientDetail({ result }: { result: EligibilityResult | null }) {
  if (!result) {
    return (
      <aside className="detail">
        <div className="detailEmpty">
          <Download size={22} aria-hidden="true" />
          <span>Run sync to populate the billing worklist.</span>
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

function formatWound(result: EligibilityResult) {
  const pieces = [titleCase(result.woundType), result.stage ? `stage ${result.stage}` : null].filter(Boolean);
  return pieces.join(", ") || "None found";
}

function formatMeasurements(result: EligibilityResult) {
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
