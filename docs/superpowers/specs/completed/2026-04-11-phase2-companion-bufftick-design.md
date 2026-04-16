# Phase 2: Companion Spell Consolidation + Config-Driven Buff Ticks

**Date:** 2026-04-11
**Status:** Draft
**Phase:** JS Audit Phase 2 — Replace computed JS logic with YAML config
and parameterized Go functions

## Goal

Eliminate ~29 JS files (14 companion spell scripts + ~15 healing/DoT buff
scripts) by replacing their duplicated logic with YAML configuration fields
and single-path Go functions. Also closes a design gap: buff tick magnitude
now scales with caster stats when applied by spells.

## Scope

**In scope:**
- New YAML fields on SpellData for companion summoning parameters
- One Go function `ResolveCompanionSummon()` replacing 14 JS `onMagic` scripts
- New YAML fields on BuffSpec for tick configuration
- Snapshot-based tick amount computed at buff application time
- Stat scaling on tick magnitude for spell-cast buffs (not potions)
- `start_remove_buffs` field on BuffSpec for cure-type buffs
- Migration and deletion of all affected JS files

**Out of scope:**
- Charm spell (Phase 4 — unique opposed-roll mechanic, not a summon)
- Item/room script migration (Phase 3)
- Mob AI / quest scripts (Phase 4)
- Changes to spell duration scaling (already works correctly in Go)
- Changes to potion aging or toxicity systems

## Design

### Part 1: Companion Spell Consolidation

#### New SpellData Fields

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `summon_mob_id` | int | 0 | Mob to spawn. Non-zero = summon spell. |
| `summon_base_pool` | int | 0 | Base stat pool before scaling. |
| `summon_scaling_divisor` | int | 500 | Charisma divisor (200 = high reward). |
| `summon_component_id` | int | 0 | Item consumed on cast. 0 = none. |
| `summon_requires_corpse` | bool | false | Consumes a room corpse if true. |
| `summon_min_corpse_pool` | int | 0 | Minimum corpse stat pool required. |

#### Go Function: ResolveCompanionSummon

Called from spell resolution when `spellData.SummonMobId > 0`. Replaces
all 14 companion JS `onMagic` functions.

**Flow:**

1. **Companion cap check:** `GetCompanionCount() >= GetMaxCompanionCount()`
   → send error message, return.
2. **Component consumption** (if `summon_component_id > 0`):
   Iterate backpack, find item with matching ID, `TakeItem()`.
   If not found → send error, return.
3. **Corpse consumption** (if `summon_requires_corpse`):
   Search room corpses for valid target (not player corpse, not
   companion corpse, pool >= `summon_min_corpse_pool`). If spell rest
   text is non-empty, match corpse name. Remove consumed corpse.
   If no valid corpse → send error, return.
4. **Stat scaling:**
   ```
   scale = 1.0 + charisma/divisor + manifestation*0.02
   pool = round(summon_base_pool * scale)
   ```
   If corpse consumed: `pool = (pool + corpsePool) / 2`
5. **Spawn and register:**
   `SpawnMobScaled(summon_mob_id, pool)` → `CharmSet(userId, 99999)` →
   `AddCompanion(instanceId, "summoned", name)`

**Error messages** use descriptive text (no raw numbers). Validation
errors (cap full, no component, no valid corpse) are sent to the caster
only.

**Corpse targeting:** If the caster typed `cast raise-skeleton goblin`,
the spell rest (`"goblin"`) is matched against corpse names
(case-insensitive substring). If no rest text, the first valid corpse
is used.

#### Spells Migrated (14 files deleted)

| Spell | mob_id | base_pool | divisor | component | corpse | min_corpse |
|-------|--------|-----------|---------|-----------|--------|------------|
| raise-skeleton | 300 | 60 | 500 | 0 | yes | 30 |
| raise-zombie | 301 | 80 | 500 | 0 | yes | 60 |
| raise-wraith | 302 | 70 | 500 | 0 | yes | 120 |
| raise-spectre | 303 | 90 | 500 | 0 | yes | 200 |
| raise-vampire | 304 | 100 | 500 | 0 | yes | 300 |
| raise-golem | 305 | 120 | 500 | 0 | yes | 500 |
| conjure-water | 310 | 80 | 500 | 0 | no | 0 |
| conjure-earth | 311 | 90 | 500 | 0 | no | 0 |
| conjure-air | 312 | 70 | 500 | 0 | no | 0 |
| conjure-fire | 313 | 85 | 500 | 0 | no | 0 |
| conjure-magma | 314 | 130 | 500 | 0 | no | 0 |
| summon-hive-swarm | 111 | 18 | 200 | 40011 | no | 0 |
| summon-steppe-spirit | 243 | 120 | 200 | 40031 | no | 0 |
Note: chrysalis-construct is **deleted, not migrated**. It's a weaker
version of raise-golem that no player has discovered on prod. Remove
the spell YAML, JS, mob definition (110), and component item (40010)
if nothing else references them. This eliminates the `summon_unique_key`
/ `GetMiscCharacterData` special case entirely — all remaining summon
spells use the standard companion system.

---

### Part 2: Config-Driven Buff Ticks

#### New BuffSpec Fields

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `tick_pool` | string | "" | `"health"`, `"stamina"`, `"conviction"`. Empty = no auto-tick. |
| `tick_percent` | float | 0 | Base % of max pool. Positive = heal, negative = damage. |
| `tick_variance` | float | 0 | Random variance added to percent (for DoTs). |
| `tick_min` | int | 1 | Minimum absolute tick amount. |
| `start_remove_buffs` | []int | nil | Buff IDs to remove when this buff starts. |

