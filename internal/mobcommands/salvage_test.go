package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Salvage mob command ─────────────────────────────────────────────────────

// buildSalvageFixtures wires up:
//   - A forager mob instance (instanceId 300) in room 1 with salvage skill 30.
//   - A spec for the prey mob (mobId 50, group "animal") used for corpses.
//   - Item specs for "leather-strip" and "sinew" (component tags match the
//     corpse_salvage table defined in internal/crafting/corpse_salvage.go).
//
// Returns the forager mob, room 1, and a cleanup func.
func buildSalvageFixtures(t *testing.T) (*mobs.Mob, *rooms.Room, func()) {
	t.Helper()

	baseCleanup := seedAllRegistries()

	// Prey mob spec — group "animal" triggers the salvage table entry.
	preySpec := &mobs.Mob{
		MobId:  50,
		Zone:   "TestZone",
		Groups: []string{"animal"},
		Character: characters.Character{
			Name: "Marsh Rat",
		},
	}

	// Forager mob instance — salvage skill 30.
	foragerInstance := &mobs.Mob{
		MobId:      3,
		InstanceId: 300,
		HomeRoomId: 1,
		Character: characters.Character{
			Name:      "Tova",
			RoomId:    1,
			Health:    80,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
			Skills:    map[string]int{string(skills.Salvage): 30},
		},
	}
	foragerInstance.Character.HealthMax.Value = 100
	foragerInstance.Character.StaminaMax.Value = 50
	foragerInstance.Character.Stamina = 50
	foragerInstance.Character.SpeciesId = 1

	cleanupExtraMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{50: preySpec},
		map[int]*mobs.Mob{300: foragerInstance},
	)

	// Seed salvage-return item specs. Component tags must match
	// crafting.corpseSalvageTable (leather-strip, sinew).
	cleanupExtraItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		200: {ItemId: 200, Name: "Leather Strip", ComponentTag: "leather-strip", Value: 5},
		201: {ItemId: 201, Name: "Sinew", ComponentTag: "sinew", Value: 3},
	})

	mob := mobs.GetInstance(300)
	require.NotNil(t, mob, "forager instance 300 must exist")
	room := rooms.LoadRoom(1)
	require.NotNil(t, room, "room 1 must exist")
	room.AddMob(300)

	return mob, room, func() {
		cleanupExtraItems()
		cleanupExtraMobs()
		baseCleanup()
	}
}

// buildAnimalCorpse creates a non-prunable animal corpse for room use.
func buildAnimalCorpse() rooms.Corpse {
	return rooms.Corpse{
		MobId:        50,
		Character:    characters.Character{Name: "Marsh Rat"},
		RoundCreated: 10,
	}
}

// ── Test 1: no corpse in room ────────────────────────────────────────────────

// Salvage returns handled=true with no side-effects when the room is empty.
// This is the critical case that prevents the "looks confused" emote fallback.
func TestSalvage_NoCorpse_HandledTrueSilently(t *testing.T) {
	mob, room, cleanup := buildSalvageFixtures(t)
	defer cleanup()

	// Ensure no corpses.
	require.Empty(t, room.Corpses)

	initialItemCount := len(mob.Character.Items)

	handled, err := Salvage("corpse", mob, room)

	assert.True(t, handled, "must return handled=true to silence confused-emote")
	assert.NoError(t, err)
	assert.Empty(t, room.Corpses, "room should still have no corpses")
	assert.Equal(t, initialItemCount, len(mob.Character.Items),
		"mob inventory must be unchanged when no corpse present")
}

// ── Test 2: corpse with no salvage spec (non-eligible group) ─────────────────

// Corpse with a group not in the salvage table (e.g., "undead") should be
// skipped; handler returns handled=true and the corpse remains in the room.
func TestSalvage_CorpseNotEligible_SkippedSilently(t *testing.T) {
	baseCleanup := seedAllRegistries()
	defer baseCleanup()

	// Seed an "undead" prey spec — NOT in the salvage table.
	undeadSpec := &mobs.Mob{
		MobId:  51,
		Zone:   "TestZone",
		Groups: []string{"undead"},
		Character: characters.Character{Name: "Zombie"},
	}
	foragerInstance := &mobs.Mob{
		MobId:      3,
		InstanceId: 300,
		HomeRoomId: 1,
		Character: characters.Character{
			Name:      "Tova",
			RoomId:    1,
			Health:    80,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
			Skills:    map[string]int{string(skills.Salvage): 30},
		},
	}
	foragerInstance.Character.HealthMax.Value = 100
	foragerInstance.Character.StaminaMax.Value = 50
	foragerInstance.Character.Stamina = 50
	foragerInstance.Character.SpeciesId = 1

	cleanupMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{51: undeadSpec},
		map[int]*mobs.Mob{300: foragerInstance},
	)
	defer cleanupMobs()

	mob := mobs.GetInstance(300)
	require.NotNil(t, mob)
	room := rooms.LoadRoom(1)
	require.NotNil(t, room)
	room.AddMob(300)

	// Add an ineligible corpse.
	room.AddCorpse(rooms.Corpse{
		MobId:        51,
		Character:    characters.Character{Name: "Zombie"},
		RoundCreated: 5,
	})
	require.Len(t, room.Corpses, 1)

	initialItemCount := len(mob.Character.Items)

	handled, err := Salvage("corpse", mob, room)

	assert.True(t, handled, "non-eligible corpse must still return handled=true")
	assert.NoError(t, err)
	assert.Len(t, room.Corpses, 1, "ineligible corpse must remain in room")
	assert.Equal(t, initialItemCount, len(mob.Character.Items),
		"mob inventory must not change for ineligible corpse")
}

