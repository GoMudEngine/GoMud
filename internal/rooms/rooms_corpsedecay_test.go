package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// Corpse-loot redesign (2026-07-07), Task 4: when a corpse decays, any
// loot it still holds must fall to the room floor as a last resort so it
// isn't destroyed alongside the corpse.
func TestUpdateCorpses_DropsLootOnDecay(t *testing.T) {
	// Corpses default OFF in tests; UpdateCorpses early-returns when
	// disabled, so enable them via a config overlay for the duration.
	if err := configs.AddOverlayOverrides(map[string]any{
		"GamePlay.Death.CorpsesEnabled": true,
	}); err != nil {
		t.Fatalf("failed to enable corpses: %v", err)
	}
	defer configs.AddOverlayOverrides(map[string]any{
		"GamePlay.Death.CorpsesEnabled": false,
	})

	room := &Room{
		RoomId: 1,
		Zone:   "TestZone",
	}

	corpse := Corpse{
		MobId:        1,
		RoundCreated: 1,
		Prunable:     true, // force decay/prune this tick
	}
	corpse.Loot.Gold = 10
	corpse.Loot.AddItem(items.Item{ItemId: 5})
	room.AddCorpse(corpse)

	room.UpdateCorpses(999999999)

	if room.Gold != 10 {
		t.Errorf("expected floor gold 10, got %d", room.Gold)
	}
	if len(room.Items) != 1 {
		t.Errorf("expected 1 item on the floor, got %d", len(room.Items))
	}
	if len(room.Corpses) != 0 {
		t.Errorf("expected corpse to be pruned, got %d remaining", len(room.Corpses))
	}
}
