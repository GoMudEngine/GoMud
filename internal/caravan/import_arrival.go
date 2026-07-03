package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/warehouse"
)

type importAction int

const (
	importNone importAction = iota
	importLoad
	importDeliver
)

// classifyImportArrival is the pure routing decision for an import-circuit
// waypoint arrival, keyed on the authored ArrivalEvent tag.
func classifyImportArrival(c ImportCircuit, arrivalEvent string) importAction {
	switch arrivalEvent {
	case c.DepotEvent:
		return importLoad
	case c.VendorEvent:
		return importDeliver
	}
	return importNone
}

// handleImportArrival drives a single import-runner waypoint stop: load the
// runner at the depot, or deliver to the room's vendors at a vendor stop.
func handleImportArrival(c ImportCircuit, arrival events.PatrolWaypointArrival) {
	runner := FindMobByTemplateInRoom(arrival.RoomId, c.RunnerMobId)
	if runner == nil {
		return
	}
	switch classifyImportArrival(c, arrival.ArrivalEvent) {
	case importLoad:
		// Stage 3 overflow capture: leftovers surviving a full vendor loop
		// mean the city's vendors were full — bank them rather than letting
		// them ride forever. No-op outside warehouse cities, and gated on
		// the master toggle (warehouse.Deposit itself doesn't check it, and
		// with it off SaveDirty never runs, so banked stock would never
		// persist).
		if bool(configs.GetGamePlayConfig().WarehousesEnabled) {
			if room := rooms.LoadRoom(arrival.RoomId); room != nil {
				if _, ok := warehouse.CityFor(room.Zone); ok {
					for i := len(runner.Character.Items) - 1; i >= 0; i-- {
						it := runner.Character.Items[i]
						if warehouse.Deposit(room.Zone, it.ItemId, 1) {
							runner.Character.RemoveItem(it)
						}
					}
				}
			}
		}
		LoadRunnerFromImport(runner, c)
	case importDeliver:
		delivered, _ := VisitVendorsInRoom(arrival.RoomId, runner, c.DeliveryBuckets, nil)
		if msg := FormatVisitMessage(runner.Character.Name, delivered, nil); msg != "" {
			if r := rooms.LoadRoom(arrival.RoomId); r != nil {
				r.SendText(messaging.CategoryMobEmote, msg)
			}
		}
	}
}
