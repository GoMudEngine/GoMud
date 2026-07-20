package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Trip(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if actions.AcquireMeleeTarget(user, room, rest, actions.MeleeTargetOpts{
		Verb:         "trip",
		CraftingVerb: "trip someone",
	}) {
		return true, nil
	}

	res := actions.ExecuteTrip(&actions.UserActor{User: user, Room: room})

	if res.Crafting {
		// Safety net — should have been caught by the pre-reject above.
		user.SendText(messaging.CategorySystem, `<ansi fg="red">You can't trip someone while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}

	if res.OnCooldown {
		user.SendText(messaging.CategorySystem, "You need a moment to recover before attempting another special move.")
		return true, nil
	}

	if res.NoTarget {
		user.SendText(messaging.CategorySystem, "You have no target!")
		return true, nil
	}

	if !res.Executed {
		return true, nil
	}

	target := res.Target
	result := res.MoveResult
	hasTail := res.Variant == actions.TripTailsweep

	targetName := target.Name
	targetPlayerId := target.UserId

	// Resolve the target user record for direct messaging (player targets).
	var targetChar *users.UserRecord
	if target.UserId > 0 {
		targetChar = users.GetByUserId(target.UserId)
	}

	// Send messages
	if result.Hit {
		if hasTail {
			if result.KnockedDown {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`Your <ansi fg="yellow-bold">tailsweep</ansi> sends <ansi fg="mobname">%s</ansi> crashing to the ground! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				if targetChar != nil {
					targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> hammers you with their tail, sending you crashing to the ground! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				}
				room.SendTextVisual(messaging.CategoryTrip,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> tailsweeps <ansi fg="mobname">%s</ansi>, sending them crashing to the ground!`, user.Character.Name, targetName),
					user.UserId, targetPlayerId,
				)
			} else {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`Your <ansi fg="yellow-bold">tailsweep</ansi> strikes <ansi fg="mobname">%s</ansi>, but they keep their footing! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				if targetChar != nil {
					targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> sweeps at you with their tail, but you manage to stay upright! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				}
				room.SendTextVisual(messaging.CategoryTrip,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> tailsweeps <ansi fg="mobname">%s</ansi>, but they keep their footing!`, user.Character.Name, targetName),
					user.UserId, targetPlayerId,
				)
			}
		} else {
			if result.KnockedDown {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`Your <ansi fg="yellow-bold">trip</ansi> sends <ansi fg="mobname">%s</ansi> crashing to the ground! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				if targetChar != nil {
					targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> sweeps your legs, sending you crashing to the ground! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				}
				room.SendTextVisual(messaging.CategoryTrip,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> trips <ansi fg="mobname">%s</ansi>, sending them crashing to the ground!`, user.Character.Name, targetName),
					user.UserId, targetPlayerId,
				)
			} else {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`Your <ansi fg="yellow-bold">trip</ansi> strikes <ansi fg="mobname">%s</ansi>, but they stay on their feet! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				if targetChar != nil {
					targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip you, but you keep your footing! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				}
				room.SendTextVisual(messaging.CategoryTrip,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip <ansi fg="mobname">%s</ansi>, but they keep their footing!`, user.Character.Name, targetName),
					user.UserId, targetPlayerId,
				)
			}
		}
	} else {
		if hasTail {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`Your <ansi fg="yellow-bold">tailsweep</ansi> misses <ansi fg="mobname">%s</ansi>!`, targetName))
			if targetChar != nil {
				targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> swings their tail at you, but you avoid it!`, user.Character.Name))
			}
			room.SendTextVisual(messaging.CategoryTrip,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts a tailsweep on <ansi fg="mobname">%s</ansi>, but misses!`, user.Character.Name, targetName),
				user.UserId, targetPlayerId,
			)
		} else {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`Your <ansi fg="yellow-bold">trip</ansi> attempt misses <ansi fg="mobname">%s</ansi>!`, targetName))
			if targetChar != nil {
				targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip you, but you avoid it!`, user.Character.Name))
			}
			room.SendTextVisual(messaging.CategoryTrip,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip <ansi fg="mobname">%s</ansi>, but misses!`, user.Character.Name, targetName),
				user.UserId, targetPlayerId,
			)
		}
	}

	// Quest engine: command notification
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "trip",
	}, bridge, bridge)

	return true, nil
}
