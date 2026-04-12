# Item Schema Reference

## 1. Filename & Location

**Path formula:**
```
_datafiles/world/dogmud/items/{type_folder}/{subtype_folder?}/{itemid}-{ConvertForFilename(name)}.yaml
```

- `{ConvertForFilename(name)}` — lowercase, keep a-z/0-9, drop apostrophes, all other chars → underscore.

**ID ranges and folder layout:**

| Range | Folder | Notes |
|-------|--------|-------|
| 10000–19999 | `weapons-10000/` | Flat — no subfolder by slot |
| 20000–29999 | `armor-20000/{slot}/` | Subfoldered by slot name |
| 30000–39999 | `consumables-30000/` | Potions, food, etc. |
| 40000+ | `materials-40000/` or custom | Crafting materials, misc |

**Known armor slot subfolders:** `head/`, `body/`, `legs/`, `feet/`, `hands/`, `neck/`, `ring/`, `belt/`, `offhand/`

**Worked examples:**
- `"iron longsword"` (ID 10006) → `weapons-10000/10006-iron_longsword.yaml`
- `"cloth belt"` (ID 20009) → `armor-20000/belt/20009-cloth_belt.yaml`
- `"small red potion"` (ID 30001) → `consumables-30000/30001-small_red_potion.yaml`

---

## 2. Field Reference

### Core Identity Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `itemid` | int | **yes** | Unique. Must match filename. |
| `name` | string | **yes** | Full display name (e.g. `"iron longsword"`). |
| `namesimple` | string | no | Simplified name for matching (e.g. `"longsword"`). |
| `description` | string | **yes** | Shown when player examines the item. |
| `type` | ItemType | **yes** | See valid types below. |
| `subtype` | ItemSubType | **yes** | Depends on type. See valid subtypes below. |
| `value` | int | no | Base gold value. |
| `weight` | float | no | Weight in pounds. Affects encumbrance. |
| `cursed` | bool | no | Cannot be removed once equipped. Default false. |
| `questtoken` | string | no | Quest flag granted when item is picked up or given. |
| `component_tag` | string | no | Spell component tag (e.g. `"stone"` for throw-stone spell). |

### Weapon Fields (type: weapon)

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `hands` | WeaponHands | **yes** | `1` or `2`. |
| `damage` | Damage | **yes** | See Damage sub-fields below. |
| `damage_multiplier` | float | **yes** | Scaling factor for unified damage pipeline (0.15–2.5). |
| `spell_damage_multiplier` | float | no | Multiplier for spell damage when equipped (caster weapons: wand/sceptre/staff). |
| `parryrating` | int | no | Adds to parry defense. |
| `speedmultiplier` | float | no | Attack speed modifier. 1.0 = unarmed baseline. <1.0 = slower, >1.0 = faster. |
| `staminacost` | int | no | Stamina consumed per attack. |
| `waitrounds` | int | no | Extra rounds added to combat when using this weapon. |
| `grapplemodifier` | float | no | Bonus/penalty to grapple attempts. |
| `element` | string | no | Elemental damage type: `"fire"`, `"ice"`, `"lightning"`, `"poison"`, `"holy"`, `"shadow"`. |

### Damage Sub-fields

```yaml
damage:
  basedamage: 6    # Base damage value
  variance: 2      # ± variance range
```

Or shorthand (resolved at load):
```yaml
damage:
  diceroll: "2d6+3"  # Standard dice notation
```

### Armor Fields (type: belt/head/body/legs/feet/hands/neck/ring/offhand)

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `physical_mitigation` | int | no | Physical damage reduction % (plate: 12–18, leather: 4–8). |
| `magical_mitigation` | int | no | Magical damage reduction % (robes: 5–12, amulets: 3–8). |
| `conviction_mitigation` | int | no | Conviction damage reduction % (willpower items: 3–8). |
| `damagereduction` | int | no | Legacy field (prefer mitigation fields above). |
| `blockrating` | int | no | Shield block bonus (offhand only). |
| `wornbuffids` | list | no | Buff IDs applied while item is worn; removed when unequipped. |
| `statmods` | map | no | Stat modifiers while worn. See StatMods below. |

### Consumable Fields (type: potion/food/etc.)

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `uses` | int | no | Number of uses before item is consumed. |
| `buffids` | list | no | Buff IDs applied when item is used. |

