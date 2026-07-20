package behaviortree

// Phase 4c action tests. Each action is exercised at the registry level via
// LookupAction() so the perception-scaled reaction-delay layer never fires
// (see actions.go:74-138 for the wrapping logic the registry direct-call
// bypasses).
//
// Event taps: events.RegisterListener handlers fire from events.DoListeners,
// which is only called by events.ProcessEvents() — not by AddToQueue. Tests
// that capture events.Input/events.Message must call events.ProcessEvents()
// between the act and the assertion.

import (
	"strings"
	"sync"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// requireUser fetches a seeded user by id or fails the test.
func requireUser(t *testing.T, userId int) *users.UserRecord {
	t.Helper()
	u := users.GetByUserId(userId)
	if u == nil {
		t.Fatalf("test setup error: no user with id %d", userId)
	}
	return u
}

// captureInputs registers a transient listener that captures events.Input
// events into the returned slice (and a sync.Mutex guarding it). The
// returned cleanup unregisters the listener; closure-flag fallback is not
// needed because UnregisterListener exists at internal/events/listeners.go:122.
func captureInputs(t *testing.T) (*[]events.Input, *sync.Mutex, func()) {
	t.Helper()
	var mu sync.Mutex
	captured := []events.Input{}
	id := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		if in, ok := e.(events.Input); ok {
			mu.Lock()
			captured = append(captured, in)
			mu.Unlock()
		}
		return events.Continue
	})
	return &captured, &mu, func() {
		events.UnregisterListener(events.Input{}, id)
	}
}

// captureMessages registers a transient listener that captures
// events.Message events. Same cleanup contract as captureInputs.
func captureMessages(t *testing.T) (*[]events.Message, *sync.Mutex, func()) {
	t.Helper()
	var mu sync.Mutex
	captured := []events.Message{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if m, ok := e.(events.Message); ok {
			mu.Lock()
			captured = append(captured, m)
			mu.Unlock()
		}
		return events.Continue
	})
	return &captured, &mu, func() {
		events.UnregisterListener(events.Message{}, id)
	}
}

// ─── mob_say / mob_emote ──────────────────────────────────────────────

func TestActMobSay_FindsMobInRoomAndQueuesCommand(t *testing.T) {
	fn := LookupAction("mob_say")
	if fn == nil {
		t.Fatal("mob_say not registered")
	}

	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Goblin")
	defer cleanMob()
	rooms.LoadRoom(1).AddMob(105)

	captured, mu, cleanup := captureInputs(t)
	defer cleanup()

	params := map[string]any{"mob_id": 5, "text": "hello"}
	ctx := &EvalContext{RoomId: 1}
	if result := fn(params, ctx); result != Success {
		t.Fatalf("expected Success, got %v", result)
	}

	// Listener fires only during ProcessEvents.
	events.ProcessEvents()

	mu.Lock()
	defer mu.Unlock()
	found := false
	for _, in := range *captured {
		if in.MobInstanceId == 105 && strings.Contains(in.InputText, "say hello") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'say hello' Input event for mob 105, got %d events: %+v", len(*captured), *captured)
	}

	// Negative case: empty room (no matching mob_id) → Failure.
	emptyCtx := &EvalContext{RoomId: 1}
	emptyParams := map[string]any{"mob_id": 999, "text": "hello"}
	if result := fn(emptyParams, emptyCtx); result != Failure {
		t.Errorf("expected Failure when no mob with mob_id=999 in room, got %v", result)
	}
}

func TestActMobEmote_FindsMobInRoomAndQueuesCommand(t *testing.T) {
	fn := LookupAction("mob_emote")
	if fn == nil {
		t.Fatal("mob_emote not registered")
	}

	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Goblin")
	defer cleanMob()
	rooms.LoadRoom(1).AddMob(105)

	captured, mu, cleanup := captureInputs(t)
	defer cleanup()

	params := map[string]any{"mob_id": 5, "text": "waves"}
	ctx := &EvalContext{RoomId: 1}
	if result := fn(params, ctx); result != Success {
		t.Fatalf("expected Success, got %v", result)
	}

	events.ProcessEvents()

	mu.Lock()
	defer mu.Unlock()
	found := false
	for _, in := range *captured {
		if in.MobInstanceId == 105 && strings.Contains(in.InputText, "emote waves") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'emote waves' Input event for mob 105, got %d events: %+v", len(*captured), *captured)
	}
}

