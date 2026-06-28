"use client";

import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  ClipboardList,
  FileText,
  Filter,
  HeartPulse,
  RefreshCw,
  Search,
  ShieldCheck,
  XCircle
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import {
  formatMeasurement,
  getPatient,
  getStats,
  listEligibility,
  patientName
} from "@/src/lib/storage";
import type {
  DashboardFilters,
  Decision,
  EligibilityRow,
  PatientDetail,
  Stats
} from "@/src/lib/types";

const decisionCopy: Record<
  Decision,
  { label: string; className: string; Icon: typeof CheckCircle2 }
> = {
  auto_accept: {
    label: "Auto accept",
    className: "badge badge-success",
    Icon: CheckCircle2
  },
  flag_for_review: {
    label: "Review",
    className: "badge badge-warning",
    Icon: AlertTriangle
  },
  reject: {
    label: "Reject",
    className: "badge badge-danger",
    Icon: XCircle
  }
};

export function Dashboard() {
  const [filters, setFilters] = useState<DashboardFilters>({
    facility: "all",
    decision: "all",
    mcbOnly: false
  });
  const [search, setSearch] = useState("");
  const [rows, setRows] = useState<EligibilityRow[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<PatientDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    setLoading(true);
    setError(null);
    try {
      const [eligibility, dashboardStats] = await Promise.all([
        listEligibility(filters),
        getStats()
      ]);
      setRows(eligibility.results);
      setStats(dashboardStats);
      setSelectedId((current) => current ?? eligibility.results[0]?.patient_id ?? null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to load dashboard");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void refresh();
  }, [filters.facility, filters.decision, filters.mcbOnly]);

  useEffect(() => {
    if (!selectedId) {
      setDetail(null);
      return;
    }
    setDetailLoading(true);
    getPatient(selectedId)
      .then(setDetail)
      .catch(() => setDetail(null))
      .finally(() => setDetailLoading(false));
  }, [selectedId]);

  const visibleRows = useMemo(() => {
    const needle = search.trim().toLowerCase();
    if (!needle) {
      return rows;
    }
    return rows.filter((row) => {
      return [
        row.patient_id,
        patientName(row),
        row.reason,
        row.wound.wound_type,
        row.wound.location,
        row.primary_payer_code
      ]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(needle));
    });
  }, [rows, search]);

  const counts = {
    total: stats?.total ?? rows.length,
    auto: stats?.by_decision.auto_accept ?? 0,
    review: stats?.by_decision.flag_for_review ?? 0,
    reject: stats?.by_decision.reject ?? 0,
    mcb: stats?.active_mcb ?? rows.filter((row) => row.has_active_mcb).length
  };

  return (
    <main className="shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Wound care billing</p>
          <h1>Eligibility Workbench</h1>
        </div>
        <div className="topbar-actions">
          <span className="source-pill">
            <Activity size={16} />
            {stats?.source === "backend" ? "Go API live" : "Backend required"}
          </span>
          <button className="icon-button" onClick={() => void refresh()} title="Refresh">
            <RefreshCw size={18} />
          </button>
        </div>
      </header>

      <section className="metric-grid" aria-label="Pipeline summary">
        <Metric icon={<ClipboardList />} label="Patients" value={counts.total} />
        <Metric icon={<CheckCircle2 />} label="Auto accept" value={counts.auto} tone="success" />
        <Metric icon={<AlertTriangle />} label="Needs review" value={counts.review} tone="warning" />
        <Metric icon={<XCircle />} label="Rejected" value={counts.reject} tone="danger" />
        <Metric icon={<ShieldCheck />} label="Medicare B" value={counts.mcb} tone="info" />
      </section>

      <section className="workbench">
        <div className="table-panel">
          <div className="toolbar">
            <label className="search-box">
              <Search size={17} />
              <input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder="Search patient, reason, wound"
              />
            </label>

            <div className="filter-group" aria-label="Dashboard filters">
              <Filter size={17} />
              <select
                value={filters.facility}
                onChange={(event) =>
                  setFilters((current) => ({
                    ...current,
                    facility: event.target.value
                  }))
                }
              >
                <option value="all">All facilities</option>
                <option value="101">Facility 101</option>
                <option value="102">Facility 102</option>
                <option value="103">Facility 103</option>
              </select>
              <select
                value={filters.decision}
                onChange={(event) =>
                  setFilters((current) => ({
                    ...current,
                    decision: event.target.value
                  }))
                }
              >
                <option value="all">All routes</option>
                <option value="auto_accept">Auto accept</option>
                <option value="flag_for_review">Review</option>
                <option value="reject">Reject</option>
              </select>
              <label className="toggle">
                <input
                  type="checkbox"
                  checked={Boolean(filters.mcbOnly)}
                  onChange={(event) =>
                    setFilters((current) => ({
                      ...current,
                      mcbOnly: event.target.checked
                    }))
                  }
                />
                MCB only
              </label>
            </div>
          </div>

          {error ? <div className="status-panel danger-text">{error}</div> : null}
          {loading ? <div className="status-panel">Loading eligibility rows...</div> : null}

          {!loading && !error ? (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Patient</th>
                    <th>Route</th>
                    <th>Facility</th>
                    <th>Payer</th>
                    <th>Wound</th>
                    <th>Measurements</th>
                    <th>Reason</th>
                  </tr>
                </thead>
                <tbody>
                  {visibleRows.map((row) => (
                    <EligibilityTableRow
                      key={row.patient_id}
                      row={row}
                      selected={row.patient_id === selectedId}
                      onSelect={() => setSelectedId(row.patient_id)}
                    />
                  ))}
                </tbody>
              </table>
              {visibleRows.length === 0 ? (
                <div className="status-panel">No patients match the current filters.</div>
              ) : null}
            </div>
          ) : null}
        </div>

        <PatientPanel
          row={rows.find((row) => row.patient_id === selectedId) ?? null}
          detail={detail}
          loading={detailLoading}
        />
      </section>
    </main>
  );
}

