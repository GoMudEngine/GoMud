# GoMud Game Items System Context

## Overview

The GoMud items system provides a comprehensive item management framework with support for equipment, consumables, weapons, and special objects. It features a dual-layer architecture with immutable item specifications and mutable item instances, supporting enchantments, durability, and complex item behaviors through type-based categorization and attribute systems.

## Architecture

The items system is built around two main components:

### Core Components

**Item Specifications (`ItemSpec`):**
- Immutable blueprint definitions for all item types
- YAML-based storage with automatic loading and validation
- Hierarchical organization by item type and subtype
- Automatic value calculation based on item properties

**Item Instances (`Item`):**
- Mutable runtime instances based on specifications
- UUID-based unique identification for each instance
- Support for enchantments, durability, and temporary modifications
- Blob storage for custom data and scripting integration

**Type System:**
- Primary types (weapon, armor, consumables, etc.) with ID ranges
- Subtypes for specialized behaviors (wearable, usable, throwable, etc.)
- Element types for magical damage and effects
- Weapon classification for combat message selection

**Attack Message System:**
- Dynamic combat message generation based on weapon subtypes
- Intensity-based message selection (miss, weak, normal, heavy, critical)
- Token replacement system for personalized combat text
- Separate messages for attacker, defender, and room observers

## Key Features

### 1. **Hierarchical Item Classification**
- Type-based organization with reserved ID ranges for different categories
- Subtype system for specialized behaviors and interactions
- Element system for magical properties and damage types
- Automatic categorization and validation

### 2. **Instance Management**
- UUID-based unique identification for every item instance
- Temporary data storage for runtime modifications
- Enchantment system with stat bonuses and curse mechanics
- Durability and usage tracking with break chance mechanics

### 3. **Dynamic Item Modification**
- Runtime enchantment system with stat modifications
- Temporary adjective system for visual effects
- Blob storage for custom content and runtime data
- Override specifications for personalized item properties

### 4. **Combat Integration**
- Weapon damage calculation with dice roll systems
- Attack message generation based on weapon type and damage intensity
- Critical hit mechanics with buff application
- Backstab compatibility based on weapon subtype

## Item Types and Categories

### Equipment Types (ID Ranges)
```go
// Weapons: 10000-19999
Weapon ItemType = "weapon"

// Armor: 20000-29999
Head    ItemType = "head"
Neck    ItemType = "neck"
Body    ItemType = "body"
Belt    ItemType = "belt"
Gloves  ItemType = "gloves"
Ring    ItemType = "ring"
Legs    ItemType = "legs"
Feet    ItemType = "feet"
Offhand ItemType = "offhand"

// Consumables: 30000-39999
Potion     ItemType = "potion"
Food       ItemType = "food"
Drink      ItemType = "drink"
Botanical  ItemType = "botanical"

// Other: 0-9999
Scroll     ItemType = "scroll"
Readable   ItemType = "readable"
Key        ItemType = "key"
Object     ItemType = "object"
Gemstone   ItemType = "gemstone"
Lockpicks  ItemType = "lockpicks"
Grenade    ItemType = "grenade"
Junk       ItemType = "junk"
```

### Item Subtypes
```go
// Behavior Subtypes
Wearable  ItemSubType = "wearable"
Drinkable ItemSubType = "drinkable"
Edible    ItemSubType = "edible"
Usable    ItemSubType = "usable"
Throwable ItemSubType = "throwable"
Mundane   ItemSubType = "mundane"

// Weapon Subtypes (for combat messages)
Generic     ItemSubType = "generic"
Bludgeoning ItemSubType = "bludgeoning"
Cleaving    ItemSubType = "cleaving"
Stabbing    ItemSubType = "stabbing"
Slashing    ItemSubType = "slashing"
Shooting    ItemSubType = "shooting"
Claws       ItemSubType = "claws"
Whipping    ItemSubType = "whipping"
```

## Item Specification Structure

