# Combat State — Chunk 4c: Position × Weapon Utility (Reach Model) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce a single `Reach float64` field on `ItemSpec` (meters) plus a default-by-subtype lookup, then wire a position-radius curve into the damage pipeline so long weapons degrade in grapples (sword in mount ~30% damage, spear in mount ~15% floored), while short weapons (dagger, fist, claws) stay punchy. Bladed weapons in grapples narrate as bludgeoning (attack-message vocabulary swap to Crushing) so the fiction tracks the math. No FSM changes, no new btree primitives, no sunsets. End-state: tactical weapon-swapping in grapples becomes a real choice; carrying a dagger as offhand is a viable response to a grappler; two-handed reach weapons become liabilities once a clinch lands.

**Architecture:** All new code lives in the `internal/items/` and `internal/combat/` packages. Single new field on `ItemSpec`, two new small files (`items/reach.go`, `combat/reach.go`), one new helper in the damage pipeline, three call-site updates in the combat hot path, and a vocabulary swap at the attack-message selection site. Three balance-config knobs (standing-grapple radius, ground-grapple radius, utility floor). Phase-1 YAML migration is zero changes — every existing weapon inherits its subtype default; per-item overrides land post-smoke as balance feedback comes in.

**Tech Stack:** Go 1.21+, existing damage pipeline (`internal/combat/damage_pipeline.go`), existing item subtype taxonomy (`internal/items/itemspec.go`), existing attack-message vocabulary system (`internal/items/attack_messages.go`).

**Spec:** `docs/superpowers/specs/2026-05-16-state-chunk-4c-position-weapon-utility-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP.

**Doc scope:** Comprehensive per user SOP. T8 surveys helpfiles + context.md files for stale references and identifies new doc surface needed for the reach concept; T9 applies helpfile updates (combat/attack/weapon-combat/grapple/clinch/mount overviews + identify integration + the handful of subtype-specific helpfiles where reach matters most); T10 applies context.md updates (`items/context.md` reach taxonomy reference table, `combat/context.md` integration with damage pipeline + bludgeon narration, `state/position/context.md` cross-link to the new utility curve). The helpfile work is deliberately wider than chunk 4b's because 4c is the first chunk that changes how *weapon choice* feels — players noticing "wait, my greatsword does nothing in mount" need a helpfile that explains why.

---

## File structure

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/items/itemspec.go` | MODIFY (T1) | Add `Reach float64 \`yaml:"reach,omitempty"\`` field to `ItemSpec` |
| `internal/items/reach.go` | NEW (T1) | `DefaultReachForSubtype(ItemSubType) float64` map + `ResolveReach(spec *ItemSpec) float64` (explicit reach → subtype default → 0 sentinel) + `ResolveNaturalReach(subtype ItemSubType) float64` for mob natural attacks |
| `internal/items/reach_test.go` | NEW (T1) | Unit tests: default lookup, explicit override, zero-fallthrough, every subtype has a default (regression gate against forgotten subtypes), natural-attack helper |
| `internal/items/context.md` | MODIFY (T1, T10) | Add reach field doc + default-by-subtype reference table + authoring guidance |
| `internal/combat/reach.go` | NEW (T2) | `PositionReachRadius(s position.State) float64` curve + `ReachUtility(weaponReach, posRadius float64) float64` formula + `ShouldBludgeon(weaponReach, posRadius float64) bool` predicate |
| `internal/combat/reach_test.go` | NEW (T2) | Unit tests for radius lookup (all 14 Position states), utility curve at boundary cases (fits exactly, exceeds floor, sentinel zero radius), bludgeon predicate (positive/negative/sentinel) |
| `internal/configs/balance.go` | MODIFY (T2) | Add `ReachStandingGrappleRadius` (0.5), `ReachGroundGrappleRadius` (0.3), `ReachUtilityFloor` (0.15) |
| `_datafiles/config.yaml` | MODIFY (T2) | Surface the three new knobs with their defaults + comments |
| `internal/combat/damage_pipeline.go` | MODIFY (T3) | Add `CalcReachAdjustedItemMult(weapon items.Item, attacker *characters.Character) float64` wrapper helper |
| `internal/combat/damage_pipeline_test.go` | MODIFY (T3) | Test the wrapper composes weaponSpec.DamageMultiplier × ReachUtility correctly across (standing, clinch, mount) × (dagger, sword, spear) matrix |
| `internal/hooks/NewRound_DoCombat_helpers.go` | MODIFY (T3) | Replace per-swing `weaponSpec.DamageMultiplier` reads with `CalcReachAdjustedItemMult(...)` calls. Hot path; expect ~3-5 sites |
| `internal/combat/skill_moves.go` | MODIFY (T3) | Kick variants already route through subtype context; wire reach adjustment for kick path. Grapple/trip/bash stay reach-agnostic (force-driven, not weapon-driven) |
| `internal/items/attack_messages.go` | MODIFY (T4) | (No signature changes if existing `{weapon}` token interpolation suffices — verify during T4. If pommel-specific phrasing needed, may add a small bludgeon-pattern set) |
| `internal/hooks/NewRound_DoCombat_helpers.go` | MODIFY (T4, again) | Message-selection site swaps subtype to `Crushing` when `combat.ShouldBludgeon` returns true for bladed weapons |
| `internal/combat/reach_bludgeon_test.go` | NEW (T4) | Tests that bladed weapons in grapples select Crushing messages; natural-blunt weapons keep their messages; caster weapons (already bludgeoning-flavored) keep theirs |
| `internal/state/position/position_test.go` or `internal/combat/reach_test.go` | MODIFY (T6) | Append PB-201 through PB-220 Behavior Matrix tests |
| `tools/testing/audits/2026-05-16-chunk-4c-doc-helpfile-audit.md` | NEW (T8) | Doc audit deliverable: which helpfiles need reach mentions, which context.md files need updates, what new player-facing prose is required |
| `_datafiles/world/dogmud/templates/help/*.template` | MODIFY (T9) | Per-audit helpfile updates. Likely `attack.template`, `combat.template`, `weapon-combat.template`, `grapple.template`, `identify.template`, individual weapon helpfiles (`iron-dagger.template`, `iron-short-sword.template`, `steel-longsword.template`, `lake-iron-hook-spear.template`), and `bash.template` / `kick.template` / `trip.template` / `stand.template` where weapon-utility-in-grapple is contextually relevant |
| `internal/combat/context.md` | MODIFY (T10) | Document reach utility integration, bludgeon narration rules, balance knobs |
| `internal/state/position/context.md` | MODIFY (T10) | Add cross-reference to the reach utility (Position × Weapon now matters for damage) under post-cutover status |
| `internal/characters/context.md` | MODIFY (T10) | Mention that `IsGrappling()` / position predicates are now reach-utility consumers (forward reference for future readers) |
| `COMBAT_STATE_ROADMAP.md` | MODIFY (T11) | Mark chunk 4c Done; add "Chunk 4c — Shipped" section |

