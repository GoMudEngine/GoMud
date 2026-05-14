package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// hiddenBuffSpec is the minimal BuffSpec required to make AddBuff(9) succeed.
// TriggerCount > 0 keeps the buff alive through the assertion.
var hiddenBuffSpec = map[int]*buffs.BuffSpec{
	9: {
		BuffId:        9,
		Name:          "Hidden",
		Flags:         []buffs.Flag{buffs.Hidden},
		TriggerCount:  15,
		RoundInterval: 1,
	},
}

// ─── condMobIsHidden ─────────────────────────────────────────────────────────

func TestCondMobIsHidden_TrueWhenBuffPresent(t *testing.T) {
	cleanBuffs := buffs.SeedBuffsForTest(hiddenBuffSpec)
	defer cleanBuffs()

	cleanMob := seedTestMob(t, 5, 105, 1, "TestThief")
	defer cleanMob()

	mob := mobs.GetInstance(105)
	if err := mob.Character.AddBuff(9, false); err != nil {
		t.Fatalf("AddBuff(9) failed: %v", err)
	}

	ctx := &EvalContext{InstanceId: 105}
	if r := condMobIsHidden(map[string]any{}, ctx); r != Success {
		t.Errorf("expected Success when mob has Hidden buff, got %v", r)
	}
}

func TestCondMobIsHidden_FalseWhenNoBuff(t *testing.T) {
	cleanMob := seedTestMob(t, 5, 105, 1, "TestThief")
	defer cleanMob()

	ctx := &EvalContext{InstanceId: 105}
	if r := condMobIsHidden(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure when mob has no Hidden buff, got %v", r)
	}
}

func TestCondMobIsHidden_FalseWhenInstanceMissing(t *testing.T) {
	ctx := &EvalContext{InstanceId: 9999}
	if r := condMobIsHidden(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure for missing instance, got %v", r)
	}
}

// ─── condTargetIsHidden ──────────────────────────────────────────────────────

func TestCondTargetIsHidden_TrueWhenTargetBuffPresent(t *testing.T) {
	cleanBuffs := buffs.SeedBuffsForTest(hiddenBuffSpec)
	defer cleanBuffs()

	cleanMob := seedTestMob(t, 5, 105, 1, "TestMob")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "sneaky", "Sneaky", 1)
	defer cleanUser()

	user := users.GetByUserId(42)
	if err := user.Character.AddBuff(9, false); err != nil {
		t.Fatalf("AddBuff(9) failed on user: %v", err)
	}

	ctx := &EvalContext{
		InstanceId: 105,
		Event:      EventContext{UserId: 42},
	}
	if r := condTargetIsHidden(map[string]any{}, ctx); r != Success {
		t.Errorf("expected Success when event user has Hidden buff, got %v", r)
	}
}

func TestCondTargetIsHidden_TrueViaSoftTarget(t *testing.T) {
	cleanBuffs := buffs.SeedBuffsForTest(hiddenBuffSpec)
	defer cleanBuffs()

	cleanMob := seedTestMob(t, 5, 105, 1, "TestMob")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "sneaky", "Sneaky", 1)
	defer cleanUser()

	// Set SoftTarget on user 42; no Event.UserId.
	// This exercises the SoftTarget priority path in resolveSkullduggeryTarget
	// (chunk-2.7 fix: target_random_player_in_room sets SoftTarget, not Aggro).
	user := users.GetByUserId(42)
	if err := user.Character.AddBuff(9, false); err != nil {
		t.Fatalf("AddBuff(9) failed on user: %v", err)
	}

	ctx := &EvalContext{
		InstanceId: 105, // no Event.UserId
		SoftTarget: state.ActorRef{UserId: 42},
	}
	if r := condTargetIsHidden(map[string]any{}, ctx); r != Success {
		t.Errorf("expected Success via SoftTarget with Hidden buff, got %v", r)
	}
}

func TestCondTargetIsHidden_FalseWhenNoTarget(t *testing.T) {
	cleanMob := seedTestMob(t, 5, 105, 1, "TestMob")
	defer cleanMob()

	ctx := &EvalContext{InstanceId: 105}
	if r := condTargetIsHidden(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure with no aggro target, got %v", r)
	}
}

func TestCondTargetIsHidden_FalseWhenTargetNotHidden(t *testing.T) {
	cleanMob := seedTestMob(t, 5, 105, 1, "TestMob")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "visible", "Visible", 1)
	defer cleanUser()

	ctx := &EvalContext{
		InstanceId: 105,
		Event:      EventContext{UserId: 42},
	}
	if r := condTargetIsHidden(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure when event user has no Hidden buff, got %v", r)
	}
}

// ─── condTargetHasGold ───────────────────────────────────────────────────────

func TestCondTargetHasGold_TrueAboveThreshold(t *testing.T) {
	cleanMob := seedTestMob(t, 5, 105, 1, "TestMob")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "richplayer", "RichPlayer", 1)
	defer cleanUser()

	user := users.GetByUserId(42)
	user.Character.Gold = 50

	ctx := &EvalContext{
		InstanceId: 105,
		Event:      EventContext{UserId: 42},
	}
	if r := condTargetHasGold(map[string]any{"min": 10}, ctx); r != Success {
		t.Errorf("expected Success (50 gold >= min 10), got %v", r)
	}
}

func TestCondTargetHasGold_TrueAtExactThreshold(t *testing.T) {
	cleanMob := seedTestMob(t, 5, 105, 1, "TestMob")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "player", "Player", 1)
	defer cleanUser()

	user := users.GetByUserId(42)
	user.Character.Gold = 10

	ctx := &EvalContext{
		InstanceId: 105,
		Event:      EventContext{UserId: 42},
	}
	if r := condTargetHasGold(map[string]any{"min": 10}, ctx); r != Success {
		t.Errorf("expected Success (10 gold == min 10), got %v", r)
	}
}

func TestCondTargetHasGold_FalseBelowThreshold(t *testing.T) {
	cleanMob := seedTestMob(t, 5, 105, 1, "TestMob")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "poorplayer", "PoorPlayer", 1)
	defer cleanUser()

	user := users.GetByUserId(42)
	user.Character.Gold = 5

	ctx := &EvalContext{
		InstanceId: 105,
		Event:      EventContext{UserId: 42},
	}
	if r := condTargetHasGold(map[string]any{"min": 10}, ctx); r != Failure {
		t.Errorf("expected Failure (5 gold < min 10), got %v", r)
	}
}

func TestCondTargetHasGold_FalseForMobSoftTarget(t *testing.T) {
	// Two mobs: the checking mob and a target mob.
	cleanMobs := seedTwoMobs(t, 1,
		5, 105, "CheckerMob",
		6, 106, "MerchantMob",
	)
	defer cleanMobs()

	// SoftTarget points at the mob — condTargetHasGold returns Failure
	// because mob gold is not stealable by the thief archetype.
	ctx := &EvalContext{
		InstanceId: 105,
		SoftTarget: state.ActorRef{MobInstanceId: 106},
	}
	if r := condTargetHasGold(map[string]any{"min": 1}, ctx); r != Failure {
		t.Errorf("expected Failure for mob SoftTarget (mob gold not stealable), got %v", r)
	}
}

func TestCondTargetHasGold_FalseNoTarget(t *testing.T) {
	cleanMob := seedTestMob(t, 5, 105, 1, "TestMob")
	defer cleanMob()

	ctx := &EvalContext{InstanceId: 105}
	if r := condTargetHasGold(map[string]any{"min": 1}, ctx); r != Failure {
		t.Errorf("expected Failure with no target, got %v", r)
	}
}
