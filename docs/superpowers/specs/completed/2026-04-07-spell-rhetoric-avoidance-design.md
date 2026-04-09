# Spell & Rhetoric Avoidance Layers — Design Spec

**Date**: 2026-04-07
**Status**: Approved

## Problem

Physical attacks have a rich defensive layer (dodge, parry, block via
best-of-all resolution) that gives defenders active counterplay and
skill progression. Spells and rhetoric attacks have no equivalent —
once the initial opposed roll succeeds, full damage lands (minus
passive gear mitigation). This makes magical and rhetorical combat
feel one-sided and denies defenders any skill growth from being
targeted.

## Solution

Add two new avoidance checks — **Spell Deflection** and **Stoic
Resolve** — that fire after a harmful spell or rhetoric attack hits
but before damage is applied. On success the damage is halved, not
fully negated, because spells and rhetoric require more investment
(mana/conviction costs, cast times, cooldowns) than a basic melee
swing.

Both layers grant the defender skill progression on every attempt,
creating a natural "learn by being attacked" path that mirrors
physical defense progression.

## Spell Deflection

### Trigger

Fires after a **damage-dealing spell** passes its initial hit roll
and before damage is applied. Does NOT fire for buffs, heals, wards,
or utility spells — only spells whose resolution path deals damage to
the target.

### Opposed Roll

| Side     | Stat        | Skill        |
|----------|-------------|--------------|
| Defender | Perception  | Spellcasting |
| Attacker | Willpower   | Spellcasting |

The roll uses the same `dice.OpposedRollStat` infrastructure as
physical defense. The attacker value is the same Willpower +
Spellcasting combination already used for the spell's initial hit
roll.

### Outcomes

| ZScore         | Result            | Damage Effect         |
|----------------|-------------------|-----------------------|
| >= 2.0 (crit)  | Full deflection   | Zero damage           |
| > 0 (win)      | Partial deflection| Damage x 0.5          |
| <= 0 (lose)    | Failed            | Full damage (no change)|
| <= -2.0 (fumble)| Failed           | Full damage (no change)|

Fumbles and normal losses are identical — there is no "worse than
full damage" outcome on the avoidance layer. The fumble threshold
exists only to prevent a crit from firing on a bad roll.

### Damage Pipeline Position

The avoidance multiplier applies to the raw damage **before**
mitigation:

```
raw damage → spell avoidance (x0.5 on success) → mitigation → variance roll
```

This means mitigation still stacks on top of avoidance, but the
reduction compounds rather than adding.

### Skill Progression

The defender receives `OnSkillUse("spellcasting")` on every spell
avoidance attempt regardless of outcome. Being targeted by hostile
magic is a learning experience — you observe casting patterns,
recognize energy signatures, and internalize defensive instincts.

The defender also receives `OnStatUse("perception")` to progress the
governing defense stat.

## Stoic Resolve

### Trigger

Fires after a **harmful rhetoric attack** (taunt, demoralize, or any
future hostile rhetoric ability) passes its initial opposed roll and
before conviction damage is applied. Does NOT fire for beneficial
rhetoric effects such as rallying shouts or buff-granting speeches
targeting allies.

### Opposed Roll

| Side     | Stat       | Skill    |
|----------|------------|----------|
| Defender | Willpower  | Rhetoric |
| Attacker | Charisma   | Rhetoric |

### Outcomes

| ZScore         | Result         | Damage Effect         |
|----------------|----------------|-----------------------|
| >= 2.0 (crit)  | Full resolve   | Zero damage           |
| > 0 (win)      | Partial resolve| Damage x 0.5          |
| <= 0 (lose)    | Failed         | Full damage (no change)|
| <= -2.0 (fumble)| Failed        | Full damage (no change)|

### Damage Pipeline Position

Same as spell deflection — avoidance multiplier applies before
conviction mitigation:

```
raw conviction damage → stoic resolve (x0.5 on success) → mitigation → variance roll
```

### Skill Progression

The defender receives `OnSkillUse("rhetoric")` on every stoic resolve
attempt regardless of outcome. Enduring verbal attacks builds
rhetorical awareness.

The defender also receives `OnStatUse("willpower")` to progress the
governing defense stat.

## Player-Facing Messages

All messages use descriptive language with no numeric values.

### Spell Deflection

**Defender perspective (success):**
> You recognize the spell's pattern and partially deflect it!

**Defender perspective (crit):**
> You read the spell perfectly and unravel it before it reaches you!

**Attacker perspective (target succeeded):**
> Your target partially deflects your spell!

**Attacker perspective (target crit):**
> Your target completely unravels your spell!

**Room perspective (success):**
> <defender> partially deflects <attacker>'s spell!

**Room perspective (crit):**
> <defender> unravels <attacker>'s spell completely!

**Failure:** No message. Damage lands as it does today.

### Stoic Resolve

**Defender perspective (success):**
> You steel yourself against the barrage of words.

**Defender perspective (crit):**
> The words wash over you harmlessly — you are unmoved.

**Attacker perspective (target succeeded):**
> Your words fail to fully penetrate <target>'s resolve!

**Attacker perspective (target crit):**
> Your words have no effect — <target> is completely unmoved!

**Room perspective (success):**
> <defender> shrugs off some of <attacker>'s verbal assault.

**Room perspective (crit):**
> <attacker>'s words bounce off <defender> without effect.

**Failure:** No message.

## Configuration

Two new config knobs in the `Balance` section of `config.yaml`:

| Key                                | Default | Description                              |
|------------------------------------|---------|------------------------------------------|
| `SpellAvoidanceDamageMultiplier`   | 0.50    | Damage multiplier on successful avoidance|
| `RhetoricAvoidanceDamageMultiplier`| 0.50    | Same for rhetoric avoidance              |

No defense floor is applied to these avoidance rolls. The attack has
already passed one opposed roll — the avoidance layer is a bonus
defensive opportunity, not a guaranteed safety net.

## Scope Exclusions

- No new skills are introduced. Both layers reuse existing skills
  (spellcasting, rhetoric).
- No changes to the physical defense system (dodge/parry/block).
- No changes to the initial spell or rhetoric hit rolls.
- No changes to gear-based mitigation (magical_mitigation,
  conviction_mitigation).
- AoE spells: avoidance is rolled per-target independently (same as
  how the initial hit roll works per-target).
- Mob AI: no changes needed. Mobs already use the same spell/rhetoric
  attack pipeline as players; avoidance applies symmetrically.

## Integration Points

### Spell Deflection

The check inserts into the spell resolution hooks in
`internal/hooks/spell_resolution.go`, specifically in the damage
application path after the hit roll succeeds and before
`combat.ApplyMitigation()` is called. The harmful-spell gate can
check whether the spell resolution path is dealing damage (i.e., the
spell has a damage component in its resolution).

### Stoic Resolve

The check inserts into the taunt resolution path in
`internal/hooks/combat_shared_helpers.go` or wherever taunt damage is
applied, after the initial opposed roll succeeds and before conviction
damage is dealt. The harmful-rhetoric gate checks that the effect is
a hostile one targeting an opponent, not a friendly buff/shout.

### Progression

Both `OnSkillUse` and `OnStatUse` calls use the existing progression
infrastructure in `internal/characters/progression.go`. No new
progression mechanics are needed.
