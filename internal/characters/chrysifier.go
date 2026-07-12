package characters

import (
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// CraftMaterialsSaved reports whether a craft should PRESERVE its ingredients
// this time instead of consuming them. The Chrysifier's Provident Hands grants
// a per-craft chance (craft_material_discount) to waste no materials at all —
// an uncannily efficient craft. Returns false (materials consumed) when the
// character has no such mutation.
func (c *Character) CraftMaterialsSaved() bool {
	save := mutations.GetCraftMaterialDiscount(c.Mutations)
	if save <= 0 {
		return false
	}
	if save >= 1 {
		return true
	}
	return util.Rand(100) < int(save*100)
}

// CraftQualityLevel returns the effective craft skill level to stamp on a
// crafted item, lifting the base level by the Chrysifier's Faithwrought
// craft-quality bonus (conviction in one's craft makes the work finer).
func (c *Character) CraftQualityLevel(baseSkill int) int {
	bonus := mutations.GetCraftQualityBonus(c.Mutations)
	if bonus <= 0 {
		return baseSkill
	}
	return baseSkill + int(float64(baseSkill)*bonus+0.5)
}
