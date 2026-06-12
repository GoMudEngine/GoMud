package characters

import (
	"math"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/enchantments"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
	"github.com/GoMudEngine/GoMud/internal/state/presence"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/stats"
)

// returns true if something has changed.
func (c *Character) RecalculateStats() {

	beforeHealthMax := c.HealthMax
	beforeStats := c.Stats

	// Build per-stat entries once, referencing live pointers into c.Stats.
	type statEntry struct {
		ptr     *stats.StatInfo
		modName string // statmods.StatName string
		mutKey  string // mutations key, e.g. "strength"
	}
	entries := []statEntry{
		{&c.Stats.Strength, string(statmods.Strength), "strength"},
		{&c.Stats.Dexterity, string(statmods.Dexterity), "dexterity"},
		{&c.Stats.Perception, string(statmods.Perception), "perception"},
		{&c.Stats.Vitality, string(statmods.Vitality), "vitality"},
		{&c.Stats.Willpower, string(statmods.Willpower), "willpower"},
		{&c.Stats.Charisma, string(statmods.Charisma), "charisma"},
	}

	// Pass 1 — species-base hydration (only when Base is 0, per original logic).
	if speciesInfo := species.GetSpecies(c.SpeciesId); speciesInfo != nil {
		speciesEntries := []struct {
			ptr  *stats.StatInfo
			base int
		}{
			{&c.Stats.Strength, speciesInfo.Stats.Strength.Base},
			{&c.Stats.Dexterity, speciesInfo.Stats.Dexterity.Base},
			{&c.Stats.Perception, speciesInfo.Stats.Perception.Base},
			{&c.Stats.Vitality, speciesInfo.Stats.Vitality.Base},
			{&c.Stats.Willpower, speciesInfo.Stats.Willpower.Base},
			{&c.Stats.Charisma, speciesInfo.Stats.Charisma.Base},
		}
		for _, e := range speciesEntries {
			if e.ptr.Base == 0 {
				e.ptr.Base = e.base
			}
		}
	}

	// Pass 2 — apply equipment mods and mutation stat_flat, then Recalculate().
	for _, e := range entries {
		e.ptr.Mods = c.StatMod(e.modName)
		e.ptr.Mods += mutations.GetStatFlat(c.Mutations, e.mutKey)
		e.ptr.Recalculate()
	}

	// Pass 3 — apply mutation stat_multiplier to ValueAdj.
	for _, e := range entries {
		if v := mutations.GetStatMultiplier(c.Mutations, e.mutKey); v != 0 {
			e.ptr.ValueAdj = int(float64(e.ptr.ValueAdj) * (1.0 + v))
		}
	}

	// ── Derive pool maxes from stats (unchanged from pre-refactor) ─────
	rb := configs.GetBalanceConfig()
	c.HealthMax.Mods = int(rb.HealthBase) +
		c.StatMod(string(statmods.HealthMax)) +
		c.Stats.Strength.ValueAdj*int(rb.HealthPerStrength) +
		c.Stats.Vitality.ValueAdj*int(rb.HealthPerVitality)

	c.StaminaMax.Mods = int(rb.StaminaBase) +
		c.Stats.Strength.ValueAdj*int(rb.StaminaPerStrength) +
		c.Stats.Willpower.ValueAdj*int(rb.StaminaPerWillpower) +
		c.Stats.Vitality.ValueAdj*int(rb.StaminaPerVitality)

	c.ConvictionMax.Mods = int(rb.ConvictionBase) +
		(c.Stats.Willpower.ValueAdj+c.Stats.Charisma.ValueAdj)*int(rb.ConvictionPerWilCha)

	c.ActionPointsMax.Mods = 200 // hard coded for now

	c.HealthMax.Recalculate()
	c.StaminaMax.Recalculate()
	c.ConvictionMax.Recalculate()
	c.ActionPointsMax.Recalculate()

	// Stage 12.1: health_multiplier mutation after HealthMax.Recalculate().
	if hMult := mutations.GetHealthMultiplier(c.Mutations); hMult != 0 {
		c.HealthMax.Value = int(float64(c.HealthMax.Value) * (1.0 + hMult))
		if c.HealthMax.Value < 1 {
			c.HealthMax.Value = 1
		}
	}

	// Floors. Pool maxes are floored at 1, not 0, because downstream
	// consumers (prompt `{sp%}` / `{mp%}` tokens at
	// internal/users/userrecord.prompt.go, ratio calcs in combat /
	// resource-multiplier curves) divide by these values without a
	// zero-guard. A degenerate character with Willpower=0 used to
	// crash the prompt-render path; floor 1 prevents the divide.
	if c.StaminaMax.Value < 1 {
		c.StaminaMax.Value = 1
	}
	if c.ConvictionMax.Value < 1 {
		c.ConvictionMax.Value = 1
	}
	if c.HealthMax.Value < 1 {
		c.HealthMax.Value = 1
	}
	if c.ActionPointsMax.Value < 50 {
		c.ActionPointsMax.Value = 50
	}

	// Chrysalis pool reservation clamping (unchanged).
	if hpRes := c.GetPoolReservation("health", c.HealthMax.Value); hpRes > 0 {
		effectiveHP := c.HealthMax.Value - hpRes
		if effectiveHP < 1 {
			effectiveHP = 1
		}
		if c.Health > effectiveHP {
			c.Health = effectiveHP
		}
	}
	if spRes := c.GetPoolReservation("stamina", c.StaminaMax.Value); spRes > 0 {
		effectiveSP := c.StaminaMax.Value - spRes
		if effectiveSP < 0 {
			effectiveSP = 0
		}
		if c.Stamina > effectiveSP {
			c.Stamina = effectiveSP
		}
	}
	if cpRes := c.GetPoolReservation("conviction", c.ConvictionMax.Value); cpRes > 0 {
		effectiveCP := c.ConvictionMax.Value - cpRes
		if effectiveCP < 0 {
			effectiveCP = 0
		}
		if c.Conviction > effectiveCP {
			c.Conviction = effectiveCP
		}
	}

	// Stage 31.6: Enchant withdrawal condition — unchanged.
	if c.HasCondition(ConditionEnchantWithdrawal) {
		mag := c.GetConditionMagnitude(ConditionEnchantWithdrawal)
		for _, cond := range c.Conditions {
			if cond.Type == ConditionEnchantWithdrawal {
				penalty := int(math.Floor(float64(c.HealthMax.Value) * mag))
				switch cond.Source {
				case "health":
					c.HealthMax.Value -= penalty
					if c.HealthMax.Value < 1 {
						c.HealthMax.Value = 1
					}
					if c.Health > c.HealthMax.Value {
						c.Health = c.HealthMax.Value
					}
				case "stamina":
					penalty = int(math.Floor(float64(c.StaminaMax.Value) * mag))
					c.StaminaMax.Value -= penalty
					if c.StaminaMax.Value < 0 {
						c.StaminaMax.Value = 0
					}
					if c.Stamina > c.StaminaMax.Value {
						c.Stamina = c.StaminaMax.Value
					}
				case "conviction":
					penalty = int(math.Floor(float64(c.ConvictionMax.Value) * mag))
					c.ConvictionMax.Value -= penalty
					if c.ConvictionMax.Value < 0 {
						c.ConvictionMax.Value = 0
					}
					if c.Conviction > c.ConvictionMax.Value {
						c.Conviction = c.ConvictionMax.Value
					}
				}
				break
			}
		}
	}

	// Emit CharacterStatsChanged if any tracked value changed.
	if c.userId != 0 {
		changed := false
		if beforeStats.Strength.ValueAdj != c.Stats.Strength.ValueAdj {
			changed = true
		} else if beforeStats.Dexterity.ValueAdj != c.Stats.Dexterity.ValueAdj {
			changed = true
		} else if beforeStats.Perception.ValueAdj != c.Stats.Perception.ValueAdj {
			changed = true
		} else if beforeStats.Vitality.ValueAdj != c.Stats.Vitality.ValueAdj {
			changed = true
		} else if beforeStats.Willpower.ValueAdj != c.Stats.Willpower.ValueAdj {
			changed = true
		} else if beforeStats.Charisma.ValueAdj != c.Stats.Charisma.ValueAdj {
			changed = true
		} else if beforeHealthMax != c.HealthMax {
			changed = true
		}

		if changed {
			events.AddToQueue(events.CharacterStatsChanged{UserId: c.userId})
		}
	}

}

