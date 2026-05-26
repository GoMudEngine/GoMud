package goals

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func resetRegistry() {
	registryMu.Lock()
	typeRegistry = map[string]GoalTypeMeta{}
	registryMu.Unlock()
}

func TestRegisterGoalType_LookupRoundTrip(t *testing.T) {
	resetRegistry()
	pred := func(g *Goal, m *mobs.Mob) bool { return true }
	RegisterGoalType("revenge", GoalTypeMeta{
		Predicate:     pred,
		ConflictsWith: []string{"protection"},
	})
	got, ok := lookupMeta("revenge")
	if !ok {
		t.Fatal("expected registered type to be found")
	}
	if got.Predicate == nil {
		t.Error("predicate not preserved")
	}
	if len(got.ConflictsWith) != 1 || got.ConflictsWith[0] != "protection" {
		t.Errorf("ConflictsWith = %v, want [protection]", got.ConflictsWith)
	}
}

func TestLookupMeta_UnknownTypeReturnsFalse(t *testing.T) {
	resetRegistry()
	if _, ok := lookupMeta("noexist"); ok {
		t.Error("expected ok=false for unknown type")
	}
}

func TestValidateSymmetry_NoWarningWhenBothDeclared(t *testing.T) {
	resetRegistry()
	RegisterGoalType("revenge", GoalTypeMeta{ConflictsWith: []string{"protection"}})
	RegisterGoalType("protection", GoalTypeMeta{ConflictsWith: []string{"revenge"}})
	warnings := ValidateSymmetry()
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %v", warnings)
	}
}

func TestValidateSymmetry_WarnsOnOneSidedDeclaration(t *testing.T) {
	resetRegistry()
	RegisterGoalType("revenge", GoalTypeMeta{ConflictsWith: []string{"protection"}})
	RegisterGoalType("protection", GoalTypeMeta{ConflictsWith: nil})
	warnings := ValidateSymmetry()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	// Warning should name both types so an author can locate the gap.
	w := warnings[0]
	if !contains(w, "revenge") || !contains(w, "protection") {
		t.Errorf("warning missing type names: %q", w)
	}
}

func TestRegisterGoalType_ReregisterOverwrites(t *testing.T) {
	resetRegistry()
	RegisterGoalType("revenge", GoalTypeMeta{ConflictsWith: []string{"a"}})
	RegisterGoalType("revenge", GoalTypeMeta{ConflictsWith: []string{"b"}})
	got, _ := lookupMeta("revenge")
	if len(got.ConflictsWith) != 1 || got.ConflictsWith[0] != "b" {
		t.Errorf("reregister did not overwrite: %v", got.ConflictsWith)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
