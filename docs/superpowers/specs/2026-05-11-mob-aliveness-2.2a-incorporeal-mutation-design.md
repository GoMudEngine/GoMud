# Mob Aliveness 2.2a — Incorporeal Mutation

> **Phase 2 tactical (inserted between 2.2 and 2.3).** New rarest
> mutation that models ethereal beings — wraiths, spectres, fire
> and air elementals, the elemental queen. Lower vit, raised wil,
> physical-attack evasion bonus, and gear effectiveness that scales
> down per rank to zero at max rank. Replaces what would otherwise
> be an ad-hoc "ethereal mob" flag in chunk 2.3's equip-if-better
> gating; instead the gate falls out naturally from item value
> scoring multiplied by gear effectiveness.

## Goal

Model "ethereal" / "incorporeal" beings as a mutation rather than
a species flag. The mutation soft-scales gear effectiveness across
four ranks (25/50/75/100% loss) so that:

- Rank-4 incorporeal mobs and players have zero gear value, so
  chunk 2.3's `itemvalue.IsUpgrade` naturally returns false for any
  candidate equipment (no special hardcoded skip path).
- Combat damage from equipped weapons scales down per rank
  (incorporeal beings hit weaker with steel, hit normally with
  natural weapons like claws or bite).
- Mitigation from equipped armor scales down per rank (gear-derived
  defense erodes as you become more ethereal).
- Equipment stat mods scale down per rank (the +5 dex ring becomes
  +1.25 dex at rank 3, nothing at rank 4).

The mutation also gives a flat-bonus physical defense (harder to
hit with weapons), a willpower bonus (more potent magic), and a
vitality penalty (more fragile in absolute HP terms). Existing
non-gear damage paths — natural weapons (claws/bite/slam),
unarmed, mutations like `natural_weapon` — stay at full
effectiveness.

The chunk's job is **a new mutation + its supporting effect types
+ integration in five consumer sites**. The five mob templates
that should be ethereal today get tagged with `mutations:
{ incorporeal: 4 }`.

## Architectural musts

The chunk brief (this sidequest, derived from chunk 2.3
brainstorming) lists the four-rank effect set and the soft-scale
gear approach as in-scope. Brainstorming locked in:

1. **Single uniform `gear_effectiveness_loss` effect type.**
   Value-per-rank summed across owned mutations. One getter,
   one multiplier, applied at every consumer site (stat
   aggregation, weapon damage, three mitigation channels,
   itemvalue scoring). Other future mutations can reuse the
   effect type if they want to scale gear similarly.

2. **Raw-level multiplication for `gear_effectiveness_loss`**,
   skipping the existing `LevelMultiplier(1.0/1.5/2.0/2.5)`
   curve. With `value: 0.25` per rank, this produces a clean
   linear progression of `0.25/0.50/0.75/1.00` loss across
   ranks 1-4. The existing LevelMultiplier curve is the wrong
   shape for percentage-loss effects (it would peak at
   0.625 loss = 37.5% effectiveness at rank 4, never reaching
   "no gear at all"). One-off carve-out via a dedicated
   `GetGearEffectivenessLoss` getter that doesn't call
   `LevelMultiplier`; all other effect types continue to use
   the standard curve via `sumEffects`.

3. **Full effect set in v1.** Gear loss + vitality penalty +
   willpower bonus + physical defense bonus (a new effect
   type added to the combat defense-roll resolution). The
   "uber-rare = unique" framing makes the half-built version
   (gear loss + stats only, no evasion) feel undercooked.