#### New Buff Instance Field

`TickAmount int` on the `Buff` struct — snapshot computed at application
time, applied each trigger tick.

#### Snapshot Computation

**Two paths based on source:**

**Spell-cast buffs** (applied via `buff_ids` in spell resolution):
```
base = targetMaxPool * tick_percent
if tick_variance > 0:
    base += targetMaxPool * random(0, tick_variance)
scaled = base * SkillMultiplier(casterSpellcasting) * casterWeaponSpellMult
amount = max(tick_min, round(scaled))
if tick_percent < 0:
    amount = -amount  (ensure negative for damage)
```

**Non-spell buffs** (potions via `drink.go`, items, scripts):
```
base = targetMaxPool * tick_percent
if tick_variance > 0:
    base += targetMaxPool * random(0, tick_variance)
amount = max(tick_min, round(abs(base)))
if tick_percent < 0:
    amount = -amount
```

No stat scaling — potions work the same for everyone.

#### Tick Handler

In `NewRound_UserRoundTick.go`, before calling JS `onTrigger`:

1. If buff has `TickAmount != 0`:
   - Apply to the specified pool (`AddHealth`, `AddStamina`, or
     `AddConviction`)
   - YAML trigger text already sent by Phase 1 wiring
2. Then call JS `onTrigger` if script exists (for exotic buffs)

#### start_remove_buffs Handler

In `Buff_ApplyBuffs.go`, after sending start text and before calling
JS `onStart`:

1. If `buffSpec.StartRemoveBuffs` is non-empty:
   - For each buff ID in the list, remove it from the character

This replaces minor antidote's JS `onStart` logic entirely.

#### Buffs Migrated

**JS files deleted entirely:**

| Buff | tick_pool | tick_percent | tick_variance | Notes |
|------|-----------|-------------|---------------|-------|
| 32-vital_surge | health | 0.05 | 0 | Spell heal |
| 33-chrysalis_regeneration | health | 0.08 | 0 | Spell heal |
| 39-venom | health | -0.08 | 0.04 | Spell DoT |
| 40-spore_toxin | health | -0.05 | 0.03 | Spell DoT |
| 78-toxic_cloud | health | -0.06 | 0.04 | Grenade DoT |

**JS files deleted (after start_remove_buffs handles cure):**

| Buff | start_remove_buffs | tick_pool | tick_percent | Notes |
|------|-------------------|-----------|-------------|-------|
| 47-minor_antidote | [39, 40] | health | 0.05 | Cure + heal |

**Default-world buffs also migrated (simple percentage pattern):**

| Buff | tick_pool | tick_percent | Notes |
|------|-----------|-------------|-------|
| 5-minor_potion_healing | health | 0.04 | Potion (no stat scaling) |
| 6-stamina_draught | stamina | 0.08 | Potion (no stat scaling) |
| 7-conviction_draught | conviction | 0.08 | Potion (no stat scaling) |
| 50-greater_healing | health | 0.12 | Potion (no stat scaling) |

**Stays as JS:**

| Buff | Reason |
|------|--------|
| 24-death_recovery | Flat dice rolls (1d10) across health+mana, conditional messaging. Does not fit percentage pattern. Migrate in Phase 4. |

---

### Part 3: Validation and Testing

#### Load-time Validation

**Spells:**
- If `summon_mob_id > 0`: warn if `summon_base_pool` is 0
- If `summon_requires_corpse` and `summon_min_corpse_pool` is 0: warn
- If `summon_component_id > 0`: validate item exists

**Buffs:**
- If `tick_pool` is set: validate it's one of health/stamina/conviction
- If `tick_percent` is 0 and `tick_pool` is set: warn
- If `start_remove_buffs` references unknown buff IDs: warn

#### Testing

**Unit tests:**
- `ResolveCompanionSummon` with various configurations (no component,
  with component, with corpse, unique key)
- Tick amount snapshot computation (spell-cast vs non-spell, with/without
  variance, min floor)

**Manual smoke tests:**
- Cast each type of summon: a conjure, a raise, a summon-with-component
- Apply a healing buff via spell, verify scaled tick amount
- Drink a healing potion, verify unscaled tick amount
- Apply a DoT, verify damage ticks
- Use minor antidote, verify poison removed + heal ticks

## Files Modified

**Go source (new/modified):**
- `internal/spells/spells.go` — 7 new fields on SpellData
- `internal/buffs/buffspec.go` — 5 new fields on BuffSpec
- `internal/buffs/buffs.go` — `TickAmount` field on Buff struct
- `internal/hooks/spell_resolution.go` — `ResolveCompanionSummon()`
  function, tick snapshot on buff application
- `internal/hooks/Buff_ApplyBuffs.go` — `start_remove_buffs` handler
- `internal/hooks/NewRound_UserRoundTick.go` — auto-tick handler

**Data files (modified/deleted):**
- 14 companion spell YAML files gain summon fields
- 14 companion spell JS files deleted
- ~6-10 buff YAML files gain tick/cure fields
- ~6-10 buff JS files deleted or stripped of onTrigger

**Estimated net change:**
- ~20-24 JS files deleted
- ~200-300 lines of new Go code
- ~20-24 YAML files gain configuration fields

## Future Work

- **Charm spell** → Go migration before/during Phase 4
- **Phase 3:** Item/room script migration
- **Phase 4:** Mob AI behavior system (structured replacement for ad-hoc JS)
- **Death recovery buff (24):** Uses multi-pool dice rolls — may need
  richer tick config or stays as JS until Phase 4
