package mobs

import (
	"fmt"
	"maps"
	"math"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/conversations"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/facts"
	"github.com/GoMudEngine/GoMud/internal/llm"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/relationships"
	"gopkg.in/yaml.v2"

	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
)

var (
	instanceCounter int          = 0
	mobs                         = map[int]*Mob{}
	mobsMu          sync.RWMutex // guards mobs + allMobNames
	allMobNames     = []string{}

	mobInstances   = map[int]*Mob{}
	mobInstancesMu sync.RWMutex // guards mobInstances + instanceCounter

	mobNameCache   = map[MobId]string{}
	mobNameCacheMu sync.RWMutex

	recentlyDied   = map[int]int{}
	recentlyDiedMu sync.RWMutex
)

type ItemTrade struct {
	AcceptedItemIds []int         `yaml:"accepteditemids,omitempty,flow"` // Must provide every item id in this list.
	AcceptedGold    int           `yaml:"acceptedgold,omitempty,flow"`    // Must provide at least this much gold.
	PrizeItemIds    []int         `yaml:"prizeitemids,omitempty,flow"`    // Will give these items in exchange.
	PrizeBuffIds    []int         `yaml:"prizebuffids,omitempty,flow"`    // Will give these buffs in exchange.
	PrizeRoomId     int           `yaml:"prizeroomid,omitempty,flow"`     // Will move player to this room in exchange.
	PrizeQuestIds   []string      `yaml:"prizequestids,omitempty,flow"`   // What quest id's will be awarded?
	PrizeGold       int           `yaml:"prizegold,omitempty,flow"`       // How much gold are they given?
	PrizeCommands   []string      `yaml:"prizecommands,omitempty,flow"`   // What commands will be executed?
	GivenItems      map[int][]int `yaml:"-"`                              // key = userId, value = Items given. Should only contain items from AcceptedItemIds
	GivenGold       map[int]int   `yaml:"-"`                              // key = userId, value = how much gold is given
}

type MobForHire struct {
	MobId    MobId
	Price    int
	Quantity int
}
type MobId int // Creating a custom type to help prevent confusion over MobId and MobInstanceId

// RelationshipYAMLEntry is the per-edge authoring shape on a mob
// template's `relationships:` field. Consumed by the relationships
// package at startup.
type RelationshipYAMLEntry struct {
	To      int    `yaml:"to"`
	Type    string `yaml:"type"`
	Subtype string `yaml:"subtype,omitempty"`
}

