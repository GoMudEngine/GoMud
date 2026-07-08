package actions

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// BuyOptions controls how a purchase is attempted.
type BuyOptions struct {
	// Request is the raw "rest" string, e.g. "5 iron ingot from marko".
	// Quantity prefix and "from <merchant>" suffix are parsed inside Buy,
	// so wrappers pass through whatever the player typed.
	Request string

	// TargetMerchantUserId, when > 0, restricts merchant selection to a
	// specific user merchant. Wrappers may set this directly for cases
	// where the caller has already resolved the target.
	TargetMerchantUserId int

	// TargetMerchantMobInstanceId, when > 0, restricts merchant selection
	// to a specific mob merchant.
	TargetMerchantMobInstanceId int
}

// BuyResult is the outcome of an attempted purchase.
type BuyResult struct {
	Success   bool   // at least one unit purchased
	Purchased int    // actual units purchased (may be < Requested)
	Requested int    // requested quantity (1 if unspecified)
	SaleType  string // "item" | "buff" | "" on failure
	Reason    string // populated on failure
}

// Failure-reason vocabulary returned in BuyResult.Reason.
const (
	BuyReasonNoRequest        = "no_request"
	BuyReasonNoMerchant       = "no_merchant"
	BuyReasonNoMatch          = "no_match"
	BuyReasonOutOfStock       = "out_of_stock"
	BuyReasonInsufficientGold = "insufficient_gold"
	BuyReasonMissingTradeItem = "missing_trade_item"
	BuyReasonOverburdened     = "overburdened"
	BuyReasonSelfTarget       = "self_target"
)

// legacyShopCatalog enumerates the in-stock items + buffs offered by a
// legacy Character.Shop. Merc and pet sale types are intentionally NOT
// surfaced (spec 2.1 drops them).
type legacyShopCatalog struct {
	nameToShopItem map[string]characters.ShopItem
	itemNames      []string
	itemNamesFancy []string
	itemPrices     map[int]int
	buffNames      []string
	buffPrices     map[int]int
}

func buildLegacyCatalog(saleItems characters.Shop) legacyShopCatalog {
	cat := legacyShopCatalog{
		nameToShopItem: map[string]characters.ShopItem{},
		itemPrices:     map[int]int{},
		buffPrices:     map[int]int{},
	}

	for _, saleItem := range saleItems {
		if saleItem.ItemId > 0 {
			item := items.New(saleItem.ItemId)
			if item.ItemId == 0 {
				continue
			}
			cat.itemNames = append(cat.itemNames, item.GetSpec().Name)
			cat.itemNamesFancy = append(cat.itemNamesFancy, item.DisplayName())
			cat.nameToShopItem[item.GetSpec().Name] = saleItem

			price := saleItem.Price
			if price == 0 {
				price = item.GetSpec().Value
			} else if price < 0 {
				price = 0
			}
			cat.itemPrices[saleItem.ItemId] = price
			continue
		}
		if saleItem.BuffId > 0 {
			buffInfo := buffs.GetBuffSpec(saleItem.BuffId)
			if buffInfo == nil {
				continue
			}
			cat.buffNames = append(cat.buffNames, buffInfo.Name)
			cat.nameToShopItem[buffInfo.Name] = saleItem

			price := saleItem.Price
			if price == 0 {
				price = 1000
			} else if price < 0 {
				price = 0
			}
			cat.buffPrices[saleItem.BuffId] = price
			continue
		}
		// Merc / pet entries on legacy shops are skipped — see spec 2.1.
	}
	return cat
}

// allNames returns the union of item + buff display names in the catalog
// for fuzzy matching. Merc/pet names are intentionally excluded.
func (c *legacyShopCatalog) allNames() []string {
	all := make([]string, 0, len(c.itemNames)+len(c.buffNames))
	all = append(all, c.itemNames...)
	all = append(all, c.buffNames...)
	return all
}

// purchaseContext holds the validated state of a purchase ready
// for execution.
type purchaseContext struct {
	matchedShopItem characters.ShopItem
	price           int
	tradeInString   string
}

