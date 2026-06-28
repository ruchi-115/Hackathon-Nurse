import { NextResponse } from "next/server";
import { getEligibilityResult } from "@/src/lib/storage";

export const dynamic = "force-dynamic";

export async function GET(
  _request: Request,
  context: { params: Promise<{ patientId: string }> }
) {
  const { patientId } = await context.params;
  const result = await getEligibilityResult(decodeURIComponent(patientId));

  if (!result) {
    return NextResponse.json({ error: "Patient result not found" }, { status: 404 });
  }

  return NextResponse.json(result);
}
