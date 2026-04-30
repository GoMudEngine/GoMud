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
//   caravan_load               — comma-joined supply buckets the caravan carries

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// caravan_load — comma-joined list of supply buckets the caravan
// currently carries. Set to ["stillwater"] when entering the
// stillwater_route, ["thornwall"] when entering the thornwall_route.
// On a successful Fernway forager handoff at room 4038, "fernway" is
// appended. Cleared on entry to a depot dwell state.
const keyCaravanLoad = "caravan_load"

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

// tickDwell: at depot, waiting for the dwell timer to elapse.
//
// Returns Success only when the dwell timer expires and the state actually
// advances. While still waiting, returns Failure so the btree falls through
// to legacy idle (which fires the mob's idlecommands and lookfortrouble).
// Without this, the caravan crew sits silent at the depot for the entire
// dwell — no flavor emotes from idlecommands.
//
// On the first tick of depot dwell, clears caravan_load so stale bucket
// data from the previous cycle doesn't bleed into the next one.
func tickDwell(cur caravan.CaravanState, mob *mobs.Mob, ctx *EvalContext) Result {
	startedStr := ctx.MobState.GetString("caravan_state_started_round")
	started, _ := strconv.ParseUint(startedStr, 10, 64)

	// On entry to a depot dwell, clear any leftover caravan_load.
	// (transitionTo doesn't clear it because the route phase needs
	// to consume it across multiple ticks.)
	if util.GetRoundCount() == started {
		caravanLoadSet(ctx.MobState, nil)
	}

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
	if hostilesInRoom(mob, ctx.RoomId) || anyPartyMemberInCombat(ctx.InstanceId) {
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
// to receive the Fernway forager's goods.
const fernwayMeetingRoomId = 4038

// fernwayForagerMobId is the mob template ID of the Fernway forager
// that triggers the caravan_load handoff.
const fernwayForagerMobId = 373

// tickFernwayPickup: brief dwell at North Road 4038 to detect the
// Fernway forager and acquire the fernway bucket. Lives between the
// transit and route phases of each cycle.
//
// Behavior:
//  1. If not at room 4038 yet, pathto it.
//  2. On arrival, dwell up to FernwayPickupDwellRounds. On the first
//     tick at the room, scan for the Fernway forager (see
//     fernwayForagerMobId). If present, append "fernway" to
//     caravan_load and emit a flavor message.
//  3. After dwell expires, advance to the next state.
//
// Returns Success while making progress. Returns Failure when at
// room 4038 dwelling so legacy idle path can fire flavor emotes.
func tickFernwayPickup(cur caravan.CaravanState, mob *mobs.Mob, ctx *EvalContext) Result {
	if hostilesInRoom(mob, ctx.RoomId) || anyPartyMemberInCombat(ctx.InstanceId) {
		return Failure
	}

	// Not at meeting point — pathto.
	if ctx.RoomId != fernwayMeetingRoomId {
		mob.Command(fmt.Sprintf("pathto %d", fernwayMeetingRoomId))
		return Success
	}

	// At meeting point. Check forager presence on the first tick (elapsed==0).
	// caravan_state_started_round was set by transitionTo when we entered
	// this state.
	startedStr := ctx.MobState.GetString("caravan_state_started_round")
	started, _ := strconv.ParseUint(startedStr, 10, 64)
	now := util.GetRoundCount()
	elapsed := now - started

	if elapsed == 0 {
		if foragerInRoom(ctx.RoomId, fernwayForagerMobId) {
			caravanLoadAppend(ctx.MobState, "fernway")
			if r := rooms.LoadRoom(ctx.RoomId); r != nil {
				r.SendText(`<ansi fg="yellow">A wiry forager hands a satchel up to the caravan; the wagon takes the load and rolls on.</ansi>`)
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
// On the first tick of each route phase, appends the destination bucket
// ("stillwater" or "thornwall") to caravan_load, preserving any "fernway"
// bucket already appended during the pickup substate.
//
// Same hostile-halt as tickTransit: if any hated mob is in the room,
// stop and let combat resolve before continuing the route.
func tickRoute(cur caravan.CaravanState, mob *mobs.Mob, ctx *EvalContext) Result {
	route := caravan.RouteForState(cur)
	if route == nil {
		return Failure
	}
	if hostilesInRoom(mob, ctx.RoomId) || anyPartyMemberInCombat(ctx.InstanceId) {
		return Failure
	}

	// On first tick of this route phase, set the destination bucket.
	// Fernway (if picked up) was already appended during the pickup
	// substate — preserve it.
	startedStr := ctx.MobState.GetString("caravan_state_started_round")
	started, _ := strconv.ParseUint(startedStr, 10, 64)
	if util.GetRoundCount() == started {
		switch cur {
		case caravan.StateStillwaterRoute:
			caravanLoadAppend(ctx.MobState, "stillwater")
		case caravan.StateThornwallRoute:
			caravanLoadAppend(ctx.MobState, "thornwall")
		}
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
		wagon := findWagonInRoom(nextRoom)
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

// caravanLoadGet returns the current caravan_load buckets. Returns
// nil if none are set.
func caravanLoadGet(s *BehaviorState) []string {
	raw := s.GetString(keyCaravanLoad)
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
}

// caravanLoadSet writes the buckets verbatim, comma-joined. Empty
// list clears the field.
func caravanLoadSet(s *BehaviorState, buckets []string) {
	if len(buckets) == 0 {
		s.Set(keyCaravanLoad, "")
		return
	}
	s.Set(keyCaravanLoad, strings.Join(buckets, ","))
}

// caravanLoadAppend adds a bucket if not already present. No-op if
// already present.
func caravanLoadAppend(s *BehaviorState, bucket string) {
	cur := caravanLoadGet(s)
	for _, b := range cur {
		if b == bucket {
			return
		}
	}
	caravanLoadSet(s, append(cur, bucket))
}

// findWagonInRoom returns the caravan wagon mob (mob 374) if it's
// in the given room. Returns nil if the wagon isn't there — which
// happens during the brief windows when followers are catching up,
// or when the caravan has been wiped and is mid-respawn.
//
// Caller decides whether nil is fatal (return Failure to let legacy
// idle take over) or recoverable (skip this tick).
func findWagonInRoom(roomId int) *mobs.Mob {
	const wagonMobId = 374
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return nil
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		if int(m.MobId) == wagonMobId {
			return m
		}
	}
	return nil
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

// foragerInRoom reports whether a mob with the given mobId is in
// the given room. Used by the caravan to detect the Fernway forager
// at the meeting point.
func foragerInRoom(roomId, mobId int) bool {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return false
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		if int(m.MobId) == mobId {
			return true
		}
	}
	return false
}
