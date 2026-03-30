package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
Skullduggery Skill
Level 1 - Sneak: attempt to enter a hidden state outside of combat.
Uses an opposed roll against each observer in the room.
*/
func Sneak(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Skullduggery)

	// If they don't have the skill, act like it's not a valid command
	if skillLevel < 1 {
		return false, nil
	}

	// Already hidden
	if user.Character.HasBuffFlag(buffs.Hidden) {
		user.SendText("You're already hidden!")
		return true, nil
	}

	if user.Character.Aggro != nil {
		user.SendText("You can't do that while in combat!")
		return true, nil
	}

	// Can't sneak while crafting
	if user.Character.CraftingState != nil {
		user.SendText(`You are busy crafting.`)
		return true, nil
	}

	cfg := configs.GetBalanceConfig()
	sneakCooldownKey := skills.Skullduggery.String(`sneak`)

	// Check cooldown — only block if on cooldown from a prior failure
	if user.Character.GetCooldown(sneakCooldownKey) > 0 {
		user.SendText(fmt.Sprintf(
			"You need to wait %d more rounds before you can try that again.",
			user.Character.GetCooldown(sneakCooldownKey)))
		return true, nil
	}

	// Sneak score: Dexterity + skill curve bonus (penalized by illumination)
	sneakScore := calcSneakScore(user.Character)

	// Build party member exclusion set so we skip allies
	partySet := map[int]bool{user.UserId: true}
	if party := parties.Get(user.UserId); party != nil {
		for _, memberId := range party.GetMembers() {
			partySet[memberId] = true
		}
	}

	spotted := false
	spotterName := ""
	rollHappened := false

	// Check each player in the room
	for _, observerId := range room.GetPlayers() {
		if partySet[observerId] {
			continue
		}
		observer := users.GetByUserId(observerId)
		if observer == nil {
			continue
		}

		searchRank := observer.Character.GetSkillLevel(skills.Search)
		observerScore := float64(observer.Character.Stats.Perception.ValueAdj) +
			combat.SkillMultiplier(searchRank)*25.0

		rollHappened = true
		success, _, _, _ := dice.OpposedRollStat(sneakScore, observerScore)
		if !success {
			spotted = true
			spotterName = observer.Character.Name
			observer.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi> tries to hide but you notice them.`,
				user.Character.Name))
			break
		}
	}

	// Check each mob in the room if not already spotted
	if !spotted {
		for _, mobInstanceId := range room.GetMobs() {
			m := mobs.GetInstance(mobInstanceId)
			if m == nil {
				continue
			}

			searchRank := m.Character.GetSkillLevel(skills.Search)
			observerScore := float64(m.Character.Stats.Perception.ValueAdj) +
				combat.SkillMultiplier(searchRank)*25.0

			rollHappened = true
			success, _, _, _ := dice.OpposedRollStat(sneakScore, observerScore)
			if !success {
				spotted = true
				spotterName = m.Character.Name
				break
			}
		}
	}

	// Progress the skill if at least one roll was made
	if rollHappened {
		user.Character.CheckSkillProgression(string(skills.Skullduggery), user.UserId, 1.0)
	}

	if spotted {
		// Apply failure cooldown so the player can't spam sneak
		user.Character.TryCooldown(sneakCooldownKey,
			fmt.Sprintf(`%d rounds`, cfg.SneakFailCooldown))
		user.SendText(fmt.Sprintf(
			`You try to blend into the shadows but <ansi fg="mobname">%s</ansi> notices you.`,
			spotterName))
		return true, nil
	}

	// Apply hidden buff via event queue (processes next tick).
	user.AddBuff(9, `skill`)
	// Set a misc-data flag so go.go knows we're hidden immediately,
	// even before the buff event processes on the next tick.
	user.Character.SetMiscData(`sneaking`, true)

	user.SendText(`You slip into the shadows.`)

	events.AddToQueue(events.SkillUsed{
		UserId: user.UserId,
		Skill:  skills.Skullduggery,
		Details: `sneak`,
	})

	return true, nil
}
