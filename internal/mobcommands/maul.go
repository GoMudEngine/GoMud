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

// Maul is a fanged beast attack that deals heavy damage and applies a strong
// bleed condition.
func Maul(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use maul; silently skip if not in combat.
	if !mob.Character.IsInCombat() {
		return true, nil
	}

	res := actions.ExecuteMaul(&actions.MobActor{Mob: mob, Room: room})

	// Anatomy/identity refusal: silently swallow so the btree can fall through.
	if res.NotFanged {
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
		if targetUser != nil {
			if canSee {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> savages you with vicious fangs, tearing bleeding wounds! (<ansi fg="damage">%s</ansi>)`, mobName, dmgDesc))
			} else {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(`Something savages you with vicious fangs, tearing bleeding wounds! (<ansi fg="damage">%s</ansi>)`, dmgDesc))
			}
		}
		sendRoomText(room, messaging.CategoryHitNaturalSharp,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> savages <ansi fg="username">%s</ansi> with vicious fangs!`, mobName, target.Name),
			target.UserId)
	} else {
		if targetUser != nil {
			if canSee {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> snaps its fangs at you, but misses!`, mobName))
			} else {
				targetUser.SendText(messaging.CategoryHitNaturalSharp, `Something snaps its fangs at you, but misses!`)
			}
		}
		sendRoomText(room, messaging.CategoryHitNaturalSharp,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> snaps its fangs at <ansi fg="username">%s</ansi>, but misses!`, mobName, target.Name),
			target.UserId)
	}

	return true, nil
}
