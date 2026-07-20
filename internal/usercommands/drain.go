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

func Drain(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if actions.AcquireMeleeTarget(user, room, rest, actions.MeleeTargetOpts{
		Verb: "drain",
	}) {
		return true, nil
	}

	// Delegate core resolution to the shared action.
	res := actions.ExecuteDrain(&actions.UserActor{User: user, Room: room})

	if res.NotLifeDrainer {
		user.SendText(messaging.CategorySystem, "Your body cannot drain the life from another.")
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
		drainMsgs := []string{
			`Your <ansi fg="item">claws</ansi> plunge into <ansi fg="mobname">%s</ansi>, sapping their vitality as warmth floods into you! (<ansi fg="damage">%s</ansi>)`,
			`You latch onto <ansi fg="mobname">%s</ansi> and drink deep, leeching their life-force! (<ansi fg="damage">%s</ansi>)`,
			`Your touch drains <ansi fg="mobname">%s</ansi>, pulling vitality through your <ansi fg="item">claws</ansi>! (<ansi fg="damage">%s</ansi>)`,
			`You tear into <ansi fg="mobname">%s</ansi> and feed, growing stronger as they weaken! (<ansi fg="damage">%s</ansi>)`,
		}
		drainTargetMsgs := []string{
			`<ansi fg="username">%s</ansi> plunges their <ansi fg="item">claws</ansi> into you, sapping your vitality! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi> latches onto you and drinks deep, leeching your life-force! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi>''s touch drains you, pulling vitality through their grasp! (<ansi fg="damage">%s</ansi>)`,
			`<ansi fg="username">%s</ansi> tears into you and feeds, growing stronger as you weaken! (<ansi fg="damage">%s</ansi>)`,
		}
		drainRoomMsgs := []string{
			`<ansi fg="username">%s</ansi> plunges their <ansi fg="item">claws</ansi> into <ansi fg="mobname">%s</ansi>, leeching their vitality!`,
			`<ansi fg="username">%s</ansi> latches onto <ansi fg="mobname">%s</ansi> and drinks deep!`,
			`<ansi fg="username">%s</ansi> tears into <ansi fg="mobname">%s</ansi> and feeds hungrily!`,
		}

		user.SendText(messaging.CategorySystem, fmt.Sprintf(drainMsgs[util.Rand(len(drainMsgs))], targetName, dmgDesc))
		if targetChar != nil {
			targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(drainTargetMsgs[util.Rand(len(drainTargetMsgs))], user.Character.Name, dmgDesc))
		}
		room.SendTextVisual(messaging.CategoryHitNaturalSharp, fmt.Sprintf(drainRoomMsgs[util.Rand(len(drainRoomMsgs))], user.Character.Name, targetName), user.UserId, res.Target.UserId)

		// Describe the heal to the draining player — no raw numbers.
		if res.Healed > 0 {
			healDesc := combat.GetHealDescription(res.Healed, user.Character.HealthMax.Value)
			healMsgs := []string{
				`<ansi fg="green">Their stolen vitality flows into you — %s.</ansi>`,
				`<ansi fg="green">Stolen life-force surges through you — %s.</ansi>`,
				`<ansi fg="green">You feel %s as their warmth becomes yours.</ansi>`,
			}
			user.SendText(messaging.CategorySystem, fmt.Sprintf(healMsgs[util.Rand(len(healMsgs))], healDesc))
		}
	} else {
		missMsgs := []string{
			`Your drain misses <ansi fg="mobname">%s</ansi>!`,
			`You reach for <ansi fg="mobname">%s</ansi> but they slip away!`,
			`Your draining lunge glances off <ansi fg="mobname">%s</ansi> harmlessly!`,
		}
		missTargetMsgs := []string{
			`<ansi fg="username">%s</ansi> reaches for you with hungry <ansi fg="item">claws</ansi>, but misses!`,
			`<ansi fg="username">%s</ansi> lunges to drain you but you slip away!`,
		}
		missRoomMsgs := []string{
			`<ansi fg="username">%s</ansi> lunges to drain <ansi fg="mobname">%s</ansi>, but misses!`,
		}

		user.SendText(messaging.CategorySystem, fmt.Sprintf(missMsgs[util.Rand(len(missMsgs))], targetName))
		if targetChar != nil {
			targetChar.SendText(messaging.CategorySystem, fmt.Sprintf(missTargetMsgs[util.Rand(len(missTargetMsgs))], user.Character.Name))
		}
		room.SendTextVisual(messaging.CategoryHitNaturalSharp, fmt.Sprintf(missRoomMsgs[util.Rand(len(missRoomMsgs))], user.Character.Name, targetName), user.UserId, res.Target.UserId)
	}

	return true, nil
}
