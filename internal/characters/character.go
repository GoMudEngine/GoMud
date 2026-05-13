package characters

import (
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/pets"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/stats"
)

var (
	startingRace   = 0
	startingHealth = 10
	StartingRoomId = 0
	startingZone     = `Nowhere`
	defaultName      = `nameless`
)

type NameRenderFlag uint8

const (
	RenderHealth NameRenderFlag = iota
	RenderAggro
	RenderShortAdjectives
)

type Character struct {
	Name             string                         // The name of the character
	Description      string                         // A description of the character.
	Adjectives       []string                       `yaml:"adjectives,omitempty"` // Decorative text for the name of the character (e.g. "sleeping", "dead", "wounded")
	RoomId           int                            // The room id the character is in.
	Zone             string                         // The zone the character is in. The folder the room can be located in too.
	SpeciesId        int                            // Character species
	Stats            stats.Statistics               // Character stats
	Health           int                            // The health of the character
	Stamina          int                            // The stamina of the character (physical energy)
	Conviction       int                            // The conviction of the character (mental/spiritual energy)
	Toxicity         float64                        `yaml:"toxicity,omitempty"`      // Current toxicity from potions
	ActionPoints     int                            // The resevoir of action points the character has to spend on movement etc.
	Gold             int                            // The gold the character is holding
	Bank             int                            // The gold the character has in the bank
	StorageFeeLastMonth int `yaml:"storagefee_lastmonth,omitempty"` // Game month when storage fees were last charged
	Shop             Shop                           `yaml:"shop,omitempty"`          // Definition of shop services/items this character stocks (or just has at the moment)
	SpellBook        map[string]int                 `yaml:"spellbook,omitempty"`     // The spells the character has learned
	KnownRecipes     map[string]int                 `yaml:"knownrecipes,omitempty"`  // The crafting recipes the character has discovered
	Charmed          *CharmInfo                     `yaml:"-"`                       // If they are charmed, this is the info
	EverCharmed      bool                           `yaml:"-"`                       // True if this mob was ever a companion (survives dismiss)
	CharmedMobs      []int                          `yaml:"-"`                       // If they have charmed anyone, this is the list of mob instance ids
	Items            []items.Item                   `yaml:"items,omitempty"`         // The items the character is holding
	ComponentItems   []items.Item                   `yaml:"componentitems,omitempty"` // Contents of equipped component bag
	PotionItems      []items.Item                   `yaml:"potionitems,omitempty"`   // Contents of equipped potion bandolier
	Buffs            buffs.Buffs                    `yaml:"buffs,omitempty"`         // The buffs the character has active
	Equipment        Worn                           `yaml:"equipment,omitempty"`     // The equipment the character is wearing
	HealthMax        stats.StatInfo                 `yaml:"-"`                       // The maximum health of the character. Don't write to yaml since is dynamically calculated.
	StaminaMax       stats.StatInfo                 `yaml:"-"`                       // The maximum stamina of the character. Don't write to yaml since is dynamically calculated.
	ConvictionMax    stats.StatInfo                 `yaml:"-"`                       // The maximum conviction of the character. Don't write to yaml since is dynamically calculated.
	ActionPointsMax          stats.StatInfo                 `yaml:"-"`                       // The maximum actions of character. Don't write to yaml since is dynamically calculated.
	Aggro                    *Aggro                         `yaml:"-"`                       // Dont' store this. If they leave they break their aggro
	// NEW (chunk 0 Task 3): Combat Phase state machine. Future source
	// of truth for "am I in combat?" and "who am I targeting?". Lives
	// alongside Aggro during the migration window; Aggro is deleted
	// in Task 18 of the chunk-0 plan.
	CombatPhase              *combatphase.Machine           `yaml:"-"`
	CombatPosition           CombatPosition                 `yaml:"-"`                       // Current combat position (Standing/Prone/Clinched/Grounded). Don't store this.
	PositionRoundsMin        int                            `yaml:"-"`                       // Minimum rounds in current position (for Prone bash/trip, etc). Don't store this.
	GrappleControllerId      int                            `yaml:"-"`                       // UserId or MobInstanceId of grapple controller (0 = none, Stage 8.2+). Don't store this.
	Conditions               []CombatCondition              `yaml:"-"`                       // Active temporary combat conditions (Stage 9.8). Don't store this.
	AttacksThisRound         int                            `yaml:"-"`                       // Stage 9.4: Tracks recent attacks for stance calculation. Don't store this.
	DefensesThisRound        int                            `yaml:"-"`                       // Stage 9.4: Tracks recent defenses for stance calculation. Don't store this.
	ConsecutiveHits          int                            `yaml:"-"`                       // Stage 9.4: Consecutive successful hits for momentum. Don't store this.
	ConsecutiveMisses        int                            `yaml:"-"`                       // Stage 9.4: Consecutive misses for momentum. Don't store this.
	ExtraArms                int                            `yaml:"-"`                       // Derived from extra-arms mutation level (0-2). Don't store this.
	IsMob                    bool                           `yaml:"-"`                       // True for mob characters; used for progression caps. Don't store this.
	Skills                   map[string]int                 `yaml:"skills,omitempty"`        // The skills the character has, and what level they are at
	Mutations        map[string]int                 `yaml:"mutations,omitempty"`     // mutationId → level (Stage 12.1)
	MutationProgress float64                        `yaml:"mutationprogress,omitempty"` // accumulates toward next mutation (Stage 12.1)
	Cooldowns        Cooldowns                      `yaml:"cooldowns,omitempty"`     // How many rounds until it is cooled down
	Settings         map[string]string              `yaml:"settings,omitempty"`      // custom setting tracking, used for anything.
	QuestProgress    map[int]string                 `yaml:"questprogress,omitempty"` // quest progress tracking
	QuestFlags       map[string]string              `yaml:"questflags,omitempty"`    // quest flag tracking (e.g., "11-branch" → "rhett")
	LastQuestId      int                            `yaml:"lastquestid,omitempty"`   // most recently progressed quest
	KeyRing          map[string]string              `yaml:"keyring,omitempty"`       // key is the lock id, value is the sequence
	KD               KDStats                        `yaml:"kd,omitempty"`            // Kill/Death stats
	MiscData         map[string]any                 `yaml:"miscdata,omitempty"`      // Any random other data that needs to be stored
	Discoveries      map[int][]string               `yaml:"discoveries,omitempty"`   // Per-room hidden object discoveries
	ExtraLives       int                            `yaml:"extralives,omitempty"`    // How many lives remain. If enabled, players can perma-die if they die at zero
	MobMastery       MobMasteries                   `yaml:"mobmastery,omitempty"`    // Tracks particular masteries around a given mob
	SkillUseCount    map[string]int                 `yaml:"skillusecount,omitempty"` // Tracks how many times each skill has been used
	StatUseCount     map[string]int                 `yaml:"statusecount,omitempty"`  // Tracks how many times each stat has been checked
	Pet              pets.Pet                       `yaml:"pet,omitempty"`           // Do they have a pet?
	Companions       []CompanionInfo                `yaml:"companions,omitempty"`    // Active companions (manifestation system)
	Created          time.Time                      `yaml:"created"`                 // When this character was created
	Timers           map[string]gametime.RoundTimer `yaml:"timers,omitempty"`        // any special timers added to this character
	roomHistory      []int                          // A stack FILO of the last X rooms the character has been in
	PlayerDamage     map[int]int                    `yaml:"-"` // key = who, value = how much
	LastPlayerDamage uint64                         `yaml:"-"` // last round a player damaged this character
	LastSuicideRound uint64                         `yaml:"-"` // runtime only — round of last Suicide execution, for double-fire dedupe
	LastAttackRejectedRound uint64                  `yaml:"-"` // runtime only — round of last player_attack_rejected event fire, for dedupe
	CastingState     *CastingState                  `yaml:"-"` // Active fold-based cast in progress (Stage 11.2). Not persisted.
	CraftingState    *CraftingState                 `yaml:"-"` // Active crafting in progress (Stage 13.1). Not persisted.
	permaBuffIds     []int                          // Buff Id's that are always present for this character
	userId           int                            // User ID of the character if any
	// Stage 3.4: spawn-time override for carry capacity. Set via
	// ApplyMobOverrides for special mobs (wagons). Zero falls through
	// to the default Strength-derived calc.
	carryCapacityOverride float64 `yaml:"-"`
}