// GetPoolReservation returns the total pool max reduction from Chrysalis enchantments
// on all equipped items that reserve the given pool ("health", "stamina", "conviction").
func (c *Character) GetPoolReservation(pool string, poolMax int) int {
	total := 0
	for _, itm := range c.Equipment.GetAllItems() {
		if !itm.HasChrysalisEnchantment() || itm.ReservePool != pool {
			continue
		}
		pct := enchantments.GetTierReservePct(itm.EnchantType, itm.EnchantTier, itm.GetSpec().Hands)
		total += int(math.Floor(float64(poolMax) * pct))
	}
	return total
}

func (c *Character) CanDualWield() bool {
	// Dual wielding is now governed by weapon-combat skill
	return c.GetSkillLevel(skills.WeaponCombat) > 0
}

// validateSkillMigrations renames legacy skills, merges retired skills,
// and removes dead skill keys. Must run BEFORE ensureAllSkills.
func (c *Character) validateSkillMigrations() {
	if c.Skills == nil {
		return
	}

	// stealth → skullduggery rename.
	if v, ok := c.Skills["stealth"]; ok {
		c.Skills["skullduggery"] = v
		delete(c.Skills, "stealth")
	}

	// tracking + foraging → search merge.
	if _, hasTracking := c.Skills["tracking"]; hasTracking {
		trackRank := c.Skills["tracking"]
		forageRank := c.Skills["foraging"]
		searchRank := max(trackRank, forageRank)
		if searchRank < 1 {
			searchRank = 1
		}
		c.Skills["search"] = searchRank
		if c.SkillUseCount == nil {
			c.SkillUseCount = make(map[string]int)
		}
		c.SkillUseCount["search"] = c.SkillUseCount["tracking"] + c.SkillUseCount["foraging"]
		delete(c.Skills, "tracking")
		delete(c.Skills, "foraging")
		delete(c.SkillUseCount, "tracking")
		delete(c.SkillUseCount, "foraging")
	} else if _, hasForaging := c.Skills["foraging"]; hasForaging {
		c.Skills["search"] = max(c.Skills["foraging"], 1)
		if c.SkillUseCount == nil {
			c.SkillUseCount = make(map[string]int)
		}
		c.SkillUseCount["search"] = c.SkillUseCount["foraging"]
		delete(c.Skills, "foraging")
		delete(c.SkillUseCount, "foraging")
	}

	// Remove retired skills.
	for _, dead := range []string{"cast", "first-aid"} {
		delete(c.Skills, dead)
		if c.SkillUseCount != nil {
			delete(c.SkillUseCount, dead)
		}
	}
}

