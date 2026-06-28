// Package models defines the domain types mirroring the mock PCC API plus the
// derived extraction and eligibility types.
package models

// Patient mirrors GET /pcc/patients. Note the two identifiers:
//   - PatientID (string, e.g. "FA-001") keys /diagnoses and /coverage
//   - ID (int) keys /notes and /assessments
type Patient struct {
	ID              int     `json:"id"`
	FacilityID      int     `json:"facility_id"`
	PatientID       string  `json:"patient_id"`
	FirstName       *string `json:"first_name"`
	LastName        *string `json:"last_name"`
	BirthDate       *string `json:"birth_date"`
	Gender          *string `json:"gender"`
	PrimaryPayer    *string `json:"primary_payer_code"`
	LastModifiedAt  *string `json:"last_modified_at"`
	IsNewAdmission  bool    `json:"is_new_admission"`
}

// Diagnosis mirrors GET /pcc/diagnoses.
type Diagnosis struct {
	ID             int     `json:"id"`
	PatientID      string  `json:"patient_id"`
	ICD10Code      *string `json:"icd10_code"`
	ICD10Desc      *string `json:"icd10_description"`
	ClinicalStatus *string `json:"clinical_status"`
	OnsetDate      *string `json:"onset_date"`
	LastModifiedAt *string `json:"last_modified_at"`
}

// Coverage mirrors GET /pcc/coverage.
type Coverage struct {
	ID             int     `json:"id"`
	PatientID      string  `json:"patient_id"`
	PayerName      *string `json:"payer_name"`
	PayerCode      *string `json:"payer_code"`
	PayerType      *string `json:"payer_type"`
	EffectiveFrom  *string `json:"effective_from"`
	EffectiveTo    *string `json:"effective_to"`
	LastModifiedAt *string `json:"last_modified_at"`
}

// Note mirrors GET /pcc/notes. PatientID here is the integer internal id.
type Note struct {
	ID            int     `json:"id"`
	PatientID     int     `json:"patient_id"`
	OrgID         string  `json:"org_id"`
	PCCNoteID     *int    `json:"pcc_note_id"`
	NoteType      *string `json:"note_type"`
	EffectiveDate *string `json:"effective_date"`
	NoteText      *string `json:"note_text"`
	CreatedBy     *string `json:"created_by"`
	NoteLabel     *string `json:"note_label"`
}

// Assessment mirrors GET /pcc/assessments. RawJSON holds the structured wound data.
type Assessment struct {
	ID             int     `json:"id"`
	PatientID      int     `json:"patient_id"`
	OrgID          string  `json:"org_id"`
	PCCAssessID    *int    `json:"pcc_assessment_id"`
	AssessType     *string `json:"assessment_type"`
	Status         *string `json:"status"`
	AssessmentDate *string `json:"assessment_date"`
	CompletionDate *string `json:"completion_date"`
	TemplateID     *int    `json:"template_id"`
	AssessTypeDesc *string `json:"assessment_type_description"`
	RawJSON        *string `json:"raw_json"`
}

// Drainage levels per the spec.
const (
	DrainageNone     = "none"
	DrainageLight    = "light"
	DrainageModerate = "moderate"
	DrainageHeavy    = "heavy"
)

// WoundFields is the normalized clinical extraction target. Pointer fields are
// nil when a value could not be extracted, so routing can distinguish "absent"
// from a real zero value.
type WoundFields struct {
	WoundType       string   `json:"wound_type,omitempty"`
	Stage           *string  `json:"stage,omitempty"`
	Location        string   `json:"location,omitempty"`
	LengthCM        *float64 `json:"length_cm,omitempty"`
	WidthCM         *float64 `json:"width_cm,omitempty"`
	DepthCM         *float64 `json:"depth_cm,omitempty"`
	DrainageAmount  string   `json:"drainage_amount,omitempty"`
	HasSecondary    bool     `json:"has_secondary_wound,omitempty"`
	Confidence      float64  `json:"confidence"`        // 0..1
	Source          string   `json:"extraction_source"` // assessment|note_rules|llm|none
}

// HasMeasurements reports whether all three dimensions were extracted.
func (w WoundFields) HasMeasurements() bool {
	return w.LengthCM != nil && w.WidthCM != nil && w.DepthCM != nil
}

// Complete reports whether every required billing field is present.
func (w WoundFields) Complete() bool {
	return w.WoundType != "" && w.Location != "" && w.HasMeasurements() && w.DrainageAmount != ""
}

// Routing decisions.
const (
	DecisionAutoAccept = "auto_accept"
	DecisionFlag       = "flag_for_review"
	DecisionReject     = "reject"
)

// EligibilityRow is the one-row-per-patient biller-facing output.
type EligibilityRow struct {
	PatientID      string      `json:"patient_id"`
	InternalID     int         `json:"internal_id"`
	FacilityID     int         `json:"facility_id"`
	FirstName      *string     `json:"first_name"`
	LastName       *string     `json:"last_name"`
	PrimaryPayer   *string     `json:"primary_payer_code"`
	HasActiveMCB   bool        `json:"has_active_mcb"`
	Wound          WoundFields `json:"wound"`
	Decision       string      `json:"decision"`
	Reason         string      `json:"reason"`
}
