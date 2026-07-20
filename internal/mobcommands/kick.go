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

func Kick(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use kick; silently skip if not in combat.
	if !mob.Character.IsInCombat() {
		return true, nil
	}

	// Delegate core kick logic to the shared action (includes stomp/knee variant
	// detection so mobs now use the appropriate variant automatically).
	res := actions.ExecuteKick(&actions.MobActor{Mob: mob, Room: room})

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
		switch res.Variant {
		case actions.KickStomp:
			if targetUser != nil {
				if canSee {
					targetUser.SendText(messaging.CategoryKick, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> stomps on you while you're down! (<ansi fg="damage">%s</ansi>)`, mobName, dmgDesc))
				} else {
					targetUser.SendText(messaging.CategoryKick, fmt.Sprintf(`Something stomps on you while you're down! (<ansi fg="damage">%s</ansi>)`, dmgDesc))
				}
			}
			room.SendTextVisual(messaging.CategoryKick,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> stomps on the downed <ansi fg="username">%s</ansi>!`, mobName, target.Name),
				target.UserId)

		case actions.KickKnee:
			if targetUser != nil {
				if canSee {
					targetUser.SendText(messaging.CategoryKick, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> drives a knee into you in the grapple! (<ansi fg="damage">%s</ansi>)`, mobName, dmgDesc))
				} else {
					targetUser.SendText(messaging.CategoryKick, fmt.Sprintf(`Something drives a knee into you! (<ansi fg="damage">%s</ansi>)`, dmgDesc))
				}
			}
			room.SendTextVisual(messaging.CategoryKick,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> drives a knee into <ansi fg="username">%s</ansi>!`, mobName, target.Name),
				target.UserId)

		default: // KickStandard
			if result.KnockedDown {
				if targetUser != nil {
					if canSee {
						targetUser.SendText(messaging.CategoryKick, fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s powerful <ansi fg="yellow-bold">kick</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi>)`, mobName, dmgDesc))
					} else {
						targetUser.SendText(messaging.CategoryKick, fmt.Sprintf(`Something's powerful <ansi fg="yellow-bold">kick</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi>)`, dmgDesc))
					}
				}
				room.SendTextVisual(messaging.CategoryKick,
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> kicks <ansi fg="username">%s</ansi>, knocking them to the ground!`, mobName, target.Name),
					target.UserId)
			} else {
				if targetUser != nil {
					if canSee {
						targetUser.SendText(messaging.CategoryKick, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> kicks you hard! (<ansi fg="damage">%s</ansi>)`, mobName, dmgDesc))
					} else {
						targetUser.SendText(messaging.CategoryKick, fmt.Sprintf(`Something kicks you hard! (<ansi fg="damage">%s</ansi>)`, dmgDesc))
					}
				}
				room.SendTextVisual(messaging.CategoryKick,
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> kicks <ansi fg="username">%s</ansi>!`, mobName, target.Name),
					target.UserId)
			}
		}
	} else {
		switch res.Variant {
		case actions.KickStomp:
			if targetUser != nil {
				if canSee {
					targetUser.SendText(messaging.CategoryKick, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to stomp you, but you roll aside!`, mobName))
				} else {
					targetUser.SendText(messaging.CategoryKick, `Something tries to stomp you, but you roll aside!`)
				}
			}
			room.SendTextVisual(messaging.CategoryKick,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to stomp <ansi fg="username">%s</ansi>, but misses!`, mobName, target.Name),
				target.UserId)

		case actions.KickKnee:
			if targetUser != nil {
				if canSee {
					targetUser.SendText(messaging.CategoryKick, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to knee you, but you block it!`, mobName))
				} else {
					targetUser.SendText(messaging.CategoryKick, `Something tries to knee you, but you block it!`)
				}
			}
			room.SendTextVisual(messaging.CategoryKick,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to knee <ansi fg="username">%s</ansi>, but misses!`, mobName, target.Name),
				target.UserId)

		default: // KickStandard
			if targetUser != nil {
				if canSee {
					targetUser.SendText(messaging.CategoryKick, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to kick you, but misses!`, mobName))
				} else {
					targetUser.SendText(messaging.CategoryKick, `Something attempts to kick you, but misses!`)
				}
			}
			room.SendTextVisual(messaging.CategoryKick,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to kick <ansi fg="username">%s</ansi>, but misses!`, mobName, target.Name),
				target.UserId)
		}
	}

	return true, nil
}