func New() *Character {
	c := &Character{
		//Name:   defaultName,
		Adjectives:     []string{},
		RoomId:         StartingRoomId,
		Zone:           startingZone,
		SpeciesId:      startingRace,
		Health:         startingHealth,
		HealthMax:      stats.StatInfo{Base: 1},
		Skills:         initAllSkills(),
		Gold:           250,
		Bank:           100,
		// Starting spells — attack, utility light for dark zones, and
		// basic item inspection so new players can evaluate drops.
		SpellBook: map[string]int{
			"conviction-spike": 1, // Conviction Spike — starting attack spell
			"chrysalis-glow":   1, // Chrysalis Glow — light source for caves
			"identify":         1, // Identify — inspect item properties
		},
		KnownRecipes: crafting.GetStarterRecipes(), // All recipes with skill_minimum == 0
		CharmedMobs:    []int{},
		Items:          []items.Item{},
		Buffs:          buffs.New(),
		Equipment:      Worn{},
		CombatPosition: PositionStanding, // Stage 8.1: Default combat position
		Cooldowns:      make(Cooldowns),  // Initialize cooldowns map
		MiscData:       make(map[string]any),
		Discoveries:    make(map[int][]string),
		SkillUseCount:  make(map[string]int),
		StatUseCount:   make(map[string]int),
		roomHistory:    make([]int, 0, 10),
		KeyRing:        make(map[string]string),
		Created:           time.Now(),
		PlayerDamage:      map[int]int{},
		Timers:            map[string]gametime.RoundTimer{},
		AttacksThisRound:  0,
		DefensesThisRound: 0,
		ConsecutiveHits:   0,
		ConsecutiveMisses: 0,
		CombatPhase:       combatphase.NewMachine(),
	}

	// Roll character stats using normal distribution
	c.Stats = RollCharacterStats()

	// Validate and calculate stats (this calls RecalculateStats internally)
	c.Validate()

	// Set starting health/stamina/conviction to max values
	c.Health = c.HealthMax.Value
	c.Stamina = c.StaminaMax.Value
	c.Conviction = c.ConvictionMax.Value

	return c
}

