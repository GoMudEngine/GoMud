package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Bash(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Must be in combat or specify a target to use bash.
	if user.Character.Aggro == nil {
		if rest == "" {
			user.SendText("Bash whom?")
			return true, nil
		}
		targetPId, targetMId := room.FindByName(rest)
		if targetPId == user.UserId {
			user.SendText("You can't bash yourself.")
			return true, nil
		}
		if targetPId == 0 && targetMId == 0 {
			user.SendText("You don't see them here.")
			return true, nil
		}
		if targetMId > 0 {
			user.Character.SetAggro(0, targetMId, characters.DefaultAttack)
		} else {
			user.Character.SetAggro(targetPId, 0, characters.DefaultAttack)
		}
	}

	// Delegate core bash logic to the shared action.
	bashResult := actions.ExecuteBash(&actions.UserActor{User: user, Room: room})

	switch {
	case bashResult.NoShield:
		user.SendText("You need a shield equipped to perform a shield bash!")
		return true, nil
	case bashResult.OnCooldown:
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	case bashResult.NoTarget:
		user.SendText("Your target is gone!")
		return true, nil
	}

	// Format and send messages.
	target := bashResult.Target
	result := bashResult.MoveResult
	dmgDesc := combat.GetDamageDescription(result.Damage, result.TargetMaxHP)

	// Look up the target player record for personal messaging (nil if mob target).
	var targetUser *users.UserRecord
	if target.UserId > 0 {
		targetUser = users.GetByUserId(target.UserId)
	}

	if result.Hit {
		if result.KnockedDown {
			user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">shield bash</ansi> knocks <ansi fg="mobname">%s</ansi> to the ground! (<ansi fg="damage">%s</ansi>)`, target.Name, dmgDesc))
			if targetUser != nil {
				targetUser.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, dmgDesc))
			}
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> knocks <ansi fg="mobname">%s</ansi> to the ground!`, user.Character.Name, target.Name),
				user.UserId, target.UserId,
			)
		} else {
			user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">shield bash</ansi> strikes <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`, target.Name, dmgDesc))
			if targetUser != nil {
				targetUser.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> strikes you! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, dmgDesc))
			}
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> bashes <ansi fg="mobname">%s</ansi> with their shield!`, user.Character.Name, target.Name),
				user.UserId, target.UserId,
			)
		}
	} else {
		user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">shield bash</ansi> misses <ansi fg="mobname">%s</ansi>!`, target.Name))
		if targetUser != nil {
			targetUser.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to bash you with their shield, but misses!`, user.Character.Name))
		}
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to bash <ansi fg="mobname">%s</ansi>, but misses!`, user.Character.Name, target.Name),
			user.UserId, target.UserId,
		)
	}

	return true, nil
}
