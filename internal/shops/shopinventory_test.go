package shops

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

// yamlMarshal / yamlUnmarshal are thin wrappers used by round-trip tests
// in this file so we reference the same library the production code uses.
var yamlMarshal = yaml.Marshal
var yamlUnmarshal = yaml.Unmarshal

// Test fixture item IDs for RestockTier tests.
const (
	testTier50ItemA = 501
	testTier50ItemB = 502
	testTier30Item  = 503
)

func TestGetStock_Found(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 3},
		},
	}
	entry := si.GetStock(100)
	assert.NotNil(t, entry)
	assert.Equal(t, 3, entry.Current)
}

func TestGetStock_NotFound(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{}}
	assert.Nil(t, si.GetStock(999))
}

func TestAddStock_Existing(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 3},
		},
	}
	si.AddStock(100, 5)
	assert.Equal(t, 8, si.GetStock(100).Current)
}

func TestAddStock_CapsAtMax(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 8},
		},
	}
	si.AddStock(100, 5)
	assert.Equal(t, 10, si.GetStock(100).Current)
}

func TestAddStock_NewItem(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{}}
	si.AddStock(200, 3)
	entry := si.GetStock(200)
	assert.NotNil(t, entry)
	assert.Equal(t, 3, entry.Current)
	assert.Equal(t, 0, entry.RestockQty)
	assert.Equal(t, 20, entry.MaxStock)
}

func TestRemoveStock(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 3},
		},
	}
	removed := si.RemoveStock(100, 2)
	assert.Equal(t, 2, removed)
	assert.Equal(t, 1, si.GetStock(100).Current)
}

func TestRemoveStock_CapsAtZero(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 1},
		},
	}
	removed := si.RemoveStock(100, 5)
	assert.Equal(t, 1, removed)
	assert.Equal(t, 0, si.GetStock(100).Current)
}

func TestRemoveStock_NotFound(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{}}
	removed := si.RemoveStock(999, 1)
	assert.Equal(t, 0, removed)
}

func TestRestock(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 3},
			{ItemId: 200, RestockQty: 0, MaxStock: 8, Current: 2},
			{ItemId: 300, RestockQty: 3, MaxStock: 5, Current: 5},
		},
	}
	restocked := si.Restock()
	assert.True(t, restocked)
	assert.Equal(t, 8, si.GetStock(100).Current)
	assert.Equal(t, 2, si.GetStock(200).Current)
	assert.Equal(t, 5, si.GetStock(300).Current)
}

func TestRestock_PartialFill(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 100, RestockQty: 8, MaxStock: 10, Current: 7},
		},
	}
	si.Restock()
	assert.Equal(t, 10, si.GetStock(100).Current)
}

func TestRestock_NothingToDo(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 10},
		},
	}
	restocked := si.Restock()
	assert.False(t, restocked)
}

func TestGoldReserve(t *testing.T) {
	si := &ShopInventory{StartingGold: 500}
	assert.Equal(t, 250, si.GoldReserve(0.50))
}

func TestCanAfford(t *testing.T) {
	si := &ShopInventory{Gold: 300}
	assert.True(t, si.CanAfford(50, 250))
	assert.False(t, si.CanAfford(100, 250))
}

func TestRestockBuckets_OnlyFillsMatchingBucket(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 40001 /*base*/, Current: 0, MaxStock: 5, RestockQty: 5},
		{ItemId: 40051 /*stillwater*/, Current: 0, MaxStock: 5, RestockQty: 5},
		{ItemId: 40010 /*thornwall*/, Current: 0, MaxStock: 5, RestockQty: 5},
	}}
	refilled := si.RestockBuckets([]string{"stillwater"})
	assert.True(t, refilled, "expected RestockBuckets to refill at least one slot")
	assert.Equal(t, 0, si.Stock[0].Current, "base slot refilled but bucket not in list")
	assert.Equal(t, 5, si.Stock[1].Current, "stillwater slot not refilled")
	assert.Equal(t, 0, si.Stock[2].Current, "thornwall slot refilled but bucket not in list")
}