function Metric({
  icon,
  label,
  value,
  tone = "neutral"
}: {
  icon: React.ReactNode;
  label: string;
  value: number;
  tone?: "neutral" | "success" | "warning" | "danger" | "info";
}) {
  return (
    <div className={`metric metric-${tone}`}>
      <div className="metric-icon">{icon}</div>
      <div>
        <span>{label}</span>
        <strong>{value.toLocaleString()}</strong>
      </div>
    </div>
  );
}

function EligibilityTableRow({
  row,
  selected,
  onSelect
}: {
  row: EligibilityRow;
  selected: boolean;
  onSelect: () => void;
}) {
  const decision = decisionCopy[row.decision];
  const DecisionIcon = decision.Icon;
  return (
    <tr className={selected ? "selected" : ""} onClick={onSelect}>
      <td>
        <button className="patient-cell" onClick={onSelect}>
          <strong>{patientName(row)}</strong>
          <span>{row.patient_id}</span>
        </button>
      </td>
      <td>
        <span className={decision.className}>
          <DecisionIcon size={14} />
          {decision.label}
        </span>
      </td>
      <td>{row.facility_id}</td>
      <td>
        <span className={row.has_active_mcb ? "payer active" : "payer"}>
          {row.primary_payer_code ?? "Unknown"}
        </span>
      </td>
      <td>
        <div className="stacked">
          <strong>{row.wound.wound_type || "Missing"}</strong>
          <span>{row.wound.location || "No location"}</span>
        </div>
      </td>
      <td>{formatMeasurement(row)}</td>
      <td className="reason">{row.reason}</td>
    </tr>
  );
}