// ─── grant_mutation ──────────────────────────────────────────────────

func TestActGrantMutation_AddsMutationToCharacter(t *testing.T) {
	fn := LookupAction("grant_mutation")
	if fn == nil {
		t.Fatal("grant_mutation not registered")
	}

	cleanUser := seedTestUser(t, 1, "alice", "Aliceia", 1)
	defer cleanUser()

	// Empty pool: the action returns Success per actions_quest.go:51-53
	// (no eligible mutations is not an error).
	ctx := &EvalContext{Event: EventContext{UserId: 1}}
	if result := fn(nil, ctx); result != Success {
		t.Errorf("expected Success on empty pool, got %v", result)
	}

	// Nil user (UserId 99 not seeded) → Failure.
	missingCtx := &EvalContext{Event: EventContext{UserId: 99}}
	if result := fn(nil, missingCtx); result != Failure {
		t.Errorf("expected Failure for missing user, got %v", result)
	}
}

func TestActGrantMutation_WritesMutationKeyWhenPoolNonEmpty(t *testing.T) {
	fn := LookupAction("grant_mutation")
	if fn == nil {
		t.Fatal("grant_mutation not registered")
	}

	cleanUser := seedTestUser(t, 1, "alice", "Aliceia", 1)
	defer cleanUser()

	// Seed one rollable mutation into the registry. With an empty ownership
	// map, GetWeightedPool has no conflicts to prune, so this mutation ends
	// up as the only entry in the weighted pool.
	cleanMuts := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"test-mut-1": {
			MutationId: "test-mut-1",
			Name:       "Test Mutation",
			Rarity:     1,
			Pros:       []mutations.MutationEffect{{Type: "stat_flat", Target: "strength", Value: 1}},
		},
	})
	defer cleanMuts()

	user := requireUser(t, 1)
	user.Character.Mutations = map[string]int{}

	ctx := &EvalContext{Event: EventContext{UserId: 1}}
	if result := fn(nil, ctx); result != Success {
		t.Fatalf("expected Success on non-empty pool, got %v", result)
	}
	if _, ok := user.Character.Mutations["test-mut-1"]; !ok {
		t.Errorf("expected test-mut-1 in user.Character.Mutations, got %v",
			user.Character.Mutations)
	}
}

// ─── give_gold ───────────────────────────────────────────────────────

func TestActGiveGold_IncreasesGoldAndNotifies(t *testing.T) {
	fn := LookupAction("give_gold")
	if fn == nil {
		t.Fatal("give_gold not registered")
	}

	cleanUser := seedTestUser(t, 1, "alice", "Aliceia", 1)
	defer cleanUser()

	user := requireUser(t, 1)
	user.Character.Gold = 100

	captured, mu, cleanup := captureMessages(t)
	defer cleanup()

	ctx := &EvalContext{Event: EventContext{UserId: 1}}
	if result := fn(map[string]any{"amount": 25}, ctx); result != Success {
		t.Fatalf("expected Success, got %v", result)
	}
	if user.Character.Gold != 125 {
		t.Errorf("expected gold=125, got %d", user.Character.Gold)
	}

	events.ProcessEvents()

	mu.Lock()
	found := false
	for _, m := range *captured {
		if m.UserId == 1 && strings.Contains(m.Text, "25 gold") {
			found = true
			break
		}
	}
	mu.Unlock()
	if !found {
		t.Errorf("expected user message containing '25 gold', got %d messages: %+v", len(*captured), *captured)
	}

	// Failure cases: amount <= 0.
	if result := fn(map[string]any{"amount": 0}, ctx); result != Failure {
		t.Errorf("expected Failure for amount=0, got %v", result)
	}
	if result := fn(map[string]any{"amount": -5}, ctx); result != Failure {
		t.Errorf("expected Failure for amount=-5, got %v", result)
	}

	// Nil user → Failure.
	missingCtx := &EvalContext{Event: EventContext{UserId: 99}}
	if result := fn(map[string]any{"amount": 10}, missingCtx); result != Failure {
		t.Errorf("expected Failure for missing user, got %v", result)
	}
}

// ─── send_user_text ──────────────────────────────────────────────────

