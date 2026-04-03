package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
Skullduggery Skill
Level 3 - Shadow: follow a target between rooms while remaining hidden.
When the target moves, the shadower automatically moves with them.
A target-specific detection roll alerts the target if they sense pursuit.
Shadow ends if the shadower loses their hidden buff or manually stops.
*/
func Shadow(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Skullduggery)

	// Requires skullduggery rank 3
	if skillLevel < 1 {
		return false, nil
	}
	if skillLevel < 3 {
		user.SendText("You aren't advanced enough at skullduggery for that.")
		return true, nil
	}

	rest = strings.TrimSpace(rest)

	// "shadow stop" cancels an active shadow
	if strings.ToLower(rest) == "stop" {
		if user.Character.GetMiscData("shadow-target-user") != nil ||
			user.Character.GetMiscData("shadow-target-mob") != nil {
			endShadow(user, "You stop shadowing your target.")
		} else {
			user.SendText("You aren't shadowing anyone.")
		}
		return true, nil
	}

	if rest == "" {
		user.SendText("Shadow whom?")
		return true, nil
	}

	// Must be hidden to shadow
	if !user.Character.HasBuffFlag(buffs.Hidden) {
		user.SendText(
			"You must be hidden to shadow someone. " +
				"Try <ansi fg=\"command\">sneak</ansi> first.")
		return true, nil
	}

	if user.Character.Aggro != nil {
		user.SendText("You can't do that while in combat!")
		return true, nil
	}

	// Check cooldown — prevents re-shadowing immediately after a shadow ends
	cooldownKey := skills.Skullduggery.String(`shadow`)
	if user.Character.GetCooldown(cooldownKey) > 0 {
		user.SendText(fmt.Sprintf(
			"You need to wait %d more rounds before shadowing again.",
			user.Character.GetCooldown(cooldownKey)))
		return true, nil
	}

	// Resolve target in the current room
	targetPlayerId, targetMobId := room.FindByName(strings.ToLower(rest))

	if targetPlayerId == 0 && targetMobId == 0 {
		user.SendText("Shadow whom?")
		return true, nil
	}

	// Store the shadow target
	if targetPlayerId > 0 {
		targetUser := users.GetByUserId(targetPlayerId)
		if targetUser == nil {
			user.SendText("They seem to have vanished.")
			return true, nil
		}
		// Can't shadow yourself
		if targetPlayerId == user.UserId {
			user.SendText("You can't shadow yourself.")
			return true, nil
		}
		user.Character.SetMiscData("shadow-target-user", targetPlayerId)
		user.Character.SetMiscData("shadow-target-mob", nil)
		user.SendText(fmt.Sprintf(
			`You begin shadowing <ansi fg="username">%s</ansi>, `+
				`watching their every move.`,
			targetUser.Character.Name))
	} else {
		mob := mobs.GetInstance(targetMobId)
		if mob == nil {
			user.SendText("They seem to have vanished.")
			return true, nil
		}
		user.Character.SetMiscData("shadow-target-user", nil)
		user.Character.SetMiscData("shadow-target-mob", targetMobId)
		user.SendText(fmt.Sprintf(
			`You begin shadowing <ansi fg="mobname">%s</ansi>, `+
				`moving silently in their wake.`,
			mob.Character.Name))
	}

	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.Skullduggery,
		Details: `shadow`,
	})

	user.Character.CheckSkillProgression(
		string(skills.Skullduggery), user.UserId, 1.0)

	return true, nil
}

// endShadow clears the shadow target state, starts the cooldown, and
// optionally sends a reason message to the shadower.
func endShadow(user *users.UserRecord, reason string) {
	user.Character.SetMiscData("shadow-target-user", nil)
	user.Character.SetMiscData("shadow-target-mob", nil)

	cfg := configs.GetBalanceConfig()
	cooldownKey := skills.Skullduggery.String(`shadow`)
	user.Character.TryCooldown(cooldownKey,
		fmt.Sprintf(`%d rounds`, cfg.ShadowCooldown))

	if reason != "" {
		user.SendText(reason)
	}
}

// getShadowTargetUserId returns the player user ID being shadowed, or 0.
func getShadowTargetUserId(user *users.UserRecord) int {
	raw := user.Character.GetMiscData("shadow-target-user")
	if raw == nil {
		return 0
	}
	if uid, ok := raw.(int); ok {
		return uid
	}
	return 0
}

// shadowIsTargetingUser returns true when the given shadower is tracking
// the mover identified by moverId. Checks the "shadow-target-user" misc slot.
func shadowIsTargetingUser(shadower *users.UserRecord, moverId int) bool {
	raw := shadower.Character.GetMiscData("shadow-target-user")
	if raw == nil {
		return false
	}
	uid, ok := raw.(int)
	return ok && uid == moverId
}

// shadowDetectionRoll performs a target-specific detection check.
// Returns true if the target sensed the follower (shadower detected).
// Uses Perception+Search vs Dex+Skullduggery (target is the attacker).
func shadowDetectionRoll(shadower *users.UserRecord, target *users.UserRecord) bool {
	sneakScore := calcSneakScore(shadower.Character)
	targetScore := actions.CalcSearchScore(target.Character)

	// OpposedRollStat(atk, def) returns true when atk (first arg) wins.
	// Target detects when targetScore beats sneakScore.
	detected, _, _, _ := dice.OpposedRollStat(targetScore, sneakScore)
	return detected
}
