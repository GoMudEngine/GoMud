package characters

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/enchantments"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/pets"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/stats"
	"github.com/GoMudEngine/GoMud/internal/util"

	//
	"maps"
	"slices"
)

var (
	startingRace   = 0
	startingHealth = 10
	StartingRoomId = 0
	startingZone     = `Nowhere`
	defaultName      = `nameless`
	descriptionCache = map[string]string{} // key is a hash, value is the description
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
	ActionPoints     int                            // The resevoir of action points the character has to spend on movement etc.
	Gold             int                            // The gold the character is holding
	Bank             int                            // The gold the character has in the bank
	Shop             Shop                           `yaml:"shop,omitempty"`          // Definition of shop services/items this character stocks (or just has at the moment)
	SpellBook        map[string]int                 `yaml:"spellbook,omitempty"`     // The spells the character has learned
	KnownRecipes     map[string]int                 `yaml:"knownrecipes,omitempty"`  // The crafting recipes the character has discovered
	Charmed          *CharmInfo                     `yaml:"-"`                       // If they are charmed, this is the info
	CharmedMobs      []int                          `yaml:"-"`                       // If they have charmed anyone, this is the list of mob instance ids
	Items            []items.Item                   `yaml:"items,omitempty"`         // The items the character is holding
	Buffs            buffs.Buffs                    `yaml:"buffs,omitempty"`         // The buffs the character has active
	Equipment        Worn                           `yaml:"equipment,omitempty"`     // The equipment the character is wearing
	HealthMax        stats.StatInfo                 `yaml:"-"`                       // The maximum health of the character. Don't write to yaml since is dynamically calculated.
	StaminaMax       stats.StatInfo                 `yaml:"-"`                       // The maximum stamina of the character. Don't write to yaml since is dynamically calculated.
	ConvictionMax    stats.StatInfo                 `yaml:"-"`                       // The maximum conviction of the character. Don't write to yaml since is dynamically calculated.
	ActionPointsMax          stats.StatInfo                 `yaml:"-"`                       // The maximum actions of character. Don't write to yaml since is dynamically calculated.
	Aggro                    *Aggro                         `yaml:"-"`                       // Dont' store this. If they leave they break their aggro
	CombatPosition           CombatPosition                 `yaml:"-"`                       // Current combat position (Standing/Prone/Clinched/Grounded). Don't store this.
	PositionRoundsMin        int                            `yaml:"-"`                       // Minimum rounds in current position (for Prone bash/trip, etc). Don't store this.
	DownedRounds             int                            `yaml:"-"`                       // Rounds since downed, for coup de grâce timer. Don't store this.
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
	KeyRing          map[string]string              `yaml:"keyring,omitempty"`       // key is the lock id, value is the sequence
	KD               KDStats                        `yaml:"kd,omitempty"`            // Kill/Death stats
	MiscData         map[string]any                 `yaml:"miscdata,omitempty"`      // Any random other data that needs to be stored
	Discoveries      map[int][]string               `yaml:"discoveries,omitempty"`   // Per-room hidden object discoveries
	ExtraLives       int                            `yaml:"extralives,omitempty"`    // How many lives remain. If enabled, players can perma-die if they die at zero
	MobMastery       MobMasteries                   `yaml:"mobmastery,omitempty"`    // Tracks particular masteries around a given mob
	SkillUseCount    map[string]int                 `yaml:"skillusecount,omitempty"` // Tracks how many times each skill has been used
	StatUseCount     map[string]int                 `yaml:"statusecount,omitempty"`  // Tracks how many times each stat has been checked
	Pet              pets.Pet                       `yaml:"pet,omitempty"`           // Do they have a pet?
	Created          time.Time                      `yaml:"created"`                 // When this character was created
	Timers           map[string]gametime.RoundTimer `yaml:"timers,omitempty"`        // any special timers added to this character
	roomHistory      []int                          // A stack FILO of the last X rooms the character has been in
	PlayerDamage     map[int]int                    `yaml:"-"` // key = who, value = how much
	LastPlayerDamage uint64                         `yaml:"-"` // last round a player damaged this character
	CastingState     *CastingState                  `yaml:"-"` // Active fold-based cast in progress (Stage 11.2). Not persisted.
	CraftingState    *CraftingState                 `yaml:"-"` // Active crafting in progress (Stage 13.1). Not persisted.
	permaBuffIds     []int                          // Buff Id's that are always present for this character
	userId           int                            // User ID of the character if any
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
		Gold:           25,
		Bank:           100,
		// Phase 25.1: Starting spells — attack + utility light for dark zones.
		SpellBook: map[string]int{
			"conviction-spike": 1, // Conviction Spike — starting attack spell
			"chrysalis-glow":   1, // Chrysalis Glow — light source for caves
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

// GetTotalSkillRanks returns the sum of all skill ranks.
func (c *Character) GetTotalSkillRanks() int {
	total := 0
	for _, rank := range c.Skills {
		total += rank
	}
	return total
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
// which points to another description location.
func (c *Character) GetDescription() string {

	if !strings.HasPrefix(c.Description, `h:`) {
		return c.Description
	}
	hash := strings.TrimPrefix(c.Description, `h:`)
	return descriptionCache[hash]
}

// GetMutationVisuals returns a space-joined string of all owned mutation visual
// descriptors, sorted by mutation id for deterministic output. Returns "" if
// no mutations have a visual field. Used by the description template (Stage 12.2).
func (c *Character) GetMutationVisuals() string {
	if len(c.Mutations) == 0 {
		return ""
	}
	ids := make([]string, 0, len(c.Mutations))
	for id := range c.Mutations {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		if spec := mutations.GetMutation(id); spec != nil && spec.Visual != "" {
			parts = append(parts, spec.Visual)
		}
	}
	return strings.Join(parts, " ")
}

// returns description unless description is a hash
// which points to another description location.
func (c *Character) TrackPlayerDamage(userId int, damageAmt int) {

	roundNow := util.GetRoundCount()
	if len(c.PlayerDamage) == 0 {
		c.PlayerDamage = map[int]int{}
	} else {
		if roundNow-c.LastPlayerDamage > 30 {
			clear(c.PlayerDamage)
		}
	}

	c.PlayerDamage[userId] = c.PlayerDamage[userId] + damageAmt
	c.LastPlayerDamage = roundNow

}

/*
All spells should have a 10% minimum chance of success.
*/
func (c *Character) GetBaseCastSuccessChance(spellId string) int {

	sp := spells.GetSpell(spellId)
	if sp == nil {
		return -1
	}

	// start with 100% chance of success
	targetNumber := 100

	// subtract spell difficulty
	// 1-100
	targetNumber -= sp.GetDifficulty()

	// add spell level bonus
	// 10-30
	skillLevel := c.GetSkillLevel(skills.Spellcasting)
	if skillLevel == 0 {
		skillLevel = c.GetSkillLevel(skills.Cast) // backward compat with legacy Cast skill
	}
	//targetNumber += (skillLevel * 5)
	//targetNumber -= 5 // cancel out the first level

	// add the proficiency of the spell (more casts == better)
	// 0-20
	profFactor := 1.0
	if skillLevel == 2 {
		profFactor = 1.25 // .25 more than lvl 1
	} else if skillLevel == 3 {
		profFactor = 1.75 // .50 more than lvl 2
	} else if skillLevel == 4 {
		profFactor = 2.50 // .75 more than lvl 3
	}
	casts := c.SpellBook[spellId]
	castsPerPoint := float64(configs.GetBalanceConfig().SpellProficiencyCastsPerPoint)
	proficiency := int(math.Floor((float64(casts) / castsPerPoint * profFactor)))
	if proficiency < 0 {
		proficiency = 0
	} else if proficiency > 20 {
		proficiency = 20
	}
	targetNumber += proficiency

	targetNumber += int(math.Floor(float64(c.Stats.Willpower.ValueAdj) / 5))

	// add by any stat mods for casting, or casting school
	// 0-xx
	targetNumber += c.StatMod(string(statmods.Casting))
	// Add stat mods for each school the spell belongs to
	for _, school := range sp.Schools {
		targetNumber += c.StatMod(string(statmods.CastingPrefix) + school)
	}

	if targetNumber < 0 {
		targetNumber = 0
	} else if targetNumber > 100 {
		targetNumber = 100
	}

	return targetNumber
}

// CarryCapacity returns weight capacity in pounds (Strength × 3)
func (c *Character) CarryCapacity() float64 {
	return float64(c.Stats.Strength.ValueAdj) * 3.0
}

// GetCarriedWeight returns the total weight of all carried items in pounds
func (c *Character) GetCarriedWeight() float64 {
	totalWeight := 0.0

	// Add weight from inventory items
	for _, item := range c.Items {
		totalWeight += item.GetSpec().GetWeight()
	}

	// Add weight from equipped items
	if c.Equipment.Weapon.ItemId > 0 {
		totalWeight += c.Equipment.Weapon.GetSpec().GetWeight()
	}
	if c.Equipment.Offhand.ItemId > 0 {
		totalWeight += c.Equipment.Offhand.GetSpec().GetWeight()
	}
	if c.Equipment.Head.ItemId > 0 {
		totalWeight += c.Equipment.Head.GetSpec().GetWeight()
	}
	if c.Equipment.Neck.ItemId > 0 {
		totalWeight += c.Equipment.Neck.GetSpec().GetWeight()
	}
	if c.Equipment.Body.ItemId > 0 {
		totalWeight += c.Equipment.Body.GetSpec().GetWeight()
	}
	if c.Equipment.Belt.ItemId > 0 {
		totalWeight += c.Equipment.Belt.GetSpec().GetWeight()
	}
	if c.Equipment.Gloves.ItemId > 0 {
		totalWeight += c.Equipment.Gloves.GetSpec().GetWeight()
	}
	if c.Equipment.Ring.ItemId > 0 {
		totalWeight += c.Equipment.Ring.GetSpec().GetWeight()
	}
	if c.Equipment.Legs.ItemId > 0 {
		totalWeight += c.Equipment.Legs.GetSpec().GetWeight()
	}
	if c.Equipment.Feet.ItemId > 0 {
		totalWeight += c.Equipment.Feet.GetSpec().GetWeight()
	}

	return totalWeight
}

func (c *Character) DeductActionPoints(amount int) bool {

	if c.ActionPoints < amount {
		return false
	}
	c.ActionPoints -= amount
	if c.ActionPoints < 0 {
		c.ActionPoints = 0
	}
	return true
}

// DeductStamina attempts to deduct the specified amount of stamina.
// Returns false if the character doesn't have enough stamina.
func (c *Character) DeductStamina(amount int) bool {
	if c.Stamina < amount {
		return false
	}
	c.Stamina -= amount
	if c.Stamina < 0 {
		c.Stamina = 0
	}
	return true
}

// GetMovementStaminaCost calculates the stamina cost for movement based on
// terrain difficulty and encumbrance.
// terrainMultiplier: 1.0 = normal terrain, 2.0 = rough terrain, etc.
// Returns stamina cost (2-20 stamina range).
func (c *Character) GetMovementStaminaCost(terrainMultiplier float64) int {
	b := configs.GetBalanceConfig()
	baseCost := float64(b.MovementBaseStaminaCost)
	maxCost := float64(b.MovementMaxStaminaCost)

	// Apply terrain multiplier
	cost := baseCost * terrainMultiplier

	// Calculate encumbrance multiplier based on weight
	encumbranceMultiplier := 1.0
	carriedWeight := c.GetCarriedWeight()
	capacity := c.CarryCapacity()

	if carriedWeight > capacity {
		// Overencumbered: scale from 1.0 to 5.0 based on how much over capacity
		overAmount := carriedWeight - capacity
		overRatio := overAmount / capacity
		// Cap at 5x multiplier when carrying 2x capacity
		encumbranceMultiplier = 1.0 + math.Min(overRatio*4.0, 4.0)
	}

	// Apply encumbrance multiplier
	cost *= encumbranceMultiplier

	// Phase 24.2: Apply mutation movement speed modifier (Hasted, Extra Legs, etc.)
	if moveMod := mutations.GetMovementSpeedModifier(c.Mutations); moveMod != 0 {
		cost *= (1.0 - moveMod) // positive moveMod = faster = less cost
	}

	// Cap at maximum stamina cost
	if cost > maxCost {
		cost = maxCost
	}

	// Minimum 1 stamina
	if cost < 1.0 {
		cost = 1.0
	}

	return int(math.Ceil(cost))
}

// GetAttackStaminaCost calculates the stamina cost for making an attack.
// Cost is based on weapon type (or unarmed if no weapon).
func (c *Character) GetAttackStaminaCost() int {
	// Check main hand weapon
	if c.Equipment.Weapon.ItemId > 0 {
		weaponSpec := c.Equipment.Weapon.GetSpec()
		return weaponSpec.GetAttackStaminaCost()
	}

	// Check offhand weapon (dual wielding)
	if c.Equipment.Offhand.ItemId > 0 {
		offhandSpec := c.Equipment.Offhand.GetSpec()
		return offhandSpec.GetAttackStaminaCost()
	}

	// Unarmed combat costs less stamina
	return int(configs.GetBalanceConfig().UnarmedAttackStaminaCost)
}

// DeductAttackStamina deducts stamina for an attack and returns the actual cost deducted.
// If character doesn't have enough stamina, deducts what they have and returns that amount.
func (c *Character) DeductAttackStamina() int {
	cost := c.GetAttackStaminaCost()

	if c.Stamina >= cost {
		c.Stamina -= cost
		return cost
	}

	// Insufficient stamina - deduct what we have
	actualCost := c.Stamina
	c.Stamina = 0
	return actualCost
}

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

func (c *Character) FindKeyInBackpack(lockId string) (items.Item, bool) {

	lockId = strings.ToLower(lockId)

	for _, itm := range c.GetAllBackpackItems() {
		itmSpec := itm.GetSpec()
		if itmSpec.Type != items.Key {
			continue
		}

		if itmSpec.KeyLockId == lockId {
			return itm, true
		}
	}

	return items.Item{}, false
}

func (c *Character) HasKey(lockId string, difficulty int) (hasKey bool, hasSequence bool) {

	sequence := util.GetLockSequence(lockId, difficulty, string(configs.GetServerConfig().Seed))

	// Check whether they ahve a key for this lock
	return c.GetKey(`key-`+lockId) != ``, c.GetKey(lockId) == sequence
}

func (c *Character) KeyCount() int {
	if c.KeyRing == nil {
		c.KeyRing = make(map[string]string)
	}
	return len(c.KeyRing)
}

func (c *Character) GetKey(lockId string) string {
	if c.KeyRing == nil {
		c.KeyRing = make(map[string]string)
	}
	return c.KeyRing[strings.ToLower(lockId)]
}

func (c *Character) SetKey(lockId string, sequence string) {
	if c.KeyRing == nil {
		c.KeyRing = make(map[string]string)
	}
	if len(sequence) == 0 {
		delete(c.KeyRing, strings.ToLower(lockId))
	} else {
		c.KeyRing[strings.ToLower(lockId)] = strings.ToUpper(sequence)
	}
}

// This should only be used for mobs.
// Not players
func (c *Character) CacheDescription() {
	// Hash the descriptions and store centrally.
	// This saves a lot of memory because many descriptions are duplicates
	hash := util.Hash(c.Description)
	if _, ok := descriptionCache[hash]; !ok {
		descriptionCache[hash] = c.Description
	}
	c.Description = fmt.Sprintf(`h:%s`, hash)
}

func (c *Character) GetDefaultDiceRoll() (attacks int, dCount int, dSides int, bonus int, buffOnCrit []int) {
	// default racial
	speciesInfo := species.GetSpecies(c.SpeciesId)

	attacks = speciesInfo.Damage.Attacks
	dCount = speciesInfo.Damage.DiceCount
	dSides = speciesInfo.Damage.SideCount
	bonus = speciesInfo.Damage.BonusDamage
	buffOnCrit = speciesInfo.Damage.CritBuffIds

	dCount += int(math.Floor((float64(c.Stats.Dexterity.ValueAdj) / 50)))
	dSides += int(math.Floor((float64(c.Stats.Strength.ValueAdj) / 12)))
	bonus += int(math.Floor((float64(c.Stats.Charisma.ValueAdj) / 25)))

	if dCount < speciesInfo.Damage.DiceCount {
		dCount = speciesInfo.Damage.DiceCount
	}
	if dSides < speciesInfo.Damage.SideCount {
		dSides = speciesInfo.Damage.SideCount
	}

	return attacks, dCount, dSides, bonus, buffOnCrit
}

// GetDefaultDistributionDamage returns distribution damage parameters for unarmed combat.
// Uses CalculateUnarmedDamage to scale with Strength and Unarmed Combat skill.
// This provides meaningful progression for unarmed fighters.
func (c *Character) GetDefaultDistributionDamage() (attacks int, baseDamage float64, variance float64, buffOnCrit []int) {
	speciesInfo := species.GetSpecies(c.SpeciesId)

	attacks = speciesInfo.Damage.Attacks
	if attacks < 1 {
		attacks = 1
	}
	buffOnCrit = speciesInfo.Damage.CritBuffIds

	// Use skill-based unarmed damage calculation (Stage 7.3)
	baseDamage, variance = c.CalculateUnarmedDamage()

	return attacks, baseDamage, variance, buffOnCrit
}

// CalculateUnarmedDamage returns the base damage and variance for unarmed attacks.
// This function is designed to be extensible for future conditions/mutations.
//
// Formula: baseDamage = baseValue + (Strength / strengthDivisor) + (UnarmedSkill / skillDivisor)
//
// Scaling examples (at 100 Strength):
//   - Skill 0 (untrained):  2 + 4 + 0  = 6 damage
//   - Skill 50 (trained):   2 + 4 + 5  = 11 damage
//   - Skill 100 (master):   2 + 4 + 10 = 16 damage
//
// Extension points for future conditions/mutations:
//   - baseDamage can be modified by multipliers (e.g., "Stone Fists" mutation: baseDamage *= 1.5)
//   - variance can be modified (e.g., "Precise Strikes" condition: variance *= 0.5)
//   - Additional additive bonuses (e.g., "Enhanced Strength" buff: +5 damage)
//
// To add a buff/condition/mutation that affects unarmed damage:
//   1. Check for the buff/condition after base calculation
//   2. Apply multipliers: baseDamage *= multiplier
//   3. Apply additive bonuses: baseDamage += bonus
//   4. Modify variance if needed: variance *= varianceMultiplier
func (c *Character) CalculateUnarmedDamage() (baseDamage float64, variance float64) {
	b := configs.GetBalanceConfig()
	baseValue := float64(b.UnarmedBaseDamage)
	strengthDivisor := float64(b.UnarmedStrengthDivisor)
	skillDivisor := float64(b.UnarmedSkillDivisor)
	baseVariance := float64(b.UnarmedBaseVariance)

	// Calculate base damage from stats and skill
	strengthBonus := float64(c.Stats.Strength.ValueAdj) / strengthDivisor
	skillBonus := float64(c.GetSkillLevel(skills.UnarmedCombat)) / skillDivisor

	baseDamage = baseValue + strengthBonus + skillBonus

	// Base variance scales slightly with skill (more skill = more consistent)
	// High skill fighters are more consistent, low skill fighters are erratic
	skillLevel := c.GetSkillLevel(skills.UnarmedCombat)
	varianceReduction := float64(skillLevel) / 50.0 // 0 at skill 0, 2 at skill 100
	variance = math.Max(1.0, baseVariance-varianceReduction)

	// --- FUTURE EXTENSION POINT: Buffs/Conditions/Mutations ---
	// Example implementations (commented out for now):
	//
	// if c.HasBuffFlag(buffs.StoneFists) {
	//     baseDamage *= 1.5  // Stone Fists: +50% damage
	//     variance *= 1.2    // But less precise (heavier strikes)
	// }
	//
	// if c.HasBuffFlag(buffs.PreciseStrikes) {
	//     variance *= 0.5    // Precise Strikes: Half variance (more consistent)
	// }
	//
	// if c.HasBuffFlag(buffs.EnhancedStrength) {
	//     baseDamage += 5.0  // Flat +5 damage bonus
	// }
	//
	// if c.HasCondition("weakened") {
	//     baseDamage *= 0.7  // Weakened: -30% damage
	// }
	//
	// if c.HasMutation("razor-claws") {
	//     baseDamage += 3.0  // Razor Claws: +3 base damage
	//     variance += 1.0    // Claws add slight variance
	// }
	//
	// if c.HasMutation("padded-fists") {
	//     baseDamage *= 0.8  // Padded Fists: -20% damage
	//     variance *= 0.6    // But more consistent
	// }
	// --- END EXTENSION POINT ---

	// Stage 12.1: Natural weapon bonus from mutations (Clawed Hands etc.)
	baseDamage += mutations.GetNaturalWeaponBonus(c.Mutations)

	// Ensure minimums
	baseDamage = math.Max(1.0, baseDamage)
	variance = math.Max(0.5, variance)

	return baseDamage, variance
}

func (c *Character) GetSpells() map[string]int {
	ret := make(map[string]int)
	maps.Copy(ret, c.SpellBook)
	return ret
}

// IsCasting returns true if the character has an active fold-based cast in progress.
func (c *Character) IsCasting() bool { return c.CastingState != nil }

// IsCrafting returns true if the character has an active crafting operation in progress.
func (c *Character) IsCrafting() bool { return c.CraftingState != nil }

func (c *Character) HasSpell(spellName string) bool {
	if intVal, ok := c.SpellBook[spellName]; ok {
		return intVal > 0
	}
	return false
}

func (c *Character) DisableSpell(spellName string) bool {
	if intVal, ok := c.SpellBook[spellName]; ok {
		if intVal > 0 {
			c.SpellBook[spellName] = intVal * -1
		}
	}
	return false
}

func (c *Character) EnableSpell(spellName string) bool {
	if intVal, ok := c.SpellBook[spellName]; ok {
		if intVal < 0 {
			c.SpellBook[spellName] = intVal * -1
		}
	}
	return false
}

func (c *Character) TrackSpellCast(spellName string) bool {
	if intVal, ok := c.SpellBook[spellName]; ok {
		if intVal > 0 {
			intVal++
			c.SpellBook[spellName] = intVal
		}
	}
	return false
}

func (c *Character) LearnSpell(spellName string) bool {
	if _, ok := c.SpellBook[spellName]; !ok {
		c.SpellBook[spellName] = 1
		return true
	}
	return false
}

func (c *Character) HasRecipe(recipeId string) bool {
	if c.KnownRecipes == nil {
		return false
	}
	if intVal, ok := c.KnownRecipes[recipeId]; ok {
		return intVal > 0
	}
	return false
}

func (c *Character) LearnRecipe(recipeId string) bool {
	if c.KnownRecipes == nil {
		c.KnownRecipes = crafting.GetStarterRecipes()
	}
	if _, ok := c.KnownRecipes[recipeId]; !ok {
		c.KnownRecipes[recipeId] = 1
		return true
	}
	return false
}

func (c *Character) TrackCharmed(mobId int, add bool) {
	for pos, mobInstanceId := range c.CharmedMobs {
		if mobInstanceId == mobId {
			if !add {
				c.CharmedMobs = slices.Delete(c.CharmedMobs, pos, pos+1)
			}
			return
		}
	}
	c.CharmedMobs = append(c.CharmedMobs, mobId)
}

func (c *Character) GetCharmIds() []int {
	return append([]int{}, c.CharmedMobs...)
}

func (c *Character) Charm(userId int, rounds int, expireCommand string) {
	c.SetAdjective(`charmed`, true)
	c.Charmed = NewCharm(userId, rounds, expireCommand)
	if c.Aggro != nil && c.Aggro.UserId == userId {
		c.Aggro = nil
	}
}

func (c *Character) KnowsFirstAid() bool {
	if r := species.GetSpecies(c.SpeciesId); r != nil {
		return r.KnowsFirstAid
	}
	return false
}

func (c *Character) GetCharmedUserId() int {
	if c.Charmed != nil {
		return c.Charmed.UserId
	}
	return 0
}

func (c *Character) IsCharmed(userId ...int) bool {

	if c.Charmed == nil {
		return false
	}

	if len(userId) == 0 {
		return c.Charmed != nil
	}

	if c.Charmed == nil {
		return false
	}
	return slices.Contains(userId, c.Charmed.UserId)
}

// Returns userId of whoever had charmed them
func (c *Character) RemoveCharm() int {
	charmUserId := 0
	c.SetAdjective(`charmed`, false)
	if c.Charmed != nil {
		charmUserId = c.Charmed.UserId
		c.Charmed = nil
	}
	return charmUserId
}

func (c *Character) GetRandomItem() (items.Item, bool) {
	if len(c.Items) == 0 {
		return items.Item{}, false
	}
	return c.Items[util.Rand(len(c.Items))], true
}

// USERNAME appears to be <BLANK>
func (c *Character) GetHealthAppearance() string {

	className := util.HealthClass(c.Health, c.HealthMax.Value)
	pct := int(float64(c.Health) / float64(c.HealthMax.Value) * 100)

	if pct < 15 {
		return fmt.Sprintf(`<ansi fg="username">%s</ansi> looks like they're <ansi fg="%s">about to die!</ansi>`, c.Name, className)
	}

	if pct < 50 {
		return fmt.Sprintf(`<ansi fg="username">%s</ansi> looks to be in <ansi fg="%s">pretty bad shape.</ansi>`, c.Name, className)
	}

	if pct < 80 {
		return fmt.Sprintf(`<ansi fg="username">%s</ansi> has some <ansi fg="%s">cuts and bruises.</ansi>`, c.Name, className)
	}

	if pct < 100 {
		return fmt.Sprintf(`<ansi fg="username">%s</ansi> has <ansi fg="%s">a few scratches.</ansi>`, c.Name, className)
	}

	return fmt.Sprintf(`<ansi fg="username">%s</ansi> is in <ansi fg="%s">perfect health.</ansi>`, c.Name, className)
}

func (c *Character) GetAllSkillRanks() map[string]int {
	retMap := make(map[string]int)
	maps.Copy(retMap, c.Skills)
	return retMap
}

// Returns an integer representing a % damage reduction
func (c *Character) GetDefense() int {

	reduction := c.Equipment.Weapon.GetDefense() +
		c.Equipment.Offhand.GetDefense() +
		c.Equipment.ExtraArm1.GetDefense() +
		c.Equipment.ExtraArm2.GetDefense() +
		c.Equipment.Head.GetDefense() +
		c.Equipment.Neck.GetDefense() +
		c.Equipment.Body.GetDefense() +
		c.Equipment.Belt.GetDefense() +
		c.Equipment.Gloves.GetDefense() +
		c.Equipment.Ring.GetDefense() +
		c.Equipment.Legs.GetDefense() +
		c.Equipment.Feet.GetDefense()

	//reduction = int(float64(reduction) / 9)

	// If wearing an offhand item like a shield, defense gets a 50% boost
	// Holdables are not considered "shield" type items.
	// Anything held in the offhand that provides a damage reduction is considered a shield.
	if c.Equipment.Offhand.ItemId != 0 && c.Equipment.Offhand.GetSpec().Type != items.Weapon && c.Equipment.Offhand.GetSpec().DamageReduction > 0 {
		reduction = int(float64(reduction) * 1.5)
	}

	// Add magical armor from Minor Shield (or any future ConditionShield source)
	reduction += int(c.GetConditionMagnitude(ConditionShield))

	// Stage 12.1: Add natural armor from mutations (Tough Skin etc.)
	reduction += mutations.GetNaturalArmor(c.Mutations)

	// Species natural armor (chitin, thick hide, etc.)
	if speciesInfo := species.GetSpecies(c.SpeciesId); speciesInfo != nil {
		reduction += speciesInfo.NaturalArmor
	}

	if reduction > 100 {
		reduction = 100
	}

	return reduction
}

// GetPhysicalMitigation returns total physical mitigation as a fraction (0.0–1.0).
// Sources: equipment physical_mitigation (falls back to DamageReduction for
// unmigrated items), mutations, species natural armor, shield spells.
func (c *Character) GetPhysicalMitigation() float64 {
	total := 0

	slots := []items.Item{
		c.Equipment.Weapon, c.Equipment.Offhand,
		c.Equipment.ExtraArm1, c.Equipment.ExtraArm2,
		c.Equipment.Head, c.Equipment.Neck, c.Equipment.Body,
		c.Equipment.Belt, c.Equipment.Gloves, c.Equipment.Ring,
		c.Equipment.Legs, c.Equipment.Feet,
	}
	for _, slot := range slots {
		if slot.ItemId <= 0 {
			continue
		}
		spec := slot.GetSpec()
		total += spec.PhysicalMitigation
	}

	// Shield condition (Minor Shield spell)
	total += int(c.GetConditionMagnitude(ConditionShield))

	// Mutation natural armor
	total += mutations.GetNaturalArmor(c.Mutations)

	// Species natural armor
	if speciesInfo := species.GetSpecies(c.SpeciesId); speciesInfo != nil {
		total += speciesInfo.NaturalArmor
	}

	return float64(total) / 100.0
}

// GetMagicalMitigation returns total magical mitigation as a fraction (0.0–1.0).
// Sources: equipment magical_mitigation, mutation magical resistance.
func (c *Character) GetMagicalMitigation() float64 {
	total := 0

	slots := []items.Item{
		c.Equipment.Weapon, c.Equipment.Offhand,
		c.Equipment.ExtraArm1, c.Equipment.ExtraArm2,
		c.Equipment.Head, c.Equipment.Neck, c.Equipment.Body,
		c.Equipment.Belt, c.Equipment.Gloves, c.Equipment.Ring,
		c.Equipment.Legs, c.Equipment.Feet,
	}
	for _, slot := range slots {
		if slot.ItemId <= 0 {
			continue
		}
		spec := slot.GetSpec()
		total += spec.MagicalMitigation
	}

	// Mutation magical resistance (returned as fraction 0.0–1.0, convert to percentage points)
	total += int(mutations.GetMagicalResistance(c.Mutations) * 100)

	return float64(total) / 100.0
}

// GetConvictionMitigation returns total conviction mitigation as a fraction (0.0–1.0).
// Sources: equipment conviction_mitigation, mutation conviction resistance.
func (c *Character) GetConvictionMitigation() float64 {
	total := 0

	slots := []items.Item{
		c.Equipment.Weapon, c.Equipment.Offhand,
		c.Equipment.ExtraArm1, c.Equipment.ExtraArm2,
		c.Equipment.Head, c.Equipment.Neck, c.Equipment.Body,
		c.Equipment.Belt, c.Equipment.Gloves, c.Equipment.Ring,
		c.Equipment.Legs, c.Equipment.Feet,
	}
	for _, slot := range slots {
		if slot.ItemId <= 0 {
			continue
		}
		spec := slot.GetSpec()
		total += spec.ConvictionMitigation
	}

	// Mutation conviction resistance (returned as fraction 0.0–1.0, convert to percentage points)
	total += int(mutations.GetConvictionResistance(c.Mutations) * 100)

	return float64(total) / 100.0
}

func (c *Character) GetMobName(viewingUserId int, renderFlags ...NameRenderFlag) FormattedName {
	return c.getFormattedName(viewingUserId, `mobname`, renderFlags...)
}

// GetMobNameIndexed returns a formatted mob name with a duplicate index marker.
// When dupIndex > 0, the name displays as "name #N" with shifted colors for
// indices 2+. Use this when multiple mobs share the same name in a room.
func (c *Character) GetMobNameIndexed(viewingUserId int, dupIndex int, renderFlags ...NameRenderFlag) FormattedName {
	f := c.getFormattedName(viewingUserId, `mobname`, renderFlags...)
	f.DuplicateIndex = dupIndex
	return f
}

func (c *Character) GetPlayerName(viewingUserId int, renderFlags ...NameRenderFlag) FormattedName {
	return c.getFormattedName(viewingUserId, `username`, renderFlags...)
}

func (c *Character) HasAdjective(adj string) bool {
	return slices.Contains(c.Adjectives, adj)
}

func (c *Character) SetAdjective(adj string, addToList bool) {
	if c.Adjectives == nil {
		c.Adjectives = []string{}
	}
	for i, a := range c.Adjectives {
		if a == adj {
			if addToList {
				return
			} else {
				c.Adjectives = slices.Delete(c.Adjectives, i, i+1)
				return
			}
		}
	}
	if addToList {
		c.Adjectives = append(c.Adjectives, adj)
	}
}

func (c *Character) GetAdjectives() []string {

	retAdjectives := []string{}

	// Start dynamic adjectives
	if c.Health < 1 {
		retAdjectives = append(retAdjectives, `downed`)
	}

	if len(c.Shop) > 0 {
		retAdjectives = append(retAdjectives, `shop`)
	}

	if c.HasFlagFromAnySource(buffs.EmitsLight) {
		retAdjectives = append(retAdjectives, `lit`)
	}

	if c.HasFlagFromAnySource(buffs.Hidden) {
		retAdjectives = append(retAdjectives, `hidden`)
	}

	if c.HasBuffFlag(buffs.Poison) {
		retAdjectives = append(retAdjectives, `poisoned`)
	}
	// End dynamic adjectives

	retAdjectives = append(retAdjectives, c.Adjectives...)

	return retAdjectives
}

// AttemptRecovery tries to recover from a condition using a stat-based chance
// Formula: min(90, 25 + 20 * ln(statValue/25))
// Returns: (attemptMade, success)
// - attemptMade: whether the character had a condition to recover from
// - success: whether the recovery succeeded (only meaningful if attemptMade is true)
func (c *Character) AttemptRecovery(statValue int) (bool, bool) {
	// Currently only handles Prone, but future-proofed for grapple/entangle/etc
	if c.CombatPosition != PositionProne {
		return false, false // No condition to recover from
	}

	// Decrement minimum prone duration counter
	if c.PositionRoundsMin > 0 {
		c.PositionRoundsMin--
		// Still in minimum prone period, can't attempt recovery yet
		// Reduce attacks to 1 this round (struggling to stand)
		c.AddCondition(ConditionRecoveryPenalty, 1, 1.0, "prone recovery")
		return false, false // No recovery attempt yet (still in minimum duration)
	}

	// Minimum duration passed, now roll for recovery based on stat
	// Calculate recovery chance using logarithmic formula
	// DEX 25 = 25%, DEX 100 = 50%, DEX 300 = 75%, caps at 90%
	chance := 25.0
	if statValue > 0 {
		chance = 25.0 + 20.0*math.Log(float64(statValue)/25.0)
		if chance > 90.0 {
			chance = 90.0
		}
		if chance < 0 {
			chance = 0
		}
	}

	// Roll for success
	roll := dice.RollStat(50) // Mean of 50
	success := roll.Value < chance

	if success {
		c.CombatPosition = PositionStanding
		c.PositionRoundsMin = 0
	} else {
		// Failed recovery attempt - reduce attacks to 1 this round
		c.AddCondition(ConditionRecoveryPenalty, 1, 1.0, "prone recovery")
	}

	return true, success
}

func (c *Character) getFormattedName(viewingUserId int, uType string, renderFlags ...NameRenderFlag) FormattedName {

	f := FormattedName{
		Name:       c.Name,
		Type:       uType,
		Adjectives: make([]string, 0, len(c.Adjectives)),
	}

	includeHealth := false
	for _, flag := range renderFlags {
		if flag == RenderHealth {
			includeHealth = true
		} else if flag == RenderShortAdjectives {
			f.UseShortAdjectives = true
		}
	}

	// If including health, only do so if not downed, because downed shows as its own adjective.
	if includeHealth && c.Health > 0 {
		pctHealth := int(math.Ceil(float64(c.Health) / float64(c.HealthMax.Value) * 100))
		f.Adjectives = append(f.Adjectives, strconv.Itoa(pctHealth)+`%`)
	}

	f.Adjectives = append(f.Adjectives, c.GetAdjectives()...)

	if c.Health < 1 {
		f.Suffix = `downed`
	} else if c.Aggro != nil && c.Aggro.UserId == viewingUserId {
		f.Suffix = `aggro`
	}

	if c.Pet.Exists() {
		f.PetName = c.Pet.DisplayName()
	}

	return f
}

func (c *Character) PruneCooldowns() {
	if len(c.Cooldowns) == 0 {
		return
	}

	c.Cooldowns.Prune()
}

func (c *Character) GetCooldown(trackingTag string) int {
	if c.Cooldowns == nil {
		c.Cooldowns = make(Cooldowns)
	}
	return c.Cooldowns[trackingTag]
}

func (c *Character) GetAllCooldowns() map[string]int {

	ret := map[string]int{}

	if c.Cooldowns == nil {
		return ret
	}

	maps.Copy(ret, c.Cooldowns)

	return ret
}

func (c *Character) TryCooldown(trackingTag string, cooldownTime string) bool {
	if c.Cooldowns == nil {
		c.Cooldowns = make(Cooldowns)
	}

	return c.Cooldowns.Try(trackingTag, cooldownTime)
}

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

func (c *Character) StoreItem(i items.Item) bool {
	if i.ItemId < 1 {
		return false
	}

	i.Validate()

	// Check if adding this item would exceed carry capacity
	newWeight := c.GetCarriedWeight() + i.GetSpec().GetWeight()
	capacity := c.CarryCapacity()

	// Allow up to 2x capacity (overloaded, but possible)
	if newWeight > capacity*2.0 {
		return false
	}

	c.Items = append(c.Items, i)

	return true
}

func (c *Character) RemoveItem(i items.Item) bool {
	for j := len(c.Items) - 1; j >= 0; j-- {
		if c.Items[j].Equals(i) {
			c.Items = append(c.Items[:j], c.Items[j+1:]...)
			return true
		}
	}
	return false
}

func (c *Character) HandsRequired(i items.Item) int {

	if i.ItemId < 1 {
		return 0
	}

	iSpec := i.GetSpec()

	// Shooting weapnos don't benefit from creature size
	// when determining how many hands they require
	if iSpec.Subtype == items.Shooting {
		return iSpec.Hands
	}

	speciesInfo := species.GetSpecies(c.SpeciesId)
	if speciesInfo.Size == species.Large {
		return 1
	}

	if speciesInfo.Size == species.Small {
		return iSpec.Hands + 1
	}

	return iSpec.Hands
}

// Copies over an existing item with a new item
// Returns true if successfully replaces an item
func (c *Character) UpdateItem(originalItm items.Item, replacement items.Item) bool {
	for j := len(c.Items) - 1; j >= 0; j-- {
		if c.Items[j].Equals(originalItm) {
			// If the number of uses remaining has decremented from the original item
			// The item gets destroyed from existence
			if originalItm.Uses >= 1 && replacement.Uses < 1 {
				c.Items = append(c.Items[:j], c.Items[j+1:]...)
			} else {
				c.Items[j] = replacement
			}
			return true
		}
	}
	return false
}

func (c *Character) UseItem(i items.Item) int {
	for j := len(c.Items) - 1; j >= 0; j-- {
		if c.Items[j].Equals(i) {
			usesLeft := c.Items[j].Uses
			if usesLeft > 0 {
				usesLeft--
			}
			if usesLeft <= 0 {
				c.Items = append(c.Items[:j], c.Items[j+1:]...)
			} else {
				c.Items[j].Uses = usesLeft
				c.Items[j].LastUsedRound = util.GetRoundCount()
			}

			return usesLeft
		}
	}

	return 0
}

func (c *Character) FindInBackpack(itemName string) (items.Item, bool) {

	if itemName == `` {
		return items.Item{}, false
	}

	closeMatchItem, matchItem := items.FindMatchIn(itemName, c.Items...)

	if matchItem.ItemId != 0 {
		return matchItem, true
	}

	if closeMatchItem.ItemId != 0 {
		return closeMatchItem, true
	}

	return items.Item{}, false
}

func (c *Character) FindOnBody(itemName string) (items.Item, bool) {

	if itemName == `` {
		return items.Item{}, false
	}

	partialMatch, fullMatch := items.FindMatchIn(itemName,
		c.Equipment.Weapon,
		c.Equipment.Offhand,
		c.Equipment.ExtraArm1,
		c.Equipment.ExtraArm2,
		c.Equipment.Head,
		c.Equipment.Neck,
		c.Equipment.Body,
		c.Equipment.Belt,
		c.Equipment.Gloves,
		c.Equipment.Ring,
		c.Equipment.Legs,
		c.Equipment.Feet)

	if fullMatch.ItemId != 0 {
		return fullMatch, true
	}

	if partialMatch.ItemId != 0 {
		return partialMatch, true
	}

	return items.Item{}, false
}

func (c *Character) GetSkills() map[string]int {
	skillResults := make(map[string]int)
	for skillName, skillLevel := range c.Skills {
		skillResults[skillName] = skillLevel
	}
	return skillResults
}

func (c *Character) SetSkill(skillName string, level int) {
	if c.Skills == nil {
		c.Skills = make(map[string]int)
	}
	skillName = strings.ToLower(skillName)

	if level == 0 {
		delete(c.Skills, skillName)
		return
	}

	c.Skills[skillName] = level
}

// Increases the skill training counter and returns the new value
func (c *Character) TrainSkill(skillName string, targetLevel ...int) int {
	if c.Skills == nil {
		c.Skills = make(map[string]int)
	}

	skillName = strings.ToLower(skillName)

	skillLevel := 0

	if lvl, ok := c.Skills[skillName]; ok {
		skillLevel = lvl
	}

	if len(targetLevel) > 0 {

		if skillLevel < targetLevel[0] {
			skillLevel = targetLevel[0]
		}

	} else {

		skillLevel++

	}

	c.Skills[skillName] = skillLevel

	return skillLevel
}

// Gets the current value of the skillname provided
func (c *Character) GetSkillLevel(skillName skills.SkillTag) int {
	if c.Skills == nil {
		c.Skills = make(map[string]int)
	}

	if level, ok := c.Skills[string(skillName)]; ok {
		return level
	}
	return 0
}

func (c *Character) GetSkillLevelCost(currentLevel int) int {
	return currentLevel
}

// IncreaseSkill increments the named skill by 1.
// No hard cap — progression is governed by the soft cap in CheckSkillProgression.
func (c *Character) IncreaseSkill(skillName string) bool {
	if c.Skills == nil {
		c.Skills = make(map[string]int)
	}
	c.Skills[skillName] = c.Skills[skillName] + 1
	return true
}

// IncreaseStat increments the Training field of the named stat by the given amount,
// then recalculates derived values via Validate.
func (c *Character) IncreaseStat(statName string, amount int) bool {
	switch statName {
	case "strength":
		c.Stats.Strength.Training += amount
	case "dexterity":
		c.Stats.Dexterity.Training += amount
	case "perception":
		c.Stats.Perception.Training += amount
	case "vitality":
		c.Stats.Vitality.Training += amount
	case "willpower":
		c.Stats.Willpower.Training += amount
	case "charisma":
		c.Stats.Charisma.Training += amount
	default:
		return false
	}
	c.Validate()
	return true
}

// GetStatValue returns the raw computed Value for the named stat, or 0 if unrecognised.
func (c *Character) GetStatValue(statName string) int {
	switch statName {
	case "strength":
		return c.Stats.Strength.Value
	case "dexterity":
		return c.Stats.Dexterity.Value
	case "perception":
		return c.Stats.Perception.Value
	case "vitality":
		return c.Stats.Vitality.Value
	case "willpower":
		return c.Stats.Willpower.Value
	case "charisma":
		return c.Stats.Charisma.Value
	}
	return 0
}

// GetCombatSkillTag returns the appropriate combat skill tag based on
// the character's equipped weapon type.
func (c *Character) GetCombatSkillTag() skills.SkillTag {
	if c.Equipment.Weapon.ItemId > 0 {
		weaponSpec := c.Equipment.Weapon.GetSpec()
		if weaponSpec.Subtype == items.Shooting {
			return skills.RangedCombat
		}
		if weaponSpec.Subtype != items.Claws {
			return skills.WeaponCombat
		}
	}
	return skills.UnarmedCombat
}

// GetCombatSkillLevel returns an effective combat skill value for use in
// combat formulas. Checks the weapon-appropriate DOG skill first, then
// falls back to legacy Brawling, then minimum 1.
func (c *Character) GetCombatSkillLevel() int {
	if level := c.GetSkillLevel(c.GetCombatSkillTag()); level > 0 {
		return level
	}
	return 1
}

// GetModifiedAttackCount calculates the number of attacks for a weapon
// considering speed multiplier, skill, and dual wielding.
// baseAttacks: The weapon's base attack count
// weaponSpeed: The weapon's speed multiplier (1.0 = unarmed baseline)
// isOffhand: Whether this is the offhand weapon
func (c *Character) GetModifiedAttackCount(baseAttacks int, weaponSpeed float64, isOffhand bool) int {
	attacks := float64(baseAttacks)

	// Apply weapon speed multiplier
	attacks *= weaponSpeed

	// Apply skill modifier (small bonus, max ~10% at skill 50)
	skillLevel := float64(c.GetCombatSkillLevel())
	skillMod := 1.0 + (skillLevel / 50.0) * 0.1
	attacks *= skillMod

	// If offhand, weapon-combat skill governs dual-wield effectiveness
	if isOffhand {
		wcLevel := float64(c.GetSkillLevel(skills.WeaponCombat))
		// Significant modifier: 0.5 at skill 0, 1.0 at skill 25, 1.2 at skill 50
		dualWieldMod := 0.5 + (wcLevel / 50.0) * 0.7
		attacks *= dualWieldMod
	}

	// Minimum 1 attack
	result := int(math.Round(attacks))
	if result < 1 {
		result = 1
	}

	return result
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

// HasShield returns true if the character is wielding a shield in offhand
func (c *Character) HasShield() bool {
	if c.Equipment.Offhand.ItemId <= 0 {
		return false
	}
	spec := c.Equipment.Offhand.GetSpec()
	// Shield detection: type offhand + has damage reduction or subtype wearable
	return spec.Type == items.Offhand && (spec.DamageReduction > 0 || spec.Subtype == items.Wearable)
}

// IsDualWielding returns true if character has weapons in both hands
func (c *Character) IsDualWielding() bool {
	if c.Equipment.Weapon.ItemId <= 0 || c.Equipment.Offhand.ItemId <= 0 {
		return false
	}
	// Dual wielding means both are weapons
	weaponSpec := c.Equipment.Weapon.GetSpec()
	offhandSpec := c.Equipment.Offhand.GetSpec()
	return weaponSpec.Type == items.Weapon && offhandSpec.Type == items.Weapon
}

// IsUnarmed returns true if character has no weapon equipped
func (c *Character) IsUnarmed() bool {
	return c.Equipment.Weapon.ItemId <= 0
}

// GetDefenseSequence returns ordered defenses based on equipment (Stage 7.1)
func (c *Character) GetDefenseSequence() []string {
	defenses := []string{}

	// Everyone can dodge
	defenses = append(defenses, DefenseDodge)

	// If unarmed, only dodge
	if c.IsUnarmed() {
		return defenses
	}

	// If dual wielding (two weapons)
	if c.IsDualWielding() {
		// dodge → parry main → parry off
		defenses = append(defenses, DefenseParry) // main hand parry
		defenses = append(defenses, DefenseParry) // offhand parry
		return defenses
	}

	// If wielding weapon + shield
	if c.Equipment.Weapon.ItemId > 0 && c.HasShield() {
		// dodge → parry → block
		defenses = append(defenses, DefenseParry)
		defenses = append(defenses, DefenseBlock)
		return defenses
	}

	// If wielding single weapon (no offhand or offhand is not shield/weapon)
	if c.Equipment.Weapon.ItemId > 0 {
		// dodge → parry
		defenses = append(defenses, DefenseParry)
		return defenses
	}

	// Default: just dodge
	return defenses
}

// GetDefenseScore calculates defense score for a given defense type (Stage 7.1)
func (c *Character) GetDefenseScore(defenseType string) float64 {
	dex := float64(c.Stats.Dexterity.ValueAdj)
	skillWeight := float64(configs.GetBalanceConfig().SkillWeight)

	switch defenseType {
	case DefenseDodge:
		// Dodge: Dexterity + UnarmedCombat skill + mutation dodge modifier
		unarmedSkill := float64(c.GetSkillLevel(skills.UnarmedCombat)) * skillWeight
		score := dex + unarmedSkill + mutations.GetDodgeModifier(c.Mutations)
		// Phase 24.5: Blinded condition reduces dodge
		if c.HasCondition(ConditionBlinded) {
			score *= c.GetConditionMagnitude(ConditionBlinded) // magnitude is 0.5–0.7 = penalty multiplier
		}
		return score

	case DefenseParry:
		// Parry: Dexterity + WeaponCombat skill + weapon ParryRating
		weaponSkill := float64(c.GetSkillLevel(skills.WeaponCombat)) * skillWeight
		parryRating := 0
		if c.Equipment.Weapon.ItemId > 0 {
			parryRating = c.Equipment.Weapon.GetSpec().ParryRating
		}
		return dex + weaponSkill + float64(parryRating)

	case DefenseBlock:
		// Block: (Strength + Dexterity)/2 + WeaponCombat skill + shield BlockRating
		str := float64(c.Stats.Strength.ValueAdj)
		weaponSkill := float64(c.GetSkillLevel(skills.WeaponCombat)) * skillWeight
		blockRating := 0
		if c.HasShield() {
			blockRating = c.Equipment.Offhand.GetSpec().BlockRating
		}
		return (str+dex)/2 + weaponSkill + float64(blockRating)

	default:
		return 0
	}
}

// GetDefenseStaminaCost returns stamina cost for a defense type (Stage 7.1)
func (c *Character) GetDefenseStaminaCost(defenseType string) int {
	bal := configs.GetBalanceConfig()

	baseCost := 0
	multiplier := 1.0

	switch defenseType {
	case DefenseDodge:
		baseCost = 2
		multiplier = float64(bal.DodgeMultiplier)
	case DefenseParry:
		baseCost = 4
		multiplier = float64(bal.ParryMultiplier)
	case DefenseBlock:
		baseCost = 5
		multiplier = float64(bal.BlockMultiplier)
	default:
		return 0
	}

	cost := int(float64(baseCost) * multiplier)
	if cost < 1 {
		cost = 1 // minimum 1 stamina
	}
	return cost
}

// DeductDefenseStamina deducts stamina for a defense and returns true if successful (Stage 7.1)
func (c *Character) DeductDefenseStamina(defenseType string) bool {
	cost := c.GetDefenseStaminaCost(defenseType)
	if c.Stamina >= cost {
		c.Stamina -= cost
		return true
	}
	return false
}

func (c *Character) GetMaxCharmedCreatures() int {
	// Taming is now handled via spellcasting; base charm capacity from spellcasting skill
	lvl := c.GetSkillLevel(skills.Spellcasting)
	return lvl + 1
}

func (c *Character) GetMemoryCapacity() int {
	// Map is now a free command; memory capacity based on Perception
	memCap := (c.Stats.Perception.ValueAdj >> 1)
	if memCap < 0 {
		memCap = 0
	}
	return memCap + 5
}

func (c *Character) GetMapSprawlCapacity() int {
	// Map is now a free command; sprawl capacity based on Perception
	sprawlCap := (c.Stats.Perception.ValueAdj >> 2)
	if sprawlCap < 0 {
		sprawlCap = 0
	}
	return sprawlCap
}

// Remember visiting a room. This may cause to forget an older room if the memory is full.
func (c *Character) RememberRoom(roomId int) {
	mapHistory := c.GetMemoryCapacity()
	if len(c.roomHistory) >= mapHistory*2 {
		// Prune out everything except {mapHistory}-1 items at the end
		c.roomHistory = c.roomHistory[len(c.roomHistory)-(mapHistory-1):]
	}
	c.roomHistory = append(c.roomHistory, roomId)
}

func (c *Character) IsQuestDone(questToken string) bool {
	testQuestId, _ := quests.TokenToParts(questToken)
	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}

	stage := c.QuestProgress[testQuestId]

	return stage == `end`
}

func (c *Character) HasQuest(questToken string) bool {

	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}

	testQuestId, testQuestStep := quests.TokenToParts(questToken)

	currentStep, ok := c.QuestProgress[testQuestId]
	if !ok {
		return false
	}

	// If on that step currently, then true
	if currentStep == testQuestStep {
		return true
	}

	currentToken := quests.PartsToToken(testQuestId, currentStep)

	// If the current token comes after the test token then they've already done that quest
	return quests.IsTokenAfter(questToken, currentToken)
}

func (c *Character) GetQuestProgress() map[int]string {

	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}

	retMap := make(map[int]string)
	for questId, stepName := range c.QuestProgress {
		retMap[questId] = stepName
	}
	return retMap
}

func (c *Character) GiveQuestToken(questToken string) bool {

	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}

	questId, newStep := quests.TokenToParts(questToken)
	currentProgress := c.QuestProgress[questId]

	currentToken := quests.PartsToToken(questId, currentProgress)

	if quests.IsTokenAfter(currentToken, questToken) {
		c.QuestProgress[questId] = newStep
		return true
	}

	return false
}

