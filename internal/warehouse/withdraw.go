package warehouse

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

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

// ReleaseToVendorsInRoom tops up existing vendor stock entries in one room
// from the local warehouse, at most maxPerItem per entry per call (slow
// release by design). Never creates slots; never touches non-warehouse
// zones. Returns units released. Callers gate on the drawdown toggle.
//
// No release-flavor emote — the carrier's own delivery pass already
// narrates the stop; a second message for the invisible backend top-up is
// future polish (not v1).
func ReleaseToVendorsInRoom(zone string, roomId int, maxPerItem int) int {
	if _, ok := cities[zone]; !ok || maxPerItem <= 0 {
		return 0
	}
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return 0
	}
	released := 0
	for _, instId := range room.GetMobs(rooms.FindMerchant) {
		vendor := mobs.GetInstance(instId)
		if vendor == nil || !vendor.HasShop() {
			continue
		}
		shop := shops.GetShopInventory(vendor.Zone, int(vendor.MobId), vendor.HomeRoomId)
		if shop == nil {
			continue
		}
		mutated := false
		for i := range shop.Stock {
			e := &shop.Stock[i]
			gap := e.MaxStock - e.Current
			if gap <= 0 {
				continue
			}
			want := gap
			if want > maxPerItem {
				want = maxPerItem
			}
			got := Withdraw(zone, e.ItemId, want)
			if got > 0 {
				e.Current += got
				released += got
				mutated = true
			}
		}
		if mutated {
			if err := shops.SaveShop(vendor.Zone, int(vendor.MobId), vendor.HomeRoomId); err != nil {
				mudlog.Error("warehouse.ReleaseToVendorsInRoom", "error", err)
			}
		}
	}
	return released
}
