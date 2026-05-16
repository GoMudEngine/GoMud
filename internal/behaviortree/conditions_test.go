package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// newTestMob seeds a mob at instance 100 and registers it via
// mobs.SetInstanceForTest. Auto-cleans at test end.
func newTestMob(t *testing.T) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{MobId: 1, InstanceId: 100}
	m.Character.Name = "TestMob"
	mobs.SetInstanceForTest(m.InstanceId, m)
	t.Cleanup(func() { mobs.SetInstanceForTest(m.InstanceId, nil) })
	return m
}

func TestCondKeywordMatch_Hit(t *testing.T) {
	fn := LookupCondition("keyword_match")
	if fn == nil {
		t.Fatal("keyword_match not registered")
	}

	ctx := &EvalContext{
		Event: EventContext{Text: "hello world"},
	}
	params := map[string]any{
		"keywords": []any{"world", "foo"},
	}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success, got %v", result)
	}
}

func TestCondKeywordMatch_Miss(t *testing.T) {
	fn := LookupCondition("keyword_match")

	ctx := &EvalContext{
		Event: EventContext{Text: "hello world"},
	}
	params := map[string]any{
		"keywords": []any{"foo", "bar"},
	}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure, got %v", result)
	}
}

func TestCondKeywordMatch_CaseInsensitive(t *testing.T) {
	fn := LookupCondition("keyword_match")

	ctx := &EvalContext{
		Event: EventContext{Text: "Hello WORLD"},
	}
	params := map[string]any{
		"keywords": []any{"world"},
	}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success, got %v", result)
	}
}

func TestCondStateEquals_Match(t *testing.T) {
	fn := LookupCondition("state_equals")
	if fn == nil {
		t.Fatal("state_equals not registered")
	}

	state := NewBehaviorState()
	state.Set("mood", "angry")

	ctx := &EvalContext{
		MobState: state,
	}
	params := map[string]any{
		"key":   "mood",
		"value": "angry",
	}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success, got %v", result)
	}
}

func TestCondStateEquals_Miss(t *testing.T) {
	fn := LookupCondition("state_equals")

	state := NewBehaviorState()
	state.Set("mood", "calm")

	ctx := &EvalContext{
		MobState: state,
	}
	params := map[string]any{
		"key":   "mood",
		"value": "angry",
	}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure, got %v", result)
	}
}

func TestCondStateEquals_NilState(t *testing.T) {
	fn := LookupCondition("state_equals")

	ctx := &EvalContext{}
	params := map[string]any{
		"key":   "mood",
		"value": "angry",
	}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure for nil state, got %v", result)
	}
}

func TestCondRandomChance_AlwaysSucceeds(t *testing.T) {
	fn := LookupCondition("random_chance")
	if fn == nil {
		t.Fatal("random_chance not registered")
	}

	ctx := &EvalContext{}
	params := map[string]any{
		"percent": 100,
	}
	for i := 0; i < 50; i++ {
		if result := fn(params, ctx); result != Success {
			t.Fatalf("100%% chance failed on iteration %d", i)
		}
	}
}

func TestCondRandomChance_AlwaysFails(t *testing.T) {
	fn := LookupCondition("random_chance")

	ctx := &EvalContext{}
	params := map[string]any{
		"percent": 0,
	}
	for i := 0; i < 50; i++ {
		if result := fn(params, ctx); result != Failure {
			t.Fatalf("0%% chance succeeded on iteration %d", i)
		}
	}
}

func TestCondRandomChance_Float64Param(t *testing.T) {
	fn := LookupCondition("random_chance")

	ctx := &EvalContext{}
	// YAML parses numbers as float64
	params := map[string]any{
		"percent": float64(100),
	}
	if result := fn(params, ctx); result != Success {
		t.Error("expected Success with float64(100)")
	}
}

func TestCondItemMatches_Hit(t *testing.T) {
	fn := LookupCondition("item_matches")
	if fn == nil {
		t.Fatal("item_matches not registered")
	}

	ctx := &EvalContext{
		Event: EventContext{ItemId: 42},
	}
	params := map[string]any{
		"item_id": 42,
	}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success, got %v", result)
	}
}