func (c *Character) ClearQuestToken(questToken string) {

	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}

	questId, _ := quests.TokenToParts(questToken)

	delete(c.QuestProgress, questId)
}

func (c *Character) SetAggroRemote(exitName string, userId int, mobInstanceId int, aggroType AggroType, roundsWaitTime ...int) {
	c.SetAggro(userId, mobInstanceId, aggroType, roundsWaitTime...)
	c.Aggro.ExitName = exitName
}

func (c *Character) SetAggro(userId int, mobInstanceId int, aggroType AggroType, roundsWaitTime ...int) {

	// Stage 8.3: Clear grapple state if switching targets
	if c.Aggro != nil {
		if c.Aggro.UserId != userId || c.Aggro.MobInstanceId != mobInstanceId {
			c.ClearGrappleState()
		}
	}

	var combatAddlWaitRounds int = 0

	if len(roundsWaitTime) > 0 {
		for _, waitAmt := range roundsWaitTime {
			combatAddlWaitRounds += waitAmt
		}
	} else {
		combatAddlWaitRounds = c.Equipment.Weapon.GetSpec().WaitRounds + c.Equipment.Offhand.GetSpec().WaitRounds
	}

	if aggroType == DefaultAttack {
		if c.Equipment.Weapon.GetSpec().Subtype == items.Shooting {
			aggroType = Shooting
		}
	}

	c.Aggro = &Aggro{
		UserId:        userId,
		MobInstanceId: mobInstanceId,
		Type:          aggroType,
		RoundsWaiting: combatAddlWaitRounds,
	}

}

