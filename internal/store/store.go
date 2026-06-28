// Package store is a thin database/sql wrapper over SQLite holding the raw PCC
// data and the derived eligibility table. Upserts are keyed on primary keys so
// the pipeline is safe to rerun and supports incremental `since` syncs.
package store

import (
	"database/sql"
	_ "embed"
	"fmt"

	"github.com/Matrix030/hackathon-nurse/internal/models"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Store wraps a SQLite connection.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the SQLite database and applies the schema.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// SQLite handles concurrency best with a single writer; serialize writes.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

// DB exposes the raw handle for read queries in the API layer.
func (s *Store) DB() *sql.DB { return s.db }

// UpsertPatient inserts or replaces a patient row.
func (s *Store) UpsertPatient(p models.Patient) error {
	_, err := s.db.Exec(`
		INSERT INTO patient (id, patient_id, facility_id, first_name, last_name,
		    birth_date, gender, primary_payer_code, last_modified_at, is_new_admission)
		VALUES (?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
		    patient_id=excluded.patient_id, facility_id=excluded.facility_id,
		    first_name=excluded.first_name, last_name=excluded.last_name,
		    birth_date=excluded.birth_date, gender=excluded.gender,
		    primary_payer_code=excluded.primary_payer_code,
		    last_modified_at=excluded.last_modified_at,
		    is_new_admission=excluded.is_new_admission`,
		p.ID, p.PatientID, p.FacilityID, p.FirstName, p.LastName, p.BirthDate,
		p.Gender, p.PrimaryPayer, p.LastModifiedAt, boolToInt(p.IsNewAdmission))
	return err
}

// UpsertDiagnosis inserts or replaces a diagnosis row.
func (s *Store) UpsertDiagnosis(d models.Diagnosis) error {
	_, err := s.db.Exec(`
		INSERT INTO diagnosis (id, patient_id, icd10_code, icd10_description,
		    clinical_status, onset_date, last_modified_at)
		VALUES (?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
		    patient_id=excluded.patient_id, icd10_code=excluded.icd10_code,
		    icd10_description=excluded.icd10_description,
		    clinical_status=excluded.clinical_status, onset_date=excluded.onset_date,
		    last_modified_at=excluded.last_modified_at`,
		d.ID, d.PatientID, d.ICD10Code, d.ICD10Desc, d.ClinicalStatus,
		d.OnsetDate, d.LastModifiedAt)
	return err
}

// UpsertCoverage inserts or replaces a coverage row.
func (s *Store) UpsertCoverage(c models.Coverage) error {
	_, err := s.db.Exec(`
		INSERT INTO coverage (id, patient_id, payer_name, payer_code, payer_type,
		    effective_from, effective_to, last_modified_at)
		VALUES (?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
		    patient_id=excluded.patient_id, payer_name=excluded.payer_name,
		    payer_code=excluded.payer_code, payer_type=excluded.payer_type,
		    effective_from=excluded.effective_from, effective_to=excluded.effective_to,
		    last_modified_at=excluded.last_modified_at`,
		c.ID, c.PatientID, c.PayerName, c.PayerCode, c.PayerType,
		c.EffectiveFrom, c.EffectiveTo, c.LastModifiedAt)
	return err
}

// UpsertNote inserts or replaces a note row.
func (s *Store) UpsertNote(n models.Note) error {
	_, err := s.db.Exec(`
		INSERT INTO note (id, patient_id, org_id, pcc_note_id, note_type,
		    effective_date, note_text, created_by, note_label)
		VALUES (?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
		    patient_id=excluded.patient_id, org_id=excluded.org_id,
		    pcc_note_id=excluded.pcc_note_id, note_type=excluded.note_type,
		    effective_date=excluded.effective_date, note_text=excluded.note_text,
		    created_by=excluded.created_by, note_label=excluded.note_label`,
		n.ID, n.PatientID, n.OrgID, n.PCCNoteID, n.NoteType, n.EffectiveDate,
		n.NoteText, n.CreatedBy, n.NoteLabel)
	return err
}

// UpsertAssessment inserts or replaces an assessment row.
func (s *Store) UpsertAssessment(a models.Assessment) error {
	_, err := s.db.Exec(`
		INSERT INTO assessment (id, patient_id, org_id, pcc_assessment_id,
		    assessment_type, status, assessment_date, completion_date, template_id,
		    assessment_type_description, raw_json)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
		    patient_id=excluded.patient_id, org_id=excluded.org_id,
		    pcc_assessment_id=excluded.pcc_assessment_id,
		    assessment_type=excluded.assessment_type, status=excluded.status,
		    assessment_date=excluded.assessment_date,
		    completion_date=excluded.completion_date, template_id=excluded.template_id,
		    assessment_type_description=excluded.assessment_type_description,
		    raw_json=excluded.raw_json`,
		a.ID, a.PatientID, a.OrgID, a.PCCAssessID, a.AssessType, a.Status,
		a.AssessmentDate, a.CompletionDate, a.TemplateID, a.AssessTypeDesc, a.RawJSON)
	return err
}

// UpsertEligibility inserts or replaces the derived eligibility row.
func (s *Store) UpsertEligibility(r models.EligibilityRow) error {
	w := r.Wound
	_, err := s.db.Exec(`
		INSERT INTO eligibility (patient_id, internal_id, facility_id, has_active_mcb,
		    wound_type, stage, location, length_cm, width_cm, depth_cm,
		    drainage_amount, has_secondary_wound, confidence, extraction_source,
		    decision, reason)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(patient_id) DO UPDATE SET
		    internal_id=excluded.internal_id, facility_id=excluded.facility_id,
		    has_active_mcb=excluded.has_active_mcb, wound_type=excluded.wound_type,
		    stage=excluded.stage, location=excluded.location, length_cm=excluded.length_cm,
		    width_cm=excluded.width_cm, depth_cm=excluded.depth_cm,
		    drainage_amount=excluded.drainage_amount,
		    has_secondary_wound=excluded.has_secondary_wound,
		    confidence=excluded.confidence, extraction_source=excluded.extraction_source,
		    decision=excluded.decision, reason=excluded.reason`,
		r.PatientID, r.InternalID, r.FacilityID, boolToInt(r.HasActiveMCB),
		nullIfEmpty(w.WoundType), w.Stage, nullIfEmpty(w.Location), w.LengthCM,
		w.WidthCM, w.DepthCM, nullIfEmpty(w.DrainageAmount), boolToInt(w.HasSecondary),
		w.Confidence, nullIfEmpty(w.Source), r.Decision, r.Reason)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
