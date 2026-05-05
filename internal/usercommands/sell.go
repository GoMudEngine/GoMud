package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// sellFindItem searches backpack → potions → components for a matching item.
func sellFindItem(user *users.UserRecord, name string) (items.Item, bool) {
	item, found := user.Character.FindInBackpack(name)
	if !found {
		item, found = user.Character.FindInPotions(name)
	}
	if !found {
		item, found = user.Character.FindInComponents(name)
	}
	return item, found
}

// sellResult is returned by trySellOne.
type sellResult int

const (
	sellOK           sellResult = iota // item sold successfully
	sellNoItem                         // player has no more matching items
	sellMerchantBroke                  // merchant ran out of gold
	sellRejected                       // merchant declined this item type
)

// trySellOne attempts to sell a single matching item to mob.
// It mirrors the body of the original Sell loop.
func trySellOne(itemName string, user *users.UserRecord, room *rooms.Room,
	mob *mobs.Mob, shopInv *shops.ShopInventory) (soldValue int, res sellResult) {

	item, found := sellFindItem(user, itemName)
	if !found {
		return 0, sellNoItem
	}

	itemSpec := item.GetSpec()
	if itemSpec.ItemId < 1 {
		return 0, sellRejected
	}
	if itemSpec.QuestToken != `` {
		user.SendText("Quest items cannot be sold!")
		return 0, sellRejected
	}

	user.Character.CancelBuffsWithFlag(buffs.Hidden)

	if item.IsSpecial() {
		mob.Command(`say I'm afraid I don't buy those.`)
		return 0, sellRejected
	}

	var sellValue int
	var buyReason string

	if shopInv != nil {
		cfg := shops.PricingConfigFromBalance()
		wornItems := mob.Character.Equipment.GetAllItemsWithEmptySlots()
		offer := shops.EvaluateBuyRules(item, shopInv, mob.CrafterSkill, mob.BuysGeneral, cfg, wornItems)
		sellValue = offer.Price
		buyReason = offer.Reason

		if sellValue > 0 {
			barterSkill := user.Character.GetSkillLevel(skills.Bartering)
			if barterSkill > 0 {
				bonus := float64(barterSkill) / 50.0 * 0.15
				if bonus > 0.15 {
					bonus = 0.15
				}
				sellValue = shops.ApplyBarterBuyBonus(sellValue, bonus)
			}
		}
	} else {
		sellValue = mob.GetSellPrice(item)
	}

	if sellValue <= 0 {
		mob.Command(`say I'm not interested in that.`)
		return 0, sellRejected
	}

	merchantGold := mob.Character.Gold
	if shopInv != nil {
		merchantGold = shopInv.Gold
	}
	if merchantGold < sellValue {
		mob.Command(`say I can't afford that right now.`)
		return 0, sellMerchantBroke
	}

	// Transfer gold.
	if shopInv != nil {
		shopInv.Gold -= sellValue
	} else {
		mob.Character.Gold -= sellValue
	}
	user.Character.Gold += sellValue
	user.Character.RemoveItem(item)

	events.AddToQueue(events.ItemOwnership{
		UserId: user.UserId,
		Item:   item,
		Gained: false,
	})
	events.AddToQueue(events.EquipmentChange{
		UserId:     user.UserId,
		GoldChange: sellValue,
	})

	// Update stock or equip.
	if shopInv != nil {
		if buyReason == "gear_upgrade" {
			newItem := items.New(item.ItemId)
			if newItem.ItemId > 0 {
				returnedItems, wore, _ := mob.Character.Wear(newItem)
				if wore {
					for _, old := range returnedItems {
						if old.ItemId > 0 {
							shopInv.AddStock(old.ItemId, 1)
						}
					}
					room.SendTextVisual(
						fmt.Sprintf(`<ansi fg="mobname">%s</ansi> examines the <ansi fg="itemname">%s</ansi> and puts it on.`, mob.Character.Name, newItem.DisplayName()),
						user.UserId,
					)
				} else {
					shopInv.AddStock(item.ItemId, 1)
				}
			} else {
				shopInv.AddStock(item.ItemId, 1)
			}
		} else {
			shopInv.AddStock(item.ItemId, 1)
		}
		if err := shops.SaveShop(mob.Zone, int(mob.MobId), mob.HomeRoomId); err != nil {
			mudlog.Error("SELL", "msg", "SaveShop failed", "error", err)
		}
	} else {
		mob.Character.Shop.StockItem(item.ItemId)
	}

	// Progression: same per-sale cost as a single sell.
	user.Character.OnSkillUse(string(skills.Bartering), user.UserId)
	mob.Character.OnStatUse("charisma", 0)

	return sellValue, sellOK
}

