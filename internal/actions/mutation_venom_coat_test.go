package actions

import "testing"

func TestTriggerVenomCoat_NoMutation(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerVenomCoat(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false without mutation")
	}
	if res.BlockReason != "no-mutation" {
		t.Errorf("expected BlockReason=no-mutation, got %s", res.BlockReason)
	}
}

func TestTriggerVenomCoat_WorksOutOfCombat(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"venom-coat": 1}
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerVenomCoat(actor, MutationOpts{})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true out of combat, got BlockReason=%s", res.BlockReason)
	}
	if res.AffectedCount != 1 {
		t.Errorf("expected AffectedCount=1 (self), got %d", res.AffectedCount)
	}
}

func TestTriggerVenomCoat_LowStamina(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"venom-coat": 1}
	mob.Character.Stamina = 2 // cost is 8
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerVenomCoat(actor, MutationOpts{})
	if res.BlockReason != "low-stamina" {
		t.Errorf("expected BlockReason=low-stamina, got %s", res.BlockReason)
	}
}
