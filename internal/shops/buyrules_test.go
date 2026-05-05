package shops

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
)

// makeItem constructs an Item with an inline ItemSpec for testing.
func makeItem(spec items.ItemSpec) items.Item {
	return items.Item{
		ItemId: spec.ItemId,
		Spec:   &spec,
	}
}

func baseShop() *ShopInventory {
	return &ShopInventory{
		Gold:         1000,
		StartingGold: 1000,
		Stock:        []StockEntry{},
	}
}

func TestBuyRules_QuestItemRejected(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId:           1,
		Value:            50,
		QuestToken:       "some-quest-token",
		Type:             items.Object,
		VendorCategories: []string{"alchemy"}, // tagged anyway, but quest rejection wins
	})
	offer := EvaluateBuyRules(item, baseShop(), "", false, DefaultPricingConfig(), nil)
	assert.Equal(t, 0, offer.Price, "quest items must always be rejected")
}

func TestBuyRules_UntaggedItemRejected(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId: 100,
		Value:  20,
		Type:   items.Object,
		// No VendorCategories.
	})
	shop := baseShop()
	shop.CraftSupport = CraftSupportAlchemy
	offer := EvaluateBuyRules(item, shop, "", false, DefaultPricingConfig(), nil)
	assert.Equal(t, 0, offer.Price, "untagged items must be rejected")
}

func TestBuyRules_TagMatchAccepted(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId:           100,
		Value:            20,
		Type:             items.Object,
		VendorCategories: []string{"alchemy"},
	})
	shop := baseShop()
	shop.CraftSupport = CraftSupportAlchemy
	offer := EvaluateBuyRules(item, shop, "", false, DefaultPricingConfig(), nil)
	assert.Greater(t, offer.Price, 0, "tag-matching item should be accepted")
}

func TestBuyRules_TagMismatchRejected(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId:           100,
		Value:            20,
		Type:             items.Object,
		VendorCategories: []string{"blacksmithing"},
	})
	shop := baseShop()
	shop.CraftSupport = CraftSupportAlchemy
	offer := EvaluateBuyRules(item, shop, "", false, DefaultPricingConfig(), nil)
	assert.Equal(t, 0, offer.Price, "alchemist must reject blacksmithing-tagged item")
}

func TestBuyRules_MultiTagMatchAccepted(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId:           100,
		Value:            20,
		Type:             items.Object,
		VendorCategories: []string{"alchemy", "jewelcrafting"},
	})
	shop := baseShop()
	shop.CraftSupport = CraftSupportJewelcrafting // matches the secondary tag
	offer := EvaluateBuyRules(item, shop, "", false, DefaultPricingConfig(), nil)
	assert.Greater(t, offer.Price, 0, "multi-tag item should match on either discipline")
}

func TestBuyRules_GeneralStoreAcceptsAnyTag(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId:           100,
		Value:            20,
		Type:             items.Object,
		VendorCategories: []string{"blacksmithing"},
	})
	shop := baseShop()
	shop.CraftSupport = CraftSupportGeneral
	shop.Gold = 5000
	shop.StartingGold = 5000
	offer := EvaluateBuyRules(item, shop, "", false, DefaultPricingConfig(), nil)
	assert.Greater(t, offer.Price, 0, "general store must accept any tagged item")
}

func TestBuyRules_GeneralStoreRejectsUntagged(t *testing.T) {
	// Even general stores don't buy items with no vendor_categories.
	item := makeItem(items.ItemSpec{
		ItemId: 100,
		Value:  20,
		Type:   items.Object,
	})
	shop := baseShop()
	shop.CraftSupport = CraftSupportGeneral
	offer := EvaluateBuyRules(item, shop, "", false, DefaultPricingConfig(), nil)
	assert.Equal(t, 0, offer.Price, "general store still requires items to carry a tag")
}

