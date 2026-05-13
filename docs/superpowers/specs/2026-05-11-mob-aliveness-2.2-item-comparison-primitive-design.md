# Mob Aliveness 2.2 — Item-Comparison Primitive

> **Phase 2 tactical (second chunk).** A two-tier item-value
> primitive used by smart-equipping (chunk 2.3) and equipment-
> aware shopping (chunk 5.3). Pure `ItemValue(spec, profile)`
> for catalog ranking, plus `ItemValueDelta(mob, candidate)`
> for "should I swap?" decisions with smart slot selection.
> Replaces the existing thin `internal/items/compare.go` v0.

## Goal

Give the tactical layer a primitive that answers two questions:

1. "Is item A better than nothing / better than item B for this
   mob?" — boolean upgrade, used by chunk 2.3's equip-if-better
   behavior tree action.
2. "How good is item X for this mob?" — numeric score, used by
   chunk 5.3's equipment-aware shopping (rank a shop's catalog).

The chunk's job is **a small, well-tested scoring package** —
`internal/itemvalue/`. No new behaviors, no decision logic. The
package consumes existing `items.ItemSpec`, `mobs.Mob`, and
`characters.Character` types and produces numbers + a swap-result
struct.

## Architectural musts

The chunk brief lists "multi-axis comparison (damage, mitigation,
weight, slot conflicts, archetype-fit), per-archetype weighting,
returns a score so callers can rank a list" as in-scope.
Brainstorming locked in:

1. **Two-tier API.** Pure `ItemValue(spec items.ItemSpec, profile
   WeightProfile) float64` for catalog ranking. Mob-aware
   `ItemValueDelta(char *characters.Character, profile
   WeightProfile, candidate items.Item) SwapDelta` for swap
   decisions. Pure layer is reusable, mob-aware layer handles
   slot conflicts and current-loadout comparison.

2. **Both archetype systems feed into named weight profiles.**
   DOGMud has two orthogonal archetype systems: `Mob.Archetype`
   (stat-pool distribution — `fighting`, `casting`, `tank`, or
   `""`) and `Mob.BehaviorArchetype` (behavior-tree selection —
   `generic_fighter`, `melee_self_buff`, `pure_caster`,
   `support_caster`, `tank_taunter`, `ambusher`, `lookout`,
   `leader`, `prey`, `combat_passive`, three `noncombat_*`).
   `ProfileFor(stat, behavior)` maps both into a set of six
   named profiles. Behavior archetype takes precedence; stat
   archetype is the fallback when no behavior is authored.

3. **Six weight profiles shipped in v1.** `PhysicalBruiser`,
   `PhysicalTank`, `Stealth`, `MagicalPure`, `MagicalSupport`,
   `Neutral`. Hardcoded as package-level `var` values in Go (not
   YAML); tuning is a balance dial that doesn't need data
   authoring.

4. **Spec-faithful axes + sign correctness.** The score
   considers: stat mods (positive AND negative contributions —
   cursed items penalize), damage multiplier (physical), spell
   damage multiplier, three mitigation channels (physical /
   magical / conviction), and item weight. Defensive secondaries
   (ParryRating, AttackSpeed, StaminaCost), aggro modifiers,
   skill mods, and consumable fields (aging, toxicity, charges)
   are out of scope.

5. **Profile-modulated weight cost.** Static linear weight cost
   in `ItemValue` (`weight × profile.WeightPenaltyPerLb`); plus
   a contextual encumbrance-tier penalty in `ItemValueDelta`
   when the swap would push the mob past a carry-weight tier
   threshold.

6. **Three offhand-strategy bonuses on the profile.** A
   `WeightProfile` carries `DualWieldBonus` (applied when a
   `Weapon` is placed in `Offhand`), `ShieldBonus` (applied when
   an `Offhand`-type item is placed in `Offhand`), and
   `TwoHandedBonus` (applied when scoring a 2H weapon candidate).
   These three bonuses are the levers that make tanks prefer
   shields, bruisers prefer dual-wield, pure casters prefer
   either dual 1H magical OR staves, and support casters prefer
   1H magical + shield. Without these, the raw score is
   dominated by the `DamageMultiplier × 100` term and weapons
   would always beat shields.

