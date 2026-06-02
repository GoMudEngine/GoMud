package shops

import "sort"

// SelectStockTransfers decides how many of each item to pull from `pool` into
// shopInv, neediest stock-gap first (gap = MaxStock-Current), capped by each
// entry's gap and by pool availability. Only items shopInv already stocks (with
// a gap) are eligible. Pure. (Lifted from forager.selectBackfillTransfers so the
// chest backfill and the enchanting-reserve draw share one allocator.)
func SelectStockTransfers(shopInv *ShopInventory, pool map[int]int) map[int]int {
	type gap struct {
		itemId int
		gap    int
	}
	var gaps []gap
	for i := range shopInv.Stock {
		e := &shopInv.Stock[i]
		g := e.MaxStock - e.Current
		if g > 0 && pool[e.ItemId] > 0 {
			gaps = append(gaps, gap{e.ItemId, g})
		}
	}
	sort.Slice(gaps, func(a, b int) bool {
		if gaps[a].gap != gaps[b].gap {
			return gaps[a].gap > gaps[b].gap
		}
		return gaps[a].itemId < gaps[b].itemId
	})
	remaining := map[int]int{}
	for id, n := range pool {
		remaining[id] = n
	}
	out := map[int]int{}
	for _, g := range gaps {
		take := g.gap
		if take > remaining[g.itemId] {
			take = remaining[g.itemId]
		}
		if take > 0 {
			out[g.itemId] = take
			remaining[g.itemId] -= take
		}
	}
	return out
}
