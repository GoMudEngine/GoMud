package actions

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
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
				buyer.SendText("You can't carry any more.")
			}
			return purchaseContext{}, BuyReasonOverburdened, false
		}
	}

	// (2) Stock check.
	if !matchedShopItem.Available() {
		if shopMob != nil {
			shopMob.Command(`say I don't have that for sale right now.`)
		} else if shopUser != nil && buyer.IsPlayer() {
			buyer.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> doesn't have that for sale right now.`, shopUser.Character.Name))
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
				buyer.SendText(fmt.Sprintf(`You must have a <ansi fg="itemname">%s</ansi> to trade for that.`, tradeItm.DisplayName()))
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
				buyer.SendText(`That's not for sale.`)
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
		buyer.SendText(userMsg)
	}
}

// Buy executes a purchase on behalf of buyer. See package context for
// the full flow.
func Buy(buyer Actor, opts BuyOptions) BuyResult {
	req := strings.TrimSpace(opts.Request)
	if req == "" {
		return BuyResult{Reason: BuyReasonNoRequest, Requested: 1}
	}

	room := buyer.GetRoom()
	if room == nil {
		return BuyResult{Reason: BuyReasonNoMerchant, Requested: 1}
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
				buyer.SendText("You can't buy from yourself.")
				return BuyResult{Reason: BuyReasonSelfTarget, Requested: 1}
			}
			buyer.SendText("Visit a merchant to purchase objects or services.")
			return BuyResult{Reason: BuyReasonNoMerchant, Requested: 1}
		} else {
			return BuyResult{Reason: BuyReasonNoMerchant, Requested: 1}
		}
	}

	merchantPlayers := room.GetPlayers(rooms.FindMerchant)
	merchantMobs := room.GetMobs(rooms.FindMerchant)

	if len(merchantPlayers) == 0 && len(merchantMobs) == 0 {
		if buyer.IsPlayer() {
			buyer.SendText("Visit a merchant to purchase objects or services.")
		}
		return BuyResult{Reason: BuyReasonNoMerchant, Requested: 1}
	}

	// Task 4 and onwards: per-merchant purchase loop.
	_ = itemRequest
	_ = targetUserId
	_ = targetMobInstanceId
	_ = strconv.Atoi // placeholder until Task 8

	return BuyResult{Reason: BuyReasonNoMatch, Requested: 1}
}
