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

	// Phase-4 counter deltas — set by ComputeShopDelta when both
	// snapshots have data. Zero when the old snapshot predates Phase 2.
	SalesDelta    int    `json:"sales_delta"`
	BuysDelta     int    `json:"buys_delta"`
	RestocksDelta int    `json:"restocks_delta"`
	MedianTtR     uint64 // median TtR (rounds) for events completed in this window
}

// ForagerDelta captures per-forager changes between snapshots.
type ForagerDelta struct {
	DeliveriesByTierDelta map[int]int
	StuckRoundsDelta      int64
	LbsDeliveredDelta     uint64 `json:"lbs_delivered_delta"`
}

// CaravanDelta captures per-caravan changes between snapshots.
type CaravanDelta struct {
	DeliveriesByTierDelta map[int]int
	LbsDeliveredDelta     uint64 `json:"lbs_delivered_delta"`
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

	// Counter deltas: always non-negative (counters only increment).
	salesDelta := now.SalesCount - old.SalesCount
	if salesDelta > 0 {
		d.SalesDelta = salesDelta
	}
	buysDelta := now.BuysCount - old.BuysCount
	if buysDelta > 0 {
		d.BuysDelta = buysDelta
	}
	restocksDelta := now.RestockCount - old.RestockCount
	if restocksDelta > 0 {
		d.RestocksDelta = restocksDelta
	}

	// MedianTtR: median of TtR durations for StockEvents whose
	// RefilledRound falls strictly after old.Round (the game-round
	// recorded on the old snapshot). If old.Round is 0 (pre-Phase-4
	// snapshot that lacked the field) we skip.
	if old.Round > 0 {
		d.MedianTtR = computeMedianTtR(now, old.Round)
	}

	return d
}

// computeMedianTtR collects TtR durations (RefilledRound - DepletedRound)
// for all StockEvents across all items where RefilledRound > windowStart,
// then returns the median. Returns 0 if there are no qualifying events.
func computeMedianTtR(snap ShopSnapshot, windowStart uint64) uint64 {
	var durations []uint64
	for _, evts := range snap.StockEvents {
		for _, ev := range evts {
			if ev.RefilledRound == 0 {
				continue // event still open (currently depleted)
			}
			if ev.RefilledRound <= windowStart {
				continue // event completed before the window
			}
			if ev.RefilledRound > ev.DepletedRound {
				durations = append(durations, ev.RefilledRound-ev.DepletedRound)
			}
		}
	}
	if len(durations) == 0 {
		return 0
	}
	// Sort to find the median.
	sortUint64(durations)
	return durations[len(durations)/2]
}

// sortUint64 is a simple insertion sort for small slices (TtR events
// per shop are typically <20; no need for a full sort import).
func sortUint64(s []uint64) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
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
	// Guard against uint64 underflow (cumulative counter may wrap if
	// a forager is replaced between snapshots).
	if now.LbsDelivered >= old.LbsDelivered {
		d.LbsDeliveredDelta = now.LbsDelivered - old.LbsDelivered
	}
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
	// Guard against uint64 underflow (cumulative counter may wrap if
	// a caravan is replaced between snapshots).
	if now.LbsDelivered >= old.LbsDelivered {
		d.LbsDeliveredDelta = now.LbsDelivered - old.LbsDelivered
	}
	return d
}