### Basic Item Properties
```go
type ItemSpec struct {
    ItemId          int
    Name            string
    DisplayName     string        // Formatted display name with colors
    NameSimple      string        // Simple name for matching
    Description     string
    Value           int           // Gold value (auto-calculated if 0)
    Type            ItemType
    Subtype         ItemSubType

    // Usage Properties
    Uses            int           // Number of uses before consumption
    BuffIds         []int         // Buffs applied when used
    WornBuffIds     []int         // Buffs applied while worn
    QuestToken      string        // Quest progress granted when obtained

    // Combat Properties
    Damage              Damage        // Weapon damage specification
    DamageReduction     int           // Legacy armor field (use mitigation fields instead)
    DamageMultiplier    float64       // Weapon/spell damage scaling (0.15–2.5)
    PhysicalMitigation  int           // Physical damage reduction % (armor)
    MagicalMitigation   int           // Magical damage reduction % (enchanted gear)
    ConvictionMitigation int          // Conviction damage reduction % (willpower items)
    WaitRounds          int           // Extra combat rounds required
    Hands               WeaponHands   // 1 or 2 handed weapon
    Element             Element       // Magical element type

    // Ranged Weapon Properties (subtype: shooting)
    AmmoTag     string  // Ammo bundle tag required to reload ("arrows","bolts","shot")
    MinStrength int     // Minimum Strength to wield without penalty (0 = no gate)

    // Enhancement Properties
    StatMods        statmods.StatMods  // Stat modifications when worn
    BreakChance     uint8              // Chance to break on use (0-100)
    Cursed          bool               // Cannot be removed when equipped
    KeyLockId       string             // Lock ID this key opens
}
```

Note: `Loaded bool` lives on the **`Item` instance struct** (not ItemSpec).
`Item.Loaded = true` when a round is chambered/nocked; persists in instance
saves and is cleared on fire.

### Damage System
```go
type Damage struct {
    Attacks     int      // Number of attacks per round
    DiceCount   int      // Number of dice to roll
    SideCount   int      // Sides per die
    BonusDamage int      // Flat damage bonus
    DiceRoll    string   // Formatted dice roll (e.g., "2d6+3")
    CritBuffIds []int    // Buffs applied on critical hits
}
```

### Unified Damage & Mitigation Pipeline (Phase 34)

All damage in DOGMud flows through three channels. Each channel uses:
`raw = stat × SkillMultiplier(rank) × item_multiplier`, then
`final = raw × (1 - min(mitigation%, cap))`, then `dice.RollStat(final)`.

**Weapon `damage_multiplier` (float64):**
- Fists/unarmed: 0.30 (config `UnarmedDamageMultiplier`)
- Crude/improvised: 0.40–0.60
- Basic iron: 0.80–1.00
- Quality steel: 1.10–1.30
- Enchanted/rare: 1.40–1.80
- Legendary: 2.00–2.50

**Armor mitigation fields (integer percentages):**
- `physical_mitigation` — reduces melee/ranged/physical spell damage
- `magical_mitigation` — reduces mind-targeting spell damage
- `conviction_mitigation` — reduces taunt/rhetoric damage

Typical values by armor tier:
| Tier | Physical | Magical | Conviction |
|------|----------|---------|------------|
| Cloth/robes | 1–3% | 5–12% | 2–5% |
| Leather | 4–8% | 2–4% | 0–2% |
| Chain/scale | 8–12% | 1–3% | 0% |
| Plate | 12–18% | 0–2% | 0% |
| Shield (offhand) | 5–10% | 0–2% | 0% |
| Amulet/ring | 0–2% | 3–8% | 3–8% |

All three channels cap at 75% by default (configurable).

**Enchantment effects for the new pipeline:**
- `physical_mitigation_bonus` — adds to physical_mitigation (int)
- `magical_mitigation_bonus` — adds to magical_mitigation (int)
- `conviction_mitigation_bonus` — adds to conviction_mitigation (int)
- `damage_multiplier_bonus` — adds to damage_multiplier (int hundredths: 10 = +0.10)

## Item Instance Management

### Item Creation and Validation
```go
// Create new item instance
func New(itemId int) Item {
    itemSpec := GetItemSpec(itemId)
    newItm := Item{
        UUID:   uuid.New(UUIDItem),
        ItemId: itemId,
        Uses:   itemSpec.Uses,
    }
    newItm.Validate()
    return newItm
}

// Item validation ensures consistency
func (i *Item) Validate() {
    if i.UUID.IsNil() {
        i.UUID = uuid.New(UUIDItem)
    }
    
    iSpec := i.GetSpec()
    if i.Uses == 0 && iSpec.Uses > 0 {
        i.Uses = iSpec.Uses
    }
}
```

