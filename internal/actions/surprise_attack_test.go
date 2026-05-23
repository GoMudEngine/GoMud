package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/stats"
)

// ---------------------------------------------------------------------------
// surprise_attack_test.go — unit tests for actions.SurpriseAttack
//
// Hidden state mechanics: IsHidden() delegates to the Awareness FSM.
// addHiddenBuff (defined in shadow_test.go) seeds both buff #9 and the
// Awareness machine so that c.IsHidden() returns true. That init() also
// seeds the buff spec registry so AddBuff(9, …) does not panic.
//
// The three tests cover the two early-out gates and the happy path.
// ---------------------------------------------------------------------------

// newSurpriseTestRoom builds a minimal room for surprise-attack tests.
func newSurpriseTestRoom(t *testing.T) *rooms.Room {
	t.Helper()
	return &rooms.Room{RoomId: 9800}
}

// newSurpriseAttackerMob builds a minimal attacker mob with Cooldowns
// initialized. Call addHiddenBuff(&mob.Character) to make it hidden.
func newSurpriseAttackerMob(t *testing.T) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{InstanceId: 9800}
	m.Character.Cooldowns = make(map[string]int)
	m.Character.Stats.Strength.ValueAdj = 100
	m.Character.Stats.Dexterity.ValueAdj = 100
	return m
}

// newSurpriseVictimMob builds a minimal victim mob registered in the room.
func newSurpriseVictimMob(t *testing.T, room *rooms.Room, instanceId int) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{InstanceId: instanceId}
	m.Character.Health = 200
	m.Character.HealthMax = stats.StatInfo{Value: 200}
	room.AddMob(instanceId)
	return m
}

// ---------------------------------------------------------------------------
// TestSurpriseAttack_NotHidden
// ---------------------------------------------------------------------------

// TestSurpriseAttack_NotHidden verifies that an actor without the Hidden
// buff is blocked with Triggered=false and BlockReason="not-hidden".
func TestSurpriseAttack_NotHidden(t *testing.T) {
	attacker := newSurpriseAttackerMob(t)
	// Do NOT call addHiddenBuff — attacker is visible.
	room := newSurpriseTestRoom(t)
	victim := newSurpriseVictimMob(t, room, 9801)
	actor := NewMobActorInRoom(attacker, room)
	target := NewMobActorInRoom(victim, room)

	res := SurpriseAttack(actor, SurpriseAttackOpts{Target: target})
	if res.Triggered {
		t.Fatal("expected Triggered=false when attacker is not hidden")
	}
	if res.BlockReason != "not-hidden" {
		t.Errorf("expected BlockReason=not-hidden, got %q", res.BlockReason)
	}
}

// ---------------------------------------------------------------------------
// TestSurpriseAttack_NoTarget
// ---------------------------------------------------------------------------

// TestSurpriseAttack_NoTarget verifies that a hidden actor with Target=nil
// is blocked with Triggered=false and BlockReason="no-target".
func TestSurpriseAttack_NoTarget(t *testing.T) {
	attacker := newSurpriseAttackerMob(t)
	addHiddenBuff(&attacker.Character)
	room := newSurpriseTestRoom(t)
	actor := NewMobActorInRoom(attacker, room)

	res := SurpriseAttack(actor, SurpriseAttackOpts{Target: nil})
	if res.Triggered {
		t.Fatal("expected Triggered=false when target is nil")
	}
	if res.BlockReason != "no-target" {
		t.Errorf("expected BlockReason=no-target, got %q", res.BlockReason)
	}
}

// ---------------------------------------------------------------------------
// TestSurpriseAttack_Success_FiresPerWeapon
// ---------------------------------------------------------------------------

// TestSurpriseAttack_Success_FiresPerWeapon verifies that a hidden actor with
// a valid target triggers the pre-combat burst and StrikeCount >= 1.
// The attacker has no weapons equipped, so the unarmed fallback fires.
func TestSurpriseAttack_Success_FiresPerWeapon(t *testing.T) {
	attacker := newSurpriseAttackerMob(t)
	addHiddenBuff(&attacker.Character)
	room := newSurpriseTestRoom(t)
	victim := newSurpriseVictimMob(t, room, 9802)
	actor := NewMobActorInRoom(attacker, room)
	target := NewMobActorInRoom(victim, room)

	res := SurpriseAttack(actor, SurpriseAttackOpts{Target: target})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true for hidden actor with valid target, got BlockReason=%q",
			res.BlockReason)
	}
	// Unarmed fallback always produces at least 1 entry in the weapons list,
	// so StrikeCount must be >= 1 (the hit-penalty roll for fists is 0% so
	// the swing always connects).
	if res.StrikeCount < 1 {
		t.Errorf("expected StrikeCount >= 1 (unarmed fallback), got %d", res.StrikeCount)
	}
}

// ---------------------------------------------------------------------------
// TestSurpriseAttack_OnCooldown_Blocked
// ---------------------------------------------------------------------------

// TestSurpriseAttack_OnCooldown_Blocked verifies that a hidden actor whose
// special-move cooldown is active is blocked with BlockReason="on-cooldown".
func TestSurpriseAttack_OnCooldown_Blocked(t *testing.T) {
	attacker := newSurpriseAttackerMob(t)
	addHiddenBuff(&attacker.Character)
	// Pre-seed the cooldown so TryCooldown returns false.
	attacker.Character.Cooldowns["special-move"] = 5
	room := newSurpriseTestRoom(t)
	victim := newSurpriseVictimMob(t, room, 9803)
	actor := NewMobActorInRoom(attacker, room)
	target := NewMobActorInRoom(victim, room)

	res := SurpriseAttack(actor, SurpriseAttackOpts{Target: target})
	if res.Triggered {
		t.Fatal("expected Triggered=false when on cooldown")
	}
	if res.BlockReason != "on-cooldown" {
		t.Errorf("expected BlockReason=on-cooldown, got %q", res.BlockReason)
	}
}
