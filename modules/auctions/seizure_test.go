package auctions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"gopkg.in/yaml.v2"
)

func TestStorageSeizedHandler_Enqueues(t *testing.T) {
	mod := &AuctionsModule{auctionMgr: AuctionManager{}}
	mod.storageSeizedHandler(events.StorageItemSeized{
		UserId: 11,
		Item:   items.Item{ItemId: 5001},
		Count:  3,
		Owed:   1,
	})
	if len(mod.auctionMgr.SeizedQueue) != 1 {
		t.Fatalf("SeizedQueue len = %d, want 1", len(mod.auctionMgr.SeizedQueue))
	}
	lot := mod.auctionMgr.SeizedQueue[0]
	if lot.ExOwnerUserId != 11 || lot.Item.ItemId != 5001 || lot.Count != 3 || lot.Owed != 1 {
		t.Errorf("enqueued lot = %+v, want {5001, count3, owner11, owed1}", lot)
	}
}

func TestSeizedQueue_YAMLRoundTrip(t *testing.T) {
	mgr := AuctionManager{SeizedQueue: []SeizedLot{
		{Item: items.Item{ItemId: 5001}, Count: 2, ExOwnerUserId: 9, Owed: 1},
	}}
	out, err := yaml.Marshal(mgr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back AuctionManager
	if err := yaml.Unmarshal(out, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back.SeizedQueue) != 1 || back.SeizedQueue[0].Item.ItemId != 5001 || back.SeizedQueue[0].Count != 2 {
		t.Errorf("round-trip lost SeizedQueue: %+v", back.SeizedQueue)
	}
}

func TestDrainSeizedQueue_ListsFrontLot(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		5001: {ItemId: 5001, Name: "gilded blade", Type: items.Weapon, Value: 1000},
	})
	defer cleanup()

	// Ex-owner offline: getUser returns nil for everyone; drain must still list.
	defer fakeUsers()()

	mod := &AuctionsModule{
		// drainSeizedQueue takes the duration as a param and never touches plug,
		// so a nil plug is fine here.
		auctionMgr: AuctionManager{SeizedQueue: []SeizedLot{{Item: items.New(5001), Count: 1, ExOwnerUserId: 77, Owed: 1}}},
	}
	mod.drainSeizedQueue(120)

	a := mod.auctionMgr.GetCurrentAuction()
	if a == nil {
		t.Fatal("drain did not list a lot")
	}
	if !a.Seized || a.OwedLien != 1 || a.SeizedCount != 1 {
		t.Errorf("lot fields = seized:%v owed:%d count:%d, want true/1/1", a.Seized, a.OwedLien, a.SeizedCount)
	}
	if a.SellerUserId != 77 || !a.Anonymous {
		t.Errorf("seller=%d anon=%v, want 77/true", a.SellerUserId, a.Anonymous)
	}
	if a.BuyoutPrice != 1000 {
		t.Errorf("buyout=%d, want 1000 (spec.Value*Count)", a.BuyoutPrice)
	}
	if a.MinimumBid != reserveFrom(1000, auctionReservePct) {
		t.Errorf("reserve=%d, want %d", a.MinimumBid, reserveFrom(1000, auctionReservePct))
	}
	if len(mod.auctionMgr.SeizedQueue) != 0 {
		t.Errorf("queue not drained: %d left", len(mod.auctionMgr.SeizedQueue))
	}
}

func TestDrainSeizedQueue_NoopWhenBusy(t *testing.T) {
	mod := &AuctionsModule{auctionMgr: AuctionManager{
		ActiveAuction: &AuctionItem{ItemData: items.Item{ItemId: 1}},
		SeizedQueue:   []SeizedLot{{Item: items.Item{ItemId: 5001}, Count: 1, ExOwnerUserId: 1, Owed: 1}},
	}}
	mod.drainSeizedQueue(120)
	if len(mod.auctionMgr.SeizedQueue) != 1 {
		t.Errorf("busy block should not drain; queue=%d", len(mod.auctionMgr.SeizedQueue))
	}
}

func TestResolveSeizedLot_PlayerWin_LienAndSurplus(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		5001: {ItemId: 5001, Name: "gilded blade", Type: items.Weapon, Value: 1000},
	})
	defer cleanup()

	winner := &users.UserRecord{UserId: 88, Character: &characters.Character{}}
	winner.Character.Name = "Winner"
	exOwner := &users.UserRecord{UserId: 77, Character: &characters.Character{}}
	exOwner.Character.Name = "ExOwner"
	defer fakeUsers(winner, exOwner)()

	mod := &AuctionsModule{auctionMgr: AuctionManager{}}
	// Winner already escrowed 400 at bid time; commission 5% -> afterCommission 380.
	// Lien 1 -> surplus 379 to ex-owner. Winner gets 1 unit.
	mod.auctionMgr.ActiveAuction = &AuctionItem{
		ItemData:          items.New(5001),
		SellerUserId:      77,
		Anonymous:         true,
		HighestBid:        400,
		HighestBidUserId:  88,
		HighestBidderName: "Winner",
		Seized:            true,
		OwedLien:          1,
		SeizedCount:       1,
	}

	mod.resolveSeizedLot(mod.auctionMgr.ActiveAuction)

	wantSurplus := 400 - commissionFor(400, auctionCommissionPct) - 1
	if exOwner.Character.Bank != wantSurplus {
		t.Errorf("ex-owner bank=%d, want %d (surplus after commission+lien)", exOwner.Character.Bank, wantSurplus)
	}
	// StoreItem adds to the winner's BACKPACK (c.Items), not the bank (ItemStorage).
	// Weightless seeded items pass the zero-stat carry-capacity gate.
	if len(winner.Character.Items) != 1 {
		t.Errorf("winner backpack items=%d, want 1", len(winner.Character.Items))
	}
}

