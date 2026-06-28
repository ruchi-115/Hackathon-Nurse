import type {
  Assessment,
  Diagnosis,
  PatientBundle,
  ProgressNote,
  WoundObservation
} from "./types";

const WOUND_TERMS = [
  "pressure ulcer",
  "diabetic foot ulcer",
  "venous stasis ulcer",
  "venous ulcer",
  "arterial ulcer",
  "surgical site infection",
  "abscess",
  "burn"
];

export function extractWoundObservations(bundle: PatientBundle): WoundObservation[] {
  const assessmentObservations = bundle.assessments.flatMap(extractAssessmentWound);
  const noteObservations = bundle.notes.flatMap(extractNoteWound);
  const diagnosisObservations = extractDiagnosisWounds(bundle.diagnoses);

  return [...assessmentObservations, ...noteObservations, ...diagnosisObservations]
    .filter((observation) => observation.woundType || observation.evidence)
    .sort(compareObservationQuality);
}

function extractAssessmentWound(assessment: Assessment): WoundObservation[] {
  if (!assessment.raw_json) return [];

  try {
    const parsed = JSON.parse(assessment.raw_json) as Record<string, unknown>;
    const woundType = normalizeWoundType(readString(parsed.wound_type ?? parsed.type));
    const drainageAmount = normalizeDrainage(readString(parsed.drainage_amount));
    const observation: WoundObservation = {
      woundType,
      stage: normalizeStage(readString(parsed.stage)),
      location: readString(parsed.location),
      lengthCm: readNumber(parsed.length_cm),
      widthCm: readNumber(parsed.width_cm),
      depthCm: readNumber(parsed.depth_cm),
      drainageAmount,
      source: "assessment",
      sourceDate: assessment.assessment_date,
      sourceLabel: assessment.assessment_type ?? "Structured wound assessment",
      confidence: scoreObservation("assessment", {
        woundType,
        lengthCm: readNumber(parsed.length_cm),
        widthCm: readNumber(parsed.width_cm),
        depthCm: readNumber(parsed.depth_cm),
        drainageAmount
      }),
      evidence: compactEvidence([
        assessment.assessment_type,
        woundType,
        readString(parsed.location),
        formatMeasurements(
          readNumber(parsed.length_cm),
          readNumber(parsed.width_cm),
          readNumber(parsed.depth_cm)
        ),
        drainageAmount ? `${drainageAmount} drainage` : null
      ])
    };

    return [observation];
  } catch {
    return [];
  }
}

function extractNoteWound(note: ProgressNote): WoundObservation[] {
  const text = note.note_text?.trim();
  if (!text) return [];

  const candidates = splitIntoWoundCandidates(text);
  return candidates.map((candidate) => {
    const woundType = extractWoundType(candidate);
    const measurements = extractMeasurements(candidate);
    const drainageAmount = extractDrainage(candidate);
    const isNarrative = isNarrativeNote(note, candidate);

    return {
      woundType,
      stage: extractStage(candidate),
      location: extractLocation(candidate),
      lengthCm: measurements.lengthCm,
      widthCm: measurements.widthCm,
      depthCm: measurements.depthCm,
      drainageAmount,
      source: "note",
      sourceDate: note.effective_date,
      sourceLabel: note.note_type ?? "Progress note",
      confidence: scoreObservation(isNarrative ? "narrative_note" : "note", {
        woundType,
        lengthCm: measurements.lengthCm,
        widthCm: measurements.widthCm,
        depthCm: measurements.depthCm,
        drainageAmount
      }),
      evidence: excerpt(candidate, woundType)
    } satisfies WoundObservation;
  });
}

function extractDiagnosisWounds(diagnoses: Diagnosis[]): WoundObservation[] {
  return diagnoses
    .filter((diagnosis) => diagnosis.clinical_status?.toLowerCase() === "active")
    .filter((diagnosis) => extractWoundType(diagnosis.icd10_description ?? ""))
    .map((diagnosis) => ({
      woundType: extractWoundType(diagnosis.icd10_description ?? ""),
      stage: extractStage(diagnosis.icd10_description ?? ""),
      location: extractLocationFromDiagnosis(diagnosis.icd10_description ?? ""),
      lengthCm: null,
      widthCm: null,
      depthCm: null,
      drainageAmount: null,
      source: "diagnosis",
      sourceDate: diagnosis.onset_date,
      sourceLabel: diagnosis.icd10_code ?? "Active diagnosis",
      confidence: 0.45,
      evidence: diagnosis.icd10_description ?? diagnosis.icd10_code ?? "Active wound diagnosis"
    }));
}

function splitIntoWoundCandidates(text: string) {
  const parts = text
    .split(/\n(?=(?:wound|site|ulcer|location)\b|\d+\.\s)/i)
    .map((part) => part.trim())
    .filter(Boolean);

  if (parts.length <= 1) return [text];
  return parts.filter((part) => WOUND_TERMS.some((term) => part.toLowerCase().includes(term)));
}

function extractWoundType(text: string) {
  const lower = text.toLowerCase();
  const matched = WOUND_TERMS.find((term) => lower.includes(term));
  if (!matched) return null;
  if (matched === "venous ulcer") return "venous stasis ulcer";
  return matched;
}

function extractStage(text: string) {
  const stageMatch =
    text.match(/stage\s*(?:\:|-)?\s*(unstageable|[2-4]|ii|iii|iv)\b/i) ??
    text.match(/pressure ulcer[^.\n]*(unstageable|stage\s*[2-4]|stage\s*ii|stage\s*iii|stage\s*iv)/i);

  if (!stageMatch) return null;
  return normalizeStage(stageMatch[1]);
}

