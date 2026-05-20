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

// Bite is a vampire-only special attack that deals moderate physical damage
// and heals the vampire for half the damage inflicted (life drain).
func Bite(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat.
	if !mob.Character.IsInCombat() {
		return true, nil
	}

	res := actions.ExecuteBite(&actions.MobActor{Mob: mob, Room: room})

	if res.OnCooldown || res.NoTarget || !res.Executed {
		return true, nil
	}

	target := res.Target
	result := res.MoveResult

	// Messaging — darkness-aware.
	mobName := mob.Character.Name
	dmgDesc := combat.GetDamageDescription(result.Damage, result.TargetMaxHP)

	var targetUser *users.UserRecord
	if target.UserId > 0 {
		targetUser = users.GetByUserId(target.UserId)
	}
	canSee := targetUser == nil || canSeeInDark(targetUser, room)

	if result.Hit {
		if targetUser != nil {
			if canSee {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> sinks its fangs into you, drawing strength from the wound! (<ansi fg="damage">%s</ansi> damage)`,
					mobName, dmgDesc))
			} else {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(
					`Something sinks its fangs into you in the darkness, drawing strength from the wound! (<ansi fg="damage">%s</ansi> damage)`,
					dmgDesc))
			}
		}
		if result.Damage > 0 {
			sendRoomText(room, messaging.CategoryHitNaturalSharp,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> sinks its fangs into <ansi fg="username">%s</ansi> and draws strength from the wound!`,
					mobName, target.Name),
				target.UserId)
		} else {
			sendRoomText(room, messaging.CategoryHitNaturalSharp,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> sinks its fangs into <ansi fg="username">%s</ansi>!`,
					mobName, target.Name),
				target.UserId)
		}
	} else {
		if targetUser != nil {
			if canSee {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> snaps its fangs at you, but misses!`,
					mobName))
			} else {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, `Something snaps its fangs at you in the darkness, but misses!`)
			}
		}
		sendRoomText(room, messaging.CategoryHitNaturalSharp,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> snaps its fangs at <ansi fg="username">%s</ansi>, but misses!`,
				mobName, target.Name),
			target.UserId)
	}

	return true, nil
}