7. **Smart slot selection inside `ItemValueDelta`.** When a
   candidate item is compatible with more than one slot (1H
   weapons fit Weapon OR Offhand; rings fit Ring OR Ring2;
   wrists fit Wrist or Wrist2 or mutation-gated extras), the
   delta picks the placement that produces the highest net
   `Score`. The displaced items are returned in the `SwapDelta`
   for downstream consumers (emote text, "what did I lose"
   tracking, etc.).

8. **The old `internal/items/compare.go` is deleted.** Its sole
   caller (`mobs/crafter.go`) is migrated to the new API in this
   chunk. The old tests (`compare_test.go`) are deleted too —
   coverage moves into `internal/itemvalue/*_test.go`.

## Architecture & module layout

`internal/itemvalue/` package, parallel to other cross-cutting
concern packages (`combat`, `shops`, `factions`).

| File | Responsibility |
|------|----------------|
| `internal/itemvalue/types.go` | `WeightProfile` struct, `SwapDelta` struct, `SlotName` typed string + slot constants |
| `internal/itemvalue/profiles.go` | The six named profiles as `var` values, `ProfileFor(stat, behavior string) WeightProfile` resolver |
| `internal/itemvalue/score.go` | `ItemValue(spec items.ItemSpec, profile WeightProfile) float64`, plus helper `IsUpgrade(char *characters.Character, profile WeightProfile, candidate items.Item) bool` (just `ItemValueDelta(...).Score > 0`) |
| `internal/itemvalue/delta.go` | `ItemValueDelta(...)`, internal helpers `compatibleSlotsFor`, `displacedItemsForSlot`, `encumbranceTierPenalty` |
| `internal/itemvalue/context.md` | Package documentation per the chunk SOP |
| `internal/itemvalue/profiles_test.go`, `score_test.go`, `delta_test.go` | Unit tests per the testing plan |
| `internal/items/compare.go` | **DELETED** |
| `internal/items/compare_test.go` | **DELETED** |
| `internal/mobs/crafter.go` | Migrated to use `itemvalue.ItemValueDelta(...)` instead of `items.IsUpgrade` / `items.ItemPower` |

## Public API