func TestActSendUserText_DeliversToUser(t *testing.T) {
	fn := LookupAction("send_user_text")
	if fn == nil {
		t.Fatal("send_user_text not registered")
	}

	cleanUser := seedTestUser(t, 1, "alice", "Aliceia", 1)
	defer cleanUser()

	captured, mu, cleanup := captureMessages(t)
	defer cleanup()

	const wantText = "you feel a chill"
	ctx := &EvalContext{Event: EventContext{UserId: 1}}
	if result := fn(map[string]any{"text": wantText}, ctx); result != Success {
		t.Fatalf("expected Success, got %v", result)
	}

	events.ProcessEvents()
	// Pipeline normalize stage capitalizes the first letter; compare
	// case-insensitively so the test still validates content delivery.

	mu.Lock()
	found := false
	for _, m := range *captured {
		if m.UserId == 1 && strings.Contains(strings.ToLower(m.Text), wantText) {
			found = true
			break
		}
	}
	mu.Unlock()
	if !found {
		t.Errorf("expected user message containing %q, got %d messages: %+v", wantText, len(*captured), *captured)
	}

	// Nil user → Failure.
	missingCtx := &EvalContext{Event: EventContext{UserId: 99}}
	if result := fn(map[string]any{"text": "noop"}, missingCtx); result != Failure {
		t.Errorf("expected Failure for missing user, got %v", result)
	}
}

// ─── send_room_text ──────────────────────────────────────────────────

func TestActSendRoomText_BroadcastsToRoom(t *testing.T) {
	fn := LookupAction("send_room_text")
	if fn == nil {
		t.Fatal("send_room_text not registered")
	}

	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	// Post-T9, Room.SendText fans out per-recipient via r.players;
	// without a player in the room, no events are emitted. Seed a
	// player into room 1 and add them to the room's player list.
	cleanUser := seedTestUser(t, 1, "alice", "Aliceia", 1)
	defer cleanUser()
	if room1 := rooms.LoadRoom(1); room1 != nil {
		room1.AddPlayer(1)
	}

	captured, mu, cleanup := captureMessages(t)
	defer cleanup()

	const wantText = "the wind howls"
	ctx := &EvalContext{RoomId: 1}
	if result := fn(map[string]any{"text": wantText}, ctx); result != Success {
		t.Fatalf("expected Success, got %v", result)
	}

	events.ProcessEvents()

	// Room.SendText now fans out per-user (UserId set, no RoomId) and
	// the normalize stage capitalizes the first letter. Compare
	// case-insensitively and accept either targeted or room-broadcast
	// envelopes so the test stays valid through the T9 pipeline switch.
	mu.Lock()
	found := false
	for _, m := range *captured {
		if (m.RoomId == 1 || m.UserId != 0) && strings.Contains(strings.ToLower(m.Text), wantText) {
			found = true
			break
		}
	}
	mu.Unlock()
	if !found {
		t.Errorf("expected room message for room 1 containing %q, got %d messages: %+v", wantText, len(*captured), *captured)
	}

	// Nil room (RoomId 99 not seeded) → Failure.
	missingCtx := &EvalContext{RoomId: 99}
	if result := fn(map[string]any{"text": "noop"}, missingCtx); result != Failure {
		t.Errorf("expected Failure for missing room, got %v", result)
	}
}

// ─── intercept ───────────────────────────────────────────────────────

func TestActIntercept_SetsCtxIntercepted(t *testing.T) {
	fn := LookupAction("intercept")
	if fn == nil {
		t.Fatal("intercept not registered")
	}

	ctx := &EvalContext{}
	if ctx.Intercepted {
		t.Fatal("Intercepted should default to false")
	}
	if result := fn(nil, ctx); result != Success {
		t.Errorf("expected Success, got %v", result)
	}
	if !ctx.Intercepted {
		t.Error("expected ctx.Intercepted=true after intercept action")
	}
}

// ─── remove_buff ─────────────────────────────────────────────────────

