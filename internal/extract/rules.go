// Package extract turns assessments and progress notes into normalized
// WoundFields. rules.go is the deterministic path: it parses structured
// assessment question/answer JSON and labeled/prose note text without any LLM
// call.
//
// NOTE: the live API's assessment raw_json shape differs from API.md. It is a
// nested {sections:[{questions:[{question,answer}]}]} structure with two
// variants: clean question/answer pairs (high confidence) and a single free-text
// "Wound narrative" answer (the "Envive" format — parsed as prose, lower
// confidence, routed for review).
package extract

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/Matrix030/hackathon-nurse/internal/models"
)

// Extraction source labels (also drive routing confidence).
const (
	SourceAssessment = "assessment" // structured Q/A assessment — high confidence
	SourceNoteRules  = "note_rules" // labeled progress note — high confidence
	SourceNarrative  = "narrative"  // Envive / prose — low confidence, review/LLM
	SourceLLM        = "llm"
	SourceNone       = "none"
)

// rawAssessment matches the live nested assessment JSON.
type rawAssessment struct {
	Sections []struct {
		SectionName string `json:"sectionName"`
		Questions   []struct {
			Question string `json:"question"`
			Answer   string `json:"answer"`
		} `json:"questions"`
	} `json:"sections"`
}

// FromAssessment parses an assessment's raw_json. Structured Q/A assessments are
// the highest-confidence source; narrative assessments are parsed as prose. ok is
// false when raw_json yields nothing usable.
func FromAssessment(a models.Assessment) (models.WoundFields, bool) {
	if a.RawJSON == nil || strings.TrimSpace(*a.RawJSON) == "" {
		return models.WoundFields{}, false
	}
	var ra rawAssessment
	if err := json.Unmarshal([]byte(*a.RawJSON), &ra); err != nil {
		return models.WoundFields{}, false
	}

	// Flatten question→answer, capturing any narrative blob separately.
	answers := map[string]string{}
	var narrative string
	for _, s := range ra.Sections {
		for _, q := range s.Questions {
			key := strings.ToLower(strings.TrimSpace(q.Question))
			answers[key] = strings.TrimSpace(q.Answer)
			if strings.Contains(key, "narrative") {
				narrative = q.Answer
			}
		}
	}

	// Narrative-only ("Envive") assessment → parse as prose, low confidence.
	hasStructured := answers["wound type"] != "" || answers["length (cm)"] != ""
	if narrative != "" && !hasStructured {
		w := parseFields(narrative)
		w.Source = SourceNarrative
		w.Confidence = completenessScore(w, 0.7)
		if isEmpty(w) {
			return models.WoundFields{}, false
		}
		return w, true
	}

	// Structured question/answer assessment.
	w := models.WoundFields{
		WoundType:      normalizeWoundType(answers["wound type"]),
		Location:       buildLocation(answers["location"], answers["laterality"]),
		LengthCM:       parseFloat(answers["length (cm)"]),
		WidthCM:        parseFloat(answers["width (cm)"]),
		DepthCM:        parseFloat(answers["depth (cm)"]),
		DrainageAmount: drainageFromStructured(answers),
		Source:         SourceAssessment,
	}
	if st := normalizeStage(answers["stage"]); st != "" {
		w.Stage = &st
	}
	w.Confidence = completenessScore(w, 1.0)
	if isEmpty(w) {
		return models.WoundFields{}, false
	}
	return w, true
}

// FromNote parses a progress note. Notes with explicit labeled measurement fields
// are treated as structured (high confidence); everything else is prose
// ("narrative"). Returns ok=false when nothing parseable is found.
func FromNote(n models.Note) (models.WoundFields, bool) {
	if n.NoteText == nil {
		return models.WoundFields{}, false
	}
	text := *n.NoteText
	w := parseFields(text)
	if isEmpty(w) {
		return models.WoundFields{}, false
	}
	// Labeled measurement fields → trust as structured; otherwise prose.
	if reLength.MatchString(text) && reWidth.MatchString(text) && reDepth.MatchString(text) {
		w.Source = SourceNoteRules
		w.Confidence = completenessScore(w, 1.0)
	} else {
		w.Source = SourceNarrative
		w.Confidence = completenessScore(w, 0.7)
	}
	return w, true
}

