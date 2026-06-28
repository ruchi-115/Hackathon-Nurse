package extract

import (
	"context"

	"github.com/Matrix030/hackathon-nurse/internal/models"
)

// LLMExtractor is the optional LLM fallback for unstructured notes. It is nil
// when no API key is configured, in which case ambiguous notes are left for the
// router to flag.
type LLMExtractor interface {
	Extract(ctx context.Context, noteText string) (models.WoundFields, bool)
}

// Extract chooses the best wound extraction for a patient using the precedence:
//
//	structured assessment  >  labeled/shorthand note (rules)  >  LLM fallback
//
// It returns the chosen WoundFields. The Source field records which path won.
// If nothing yields a wound, Source is "none".
func Extract(ctx context.Context, assessments []models.Assessment, notes []models.Note, llm LLMExtractor) models.WoundFields {
	// 1. Structured assessments (highest confidence). Prefer complete ones.
	var best models.WoundFields
	found := false
	for _, a := range assessments {
		if w, ok := FromAssessment(a); ok {
			if !found || w.Confidence > best.Confidence {
				best, found = w, true
			}
		}
	}
	if found && best.Complete() {
		return best
	}

	// 2. Labeled / shorthand notes (rules). May improve on a partial assessment.
	var bestNote models.WoundFields
	noteFound := false
	var llmCandidate *models.Note
	for i := range notes {
		if w, ok := FromNote(notes[i]); ok {
			if !noteFound || w.Confidence > bestNote.Confidence {
				bestNote, noteFound = w, true
			}
		} else if notes[i].NoteText != nil && *notes[i].NoteText != "" {
			// Unparseable narrative — candidate for the LLM fallback.
			llmCandidate = &notes[i]
		}
	}
	if noteFound && (!found || bestNote.Confidence > best.Confidence) {
		best, found = bestNote, true
	}
	if found && best.Complete() {
		return best
	}

	// 3. LLM fallback for unstructured prose (last resort, only if configured).
	if llm != nil && llmCandidate != nil {
		if w, ok := llm.Extract(ctx, *llmCandidate.NoteText); ok {
			if !found || w.Confidence > best.Confidence {
				best, found = w, true
			}
		}
	}

	if !found {
		return models.WoundFields{Source: "none"}
	}
	return best
}
