package usercommands

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/pets"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func List(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	listedSomething := false

	for _, mobId := range room.GetMobs(rooms.FindMerchant) {

		mob := mobs.GetInstance(mobId)
		if mob == nil {
			continue
		}

		user.DidTip(`list`, true)

		/// Run restock routine
		mob.Character.Shop.Restock()

		listedSomething = true

		itemsAvailable := characters.Shop{}
		mercsAvailable := characters.Shop{}
		buffsAvailable := characters.Shop{}
		petsAvailable := characters.Shop{}

		for _, saleItem := range mob.Character.Shop.GetInstock() {

			if saleItem.ItemId > 0 {
				itemsAvailable = append(itemsAvailable, saleItem)
				continue
			}

			if saleItem.MobId > 0 {
				mercsAvailable = append(mercsAvailable, saleItem)
				continue
			}

			if saleItem.BuffId > 0 {
				buffsAvailable = append(buffsAvailable, saleItem)
			}

			if saleItem.PetType != `` {
				petsAvailable = append(petsAvailable, saleItem)
			}

		}

		if len(itemsAvailable) == 0 && len(mercsAvailable) == 0 && len(buffsAvailable) == 0 && len(petsAvailable) == 0 {
			mob.Command(`say I have nothing to sell right now, but check again later.`)
			continue
		}

		if len(itemsAvailable) > 0 {

			hasGoldItems := false
			hasTradeItems := false
			for _, stockItm := range itemsAvailable {
				if stockItm.TradeItemId > 0 {
					hasTradeItems = true
				}
				if stockItm.Price >= 0 {
					hasGoldItems = true
				}
			}

			headers := []string{"Qty", "Name", "Type"}
			if hasGoldItems {
				headers = append(headers, "Price")
			}
			if hasTradeItems {
				headers = append(headers, "Trade")
			}

			rows := [][]string{}

			for _, stockItm := range itemsAvailable {
				item := items.New(stockItm.ItemId)

				qtyStr := `N/A`
				if stockItm.QuantityMax != 0 {
					qtyStr = strconv.Itoa(stockItm.Quantity)
				}

				price := stockItm.Price
				if price == 0 {
					price = item.GetSpec().Value
				} else if price < 0 {
					price = 0
				}

				entryRow := []string{
					qtyStr,
					item.DisplayName(),
					string(item.GetSpec().Type),
				}

				if hasGoldItems {
					if price > 0 {
						entryRow = append(entryRow, strconv.Itoa(price))
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				if hasTradeItems {
					if stockItm.TradeItemId > 0 {
						tradeItm := items.New(stockItm.TradeItemId)
						entryRow = append(entryRow, tradeItm.DisplayName())
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				rows = append(rows, entryRow)
			}

			sort.Slice(rows, func(i, j int) bool {
				return rows[i][0] < rows[j][0]
			})

			saleItemsData := templates.GetTable(fmt.Sprintf(`%s by <ansi fg="mobname">%s</ansi>`, colorpatterns.ApplyColorPattern(`Items available`, `cyan`), mob.Character.Name), headers, rows)
			tplTxt, _ := templates.Process("tables/shoplist", saleItemsData, user.UserId, user.UserId)
			user.SendText(tplTxt)
			user.SendText(fmt.Sprintf(`To buy something, type: <ansi fg="command">buy [name]</ansi>%s`, term.CRLFStr))
		}

		if len(mercsAvailable) > 0 {

			hasGoldItems := false
			hasTradeItems := false
			for _, stockItm := range mercsAvailable {
				if stockItm.TradeItemId > 0 {
					hasTradeItems = true
				}
				if stockItm.Price >= 0 {
					hasGoldItems = true
				}
			}

			headers := []string{"Qty", "Name", "Level", "Race"}
			if hasGoldItems {
				headers = append(headers, "Price")
			}
			if hasTradeItems {
				headers = append(headers, "Trade")
			}

			rows := [][]string{}

			for _, stockMerc := range mercsAvailable {

				mobInfo := mobs.GetMobSpec(mobs.MobId(stockMerc.MobId))
				if mobInfo == nil {
					continue
				}
				raceInfo := species.GetSpecies(mobInfo.Character.SpeciesId)
				if raceInfo == nil {
					continue
				}

				qtyStr := `N/A`
				if stockMerc.QuantityMax != 0 {
					qtyStr = strconv.Itoa(stockMerc.Quantity)
				}

				price := stockMerc.Price
				if price == 0 {
					price = 250
				} else if price < 0 {
					price = 0
				}

				entryRow := []string{
					qtyStr,
					`<ansi fg="mobname">` + mobInfo.Character.Name + `</ansi>`,
					`-`,
					raceInfo.Name,
				}

				if hasGoldItems {
					if price > 0 {
						entryRow = append(entryRow, strconv.Itoa(price))
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				if hasTradeItems {
					if stockMerc.TradeItemId > 0 {
						tradeItm := items.New(stockMerc.TradeItemId)
						entryRow = append(entryRow, tradeItm.DisplayName())
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				rows = append(rows, entryRow)

			}

			sort.Slice(rows, func(i, j int) bool {
				num1, _ := strconv.Atoi(rows[i][4])
				num2, _ := strconv.Atoi(rows[j][4])
				return num1 < num2
			})

			saleItemsData := templates.GetTable(fmt.Sprintf(`%s by <ansi fg="mobname">%s</ansi>`, colorpatterns.ApplyColorPattern(`Mercenaries for hire`, `flame`), mob.Character.Name), headers, rows)
			tplTxt, _ := templates.Process("tables/shoplist", saleItemsData, user.UserId, user.UserId)
			user.SendText(tplTxt)
			user.SendText(fmt.Sprintf(`To Hire a merc, type: <ansi fg="command">hire [name]</ansi>%s`, term.CRLFStr))
		}

		if len(buffsAvailable) > 0 {

			hasGoldItems := false
			hasTradeItems := false
			for _, stockItm := range buffsAvailable {
				if stockItm.TradeItemId > 0 {
					hasTradeItems = true
				}
				if stockItm.Price >= 0 {
					hasGoldItems = true
				}
			}

			headers := []string{"Qty", "Enchantment"}
			if hasGoldItems {
				headers = append(headers, "Price")
			}
			if hasTradeItems {
				headers = append(headers, "Trade")
			}

			rows := [][]string{}

			for _, stockBuff := range buffsAvailable {

				buffInfo := buffs.GetBuffSpec(stockBuff.BuffId)
				if buffInfo == nil {
					continue
				}

				qtyStr := `N/A`
				if stockBuff.QuantityMax != 0 {
					qtyStr = strconv.Itoa(stockBuff.Quantity)
				}

				entryRow := []string{
					qtyStr,
					buffInfo.Name,
				}

				if hasGoldItems {
					if stockBuff.Price > 0 {
						entryRow = append(entryRow, strconv.Itoa(stockBuff.Price))
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				if hasTradeItems {
					if stockBuff.TradeItemId > 0 {
						tradeItm := items.New(stockBuff.TradeItemId)
						entryRow = append(entryRow, tradeItm.DisplayName())
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				rows = append(rows, entryRow)

			}

			sort.Slice(rows, func(i, j int) bool {
				num1, _ := strconv.Atoi(rows[i][2])
				num2, _ := strconv.Atoi(rows[j][2])
				return num1 < num2
			})

			saleItemsData := templates.GetTable(fmt.Sprintf(`%s by <ansi fg="mobname">%s</ansi>`, colorpatterns.ApplyColorPattern(`Enchantments`, `rainbow`), mob.Character.Name), headers, rows)
			tplTxt, _ := templates.Process("tables/shoplist", saleItemsData, user.UserId, user.UserId)
			user.SendText(tplTxt)
			user.SendText(fmt.Sprintf(`To buy an enchantment, type: <ansi fg="command">buy [name]</ansi>%s`, term.CRLFStr))
		}

		if len(petsAvailable) > 0 {

			hasGoldItems := false
			hasTradeItems := false
			for _, stockItm := range petsAvailable {
				if stockItm.TradeItemId > 0 {
					hasTradeItems = true
				}
				if stockItm.Price >= 0 {
					hasGoldItems = true
				}
			}

			headers := []string{"Qty", "Pet-Type"}
			if hasGoldItems {
				headers = append(headers, "Price")
			}
			if hasTradeItems {
				headers = append(headers, "Trade")
			}

			rows := [][]string{}

			for _, stockPet := range petsAvailable {

				petInfo := pets.GetPetCopy(stockPet.PetType)
				if !petInfo.Exists() {
					continue
				}

				qtyStr := `N/A`
				if stockPet.QuantityMax != 0 {
					qtyStr = strconv.Itoa(stockPet.Quantity)
				}

				price := stockPet.Price
				if price == 0 {
					price = 10000
				} else if price < 0 {
					price = 0
				}

				entryRow := []string{
					qtyStr,
					petInfo.Type,
				}

				if hasGoldItems {
					if price > 0 {
						entryRow = append(entryRow, strconv.Itoa(price))
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				if hasTradeItems {
					if stockPet.TradeItemId > 0 {
						tradeItm := items.New(stockPet.TradeItemId)
						entryRow = append(entryRow, tradeItm.DisplayName())
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				rows = append(rows, entryRow)
			}

			sort.Slice(rows, func(i, j int) bool {
				num1, _ := strconv.Atoi(rows[i][2])
				num2, _ := strconv.Atoi(rows[j][2])
				return num1 < num2
			})

			saleItemsData := templates.GetTable(fmt.Sprintf(`%s by <ansi fg="mobname">%s</ansi>`, colorpatterns.ApplyColorPattern(`Pets`, `turquoise`), mob.Character.Name), headers, rows)
			tplTxt, _ := templates.Process("tables/shoplist", saleItemsData, user.UserId, user.UserId)
			user.SendText(tplTxt)
			user.SendText(fmt.Sprintf(`To buy a pet, type: <ansi fg="command">buy [name]</ansi>%s`, term.CRLFStr))
		}
	}

	for _, uid := range room.GetPlayers(rooms.FindMerchant) {

		if uid == user.UserId {
			continue
		}

		shopUser := users.GetByUserId(uid)
		if shopUser == nil {
			continue
		}

		listedSomething = true

		itemsAvailable := characters.Shop{}
		mercsAvailable := characters.Shop{}
		buffsAvailable := characters.Shop{}
		petsAvailable := characters.Shop{}

		for _, saleItem := range shopUser.Character.Shop.GetInstock() {

			if saleItem.ItemId > 0 {
				itemsAvailable = append(itemsAvailable, saleItem)
				continue
			}

			if saleItem.MobId > 0 {
				mercsAvailable = append(mercsAvailable, saleItem)
				continue
			}

			if saleItem.BuffId > 0 {
				buffsAvailable = append(buffsAvailable, saleItem)
			}

			if saleItem.PetType != `` {
				petsAvailable = append(petsAvailable, saleItem)
			}
		}

		if len(itemsAvailable) == 0 && len(mercsAvailable) == 0 && len(buffsAvailable) == 0 && len(petsAvailable) == 0 {
			continue
		}

		if len(itemsAvailable) > 0 {

			hasGoldItems := false
			hasTradeItems := false
			for _, stockItm := range itemsAvailable {
				if stockItm.TradeItemId > 0 {
					hasTradeItems = true
				}
				if stockItm.Price >= 0 { // 0 means use specified item value
					hasGoldItems = true
				}
			}

			headers := []string{"Qty", "Name", "Type"}

			if hasGoldItems {
				headers = append(headers, `Price`)
			}
			if hasTradeItems {
				headers = append(headers, `Trade`)
			}

			rows := [][]string{}

			for _, stockItm := range itemsAvailable {
				item := items.New(stockItm.ItemId)

				qtyStr := `N/A`
				if stockItm.QuantityMax != 0 {
					qtyStr = strconv.Itoa(stockItm.Quantity)
				}

				price := stockItm.Price
				if price == 0 {
					price = item.GetSpec().Value
				} else if price < 0 {
					price = 0
				}

				entryRow := []string{
					qtyStr,
					item.DisplayName(),
					string(item.GetSpec().Type),
				}

				if hasGoldItems {
					if price > 0 {
						entryRow = append(entryRow, strconv.Itoa(price))
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				if hasTradeItems {
					if stockItm.TradeItemId > 0 {
						tradeItm := items.New(stockItm.TradeItemId)
						entryRow = append(entryRow, tradeItm.DisplayName())
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				rows = append(rows, entryRow)

			}

			sort.Slice(rows, func(i, j int) bool {
				num1, _ := strconv.Atoi(rows[i][3])
				num2, _ := strconv.Atoi(rows[j][3])
				return num1 < num2
			})

			saleItemsData := templates.GetTable(fmt.Sprintf(`%s by <ansi fg="username">%s</ansi>`, colorpatterns.ApplyColorPattern(`Items available`, `cyan`), shopUser.Character.Name), headers, rows)
			tplTxt, _ := templates.Process("tables/shoplist", saleItemsData, user.UserId, user.UserId)
			user.SendText(tplTxt)
			user.SendText(fmt.Sprintf(`To buy something, type: <ansi fg="command">buy [name]</ansi>%s`, term.CRLFStr))
		}

		if len(mercsAvailable) > 0 {

			hasGoldItems := false
			hasTradeItems := false
			for _, stockMerc := range itemsAvailable {
				if stockMerc.TradeItemId > 0 {
					hasTradeItems = true
				}
				if stockMerc.Price >= 0 { // 0 means auto-calculate a value
					hasGoldItems = true
				}
			}

			headers := []string{"Qty", "Name", "Level", "Race"}

			if hasGoldItems {
				headers = append(headers, `Price`)
			}
			if hasTradeItems {
				headers = append(headers, `Trade`)
			}

			rows := [][]string{}

			for _, stockMerc := range mercsAvailable {

				mobInfo := mobs.GetMobSpec(mobs.MobId(stockMerc.MobId))
				if mobInfo == nil {
					continue
				}
				raceInfo := species.GetSpecies(mobInfo.Character.SpeciesId)
				if raceInfo == nil {
					continue
				}

				qtyStr := `N/A`
				if stockMerc.QuantityMax != 0 {
					qtyStr = strconv.Itoa(stockMerc.Quantity)
				}

				price := stockMerc.Price
				if price == 0 {
					price = 250
				} else if price < 0 {
					price = 0
				}

				entryRow := []string{
					qtyStr,
					`<ansi fg="mobname">` + mobInfo.Character.Name + `</ansi>`,
					`-`,
					raceInfo.Name,
				}

				if hasGoldItems {
					if price > 0 {
						entryRow = append(entryRow, strconv.Itoa(price))
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				if hasTradeItems {
					if stockMerc.TradeItemId > 0 {
						tradeItm := items.New(stockMerc.TradeItemId)
						entryRow = append(entryRow, tradeItm.DisplayName())
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				rows = append(rows, entryRow)

			}

			sort.Slice(rows, func(i, j int) bool {
				num1, _ := strconv.Atoi(rows[i][4])
				num2, _ := strconv.Atoi(rows[j][4])
				return num1 < num2
			})

			saleItemsData := templates.GetTable(fmt.Sprintf(`%s by <ansi fg="username">%s</ansi>`, colorpatterns.ApplyColorPattern(`Mercenaries for hire`, `flame`), shopUser.Character.Name), headers, rows)
			tplTxt, _ := templates.Process("tables/shoplist", saleItemsData, user.UserId, user.UserId)
			user.SendText(tplTxt)
			user.SendText(fmt.Sprintf(`To Hire a merc, type: <ansi fg="command">hire [name]</ansi>%s`, term.CRLFStr))
		}

		if len(buffsAvailable) > 0 {

			hasGoldItems := false
			hasTradeItems := false
			for _, stockBuff := range itemsAvailable {
				if stockBuff.TradeItemId > 0 {
					hasTradeItems = true
				}
				if stockBuff.Price > 0 {
					hasGoldItems = true
				}
			}

			headers := []string{"Qty", "Enchantment"}

			if hasGoldItems {
				headers = append(headers, `Price`)
			}
			if hasTradeItems {
				headers = append(headers, `Trade`)
			}

			rows := [][]string{}

			for _, stockBuff := range buffsAvailable {

				buffInfo := buffs.GetBuffSpec(stockBuff.BuffId)
				if buffInfo == nil {
					continue
				}

				qtyStr := `N/A`
				if stockBuff.QuantityMax != 0 {
					qtyStr = strconv.Itoa(stockBuff.Quantity)
				}

				entryRow := []string{
					qtyStr,
					buffInfo.Name,
				}

				if hasGoldItems {
					if stockBuff.Price > 0 {
						entryRow = append(entryRow, strconv.Itoa(stockBuff.Price))
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				if hasTradeItems {
					if stockBuff.TradeItemId > 0 {
						tradeItm := items.New(stockBuff.TradeItemId)
						entryRow = append(entryRow, tradeItm.DisplayName())
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				rows = append(rows, entryRow)
			}

			sort.Slice(rows, func(i, j int) bool {
				num1, _ := strconv.Atoi(rows[i][2])
				num2, _ := strconv.Atoi(rows[j][2])
				return num1 < num2
			})

			saleItemsData := templates.GetTable(fmt.Sprintf(`%s by <ansi fg="username">%s</ansi>`, colorpatterns.ApplyColorPattern(`Enchantments`, `rainbow`), shopUser.Character.Name), headers, rows)
			tplTxt, _ := templates.Process("tables/shoplist", saleItemsData, user.UserId, user.UserId)
			user.SendText(tplTxt)
			user.SendText(fmt.Sprintf(`To buy an enchantment, type: <ansi fg="command">buy [name]</ansi>%s`, term.CRLFStr))
		}

		if len(petsAvailable) > 0 {

			hasGoldItems := false
			hasTradeItems := false
			for _, stockPet := range itemsAvailable {
				if stockPet.TradeItemId > 0 {
					hasTradeItems = true
				}
				if stockPet.Price >= 0 { // zero means assign a flat price
					hasGoldItems = true
				}
			}

			headers := []string{"Qty", "Pet-Type"}

			if hasGoldItems {
				headers = append(headers, `Price`)
			}
			if hasTradeItems {
				headers = append(headers, `Trade`)
			}

			rows := [][]string{}

			for _, stockPet := range petsAvailable {

				petInfo := pets.GetPetCopy(stockPet.PetType)
				if !petInfo.Exists() {
					continue
				}

				qtyStr := `N/A`
				if stockPet.QuantityMax != 0 {
					qtyStr = strconv.Itoa(stockPet.Quantity)
				}

				price := stockPet.Price
				if price == 0 {
					price = 10000
				} else if price < 0 {
					price = 0
				}

				entryRow := []string{
					qtyStr,
					petInfo.Type,
				}

				if hasGoldItems {
					if price > 0 {
						entryRow = append(entryRow, strconv.Itoa(price))
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				if hasTradeItems {
					if stockPet.TradeItemId > 0 {
						tradeItm := items.New(stockPet.TradeItemId)
						entryRow = append(entryRow, tradeItm.DisplayName())
					} else {
						entryRow = append(entryRow, ``)
					}
				}

				rows = append(rows, entryRow)
			}

			sort.Slice(rows, func(i, j int) bool {
				num1, _ := strconv.Atoi(rows[i][2])
				num2, _ := strconv.Atoi(rows[j][2])
				return num1 < num2
			})

			saleItemsData := templates.GetTable(fmt.Sprintf(`%s by <ansi fg="username">%s</ansi>`, colorpatterns.ApplyColorPattern(`Pets`, `turquoise`), user.Character.Name), headers, rows)
			tplTxt, _ := templates.Process("tables/shoplist", saleItemsData, user.UserId, user.UserId)
			user.SendText(tplTxt)
			user.SendText(fmt.Sprintf(`To buy a pet, type: <ansi fg="command">buy [name]</ansi>%s`, term.CRLFStr))
		}

	}

	if !listedSomething {
		user.SendText("Visit a merchant to list and buy objects.")
	}

	return true, nil
}
