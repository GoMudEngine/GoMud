package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleCombatRound_HiddenDefenderClearsHiddenState locks in the
// chunk-1 / chunk-4b fix for the "can't seem to find your target"
// targeting bug. Setup:
//   - Mob defender is fully hidden (Awareness FSM = Hidden AND buff #9
//     active — matches the ambusher-spawn pattern used by mobs like
//     thornwall_highwayman with `idlecommands: [sneak]`).
//   - Player attacker has aggro on the mob (as if just executed
//     `attack mob` or `grapple mob`).
//   - handleCombatRound fires for this player→mob pair.
//
// Expected after the call:
//   - mob.Character.IsHidden() == false (Awareness FSM forced Visible
//     via ForceVisible; cascade strips buff #9 too).
//
// Without the fix, the FSM stays at Hidden because the cascade in
// Awareness_Cascades.go only fires on the defender's OWN CombatPhase
// Idle→Engaging transition — which doesn't happen when the defender
// is targeted but never SetAggro's the attacker. The legacy
// CancelCombatBuffs call strips the buff but leaves the FSM stale, so
// IsHidden (FSM-driven post-chunk-1) keeps returning true and the
// IsHidden check at handleCombatRound bails every round.
//
// See bug log 2026-05-16 (highwayman grapple — "can't seem to find
// your target" loop).
func TestHandleCombatRound_HiddenDefenderClearsHiddenState(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Combat lookups dereference species.GetSpecies(SpeciesId) without
	// nil-checking, so seed a minimal Human species and stamp
	// SpeciesId=1 on both combatants.
	cleanupSpecies := species.SeedSpeciesForTest(map[int]*species.Species{
		1: {SpeciesId: 1, Name: "Human", UnarmedName: "fist"},
	})
	defer cleanupSpecies()

	// items.GetAttackMessage infinite-recurses when attackMessages is
	// empty; seed a minimal fixture so the lookup resolves first try.
	cleanupAtkMsgs := items.SeedAttackMessagesForTest(items.MinimalCombatMessageFixture())
	defer cleanupAtkMsgs()

	u := users.GetByUserId(1)
	require.NotNil(t, u)
	u.Character.SpeciesId = 1

	m := mobs.GetInstance(100)
	require.NotNil(t, m)
	m.Character.SpeciesId = 1
	// hooks_test fixture only seeds Position; the other state machines
	// stay nil until Validate runs. Awareness must be non-nil for
	// TransitionToConcealing below.
	m.Character.Validate()

	// Put the mob into the hidden state — both the FSM and the buff.
	// Match the production ambusher path: TransitionToConcealing →
	// ResolveConcealment(true). The Awareness_Cascades observer adds
	// buff #9 automatically on the cascade.
	err := m.Character.Awareness.TransitionToConcealing(
		awareness.ConcealingData{},
		state.TransitionReason{Trigger: "test_setup"},
	)
	require.NoError(t, err, "TransitionToConcealing must succeed for a Visible mob")
	m.Character.Awareness.ResolveConcealment(true, state.TransitionReason{Trigger: "test_setup"})
	require.True(t, m.Character.IsHidden(), "mob must be Hidden before the test runs")

	// Player aggros the mob — exactly what happens after a successful
	// `attack` or `grapple` command. The player's own CombatPhase goes
	// Idle→Engaging here (the cascade fires on the player, not the mob).
	u.Character.SetAggro(0, m.InstanceId, characters.DefaultAttack)

	// Drive the unified handler.
	room1 := rooms.LoadRoom(1)
	require.NotNil(t, room1)
	atk := actions.NewUserActorInRoom(u, room1)
	def := actions.NewMobActorInRoom(m, room1)

	var (
		affPlayers []int
		affMobs    []int
	)
	cfg := configs.GetConfig()
	handleCombatRound(
		atk, def,
		events.NewRound{RoundNumber: 1},
		0, // moonMod
		&cfg,
		&affPlayers,
		&affMobs,
		false, // forceCrit: defender not sleeping
	)

	// Fix invariant: defender's Awareness FSM is no longer Hidden.
	assert.False(t, m.Character.IsHidden(),
		"defender's Awareness FSM should be cleared of Hidden after handleCombatRound — got IsHidden=true")
}
