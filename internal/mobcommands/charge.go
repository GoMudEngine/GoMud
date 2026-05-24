package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Charge is a boar trip variant — same mechanics as trip, charge-specific
// narration. Trip resolution (skill move, knockdown roll, prone application,
// analytics, round consumption) is delegated to actions.ExecuteTrip; only
// the charge-specific messages are handled here.
func Charge(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if !mob.Character.IsInCombat() {
		return true, nil
	}

	res := actions.ExecuteTrip(actions.NewMobActorInRoom(mob, room))

	if res.OnCooldown || res.NoTarget || !res.Executed {
		return true, nil
	}

	target := res.Target
	result := res.MoveResult

	mobName := mob.Character.Name
	targetName := target.Name
	targetPlayerId := target.UserId

	// Resolve the target user record for direct messaging (player targets only).
	var targetChar *users.UserRecord
	if target.UserId > 0 {
		targetChar = users.GetByUserId(target.UserId)
	}

	canSee := targetChar == nil || canSeeInDark(targetChar, room)
	dmgDesc := combat.GetDamageDescription(result.Damage, result.TargetMaxHP)

	if result.Hit {
		if result.KnockedDown {
			if targetChar != nil {
				if canSee {
					targetChar.SendText(messaging.CategoryTrip, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> charges and slams into you, sending you sprawling! (<ansi fg="damage">%s</ansi> damage)`, mobName, dmgDesc))
				} else {
					targetChar.SendText(messaging.CategoryTrip, fmt.Sprintf(`Something charges and slams into you, sending you sprawling! (<ansi fg="damage">%s</ansi> damage)`, dmgDesc))
				}
			}
			sendRoomText(room, messaging.CategoryTrip,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> charges and slams into <ansi fg="username">%s</ansi>, sending them sprawling!`, mobName, targetName),
				targetPlayerId)
		} else {
			if targetChar != nil {
				if canSee {
					targetChar.SendText(messaging.CategoryTrip, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> charges at you, but you keep your footing! (<ansi fg="damage">%s</ansi> damage)`, mobName, dmgDesc))
				} else {
					targetChar.SendText(messaging.CategoryTrip, fmt.Sprintf(`Something charges at you, but you keep your footing! (<ansi fg="damage">%s</ansi> damage)`, dmgDesc))
				}
			}
			sendRoomText(room, messaging.CategoryTrip,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> charges at <ansi fg="username">%s</ansi>, but they keep their footing!`, mobName, targetName),
				targetPlayerId)
		}
	} else {
		if targetChar != nil {
			if canSee {
				targetChar.SendText(messaging.CategoryTrip, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> charges past you, missing entirely!`, mobName))
			} else {
				targetChar.SendText(messaging.CategoryTrip, `Something charges past you, missing entirely!`)
			}
		}
		sendRoomText(room, messaging.CategoryTrip,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> charges past <ansi fg="username">%s</ansi>, missing entirely!`, mobName, targetName),
			targetPlayerId)
	}

	return true, nil
}
