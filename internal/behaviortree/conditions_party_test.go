package behaviortree

// conditions_party_test.go — unit tests for the 5 NPC party btree conditions.
//
// Test strategy:
//   - All 5 conditions: "no party → Failure" (validates registration +
//     graceful nil-party handling).
//   - party_member_below_pct: hp-below and hp-full paths.
//   - party_in_combat: any member in combat → Success; none → Failure.
//   - party_leader_in_combat: leader in combat → Success; not → Failure.
//   - party_in_room: all same room → Success; dispersed → Failure.
//   - party_at_home: all at HomeRoomId → Success; away → Failure;
//     HomeRoomId==0 → Failure.
//
// Room-dependent tests (party_in_room, party_at_home) build MobActor values
// with Room set explicitly — MobActor.GetRoom() returns the Room field, which
// is nil when the actor is created without one (see actor_mob.go).

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/stats"
)

// ─── registration check ───────────────────────────────────────────────────────

func TestCondPartyAllRegistered(t *testing.T) {
	for _, name := range []string{
		"party_member_below_pct",
		"party_in_combat",
		"party_leader_in_combat",
		"party_in_room",
		"party_at_home",
		"party_help_inactive",
	} {
		if LookupCondition(name) == nil {
			t.Errorf("condition %q not registered", name)
		}
	}
}

// ─── no-party (graceful Failure) ─────────────────────────────────────────────

func TestCondPartyMemberBelowPct_NoParty(t *testing.T) {
	fn := LookupCondition("party_member_below_pct")
	ctx := &EvalContext{InstanceId: 89001}
	params := map[string]any{"pool": "hp", "percent": 50}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure for caller not in party, got %v", result)
	}
}

func TestCondPartyInCombat_NoParty(t *testing.T) {
	fn := LookupCondition("party_in_combat")
	ctx := &EvalContext{InstanceId: 89002}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure for caller not in party, got %v", result)
	}
}

func TestCondPartyLeaderInCombat_NoParty(t *testing.T) {
	fn := LookupCondition("party_leader_in_combat")
	ctx := &EvalContext{InstanceId: 89003}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure for caller not in party, got %v", result)
	}
}

func TestCondPartyInRoom_NoParty(t *testing.T) {
	fn := LookupCondition("party_in_room")
	ctx := &EvalContext{InstanceId: 89004}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure for caller not in party, got %v", result)
	}
}

func TestCondPartyAtHome_NoParty(t *testing.T) {
	fn := LookupCondition("party_at_home")
	ctx := &EvalContext{InstanceId: 89005}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure for caller not in party, got %v", result)
	}
}

func TestCondPartyHelpInactive_NoParty(t *testing.T) {
	fn := LookupCondition("party_help_inactive")
	ctx := &EvalContext{InstanceId: 89006}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure for caller not in party, got %v", result)
	}
}

// TestCondPartyHelpInactive_HelpRoomZero verifies that the condition succeeds
// when the party has no active help call. This is the gate that lets flee /
// retreat branches fire only when the party isn't currently rallying.
func TestCondPartyHelpInactive_HelpRoomZero(t *testing.T) {
	fn := LookupCondition("party_help_inactive")

	cleanRoom := seedTestRoom(t, 80, "TestZone")
	defer cleanRoom()

	_, p := makePartyMob(t, 9001, 80)
	p.HelpRoomId = 0 // explicit: no active call

	ctx := &EvalContext{InstanceId: 9001, RoomId: 80}
	if result := fn(nil, ctx); result != Success {
		t.Errorf("expected Success when HelpRoomId == 0, got %v", result)
	}
}

