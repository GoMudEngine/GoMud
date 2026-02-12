package items

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/util"
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
		{string(Whipping), `A whipping weapon.`, 0, 0, 0},
		// Miscellaneous data
		{string(BlobContent), `Can store blob content in the item data.`, 0, 0, 0},
	}
}

const (
	Unknown ItemType = ""

	// Equipment
	Weapon  ItemType = "weapon"
	Offhand ItemType = "offhand"
	Head    ItemType = "head"
	Neck    ItemType = "neck"
	Body    ItemType = "body"
	Belt    ItemType = "belt"
	Gloves  ItemType = "gloves"
	Ring    ItemType = "ring"
	Legs    ItemType = "legs"
	Feet    ItemType = "feet"
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

	// Subtypes for wearables
	Wearable  ItemSubType = "wearable"
	Drinkable ItemSubType = "drinkable"
	Edible    ItemSubType = "edible"
	Usable    ItemSubType = "usable"
	Throwable ItemSubType = "throwable" // If dropped/thrown, triggers buff effects on room and is lost
	Mundane   ItemSubType = "mundane"

	// Subtypes for weapons, chooses attack messages.
	Generic     ItemSubType = "generic"
	Bludgeoning ItemSubType = "bludgeoning"
	Cleaving    ItemSubType = "cleaving"
	Stabbing    ItemSubType = "stabbing"
	Slashing    ItemSubType = "slashing"
	Shooting    ItemSubType = "shooting" // bows, crossbows, guns, etc.
	Claws       ItemSubType = "claws"
	Whipping    ItemSubType = "whipping"

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

	POVUser  = 0
	POVOther = 1
)

type Damage struct {
	Attacks     int    `yaml:"attacks,omitempty"` // How many attacks this weapon gets (usually 1)
	DiceRoll    string `yaml:"diceroll,omitempty"` // legacy: 1d6, etc.
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

// The blueprint for an item
type ItemSpec struct {
	ItemId          int
	Value           int
	Uses            int         `yaml:"uses,omitempty"`            // How many uses it starts with
	BuffIds         []int       `yaml:"buffids,omitempty"`         // What buffs it can apply (if used)
	WornBuffIds     []int       `yaml:"wornbuffids,omitempty"`     // BuffId's that are applied while worn, and expired when removed.
	DamageReduction int         `yaml:"damagereduction,omitempty"` // % of damage it reduces when it blocks attacks
	WaitRounds      int         `yaml:"waitrounds,omitempty"`      // How many extra rounds each combat requires
	Hands           WeaponHands `yaml:"hands"`                     // How many hands it takes to wield
	Name            string
	DisplayName     string `yaml:"displayname,omitempty"` // Name that is typically displayed to the user
	NameSimple      string // A simpler name for the item, for example "Golden Battleaxe" should be "Battleaxe" or "Axe" for simple
	Description     string
	QuestToken      string `yaml:"questtoken,omitempty"` // Grants this quest if given/picked up
	Type            ItemType
	Subtype         ItemSubType
	Damage          Damage
	Element         Element           `yaml:"element,omitempty"`
	StatMods        statmods.StatMods `yaml:"statmods,omitempty"`    // What stats it modifies when equipped
	BreakChance     uint8             `yaml:"breakchance,omitempty"` // Chance in 100 that the item will break when used, or when the character is hit with it equipped, or if it is in the characters inventory during an explosion, etc.
	Cursed          bool              `yaml:"cursed,omitempty"`      // Can't be removed once equipped
	KeyLockId       string            `yaml:"keylockid,omitempty"`   // Example: `778-north` - If it's a key, what lock does it open? roomid-exitname etc.
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
	if i.ItemId >= 30000 {
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

	return nil
}

func (i *ItemSpec) Filename() string {

	filename := util.ConvertForFilename(i.Name)
	return fmt.Sprintf("%d-%s.yaml", i.ItemId, filename)
}

func (i *ItemSpec) Filepath() string {
	return i.ItemFolder() + `/` + i.Filename()
}

func (i ItemSpec) GetScript() string {

	scriptPath := i.GetScriptPath()

	// Load the script into a string
	if _, err := os.Stat(scriptPath); err == nil {
		if bytes, err := os.ReadFile(scriptPath); err == nil {
			return string(bytes)
		}
	}

	return ``
}

func (i *ItemSpec) GetScriptPath() string {
	// Load any script for the room
	return strings.Replace(string(configs.GetFilePathsConfig().DataFiles)+`/items/`+i.Filepath(), `.yaml`, `.js`, 1)
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

// file self loads due to init()
func LoadDataFiles() {

	start := time.Now()

	tmpItems, err := fileloader.LoadAllFlatFiles[int, *ItemSpec](string(configs.GetFilePathsConfig().DataFiles) + `/items`)
	if err != nil {
		panic(err)
	}

	items = tmpItems

	tmpAttackMessages, err := fileloader.LoadAllFlatFiles[ItemSubType, *WeaponAttackMessageGroup](string(configs.GetFilePathsConfig().DataFiles) + `/combat-messages`)
	if err != nil {
		panic(err)
	}

	attackMessages = tmpAttackMessages

	mudlog.Info("itemspec.LoadDataFiles()", "itemLoadedCount", len(items), "attackMessageCount", len(attackMessages), "Time Taken", time.Since(start))

}
