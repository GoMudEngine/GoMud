package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// ---------------------------------------------------------------------------
// mutation_helpers test-local helpers
// ---------------------------------------------------------------------------

// newTestMobBare returns the minimal *mobs.Mob needed for MobActor tests.
// Buffs is initialised so downstream code that reads Buffs doesn't panic.
func newTestMobBare(t *testing.T) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{InstanceId: 9900}
	m.Character.Buffs = buffs.New()
	m.Character.Cooldowns = make(map[string]int)
	return m
}

// newTestRoomBare returns a minimal *rooms.Room with a stable RoomId.
func newTestRoomBare(t *testing.T) *rooms.Room {
	t.Helper()
	return &rooms.Room{RoomId: 9900}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestMutationPreamble_NoMutation verifies that an actor without the
// requested mutation receives BlockReason="no-mutation" and OK=false.
func TestMutationPreamble_NoMutation(t *testing.T) {
	mob := newTestMobBare(t)
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := mutationPreamble(actor, "blinding-flash", true, 14)
	if res.OK {
		t.Fatal("expected preamble to fail without mutation")
	}
	if res.BlockReason != "no-mutation" {
		t.Errorf("expected BlockReason=no-mutation, got %s", res.BlockReason)
	}
}

// TestMutationPreamble_NotInCombat verifies that a mutation-owning actor
// that is not in combat is blocked when combatRequired=true.
func TestMutationPreamble_NotInCombat(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"blinding-flash": 1}
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := mutationPreamble(actor, "blinding-flash", true, 14)
	if res.OK {
		t.Fatal("expected preamble to fail when not in combat")
	}
	if res.BlockReason != "not-in-combat" {
		t.Errorf("expected BlockReason=not-in-combat, got %s", res.BlockReason)
	}
}

// TestMutationPreamble_OnCooldown verifies that an actor with an active
// special-move cooldown is blocked with BlockReason="on-cooldown".
// healing-gel uses combatRequired=false so we skip the combat gate.
func TestMutationPreamble_OnCooldown(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"healing-gel": 1}
	mob.Character.Stamina = 100
	// Pre-seed the cooldown so Try() returns false.
	mob.Character.Cooldowns["special-move"] = 5

	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := mutationPreamble(actor, "healing-gel", false, 14)
	if res.OK {
		t.Fatal("expected preamble to fail when on cooldown")
	}
	if res.BlockReason != "on-cooldown" {
		t.Errorf("expected BlockReason=on-cooldown, got %s", res.BlockReason)
	}
	// Verify stamina was NOT deducted (it's an early-out before the stamina gate)
	if mob.Character.Stamina != 100 {
		t.Errorf("expected stamina unchanged at 100, got %d", mob.Character.Stamina)
	}
}

// TestMutationPreamble_LowStamina verifies that an actor with insufficient
// stamina is blocked with BlockReason="low-stamina".
// healing-gel uses combatRequired=false so we skip the combat gate.
func TestMutationPreamble_LowStamina(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"healing-gel": 1}
	mob.Character.Stamina = 1
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := mutationPreamble(actor, "healing-gel", false, 14)
	if res.OK {
		t.Fatal("expected preamble to fail with low stamina")
	}
	if res.BlockReason != "low-stamina" {
		t.Errorf("expected BlockReason=low-stamina, got %s", res.BlockReason)
	}
}

// TestMutationPreamble_Success verifies that a fully qualified actor passes
// the preamble and has the stamina cost deducted.
func TestMutationPreamble_Success(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"healing-gel": 1}
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := mutationPreamble(actor, "healing-gel", false, 14)
	if !res.OK {
		t.Fatalf("expected preamble OK, got BlockReason=%s", res.BlockReason)
	}
	if mob.Character.Stamina != 86 {
		t.Errorf("expected stamina deducted to 86, got %d", mob.Character.Stamina)
	}
}
