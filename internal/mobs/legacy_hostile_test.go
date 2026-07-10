package mobs

import (
	"testing"

	"gopkg.in/yaml.v2"
)

// TestLegacyHostileYAMLBackcompat guards the `hostile:` → AutoAggro
// backward-compat path. The b1145cdb6 aggro sunset lowercased the
// LegacyHostile field; yaml unmarshal silently skips unexported fields,
// so every `hostile: true` mob in the world stopped auto-aggroing on
// player entry — undetected from 2026-05-13 to 2026-07-10. This test
// exists so that can never happen silently again.
func TestLegacyHostileYAMLBackcompat(t *testing.T) {
	var m Mob
	if err := yaml.Unmarshal([]byte("mobid: 999\nhostile: true\ncharacter:\n  name: probe\n"), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !m.LegacyHostile {
		t.Fatalf("`hostile: true` did not populate LegacyHostile — yaml tag broken or field unexported again")
	}
	m.Validate()
	if !m.AutoAggro {
		t.Fatalf("`hostile: true` did not propagate to AutoAggro via Validate()")
	}

	// The canonical spelling must work too, and false must stay false.
	var m2 Mob
	if err := yaml.Unmarshal([]byte("mobid: 998\nauto_aggro: true\ncharacter:\n  name: probe2\n"), &m2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	m2.Validate()
	if !m2.AutoAggro {
		t.Fatalf("`auto_aggro: true` did not populate AutoAggro")
	}

	var m3 Mob
	if err := yaml.Unmarshal([]byte("mobid: 997\nhostile: false\ncharacter:\n  name: probe3\n"), &m3); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	m3.Validate()
	if m3.AutoAggro {
		t.Fatalf("`hostile: false` must not set AutoAggro")
	}
}
