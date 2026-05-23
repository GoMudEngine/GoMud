package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// ---------------------------------------------------------------------------
// TriggerToxicBite tests
// ---------------------------------------------------------------------------

// TestTriggerToxicBite_NoMutation verifies that an actor without the
// toxic-bite mutation is blocked with BlockReason="no-mutation".
// The no-mutation gate fires before target resolution, so TargetActor
// can be nil.
func TestTriggerToxicBite_NoMutation(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerToxicBite(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false without mutation")
	}
	if res.BlockReason != "no-mutation" {
		t.Errorf("expected BlockReason=no-mutation, got %s", res.BlockReason)
	}
}

// TestTriggerToxicBite_NoTarget verifies that an actor with the mutation
// and in combat, but with no TargetActor, is blocked with
// BlockReason="no-target".
func TestTriggerToxicBite_NoTarget(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"toxic-bite": 1}
	mob.Character.Stamina = 100
	mob.Character.SetAggro(0, 9901, characters.DefaultAttack)
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerToxicBite(actor, MutationOpts{TargetActor: nil})
	if res.Triggered {
		t.Fatal("expected Triggered=false with no target")
	}
	if res.BlockReason != "no-target" {
		t.Errorf("expected BlockReason=no-target, got %s", res.BlockReason)
	}
}

// TestTriggerToxicBite_NotInCombat verifies that an actor with the
// mutation and a valid target, but not in combat, is blocked with
// BlockReason="not-in-combat".
func TestTriggerToxicBite_NotInCombat(t *testing.T) {
	attacker := newTestMobBare(t)
	attacker.Character.Mutations = map[string]int{"toxic-bite": 1}
	attacker.Character.Stamina = 100
	room := newTestRoomBare(t)

	victim := newTestMobBare(t)
	victim.InstanceId = 9901
	victim.Character.Stamina = 100
	victimActor := NewMobActorInRoom(victim, room)

	actor := NewMobActorInRoom(attacker, room)

	res := TriggerToxicBite(actor, MutationOpts{TargetActor: victimActor})
	if res.Triggered {
		t.Fatal("expected Triggered=false when not in combat")
	}
	if res.BlockReason != "not-in-combat" {
		t.Errorf("expected BlockReason=not-in-combat, got %s", res.BlockReason)
	}
}

// TestTriggerToxicBite_LowStamina verifies the stamina gate blocks the
// mutation when stamina is insufficient.
func TestTriggerToxicBite_LowStamina(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"toxic-bite": 1}
	mob.Character.Stamina = 5 // Below cost of 12
	mob.Character.SetAggro(0, 9901, characters.DefaultAttack)
	room := newTestRoomBare(t)

	victim := newTestMobBare(t)
	victim.InstanceId = 9901
	victim.Character.Stamina = 100
	victimActor := NewMobActorInRoom(victim, room)

	actor := NewMobActorInRoom(mob, room)

	res := TriggerToxicBite(actor, MutationOpts{TargetActor: victimActor})
	if res.Triggered {
		t.Fatal("expected Triggered=false with low stamina")
	}
	if res.BlockReason != "low-stamina" {
		t.Errorf("expected BlockReason=low-stamina, got %s", res.BlockReason)
	}
}

// TestTriggerToxicBite_DeductsStamina verifies that stamina is deducted
// on a successful trigger (mutation present, target provided, in combat).
// AffectedCount may be 0 if the roll fails, but Triggered must be true.
func TestTriggerToxicBite_DeductsStamina(t *testing.T) {
	attacker := newTestMobBare(t)
	attacker.Character.Mutations = map[string]int{"toxic-bite": 1}
	attacker.Character.Stamina = 100
	attacker.Character.SetAggro(0, 9901, characters.DefaultAttack)
	room := newTestRoomBare(t)

	victim := newTestMobBare(t)
	victim.InstanceId = 9901
	victim.Character.Stamina = 100
	victimActor := NewMobActorInRoom(victim, room)

	actor := NewMobActorInRoom(attacker, room)

	res := TriggerToxicBite(actor, MutationOpts{TargetActor: victimActor})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	// Stamina cost is 12.
	if attacker.Character.Stamina != 88 {
		t.Errorf("expected stamina deducted to 88 (100-12), got %d",
			attacker.Character.Stamina)
	}
}

// TestTriggerToxicBite_BiteDamageViaPipeline verifies that bite damage flows
// through the unified physical pipeline when the attack hits. We use a very
// high Strength attacker so the pipeline produces measurable damage regardless
// of dice variance; we only assert that health decreased (not an exact number)
// to avoid brittleness from dice.RollStat spread.
func TestTriggerToxicBite_BiteDamageViaPipeline(t *testing.T) {
	attacker := newTestMobBare(t)
	attacker.Character.Mutations = map[string]int{"toxic-bite": 1}
	attacker.Character.Stamina = 100
	// Very high Strength so CalcRawDamage (Physical channel) always
	// produces non-trivial damage.
	attacker.Character.Stats.Strength.Value = 300
	attacker.Character.Stats.Strength.ValueAdj = 300
	// Very high Dex to dominate the opposed roll and almost-certainly hit.
	attacker.Character.Stats.Dexterity.Value = 300
	attacker.Character.Stats.Dexterity.ValueAdj = 300
	attacker.Character.SetAggro(0, 9902, characters.DefaultAttack)

	room := newTestRoomBare(t)

	// Place victim in the room with bare stats (no physical mitigation).
	victim := newTestMobInRoom(t, room, 9902)
	victim.Character.Health = 500
	victim.Character.HealthMax.Value = 500
	startHealth := victim.Character.Health

	actor := NewMobActorInRoom(attacker, room)
	victimActor := NewMobActorInRoom(victim, room)

	res := TriggerToxicBite(actor, MutationOpts{TargetActor: victimActor})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	// The opposed roll can still fail (dice are random), so only assert
	// damage on a confirmed hit (AffectedCount > 0).
	if res.AffectedCount > 0 && victim.Character.Health >= startHealth {
		t.Errorf("AffectedCount=%d but victim health did not decrease: was %d, now %d",
			res.AffectedCount, startHealth, victim.Character.Health)
	}
	_ = mobs.GetInstance // ensure mobs import is exercised
}