// validatePurchase runs every pre-side-effect check, then applies
// side effects (destock, gold deduction, trade-item consume) only
// when all checks pass. Returns ok=false plus a populated Reason
// when any check fails; on failure no state is mutated.
func validatePurchase(
	buyer Actor,
	shopMob *mobs.Mob,
	shopUser *users.UserRecord,
	matchedShopItem characters.ShopItem,
	itemPrices map[int]int,
	buffPrices map[int]int,
) (purchaseContext, string, bool) {

	char := buyer.GetCharacter()

	// (1) Encumbrance gate — item purchases only.
	if matchedShopItem.ItemId > 0 {
		newItm := items.New(matchedShopItem.ItemId)
		weight := newItm.GetSpec().Weight
		if char.GetCarriedWeight()+weight > char.CarryCapacity() {
			if buyer.IsPlayer() {
				buyer.SendText(messaging.CategoryError, "You can't carry any more.")
			}
			return purchaseContext{}, BuyReasonOverburdened, false
		}
	}

	// (2) Stock check.
	if !matchedShopItem.Available() {
		if shopMob != nil {
			shopMob.Command(`say I don't have that for sale right now.`)
		} else if shopUser != nil && buyer.IsPlayer() {
			buyer.SendText(messaging.CategoryError, fmt.Sprintf(`<ansi fg="username">%s</ansi> doesn't have that for sale right now.`, shopUser.Character.Name))
		}
		return purchaseContext{}, BuyReasonOutOfStock, false
	}

	// (3) Price lookup.
	price := 0
	if matchedShopItem.ItemId > 0 {
		price = itemPrices[matchedShopItem.ItemId]
	} else if matchedShopItem.BuffId > 0 {
		price = buffPrices[matchedShopItem.BuffId]
	}

	// (4) Gold check.
	if char.Gold < price {
		sendMerchantMessage(buyer, shopMob, shopUser,
			`say You don't have enough gold for that.`,
			`You don't have enough gold for that.`)
		return purchaseContext{}, BuyReasonInsufficientGold, false
	}

	// (5) Trade-item check.
	tradeItemName := ""
	if matchedShopItem.TradeItemId > 0 {
		tradeItm := items.New(matchedShopItem.TradeItemId)
		tradeItemName = tradeItm.Name()
		if _, found := char.FindInBackpack(tradeItemName); !found {
			if buyer.IsPlayer() {
				buyer.SendText(messaging.CategoryError, fmt.Sprintf(`You must have a <ansi fg="itemname">%s</ansi> to trade for that.`, tradeItm.DisplayName()))
			}
			return purchaseContext{}, BuyReasonMissingTradeItem, false
		}
	}

	// All checks passed — apply side effects.
	if shopMob != nil {
		if !shopMob.Character.Shop.Destock(matchedShopItem) {
			shopMob.Command(`say I don't have that item right now.`)
			return purchaseContext{}, BuyReasonOutOfStock, false
		}
	} else if shopUser != nil {
		if !shopUser.Character.Shop.Destock(matchedShopItem) {
			if buyer.IsPlayer() {
				buyer.SendText(messaging.CategoryError, `That's not for sale.`)
			}
			return purchaseContext{}, BuyReasonOutOfStock, false
		}
	}

	if buyer.IsPlayer() {
		events.AddToQueue(events.EquipmentChange{
			UserId:     buyer.GetUserId(),
			GoldChange: -price,
		})
	}

	char.Gold -= price
	if shopMob != nil {
		shopMob.Character.Gold += 1 // legacy +1 cheat preserved
	} else if shopUser != nil {
		shopUser.Character.Gold += price
		events.AddToQueue(events.EquipmentChange{
			UserId:     shopUser.UserId,
			GoldChange: price,
		})
	}

	tradeInString := ""
	if price > 0 {
		tradeInString = fmt.Sprintf(`<ansi fg="gold">%d gold</ansi>`, price)
	}
	if tradeItemName != "" {
		if itm, found := char.FindInBackpack(tradeItemName); found {
			char.RemoveItem(itm)
			if buyer.IsPlayer() {
				events.AddToQueue(events.ItemOwnership{
					UserId: buyer.GetUserId(),
					Item:   itm,
					Gained: false,
				})
			} else {
				events.AddToQueue(events.ItemOwnership{
					MobInstanceId: buyer.GetMobInstanceId(),
					Item:          itm,
					Gained:        false,
				})
			}

			if tradeInString != "" {
				tradeInString += fmt.Sprintf(` and a <ansi fg="itemname">%s</ansi>`, itm.DisplayName())
			} else {
				tradeInString = fmt.Sprintf(`a <ansi fg="itemname">%s</ansi>`, itm.DisplayName())
			}
		}
	}
	if tradeInString == "" {
		tradeInString = "nothing"
	}

	return purchaseContext{
		matchedShopItem: matchedShopItem,
		price:           price,
		tradeInString:   tradeInString,
	}, "", true
}

