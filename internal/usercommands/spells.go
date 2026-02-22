package usercommands

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Spells(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	headers := []string{`SpellId`, `Name`, `Description`, `Schools`, `Target`, `Cost`, `Cast time`, `Familiarity`, `Reliability`}

	helpfulRowFormatting := [][]string{}
	helpfulRows := [][]string{}

	harmfulRowFormatting := [][]string{}
	harmfulRows := [][]string{}

	neutralRowFormatting := [][]string{}
	neutralRows := [][]string{}

	rowFormatting := [][]string{}
	rows := [][]string{}

	for spellId, casts := range user.Character.GetSpells() {

		if casts < 0 {
			continue
		}

		casts -= 1

		if sp := spells.GetSpell(spellId); sp != nil {

			helpOrHarm := strings.ToLower(sp.Type.HelpOrHarmString())

			targetColor := `spell-` + helpOrHarm
			target := sp.Type.TargetTypeString(true)

			formatRow := []string{
				`<ansi fg="yellow-bold">%s</ansi>`,
				`<ansi fg="white-bold">%s</ansi>`,
				`<ansi fg="yellow">%s</ansi>`,
				`<ansi fg="cyan">%s</ansi>`,
				`<ansi fg="` + targetColor + `">%s</ansi>`,
				`<ansi fg="magenta">%s</ansi>`,
				`<ansi fg="white">%s</ansi>`,
				`<ansi fg="red">%s</ansi>`,
				`<ansi fg="red">%s</ansi>`,
			}

			// Format cost display - show qualitative conviction cost and optionally health
			costStr := combat.GetConvictionCostDescription(sp.Cost)
			if sp.HealthCost > 0 {
				costStr += ", drains health"
			}

			row := []string{sp.SpellId,
				sp.Name,
				sp.Description,
				sp.GetSchoolsString(),
				target,
				costStr,
				combat.GetWaitRoundsDescription(sp.WaitRounds),
				combat.GetCastCountDescription(casts),
				combat.GetSuccessChanceDescription(user.Character.GetBaseCastSuccessChance(sp.SpellId)),
			}

			if helpOrHarm == `helpful` {
				helpfulRowFormatting = append(helpfulRowFormatting, formatRow)
				helpfulRows = append(helpfulRows, row)
			} else if helpOrHarm == `harmful` {
				harmfulRowFormatting = append(harmfulRowFormatting, formatRow)
				harmfulRows = append(harmfulRows, row)
			} else {
				neutralRowFormatting = append(neutralRowFormatting, formatRow)
				neutralRows = append(neutralRows, row)
			}

		}

	}

	if len(helpfulRows) > 0 {
		for i := 0; i < len(helpfulRows); i++ {
			rowFormatting = append(rowFormatting, helpfulRowFormatting[i])
			rows = append(rows, helpfulRows[i])
		}
	}

	if len(harmfulRows) > 0 {
		for i := 0; i < len(harmfulRows); i++ {
			rowFormatting = append(rowFormatting, harmfulRowFormatting[i])
			rows = append(rows, harmfulRows[i])
		}
	}

	if len(neutralRows) > 0 {
		for i := 0; i < len(neutralRows); i++ {
			rowFormatting = append(rowFormatting, neutralRowFormatting[i])
			rows = append(rows, neutralRows[i])
		}
	}

	onlineResultsTable := templates.GetTable(`Spells`, headers, rows, rowFormatting...)
	tplTxt, _ := templates.Process("tables/generic", onlineResultsTable, user.UserId)
	user.SendText(tplTxt)

	/*
		if len(neutralRows) > 0 {
			onlineResultsTable := templates.GetTable(`<ansi fg="spell-neutral">Neutral</ansi> Spells`, headers, neutralRows, neutralRowFormatting...)
			tplTxt, _ := templates.Process("tables/generic", onlineResultsTable, user.UserId)
			user.SendText( tplTxt)
		}

		if len(harmfulRows) > 0 {
			onlineResultsTable := templates.GetTable(`<ansi fg="spell-helpful">Helpful</ansi> Spells`, headers, harmfulRows, harmfulRowFormatting...)
			tplTxt, _ := templates.Process("tables/generic", onlineResultsTable, user.UserId)
			user.SendText( tplTxt)
		}

		if len(helpfulRows) > 0 {
			onlineResultsTable := templates.GetTable(`<ansi fg="spell-harmful">Harmful</ansi> Spells`, headers, helpfulRows, helpfulRowFormatting...)
			tplTxt, _ := templates.Process("tables/generic", onlineResultsTable, user.UserId)
			user.SendText( tplTxt)
		}
	*/
	return true, nil
}
