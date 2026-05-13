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
	if user.Character.IsCrafting() {
		user.SendText(`<ansi fg="red">You can't bash while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}

	// Must be in combat or specify a target to use bash.
	if !user.Character.IsInCombat() {
		if rest == "" {
			user.SendText("Bash whom?")
			return true, nil
		}

		target, err := actions.ResolveTargetActor(room, rest, actions.ResolveTargetOptions{
			ExcludeUserId: user.UserId,
		})
		if err != nil {
			// Self-exclusion collapses to NotFound; pre-check for self-targeting message.
			if pId, _ := room.FindByName(rest); pId == user.UserId {
				user.SendText("You can't bash yourself.")
				return true, nil
			}
			user.SendText("You don't see them here.")
			return true, nil
		}

		if !target.IsPlayer() {
			mob := target.(*actions.MobActor).Mob
			if mob.IsNonCombatant() || mob.PlayerAttackImmune {
				user.SendText(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, mob.Character.Name))
				return true, nil
			}
			user.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
		} else {
			p := target.(*actions.UserActor).User
			if pvpErr := room.CanPvp(user, p); pvpErr != nil {
				user.SendText(pvpErr.Error())
				return true, nil
			}
			user.Character.SetAggro(p.UserId, 0, characters.DefaultAttack)
		}
	}

	// Delegate core bash logic to the shared action.
	bashResult := actions.ExecuteBash(&actions.UserActor{User: user, Room: room})

	if bashResult.Crafting {
		// Safety net — should have been caught by the pre-reject above.
		user.SendText(`<ansi fg="red">You can't bash while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}

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
			room.SendTextVisual(
				fmt.Sprintf(`<ansi fg="username">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> knocks <ansi fg="mobname">%s</ansi> to the ground!`, user.Character.Name, target.Name),
				user.UserId, target.UserId,
			)
		} else {
			user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">shield bash</ansi> strikes <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`, target.Name, dmgDesc))
			if targetUser != nil {
				targetUser.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> strikes you! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, dmgDesc))
			}
			room.SendTextVisual(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> bashes <ansi fg="mobname">%s</ansi> with their shield!`, user.Character.Name, target.Name),
				user.UserId, target.UserId,
			)
		}
	} else {
		user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">shield bash</ansi> misses <ansi fg="mobname">%s</ansi>!`, target.Name))
		if targetUser != nil {
			targetUser.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to bash you with their shield, but misses!`, user.Character.Name))
		}
		room.SendTextVisual(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to bash <ansi fg="mobname">%s</ansi>, but misses!`, user.Character.Name, target.Name),
			user.UserId, target.UserId,
		)
	}

	return true, nil
}
