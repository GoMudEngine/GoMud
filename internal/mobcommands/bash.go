package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Bash(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use bash; silently skip if no aggro.
	if mob.Character.Aggro == nil {
		return true, nil
	}

	// Delegate core bash logic to the shared action.
	bashResult := actions.ExecuteBash(&actions.MobActor{Mob: mob, Room: room})

	// Any early-exit condition: silently return.
	if !bashResult.Executed {
		return true, nil
	}

	// Fire skill progression for the executed special move.
	mob.Character.OnSkillUse(string(skills.WeaponCombat), 0)

	// Format and send darkness-aware messages.
	target := bashResult.Target
	result := bashResult.MoveResult
	mobName := mob.Character.Name
	dmgDesc := combat.GetDamageDescription(result.Damage, result.TargetMaxHP)

	// Look up the target player record for darkness-aware personal messaging.
	var targetUser *users.UserRecord
	if target.UserId > 0 {
		targetUser = users.GetByUserId(target.UserId)
	}
	canSee := targetUser == nil || canSeeInDark(targetUser, room)

	if result.Hit {
		if result.KnockedDown {
			if targetUser != nil {
				if canSee {
					targetUser.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi> damage)`, mobName, dmgDesc))
				} else {
					targetUser.SendText(fmt.Sprintf(`Something's <ansi fg="yellow-bold">shield bash</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi> damage)`, dmgDesc))
				}
			}
			sendRoomText(room,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> knocks <ansi fg="username">%s</ansi> to the ground!`, mobName, target.Name),
				target.UserId)
		} else {
			if targetUser != nil {
				if canSee {
					targetUser.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> strikes you! (<ansi fg="damage">%s</ansi> damage)`, mobName, dmgDesc))
				} else {
					targetUser.SendText(fmt.Sprintf(`Something's <ansi fg="yellow-bold">shield bash</ansi> strikes you! (<ansi fg="damage">%s</ansi> damage)`, dmgDesc))
				}
			}
			sendRoomText(room,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> bashes <ansi fg="username">%s</ansi> with their shield!`, mobName, target.Name),
				target.UserId)
		}
	} else {
		if targetUser != nil {
			if canSee {
				targetUser.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to bash you with their shield, but misses!`, mobName))
			} else {
				targetUser.SendText(`Something attempts to bash you with a shield, but misses!`)
			}
		}
		sendRoomText(room,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to bash <ansi fg="username">%s</ansi>, but misses!`, mobName, target.Name),
			target.UserId)
	}

	return true, nil
}
