package ferry

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// FactorPhase is the trade factor's lifecycle state.
type FactorPhase int

const (
	FactorWaiting FactorPhase = iota
	FactorAboard
	FactorDelivering
	FactorReturning
)

// factorState is a trade factor's mutable progress through its circuit,
// keyed by mob instance id in factorStates. Phase/PortIdx/StopIdx are the
// pure-decision contract (see factorDecide); the remaining fields are
// impure-shell bookkeeping only (walk-reissue throttling, stuck detection).
type factorState struct {
	Phase   FactorPhase
	PortIdx int // port being serviced (Delivering/Returning) or waited at (Waiting)
	StopIdx int // next/current index into PortStops[PortIdx]

	walkIssued          bool
	lastWalkIssuedRound uint64
	lastRoom            int
	lastProgressRound   uint64
}

// factorPos is the factor's physical position this round, as observed by
// the impure shell and handed to the pure core.
type factorPos struct {
	AtPortIdx int // 0/1 when standing in that port's dock room, else -1
	OnDeck    bool
	InRoom    int // current room id (used during Delivering)
}

// FactorActionKind is the action factorDecide recommends this round.
type FactorActionKind int

const (
	ActNone FactorActionKind = iota
	ActBoard
	ActDisembark
	ActWalkTo      // issue pathto d.NextStop
	ActDeliverHere // deliver at current room, then walk to d.NextStop (or dock if LastStop)
)

type factorAction struct {
	Kind     FactorActionKind
	PortIdx  int
	NextStop int
	LastStop bool
}

// factorDecide is pure: given the circuit, the vessel state for its route,
// the factor's current physical position, and its machine state, return
// the action to take this round. No world mutation, no globals — fully
// unit-testable (see factor_test.go, which pins this contract).
func factorDecide(c TradeCircuit, vs VesselState, pos factorPos, st factorState) factorAction {
	switch st.Phase {
	case FactorWaiting:
		if vs.Docked && vs.PortIdx == st.PortIdx && pos.AtPortIdx == st.PortIdx {
			return factorAction{Kind: ActBoard, PortIdx: st.PortIdx}
		}
	case FactorAboard:
		if vs.Docked && pos.OnDeck {
			return factorAction{Kind: ActDisembark, PortIdx: vs.PortIdx}
		}
	case FactorDelivering:
		stops := c.PortStops[st.PortIdx]
		if st.StopIdx >= len(stops) {
			return factorAction{Kind: ActWalkTo, PortIdx: st.PortIdx, LastStop: true} // NextStop filled by caller with dock room
		}
		cur := stops[st.StopIdx]
		if pos.InRoom == cur {
			next := 0
			last := st.StopIdx == len(stops)-1
			if !last {
				next = stops[st.StopIdx+1]
			}
			return factorAction{Kind: ActDeliverHere, PortIdx: st.PortIdx, NextStop: next, LastStop: last}
		}
		return factorAction{Kind: ActWalkTo, PortIdx: st.PortIdx, NextStop: cur}
	case FactorReturning:
		if pos.AtPortIdx == st.PortIdx {
			return factorAction{Kind: ActNone} // shell transitions to Waiting + reload
		}
	}
	return factorAction{Kind: ActNone}
}

// ─── Impure shell ────────────────────────────────────────────────────────

// factorWalkReissueRounds throttles re-issuing "pathto" while a factor is
// already mid-path, so a stalled path retries periodically instead of
// spamming the mob's command queue every round.
const factorWalkReissueRounds = 5

// factorStuckRounds is how long a factor's room can go unchanged while
// Delivering/Returning before the stuck-safety reset kicks in.
const factorStuckRounds = 60

// factorStates is in-memory only. On restart, factors are re-discovered in
// their spawn rooms (anchored spawninfo) and re-enter the loop as Waiting
// (or Returning, if found mid-world).
var factorStates = map[int]*factorState{} // mob instance id → state

// portIdxOfRoom returns the port index (0 or 1) whose dock_room matches
// roomId, or -1 if roomId is neither port's dock.
func portIdxOfRoom(r Route, roomId int) int {
	for i, p := range r.Ports {
		if p.DockRoom == roomId {
			return i
		}
	}
	return -1
}

