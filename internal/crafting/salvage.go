package crafting

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// CalcSalvageChance returns the per-ingredient recovery probability for
// a given salvage skill level. Uses a sqrt curve:
//
//	chance = min + (max - min) * sqrt(clamp(skill, 1, softCap) / softCap)
func CalcSalvageChance(skill int, minChance, maxChance float64, softCap int) float64 {
	if skill < 1 {
		skill = 1
	}
	ratio := float64(skill) / float64(softCap)
	if ratio > 1.0 {
		ratio = 1.0
	}
	return minChance + (maxChance-minChance)*math.Sqrt(ratio)
}

// CalcSalvageRounds determines how many rounds a salvage attempt takes
// based on the total gold value of ingredients.
func CalcSalvageRounds(totalGoldValue int, goldPerRound int, maxRounds int) int {
	if goldPerRound < 1 {
		goldPerRound = 10
	}
	rounds := totalGoldValue / goldPerRound
	if rounds < 1 {
		rounds = 1
	}
	if rounds > maxRounds {
		rounds = maxRounds
	}
	return rounds
}

// RollSalvageReturns rolls each ingredient independently and returns
// the items recovered. Each unit of each ingredient is rolled separately.
func RollSalvageReturns(ingredients []RecipeIngredient, chance float64) []RecipeIngredient {
	var recovered []RecipeIngredient
	for _, ing := range ingredients {
		qty := 0
		for i := 0; i < ing.Quantity; i++ {
			if util.Rand(10000) < int(chance*10000) {
				qty++
			}
		}
		if qty > 0 {
			recovered = append(recovered, RecipeIngredient{
				ItemTag:  ing.ItemTag,
				Quantity: qty,
			})
		}
	}
	return recovered
}

// RollSalvageReturnsFromSpec rolls salvage returns for tagged items
// (items with SalvageReturns on their ItemSpec).
func RollSalvageReturnsFromSpec(returns []items.SalvageReturn, chance float64) []RecipeIngredient {
	var recovered []RecipeIngredient
	for _, ret := range returns {
		qty := 0
		for i := 0; i < ret.Quantity; i++ {
			if util.Rand(10000) < int(chance*10000) {
				qty++
			}
		}
		if qty > 0 {
			recovered = append(recovered, RecipeIngredient{
				ItemTag:  ret.ItemTag,
				Quantity: qty,
			})
		}
	}
	return recovered
}

// CalcIngredientGoldValue sums the gold value of all ingredients in a recipe
// by looking up each component tag's item value.
func CalcIngredientGoldValue(ingredients []RecipeIngredient) int {
	total := 0
	for _, ing := range ingredients {
		if spec := items.FindSpecByComponentTag(ing.ItemTag); spec != nil {
			total += spec.Value * ing.Quantity
		}
	}
	return total
}

// CalcSalvageReturnGoldValue sums the gold value of salvage returns from
// tagged items.
func CalcSalvageReturnGoldValue(returns []items.SalvageReturn) int {
	total := 0
	for _, ret := range returns {
		if spec := items.FindSpecByComponentTag(ret.ItemTag); spec != nil {
			total += spec.Value * ret.Quantity
		}
	}
	return total
}
