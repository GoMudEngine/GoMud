package usercommands

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Uncurse(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	return Enchant("uncurse "+rest, user, room, flags)
}

func Unenchant(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	return Enchant("remove "+rest, user, room, flags)
}

/*
Enchant Skill
Level 1 - Enchant a weapon with a damage bonus.
Level 2 - Enchant equipment with a defensive bonus.
Level 3 - Add a stat bonus to a weapon or equipment in addition to the above.
Level 4 - Remove the enchantment or curse from any object.
*/
func Enchant(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Enchant)

	if skillLevel == 0 {
		user.SendText("You don't know how to enchant.")
		return true, fmt.Errorf("you don't know how to enchant")
	}

	if len(rest) == 0 {
		user.SendText(`Type <ansi fg="command">help enchant</ansi> for more information on the enchant skill.`)
		return true, nil
	}

	removeEnchantment := false
	if strings.HasPrefix(rest, `remove `) {
		rest = strings.TrimPrefix(rest, `remove `)
		removeEnchantment = true
	}

	removeCurse := false
	if strings.HasPrefix(rest, `uncurse `) {
		rest = strings.TrimPrefix(rest, `uncurse `)
		removeCurse = true
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 1 {
		user.SendText(`You must be more specific.`)
		return true, nil
	}

	onlyConsider := false
	if args[0] == `chance` {
		onlyConsider = true
		args = args[1:]
		rest = strings.Join(args, ` `)
	}

	// Check whether the user has an item in their inventory that matches
	matchItem, found := user.Character.FindInBackpack(rest)

	if !found {
		user.SendText(fmt.Sprintf("You don't have a %s to enchant. Is it still worn, perhaps?", rest))
	} else {

		if matchItem.GetSpec().Type != items.Weapon && matchItem.GetSpec().Subtype != items.Wearable {
			user.SendText(`Enchant only works on weapons and armor.`)
			return true, nil
		}

		if removeCurse || removeEnchantment {
			if skillLevel < 4 {
				user.SendText(`Your skills are not good enough. Type <ansi fg="command">help enchant</ansi> for more information on the enchant skill.`)
				return true, nil
			}
		}

		if removeCurse {

			if !matchItem.IsCursed() {
				user.SendText(`That's not cursed.`)
				return true, nil
			}

			// Fire an event that a skill has been used
			events.AddToQueue(events.SkillUsed{user.UserId, skills.Enchant, `uncurse`})

			user.Character.RemoveItem(matchItem)
			matchItem.Uncurse()
			user.Character.StoreItem(matchItem)

			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> holds out their cursed <ansi fg="itemname">%s</ansi> concentrates. An malevolent ethereal whisp seems to float away.`, user.Character.Name, matchItem.DisplayName()),
				user.UserId,
			)

			user.SendText(
				fmt.Sprintf(`You remove the curse from the <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()),
			)

			return true, nil
		}

		if removeEnchantment {

			if skillLevel < 4 {
				user.SendText(`Type <ansi fg="command">help enchant</ansi> for more information on the enchant skill.`)
				return true, nil
			}

			// Fire an event that a skill has been used
			events.AddToQueue(events.SkillUsed{user.UserId, skills.Enchant, `remove`})

			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> holds out their <ansi fg="itemname">%s</ansi> and slowly waves their hand over it. Runes appear to glow on the surface, which fade as they float away.`, user.Character.Name, matchItem.DisplayName()),
				user.UserId,
			)

			user.SendText(
				fmt.Sprintf(`You remove the enchantment from the <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()),
			)

			user.Character.RemoveItem(matchItem)
			matchItem.UnEnchant()
			user.Character.StoreItem(matchItem)

			return true, nil

		}

		/*
			if matchItem.IsEnchanted() {
				user.SendText( fmt.Sprintf(`The <ansi fg="itemname">%s</ansi> is already enchanted.`, matchItem.DisplayName()))
				return true, nil
			}
		*/

		chanceToDestroy := 50
		chanceToDestroy -= skillLevel * 10                              // 10-40% reduction
		chanceToDestroy += int(matchItem.Enchantments) * 20             // 20% greater chance for each enchantment.
		chanceToDestroy -= user.Character.Stats.Willpower.ValueAdj >> 2 // 1% less chance for each 4 willpower points

		if onlyConsider {
			user.SendText(
				fmt.Sprintf(`Your <ansi fg="itemname">%s</ansi> has been enchanted %d times. There is a %d%% chance it would be destroyed.`, matchItem.DisplayName(), matchItem.Enchantments, chanceToDestroy),
			)
			return true, nil
		}

		if !user.Character.TryCooldown(skills.Enchant.String(), "15 real minutes") {
			user.SendText(
				fmt.Sprintf("You need to wait %d more rounds to use that skill again.", user.Character.GetCooldown(skills.Enchant.String())),
			)
			return true, errors.New(`you're doing that too often`)
		}

		// Fire an event that a skill has been used
		events.AddToQueue(events.SkillUsed{user.UserId, skills.Enchant, ``})

		damageBonus := 0
		defenseBonus := 0
		statBonus := map[string]int{}
		cursed := false

		// At skill level 1, can enchant only weapons
		if matchItem.GetSpec().Type == items.Weapon {
			damageBonus = int(math.Ceil(math.Sqrt(float64(user.Character.Stats.Willpower.ValueAdj))))
		}

		// At skill level 2, can enchant weapons and armor
		if skillLevel >= 2 && matchItem.GetSpec().Subtype == items.Wearable {
			defenseBonus = int(math.Ceil(math.Sqrt(float64(user.Character.Stats.Willpower.ValueAdj))))
		}

		// At skill level 3, can  provide stat bonuses
		if skillLevel >= 3 {
			allStats := []string{`strength`, `dexterity`, `perception`, `vitality`, `willpower`, `charisma`}

			bonusCt := skillLevel - 1 //
			for i := 0; i < bonusCt; i++ {
				// select a random stat
				chosenStat := allStats[util.Rand(len(allStats))]
				statBonus[chosenStat] = int(math.Ceil(math.Sqrt(float64(user.Character.Stats.Willpower.ValueAdj))))
			}
		}

		roll := util.Rand(100)

		util.LogRoll(`Enchant->Cursed`, roll, 25)

		if roll < 25 {
			cursed = true
		}

		user.Character.RemoveItem(matchItem)

		roll = util.Rand(100)

		util.LogRoll(`Enchant->Destroy`, roll, chanceToDestroy)

		if roll < chanceToDestroy {

			events.AddToQueue(events.ItemOwnership{
				UserId: user.UserId,
				Item:   matchItem,
				Gained: false,
			})

			user.SendText(fmt.Sprintf(`The <ansi fg="itemname">%s</ansi> explodes in a shower of sparks!`, matchItem.DisplayName()))
			room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> holds out their <ansi fg="itemname">%s</ansi> and slowly waves their hand over it. The <ansi fg="itemname">%s</ansi> explodes in a shower of sparks!`, user.Character.Name, matchItem.DisplayName(), matchItem.DisplayName()), user.UserId)
			return true, nil
		}

		matchItem.Enchant(damageBonus, defenseBonus, statBonus, cursed)
		user.Character.StoreItem(matchItem)

		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> holds out their <ansi fg="itemname">%s</ansi> and slowly waves their hand over it. The <ansi fg="itemname">%s</ansi> glows briefly for a moment and then the glow fades away.`, user.Character.Name, matchItem.DisplayName(), matchItem.DisplayName()),
			user.UserId,
		)

		user.SendText(
			fmt.Sprintf(`You enchant the <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()),
		)

		if cursed {
			user.SendText(fmt.Sprintf(`%14s  %s`, ``, `<ansi fg="red-bold">CURSED!</ansi>`))
		}

		if damageBonus > 0 {
			user.SendText(fmt.Sprintf(`%14s: +%d`, `Damage Bonus`, damageBonus))
		}
		if defenseBonus > 0 {
			user.SendText(fmt.Sprintf(`%14s: +%d`, `Armor`, defenseBonus))
		}
		if len(statBonus) > 0 {
			for statName, statBonusAmt := range statBonus {
				statName = strings.Title(statName)
				user.SendText(fmt.Sprintf(`%14s: +%d`, statName, statBonusAmt))
			}
		}

	}

	return true, nil
}
