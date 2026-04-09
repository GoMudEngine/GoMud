# Map Threshold Fix + Identify Spell Design

**Date:** 2026-03-17
**Status:** Draft

---

## Part 1: Map Perception Threshold Fix

### Problem
`skill.map.go:31-39` uses hardcoded Perception thresholds (25/50/75) from
the old stat system where stats started at ~1. With the current 100-baseline
system, every character gets tier 4 (max zoom) immediately.

### Fix
Replace thresholds with values centered around 100:

| Tier | Old Threshold | New Threshold |
|------|--------------|---------------|
| 2    | >= 25        | >= 110        |
| 3    | >= 50        | >= 135        |
| 4    | >= 75        | >= 175        |

New characters (Perception ~100) start at tier 1. High-Perception characters
earn better map zoom through stat progression.

### Files Changed
- `internal/usercommands/skill.map.go` — lines 31-39

---

## Part 2: Inspect Command Removal + Identify Spell

### Problem
The `inspect` command uses the same broken Perception thresholds (25/50/75)
and displays hard numbers (dice rolls, mitigation percentages, stat mod
values) that leak internal balance values to players. Per project convention,
player-facing messages should use descriptive language, not raw numbers.

### Solution
1. Delete the `inspect` command entirely
2. Create an `identify` spell (Mental school) that reveals item properties
   using descriptive language
3. Rework the shared template to be fully descriptive
4. Appraise (merchant command) automatically benefits from the template
   rework

### What Gets Deleted
- `internal/usercommands/skill.inspect.go` — entire file
- `inspect` registration in `internal/usercommands/usercommands.go`
- `_datafiles/world/default/templates/help/inspect.template`
- `_datafiles/world/dogmud/templates/help/inspect.template`
- `_datafiles/world/empty/templates/help/inspect.template`
- `_datafiles/world/default/templates/descriptions/inspect.template`
  (if exists — verify before implementation)
- `_datafiles/world/empty/templates/descriptions/inspect.template`
  (if exists — verify before implementation)

### New Spell: Identify

**Spell definition** (`_datafiles/world/dogmud/spells/identify.yaml`):

```yaml
spellid: identify
name: Identify
description: >
  Reach out with your mind to sense the hidden properties
  of an item you are carrying or wearing.
type: neutral
schools: [mental]
cost: 20
waitrounds: 30
primarystat: willpower
base_folds: 3
difficulty: 0
effect_type: identify
```

This is the first `neutral` type spell in the game. The engine already
supports neutral spells — `internal/scripting/spell.go:56-59` passes
`SpellRest` (the text after the spell name) as a string argument to
the `onMagic` JS handler.

**Cast syntax:** `cast identify <item name>`

### Resolution: Go-side Effect Handler

The JS scripting API lacks the ability to:
- Search items by name (`FindInBackpack`/`FindOnBody` are Go-only)
- Render Go templates

Therefore, identify resolution is handled **Go-side** in
`internal/hooks/spell_resolution.go` as a new `identify` effect type
handler. The flow:

1. Extract item name from `SpellRest` (the args after `cast identify`)
2. Search caster's backpack (`FindInBackpack`) then equipped slots
   (`FindOnBody`) — both are valid targets
3. On item not found: send "You can't seem to identify that." to caster
4. On item found: build template data, render `descriptions/identify`
   template, send to caster
5. Send room message: "<player> concentrates on their <item>..."

**Edge cases:**
- Empty item name: "Identify what? (Usage: cast identify <item>)"
- Item not found: "You can't seem to identify that."
- Combat: spell can be cast in combat (uses fold system, can be
  interrupted like any spell — no special handling needed)
- Cooldown prevents grind: `waitrounds: 30` applies regardless of
  success/failure

**Spellcasting skill progression:** Yes — the fold-based casting system
already triggers `SkillUsed` events during the casting process. No
additional progression hooks needed.

### Minimal JS Script

`_datafiles/world/dogmud/spells/identify.js` — nearly empty since
resolution is Go-side. Only needed if we want custom `onCast` flavor
text:

```javascript
function onCast(actor, itemName) {
    actor.SendText("You focus your mind, reaching out to sense the "
        + "item's essence...");
    return true;
}
```

The `onMagic` callback is NOT needed — the Go-side `identify` effect
handler runs instead.

### Reworked Template: `descriptions/identify.template`

Replaces the old `descriptions/inspect.template`. Both the Identify
spell and the Appraise command use this template. All sections use
descriptive language — no raw numbers.

**Template data structure** (replaces old `inspectDetails`):

```go
type identifyDetails struct {
    Item     *items.Item
    ItemSpec *items.ItemSpec
}
```

