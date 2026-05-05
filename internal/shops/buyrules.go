package shops

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// BuyOffer represents what an NPC is willing to pay for an item.
type BuyOffer struct {
	Price  int
	Reason string // "craft_material", "potion", "general", ""
}

// EvaluateBuyRules returns what an NPC will pay for an item offered by
// a player. Single-rule tag-overlap with overstock and gold-reserve gates.
//
// Reject conditions, in order:
//  1. Item has no ItemSpec or carries a QuestToken.
//  2. Item is a potion in PhaseDeclining or PhaseSpoiled.
//  3. Item has no vendor_categories tags.
//  4. Vendor's craft_support doesn't accept any of the item's tags.
//  5. Vendor is at MaxStock for this item ("48 iron ores" overstock cap).
//  6. Vendor can't afford the buy price without dropping below
//     shopInv.GoldReserve(BalanceConfig.ShopGoldReserveRatio) — defaults
//     to 0.50 when the config knob is unset.
//
// Otherwise returns a BuyOffer with dynamic price from CalcBuyPrice.
//
// crafterSkill, buysGeneral, wornItems are unused in the new logic but
// kept in the signature for back-compat with the call site in
// internal/usercommands/sell.go.
func EvaluateBuyRules(
	item items.Item,
	shopInv *ShopInventory,
	crafterSkill string,
	buysGeneral bool,
	cfg PricingConfig,
	wornItems []items.Item,
) BuyOffer {
	spec := item.GetSpec()
	if spec.ItemId < 1 || spec.QuestToken != "" {
		return BuyOffer{}
	}

	if spec.Type == items.Potion && isPotionDeclining(item, &spec) {
		return BuyOffer{}
	}

	if len(spec.VendorCategories) == 0 {
		return BuyOffer{}
	}
	if !vendorAcceptsAny(shopInv.CraftSupport, spec.VendorCategories) {
		return BuyOffer{}
	}

	// Overstock cap.
	if entry := shopInv.GetStock(spec.ItemId); entry != nil &&
		entry.MaxStock > 0 && entry.Current >= entry.MaxStock {
		return BuyOffer{}
	}

	// Compute price. Two paths:
	//   - Item is in the shop's stock list → use full dynamic pricing
	//     (scarcity multiplier from current vs RestockQty).
	//   - Item is NOT in the stock list → flat value × BuyRatio. The
	//     scarcity concept only makes sense for items the shop actively
	//     stocks; one-off offhand items shouldn't trigger the 5×
	//     PriceCeiling, which would push the price above the gold
	//     reserve and self-reject. (Issue caught 2026-05-04 — Maren
	//     rejecting cattail cloak, Kerra rejecting arena tower shield.)
	var price int
	entry := shopInv.GetStock(spec.ItemId)
	if entry != nil {
		current := entry.Current
		restock := 1
		if entry.RestockQty > 0 {
			restock = entry.RestockQty
		}
		price = CalcBuyPrice(spec.Value, current, restock, cfg)
	} else {
		flat := int(float64(spec.Value) * cfg.BuyRatio)
		if flat < 1 {
			flat = 1
		}
		price = flat
	}

	// Gold-reserve gate.
	b := configs.GetBalanceConfig()
	reserveRatio := float64(b.ShopGoldReserveRatio)
	if reserveRatio <= 0 {
		reserveRatio = 0.50 // fallback default
	}
	reserve := shopInv.GoldReserve(reserveRatio)
	if !shopInv.CanAfford(price, reserve) {
		return BuyOffer{}
	}

	return BuyOffer{Price: price, Reason: pickReason(&spec)}
}

// vendorAcceptsAny returns true if craftSupport is "general", or any
// of the item's tags matches craftSupport.
func vendorAcceptsAny(craftSupport string, itemTags []string) bool {
	if craftSupport == CraftSupportGeneral {
		return true
	}
	for _, t := range itemTags {
		if t == craftSupport {
			return true
		}
	}
	return false
}

// pickReason returns the legacy Reason string for back-compat with
// any caller that inspects it (the sell command does, for flavor text).
func pickReason(spec *items.ItemSpec) string {
	if spec.Type == items.Potion {
		return "potion"
	}
	if spec.IsComponent {
		return "craft_material"
	}
	return "general"
}

// isPotionDeclining reports whether a potion's aging phase is Declining
// or Spoiled — those should never be bought (potions whose magic has
// faded or gone toxic).
func isPotionDeclining(item items.Item, spec *items.ItemSpec) bool {
	if !spec.Aging.HasAging() || item.CraftedRound == 0 {
		return false
	}
	currentRound := util.GetRoundCount()
	var elapsed uint64
	if currentRound >= item.CraftedRound {
		elapsed = currentRound - item.CraftedRound
	}
	bottleMult := item.BottleMultiplier
	if bottleMult <= 0 {
		bottleMult = spec.BottleAgingMultiplier
	}
	effectiveSpeed := items.CalcEffectiveAgingSpeed(bottleMult, item.CraftSkill)
	phase, _ := items.GetAgingPhase(elapsed, spec.Aging, effectiveSpeed)
	return phase == items.PhaseDeclining || phase == items.PhaseSpoiled
}
