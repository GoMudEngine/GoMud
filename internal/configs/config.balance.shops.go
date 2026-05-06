package configs

// validateShops sets defaults for shop economy, bartering, storage,
// crafting, and recipe discovery fields.
func (b *Balance) validateShops() {
	// ── SHOP ECONOMY ─────────────────────────────────────────────────────────
	if b.ShopBuyRatio <= 0 {
		b.ShopBuyRatio = 0.50
	}
	if b.ShopPriceFloor <= 0 {
		b.ShopPriceFloor = 0.25
	}
	if b.ShopPriceCeiling <= 0 {
		b.ShopPriceCeiling = 5.0
	}
	if b.ShopAbundanceThreshold <= 0 {
		b.ShopAbundanceThreshold = 3.0
	}
	if b.ShopMaterialReserve < 0 {
		b.ShopMaterialReserve = 1
	}
	if b.CrafterIngredientReservePct <= 0 {
		b.CrafterIngredientReservePct = 0.25
	}
	if b.ShopGoldReserveRatio <= 0 {
		b.ShopGoldReserveRatio = 0.50
	}

	// ── BARTERING ────────────────────────────────────────────────────────────
	if b.BarterMaxDiscount <= 0 {
		b.BarterMaxDiscount = 0.15
	}
	if b.BarterMaxBonus <= 0 {
		b.BarterMaxBonus = 0.15
	}

	// ── STORAGE FEES ─────────────────────────────────────────────────────────
	if b.StorageFeePerItem < 0 {
		b.StorageFeePerItem = 1
	}

	// ── CRAFTER MOBS ─────────────────────────────────────────────────────────
	if b.CrafterMaterialRestockRate < 1 {
		b.CrafterMaterialRestockRate = 200
	}
	if b.CrafterRareThreshold < 1 {
		b.CrafterRareThreshold = 3
	}
	if b.RestockCadenceTier50Hours == 0 {
		b.RestockCadenceTier50Hours = 1
	}
	if b.RestockCadenceTier40Hours == 0 {
		b.RestockCadenceTier40Hours = 2
	}
	if b.RestockCadenceTier30Hours == 0 {
		b.RestockCadenceTier30Hours = 6
	}
	if b.RestockCadenceTier20Hours == 0 {
		b.RestockCadenceTier20Hours = 24
	}
	if b.RestockCadenceTier10Days == 0 {
		b.RestockCadenceTier10Days = 5
	}

	// ── CRAFTING ──────────────────────────────────────────────────────────────
	if b.CraftingBaseSuccessChance <= 0 || b.CraftingBaseSuccessChance > 100 {
		b.CraftingBaseSuccessChance = 50
	}
	if b.CraftingSkillBonusPerLevel <= 0 {
		b.CraftingSkillBonusPerLevel = 5
	}
	if b.CraftingMinSuccessChance < 1 {
		b.CraftingMinSuccessChance = 5
	}
	if b.CraftingMaxSuccessChance <= 0 || b.CraftingMaxSuccessChance > 100 {
		b.CraftingMaxSuccessChance = 95
	}

	// ── CRAFT DIFFICULTY ─────────────────────────────────────────────────────
	if b.CraftDifficultyProgressionScale <= 0 {
		b.CraftDifficultyProgressionScale = 0.02
	}

	// ── RECIPE DISCOVERY ─────────────────────────────────────────────────────
	if b.RecipeDiscoveryBaseChance <= 0 {
		b.RecipeDiscoveryBaseChance = 8.0
	}
	if b.RecipeDiscoveryDecayRate <= 0 {
		b.RecipeDiscoveryDecayRate = 0.1
	}

	// ── ECONOMY SCORING ───────────────────────────────────────────────────────
	if b.TtRTargetTier50Hours == 0 {
		b.TtRTargetTier50Hours = 3
	}
	if b.TtRTargetTier40Hours == 0 {
		b.TtRTargetTier40Hours = 6
	}
	if b.TtRTargetTier30Hours == 0 {
		b.TtRTargetTier30Hours = 18
	}
	if b.TtRTargetTier20Days == 0 {
		b.TtRTargetTier20Days = 3
	}
	if b.TtRTargetTier10Days == 0 {
		b.TtRTargetTier10Days = 7
	}
	if b.TtRWindowGameDays == 0 {
		b.TtRWindowGameDays = 7
	}
	if b.LogisticsStuckRounds == 0 {
		b.LogisticsStuckRounds = 3000
	}
	if b.LogisticsStuckMultiplier == 0 {
		b.LogisticsStuckMultiplier = 0.4
	}
	if b.ScoreWeightStock == 0 {
		b.ScoreWeightStock = 0.40
	}
	if b.ScoreWeightInput == 0 {
		b.ScoreWeightInput = 0.30
	}
	if b.ScoreWeightThroughput == 0 {
		b.ScoreWeightThroughput = 0.20
	}
	if b.ScoreWeightShopGold == 0 {
		b.ScoreWeightShopGold = 0.10
	}
}
