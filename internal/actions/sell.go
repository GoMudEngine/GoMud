package actions

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// UnlimitedSell is the Quantity sentinel meaning "sell every match".
const UnlimitedSell = math.MaxInt

type SellOptions struct {
	ItemName        string // ignored when SellAllSellable
	Quantity        int    // 1, N, or UnlimitedSell
	SellAllSellable bool   // mob inventory-sweep mode (every sellable item)
	MerchantName    string // optional target merchant name; "" = first willing
}

type SellStopReason int

const (
	SellStopSoldAll       SellStopReason = iota // ran out of matching items (normal)
	SellStopNoItem                              // seller never had the item
	SellStopNoMerchant                          // no willing merchant in room
	SellStopMerchantBroke                       // merchant ran out of gold (player path only)
	SellStopRejected                            // merchant declined the item type
)

type SellResult struct {
	Sold         int
	TotalGold    int
	Reason       SellStopReason
	LastItemName string
}

// Sell is the shared seller entry point for players and mobs. The seller is
// abstracted via Actor; the merchant is a shopkeeper mob resolved from the
// seller's room. Player sells draw down shop gold; mob sells credit the seller
// but leave shop gold intact (see internal/shops/context.md "two sell models").
//
// NOTE: forager.SellToVendor is a different path — a free supply handoff, not a
// sale. See internal/forager/vendor_sell.go.
func Sell(seller Actor, opts SellOptions) SellResult {
	room := seller.GetRoom()
	if room == nil {
		return SellResult{Reason: SellStopNoMerchant}
	}
	if opts.SellAllSellable {
		return sellSweep(seller, room)
	}
	if opts.Quantity < 1 {
		opts.Quantity = 1
	}
	return sellNamed(seller, room, opts.ItemName, opts.Quantity)
}

// merchantSay makes the merchant speak a line to the room immediately.
//
// Merchant refusals ("I'm not interested in that.", "I can't afford that
// right now.", etc.) were previously delivered via mob.Command("say ..."),
// which enqueues the line on the mob's ASYNC command pipeline (events.Input,
// gated on the mob's next turn). On busy shopkeepers — scheduled townsfolk and
// autonomous crafters like Kerra and Voss — that pipeline can defer the line
// for many turns, long enough that the seller never associates it with their
// sell attempt and the sale appears to fail silently. Every other sell outcome
// (success, no-item, no-merchant, quest-item) reports synchronously, so the
// refusal path was the lone async outlier. Speak synchronously instead — the
// same broadcast mobcommands.Say performs — so the seller always gets
// immediate, reliable feedback.
func merchantSay(room *rooms.Room, mob *mobs.Mob, line string) {
	if mob == nil || room == nil {
		return
	}
	actor := &MobActor{Mob: mob, Room: room}
	result := Say(actor, line)
	room.SendText(messaging.CategorySpeech,
		FormatSayText(mob.Character.Name, result.Text, false, "mobname", "saytext-mob"))
}

// affixedSellPrice is the fixed-spread price a shop pays for an affix-scaled
// instance item: its (Stage-1 stamped) value times the buy/sell spread. It
// deliberately bypasses the scarcity curve and the legacy 25% cap — unique
// affixed gear is priced on value, not commodity stock levels.
func affixedSellPrice(item items.Item, cfg shops.PricingConfig) int {
	price := int(math.Ceil(float64(item.GetSpec().Value) * cfg.BuyRatio))
	if price < 1 {
		price = 1
	}
	return price
}

// resolveMerchant finds the first merchant in the room willing to buy probe,
// returning the merchant mob and its living-economy ShopInventory (nil for
// legacy-shop merchants).
func resolveMerchant(room *rooms.Room, probe items.Item) (*mobs.Mob, *shops.ShopInventory) {
	for _, mobId := range room.GetMobs(rooms.FindMerchant) {
		mob := mobs.GetInstance(mobId)
		if mob == nil {
			continue
		}
		shopInv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)
		var probeValue int
		if probe.Affixed {
			probeValue = affixedSellPrice(probe, shops.PricingConfigFromBalance())
		} else if shopInv != nil {
			cfg := shops.PricingConfigFromBalance()
			wornItems := mob.Character.Equipment.GetAllItemsWithEmptySlots()
			offer := shops.EvaluateBuyRules(probe, shopInv, mob.CrafterSkill, mob.BuysGeneral, cfg, wornItems)
			probeValue = offer.Price
		} else {
			probeValue = mob.GetSellPrice(probe)
		}
		if probeValue > 0 {
			return mob, shopInv
		}
	}
	return nil, nil
}

