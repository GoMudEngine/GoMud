package actions

import (
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
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
