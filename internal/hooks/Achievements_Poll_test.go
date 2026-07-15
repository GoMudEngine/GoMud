package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/achievements"
	"github.com/GoMudEngine/GoMud/internal/characters"
)

func TestNewlyEarnedAchievements(t *testing.T) {
	defs := []achievements.Definition{
		{Id: "kills-1", Category: "combat", Points: 5, Trigger: achievements.Trigger{Type: "mob_kills", Threshold: 1}},
		{Id: "kills-100", Category: "combat", Points: 10, Trigger: achievements.Trigger{Type: "mob_kills", Threshold: 100}},
	}
	c := &characters.Character{}
	c.KD.TotalKills = 5
	c.GrantAchievement("kills-1", 1) // already earned

	earned := newlyEarnedAchievements(defs, c, 10)
	if len(earned) != 0 {
		t.Fatalf("kills-1 already earned, kills-100 not met at 5 kills; want 0 new, got %d", len(earned))
	}

	c.KD.TotalKills = 100
	earned = newlyEarnedAchievements(defs, c, 20)
	if len(earned) != 1 || earned[0].Id != "kills-100" {
		t.Fatalf("want [kills-100], got %+v", earned)
	}
}
