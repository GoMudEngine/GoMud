package auctions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
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
