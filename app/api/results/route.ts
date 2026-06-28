import { NextResponse } from "next/server";
import { queryEligibilityResults } from "@/src/lib/storage";
import type { RoutingDecision } from "@/src/lib/types";

export const dynamic = "force-dynamic";

const decisions = new Set(["all", "auto_accept", "flag_for_review", "reject"]);

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const decision = searchParams.get("decision");

  const payload = await queryEligibilityResults({
    page: toNumber(searchParams.get("page")),
    pageSize: toNumber(searchParams.get("pageSize")),
    facility: searchParams.get("facility"),
    decision: decisions.has(decision ?? "") ? (decision as RoutingDecision | "all") : "all",
    q: searchParams.get("q")
  });

  return NextResponse.json(payload);
}

function toNumber(value: string | null) {
  if (!value) return undefined;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}
