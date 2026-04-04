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
		ItemId:     1,
		Value:      50,
		QuestToken: "some-quest-token",
		Type:       items.Object,
	})
	offer := EvaluateBuyRules(item, baseShop(), "", false, DefaultPricingConfig(), nil)
	assert.Equal(t, 0, offer.Price, "quest items must always be rejected")
	assert.Equal(t, "", offer.Reason)
}

func TestBuyRules_CraftMaterialWithCrafterSkill(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId:       100,
		Value:        20,
		ComponentTag: "iron",
		Type:         items.Object,
	})
	// NPC must stock this item for the craft material rule to fire.
	shop := baseShop()
	shop.Stock = []StockEntry{
		{ItemId: 100, RestockQty: 3, MaxStock: 10, Current: 2},
	}
	offer := EvaluateBuyRules(item, shop, "blacksmithing", false, DefaultPricingConfig(), nil)
	assert.Greater(t, offer.Price, 0, "crafter NPC should offer a price for stocked materials")
	assert.Equal(t, "craft_material", offer.Reason)
}

func TestBuyRules_CraftMaterialNotUsed(t *testing.T) {
	// NPC has a crafter skill but no recipes or stock using this tag.
	item := makeItem(items.ItemSpec{
		ItemId:       100,
		Value:        20,
		ComponentTag: "herbs",
		Type:         items.Object,
	})
	shop := baseShop()
	// Stock list has iron, not herbs.
	shop.Stock = []StockEntry{
		{ItemId: 200, RestockQty: 3, MaxStock: 10, Current: 2},
	}
	offer := EvaluateBuyRules(item, shop, "blacksmithing", false, DefaultPricingConfig(), nil)
	assert.Equal(t, 0, offer.Price, "crafter should not buy materials from other professions")
}

func TestBuyRules_CraftMaterialWithoutCrafterSkill(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId:       100,
		Value:        20,
		ComponentTag: "iron",
		Type:         items.Object,
	})
	// No crafter skill and no buysGeneral — should reject
	offer := EvaluateBuyRules(item, baseShop(), "", false, DefaultPricingConfig(), nil)
	assert.Equal(t, 0, offer.Price, "non-crafter should not buy craft materials")
}

func TestBuyRules_CraftMaterialCrafterSkillFallsToGeneral(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId:       100,
		Value:        20,
		ComponentTag: "iron",
		Type:         items.Object,
	})
	// No crafter skill but buysGeneral = true → falls through to general rule
	offer := EvaluateBuyRules(item, baseShop(), "", true, DefaultPricingConfig(), nil)
	assert.Greater(t, offer.Price, 0)
	assert.Equal(t, "general", offer.Reason)
}

func TestBuyRules_CraftMaterialAtMaxStock_Rejected(t *testing.T) {
	shop := baseShop()
	shop.Stock = []StockEntry{
		{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 10}, // already full
	}
	item := makeItem(items.ItemSpec{
		ItemId:       100,
		Value:        20,
		ComponentTag: "iron",
		Type:         items.Object,
	})
	// At max_stock — crafter should decline this rule and fall through to empty
	offer := EvaluateBuyRules(item, shop, "blacksmithing", false, DefaultPricingConfig(), nil)
	assert.Equal(t, 0, offer.Price, "should reject material already at max stock")
}

func TestBuyRules_CraftMaterialAtMaxStock_FallsToGeneral(t *testing.T) {
	shop := baseShop()
	shop.Stock = []StockEntry{
		{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 10},
	}
	item := makeItem(items.ItemSpec{
		ItemId:       100,
		Value:        20,
		ComponentTag: "iron",
		Type:         items.Object,
	})
	// buysGeneral = true — max stock material falls through to general
	offer := EvaluateBuyRules(item, shop, "blacksmithing", true, DefaultPricingConfig(), nil)
	assert.Greater(t, offer.Price, 0)
	assert.Equal(t, "general", offer.Reason)
}

func TestBuyRules_PotionNoAging_Accepted(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId: 200,
		Value:  40,
		Type:   items.Potion,
		// No aging thresholds — always fresh
	})
	offer := EvaluateBuyRules(item, baseShop(), "", false, DefaultPricingConfig(), nil)
	assert.Greater(t, offer.Price, 0)
	assert.Equal(t, "potion", offer.Reason)
}

func TestBuyRules_PotionFresh_Accepted(t *testing.T) {
	// CraftedRound = 0 means no recorded craft time, so aging check is skipped
	// and the potion is treated as acceptable (not declining/spoiled).
	item := makeItem(items.ItemSpec{
		ItemId: 201,
		Value:  40,
		Type:   items.Potion,
		Aging: items.AgingThresholds{
			FermentRounds: 100,
			PeakRounds:    200,
			DecayRounds:   300,
			SpoilRounds:   400,
		},
	})
	item.CraftedRound = 0
	item.BottleMultiplier = 1.0
	offer := EvaluateBuyRules(item, baseShop(), "", false, DefaultPricingConfig(), nil)
	assert.Greater(t, offer.Price, 0)
	assert.Equal(t, "potion", offer.Reason)
}

