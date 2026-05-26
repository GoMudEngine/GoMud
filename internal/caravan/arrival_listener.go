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
// ArrivalEvent name. Accepts both the main caravan patrol and Lars's
// runner-circuit oneshot patrols.
//
// Dispatches for main caravan patrol (CaravanPatrolId):
//   - caravan_depot → crew regroup (if fresh-respawn) + start Lars runner circuit
//   - caravan_fernway_pickup → forager handoff bookkeeping
//   - empty / unknown → no-op
//
// Dispatches for runner-circuit patrols (thornwall/stillwater_runner_circuit):
//   - caravan_vendor → bidirectional VisitVendorsInRoom + room flavor via Lars
//
// Chunk 3.7/3.8. Registered with the event bus in hooks.RegisterListeners.
func CaravanArrivalListener(e events.Event) events.ListenerReturn {
	arrival, ok := e.(events.PatrolWaypointArrival)
	if !ok {
		return events.Continue
	}

	// Runner-circuit vendor stops (chunk 3.8): Lars walks vendor rooms on
	// his oneshot circuit; dispatch directly without the main-caravan leader lookup.
	if _, isRunner := runnerCircuitPatrols[arrival.PatrolId]; isRunner {
		if arrival.ArrivalEvent == "caravan_vendor" {
			mob := mobs.GetInstance(arrival.MobInstanceId)
			if mob != nil {
				handleVendorArrival(mob, arrival)
			}
		}
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

// handleDepotArrival kicks off Lars's runner circuit when the leader
// arrives at a depot waypoint. Chunk 3.8: with the truncated
// 4-waypoint main route, wp0 (Thornwall, 360-round dwell) starts the
// Thornwall vendor circuit; wp2 (Stillwater, 180-round dwell) starts
// the Stillwater vendor circuit.
//
// Also handles two safeties:
//   5.2 (chunk 3.7 carryover): if the leader carries the
//   patrol_fresh_respawn marker (just respawned at depot after a
//   death), regroup any stranded crew via ForceRegroupCrew.
//   5.3 (chunk 3.8): if Lars is co-located at the depot with cargo
//   in his inventory and no active patrol (e.g., his oneshot
//   home-fallback fired and never produced a PatrolCompleted),
//   transfer his cargo back to the wagon now.
func handleDepotArrival(leader *mobs.Mob, arrival events.PatrolWaypointArrival) {
	// 5.2 fresh-respawn regroup (chunk 3.7 carryover).
	if arrival.WaypointIdx == 0 {
		if fresh, _ := leader.Character.GetMiscData("patrol_fresh_respawn").(bool); fresh {
			leader.Character.SetMiscData("patrol_fresh_respawn", false)
			ForceRegroupCrew(leader)
		}
	}

	// 5.3 stranded-cargo safety: pull Lars's cargo back to the wagon
	// if he's at the depot carrying inventory with no active patrol.
	lars := FindRunnerInRoom(leader.Character.RoomId)
	wagon := FindWagonInRoom(leader.Character.RoomId)
	if lars != nil && wagon != nil && len(lars.Character.Items) > 0 && lars.PatrolId == "" {
		TransferAllCargoBack(lars, wagon)
	}

	// Start Lars's runner circuit at wp0 (Thornwall) and wp2 (Stillwater).
	switch arrival.WaypointIdx {
	case 0:
		startRunnerCircuit(leader, arrival, "thornwall_runner_circuit", []string{"stillwater", "fernway"})
	case 2:
		startRunnerCircuit(leader, arrival, "stillwater_runner_circuit", []string{"thornwall", "fernway"})
	}
}

// startRunnerCircuit transfers outbound-bucket cargo from wagon → Lars
// and assigns him the oneshot runner-circuit patrol. No-op if Lars is
// not in the depot, the wagon is not in the depot, or Lars already has
// a patrol assigned (don't double-start). Chunk 3.8.
func startRunnerCircuit(leader *mobs.Mob, arrival events.PatrolWaypointArrival, circuitPatrolId string, outboundBuckets []string) {
	lars := FindRunnerInRoom(arrival.RoomId)
	if lars == nil {
		mudlog.Warn("caravan depot without runner",
			"leader", leader.Character.Name,
			"room", arrival.RoomId,
			"circuit", circuitPatrolId,
		)
		return
	}
	if lars.PatrolId != "" {
		return // already on a circuit — don't double-start
	}
	wagon := FindWagonInRoom(arrival.RoomId)
	if wagon == nil {
		mudlog.Warn("caravan depot without wagon",
			"leader", leader.Character.Name,
			"room", arrival.RoomId,
		)
		return
	}
	TransferCargoToRunner(wagon, lars, outboundBuckets)
	mobs.StartOneshotPatrol(lars, circuitPatrolId)
}

// handleVendorArrival fires the bidirectional vendor trade and prints
// the room flavor message. Chunk 3.8: the source mob is Lars (runner),
// not the wagon — the wagon is parked back at the depot. Lars's oneshot
// patrol queues caravan_vendor arrival events as he walks the circuit;
// each event invokes this handler to drive a sell.
//
// The `leader` argument is the patrol-running mob (Lars at this point,
// not Ketil) because the arrival event is emitted off the runner's
// patrol. We use FindRunnerInRoom for consistency with FindWagonInRoom
// but the leader argument already IS the runner — keep both lookups
// defensively in case the listener gets re-purposed.
func handleVendorArrival(leader *mobs.Mob, arrival events.PatrolWaypointArrival) {
	lars := FindRunnerInRoom(arrival.RoomId)
	if lars == nil {
		mudlog.Warn("caravan vendor stop without runner",
			"leader", leader.Character.Name,
			"room", arrival.RoomId,
		)
		return
	}

	deliveryBuckets, pickupBuckets := bucketsForRunnerPatrol(arrival.PatrolId)

	delivered, pickedUp := VisitVendorsInRoom(arrival.RoomId, lars, deliveryBuckets, pickupBuckets)
	if msg := FormatVisitMessage(delivered, pickedUp); msg != "" {
		if r := rooms.LoadRoom(arrival.RoomId); r != nil {
			r.SendText(messaging.CategoryMobEmote, msg)
		}
	}
}

// bucketsForRunnerPatrol returns the (delivery, pickup) bucket lists
// for a caravan_vendor arrival, keyed by the patrol id rather than
// the waypoint index. Chunk 3.8: caravan vendor stops live entirely
// on Lars's runner-circuit oneshot patrols, so the dispatch key is
// which circuit fired the event.
//
//	thornwall_runner_circuit: Lars is in Thornwall — wagon brought
//	  stillwater + fernway goods on the inbound leg; we deliver
//	  those and pick up thornwall.
//	stillwater_runner_circuit: Lars is in Stillwater — wagon brought
//	  thornwall + fernway goods on the outbound leg; we deliver
//	  those and pick up stillwater.
func bucketsForRunnerPatrol(patrolId string) (delivery, pickup []string) {
	switch patrolId {
	case "thornwall_runner_circuit":
		return []string{"stillwater", "fernway"}, []string{"thornwall"}
	case "stillwater_runner_circuit":
		return []string{"thornwall", "fernway"}, []string{"stillwater"}
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
