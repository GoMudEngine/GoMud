package items

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/casing"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
)

type ItemType string
type ItemSubType string
type Element string
type Intensity string
type TokenName string

type WeaponHands = int

var (
	items map[int]*ItemSpec = make(map[int]*ItemSpec)
)

type ItemTypeInfo struct {
	Type        string
	Description string
	Count       int
	MinItemId   int
	MaxItemId   int
}

// Returns key=type and value=description
func ItemTypes() []ItemTypeInfo {
	return []ItemTypeInfo{
		// Equipment
		// Equipment - Weapons
		{string(Weapon), `This can be wielded as a weapon.`, 0, 10000, 19999},
		// Equipment - Armor
		{string(Offhand), `This can be worn in the offhand.`, 0, 20000, 29999},
		{string(Head), `This can be worn in the players head equipment slot.`, 0, 20000, 29999},
		{string(Neck), `This can be worn in the players neck equipment slot.`, 0, 20000, 29999},
		{string(Body), `This can be worn in the players body equipment slot.`, 0, 20000, 29999},
		{string(Belt), `This can be worn in the players belt equipment slot.`, 0, 20000, 29999},
		{string(Gloves), `This can be worn in the players gloves equipment slot.`, 0, 20000, 29999},
		{string(Ring), `This can be worn in the players ring equipment slot.`, 0, 20000, 29999},
		{string(Legs), `This can be worn in the players legs equipment slot.`, 0, 20000, 29999},
		{string(Feet), `This can be worn in the players feet equipment slot.`, 0, 20000, 29999},
		{string(Tail), `Worn on the tail.`, 0, 20000, 29999},
		{string(Wrist), `Worn on the wrist.`, 0, 20000, 29999},
		{string(Back), `Worn on the back.`, 0, 20000, 29999},
		{string(Shoulders), `Worn on the shoulders.`, 0, 20000, 29999},
		{string(ComponentBag), `A bag for crafting materials.`, 0, 30000, 39999},
		// Consumables
		{string(Potion), `This is a magic potion.`, 0, 30000, 39999},
		{string(Food), `This is food.`, 0, 30000, 39999},
		{string(Drink), `This is a drink.`, 0, 30000, 39999},
		{string(Scroll), `This is a scroll.`, 0, 0, 9999},
		{string(Grenade), `This is an explosive object.`, 0, 0, 9999},
		{string(Junk), `This is garbage.`, 0, 0, 9999},
		// Other
		{string(Readable), `This can be read.`, 0, 0, 9999},
		{string(Key), `This is a key that opens a locked container or door.`, 0, 0, 9999},
		{string(Object), `This is a catch-all generic object without pre-defined special behaviors.`, 0, 0, 9999},
		{string(Gemstone), `This is a gemstone.`, 0, 0, 9999},
		{string(Lockpicks), `This allows use of the picklock skill.`, 0, 0, 9999},
		{string(Botanical), `This is an herb.`, 0, 30000, 39999},
		{string(Ammo), `This is ranged ammo (arrows, bolts, etc.).`, 0, 0, 9999},
	}
}

// Returns key=subtype and value=description
func ItemSubtypes() []ItemTypeInfo {
	return []ItemTypeInfo{
		// Miscellaneous
		{string(Wearable), `Can be targetted with the equip/wear/wield command.`, 0, 0, 0},
		{string(Drinkable), `Can be targetted with the drink command.`, 0, 0, 0},
		{string(Edible), `Can be targetted with the eat command.`, 0, 0, 0},
		{string(Usable), `Can be targetted with the use command.`, 0, 0, 0},
		{string(Throwable), `Can be targetted with the throw command.`, 0, 0, 0},
		{string(Mundane), `No special behavior built in.`, 0, 0, 0},
		// Weapons
		{string(Generic), `Any weapon that doesn't get assigned an actual weapon subcategory.`, 0, 0, 0},
		{string(Bludgeoning), `A blunt weapon.`, 0, 0, 0},
		{string(Cleaving), `A hacking/chopping weapon.`, 0, 0, 0},
		{string(Stabbing), `A piercing weapon.`, 0, 0, 0},
		{string(Slashing), `A slicing and slashing weapon.`, 0, 0, 0},
		{string(Shooting), `A ranged weapon.`, 0, 0, 0},
		{string(Claws), `A slashing weapon worn on the hands.`, 0, 0, 0},
		{string(Fist), `A weapon that enhances unarmed strikes.`, 0, 0, 0},
		{string(Whipping), `A whipping weapon.`, 0, 0, 0},
		{string(Wand), `A caster weapon that boosts spell damage.`, 0, 0, 0},
		{string(Sceptre), `A caster weapon that boosts spell damage.`, 0, 0, 0},
		{string(Staff), `A two-handed caster weapon with high spell damage boost.`, 0, 0, 0},
		// Miscellaneous data
		{string(BlobContent), `Can store blob content in the item data.`, 0, 0, 0},
	}
}

