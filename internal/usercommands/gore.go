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

func Gore(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if actions.AcquireMeleeTarget(user, room, rest, actions.MeleeTargetOpts{
		Verb: "gore",
	}) {
		return true, nil
	}

	// Delegate core resolution to the shared action.
	res := actions.ExecuteGore(&actions.UserActor{User: user, Room: room})

	if res.NotHorned {
		user.SendText(messaging.CategorySystem, "You have no horns to gore with.")
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
			// Hit + knockdown: horned charge tosses target off their feet.
			goreMsgs := []string{
				`You lower your head and charge <ansi fg="mobname">%s</ansi>, driving your horns in and sending them flying! (<ansi fg="damage">%s</ansi>)`,
				`Your charge slams into <ansi fg="mobname">%s</ansi> with bone-jarring force, hurling them to the ground! (<ansi fg="damage">%s</ansi>)`,
				`You thunder forward and gore <ansi fg="mobname">%s</ansi>, tossing them aside like a ragdoll! (<ansi fg="damage">%s</ansi>)`,
				`Your horns catch <ansi fg="mobname">%s</ansi> and you heave them backward off their feet! (<ansi fg="damage">%s</ansi>)`,
			}
			goreTargetMsgs := []string{
				`<ansi fg="username">%s</ansi> lowers their head and charges you — the impact sends you sprawling! (<ansi fg="damage">%s</ansi>)`,
				`<ansi fg="username">%s</ansi> drives their horns into you with shattering force and hurls you to the ground! (<ansi fg="damage">%s</ansi>)`,
				`<ansi fg="username">%s</ansi>'s charge catches you and tosses you off your feet! (<ansi fg="damage">%s</ansi>)`,
			}
			goreRoomMsgs := []string{
				`<ansi fg="username">%s</ansi> charges <ansi fg="mobname">%s</ansi> and hurls them to the ground!`,
				`<ansi fg="username">%s</ansi> drives their horns into <ansi fg="mobname">%s</ansi> and sends them flying!`,
				`<ansi fg="username">%s</ansi>'s horned charge tosses <ansi fg="mobname">%s</ansi> off their feet!`,
			}
			user.SendText(messaging.CategorySystem, fmt.Sprintf(goreMsgs[util.Rand(len(goreMsgs))], targetName, dmgDesc))
			if targetChar != nil {
				targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(goreTargetMsgs[util.Rand(len(goreTargetMsgs))], user.Character.Name, dmgDesc))
			}
			room.SendTextVisual(messaging.CategoryHitNaturalSharp, fmt.Sprintf(goreRoomMsgs[util.Rand(len(goreRoomMsgs))], user.Character.Name, targetName), user.UserId, res.Target.UserId)
		} else {
			// Hit but no knockdown: the charge connects but target stays up.
			goreMsgs := []string{
				`You ram your horns into <ansi fg="mobname">%s</ansi> with a powerful charge! (<ansi fg="damage">%s</ansi>)`,
				`Your horned charge drives into <ansi fg="mobname">%s</ansi>, rocking them back! (<ansi fg="damage">%s</ansi>)`,
				`You lower your head and slam your horns into <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
				`Your charge catches <ansi fg="mobname">%s</ansi> with your horns, drawing a pained grunt! (<ansi fg="damage">%s</ansi>)`,
			}
			goreTargetMsgs := []string{
				`<ansi fg="username">%s</ansi> charges and drives their horns into you! (<ansi fg="damage">%s</ansi>)`,
				`<ansi fg="username">%s</ansi>'s horned charge slams into you and rocks you back! (<ansi fg="damage">%s</ansi>)`,
				`<ansi fg="username">%s</ansi> lowers their head and rams their horns into you! (<ansi fg="damage">%s</ansi>)`,
			}
			goreRoomMsgs := []string{
				`<ansi fg="username">%s</ansi> charges <ansi fg="mobname">%s</ansi> and drives their horns in!`,
				`<ansi fg="username">%s</ansi>'s horned charge slams into <ansi fg="mobname">%s</ansi>!`,
			}
			user.SendText(messaging.CategorySystem, fmt.Sprintf(goreMsgs[util.Rand(len(goreMsgs))], targetName, dmgDesc))
			if targetChar != nil {
				targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(goreTargetMsgs[util.Rand(len(goreTargetMsgs))], user.Character.Name, dmgDesc))
			}
			room.SendTextVisual(messaging.CategoryHitNaturalSharp, fmt.Sprintf(goreRoomMsgs[util.Rand(len(goreRoomMsgs))], user.Character.Name, targetName), user.UserId, res.Target.UserId)
		}
	} else {
		missMsgs := []string{
			`Your charge at <ansi fg="mobname">%s</ansi> misses as they sidestep your horns!`,
			`You thunder toward <ansi fg="mobname">%s</ansi> but they dodge your gore!`,
			`Your horned charge carries you past <ansi fg="mobname">%s</ansi> — they step aside!`,
		}
		missTargetMsgs := []string{
			`<ansi fg="username">%s</ansi> charges you but you sidestep their horns!`,
			`<ansi fg="username">%s</ansi> thunders toward you, but you dodge the gore!`,
		}
		missRoomMsgs := []string{
			`<ansi fg="username">%s</ansi> charges <ansi fg="mobname">%s</ansi>, but misses!`,
		}

		user.SendText(messaging.CategorySystem, fmt.Sprintf(missMsgs[util.Rand(len(missMsgs))], targetName))
		if targetChar != nil {
			targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(missTargetMsgs[util.Rand(len(missTargetMsgs))], user.Character.Name))
		}
		room.SendTextVisual(messaging.CategoryHitNaturalSharp, fmt.Sprintf(missRoomMsgs[util.Rand(len(missRoomMsgs))], user.Character.Name, targetName), user.UserId, res.Target.UserId)
	}

	return true, nil
}
