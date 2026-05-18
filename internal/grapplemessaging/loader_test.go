package grapplemessaging

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test_grapple_outcomes.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp yaml: %v", err)
	}
	return path
}

func TestLoaderEmptySkeleton(t *testing.T) {
	path := writeTempYAML(t, `
advancements: {}
degradations: {}
reversals: {}
escapes: {}
holds: {}
striking_apex: {}
`)
	lib, err := Load(path)
	if err != nil {
		t.Fatalf("Load skeleton: %v", err)
	}
	if lib == nil {
		t.Fatal("Load returned nil library")
	}
	if len(lib.Advancements) != 0 {
		t.Errorf("expected empty advancements, got %d", len(lib.Advancements))
	}
}

func TestLoaderTriadParse(t *testing.T) {
	path := writeTempYAML(t, `
advancements:
  clinch_to_mount:
    controller:
      - "You drive forward and ride them into mount."
    controlled:
      - "{controllerName} drives forward and mounts you."
    observers:
      - "{controllerName} drives forward and mounts {controlledName}."
degradations: {}
reversals: {}
escapes: {}
holds: {}
striking_apex: {}
`)
	lib, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	triad, ok := lib.Advancements["clinch_to_mount"]
	if !ok {
		t.Fatal("missing clinch_to_mount key")
	}
	if len(triad.Controller) != 1 || len(triad.Controlled) != 1 || len(triad.Observers) != 1 {
		t.Errorf("triad lengths: ctrl=%d cd=%d obs=%d, want 1/1/1",
			len(triad.Controller), len(triad.Controlled), len(triad.Observers))
	}
}

func TestLoaderMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/grapple_outcomes.yaml")
	if err == nil {
		t.Error("Load should error on missing file")
	}
}

func TestLoaderInvalidYAML(t *testing.T) {
	path := writeTempYAML(t, `this is not: valid: yaml: at all: ::`)
	_, err := Load(path)
	if err == nil {
		t.Error("Load should error on invalid YAML")
	}
}

func TestStrikingApexParse(t *testing.T) {
	path := writeTempYAML(t, `
advancements: {}
degradations: {}
reversals: {}
escapes: {}
holds: {}
striking_apex:
  mount_strike_flavor:
    - "You ride high in mount and rain elbows down."
    - "Their arms tire from defending; you sit heavy."
`)
	lib, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	apex, ok := lib.StrikingApex["mount_strike_flavor"]
	if !ok {
		t.Fatal("missing mount_strike_flavor key")
	}
	if len(apex) != 2 {
		t.Errorf("expected 2 strike flavor lines, got %d", len(apex))
	}
}

func TestValidateCompletenessEmpty(t *testing.T) {
	lib := &Library{
		Advancements: map[string]TemplateTriad{},
		Degradations: map[string]TemplateTriad{},
		Reversals:    map[string]TemplateTriad{},
		Escapes:      map[string]TemplateTriad{},
		Holds:        map[string]TemplateTriad{},
		StrikingApex: map[string][]string{},
	}
	errs := ValidateCompleteness(lib)
	if len(errs) == 0 {
		t.Error("empty library should produce completeness errors")
	}
}

func TestValidateCompletenessMissingTriadSpeaker(t *testing.T) {
	lib := &Library{
		Advancements: map[string]TemplateTriad{
			"clinch_to_mount": {
				Controller: []string{"a", "b", "c"},
				Controlled: []string{"a", "b", "c"},
				// Observers missing — should error
			},
		},
		Degradations: map[string]TemplateTriad{},
		Reversals:    map[string]TemplateTriad{},
		Escapes:      map[string]TemplateTriad{},
		Holds:        map[string]TemplateTriad{},
		StrikingApex: map[string][]string{},
	}
	errs := ValidateCompleteness(lib)
	if len(errs) == 0 {
		t.Error("triad with empty Observers should fail validation")
	}
}

func TestValidateCompletenessUnderMinCount(t *testing.T) {
	lib := &Library{
		Advancements: map[string]TemplateTriad{
			"clinch_to_mount": {
				Controller: []string{"only one"},
				Controlled: []string{"only one"},
				Observers:  []string{"only one"},
			},
		},
		Degradations: map[string]TemplateTriad{},
		Reversals:    map[string]TemplateTriad{},
		Escapes:      map[string]TemplateTriad{},
		Holds:        map[string]TemplateTriad{},
		StrikingApex: map[string][]string{},
	}
	errs := ValidateCompleteness(lib)
	if len(errs) == 0 {
		t.Error("triad with only 1 template per speaker should fail (minimum 3)")
	}
}

