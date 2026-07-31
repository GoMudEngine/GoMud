# internal/itemvalue/

Tactical scoring primitive used by behavior trees and shopping
logic to answer "is item A an upgrade?" and "how good is
item X for this mob?"

## Overview

`itemvalue` is the chunk 2.2 (mob aliveness) consolidation of
item-comparison logic. It replaces the v0 helpers
`items.ItemPower` and `items.IsUpgrade` (deleted) with a
two-tier API:

- **Pure score:** `ItemValue(spec, profile) float64` — rank a
  catalog item independently of any mob's loadout. Used for
  e.g., "rank this shop's stock by what's good for me"
  (chunk 5.3 equipment-aware shopping).
- **Mob-aware delta:** `ItemValueDelta(char, profile, candidate)
  SwapDelta` — would this swap improve my loadout? Handles
  slot conflicts, current-loadout comparison, and encumbrance
  tier crossings. Used by chunk 2.3 equip-if-better behavior
  tree action.

`IsUpgrade(char, profile, candidate) bool` is a one-line
sugar over `ItemValueDelta(...).Score > 0`.

## Key Components (file map)

- `types.go` — `SlotName` typed string + slot constants
  matching `characters.Worn` field names verbatim;
  `WeightProfile` (per-axis multipliers + three offhand-strategy
  bonuses); `SwapDelta` (result struct).
- `profiles.go` — Six named profiles (`PhysicalBruiser`,
  `PhysicalTank`, `Stealth`, `MagicalPure`, `MagicalSupport`,
  `Neutral`) as package-level vars; `ProfileFor(stat, behavior)`
  resolver.
- `score.go` — `ItemValue` formula + `IsUpgrade` wrapper.
- `delta.go` — `ItemValueDelta` main algorithm + internal
  helpers (`compatibleSlotsFor`, `displacedItemsForSlot`,
  `placementBonus`, `slotOf`, `itemInSlot`,
  `encumbranceTierPenalty`, `canonicalRank`, `extraArmsLevel`,
  `hasTailMutation`).

## Public API

```go
func ProfileFor(statArchetype, behaviorArchetype string) WeightProfile
func ItemValue(spec items.ItemSpec, profile WeightProfile) float64
func ItemValueDelta(char *characters.Character, profile WeightProfile,
  candidate items.Item) SwapDelta
func IsUpgrade(char *characters.Character, profile WeightProfile,
  candidate items.Item) bool
func CanEquipFromGive(character *characters.Character, behaviorArchetype string) bool
func CanScanFloorLoot(character *characters.Character, behaviorArchetype string) bool
```

## Score formula (ItemValue)

```
score = sum_over_stats(mod × profile.StatWeights[stat] or 1.0)
      + DamageMultiplier × 100 × profile.PhysicalDamageWeight
      + SpellDamageMultiplier × 100 × profile.SpellDamageWeight
      + PhysicalMitigation × profile.PhysicalMitigationWeight
      + MagicalMitigation × profile.MagicalMitigationWeight
      + ConvictionMitigation × profile.ConvictionMitigationWeight
      - Weight × profile.WeightPenaltyPerLb
```

Negative stat mods penalize (cursed items score below zero).

## ItemValueDelta algorithm (sketch)

1. Resolve candidate's compatible slots via
   `compatibleSlotsFor(candidateSpec, char)` (respects
   mutations: Tail, Extra Arms).
2. For each compatible slot:
   - Compute placement bonus on the candidate
     (`TwoHandedBonus` if 2H, `DualWieldBonus` if Weapon at
     Offhand AND main is 1H, `ShieldBonus` if Offhand-type
     at Offhand).
   - Determine displaced items via `displacedItemsForSlot`.
   - Score displaced items symmetrically (their current-slot
     bonuses included).
   - **Apply gear-effectiveness multiplier** (chunk 2.2a): multiply
     both candidate's raw score AND each displaced item's
     `(ItemValue + placementBonus)` by
     `mutations.GearEffectivenessMultiplier(char.Mutations)`.
     Placement bonuses on the candidate side are NOT scaled
     (playstyle synergies, not gear-derived).
   - Subtract encumbrance tier penalty if the swap crosses a
     carry-weight tier.
3. Pick the slot with highest net score. Tiebreaker: canonical
   slot order (Weapon < Offhand < ... ).
4. Return `SwapDelta{Score, Slot, Displaced}`.

**Incorporeal (rank 4) special case:** When `GearEffectivenessMultiplier`
is 0.0 (all gear scores to 0), `ItemValueDelta` naturally returns
`Score <= 0` for any candidate, and chunk 2.3's equip-if-better
behavior tree action skips the swap. No hardcoded special path needed.

## Profiles

`ProfileFor` resolves two archetype systems:

- `Mob.BehaviorArchetype` (primary): `tank_taunter`,
  `pure_caster`, `support_caster`, `ambusher`, `lookout`,
  `generic_fighter`, `melee_self_buff`, `leader`,
  `combat_passive`, `prey`, `noncombat_*`.
- `Mob.Archetype` (stat-pool, fallback): `fighting`, `casting`,
  `tank`, `""`.

The six profiles each have weight tables tuned for distinct
gearing preferences:

- `PhysicalBruiser`: high physical damage, high strength/dex/vit
- `PhysicalTank`: high mitigation, high vitality/charisma
- `Stealth`: high dexterity/perception, weight penalty
- `MagicalPure`: high spell damage/willpower
- `MagicalSupport`: high spell damage and mitigation
- `Neutral`: balanced fallback

Profile values are hardcoded in Go; tuning is a code change,
not data authoring.

## Bonus application rule