const (
	Unknown ItemType = ""

	// Equipment
	Weapon       ItemType = "weapon"
	Offhand      ItemType = "offhand"
	Head         ItemType = "head"
	Neck         ItemType = "neck"
	Body         ItemType = "body"
	Belt         ItemType = "belt"
	Gloves       ItemType = "gloves"
	Ring         ItemType = "ring"
	Wrist        ItemType = "wrist"        // Bracelets, bracers
	Back         ItemType = "back"         // Cloaks, backpacks
	Shoulders    ItemType = "shoulders"    // Pauldrons, mantles
	ComponentBag ItemType = "componentbag" // Crafting material bags
	Legs         ItemType = "legs"
	Feet         ItemType = "feet"
	Tail         ItemType = "tail" // Tail attachments (tail mutation slot)
	// Consumables
	Potion  ItemType = "potion"
	Food    ItemType = "food"
	Drink   ItemType = "drink"
	Scroll  ItemType = "scroll"
	Grenade ItemType = "grenade" // Expected to be thrown
	Junk    ItemType = "junk"

	// Other
	Readable  ItemType = "readable"  // Something with writing to reveal when read
	Key       ItemType = "key"       // A key for a door
	Object    ItemType = "object"    // A mundane object
	Gemstone  ItemType = "gemstone"  // A gem
	Lockpicks ItemType = "lockpicks" // Used for lockpicking
	Botanical ItemType = "botanical" // A plant, herb, etc.
	Service   ItemType = "service"   // Possibly a ticket,action, or favor being purchased
	Ammo      ItemType = "ammo"      // Ranged ammo (arrows, bolts) — forward-declared for ranged combat

	// Subtypes for wearables
	Wearable  ItemSubType = "wearable"
	Drinkable ItemSubType = "drinkable"
	Edible    ItemSubType = "edible"
	Usable    ItemSubType = "usable"
	Throwable ItemSubType = "throwable" // If dropped/thrown, triggers buff effects on room and is lost
	Mundane   ItemSubType = "mundane"

	// Subtypes for weapons, chooses attack messages.
	Unarmed     ItemSubType = "unarmed"
	Generic     ItemSubType = "generic"
	Bludgeoning ItemSubType = "bludgeoning"
	Cleaving    ItemSubType = "cleaving"
	Stabbing    ItemSubType = "stabbing"
	Slashing    ItemSubType = "slashing"
	Shooting    ItemSubType = "shooting" // bows, crossbows, guns, etc.
	Claws       ItemSubType = "claws"
	Fist        ItemSubType = "fist"
	Bite        ItemSubType = "bite"
	Sting       ItemSubType = "sting"
	Gore        ItemSubType = "gore"
	Slam        ItemSubType = "slam"
	Whipping    ItemSubType = "whipping"
	Wand        ItemSubType = "wand"
	Sceptre     ItemSubType = "sceptre"
	Staff       ItemSubType = "staff"

	BlobContent ItemSubType = "blobcontent"

	OneHanded WeaponHands = 1
	TwoHanded WeaponHands = 2

	Fire        Element = "fire"
	Water       Element = "water"
	Ice         Element = "ice"
	Electricity Element = "electricity"
	Acid        Element = "acid"
	Life        Element = "life"
	Death       Element = "death"

	// Intensity of the attack
	Prepare  Intensity = "prepare"
	Wait     Intensity = "wait"
	Miss     Intensity = "miss"
	Weak     Intensity = "weak"
	Normal   Intensity = "normal"
	Heavy    Intensity = "heavy"
	Critical Intensity = "critical"
	Fumble   Intensity = "fumble"

	// Tokens
	TokenItemName     TokenName = "{itemname}"
	TokenSource       TokenName = "{source}"
	TokenSourceType   TokenName = "{sourcetype}" // will be 'user' or 'mob'
	TokenTarget       TokenName = "{target}"
	TokenTargetType   TokenName = "{targettype}" // will be 'user' or 'mob'
	TokenUsesLeft     TokenName = "{usesleft}"
	TokenDamage       TokenName = "{damage}"
	TokenEntranceName TokenName = "{entrancename}"
	TokenExitName     TokenName = "{exitname}"
	TokenDefender     TokenName = "{defender}" // Stage 9.3: defensive action messages
	TokenAttacker     TokenName = "{attacker}" // Stage 9.3: defensive action messages
	TokenWeapon       TokenName = "{weapon}"   // Stage 9.3: defensive action messages
	TokenAttack       TokenName = "{attack}"   // Generic "strike"/"blow" for unarmed, weapon name for armed
	TokenStance       TokenName = "{stance}"   // Stage 9.4: combat stance (aggressive/defensive/balanced/reckless)
	TokenPosition     TokenName = "{position}" // Stage 9.4: combat position (standing/prone/clinched/grounded)
	TokenMomentum     TokenName = "{momentum}" // Stage 9.4: combat momentum (offensive/defensive/pressured/in control)
	TokenBodyPart     TokenName = "{bodypart}" // Stage 42.4: random body part for hit location flavor

	POVUser  = 0
	POVOther = 1
)

