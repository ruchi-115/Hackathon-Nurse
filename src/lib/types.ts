export type FacilityId = 101 | 102 | 103;

export type RoutingDecision = "auto_accept" | "flag_for_review" | "reject";

export type DrainageAmount = "none" | "light" | "moderate" | "heavy" | null;

export type Patient = {
  id: number;
  facility_id: FacilityId;
  patient_id: string;
  first_name: string | null;
  last_name: string | null;
  birth_date: string | null;
  gender: string | null;
  primary_payer_code: string | null;
  last_modified_at: string | null;
  is_new_admission: boolean;
};

export type Diagnosis = {
  id: number;
  patient_id: string;
  icd10_code: string | null;
  icd10_description: string | null;
  clinical_status: string | null;
  onset_date: string | null;
  last_modified_at: string | null;
};

export type Coverage = {
  id: number;
  patient_id: string;
  payer_name: string | null;
  payer_code: string | null;
  payer_type: string | null;
  effective_from: string | null;
  effective_to: string | null;
  last_modified_at: string | null;
};

export type ProgressNote = {
  id: number;
  patient_id: number;
  org_id: string | null;
  pcc_note_id: number | null;
  note_type: string | null;
  effective_date: string | null;
  note_text: string | null;
  created_by: string | null;
  note_label: string | null;
  sync_version: number | null;
  is_current: boolean;
};

export type Assessment = {
  id: number;
  patient_id: number;
  org_id: string | null;
  pcc_assessment_id: number | null;
  assessment_type: string | null;
  status: string | null;
  assessment_date: string | null;
  completion_date: string | null;
  template_id: number | null;
  assessment_type_description: string | null;
  raw_json: string | null;
  sync_version: number | null;
  is_current: boolean;
};

export type PatientBundle = {
  patient: Patient;
  diagnoses: Diagnosis[];
  coverage: Coverage[];
  notes: ProgressNote[];
  assessments: Assessment[];
};

export type SyncSnapshot = {
  generatedAt: string;
  baseUrl: string;
  facilities: FacilityId[];
  patientCount: number;
  bundles: PatientBundle[];
  errors: SyncError[];
};

export type SyncError = {
  patient_id?: string;
  internal_id?: number;
  facility_id?: FacilityId;
  endpoint: string;
  message: string;
};

export type WoundObservation = {
  woundType: string | null;
  stage: string | null;
  location: string | null;
  lengthCm: number | null;
  widthCm: number | null;
  depthCm: number | null;
  drainageAmount: DrainageAmount;
  source: "assessment" | "note" | "diagnosis";
  sourceDate: string | null;
  sourceLabel: string;
  confidence: number;
  evidence: string;
};

export type EligibilityResult = {
  patientId: string;
  internalId: number;
  facilityId: FacilityId;
  patientName: string;
  birthDate: string | null;
  primaryPayerCode: string | null;
  hasActiveMedicareB: boolean;
  activeCoverageLabel: string | null;
  hasActiveWoundDiagnosis: boolean;
  woundType: string | null;
  stage: string | null;
  location: string | null;
  lengthCm: number | null;
  widthCm: number | null;
  depthCm: number | null;
  drainageAmount: DrainageAmount;
  source: string | null;
  sourceDate: string | null;
  confidence: number;
  routingDecision: RoutingDecision;
  reason: string;
  evidence: string;
  observations: WoundObservation[];
};

export type ResultsPayload = {
  generatedAt: string | null;
  summary: {
    totalPatients: number;
    autoAccept: number;
    flagForReview: number;
    reject: number;
    medicareB: number;
    withWoundEvidence: number;
  };
  results: EligibilityResult[];
  errors: SyncError[];
};