```go
package itemvalue

// WeightProfile defines per-axis multipliers used to score items
// for a given archetype / role. Constructed via ProfileFor.
type WeightProfile struct {
    Name string

    // Damage axes (applied to DamageMultiplier × 100 and
    // SpellDamageMultiplier × 100).
    PhysicalDamageWeight float64
    SpellDamageWeight    float64

    // Mitigation axes (per percentage point on the spec).
    PhysicalMitigationWeight   float64
    MagicalMitigationWeight    float64
    ConvictionMitigationWeight float64

    // StatWeights overrides the default weight of 1.0 per stat
    // point. Stat keys ("strength", "dexterity", etc.) absent
    // from this map default to 1.0. Negative weights are
    // allowed (a profile that actively dislikes a stat).
    StatWeights map[string]float64

    // Static weight (in lb) cost — applied in ItemValue.
    WeightPenaltyPerLb float64

    // Contextual penalty applied in ItemValueDelta when the
    // swap pushes the buyer's carry weight past a tier
    // threshold (light→moderate→heavy→overburdened→crushed).
    EncumbranceTierPenalty float64

    // Offhand-strategy bonuses — applied to the candidate's
    // score at placement time, not symmetrically when
    // displaced. See End-to-end flow below.
    DualWieldBonus  float64 // Weapon placed in Offhand
    ShieldBonus     float64 // Offhand-type placed in Offhand
    TwoHandedBonus  float64 // 2H Weapon candidate
}

// SlotName identifies a specific equipment field on
// characters.Worn. Unlike items.ItemType (which can map to
// multiple slots, e.g., ItemType=Ring → SlotRing, SlotRing2),
// a SlotName is unambiguous. Values match the Worn struct field
// names exactly (lowercase first letter not required — strings
// are compared exact).
type SlotName string

const (
    SlotWeapon       SlotName = "Weapon"
    SlotOffhand      SlotName = "Offhand"
    SlotExtraArm1    SlotName = "ExtraArm1"
    SlotExtraArm2    SlotName = "ExtraArm2"
    SlotExtraArm3    SlotName = "ExtraArm3"
    SlotExtraArm4    SlotName = "ExtraArm4"
    SlotHead         SlotName = "Head"
    SlotNeck         SlotName = "Neck"
    SlotShoulders    SlotName = "Shoulders"
    SlotBody         SlotName = "Body"
    SlotBack         SlotName = "Back"
    SlotBelt         SlotName = "Belt"
    SlotWrist1       SlotName = "Wrist1"
    SlotWrist2       SlotName = "Wrist2"
    SlotExtraWrist1  SlotName = "ExtraWrist1"
    SlotExtraWrist2  SlotName = "ExtraWrist2"
    SlotExtraWrist3  SlotName = "ExtraWrist3"
    SlotExtraWrist4  SlotName = "ExtraWrist4"
    SlotGloves       SlotName = "Gloves"
    SlotRing         SlotName = "Ring"
    SlotRing2        SlotName = "Ring2"
    SlotLegs         SlotName = "Legs"
    SlotFeet         SlotName = "Feet"
    SlotTail         SlotName = "Tail"
    SlotComponentBag SlotName = "ComponentBag"
)

// SwapDelta is the result of considering equipping a candidate
// over the character's current loadout.
type SwapDelta struct {
    Score     float64       // net value change (gain - sum of displaced values - encumbrance penalty)
    Slot      SlotName      // chosen target slot ("" if not equippable)
    Displaced []items.Item  // items unequipped to make room (0, 1, or 2)
}

// ProfileFor resolves a mob's archetype fields to a named
// WeightProfile. Behavior archetype takes precedence; stat
// archetype is the fallback. Empty input returns Neutral.
func ProfileFor(statArchetype, behaviorArchetype string) WeightProfile

// ItemValue returns the raw value of an item under the given
// profile. Positive AND negative stat mods contribute. Used to
// rank items independently of any mob's current loadout
// (e.g., "rank this shop's catalog by what's good for me").
func ItemValue(spec items.ItemSpec, profile WeightProfile) float64

// ItemValueDelta returns the net effect of equipping candidate
// over char's current loadout under the given profile. Smart
// slot selection picks the optimal placement. Returns
// SwapDelta{Score: 0, Slot: "", Displaced: nil} when candidate
// is not equippable on this character (e.g., a Tail item on a
// non-tailed mob).
func ItemValueDelta(char *characters.Character, profile WeightProfile,
    candidate items.Item) SwapDelta

// IsUpgrade is sugar over ItemValueDelta(...).Score > 0. Used
// by callers that just want the boolean answer.
func IsUpgrade(char *characters.Character, profile WeightProfile,
    candidate items.Item) bool
```

## Profile resolution

`ProfileFor(stat, behavior)` ordered resolution:

```go
func ProfileFor(stat, behavior string) WeightProfile {
    // (1) Behavior archetype takes precedence.
    switch behavior {
    case "tank_taunter":
        return PhysicalTank
    case "pure_caster":
        return MagicalPure
    case "support_caster":
        return MagicalSupport
    case "ambusher", "lookout":
        return Stealth
    case "generic_fighter", "melee_self_buff", "leader":
        return PhysicalBruiser
    case "combat_passive", "prey",
         "noncombat_passive", "noncombat_questgiver",
         "noncombat_shopkeeper":
        return Neutral
    }
    // (2) Stat archetype fallback when no behavior label.
    switch stat {
    case "fighting":
        return PhysicalBruiser
    case "casting":
        return MagicalSupport
    case "tank":
        return PhysicalTank
    }
    // (3) Default.
    return Neutral
}
```

Note on `MagicalSupport` as the `casting` stat-archetype
fallback: a `casting` mob with no behavior label is ambiguous
(could be pure or support). `MagicalSupport` is the safer
default — slightly lower spell damage weighting and a stronger
shield preference fit "I don't know exactly what this mob does
in combat" better than `MagicalPure`'s dual-wand assumption.

## Profile values (v1 tuning)

All six profiles, full field set:

