package goals

import (
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// PruneReason categorizes why a goal was removed by Prune.
type PruneReason string

const (
	ReasonSatisfied PruneReason = "satisfied"
	ReasonExpired   PruneReason = "expired"
	ReasonAbandoned PruneReason = "abandoned-stale"
)

// PruneRecord reports one removed goal (returned for logging/tests).
type PruneRecord struct {
	GoalId string
	Type   string
	Reason PruneReason
}

// Prune evaluates every goal on the mob template and removes those that
// are satisfied, expired, or abandoned-stale (context score ~0 for at
// least GoalAbandonDormantRounds). Per-goal DormantSinceRound stamps are
// updated in place. Persists once if anything changed and triggers one
// Recompute if any goal was removed. Safe with a nil mob.
func Prune(mobId int, namesimple string, mob *mobs.Mob, now time.Time, nowRound uint64) []PruneRecord {
	mg := loadOrLazyInit(mobId, namesimple)
	abandonRounds := abandonDormantRoundsConfig()

	cacheMu.Lock()
	var records []PruneRecord
	removeIds := map[string]bool{}
	changed := false

	for _, g := range mg.Goals {
		if IsSatisfied(g, mob) {
			records = append(records, PruneRecord{g.Id, g.Type, ReasonSatisfied})
			removeIds[g.Id] = true
			continue
		}
		if IsExpired(g, now) {
			records = append(records, PruneRecord{g.Id, g.Type, ReasonExpired})
			removeIds[g.Id] = true
			continue
		}
		if effectiveContextMod(g, mob) > 0 {
			if g.DormantSinceRound != 0 {
				g.DormantSinceRound = 0
				changed = true
			}
			continue
		}
		// Dormant (score == 0).
		if g.DormantSinceRound == 0 {
			g.DormantSinceRound = nowRound
			changed = true
		}
		if abandonRounds > 0 && nowRound >= g.DormantSinceRound &&
			nowRound-g.DormantSinceRound >= uint64(abandonRounds) {
			records = append(records, PruneRecord{g.Id, g.Type, ReasonAbandoned})
			removeIds[g.Id] = true
		}
	}

	if len(removeIds) > 0 {
		out := mg.Goals[:0:0]
		for _, g := range mg.Goals {
			if !removeIds[g.Id] {
				out = append(out, g)
			}
		}
		mg.Goals = out
		if removeIds[mg.CurrentGoalId] {
			mg.CurrentGoalId = ""
			mg.CurrentSinceRound = 0
			mg.LastSwitchRound = 0
		}
		changed = true
	}
	nameByMobId[mobId] = namesimple
	cacheMu.Unlock()

	if changed {
		if err := saveToDisk(mobId, namesimple); err != nil {
			mudlog.Warn("goals.Prune: save failed", "mob_id", mobId, "error", err)
		}
	}
	for _, r := range records {
		mudlog.Debug("goals.prune",
			"mob_id", mobId,
			"goal", r.GoalId,
			"type", r.Type,
			"reason", string(r.Reason),
			"round", nowRound)
	}
	if len(removeIds) > 0 {
		Recompute(mobId, namesimple, mob, nowRound)
	}
	return records
}

// abandonDormantRoundsConfig reads GoalAbandonDormantRounds, defaulting
// to 600 if unset/invalid (mirrors the config validate default).
func abandonDormantRoundsConfig() int {
	v := int(configs.GetBalanceConfig().GoalAbandonDormantRounds)
	if v < 1 {
		v = 600
	}
	return v
}
