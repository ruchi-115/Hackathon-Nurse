import { promises as fs } from "fs";
import path from "path";
import type {
  EligibilityListRow,
  EligibilityResult,
  PaginatedResultsPayload,
  ResultsPayload,
  RoutingDecision,
  SyncSnapshot
} from "./types";

const DATA_DIR = path.join(process.cwd(), "data");
const SNAPSHOT_PATH = path.join(DATA_DIR, "sync-snapshot.json");
const RESULTS_PATH = path.join(DATA_DIR, "eligibility-results.json");

export async function saveSnapshot(snapshot: SyncSnapshot) {
  await fs.mkdir(DATA_DIR, { recursive: true });
  await fs.writeFile(SNAPSHOT_PATH, JSON.stringify(snapshot, null, 2), "utf8");
}

export async function loadSnapshot(): Promise<SyncSnapshot | null> {
  try {
    const raw = await fs.readFile(SNAPSHOT_PATH, "utf8");
    return JSON.parse(raw) as SyncSnapshot;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ENOENT") {
      return null;
    }
    throw error;
  }
}

export async function saveResultsPayload(payload: ResultsPayload) {
  await fs.mkdir(DATA_DIR, { recursive: true });
  await fs.writeFile(RESULTS_PATH, JSON.stringify(payload, null, 2), "utf8");
}

export async function loadResultsPayload(): Promise<ResultsPayload | null> {
  try {
    const raw = await fs.readFile(RESULTS_PATH, "utf8");
    return JSON.parse(raw) as ResultsPayload;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ENOENT") {
      return null;
    }
    throw error;
  }
}

export async function queryEligibilityResults(params: {
  page?: number;
  pageSize?: number;
  facility?: string | null;
  decision?: RoutingDecision | "all" | null;
  q?: string | null;
}): Promise<PaginatedResultsPayload> {
  const payload = await loadResultsPayload();
  if (!payload) {
    return {
      generatedAt: null,
      rows: [],
      page: 1,
      pageSize: normalizePageSize(params.pageSize),
      totalRows: 0,
      totalPages: 0,
      errorCount: 0
    };
  }

  const pageSize = normalizePageSize(params.pageSize);
  const normalizedQuery = params.q?.trim().toLowerCase() ?? "";
  const filtered = payload.results.filter((result) => {
    const matchesFacility = !params.facility || params.facility === "all" || String(result.facilityId) === params.facility;
    const matchesDecision = !params.decision || params.decision === "all" || result.routingDecision === params.decision;
    const matchesQuery =
      !normalizedQuery ||
      result.patientName.toLowerCase().includes(normalizedQuery) ||
      result.patientId.toLowerCase().includes(normalizedQuery) ||
      (result.woundType ?? "").toLowerCase().includes(normalizedQuery) ||
      (result.location ?? "").toLowerCase().includes(normalizedQuery);

    return matchesFacility && matchesDecision && matchesQuery;
  });

  const totalRows = filtered.length;
  const totalPages = Math.max(1, Math.ceil(totalRows / pageSize));
  const page = clamp(params.page ?? 1, 1, totalPages);
  const offset = (page - 1) * pageSize;

  return {
    generatedAt: payload.generatedAt,
    rows: filtered.slice(offset, offset + pageSize).map(toListRow),
    page,
    pageSize,
    totalRows,
    totalPages: totalRows === 0 ? 0 : totalPages,
    errorCount: payload.errors.length
  };
}

export async function getEligibilityResult(patientId: string): Promise<EligibilityResult | null> {
  const payload = await loadResultsPayload();
  return payload?.results.find((result) => result.patientId === patientId) ?? null;
}

export function snapshotPath() {
  return SNAPSHOT_PATH;
}

export function resultsPath() {
  return RESULTS_PATH;
}

function toListRow(result: EligibilityResult): EligibilityListRow {
  const { evidence: _evidence, observations, ...row } = result;
  return {
    ...row,
    observationCount: observations.length
  };
}

function normalizePageSize(value: number | undefined) {
  if (!value || Number.isNaN(value)) return 50;
  return clamp(value, 10, 200);
}

function clamp(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), max);
}
