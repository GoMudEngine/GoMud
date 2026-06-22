package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// LoadRunnerFromImport tops the runner's inventory up to circuit.LoadCap with
// fresh items drawn round-robin from circuit.ImportItems — the "sea import"
// abstraction for a stationary warehouse source. Bounded (LoadCap) so a looping
// runner can never accumulate unbounded cargo. Returns the number loaded.
func LoadRunnerFromImport(runner *mobs.Mob, c ImportCircuit) int {
	if runner == nil || len(c.ImportItems) == 0 || c.LoadCap <= 0 {
		return 0
	}
	loaded := 0
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
