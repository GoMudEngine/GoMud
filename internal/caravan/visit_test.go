package caravan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)

	// Redirect FilePaths.DataFiles to an OS-temp subdir so any
	// shops.SaveShop / SaveThroughput calls triggered by the pickup
	// loop write to a throwaway location instead of polluting the
	// repo at _datafiles/world/default/shops/.
	tmpRoot := filepath.Join(os.TempDir(), "gomud-caravan-test-datafiles")
	_ = os.RemoveAll(tmpRoot)
	_ = os.MkdirAll(tmpRoot, 0755)
	if err := configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles": tmpRoot,
	}); err != nil {
		panic(err)
	}

	code := m.Run()
	_ = os.RemoveAll(tmpRoot)
	os.Exit(code)
}

// ─── VisitVendorsInRoom ──────────────────────────────────────────────────────

func TestVisitVendorsInRoom_NoRoomReturnsNil(t *testing.T) {
	// Use a roomid that's guaranteed not to exist.
	delivered, pickedUp := VisitVendorsInRoom(99999999, &mobs.Mob{}, []string{"stillwater"}, nil)
	if delivered != nil || pickedUp != nil {
		t.Errorf("VisitVendorsInRoom(missing) = (%v, %v), want both nil", delivered, pickedUp)
	}
}

func TestVisitVendorsInRoom_NilWagonReturnsNil(t *testing.T) {
	delivered, pickedUp := VisitVendorsInRoom(7777, nil, []string{"stillwater"}, nil)
	if delivered != nil || pickedUp != nil {
		t.Errorf("VisitVendorsInRoom(nil wagon) = (%v, %v), want both nil", delivered, pickedUp)
	}
}

func TestVisitVendorsInRoom_NoShopMobsReturnsNil(t *testing.T) {
	cleanup := seedTestRoomWithMobs(t, 7777, "TestZone", []mobs.MobId{1})
	defer cleanup()

	delivered, pickedUp := VisitVendorsInRoom(7777, &mobs.Mob{}, []string{"stillwater"}, nil)
	if delivered != nil || pickedUp != nil {
		t.Errorf("VisitVendorsInRoom = (%v, %v), want both nil for room with no shop mobs",
			delivered, pickedUp)
	}
}

// TestVisitVendorsInRoom_ShopMobNoInventory verifies that a shop-bearing
// mob whose shop inventory is not registered (no save file, no cache) is
// silently skipped, so VisitVendorsInRoom returns (nil, nil) rather than
// crashing. This is the expected behavior per the plan note: in tests,
// shops.GetShopInventory returns nil because no inventory is registered
// through the normal load path.
func TestVisitVendorsInRoom_ShopMobNoInventory(t *testing.T) {
	mob := buildShopBearingTestMob(t, "TestShopper")
	cleanup := seedTestRoomWithExistingMobs(t, 7778, "TestZone", []*mobs.Mob{mob})
	defer cleanup()

	delivered, pickedUp := VisitVendorsInRoom(7778, &mobs.Mob{}, []string{"stillwater"}, nil)
	if delivered != nil || pickedUp != nil {
		t.Errorf("VisitVendorsInRoom = (%v, %v), want both nil (shop mob present but no inventory)",
			delivered, pickedUp)
	}
}

// ─── FormatVisitMessage ──────────────────────────────────────────────────────

func TestFormatVisitMessage_EmptyReturnsEmpty(t *testing.T) {
	if got := FormatVisitMessage("Lars", nil, nil); got != "" {
		t.Errorf("FormatVisitMessage(nil, nil) = %q, want empty", got)
	}
	if got := FormatVisitMessage("Lars", []ItemMove{}, []ItemMove{}); got != "" {
		t.Errorf("FormatVisitMessage(empty, empty) = %q, want empty", got)
	}
}

func TestFormatVisitMessage_DeliveryOnly(t *testing.T) {
	delivered := []ItemMove{{Vendor: "Brindle", ItemName: "iron ingot"}}
	got := FormatVisitMessage("Lars", delivered, nil)
	if got == "" {
		t.Error("FormatVisitMessage(delivery, nil) returned empty")
	}
	if !contains(got, "unloads") {
		t.Errorf("delivery-only message %q does not match expected wording", got)
	}
	if !contains(got, "Lars") {
		t.Errorf("delivery-only message %q does not name the runner", got)
	}
	if !contains(got, "Brindle") {
		t.Errorf("delivery-only message %q does not name the vendor", got)
	}
}

