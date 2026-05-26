package caravan

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/sealedcrate"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// CaravanArrivalListener consumes events.PatrolWaypointArrival and
// dispatches caravan-specific bookkeeping based on the event's
// ArrivalEvent name. Filters on the caravan patrol id; ignores
// arrivals from any other patrol.
//
// Dispatches:
//   - caravan_depot at wp0 (long-dwell Thornwall start) → crew regroup
//     (handles the "leader died mid-route, respawned at depot, fresh
//      cycle starts" contingency self-healingly).
//   - caravan_depot at any other wp → state-name transition stamp.
//   - caravan_vendor → bidirectional VisitVendorsInRoom + room flavor.
//   - caravan_fernway_pickup → forager handoff bookkeeping.
//   - empty / unknown → no-op (free-form arrival_event contract).
//
// Chunk 3.7. Registered with the event bus in hooks.RegisterListeners (T9).
func CaravanArrivalListener(e events.Event) events.ListenerReturn {
	arrival, ok := e.(events.PatrolWaypointArrival)
	if !ok {
		return events.Continue
	}
	if arrival.PatrolId != CaravanPatrolId {
		return events.Continue
	}

	leader := mobs.GetInstance(arrival.MobInstanceId)
	if leader == nil {
		return events.Continue
	}

	// Stamp synthesized-state-started-round on transitions. The dashboard
	// reads this from MiscData (no longer from BTreeState).
	stampStateStartedRound(leader)

	switch arrival.ArrivalEvent {
	case "caravan_depot":
		handleDepotArrival(leader, arrival)
	case "caravan_vendor":
		handleVendorArrival(leader, arrival)
	case "caravan_fernway_pickup":
		handleFernwayPickupArrival(leader, arrival)
	}

	return events.Continue
}

// stampStateStartedRound writes caravan_state_started_round MiscData on
// the leader if the synthesized state name has flipped since the last
// stamp. Idempotent across multiple arrivals at the same state.
func stampStateStartedRound(leader *mobs.Mob) {
	state, ok := SynthesizeStateForLeader(leader)
	if !ok {
		return
	}
	prevName, _ := leader.Character.GetMiscData("caravan_state_last").(string)
	if prevName == state.Name() {
		return // no transition
	}
	leader.Character.SetMiscData("caravan_state_started_round", util.GetRoundCount())
	leader.Character.SetMiscData("caravan_state_last", state.Name())
}

// handleDepotArrival fires the crew-regroup mechanism when the leader is
// at wp0 (the long-dwell Thornwall start — also the post-respawn landing
// point). At other depot waypoints the leader stamp has already been
// recorded above; no further action needed until cargo settlement is
// designed for future chunks.
func handleDepotArrival(leader *mobs.Mob, arrival events.PatrolWaypointArrival) {
	if arrival.WaypointIdx != 0 {
		return // only wp0 triggers the fresh-cycle regroup
	}
	// Force-move wagon + any in-world crew to the leader's room so a
	// fresh cycle starts with everyone co-located. This also covers the
	// "leader died mid-route and respawned at depot" contingency — on
	// next wp0 arrival all stragglers are pulled home.
	leaderRoom := leader.Character.RoomId
	for templateId := range caravanMobIds {
		if templateId == int(leader.MobId) {
			continue // skip the leader itself
		}
		for _, instId := range mobs.GetAllMobInstanceIds() {
			m := mobs.GetInstance(instId)
			if m == nil || int(m.MobId) != templateId {
				continue
			}
			if m.Character.RoomId == leaderRoom {
				continue // already co-located, nothing to do
			}
			oldRoomId := m.Character.RoomId
			oldRoom := rooms.LoadRoom(oldRoomId)
			newRoom := rooms.LoadRoom(leaderRoom)
			if newRoom == nil {
				continue
			}
			if oldRoom != nil {
				oldRoom.RemoveMob(m.InstanceId)
			}
			newRoom.AddMob(m.InstanceId) // sets m.Character.RoomId internally
			mudlog.Info("caravan crew regroup",
				"leader", leader.Character.Name,
				"crew_template", templateId,
				"crew_instance", m.InstanceId,
				"from_room", oldRoomId,
				"to_room", leaderRoom,
			)
		}
	}
}