func (c *Character) SetCast(roundsWaitTime int, sInfo SpellAggroInfo) {

	c.Aggro = &Aggro{
		Type:          SpellCast,
		RoundsWaiting: roundsWaitTime,
		SpellInfo:     sInfo,
	}

}

func (c *Character) EndAggro() {
	c.Aggro = nil
	c.ClearGrappleState()
}

// ClearGrappleState clears all grapple-related state
// Stage 8.3: Called when combat ends, targets change, or participant dies
func (c *Character) ClearGrappleState() {
	c.GrappleControllerId = 0
	c.RemoveCondition(ConditionGrappleController)
	// Reset to standing if in a grapple position
	if c.CombatPosition.IsGrapplePosition() {
		c.CombatPosition = PositionStanding
	}
}

func (c *Character) IsAggro(targetUserId int, targetMobInstanceId int) bool {

	if c.Aggro != nil {

		if c.Aggro.MobInstanceId > 0 && c.Aggro.MobInstanceId == targetMobInstanceId {
			return true
		}

		if c.Aggro.UserId > 0 && c.Aggro.UserId == targetUserId {
			return true
		}

		if c.Aggro.Type == SpellCast {
			if len(c.Aggro.SpellInfo.TargetUserIds) > 0 {
				for _, uId := range c.Aggro.SpellInfo.TargetUserIds {
					if uId == targetUserId {
						return true
					}
				}
			}

			if len(c.Aggro.SpellInfo.TargetMobInstanceIds) > 0 {
				for _, mId := range c.Aggro.SpellInfo.TargetMobInstanceIds {
					if mId == targetMobInstanceId {
						return true
					}
				}
			}
		}

	}
	return false
}

