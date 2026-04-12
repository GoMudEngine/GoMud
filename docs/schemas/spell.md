# Spell Schema Reference

## 1. Filename & Location

**Path formula:**
```
_datafiles/world/dogmud/spells/{spellid}.yaml
_datafiles/world/dogmud/spells/{spellid}.js   (optional — only if spell has logic)
```

- `{spellid}` is used **directly** as the filename — no `ConvertForFilename` conversion.
- The `.js` file is **optional**. Flavor-only spells use YAML text fields
  instead (see Section 2b). Only create a `.js` when the spell needs
  custom logic (companion spawning, teleportation, validation, etc.).

**Worked example:**
- spellid: `fire-bolt`
- YAML: `_datafiles/world/dogmud/spells/fire-bolt.yaml`
- JS:   `_datafiles/world/dogmud/spells/fire-bolt.js` (only if logic needed)

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
| `effect_type` | string | no | `"damage"`, `"heal"`, `"buff"`, `"tame"`, `"shield"`, `"charm"`. |
| `effect_magnitude` | int | no | Base damage or heal amount for simple effects. |
| `buff_ids` | list | no | Buff IDs applied to target on success (for `effect_type: buff`). |
| `summon_mob_id` | int | no | Mob ID to summon. Non-zero = this is a summon spell. |
| `summon_base_pool` | int | no | Base stat pool before charisma/skill scaling. |
| `summon_scaling_divisor` | int | no | Charisma divisor for scaling (default 500, lower = more scaling). |
| `summon_component_id` | int | no | Item ID consumed on cast. 0 = no component needed. |
| `summon_requires_corpse` | bool | no | If true, requires and consumes a room corpse. |
| `summon_min_corpse_pool` | int | no | Minimum corpse stat pool required for raise spells. |
| `cast_user_text` | string | no | Text sent to caster on cast. Supports `{source}`, `{target}` tokens. |
| `cast_room_text` | string | no | Text sent to room on cast. Supports `{source}`, `{target}` tokens. |
| `wait_user_text` | string | no | Text sent to caster each wait round. |
| `wait_room_text` | string | no | Text sent to room each wait round. |
| `magic_user_text` | string | no | Text sent to caster on resolution. |
| `magic_room_text` | string | no | Text sent to room on resolution. |

### YAML Text Fields (Section 2b)

Flavor text can live in YAML instead of JS. The engine sends YAML text
automatically before calling any JS hooks. If a `.js` file also exists,
both run (YAML text first, then JS).

**Token substitution:**

| Token | Resolves to |
|-------|------------|
| `{source}` | Caster's ANSI-tagged display name |
| `{target}` | Target's ANSI-tagged display name |
| `{source_plain}` | Caster's plain name (for possessives) |
| `{target_plain}` | Target's plain name |

**Example — flavor-only spell (no JS needed):**
```yaml
spellid: conviction-surge
name: Conviction Surge
type: helpsingle
schools:
  - enhancement
cost: 35
waitrounds: 1
effect_type: buff
buff_ids:
  - 26
cast_user_text: You channel conviction into empowering energy.
cast_room_text: "{source} gathers conviction, a fierce glow building."
```

**Example — spell with logic (JS handles onMagic only):**
```yaml
spellid: raise-skeleton
name: Raise Skeleton
type: neutral
# ... other fields ...
cast_user_text: You reach toward the remains, dark energy gathering.
cast_room_text: "{source} reaches toward the remains, tendrils of shadow curling from outstretched fingers."
# JS file still exists for onMagic companion spawning logic
```

**Example — Raise Skeleton (corpse-consuming summon):**
```yaml
spellid: raise-skeleton
name: Raise Skeleton
type: neutral
schools:
  - manifestation
cost: 40
waitrounds: 2
summon_mob_id: 300
summon_base_pool: 60
summon_scaling_divisor: 500
summon_requires_corpse: true
summon_min_corpse_pool: 30
cast_user_text: "You reach toward the remains, dark energy gathering."
cast_room_text: "{source} reaches toward the remains, tendrils of shadow curling."
```

**Example — Conjure Earth Elemental (no corpse, no component):**
```yaml
spellid: conjure-earth
name: Conjure Earth Elemental
type: neutral
schools:
  - manifestation
cost: 55
waitrounds: 2
summon_mob_id: 311
summon_base_pool: 90
summon_scaling_divisor: 500
cast_user_text: "You slam your fist into the ground, willing stone to rise."
cast_room_text: "{source} slams a fist into the ground with a thunderous crack."
```

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

A `.js` file is only needed for spells with custom logic (validation,
companion spawning, teleportation, etc.). Flavor-only spells should use
YAML text fields instead.

When a `.js` is needed, it can define up to three functions:

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
Unlike mobs/items/buffs, spell filenames use the `spellid` value directly.
`spellid: fire-bolt` → `fire-bolt.yaml`. Do not apply underscore conversion.

**JS is optional.** Flavor-only spells use YAML text fields. Only create
a `.js` file when the spell needs custom logic (companion spawning,
validation, teleportation, etc.). If a `.js` exists, it runs after YAML
text is sent.

**`waitrounds: 0` means instant.**
The `onWait` function is never called for instant spells.

**`effect_magnitude` for simple spells only.**
For spells with complex logic in JS, `effect_magnitude` is ignored. The JS `onMagic` function handles all effect application. Only use `effect_magnitude` for spells that rely on the engine's built-in effect system.

**Never display raw damage/heal numbers to players.**
The JS must use `combat.GetDamageDescription()` / `combat.GetHealDescription()` or equivalent descriptive language. See CLAUDE.md: "Player-Facing Messages — No Hard Numbers".

**School tags affect progression, not just flavor.**
Players progress different skill trees based on spell schools. An `elemental` spell advances the elemental magic skill; a `vital` spell advances the vital magic skill. Assign schools accurately.