// ── Test 3: eligible corpse, zero-skill mob — corpse consumed, no yield ──────

// Even at skill 0 the corpse is always removed. Inventory is unchanged but the
// corpse is gone — matches "item always consumed" semantics from CLAUDE.md.
func TestSalvage_ZeroSkill_CorpseConsumedNoYield(t *testing.T) {
	mob, room, cleanup := buildSalvageFixtures(t)
	defer cleanup()

	// Override skill to 0.
	mob.Character.Skills[string(skills.Salvage)] = 0

	room.AddCorpse(buildAnimalCorpse())
	require.Len(t, room.Corpses, 1)

	initialItemCount := len(mob.Character.Items)

	handled, err := Salvage("corpse", mob, room)

	assert.True(t, handled)
	assert.NoError(t, err)
	assert.Empty(t, room.Corpses, "corpse must be removed even on zero yield")
	// At skill 0 CalcSalvageChance returns minChance (~0.15); yield is
	// probabilistic so we can only verify inventory doesn't shrink.
	assert.GreaterOrEqual(t, len(mob.Character.Items), initialItemCount,
		"inventory must not shrink")
}

// ── Test 4: eligible corpse, high-skill mob — corpse consumed, yield present ─

// At skill 50 (soft-cap) chance is ~0.85 per item. With 3 total rolls (2 leather
// + 1 sinew) we expect at least one item in most runs. We run 10 attempts and
// require at least one yields something. This is probabilistically sound —
// probability of all 10 attempts returning zero is (0.15^3)^10 ≈ 1.6e-22.
func TestSalvage_HighSkill_YieldExpected(t *testing.T) {
	mob, room, cleanup := buildSalvageFixtures(t)
	defer cleanup()

	mob.Character.Skills[string(skills.Salvage)] = 50

	for attempt := 0; attempt < 10; attempt++ {
		room.AddCorpse(buildAnimalCorpse())
		require.Len(t, room.Corpses, 1)

		before := len(mob.Character.Items)
		handled, err := Salvage("corpse", mob, room)
		assert.True(t, handled)
		assert.NoError(t, err)
		assert.Empty(t, room.Corpses, "corpse must be consumed each attempt")
		if len(mob.Character.Items) > before {
			// At least one attempt yielded mats — success.
			return
		}
	}
	t.Fatal("expected at least one salvage attempt to yield materials at skill 50")
}

// ── Test 5: prunable corpse is skipped ───────────────────────────────────────

// Prunable corpses (already decayed / being removed by the engine) must be
// skipped. A fresh eligible corpse in the same room should still be processed.
func TestSalvage_SkipsPrunableCorpse(t *testing.T) {
	mob, room, cleanup := buildSalvageFixtures(t)
	defer cleanup()

	mob.Character.Skills[string(skills.Salvage)] = 50

	prunable := buildAnimalCorpse()
	prunable.Prunable = true
	fresh := buildAnimalCorpse()
	fresh.RoundCreated = 20

	room.AddCorpse(prunable)
	room.AddCorpse(fresh)
	require.Len(t, room.Corpses, 2)

	handled, err := Salvage("corpse", mob, room)
	assert.True(t, handled)
	assert.NoError(t, err)

	// The fresh corpse should be gone; the prunable one remains.
	assert.Len(t, room.Corpses, 1,
		"only the fresh corpse should be consumed")
	assert.True(t, room.Corpses[0].Prunable,
		"remaining corpse should be the prunable one")
}

// ── Test 6: salvage is registered in the mob command map ─────────────────────

func TestSalvageCommandRegistered(t *testing.T) {
	cmds := GetAllMobCommands()
	found := false
	for _, c := range cmds {
		if c == "salvage" {
			found = true
			break
		}
	}
	assert.True(t, found, "salvage must be registered in the mob command map")
}
