package usercommands

import (
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// spellEntry holds a display row together with its sort key.
type spellEntry struct {
	baseFolds  int
	row        []string
	formatRow  []string
}

func Spells(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	headers := []string{`SpellId`, `Name`, `Target`, `Cost`, `Cast time`, `Familiarity`, `Reliability`}

	helpfulEntries := []spellEntry{}
	harmfulEntries := []spellEntry{}
	neutralEntries := []spellEntry{}

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
				target,
				costStr,
				combat.GetWaitRoundsDescription(sp.WaitRounds),
				combat.GetCastCountDescription(casts),
				combat.GetSuccessChanceDescription(user.Character.GetBaseCastSuccessChance(sp.SpellId)),
			}

			entry := spellEntry{baseFolds: sp.BaseFolds, row: row, formatRow: formatRow}

			if helpOrHarm == `helpful` {
				helpfulEntries = append(helpfulEntries, entry)
			} else if helpOrHarm == `harmful` {
				harmfulEntries = append(harmfulEntries, entry)
			} else {
				neutralEntries = append(neutralEntries, entry)
			}

		}

	}

	// Sort each group ascending by BaseFolds (simplest spells first).
	sort.Slice(helpfulEntries, func(i, j int) bool { return helpfulEntries[i].baseFolds < helpfulEntries[j].baseFolds })
	sort.Slice(harmfulEntries, func(i, j int) bool { return harmfulEntries[i].baseFolds < harmfulEntries[j].baseFolds })
	sort.Slice(neutralEntries, func(i, j int) bool { return neutralEntries[i].baseFolds < neutralEntries[j].baseFolds })

	for _, e := range helpfulEntries {
		rowFormatting = append(rowFormatting, e.formatRow)
		rows = append(rows, e.row)
	}
	for _, e := range harmfulEntries {
		rowFormatting = append(rowFormatting, e.formatRow)
		rows = append(rows, e.row)
	}
	for _, e := range neutralEntries {
		rowFormatting = append(rowFormatting, e.formatRow)
		rows = append(rows, e.row)
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
