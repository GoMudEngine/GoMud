package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Target(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Must be in combat to switch targets
	if user.Character.Aggro == nil {
		user.SendText("You're not in combat. Use <ansi fg=\"command\">attack</ansi> to initiate combat.")
		return true, nil
	}

	// Can't switch during spell casting or other special aggro types
	if user.Character.Aggro.Type != characters.DefaultAttack && user.Character.Aggro.Type != characters.Shooting {
		user.SendText("You can't switch targets right now.")
		return true, nil
	}

	if rest == "" {
		user.SendText("Switch to which target?")
		return true, nil
	}

	// Find the new target
	target, err := actions.ResolveTargetActor(room, rest, actions.ResolveTargetOptions{
		ExcludeUserId: user.UserId,
	})
	if err != nil {
		// Distinguish self-targeting vs not-found via the original wording.
		if pId, _ := room.FindByName(rest); pId == user.UserId {
			user.SendText("You can't target yourself!")
			return true, nil
		}
		user.SendText(fmt.Sprintf("You don't see '%s' here.", rest))
		return true, nil
	}

	newTargetPlayerId := target.GetUserId()
	newTargetMobInstanceId := target.GetMobInstanceId()

	// Check if already targeting this entity
	currentTargetUserId := user.Character.Aggro.UserId
	currentTargetMobId := user.Character.Aggro.MobInstanceId

	if newTargetPlayerId > 0 && newTargetPlayerId == currentTargetUserId {
		user.SendText("You're already targeting them!")
		return true, nil
	}

	if newTargetMobInstanceId > 0 && newTargetMobInstanceId == currentTargetMobId {
		user.SendText("You're already targeting them!")
		return true, nil
	}

	// Validate the new target
	if newTargetMobInstanceId > 0 {
		m := mobs.GetInstance(newTargetMobInstanceId)
		if m == nil {
			user.SendText(fmt.Sprintf("You don't see '%s' here.", rest))
			return true, nil
		}

		// Can't target any companion
		if m.Character.IsCharmed() {
			user.SendText(fmt.Sprintf("<ansi fg=\"mobname\">%s</ansi> is someone's companion!", m.Character.Name))
			return true, nil
		}
	}

	if newTargetPlayerId > 0 {
		p := users.GetByUserId(newTargetPlayerId)
		if p == nil {
			user.SendText(fmt.Sprintf("You don't see '%s' here.", rest))
			return true, nil
		}

		// Check PvP restrictions
		if pvpErr := room.CanPvp(user, p); pvpErr != nil {
			user.SendText(pvpErr.Error())
			return true, nil
		}

		// Can't target party members
		if partyInfo := parties.Get(user.UserId); partyInfo != nil {
			if partyInfo.IsMember(newTargetPlayerId) {
				user.SendText(fmt.Sprintf("<ansi fg=\"username\">%s</ansi> is in your party!", p.Character.Name))
				return true, nil
			}
		}
	}

	// If current target is dead or gone, switch freely (no roll, no round cost)
	currentTargetGone := false
	if currentTargetMobId > 0 {
		curMob := mobs.GetInstance(currentTargetMobId)
		if curMob == nil || curMob.Character.Health < 1 || curMob.Character.RoomId != user.Character.RoomId {
			currentTargetGone = true
		}
	} else if currentTargetUserId > 0 {
		curUser := users.GetByUserId(currentTargetUserId)
		if curUser == nil || curUser.Character.Health < 1 || curUser.Character.RoomId != user.Character.RoomId {
			currentTargetGone = true
		}
	} else {
		// No actual target set (MobInstanceId=0, UserId=0) — switch freely
		currentTargetGone = true
	}

	if currentTargetGone {
		aggroType := user.Character.Aggro.Type
		user.Character.SetAggro(newTargetPlayerId, newTargetMobInstanceId, aggroType)

		if newTargetMobInstanceId > 0 {
			if m := mobs.GetInstance(newTargetMobInstanceId); m != nil {
				user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"mobname\">%s</ansi>!", m.Character.Name))
			}
		} else if newTargetPlayerId > 0 {
			if p := users.GetByUserId(newTargetPlayerId); p != nil {
				user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"username\">%s</ansi>!", p.Character.Name))
			}
		}
		return true, nil
	}

	// Perform skill check to see if target switch succeeds
	switchChance := combat.ChanceToSwitchTarget(user.Character)
	roll := util.Rand(100)

	util.LogRoll("Target Switch", roll, switchChance)

	if roll < switchChance {
		// SUCCESS: Switch targets
		// Store the aggro type to preserve shooting/melee
		aggroType := user.Character.Aggro.Type

		// Switch to new target with 1 round wait (cost of repositioning)
		user.Character.SetAggro(newTargetPlayerId, newTargetMobInstanceId, aggroType, 1)

		if newTargetMobInstanceId > 0 {
			m := mobs.GetInstance(newTargetMobInstanceId)
			user.SendText(fmt.Sprintf("You shift your focus to <ansi fg=\"mobname\">%s</ansi>!", m.Character.Name))
			room.SendTextVisual(
				fmt.Sprintf("<ansi fg=\"username\">%s</ansi> shifts focus to <ansi fg=\"mobname\">%s</ansi>!", user.Character.Name, m.Character.Name),
				user.UserId,
			)
		} else if newTargetPlayerId > 0 {
			p := users.GetByUserId(newTargetPlayerId)
			user.SendText(fmt.Sprintf("You shift your focus to <ansi fg=\"username\">%s</ansi>!", p.Character.Name))
			p.SendText(fmt.Sprintf("<ansi fg=\"username\">%s</ansi> shifts focus to you!", user.Character.Name))
			room.SendTextVisual(
				fmt.Sprintf("<ansi fg=\"username\">%s</ansi> shifts focus to <ansi fg=\"username\">%s</ansi>!", user.Character.Name, p.Character.Name),
				user.UserId, newTargetPlayerId,
			)
		}

	} else {
		// FAILURE: Keep attacking current target this round
		user.SendText("You try to reposition but can't break away from your current opponent!")

		// Still costs a round (set RoundsWaiting to 1)
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