### YAML-Driven Use Effects

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `on_use_train_skill` | string | no | Skill name trained when item is used. |
| `on_use_train_amount` | int | no | Amount to train (default 1). |
| `on_use_user_text` | string | no | Text sent to user on use. Supports `{source}` token. |
| `on_use_room_text` | string | no | Text sent to room on use. Supports `{source}` token. |

Example — consumable recipe page that trains a skill:
```yaml
itemid: 40042
name: herbalism recipe page
type: object
subtype: usable
uses: 1
on_use_train_skill: search
on_use_train_amount: 1
on_use_user_text: "You study the recipe page carefully. Something clicks."
```

### StatMods Sub-fields

```yaml
statmods:
  strength: 5       # Positive or negative integer
  perception: -2
  vitality: 10
```

Valid stat names: `strength`, `dexterity`, `perception`, `vitality`, `willpower`, `charisma`

### Valid Types and Subtypes

**Weapons (`type: weapon`):**
- `subtype: slashing` — swords, axes
- `subtype: blunt` — hammers, clubs
- `subtype: piercing` — daggers, spears, bows
- `subtype: ranged` — bows, crossbows
- `subtype: wand` — caster weapon, light, 1-handed (uses `spell_damage_multiplier`)
- `subtype: sceptre` — caster weapon, moderate, 1-handed (uses `spell_damage_multiplier`)
- `subtype: staff` — caster weapon, defensive, 2-handed (uses `spell_damage_multiplier`)

**Armor (type IS the slot):**
- `type: head`, `type: body`, `type: legs`, `type: feet`, `type: hands`
- `type: neck`, `type: ring`, `type: belt`
- `type: offhand` — shields; `subtype: shield`
- All armor: `subtype: wearable`

**Consumables:**
- `type: potion`, `subtype: drinkable`
- `type: food`, `subtype: edible`
- `type: scroll`, `subtype: readable`

**Keys:**
- `type: key`, `subtype: key`
- `keylockid: "778-north"` — format: `{roomid}-{direction}`

---

## 3. Annotated Examples

**Weapon:**
```yaml
# _datafiles/world/dogmud/items/weapons-10000/10006-iron_longsword.yaml
itemid: 10006
name: iron longsword
namesimple: longsword          # Players can refer to it as just "longsword"
description: A sturdy iron longsword with a leather-wrapped hilt.
type: weapon
hands: 1                       # One-handed
subtype: slashing
weight: 8.0
speedmultiplier: 0.7           # Slightly slower than unarmed baseline
staminacost: 8
grapplemodifier: 0.7           # Harder to grapple while holding
damage_multiplier: 1.0         # Standard iron weapon (0.15=fists, 2.5=legendary)
damage:
  basedamage: 6
  variance: 2                  # Hits for 4–8 damage
parryrating: 12                # Decent parry bonus
```

**Armor:**
```yaml
# _datafiles/world/dogmud/items/armor-20000/belt/20009-cloth_belt.yaml
itemid: 20009
name: cloth belt
namesimple: belt
description: Just enough to hold your pants up.
type: belt
subtype: wearable
physical_mitigation: 1         # 1% physical damage reduction
```

**Consumable:**
```yaml
# _datafiles/world/dogmud/items/consumables-30000/30001-small_red_potion.yaml
itemid: 30001
name: small red potion
namesimple: bottle
description: A small red potion... you COULD drink it...
type: potion
subtype: drinkable
uses: 1                        # Single use
buffids:
- 5                            # Applies buff ID 5 on use (e.g. healing buff)
```

---

## 4. Gotchas

**Filename must match name via `ConvertForFilename` exactly.**
`"Iron Longsword"` → `10006-iron_longsword.yaml`. Spaces → underscores, all caps → lowercase. A mismatch causes the item to either not load or panic.

**Place items in the correct ID range subfolder.**
Weapons go in `weapons-10000/` regardless of subtype. Armor goes in `armor-20000/{slot}/`. The loader derives the expected path — an item in the wrong folder will panic on load.

**`wornbuffids` vs `buffids`:**
- `buffids` — applied when the item is *used* (consumed, activated)
- `wornbuffids` — applied while the item is *equipped*; automatically removed when unequipped

**Damage raw numbers never reach players.**
Use `combat.GetDamageDescription(amount, targetMaxHP)` in combat messages. Never display `basedamage` or `variance` values directly. See CLAUDE.md: "Player-Facing Messages — No Hard Numbers".

**`statmods` stack with all other modifiers.**
An item with `strength: 20` will be a significant balance decision. Keep item mods modest; they are not soft-capped.
