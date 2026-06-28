export type Decision = "auto_accept" | "flag_for_review" | "reject";

export type WoundFields = {
  wound_type?: string;
  stage?: string | null;
  location?: string;
  length_cm?: number | null;
  width_cm?: number | null;
  depth_cm?: number | null;
  drainage_amount?: string;
  has_secondary_wound?: boolean;
  confidence: number;
  extraction_source?: string;
};

export type EligibilityRow = {
  patient_id: string;
  internal_id: number;
  facility_id: number;
  first_name?: string | null;
  last_name?: string | null;
  primary_payer_code?: string | null;
  has_active_mcb: boolean;
  wound: WoundFields;
  decision: Decision;
  reason: string;
};

export type EligibilityResponse = {
  count: number;
  results: EligibilityRow[];
  source?: "backend";
};

export type Stats = {
  total: number;
  by_decision: Partial<Record<Decision, number>>;
  by_facility: Record<string, number>;
  active_mcb: number;
  source?: "backend";
};

export type Diagnosis = {
  id: number;
  patient_id: string;
  icd10_code?: string | null;
  icd10_description?: string | null;
  clinical_status?: string | null;
  onset_date?: string | null;
  last_modified_at?: string | null;
};

export type Coverage = {
  id: number;
  patient_id: string;
  payer_name?: string | null;
  payer_code?: string | null;
  payer_type?: string | null;
  effective_from?: string | null;
  effective_to?: string | null;
  last_modified_at?: string | null;
};

export type Note = {
  id: number;
  patient_id: number;
  note_type?: string | null;
  effective_date?: string | null;
  note_text?: string | null;
  note_label?: string | null;
};

export type Assessment = {
  id: number;
  patient_id: number;
  assessment_type?: string | null;
  status?: string | null;
  assessment_date?: string | null;
  assessment_type_description?: string | null;
  raw_json?: string | null;
};

export type Observation = {
  woundType?: string | null;
  stage?: string | null;
  location?: string | null;
  lengthCm?: number | null;
  widthCm?: number | null;
  depthCm?: number | null;
  drainageAmount?: string | null;
  source?: string | null;
  sourceDate?: string | null;
  sourceLabel?: string | null;
  confidence?: number | null;
  evidence?: string | null;
};

export type PatientDetail = {
  eligibility: EligibilityRow;
  diagnoses: Diagnosis[];
  coverage: Coverage[];
  notes: Note[];
  assessments: Assessment[];
  observations?: Observation[];
};

export type DashboardFilters = {
  facility?: string;
  decision?: string;
  mcbOnly?: boolean;
};
