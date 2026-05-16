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
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/stats"
)

var (
	startingRace   = 0
	startingHealth = 10
	StartingRoomId = 0
	startingZone   = `Nowhere`
	defaultName    = `nameless`
)

// onCharacterCreatedCallbacks fire after every fresh Character instance is
// constructed via New() or after Validate() initializes a freshly-loaded
// Character. Used by hook packages to wire up per-Character state-machine
// veto callbacks without import cycles.
var onCharacterCreatedCallbacks []func(*Character)

// OnCharacterCreated registers a callback that fires whenever a Character is
// fully constructed (post-init). The callback is called with a pointer to the
// Character so it can register vetoes/cascades/observers on its CombatPhase
// machine.
//
// Used by internal/hooks/* to wire state-machine veto callbacks to Character
// fields without creating import cycles.
func OnCharacterCreated(fn func(*Character)) {
	onCharacterCreatedCallbacks = append(onCharacterCreatedCallbacks, fn)
}

// fireCharacterCreated runs all registered OnCharacterCreated callbacks.
// Called by New() and guarded by combatPhaseWired in Validate() so
// YAML-loaded characters fire callbacks exactly once.
func fireCharacterCreated(c *Character) {
	for _, fn := range onCharacterCreatedCallbacks {
		fn(c)
	}
}

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
	// NonCombatant indicates whether this character is exempt from combat.
	// Default false (i.e., everyone is a combatant by default). Set true for
	// non-combatant NPCs and future player passivity spells. Inverted name
	// gives a sensible zero value — no init needed in New().
	// For mobs, this is populated from Mob.NonCombatant during Mob.Validate().
	// Consumed by Combat Phase's veto chain (chunk 0 Task 10).
	NonCombatant     bool                           `yaml:"non_combatant,omitempty"` // True = exempt from combat
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
	Aggro                    *Aggro                         `yaml:"-"`                       // Runtime combat target. All writes go through SetAggro/EndAggro (dual-write to CombatPhase).
	// CombatPhase is the canonical state machine for "am I in combat?" and
	// "who am I targeting?". It runs alongside the Aggro field; both are
	// kept in sync by SetAggro/EndAggro. Direct .Aggro reads remain valid.
	CombatPhase              *combatphase.Machine           `yaml:"-"`
	// Awareness state machine (chunk 1). Source of truth for
	// "is this character hidden?" Buff #9 still exists as effect
	// carrier; this machine drives its add/remove via cascade.
	Awareness                *awareness.Machine             `yaml:"-"`
	// Life state machine (chunk 2). Source of truth for "is this
	// character alive/dead/respawning?" Cascade handlers in
	// internal/hooks/Life_Cascades.go fire on transitions to clean
	// up other machines and trigger per-actor effects (loot,
	// teleport, decay, etc.).
	Life                     *life.Machine                  `yaml:"-"`
	// Activity state machine (chunk 3). Source of truth for "what
	// activity is this character engaged in?" Replaces
	// CastingState + CraftingState pointer fields (Task 11).
	Activity                 *activity.Machine              `yaml:"-"`
	// Position state machine (chunk 4a). Source of truth for body
	// position + grapple geometry. 14 states (Standing/Prone/Supine/
	// Clinch/BackStanding/Mount/SideControl/KneeOnBelly/NorthSouth/
	// Crucifix/BackGround/HalfGuard/Guard/Turtle). Coexists with the
	// legacy CombatPosition enum through the 4b migration window.
	Position                 *position.Machine              `yaml:"-"`
	// PerGrappleMessageCooldowns tracks which gradient/stamina
	// messages have already fired during the current grapple session.
	// Resets when the character returns to a non-grapple state.
	// Non-persistent — combat doesn't survive logout.
	PerGrappleMessageCooldowns map[string]bool               `yaml:"-"`
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
	MobInstanceId            int                            `yaml:"-"`                       // Non-zero for mob characters; mirrors Mob.InstanceId. Don't store this.
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
	permaBuffIds      []int // Buff Id's that are always present for this character
	userId            int  // User ID of the character if any
	combatPhaseWired  bool `yaml:"-"` // true after OnCharacterCreated callbacks have fired once
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
		CombatPhase:                combatphase.NewMachine(),
		Awareness:                  awareness.NewMachine(),
		Life:                       life.NewMachine(),
		Activity:                   activity.NewMachine(),
		Position:                   position.NewMachine(),
		PerGrappleMessageCooldowns: map[string]bool{},
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

// GetMobInstanceId returns the mob instance ID (non-zero for mobs, zero for
// players). Satisfies position.GrappleActor for TransitionPair callers.
func (c *Character) GetMobInstanceId() int {
	return c.MobInstanceId
}

