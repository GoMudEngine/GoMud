package hooks

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
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
// perspective. viewerKey is the viewer's fighterRef.Key when they are a
// participant (their side renders as "You" and their incoming LANDED
// hits are omitted — full prose already showed them under the floor
// rule). A spectator's key simply matches neither fighter, so they
// render third-person; "" is the logged-off/cleanup path.
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

// ── Drain-side glue ────────────────────────────────────────────────────

// roundTallies is the per-round accumulator. Game-loop goroutine only.
var roundTallies = newCombatTallies()

// fighterRefFor builds a tally identity for an Actor. For mobs the
// display name is stamped with the room duplicate-index suffix (e.g.
// "Skeleton #2") so same-named mobs produce distinct tally labels —
// matching the disambiguation already applied to per-swing combat lines.
func fighterRefFor(a actions.Actor) fighterRef {
	if a.IsPlayer() {
		return fighterRef{Key: fmt.Sprintf("u:%d", a.GetUserId()), Name: a.GetCharacter().Name, IsMob: false}
	}
	name := a.GetCharacter().Name
	if room := a.GetRoom(); room != nil {
		if dupIdx := room.GetMobDuplicateIndex(a.GetMobInstanceId()); dupIdx > 0 {
			name = fmt.Sprintf("%s #%d", name, dupIdx)
		}
	}
	return fighterRef{Key: fmt.Sprintf("m:%d", a.GetMobInstanceId()), Name: name, IsMob: true}
}

// swingStatsFor extracts tally stats from an AttackResult. Rounds with
// no per-swing analytics (defensive fallback) degrade to one synthetic
// swing from the top-level Hit/DamageToTarget.
func swingStatsFor(res *combat.AttackResult) []swingStat {
	if len(res.SwingEvents) > 0 {
		out := make([]swingStat, 0, len(res.SwingEvents))
		for _, s := range res.SwingEvents {
			out = append(out, swingStat{Hit: s.Hit, Damage: s.Damage})
		}
		return out
	}
	if res.DefenderWasAttacked || res.Hit {
		return []swingStat{{Hit: res.Hit, Damage: res.DamageToTarget}}
	}
	return nil
}

// recordTallyFor records one AttackResult into a viewer's round tally.
func recordTallyFor(viewerId int, atk, def actions.Actor, res *combat.AttackResult) {
	swings := swingStatsFor(res)
	if len(swings) == 0 {
		return
	}
	roundTallies.record(viewerId, fighterRefFor(atk), fighterRefFor(def), swings,
		def.GetCharacter().HealthMax.Value)
}

// drainParticipantLines sends a participant's combat lines subject to
// their verbosity. incoming=true marks lines describing swings AGAINST
// the viewer: landed hits there are floor-protected (always full prose,
// any level); only defense/miss lines are suppressible.
func drainParticipantLines(u *users.UserRecord, msgs []combat.TaggedMessage, lvl messaging.Verbosity, incoming bool) {
	for _, msg := range msgs {
		if incoming && isHitCategory(msg.Category) {
			u.SendText(msg.Category, msg.Text) // floor: damage to you always shows
			continue
		}
		if lvl.Suppresses(msg.Category) {
			continue
		}
		u.SendText(msg.Category, msg.Text)
	}
}

// isHitCategory reports whether a category is one of the CategoryHit*
// damage bands.
func isHitCategory(cat messaging.Category) bool {
	switch cat {
	case messaging.CategoryHitMelee, messaging.CategoryHitBlunt, messaging.CategoryHitNaturalSharp,
		messaging.CategoryHitRanged, messaging.CategoryHitCaster, messaging.CategoryHitUnarmed:
		return true
	}
	return false
}

// drainSpectatorLines delivers room combat lines per spectator at their
// effective (one-step-lower) verbosity, preserving the sight gate via
// SendTextVisualToUser. Excluded ids are the combatants (they got their
// participant lines already).
func drainSpectatorLines(room *rooms.Room, msgs []combat.TaggedMessage, excludeUserIds []int) {
	if room == nil || len(msgs) == 0 {
		return
	}
	for _, uid := range room.GetPlayers() {
		if isExcludedUser(uid, excludeUserIds) {
			continue
		}
		u := users.GetByUserId(uid)
		if u == nil {
			continue
		}
		lvl := u.GetCombatVerbosity().OneStepLower()
		for _, msg := range msgs {
			if lvl.Suppresses(msg.Category) {
				continue
			}
			room.SendTextVisualToUser(u, msg.Category, msg.Text)
		}
	}
}

// recordSpectatorTallies records this AttackResult for every spectator
// whose effective verbosity is Light. Called once per AttackResult
// (NOT per message batch).
func recordSpectatorTallies(atkRoom, defRoom *rooms.Room, atk, def actions.Actor, res *combat.AttackResult, excludeUserIds []int) {
	seen := map[int]bool{}
	for _, room := range []*rooms.Room{atkRoom, defRoom} {
		if room == nil {
			continue
		}
		for _, uid := range room.GetPlayers() {
			if seen[uid] || isExcludedUser(uid, excludeUserIds) {
				continue
			}
			seen[uid] = true
			u := users.GetByUserId(uid)
			if u == nil {
				continue
			}
			// Sight gate: a spectator in darkness receives the generic
			// sounds-of-fighting fallback but must not receive a named
			// tally summary, which would leak combatant identities they
			// cannot see. Shapes-only (infrared) viewers are treated the
			// same as blind here — the named tally requires clear sight.
			if !messaging.CanSeeClearly(u.Character, room) {
				continue
			}
			if u.GetCombatVerbosity().OneStepLower() == messaging.VerbosityLight {
				recordTallyFor(uid, atk, def, res)
			}
		}
	}
}

// flushCombatTallies emits every pending tally line and clears the
// accumulator. Called once at the end of DoCombat each round.
func flushCombatTallies() {
	for _, viewerId := range roundTallies.viewerIds() {
		u := users.GetByUserId(viewerId)
		if u == nil {
			// Viewer logged off mid-round; drop their tallies.
			roundTallies.flushForViewer(viewerId, "")
			continue
		}
		viewerKey := fmt.Sprintf("u:%d", viewerId)
		for _, line := range roundTallies.flushForViewer(viewerId, viewerKey) {
			u.SendText(messaging.CategoryCombatSummary, line)
		}
	}
}