func TestBuyRules_UnstockedItem_FlatPrice(t *testing.T) {
	// Regression for the 2026-05-04 hotfix: vendors used to price
	// items they don't normally stock at the 5x scarcity ceiling
	// (current=0, restock=1 → ratio=0 → PriceCeiling=5.0), which
	// could push the offer above the gold reserve and self-reject.
	// New rule: unstocked items get flat value × BuyRatio.
	item := makeItem(items.ItemSpec{
		ItemId:           100,
		Value:            60, // arena tower shield-tier value
		Type:             items.Offhand,
		VendorCategories: []string{"blacksmithing"},
	})
	shop := baseShop()
	shop.CraftSupport = CraftSupportBlacksmithing
	// Empty stock list → item is unstocked.
	cfg := DefaultPricingConfig()
	offer := EvaluateBuyRules(item, shop, "", false, cfg, nil)
	expected := int(60 * cfg.BuyRatio)
	assert.Equal(t, expected, offer.Price,
		"unstocked items should price at flat value × BuyRatio, not the 5x scarcity ceiling")
}

func TestBuyRules_StockedItem_StillUsesScarcity(t *testing.T) {
	// Companion to TestBuyRules_UnstockedItem_FlatPrice: confirms
	// stocked items still get the dynamic scarcity multiplier.
	item := makeItem(items.ItemSpec{
		ItemId:           100,
		Value:            20,
		Type:             items.Object,
		VendorCategories: []string{"alchemy"},
	})
	shop := baseShop()
	shop.CraftSupport = CraftSupportAlchemy
	// Out-of-stock entry → expect higher-than-flat price (scarcity > 1).
	shop.Stock = []StockEntry{{ItemId: 100, RestockQty: 5, MaxStock: 20, Current: 0}}
	cfg := DefaultPricingConfig()
	offer := EvaluateBuyRules(item, shop, "", false, cfg, nil)
	flat := int(20 * cfg.BuyRatio)
	assert.Greater(t, offer.Price, flat,
		"stocked but empty entries should get scarcity-bonus pricing above flat")
}

func TestBuyRules_AtMaxStockRejected(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId:           100,
		Value:            20,
		Type:             items.Object,
		VendorCategories: []string{"alchemy"},
	})
	shop := baseShop()
	shop.CraftSupport = CraftSupportAlchemy
	shop.Stock = []StockEntry{{ItemId: 100, MaxStock: 10, Current: 10}}
	offer := EvaluateBuyRules(item, shop, "", false, DefaultPricingConfig(), nil)
	assert.Equal(t, 0, offer.Price, "vendor at MaxStock must reject (overstock cap)")
}

func TestBuyRules_InsufficientGoldRejected(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId:           100,
		Value:            10000, // very expensive
		Type:             items.Object,
		VendorCategories: []string{"alchemy"},
	})
	shop := baseShop()
	shop.CraftSupport = CraftSupportAlchemy
	shop.Gold = 50          // not enough
	shop.StartingGold = 1000 // reserve = 500 (50% default from config)
	offer := EvaluateBuyRules(item, shop, "", false, DefaultPricingConfig(), nil)
	assert.Equal(t, 0, offer.Price, "vendor must refuse if buying drops below gold reserve")
}

func TestBuyRules_GearUpgradeRuleRemoved(t *testing.T) {
	// Regression: in the old chain, an alchemist with empty equipment
	// slots would buy a sword as a "gear upgrade." The new rule has no
	// such concept; tag mismatch alone rejects.
	item := makeItem(items.ItemSpec{
		ItemId:           100,
		Value:            500,
		Type:             items.Weapon,
		VendorCategories: []string{"blacksmithing"},
	})
	shop := baseShop()
	shop.CraftSupport = CraftSupportAlchemy
	// wornItems used to be a gear-upgrade input; now ignored.
	offer := EvaluateBuyRules(item, shop, "", false, DefaultPricingConfig(), nil)
	assert.Equal(t, 0, offer.Price, "gear-upgrade rule must be removed; alchemist rejects sword")
}

