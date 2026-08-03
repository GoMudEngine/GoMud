package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
)

// Gate 0 (2026-08-03 crafting-focus audit): a mutation active must refuse
// while the actor's Activity machine is occupied — firing venom-coat or
// cocoon mid-craft was the rally/warcry bug through a different door.
func TestMutationPreamble_BusyWhileCrafting(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"venom-coat": 1}
	mob.Character.Activity = activity.NewMachine()
	if err := mob.Character.Activity.TransitionToCrafting(
		activity.CraftingData{RecipeId: "test", RoundsTotal: 1},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin},
	); err != nil {
		t.Fatalf("fixture: could not enter Crafting state: %v", err)
	}
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := mutationPreamble(actor, "venom-coat", false, 8)
	if res.OK {
		t.Fatal("preamble must fail while crafting")
	}
	if res.BlockReason != "busy" {
		t.Errorf("expected BlockReason=busy, got %s", res.BlockReason)
	}
	// The block must happen BEFORE the cooldown gate — no cooldown consumed,
	// no stamina spent.
	if mob.Character.Cooldowns["special-move"] != 0 {
		t.Error("busy block must not consume the special-move cooldown")
	}
}
