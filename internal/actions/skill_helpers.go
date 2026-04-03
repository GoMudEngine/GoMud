package actions

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// CalcSneakScore returns the stealth score for a character attempting to hide.
// Higher values make the character harder to detect.
// Formula: Dexterity + SkillMultiplier(skullduggery)*25 + mutation bonus,
// then halved if the character emits light.
func CalcSneakScore(c *characters.Character) float64 {
	score := float64(c.Stats.Dexterity.ValueAdj) +
		combat.SkillMultiplier(c.GetSkillLevel(skills.Skullduggery))*25.0

	// Mutation stealth bonus (e.g. chameleon skin)
	score += mutations.GetStealthBonus(c.Mutations)

	// Emitting light makes it much harder to hide
	if c.HasBuffFlag(buffs.EmitsLight) {
		score *= 0.5
	}

	return score
}

// CalcSearchScore returns the observation score for a character detecting
// hidden things. Higher values mean better detection ability.
// Formula: Perception + SkillMultiplier(search)*25.
func CalcSearchScore(c *characters.Character) float64 {
	return float64(c.Stats.Perception.ValueAdj) +
		combat.SkillMultiplier(c.GetSkillLevel(skills.Search))*25.0
}
