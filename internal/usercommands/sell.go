package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Sell(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	item, found := user.Character.FindInBackpack(rest)

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

	for _, mobId := range room.GetMobs(rooms.FindMerchant) {

		mob := mobs.GetInstance(mobId)
		if mob == nil {
			continue
		}

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		if item.IsSpecial() {
			mob.Command(`say I'm afraid I don't buy those.`)
			continue
		}

		// Check for ShopInventory-backed merchant first; fall back to legacy.
		shopInv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)

		var sellValue int

		if shopInv != nil {
			cfg := shops.PricingConfigFromBalance()
			offer := shops.EvaluateBuyRules(item, shopInv, mob.CrafterSkill, mob.BuysGeneral, cfg)
			sellValue = offer.Price

			if sellValue > 0 {
				// Apply bartering bonus
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
			continue
		}

		// Merchant must have enough gold to buy the item.
		merchantGold := mob.Character.Gold
		if shopInv != nil {
			merchantGold = shopInv.Gold
		}

		if merchantGold < sellValue {
			mob.Command(`say I can't afford that right now.`)
			continue
		}

		// Transfer gold from merchant to player.
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

		// Update stock.
		if shopInv != nil {
			shopInv.AddStock(item.ItemId, 1)
			if err := shops.SaveShop(mob.Zone, int(mob.MobId), mob.HomeRoomId); err != nil {
				mudlog.Error("SELL", "msg", "SaveShop failed", "error", err)
			}
		} else {
			mob.Character.Shop.StockItem(item.ItemId)
		}

		user.EventLog.Add(`shop`, fmt.Sprintf(`Sold your <ansi fg="itemname">%s</ansi> to <ansi fg="mobname">%s</ansi> for <ansi fg="gold">%d gold</ansi>`, item.DisplayName(), mob.Character.Name, sellValue))

		user.SendText(
			fmt.Sprintf(`You sell a <ansi fg="itemname">%s</ansi> for <ansi fg="gold">%d gold</ansi>.`, item.DisplayName(), sellValue),
		)
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> sells a <ansi fg="itemname">%s</ansi>.`, user.Character.Name, item.DisplayName()),
			user.UserId,
		)

		// Track charisma use on successful sale.
		user.Character.OnStatUse("charisma", user.UserId)
		// Stage 38.5.5: Merchant mob gains charisma from trade interactions.
		mob.Character.OnStatUse("charisma", 0)

		break
	}

	return true, nil

}
