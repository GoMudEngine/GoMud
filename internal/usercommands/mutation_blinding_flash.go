package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func BlindingFlash(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !mutations.HasMutation(user.Character.Mutations, "blinding-flash") {
		user.SendText("You don't have that ability.")
		return true, nil
	}

	if !user.Character.IsInCombat() {
		user.SendText("You must be in combat to use blinding flash!")
		return true, nil
	}

	cfg := configs.GetBalanceConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	staminaCost := 14
	if user.Character.Stamina < staminaCost {
		user.SendText("You're too exhausted!")
		return true, nil
	}
	user.Character.Stamina -= staminaCost

	attackerScore := float64(user.Character.GetSkillLevel(skills.UnarmedCombat)) + float64(user.Character.Stats.Willpower.ValueAdj)

	user.SendText(`<ansi fg="white-bold">Blinding light erupts from your skin in a searing flash!</ansi>`)
	room.SendTextVisual(
		fmt.Sprintf(`<ansi fg="white-bold">A blinding flash of light erupts from <ansi fg="username">%s</ansi>!</ansi>`, user.Character.Name),
		user.UserId,
	)

	// AoE blind: all mobs in room
	blindedCount := 0
	for _, mobInstId := range room.GetMobs(rooms.FindAll) {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil {
			continue
		}

		defenderScore := float64(mob.Character.Stats.Willpower.ValueAdj) + float64(mob.Character.GetCombatSkillLevel())
		attackSuccess, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)

		if attackSuccess {
			mob.Character.AddCondition(characters.ConditionBlinded, 3, 0.5, "blinding-flash")
			blindedCount++
		}
	}

	if blindedCount > 0 {
		user.SendText(fmt.Sprintf(`<ansi fg="white">%d creature(s) are blinded by the flash!</ansi>`, blindedCount))
	}

	// Self-blind (shorter duration)
	user.Character.AddCondition(characters.ConditionBlinded, 1, 0.7, "blinding-flash self")
	user.SendText(`<ansi fg="yellow">The afterimage sears your own vision briefly.</ansi>`)

	events.AddToQueue(events.SkillUsed{UserId: user.UserId, Skill: skills.UnarmedCombat, Details: "blinding-flash"})

	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