type Mob struct {
	MobId           MobId
	Zone            string   `yaml:"zone,omitempty"`
	StatPool        int      `yaml:"statpool,omitempty"` // Stat points randomly distributed across stats on spawn
	ItemDropChance  int      // chance in 100
	LootPool        []int    `yaml:"loot_pool,omitempty"`     // Item IDs for instance loot generation
	ActivityLevel   int      `yaml:"activitylevel,omitempty"` // 1-100%
	InstanceId      int      `yaml:"-"`
	HomeRoomId      int      `yaml:"-"`
	// legacyHostile is the backward-compat YAML field. Loaders read `hostile:`
	// and copy to AutoAggro in Validate(). New YAML should use `auto_aggro: true`.
	// Future cleanup: bulk-rename hostile: → auto_aggro: in mob YAMLs, then
	// remove this field.
	legacyHostile bool `yaml:"hostile,omitempty"`
	PackFleeImmune  bool     `yaml:"pack_flee_immune,omitempty"` // if true, won't flee when packmates die
	LastIdleCommand uint8    `yaml:"-"`                          // Track what hte last used idlecommand was
	BoredomCounter  uint8    `yaml:"-"`                          // how many rounds have passed since this mob has seen a player
	Groups          []string // What group do they identify with? Helps with teamwork
	FoldAnchorRoom  int      `yaml:"fold_anchor_room,omitempty"` // Spawn-time fold-recall anchor (room ID)
	// Pack-combat routine (v2-ready — see docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md).
	// Freeform string compared with equality to other mobs' Routine for pack
	// identification. Mobs without a routine don't participate in packs.
	Routine string `yaml:"routine,omitempty"`

	// Other routine strings this mob also reacts to. Example: a bandit
	// lookout with routine "watch_north_road" might list "bandit_camp_guard"
	// here so it receives the camp's call-for-help.
	RoutineLinks []string `yaml:"routine_links,omitempty"`

	Hates              []string       `yaml:"hates,omitempty"`        // What NPC groups or races do they hate and probably fight if encountered?
	IdleCommands       []string       `yaml:"idlecommands,omitempty"` // Commands they may do while idle (not in combat)
	AngryCommands      []string       // randomly chosen to queue when they are angry/entering combat.
	CombatCommands     []string       `yaml:"combatcommands,omitempty"`    // Commands they may do while in combat
	AIProfile          string         `yaml:"aiprofile,omitempty"`         // Combat AI profile: "default", "aggressive", "defensive", "grappler", "brawler", "tactical" (Stage 8.9)
	SpecialMoveChance  int            `yaml:"specialmovechance,omitempty"` // Base % to use special moves (0-100) (Stage 8.9)
	MovePreferences    map[string]int `yaml:"movepreferences,omitempty"`   // Custom weights per move (Stage 8.9)
	Character          characters.Character
	MaxWander          int             `yaml:"maxwander,omitempty"`           // Max rooms to wander from home
	WanderCount        int             `yaml:"-"`                             // How many times this mob has wandered
	PreventIdle        bool            `yaml:"-"`                             // Whether they can't possibly be idle
	ScriptTag          string          `yaml:"scripttag"`                     // Script for this mob: mobs/frostfang/scripts/{mobId}-{mobname}-{ScriptTag}.js
	QuestFlags         []string        `yaml:"questflags,omitempty,flow"`     // What quest flags are set on this mob?
	BuffIds            []int           `yaml:"buffids,omitempty"`             // Buff Id's this mob always has upon spawn
	LLMProfile         *llm.LLMProfile `yaml:"llmprofile,omitempty"`          // Optional LLM-driven dialogue profile
	Archetype          string          `yaml:"archetype,omitempty"`           // "fighting", "casting", or "" (default even distribution)
	DefaultDisposition int             `yaml:"default_disposition,omitempty"` // Per-NPC starting disposition score on the [-100, +100] scale; 0 means neutral. Used by internal/opinions to seed first-time interactions and as the asymptote for decay.
	CombatMemory   *CombatMemory `yaml:"-"` // Runtime combat memory (not persisted)
	SpawnMutations          []string            `yaml:"spawnmutations,omitempty,flow"`     // Mutations always granted at spawn (Phase 24.3)
	MutationChance          int                 `yaml:"mutationchance,omitempty"`          // % chance to gain 1 random bonus mutation on spawn (Phase 24.3)
	CharmImmune             bool                `yaml:"charm_immune,omitempty"`            // If true, charm spells cannot affect this mob
	NonCombatant            bool                `yaml:"non_combatant,omitempty"`           // If true, cannot be attacked, stolen from, or aggroed. Synced → Character.NonCombatant in Validate().
	// AutoAggro indicates whether this mob auto-attacks players on sight.
	// Replaces the conflated Hostile field's auto-attack semantic.
	// Loaded from YAML field `auto_aggro:`; if absent, backward-compat
	// copy from the legacy `hostile:` field in Validate() (sunset Task 18).
	AutoAggro               bool                `yaml:"auto_aggro,omitempty"`
	PlayerAttackImmune      bool                `yaml:"player_attack_immune,omitempty"`    // If true, players cannot attack this mob (but mob can still fight)
	BuysGeneral             bool                `yaml:"buys_general,omitempty"`            // Whether this merchant buys misc goods
	Crafter                 bool                `yaml:"crafter,omitempty"`                 // Whether this mob crafts autonomously (Stage 38.5.4)
	CrafterSkill            string              `yaml:"crafterskill,omitempty"`            // Craft skill used (e.g. "blacksmithing")
	CrafterRecipeIds        []string            `yaml:"crafterrecipeids,omitempty"`        // Recipe IDs this mob can craft
	CrafterRestockMaterials []int               `yaml:"crafterrestockmaterials,omitempty"` // Item IDs restocked periodically
	ShopCraftSupport        string              `yaml:"craft_support,omitempty"`           // Crafting discipline this shop supports (one of shops.ValidCraftSupports)

	// ── Stage 3.4: spawn-time overrides for special mobs (wagons, statues, etc.) ──
	CarryCapacityOverride float64 `yaml:"carry_capacity,omitempty"`       // overrides Strength-derived calc when > 0
	HealthMaxOverride     int     `yaml:"health_max,omitempty"`           // overrides Vitality-derived calc when > 0
	StaminaMaxOverride    int     `yaml:"stamina_max,omitempty"`          // overrides default calc when > 0
	CorpseName            string  `yaml:"corpse_name,omitempty"`          // overrides "<Name> corpse" rendering when set
	CorpseDescription     string  `yaml:"corpse_description,omitempty"`   // overrides default corpse look-text when set
	StockMultiplier       float64 `yaml:"stock_multiplier,omitempty"`     // shop stock-cap scale; default 1.0 (treated as 1.0 by EffectiveMaxStock if unset)
	HideEquipmentSlots    bool    `yaml:"hide_equipment_slots,omitempty"` // suppresses the Equipment block in look-mob render. For object mobs (wagons, statues) where equipment slots don't make sense.

	PackBonusTotal          int    `yaml:"-"` // Total training points from pack scaling (Stage 38.5.3)
	PackAlphaId             int    `yaml:"-"` // InstanceId of alpha this mob follows (0 = none)
	IsPackAlpha             bool   `yaml:"-"` // Whether this mob is currently the pack alpha
	ScatterRounds           int    `yaml:"-"` // Rounds remaining where mob skips wander (after alpha death)
	crafterLastRestockRound uint64 // Last round materials were restocked (transient)
	BehaviorArchetype       string `yaml:"behavior_archetype,omitempty"` // Archetype name (e.g., "melee_self_buff") — resolved to behaviors/archetypes/<name>.yaml if per-mob tree absent.
	BTreeState              any    `yaml:"-"`                            // Behavior tree per-instance state (*behaviortree.BehaviorState)
	tempDataStore           map[string]any
	conversationId          int              // Identifier of conversation currently involved in.
	Path                    PathQueue        `yaml:"-"` // a pre-calculated path the mob is following.
	lastCommandTurn         uint64           // The last turn a command was scheduled for
	playersAttacked         map[int]struct{} // all players this mob has attacked at some point
	Relationships           []RelationshipYAMLEntry `yaml:"relationships,omitempty"` // Authored relationship edges; consumed by relationships.LoadFromMobs at startup.
	KnowsFacts              []string                `yaml:"knows_facts,omitempty"`   // Fact IDs this mob knows at startup; consumed by facts.LoadFromMobs at startup.
}

