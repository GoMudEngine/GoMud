package forager

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

// selectBackfillTransfers decides how many of each item to pull from the
// aggregate chest pool into the vendor, neediest stock gap first, capped by
// each entry's MaxStock and by pool availability. Only items the vendor already
// stocks (has a StockEntry for, below MaxStock) are eligible. Pure + testable.
func selectBackfillTransfers(si *shops.ShopInventory, pool map[int]int) map[int]int {
	type gap struct {
		itemId int
		gap    int
	}
	var gaps []gap
	for i := range si.Stock {
		e := &si.Stock[i]
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

// loadRoomFn is a seam so chestPoolForZone is testable without disk room data.
// Production uses rooms.LoadRoom; tests override it.
var loadRoomFn = rooms.LoadRoom

// chestPoolForZone aggregates item counts across the forager lockboxes
// registered for the given zone (via the chest index — no instance scan).
// Returns the pool (itemId -> count) plus the chest room ids so the transfer
// step can remove from the right container.
func chestPoolForZone(zone string) (pool map[int]int, chestRooms []int) {
	pool = map[int]int{}
	for _, chestRoom := range ChestRoomsForZone(zone) {
		room := loadRoomFn(chestRoom)
		if room == nil {
			continue
		}
		key := room.FindContainerByName("lockbox")
		if key == "" {
			continue
		}
		c := room.Containers[key]
		empty := true
		for _, it := range c.Items {
			pool[it.ItemId]++
			empty = false
		}
		if !empty {
			chestRooms = append(chestRooms, chestRoom)
		}
	}
	return pool, chestRooms
}

// BackfillVendorFromChests tops off vendorMob's shop from forager chests in its
// zone. Free supply handoff — no gold. Mirrors SellToVendor's persistence.
func BackfillVendorFromChests(vendorMob *mobs.Mob, shopInv *shops.ShopInventory) {
	if vendorMob == nil || shopInv == nil {
		return
	}
	pool, chestRooms := chestPoolForZone(vendorMob.Zone)
	if len(pool) == 0 {
		return
	}
	transfers := selectBackfillTransfers(shopInv, pool)
	if len(transfers) == 0 {
		return
	}

	mutated := false
	for itemId, want := range transfers {
		moved := 0
		for _, chestRoomId := range chestRooms {
			if moved >= want {
				break
			}
			room := loadRoomFn(chestRoomId)
			if room == nil {
				continue
			}
			key := room.FindContainerByName("lockbox")
			if key == "" {
				continue
			}
			c := room.Containers[key]
			for moved < want {
				it, ok := c.FindItemById(itemId)
				if !ok {
					break
				}
				c.RemoveItem(it)
				entry := shopInv.GetStock(itemId)
				if entry == nil || entry.Current >= entry.MaxStock {
					// Shouldn't happen (selectBackfillTransfers capped it), but
					// put the item back rather than vaporize it.
					c.AddItem(it)
					break
				}
				entry.Current++
				moved++
				mutated = true
			}
			room.Containers[key] = c
		}
	}

	if mutated {
		if err := shops.SaveShop(vendorMob.Zone, int(vendorMob.MobId), vendorMob.HomeRoomId); err != nil {
			mudlog.Error("forager.BackfillVendorFromChests", "vendor", vendorMob.Character.Name, "error", err)
		}
	}
}