// tickFactors drives every registered trade circuit's factor one step.
// Called once per round from Tick(), after the vessel/gangplank loop.
func tickFactors(rpd int, now uint64) {
	for routeId, c := range tradeCircuits {
		r, ok := RouteFor(routeId)
		if !ok {
			continue // route not loaded (data-optional content)
		}
		homeDock := r.Ports[c.HomePortIdx].DockRoom
		factor := mobs.FindLiveInstanceByHomeAndId(homeDock, mobs.MobId(c.FactorMobId))
		if factor == nil {
			continue // not spawned (yet); anchored spawninfo will bring it back
		}

		st, ok := factorStates[factor.InstanceId]
		if !ok {
			st = &factorState{Phase: FactorWaiting, PortIdx: portIdxOfRoom(r, factor.Character.RoomId)}
			if st.PortIdx < 0 { // mid-world after restart: walk home
				st.Phase = FactorReturning
				st.PortIdx = c.HomePortIdx
			}
			st.lastRoom = factor.Character.RoomId
			st.lastProgressRound = now
			factorStates[factor.InstanceId] = st
			ensureFactorLoaded(factor, c, st.PortIdx)
			if st.Phase == FactorReturning {
				// Nothing else will kick off movement for a mid-world
				// recovery — issue the walk-home command once here.
				issueWalkIfDue(factor, st, r.Ports[st.PortIdx].DockRoom, now)
			}
		}

		// Stuck-safety: if the factor's room hasn't advanced in
		// factorStuckRounds while it's supposed to be walking somewhere,
		// teleport it back to the current port's dock and recover as if
		// it had just arrived there. Quiet (log-only) by design — this is
		// a recovery path, not narrative content.
		if factor.Character.RoomId != st.lastRoom {
			st.lastRoom = factor.Character.RoomId
			st.lastProgressRound = now
		} else if (st.Phase == FactorDelivering || st.Phase == FactorReturning) &&
			now-st.lastProgressRound >= factorStuckRounds {
			mudlog.Warn(`ferry.factor`, `warn`, `stuck-reset: teleporting factor back to dock`,
				`factor`, factor.Character.Name, `room`, factor.Character.RoomId, `phase`, int(st.Phase))
			dock := r.Ports[st.PortIdx].DockRoom
			moveFactorSilently(factor, factor.Character.RoomId, dock)
			st.Phase = FactorWaiting
			st.walkIssued = false
			st.lastRoom = factor.Character.RoomId
			st.lastProgressRound = now
			ensureFactorLoaded(factor, c, st.PortIdx)
			continue // state resolved for this round
		}

		vs := StateAt(r, now, rpd)
		pos := factorPos{
			AtPortIdx: portIdxOfRoom(r, factor.Character.RoomId),
			OnDeck:    factor.Character.RoomId == r.DeckRoom,
			InRoom:    factor.Character.RoomId,
		}

		// FactorReturning: factorDecide only reports the arrival edge (see
		// its ActNone default) — the walk-there command was already issued
		// once, at the moment the shell entered Returning. Check arrival
		// directly here so we can transition to Waiting + reload.
		if st.Phase == FactorReturning && pos.AtPortIdx == st.PortIdx {
			st.Phase = FactorWaiting
			st.walkIssued = false
			ensureFactorLoaded(factor, c, st.PortIdx)
			continue
		}

		act := factorDecide(c, vs, pos, *st)
		applyFactorAction(factor, r, c, st, act, now)
	}
}

// applyFactorAction performs the world mutation (or command issuance)
// factorDecide recommended.
func applyFactorAction(factor *mobs.Mob, r Route, c TradeCircuit, st *factorState, act factorAction, now uint64) {
	switch act.Kind {
	case ActBoard:
		transportFactor(factor, factor.Character.RoomId, r.DeckRoom,
			fmt.Sprintf(`%s leads a laden handcart up the gangplank.`, factor.Character.Name),
			fmt.Sprintf(`%s wheels a laden handcart aboard and settles it against the rail.`, factor.Character.Name))
		st.Phase = FactorAboard
		st.walkIssued = false

	case ActDisembark:
		dock := r.Ports[act.PortIdx].DockRoom
		transportFactor(factor, factor.Character.RoomId, dock,
			fmt.Sprintf(`%s wheels a laden handcart down the gangplank.`, factor.Character.Name),
			fmt.Sprintf(`%s trundles ashore with a laden handcart, tally-slip in hand.`, factor.Character.Name))
		st.Phase = FactorDelivering
		st.PortIdx = act.PortIdx
		st.StopIdx = 0
		st.walkIssued = false

	case ActWalkTo:
		target := act.NextStop
		if act.LastStop {
			// Defensive fallback for the "stops exhausted" branch of
			// factorDecide — normally ActDeliverHere's LastStop handling
			// below transitions to Returning before this can be reached.
			target = r.Ports[act.PortIdx].DockRoom
			st.Phase = FactorReturning
		}
		issueWalkIfDue(factor, st, target, now)

	case ActDeliverHere:
		delivered, _ := caravan.VisitVendorsInRoomOpts(factor.Character.RoomId, factor, caravan.VisitOpts{
			DeliveryBuckets:    c.PortDeliveryBuckets[st.PortIdx],
			CreateMissingSlots: true,
			NewSlotMaxStock:    c.NewSlotMaxStock,
		})
		if msg := caravan.FormatVisitMessage(factor.Character.Name, delivered, nil); msg != `` {
			if room := rooms.LoadRoom(factor.Character.RoomId); room != nil {
				room.SendText(messaging.CategoryRoomDescription, msg)
			}
		}
		st.StopIdx++
		st.walkIssued = false
		if act.LastStop {
			st.Phase = FactorReturning
			issueWalkIfDue(factor, st, r.Ports[st.PortIdx].DockRoom, now)
		}

	case ActNone:
		// Nothing to do this round.
	}
}