// validatePoolClamps clamps current Health/Stamina/Conviction into their
// legal ranges after RecalculateStats has been called.
func (c *Character) validatePoolClamps() {
	if c.Stamina > c.StaminaMax.Value {
		c.Stamina = c.StaminaMax.Value
	}
	if c.Conviction > c.ConvictionMax.Value {
		c.Conviction = c.ConvictionMax.Value
	}
	if c.Health > c.HealthMax.Value {
		c.Health = c.HealthMax.Value
	}
	// No lower Health clamp: Health <= 0 means dead, handled by the per-
	// round death check (NewRound_DoCombat / NewRound_AutoHeal → suicide).
	if c.Stamina < 0 {
		c.Stamina = 0
	}
	if c.Conviction < 0 {
		c.Conviction = 0
	}
}

// validateEquipmentItems calls items.Item.Validate() on every in-play item
// (backpack, bandolier, component-bag contents, and EVERY worn slot) to ensure
// they all have a uid. This MUST cover all slots: an un-validated item keeps a
// nil UUID, which stringifies to a constant — so two such items collide for
// @<handle> targeting (a missing slot here = the web inventory panel's actions
// hitting the wrong item). Iterate via a pointer list so a forgotten slot is
// obvious.
func (c *Character) validateEquipmentItems() {
	for i := range c.Items {
		c.Items[i].Validate()
	}
	for i := range c.PotionItems { // bandolier
		c.PotionItems[i].Validate()
	}
	for i := range c.ComponentItems { // component-bag contents
		c.ComponentItems[i].Validate()
	}
	for _, s := range c.Equipment.AllSlots() {
		s.Item.Validate()
	}
}

