package behaviortree

// actions_party_test.go — unit tests for the 6 NPC party btree actions.
//
// Test strategy:
//   - Each action gets a "no party → Failure" test.
//   - actPartyCallHelp: verify HelpRoomId set + PartyHelpRequested queued.
//   - actPartyAtHomeStand: verify Failure when not at home, Success when at home.
//   - actPartyRespondToHelp: verify Failure when HelpRoomId==0, Success when
//     already in the rally room.
//   - actPartyFollowLeader: verify Failure when leader.GetRoom()==nil.
//   - actPartyAssistTarget: verify Failure when leader not in combat.
//   - actPartyFleeToRoom: verify Failure when no room_id param.
//
// Party setup: actions.MobActor satisfies the partyActor interface in the
// parties package via structural typing, so we use it directly.

import (
	"sync"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// makePartyMob creates a mob instance with the given instanceId in the given
// room, seeds it, creates a party with an MobActor leader, and returns both.
// Cleanup is handled via t.Cleanup.
func makePartyMob(t *testing.T, instanceId int, roomId int) (*mobs.Mob, *parties.Party) {
	t.Helper()
	mob := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: instanceId,
		HomeRoomId: roomId,
	}
	mob.Character.Name = "TestPartyMob"
	mob.Character.RoomId = roomId
	mob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(instanceId, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(instanceId, nil) })

	// Build an actor (MobActor satisfies partyActor structurally).
	actor := &actions.MobActor{Mob: mob}
	p := parties.NewByActor(actor)
	if p == nil {
		t.Fatal("makePartyMob: NewByActor returned nil — instance already in a party?")
	}
	// Clean up the party registration on test end.
	t.Cleanup(func() { p.Dissolve("test-cleanup") })
	return mob, p
}

// capturePartyHelpRequested registers a transient listener that captures
// PartyHelpRequested events. Same cleanup contract as captureInputs.
func capturePartyHelpRequested(t *testing.T) (*[]events.PartyHelpRequested, *sync.Mutex, func()) {
	t.Helper()
	var mu sync.Mutex
	captured := []events.PartyHelpRequested{}
	id := events.RegisterListener(events.PartyHelpRequested{}, func(e events.Event) events.ListenerReturn {
		if ev, ok := e.(events.PartyHelpRequested); ok {
			mu.Lock()
			captured = append(captured, ev)
			mu.Unlock()
		}
		return events.Continue
	})
	return &captured, &mu, func() {
		events.UnregisterListener(events.PartyHelpRequested{}, id)
	}
}

// ─── party_call_help ─────────────────────────────────────────────────────────

func TestActPartyCallHelp_SetsHelpRoomAndFiresEvent(t *testing.T) {
	fn := LookupAction("party_call_help")
	if fn == nil {
		t.Fatal("party_call_help not registered")
	}

	cleanRoom := seedTestRoom(t, 50, "TestZone")
	defer cleanRoom()

	_, p := makePartyMob(t, 5001, 50)

	captured, mu, cleanup := capturePartyHelpRequested(t)
	defer cleanup()

	ctx := &EvalContext{InstanceId: 5001, RoomId: 50}
	if result := fn(nil, ctx); result != Success {
		t.Fatalf("expected Success, got %v", result)
	}
	if p.HelpRoomId != 50 {
		t.Errorf("HelpRoomId = %d, want 50", p.HelpRoomId)
	}

	events.ProcessEvents()

	mu.Lock()
	defer mu.Unlock()
	if len(*captured) == 0 {
		t.Error("expected PartyHelpRequested event, got none")
	}
	ev := (*captured)[0]
	if ev.RallyRoomId != 50 {
		t.Errorf("RallyRoomId = %d, want 50", ev.RallyRoomId)
	}
	if ev.CallerActorId != 5001 {
		t.Errorf("CallerActorId = %d, want 5001", ev.CallerActorId)
	}
}

func TestActPartyCallHelp_NoPartyReturnsFailure(t *testing.T) {
	fn := LookupAction("party_call_help")
	ctx := &EvalContext{InstanceId: 99990, RoomId: 1}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure for caller not in party, got %v", result)
	}
}

// ─── party_at_home_stand ─────────────────────────────────────────────────────