func (c *Character) IsDisabled() bool {
	return c.Health <= 0
}

func (c *Character) HasBuffFlag(buffFlag buffs.Flag) bool {
	return c.Buffs.HasFlag(buffFlag, false)
}

// HasFlagFromAnySource returns true if the character has the given flag from
// either active buffs OR permanent mutation effects. Use this instead of
// HasBuffFlag when the check should also honor mutation-granted flags.
func (c *Character) HasFlagFromAnySource(buffFlag buffs.Flag) bool {
	if c.Buffs.HasFlag(buffFlag, false) {
		return true
	}
	return mutations.HasMutationFlag(c.Mutations, string(buffFlag))
}

func (c *Character) CancelBuffsWithFlag(buffFlag buffs.Flag) bool {
	if c.Buffs.HasFlag(buffFlag, true) {
		c.Validate(true)
		return true
	}
	return false
}

func (c *Character) HasBuff(buffId int) bool {
	return c.Buffs.HasBuff(buffId)
}

func (c *Character) AddBuff(buffId int, isPermanent bool) error {
	buffId = int(math.Abs(float64(buffId)))
	if !c.Buffs.AddBuff(buffId, isPermanent) {
		return fmt.Errorf(`failed to add buff. target: "%s" buffId: %d`, c.Name, buffId)
	}
	c.Validate()
	return nil
}

