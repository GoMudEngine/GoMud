package characters

import "testing"

// TestStarterSpellsSeededOnNew guards the shared baseline: New()
// (player creation) and mobs.Mob.Validate() (actor parity) must both
// derive from StarterSpells, so a change to the list propagates to both.
func TestStarterSpellsSeededOnNew(t *testing.T) {
	want := []string{"conviction-spike", "chrysalis-glow", "identify"}
	if len(StarterSpells) != len(want) {
		t.Fatalf("StarterSpells = %v, want %v", StarterSpells, want)
	}
	for i, id := range want {
		if StarterSpells[i] != id {
			t.Fatalf("StarterSpells[%d] = %q, want %q", i, StarterSpells[i], id)
		}
	}

	c := New()
	for _, id := range StarterSpells {
		if c.SpellBook[id] != 1 {
			t.Errorf("New().SpellBook[%q] = %d, want 1", id, c.SpellBook[id])
		}
	}
}
