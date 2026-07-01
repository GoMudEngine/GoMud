package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// TestCalcSpellDamage_DampenedSuppressed exercises the real
// calcSpellDamageForCharacter pipeline and asserts that a caster carrying the
// #22 crash-site Dampened flag deals strictly less spell damage than an
// otherwise-identical un-dampened caster. Damage is randomized per roll
// (dice.RollStat), so we sum over many rolls to average out the variance —
// with the default CrashSiteSuppressionFactor (0.35) the dampened total is a
// large, reliable margin below the normal total.
func TestCalcSpellDamage_DampenedSuppressed(t *testing.T) {
	cleanup := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		200: {
			BuffId:       200,
			Name:         "Dampened",
			TriggerCount: 1000000,
			Flags:        []buffs.Flag{buffs.Dampened},
		},
	})
	defer cleanup()

	spellData := &spells.SpellData{
		SpellId:           "sparks",
		DamageMultiplier:  0.8,
		TargetDefenseType: "mental",
	}

	newCaster := func() *characters.Character {
		c := &characters.Character{}
		c.Stats.Willpower.ValueAdj = 150
		c.ConvictionMax.Value = 50
		c.Conviction = 50
		return c
	}
	newTarget := func() *characters.Character {
		tgt := &characters.Character{}
		tgt.HealthMax.Value = 100
		tgt.Stats.Willpower.ValueAdj = 50
		return tgt
	}

	const iters = 200

	var normalSum int
	for i := 0; i < iters; i++ {
		normalSum += calcSpellDamageForCharacter(spellData, newCaster(), newTarget(), 10, false)
	}

	damp := newCaster()
	damp.Buffs = buffs.New()
	if err := damp.AddBuff(200, true); err != nil {
		t.Fatalf("AddBuff(Dampened): %v", err)
	}
	if !damp.HasBuffFlag(buffs.Dampened) {
		t.Fatal("caster should carry the Dampened flag after AddBuff")
	}

	var dampSum int
	for i := 0; i < iters; i++ {
		dampSum += calcSpellDamageForCharacter(spellData, damp, newTarget(), 10, false)
	}

	if dampSum >= normalSum {
		t.Errorf("Dampened caster total %d should be strictly less than normal total %d", dampSum, normalSum)
	}
}
