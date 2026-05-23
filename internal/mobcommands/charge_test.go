package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCharge_Decoration_MobTarget verifies that Charge executes cleanly when
// targeting a mob (mob-vs-mob path) — a path not covered by predator_test.go.
func TestCharge_Decoration_MobTarget(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	// Point attacker (mob 100) at mob instance 200 (Merchant, seeded in room 1).
	mob.Character.Aggro = &characters.Aggro{MobInstanceId: 200}
	defer func() { mob.Character.Aggro = nil }()

	mob.Character.Cooldowns = map[string]int{}

	handled, err := Charge("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
}

// TestCharge_Cooldown_Blocks verifies that Charge silently exits when the
// special-move cooldown is active (OnCooldown path).
func TestCharge_Cooldown_Blocks(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	mob.Character.Aggro = &characters.Aggro{UserId: 1}
	defer func() { mob.Character.Aggro = nil }()

	// Mark the cooldown as already active.
	mob.Character.Cooldowns["special-move"] = 5

	handled, err := Charge("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
}

// TestCharge_VanishedTarget verifies that Charge returns gracefully when
// the aggro target no longer exists in the registry (race-condition path).
func TestCharge_VanishedTarget(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	// Point aggro at a non-existent mob instance.
	mob.Character.Aggro = &characters.Aggro{MobInstanceId: 9999}
	defer func() { mob.Character.Aggro = nil }()

	mob.Character.Cooldowns = map[string]int{}

	handled, err := Charge("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
}

// TestCharge_IsRegistered verifies that the "charge" command is registered in
// the mob command registry.
func TestCharge_IsRegistered(t *testing.T) {
	cmds := GetAllMobCommands()
	found := false
	for _, c := range cmds {
		if c == "charge" {
			found = true
			break
		}
	}
	assert.True(t, found, `"charge" must be registered as a mob command`)
}

// TestCharge_AggroRoundsWaiting_HighStats verifies that after a successful
// Charge execution with a high-stat attacker, mob.Character.Aggro.RoundsWaiting
// is set to 1 (consumed round). Tests the refactored delegation path under
// near-guaranteed hit conditions.
func TestCharge_AggroRoundsWaiting_HighStats(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	// Give attacker high stats so the hit roll is very likely to succeed,
	// exercising the full hit-path decoration.
	mob.Character.Stats.Strength.ValueAdj = 200
	mob.Character.Stats.Dexterity.ValueAdj = 200

	mob.Character.Aggro = &characters.Aggro{UserId: 1}
	defer func() { mob.Character.Aggro = nil }()

	mob.Character.Cooldowns = map[string]int{}

	_, err := Charge("", mob, room)
	require.NoError(t, err)

	// After charge fires, aggro.RoundsWaiting must be 1 (round consumed
	// by the special move via actions.ExecuteTrip → RecordAndWait).
	require.NotNil(t, mob.Character.Aggro)
	assert.Equal(t, 1, mob.Character.Aggro.RoundsWaiting,
		"Charge must consume the combat round (RoundsWaiting=1)")
}
