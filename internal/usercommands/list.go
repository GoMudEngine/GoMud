package usercommands

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/pets"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func List(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	listedSomething := false

	// Shut for the night — see buy.go.
	if asleep := actions.ShopClosedForSleep(room); asleep != nil {
		actions.RefuseMobIfAsleep(asleep, user)
		return true, nil
	}

	for _, mobId := range room.GetMobs(rooms.FindMerchant) {

		mob := mobs.GetInstance(mobId)
		if mob == nil {
			continue
		}
		// A sleeping keeper alongside an awake one: skip the sleeper's stock
		// rather than advertising goods nobody is there to sell.
		if actions.TargetAsleep(&mob.Character) {
			continue
		}

		user.DidTip(`list`, true)

		listedSomething = true

		// Use ShopInventory for dynamic pricing if available; legacy otherwise.
		shopInv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)
		if shopInv != nil {
			stock := buildShopStockFromInventory(shopInv, user)
			// Also include non-item entries (buffs, mercs, pets) from legacy shop
			for _, si := range mob.Character.Shop {
				if si.ItemId == 0 { // buff, merc, or pet entry
					stock = append(stock, si)
				}
			}
			if !renderMobMerchantListing(user, stock, mob.Character.Name) {
				mob.Command(`say I have nothing to sell right now, but check again later.`)
			}
		} else {
			mob.Character.Shop.Restock()
			if !renderMobMerchantListing(user, mob.Character.Shop.GetInstock(), mob.Character.Name) {
				mob.Command(`say I have nothing to sell right now, but check again later.`)
			}
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

		renderPlayerMerchantListing(user, shopUser.Character.Shop.GetInstock(), shopUser.Character.Name, user.Character.Name)
	}

	if !listedSomething {
		// The keeper of THIS shop has gone to the back room / upstairs to
		// sleep. "Visit a merchant" would read as though the player came to
		// the wrong place, when they simply came at the wrong hour.
		if away := actions.MerchantAwayAsleep(room); away != nil {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> has closed up for the night and gone to bed. `+
					`Come back in the morning -- or make some noise nearby.`, away.Character.Name))
			return true, nil
		}
		user.SendText(messaging.CategorySystem, "Visit a merchant to list and buy objects.")
		return true, nil
	}

	// Quest engine: viewing a merchant's wares counts as a "list" command, so
	// quests can credit browsing a shop and not only purchasing (mirrors the
	// "buy" notification in actions.Buy). Gated on having actually listed a
	// merchant so a no-op `list` in an empty room doesn't fire.
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "list",
	}, bridge, bridge)

	return true, nil
}

// buildShopStockFromInventory converts a ShopInventory into a legacy characters.Shop
// slice for display in the existing table renderer. Prices are dynamically calculated.
func buildShopStockFromInventory(shopInv *shops.ShopInventory, user *users.UserRecord) characters.Shop {
	cfg := shops.PricingConfigFromBalance()
	var stock characters.Shop

	for _, entry := range shopInv.Stock {
		if entry.Current <= 0 {
			continue
		}
		itm := items.New(entry.ItemId)
		if itm.ItemId == 0 {
			continue
		}
		spec := itm.GetSpec()
		restock := shops.PricingBaseline(&entry, cfg)
		price := shops.CalcSellPrice(spec.Value, entry.Current, restock, cfg)

		stock = append(stock, characters.ShopItem{
			ItemId:      entry.ItemId,
			Quantity:    entry.Current,
			QuantityMax: entry.MaxStock, // Non-zero so the display shows the actual count, not "N/A"
			Price:       price,
		})
	}
	return stock
}

// partitionShopStock splits shop stock into four categories: items, mercs, buffs, pets.
func partitionShopStock(stock characters.Shop) (itemStock, mercStock, buffStock, petStock characters.Shop) {
	for _, saleItem := range stock {

		if saleItem.ItemId > 0 {
			itemStock = append(itemStock, saleItem)
			continue
		}

		if saleItem.MobId > 0 {
			mercStock = append(mercStock, saleItem)
			continue
		}

		if saleItem.BuffId > 0 {
			buffStock = append(buffStock, saleItem)
		}

		if saleItem.PetType != `` {
			petStock = append(petStock, saleItem)
		}

	}
	return
}

// checkGoldTrade scans a stock list to determine whether gold prices and/or trade items exist.
// goldGte0 controls whether Price >= 0 (true) or Price > 0 (false) counts as having gold.
func checkGoldTrade(stock characters.Shop, goldGte0 bool) (hasGold, hasTrade bool) {
	for _, stockItm := range stock {
		if stockItm.TradeItemId > 0 {
			hasTrade = true
		}
		if goldGte0 && stockItm.Price >= 0 {
			hasGold = true
		}
		if !goldGte0 && stockItm.Price > 0 {
			hasGold = true
		}
	}
	return
}

// buildItemRows constructs table headers and rows for an items shop section.
func buildItemRows(stock characters.Shop, hasGold, hasTrade bool) ([]string, [][]string) {
	headers := []string{"Qty", "Name", "Type"}
	if hasGold {
		headers = append(headers, "Price")
	}
	if hasTrade {
		headers = append(headers, "Trade")
	}

	rows := [][]string{}

	for _, stockItm := range stock {
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

		if hasGold {
			if price > 0 {
				entryRow = append(entryRow, strconv.Itoa(price))
			} else {
				entryRow = append(entryRow, ``)
			}
		}

		if hasTrade {
			if stockItm.TradeItemId > 0 {
				tradeItm := items.New(stockItm.TradeItemId)
				entryRow = append(entryRow, tradeItm.DisplayName())
			} else {
				entryRow = append(entryRow, ``)
			}
		}

		rows = append(rows, entryRow)
	}
	return headers, rows
}

// buildMercRows constructs table headers and rows for a mercenaries shop section.
func buildMercRows(stock characters.Shop, hasGold, hasTrade bool) ([]string, [][]string) {
	headers := []string{"Qty", "Name", "Race"}
	if hasGold {
		headers = append(headers, "Price")
	}
	if hasTrade {
		headers = append(headers, "Trade")
	}

	rows := [][]string{}

	for _, stockMerc := range stock {

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
			raceInfo.Name,
		}

		if hasGold {
			if price > 0 {
				entryRow = append(entryRow, strconv.Itoa(price))
			} else {
				entryRow = append(entryRow, ``)
			}
		}

		if hasTrade {
			if stockMerc.TradeItemId > 0 {
				tradeItm := items.New(stockMerc.TradeItemId)
				entryRow = append(entryRow, tradeItm.DisplayName())
			} else {
				entryRow = append(entryRow, ``)
			}
		}

		rows = append(rows, entryRow)

	}
	return headers, rows
}

// buildBuffRows constructs table headers and rows for a buffs/enchantments shop section.
func buildBuffRows(stock characters.Shop, hasGold, hasTrade bool) ([]string, [][]string) {
	headers := []string{"Qty", "Enchantment"}
	if hasGold {
		headers = append(headers, "Price")
	}
	if hasTrade {
		headers = append(headers, "Trade")
	}

	rows := [][]string{}

	for _, stockBuff := range stock {

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

		if hasGold {
			if stockBuff.Price > 0 {
				entryRow = append(entryRow, strconv.Itoa(stockBuff.Price))
			} else {
				entryRow = append(entryRow, ``)
			}
		}

		if hasTrade {
			if stockBuff.TradeItemId > 0 {
				tradeItm := items.New(stockBuff.TradeItemId)
				entryRow = append(entryRow, tradeItm.DisplayName())
			} else {
				entryRow = append(entryRow, ``)
			}
		}

		rows = append(rows, entryRow)

	}
	return headers, rows
}

// buildPetRows constructs table headers and rows for a pets shop section.
func buildPetRows(stock characters.Shop, hasGold, hasTrade bool) ([]string, [][]string) {
	headers := []string{"Qty", "Pet-Type"}
	if hasGold {
		headers = append(headers, "Price")
	}
	if hasTrade {
		headers = append(headers, "Trade")
	}

	rows := [][]string{}

	for _, stockPet := range stock {

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

		if hasGold {
			if price > 0 {
				entryRow = append(entryRow, strconv.Itoa(price))
			} else {
				entryRow = append(entryRow, ``)
			}
		}

		if hasTrade {
			if stockPet.TradeItemId > 0 {
				tradeItm := items.New(stockPet.TradeItemId)
				entryRow = append(entryRow, tradeItm.DisplayName())
			} else {
				entryRow = append(entryRow, ``)
			}
		}

		rows = append(rows, entryRow)
	}
	return headers, rows
}

// sortRowsByCol sorts rows by numeric value at the given column index.
func sortRowsByCol(rows [][]string, col int) {
	sort.Slice(rows, func(i, j int) bool {
		num1, _ := strconv.Atoi(rows[i][col])
		num2, _ := strconv.Atoi(rows[j][col])
		return num1 < num2
	})
}

// renderShopTable renders a sorted shop section table and sends it to the user.
func renderShopTable(user *users.UserRecord, title, colorPattern, sellerName, sellerTag string, headers []string, rows [][]string, helpText string) {
	saleItemsData := templates.GetTable(fmt.Sprintf(`%s by <ansi fg="%s">%s</ansi>`, colorpatterns.ApplyColorPattern(title, colorPattern), sellerTag, sellerName), headers, rows)
	tplTxt, _ := templates.Process("tables/shoplist", saleItemsData, user.UserId, user.UserId)
	user.SendText(messaging.CategorySystem, tplTxt)
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`%s%s`, helpText, term.CRLFStr))
}

// renderMobMerchantListing renders all shop sections for a mob merchant.
// Returns false if the merchant has nothing in stock.
func renderMobMerchantListing(user *users.UserRecord, stock characters.Shop, sellerName string) bool {
	itemStock, mercStock, buffStock, petStock := partitionShopStock(stock)

	if len(itemStock) == 0 && len(mercStock) == 0 && len(buffStock) == 0 && len(petStock) == 0 {
		return false
	}

	sellerTag := `mobname`

	if len(itemStock) > 0 {
		hasGold, hasTrade := checkGoldTrade(itemStock, true)
		headers, rows := buildItemRows(itemStock, hasGold, hasTrade)
		sort.Slice(rows, func(i, j int) bool {
			return rows[i][0] < rows[j][0]
		})
		renderShopTable(user, `Items available`, `cyan`, sellerName, sellerTag, headers, rows,
			`To buy something, type: <ansi fg="command">buy [name]</ansi>`)
	}

	if len(mercStock) > 0 {
		hasGold, hasTrade := checkGoldTrade(mercStock, true)
		headers, rows := buildMercRows(mercStock, hasGold, hasTrade)
		sortRowsByCol(rows, 4)
		renderShopTable(user, `Mercenaries for hire`, `flame`, sellerName, sellerTag, headers, rows,
			`To Hire a merc, type: <ansi fg="command">hire [name]</ansi>`)
	}

	if len(buffStock) > 0 {
		hasGold, hasTrade := checkGoldTrade(buffStock, true)
		headers, rows := buildBuffRows(buffStock, hasGold, hasTrade)
		sortRowsByCol(rows, 2)
		renderShopTable(user, `Enchantments`, `rainbow`, sellerName, sellerTag, headers, rows,
			`To buy an enchantment, type: <ansi fg="command">buy [name]</ansi>`)
	}

	if len(petStock) > 0 {
		hasGold, hasTrade := checkGoldTrade(petStock, true)
		headers, rows := buildPetRows(petStock, hasGold, hasTrade)
		sortRowsByCol(rows, 2)
		renderShopTable(user, `Pets`, `turquoise`, sellerName, sellerTag, headers, rows,
			`To buy a pet, type: <ansi fg="command">buy [name]</ansi>`)
	}

	return true
}

// renderPlayerMerchantListing renders all shop sections for a player merchant.
// browsingUserName is passed for the pet section title (pre-existing behavior preserved).
func renderPlayerMerchantListing(user *users.UserRecord, stock characters.Shop, sellerName, browsingUserName string) {
	itemStock, mercStock, buffStock, petStock := partitionShopStock(stock)

	if len(itemStock) == 0 && len(mercStock) == 0 && len(buffStock) == 0 && len(petStock) == 0 {
		return
	}

	sellerTag := `username`

	if len(itemStock) > 0 {
		hasGold, hasTrade := checkGoldTrade(itemStock, true)
		headers, rows := buildItemRows(itemStock, hasGold, hasTrade)
		sortRowsByCol(rows, 3)
		renderShopTable(user, `Items available`, `cyan`, sellerName, sellerTag, headers, rows,
			`To buy something, type: <ansi fg="command">buy [name]</ansi>`)
	}

	if len(mercStock) > 0 {
		// Pre-existing behavior preserved: checks itemStock instead of mercStock for hasGold/hasTrade
		hasGold, hasTrade := checkGoldTrade(itemStock, true)
		headers, rows := buildMercRows(mercStock, hasGold, hasTrade)
		sortRowsByCol(rows, 4)
		renderShopTable(user, `Mercenaries for hire`, `flame`, sellerName, sellerTag, headers, rows,
			`To Hire a merc, type: <ansi fg="command">hire [name]</ansi>`)
	}

	if len(buffStock) > 0 {
		// Pre-existing behavior preserved: checks itemStock instead of buffStock, uses Price > 0 instead of >= 0
		hasGold, hasTrade := checkGoldTrade(itemStock, false)
		headers, rows := buildBuffRows(buffStock, hasGold, hasTrade)
		sortRowsByCol(rows, 2)
		renderShopTable(user, `Enchantments`, `rainbow`, sellerName, sellerTag, headers, rows,
			`To buy an enchantment, type: <ansi fg="command">buy [name]</ansi>`)
	}

	if len(petStock) > 0 {
		// Pre-existing behavior preserved: checks itemStock instead of petStock, uses browsingUserName
		hasGold, hasTrade := checkGoldTrade(itemStock, true)
		headers, rows := buildPetRows(petStock, hasGold, hasTrade)
		sortRowsByCol(rows, 2)
		renderShopTable(user, `Pets`, `turquoise`, browsingUserName, sellerTag, headers, rows,
			`To buy a pet, type: <ansi fg="command">buy [name]</ansi>`)
	}
}
