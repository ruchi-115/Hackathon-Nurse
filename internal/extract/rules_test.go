package extract

import (
	"testing"

	"github.com/Matrix030/hackathon-nurse/internal/models"
)

func strptr(s string) *string { return &s }

func TestFromAssessmentStructured(t *testing.T) {
	// Real nested question/answer shape from the live API.
	raw := `{"sections":[
		{"sectionName":"LOCATION","questions":[{"question":"Location","answer":"buttock"},{"question":"Laterality","answer":"Left"}]},
		{"sectionName":"WOUND","questions":[{"question":"Wound Type","answer":"Pressure Ulcer"},{"question":"Stage","answer":"N/A"},{"question":"Length (cm)","answer":"5.9"},{"question":"Width (cm)","answer":"4.5"},{"question":"Depth (cm)","answer":"1.8"}]},
		{"sectionName":"DRAINAGE","questions":[{"question":"Drainage Present","answer":"Yes"},{"question":"Drainage Type","answer":"Serosanguineous"},{"question":"Drainage Amount","answer":"Heavy"}]}
	]}`
	w, ok := FromAssessment(models.Assessment{RawJSON: &raw})
	if !ok {
		t.Fatal("expected ok")
	}
	if w.WoundType != "pressure_ulcer" || w.Location != "Left buttock" {
		t.Fatalf("type/location wrong: %+v", w)
	}
	if w.LengthCM == nil || *w.LengthCM != 5.9 || w.DepthCM == nil || *w.DepthCM != 1.8 {
		t.Fatalf("measurements wrong: %+v", w)
	}
	if w.DrainageAmount != models.DrainageHeavy {
		t.Fatalf("drainage wrong: %q", w.DrainageAmount)
	}
	if w.Source != SourceAssessment || !w.Complete() {
		t.Fatalf("expected complete assessment source, got %+v", w)
	}
}

func TestFromAssessmentNarrative(t *testing.T) {
	// Envive narrative variant → parsed as prose, low-confidence source.
	raw := `{"sections":[{"sectionName":"WOUND_INFO","questions":[{"question":"Wound narrative","answer":"Pressure Ulcer to Right hip / Measures 2.9 cm x 2.8 cm / Stage: Stage 3 / Drainage: serosanguineous, heavy"}]}]}`
	w, ok := FromAssessment(models.Assessment{RawJSON: &raw})
	if !ok {
		t.Fatal("expected ok")
	}
	if w.Source != SourceNarrative {
		t.Fatalf("expected narrative source, got %q", w.Source)
	}
	if w.WoundType != "pressure_ulcer" || w.Location != "Right hip" {
		t.Fatalf("narrative type/location wrong: %+v", w)
	}
	if w.LengthCM == nil || *w.LengthCM != 2.9 {
		t.Fatalf("narrative length wrong: %+v", w)
	}
	if w.DrainageAmount != models.DrainageHeavy {
		t.Fatalf("narrative drainage wrong: %q", w.DrainageAmount)
	}
}

func TestFromNoteSPN(t *testing.T) {
	text := "Wound Assessment Note\nLocation: Sacrum\nWound Type: Pressure Ulcer, Stage 2\nLength: 3.2 cm  Width: 2.1 cm  Depth: 0.4 cm\nDrainage: Moderate serosanguineous\nPeriwound: Intact skin"
	w, ok := FromNote(models.Note{NoteText: &text})
	if !ok {
		t.Fatal("expected ok")
	}
	if w.WoundType != "pressure_ulcer" {
		t.Fatalf("type wrong: %q", w.WoundType)
	}
	if w.Location != "Sacrum" {
		t.Fatalf("location wrong: %q", w.Location)
	}
	if !w.HasMeasurements() || *w.WidthCM != 2.1 {
		t.Fatalf("measurements wrong: %+v", w)
	}
	if w.DrainageAmount != models.DrainageModerate {
		t.Fatalf("drainage wrong: %q", w.DrainageAmount)
	}
}

func TestFromNoteShorthand(t *testing.T) {
	text := "DFU right heel. Meas 4.2x3.1x1.5cm. Scant drainage."
	w, ok := FromNote(models.Note{NoteText: &text})
	if !ok {
		t.Fatal("expected ok")
	}
	if w.WoundType != "diabetic_foot_ulcer" {
		t.Fatalf("type wrong: %q", w.WoundType)
	}
	if w.LengthCM == nil || *w.LengthCM != 4.2 || *w.WidthCM != 3.1 || *w.DepthCM != 1.5 {
		t.Fatalf("shorthand measurements wrong: %+v", w)
	}
	if w.DrainageAmount != models.DrainageLight {
		t.Fatalf("drainage wrong: %q", w.DrainageAmount)
	}
}

func TestFromNoteMultiWound(t *testing.T) {
	text := "Two wounds noted. Wound 1: sacral pressure ulcer 3x2x1cm. Wound 2: heel ulcer. Moderate drainage."
	w, ok := FromNote(models.Note{NoteText: &text})
	if !ok {
		t.Fatal("expected ok")
	}
	if !w.HasSecondary {
		t.Fatal("expected HasSecondary=true for multi-wound note")
	}
}

func TestFromNoteEnviveNarrativeUnparseable(t *testing.T) {
	// Pure narrative with no measurements / labels / keywords → not parseable by rules.
	text := "Patient resting comfortably. Skin integrity reviewed during rounds; dressing intact and clean."
	if _, ok := FromNote(models.Note{NoteText: &text}); ok {
		t.Fatal("expected narrative note to be unparseable by rules")
	}
}