// TestCondPartyHelpInactive_HelpRoomSet verifies that the condition fails
// during an active help call — this is what prevents the leader's flee
// branch from triggering a blanket retreat from the rally room.
func TestCondPartyHelpInactive_HelpRoomSet(t *testing.T) {
	fn := LookupCondition("party_help_inactive")

	cleanRoom := seedTestRoom(t, 81, "TestZone")
	defer cleanRoom()

	_, p := makePartyMob(t, 9002, 81)
	p.HelpRoomId = 4043 // active call in progress

	ctx := &EvalContext{InstanceId: 9002, RoomId: 81}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure when HelpRoomId != 0, got %v", result)
	}
}

// ─── party_member_below_pct ───────────────────────────────────────────────────

// makeHPPartyMob builds a mob with the given HP values and returns a party
// containing only that mob as leader+member. Caller is responsible for cleanup.
func makeHPPartyMob(
	t *testing.T,
	instanceId int,
	health int,
	healthMaxVal int,
) (*mobs.Mob, *parties.Party) {
	t.Helper()
	mob := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: instanceId,
		HomeRoomId: 1,
	}
	mob.Character.Name = "HPTestMob"
	mob.Character.Buffs = buffs.New()
	mob.Character.Health = health
	mob.Character.HealthMax = stats.StatInfo{Base: healthMaxVal, Value: healthMaxVal}
	mob.Character.StaminaMax = stats.StatInfo{Base: 100, Value: 100}
	mob.Character.ConvictionMax = stats.StatInfo{Base: 100, Value: 100}
	mob.Character.Stamina = 100
	mob.Character.Conviction = 100
	mobs.SetInstanceForTest(instanceId, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(instanceId, nil) })

	actor := &actions.MobActor{Mob: mob}
	p := parties.NewByActor(actor)
	if p == nil {
		t.Fatal("makeHPPartyMob: NewByActor returned nil")
	}
	t.Cleanup(func() { p.Dissolve("test-cleanup") })
	return mob, p
}

func TestCondPartyMemberBelowPct_HPBelow_Success(t *testing.T) {
	fn := LookupCondition("party_member_below_pct")
	// health=20, max=100 → 20% — below 50%
	_, _ = makeHPPartyMob(t, 8001, 20, 100)
	ctx := &EvalContext{InstanceId: 8001}
	params := map[string]any{"pool": "hp", "percent": 50}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success when hp=20/100 < 50%%, got %v", result)
	}
}

func TestCondPartyMemberBelowPct_HPFull_Failure(t *testing.T) {
	fn := LookupCondition("party_member_below_pct")
	// health=100, max=100 → 100% — not below 50%
	_, _ = makeHPPartyMob(t, 8002, 100, 100)
	ctx := &EvalContext{InstanceId: 8002}
	params := map[string]any{"pool": "hp", "percent": 50}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure when hp=100/100 >= 50%%, got %v", result)
	}
}

func TestCondPartyMemberBelowPct_SPBelow_Success(t *testing.T) {
	fn := LookupCondition("party_member_below_pct")
	mob := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: 8003,
		HomeRoomId: 1,
	}
	mob.Character.Name = "SPTestMob"
	mob.Character.Buffs = buffs.New()
	mob.Character.Health = 100
	mob.Character.HealthMax = stats.StatInfo{Base: 100, Value: 100}
	mob.Character.Stamina = 10
	mob.Character.StaminaMax = stats.StatInfo{Base: 100, Value: 100}
	mob.Character.ConvictionMax = stats.StatInfo{Base: 100, Value: 100}
	mob.Character.Conviction = 100
	mobs.SetInstanceForTest(8003, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(8003, nil) })

	actor := &actions.MobActor{Mob: mob}
	p := parties.NewByActor(actor)
	if p == nil {
		t.Fatal("NewByActor returned nil")
	}
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	ctx := &EvalContext{InstanceId: 8003}
	params := map[string]any{"pool": "sp", "percent": 25}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success when sp=10/100 < 25%%, got %v", result)
	}
}

