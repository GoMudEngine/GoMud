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
	GoldDelta    int
	BucketDeltas map[string]int
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