func TestFormatVisitMessage_PickupOnly(t *testing.T) {
	pickedUp := []ItemMove{{Vendor: "Brindle", ItemName: "lake-iron nodule"}}
	got := FormatVisitMessage("Lars", nil, pickedUp)
	if got == "" {
		t.Error("FormatVisitMessage(nil, pickup) returned empty")
	}
	if !contains(got, "loads up") {
		t.Errorf("pickup-only message %q does not match expected wording", got)
	}
	if !contains(got, "Lars") {
		t.Errorf("pickup-only message %q does not name the runner", got)
	}
}

func TestFormatVisitMessage_BothDeliveryAndPickup(t *testing.T) {
	delivered := []ItemMove{{Vendor: "Brindle", ItemName: "steel"}}
	pickedUp := []ItemMove{{Vendor: "Brindle", ItemName: "lake-iron"}}
	got := FormatVisitMessage("Lars", delivered, pickedUp)
	if got == "" {
		t.Error("FormatVisitMessage(both) returned empty")
	}
	if !contains(got, "trade") {
		t.Errorf("both message %q does not match expected 'in trade' wording", got)
	}
	if !contains(got, "Lars") {
		t.Errorf("both message %q does not name the runner", got)
	}
}

// contains is a simple substring check to avoid importing strings in test
// output assertions.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}()
}

// ─── Pickup-filter expansion (3.8 hotfix H2) ─────────────────────────────────

// TestVisitVendorsInRoom_PicksUpComponentsBeyondZoneBucket verifies the H2
// hotfix: items flagged is_component=true should be picked up by the caravan
// pickup pass even when their economy bucket is NOT in pickupBuckets, as long
// as their ItemType is a raw-material type (object) rather than a finished
// good. Models real-world steel/iron ingots at Smith Brindle — bucket "base"
// or "thornwall", but pickupBuckets is ["stillwater"]. Pre-fix: 0 picked up.
func TestVisitVendorsInRoom_PicksUpComponentsBeyondZoneBucket(t *testing.T) {
	cleanup := setupPickupTestFixtures(t, []shops.StockEntry{
		// Iron ingot: bucket "base", is_component=true, type=object.
		// 80 current, RestockQty 5 -> qty = max(1, 80/10) = 8.
		{ItemId: 40001, RestockQty: 5, MaxStock: 100, Current: 80},
	}, map[int]*items.ItemSpec{
		40001: {
			ItemId:      40001,
			Name:        "iron ingot",
			Type:        items.Object,
			Subtype:     items.Mundane,
			IsComponent: true,
		},
	})
	defer cleanup()

	wagon := buildPickupTestWagon(t)
	_, picked := VisitVendorsInRoom(pickupTestRoomId, wagon,
		nil, []string{"stillwater"})

	if got := len(picked); got != 8 {
		t.Errorf("expected 8 iron ingots picked up, got %d (items=%v)", got, picked)
	}
}

// TestVisitVendorsInRoom_SkipsFinishedGoodsEvenIfComponent is the defensive
// case: if a finished-good item (Type=weapon) were ever mis-tagged
// is_component=true, the Type filter must still exclude it from caravan
// pickup. Finished goods stay where they're crafted.
func TestVisitVendorsInRoom_SkipsFinishedGoodsEvenIfComponent(t *testing.T) {
	cleanup := setupPickupTestFixtures(t, []shops.StockEntry{
		// Mis-tagged "iron dagger": Type=weapon BUT IsComponent=true.
		{ItemId: 10500, RestockQty: 5, MaxStock: 50, Current: 40},
	}, map[int]*items.ItemSpec{
		10500: {
			ItemId:      10500,
			Name:        "iron dagger",
			Type:        items.Weapon,
			Subtype:     items.Stabbing,
			IsComponent: true,
		},
	})
	defer cleanup()

	wagon := buildPickupTestWagon(t)
	_, picked := VisitVendorsInRoom(pickupTestRoomId, wagon,
		nil, []string{"stillwater"})

	if got := len(picked); got != 0 {
		t.Errorf("expected 0 iron daggers picked up (finished good), got %d", got)
	}
}

// TestVisitVendorsInRoom_StillRespectsZoneBucketForNonComponents verifies the
// original zone-source bucket-match path keeps working post-fix. Items in
// pickupBuckets are picked up regardless of IsComponent (the original gate
// is the fast path; the new path is purely additive).
func TestVisitVendorsInRoom_StillRespectsZoneBucketForNonComponents(t *testing.T) {
	cleanup := setupPickupTestFixtures(t, []shops.StockEntry{
		// Lake-iron nodule: bucket "stillwater", is_component=true, type=object.
		// Current=20, RestockQty=5 -> qty = max(1, 20/10) = 2.
		{ItemId: 40059, RestockQty: 5, MaxStock: 50, Current: 20},
	}, map[int]*items.ItemSpec{
		40059: {
			ItemId:      40059,
			Name:        "lake-iron nodule",
			Type:        items.Object,
			Subtype:     items.Mundane,
			IsComponent: true,
		},
	})
	defer cleanup()

	wagon := buildPickupTestWagon(t)
	_, picked := VisitVendorsInRoom(pickupTestRoomId, wagon,
		nil, []string{"stillwater"})

	if got := len(picked); got != 2 {
		t.Errorf("expected 2 lake-iron nodules picked up via bucket match, got %d", got)
	}
}

