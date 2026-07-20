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

// Throttle is a fanged beast attack that clamps the throat, dealing damage,
// applying a stamina-drain DoT, and potentially interrupting an active spellcast.
func Throttle(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use throttle; silently skip if not in combat.
	if !mob.Character.IsInCombat() {
		return true, nil
	}

	res := actions.ExecuteThrottle(&actions.MobActor{Mob: mob, Room: room})

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
				targetUser.SendText(messaging.CategoryHitNaturalSharp,
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> clamps crushing fangs around your throat, cutting off your air! (<ansi fg="damage">%s</ansi>)`, mobName, dmgDesc))
			} else {
				targetUser.SendText(messaging.CategoryHitNaturalSharp,
					fmt.Sprintf(`Something crushes your throat with savage fangs! (<ansi fg="damage">%s</ansi>)`, dmgDesc))
			}

			// Cast-interrupt message.
			if res.InterruptedCast {
				if canSee {
					targetUser.SendText(messaging.CategorySystem,
						fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s choke shatters your concentration — your spell collapses!`, mobName))
				} else {
					targetUser.SendText(messaging.CategorySystem,
						`The crushing grip shatters your concentration — your spell collapses!`)
				}
			}
		}
		room.SendTextVisual(messaging.CategoryHitNaturalSharp,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> clamps crushing fangs around <ansi fg="username">%s</ansi>'s throat!`, mobName, target.Name),
			target.UserId)
	} else {
		if targetUser != nil {
			if canSee {
				targetUser.SendText(messaging.CategoryHitNaturalSharp,
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> lunges for your throat but misses!`, mobName))
			} else {
				targetUser.SendText(messaging.CategoryHitNaturalSharp,
					`Something snaps at your throat but misses!`)
			}
		}
		room.SendTextVisual(messaging.CategoryHitNaturalSharp,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> lunges for <ansi fg="username">%s</ansi>'s throat but misses!`, mobName, target.Name),
			target.UserId)
	}

	return true, nil
}