func (c *Character) TrackBuffStarted(buffId int) {
	c.Buffs.Started(buffId)
}

func (c *Character) GetBuffs(buffId ...int) []*buffs.Buff {
	return c.Buffs.GetBuffs(buffId...)
}

func (c *Character) RemoveBuff(buffId int) {
	buffId = int(math.Abs(float64(buffId)))
	c.Buffs.RemoveBuff(buffId)
	c.Validate()
}

func (c *Character) TimerSet(name, period string) {
	if c.Timers == nil {
		c.Timers = map[string]gametime.RoundTimer{}
	}
	c.Timers[name] = gametime.RoundTimer{
		RoundStart: util.GetRoundCount(),
		Period:     period,
	}
}

func (c *Character) TimerExpired(name string) bool {
	if c.Timers == nil {
		return true
	}

	t, ok := c.Timers[name]

	if !ok {
		return true
	}

	if t.Expired() {
		delete(c.Timers, name)
		return true
	}

	return false
}

func (c *Character) TimerExists(name string) bool {
	if c.Timers == nil {
		return false
	}

	_, ok := c.Timers[name]
	return ok
}

func (c *Character) ApplyHealthChange(healthChange int) int {
	oldHealth := c.Health
	newHealth := c.Health + healthChange
	if newHealth < 0 {
		c.CancelBuffsWithFlag(buffs.CancelIfCombat)

		// If they haven't dropped yet, require a drop before going straight to death.
		// Don't allow players to drop under -5 in a single hit.
		if newHealth < -5 && oldHealth > 0 {
			newHealth = -5
		} else if newHealth <= -10 {
			newHealth = -10
		}
	} else if newHealth > c.HealthMax.Value {
		newHealth = c.HealthMax.Value
	}

	c.Health = newHealth

	return newHealth - oldHealth
}

