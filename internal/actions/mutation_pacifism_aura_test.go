package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
)

// ---------------------------------------------------------------------------
// TriggerPacifismAura tests
// ---------------------------------------------------------------------------

// TestTriggerPacifismAura_NoMutation verifies that an actor without the
// pacifism-aura mutation is blocked with BlockReason="no-mutation".
func TestTriggerPacifismAura_NoMutation(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerPacifismAura(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false without mutation")
	}
	if res.BlockReason != "no-mutation" {
		t.Errorf("expected BlockReason=no-mutation, got %s", res.BlockReason)
	}
}

// TestTriggerPacifismAura_LowStamina verifies the stamina gate blocks the
// mutation when stamina is insufficient.
func TestTriggerPacifismAura_LowStamina(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"pacifism-aura": 1}
	mob.Character.Stamina = 5 // Below cost of 12
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerPacifismAura(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false with low stamina")
	}
	if res.BlockReason != "low-stamina" {
		t.Errorf("expected BlockReason=low-stamina, got %s", res.BlockReason)
	}
}

// TestTriggerPacifismAura_DeductsStamina verifies that stamina is deducted on
// success and the actor receives ConditionRecoveryPenalty (the self pacifism
// penalty). No combat requirement — pacifism-aura can fire out of combat.
func TestTriggerPacifismAura_DeductsStamina(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"pacifism-aura": 1}
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerPacifismAura(actor, MutationOpts{})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	if mob.Character.Stamina != 88 {
		t.Errorf("expected stamina deducted to 88 (100-12), got %d", mob.Character.Stamina)
	}
}

// TestTriggerPacifismAura_SelfPenalty verifies that on success the actor
// receives ConditionRecoveryPenalty (the pacifism self-penalty that prevents
// attacking for 3 rounds) and Aggro is cleared.
func TestTriggerPacifismAura_SelfPenalty(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"pacifism-aura": 1}
	mob.Character.Stamina = 100
	mob.Character.SetAggro(0, 9901, characters.DefaultAttack)
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerPacifismAura(actor, MutationOpts{})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	if !mob.Character.HasCondition(characters.ConditionRecoveryPenalty) {
		t.Error("expected actor to have ConditionRecoveryPenalty after firing pacifism-aura")
	}
	if mob.Character.IsInCombat() {
		t.Error("expected actor Aggro to be cleared (EndAggro) after firing pacifism-aura")
	}
}
