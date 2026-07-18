package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/achievements"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// newlyEarnedAchievements returns the definitions the character now meets but
// hasn't unlocked yet. earnedPoints is the character's current total (for the
// achievement_points meta trigger). Pure.
func newlyEarnedAchievements(defs []achievements.Definition, c *characters.Character, earnedPoints int) []achievements.Definition {
	var earned []achievements.Definition
	for _, d := range defs {
		if c.HasAchievement(d.Id) {
			continue
		}
		if achievements.Evaluate(d.Trigger, c, earnedPoints) {
			earned = append(earned, d)
		}
	}
	return earned
}

// CheckAchievements polls online players and grants newly-earned achievements.
// Gated to a modest interval (AchievementPollRounds) to bound cost. Announcements
// are private and ANSI-styled (no emoji — the client has a glyph gap).
func CheckAchievements(e events.Event) events.ListenerReturn {
	interval := uint64(configs.GetBalanceConfig().AchievementPollRounds)
	if interval == 0 {
		interval = 10
	}
	round := util.GetRoundCount()
	if round%interval != 0 {
		return events.Continue
	}

	defs := achievements.All()
	if len(defs) == 0 {
		return events.Continue
	}

	for _, u := range users.GetAllActiveUsers() {
		// Match the leaderboard exclusion: no admins, no AI-flagged accounts.
		if u.Role == users.RoleAdmin || u.IsAI {
			continue
		}
		earnedPoints := achievements.PointsFor(u.Character.Achievements)
		earned := newlyEarnedAchievements(defs, u.Character, earnedPoints)
		if len(earned) == 0 {
			continue
		}
		// First-ever achievement: the banner is meaningless without context (the
		// player doesn't yet know what achievements are — 2026-07-18 feedback).
		// Prime them once, before the banner, with a one-line explainer + pointer.
		if len(u.Character.Achievements) == 0 {
			u.SendText(messaging.CategorySystem,
				`<ansi fg="white">(You just earned your first </ansi><ansi fg="yellow-bold">achievement</ansi><ansi fg="white"> -- a milestone you unlock simply by playing. Type </ansi><ansi fg="command">achievements</ansi><ansi fg="white"> anytime to see the ones you've earned and what else you can chase.)</ansi>`)
		}
		for _, d := range earned {
			u.Character.GrantAchievement(d.Id, round)
			u.SendText(messaging.CategorySystem,
				fmt.Sprintf(`<ansi fg="yellow-bold">*** Achievement unlocked: %s ***</ansi>`, d.Name))
			u.SendText(messaging.CategorySystem,
				fmt.Sprintf(`<ansi fg="white">%s  (+%d points)</ansi>`, d.Description, d.Points))
		}
		users.SaveUser(*u)
	}
	return events.Continue
}
