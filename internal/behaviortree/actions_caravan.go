package behaviortree

// actions_caravan.go — Stage 2 caravan state-machine btree action.
//
// caravan_step is the single workhorse action that drives the
// continuous Thornwall<->Stillwater caravan cycle. It reads the leader's
// caravan_state from MobState, dispatches to internal/caravan, and
// advances the state when transitions are warranted.
//
// State persistence in MobState (string->string):
//   caravan_state              — current state name (caravan.CaravanState.Name())
//   caravan_state_started_round — round when current state was entered (uint64)
//   caravan_route_index        — index into the current Route's VendorStopIds

import (
	"fmt"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func init() {
	actionRegistry["caravan_step"] = actCaravanStep
}

func actCaravanStep(params map[string]any, ctx *EvalContext) Result {
	if ctx.MobState == nil {
		return Failure
	}
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}

	cur := readCaravanState(ctx.MobState)

	// Dispatch by category.
	switch {
	case caravan.IsDwellState(cur):
		return tickDwell(cur, mob, ctx)
	case caravan.IsTransitState(cur):
		return tickTransit(cur, mob, ctx)
	case caravan.IsRouteState(cur):
		return tickRoute(cur, mob, ctx)
	}
	return Failure
}

// readCaravanState fetches the current state from MobState, defaulting
// to StateThornwallDwell on first tick (no value set).
func readCaravanState(s *BehaviorState) caravan.CaravanState {
	raw := s.GetString("caravan_state")
	if raw == "" {
		s.Set("caravan_state", caravan.StateThornwallDwell.Name())
		s.Set("caravan_state_started_round", strconv.FormatUint(util.GetRoundCount(), 10))
		s.Set("caravan_route_index", "0")
		return caravan.StateThornwallDwell
	}
	parsed, ok := caravan.ParseState(raw)
	if !ok {
		// Corrupt value — reset.
		s.Set("caravan_state", caravan.StateThornwallDwell.Name())
		s.Set("caravan_state_started_round", strconv.FormatUint(util.GetRoundCount(), 10))
		s.Set("caravan_route_index", "0")
		return caravan.StateThornwallDwell
	}
	return parsed
}

// transitionTo writes the new state to MobState and resets per-state
// counters (started-round, route index).
func transitionTo(s *BehaviorState, next caravan.CaravanState) {
	s.Set("caravan_state", next.Name())
	s.Set("caravan_state_started_round", strconv.FormatUint(util.GetRoundCount(), 10))
	s.Set("caravan_route_index", "0")
}

// tickDwell: at depot, waiting for the dwell timer to elapse.
//
// Returns Success only when the dwell timer expires and the state actually
// advances. While still waiting, returns Failure so the btree falls through
// to legacy idle (which fires the mob's idlecommands and lookfortrouble).
// Without this, the caravan crew sits silent at the depot for the entire
// dwell — no flavor emotes from idlecommands.
func tickDwell(cur caravan.CaravanState, mob *mobs.Mob, ctx *EvalContext) Result {
	startedStr := ctx.MobState.GetString("caravan_state_started_round")
	started, _ := strconv.ParseUint(startedStr, 10, 64)
	dwell := uint64(configs.GetBalanceConfig().CaravanDepotDwellRounds)
	if util.GetRoundCount() >= started+dwell {
		transitionTo(ctx.MobState, caravan.AdvanceState(cur))
		return Success
	}
	// Still resting — let legacy idle path (lookfortrouble + idlecommands)
	// handle the round so the crew shows flavor emotes during dwell.
	return Failure
}

// tickTransit: walking toward the destination depot. Issues pathto on
// each tick; the engine's path step + mob walk loop handle progress.
// On arrival, transition to the route phase.
func tickTransit(cur caravan.CaravanState, mob *mobs.Mob, ctx *EvalContext) Result {
	route := caravan.RouteForState(cur)
	if route == nil {
		return Failure
	}
	if ctx.RoomId == route.ArriveAtRoomId {
		transitionTo(ctx.MobState, caravan.AdvanceState(cur))
		return Success
	}
	mob.Command(fmt.Sprintf("pathto %d", route.ArriveAtRoomId))
	return Success
}

// tickRoute: visiting vendor stops in order. On arrival at the next
// stop, fire VisitVendorsInRoom + emit flavor + advance the index. When
// all stops done, transition to the depot dwell state.
func tickRoute(cur caravan.CaravanState, mob *mobs.Mob, ctx *EvalContext) Result {
	route := caravan.RouteForState(cur)
	if route == nil {
		return Failure
	}
	idxStr := ctx.MobState.GetString("caravan_route_index")
	idx, _ := strconv.Atoi(idxStr)
	if idx >= len(route.VendorStopIds) {
		// All stops visited — exit the route phase.
		transitionTo(ctx.MobState, caravan.AdvanceState(cur))
		return Success
	}
	nextRoom := route.VendorStopIds[idx]
	if ctx.RoomId == nextRoom {
		// Arrived at this stop — restock + advance index.
		visited := caravan.VisitVendorsInRoom(nextRoom)
		if msg := caravan.FormatDeliveryMessage(visited); msg != "" {
			if r := rooms.LoadRoom(nextRoom); r != nil {
				r.SendText(msg)
			}
		}
		newIdx := idx + 1
		ctx.MobState.Set("caravan_route_index", strconv.Itoa(newIdx))
		if newIdx >= len(route.VendorStopIds) {
			// All stops done — exit route phase this same tick.
			transitionTo(ctx.MobState, caravan.AdvanceState(cur))
		}
		return Success
	}
	// Not at the next stop yet — pathto it.
	mob.Command(fmt.Sprintf("pathto %d", nextRoom))
	return Success
}
