package behaviortree

import (
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
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

	// Mob 371 = Tova (Marsh forager). Sanctuary room = 4123.
	mob := buildForagerMob(t, 8200, 371, 4123, 100, 100)
	_ = mob

	state := NewBehaviorState()
	ctx := &EvalContext{
		InstanceId: 8200,
		RoomId:     4123,
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

	// Mob 371, full HP, at sanctuary (4123).
	mob := buildForagerMob(t, 8203, 371, 4123, 100, 100)
	_ = mob

	state := NewBehaviorState()
	state.Set(keyForagerState, forager.StateResting.Name())
	// Set started_round to 0 so dwell has elapsed (current round >= 120).
	state.Set(keyStateStartedRound, "0")
	// Ensure we're past round 120 — util.GetRoundCount() in tests is >= 1.
	// Force it past restingDuration by setting started to a round far in
	// the past.
	state.Set(keyStateStartedRound,
		strconv.FormatUint(util.GetRoundCount()-restingDuration-1, 10))

	ctx := &EvalContext{
		InstanceId: 8203,
		RoomId:     4123, // sanctuary
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
	mob := buildForagerMob(t, 8204, 371, 4123, 80, 100)
	_ = mob

	state := NewBehaviorState()
	state.Set(keyForagerState, forager.StateResting.Name())
	state.Set(keyStateStartedRound,
		strconv.FormatUint(util.GetRoundCount()-restingDuration-1, 10))

	ctx := &EvalContext{
		InstanceId: 8204,
		RoomId:     4123,
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

	mob := buildForagerMob(t, 8206, 371, 4123 /* sanctuary */, 100, 100)
	_ = mob

	state := NewBehaviorState()
	state.Set(keyForagerState, forager.StateRecalling.Name())

	ctx := &EvalContext{
		InstanceId: 8206,
		RoomId:     4123,
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
	mob := buildForagerMob(t, 8207, 371, 4123, 100, 100)
	_ = mob
	ctx := &EvalContext{
		InstanceId: 8207,
		RoomId:     4123,
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
	const sanctuaryRoom = 4123

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

	// Mob 371 = Tova (Marsh forager). Sanctuary room = 4123.
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
		t.Fatal("room 4123 missing")
	}
	box, ok := room.Containers["lockbox"]
	if !ok {
		t.Fatal("room 4123 has no lockbox")
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