type Damage struct {
	Attacks     int    `yaml:"attacks,omitempty"`     // How many attacks this weapon gets (usually 1)
	DiceRoll    string `yaml:"diceroll,omitempty"`    // legacy: 1d6, etc.
	CritBuffIds []int  `yaml:"critbuffids,omitempty"` // If this damage is a crit, what buffs does it apply?
	DiceCount   int    `yaml:"dicecount,omitempty"`   // how many dice to roll for this weapons damage
	SideCount   int    `yaml:"sidecount,omitempty"`   // how many sides per dice roll
	BonusDamage int    `yaml:"bonusdamage,omitempty"` // flat damage bonus, so for example 1d6+1
	BaseDamage  int    `yaml:"basedamage,omitempty"`  // distribution mode: mean damage
	Variance    int    `yaml:"variance,omitempty"`    // distribution mode: standard deviation
}

type ItemMessage string

// Attack messages
type AttackMessageOptions []ItemMessage
type AttackEffects map[Intensity]AttackMessageOptions
type AttackMessages map[ItemSubType]AttackEffects

// ItemProc is a data-driven proc an item fires from a combat/round trigger.
// Dispatch lives in internal/hooks (import direction: hooks → items).
type ItemProc struct {
	Trigger        string             `yaml:"trigger"`                   // on_hit | on_kill | on_block | on_grapple | on_spell_hit
	Chance         int                `yaml:"chance"`                    // percent per trigger event (1-100)
	CooldownRounds int                `yaml:"cooldown_rounds,omitempty"` // internal cooldown, 0 = none
	Effect         string             `yaml:"effect"`                    // lifesteal | steal_pool | aoe_stun | apply_condition
	Params         map[string]float64 `yaml:"params,omitempty"`
}

var validProcTriggers = map[string]bool{
	"on_hit": true, "on_kill": true, "on_block": true, "on_grapple": true, "on_spell_hit": true,
}

var validProcEffects = map[string]bool{
	"lifesteal": true, "steal_pool": true, "aoe_stun": true, "apply_condition": true,
}

