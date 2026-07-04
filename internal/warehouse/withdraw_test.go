package warehouse

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func TestWithdraw_DecrementsAndCounts(t *testing.T) {
	ResetForTest()
	Deposit("The Confluence", 40123, 5)
	got := Withdraw("The Confluence", 40123, 3)
	if got != 3 {
		t.Fatalf("Withdraw = %d, want 3", got)
	}
	w := WarehouseFor("The Confluence")
	if w.StockOf(40123) != 2 || w.DrawnCount != 3 {
		t.Fatalf("stock=%d drawn=%d, want 2/3", w.StockOf(40123), w.DrawnCount)
	}
}

func TestWithdraw_FloorsAtZeroStock(t *testing.T) {
	ResetForTest()
	Deposit("The Confluence", 40123, 2)
	if got := Withdraw("The Confluence", 40123, 10); got != 2 {
		t.Fatalf("partial withdraw = %d, want 2", got)
	}
	if got := Withdraw("The Confluence", 40123, 1); got != 0 {
		t.Fatalf("empty withdraw = %d, want 0", got)
	}
	if got := Withdraw("Stillwater", 40123, 1); got != 0 {
		t.Fatalf("unknown-zone withdraw = %d, want 0", got)
	}
}

// ─── ReleaseToVendorsInRoom ─────────────────────────────────────────────────

const (
	releaseTestZone         = "The Confluence"
	releaseTestRoomId       = 8800
	releaseTestVendorMobId  = 9100
	releaseTestVendorInstId = 99100
	releaseGapItemId        = 40123 // below MaxStock — should be topped up
	releaseAtCapItemId      = 40124 // already at MaxStock — must be untouched
	releaseNotStockedItemId = 40125 // no StockEntry at all — must stay uncreated
)

// TestReleaseToVendorsInRoom_TopsGapsBounded pins the Stage 4 delivery-time
// local release contract: existing vendor stock gaps are topped up from the
// local warehouse, bounded by maxPerItem per entry per call, never creating
// new slots. Mirrors the plan's Task 3 worked example exactly.
func TestReleaseToVendorsInRoom_TopsGapsBounded(t *testing.T) {
	ResetForTest()
	cleanup := setupReleaseTestFixtures(t, []shops.StockEntry{
		{ItemId: releaseGapItemId, MaxStock: 6, Current: 1},
		{ItemId: releaseAtCapItemId, MaxStock: 6, Current: 6},
	})
	defer cleanup()

	Deposit(releaseTestZone, releaseGapItemId, 10)

	got := ReleaseToVendorsInRoom(releaseTestZone, releaseTestRoomId, 2)
	if got != 2 {
		t.Fatalf("release = %d, want 2 (bounded by maxPerItem, not the gap of 5)", got)
	}

	shop := shops.GetShopInventory(releaseTestZone, releaseTestVendorMobId, releaseTestRoomId)
	if shop == nil {
		t.Fatal("expected vendor shop inventory to still be registered")
	}
	entry := shop.GetStock(releaseGapItemId)
	if entry == nil || entry.Current != 3 {
		t.Fatalf("gap entry Current = %+v, want 3 (1 + bounded 2)", entry)
	}

	w := WarehouseFor(releaseTestZone)
	if w.StockOf(releaseGapItemId) != 8 || w.DrawnCount != 2 {
		t.Fatalf("warehouse stock=%d drawn=%d, want 8/2", w.StockOf(releaseGapItemId), w.DrawnCount)
	}

	atCap := shop.GetStock(releaseAtCapItemId)
	if atCap == nil || atCap.Current != 6 {
		t.Fatalf("at-cap entry Current = %+v, want unchanged 6 (no gap, no withdraw)", atCap)
	}

	if entry := shop.GetStock(releaseNotStockedItemId); entry != nil {
		t.Fatalf("not-stocked item should never get a created slot, got %+v", entry)
	}

	// Second call: the top-up continues from where it left off (gap is now
	// 3, bounded take is still 2).
	got2 := ReleaseToVendorsInRoom(releaseTestZone, releaseTestRoomId, 2)
	if got2 != 2 {
		t.Fatalf("second release = %d, want 2", got2)
	}
	entry = shop.GetStock(releaseGapItemId)
	if entry.Current != 5 {
		t.Fatalf("gap entry Current after 2nd call = %d, want 5", entry.Current)
	}
	if got := w.StockOf(releaseGapItemId); got != 6 {
		t.Fatalf("warehouse stock after 2nd call = %d, want 6", got)
	}
	if w.DrawnCount != 4 {
		t.Fatalf("DrawnCount after 2nd call = %d, want 4", w.DrawnCount)
	}
}

// setupReleaseTestFixtures wires a room + shop-bearing vendor mob (with the
// given initial stock) for ReleaseToVendorsInRoom tests. Mirrors
// caravan/visit_test.go's setupCreateSlotTestFixtures — RegisterShop seeds
// Current from the abundance formula, not the raw template values, so the
// authored Current values are force-set after registration for deterministic
// fixtures.
func setupReleaseTestFixtures(t *testing.T, stock []shops.StockEntry) func() {
	t.Helper()

	shops.ClearCache()
	wipeReleaseTestShopFile(t)

	tmpl := shops.ShopInventory{
		Gold:         500,
		StartingGold: 500,
		CraftSupport: shops.CraftSupportGeneral,
		Stock:        stock,
	}
	inv := shops.RegisterShop(releaseTestZone, releaseTestVendorMobId, releaseTestRoomId, tmpl)
	for i, want := range stock {
		inv.Stock[i].Current = want.Current
	}

	vendor := &mobs.Mob{
		MobId:      mobs.MobId(releaseTestVendorMobId),
		InstanceId: releaseTestVendorInstId,
		HomeRoomId: releaseTestRoomId,
		Zone:       releaseTestZone,
	}
	vendor.Character.Name = "TestReleaseVendor"
	vendor.Character.Buffs = buffs.New()
	vendor.Character.RoomId = releaseTestRoomId
	vendor.Character.Shop = characters.Shop{
		{ItemId: 1, QuantityMax: 5, Quantity: 5},
	}
	mobs.SetInstanceForTest(vendor.InstanceId, vendor)

	r := &rooms.Room{
		RoomId: releaseTestRoomId,
		Zone:   releaseTestZone,
		Title:  "Test Room",
		Exits:  map[string]exit.RoomExit{},
	}
	cleanRoom := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{releaseTestRoomId: r},
		map[string]*rooms.ZoneConfig{},
	)
	r.AddMob(vendor.InstanceId)

	return func() {
		mobs.SetInstanceForTest(vendor.InstanceId, nil)
		cleanRoom()
		shops.ClearCache()
		wipeReleaseTestShopFile(t)
	}
}

// wipeReleaseTestShopFile removes the on-disk shop YAML for the release
// test zone/vendor, so repeated test runs (and the shops.RegisterShop ->
// loadFromDisk fallback) don't leak a prior run's stock levels into the
// next. Mirrors caravan/visit_test.go's wipePickupTestShopFile.
func wipeReleaseTestShopFile(t *testing.T) {
	t.Helper()
	zoneDir := filepath.Join(
		configs.GetFilePathsConfig().DataFiles.String(),
		"shops",
		util.ConvertForFilename(releaseTestZone),
	)
	_ = os.RemoveAll(zoneDir)
}
