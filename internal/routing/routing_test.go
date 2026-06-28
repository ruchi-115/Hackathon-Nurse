package routing

import (
	"testing"

	"github.com/Matrix030/hackathon-nurse/internal/models"
)

func f(v float64) *float64 { return &v }
func sp(s string) *string  { return &s }

func completeWound(source string) models.WoundFields {
	return models.WoundFields{
		WoundType:      "pressure_ulcer",
		Location:       "Sacrum",
		LengthCM:       f(3.2), WidthCM: f(2.1), DepthCM: f(0.4),
		DrainageAmount: models.DrainageModerate,
		Confidence:     1.0,
		Source:         source,
	}
}

func TestDecideAutoAccept(t *testing.T) {
	d, reason := Decide(completeWound("assessment"), true, true)
	if d != models.DecisionAutoAccept {
		t.Fatalf("want auto_accept, got %s (%s)", d, reason)
	}
}

func TestDecideRejectNoWound(t *testing.T) {
	d, _ := Decide(models.WoundFields{Source: "none"}, true, false)
	if d != models.DecisionReject {
		t.Fatalf("want reject, got %s", d)
	}
}

func TestDecideFlagNoMCB(t *testing.T) {
	d, _ := Decide(completeWound("assessment"), false, true)
	if d != models.DecisionFlag {
		t.Fatalf("want flag (no MCB), got %s", d)
	}
}

func TestDecideFlagIncomplete(t *testing.T) {
	w := completeWound("assessment")
	w.DepthCM = nil // missing a measurement
	w.Confidence = 0.5
	d, reason := Decide(w, true, true)
	if d != models.DecisionFlag {
		t.Fatalf("want flag (incomplete), got %s (%s)", d, reason)
	}
}

func TestDecideFlagLLMSource(t *testing.T) {
	w := completeWound("llm")
	w.Confidence = 0.9
	d, _ := Decide(w, true, true)
	if d != models.DecisionFlag {
		t.Fatalf("want flag (LLM source), got %s", d)
	}
}

func TestHasActiveMCB(t *testing.T) {
	cov := []models.Coverage{
		{PayerType: sp("Medicare B"), PayerCode: sp("MCB"), EffectiveTo: nil},
	}
	if !HasActiveMCB(cov) {
		t.Fatal("expected active MCB")
	}
	expired := []models.Coverage{
		{PayerType: sp("Medicare B"), EffectiveTo: sp("2025-01-01T00:00:00")},
	}
	if HasActiveMCB(expired) {
		t.Fatal("expected expired MCB to be inactive")
	}
	other := []models.Coverage{{PayerType: sp("HMO"), EffectiveTo: nil}}
	if HasActiveMCB(other) {
		t.Fatal("expected HMO not to count as MCB")
	}
}

func TestHasActiveWoundDx(t *testing.T) {
	dx := []models.Diagnosis{
		{ICD10Code: sp("L89.152"), ClinicalStatus: sp("active")},
	}
	if !HasActiveWoundDx(dx) {
		t.Fatal("expected wound dx (L89 pressure ulcer)")
	}
	resolved := []models.Diagnosis{
		{ICD10Code: sp("L89.152"), ClinicalStatus: sp("resolved")},
	}
	if HasActiveWoundDx(resolved) {
		t.Fatal("expected resolved dx not to count")
	}
	nonWound := []models.Diagnosis{
		{ICD10Code: sp("E11.9"), ClinicalStatus: sp("active")},
	}
	if HasActiveWoundDx(nonWound) {
		t.Fatal("expected non-wound dx not to count")
	}
}