func MobInstanceExists(instanceId int) bool {
	mobInstancesMu.RLock()
	defer mobInstancesMu.RUnlock()
	_, ok := mobInstances[instanceId]
	return ok
}

// IsNonCombatant returns true if the mob is flagged as a non-combatant
// (shopkeepers, quest NPCs, etc.) that cannot be attacked or stolen from.
//
// Delegates to Character.NonCombatant (the canonical field after Task 9).
// Also checks the legacy Mob.NonCombatant field during the migration window
// (sunset in Task 18) so that test fixtures that set the field directly
// without calling Validate() continue to work correctly.
func (m *Mob) IsNonCombatant() bool {
	return m.Character.NonCombatant || m.NonCombatant
}

// GetZone returns the mob's zone name.
func (m *Mob) GetZone() string {
	return m.Zone
}

// GetName returns the mob's display name.
func (m *Mob) GetName() string {
	return m.Character.Name
}

// Gets a copy of all mob info
func GetAllMobInfo() []Mob {
	mobsMu.RLock()
	defer mobsMu.RUnlock()
	ret := make([]Mob, 0, len(mobs))
	for _, m := range mobs {
		ret = append(ret, *m)
	}
	return ret
}

// AllMobTemplates returns pointers to every loaded mob template. Intended for
// startup validators that need to inspect template fields without copying.
// Callers must not mutate the returned pointers.
func AllMobTemplates() []*Mob {
	mobsMu.RLock()
	defer mobsMu.RUnlock()
	ret := make([]*Mob, 0, len(mobs))
	for _, m := range mobs {
		ret = append(ret, m)
	}
	return ret
}

func GetAllMobNames() []string {
	mobsMu.RLock()
	defer mobsMu.RUnlock()
	out := make([]string, len(allMobNames))
	copy(out, allMobNames)
	return out
}

func TrackRecentDeath(instanceId int) {
	recentlyDiedMu.Lock()
	recentlyDied[instanceId] = int(util.GetRoundCount())
	recentlyDiedMu.Unlock()
}

func RecentlyDied(instanceId int) bool {

	recentlyDiedMu.Lock()
	defer recentlyDiedMu.Unlock()

	if len(recentlyDied) > 30 {
		roundNow := int(util.GetRoundCount())
		for k, v := range recentlyDied {
			if roundNow-v > 15 {
				delete(recentlyDied, k)
			}
		}
	}

	_, ok := recentlyDied[instanceId]

	return ok
}

func MobIdByName(mobName string) MobId {

	mobsMu.RLock()
	names := make([]string, len(allMobNames))
	copy(names, allMobNames)
	mobsMu.RUnlock()

	match, partial := util.FindMatchIn(mobName, names...)
	if match == "" {
		match = partial
	}
	if match == "" {
		return 0
	}

	mobsMu.RLock()
	defer mobsMu.RUnlock()

	for _, m := range mobs {
		if m.Character.Name == match {
			return m.MobId
		}
	}

	for _, m := range mobs {
		if strings.HasPrefix(m.Character.Name, match) {
			return m.MobId
		}
	}

	for _, m := range mobs {
		if strings.Contains(m.Character.Name, match) {
			return m.MobId
		}
	}

	return 0
}

// NewMobById creates a new mob instance from the template `mobId`,
// placed at `homeRoomId`. If a saved instance file exists for this
// (mobId, zone, mobName, homeRoomId) tuple, its progression is loaded
// onto the new mob. This is the constructor for organic world spawns.
//
// Companion-spawning callers (summon / raise / conjure / charm-respawn)
// must use NewMobByIdFresh instead — their progression lives on
// CompanionInfo, not on the file system. See
// docs/superpowers/specs/2026-04-21-summons-dont-persist-design.md.
func NewMobById(mobId MobId, homeRoomId int, forceStatPool ...int) *Mob {
	return newMobByIdInternal(mobId, homeRoomId, false, forceStatPool...)
}

// NewMobByIdFresh creates a mob instance from the template without
// reading any saved progression file. Used by companion-spawning code
// paths (summon / raise / conjure / login-respawn of companions /
// companion-vending NPCs / admin suicide-vanish). Template defaults
// (including random stat pool distribution) apply as if no instance
// file existed.
func NewMobByIdFresh(mobId MobId, homeRoomId int, forceStatPool ...int) *Mob {
	return newMobByIdInternal(mobId, homeRoomId, true, forceStatPool...)
}