func TestBuyRules_PotionWithAging_NoCraftedRound_Accepted(t *testing.T) {
	// When CraftedRound == 0, the aging check is skipped entirely (item has
	// no recorded craft time — treat as fresh). The potion is accepted.
	item := makeItem(items.ItemSpec{
		ItemId: 202,
		Value:  40,
		Type:   items.Potion,
		Aging: items.AgingThresholds{
			FermentRounds: 10,
			PeakRounds:    20,
			DecayRounds:   30,
			SpoilRounds:   100,
		},
	})
	item.CraftedRound = 0 // No recorded craft round — skip aging check
	item.BottleMultiplier = 1.0
	offer := EvaluateBuyRules(item, baseShop(), "", false, DefaultPricingConfig(), nil)
	assert.Greater(t, offer.Price, 0, "potion with no craft round should be accepted")
	assert.Equal(t, "potion", offer.Reason)
}

// Note: Testing declining/spoiled phases in isolation requires controlling
// util.GetRoundCount(). The real aging rejection path is exercised during
// integration/server tests where the round counter advances. The unit tests
// above cover the structural routing; phase computation is separately verified
// in internal/items/aging_test.go.

func TestBuyRules_GeneralGoods_GeneralMerchant(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId: 300,
		Value:  100,
		Type:   items.Object,
	})
	offer := EvaluateBuyRules(item, baseShop(), "", true, DefaultPricingConfig(), nil)
	assert.Greater(t, offer.Price, 0)
	assert.Equal(t, "general", offer.Reason)
	// Verify 25% pricing
	assert.Equal(t, 25, offer.Price)
}

func TestBuyRules_GeneralGoods_SpecialistDeclines(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId: 300,
		Value:  100,
		Type:   items.Object,
	})
	// Specialist crafter that doesn't buy general goods
	offer := EvaluateBuyRules(item, baseShop(), "blacksmithing", false, DefaultPricingConfig(), nil)
	assert.Equal(t, 0, offer.Price, "specialist without buysGeneral should reject generic items")
}

func TestBuyRules_GeneralGoods_MinimumOne(t *testing.T) {
	item := makeItem(items.ItemSpec{
		ItemId: 301,
		Value:  0, // Zero value item
		Type:   items.Object,
	})
	offer := EvaluateBuyRules(item, baseShop(), "", true, DefaultPricingConfig(), nil)
	assert.GreaterOrEqual(t, offer.Price, 1, "offer should never be zero for accepted items")
}

func TestBuyRules_GearUpgrade_EmptySlot(t *testing.T) {
	// NPC has no weapon — any weapon with power > 0 is an upgrade.
	item := makeItem(items.ItemSpec{
		ItemId:           500,
		Value:            80,
		Type:             items.Weapon,
		DamageMultiplier: 0.5,
	})
	worn := []items.Item{} // NPC wears nothing
	offer := EvaluateBuyRules(item, baseShop(), "", false, DefaultPricingConfig(), worn)
	assert.Greater(t, offer.Price, 0, "NPC with empty weapon slot should want a weapon")
	assert.Equal(t, "gear_upgrade", offer.Reason)
}

func TestBuyRules_GearUpgrade_BetterWeapon(t *testing.T) {
	// NPC has a weak weapon — offered a better one.
	item := makeItem(items.ItemSpec{
		ItemId:           501,
		Value:            120,
		Type:             items.Weapon,
		DamageMultiplier: 0.8,
	})
	currentWeapon := makeItem(items.ItemSpec{
		ItemId:           500,
		Type:             items.Weapon,
		DamageMultiplier: 0.4,
	})
	worn := []items.Item{currentWeapon}
	offer := EvaluateBuyRules(item, baseShop(), "", false, DefaultPricingConfig(), worn)
	assert.Greater(t, offer.Price, 0, "NPC should buy a better weapon")
	assert.Equal(t, "gear_upgrade", offer.Reason)
}

func TestBuyRules_GearUpgrade_WorseWeapon(t *testing.T) {
	// NPC already has a better weapon — should decline.
	item := makeItem(items.ItemSpec{
		ItemId:           500,
		Value:            40,
		Type:             items.Weapon,
		DamageMultiplier: 0.3,
	})
	currentWeapon := makeItem(items.ItemSpec{
		ItemId:           501,
		Type:             items.Weapon,
		DamageMultiplier: 0.8,
	})
	worn := []items.Item{currentWeapon}
	offer := EvaluateBuyRules(item, baseShop(), "", false, DefaultPricingConfig(), worn)
	assert.Equal(t, 0, offer.Price, "NPC should not buy a downgrade")
}

func TestBuyRules_GearUpgrade_NonEquipmentIgnored(t *testing.T) {
	// A potion is not equipment — gear rule should not fire.
	item := makeItem(items.ItemSpec{
		ItemId: 600,
		Value:  50,
		Type:   items.Potion,
	})
	worn := []items.Item{}
	offer := EvaluateBuyRules(item, baseShop(), "", false, DefaultPricingConfig(), worn)
	// Should fall through to potion rule, not gear_upgrade
	assert.Equal(t, "potion", offer.Reason)
}

func TestBuyRules_GearUpgrade_NilWornSkipsRule(t *testing.T) {
	// When wornItems is nil, gear-upgrade rule is skipped entirely.
	item := makeItem(items.ItemSpec{
		ItemId:           500,
		Value:            80,
		Type:             items.Weapon,
		DamageMultiplier: 0.5,
	})
	// nil wornItems + no crafter + no buysGeneral = rejection
	offer := EvaluateBuyRules(item, baseShop(), "", false, DefaultPricingConfig(), nil)
	assert.Equal(t, 0, offer.Price, "nil wornItems should skip gear upgrade rule")
}