// initAllSkills creates a skill map with all known skills at rank 1.
func initAllSkills() map[string]int {
	allSkills := skills.GetAllSkillNames()
	m := make(map[string]int, len(allSkills))
	for _, sk := range allSkills {
		m[string(sk)] = 1
	}
	return m
}

// ensureAllSkills ensures all known skills exist in the map at rank 1 minimum.
// Used during Validate() to retroactively update existing characters.
func ensureAllSkills(existing map[string]int) map[string]int {
	if existing == nil {
		return initAllSkills()
	}
	for _, sk := range skills.GetAllSkillNames() {
		if existing[string(sk)] < 1 {
			existing[string(sk)] = 1
		}
	}
	return existing
}

// RollCharacterStats generates a new set of character stats using normal distribution.
// Parameters are driven by the Balance config (StatRollMean, StatRollStdDev, StatRollMin, StatRollMax).
func RollCharacterStats() stats.Statistics {
	b := configs.GetBalanceConfig()
	statMean := float64(b.StatRollMean)
	statStdDev := float64(b.StatRollStdDev)
	statMin := float64(b.StatRollMin)
	statMax := float64(b.StatRollMax)

	// Roll 6 stats
	rolledStats := dice.RollStatArray(6, statMean, statStdDev, statMin, statMax)

	return stats.Statistics{
		Strength:   stats.StatInfo{Base: rolledStats[0]},
		Dexterity:  stats.StatInfo{Base: rolledStats[1]},
		Perception: stats.StatInfo{Base: rolledStats[2]},
		Vitality:   stats.StatInfo{Base: rolledStats[3]},
		Willpower:  stats.StatInfo{Base: rolledStats[4]},
		Charisma:   stats.StatInfo{Base: rolledStats[5]},
	}
}

// returns description unless description is a hash



// DeductStamina attempts to deduct the specified amount of stamina.
// Returns false if the character doesn't have enough stamina.

// GetMovementStaminaCost calculates the stamina cost for movement based on
// terrain difficulty and encumbrance.
// terrainMultiplier: 1.0 = normal terrain, 2.0 = rough terrain, etc.
// Returns stamina cost (2-20 stamina range).

