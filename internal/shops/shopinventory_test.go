package shops

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
