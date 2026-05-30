package justice

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// EnforceAction reports one enforcement decision (returned for tests/telemetry).
type EnforceAction struct {
	UserId    int
	Severity  Severity
	Escalated bool // a prior warning escalated to attack this tick
}

type warnOutcome int

const (
	warnOutcomeNone warnOutcome = iota
	warnOutcomeWarn
	warnOutcomeAttack
)

type arrestOutcome int

const (
	arrestOutcomeNone    arrestOutcome = iota // pending, still within grace
	arrestOutcomeDeclare                      // first sight — stamp & say intent
	arrestOutcomeHaul                         // past grace — call ExecuteArrest
	arrestOutcomeAttack                       // resist policy — issue attack command
)

// resolveWarn is the pure escalation decision for a Warn-severity player.
func resolveWarn(alreadyWarned bool, warnedRound, nowRound, grace uint64) warnOutcome {
	if !alreadyWarned {
		return warnOutcomeWarn
	}
	if nowRound >= warnedRound && nowRound-warnedRound >= grace {
		return warnOutcomeAttack
	}
	return warnOutcomeNone
}

// resolveArrest is the pure escalation decision for a SeverityArrest player.
// resist=true means the player's ArrestPolicy is resist.
// pending=true means the guard already stamped a pending-arrest round.
func resolveArrest(resist, pending bool, pendingRound, nowRound, grace uint64) arrestOutcome {
	if resist {
		return arrestOutcomeAttack
	}
	if !pending {
		return arrestOutcomeDeclare
	}
	if nowRound >= pendingRound && nowRound-pendingRound >= grace {
		return arrestOutcomeHaul
	}
	return arrestOutcomeNone
}

// miscDataRound reads a round value stored in MiscData under key, tolerating
// the numeric kinds a YAML round-trip can produce.
func miscDataRound(misc map[string]any, key string) (uint64, bool) {
	v, ok := misc[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case uint64:
		return n, true
	case int64:
		return uint64(n), true
	case int:
		return uint64(n), true
	case float64:
		return uint64(n), true
	}
	return 0, false
}

// defaultGuardWarnGraceRounds mirrors the GuardWarnGraceRounds config default
// (internal/configs/config.balance.mobs.go); used when the knob is unset.
const defaultGuardWarnGraceRounds = 50

func warnGraceRounds() uint64 {
	v := configs.GetBalanceConfig().GuardWarnGraceRounds
	if v < 1 {
		return defaultGuardWarnGraceRounds
	}
	return uint64(v)
}

// defaultArrestGraceRounds mirrors the ArrestResistGraceRounds config default.
const defaultArrestGraceRounds = 3

func arrestGraceRounds() uint64 {
	v := configs.GetBalanceConfig().ArrestResistGraceRounds
	if v < 1 {
		return defaultArrestGraceRounds
	}
	return uint64(v)
}

// warnStampStaleAfter returns the number of rounds after which an unseen
// justice_warned_* stamp is considered stale and eligible for pruning.
// Reuses JusticeCrimeLookbackRounds (the same recency window the Verdict
// function uses) so the two windows stay in sync with one config knob.
func warnStampStaleAfter() uint64 {
	v := configs.GetBalanceConfig().JusticeCrimeLookbackRounds
	if v < 1 {
		return 1000
	}
	return uint64(v)
}

// pruneStaleWarnStamps deletes justice_warned_* entries older than staleAfter
// rounds. Cold-rep warn stamps are never revisited once a player's rep
// recovers (the Warn branch stops running), so they would otherwise leak.
// Leaves justice_arrest_pending_* (self-cleaning on haul) and all other keys.
// Deleting from a map during range over its keys is safe in Go.
func pruneStaleWarnStamps(md map[string]any, now, staleAfter uint64) {
	for key := range md {
		if !strings.HasPrefix(key, "justice_warned_") {
			continue
		}
		stamped, ok := miscDataRound(md, key)
		if !ok {
			continue
		}
		if now >= stamped && now-stamped > staleAfter {
			delete(md, key)
		}
	}
}

// executeArrestFn seam — tests override to intercept without live mobs.
var executeArrestFn = func(player *characters.Character, userId int, faction string, isMurder bool) bool {
	return ExecuteArrest(player, userId, faction, isMurder)
}

// firstFactionWithCell returns the first faction in order that owns a
// holding cell, or "" if none do. Used to pick the arresting faction when a
// guard belongs to several factions (e.g. guards + citizens).
func firstFactionWithCell(guardFactions []string) string {
	for _, f := range guardFactions {
		if cellRoomFn(f) != 0 {
			return f
		}
	}
	return ""
}

