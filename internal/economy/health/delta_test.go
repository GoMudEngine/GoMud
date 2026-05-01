package health_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/economy/health"
)

func TestPickClosest_PrefersWithinTolerance(t *testing.T) {
	metas := []health.SnapshotMeta{
		{UnixTs: 100}, {UnixTs: 200}, {UnixTs: 300}, {UnixTs: 1000},
	}
	got := health.PickClosestSnapshot(metas, 250, 60)
	if got == nil || got.UnixTs != 200 {
		t.Errorf("got %v, want 200 (within tolerance 60)", got)
	}
}

func TestPickClosest_NoneInTolerance(t *testing.T) {
	metas := []health.SnapshotMeta{{UnixTs: 100}, {UnixTs: 1000}}
	got := health.PickClosestSnapshot(metas, 500, 60)
	if got != nil {
		t.Errorf("got %v, want nil (no snapshot within 60s of 500)", got)
	}
}

func TestComputeShopDelta_GoldAndStock(t *testing.T) {
	now := health.ShopSnapshot{
		Zone: "z", MobId: 1, RoomId: 1, Gold: 600,
		Stock: []health.StockSnapshot{{ItemId: 40001, Bucket: "base", Current: 5, Max: 10}},
	}
	old := health.ShopSnapshot{
		Zone: "z", MobId: 1, RoomId: 1, Gold: 500,
		Stock: []health.StockSnapshot{{ItemId: 40001, Bucket: "base", Current: 8, Max: 10}},
	}
	d := health.ComputeShopDelta(now, &old)
	if d.GoldDelta != 100 {
		t.Errorf("GoldDelta: got %d, want 100", d.GoldDelta)
	}
	if d.BucketDeltas["base"] != -3 {
		t.Errorf("base bucket: got %d, want -3", d.BucketDeltas["base"])
	}
}
