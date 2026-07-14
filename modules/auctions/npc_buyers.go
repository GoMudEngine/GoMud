package auctions

import "github.com/GoMudEngine/GoMud/internal/items"

// NpcBuyer is one living-world auction buyer archetype.
type NpcBuyer interface {
	Name() string
	Interested(item items.Item) bool
	MaxBid(item items.Item) int
	Wallet() *NpcWallet
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
func (c *collector) Wallet() *NpcWallet { return c.wallet }

// ── Registry of active NPC buyers (#2.1: two collectors) ──
var npcBuyers = []NpcBuyer{
	&collector{name: "Collector Veyd", wallet: &NpcWallet{Balance: 10000, Cap: 10000}},
	&collector{name: "Lady Ashcombe", wallet: &NpcWallet{Balance: 10000, Cap: 10000}},
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
		if !b.Wallet().CanAfford(next) {
			continue
		}
		return b, next, true
	}
	return nil, 0, false
}