func TestCondItemMatches_Miss(t *testing.T) {
	fn := LookupCondition("item_matches")

	ctx := &EvalContext{
		Event: EventContext{ItemId: 42},
	}
	params := map[string]any{
		"item_id": 99,
	}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure, got %v", result)
	}
}

func TestCondItemMatches_Float64Param(t *testing.T) {
	fn := LookupCondition("item_matches")

	ctx := &EvalContext{
		Event: EventContext{ItemId: 42},
	}
	params := map[string]any{
		"item_id": float64(42),
	}
	if result := fn(params, ctx); result != Success {
		t.Error("expected Success with float64 item_id")
	}
}

func TestCondLookup_AllRegistered(t *testing.T) {
	names := []string{
		"keyword_match", "player_has_quest", "player_missing_quest",
		"player_has_item", "player_has_gold", "player_has_flag",
		"mob_in_combat", "mob_health_below", "mob_at_home",
		"time_of_day", "round_mod", "random_chance",
		"state_equals", "players_in_room", "item_matches",
	}
	for _, name := range names {
		if LookupCondition(name) == nil {
			t.Errorf("condition %q not registered", name)
		}
	}
}

func TestCondLookup_Unknown(t *testing.T) {
	if LookupCondition("nonexistent") != nil {
		t.Error("expected nil for unknown condition")
	}
}

func TestCondStateGreaterThan_Above(t *testing.T) {
	state := NewBehaviorState()
	state.Set("counter", 5)
	ctx := &EvalContext{MobState: state}
	fn := LookupCondition("state_greater_than")
	if fn == nil {
		t.Fatal("state_greater_than not registered")
	}
	result := fn(map[string]any{"key": "counter", "value": 3}, ctx)
	if result != Success {
		t.Error("expected Success when 5 > 3")
	}
}

func TestCondStateGreaterThan_Equal(t *testing.T) {
	state := NewBehaviorState()
	state.Set("counter", 3)
	ctx := &EvalContext{MobState: state}
	fn := LookupCondition("state_greater_than")
	result := fn(map[string]any{"key": "counter", "value": 3}, ctx)
	if result != Failure {
		t.Error("expected Failure when 3 == 3 (not greater)")
	}
}

func TestCondStateGreaterThan_Below(t *testing.T) {
	state := NewBehaviorState()
	state.Set("counter", 1)
	ctx := &EvalContext{MobState: state}
	fn := LookupCondition("state_greater_than")
	result := fn(map[string]any{"key": "counter", "value": 3}, ctx)
	if result != Failure {
		t.Error("expected Failure when 1 < 3")
	}
}

func TestCondStateGreaterThan_NilState(t *testing.T) {
	ctx := &EvalContext{}
	fn := LookupCondition("state_greater_than")
	result := fn(map[string]any{"key": "counter", "value": 0}, ctx)
	if result != Failure {
		t.Error("expected Failure for nil state")
	}
}

func TestCondStateGreaterThan_Float64Param(t *testing.T) {
	state := NewBehaviorState()
	state.Set("counter", 5)
	ctx := &EvalContext{MobState: state}
	fn := LookupCondition("state_greater_than")
	// YAML parses numbers as float64
	result := fn(map[string]any{"key": "counter", "value": float64(3)}, ctx)
	if result != Success {
		t.Error("expected Success with float64 threshold")
	}
}

func TestCondAllNewRegistered(t *testing.T) {
	for _, name := range []string{
		"mob_has_buff", "player_has_spell",
		"player_has_misc_data", "state_greater_than",
		"multiple_enemies",
	} {
		if LookupCondition(name) == nil {
			t.Errorf("condition %q not registered", name)
		}
	}
}

// Phase 4c condition tests

func TestCondCommandMatches_Hit(t *testing.T) {
	fn := LookupCondition("command_matches")
	if fn == nil {
		t.Fatal("command_matches not registered")
	}

	params := map[string]any{"commands": []any{"look", "examine"}}
	ctx := &EvalContext{Event: EventContext{Command: "look"}}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success for command 'look', got %v", result)
	}
}