func newMobByIdInternal(mobId MobId, homeRoomId int, skipInstanceLoad bool, forceStatPool ...int) *Mob {

	mobsMu.RLock()
	m, ok := mobs[int(mobId)]
	mobsMu.RUnlock()

	if ok {

		mobInstancesMu.Lock()
		instanceCounter++
		newInstanceId := instanceCounter
		mobInstancesMu.Unlock()

		mob := *m // Make a copy of the mob
		mob.InstanceId = newInstanceId

		mob.HomeRoomId = homeRoomId
		mob.Character.RoomId = homeRoomId
		mob.Character.IsMob = true
		mob.Character.PlayerDamage = make(map[int]int)

		// Deep copy maps to prevent shared state with template.
		// Go shallow copy shares map backing data — mutations to an
		// instance's skills/spellbook would contaminate the template.
		if mob.Character.Skills != nil {
			skillsCopy := make(map[string]int, len(mob.Character.Skills))
			maps.Copy(skillsCopy, mob.Character.Skills)
			mob.Character.Skills = skillsCopy
		}
		if mob.Character.SpellBook != nil {
			spellCopy := make(map[string]int, len(mob.Character.SpellBook))
			maps.Copy(spellCopy, mob.Character.SpellBook)
			mob.Character.SpellBook = spellCopy
		}

		// Stage 38.4: Try to load a saved instance (progression data from disk).
		// If found, apply saved training values instead of randomizing.
		var savedInstance *MobInstanceData
		if !skipInstanceLoad {
			savedInstance = LoadMobInstance(mob.MobId, mob.Zone, mob.Character.Name, homeRoomId)
		}
		if savedInstance != nil {
			// Restore saved progression
			mob.Character.Stats.Strength.Training = savedInstance.StrengthTraining
			mob.Character.Stats.Dexterity.Training = savedInstance.DexterityTraining
			mob.Character.Stats.Perception.Training = savedInstance.PerceptionTraining
			mob.Character.Stats.Vitality.Training = savedInstance.VitalityTraining
			mob.Character.Stats.Willpower.Training = savedInstance.WillpowerTraining
			mob.Character.Stats.Charisma.Training = savedInstance.CharismaTraining
			if savedInstance.Skills != nil {
				mob.Character.Skills = savedInstance.Skills
			}
			if savedInstance.SkillUseCount != nil {
				mob.Character.SkillUseCount = savedInstance.SkillUseCount
			}
			if savedInstance.StatUseCount != nil {
				mob.Character.StatUseCount = savedInstance.StatUseCount
			}
			if savedInstance.Mutations != nil {
				mob.Character.Mutations = savedInstance.Mutations
			}
			mob.Character.MutationProgress = savedInstance.MutationProgress
		} else {
			// No saved instance — randomize stat pool as usual
			statPool := mob.StatPool
			if len(forceStatPool) > 0 && forceStatPool[0] > 0 {
				statPool = forceStatPool[0]
			}
			// Distribute stat pool across training stats using archetype weighting
			for i := 0; i < statPool; i++ {
				var statIdx int
				switch mob.Archetype {
				case "fighting":
					// 80% physical (Str/Dex/Vit), 20% mental (Per/Wil/Cha)
					if util.Rand(100) < 80 {
						statIdx = util.Rand(3) // 0=Str, 1=Dex, 2=Vit
					} else {
						statIdx = 3 + util.Rand(3) // 3=Per, 4=Wil, 5=Cha
					}
				case "casting":
					// 20% physical (Str/Dex/Vit), 80% mental (Per/Wil/Cha)
					if util.Rand(100) < 20 {
						statIdx = util.Rand(3)
					} else {
						statIdx = 3 + util.Rand(3)
					}
				case "tank":
					// Tank/taunter: 25% Cha (taunt), 20% Vit (HP buffer),
					// 15% each Str/Dex/Wil, 10% Per.
					r := util.Rand(100)
					switch {
					case r < 25:
						statIdx = 5 // Charisma
					case r < 45:
						statIdx = 2 // Vitality
					case r < 60:
						statIdx = 0 // Strength
					case r < 75:
						statIdx = 1 // Dexterity
					case r < 90:
						statIdx = 4 // Willpower
					default:
						statIdx = 3 // Perception
					}
				default:
					// Even distribution across all 6 stats
					statIdx = util.Rand(6)
				}
				switch statIdx {
				case 0:
					mob.Character.Stats.Strength.Training++
				case 1:
					mob.Character.Stats.Dexterity.Training++
				case 2:
					mob.Character.Stats.Vitality.Training++
				case 3:
					mob.Character.Stats.Perception.Training++
				case 4:
					mob.Character.Stats.Willpower.Training++
				case 5:
					mob.Character.Stats.Charisma.Training++
				}
			}
		}
		mob.Character.Validate()
		// Stage 3.4: apply override fields from mob YAML if set.
		// Must run after Validate() (which computes stat-derived
		// defaults) and before the Health/Conviction assignments
		// below so the override propagates to current pool values.
		characters.ApplyMobOverrides(
			&mob.Character,
			mob.HealthMaxOverride,
			mob.StaminaMaxOverride,
			mob.CarryCapacityOverride,
		)
		mob.Character.Health = mob.Character.HealthMax.Value
		mob.Character.Stamina = mob.Character.StaminaMax.Value
		mob.Character.Conviction = mob.Character.ConvictionMax.Value

		mob.Character.SetPermaBuffs(mob.BuffIds)

		mob.Character.Buffs = buffs.New()

		// Deep copy item slices to prevent shared backing array with template.
		// Without this, giving items to a mob instance can contaminate the
		// template, causing all future spawns to carry the given items.
		if len(mob.Character.Items) > 0 {
			itemsCopy := make([]items.Item, len(mob.Character.Items))
			copy(itemsCopy, mob.Character.Items)
			mob.Character.Items = itemsCopy
		}
		if len(mob.Character.ComponentItems) > 0 {
			compCopy := make([]items.Item, len(mob.Character.ComponentItems))
			copy(compCopy, mob.Character.ComponentItems)
			mob.Character.ComponentItems = compCopy
		}
		if len(mob.Character.PotionItems) > 0 {
			potCopy := make([]items.Item, len(mob.Character.PotionItems))
			copy(potCopy, mob.Character.PotionItems)
			mob.Character.PotionItems = potCopy
		}

		for idx := range mob.Character.Items {
			mob.Character.Items[idx].Validate()
		}

		// Stage 8.9: Initialize AI defaults
		if mob.AIProfile == "" {
			mob.AIProfile = "default"
		}
		if mob.SpecialMoveChance == 0 {
			mob.SpecialMoveChance = 30 // 30% default chance to use special moves
		}

		// Phase 24.3 / 2.5: Apply spawn mutations with body-parts gate
		// Hoist the species lookup to avoid redundant calls.
		sp := species.GetSpecies(mob.Character.SpeciesId)

		if len(mob.SpawnMutations) > 0 {
			if mob.Character.Mutations == nil {
				mob.Character.Mutations = make(map[string]int)
			}
			for _, mutId := range mob.SpawnMutations {
				spec := mutations.GetMutation(mutId)
				if spec == nil {
					mudlog.Warn("MobSpawn",
						"msg", "unknown mutation id in SpawnMutations",
						"mobId", mob.MobId,
						"mutation", mutId)
					continue
				}
				if !spec.CanApplyTo(sp) {
					// Defensive: nil-guard species fields in warning log
					fields := []any{
						"msg", "mutation requirements not met by species",
						"mobId", mob.MobId,
						"mutation", mutId,
						"requires", spec.RequiresBodyParts,
					}
					if sp != nil {
						fields = append(fields,
							"species", sp.Name,
							"species_body_parts", sp.BodyParts)
					}
					mudlog.Warn("MobSpawn", fields...)
					continue
				}
				mob.Character.Mutations[mutId] = 1
			}
		}
		// Phase 24.3: Roll for random bonus mutation
		if mob.MutationChance > 0 && util.Rand(100) < mob.MutationChance {
			if mob.Character.Mutations == nil {
				mob.Character.Mutations = make(map[string]int)
			}
			pool := mutations.GetWeightedPool(mob.Character.Mutations, sp)
			if len(pool) > 0 {
				mutId := mutations.RollAcquisition(pool)
				mob.Character.Mutations[mutId] = 1
			}
		}
		// Phase 2.5: Merge species intrinsic mutations after both curated
		// and random-roll paths have resolved.
		mob.Character.ApplyIntrinsicMutations(sp)

		mob.Character.Equipment.Weapon.Validate()
		mob.Character.Equipment.Offhand.Validate()
		mob.Character.Equipment.Head.Validate()
		mob.Character.Equipment.Neck.Validate()
		mob.Character.Equipment.Body.Validate()
		mob.Character.Equipment.Belt.Validate()
		mob.Character.Equipment.Gloves.Validate()
		mob.Character.Equipment.Ring.Validate()
		mob.Character.Equipment.Legs.Validate()
		mob.Character.Equipment.Feet.Validate()

		mob.Validate()
		mob.Character.Validate(true)

		// Stage 3.0d: pre-stamp fold-recall anchor if the YAML template set one.
		stampFoldAnchor(&mob.Character, m.FoldAnchorRoom)

		// Register the mob's shop with the living economy system if applicable.
		// Must happen after HomeRoomId and Zone are set (they key the shop store).
		RegisterMobShop(&mob)

		// Save the mob instance
		mobInstancesMu.Lock()
		mobInstances[mob.InstanceId] = &mob
		mobInstancesMu.Unlock()

		return &mob
	}
	return nil
}

