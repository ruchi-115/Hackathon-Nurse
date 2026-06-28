import { NextResponse } from "next/server";
import {
  backendErrorResponse,
  fetchBackend
} from "@/app/api/_lib/pipeline-data";
import type { Stats } from "@/src/lib/types";

export const dynamic = "force-dynamic";

export async function GET() {
  try {
    const data = await fetchBackend<Stats>("/stats");
    return NextResponse.json({ ...data, source: "backend" });
  } catch (error) {
    return backendErrorResponse(error);
  }
}
