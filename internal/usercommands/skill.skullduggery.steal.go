package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
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
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
Skullduggery Skill
Level 2 - Steal: attempt to steal from a mob or room container using
an opposed Dex+skill vs Perception roll.
*/
func Steal(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

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
		user.SendText("Steal from whom?")
		return true, nil
	}

	cfg := configs.GetBalanceConfig()
	cooldownKey := skills.Skullduggery.String(`steal`)

	// Check cooldown before doing target resolution
	if !user.Character.TryCooldown(cooldownKey,
		fmt.Sprintf(`%d real seconds`, int(cfg.StealCooldown))) {
		user.SendText(fmt.Sprintf(
			"You need to wait %d rounds before you can do that again.",
			user.Character.GetCooldown(cooldownKey)))
		return true, nil
	}

	isHidden := user.Character.HasBuffFlag(buffs.Hidden)

	// Compute attacker score
	rank := skillLevel
	base := float64(user.Character.Stats.Dexterity.ValueAdj) +
		combat.SkillMultiplier(rank)*25.0
	attackerScore := base * float64(cfg.StealSkillMultiplier)
	if isHidden {
		attackerScore += float64(cfg.StealHiddenBonus)
	}

	// Try to find a mob or player by name
	target, err := actions.ResolveTargetActor(room, args[0])
	if err == nil {
		if target.IsPlayer() {
			user.SendText("You can't steal from other players.")
			return true, nil
		}
		return stealFromMob(target.(*actions.MobActor).Mob.InstanceId, attackerScore, rank, user, room, cfg)
	}

	// Try container steal
	containerName := room.FindContainerByName(args[0])
	if containerName != `` {
		return stealFromContainer(containerName, attackerScore, rank, user, room)
	}

	user.SendText("Steal from whom?")
	return true, nil
}

// stealFromMob handles the creature steal path.
func stealFromMob(mobInstanceId int, attackerScore float64, rank int,
	user *users.UserRecord, room *rooms.Room, cfg configs.Balance) (bool, error) {

	m := mobs.GetInstance(mobInstanceId)
	if m == nil {
		user.SendText("They seem to have vanished.")
		return true, nil
	}

	if m.IsNonCombatant() {
		user.SendText(fmt.Sprintf(`You can't steal from <ansi fg="mobname">%s</ansi>.`, m.Character.Name))
		return true, nil
	}

	// Fire skill-used event
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.Skullduggery,
		Details: `steal`,
	})

	defenderScore := float64(m.Character.Stats.Perception.ValueAdj)

	success, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)

	// Always check skill progression regardless of outcome
	defer user.Character.CheckSkillProgression(string(skills.Skullduggery), user.UserId, 1.0)

	if success {
		stolenStuff := []string{}

		if m.Character.Gold > 0 {
			quarterGold := m.Character.Gold >> 2
			minGold := m.Character.Gold - quarterGold
			goldStolen := util.Rand(quarterGold) + minGold
			if goldStolen > 0 {
				m.Character.Gold -= goldStolen
				user.Character.Gold += goldStolen
				stolenStuff = append(stolenStuff,
					fmt.Sprintf(`<ansi fg="yellow-bold">%d gold</ansi>`, goldStolen))

				events.AddToQueue(events.EquipmentChange{
					UserId:     user.UserId,
					GoldChange: goldStolen,
				})
			}
		}

		if itemStolen, found := m.Character.GetRandomItem(); found {
			m.Character.RemoveItem(itemStolen)
			user.Character.StoreItem(itemStolen)

			events.AddToQueue(events.ItemOwnership{
				MobInstanceId: m.InstanceId,
				Item:          itemStolen,
				Gained:        false,
			})

			events.AddToQueue(events.ItemOwnership{
				UserId: user.UserId,
				Item:   itemStolen,
				Gained: true,
			})

			stolenStuff = append(stolenStuff,
				fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, itemStolen.DisplayName()))
		}

		if len(stolenStuff) == 0 {
			user.SendText(fmt.Sprintf(
				`You deftly rifle through <ansi fg="mobname">%s</ansi>'s `+
					`belongings but find nothing worth taking.`,
				m.Character.Name))
		} else {
			user.SendText(fmt.Sprintf(
				`You successfully steal %s from <ansi fg="mobname">%s</ansi>.`,
				strings.Join(stolenStuff, ` and `), m.Character.Name))
		}

	} else {
		user.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> catches you in the act!`,
			m.Character.Name))

		room.SendTextVisual(
			fmt.Sprintf(
				`<ansi fg="username">%s</ansi> gets caught trying to steal `+
					`from <ansi fg="mobname">%s</ansi>!`,
				user.Character.Name, m.Character.Name),
			user.UserId,
		)

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		m.Command(fmt.Sprintf(`attack @%d`, user.UserId))
	}

	return true, nil
}

// stealFromContainer handles the room-container steal path.
func stealFromContainer(containerName string, attackerScore float64, rank int,
	user *users.UserRecord, room *rooms.Room) (bool, error) {

	container, ok := room.Containers[containerName]
	if !ok {
		user.SendText("You don't see that here.")
		return true, nil
	}

	if len(container.Items) == 0 && container.Gold == 0 {
		user.SendText(fmt.Sprintf(
			`You root around inside the <ansi fg="itemname">%s</ansi> `+
				`but find it empty.`,
			containerName))
		return true, nil
	}

	// Fire skill-used event
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.Skullduggery,
		Details: `steal`,
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
			`<ansi fg="mobname">%s</ansi> spots you reaching into the `+
				`<ansi fg="itemname">%s</ansi>!`,
			spotterName, containerName))

		room.SendTextVisual(
			fmt.Sprintf(
				`<ansi fg="username">%s</ansi> is caught stealing from `+
					`the <ansi fg="itemname">%s</ansi>!`,
				user.Character.Name, containerName),
			user.UserId,
		)

		user.Character.CancelBuffsWithFlag(buffs.Hidden)
		return true, nil
	}

	// Success — take a random item (or gold) from the container
	if len(container.Items) > 0 {
		idx := util.Rand(len(container.Items))
		stolen := container.Items[idx]
		container.RemoveItem(stolen)
		room.Containers[containerName] = container
		user.Character.StoreItem(stolen)

		events.AddToQueue(events.ItemOwnership{
			UserId: user.UserId,
			Item:   stolen,
			Gained: true,
		})

		user.SendText(fmt.Sprintf(
			`You quietly slip <ansi fg="itemname">%s</ansi> from the `+
				`<ansi fg="itemname">%s</ansi>.`,
			stolen.DisplayName(), containerName))
	} else {
		// Container only has gold
		quarterGold := container.Gold >> 2
		minGold := container.Gold - quarterGold
		goldStolen := util.Rand(quarterGold) + minGold
		if goldStolen > 0 {
			container.Gold -= goldStolen
			room.Containers[containerName] = container
			user.Character.Gold += goldStolen

			events.AddToQueue(events.EquipmentChange{
				UserId:     user.UserId,
				GoldChange: goldStolen,
			})

			user.SendText(fmt.Sprintf(
				`You quietly lift <ansi fg="yellow-bold">%d gold</ansi> `+
					`from the <ansi fg="itemname">%s</ansi>.`,
				goldStolen, containerName))
		}
	}

	return true, nil
}
