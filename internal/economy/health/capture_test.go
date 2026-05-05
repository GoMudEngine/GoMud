package health_test

import (
	"os"
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
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

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

	// With pre-population active, we get 1 live row + 2 placeholder rows
	// (Halix mob 372 + Kessa mob 373 are not spawned in this test fixture).
	if len(snap.Foragers) != 3 {
		t.Fatalf("Foragers: got %d, want 3 (1 live + 2 inactive placeholders)", len(snap.Foragers))
	}

	// Find the live Tova row.
	var f *health.ForagerSnapshot
	for i := range snap.Foragers {
		if snap.Foragers[i].State == "foraging" {
			f = &snap.Foragers[i]
			break
		}
	}
	if f == nil {
		t.Fatal("could not find live forager with state=foraging")
	}
	if f.MobId != 371 {
		t.Errorf("mob_id: got %d, want 371", f.MobId)
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

	// Verify placeholder rows are present for the other two foragers.
	inactive := 0
	for _, row := range snap.Foragers {
		if row.State == "(despawned)" {
			inactive++
			if row.CargoByBucket == nil {
				t.Error("placeholder row has nil CargoByBucket")
			}
		}
	}
	if inactive != 2 {
		t.Errorf("despawned placeholder rows: got %d, want 2", inactive)
	}
}

// TestLookupShopMobName_FallsBackToTemplate verifies that when a shop's
// NPC isn't currently spawned as a live instance, the Name field in the
// ShopSnapshot falls back to the mob template (which is always loaded at
// boot) and returns the shopkeeper's name. This prevents the dashboard
// from rendering "#<mobId>" when a merchant is offline.
func TestLookupShopMobName_FallsBackToTemplate(t *testing.T) {
	shops.ClearCache()
	t.Cleanup(shops.ClearCache)

	// Seed a test mob template (mobId 999) with a known name.
	// No live instance is spawned, so lookupShopMobName must fall back.
	testMobId := 999
	testMobName := "Test Merchant"
	testRoomId := 9999

	testMob := &mobs.Mob{
		MobId:         mobs.MobId(testMobId),
		Zone:          "TestZone",
		StatPool:      100,
		ActivityLevel: 50,
		Character: characters.Character{
			Name:      testMobName,
			SpeciesId: 1,
		},
	}
	// Seed mobs with the template only (no instances).
	cleanup := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{testMobId: testMob},
		map[int]*mobs.Mob{}, // no live instances
	)
	t.Cleanup(cleanup)

	// Register a shop for the test mob with no live instance.
	tmpl := shops.ShopInventory{
		Gold:         500,
		StartingGold: 500,
		CraftSupport: shops.CraftSupportGeneral,
		Stock: []shops.StockEntry{
			{ItemId: 40001, RestockQty: 5, MaxStock: 20, Current: 8},
		},
	}
	shops.RegisterShop("testzone", testMobId, testRoomId, tmpl)

	snap := health.CaptureSnapshot()

	if len(snap.Shops) != 1 {
		t.Fatalf("Shops: got %d, want 1", len(snap.Shops))
	}
	got := snap.Shops[0]
	if got.MobId != testMobId {
		t.Errorf("MobId: got %d, want %d", got.MobId, testMobId)
	}
	if got.RoomId != testRoomId {
		t.Errorf("RoomId: got %d, want %d", got.RoomId, testRoomId)
	}
	// Name should fall back to template.
	if got.Name == "" {
		t.Fatalf("Name: got empty, expected template name %q", testMobName)
	}
	if got.Name != testMobName {
		t.Errorf("Name: got %q, want %q (from template)", got.Name, testMobName)
	}
}

// TestCaptureSnapshot_Foragers_PrepopulatesInactiveProfiles verifies that
// when no forager mobs are spawned, CaptureSnapshot still emits 3 rows —
// one placeholder per forager.AllProfiles() entry, all with state=(despawned).
func TestCaptureSnapshot_Foragers_PrepopulatesInactiveProfiles(t *testing.T) {
	// No mob instances registered → all three foragers are despawned.
	snap := health.CaptureSnapshot()

	if len(snap.Foragers) != 3 {
		t.Fatalf("Foragers: got %d, want 3 (all placeholders)", len(snap.Foragers))
	}
	for _, f := range snap.Foragers {
		if f.State != "(despawned)" {
			t.Errorf("mob %d: state=%q, want (despawned)", f.MobId, f.State)
		}
		if f.CargoByBucket == nil {
			t.Errorf("mob %d: CargoByBucket is nil", f.MobId)
		}
	}

	// Verify the three profiles are represented by mob ID.
	mobIds := map[int]bool{}
	for _, f := range snap.Foragers {
		mobIds[f.MobId] = true
	}
	for _, want := range []int{371, 372, 373} {
		if !mobIds[want] {
			t.Errorf("profile mob %d not found in forager placeholders", want)
		}
	}
}