The three bonuses (`DualWieldBonus`, `ShieldBonus`,
`TwoHandedBonus`) apply **symmetrically** — same `placementBonus`
math evaluated against pre-swap state, applied to both
candidate (at prospective slot) and displaced items (at their
current slots). `DualWieldBonus` is **conditional**: only fires
when the (pre-swap) main hand has a 1H weapon. Without that
conditional, empty-handed mobs would score a wand-in-offhand
above a wand-in-main, sending the wand to the wrong slot.

## SwapDelta struct

```go
type SwapDelta struct {
	Score     float64      // net value change (gain - displaced - encumbrance penalty)
	Slot      SlotName     // chosen target slot ("" if not equippable)
	Displaced []items.Item // items unequipped to make room (0, 1, or 2)
}
```

## Integration Notes

**Consumed by:**
- `internal/mobs/crafter.go` — replaced old `items.IsUpgrade`
  call; crafter mobs decide whether to craft a candidate
  upgrade.
- `internal/mobcommands/gearup.go` (chunk 2.3) — push-equip
  path; calls `IsUpgrade` and `CanEquipFromGive`.
- `internal/hooks/mob_equip_best_floor_item.go` (chunk 2.3) —
  pull-equip orchestrator; calls `ItemValueDelta`.
- Future: `internal/economy/` or shopping-decision code path
  (chunk 5.3 equipment-aware shopping).

**Depends on:**
- `internal/items` — `ItemSpec`, `Item`, `ItemType` enum,
  `WeaponHands` constants, `GetSpec()`.
- `internal/characters` — `Character`, `Equipment` (Worn),
  `Mutations`.
- `internal/mutations` — `GetMutationLevel()`, `HasMutation()`.

## Global State

None. All functions are pure (no package-level state mutation).
The profile `var` values are read-only after init; callers
must not mutate them.

## Testing Notes

- `profiles_test.go` — table-driven `ProfileFor` resolution
  with all known behavior + stat archetype values.
- `score_test.go` — axis-by-axis coverage: stat mods
  (positive, negative, unknown-key default), damage, spell
  damage, three mitigation channels, weight cost. Worked
  examples from the spec are explicit test cases.
- `delta_test.go` — slot-helper coverage (1H vs 2H, ring,
  consumable, mutation-gated Tail, Extra Arms), placement bonus
  (conditional DualWield, unconditional Shield), main
  algorithm via documented expectations (some tests `t.Skip`
  pending integration fixtures; full coverage in chunk smoke).

Tests do NOT require the test data directory or balance config
to be loaded; they construct synthetic `ItemSpec` and
`Character` values inline.

## SlotName Constants

All 25 equipment slots, matching `characters.Equipment` field names:

- Weapon slots: `Weapon`, `Offhand`, `ExtraArm1-4`
- Head/neck/torso: `Head`, `Neck`, `Shoulders`, `Body`, `Back`
- Waist: `Belt`
- Hands: `Wrist1`, `Wrist2`, `ExtraWrist1-4`, `Gloves`
- Jewelry: `Ring`, `Ring2`
- Legs/feet: `Legs`, `Feet`
- Mutation-gated: `Tail` (requires tail mutation)
- Utility: `ComponentBag`

## Performance Considerations

- `ItemValue` is O(1) in the number of stats (max 6) and
  damage axes (max 2).
- `ItemValueDelta` iterates compatible slots (typically 1-2,
  max 6 for rings) and displaced items (0-2). Scales well
  even for complex loadouts.
- No heap allocations during scoring; all computations are
  floating-point math.

## Equip-If-Better Integration (Chunk 2.3)

Two consumer paths use `IsUpgrade` to drive automatic mob gear
behavior:

**Push (give-equip):** When a player gives a mob an item,
`internal/mobcommands/gearup.go` calls `IsUpgrade` per backpack
item and equips upgrades. Non-combat archetypes and animal
species silently skip via `CanEquipFromGive` (gates via
`Species.DisabledSlots` containing "Weapon", and non-combat
archetype list: `noncombat_passive`, `noncombat_questgiver`,
`noncombat_shopkeeper`, `prey`, `combat_passive`). Charmed mobs
(companions, mercs) accept pushes from their owner; displaced
gear drops to the floor so the owner can reclaim it.

**Pull (idle floor-loot scan):** Every idle tick,
`internal/hooks/MobIdle_HandleIdleMobs.go` calls
`hooks.EquipBestFloorItem(mob, room)` (lives in the `hooks`
package — see import-cycle note below). That function scans
`room.Items`, scores each via `ItemValueDelta`, and equips the
best positive-scoring upgrade. Combat-state gated. Charmed mobs
are additionally excluded via `CanScanFloorLoot` (owner has
dibs on floor loot).

**Incorporeal handling (chunk 2.2a):** Incorporeal mobs are handled
automatically: their gear scores 0 via
`mutations.GearEffectivenessMultiplier`, so `IsUpgrade` returns
false and no swap occurs. No special skip path is needed at the
eligibility-gate layer.

**Import-cycle note:** The gate helpers (`CanEquipFromGive`,
`CanScanFloorLoot`) take `(*characters.Character, string)`
rather than `(*mobs.Mob)` because `mobs` already imports
`itemvalue` (via `internal/mobs/crafter.go`). For the same
reason, `EquipBestFloorItem` lives in `internal/hooks/` rather
than here — it needs `*rooms.Room` (which transitively imports
`mobs`), and that would close the cycle. Callers pass
`&mob.Character, mob.BehaviorArchetype`.

## Files

| File | Purpose |
|------|---------|
| `types.go` | Value/score shapes |
| `score.go` | Scoring an item |
| `delta.go` | Comparing two items |
| `profiles.go` | Per-archetype weighting profiles |
| `equip_eligibility.go` | Whether an actor can use an item at all |

Used by mob AI to decide whether looted gear is an upgrade, and by the
appraisal display.
