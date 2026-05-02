// Package caravan owns the route data, state machine, and vendor-visit
// logic for the Stage 2 NPC caravan that runs Thornwall <-> Stillwater.
//
// See docs/superpowers/specs/2026-04-27-caravan-system-design.md for
// the full spec. Stage 1 NPC party primitives (internal/parties +
// internal/behaviortree party_* actions) drive the crew's coordinated
// movement and combat; this package owns only the route topology, the
// state-enum, and the per-stop restock invocation.
package caravan

// Depot rooms (decided in plan task 0).
const (
	ThornwallDepotRoomId  = 465  // Market Square, Center
	StillwaterDepotRoomId = 4109 // North Square
)

// Stillwater vendor rooms in visit order (cluster-local).
var stillwaterVendorRooms = []int{
	4102, // fishmonger Tov Brann
	4103, // innkeeper Sigrid
	4105, // storekeeper Wulf
	4106, // smith Brindle
	4125, // apothecary Ilsa
	4126, // pearl-carver Kess
	4135, // miller Bram
	4143, // weaver Edda
}

// Thornwall vendor rooms in visit order.
var thornwallVendorRooms = []int{
	464, // food vendor
	470, // blacksmith Kerra
	471, // apothecary Voss
	475, // fence dealer Siv
	480, // weaver Maren
	481, // tavern cook Brynn
	482, // jeweler Tess
	483, // enchanter Vael
}

// Route is one leg of the caravan cycle.
type Route struct {
	// DepartFromRoomId is the depot the caravan starts this leg at.
	DepartFromRoomId int
	// ArriveAtRoomId is the depot the caravan reaches at the end of
	// transit (before vendor visits begin).
	ArriveAtRoomId int
	// VendorStopIds is the ordered list of rooms the caravan visits
	// after arriving. Each room's shop-bearing mobs get a Restock()
	// call when the caravan stops in that room.
	VendorStopIds []int
}

var (
	// OutboundRoute: Thornwall -> Stillwater. Vendors visited after arrival.
	OutboundRoute = Route{
		DepartFromRoomId: ThornwallDepotRoomId,
		ArriveAtRoomId:   StillwaterDepotRoomId,
		VendorStopIds:    stillwaterVendorRooms,
	}
	// InboundRoute: Stillwater -> Thornwall. Vendors visited after arrival.
	InboundRoute = Route{
		DepartFromRoomId: StillwaterDepotRoomId,
		ArriveAtRoomId:   ThornwallDepotRoomId,
		VendorStopIds:    thornwallVendorRooms,
	}
)
