import { NextResponse } from "next/server";
import { buildResults, emptyResults } from "@/src/lib/eligibility";
import { loadSnapshot } from "@/src/lib/storage";

export const dynamic = "force-dynamic";

export async function GET() {
  const snapshot = await loadSnapshot();
  if (!snapshot) {
    return NextResponse.json(emptyResults());
  }

  return NextResponse.json(buildResults(snapshot));
}
