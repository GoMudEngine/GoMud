package combat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/natefinch/lumberjack"
)

// CombatEvent captures a single combat action for analytics.
type CombatEvent struct {
	SourceType                SourceTarget `json:"source_type"`
	TargetType                SourceTarget `json:"target_type"`
	AttackType                string       `json:"attack_type"`
	Hit                       bool         `json:"hit"`
	Crit                      bool         `json:"crit"`
	Fumble                    bool         `json:"fumble"`
	Backfire                  bool         `json:"backfire"`
	Fizzle                    bool         `json:"fizzle"`
	DamageDealt               int          `json:"damage_dealt"`
	DamageReduced             int          `json:"damage_reduced"`
	DefenseUsed               string       `json:"defense_used"`
	AttackZScore              float64      `json:"attack_z_score"`
	DefenseZScore             float64      `json:"defense_z_score"`
	SourcePosition            string       `json:"source_position"`
	TargetPosition            string       `json:"target_position"`
	SourceIsGrappleController bool         `json:"source_is_grapple_controller"`
	TargetIsGrappleController bool         `json:"target_is_grapple_controller"`
	RoundNumber               uint64       `json:"round_number"`
}

// AnalyticsSummary is the aggregated data written as one JSON line per flush.
type AnalyticsSummary struct {
	// Totals
	TotalEvents int `json:"total_events"`
	Hits        int `json:"hits"`
	Misses      int `json:"misses"`
	Crits       int `json:"crits"`
	Fumbles     int `json:"fumbles"`
	Backfires   int `json:"backfires"`
	Fizzles     int `json:"fizzles"`
	TotalDamage int `json:"total_damage"`

	// By attack type
	ByAttackType map[string]*AttackTypeStats `json:"by_attack_type"`

	// Defense breakdown
	DodgeSuccesses int `json:"dodge_successes"`
	ParrySuccesses int `json:"parry_successes"`
	BlockSuccesses int `json:"block_successes"`

	// Matchup breakdown
	PvMEvents int `json:"pvm_events"`
	MvPEvents int `json:"mvp_events"`
	PvPEvents int `json:"pvp_events"`
	MvMEvents int `json:"mvm_events"`

	// Position stats
	HitRateTargetStanding float64 `json:"hit_rate_target_standing"`
	HitRateTargetProne    float64 `json:"hit_rate_target_prone"`
	HitRateTargetClinched float64 `json:"hit_rate_target_clinched"`
	HitRateTargetGrounded float64 `json:"hit_rate_target_grounded"`

	HitRateGrappleController    float64 `json:"hit_rate_grapple_controller"`
	HitRateNonGrappleController float64 `json:"hit_rate_non_grapple_controller"`

	// Z-score averages
	AvgAttackZScore  float64 `json:"avg_attack_z_score"`
	AvgDefenseZScore float64 `json:"avg_defense_z_score"`

	// Round range
	EarliestRound uint64 `json:"earliest_round"`
	LatestRound   uint64 `json:"latest_round"`
}

// AttackTypeStats holds per-attack-type counters.
type AttackTypeStats struct {
	Events      int `json:"events"`
	Hits        int `json:"hits"`
	Crits       int `json:"crits"`
	TotalDamage int `json:"total_damage"`
}

var (
	analyticsReady bool
	analyticsOnce  sync.Once
	eventBuffer    []CombatEvent
	maxEvents      int
	logWriter      *lumberjack.Logger
)

// InitAnalytics initializes the analytics subsystem (call once at startup).
func InitAnalytics() {
	analyticsOnce.Do(func() {
		cfg := configs.GetAnalyticsConfig()
		if !bool(cfg.Enabled) {
			return
		}

		maxEvents = int(cfg.MaxEvents)
		if maxEvents < 100 {
			maxEvents = 100
		}
		eventBuffer = make([]CombatEvent, 0, 1024)

		logPath := string(cfg.LogPath)
		dir := filepath.Dir(logPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			mudlog.Error("InitAnalytics", "error", "failed to create log directory: "+err.Error())
			return
		}

		logWriter = &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    50,
			MaxBackups: 10,
			Compress:   true,
		}

		analyticsReady = true
		mudlog.Info("InitAnalytics", "state", "Combat analytics enabled",
			"maxEvents", maxEvents, "logPath", logPath)
	})
}

