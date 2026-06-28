import type {
  DashboardFilters,
  EligibilityResponse,
  PatientDetail,
  Stats
} from "@/src/lib/types";

function buildQuery(filters?: DashboardFilters) {
  const params = new URLSearchParams();
  if (filters?.facility && filters.facility !== "all") {
    params.set("facility_id", filters.facility);
  }
  if (filters?.decision && filters.decision !== "all") {
    params.set("decision", filters.decision);
  }
  if (filters?.mcbOnly) {
    params.set("mcb", "true");
  }
  return params.toString();
}

async function requestJSON<T>(url: string): Promise<T> {
  const res = await fetch(url, { cache: "no-store" });
  if (!res.ok) {
    throw new Error(`Request failed: ${res.status}`);
  }
  return res.json() as Promise<T>;
}

export async function listEligibility(
  filters?: DashboardFilters
): Promise<EligibilityResponse> {
  const query = buildQuery(filters);
  return requestJSON<EligibilityResponse>(
    `/api/eligibility${query ? `?${query}` : ""}`
  );
}

export async function getStats(): Promise<Stats> {
  return requestJSON<Stats>("/api/stats");
}

export async function getPatient(patientId: string): Promise<PatientDetail> {
  return requestJSON<PatientDetail>(
    `/api/patients/${encodeURIComponent(patientId)}`
  );
}

export function patientName(row: {
  first_name?: string | null;
  last_name?: string | null;
  patient_id: string;
}) {
  const name = [row.first_name, row.last_name].filter(Boolean).join(" ").trim();
  return name || row.patient_id;
}

export function formatMeasurement(row: {
  wound: {
    length_cm?: number | null;
    width_cm?: number | null;
    depth_cm?: number | null;
  };
}) {
  const { length_cm, width_cm, depth_cm } = row.wound;
  if (length_cm == null || width_cm == null || depth_cm == null) {
    return "Missing";
  }
  return `${length_cm} x ${width_cm} x ${depth_cm} cm`;
}