---

## Task 1: Reach field on ItemSpec + default-by-subtype lookup

**Files:**
- Create: `internal/items/reach.go`
- Create: `internal/items/reach_test.go`
- Modify: `internal/items/itemspec.go` (add field)
- Modify: `internal/items/context.md` (add reach taxonomy table — incremental; full audit happens in T10)

The single source of truth for the reach taxonomy. Authors leave `reach` empty in YAML for normal items; engine falls through to the subtype default. Explicit YAML overrides for outliers.

- [ ] **Step 1: Add `Reach` field to `ItemSpec`**

In `internal/items/itemspec.go`, add a new field. Suggested placement near `DamageMultiplier` / `GrappleModifier` so reach reads as a weapon-physical-stat.

```go
// Reach is the weapon's operational reach in meters. Combat consults
// reach in grapple positions: weapons whose reach exceeds the
// position's effective radius are penalized (see
// internal/combat/reach.go). Zero is a sentinel meaning "use
// DefaultReachForSubtype lookup based on Subtype"; authors set an
// explicit non-zero value only for outliers (a particularly short
// dagger, an oversized hilt, etc.).
//
// Reach is weapon-only — arm length / species reach is intentionally
// out of scope for chunk 4c per the design spec.
Reach float64 `yaml:"reach,omitempty"`
```

- [ ] **Step 2: Create `internal/items/reach.go`**

```go
package items

// DefaultReachForSubtype returns the canonical reach (meters) for
// items of a given weapon subtype. Authors who leave the per-item
// Reach field at zero get this value via ResolveReach.
//
// Subtypes not in this map return 0 — typically non-weapon subtypes
// (BlobContent, etc.) where reach is meaningless.
//
// See docs/superpowers/specs/2026-05-16-state-chunk-4c-position-weapon-utility-design.md
// for the full taxonomy table and reasoning.
func DefaultReachForSubtype(s ItemSubType) float64 {
    switch s {
    // Natural attacks
    case Fist:       return 0.1
    case Claws:      return 0.15
    case Bite:       return 0.15
    case Sting:      return 0.2
    case Slam:       return 0.3
    case Gore:       return 0.4
    case Whipping:   return 0.5 // hand-held whip; mob tail-weapon would override

    // Melee (existing in-engine subtypes)
    case Stabbing:   return 0.3 // dagger / shiv family
    case Slashing:   return 1.0 // sword family
    case Cleaving:   return 0.9 // axe family

    // Ranged (melee-fallback reach when used as a club)
    case Shooting:   return 1.0 // bow/crossbow average; per-item overrides for compacts

    // Caster
    case Wand:       return 0.4
    case Sceptre:    return 0.6
    case Staff:      return 1.5

    default:
        return 0 // non-weapon subtype or unknown
    }
}

// ResolveReach returns the effective reach for a weapon: explicit
// per-item Reach if set, otherwise the subtype default. Zero means
// "no reach data" — treated as no penalty by the combat path
// (combat.ReachUtility handles zero gracefully).
func ResolveReach(spec *ItemSpec) float64 {
    if spec == nil {
        return 0
    }
    if spec.Reach > 0 {
        return spec.Reach
    }
    return DefaultReachForSubtype(spec.Subtype)
}

// ResolveNaturalReach returns reach for a mob's natural attack
// (claws/bite/etc.) where there's no ItemSpec to inspect. Calls
// straight through to DefaultReachForSubtype; provided as a sibling
// helper to make caller intent explicit.
func ResolveNaturalReach(subtype ItemSubType) float64 {
    return DefaultReachForSubtype(subtype)
}
```

- [ ] **Step 3: Create `internal/items/reach_test.go`**

```go
package items

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestDefaultReachForSubtype_KnownSubtypes(t *testing.T) {
    cases := map[ItemSubType]float64{
        Fist: 0.1, Claws: 0.15, Bite: 0.15, Sting: 0.2,
        Slam: 0.3, Gore: 0.4, Whipping: 0.5,
        Stabbing: 0.3, Slashing: 1.0, Cleaving: 0.9,
        Shooting: 1.0,
        Wand: 0.4, Sceptre: 0.6, Staff: 1.5,
    }
    for subtype, want := range cases {
        assert.Equal(t, want, DefaultReachForSubtype(subtype),
            "subtype %s", subtype)
    }
}

func TestDefaultReachForSubtype_UnknownReturnsZero(t *testing.T) {
    assert.Equal(t, 0.0, DefaultReachForSubtype(BlobContent))
    assert.Equal(t, 0.0, DefaultReachForSubtype(ItemSubType("nonexistent")))
}

func TestResolveReach_ExplicitOverridesSubtypeDefault(t *testing.T) {
    spec := &ItemSpec{Subtype: Slashing, Reach: 0.7} // shorter than 1.0 default
    assert.Equal(t, 0.7, ResolveReach(spec))
}

func TestResolveReach_ZeroFallsThroughToSubtypeDefault(t *testing.T) {
    spec := &ItemSpec{Subtype: Slashing, Reach: 0}
    assert.Equal(t, 1.0, ResolveReach(spec))
}

func TestResolveReach_NilSafe(t *testing.T) {
    assert.Equal(t, 0.0, ResolveReach(nil))
}

func TestResolveNaturalReach_DelegatesToDefault(t *testing.T) {
    assert.Equal(t, 0.15, ResolveNaturalReach(Claws))
    assert.Equal(t, 0.4, ResolveNaturalReach(Gore))
}
```

- [ ] **Step 4: Add reach taxonomy table to `internal/items/context.md`**