```
PhysicalBruiser:
  PhysicalDamageWeight        = 1.0
  SpellDamageWeight           = 0.1
  PhysicalMitigationWeight    = 1.0
  MagicalMitigationWeight     = 0.6
  ConvictionMitigationWeight  = 0.4
  StatWeights = {strength:1.5, dexterity:1.2, vitality:1.3,
                 willpower:0.3, charisma:0.3, perception:0.7}
  WeightPenaltyPerLb          = 0.5
  EncumbranceTierPenalty      = 25
  DualWieldBonus              = +80
  ShieldBonus                 = +20
  TwoHandedBonus              = +60

PhysicalTank:
  PhysicalDamageWeight        = 0.5
  SpellDamageWeight           = 0.1
  PhysicalMitigationWeight    = 1.2
  MagicalMitigationWeight     = 1.0
  ConvictionMitigationWeight  = 1.2
  StatWeights = {vitality:1.5, strength:1.0, charisma:1.3,
                 willpower:0.5, dexterity:0.7, perception:0.5}
  WeightPenaltyPerLb          = 0.2
  EncumbranceTierPenalty      = 10
  DualWieldBonus              = 0
  ShieldBonus                 = +80
  TwoHandedBonus              = -20

Stealth:
  PhysicalDamageWeight        = 1.1
  SpellDamageWeight           = 0.1
  PhysicalMitigationWeight    = 0.6
  MagicalMitigationWeight     = 0.4
  ConvictionMitigationWeight  = 0.3
  StatWeights = {dexterity:1.5, perception:1.3, strength:1.0,
                 vitality:0.4, willpower:0.3, charisma:0.3}
  WeightPenaltyPerLb          = 1.8
  EncumbranceTierPenalty      = 80
  DualWieldBonus              = +100
  ShieldBonus                 = 0
  TwoHandedBonus              = -40

MagicalPure:
  PhysicalDamageWeight        = 0.2
  SpellDamageWeight           = 1.5
  PhysicalMitigationWeight    = 0.5
  MagicalMitigationWeight     = 1.0
  ConvictionMitigationWeight  = 0.5
  StatWeights = {willpower:1.5, perception:1.2, charisma:1.0,
                 vitality:0.5, dexterity:0.5, strength:0.3}
  WeightPenaltyPerLb          = 1.5
  EncumbranceTierPenalty      = 60
  DualWieldBonus              = +70
  ShieldBonus                 = -40
  TwoHandedBonus              = +80

MagicalSupport:
  PhysicalDamageWeight        = 0.2
  SpellDamageWeight           = 1.2
  PhysicalMitigationWeight    = 0.7
  MagicalMitigationWeight     = 1.0
  ConvictionMitigationWeight  = 0.8
  StatWeights = {willpower:1.4, charisma:1.3, perception:1.1,
                 vitality:0.8, dexterity:0.5, strength:0.3}
  WeightPenaltyPerLb          = 1.5
  EncumbranceTierPenalty      = 60
  DualWieldBonus              = -80
  ShieldBonus                 = +80
  TwoHandedBonus              = -20

Neutral:
  PhysicalDamageWeight        = 0.7
  SpellDamageWeight           = 0.5
  PhysicalMitigationWeight    = 0.7
  MagicalMitigationWeight     = 0.7
  ConvictionMitigationWeight  = 0.7
  StatWeights = {}  // all stats default to 1.0
  WeightPenaltyPerLb          = 1.0
  EncumbranceTierPenalty      = 35
  DualWieldBonus              = +20
  ShieldBonus                 = +20
  TwoHandedBonus              = 0
```

`StatWeights` keys map by lowercase stat name. Stats absent from
the map default to 1.0 (so a `dexterity` ring on a profile that
doesn't list dexterity still contributes — the map encodes
deviations from neutral).

## ItemValue score formula

```
score := 0.0

// (1) Stat mods (positive AND negative contributions).
for statName, mod := range spec.StatMods:
    weight := profile.StatWeights[statName]  // or 1.0 if absent
    score += float64(mod) × weight

// (2) Physical damage.
if spec.DamageMultiplier > 0:
    score += spec.DamageMultiplier × 100 × profile.PhysicalDamageWeight

// (3) Spell damage.
if spec.SpellDamageMultiplier > 0:
    score += spec.SpellDamageMultiplier × 100 × profile.SpellDamageWeight

// (4) Mitigation channels.
score += float64(spec.PhysicalMitigation)   × profile.PhysicalMitigationWeight
score += float64(spec.MagicalMitigation)    × profile.MagicalMitigationWeight
score += float64(spec.ConvictionMitigation) × profile.ConvictionMitigationWeight

// (5) Weight cost (profile-modulated).
score -= spec.Weight × profile.WeightPenaltyPerLb

return score
```

**Worked example — bruiser comparing two swords:**

- Sword A: `DamageMultiplier=1.5, Weight=4` →
  `score = 0 + 150 × 1.0 - 4 × 0.5 = 148`.
