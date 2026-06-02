package shops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectStockTransfers_NeediestGapFirst(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 10, MaxStock: 10, Current: 9}, // gap 1
		{ItemId: 20, MaxStock: 10, Current: 2}, // gap 8 (neediest)
	}}
	pool := map[int]int{10: 5, 20: 5}
	got := SelectStockTransfers(si, pool)
	assert.Equal(t, 5, got[20]) // neediest filled first, pool-capped at 5
	assert.Equal(t, 1, got[10])
}

func TestSelectStockTransfers_OnlyStockedAndCapped(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{{ItemId: 10, MaxStock: 10, Current: 10}}}
	assert.Empty(t, SelectStockTransfers(si, map[int]int{10: 5})) // no gap
	si2 := &ShopInventory{Stock: []StockEntry{{ItemId: 10, MaxStock: 10, Current: 5}}}
	assert.Empty(t, SelectStockTransfers(si2, map[int]int{99: 5})) // not stocked
}
