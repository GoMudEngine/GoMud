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

// Rake is a clawed beast attack that deals damage and applies a bleed
// condition.
func Rake(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use rake; silently skip if not in combat.
	if !mob.Character.IsInCombat() {
		return true, nil
	}

	res := actions.ExecuteRake(&actions.MobActor{Mob: mob, Room: room})

	// Any early-exit condition: silently return.
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
		if targetUser != nil {
			if canSee {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> rakes its claws across you, opening bleeding wounds! (<ansi fg="damage">%s</ansi> damage)`, mobName, dmgDesc))
			} else {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(`Something rakes its claws across you, opening bleeding wounds! (<ansi fg="damage">%s</ansi> damage)`, dmgDesc))
			}
		}
		sendRoomText(room, messaging.CategoryHitNaturalSharp,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> rakes its claws across <ansi fg="username">%s</ansi>!`, mobName, target.Name),
			target.UserId)
	} else {
		if targetUser != nil {
			if canSee {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> swipes its claws at you, but misses!`, mobName))
			} else {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, `Something swipes its claws at you, but misses!`)
			}
		}
		sendRoomText(room, messaging.CategoryHitNaturalSharp,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> swipes its claws at <ansi fg="username">%s</ansi>, but misses!`, mobName, target.Name),
			target.UserId)
	}

	return true, nil
}
