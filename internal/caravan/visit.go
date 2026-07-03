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

// VisitOpts controls a vendor-visit pass.
type VisitOpts struct {
	DeliveryBuckets []string
	PickupBuckets   []string
	// CreateMissingSlots lets a delivery CREATE a StockEntry the vendor
	// never stocked (RestockQty stays 0: the slot is carrier-fed only —
	// the restock ticker never touches RestockQty<=0 entries). Used by
	// ferry trade factors to seed cross-region goods. Legacy caravan and
	// forager deliveries keep this off so random cargo can't invent
	// shelf space.
	CreateMissingSlots bool
	NewSlotMaxStock    int // MaxStock for created slots (required when CreateMissingSlots)
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
//
// VisitVendorsInRoom preserves the legacy behavior (no slot creation).
// New callers that need opt-in missing-slot creation (the ferry trade
// factors) should call VisitVendorsInRoomOpts directly.
func VisitVendorsInRoom(
	roomId int,
	wagon *mobs.Mob,
	deliveryBuckets []string,
	pickupBuckets []string,
) (delivered, pickedUp []ItemMove) {
	return VisitVendorsInRoomOpts(roomId, wagon, VisitOpts{
		DeliveryBuckets: deliveryBuckets,
		PickupBuckets:   pickupBuckets,
	})
}

// VisitVendorsInRoomOpts is the options-driven form of VisitVendorsInRoom.
// See VisitOpts for the CreateMissingSlots contract.
func VisitVendorsInRoomOpts(
	roomId int,
	wagon *mobs.Mob,
	opts VisitOpts,
) (delivered, pickedUp []ItemMove) {
	deliveryBuckets := opts.DeliveryBuckets
	pickupBuckets := opts.PickupBuckets
	if wagon == nil {
		return nil, nil
	}
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return nil, nil
	}

	// Warn once per call (not once per item) when the caller asked for
	// slot creation but didn't give it a usable cap — every missing-slot
	// item will silently fall back to legacy skip behavior below.
	warnedMisconfiguredCreate := false

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

		// Track whether we touched the wagon's throughput counters. Only the
		// DELIVER pass updates throughput; a pickup-only stop must NOT call
		// SaveThroughput, which would log a spurious "no cached entry" error
		// because nothing seeded the wagon's throughput entry.
		throughputChanged := false

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
				if entry == nil {
					if opts.CreateMissingSlots && opts.NewSlotMaxStock <= 0 && !warnedMisconfiguredCreate {
						mudlog.Warn("caravan.VisitVendorsInRoomOpts", "warn",
							"CreateMissingSlots set but NewSlotMaxStock <= 0 — skipping slot creation",
							"roomId", roomId, "wagon", wagon.Character.Name)
						warnedMisconfiguredCreate = true
					}
					if !opts.CreateMissingSlots || opts.NewSlotMaxStock <= 0 {
						continue
					}
					shop.Stock = append(shop.Stock, shops.StockEntry{
						ItemId:   item.ItemId,
						MaxStock: opts.NewSlotMaxStock,
						// RestockQty deliberately 0: ferry-fed only —
						// the restock ticker skips RestockQty<=0
						// entries (see ShopInventory.Restock).
					})
					entry = shop.GetStock(item.ItemId)
				}
				if entry.Current >= entry.MaxStock {
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
						throughputChanged = true
					}
					AddLbsDelivered(wagon.Zone, int(wagon.MobId), uint64(math.Round(spec.Weight)))
					if spec.Weight > 0 {
						throughputChanged = true
					}
				}
				delivered = append(delivered, ItemMove{
					Vendor:   vendor.Character.Name,
					ItemName: item.DisplayName(),
				})
			}
		}

		// PICKUP pass: vendor → wagon.
		// Iterate vendor stock; extract qualifying entries with
		// Current > RestockQty (vendor keeps one restock-batch buffer
		// for local customers). Take 10% of available stock, floor of 1.
		//
		// "Qualifying" (see pickupQualifies) is EITHER:
		//   1. The item's bucket is in pickupBuckets (zone-source path),
		//      OR
		//   2. The item is flagged IsComponent and its ItemType is a
		//      raw-material type (not a finished good). This lets common
		//      crafting materials (iron ingot, steel ingot, coal dust)
		//      flow freely cross-city even when their bucket is "base"
		//      or "overlap" and not in pickupBuckets.
		//
		// Chunk 3.8: tuned vs the pre-3.8 MaxStock/2 gate which was
		// too restrictive given player-driven stock depletion.
		// Chunk 3.8 hotfix (H2): expanded qualifying gate to include
		// any IsComponent raw material, not just zone-source bucket.
		if len(pickupBuckets) > 0 {
			for i := range shop.Stock {
				entry := &shop.Stock[i]
				if !pickupQualifies(entry.ItemId, pickupBuckets) {
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
			// Only persist throughput if the deliver pass actually updated it.
			if throughputChanged {
				if err := SaveThroughput(wagon.Zone, int(wagon.MobId)); err != nil {
					mudlog.Error("caravan.SaveThroughput", "wagon", wagon.Character.Name, "error", err)
				}
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

// pickupQualifies reports whether the vendor stock entry should be
// considered for caravan pickup. Two qualifying paths:
//
//  1. Zone-source bucket match — the item is in pickupBuckets per
//     economy.BucketFor. Pre-3.8-hotfix this was the only path.
//
//  2. Component-and-not-finished-good — the item is flagged
//     IsComponent (raw material), AND its Type is not a finished
//     good (weapon, armor, etc.). Lets crafting materials flow
//     freely cross-city without sweeping up crafted output.
//
// Returns false defensively when the item spec can't be resolved.
func pickupQualifies(itemId int, pickupBuckets []string) bool {
	bucket := economy.BucketFor(itemId)
	if bucket != "" && slices.Contains(pickupBuckets, bucket) {
		return true
	}
	spec := items.GetItemSpec(itemId)
	if spec == nil || !spec.IsComponent {
		return false
	}
	return !isFinishedGood(spec.Type)
}

// isFinishedGood reports whether an item type represents a player-
// usable end-product (worn, wielded, consumed, otherwise crafted-into-
// existence) rather than a raw crafting material. Caravan cross-city
// distribution moves raw materials only — finished goods stay where
// they're crafted.
//
// Raw-material item types that may carry IsComponent=true and SHOULD
// flow through cross-city distribution (and so are absent from this
// list): Object (catch-all for ingots, ores, raw mats), Gemstone (raw
// gems), Botanical (herbs).
func isFinishedGood(itemType items.ItemType) bool {
	switch itemType {
	// Wearables / equipment
	case items.Weapon,
		items.Offhand,
		items.Head,
		items.Neck,
		items.Body,
		items.Belt,
		items.Gloves,
		items.Ring,
		items.Wrist,
		items.Back,
		items.Shoulders,
		items.ComponentBag,
		items.Legs,
		items.Feet,
		items.Tail:
		return true
	// Consumables
	case items.Potion,
		items.Food,
		items.Drink,
		items.Scroll,
		items.Grenade,
		items.Junk:
		return true
	// Other end-product types
	case items.Readable,
		items.Key,
		items.Lockpicks,
		items.Service,
		items.Ammo:
		return true
	}
	return false
}