// stampFoldAnchor pre-stamps a mob's fold-recall anchor in MiscData. Called
// from newMobByIdInternal at spawn time. No-op when anchorRoom <= 0 so the
// default YAML value (omitted field) doesn't create a spurious anchor at
// room 0. Stage 3.0d.
func stampFoldAnchor(c *characters.Character, anchorRoom int) {
	if anchorRoom <= 0 {
		return
	}
	c.SetMiscData("fold-anchor-room", anchorRoom)
}

func GetMobSpec(mobId MobId) *Mob {
	mobsMu.RLock()
	defer mobsMu.RUnlock()
	if m, ok := mobs[int(mobId)]; ok {
		mob := *m // Make a copy of the mob
		return &mob
	}
	return nil
}

func GetInstance(instanceId int) *Mob {
	mobInstancesMu.RLock()
	defer mobInstancesMu.RUnlock()
	if m, ok := mobInstances[instanceId]; ok {
		return m
	}
	return nil
}

func GetAllMobInstanceIds() []int {
	mobInstancesMu.RLock()
	defer mobInstancesMu.RUnlock()
	ids := make([]int, 0, len(mobInstances))
	for id := range mobInstances {
		ids = append(ids, id)
	}
	return ids
}

func DestroyInstance(instanceId int) {
	mobInstancesMu.Lock()
	delete(mobInstances, instanceId)
	mobInstancesMu.Unlock()
}

func (m *Mob) ShorthandId() string {
	return fmt.Sprintf(`#%d`, m.InstanceId)
}

