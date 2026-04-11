# Buff Schema Reference

## 1. Filename & Location

**Path formula:**
```
_datafiles/world/dogmud/buffs/{buffid}-{ConvertForFilename(name)}.yaml
```

- `{ConvertForFilename(name)}` — lowercase, keep a-z/0-9, drop apostrophes, all other chars → underscore.

**Optional JS script** (for trigger/expiry hooks):
```
_datafiles/world/dogmud/buffs/{buffid}-{ConvertForFilename(name)}.js
```

**Worked examples:**
- buffid: `2`, name: `"Stunned"` → `2-stunned.yaml`
- buffid: `3`, name: `"Blinded"` → `3-blinded.yaml`
- buffid: `0`, name: `"Meditating"` → `0-meditating.yaml`

**Existing buffs** (for reference IDs):
`0-meditating`, `1-illumination`, `2-stunned`, `3-blinded`, `5-minor_potion_healing`, `6-stamina_draught`, `7-conviction_draught`, `24-death_recovery`

---

## 2. Field Reference

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `buffid` | int | **yes** | Unique integer. Must match filename. |
| `name` | string | **yes** | Display name. Must match filename via `ConvertForFilename`. |
| `description` | string | **yes** | Shown to the player when they have this buff (in status). |
| `secret` | bool | no | If true, the buff is hidden from the player's status display. Default false. |
| `triggernow` | bool | no | If true, the trigger fires immediately on application. Default false. |
| `triggerrate` | string | no | How often the trigger fires. See triggerrate formats below. |
| `triggercount` | int | no | How many times the trigger fires before the buff expires. 0 = permanent until removed. |
| `statmods` | map | no | Stat modifiers applied while buff is active. |
| `flags` | list | no | Behavior flags. See valid flags below. |
| `start_user_text` | string | no | Text sent to holder when buff starts. Supports `{source}` token. |
| `start_room_text` | string | no | Text sent to room when buff starts. Supports `{source}` token. |
| `trigger_user_text` | string | no | Text sent to holder each trigger tick. |
| `trigger_room_text` | string | no | Text sent to room each trigger tick. |
| `end_user_text` | string | no | Text sent to holder when buff expires. |
| `end_room_text` | string | no | Text sent to room when buff expires. |

### triggerrate Formats

```yaml
triggerrate: "1 round"          # Each combat/game round
triggerrate: "3 rounds"
triggerrate: "5 real minutes"   # Real-world time
triggerrate: "2 irl hours"      # Alias for real hours
triggerrate: "1 game day"       # In-game time
```

### statmods Sub-fields

```yaml
statmods:
  strength: 10          # Positive or negative integer
  perception: -40       # Applied while buff is active; removed on expiry
  vitality: 5
  dexterity: -5
  willpower: 15
  charisma: -10
```

### Valid Flags

| Flag | Effect |
|------|--------|
| `no-combat` | Prevents the buffed character from engaging in combat. |
| `no-go` | Prevents the buffed character from moving between rooms. |
| `cancel-on-combat` | Buff is automatically removed when combat begins. |
| `cancel-on-action` | Buff is automatically removed when any action is taken. |
| `hidden` | Buff is not visible to other players examining the character. |
| `secret` | Buff hidden from the character's own status (same as `secret: true`). |
| `lightsource` | Character acts as a light source while this buff is active. |
| `see-nouns` | Character can perceive hidden nouns in rooms. |
| `nightvision` | Character can see in darkness while this buff is active. |
| `poison` | Character takes periodic damage (requires JS trigger to deal damage). |
| `drunk` | Applies intoxication effects. |

Multiple flags can be combined:
```yaml
flags:
  - cancel-on-combat
  - cancel-on-action
```

---

## 3. Annotated Examples

