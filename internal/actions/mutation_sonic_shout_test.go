package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
)

// ---------------------------------------------------------------------------
// TriggerSonicShout tests
// ---------------------------------------------------------------------------

// TestTriggerSonicShout_NoMutation verifies that an actor without the
// sonic-shout mutation is blocked with BlockReason="no-mutation".
func TestTriggerSonicShout_NoMutation(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerSonicShout(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false without mutation")
	}
	if res.BlockReason != "no-mutation" {
		t.Errorf("expected BlockReason=no-mutation, got %s", res.BlockReason)
	}
}

// TestTriggerSonicShout_NotInCombat verifies that an actor with the
// mutation but not in combat is blocked with BlockReason="not-in-combat".
func TestTriggerSonicShout_NotInCombat(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"sonic-shout": 1}
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerSonicShout(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false when not in combat")
	}
	if res.BlockReason != "not-in-combat" {
		t.Errorf("expected BlockReason=not-in-combat, got %s", res.BlockReason)
	}
}

// TestTriggerSonicShout_LowStamina verifies the stamina gate blocks the
// mutation when stamina is insufficient.
func TestTriggerSonicShout_LowStamina(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"sonic-shout": 1}
	mob.Character.Stamina = 5 // Below cost of 15
	mob.Character.SetAggro(0, 9901, characters.DefaultAttack)
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerSonicShout(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false with low stamina")
	}
	if res.BlockReason != "low-stamina" {
		t.Errorf("expected BlockReason=low-stamina, got %s", res.BlockReason)
	}
}

// TestTriggerSonicShout_DeductsStamina verifies that stamina is deducted on
// success and the actor receives the self-deafen ConditionBlinded.
func TestTriggerSonicShout_DeductsStamina(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"sonic-shout": 1}
	mob.Character.Stamina = 100
	mob.Character.SetAggro(0, 9901, characters.DefaultAttack)
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerSonicShout(actor, MutationOpts{})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	if mob.Character.Stamina != 85 {
		t.Errorf("expected stamina deducted to 85 (100-15), got %d", mob.Character.Stamina)
	}
}

// TestTriggerSonicShout_SelfDeafen verifies that on success the actor
// always receives a ConditionBlinded (the self-deafen side-effect), even
// when there are no mob targets in the room.
func TestTriggerSonicShout_SelfDeafen(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"sonic-shout": 1}
	mob.Character.Stamina = 100
	mob.Character.SetAggro(0, 9901, characters.DefaultAttack)
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerSonicShout(actor, MutationOpts{})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	// Self-deafen: actor always receives ConditionBlinded after firing.
	if !mob.Character.HasCondition(characters.ConditionBlinded) {
		t.Error("expected actor to have ConditionBlinded (self-deafen) after firing sonic-shout")
	}
}
