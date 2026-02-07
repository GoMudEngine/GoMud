package usercommands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Experience now shows skill progression info instead of XP/Level.
// Stage 3.5: Progression is 100% skill-based.
func Experience(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) > 0 && args[0] == `skills` {
		// Show detailed skill use counts
		type SkillDetail struct {
			Name  string
			Rank  int
			Uses  int
		}

		allSkills := skills.GetAllSkillNames()
		details := []SkillDetail{}
		for _, sk := range allSkills {
			skStr := string(sk)
			rank := user.Character.GetSkillLevel(sk)
			uses := user.Character.GetSkillUseCount(skStr)
			details = append(details, SkillDetail{Name: skStr, Rank: rank, Uses: uses})
		}

		headers := []string{`Skill`, `Rank`, `Uses`}
		rows := [][]string{}
		formatting := []string{
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="white-bold">%s</ansi>`,
			`<ansi fg="cyan">%s</ansi>`,
		}

		for _, d := range details {
			rankStr := fmt.Sprintf(`%d`, d.Rank)
			rows = append(rows, []string{d.Name, rankStr, fmt.Sprintf(`%d`, d.Uses)})
		}

		searchResultsTable := templates.GetTable(`Skill Progression`, headers, rows, formatting)
		tplTxt, _ := templates.Process("tables/generic", searchResultsTable, user.UserId)
		user.SendText(tplTxt)

		return true, nil
	}

	if len(args) > 0 && args[0] == `stats` {
		// Show stat use counts
		statNames := []string{"strength", "dexterity", "perception", "vitality", "willpower", "charisma"}

		headers := []string{`Stat`, `Uses`}
		rows := [][]string{}
		formatting := []string{
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="cyan">%s</ansi>`,
		}

		for _, s := range statNames {
			uses := user.Character.GetStatUseCount(s)
			rows = append(rows, []string{s, fmt.Sprintf(`%d`, uses)})
		}

		searchResultsTable := templates.GetTable(`Stat Usage`, headers, rows, formatting)
		tplTxt, _ := templates.Process("tables/generic", searchResultsTable, user.UserId)
		user.SendText(tplTxt)

		return true, nil
	}

	// Default: show skill summary
	allRanks := user.Character.GetAllSkillRanks()
	totalRanks := user.Character.GetTotalSkillRanks()

	// Sort skill names
	skillNames := make([]string, 0, len(allRanks))
	for name := range allRanks {
		skillNames = append(skillNames, name)
	}
	sort.Strings(skillNames)

	user.SendText(``)
	user.SendText(fmt.Sprintf(`<ansi fg="white-bold">  Skill Progression</ansi> <ansi fg="8">(total ranks: %d)</ansi>`, totalRanks))
	user.SendText(`<ansi fg="8">  ───────────────────────────────────</ansi>`)

	for _, name := range skillNames {
		rank := allRanks[name]
		rankDisplay := fmt.Sprintf(`Rank %d`, rank)
		color := "cyan"
		if rank >= 10 {
			color = "yellow-bold"
		} else if rank >= 5 {
			color = "green"
		}
		user.SendText(fmt.Sprintf(`  <ansi fg="yellow">%-15s</ansi> <ansi fg="%s">[%s]</ansi>`, name, color, rankDisplay))
	}

	user.SendText(``)
	user.SendText(`<ansi fg="8">  Type </ansi><ansi fg="command">experience skills</ansi><ansi fg="8"> for detailed use counts.</ansi>`)
	user.SendText(`<ansi fg="8">  Type </ansi><ansi fg="command">experience stats</ansi><ansi fg="8"> for stat usage.</ansi>`)
	user.SendText(``)

	return true, nil
}