### Item Identification and Matching
```go
// Multiple identification methods
func (i *Item) ShorthandId() string {
    return fmt.Sprintf("!%d:%s", i.ItemId, i.UUID.String())
}

// Name matching with partial and full match support
func (i *Item) NameMatch(input string, allowContains bool) (partialMatch bool, fullMatch bool) {
    input = strings.ToLower(input)
    simpleName := strings.ToLower(i.Name())
    
    if allowContains && strings.Contains(simpleName, input) {
        return true, simpleName == input
    }
    
    if strings.HasPrefix(simpleName, input) {
        return true, simpleName == input
    }
    
    return false, false
}
```

## Enchantment and Modification System

### Dynamic Item Enhancement
```go
// Enchant item with bonuses
func (i *Item) Enchant(damageBonus int, defenseBonus int, statBonus map[string]int, cursed bool) {
    var newSpec ItemSpec
    
    if i.Spec == nil {
        specCopy := *GetItemSpec(i.ItemId)
        newSpec = specCopy
    } else {
        newSpec = *i.Spec
    }
    
    // Apply enhancements
    newSpec.Damage.BonusDamage += damageBonus
    newSpec.DamageReduction += defenseBonus
    
    for statName, statBonusAmt := range statBonus {
        newSpec.StatMods.Add(statName, statBonusAmt)
    }
    
    i.Enchantments++
    newSpec.Cursed = cursed
    newSpec.AutoCalculateValue()
    
    i.Spec = &newSpec
}

// Curse management
func (i *Item) IsCursed() bool {
    return i.GetSpec().Cursed && !i.Uncursed
}

func (i *Item) Uncurse() {
    i.Uncursed = true
}
```

### Adjective System
```go
// Visual effects through adjectives
func (i *Item) SetAdjective(adj string, addToList bool) {
    if i.Adjectives == nil {
        i.Adjectives = []string{}
    }
    
    for idx, a := range i.Adjectives {
        if a == adj {
            if !addToList {
                i.Adjectives = append(i.Adjectives[:idx], i.Adjectives[idx+1:]...)
            }
            return
        }
    }
    
    if addToList {
        i.Adjectives = append(i.Adjectives, adj)
    }
}

// Display name with adjectives
func (i *Item) DisplayName() string {
    name := i.GetSpec().Name
    
    if len(i.Adjectives) > 0 {
        suffix := " <ansi fg=\"black-bold\">(" + strings.Join(i.Adjectives, "|") + ")</ansi>"
        name += suffix
    }
    
    return name
}
```

## Combat Message System

### Attack Message Structure
```go
type WeaponAttackMessageGroup struct {
    OptionId ItemSubType
    Options  AttackTypes
}

type AttackTypes map[Intensity]AttackOptions

type AttackOptions struct {
    Together TogetherMessages  // Same room messages
    Separate SeparateMessages  // Different room messages
}

type TogetherMessages struct {
    ToAttacker MessageOptions  // Messages to attacker
    ToDefender MessageOptions  // Messages to defender
    ToRoom     MessageOptions  // Messages to room observers
}
```

### Message Selection and Token Replacement
```go
// Get attack message based on damage percentage
func GetAttackMessage(subType ItemSubType, pctDamage int) AttackOptions {
    var intensity Intensity
    if pctDamage >= 101 {
        intensity = Critical
    } else if pctDamage >= 75 {
        intensity = Heavy
    } else if pctDamage >= 30 {
        intensity = Normal
    } else if pctDamage >= 1 {
        intensity = Weak
    } else {
        intensity = Miss
    }
    
    // Get messages for weapon subtype and intensity
    if attackMsgOptions, ok := attackMessages[subType]; ok {
        if messages, ok := attackMsgOptions.Options[intensity]; ok {
            return messages
        }
    }
    
    // Fallback to generic messages
    return GetAttackMessage(Generic, pctDamage)
}

// Token replacement in messages
func (am ItemMessage) SetTokenValue(tokenName TokenName, tokenValue string) ItemMessage {
    return ItemMessage(strings.Replace(string(am), string(tokenName), tokenValue, -1))
}
```

## Durability and Usage System