func TestBuyRules_PotionNoAging_Accepted(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId:           200,
		Value:            40,
		Type:             items.Potion,
		VendorCategories: []string{"alchemy"},
		// No aging thresholds — always fresh
	})
	shop := baseShop()
	shop.CraftSupport = CraftSupportAlchemy
	offer := EvaluateBuyRules(item, shop, "", false, DefaultPricingConfig(), nil)
	assert.Greater(t, offer.Price, 0, "potion with no aging thresholds should be accepted")
}

func TestBuyRules_PotionFresh_Accepted(t *testing.T) {
	// CraftedRound = 0 means no recorded craft time, so aging check is skipped
	// and the potion is treated as acceptable (not declining/spoiled).
	item := makeItem(items.ItemSpec{
		ItemId:           201,
		Value:            40,
		Type:             items.Potion,
		VendorCategories: []string{"alchemy"},
		Aging: items.AgingThresholds{
			FermentRounds: 100,
			PeakRounds:    200,
			DecayRounds:   300,
			SpoilRounds:   400,
		},
	})
	item.CraftedRound = 0
	item.BottleMultiplier = 1.0
	shop := baseShop()
	shop.CraftSupport = CraftSupportAlchemy
	offer := EvaluateBuyRules(item, shop, "", false, DefaultPricingConfig(), nil)
	assert.Greater(t, offer.Price, 0, "fresh potion should be accepted")
}

func TestBuyRules_PotionWithAging_NoCraftedRound_Accepted(t *testing.T) {
	// When CraftedRound == 0, the aging check is skipped entirely (item has
	// no recorded craft time — treat as fresh). The potion is accepted.
	item := makeItem(items.ItemSpec{
		ItemId:           202,
		Value:            40,
		Type:             items.Potion,
		VendorCategories: []string{"alchemy"},
		Aging: items.AgingThresholds{
			FermentRounds: 10,
			PeakRounds:    20,
			DecayRounds:   30,
			SpoilRounds:   100,
		},
	})
	item.CraftedRound = 0 // No recorded craft round — skip aging check
	item.BottleMultiplier = 1.0
	shop := baseShop()
	shop.CraftSupport = CraftSupportAlchemy
	offer := EvaluateBuyRules(item, shop, "", false, DefaultPricingConfig(), nil)
	assert.Greater(t, offer.Price, 0, "potion with no craft round should be accepted")
}

func TestBuyRules_DecliningPotion_Rejected(t *testing.T) {
	// Set up an aging potion that's reached the Declining phase.
	// Aging triggers when CraftedRound is set, the spec has aging
	// thresholds, and the elapsed time × effective speed exceeds
	// the DecayRounds threshold.
	spec := items.ItemSpec{
		ItemId:                200,
		Value:                 50,
		Type:                  items.Potion,
		VendorCategories:      []string{"alchemy"},
		BottleAgingMultiplier: 1.0,
		Aging: items.AgingThresholds{
			FermentRounds: 10,
			PeakRounds:    20,
			DecayRounds:   30, // Declining phase starts here
			SpoilRounds:   60,
		},
	}
	// CraftedRound far in the past so we're well into Declining.
	item := items.Item{
		ItemId:       spec.ItemId,
		Spec:         &spec,
		CraftedRound: 1, // ancient
		BottleMultiplier: 1.0,
	}
	// Set round count high enough to be in Declining phase.
	// elapsed = current - crafted = 100 - 1 = 99
	// decay threshold = 30, spoil threshold = 60
	// Since 60 < 99 < 60, we're in Declining (elapsed >= decay && elapsed < spoil is false, so we check > 60)
	// Actually, 99 > 60, so we hit PhaseSpoiled. Let's adjust:
	// We want 30 < elapsed < 60 for Declining. Set crafted=50, current=80: elapsed=30 (not in Declining yet)
	// Set crafted=1, current=50: elapsed=49, which is > 30 and < 60, so Declining!
	util.SetRoundCountForTest(50)
	shop := baseShop()
	shop.CraftSupport = CraftSupportAlchemy
	offer := EvaluateBuyRules(item, shop, "", false, DefaultPricingConfig(), nil)
	assert.Equal(t, 0, offer.Price, "declining potion must be rejected even with matching tag")
}
