package behaviortree

import (
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/sealedcrate"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// buildForagerMob creates a minimal forager mob instance and registers it
// for test use. Cleans up on test end.
func buildForagerMob(
	t *testing.T,
	instanceId int,
	mobId mobs.MobId,
	roomId int,
	hp, hpMax int,
) *mobs.Mob {
	t.Helper()
	mob := &mobs.Mob{
		MobId:      mobId,
		InstanceId: instanceId,
		HomeRoomId: roomId,
	}
	mob.Character.RoomId = roomId
	mob.Character.Health = hp
	mob.Character.HealthMax.Value = hpMax
	mob.Character.Buffs = buffs.New()
	mob.Character.Stats.Strength.ValueAdj = 100
	mob.Character.Stats.Dexterity.ValueAdj = 100
	mob.Character.Stats.Vitality.ValueAdj = 100
	mob.Character.Stats.Perception.ValueAdj = 100
	mob.Character.Stats.Willpower.ValueAdj = 100
	mob.Character.Stats.Charisma.ValueAdj = 100
	mobs.SetInstanceForTest(instanceId, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(instanceId, nil) })
	return mob
}

// TestForagerStep_Registered verifies the action is registered.
func TestForagerStep_Registered(t *testing.T) {
	fn := LookupAction("forager_step")
	if fn == nil {
		t.Fatal("forager_step not registered in actionRegistry")
	}
}

// TestForagerStep_DefaultsToResting verifies that on the very first tick
// (empty MobState) the state key is initialised to "resting".
func TestForagerStep_DefaultsToResting(t *testing.T) {
	fn := LookupAction("forager_step")

	// Mob 371 = Tova (Marsh forager). Sanctuary room = 4198.
	mob := buildForagerMob(t, 8200, 371, 4198, 100, 100)
	_ = mob

	state := NewBehaviorState()
	ctx := &EvalContext{
		InstanceId: 8200,
		RoomId:     4198,
		MobState:   state,
	}
	fn(nil, ctx)

	got := state.GetString(keyForagerState)
	if got != forager.StateResting.Name() {
		t.Errorf("after first tick forager_state = %q, want %q",
			got, forager.StateResting.Name())
	}
}

// TestForagerStep_NotAForagerReturnsFailure verifies that a mob with no
// ForagerProfile returns Failure immediately.
func TestForagerStep_NotAForagerReturnsFailure(t *testing.T) {
	fn := LookupAction("forager_step")

	// Mob ID 99999 has no registered profile.
	mob := buildForagerMob(t, 8201, 99999, 1, 100, 100)
	_ = mob

	state := NewBehaviorState()
	ctx := &EvalContext{
		InstanceId: 8201,
		RoomId:     1,
		MobState:   state,
	}
	result := fn(nil, ctx)
	if result != Failure {
		t.Errorf("expected Failure for unregistered mob, got %v", result)
	}
}

// TestForagerStep_HPEmergencyTransitionsToRecalling verifies that a mob
// with critically low HP is short-circuited to Recalling regardless of
// what state it is currently in.
func TestForagerStep_HPEmergencyTransitionsToRecalling(t *testing.T) {
	fn := LookupAction("forager_step")

	// Mob 371, at 10% HP — well below the 50% threshold.
	mob := buildForagerMob(t, 8202, 371, 4177 /* territory room */, 10, 100)
	_ = mob

	state := NewBehaviorState()
	// Start in Foraging — the HP emergency must override this.
	state.Set(keyForagerState, forager.StateForaging.Name())

	ctx := &EvalContext{
		InstanceId: 8202,
		RoomId:     4177,
		MobState:   state,
	}
	result := fn(nil, ctx)
	if result != Success {
		t.Errorf("HP emergency: expected Success, got %v", result)
	}
	got := state.GetString(keyForagerState)
	if got != forager.StateRecalling.Name() {
		t.Errorf("HP emergency: forager_state = %q, want %q",
			got, forager.StateRecalling.Name())
	}
}