func (m *Mob) AddBuff(buffId int, source string) {

	events.AddToQueue(events.Buff{
		MobInstanceId: m.InstanceId,
		BuffId:        buffId,
		Source:        source,
	})

}

func (m *Mob) PlayerAttacked(userId int) {
	if m.playersAttacked == nil {
		m.playersAttacked = map[int]struct{}{}
	}
	m.playersAttacked[userId] = struct{}{}
}

func (m *Mob) HasAttackedPlayer(userId int) bool {
	if m.playersAttacked == nil {
		return false
	}
	_, ok := m.playersAttacked[userId]
	return ok
}

func (m *Mob) InConversation() bool {
	return m.conversationId > 0
}

func (m *Mob) SetConversation(id int) {
	m.conversationId = id
}

func (m *Mob) Converse() {

	mobInst1, mobInst2, actions := conversations.GetNextActions(m.conversationId)

	var mob1 *Mob = nil
	var mob2 *Mob = nil

	if mobInst1 == int(m.InstanceId) {
		mob1 = m
		mob2 = GetInstance(mobInst2)
	} else {
		mob1 = GetInstance(mobInst1)
		mob2 = m
	}

	if mob1 == nil || mob2 == nil {
		conversations.Destroy(m.conversationId)
		if mob1 != nil {
			mob1.SetConversation(0)
		}
		if mob2 != nil {
			mob2.SetConversation(0)
		}
		return
	}

	for _, act := range actions {
		if len(act) >= 3 {

			target := act[0:3]
			cmd := act[3:]

			cmd = strings.ReplaceAll(cmd, ` #1 `, ` `+mob1.ShorthandId()+` `)
			cmd = strings.ReplaceAll(cmd, ` #2 `, ` `+mob2.ShorthandId()+` `)

			if target == `#1 ` {
				mob1.Command(cmd)
			} else {
				mob2.Command(cmd, 1)
			}
		}
	}

	if conversations.IsComplete(m.conversationId) {
		conversations.Destroy(m.conversationId)
		mob1.SetConversation(0)
		mob2.SetConversation(0)
		return
	}
}

// GetLastCommandTurn returns the turn at which the mob's last scheduled command will execute.
func (m *Mob) GetLastCommandTurn() uint64 {
	return m.lastCommandTurn
}

// Cause the mob to basically wait and do nothing for x seconds
func (m *Mob) Sleep(seconds int) {
	m.Command(`noop`, float64(seconds))
}

func (m *Mob) Command(inputTxt string, waitSeconds ...float64) {

	readyTurn := util.GetTurnCount()
	turnDelay := uint64(0)

	// m.lastCommandTurn is used so that subsequent calls to Command()
	// are scheduled from this period forward.
	// If it's been long enough that the current turn has surpassed the lastCommandTurn, we failover to that.
	if readyTurn > m.lastCommandTurn {
		m.lastCommandTurn = readyTurn
	} else {
		readyTurn = m.lastCommandTurn
	}

	if len(waitSeconds) > 0 {
		turnDelay = uint64(float64(configs.GetTimingConfig().SecondsToTurns(1)) * waitSeconds[0])
	}

	for i, cmd := range strings.Split(inputTxt, `;`) {

		// Update lastCommandTurn to whenever this command is scheduled for
		m.lastCommandTurn = readyTurn + turnDelay + uint64(i)

		events.AddToQueue(events.Input{
			MobInstanceId: m.InstanceId,
			InputText:     cmd,
			ReadyTurn:     m.lastCommandTurn,
		})

	}

}

func (m *Mob) HasShop() bool {
	return len(m.Character.Shop) > 0
}

// GetMobId satisfies shops.ShopBearingMob — returns the template mob ID as int.
func (m *Mob) GetMobId() int {
	return int(m.MobId)
}

// IsCrafter satisfies shops.ShopBearingMob — returns whether this mob crafts
// autonomously.
func (m *Mob) IsCrafter() bool {
	return m.Crafter
}

// GetShopCraftSupport satisfies shops.ShopBearingMob — returns the
// craft_support tag for this mob's shop.
func (m *Mob) GetShopCraftSupport() string {
	return m.ShopCraftSupport
}

func (m *Mob) IsTameable() bool {
	if m.HasShop() {
		return false
	}
	if len(m.ScriptTag) > 0 {
		return false
	}
	if r := species.GetSpecies(m.Character.SpeciesId); r != nil {
		if !r.Tameable {
			return false
		}
	}
	return true
}

func (m *Mob) SetTempData(key string, value any) {

	if m.tempDataStore == nil {
		m.tempDataStore = make(map[string]any)
	}

	if value == nil {
		delete(m.tempDataStore, key)
		return
	}
	m.tempDataStore[key] = value
}

func (m *Mob) GetTempData(key string) any {

	if m.tempDataStore == nil {
		m.tempDataStore = make(map[string]any)
	}

	if value, ok := m.tempDataStore[key]; ok {
		return value
	}
	return nil
}

func (m *Mob) Despawns() bool {
	if m.HasShop() {
		return false
	}
	// Charmed companions should not despawn from boredom.
	if m.Character.IsCharmed() {
		return false
	}
	return true
}

