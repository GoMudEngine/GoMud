package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/itemvalue"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

func TestStockRestockNorm(t *testing.T) {
	cases := []struct {
		name string
		e    shops.StockEntry
		want int
	}{
		{"explicit restock qty", shops.StockEntry{RestockQty: 5, MaxStock: 20}, 5},
		{"crafted-only uses maxstock/2", shops.StockEntry{RestockQty: 0, MaxStock: 20}, 10},
		{"maxstock/2 floors at 1", shops.StockEntry{RestockQty: 0, MaxStock: 1}, 1},
		{"no info floors at 1", shops.StockEntry{}, 1},
	}
	for _, c := range cases {
		if got := stockRestockNorm(c.e); got != c.want {
			t.Errorf("%s: stockRestockNorm = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestScanZoneUpgrades_NilMob(t *testing.T) {
	if _, ok := scanZoneUpgrades(nil, itemvalue.WeightProfile{}, 100, true, 1.0); ok {
		t.Errorf("expected ok=false for nil mob")
	}
}

func TestScanZoneUpgrades_NoShopsLoaded(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	if _, ok := scanZoneUpgrades(mob, itemvalue.WeightProfile{}, 1000, true, 1.0); ok {
		t.Errorf("expected ok=false when no shops are loaded")
	}
}