// The blueprint for an item
type ItemSpec struct {
	ItemId      int
	Value       int
	Uses        int   `yaml:"uses,omitempty"`        // How many uses it starts with
	BuffIds     []int `yaml:"buffids,omitempty"`     // What buffs it can apply (if used)
	WornBuffIds []int `yaml:"wornbuffids,omitempty"` // BuffId's that are applied while worn, and expired when removed.
	// ── Pinnacle Stage 1: procs, reserves, bandolier, mutation drip, hunger, voice ──
	Procs                 []ItemProc `yaml:"procs,omitempty"`                   // data-driven combat procs
	ReserveHealthPct      float64    `yaml:"reserve_health_pct,omitempty"`      // 0-1 fraction of HealthMax reserved while equipped
	ReserveStaminaPct     float64    `yaml:"reserve_stamina_pct,omitempty"`     // 0-1 fraction of StaminaMax reserved while equipped
	ReserveConvictionPct  float64    `yaml:"reserve_conviction_pct,omitempty"`  // 0-1 fraction of ConvictionMax reserved while equipped
	PreservesContents     bool       `yaml:"preserves_contents,omitempty"`      // bandolier: contents never age
	AmbientPotions        bool       `yaml:"ambient_potions,omitempty"`         // bandolier: slotted potion buffs always-on at Peak
	MutationTickInterval  int        `yaml:"mutation_tick_interval,omitempty"`  // rounds between mutation rolls while worn (0 = never)
	MutationTickChance    int        `yaml:"mutation_tick_chance,omitempty"`    // percent chance per roll
	MutationRarityFloor   int        `yaml:"mutation_rarity_floor,omitempty"`   // min mutation rarity in the pool (0 = no floor)
	VoiceId               string     `yaml:"voice_id,omitempty"`                // sentient item voice file id (itemvoices/)
	HungerRounds          int        `yaml:"hunger_rounds,omitempty"`           // rounds without a kill before the item feeds on the wielder (0 = never)
	HungerDrainPct        float64    `yaml:"hunger_drain_pct,omitempty"`        // fraction of HealthMax drained per hungry round
	TauntPull             bool       `yaml:"taunt_pull,omitempty"`              // sentient chatter on_taunt also pulls the bearer's target's aggro (Aegis)
	DamageReduction       int        `yaml:"damagereduction,omitempty"`         // Legacy for MITIGATION (superseded by the *_mitigation fields), but NOT dead: still classifies offhands as shields (HasAnyShield), feeds the item-value formula, and receives defense-bonus enchants. Removing it is a redesign, not a cleanup.
	PhysicalMitigation    int        `yaml:"physical_mitigation,omitempty"`     // % physical damage reduction (Stage 34)
	MagicalMitigation     int        `yaml:"magical_mitigation,omitempty"`      // % magical damage reduction (Stage 34)
	ConvictionMitigation  int        `yaml:"conviction_mitigation,omitempty"`   // % conviction damage reduction (Stage 34)
	DamageMultiplier      float64    `yaml:"damage_multiplier,omitempty"`       // Weapon damage multiplier for new pipeline (Stage 34)
	SpellDamageMultiplier float64    `yaml:"spell_damage_multiplier,omitempty"` // Spell damage multiplier for caster weapons (wand/sceptre/staff)
	ParryRating           int        `yaml:"parryrating,omitempty"`             // Weapon parry bonus (Stage 7.1)
	BlockRating           int        `yaml:"blockrating,omitempty"`             // Shield block bonus (Stage 7.1)
	AmmoTag               string     `yaml:"ammo_tag,omitempty"`                // Ranged weapons: ammo type required (arrows/bolts/shot). Ammo items: type provided.
	MinStrength           int        `yaml:"min_strength,omitempty"`            // Minimum Strength to wield (heavy bows/arbalest)
	WaitRounds            int        `yaml:"waitrounds,omitempty"`              // How many extra rounds each combat requires
	StaminaCost           int        `yaml:"staminacost,omitempty"`             // Stamina cost per attack with this weapon
	SpeedMultiplier       float64    `yaml:"speedmultiplier,omitempty"`         // Attack speed modifier (1.0 = unarmed baseline, <1.0 slower, >1.0 faster)
	Weight                float64    `yaml:"weight,omitempty"`                  // Weight in pounds (affects encumbrance)
	GrappleModifier       float64    `yaml:"grapplemodifier,omitempty"`         // Grapple bonus/penalty (Stage 8.2)
	EscapeModifier        float64    `yaml:"escapemodifier,omitempty"`          // Armor escape modifier for Grounded position (Stage 8.7)
	// Reach is the weapon's operational reach in meters. Combat
	// consults reach in grapple positions: weapons whose reach
	// exceeds the position's effective radius are penalized (see
	// internal/combat/reach.go). Zero is a sentinel meaning "use
	// DefaultReachForSubtype lookup based on Subtype"; authors set
	// an explicit non-zero value only for outliers (a particularly
	// short dagger, an oversized hilt, etc.).
	//
	// Reach is weapon-only — arm length / species reach is
	// intentionally out of scope for chunk 4c per the design spec.
	Reach                 float64     `yaml:"reach,omitempty"`
	Hands                 WeaponHands `yaml:"hands"` // How many hands it takes to wield
	Name                  string
	DisplayName           string `yaml:"displayname,omitempty"` // Name that is typically displayed to the user
	NameSimple            string // A simpler name for the item, for example "Golden Battleaxe" should be "Battleaxe" or "Axe" for simple
	Description           string
	QuestToken            string `yaml:"questtoken,omitempty"` // Grants this quest if given/picked up
	Type                  ItemType
	Subtype               ItemSubType
	Damage                Damage
	Element               Element           `yaml:"element,omitempty"`
	StatMods              statmods.StatMods `yaml:"statmods,omitempty"`                // What stats it modifies when equipped
	BreakChance           uint8             `yaml:"breakchance,omitempty"`             // Chance in 100 that the item will break when used, or when the character is hit with it equipped, or if it is in the characters inventory during an explosion, etc.
	Cursed                bool              `yaml:"cursed,omitempty"`                  // Can't be removed once equipped
	KeyLockId             string            `yaml:"keylockid,omitempty"`               // Example: `778-north` - If it's a key, what lock does it open? roomid-exitname etc.
	ComponentTag          string            `yaml:"component_tag,omitempty"`           // Spell component tag (e.g. "stone" for throw-stone)
	IsComponent           bool              `yaml:"is_component,omitempty"`            // Auto-routes to component bag on pickup
	WeightReduction       float64           `yaml:"weight_reduction,omitempty"`        // 0.0-1.0, fraction of contents weight reduced
	BagCapacity           int               `yaml:"bag_capacity,omitempty"`            // Max items storable in component bag
	Aging                 AgingThresholds   `yaml:"aging,omitempty"`                   // Potion aging phase thresholds
	BottleAgingMultiplier float64           `yaml:"bottle_aging_multiplier,omitempty"` // Bottle aging speed (clay=3.0, glass=1.0, phial=0.5, decanter=0.25)
	Toxicity              int               `yaml:"toxicity,omitempty"`                // Toxicity cost when consumed
	IsBandolier           bool              `yaml:"is_bandolier,omitempty"`            // Belt item that holds potions
	BandolierCapacity     int               `yaml:"bandolier_capacity,omitempty"`      // Max potions storable in bandolier
	SalvageReturns        []SalvageReturn   `yaml:"salvage_returns,omitempty"`         // Custom salvage returns for non-crafted items
	RarityTier            int               `yaml:"rarity_tier,omitempty"`             // Vendor stock cap tier (50/40/30/20/10). Used by shops.EffectiveMaxStock with mob.StockMultiplier. 0 = untiered (quest items, defer-to-3.0e items).
	VendorCategories      []string          `yaml:"vendor_categories,omitempty"`       // Disciplines that buy/sell this item; mirrors shops.ValidCraftSupports minus "general"
	NotSalable            bool              `yaml:"not_salable,omitempty"`             // True for lore / flavor / legacy items excluded from vendor economy validation
	NeverDrops            bool              `yaml:"never_drops,omitempty"`             // Equipped-only: this item is skipped entirely by mob death-loot drops (boss-only gear that must never reach players). Does not affect carried Character.Items — use a separate mechanism (loot_pool / character.items) for guaranteed loot on the same mob.
	Restricted            bool              `yaml:"restricted,omitempty"`              // Contraband: bid on by the auction Official (The Crown Assessor, econ #2.5). Interest tag only — no other mechanics.

	// YAML-driven use effects — replaces JS onUse/onCommand_use
	OnUseTrainSkill  string `yaml:"on_use_train_skill,omitempty"`
	OnUseTrainAmount int    `yaml:"on_use_train_amount,omitempty"`
	OnUseUserText    string `yaml:"on_use_user_text,omitempty"`
	OnUseRoomText    string `yaml:"on_use_room_text,omitempty"`
}

