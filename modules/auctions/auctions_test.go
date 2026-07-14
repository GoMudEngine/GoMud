package auctions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// fakeUsers overrides the auctions package's user lookup with an in-memory set,
// so the manager methods are unit-testable without the full data-file env.
func fakeUsers(recs ...*users.UserRecord) func() {
	m := map[int]*users.UserRecord{}
	for _, u := range recs {
		m[u.UserId] = u
	}
	prev := getUser
	getUser = func(id int) *users.UserRecord { return m[id] }
	return func() { getUser = prev }
}

func TestReserveFrom(t *testing.T) {
	if got := reserveFrom(1000, 0.25); got != 250 {
		t.Errorf("reserveFrom(1000,0.25)=%d want 250", got)
	}
	if got := reserveFrom(2, 0.25); got != 1 { // rounds to 0 -> min 1
		t.Errorf("reserveFrom(2,0.25)=%d want 1", got)
	}
	if got := reserveFrom(0, 0.25); got != 1 {
		t.Errorf("reserveFrom(0,..)=%d want 1", got)
	}
}

func TestStartAuction_DerivesReserve(t *testing.T) {
	seller := users.NewTestUser(9012, "sell2", "Sell2", 9912)
	defer fakeUsers(seller)()

	am := &AuctionManager{}
	if !am.StartAuction(items.Item{ItemId: 1}, 9012, 1000, 60, false) {
		t.Fatal("StartAuction should succeed for a valid user")
	}
	if am.ActiveAuction.BuyoutPrice != 1000 {
		t.Errorf("BuyoutPrice=%d want 1000", am.ActiveAuction.BuyoutPrice)
	}
	if am.ActiveAuction.MinimumBid != 250 {
		t.Errorf("MinimumBid(reserve)=%d want 250", am.ActiveAuction.MinimumBid)
	}
}

func TestBid_EscrowsAndRefunds(t *testing.T) {
	a := users.NewTestUser(9010, "abid", "Abid", 9910)
	b := users.NewTestUser(9011, "bbid", "Bbid", 9911)
	seller := users.NewTestUser(9012, "sell2", "Sell2", 9912)
	a.Character.Bank = 1000
	b.Character.Bank = 1000
	defer fakeUsers(a, b, seller)()

	am := &AuctionManager{}
	am.StartAuction(items.Item{ItemId: 1}, 9012, 1000, 60, false) // buyout 1000, reserve 250

	if err := am.Bid(9010, 100); err == nil {
		t.Fatal("bid below reserve should be rejected")
	}
	a.Character.Bank = 10
	if err := am.Bid(9010, 300); err == nil {
		t.Fatal("bid above bank should be rejected")
	}
	a.Character.Bank = 1000

	if err := am.Bid(9010, 300); err != nil {
		t.Fatalf("valid bid rejected: %v", err)
	}
	if a.Character.Bank != 700 {
		t.Errorf("A bank=%d want 700 (300 escrowed)", a.Character.Bank)
	}

	if err := am.Bid(9011, 400); err != nil {
		t.Fatalf("outbid rejected: %v", err)
	}
	if a.Character.Bank != 1000 {
		t.Errorf("A bank=%d want 1000 (refunded)", a.Character.Bank)
	}
	if b.Character.Bank != 600 {
		t.Errorf("B bank=%d want 600 (400 escrowed)", b.Character.Bank)
	}
}

func TestBid_BuyItNow(t *testing.T) {
	b := users.NewTestUser(9020, "buyer", "Buyer", 9920)
	seller := users.NewTestUser(9021, "sell3", "Sell3", 9921)
	b.Character.Bank = 5000
	defer fakeUsers(b, seller)()

	am := &AuctionManager{}
	am.StartAuction(items.Item{ItemId: 1}, 9021, 1000, 600, false)

	if err := am.Bid(9020, 5000); err != nil {
		t.Fatalf("buyout bid rejected: %v", err)
	}
	if am.ActiveAuction.HighestBid != 1000 {
		t.Errorf("buyout should cap bid at 1000, got %d", am.ActiveAuction.HighestBid)
	}
	if !am.ActiveAuction.IsEnded() {
		t.Error("a buyout bid should end the lot immediately")
	}
	if b.Character.Bank != 4000 {
		t.Errorf("buyer bank=%d want 4000 (1000 escrowed)", b.Character.Bank)
	}
}
