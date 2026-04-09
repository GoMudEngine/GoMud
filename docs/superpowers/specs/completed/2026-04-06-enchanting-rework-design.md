# Enchanting System Rework — Design Spec

**Date:** 2026-04-06
**Scope:** Fix broken targeting UX, full enchantment audit + rebalance, new
enchants for uncovered slots, 2H weapon scaling.

---

## 1. Problem Statement

The current enchanting system has four bugs that make it largely unusable:

1. **Menu displays but can't be used** — shows `[1] [2] [3]` numbered options
   but no code handles numeric input.
2. **Item specifier parsing is fragile** — strips recipe name from input and
   does substring matching on `DisplayName()`, which fails frequently.
3. **CraftingState doesn't store the target** — only stores `RecipeId` + round
   counters, so at completion time it re-discovers targets and blindly picks
   `candidates[0]`.
4. **Completion always enchants the wrong item** — since no selection is
   stored, it grabs the first matching item type regardless of intent.

Additionally, 8 equipment slots have zero enchantment coverage, existing weapon
enchants are mechanically identical (all `damage_multiplier_bonus`), and
mitigation enchants only cover physical damage.

---

## 2. Targeting UX — Equip-Slot Model

### Core Principle

Enchanting targets **equipped items by slot**, not by item name search. The
command syntax becomes:

```
craft <recipe-name> [slot-specifier]
```

### Slot Resolution

The system builds an ordered list of equipment slots whose **occupied item**
matches the recipe's `target_type`. The slot specifier uses the same `#N` /
`N.` disambiguation the game already uses everywhere else.

**Slot ordering by target_type:**

| target_type | Slot order |
|-------------|------------|
| `weapon`    | Wielded → Offhand → ExtraArm1-4 (only slots with weapon-type items) |
| `shield`    | Offhand → ExtraArm1-4 (only slots with shield-type items) |
| `head`      | Head |
| `neck`      | Neck |
| `shoulders` | Shoulders |
| `body`      | Body |
| `back`      | Back |
| `belt`      | Belt |
| `wrist`     | Wrist1 → Wrist2 → ExtraWrist1-4 |
| `gloves`    | Gloves |
| `ring`      | Ring1 → Ring2 |
| `legs`      | Legs |
| `feet`      | Feet |

### Command Examples

```
craft honed-edge              → weapon #1 (wielded)
craft honed-edge weapon       → same, explicit
craft honed-edge weapon#2     → offhand weapon
craft honed-edge 2.weapon     → offhand weapon (diku-style)
craft honed-edge weapon#3     → extra arm 1
craft thornguard              → shield #1
craft thornguard shield#2     → second shield
craft ironblood ring          → ring #1
craft ironblood ring#2        → ring #2
craft ironblood 2.ring        → ring #2 (diku-style)
craft chitin-brace wrist      → wrist #1
craft chitin-brace wrist#2    → wrist #2
craft carapace-ward           → body (only one body slot, no specifier needed)
```

### Parsing Rules

1. Strip recipe name (by Name or RecipeId) from the input to get the
   slot specifier. If nothing remains, specifier is empty.
2. Parse specifier for `#N` suffix or `N.` prefix to extract the index
   (1-based, default 1).
3. Strip the index syntax to get the base slot name (e.g., `weapon`,
   `ring`, `wrist`, `shield`).
4. Build the ordered candidate list for the recipe's `target_type`.
5. If specifier is empty and exactly one candidate exists, use it.
6. If specifier is empty and multiple candidates exist, error with a
   message listing the occupied slots and their indices.
7. If specifier is provided, select by index from the candidate list.
8. If the selected slot is empty or the item doesn't match, error.

### Storing the Target

`CraftingState` gains a new field:

```go
type CraftingState struct {
    RecipeId       string
    RoundsTotal    int
    RoundsComplete int
    TargetSlot     string  // NEW: slot label (e.g., "wielded", "worn - ring2")
}
```

