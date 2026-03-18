package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func HealingGel(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !mutations.HasMutation(user.Character.Mutations, "healing-gel") {
		user.SendText("You don't have that ability.")
		return true, nil
	}

	cfg := configs.GetBalanceConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	staminaCost := 8
	if user.Character.Stamina < staminaCost {
		user.SendText("You're too exhausted!")
		return true, nil
	}
	user.Character.Stamina -= staminaCost

	// Heal based on Vitality
	healAmount := int(float64(user.Character.Stats.Vitality.ValueAdj) * 0.15)
	if healAmount < 5 {
		healAmount = 5
	}

	healed := user.Character.Heal(healAmount)

	user.SendText(fmt.Sprintf(`<ansi fg="green-bold">A thick, regenerative gel oozes from your pores, knitting wounds as it spreads. (%s)</ansi>`,
		combat.GetHealDescription(healed, user.Character.HealthMax.Value)))

	room.SendText(
		fmt.Sprintf(`<ansi fg="green">A glistening gel coats <ansi fg="username">%s</ansi>'s skin, sealing wounds before your eyes.</ansi>`, user.Character.Name),
		user.UserId,
	)

	// Movement penalty for 2 rounds (slowed by gel)
	user.Character.AddCondition(characters.ConditionRecoveryPenalty, 2, 1.0, "healing-gel movement penalty")
	user.SendText(`<ansi fg="yellow">The gel weighs you down, slowing your movements.</ansi>`)

	events.AddToQueue(events.SkillUsed{UserId: user.UserId, Skill: skills.Skullduggery, Details: "healing-gel"})

	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
