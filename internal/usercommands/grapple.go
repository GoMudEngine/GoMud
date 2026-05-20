package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Grapple(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if user.Character.IsActing() {
		user.SendTextLegacy(`<ansi fg="red">You can't grapple while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}

	// Must be in combat or specify a target to use grapple
	if !user.Character.IsInCombat() {
		if rest == "" {
			user.SendTextLegacy("Grapple whom?")
			return true, nil
		}

		target, err := actions.ResolveTargetActor(room, rest, actions.ResolveTargetOptions{
			ExcludeUserId: user.UserId,
		})
		if err != nil {
			// Self-exclusion collapses to NotFound; pre-check for self-targeting message.
			if pId, _ := room.FindByName(rest); pId == user.UserId {
				user.SendTextLegacy("You can't grapple yourself.")
				return true, nil
			}
			user.SendTextLegacy("You don't see them here.")
			return true, nil
		}

		if !target.IsPlayer() {
			mob := target.(*actions.MobActor).Mob
			if mob.IsNonCombatant() || mob.PlayerAttackImmune {
				user.SendTextLegacy(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, mob.Character.Name))
				return true, nil
			}
			user.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
		} else {
			p := target.(*actions.UserActor).User
			if pvpErr := room.CanPvp(user, p); pvpErr != nil {
				user.SendTextLegacy(pvpErr.Error())
				return true, nil
			}
			user.Character.SetAggro(p.UserId, 0, characters.DefaultAttack)
		}
	}

	res := actions.ExecuteGrapple(&actions.UserActor{User: user, Room: room})

	if res.Crafting {
		// Safety net — should have been caught by the pre-reject above.
		user.SendTextLegacy(`<ansi fg="red">You can't grapple while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}

	if res.OnCooldown {
		user.SendTextLegacy("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	if res.NoTarget {
		user.SendTextLegacy("You have no target!")
		return true, nil
	}

	if res.GrappleImmune {
		user.SendTextLegacy(fmt.Sprintf("You reach for %s but your hands pass right through!", res.Target.Name))
		return true, nil
	}

	if !res.Executed {
		return true, nil
	}

	target := res.Target
	result := res.MoveResult
	targetName := target.Name
	targetPlayerId := target.UserId

	// Resolve the target user record for direct messaging (player targets).
	var targetChar *users.UserRecord
	if target.UserId > 0 {
		targetChar = users.GetByUserId(target.UserId)
	}

	// Send messages based on result
	if result.Success {
		user.SendTextLegacy(fmt.Sprintf(`You <ansi fg="yellow-bold">grapple</ansi> <ansi fg="mobname">%s</ansi>, transitioning to <ansi fg="cyan">%s</ansi> position!`, targetName, result.PositionDesc))
		if targetChar != nil {
			targetChar.SendTextLegacy(fmt.Sprintf(`<ansi fg="username">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> you, transitioning to <ansi fg="cyan">%s</ansi> position!`, user.Character.Name, result.PositionDesc))
		}
		room.SendTextVisualLegacy(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> <ansi fg="mobname">%s</ansi> into <ansi fg="cyan">%s</ansi> position!`, user.Character.Name, targetName, result.PositionDesc),
			user.UserId, targetPlayerId,
		)

		// Flavor text for prone targets
		if result.PositionPenalty < 0 {
			user.SendTextLegacy(fmt.Sprintf(`<ansi fg="yellow">%s was already prone - they had little chance to resist!</ansi>`, targetName))
		}

		// Disarm messaging
		if result.DisarmResult != nil {
			user.SendTextLegacy(result.DisarmResult.Message)
			if targetChar != nil {
				targetChar.SendTextLegacy(result.DisarmResult.TargetMsg)
			}
			room.SendTextVisualLegacy(result.DisarmResult.RoomMessage, user.UserId, targetPlayerId)
		}
	} else {
		user.SendTextLegacy(fmt.Sprintf(`Your <ansi fg="yellow-bold">grapple</ansi> attempt against <ansi fg="mobname">%s</ansi> fails!`, targetName))
		if targetChar != nil {
			targetChar.SendTextLegacy(fmt.Sprintf(`<ansi fg="username">%s</ansi> tries to grapple you, but you slip away!`, user.Character.Name))
		}
		room.SendTextVisualLegacy(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> tries to grapple <ansi fg="mobname">%s</ansi>, but fails!`, user.Character.Name, targetName),
			user.UserId, targetPlayerId,
		)

		// Defense penalty messaging
		if result.DefensePenalty {
			user.SendTextLegacy(`<ansi fg="red">Your failed attempt leaves you exposed!</ansi>`)
		}

		// Critical failure messaging
		if result.CritFailure != nil {
			user.SendTextLegacy(result.CritFailure.Message)
			if targetChar != nil {
				targetChar.SendTextLegacy(result.CritFailure.TargetMessage)
			}
			room.SendTextVisualLegacy(result.CritFailure.RoomMessage, user.UserId, targetPlayerId)
		}
	}

	return true, nil
}
