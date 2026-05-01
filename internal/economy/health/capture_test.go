package health_test

import (
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/characters"
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
	// Override carry capacity to a known value so CargoCapacity in the
	// snapshot is deterministic. Item specs aren't loaded in test, so
	// per-item weights resolve to 0; we only assert the structural
	// wiring (capacity populated, cargo fields present).
	characters.ApplyMobOverrides(&wagon.Character, 0, 0, 5000)
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
	if c.CargoCapacity != 5000 {
		t.Errorf("cargo_capacity: got %d, want 5000 (override set in fixture)", c.CargoCapacity)
	}
	// Item specs aren't loaded in test, so per-item weight resolves
	// to 0 and CargoWeight + CargoByBucket sums are 0. We only assert
	// the structural wiring is present (capacity populated, map
	// non-nil), not specific weight values.
	if c.CargoByBucket == nil {
		t.Error("cargo_by_bucket: got nil map, want initialized map")
	}
}

func TestCaptureSnapshot_Foragers(t *testing.T) {
	const roomId = 9100

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

	// Marsh forager (Tova, mob 371). forager.ProfileFor(371) returns
	// KindMarsh, which territoryFor() translates to "stillwater_marsh".
	const marshForagerMobId = 371
	const foragerInstanceId = 91371
	forager := &mobs.Mob{
		MobId:      mobs.MobId(marshForagerMobId),
		InstanceId: foragerInstanceId,
		HomeRoomId: roomId,
		Zone:       "TestZone",
	}
	forager.Character.Name = "TestTova"
	forager.Character.Buffs = buffs.New()
	forager.Character.RoomId = roomId
	characters.ApplyMobOverrides(&forager.Character, 0, 0, 60) // 60lb capacity
	forager.Character.Items = append(forager.Character.Items, items.Item{ItemId: 40051}) // skitter-shrimp shell, "stillwater"

	bs := behaviortree.NewBehaviorState()
	bs.Set("forager_state", "foraging")
	bs.Set("forager_state_started_round", strconv.FormatUint(12200, 10))
	forager.BTreeState = bs

	mobs.SetInstanceForTest(forager.InstanceId, forager)
	defer mobs.SetInstanceForTest(forager.InstanceId, nil)
	r.AddMob(forager.InstanceId)

	snap := health.CaptureSnapshot()

	if len(snap.Foragers) != 1 {
		t.Fatalf("Foragers: got %d, want 1", len(snap.Foragers))
	}
	f := snap.Foragers[0]
	if f.State != "foraging" {
		t.Errorf("state: got %q, want foraging", f.State)
	}
	if f.Territory != "stillwater_marsh" {
		t.Errorf("territory: got %q, want stillwater_marsh (derived from KindMarsh)", f.Territory)
	}
	if f.StateEnteredRound != 12200 {
		t.Errorf("state_entered_round: got %d, want 12200", f.StateEnteredRound)
	}
	if f.CargoCapacity != 60 {
		t.Errorf("cargo_capacity: got %d, want 60 (override set in fixture)", f.CargoCapacity)
	}
	// Item specs aren't loaded in test, so per-item weights resolve to
	// 0 — only verify structural wiring (CargoByBucket non-nil).
	if f.CargoByBucket == nil {
		t.Error("cargo_by_bucket: got nil map, want initialized map")
	}
}