func TestCondCommandMatches_Miss(t *testing.T) {
	fn := LookupCondition("command_matches")

	params := map[string]any{"commands": []any{"look", "examine"}}
	ctx := &EvalContext{Event: EventContext{Command: "east"}}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure for command 'east', got %v", result)
	}
}

func TestCondCommandMatches_MissingParam(t *testing.T) {
	fn := LookupCondition("command_matches")

	params := map[string]any{}
	ctx := &EvalContext{Event: EventContext{Command: "look"}}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure when 'commands' param absent, got %v", result)
	}
}

func TestCondCommandRestContains_Hit(t *testing.T) {
	fn := LookupCondition("command_rest_contains")
	if fn == nil {
		t.Fatal("command_rest_contains not registered")
	}

	// "chest" in mixed-case rest — verify case-insensitive match
	params := map[string]any{"keywords": []any{"chest"}}
	ctx := &EvalContext{Event: EventContext{Rest: "open the wooden CHEST"}}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success for case-insensitive rest match, got %v", result)
	}
}

func TestCondCommandRestContains_EmptyRest(t *testing.T) {
	fn := LookupCondition("command_rest_contains")

	params := map[string]any{"keywords": []any{"chest"}}
	ctx := &EvalContext{Event: EventContext{Rest: ""}}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure for empty rest, got %v", result)
	}
}

func TestCondMobInRoom_Hit(t *testing.T) {
	fn := LookupCondition("mob_in_room")
	if fn == nil {
		t.Fatal("mob_in_room not registered")
	}

	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Goblin")
	defer cleanMob()

	// seedTestMob registers the instance in the mob registry but does NOT
	// wire it into the room's mob list — add it explicitly.
	room := rooms.LoadRoom(1)
	if room == nil {
		t.Fatal("seedTestRoom did not register room 1")
	}
	room.AddMob(105)

	params := map[string]any{"mob_id": 5}
	ctx := &EvalContext{RoomId: 1}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success when mob template 5 is in room, got %v", result)
	}
}

func TestCondMobInRoom_NoRoom(t *testing.T) {
	fn := LookupCondition("mob_in_room")

	// No room seeded — LoadRoom(99) should return nil; condition must Fail
	// without panicking.
	params := map[string]any{"mob_id": 5}
	ctx := &EvalContext{RoomId: 99}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure when room does not exist, got %v", result)
	}
}

// condMultipleEnemies perspective-awareness tests
//
// seedMultipleEnemiesRoom is a test helper that seeds mobs, users, and a room
// in one shot (avoids the single-entity-per-call limitation of the package
// seedTestMob/seedTestUser helpers). Returns a combined cleanup function.
//
// mobDefs is a map of instanceId → (templateId, name, charmedByUserId).
// userIds is a slice of userIds to place in the room.
func seedMultipleEnemiesRoom(
	t *testing.T,
	roomId int,
	userIds []int,
	mobDefs map[int][3]int, // instanceId → [templateId, charmedByUserId, unused]
	mobNames map[int]string, // instanceId → name
) func() {
	t.Helper()

	// Seed users.
	userMap := map[int]*users.UserRecord{}
	for _, uid := range userIds {
		u := users.NewTestUser(uid, "testuser", "TestUser", uint64(uid))
		u.Character.RoomId = roomId
		userMap[uid] = u
	}
	cleanupUsers := users.SeedUsersForTest(userMap)

	// Seed mobs.
	specs := map[int]*mobs.Mob{}
	instances := map[int]*mobs.Mob{}
	for iid, def := range mobDefs {
		tid := def[0]
		charmedBy := def[1]
		name := mobNames[iid]
		charm := (*characters.CharmInfo)(nil)
		if charmedBy > 0 {
			charm = characters.NewCharm(charmedBy, -1, "")
		}
		specs[tid] = &mobs.Mob{
			MobId: mobs.MobId(tid),
			Character: characters.Character{
				Name:  name,
				RoomId: roomId,
				Buffs: buffs.New(),
			},
		}
		instances[iid] = &mobs.Mob{
			MobId:      mobs.MobId(tid),
			InstanceId: iid,
			HomeRoomId: roomId,
			Character: characters.Character{
				Name:    name,
				RoomId:  roomId,
				Charmed: charm,
				Buffs:   buffs.New(),
			},
		}
	}
	cleanupMobs := mobs.SeedMobsForTest(specs, instances)

	// Seed room.
	r := &rooms.Room{
		RoomId: roomId,
		Zone:   "test",
		Title:  "Test Room",
		Exits:  map[string]exit.RoomExit{},
	}
	cleanupRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{roomId: r},
		map[string]*rooms.ZoneConfig{},
	)

	// Wire players and mobs into the room.
	room := rooms.LoadRoom(roomId)
	for _, uid := range userIds {
		room.AddPlayer(uid)
	}
	for iid := range mobDefs {
		room.AddMob(iid)
	}

	return func() {
		cleanupRooms()
		cleanupMobs()
		cleanupUsers()
	}
}

