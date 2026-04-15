package combat_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/stretchr/testify/assert"
)

func TestTrySpellDeflection_ReturnsValidMultiplier(t *testing.T) {
	attacker := characters.New()
	attacker.Stats.Willpower.ValueAdj = 100
	attacker.Skills["spellcasting"] = 10

	defender := characters.New()
	defender.Stats.Perception.ValueAdj = 100
	defender.Skills["spellcasting"] = 10

	for i := 0; i < 200; i++ {
		mult := combat.TrySpellDeflection(attacker, defender, 0)
		assert.GreaterOrEqual(t, mult, 0.0)
		assert.LessOrEqual(t, mult, 1.0)
	}
}

func TestTrySpellDeflection_HighDefenderAdvantage(t *testing.T) {
	attacker := characters.New()
	attacker.Stats.Willpower.ValueAdj = 50
	attacker.Skills["spellcasting"] = 1

	defender := characters.New()
	defender.Stats.Perception.ValueAdj = 200
	defender.Skills["spellcasting"] = 40

	deflections := 0
	trials := 500
	for i := 0; i < trials; i++ {
		if combat.TrySpellDeflection(attacker, defender, 0) < 1.0 {
			deflections++
		}
	}
	assert.Greater(t, deflections, trials/2)
}

func TestTryStoicResolve_ReturnsValidMultiplier(t *testing.T) {
	attacker := characters.New()
	attacker.Stats.Charisma.ValueAdj = 100
	attacker.Skills["rhetoric"] = 10

	defender := characters.New()
	defender.Stats.Willpower.ValueAdj = 100
	defender.Skills["rhetoric"] = 10

	for i := 0; i < 200; i++ {
		mult := combat.TryStoicResolve(attacker, defender, 0)
		assert.GreaterOrEqual(t, mult, 0.0)
		assert.LessOrEqual(t, mult, 1.0)
	}
}

func TestTryStoicResolve_HighDefenderAdvantage(t *testing.T) {
	attacker := characters.New()
	attacker.Stats.Charisma.ValueAdj = 50
	attacker.Skills["rhetoric"] = 1

	defender := characters.New()
	defender.Stats.Willpower.ValueAdj = 200
	defender.Skills["rhetoric"] = 40

	resolves := 0
	trials := 500
	for i := 0; i < trials; i++ {
		if combat.TryStoicResolve(attacker, defender, 0) < 1.0 {
			resolves++
		}
	}
	assert.Greater(t, resolves, trials/2)
}

func TestTrySpellDeflection_LowDefenderRarelyDeflects(t *testing.T) {
	attacker := characters.New()
	attacker.Stats.Willpower.ValueAdj = 200
	attacker.Skills["spellcasting"] = 40

	defender := characters.New()
	defender.Stats.Perception.ValueAdj = 50
	defender.Skills["spellcasting"] = 1

	deflections := 0
	trials := 500
	for i := 0; i < trials; i++ {
		if combat.TrySpellDeflection(attacker, defender, 0) < 1.0 {
			deflections++
		}
	}
	assert.Less(t, deflections, trials/2)
}
