package hooks

import "testing"

// parseSkillGrants backs multi-skill quest rewards (e.g. Spoke A's outer
// cert grants a rank of both weapon-combat and unarmed-combat). The
// floor-raise guard lives in the handler; these tests pin the parse.

func TestParseSkillGrants_Empty(t *testing.T) {
	if g := parseSkillGrants(""); g != nil {
		t.Errorf("empty skill_info: want nil, got %v", g)
	}
}

func TestParseSkillGrants_LegacySingle(t *testing.T) {
	g := parseSkillGrants("map:1")
	if len(g) != 1 || g[0].skill != "map" || g[0].level != 1 {
		t.Errorf("single grant: got %+v", g)
	}
}

func TestParseSkillGrants_MultiSkill(t *testing.T) {
	g := parseSkillGrants("weapon-combat:1,unarmed-combat:1")
	if len(g) != 2 {
		t.Fatalf("multi grant: want 2, got %d (%+v)", len(g), g)
	}
	if g[0].skill != "weapon-combat" || g[0].level != 1 {
		t.Errorf("grant[0]: got %+v", g[0])
	}
	if g[1].skill != "unarmed-combat" || g[1].level != 1 {
		t.Errorf("grant[1]: got %+v", g[1])
	}
}

func TestParseSkillGrants_TrimsWhitespaceAndCase(t *testing.T) {
	g := parseSkillGrants(" Weapon-Combat : 2 , Unarmed-Combat:3 ")
	if len(g) != 2 {
		t.Fatalf("want 2, got %d (%+v)", len(g), g)
	}
	if g[0].skill != "weapon-combat" || g[0].level != 2 {
		t.Errorf("grant[0]: got %+v", g[0])
	}
	if g[1].skill != "unarmed-combat" || g[1].level != 3 {
		t.Errorf("grant[1]: got %+v", g[1])
	}
}

func TestParseSkillGrants_SkipsMalformed(t *testing.T) {
	// no colon, unparseable level, and empty skill are each dropped;
	// the one valid entry survives.
	g := parseSkillGrants("garbage,:5,foo:notanumber,salvage:2")
	if len(g) != 1 || g[0].skill != "salvage" || g[0].level != 2 {
		t.Errorf("want only salvage:2, got %+v", g)
	}
}
