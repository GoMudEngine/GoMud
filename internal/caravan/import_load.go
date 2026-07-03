package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/warehouse"
)

// drawdownEnabledFn resolves whether Stage 4 warehouse drawdown is active
// (both the warehouse master toggle AND the drawdown kill switch). Swapped
// in tests so unit tests never read live config — mirrors the warehouse
// package's itemCapFn pattern.
var drawdownEnabledFn = func() bool {
	return bool(configs.GetGamePlayConfig().WarehousesEnabled) && bool(configs.GetGamePlayConfig().WarehouseDrawdownEnabled)
}

// LoadRunnerFromImport tops the runner's inventory up to circuit.LoadCap with
// fresh items drawn round-robin from circuit.ImportItems — the "sea import"
// abstraction for a stationary warehouse source. Bounded (LoadCap) so a looping
// runner can never accumulate unbounded cargo.
//
// Stage 4: when warehouse drawdown is enabled and the runner's current room
// sits in a warehouse city, a first pass (loadRunnerFromWarehouse) draws
// import items from that local warehouse before the unchanged manifest
// round-robin tops up any remaining capacity from the infinite trickle. No
// demand prioritization here (unlike the ferry factor loader) — Dobb's
// LoadCap comfortably covers his whole manifest, so draw order doesn't
// change the outcome. Returns the number loaded.
func LoadRunnerFromImport(runner *mobs.Mob, c ImportCircuit) int {
	if runner == nil || len(c.ImportItems) == 0 || c.LoadCap <= 0 {
		return 0
	}

	loaded := 0
	if drawdownEnabledFn() {
		if room := rooms.LoadRoom(runner.Character.RoomId); room != nil {
			if _, ok := warehouse.CityFor(room.Zone); ok {
				loaded += loadRunnerFromWarehouse(runner, c, room.Zone)
			}
		}
	}

	maxTries := c.LoadCap*2 + len(c.ImportItems) // bound: avoids spin on invalid ids
	for tries := 0; tries < maxTries && len(runner.Character.Items) < c.LoadCap; tries++ {
		it := items.New(c.ImportItems[tries%len(c.ImportItems)])
		if !it.IsValid() {
			continue
		}
		if !runner.Character.StoreItem(it) {
			break // runner at carry capacity
		}
		loaded++
	}
	return loaded
}

// loadRunnerFromWarehouse draws c.ImportItems round-robin from zone's
// warehouse, minting the physical item for each unit successfully
// withdrawn and storing it on the runner. On a StoreItem failure (runner
// at carry capacity) the withdrawn unit is re-deposited rather than
// destroyed — Stage 4 never loses stock to a full cart. Bounded like the
// sibling manifest loader. Returns the number of items loaded.
func loadRunnerFromWarehouse(runner *mobs.Mob, c ImportCircuit, zone string) int {
	loaded := 0
	maxTries := c.LoadCap*2 + len(c.ImportItems)
	for tries := 0; tries < maxTries && len(runner.Character.Items) < c.LoadCap; tries++ {
		itemId := c.ImportItems[tries%len(c.ImportItems)]
		n := warehouse.Withdraw(zone, itemId, 1)
		if n <= 0 {
			continue
		}
		it := items.New(itemId)
		if !it.IsValid() {
			// Can't mint the physical item — re-deposit so the unit isn't
			// silently lost (Deposit re-counts it as captured, acceptable).
			warehouse.Deposit(zone, itemId, 1)
			continue
		}
		if !runner.Character.StoreItem(it) {
			warehouse.Deposit(zone, itemId, 1)
			break // runner at carry capacity
		}
		loaded++
	}
	return loaded
}
