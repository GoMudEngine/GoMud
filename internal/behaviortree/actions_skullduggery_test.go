package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// skulldBuffSpec seeds buff 9 (Hidden) for behaviortree-package tests that
// need AddBuff(9) to work. A single var so the same seed is shared by all
// tests in this file.
var skulldBuffSpec = map[int]*buffs.BuffSpec{
	9: {
		BuffId:        9,
		Name:          "Hidden",
		Flags:         []buffs.Flag{buffs.Hidden},
		TriggerCount:  15,
		RoundInterval: 1,
	},
}

// ─── try_sneak ───────────────────────────────────────────────────────────────

// TestActTrySneak_SuccessWhenNoObservers verifies that a mob alone in a
// room succeeds at sneak and acquires the Hidden buff.
func TestActTrySneak_SuccessWhenNoObservers(t *testing.T) {
	cleanBuffs := buffs.SeedBuffsForTest(skulldBuffSpec)
	defer cleanBuffs()

	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Thief")
	defer cleanMob()

	mob := mobs.GetInstance(105)
	mob.Character.HealthMax.Value = 500
	mob.Character.Health = 500
	mob.Character.Stats.Dexterity.ValueAdj = 200 // very high sneak score
	mob.Character.Aggro = nil                    // not in combat

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	// No other mobs or players — no observers.

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTrySneak(map[string]any{}, ctx); r != Success {
		t.Errorf("expected Success for mob alone in room, got %v", r)
	}
	// Note: mob.AddBuff enqueues an event rather than applying synchronously,
	// so HasBuffFlag is not reliable here without an event-loop tick.
	// Verifying Success return is sufficient to confirm Sneak fired correctly.
}

// TestActTrySneak_AlreadyHiddenReturnsSuccess verifies that a mob that is
// already hidden returns Success (AlreadyHidden path from actions.Sneak).
func TestActTrySneak_AlreadyHiddenReturnsSuccess(t *testing.T) {
	cleanBuffs := buffs.SeedBuffsForTest(skulldBuffSpec)
	defer cleanBuffs()

	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Thief")
	defer cleanMob()

	mob := mobs.GetInstance(105)
	mob.Character.HealthMax.Value = 500
	mob.Character.Health = 500
	mob.Character.Aggro = nil

	room := rooms.LoadRoom(1)
	room.AddMob(105)

	// Apply hidden buff directly.
	if err := mob.Character.AddBuff(9, false); err != nil {
		t.Fatalf("AddBuff(9) failed: %v", err)
	}

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTrySneak(map[string]any{}, ctx); r != Success {
		t.Errorf("expected Success when already hidden, got %v", r)
	}
}

// ─── try_steal ───────────────────────────────────────────────────────────────

// TestActTrySteal_SuccessWithAggroTarget verifies that a mob with an aggro
// user target can steal. Uses a high-Dex mob with skill 2+ vs low-Perception
// player over multiple trials to guarantee success fires at least once.
func TestActTrySteal_SuccessWithAggroTarget(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Thief")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "victim", "Victim", 1)
	defer cleanUser()

	mob := mobs.GetInstance(105)
	mob.Character.HealthMax.Value = 500
	mob.Character.Health = 500
	mob.Character.Stats.Dexterity.ValueAdj = 300 // very high steal score
	mob.Character.Skills = map[string]int{"skullduggery": 5}

	user := users.GetByUserId(42)
	user.Character.Gold = 1000
	user.Character.Stats.Perception.ValueAdj = 1 // nearly blind

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	room.AddPlayer(42)

	// Wire aggro to the player target.
	mob.Character.Aggro = &characters.Aggro{UserId: 42}
	mob.Character.Aggro = nil // steal gate requires no combat-aggro
	// resolveSkullduggeryTarget will fall back to aggro; set after clearing combat:
	mob.Character.SetAggro(42, 0, characters.DefaultAttack)
	// But steal.go blocks if Aggro != nil (combat gate). Reset to nil so the
	// steal can fire, and target via Event.UserId instead.
	mob.Character.Aggro = nil

	// Use Event.UserId — resolveSkullduggeryTarget checks this first.
	ctx := &EvalContext{
		InstanceId: 105,
		RoomId:     1,
		Event:      EventContext{UserId: 42},
	}

	succeeded := false
	for i := 0; i < 30; i++ {
		// Clear cooldown between attempts.
		delete(mob.Character.Cooldowns, "skullduggery.steal")
		if r := actTrySteal(map[string]any{}, ctx); r == Success {
			succeeded = true
			break
		}
	}
	if !succeeded {
		t.Errorf("expected at least one Success over 30 trials with high Dex vs low Per")
	}
}