func TestRestockBuckets_MultipleBucketsUnion(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 40001 /*base*/, Current: 0, MaxStock: 5, RestockQty: 5},
		{ItemId: 40051 /*stillwater*/, Current: 0, MaxStock: 5, RestockQty: 5},
		{ItemId: 40046 /*fernway*/, Current: 0, MaxStock: 5, RestockQty: 5},
	}}
	si.RestockBuckets([]string{"stillwater", "fernway"})
	assert.Equal(t, 0, si.Stock[0].Current, "base slot refilled")
	assert.Equal(t, 5, si.Stock[1].Current, "stillwater not refilled")
	assert.Equal(t, 5, si.Stock[2].Current, "fernway not refilled")
}

func TestRestockBuckets_EmptyListNoOp(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 40001, Current: 0, MaxStock: 5, RestockQty: 5},
	}}
	assert.False(t, si.RestockBuckets(nil), "nil bucket list should be no-op")
	assert.False(t, si.RestockBuckets([]string{}), "empty bucket list should be no-op")
	assert.Equal(t, 0, si.Stock[0].Current, "entry refilled despite empty bucket list")
}

func TestRestockBuckets_SkipsZeroRestockQty(t *testing.T) {
	// Items with RestockQty <= 0 are NPC-crafted, not supply-cart-fed.
	// RestockBuckets must not touch them even if their bucket is in the list.
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 40051 /*stillwater*/, Current: 0, MaxStock: 5, RestockQty: 0}, // crafted
		{ItemId: 40057 /*stillwater*/, Current: 0, MaxStock: 5, RestockQty: 5}, // delivered
	}}
	si.RestockBuckets([]string{"stillwater"})
	assert.Equal(t, 0, si.Stock[0].Current, "zero-RestockQty slot was refilled")
	assert.Equal(t, 5, si.Stock[1].Current, "normal slot not refilled")
}

func TestRestockBuckets_RespectsMaxStockCap(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 40051, Current: 4, MaxStock: 5, RestockQty: 5},
	}}
	si.RestockBuckets([]string{"stillwater"})
	assert.Equal(t, 5, si.Stock[0].Current, "expected capped at MaxStock=5")
}

func TestRestockBaselineTiers_TopsUpTier50And40(t *testing.T) {
	// Register test items with specific rarity tiers.
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 1, RarityTier: 50})
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 2, RarityTier: 40})
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 3, RarityTier: 30})

	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 1, RestockQty: 5, MaxStock: 50, Current: 10},
			{ItemId: 2, RestockQty: 5, MaxStock: 40, Current: 10},
			{ItemId: 3, RestockQty: 5, MaxStock: 30, Current: 10},
		},
	}
	if !si.RestockBaselineTiers() {
		t.Errorf("expected restocked=true")
	}
	if si.Stock[0].Current != 15 {
		t.Errorf("tier-50 entry: Current = %d, want 15", si.Stock[0].Current)
	}
	if si.Stock[1].Current != 15 {
		t.Errorf("tier-40 entry: Current = %d, want 15", si.Stock[1].Current)
	}
	if si.Stock[2].Current != 10 {
		t.Errorf("tier-30 entry should not restock: Current = %d, want 10", si.Stock[2].Current)
	}
}

func TestRestockBaselineTiers_SkipsCrafterEntries(t *testing.T) {
	// Items with RestockQty <= 0 (NPC-crafted) are always skipped.
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 10, RarityTier: 50})

	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 10, RestockQty: 0, MaxStock: 50, Current: 0},
		},
	}
	if si.RestockBaselineTiers() {
		t.Errorf("RestockQty=0 entry should not restock")
	}
	if si.Stock[0].Current != 0 {
		t.Errorf("Current changed unexpectedly: %d", si.Stock[0].Current)
	}
}

func TestRestockBaselineTiers_SkipsAtCap(t *testing.T) {
	// Entries already at MaxStock are skipped.
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 11, RarityTier: 50})

	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 11, RestockQty: 5, MaxStock: 50, Current: 50},
		},
	}
	if si.RestockBaselineTiers() {
		t.Errorf("at-cap entry should not restock")
	}
	if si.Stock[0].Current != 50 {
		t.Errorf("Current = %d, want 50", si.Stock[0].Current)
	}
}

func TestRestockBaselineTiers_CapsAtMaxStock(t *testing.T) {
	// When RestockQty would exceed remaining room, cap at MaxStock.
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 12, RarityTier: 50})

	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: 12, RestockQty: 10, MaxStock: 50, Current: 47},
		},
	}
	si.RestockBaselineTiers()
	if si.Stock[0].Current != 50 {
		t.Errorf("should cap at MaxStock: Current = %d, want 50", si.Stock[0].Current)
	}
}

