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

func TestComputeShopDelta_ExcludesEmptyBucket(t *testing.T) {
	// Stock entries with Bucket=="" must NOT show up in BucketDeltas
	// (untracked items shouldn't count against the supply chain).
	now := health.ShopSnapshot{
		Zone: "z", MobId: 1, RoomId: 1,
		Stock: []health.StockSnapshot{
			{ItemId: 40001, Bucket: "base", Current: 5, Max: 10},
			{ItemId: 99999, Bucket: "", Current: 7, Max: 10}, // untracked
		},
	}
	old := health.ShopSnapshot{
		Zone: "z", MobId: 1, RoomId: 1,
		Stock: []health.StockSnapshot{
			{ItemId: 40001, Bucket: "base", Current: 8, Max: 10},
			{ItemId: 99999, Bucket: "", Current: 4, Max: 10},
		},
	}
	d := health.ComputeShopDelta(now, &old)
	if _, ok := d.BucketDeltas[""]; ok {
		t.Errorf("empty bucket leaked into BucketDeltas: %v", d.BucketDeltas)
	}
	if len(d.BucketDeltas) != 1 || d.BucketDeltas["base"] != -3 {
		t.Errorf("BucketDeltas: got %v, want {base: -3} only", d.BucketDeltas)
	}
}

func TestComputeShopDelta_OldOnlyBucketEmitsNegative(t *testing.T) {
	// A bucket present in old but absent from now should emit a
	// negative delta equal to the old total — covers the
	// "depleted to zero" case.
	now := health.ShopSnapshot{Zone: "z", MobId: 1, RoomId: 1}
	old := health.ShopSnapshot{
		Zone: "z", MobId: 1, RoomId: 1,
		Stock: []health.StockSnapshot{{ItemId: 40051, Bucket: "stillwater", Current: 6, Max: 10}},
	}
	d := health.ComputeShopDelta(now, &old)
	if d.BucketDeltas["stillwater"] != -6 {
		t.Errorf("stillwater bucket: got %d, want -6 (old-only branch)", d.BucketDeltas["stillwater"])
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