// GetPosition returns the Position state machine pointer. Satisfies
// position.GrappleActor for TransitionPair callers.
func (c *Character) GetPosition() *position.Machine {
	return c.Position
}

// IsCombatant returns true unless the character is flagged NonCombatant.
// Used by Combat Phase's veto chain (chunk 0 Task 10) and any code that
// needs to ask "can this character be in combat at all?".
func (c *Character) IsCombatant() bool {
	return !c.NonCombatant
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

// ===================================================================
// Combat Phase Convenience Methods (Task 11)
// ===================================================================

// IsEngaged returns true if Combat Phase is Engaged (actively
// in a combat round). Replacement for `c.Aggro != nil && wait==0`
// (the closest historical equivalent).
func (c *Character) IsEngaged() bool {
	if c.CombatPhase == nil {
		return false
	}
	return c.CombatPhase.IsEngaged()
}

// IsInCombat returns true if the character is in any non-Idle combat state.
// CombatPhase is the primary source of truth; Aggro is checked as a fallback
// so that test fixtures which set the field directly continue to work.
func (c *Character) IsInCombat() bool {
	if c.CombatPhase != nil && c.CombatPhase.IsInCombat() {
		return true
	}
	return c.Aggro != nil
}

// IsDisengaging returns true if Combat Phase is Disengaging (flee in
// progress). Replacement for `c.Aggro != nil && c.Aggro.Type == characters.Flee`.
func (c *Character) IsDisengaging() bool {
	if c.CombatPhase == nil {
		return false
	}
	return c.CombatPhase.State() == combatphase.Disengaging
}

// EngagedTarget returns the current Engaged target as an ActorRef.
// Returns zero ActorRef when not Engaged (or Engaging/Disengaging).
// Replacement for `c.Aggro.UserId` / `c.Aggro.MobInstanceId` reads
// during Engaged state.
func (c *Character) EngagedTarget() state.ActorRef {
	if c.CombatPhase == nil {
		return state.ActorRef{}
	}
	if d, ok := c.CombatPhase.EngagedData(); ok {
		return d.Target
	}
	return state.ActorRef{}
}

// CurrentCombatTarget returns the current combat target across all non-Idle
// states (Engaging.Target, Engaged.Target, or Disengaging.LastTarget).
// Returns zero ActorRef when Idle.
//
// CombatPhase is the primary source of truth. The Aggro field is checked as a
// fallback so that test fixtures and any site that writes Aggro directly
// continue to return valid targets.
func (c *Character) CurrentCombatTarget() state.ActorRef {
	if c.CombatPhase != nil {
		if ref := c.CombatPhase.CurrentTarget(); !ref.IsZero() {
			return ref
		}
	}
	// Aggro fallback.
	if c.Aggro != nil {
		return state.ActorRef{
			UserId:        c.Aggro.UserId,
			MobInstanceId: c.Aggro.MobInstanceId,
		}
	}
	return state.ActorRef{}
}

// Attackers returns the framework-maintained inbound attacker
// list — every character currently Engaging or Engaged with
// this Character as their target.
//
// Replaces room-scan loops for "who's attacking me?". The list
// is updated atomically by the Combat Phase framework on every
// transition.
func (c *Character) Attackers() []state.ActorRef {
	if c.CombatPhase == nil {
		return nil
	}
	return c.CombatPhase.Attackers()
}

// ===================================================================
// Awareness Convenience Methods (Task 10)
// ===================================================================

// IsHidden returns true when the character's Awareness state is
// Hidden. Replacement for the legacy HasBuffFlag(buffs.Hidden)
// pattern. Buff #9 still exists as an effect carrier; the cascade
// in internal/hooks/Awareness_Cascades.go keeps the buff and the
// Awareness state synchronized.
func (c *Character) IsHidden() bool {
	if c.Awareness == nil {
		return false
	}
	return c.Awareness.IsHidden()
}

// ===================================================================
// Life Convenience Methods (Task 4)
// ===================================================================

// IsAlive returns true when Life state is Alive.
// Replacement for ad-hoc Health > 0 checks once callers migrate.
func (c *Character) IsAlive() bool {
	if c.Life == nil {
		return true // defensive: pre-init characters treated as alive
	}
	return c.Life.IsAlive()
}

// IsDead returns true when Life state is Dead.
func (c *Character) IsDead() bool {
	if c.Life == nil {
		return false
	}
	return c.Life.IsDead()
}

// IsRespawning returns true when Life state is Respawning.
func (c *Character) IsRespawning() bool {
	if c.Life == nil {
		return false
	}
	return c.Life.IsRespawning()
}
