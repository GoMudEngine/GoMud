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
