package actions

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// DefuseOptions parameterizes a defuse attempt.
// TargetNoun names the container or exit that holds the trap. When empty the
// action returns Reason:"no target". Mob callers that want to target the first
// available trapped lock should resolve the noun before calling.
type DefuseOptions struct {
	TargetNoun string
}

// DefuseResult is the structured outcome of a defuse attempt.
type DefuseResult struct {
	Succeeded   bool   // skill check passed and trap was removed
	TrapName    string // display name of the targeted container/exit
	KitConsumed bool   // disarm kit was consumed (always true when a kit was
	//                       present and the attempt proceeded past the kit gate)
	KitBonusUsed   int    // stat bonus contributed by the consumed kit
	TriggeredTraps []int  // on failure, buff IDs of traps that fired
	Reason         string // when Succeeded==false, why (empty on success)
}

// lockTargetKind distinguishes container vs. exit locks.
type lockTargetKind int

const (
	lockTargetNone      lockTargetKind = iota
	lockTargetContainer lockTargetKind = iota
	lockTargetExit      lockTargetKind = iota
)

// defuseLockTarget carries resolved trap information.
type defuseLockTarget struct {
	kind           lockTargetKind
	containerName  string
	exitName       string
	lockTrap       []int
	lockDifficulty int
}

// Defuse attempts to disarm a trap on a room exit or container lock.
//
// A disarm kit is REQUIRED and is consumed regardless of outcome (matching
// source behavior). No cooldown is applied — the source command has none.
// Skullduggery rank 3 is the minimum.
func Defuse(actor Actor, opts DefuseOptions) DefuseResult {
	char := actor.GetCharacter()

	// Skill-rank gates (verbatim from source).
	skillLevel := char.GetSkillLevel(skills.Skullduggery)
	if skillLevel < 1 {
		return DefuseResult{Reason: "not advanced enough"}
	}
	if skillLevel < 3 {
		actor.SendText(messaging.CategorySystem, "You aren't advanced enough at skullduggery for that.")
		return DefuseResult{Reason: "not advanced enough"}
	}

	// Combat gate.
	if char.Aggro != nil {
		actor.SendText(messaging.CategorySystem, "You can't do that while in combat!")
		return DefuseResult{Reason: "in combat"}
	}

	room := actor.GetRoom()
	if room == nil {
		return DefuseResult{Reason: "no room"}
	}

	if room.AreMobsAttacking(actor.GetUserId()) {
		actor.SendText(messaging.CategorySystem, "You can't do that while you are under attack!")
		return DefuseResult{Reason: "under attack"}
	}

	// Require a target noun.
	if opts.TargetNoun == "" {
		actor.SendText(messaging.CategorySystem, "Defuse what? Specify an exit or container.")
		return DefuseResult{Reason: "no target"}
	}

	// ── Locate the target lock ──────────────────────────────────────────────

	noun := strings.ToLower(opts.TargetNoun)
	tgt := resolveDefuseLockTarget(actor, room, noun)

	if tgt.kind == lockTargetNone {
		// resolveDefuseLockTarget already sent the specific error message.
		return DefuseResult{Reason: "no trap"}
	}

	// ── Find a disarm kit in the actor's backpack ───────────────────────────

	kitBonus := 0
	kitFound := false

	for _, itm := range char.GetAllBackpackItems() {
		spec := itm.GetSpec()
		if strings.Contains(strings.ToLower(spec.Name), "disarm kit") {
			kitBonus = itm.StatMod("defuse_bonus")
			char.UseItem(itm)
			kitFound = true
			break
		}
	}

	if !kitFound {
		actor.SendText(messaging.CategorySystem, `You need a <ansi fg="item">disarm kit</ansi> to attempt this.`)
		return DefuseResult{
			TrapName: trapTargetDisplayName(tgt),
			Reason:   "no disarm kit",
		}
	}

	// ── Skill-use event (progression) ──────────────────────────────────────
	// actor.OnSkillUse handles CheckSkillProgression. The SkillUsed event is
	// added for player actors (mirrors the source's defer + events pattern).

	actor.OnSkillUse(string(skills.Skullduggery))

	if actor.IsPlayer() {
		events.AddToQueue(events.SkillUsed{
			UserId:  actor.GetUserId(),
			Skill:   skills.Skullduggery,
			Details: `defuse`,
		})
	}

	// ── Opposed roll: (Per + skillLevel*25 + kitBonus) vs (difficulty*10) ──

	defuseScore := float64(char.Stats.Perception.ValueAdj) +
		float64(skillLevel)*25.0 +
		float64(kitBonus)
	trapDifficulty := float64(tgt.lockDifficulty) * 10.0

	success, _, _, _ := dice.OpposedRollStat(defuseScore, trapDifficulty)

	displayName := trapTargetDisplayName(tgt)

	if success {
		clearDefuseTrap(actor, room, tgt)
		return DefuseResult{
			Succeeded:    true,
			TrapName:     displayName,
			KitConsumed:  true,
			KitBonusUsed: kitBonus,
		}
	}

	// ── Failure: trigger the trap ───────────────────────────────────────────

	actor.SendText(messaging.CategorySystem, `<ansi fg="red-bold">The trap triggers as you fumble the mechanism!</ansi>`)
	room.SendTextVisual(messaging.CategoryMobEmote,
		fmt.Sprintf(`<ansi fg="alert-3"><ansi fg="username">%s</ansi> triggers a trap!</ansi>`,
			actor.GetName()),
		actor.GetUserId(),
	)

	for _, buffId := range tgt.lockTrap {
		actor.AddBuff(buffId, `trap`)
	}

	return DefuseResult{
		Succeeded:      false,
		TrapName:       displayName,
		KitConsumed:    true,
		KitBonusUsed:   kitBonus,
		TriggeredTraps: tgt.lockTrap,
		Reason:         "trap triggered",
	}
}

