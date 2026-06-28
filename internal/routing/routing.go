// Package routing turns a patient's coverage, diagnoses, and extracted wound
// fields into a billing decision (auto_accept / flag_for_review / reject) with a
// plain-English reason a biller can act on.
package routing

import (
	"fmt"
	"strings"

	"github.com/Matrix030/hackathon-nurse/internal/models"
)

// woundICDPrefixes are ICD-10 prefixes that indicate an active wound.
var woundICDPrefixes = []string{
	"L89", // pressure ulcer
	"L97", // non-pressure chronic ulcer of lower limb
	"L98", // other ulcers of skin
	"E11.621", "E10.621", // diabetic foot ulcer
	"I83.0", "I83.2", // venous ulcer
	"I70.23", "I70.24", "I70.25", // arterial ulcer
	"T81.4", // surgical site infection
	"L02",   // abscess
	"T20", "T21", "T22", "T23", "T24", "T25", // burns
}

// HasActiveMCB reports whether the patient has active Medicare Part B coverage
// (payer type "Medicare B" / code "MCB" with no end date).
func HasActiveMCB(coverage []models.Coverage) bool {
	for _, c := range coverage {
		typeMCB := c.PayerType != nil && strings.EqualFold(strings.TrimSpace(*c.PayerType), "Medicare B")
		codeMCB := c.PayerCode != nil && strings.EqualFold(strings.TrimSpace(*c.PayerCode), "MCB")
		active := c.EffectiveTo == nil || strings.TrimSpace(*c.EffectiveTo) == ""
		if (typeMCB || codeMCB) && active {
			return true
		}
	}
	return false
}

// HasActiveWoundDx reports whether any active diagnosis matches a wound ICD-10 prefix.
func HasActiveWoundDx(diagnoses []models.Diagnosis) bool {
	for _, d := range diagnoses {
		if d.ClinicalStatus != nil && !strings.EqualFold(*d.ClinicalStatus, "active") {
			continue
		}
		if d.ICD10Code == nil {
			continue
		}
		code := strings.ToUpper(strings.TrimSpace(*d.ICD10Code))
		for _, p := range woundICDPrefixes {
			if strings.HasPrefix(code, strings.ToUpper(p)) {
				return true
			}
		}
	}
	return false
}

// Decide produces the routing decision and reason.
//
//	reject          — no active wound at all (no dx and no extractable wound)
//	auto_accept     — active wound + active MCB + all required fields from a
//	                  high-confidence source
//	flag_for_review — everything in between (missing data, ambiguous source,
//	                  low confidence, multi-wound, or non-MCB payer)
func Decide(w models.WoundFields, mcb bool, hasWoundDx bool) (decision, reason string) {
	hasWound := hasWoundDx || w.Source != "none" && w.Source != ""

	if !hasWound {
		return models.DecisionReject,
			"No active wound diagnosis and no wound documented in notes or assessments — not eligible for wound care billing."
	}

	// Reliable, complete extraction from a trustworthy source?
	highConfidence := (w.Source == "assessment" || w.Source == "note_rules") && w.Confidence >= 0.75
	complete := w.Complete()

	if mcb && complete && highConfidence {
		d := models.DecisionAutoAccept
		r := fmt.Sprintf("Active Medicare Part B coverage and a fully documented %s at %s (%s) with %s drainage from %s — safe to route to billing.",
			humanWoundType(w.WoundType), w.Location, dims(w), w.DrainageAmount, humanSource(w.Source))
		if w.HasSecondary {
			// Multi-wound: still acceptable but call it out.
			r += " Note: a secondary wound was documented; primary wound used."
		}
		return d, r
	}

	// Build a flag/reject reason from what's missing.
	var missing []string
	if w.WoundType == "" {
		missing = append(missing, "wound type")
	}
	if w.Location == "" {
		missing = append(missing, "location")
	}
	if !w.HasMeasurements() {
		missing = append(missing, "measurements (L/W/D)")
	}
	if w.DrainageAmount == "" {
		missing = append(missing, "drainage amount")
	}

	if !mcb {
		return models.DecisionFlag,
			"Wound documented but patient does not have active Medicare Part B coverage — biller should confirm payer before billing."
	}

	if w.Source == "llm" {
		return models.DecisionFlag,
			fmt.Sprintf("Wound details extracted from an unstructured narrative note by LLM (confidence %.0f%%) — clinician should verify before billing.", w.Confidence*100)
	}

	if len(missing) > 0 {
		return models.DecisionFlag,
			fmt.Sprintf("Active Medicare Part B coverage, but documentation is incomplete — missing %s. A clinician or biller should review.", strings.Join(missing, ", "))
	}

	if w.HasSecondary {
		return models.DecisionFlag,
			"Multiple wounds documented and the primary wound is ambiguous — needs human review to pick the billable wound."
	}

	return models.DecisionFlag,
		"Wound documented with active Medicare Part B coverage, but extraction confidence is low — review recommended."
}

func dims(w models.WoundFields) string {
	if !w.HasMeasurements() {
		return "dimensions n/a"
	}
	return fmt.Sprintf("%.1f×%.1f×%.1f cm", *w.LengthCM, *w.WidthCM, *w.DepthCM)
}

func humanWoundType(s string) string {
	if s == "" {
		return "wound"
	}
	return strings.ReplaceAll(s, "_", " ")
}

func humanSource(s string) string {
	switch s {
	case "assessment":
		return "a structured assessment"
	case "note_rules":
		return "a structured progress note"
	case "llm":
		return "an LLM-parsed narrative note"
	}
	return "the record"
}