// TestForagerStep_RestingFullHPAdvances verifies that when the forager is
// at its sanctuary, fully healed, and the resting dwell has elapsed, it
// advances to TravelingToTerritory.
func TestForagerStep_RestingFullHPAdvances(t *testing.T) {
	fn := LookupAction("forager_step")

	// Mob 371, full HP, at sanctuary (4198).
	mob := buildForagerMob(t, 8203, 371, 4198, 100, 100)
	_ = mob

	state := NewBehaviorState()
	state.Set(keyForagerState, forager.StateResting.Name())
	// Set started_round to 0 so dwell has elapsed (current round >= 120).
	state.Set(keyStateStartedRound, "0")
	// Ensure we're past round 120 — util.GetRoundCount() in tests is >= 1.
	// Force it past restingDuration by setting started to a round far in
	// the past.
	state.Set(keyStateStartedRound,
		strconv.FormatUint(util.GetRoundCount()-uint64(configs.GetBalanceConfig().ForagerRestDurationRounds)-1, 10))

	ctx := &EvalContext{
		InstanceId: 8203,
		RoomId:     4198, // sanctuary
		MobState:   state,
	}
	result := fn(nil, ctx)
	if result != Success {
		t.Errorf("resting+full-HP: expected Success, got %v", result)
	}
	got := state.GetString(keyForagerState)
	if got != forager.StateTravelingToTerritory.Name() {
		t.Errorf("resting+full-HP: forager_state = %q, want %q",
			got, forager.StateTravelingToTerritory.Name())
	}
}

// TestForagerStep_RestingNotFullHPStaysResting verifies that a forager
// still recovering (HP < max) stays in Resting even if dwell has elapsed.
func TestForagerStep_RestingNotFullHPStaysResting(t *testing.T) {
	fn := LookupAction("forager_step")

	// Mob 371, partial HP, at sanctuary.
	mob := buildForagerMob(t, 8204, 371, 4198, 80, 100)
	_ = mob

	state := NewBehaviorState()
	state.Set(keyForagerState, forager.StateResting.Name())
	state.Set(keyStateStartedRound,
		strconv.FormatUint(util.GetRoundCount()-uint64(configs.GetBalanceConfig().ForagerRestDurationRounds)-1, 10))

	ctx := &EvalContext{
		InstanceId: 8204,
		RoomId:     4198,
		MobState:   state,
	}
	result := fn(nil, ctx)
	// Should return Failure (let legacy idle fire) since HP < max.
	if result != Failure {
		t.Errorf("resting+partial-HP: expected Failure, got %v", result)
	}
	got := state.GetString(keyForagerState)
	if got != forager.StateResting.Name() {
		t.Errorf("resting+partial-HP: forager_state = %q, want %q",
			got, forager.StateResting.Name())
	}
}

// TestForagerStep_TravelingArrivesInTerritory verifies that reaching a
// territory room advances the state to Foraging.
func TestForagerStep_TravelingArrivesInTerritory(t *testing.T) {
	fn := LookupAction("forager_step")

	// Territory room 4177 is the first room in Tova's TerritoryRooms.
	mob := buildForagerMob(t, 8205, 371, 4177, 100, 100)
	_ = mob

	state := NewBehaviorState()
	state.Set(keyForagerState, forager.StateTravelingToTerritory.Name())

	ctx := &EvalContext{
		InstanceId: 8205,
		RoomId:     4177,
		MobState:   state,
	}
	result := fn(nil, ctx)
	if result != Success {
		t.Errorf("arrived territory: expected Success, got %v", result)
	}
	got := state.GetString(keyForagerState)
	if got != forager.StateForaging.Name() {
		t.Errorf("arrived territory: forager_state = %q, want %q",
			got, forager.StateForaging.Name())
	}
}

// TestForagerStep_RecallingAtSanctuaryTransitionsToResting verifies that
// arriving at the sanctuary while in Recalling transitions to Resting.
func TestForagerStep_RecallingAtSanctuaryTransitionsToResting(t *testing.T) {
	fn := LookupAction("forager_step")

	mob := buildForagerMob(t, 8206, 371, 4198 /* sanctuary */, 100, 100)
	_ = mob

	state := NewBehaviorState()
	state.Set(keyForagerState, forager.StateRecalling.Name())

	ctx := &EvalContext{
		InstanceId: 8206,
		RoomId:     4198,
		MobState:   state,
	}
	result := fn(nil, ctx)
	if result != Success {
		t.Errorf("recalling at sanctuary: expected Success, got %v", result)
	}
	got := state.GetString(keyForagerState)
	if got != forager.StateResting.Name() {
		t.Errorf("recalling at sanctuary: forager_state = %q, want %q",
			got, forager.StateResting.Name())
	}
}

