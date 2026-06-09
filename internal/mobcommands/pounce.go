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

// Pounce is a quadruped predator's leaping opener: it deals bonus damage and
// attempts to knock the target backward (Supine). Requires legs and a
// fanged or clawed natural attack.
func Pounce(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use pounce; silently skip if not in combat.
	if !mob.Character.IsInCombat() {
		return true, nil
	}

	res := actions.ExecutePounce(&actions.MobActor{Mob: mob, Room: room})

	// Anatomy/identity refusal: silently swallow so the btree can fall through.
	if res.NotPredator {
		return true, nil
	}
	// Any other early-exit condition (OnCooldown, NoTarget): silently return.
	if !res.Executed {
		return true, nil
	}

	// Format and send darkness-aware messages.
	target := res.Target
	result := res.MoveResult
	mobName := mob.Character.Name
	dmgDesc := combat.GetDamageDescription(result.Damage, result.TargetMaxHP)

	// Look up target player record for darkness-aware personal messaging.
	var targetUser *users.UserRecord
	if target.UserId > 0 {
		targetUser = users.GetByUserId(target.UserId)
	}
	canSee := targetUser == nil || canSeeInDark(targetUser, room)

	if result.Hit {
		if result.KnockedDown {
			// Hit + knockdown: predator bears prey to the ground.
			if targetUser != nil {
				if canSee {
					targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> leaps at you and slams you to the ground! (<ansi fg="damage">%s</ansi> damage)`, mobName, dmgDesc))
				} else {
					targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(`Something leaps at you and slams you to the ground! (<ansi fg="damage">%s</ansi> damage)`, dmgDesc))
				}
			}
			sendRoomText(room, messaging.CategoryHitNaturalSharp,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> leaps at <ansi fg="username">%s</ansi> and slams them to the ground!`, mobName, target.Name),
				target.UserId)
		} else {
			// Hit but no knockdown: landed but target kept their feet.
			if targetUser != nil {
				if canSee {
					targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> springs at you and crashes into your body! (<ansi fg="damage">%s</ansi> damage)`, mobName, dmgDesc))
				} else {
					targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(`Something springs at you and crashes into your body! (<ansi fg="damage">%s</ansi> damage)`, dmgDesc))
				}
			}
			sendRoomText(room, messaging.CategoryHitNaturalSharp,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> springs at <ansi fg="username">%s</ansi> and crashes into them!`, mobName, target.Name),
				target.UserId)
		}
	} else {
		if targetUser != nil {
			if canSee {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> leaps at you, but you sidestep the pounce!`, mobName))
			} else {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, `Something leaps at you, but you sidestep!`)
			}
		}
		sendRoomText(room, messaging.CategoryHitNaturalSharp,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> leaps at <ansi fg="username">%s</ansi>, but misses!`, mobName, target.Name),
			target.UserId)
	}

	return true, nil
}