func TestCondPartyMemberBelowPct_InvalidPool_Failure(t *testing.T) {
	fn := LookupCondition("party_member_below_pct")
	_, _ = makeHPPartyMob(t, 8004, 10, 100)
	ctx := &EvalContext{InstanceId: 8004}
	params := map[string]any{"pool": "invalid", "percent": 50}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure for unknown pool name, got %v", result)
	}
}

func TestCondPartyMemberBelowPct_MissingParams_Failure(t *testing.T) {
	fn := LookupCondition("party_member_below_pct")
	_, _ = makeHPPartyMob(t, 8005, 10, 100)
	ctx := &EvalContext{InstanceId: 8005}
	// percent=0 → guard returns Failure
	params := map[string]any{"pool": "hp", "percent": 0}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure for percent==0 param, got %v", result)
	}
}

// ─── party_in_combat ─────────────────────────────────────────────────────────

func TestCondPartyInCombat_MemberInCombat_Success(t *testing.T) {
	fn := LookupCondition("party_in_combat")

	mob := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: 8010,
		HomeRoomId: 1,
	}
	mob.Character.Name = "CombatMob"
	mob.Character.Buffs = buffs.New()
	mob.Character.Aggro = &characters.Aggro{UserId: 5, Type: characters.DefaultAttack}
	mobs.SetInstanceForTest(8010, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(8010, nil) })

	actor := &actions.MobActor{Mob: mob}
	p := parties.NewByActor(actor)
	if p == nil {
		t.Fatal("NewByActor returned nil")
	}
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	ctx := &EvalContext{InstanceId: 8010}
	if result := fn(nil, ctx); result != Success {
		t.Errorf("expected Success when member has Aggro set, got %v", result)
	}
}

func TestCondPartyInCombat_NoCombat_Failure(t *testing.T) {
	fn := LookupCondition("party_in_combat")

	mob := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: 8011,
		HomeRoomId: 1,
	}
	mob.Character.Name = "PeacefulMob"
	mob.Character.Buffs = buffs.New()
	// Aggro is nil (not in combat)
	mobs.SetInstanceForTest(8011, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(8011, nil) })

	actor := &actions.MobActor{Mob: mob}
	p := parties.NewByActor(actor)
	if p == nil {
		t.Fatal("NewByActor returned nil")
	}
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	ctx := &EvalContext{InstanceId: 8011}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure when no member has Aggro, got %v", result)
	}
}

// ─── party_leader_in_combat ───────────────────────────────────────────────────

func TestCondPartyLeaderInCombat_LeaderInCombat_Success(t *testing.T) {
	fn := LookupCondition("party_leader_in_combat")

	leaderMob := &mobs.Mob{
		MobId:      mobs.MobId(2),
		InstanceId: 8020,
		HomeRoomId: 1,
	}
	leaderMob.Character.Name = "LeaderCombat"
	leaderMob.Character.Buffs = buffs.New()
	leaderMob.Character.Aggro = &characters.Aggro{UserId: 3, Type: characters.DefaultAttack}
	mobs.SetInstanceForTest(8020, leaderMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(8020, nil) })

	leaderActor := &actions.MobActor{Mob: leaderMob}
	p := parties.NewByActor(leaderActor)
	if p == nil {
		t.Fatal("NewByActor returned nil")
	}
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	// Member mob is the caller
	memberMob := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: 8021,
		HomeRoomId: 1,
	}
	memberMob.Character.Name = "FollowerMob"
	memberMob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(8021, memberMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(8021, nil) })

	memberActor := &actions.MobActor{Mob: memberMob}
	p.AddActor(memberActor)
	t.Cleanup(func() { p.RemoveActor(memberActor) })

	ctx := &EvalContext{InstanceId: 8021}
	if result := fn(nil, ctx); result != Success {
		t.Errorf("expected Success when leader has Aggro, got %v", result)
	}
}

