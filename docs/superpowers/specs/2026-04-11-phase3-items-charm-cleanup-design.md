# Phase 3: Item Script Cleanup + Charm Spell Migration

**Date:** 2026-04-11
**Status:** Draft
**Phase:** JS Audit Phase 3 — Dead code removal, one item migration,
charm spell ported to Go

## Goal

Eliminate 13 JS files: 11 dead default-world item scripts (shadowed or
unused), 1 DOGMud item script (migrated to YAML), and the charm spell
(ported to a Go function). After this phase, all remaining JS files are
mob scripts, room scripts, and tutorial sequences — Phase 4 territory.

## Scope

**In scope:**
- Delete 11 unused default item JS files
- New YAML fields on ItemSpec for use-triggered skill training
- Migrate herbalism_recipe_page.js to YAML fields
- Port charm.js to a Go function `resolveCharmSpell()`
- Delete charm.js

**Out of scope:**
- Room scripts (deferred to Phase 4 behavior system)
- Complex item behaviors (sentient weapons, proc effects — Phase 4)
- Default-world item YAML cleanup (only deleting JS, not touching YAMLs)
- Mob scripts (Phase 4)

## Design

### Part 1: Dead Code Deletion

Delete 11 JS files from `_datafiles/world/default/items/other-0/`:

| File | Reason |
|------|--------|
| `4-winterfire_crystal.js` | No DOGMud item 4, no references |
| `6-sleeping_bag.js` | No DOGMud item 6, no references |
| `10-history_of_frostfang.js` | Shadowed by DOGMud item 10 |
| `19-the_shadow_herbarium.js` | Shadowed by DOGMud item 19 |
| `21-stat_coupon.js` | Shadowed by DOGMud item 21 |
| `22-training_coupon.js` | Shadowed by DOGMud item 22 |
| `24-spellbound_projectiles.js` | Shadowed by DOGMud item 24 |
| `26-broom.js` | Shadowed by DOGMud item 26 |
| `100-newbie_kit.js` | Not used in DOGMud |
| `101-arcane_flute.js` | Not used in DOGMud |
| `102-room_rental.js` | Not used in DOGMud |

No YAML or Go changes. Engine discovers scripts by filename — missing
file means no script, which is correct since these items either don't
exist or are shadowed by DOGMud items at the same ID.

### Part 2: Herbalism Recipe Page Migration

#### New ItemSpec Fields

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `on_use_train_skill` | string | "" | Skill name to train on use |
| `on_use_train_amount` | int | 1 | Amount to train |
| `on_use_user_text` | string | "" | Text sent to user on use. Supports `{source}` token. |
| `on_use_room_text` | string | "" | Text sent to room on use. Supports `{source}` token. |

#### Go Handler

In the item use handler (wherever `onCommand_use` is dispatched from
Go), add a check: if `itemSpec.OnUseTrainSkill` is non-empty, train the
skill, consume a use, send YAML text, and return. This runs before any
JS `onCommand_use` hook.

#### Migration

`_datafiles/world/dogmud/items/materials-40000/40042-herbalism_recipe_page.yaml`
gains:
```yaml
on_use_train_skill: search
on_use_train_amount: 1
on_use_user_text: "You study the recipe page carefully, absorbing its wisdom."
on_use_room_text: "{source} studies a worn recipe page intently."
```

Delete `40042-herbalism_recipe_page.js`.

### Part 3: Charm Spell → Go

#### Go Function: resolveCharmSpell

New file `internal/hooks/charm_spell.go`, called from spell resolution
when `spellData.EffectType == "charm"` (or a new field — check what
makes sense with the existing effect type system).

**Flow:**

1. **Validate target** — must be a mob, not a player. Check
   `IsCharmImmune()` species flag. Error if invalid.
2. **Companion cap check** — same as summon spells.
3. **Compute attack score:**
   ```
   attack = charisma + manifestation_skill * 25
   ```
4. **Compute defense score:**
   ```
   defense = target_willpower + target_stat_pool * 0.10
   ```
5. **Apply aggro penalties:**
   - If target is fighting the caster: `defense *= 0.75`
   - If target is fighting someone else: `defense *= 0.85`
   - If target is not fighting: no penalty
6. **Opposed roll:** `dice.OpposedRollStat(attack, defense)`
7. **On success:** charm the mob (`CharmSet(userId, duration)`),
   register as companion, clear target's aggro, send success text.
8. **On failure:** send failure text. If roll was close, flavor text
   hints the target wavered. If roll was far, flat rejection.

**Spell YAML change:** Set `effect_type: charm` on `charm.yaml` (or
add a new field if `effect_type` is already used for something else
on this spell). The spell resolution code checks for this and calls
`resolveCharmSpell()`.

**Cast/wait text** already in YAML from Phase 1. The Go function only
handles `onMagic` phase logic.

Delete `_datafiles/world/dogmud/spells/charm.js`.

### Part 4: Validation and Testing

**Load-time validation:**
- If `on_use_train_skill` references an unknown skill name: warn

**Testing:**
- Manual: use the herbalism recipe page, verify skill trains
- Manual: cast charm on a mob, verify opposed roll + companion
- Manual: cast charm on a charm-immune mob, verify rejection
- Manual: cast charm at companion cap, verify error

## Files Modified

**Go source:**
- `internal/items/itemspec.go` — 4 new fields on ItemSpec
- `internal/usercommands/use.go` (or wherever item use is handled) —
  check for `OnUseTrainSkill`, train + consume + send text
- `internal/hooks/charm_spell.go` — new file, `resolveCharmSpell()`
- `internal/hooks/spell_resolution.go` — wire charm resolution

**Data files:**
- 11 default item JS files deleted
- 1 DOGMud item YAML modified, JS deleted
- 1 spell YAML modified (charm.yaml: effect_type or new field), JS deleted

**Estimated net change:**
- 13 JS files deleted (~350 lines removed)
- ~100-150 lines of new Go code (charm function + item use handler)
- ~10 lines of YAML changes

## Post-Phase 3 JS Inventory

After this phase, remaining JS files are ALL in these categories:
- **Mob scripts** (~26 files) — quest NPCs, AI behaviors, routines
- **Room scripts** (~31 files) — tutorials, ceremonies, state machines
- **Default world buffs** (1 file) — death recovery (dice-based multi-pool)

All of these are Phase 4 behavior system candidates. No orphan scripts
remain outside those categories.
