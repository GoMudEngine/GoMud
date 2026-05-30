// Package justice computes whether a player is "wanted" by a faction and how
// severely, from the existing crime/rep/bounty substrate. The decision
// (Verdict) is pure and reusable (5.2 bounty hunters); the enforcement action
// lives in enforce.go.
package justice

import (
	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Severity is ordered: a future Arrest rung (5.1c) slots between Warn and
// Attack without reshaping callers. Compare with >.
type Severity int

const (
	SeverityNone   Severity = iota // citizen / not wanted
	SeverityWarn                   // wanted, mild — verbal warning
	SeverityArrest                 // RESERVED for 5.1c; not produced here
	SeverityAttack                 // wanted, severe — engage
)

// Read-seams — production wiring below; tests override.
var (
	repTierFn = func(faction string, userId int) opinions.Tier {
		return factions.TierFor(faction, userId)
	}
	alliesFn = func(faction string) []string {
		d := factions.GetDefinition(faction)
		if d == nil {
			return nil
		}
		return d.Allies
	}
	openFactionBountyFn = func(userId int, factionSet map[string]bool) bool {
		for _, b := range bounties.OpenAgainstPlayer(userId) {
			if b.Issuer.Type == bounties.IssuerFaction && factionSet[b.Issuer.Id] {
				return true
			}
		}
		return false
	}
	unresolvedCrimeFn = func(faction string, userId int, lookback, now uint64) bool {
		for _, c := range crimes.AllForFaction(faction, false) {
			if c.Perpetrator.Type != crimes.PerpPlayer || c.Perpetrator.Id != userId {
				continue
			}
			// now >= c.Round guards against uint64 underflow on the subtraction.
			if now >= c.Round && now-c.Round <= lookback {
				return true
			}
		}
		return false
	}
	nowRoundFn = util.GetRoundCount
	lookbackFn = func() uint64 {
		v := configs.GetBalanceConfig().JusticeCrimeLookbackRounds
		if v < 1 {
			return 1000
		}
		return uint64(v)
	}
)

// Verdict returns how a guard belonging to guardFactions should treat a
// player, taking the most severe of bounty / rep-tier / unresolved-crime
// signals across the guard factions and their declared allies.
func Verdict(guardFactions []string, userId int) Severity {
	set := map[string]bool{}
	for _, f := range guardFactions {
		set[f] = true
		for _, a := range alliesFn(f) {
			set[a] = true
		}
	}
	if openFactionBountyFn(userId, set) {
		return SeverityArrest
	}
	now := nowRoundFn()
	lookback := lookbackFn()
	worst := SeverityNone
	for f := range set {
		switch repTierFn(f, userId) {
		case opinions.TierHostile:
			return SeverityArrest
		case opinions.TierCold:
			if worst < SeverityWarn {
				worst = SeverityWarn
			}
		}
		if unresolvedCrimeFn(f, userId, lookback, now) {
			return SeverityArrest
		}
	}
	return worst
}
