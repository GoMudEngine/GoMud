// Package catalog — befriend-faction goal type.
//
// # MiscData rep substrate
//
// Mob-to-faction relationship progress is tracked via a per-mob MiscData
// counter keyed as "faction_rep_built_with:<factionId>" (e.g.
// "faction_rep_built_with:merchants"). This mirrors the revenge-faction
// "faction_kills_inflicted:<factionId>" pattern exactly.
//
// The counter is READ here (4.3). The WRITER — an event hook that bumps
// the counter when the mob has positive interactions with faction members
// (greetings, trades, assistance, conversation exchanges) — ships in
// chunk 4.5 (reactive hooks). Until 4.5 lands, factionRepOf always returns
// 0 and the predicate never satisfies; the goal can still be seeded and
// scored, it simply never completes.
//
// Authors of the 4.5 reactive hooks: increment
// "faction_rep_built_with:<factionId>" on the mob's MiscData whenever the
// mob has a qualifying positive interaction with a member of that faction.
// The counter is an unbounded int; rep_threshold (default 60) is the
// satisfaction level.
package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// factionRepMiscPrefix is the MiscData key prefix for per-faction rep
// built by a mob. Full key: factionRepMiscPrefix + factionId.
// Mirrors factionKillsMiscPrefix from revenge_faction.go.
const factionRepMiscPrefix = "faction_rep_built_with:"

// befriendFactionDefaultThreshold is the rep score at which the
// befriend-faction goal is considered satisfied (no rep_threshold param
// supplied).
const befriendFactionDefaultThreshold = 60

func init() {
	goals.RegisterGoalType("befriend-faction", goals.GoalTypeMeta{
		Predicate:     befriendFactionPredicate,
		ContextScore:  befriendFactionContextScore,
		AllowMultiple: true,
		DedupKey:      befriendFactionDedupKey,
		Params: []goals.ParamSchema{
			{Key: "faction_id", Required: true, GoType: "string"},
			{Key: "rep_threshold", Required: false, GoType: "int"},
		},
	})
}

func befriendFactionDedupKey(g *goals.Goal) string {
	if fid, ok := g.Params["faction_id"].(string); ok {
		return fid
	}
	return ""
}

// befriendFactionPredicate: satisfied when the mob's rep with the faction
// (stored in MiscData under "faction_rep_built_with:<faction_id>") reaches
// rep_threshold (default 60).
//
// The counter is incremented by a 4.5 reactive hook on positive-interaction
// events; 4.3 only defines the read path. Until 4.5 ships the predicate
// always returns false for freshly spawned mobs.
func befriendFactionPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	fid, _ := g.Params["faction_id"].(string)
	if fid == "" {
		return false
	}
	thr := paramIntOr(g, "rep_threshold", befriendFactionDefaultThreshold)
	return factionRepOf(mob, fid) >= thr
}

// befriendFactionContextScore: 0 if no faction members are in the mob's
// zone (no opportunity to build rep); otherwise baseline 1.0 with a small
// positive bump proportional to the remaining rep gap, capped at 1.8.
//
// score = 1.0 + 0.1 * (threshold - current) / threshold
//
// This intentionally biases goal selection toward factions where meaningful
// progress still needs to be made while keeping the ceiling below 2.0 (the
// revenge-faction ceiling) to reflect lower urgency than revenge.
func befriendFactionContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	fid, _ := g.Params["faction_id"].(string)
	if fid == "" {
		return 0
	}
	if factionMembersInZone(mob, fid) == 0 {
		return 0
	}
	thr := paramIntOr(g, "rep_threshold", befriendFactionDefaultThreshold)
	current := factionRepOf(mob, fid)
	if thr <= 0 || current >= thr {
		return 1.0
	}
	gap := float64(thr-current) / float64(thr)
	score := 1.0 + 0.1*gap
	if score > 1.8 {
		return 1.8
	}
	return score
}

// factionRepOf reads the mob's accumulated rep with the given faction from
// MiscData. Returns 0 if the counter has not been written yet (i.e. before
// the chunk-4.5 reactive hooks ship). Uses mobMiscInt from revenge_faction.go
// (same package — no duplication needed).
func factionRepOf(mob *mobs.Mob, factionId string) int {
	return mobMiscInt(mob, factionRepMiscPrefix+factionId)
}
