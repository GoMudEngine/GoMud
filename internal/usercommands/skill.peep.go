package usercommands

import (
	"errors"
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
Peep Skill
Level 1 - Always visibly see the health % of an NPC
Level 2 - Reveals detailed stats of a player or mob.
Level 3 - Reveals detailed stats of the player or mob, plus equipment and items
Level 4 - eveals detailed stats of the player or mob, plus equipment and items, and tells you the % chance of dropping items.
*/
func Peep(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Peep)

	if skillLevel == 0 {
		user.SendText("You don't know how to peep.")
		return true, errors.New(`you don't know how to peep`)
	}

	if len(rest) == 0 {
		user.SendText("Type `help peep` for more information on the peep skill.")
		return true, nil
	}

	if skillLevel < 2 {
		user.SendText("At level 1, peep is a passive skill.")
		user.SendText("Type `help peep` for more information on the peep skill.")
		return true, errors.New(`at level 1, peep is a passive skill`)
	}

	if !user.Character.TryCooldown(skills.Peep.String(), "1 round") {
		user.SendText(
			`You're using that skill just a little too fast.`,
		)
		return true, errors.New(`you're doing that too often`)
	}

	// valid peep targets are: mobs, players
	playerId, mobId := room.FindByName(rest)

	if playerId > 0 || mobId > 0 {

		// Fire an event that a skill has been used
		events.AddToQueue(events.SkillUsed{user.UserId, skills.Peep, ``})

		statusTxt := ""
		invTxt := ""
		dropTxt := ""

		if playerId > 0 {

			u := *users.GetByUserId(playerId)
			targetName := u.Character.GetPlayerName(user.UserId).String()

			if skillLevel >= 2 {
				statusTxt, _ = templates.Process("character/status-lite", u.Character, user.UserId)
			}

			if skillLevel >= 3 {

				itemNames := []string{}
				itemNamesFormatted := []string{}

				for _, item := range u.Character.Items {

					iName := item.DisplayName()
					iNameFormatted := fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, iName)

					iSpec := item.GetSpec()
					if iSpec.Subtype == items.Drinkable || iSpec.Subtype == items.Edible {
						if iSpec.Uses > 0 { // Does the spec indicate a number of uses?
							iName = fmt.Sprintf(`%s (%d)`, iName, item.Uses)                                               // Display uses left
							iNameFormatted = fmt.Sprintf(`%s <ansi fg="uses-left">(%d)</ansi>`, iNameFormatted, item.Uses) // Display uses left
						}
					}
					itemNames = append(itemNames, iName)
					itemNamesFormatted = append(itemNamesFormatted, iNameFormatted)
				}

				raceInfo := species.GetSpecies(u.Character.SpeciesId)

				diceRoll := raceInfo.Damage.DiceRoll
				if u.Character.Equipment.Weapon.ItemId != 0 {
					iSpec := u.Character.Equipment.Weapon.GetSpec()
					diceRoll = iSpec.Damage.DiceRoll
				}

				invData := map[string]any{
					`Equipment`:          &u.Character.Equipment,
					`ItemNames`:          itemNames,
					`ItemNamesFormatted`: itemNamesFormatted,
					`AttackDamage`:       diceRoll,
					`RaceInfo`:           raceInfo,
					`Count`:              fmt.Sprintf(`(%d/%.0f lbs)`, len(u.Character.Items), u.Character.CarryCapacity()),
				}

				invTxt, _ = templates.Process("character/inventory", invData, user.UserId)
			}

			if skillLevel >= 4 {
				dropTxt = fmt.Sprintf(` <ansi fg="username">%s</ansi> has a 100%% chance of dropping their equipment if killed.%s%s`, targetName, term.CRLFStr, term.CRLFStr)
			}

			if targetUser := users.GetByUserId(playerId); targetUser != nil {
				targetUser.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> is peeping at you.`, user.Character.Name),
				)
			}

			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> is peeping at <ansi fg="username">%s</ansi>.`, user.Character.Name, u.Character.Name),
				user.UserId,
				u.UserId)

		} else if mobId > 0 {

			m := mobs.GetInstance(mobId)
			targetName := m.Character.GetMobName(user.UserId).String()

			if skillLevel >= 2 {
				statusTxt, _ = templates.Process("character/status-lite", &m.Character, user.UserId)
			}

			if skillLevel >= 3 {

				itemNames := []string{}
				itemNamesFormatted := []string{}

				for _, item := range m.Character.Items {

					iName := item.DisplayName()
					iNameFormatted := fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, iName)

					iSpec := item.GetSpec()
					if iSpec.Subtype == items.Drinkable || iSpec.Subtype == items.Edible {
						if iSpec.Uses > 0 { // Does the spec indicate a number of uses?
							iName = fmt.Sprintf(`%s (%d)`, iName, item.Uses)                                               // Display uses left
							iNameFormatted = fmt.Sprintf(`%s <ansi fg="uses-left">(%d)</ansi>`, iNameFormatted, item.Uses) // Display uses left
						}
					}
					itemNames = append(itemNames, iName)
					itemNamesFormatted = append(itemNamesFormatted, iNameFormatted)
				}

				raceInfo := species.GetSpecies(m.Character.SpeciesId)

				diceRoll := raceInfo.Damage.DiceRoll
				if m.Character.Equipment.Weapon.ItemId != 0 {
					iSpec := m.Character.Equipment.Weapon.GetSpec()
					diceRoll = iSpec.Damage.DiceRoll
				}

				invData := map[string]any{
					`Equipment`:          &m.Character.Equipment,
					`ItemNames`:          itemNames,
					`ItemNamesFormatted`: itemNamesFormatted,
					`AttackDamage`:       diceRoll,
					`RaceInfo`:           raceInfo,
				}

				invTxt, _ = templates.Process("character/inventory", invData, user.UserId)

			}

			if skillLevel >= 4 {
				dropTxt = fmt.Sprintf(`<ansi fg="mobname">%s</ansi> has a %d%% chance of dropping their equipment if killed.%s%s`, targetName, m.ItemDropChance, term.CRLFStr, term.CRLFStr)
			}

			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> is peeping at %s.`, user.Character.Name, targetName),
				user.UserId,
			)

		}

		if statusTxt != `` {
			user.SendText(statusTxt)
		}
		if invTxt != `` {
			user.SendText(invTxt)
		}
		if dropTxt != `` {
			user.SendText(dropTxt)
		}

		return true, nil

	}

	user.SendText("You don't see that here.")

	return true, errors.New(`you don't see that here`)

}
