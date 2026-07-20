package forager

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// loadRoomFn is a seam so chestPoolForZone is testable without disk room data.
// Production uses rooms.LoadRoom; tests override it.
var loadRoomFn = rooms.LoadRoom

// chestPoolFromRooms aggregates item counts across the given chest rooms.
func chestPoolFromRooms(chestRooms []int) (pool map[int]int, nonEmpty []int) {
	pool = map[int]int{}
	for _, chestRoom := range chestRooms {
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
			nonEmpty = append(nonEmpty, chestRoom)
		}
	}
	return pool, nonEmpty
}

func chestPoolForZone(zone string) (map[int]int, []int) {
	return chestPoolFromRooms(ChestRoomsForZone(zone))
}
func chestPoolAll() (map[int]int, []int) { return chestPoolFromRooms(ChestRoomsAll()) }

// BackfillVendorFromChests tops off vendorMob's shop from the global pool of
// all forager chests (aggregated across every zone). Free supply handoff — no
// gold. Mirrors SellToVendor's persistence.
func BackfillVendorFromChests(vendorMob *mobs.Mob, shopInv *shops.ShopInventory) {
	if vendorMob == nil || shopInv == nil {
		return
	}
	pool, chestRooms := chestPoolAll()
	if len(pool) == 0 {
		return
	}
	transfers := shops.SelectStockTransfers(shopInv, pool)
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
					// Shouldn't happen (SelectStockTransfers capped it), but
					// put the item back rather than vaporize it.
					c.AddItem(it)
					break
				}
				entry.Current++
				entry.LastGrewRound = util.GetRoundCount()
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
