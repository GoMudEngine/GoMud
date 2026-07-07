package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// TestActCast_NoTarget_EmitsPlainCast verifies backward compatibility: a
// cast node with no target param still emits "cast <spell>" (no trailing
// target token).
func TestActCast_NoTarget_EmitsPlainCast(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Repair Frame")
	defer cleanMob()
	defer events.DrainQueuedInputsForTest(105)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actCast(map[string]any{"spell": "sparks"}, ctx); r != Success {
		t.Fatalf("expected Success, got %v", r)
	}

	cmd := events.InspectQueuedInputForTest(105, "cast ")
	if cmd != "cast sparks" {
		t.Errorf(`expected queued command "cast sparks", got %q`, cmd)
	}
}

// TestActCast_WithTarget_EmitsCastWithTargetName verifies the Task A6
// target passthrough: a cast node with a target param emits
// "cast <spell> <target>" so a mob (e.g. a Repair Frame add) can heal
// another named mob (the boss) instead of itself.
func TestActCast_WithTarget_EmitsCastWithTargetName(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Repair Frame")
	defer cleanMob()
	defer events.DrainQueuedInputsForTest(105)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	r := actCast(map[string]any{"spell": "mend", "target": "the_core_guardian"}, ctx)
	if r != Success {
		t.Fatalf("expected Success, got %v", r)
	}

	cmd := events.InspectQueuedInputForTest(105, "cast ")
	if cmd != "cast mend the_core_guardian" {
		t.Errorf(`expected queued command "cast mend the_core_guardian", got %q`, cmd)
	}
}

// TestActCast_NoMobInstance_Failure verifies actCast fails gracefully when
// the acting mob instance can't be resolved.
func TestActCast_NoMobInstance_Failure(t *testing.T) {
	ctx := &EvalContext{InstanceId: 99999, RoomId: 1}
	if r := actCast(map[string]any{"spell": "sparks", "target": "boss"}, ctx); r != Failure {
		t.Errorf("expected Failure with no mob instance, got %v", r)
	}
}

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

func TestActTargetRandomPlayerInRoom_PicksAPlayer(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Thief")
	defer cleanMob()
	cleanUser1 := seedTestUser(t, 1, "alice", "Alice", 1)
	defer cleanUser1()
	cleanUser2 := seedTestUser(t, 2, "bob", "Bob", 1)
	defer cleanUser2()

	thief := mobs.GetInstance(105)
	thief.Character.HealthMax.Value = 500
	thief.Character.Health = 500

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	room.AddPlayer(1)
	room.AddPlayer(2)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTargetRandomPlayerInRoom(map[string]any{}, ctx); r != Success {
		t.Errorf("expected Success with players present, got %v", r)
	}
	// SoftTarget must be set — NOT Aggro. Chunk 2.7 fix: picking a target
	// for skullduggery must not silently engage combat.
	if ctx.SoftTarget.IsZero() {
		t.Fatal("expected SoftTarget to be set, got zero value")
	}
	pickedId := ctx.SoftTarget.UserId
	if pickedId != 1 && pickedId != 2 {
		t.Errorf("expected SoftTarget.UserId to be 1 or 2, got %d", pickedId)
	}
	// Combat must NOT be engaged — Aggro must remain nil.
	if thief.Character.Aggro != nil {
		t.Errorf("expected Aggro to remain nil (no combat engagement), got %+v", thief.Character.Aggro)
	}
}

func TestActTargetRandomPlayerInRoom_EmptyRoom(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Thief")
	defer cleanMob()

	thief := mobs.GetInstance(105)
	thief.Character.HealthMax.Value = 500
	thief.Character.Health = 500

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	// No players added.

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTargetRandomPlayerInRoom(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure with no players, got %v", r)
	}
	if !ctx.SoftTarget.IsZero() {
		t.Errorf("expected SoftTarget to remain zero, got %+v", ctx.SoftTarget)
	}
	if thief.Character.Aggro != nil {
		t.Errorf("expected Aggro to remain nil, got %+v", thief.Character.Aggro)
	}
}