4. **Defense-margin bonus for physical evasion.** New effect
   type `physical_defense_bonus` adds points to the defender's
   roll margin in the best-of-all defense resolution
   (`combat_helpers.go`). Channel-scoped to physical attacks
   only — incorporeal beings stay fully vulnerable to magical
   and conviction damage. Uses standard `LevelMultiplier`
   scaling (the diminishing-returns curve fits "scaling
   defense" semantically).

5. **Conflict list = definitive body-dependent mutations.**
   `extra-arms`, `extra-legs`, `clawed-hands`, `tail`,
   `dense-muscles`, `hollow-bones`, `healing-gel`. Ambiguous
   cases (`cold-blooded`, `iron-constitution`,
   `adrenaline-surge`, `elongated-limbs`, `infrared-vision`)
   stay non-conflicting in v1 — content authors can layer
   conflicts later if specific combos prove unbalanced or
   nonsensical.

6. **Mob tagging — five templates.** Wraith, spectre, fire
   elemental, air elemental, and the Elemental Queen (mob 321,
   `_datafiles/world/dogmud/mobs/instance_planar_oasis/321-elemental_queen.yaml`).
   Earth and water elementals stay corporeal (rock and water
   have physical substance). Per-species judgment for finer
   tagging is a follow-on content concern.

7. **Carry capacity unaffected.** Incorporeal characters still
   have normal carry capacity — backpack items ride along in
   the ethereal form, they just don't function as gear when
   equipped. Avoids edge cases where mutating mid-game would
   dump half a player's inventory.

## Architecture & module layout

Single new mutation YAML; modifications spread across mutations,
characters, combat, and itemvalue packages.

| File | Status | Purpose |
|------|--------|---------|
| `_datafiles/world/dogmud/mutations/incorporeal.yaml` | NEW | Mutation definition |
| `internal/mutations/mutations.go` | MODIFY | `GetGearEffectivenessLoss`, `GearEffectivenessMultiplier`, `GetPhysicalDefenseBonus` |
| `internal/mutations/mutations_test.go` | MODIFY | Tests for the three new helpers + conflict-list verification |
| `internal/characters/character.go` | MODIFY | `Recalculate()` scales gear-derived stat contributions; `GetPhysicalMitigation()` / `GetMagicalMitigation()` / `GetConvictionMitigation()` scale gear-derived portion only |
| `internal/characters/character_test.go` | MODIFY | Tests for scaled aggregation |
| `internal/combat/damage_pipeline.go` | MODIFY | Weapon `itemMult` includes `GearEffectivenessMultiplier(attacker.Mutations)` |
| `internal/combat/combat_helpers.go` | MODIFY | Defense margin gets `GetPhysicalDefenseBonus(defender.Mutations)` added for physical-channel attacks |
| `internal/combat/combat_helpers_test.go` | MODIFY | Tests for the weapon-damage and defense-margin scaling |
| `internal/itemvalue/delta.go` | MODIFY | `ItemValueDelta` applies the gear multiplier to candidate + displaced totals |
| `internal/itemvalue/delta_test.go` | MODIFY | Test that ItemValueDelta returns 0 for rank-4 incorporeal char |
| Mob template YAMLs | MODIFY | Wraith / spectre / fire elemental / air elemental templates + mob 321 (elemental queen) get `mutations: { incorporeal: 4 }` |
| `_datafiles/world/dogmud/templates/help/mutations.template` | MODIFY | Append Incorporeal entry |
| `internal/mutations/context.md` | MODIFY | Document the two new effect types + the raw-level carve-out |
| `internal/itemvalue/context.md` | MODIFY | Note `ItemValueDelta` gear-effectiveness application |
| `internal/characters/context.md` | MODIFY | Note `Recalculate` and mitigation methods apply the gear-effectiveness multiplier |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY | Insert chunk 2.2a entry; bump roll-up |

## The mutation YAML

`_datafiles/world/dogmud/mutations/incorporeal.yaml`:

```yaml
mutationid: incorporeal
name: Incorporeal
description: |
  Your form has slipped between planes — flesh, bone, and sinew
  giving way to something less tangible. Physical attacks struggle
  to find purchase against your shifting form, but the same
  ethereality that protects you weakens your constitution. Worn
  armor and wielded weapons grow less effective the further you
  drift from the physical world. At the deepest rank, gear becomes
  ornamental — passing through your form as if it weren't there.
rarity: 10
visual: Their form shimmers and grows translucent, edges blurring as if not entirely present.
conflicts:
  - extra-arms
  - extra-legs
  - clawed-hands
  - tail
  - dense-muscles
  - hollow-bones
  - healing-gel
pros:
  - type: stat_flat
    target: willpower
    value: 5
  - type: physical_defense_bonus
    target: ""
    value: 15
cons:
  - type: stat_flat
    target: vitality
    value: -10
  - type: gear_effectiveness_loss
    target: ""
    value: 0.25
```

**Per-rank realized effects** (after LevelMultiplier 1.0/1.5/2.0/2.5
except `gear_effectiveness_loss` which uses raw level):

| Rank | Vitality Δ | Willpower Δ | Phys Defense | Gear Loss | Gear Multiplier |
|---|---:|---:|---:|---:|---:|
| 1 | -10 | +5 | +15 | 25% | 0.75× |
| 2 | -15 | +7.5 | +22.5 | 50% | 0.50× |
| 3 | -20 | +10 | +30 | 75% | 0.25× |
| 4 | -25 | +12.5 | +37.5 | 100% | 0.00× |

## New effect types & helpers

Two new effect types in the mutation system, with dedicated
getters in `internal/mutations/mutations.go`:

```go
// GetGearEffectivenessLoss returns the total fraction (0.0–1.0)
// by which all equipment effects (stat mods, weapon damage,
// mitigation values) should be reduced for this character.
// Computed with raw level multiplication (NOT LevelMultiplier)
// so the four-rank progression is linear: 0.25/0.50/0.75/1.00.
// Clamped to [0.0, 1.0].
func GetGearEffectivenessLoss(owned map[string]int) float64 {
    loss := 0.0
    for id, level := range owned {
        spec := GetMutation(id)
        if spec == nil {
            continue
        }
        // Both pros and cons can declare this effect, though the
        // type-name implies a downside; either side is summed.
        for _, c := range spec.Cons {
            if c.Type == "gear_effectiveness_loss" {
                loss += c.Value * float64(level)
            }
        }
        for _, p := range spec.Pros {
            if p.Type == "gear_effectiveness_loss" {
                loss += p.Value * float64(level)
            }
        }
    }
    if loss < 0 {
        loss = 0
    } else if loss > 1 {
        loss = 1
    }
    return loss
}

// GearEffectivenessMultiplier returns the multiplier consumers
// apply to gear-derived values (1.0 = full effectiveness, 0.0 = none).
// Convenience wrapper over GetGearEffectivenessLoss.
func GearEffectivenessMultiplier(owned map[string]int) float64 {
    return 1.0 - GetGearEffectivenessLoss(owned)
}

// GetPhysicalDefenseBonus returns the total bonus added to the
// defender's roll margin for physical-channel attacks. Standard
// LevelMultiplier scaling (diminishing returns curve fits
// "scaling defense" semantically).
func GetPhysicalDefenseBonus(owned map[string]int) float64 {
    return sumEffects(owned, "physical_defense_bonus", "")
}
```

`sumEffects` already handles `LevelMultiplier` internally for any
effect type; the new `physical_defense_bonus` becomes one more
thing it can sum. The raw-level carve-out for
`gear_effectiveness_loss` is implemented as a parallel function
that walks the same data but skips `LevelMultiplier` — keeps
`sumEffects` simple, documents the divergence at the call site.

## Consumer integration sites

Five sites apply one of the two new helpers.

**Site 1: `internal/characters/character.go` — `Recalculate()`**

Where equipment contributions roll up into the character's final
stats. Apply `mutations.GearEffectivenessMultiplier(c.Mutations)`
to the gear-derived portion of the aggregation. Non-gear
contributions (mutation `stat_flat`, buff stat mods, race base)
pass through unchanged. `Worn.StatMod` stays pure (no
char dependency); the multiplier applies at the character-level
seam.

The exact insertion point depends on the current `Recalculate`
shape. If gear and non-gear contributions are summed in a single
pass without separation, that needs a small refactor to make them
distinct so the multiplier can apply to the gear portion only.

**Site 2: `internal/characters/character.go` — three mitigation getters**

```go
func (c *Character) GetPhysicalMitigation() int {
    gearMit := c.Equipment.PhysicalMitigation()
    gearMit = int(float64(gearMit) * mutations.GearEffectivenessMultiplier(c.Mutations))
    // Add non-gear contributions (mutations.natural_armor, buffs, etc.):
    return gearMit + nonGearMit
}
```

Identical shape for `GetMagicalMitigation()` and
`GetConvictionMitigation()`. If the current implementation doesn't
cleanly separate gear vs non-gear, the refactor to separate them
is included in this chunk.

**Site 3: Weapon damage in `internal/combat/damage_pipeline.go`**

The unified formula is:
```
raw = stat × SkillMultiplier(rank) × itemMult × ChannelScale
```

Where `itemMult` for physical damage = the equipped weapon's
`damage_multiplier`. The change:

```go
itemMult := weaponSpec.DamageMultiplier *
    mutations.GearEffectivenessMultiplier(attacker.Mutations)
```

At rank 4, `itemMult` becomes 0, so the attacker deals literally
zero weapon damage. Natural-weapon contributions
(`mutations.GetNaturalWeaponBonus`, unarmed via species
`naturalbash`, claws/bite/etc.) stay at full effectiveness — they
are not gear-derived.

**Site 4: `internal/combat/combat_helpers.go` — defense roll resolution**

In the best-of-all defense resolution near line 714, where
`best.margin` is computed:

```go
if attackChannel == damage.Physical {
    bonus := mutations.GetPhysicalDefenseBonus(targetChar.Mutations)
    best.margin += bonus  // applied to defender's margin
}
```

If the channel info isn't already plumbed to this site, a small
threading pass (~5-10 lines) routes it through. If it is, the
addition is one block.

**Site 5: `internal/itemvalue/delta.go` — `ItemValueDelta`**

```go
candidateRaw := ItemValue(candidateSpec, profile)
gearMul := mutations.GearEffectivenessMultiplier(char.Mutations)
candidateRaw *= gearMul

// ... in the displaced loop ...
displacedTotal += (ItemValue(dSpec, profile) +
    placementBonus(profile, dSpec, currentSlot, char)) * gearMul
```

`ItemValue` stays pure (no char dependency). `ItemValueDelta`
applies the gear multiplier at the delta layer where char is
already available. This makes chunk 2.3's equip-if-better gate
naturally: a rank-4 incorporeal mob sees all gear score 0, so
`IsUpgrade` returns false for any candidate.

**Edge case noted:** the multiplier applies to `displacedTotal`
too. An incorporeal mob who already has gear equipped (e.g.,
spawned with a weapon and then mutated) would happily "trade"
its useless weapon for a slightly better useless weapon — both
score 0 net. The mob's gear-pickup behavior becomes neutral, no
harm done. The combat-pipeline scaling kicks in regardless of
whether the mob accepts new gear.

## Mob audit & tagging

Five mob categories get `mutations: { incorporeal: 4 }` added to
their YAML during implementation. Per-category audit approach:

| Category | Audit method | Expected count |
|---|---|---|
| Wraith | grep `^name:.*[Ww]raith` in `_datafiles/world/.../mobs/**/*.yaml` | Small (1-3 variants) |
| Spectre | grep `^name:.*[Ss]pec?tre` | Small (1-3 variants) |
| Fire elemental | grep `^name:.*[Ff]ire [Ee]lemental` | Small (1-5 variants) |
| Air elemental | grep `^name:.*[Aa]ir [Ee]lemental` | Small (1-5 variants) |
| Elemental queen | mob 321 (confirmed) | 1 |

If any of these mob templates currently start with equipped gear
in their YAML, the gear stays — content authors may want it
equipped for visual flavor. Combat-pipeline scaling correctly
zeros out its effect at rank 4.

Earth and water elementals stay corporeal (rock and water have
physical substance). Per-species judgment for finer tagging is a
follow-on content concern.

## Helpfile

`_datafiles/world/dogmud/templates/help/mutations.template` —
append an entry for Incorporeal in the existing list format:

- Name + rarity tier ("very rare", max)
- Four-rank progression description (vit/wil/defense/gear loss)
- Conflict notice ("incompatible with: extra-arms, extra-legs, clawed-hands, tail, dense-muscles, hollow-bones, healing-gel")
- Lore voice consistent with existing entries (second-person, descriptive)

## context.md updates

- **`internal/mutations/context.md`** — document the two new
  effect types, the raw-level carve-out for
  `gear_effectiveness_loss` (why it skips `LevelMultiplier`,
  and the per-rank progression that results), and the three new
  helper functions.
- **`internal/itemvalue/context.md`** — note that
  `ItemValueDelta` multiplies its computed scores by the
  gear-effectiveness multiplier, so chunk 2.3's equip-if-better
  behavior naturally returns "not an upgrade" for incorporeal
  mobs at higher ranks.
- **`internal/characters/context.md`** — note that
  `Recalculate()` and the three `Get*Mitigation()` methods
  scale gear-derived contributions by the gear-effectiveness
  multiplier; non-gear contributions are unaffected.

## Testing

Per the previous chunks, tests rely on synthetic mutation /
character / item values inline — no test data dir loading
required.

| Test file | Cases |
|---|---|
| `internal/mutations/mutations_test.go` | `GetGearEffectivenessLoss` per-rank (1/2/3/4 → 0.25/0.50/0.75/1.00); clamping (multi-source > 1.0 clamps to 1.0; negative clamps to 0.0); no-effect (no incorporeal owned → 0.0). `GearEffectivenessMultiplier` inverts cleanly across ranks. `GetPhysicalDefenseBonus` per-rank via LevelMultiplier (15/22.5/30/37.5). Conflict-list verification: trying to acquire `extra-arms` when `incorporeal` is owned (and vice versa) is rejected by `HasConflict`. |
| `internal/characters/character_test.go` | `Recalculate()` correctly scales gear-derived stat contributions by gear-effectiveness multiplier; non-gear contributions unaffected. `GetPhysicalMitigation()` / `GetMagicalMitigation()` / `GetConvictionMitigation()` scale gear-derived portion only. |
| `internal/combat/damage_pipeline_test.go` (or whichever combat test file exists) | Weapon damage scales by gear-effectiveness multiplier for incorporeal attacker; at rank 4, weapon damage is 0. Natural-weapon damage (unarmed + `natural_weapon` mutation bonus) is unaffected. |
| `internal/combat/combat_helpers_test.go` | Defense margin receives `GetPhysicalDefenseBonus` for physical-channel attacks; magical and conviction attacks don't receive the bonus. |
| `internal/itemvalue/delta_test.go` | `ItemValueDelta` returns Score 0 for rank-4 incorporeal character considering any gear. Lower ranks return scores at the corresponding scaled value (rank 1 = 75% of pure-char score, etc.). |

## Smoke test

After unit tests pass:

1. `go build ./...` clean compile.
2. `go test ./...` no FAILs; SKIPs OK for fixture-dependent cases.
3. Boot the server locally and confirm clean startup past
   data-file loading. The `mutations.LoadMutationFiles()` line
   should report `loadedCount=` incremented by 1 (the new
   incorporeal.yaml).
4. Spot-check a tagged mob: spawn a fire elemental, attack it.
   Expect to miss more often than normal physical attacks (the
   defense-margin bonus is in effect). Equip the mob with a
   weapon via admin command, then watch combat — the weapon's
   damage should be zero (gear effectiveness 0 at rank 4).

## Out of scope / deferred

- **Per-rank tuning values.** Vit Δ (-10 base), wil Δ (+5 base),
  defense bonus (+15 base), gear loss (0.25/level) are starting
  values. Captured as MEMORY follow-on for a balance pass after
  we have combat data from the tagged mobs.
- **Player progression tuning.** Rarity 10 + the existing
  acquisition system is the only gate. No special trigger added
  (e.g., "near-death experience grants Incorporeal"). Could be
  a future content chunk.
- **Carry capacity scaling.** Incorporeal characters still have
  normal carry capacity. Items can ride along in the ethereal
  form, they just don't function as gear when equipped.
- **Earth/water elemental tagging.** Stay corporeal in v1. Per-
  species judgment for follow-on content pass.
- **Per-rank visual differentiation.** No icon / look variation
  by rank — just the static `visual:` field from the YAML.
- **Spell damage scaling.** Spell damage goes through unaffected
  (incorporeal beings *should* be vulnerable to magic).
  Only physical-channel attacks get the defense bonus + only
  weapon `DamageMultiplier` scaling applies.
- **Natural-weapon damage.** Unarmed, claws, bite, slam, the
  `natural_weapon` mutation bonus — none are gear-derived,
  none scale by the multiplier. A rank-4 incorporeal mob with
  claws still hits hard via the claws.
- **`requires_arms` flag interaction.** `requires_arms` only
  affects mutation acquisition filtering (which mutations can
  even be considered for a species). Incorporeal doesn't
  require arms, so the flag doesn't apply. The
  `gear_effectiveness_loss` effect doesn't strip arms — it just
  zeros out their item-equipping benefits.
- **Conflict list expansion.** Ambiguous mutations
  (`cold-blooded`, `iron-constitution`, etc.) stay
  non-conflicting in v1. Authors can add conflicts later if
  specific combos prove unbalanced.

## Roadmap touchpoints

This chunk:

- Inserts **chunk 2.2a — Incorporeal mutation** in
  `MOB_ALIVENESS_ROADMAP.md` between 2.2 and 2.3.
- Adjusts the Progress tracker total from 40 to 41 rows;
  roll-up moves to 10/41 on completion.
- Unblocks **2.3** (equip-if-better behavior) — chunk 2.3 will
  consume the gear-effectiveness multiplier naturally via
  `ItemValueDelta` scoring, with no special incorporeal skip
  path needed.
- Surfaces a follow-on memory entry for the balance-tuning pass
  on per-rank values (vit / wil / defense bonus / gear loss
  magnitudes) once combat data from tagged mobs is available.