// sendMerchantMessage delivers a message to the buyer, branching on
// mob vs player merchant. Mob merchants speak; player merchants send
// to the buyer directly.
func sendMerchantMessage(buyer Actor, shopMob *mobs.Mob, shopUser *users.UserRecord, mobMsg string, userMsg string) {
	if shopMob != nil {
		shopMob.Command(mobMsg)
	} else if shopUser != nil && buyer.IsPlayer() {
		buyer.SendText(messaging.CategoryError, userMsg)
	}
}

// Buy executes a purchase on behalf of buyer. See package context for
// the full flow.
func Buy(buyer Actor, opts BuyOptions) BuyResult {
	req := strings.TrimSpace(opts.Request)
	if req == "" {
		return BuyResult{Reason: BuyReasonNoRequest, Requested: 1}
	}

	// Parse leading quantity: "buy 5 iron ingot".
	quantity := 1
	args0 := strings.SplitN(strings.TrimSpace(req), " ", 2)
	if len(args0) == 2 {
		if n, err := strconv.Atoi(args0[0]); err == nil && n >= 1 {
			quantity = n
			req = args0[1]
		}
	}

	room := buyer.GetRoom()
	if room == nil {
		return BuyResult{Reason: BuyReasonNoMerchant, Requested: quantity}
	}

	// Parse trailing "from <name>" clause to resolve a specific merchant.
	itemRequest := req
	targetUserId := opts.TargetMerchantUserId
	targetMobInstanceId := opts.TargetMerchantMobInstanceId

	args := util.SplitButRespectQuotes(strings.ToLower(req))
	if len(args) >= 3 && args[len(args)-2] == "from" {
		mercName := args[len(args)-1]
		exclude := ResolveTargetOptions{}
		if buyer.IsPlayer() {
			exclude.ExcludeUserId = buyer.GetUserId()
		}
		target, terr := ResolveTargetActor(room, mercName, exclude)
		if terr == nil {
			if target.IsPlayer() {
				targetUserId = target.(*UserActor).User.UserId
			} else {
				targetMobInstanceId = target.(*MobActor).Mob.InstanceId
			}
			itemRequest = strings.Join(args[:len(args)-2], " ")
		} else if buyer.IsPlayer() {
			// Self-targeting collapses to NotFound under ExcludeUserId;
			// check explicitly so we can return BuyReasonSelfTarget.
			if pId, _ := room.FindByName(mercName); pId == buyer.GetUserId() {
				buyer.SendText(messaging.CategoryError, "You can't buy from yourself.")
				return BuyResult{Reason: BuyReasonSelfTarget, Requested: quantity}
			}
			buyer.SendText(messaging.CategorySystem, "Visit a merchant to purchase objects or services.")
			return BuyResult{Reason: BuyReasonNoMerchant, Requested: quantity}
		} else {
			return BuyResult{Reason: BuyReasonNoMerchant, Requested: quantity}
		}
	}

	merchantPlayers := room.GetPlayers(rooms.FindMerchant)
	merchantMobs := room.GetMobs(rooms.FindMerchant)

	if len(merchantPlayers) == 0 && len(merchantMobs) == 0 {
		if buyer.IsPlayer() {
			buyer.SendText(messaging.CategorySystem, "Visit a merchant to purchase objects or services.")
		}
		return BuyResult{Reason: BuyReasonNoMerchant, Requested: quantity}
	}

	// tryMerchant wraps a per-merchant attempt closure to support retries up to quantity.
	// The closure-passed attempt callback returns the result of a single purchase.
	// If successful, tryMerchant loops up to quantity times, calling attempt()
	// for each additional unit purchase.
	tryMerchant := func(attempt func() BuyResult) (BuyResult, bool) {
		first := attempt()
		if !first.Success {
			return first, false
		}
		purchased := 1
		for purchased < quantity {
			next := attempt()
			if !next.Success {
				break
			}
			purchased++
		}
		if quantity > 1 && purchased < quantity {
			if buyer.IsPlayer() {
				buyer.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="yellow">Purchased %d of %d before running short.</ansi>`, purchased, quantity))
			}
		}
		return BuyResult{
			Success:   true,
			Purchased: purchased,
			Requested: quantity,
			SaleType:  first.SaleType,
		}, true
	}

	// Iterate merchants — players first, then mobs.
	for _, uid := range merchantPlayers {
		if targetUserId > 0 && uid != targetUserId {
			continue
		}
		shopUser := users.GetByUserId(uid)
		if shopUser == nil {
			continue
		}
		result, sold := tryMerchant(func() BuyResult {
			r := tryPurchaseLegacy(buyer, itemRequest, nil, shopUser)
			if r.Success {
				postSuccessBookkeeping(buyer, nil, shopUser)
			}
			return r
		})
		if sold {
			return result
		}
	}

	for _, miid := range merchantMobs {
		if targetMobInstanceId > 0 && miid != targetMobInstanceId {
			continue
		}
		shopMob := mobs.GetInstance(miid)
		if shopMob == nil {
			continue
		}

		shopInv := shops.GetShopInventory(shopMob.Zone, int(shopMob.MobId), shopMob.HomeRoomId)
		if shopInv != nil {
			result, sold := tryMerchant(func() BuyResult {
				r := tryPurchaseFromInventory(buyer, itemRequest, shopMob, shopInv)
				if r.Success {
					postSuccessBookkeeping(buyer, shopMob, nil)
				}
				return r
			})
			if sold {
				return result
			}
		} else {
			result, sold := tryMerchant(func() BuyResult {
				shopMob.Character.Shop.Restock()
				r := tryPurchaseLegacy(buyer, itemRequest, shopMob, nil)
				if r.Success {
					postSuccessBookkeeping(buyer, shopMob, nil)
				}
				return r
			})
			if sold {
				return result
			}
		}
	}

	return BuyResult{Reason: BuyReasonNoMatch, Requested: quantity}
}

// tryPurchaseLegacy attempts a single purchase against a merchant
// backed by the legacy Character.Shop. Returns the populated
// BuyResult on either success or failure; non-empty Reason on
// failure means the outer Buy loop should try the next merchant.
func tryPurchaseLegacy(buyer Actor, request string, shopMob *mobs.Mob, shopUser *users.UserRecord) BuyResult {
	var saleItems characters.Shop
	if shopMob != nil {
		saleItems = shopMob.Character.Shop.GetInstock()
	} else if shopUser != nil {
		saleItems = shopUser.Character.Shop.GetInstock()
	}

	cat := buildLegacyCatalog(saleItems)

	match, closeMatch := util.FindMatchIn(request, cat.allNames()...)
	if match == "" {
		match = closeMatch
	}
	if match == "" {
		if shopMob != nil {
			extraSay := ""
			if len(cat.itemNamesFancy) > 0 {
				randSelection := util.Rand(len(cat.itemNamesFancy))
				extraSay = fmt.Sprintf(` Any interest in this <ansi fg="itemname">%s</ansi>?`, cat.itemNamesFancy[randSelection])
			} else if len(cat.buffNames) > 0 {
				randSelection := util.Rand(len(cat.buffNames))
				extraSay = fmt.Sprintf(` Maybe you would enjoy this %s enchantment?`, cat.buffNames[randSelection])
			}
			shopMob.Command(`say Sorry, I can't offer that right now.`+extraSay, 1)
		}
		return BuyResult{Reason: BuyReasonNoMatch}
	}

	ctx, reason, ok := validatePurchase(buyer, shopMob, shopUser, cat.nameToShopItem[match], cat.itemPrices, cat.buffPrices)
	if !ok {
		return BuyResult{Reason: reason}
	}

	if ctx.matchedShopItem.ItemId > 0 {
		executePurchaseItem(buyer, shopMob, shopUser, ctx.matchedShopItem, ctx.price, ctx.tradeInString)
		return BuyResult{Success: true, Purchased: 1, SaleType: "item"}
	}
	if ctx.matchedShopItem.BuffId > 0 {
		executePurchaseBuff(buyer, shopMob, shopUser, ctx.matchedShopItem, ctx.price, ctx.tradeInString)
		return BuyResult{Success: true, Purchased: 1, SaleType: "buff"}
	}

	// Merc/pet sale types are filtered out at catalog build time;
	// reaching this branch means malformed data.
	return BuyResult{Reason: BuyReasonNoMatch}
}

// tryPurchaseFromInventory attempts a single purchase against a
// ShopInventory-backed mob merchant. Buff/merc/pet purchases are
// NOT handled here — ShopInventory only carries items.
func tryPurchaseFromInventory(buyer Actor, request string, shopMob *mobs.Mob, shopInv *shops.ShopInventory) BuyResult {
	cfg := shops.PricingConfigFromBalance()

	type invEntry struct {
		entry      *shops.StockEntry
		item       items.Item
		plainName  string
		price      int
		affixedIdx int // index into shopInv.AffixedStock, or -1 for base-ItemId stock
	}

	var available []invEntry
	var itemNames []string
	var itemNamesFancy []string

	char := buyer.GetCharacter()

	for i := range shopInv.Stock {
		entry := &shopInv.Stock[i]
		if entry.Current <= 0 {
			continue
		}
		itm := items.New(entry.ItemId)
		if itm.ItemId == 0 {
			continue
		}
		spec := itm.GetSpec()
		restock := shops.PricingBaseline(entry, cfg)
		basePrice := shops.CalcSellPrice(spec.Value, entry.Current, restock, cfg)

		// Bartering discount — works symmetrically for both buyer types.
		barterSkill := char.GetSkillLevel(skills.Bartering)
		if barterSkill > 0 {
			discount := float64(barterSkill) / 50.0 * 0.15 // max 15% at skill 50
			basePrice = shops.ApplyBarterSellDiscount(basePrice, discount)
		}

		available = append(available, invEntry{
			entry:      entry,
			item:       itm,
			plainName:  spec.Name,
			price:      basePrice,
			affixedIdx: -1,
		})
		itemNames = append(itemNames, spec.Name)
		itemNamesFancy = append(itemNamesFancy, itm.DisplayName())
	}

	// Per-instance affixed resale stock (Stage 3): unique bought-back gear,
	// priced at its stored relist price (AffixValue x 1.0), less any barter.
	for i := range shopInv.AffixedStock {
		e := &shopInv.AffixedStock[i]
		spec := e.Item.GetSpec()
		price := e.Price
		if barterSkill := char.GetSkillLevel(skills.Bartering); barterSkill > 0 {
			discount := float64(barterSkill) / 50.0 * 0.15
			price = shops.ApplyBarterSellDiscount(price, discount)
		}
		available = append(available, invEntry{
			item:       e.Item,
			plainName:  spec.Name,
			price:      price,
			affixedIdx: i,
		})
		itemNames = append(itemNames, spec.Name)
		itemNamesFancy = append(itemNamesFancy, e.Item.DisplayName())
	}

	match, closeMatch := util.FindMatchIn(request, itemNames...)
	if match == "" {
		match = closeMatch
	}
	if match == "" {
		if shopMob != nil {
			extraSay := ""
			if len(itemNamesFancy) > 0 {
				randSelection := util.Rand(len(itemNamesFancy))
				extraSay = fmt.Sprintf(` Any interest in this <ansi fg="itemname">%s</ansi>?`, itemNamesFancy[randSelection])
			}
			shopMob.Command(`say Sorry, I can't offer that right now.` + extraSay)
		}
		return BuyResult{Reason: BuyReasonNoMatch}
	}

	var matched *invEntry
	for i := range available {
		if available[i].plainName == match {
			matched = &available[i]
			break
		}
	}
	if matched == nil {
		return BuyResult{Reason: BuyReasonNoMatch}
	}

	// Encumbrance gate — same pre-side-effect check as the legacy
	// path, since ShopInventory bypasses validatePurchase.
	if char.GetCarriedWeight()+matched.item.GetSpec().Weight > char.CarryCapacity() {
		if buyer.IsPlayer() {
			buyer.SendText(messaging.CategoryError, "You can't carry any more.")
		}
		return BuyResult{Reason: BuyReasonOverburdened}
	}

	// Affixed resale purchase (Stage 3): hand the exact stored item with a fresh
	// UUID and remove the entry. Bypasses the base-ItemId destock path below
	// (matched.entry is nil for affixed rows).
	if matched.affixedIdx >= 0 {
		if char.Gold < matched.price {
			if shopMob != nil {
				shopMob.Command(`say You don't have enough gold for that.`)
			} else if buyer.IsPlayer() {
				buyer.SendText(messaging.CategoryError, `You don't have enough gold for that.`)
			}
			return BuyResult{Reason: BuyReasonInsufficientGold}
		}
		bought, ok := shopInv.RemoveAffixedStock(matched.affixedIdx)
		if !ok {
			return BuyResult{Reason: BuyReasonOutOfStock}
		}
		bought.UUID = items.NewItemUUID()
		if !char.StoreItem(bought) {
			shopInv.AddAffixedStock(bought, matched.price, 0) // roll back on carry failure
			if buyer.IsPlayer() {
				buyer.SendText(messaging.CategoryError, "You can't carry any more.")
			}
			return BuyResult{Reason: BuyReasonOverburdened}
		}
		shopInv.SalesCount++
		char.Gold -= matched.price
		shopInv.Gold += matched.price
		if buyer.IsPlayer() {
			events.AddToQueue(events.ItemOwnership{UserId: buyer.GetUserId(), Item: bought, Gained: true})
			events.AddToQueue(events.EquipmentChange{UserId: buyer.GetUserId(), GoldChange: -matched.price})
		}
		if err := shops.SaveShop(shopInv.Zone, shopInv.MobId, shopInv.RoomId); err != nil {
			mudlog.Error("PURCHASE", "msg", "SaveShop failed", "error", err)
		}
		buyer.SendText(messaging.CategoryLoot, fmt.Sprintf(`You buy the <ansi fg="itemname">%s</ansi>.`, bought.DisplayName()))
		if room := buyer.GetRoom(); room != nil {
			room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="username">%s</ansi> buys the <ansi fg="itemname">%s</ansi>.`, buyer.GetName(), bought.DisplayName()), buyer.GetUserId())
		}
		return BuyResult{Success: true, Purchased: 1, SaleType: "item"}
	}

	if matched.entry.Current <= 0 {
		if shopMob != nil {
			shopMob.Command(`say I don't have that for sale right now.`)
		}
		return BuyResult{Reason: BuyReasonOutOfStock}
	}

	if char.Gold < matched.price {
		if shopMob != nil {
			shopMob.Command(`say You don't have enough gold for that.`)
		} else if buyer.IsPlayer() {
			buyer.SendText(messaging.CategoryError, `You don't have enough gold for that.`)
		}
		return BuyResult{Reason: BuyReasonInsufficientGold}
	}

	if shopInv.RemoveStockAtRound(matched.entry.ItemId, 1, util.GetRoundCount()) == 0 {
		if shopMob != nil {
			shopMob.Command(`say I don't have that item right now.`)
		}
		return BuyResult{Reason: BuyReasonOutOfStock}
	}
	shopInv.SalesCount++

	if buyer.IsPlayer() {
		events.AddToQueue(events.EquipmentChange{
			UserId:     buyer.GetUserId(),
			GoldChange: -matched.price,
		})
	}
	char.Gold -= matched.price
	shopInv.Gold += matched.price

	if err := shops.SaveShop(shopInv.Zone, shopInv.MobId, shopInv.RoomId); err != nil {
		mudlog.Error("PURCHASE", "msg", "SaveShop failed", "error", err)
	}

	tradeInString := fmt.Sprintf(`<ansi fg="gold">%d gold</ansi>`, matched.price)
	if matched.price == 0 {
		tradeInString = "nothing"
	}

	executePurchaseItem(buyer, shopMob, nil,
		characters.ShopItem{ItemId: matched.entry.ItemId},
		matched.price, tradeInString)

	return BuyResult{Success: true, Purchased: 1, SaleType: "item"}
}

// executePurchaseItem lands a purchased item in the buyer's
// inventory and emits the buyer + room messages.
func executePurchaseItem(buyer Actor, shopMob *mobs.Mob, shopUser *users.UserRecord, matchedShopItem characters.ShopItem, price int, tradeInString string) {
	newItm := items.New(matchedShopItem.ItemId)

	if buyer.IsPlayer() {
		userId := buyer.GetUserId()
		if u := users.GetByUserId(userId); u != nil {
			u.PlaySound(`purchase`, `other`)
		}
		events.AddToQueue(events.ItemOwnership{
			UserId: userId,
			Item:   newItm,
			Gained: true,
		})
	} else {
		events.AddToQueue(events.ItemOwnership{
			MobInstanceId: buyer.GetMobInstanceId(),
			Item:          newItm,
			Gained:        true,
		})
	}

	buyerName := buyer.GetName()

	if shopMob != nil {
		if buyer.IsPlayer() {
			if u := users.GetByUserId(buyer.GetUserId()); u != nil {
				u.EventLog.Add(`shop`, fmt.Sprintf(`Purchased a <ansi fg="itemname">%s</ansi> from <ansi fg="mobname">%s</ansi> for %s`, newItm.DisplayName(), shopMob.Character.Name, tradeInString))
			}
		}
		buyer.SendText(messaging.CategoryLoot, fmt.Sprintf(`You purchase the <ansi fg="itemname">%s</ansi> from <ansi fg="mobname">%s</ansi> for %s.`, newItm.DisplayName(), shopMob.Character.Name, tradeInString))
		if room := buyer.GetRoom(); room != nil {
			room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="username">%s</ansi> purchases the <ansi fg="itemname">%s</ansi> from <ansi fg="mobname">%s</ansi>.`, buyerName, newItm.DisplayName(), shopMob.Character.Name), buyer.GetUserId())
		}
	} else if shopUser != nil {
		if buyer.IsPlayer() {
			if u := users.GetByUserId(buyer.GetUserId()); u != nil {
				u.EventLog.Add(`shop`, fmt.Sprintf(`Purchased a <ansi fg="itemname">%s</ansi> from <ansi fg="username">%s</ansi> for %s.`, newItm.DisplayName(), shopUser.Character.Name, tradeInString))
			}
		}
		buyer.SendText(messaging.CategoryLoot, fmt.Sprintf(`You purchase the <ansi fg="itemname">%s</ansi> from <ansi fg="username">%s</ansi> for %s.`, newItm.DisplayName(), shopUser.Character.Name, tradeInString))
		shopUser.SendText(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="username">%s</ansi> purchased the <ansi fg="itemname">%s</ansi> you were selling for %s.`, buyerName, newItm.DisplayName(), tradeInString))
		if room := buyer.GetRoom(); room != nil {
			room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="username">%s</ansi> purchases the <ansi fg="itemname">%s</ansi> from <ansi fg="username">%s</ansi>.`, buyerName, newItm.DisplayName(), shopUser.Character.Name), buyer.GetUserId())
		}
	}

	buyer.GetCharacter().StoreItem(newItm)
}

// executePurchaseBuff applies the bought buff to the buyer and
// emits the merchant emote follow-up.
func executePurchaseBuff(buyer Actor, shopMob *mobs.Mob, shopUser *users.UserRecord, matchedShopItem characters.ShopItem, price int, tradeInString string) {
	buffSpec := buffs.GetBuffSpec(matchedShopItem.BuffId)
	buyerName := buyer.GetName()

	if shopMob != nil {
		if buyer.IsPlayer() {
			if u := users.GetByUserId(buyer.GetUserId()); u != nil {
				u.EventLog.Add(`shop`, fmt.Sprintf(`Purchased a <ansi fg="buff">%s</ansi> enchantment from <ansi fg="mobname">%s</ansi> for %s`, buffSpec.Name, shopMob.Character.Name, tradeInString))
			}
		}
		buyer.SendText(messaging.CategoryLoot, fmt.Sprintf(`You pay %s to <ansi fg="mobname">%s</ansi>.`, tradeInString, shopMob.Character.Name))
		if room := buyer.GetRoom(); room != nil {
			room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="username">%s</ansi> pays %s to <ansi fg="mobname">%s</ansi>.`, buyerName, tradeInString, shopMob.Character.Name), buyer.GetUserId())
		}
		shopMob.Command(`emote mutters a soft incantation.`, 1)
	} else if shopUser != nil {
		if buyer.IsPlayer() {
			if u := users.GetByUserId(buyer.GetUserId()); u != nil {
				u.EventLog.Add(`shop`, fmt.Sprintf(`Purchased a <ansi fg="buff">%s</ansi> enchantment from  <ansi fg="username">%s</ansi> for %s`, buffSpec.Name, shopUser.Character.Name, tradeInString))
			}
		}
		buyer.SendText(messaging.CategoryLoot, fmt.Sprintf(`You pay %s to <ansi fg="username">%s</ansi>.`, tradeInString, shopUser.Character.Name))
		shopUser.SendText(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="username">%s</ansi> pays you %s for an enchantment.`, buyerName, tradeInString))
		if room := buyer.GetRoom(); room != nil {
			room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="username">%s</ansi> pays to <ansi fg="username">%s</ansi> for an enchantment.`, buyerName, shopUser.Character.Name), buyer.GetUserId())
		}
		// player-merchant doesn't emote (matches existing behavior).
	}

	buyer.AddBuff(matchedShopItem.BuffId, "shop")

	if shopMob != nil {
		shopMob.Command(`say I've done what I can.`, 1)
	}
}

// postSuccessBookkeeping runs after any successful purchase: skill
// progression for the buyer, charisma stat-use on the merchant mob,
// and quest-engine notification gated by buyer.IsPlayer().
func postSuccessBookkeeping(buyer Actor, shopMob *mobs.Mob, shopUser *users.UserRecord) {
	buyer.OnSkillUse("bartering")

	if shopMob != nil {
		shopMob.Character.OnStatUse("charisma", 0)
	}

	if buyer.IsPlayer() {
		userId := buyer.GetUserId()
		if u := users.GetByUserId(userId); u != nil {
			room := buyer.GetRoom()
			bridge := questengine.NewGameBridge(u, room.RoomId)
			questengine.GetEngine().Notify("command", questengine.EventDetails{
				UserId:  userId,
				RoomId:  room.RoomId,
				Command: "buy",
			}, bridge, bridge)
		}
	}
}
