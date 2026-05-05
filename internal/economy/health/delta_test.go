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

func TestStockScoreDelta_Computation(t *testing.T) {
	now := health.ShopSnapshot{
		Stock: []health.StockSnapshot{
			{Current: 10, Max: 20},
			{Current: 5, Max: 10},
		},
		StockScore: 0.5, // 15/30 = 0.5 (50%)
	}
	old := health.ShopSnapshot{StockScore: 0.3}
	d := health.ComputeShopDelta(now, &old)
	if d.StockScoreDelta != 20 {
		t.Errorf("StockScoreDelta = %d, want 20 (50%% - 30%% = 20pp)", d.StockScoreDelta)
	}
}

func TestComputeForagerDelta_BasicCounts(t *testing.T) {
	now := health.ForagerSnapshot{
		DeliveriesByTier: map[int]int{50: 100, 40: 50, 30: 10},
		StuckRounds:      200,
	}
	old := health.ForagerSnapshot{
		DeliveriesByTier: map[int]int{50: 80, 40: 30},
		StuckRounds:      50,
	}
	d := health.ComputeForagerDelta(now, &old)
	if d.DeliveriesByTierDelta[50] != 20 {
		t.Errorf("tier 50: got %d, want 20", d.DeliveriesByTierDelta[50])
	}
	if d.DeliveriesByTierDelta[40] != 20 {
		t.Errorf("tier 40: got %d, want 20", d.DeliveriesByTierDelta[40])
	}
	if d.DeliveriesByTierDelta[30] != 10 {
		t.Errorf("tier 30: got %d, want 10 (new tier)", d.DeliveriesByTierDelta[30])
	}
	if d.StuckRoundsDelta != 150 {
		t.Errorf("StuckRoundsDelta: got %d, want 150", d.StuckRoundsDelta)
	}
}

func TestComputeCaravanDelta_BasicCounts(t *testing.T) {
	now := health.CaravanSnapshot{
		DeliveriesByTier: map[int]int{50: 30, 40: 15},
	}
	old := health.CaravanSnapshot{
		DeliveriesByTier: map[int]int{50: 20},
	}
	d := health.ComputeCaravanDelta(now, &old)
	if d.DeliveriesByTierDelta[50] != 10 {
		t.Errorf("tier 50: got %d, want 10", d.DeliveriesByTierDelta[50])
	}
	if d.DeliveriesByTierDelta[40] != 15 {
		t.Errorf("tier 40: got %d, want 15", d.DeliveriesByTierDelta[40])
	}
}

func TestComputeForagerDelta_NilOldReturnsEmptyDelta(t *testing.T) {
	now := health.ForagerSnapshot{
		DeliveriesByTier: map[int]int{50: 5},
		StuckRounds:      100,
	}
	d := health.ComputeForagerDelta(now, nil)
	if len(d.DeliveriesByTierDelta) != 0 {
		t.Errorf("expected empty delta map for nil old, got %v", d.DeliveriesByTierDelta)
	}
	if d.StuckRoundsDelta != 0 {
		t.Errorf("expected 0 StuckRoundsDelta for nil old, got %d", d.StuckRoundsDelta)
	}
}

func TestFindForagerInSnapshot(t *testing.T) {
	snap := &health.Snapshot{
		Foragers: []health.ForagerSnapshot{
			{MobId: 101, Name: "Alice"},
			{MobId: 102, Name: "Bob"},
		},
	}
	f := health.FindForagerInSnapshot(snap, 102)
	if f == nil || f.Name != "Bob" {
		t.Errorf("FindForagerInSnapshot(102) failed")
	}
	f = health.FindForagerInSnapshot(snap, 999)
	if f != nil {
		t.Errorf("FindForagerInSnapshot(999) should return nil")
	}
}

func TestFindCaravanInSnapshot(t *testing.T) {
	snap := &health.Snapshot{
		Caravans: []health.CaravanSnapshot{
			{InstId: 1001, Name: "Caravan A"},
			{InstId: 1002, Name: "Caravan B"},
		},
	}
	c := health.FindCaravanInSnapshot(snap, 1002)
	if c == nil || c.Name != "Caravan B" {
		t.Errorf("FindCaravanInSnapshot(1002) failed")
	}
	c = health.FindCaravanInSnapshot(snap, 9999)
	if c != nil {
		t.Errorf("FindCaravanInSnapshot(9999) should return nil")
	}
}