function extractLocation(text: string) {
  const labeled =
    text.match(/location\s*(?:\:|-)\s*([a-z0-9 /-]+?)(?:\n|\.|,|;| wound| type| length| width| depth| drainage)/i) ??
    text.match(/\b(?:at|on|to)\s+(?:the\s+)?([a-z]+(?:\s+[a-z]+){0,3})\s+(?:with|meas|measuring|ulcer|wound)/i);

  return cleanValue(labeled?.[1] ?? null);
}

function extractLocationFromDiagnosis(text: string) {
  const match = text.match(/of\s+(.+?)(?:,\s*stage|\s+stage|$)/i);
  return cleanValue(match?.[1] ?? null);
}

function extractMeasurements(text: string) {
  const shorthand = text.match(
    /(?:meas(?:ures|urement|\.?)?\s*)?(\d+(?:\.\d+)?)\s*(?:cm)?\s*x\s*(\d+(?:\.\d+)?)\s*(?:cm)?\s*x\s*(\d+(?:\.\d+)?)\s*cm?/i
  );

  if (shorthand) {
    return {
      lengthCm: toNumber(shorthand[1]),
      widthCm: toNumber(shorthand[2]),
      depthCm: toNumber(shorthand[3])
    };
  }

  return {
    lengthCm: extractLabeledNumber(text, "length"),
    widthCm: extractLabeledNumber(text, "width"),
    depthCm: extractLabeledNumber(text, "depth")
  };
}

function extractDrainage(text: string) {
  const lower = text.toLowerCase();
  const labeled = lower.match(/drainage\s*(?:\:|-)?\s*(none|light|small|scant|moderate|heavy|large|copious)/i);
  const raw = labeled?.[1] ?? lower.match(/\b(none|light|small|scant|moderate|heavy|large|copious)\s+drainage\b/i)?.[1];
  return normalizeDrainage(raw ?? null);
}

function extractLabeledNumber(text: string, label: string) {
  const match = text.match(new RegExp(`${label}\\s*(?:\\:|-)?\\s*(\\d+(?:\\.\\d+)?)\\s*cm`, "i"));
  return toNumber(match?.[1] ?? null);
}

function normalizeWoundType(value: string | null) {
  if (!value) return null;
  return value.replace(/_/g, " ").toLowerCase();
}

function normalizeStage(value: string | null) {
  if (!value) return null;
  const lower = value.toLowerCase().replace(/^stage\s*/, "").trim();
  if (lower === "ii") return "2";
  if (lower === "iii") return "3";
  if (lower === "iv") return "4";
  if (lower.includes("unstageable")) return "unstageable";
  return lower;
}

function normalizeDrainage(value: string | null) {
  if (!value) return null;
  const lower = value.toLowerCase();
  if (["small", "scant"].includes(lower)) return "light";
  if (["large", "copious"].includes(lower)) return "heavy";
  if (["none", "light", "moderate", "heavy"].includes(lower)) {
    return lower as "none" | "light" | "moderate" | "heavy";
  }
  return null;
}

function readString(value: unknown) {
  if (value === null || value === undefined) return null;
  return String(value).trim() || null;
}

function readNumber(value: unknown) {
  if (typeof value === "number") return Number.isFinite(value) ? value : null;
  if (typeof value === "string") return toNumber(value);
  return null;
}

function toNumber(value: string | null) {
  if (!value) return null;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

function cleanValue(value: string | null) {
  if (!value) return null;
  return value
    .replace(/\s+/g, " ")
    .replace(/\b(left|right)\b$/i, "")
    .trim()
    .replace(/[.,;:]$/, "");
}

function scoreObservation(
  source: "assessment" | "note" | "narrative_note",
  fields: {
    woundType: string | null;
    lengthCm: number | null;
    widthCm: number | null;
    depthCm: number | null;
    drainageAmount: string | null;
  }
) {
  let score = source === "assessment" ? 0.45 : source === "note" ? 0.3 : 0.2;
  if (fields.woundType) score += 0.15;
  if (fields.lengthCm !== null) score += 0.1;
  if (fields.widthCm !== null) score += 0.1;
  if (fields.depthCm !== null) score += 0.1;
  if (fields.drainageAmount) score += 0.1;
  return Math.min(0.98, Number(score.toFixed(2)));
}

function compareObservationQuality(a: WoundObservation, b: WoundObservation) {
  return b.confidence - a.confidence || dateValue(b.sourceDate) - dateValue(a.sourceDate);
}

function dateValue(value: string | null) {
  return value ? new Date(value).getTime() : 0;
}

function isNarrativeNote(note: ProgressNote, text: string) {
  return /envive|narrative/i.test(note.note_type ?? "") || !/location\s*:|wound type\s*:|length\s*:/i.test(text);
}

function excerpt(text: string, woundType: string | null) {
  const normalized = text.replace(/\s+/g, " ").trim();
  if (!woundType) return normalized.slice(0, 240);

  const index = normalized.toLowerCase().indexOf(woundType.toLowerCase());
  if (index === -1) return normalized.slice(0, 240);
  return normalized.slice(Math.max(0, index - 50), index + 190).trim();
}

function formatMeasurements(lengthCm: number | null, widthCm: number | null, depthCm: number | null) {
  if (lengthCm === null || widthCm === null || depthCm === null) return null;
  return `${lengthCm} x ${widthCm} x ${depthCm} cm`;
}

function compactEvidence(parts: Array<string | null>) {
  return parts.filter(Boolean).join(" | ");
}
