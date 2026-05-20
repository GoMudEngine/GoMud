package usercommands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
* Role Permissions:
* combatstats 				(All)
 */
func CombatStats(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		infoOutput, _ := templates.Process("admincommands/help/command.combatstats", nil, user.UserId)
		user.SendText(messaging.CategorySystem, infoOutput)
		return true, nil
	}

	args := util.SplitButRespectQuotes(rest)

	switch strings.ToLower(args[0]) {
	case "summary":
		return combatStats_Summary(args[1:], user)
	case "types":
		return combatStats_Types(user)
	case "matchups":
		return combatStats_Matchups(user)
	case "defense":
		return combatStats_Defense(user)
	case "position":
		return combatStats_Position(user)
	case "reset":
		return combatStats_Reset(user)
	case "export":
		return combatStats_Export(user)
	default:
		infoOutput, _ := templates.Process("admincommands/help/command.combatstats", nil, user.UserId)
		user.SendText(messaging.CategorySystem, infoOutput)
	}

	return true, nil
}

func combatStats_Summary(args []string, user *users.UserRecord) (bool, error) {

	var s combat.AnalyticsSummary
	filterLabel := ""

	if len(args) > 0 {
		filterLabel = strings.ToLower(args[0])
		s = combat.GetFilteredSummaryByAttackType(filterLabel)
	} else {
		s = combat.GetSummary()
	}

	if s.TotalEvents == 0 {
		if filterLabel != "" {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(
				`No combat events found for attack type "%s".`, filterLabel))
		} else {
			user.SendText(messaging.CategorySystem, `No combat events in the buffer.`)
		}
		return true, nil
	}

	hitRate := float64(0)
	if s.TotalEvents > 0 {
		hitRate = float64(s.Hits) / float64(s.TotalEvents) * 100
	}

	avgDmg := float64(0)
	if s.Hits > 0 {
		avgDmg = float64(s.TotalDamage) / float64(s.Hits)
	}

	title := fmt.Sprintf(`Combat Analytics Dashboard (%s events)`,
		formatNumber(s.TotalEvents))
	if filterLabel != "" {
		title = fmt.Sprintf(`Combat Analytics: %s (%s events)`,
			filterLabel, formatNumber(s.TotalEvents))
	}

	headers := []string{"Metric", "Value"}
	formatting := []string{
		`<ansi fg="yellow-bold">%s</ansi>`,
		`<ansi fg="cyan-bold">%s</ansi>`,
	}

	rows := [][]string{
		{"Total Events", formatNumber(s.TotalEvents)},
		{"Hits", formatNumber(s.Hits)},
		{"Misses", formatNumber(s.Misses)},
		{"Hit Rate", fmt.Sprintf("%.1f%%", hitRate)},
		{"Crits", formatNumber(s.Crits)},
		{"Fumbles", formatNumber(s.Fumbles)},
		{"Backfires", formatNumber(s.Backfires)},
		{"Fizzles", formatNumber(s.Fizzles)},
		{"Total Damage", formatNumber(s.TotalDamage)},
		{"Avg Dmg/Hit", fmt.Sprintf("%.1f", avgDmg)},
		{"Avg Atk Z-Score", fmt.Sprintf("%.3f", s.AvgAttackZScore)},
		{"Avg Def Z-Score", fmt.Sprintf("%.3f", s.AvgDefenseZScore)},
		{"Rounds", fmt.Sprintf("%d-%d", s.EarliestRound, s.LatestRound)},
	}

	tblData := templates.GetTable(title, headers, rows, formatting)
	tplTxt, _ := templates.Process("tables/generic", tblData, user.UserId)
	user.SendText(messaging.CategorySystem, tplTxt)

	return true, nil
}

