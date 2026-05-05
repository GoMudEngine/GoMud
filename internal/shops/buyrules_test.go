package shops

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
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