// TestRestockTier_UnratedItemDefaultsToTier50 covers the regression
// where pre-tier-system content (tutorial gear like Adela's stock)
// has no rarity_tier set on its YAML, defaults to RarityTier=0, and
// would otherwise be invisible to every per-tier ticker. The fallback
// treats unrated items as tier 50 so they restock on the common
// cadence — the safe default that keeps tutorial-gear shops working.
func TestRestockTier_UnratedItemDefaultsToTier50(t *testing.T) {
	const unratedItem = 999081
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: unratedItem, RarityTier: 0})

	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: unratedItem, RestockQty: 5, MaxStock: 10, Current: 0},
		},
	}

	// Tier-50 ticker should pick it up (defaulted from 0).
	if !si.RestockTier(50) {
		t.Errorf("expected unrated item to be restocked by tier-50 ticker")
	}
	if si.Stock[0].Current != 5 {
		t.Errorf("unrated item current after tier-50 restock: got %d, want 5", si.Stock[0].Current)
	}

	// Tier-40 ticker should NOT pick it up (defaulted to 50, not 40).
	si.Stock[0].Current = 0
	if si.RestockTier(40) {
		t.Errorf("tier-40 ticker should not affect unrated (default-50) item")
	}
	if si.Stock[0].Current != 0 {
		t.Errorf("unrated item current after tier-40 ticker: got %d, want 0", si.Stock[0].Current)
	}
}

func TestRestockTier_OnlyTopsUpMatchingTier(t *testing.T) {
	// Register test items with specific rarity tiers.
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: testTier50ItemA, RarityTier: 50})
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: testTier50ItemB, RarityTier: 50})
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: testTier30Item, RarityTier: 30})

	// Arrange: shop has one tier-50 item half-empty, one tier-30
	// item half-empty, one tier-50 item already full.
	// Both tier-50s have RestockQty=5, MaxStock=10.
	// Tier-30 has RestockQty=5, MaxStock=10.
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: testTier50ItemA, RestockQty: 5, MaxStock: 10, Current: 5},
			{ItemId: testTier30Item, RestockQty: 5, MaxStock: 10, Current: 5},
			{ItemId: testTier50ItemB, RestockQty: 5, MaxStock: 10, Current: 10},
		},
	}
	// Act: restock tier 50 only.
	added := si.RestockTier(50)
	// Assert: tier-50 half-empty filled, tier-30 untouched, tier-50
	// already-full untouched. Returns true (something added).
	if !added {
		t.Errorf("expected RestockTier to return true")
	}
	if si.Stock[0].Current != 10 {
		t.Errorf("tier-50 half-empty: got %d, want 10", si.Stock[0].Current)
	}
	if si.Stock[1].Current != 5 {
		t.Errorf("tier-30 should be untouched: got %d", si.Stock[1].Current)
	}
	if si.Stock[2].Current != 10 {
		t.Errorf("tier-50 full should be untouched: got %d", si.Stock[2].Current)
	}
}

func TestRestockTier_IncrementsRestockCount(t *testing.T) {
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: testTier50ItemA, RarityTier: 50})
	si := &ShopInventory{
		Stock: []StockEntry{{ItemId: testTier50ItemA, RestockQty: 5, MaxStock: 10, Current: 5}},
	}
	si.RestockTier(50)
	if si.RestockCount != 5 {
		t.Errorf("RestockCount = %d, want 5", si.RestockCount)
	}
}

func TestRemoveStock_MarksDepletionWhenHittingZero(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 3}},
	}
	si.RemoveStockAtRound(100, 3, 2000)
	got, ok := si.CurrentDepletion[100]
	if !ok {
		t.Fatalf("expected CurrentDepletion[100] to be set")
	}
	if got != 2000 {
		t.Errorf("CurrentDepletion[100] = %d, want 2000", got)
	}
}

func TestRemoveStock_NoMarkIfNotZero(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 5}},
	}
	si.RemoveStockAtRound(100, 2, 2000)
	if _, ok := si.CurrentDepletion[100]; ok {
		t.Errorf("CurrentDepletion[100] should not be set when stock > 0")
	}
}

