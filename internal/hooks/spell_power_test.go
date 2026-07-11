package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

func TestCalcSpellDamage_SpellPowerAmplifies(t *testing.T) {
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"ether-gland": {MutationId: "ether-gland", Name: "Ether Gland", Rarity: 4,
			Pros: []mutations.MutationEffect{{Type: "spell_power", Value: 0.5}}},
	})
	defer cleanup()

	spell := &spells.SpellData{Name: "Test Bolt", DamageMultiplier: 1.0, TargetDefenseType: "magical"}
	caster := &characters.Character{}
	caster.Stats.Willpower.ValueAdj = 100
	caster.Conviction, caster.ConvictionMax.Value = 100, 100
	target := &characters.Character{}
	target.HealthMax.Value = 1000

	// calcSpellDamageForCharacter applies dice.RollStat variance, so average
	// over many rolls to compare means deterministically.
	avg := func(c *characters.Character) int {
		total := 0
		for i := 0; i < 300; i++ {
			total += calcSpellDamageForCharacter(spell, c, target, 0, false)
		}
		return total
	}
	base := avg(caster)
	caster.Mutations = map[string]int{"ether-gland": 1}
	boosted := avg(caster)

	if !(boosted > base) {
		t.Fatalf("spell_power should raise average spell damage: base=%d boosted=%d", base, boosted)
	}
}