// IsEssential returns true when this mob drives a living-economy system
// (foragers, caravan crew, or shopkeepers). Essential mobs persist in their
// rooms so their BTree state survives unattended periods — the room manager
// skips unloading rooms that contain them. Memory cost is small: typically
// fewer than 20–50 rooms pinned across the world at any moment.
func (m *Mob) IsEssential() bool {
	// Shopkeepers must stay alive so TickMobShopRestock, TickMobCraft, and
	// caravan/forager deliveries all have a live mob instance to work with.
	if m.HasShop() {
		return true
	}
	for _, g := range m.Groups {
		if g == "forager" || g == "caravan" {
			return true
		}
	}
	return false
}

func (m *Mob) GetSellPrice(item items.Item) int {

	if item.IsSpecial() {
		return 0
	}

	itemType := item.GetSpec().Type
	itemSubtype := item.GetSpec().Subtype
	value := 0
	likesType := false
	likesSubtype := false
	newAddition := true
	priceScale := 0.0

	currentSaleItems := m.Character.Shop.GetInstock()

	for _, stockItm := range currentSaleItems {
		if stockItm.ItemId == 0 {
			continue
		}

		if stockItm.ItemId == item.ItemId { // If it's in stock, we can set everyting and break out
			newAddition = false // already stocking this item
			likesType = true
			likesSubtype = true
			value = stockItm.Price
			// Scale down amount willing to pay based on how many there are already in stock
			priceScale = 1.0 - (float64(stockItm.Quantity) / 20)
			break
		}

		tmpItm := items.New(stockItm.ItemId)
		if tmpItm.ItemId == 0 {
			continue
		}

		if !likesType && tmpItm.GetSpec().Type == itemType {
			likesType = true
			priceScale += 0.5
		}

		if !likesSubtype && tmpItm.GetSpec().Subtype == itemSubtype {
			likesSubtype = true
			priceScale += 0.5
		}
	}

	// If this is a new addition, don't allow more than 20 varieites
	if newAddition && len(currentSaleItems) >= 20 {
		return 0
	}

	if value == 0 {
		value = item.GetSpec().Value
	}

	if priceScale < 0 {
		priceScale = 0
	} else if priceScale > 100 {
		priceScale = 100
	}

	priceScale *= .25 // Can never be more than 25% value of object

	return int(math.Ceil(float64(value) * priceScale))
}

func (r *Mob) HatesSpecies(raceName string) bool {
	return slices.Contains(r.Hates, strings.ToLower(raceName))
}

func (r *Mob) HatesMob(m *Mob) bool {
	if r.MobId == m.MobId {
		return false // Can't hate exact same as self
	}

	// Check hates list against target's groups first — group hatred
	// overrides species alliance (a warden hates bandits even if both human)
	if r.hatesAnyGroup(m.Groups) {
		return true
	}

	// Same species = ally, never hate
	if r.Character.SpeciesId > 0 &&
		r.Character.SpeciesId == m.Character.SpeciesId {
		return false
	}

	// Check hates list against target's species name
	mRace := species.GetSpecies(m.Character.SpeciesId)
	raceName := strings.ToLower(mRace.Name)
	for _, hateName := range r.Hates {
		if hateName == `*` {
			return true
		}
		if hateName == raceName {
			return true
		}
	}
	return false
}

func (m *Mob) GetAngryCommand() string {

	// First check if the mob has a specific action
	if len(m.AngryCommands) > 0 {
		return m.AngryCommands[util.Rand(len(m.AngryCommands))]
	}

	// default to race based actions
	r := species.GetSpecies(m.Character.SpeciesId)
	actionCt := len(r.AngryCommands)
	if actionCt > 0 {
		return r.AngryCommands[util.Rand(actionCt)]
	}
	return ``
}

func (m *Mob) GetIdleCommand() string {

	// Always a 1 in 100 chance it will do nothing for an idle.
	// This is to prevent requiring Admins to assign an empy idlecommand to mob definitions
	// while still allowing "no idle command found" behavior to run.
	// Empty idle commands can still be defined in mobs, however.
	if util.Rand(100) == 0 {
		return ``
	}

	// First check if the mob has a specific action
	if len(m.IdleCommands) > 0 {
		return m.IdleCommands[util.Rand(len(m.IdleCommands))]
	}

	return ``
}

func (r *Mob) ConsidersAnAlly(m *Mob) bool {

	// Same mob type always allies
	if m.MobId == r.MobId {
		return true
	}

	// If either mob hates any of the other's groups, they are NOT allies
	// regardless of species. A warden should never ally with a bandit.
	if r.hatesAnyGroup(m.Groups) || m.hatesAnyGroup(r.Groups) {
		return false
	}

	// Same species = allies (SpeciesId 0 is unset/ghostly spirit, skip)
	if r.Character.SpeciesId > 0 &&
		r.Character.SpeciesId == m.Character.SpeciesId {
		return true
	}

	return false
}

// hatesAnyGroup returns true if this mob's Hates list includes any of the given groups.
func (r *Mob) hatesAnyGroup(groups []string) bool {
	for _, hate := range r.Hates {
		for _, grp := range groups {
			if strings.EqualFold(hate, grp) {
				return true
			}
		}
	}
	return false
}

func (r *Mob) Id() int {
	return int(r.MobId)
}

