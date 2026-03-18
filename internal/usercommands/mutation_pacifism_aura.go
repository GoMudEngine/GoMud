package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func PacifismAura(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !mutations.HasMutation(user.Character.Mutations, "pacifism-aura") {
		user.SendText("You don't have that ability.")
		return true, nil
	}

	cfg := configs.GetBalanceConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	staminaCost := 12
	if user.Character.Stamina < staminaCost {
		user.SendText("You're too exhausted!")
		return true, nil
	}
	user.Character.Stamina -= staminaCost

	// De-aggro mobs in the room that are targeting this player
	deAggroCount := 0
	for _, mobInstId := range room.GetMobs(rooms.FindAll) {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil || mob.Character.Aggro == nil {
			continue
		}
		if mob.Character.Aggro.UserId == user.UserId {
			mob.Character.Aggro = nil
			deAggroCount++
		}
	}

	user.SendText(`<ansi fg="cyan-bold">A wave of calming energy pulses outward from you. Hostile intent fades from nearby creatures.</ansi>`)
	room.SendText(
		fmt.Sprintf(`<ansi fg="cyan">A soothing aura radiates from <ansi fg="username">%s</ansi>. Nearby creatures seem to lose their aggression.</ansi>`, user.Character.Name),
		user.UserId,
	)

	if deAggroCount > 0 {
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">%d creature(s) lose interest in attacking you.</ansi>`, deAggroCount))
	}

	// Self cannot attack for 3 rounds (pacifism penalty)
	user.Character.AddCondition(characters.ConditionRecoveryPenalty, 3, 1.0, "pacifism-aura")
	user.SendText(`<ansi fg="yellow">The aura's peaceful influence washes over you as well. You cannot bring yourself to attack.</ansi>`)

	// Also clear own aggro
	user.Character.Aggro = nil

	events.AddToQueue(events.SkillUsed{UserId: user.UserId, Skill: skills.Skullduggery, Details: "pacifism-aura"})

	return true, nil
}
