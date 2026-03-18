package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
Skullduggery Skill
Level 2 - Plant: slip an item from your backpack into an NPC's inventory
or a room container, unnoticed. Mirror of steal — same roll formula and
cooldown.
*/
func Plant(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Skullduggery)

	// Requires skullduggery rank 2
	if skillLevel < 1 {
		return false, nil
	}
	if skillLevel < 2 {
		user.SendText("You aren't advanced enough at skullduggery for that.")
		return true, nil
	}

	if user.Character.Aggro != nil {
		user.SendText("You can't do that while in combat!")
		return true, nil
	}

	if room.AreMobsAttacking(user.UserId) {
		user.SendText("You can't do that while you are under attack!")
		return true, nil
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		user.SendText("Plant what?")
		return true, nil
	}

	if len(args) == 1 {
		user.SendText("Plant on whom?")
		return true, nil
	}

	cfg := configs.GetBalanceConfig()
	cooldownKey := skills.Skullduggery.String(`steal`)

	// Shares the steal cooldown
	if !user.Character.TryCooldown(cooldownKey,
		fmt.Sprintf(`%d real seconds`, int(cfg.StealCooldown))) {
		user.SendText(fmt.Sprintf(
			"You need to wait %d rounds before you can do that again.",
			user.Character.GetCooldown(cooldownKey)))
		return true, nil
	}

	// Target is the last word; item name is everything before it.
	targetName := args[len(args)-1]
	itemName := strings.Join(args[:len(args)-1], " ")

	// Find item in player's backpack
	plantItem, found := user.Character.FindInBackpack(itemName)
	if !found {
		user.SendText("You don't have that.")
		return true, nil
	}

	isHidden := user.Character.HasBuffFlag(buffs.Hidden)

	// Compute attacker score (identical formula to steal)
	rank := skillLevel
	base := float64(user.Character.Stats.Dexterity.ValueAdj) +
		combat.SkillMultiplier(rank)*25.0
	attackerScore := base * float64(cfg.StealSkillMultiplier)
	if isHidden {
		attackerScore += float64(cfg.StealHiddenBonus)
	}

	// Resolve target — mob or container only, not players
	targetPlayerId, targetMobInstanceId := room.FindByName(targetName)

	if targetPlayerId > 0 {
		user.SendText("You can't plant items on other players.")
		return true, nil
	}

	if targetMobInstanceId > 0 {
		return plantOnMob(targetMobInstanceId, plantItem, attackerScore, rank, user, room)
	}

	// Try container
	containerName := room.FindContainerByName(targetName)
	if containerName != `` {
		return plantInContainer(containerName, plantItem, attackerScore, rank, user, room)
	}

	user.SendText("Plant on whom?")
	return true, nil
}

// plantOnMob handles slipping an item into a creature's inventory.
func plantOnMob(mobInstanceId int, plantItem items.Item, attackerScore float64, rank int,
	user *users.UserRecord, room *rooms.Room) (bool, error) {

	m := mobs.GetInstance(mobInstanceId)
	if m == nil {
		user.SendText("They seem to have vanished.")
		return true, nil
	}

	// Fire skill-used event
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.Skullduggery,
		Details: `plant`,
	})

	defenderScore := float64(m.Character.Stats.Perception.ValueAdj)

	success, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)

	// Always check skill progression regardless of outcome
	defer user.Character.CheckSkillProgression(string(skills.Skullduggery), user.UserId, 1.0)

	if success {
		m.Character.StoreItem(plantItem)
		user.Character.RemoveItem(plantItem)

		events.AddToQueue(events.ItemOwnership{
			UserId: user.UserId,
			Item:   plantItem,
			Gained: false,
		})

		events.AddToQueue(events.ItemOwnership{
			MobInstanceId: m.InstanceId,
			Item:          plantItem,
			Gained:        true,
		})

		user.SendText(fmt.Sprintf(
			`You deftly slip the <ansi fg="itemname">%s</ansi> into `+
				`<ansi fg="mobname">%s</ansi>'s belongings unnoticed.`,
			plantItem.DisplayName(), m.Character.Name))
	} else {
		user.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> catches you in the act!`,
			m.Character.Name))

		room.SendText(
			fmt.Sprintf(
				`<ansi fg="username">%s</ansi> gets caught trying to plant `+
					`something on <ansi fg="mobname">%s</ansi>!`,
				user.Character.Name, m.Character.Name),
			user.UserId,
		)

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		m.Command(fmt.Sprintf(`attack @%d`, user.UserId))
	}

	return true, nil
}

// plantInContainer handles slipping an item into a room container.
func plantInContainer(containerName string, plantItem items.Item, attackerScore float64, rank int,
	user *users.UserRecord, room *rooms.Room) (bool, error) {

	container, ok := room.Containers[containerName]
	if !ok {
		user.SendText("You don't see that here.")
		return true, nil
	}

	// Fire skill-used event
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.Skullduggery,
		Details: `plant`,
	})

	// Always check skill progression regardless of outcome
	defer user.Character.CheckSkillProgression(string(skills.Skullduggery), user.UserId, 1.0)

	// Find highest Perception observer (players + mobs, excluding party)
	partySet := map[int]bool{user.UserId: true}
	if party := parties.Get(user.UserId); party != nil {
		for _, memberId := range party.GetMembers() {
			partySet[memberId] = true
		}
	}

	highestPerception := 0.0
	spotterName := ""
	hasObserver := false

	for _, observerId := range room.GetPlayers() {
		if partySet[observerId] {
			continue
		}
		observer := users.GetByUserId(observerId)
		if observer == nil {
			continue
		}
		perScore := float64(observer.Character.Stats.Perception.ValueAdj)
		if perScore > highestPerception {
			highestPerception = perScore
			spotterName = observer.Character.Name
			hasObserver = true
		}
	}

	for _, mobInstanceId := range room.GetMobs() {
		m := mobs.GetInstance(mobInstanceId)
		if m == nil {
			continue
		}
		perScore := float64(m.Character.Stats.Perception.ValueAdj)
		if perScore > highestPerception {
			highestPerception = perScore
			spotterName = m.Character.Name
			hasObserver = true
		}
	}

	success := true
	if hasObserver {
		success, _, _, _ = dice.OpposedRollStat(attackerScore, highestPerception)
	}

	if !success {
		user.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> spots you slipping something into `+
				`the <ansi fg="itemname">%s</ansi>!`,
			spotterName, containerName))

		room.SendText(
			fmt.Sprintf(
				`<ansi fg="username">%s</ansi> is caught planting something in `+
					`the <ansi fg="itemname">%s</ansi>!`,
				user.Character.Name, containerName),
			user.UserId,
		)

		user.Character.CancelBuffsWithFlag(buffs.Hidden)
		return true, nil
	}

	// Success — place item into container
	container.AddItem(plantItem)
	user.Character.RemoveItem(plantItem)
	room.Containers[containerName] = container

	events.AddToQueue(events.ItemOwnership{
		UserId: user.UserId,
		Item:   plantItem,
		Gained: false,
	})

	user.SendText(fmt.Sprintf(
		`You quietly slip your <ansi fg="itemname">%s</ansi> into `+
			`the <ansi fg="itemname">%s</ansi>.`,
		plantItem.DisplayName(), containerName))

	return true, nil
}
