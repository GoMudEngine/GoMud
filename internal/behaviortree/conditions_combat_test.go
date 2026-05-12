package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestTargetPowerRatioAbove_MissingValue(t *testing.T) {
	cleanMob := seedTestMob(t, 5, 105, 1, "TestMob")
	defer cleanMob()

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := condTargetPowerRatioAbove(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure on missing value param, got %v", r)
	}
}

func TestTargetPowerRatioAbove_NoTargetResolvable(t *testing.T) {
	cleanMob := seedTestMob(t, 5, 105, 1, "TestMob")
	defer cleanMob()

	// No Event.UserId, no Aggro — nothing to compare against.
	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := condTargetPowerRatioAbove(map[string]any{"value": 1.0}, ctx); r != Failure {
		t.Errorf("expected Failure with no resolvable target, got %v", r)
	}
}

func TestTargetPowerRatioAbove_StrongMobVsWeakPlayer(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "StrongMob")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "weakling", "Weakling", 1)
	defer cleanUser()

	// Pump mob stats + durability to dominate the user's NewTestUser
	// defaults (ValueAdj=100, HealthMax=100).
	mob := mobs.GetInstance(105)
	mob.Character.HealthMax.Value = 5000
	mob.Character.StaminaMax.Value = 5000
	mob.Character.ConvictionMax.Value = 2500
	mob.Character.Stats.Strength.ValueAdj = 500
	mob.Character.Stats.Dexterity.ValueAdj = 500
	mob.Character.Stats.Willpower.ValueAdj = 500
	mob.Character.Stats.Charisma.ValueAdj = 500

	ctx := &EvalContext{
		InstanceId: 105, RoomId: 1,
		Event: EventContext{UserId: 42},
	}
	if r := condTargetPowerRatioAbove(map[string]any{"value": 1.0}, ctx); r != Success {
		t.Errorf("expected Success (strong mob > weak player), got %v", r)
	}
	if r := condTargetPowerRatioBelow(map[string]any{"value": 1.0}, ctx); r != Failure {
		t.Errorf("expected Failure for below-1.0 when self stronger, got %v", r)
	}
}

func TestTargetPowerRatioBelow_WeakMobVsStrongPlayer(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "WeakMob")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "champion", "Champion", 1)
	defer cleanUser()

	// User already has ValueAdj=100 + HealthMax=100 from NewTestUser.
	// Mob is fresh (no stats, no health). User dominates.
	ctx := &EvalContext{
		InstanceId: 105, RoomId: 1,
		Event: EventContext{UserId: 42},
	}
	if r := condTargetPowerRatioBelow(map[string]any{"value": 1.0}, ctx); r != Success {
		t.Errorf("expected Success (weak mob < strong player), got %v", r)
	}
	if r := condTargetPowerRatioAbove(map[string]any{"value": 1.0}, ctx); r != Failure {
		t.Errorf("expected Failure for above-1.0 when self weaker, got %v", r)
	}
}

func TestTargetPowerRatio_AggroFallback(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "StrongMob")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "weakling", "Weakling", 1)
	defer cleanUser()

	mob := mobs.GetInstance(105)
	mob.Character.HealthMax.Value = 5000
	mob.Character.Stats.Strength.ValueAdj = 500
	// Aggro on user 42 — no Event.UserId set; condition must fall back.
	mob.Character.Aggro = &characters.Aggro{UserId: 42}

	ctx := &EvalContext{InstanceId: 105, RoomId: 1} // no Event.UserId
	if r := condTargetPowerRatioAbove(map[string]any{"value": 1.0}, ctx); r != Success {
		t.Errorf("expected Success via Aggro fallback, got %v", r)
	}
}