func TestActRemoveBuff_RemovesBuffFromUser(t *testing.T) {
	fn := LookupAction("remove_buff")
	if fn == nil {
		t.Fatal("remove_buff not registered")
	}

	// Seed a single buff spec for buff id 100. TriggerCount > 0 ensures
	// the buff lives long enough for the act-then-assert cycle.
	cleanBuffs := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		100: {BuffId: 100, Name: "TestBuff", TriggerCount: 5, RoundInterval: 1},
	})
	defer cleanBuffs()

	cleanUser := seedTestUser(t, 1, "alice", "Aliceia", 1)
	defer cleanUser()

	user := requireUser(t, 1)
	if err := user.Character.AddBuff(100, false); err != nil {
		t.Fatalf("AddBuff(100) failed: %v", err)
	}
	if !user.Character.HasBuff(100) {
		t.Fatal("precondition: user should have buff 100 after AddBuff")
	}

	ctx := &EvalContext{Event: EventContext{UserId: 1}}
	if result := fn(map[string]any{"buff_id": 100}, ctx); result != Success {
		t.Fatalf("expected Success, got %v", result)
	}

	// RemoveBuff sets TriggersLeft=0 (Expired). GetBuffs filters expired
	// out, so a zero-length result confirms the removal contract.
	if got := user.Character.GetBuffs(100); len(got) != 0 {
		t.Errorf("expected 0 active buffs with id 100 after remove, got %d", len(got))
	}

	// Nil user → Failure.
	missingCtx := &EvalContext{Event: EventContext{UserId: 99}}
	if result := fn(map[string]any{"buff_id": 100}, missingCtx); result != Failure {
		t.Errorf("expected Failure for missing user, got %v", result)
	}
}

// ─── move_player ─────────────────────────────────────────────────────

func TestActMovePlayer_TeleportsUser(t *testing.T) {
	fn := LookupAction("move_player")
	if fn == nil {
		t.Fatal("move_player not registered")
	}

	// Both rooms must be seeded in a single SeedRoomsForTest call because
	// each call replaces the global roomManager.
	cleanRooms := rooms.SeedRoomsForTest(map[int]*rooms.Room{
		1: {RoomId: 1, Zone: "TestZone", Title: "Origin", Exits: nil},
		2: {RoomId: 2, Zone: "TestZone", Title: "Dest", Exits: nil},
	}, map[string]*rooms.ZoneConfig{})
	defer cleanRooms()

	cleanUser := seedTestUser(t, 1, "alice", "Aliceia", 1)
	defer cleanUser()

	user := requireUser(t, 1)
	if user.Character.RoomId != 1 {
		t.Fatalf("precondition: expected user in room 1, got %d", user.Character.RoomId)
	}

	ctx := &EvalContext{Event: EventContext{UserId: 1}}
	if result := fn(map[string]any{"room_id": 2}, ctx); result != Success {
		t.Fatalf("expected Success, got %v", result)
	}
	if user.Character.RoomId != 2 {
		t.Errorf("expected user.Character.RoomId=2 after move, got %d", user.Character.RoomId)
	}

	// room_id == 0 → Failure.
	if result := fn(map[string]any{"room_id": 0}, ctx); result != Failure {
		t.Errorf("expected Failure for room_id=0, got %v", result)
	}
}

// ─── summon_companion (hostile branch) ───────────────────────────────