// firstMerchantInRoom returns the first merchant present in the room and its
// ShopInventory (nil for legacy-shop merchants), regardless of whether it will
// buy any particular item. Returns (nil, nil) only when no merchant is present
// at all. Used by the named-sell path to distinguish "no merchant here" from
// "a merchant is here but won't buy this", so the latter routes through
// sellOneToMerchant for the proper spoken refusal instead of the misleading
// "There's no merchant here." message.
func firstMerchantInRoom(room *rooms.Room) (*mobs.Mob, *shops.ShopInventory) {
	for _, mobId := range room.GetMobs(rooms.FindMerchant) {
		mob := mobs.GetInstance(mobId)
		if mob == nil {
			continue
		}
		shopInv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)
		return mob, shopInv
	}
	return nil, nil
}

func sellNamed(seller Actor, room *rooms.Room, itemName string, quantity int) SellResult {
	char := seller.GetCharacter()
	probe, found := sellFindItemInChar(char, itemName)
	if !found {
		if seller.IsPlayer() {
			seller.SendText(messaging.CategorySystem, "You don't have that item.")
		}
		return SellResult{Reason: SellStopNoItem}
	}
	if probe.GetSpec().QuestToken != "" {
		if seller.IsPlayer() {
			seller.SendText(messaging.CategorySystem, "Quest items cannot be sold!")
		}
		return SellResult{Reason: SellStopRejected}
	}
	mob, shopInv := resolveMerchant(room, probe)
	if mob == nil {
		// No WILLING merchant. Distinguish "no merchant present at all" from
		// "a merchant is here but won't buy this item." For the latter, route
		// through sellOneToMerchant so the merchant gives the right spoken
		// refusal ("I'm not interested in that." / "I can't afford that right
		// now.") and we return SellStopRejected / SellStopMerchantBroke —
		// rather than the misleading SellStopNoMerchant ("There's no merchant
		// here.").
		mob, shopInv = firstMerchantInRoom(room)
		if mob == nil {
			return SellResult{Reason: SellStopNoMerchant}
		}
	}
	var out SellResult
	out.Reason = SellStopSoldAll
	for out.Sold < quantity {
		value, res := sellOneToMerchant(seller, itemName, room, mob, shopInv)
		if res != SellStopSoldAll {
			// Running out of matching items mid-loop is a NORMAL completion
			// (the "sold all I had" case) — only surface it as SellStopNoItem
			// when nothing was sold at all (the seller never had the item).
			// Merchant-side stops (broke / rejected) always surface.
			if res == SellStopNoItem && out.Sold > 0 {
				out.Reason = SellStopSoldAll
			} else {
				out.Reason = res
			}
			break
		}
		out.Sold++
		out.TotalGold += value
		out.LastItemName = probe.GetSpec().Name
	}
	return out
}

func sellSweep(seller Actor, room *rooms.Room) SellResult {
	char := seller.GetCharacter()
	var out SellResult
	out.Reason = SellStopSoldAll
	snapshot := append([]items.Item{}, char.Items...)
	soldAny := false
	for _, itm := range snapshot {
		spec := itm.GetSpec()
		if spec.ItemId < 1 || spec.QuestToken != "" || spec.Value <= 0 || spec.IsComponent {
			continue
		}
		mob, shopInv := resolveMerchant(room, itm)
		if mob == nil {
			continue
		}
		value, res := sellOneToMerchant(seller, itm.Name(), room, mob, shopInv)
		if res == SellStopSoldAll {
			soldAny = true
			out.Sold++
			out.TotalGold += value
			out.LastItemName = spec.Name
		}
	}
	if !soldAny {
		out.Reason = SellStopRejected
	}
	return out
}