func TestActPartyAtHomeStand_SuccessAtHomeRoom(t *testing.T) {
	fn := LookupAction("party_at_home_stand")
	if fn == nil {
		t.Fatal("party_at_home_stand not registered")
	}

	cleanRoom := seedTestRoom(t, 51, "TestZone")
	defer cleanRoom()

	_, p := makePartyMob(t, 5002, 51)
	p.HomeRoomId = 51

	state := NewBehaviorState()
	ctx := &EvalContext{InstanceId: 5002, RoomId: 51, MobState: state}
	if result := fn(nil, ctx); result != Success {
		t.Fatalf("expected Success at home room, got %v", result)
	}
	if state.GetString("party_standing") != "true" {
		t.Errorf("expected party_standing='true', got %q", state.GetString("party_standing"))
	}
}

func TestActPartyAtHomeStand_FailureWhenNotAtHome(t *testing.T) {
	fn := LookupAction("party_at_home_stand")

	cleanRoom := seedTestRoom(t, 51, "TestZone")
	defer cleanRoom()
	cleanRoom2 := seedTestRoom(t, 52, "TestZone")
	defer cleanRoom2()

	_, p := makePartyMob(t, 5003, 51)
	p.HomeRoomId = 51

	// Caller is in room 52, home is 51 → Failure
	ctx := &EvalContext{InstanceId: 5003, RoomId: 52}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure when not at home room, got %v", result)
	}
}

func TestActPartyAtHomeStand_NoPartyReturnsFailure(t *testing.T) {
	fn := LookupAction("party_at_home_stand")
	ctx := &EvalContext{InstanceId: 99991, RoomId: 1}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure for caller not in party, got %v", result)
	}
}

func TestActPartyAtHomeStand_HomeRoomZeroReturnsFailure(t *testing.T) {
	fn := LookupAction("party_at_home_stand")

	cleanRoom := seedTestRoom(t, 51, "TestZone")
	defer cleanRoom()

	_, p := makePartyMob(t, 5004, 51)
	p.HomeRoomId = 0 // explicitly no home

	ctx := &EvalContext{InstanceId: 5004, RoomId: 51}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure when HomeRoomId==0, got %v", result)
	}
}

// ─── party_respond_to_help ───────────────────────────────────────────────────

func TestActPartyRespondToHelp_NoPartyReturnsFailure(t *testing.T) {
	fn := LookupAction("party_respond_to_help")
	if fn == nil {
		t.Fatal("party_respond_to_help not registered")
	}
	ctx := &EvalContext{InstanceId: 99992, RoomId: 1}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure for caller not in party, got %v", result)
	}
}

func TestActPartyRespondToHelp_NoHelpRoomReturnsFailure(t *testing.T) {
	fn := LookupAction("party_respond_to_help")

	cleanRoom := seedTestRoom(t, 53, "TestZone")
	defer cleanRoom()

	_, p := makePartyMob(t, 5005, 53)
	p.HelpRoomId = 0 // no active call

	ctx := &EvalContext{InstanceId: 5005, RoomId: 53}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure when HelpRoomId==0, got %v", result)
	}
}

func TestActPartyRespondToHelp_AlreadyAtHelpRoomReturnsSuccess(t *testing.T) {
	fn := LookupAction("party_respond_to_help")

	cleanRoom := seedTestRoom(t, 54, "TestZone")
	defer cleanRoom()

	_, p := makePartyMob(t, 5006, 54)
	p.HelpRoomId = 54 // caller is already at the rally room

	ctx := &EvalContext{InstanceId: 5006, RoomId: 54}
	if result := fn(nil, ctx); result != Success {
		t.Errorf("expected Success when already at HelpRoomId, got %v", result)
	}
}

// ─── party_follow_leader ─────────────────────────────────────────────────────

func TestActPartyFollowLeader_NoPartyReturnsFailure(t *testing.T) {
	fn := LookupAction("party_follow_leader")
	if fn == nil {
		t.Fatal("party_follow_leader not registered")
	}
	ctx := &EvalContext{InstanceId: 99993, RoomId: 1}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure for caller not in party, got %v", result)
	}
}

func TestActPartyFollowLeader_LeaderRoomNilReturnsFailure(t *testing.T) {
	fn := LookupAction("party_follow_leader")

	cleanRoom := seedTestRoom(t, 55, "TestZone")
	defer cleanRoom()

	_, _ = makePartyMob(t, 5007, 55)
	// The MobActor leader has Room==nil (makePartyMob uses MobActor{Mob:mob}
	// without a room). GetRoom() returns nil → Failure.

	ctx := &EvalContext{InstanceId: 5007, RoomId: 55}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure when leader.GetRoom()==nil, got %v", result)
	}
}