// validateDisabledSlotsForSpecies enables all slots, then disables the ones
// the species requires to be disabled. Items found in to-be-disabled slots
// are moved to the backpack.
func (c *Character) validateDisabledSlotsForSpecies() {
	speciesInfo := species.GetSpecies(c.SpeciesId)
	if speciesInfo == nil {
		return
	}

	if len(speciesInfo.DisabledSlots) == 0 {
		return
	}

	for _, disabledSlot := range speciesInfo.DisabledSlots {
		var itemFoundInDisabledSlot items.Item = items.ItemDisabledSlot

		switch items.ItemType(disabledSlot) {
		case items.Weapon:
			if c.Equipment.Weapon.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Weapon
			}
			c.Equipment.Weapon = items.ItemDisabledSlot
		case items.Offhand:
			if c.Equipment.Offhand.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Offhand
			}
			c.Equipment.Offhand = items.ItemDisabledSlot
		case items.Head:
			if c.Equipment.Head.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Head
			}
			c.Equipment.Head = items.ItemDisabledSlot
		case items.Neck:
			if c.Equipment.Neck.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Neck
			}
			c.Equipment.Neck = items.ItemDisabledSlot
		case items.Body:
			if c.Equipment.Body.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Body
			}
			c.Equipment.Body = items.ItemDisabledSlot
		case items.Belt:
			if c.Equipment.Belt.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Belt
			}
			c.Equipment.Belt = items.ItemDisabledSlot
		case items.Gloves:
			if c.Equipment.Gloves.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Gloves
			}
			c.Equipment.Gloves = items.ItemDisabledSlot
		case items.Ring:
			if c.Equipment.Ring.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Ring
			}
			c.Equipment.Ring = items.ItemDisabledSlot
		case items.Legs:
			if c.Equipment.Legs.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Legs
			}
			c.Equipment.Legs = items.ItemDisabledSlot
		case items.Feet:
			if c.Equipment.Feet.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Feet
			}
			c.Equipment.Feet = items.ItemDisabledSlot
		case items.Wrist:
			if c.Equipment.Wrist1.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Wrist1
			}
			c.Equipment.Wrist1 = items.ItemDisabledSlot
			if c.Equipment.Wrist2.ItemId > 0 {
				c.StoreItem(c.Equipment.Wrist2)
			}
			c.Equipment.Wrist2 = items.ItemDisabledSlot
		case items.Back:
			if c.Equipment.Back.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Back
			}
			c.Equipment.Back = items.ItemDisabledSlot
		case items.Shoulders:
			if c.Equipment.Shoulders.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Shoulders
			}
			c.Equipment.Shoulders = items.ItemDisabledSlot
		case items.ComponentBag:
			if c.Equipment.ComponentBag.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.ComponentBag
			}
			c.Equipment.ComponentBag = items.ItemDisabledSlot
		}

		// Non-ItemType disabled slots (string-keyed).
		if disabledSlot == "ring2" {
			if c.Equipment.Ring2.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Ring2
			}
			c.Equipment.Ring2 = items.ItemDisabledSlot
		}

		if !itemFoundInDisabledSlot.IsDisabled() {
			c.StoreItem(itemFoundInDisabledSlot)
			mudlog.Debug("Disabled Check", "error", "Item found in disabled slot", "name", itemFoundInDisabledSlot.Name(), "slot", disabledSlot, "character", c.Name)
		}
	}
}