func TestAddStock_PushesStockEventOnRefill(t *testing.T) {
	si := &ShopInventory{
		Stock:            []StockEntry{{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 0}},
		CurrentDepletion: map[int]uint64{100: 1000},
	}
	si.AddStockAtRound(100, 5, 1500)
	if len(si.StockEvents[100]) != 1 {
		t.Fatalf("expected 1 stock event, got %d", len(si.StockEvents[100]))
	}
	ev := si.StockEvents[100][0]
	if ev.DepletedRound != 1000 || ev.RefilledRound != 1500 {
		t.Errorf("event = %+v, want {1000, 1500}", ev)
	}
	if _, still := si.CurrentDepletion[100]; still {
		t.Errorf("CurrentDepletion[100] should be cleared")
	}
}

func TestAddStock_NoEventIfNotDepleted(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 3}},
	}
	si.AddStockAtRound(100, 5, 1500)
	if len(si.StockEvents[100]) != 0 {
		t.Errorf("no event expected when item wasn't fully depleted, got %d", len(si.StockEvents[100]))
	}
}

// TestShopInventory_ConsumedByCrafterCount_DefaultsZero verifies the new
// counter field initialises to zero on a fresh ShopInventory value.
func TestShopInventory_ConsumedByCrafterCount_DefaultsZero(t *testing.T) {
	si := &ShopInventory{}
	if si.ConsumedByCrafterCount != 0 {
		t.Errorf("default ConsumedByCrafterCount = %d, want 0", si.ConsumedByCrafterCount)
	}
}

// TestShopInventory_ConsumedByCrafterCount_IncrementAndRoundTrip verifies
// that the counter can be set and is preserved through a YAML round-trip
// (the yaml tag must match and omitempty must not suppress non-zero values).
func TestShopInventory_ConsumedByCrafterCount_IncrementAndRoundTrip(t *testing.T) {
	si := &ShopInventory{ConsumedByCrafterCount: 7}
	data, err := yamlMarshal(si)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	var got ShopInventory
	if err := yamlUnmarshal(data, &got); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	if got.ConsumedByCrafterCount != 7 {
		t.Errorf("ConsumedByCrafterCount after round-trip = %d, want 7", got.ConsumedByCrafterCount)
	}
}

func TestIsValidVendorCategory(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"alchemy", true},
		{"blacksmithing", true},
		{"jewelcrafting", true},
		{"tailoring", true},
		{"cooking", true},
		{"enchanting", true},
		{"general", false}, // general is a vendor type, not an item tag
		{"", false},
		{"unknown", false},
	}
	for _, tt := range tests {
		if got := IsValidVendorCategory(tt.in); got != tt.want {
			t.Errorf("IsValidVendorCategory(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestAffixedStock_AddListRemove(t *testing.T) {
	si := &ShopInventory{}
	a := items.Item{ItemId: 10, Affixed: true, Spec: &items.ItemSpec{Value: 400, Name: "Keen Torc"}}
	b := items.Item{ItemId: 11, Affixed: true, Spec: &items.ItemSpec{Value: 300, Name: "Warding Ring"}}

	si.AddAffixedStock(a, 200, 100)
	si.AddAffixedStock(b, 150, 100)
	if len(si.AffixedStock) != 2 {
		t.Fatalf("want 2 affixed entries, got %d", len(si.AffixedStock))
	}

	got, ok := si.RemoveAffixedStock(0)
	if !ok || got.ItemId != 10 {
		t.Fatalf("RemoveAffixedStock(0) = %+v, %v", got, ok)
	}
	if len(si.AffixedStock) != 1 || si.AffixedStock[0].Item.ItemId != 11 {
		t.Fatalf("after remove, want [11], got %+v", si.AffixedStock)
	}
}

func TestAffixedStock_CapEvictsOldest(t *testing.T) {
	si := &ShopInventory{}
	for i := 0; i < 5; i++ {
		si.AddAffixedStock(items.Item{ItemId: 100 + i, Affixed: true,
			Spec: &items.ItemSpec{Value: 100}}, 50, 3)
	}
	if len(si.AffixedStock) != 3 {
		t.Fatalf("cap 3: want 3 entries, got %d", len(si.AffixedStock))
	}
	for i, want := range []int{102, 103, 104} {
		if si.AffixedStock[i].Item.ItemId != want {
			t.Errorf("entry %d = %d; want %d", i, si.AffixedStock[i].Item.ItemId, want)
		}
	}
}