func TestActPartyFollowLeader_AlreadyWithLeaderReturnsSuccess(t *testing.T) {
	fn := LookupAction("party_follow_leader")

	// Create a room and a leader actor that returns it from GetRoom().
	cleanRoom := seedTestRoom(t, 56, "TestZone")
	defer cleanRoom()

	leaderMob := &mobs.Mob{
		MobId:      mobs.MobId(2),
		InstanceId: 5100,
		HomeRoomId: 56,
	}
	leaderMob.Character.Name = "Leader"
	leaderMob.Character.RoomId = 56
	leaderMob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(5100, leaderMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(5100, nil) })

	leaderRoom := rooms.LoadRoom(56)
	leaderActor := &actions.MobActor{Mob: leaderMob, Room: leaderRoom}

	// Seed the member mob (instance 5008, also in room 56).
	memberMob := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: 5008,
		HomeRoomId: 56,
	}
	memberMob.Character.Name = "Member"
	memberMob.Character.RoomId = 56
	memberMob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(5008, memberMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(5008, nil) })

	// Build party: leader creates it, member joins.
	p := parties.NewByActor(leaderActor)
	if p == nil {
		t.Fatal("NewByActor returned nil for leaderActor")
	}
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	memberActor := &actions.MobActor{Mob: memberMob}
	p.AddActor(memberActor)
	t.Cleanup(func() { p.RemoveActor(memberActor) })

	// Member is already in the same room as leader → Success without moving.
	ctx := &EvalContext{InstanceId: 5008, RoomId: 56}
	if result := fn(nil, ctx); result != Success {
		t.Errorf("expected Success when member already in leader's room, got %v", result)
	}
}

// ─── party_assist_target ─────────────────────────────────────────────────────

func TestActPartyAssistTarget_NoPartyReturnsFailure(t *testing.T) {
	fn := LookupAction("party_assist_target")
	if fn == nil {
		t.Fatal("party_assist_target not registered")
	}
	ctx := &EvalContext{InstanceId: 99994, RoomId: 1}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure for caller not in party, got %v", result)
	}
}

func TestActPartyAssistTarget_LeaderNotInCombatReturnsFailure(t *testing.T) {
	fn := LookupAction("party_assist_target")

	cleanRoom := seedTestRoom(t, 57, "TestZone")
	defer cleanRoom()

	_, p := makePartyMob(t, 5009, 57)
	// Leader's Aggro is nil (not in combat).
	if p.Leader.GetCharacter() != nil {
		p.Leader.GetCharacter().Aggro = nil
	}

	ctx := &EvalContext{InstanceId: 5009, RoomId: 57}
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure when leader not in combat, got %v", result)
	}
}

func TestActPartyAssistTarget_CopiesLeaderAggro(t *testing.T) {
	fn := LookupAction("party_assist_target")

	cleanRoom := seedTestRoom(t, 58, "TestZone")
	defer cleanRoom()

	// Leader mob (instance 5200).
	leaderMob := &mobs.Mob{
		MobId:      mobs.MobId(2),
		InstanceId: 5200,
		HomeRoomId: 58,
	}
	leaderMob.Character.Name = "Leader"
	leaderMob.Character.RoomId = 58
	leaderMob.Character.Buffs = buffs.New()
	// Leader is attacking user 7.
	leaderMob.Character.Aggro = &characters.Aggro{
		UserId: 7,
		Type:   characters.DefaultAttack,
	}
	mobs.SetInstanceForTest(5200, leaderMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(5200, nil) })

	leaderActor := &actions.MobActor{Mob: leaderMob}
	p := parties.NewByActor(leaderActor)
	if p == nil {
		t.Fatal("NewByActor returned nil for leaderActor")
	}
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	// Member mob (instance 5010).
	memberMob := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: 5010,
		HomeRoomId: 58,
	}
	memberMob.Character.Name = "Member"
	memberMob.Character.RoomId = 58
	memberMob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(5010, memberMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(5010, nil) })

	memberActor := &actions.MobActor{Mob: memberMob}
	p.AddActor(memberActor)
	t.Cleanup(func() { p.RemoveActor(memberActor) })

	ctx := &EvalContext{InstanceId: 5010, RoomId: 58}
	if result := fn(nil, ctx); result != Success {
		t.Fatalf("expected Success, got %v", result)
	}
	if memberMob.Character.Aggro == nil {
		t.Fatal("expected member Aggro to be set, got nil")
	}
	if memberMob.Character.Aggro.UserId != 7 {
		t.Errorf("expected member Aggro.UserId=7, got %d", memberMob.Character.Aggro.UserId)
	}
}

// ─── party_flee_to_room ───────────────────────────────────────────────────────

