import { NextResponse } from "next/server";
import { emptyResults } from "@/src/lib/eligibility";
import { loadResultsPayload } from "@/src/lib/storage";

export const dynamic = "force-dynamic";

export async function GET() {
  const payload = await loadResultsPayload();

  if (!payload) {
    return NextResponse.json({
      generatedAt: null,
      summary: emptyResults().summary,
      errorCount: 0
    });
  }

  return NextResponse.json({
    generatedAt: payload.generatedAt,
    summary: payload.summary,
    errorCount: payload.errors.length
  });
}
