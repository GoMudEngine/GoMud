package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Hamstring is a wolf physical attack that applies a bleed condition.
func Hamstring(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if !mob.Character.IsInCombat() {
		return true, nil
	}

	res := actions.ExecuteHamstring(&actions.MobActor{Mob: mob, Room: room})

	if res.OnCooldown || res.NoTarget || !res.Executed {
		return true, nil
	}

	target := res.Target
	result := res.MoveResult

	mobName := mob.Character.Name
	targetPlayerId := target.UserId

	// Resolve the target user record for direct messaging (player targets).
	var targetChar *users.UserRecord
	if target.UserId > 0 {
		targetChar = users.GetByUserId(target.UserId)
	}

	canSee := targetChar == nil || canSeeInDark(targetChar, room)

	if result.Hit {
		dmgDesc := combat.GetDamageDescription(result.Damage, result.TargetMaxHP)
		if targetChar != nil {
			if canSee {
				targetChar.SendTextLegacy(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> rakes its fangs across your legs, opening deep wounds! (<ansi fg="damage">%s</ansi> damage)`, mobName, dmgDesc))
			} else {
				targetChar.SendTextLegacy(fmt.Sprintf(`Something rakes its fangs across your legs, opening deep wounds! (<ansi fg="damage">%s</ansi> damage)`, dmgDesc))
			}
		}
		sendRoomText(room,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> lunges low and rakes its fangs across <ansi fg="username">%s</ansi>'s legs!`, mobName, target.Name),
			targetPlayerId)
	} else {
		if targetChar != nil {
			if canSee {
				targetChar.SendTextLegacy(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> lunges at your legs, but you sidestep the attack!`, mobName))
			} else {
				targetChar.SendTextLegacy(`Something lunges at your legs, but you sidestep the attack!`)
			}
		}
		sendRoomText(room,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> lunges at <ansi fg="username">%s</ansi>'s legs, but misses!`, mobName, target.Name),
			targetPlayerId)
	}

	return true, nil
}
