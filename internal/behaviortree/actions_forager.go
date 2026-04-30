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
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
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
		return tickForagerDelivering(profile, mob, ctx, cfg)
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

	// Salvage any corpse in current room.
	mob.Command("salvage corpse")

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
	cfg configs.Balance,
) Result {
	if p.Kind == forager.KindFernway {
		return tickForagerDeliveringFernway(p, mob, ctx, cfg)
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
	npcVisitVendorsInRoom(target, p)
	ctx.MobState.Set(keyVisitIndex, strconv.Itoa(idx+1))
	return Success
}

func tickForagerDeliveringFernway(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
	cfg configs.Balance,
) Result {
	if ctx.RoomId != p.MeetingRoom {
		mob.Command(fmt.Sprintf("pathto %d", p.MeetingRoom))
		return Success
	}
	waitT := getIntFromState(ctx.MobState, keyWaitTimer) + 1
	ctx.MobState.Set(keyWaitTimer, strconv.Itoa(waitT))
	if waitT >= int(cfg.ForagerWaitTimeoutRounds) {
		transitionForager(ctx.MobState, forager.StateRecalling)
		return Success
	}
	return Success
}

func tickForagerRecalling(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
) Result {
	if ctx.RoomId == p.SanctuaryRoom {
		transitionForager(ctx.MobState, forager.StateResting)
		return Success
	}
	// Don't re-issue the cast every idle tick — re-issuing while a cast
	// is in progress can reset its progress and trap the forager mid-cast
	// indefinitely (observed 2026-04-30: Kessa "begins weaving a spell"
	// repeatedly but never actually teleports). Wait for the active cast
	// to resolve.
	if mob.Character.IsCasting() {
		return Success
	}
	mob.Command("cast fold-recall")
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

func npcVisitVendorsInRoom(roomId int, p *forager.ForagerProfile) {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		m := mobs.GetInstance(instId)
		if m == nil || !m.HasShop() {
			continue
		}
		si := shops.GetShopInventory(m.Zone, int(m.MobId), roomId)
		if si == nil {
			continue
		}
		if si.RestockBuckets(p.Buckets) {
			room.SendText(fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> lays a satchel of`+
					` mats on <ansi fg="mobname">%s</ansi>'s counter.`,
				p.Name, m.Character.Name))
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