// TestForagerStep_NilMobStateReturnsFailure ensures the nil-guard fires.
func TestForagerStep_NilMobStateReturnsFailure(t *testing.T) {
	fn := LookupAction("forager_step")
	mob := buildForagerMob(t, 8207, 371, 4198, 100, 100)
	_ = mob
	ctx := &EvalContext{
		InstanceId: 8207,
		RoomId:     4198,
		MobState:   nil, // deliberately nil
	}
	if got := fn(nil, ctx); got != Failure {
		t.Errorf("nil MobState: expected Failure, got %v", got)
	}
}

// TestTickForagerRecalling_DumpsSurplusIntoLockbox verifies that when a
// forager arrives at her sanctuary while in Recalling state, items in her
// satchel are transferred into the room's "lockbox" container, the lock's
// RotationSeed is bumped, and the state transitions to Resting.
func TestTickForagerRecalling_DumpsSurplusIntoLockbox(t *testing.T) {
	const sanctuaryRoom = 4198

	// Build a synthetic room with a lockbox container (difficulty 3,
	// RotationSeed 1 as per Task 7 YAML template).
	testRoom := &rooms.Room{
		RoomId: sanctuaryRoom,
		Zone:   "stillwater",
		Title:  "Stillwater Temple Sanctuary",
		Exits:  map[string]exit.RoomExit{},
		Containers: map[string]rooms.Container{
			"lockbox": {
				Lock: gamelock.Lock{
					Difficulty:   3,
					RotationSeed: 1,
				},
			},
		},
	}
	cleanRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{sanctuaryRoom: testRoom},
		map[string]*rooms.ZoneConfig{},
	)
	defer cleanRooms()

	// Mob 371 = Tova (Marsh forager). Sanctuary room = 4198.
	mob := buildForagerMob(t, 8210, 371, sanctuaryRoom, 100, 100)

	// Stash two items directly into the satchel (Items slice, bypassing
	// carry-capacity routing so the test is weight-agnostic).
	mob.Character.Items = append(mob.Character.Items,
		items.New(40021),
		items.New(40021),
	)

	p := forager.ProfileFor(371)
	if p == nil {
		t.Fatal("no forager profile for mob 371")
	}

	state := NewBehaviorState()
	state.Set(keyForagerState, forager.StateRecalling.Name())

	ctx := &EvalContext{
		InstanceId: 8210,
		RoomId:     sanctuaryRoom,
		MobState:   state,
	}

	res := tickForagerRecalling(p, mob, ctx)
	if res != Success {
		t.Fatalf("tickForagerRecalling = %v, want Success", res)
	}

	// Satchel should be empty after the dump.
	if len(mob.Character.Items) != 0 {
		t.Errorf("satchel after dump = %d items, want 0",
			len(mob.Character.Items))
	}

	// Lockbox should contain the dumped items.
	room := rooms.LoadRoom(sanctuaryRoom)
	if room == nil {
		t.Fatal("room 4198 missing")
	}
	box, ok := room.Containers["lockbox"]
	if !ok {
		t.Fatal("room 4198 has no lockbox")
	}
	if len(box.Items) < 2 {
		t.Errorf("lockbox after dump = %d items, want at least 2",
			len(box.Items))
	}
	// SetLocked() bumps RotationSeed — starts at 1, expect > 1 after dump.
	if box.Lock.RotationSeed <= 1 {
		t.Errorf("lockbox RotationSeed = %d, want > 1 (bumped on dump)",
			box.Lock.RotationSeed)
	}

	// State should have transitioned to Resting.
	if state.GetString(keyForagerState) != forager.StateResting.Name() {
		t.Errorf("forager_state = %q, want %q",
			state.GetString(keyForagerState),
			forager.StateResting.Name())
	}
}