func TestActPartyFleeToRoom_NoPartyReturnsFailure(t *testing.T) {
	fn := LookupAction("party_flee_to_room")
	if fn == nil {
		t.Fatal("party_flee_to_room not registered")
	}
	ctx := &EvalContext{InstanceId: 99995, RoomId: 1}
	if result := fn(map[string]any{"room_id": 1}, ctx); result != Failure {
		t.Errorf("expected Failure for caller not in party, got %v", result)
	}
}

func TestActPartyFleeToRoom_NoRoomIdReturnsFailure(t *testing.T) {
	fn := LookupAction("party_flee_to_room")

	cleanRoom := seedTestRoom(t, 59, "TestZone")
	defer cleanRoom()

	_, _ = makePartyMob(t, 5011, 59)

	ctx := &EvalContext{InstanceId: 5011, RoomId: 59}
	if result := fn(map[string]any{}, ctx); result != Failure {
		t.Errorf("expected Failure when room_id not provided, got %v", result)
	}
}

// ─── party_ensure_npc_party ───────────────────────────────────────────────────

func TestActPartyEnsureNpcParty_AlreadyInPartyReturnsSuccess(t *testing.T) {
	fn := LookupAction("party_ensure_npc_party")
	if fn == nil {
		t.Fatal("party_ensure_npc_party not registered")
	}

	cleanRoom := seedTestRoom(t, 60, "TestZone")
	defer cleanRoom()

	// makePartyMob seeds the mob AND creates a party for it.
	mob, p := makePartyMob(t, 6001, 60)
	_ = mob

	params := map[string]any{"leader_mob_id": 286, "home_room_id": 4052}
	ctx := &EvalContext{InstanceId: 6001, RoomId: 60}
	if result := fn(params, ctx); result != Success {
		t.Fatalf("expected Success when already in party, got %v", result)
	}
	// Party should be unchanged — still only the one member.
	if len(p.Members) != 1 {
		t.Errorf("expected party to still have 1 member, got %d", len(p.Members))
	}
}

func TestActPartyEnsureNpcParty_MissingParamsReturnsFailure(t *testing.T) {
	fn := LookupAction("party_ensure_npc_party")

	cleanRoom := seedTestRoom(t, 61, "TestZone")
	defer cleanRoom()

	// Seed a mob instance that is NOT yet in any party.
	mob := &mobs.Mob{MobId: mobs.MobId(1), InstanceId: 6002, HomeRoomId: 61}
	mob.Character.Name = "TestEnsure"
	mob.Character.RoomId = 61
	mob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(6002, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(6002, nil) })

	ctx := &EvalContext{InstanceId: 6002, RoomId: 61}

	// Missing both params.
	if result := fn(map[string]any{}, ctx); result != Failure {
		t.Errorf("expected Failure with no params, got %v", result)
	}
	// Missing home_room_id.
	if result := fn(map[string]any{"leader_mob_id": 286}, ctx); result != Failure {
		t.Errorf("expected Failure when home_room_id missing, got %v", result)
	}
	// Missing leader_mob_id.
	if result := fn(map[string]any{"home_room_id": 4052}, ctx); result != Failure {
		t.Errorf("expected Failure when leader_mob_id missing, got %v", result)
	}
}

func TestActPartyEnsureNpcParty_NoLeaderSpawnedCallerBecomesLeader(t *testing.T) {
	fn := LookupAction("party_ensure_npc_party")

	cleanRoom := seedTestRoom(t, 62, "TestZone")
	defer cleanRoom()

	// Seed the calling mob — template 285 (not 286 leader). Leader 286 not loaded.
	mob := &mobs.Mob{MobId: mobs.MobId(285), InstanceId: 6003, HomeRoomId: 62}
	mob.Character.Name = "bandit caster"
	mob.Character.RoomId = 62
	mob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(6003, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(6003, nil) })

	params := map[string]any{"leader_mob_id": 286, "home_room_id": 4052}
	ctx := &EvalContext{InstanceId: 6003, RoomId: 62}

	result := fn(params, ctx)
	if result != Success {
		t.Fatalf("expected Success when leader not spawned, got %v", result)
	}

	// Caller should now be in a party.
	p := parties.GetByMobInstanceId(6003)
	if p == nil {
		t.Fatal("expected caller to be in a party after ensure, got nil")
	}
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	if p.HomeRoomId != 4052 {
		t.Errorf("expected HomeRoomId=4052, got %d", p.HomeRoomId)
	}
}

