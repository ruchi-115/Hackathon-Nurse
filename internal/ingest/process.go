package ingest

import (
	"context"
	"fmt"
	"log"

	"github.com/Matrix030/hackathon-nurse/internal/extract"
	"github.com/Matrix030/hackathon-nurse/internal/models"
	"github.com/Matrix030/hackathon-nurse/internal/routing"
	"github.com/Matrix030/hackathon-nurse/internal/store"
)

// ProcessResult summarizes an extraction/routing pass.
type ProcessResult struct {
	Patients   int
	ByDecision map[string]int
	LLMUsed    int
}

// Process runs extraction + routing over every stored patient and writes the
// derived eligibility rows. llm may be nil (no LLM fallback configured).
func Process(ctx context.Context, st *store.Store, llm extract.LLMExtractor) (*ProcessResult, error) {
	patients, err := st.AllPatients()
	if err != nil {
		return nil, err
	}
	res := &ProcessResult{ByDecision: map[string]int{}}

	for _, p := range patients {
		diagnoses, err := st.DiagnosesFor(p.PatientID)
		if err != nil {
			return nil, fmt.Errorf("diagnoses %s: %w", p.PatientID, err)
		}
		coverage, err := st.CoverageFor(p.PatientID)
		if err != nil {
			return nil, fmt.Errorf("coverage %s: %w", p.PatientID, err)
		}
		notes, err := st.NotesFor(p.ID)
		if err != nil {
			return nil, fmt.Errorf("notes %d: %w", p.ID, err)
		}
		assessments, err := st.AssessmentsFor(p.ID)
		if err != nil {
			return nil, fmt.Errorf("assessments %d: %w", p.ID, err)
		}

		wound := extract.Extract(ctx, assessments, notes, llm)
		if wound.Source == "llm" {
			res.LLMUsed++
		}
		mcb := routing.HasActiveMCB(coverage)
		hasDx := routing.HasActiveWoundDx(diagnoses)
		decision, reason := routing.Decide(wound, mcb, hasDx)

		row := models.EligibilityRow{
			PatientID:    p.PatientID,
			InternalID:   p.ID,
			FacilityID:   p.FacilityID,
			FirstName:    p.FirstName,
			LastName:     p.LastName,
			PrimaryPayer: p.PrimaryPayer,
			HasActiveMCB: mcb,
			Wound:        wound,
			Decision:     decision,
			Reason:       reason,
		}
		if err := st.UpsertEligibility(row); err != nil {
			return nil, fmt.Errorf("store eligibility %s: %w", p.PatientID, err)
		}
		res.ByDecision[decision]++
		res.Patients++
	}

	log.Printf("processed %d patients: %v (LLM used on %d)", res.Patients, res.ByDecision, res.LLMUsed)
	return res, nil
}
