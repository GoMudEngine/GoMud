package characters

import (
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ── Pet → Companion migration (DEFERRED) ─────────────────────────────────────
//
// The legacy Pet system (Character.Pet of type pets.Pet) is fundamentally
// different from the Companions system and cannot be cleanly migrated today:
//
//   - Pets have no MobId — they are identified by a Type string and loaded
//     from pet YAML definitions, not mob templates. CompanionInfo requires
//     a MobId to respawn the companion on login.
//
//   - Pets provide stat mods and buffs directly to the player character
//     (GetStatMod, GetBuffs). The Companions system does not have this concept.
//
//   - Pets have their own inventory (Pet.Items) which has no counterpart in
//     CompanionInfo.
//
// Migration plan (future work):
//  1. Create pet-backed mob templates that mirror each pet type's stats/buffs.
//  2. Add a PetMobId field to each pet YAML definition.
//  3. In Validate(), create a CompanionInfo{SourceType: CompanionPet, MobId: …}
//     for each existing Pet and clear Character.Pet.
//  4. Remove the pets package once all players have migrated.
//
// Until then, Character.Pet and Character.Companions coexist. CompanionPet
// source type is reserved for when this migration is implemented.
// ─────────────────────────────────────────────────────────────────────────────

// CompanionSourceType describes how a companion came to follow the player.
type CompanionSourceType string

const (
	CompanionSummoned CompanionSourceType = "summoned"
	CompanionConjured CompanionSourceType = "conjured"
	CompanionCharmed  CompanionSourceType = "charmed"
	CompanionRaised   CompanionSourceType = "raised"
	CompanionPet      CompanionSourceType = "pet"
)

// CompanionInfo holds the persistent state of a single companion.
// InstanceId is runtime-only and is not saved to disk.
type CompanionInfo struct {
	MobId            int                 `yaml:"mobid"`
	InstanceId       int                 `yaml:"-"` // runtime only
	SourceType       CompanionSourceType `yaml:"source_type"`
	Name             string              `yaml:"name"`
	BaseName         string              `yaml:"base_name,omitempty"` // mob template name, e.g. "Spirit Wolf"
	Nickname         string              `yaml:"nickname,omitempty"`  // player-given name, e.g. "Fred"
	AutoAssist       bool                `yaml:"auto_assist"`
	StatTraining     map[string]int      `yaml:"stat_training,omitempty"`
	Skills           map[string]int      `yaml:"skills,omitempty"`
	SkillUseCount    map[string]int      `yaml:"skill_use_count,omitempty"`
	Mutations        map[string]int      `yaml:"mutations,omitempty"`
	SpellBook        map[string]int      `yaml:"spellbook,omitempty"`
	MutationProgress float64             `yaml:"mutation_progress,omitempty"`
	CharmDuration    int                 `yaml:"charm_duration,omitempty"`  // Rounds until charm re-roll (0 = no timer)
	CharmRerolls     int                 `yaml:"charm_rerolls,omitempty"`   // Number of times charm has been re-rolled
	// Gear persistence — snapshotted at logout, restored on respawn.
	Items     []items.Item `yaml:"items,omitempty"`     // Carried (backpack)
	Equipment Worn         `yaml:"equipment,omitempty"` // Equipped/worn slots
}

// GetCompanion finds a companion by name (case-insensitive partial match).
// Supports N.name / name#N disambiguation (via util.GetMatchNumber).
// Returns nil if no match is found.
func (c *Character) GetCompanion(name string) *CompanionInfo {
	search, matchNum := util.GetMatchNumber(name)
	lower := strings.ToLower(search)
	count := 0
	for i := range c.Companions {
		if strings.Contains(strings.ToLower(c.Companions[i].Name), lower) {
			count++
			if count == matchNum {
				return &c.Companions[i]
			}
		}
	}
	return nil
}

// GetCompanionByInstanceId finds a companion by its runtime mob instance ID.
// Returns nil if not found.
func (c *Character) GetCompanionByInstanceId(instanceId int) *CompanionInfo {
	for i := range c.Companions {
		if c.Companions[i].InstanceId == instanceId {
			return &c.Companions[i]
		}
	}
	return nil
}

// AddCompanion adds a companion to the character's companion list.
// Returns false if the character is already at max companion capacity.
func (c *Character) AddCompanion(info CompanionInfo) bool {
	if len(c.Companions) >= c.GetMaxCompanions() {
		return false
	}
	c.Companions = append(c.Companions, info)
	return true
}

// RemoveCompanion removes a companion by runtime instance ID.
// Returns a pointer to the removed CompanionInfo, or nil if not found.
func (c *Character) RemoveCompanion(instanceId int) *CompanionInfo {
	for i := range c.Companions {
		if c.Companions[i].InstanceId == instanceId {
			removed := c.Companions[i]
			c.Companions = append(c.Companions[:i], c.Companions[i+1:]...)
			return &removed
		}
	}
	return nil
}

// GetMaxCompanions returns the maximum number of companions this character
// may have at once, based on the manifestation skill level.
// Formula: skill / 19, capped at 4. Minimum 1 if the character has any
// manifestation-school spells in their spellbook.
func (c *Character) GetMaxCompanions() int {
	skill := c.GetSkillLevel(skills.Manifestation)
	max := skill / 19
	if max > 4 {
		max = 4
	}

	// Minimum 1 if the character knows any manifestation-school spell.
	if max < 1 && c.hasManifestationSpell() {
		max = 1
	}

	return max
}

// hasManifestationSpell returns true if the character's spellbook contains
// at least one spell belonging to the "manifestation" school.
func (c *Character) hasManifestationSpell() bool {
	for spellId := range c.SpellBook {
		sd := spells.GetSpell(spellId)
		if sd == nil {
			continue
		}
		for _, school := range sd.Schools {
			if strings.EqualFold(school, "manifestation") {
				return true
			}
		}
	}
	return false
}

// CalcCompanionStatPool computes the stat pool for a summoned companion,
// scaling the mob's base stat pool by the caster's Charisma and
// manifestation skill level.
//
//	scale = 1.0 + charisma/chaFactor + manifestationSkill*skillFactor
//	result = round(baseStatPool * scale)
//
// Config knobs: ManifestStatScaleChaFactor (default 200),
// ManifestStatScaleSkillFactor (default 0.02).
func CalcCompanionStatPool(baseStatPool int, charisma int, manifestationSkill int) int {
	cfg := configs.GetBalanceConfig()
	chaFactor := float64(cfg.ManifestStatScaleChaFactor)
	skillFactor := float64(cfg.ManifestStatScaleSkillFactor)
	if chaFactor <= 0 {
		chaFactor = 200
	}
	scale := 1.0 + float64(charisma)/chaFactor + float64(manifestationSkill)*skillFactor
	return int(math.Round(float64(baseStatPool) * scale))
}