- Sword B (cursed): `DamageMultiplier=1.2, Weight=4,
  StatMods{strength:-5}` →
  `score = (-5 × 1.5) + 120 × 1.0 - 2 = 110.5`. Cursed sword is
  correctly worse.

## ItemValueDelta algorithm

```
ItemValueDelta(char, profile, candidate):
    candidateSpec := candidate.GetSpec()
    candidateRaw := ItemValue(candidateSpec, profile)

    slots := compatibleSlotsFor(candidateSpec, char)
    if len(slots) == 0:
        return SwapDelta{}  // not equippable on this character

    best := SwapDelta{Score: -infinity}
    bestRank := math.MaxInt  // for canonical slot tiebreaker

    for slot in slots:
        displaced := displacedItemsForSlot(char, slot, candidateSpec)

        // Candidate's score INCLUDING placement bonus at this slot.
        // Bonus computed against PRE-swap char state.
        candidateAt := candidateRaw + placementBonus(profile, candidateSpec, slot, char)

        // Displaced items' value INCLUDING the bonuses they were
        // contributing at their current slots (symmetric).
        displacedTotal := 0.0
        for d in displaced:
            dSpec := d.GetSpec()
            currentSlot := slotOf(d, char)
            displacedTotal += ItemValue(dSpec, profile) +
                              placementBonus(profile, dSpec, currentSlot, char)

        netScore := candidateAt - displacedTotal
        netScore -= encumbranceTierPenalty(char, displaced, candidate, profile)

        // Canonical slot ranking for tiebreaker (lower = preferred):
        // SlotWeapon=0, SlotOffhand=1, others by SlotName const
        // definition order.
        rank := canonicalRank(slot)

        if netScore > best.Score ||
           (netScore == best.Score && rank < bestRank):
            best = SwapDelta{Score: netScore, Slot: slot, Displaced: displaced}
            bestRank = rank

    return best
```

`placementBonus` computes the bonuses an item *would receive* at a
given slot, evaluated against pre-swap char state. Identical
function for both candidate (prospective placement) and displaced
items (current placement):

```
placementBonus(profile, spec, slot, char):
    bonus := 0.0

    // 2H weapons always carry their bonus, regardless of slot
    // (2H items only fit Weapon, but the rule is slot-agnostic).
    if spec.Hands == items.TwoHanded:
        bonus += profile.TwoHandedBonus

    if slot == SlotOffhand:
        if spec.Type == items.Weapon:
            // DualWieldBonus is CONDITIONAL: only applies when
            // the (pre-swap) main hand holds a 1H weapon, so the
            // "second attack" synergy is real.
            if char.Equipment.Weapon.ItemId > 0 &&
               char.Equipment.Weapon.GetSpec().Hands != items.TwoHanded:
                bonus += profile.DualWieldBonus
        else if spec.Type == items.Offhand:
            // ShieldBonus is unconditional — shields/foci have
            // intrinsic defensive/utility value even without a
            // main-hand weapon.
            bonus += profile.ShieldBonus

    return bonus
```

**Why symmetric:** applying bonuses only to the candidate (the
naive "asymmetric" approach) breaks the tank-with-shield case.
A tank's currently-equipped shield contributes its raw mit value
(say 18) but loses its `+80 ShieldBonus` on the displaced side,
so an offhand sword candidate scoring `60 + 0 = 60` would beat
the shield's 18 and trigger a wrong swap. Symmetric application
treats the shield's contribution as `18 + 80 = 98` when computing
"what would I lose," correctly keeping the tank's shield.

**Why DualWieldBonus is conditional:** a mob with empty hands or
a 2H main weapon should not get credit for "dual-wielding" when
placing a 1H weapon in offhand — there's no main-hand partner to
generate the extra attack. The conditional check on
`char.Equipment.Weapon` keeps the bonus tied to its actual
gameplay benefit. Side effect: with empty hands, both Weapon and
Offhand placements score the same raw value; the canonical
slot tiebreaker routes the item to main hand by default.

**`slotOf(item, char)`** is a trivial helper that returns the
slot field name an item currently occupies on the character
(scans `char.Equipment` to find where the item lives). For 2H
weapons, that's always `Weapon`. For displaced items returned
by `displacedItemsForSlot`, the current slot is known by
construction — we can plumb it through directly instead of
re-scanning, but the helper exists for clarity.

## Slot compatibility table