// validateMutationSlots enforces extra-arm / tail slot availability based on
// the character's current ExtraArms count and tail mutation.
func (c *Character) validateMutationSlots() {
	// Derive ExtraArms from mutation level (capped at 4).
	if lvl, ok := c.Mutations["extra-arms"]; ok && lvl > 0 {
		c.ExtraArms = lvl
		if c.ExtraArms > 4 {
			c.ExtraArms = 4
		}
	} else {
		c.ExtraArms = 0
	}

	// Extra arms: unavailable levels move items back to backpack.
	if c.ExtraArms < 4 {
		if c.Equipment.ExtraArm4.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraArm4)
		}
		c.Equipment.ExtraArm4 = items.ItemDisabledSlot
		if c.Equipment.ExtraWrist4.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraWrist4)
		}
		c.Equipment.ExtraWrist4 = items.ItemDisabledSlot
	}
	if c.ExtraArms < 3 {
		if c.Equipment.ExtraArm3.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraArm3)
		}
		c.Equipment.ExtraArm3 = items.ItemDisabledSlot
		if c.Equipment.ExtraWrist3.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraWrist3)
		}
		c.Equipment.ExtraWrist3 = items.ItemDisabledSlot
	}
	if c.ExtraArms < 2 {
		if c.Equipment.ExtraArm2.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraArm2)
			mudlog.Debug("Extra Arms Check", "info", "Item returned from extra arm 2 slot", "name", c.Equipment.ExtraArm2.Name(), "character", c.Name)
		}
		c.Equipment.ExtraArm2 = items.ItemDisabledSlot
		if c.Equipment.ExtraWrist2.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraWrist2)
		}
		c.Equipment.ExtraWrist2 = items.ItemDisabledSlot
	}
	if c.ExtraArms < 1 {
		if c.Equipment.ExtraArm1.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraArm1)
			mudlog.Debug("Extra Arms Check", "info", "Item returned from extra arm 1 slot", "name", c.Equipment.ExtraArm1.Name(), "character", c.Name)
		}
		c.Equipment.ExtraArm1 = items.ItemDisabledSlot
		if c.Equipment.ExtraWrist1.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraWrist1)
		}
		c.Equipment.ExtraWrist1 = items.ItemDisabledSlot
	}

	// Tail mutation: enable tail slot if mutation present, disable otherwise.
	if _, hasTail := c.Mutations["tail"]; hasTail {
		if c.Equipment.Tail.ItemId < 0 {
			c.Equipment.Tail = items.Item{}
		}
	} else {
		if c.Equipment.Tail.ItemId > 0 {
			c.StoreItem(c.Equipment.Tail)
		}
		c.Equipment.Tail = items.ItemDisabledSlot
	}

	// Tail mutation disables legs slot via disable-legs flag.
	if flags := mutations.GetMutationFlags(c.Mutations); flags["disable-legs"] {
		if c.Equipment.Legs.ItemId > 0 {
			c.StoreItem(c.Equipment.Legs)
			mudlog.Debug("Mutation Check", "info", "Item returned from legs slot (tail mutation)", "name", c.Equipment.Legs.Name(), "character", c.Name)
		}
		c.Equipment.Legs = items.ItemDisabledSlot
	}
}