func TestActPartyEnsureNpcParty_LeaderSpawnedFirstCreatesParty(t *testing.T) {
	fn := LookupAction("party_ensure_npc_party")

	cleanRoom := seedTestRoom(t, 63, "TestZone")
	defer cleanRoom()

	// Seed leader mob (template 286, instance 6100) — not yet in any party.
	leaderMob := &mobs.Mob{MobId: mobs.MobId(286), InstanceId: 6100, HomeRoomId: 63}
	leaderMob.Character.Name = "Soren"
	leaderMob.Character.RoomId = 63
	leaderMob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(6100, leaderMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(6100, nil) })

	// Seed member mob (template 284, instance 6004) — not yet in any party.
	memberMob := &mobs.Mob{MobId: mobs.MobId(284), InstanceId: 6004, HomeRoomId: 63}
	memberMob.Character.Name = "bandit fighter"
	memberMob.Character.RoomId = 63
	memberMob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(6004, memberMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(6004, nil) })

	params := map[string]any{"leader_mob_id": 286, "home_room_id": 4052}
	ctx := &EvalContext{InstanceId: 6004, RoomId: 63}

	result := fn(params, ctx)
	if result != Success {
		t.Fatalf("expected Success when leader exists, got %v", result)
	}

	// A party should exist and the member should be in it.
	p := parties.GetByMobInstanceId(6004)
	if p == nil {
		t.Fatal("expected member to be in a party after ensure, got nil")
	}
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	if p.HomeRoomId != 4052 {
		t.Errorf("expected HomeRoomId=4052, got %d", p.HomeRoomId)
	}
	// Leader should also be in this party.
	leaderParty := parties.GetByMobInstanceId(6100)
	if leaderParty == nil {
		t.Fatal("expected leader to be in a party after ensure, got nil")
	}
	if leaderParty != p {
		t.Error("expected leader and member to be in the same party")
	}
}

func TestActPartyEnsureNpcParty_LeaderCallsEnsureOnSelf(t *testing.T) {
	fn := LookupAction("party_ensure_npc_party")

	cleanRoom := seedTestRoom(t, 64, "TestZone")
	defer cleanRoom()

	// Leader mob calls ensure with its own template ID as leader_mob_id.
	leaderMob := &mobs.Mob{MobId: mobs.MobId(286), InstanceId: 6005, HomeRoomId: 64}
	leaderMob.Character.Name = "Soren"
	leaderMob.Character.RoomId = 64
	leaderMob.Character.Buffs = buffs.New()
	mobs.SetInstanceForTest(6005, leaderMob)
	t.Cleanup(func() { mobs.SetInstanceForTest(6005, nil) })

	params := map[string]any{"leader_mob_id": 286, "home_room_id": 4052}
	ctx := &EvalContext{InstanceId: 6005, RoomId: 64}

	result := fn(params, ctx)
	if result != Success {
		t.Fatalf("expected Success when leader calls ensure on self, got %v", result)
	}

	p := parties.GetByMobInstanceId(6005)
	if p == nil {
		t.Fatal("expected leader to be in a party after self-ensure, got nil")
	}
	t.Cleanup(func() { p.Dissolve("test-cleanup") })

	if p.HomeRoomId != 4052 {
		t.Errorf("expected HomeRoomId=4052, got %d", p.HomeRoomId)
	}
	// Should have exactly 1 member (the leader itself).
	if len(p.Members) != 1 {
		t.Errorf("expected 1 party member (leader only), got %d", len(p.Members))
	}
}

// ─── bandit btree YAML load validation ───────────────────────────────────────

// TestBanditBtreeYAMLLoads verifies that all four bandit behavior-tree YAML
// files parse and compile without error. This catches YAML syntax errors or
// unknown action/condition names before a server-boot regression reaches prod.
func TestBanditBtreeYAMLLoads(t *testing.T) {
	files := []struct {
		mobId int
		path  string
	}{
		{283, "../../_datafiles/world/dogmud/behaviors/north_road/283-bandit_lookout.yaml"},
		{284, "../../_datafiles/world/dogmud/behaviors/north_road/284-bandit_fighter.yaml"},
		{285, "../../_datafiles/world/dogmud/behaviors/north_road/285-bandit_caster.yaml"},
		{286, "../../_datafiles/world/dogmud/behaviors/north_road/286-soren.yaml"},
	}
	for _, f := range files {
		f := f
		t.Run(f.path, func(t *testing.T) {
			node, err := LoadTreeFromFile(f.path)
			if err != nil {
				t.Fatalf("LoadTreeFromFile(%s): %v", f.path, err)
			}
			if node == nil {
				t.Fatalf("LoadTreeFromFile(%s): returned nil node", f.path)
			}
		})
	}
}
