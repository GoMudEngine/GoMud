package crafting

import "github.com/GoMudEngine/GoMud/internal/configs"

// EnchantSalvageBands holds band thresholds (by potion alchemy-recipe
// skill_minimum) and per-mat drop chances for tiered potion salvage.
type EnchantSalvageBands struct {
	Band2Min, Band3Min, Band4Min                    int
	Band2SettingPct                                 int
	Band3SettingPct, Band3CatalystPct               int
	Band4CatalystPct, Band4SettingPct, Band4CorePct int
}

// EnchantSalvageYieldWith is the pure, testable core: maps a potion's recipe
// skill_minimum to enchanting-mat returns, rolling per-band chances via the
// injected roll() ([0,1)). qtyBonus (0+) bumps the binding-paste floor. Bands
// 1-3 always include a binding-paste floor; band 4 rolls rare mats and falls
// back to binding-paste if all rolls miss.
func EnchantSalvageYieldWith(skillMin int, roll func() float64, qtyBonus int, b EnchantSalvageBands) []RecipeIngredient {
	hit := func(pct int) bool { return roll()*100.0 < float64(pct) }
	var out []RecipeIngredient
	switch {
	case skillMin >= b.Band4Min:
		if hit(b.Band4CatalystPct) {
			out = append(out, RecipeIngredient{ItemTag: "mutation-catalyst", Quantity: 1})
		}
		if hit(b.Band4SettingPct) {
			out = append(out, RecipeIngredient{ItemTag: "chrysalis-setting", Quantity: 1})
		}
		if hit(b.Band4CorePct) {
			out = append(out, RecipeIngredient{ItemTag: "chrysalis-core", Quantity: 1})
		}
	case skillMin >= b.Band3Min:
		out = append(out, RecipeIngredient{ItemTag: "binding-paste", Quantity: 1 + qtyBonus})
		if hit(b.Band3SettingPct) {
			out = append(out, RecipeIngredient{ItemTag: "chrysalis-setting", Quantity: 1})
		}
		if hit(b.Band3CatalystPct) {
			out = append(out, RecipeIngredient{ItemTag: "mutation-catalyst", Quantity: 1})
		}
	case skillMin >= b.Band2Min:
		out = append(out, RecipeIngredient{ItemTag: "binding-paste", Quantity: 1 + qtyBonus})
		if hit(b.Band2SettingPct) {
			out = append(out, RecipeIngredient{ItemTag: "chrysalis-setting", Quantity: 1})
		}
	default:
		out = append(out, RecipeIngredient{ItemTag: "binding-paste", Quantity: 1 + qtyBonus})
	}
	if len(out) == 0 {
		out = append(out, RecipeIngredient{ItemTag: "binding-paste", Quantity: 1})
	}
	return out
}

// EnchantSalvageYield resolves a potion item id to its recipe skill_minimum and
// applies the configured bands. Unknown/no-recipe potions fall to band 1.
func EnchantSalvageYield(potionItemId int, roll func() float64, qtyBonus int) []RecipeIngredient {
	skillMin := 0
	if r := GetRecipeByOutputItemId(potionItemId); r != nil {
		skillMin = r.SkillMinimum
	}
	c := configs.GetBalanceConfig()
	bands := EnchantSalvageBands{
		Band2Min:         int(c.EnchantSalvageBand2Min),
		Band3Min:         int(c.EnchantSalvageBand3Min),
		Band4Min:         int(c.EnchantSalvageBand4Min),
		Band2SettingPct:  int(c.EnchantSalvageBand2SettingPct),
		Band3SettingPct:  int(c.EnchantSalvageBand3SettingPct),
		Band3CatalystPct: int(c.EnchantSalvageBand3CatalystPct),
		Band4CatalystPct: int(c.EnchantSalvageBand4CatalystPct),
		Band4SettingPct:  int(c.EnchantSalvageBand4SettingPct),
		Band4CorePct:     int(c.EnchantSalvageBand4CorePct),
	}
	return EnchantSalvageYieldWith(skillMin, roll, qtyBonus, bands)
}
