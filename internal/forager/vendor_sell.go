package forager

import (
	"fmt"
	"math"
	"slices"

	"github.com/GoMudEngine/GoMud/internal/economy"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

// SellToVendor transfers bucket-matching items from the forager's
// satchel (mob.Character.Items) into matching vendor stock entries at
// the given room. Items whose bucket isn't in p.Buckets are skipped.
// Items that don't fit (vendor at MaxStock, or no matching stock
// entry) stay in the satchel for the next vendor or next delivery
// cycle.
//
// Replaces the abstract RestockBuckets call from Stage 3.1. Moved
// from internal/behaviortree/actions_forager.go to internal/forager
// in chunk 3.8 to break the import cycle the new arrival listener
// would otherwise introduce.
func SellToVendor(roomId int, p *ForagerProfile, mob *mobs.Mob) {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		vendor := mobs.GetInstance(instId)
		if vendor == nil || !vendor.HasShop() {
			continue
		}
		shop := shops.GetShopInventory(vendor.Zone, int(vendor.MobId), roomId)
		if shop == nil {
			continue
		}
		// Track whether we mutated this vendor's stock so we only
		// persist when something actually transferred.
		mutated := false
		// Walk forager satchel in reverse so RemoveItem is index-safe.
		for i := len(mob.Character.Items) - 1; i >= 0; i-- {
			item := mob.Character.Items[i]
			bucket := economy.BucketFor(item.ItemId)
			if bucket == "" || !slices.Contains(p.Buckets, bucket) {
				continue
			}
			entry := shop.GetStock(item.ItemId)
			if entry == nil || entry.Current >= entry.MaxStock {
				continue
			}
			mob.Character.RemoveItem(item)
			entry.Current++
			mutated = true
			// Increment throughput counters for delivery tracking.
			spec := items.GetItemSpec(item.ItemId)
			if spec != nil {
				if spec.RarityTier > 0 {
					IncrementDelivery(mob.Zone, int(mob.MobId), spec.RarityTier)
				}
				AddLbsDelivered(mob.Zone, int(mob.MobId), uint64(math.Round(spec.Weight)))
			}
			room.SendText(messaging.CategoryMobEmote, fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> hands a %s to`+
					` <ansi fg="mobname">%s</ansi>.`,
				p.Name, item.DisplayName(), vendor.Character.Name,
			))
		}
		// Persist when stock actually changed. Mirrors the caravan-side
		// crash-safety pattern in internal/caravan/visit.go — without
		// this, forager restocks only hit disk on graceful shutdown
		// and a panic loses an in-flight cycle's deliveries.
		if mutated {
			if err := shops.SaveShop(vendor.Zone, int(vendor.MobId), roomId); err != nil {
				mudlog.Error("forager.SellToVendor", "forager", p.Name, "vendor", vendor.Character.Name, "error", err)
			}
			if err := SaveThroughput(mob.Zone, int(mob.MobId)); err != nil {
				mudlog.Error("forager.SaveThroughput", "forager", p.Name, "error", err)
			}
		}
	}
}
