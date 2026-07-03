package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/warehouse"
)

// WarehouseTick runs ambient accrual on cadence and saves dirty pools.
// The WarehousesEnabled gate lives inside warehouse.Tick().
func WarehouseTick(e events.Event) events.ListenerReturn {
	warehouse.Tick()
	return events.Continue
}