**Simple duration buff with statmod:**
```yaml
# _datafiles/world/dogmud/buffs/3-blinded.yaml
buffid: 3                      # Must match filename (3-blinded.yaml)
name: Blinded
description: Your vision is obscured, hindering your senses.
triggerrate: 1 round           # Ticks every round
triggercount: 3                # Expires after 3 ticks (3 rounds)
statmods:
  perception: -40              # Heavy perception penalty while blinded
```

**Behavior-controlling buff:**
```yaml
# _datafiles/world/dogmud/buffs/0-meditating.yaml
# buff 0 is special: natural expiry removes the player from the game
buffid: 0
name: Meditating
description: You are meditating before leaving the realm.
triggerrate: 1 round
triggercount: 5                # 5-round countdown
flags:
  - cancel-on-action           # Any action interrupts meditation
  - cancel-on-combat           # Combat interrupts meditation
```

**Utility buff (light source, no expiry):**
```yaml
# _datafiles/world/dogmud/buffs/1-illumination.yaml
buffid: 1
name: Illumination
description: A soft glow surrounds you, lighting the way.
triggercount: 0                # 0 = permanent until removed
flags:
  - lightsource
```

**Worn-item buff (referenced from item's wornbuffids):**
```yaml
buffid: 15
name: Ring of Swiftness
description: Your movements feel unusually fluid.
triggercount: 0                # Permanent while worn
statmods:
  dexterity: 8
```

---

## 4. YAML Text Fields

Buff messaging can live in YAML instead of JS. The engine sends YAML text
automatically before calling any JS hooks. Use `{source}` for the buff
holder's name.

**Flavor-only buff (no JS needed):**
```yaml
buffid: 27
name: Iron Will
description: Your mind is fortified against intrusion.
triggerrate: 1 round
triggercount: 12
statmods:
  willpower: 10
start_user_text: "Your mind hardens like iron, walling off intrusion."
end_user_text: "The iron resolve softens, leaving your thoughts exposed."
```

**Buff with trigger logic (JS handles onTrigger only):**
```yaml
buffid: 39
name: Venom
description: Poison courses through your veins.
triggerrate: 1 round
triggercount: 5
triggernow: true
flags:
  - poison
start_user_text: "Venom seeps into your blood, burning from within."
start_room_text: "{source} winces as venom takes hold."
end_user_text: "The venom finally runs its course."
# JS file still exists for onTrigger damage calculation
```

---

## 5. JS Hooks (Optional)

A `.js` file is only needed for buffs with custom trigger logic (damage
calculation, healing, buff removal, etc.). Flavor-only messaging should
use YAML text fields instead.

When a `.js` is needed:

```javascript
// Called each time the buff triggers (per triggerrate interval)
function onTrigger(actor, triggersLeft) {
    // e.g., calculate and apply poison damage
    var maxHP = actor.GetHealthMax();
    var dmg = Math.max(2, Math.floor(maxHP * 0.05));
    actor.AddHealth(-dmg);
}
```

---

## 5. Gotchas

**`{buffid}-{ConvertForFilename(name)}.yaml` — both parts required.**
Unlike spells, buff filenames MUST include both the ID and the converted name. `3.yaml` won't be found; it must be `3-blinded.yaml`.

**`triggercount: 0` means permanent.**
A buff with `triggercount: 0` never expires from tick-down. It must be explicitly removed by a spell, item, or script.

**`triggernow: true` fires the trigger on application.**
If the buff does damage per tick (e.g. poison), setting `triggernow: true` means it also fires immediately when applied. Usually desirable for debuffs.

**statmods are active only while buff is applied.**
When the buff expires, all statmods are automatically reversed. Do not duplicate stat changes in onTrigger / onExpire JS.

**Buff IDs must be globally unique.**
Scan the existing buffs folder before assigning a new ID. The engine indexes buffs by ID — a collision will cause one buff to overwrite the other at load time.

**buff 0 is reserved.**
buffid 0 (`Meditating`) has special engine behavior: when it expires naturally, it removes the player from the game gracefully. Do not use buffid 0 for any other purpose.