// SalvageReturn defines a material recovered when salvaging a tagged item.
type SalvageReturn struct {
	ItemTag  string `yaml:"item_tag"`
	Quantity int    `yaml:"quantity"`
}

func (i Element) String() string {
	return string(i)
}

func (i ItemType) String() string {
	return string(i)
}

func (i ItemSubType) String() string {
	return string(i)
}

func (d *Damage) String() string {
	if d.BaseDamage > 0 {
		return fmt.Sprintf("~%d ±%d", d.BaseDamage, d.Variance)
	}
	if d.DiceRoll == "" {
		return "N/A"
	}
	return d.DiceRoll
}

func (d *Damage) FormatDiceRoll() string {

	d.DiceRoll = util.FormatDiceRoll(d.Attacks, d.DiceCount, d.SideCount, d.BonusDamage, d.CritBuffIds)

	return d.DiceRoll
}

func (d *Damage) InitDiceRoll(dRoll string) {
	// If diceroll is specified, it overrides whatever stats are already there
	if len(dRoll) < 1 {
		return
	}

	d.Attacks, d.DiceCount, d.SideCount, d.BonusDamage, _ = util.ParseDiceRoll(dRoll)
}

// GetAttackStaminaCost returns the stamina cost for attacking with this weapon.
// Returns a default based on weapon type if not explicitly set.
func (is *ItemSpec) GetAttackStaminaCost() int {
	if is.StaminaCost > 0 {
		return is.StaminaCost
	}

	// Default costs based on weapon type if not specified
	// Unarmed (no weapon) will be handled separately
	switch is.Type {
	case Weapon:
		// Default to medium weapon cost
		return 8
	default:
		return 0 // Non-weapons don't cost stamina
	}
}

