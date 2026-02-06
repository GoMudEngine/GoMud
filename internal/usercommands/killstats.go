package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Killstats(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	otherSuggestions := []string{}

	var headers []string

	tableTitle := `Kill Stats`

	rows := [][]string{}

	formatting := []string{
		`<ansi fg="mobname">%s</ansi>`,
		`<ansi fg="red">%s</ansi>`,
		`<ansi fg="230">%s</ansi>`,
	}

	totalMobKills := 0
	totalPVPKills := 0
	//totalPVPDeaths := 0

	mobKills := map[string]int{}
	raceKills := map[string]int{}
	areaKills := map[string]int{}
	charKills := map[string]int{}

	for mid, kCt := range user.Character.KD.Kills {

		if mobSpec := mobs.GetMobSpec(mobs.MobId(mid)); mobSpec != nil {

			totalMobKills += kCt

			// Populate mob kills
			mobKills[mobSpec.Character.Name] = mobKills[mobSpec.Character.Name] + kCt

			// Populate race kills
			if raceInfo := species.GetSpecies(mobSpec.Character.SpeciesId); raceInfo != nil {
				raceKills[raceInfo.Name] = raceKills[raceInfo.Name] + kCt
			}

			// Populate area kills
			areaKills[mobSpec.Zone] = areaKills[mobSpec.Zone] + kCt
		}
	}

	for userIdNameStr, killCount := range user.Character.KD.PlayerKills {
		parts := strings.Split(userIdNameStr, `:`)
		charKills[parts[1]] = killCount
		totalPVPKills++
	}

	renderStats := mobKills
	totalKills := totalMobKills
	totalDeaths := user.Character.KD.GetMobDeaths()
	kdRatio := user.Character.KD.GetMobKDRatio()

	if rest == `pvp` {

		renderStats = charKills
		totalKills = totalPVPKills
		kdRatio = user.Character.KD.GetPvpKDRatio()
		totalDeaths = user.Character.KD.GetPvpDeaths()

		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats area</ansi>`)
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats race</ansi>`)

	} else if rest == `race` || rest == `races` {

		renderStats = raceKills
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats pvp</ansi>`)
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats area</ansi>`)

	} else if rest == `zone` || rest == `zones` || rest == `area` || rest == `areas` {

		renderStats = areaKills
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats pvp</ansi>`)
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats race</ansi>`)

	} else {

		rest = `mob` // default to mob

		renderStats = mobKills
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats pvp</ansi>`)
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats area</ansi>`)
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats race</ansi>`)
	}

	headers = []string{strings.Title(rest), `Quantity`, `%`}

	for name, killCt := range renderStats {

		rows = append(rows, []string{
			name,
			fmt.Sprintf("%d", killCt),
			fmt.Sprintf("%2.f%%", float64(killCt)/float64(totalKills)*100),
		})
	}

	rows = append(rows, []string{
		``,
		``,
		``,
	})

	rows = append(rows, []string{
		`Total Kills`,
		fmt.Sprintf("%d", totalKills),
		``,
	})

	if totalDeaths == 0 {
		rows = append(rows, []string{
			`Total Deaths`,
			fmt.Sprintf("%d", totalDeaths),
			`N/A`,
		})
	} else {
		rows = append(rows, []string{
			`Total Deaths`,
			fmt.Sprintf("%d", totalDeaths),
			fmt.Sprintf("%.2f:1", kdRatio),
		})
	}

	searchResultsTable := templates.GetTable(tableTitle+` by `+strings.Title(rest), headers, rows, formatting)
	tplTxt, _ := templates.Process("tables/generic", searchResultsTable, user.UserId)
	tplTxt += fmt.Sprintf("Also try: %s\n", strings.Join(otherSuggestions, `, `))
	user.SendText(tplTxt)

	//user.SendText(fmt.Sprintf(`Also try: %s`, strings.Join(otherSuggestions, `, `)))

	return true, nil
}
