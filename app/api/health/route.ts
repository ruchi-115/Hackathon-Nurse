import { NextResponse } from "next/server";
import { loadSnapshot, snapshotPath } from "@/src/lib/storage";

export const dynamic = "force-dynamic";

export async function GET() {
  const snapshot = await loadSnapshot();

  return NextResponse.json({
    ok: true,
    snapshotPath: snapshotPath(),
    hasSnapshot: Boolean(snapshot),
    generatedAt: snapshot?.generatedAt ?? null,
    patientCount: snapshot?.patientCount ?? 0
  });
}
