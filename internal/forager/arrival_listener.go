package forager

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// ForagerArrivalListener consumes events.PatrolWaypointArrival for
// the forager-delivery oneshot patrols. When a forager arrives at a
// vendor waypoint (arrival_event: forager_vendor), fire the per-vendor
// sell handoff (SellToVendor) using items from the forager's own
// satchel.
//
// Non-forager_vendor events, non-forager mobs, and unknown instances
// are silently ignored — the listener is registered globally and
// fields every PatrolWaypointArrival regardless of source. Chunk 3.8.
func ForagerArrivalListener(e events.Event) events.ListenerReturn {
	pwa, ok := e.(events.PatrolWaypointArrival)
	if !ok {
		return events.Continue
	}
	if pwa.ArrivalEvent != "forager_vendor" {
		return events.Continue
	}
	foragerMob := mobs.GetInstance(pwa.MobInstanceId)
	if foragerMob == nil {
		return events.Continue
	}
	profile := ProfileFor(int(foragerMob.MobId))
	if profile == nil {
		return events.Continue
	}
	SellToVendor(pwa.RoomId, profile, foragerMob)
	return events.Continue
}
