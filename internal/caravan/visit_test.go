package caravan

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

// ─── VisitVendorsInRoom ──────────────────────────────────────────────────────

func TestVisitVendorsInRoom_NoRoomReturnsNil(t *testing.T) {
	// Use a roomid that's guaranteed not to exist.
	if got := VisitVendorsInRoom(99999999); got != nil {
		t.Errorf("VisitVendorsInRoom(missing) = %v, want nil", got)
	}
}

func TestVisitVendorsInRoom_NoShopMobsReturnsNil(t *testing.T) {
	cleanup := seedTestRoomWithMobs(t, 7777, "TestZone", []mobs.MobId{1})
	defer cleanup()

	got := VisitVendorsInRoom(7777)
	if got != nil {
		t.Errorf("VisitVendorsInRoom = %v, want nil for room with no shop mobs", got)
	}
}

// TestVisitVendorsInRoom_ShopMobNoInventory verifies that a shop-bearing
// mob whose shop inventory is not registered (no save file, no cache) is
// silently skipped, so VisitVendorsInRoom returns nil rather than
// crashing. This is the expected behavior per the plan note: in tests,
// shops.GetShopInventory returns nil because no inventory is registered
// through the normal load path.
func TestVisitVendorsInRoom_ShopMobNoInventory(t *testing.T) {
	mob := buildShopBearingTestMob(t, "TestShopper")
	cleanup := seedTestRoomWithExistingMobs(t, 7778, "TestZone", []*mobs.Mob{mob})
	defer cleanup()

	got := VisitVendorsInRoom(7778)
	if got != nil {
		t.Errorf("VisitVendorsInRoom = %v, want nil (shop mob present but no inventory registered)",
			got)
	}
}

// ─── FormatDeliveryMessage ───────────────────────────────────────────────────

func TestFormatDeliveryMessage_EmptyReturnsEmpty(t *testing.T) {
	if got := FormatDeliveryMessage(nil); got != "" {
		t.Errorf("FormatDeliveryMessage(nil) = %q, want empty string", got)
	}
	if got := FormatDeliveryMessage([]string{}); got != "" {
		t.Errorf("FormatDeliveryMessage([]) = %q, want empty string", got)
	}
}

func TestFormatDeliveryMessage_SingleVendor(t *testing.T) {
	got := FormatDeliveryMessage([]string{"Ketil"})
	if got == "" {
		t.Error("FormatDeliveryMessage([Ketil]) returned empty, want non-empty")
	}
	// Should mention the vendor name.
	if !contains(got, "Ketil") {
		t.Errorf("FormatDeliveryMessage single: %q does not contain 'Ketil'", got)
	}
}

func TestFormatDeliveryMessage_MultipleVendors(t *testing.T) {
	got := FormatDeliveryMessage([]string{"Ketil", "Marta", "Lars"})
	if got == "" {
		t.Error("FormatDeliveryMessage([Ketil,Marta,Lars]) returned empty")
	}
	for _, name := range []string{"Ketil", "Marta", "Lars"} {
		if !contains(got, name) {
			t.Errorf("FormatDeliveryMessage multi: %q does not contain %q", got, name)
		}
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