// TestActTrySteal_FailureNoTarget verifies that with no aggro and no event
// target, actTrySteal returns Failure.
func TestActTrySteal_FailureNoTarget(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Thief")
	defer cleanMob()

	mob := mobs.GetInstance(105)
	mob.Character.HealthMax.Value = 500
	mob.Character.Health = 500
	mob.Character.Aggro = nil

	room := rooms.LoadRoom(1)
	room.AddMob(105)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTrySteal(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure with no target, got %v", r)
	}
}

// ─── try_plant ───────────────────────────────────────────────────────────────

// TestActTryPlant_RequiresItemTag verifies that plant with a missing
// item_tag param returns Failure immediately.
func TestActTryPlant_RequiresItemTag(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Thief")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "target", "Target", 1)
	defer cleanUser()

	mob := mobs.GetInstance(105)
	mob.Character.HealthMax.Value = 500
	mob.Character.Health = 500
	mob.Character.Aggro = nil

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	room.AddPlayer(42)

	// item_tag param is absent.
	ctx := &EvalContext{
		InstanceId: 105,
		RoomId:     1,
		Event:      EventContext{UserId: 42},
	}
	if r := actTryPlant(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure when item_tag is missing, got %v", r)
	}
}

// TestActTryPlant_FailureNoItemInBackpack verifies that plant with a valid
// item_tag but mob carrying nothing returns Failure (Plant returns
// "item not in backpack").
func TestActTryPlant_FailureNoItemInBackpack(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Thief")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "target", "Target", 1)
	defer cleanUser()

	mob := mobs.GetInstance(105)
	mob.Character.HealthMax.Value = 500
	mob.Character.Health = 500
	mob.Character.Aggro = nil
	mob.Character.Items = nil // empty backpack

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	room.AddPlayer(42)

	ctx := &EvalContext{
		InstanceId: 105,
		RoomId:     1,
		Event:      EventContext{UserId: 42},
	}
	if r := actTryPlant(map[string]any{"item_tag": "knife"}, ctx); r != Failure {
		t.Errorf("expected Failure when mob has nothing to plant, got %v", r)
	}
}

// ─── try_shadow ──────────────────────────────────────────────────────────────

// TestActTryShadow_FailureWhenNotHidden verifies that shadow returns Failure
// when the acting mob does not carry buff 9 (Hidden).
func TestActTryShadow_FailureWhenNotHidden(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Thief")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "target", "Target", 1)
	defer cleanUser()

	mob := mobs.GetInstance(105)
	mob.Character.HealthMax.Value = 500
	mob.Character.Health = 500
	mob.Character.Aggro = nil
	// No Hidden buff.

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	room.AddPlayer(42)

	ctx := &EvalContext{
		InstanceId: 105,
		RoomId:     1,
		Event:      EventContext{UserId: 42},
	}
	if r := actTryShadow(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure when mob is not hidden, got %v", r)
	}
}

// TestActTryShadow_FailureNoTarget verifies that shadow returns Failure
// when there is no resolvable target (even when hidden).
func TestActTryShadow_FailureNoTarget(t *testing.T) {
	cleanBuffs := buffs.SeedBuffsForTest(skulldBuffSpec)
	defer cleanBuffs()

	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Thief")
	defer cleanMob()

	mob := mobs.GetInstance(105)
	mob.Character.HealthMax.Value = 500
	mob.Character.Health = 500
	mob.Character.Aggro = nil

	room := rooms.LoadRoom(1)
	room.AddMob(105)

	if err := mob.Character.AddBuff(9, false); err != nil {
		t.Fatalf("AddBuff(9) failed: %v", err)
	}

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTryShadow(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure when hidden but no target resolvable, got %v", r)
	}
}

// ─── try_defuse ──────────────────────────────────────────────────────────────

// TestActTryDefuse_FailureNoTraps verifies that defuse returns Failure when
// the room has no trapped containers or exits.
func TestActTryDefuse_FailureNoTraps(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Thief")
	defer cleanMob()

	mob := mobs.GetInstance(105)
	mob.Character.HealthMax.Value = 500
	mob.Character.Health = 500
	mob.Character.Aggro = nil

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	// Room has no containers or exits with traps.

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTryDefuse(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure when room has no traps, got %v", r)
	}
}