No `InspectLevel` field — all sections are always shown. Binary
pass/fail: if you successfully cast the spell (or pay the merchant),
you see everything.

**Section: Basic Info** (always shown)
- Name, description, type/subtype, approximate value
- These are things you can see by looking at the item

**Section: Combat Properties**
Example output for a weapon:
```
   Striking Power:  moderate striking power
   Arcane Resonance: N/A
   Physical:    light
   Magical:     none
   Conviction:  none
```
Example output for armor:
```
   Striking Power:  N/A
   Arcane Resonance: N/A
   Physical:    heavy
   Magical:     thin
   Conviction:  none
```
- "Striking Power" = `damageQuality(damage_multiplier)`, shown for
  weapons, "N/A" otherwise
- "Arcane Resonance" = `spellDamageQuality(spell_damage_multiplier)`,
  shown for caster weapons (wand/sceptre/staff), "N/A" otherwise
- Physical/Magical/Conviction = `mitigationQuality()` applied to the
  single item's mitigation percentage (same function the `status`
  command uses for total mitigation, but here it's per-item)

**Section: Modifiers**
Example output:
```
   Bolsters your strength
   Slightly saps your dexterity
   Applies: Flame Shield - lingers briefly
```
- Stat mods: `statModDescription()` for each non-zero stat mod
- Applied buffs: buff name + descriptive duration

**Section: Magical Effects**
Example output:
```
   It's CURSED!
   Element:     Fire
   Crits Apply: Burning - lingers briefly
```
- Curse status (only shown if cursed)
- Element (only shown if present)
- Critical hit effects: buff name + descriptive duration

### New Template Helper Functions

All implemented as Go template functions in
`internal/templates/templatesfunctions.go`, following the existing
`mitigationQuality()` pattern.

**`damageQuality(mult float64) string`** — weapon damage multiplier:

| Range     | Description                  |
|-----------|------------------------------|
| < 0.3     | "negligible striking power"  |
| 0.3-0.6   | "feeble striking power"      |
| 0.6-1.0   | "light striking power"       |
| 1.0-1.5   | "moderate striking power"    |
| 1.5-2.5   | "strong striking power"      |
| 2.5-4.0   | "devastating striking power" |
| > 4.0     | "legendary striking power"   |

**`spellDamageQuality(mult float64) string`** — spell damage multiplier:

| Range     | Description                     |
|-----------|---------------------------------|
| < 0.5     | "negligible arcane resonance"   |
| 0.5-0.8   | "faint arcane resonance"        |
| 0.8-1.2   | "mild arcane resonance"         |
| 1.2-1.6   | "moderate arcane resonance"     |
| 1.6-2.5   | "strong arcane resonance"       |
| 2.5-4.0   | "intense arcane resonance"      |
| > 4.0     | "legendary arcane resonance"    |

**`statModDescription(statName string, value int) string`**:

| Value Range | Prefix             | Example                            |
|-------------|--------------------|------------------------------------|
| >= 20       | "greatly bolsters" | "greatly bolsters your strength"   |
| >= 10       | "bolsters"         | "bolsters your strength"           |
| >= 1        | "slightly bolsters"| "slightly bolsters your strength"  |
| <= -20      | "greatly saps"     | "greatly saps your strength"       |
| <= -10      | "saps"             | "saps your strength"               |
| <= -1       | "slightly saps"    | "slightly saps your strength"      |

### Impact on Appraise

`appraise.go` currently hardcodes `InspectLevel: 3` and uses
`descriptions/inspect` template. Changes needed:

1. Update struct to `identifyDetails` (drop `InspectLevel`)
2. Update template call from `descriptions/inspect` to
   `descriptions/identify`
3. No other changes — merchant still charges 20g, still requires
   merchant room

Result: appraise shows the full descriptive output (you're paying
gold for it). The merchant is a non-magical alternative to the spell.

### Help Files

**Delete:** `inspect.template` help files (all three world variants)

**Create:** `_datafiles/world/dogmud/templates/help/identify.template`

```
<ansi fg="command">identify</ansi> <ansi fg="black-bold">[item]</ansi>

  A mental spell that reveals the hidden properties of an
  item you are carrying or wearing.

  Usage: <ansi fg="command">cast identify <item name></ansi>

  Requires the Spellcasting skill. Costs conviction to cast.
  For a non-magical alternative, visit a merchant and use
  the <ansi fg="command">appraise</ansi> command.
```

---

## Out of Scope

- Stealth skill consolidation (separate design)
- Appraise cost scaling (currently flat 20g — fine for now)
- Legacy DiceRoll field cleanup on existing items
- Mob casting of identify (no use case currently)