At craft initiation, resolve the slot and store its label. At completion,
use `GetSlotPointer(TargetSlot)` to get the item pointer directly. If the
item was unequipped mid-craft, the craft fails gracefully ("The item is no
longer equipped.").

---

## 3. Two-Handed Weapon Scaling

When `ApplyTier` detects a 2-handed weapon (`spec.Hands >= 2` or the item
occupies both Weapon and Offhand), **both effects and reserve_pct are
doubled**.

This reflects that a 2H weapon takes two equipment slots, so the enchantment
should be proportionally stronger (and costlier). Implementation is a simple
`2.0` multiplier applied in `ApplyTier` after computing the base values.

The doubling applies to:
- All numeric effect values (`damage_multiplier_bonus`, stat mods, etc.)
- The `reserve_pct` for that tier

The item's `Hands` field on ItemSpec determines this — no new data needed.

---

## 4. Enchantment Roster — Full Audit

All enchantments get **5 tiers** with the standard reserve curve:
**1% → 2% → 4% → 6% → 8%** (doubled for 2H weapons).

### 4.1 Weapon Enchants (target_type: weapon)

**Honed Edge** — Starter enchant. Pure damage scaling.
- Pool: Health
- Skill minimum: 0
- Effect: `damage_multiplier_bonus` (2/4/6/8/10 per tier, i.e., +0.02 to
  +0.10)
- Expanded from 2 to 5 tiers.

**Serpent's Edge** — Damage plus stamina drain flavor.
- Pool: **Stamina**
- Skill minimum: 4
- Effect: `damage_multiplier_bonus` (5/10/15/20/25 per tier)
- Higher damage ceiling than Honed Edge but drains stamina instead of health.

**Hungering Touch** — Lifesteal. Heals on hit.
- Pool: **Health**
- Skill minimum: 5
- Effect: `lifesteal_pct` (new effect key). At each tier, a percentage of
  physical damage dealt is returned as healing: 3%/5%/8%/11%/15%.
- Requires new code in the combat damage resolution to check for lifesteal
  and apply healing post-hit.

### 4.2 Shield Enchant (target_type: shield)

**Thornguard** (NEW)
- Pool: Health
- Skill minimum: 3
- Effect: `block_return_pct` (new effect key). When the wearer blocks, a
  percentage of the blocked damage is dealt back: 5%/10%/15%/20%/25%.
- Leverages the existing return damage system. Implementation adds a check
  in the block resolution path.

### 4.3 Head Enchants (target_type: head)

**Mindweave** — Willpower boost for casters.
- Pool: Conviction
- Skill minimum: 4
- Effect: `willpower_statmod` (1/2/3/4/5)

**Chrysalis Sight** — Perception boost for scouts/ranged.
- Pool: Conviction
- Skill minimum: 6
- Effect: `perception_statmod` (1/2/3/4/5)

### 4.4 Neck Enchant (target_type: neck)

**Ironblood** — Conviction mitigation + willpower.
- Pool: Conviction
- Skill minimum: 5
- Effect: `conviction_mitigation_bonus` (1/2/3/4/5) +
  `willpower_statmod` (1/1/2/2/3)

### 4.5 Shoulders Enchant (target_type: shoulders) — NEW

**Spore Mantle**
- Pool: Stamina
- Skill minimum: 3
- Effect: `magical_mitigation_bonus` (1/2/3/4/5)
- Flavor: fungal spores form a living ward against magical attacks.

### 4.6 Body Enchants (target_type: body)

**Carapace Ward** — Physical mitigation.
- Pool: Health
- Skill minimum: 3
- Effect: `physical_mitigation_bonus` (1/2/3/4/5)

**Sporeweave** — Dexterity boost.
- Pool: Stamina
- Skill minimum: 2
- Effect: `dexterity_statmod` (1/2/3/4/5)

### 4.7 Back Enchant (target_type: back) — NEW

**Shadowweave**
- Pool: Conviction
- Skill minimum: 4
- Effect: `magical_mitigation_bonus` (1/2/3/4/5)
- Flavor: the cloak absorbs incoming magical energy.

### 4.8 Belt Enchant (target_type: belt) — NEW

**Rootbind**
- Pool: Health
- Skill minimum: 2
- Effect: `vitality_statmod` (1/2/3/4/5)
- Flavor: root-like growths anchor you to the earth, bolstering endurance.

### 4.9 Wrist Enchant (target_type: wrist) — NEW

**Chitin Brace**
- Pool: Health
- Skill minimum: 1
- Effect: `physical_mitigation_bonus` (1/2/3/4/5)
- Low skill requirement — early access to defensive enchanting.

### 4.10 Gloves Enchant (target_type: gloves) — NEW

**Venomgrip**
- Pool: Stamina
- Skill minimum: 4
- Effect: `strength_statmod` (1/2/3/4/5)
- Flavor: venom-laced chitin hardens around the hands.

### 4.11 Ring Enchants (target_type: ring)

**Predator's Instinct** — Perception boost.
- Pool: Stamina
- Skill minimum: 5
- Effect: `perception_statmod` (1/2/3/4/5)

**Chrysalis Bond** (NEW) — Conviction mitigation.
- Pool: Conviction
- Skill minimum: 3
- Effect: `conviction_mitigation_bonus` (1/2/3/4/5)
- Flavor: the ring bonds to your psyche, warding against mental assault.

### 4.12 Legs Enchant (target_type: legs)

**Chrysalis Stride** — Dexterity boost.
- Pool: Stamina
- Skill minimum: 4
- Effect: `dexterity_statmod` (1/2/3/4/5)

### 4.13 Feet Enchant (target_type: feet) — NEW

**Rootwalker**
- Pool: Health
- Skill minimum: 2
- Effect: `physical_mitigation_bonus` (1/2/3/4/5)
- Flavor: root-like soles grip the earth, grounding against impacts.

---

## 5. Mitigation Coverage Summary

| Channel    | Sources |
|------------|---------|
| Physical   | Carapace Ward (body), Chitin Brace (wrist), Rootwalker (feet) |
| Magical    | Spore Mantle (shoulders), Shadowweave (back) |
| Conviction | Ironblood (neck), Chrysalis Bond (ring) |

---

## 6. Pool Distribution Summary

| Pool       | Count | Enchantments |
|------------|-------|-------------|
| Health     | 7     | Honed Edge, Hungering Touch, Thornguard, Carapace Ward, Rootbind, Chitin Brace, Rootwalker |
| Stamina    | 6     | Serpent's Edge, Sporeweave, Spore Mantle, Venomgrip, Predator's Instinct, Chrysalis Stride |
| Conviction | 5     | Mindweave, Chrysalis Sight, Ironblood, Shadowweave, Chrysalis Bond |

---

## 7. Recipe Ingredients

Each recipe needs ingredients. Reuse existing crafting materials where
possible, with higher-skill recipes requiring rarer drops.

| Skill Min | Ingredient Pattern |
|-----------|-------------------|
| 0-1       | 1x binding-paste + 1x common forage herb |
| 2-3       | 1x binding-paste + 1x uncommon material |
| 4-5       | 1x chrysalis-setting + 1x mutation-catalyst + 1x binding-paste |
| 6         | 1x chrysalis-setting + 2x mutation-catalyst + 1x binding-paste |

Specific ingredient assignments per recipe will follow this pattern using
existing crafting materials. The pattern is the spec; exact herb/material
choices per recipe are an implementation detail.

### Implementation Note: Data Alignment

The existing Predator's Instinct recipe may have `target_type: legs` while
the enchantment def has `target_type: ring`. All recipe/enchantment pairs
must be verified and aligned during implementation. The enchantment roster
in Section 4 is authoritative for target_type assignments.

The actual `ItemType` constant for shields must be verified in the codebase
(likely `offhand` or `shield`) to ensure the `target_type: shield` matching
works correctly.

---

## 8. Code Changes Required

### 8.1 `internal/characters/crafting.go`
- Add `TargetSlot string` field to `CraftingState`.

### 8.2 `internal/usercommands/craft.go` — `craftEnchanting()`
- Replace the entire item-search/menu flow with slot-based targeting.
- New helper: `resolveEnchantSlot(equipment, targetType, specifier)` that
  builds the ordered candidate list, parses `#N`/`N.` syntax, and returns
  the slot label + item pointer (or an error message).
- Store `TargetSlot` in `CraftingState` at initiation.
- Remove the `EquipmentSlot` slice construction (no longer needed here).

### 8.3 `internal/hooks/NewRound_UserRoundTick.go` — completion path
- Replace the `FindTargetItems` call with `GetSlotPointer(TargetSlot)`.
- Validate the item is still equipped and matches expected type.
- Remove the `EquipmentSlot` slice construction (no longer needed here).

### 8.4 `internal/enchantments/enchantments.go` — `ApplyTier()`
- After computing tier effects, check `item.GetSpec().Hands >= 2`.
- If 2H, multiply all numeric effect values by 2.
- The `reserve_pct` doubling is handled separately wherever reserve costs
  are read (likely `GetTierReservePct` or the caller).

### 8.5 `internal/enchantments/enchantments.go` — new effect keys
- `lifesteal_pct`: stored on the item spec, read during combat damage
  resolution to heal the attacker.
- `block_return_pct`: stored on the item spec, read during block resolution
  to deal return damage.

### 8.6 Combat integration (lifesteal + thornguard)
- Physical damage resolution: after applying damage, check attacker's weapon
  for `lifesteal_pct` and heal attacker by that fraction of damage dealt.
- Block resolution: check defender's shield for `block_return_pct` and deal
  that fraction of blocked damage back to attacker.

### 8.7 `internal/crafting/crafting.go`
- `FindTargetItems` and `TargetCandidate` can be simplified or removed since
  the new flow doesn't use them for enchanting. Keep them if other code
  references them.
- `EquipmentSlot` type may become unused — clean up if so.

### 8.8 Data files
- Update all 10 existing enchantment YAML files (rebalanced tiers, corrected
  target_types, pool swaps).
- Create 8 new enchantment YAML files.
- Update all 10 existing recipe YAML files (corrected target_types, adjusted
  ingredients).
- Create 8 new recipe YAML files.
- All enchantments get Chrysalis-themed adjectives and description suffixes
  across 5 tiers.

### 8.9 Production Migration — Re-apply Enchantments In Place

Existing players on prod have enchanted items that need updating. A one-time
migration runs at server startup (gated by a version flag or config toggle so
it only fires once).

**Algorithm:**

1. Iterate all character save files (online + offline).
2. For each item in backpack, component bag, and all equipment slots:
   a. Skip items with `EnchantType == ""`.
   b. Look up the new `EnchantmentDef` by `EnchantType`.
   c. If the def no longer exists (enchant was removed), call
      `StripEnchantment()` and continue.
   d. **Slot validation:** If the item is equipped, verify that the item's
      actual type matches the enchantment's `target_type` in the new roster.
      If mismatched (e.g., a legs enchant on a ring from a prior bug), call
      `StripEnchantment()` and continue.
   e. Clamp `EnchantTier` to `min(currentTier, len(def.Tiers)-1)` in case
      the new def has fewer tiers than the old one (unlikely but safe).
   f. Update `ReservePool` to `def.ReservePool` (handles pool swaps like
      Serpent's Edge Health→Stamina).
   g. Call `ApplyTier(item, def, clampedTier)` to rewrite effects,
      adjectives, and description suffix to new values.
3. Log a summary: `"Migration: updated N enchanted items across M characters"`.

**What players experience:**

- Their enchanted items keep their tier progress.
- Effects silently update to the new balance values (e.g., Hungering Touch
  changes from damage bonus to lifesteal at whatever tier they'd reached).
- Reserve pool assignments correct themselves.
- Items with invalid slot/enchant pairings are stripped (rare edge case).
- No player action required — items just work differently after restart.

**Implementation location:** A new function in `internal/enchantments/` called
from the server startup sequence, after enchantment defs are loaded but before
the game loop starts. Could also live in a `migrations/` package if one exists.

---

## 9. What This Does NOT Change

- Tier-up mechanic (combat use → tier advancement) stays the same.
- `EnchantMaxTier` config knob stays the same.
- Recipe discovery system stays the same.
- Crafting stations (enchanting_circle requirement) stay the same.
- Non-enchanting crafting is completely unaffected.
- The `craft list` display is unaffected.
