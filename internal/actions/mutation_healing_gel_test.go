package actions

import (
	"testing"
)

// ---------------------------------------------------------------------------
// TriggerHealingGel tests
// ---------------------------------------------------------------------------

// TestTriggerHealingGel_NoMutation verifies that an actor without the
// healing-gel mutation is blocked with BlockReason="no-mutation".
func TestTriggerHealingGel_NoMutation(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Stamina = 100
	mob.Character.HealthMax.Value = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerHealingGel(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false without mutation")
	}
	if res.BlockReason != "no-mutation" {
		t.Errorf("expected BlockReason=no-mutation, got %s", res.BlockReason)
	}
}

// TestTriggerHealingGel_WorksOutOfCombat verifies that an actor with the
// healing-gel mutation succeeds even when NOT in combat. This is the key
// distinction from the offensive mutations (combatRequired=false).
func TestTriggerHealingGel_WorksOutOfCombat(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"healing-gel": 1}
	mob.Character.Stamina = 100
	mob.Character.HealthMax.Value = 100
	mob.Character.Health = 50
	// Deliberately NOT in combat — no SetAggro.
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerHealingGel(actor, MutationOpts{})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true out of combat, got BlockReason=%s",
			res.BlockReason)
	}
	if res.AffectedCount != 1 {
		t.Errorf("expected AffectedCount=1 (self), got %d", res.AffectedCount)
	}
}

// TestTriggerHealingGel_LowStamina verifies the stamina gate blocks the
// mutation when stamina is insufficient (cost is 8).
func TestTriggerHealingGel_LowStamina(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"healing-gel": 1}
	mob.Character.Stamina = 5 // Below cost of 8
	mob.Character.HealthMax.Value = 100
	mob.Character.Health = 50
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerHealingGel(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false with low stamina")
	}
	if res.BlockReason != "low-stamina" {
		t.Errorf("expected BlockReason=low-stamina, got %s", res.BlockReason)
	}
}

// TestTriggerHealingGel_HealsFractionOfMax verifies that the heal amount
// scales with HealthMax (not a flat value or Vitality-based value).
// A mob with HealthMax=200 must heal more than one with HealthMax=100
// when both start at the same percentage of health.
func TestTriggerHealingGel_HealsFractionOfMax(t *testing.T) {
	mobA := newTestMobBare(t)
	mobA.InstanceId = 201
	mobA.Character.Mutations = map[string]int{"healing-gel": 1}
	mobA.Character.Stamina = 100
	mobA.Character.HealthMax.Value = 100
	mobA.Character.Health = 10
	roomA := newTestRoomBare(t)
	actorA := NewMobActorInRoom(mobA, roomA)

	mobB := newTestMobBare(t)
	mobB.InstanceId = 202
	mobB.Character.Mutations = map[string]int{"healing-gel": 1}
	mobB.Character.Stamina = 100
	mobB.Character.HealthMax.Value = 200
	mobB.Character.Health = 10
	roomB := newTestRoomBare(t)
	actorB := NewMobActorInRoom(mobB, roomB)

	resA := TriggerHealingGel(actorA, MutationOpts{})
	if !resA.Triggered {
		t.Fatalf("actor A: expected Triggered=true, got BlockReason=%s", resA.BlockReason)
	}

	resB := TriggerHealingGel(actorB, MutationOpts{})
	if !resB.Triggered {
		t.Fatalf("actor B: expected Triggered=true, got BlockReason=%s", resB.BlockReason)
	}

	healedA := mobA.Character.Health - 10
	healedB := mobB.Character.Health - 10

	// Both should have healed something.
	if healedA <= 0 {
		t.Errorf("actor A (HealthMax=100) healed %d, expected > 0", healedA)
	}
	if healedB <= 0 {
		t.Errorf("actor B (HealthMax=200) healed %d, expected > 0", healedB)
	}
	// The higher-HealthMax mob should heal more (fraction-of-max scaling).
	if healedB <= healedA {
		t.Errorf("expected healedB (%d) > healedA (%d) for HealthMax 200 vs 100",
			healedB, healedA)
	}
}

// TestTriggerHealingGel_DeductsStamina verifies stamina is deducted on
// success (cost = 8).
func TestTriggerHealingGel_DeductsStamina(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"healing-gel": 1}
	mob.Character.Stamina = 100
	mob.Character.HealthMax.Value = 100
	mob.Character.Health = 50
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerHealingGel(actor, MutationOpts{})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	if mob.Character.Stamina != 92 { // 100 - 8
		t.Errorf("expected stamina deducted to 92 (100-8), got %d", mob.Character.Stamina)
	}
}

// TestTriggerHealingGel_HealthCappedAtMax verifies that health does not
// exceed HealthMax after healing (no overheal).
func TestTriggerHealingGel_HealthCappedAtMax(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"healing-gel": 1}
	mob.Character.Stamina = 100
	mob.Character.HealthMax.Value = 100
	mob.Character.Health = 99 // Already near full
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	TriggerHealingGel(actor, MutationOpts{})
	if mob.Character.Health > mob.Character.HealthMax.Value {
		t.Errorf("health %d exceeds HealthMax %d after healing",
			mob.Character.Health, mob.Character.HealthMax.Value)
	}
}
