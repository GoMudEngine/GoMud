package ferry

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/warehouse"
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
		// vs.PortIdx != st.PortIdx guards against disembarking back at the
		// boarding port: the vessel stays docked for the whole layover after
		// the factor boards, so without this the factor would step ashore
		// the very next round and never sail anywhere. st.PortIdx still
		// holds the boarding port while Aboard (set when Waiting; only
		// overwritten by the ActDisembark handling).
		if vs.Docked && pos.OnDeck && vs.PortIdx != st.PortIdx {
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

// warehouseReleaseMaxPerItem bounds Stage 4's delivery-time local release
// (see ReleaseToVendorsInRoom below) — deliberately slow per the spec, so a
// warehouse-adjacent vendor gap closes gradually across several factor
// visits rather than refilling in one pass.
const warehouseReleaseMaxPerItem = 2

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
	seen := make(map[int]bool, len(tradeCircuits))
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
		seen[factor.InstanceId] = true

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
			// Stuck-reset ends the circuit here too (the factor never makes
			// it back to a normal Returning→Waiting arrival) — bank leftover
			// cargo symmetrically with that path before reloading.
			bankLeftoverCargo(factor, dock)
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
		// its ActNone default) — the shell owns this phase. On arrival,
		// transition to Waiting + reload. While still en route, keep the
		// walk alive with the same throttled reissue Delivering gets, so a
		// dropped path recovers within factorWalkReissueRounds instead of
		// waiting out the 60-round stuck teleport.
		if st.Phase == FactorReturning {
			if pos.AtPortIdx == st.PortIdx {
				st.Phase = FactorWaiting
				st.walkIssued = false
				// End-of-circuit overflow capture (Stage 3): bank whatever
				// the factor is still carrying before topping it back up.
				bankLeftoverCargo(factor, r.Ports[st.PortIdx].DockRoom)
				ensureFactorLoaded(factor, c, st.PortIdx)
			} else {
				issueWalkIfDue(factor, st, r.Ports[st.PortIdx].DockRoom, now)
			}
			continue
		}

		act := factorDecide(c, vs, pos, *st)
		applyFactorAction(factor, r, c, st, act, now)
	}

	// Evict stale entries: factorStates is keyed by mob instance id, and
	// instance ids churn on despawn/respawn (room unload, server restart
	// recovery, admin recall). Without this, a despawned factor's old
	// instance id lingers forever as dead weight in the map.
	for instId := range factorStates {
		if !seen[instId] {
			delete(factorStates, instId)
		}
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
		// Stage 4: delivery-time local release. After the factor's own
		// cargo lands, top up any REMAINING vendor gaps in this room from
		// the local warehouse (bounded, slow — no emote for the invisible
		// backend top-up; the delivery message above already narrates the
		// stop).
		if bool(configs.GetGamePlayConfig().WarehousesEnabled) && bool(configs.GetGamePlayConfig().WarehouseDrawdownEnabled) {
			if room := rooms.LoadRoom(factor.Character.RoomId); room != nil {
				warehouse.ReleaseToVendorsInRoom(room.Zone, factor.Character.RoomId, warehouseReleaseMaxPerItem)
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

	// Self-heal: clear any stuck/lost flags mobcommands.Pathto may have set
	// on this factor. Before the idle-hook exemption (shouldRecoverDisplacedHome,
	// internal/hooks/MobIdle_HandleIdleMobs.go), a factor standing on an
	// exitless vessel deck would get "pathto home" issued every idle tick,
	// which can never resolve there — Pathto flags `home-impossible` in
	// TempData and sets the `lost` adjective, then drains 10% max HP per
	// tick forever (2026-07-03 playtest, BUG-1). This is the single choke
	// point every factor move passes through (board/disembark/deliver/
	// return/stuck-teleport), so clearing here guarantees any factor that
	// was ever flagged — by this bug or any future edge case — recovers on
	// its very next transport instead of staying flagged and bleeding HP.
	factor.SetTempData(`home-impossible`, nil)
	factor.Character.SetAdjective(`lost`, false)
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

// FactorPhaseName returns a human-readable state string for the economy
// dashboard, and ok=false when instanceId isn't a currently-registered
// trade factor (e.g. despawned, or never a factor at all).
func FactorPhaseName(instanceId int) (string, bool) {
	st, ok := factorStates[instanceId]
	if !ok {
		return ``, false
	}
	switch st.Phase {
	case FactorWaiting:
		return `waiting_at_dock`, true
	case FactorAboard:
		return `aboard`, true
	case FactorDelivering:
		return `delivering`, true
	case FactorReturning:
		return `returning_to_dock`, true
	}
	return `unknown`, true
}

// SetFactorStateForTest registers a minimal factorStates entry for
// instanceId with the given phase (PortIdx/StopIdx default to zero).
// Test-only: lets economy/health capture tests exercise FactorPhaseName /
// captureFerryFactors without driving the full tickFactors machinery.
func SetFactorStateForTest(instanceId int, phase FactorPhase) {
	factorStates[instanceId] = &factorState{Phase: phase}
}

// ClearFactorStateForTest removes a test-registered factor state. Pair
// with SetFactorStateForTest via t.Cleanup so tests don't leak state into
// the package-level factorStates map between runs.
func ClearFactorStateForTest(instanceId int) {
	delete(factorStates, instanceId)
}

// bankLeftoverCargo consigns a factor's undelivered bucketed cargo to the
// dock city's warehouse (Stage 3 overflow capture). Called ONLY at the
// end of a circuit (Returning→Waiting, or the stuck-reset path that ends
// a circuit early), never mid-circuit — later stops still need mid-circuit
// cargo. Non-warehouse cities (Stillwater): no-op, leftovers stay aboard
// and count against the reload cap as before. Gated on WarehousesEnabled
// so the whole feature is a no-op with the master toggle off (warehouse.
// Deposit itself doesn't check the toggle, and with it off SaveDirty never
// runs — so silently accumulating stock here would just leak memory).
func bankLeftoverCargo(factor *mobs.Mob, dockRoomId int) {
	if !bool(configs.GetGamePlayConfig().WarehousesEnabled) {
		return
	}
	dock := rooms.LoadRoom(dockRoomId)
	if dock == nil {
		return
	}
	if _, ok := warehouse.CityFor(dock.Zone); !ok {
		return
	}
	banked := 0
	for i := len(factor.Character.Items) - 1; i >= 0; i-- {
		it := factor.Character.Items[i]
		if warehouse.Deposit(dock.Zone, it.ItemId, 1) {
			factor.Character.RemoveItem(it)
			banked++
		}
	}
	if banked > 0 {
		dock.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(`%s consigns the unsold remainder of the cargo to the warehouse.`, factor.Character.Name))
	}
}

// ensureFactorLoaded tops the factor's inventory up to c.LoadCap. Stage 4:
// when warehouse drawdown is enabled and the loading port's dock city has a
// warehouse, a first pass (loadFromWarehouse) draws manifest items from the
// local warehouse — prioritized by downstream vendor demand — before the
// unchanged manifest round-robin (mirrors caravan.LoadRunnerFromImport's
// bounded-retry pattern) tops up any remaining capacity from the infinite
// trickle. Returns the total number of items loaded.
func ensureFactorLoaded(factor *mobs.Mob, c TradeCircuit, portIdx int) int {
	if factor == nil || portIdx < 0 || portIdx > 1 {
		return 0
	}
	manifest := c.PortExports[portIdx]
	if len(manifest) == 0 || c.LoadCap <= 0 {
		return 0
	}

	loaded := 0
	if bool(configs.GetGamePlayConfig().WarehousesEnabled) && bool(configs.GetGamePlayConfig().WarehouseDrawdownEnabled) {
		if dock := portDockRoom(c, portIdx); dock != nil {
			if _, ok := warehouse.CityFor(dock.Zone); ok {
				loaded += loadFromWarehouse(factor, c, portIdx, dock.Zone)
			}
		}
	}

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

// loadFromWarehouse draws manifest items (c.PortExports[portIdx]) from
// dockZone's warehouse, prioritized by downstream vendor demand, minting
// the physical item for each unit successfully withdrawn and storing it on
// the factor. On a StoreItem failure (factor at carry capacity), the
// withdrawn unit is re-deposited rather than destroyed — Stage 4 never
// loses stock to a full cart. Draws round-robin over the priority order,
// bounded like the sibling loaders, until capacity or warehouse stock (for
// every manifest item) is exhausted. Returns the number of items loaded.
func loadFromWarehouse(factor *mobs.Mob, c TradeCircuit, portIdx int, dockZone string) int {
	manifest := c.PortExports[portIdx]
	priority := prioritizeByDemand(manifest, exportDemand(c, portIdx))
	loaded := 0
	maxTries := c.LoadCap*2 + len(priority)
	for tries := 0; tries < maxTries && len(factor.Character.Items) < c.LoadCap; tries++ {
		itemId := priority[tries%len(priority)]
		n := warehouse.Withdraw(dockZone, itemId, 1)
		if n <= 0 {
			continue
		}
		it := items.New(itemId)
		if !it.IsValid() {
			// Can't mint the physical item — re-deposit so the unit isn't
			// silently lost (Deposit re-counts it as captured, acceptable).
			warehouse.Deposit(dockZone, itemId, 1)
			continue
		}
		if !factor.Character.StoreItem(it) {
			warehouse.Deposit(dockZone, itemId, 1)
			break // factor at carry capacity
		}
		loaded++
	}
	return loaded
}

// portDockRoom resolves the dock room for c's port at portIdx, via the
// registered Route. Returns nil if the route isn't loaded or the room
// doesn't exist (data-optional content, mirrors the tickFactors nil-guard).
func portDockRoom(c TradeCircuit, portIdx int) *rooms.Room {
	r, ok := RouteFor(c.RouteId)
	if !ok {
		return nil
	}
	return rooms.LoadRoom(r.Ports[portIdx].DockRoom)
}

// prioritizeByDemand returns manifest items sorted by downstream demand
// (desc), ties keeping manifest order. Pure.
func prioritizeByDemand(manifest []int, demand map[int]int) []int {
	out := append([]int(nil), manifest...)
	sort.SliceStable(out, func(a, b int) bool {
		return demand[out[a]] > demand[out[b]]
	})
	return out
}

// exportDemand sums downstream vendor gaps for each export item of the
// given port: existing entries contribute MaxStock-Current; a vendor
// missing the entry contributes NewSlotMaxStock (CreateMissingSlots will
// open one on delivery). Unspawned vendors / unloaded rooms contribute 0
// — clean fallback to manifest order at cold boot.
func exportDemand(c TradeCircuit, portIdx int) map[int]int {
	demand := map[int]int{}
	for _, stop := range c.PortStops[1-portIdx] {
		room := rooms.LoadRoom(stop)
		if room == nil {
			continue
		}
		for _, instId := range room.GetMobs(rooms.FindMerchant) {
			vendor := mobs.GetInstance(instId)
			if vendor == nil || !vendor.HasShop() {
				continue
			}
			shop := shops.GetShopInventory(vendor.Zone, int(vendor.MobId), vendor.HomeRoomId)
			if shop == nil {
				continue
			}
			for _, itemId := range c.PortExports[portIdx] {
				if e := shop.GetStock(itemId); e != nil {
					if gap := e.MaxStock - e.Current; gap > 0 {
						demand[itemId] += gap
					}
				} else {
					demand[itemId] += c.NewSlotMaxStock
				}
			}
		}
	}
	return demand
}