// TestVisitVendorsInRoom_SkipsNonComponentNonBucketItems is the negative
// guard: items that are NEITHER bucket-matching NOR is_component should
// still be ignored. Confirms the helper isn't accidentally too permissive.
func TestVisitVendorsInRoom_SkipsNonComponentNonBucketItems(t *testing.T) {
	cleanup := setupPickupTestFixtures(t, []shops.StockEntry{
		// Some plain object that's NOT a component and NOT bucketed.
		{ItemId: 11111, RestockQty: 5, MaxStock: 50, Current: 30},
	}, map[int]*items.ItemSpec{
		11111: {
			ItemId:      11111,
			Name:        "trinket",
			Type:        items.Object,
			Subtype:     items.Mundane,
			IsComponent: false,
		},
	})
	defer cleanup()

	wagon := buildPickupTestWagon(t)
	_, picked := VisitVendorsInRoom(pickupTestRoomId, wagon,
		nil, []string{"stillwater"})

	if got := len(picked); got != 0 {
		t.Errorf("expected 0 trinkets picked up (no bucket, not component), got %d", got)
	}
}

// ─── Test helpers ────────────────────────────────────────────────────────────

// seedTestRoomWithMobs creates a fresh rooms.Room at roomId, seeds one
// mob instance per mobId (with HasShop() == false), and places them in
// the room. Returns a cleanup function that removes both the room and
// mob instances.
func seedTestRoomWithMobs(t *testing.T, roomId int, zone string, mobIds []mobs.MobId) func() {
	t.Helper()

	r := &rooms.Room{
		RoomId: roomId,
		Zone:   zone,
		Title:  "Test Room",
		Exits:  map[string]exit.RoomExit{},
	}
	cleanRoom := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{roomId: r},
		map[string]*rooms.ZoneConfig{},
	)

	// Seed each mob as a plain (no-shop) mob instance.
	for i, mobId := range mobIds {
		instId := 90000 + roomId*100 + i
		mob := &mobs.Mob{
			MobId:      mobId,
			InstanceId: instId,
			HomeRoomId: roomId,
			Zone:       zone,
		}
		mob.Character.Name = "TestMob"
		mob.Character.RoomId = roomId
		mob.Character.Buffs = buffs.New()
		mobs.SetInstanceForTest(instId, mob)
		r.AddMob(instId)
	}

	return func() {
		for i := range mobIds {
			instId := 90000 + roomId*100 + i
			mobs.SetInstanceForTest(instId, nil)
		}
		cleanRoom()
	}
}

// seedTestRoomWithExistingMobs is like seedTestRoomWithMobs but takes
// already-built *mobs.Mob values (used when the mob needs specific fields
// set, e.g. a shop-bearing mob).
func seedTestRoomWithExistingMobs(t *testing.T, roomId int, zone string, list []*mobs.Mob) func() {
	t.Helper()

	r := &rooms.Room{
		RoomId: roomId,
		Zone:   zone,
		Title:  "Test Room",
		Exits:  map[string]exit.RoomExit{},
	}
	cleanRoom := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{roomId: r},
		map[string]*rooms.ZoneConfig{},
	)

	for _, mob := range list {
		mob.Character.RoomId = roomId
		// Buffs is a struct value; only init if it has no entries yet.
		if len(mob.Character.Buffs.List) == 0 {
			mob.Character.Buffs = buffs.New()
		}
		mobs.SetInstanceForTest(mob.InstanceId, mob)
		r.AddMob(mob.InstanceId)
	}

	return func() {
		for _, mob := range list {
			mobs.SetInstanceForTest(mob.InstanceId, nil)
		}
		cleanRoom()
	}
}

// buildShopBearingTestMob builds a *mobs.Mob with HasShop() == true.
// The shop has one stock entry. No shop inventory is registered with the
// shops package, so VisitVendorsInRoom will silently skip it.
func buildShopBearingTestMob(t *testing.T, name string) *mobs.Mob {
	t.Helper()

	mob := &mobs.Mob{
		MobId:      mobs.MobId(9001),
		InstanceId: 99001,
		HomeRoomId: 7778,
		Zone:       "TestZone",
	}
	mob.Character.Name = name
	mob.Character.Buffs = buffs.New()
	// Populate Shop with one item so HasShop() returns true.
	mob.Character.Shop = characters.Shop{
		{ItemId: 1, QuantityMax: 5, Quantity: 5},
	}
	return mob
}

