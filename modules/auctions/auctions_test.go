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
