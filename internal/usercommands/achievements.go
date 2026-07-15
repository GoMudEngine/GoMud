package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/achievements"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// achievementCategories is the display order.
var achievementCategories = []string{
	achievements.CategoryCombat,
	achievements.CategoryExploration,
	achievements.CategoryWealth,
	achievements.CategoryProgression,
	achievements.CategoryQuests,
}

// achievementProgressHint renders a locked achievement's progress toward its goal.
func achievementProgressHint(d achievements.Definition, c *characters.Character) string {
	if cur, tgt, numeric := achievements.Progress(d.Trigger, c); numeric {
		return fmt.Sprintf("%d/%d", cur, tgt)
	}
	return "not yet"
}

// Achievements lists the caller's accolades grouped by category, with a progress
// hint for locked ones and a running total.
func Achievements(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	filter := strings.ToLower(strings.TrimSpace(rest))
	defs := achievements.All()
	char := user.Character

	if len(defs) == 0 {
		user.SendText(messaging.CategorySystem, `There are no accolades to earn yet.`)
		return true, nil
	}

	earnedCount := 0
	for _, d := range defs {
		if char.HasAchievement(d.Id) {
			earnedCount++
		}
	}
	earnedPoints := achievements.PointsFor(char.Achievements)

	user.SendText(messaging.CategorySystem,
		fmt.Sprintf(`<ansi fg="yellow-bold">Achievements</ansi> — earned <ansi fg="white-bold">%d/%d</ansi>, <ansi fg="yellow">%d points</ansi>`,
			earnedCount, len(defs), earnedPoints))

	for _, cat := range achievementCategories {
		if filter != "" && filter != cat {
			continue
		}

		// Collect this category's defs.
		var inCat []achievements.Definition
		for _, d := range defs {
			if d.Category == cat {
				inCat = append(inCat, d)
			}
		}
		if len(inCat) == 0 {
			continue
		}

		user.SendText(messaging.CategorySystem, ``)
		user.SendText(messaging.CategorySystem,
			fmt.Sprintf(`<ansi fg="cyan-bold">%s%s</ansi>`, strings.ToUpper(cat[:1]), cat[1:]))

		for _, d := range inCat {
			if char.HasAchievement(d.Id) {
				user.SendText(messaging.CategorySystem,
					fmt.Sprintf(`  <ansi fg="green">[x] %s</ansi> <ansi fg="yellow">(%d)</ansi> — %s`, d.Name, d.Points, d.Description))
			} else {
				user.SendText(messaging.CategorySystem,
					fmt.Sprintf(`  <ansi fg="black-bold">[ ] %s (%d) — %s</ansi>  <ansi fg="white">(%s)</ansi>`,
						d.Name, d.Points, d.Description, achievementProgressHint(d, char)))
			}
		}
	}

	return true, nil
}
