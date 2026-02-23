# Spell Schema Reference

## 1. Filename & Location

**Path formula:**
```
_datafiles/world/dogmud/spells/{spellid}.yaml
_datafiles/world/dogmud/spells/{spellid}.js
```

- `{spellid}` is used **directly** as the filename — no `ConvertForFilename` conversion.
- The `.yaml` and `.js` files must share the exact same base name.

**Worked example:**
- spellid: `fire-bolt`
- YAML: `_datafiles/world/dogmud/spells/fire-bolt.yaml`
- JS:   `_datafiles/world/dogmud/spells/fire-bolt.js`

**Existing spells** (for reference IDs):
`aidskill`, `blind`, `curepoison`, `fireball`, `fire-bolt`, `heal`, `healall`, `illum`, `minor-shield`, `mm`, `sparks`, `stun`, `tame`, `throw-stone`

---

## 2. Field Reference

### SpellData Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `spellid` | string | **yes** | Must match filename exactly. |
| `name` | string | **yes** | Display name shown to players. |
| `description` | string | **yes** | Flavor text describing the spell. |
| `type` | SpellType | **yes** | See valid SpellType values below. |
| `schools` | list | **yes** | One or more school tags. See valid schools below. |
| `cost` | int | no | Conviction (mana) cost. Default: 0. |
| `healthcost` | int | no | HP cost on cast (for blood magic / life-force spells). |
| `waitrounds` | int | no | Rounds of casting time before the spell fires. |
| `difficulty` | int | no | Adjusts final success chance by this percentage (negative = harder). |
| `primarystat` | string | no | Stat used for spell rolls and progression. Usually `willpower` or `perception`. |
| `base_folds` | int | no | Base fold complexity. 0 = defaults to 4. |
| `target_defense_type` | string | no | `"physical"`, `"mental"`, or `""` (no defense roll). |
| `component_tag` | string | no | Required item component (e.g. `"stone"` requires throw-stone component). |
| `effect_type` | string | no | `"damage"`, `"heal"`, `"buff"`, `"tame"`, `"shield"`. |
| `effect_magnitude` | int | no | Base damage or heal amount for simple effects. |
| `buff_ids` | list | no | Buff IDs applied to target on success (for `effect_type: buff`). |

### Valid SpellType Values

| Value | Meaning |
|-------|---------|
| `harmsingle` | Damages or debuffs one target |
| `helpsingle` | Heals or buffs one target |
| `harmarea` | Damages all enemies in room |
| `harmmulti` | Damages multiple selected targets |
| `helparea` | Heals/buffs all allies in room |
| `helpmulti` | Heals/buffs multiple selected targets |
| `neutral` | No direct harm/help (utility, movement, etc.) |

### Valid School Values

| Value | Meaning |
|-------|---------|
| `elemental` | Fire, ice, lightning, earth spells |
| `enhancement` | Buffs, shields, stat boosts |
| `mental` | Mind control, illusion, stunning |
| `vital` | Healing, life force, death |

A spell can belong to multiple schools:
```yaml
schools:
  - elemental
  - mental
```

---

## 3. JS Script Contract

Every non-trivial spell requires a `.js` file with three required functions:

```javascript
// Called when casting begins (the cast command is issued)
// Return false to abort the cast (with a reason message already sent)
function onCast(sourceActor, targetActor) {
    // Validate target, send pre-cast messages
    // Return true to proceed, false to cancel
    return true;
}

// Called each wait round (if waitrounds > 0)
// Return false to cancel mid-cast
function onWait(sourceActor, targetActor) {
    // Send "still casting" messages
    // Return true to continue, false to cancel
    return true;
}

// Called when the spell successfully resolves
function onMagic(sourceActor, targetActor) {
    // Apply effects, send result messages
    // No return value needed
}
```

**Key JS API methods:**
```javascript
sourceActor.GetRoomId()           // Room the caster is in
sourceActor.UserId()              // User ID (0 for mobs)
sourceActor.GetCharacterName(true) // Display name
targetActor.GetHealth()           // Current HP (negative = incapacitated)
targetActor.AddHealth(amount)     // Heal/damage target
targetActor.AddBuff(buffId)       // Apply a buff

SendUserMessage(userId, text)     // Send to one player
SendRoomMessage(roomId, text, ...excludeIds)  // Send to room, excluding IDs
```

---

## 4. Annotated Example

```yaml
# _datafiles/world/dogmud/spells/aidskill.yaml
spellid: aidskill              # Filename: aidskill.yaml (no conversion)
name: Aid
description: Revives a fallen ally
type: helpsingle               # Targets one ally
schools:
  - vital                      # Healing school
cost: 0                        # No conviction cost (tied to skill use)
waitrounds: 2                  # 2-round casting time
difficulty: 0                  # Standard difficulty
primarystat: willpower         # Willpower governs rolls and progression
```

**Corresponding JS** (abbreviated):
```javascript
// aidskill.js
function onCast(sourceActor, targetActor) {
    if (targetActor.GetHealth() > 0) {
        SendUserMessage(sourceActor.UserId(), targetActor.GetCharacterName(true) + ' is not in need of aid.');
        return false;  // Abort — target isn't down
    }
    // Send pre-cast messages to source, target, room
    return true;
}

function onWait(sourceActor, targetActor) {
    if (targetActor.GetHealth() > 0) {
        SendUserMessage(sourceActor.UserId(), 'They are no longer in need of aid.');
        return false;
    }
    // Send "still working" messages
    return true;
}

function onMagic(sourceActor, targetActor) {
    let hp = targetActor.GetHealth();
    if (hp > 0) { return; }
    targetActor.AddHealth((hp * -1) + 1);  // Revive to 1 HP
    // Send success messages — NO raw numbers to player
}
```

---

## 5. Gotchas

**spellid IS the filename — no ConvertForFilename.**
Unlike mobs/items/buffs, spell filenames use the `spellid` value directly. `spellid: fire-bolt` → `fire-bolt.yaml` and `fire-bolt.js`. Do not apply underscore conversion.

**Both `.yaml` and `.js` must exist.**
If a spell YAML references a JS behavior (via non-trivial logic), the `.js` file must exist with the correct name. A missing `.js` causes the spell to silently do nothing on cast.

**`waitrounds: 0` means instant.**
The `onWait` function is never called for instant spells. Still provide it as an empty function for safety.

**`effect_magnitude` for simple spells only.**
For spells with complex logic in JS, `effect_magnitude` is ignored. The JS `onMagic` function handles all effect application. Only use `effect_magnitude` for spells that rely on the engine's built-in effect system.

**Never display raw damage/heal numbers to players.**
The JS must use `combat.GetDamageDescription()` / `combat.GetHealDescription()` or equivalent descriptive language. See CLAUDE.md: "Player-Facing Messages — No Hard Numbers".

**School tags affect progression, not just flavor.**
Players progress different skill trees based on spell schools. An `elemental` spell advances the elemental magic skill; a `vital` spell advances the vital magic skill. Assign schools accurately.