// sellOneToMerchant sells a single matching item from seller to the given
// merchant. Mirrors the player trySellOne, with two changes:
//   - seller-side access via Actor (GetCharacter / Gold / RemoveItem).
//   - shop-gold drain + merchant-broke check apply only to player sellers
//     (decision 2: NPC selling never bankrupts a shop).
func sellOneToMerchant(seller Actor, itemName string, room *rooms.Room,
	mob *mobs.Mob, shopInv *shops.ShopInventory) (soldValue int, res SellStopReason) {

	char := seller.GetCharacter()
	item, found := sellFindItemInChar(char, itemName)
	if !found {
		return 0, SellStopNoItem
	}
	itemSpec := item.GetSpec()
	if itemSpec.ItemId < 1 {
		return 0, SellStopRejected
	}
	if itemSpec.QuestToken != "" {
		if seller.IsPlayer() {
			seller.SendText(messaging.CategorySystem, "Quest items cannot be sold!")
		}
		return 0, SellStopRejected
	}

	char.CancelBuffsWithFlag(buffs.Hidden)
	// Affixed instance loot is sellable despite carrying a per-instance Spec;
	// every other custom-spec item (enchanted / blob / uses) stays blocked.
	if item.IsSpecial() && !item.Affixed {
		merchantSay(room, mob, "I'm afraid I don't buy those.")
		return 0, SellStopRejected
	}

	var sellValue int
	var buyReason string
	if item.Affixed {
		// Unique affix-scaled loot: fixed spread off its stamped value, bypassing
		// scarcity pricing and the legacy 25% cap.
		sellValue = affixedSellPrice(item, shops.PricingConfigFromBalance())
	} else if shopInv != nil {
		cfg := shops.PricingConfigFromBalance()
		wornItems := mob.Character.Equipment.GetAllItemsWithEmptySlots()
		offer := shops.EvaluateBuyRules(item, shopInv, mob.CrafterSkill, mob.BuysGeneral, cfg, wornItems)
		sellValue = offer.Price
		buyReason = offer.Reason
		if sellValue > 0 {
			barterSkill := char.GetSkillLevel(skills.Bartering)
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
		merchantSay(room, mob, "I'm not interested in that.")
		return 0, SellStopRejected
	}

	// Gold-model gate: only players are constrained by — and draw down —
	// the merchant's gold. Mob sales mint the seller's payout.
	if seller.IsPlayer() {
		merchantGold := mob.Character.Gold
		if shopInv != nil {
			merchantGold = shopInv.Gold
		}
		if merchantGold < sellValue {
			merchantSay(room, mob, "I can't afford that right now.")
			return 0, SellStopMerchantBroke
		}
		if shopInv != nil {
			shopInv.Gold -= sellValue
		} else {
			mob.Character.Gold -= sellValue
		}
	}

	char.Gold += sellValue
	char.RemoveItem(item)

	if seller.IsPlayer() {
		events.AddToQueue(events.ItemOwnership{UserId: seller.GetUserId(), Item: item, Gained: false})
		events.AddToQueue(events.EquipmentChange{UserId: seller.GetUserId(), GoldChange: sellValue})
	} else {
		events.AddToQueue(events.ItemOwnership{MobInstanceId: seller.GetMobInstanceId(), Item: item, Gained: false})
	}

	// Stock update (merchant side). Affixed items are melted in Stage 2 — NOT
	// stocked as a base ItemId (Stage 3 adds per-instance resale stock).
	if !item.Affixed {
		if shopInv != nil {
			shopInv.BuysCount++
			if buyReason == "gear_upgrade" {
				newItem := items.New(item.ItemId)
				if newItem.ItemId > 0 {
					returnedItems, wore, _ := mob.Character.Wear(newItem)
					if wore {
						for _, old := range returnedItems {
							if old.ItemId > 0 {
								shopInv.AddStockAtRound(old.ItemId, 1, util.GetRoundCount())
							}
						}
						room.SendTextVisual(messaging.CategoryLoot,
							fmt.Sprintf(`<ansi fg="mobname">%s</ansi> examines the <ansi fg="itemname">%s</ansi> and puts it on.`, mob.Character.Name, newItem.DisplayName()),
							seller.GetUserId(),
						)
					} else {
						shopInv.AddStockAtRound(item.ItemId, 1, util.GetRoundCount())
					}
				} else {
					shopInv.AddStockAtRound(item.ItemId, 1, util.GetRoundCount())
				}
			} else {
				shopInv.AddStockAtRound(item.ItemId, 1, util.GetRoundCount())
			}
			if err := shops.SaveShop(mob.Zone, int(mob.MobId), mob.HomeRoomId); err != nil {
				mudlog.Error("SELL", "msg", "SaveShop failed", "error", err)
			}
		} else {
			mob.Character.Shop.StockItem(item.ItemId)
		}
	}

	// Progression.
	seller.OnSkillUse(string(skills.Bartering))
	mob.Character.OnStatUse("charisma", 0)

	return sellValue, SellStopSoldAll
}

// sellFindItemInChar searches backpack → potions → components for a match.
func sellFindItemInChar(char *characters.Character, name string) (items.Item, bool) {
	item, found := char.FindInBackpack(name)
	if !found {
		item, found = char.FindInPotions(name)
	}
	if !found {
		item, found = char.FindInComponents(name)
	}
	return item, found
}
