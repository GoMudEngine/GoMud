package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedGolemSpell registers a raise-golem-shaped summon spell so the backfill
// can resolve a base reserve from SummonMobId. Mirrors the live
// raise-golem.yaml numbers (mob 305, base reserve 440).
func seedGolemSpell() func() {
	return spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"raise-golem": {
			SpellId:                 "raise-golem",
			Name:                    "Raise Flesh Golem",
			SummonMobId:             305,
			SummonConvictionReserve: 440,
		},
	})
}

// Companions saved before the Conviction economy shipped (2026-07-13) have no
// conviction_reserve in their save and load as 0 — they'd sustain for free
// forever. The login backfill must stamp them with what they'd cost today.
func TestBackfillCompanionReserves_LegacyGolem(t *testing.T) {
	cleanup := seedGolemSpell()
	defer cleanup()

	ch := characters.New()
	ch.Skills[string(skills.Manifestation)] = 48
	require.True(t, ch.AddCompanion(characters.CompanionInfo{
		MobId: 305, InstanceId: 1, SourceType: characters.CompanionRaised,
		Name: "a flesh golem", ConvictionReserve: 0, // legacy record
	}))
	require.True(t, ch.AddCompanion(characters.CompanionInfo{
		MobId: 305, InstanceId: 2, SourceType: characters.CompanionRaised,
		Name: "a flesh golem", ConvictionReserve: 0, // legacy record
	}))

	changed := backfillCompanionReserves(ch)

	assert.True(t, changed, "legacy companions present -> backfill must report a change")
	// Meirok calibration: 440 * (1 - 48*0.01) = 228.8 -> 229 each.
	assert.Equal(t, 229, ch.Companions[0].ConvictionReserve)
	assert.Equal(t, 229, ch.Companions[1].ConvictionReserve)
}

func TestBackfillCompanionReserves_ExistingReserveUntouched(t *testing.T) {
	cleanup := seedGolemSpell()
	defer cleanup()

	ch := characters.New()
	ch.Skills[string(skills.Manifestation)] = 48
	require.True(t, ch.AddCompanion(characters.CompanionInfo{
		MobId: 305, InstanceId: 1, Name: "a flesh golem", ConvictionReserve: 440,
	}))

	changed := backfillCompanionReserves(ch)

	assert.False(t, changed, "no legacy companions -> no change")
	assert.Equal(t, 440, ch.Companions[0].ConvictionReserve,
		"a snapshotted reserve must never be recomputed")
}

func TestBackfillCompanionReserves_UnknownMobUsesDefault(t *testing.T) {
	cleanup := seedGolemSpell()
	defer cleanup()

	ch := characters.New()
	require.True(t, ch.AddCompanion(characters.CompanionInfo{
		MobId: 424242, InstanceId: 1, Name: "a mystery", ConvictionReserve: 0,
	}))

	changed := backfillCompanionReserves(ch)

	assert.True(t, changed)
	// No summon spell matches -> CompanionReserveDefault (350), no skill reduction.
	want := ch.CalcCompanionReserve(int(configs.GetBalanceConfig().CompanionReserveDefault))
	assert.Equal(t, want, ch.Companions[0].ConvictionReserve)
	assert.Greater(t, ch.Companions[0].ConvictionReserve, 0)
}

func TestLegacyCompanionBaseReserve_SpecialMobs(t *testing.T) {
	cleanup := seedGolemSpell()
	defer cleanup()

	cfg := configs.GetBalanceConfig()
	assert.Equal(t, 440, legacyCompanionBaseReserve(305), "spell-summoned mob uses the spell's reserve")
	assert.Equal(t, int(cfg.HomunculusConvictionReserve), legacyCompanionBaseReserve(homunculusMobId))
	assert.Equal(t, broodFloorReserve, legacyCompanionBaseReserve(broodSpawnMobId))
	assert.Equal(t, int(cfg.CompanionReserveDefault), legacyCompanionBaseReserve(424242))
}
