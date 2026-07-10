package mobs

import (
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/GoMudEngine/GoMud/internal/characters"
)

// TestSpellbookParitySeeding: every mob gets the player StarterSpells
// merged UNDER any authored spellbook at Validate() — authored entries
// are never modified, baseline entries seed at 1.
func TestSpellbookParitySeeding(t *testing.T) {
	// Case 1: no authored spellbook → all baseline spells at 1.
	var plain Mob
	if err := yaml.Unmarshal([]byte("mobid: 991\ncharacter:\n  name: plainprobe\n"), &plain); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	plain.Validate()
	for _, id := range characters.StarterSpells {
		if plain.Character.SpellBook[id] != 1 {
			t.Errorf("plain mob SpellBook[%q] = %d, want 1", id, plain.Character.SpellBook[id])
		}
	}

	// Case 2: authored spellbook preserved, baseline merged under it.
	src := "mobid: 992\ncharacter:\n  name: casterprobe\n  spellbook:\n    nerve-disruption: 50\n    conviction-spike: 25\n"
	var caster Mob
	if err := yaml.Unmarshal([]byte(src), &caster); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	caster.Validate()
	if caster.Character.SpellBook["nerve-disruption"] != 50 {
		t.Errorf("authored nerve-disruption = %d, want 50 (untouched)", caster.Character.SpellBook["nerve-disruption"])
	}
	if caster.Character.SpellBook["conviction-spike"] != 25 {
		t.Errorf("authored conviction-spike = %d, want 25 (never overwritten by baseline)", caster.Character.SpellBook["conviction-spike"])
	}
	if caster.Character.SpellBook["identify"] != 1 {
		t.Errorf("baseline identify = %d, want 1 (merged)", caster.Character.SpellBook["identify"])
	}
}