// GetSpeedMultiplier returns the attack speed modifier for this weapon.
// 1.0 = unarmed baseline, <1.0 = slower, >1.0 = faster
// Most weapons should be <1.0 (fewer attacks than unarmed)
func (is ItemSpec) GetSpeedMultiplier() float64 {
	if is.SpeedMultiplier > 0 {
		return is.SpeedMultiplier
	}

	// Default to 1.0 (unarmed baseline)
	// Most weapons will explicitly set values <1.0
	return 1.0
}

// GetWeight returns the weight of this item in pounds.
// Returns 0 for weightless items or if not specified.
func (is ItemSpec) GetWeight() float64 {
	return is.Weight
}

func FindItem(nameOrId string) int {
	if itemId, err := strconv.Atoi(nameOrId); err == nil {
		if itm := New(itemId); itm.ItemId != 0 {
			return itm.ItemId
		}
	}

	return FindItemByName(nameOrId)
}

func FindKeyByLockId(lockId string) int {

	for _, item := range items {
		if item.Type != Key {
			continue
		}
		if item.KeyLockId == lockId {
			return item.ItemId
		}
	}

	return 0
}

func FindItemByName(name string) int {
	name = strings.ToLower(name)

	for _, item := range items {
		if strings.ToLower(item.Name) == name {
			return item.ItemId
		}
	}

	for _, item := range items {
		if strings.HasPrefix(strings.ToLower(item.Name), name) {
			return item.ItemId
		}
	}

	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), name) {
			return item.ItemId
		}
	}

	return 0
}

func GetAllItemSpecs() []ItemSpec {

	itemSpecs := []ItemSpec{}
	for _, item := range items {
		itemSpecs = append(itemSpecs, *item)
	}
	return itemSpecs
}

// GetAllItemSpecsMap returns all item specs as a map keyed by item ID.
// The returned pointers reference cached specs — callers must not mutate.
func GetAllItemSpecsMap() map[int]*ItemSpec {
	out := make(map[int]*ItemSpec, len(items))
	for id, s := range items {
		out[id] = s
	}
	return out
}