func combatStats_Types(user *users.UserRecord) (bool, error) {

	types := combat.GetAttackTypes()
	if len(types) == 0 {
		user.SendText(messaging.CategorySystem, `No combat events in the buffer.`)
		return true, nil
	}

	// Sort by count descending
	type typeCount struct {
		name  string
		count int
	}
	sorted := make([]typeCount, 0, len(types))
	for name, ct := range types {
		sorted = append(sorted, typeCount{name, ct})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	headers := []string{"Attack Type", "Events"}
	formatting := []string{
		`<ansi fg="yellow-bold">%s</ansi>`,
		`<ansi fg="cyan-bold">%s</ansi>`,
	}

	rows := make([][]string, 0, len(sorted))
	for _, tc := range sorted {
		rows = append(rows, []string{tc.name, formatNumber(tc.count)})
	}

	tblData := templates.GetTable(
		fmt.Sprintf(`Attack Types (%d total)`, combat.GetBufferLen()),
		headers, rows, formatting)
	tplTxt, _ := templates.Process("tables/generic", tblData, user.UserId)
	user.SendText(messaging.CategorySystem, tplTxt)

	return true, nil
}

func combatStats_Matchups(user *users.UserRecord) (bool, error) {

	s := combat.GetSummary()
	if s.TotalEvents == 0 {
		user.SendText(messaging.CategorySystem, `No combat events in the buffer.`)
		return true, nil
	}

	headers := []string{"Matchup", "Events", "% of Total"}
	formatting := []string{
		`<ansi fg="yellow-bold">%s</ansi>`,
		`<ansi fg="cyan-bold">%s</ansi>`,
		`<ansi fg="cyan-bold">%s</ansi>`,
	}

	total := float64(s.TotalEvents)
	rows := [][]string{
		{"Player vs Mob", formatNumber(s.PvMEvents),
			fmt.Sprintf("%.1f%%", float64(s.PvMEvents)/total*100)},
		{"Mob vs Player", formatNumber(s.MvPEvents),
			fmt.Sprintf("%.1f%%", float64(s.MvPEvents)/total*100)},
		{"Player vs Player", formatNumber(s.PvPEvents),
			fmt.Sprintf("%.1f%%", float64(s.PvPEvents)/total*100)},
		{"Mob vs Mob", formatNumber(s.MvMEvents),
			fmt.Sprintf("%.1f%%", float64(s.MvMEvents)/total*100)},
	}

	tblData := templates.GetTable(
		fmt.Sprintf(`Combat Matchups (%s events)`, formatNumber(s.TotalEvents)),
		headers, rows, formatting)
	tplTxt, _ := templates.Process("tables/generic", tblData, user.UserId)
	user.SendText(messaging.CategorySystem, tplTxt)

	return true, nil
}

func combatStats_Defense(user *users.UserRecord) (bool, error) {

	s := combat.GetSummary()
	if s.TotalEvents == 0 {
		user.SendText(messaging.CategorySystem, `No combat events in the buffer.`)
		return true, nil
	}

	totalDefenses := s.DodgeSuccesses + s.ParrySuccesses + s.BlockSuccesses
	headers := []string{"Defense", "Successes", "% of Defenses"}
	formatting := []string{
		`<ansi fg="yellow-bold">%s</ansi>`,
		`<ansi fg="cyan-bold">%s</ansi>`,
		`<ansi fg="cyan-bold">%s</ansi>`,
	}

	pct := func(n int) string {
		if totalDefenses == 0 {
			return "0.0%"
		}
		return fmt.Sprintf("%.1f%%", float64(n)/float64(totalDefenses)*100)
	}

	rows := [][]string{
		{"Dodge", formatNumber(s.DodgeSuccesses), pct(s.DodgeSuccesses)},
		{"Parry", formatNumber(s.ParrySuccesses), pct(s.ParrySuccesses)},
		{"Block", formatNumber(s.BlockSuccesses), pct(s.BlockSuccesses)},
		{"Total", formatNumber(totalDefenses), "100.0%"},
	}

	tblData := templates.GetTable(
		fmt.Sprintf(`Defense Breakdown (%s events)`, formatNumber(s.TotalEvents)),
		headers, rows, formatting)
	tplTxt, _ := templates.Process("tables/generic", tblData, user.UserId)
	user.SendText(messaging.CategorySystem, tplTxt)

	return true, nil
}

func combatStats_Position(user *users.UserRecord) (bool, error) {

	s := combat.GetSummary()
	if s.TotalEvents == 0 {
		user.SendText(messaging.CategorySystem, `No combat events in the buffer.`)
		return true, nil
	}

	headers := []string{"Position / Status", "Hit Rate"}
	formatting := []string{
		`<ansi fg="yellow-bold">%s</ansi>`,
		`<ansi fg="cyan-bold">%s</ansi>`,
	}

	fmtRate := func(r float64) string {
		return fmt.Sprintf("%.1f%%", r*100)
	}

	rows := [][]string{
		{"Target Standing", fmtRate(s.HitRateTargetStanding)},
		{"Target Prone", fmtRate(s.HitRateTargetProne)},
		{"Target Clinched", fmtRate(s.HitRateTargetClinched)},
		{"Target Grounded", fmtRate(s.HitRateTargetGrounded)},
		{"", ""},
		{"Grapple Controller", fmtRate(s.HitRateGrappleController)},
		{"Non-Controller", fmtRate(s.HitRateNonGrappleController)},
	}

	tblData := templates.GetTable(
		fmt.Sprintf(`Position Hit Rates (%s events)`, formatNumber(s.TotalEvents)),
		headers, rows, formatting)
	tplTxt, _ := templates.Process("tables/generic", tblData, user.UserId)
	user.SendText(messaging.CategorySystem, tplTxt)

	return true, nil
}

func combatStats_Reset(user *users.UserRecord) (bool, error) {

	ct := combat.ResetBuffer()
	user.SendText(messaging.CategorySystem, fmt.Sprintf(
		`Combat analytics buffer cleared. %s events removed.`,
		formatNumber(ct)))

	return true, nil
}

func combatStats_Export(user *users.UserRecord) (bool, error) {

	bufLen := combat.GetBufferLen()
	if bufLen == 0 {
		user.SendText(messaging.CategorySystem, `No combat events to export.`)
		return true, nil
	}

	combat.ExportNow()
	user.SendText(messaging.CategorySystem, fmt.Sprintf(
		`Flushed %s events to analytics log.`,
		formatNumber(bufLen)))

	return true, nil
}

// formatNumber adds comma separators to an integer for display.
func formatNumber(n int) string {
	if n < 0 {
		return "-" + formatNumber(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result strings.Builder
	remainder := len(s) % 3
	if remainder > 0 {
		result.WriteString(s[:remainder])
	}
	for i := remainder; i < len(s); i += 3 {
		if result.Len() > 0 {
			result.WriteByte(',')
		}
		result.WriteString(s[i : i+3])
	}
	return result.String()
}