var (
	reLength = regexp.MustCompile(`(?i)length\s*[:=]?\s*([0-9]+(?:\.[0-9]+)?)\s*cm`)
	reWidth  = regexp.MustCompile(`(?i)width\s*[:=]?\s*([0-9]+(?:\.[0-9]+)?)\s*cm`)
	reDepth  = regexp.MustCompile(`(?i)depth\s*[:=]?\s*([0-9]+(?:\.[0-9]+)?)\s*cm`)
	// 3-D shorthand "Meas 4.2x3.1x1.5cm" or "4.2 x 3.1 x 1.5 cm"
	reLWH = regexp.MustCompile(`(?i)([0-9]+(?:\.[0-9]+)?)\s*(?:cm)?\s*[x×]\s*([0-9]+(?:\.[0-9]+)?)\s*(?:cm)?\s*[x×]\s*([0-9]+(?:\.[0-9]+)?)\s*cm`)
	// 2-D "2.9 cm x 2.8 cm" or "5.9 x 4.5cm"
	reLW = regexp.MustCompile(`(?i)([0-9]+(?:\.[0-9]+)?)\s*(?:cm)?\s*[x×]\s*([0-9]+(?:\.[0-9]+)?)\s*cm`)
	// Depth phrased separately: "depth 1.8cm" / "1.8 cm deep" / "0.9cm deep"
	reDepthWordA = regexp.MustCompile(`(?i)depth\s*[:=]?\s*([0-9]+(?:\.[0-9]+)?)\s*cm`)
	reDepthWordB = regexp.MustCompile(`(?i)([0-9]+(?:\.[0-9]+)?)\s*cm\s*deep`)

	reLocationLbl = regexp.MustCompile(`(?i)location\s*[:=]\s*([A-Za-z][A-Za-z /\-]+)`)
	// "ulcer to Right hip", "wound on left heel", "Pressure Ulcer Left buttock"
	reLocationTo  = regexp.MustCompile(`(?i)(?:ulcer|wound|injury)\s+(?:to|on|of|at)\s+([A-Za-z][A-Za-z ]+?)(?:\s*/|\s+measures|\s+meas|\s+approx|\s+aprx|,|\.|$)`)
	reWoundTypeLbl = regexp.MustCompile(`(?i)wound\s*type\s*[:=]\s*([A-Za-z][A-Za-z ,\-]+)`)

	reStageNum    = regexp.MustCompile(`(?i)\bstage\s*([0-9IV]+)`)
	reUnstageable = regexp.MustCompile(`(?i)\bunstageable\b`)
	reDrainageLbl = regexp.MustCompile(`(?i)drainage\s*(?:amount)?\s*[:=]\s*([A-Za-z]+)`)
)

// parseFields runs the shared regex extraction over free text. The caller sets
// Source and Confidence.
func parseFields(text string) models.WoundFields {
	var w models.WoundFields

	// Measurements: labeled fields win; then 3-D shorthand; then 2-D + separate depth.
	if m := reLength.FindStringSubmatch(text); m != nil {
		w.LengthCM = parseFloatStr(m[1])
	}
	if m := reWidth.FindStringSubmatch(text); m != nil {
		w.WidthCM = parseFloatStr(m[1])
	}
	if m := reDepth.FindStringSubmatch(text); m != nil {
		w.DepthCM = parseFloatStr(m[1])
	}
	if !w.HasMeasurements() {
		if m := reLWH.FindStringSubmatch(text); m != nil {
			w.LengthCM = parseFloatStr(m[1])
			w.WidthCM = parseFloatStr(m[2])
			w.DepthCM = parseFloatStr(m[3])
		} else {
			if m := reLW.FindStringSubmatch(text); m != nil {
				if w.LengthCM == nil {
					w.LengthCM = parseFloatStr(m[1])
				}
				if w.WidthCM == nil {
					w.WidthCM = parseFloatStr(m[2])
				}
			}
			if w.DepthCM == nil {
				if m := reDepthWordA.FindStringSubmatch(text); m != nil {
					w.DepthCM = parseFloatStr(m[1])
				} else if m := reDepthWordB.FindStringSubmatch(text); m != nil {
					w.DepthCM = parseFloatStr(m[1])
				}
			}
		}
	}

	if m := reLocationLbl.FindStringSubmatch(text); m != nil {
		w.Location = cleanLocation(m[1])
	} else if m := reLocationTo.FindStringSubmatch(text); m != nil {
		w.Location = cleanLocation(m[1])
	}

	if m := reWoundTypeLbl.FindStringSubmatch(text); m != nil {
		w.WoundType = normalizeWoundType(m[1])
	} else {
		w.WoundType = detectWoundType(text)
	}

	if reUnstageable.MatchString(text) {
		s := "unstageable"
		w.Stage = &s
	} else if m := reStageNum.FindStringSubmatch(text); m != nil {
		s := "stage " + strings.ToLower(m[1])
		w.Stage = &s
	}

	// Labeled drainage may capture the drainage *type* (e.g. "serosanguineous")
	// rather than the amount; if it doesn't map to a level, scan the whole text.
	if m := reDrainageLbl.FindStringSubmatch(text); m != nil {
		w.DrainageAmount = normalizeDrainage(m[1])
	}
	if w.DrainageAmount == "" {
		w.DrainageAmount = detectDrainage(text)
	}

	w.HasSecondary = countWoundMentions(text) > 1
	return w
}