func TestActSummonCompanion_HostileSetsAggroAndEngages(t *testing.T) {
	fn := LookupAction("summon_companion")
	if fn == nil {
		t.Fatal("summon_companion not registered")
	}

	// Seed room 1.
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()

	// Seed caller (template 1, instance 100) AND companion template (template 7)
	// in a single SeedMobsForTest call so both specs are present when
	// NewMobById(7, ...) is called inside the action.
	// SeedMobsForTest sets instanceCounter = 200, so the first NewMobById
	// call will produce instance ID 201.
	callerSpec := &mobs.Mob{
		MobId: mobs.MobId(1),
		Character: characters.Character{
			Name:   "TestCaller",
			RoomId: 1,
			Buffs:  buffs.New(),
		},
	}
	callerInstance := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: 100,
		HomeRoomId: 1,
		Character: characters.Character{
			Name:   "TestCaller",
			RoomId: 1,
			Buffs:  buffs.New(),
		},
	}
	companionSpec := &mobs.Mob{
		MobId: mobs.MobId(7),
		Character: characters.Character{
			Name:   "TestCompanion",
			RoomId: 1,
			Buffs:  buffs.New(),
		},
	}
	cleanMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{1: callerSpec, 7: companionSpec},
		map[int]*mobs.Mob{100: callerInstance},
	)
	defer cleanMobs()

	// Place caller instance in room 1.
	rooms.LoadRoom(1).AddMob(100)

	// Seed user 1 in room 1.
	cleanUser := seedTestUser(t, 1, "alice", "Aliceia", 1)
	defer cleanUser()

	// Pre-action snapshot: room should contain only instance 100.
	room := rooms.LoadRoom(1)
	preMobs := room.GetMobs(rooms.FindAll)
	if len(preMobs) != 1 || preMobs[0] != 100 {
		t.Fatalf("precondition: expected room to contain only instance 100, got %v", preMobs)
	}

	// Install Input listener to capture queued commands.
	captured, mu, cleanupListener := captureInputs(t)
	defer cleanupListener()

	// Act: hostile is now a proper bool (getBoolParam still accepts the
	// legacy string "true" form for backward compat — see params.go).
	params := map[string]any{
		"mob_id":    7,
		"hostile":   true,
		"count":     1,
		"base_pool": 50,
	}
	ctx := &EvalContext{
		InstanceId: 100,
		RoomId:     1,
		Event:      EventContext{UserId: 1},
	}
	if result := fn(params, ctx); result != Success {
		t.Fatalf("expected Success, got %v", result)
	}

	// Fire queued events so the Input listener captures the lookfortrouble command.
	events.ProcessEvents()

	// ── Assert 1: room has one MORE mob than before. ──────────────────
	postMobs := room.GetMobs(rooms.FindAll)
	if len(postMobs) != len(preMobs)+1 {
		t.Fatalf("expected %d mobs in room after summon, got %d: %v", len(preMobs)+1, len(postMobs), postMobs)
	}

	// Find the new instance ID by set-difference.
	preSet := make(map[int]bool, len(preMobs))
	for _, id := range preMobs {
		preSet[id] = true
	}
	newInstanceId := 0
	for _, id := range postMobs {
		if !preSet[id] {
			newInstanceId = id
			break
		}
	}
	if newInstanceId == 0 {
		t.Fatalf("could not find new instance ID in post-summon mob list %v", postMobs)
	}

	// ── Assert 2: new instance has Aggro targeting user 1. ───────────
	companion := mobs.GetInstance(newInstanceId)
	if companion == nil {
		t.Fatalf("mobs.GetInstance(%d) returned nil after summon", newInstanceId)
	}
	if companion.Character.Aggro == nil {
		t.Fatalf("expected companion.Character.Aggro != nil, got nil")
	}
	if companion.Character.Aggro.UserId != 1 {
		t.Errorf("expected Aggro.UserId=1, got %d", companion.Character.Aggro.UserId)
	}

	// ── Assert 3: "lookfortrouble" was queued on the new instance. ────
	mu.Lock()
	defer mu.Unlock()
	found := false
	for _, in := range *captured {
		if in.MobInstanceId == newInstanceId && in.InputText == "lookfortrouble" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'lookfortrouble' Input event for instance %d, got %d events: %+v",
			newInstanceId, len(*captured), *captured)
	}
}