// TestTickForagerDeliveringFernway_DumpsIntoSealedCrate verifies that Kessa
// (mob 373) dumps all fernway-bucket items from her satchel into the room's
// SealedCrate when she arrives at the meeting room, and immediately
// transitions to Recalling — no 150-round wait timer.
func TestTickForagerDeliveringFernway_DumpsIntoSealedCrate(t *testing.T) {
	const meetingRoom = 4038

	// Build a synthetic room with an attached sealed crate.
	crate := sealedcrate.New(meetingRoom, 2000)
	testRoom := &rooms.Room{
		RoomId: meetingRoom,
		Zone:   "North Road",
		Exits:  map[string]exit.RoomExit{},
	}
	testRoom.AttachSealedCrate(crate)

	cleanRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{meetingRoom: testRoom},
		map[string]*rooms.ZoneConfig{},
	)
	defer cleanRooms()

	// Mob 373 = Kessa (Fernway forager). Meeting room = 4038.
	mob := buildForagerMob(t, 8220, 373, meetingRoom, 100, 100)

	// Stash fernway-bucket items directly in the satchel. Use item
	// literals because standalone tests don't load spec data files.
	// 40046 = moonpetal, 40049 = ironbark shaving — both in "fernway"
	// bucket per internal/economy/buckets.go.
	mob.Character.Items = append(mob.Character.Items,
		items.Item{ItemId: 40046},
		items.Item{ItemId: 40049},
	)

	p := forager.ProfileFor(373)
	if p == nil {
		t.Fatal("no forager profile for mob 373")
	}

	state := NewBehaviorState()
	state.Set(keyForagerState, forager.StateDelivering.Name())

	ctx := &EvalContext{
		InstanceId: 8220,
		RoomId:     meetingRoom,
		MobState:   state,
	}

	res := tickForagerDeliveringFernway(p, mob, ctx)
	if res != Success {
		t.Fatalf("tickForagerDeliveringFernway = %v, want Success", res)
	}

	// All fernway-bucket items should have been dumped into the crate.
	if crate.Len() != 2 {
		t.Errorf("crate.Len() after dump = %d, want 2", crate.Len())
	}

	// Satchel should be empty.
	if len(mob.Character.Items) != 0 {
		t.Errorf("satchel after dump = %d items, want 0",
			len(mob.Character.Items))
	}

	// State must have transitioned to Recalling — no wait timer.
	if state.GetString(keyForagerState) != forager.StateRecalling.Name() {
		t.Errorf("forager_state = %q, want %q",
			state.GetString(keyForagerState),
			forager.StateRecalling.Name())
	}
}

// TestForagerWatchdog_ResetsStuckMobToRecalling verifies that when a
// forager has been in the same state for longer than the threshold,
// the watchdog fires and transitions to Recalling.
func TestForagerWatchdog_ResetsStuckMobToRecalling(t *testing.T) {
	fn := LookupAction("forager_step")

	// Mob 371 = Tova (Marsh forager). Sanctuary room = 4198.
	mob := buildForagerMob(t, 8230, 371, 4198, 100, 100)
	_ = mob

	state := NewBehaviorState()
	state.Set(keyForagerState, forager.StateResting.Name())
	// Set started_round to 1 (far in the past). With a threshold of 600,
	// any current round > 601 will trigger the watchdog.
	// Since test round count is likely > 1, and current will be > 600,
	// the watchdog should fire.
	state.Set(keyStateStartedRound, "1")

	ctx := &EvalContext{
		InstanceId: 8230,
		RoomId:     4198, // sanctuary
		MobState:   state,
	}
	result := fn(nil, ctx)
	// If watchdog fires, it should return Success (and transition to Recalling).
	// If watchdog doesn't fire, normal state handling proceeds (likely Failure
	// if HP is not full or dwell hasn't elapsed).
	if result != Success {
		t.Logf("watchdog reset: got result %v (expected Success if watchdog fired)", result)
	}
	got := state.GetString(keyForagerState)
	// The watchdog should have transitioned to Recalling.
	if got != forager.StateRecalling.Name() {
		t.Errorf("watchdog reset: forager_state = %q, want %q (watchdog did not fire)",
			got, forager.StateRecalling.Name())
	}
}

