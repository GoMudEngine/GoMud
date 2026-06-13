package hooks

import "testing"

// parseRecipeGrants backs multi-recipe quest rewards (e.g. a crafting quest
// grants both a blade and a buckler recipe). These tests pin the parse; the
// application logic lives in the handler itself.

func TestParseRecipeGrants_Empty(t *testing.T) {
	if g := parseRecipeGrants(""); g != nil {
		t.Errorf("empty recipe_info: want nil, got %v", g)
	}
}

func TestParseRecipeGrants_Single(t *testing.T) {
	g := parseRecipeGrants("iron-dagger")
	if len(g) != 1 || g[0] != "iron-dagger" {
		t.Errorf("single grant: got %v", g)
	}
}

func TestParseRecipeGrants_Multi(t *testing.T) {
	g := parseRecipeGrants("iron-dagger,iron-buckler")
	if len(g) != 2 {
		t.Fatalf("multi grant: want 2, got %d (%v)", len(g), g)
	}
	if g[0] != "iron-dagger" {
		t.Errorf("grant[0]: want iron-dagger, got %q", g[0])
	}
	if g[1] != "iron-buckler" {
		t.Errorf("grant[1]: want iron-buckler, got %q", g[1])
	}
}

func TestParseRecipeGrants_TrimsWhitespace(t *testing.T) {
	g := parseRecipeGrants(" iron-dagger , iron-buckler ")
	if len(g) != 2 {
		t.Fatalf("want 2, got %d (%v)", len(g), g)
	}
	if g[0] != "iron-dagger" {
		t.Errorf("grant[0]: want iron-dagger, got %q", g[0])
	}
	if g[1] != "iron-buckler" {
		t.Errorf("grant[1]: want iron-buckler, got %q", g[1])
	}
}

func TestParseRecipeGrants_SkipsEmptyEntries(t *testing.T) {
	// trailing comma, double-comma — empties are skipped
	g := parseRecipeGrants("iron-dagger,,healing-salve,")
	if len(g) != 2 {
		t.Fatalf("want 2, got %d (%v)", len(g), g)
	}
	if g[0] != "iron-dagger" {
		t.Errorf("grant[0]: want iron-dagger, got %q", g[0])
	}
	if g[1] != "healing-salve" {
		t.Errorf("grant[1]: want healing-salve, got %q", g[1])
	}
}
