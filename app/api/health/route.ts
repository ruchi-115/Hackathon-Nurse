import { NextResponse } from "next/server";
import { loadResultsPayload, loadSnapshot, resultsPath, snapshotPath } from "@/src/lib/storage";

export const dynamic = "force-dynamic";

export async function GET() {
  const [snapshot, results] = await Promise.all([loadSnapshot(), loadResultsPayload()]);

  return NextResponse.json({
    ok: true,
    snapshotPath: snapshotPath(),
    resultsPath: resultsPath(),
    hasSnapshot: Boolean(snapshot),
    hasResultsCache: Boolean(results),
    generatedAt: snapshot?.generatedAt ?? null,
    patientCount: snapshot?.patientCount ?? 0,
    resultCount: results?.results.length ?? 0
  });
}
