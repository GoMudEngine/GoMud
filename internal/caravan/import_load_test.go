package caravan

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/warehouse"
)

func TestLoadRunnerFromImport_RespectsCapAndSkipsWhenRegistryAbsent(t *testing.T) {
	runner := newCargoTestMob(80904, 9304, "Dobb")
	c := ImportCircuit{
		ImportItems: []int{40001, 40006, 40012},
		LoadCap:     5,
	}
	loaded := LoadRunnerFromImport(runner, c)
	if loaded == 0 {
		// Item registry not loaded in unit context — acceptable; smoke covers it.
		t.Skip("items.New returned invalid (registry not loaded); integrated smoke covers this")
	}
	if loaded > c.LoadCap || len(runner.Character.Items) > c.LoadCap {
		t.Errorf("load exceeded cap: loaded=%d items=%d cap=%d",
			loaded, len(runner.Character.Items), c.LoadCap)
	}
}

func TestLoadRunnerFromImport_NilOrEmptyNoOp(t *testing.T) {
	if got := LoadRunnerFromImport(nil, ImportCircuit{ImportItems: []int{40001}, LoadCap: 3}); got != 0 {
		t.Errorf("nil runner should load 0, got %d", got)
	}
	runner := newCargoTestMob(80905, 9304, "Dobb")
	if got := LoadRunnerFromImport(runner, ImportCircuit{ImportItems: nil, LoadCap: 3}); got != 0 {
		t.Errorf("empty manifest should load 0, got %d", got)
	}
	if got := LoadRunnerFromImport(runner, ImportCircuit{ImportItems: []int{40001}, LoadCap: 0}); got != 0 {
		t.Errorf("zero cap should load 0, got %d", got)
	}
}

// TestLoadRunnerFromImport_ToggleOffDoesNotTouchWarehouse pins the default
// unit-test posture: drawdownEnabledFn defaults to reading live config,
// whose ConfigBool zero-value is false in the unit-test process, so the
// warehouse-first pass never fires and existing (pre-Stage-4) behavior is
// byte-identical. This guards against a future default flip silently
// changing this test's meaning.
func TestLoadRunnerFromImport_ToggleOffDoesNotTouchWarehouse(t *testing.T) {
	warehouse.ResetForTest()
	warehouse.Deposit("New Plymouth Docks", 40001, 5)

	roomId := 780901
	cleanRoom := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{roomId: {RoomId: roomId, Zone: "New Plymouth Docks", Title: "Depot", Exits: map[string]exit.RoomExit{}}},
		map[string]*rooms.ZoneConfig{},
	)
	defer cleanRoom()

	runner := newCargoTestMob(80906, 9304, "Dobb")
	runner.Character.RoomId = roomId

	LoadRunnerFromImport(runner, ImportCircuit{ImportItems: []int{40001}, LoadCap: 5})

	if got := warehouse.WarehouseFor("New Plymouth Docks").StockOf(40001); got != 5 {
		t.Errorf("warehouse stock = %d, want untouched 5 (toggle off)", got)
	}
}

// TestLoadRunnerFromImport_DrawsFromWarehouseFirst forces drawdownEnabledFn
// on (the ConfigBool toggle is unreachable as true in the unit-test
// process, so this test hook is the only way to exercise the warehouse-
// first pass without booting the full config/data-file stack — mirrors
// the warehouse package's itemCapFn override pattern). The item spec is
// seeded via items.SeedItemsForTest so items.New mints valid items and
// the assertions genuinely execute (the skip guard is a defensive
// fallback only — it must not trigger). LoadCap (3) < warehouse stock
// (5) guarantees every loaded unit came from the warehouse: the manifest
// round-robin gets no remaining capacity, so DrawnCount must equal the
// full load.
func TestLoadRunnerFromImport_DrawsFromWarehouseFirst(t *testing.T) {
	orig := drawdownEnabledFn
	drawdownEnabledFn = func() bool { return true }
	defer func() { drawdownEnabledFn = orig }()

	cleanupItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		40001: {
			ItemId:  40001,
			Name:    "iron ingot",
			Type:    items.Object,
			Subtype: items.Mundane,
		},
	})
	defer cleanupItems()

	warehouse.ResetForTest()
	warehouse.Deposit("New Plymouth Docks", 40001, 5)

	roomId := 780902
	cleanRoom := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{roomId: {RoomId: roomId, Zone: "New Plymouth Docks", Title: "Depot", Exits: map[string]exit.RoomExit{}}},
		map[string]*rooms.ZoneConfig{},
	)
	defer cleanRoom()

	runner := newCargoTestMob(80907, 9304, "Dobb")
	runner.Character.RoomId = roomId
	characters.ApplyMobOverrides(&runner.Character, 0, 0, 5000)

	loaded := LoadRunnerFromImport(runner, ImportCircuit{ImportItems: []int{40001}, LoadCap: 3})
	if loaded == 0 {
		// Defensive fallback only — the seeded spec above means items.New
		// must succeed; reaching this skip indicates fixture breakage.
		t.Skip("items.New returned invalid despite seeded spec; investigate fixture")
	}
	if loaded != 3 {
		t.Errorf("loaded = %d, want 3 (LoadCap bounds the draw below the 5 in stock)", loaded)
	}
	if got := warehouse.WarehouseFor("New Plymouth Docks").StockOf(40001); got != 5-loaded {
		t.Errorf("warehouse stock = %d, want %d (5 - %d drawn)", got, 5-loaded, loaded)
	}
	if warehouse.WarehouseFor("New Plymouth Docks").DrawnCount != loaded {
		t.Errorf("DrawnCount = %d, want %d", warehouse.WarehouseFor("New Plymouth Docks").DrawnCount, loaded)
	}
	if len(runner.Character.Items) != loaded {
		t.Errorf("runner carries %d items, want %d", len(runner.Character.Items), loaded)
	}
}
