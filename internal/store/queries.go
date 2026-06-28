package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/Matrix030/hackathon-nurse/internal/models"
)

// EligibilityFilter narrows the eligibility listing.
type EligibilityFilter struct {
	FacilityID int    // 0 = any
	Decision   string // "" = any
	MCBOnly    bool
	Limit      int
	Offset     int
}

// ListEligibility returns eligibility rows matching the filter.
func (s *Store) ListEligibility(f EligibilityFilter) ([]models.EligibilityRow, error) {
	var where []string
	var args []any
	if f.FacilityID != 0 {
		where = append(where, "e.facility_id = ?")
		args = append(args, f.FacilityID)
	}
	if f.Decision != "" {
		where = append(where, "e.decision = ?")
		args = append(args, f.Decision)
	}
	if f.MCBOnly {
		where = append(where, "e.has_active_mcb = 1")
	}
	q := `
		SELECT e.patient_id, e.internal_id, e.facility_id, p.first_name, p.last_name,
		    p.primary_payer_code, e.has_active_mcb, e.wound_type, e.stage, e.location,
		    e.length_cm, e.width_cm, e.depth_cm, e.drainage_amount,
		    e.has_secondary_wound, e.confidence, e.extraction_source, e.decision, e.reason
		FROM eligibility e
		LEFT JOIN patient p ON p.id = e.internal_id`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY e.facility_id, e.patient_id"
	if f.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d OFFSET %d", f.Limit, f.Offset)
	}

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.EligibilityRow
	for rows.Next() {
		var r models.EligibilityRow
		var mcb, secondary int
		var stage *string
		// Nullable text columns scan into sql.NullString.
		var woundType, location, drainage, source sql.NullString
		if err := rows.Scan(&r.PatientID, &r.InternalID, &r.FacilityID, &r.FirstName,
			&r.LastName, &r.PrimaryPayer, &mcb, &woundType, &stage,
			&location, &r.Wound.LengthCM, &r.Wound.WidthCM, &r.Wound.DepthCM,
			&drainage, &secondary, &r.Wound.Confidence, &source,
			&r.Decision, &r.Reason); err != nil {
			return nil, err
		}
		r.HasActiveMCB = mcb == 1
		r.Wound.HasSecondary = secondary == 1
		r.Wound.Stage = stage
		r.Wound.WoundType = woundType.String
		r.Wound.Location = location.String
		r.Wound.DrainageAmount = drainage.String
		r.Wound.Source = source.String
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetEligibility returns a single eligibility row by external patient id.
func (s *Store) GetEligibility(patientID string) (*models.EligibilityRow, error) {
	rows, err := s.ListEligibility(EligibilityFilter{})
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].PatientID == patientID {
			return &rows[i], nil
		}
	}
	return nil, nil
}

// PatientDetail bundles a patient with all related records for the drill-down view.
type PatientDetail struct {
	Eligibility *models.EligibilityRow `json:"eligibility"`
	Diagnoses   []models.Diagnosis     `json:"diagnoses"`
	Coverage    []models.Coverage      `json:"coverage"`
	Notes       []models.Note          `json:"notes"`
	Assessments []models.Assessment    `json:"assessments"`
}

