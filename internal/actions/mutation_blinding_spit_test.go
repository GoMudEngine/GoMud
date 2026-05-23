package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
)

// ---------------------------------------------------------------------------
// TriggerBlindingSpit tests
// ---------------------------------------------------------------------------

// TestTriggerBlindingSpit_NoMutation verifies that an actor without the
// blinding-spit mutation is blocked with BlockReason="no-mutation".
// The no-mutation gate fires before target resolution, so TargetActor
// can be nil.
func TestTriggerBlindingSpit_NoMutation(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerBlindingSpit(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false without mutation")
	}
	if res.BlockReason != "no-mutation" {
		t.Errorf("expected BlockReason=no-mutation, got %s", res.BlockReason)
	}
}

// TestTriggerBlindingSpit_NoTarget verifies that an actor with the mutation
// and in combat, but with no TargetActor, is blocked with
// BlockReason="no-target".
func TestTriggerBlindingSpit_NoTarget(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"blinding-spit": 1}
	mob.Character.Stamina = 100
	mob.Character.SetAggro(0, 9901, characters.DefaultAttack)
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerBlindingSpit(actor, MutationOpts{TargetActor: nil})
	if res.Triggered {
		t.Fatal("expected Triggered=false with no target")
	}
	if res.BlockReason != "no-target" {
		t.Errorf("expected BlockReason=no-target, got %s", res.BlockReason)
	}
}

// TestTriggerBlindingSpit_NotInCombat verifies that an actor with the
// mutation and a valid target, but not in combat, is blocked with
// BlockReason="not-in-combat".
func TestTriggerBlindingSpit_NotInCombat(t *testing.T) {
	attacker := newTestMobBare(t)
	attacker.Character.Mutations = map[string]int{"blinding-spit": 1}
	attacker.Character.Stamina = 100
	room := newTestRoomBare(t)

	victim := newTestMobBare(t)
	victim.InstanceId = 9901
	victim.Character.Stamina = 100
	victimActor := NewMobActorInRoom(victim, room)

	actor := NewMobActorInRoom(attacker, room)

	res := TriggerBlindingSpit(actor, MutationOpts{TargetActor: victimActor})
	if res.Triggered {
		t.Fatal("expected Triggered=false when not in combat")
	}
	if res.BlockReason != "not-in-combat" {
		t.Errorf("expected BlockReason=not-in-combat, got %s", res.BlockReason)
	}
}

// TestTriggerBlindingSpit_DeductsStamina verifies that stamina is deducted
// on a successful trigger (mutation present, target provided, in combat).
// AffectedCount may be 0 if the roll fails, but Triggered must be true.
func TestTriggerBlindingSpit_DeductsStamina(t *testing.T) {
	attacker := newTestMobBare(t)
	attacker.Character.Mutations = map[string]int{"blinding-spit": 1}
	attacker.Character.Stamina = 100
	attacker.Character.SetAggro(0, 9901, characters.DefaultAttack)
	room := newTestRoomBare(t)

	victim := newTestMobBare(t)
	victim.InstanceId = 9901
	victim.Character.Stamina = 100
	victimActor := NewMobActorInRoom(victim, room)

	actor := NewMobActorInRoom(attacker, room)

	res := TriggerBlindingSpit(actor, MutationOpts{TargetActor: victimActor})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	// Stamina cost is 10.
	if attacker.Character.Stamina != 90 {
		t.Errorf("expected stamina deducted to 90 (100-10), got %d",
			attacker.Character.Stamina)
	}
}
