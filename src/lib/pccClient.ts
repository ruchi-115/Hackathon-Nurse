import type {
  Assessment,
  Coverage,
  Diagnosis,
  FacilityId,
  Patient,
  ProgressNote,
  SyncError
} from "./types";

const DEFAULT_BASE_URL = "https://hackathon.prod.pulsefoundry.ai";
const MAX_ATTEMPTS = 7;

export class PccClient {
  private readonly baseUrl: string;

  constructor(baseUrl = process.env.PCC_BASE_URL ?? DEFAULT_BASE_URL) {
    this.baseUrl = baseUrl.replace(/\/$/, "");
  }

  get url() {
    return this.baseUrl;
  }

  async getPatients(facilityId: FacilityId, since?: string) {
    const params = new URLSearchParams({ facility_id: String(facilityId) });
    if (since) params.set("since", since);
    return this.getJson<Patient[]>(`/pcc/patients?${params.toString()}`);
  }

  async getDiagnoses(patientId: string) {
    return this.getJson<Diagnosis[]>(
      `/pcc/diagnoses?${new URLSearchParams({ patient_id: patientId }).toString()}`
    );
  }

  async getCoverage(patientId: string) {
    return this.getJson<Coverage[]>(
      `/pcc/coverage?${new URLSearchParams({ patient_id: patientId }).toString()}`
    );
  }

  async getNotes(internalPatientId: number, since?: string) {
    const params = new URLSearchParams({ patient_id: String(internalPatientId) });
    if (since) params.set("since", since);
    return this.getJson<ProgressNote[]>(`/pcc/notes?${params.toString()}`);
  }

  async getAssessments(internalPatientId: number, since?: string) {
    const params = new URLSearchParams({ patient_id: String(internalPatientId) });
    if (since) params.set("since", since);
    return this.getJson<Assessment[]>(`/pcc/assessments?${params.toString()}`);
  }

  private async getJson<T>(path: string): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    let lastError: Error | null = null;

    for (let attempt = 1; attempt <= MAX_ATTEMPTS; attempt += 1) {
      try {
        const response = await fetch(url, {
          cache: "no-store",
          headers: { accept: "application/json" }
        });

        if (response.status === 429) {
          const retryAfter = Number(response.headers.get("retry-after") ?? "1");
          await sleep(Math.max(retryAfter, 1) * 1000 + jitter());
          continue;
        }

        if (response.status >= 500 && attempt < MAX_ATTEMPTS) {
          await sleep(500 * attempt + jitter());
          continue;
        }

        if (!response.ok) {
          const body = await response.text();
          throw new Error(`${response.status} ${response.statusText}: ${body}`);
        }

        return (await response.json()) as T;
      } catch (error) {
        lastError = error instanceof Error ? error : new Error(String(error));
        if (attempt < MAX_ATTEMPTS) {
          await sleep(400 * attempt + jitter());
        }
      }
    }

    throw lastError ?? new Error(`Failed to fetch ${url}`);
  }
}

export async function settleEndpoint<T>(
  endpoint: string,
  getData: () => Promise<T>,
  errorContext: Omit<SyncError, "endpoint" | "message">
): Promise<{ data: T | null; error: SyncError | null }> {
  try {
    return { data: await getData(), error: null };
  } catch (error) {
    return {
      data: null,
      error: {
        ...errorContext,
        endpoint,
        message: error instanceof Error ? error.message : String(error)
      }
    };
  }
}

function jitter() {
  return Math.floor(Math.random() * 250);
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
