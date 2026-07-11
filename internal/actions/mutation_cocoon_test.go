package actions

import "testing"

func TestTriggerCocoon_NoMutation(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerCocoon(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false without mutation")
	}
	if res.BlockReason != "no-mutation" {
		t.Errorf("expected BlockReason=no-mutation, got %s", res.BlockReason)
	}
}

func TestTriggerCocoon_WorksOutOfCombat(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"cocoon": 1}
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerCocoon(actor, MutationOpts{})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true out of combat, got BlockReason=%s", res.BlockReason)
	}
}

func TestTriggerCocoon_LowStamina(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"cocoon": 1}
	mob.Character.Stamina = 3 // cost is 10
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerCocoon(actor, MutationOpts{})
	if res.BlockReason != "low-stamina" {
		t.Errorf("expected BlockReason=low-stamina, got %s", res.BlockReason)
	}
}
