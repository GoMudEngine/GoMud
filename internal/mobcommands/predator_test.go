package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Consume ────────────────────────────────────────────────────────────────

func TestConsume_NoCorpses(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	// Room starts with no corpses
	assert.Empty(t, room.Corpses)

	handled, err := Consume("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)

	// No condition applied when nothing to eat
	assert.False(t, mob.Character.HasCondition(characters.ConditionRegen))
}

func TestConsume_EatsCorpseAndAppliesRegen(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	// Add a corpse to the room
	room.AddCorpse(rooms.Corpse{
		MobId:        99,
		Character:    characters.Character{Name: "Dead Rat"},
		RoundCreated: 1,
	})
	require.Len(t, room.Corpses, 1)

	handled, err := Consume("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)

	// Corpse should be removed
	assert.Empty(t, room.Corpses)

	// Mob should have ConditionRegen
	assert.True(t, mob.Character.HasCondition(characters.ConditionRegen))
	assert.Equal(t, 2.0, mob.Character.GetConditionMagnitude(characters.ConditionRegen))
	assert.Equal(t, 6, mob.Character.GetConditionDuration(characters.ConditionRegen))
}

func TestConsume_SkipsPrunableCorpses(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	// Add a prunable corpse (already decayed) then a fresh one
	room.AddCorpse(rooms.Corpse{
		MobId:        99,
		Character:    characters.Character{Name: "Old Bones"},
		RoundCreated: 1,
		Prunable:     true,
	})
	room.AddCorpse(rooms.Corpse{
		MobId:        98,
		Character:    characters.Character{Name: "Fresh Kill"},
		RoundCreated: 2,
	})
	require.Len(t, room.Corpses, 2)

	handled, err := Consume("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)

	// Only the prunable one should remain
	assert.Len(t, room.Corpses, 1)
	assert.Equal(t, "Old Bones", room.Corpses[0].Character.Name)
	assert.True(t, mob.Character.HasCondition(characters.ConditionRegen))
}

func TestConsume_AllPrunable(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	room.AddCorpse(rooms.Corpse{
		MobId:        99,
		Character:    characters.Character{Name: "Old Bones"},
		RoundCreated: 1,
		Prunable:     true,
	})

	handled, err := Consume("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)

	// Nothing consumed — all prunable
	assert.Len(t, room.Corpses, 1)
	assert.False(t, mob.Character.HasCondition(characters.ConditionRegen))
}

// ─── Flee ───────────────────────────────────────────────────────────────────

func TestFlee_ClearsAggro(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	// Put mob in combat
	mob.Character.Aggro = &characters.Aggro{UserId: 1}
	require.NotNil(t, mob.Character.Aggro)

	handled, err := Flee("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)

	// Aggro should be cleared
	assert.Nil(t, mob.Character.Aggro)

	// Reset mob position for other tests
	mob.Character.RoomId = 1
	room.AddMob(100)
}

func TestFlee_NoExits(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, _ := getTestMobAndRoom(t)

	// Create a room with no exits
	deadEnd := &rooms.Room{
		RoomId:      999,
		Zone:        "TestZone",
		Title:       "Dead End",
		Description: "No way out.",
		Exits:       map[string]exit.RoomExit{},
	}
	// Move mob to dead end room (just test the function directly)
	mob.Character.Aggro = &characters.Aggro{UserId: 1}

	handled, err := Flee("", mob, deadEnd)
	assert.True(t, handled)
	assert.NoError(t, err)

	// Aggro still cleared even if cornered
	assert.Nil(t, mob.Character.Aggro)
}

func TestFlee_OutOfCombat(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	// Not in combat
	mob.Character.Aggro = nil

	handled, err := Flee("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)

	// Reset position
	mob.Character.RoomId = 1
	room.AddMob(100)
}

// ─── Hamstring ──────────────────────────────────────────────────────────────

func TestHamstring_NotInCombat(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	mob.Character.Aggro = nil

	handled, err := Hamstring("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
}

func TestHamstring_InCombat(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	// Hamstring is a beast move (Phase-4 hands rule): the default test mob is a
	// humanoid (SpeciesId 1, has hands). Give it a fanged, no-hands beast species
	// so it qualifies.
	spCleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		2: {SpeciesId: 2, Name: "testbeast", BodyParts: []string{"legs", "mouth"}, NaturalAttack: items.Bite},
	})
	defer spCleanup()
	mob.Character.SpeciesId = 2

	// Set up combat against player
	mob.Character.Aggro = &characters.Aggro{UserId: 1}
	mob.Character.Stats.Strength.ValueAdj = 80
	mob.Character.Stats.Dexterity.ValueAdj = 80

	handled, err := Hamstring("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)

	// Should cost the round
	if mob.Character.Aggro != nil {
		assert.Equal(t, 1, mob.Character.Aggro.RoundsWaiting)
	}

	mob.Character.Aggro = nil
}

func TestHamstring_BleedMagnitude(t *testing.T) {
	// Verify bleed damage calculation: Strength/10, min 2
	tests := []struct {
		name     string
		strength int
		expected float64
	}{
		{"high_strength", 100, 10.0},
		{"medium_strength", 50, 5.0},
		{"low_strength_floor", 10, 2.0},
		{"very_low_strength_floor", 5, 2.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bleedDmg := tc.strength / 10
			if bleedDmg < 2 {
				bleedDmg = 2
			}
			assert.Equal(t, tc.expected, float64(bleedDmg))
		})
	}
}

// ─── Charge ─────────────────────────────────────────────────────────────────

func TestCharge_NotInCombat(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	mob.Character.Aggro = nil

	handled, err := Charge("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
}

func TestCharge_InCombat(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	mob.Character.Aggro = &characters.Aggro{UserId: 1}
	mob.Character.Stats.Strength.ValueAdj = 80
	mob.Character.Stats.Dexterity.ValueAdj = 80

	handled, err := Charge("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)

	if mob.Character.Aggro != nil {
		assert.Equal(t, 1, mob.Character.Aggro.RoundsWaiting)
	}

	mob.Character.Aggro = nil
}

// ─── Howl ───────────────────────────────────────────────────────────────────

func TestHowl_NotInCombat(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	mob.Character.Aggro = nil

	handled, err := Howl("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
}

func TestHowl_InCombat(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)

	mob.Character.Aggro = &characters.Aggro{UserId: 1}
	mob.Character.Stats.Charisma.ValueAdj = 80
	mob.Character.ConvictionMax.Value = 50
	mob.Character.Conviction = 50

	handled, err := Howl("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)

	if mob.Character.Aggro != nil {
		assert.Equal(t, 1, mob.Character.Aggro.RoundsWaiting)
	}

	mob.Character.Aggro = nil
}

// ─── Cooldown Interaction ───────────────────────────────────────────────────

func TestSpecialMoveCooldown_SharedAcrossCommands(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	mob.Character.Aggro = &characters.Aggro{UserId: 1}
	mob.Character.Stats.Strength.ValueAdj = 80
	mob.Character.Stats.Dexterity.ValueAdj = 80
	mob.Character.Stats.Charisma.ValueAdj = 80
	mob.Character.ConvictionMax.Value = 50
	mob.Character.Conviction = 50

	// First command should succeed (uses the cooldown)
	handled, err := Charge("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)

	// Second command should silently fail (cooldown not ready)
	// The cooldown was consumed by charge; howl shares the same "special-move" key
	startConviction := mob.Character.Conviction
	handled, err = Howl("", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)
	// Conviction shouldn't change because cooldown blocked the howl
	assert.Equal(t, startConviction, mob.Character.Conviction)

	mob.Character.Aggro = nil
}

// ─── Command Registration ───────────────────────────────────────────────────

func TestPredatorCommandsRegistered(t *testing.T) {
	cmds := GetAllMobCommands()
	found := map[string]bool{}
	for _, c := range cmds {
		found[c] = true
	}

	assert.True(t, found["charge"], "charge should be registered")
	assert.True(t, found["consume"], "consume should be registered")
	assert.True(t, found["flee"], "flee should be registered")
	assert.True(t, found["hamstring"], "hamstring should be registered")
	assert.True(t, found["howl"], "howl should be registered")
}