func isEmpty(w models.WoundFields) bool {
	return w.WoundType == "" && w.Location == "" && !w.HasMeasurements() && w.DrainageAmount == ""
}

func parseFloat(s string) *float64  { return parseFloatStr(s) }
func parseFloatStr(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &f
}

func buildLocation(loc, laterality string) string {
	loc = cleanLocation(loc)
	laterality = strings.TrimSpace(laterality)
	if loc == "" {
		return ""
	}
	// Avoid duplicating laterality if already present in the location.
	if laterality != "" && !strings.Contains(strings.ToLower(loc), strings.ToLower(laterality)) {
		return laterality + " " + loc
	}
	return loc
}

func cleanLocation(s string) string {
	s = strings.TrimSpace(s)
	if strings.EqualFold(s, "n/a") {
		return ""
	}
	return s
}

func normalizeStage(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "n/a") {
		return ""
	}
	lower := strings.ToLower(s)
	if strings.Contains(lower, "unstageable") {
		return "unstageable"
	}
	if m := reStageNum.FindStringSubmatch(s); m != nil {
		return "stage " + strings.ToLower(m[1])
	}
	return lower
}

// drainageFromStructured reads the structured drainage answers, respecting a
// "Drainage Present: No" answer as "none".
func drainageFromStructured(answers map[string]string) string {
	if amt := normalizeDrainage(answers["drainage amount"]); amt != "" {
		return amt
	}
	if present, ok := answers["drainage present"]; ok && strings.EqualFold(present, "no") {
		return models.DrainageNone
	}
	return ""
}

func normalizeWoundType(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case s == "" || s == "n/a":
		return ""
	case strings.Contains(s, "pressure"):
		return "pressure_ulcer"
	case strings.Contains(s, "diabetic"):
		return "diabetic_foot_ulcer"
	case strings.Contains(s, "venous"):
		return "venous_stasis_ulcer"
	case strings.Contains(s, "arterial"):
		return "arterial_ulcer"
	case strings.Contains(s, "surgical"):
		return "surgical_site_infection"
	case strings.Contains(s, "abscess"):
		return "abscess"
	case strings.Contains(s, "burn"):
		return "burn"
	}
	return strings.ReplaceAll(s, " ", "_")
}

// detectWoundType scans free text for a known wound type keyword or common
// clinical abbreviation (DFU, VLU, PU, SSI).
func detectWoundType(text string) string {
	lower := strings.ToLower(text)
	abbrev := map[string]string{
		"dfu": "diabetic_foot_ulcer",
		"vlu": "venous_stasis_ulcer",
		"ssi": "surgical_site_infection",
		"pu":  "pressure_ulcer",
	}
	for ab, canon := range abbrev {
		if regexp.MustCompile(`(?i)\b` + ab + `\b`).MatchString(lower) {
			return canon
		}
	}
	return normalizeWoundType(firstMatchKeyword(text, []string{
		"pressure ulcer", "pressure injury", "diabetic foot ulcer", "diabetic ulcer",
		"venous stasis ulcer", "venous ulcer", "arterial ulcer",
		"surgical site infection", "abscess", "burn",
	}))
}

func firstMatchKeyword(text string, keys []string) string {
	lower := strings.ToLower(text)
	for _, k := range keys {
		if strings.Contains(lower, k) {
			return k
		}
	}
	return ""
}

func normalizeDrainage(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case s == "" || s == "n/a":
		return ""
	case strings.Contains(s, "none") || strings.Contains(s, "dry") || strings.Contains(s, "nil") || strings.Contains(s, "no "):
		return models.DrainageNone
	case strings.Contains(s, "light") || strings.Contains(s, "scant") || strings.Contains(s, "minimal") || strings.Contains(s, "min") || strings.Contains(s, "small") || strings.Contains(s, "slight"):
		return models.DrainageLight
	case strings.Contains(s, "moderate") || strings.Contains(s, "mod"):
		return models.DrainageModerate
	case strings.Contains(s, "heavy") || strings.Contains(s, "large") || strings.Contains(s, "copious"):
		return models.DrainageHeavy
	}
	return ""
}

func detectDrainage(text string) string {
	return normalizeDrainage(firstMatchKeyword(text, []string{
		"no drainage", "none", "scant", "minimal", "slight", "light", "moderate", "heavy", "copious", "large amount", "min drainage",
	}))
}

func countWoundMentions(text string) int {
	lower := strings.ToLower(text)
	return strings.Count(lower, "wound") + strings.Count(lower, "ulcer")
}

func completenessScore(w models.WoundFields, max float64) float64 {
	have := 0
	total := 4
	if w.WoundType != "" {
		have++
	}
	if w.Location != "" {
		have++
	}
	if w.HasMeasurements() {
		have++
	}
	if w.DrainageAmount != "" {
		have++
	}
	return (float64(have) / float64(total)) * max
}