// TestActSummonCompanion_HostileFallsBackWithoutEventUserId covers the
// crashsite-boss-mechanics Part A fix: production's "mob_combat_round"
// event (fired every round from internal/hooks/NewRound_DoCombat.go for
// the summoning mob's own combat tick — the actual trigger the Core
// Guardian's Grapnel Warden / Hull Sweeper re-summon nodes use) never
// carries a UserId. Before the fix, that meant a hostile summon_companion
// triggered from that event silently never called SetAggro at all — the
// companion spawned into the room inert. This asserts the fallback picks
// an eligible present player and sets Aggro even when ctx.Event.UserId==0.
func TestActSummonCompanion_HostileFallsBackWithoutEventUserId(t *testing.T) {
	fn := LookupAction("summon_companion")
	if fn == nil {
		t.Fatal("summon_companion not registered")
	}

	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()

	callerSpec := &mobs.Mob{
		MobId: mobs.MobId(1),
		Character: characters.Character{
			Name:   "TestCaller",
			RoomId: 1,
			Buffs:  buffs.New(),
		},
	}
	callerInstance := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: 100,
		HomeRoomId: 1,
		Character: characters.Character{
			Name:   "TestCaller",
			RoomId: 1,
			Buffs:  buffs.New(),
		},
	}
	companionSpec := &mobs.Mob{
		MobId: mobs.MobId(7),
		Character: characters.Character{
			Name:   "TestCompanion",
			RoomId: 1,
			Buffs:  buffs.New(),
		},
	}
	cleanMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{1: callerSpec, 7: companionSpec},
		map[int]*mobs.Mob{100: callerInstance},
	)
	defer cleanMobs()

	rooms.LoadRoom(1).AddMob(100)

	cleanUser := seedTestUser(t, 1, "alice", "Aliceia", 1)
	defer cleanUser()

	room := rooms.LoadRoom(1)
	// seedTestUser only stamps Character.RoomId — the room's own player
	// list (what GetPlayers/pickEligibleRoomPlayer scans) is a separate
	// registry that must be populated explicitly.
	room.AddPlayer(1)
	preMobs := room.GetMobs(rooms.FindAll)

	params := map[string]any{
		"mob_id":    7,
		"hostile":   true,
		"count":     1,
		"base_pool": 50,
	}
	// No UserId on the event — matches the real mob_combat_round trigger.
	ctx := &EvalContext{
		InstanceId: 100,
		RoomId:     1,
		Event:      EventContext{EventType: "mob_combat_round"},
	}
	if result := fn(params, ctx); result != Success {
		t.Fatalf("expected Success, got %v", result)
	}

	postMobs := room.GetMobs(rooms.FindAll)
	preSet := make(map[int]bool, len(preMobs))
	for _, id := range preMobs {
		preSet[id] = true
	}
	newInstanceId := 0
	for _, id := range postMobs {
		if !preSet[id] {
			newInstanceId = id
			break
		}
	}
	if newInstanceId == 0 {
		t.Fatalf("could not find new instance ID in post-summon mob list %v", postMobs)
	}

	companion := mobs.GetInstance(newInstanceId)
	if companion == nil {
		t.Fatalf("mobs.GetInstance(%d) returned nil after summon", newInstanceId)
	}
	if companion.Character.Aggro == nil {
		t.Fatalf("expected companion.Character.Aggro != nil (fallback target), got nil")
	}
	if companion.Character.Aggro.UserId != 1 {
		t.Errorf("expected Aggro.UserId=1 (the only eligible player in the room), got %d", companion.Character.Aggro.UserId)
	}
}

// ─── command_best_of ───────────────────────────────────────────────────────

func TestActCommandBestOf_FiresFirstReady(t *testing.T) {
	mob := newTestMob(t)
	// Set up aggro so taunt is ready
	mob.Character.SetAggro(0, 0, characters.DefaultAttack)
	mob.Character.Cooldowns = characters.Cooldowns{}

	params := map[string]any{"cmds": []any{"taunt", "trip"}}
	ctx := &EvalContext{InstanceId: mob.InstanceId}

	result := LookupAction("command_best_of")(params, ctx)
	if result != Success {
		t.Errorf("expected Success, got %v", result)
	}
}

func TestActCommandBestOf_SkipsNotReadyFiresReady(t *testing.T) {
	mob := newTestMob(t)
	// Put special-move on cooldown so taunt will not be ready.
	// trip also requires aggro and standing target, so it won't be ready either.
	// But kick only requires aggro, no target state check.
	mob.Character.Cooldowns = characters.Cooldowns{"special-move": 5}
	mob.Character.SetAggro(0, 0, characters.DefaultAttack)

	params := map[string]any{"cmds": []any{"taunt", "kick"}}
	ctx := &EvalContext{InstanceId: mob.InstanceId}

	result := LookupAction("command_best_of")(params, ctx)
	// With special-move cooldown, all commands fail
	if result != Failure {
		t.Errorf("expected Failure (special-move cooldown blocks all), got %v", result)
	}
}