func GetAllItemNames() []string {

	itemNames := []string{}
	for _, item := range items {
		itemNames = append(itemNames, item.Name)
	}
	return itemNames
}

// Presumably to ensure the datafile hasn't messed something up.
func (i *ItemSpec) Id() int {
	return i.ItemId
}

func CanBackstab(iSubType ItemSubType) bool {
	if iSubType == Cleaving || iSubType == Stabbing || iSubType == Slashing || iSubType == Claws {
		return true
	}
	return false
}

func (i *ItemSpec) AutoCalculateValue() {

	val := 5 // base value of 5

	// Weapon based damage valuation
	if i.Damage.BaseDamage > 0 {
		val += i.Damage.BaseDamage * i.Damage.BaseDamage * 2
	} else {
		val += (i.Damage.DiceCount * i.Damage.DiceCount) * (i.Damage.SideCount * i.Damage.SideCount * 2)
		val += i.Damage.BonusDamage * 25
	}
	// Armor based damage valuation
	val += (i.DamageReduction * i.DamageReduction) * 17

	// Get the value of any buff it applies
	for _, buffId := range i.BuffIds {
		if buffSpec := buffs.GetBuffSpec(buffId); buffSpec != nil {
			val += buffSpec.GetValue()
		}
	}

	for _, statMod := range i.StatMods {
		val += statMod * 11
	}

	// Special considerations
	if i.Uses > 1 {
		val *= i.Uses
	}

	if i.Type == Lockpicks {
		val *= 2
	}

	if i.Hands > 1 {
		val = int(math.Ceil(float64(val) * 1.25))
	}

	if i.Type == Ring {
		// rings are atomatically worth more, since they are jewelry
		val *= 2
	}

	i.Value = val
}

func (i *ItemSpec) ItemFolder(baseonly ...bool) string {
	folderName := ``
	if i.ItemId >= 40000 {
		folderName = `materials-40000` // Stage 13.1: crafting material items
	} else if i.ItemId >= 30000 {
		folderName = `consumables-30000`
	} else if i.ItemId >= 20000 {

		if len(baseonly) > 0 && baseonly[0] {
			folderName = `armor-20000`
		} else {
			folderName = `armor-20000/` + string(i.Type)
		}

	} else if i.ItemId >= 10000 {
		folderName = `weapons-10000`
	} else {
		folderName = `other-0`
	}

	return folderName
}

// Presumably to ensure the datafile hasn't messed something up.
func (i *ItemSpec) Validate() error {

	if i.Type == Weapon {
		if i.Hands == 0 {
			i.Hands = 1
		}
		if i.Damage.Attacks < 1 {
			i.Damage.Attacks = 1
		}
	}

	if i.NameSimple == `` {
		i.NameSimple = i.Name
	}

	if i.DisplayName != `` {
		i.DisplayName = util.ConvertColorShortTags(i.DisplayName)
	}

	if i.Damage.BaseDamage == 0 {
		i.Damage.InitDiceRoll(i.Damage.DiceRoll)
		i.Damage.FormatDiceRoll()
	}

	if i.Value < 1 {
		i.AutoCalculateValue()
	}

	for idx, p := range i.Procs {
		if !validProcTriggers[p.Trigger] {
			return fmt.Errorf("item %d proc %d: invalid trigger %q", i.ItemId, idx, p.Trigger)
		}
		if !validProcEffects[p.Effect] {
			return fmt.Errorf("item %d proc %d: invalid effect %q", i.ItemId, idx, p.Effect)
		}
		if p.Chance < 1 || p.Chance > 100 {
			return fmt.Errorf("item %d proc %d: chance must be 1-100, got %d", i.ItemId, idx, p.Chance)
		}
	}
	for name, v := range map[string]float64{
		"reserve_health_pct": i.ReserveHealthPct, "reserve_stamina_pct": i.ReserveStaminaPct, "reserve_conviction_pct": i.ReserveConvictionPct,
		"hunger_drain_pct": i.HungerDrainPct,
	} {
		if v < 0 || v >= 1 {
			return fmt.Errorf("item %d: %s must be in [0,1), got %v", i.ItemId, name, v)
		}
	}
	if i.HungerRounds < 0 {
		return fmt.Errorf("item %d: hunger_rounds must not be negative, got %d", i.ItemId, i.HungerRounds)
	}
	if i.MutationTickInterval < 0 {
		return fmt.Errorf("item %d: mutation_tick_interval must not be negative, got %d", i.ItemId, i.MutationTickInterval)
	}
	if i.MutationTickChance < 0 || i.MutationTickChance > 100 {
		return fmt.Errorf("item %d: mutation_tick_chance must be 0-100, got %d", i.ItemId, i.MutationTickChance)
	}
	if i.MutationTickInterval > 0 && (i.MutationTickChance < 1 || i.MutationTickChance > 100) {
		return fmt.Errorf("item %d: mutation_tick_chance must be 1-100 when mutation_tick_interval is set, got %d", i.ItemId, i.MutationTickChance)
	}
	if i.MutationRarityFloor < 0 || i.MutationRarityFloor > 10 {
		return fmt.Errorf("item %d: mutation_rarity_floor must be 0-10, got %d", i.ItemId, i.MutationRarityFloor)
	}

	return nil
}

