import { extractWoundObservations } from "./extraction";
import type {
  Coverage,
  EligibilityResult,
  PatientBundle,
  ResultsPayload,
  RoutingDecision,
  SyncSnapshot,
  WoundObservation
} from "./types";

export function buildResults(snapshot: SyncSnapshot): ResultsPayload {
  const results = snapshot.bundles.map(evaluatePatient);

  return {
    generatedAt: snapshot.generatedAt,
    summary: {
      totalPatients: results.length,
      autoAccept: countDecision(results, "auto_accept"),
      flagForReview: countDecision(results, "flag_for_review"),
      reject: countDecision(results, "reject"),
      medicareB: results.filter((result) => result.hasActiveMedicareB).length,
      withWoundEvidence: results.filter((result) => result.observations.length > 0).length
    },
    results,
    errors: snapshot.errors
  };
}

export function emptyResults(): ResultsPayload {
  return {
    generatedAt: null,
    summary: {
      totalPatients: 0,
      autoAccept: 0,
      flagForReview: 0,
      reject: 0,
      medicareB: 0,
      withWoundEvidence: 0
    },
    results: [],
    errors: []
  };
}

function evaluatePatient(bundle: PatientBundle): EligibilityResult {
  const observations = extractWoundObservations(bundle);
  const bestObservation = choosePrimaryObservation(observations);
  const activeMedicareB = bundle.coverage.find(isActiveMedicareB) ?? null;
  const hasActiveWoundDiagnosis = observations.some((observation) => observation.source === "diagnosis");
  const requiredComplete = Boolean(
    bestObservation?.woundType &&
      bestObservation.lengthCm !== null &&
      bestObservation.widthCm !== null &&
      bestObservation.depthCm !== null &&
      bestObservation.drainageAmount
  );

  const routingDecision = decide({
    hasActiveMedicareB: Boolean(activeMedicareB),
    hasWoundEvidence: observations.length > 0,
    requiredComplete,
    confidence: bestObservation?.confidence ?? 0
  });

  return {
    patientId: bundle.patient.patient_id,
    internalId: bundle.patient.id,
    facilityId: bundle.patient.facility_id,
    patientName: [bundle.patient.first_name, bundle.patient.last_name].filter(Boolean).join(" ") || bundle.patient.patient_id,
    birthDate: bundle.patient.birth_date,
    primaryPayerCode: bundle.patient.primary_payer_code,
    hasActiveMedicareB: Boolean(activeMedicareB),
    activeCoverageLabel: activeMedicareB ? formatCoverage(activeMedicareB) : null,
    hasActiveWoundDiagnosis,
    woundType: bestObservation?.woundType ?? null,
    stage: bestObservation?.stage ?? null,
    location: bestObservation?.location ?? null,
    lengthCm: bestObservation?.lengthCm ?? null,
    widthCm: bestObservation?.widthCm ?? null,
    depthCm: bestObservation?.depthCm ?? null,
    drainageAmount: bestObservation?.drainageAmount ?? null,
    source: bestObservation?.sourceLabel ?? null,
    sourceDate: bestObservation?.sourceDate ?? null,
    confidence: bestObservation?.confidence ?? 0,
    routingDecision,
    reason: buildReason(routingDecision, Boolean(activeMedicareB), requiredComplete, bestObservation),
    evidence: bestObservation?.evidence ?? "No active wound evidence found in diagnoses, notes, or assessments.",
    observations
  };
}

function choosePrimaryObservation(observations: WoundObservation[]) {
  return observations.find((observation) => observation.lengthCm !== null && observation.widthCm !== null && observation.depthCm !== null) ?? observations[0] ?? null;
}

function decide(input: {
  hasActiveMedicareB: boolean;
  hasWoundEvidence: boolean;
  requiredComplete: boolean;
  confidence: number;
}): RoutingDecision {
  if (!input.hasActiveMedicareB) return "reject";
  if (!input.hasWoundEvidence) return "reject";
  if (input.requiredComplete && input.confidence >= 0.72) return "auto_accept";
  return "flag_for_review";
}

function buildReason(
  decision: RoutingDecision,
  hasActiveMedicareB: boolean,
  requiredComplete: boolean,
  observation: WoundObservation | null
) {
  if (!hasActiveMedicareB) {
    return "Rejected because active Medicare Part B coverage was not found.";
  }

  if (!observation) {
    return "Rejected because no active wound documentation was found.";
  }

  if (decision === "auto_accept") {
    const measurement = `${observation.lengthCm} x ${observation.widthCm} x ${observation.depthCm} cm`;
    return `Ready for billing review: active Medicare Part B coverage and ${observation.woundType} documentation with ${measurement} measurements and ${observation.drainageAmount} drainage.`;
  }

  if (!requiredComplete) {
    return `Needs review because wound evidence exists, but one or more required fields are missing: ${missingFields(observation).join(", ")}.`;
  }

  return "Needs review because the wound documentation is complete but extraction confidence is below the auto-accept threshold.";
}

function missingFields(observation: WoundObservation) {
  const missing: string[] = [];
  if (!observation.woundType) missing.push("wound type");
  if (observation.lengthCm === null) missing.push("length");
  if (observation.widthCm === null) missing.push("width");
  if (observation.depthCm === null) missing.push("depth");
  if (!observation.drainageAmount) missing.push("drainage amount");
  return missing;
}

function isActiveMedicareB(coverage: Coverage) {
  const isMedicareB = coverage.payer_code === "MCB" || coverage.payer_type?.toLowerCase() === "medicare b";
  if (!isMedicareB) return false;

  if (!coverage.effective_to) return true;
  return new Date(coverage.effective_to).getTime() >= Date.now();
}

function formatCoverage(coverage: Coverage) {
  return `${coverage.payer_name ?? "Medicare Part B"}${coverage.effective_from ? ` since ${coverage.effective_from.slice(0, 10)}` : ""}`;
}

function countDecision(results: EligibilityResult[], decision: RoutingDecision) {
  return results.filter((result) => result.routingDecision === decision).length;
}