// GetAttackStaminaCost calculates the stamina cost for making an attack.
// Cost is based on weapon type (or unarmed if no weapon).

// DeductAttackStamina deducts stamina for an attack and returns the actual cost deducted.
// If character doesn't have enough stamina, deducts what they have and returns that amount.

// Sometimes it's useful for a character to know what user it belongs to.
func (c *Character) SetUserId(userId int) {
	c.userId = userId
}

func (c *Character) GetUserId() int {
	return c.userId
}

func (c *Character) SetMiscData(key string, value any) {

	if c.MiscData == nil {
		c.MiscData = make(map[string]any)
	}

	if value == nil {
		delete(c.MiscData, key)
		return
	}
	c.MiscData[key] = value
}

func (c *Character) GetMiscData(key string) any {

	if c.MiscData == nil {
		c.MiscData = make(map[string]any)
	}

	if value, ok := c.MiscData[key]; ok {
		return value
	}
	return nil
}

func (c *Character) GetMiscDataKeys(prefixMatch ...string) []string {

	if c.MiscData == nil {
		c.MiscData = make(map[string]any)
	}

	allKeys := []string{}
	for key := range c.MiscData {
		allKeys = append(allKeys, key)
	}

	if len(prefixMatch) == 0 {
		return allKeys
	}

	retKeys := []string{}
	for _, prefix := range prefixMatch {
		for _, key := range allKeys {
			if finalKey, ok := strings.CutPrefix(key, prefix); ok {
				retKeys = append(retKeys, finalKey)
			}
		}
	}

	return retKeys
}

func (c *Character) HasDiscovery(roomId int, key string) bool {
	if c.Discoveries == nil {
		return false
	}
	for _, k := range c.Discoveries[roomId] {
		if k == key {
			return true
		}
	}
	return false
}

func (c *Character) AddDiscovery(roomId int, key string) {
	if c.HasDiscovery(roomId, key) {
		return
	}
	if c.Discoveries == nil {
		c.Discoveries = make(map[int][]string)
	}
	c.Discoveries[roomId] = append(c.Discoveries[roomId], key)
}


// AttemptRecovery tries to recover from a condition using a stat-based chance
// Formula: min(90, 25 + 20 * ln(statValue/25))
func (c *Character) SetSetting(settingName string, settingValue string) {
	if c.Settings == nil {
		c.Settings = make(map[string]string)
	}

	if settingValue == "" {
		delete(c.Settings, settingName)
	} else {
		c.Settings[settingName] = settingValue
	}
}

func (c *Character) GetSetting(settingName string) string {
	if c.Settings == nil {
		c.Settings = make(map[string]string)
	}
	if settingValue, ok := c.Settings[settingName]; ok {
		return settingValue
	}
	return ""
}


// ===================================================================
// Stage 7.1: Segmented Defense Helper Methods
// ===================================================================

const (
	DefenseNone  string = ""
	DefenseDodge string = "dodge"
	DefenseParry string = "parry"
	DefenseBlock string = "block"
)


// GetDefenseStaminaCost returns stamina cost for a defense type (Stage 7.1)

// DeductDefenseStamina deducts stamina for a defense and returns true if successful (Stage 7.1)

// GetToxicityMax returns the maximum toxicity this character can handle.
// Formula: BaseMax + Vitality / VitalityScale

// AddToxicity attempts to add toxicity. Returns false if it would exceed max.

// GetToxicityPenalties returns stat multipliers based on toxicity threshold.
// Returns (regenMult, perceptionMult, dexterityMult) where 1.0 = no penalty.

// Where 1000 = a full round

// StatMod aggregates stat-mod contributions from gear, buffs,
// and pets. Equipment contributions are scaled by the gear-
// effectiveness multiplier from the character's mutations
// (Incorporeal scales gear to zero at max rank). Buff and pet
// contributions are unaffected — they're not gear-derived.
func (c *Character) StatMod(statName string) int {
	gearStat := c.Equipment.StatMod(statName)
	gearStat = int(float64(gearStat) * mutations.GearEffectivenessMultiplier(c.Mutations))
	return gearStat + c.Buffs.StatMod(statName) + c.Pet.StatMod(statName)
}
