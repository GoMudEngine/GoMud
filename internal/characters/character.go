package characters

import (
	"math"
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
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/stats"
	"github.com/GoMudEngine/GoMud/internal/util"
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


// CarryCapacity returns weight capacity in pounds (Strength × config multiplier)
func (c *Character) CarryCapacity() float64 {
	bal := configs.GetBalanceConfig()
	return float64(c.Stats.Strength.ValueAdj) * float64(bal.CarryCapacityMultiplier)
}

// GetCarriedWeight returns the total weight of all carried items in pounds
func (c *Character) GetCarriedWeight() float64 {
	// Backpack item weights
	backpackWeight := 0.0
	for _, item := range c.Items {
		backpackWeight += item.GetSpec().GetWeight()
	}
	// Apply Back slot weight reduction to backpack contents
	if c.Equipment.Back.ItemId > 0 {
		reduction := c.Equipment.Back.GetSpec().WeightReduction
		if reduction > 0 && reduction <= 1.0 {
			backpackWeight *= (1.0 - reduction)
		}
	}

	// Component bag item weights
	componentWeight := 0.0
	for _, item := range c.ComponentItems {
		componentWeight += item.GetSpec().GetWeight()
	}
	// Apply ComponentBag weight reduction
	if c.Equipment.ComponentBag.ItemId > 0 {
		reduction := c.Equipment.ComponentBag.GetSpec().WeightReduction
		if reduction > 0 && reduction <= 1.0 {
			componentWeight *= (1.0 - reduction)
		}
	}

	// Potion bandolier item weights
	potionWeight := 0.0
	for _, item := range c.PotionItems {
		potionWeight += item.GetSpec().GetWeight()
	}
	// Apply bandolier weight reduction
	if c.Equipment.Belt.ItemId > 0 {
		beltSpec := c.Equipment.Belt.GetSpec()
		if beltSpec.IsBandolier && beltSpec.WeightReduction > 0 && beltSpec.WeightReduction <= 1.0 {
			potionWeight *= (1.0 - beltSpec.WeightReduction)
		}
	}

	// Equipped item weights (all slots via GetAllItems)
	// Equipped items count at 50% weight — wearing something distributes
	// its load across your body rather than stuffing it in a bag.
	equippedWeight := 0.0
	for _, item := range c.Equipment.GetAllItems() {
		equippedWeight += item.GetSpec().GetWeight() * 0.5
	}

	return backpackWeight + componentWeight + potionWeight + equippedWeight
}

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




func (c *Character) GetRandomItem() (items.Item, bool) {
	if len(c.Items) == 0 {
		return items.Item{}, false
	}
	return c.Items[util.Rand(len(c.Items))], true
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

	// Auto-route component items to the component bag
	iSpec := i.GetSpec()
	if iSpec.IsComponent && c.Equipment.ComponentBag.ItemId > 0 {
		bagSpec := c.Equipment.ComponentBag.GetSpec()
		if bagSpec.BagCapacity > 0 && len(c.ComponentItems) < bagSpec.BagCapacity {
			c.ComponentItems = append(c.ComponentItems, i)
			return true
		}
	}

	// Auto-route potions and throwables to the bandolier
	if (iSpec.Type == items.Potion || (iSpec.Subtype == items.Drinkable && len(iSpec.BuffIds) > 0) || iSpec.Subtype == items.Throwable) && c.Equipment.Belt.ItemId > 0 {
		beltSpec := c.Equipment.Belt.GetSpec()
		if beltSpec.IsBandolier && beltSpec.BandolierCapacity > 0 && len(c.PotionItems) < beltSpec.BandolierCapacity {
			c.PotionItems = append(c.PotionItems, i)
			return true
		}
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
	for j := len(c.ComponentItems) - 1; j >= 0; j-- {
		if c.ComponentItems[j].Equals(i) {
			c.ComponentItems = append(c.ComponentItems[:j], c.ComponentItems[j+1:]...)
			return true
		}
	}
	for j := len(c.PotionItems) - 1; j >= 0; j-- {
		if c.PotionItems[j].Equals(i) {
			c.PotionItems = append(c.PotionItems[:j], c.PotionItems[j+1:]...)
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

// FindInPotions searches the bandolier for a matching potion, oldest first.
func (c *Character) FindInPotions(itemName string) (items.Item, bool) {
	if itemName == `` || len(c.PotionItems) == 0 {
		return items.Item{}, false
	}

	// Sort by CraftedRound ascending (oldest first) for consumption priority
	oldestIdx := -1
	var oldestRound uint64 = ^uint64(0)

	for idx := range c.PotionItems {
		closeMatch, fullMatch := items.FindMatchIn(itemName, c.PotionItems[idx])
		matched := fullMatch.ItemId != 0 || closeMatch.ItemId != 0
		if matched && c.PotionItems[idx].CraftedRound < oldestRound {
			oldestIdx = idx
			oldestRound = c.PotionItems[idx].CraftedRound
		}
	}

	if oldestIdx >= 0 {
		return c.PotionItems[oldestIdx], true
	}

	return items.Item{}, false
}

// UseItemFromPotions consumes a potion from the bandolier.
func (c *Character) UseItemFromPotions(i items.Item) int {
	for j := len(c.PotionItems) - 1; j >= 0; j-- {
		if c.PotionItems[j].Equals(i) {
			usesLeft := c.PotionItems[j].Uses
			if usesLeft > 0 {
				usesLeft--
			}
			if usesLeft <= 0 {
				c.PotionItems = append(c.PotionItems[:j], c.PotionItems[j+1:]...)
			} else {
				c.PotionItems[j].Uses = usesLeft
				c.PotionItems[j].LastUsedRound = util.GetRoundCount()
			}
			return usesLeft
		}
	}
	return 0
}

func (c *Character) FindInComponents(itemName string) (items.Item, bool) {
	if itemName == `` || len(c.ComponentItems) == 0 {
		return items.Item{}, false
	}

	closeMatchItem, matchItem := items.FindMatchIn(itemName, c.ComponentItems...)

	if matchItem.ItemId != 0 {
		return matchItem, true
	}

	if closeMatchItem.ItemId != 0 {
		return closeMatchItem, true
	}

	return items.Item{}, false
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
		c.Equipment.ExtraArm3,
		c.Equipment.ExtraArm4,
		c.Equipment.Head,
		c.Equipment.Neck,
		c.Equipment.Shoulders,
		c.Equipment.Body,
		c.Equipment.Back,
		c.Equipment.Belt,
		c.Equipment.Wrist1,
		c.Equipment.Wrist2,
		c.Equipment.ExtraWrist1,
		c.Equipment.ExtraWrist2,
		c.Equipment.ExtraWrist3,
		c.Equipment.ExtraWrist4,
		c.Equipment.Gloves,
		c.Equipment.Ring,
		c.Equipment.Ring2,
		c.Equipment.Legs,
		c.Equipment.Feet,
		c.Equipment.Tail,
		c.Equipment.ComponentBag)

	if fullMatch.ItemId != 0 {
		return fullMatch, true
	}

	if partialMatch.ItemId != 0 {
		return partialMatch, true
	}

	return items.Item{}, false
}

// FindItem searches backpack and equipped items as a single pool for
// disambiguation. Returns the item, a source description, and whether found.
func (c *Character) FindItem(itemName string) (items.Item, string, bool) {
	if itemName == "" {
		return items.Item{}, "", false
	}

	// Build combined pool of all items with source labels
	type candidate struct {
		item   items.Item
		source string
	}
	var pool []candidate

	for _, item := range c.Items {
		if item.ItemId > 0 {
			pool = append(pool, candidate{item, "in your backpack"})
		}
	}

	for _, item := range c.PotionItems {
		if item.ItemId > 0 {
			pool = append(pool, candidate{item, "in your bandolier"})
		}
	}

	slotItems := []struct {
		item   items.Item
		source string
	}{
		{c.Equipment.Weapon, "wielded"},
		{c.Equipment.Offhand, "offhand"},
		{c.Equipment.ExtraArm1, "extra arm"},
		{c.Equipment.ExtraArm2, "extra arm"},
		{c.Equipment.ExtraArm3, "extra arm 3"},
		{c.Equipment.ExtraArm4, "extra arm 4"},
		{c.Equipment.Head, "worn - head"},
		{c.Equipment.Neck, "worn - neck"},
		{c.Equipment.Shoulders, "worn - shoulders"},
		{c.Equipment.Body, "worn - body"},
		{c.Equipment.Back, "worn - back"},
		{c.Equipment.Belt, "worn - belt"},
		{c.Equipment.Wrist1, "worn - wrist"},
		{c.Equipment.Wrist2, "worn - wrist"},
		{c.Equipment.ExtraWrist1, "extra wrist 1"},
		{c.Equipment.ExtraWrist2, "extra wrist 2"},
		{c.Equipment.ExtraWrist3, "extra wrist 3"},
		{c.Equipment.ExtraWrist4, "extra wrist 4"},
		{c.Equipment.Gloves, "worn - gloves"},
		{c.Equipment.Ring, "worn - ring"},
		{c.Equipment.Ring2, "worn - ring"},
		{c.Equipment.Legs, "worn - legs"},
		{c.Equipment.Feet, "worn - feet"},
		{c.Equipment.Tail, "worn - tail"},
		{c.Equipment.ComponentBag, "worn - componentbag"},
	}
	for _, slot := range slotItems {
		if slot.item.ItemId > 0 {
			pool = append(pool, candidate{slot.item, slot.source})
		}
	}

	// Extract items for FindMatchIn
	poolItems := make([]items.Item, len(pool))
	for i, c := range pool {
		poolItems[i] = c.item
	}

	closeMatch, fullMatch := items.FindMatchIn(itemName, poolItems...)

	// Find source for the match
	findSource := func(match items.Item) string {
		for _, c := range pool {
			if c.item.ItemId == match.ItemId && c.item.UUID == match.UUID {
				return c.source
			}
		}
		return "in your backpack"
	}

	if fullMatch.ItemId != 0 {
		return fullMatch, findSource(fullMatch), true
	}
	if closeMatch.ItemId != 0 {
		return closeMatch, findSource(closeMatch), true
	}

	return items.Item{}, "", false
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

// HasShield returns true if the character is wielding a shield in offhand,
// or if the species has natural bash ability (earth elemental, etc.).
func (c *Character) HasShield() bool {
	// Species-based natural bash (elementals, golems, etc.)
	if sp := species.GetSpecies(c.SpeciesId); sp != nil && sp.NaturalBash {
		return true
	}
	return c.HasAnyShield()
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

// IsUnarmedStyle returns true if the character fights with fists, claws, or
// bare hands — anything that uses unarmed-combat skill. These fighters cannot
// parry (no weapon to deflect with) but get a speed bonus instead.
func (c *Character) IsUnarmedStyle() bool {
	if c.Equipment.Weapon.ItemId <= 0 {
		return true
	}
	sub := c.Equipment.Weapon.GetSpec().Subtype
	return sub == items.Fist || sub == items.Claws
}


// GetDefenseStaminaCost returns stamina cost for a defense type (Stage 7.1)

// DeductDefenseStamina deducts stamina for a defense and returns true if successful (Stage 7.1)

// GetToxicityMax returns the maximum toxicity this character can handle.
// Formula: BaseMax + Vitality / VitalityScale

// AddToxicity attempts to add toxicity. Returns false if it would exceed max.

// GetToxicityPenalties returns stat multipliers based on toxicity threshold.
// Returns (regenMult, perceptionMult, dexterityMult) where 1.0 = no penalty.

// Where 1000 = a full round

func (c *Character) StatMod(statName string) int {
	return c.Equipment.StatMod(statName) + c.Buffs.StatMod(statName) + c.Pet.StatMod(statName)
}

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

	// Floors.
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
	for _, dead := range []string{"cast", "ranged-combat", "first-aid"} {
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
	if c.Health < -10 {
		c.Health = -10
	}
	if c.Stamina < 0 {
		c.Stamina = 0
	}
	if c.Conviction < 0 {
		c.Conviction = 0
	}
}

// validateEquipmentItems calls items.Item.Validate() on every backpack and
// worn item to ensure all in-play items have a uid.
func (c *Character) validateEquipmentItems() {
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
	c.Equipment.Tail.Validate()
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

func (c *Character) GetAllBackpackItems() []items.Item {
	return append([]items.Item{}, c.Items...)
}

// GetAllCarriedItems returns items from all pools (backpack, component bag, bandolier).
// Used by scripting APIs that need to find items regardless of which pool they're in.
func (c *Character) GetAllCarriedItems() []items.Item {
	all := append([]items.Item{}, c.Items...)
	all = append(all, c.ComponentItems...)
	all = append(all, c.PotionItems...)
	return all
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

// wearWeaponOrShield handles pair-based placement for weapons and offhands.
// Returns the same tuple as Wear. Caller is responsible for calling
// reapplyPermabuffs (this helper calls it for 2H and shield cases internally
// to preserve pre-refactor semantics).
func (c *Character) wearWeaponOrShield(i items.Item, spec items.ItemSpec, iHandsRequired int, canDualWield bool) (returnItems []items.Item, newItemWorn bool, failureReason string) {
	pairs := c.GetHandPairs()
	isShield := spec.Type == items.Offhand

	if iHandsRequired >= 2 {
		freePair := FindFirstFreePair(pairs)
		if freePair == nil {
			freePair = FindCheapestPairToDisplace(pairs)
		}
		if freePair == nil {
			return returnItems, false, `You have no free pair of hands for a two-handed weapon.`
		}
		if !freePair.First.IsEmpty() && freePair.First.ItemPtr.IsCursed() {
			return returnItems, false, `Your ` + freePair.First.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`
		}
		if !freePair.Second.IsEmpty() && freePair.Second.ItemPtr.IsCursed() {
			return returnItems, false, `Your ` + freePair.Second.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`
		}
		if !freePair.First.IsEmpty() {
			returnItems = append(returnItems, *freePair.First.ItemPtr)
		}
		if !freePair.IsHalfPair() && !freePair.Second.IsEmpty() {
			returnItems = append(returnItems, *freePair.Second.ItemPtr)
		}
		*freePair.First.ItemPtr = i
		if !freePair.IsHalfPair() {
			*freePair.Second.ItemPtr = items.Item{}
		}
		c.reapplyPermabuffs()
		return returnItems, true, ``
	}

	if isShield {
		slot := c.FindFirstEmptySlot(pairs, true)
		if slot != nil {
			*slot.ItemPtr = i
			c.reapplyPermabuffs()
			return returnItems, true, ``
		}
		if pairs[0].First.Is2H(c) {
			return returnItems, false, `Your two-handed weapon leaves no room for a shield.`
		}
		if pairs[0].Second.ItemPtr.IsCursed() {
			return returnItems, false, `Your ` + pairs[0].Second.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`
		}
		returnItems = append(returnItems, *pairs[0].Second.ItemPtr)
		*pairs[0].Second.ItemPtr = i
		c.reapplyPermabuffs()
		return returnItems, true, ``
	}

	// 1H weapon
	bothMartial := spec.Subtype == items.Claws && c.Equipment.Weapon.GetSpec().Subtype == items.Claws

	slot := c.FindFirstEmptySlot(pairs, false)
	if slot != nil {
		if slot.Label == "offhand" && !canDualWield && !bothMartial {
			slot = nil
			for pi := 1; pi < len(pairs); pi++ {
				p := &pairs[pi]
				if p.First.Is2H(c) {
					continue
				}
				if p.First.IsEmpty() {
					slot = &p.First
					break
				}
				if !p.IsHalfPair() && p.Second.IsEmpty() {
					slot = &p.Second
					break
				}
			}
		}
		if slot != nil {
			*slot.ItemPtr = i
			c.reapplyPermabuffs()
			return returnItems, true, ``
		}
	}

	// No empty slots — displace Weapon slot (arm 1)
	if c.Equipment.Weapon.IsCursed() {
		return returnItems, false, `Your ` + c.Equipment.Weapon.DisplayName() + ` is cursed and prevents you from removing it.`
	}
	if pairs[0].First.Is2H(c) && !pairs[0].Second.IsEmpty() {
		returnItems = append(returnItems, *pairs[0].Second.ItemPtr)
		*pairs[0].Second.ItemPtr = items.Item{}
	}
	returnItems = append(returnItems, c.Equipment.Weapon)
	c.Equipment.Weapon = i
	c.reapplyPermabuffs()
	return returnItems, true, ``
}

// wearArmorSlot handles placement for non-weapon equipment (armor, rings, wrists,
// back, shoulders, component bag, tail). Returns the same tuple as Wear.
// Does NOT call reapplyPermabuffs — the caller handles that (to preserve the
// pre-refactor semantics where reapplyPermabuffs is called with returnItems).
func (c *Character) wearArmorSlot(i items.Item, spec items.ItemSpec) (returnItems []items.Item, newItemWorn bool, failureReason string) {
	switch spec.Type {
	case items.Head:
		if c.Equipment.Head.IsDisabled() {
			return returnItems, false, `You can't wear things on your head.`
		}
		returnItems = append(returnItems, c.Equipment.Head)
		c.Equipment.Head = i
	case items.Neck:
		if c.Equipment.Neck.IsDisabled() {
			return returnItems, false, `You can't wear things on your neck.`
		}
		returnItems = append(returnItems, c.Equipment.Neck)
		c.Equipment.Neck = i
	case items.Body:
		if c.Equipment.Body.IsDisabled() {
			return returnItems, false, `You can't wear things on your body.`
		}
		returnItems = append(returnItems, c.Equipment.Body)
		c.Equipment.Body = i
	case items.Belt:
		if c.Equipment.Belt.IsDisabled() {
			return returnItems, false, `You can't wear things on your head.`
		}
		returnItems = append(returnItems, c.Equipment.Belt)
		c.Equipment.Belt = i
	case items.Gloves:
		if c.Equipment.Gloves.IsDisabled() {
			return returnItems, false, `You can't wear things as gloves.`
		}
		returnItems = append(returnItems, c.Equipment.Gloves)
		c.Equipment.Gloves = i
	case items.Ring:
		if c.Equipment.Ring.IsDisabled() && c.Equipment.Ring2.IsDisabled() {
			return returnItems, false, `You can't wear rings.`
		}
		if !c.Equipment.Ring.IsDisabled() && c.Equipment.Ring.ItemId == 0 {
			c.Equipment.Ring = i
		} else if !c.Equipment.Ring2.IsDisabled() && c.Equipment.Ring2.ItemId == 0 {
			c.Equipment.Ring2 = i
		} else {
			returnItems = append(returnItems, c.Equipment.Ring)
			c.Equipment.Ring = i
		}
	case items.Wrist:
		if c.Equipment.Wrist1.IsDisabled() && c.Equipment.Wrist2.IsDisabled() {
			return returnItems, false, `You can't wear things on your wrists.`
		}
		if !c.Equipment.Wrist1.IsDisabled() && c.Equipment.Wrist1.ItemId == 0 {
			c.Equipment.Wrist1 = i
		} else if !c.Equipment.Wrist2.IsDisabled() && c.Equipment.Wrist2.ItemId == 0 {
			c.Equipment.Wrist2 = i
		} else if c.ExtraArms >= 1 && !c.Equipment.ExtraWrist1.IsDisabled() && c.Equipment.ExtraWrist1.ItemId == 0 {
			c.Equipment.ExtraWrist1 = i
		} else if c.ExtraArms >= 2 && !c.Equipment.ExtraWrist2.IsDisabled() && c.Equipment.ExtraWrist2.ItemId == 0 {
			c.Equipment.ExtraWrist2 = i
		} else if c.ExtraArms >= 3 && !c.Equipment.ExtraWrist3.IsDisabled() && c.Equipment.ExtraWrist3.ItemId == 0 {
			c.Equipment.ExtraWrist3 = i
		} else if c.ExtraArms >= 4 && !c.Equipment.ExtraWrist4.IsDisabled() && c.Equipment.ExtraWrist4.ItemId == 0 {
			c.Equipment.ExtraWrist4 = i
		} else {
			returnItems = append(returnItems, c.Equipment.Wrist1)
			c.Equipment.Wrist1 = i
		}
	case items.Back:
		if c.Equipment.Back.IsDisabled() {
			return returnItems, false, `You can't wear things on your back.`
		}
		returnItems = append(returnItems, c.Equipment.Back)
		c.Equipment.Back = i
	case items.Shoulders:
		if c.Equipment.Shoulders.IsDisabled() {
			return returnItems, false, `You can't wear things on your shoulders.`
		}
		returnItems = append(returnItems, c.Equipment.Shoulders)
		c.Equipment.Shoulders = i
	case items.ComponentBag:
		if c.Equipment.ComponentBag.IsDisabled() {
			return returnItems, false, `You can't equip a component bag.`
		}
		returnItems = append(returnItems, c.Equipment.ComponentBag)
		c.Equipment.ComponentBag = i
		c.SortComponentItems()
	case items.Legs:
		if c.Equipment.Legs.IsDisabled() {
			return returnItems, false, `You can't wear things on your legs.`
		}
		returnItems = append(returnItems, c.Equipment.Legs)
		c.Equipment.Legs = i
	case items.Feet:
		if c.Equipment.Feet.IsDisabled() {
			return returnItems, false, `You can't wear things on your feet.`
		}
		returnItems = append(returnItems, c.Equipment.Feet)
		c.Equipment.Feet = i
	case items.Tail:
		if c.Equipment.Tail.IsDisabled() {
			return returnItems, false, `You don't have a tail to attach that to.`
		}
		returnItems = append(returnItems, c.Equipment.Tail)
		c.Equipment.Tail = i
	default:
		return returnItems, false, `Unrecognized object.`
	}
	return returnItems, true, ``
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

	// Weapon + shield placement uses pair-based logic.
	if spec.Type == items.Weapon || spec.Type == items.Offhand {
		return c.wearWeaponOrShield(i, spec, iHandsRequired, c.CanDualWield())
	}

	// Armor + non-weapon slots use the simple switch.
	returnItems, newItemWorn, failureReason = c.wearArmorSlot(i, spec)
	if newItemWorn {
		c.reapplyPermabuffs(returnItems...)
	}
	return returnItems, newItemWorn, failureReason
}

// SortComponentItems moves is_component items from backpack into ComponentItems
// up to the equipped bag's capacity. Returns the count of items moved.
func (c *Character) SortComponentItems() int {
	if c.Equipment.ComponentBag.ItemId < 1 {
		return 0
	}
	bagSpec := c.Equipment.ComponentBag.GetSpec()
	capacity := bagSpec.BagCapacity
	if capacity <= 0 {
		return 0
	}

	moved := 0
	remaining := make([]items.Item, 0, len(c.Items))
	for _, item := range c.Items {
		if item.GetSpec().IsComponent && len(c.ComponentItems) < capacity {
			c.ComponentItems = append(c.ComponentItems, item)
			moved++
		} else {
			remaining = append(remaining, item)
		}
	}
	c.Items = remaining
	return moved
}

// SortPotionItems moves drinkable items from backpack into PotionItems
// up to the equipped bandolier's capacity. Returns the count of items moved.
func (c *Character) SortPotionItems() int {
	if c.Equipment.Belt.ItemId < 1 {
		return 0
	}
	beltSpec := c.Equipment.Belt.GetSpec()
	if !beltSpec.IsBandolier {
		return 0
	}
	capacity := beltSpec.BandolierCapacity
	if capacity <= 0 {
		return 0
	}

	moved := 0
	remaining := make([]items.Item, 0, len(c.Items))
	for _, item := range c.Items {
		sub := item.GetSpec().Subtype
		if (sub == items.Drinkable || sub == items.Throwable) && len(c.PotionItems) < capacity {
			c.PotionItems = append(c.PotionItems, item)
			moved++
		} else {
			remaining = append(remaining, item)
		}
	}
	c.Items = remaining
	return moved
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
		// If removing a bandolier, spill potions to backpack
		beltSpec := c.Equipment.Belt.GetSpec()
		if beltSpec.IsBandolier && len(c.PotionItems) > 0 {
			for _, pi := range c.PotionItems {
				c.Items = append(c.Items, pi)
			}
			c.PotionItems = nil
		}
		c.Equipment.Belt = items.Item{}
	} else if i.Equals(c.Equipment.Gloves) {
		c.Equipment.Gloves = items.Item{}
	} else if i.Equals(c.Equipment.Ring) {
		c.Equipment.Ring = items.Item{}
	} else if i.Equals(c.Equipment.Legs) {
		c.Equipment.Legs = items.Item{}
	} else if i.Equals(c.Equipment.Feet) {
		c.Equipment.Feet = items.Item{}
	} else if i.Equals(c.Equipment.ExtraArm3) {
		c.Equipment.ExtraArm3 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraArm4) {
		c.Equipment.ExtraArm4 = items.Item{}
	} else if i.Equals(c.Equipment.Shoulders) {
		c.Equipment.Shoulders = items.Item{}
	} else if i.Equals(c.Equipment.Back) {
		c.Equipment.Back = items.Item{}
	} else if i.Equals(c.Equipment.Wrist1) {
		c.Equipment.Wrist1 = items.Item{}
	} else if i.Equals(c.Equipment.Wrist2) {
		c.Equipment.Wrist2 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraWrist1) {
		c.Equipment.ExtraWrist1 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraWrist2) {
		c.Equipment.ExtraWrist2 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraWrist3) {
		c.Equipment.ExtraWrist3 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraWrist4) {
		c.Equipment.ExtraWrist4 = items.Item{}
	} else if i.Equals(c.Equipment.Ring2) {
		c.Equipment.Ring2 = items.Item{}
	} else if i.Equals(c.Equipment.Tail) {
		c.Equipment.Tail = items.Item{}
	} else if i.Equals(c.Equipment.ComponentBag) {
		// Spill component bag contents back to backpack
		for _, ci := range c.ComponentItems {
			c.Items = append(c.Items, ci)
		}
		c.ComponentItems = nil
		c.Equipment.ComponentBag = items.Item{}
	} else {
		return false
	}

	c.reapplyPermabuffs(i)

	return true
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