// TestForagerWatchdog_DoesNotResetActiveForager verifies that a forager
// with a recent state_started_round (within the threshold) is not reset.
func TestForagerWatchdog_DoesNotResetActiveForager(t *testing.T) {
	fn := LookupAction("forager_step")

	// Mob 371 = Tova (Marsh forager).
	mob := buildForagerMob(t, 8231, 371, 4177, 100, 100)
	_ = mob

	state := NewBehaviorState()
	state.Set(keyForagerState, forager.StateForaging.Name())
	// Set started_round to current round (or just a few rounds ago),
	// well within the 600-round threshold.
	started := util.GetRoundCount() - 50 // 50 rounds ago
	state.Set(keyStateStartedRound, strconv.FormatUint(started, 10))

	ctx := &EvalContext{
		InstanceId: 8231,
		RoomId:     4177,
		MobState:   state,
	}
	result := fn(nil, ctx)
	// Foraging at 4177 should advance (successful forage attempt or
	// successful action), so Success is expected.
	if result != Success && result != Failure {
		t.Logf("result = %v (Success or Failure both acceptable)", result)
	}
	got := state.GetString(keyForagerState)
	if got != forager.StateForaging.Name() {
		t.Errorf("active forager: forager_state = %q, want %q (unchanged)",
			got, forager.StateForaging.Name())
	}
}

// TestTickForagerRecalling_TeleportsDirectly verifies that tickForagerRecalling
// teleports the mob to its anchor room without going through the fold-cast system,
// clears any stale CastingState, and leaves the forager_state still in
// "recalling" (the at-sanctuary path handles the resting transition on the
// next tick).
func TestTickForagerRecalling_TeleportsDirectly(t *testing.T) {
	const (
		territoryRoom = 4177 // somewhere in Tova's territory (not sanctuary)
		sanctuaryRoom = 4198 // Tova's sanctuary
	)

	// Build two rooms: the forager's current location and the anchor destination.
	testRooms := map[int]*rooms.Room{
		territoryRoom: {
			RoomId: territoryRoom,
			Zone:   "stillwater",
			Title:  "Marsh Territory",
			Exits:  map[string]exit.RoomExit{},
		},
		sanctuaryRoom: {
			RoomId: sanctuaryRoom,
			Zone:   "stillwater",
			Title:  "Stillwater Temple Sanctuary",
			Exits:  map[string]exit.RoomExit{},
		},
	}
	cleanRooms := rooms.SeedRoomsForTest(testRooms, map[string]*rooms.ZoneConfig{})
	defer cleanRooms()

	// Mob 371 = Tova (Marsh forager). Start her in the territory room.
	mob := buildForagerMob(t, 8240, 371, territoryRoom, 100, 100)

	// Put Activity into Casting state to simulate the previously-stuck state.
	mob.Character.Activity = activity.NewMachine()
	_ = mob.Character.Activity.TransitionToCasting(
		activity.CastingData{SpellId: "fold-recall"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin},
	)

	// Set the fold-anchor-room in MiscData (matches Tova's sanctuary).
	mob.Character.SetMiscData("fold-anchor-room", sanctuaryRoom)

	// Register mob in the territory room's mob list so RemoveMob works.
	rooms.LoadRoom(territoryRoom).AddMob(mob.InstanceId)

	p := forager.ProfileFor(371)
	if p == nil {
		t.Fatal("no forager profile for mob 371")
	}

	state := NewBehaviorState()
	state.Set(keyForagerState, forager.StateRecalling.Name())

	ctx := &EvalContext{
		InstanceId: 8240,
		RoomId:     territoryRoom,
		MobState:   state,
	}

	res := tickForagerRecalling(p, mob, ctx)
	if res != Success {
		t.Fatalf("tickForagerRecalling = %v, want Success", res)
	}

	// Mob should now be at the sanctuary room.
	if mob.Character.RoomId != sanctuaryRoom {
		t.Errorf("after teleport: mob.Character.RoomId = %d, want %d",
			mob.Character.RoomId, sanctuaryRoom)
	}

	// Activity machine must have been cleared back to Free.
	if mob.Character.Activity != nil && !mob.Character.Activity.IsFree() {
		t.Errorf("Activity.State() = %v, want Free (stale cast not cleared)",
			mob.Character.Activity.State())
	}

	// State should still be "recalling" — the sanctuary path fires on the
	// next tick, not this one.
	if got := state.GetString(keyForagerState); got != forager.StateRecalling.Name() {
		t.Errorf("forager_state = %q, want %q (premature state transition)",
			got, forager.StateRecalling.Name())
	}
}