func TestCondPartyLeaderInCombat_LeaderNotInCombat_Failure(t *testing.T) {
	fn := LookupCondition("party_leader_in_combat")

	leaderMob := &mobs.Mob{
		MobId:      mobs.MobId(2),
		InstanceId: 8022,
		HomeRoomId: 1,
	}
	leaderMob.Character.Name = "LeaderPeace"
	leaderMob.Character.Buffs = buffs.New()
	// Aggro is nil
	mobs.SetInstanceForTest(8022, leaderMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(8022, nil) })

	leaderActor := &actions.MobActor{Mob: leaderMob}
	p := parties.NewByActor(leaderActor)
	if p == nil {
		t.Fatal("NewByActor returned nil")
	}
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	ctx := &EvalContext{InstanceId: 8022}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure when leader has no Aggro, got %v", result)
	}
}

// ─── party_in_room ────────────────────────────────────────────────────────────

// makeRoomForTest builds a minimal *rooms.Room with the given id and seeds it.
// Returns the room and a cleanup function.
func makeRoomForTest(t *testing.T, roomId int) (*rooms.Room, func()) {
	t.Helper()
	r := &rooms.Room{
		RoomId: roomId,
		Zone:   "TestZone",
		Title:  "Test Room",
		Exits:  map[string]exit.RoomExit{},
	}
	cleanup := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{roomId: r},
		map[string]*rooms.ZoneConfig{},
	)
	// rooms.SeedRoomsForTest registers the room; rooms.LoadRoom returns it.
	loaded := rooms.LoadRoom(roomId)
	if loaded == nil {
		t.Fatalf("makeRoomForTest: LoadRoom(%d) returned nil after seeding", roomId)
	}
	return loaded, cleanup
}

func TestCondPartyInRoom_AllSameRoom_Success(t *testing.T) {
	fn := LookupCondition("party_in_room")

	room, cleanRoom := makeRoomForTest(t, 8100)
	defer cleanRoom()

	// Leader mob with Room set
	leaderMob := &mobs.Mob{
		MobId:      mobs.MobId(2),
		InstanceId: 8030,
		HomeRoomId: 8100,
	}
	leaderMob.Character.Name = "Leader"
	leaderMob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(8030, leaderMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(8030, nil) })

	leaderActor := &actions.MobActor{Mob: leaderMob, Room: room}
	p := parties.NewByActor(leaderActor)
	if p == nil {
		t.Fatal("NewByActor returned nil")
	}
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	// Member mob also in the same room
	memberMob := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: 8031,
		HomeRoomId: 8100,
	}
	memberMob.Character.Name = "Member"
	memberMob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(8031, memberMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(8031, nil) })

	memberActor := &actions.MobActor{Mob: memberMob, Room: room}
	p.AddActor(memberActor)
	t.Cleanup(func() { p.RemoveActor(memberActor) })

	ctx := &EvalContext{InstanceId: 8030}
	if result := fn(nil, ctx); result != Success {
		t.Errorf("expected Success when all members in same room, got %v", result)
	}
}

func TestCondPartyInRoom_MembersDispersed_Failure(t *testing.T) {
	fn := LookupCondition("party_in_room")

	room1, cleanRoom1 := makeRoomForTest(t, 8101)
	defer cleanRoom1()
	room2, cleanRoom2 := makeRoomForTest(t, 8102)
	defer cleanRoom2()

	leaderMob := &mobs.Mob{
		MobId:      mobs.MobId(2),
		InstanceId: 8032,
		HomeRoomId: 8101,
	}
	leaderMob.Character.Name = "Leader"
	leaderMob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(8032, leaderMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(8032, nil) })

	leaderActor := &actions.MobActor{Mob: leaderMob, Room: room1}
	p := parties.NewByActor(leaderActor)
	if p == nil {
		t.Fatal("NewByActor returned nil")
	}
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	// Member in a different room
	memberMob := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: 8033,
		HomeRoomId: 8102,
	}
	memberMob.Character.Name = "Straggler"
	memberMob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(8033, memberMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(8033, nil) })

	memberActor := &actions.MobActor{Mob: memberMob, Room: room2}
	p.AddActor(memberActor)
	t.Cleanup(func() { p.RemoveActor(memberActor) })

	ctx := &EvalContext{InstanceId: 8032}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure when members are in different rooms, got %v", result)
	}
}

