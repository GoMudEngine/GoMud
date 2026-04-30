package behaviortree

// actions_forager.go — Stage 3.1 forager state-machine btree action.
//
// forager_step is the workhorse action that drives a forager NPC's
// daily cycle (forage → deliver → recall → rest → repeat). Mirrors
// the caravan_step pattern from actions_caravan.go.
//
// Import-cycle note: behaviortree cannot import usercommands (usercommands
// imports behaviortree). The forage tables and roll logic from
// usercommands/skill.forage.go are therefore duplicated here under the
// "npcForage*" names. Any change to the player forage tables must also
// be reflected here.

import (
	"fmt"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
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
	if cur != forager.StateRecalling &&
		hpRatio(mob) <= float64(cfg.ForagerHPRecallThresholdPct) {
		transitionForager(ctx.MobState, forager.StateRecalling)
		mob.Command("cast fold-recall")
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

	return Success
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
	mob.Command("cast fold-recall")
	return Success
}

// ── NPC forage helpers ───────────────────────────────────────────────
//
// These duplicate the tables from usercommands/skill.forage.go because
// behaviortree cannot import usercommands (import cycle). Any update to
// the player forage tables must be mirrored here.

var npcForageDifficulty = map[string]float64{
	"farmland":  110,
	"forest":    120,
	"land":      125,
	"swamp":     130,
	"shore":     135,
	"water":     135,
	"cave":      135,
	"mountains": 140,
	"cliffs":    145,
}

var npcForageYields = map[string][]int{
	"forest":    {40004, 40004, 40005, 40005, 40049, 40049},
	"land":      {40004, 40005, 40049, 40047},
	"farmland":  {40004, 40004, 40005, 40007},
	"swamp":     {40005, 40005, 40004, 40055, 40055, 40056, 40057, 40057},
	"shore":     {40004, 40058},
	"water":     {40058, 40058, 40058, 40058, 40058, 40059},
	"mountains": {40001, 40004, 40005, 40020, 40024, 40025},
	"cliffs":    {40005, 40020, 40024},
	"cave": {
		40001, 40001, 40020, 40020, 40005, 40024,
		40025, 40026, 40027, 40029,
	},
}

var npcNightForageYields = map[string][]int{
	"forest":    {40046},
	"mountains": {40046},
	"cave":      {40046},
	"land":      {40046},
}

type npcForageResult struct {
	found  bool
	itemId int
}

func npcForageCore(biomeId string, searchScore float64) npcForageResult {
	yields, ok := npcForageYields[biomeId]
	if !ok || len(yields) == 0 {
		return npcForageResult{}
	}
	if gametime.IsNight() {
		if night, hasNight := npcNightForageYields[biomeId]; hasNight {
			yields = append(append([]int{}, yields...), night...)
		}
	}
	difficulty := npcForageDifficulty[biomeId]
	if difficulty == 0 {
		difficulty = 130
	}
	roll := dice.RollStat(searchScore)
	if roll.Value < difficulty {
		return npcForageResult{}
	}
	return npcForageResult{
		found:  true,
		itemId: yields[util.Rand(len(yields))],
	}
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

	result := npcForageCore(biome.BiomeId, searchScore)
	if !result.found {
		return
	}
	item := items.New(result.itemId)
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
