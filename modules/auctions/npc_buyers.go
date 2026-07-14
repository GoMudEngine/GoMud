package auctions

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/uuid"
)

// NpcBuyer is one living-world auction buyer archetype.
type NpcBuyer interface {
	Name() string
	Interested(item items.Item) bool
	MaxBid(item items.Item) int
	CanAfford(n int) bool // escrow seam: does the buyer's purse cover n?
	Spend(n int)          // escrow seam: debit the purse
	Refund(n int)         // escrow seam: credit the purse back (on outbid)
	Wallet() *NpcWallet   // synthetic regen wallet for persistence/regen; nil for real-gold buyers
	Flavor() string       // trailing phrase in the win broadcast, e.g. "for their collection"
}

// NpcWallet is a persistent, gold-gated balance that regenerates toward Cap.
type NpcWallet struct {
	Balance int `json:"balance"`
	Cap     int `json:"cap"`
}

func (w *NpcWallet) CanAfford(n int) bool { return w.Balance >= n }
func (w *NpcWallet) Spend(n int)          { w.Balance -= n }
func (w *NpcWallet) Refund(n int)         { w.add(n) }
func (w *NpcWallet) Regen(amount int)     { w.add(amount) }
func (w *NpcWallet) add(n int) {
	w.Balance += n
	if w.Balance > w.Cap {
		w.Balance = w.Cap
	}
}

// ── Provisional knobs (tuned in playtest; overridden from config in load) ──
var (
	npcBuyersEnabled      = true
	npcBidChancePct       = 35 // % chance per update tick an interested NPC nudges the bid
	collectorMinValue     = 500
	collectorPremium      = 1.0
	collectorRegenPerTick = 5 // gold per update tick (config sets the real rate)

	craftMinValue = 50 // materials are cheaper than gear
	craftPremium  = 1.0
	advMinValue   = 300
	advPremium    = 0.9 // an adventurer haggles a bit

	shopkeeperEnabled = true // gated by AuctionShopkeeperEnabled config

	// saveShopFn persists a shop after the shopkeeper mutates its gold/stock.
	// Overridable in tests to avoid disk I/O.
	saveShopFn = shops.SaveShop
)

// isEquipment reports whether an item type is wearable/wieldable gear.
func isEquipment(t items.ItemType) bool {
	switch t {
	case items.Weapon, items.Offhand, items.Head, items.Neck, items.Body,
		items.Belt, items.Gloves, items.Ring, items.Wrist, items.Legs,
		items.Feet, items.Back, items.Shoulders:
		return true
	}
	return false
}

// ── Collector archetype ──
type collector struct {
	name   string
	wallet *NpcWallet
}

func (c *collector) Name() string { return c.name }
func (c *collector) Interested(item items.Item) bool {
	spec := item.GetSpec()
	return isEquipment(spec.Type) && spec.Value >= collectorMinValue
}
func (c *collector) MaxBid(item items.Item) int {
	return int(float64(item.GetSpec().Value) * collectorPremium)
}
func (c *collector) CanAfford(n int) bool { return c.wallet.CanAfford(n) }
func (c *collector) Spend(n int)          { c.wallet.Spend(n) }
func (c *collector) Refund(n int)         { c.wallet.Refund(n) }
func (c *collector) Wallet() *NpcWallet   { return c.wallet }
func (c *collector) Flavor() string       { return "for their collection" }

// ── Craftsperson archetype: buys valuable crafting materials ──
type craftsperson struct {
	name   string
	wallet *NpcWallet
}

func (c *craftsperson) Name() string { return c.name }
func (c *craftsperson) Interested(item items.Item) bool {
	spec := item.GetSpec()
	return spec.IsComponent && spec.Value >= craftMinValue
}
func (c *craftsperson) MaxBid(item items.Item) int {
	return int(float64(item.GetSpec().Value) * craftPremium)
}
func (c *craftsperson) CanAfford(n int) bool { return c.wallet.CanAfford(n) }
func (c *craftsperson) Spend(n int)          { c.wallet.Spend(n) }
func (c *craftsperson) Refund(n int)         { c.wallet.Refund(n) }
func (c *craftsperson) Wallet() *NpcWallet   { return c.wallet }
func (c *craftsperson) Flavor() string       { return "for their workshop" }

// ── Adventurer archetype: buys usable gear upgrades (stat-bearing equipment) ──
type adventurer struct {
	name   string
	wallet *NpcWallet
}

func (a *adventurer) Name() string { return a.name }
func (a *adventurer) Interested(item items.Item) bool {
	spec := item.GetSpec()
	return isEquipment(spec.Type) && len(spec.StatMods) > 0 && spec.Value >= advMinValue
}
func (a *adventurer) MaxBid(item items.Item) int {
	return int(float64(item.GetSpec().Value) * advPremium)
}
func (a *adventurer) CanAfford(n int) bool { return a.wallet.CanAfford(n) }
func (a *adventurer) Spend(n int)          { a.wallet.Spend(n) }
func (a *adventurer) Refund(n int)         { a.wallet.Refund(n) }
func (a *adventurer) Wallet() *NpcWallet   { return a.wallet }
func (a *adventurer) Flavor() string       { return "to gear up" }

// reserveRatio returns the shop gold-reserve fraction from config (0.50 fallback).
func reserveRatio() float64 {
	r := float64(configs.GetBalanceConfig().ShopGoldReserveRatio)
	if r <= 0 {
		r = 0.50
	}
	return r
}

func persistShop(inv *shops.ShopInventory) {
	if err := saveShopFn(inv.Zone, inv.MobId, inv.RoomId); err != nil {
		mudlog.Error("auctions.shopkeeper", "msg", "SaveShop failed", "error", err)
	}
}

