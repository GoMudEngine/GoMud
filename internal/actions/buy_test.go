package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

// TestBuy_AffixedStockItem: a shop's per-instance affixed stock is listed and,
// on purchase, the buyer receives the EXACT stored item (per-instance value,
// affixed flag) and the entry is removed. Exercises tryPurchaseFromInventory
// directly (bypassing room/merchant discovery).
func TestBuy_AffixedStockItem(t *testing.T) {
	defer seedSellItemSpecs()() // registers sellTestItemId ("iron sword")

	m := &mobs.Mob{}
	m.Character.Name = "Buyer"
	m.Character.Gold = 1000
	m.Character.Buffs = buffs.New()
	m.Character.Stats.Strength.ValueAdj = 100 // carry capacity
	buyer := &MobActor{Mob: m}

	shopInv := &shops.ShopInventory{Gold: 1000}
	shopInv.AddAffixedStock(items.Item{ItemId: sellTestItemId, Affixed: true,
		Spec: &items.ItemSpec{Value: 400, Name: "iron sword", NameSimple: "sword", Type: items.Weapon}}, 400, 8)

	shopMob := &mobs.Mob{Character: characters.Character{Name: "Merchant"}}

	res := tryPurchaseFromInventory(buyer, "iron sword", shopMob, shopInv)
	if !res.Success {
		t.Fatalf("affixed purchase failed: %+v", res)
	}
	if len(shopInv.AffixedStock) != 0 {
		t.Errorf("affixed entry should be removed after purchase; have %d", len(shopInv.AffixedStock))
	}
	got, has := m.Character.FindInBackpack("iron sword")
	if !has {
		t.Fatal("buyer should hold the bought item")
	}
	if got.GetSpec().Value != 400 || !got.Affixed {
		t.Errorf("bought item not the affixed instance: value=%d affixed=%v", got.GetSpec().Value, got.Affixed)
	}
	if m.Character.Gold != 600 { // 1000 - relist 400
		t.Errorf("buyer gold = %d; want 600", m.Character.Gold)
	}
}

func TestBuy_EmptyRequest(t *testing.T) {
	result := Buy(nil, BuyOptions{Request: ""})
	if result.Success {
		t.Errorf("expected Success=false on empty request")
	}
	if result.Reason != BuyReasonNoRequest {
		t.Errorf("expected Reason=%q, got %q", BuyReasonNoRequest, result.Reason)
	}
}

func TestBuildLegacyCatalog_SkipsMercAndPet(t *testing.T) {
	saleItems := characters.Shop{
		// Items and buffs should appear; mercs/pets should be skipped.
		{ItemId: 20000, Price: 50, Quantity: 1, QuantityMax: 1},
		{BuffId: 1, Price: 100, Quantity: 1, QuantityMax: 1},
		{MobId: 100, Price: 250, Quantity: 1, QuantityMax: 1},
		{PetType: "kitten", Price: 10000, Quantity: 1, QuantityMax: 1},
	}
	cat := buildLegacyCatalog(saleItems)

	// Even if itemId 20000 doesn't resolve to a real item (the test data
	// dir might not be initialized), the merc/pet entries must always
	// be absent.
	if _, ok := cat.nameToShopItem["100"]; ok {
		t.Errorf("merc MobId should not appear in nameToShopItem")
	}
	if _, ok := cat.nameToShopItem["kitten"]; ok {
		t.Errorf("pet PetType should not appear in nameToShopItem")
	}
}

func newEmptyTestRoom(t *testing.T) *rooms.Room {
	t.Helper()
	r := &rooms.Room{RoomId: 99999}
	return r
}

func TestBuy_NoMerchant(t *testing.T) {
	room := newEmptyTestRoom(t)
	mobActor := &MobActor{Room: room} // no User, no Mob — only Room matters
	result := Buy(mobActor, BuyOptions{Request: "iron ingot"})
	if result.Success {
		t.Errorf("expected Success=false with no merchant in room")
	}
	if result.Reason != BuyReasonNoMerchant {
		t.Errorf("Reason = %q, want %q", result.Reason, BuyReasonNoMerchant)
	}
}

func TestValidatePurchase_OverburdenedBlocksPreSideEffect(t *testing.T) {
	// Fabricate a buyer whose Character has carriedWeight >= capacity.
	// We do this by:
	// 1. Setting very low strength (tiny capacity)
	// 2. Pre-stuffing inventory with items to reach/exceed capacity
	// 3. Attempting purchase of another item
	m := &mobs.Mob{}
	m.Character.Stats.Strength.ValueAdj = 1 // tiny capacity (~0.65 lbs at default multiplier)
	startingGold := 999
	m.Character.Gold = startingGold

	// Pre-fill inventory with a heavy item to exceed capacity.
	// We'll add a real item if it exists; otherwise skip the test.
	// Item 20000 is a common test fixture in DOGMud.
	heavy := items.New(20000)
	if heavy.ItemId == 0 {
		t.Skip("test data fixture missing itemId 20000")
	}
	// Add enough copies to exceed capacity.
	m.Character.Items = append(m.Character.Items, heavy)
	m.Character.Items = append(m.Character.Items, heavy)
	m.Character.Items = append(m.Character.Items, heavy)

	// Now attempt purchase of another item that would overflow.
	saleItem := characters.ShopItem{ItemId: 20000, Price: 1, Quantity: 1, QuantityMax: 1}
	itemPrices := map[int]int{20000: 1}

	buyer := &MobActor{Mob: m}

	_, reason, ok := validatePurchase(buyer, nil, nil, saleItem, itemPrices, nil)

	if ok {
		t.Fatalf("expected validatePurchase to reject overburdened buyer")
	}
	if reason != BuyReasonOverburdened {
		t.Errorf("reason = %q, want %q", reason, BuyReasonOverburdened)
	}
	if m.Character.Gold != startingGold {
		t.Errorf("buyer gold should not be deducted on overburdened rejection; got %d want %d",
			m.Character.Gold, startingGold)
	}
	if saleItem.Quantity != 1 {
		t.Errorf("shop stock should not be destocked on overburdened rejection; got %d", saleItem.Quantity)
	}
}

func TestBuy_QuantityParse(t *testing.T) {
	// We can't easily exercise the full purchase path without a shop
	// fixture; the parsing behavior is observable via Requested even
	// when the purchase fails for unrelated reasons.
	room := newEmptyTestRoom(t)
	mobActor := &MobActor{Room: room}

	result := Buy(mobActor, BuyOptions{Request: "5 iron ingot"})
	if result.Requested != 5 {
		t.Errorf("Requested = %d, want 5", result.Requested)
	}
	if result.Success {
		t.Errorf("expected Success=false with no merchant in room")
	}
}
