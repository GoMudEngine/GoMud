package achievements

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
)

func charWith(fn func(*characters.Character)) *characters.Character {
	c := &characters.Character{}
	fn(c)
	return c
}

func TestEvaluate_MobKills(t *testing.T) {
	c := charWith(func(c *characters.Character) { c.KD.TotalKills = 100 })
	if !Evaluate(Trigger{Type: "mob_kills", Threshold: 100}, c, 0) {
		t.Error("100 kills should meet threshold 100")
	}
	if Evaluate(Trigger{Type: "mob_kills", Threshold: 101}, c, 0) {
		t.Error("100 kills should not meet threshold 101")
	}
}

func TestEvaluate_GoldTotal(t *testing.T) {
	c := charWith(func(c *characters.Character) { c.Gold = 600; c.Bank = 500 })
	if !Evaluate(Trigger{Type: "gold_total", Threshold: 1000}, c, 0) {
		t.Error("600 purse + 500 bank = 1100 should meet 1000")
	}
}

func TestEvaluate_StatReachedAny(t *testing.T) {
	c := &characters.Character{}
	c.Stats.Strength.ValueAdj = 155
	if !Evaluate(Trigger{Type: "stat_reached", Stat: "any", Threshold: 150}, c, 0) {
		t.Error("a 155 stat should satisfy any>=150")
	}
	if Evaluate(Trigger{Type: "stat_reached", Stat: "dexterity", Threshold: 150}, c, 0) {
		t.Error("dexterity is 0; should not satisfy 150")
	}
	if !Evaluate(Trigger{Type: "stat_reached", Stat: "strength", Threshold: 150}, c, 0) {
		t.Error("strength 155 should satisfy strength>=150")
	}
}

func TestEvaluate_MutationCount(t *testing.T) {
	c := charWith(func(c *characters.Character) { c.Mutations = map[string]int{"a": 1, "b": 2} })
	if !Evaluate(Trigger{Type: "mutation_count", Threshold: 2}, c, 0) {
		t.Error("2 mutations should meet threshold 2")
	}
}

func TestEvaluate_QuestsCompleted(t *testing.T) {
	c := charWith(func(c *characters.Character) {
		c.QuestProgress = map[int]string{10: "end", 11: "start", 12: "end"}
	})
	if !Evaluate(Trigger{Type: "quests_completed", Threshold: 2}, c, 0) {
		t.Error("two 'end' steps should meet quests_completed 2")
	}
	if Evaluate(Trigger{Type: "quests_completed", Threshold: 3}, c, 0) {
		t.Error("only two completed; should not meet 3")
	}
}

func TestEvaluate_AchievementPoints(t *testing.T) {
	c := &characters.Character{}
	if !Evaluate(Trigger{Type: "achievement_points", Threshold: 50}, c, 60) {
		t.Error("60 earned points should meet 50")
	}
	if Evaluate(Trigger{Type: "achievement_points", Threshold: 50}, c, 40) {
		t.Error("40 earned points should not meet 50")
	}
}

func TestProgress(t *testing.T) {
	c := charWith(func(c *characters.Character) { c.KD.TotalKills = 42 })
	cur, tgt, numeric := Progress(Trigger{Type: "mob_kills", Threshold: 100}, c)
	if !numeric || cur != 42 || tgt != 100 {
		t.Errorf("mob_kills progress = %d/%d numeric=%v, want 42/100 true", cur, tgt, numeric)
	}
	if _, _, numeric := Progress(Trigger{Type: "quest_completed", Token: "10-end"}, c); numeric {
		t.Error("quest_completed should not report numeric progress")
	}
	if _, _, numeric := Progress(Trigger{Type: "item_rarity", Threshold: 82}, c); numeric {
		t.Error("item_rarity should not report numeric progress")
	}
}

func TestEvaluate_UnknownType(t *testing.T) {
	if Evaluate(Trigger{Type: "nonsense", Threshold: 1}, &characters.Character{}, 0) {
		t.Error("unknown trigger type should never satisfy")
	}
}
