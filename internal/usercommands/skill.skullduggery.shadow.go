package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
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
		user.SendText(messaging.CategorySystem, "You aren't advanced enough at skullduggery for that.")
		return true, nil
	}

	rest = strings.TrimSpace(rest)

	// "shadow stop" cancels an active shadow
	if strings.ToLower(rest) == "stop" {
		if user.Character.GetMiscData("shadow-target-user") != nil ||
			user.Character.GetMiscData("shadow-target-mob") != nil {
			endShadow(user, "You stop shadowing your target.")
		} else {
			user.SendText(messaging.CategorySystem, "You aren't shadowing anyone.")
		}
		return true, nil
	}

	if rest == "" {
		user.SendText(messaging.CategorySystem, "Shadow whom?")
		return true, nil
	}

	// Resolve target in the current room, excluding the player themselves.
	target, err := actions.ResolveTargetActor(room, strings.ToLower(rest), actions.ResolveTargetOptions{
		ExcludeUserId: user.UserId,
	})
	if err != nil {
		// Check whether the name matched the player themselves.
		if pId, _ := room.FindByName(strings.ToLower(rest)); pId == user.UserId {
			user.SendText(messaging.CategorySystem, "You can't shadow yourself.")
			return true, nil
		}
		user.SendText(messaging.CategorySystem, "Shadow whom?")
		return true, nil
	}

	opts := actions.ShadowOptions{}
	if target.IsPlayer() {
		opts.TargetUserId = target.GetUserId()
	} else {
		opts.TargetMobInstanceId = target.GetMobInstanceId()
	}

	actor := &actions.UserActor{User: user, Room: room}
	result := actions.Shadow(actor, opts)

	if result.OnCooldown {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			"You need to wait %s before shadowing again.",
			result.Reason))
	}

	return true, nil
}

// endShadow clears the shadow target state, starts the cooldown, and
// optionally sends a reason message to the shadower.
func endShadow(user *users.UserRecord, reason string) {
	user.Character.SetMiscData("shadow-target-user", nil)
	user.Character.SetMiscData("shadow-target-mob", nil)
	user.Character.RemoveBuff(87)

	cfg := configs.GetBalanceConfig()
	cooldownKey := skills.Skullduggery.String(`shadow`)
	user.Character.TryCooldown(cooldownKey,
		fmt.Sprintf(`%d rounds`, cfg.ShadowCooldown))

	if reason != "" {
		user.SendText(messaging.CategorySystem, reason)
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
// room is the current room in which the detection check occurs; used
// to compute per-observer light conditions (NightVision, room darkness).
func shadowDetectionRoll(shadower *users.UserRecord, target *users.UserRecord, room *rooms.Room) bool {
	sneakScore := actions.CalcSneakScoreVsObserver(shadower.Character, target.Character, room)
	targetScore := actions.CalcSearchScore(target.Character)

	// OpposedRollStat(atk, def) returns true when atk (first arg) wins.
	// Target detects when targetScore beats sneakScore.
	detected, _, _, _ := dice.OpposedRollStat(targetScore, sneakScore)
	return detected
}
