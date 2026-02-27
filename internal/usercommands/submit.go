package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Submit(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Must be in combat to use submit
	if user.Character.Aggro == nil {
		user.SendText("You must be in combat to attempt a submission!")
		return true, nil
	}

	// Must be in Grounded position
	if user.Character.CombatPosition != characters.PositionGrounded {
		user.SendText("You must be in a grounded grapple to attempt a submission!")
		return true, nil
	}

	// Must be the grapple controller
	if !user.Character.HasCondition(characters.ConditionGrappleController) {
		user.SendText("You must be in control of the grapple to attempt a submission!")
		return true, nil
	}

	// Check shared special move cooldown
	cfg := configs.GetGamePlayConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	// Get current target from aggro
	targetPlayerId := user.Character.Aggro.UserId
	targetMobId := user.Character.Aggro.MobInstanceId

	var targetChar *users.UserRecord
	var targetMob *mobs.Mob
	var targetName string

	if targetMobId > 0 {
		targetMob = mobs.GetInstance(targetMobId)
		if targetMob == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetName = targetMob.Character.Name
	} else if targetPlayerId > 0 {
		targetChar = users.GetByUserId(targetPlayerId)
		if targetChar == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetName = targetChar.Character.Name
	} else {
		user.SendText("You have no target!")
		return true, nil
	}

	// Attempt the submission
	var result combat.SubmissionResult
	if targetMob != nil {
		result = combat.AttemptSubmission(user.Character, &targetMob.Character)
	} else {
		result = combat.AttemptSubmission(user.Character, targetChar.Character)
	}

	// Apply result and send messages
	if result.Success {
		// Determine opponent choice (MVP: auto-decision based on health/type)
		// NPCs always resist, players yield if health < 25%
		choice := "resist"
		if targetChar != nil {
			healthPercent := (float64(targetChar.Character.Health) / float64(targetChar.Character.HealthMax.Value)) * 100.0
			if healthPercent < 25.0 {
				choice = "yield"
			}
		}
		// targetMob always resists (already set to "resist")

		// Apply the submission result
		if targetMob != nil {
			combat.ApplySubmissionSuccess(user.Character, &targetMob.Character, choice)
		} else {
			combat.ApplySubmissionSuccess(user.Character, targetChar.Character, choice)
		}

		if choice == "yield" {
			// Target yields - combat ends
			user.SendText(fmt.Sprintf(`<ansi fg="yellow-bold">You force %s into submission! They yield!</ansi>`, targetName))
			user.SendText(fmt.Sprintf(`<ansi fg="green">%s is now helpless at your mercy.</ansi>`, targetName))

			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="red-bold">%s forces you into submission! You have no choice but to yield!</ansi>`, user.Character.Name))
				targetChar.SendText(`<ansi fg="red">You are helpless on the ground.</ansi>`)
			}

			room.SendText(
				fmt.Sprintf(`<ansi fg="combat">%s forces %s into submission! %s yields!</ansi>`, user.Character.Name, targetName, targetName),
				user.UserId, targetPlayerId,
			)

			// End combat for both
			user.Character.Aggro = nil
			if targetMob != nil {
				targetMob.Character.Aggro = nil
			} else {
				targetChar.Character.Aggro = nil
			}

		} else {
			// Target resists - takes damage
			damage := int(float64(user.Character.Stats.Strength.ValueAdj) * 2.0)
			if damage < 1 {
				damage = 1
			}

			// Get target's max HP for damage description
			targetMaxHP := 0
			if targetMob != nil {
				targetMaxHP = targetMob.Character.HealthMax.Value
			} else if targetChar != nil {
				targetMaxHP = targetChar.Character.HealthMax.Value
			}

			user.SendText(fmt.Sprintf(`<ansi fg="yellow-bold">You attempt to force %s into submission, but they resist!</ansi>`, targetName))
			user.SendText(fmt.Sprintf(`<ansi fg="combat">You wrench and torque, dealing %s!</ansi>`, combat.GetDamageDescription(damage, targetMaxHP)))

			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="red">%s attempts to force you into submission!</ansi>`, user.Character.Name))
				targetChar.SendText(fmt.Sprintf(`<ansi fg="red-bold">You resist, but take %s from the brutal hold!</ansi>`, combat.GetDamageDescription(damage, targetMaxHP)))
			}

			room.SendText(
				fmt.Sprintf(`<ansi fg="combat">%s attempts to force %s into submission! %s resists through the pain!</ansi>`, user.Character.Name, targetName, targetName),
				user.UserId, targetPlayerId,
			)
		}
	} else {
		// Submission failed - dramatic reversal
		if targetMob != nil {
			combat.ApplySubmissionFailure(user.Character, &targetMob.Character)
		} else {
			combat.ApplySubmissionFailure(user.Character, targetChar.Character)
		}

		user.SendText(fmt.Sprintf(`<ansi fg="red-bold">Your submission attempt against %s fails!</ansi>`, targetName))
		user.SendText(`<ansi fg="red">You overcommit and fall to the ground as they escape!</ansi>`)

		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="yellow">%s tries to force you into submission, but you escape!`, user.Character.Name))
			targetChar.SendText(`<ansi fg="green">They fall to the ground from overcommitting!</ansi>`)
		}

		room.SendText(
			fmt.Sprintf(`<ansi fg="combat">%s attempts a submission, but %s escapes and %s falls to the ground!</ansi>`, user.Character.Name, targetName, user.Character.Name),
			user.UserId, targetPlayerId,
		)
	}

	// Stage 30.1: Record combat analytics
	submitTgtType := combat.Mob
	var submitTgtChar *characters.Character
	if targetMob != nil {
		submitTgtChar = &targetMob.Character
	} else {
		submitTgtType = combat.User
		submitTgtChar = targetChar.Character
	}
	combat.RecordSpecialMove(combat.User, submitTgtType, "submit", result.Success, 0, user.Character, submitTgtChar, util.GetRoundCount())

	// Progress unarmed combat skill
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.UnarmedCombat,
		Details: "submit",
	})

	// Submit costs the current combat round
	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