Append a new "Weapon reach (chunk 4c)" section. Include:
- One-paragraph overview: what reach means, why it exists (link to spec)
- The default-by-subtype table (matches the spec's taxonomy)
- Authoring guidance: leave empty for normal items, override for outliers, meters not abstract units
- Cross-reference to `internal/combat/reach.go` for the consumer side

This is the incremental doc update — T10 does the broader audit-driven pass across all context.md files.

- [ ] **Step 5: Build + test + commit**

```bash
go build ./...
go test ./internal/items/...
git add internal/items/itemspec.go internal/items/reach.go internal/items/reach_test.go internal/items/context.md
git commit -m "$(cat <<'EOF'
feat(items): T1 — Reach field on ItemSpec + default-by-subtype lookup

Foundation for chunk 4c (Position × Weapon Utility). Adds a single
Reach float64 (meters) field to ItemSpec, a DefaultReachForSubtype
map covering all in-engine weapon subtypes, a ResolveReach helper
that falls through to the subtype default when per-item Reach is
zero (the YAML omitted-field sentinel), and a ResolveNaturalReach
sibling for mob natural-attack call paths.

Phase-1 migration is zero YAML changes: every existing weapon
inherits its subtype default. Per-item overrides land post-smoke
when balance feedback surfaces outliers.

Authoring SOP added to internal/items/context.md.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Position radius curve + reach utility formula

**Files:**
- Create: `internal/combat/reach.go`
- Create: `internal/combat/reach_test.go`
- Modify: `internal/configs/balance.go`
- Modify: `_datafiles/config.yaml`

Combat-package logic that consumes the reach field. Pure functions; no state, no character mutation.

- [ ] **Step 1: Add balance knobs**

In `internal/configs/balance.go` (find the existing block of balance fields and add near grapple/position-related knobs):

```go
// Reach utility curve (chunk 4c). See
// internal/combat/reach.go for the formula and the design spec for
// reasoning.

// ReachStandingGrappleRadius is the effective radius (meters) at
// which a weapon stops fitting in Clinch / BackStanding positions.
// Default 0.5 — about chest-to-chest distance in a clinch.
ReachStandingGrappleRadius float64 `yaml:"reach_standing_grapple_radius"`

// ReachGroundGrappleRadius is the effective radius (meters) at
// which a weapon stops fitting in any ground grapple (Mount,
// SideControl, KneeOnBelly, NorthSouth, Crucifix, BackGround,
// HalfGuard, Guard). Default 0.3 — body-on-body distance.
ReachGroundGrappleRadius float64 `yaml:"reach_ground_grapple_radius"`

// ReachUtilityFloor caps the minimum damage multiplier from the
// reach curve. Without a floor, a pike in mount would multiply
// damage by ~0.1 — a floor at 0.15 ensures even the longest
// weapon can poke for chip damage (pommel jab, hilt-strike).
// Tunable; smoke may push this lower.
ReachUtilityFloor float64 `yaml:"reach_utility_floor"`
```

Defaults in the `DefaultBalanceConfig()` constructor: `0.5`, `0.3`, `0.15`.

In `_datafiles/config.yaml`, surface the three knobs under the existing Balance section with the same defaults + comments.

- [ ] **Step 2: Create `internal/combat/reach.go`**

```go
package combat

import (
    "github.com/GoMudEngine/GoMud/internal/configs"
    "github.com/GoMudEngine/GoMud/internal/state/position"
)

// PositionReachRadius returns the effective grapple radius (meters)
// for a Position state. Returns 0 for non-grapple states (Standing,
// Prone, Supine, Turtle); ReachUtility treats 0 as "no penalty,
// full damage."
//
// Standing-grapple states (Clinch, BackStanding) and ground-grapple
// states (Mount, SideControl, KneeOnBelly, NorthSouth, Crucifix,
// BackGround, HalfGuard, Guard) get the configured radii.
func PositionReachRadius(s position.State) float64 {
    cfg := configs.GetBalanceConfig()
    switch s {
    case position.Clinch, position.BackStanding:
        return cfg.ReachStandingGrappleRadius
    case position.Mount, position.SideControl, position.KneeOnBelly,
        position.NorthSouth, position.Crucifix, position.BackGround,
        position.HalfGuard, position.Guard:
        return cfg.ReachGroundGrappleRadius
    default:
        // Standing, Prone, Supine, Turtle — no grapple-radius penalty.
        return 0
    }
}

// ReachUtility returns the damage multiplier from the reach curve.
// Returns 1.0 (no penalty) when the position has no grapple radius
// (zero sentinel) or when the weapon's reach fits inside the
// radius. Otherwise returns radius/reach, floored at
// Balance.ReachUtilityFloor so even maximally long weapons can
// still poke.
func ReachUtility(weaponReach, posRadius float64) float64 {
    if posRadius == 0 {
        return 1.0
    }
    if weaponReach <= posRadius {
        return 1.0
    }
    cfg := configs.GetBalanceConfig()
    util := posRadius / weaponReach
    if util < cfg.ReachUtilityFloor {
        return cfg.ReachUtilityFloor
    }
    return util
}

// ShouldBludgeon reports whether the weapon's reach exceeds the
// position's grapple radius — i.e., the swing degraded to a
// pommel/hilt strike. Used by the attack-message selection site to
// swap bladed-weapon vocabulary to Crushing so the fiction tracks
// the math. Returns false for non-grapple positions (radius == 0).
func ShouldBludgeon(weaponReach, posRadius float64) bool {
    return posRadius > 0 && weaponReach > posRadius
}
```

- [ ] **Step 3: Create `internal/combat/reach_test.go`**

```go
package combat

import (
    "testing"

    "github.com/GoMudEngine/GoMud/internal/configs"
    "github.com/GoMudEngine/GoMud/internal/state/position"
    "github.com/stretchr/testify/assert"
)

func init() {
    // Defensive: ensure balance config is loaded for these tests.
    // (Test runner typically loads config; this guards against a
    // package-test in isolation.)
    if cfg := configs.GetBalanceConfig(); cfg.ReachStandingGrappleRadius == 0 {
        configs.SetBalanceConfigForTest(configs.DefaultBalanceConfig())
    }
}

func TestPositionReachRadius_AllStates(t *testing.T) {
    cfg := configs.GetBalanceConfig()
    cases := map[position.State]float64{
        position.Standing:       0,
        position.Prone:          0,
        position.Supine:         0,
        position.Turtle:         0,
        position.Clinch:         cfg.ReachStandingGrappleRadius,
        position.BackStanding:   cfg.ReachStandingGrappleRadius,
        position.Mount:          cfg.ReachGroundGrappleRadius,
        position.SideControl:    cfg.ReachGroundGrappleRadius,
        position.KneeOnBelly:    cfg.ReachGroundGrappleRadius,
        position.NorthSouth:     cfg.ReachGroundGrappleRadius,
        position.Crucifix:       cfg.ReachGroundGrappleRadius,
        position.BackGround:     cfg.ReachGroundGrappleRadius,
        position.HalfGuard:      cfg.ReachGroundGrappleRadius,
        position.Guard:          cfg.ReachGroundGrappleRadius,
    }
    for s, want := range cases {
        assert.Equal(t, want, PositionReachRadius(s), "state %s", s)
    }
}

func TestReachUtility_ZeroRadius_NoPenalty(t *testing.T) {
    assert.Equal(t, 1.0, ReachUtility(2.0, 0)) // pike in standing
}

func TestReachUtility_WeaponFits_NoPenalty(t *testing.T) {
    assert.Equal(t, 1.0, ReachUtility(0.3, 0.3)) // dagger in mount, exact
    assert.Equal(t, 1.0, ReachUtility(0.1, 0.3)) // fist in mount, under
}

func TestReachUtility_WeaponExceedsRadius_PenaltyApplies(t *testing.T) {
    // Sword (1.0m) in mount (0.3m) → 0.3
    assert.InDelta(t, 0.3, ReachUtility(1.0, 0.3), 0.001)
    // Sword in clinch (0.5m) → 0.5
    assert.InDelta(t, 0.5, ReachUtility(1.0, 0.5), 0.001)
}

func TestReachUtility_FlooredAtConfigMin(t *testing.T) {
    cfg := configs.GetBalanceConfig()
    // Pike (3.0m) in mount (0.3m) raw would be 0.1 → floored to cfg floor
    assert.Equal(t, cfg.ReachUtilityFloor, ReachUtility(3.0, 0.3))
}

func TestShouldBludgeon_PositiveAndNegative(t *testing.T) {
    assert.True(t, ShouldBludgeon(1.0, 0.3))  // sword in mount
    assert.False(t, ShouldBludgeon(0.3, 0.3)) // dagger fits exactly
    assert.False(t, ShouldBludgeon(0.1, 0.3)) // fist under
    assert.False(t, ShouldBludgeon(1.0, 0))   // not grappling
}
```

- [ ] **Step 4: Build + test + commit**

```bash
go build ./...
go test ./internal/combat/... ./internal/configs/...
git add internal/combat/reach.go internal/combat/reach_test.go internal/configs/balance.go _datafiles/config.yaml
git commit -m "feat(combat): T2 — position radius curve + reach utility formula

Adds combat.PositionReachRadius (per-state grapple radius lookup),
combat.ReachUtility (radius/reach formula, floored), and
combat.ShouldBludgeon (the bladed-vs-grapple narration predicate).
Pure functions; no state. Three new balance knobs:
ReachStandingGrappleRadius (0.5m), ReachGroundGrappleRadius (0.3m),
ReachUtilityFloor (0.15). Defaults surfaced in config.yaml.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Damage pipeline integration

**Files:**
- Modify: `internal/combat/damage_pipeline.go` (add `CalcReachAdjustedItemMult`)
- Modify: `internal/combat/damage_pipeline_test.go` (or new sibling test file)
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go` (per-swing call sites)
- Modify: `internal/combat/skill_moves.go` (kick paths if subtype-routed)

Wire the reach multiplier into the existing damage flow. The new helper `CalcReachAdjustedItemMult(weapon, attacker)` is the only place callers need to touch — internally it composes `weaponSpec.DamageMultiplier × ReachUtility(reach, radius)`.

- [ ] **Step 1: Add `CalcReachAdjustedItemMult` helper**

In `internal/combat/damage_pipeline.go`:

```go
// CalcReachAdjustedItemMult returns the weapon's effective damage
// multiplier with the reach-utility factor applied for the
// attacker's current Position. Use this everywhere CalcRawDamage's
// itemMult argument was previously fed weaponSpec.DamageMultiplier
// directly.
//
// For attackers in non-grapple positions, ReachUtility returns 1.0
// and the result equals weaponSpec.DamageMultiplier — zero behavior
// change. For attackers in grapples, long weapons (reach exceeding
// the position's radius) take a multiplicative penalty.
//
// Natural-attack call paths (mob fist/claws/bite) construct a
// pseudo-spec or call ReachUtility(items.ResolveNaturalReach(subtype), ...)
// directly — this helper assumes an Item with a real Spec.
func CalcReachAdjustedItemMult(
    weapon items.Item,
    attacker *characters.Character,
) float64 {
    spec := weapon.GetSpec()
    if spec == nil {
        return 1.0
    }
    baseMult := spec.DamageMultiplier
    if baseMult == 0 {
        baseMult = 1.0 // weapons without explicit multiplier default to 1×
    }
    if attacker == nil || attacker.Position == nil {
        return baseMult
    }
    reach := items.ResolveReach(spec)
    posRadius := PositionReachRadius(attacker.Position.State())
    return baseMult * ReachUtility(reach, posRadius)
}
```

- [ ] **Step 2: Test the wrapper composes correctly**

Add tests to `internal/combat/damage_pipeline_test.go` (or new `internal/combat/damage_pipeline_reach_test.go`):

```go
func TestCalcReachAdjustedItemMult_StandingNoPenalty(t *testing.T) {
    attacker := characters.New() // Position defaults to Standing
    weapon := makeTestWeapon(items.Slashing, 1.2 /*dmgMult*/, 1.0 /*reach*/)
    got := CalcReachAdjustedItemMult(weapon, attacker)
    assert.InDelta(t, 1.2, got, 0.001) // no grapple → utility 1.0
}

func TestCalcReachAdjustedItemMult_SwordInMount(t *testing.T) {
    attacker := characters.New()
    forceMount(attacker) // helper: transition FSM to Mount
    weapon := makeTestWeapon(items.Slashing, 1.0, 1.0) // sword, reach 1.0
    got := CalcReachAdjustedItemMult(weapon, attacker)
    // utility = 0.3 / 1.0 = 0.3, base 1.0 → 0.3
    assert.InDelta(t, 0.3, got, 0.001)
}

func TestCalcReachAdjustedItemMult_DaggerInMount(t *testing.T) {
    attacker := characters.New()
    forceMount(attacker)
    weapon := makeTestWeapon(items.Stabbing, 0.8, 0) // dagger, reach 0 → default 0.3
    got := CalcReachAdjustedItemMult(weapon, attacker)
    assert.InDelta(t, 0.8, got, 0.001) // reach fits → no penalty
}

func TestCalcReachAdjustedItemMult_PikeInMountFloors(t *testing.T) {
    attacker := characters.New()
    forceMount(attacker)
    weapon := makeTestWeapon(items.Stabbing, 1.0, 3.0) // pike, explicit reach
    got := CalcReachAdjustedItemMult(weapon, attacker)
    // raw utility = 0.3/3.0 = 0.1, floored to 0.15
    assert.InDelta(t, 0.15, got, 0.001)
}
```

(Use the existing `setCombatPositionParallel`-style helper or transition the FSM directly.)

- [ ] **Step 3: Wire `CalcReachAdjustedItemMult` into the per-swing damage path**

Audit `internal/hooks/NewRound_DoCombat_helpers.go` for every site that reads `weaponSpec.DamageMultiplier`. Each becomes `combat.CalcReachAdjustedItemMult(weaponItem, attacker)`. Expect 3-5 sites.

For natural-attack paths (mob unarmed, no weapon Item), compose directly:
```go
naturalReach := items.ResolveNaturalReach(subtype)
posRadius := combat.PositionReachRadius(attacker.Position.State())
mult := combat.ReachUtility(naturalReach, posRadius)
```

Mob fist/claws/bite all stay punchy in grapples (reach ≤ ground-grapple radius), so this is mostly defensive — no mob damage actually changes for non-weapon attacks. Good — sanity-checks the model.

- [ ] **Step 4: Wire into skill moves where appropriate**

`internal/combat/skill_moves.go:ExecuteSkillMove` is the entry for kick/grapple/trip/bash. Decision per move:
- **Kick variants** (standard / stomp / knee) — use reach-adjustment for the weapon if armed. Knee fires in grapples, where any wielded weapon is constrained anyway. Likely yes for safety.
- **Grapple entry** — force-driven, weapon doesn't apply. Skip reach.
- **Trip / bash** — same as grapple, force-driven. Skip reach.

Implementation: route the `DamagePercent` calculation through `CalcReachAdjustedItemMult` only for kick path. Document the decision in skill_moves.go comments.

- [ ] **Step 5: Build + test + commit**

```bash
go build ./...
go test ./internal/combat/... ./internal/hooks/...
git add internal/combat/damage_pipeline.go internal/combat/damage_pipeline_*test.go internal/hooks/NewRound_DoCombat_helpers.go internal/combat/skill_moves.go
git commit -m "feat(combat): T3 — wire reach utility into per-swing damage pipeline

Adds CalcReachAdjustedItemMult helper composing weapon damage_multiplier
with the position-radius reach curve. Every per-swing damage site in
NewRound_DoCombat_helpers.go now reads through this helper instead of
the raw weaponSpec.DamageMultiplier. Kick variants pass through skill_moves
similarly; grapple/trip/bash stay reach-agnostic (force-driven moves).

Behavior change: long melee weapons (reach > position radius) now
multiply damage by radius/reach (floored) when attacker is grappled.
Standing engagements unchanged.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Attack-message vocabulary swap (bludgeon narration)

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go` (message-selection site)
- Create: `internal/combat/reach_bludgeon_test.go`
- Possibly modify: `internal/items/attack_messages.go` (only if existing `{weapon}` interpolation doesn't suffice)

When `ShouldBludgeon(reach, radius)` is true AND the weapon is bladed (Slashing/Cleaving/Stabbing) or ranged-as-melee (Shooting), narrate the strike with the Crushing vocabulary. Damage math is already adjusted (T3); this swap only affects display strings.

- [ ] **Step 1: Inspect the existing attack-message vocabulary**

```bash
git --no-pager grep -n "GetAttackMessage" internal/
```

Confirm `GetAttackMessage(subtype, pctDamage)` returns an `AttackOptions` with first/third-person message templates and that the templates use `{weapon}` / `{target}` tokens (not subtype-baked strings). Read `_datafiles/world/dogmud/templates/.../attackmessages/*.yaml` or the relevant data location to verify Crushing messages render correctly when the weapon is actually a sword. If the Crushing templates already say "you slam X" / "you crush X", they'll work with a sword name interpolated.

If Crushing templates assume a specifically-blunt weapon ("you crush the goblin with your mace"), they may sound wrong with "sword" — in that case add a new vocabulary pattern (e.g., `BludgeonImprovised` or `PommelStrike`) that uses verbs/phrasings appropriate for "weapon used as bludgeon" rather than "intentional blunt weapon." Document the decision in this task's commit.

- [ ] **Step 2: Add the message-swap at the call site**

In `internal/hooks/NewRound_DoCombat_helpers.go`, find the existing `items.GetAttackMessage(...)` call(s) and wrap the subtype argument:

```go
displaySubtype := weaponSpec.Subtype
posRadius := combat.PositionReachRadius(attacker.Position.State())
weaponReach := items.ResolveReach(weaponSpec)
if combat.ShouldBludgeon(weaponReach, posRadius) {
    // Bladed/ranged weapons in close-quarters grapples narrate as
    // pommel/hilt strikes. Natural-blunt subtypes (Slam, Gore,
    // Whipping, Fist) keep their existing vocabulary — they're
    // already blunt. Caster weapons (Wand, Sceptre, Staff) keep
    // theirs — their templates already read as bludgeoning.
    switch displaySubtype {
    case items.Slashing, items.Cleaving, items.Stabbing, items.Shooting:
        displaySubtype = items.Crushing
    }
}
msg := items.GetAttackMessage(displaySubtype, pctDmg)
```

- [ ] **Step 3: Create `internal/combat/reach_bludgeon_test.go`**

Tests that verify the swap fires/doesn't fire correctly. Examples:

```go
func TestBludgeonSwap_BladedInMount_SelectsCrushing(t *testing.T) {
    // Set up attacker in Mount with a sword wielded; verify the
    // selected message-subtype is Crushing.
}

func TestBludgeonSwap_DaggerFits_KeepsStabbing(t *testing.T) {
    // Dagger reach 0.3 = mount radius 0.3 → no swap.
}

func TestBludgeonSwap_FistKeepsFist(t *testing.T) {
    // Fist reach 0.1 < 0.3 → no swap regardless.
}

func TestBludgeonSwap_CasterStaffInMount_KeepsStaff(t *testing.T) {
    // Staff is in the "natural-blunt-or-caster" exemption — though
    // technically Cleaving/Slashing it isn't a real bladed weapon;
    // we want to keep the staff's own message vocabulary. Verify
    // the switch handles caster subtypes correctly. (May surface a
    // refinement to the switch above.)
}
```

The caster-staff case may push a refinement on the subtype switch — caster subtypes (Wand, Sceptre, Staff) shouldn't swap because their own vocabulary is already appropriate. Verify and refine during this task.

- [ ] **Step 4: Build + test + commit**

```bash
go build ./...
go test ./internal/combat/... ./internal/hooks/... ./internal/items/...
git add internal/hooks/NewRound_DoCombat_helpers.go internal/combat/reach_bludgeon_test.go [internal/items/attack_messages.go if touched]
git commit -m "feat(combat): T4 — bludgeon narration for bladed weapons in grapples

When ShouldBludgeon fires (weapon reach > position radius), the
attack-message subtype swaps to Crushing for bladed (Slashing /
Cleaving / Stabbing) and Shooting weapons. Natural-blunt
(Slam/Gore/Whipping/Fist) and caster weapons (Wand/Sceptre/Staff)
keep their existing vocabulary. Damage math unchanged from T3;
this is messaging only.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Behavior Matrix tests (PB-201 through PB-220)

**Files:**
- Modify: `internal/combat/reach_test.go` (append) OR new `internal/combat/reach_matrix_test.go`

The 20-case behavior matrix from the design spec, executed as table-driven tests. Some overlap T2's unit tests; consolidate into the matrix file for traceability.

- [ ] **Step 1: Author the matrix tests**

```go
func TestBehaviorMatrix_Reach(t *testing.T) {
    setupBalanceForTest(t)
    tests := []struct {
        id    string
        pos   position.State
        reach float64
        wantMult     float64
        wantBludgeon bool
    }{
        {"PB-201 fist in mount",       position.Mount,        0.1, 1.00, false},
        {"PB-202 dagger in mount",     position.Mount,        0.3, 1.00, false},
        {"PB-203 sword in mount",      position.Mount,        1.0, 0.30, true},
        {"PB-204 spear in mount",      position.Mount,        2.0, 0.15, true}, // floored
        {"PB-205 sword standing",      position.Standing,     1.0, 1.00, false},
        {"PB-206 sword vs prone",      position.Standing,     1.0, 1.00, false}, // attacker pos drives
        {"PB-207 sword in clinch",     position.Clinch,       1.0, 0.50, true},
        {"PB-208 wand in clinch",      position.Clinch,       0.4, 1.00, false},
        {"PB-209 wand in mount",       position.Mount,        0.4, 0.75, true},
        {"PB-210 staff in mount",      position.Mount,        1.5, 0.20, true},
        {"PB-211 greatsword halfguard",position.HalfGuard,    1.5, 0.20, true},
        {"PB-212 pike in clinch",      position.Clinch,       3.0, 0.17, true},
        {"PB-219 pike in mount floor", position.Mount,        3.0, 0.15, true},
        {"PB-220 turtle no penalty",   position.Turtle,       1.5, 1.00, false},
    }
    for _, tc := range tests {
        t.Run(tc.id, func(t *testing.T) {
            radius := PositionReachRadius(tc.pos)
            mult := ReachUtility(tc.reach, radius)
            bld := ShouldBludgeon(tc.reach, radius)
            assert.InDelta(t, tc.wantMult, mult, 0.005, "mult")
            assert.Equal(t, tc.wantBludgeon, bld, "bludgeon")
        })
    }
}

// PB-213 / PB-214 — ResolveReach behavior (already covered in
// internal/items/reach_test.go — reference the cross-link here).

// PB-215 / PB-216 / PB-217 / PB-218 — bludgeon-narration matrix
// covered in internal/combat/reach_bludgeon_test.go.
```

- [ ] **Step 2: Build + test + commit**

```bash
go test ./internal/combat/... -run TestBehaviorMatrix_Reach
git add internal/combat/reach_test.go [or reach_matrix_test.go]
git commit -m "test(combat): T5 — Behavior Matrix PB-201..PB-220 (reach)"
```

---

## Task 6: Build / test / smoke validation

**Files:** (verification only — may produce a smoke notes file)

- [ ] **Step 1: Full build**

```bash
go build ./...
```

- [ ] **Step 2: Full test suite**

```bash
go test ./... -count=1 2>&1 | grep -E "^FAIL"
```

Expected: zero FAILs.

- [ ] **Step 3: Reach tests targeted run**

```bash
go test ./internal/items/ -run TestDefaultReach -v -count=1
go test ./internal/combat/ -run "TestReach|TestPositionReachRadius|TestShouldBludgeon|TestBehaviorMatrix_Reach|TestBludgeon" -v -count=1
```

All PASS.

- [ ] **Step 4: Chunks 0-4b regression**

```bash
go test ./internal/state/... ./internal/characters/... ./internal/hooks/... ./internal/combat/... ./internal/usercommands/... ./internal/mobcommands/... -count=1 2>&1 | grep -E "^(ok|FAIL)"
```

- [ ] **Step 5: Server boot smoke**

```bash
go build -o /tmp/dogmud-chunk4c.exe . && /tmp/dogmud-chunk4c.exe > /tmp/dogmud-chunk4c.log 2>&1 &
PID=$!
until grep -qE "Server Ready|panic|FATAL" /tmp/dogmud-chunk4c.log; do sleep 3; done
grep -E "Server Ready|panic|FATAL|loadedCount" /tmp/dogmud-chunk4c.log | tail -25
kill -9 $PID 2>/dev/null
rm -f /tmp/dogmud-chunk4c.exe /tmp/dogmud-chunk4c.log
```

Expected: `Server Ready`, no panic, all data files load.

- [ ] **Step 6: User in-game smoke (deferred to user session)**

Per the chunk 0-4b pattern. User runs rich scenarios:
- Wield greatsword, grapple a mob, observe "you slam the steel greatsword's pommel..." messages
- Wield dagger offhand, grapple a mob, dagger swings stay full damage with stabbing narration
- Swap from spear to dagger mid-fight (drop spear, draw dagger), observe damage jump
- Caster character: equip wand, equip caster-staff, take a hit in clinch, observe wand damage stays high vs staff drops

DO NOT commit anything in T6.

---

## Task 7: Reserved (advance T-numbering parity with chunk 4b)

(4b's plan used T7 for messaging; 4c has no analogue. Skipped to keep the doc-audit / doc-update / roadmap-closeout tasks at T8 / T9 / T10 / T11 for the SOP user feedback memory "T22-style audit, T23-style updates" pattern. Mark this as **completed-by-design** when executing.)

---

## Task 8: Documentation audit (T22-style)

**Files:**
- Create: `tools/testing/audits/2026-05-16-chunk-4c-doc-helpfile-audit.md`

Comprehensive survey of doc surface affected by 4c. **Wider helpfile scope than chunks 4a/4b** because 4c is the first chunk that changes how *weapon choice* feels — players noticing "wait, my greatsword does nothing in mount" need a helpfile that explains why.

- [ ] **Step 1: Helpfile grep — combat / weapon / grapple surface**

```bash
ls _datafiles/world/dogmud/templates/help/ | grep -iE "weapon|attack|combat|grapple|clinch|mount|stand|kick|bash|trip|identify|equip|wield"
git --no-pager grep -nlE "grapple|reach|weapon.*long|weapon.*short" -- _datafiles/world/dogmud/templates/help/
```

For each hit: classify DELETE / UPDATE / KEEP-AS-IS. Most will be UPDATE (need a sentence about reach) or KEEP (unrelated).

- [ ] **Step 2: Individual weapon helpfile grep**

```bash
ls _datafiles/world/dogmud/templates/help/ | grep -iE "dagger|sword|axe|spear|staff|polearm|hammer|mace|bow|crossbow|wand|sceptre"
```

For each specific weapon helpfile: should it mention reach? E.g., the spear helpfile probably wants "Reach: 2.0m. Becomes a poor choice in grapples — consider drawing a dagger instead." The dagger helpfile probably wants "Reach: 0.3m. Stays fully effective in any grapple — your weapon of choice in close quarters."

- [ ] **Step 3: context.md grep**

```bash
git --no-pager grep -nlE "damage_multiplier|grapple|position.*weapon|weapon.*position|reach" -- '**/context.md'
```

Files almost certainly affected:
- `internal/items/context.md` — needs the full reach taxonomy + authoring SOP (incrementally added in T1; verify completeness in T8 audit)
- `internal/combat/context.md` — needs section on reach pipeline integration + bludgeon narration + balance knobs
- `internal/state/position/context.md` — short cross-reference to reach utility
- `internal/characters/context.md` — short mention that position predicates now drive damage (forward reference)
- `internal/hooks/context.md` — short mention of the reach-adjusted multiplier site in NewRound_DoCombat_helpers
- `internal/configs/context.md` (if exists) — three new knobs

- [ ] **Step 4: Identify NEW player-facing surface**

- "reach" stat — does it appear in `identify`? In the future, should it? (Decision: 4c doesn't add it; helpfile mentions reach as a stat to be aware of. Stat display added when smoke shows players confused.)
- Combat round messages — bludgeon narration is the new visible surface; verify the prose feels good in smoke
- Per-weapon helpfiles — first time we've had a per-weapon-relevant mechanic that requires per-weapon doc updates. Establishes a precedent for future per-weapon mechanics

- [ ] **Step 5: Write the audit**

Path: `tools/testing/audits/2026-05-16-chunk-4c-doc-helpfile-audit.md`. Follow chunk 4b's audit structure:
- Header / metadata
- Files reviewed summary table (helpfiles + context.md split into sections)
- Per-helpfile findings (file path → DELETE / UPDATE / KEEP-AS-IS verdict per hit + suggested copy where appropriate)
- Per-context.md findings (same structure)
- "New documentation surface" section: identifies whether `identify` integration is in/out of scope, whether a top-level `help reach` file should be authored, whether the bludgeon-narration deserves its own helpfile
- Per-weapon helpfile inventory (which existing weapons get explicit reach mentions in T9, which inherit silently)
- Summary counts

- [ ] **Step 6: Commit**

```bash
git add tools/testing/audits/2026-05-16-chunk-4c-doc-helpfile-audit.md
git commit -m "docs(audits): chunk-4c doc + helpfile audit

Survey of helpfile and context.md surface affected by the reach
model. Identifies UPDATE / KEEP-AS-IS verdicts for each combat /
weapon / grapple helpfile, picks per-weapon helpfiles for explicit
reach mentions, and flags whether a dedicated 'help reach' file is
worth authoring. Action list feeds T9 (helpfile updates) and T10
(context.md updates).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: Helpfile updates

**Files:** (per T8 audit; typical scope ~10-15 helpfiles)

Walk the T8 audit's helpfile action list and apply each UPDATE. The list will likely include:

- `attack.template` — short paragraph: "Your weapon's effective reach matters in close quarters. Long weapons (spears, polearms, two-handed swords) become awkward when you're grappling — they get used as clubs at reduced effectiveness. Short weapons (daggers, fists, claws) stay fully effective. See `help reach` for details."
- `combat.template` — same line, contextualized for the combat overview
- `weapon-combat.template` — mention reach as a stat that interacts with grappling
- `unarmed-combat.template` — note that fists/claws stay effective in any grapple
- `grapple.template` — note that long weapons become poor choices when grappled
- `identify.template` — note that weapon stats include reach (forward-looking — even though `identify` may not surface reach in 4c, document the existence)
- `bash.template` / `kick.template` / `trip.template` / `stand.template` — short context-appropriate mentions where relevant
- Per-weapon templates (`iron-dagger.template`, `iron-short-sword.template`, `steel-longsword.template`, `lake-iron-hook-spear.template`) — one-line reach mention with the per-weapon reach value
- Potentially new: `reach.template` (top-level) — full explainer if T8 decides it's worth its own helpfile

**Voice conventions** (per project SOP):
- No hard numbers in player-facing text (use descriptive language) — exception: helpfiles describing mechanical stats CAN use numbers if it aids understanding. Per-weapon reach values are mechanical info, ok to display.
- 80-char line wrap
- Existing helpfile tone is friendly-explanatory; match it
- Use existing ansi color conventions (see other helpfiles for examples)

- [ ] **Step 1: Apply audit-recommended copy per helpfile**

- [ ] **Step 2: If a top-level `reach.template` was recommended, author it**

```
.: Help for reach

Weapons in DOGMud have a "reach" stat measured in meters. Reach
determines how well a weapon performs when you're grappling — long
weapons become awkward in close quarters; short weapons stay
useful.

[... etc. — full explainer with table, examples, tactical advice ...]
```

- [ ] **Step 3: Spot-check each updated helpfile renders cleanly**

```bash
# Verify no ansi-tag breakage (extra closing tags, missing fg attrs, etc.)
git --no-pager grep -lE "<ansi[^>]*>[^<]*<ansi" _datafiles/world/dogmud/templates/help/ | head -5
```

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/templates/help/
git commit -m "docs(helpfiles): chunk-4c — reach + weapon utility in grapples

Per audit findings. Updates [N] helpfiles: combat / attack / weapon-
combat / unarmed-combat / grapple / [bash / kick / trip / stand /
identify per applicability], plus per-weapon templates for the
representative dagger / shortsword / longsword / spear that exist
in the world. [New reach.template if authored.] Voice matches the
existing friendly-explanatory tone; 80-char wrap.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 10: Context.md updates

**Files:** (per T8 audit; typical scope 5-7 context.md files)

Walk the T8 audit's context.md action list and apply each UPDATE.

- [ ] **Step 1: `internal/items/context.md`**

The reach taxonomy table was added incrementally in T1. T10 verifies completeness against the audit's findings:
- All in-engine subtypes listed
- Authoring SOP is complete (when to override, units, where to look for the consumer side)
- Cross-link to `internal/combat/reach.go` and the spec

- [ ] **Step 2: `internal/combat/context.md`**

Add a "Weapon reach utility (chunk 4c)" section. Cover:
- One-paragraph overview: what the reach pipeline does
- The three formulas (`PositionReachRadius`, `ReachUtility`, `ShouldBludgeon`) with signatures
- `CalcReachAdjustedItemMult` — the pipeline-integration helper, what calls it
- Bludgeon narration rules: when the subtype swap fires, which subtypes are exempt (natural-blunt, caster), the cosmetic-only nature of the swap
- Balance knobs (`ReachStandingGrappleRadius`, `ReachGroundGrappleRadius`, `ReachUtilityFloor`) with defaults

- [ ] **Step 3: `internal/state/position/context.md`**

Add a short cross-reference under "Status" or a new "Consumers" subsection: "Position × Weapon-Reach: chunk 4c (`internal/combat/reach.go`) reads `IsGrappling()` / `State()` to compute a damage multiplier; see combat package context for details."

- [ ] **Step 4: `internal/characters/context.md`**

Under the position predicates section, add one line: "Position predicates also drive the chunk-4c reach utility — `IsGrappling()` + the per-state radius determine whether long weapons take a damage penalty."

- [ ] **Step 5: `internal/hooks/context.md`**

Under the combat-round flow description, add a one-line mention that `NewRound_DoCombat_helpers.go` now reads through `combat.CalcReachAdjustedItemMult` for per-swing damage and swaps attack-message subtype to Crushing when `combat.ShouldBludgeon` fires.

- [ ] **Step 6: `internal/configs/context.md` (if exists)**

Document the three new balance knobs.

- [ ] **Step 7: Build verify (defensive — doc-only changes shouldn't break it)**

```bash
go build ./...
```

- [ ] **Step 8: Commit**

```bash
git add internal/items/context.md internal/combat/context.md internal/state/position/context.md internal/characters/context.md internal/hooks/context.md [internal/configs/context.md if touched]
git commit -m "docs(position): chunk-4c context.md updates

Applies T8 audit findings. items/context.md verified complete; combat/
context.md gets new 'Weapon reach utility' section covering formulas,
pipeline-integration helper, bludgeon narration rules, and balance
knobs; state/position + characters + hooks context.md files get short
cross-references back to the reach consumer.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 11: Roadmap closeout

**Files:**
- Modify: `COMBAT_STATE_ROADMAP.md`

- [ ] **Step 1: Mark chunk 4c Done in the progress table**

Update the 4c row (currently "Not started"): change to "Done (YYYY-MM-DD)" with one-line summary.

- [ ] **Step 2: Add a "Chunk 4c — Shipped" section**

After the existing "Chunk 4b — Shipped" section. Sections to include (mirror 4b's structure):
- Goal restate (one paragraph)
- What shipped: reach field, default-by-subtype, position-radius curve, utility formula, pipeline integration, bludgeon narration, balance knobs
- Behavior Matrix outcome (PB-201..PB-220 tally)
- Doc work: helpfile updates + context.md updates (T8/T9/T10)
- "What's next" pointing to chunk 4d (Submission rework)

- [ ] **Step 3: Update tail**

Find the "Aliveness work stays paused for chunks 4d-6" line (or equivalent) and update sub-chunk numbering. Update the "Next: chunk 4c" line to "Next: chunk 4d — Submission rework (opportunistic submissions gated on Position + ControlLevel)."

- [ ] **Step 4: Commit**

```bash
git add COMBAT_STATE_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(roadmap): chunk 4c (Position × Weapon Utility) Done

Reach model shipped. Single Reach float64 (meters) field on
ItemSpec with default-by-subtype lookup; position-radius curve
(standing-grapple 0.5m, ground-grapple 0.3m, other 0 sentinel);
ReachUtility formula floored at 0.15. CalcReachAdjustedItemMult
wired into per-swing damage path. Bladed weapons in grapples
narrate as bludgeoning (attack-message subtype swap to Crushing)
so fiction tracks the math. Three new balance knobs in config.yaml.

Per-weapon doc updates: combat / attack / weapon-combat / unarmed-
combat / grapple / [bash / kick / trip / stand] helpfiles updated;
representative per-weapon templates (dagger / shortsword / longsword
/ spear) gained reach mentions. Six context.md files updated.

Behavior Matrix PB-201..PB-220 PASS. Chunks 0-4b regression clean.
Server boots cleanly.

Phase-1 YAML migration zero — every existing weapon inherits its
subtype default. Per-item overrides will land post-smoke as
balance feedback surfaces outliers.

Next: chunk 4d — Submission rework.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Spec coverage check

| Spec section | Task(s) |
|---|---|
| Reach field on ItemSpec | T1 |
| DefaultReachForSubtype + ResolveReach + ResolveNaturalReach | T1 |
| Reach taxonomy table (authoring guidance) | T1 (+ T10 verify) |
| PositionReachRadius curve | T2 |
| ReachUtility formula | T2 |
| ShouldBludgeon predicate | T2 |
| Balance knobs | T2 |
| CalcReachAdjustedItemMult pipeline helper | T3 |
| Per-swing damage site updates | T3 |
| Kick-variant integration | T3 |
| Bludgeon vocabulary swap | T4 |
| Behavior Matrix PB-201..PB-220 | T5 |
| Build/test/smoke | T6 |
| Helpfile audit | T8 |
| Helpfile updates | T9 |
| Context.md updates | T10 |
| Roadmap closeout | T11 |

All spec sections covered.

## Known followups (out of chunk 4c)

- Defense-side reach penalty (long parry weapons should be hard to use defensively from mount) — 4f flavor pass.
- Per-grapple-state radius overrides (BackGround tighter than Mount, etc.) — 4f flavor pass.
- Compound reach (arm length + weapon haft) for species reach variance — future spec.
- Reach affecting grapple-entry rolls (long-weapon wielder harder to clinch) — 4d/4e candidate.
- Player-facing reach UI (identify integration, prompt token) — post-smoke if confusion surfaces.
- Per-weapon reach overrides for the ~50 existing weapons — post-smoke balance pass.
- Casting-while-grappled penalty (caster-staff in mount degrades spell damage too) — future 4f.
- "Drop weapon to grapple" mechanic (auto or manual) — out of scope.
- Pommel-strike / hilt-bash as an explicit skill move (currently just damage degradation) — out of scope.
