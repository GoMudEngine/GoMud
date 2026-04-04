package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
Skullduggery Skill
Level 3 - Defuse: attempt to disarm a trap on a locked exit or container.
Consumes a disarm kit. Success clears trapbuffids; failure triggers the trap.
*/
func Defuse(rest string, user *users.UserRecord,
	room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Skullduggery)
	if skillLevel < 1 {
		return false, nil
	}
	if skillLevel < 3 {
		user.SendText("You aren't advanced enough at skullduggery for that.")
		return true, nil
	}

	if user.Character.Aggro != nil {
		user.SendText("You can't do that while in combat!")
		return true, nil
	}

	if room.AreMobsAttacking(user.UserId) {
		user.SendText("You can't do that while you are under attack!")
		return true, nil
	}

	if rest == "" {
		user.SendText("Defuse what? Specify an exit or container.")
		return true, nil
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))
	if len(args) < 1 {
		user.SendText("Defuse what? Specify an exit or container.")
		return true, nil
	}

	// ── Locate the target lock ──────────────────────────────────────────────

	type lockTarget int
	const (
		targetNone      lockTarget = iota
		targetContainer lockTarget = iota
		targetExit      lockTarget = iota
	)

	which := targetNone
	containerName := ``
	exitName := ``
	lockTrap := []int{}
	lockDifficulty := 0

	containerName = room.FindContainerByName(args[0])
	if containerName != `` {
		// Respect hidden-container discovery
		if c, exists := room.Containers[containerName]; exists && c.Hidden {
			if !user.Character.HasDiscovery(room.RoomId, containerName) {
				containerName = ``
			}
		}
	}

	if containerName != `` {
		container := room.Containers[containerName]
		if !container.HasLock() {
			user.SendText("There is no lock there.")
			return true, nil
		}
		if len(container.Lock.TrapBuffIds) == 0 {
			user.SendText("You don't detect any traps on that.")
			return true, nil
		}
		lockTrap = container.Lock.TrapBuffIds
		lockDifficulty = int(container.Lock.Difficulty)
		which = targetContainer
	} else {
		exitName, _ = room.FindExitByName(args[0])
		if exitName != `` {
			exitInfo, _ := room.GetExitInfo(exitName)
			if !exitInfo.HasLock() {
				user.SendText("There is no lock there.")
				return true, nil
			}
			if len(exitInfo.Lock.TrapBuffIds) == 0 {
				user.SendText("You don't detect any traps on that.")
				return true, nil
			}
			lockTrap = exitInfo.Lock.TrapBuffIds
			lockDifficulty = int(exitInfo.Lock.Difficulty)
			which = targetExit
		}
	}

	if which == targetNone {
		user.SendText("There is no such exit or container.")
		return true, nil
	}

	// ── Find a disarm kit in the player's backpack ──────────────────────────

	disarmKit := items.Item{}
	for _, itm := range user.Character.GetAllBackpackItems() {
		spec := itm.GetSpec()
		if strings.Contains(strings.ToLower(spec.Name), "disarm kit") {
			disarmKit = itm
			break
		}
	}

	if disarmKit.ItemId < 1 {
		user.SendText(`You need a <ansi fg="item">disarm kit</ansi> to attempt this.`)
		return true, nil
	}

	// ── Consume the disarm kit regardless of outcome ────────────────────────

	user.Character.UseItem(disarmKit)

	// ── Fire skill-use event (for progression) ──────────────────────────────

	defer user.Character.CheckSkillProgression(string(skills.Skullduggery), user.UserId, 1.0)

	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.Skullduggery,
		Details: `defuse`,
	})

	// ── Opposed roll: Perception + skullduggery vs trap difficulty ───────────

	kitBonus := disarmKit.StatMod("defuse_bonus")
	defuseScore := float64(user.Character.Stats.Perception.ValueAdj) +
		float64(skillLevel)*25.0 +
		float64(kitBonus)
	trapDifficulty := float64(lockDifficulty) * 10.0

	success, _, _, _ := dice.OpposedRollStat(defuseScore, trapDifficulty)

	if success {
		// ── Clear the trap ──────────────────────────────────────────────────
		if which == targetContainer {
			container := room.Containers[containerName]
			container.Lock.TrapBuffIds = nil
			room.Containers[containerName] = container
			user.SendText(`<ansi fg="green">You carefully disarm the trap mechanism.</ansi>`)
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> disarms a trap on the <ansi fg="container">%s</ansi>.`,
					user.Character.Name, containerName),
				user.UserId)
		} else {
			// Modify the exit's trap in-place by writing back through SetExitTrap helper.
			// Exits are value-type map entries; read → mutate → write back.
			if exitInfo, ok := room.Exits[exitName]; ok {
				exitInfo.Lock.TrapBuffIds = nil
				room.Exits[exitName] = exitInfo
			}
			user.SendText(`<ansi fg="green">You carefully disarm the trap mechanism.</ansi>`)
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> disarms a trap on the <ansi fg="exit">%s</ansi> exit.`,
					user.Character.Name, exitName),
				user.UserId)
		}
	} else {
		// ── Trigger the trap ────────────────────────────────────────────────
		user.SendText(`<ansi fg="red-bold">The trap triggers as you fumble the mechanism!</ansi>`)
		room.SendText(
			fmt.Sprintf(`<ansi fg="alert-3"><ansi fg="username">%s</ansi> triggers a trap!</ansi>`,
				user.Character.Name),
			user.UserId)
		for _, buffId := range lockTrap {
			user.AddBuff(buffId, `trap`)
		}
	}

	return true, nil
}