### Break Mechanics
```go
// Break chance testing
func (i *Item) BreakTest(increaseChance ...int) bool {
    bc := i.GetSpec().BreakChance
    if bc < 1 {
        return false
    }
    
    randNum := uint8(util.Rand(100))
    if len(increaseChance) > 0 {
        if uint8(increaseChance[0]) >= randNum {
            randNum = 0
        } else {
            randNum -= uint8(increaseChance[0])
        }
    }
    
    return bc > randNum
}

// Usage tracking
func (i *Item) UseItem() bool {
    if i.Uses > 0 {
        i.Uses--
        return i.Uses > 0  // Return true if item still has uses
    }
    return false
}
```

## Data Storage and Persistence

### Blob Content System
```go
// Store custom data in items
func (i *Item) SetBlob(blob string) {
    compressed := util.Compress([]byte(blob))
    i.Blob = util.Encode(compressed)
}

func (i *Item) GetBlob() string {
    if len(i.Blob) == 0 {
        return ""
    }
    
    decoded := util.Decode(i.Blob)
    return string(util.Decompress(decoded))
}

// Temporary data storage
func (i *Item) SetTempData(key string, value any) {
    if i.tempDataStore == nil {
        i.tempDataStore = make(map[string]any)
    }
    
    if value == nil {
        delete(i.tempDataStore, key)
        return
    }
    i.tempDataStore[key] = value
}
```

### File Organization
```go
// Automatic file organization by item ID ranges
func (i *ItemSpec) ItemFolder(baseonly ...bool) string {
    if i.ItemId >= 30000 {
        return "consumables-30000"
    } else if i.ItemId >= 20000 {
        if len(baseonly) > 0 && baseonly[0] {
            return "armor-20000"
        } else {
            return "armor-20000/" + string(i.Type)
        }
    } else if i.ItemId >= 10000 {
        return "weapons-10000"
    } else {
        return "other-0"
    }
}
```

## Integration Patterns

### Character Equipment Integration
```go
// Stat modification when equipped
func (i *Item) StatMod(statName ...string) int {
    if i.ItemId < 1 {
        return 0
    }
    
    itemInfo := i.GetSpec()
    return itemInfo.StatMods.Get(statName...)
}

// Equipment comparison
func (i *Item) IsBetterThan(otherItm Item) bool {
    if otherItm.ItemId < 1 {
        return i.ItemId > 0
    }
    return i.GetSpec().Value > otherItm.GetSpec().Value
}
```

### Quest System Integration
```go
// Automatic quest progress when item obtained
type ItemSpec struct {
    QuestToken string  // Quest progress granted when obtained
}

// Quest integration happens through event system
// when ItemOwnership events are fired
```

## Search and Discovery

### Item Finding Functions
```go
// Multiple search methods
func FindItem(nameOrId string) int {
    if itemId, err := strconv.Atoi(nameOrId); err == nil {
        if itm := New(itemId); itm.ItemId != 0 {
            return itm.ItemId
        }
    }
    return FindItemByName(nameOrId)
}

func FindItemByName(name string) int {
    name = strings.ToLower(name)
    
    // Exact match first
    for _, item := range items {
        if strings.ToLower(item.Name) == name {
            return item.ItemId
        }
    }
    
    // Prefix match
    for _, item := range items {
        if strings.HasPrefix(strings.ToLower(item.Name), name) {
            return item.ItemId
        }
    }
    
    // Contains match
    for _, item := range items {
        if strings.Contains(strings.ToLower(item.Name), name) {
            return item.ItemId
        }
    }
    
    return 0
}
```

### Advanced Item Matching
```go
// Find items in collections with numbering support
func FindMatchIn(itemName string, items ...Item) (pMatch Item, fMatch Item) {
    // Support for !itemId:uuid format for exact identification
    if len(itemName) > 1 && itemName[0] == '!' {
        parts := strings.Split(itemName[1:], ":")
        itemIdMatch, _ := strconv.Atoi(parts[0])
        
        var itemUUIDMatch uuid.UUID
        if len(parts) > 1 {
            itemUUIDMatch, _ = uuid.FromString(parts[1])
        }
        
        for _, itm := range items {
            if !itemUUIDMatch.IsNil() && itm.UUID == itemUUIDMatch {
                return itm, itm
            }
            if itemIdMatch > 0 && itm.ItemId == itemIdMatch {
                return itm, itm
            }
        }
    }
    
    // Support for numbered items (e.g., "sword 2" for second sword)
    itemName, itemNumber := util.GetMatchNumber(itemName)
    
    // Find matches with numbering support
    // Returns partial match and full match separately
}
```