func (c *Character) BarterPrice(startPrice int) int {
	factor := (float64(c.Stats.Charisma.ValueAdj) / 3) / 100 // 100 = 33% discount, 0 = 0% discount, 300 = 100% discount
	if factor > .75 {
		factor = .75
	}
	return int(factor * float64(startPrice))
}

func (c *Character) Heal(hp int) int {
	startHP := c.Health

	c.Health += hp
	if c.Health > c.HealthMax.Value {
		c.Health = c.HealthMax.Value
	}

	return c.Health - startHP
}

func (c *Character) HealthPerRound() int {
	b := configs.GetBalanceConfig()
	pct := float64(b.PlayerHealthRegenPct)
	if c.IsMob {
		pct = float64(b.MobHealthRegenPct)
	}
	// StatMod reinterpreted as percentage bonus (e.g. 5 → +5%)
	pct += float64(c.StatMod(string(statmods.HealthRecovery))) / 100.0
	base := int(pct * float64(c.HealthMax.Value))
	if base < 1 {
		base = 1
	}
	return base
}

func (c *Character) StaminaPerRound() int {
	b := configs.GetBalanceConfig()
	pct := float64(b.PlayerStaminaRegenPct)
	if c.IsMob {
		pct = float64(b.MobStaminaRegenPct)
	}
	pct += float64(c.StatMod(string(statmods.StaminaRecovery))) / 100.0
	base := int(pct * float64(c.StaminaMax.Value))
	if base < 1 {
		base = 1
	}
	// Apply stamina_regen_multiplier mutations
	if mult := mutations.GetStaminaRegenMultiplier(c.Mutations); mult != 0 {
		base = int(float64(base) * (1.0 + mult))
		if base < 1 {
			base = 1
		}
	}
	return base
}

func (c *Character) ConvictionPerRound() int {
	b := configs.GetBalanceConfig()
	pct := float64(b.PlayerConvictionRegenPct)
	if c.IsMob {
		pct = float64(b.MobConvictionRegenPct)
	}
	pct += float64(c.StatMod(string(statmods.ConvictionRecovery))) / 100.0
	base := int(pct * float64(c.ConvictionMax.Value))
	if base < 1 {
		base = 1
	}
	return base
}

// Where 1000 = a full round
func (c *Character) MovementCost() int {
	modifier := 3                                    // by default they should be able to move 3 times per round.
	modifier += int(c.Stats.Dexterity.ValueAdj / 15) // Every 15 dexterity, get an extra movement
	return int(1000 / modifier)
}

func (c *Character) StatMod(statName string) int {
	return c.Equipment.StatMod(statName) + c.Buffs.StatMod(statName) + c.Pet.StatMod(statName)
}

