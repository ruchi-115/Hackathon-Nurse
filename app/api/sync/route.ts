import { NextResponse } from "next/server";
import { PccClient, settleEndpoint } from "@/src/lib/pccClient";
import { buildResults } from "@/src/lib/eligibility";
import { saveSnapshot } from "@/src/lib/storage";
import type { FacilityId, Patient, PatientBundle, SyncError, SyncSnapshot } from "@/src/lib/types";

export const dynamic = "force-dynamic";
export const maxDuration = 120;

const FACILITIES: FacilityId[] = [101, 102, 103];
const CONCURRENCY = 8;

export async function POST(request: Request) {
  const body = await readBody(request);
  const since = typeof body.since === "string" && body.since.trim() ? body.since.trim() : undefined;
  const client = new PccClient();
  const errors: SyncError[] = [];

  const patientGroups = await Promise.all(
    FACILITIES.map(async (facilityId) => {
      const result = await settleEndpoint(
        "patients",
        () => client.getPatients(facilityId, since),
        { facility_id: facilityId }
      );
      if (result.error) errors.push(result.error);
      return result.data ?? [];
    })
  );

  const patients = patientGroups.flat();
  const bundles = await mapWithConcurrency(patients, CONCURRENCY, async (patient) => {
    const bundle = await fetchPatientBundle(client, patient, since);
    errors.push(...bundle.errors);
    return bundle.data;
  });

  const snapshot: SyncSnapshot = {
    generatedAt: new Date().toISOString(),
    baseUrl: client.url,
    facilities: FACILITIES,
    patientCount: bundles.length,
    bundles,
    errors
  };

  await saveSnapshot(snapshot);

  return NextResponse.json({
    snapshot: {
      generatedAt: snapshot.generatedAt,
      patientCount: snapshot.patientCount,
      errorCount: snapshot.errors.length
    },
    results: buildResults(snapshot)
  });
}

async function fetchPatientBundle(client: PccClient, patient: Patient, since?: string) {
  const context = {
    patient_id: patient.patient_id,
    internal_id: patient.id,
    facility_id: patient.facility_id
  };

  const [diagnoses, coverage, notes, assessments] = await Promise.all([
    settleEndpoint("diagnoses", () => client.getDiagnoses(patient.patient_id), context),
    settleEndpoint("coverage", () => client.getCoverage(patient.patient_id), context),
    settleEndpoint("notes", () => client.getNotes(patient.id, since), context),
    settleEndpoint("assessments", () => client.getAssessments(patient.id, since), context)
  ]);

  return {
    data: {
      patient,
      diagnoses: diagnoses.data ?? [],
      coverage: coverage.data ?? [],
      notes: notes.data ?? [],
      assessments: assessments.data ?? []
    } satisfies PatientBundle,
    errors: [diagnoses.error, coverage.error, notes.error, assessments.error].filter(Boolean) as SyncError[]
  };
}

async function mapWithConcurrency<T, R>(
  items: T[],
  concurrency: number,
  mapper: (item: T, index: number) => Promise<R>
) {
  const results = new Array<R>(items.length);
  let nextIndex = 0;

  async function worker() {
    while (nextIndex < items.length) {
      const currentIndex = nextIndex;
      nextIndex += 1;
      results[currentIndex] = await mapper(items[currentIndex], currentIndex);
    }
  }

  await Promise.all(Array.from({ length: Math.min(concurrency, items.length) }, worker));
  return results;
}

async function readBody(request: Request): Promise<Record<string, unknown>> {
  try {
    return (await request.json()) as Record<string, unknown>;
  } catch {
    return {};
  }
}
