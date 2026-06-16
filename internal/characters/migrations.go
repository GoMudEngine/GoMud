package characters

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// MigratePairedSpells is a one-time migration that grants missing
// paired spells to existing characters. Call on character load.
// Uses MiscData flag to run only once per character.
func (c *Character) MigratePairedSpells() {
	const migrationKey = "migration-fold-pair-done"
	if c.GetMiscData(migrationKey) != nil {
		return
	}
	for spell, paired := range pairedSpells {
		if _, known := c.SpellBook[spell]; known {
			if _, hasPartner := c.SpellBook[paired]; !hasPartner {
				c.SpellBook[paired] = 1
			}
		}
	}
	c.SetMiscData(migrationKey, "1")
}

// MigrateNeckToBack is a one-time migration that moves cloak/cape items
// from the Neck slot to the Back slot. Runs on character load.
func (c *Character) MigrateNeckToBack() {
	const migrationKey = "migration-neck-to-back-done"
	if c.GetMiscData(migrationKey) != nil {
		return
	}
	// If neck item is a back-type item (cloak data was updated), move it
	if c.Equipment.Neck.ItemId > 0 {
		spec := c.Equipment.Neck.GetSpec()
		if spec.Type == items.Back {
			// Only move if back slot is empty
			if c.Equipment.Back.ItemId <= 0 && !c.Equipment.Back.IsDisabled() {
				c.Equipment.Back = c.Equipment.Neck
				c.Equipment.Neck = items.Item{}
			}
		}
	}
	c.SetMiscData(migrationKey, "1")
}

// MigrateQuestSpells is a one-time migration that grants spells to
// characters who completed quests before spell rewards were added.
func (c *Character) MigrateQuestSpells() {
	const migrationKey = "migration-quest-spells-done"
	if c.GetMiscData(migrationKey) != nil {
		return
	}
	// Quest 12 (Warden's Covenant) should grant summon-steppe-spirit
	if c.HasQuest("12-end") {
		c.LearnSpell("summon-steppe-spirit")
	}
	c.SetMiscData(migrationKey, "1")
}

// MigrateDescriptionWrapping strips embedded line breaks from player
// descriptions so wrapping only happens at display time.
func (c *Character) MigrateDescriptionWrapping() {
	const migrationKey = "migration-desc-unwrap-done"
	if c.GetMiscData(migrationKey) != nil {
		return
	}

	if c.Description != "" && !strings.HasPrefix(c.Description, "h:") {
		// Collapse \r\n and \n back to spaces
		d := strings.ReplaceAll(c.Description, "\r\n", " ")
		d = strings.ReplaceAll(d, "\n", " ")
		// Collapse any double spaces from the join
		for strings.Contains(d, "  ") {
			d = strings.ReplaceAll(d, "  ", " ")
		}
		c.Description = strings.TrimSpace(d)
	}

	c.SetMiscData(migrationKey, "1")
}

// MigrateAlchemyPotions replaces old potion items with new equivalents.
func (c *Character) MigrateAlchemyPotions() {
	const migrationKey = "migration-alchemy-potions-done"
	if c.GetMiscData(migrationKey) != nil {
		return
	}

	potionMap := map[int]int{
		30010: 30036, // Healing Poultice → Healing Salve
		30011: 30037, // Stamina Draught → Stamina Tonic
		30012: 30038, // Conviction Draught → Conviction Draught
		30028: 30046, // Minor Antidote → Stone Stomach
		30029: 30044, // Clarity Tonic → Mindshield Elixir
		30030: 30043, // Fire Resistance → Ironhide Brew
		30031: 30042, // Greater Healing → Elixir of Renewal
		30032: 30049, // Berserker Elixir → Berserker Elixir
	}

	migrateSlice := func(slice []items.Item) []items.Item {
		for i := range slice {
			if newId, ok := potionMap[slice[i].ItemId]; ok {
				slice[i].ItemId = newId
				slice[i].CraftedRound = util.GetRoundCount()
				slice[i].CraftSkill = 10
				slice[i].BottleMultiplier = 1.0 // Glass vial baseline
			}
		}
		return slice
	}

	c.Items = migrateSlice(c.Items)
	c.PotionItems = migrateSlice(c.PotionItems)

	c.SetMiscData(migrationKey, "1")
}

// MigrateAlchemyRecipes grants new recipe equivalents for old known recipes.
func (c *Character) MigrateAlchemyRecipes() {
	const migrationKey = "migration-alchemy-recipes-done"
	if c.GetMiscData(migrationKey) != nil {
		return
	}

	recipeMap := map[string]string{
		"healing-poultice":        "healing-salve",
		"stamina-draught":         "stamina-tonic",
		"conviction-draught":      "conviction-draught",
		"minor-antidote":          "stone-stomach",
		"clarity-tonic":           "mindshield-elixir",
		"fire-resistance-draught": "ironhide-brew",
		"greater-healing-poultice": "elixir-of-renewal",
		"berserker-elixir":        "berserker-elixir",
	}

	if c.KnownRecipes != nil {
		for oldId, newId := range recipeMap {
			if _, known := c.KnownRecipes[oldId]; known {
				c.KnownRecipes[newId] = 1
			}
		}
	}

	c.SetMiscData(migrationKey, "1")
}

