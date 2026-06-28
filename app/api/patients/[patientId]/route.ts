import { NextResponse } from "next/server";
import {
  backendErrorResponse,
  fetchBackend
} from "@/app/api/_lib/pipeline-data";
import type { PatientDetail } from "@/src/lib/types";

export const dynamic = "force-dynamic";

export async function GET(
  _request: Request,
  context: { params: Promise<{ patientId: string }> }
) {
  const { patientId } = await context.params;
  try {
    const data = await fetchBackend<PatientDetail>(
      `/patients/${encodeURIComponent(patientId)}`
    );
    return NextResponse.json(data);
  } catch (error) {
    return backendErrorResponse(error);
  }
}