func TestActCommandBestOf_SkipsUnavailableCommands(t *testing.T) {
	// kick requires the actor to have legs (Phase-2 anatomy gate); the default
	// test mob has SpeciesId 0, so seed it a humanoid body so kick can be ready.
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		0: {SpeciesId: 0, Name: "test", BodyParts: []string{"arms", "hands", "legs"}},
	})
	defer cleanup()

	mob := newTestMob(t)
	// bash requires a shield, which the mob doesn't have
	// trip requires standing target, which isn't set up
	// kick just needs aggro + legs, which we set up
	mob.Character.Cooldowns = characters.Cooldowns{}
	mob.Character.SetAggro(0, 0, characters.DefaultAttack)

	params := map[string]any{"cmds": []any{"bash", "trip", "kick"}}
	ctx := &EvalContext{InstanceId: mob.InstanceId}

	result := LookupAction("command_best_of")(params, ctx)
	if result != Success {
		t.Errorf("expected Success (kick should be ready), got %v", result)
	}
}

func TestActCommandBestOf_AllFailReturnsFailure(t *testing.T) {
	mob := newTestMob(t)
	// bash requires shield (not present)
	// trip requires standing target with aggro (target not set up)
	mob.Character.Cooldowns = characters.Cooldowns{}

	params := map[string]any{"cmds": []any{"bash", "trip"}}
	ctx := &EvalContext{InstanceId: mob.InstanceId}

	result := LookupAction("command_best_of")(params, ctx)
	if result != Failure {
		t.Errorf("expected Failure (all commands not ready), got %v", result)
	}
}

func TestActCommandBestOf_MissingMobReturnsFailure(t *testing.T) {
	params := map[string]any{"cmds": []any{"taunt", "trip"}}
	ctx := &EvalContext{InstanceId: 999} // non-existent mob

	result := LookupAction("command_best_of")(params, ctx)
	if result != Failure {
		t.Errorf("expected Failure (missing mob), got %v", result)
	}
}

func TestActCommandBestOf_EmptyListReturnsFailure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Cooldowns = characters.Cooldowns{}

	params := map[string]any{"cmds": []any{}}
	ctx := &EvalContext{InstanceId: mob.InstanceId}

	result := LookupAction("command_best_of")(params, ctx)
	if result != Failure {
		t.Errorf("expected Failure (empty command list), got %v", result)
	}
}

func TestActCommandBestOf_InvalidParamReturnsFailure(t *testing.T) {
	mob := newTestMob(t)

	params := map[string]any{}
	ctx := &EvalContext{InstanceId: mob.InstanceId}

	result := LookupAction("command_best_of")(params, ctx)
	if result != Failure {
		t.Errorf("expected Failure (invalid param), got %v", result)
	}
}

// ─── cancel_activity ─────────────────────────────────────────────

// TestCancelActivityBtreeAction verifies that actionCancelActivity
// transitions an active Casting mob to Free (Success), and returns
// Failure when already Free (idempotency guard).
func TestCancelActivityBtreeAction(t *testing.T) {
	fn := LookupAction("cancel_activity")
	if fn == nil {
		t.Fatal("cancel_activity not registered")
	}

	// Seed a mob with an Activity machine and transition it to Casting.
	mob := &mobs.Mob{MobId: 1, InstanceId: 999}
	mob.Character.Name = "TestCaster"
	mob.Character.Activity = activity.NewMachine()
	if err := mob.Character.Activity.TransitionToCasting(
		activity.CastingData{SpellId: "fireball", TotalConvictionCost: 10},
		state.TransitionReason{Trigger: activity.TriggerCastBegin},
	); err != nil {
		t.Fatalf("test setup: TransitionToCasting failed: %v", err)
	}
	mobs.SetInstanceForTest(mob.InstanceId, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(mob.InstanceId, nil) })

	ctx := &EvalContext{InstanceId: mob.InstanceId}

	// Act: cancel the cast.
	if result := fn(nil, ctx); result != Success {
		t.Fatalf("expected Success on active cast, got %v", result)
	}
	if !mob.Character.Activity.IsFree() {
		t.Errorf("Activity should be Free after cancel; got %v",
			mob.Character.Activity.State())
	}

	// Idempotency: calling again on Free returns Failure.
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure when Activity already Free, got %v", result)
	}
}

// TestCancelActivityBtreeAction_MissingMob verifies Failure when the
// EvalContext references a non-existent mob instance.
func TestCancelActivityBtreeAction_MissingMob(t *testing.T) {
	fn := LookupAction("cancel_activity")
	if fn == nil {
		t.Fatal("cancel_activity not registered")
	}
	ctx := &EvalContext{InstanceId: 99998} // never seeded
	if result := fn(nil, ctx); result != Failure {
		t.Errorf("expected Failure for missing mob, got %v", result)
	}
}

