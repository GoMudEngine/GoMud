package shops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverstockDecay_DrainsNonMaterialAboveBaseline(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 100, RestockQty: 2, MaxStock: 10, Current: 6, LastGrewRound: 0},
	}}
	TickOverstockDecayWith(si, 100000, func(int) bool { return false }, 21600, 1)
	assert.Equal(t, 5, si.Stock[0].Current, "one unit should decay")
}

func TestOverstockDecay_SkipsComponents(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 200, RestockQty: 0, MaxStock: 10, Current: 8, LastGrewRound: 0},
	}}
	TickOverstockDecayWith(si, 100000, func(int) bool { return true }, 21600, 1)
	assert.Equal(t, 8, si.Stock[0].Current, "components never decay")
}

func TestOverstockDecay_RespectsBaselineFloor(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 5, LastGrewRound: 0},
	}}
	TickOverstockDecayWith(si, 100000, func(int) bool { return false }, 21600, 1)
	assert.Equal(t, 5, si.Stock[0].Current, "at baseline → no decay")
}

func TestOverstockDecay_GracePeriodNotElapsed(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 100, RestockQty: 2, MaxStock: 10, Current: 6, LastGrewRound: 99000},
	}}
	TickOverstockDecayWith(si, 100000, func(int) bool { return false }, 21600, 1)
	assert.Equal(t, 6, si.Stock[0].Current, "1000 < 21600 grace → no decay")
}