function PatientPanel({
  row,
  detail,
  loading
}: {
  row: EligibilityRow | null;
  detail: PatientDetail | null;
  loading: boolean;
}) {
  if (!row) {
    return (
      <aside className="detail-panel">
        <div className="status-panel">Select a patient</div>
      </aside>
    );
  }

  const panelRow = detail?.eligibility ?? row;
  const decision = decisionCopy[panelRow.decision];
  const DecisionIcon = decision.Icon;

  return (
    <aside className="detail-panel">
      <div className="detail-head">
        <div>
          <p className="eyebrow">{panelRow.patient_id}</p>
          <h2>{patientName(panelRow)}</h2>
        </div>
        <span className={decision.className}>
          <DecisionIcon size={14} />
          {decision.label}
        </span>
      </div>

      {loading ? <div className="status-panel">Loading patient record...</div> : null}

      <dl className="fact-grid">
        <div>
          <dt>Coverage</dt>
          <dd>{panelRow.has_active_mcb ? "Active Medicare B" : "No active MCB"}</dd>
        </div>
        <div>
          <dt>Confidence</dt>
          <dd>{Math.round((panelRow.wound.confidence ?? 0) * 100)}%</dd>
        </div>
        <div>
          <dt>Drainage</dt>
          <dd>{panelRow.wound.drainage_amount || "Missing"}</dd>
        </div>
        <div>
          <dt>Source</dt>
          <dd>{panelRow.wound.extraction_source || "Unknown"}</dd>
        </div>
      </dl>

      <section className="detail-section">
        <h3>
          <HeartPulse size={17} />
          Wound summary
        </h3>
        <div className="wound-summary">
          <div>
            <span>Type</span>
            <strong>{panelRow.wound.wound_type || "Missing"}</strong>
          </div>
          <div>
            <span>Stage</span>
            <strong>{panelRow.wound.stage || "N/A"}</strong>
          </div>
          <div>
            <span>Location</span>
            <strong>{panelRow.wound.location || "Missing"}</strong>
          </div>
          <div>
            <span>Measurements</span>
            <strong>{formatMeasurement(panelRow)}</strong>
          </div>
        </div>
        <p className="panel-reason">{panelRow.reason}</p>
      </section>

      <RecordSection title="Coverage" icon={<ShieldCheck size={17} />}>
        {detail?.coverage?.length ? (
          detail.coverage.map((item) => (
            <div className="record-line" key={`${item.id}-${item.payer_name}`}>
              <strong>{item.payer_name ?? item.payer_type ?? "Coverage"}</strong>
              <span>{[item.payer_code, item.effective_from].filter(Boolean).join(" / ")}</span>
            </div>
          ))
        ) : (
          <span className="muted">No coverage detail loaded.</span>
        )}
      </RecordSection>

      <RecordSection title="Diagnoses" icon={<ClipboardList size={17} />}>
        {detail?.diagnoses?.length ? (
          detail.diagnoses.slice(0, 5).map((item) => (
            <div className="record-line" key={item.id}>
              <strong>{item.icd10_code ?? "Diagnosis"}</strong>
              <span>{item.icd10_description ?? "No description"}</span>
            </div>
          ))
        ) : (
          <span className="muted">No diagnosis detail loaded.</span>
        )}
      </RecordSection>

      <RecordSection title="Evidence" icon={<FileText size={17} />}>
        {detail?.observations?.length
          ? detail.observations.slice(0, 4).map((item, index) => (
              <div className="record-line" key={`${item.sourceLabel}-${index}`}>
                <strong>{item.sourceLabel ?? item.source ?? "Observation"}</strong>
                <span>{item.evidence ?? "No evidence text"}</span>
              </div>
            ))
          : detail?.notes?.length
            ? detail.notes.slice(0, 3).map((item) => (
                <div className="record-line" key={item.id}>
                  <strong>{item.note_label ?? item.note_type ?? "Note"}</strong>
                  <span>{item.note_text ?? "No note text"}</span>
                </div>
              ))
            : <span className="muted">No note evidence loaded.</span>}
      </RecordSection>
    </aside>
  );
}

function RecordSection({
  title,
  icon,
  children
}: {
  title: string;
  icon: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className="detail-section">
      <h3>
        {icon}
        {title}
      </h3>
      <div className="record-list">{children}</div>
    </section>
  );
}