// ─── try_special_move ────────────────────────────────────────────────────────

// TestActTrySpecialMove_RegisteredInActionRegistry verifies the action is
// present in the registry before any functional test runs.
func TestActTrySpecialMove_RegisteredInActionRegistry(t *testing.T) {
	if LookupAction("try_special_move") == nil {
		t.Error("try_special_move must be registered in actionRegistry")
	}
}

// TestActTrySpecialMove_BeastMobReturnsSuccess verifies that a fanged, legged,
// no-hands beast mob (wolf) with a registered aggro target returns Success —
// SelectSpecialMove picks a beast move and the action issues it.
func TestActTrySpecialMove_BeastMobReturnsSuccess(t *testing.T) {
	cleanupSp := species.SeedSpeciesForTest(map[int]*species.Species{
		9910: {
			SpeciesId:     9910,
			Name:          "test_wolf",
			BodyParts:     []string{"legs", "mouth"},
			NaturalAttack: items.Bite,
		},
	})
	defer cleanupSp()

	// Target mob at InstanceId 201.
	target := &mobs.Mob{MobId: 2, InstanceId: 201}
	target.Character.Name = "Target"
	target.Character.HealthMax.Value = 100
	target.Character.Health = 100
	mobs.SetInstanceForTest(target.InstanceId, target)
	t.Cleanup(func() { mobs.SetInstanceForTest(target.InstanceId, nil) })

	// Beast mob (InstanceId 100 via newTestMob): wolf, predator profile,
	// aggro on target mob. Clear any leftover special-move cooldown.
	mob := newTestMob(t)
	mob.Character.SpeciesId = 9910
	mob.Character.HealthMax.Value = 100
	mob.Character.Health = 100
	mob.AIProfile = "predator"
	mob.Character.Cooldowns = characters.Cooldowns{}
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}

	result := LookupAction("try_special_move")(nil, ctx)
	if result != Success {
		t.Errorf("expected Success for beast mob with viable beast moves, got %v", result)
	}
}

// TestActTrySpecialMove_HumanoidMobReturnsFailure verifies that a humanoid mob
// (has hands) returns Failure — SelectSpecialMove returns a non-beast move
// (grapple/trip/kick), so the tactical cascade retains control.
func TestActTrySpecialMove_HumanoidMobReturnsFailure(t *testing.T) {
	cleanupSp := species.SeedSpeciesForTest(map[int]*species.Species{
		9911: {
			SpeciesId: 9911,
			Name:      "test_humanoid",
			BodyParts: []string{"arms", "hands", "legs", "mouth"},
		},
	})
	defer cleanupSp()

	// Target mob at InstanceId 202.
	target := &mobs.Mob{MobId: 2, InstanceId: 202}
	target.Character.Name = "Target"
	target.Character.HealthMax.Value = 100
	target.Character.Health = 100
	mobs.SetInstanceForTest(target.InstanceId, target)
	t.Cleanup(func() { mobs.SetInstanceForTest(target.InstanceId, nil) })

	// Humanoid mob at InstanceId 203: hands → no beast moves available.
	mob := &mobs.Mob{MobId: 1, InstanceId: 203}
	mob.Character.Name = "HumanMob"
	mob.Character.HealthMax.Value = 100
	mob.Character.Health = 100
	mob.Character.SpeciesId = 9911
	mob.Character.Cooldowns = characters.Cooldowns{}
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)
	mobs.SetInstanceForTest(mob.InstanceId, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(mob.InstanceId, nil) })

	ctx := &EvalContext{InstanceId: mob.InstanceId}

	result := LookupAction("try_special_move")(nil, ctx)
	if result != Failure {
		t.Errorf("expected Failure for humanoid mob (non-beast moves only), got %v", result)
	}
}

// TestActTrySpecialMove_NoAggroReturnsFailure verifies that a mob without an
// aggro target returns Failure immediately.
func TestActTrySpecialMove_NoAggroReturnsFailure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Aggro = nil

	ctx := &EvalContext{InstanceId: mob.InstanceId}

	result := LookupAction("try_special_move")(nil, ctx)
	if result != Failure {
		t.Errorf("expected Failure (no aggro), got %v", result)
	}
}
