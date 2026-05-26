package caravan

import (
	"fmt"
	"math"
	"slices"

	"github.com/GoMudEngine/GoMud/internal/economy"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
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
// Pickup is gated by entry.Current > entry.RestockQty — the vendor
// keeps at least one normal restock-batch as buffer for local
// customers. Anything beyond that is fair game for cross-city
// distribution. Pre-3.8 the gate was entry.MaxStock/2 which was
// too restrictive against player-driven stock depletion, leaving
// cross-city flow effectively dead for partly-stocked vendors.
//
// Pickup quantity is max(1, Current/10) — gently drain 10% of
// available stock, with a 1-unit floor so partially-stocked
// vendors still contribute. Slowly builds the wagon's cross-city
// surplus without looting any single vendor.
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
				// Increment throughput counters for delivery tracking.
				spec := items.GetItemSpec(item.ItemId)
				if spec != nil {
					if spec.RarityTier > 0 {
						IncrementDelivery(wagon.Zone, int(wagon.MobId), spec.RarityTier)
					}
					AddLbsDelivered(wagon.Zone, int(wagon.MobId), uint64(math.Round(spec.Weight)))
				}
				delivered = append(delivered, ItemMove{
					Vendor:   vendor.Character.Name,
					ItemName: item.DisplayName(),
				})
			}
		}

		// PICKUP pass: vendor → wagon.
		// Iterate vendor stock; extract bucket-matching entries with
		// Current > RestockQty (vendor keeps one restock-batch buffer
		// for local customers). Take 10% of available stock, floor of 1.
		// Chunk 3.8: tuned vs the pre-3.8 MaxStock/2 gate which was
		// too restrictive given player-driven stock depletion.
		if len(pickupBuckets) > 0 {
			for i := range shop.Stock {
				entry := &shop.Stock[i]
				bucket := economy.BucketFor(entry.ItemId)
				if bucket == "" || !slices.Contains(pickupBuckets, bucket) {
					continue
				}
				if entry.Current <= entry.RestockQty {
					continue // keep at least one restock batch for local
				}
				qty := entry.Current / 10
				if qty < 1 {
					qty = 1
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
// runnerName is the visible source mob — chunk 3.8 makes Lars (Ketil's
// son) the runner who carries cargo wagon↔vendor while the wagon stays
// parked at the depot. Pre-3.8 the message said "the caravan crew" but
// only one mob is actually in the room with the vendor now; naming the
// runner reads better.
//
// Vendor name pulled from the first ItemMove's Vendor field (all moves
// in a single call are at the same vendor). Falls back to "the local
// merchant" if both slices are empty (defensive — caller skips on
// empty).
//
// Three flavor variants:
//   - Both delivered + picked up — "trades supplies for cargo" wording
//   - Delivery only — "unloads a satchel of supplies"
//   - Pickup only — "loads up a satchel of cargo"
func FormatVisitMessage(runnerName string, delivered, pickedUp []ItemMove) string {
	vendor := "the local merchant"
	switch {
	case len(delivered) > 0:
		vendor = delivered[0].Vendor
	case len(pickedUp) > 0:
		vendor = pickedUp[0].Vendor
	}
	switch {
	case len(delivered) > 0 && len(pickedUp) > 0:
		return fmt.Sprintf(
			`<ansi fg="yellow"><ansi fg="mobname">%s</ansi> trades a satchel of supplies for fresh cargo with <ansi fg="mobname">%s</ansi>.</ansi>`,
			runnerName, vendor)
	case len(delivered) > 0:
		return fmt.Sprintf(
			`<ansi fg="yellow"><ansi fg="mobname">%s</ansi> unloads a satchel of supplies for <ansi fg="mobname">%s</ansi>.</ansi>`,
			runnerName, vendor)
	case len(pickedUp) > 0:
		return fmt.Sprintf(
			`<ansi fg="yellow"><ansi fg="mobname">%s</ansi> loads up a satchel of fresh cargo from <ansi fg="mobname">%s</ansi>.</ansi>`,
			runnerName, vendor)
	}
	return ""
}
