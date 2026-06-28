package extract

import (
	"context"

	"github.com/Matrix030/hackathon-nurse/internal/config"
	"github.com/Matrix030/hackathon-nurse/internal/models"
)

// NewLLM returns the LLM-backed extractor used as the last-resort fallback for
// unstructured ("Envive") narrative notes the rules can't parse. It returns nil
// when no API key is configured, so the pipeline runs rules-only and the router
// flags those notes for review.
//
// This is currently a STUB: the wiring (interface, precedence in Extract,
// routing for the "llm" source) is fully in place, but the actual model call is
// not implemented yet — see stubLLM.Extract.
func NewLLM(cfg config.Config) LLMExtractor {
	if cfg.AnthropicKey == "" {
		return nil
	}
	return &stubLLM{model: cfg.AnthropicModel}
}

// stubLLM is a placeholder LLMExtractor. Swap the body of Extract for a real
// Anthropic structured-output call when ready.
type stubLLM struct {
	model string
}

// Extract is where the LLM fallback would parse an unstructured note into
// WoundFields. Right now it does nothing and reports ok=false, so narrative
// notes continue to route to flag_for_review.
func (s *stubLLM) Extract(ctx context.Context, noteText string) (models.WoundFields, bool) {
	// ─────────────────────────────────────────────────────────────────────
	// LLM RESPONSE GOES HERE
	//
	// Send `noteText` to the model (structured output → models.WoundFields),
	// then return the parsed fields with Source = SourceLLM and a Confidence
	// from the model. For example:
	//
	//   w := callModel(ctx, s.model, noteText) // -> models.WoundFields
	//   w.Source = SourceLLM
	//   return w, true
	// ─────────────────────────────────────────────────────────────────────
	_ = noteText
	return models.WoundFields{}, false
}