// ─── Pickup-filter test fixtures (3.8 hotfix H2) ─────────────────────────────

const (
	pickupTestRoomId       = 7780
	pickupTestVendorMobId  = 9002
	pickupTestVendorInstId = 99002
	pickupTestZone         = "PickupTestZone"
)

// setupPickupTestFixtures wires everything needed for a vendor-pickup
// integration test: a room, a shop-bearing vendor mob in the room, a
// registered shop inventory with the given stock, and a temporary
// items-package seed so items.New produces valid items in the pickup
// loop. Returns a single cleanup function that tears down all of it.
//
// Note on shop-cache isolation: shops.SaveShop writes the shop state to
// disk (configs.DataFiles, redirected to a temp dir in TestMain).
// shops.RegisterShop will load that file back on subsequent calls,
// which would leak the previous test's stock into the next one. We
// scrub the on-disk shop file in both setup and cleanup so each test
// starts with a clean slate.
func setupPickupTestFixtures(
	t *testing.T,
	stock []shops.StockEntry,
	specs map[int]*items.ItemSpec,
) func() {
	t.Helper()

	cleanupItems := items.SeedItemsForTest(specs)

	shops.ClearCache()
	wipePickupTestShopFile(t)

	tmpl := shops.ShopInventory{
		Gold:         500,
		StartingGold: 500,
		CraftSupport: shops.CraftSupportGeneral,
		Stock:        stock,
	}
	// RegisterShop seeds Current to RestockQty by default; force the
	// authored Current values so tests have deterministic stock levels.
	inv := shops.RegisterShop(pickupTestZone, pickupTestVendorMobId, pickupTestRoomId, tmpl)
	for i, want := range stock {
		inv.Stock[i].Current = want.Current
	}

	vendor := &mobs.Mob{
		MobId:      mobs.MobId(pickupTestVendorMobId),
		InstanceId: pickupTestVendorInstId,
		HomeRoomId: pickupTestRoomId,
		Zone:       pickupTestZone,
	}
	vendor.Character.Name = "TestVendor"
	vendor.Character.Buffs = buffs.New()
	vendor.Character.Shop = characters.Shop{
		{ItemId: stock[0].ItemId, QuantityMax: 5, Quantity: 5},
	}
	cleanupRoom := seedTestRoomWithExistingMobs(
		t, pickupTestRoomId, pickupTestZone, []*mobs.Mob{vendor})

	return func() {
		cleanupRoom()
		shops.ClearCache()
		wipePickupTestShopFile(t)
		cleanupItems()
	}
}

// wipePickupTestShopFile removes the on-disk shop YAML produced by
// shops.SaveShop during the pickup test. Used by setup AND cleanup so
// no test leaks shop state into the next.
//
// The zone directory MUST be sanitized with util.ConvertForFilename, exactly
// as shops.shopPath does when SaveShop writes the file — "PickupTestZone"
// lands in a lowercase "pickuptestzone/" directory. Deleting the raw mixed-case
// name only works on a case-insensitive filesystem (Windows); on Linux CI the
// mismatch leaves the file behind, so a prior pickup test's saved stock leaks
// into the next via RegisterShop -> loadFromDisk and the bucket assertion picks
// up 0. Mirror the save-path sanitization here.
func wipePickupTestShopFile(t *testing.T) {
	t.Helper()
	zoneDir := filepath.Join(
		configs.GetFilePathsConfig().DataFiles.String(),
		"shops",
		util.ConvertForFilename(pickupTestZone),
	)
	_ = os.RemoveAll(zoneDir)
}

// buildPickupTestWagon builds a wagon mob with enough carry capacity to
// receive picked-up items in the test pickup loop. Item specs are seeded
// with weight=0 by default, so capacity above zero is sufficient.
func buildPickupTestWagon(t *testing.T) *mobs.Mob {
	t.Helper()
	wagon := &mobs.Mob{
		MobId:      mobs.MobId(9003),
		InstanceId: 99003,
		HomeRoomId: pickupTestRoomId,
		Zone:       pickupTestZone,
	}
	wagon.Character.Name = "TestWagon"
	wagon.Character.Buffs = buffs.New()
	wagon.Character.RoomId = pickupTestRoomId
	characters.ApplyMobOverrides(&wagon.Character, 0, 0, 5000)
	return wagon
}
