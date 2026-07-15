package achievements

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// pinnacleEquipTypes are the wearable/wieldable item types that count for the
// item_rarity ("acquired a pinnacle item") trigger — excludes components,
// materials, potions, and quest junk so a raw high-rarity reagent doesn't count.
var pinnacleEquipTypes = map[items.ItemType]bool{
	items.Weapon: true, items.Offhand: true, items.Head: true, items.Neck: true,
	items.Shoulders: true, items.Body: true, items.Back: true, items.Belt: true,
	items.Wrist: true, items.Gloves: true, items.Ring: true, items.Legs: true,
	items.Feet: true,
}

// Evaluate reports whether trigger t is satisfied by character c. earnedPoints is
// the character's current total achievement points (only used by the meta
// achievement_points trigger). Pure — no mutation, no side effects.
func Evaluate(t Trigger, c *characters.Character, earnedPoints int) bool {
	switch t.Type {
	case "mob_kills":
		return c.KD.TotalKills >= t.Threshold
	case "pvp_kills":
		return c.KD.TotalPvpKills >= t.Threshold
	case "deaths":
		return c.KD.TotalDeaths >= t.Threshold
	case "gold_total":
		return c.Gold+c.Bank >= t.Threshold
	case "mutation_count":
		return len(c.Mutations) >= t.Threshold
	case "rooms_explored":
		return roomsExplored(c) >= t.Threshold
	case "quest_completed":
		return c.HasQuest(t.Token)
	case "quests_completed":
		return completedQuestCount(c) >= t.Threshold
	case "stat_reached":
		return statValue(c, t.Stat) >= t.Threshold
	case "skill_reached":
		return skillValue(c, t.Skill) >= t.Threshold
	case "item_rarity":
		return ownsPinnacleItem(c, t.Threshold)
	case "achievement_points":
		return earnedPoints >= t.Threshold
	}
	return false
}

// Progress returns the character's current value toward a numeric trigger and its
// target, plus whether a numeric progress bar applies. numeric is false for
// triggers without a simple per-character running value (quest_completed,
// item_rarity, achievement_points) — the caller shows "not yet" for those.
func Progress(t Trigger, c *characters.Character) (current, target int, numeric bool) {
	switch t.Type {
	case "mob_kills":
		return c.KD.TotalKills, t.Threshold, true
	case "pvp_kills":
		return c.KD.TotalPvpKills, t.Threshold, true
	case "deaths":
		return c.KD.TotalDeaths, t.Threshold, true
	case "gold_total":
		return c.Gold + c.Bank, t.Threshold, true
	case "mutation_count":
		return len(c.Mutations), t.Threshold, true
	case "rooms_explored":
		return roomsExplored(c), t.Threshold, true
	case "quests_completed":
		return completedQuestCount(c), t.Threshold, true
	case "stat_reached":
		return statValue(c, t.Stat), t.Threshold, true
	case "skill_reached":
		return skillValue(c, t.Skill), t.Threshold, true
	}
	return 0, 0, false
}

func roomsExplored(c *characters.Character) int {
	total := 0
	for _, ids := range c.VisitedRooms {
		total += len(ids)
	}
	return total
}

// completedQuestCount counts quests whose current step is the "end" step (the
// project's quest-completion convention: end token "{questid}-end").
func completedQuestCount(c *characters.Character) int {
	n := 0
	for _, step := range c.QuestProgress {
		if step == "end" {
			n++
		}
	}
	return n
}

func statValue(c *characters.Character, name string) int {
	switch name {
	case "strength":
		return c.Stats.Strength.ValueAdj
	case "dexterity":
		return c.Stats.Dexterity.ValueAdj
	case "perception":
		return c.Stats.Perception.ValueAdj
	case "vitality":
		return c.Stats.Vitality.ValueAdj
	case "willpower":
		return c.Stats.Willpower.ValueAdj
	case "charisma":
		return c.Stats.Charisma.ValueAdj
	case "any":
		best := 0
		for _, v := range []int{
			c.Stats.Strength.ValueAdj, c.Stats.Dexterity.ValueAdj, c.Stats.Perception.ValueAdj,
			c.Stats.Vitality.ValueAdj, c.Stats.Willpower.ValueAdj, c.Stats.Charisma.ValueAdj,
		} {
			if v > best {
				best = v
			}
		}
		return best
	}
	return 0
}

func skillValue(c *characters.Character, name string) int {
	if name == "any" {
		best := 0
		for _, tag := range skills.GetAllSkillNames() {
			if lvl := c.GetSkillLevel(tag); lvl > best {
				best = lvl
			}
		}
		return best
	}
	return c.GetSkillLevel(skills.SkillTag(name))
}

// ownsPinnacleItem reports whether the character holds any equipment-type item at
// or above the given rarity tier (backpack + equipped). Bank storage lives on the
// UserRecord (not the Character), so it is out of scope for this pure evaluator.
func ownsPinnacleItem(c *characters.Character, minRarity int) bool {
	qualifies := func(it items.Item) bool {
		spec := it.GetSpec()
		return pinnacleEquipTypes[spec.Type] && spec.RarityTier >= minRarity
	}
	for _, it := range c.Items {
		if qualifies(it) {
			return true
		}
	}
	for _, it := range c.Equipment.GetAllItems() {
		if qualifies(it) {
			return true
		}
	}
	return false
}
