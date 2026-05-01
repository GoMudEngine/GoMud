package health_test

import (
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/economy/health"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

func TestCaptureSnapshot_Shops(t *testing.T) {
	shops.ClearCache()
	t.Cleanup(shops.ClearCache)

	tmpl := shops.ShopInventory{
		Gold:         500,
		StartingGold: 500,
		CraftSupport: shops.CraftSupportGeneral,
		Stock: []shops.StockEntry{
			{ItemId: 40001, RestockQty: 5, MaxStock: 20, Current: 8}, // base
			{ItemId: 40051, RestockQty: 3, MaxStock: 10, Current: 4}, // stillwater
		},
	}
	shops.RegisterShop("stillwater", 341, 4105, tmpl)

	snap := health.CaptureSnapshot()

	if len(snap.Shops) != 1 {
		t.Fatalf("Shops: got %d, want 1", len(snap.Shops))
	}
	got := snap.Shops[0]
	if got.MobId != 341 || got.RoomId != 4105 {
		t.Errorf("location: got %d/%d, want 341/4105", got.MobId, got.RoomId)
	}
	if got.CraftSupport != "general" {
		t.Errorf("craft_support: got %q, want general", got.CraftSupport)
	}
	if len(got.Stock) != 2 {
		t.Fatalf("stock entries: got %d, want 2", len(got.Stock))
	}
	if got.Stock[0].Bucket != "base" {
		t.Errorf("first stock bucket: got %q, want base", got.Stock[0].Bucket)
	}
	if got.Stock[1].Bucket != "stillwater" {
		t.Errorf("second stock bucket: got %q, want stillwater", got.Stock[1].Bucket)
	}
}

func TestCaptureSnapshot_Caravans(t *testing.T) {
	const roomId = 9000

	r := &rooms.Room{
		RoomId: roomId,
		Zone:   "TestZone",
		Title:  "Test Room",
		Exits:  map[string]exit.RoomExit{},
	}
	cleanRoom := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{roomId: r},
		map[string]*rooms.ZoneConfig{},
	)
	defer cleanRoom()

	// Build the wagon mob literal (same pattern as wagon_test.go).
	wagon := &mobs.Mob{
		MobId:      mobs.MobId(caravan.WagonMobId),
		InstanceId: 90001,
		HomeRoomId: roomId,
		Zone:       "TestZone",
	}
	wagon.Character.Name = "TestWagon"
	wagon.Character.Buffs = buffs.New()
	wagon.Character.RoomId = roomId
	// Stuff one base-bucket item (iron ingot 40001) into the wagon's
	// inventory directly. items.New returns ItemId=0 when the item spec
	// is not loaded in test, so construct the Item literal instead.
	wagon.Character.Items = append(wagon.Character.Items, items.Item{ItemId: 40001})

	mobs.SetInstanceForTest(wagon.InstanceId, wagon)
	defer mobs.SetInstanceForTest(wagon.InstanceId, nil)
	r.AddMob(wagon.InstanceId)

	// Build a leader mob literal with caravan_state stamped on BTreeState.
	const leaderInstanceId = 90002
	leader := &mobs.Mob{
		MobId:      mobs.MobId(357),
		InstanceId: leaderInstanceId,
		HomeRoomId: roomId,
		Zone:       "TestZone",
	}
	leader.Character.Name = "TestLeader"
	leader.Character.Buffs = buffs.New()
	leader.Character.RoomId = roomId

	bs := behaviortree.NewBehaviorState()
	bs.Set("caravan_state", "outbound_transit")
	bs.Set("caravan_state_started_round", strconv.FormatUint(12100, 10))
	leader.BTreeState = bs

	mobs.SetInstanceForTest(leader.InstanceId, leader)
	defer mobs.SetInstanceForTest(leader.InstanceId, nil)
	r.AddMob(leader.InstanceId)

	snap := health.CaptureSnapshot()

	if len(snap.Caravans) != 1 {
		t.Fatalf("Caravans: got %d, want 1", len(snap.Caravans))
	}
	c := snap.Caravans[0]
	if c.State != "outbound_transit" {
		t.Errorf("state: got %q, want outbound_transit", c.State)
	}
	if c.StateEnteredRound != 12100 {
		t.Errorf("state_entered_round: got %d, want 12100", c.StateEnteredRound)
	}
	if c.CargoCount != 1 {
		t.Errorf("cargo_count: got %d, want 1", c.CargoCount)
	}
	if c.CargoByBucket["base"] != 1 {
		t.Errorf("cargo_by_bucket[base]: got %d, want 1", c.CargoByBucket["base"])
	}
}