func TestResolveSeizedLot_StackDeliversAllUnits(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		5003: {ItemId: 5003, Name: "silk bolt", Type: items.Object, Value: 50},
	})
	defer cleanup()

	winner := &users.UserRecord{UserId: 88, Character: &characters.Character{}}
	defer fakeUsers(winner)() // ex-owner (77) offline

	mod := &AuctionsModule{auctionMgr: AuctionManager{}}
	mod.auctionMgr.ActiveAuction = &AuctionItem{
		ItemData:         items.New(5003),
		SellerUserId:     77,
		HighestBid:       300,
		HighestBidUserId: 88,
		Seized:           true,
		OwedLien:         1,
		SeizedCount:      6,
	}
	mod.resolveSeizedLot(mod.auctionMgr.ActiveAuction)

	// All 6 units delivered to the backpack (c.Items). Plain Object (no IsComponent)
	// so nothing auto-routes to a component bag; 6 StoreItem calls -> 6 backpack entries.
	if len(winner.Character.Items) != 6 {
		t.Errorf("winner received %d units, want 6", len(winner.Character.Items))
	}
}

func TestResolveSeizedLot_Unsold_Disposed(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		5001: {ItemId: 5001, Name: "gilded blade", Type: items.Weapon, Value: 1000},
	})
	defer cleanup()

	exOwner := &users.UserRecord{UserId: 77, Character: &characters.Character{}}
	defer fakeUsers(exOwner)()

	mod := &AuctionsModule{auctionMgr: AuctionManager{}}
	mod.auctionMgr.ActiveAuction = &AuctionItem{
		ItemData:         items.New(5001),
		SellerUserId:     77,
		HighestBid:       0,
		HighestBidUserId: 0,
		HighestBidIsNPC:  false, // no bids at all
		Seized:           true,
		OwedLien:         1,
		SeizedCount:      1,
	}
	mod.resolveSeizedLot(mod.auctionMgr.ActiveAuction)

	// Disposed: NOT returned to ex-owner storage.
	if exOwner.ItemStorage.SlotCount() != 0 {
		t.Errorf("unsold seized lot must NOT return to storage; slots=%d", exOwner.ItemStorage.SlotCount())
	}
	// Ex-owner got a disposal notice.
	if len(exOwner.Inbox) == 0 {
		t.Error("expected a disposal inbox notice")
	}
}

func TestResolveSeizedLot_SaleBelowLien_NoNegativeCredit(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		5001: {ItemId: 5001, Name: "gilded blade", Type: items.Weapon, Value: 1000},
	})
	defer cleanup()

	winner := &users.UserRecord{UserId: 88, Character: &characters.Character{}}
	exOwner := &users.UserRecord{UserId: 77, Character: &characters.Character{}}
	defer fakeUsers(winner, exOwner)()

	mod := &AuctionsModule{auctionMgr: AuctionManager{}}
	// Sale of 5 with a huge lien (10) -> afterCommission <= 10 -> lien capped, surplus 0.
	mod.auctionMgr.ActiveAuction = &AuctionItem{
		ItemData:         items.New(5001),
		SellerUserId:     77,
		HighestBid:       5,
		HighestBidUserId: 88,
		Seized:           true,
		OwedLien:         10,
		SeizedCount:      1,
	}
	mod.resolveSeizedLot(mod.auctionMgr.ActiveAuction)

	if exOwner.Character.Bank != 0 {
		t.Errorf("ex-owner bank=%d, want 0 (sale below lien -> no surplus, no negative credit)", exOwner.Character.Bank)
	}
}

func TestResolveSeizedLot_NpcWin_LienSettlesToExOwner(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		5001: {ItemId: 5001, Name: "gilded blade", Type: items.Weapon, Value: 1000},
	})
	defer cleanup()

	exOwner := &users.UserRecord{UserId: 77, Character: &characters.Character{}}
	defer fakeUsers(exOwner)()

	mod := &AuctionsModule{auctionMgr: AuctionManager{}}
	// NPC winner (name not registered -> buyerByName nil -> no sink, but lien settles).
	mod.auctionMgr.ActiveAuction = &AuctionItem{
		ItemData:          items.New(5001),
		SellerUserId:      77,
		HighestBid:        400,
		HighestBidUserId:  0,
		HighestBidIsNPC:   true,
		HighestBidderName: "Nonexistent Collector",
		Seized:            true,
		OwedLien:          1,
		SeizedCount:       1,
	}
	mod.resolveSeizedLot(mod.auctionMgr.ActiveAuction)

	wantSurplus := 400 - commissionFor(400, auctionCommissionPct) - 1
	if exOwner.Character.Bank != wantSurplus {
		t.Errorf("ex-owner bank=%d, want %d (surplus settles even on an NPC win)", exOwner.Character.Bank, wantSurplus)
	}
}