`compatibleSlotsFor(candidateSpec, char)` returns the list of
`SlotName` values this item could be placed in, respecting
mutations.

| `items.ItemType` | Candidate `SlotName`(s) | Notes |
|---|---|---|
| `Weapon` (1H) | `SlotWeapon`, `SlotOffhand` | Try both; pick higher Score |
| `Weapon` (`Hands == TwoHanded` or subtype "staff") | `SlotWeapon` only — displaces both `SlotWeapon` and `SlotOffhand` | |
| `Offhand` | `SlotOffhand` | |
| `Head` | `SlotHead` | |
| `Body` | `SlotBody` | |
| `Back` | `SlotBack` | |
| `Belt` | `SlotBelt` | |
| `Gloves` | `SlotGloves` | |
| `Legs` | `SlotLegs` | |
| `Feet` | `SlotFeet` | |
| `Neck` | `SlotNeck` | |
| `Shoulders` | `SlotShoulders` | |
| `Wrist` | `SlotWrist1`, `SlotWrist2` (+ `SlotExtraWrist1`–`SlotExtraWrist4` per Extra Arms mutation level) | |
| `Ring` | `SlotRing`, `SlotRing2` | |
| `Tail` | `SlotTail` if Tail mutation active; else not equippable | |
| `Componentbag` | `SlotComponentBag` | |
| Mutation-gated extra-arm items (custom item type) | `SlotExtraArm1`...`SlotExtraArm4` per Extra Arms mutation level | |
| Consumables, materials, quest items, non-equippables | — | Return empty list → `SwapDelta{}` |

`displacedItemsForSlot(char, slot, candidateSpec)`:

- For singleton slots: returns `[current_item]` if occupied, else `[]`.
- For a 2H weapon candidate placed in `Weapon`: returns
  `[current Weapon, current Offhand]` (both slots cleared).
- For a 1H weapon placed in `Offhand` while the current
  `Weapon` is 2H: returns `[current Weapon]` (the 2H weapon
  must be unequipped to free the offhand). This case will
  typically score badly because the candidate loses the
  `DualWieldBonus` benefit (there's no longer a weapon in main
  hand to dual-wield with).
- For ring / wrist slots with multiple slots full: returns the
  lower-`ItemValue` occupant (smart slot selection picks which
  one to displace).

## Encumbrance tier penalty

`encumbranceTierPenalty(char, displaced, candidate, profile)`:

1. Compute carry-weight delta:
   `delta = candidate.Weight - sum(d.Weight for d in displaced)`.
2. Compute pre-swap encumbrance tier from `char.GetCarriedWeight()`
   and post-swap tier from `char.GetCarriedWeight() + delta`,
   using the same ratio thresholds as `userrecord.prompt.go`'s
   encumbrance label renderer (`light` ≤ 0.50, `moderate` ≤ 0.75,
   `heavy` ≤ 1.00, `overburdened` > 1.00, `crushed` >> 1.00).
3. If post-swap tier is *higher* than pre-swap tier (worse):
   return `profile.EncumbranceTierPenalty × tiers_crossed`.
4. If post-swap tier is *lower* than pre-swap tier (better):
   return `-profile.EncumbranceTierPenalty × tiers_crossed`
   (a score *bonus* for shedding weight).
5. Same tier: return 0.

## Bonus application rule

The three bonuses (`DualWieldBonus`, `ShieldBonus`,
`TwoHandedBonus`) apply **symmetrically** — they are part of
both the candidate's score (when scored at its prospective slot)
and the displaced items' scores (when scored at their current
slots, against pre-swap char state). This is critical for
correct behavior: see "Why symmetric" in the algorithm section.

The `DualWieldBonus` is additionally **conditional**: it only
applies to a `Weapon` placed in `Offhand` when the pre-swap main
hand currently holds a 1H weapon. With an empty main or a 2H
main, the bonus is suppressed because there's no synergy partner.

`ShieldBonus` and `TwoHandedBonus` are unconditional once their
type/slot requirement is met.

This ruleset produces the intended preferences without
thrashing: tanks keep shields, bruisers maintain dual-wield,
pure casters stay on dual wands or staff, support casters stay
on wand+shield. A swap only fires when the new item is enough
better to overcome the symmetric weighting of what would be
lost.

## Worked scenarios

**Bruiser, nothing equipped, considering items:**

