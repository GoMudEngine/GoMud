package justice

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
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
