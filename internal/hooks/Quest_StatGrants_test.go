package hooks

import "testing"

// parseStatGrants backs multi-stat quest rewards (e.g. a culmination quest
// grants both strength and vitality). These tests pin the parse; the additive
// application logic lives in the handler itself.

func TestParseStatGrants_Empty(t *testing.T) {
	if g := parseStatGrants(""); g != nil {
		t.Errorf("empty stat_info: want nil, got %v", g)
	}
}

func TestParseStatGrants_Single(t *testing.T) {
	g := parseStatGrants("strength:5")
	if len(g) != 1 || g[0].stat != "strength" || g[0].amount != 5 {
		t.Errorf("single grant: got %+v", g)
	}
}

func TestParseStatGrants_Multi(t *testing.T) {
	g := parseStatGrants("strength:3,vitality:2")
	if len(g) != 2 {
		t.Fatalf("multi grant: want 2, got %d (%+v)", len(g), g)
	}
	if g[0].stat != "strength" || g[0].amount != 3 {
		t.Errorf("grant[0]: got %+v", g[0])
	}
	if g[1].stat != "vitality" || g[1].amount != 2 {
		t.Errorf("grant[1]: got %+v", g[1])
	}
}

func TestParseStatGrants_TrimsWhitespaceAndCase(t *testing.T) {
	g := parseStatGrants(" Strength : 4 , Willpower:1 ")
	if len(g) != 2 {
		t.Fatalf("want 2, got %d (%+v)", len(g), g)
	}
	if g[0].stat != "strength" || g[0].amount != 4 {
		t.Errorf("grant[0]: got %+v", g[0])
	}
	if g[1].stat != "willpower" || g[1].amount != 1 {
		t.Errorf("grant[1]: got %+v", g[1])
	}
}

func TestParseStatGrants_SkipsMalformed(t *testing.T) {
	// no colon, unparseable amount, and empty name are each dropped;
	// the one valid entry survives.
	g := parseStatGrants("garbage,:5,strength:notanumber,perception:2")
	if len(g) != 1 || g[0].stat != "perception" || g[0].amount != 2 {
		t.Errorf("want only perception:2, got %+v", g)
	}
}

func TestParseStatGrants_AllSixStats(t *testing.T) {
	g := parseStatGrants(
		"strength:1,dexterity:1,perception:1,vitality:1,willpower:1,charisma:1",
	)
	if len(g) != 6 {
		t.Fatalf("want 6 grants, got %d (%+v)", len(g), g)
	}
	expected := []string{
		"strength", "dexterity", "perception",
		"vitality", "willpower", "charisma",
	}
	for i, want := range expected {
		if g[i].stat != want || g[i].amount != 1 {
			t.Errorf("grant[%d]: want {%s 1}, got %+v", i, want, g[i])
		}
	}
}