// handleVendorArrival fires the bidirectional vendor trade and prints the
// room flavor message. Bucket lists match the legacy bucketsForRouteState
// logic lifted from internal/behaviortree/actions_caravan.go.
func handleVendorArrival(leader *mobs.Mob, arrival events.PatrolWaypointArrival) {
	wagon := FindWagonInRoom(arrival.RoomId)
	if wagon == nil {
		mudlog.Warn("caravan vendor stop without wagon",
			"leader", leader.Character.Name,
			"room", arrival.RoomId,
		)
		return
	}

	deliveryBuckets, pickupBuckets := bucketsForWaypointIdx(arrival.WaypointIdx)

	delivered, pickedUp := VisitVendorsInRoom(arrival.RoomId, wagon, deliveryBuckets, pickupBuckets)
	if msg := FormatVisitMessage(delivered, pickedUp); msg != "" {
		if r := rooms.LoadRoom(arrival.RoomId); r != nil {
			r.SendText(messaging.CategoryMobEmote, msg)
		}
	}
}

// bucketsForWaypointIdx returns the (delivery, pickup) bucket lists for
// a caravan vendor stop based on the waypoint index in the patrol.
//
// Waypoints 3-10 are Stillwater vendor stops (outbound leg):
//   deliver thornwall + fernway, pick up stillwater.
// Waypoints 14-21 are Thornwall vendor stops (inbound leg):
//   deliver stillwater + fernway, pick up thornwall.
//
// This mirrors the legacy bucketsForRouteState in
// internal/behaviortree/actions_caravan.go.
func bucketsForWaypointIdx(idx int) (delivery, pickup []string) {
	switch {
	case idx >= 3 && idx <= 10:
		// Stillwater vendor circuit (outbound)
		return []string{"thornwall", "fernway"}, []string{"stillwater"}
	case idx >= 14 && idx <= 21:
		// Thornwall vendor circuit (inbound)
		return []string{"stillwater", "fernway"}, []string{"thornwall"}
	}
	return nil, nil
}

// handleFernwayPickupArrival fires the forager handoff that previously
// lived in the tickFernwayPickup btree action in
// internal/behaviortree/actions_caravan.go. Drains the sealed crate at
// room 4038 into the wagon's inventory and emits a flavor message when
// items are transferred.
func handleFernwayPickupArrival(leader *mobs.Mob, arrival events.PatrolWaypointArrival) {
	wagon := FindWagonInRoom(arrival.RoomId)
	if wagon == nil {
		mudlog.Warn("caravan fernway pickup without wagon",
			"leader", leader.Character.Name,
			"room", arrival.RoomId,
		)
		return
	}

	r := rooms.LoadRoom(arrival.RoomId)
	if r == nil || r.SealedCrate == nil {
		return // no crate present — foragers haven't stocked it yet
	}

	stored, rejected := drainCrateIntoWagonCaravan(r.SealedCrate, wagon)
	if stored > 0 || rejected > 0 {
		persistCaravanCrate(arrival.RoomId, r.SealedCrate)
	}
	if stored > 0 {
		r.SendText(messaging.CategoryMobEmote, fmt.Sprintf(
			`<ansi fg="yellow">The caravan pulls up to the roadside crate, breaks the seal,`+
				` and loads its contents into the wagon — %d %s in all.</ansi>`,
			stored, caravanPluralize("crate-load", stored)))
	}
	if rejected > 0 {
		mudlog.Warn("caravan.handleFernwayPickupArrival: wagon refused some items",
			"roomId", arrival.RoomId,
			"stored", stored,
			"returnedToCrate", rejected,
		)
	}
}

// drainCrateIntoWagonCaravan transfers as many items as possible from
// crate into the wagon's inventory. Items the wagon refuses (over carry
// capacity) are returned to the crate so they stay safe for the next
// pickup. Returns (stored, rejected). Persistence is the caller's
// responsibility.
//
// Lifted from drainCrateIntoWagon in
// internal/behaviortree/actions_caravan.go.
func drainCrateIntoWagonCaravan(crate *sealedcrate.Crate, wagon *mobs.Mob) (stored, rejected int) {
	drained := crate.DrainAll()
	for _, it := range drained {
		if wagon.Character.StoreItem(it) {
			stored++
		} else {
			crate.Add(it)
			rejected++
		}
	}
	return stored, rejected
}

// persistCaravanCrate writes the sealed crate's current state to its
// YAML file at <DataFiles>/crates/<roomid>-fernway_shipment.yaml.
// Errors are logged but not surfaced — persistence is best-effort
// crash-safety.
//
// Lifted from persistCrate in internal/behaviortree/actions_forager.go
// and actions_caravan.go.
func persistCaravanCrate(roomId int, c *sealedcrate.Crate) {
	path := util.FilePath(
		configs.GetConfig().FilePaths.DataFiles.String(),
		fmt.Sprintf("/crates/%d-fernway_shipment.yaml", roomId),
	)
	if err := sealedcrate.SaveTo(path, c); err != nil {
		mudlog.Error("caravan.persistCaravanCrate", "roomId", roomId, "error", err)
	}
}

// caravanPluralize returns word+"s" when n != 1, else word unchanged.
func caravanPluralize(word string, n int) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
