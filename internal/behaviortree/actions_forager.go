package behaviortree

// actions_forager.go — Stage 3.1 forager state-machine btree action.
//
// forager_step is the workhorse action that drives a forager NPC's
// daily cycle (forage → deliver → recall → rest → repeat). Mirrors
// the caravan_step pattern from actions_caravan.go.
//
// Forage roll logic and yield tables live in internal/forager/forage_core.go
// (the leaf forager package), shared by both this file and
// usercommands/skill.forage.go.

import (
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/economy"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/sealedcrate"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func init() {
	actionRegistry["forager_step"] = actForagerStep
}

const (
	keyForagerState      = "forager_state"
	keyStateStartedRound = "forager_state_started_round"
	keyForageTimer       = "forager_forage_timer"
	keyFatigueTimer      = "forager_fatigue_timer"
	keyVisitIndex        = "forager_visit_index"
	keyWaitTimer         = "forager_wait_timer"
)

func actForagerStep(params map[string]any, ctx *EvalContext) Result {
	if ctx.MobState == nil {
		return Failure
	}
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	profile := forager.ProfileFor(int(mob.MobId))
	if profile == nil {
		return Failure // not a registered forager
	}

	cur := readForagerState(ctx.MobState)
	cfg := configs.GetBalanceConfig()

	// HP-emergency short-circuit. Any state can drop to Recalling.
	// The cast itself fires on the next tick via tickForagerRecalling
	// — no need to issue it here too.
	if cur != forager.StateRecalling &&
		hpRatio(mob) <= float64(cfg.ForagerHPRecallThresholdPct) {
		transitionForager(ctx.MobState, forager.StateRecalling)
		return Success
	}

	// Stuck-state watchdog. If the forager has been sitting in one state
	// longer than the threshold, force-reset to Recalling so it heads
	// home, dumps satchel, and re-cycles. Logs a Warn for ops visibility.
	{
		startedStr := ctx.MobState.GetString(keyStateStartedRound)
		started, _ := strconv.ParseUint(startedStr, 10, 64)
		threshold := uint64(cfg.ForagerStuckThresholdRounds)
		now := util.GetRoundCount()
		// Guard against unsigned underflow when started has somehow ended
		// up in the future (clock-skew, replay, fresh-init race).
		if started > 0 && threshold > 0 && now > started &&
			now-started > threshold {
			mudlog.Warn("forager watchdog: stuck state, force-resetting to recalling",
				"mobId", mob.MobId, "name", profile.Name,
				"state", ctx.MobState.GetString(keyForagerState),
				"stuckRounds", now-started)
			transitionForager(ctx.MobState, forager.StateRecalling)
			return Success
		}
	}

	switch cur {
	case forager.StateResting:
		return tickForagerResting(profile, mob, ctx)
	case forager.StateTravelingToTerritory:
		return tickForagerTravelingToTerritory(profile, mob, ctx)
	case forager.StateForaging:
		return tickForagerForaging(profile, mob, ctx, cfg)
	case forager.StateTravelingToDropoff:
		return tickForagerTravelingToDropoff(profile, mob, ctx)
	case forager.StateDelivering:
		return tickForagerDelivering(profile, mob, ctx)
	case forager.StateRecalling:
		return tickForagerRecalling(profile, mob, ctx)
	}
	return Failure
}

// readForagerState reads MobState["forager_state"], defaulting to
// StateResting on first tick or unparseable input.
func readForagerState(s *BehaviorState) forager.ForagerState {
	raw := s.GetString(keyForagerState)
	if raw == "" {
		s.Set(keyForagerState, forager.StateResting.Name())
		s.Set(keyStateStartedRound,
			strconv.FormatUint(util.GetRoundCount(), 10))
		return forager.StateResting
	}
	parsed, ok := forager.ParseState(raw)
	if !ok {
		s.Set(keyForagerState, forager.StateResting.Name())
		s.Set(keyStateStartedRound,
			strconv.FormatUint(util.GetRoundCount(), 10))
		return forager.StateResting
	}
	return parsed
}

// transitionForager writes the new state to MobState and resets all
// per-state counters so they start fresh.
func transitionForager(s *BehaviorState, next forager.ForagerState) {
	s.Set(keyForagerState, next.Name())
	s.Set(keyStateStartedRound,
		strconv.FormatUint(util.GetRoundCount(), 10))
	s.Set(keyForageTimer, "0")
	s.Set(keyFatigueTimer, "0")
	s.Set(keyVisitIndex, "0")
	s.Set(keyWaitTimer, "0")
}

func hpRatio(mob *mobs.Mob) float64 {
	if mob.Character.HealthMax.Value <= 0 {
		return 1.0
	}
	return float64(mob.Character.Health) /
		float64(mob.Character.HealthMax.Value)
}

// ── State handlers ───────────────────────────────────────────────────
//
// Each handler returns Success when it did meaningful work (state
// transition issued or action queued). Returns Failure to fall through
// to the legacy idle path (idlecommands + lookfortrouble), matching
// the caravan_step pattern.

const restingDuration uint64 = 120

func tickForagerResting(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
) Result {
	if ctx.RoomId != p.SanctuaryRoom {
		// Off-sanctuary — path back before resting.
		mob.Command(fmt.Sprintf("pathto %d", p.SanctuaryRoom))
		return Success
	}
	startedStr := ctx.MobState.GetString(keyStateStartedRound)
	started, _ := strconv.ParseUint(startedStr, 10, 64)
	dwellElapsed := util.GetRoundCount() >= started+restingDuration
	if dwellElapsed && mob.Character.Health >= mob.Character.HealthMax.Value {
		// 2026-05-02: Stage 3.4 carry-ratio gate retained as a
		// BACKSTOP. The primary deadlock-avoidance mechanism is now
		// dumpSatchelToLockbox in tickForagerRecalling, which empties
		// the satchel on arrival. The carry-ratio check is only
		// reached when the lockbox itself was full and surplus
		// stayed in the satchel — in that case, continue resting
		// until the player picks the lockbox open and frees space.
		restThreshold := float64(configs.GetBalanceConfig().ForagerRestCarryThreshold)
		if carryRatio(mob) > restThreshold {
			return Failure
		}
		transitionForager(ctx.MobState, forager.StateTravelingToTerritory)
		return Success
	}
	// Still resting — let legacy idle fire flavor emotes.
	return Failure
}

func tickForagerTravelingToTerritory(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
) Result {
	if len(p.TerritoryRooms) == 0 {
		return Failure
	}
	if containsInt(p.TerritoryRooms, ctx.RoomId) {
		transitionForager(ctx.MobState, forager.StateForaging)
		return Success
	}
	mob.Command(fmt.Sprintf("pathto %d", p.TerritoryRooms[0]))
	return Success
}

const fatigueLimit = 480

func tickForagerForaging(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
	cfg configs.Balance,
) Result {
	// Fatigue tick.
	fatigue := getIntFromState(ctx.MobState, keyFatigueTimer) + 1
	ctx.MobState.Set(keyFatigueTimer, strconv.Itoa(fatigue))

	// Carry-cap or fatigue → head to dropoff.
	carry := carryRatio(mob)
	if fatigue >= fatigueLimit ||
		carry >= float64(cfg.ForagerCarryThresholdPct) {
		transitionForager(ctx.MobState, forager.StateTravelingToDropoff)
		return Success
	}

	// Forage tick.
	forageT := getIntFromState(ctx.MobState, keyForageTimer) + 1
	if forageT >= int(cfg.ForagerForageDwellRounds) {
		ctx.MobState.Set(keyForageTimer, "0")
		npcAttemptForage(p, mob, ctx)
	} else {
		ctx.MobState.Set(keyForageTimer, strconv.Itoa(forageT))
	}

	// Salvage any corpse in current room. Guard prevents dispatch noise
	// in rooms with no corpses; the handler also returns handled=true
	// silently on no-op, so this is belt-and-suspenders.
	if salvageRoom := rooms.LoadRoom(ctx.RoomId); salvageRoom != nil && len(salvageRoom.Corpses) > 0 {
		mob.Command("salvage corpse")
	}

	// Wander to a random adjacent territory neighbor.
	npcWanderTerritoryNeighbor(p, mob, ctx)

	// Return Failure to let the legacy idle path fire lookfortrouble,
	// which sets aggro on prey wildlife in the room — without that,
	// the mob_can_safely_engage condition would never have an aggro
	// target to evaluate. Mirror's the caravan tickDwell pattern.
	return Failure
}

func tickForagerTravelingToDropoff(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
) Result {
	var dest int
	switch p.Kind {
	case forager.KindFernway:
		dest = p.MeetingRoom
	default:
		if len(p.VendorRooms) == 0 {
			return Failure
		}
		dest = p.VendorRooms[0]
	}
	if ctx.RoomId == dest {
		transitionForager(ctx.MobState, forager.StateDelivering)
		return Success
	}
	mob.Command(fmt.Sprintf("pathto %d", dest))
	return Success
}

func tickForagerDelivering(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
) Result {
	if p.Kind == forager.KindFernway {
		return tickForagerDeliveringFernway(p, mob, ctx)
	}
	return tickForagerDeliveringTown(p, mob, ctx)
}

func tickForagerDeliveringTown(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
) Result {
	idx := getIntFromState(ctx.MobState, keyVisitIndex)
	if idx >= len(p.VendorRooms) {
		transitionForager(ctx.MobState, forager.StateRecalling)
		return Success
	}
	target := p.VendorRooms[idx]
	if ctx.RoomId != target {
		mob.Command(fmt.Sprintf("pathto %d", target))
		return Success
	}
	npcVisitVendorsInRoom(target, p, mob)
	ctx.MobState.Set(keyVisitIndex, strconv.Itoa(idx+1))
	return Success
}

func tickForagerDeliveringFernway(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
) Result {
	if ctx.RoomId != p.MeetingRoom {
		mob.Command(fmt.Sprintf("pathto %d", p.MeetingRoom))
		return Success
	}
	// Arrived at meeting room. Dump fernway-bucket items into the
	// sealed crate — Kessa drops the load and turns for home, no
	// wait for the caravan to coincide.
	room := rooms.LoadRoom(ctx.RoomId)
	if dumped := dumpFernwayLoadIntoCrate(p, mob, room); dumped > 0 {
		persistCrate(ctx.RoomId, room.SealedCrate)
		room.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> hauls a satchel up to the crate, latches it shut, and turns for home.`,
			p.Name))
	}
	// Always advance to Recalling — no more 150-round wait timer.
	transitionForager(ctx.MobState, forager.StateRecalling)
	return Success
}

// dumpFernwayLoadIntoCrate transfers fernway-bucket items from the
// forager's satchel into the room's sealed crate (in-memory only —
// persistence is the caller's responsibility). Returns the number
// of items dumped. Returns 0 if the room has no SealedCrate or the
// crate is full before any items match.
func dumpFernwayLoadIntoCrate(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	room *rooms.Room,
) int {
	if room == nil || room.SealedCrate == nil {
		return 0
	}
	crate := room.SealedCrate
	dumped := 0
	for i := len(mob.Character.Items) - 1; i >= 0; i-- {
		item := mob.Character.Items[i]
		bucket := economy.BucketFor(item.ItemId)
		if !slices.Contains(p.Buckets, bucket) {
			continue
		}
		if !crate.Add(item) {
			break // crate full
		}
		mob.Character.RemoveItem(item)
		dumped++
	}
	return dumped
}

// persistCrate writes the crate's current state to its YAML file at
// <DataFiles>/crates/<roomid>-fernway_shipment.yaml.
// Errors are logged but not surfaced — persistence is best-effort
// crash-safety, not a correctness requirement.
func persistCrate(roomId int, c *sealedcrate.Crate) {
	path := util.FilePath(
		configs.GetConfig().FilePaths.DataFiles.String(),
		fmt.Sprintf("/crates/%d-fernway_shipment.yaml", roomId),
	)
	if err := sealedcrate.SaveTo(path, c); err != nil {
		mudlog.Error("forager.persistCrate", "roomId", roomId, "error", err)
	}
}

func tickForagerRecalling(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
) Result {
	if ctx.RoomId == p.SanctuaryRoom {
		// Dump remaining satchel into the sanctuary lockbox before
		// resting. Surplus accumulates in the lockbox where players
		// can pick to retrieve it; the cycle resumes immediately
		// regardless of vendor saturation.
		dumpSatchelToLockbox(mob, ctx)
		transitionForager(ctx.MobState, forager.StateResting)
		return Success
	}

	// Teleport directly to anchor rather than issuing "cast fold-recall".
	// The fold-cast system only advances casts inside the combat loop
	// (gated on mob.Character.Aggro != nil), so foragers recalling
	// outside combat would be permanently stuck with IsCasting()=true
	// and never actually move. Direct room manipulation mirrors exactly
	// what resolveFoldRecall does internally for mobs.
	anchorRoomId := getMiscDataIntForager(mob, "fold-anchor-room")
	if anchorRoomId <= 0 {
		// No anchor set in MiscData — fall back to the profile sanctuary.
		anchorRoomId = p.SanctuaryRoom
	}
	if anchorRoomId > 0 && anchorRoomId != ctx.RoomId {
		fromRoom := rooms.LoadRoom(ctx.RoomId)
		toRoom := rooms.LoadRoom(anchorRoomId)
		if toRoom != nil {
			mob.Character.EndAggro()
			mob.Character.CastingState = nil // clear any stale cast
			if fromRoom != nil {
				fromRoom.RemoveMob(mob.InstanceId)
			}
			toRoom.AddMob(mob.InstanceId) // sets mob.Character.RoomId internally
		}
	}
	// Don't transition state here — the next tick will see
	// ctx.RoomId == p.SanctuaryRoom and run the at-sanctuary path
	// (dump satchel → transition to resting).
	return Success
}

func npcAttemptForage(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
) {
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return
	}
	biome := room.GetBiome()
	if biome == nil {
		return
	}
	searchRank := mob.Character.GetSkillLevel(skills.Search)
	searchScore := float64(mob.Character.Stats.Perception.ValueAdj) +
		combat.SkillMultiplier(searchRank)*25.0

	result := forager.ForageCore(forager.ForageAttempt{
		Biome:       biome.BiomeId,
		SearchScore: searchScore,
		AtNight:     gametime.IsNight(),
	})
	if !result.Found {
		return
	}
	item := items.New(result.ItemId)
	if !item.IsValid() {
		return
	}
	mob.Character.StoreItem(item)
	room.SendText(fmt.Sprintf(
		`<ansi fg="mobname">%s</ansi> stoops over a patch of growth`+
			` and tucks something into a satchel.`,
		p.Name))
}

// dumpSatchelToLockbox transfers every item in the forager's satchel
// into the room's "lockbox" container, then bumps the lockbox lock's
// RotationSeed so existing player keyring entries become invalid.
// If the room has no lockbox container or the lockbox is full
// (>= ForagerLockboxCapacity), items remain in the satchel and the
// caller falls back to the legacy rest-extension behavior.
//
// Returns true if any items were dumped.
func dumpSatchelToLockbox(mob *mobs.Mob, ctx *EvalContext) bool {
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return false
	}
	box, ok := room.Containers["lockbox"]
	if !ok {
		return false
	}
	cap := configs.GetBalanceConfig().ForagerLockboxCapacity
	if cap <= 0 {
		cap = 500
	}
	dumped := false
	stalledFull := false
	// Walk satchel in reverse so index shifts from RemoveItem stay valid.
	for i := len(mob.Character.Items) - 1; i >= 0; i-- {
		if len(box.Items) >= int(cap) {
			stalledFull = true
			break
		}
		item := mob.Character.Items[i]
		mob.Character.RemoveItem(item)
		box.AddItem(item)
		dumped = true
	}
	if stalledFull {
		// Surface so ops can diagnose foragers stuck in rest-extension.
		// The carry-ratio backstop in tickForagerResting will park the
		// forager until a player picks the lockbox open.
		mudlog.Debug("forager.dumpSatchelToLockbox: lockbox full",
			"roomId", ctx.RoomId, "capacity", cap, "satchelRemaining", len(mob.Character.Items))
	}
	if dumped {
		box.Lock.SetLocked() // bumps RotationSeed
		room.Containers["lockbox"] = box
		room.SendText(`<ansi fg="yellow">A latch clicks shut from somewhere in the sanctuary.</ansi>`)
	}
	return dumped
}

// npcVisitVendorsInRoom (Stage 3.4): physically transfers items from the
// forager's satchel (mob.Character.Items) into matching vendor stock entries
// at the given room. Items whose bucket isn't in p.Buckets are skipped.
// Items that don't fit (vendor at MaxStock, or no matching stock entry)
// stay in the satchel for the next vendor or next delivery cycle.
//
// Replaces the abstract RestockBuckets call from Stage 3.1.
func npcVisitVendorsInRoom(
	roomId int,
	p *forager.ForagerProfile,
	mob *mobs.Mob,
) {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		vendor := mobs.GetInstance(instId)
		if vendor == nil || !vendor.HasShop() {
			continue
		}
		shop := shops.GetShopInventory(vendor.Zone, int(vendor.MobId), roomId)
		if shop == nil {
			continue
		}
		// Track whether we mutated this vendor's stock so we only
		// persist when something actually transferred.
		mutated := false
		// Walk forager satchel in reverse so RemoveItem is index-safe.
		for i := len(mob.Character.Items) - 1; i >= 0; i-- {
			item := mob.Character.Items[i]
			bucket := economy.BucketFor(item.ItemId)
			if bucket == "" || !slices.Contains(p.Buckets, bucket) {
				continue
			}
			entry := shop.GetStock(item.ItemId)
			if entry == nil || entry.Current >= entry.MaxStock {
				continue
			}
			mob.Character.RemoveItem(item)
			entry.Current++
			mutated = true
			// Increment throughput counters for delivery tracking.
			spec := items.GetItemSpec(item.ItemId)
			if spec != nil {
				if spec.RarityTier > 0 {
					forager.IncrementDelivery(mob.Zone, int(mob.MobId), spec.RarityTier)
				}
				forager.AddLbsDelivered(mob.Zone, int(mob.MobId), uint64(math.Round(spec.Weight)))
			}
			room.SendText(fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> hands a %s to`+
					` <ansi fg="mobname">%s</ansi>.`,
				p.Name, item.DisplayName(), vendor.Character.Name,
			))
		}
		// Persist when stock actually changed. Mirrors the caravan-side
		// crash-safety pattern in internal/caravan/visit.go — without
		// this, forager restocks only hit disk on graceful shutdown
		// and a panic loses an in-flight cycle's deliveries.
		if mutated {
			if err := shops.SaveShop(vendor.Zone, int(vendor.MobId), roomId); err != nil {
				mudlog.Error("forager.npcVisitVendorsInRoom", "forager", p.Name, "vendor", vendor.Character.Name, "error", err)
			}
			if err := forager.SaveThroughput(mob.Zone, int(mob.MobId)); err != nil {
				mudlog.Error("forager.SaveThroughput", "forager", p.Name, "error", err)
			}
		}
	}
}