// ---- try_store_excess tests ----

// TestTryStoreExcess_RegisteredInActionRegistry verifies the action is
// registered in the global registry.
func TestTryStoreExcess_RegisteredInActionRegistry(t *testing.T) {
	if _, ok := actionRegistry["try_store_excess"]; !ok {
		t.Fatal("try_store_excess not registered in actionRegistry")
	}
}

// TestTryStoreExcess_NoChestRoomParam verifies that omitting the required
// chest_room param returns Failure.
func TestTryStoreExcess_NoChestRoomParam(t *testing.T) {
	mob := buildForagerMob(t, 8250, 371, 4198, 100, 100)
	mob.Character.Items = append(mob.Character.Items, items.New(40021))

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	res := actTryStoreExcess(map[string]any{}, ctx)
	if res != Failure {
		t.Errorf("expected Failure with missing chest_room param, got %v", res)
	}
}

// TestTryStoreExcess_NoItems verifies that an empty satchel returns Failure
// (nothing to deposit).
func TestTryStoreExcess_NoItems(t *testing.T) {
	mob := buildForagerMob(t, 8251, 371, 4198, 100, 100)
	// Leave mob.Character.Items nil (zero value)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	res := actTryStoreExcess(map[string]any{"chest_room": 4198}, ctx)
	if res != Failure {
		t.Errorf("expected Failure with empty satchel, got %v", res)
	}
}

// TestTryStoreExcess_NotInChestRoom_IssuesPathto verifies that when the mob
// is not in the chest_room, it issues pathto and returns Success (continue
// next tick).
func TestTryStoreExcess_NotInChestRoom_IssuesPathto(t *testing.T) {
	const chestRoom = 9999
	mob := buildForagerMob(t, 8252, 371, 4177 /* not chestRoom */, 100, 100)
	mob.Character.Items = append(mob.Character.Items, items.New(40021))

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	res := actTryStoreExcess(map[string]any{"chest_room": chestRoom}, ctx)
	// Should issue pathto and return Success (multi-tick: one step per tick)
	if res != Success {
		t.Errorf("expected Success when mob is not in chest_room, got %v", res)
	}
}

// TestTryStoreExcess_NilMob_ReturnsFailure verifies that a nil mob instance
// returns Failure safely.
func TestTryStoreExcess_NilMob_ReturnsFailure(t *testing.T) {
	ctx := &EvalContext{InstanceId: 99999}
	res := actTryStoreExcess(map[string]any{"chest_room": 4198}, ctx)
	if res != Failure {
		t.Errorf("expected Failure for nil mob, got %v", res)
	}
}

// TestTryStoreExcess_InChestRoom_NoLockbox_ReturnsFailure verifies that when
// the chest_room has no lockbox container, the action returns Failure.
func TestTryStoreExcess_InChestRoom_NoLockbox_ReturnsFailure(t *testing.T) {
	const chestRoom = 8800

	testRoom := &rooms.Room{
		RoomId:     chestRoom,
		Zone:       "test_zone",
		Containers: map[string]rooms.Container{}, // no lockbox
	}
	cleanRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{chestRoom: testRoom},
		map[string]*rooms.ZoneConfig{},
	)
	defer cleanRooms()

	mob := buildForagerMob(t, 8253, 371, chestRoom, 100, 100)
	mob.Character.Items = append(mob.Character.Items, items.New(40021))

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	res := actTryStoreExcess(map[string]any{"chest_room": chestRoom}, ctx)
	if res != Failure {
		t.Errorf("expected Failure when chest_room has no lockbox, got %v", res)
	}
}

