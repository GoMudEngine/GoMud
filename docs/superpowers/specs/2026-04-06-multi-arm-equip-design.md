# Multi-Arm Equip Rework — Design Spec

**Date:** 2026-04-06
**Scope:** Fix weapon equipping for characters with extra arms mutation.

---

## 1. Problem

The current `Wear()` function can't handle multi-arm loadouts:

1. Equipping a 2H weapon clears ALL extra arm slots (should only need
   one pair of hands).
2. Can't equip 1H weapons after a 2H is wielded — the extra arm auto-
   placement check requires offhand to be occupied, but 2H cleared it.
3. Extra arms can never hold 2H weapons — hardcoded restriction.

## 2. Pair-Based Hand Model

Arms are grouped into pairs:

  - Pair A: Weapon + Offhand     (arm 1 + arm 2)
  - Pair B: ExtraArm1 + ExtraArm2 (arm 3 + arm 4)
  - Pair C: ExtraArm3 + ExtraArm4 (arm 5 + arm 6)

A 2H weapon occupies a full pair (both slots). A 1H weapon or shield
occupies one slot within a pair. Pairs B and C require the extra-arms
mutation at level 1-2 and 3-4 respectively.

**Slot type restrictions:**
- Arm 1 (Weapon slot): weapons only
- Arms 2-6: weapons OR shields

This means the maximum shield loadout is 1 weapon + 5 shields.

## 3. Equip Logic

### 3a. Auto-placement (no slot specified)

**1H weapon/shield:** Find the first empty slot scanning Pair A → B → C.

**2H weapon:** Find the first pair where both slots are free, scanning
A → B → C. If no pair is fully free, displace from the first pair
that has the fewest items (prefer displacing 1 item over 2). Displaced
items return to backpack.

### 3b. Explicit slot targeting

Player can specify which arm: `equip greatsword arm#3` or
`equip greatsword 3.arm`.

Arms 1-6 map to:
  - arm 1 = Weapon, arm 2 = Offhand
  - arm 3 = ExtraArm1, arm 4 = ExtraArm2
  - arm 5 = ExtraArm3, arm 6 = ExtraArm4

A 2H weapon MUST go in an odd-numbered arm (1, 3, 5) — the first
slot of a pair. Attempting an even-numbered arm gives:
"A two-handed weapon needs a pair of arms — try arm 1, 3, or 5."

A 1H weapon/shield can go in any arm slot (1-6).

### 3c. Displacement rules

When a 2H weapon is placed in a pair, any items in both slots of
that pair are displaced to backpack. When a 1H item is placed in an
occupied slot, the existing item is displaced.

Cursed items block displacement as before.

## 4. Dual-Wield Interaction

The existing `CanDualWield()` check (weapon-combat skill > 0) still
gates putting a second weapon in Pair A's offhand slot. However, extra
arm pairs (B and C) do NOT require dual-wield skill — the mutation
itself grants the ability to hold weapons in extra hands.

Claws/martial still bypass dual-wield as before.

## 5. Example Loadouts (6 arms)

  3x 2H:        Pair A (2H), Pair B (2H), Pair C (2H)
  2H + 4x 1H:   Pair A (2H), Pair B (1H+1H), Pair C (1H+1H)
  1H + shield + 2H + 2x shield:
                 Pair A (1H+shield), Pair B (2H), Pair C (shield+shield)
  6x 1H:        One per slot

## 6. Files Changed

### 6a. `internal/characters/character.go` — `Wear()`
Replace the weapon/offhand/extra-arm blocks (lines ~3396-3485) with
pair-based logic. Remove the "clear all extra arms" block.

### 6b. `internal/usercommands/equip.go`
- Replace the `arm1`/`arm2`/`arm3`/`arm4` suffix parsing with
  `arm#N` / `N.arm` parsing using `util.GetMatchNumber()`.
- Support arms 1-6 (currently only 1-4).
- Allow 2H weapons in extra arms (remove the `handsReq > 1` block).
- Add the pair-boundary error for 2H in even-numbered arms.

### 6c. Help files
- Update `help equipment` / `help inventory` (whichever covers equip
  slots) to explain the pair system and arm numbering.
- Add error message for 2H-in-even-arm attempts.

## 7. Defense Score Fix — Parry and Block From Extra Arms

Currently `GetDefenseScore()` only reads the main Weapon slot for parry
and the Offhand slot for block. Extra arm weapons/shields are ignored
for defense. This needs fixing:

### Parry
Sum the **best** ParryRating across all equipped weapons (Weapon +
ExtraArm1-4). Use the highest single ParryRating, not the sum — you
parry with one weapon at a time, but having more weapons means a
better chance of using the best one.

### Block
Sum the **best** BlockRating across all equipped shields (Offhand +
ExtraArm1-4 where item type is offhand). Same logic — block with one
shield at a time, use the best available.

### Files
- Modify: `internal/characters/character.go` — `GetDefenseScore()`
  cases for DefenseParry and DefenseBlock.

## 8. What Does NOT Change

- Enchanting (slot targeting is independent of equip logic)
- Remove/unequip (already works per-slot)
- Shield type detection
- The extra-arms mutation itself
- Attack weapon scanning (already reads all extra arms)