func TestCondPartyInRoom_MemberRoomNil_Failure(t *testing.T) {
	fn := LookupCondition("party_in_room")

	// Leader with Room set, member without (nil) — first member's Room is nil
	mob := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: 8034,
		HomeRoomId: 1,
	}
	mob.Character.Name = "NilRoomMob"
	mob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(8034, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(8034, nil) })

	// Actor has nil Room field
	actor := &actions.MobActor{Mob: mob, Room: nil}
	p := parties.NewByActor(actor)
	if p == nil {
		t.Fatal("NewByActor returned nil")
	}
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	ctx := &EvalContext{InstanceId: 8034}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure when first member's room is nil, got %v", result)
	}
}

// ─── party_at_home ────────────────────────────────────────────────────────────

func TestCondPartyAtHome_AllAtHome_Success(t *testing.T) {
	fn := LookupCondition("party_at_home")

	room, cleanRoom := makeRoomForTest(t, 8200)
	defer cleanRoom()

	mob := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: 8040,
		HomeRoomId: 8200,
	}
	mob.Character.Name = "HomeMob"
	mob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(8040, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(8040, nil) })

	actor := &actions.MobActor{Mob: mob, Room: room}
	p := parties.NewByActor(actor)
	if p == nil {
		t.Fatal("NewByActor returned nil")
	}
	p.HomeRoomId = 8200
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	ctx := &EvalContext{InstanceId: 8040}
	if result := fn(nil, ctx); result != Success {
		t.Errorf("expected Success when all members are at HomeRoomId, got %v", result)
	}
}

func TestCondPartyAtHome_MemberAway_Failure(t *testing.T) {
	fn := LookupCondition("party_at_home")

	homeRoom, cleanHome := makeRoomForTest(t, 8201)
	defer cleanHome()
	awayRoom, cleanAway := makeRoomForTest(t, 8202)
	defer cleanAway()

	leaderMob := &mobs.Mob{
		MobId:      mobs.MobId(2),
		InstanceId: 8041,
		HomeRoomId: 8201,
	}
	leaderMob.Character.Name = "HomeLeader"
	leaderMob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(8041, leaderMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(8041, nil) })

	leaderActor := &actions.MobActor{Mob: leaderMob, Room: homeRoom}
	p := parties.NewByActor(leaderActor)
	if p == nil {
		t.Fatal("NewByActor returned nil")
	}
	p.HomeRoomId = 8201
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	memberMob := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: 8042,
		HomeRoomId: 8202,
	}
	memberMob.Character.Name = "Wanderer"
	memberMob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(8042, memberMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(8042, nil) })

	memberActor := &actions.MobActor{Mob: memberMob, Room: awayRoom}
	p.AddActor(memberActor)
	t.Cleanup(func() { p.RemoveActor(memberActor) })

	ctx := &EvalContext{InstanceId: 8041}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure when a member is away from HomeRoomId, got %v", result)
	}
}

func TestCondPartyAtHome_HomeRoomZero_Failure(t *testing.T) {
	fn := LookupCondition("party_at_home")

	mob := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: 8043,
		HomeRoomId: 1,
	}
	mob.Character.Name = "NoHomeMob"
	mob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(8043, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(8043, nil) })

	actor := &actions.MobActor{Mob: mob}
	p := parties.NewByActor(actor)
	if p == nil {
		t.Fatal("NewByActor returned nil")
	}
	p.HomeRoomId = 0 // explicitly unset
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	ctx := &EvalContext{InstanceId: 8043}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure when HomeRoomId==0, got %v", result)
	}
}