## Performance Considerations

### Memory Management
- Item specifications loaded once at startup and cached
- Item instances use minimal memory with spec references
- Temporary data cleared automatically on item destruction
- UUID generation optimized for performance

### File Loading Optimization
```go
// Batch loading of all item specifications
func LoadDataFiles() {
    start := time.Now()
    
    tmpItems, err := fileloader.LoadAllFlatFiles[int, *ItemSpec](
        string(configs.GetFilePathsConfig().DataFiles) + "/items"
    )
    if err != nil {
        panic(err)
    }
    
    items = tmpItems
    
    // Load attack messages
    tmpAttackMessages, err := fileloader.LoadAllFlatFiles[ItemSubType, *WeaponAttackMessageGroup](
        string(configs.GetFilePathsConfig().DataFiles) + "/combat-messages"
    )
    if err != nil {
        panic(err)
    }
    
    attackMessages = tmpAttackMessages
    
    mudlog.Info("itemspec.LoadDataFiles()", 
        "itemLoadedCount", len(items),
        "attackMessageCount", len(attackMessages),
        "Time Taken", time.Since(start))
}
```

## Dependencies

- `internal/buffs` - Status effect integration for item usage and worn effects
- `internal/configs` - Configuration management for file paths and settings
- `internal/statmods` - Stat modification system for equipment bonuses
- `internal/uuid` - Unique identification system for item instances
- `internal/util` - Utility functions for dice rolls, compression, and string processing
- `internal/colorpatterns` - Color pattern application for display names
- `internal/fileloader` - YAML file loading and validation system

## Usage Examples

### Creating and Modifying Items
```go
// Create new item instance
sword := items.New(12345)

// Enchant the sword
sword.Enchant(5, 0, map[string]int{"strength": 2}, false)

// Add visual effect
sword.SetAdjective("glowing", true)

// Check if cursed
if sword.IsCursed() {
    sword.Uncurse()
}

// Test for breakage
if sword.BreakTest(10) {
    // Item broke with 10% increased chance
}
```

### Item Searching and Matching
```go
// Find item by name or ID
itemId := items.FindItem("steel sword")

// Find in player inventory
inventory := []items.Item{sword, shield, potion}
partial, exact := items.FindMatchIn("sw", inventory...)

if exact.ItemId > 0 {
    // Found exact match
} else if partial.ItemId > 0 {
    // Found partial match
}
```

### Combat Integration
```go
// Get weapon damage
attacks, dCount, dSides, bonus, critBuffs := weapon.GetDiceRoll()

// Get attack messages
messages := items.GetAttackMessage(items.Slashing, 85) // 85% damage = Heavy

// Apply token replacements
message := messages.Together.ToAttacker.Get()
message = message.SetTokenValue(items.TokenDamage, "15")
message = message.SetTokenValue(items.TokenTarget, "orc")
```

This comprehensive item system provides the foundation for all equipment,
consumables, and objects in GoMud, supporting complex interactions,
modifications, and integration with all other game systems.

---

## Item Comparison Utilities (Living Economy)

Two utility functions support NPC crafting decisions in the living economy
system:

```go
// ItemPower returns a rough numeric power score for an ItemSpec by summing
// stat mods, damage values, and mitigation fields. Used to compare two
// items of the same slot type without caring about gold value.
func ItemPower(spec ItemSpec) float64

// IsUpgrade returns true if candidate is strictly better than current for
// the same slot type. Compares ItemPower scores. Used by crafter mobs to
// decide whether to craft self-gear.
func IsUpgrade(current, candidate ItemSpec) bool
```

These functions are intentionally simple (sum-based, not weighted). They
are only used internally by the NPC craft AI — players never see the raw
scores.

---

## Rarity Tiers (Stage 3.0e)

Items intended for the living-economy supply pipeline carry a
`RarityTier` field on their ItemSpec. The tier integer doubles as the
**per-vendor MaxStock cap** when scaled by the vendor's `StockMultiplier`:

```yaml
rarity_tier: 50  # caps shop stock at 50 × mob.stock_multiplier
```

Tier semantics for materials (item IDs 40000+):

| Tier | Meaning         | MaxStock @ mult=1.0 | Examples |
|------|-----------------|---------------------|----------|
| 50   | Common          | 50                  | iron ingot, raw meat, wooden plank, glass vial |
| 40   | Standard        | 40                  | steel ingot, leather strip, healer's root |
| 30   | Regional        | 30                  | lake-iron nodule, blood-moss, wild hare meat |
| 20   | Uncommon        | 20                  | gold wire, mutation catalyst, Stillwater pearl |
| 10   | Rare (reserved) | 10                  | future: legendary materials |

Untiered items (no `rarity_tier`) are quest tokens, lore items, or
anything that should not flow through the caravan/forager pipeline.
`shops.EffectiveMaxStock(itemId, mult)` returns 0 for these — callers
fall back to legacy hardcoded values from the shop YAML.

The mob pointer is intentionally not taken inside `EffectiveMaxStock`
to avoid an import cycle (mobs imports shops). Callers pass
`mob.StockMultiplier` directly as a `float64`.

## Pricing Bands (Stage 3.4)

The `value` field on ItemSpec is the **base gold price**. Final shop
prices are dynamic — `internal/shops/pricing.go` computes a scarcity
multiplier that swings 0.25x (overstocked) to 5.0x (out of stock):

```
ratio = current / restock_qty
ratio >= 3.0          →  0.25x  (PriceFloor)
ratio == 0            →  5.0x   (PriceCeiling)
otherwise             →  0.25 + 4.75 × (1 - ratio/3)²
```

Because the multiplier already encodes scarcity, base values shouldn't
double-encode rarity — they define a sensible midpoint inside which
dynamic pricing operates.

Recommended base-value bands by tier (Approach B, applied 2026-04-30):

| Tier | Band       | Notes |
|------|------------|-------|
| 50   | 1–3g       | Utility outliers allowed (sealed phial 10g, crystalline decanter 30g — bottles whose aging multiplier is the value) |
| 40   | 5–25g      | Raw forage 5g, refined commons 8g, drops/alchemy 12g, premium 25g |
| 30   | 25–75g     | Forage 25–30g, processed 35–50g, drops 50–60g |
| 20   | 80–500g    | Common drops/refined 80–100g, quest specialties 400–500g |

Effective shop sell price = `ceil(value × ScarcityMultiplier(current, restockQty))`.
Player buy-back from NPCs applies `BuyRatio` (default 0.50) on top.

## Supply Pipeline: Caravan & Foragers

Materials enter vendor stock via two NPC-driven supply systems:

