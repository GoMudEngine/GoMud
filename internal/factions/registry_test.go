package factions

import (
	"os"
	"path/filepath"
	"testing"
)

// helper: write a definition YAML into a temp dir and return the dir path.
func writeFactionDef(t *testing.T, dir, slug, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, slug+".yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAllDefinitions_LoadsValidYAML(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", dir)

	writeFactionDef(t, dir, "warren", `
faction_id: warren
display_name: "The Warren"
description: "A coalition of warren scouts and warriors."
default_rep: -25
allies: []
enemies: []
`)
	writeFactionDef(t, dir, "thornwall_guards", `
faction_id: thornwall_guards
display_name: "Thornwall Guards"
description: "City watch."
default_rep: 0
allies: []
enemies: [warren]
`)

	clearRegistryForTest()
	if err := LoadAllDefinitions(); err != nil {
		t.Fatalf("LoadAllDefinitions: %v", err)
	}

	if d := GetDefinition("warren"); d == nil || d.DisplayName != "The Warren" {
		t.Errorf("warren definition not loaded: %+v", d)
	}
	if d := GetDefinition("thornwall_guards"); d == nil || d.DefaultRep != 0 {
		t.Errorf("thornwall_guards definition not loaded: %+v", d)
	}
	if all := AllDefinitions(); len(all) != 2 {
		t.Errorf("AllDefinitions returned %d, want 2", len(all))
	}
}

func TestLoadAllDefinitions_PanicsOnUnknownAllyReference(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", dir)

	writeFactionDef(t, dir, "warren", `
faction_id: warren
display_name: "Warren"
description: "x"
default_rep: 0
allies: [nonexistent_faction]
enemies: []
`)

	clearRegistryForTest()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on unknown ally reference, got none")
		}
	}()
	_ = LoadAllDefinitions()
}

func TestLoadAllDefinitions_PanicsOnUnknownEnemyReference(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", dir)

	writeFactionDef(t, dir, "guards", `
faction_id: guards
display_name: "Guards"
description: "x"
default_rep: 0
allies: []
enemies: [missing_faction]
`)

	clearRegistryForTest()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on unknown enemy reference, got none")
		}
	}()
	_ = LoadAllDefinitions()
}

func TestLoadAllDefinitions_MissingDirReturnsCleanly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", dir)
	clearRegistryForTest()
	if err := LoadAllDefinitions(); err != nil {
		t.Errorf("missing dir should be clean: %v", err)
	}
	if all := AllDefinitions(); len(all) != 0 {
		t.Errorf("AllDefinitions = %d, want 0", len(all))
	}
}

func TestValidateHoldingCells(t *testing.T) {
	clearRegistryForTest()
	definitionsMu.Lock()
	definitions = map[string]*Definition{
		"good_guards": {FactionId: "good_guards", HoldingCellRoom: 100},
		"no_cell":     {FactionId: "no_cell", HoldingCellRoom: 0},
	}
	definitionsMu.Unlock()

	// Room 100 exists -> no panic.
	ValidateHoldingCells(func(roomId int) bool { return roomId == 100 })

	// Room 100 does NOT exist -> panic.
	defer func() {
		if recover() == nil {
			t.Fatalf("ValidateHoldingCells must panic on a dangling cell room")
		}
	}()
	ValidateHoldingCells(func(roomId int) bool { return false })
}
