package auctions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestNpcWallet(t *testing.T) {
	w := &NpcWallet{Balance: 100, Cap: 200}
	if !w.CanAfford(100) || w.CanAfford(101) {
		t.Fatal("CanAfford wrong")
	}
	w.Spend(60)
	if w.Balance != 40 {
		t.Fatalf("Spend: balance=%d want 40", w.Balance)
	}
	w.Refund(1000) // clamps to cap
	if w.Balance != 200 {
		t.Fatalf("Refund clamp: balance=%d want 200", w.Balance)
	}
	w.Regen(1000) // clamps to cap
	if w.Balance != 200 {
		t.Fatalf("Regen clamp: balance=%d want 200", w.Balance)
	}
}

func TestCollector_InterestAndMaxBid(t *testing.T) {
	collectorMinValue = 500
	collectorPremium = 1.0
	c := &collector{name: "Veyd", wallet: &NpcWallet{Balance: 10000, Cap: 10000}}

	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		101: {ItemId: 101, Name: "fine blade", Type: items.Weapon, Value: 800},
		102: {ItemId: 102, Name: "cheap dagger", Type: items.Weapon, Value: 100},
		103: {ItemId: 103, Name: "herb", Type: items.Potion, Value: 900},
	})()
	if !c.Interested(items.New(101)) {
		t.Error("collector should want a valuable weapon")
	}
	if c.Interested(items.New(102)) {
		t.Error("collector should NOT want a cheap weapon (below min value)")
	}
	if c.Interested(items.New(103)) {
		t.Error("collector should NOT want a non-equipment potion")
	}
	if got := c.MaxBid(items.New(101)); got != 800 {
		t.Errorf("MaxBid=%d want 800 (Value*1.0)", got)
	}
}
