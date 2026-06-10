package hooks

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/combat"
)

// File: combat_verbosity.go
//
// Light-verbosity round tally for combat narration (spec:
// docs/superpowers/specs/2026-06-10-combat-verbosity-design.md).
// When a viewer's effective verbosity is Light, the per-swing combat
// lines are suppressed at the drain (dispatchCritAndMessaging) and the
// AttackResult's swing data is recorded here instead. flushCombatTallies
// (Task 5) emits one compact line per fight pair per viewer at the end
// of DoCombat. All state is touched only from the game-loop goroutine.

// fighterRef identifies one combatant for tally purposes. Key is a
// stable identity ("u:<userId>" / "m:<mobInstanceId>") so same-named
// mobs don't merge; Name/IsMob drive rendering.
type fighterRef struct {
	Key   string
	Name  string
	IsMob bool
}

// swingStat is the slice of SwingEvent the tally needs.
type swingStat struct {
	Hit    bool
	Damage int
}

// tallyDir accumulates one attack direction within a fight pair.
type tallyDir struct {
	Hits        int
	Misses      int
	WorstHit    int
	TargetMaxHP int
}

func (d *tallyDir) add(swings []swingStat, targetMaxHP int) {
	for _, s := range swings {
		if s.Hit {
			d.Hits++
			if s.Damage > d.WorstHit {
				d.WorstHit = s.Damage
			}
		} else {
			d.Misses++
		}
	}
	if targetMaxHP > 0 {
		d.TargetMaxHP = targetMaxHP
	}
}

// combatTally is one (viewer, fight-pair) accumulator. A/B orientation
// is fixed by whichever direction is recorded first.
type combatTally struct {
	A, B fighterRef
	AtoB tallyDir
	BtoA tallyDir
}

type tallyKey struct {
	viewerId int
	pairKey  string // canonical unordered pair: min(key)+"|"+max(key)
}

type combatTallies struct {
	m map[tallyKey]*combatTally
}

func newCombatTallies() *combatTallies {
	return &combatTallies{m: map[tallyKey]*combatTally{}}
}

func pairKeyFor(a, b string) string {
	if a < b {
		return a + "|" + b
	}
	return b + "|" + a
}

// record adds one AttackResult's swings (attacker → defender) to the
// viewer's tally for that fight pair.
func (ct *combatTallies) record(viewerId int, attacker, defender fighterRef, swings []swingStat, defenderMaxHP int) {
	k := tallyKey{viewerId: viewerId, pairKey: pairKeyFor(attacker.Key, defender.Key)}
	t, ok := ct.m[k]
	if !ok {
		t = &combatTally{A: attacker, B: defender}
		ct.m[k] = t
	}
	if attacker.Key == t.A.Key {
		t.AtoB.add(swings, defenderMaxHP)
	} else {
		t.BtoA.add(swings, defenderMaxHP)
	}
}

// countWord renders a hit count as prose. 1 → "" (the verb carries it),
// per the no-hard-numbers rule everything stays qualitative.
func countWord(n int) string {
	switch {
	case n <= 1:
		return ""
	case n == 2:
		return " twice"
	case n == 3:
		return " three times"
	default:
		return " again and again"
	}
}

// nameToken renders a fighter's name with the engine's standard color
// alias for their kind.
func nameToken(f fighterRef) string {
	if f.IsMob {
		return `<ansi fg="mobname">` + f.Name + `</ansi>`
	}
	return `<ansi fg="username">` + f.Name + `</ansi>`
}

// pronounFails is the subject stand-in for a fighter on second mention,
// with its agreeing verb form for "fail".
func pronounFails(f fighterRef) string {
	if f.IsMob {
		return "it fails"
	}
	return "they fail"
}

// renderTally builds the tally line for one fight pair from a viewer's
// perspective. viewerKey is "" for spectators, or the viewer's
// fighterRef.Key when they are a participant (their side renders as
// "You" and their incoming LANDED hits are omitted — full prose already
// showed them under the floor rule).
func renderTally(t *combatTally, viewerKey string) string {
	// Orient so X = viewer (participant) or t.A (spectator).
	x, y := t.A, t.B
	xOut, yOut := t.AtoB, t.BtoA
	if viewerKey != "" && t.B.Key == viewerKey {
		x, y = t.B, t.A
		xOut, yOut = t.BtoA, t.AtoB
	}
	isParticipant := viewerKey != "" && x.Key == viewerKey

	xSwings := xOut.Hits + xOut.Misses
	ySwings := yOut.Hits + yOut.Misses

	// Whiff round: swings happened, nothing landed either way.
	if xOut.Hits == 0 && yOut.Hits == 0 && (xSwings > 0 || ySwings > 0) {
		if isParticipant {
			return fmt.Sprintf("You trade swings with %s; neither side draws blood.", nameToken(y))
		}
		return fmt.Sprintf("%s and %s trade swings without drawing blood.", nameToken(x), nameToken(y))
	}

	segs := []string{}

	// X's outgoing segment.
	if xOut.Hits > 0 {
		tier := combat.GetDamageDescription(xOut.WorstHit, xOut.TargetMaxHP)
		if isParticipant {
			segs = append(segs, fmt.Sprintf("You strike %s%s (%s)", nameToken(y), countWord(xOut.Hits), tier))
		} else {
			segs = append(segs, fmt.Sprintf("%s strikes %s%s (%s)", nameToken(x), nameToken(y), countWord(xOut.Hits), tier))
		}
	} else if xSwings > 0 {
		if isParticipant {
			segs = append(segs, fmt.Sprintf("You fail to break %s's guard", nameToken(y)))
		} else {
			segs = append(segs, fmt.Sprintf("%s can't get past %s's guard", nameToken(x), nameToken(y)))
		}
	}

	// Y's segment. For participants, landed incoming hits already showed
	// in full prose (floor rule) — only whiffs are worth a mention.
	if yOut.Hits > 0 {
		if !isParticipant {
			tier := combat.GetDamageDescription(yOut.WorstHit, yOut.TargetMaxHP)
			segs = append(segs, fmt.Sprintf("%s lands %s%s (%s)",
				nameToken(y), hitNoun(yOut.Hits), countWord(yOut.Hits), tier))
		}
	} else if ySwings > 0 {
		if isParticipant {
			segs = append(segs, fmt.Sprintf("%s to land a blow", pronounFails(y)))
		} else {
			segs = append(segs, fmt.Sprintf("%s fails to land a blow", nameToken(y)))
		}
	}

	if len(segs) == 0 {
		return ""
	}
	return strings.Join(segs, "; ") + "."
}

// hitNoun: "a blow" vs "blows".
func hitNoun(n int) string {
	if n == 1 {
		return "a blow"
	}
	return "blows"
}

// flushForViewer renders and removes all of one viewer's tallies,
// sorted by pair key for deterministic output.
func (ct *combatTallies) flushForViewer(viewerId int, viewerKey string) []string {
	keys := []tallyKey{}
	for k := range ct.m {
		if k.viewerId == viewerId {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].pairKey < keys[j].pairKey })

	lines := []string{}
	for _, k := range keys {
		if line := renderTally(ct.m[k], viewerKey); line != "" {
			lines = append(lines, line)
		}
		delete(ct.m, k)
	}
	return lines
}

// viewerIds returns the distinct viewers with pending tallies.
func (ct *combatTallies) viewerIds() []int {
	seen := map[int]bool{}
	out := []int{}
	for k := range ct.m {
		if !seen[k.viewerId] {
			seen[k.viewerId] = true
			out = append(out, k.viewerId)
		}
	}
	sort.Ints(out)
	return out
}
