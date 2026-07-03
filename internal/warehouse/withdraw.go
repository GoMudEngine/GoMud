package warehouse

// Withdraw removes up to qty of an item from a city's pool, returning the
// amount actually withdrawn (0 for unknown zones / empty stock). Increments
// DrawnCount and marks the zone dirty. Stage 4 drawdown's only exit path —
// callers mint the physical items (items.New) for what they receive.
func Withdraw(zone string, itemId int, qty int) int {
	if qty <= 0 {
		return 0
	}
	if _, ok := cities[zone]; !ok {
		return 0
	}
	mu.Lock()
	defer mu.Unlock()
	w := getOrCreateLocked(zone)
	for i := range w.Stock {
		if w.Stock[i].ItemId != itemId {
			continue
		}
		take := qty
		if take > w.Stock[i].Current {
			take = w.Stock[i].Current
		}
		if take <= 0 {
			return 0
		}
		w.Stock[i].Current -= take
		w.DrawnCount += take
		dirty[zone] = true
		return take
	}
	return 0
}
