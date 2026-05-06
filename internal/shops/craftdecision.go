package shops

import (
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// CraftDecision describes what a crafter NPC should do on its next tick.
type CraftDecision struct {
	Action   string  // "craft", "salvage", or "" (nothing to do)
	RecipeId string  // Recipe to craft (if Action == "craft")
	ItemId   int     // Item to salvage (if Action == "salvage")
	Profit   float64 // Expected profit margin (sell value − material cost)
	ForSelf  bool    // Crafting for a self-gear upgrade
}

// MaterialCost computes the opportunity cost of consuming all ingredients
// for recipe. Each ingredient is valued at its current sell price in
// shopInv (using MaxStock/2 as the normalizer for crafted-only items).
// Returns 0 if shopInv is nil.
func MaterialCost(recipe *crafting.RecipeSpec, shopInv *ShopInventory, cfg PricingConfig) float64 {
	if shopInv == nil || recipe == nil {
		return 0
	}
	total := 0.0
	for _, ing := range recipe.Ingredients {
		spec := items.FindSpecByComponentTag(ing.ItemTag)
		if spec == nil {
			continue
		}
		entry := shopInv.GetStock(spec.ItemId)
		current, restockQty := 0, 1
		if entry != nil {
			current = entry.Current
			if entry.RestockQty > 0 {
				restockQty = entry.RestockQty
			} else if entry.MaxStock > 0 {
				restockQty = entry.MaxStock / 2
				if restockQty < 1 {
					restockQty = 1
				}
			}
		}
		price := CalcSellPrice(spec.Value, current, restockQty, cfg)
		total += float64(price) * float64(ing.Quantity)
	}
	return total
}

// ProductValue computes what the output item of recipe is worth at current
// stock levels. Uses MaxStock/2 as the normalizer for crafted goods.
// Returns 0 if shopInv is nil or no output item is defined.
func ProductValue(recipe *crafting.RecipeSpec, shopInv *ShopInventory, cfg PricingConfig) float64 {
	if shopInv == nil || recipe == nil || recipe.Output.ItemId <= 0 {
		return 0
	}
	spec := items.GetItemSpec(recipe.Output.ItemId)
	if spec == nil {
		return 0
	}
	entry := shopInv.GetStock(recipe.Output.ItemId)
	current, restockQty := 0, 1
	if entry != nil {
		current = entry.Current
		if entry.RestockQty > 0 {
			restockQty = entry.RestockQty
		} else if entry.MaxStock > 0 {
			restockQty = entry.MaxStock / 2
			if restockQty < 1 {
				restockQty = 1
			}
		}
	}
	price := CalcSellPrice(spec.Value, current, restockQty, cfg)
	return float64(price) * float64(recipe.Output.Quantity)
}

// HasMaterialsWithReserve returns true if shopInv contains all ingredients
// for recipe while keeping at least reserve units of each ingredient in stock.
func HasMaterialsWithReserve(recipe *crafting.RecipeSpec, shopInv *ShopInventory, reserve int) bool {
	if shopInv == nil || recipe == nil {
		return false
	}
	for _, ing := range recipe.Ingredients {
		spec := items.FindSpecByComponentTag(ing.ItemTag)
		if spec == nil {
			return false
		}
		entry := shopInv.GetStock(spec.ItemId)
		available := 0
		if entry != nil {
			available = entry.Current - reserve
		}
		if available < ing.Quantity {
			return false
		}
	}
	return true
}

// HasMaterialsWithReservePct returns true if shopInv contains all ingredients
// for recipe while keeping a per-ingredient reserve floor of
// max(1, int(MaxStock * reservePct)) units of each ingredient in stock.
// This prevents the crafter from draining its own supply to a level where
// players cannot buy.
func HasMaterialsWithReservePct(recipe *crafting.RecipeSpec, shopInv *ShopInventory, reservePct float64) bool {
	if shopInv == nil || recipe == nil {
		return false
	}
	for _, ing := range recipe.Ingredients {
		spec := items.FindSpecByComponentTag(ing.ItemTag)
		if spec == nil {
			return false
		}
		entry := shopInv.GetStock(spec.ItemId)
		if entry == nil {
			return false
		}
		reserve := int(float64(entry.MaxStock) * reservePct)
		if reserve < 1 {
			reserve = 1
		}
		available := entry.Current - reserve
		if available < ing.Quantity {
			return false
		}
	}
	return true
}

// EvaluateCraftOptions scores all profitable craft recipes and returns the
// most profitable one. Returns nil if no recipe is worth crafting.
//
// A recipe is skipped if:
//   - Output item is at max_stock
//   - Materials are insufficient (respecting reservePct floor)
//   - Product value does not exceed material cost (not profitable)
//
// reservePct is the fraction of each ingredient's MaxStock to keep in
// reserve. The per-ingredient reserve is max(1, int(MaxStock * reservePct)).
func EvaluateCraftOptions(recipes []string, shopInv *ShopInventory, cfg PricingConfig, reservePct float64) *CraftDecision {
	if shopInv == nil {
		return nil
	}

	var best *CraftDecision
	for _, recipeId := range recipes {
		recipe := crafting.GetRecipe(recipeId)
		if recipe == nil || recipe.Output.ItemId <= 0 {
			continue
		}

		// Skip enchanting recipes — they don't have normal output items
		if crafting.IsEnchantingRecipe(recipe) {
			continue
		}

		// Skip if output is already at max stock
		entry := shopInv.GetStock(recipe.Output.ItemId)
		if entry != nil && entry.Current >= entry.MaxStock {
			continue
		}

		// Skip if materials are insufficient (with per-ingredient reserve)
		if !HasMaterialsWithReservePct(recipe, shopInv, reservePct) {
			continue
		}

		// Compute profit
		cost := MaterialCost(recipe, shopInv, cfg)
		value := ProductValue(recipe, shopInv, cfg)
		profit := value - cost
		if profit <= 0 {
			continue
		}

		if best == nil || profit > best.Profit {
			best = &CraftDecision{
				Action:   "craft",
				RecipeId: recipeId,
				Profit:   profit,
			}
		}
	}
	return best
}

// EvaluateSalvageOptions finds the most profitable item to break down in
// shopInv. Only items with Current > 1 (keep at least one) and
// RestockQty == 0 (crafted-only goods) are candidates. The item must have
// SalvageReturns defined on its spec.
// Returns nil if no profitable salvage opportunity exists.
func EvaluateSalvageOptions(shopInv *ShopInventory, cfg PricingConfig) *CraftDecision {
	if shopInv == nil {
		return nil
	}

	var best *CraftDecision
	for _, entry := range shopInv.Stock {
		// Must have surplus and be NPC-crafted only
		if entry.Current <= 1 || entry.RestockQty != 0 {
			continue
		}

		spec := items.GetItemSpec(entry.ItemId)
		if spec == nil || len(spec.SalvageReturns) == 0 {
			continue
		}

		// Value of the item being broken down (at current stock with one extra)
		restockNorm := entry.MaxStock / 2
		if restockNorm < 1 {
			restockNorm = 1
		}
		itemPrice := float64(CalcSellPrice(spec.Value, entry.Current, restockNorm, cfg))

		// Value of materials recovered
		returnValue := 0.0
		for _, ret := range spec.SalvageReturns {
			matSpec := items.FindSpecByComponentTag(ret.ItemTag)
			if matSpec == nil {
				continue
			}
			matEntry := shopInv.GetStock(matSpec.ItemId)
			matCurrent, matRestock := 0, 1
			if matEntry != nil {
				matCurrent = matEntry.Current
				if matEntry.RestockQty > 0 {
					matRestock = matEntry.RestockQty
				} else if matEntry.MaxStock > 0 {
					matRestock = matEntry.MaxStock / 2
					if matRestock < 1 {
						matRestock = 1
					}
				}
			}
			matPrice := CalcSellPrice(matSpec.Value, matCurrent, matRestock, cfg)
			returnValue += float64(matPrice) * float64(ret.Quantity)
		}

		profit := returnValue - itemPrice
		if profit <= 0 {
			continue
		}

		if best == nil || profit > best.Profit {
			best = &CraftDecision{
				Action: "salvage",
				ItemId: entry.ItemId,
				Profit: profit,
			}
		}
	}
	return best
}
