package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Pounce(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if actions.AcquireMeleeTarget(user, room, rest, actions.MeleeTargetOpts{
		Verb:          "pounce",
		PromptMsg:     "Pounce on whom?",
		SelfTargetMsg: "You can't pounce on yourself.",
		CharmedMsg:    "You can't pounce on a companion.",
	}) {
		return true, nil
	}

	// Delegate core resolution to the shared action.
	res := actions.ExecutePounce(&actions.UserActor{User: user, Room: room})

	if res.Grappling {
		user.SendText(messaging.CategorySystem, "You can't pounce from a clinch.")
		return true, nil
	}
	if res.NotPredator {
		user.SendText(messaging.CategorySystem, "You aren't built to pounce.")
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

	targetName := res.Target.Name

	// Resolve player target for direct messaging.
	var targetChar *users.UserRecord
	if res.Target.UserId > 0 {
		targetChar = users.GetByUserId(res.Target.UserId)
	}

	dmgDesc := combat.GetDamageDescription(res.MoveResult.Damage, res.MoveResult.TargetMaxHP)

	if res.MoveResult.Hit {
		if res.MoveResult.KnockedDown {
			// Hit + knockdown: predator bears prey to the ground.
			pounceMsgs := []string{
				`You leap at <ansi fg="mobname">%s</ansi> and slam them to the ground! (<ansi fg="damage">%s</ansi>)`,
				`You spring forward and bear <ansi fg="mobname">%s</ansi> down under your weight! (<ansi fg="damage">%s</ansi>)`,
				`Your pounce drives <ansi fg="mobname">%s</ansi> backward off their feet! (<ansi fg="damage">%s</ansi>)`,
				`You hurl yourself at <ansi fg="mobname">%s</ansi>, crashing into them and sending them sprawling! (<ansi fg="damage">%s</ansi>)`,
			}
			pounceTargetMsgs := []string{
				`<ansi fg="username">%s</ansi> leaps at you and slams you to the ground! (<ansi fg="damage">%s</ansi>)`,
				`<ansi fg="username">%s</ansi> springs forward and bears you down! (<ansi fg="damage">%s</ansi>)`,
				`<ansi fg="username">%s</ansi>'s pounce drives you backward off your feet! (<ansi fg="damage">%s</ansi>)`,
			}
			pounceRoomMsgs := []string{
				`<ansi fg="username">%s</ansi> leaps at <ansi fg="mobname">%s</ansi> and slams them to the ground!`,
				`<ansi fg="username">%s</ansi> springs forward and bears <ansi fg="mobname">%s</ansi> down!`,
				`<ansi fg="username">%s</ansi>'s pounce drives <ansi fg="mobname">%s</ansi> backward off their feet!`,
			}
			user.SendText(messaging.CategorySystem, fmt.Sprintf(pounceMsgs[util.Rand(len(pounceMsgs))], targetName, dmgDesc))
			if targetChar != nil {
				targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(pounceTargetMsgs[util.Rand(len(pounceTargetMsgs))], user.Character.Name, dmgDesc))
			}
			room.SendTextVisual(messaging.CategoryHitNaturalSharp, fmt.Sprintf(pounceRoomMsgs[util.Rand(len(pounceRoomMsgs))], user.Character.Name, targetName), user.UserId, res.Target.UserId)
		} else {
			// Hit but no knockdown: landed but target kept their feet.
			pounceMsgs := []string{
				`You spring at <ansi fg="mobname">%s</ansi>, raking into them before they push you back! (<ansi fg="damage">%s</ansi>)`,
				`Your pounce connects with <ansi fg="mobname">%s</ansi>, driving hard into their body! (<ansi fg="damage">%s</ansi>)`,
				`You lunge forward and crash into <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
				`You leap and bear into <ansi fg="mobname">%s</ansi>, drawing a grunt of pain! (<ansi fg="damage">%s</ansi>)`,
			}
			pounceTargetMsgs := []string{
				`<ansi fg="username">%s</ansi> springs at you and crashes into your body! (<ansi fg="damage">%s</ansi>)`,
				`<ansi fg="username">%s</ansi>'s pounce drives hard into you! (<ansi fg="damage">%s</ansi>)`,
				`<ansi fg="username">%s</ansi> lunges and slams into you! (<ansi fg="damage">%s</ansi>)`,
			}
			pounceRoomMsgs := []string{
				`<ansi fg="username">%s</ansi> springs at <ansi fg="mobname">%s</ansi> and crashes into them!`,
				`<ansi fg="username">%s</ansi>'s pounce drives hard into <ansi fg="mobname">%s</ansi>!`,
			}
			user.SendText(messaging.CategorySystem, fmt.Sprintf(pounceMsgs[util.Rand(len(pounceMsgs))], targetName, dmgDesc))
			if targetChar != nil {
				targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(pounceTargetMsgs[util.Rand(len(pounceTargetMsgs))], user.Character.Name, dmgDesc))
			}
			room.SendTextVisual(messaging.CategoryHitNaturalSharp, fmt.Sprintf(pounceRoomMsgs[util.Rand(len(pounceRoomMsgs))], user.Character.Name, targetName), user.UserId, res.Target.UserId)
		}
	} else {
		missMsgs := []string{
			`Your leap at <ansi fg="mobname">%s</ansi> misses as they sidestep!`,
			`You spring at <ansi fg="mobname">%s</ansi> but they dodge your pounce!`,
			`Your pounce carries you past <ansi fg="mobname">%s</ansi> — they step aside!`,
		}
		missTargetMsgs := []string{
			`<ansi fg="username">%s</ansi> leaps at you but you sidestep their pounce!`,
			`<ansi fg="username">%s</ansi> springs at you, but you dodge!`,
		}
		missRoomMsgs := []string{
			`<ansi fg="username">%s</ansi> leaps at <ansi fg="mobname">%s</ansi>, but misses!`,
		}

		user.SendText(messaging.CategorySystem, fmt.Sprintf(missMsgs[util.Rand(len(missMsgs))], targetName))
		if targetChar != nil {
			targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(missTargetMsgs[util.Rand(len(missTargetMsgs))], user.Character.Name))
		}
		room.SendTextVisual(messaging.CategoryHitNaturalSharp, fmt.Sprintf(missRoomMsgs[util.Rand(len(missRoomMsgs))], user.Character.Name, targetName), user.UserId, res.Target.UserId)
	}

	return true, nil
}
