package health

// PickClosestSnapshot returns the meta whose UnixTs is nearest to
// target, but only within toleranceSeconds. Returns nil if no meta
// is within tolerance. metas is expected to be sorted descending by
// UnixTs (the order ListSnapshots returns).
func PickClosestSnapshot(metas []SnapshotMeta, target int64, toleranceSeconds int64) *SnapshotMeta {
	var best *SnapshotMeta
	var bestDelta int64 = -1
	for i := range metas {
		m := metas[i]
		delta := m.UnixTs - target
		if delta < 0 {
			delta = -delta
		}
		if toleranceSeconds > 0 && delta > toleranceSeconds {
			continue
		}
		if bestDelta < 0 || delta < bestDelta {
			bestDelta = delta
			best = &metas[i]
		}
	}
	return best
}

// ShopDelta is the per-shop delta vs a comparison snapshot. Bucket
// deltas are sums of per-bucket Current values (this snapshot − old).
type ShopDelta struct {
	GoldDelta       int
	BucketDeltas    map[string]int
	StockScoreDelta int `json:"stock_score_delta"` // percentage points (now - old) × 100, integer
}

// ForagerDelta captures per-forager changes between snapshots.
type ForagerDelta struct {
	DeliveriesByTierDelta map[int]int
	StuckRoundsDelta      int64
}

// CaravanDelta captures per-caravan changes between snapshots.
type CaravanDelta struct {
	DeliveriesByTierDelta map[int]int
}

// ComputeShopDelta returns the shop's delta against old. If old is
// nil, returns a zero-value delta with empty BucketDeltas (the UI
// should distinguish "no comparison snapshot" from "zero change").
func ComputeShopDelta(now ShopSnapshot, old *ShopSnapshot) ShopDelta {
	d := ShopDelta{BucketDeltas: map[string]int{}}
	if old == nil {
		return d
	}
	d.GoldDelta = now.Gold - old.Gold

	bucketSum := func(s ShopSnapshot) map[string]int {
		out := map[string]int{}
		for _, e := range s.Stock {
			if e.Bucket == "" {
				continue
			}
			out[e.Bucket] += e.Current
		}
		return out
	}
	curBuckets := bucketSum(now)
	oldBuckets := bucketSum(*old)
	for b, n := range curBuckets {
		d.BucketDeltas[b] = n - oldBuckets[b]
	}
	for b, n := range oldBuckets {
		if _, seen := curBuckets[b]; !seen {
			d.BucketDeltas[b] = -n
		}
	}

	d.StockScoreDelta = int((now.StockScore - old.StockScore) * 100)
	return d
}

// FindShopInSnapshot returns a pointer to the matching shop in s, or
// nil if absent. Match key is (Zone, MobId, RoomId).
func FindShopInSnapshot(s *Snapshot, zone string, mobId, roomId int) *ShopSnapshot {
	if s == nil {
		return nil
	}
	for i := range s.Shops {
		if s.Shops[i].Zone == zone && s.Shops[i].MobId == mobId && s.Shops[i].RoomId == roomId {
			return &s.Shops[i]
		}
	}
	return nil
}

// FindForagerInSnapshot returns a pointer to the matching forager in s,
// or nil if absent. Match key is MobId.
func FindForagerInSnapshot(s *Snapshot, mobId int) *ForagerSnapshot {
	if s == nil {
		return nil
	}
	for i := range s.Foragers {
		if s.Foragers[i].MobId == mobId {
			return &s.Foragers[i]
		}
	}
	return nil
}

// FindCaravanInSnapshot returns a pointer to the matching caravan in s,
// or nil if absent. Match key is InstId.
func FindCaravanInSnapshot(s *Snapshot, instId int) *CaravanSnapshot {
	if s == nil {
		return nil
	}
	for i := range s.Caravans {
		if s.Caravans[i].InstId == instId {
			return &s.Caravans[i]
		}
	}
	return nil
}

// ComputeForagerDelta returns the forager's delta against old. If old is
// nil, returns a zero-value delta with empty DeliveriesByTierDelta.
func ComputeForagerDelta(now ForagerSnapshot, old *ForagerSnapshot) ForagerDelta {
	d := ForagerDelta{DeliveriesByTierDelta: map[int]int{}}
	if old == nil {
		return d
	}
	for tier, count := range now.DeliveriesByTier {
		d.DeliveriesByTierDelta[tier] = count - old.DeliveriesByTier[tier]
	}
	for tier, count := range old.DeliveriesByTier {
		if _, seen := now.DeliveriesByTier[tier]; !seen {
			d.DeliveriesByTierDelta[tier] = -count
		}
	}
	d.StuckRoundsDelta = int64(now.StuckRounds) - int64(old.StuckRounds)
	return d
}

// ComputeCaravanDelta returns the caravan's delta against old. If old is
// nil, returns a zero-value delta with empty DeliveriesByTierDelta.
func ComputeCaravanDelta(now CaravanSnapshot, old *CaravanSnapshot) CaravanDelta {
	d := CaravanDelta{DeliveriesByTierDelta: map[int]int{}}
	if old == nil {
		return d
	}
	for tier, count := range now.DeliveriesByTier {
		d.DeliveriesByTierDelta[tier] = count - old.DeliveriesByTier[tier]
	}
	for tier, count := range old.DeliveriesByTier {
		if _, seen := now.DeliveriesByTier[tier]; !seen {
			d.DeliveriesByTierDelta[tier] = -count
		}
	}
	return d
}