// resolveDefuseLockTarget looks up the noun in room containers and exits,
// validates lock/trap presence, and sends error messages to the actor on
// failure. Returns lockTargetNone if nothing usable is found.
func resolveDefuseLockTarget(actor Actor, room *rooms.Room, noun string) defuseLockTarget {
	// Try containers first.
	containerName := room.FindContainerByName(noun)
	if containerName != "" {
		// Respect hidden-container discovery (source behavior).
		if c, exists := room.Containers[containerName]; exists && c.Hidden {
			discovered := false
			if actor.IsPlayer() {
				discovered = actor.GetCharacter().HasDiscovery(room.RoomId,
					containerName)
			}
			if !discovered {
				containerName = ""
			}
		}
	}

	if containerName != "" {
		container := room.Containers[containerName]
		if !container.HasLock() {
			actor.SendText(messaging.CategorySystem, "There is no lock there.")
			return defuseLockTarget{kind: lockTargetNone}
		}
		if len(container.Lock.TrapBuffIds) == 0 {
			actor.SendText(messaging.CategorySystem, "You don't detect any traps on that.")
			return defuseLockTarget{kind: lockTargetNone}
		}
		return defuseLockTarget{
			kind:           lockTargetContainer,
			containerName:  containerName,
			lockTrap:       container.Lock.TrapBuffIds,
			lockDifficulty: int(container.Lock.Difficulty),
		}
	}

	// Try exits.
	exitName, _ := room.FindExitByName(noun)
	if exitName != "" {
		exitInfo, ok := room.GetExitInfo(exitName)
		if ok {
			if !exitInfo.HasLock() {
				actor.SendText(messaging.CategorySystem, "There is no lock there.")
				return defuseLockTarget{kind: lockTargetNone}
			}
			if len(exitInfo.Lock.TrapBuffIds) == 0 {
				actor.SendText(messaging.CategorySystem, "You don't detect any traps on that.")
				return defuseLockTarget{kind: lockTargetNone}
			}
			return defuseLockTarget{
				kind:           lockTargetExit,
				exitName:       exitName,
				lockTrap:       exitInfo.Lock.TrapBuffIds,
				lockDifficulty: int(exitInfo.Lock.Difficulty),
			}
		}
	}

	actor.SendText(messaging.CategorySystem, "There is no such exit or container.")
	return defuseLockTarget{kind: lockTargetNone}
}

// clearDefuseTrap removes the trap buff IDs from the target lock and notifies
// the room on success.
func clearDefuseTrap(actor Actor, room *rooms.Room, tgt defuseLockTarget) {
	switch tgt.kind {
	case lockTargetContainer:
		container := room.Containers[tgt.containerName]
		container.Lock.TrapBuffIds = nil
		room.Containers[tgt.containerName] = container
		actor.SendText(messaging.CategorySystem, `<ansi fg="green">You carefully disarm the trap mechanism.</ansi>`)
		room.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(
				`<ansi fg="username">%s</ansi> disarms a trap on the `+
					`<ansi fg="container">%s</ansi>.`,
				actor.GetName(), tgt.containerName),
			actor.GetUserId())

	case lockTargetExit:
		// MarkExitTrapDefused both clears the live trap and records the exit
		// name so the disarm survives a restart/copyover. Room.Exits is
		// instance:"skip", so writing the cleared exit alone would be undone by
		// restoreSkipTaggedFields on the next load — the container branch above
		// persists for free because Room.Containers is not skip-tagged.
		room.MarkExitTrapDefused(tgt.exitName)
		actor.SendText(messaging.CategorySystem, `<ansi fg="green">You carefully disarm the trap mechanism.</ansi>`)
		room.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(
				`<ansi fg="username">%s</ansi> disarms a trap on the `+
					`<ansi fg="exit">%s</ansi> exit.`,
				actor.GetName(), tgt.exitName),
			actor.GetUserId())
	}
}

// trapTargetDisplayName returns a human-readable label for the targeted lock.
func trapTargetDisplayName(t defuseLockTarget) string {
	switch t.kind {
	case lockTargetContainer:
		return t.containerName
	case lockTargetExit:
		return t.exitName
	}
	return ""
}
