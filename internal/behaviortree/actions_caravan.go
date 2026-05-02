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
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
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

	// Stuck-watchdog: if the caravan has been in its current state far
	// longer than the longest legitimate phase, reset to ThornwallDwell.
	// Catches cases where a state never advances (orphaned transit after
	// restart, permanent hostile detection, etc.). Threshold scales with
	// configured dwell so non-test configs don't false-positive.
	startedStr := ctx.MobState.GetString("caravan_state_started_round")
	started, _ := strconv.ParseUint(startedStr, 10, 64)
	threshold := uint64(5 * configs.GetBalanceConfig().CaravanDepotDwellRounds)
	if threshold < 300 {
		threshold = 300
	}
	if started > 0 && util.GetRoundCount() > started+threshold {
		mudlog.Warn("caravan stuck, auto-resetting state",
			"currentState", cur.Name(),
			"startedRound", started,
			"nowRound", util.GetRoundCount(),
			"thresholdRounds", threshold,
			"leaderInstanceId", ctx.InstanceId,
			"leaderName", mob.Character.Name)
		resetCaravanState(ctx.MobState)
		return Failure
	}

	// Dispatch by category. FernwayPickup must come BEFORE the generic
	// IsTransitState check because IsFernwayPickupState states also
	// return true for IsTransitState (Task 9 widened it for RouteForState
	// lookups) — dispatching here first overrides that.
	switch {
	case caravan.IsDwellState(cur):
		return tickDwell(cur, mob, ctx)
	case caravan.IsFernwayPickupState(cur):
		return tickFernwayPickup(cur, mob, ctx)
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

// resetCaravanState clears all per-cycle state and returns the caravan
// to its canonical starting state (ThornwallDwell). Used by both the
// auto-watchdog and the admin override.
func resetCaravanState(s *BehaviorState) {
	s.Set("caravan_state", caravan.StateThornwallDwell.Name())
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
//
// Halts (returns Failure) if any mob in the current room is on the
// caller's hates list — this lets the legacy idle path fire
// lookfortrouble, which sets aggro on the hostile mob and engages
// combat. Without this halt, the caravan would issue another pathto
// step and keep walking past a brawl in progress.
func tickTransit(cur caravan.CaravanState, mob *mobs.Mob, ctx *EvalContext) Result {
	route := caravan.RouteForState(cur)
	if route == nil {
		return Failure
	}
	if partyHostilesNearby(ctx.InstanceId) || anyPartyMemberInCombat(ctx.InstanceId) {
		return Failure
	}
	if ctx.RoomId == route.ArriveAtRoomId {
		transitionTo(ctx.MobState, caravan.AdvanceState(cur))
		return Success
	}
	mob.Command(fmt.Sprintf("pathto %d", route.ArriveAtRoomId))
	return Success
}

// fernwayMeetingRoomId is the North Road room where the caravan pauses
// to pick up the sealed crate of Fernway forager goods.
const fernwayMeetingRoomId = 4038

// tickFernwayPickup: brief dwell at North Road 4038 to drain the sealed
// crate into the wagon's inventory. Lives between the transit and route
// phases of each cycle.
//
// Behavior:
//  1. If not at room 4038 yet, pathto it.
//  2. On arrival (elapsed==0), drain the room's sealed crate into the
//     wagon's inventory and emit a flavor message if any items moved.
//  3. Dwell for FernwayPickupDwellRounds, then advance to the next state.
//
// Returns Success while making progress. Returns Failure when dwelling
// at room 4038 so legacy idle path can fire flavor emotes.
func tickFernwayPickup(cur caravan.CaravanState, mob *mobs.Mob, ctx *EvalContext) Result {
	if partyHostilesNearby(ctx.InstanceId) || anyPartyMemberInCombat(ctx.InstanceId) {
		return Failure
	}

	// Not at meeting point — pathto.
	if ctx.RoomId != fernwayMeetingRoomId {
		mob.Command(fmt.Sprintf("pathto %d", fernwayMeetingRoomId))
		return Success
	}

	// At meeting point. On the first tick, drain the sealed crate into
	// the wagon. caravan_state_started_round was set by transitionTo.
	startedStr := ctx.MobState.GetString("caravan_state_started_round")
	started, _ := strconv.ParseUint(startedStr, 10, 64)
	now := util.GetRoundCount()
	elapsed := now - started

	if elapsed == 0 {
		r := rooms.LoadRoom(ctx.RoomId)
		if r != nil && r.SealedCrate != nil {
			wagon := findCaravanWagon(ctx.InstanceId)
			if wagon != nil {
				drained := r.SealedCrate.DrainAll()
				for _, it := range drained {
					wagon.Character.StoreItem(it)
				}
				if len(drained) > 0 {
					persistCrate(ctx.RoomId, r.SealedCrate)
					r.SendText(fmt.Sprintf(
						`<ansi fg="yellow">The caravan pulls up to the roadside crate, breaks the seal,`+
							` and loads its contents into the wagon — %d %s in all.</ansi>`,
						len(drained), caravanPluralize("crate-load", len(drained))))
				}
			}
		}
	}

	// Dwell timer expired — advance to the next state.
	dwell := uint64(configs.GetBalanceConfig().FernwayPickupDwellRounds)
	if elapsed >= dwell {
		transitionTo(ctx.MobState, caravan.AdvanceState(cur))
		return Success
	}

	// Still dwelling — let legacy idle path fire flavor.
	return Failure
}

// tickRoute: visiting vendor stops in order. On arrival at the next
// stop, fire VisitVendorsInRoom + emit flavor + advance the index. When
// all stops done, transition to the depot dwell state.
//
// Same hostile-halt as tickTransit: if any hated mob is in the room,
// stop and let combat resolve before continuing the route.
func tickRoute(cur caravan.CaravanState, mob *mobs.Mob, ctx *EvalContext) Result {
	route := caravan.RouteForState(cur)
	if route == nil {
		return Failure
	}
	if partyHostilesNearby(ctx.InstanceId) || anyPartyMemberInCombat(ctx.InstanceId) {
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
		// Arrived at this stop — transfer items + advance index.
		wagon := caravan.FindWagonInRoom(nextRoom)
		if wagon == nil {
			// Wagon not present — caravan is wiped, mid-respawn,
			// or a follower lagged. Let legacy idle handle this tick;
			// next tick will retry.
			return Failure
		}
		deliveryBuckets, pickupBuckets := bucketsForRouteState(cur)
		delivered, pickedUp := caravan.VisitVendorsInRoom(
			nextRoom, wagon, deliveryBuckets, pickupBuckets,
		)
		if msg := caravan.FormatVisitMessage(delivered, pickedUp); msg != "" {
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

// hostilesInRoom reports whether any other mob in the given room is on
// the caller's hates list (via mob.HatesMob). Used by transit/route
// ticks to halt movement so the legacy idle path can fire
// lookfortrouble + engage combat.
func hostilesInRoom(mob *mobs.Mob, roomId int) bool {
	if len(mob.Hates) == 0 {
		return false
	}
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return false
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		if instId == mob.InstanceId {
			continue
		}
		other := mobs.GetInstance(instId)
		if other == nil {
			continue
		}
		if mob.HatesMob(other) {
			return true
		}
	}
	return false
}

// partyHostilesNearby is the party-wide variant of hostilesInRoom.
// Pause-the-caravan check: if ANY party member (leader or follower)
// is in a room with a mob on the caller's hates list, return true.
//
// Without this, a follower lagging into a hostile room could be
// jumped while the leader walks on — the caravan would resume
// movement because the leader's room is clear, even though the
// follower is mid-fight or about to die. Stage 3.4 caught this in
// playtest: lookout aggroed Lars, Lars's btree didn't auto-set
// his own Aggro back, anyPartyMemberInCombat returned false, and
// the caravan abandoned him.
//
// Solo (no party) callers fall back to checking just their own room.
func partyHostilesNearby(callerInstId int) bool {
	caller := mobs.GetInstance(callerInstId)
	if caller == nil {
		return false
	}
	if len(caller.Hates) == 0 {
		return false
	}
	p := parties.GetByMobInstanceId(callerInstId)
	if p == nil {
		return hostilesInRoom(caller, caller.Character.RoomId)
	}
	seen := map[int]bool{}
	for _, m := range p.Members {
		c := m.GetCharacter()
		if c == nil {
			continue
		}
		if seen[c.RoomId] {
			continue
		}
		seen[c.RoomId] = true
		if hostilesInRoom(caller, c.RoomId) {
			return true
		}
	}
	return false
}

// anyPartyMemberInCombat reports whether any member of the caller's
// party currently has aggro set. Used to keep the caravan leader from
// walking away while a follower is mid-fight in another room.
func anyPartyMemberInCombat(callerInstId int) bool {
	p := parties.GetByMobInstanceId(callerInstId)
	if p == nil {
		return false
	}
	for _, m := range p.Members {
		c := m.GetCharacter()
		if c != nil && c.Aggro != nil {
			return true
		}
	}
	return false
}

// findCaravanWagon returns the wagon mob (mobId 374) belonging to the
// caravan party led by callerInstId. Returns nil if the wagon is not
// found (e.g. mid-respawn or party not yet formed).
func findCaravanWagon(callerInstId int) *mobs.Mob {
	p := parties.GetByMobInstanceId(callerInstId)
	if p == nil {
		return nil
	}
	for _, m := range p.Members {
		instId := m.GetMobInstanceId()
		if instId == 0 {
			continue // player member
		}
		mob := mobs.GetInstance(instId)
		if mob != nil && int(mob.MobId) == 374 {
			return mob
		}
	}
	return nil
}

// caravanPluralize returns word+"s" when n != 1, else word unchanged.
func caravanPluralize(word string, n int) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// bucketsForRouteState returns the (delivery, pickup) bucket lists
// for the caravan at a vendor stop in the given route state.
//
// Outbound at Stillwater vendors: deliver thornwall + fernway
// (items the wagon carries from Thornwall + Fernway pickup), pick
// up stillwater (Stillwater-unique surplus for return trip).
//
// Inbound at Thornwall vendors: mirror — deliver stillwater +
// fernway, pick up thornwall.
//
// Other states (dwell, transit, fernway pickup) have no vendor
// transfer logic, so we return (nil, nil) defensively.
func bucketsForRouteState(s caravan.CaravanState) (delivery, pickup []string) {
	switch s {
	case caravan.StateStillwaterRoute:
		return []string{"thornwall", "fernway"}, []string{"stillwater"}
	case caravan.StateThornwallRoute:
		return []string{"stillwater", "fernway"}, []string{"thornwall"}
	}
	return nil, nil
}