func TestCondMultipleEnemies_WildMob_CountsPlayersAndCharmed(t *testing.T) {
	// Wild mob (no owner): existing behavior preserved.
	// Room: 1 player + charmed mob 9101 + wild caller 9102.
	// From wild mob's POV: 1 player + 1 charmed companion = 2 → Success.
	defer seedMultipleEnemiesRoom(t, 7501,
		[]int{501},
		map[int][3]int{
			9101: {9101, 501, 0}, // charmed by 501
			9102: {9102, 0, 0},   // wild, this is the caller
		},
		map[int]string{9101: "testvamp", 9102: "testwild"},
	)()

	ctx := &EvalContext{InstanceId: 9102, RoomId: 7501}
	if got := condMultipleEnemies(nil, ctx); got != Success {
		t.Fatalf("wild mob with 1 player + 1 charmed expected Success, got %v", got)
	}
}

func TestCondMultipleEnemies_CharmedMob_SkipsSummonerAndFellows(t *testing.T) {
	// Summoned mob (owner=501). Room: owner + fellow companion 9202 + wild mob
	// 9203. From caster 9201's POV: skip owner (501) + skip fellow (9202).
	// Only wild mob counts → count=1 → Failure.
	defer seedMultipleEnemiesRoom(t, 7502,
		[]int{501},
		map[int][3]int{
			9201: {9201, 501, 0}, // caller, charmed by 501
			9202: {9202, 501, 0}, // fellow companion, charmed by 501
			9203: {9203, 0, 0},   // wild mob
		},
		map[int]string{9201: "caster", 9202: "fellow", 9203: "wild"},
	)()

	ctx := &EvalContext{InstanceId: 9201, RoomId: 7502}
	if got := condMultipleEnemies(nil, ctx); got != Failure {
		t.Fatalf("summoned mob with only 1 wild enemy expected Failure, got %v", got)
	}
}

func TestCondMultipleEnemies_CharmedMob_WildMobsCount(t *testing.T) {
	// Summoned mob (owner=501). Room: owner + 2 wild mobs.
	// From caster 9301's POV: skip owner → 2 wild enemies → Success.
	defer seedMultipleEnemiesRoom(t, 7503,
		[]int{501},
		map[int][3]int{
			9301: {9301, 501, 0}, // caller, charmed by 501
			9302: {9302, 0, 0},   // wild
			9303: {9303, 0, 0},   // wild
		},
		map[int]string{9301: "caster", 9302: "wild1", 9303: "wild2"},
	)()

	ctx := &EvalContext{InstanceId: 9301, RoomId: 7503}
	if got := condMultipleEnemies(nil, ctx); got != Success {
		t.Fatalf("summoned mob with 2 wild enemies expected Success, got %v", got)
	}
}

