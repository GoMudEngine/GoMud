package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
)

// Corpse resolution should prefer the MOST RECENT corpse, not the oldest.
// Corpses are appended on death, and `look/loot corpse` (and repeated-look
// triggers) should inspect the last thing that died in the room.

func TestFindCorpse_SameName_ResolvesNewest(t *testing.T) {
	r := &Room{}
	r.AddCorpse(Corpse{MobId: 1, Character: characters.Character{Name: "goblin"}, RoundCreated: 1})
	r.AddCorpse(Corpse{MobId: 1, Character: characters.Character{Name: "goblin"}, RoundCreated: 2}) // newest

	got, ok := r.FindCorpse("goblin corpse")
	if !ok {
		t.Fatal("expected to find a goblin corpse")
	}
	if got.RoundCreated != 2 {
		t.Errorf("FindCorpse resolved RoundCreated=%d, want 2 (newest)", got.RoundCreated)
	}

	idx := r.FindCorpseIndex("goblin corpse")
	if idx < 0 {
		t.Fatal("FindCorpseIndex found nothing")
	}
	if r.Corpses[idx].RoundCreated != 2 {
		t.Errorf("FindCorpseIndex resolved RoundCreated=%d, want 2 (newest)", r.Corpses[idx].RoundCreated)
	}
}

func TestFindCorpse_GenericQuery_ResolvesNewest(t *testing.T) {
	r := &Room{}
	r.AddCorpse(Corpse{MobId: 1, Character: characters.Character{Name: "goblin"}, RoundCreated: 1})
	r.AddCorpse(Corpse{MobId: 2, Character: characters.Character{Name: "rat"}, RoundCreated: 2}) // newest

	// A generic "corpse" query should inspect the most recent kill (the rat).
	got, ok := r.FindCorpse("corpse")
	if !ok {
		t.Fatal("expected a generic corpse match")
	}
	if got.RoundCreated != 2 {
		t.Errorf("generic FindCorpse resolved RoundCreated=%d (%s), want 2 (rat, newest)", got.RoundCreated, got.Character.Name)
	}
}
