package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
Stealth Skill
Level 3 - Pickpocket: steal from a target while hidden
*/
func Pickpocket(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Stealth)

	// Requires stealth rank 3
	if skillLevel < 3 {
		return false, nil
	}

	// Must be sneaking
	isSneaking := user.Character.HasBuffFlag(buffs.Hidden)

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
		user.SendText("Pickpocket whom?")
		return true, nil
	}

	pickPlayerId, pickMobInstanceId := room.FindByName(args[0])

	if pickPlayerId > 0 || pickMobInstanceId > 0 {

		if !user.Character.TryCooldown(skills.Stealth.String(`pickpocket`), "1 real minute") {
			user.SendText(fmt.Sprintf("You need to wait %d rounds before you can do that again!", user.Character.GetCooldown(skills.Stealth.String(`pickpocket`))))
			return true, nil
		}

	}

	if pickMobInstanceId > 0 {

		// Fire an event that a skill has been used
		events.AddToQueue(events.SkillUsed{user.UserId, skills.Stealth, `pickpocket`})

		m := mobs.GetInstance(pickMobInstanceId)

		if m != nil {

			levelDelta := user.Character.Level - m.Character.Level
			if levelDelta < 1 {
				levelDelta = 1
			}

			chanceIn100 := (user.Character.Stats.Dexterity.ValueAdj+user.Character.Stats.Perception.ValueAdj+user.Character.Stats.Charisma.ValueAdj)/3 - m.Character.Stats.Charisma.ValueAdj
			chanceIn100 /= levelDelta
			if chanceIn100 < 0 {
				chanceIn100 = 1
			}
			if isSneaking {
				chanceIn100 += 15
			}

			roll := util.Rand(100)

			util.LogRoll(`Pickpocket`, roll, chanceIn100)

			if roll < chanceIn100 {

				stolenStuff := []string{}

				if m.Character.Gold > 0 {
					halfGold := m.Character.Gold >> 2
					minGold := m.Character.Gold - halfGold
					goldStolen := util.Rand(halfGold) + minGold
					if goldStolen > 0 {
						m.Character.Gold -= goldStolen
						user.Character.Gold += goldStolen
						stolenStuff = append(stolenStuff, fmt.Sprintf(`<ansi fg="yellow-bold">%d gold</ansi>`, goldStolen))

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

					stolenStuff = append(stolenStuff, fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, itemStolen.DisplayName()))
				}

				if len(stolenStuff) < 1 {
					user.SendText(
						fmt.Sprintf(`You succeed in picking the pockets of <ansi fg="mobname">%s</ansi> but find nothing!`, m.Character.Name))
				} else {
					user.SendText(
						fmt.Sprintf(`You succeed in picking the pockets of <ansi fg="mobname">%s</ansi> and steal %s`, m.Character.Name, strings.Join(stolenStuff, ` and `)))
				}

			} else {

				user.SendText(
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> catches you in the act!`, m.Character.Name))

				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> gets caught trying to pick the pockets of <ansi fg="mobname">%s</ansi>!`, user.Character.Name, m.Character.Name),
					user.UserId,
				)

				user.Character.CancelBuffsWithFlag(buffs.Hidden)

				m.Command(fmt.Sprintf(`attack @%d`, user.UserId))

			}

		}

	} else if pickPlayerId > 0 {

		// Fire an event that a skill has been used
		events.AddToQueue(events.SkillUsed{user.UserId, skills.Stealth, `pickpocket`})

		if p := users.GetByUserId(pickPlayerId); p != nil {

			if pvpErr := room.CanPvp(user, p); pvpErr != nil {
				user.SendText(pvpErr.Error())
				return true, nil
			}

			levelDelta := user.Character.Level - p.Character.Level
			if levelDelta < 1 {
				levelDelta = 1
			}

			chanceIn100 := (user.Character.Stats.Dexterity.ValueAdj+user.Character.Stats.Perception.ValueAdj+user.Character.Stats.Charisma.ValueAdj)/3 - p.Character.Stats.Charisma.ValueAdj
			chanceIn100 /= levelDelta
			if chanceIn100 < 0 {
				chanceIn100 = 1
			}
			if isSneaking {
				chanceIn100 += 15
			}

			roll := util.Rand(100)

			util.LogRoll(`Pickpocket`, roll, chanceIn100)

			if roll < chanceIn100 {

				stolenStuff := []string{}

				if p.Character.Gold > 0 {
					halfGold := p.Character.Gold >> 2
					minGold := p.Character.Gold - halfGold
					goldStolen := util.Rand(halfGold) + minGold
					if goldStolen > 0 {
						p.Character.Gold -= goldStolen
						user.Character.Gold += goldStolen
						stolenStuff = append(stolenStuff, fmt.Sprintf(`<ansi fg="yellow-bold">%d gold</ansi>`, goldStolen))

						events.AddToQueue(events.EquipmentChange{
							UserId:     user.UserId,
							GoldChange: goldStolen,
						})

						events.AddToQueue(events.EquipmentChange{
							UserId:     p.UserId,
							GoldChange: -goldStolen,
						})
					}
				}

				if itemStolen, found := p.Character.GetRandomItem(); found {

					p.Character.RemoveItem(itemStolen)
					user.Character.StoreItem(itemStolen)

					events.AddToQueue(events.ItemOwnership{
						UserId: user.UserId,
						Item:   itemStolen,
						Gained: true,
					})

					events.AddToQueue(events.ItemOwnership{
						UserId: p.UserId,
						Item:   itemStolen,
						Gained: false,
					})

					stolenStuff = append(stolenStuff, fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, itemStolen.DisplayName()))
				}

				if len(stolenStuff) < 1 {
					user.SendText(
						fmt.Sprintf(`You succeed in picking the pockets of <ansi fg="username">%s</ansi> but find nothing!`, p.Character.Name))
				} else {
					user.SendText(
						fmt.Sprintf(`You succeed in picking the pockets of <ansi fg="username">%s</ansi> and steal %s`, p.Character.Name, strings.Join(stolenStuff, ` and `)))
				}

			} else {

				user.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> catches you in the act!`, p.Character.Name))

				p.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> is trying to pick your pockets!`, user.Character.Name))

				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> gets caught trying to pick the pockets of <ansi fg="username">%s</ansi>!`, user.Character.Name, p.Character.Name),
					user.UserId,
				)

				user.Character.CancelBuffsWithFlag(buffs.Hidden)

			}
		}

	} else {

		user.SendText("Pickpocket who?")
	}

	return true, nil
}