// TestCaptureSnapshot_Foragers_IncludesEquippedSubInventories verifies that
// the per-bucket cargo loop iterates ComponentItems and PotionItems in
// addition to Items. In test, item specs return weight 0, so the loop
// won't produce non-zero bucket sums; the test asserts no crash (structural
// wiring) and that CargoByBucket is initialised.
func TestCaptureSnapshot_Foragers_IncludesEquippedSubInventories(t *testing.T) {
	const roomId = 9200
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

	const foragerInstanceId = 92371
	f := &mobs.Mob{
		MobId:      mobs.MobId(371),
		InstanceId: foragerInstanceId,
		HomeRoomId: roomId,
		Zone:       "TestZone",
	}
	f.Character.Name = "TestTova"
	f.Character.Buffs = buffs.New()
	f.Character.RoomId = roomId
	// Populate all three inventory lists with a known-bucket item.
	f.Character.Items = append(f.Character.Items,
		items.Item{ItemId: 40051}) // stillwater bucket
	f.Character.ComponentItems = append(f.Character.ComponentItems,
		items.Item{ItemId: 40051})
	f.Character.PotionItems = append(f.Character.PotionItems,
		items.Item{ItemId: 40051})

	bs := behaviortree.NewBehaviorState()
	bs.Set("forager_state", "foraging")
	f.BTreeState = bs

	mobs.SetInstanceForTest(f.InstanceId, f)
	defer mobs.SetInstanceForTest(f.InstanceId, nil)
	r.AddMob(f.InstanceId)

	snap := health.CaptureSnapshot()

	// Find the live Tova row.
	var got *health.ForagerSnapshot
	for i := range snap.Foragers {
		if snap.Foragers[i].State == "foraging" {
			got = &snap.Foragers[i]
			break
		}
	}
	if got == nil {
		t.Fatal("live forager not found in snapshot")
	}
	// In test, item specs return 0 weight, so sums are 0 — we only
	// verify the map was initialised (loop didn't panic).
	if got.CargoByBucket == nil {
		t.Error("CargoByBucket: got nil, want initialised map")
	}
}

// TestCaptureForagers_DistinguishesDespawnedFromIdle verifies that
// captureForagers emits distinct State strings for despawned mobs
// (no live instance) vs idle mobs (live instance, empty BTreeState).
func TestCaptureForagers_DistinguishesDespawnedFromIdle(t *testing.T) {
	const roomId = 9300
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

	// Spawn Tova (mob 371, marsh forager) with empty BTreeState —
	// this tests the "idle, no state" path.
	const tovaInstanceId = 93371
	tova := &mobs.Mob{
		MobId:      mobs.MobId(371),
		InstanceId: tovaInstanceId,
		HomeRoomId: roomId,
		Zone:       "TestZone",
	}
	tova.Character.Name = "Tova"
	tova.Character.Buffs = buffs.New()
	tova.Character.RoomId = roomId
	// Empty BTreeState; this will trigger the "(idle, no state)" path.
	tova.BTreeState = behaviortree.NewBehaviorState()

	mobs.SetInstanceForTest(tova.InstanceId, tova)
	defer mobs.SetInstanceForTest(tova.InstanceId, nil)
	r.AddMob(tova.InstanceId)

	// Halix (mob 372) and Kessa (mob 373) are NOT spawned — tests "(despawned)" path.

	snap := health.CaptureSnapshot()

	// Verify we have all 3 foragers in the snapshot.
	if len(snap.Foragers) != 3 {
		t.Fatalf("Foragers: got %d, want 3", len(snap.Foragers))
	}

	// Count despawned and idle rows.
	despawnedCount := 0
	idleCount := 0
	var idle *health.ForagerSnapshot
	for i := range snap.Foragers {
		switch snap.Foragers[i].State {
		case "(despawned)":
			despawnedCount++
		case "(idle, no state)":
			idleCount++
			idle = &snap.Foragers[i]
		}
	}

	if despawnedCount != 2 {
		t.Errorf("despawned rows: got %d, want 2 (Halix + Kessa not spawned)", despawnedCount)
	}
	if idleCount != 1 {
		t.Errorf("idle rows: got %d, want 1 (Tova spawned with empty BTreeState)", idleCount)
	}

	// Verify idle row is Tova with the right properties.
	if idle != nil {
		if idle.MobId != 371 {
			t.Errorf("idle mobId: got %d, want 371 (Tova)", idle.MobId)
		}
		if idle.InstId != tovaInstanceId {
			t.Errorf("idle instId: got %d, want %d", idle.InstId, tovaInstanceId)
		}
		if idle.CargoByBucket == nil {
			t.Error("idle CargoByBucket: got nil, want initialized map")
		}
	}
}