func npcWanderTerritoryNeighbor(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
) {
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return
	}
	var candidates []string
	for dir, exit := range room.Exits {
		if containsInt(p.TerritoryRooms, exit.RoomId) {
			candidates = append(candidates, dir)
		}
	}
	if len(candidates) == 0 {
		return
	}
	mob.Command(candidates[util.Rand(len(candidates))])
}

// ── Shared helpers ───────────────────────────────────────────────────

// getIntFromState reads a string key from BehaviorState and parses it
// as an int. Returns 0 on missing or unparseable value.
func getIntFromState(s *BehaviorState, key string) int {
	n, _ := strconv.Atoi(s.GetString(key))
	return n
}

// containsInt reports whether needle appears in haystack.
func containsInt(haystack []int, needle int) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// carryRatio returns carried-weight / carry-capacity. Uses the
// Character's own GetCarriedWeight + CarryCapacity methods so
// weight-reduction bags are handled correctly.
func carryRatio(mob *mobs.Mob) float64 {
	cap := mob.Character.CarryCapacity()
	if cap <= 0 {
		return 0
	}
	return mob.Character.GetCarriedWeight() / cap
}

// getMiscDataIntForager retrieves an integer from a mob's MiscData map,
// handling both int and float64 types (float64 can appear after YAML
// round-trips). Returns 0 if the key is absent or the type is
// unrecognized. Mirrors hooks.getMiscDataInt without importing that
// package (which would create an import cycle).
func getMiscDataIntForager(mob *mobs.Mob, key string) int {
	val := mob.Character.GetMiscData(key)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}