// ── Shopkeeper archetype: bids from a REAL shop's gold on items that shop
// deals in (VendorCategories ↔ CraftSupport), then relists the win into that
// shop's stock. Thin adapter over shops.EvaluateBuyRules. ──

// shopSel is the shopkeeper's memoized per-item shop selection.
type shopSel struct {
	uuid  uuid.UUID
	shop  *shops.ShopInventory
	offer int // best affordable BuyOffer.Price across live shops (0 = none)
}

type shopkeeper struct {
	name  string
	sel   shopSel              // memoized selection for the current decision
	bound *shops.ShopInventory // shop escrowed against while high bidder
}

func (s *shopkeeper) Name() string       { return s.name }
func (s *shopkeeper) Flavor() string     { return "for the shelves" }
func (s *shopkeeper) Wallet() *NpcWallet { return nil } // real-gold buyer

// selectFor picks the shop with the highest affordable counter-offer for item.
// Memoized by item UUID: one shops.AllShops() scan per lot (the item is stable
// for a lot's lifetime), reused across the Interested→MaxBid→CanAfford calls
// nextNpcBid makes on the single-threaded auction tick. CanAfford still re-reads
// the shop's live Gold, so a stale offer can never cause an overspend.
func (s *shopkeeper) selectFor(item items.Item) shopSel {
	if !item.UUID.IsNil() && s.sel.uuid == item.UUID {
		return s.sel
	}
	cfg := shops.PricingConfigFromBalance()
	best := shopSel{uuid: item.UUID}
	for _, inv := range shops.AllShops() {
		off := shops.EvaluateBuyRules(item, inv, "", false, cfg, nil)
		if off.Price > best.offer {
			best.shop, best.offer = inv, off.Price
		}
	}
	s.sel = best
	return best
}

func (s *shopkeeper) Interested(item items.Item) bool {
	if !shopkeeperEnabled {
		return false
	}
	return s.selectFor(item).offer > 0
}

func (s *shopkeeper) MaxBid(item items.Item) int { return s.selectFor(item).offer }

func (s *shopkeeper) CanAfford(n int) bool {
	if s.sel.shop == nil {
		return false
	}
	return s.sel.shop.CanAfford(n, s.sel.shop.GoldReserve(reserveRatio()))
}

func (s *shopkeeper) Spend(n int) {
	s.bound = s.sel.shop // freeze the binding for refund/win
	if s.bound == nil {
		return
	}
	s.bound.Gold -= n
	persistShop(s.bound)
}

func (s *shopkeeper) Refund(n int) {
	if s.bound == nil {
		return
	}
	s.bound.Gold += n
	persistShop(s.bound)
}

// shopBinder is implemented by NPC buyers escrowed against a real shop, so the
// auction can persist which shop was debited and restore it after a restart.
type shopBinder interface {
	BoundShop() (zone string, mobId, roomId int, ok bool)
	RestoreBoundShop(zone string, mobId, roomId int)
}

func (s *shopkeeper) BoundShop() (zone string, mobId, roomId int, ok bool) {
	if s.bound == nil {
		return "", 0, 0, false
	}
	return s.bound.Zone, s.bound.MobId, s.bound.RoomId, true
}

func (s *shopkeeper) RestoreBoundShop(zone string, mobId, roomId int) {
	s.bound = shops.GetShopInventory(zone, mobId, roomId)
}

// auctionWinReceiver lets a buyer take custody of the item it won (instead of
// the default sink). Only the shopkeeper implements it.
type auctionWinReceiver interface {
	Receive(item items.Item)
}

// Receive routes a won lot into the bound shop's resale stock. AddAffixedStock
// holds the full item instance, so exact affixes/enchants survive and the item
// becomes purchasable — mirroring the counter-buyback path in actions/sell.go.
func (s *shopkeeper) Receive(item items.Item) {
	if s.bound == nil {
		return
	}
	c := int(configs.GetBalanceConfig().ShopAffixedStockCap)
	s.bound.AddAffixedStock(item, item.GetSpec().Value, c)
	s.bound.BuysCount++
	persistShop(s.bound)
	s.bound = nil
}

// ── Registry of active NPC buyers (#2.1: two collectors) ──
var npcBuyers = []NpcBuyer{
	&collector{name: "Collector Veyd", wallet: &NpcWallet{Balance: 10000, Cap: 10000}},
	&collector{name: "Lady Ashcombe", wallet: &NpcWallet{Balance: 10000, Cap: 10000}},
	&craftsperson{name: "Master Ordwin", wallet: &NpcWallet{Balance: 6000, Cap: 6000}},
	&adventurer{name: "Sellsword Kest", wallet: &NpcWallet{Balance: 6000, Cap: 6000}},
	&shopkeeper{name: "The Merchants' Guild"},
}

func buyerByName(name string) NpcBuyer {
	for _, b := range npcBuyers {
		if b.Name() == name {
			return b
		}
	}
	return nil
}

// nextNpcBid picks the first enabled buyer that should place a competing bid on
// the current lot, and the amount (one increment above the current high bid).
// Pure over the buyer set + current bid state; the caller applies the
// probability gate. Returns ok=false if no buyer should bid.
func nextNpcBid(buyers []NpcBuyer, item items.Item, highBid, minBid int, highName string, highIsNPC bool) (NpcBuyer, int, bool) {
	next := minBid
	if highBid > 0 {
		next = highBid + 1
	}
	for _, b := range buyers {
		if highIsNPC && b.Name() == highName {
			continue // already the high bidder
		}
		if !b.Interested(item) {
			continue
		}
		if next > b.MaxBid(item) {
			continue
		}
		if !b.CanAfford(next) {
			continue
		}
		return b, next, true
	}
	return nil, 0, false
}