// ProcsFor returns the procs matching a trigger. Cheap; no allocation when empty.
func (i *ItemSpec) ProcsFor(trigger string) []ItemProc {
	if len(i.Procs) == 0 {
		return nil
	}
	var out []ItemProc
	for _, p := range i.Procs {
		if p.Trigger == trigger {
			out = append(out, p)
		}
	}
	return out
}

func (i *ItemSpec) Filename() string {

	filename := util.ConvertForFilename(i.Name)
	return fmt.Sprintf("%d-%s.yaml", i.ItemId, filename)
}

func (i *ItemSpec) Filepath() string {
	return i.ItemFolder() + `/` + i.Filename()
}

func GetItemSpec(itemId int) *ItemSpec {
	if itemId > 0 {
		spec, ok := items[itemId]
		if ok {
			return spec
		}
	}
	return nil
}

// FindSpecByComponentTag returns the first ItemSpec with a matching ComponentTag,
// or nil if none is found.
func FindSpecByComponentTag(tag string) *ItemSpec {
	for _, spec := range items {
		if spec.ComponentTag == tag {
			return spec
		}
	}
	return nil
}

// file self loads due to init()
func LoadDataFiles() {

	start := time.Now()

	dataPath := string(configs.GetFilePathsConfig().DataFiles)
	tmpItems, err := fileloader.LoadAllFlatFiles[int, *ItemSpec](dataPath + `/items`)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath+`/items`))
	}

	for id, spec := range tmpItems {
		if spec.Name != "" {
			casing.AssertCanonical(spec.Name, "item", fmt.Sprintf("%d", id))
		}
		if spec.DisplayName != "" {
			casing.AssertCanonical(spec.DisplayName, "item displayname", fmt.Sprintf("%d", id))
		}
	}

	items = tmpItems

	tmpAttackMessages, err := fileloader.LoadAllFlatFiles[ItemSubType, *WeaponAttackMessageGroup](dataPath + `/combat-messages`)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath+`/combat-messages`))
	}

	attackMessages = tmpAttackMessages

	tmpDefenseMessages, err := fileloader.LoadAllFlatFiles[DefenseType, *DefenseMessageGroup](dataPath + `/defense-messages`)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath+`/defense-messages`))
	}

	defenseMessages = tmpDefenseMessages

	mudlog.Info("itemspec.LoadDataFiles()", "itemLoadedCount", len(items), "attackMessageCount", len(attackMessages), "defenseMessageCount", len(defenseMessages), "Time Taken", time.Since(start))

}

// RegisterTestItemSpec is a test-only helper that registers an ItemSpec
// in the global items registry. Used for unit tests that need GetItemSpec()
// to return a spec. The spec is persisted across tests in the same run.
func RegisterTestItemSpec(spec *ItemSpec) {
	if spec.ItemId > 0 {
		items[spec.ItemId] = spec
	}
}
