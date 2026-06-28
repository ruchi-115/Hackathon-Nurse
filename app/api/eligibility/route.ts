import { NextResponse } from "next/server";
import {
  backendErrorResponse,
  fetchBackend
} from "@/app/api/_lib/pipeline-data";
import type { EligibilityResponse } from "@/src/lib/types";

export const dynamic = "force-dynamic";

export async function GET(request: Request) {
  const query = new URL(request.url).search;
  try {
    const data = await fetchBackend<EligibilityResponse>(`/eligibility${query}`);
    return NextResponse.json({ ...data, source: "backend" });
  } catch (error) {
    return backendErrorResponse(error);
  }
}
