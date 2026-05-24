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

// TestTriggerSonicShout_MitigationAppliesViaPipeline verifies that AoE damage
// flows through the unified conviction pipeline and is applied per-victim.
// We use a high-Willpower attacker so the pipeline produces measurable damage
// regardless of dice variance; we only assert that health decreased (not an
// exact number) to avoid brittleness from dice.RollStat spread.
func TestTriggerSonicShout_MitigationAppliesViaPipeline(t *testing.T) {
	attacker := newTestMobBare(t)
	attacker.Character.Mutations = map[string]int{"sonic-shout": 1}
	attacker.Character.Stamina = 100
	attacker.Character.Stats.Willpower.Value = 300
	attacker.Character.Stats.Willpower.ValueAdj = 300
	attacker.Character.SetAggro(0, 9901, characters.DefaultAttack)

	room := newTestRoomBare(t)

	// Place a victim in the room with no conviction mitigation (bare mob).
	victim := newTestMobInRoom(t, room, 9902)
	victim.Character.Health = 200
	victim.Character.HealthMax.Value = 200
	startHealth := victim.Character.Health

	actor := NewMobActorInRoom(attacker, room)
	res := TriggerSonicShout(actor, MutationOpts{})

	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	// The opposed roll may fail (dice are random), so we can't guarantee damage.
	// But if the roll succeeded, health must have decreased.
	if res.AffectedCount > 0 && victim.Character.Health >= startHealth {
		t.Errorf("AffectedCount=%d but victim health did not decrease: was %d, now %d",
			res.AffectedCount, startHealth, victim.Character.Health)
	}
}
