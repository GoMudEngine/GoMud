package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Submit(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use submit
	if mob.Character.Aggro == nil {
		return true, nil
	}

	// Must be in Grounded position
	if mob.Character.CombatPosition != characters.PositionGrounded {
		return true, nil
	}

	// Must be the grapple controller
	if !mob.Character.HasCondition(characters.ConditionGrappleController) {
		return true, nil
	}

	// Check shared special move cooldown
	cfg := configs.GetBalanceConfig()
	if !mob.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return true, nil
	}

	// Get current target from aggro
	targetPlayerId := mob.Character.Aggro.UserId
	targetMobId := mob.Character.Aggro.MobInstanceId

	var targetChar *users.UserRecord
	var targetMob *mobs.Mob
	var targetName string

	if targetMobId > 0 {
		targetMob = mobs.GetInstance(targetMobId)
		if targetMob == nil {
			return true, nil
		}
		targetName = targetMob.Character.Name
	} else if targetPlayerId > 0 {
		targetChar = users.GetByUserId(targetPlayerId)
		if targetChar == nil {
			return true, nil
		}
		targetName = targetChar.Character.Name
	} else {
		return true, nil
	}

	// Attempt the submission
	var result combat.SubmissionResult
	if targetMob != nil {
		result = combat.AttemptSubmission(&mob.Character, &targetMob.Character)
	} else {
		result = combat.AttemptSubmission(&mob.Character, targetChar.Character)
	}

	// Apply result and send messages
	canSee := targetChar == nil || canSeeInDark(targetChar, room)
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
			combat.ApplySubmissionSuccess(&mob.Character, &targetMob.Character, choice)
		} else {
			combat.ApplySubmissionSuccess(&mob.Character, targetChar.Character, choice)
		}

		if choice == "yield" {
			// Target yields - combat ends
			if targetChar != nil {
				if canSee {
					targetChar.SendText(fmt.Sprintf(`<ansi fg="red-bold">%s forces you into submission! You have no choice but to yield!</ansi>`, mob.Character.Name))
				} else {
					targetChar.SendText(`<ansi fg="red-bold">Something forces you into submission! You have no choice but to yield!</ansi>`)
				}
				targetChar.SendText(`<ansi fg="red">You are helpless on the ground.</ansi>`)
			}

			sendRoomText(room,
				fmt.Sprintf(`<ansi fg="combat">%s forces %s into submission! %s yields!</ansi>`, mob.Character.Name, targetName, targetName),
				targetPlayerId)

			// End combat for both
			mob.Character.EndAggro()
			if targetMob != nil {
				targetMob.Character.EndAggro()
			} else {
				targetChar.Character.EndAggro()
			}

		} else {
			// Target resists - takes damage
			damage := int(float64(mob.Character.Stats.Strength.ValueAdj) * 2.0)
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

			if targetChar != nil {
				if canSee {
					targetChar.SendText(fmt.Sprintf(`<ansi fg="red">%s attempts to force you into submission!</ansi>`, mob.Character.Name))
				} else {
					targetChar.SendText(`<ansi fg="red">Something attempts to force you into submission!</ansi>`)
				}
				targetChar.SendText(fmt.Sprintf(`<ansi fg="red-bold">You resist, but take %s damage from the brutal hold!</ansi>`, combat.GetDamageDescription(damage, targetMaxHP)))
			}

			sendRoomText(room,
				fmt.Sprintf(`<ansi fg="combat">%s attempts to force %s into submission! %s resists through the pain!</ansi>`, mob.Character.Name, targetName, targetName),
				targetPlayerId)
		}
	} else {
		// Submission failed - dramatic reversal
		if targetMob != nil {
			combat.ApplySubmissionFailure(&mob.Character, &targetMob.Character)
		} else {
			combat.ApplySubmissionFailure(&mob.Character, targetChar.Character)
		}

		if targetChar != nil {
			if canSee {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="yellow">%s tries to force you into submission, but you escape!</ansi>`, mob.Character.Name))
			} else {
				targetChar.SendText(`<ansi fg="yellow">Something tries to force you into submission, but you escape!</ansi>`)
			}
			targetChar.SendText(`<ansi fg="green">They fall to the ground from overcommitting!</ansi>`)
		}

		sendRoomText(room,
			fmt.Sprintf(`<ansi fg="combat">%s attempts a submission, but %s escapes and %s falls to the ground!</ansi>`, mob.Character.Name, targetName, mob.Character.Name),
			targetPlayerId)
	}

	// Submit costs the current combat round
	if mob.Character.Aggro != nil {
		mob.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