**Caravan (Stages 3.0–3.4):** A wagon-as-mob with two handlers (Hob,
Bran) runs a depot loop. At each stop,
`caravan.VisitVendorsInRoom(roomId, wagon, deliveryBuckets, pickupBuckets)`
performs a bidirectional vendor stop: deliver wagon items whose bucket
matches `deliveryBuckets`, then pick up vendor surplus matching
`pickupBuckets`. Pickup is gated by `entry.Current >= entry.MaxStock/2`
(don't extract from a starving vendor). Stock changes persist
immediately via `shops.SaveShop` so a panic mid-cycle doesn't lose
in-flight deliveries.

**Foragers (Stage 3.1):** Territory-bound forager NPCs (Halix in
Fernway South, Kessa in Stillwater Marsh) gather mats from their region
and deliver to the local depot. A behavior-tree state machine drives
five states: resting → traveling → foraging → delivering → recalling.
Recall uses fold-recall, gated on `mob.Character.IsCasting()` so the
cast doesn't reset every idle tick.

**Supply buckets** (see `internal/economy/buckets.go`) group items by
source territory:

| Bucket       | Source                           |
|--------------|----------------------------------|
| `base`       | universal craft staples (iron, leather, thread) |
| `stillwater` | Stillwater Marsh forage (lake-iron, blood-moss) |
| `fernway`    | Fernway South forage (moonpetal, ironbark) |
| `thornwall`  | Thornwall regional drops |
| `overlap`    | items appearing in multiple territories |

Each shop YAML stock entry implicitly inherits its bucket from
`economy.BucketFor(itemId)`. Caravan and forager route configurations
list which buckets to deliver vs. pick up at each stop.

**`mobs.Mob.StockMultiplier`** lets an exceptionally large or small
vendor scale all MaxStock caps. Default 1.0 means MaxStock = RarityTier
exactly. Set this on big trading-post mobs (e.g., 2.0 for a major
depot) or small specialty stalls (e.g., 0.5 for a single-recipe niche
vendor).

---

## Weapon Reach (chunk 4c)

`ItemSpec.Reach` (float64, meters) encodes how much physical space a weapon
needs to operate at full effectiveness. The combat pipeline consults reach
when the attacker is grappling — weapons whose reach exceeds the position's
effective radius are penalized multiplicatively, while weapons that fit the
radius pay no penalty. A greatsword in mount swings awkwardly (the haft
catches the attacker's own body); a dagger in mount stays fully dangerous.
The attack-message vocabulary swaps to a bludgeoning set when the penalty
fires (pommel/hilt strike narration). Full design spec and formula:
`docs/superpowers/specs/2026-05-16-state-chunk-4c-position-weapon-utility-design.md`

**Consumer side:** `internal/combat/reach.go` (T2) — `PositionReachRadius`,
`ReachUtility`, `ShouldBludgeon`, `CalcReachAdjustedItemMult`.

### Default reach by subtype

| Subtype     | Reach (m) | Notes                                    |
|-------------|-----------|------------------------------------------|
| Fist        | 0.10      | Punch length                             |
| Claws       | 0.15      | Fingers extended                         |
| Bite        | 0.15      | Head-mounted, neck-length                |
| Sting       | 0.20      | Abdomen/tail-tip                         |
| Slam        | 0.30      | Bull rush, shoulder-check                |
| Stabbing    | 0.30      | Dagger / shiv family                     |
| Gore        | 0.40      | Horn-tip from skull base                 |
| Wand        | 0.40      | Foot-long focus                          |
| Whipping    | 0.50      | Hand-held whip; mob tail overrides       |
| Sceptre     | 0.60      | Larger ornamented focus                  |
| Bludgeoning | 0.80      | Mace / hammer family                     |
| Cleaving    | 0.90      | Axe family                               |
| Slashing    | 1.00      | Sword family                             |
| Shooting    | 1.00      | Bow/crossbow as club; override compacts  |
| Staff       | 1.50      | Quarterstaff equivalent in close quarters|

**`Bludgeoning` wears two hats.** It's both a weapon carry-subtype (set in
YAML on real mace/hammer items like `steel_warhammer.yaml`, with reach 0.8 m
by default) AND the destination subtype for the bludgeon narration swap in
`internal/combat/combat_helpers.go:buildAttackMessages` (bladed weapons in
grapples render with Bludgeoning vocabulary). The two uses don't conflict —
the swap targets the message-rendering subtype only; the underlying weapon's
carry-subtype is unchanged.

Natural-attack subtypes (Fist through Whipping) stay at or below the default
ground-grapple radius (0.3 m), so they pay no penalty — by design.

### Authoring guidance

1. **Leave `reach` empty** in YAML for normal items of a known subtype.
   The engine falls through to the subtype default via `ResolveReach`.
2. **Set `reach: <meters>` only for outliers** — e.g., an unusually long
   dagger (`reach: 0.5`) or a compact crossbow (`reach: 0.6`).
3. **Use meters**, not abstract units. Real-world references help balance.
4. **New subtypes:** add a `case` in `DefaultReachForSubtype` in
   `internal/items/reach.go` and update this table.
5. Arm length / species reach is intentionally out of scope (chunk 4c
   decision). Weapon reach only.

### Natural-attack subtypes: live for mob basic attacks

The natural-attack `ItemSubType`s — `bite`, `claws`, `slam`, `gore`,
and `sting` — and their combat-message files are now the standard
rendering path for non-human mob melee (Phase 1 non-human attack
messaging). When a species YAML sets `natural_attack:`, `buildWeaponSetup`
in `internal/combat/combat_helpers.go` routes unarmed mob attacks through
that subtype's message pool. Previously these subtypes were defined and
used only for reach accounting or special-case weapon items; they are now
actively selected at runtime for basic attacks on ~30 tagged species.