func (r *Mob) Validate() error {

	if r.ActivityLevel < 1 {
		r.ActivityLevel = 10
	} else if r.ActivityLevel > 100 {
		r.ActivityLevel = 100
	}

	// Sync legacy Mob.NonCombatant → Character.NonCombatant.
	// The YAML field lives on Mob for backward compat; Character is the
	// canonical home going forward (consumed by CombatPhase veto, Task 10).
	if r.NonCombatant {
		r.Character.NonCombatant = true
	}

	// Backward-compat: populate AutoAggro from the legacy `hostile:` YAML field
	// if AutoAggro wasn't set explicitly. New mob YAMLs should use `auto_aggro: true`.
	if r.legacyHostile && !r.AutoAggro {
		r.AutoAggro = true
	}

	r.Character.Validate()

	return nil
}

func (m *Mob) Filename() string {
	mobNameCacheMu.RLock()
	name, ok := mobNameCache[m.MobId]
	mobNameCacheMu.RUnlock()
	if ok {
		return fmt.Sprintf("%d-%s.yaml", m.Id(), util.ConvertForFilename(name))
	}
	// Failover to character name
	filename := util.ConvertForFilename(m.Character.Name)
	return fmt.Sprintf("%d-%s.yaml", m.Id(), filename)
}

func (m *Mob) Filepath() string {
	zone := ZoneNameSanitize(m.Zone)
	return util.FilePath(zone, `/`, m.Filename())
}

func (r *Mob) Save() error {

	fileName := r.Filename()

	bytes, err := yaml.Marshal(r)
	if err != nil {
		return err
	}

	saveFilePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `mobs`, `/`, fmt.Sprintf("%s.yaml", fileName))

	err = os.WriteFile(saveFilePath, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func ZoneNameSanitize(zone string) string {
	if zone == "" {
		return ""
	}
	// Convert spaces to underscores
	zone = strings.ReplaceAll(zone, " ", "_")
	// Lowercase it all, and add a slash at the end
	return strings.ToLower(zone)
}

// file self loads due to init()
func LoadDataFiles() {

	start := time.Now()

	dataPath := configs.GetFilePathsConfig().DataFiles.String() + `/mobs`
	tmpMobs, err := fileloader.LoadAllFlatFiles[int, *Mob](dataPath)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath))
	}

	// Build the derived caches outside the lock (no contention risk during startup).
	tmpNames := make([]string, 0, len(tmpMobs))
	tmpNameCache := make(map[MobId]string, len(tmpMobs))
	for _, mob := range tmpMobs {
		mob.Character.CacheDescription()
		tmpNames = append(tmpNames, mob.Character.Name)
		tmpNameCache[mob.MobId] = mob.Character.Name
	}

	mobsMu.Lock()
	mobs = tmpMobs
	allMobNames = tmpNames
	mobsMu.Unlock()

	mobNameCacheMu.Lock()
	mobNameCache = tmpNameCache
	mobNameCacheMu.Unlock()

	mudlog.Info("mobs.LoadDataFiles()", "loadedCount", len(tmpMobs), "Time Taken", time.Since(start))

	// Populate the relationships graph from authored YAML edges.
	mobsMu.RLock()
	edges := make([]relationships.MobEdges, 0, len(mobs))
	for id, spec := range mobs {
		if len(spec.Relationships) == 0 {
			continue
		}
		conv := make([]relationships.EdgeInput, 0, len(spec.Relationships))
		for _, e := range spec.Relationships {
			conv = append(conv, relationships.EdgeInput{
				To:      e.To,
				Type:    relationships.Type(e.Type),
				Subtype: e.Subtype,
			})
		}
		edges = append(edges, relationships.MobEdges{
			MobId: id,
			Edges: conv,
		})
	}
	mobsMu.RUnlock()

	relationships.LoadFromMobs(edges, func(mobId int) bool {
		mobsMu.RLock()
		defer mobsMu.RUnlock()
		_, ok := mobs[mobId]
		return ok
	})

	// Seed per-NPC fact awareness from authored knows_facts: declarations.
	mobsMu.RLock()
	factSeeds := make([]facts.MobAwarenessSeed, 0, len(mobs))
	for id, spec := range mobs {
		if len(spec.KnowsFacts) == 0 {
			continue
		}
		factSeeds = append(factSeeds, facts.MobAwarenessSeed{
			MobId:      id,
			MobName:    spec.Character.Name,
			KnowsFacts: spec.KnowsFacts,
		})
	}
	mobsMu.RUnlock()

	facts.LoadFromMobs(factSeeds, func(mobId int) string {
		mobsMu.RLock()
		defer mobsMu.RUnlock()
		if spec, ok := mobs[mobId]; ok {
			return spec.Character.Name
		}
		return ""
	})

}

// AuditMobNameCollisions scans loaded mob template names against the
// supplied playerNameLookup and warns on each collision. Warn-only —
// never blocks startup. Called once after LoadDataFiles completes and
// the user index is reachable. Dependency injection keeps this package
// free of a users-package import.
func AuditMobNameCollisions(playerNameLookup func(name string) (userId int, ok bool)) {
	mobsMu.RLock()
	names := make([]string, len(allMobNames))
	copy(names, allMobNames)
	mobsMu.RUnlock()

	collisions := 0
	for _, mobName := range names {
		if userId, ok := playerNameLookup(mobName); ok {
			mudlog.Warn("mob/player name collision",
				"mobName", mobName,
				"playerUserId", userId,
				"advice", "rename mob template or notify player to use rename command")
			collisions++
		}
	}
	if collisions > 0 {
		mudlog.Warn("mob name collision audit complete", "collisions", collisions)
	}
}
