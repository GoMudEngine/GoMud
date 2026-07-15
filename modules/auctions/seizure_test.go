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