func TestCondMultipleEnemies_CharmedMob_OtherPlayerHostile(t *testing.T) {
	// Summoned mob (owner=501). Room: owner 501 + other player 502 + wild mob.
	// From caster 9401's POV: skip owner 501; count player 502 + wild → 2
	// → Success.
	defer seedMultipleEnemiesRoom(t, 7504,
		[]int{501, 502},
		map[int][3]int{
			9401: {9401, 501, 0}, // caller, charmed by 501
			9402: {9402, 0, 0},   // wild
		},
		map[int]string{9401: "caster", 9402: "wild"},
	)()

	ctx := &EvalContext{InstanceId: 9401, RoomId: 7504}
	if got := condMultipleEnemies(nil, ctx); got != Success {
		t.Fatalf("charmed mob with hostile player + wild mob expected Success, got %v", got)
	}
}

func TestCondMultipleEnemies_EmptyRoom_Failure(t *testing.T) {
	// Wild mob alone in room → 0 enemies → Failure.
	defer seedMultipleEnemiesRoom(t, 7505,
		[]int{},
		map[int][3]int{
			9501: {9501, 0, 0}, // wild loner, the caller
		},
		map[int]string{9501: "loner"},
	)()

	ctx := &EvalContext{InstanceId: 9501, RoomId: 7505}
	if got := condMultipleEnemies(nil, ctx); got != Failure {
		t.Fatalf("empty-room wild mob expected Failure, got %v", got)
	}
}

// ─── target_is_casting ──────────────────────────────────────────────────────

func TestCondTargetIsCasting_TargetCasting_ReturnsSuccess(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 200}
	target.Character.Name = "Target"
	target.Character.Activity = activity.NewMachine()
	_ = target.Character.Activity.TransitionToCasting(
		activity.CastingData{SpellId: "fireball"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin},
	)
	target.Character.CastingState = &characters.CastingState{} // parallel-write (Task 11 sunsets)
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Success, condTargetIsCasting(nil, ctx))
}

func TestCondTargetIsCasting_TargetNotCasting_ReturnsFailure(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 201}
	target.Character.Name = "Target"
	// CastingState nil = not casting
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condTargetIsCasting(nil, ctx))
}

func TestCondTargetIsCasting_NoTarget_ReturnsFailure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.EndAggro()

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condTargetIsCasting(nil, ctx))
}

// ─── target_aggro_not_on_me ────────────────────────────────────────────────

func TestCondTargetAggroNotOnMe_TargetAggrosMe_ReturnsFailure(t *testing.T) {
	mob := newTestMob(t) // instance 100
	target := &mobs.Mob{InstanceId: 202}
	target.Character.Name = "Target"
	target.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condTargetAggroNotOnMe(nil, ctx),
		"target aggros me → should NOT taunt")
}

func TestCondTargetAggroNotOnMe_TargetAggrosSomeoneElse_ReturnsSuccess(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 203}
	target.Character.Name = "Target"
	target.Character.SetAggro(42, 0, characters.DefaultAttack) // aggros user 42
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Success, condTargetAggroNotOnMe(nil, ctx),
		"target aggros someone else → SHOULD taunt")
}

func TestCondTargetAggroNotOnMe_TargetHasNoAggro_ReturnsSuccess(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 204}
	target.Character.Name = "Target"
	// target.Character.Aggro is nil
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Success, condTargetAggroNotOnMe(nil, ctx))
}

// ─── target_not_standing ───────────────────────────────────────────────────

func TestCondTargetNotStanding_TargetStanding_ReturnsFailure(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 205}
	target.Character.Name = "Target"
	target.Character.CombatPosition = characters.PositionStanding
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condTargetNotStanding(nil, ctx))
}

func TestCondTargetNotStanding_TargetProne_ReturnsSuccess(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 206}
	target.Character.Name = "Target"
	target.Character.CombatPosition = characters.PositionProne
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Success, condTargetNotStanding(nil, ctx))
}

func TestCondTargetNotStanding_TargetClinched_ReturnsSuccess(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 207}
	target.Character.Name = "Target"
	target.Character.CombatPosition = characters.PositionClinched
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Success, condTargetNotStanding(nil, ctx))
}
