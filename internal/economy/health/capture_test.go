package health_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/economy/health"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

func TestCaptureSnapshot_Shops(t *testing.T) {
	shops.ClearCache()
	t.Cleanup(shops.ClearCache)

	tmpl := shops.ShopInventory{
		Gold:         500,
		StartingGold: 500,
		CraftSupport: shops.CraftSupportGeneral,
		Stock: []shops.StockEntry{
			{ItemId: 40001, RestockQty: 5, MaxStock: 20, Current: 8}, // base
			{ItemId: 40051, RestockQty: 3, MaxStock: 10, Current: 4}, // stillwater
		},
	}
	shops.RegisterShop("stillwater", 341, 4105, tmpl)

	snap := health.CaptureSnapshot()

	if len(snap.Shops) != 1 {
		t.Fatalf("Shops: got %d, want 1", len(snap.Shops))
	}
	got := snap.Shops[0]
	if got.MobId != 341 || got.RoomId != 4105 {
		t.Errorf("location: got %d/%d, want 341/4105", got.MobId, got.RoomId)
	}
	if got.CraftSupport != "general" {
		t.Errorf("craft_support: got %q, want general", got.CraftSupport)
	}
	if len(got.Stock) != 2 {
		t.Fatalf("stock entries: got %d, want 2", len(got.Stock))
	}
	if got.Stock[0].Bucket != "base" {
		t.Errorf("first stock bucket: got %q, want base", got.Stock[0].Bucket)
	}
}
