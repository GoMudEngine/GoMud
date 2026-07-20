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

func Maul(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if actions.AcquireMeleeTarget(user, room, rest, actions.MeleeTargetOpts{
		Verb: "maul",
	}) {
		return true, nil
	}

	// Delegate core resolution to the shared action.
	res := actions.ExecuteMaul(&actions.UserActor{User: user, Room: room})

	if res.NotFanged {
		user.SendText(messaging.CategorySystem, "You have no fangs to maul with.")
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
		maulMsgs := []string{
			`Your fangs savage <ansi fg="mobname">%s</ansi>, tearing wounds that weep blood! (<ansi fg="damage">%s</ansi>)`,
			`You maul <ansi fg="mobname">%s</ansi>, worrying their flesh with savage fury! (<ansi fg="damage">%s</ansi>)`,
			`Your savage bite tears into <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
			`You drive your fangs into <ansi fg="mobname">%s</ansi> and worry the wound viciously! (<ansi fg="damage">%s</ansi>)`,
			`A brutal mauling drives your teeth deep into <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
		}
		maulTargetMsgs := []string{
			`<ansi fg="username">%s</ansi> savages you with their fangs, tearing bleeding wounds! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi> mauls you, worrying your flesh with savage fury! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi>'s savage bite tears into you! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi> drives their fangs into you and worries the wound! (<ansi fg="damage">%s</ansi>)`,
		}
		maulRoomMsgs := []string{
			`<ansi fg="username">%s</ansi> savages <ansi fg="mobname">%s</ansi> with vicious fangs!`,
			`<ansi fg="username">%s</ansi> mauls <ansi fg="mobname">%s</ansi> savagely!`,
			`<ansi fg="username">%s</ansi>'s fangs tear deep into <ansi fg="mobname">%s</ansi>!`,
		}

		user.SendText(messaging.CategorySystem, fmt.Sprintf(maulMsgs[util.Rand(len(maulMsgs))], targetName, dmgDesc))
		if targetChar != nil {
			targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(maulTargetMsgs[util.Rand(len(maulTargetMsgs))], user.Character.Name, dmgDesc))
		}
		room.SendTextVisual(messaging.CategoryHitNaturalSharp, fmt.Sprintf(maulRoomMsgs[util.Rand(len(maulRoomMsgs))], user.Character.Name, targetName), user.UserId, res.Target.UserId)
	} else {
		missMsgs := []string{
			`Your savage bite misses <ansi fg="mobname">%s</ansi>!`,
			`You snap at <ansi fg="mobname">%s</ansi> but they dodge your fangs!`,
			`Your mauling lunge glances off <ansi fg="mobname">%s</ansi> harmlessly!`,
		}
		missTargetMsgs := []string{
			`<ansi fg="username">%s</ansi> snaps their fangs at you, but misses!`,
			`<ansi fg="username">%s</ansi> lunges to maul you but you dodge!`,
		}
		missRoomMsgs := []string{
			`<ansi fg="username">%s</ansi> lunges to maul <ansi fg="mobname">%s</ansi>, but misses!`,
		}

		user.SendText(messaging.CategorySystem, fmt.Sprintf(missMsgs[util.Rand(len(missMsgs))], targetName))
		if targetChar != nil {
			targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(missTargetMsgs[util.Rand(len(missTargetMsgs))], user.Character.Name))
		}
		room.SendTextVisual(messaging.CategoryHitNaturalSharp, fmt.Sprintf(missRoomMsgs[util.Rand(len(missRoomMsgs))], user.Character.Name, targetName), user.UserId, res.Target.UserId)
	}

	return true, nil
}