// returns true if something has changed.
func (c *Character) RecalculateStats() {

	// Make sure racial base stats are set
	beforeHealthMax := c.HealthMax
	beforeStats := c.Stats

	if speciesInfo := species.GetSpecies(c.SpeciesId); speciesInfo != nil {

		// Only set base stats from racial if they haven't been rolled yet
		// (Base values of 0 indicate uninitialized stats)
		// Rolled stats (from RollCharacterStats) will be 85-115, so they won't be overwritten
		if c.Stats.Strength.Base == 0 {
			c.Stats.Strength.Base = speciesInfo.Stats.Strength.Base
		}
		if c.Stats.Dexterity.Base == 0 {
			c.Stats.Dexterity.Base = speciesInfo.Stats.Dexterity.Base
		}
		if c.Stats.Perception.Base == 0 {
			c.Stats.Perception.Base = speciesInfo.Stats.Perception.Base
		}
		if c.Stats.Vitality.Base == 0 {
			c.Stats.Vitality.Base = speciesInfo.Stats.Vitality.Base
		}
		if c.Stats.Willpower.Base == 0 {
			c.Stats.Willpower.Base = speciesInfo.Stats.Willpower.Base
		}
		if c.Stats.Charisma.Base == 0 {
			c.Stats.Charisma.Base = speciesInfo.Stats.Charisma.Base
		}
	}

	// Add any mods for equipment
	c.Stats.Strength.Mods = c.StatMod(string(statmods.Strength))
	c.Stats.Dexterity.Mods = c.StatMod(string(statmods.Dexterity))
	c.Stats.Perception.Mods = c.StatMod(string(statmods.Perception))
	c.Stats.Vitality.Mods = c.StatMod(string(statmods.Vitality))
	c.Stats.Willpower.Mods = c.StatMod(string(statmods.Willpower))
	c.Stats.Charisma.Mods = c.StatMod(string(statmods.Charisma))

	// Stage 12.1: Apply stat_flat mutation bonuses to Mods before Recalculate()
	c.Stats.Strength.Mods += mutations.GetStatFlat(c.Mutations, "strength")
	c.Stats.Dexterity.Mods += mutations.GetStatFlat(c.Mutations, "dexterity")
	c.Stats.Perception.Mods += mutations.GetStatFlat(c.Mutations, "perception")
	c.Stats.Vitality.Mods += mutations.GetStatFlat(c.Mutations, "vitality")
	c.Stats.Willpower.Mods += mutations.GetStatFlat(c.Mutations, "willpower")
	c.Stats.Charisma.Mods += mutations.GetStatFlat(c.Mutations, "charisma")

	// Recalculate stats
	// Stats are basically:
	// level*base + training + mods
	c.Stats.Strength.Recalculate()
	c.Stats.Dexterity.Recalculate()
	c.Stats.Perception.Recalculate()
	c.Stats.Vitality.Recalculate()
	c.Stats.Willpower.Recalculate()
	c.Stats.Charisma.Recalculate()

	// Stage 12.1: Apply stat_multiplier mutations after Recalculate()
	if v := mutations.GetStatMultiplier(c.Mutations, "strength"); v != 0 {
		c.Stats.Strength.ValueAdj = int(float64(c.Stats.Strength.ValueAdj) * (1.0 + v))
	}
	if v := mutations.GetStatMultiplier(c.Mutations, "dexterity"); v != 0 {
		c.Stats.Dexterity.ValueAdj = int(float64(c.Stats.Dexterity.ValueAdj) * (1.0 + v))
	}
	if v := mutations.GetStatMultiplier(c.Mutations, "perception"); v != 0 {
		c.Stats.Perception.ValueAdj = int(float64(c.Stats.Perception.ValueAdj) * (1.0 + v))
	}
	if v := mutations.GetStatMultiplier(c.Mutations, "vitality"); v != 0 {
		c.Stats.Vitality.ValueAdj = int(float64(c.Stats.Vitality.ValueAdj) * (1.0 + v))
	}
	if v := mutations.GetStatMultiplier(c.Mutations, "willpower"); v != 0 {
		c.Stats.Willpower.ValueAdj = int(float64(c.Stats.Willpower.ValueAdj) * (1.0 + v))
	}
	if v := mutations.GetStatMultiplier(c.Mutations, "charisma"); v != 0 {
		c.Stats.Charisma.ValueAdj = int(float64(c.Stats.Charisma.ValueAdj) * (1.0 + v))
	}

	// Set HP/Stamina/Conviction maxes (skill-based, no level dependency)
	// This relies on the above stats so has to be calculated afterwards
	rb := configs.GetBalanceConfig()
	c.HealthMax.Mods = int(rb.HealthBase) +
		c.StatMod(string(statmods.HealthMax)) + // Any sort of spell buffs etc. are just direct modifiers
		c.Stats.Strength.ValueAdj*int(rb.HealthPerStrength) + // Strength contributes to health
		c.Stats.Vitality.ValueAdj*int(rb.HealthPerVitality) // Vitality is primary health stat

	c.StaminaMax.Mods = int(rb.StaminaBase) +
		c.Stats.Strength.ValueAdj*int(rb.StaminaPerStrength) + // Strength contributes to stamina
		c.Stats.Willpower.ValueAdj*int(rb.StaminaPerWillpower) + // Willpower contributes to stamina
		c.Stats.Vitality.ValueAdj*int(rb.StaminaPerVitality) // Vitality is primary stamina stat

	c.ConvictionMax.Mods = int(rb.ConvictionBase) +
		(c.Stats.Willpower.ValueAdj+c.Stats.Charisma.ValueAdj)*int(rb.ConvictionPerWilCha) // Willpower+Charisma drive conviction

	// Set max action points
	c.ActionPointsMax.Mods = 200 // hard coded for now

	// Recalculate HP/Stamina/Conviction stats
	c.HealthMax.Recalculate()
	c.StaminaMax.Recalculate()
	c.ConvictionMax.Recalculate()
	c.ActionPointsMax.Recalculate()

	// Stage 12.1: Apply health_multiplier mutations after HealthMax.Recalculate()
	if hMult := mutations.GetHealthMultiplier(c.Mutations); hMult != 0 {
		c.HealthMax.Value = int(float64(c.HealthMax.Value) * (1.0 + hMult))
		if c.HealthMax.Value < 1 {
			c.HealthMax.Value = 1
		}
	}

	// HP can't max less than 1, Stamina/Conviction can't max less than 0
	if c.StaminaMax.Value < 0 {
		c.StaminaMax.Value = 0
	}
	if c.ConvictionMax.Value < 0 {
		c.ConvictionMax.Value = 0
	}
	if c.HealthMax.Value < 1 {
		c.HealthMax.Value = 1
	}
	if c.ActionPointsMax.Value < 50 {
		c.ActionPointsMax.Value = 50
	}

	// Chrysalis enchantment pool reservation: clamp current pools to effective max
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

	// Stage 31.6: Enchant withdrawal condition — temporarily reduces pool max
	if c.HasCondition(ConditionEnchantWithdrawal) {
		mag := c.GetConditionMagnitude(ConditionEnchantWithdrawal)
		// Source stores which pool to penalize
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

	if c.userId != 0 {
		changed := false
		// return true if something has changed.
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
		pct := enchantments.GetTierReservePct(itm.EnchantType, itm.EnchantTier)
		total += int(math.Floor(float64(poolMax) * pct))
	}
	return total
}

func (c *Character) CanDualWield() bool {
	// Dual wielding is now governed by weapon-combat skill
	return c.GetSkillLevel(skills.WeaponCombat) > 0
}

// Returns whether a correction was in order
func (c *Character) Validate(recalcPermaBuffs ...bool) error {

	// ── Skill rename migrations ─────────────────────────────────
	// Rename legacy skill keys so existing saves pick up new names.
	if c.Skills != nil {
		if v, ok := c.Skills["stealth"]; ok {
			c.Skills["skullduggery"] = v
			delete(c.Skills, "stealth")
		}
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

	// Derive ExtraArms from mutation level (capped at 2)
	if lvl, ok := c.Mutations["extra-arms"]; ok && lvl > 0 {
		c.ExtraArms = lvl
		if c.ExtraArms > 2 {
			c.ExtraArms = 2
		}
	} else {
		c.ExtraArms = 0
	}

	if c.Zone == "" {
		c.Zone = startingZone
	}

	if c.Name == "" {
		c.Name = defaultName
	}
	c.Buffs.Validate()

	// Migrate tracking/foraging → search (must run before ensureAllSkills)
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

	// Ensure all known skills exist at rank 1 minimum (retroactive for existing characters)
	c.Skills = ensureAllSkills(c.Skills)

	// Do a stats recalc based on equipment, race, level, etc.
	c.RecalculateStats()

	// Recalculate health, stamina, and conviction

	if c.Stamina > c.StaminaMax.Value {
		c.Stamina = c.StaminaMax.Value
	}
	if c.Conviction > c.ConvictionMax.Value {
		c.Conviction = c.ConvictionMax.Value
	}
	if c.Health > c.HealthMax.Value {
		c.Health = c.HealthMax.Value
	}

	if c.Health < -10 {
		c.Health = -10
	}

	if c.Stamina < 0 {
		c.Stamina = 0
	}
	if c.Conviction < 0 {
		c.Conviction = 0
	}

	c.Cooldowns.Prune()

	// Validate possessed/worn items
	// This helps ensure all in-play items have a uid
	for i := range c.Items {
		c.Items[i].Validate()
	}
	c.Equipment.Weapon.Validate()
	c.Equipment.Offhand.Validate()
	c.Equipment.ExtraArm1.Validate()
	c.Equipment.ExtraArm2.Validate()
	c.Equipment.Head.Validate()
	c.Equipment.Neck.Validate()
	c.Equipment.Body.Validate()
	c.Equipment.Belt.Validate()
	c.Equipment.Gloves.Validate()
	c.Equipment.Ring.Validate()
	c.Equipment.Legs.Validate()
	c.Equipment.Feet.Validate()
	// Done with validation

	if speciesInfo := species.GetSpecies(c.SpeciesId); speciesInfo != nil {

		c.Equipment.EnableAll()

		// Are there slots that SHOULD be disabled?
		if len(speciesInfo.DisabledSlots) > 0 {

			for _, disabledSlot := range speciesInfo.DisabledSlots {

				var itemFoundInDisabledSlot items.Item = items.ItemDisabledSlot

				switch items.ItemType(disabledSlot) {
				case items.Weapon:
					if c.Equipment.Weapon.ItemId > 0 { // Did we find somethign in a disabled slot?
						itemFoundInDisabledSlot = c.Equipment.Weapon
					}
					c.Equipment.Weapon = items.ItemDisabledSlot
				case items.Offhand:
					if c.Equipment.Offhand.ItemId > 0 { // Did we find somethign in a disabled slot?
						itemFoundInDisabledSlot = c.Equipment.Offhand
					}
					c.Equipment.Offhand = items.ItemDisabledSlot
				case items.Head:
					if c.Equipment.Head.ItemId > 0 { // Did we find somethign in a disabled slot?
						itemFoundInDisabledSlot = c.Equipment.Head
					}
					c.Equipment.Head = items.ItemDisabledSlot
				case items.Neck:
					if c.Equipment.Neck.ItemId > 0 { // Did we find somethign in a disabled slot?
						itemFoundInDisabledSlot = c.Equipment.Neck
					}
					c.Equipment.Neck = items.ItemDisabledSlot
				case items.Body:
					if c.Equipment.Body.ItemId > 0 { // Did we find somethign in a disabled slot?
						itemFoundInDisabledSlot = c.Equipment.Body
					}
					c.Equipment.Body = items.ItemDisabledSlot
				case items.Belt:
					if c.Equipment.Belt.ItemId > 0 { // Did we find somethign in a disabled slot?
						itemFoundInDisabledSlot = c.Equipment.Belt
					}
					c.Equipment.Belt = items.ItemDisabledSlot
				case items.Gloves:
					if c.Equipment.Gloves.ItemId > 0 { // Did we find somethign in a disabled slot?
						itemFoundInDisabledSlot = c.Equipment.Gloves
					}
					c.Equipment.Gloves = items.ItemDisabledSlot
				case items.Ring:
					if c.Equipment.Ring.ItemId > 0 { // Did we find somethign in a disabled slot?
						itemFoundInDisabledSlot = c.Equipment.Ring
					}
					c.Equipment.Ring = items.ItemDisabledSlot
				case items.Legs:
					if c.Equipment.Legs.ItemId > 0 { // Did we find somethign in a disabled slot?
						itemFoundInDisabledSlot = c.Equipment.Legs
					}
					c.Equipment.Legs = items.ItemDisabledSlot
				case items.Feet:
					if c.Equipment.Feet.ItemId > 0 { // Did we find somethign in a disabled slot?
						itemFoundInDisabledSlot = c.Equipment.Feet
					}
					c.Equipment.Feet = items.ItemDisabledSlot
				}

				if !itemFoundInDisabledSlot.IsDisabled() {
					c.StoreItem(itemFoundInDisabledSlot)
					mudlog.Debug("Disabled Check", "error", "Item found in disabled slot", "name", itemFoundInDisabledSlot.Name(), "slot", disabledSlot, "character", c.Name)
				}
			}

		}

	}

	// Handle extra arm slots based on ExtraArms mutation level
	// If character lacks enough extra arms, move items back to backpack
	if c.ExtraArms < 2 {
		if c.Equipment.ExtraArm2.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraArm2)
			mudlog.Debug("Extra Arms Check", "info", "Item returned from extra arm 2 slot", "name", c.Equipment.ExtraArm2.Name(), "character", c.Name)
		}
		c.Equipment.ExtraArm2 = items.ItemDisabledSlot
	}
	if c.ExtraArms < 1 {
		if c.Equipment.ExtraArm1.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraArm1)
			mudlog.Debug("Extra Arms Check", "info", "Item returned from extra arm 1 slot", "name", c.Equipment.ExtraArm1.Name(), "character", c.Name)
		}
		c.Equipment.ExtraArm1 = items.ItemDisabledSlot
	}

	if len(recalcPermaBuffs) > 0 && recalcPermaBuffs[0] {
		c.reapplyPermabuffs()
	}

	return nil
}

func (c *Character) Species() string {
	if r := species.GetSpecies(c.SpeciesId); r != nil {
		return r.Name
	}
	return `Ghostly Spirit`
}

func (c *Character) GetAllBackpackItems() []items.Item {
	return append([]items.Item{}, c.Items...)
}

func (c *Character) GetAllWornItems() []items.Item {
	wornItems := []items.Item{}
	if c.Equipment.Weapon.ItemId > 0 {
		wornItems = append(wornItems, c.Equipment.Weapon)
	}
	if c.Equipment.Offhand.ItemId > 0 {
		wornItems = append(wornItems, c.Equipment.Offhand)
	}
	if c.Equipment.ExtraArm1.ItemId > 0 {
		wornItems = append(wornItems, c.Equipment.ExtraArm1)
	}
	if c.Equipment.ExtraArm2.ItemId > 0 {
		wornItems = append(wornItems, c.Equipment.ExtraArm2)
	}
	if c.Equipment.Head.ItemId > 0 {
		wornItems = append(wornItems, c.Equipment.Head)
	}
	if c.Equipment.Neck.ItemId > 0 {
		wornItems = append(wornItems, c.Equipment.Neck)
	}
	if c.Equipment.Body.ItemId > 0 {
		wornItems = append(wornItems, c.Equipment.Body)
	}
	if c.Equipment.Belt.ItemId > 0 {
		wornItems = append(wornItems, c.Equipment.Belt)
	}
	if c.Equipment.Gloves.ItemId > 0 {
		wornItems = append(wornItems, c.Equipment.Gloves)
	}
	if c.Equipment.Ring.ItemId > 0 {
		wornItems = append(wornItems, c.Equipment.Ring)
	}
	if c.Equipment.Legs.ItemId > 0 {
		wornItems = append(wornItems, c.Equipment.Legs)
	}
	if c.Equipment.Feet.ItemId > 0 {
		wornItems = append(wornItems, c.Equipment.Feet)
	}
	return wornItems
}

func (c *Character) GetGearValue() int {
	value := 0
	if c.Equipment.Weapon.ItemId > 0 {
		value += c.Equipment.Weapon.GetSpec().Value
	}
	if c.Equipment.Offhand.ItemId > 0 {
		value += c.Equipment.Offhand.GetSpec().Value
	}
	if c.Equipment.ExtraArm1.ItemId > 0 {
		value += c.Equipment.ExtraArm1.GetSpec().Value
	}
	if c.Equipment.ExtraArm2.ItemId > 0 {
		value += c.Equipment.ExtraArm2.GetSpec().Value
	}
	if c.Equipment.Head.ItemId > 0 {
		value += c.Equipment.Head.GetSpec().Value
	}
	if c.Equipment.Neck.ItemId > 0 {
		value += c.Equipment.Neck.GetSpec().Value
	}
	if c.Equipment.Body.ItemId > 0 {
		value += c.Equipment.Body.GetSpec().Value
	}
	if c.Equipment.Belt.ItemId > 0 {
		value += c.Equipment.Belt.GetSpec().Value
	}
	if c.Equipment.Gloves.ItemId > 0 {
		value += c.Equipment.Gloves.GetSpec().Value
	}
	if c.Equipment.Ring.ItemId > 0 {
		value += c.Equipment.Ring.GetSpec().Value
	}
	if c.Equipment.Legs.ItemId > 0 {
		value += c.Equipment.Legs.GetSpec().Value
	}
	if c.Equipment.Feet.ItemId > 0 {
		value += c.Equipment.Feet.GetSpec().Value
	}
	return value
}

