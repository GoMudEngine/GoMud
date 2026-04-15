package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Trip(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Must be in combat or specify a target to use trip
	if user.Character.Aggro == nil {
		if rest == "" {
			user.SendText("Trip whom?")
			return true, nil
		}
		targetPId, targetMId := room.FindByName(rest)
		if targetPId == user.UserId {
			user.SendText("You can't trip yourself.")
			return true, nil
		}
		if targetPId == 0 && targetMId == 0 {
			user.SendText("You don't see them here.")
			return true, nil
		}
		if targetMId > 0 {
			if m := mobs.GetInstance(targetMId); m != nil && m.IsNonCombatant() {
				user.SendText(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, m.Character.Name))
				return true, nil
			}
			user.Character.SetAggro(0, targetMId, characters.DefaultAttack)
		} else {
			if p := users.GetByUserId(targetPId); p != nil {
				if pvpErr := room.CanPvp(user, p); pvpErr != nil {
					user.SendText(pvpErr.Error())
					return true, nil
				}
			}
			user.Character.SetAggro(targetPId, 0, characters.DefaultAttack)
		}
	}

	res := actions.ExecuteTrip(&actions.UserActor{User: user, Room: room})

	if res.OnCooldown {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	if res.NoTarget {
		user.SendText("You have no target!")
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
				user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">tailsweep</ansi> sends <ansi fg="mobname">%s</ansi> crashing to the ground! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				if targetChar != nil {
					targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> hammers you with their tail, sending you crashing to the ground! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				}
				room.SendTextVisual(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> tailsweeps <ansi fg="mobname">%s</ansi>, sending them crashing to the ground!`, user.Character.Name, targetName),
					user.UserId, targetPlayerId,
				)
			} else {
				user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">tailsweep</ansi> strikes <ansi fg="mobname">%s</ansi>, but they keep their footing! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				if targetChar != nil {
					targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> sweeps at you with their tail, but you manage to stay upright! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				}
				room.SendTextVisual(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> tailsweeps <ansi fg="mobname">%s</ansi>, but they keep their footing!`, user.Character.Name, targetName),
					user.UserId, targetPlayerId,
				)
			}
		} else {
			if result.KnockedDown {
				user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">trip</ansi> sends <ansi fg="mobname">%s</ansi> crashing to the ground! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				if targetChar != nil {
					targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> sweeps your legs, sending you crashing to the ground! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				}
				room.SendTextVisual(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> trips <ansi fg="mobname">%s</ansi>, sending them crashing to the ground!`, user.Character.Name, targetName),
					user.UserId, targetPlayerId,
				)
			} else {
				user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">trip</ansi> strikes <ansi fg="mobname">%s</ansi>, but they stay on their feet! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				if targetChar != nil {
					targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip you, but you keep your footing! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
				}
				room.SendTextVisual(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip <ansi fg="mobname">%s</ansi>, but they keep their footing!`, user.Character.Name, targetName),
					user.UserId, targetPlayerId,
				)
			}
		}
	} else {
		if hasTail {
			user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">tailsweep</ansi> misses <ansi fg="mobname">%s</ansi>!`, targetName))
			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> swings their tail at you, but you avoid it!`, user.Character.Name))
			}
			room.SendTextVisual(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts a tailsweep on <ansi fg="mobname">%s</ansi>, but misses!`, user.Character.Name, targetName),
				user.UserId, targetPlayerId,
			)
		} else {
			user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">trip</ansi> attempt misses <ansi fg="mobname">%s</ansi>!`, targetName))
			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip you, but you avoid it!`, user.Character.Name))
			}
			room.SendTextVisual(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip <ansi fg="mobname">%s</ansi>, but misses!`, user.Character.Name, targetName),
				user.UserId, targetPlayerId,
			)
		}
	}

	return true, nil
}
