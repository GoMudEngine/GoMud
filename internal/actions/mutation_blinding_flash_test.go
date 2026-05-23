package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// newTestMobInRoom creates a mob and registers it in the room's mob list, so
// GetMobs() returns it. Uses an InstanceId that won't collide with the default
// 9900 attacker used in other helpers.
func newTestMobInRoom(t *testing.T, room *rooms.Room, instanceId int) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{InstanceId: instanceId}
	m.Character.Buffs = buffs.New()
	m.Character.Cooldowns = make(map[string]int)
	room.AddMob(instanceId)
	return m
}

// ---------------------------------------------------------------------------
// TriggerBlindingFlash tests
// ---------------------------------------------------------------------------

// TestTriggerBlindingFlash_NoMutation verifies that an actor without the
// blinding-flash mutation is blocked with BlockReason="no-mutation".
func TestTriggerBlindingFlash_NoMutation(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerBlindingFlash(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false without mutation")
	}
	if res.BlockReason != "no-mutation" {
		t.Errorf("expected BlockReason=no-mutation, got %s", res.BlockReason)
	}
}

// TestTriggerBlindingFlash_NotInCombat verifies that an actor with the
// mutation but not in combat is blocked with BlockReason="not-in-combat".
func TestTriggerBlindingFlash_NotInCombat(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"blinding-flash": 1}
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerBlindingFlash(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false when not in combat")
	}
	if res.BlockReason != "not-in-combat" {
		t.Errorf("expected BlockReason=not-in-combat, got %s", res.BlockReason)
	}
}

// TestTriggerBlindingFlash_Success_AoE verifies that a mob actor with the
// mutation, in combat (via SetAggro), fires the flash successfully. The
// victim receives ConditionBlinded and the attacker takes the self-blind.
// Room's GetMobs uses the live mob registry, so we only verify structural
// outcomes reachable without a full registry (AffectedCount may be 0 since
// the victim isn't seeded in the live registry, but Triggered must be true).
func TestTriggerBlindingFlash_Success_NoTargets(t *testing.T) {
	attacker := newTestMobBare(t)
	attacker.Character.Mutations = map[string]int{"blinding-flash": 1}
	attacker.Character.Stamina = 100
	attacker.Character.Stats.Willpower.ValueAdj = 200
	// Put attacker in combat using SetAggro (userId=0, mobInstanceId=9901).
	attacker.Character.SetAggro(0, 9901, characters.DefaultAttack)

	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(attacker, room)

	res := TriggerBlindingFlash(actor, MutationOpts{})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	// Attacker should have the self-blind condition.
	if !attacker.Character.HasCondition(characters.ConditionBlinded) {
		t.Error("expected attacker to have self-blind ConditionBlinded after firing")
	}
}

// TestTriggerBlindingFlash_LowStamina verifies the stamina gate blocks the
// mutation when stamina is insufficient.
func TestTriggerBlindingFlash_LowStamina(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"blinding-flash": 1}
	mob.Character.Stamina = 5 // Below cost of 14
	mob.Character.SetAggro(0, 9901, characters.DefaultAttack)
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerBlindingFlash(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false with low stamina")
	}
	if res.BlockReason != "low-stamina" {
		t.Errorf("expected BlockReason=low-stamina, got %s", res.BlockReason)
	}
}

// TestTriggerBlindingFlash_DeductsStamina verifies stamina is deducted on
// success.
func TestTriggerBlindingFlash_DeductsStamina(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"blinding-flash": 1}
	mob.Character.Stamina = 100
	mob.Character.SetAggro(0, 9901, characters.DefaultAttack)
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerBlindingFlash(actor, MutationOpts{})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	if mob.Character.Stamina != 86 {
		t.Errorf("expected stamina deducted to 86 (100-14), got %d", mob.Character.Stamina)
	}
}