// Returns whether a correction was in order
func (c *Character) Validate(recalcPermaBuffs ...bool) error {

	// ── Skill migrations must run before ensureAllSkills ────────────
	c.validateSkillMigrations()

	// Ensure runtime-only fields are initialised after YAML load
	// (yaml:"-" fields are not populated by yaml.Unmarshal).
	if c.CombatPhase == nil {
		c.CombatPhase = combatphase.NewMachine()
	}
	if c.Awareness == nil {
		c.Awareness = awareness.NewMachine()
	}
	if c.Life == nil {
		c.Life = life.NewMachine()
	}
	if c.Activity == nil {
		c.Activity = activity.NewMachine()
	}
	if c.Position == nil {
		c.Position = position.NewMachine()
	}
	if c.Presence == nil {
		// Player default — mob.Validate() overwrites with NewMobPresence()
		// AFTER calling this. Control intentionally lacks a parallel guard
		// here and uses per-call-site nil checks instead (see
		// position_predicates.go for examples).
		c.Presence = presence.NewPlayerPresence()
		// Chunk 5 (Presence) T8: terminal-state cleanup. On entry to
		// Disconnected, cancel all pending scheduled transitions for this
		// character (Activity casting timers, Position recovery timers, etc.)
		// so they don't fire after the player has left the world.
		c.Presence.RegisterObserver("scheduler_cancel_on_disconnected",
			func(from, to presence.State, r state.TransitionReason) {
				if to == presence.Disconnected {
					c.CancelAllScheduled()
				}
			})
	}
	if c.Perception == nil {
		// Player default — mob.Validate() overwrites unconditionally
		// after this runs. Same constructor for both actor types but
		// the unconditional overwrite matches the Presence pattern.
		c.Perception = perception.NewMachine()
	}
	if c.PerGrappleMessageCooldowns == nil {
		c.PerGrappleMessageCooldowns = map[string]bool{}
	}
	// Fire OnCharacterCreated callbacks exactly once per Character.
	// The guard prevents repeated firing on re-validation (e.g. stat
	// recalcs, equipment changes) while still covering the YAML-load
	// path where New() was never called.
	if !c.combatPhaseWired {
		c.combatPhaseWired = true
		fireCharacterCreated(c)
	}

	if len(c.Description) == 0 {
		c.Description = "They seem thoroughly uninteresting."
	}

	if sp := species.GetSpecies(c.SpeciesId); sp == nil {
		c.SpeciesId = 1
	}

	if c.Created.IsZero() {
		c.Created = time.Now()
	}

	if c.Pet.Exists() {
		c.Pet.Validate()
	}

	if c.SpellBook == nil {
		c.SpellBook = make(map[string]int)
	}

	if c.KnownRecipes == nil {
		c.KnownRecipes = crafting.GetStarterRecipes()
	} else {
		// Backfill any new starter recipes added since character creation
		for id, val := range crafting.GetStarterRecipes() {
			if _, ok := c.KnownRecipes[id]; !ok {
				c.KnownRecipes[id] = val
			}
		}
	}

	if c.Mutations == nil {
		c.Mutations = make(map[string]int)
	}

	if c.Zone == "" {
		c.Zone = startingZone
	}

	if c.Name == "" {
		c.Name = defaultName
	}
	c.Buffs.Validate()

	// Ensure all known skills exist at rank 1 minimum.
	c.Skills = ensureAllSkills(c.Skills)

	// Stats recalc based on equipment, race, level, etc.
	c.RecalculateStats()

	// Pool clamping after recalc.
	c.validatePoolClamps()

	c.Cooldowns.Prune()

	// Validate possessed/worn items (UIDs).
	c.validateEquipmentItems()
	// Reset all slots; both helpers below layer their rules on top.
	c.Equipment.EnableAll()

	// Apply species-disabled slot rules (requires validateEquipmentItems first).
	c.validateDisabledSlotsForSpecies()

	// Apply mutation-driven slot rules (extra arms, tail, disable-legs).
	c.validateMutationSlots()

	if len(recalcPermaBuffs) > 0 && recalcPermaBuffs[0] {
		c.reapplyPermabuffs()
	}

	return nil
}
