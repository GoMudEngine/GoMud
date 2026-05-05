package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/economy"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"slices"
)

// ItemMove describes a single item that moved between wagon and a vendor.
type ItemMove struct {
	Vendor   string
	ItemName string
}

// VisitVendorsInRoom performs a bidirectional vendor-stop for the
// caravan: deliver wagon items whose bucket is in deliveryBuckets,
// and pick up vendor items whose bucket is in pickupBuckets.
//
// Pickup is gated by entry.Current >= entry.MaxStock/2 — caravan
// won't extract from a starving vendor (narrative: wholesalers
// don't loot a struggling shop).
//
// Pickup quantity is RestockQty per matching stock entry, capped
// at the wagon's CarryCapacity remaining (StoreItem returns false
// when full).
//
// Returns (nil, nil) if the room doesn't exist OR the wagon is nil.
//
// Empty/nil buckets means "skip that pass" — pass nil for both to
// no-op cleanly.
func VisitVendorsInRoom(
	roomId int,
	wagon *mobs.Mob,
	deliveryBuckets []string,
	pickupBuckets []string,
) (delivered, pickedUp []ItemMove) {
	if wagon == nil {
		return nil, nil
	}
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return nil, nil
	}

	for _, instId := range room.GetMobs(rooms.FindAll) {
		vendor := mobs.GetInstance(instId)
		if vendor == nil || !vendor.HasShop() {
			continue
		}
		shop := shops.GetShopInventory(vendor.Zone, int(vendor.MobId), vendor.HomeRoomId)
		if shop == nil {
			continue
		}

		// Track whether we mutated this vendor's stock so we only
		// persist when something actually changed.
		mutated := false

		// DELIVER pass: wagon → vendor.
		// Walk wagon items in reverse so RemoveItem is index-safe.
		if len(deliveryBuckets) > 0 {
			for i := len(wagon.Character.Items) - 1; i >= 0; i-- {
				item := wagon.Character.Items[i]
				bucket := economy.BucketFor(item.ItemId)
				if bucket == "" || !slices.Contains(deliveryBuckets, bucket) {
					continue
				}
				entry := shop.GetStock(item.ItemId)
				if entry == nil || entry.Current >= entry.MaxStock {
					continue
				}
				wagon.Character.RemoveItem(item)
				entry.Current++
				mutated = true
				// Increment throughput counter for delivery tracking.
				spec := items.GetItemSpec(item.ItemId)
				if spec != nil && spec.RarityTier > 0 {
					IncrementDelivery(wagon.Zone, int(wagon.MobId), spec.RarityTier)
				}
				delivered = append(delivered, ItemMove{
					Vendor:   vendor.Character.Name,
					ItemName: item.DisplayName(),
				})
			}
		}

		// PICKUP pass: vendor → wagon.
		// Iterate vendor stock; extract bucket-matching entries when
		// Current >= MaxStock/2 (the supply-floor gate).
		if len(pickupBuckets) > 0 {
			for i := range shop.Stock {
				entry := &shop.Stock[i]
				bucket := economy.BucketFor(entry.ItemId)
				if bucket == "" || !slices.Contains(pickupBuckets, bucket) {
					continue
				}
				if entry.Current < entry.MaxStock/2 {
					continue
				}
				qty := entry.RestockQty
				if qty <= 0 {
					continue
				}
				if qty > entry.Current {
					qty = entry.Current
				}
				for j := 0; j < qty; j++ {
					newItem := items.New(entry.ItemId)
					if !newItem.IsValid() {
						break
					}
					if !wagon.Character.StoreItem(newItem) {
						break // wagon at carry cap
					}
					entry.Current--
					mutated = true
					pickedUp = append(pickedUp, ItemMove{
						Vendor:   vendor.Character.Name,
						ItemName: newItem.DisplayName(),
					})
				}
			}
		}

		// Persist when stock actually changed. Crash-safe: without
		// this, caravan restocks only persist on graceful shutdown,
		// so a panic loses an in-flight cycle's deliveries. Non-fatal
		// on write error — the in-memory state stays live and
		// graceful shutdown will retry the write.
		if mutated {
			if err := shops.SaveShop(vendor.Zone, int(vendor.MobId), vendor.HomeRoomId); err != nil {
				mudlog.Error("caravan.VisitVendorsInRoom", "vendor", vendor.Character.Name, "error", err)
			}
			if err := SaveThroughput(wagon.Zone, int(wagon.MobId)); err != nil {
				mudlog.Error("caravan.SaveThroughput", "wagon", wagon.Character.Name, "error", err)
			}
		}
	}

	return delivered, pickedUp
}

// FormatVisitMessage builds the room-flavor text for a vendor stop.
// Returns "" when no transfers happened (caller should skip sending).
//
// Three flavor variants:
//   - Both delivered + picked up — "in trade" wording
//   - Delivery only — "unloads supplies"
//   - Pickup only — "loads up cargo"
func FormatVisitMessage(delivered, pickedUp []ItemMove) string {
	switch {
	case len(delivered) > 0 && len(pickedUp) > 0:
		return `<ansi fg="yellow">Marta hands a small purse across the counter; the caravan unloads and reloads in trade.</ansi>`
	case len(delivered) > 0:
		return `<ansi fg="yellow">The caravan crew unloads supplies for the local merchants.</ansi>`
	case len(pickedUp) > 0:
		return `<ansi fg="yellow">The caravan crew loads up cargo from the local merchants for the road.</ansi>`
	}
	return ""
}
