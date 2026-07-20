package characters

import (
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/skills"
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
	CharmDuration    int                 `yaml:"charm_duration,omitempty"` // Rounds until charm re-roll (0 = no timer)
	CharmRerolls     int                 `yaml:"charm_rerolls,omitempty"`  // Number of times charm has been re-rolled
	// Gear persistence — snapshotted at logout, restored on respawn.
	Items     []items.Item `yaml:"items,omitempty"`     // Carried (backpack)
	Equipment Worn         `yaml:"equipment,omitempty"` // Equipped/worn slots
	// Conviction reserved to keep this companion fielded, snapshotted at summon
	// time so it doesn't drift when the summoner's skill/mutation changes mid-life.
	ConvictionReserve int `yaml:"conviction_reserve,omitempty"`
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

// GetMaxCompanions returns the SOFT count backstop on simultaneous companions.
// It is a safety limit only — the real constraint is the Conviction budget
// (see CanAffordCompanion). The Manifester apex ("companion-cap-raise" flag)
// raises the backstop.
func (c *Character) GetMaxCompanions() int {
	cfg := configs.GetBalanceConfig()
	cap := int(cfg.CompanionSoftCap)
	if cap < 1 {
		cap = 5
	}
	if mutations.HasMutationFlag(c.Mutations, "companion-cap-raise") {
		if apex := int(cfg.CompanionSoftCapApex); apex > cap {
			cap = apex
		}
	}
	return cap
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

// CalcCompanionReserve returns the Conviction a companion of the given base
// cost reserves for THIS summoner, after manifestation-skill and Manifester-
// mutation reductions. reservation = round(base * (1 - reduction)).
func (c *Character) CalcCompanionReserve(baseCost int) int {
	cfg := configs.GetBalanceConfig()
	manif := c.GetSkillLevel(skills.Manifestation)
	manifRed := math.Min(float64(cfg.CompanionReserveSkillCap), float64(manif)*float64(cfg.CompanionReserveSkillPct))
	mutRank := mutations.GetCompanionReserveRank(c.Mutations)
	mutRed := math.Min(float64(cfg.CompanionReserveMutCap), float64(mutRank)*float64(cfg.CompanionReserveMutPctPerRank))
	reduction := math.Min(float64(cfg.CompanionReserveTotalCap), manifRed+mutRed)
	return int(math.Round(float64(baseCost) * (1.0 - reduction)))
}

// CanAffordCompanion reports whether the character can field one more companion
// reserving `reserveCost` Conviction: the new total reservation (plus any
// casting floor) must fit within ConvictionMax, and the soft count backstop
// must not be exceeded.
func (c *Character) CanAffordCompanion(reserveCost int) bool {
	if len(c.Companions) >= c.GetMaxCompanions() {
		return false
	}
	cfg := configs.GetBalanceConfig()
	current := c.GetPoolReservation("conviction", c.ConvictionMax.Value)
	floor := int(math.Round(float64(c.ConvictionMax.Value) * float64(cfg.CompanionCastingFloorPct)))
	return current+reserveCost+floor <= c.ConvictionMax.Value
}