// MigrateQuestFlags infers quest flags from downstream quest progress.
// For Quest 11's branch flag: if the player has Q12 progress, they took
// the Sylara path; if Q13 progress, the Rhett path.
func (c *Character) MigrateQuestFlags() {
	if c.QuestFlags != nil {
		return // already has flags — skip
	}

	// Infer Q11 branch from Q12/Q13 progress
	q12Progress := c.QuestProgress[12]
	q13Progress := c.QuestProgress[13]

	if q12Progress != "" {
		c.SetQuestFlag("11-branch", "sylara")
	} else if q13Progress != "" {
		c.SetQuestFlag("11-branch", "rhett")
	}
	// If neither Q12 nor Q13 started, leave unset —
	// the player will pick a branch when they next interact.
}

// MigrateLegacyPotions replaces removed alchemy items and recipes
// with their current equivalents.
// 30010 (healing poultice) → 30036 (healing salve)
// 30011 (stamina draught)  → 30037 (stamina tonic)
// 30031 (greater healing poultice) → 30036 (healing salve)
func (c *Character) MigrateLegacyPotions() {
	// Replace items in backpack
	for i := range c.Items {
		switch c.Items[i].ItemId {
		case 30010, 30031:
			c.Items[i].ItemId = 30036
		case 30011:
			c.Items[i].ItemId = 30037
		}
	}

	// Replace items in component bag
	for i := range c.ComponentItems {
		switch c.ComponentItems[i].ItemId {
		case 30010, 30031:
			c.ComponentItems[i].ItemId = 30036
		case 30011:
			c.ComponentItems[i].ItemId = 30037
		}
	}

	// Replace items in potion bandolier
	for i := range c.PotionItems {
		switch c.PotionItems[i].ItemId {
		case 30010, 30031:
			c.PotionItems[i].ItemId = 30036
		case 30011:
			c.PotionItems[i].ItemId = 30037
		}
	}

	// Replace recipe knowledge
	if c.KnownRecipes != nil {
		if _, ok := c.KnownRecipes["healing-poultice"]; ok {
			delete(c.KnownRecipes, "healing-poultice")
			c.KnownRecipes["healing-salve"] = 1
		}
		if _, ok := c.KnownRecipes["stamina-draught"]; ok {
			delete(c.KnownRecipes, "stamina-draught")
			c.KnownRecipes["stamina-tonic"] = 1
		}
		delete(c.KnownRecipes, "greater-healing-poultice")
	}
}

// MigrateChrysalisAidRemoved prunes the deleted chrysalis-aid spell from
// any spellbook that still contains it. Runs once per character on load.
func (c *Character) MigrateChrysalisAidRemoved() {
	const migrationKey = "migration-chrysalis-aid-removed"
	if c.GetMiscData(migrationKey) != nil {
		return
	}
	delete(c.SpellBook, "chrysalis-aid")
	c.SetMiscData(migrationKey, "1")
}

// MigrateRecipeDisciplineShuffle handles the 2026-05-04 reclassification
// of two recipes that moved between crafting disciplines:
//   - master-lockpicks: jewelcrafting → blacksmithing
//   - reinforced-disarm-kit: blacksmithing → jewelcrafting
//
// The recipe IDs are unchanged, so KnownRecipes still references them
// correctly. But because each recipe now gates on a different skill,
// players who learned a recipe under the OLD discipline could lose
// craft access if their NEW discipline rank is below the recipe's
// skill_minimum. This migration bumps the relevant skill to the recipe
// minimum (or leaves higher ranks alone) for any character that knows
// the affected recipe.
//
// Runs once per character.
func (c *Character) MigrateRecipeDisciplineShuffle() {
	const migrationKey = "migration-recipe-discipline-shuffle-2026-05-04"
	if c.GetMiscData(migrationKey) != nil {
		return
	}

	if c.KnownRecipes != nil {
		if _, ok := c.KnownRecipes["master-lockpicks"]; ok {
			c.TrainSkill("blacksmithing", 20)
		}
		if _, ok := c.KnownRecipes["reinforced-disarm-kit"]; ok {
			c.TrainSkill("jewelcrafting", 15)
		}
	}

	c.SetMiscData(migrationKey, "1")
}

// MigrateNewbieAwakening grants quest-30 "end" to pre-Pothole veteran
// characters so the hub movement gate (room 5200, gated on 30-end) never
// traps them. Run-once via MiscData.
//
// Guard logic — grant only when ALL of these hold:
//   - no prior run (MiscData key absent)
//   - QuestProgress is non-empty (veteran has done something)
//   - quest 30 has no entry at all (not mid-rite, not already done)
//
// A brand-new character has empty QuestProgress → skipped, so it still
// does the Awakening rite. A character mid-rite (30:"start") has a q30
// entry → skipped, so it is not short-circuited. Errs toward NOT
// granting: a stray veteran is merely redirected by the gate NPC; a
// skipped rite would be worse.
func (c *Character) MigrateNewbieAwakening() {
	const migrationKey = "migration-newbie-awakening-done"
	if c.GetMiscData(migrationKey) != nil {
		return
	}
	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}
	if _, hasQ30 := c.QuestProgress[30]; !hasQ30 && len(c.QuestProgress) > 0 {
		c.QuestProgress[30] = "end"
	}
	c.SetMiscData(migrationKey, "1")
}
