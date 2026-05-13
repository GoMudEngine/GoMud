package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestTargetWeakestMobInRoom_EmptyRoom(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Wolf")
	defer cleanMob()
	// Room exists but the wolf is the only mob in it.

	wolf := mobs.GetInstance(105)
	wolf.Character.HealthMax.Value = 1000
	wolf.Character.Stats.Strength.ValueAdj = 200

	room := rooms.LoadRoom(1)
	room.AddMob(105)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTargetWeakestMobInRoom(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure with no other mobs, got %v", r)
	}
}

func TestTargetWeakestMobInRoom_HatedWeakerMob_Success(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMobs := seedTwoMobs(t, 1, 5, 105, "Wolf", 10, 110, "Rat")
	defer cleanMobs()

	wolf := mobs.GetInstance(105)
	rat := mobs.GetInstance(110)

	// Pump wolf, leave rat weaker so PowerScore(rat) < PowerScore(wolf).
	wolf.Character.HealthMax.Value = 1000
	wolf.Character.Health = 1000
	wolf.Character.Stats.Strength.ValueAdj = 200
	rat.Character.HealthMax.Value = 50
	rat.Character.Health = 50 // alive but weak
	// Wire the hates list + groups so wolf.HatesMob(rat) returns true.
	wolf.Hates = []string{"rodent"}
	rat.Groups = []string{"rodent"}

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	room.AddMob(110)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTargetWeakestMobInRoom(map[string]any{}, ctx); r != Success {
		t.Errorf("expected Success picking the rat, got %v", r)
	}
	if wolf.Character.Aggro == nil || wolf.Character.Aggro.MobInstanceId != 110 {
		t.Errorf("expected Aggro set to rat (110), got %+v", wolf.Character.Aggro)
	}
}

func TestTargetWeakestMobInRoom_HatedButStronger_Failure(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMobs := seedTwoMobs(t, 1, 5, 105, "Wolf", 12, 112, "Bear")
	defer cleanMobs()

	wolf := mobs.GetInstance(105)
	bear := mobs.GetInstance(112)

	// Bear stronger than wolf — wolf hates bears but won't engage.
	wolf.Character.HealthMax.Value = 1000
	wolf.Character.Health = 1000
	bear.Character.HealthMax.Value = 5000
	bear.Character.Health = 5000
	bear.Character.Stats.Strength.ValueAdj = 500
	wolf.Hates = []string{"ursine"}
	bear.Groups = []string{"ursine"}

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	room.AddMob(112)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTargetWeakestMobInRoom(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure (target is stronger), got %v", r)
	}
	if wolf.Character.Aggro != nil {
		t.Errorf("expected no Aggro set, got %+v", wolf.Character.Aggro)
	}
}

func TestTargetWeakestMobInRoom_NotHated_Failure(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	// Same template (5) for both — HatesMob returns false on same MobId.
	cleanMobs := seedTwoMobs(t, 1, 5, 105, "Wolf", 5, 106, "OtherWolf")
	defer cleanMobs()

	wolf := mobs.GetInstance(105)
	wolf.Character.HealthMax.Value = 1000
	wolf.Character.Health = 1000
	otherwolf := mobs.GetInstance(106)
	otherwolf.Character.HealthMax.Value = 50
	otherwolf.Character.Health = 50 // alive but same template → not hated

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	room.AddMob(106)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTargetWeakestMobInRoom(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure (same template, HatesMob false), got %v", r)
	}
}

func TestTargetWeakestMobInRoom_DeadMobSkipped(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMobs := seedTwoMobs(t, 1, 5, 105, "Wolf", 10, 110, "DeadRat")
	defer cleanMobs()

	wolf := mobs.GetInstance(105)
	rat := mobs.GetInstance(110)

	wolf.Character.HealthMax.Value = 1000
	wolf.Hates = []string{"rodent"}
	rat.Groups = []string{"rodent"}
	rat.Character.Health = 0 // already dead — skip

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	room.AddMob(110)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTargetWeakestMobInRoom(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure (only candidate is dead), got %v", r)
	}
}

func TestTargetWeakestMobInRoom_RatioBelowCap(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMobs := seedTwoMobs(t, 1, 5, 105, "Wolf", 10, 110, "Rat")
	defer cleanMobs()

	wolf := mobs.GetInstance(105)
	rat := mobs.GetInstance(110)

	// Make the wolf only slightly stronger than the rat — ratio ~0.9.
	// A ratio_below: 0.5 ceiling should reject the engagement.
	wolf.Character.HealthMax.Value = 1100
	wolf.Character.Health = 1100
	rat.Character.HealthMax.Value = 1000
	rat.Character.Health = 1000
	wolf.Hates = []string{"rodent"}
	rat.Groups = []string{"rodent"}

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	room.AddMob(110)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	r := actTargetWeakestMobInRoom(map[string]any{"ratio_below": 0.5}, ctx)
	if r != Failure {
		t.Errorf("expected Failure (target ratio above 0.5 ceiling), got %v", r)
	}
}