func TestProductionLibraryAdvancementsComplete(t *testing.T) {
	lib, err := Load("../../_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
	if err != nil {
		t.Fatalf("Load prod library: %v", err)
	}
	for _, key := range RequiredAdvancementKeys {
		triad, ok := lib.Advancements[key]
		if !ok {
			t.Errorf("missing advancement key: %s", key)
			continue
		}
		if len(triad.Controller) < MinTemplatesPerSpeaker {
			t.Errorf("%s.controller: %d < %d", key, len(triad.Controller), MinTemplatesPerSpeaker)
		}
		if len(triad.Controlled) < MinTemplatesPerSpeaker {
			t.Errorf("%s.controlled: %d < %d", key, len(triad.Controlled), MinTemplatesPerSpeaker)
		}
		if len(triad.Observers) < MinTemplatesPerSpeaker {
			t.Errorf("%s.observers: %d < %d", key, len(triad.Observers), MinTemplatesPerSpeaker)
		}
	}
}

func TestProductionLibraryDegradationsComplete(t *testing.T) {
	lib, err := Load("../../_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
	if err != nil {
		t.Fatalf("Load prod library: %v", err)
	}
	for _, key := range RequiredDegradationKeys {
		triad, ok := lib.Degradations[key]
		if !ok {
			t.Errorf("missing degradation key: %s", key)
			continue
		}
		if len(triad.Controller) < MinTemplatesPerSpeaker {
			t.Errorf("%s.controller: %d < %d", key, len(triad.Controller), MinTemplatesPerSpeaker)
		}
		if len(triad.Controlled) < MinTemplatesPerSpeaker {
			t.Errorf("%s.controlled: %d < %d", key, len(triad.Controlled), MinTemplatesPerSpeaker)
		}
		if len(triad.Observers) < MinTemplatesPerSpeaker {
			t.Errorf("%s.observers: %d < %d", key, len(triad.Observers), MinTemplatesPerSpeaker)
		}
	}
}

func TestProductionLibraryReversalsComplete(t *testing.T) {
	lib, err := Load("../../_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
	if err != nil {
		t.Fatalf("Load prod library: %v", err)
	}
	for _, key := range RequiredReversalKeys {
		triad, ok := lib.Reversals[key]
		if !ok {
			t.Errorf("missing reversal key: %s", key)
			continue
		}
		if len(triad.Controller) < MinTemplatesPerSpeaker {
			t.Errorf("%s.controller: %d < %d", key, len(triad.Controller), MinTemplatesPerSpeaker)
		}
		if len(triad.Controlled) < MinTemplatesPerSpeaker {
			t.Errorf("%s.controlled: %d < %d", key, len(triad.Controlled), MinTemplatesPerSpeaker)
		}
		if len(triad.Observers) < MinTemplatesPerSpeaker {
			t.Errorf("%s.observers: %d < %d", key, len(triad.Observers), MinTemplatesPerSpeaker)
		}
	}
}

func TestProductionLibraryEscapesComplete(t *testing.T) {
	lib, err := Load("../../_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
	if err != nil {
		t.Fatalf("Load prod library: %v", err)
	}
	for _, key := range RequiredEscapeKeys {
		triad, ok := lib.Escapes[key]
		if !ok {
			t.Errorf("missing escape key: %s", key)
			continue
		}
		if len(triad.Controller) < MinTemplatesPerSpeaker ||
			len(triad.Controlled) < MinTemplatesPerSpeaker ||
			len(triad.Observers) < MinTemplatesPerSpeaker {
			t.Errorf("escape %s under-populated", key)
		}
	}
}

func TestProductionLibraryHoldsComplete(t *testing.T) {
	lib, err := Load("../../_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
	if err != nil {
		t.Fatalf("Load prod library: %v", err)
	}
	for _, key := range RequiredHoldKeys {
		triad, ok := lib.Holds[key]
		if !ok {
			t.Errorf("missing hold key: %s", key)
			continue
		}
		if len(triad.Controller) < MinTemplatesPerSpeaker ||
			len(triad.Controlled) < MinTemplatesPerSpeaker ||
			len(triad.Observers) < MinTemplatesPerSpeaker {
			t.Errorf("hold %s under-populated", key)
		}
	}
}

func TestProductionLibraryStrikingApexComplete(t *testing.T) {
	lib, err := Load("../../_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
	if err != nil {
		t.Fatalf("Load prod library: %v", err)
	}
	for _, key := range RequiredStrikingApexKeys {
		templates, ok := lib.StrikingApex[key]
		if !ok {
			t.Errorf("missing striking_apex key: %s", key)
			continue
		}
		if len(templates) < MinStrikingApexTemplates {
			t.Errorf("striking_apex.%s: %d < %d", key, len(templates), MinStrikingApexTemplates)
		}
	}
}

func TestProductionLibraryFullValidation(t *testing.T) {
	lib, err := Load("../../_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
	if err != nil {
		t.Fatalf("Load prod library: %v", err)
	}
	errs := ValidateCompleteness(lib)
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("completeness: %v", e)
		}
	}
}