// RunGuardEnforcement scans players in the room and applies warn/attack
// against wanted players for this guard, managing warn-grace memory in the
// guard's MiscData. Returns the actions taken (for tests). Both the per-round
// tick (now) and a future protection-faction btree action (later) call this.
func RunGuardEnforcement(mob *mobs.Mob, room *rooms.Room, nowRound uint64) []EnforceAction {
	if mob == nil || room == nil || mob.Character.IsInCombat() || mob.Character.IsCharmed() {
		return nil
	}
	guardFactions := factions.FactionsForMob(mob)
	if len(guardFactions) == 0 {
		return nil
	}

	// Sweep stale warn stamps once per tick per guard. Warn stamps are written
	// when a Cold-rep player is first sighted, but never cleared once the
	// player's rep recovers (the Warn branch simply stops firing). Without this
	// sweep they accumulate on the guard's MiscData indefinitely.
	pruneStaleWarnStamps(mob.Character.MiscData, nowRound, warnStampStaleAfter())

	grace := warnGraceRounds()
	var acts []EnforceAction

	for _, uid := range room.GetPlayers(rooms.FindAll) {
		user := users.GetByUserId(uid)
		if user == nil {
			continue
		}
		if user.Character.HasBuffFlag(buffs.NoAggroTarget) ||
			user.Character.IsHidden() || user.Character.Health < 1 {
			continue
		}
		// A player already serving a sentence is in custody — guards leave them
		// be. Without this, a wandering guard enters the cell and re-arrests or
		// attacks someone already locked up (5.1c smoke BUG-04).
		if _, jailed := miscDataRound(user.Character.MiscData, keyJailUntilRound); jailed {
			continue
		}

		sev := Verdict(guardFactions, uid)
		switch sev {
		case SeverityAttack:
			// Intentionally leaves any prior warn-stamp in MiscData; it is
			// inert while SeverityAttack applies and harmless if rep recovers.
			mob.Command(fmt.Sprintf("attack @%d", uid))
			acts = append(acts, EnforceAction{uid, SeverityAttack, false})
		case SeverityWarn:
			key := fmt.Sprintf("justice_warned_%d", uid)
			warnedRound, warned := miscDataRound(mob.Character.MiscData, key)
			switch resolveWarn(warned, warnedRound, nowRound, grace) {
			case warnOutcomeWarn:
				guardSayFn(room, mob, "Move along — you're not welcome here.")
				mob.Character.SetMiscData(key, nowRound)
				acts = append(acts, EnforceAction{uid, SeverityWarn, false})
			case warnOutcomeAttack:
				mob.Command(fmt.Sprintf("attack @%d", uid))
				acts = append(acts, EnforceAction{uid, SeverityAttack, true})
			}
		case SeverityArrest:
			resist := user.Character.ArrestPolicy == characters.ArrestResist
			pendingKey := fmt.Sprintf("justice_arrest_pending_%d", uid)
			pendingRound, pending := miscDataRound(mob.Character.MiscData, pendingKey)
			switch resolveArrest(resist, pending, pendingRound, nowRound, arrestGraceRounds()) {
			case arrestOutcomeAttack:
				mob.Command(fmt.Sprintf("attack @%d", uid))
				acts = append(acts, EnforceAction{uid, SeverityAttack, false})
			case arrestOutcomeDeclare:
				guardSayFn(room, mob,
					"Move along is past — you're under arrest. Come quietly.")
				mob.Character.SetMiscData(pendingKey, nowRound)
				acts = append(acts, EnforceAction{uid, SeverityArrest, false})
			case arrestOutcomeHaul:
				faction := firstFactionWithCell(guardFactions)
				if faction == "" {
					// No arresting faction owns a cell — cannot haul; leave
					// the pending stamp so the declaration line isn't lost.
					break
				}
				executeArrestFn(user.Character, uid, faction, false)
				mob.Character.SetMiscData(pendingKey, nil)
				acts = append(acts, EnforceAction{uid, SeverityArrest, true})
			}
		}
	}
	return acts
}

// guardSayFn speaks a guard's line. Default is a no-op so package justice has no
// dependency on internal/actions; internal/hooks wires the real broadcaster at
// init (hooks/justice_wiring.go). Injectability also breaks the actions↔justice
// import cycle so crime sites in internal/actions can call MaybeDeclareBounty.
var guardSayFn = func(room *rooms.Room, mob *mobs.Mob, line string) {}

// SetGuardSay installs the guard-speech implementation (called once from
// internal/hooks at init). A nil fn is ignored, keeping the no-op default.
func SetGuardSay(fn func(room *rooms.Room, mob *mobs.Mob, line string)) {
	if fn != nil {
		guardSayFn = fn
	}
}