func Sell(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		user.SendText("What would you like to sell?")
		return true, nil
	}

	// ── Parse optional leading quantity ───────────────────────────────────
	// Supported forms:
	//   sell 5 iron-ore        → quantity=5, itemName="iron-ore"
	//   sell all iron-ore      → quantity=unlimited (math.MaxInt), name="iron-ore"
	//   sell all.iron-ore      → diku-style alias for the above
	//   sell iron-ore          → quantity=1 (existing single-sell)
	//
	// "sell all" with no item name is rejected.
	const unlimited = int(^uint(0) >> 1) // math.MaxInt without import

	quantity := 1
	itemName := rest

	lower := strings.ToLower(strings.TrimSpace(rest))

	// Detect "all.name" prefix (diku-style).
	if strings.HasPrefix(lower, "all.") {
		suffix := strings.TrimPrefix(lower, "all.")
		if strings.TrimSpace(suffix) == "" {
			user.SendText("Specify what you want to sell. (e.g. <ansi fg=\"command\">sell all iron-ore</ansi>)")
			return true, nil
		}
		quantity = unlimited
		// Preserve original casing: skip past "all." (4 chars).
		itemName = rest[4:]
	} else {
		// Check for leading word: a number or the word "all".
		args := strings.SplitN(strings.TrimSpace(rest), " ", 2)
		if len(args) == 2 {
			firstWord := strings.ToLower(args[0])
			if firstWord == "all" {
				if strings.TrimSpace(args[1]) == "" {
					user.SendText("Specify what you want to sell. (e.g. <ansi fg=\"command\">sell all iron-ore</ansi>)")
					return true, nil
				}
				quantity = unlimited
				itemName = args[1]
			} else if n, err := strconv.Atoi(args[0]); err == nil && n >= 1 {
				quantity = n
				itemName = args[1]
			}
		} else if strings.ToLower(strings.TrimSpace(rest)) == "all" {
			// bare "sell all" with no item name
			user.SendText("Specify what you want to sell. (e.g. <ansi fg=\"command\">sell all iron-ore</ansi>)")
			return true, nil
		}
	}

	// ── Validate item exists and is sellable before resolving merchant ─────
	item, found := sellFindItem(user, itemName)
	if !found {
		user.SendText("You don't have that item.")
		return true, nil
	}
	itemSpec := item.GetSpec()
	if itemSpec.ItemId < 1 {
		return true, nil
	}
	if itemSpec.QuestToken != `` {
		user.SendText("Quest items cannot be sold!")
		return true, nil
	}

	// ── Resolve merchant once ─────────────────────────────────────────────
	merchantMobs := room.GetMobs(rooms.FindMerchant)
	if len(merchantMobs) == 0 {
		user.SendText("There's no merchant here.")
		return true, nil
	}

	// Find the first merchant willing to buy this item.
	var selectedMob *mobs.Mob
	var selectedShopInv *shops.ShopInventory

	for _, mobId := range merchantMobs {
		mob := mobs.GetInstance(mobId)
		if mob == nil {
			continue
		}
		shopInv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)

		// Quick probe: will this merchant buy the item at all?
		var probeValue int
		if shopInv != nil {
			cfg := shops.PricingConfigFromBalance()
			wornItems := mob.Character.Equipment.GetAllItemsWithEmptySlots()
			offer := shops.EvaluateBuyRules(item, shopInv, mob.CrafterSkill, mob.BuysGeneral, cfg, wornItems)
			probeValue = offer.Price
		} else {
			probeValue = mob.GetSellPrice(item)
		}

		if probeValue > 0 {
			selectedMob = mob
			selectedShopInv = shopInv
			break
		}
	}

	if selectedMob == nil {
		// No merchant wants it — let the first mob say so.
		firstMob := mobs.GetInstance(merchantMobs[0])
		if firstMob != nil {
			firstMob.Command(`say I'm not interested in that.`)
		}
		return true, nil
	}

	// ── Sell loop ─────────────────────────────────────────────────────────
	sold := 0
	totalGold := 0
	var lastDisplayName string
	var stopReason sellResult

	for sold < quantity {
		value, res := trySellOne(itemName, user, room, selectedMob, selectedShopInv)
		if res != sellOK {
			stopReason = res
			break
		}
		sold++
		totalGold += value
		// Capture display name from the item we just found (before removal).
		// We re-fetch here only when we have it fresh — after trySellOne
		// the item is already removed, so we grab it once on first success.
		if sold == 1 {
			// Get a fresh match just for display name — item was sold, so use spec.
			spec := items.GetItemSpec(item.ItemId)
			if spec != nil {
				lastDisplayName = spec.Name
			} else {
				lastDisplayName = item.DisplayName()
			}
		}
	}

	if sold == 0 {
		// trySellOne already sent the appropriate merchant message.
		return true, nil
	}

	// ── Confirmation messages ─────────────────────────────────────────────
	if sold == 1 {
		// Single sell: use original message format with DisplayName.
		// Re-derive display name from spec since item is gone.
		spec := items.GetItemSpec(item.ItemId)
		displayName := item.DisplayName()
		if spec != nil {
			// Build a fresh display item for its DisplayName() method.
			freshItm := items.New(item.ItemId)
			if freshItm.ItemId != 0 {
				displayName = freshItm.DisplayName()
			}
		}
		user.EventLog.Add(`shop`, fmt.Sprintf(`Sold your <ansi fg="itemname">%s</ansi> to <ansi fg="mobname">%s</ansi> for <ansi fg="gold">%d gold</ansi>`, displayName, selectedMob.Character.Name, totalGold))
		user.SendText(
			fmt.Sprintf(`You sell a <ansi fg="itemname">%s</ansi> for <ansi fg="gold">%d gold</ansi>.`, displayName, totalGold),
		)
		room.SendTextVisual(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> sells a <ansi fg="itemname">%s</ansi>.`, user.Character.Name, displayName),
			user.UserId,
		)
	} else {
		// Multi-sell: "You sell N <name>s for <total> gold."
		// Use the item name with a simple plural (append "s"). If we ever add
		// a plural field to ItemSpec, hook it here.
		pluralName := lastDisplayName + "s"

		user.EventLog.Add(`shop`, fmt.Sprintf(
			`Sold %d <ansi fg="itemname">%s</ansi> to <ansi fg="mobname">%s</ansi> for <ansi fg="gold">%d gold</ansi>`,
			sold, pluralName, selectedMob.Character.Name, totalGold,
		))
		user.SendText(fmt.Sprintf(
			`You sell %d <ansi fg="itemname">%s</ansi> for <ansi fg="gold">%d gold</ansi>.`,
			sold, pluralName, totalGold,
		))
		room.SendTextVisual(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> sells %d <ansi fg="itemname">%s</ansi>.`,
				user.Character.Name, sold, pluralName),
			user.UserId,
		)
	}

	// Early-stop warning.
	if quantity > 1 && sold < quantity && stopReason != sellNoItem || (quantity == unlimited && stopReason != sellNoItem) {
		var reason string
		switch stopReason {
		case sellMerchantBroke:
			reason = fmt.Sprintf(`<ansi fg="mobname">%s</ansi> ran out of gold`, selectedMob.Character.Name)
		case sellRejected:
			reason = fmt.Sprintf(`<ansi fg="mobname">%s</ansi> lost interest`, selectedMob.Character.Name)
		default:
			reason = fmt.Sprintf("running out of %s", lastDisplayName)
		}
		user.SendText(fmt.Sprintf(
			`<ansi fg="yellow">Sold %d before %s.</ansi>`,
			sold, reason,
		))
	}

	return true, nil
}
