import type {
  DashboardFilters,
  EligibilityResponse,
  PatientDetail,
  Stats
} from "@/src/lib/types";

const publicBackendBase = process.env.NEXT_PUBLIC_GO_API_BASE_URL?.replace(
  /\/$/,
  ""
);

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

function endpoint(backendPath: string, proxyPath: string) {
  return publicBackendBase ? `${publicBackendBase}${backendPath}` : proxyPath;
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
  const path = `/eligibility${query ? `?${query}` : ""}`;
  const data = await requestJSON<EligibilityResponse>(
    endpoint(path, `/api${path}`)
  );
  return { ...data, source: "backend" };
}

export async function getStats(): Promise<Stats> {
  const data = await requestJSON<Stats>(endpoint("/stats", "/api/stats"));
  return { ...data, source: "backend" };
}

export async function getPatient(patientId: string): Promise<PatientDetail> {
  const path = `/patients/${encodeURIComponent(patientId)}`;
  return requestJSON<PatientDetail>(
    endpoint(path, `/api${path}`)
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
