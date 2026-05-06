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

func TestComputeShopDelta_SalesAndBuysAndRestocks(t *testing.T) {
	now := health.ShopSnapshot{
		Zone: "z", MobId: 1, RoomId: 1,
		SalesCount:   50,
		BuysCount:    20,
		RestockCount: 15,
		Round:        2000,
	}
	old := health.ShopSnapshot{
		Zone: "z", MobId: 1, RoomId: 1,
		SalesCount:   30,
		BuysCount:    10,
		RestockCount: 5,
		Round:        1000,
	}
	d := health.ComputeShopDelta(now, &old)
	if d.SalesDelta != 20 {
		t.Errorf("SalesDelta: got %d, want 20", d.SalesDelta)
	}
	if d.BuysDelta != 10 {
		t.Errorf("BuysDelta: got %d, want 10", d.BuysDelta)
	}
	if d.RestocksDelta != 10 {
		t.Errorf("RestocksDelta: got %d, want 10", d.RestocksDelta)
	}
}

func TestComputeShopDelta_MedianTtR(t *testing.T) {
	// Three completed events with TtR values of 100, 200, 300 rounds.
	// All have RefilledRound > old.Round (1000), so all qualify.
	// Median of [100, 200, 300] = 200 (middle of sorted).
	now := health.ShopSnapshot{
		Zone: "z", MobId: 1, RoomId: 1,
		Round: 2000,
		StockEvents: map[int][]health.StockEvent{
			101: {
				{DepletedRound: 1100, RefilledRound: 1200}, // TtR=100
				{DepletedRound: 1300, RefilledRound: 1500}, // TtR=200
				{DepletedRound: 1600, RefilledRound: 1900}, // TtR=300
			},
		},
	}
	old := health.ShopSnapshot{
		Zone: "z", MobId: 1, RoomId: 1,
		Round: 1000,
	}
	d := health.ComputeShopDelta(now, &old)
	if d.MedianTtR != 200 {
		t.Errorf("MedianTtR: got %d, want 200", d.MedianTtR)
	}
}

func TestComputeShopDelta_MedianTtR_FiltersOldEvents(t *testing.T) {
	// One event completed before old.Round, one after. Only the newer
	// one should contribute. Median of [300] = 300.
	now := health.ShopSnapshot{
		Zone: "z", MobId: 1, RoomId: 1,
		Round: 2000,
		StockEvents: map[int][]health.StockEvent{
			101: {
				{DepletedRound: 500, RefilledRound: 800},   // before old.Round(1000) → excluded
				{DepletedRound: 1200, RefilledRound: 1500}, // TtR=300, after old.Round → included
			},
		},
	}
	old := health.ShopSnapshot{
		Zone: "z", MobId: 1, RoomId: 1,
		Round: 1000,
	}
	d := health.ComputeShopDelta(now, &old)
	if d.MedianTtR != 300 {
		t.Errorf("MedianTtR: got %d, want 300", d.MedianTtR)
	}
}

func TestComputeShopDelta_NilOldReturnsZeroDeltas(t *testing.T) {
	now := health.ShopSnapshot{
		Zone: "z", MobId: 1, RoomId: 1, SalesCount: 10, BuysCount: 5,
	}
	d := health.ComputeShopDelta(now, nil)
	if d.SalesDelta != 0 || d.BuysDelta != 0 || d.RestocksDelta != 0 || d.MedianTtR != 0 {
		t.Errorf("nil old: expected zero counter deltas, got sales=%d buys=%d restocks=%d ttr=%d",
			d.SalesDelta, d.BuysDelta, d.RestocksDelta, d.MedianTtR)
	}
}

func TestComputeCaravanDelta_LbsDelivered(t *testing.T) {
	now := health.CaravanSnapshot{
		InstId:       1001,
		LbsDelivered: 500,
		DeliveriesByTier: map[int]int{50: 10},
	}
	old := health.CaravanSnapshot{
		InstId:       1001,
		LbsDelivered: 200,
		DeliveriesByTier: map[int]int{50: 5},
	}
	d := health.ComputeCaravanDelta(now, &old)
	if d.LbsDeliveredDelta != 300 {
		t.Errorf("LbsDeliveredDelta: got %d, want 300", d.LbsDeliveredDelta)
	}
}

func TestComputeForagerDelta_LbsDelivered(t *testing.T) {
	now := health.ForagerSnapshot{
		MobId:        201,
		LbsDelivered: 750,
		DeliveriesByTier: map[int]int{50: 20},
	}
	old := health.ForagerSnapshot{
		MobId:        201,
		LbsDelivered: 400,
		DeliveriesByTier: map[int]int{50: 12},
	}
	d := health.ComputeForagerDelta(now, &old)
	if d.LbsDeliveredDelta != 350 {
		t.Errorf("LbsDeliveredDelta: got %d, want 350", d.LbsDeliveredDelta)
	}
}
