package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Experience(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) > 0 && args[0] == `chart` {

		args = args[1:]

		startLevel := 1
		endLevel := 25

		chartSpecies := user.Character.SpeciesId

		// xp chart elf 50
		if len(args) > 1 {

			if lvl, err := strconv.Atoi(args[len(args)-1]); err == nil {
				endLevel = lvl
				args = args[:len(args)-1]
			} else if lvl, err := strconv.Atoi(args[0]); err == nil {
				endLevel = lvl
				args = args[1:]
			}

			speciesName := strings.Join(args, ` `)
			if speciesInfo, found := species.FindSpecies(speciesName); found {
				chartSpecies = speciesInfo.SpeciesId
			}
		} else if len(args) == 1 {

			if lvl, err := strconv.Atoi(args[0]); err == nil {
				endLevel = lvl
			} else {
				if speciesInfo, found := species.FindSpecies(args[0]); found {
					chartSpecies = speciesInfo.SpeciesId
				}
			}

		}

		if endLevel < 2 {
			endLevel = 2
		}
		if endLevel > 200 {
			endLevel = 200
		}
		if endLevel-startLevel > 25 {
			startLevel = endLevel - 25
		}
		if startLevel < 1 {
			startLevel = 1
		}

		mockChar := characters.New()
		mockChar.SpeciesId = chartSpecies
		mockChar.Validate()

		headers := []string{`Level`, `Experience`, `Str`, `Spd`, `Smt`, `Vit`, `Mys`, `Per`, `ALL`}
		rows := [][]string{}

		formatting := []string{
			`<ansi fg="white-bold">%s</ansi>`,
			`<ansi fg="red">%s</ansi>`,
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="white">%s</ansi>`,
		}

		zeroStr := ``

		stats := []string{`str`, `dex`, `per`, `vit`, `wil`, `cha`}

		oldG := map[string]int{
			`str`: mockChar.Stats.Strength.GainsForLevel(startLevel - 1),
			`dex`: mockChar.Stats.Dexterity.GainsForLevel(startLevel - 1),
			`per`: mockChar.Stats.Perception.GainsForLevel(startLevel - 1),
			`vit`: mockChar.Stats.Vitality.GainsForLevel(startLevel - 1),
			`wil`: mockChar.Stats.Willpower.GainsForLevel(startLevel - 1),
			`cha`: mockChar.Stats.Charisma.GainsForLevel(startLevel - 1),
		}

		newG := map[string]int{}
		totalG := map[string]int{}
		for _, stat := range stats {
			totalG[stat] = oldG[stat]
		}

		for i := startLevel; i <= endLevel; i++ {

			newG = map[string]int{
				`str`: mockChar.Stats.Strength.GainsForLevel(i),
				`dex`: mockChar.Stats.Dexterity.GainsForLevel(i),
				`per`: mockChar.Stats.Perception.GainsForLevel(i),
				`vit`: mockChar.Stats.Vitality.GainsForLevel(i),
				`wil`: mockChar.Stats.Willpower.GainsForLevel(i),
				`cha`: mockChar.Stats.Charisma.GainsForLevel(i),
			}

			tnlXP := mockChar.XPTL(i) - mockChar.XPTL(i-1)

			if i == 1 {
				tnlXP = 0
			}

			if i > 1 {

				row := []string{fmt.Sprintf(`%d`, i), fmt.Sprintf(`%d`, tnlXP)}
				gainStr := zeroStr
				all := 0
				for _, stat := range stats {
					gain := newG[stat] - oldG[stat]
					all += gain
					totalG[stat] += gain

					if gain > 0 {
						gainStr = fmt.Sprintf(`%d`, gain)
					} else {
						gainStr = zeroStr
					}

					row = append(row, gainStr)
				}

				if all > 0 {
					row = append(row, fmt.Sprintf(`%d`, all))
				} else {
					row = append(row, zeroStr)
				}

				rows = append(rows, row)
			}

			for _, stat := range stats {
				oldG[stat] = newG[stat]
			}
		}

		row := []string{`Total`, fmt.Sprintf(`%d`, mockChar.XPTL(endLevel)-1500)}
		all := 0
		for _, stat := range stats {
			gain := totalG[stat]
			all += gain
			gainStr := ``
			if gain > 0 {
				gainStr = fmt.Sprintf(`%d`, gain)
			}

			row = append(row, gainStr)
		}

		if all > 0 {
			row = append(row, fmt.Sprintf(`%d`, all))
		} else {
			row = append(row, zeroStr)
		}

		rows = append(rows, row)

		raceInfo := species.GetSpecies(mockChar.SpeciesId)
		searchResultsTable := templates.GetTable(fmt.Sprintf(`Experience Chart for %s`, raceInfo.Name), headers, rows, formatting)
		tplTxt, _ := templates.Process("tables/generic", searchResultsTable, user.UserId)
		user.SendText(tplTxt)

		return true, nil
	}

	realXPNow, realXPTNL := user.Character.XPTNLActual()
	xpInfo := map[string]int{
		"Level": user.Character.Level,
		"Exp":   realXPNow,
		"Tnl":   realXPTNL,
		"Tp":    user.Character.TrainingPoints,
		"Sp":    user.Character.StatPoints,
	}

	tplTxt, _ := templates.Process("character/experience", xpInfo, user.UserId)
	user.SendText(tplTxt)

	return true, nil
}