// appendEvent adds an event to the ring buffer.
// Must be called under util.LockMud().
func appendEvent(evt CombatEvent) {
	if !analyticsReady {
		return
	}

	if len(eventBuffer) >= maxEvents {
		// Drop oldest event (FIFO)
		copy(eventBuffer, eventBuffer[1:])
		eventBuffer = eventBuffer[:len(eventBuffer)-1]
	}
	eventBuffer = append(eventBuffer, evt)
}

// positionFields populates position and grapple controller fields from a character.
func positionFields(char *characters.Character) (string, bool) {
	if char == nil {
		return "standing", false
	}
	pos := string(char.CombatPosition)
	if pos == "" {
		pos = "standing"
	}
	isController := char.HasCondition(characters.ConditionGrappleController)
	return pos, isController
}

// RecordAttack records a standard auto-attack event.
func RecordAttack(result AttackResult, src, tgt SourceTarget, atkType string,
	srcChar, tgtChar *characters.Character, round uint64) {
	if !analyticsReady {
		return
	}

	srcPos, srcCtrl := positionFields(srcChar)
	tgtPos, tgtCtrl := positionFields(tgtChar)

	evt := CombatEvent{
		SourceType:                src,
		TargetType:                tgt,
		AttackType:                atkType,
		Hit:                       result.Hit,
		Crit:                      result.Crit,
		Fumble:                    result.Fumble,
		DamageDealt:               result.DamageToTarget,
		DamageReduced:             result.DamageToTargetReduction,
		DefenseUsed:               string(result.DefenseUsed),
		AttackZScore:              result.AttackZScore,
		DefenseZScore:             result.DefenseZScore,
		SourcePosition:            srcPos,
		TargetPosition:            tgtPos,
		SourceIsGrappleController: srcCtrl,
		TargetIsGrappleController: tgtCtrl,
		RoundNumber:               round,
	}
	appendEvent(evt)
}

// RecordSpecialMove records a special combat move (bash, kick, trip, submit, grapple, mutations).
func RecordSpecialMove(src, tgt SourceTarget, atkType string, hit bool,
	dmg int, srcChar, tgtChar *characters.Character, round uint64) {
	if !analyticsReady {
		return
	}

	srcPos, srcCtrl := positionFields(srcChar)
	tgtPos, tgtCtrl := positionFields(tgtChar)

	evt := CombatEvent{
		SourceType:                src,
		TargetType:                tgt,
		AttackType:                atkType,
		Hit:                       hit,
		DamageDealt:               dmg,
		SourcePosition:            srcPos,
		TargetPosition:            tgtPos,
		SourceIsGrappleController: srcCtrl,
		TargetIsGrappleController: tgtCtrl,
		RoundNumber:               round,
	}
	appendEvent(evt)
}

// RecordSpell records a spell resolution event.
func RecordSpell(src, tgt SourceTarget, hit, crit, backfire, fizzle bool,
	dmg int, zScore float64, srcChar, tgtChar *characters.Character, round uint64) {
	if !analyticsReady {
		return
	}

	srcPos, srcCtrl := positionFields(srcChar)
	tgtPos, tgtCtrl := positionFields(tgtChar)

	evt := CombatEvent{
		SourceType:                src,
		TargetType:                tgt,
		AttackType:                "spell",
		Hit:                       hit,
		Crit:                      crit,
		Backfire:                  backfire,
		Fizzle:                    fizzle,
		DamageDealt:               dmg,
		AttackZScore:              zScore,
		SourcePosition:            srcPos,
		TargetPosition:            tgtPos,
		SourceIsGrappleController: srcCtrl,
		TargetIsGrappleController: tgtCtrl,
		RoundNumber:               round,
	}
	appendEvent(evt)
}

// FlushAnalytics computes a summary from the current buffer and writes it as JSON.
// Must be called under util.LockMud().
func FlushAnalytics() {
	if !analyticsReady || len(eventBuffer) == 0 {
		return
	}

	summary := computeSummary(eventBuffer)

	data, err := json.Marshal(summary)
	if err != nil {
		mudlog.Error("FlushAnalytics", "error", err.Error())
		return
	}
	data = append(data, '\n')

	if _, err := logWriter.Write(data); err != nil {
		mudlog.Error("FlushAnalytics", "error", "write failed: "+err.Error())
	}

	mudlog.Info("FlushAnalytics", "events", summary.TotalEvents,
		"hits", summary.Hits, "crits", summary.Crits)
}