// TestTryStoreExcess_InChestRoom_Locked_IssuesUnlock verifies that when the
// mob is in the chest_room and the lockbox is locked, it issues unlock and
// returns Success.
func TestTryStoreExcess_InChestRoom_Locked_IssuesUnlock(t *testing.T) {
	const chestRoom = 8801

	// Difficulty > 0 + UnlockedRound == 0 => IsLocked() == true
	lockedContainer := rooms.Container{}
	lockedContainer.Lock.Difficulty = 3

	testRoom := &rooms.Room{
		RoomId: chestRoom,
		Zone:   "test_zone",
		Containers: map[string]rooms.Container{
			"lockbox": lockedContainer,
		},
	}
	cleanRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{chestRoom: testRoom},
		map[string]*rooms.ZoneConfig{},
	)
	defer cleanRooms()

	mob := buildForagerMob(t, 8254, 371, chestRoom, 100, 100)
	mob.Character.Items = append(mob.Character.Items, items.New(40021))

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	res := actTryStoreExcess(map[string]any{"chest_room": chestRoom}, ctx)
	// Issues unlock, returns Success (next tick will proceed past unlock step)
	if res != Success {
		t.Errorf("expected Success when lockbox is locked (issues unlock), got %v", res)
	}
}

// TestTryStoreExcess_InChestRoom_Unlocked_IssuesPutAndLock verifies that
// when in the chest_room with an unlocked lockbox, the action issues put
// commands for all items and a lock command, returning Success.
func TestTryStoreExcess_InChestRoom_Unlocked_IssuesPutAndLock(t *testing.T) {
	const chestRoom = 8802

	// Unlocked container (default lock is unlocked: IsLocked() = false when
	// Difficulty == 0 and RotationSeed == 0).
	testRoom := &rooms.Room{
		RoomId: chestRoom,
		Zone:   "test_zone",
		Containers: map[string]rooms.Container{
			"lockbox": {}, // default: unlocked
		},
	}
	cleanRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{chestRoom: testRoom},
		map[string]*rooms.ZoneConfig{},
	)
	defer cleanRooms()

	mob := buildForagerMob(t, 8255, 371, chestRoom, 100, 100)
	mob.Character.Items = append(mob.Character.Items,
		items.New(40021),
		items.New(40021),
	)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	res := actTryStoreExcess(map[string]any{"chest_room": chestRoom}, ctx)
	// Issues put+lock, returns Success
	if res != Success {
		t.Errorf("expected Success when issuing put+lock commands, got %v", res)
	}
}

// TestTickForagerRecalling_AtSanctuaryTransitionsToResting verifies the
// at-sanctuary path: forager in Recalling state with cargo arrives at her
// sanctuary, dumps satchel, and transitions to Resting.
func TestTickForagerRecalling_AtSanctuaryTransitionsToResting(t *testing.T) {
	const sanctuaryRoom = 4198

	testRoom := &rooms.Room{
		RoomId: sanctuaryRoom,
		Zone:   "stillwater",
		Title:  "Stillwater Temple Sanctuary",
		Exits:  map[string]exit.RoomExit{},
		Containers: map[string]rooms.Container{
			"lockbox": {
				Lock: gamelock.Lock{
					Difficulty:   3,
					RotationSeed: 1,
				},
			},
		},
	}
	cleanRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{sanctuaryRoom: testRoom},
		map[string]*rooms.ZoneConfig{},
	)
	defer cleanRooms()

	// Mob 371 = Tova (Marsh forager). Already at sanctuary.
	mob := buildForagerMob(t, 8241, 371, sanctuaryRoom, 100, 100)

	// Give her one item in her satchel.
	mob.Character.Items = append(mob.Character.Items, items.New(40021))

	p := forager.ProfileFor(371)
	if p == nil {
		t.Fatal("no forager profile for mob 371")
	}

	state := NewBehaviorState()
	state.Set(keyForagerState, forager.StateRecalling.Name())

	ctx := &EvalContext{
		InstanceId: 8241,
		RoomId:     sanctuaryRoom,
		MobState:   state,
	}

	res := tickForagerRecalling(p, mob, ctx)
	if res != Success {
		t.Fatalf("tickForagerRecalling = %v, want Success", res)
	}

	// State should have transitioned to Resting.
	if got := state.GetString(keyForagerState); got != forager.StateResting.Name() {
		t.Errorf("forager_state = %q, want %q", got, forager.StateResting.Name())
	}

	// Satchel should be empty (dumped to lockbox).
	if len(mob.Character.Items) != 0 {
		t.Errorf("satchel after dump = %d items, want 0", len(mob.Character.Items))
	}
}
