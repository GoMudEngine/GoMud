package auctions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestOfficial_InterestAndMaxBid(t *testing.T) {
	officialEnabled = true
	officialPremium = 1.25
	o := &official{name: "The Crown Assessor", wallet: &NpcWallet{Balance: 25000, Cap: 25000}}

	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		401: {ItemId: 401, Name: "warden core", Type: items.Object, IsComponent: true, Value: 1400, Restricted: true},
		402: {ItemId: 402, Name: "iron ore", Type: items.Object, IsComponent: true, Value: 1400}, // not restricted
		403: {ItemId: 403, Name: "prize blade", Type: items.Weapon, Value: 5000},                 // valuable but not restricted
	})()

	if !o.Interested(items.New(401)) {
		t.Error("official should want a restricted item")
	}
	if o.Interested(items.New(402)) {
		t.Error("official should NOT want a non-restricted component")
	}
	if o.Interested(items.New(403)) {
		t.Error("official should NOT want valuable non-restricted gear")
	}
	if got := o.MaxBid(items.New(401)); got != 1750 {
		t.Errorf("MaxBid=%d want 1750 (Value 1400 * 1.25)", got)
	}
}

func TestOfficial_DisabledDeclinesEverything(t *testing.T) {
	officialEnabled = false
	defer func() { officialEnabled = true }()
	o := &official{name: "The Crown Assessor", wallet: &NpcWallet{Balance: 25000, Cap: 25000}}
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		401: {ItemId: 401, Name: "warden core", Type: items.Object, IsComponent: true, Value: 1400, Restricted: true},
	})()
	if o.Interested(items.New(401)) {
		t.Error("disabled official must not be interested in anything")
	}
}

func TestOfficial_EscrowSeam(t *testing.T) {
	o := &official{name: "The Crown Assessor", wallet: &NpcWallet{Balance: 1000, Cap: 1000}}
	if !o.CanAfford(400) {
		t.Fatal("should afford 400 of 1000")
	}
	o.Spend(400)
	if o.CanAfford(700) {
		t.Error("should not afford 700 after spending 400 (600 left)")
	}
	o.Refund(400)
	if !o.CanAfford(1000) {
		t.Error("refund should restore to 1000")
	}
}

func TestOfficial_RegisteredAndIsSink(t *testing.T) {
	b := buyerByName("The Crown Assessor")
	if b == nil {
		t.Fatal("The Crown Assessor must be in the npcBuyers registry")
	}
	if b.Wallet() == nil {
		t.Error("official must expose a wallet so persistence/regen include it")
	}
	// The official is a SINK: it must NOT implement auctionWinReceiver (only the
	// shopkeeper relists). A sink lets the won item leave circulation.
	if _, ok := b.(auctionWinReceiver); ok {
		t.Error("official must be a sink, not an auctionWinReceiver")
	}
}
