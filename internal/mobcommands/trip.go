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

func Trip(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use trip
	if !mob.Character.IsInCombat() {
		return true, nil
	}

	res := actions.ExecuteTrip(&actions.MobActor{Mob: mob, Room: room})

	if res.OnCooldown || res.NoTarget || !res.Executed {
		return true, nil
	}

	target := res.Target
	result := res.MoveResult
	hasTail := res.Variant == actions.TripTailsweep

	mobName := mob.Character.Name
	targetName := target.Name
	targetPlayerId := target.UserId

	// Resolve the target user record for direct messaging (player targets).
	var targetChar *users.UserRecord
	if target.UserId > 0 {
		targetChar = users.GetByUserId(target.UserId)
	}

	canSee := targetChar == nil || canSeeInDark(targetChar, room)
	dmgDesc := combat.GetDamageDescription(result.Damage, result.TargetMaxHP)

	if result.Hit {
		if hasTail {
			if result.KnockedDown {
				if targetChar != nil {
					if canSee {
						targetChar.SendText(messaging.CategoryTrip,fmt.Sprintf(`<ansi fg="mobname">%s</ansi> hammers you with their tail, sending you crashing to the ground! (<ansi fg="damage">%s</ansi>)`, mobName, dmgDesc))
					} else {
						targetChar.SendText(messaging.CategoryTrip,fmt.Sprintf(`Something hammers you with a powerful sweep, sending you crashing to the ground! (<ansi fg="damage">%s</ansi>)`, dmgDesc))
					}
				}
				sendRoomText(room, messaging.CategoryTrip,
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tailsweeps <ansi fg="username">%s</ansi>, sending them crashing to the ground!`, mobName, targetName),
					targetPlayerId)
			} else {
				if targetChar != nil {
					if canSee {
						targetChar.SendText(messaging.CategoryTrip,fmt.Sprintf(`<ansi fg="mobname">%s</ansi> sweeps at you with their tail, but you manage to stay upright! (<ansi fg="damage">%s</ansi>)`, mobName, dmgDesc))
					} else {
						targetChar.SendText(messaging.CategoryTrip,fmt.Sprintf(`Something sweeps at you powerfully, but you manage to stay upright! (<ansi fg="damage">%s</ansi>)`, dmgDesc))
					}
				}
				sendRoomText(room, messaging.CategoryTrip,
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tailsweeps <ansi fg="username">%s</ansi>, but they keep their footing!`, mobName, targetName),
					targetPlayerId)
			}
		} else {
			if result.KnockedDown {
				if targetChar != nil {
					if canSee {
						targetChar.SendText(messaging.CategoryTrip,fmt.Sprintf(`<ansi fg="mobname">%s</ansi> sweeps your legs, sending you crashing to the ground! (<ansi fg="damage">%s</ansi>)`, mobName, dmgDesc))
					} else {
						targetChar.SendText(messaging.CategoryTrip,fmt.Sprintf(`Something sweeps your legs, sending you crashing to the ground! (<ansi fg="damage">%s</ansi>)`, dmgDesc))
					}
				}
				sendRoomText(room, messaging.CategoryTrip,
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> trips <ansi fg="username">%s</ansi>, sending them crashing to the ground!`, mobName, targetName),
					targetPlayerId)
			} else {
				if targetChar != nil {
					if canSee {
						targetChar.SendText(messaging.CategoryTrip,fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to trip you, but you keep your footing! (<ansi fg="damage">%s</ansi>)`, mobName, dmgDesc))
					} else {
						targetChar.SendText(messaging.CategoryTrip,fmt.Sprintf(`Something attempts to trip you, but you keep your footing! (<ansi fg="damage">%s</ansi>)`, dmgDesc))
					}
				}
				sendRoomText(room, messaging.CategoryTrip,
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to trip <ansi fg="username">%s</ansi>, but they keep their footing!`, mobName, targetName),
					targetPlayerId)
			}
		}
	} else {
		if hasTail {
			if targetChar != nil {
				if canSee {
					targetChar.SendText(messaging.CategoryTrip,fmt.Sprintf(`<ansi fg="mobname">%s</ansi> swings their tail at you, but you avoid it!`, mobName))
				} else {
					targetChar.SendText(messaging.CategoryTrip,`Something sweeps at you powerfully, but you avoid it!`)
				}
			}
			sendRoomText(room, messaging.CategoryTrip,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts a tailsweep on <ansi fg="username">%s</ansi>, but misses!`, mobName, targetName),
				targetPlayerId)
		} else {
			if targetChar != nil {
				if canSee {
					targetChar.SendText(messaging.CategoryTrip,fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to trip you, but you avoid it!`, mobName))
				} else {
					targetChar.SendText(messaging.CategoryTrip,`Something attempts to trip you, but you avoid it!`)
				}
			}
			sendRoomText(room, messaging.CategoryTrip,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to trip <ansi fg="username">%s</ansi>, but misses!`, mobName, targetName),
				targetPlayerId)
		}
	}

	return true, nil
}