// GetPatientDetail loads the full record set for one external patient id.
func (s *Store) GetPatientDetail(patientID string) (*PatientDetail, error) {
	elig, err := s.GetEligibility(patientID)
	if err != nil {
		return nil, err
	}
	if elig == nil {
		return nil, nil
	}
	d := &PatientDetail{Eligibility: elig}

	if d.Diagnoses, err = s.diagnosesFor(patientID); err != nil {
		return nil, err
	}
	if d.Coverage, err = s.coverageFor(patientID); err != nil {
		return nil, err
	}
	if d.Notes, err = s.notesFor(elig.InternalID); err != nil {
		return nil, err
	}
	if d.Assessments, err = s.assessmentsFor(elig.InternalID); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Store) diagnosesFor(patientID string) ([]models.Diagnosis, error) {
	rows, err := s.db.Query(`SELECT id, patient_id, icd10_code, icd10_description,
		clinical_status, onset_date, last_modified_at FROM diagnosis WHERE patient_id=?`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Diagnosis
	for rows.Next() {
		var d models.Diagnosis
		if err := rows.Scan(&d.ID, &d.PatientID, &d.ICD10Code, &d.ICD10Desc,
			&d.ClinicalStatus, &d.OnsetDate, &d.LastModifiedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) coverageFor(patientID string) ([]models.Coverage, error) {
	rows, err := s.db.Query(`SELECT id, patient_id, payer_name, payer_code, payer_type,
		effective_from, effective_to, last_modified_at FROM coverage WHERE patient_id=?`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Coverage
	for rows.Next() {
		var c models.Coverage
		if err := rows.Scan(&c.ID, &c.PatientID, &c.PayerName, &c.PayerCode,
			&c.PayerType, &c.EffectiveFrom, &c.EffectiveTo, &c.LastModifiedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) notesFor(internalID int) ([]models.Note, error) {
	rows, err := s.db.Query(`SELECT id, patient_id, org_id, pcc_note_id, note_type,
		effective_date, note_text, created_by, note_label FROM note WHERE patient_id=?`, internalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Note
	for rows.Next() {
		var n models.Note
		if err := rows.Scan(&n.ID, &n.PatientID, &n.OrgID, &n.PCCNoteID, &n.NoteType,
			&n.EffectiveDate, &n.NoteText, &n.CreatedBy, &n.NoteLabel); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) assessmentsFor(internalID int) ([]models.Assessment, error) {
	rows, err := s.db.Query(`SELECT id, patient_id, org_id, pcc_assessment_id,
		assessment_type, status, assessment_date, completion_date, template_id,
		assessment_type_description, raw_json FROM assessment WHERE patient_id=?`, internalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Assessment
	for rows.Next() {
		var a models.Assessment
		if err := rows.Scan(&a.ID, &a.PatientID, &a.OrgID, &a.PCCAssessID, &a.AssessType,
			&a.Status, &a.AssessmentDate, &a.CompletionDate, &a.TemplateID,
			&a.AssessTypeDesc, &a.RawJSON); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Stats summarizes the eligibility table for dashboard cards.
type Stats struct {
	Total       int            `json:"total"`
	ByDecision  map[string]int `json:"by_decision"`
	ByFacility  map[string]int `json:"by_facility"`
	ActiveMCB   int            `json:"active_mcb"`
}

// Stats computes counts grouped by decision and facility.
func (s *Store) Stats() (*Stats, error) {
	st := &Stats{ByDecision: map[string]int{}, ByFacility: map[string]int{}}

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM eligibility`).Scan(&st.Total); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM eligibility WHERE has_active_mcb=1`).Scan(&st.ActiveMCB); err != nil {
		return nil, err
	}

	if err := scanCounts(s, `SELECT decision, COUNT(*) FROM eligibility GROUP BY decision`, st.ByDecision); err != nil {
		return nil, err
	}
	if err := scanCounts(s, `SELECT facility_id, COUNT(*) FROM eligibility GROUP BY facility_id`, st.ByFacility); err != nil {
		return nil, err
	}
	return st, nil
}

func scanCounts(s *Store, query string, dst map[string]int) error {
	rows, err := s.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var n int
		if err := rows.Scan(&key, &n); err != nil {
			return err
		}
		dst[key] = n
	}
	return rows.Err()
}

// AllInternalIDs returns every patient's (externalID, internalID, facilityID)
// for the extraction pass.
func (s *Store) AllPatients() ([]models.Patient, error) {
	rows, err := s.db.Query(`SELECT id, patient_id, facility_id, first_name, last_name,
		birth_date, gender, primary_payer_code, last_modified_at, is_new_admission FROM patient`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Patient
	for rows.Next() {
		var p models.Patient
		var newAdm int
		if err := rows.Scan(&p.ID, &p.PatientID, &p.FacilityID, &p.FirstName, &p.LastName,
			&p.BirthDate, &p.Gender, &p.PrimaryPayer, &p.LastModifiedAt, &newAdm); err != nil {
			return nil, err
		}
		p.IsNewAdmission = newAdm == 1
		out = append(out, p)
	}
	return out, rows.Err()
}

// Raw record loaders used by the extraction pass (by external/internal id).
func (s *Store) DiagnosesFor(patientID string) ([]models.Diagnosis, error) {
	return s.diagnosesFor(patientID)
}
func (s *Store) CoverageFor(patientID string) ([]models.Coverage, error) {
	return s.coverageFor(patientID)
}
func (s *Store) NotesFor(internalID int) ([]models.Note, error) { return s.notesFor(internalID) }
func (s *Store) AssessmentsFor(internalID int) ([]models.Assessment, error) {
	return s.assessmentsFor(internalID)
}