func (c *Character) Wear(i items.Item) (returnItems []items.Item, newItemWorn bool, failureReason string) {

	i.Validate()

	spec := i.GetSpec()

	if spec.Type != items.Weapon && spec.Subtype != items.Wearable {
		return returnItems, false, `That item cannot be equipped.`
	}

	iHandsRequired := c.HandsRequired(i)
	if iHandsRequired > 2 {
		return returnItems, false, `That requires too many hands.`
	}

	// are botht he currently equipped weapon and this weapon claws?
	bothMartial := false
	if spec.Subtype == items.Claws && c.Equipment.Weapon.GetSpec().Subtype == items.Claws {
		bothMartial = true
	}

	canDualWield := c.CanDualWield()

	// Weapons can go in either hand.
	// Only do this if this is a 1 handed weapon
	if spec.Type == items.Weapon && iHandsRequired < 2 {

		// If they can dual wield
		if canDualWield || bothMartial {

			// If they have a weapon equippment and it is 1 handed
			if c.Equipment.Weapon.ItemId != 0 && c.HandsRequired(c.Equipment.Weapon) == 1 {
				// If nothing is in their offhand
				if c.Equipment.Offhand.ItemId == 0 {
					// Put it in the offhand.
					//returnItems = append(returnItems, c.Equipment.Offhand)
					c.Equipment.Offhand = i

					c.reapplyPermabuffs()

					return returnItems, true, ``
				}
			}

		}

	}

	// Extra arms: if main hand and offhand are both occupied, try extra arm slots
	if spec.Type == items.Weapon && iHandsRequired < 2 && c.ExtraArms > 0 {
		if c.Equipment.Weapon.ItemId > 0 && c.Equipment.Offhand.ItemId > 0 {
			if c.ExtraArms >= 1 && !c.Equipment.ExtraArm1.IsDisabled() && c.Equipment.ExtraArm1.ItemId == 0 {
				c.Equipment.ExtraArm1 = i
				c.reapplyPermabuffs()
				return returnItems, true, ``
			}
			if c.ExtraArms >= 2 && !c.Equipment.ExtraArm2.IsDisabled() && c.Equipment.ExtraArm2.ItemId == 0 {
				c.Equipment.ExtraArm2 = i
				c.reapplyPermabuffs()
				return returnItems, true, ``
			}
		}
	}

	// First handle weapon/offhand, since they are special cases
	switch spec.Type {
	case items.Weapon:
		if c.Equipment.Weapon.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't use a weapon.`
		}

		if !c.Equipment.Offhand.IsDisabled() { // Don't allow equipping on a disabled slot
			// If it's a 2 handed weapon, remove whatever is in the offhand
			if iHandsRequired == 2 || !canDualWield && c.Equipment.Offhand.GetSpec().Type == items.Weapon {
				returnItems = append(returnItems, c.Equipment.Offhand)
				c.Equipment.Offhand = items.Item{}
			}
		}

		// 2H weapons require ALL hands — clear extra arm weapons too
		if iHandsRequired == 2 {
			if c.Equipment.ExtraArm1.ItemId > 0 {
				returnItems = append(returnItems, c.Equipment.ExtraArm1)
				c.Equipment.ExtraArm1 = items.Item{}
			}
			if c.Equipment.ExtraArm2.ItemId > 0 {
				returnItems = append(returnItems, c.Equipment.ExtraArm2)
				c.Equipment.ExtraArm2 = items.Item{}
			}
		}

		if c.Equipment.Weapon.IsCursed() {
			return returnItems, false, `Your ` + c.Equipment.Weapon.DisplayName() + ` is cursed and prevents you from removing it.`
		}

		returnItems = append(returnItems, c.Equipment.Weapon)
		c.Equipment.Weapon = i
	case items.Offhand:
		if c.Equipment.Offhand.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't hold things in an offhand.`
		}

		if !c.Equipment.Weapon.IsDisabled() { // Don't allow equipping on a disabled slot
			// If they have a 2h weapon equipped, remove it
			if c.HandsRequired(c.Equipment.Weapon) == 2 {
				// If the weapon is cursed, do not allow the offhand to be equipped
				if c.Equipment.Weapon.IsCursed() {
					return returnItems, false, `Your ` + c.Equipment.Weapon.DisplayName() + ` is cursed and prevents you from removing it.`
				}
				returnItems = append(returnItems, c.Equipment.Weapon)
				c.Equipment.Weapon = items.Item{}
			}
		}
		returnItems = append(returnItems, c.Equipment.Offhand)
		c.Equipment.Offhand = i
	case items.Head:
		if c.Equipment.Head.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear things on your head.`
		}
		returnItems = append(returnItems, c.Equipment.Head)
		c.Equipment.Head = i
	case items.Neck:
		if c.Equipment.Neck.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear things on your neck.`
		}
		returnItems = append(returnItems, c.Equipment.Neck)
		c.Equipment.Neck = i
	case items.Body:
		if c.Equipment.Body.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear things on your body.`
		}
		returnItems = append(returnItems, c.Equipment.Body)
		c.Equipment.Body = i
	case items.Belt:
		if c.Equipment.Belt.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear things on your head.`
		}
		returnItems = append(returnItems, c.Equipment.Belt)
		c.Equipment.Belt = i
	case items.Gloves:
		if c.Equipment.Gloves.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear things as gloves.`
		}
		returnItems = append(returnItems, c.Equipment.Gloves)
		c.Equipment.Gloves = i
	case items.Ring:
		if c.Equipment.Ring.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear rings.`
		}
		returnItems = append(returnItems, c.Equipment.Ring)
		c.Equipment.Ring = i
	case items.Legs:
		if c.Equipment.Legs.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear things on your legs.`
		}
		returnItems = append(returnItems, c.Equipment.Legs)
		c.Equipment.Legs = i
	case items.Feet:
		if c.Equipment.Feet.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear things on your feet.`
		}
		returnItems = append(returnItems, c.Equipment.Feet)
		c.Equipment.Feet = i
	default:
		return returnItems, false, `Unrecognized object.`
	}

	c.reapplyPermabuffs(returnItems...)

	return returnItems, true, ``
}

func (c *Character) RemoveFromBody(i items.Item) bool {

	if i.Equals(c.Equipment.Weapon) {
		c.Equipment.Weapon = items.Item{}
	} else if i.Equals(c.Equipment.Offhand) {
		c.Equipment.Offhand = items.Item{}
	} else if i.Equals(c.Equipment.ExtraArm1) {
		c.Equipment.ExtraArm1 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraArm2) {
		c.Equipment.ExtraArm2 = items.Item{}
	} else if i.Equals(c.Equipment.Head) {
		c.Equipment.Head = items.Item{}
	} else if i.Equals(c.Equipment.Neck) {
		c.Equipment.Neck = items.Item{}
	} else if i.Equals(c.Equipment.Body) {
		c.Equipment.Body = items.Item{}
	} else if i.Equals(c.Equipment.Belt) {
		c.Equipment.Belt = items.Item{}
	} else if i.Equals(c.Equipment.Gloves) {
		c.Equipment.Gloves = items.Item{}
	} else if i.Equals(c.Equipment.Ring) {
		c.Equipment.Ring = items.Item{}
	} else if i.Equals(c.Equipment.Legs) {
		c.Equipment.Legs = items.Item{}
	} else if i.Equals(c.Equipment.Feet) {
		c.Equipment.Feet = items.Item{}
	} else {
		return false
	}

	c.reapplyPermabuffs(i)

	return true
}

// Used with SpawnInfo to gift spawning mobs with permabuffs
func (c *Character) SetPermaBuffs(buffIds []int) {
	c.permaBuffIds = buffIds
}

func (c *Character) reapplyPermabuffs(removedItems ...items.Item) {

	buffIdCount := map[int]int{}

	for _, buffId := range c.permaBuffIds {
		buffIdCount[buffId] = 100 // Special case permabuffs associated with certain mobs
	}

	// Apply any buffs that come from a species
	if rInfo := species.GetSpecies(c.SpeciesId); rInfo != nil {
		for _, buffId := range rInfo.BuffIds {
			buffIdCount[buffId] = 100 // Don't allow species buffs to be removed, keep this number high
		}
	}

	// Apply any buffs from pet
	if c.Pet.Exists() {
		for _, buffId := range c.Pet.GetBuffs() {
			buffIdCount[buffId] = 100 // Don't allow pet buffs to be removed, keep this number high
		}
	}

	// Track any buffs that come from an item
	// If these don't show up as still being required by an item (such as a yaml file was changed)
	// This will cause them to be removed.
	for _, b := range c.Buffs.List {
		if b.PermaBuff {
			if _, ok := buffIdCount[b.BuffId]; !ok {
				buffIdCount[b.BuffId] = 0
			}
		}
	}

	// Make a list of all item buffs provided by existing worn items
	for _, itm := range c.GetAllWornItems() {
		spec := itm.GetSpec()
		for _, buffId := range spec.WornBuffIds {
			buffIdCount[buffId] = buffIdCount[buffId] + 1
		}

	}
	// Remove any buffs that come specifically from item
	for _, removedItem := range removedItems {
		iSpec := removedItem.GetSpec()
		if len(iSpec.WornBuffIds) > 0 {
			for _, buffId := range iSpec.WornBuffIds {
				buffIdCount[buffId] = buffIdCount[buffId] - 1
			}
		}
	}

	for buffId, ct := range buffIdCount {
		if ct < 1 {
			c.RemoveBuff(buffId)
		} else {
			c.AddBuff(buffId, true)
		}
	}
}

func (c *Character) Uncurse() []items.Item {

	uncursedList := []items.Item{}

	if c.Equipment.Weapon.IsCursed() {
		c.Equipment.Weapon.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Weapon)
	}

	if c.Equipment.Offhand.IsCursed() {
		c.Equipment.Offhand.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Offhand)
	}

	if c.Equipment.Head.IsCursed() {
		c.Equipment.Head.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Head)
	}

	if c.Equipment.Neck.IsCursed() {
		c.Equipment.Neck.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Neck)
	}

	if c.Equipment.Body.IsCursed() {
		c.Equipment.Body.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Body)
	}

	if c.Equipment.Belt.IsCursed() {
		c.Equipment.Belt.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Belt)
	}

	if c.Equipment.Gloves.IsCursed() {
		c.Equipment.Gloves.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Gloves)
	}

	if c.Equipment.Ring.IsCursed() {
		c.Equipment.Ring.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Ring)
	}

	if c.Equipment.Legs.IsCursed() {
		c.Equipment.Legs.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Legs)
	}

	if c.Equipment.Feet.IsCursed() {
		c.Equipment.Feet.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Feet)
	}

	return uncursedList
}