// issueWalkIfDue sends the factor toward targetRoom via the same "pathto"
// plumbing schedules/patrols use, throttled to at most once every
// factorWalkReissueRounds so a mid-path factor doesn't get spammed with
// redundant pathto commands every round.
func issueWalkIfDue(factor *mobs.Mob, st *factorState, targetRoom int, now uint64) {
	if st.walkIssued && now < st.lastWalkIssuedRound+factorWalkReissueRounds {
		return
	}
	factor.Command(fmt.Sprintf(`pathto %d`, targetRoom))
	st.walkIssued = true
	st.lastWalkIssuedRound = now
}

// moveFactorSilently performs the bare RemoveMob/AddMob/RoomId transport
// triple (the TransportCompanions pattern, internal/hooks/companion_follow.go)
// with no room emotes. Used by the stuck-safety recovery path, which is
// deliberately quiet (log-only) rather than narrated.
func moveFactorSilently(factor *mobs.Mob, oldRoomId, newRoomId int) {
	if oldRoomId == newRoomId {
		return
	}
	if curRoom := rooms.LoadRoom(oldRoomId); curRoom != nil {
		curRoom.RemoveMob(factor.InstanceId)
	}
	destRoom := rooms.LoadRoom(newRoomId)
	if destRoom == nil {
		mudlog.Error(`ferry.moveFactorSilently`, `error`,
			fmt.Sprintf(`destination room %d not found for factor %d`, newRoomId, factor.InstanceId))
		return
	}
	destRoom.AddMob(factor.InstanceId)
	factor.Character.RoomId = newRoomId
}

// transportFactor is moveFactorSilently plus a departure emote in the old
// room and an arrival emote in the new room — the visible "the factor is
// alive" narration for boarding/disembarking.
func transportFactor(factor *mobs.Mob, oldRoomId, newRoomId int, departEmote, arriveEmote string) {
	if oldRoomId == newRoomId {
		return
	}
	if curRoom := rooms.LoadRoom(oldRoomId); curRoom != nil {
		curRoom.SendText(messaging.CategoryMobEmote, departEmote)
	}
	moveFactorSilently(factor, oldRoomId, newRoomId)
	if destRoom := rooms.LoadRoom(newRoomId); destRoom != nil {
		destRoom.SendText(messaging.CategoryMobEmote, arriveEmote)
	}
}

// ensureFactorLoaded tops the factor's inventory up to c.LoadCap with fresh
// items drawn round-robin from c.PortExports[portIdx] — mirrors
// caravan.LoadRunnerFromImport's bounded-retry pattern. Returns the number
// of items loaded.
func ensureFactorLoaded(factor *mobs.Mob, c TradeCircuit, portIdx int) int {
	if factor == nil || portIdx < 0 || portIdx > 1 {
		return 0
	}
	manifest := c.PortExports[portIdx]
	if len(manifest) == 0 || c.LoadCap <= 0 {
		return 0
	}
	loaded := 0
	maxTries := c.LoadCap*2 + len(manifest) // bound: avoids spin on invalid ids
	for tries := 0; tries < maxTries && len(factor.Character.Items) < c.LoadCap; tries++ {
		it := items.New(manifest[tries%len(manifest)])
		if !it.IsValid() {
			continue
		}
		if !factor.Character.StoreItem(it) {
			break // factor at carry capacity
		}
		loaded++
	}
	return loaded
}