- 1H sword (1.2× dmg, 4 lb) → Weapon slot: `120 - 2 = 118`.
- 2H greatsword (1.8× dmg, 8 lb) → Weapon (with offhand
  displaced=[]): `180 - 4 + 60 (TwoHanded) = 236`.
- Dual swords (after main equipped): second sword → Offhand:
  `120 - 2 + 80 (DualWield) = 198`. Total loadout: 118 + 198 = 316.
- Shield (15% mit, 6 lb) → Offhand: `15 - 3 + 20 (Shield) = 32`.

Bruiser prefers dual-wield (316) > 2H (236) > shield+sword (118+32=150).

**Tank, currently 1H sword + shield, considering offhand sword:**

- Current loadout value: `1H sword (60) + shield (98) = 158`.
- Try placing new sword in Weapon (displaces current sword):
  net = new_sword - current_sword = small.
- Try placing new sword in Offhand (displaces shield):
  net = new_sword + 0 (DualWieldBonus for tank) - 98 = strongly
  negative. **Tank correctly keeps the shield.**

**Pure caster, nothing equipped:**

- Wand alone (Weapon): `1.3 × 100 × 1.5 + 0.4 × 100 × 0.2 = 203`.
- Dual wands (main + offhand): 203 + (203 + 70 DualWield) = 476.
- Staff: `1.6 × 100 × 1.5 + 0.8 × 100 × 0.2 + 80 (TwoHanded) = 336`.
- Wand + shield: `203 + (15 × 0.5 + (-40) ShieldBonus) = ~170`.

Pure prefers dual-wand (476) > staff (336) > wand+shield (170).
Matches user preference.

**Support caster, considering loadout options:**

- Wand main: `0.4 × 100 × 0.2 + 1.3 × 100 × 1.2 = 164`.
- Wand + shield: main `164` + shield-in-offhand
  `(15 × 0.7) + 80 ShieldBonus = 90.5` → **total 254.5**.
- Dual wands: main `164` + wand-in-offhand
  `164 + (-80) DualWieldBonus = 84` (DualWieldBonus active
  because main is 1H wand) → **total 248**.
- Staff: `1.6 × 100 × 1.2 + 0.8 × 100 × 0.2 + (-20) TwoHandedBonus = 188`.

Support prefers wand+shield (254.5) > dual wands (248) > staff
(188). Matches user preference.

## Migration

`internal/items/compare.go` and its tests are deleted.
`internal/mobs/crafter.go:390,397` is rewritten:

```go
// Before:
isUpgrade := false
wornSameType := false
for _, worn := range wornItems {
    wornSpec := worn.GetSpec()
    if wornSpec.Type == candidateSpec.Type {
        wornSameType = true
        if items.IsUpgrade(wornSpec, *candidateSpec) {
            isUpgrade = true
            break
        }
    }
}
if !wornSameType && items.ItemPower(*candidateSpec) > 0 {
    isUpgrade = true
}

// After:
profile := itemvalue.ProfileFor(mob.Archetype, mob.BehaviorArchetype)
candidate := items.New(recipe.Output.ItemId)
isUpgrade := itemvalue.IsUpgrade(&mob.Character, profile, candidate)
```

The new version handles "what slot does this go in?" and "which
worn item gets displaced?" internally, so the per-worn-item
iteration loop disappears. Net ~10 lines deleted from
`crafter.go`.

## Testing

**`profiles_test.go`** — `ProfileFor` resolution:

- Each named behavior archetype → expected profile (table-driven).
- Each stat archetype with empty behavior → expected fallback.
- Empty input → `Neutral`.
- Unknown behavior → falls through to stat archetype.

**`score_test.go`** — `ItemValue` cases:

- Empty spec → 0.
- Positive stat mod contributes per profile weight.
- **Negative stat mod penalizes** (cursed strength gauntlet on
  bruiser scores below zero).
- Different profiles produce different scores for the same item
  (a `+10 strength` ring scores higher for `PhysicalBruiser`
  than for `MagicalPure`).
- `DamageMultiplier` scales linearly with profile weight.
- `SpellDamageMultiplier` ditto.
- Mitigation channels each contribute independently.
- Weight cost reduces score; profile-modulated weight costs
  differ between `Stealth` (high penalty) and `PhysicalTank`
  (low penalty).

**`delta_test.go`** — `ItemValueDelta` and `IsUpgrade` cases:

- Empty slot → clean gain (Displaced empty, Score = ItemValue +
  any placement bonus).
- Same-slot swap → `Score = new - displaced`.
- 2H weapon displaces both Weapon and Offhand.
- Ring slot with two occupants → displaces the weaker.
- Offhand-capable 1H weapon picks the slot with higher Score.
- Non-equippable item → `SwapDelta{Score:0, Slot:"", Displaced:nil}`.
- Mutation-gated slots respected (Tail without mutation → not
  equippable).
- Tank with shield-vs-sword scenario picks shield.
- Bruiser with sword-vs-sword scenario dual-wields.
- Pure caster prefers dual wands over staff.
- Support caster prefers wand+shield over dual wands.
- Encumbrance tier penalty fires when swap crosses a threshold.
- Encumbrance tier bonus applies when swap relieves tier pressure.

**Existing crafter tests** in `internal/mobs/crafter_test.go`
must still pass. If any test asserts a specific `ItemPower`
value, update it to assert through the new API or delete if it
was testing internals. No fixture-loading concerns.

## Smoke test

After unit tests pass:

1. Run `go build ./...` and confirm clean compile.
2. Boot the server locally; confirm clean startup past data
   loading (per CLAUDE.md SOP).
3. From the admin console, spawn a crafter mob (`mobs/crafter.go`
   consumer of the new API). Verify the mob's crafting decisions
   still produce reasonable upgrades.
4. (Optional) Quick spot-check via admin: dump the score of a
   few catalog items for a specific mob to confirm the profiles
   produce qualitatively right rankings.

## Out of scope / deferred

- **Hysteresis.** v1 uses strict `Score > 0` as the upgrade
  threshold. Add a `MinUpgradeDelta` profile knob if thrashing
  becomes a real problem.
- **Defensive secondaries.** `ParryRating`, `AttackSpeed`,
  weapon `StaminaCost` are not factored into the score. Picked
  up in a tuning pass once data shows it's needed.
- **Aggro modifiers.** Useful for tank, undesirable for
  non-tanks. Out of scope here.
- **Skill mods on item instances.** Instance-zone dropped items
  often carry `+skill` modifiers as part of their generated
  affixes (loot affix system, not on the base `ItemSpec`
  template). The score formula reads `spec.StatMods` only, so
  instance-level skill mods don't currently contribute. When
  the scoring picks up the resolved instance data (i.e., the
  `items.Item.GetSpec()` after affix resolution merges them
  into the spec view), skill mods would naturally appear in
  `StatMods` and start contributing. Out of scope for chunk
  2.2 tuning; profile `StatWeights` would just need entries
  for skill names alongside stat names when this is taken up.
- **Toxicity / aging / charge counts.** Consumable-specific
  fields. The score is for equippable gear; potion decisions
  go through a separate path.
- **Per-item-instance variance.** Two longswords with the same
  `ItemId` but different `EnchantTier`: `items.New()` already
  returns post-resolved values, so they get different scores
  through the existing `GetSpec()` view. No special handling.
- **Mitigation cap awareness.** The score ignores the 75%
  mitigation cap. Cap isn't reachable in current content; not
  worth modeling yet.
- **Caching / memoization.** `ItemValue` is a pure function
  over `(ItemSpec, WeightProfile)`. Could be memoized later if
  profiling shows it's hot. v1 runs the math each call.
- **Symmetric bonus application on displaced items.** The
  three offhand-strategy bonuses apply only when scoring
  candidates, not when scoring displaced. Conservative
  "sticky" behavior. Revisit if observed mob behavior is too
  stuck on a loadout once adopted.
- **Player gear advisor.** This is mob-side scoring. Building
  a player-facing "compare these items" UI is out of scope but
  would be straightforward on top of `ItemValue` later.
- **Data-defined profiles.** v1 profiles are Go vars, not YAML.
  Tuning is code change. Promote to YAML if content authoring
  ever wants to tune profiles per zone or per content pass.

## Roadmap touchpoints

This chunk:

- Closes chunk **2.2** on `MOB_ALIVENESS_ROADMAP.md`.
- Unblocks **2.3** (equip-if-better behavior) and **5.3**
  (equipment-aware shopping) by providing the value primitive
  they will consume.
- Deletes the v0 helpers from `internal/items/compare.go`;
  migrates `mobs/crafter.go` to the new API.