// computeSummary aggregates the event buffer into an AnalyticsSummary.
func computeSummary(events []CombatEvent) AnalyticsSummary {
	s := AnalyticsSummary{
		ByAttackType: make(map[string]*AttackTypeStats),
	}

	// Position hit tracking
	type posHit struct{ hits, total int }
	posMap := map[string]*posHit{
		"standing": {}, "prone": {}, "clinched": {}, "grounded": {},
	}
	var grappleCtrlHits, grappleCtrlTotal int
	var nonCtrlHits, nonCtrlTotal int

	var sumAtkZ, sumDefZ float64

	for _, e := range events {
		s.TotalEvents++

		if e.Hit {
			s.Hits++
		} else {
			s.Misses++
		}
		if e.Crit {
			s.Crits++
		}
		if e.Fumble {
			s.Fumbles++
		}
		if e.Backfire {
			s.Backfires++
		}
		if e.Fizzle {
			s.Fizzles++
		}
		s.TotalDamage += e.DamageDealt

		// By attack type
		at, ok := s.ByAttackType[e.AttackType]
		if !ok {
			at = &AttackTypeStats{}
			s.ByAttackType[e.AttackType] = at
		}
		at.Events++
		if e.Hit {
			at.Hits++
		}
		if e.Crit {
			at.Crits++
		}
		at.TotalDamage += e.DamageDealt

		// Defense breakdown
		switch e.DefenseUsed {
		case "dodge":
			s.DodgeSuccesses++
		case "parry":
			s.ParrySuccesses++
		case "block":
			s.BlockSuccesses++
		}

		// Matchups
		switch {
		case e.SourceType == User && e.TargetType == Mob:
			s.PvMEvents++
		case e.SourceType == Mob && e.TargetType == User:
			s.MvPEvents++
		case e.SourceType == User && e.TargetType == User:
			s.PvPEvents++
		case e.SourceType == Mob && e.TargetType == Mob:
			s.MvMEvents++
		}

		// Position hit rates
		if p, ok := posMap[e.TargetPosition]; ok {
			p.total++
			if e.Hit {
				p.hits++
			}
		}

		// Grapple controller hit rates
		if e.SourceIsGrappleController {
			grappleCtrlTotal++
			if e.Hit {
				grappleCtrlHits++
			}
		} else {
			nonCtrlTotal++
			if e.Hit {
				nonCtrlHits++
			}
		}

		sumAtkZ += e.AttackZScore
		sumDefZ += e.DefenseZScore

		// Round range
		if s.EarliestRound == 0 || e.RoundNumber < s.EarliestRound {
			s.EarliestRound = e.RoundNumber
		}
		if e.RoundNumber > s.LatestRound {
			s.LatestRound = e.RoundNumber
		}
	}

	// Compute rates
	if p := posMap["standing"]; p.total > 0 {
		s.HitRateTargetStanding = float64(p.hits) / float64(p.total)
	}
	if p := posMap["prone"]; p.total > 0 {
		s.HitRateTargetProne = float64(p.hits) / float64(p.total)
	}
	if p := posMap["clinched"]; p.total > 0 {
		s.HitRateTargetClinched = float64(p.hits) / float64(p.total)
	}
	if p := posMap["grounded"]; p.total > 0 {
		s.HitRateTargetGrounded = float64(p.hits) / float64(p.total)
	}

	if grappleCtrlTotal > 0 {
		s.HitRateGrappleController = float64(grappleCtrlHits) / float64(grappleCtrlTotal)
	}
	if nonCtrlTotal > 0 {
		s.HitRateNonGrappleController = float64(nonCtrlHits) / float64(nonCtrlTotal)
	}

	if s.TotalEvents > 0 {
		s.AvgAttackZScore = sumAtkZ / float64(s.TotalEvents)
		s.AvgDefenseZScore = sumDefZ / float64(s.TotalEvents)
	}

	return s
}
