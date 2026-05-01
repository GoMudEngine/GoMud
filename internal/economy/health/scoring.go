package health

// PerShopScore returns a 0-100 health score for one shop, weighted by
// RestockQty. Shops with no stock entries return 0; callers that want
// to distinguish "no data" from "zero" should use PerShopScoreOpt.
func PerShopScore(s ShopSnapshot) float64 {
	v, _ := PerShopScoreOpt(s)
	return v
}

// PerShopScoreOpt returns (score, true) when the shop has stock data
// to score, or (0, false) when there is no signal.
func PerShopScoreOpt(s ShopSnapshot) (float64, bool) {
	if len(s.Stock) == 0 {
		return 0, false
	}
	var weightedSum float64
	var totalWeight float64
	for _, e := range s.Stock {
		if e.Max <= 0 {
			continue
		}
		fill := float64(e.Current) / float64(e.Max)
		if fill < 0 {
			fill = 0
		}
		if fill > 1 {
			fill = 1
		}
		weight := float64(e.RestockQty)
		if weight < 1 {
			weight = 1
		}
		weightedSum += weight * fill
		totalWeight += weight
	}
	if totalWeight == 0 {
		return 0, false
	}
	return 100 * weightedSum / totalWeight, true
}

// PerCraftSupportScores returns mean per-shop score grouped by the
// craft discipline each shop supports. Shops with empty CraftSupport
// roll into key "" (should never happen in production thanks to
// startup validation; surfaces clearly in the UI as "(uncategorized)"
// if it ever does).
func PerCraftSupportScores(snap Snapshot) map[string]float64 {
	type bucket struct {
		sum   float64
		count int
	}
	buckets := map[string]*bucket{}
	for _, s := range snap.Shops {
		score, ok := PerShopScoreOpt(s)
		if !ok {
			continue
		}
		b, exists := buckets[s.CraftSupport]
		if !exists {
			b = &bucket{}
			buckets[s.CraftSupport] = b
		}
		b.sum += score
		b.count++
	}
	out := map[string]float64{}
	for k, b := range buckets {
		if b.count > 0 {
			out[k] = b.sum / float64(b.count)
		}
	}
	return out
}
