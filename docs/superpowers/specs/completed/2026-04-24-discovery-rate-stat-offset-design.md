# Discovery Rate Stat Offset — Design

## Goal

Modify the spell + recipe discovery formula so that a character's
**Perception** and the **relevant skill** (spellcasting, manifestation,
or the specific crafting skill) partially offset the decay that makes
discovery slow down as known count grows. High-investment characters
should keep discovering new content at a reasonable rate into the
late game without the base-chance having to be set so high that
newbies are flooded with discoveries.

## Context

### Current formula (unchanged inputs)

```
chance = BaseChance / (1 + known × BaseDecay)
```

Defaults in `_datafiles/config.yaml`:
- `SpellDiscoveryBaseChance: 5.0` / `SpellDiscoveryDecayRate: 0.1`
- `RecipeDiscoveryBaseChance: 10.0` / `RecipeDiscoveryDecayRate: 0.1`

At 20 known, effective chance drops to 1.67% (spells) / 3.33% (recipes).

### Feedback that prompted this

Duard's VC feedback from 2026-04-17: discovery feels too slow once a
player knows more than a handful of spells or recipes.

### Why additive doesn't work

A simple `offset = (Per-100)/PerScale + skill/SkillScale` model gives
casual players with moderate Per and moderate skill noticeable help,
but then gets capped quickly and veterans/GMs plateau. User wants
**both** stats to matter and the combo to be what unlocks the
full benefit.

## Proposed Formula

Independent-probability combination:

```
perContrib   = max(0, min(1, (perception - 100) / PerceptionScale))
skillContrib = max(0, min(1, skill / SkillScale))
offset       = min(MaxOffset, 1 - (1 - perContrib) × (1 - skillContrib))
effDecay     = BaseDecay × (1 - offset)
chance       = BaseChance / (1 + known × effDecay)
```

The `1 - (1-a)(1-b)` shape means Per and skill each independently chip
away at decay; the combination is stronger than either alone but
never exceeds 1.0 (pre-cap). The `MaxOffset` cap then keeps the
bottom floor of `effDecay` at a non-zero minimum so discovery never
runs away.

## New Config Knobs (`Balance`)

| Knob | Default | Purpose |
|------|---------|---------|
| `DiscoveryPerceptionScale` | 200 | Raw Per-contribution reaches 1.0 at Per=300 (100 + PerScale). Per=150 → 0.25, Per=200 → 0.50. |
| `DiscoverySkillScale` | 100 | Raw skill-contribution reaches 1.0 at rank 100. Softcap 50 → 0.50, GM 75 → 0.75. |
| `DiscoveryMaxDecayOffset` | 0.8 | Hard ceiling on total offset. effDecay never drops below 20% of BaseDecay. |

All three knobs are **shared** between spell and recipe discovery —
they describe character-level effectiveness, not content-specific
tuning. Base chance and base decay remain per-system (`Spell*` /
`Recipe*`) as they are today.

## Shape Verification

### Realistic scenarios — spells (base 5%, decay 0.1)

| Profile  | Per | Skill | Known | Offset | effDecay | Chance | vs baseline |
|----------|-----|-------|-------|--------|----------|--------|-------------|
| Newbie   | 100 |   0   |   3   | 0.000  | 0.100    | 3.85%  | = |
| Early    | 110 |  10   |   8   | 0.145  | 0.085    | 2.97%  | +6%  |
| Mid      | 130 |  25   |  12   | 0.362  | 0.064    | 2.83%  | +31% |
| Late     | 160 |  50   |  18   | 0.650  | 0.035    | 3.07%  | +42% |
| GM       | 200 | 100   |  20   | 0.800* | 0.020    | 3.57%  | +79% |

### Realistic scenarios — recipes (base 10%, decay 0.1)

| Profile  | Per | Skill | Known | Offset | effDecay | Chance | vs baseline |
|----------|-----|-------|-------|--------|----------|--------|-------------|
| Newbie   | 100 |   0   |   3   | 0.000  | 0.100    | 7.69%  | = |
| Early    | 110 |  10   |   8   | 0.145  | 0.085    | 5.94%  | +6%  |
| Mid      | 130 |  25   |  12   | 0.362  | 0.064    | 5.66%  | +30% |
| Late     | 160 |  50   |  18   | 0.650  | 0.035    | 6.13%  | +42% |
| GM       | 200 | 100   |  20   | 0.800* | 0.020    | 7.14%  | +79% |

`*` offset hits MaxOffset cap.

Pure-Per (Per=200, skill=0) and pure-skill (Per=100, skill=100)
builds each reach offset 0.5 independently — meaningful but not
maxed. The combo is what hits the 0.8 cap.

## Skill Mapping

Each call site knows which skill to pass:

| Content | Relevant skill |
|---------|---------------|
| Player spell cast — traditional schools (elemental, enhancement, mental, vital) | `Spellcasting` |
| Player spell cast — manifestation school | `Manifestation` |
| Mob spell cast — traditional schools | `Spellcasting` |
| Mob spell cast — manifestation school | `Manifestation` |
| Recipe craft | `recipe.Skill` (the skill of the recipe being crafted — blacksmithing, alchemy, tailoring, cooking, jewelcrafting, enchanting) |

Each call site already reads the relevant skill level (for the
eligibility filter that picks which spell/recipe to award). The
offset formula reuses that same value.

## Implementation

### New helper

`internal/configs/discovery.go`:

```go
package configs

// DiscoveryParams bundles the inputs for DiscoveryChance.
type DiscoveryParams struct {
    Base        float64 // base chance (e.g. 5.0 = 5%)
    Decay       float64 // per-known decay rate
    Known       int     // count of already-known spells/recipes
    Perception  int     // character Perception stat
    Skill       int     // relevant skill rank
}

// DiscoveryChance computes the discovery chance % with Per + skill offset
// reducing effective decay. Returns a value in [0, Base].
func DiscoveryChance(p DiscoveryParams) float64 {
    bal := GetBalanceConfig()
    perScale := float64(bal.DiscoveryPerceptionScale)
    skillScale := float64(bal.DiscoverySkillScale)
    maxOffset := float64(bal.DiscoveryMaxDecayOffset)

    perContrib := (float64(p.Perception) - 100) / perScale
    if perContrib < 0 { perContrib = 0 }
    if perContrib > 1 { perContrib = 1 }

    skillContrib := float64(p.Skill) / skillScale
    if skillContrib < 0 { skillContrib = 0 }
    if skillContrib > 1 { skillContrib = 1 }

    offset := 1 - (1-perContrib)*(1-skillContrib)
    if offset > maxOffset { offset = maxOffset }

    effDecay := p.Decay * (1 - offset)
    return p.Base / (1 + float64(p.Known)*effDecay)
}
```

### Config struct additions

`internal/configs/config.balance.go`:

```go
// Add to Balance struct, in a "Discovery" cluster near the existing
// RecipeDiscovery* / SpellDiscovery* fields:
DiscoveryPerceptionScale ConfigFloat `yaml:"DiscoveryPerceptionScale"` // Raw Per contribution reaches 1.0 at (Per - 100) / this (default 200)
DiscoverySkillScale      ConfigFloat `yaml:"DiscoverySkillScale"`      // Raw skill contribution reaches 1.0 at rank / this (default 100)
DiscoveryMaxDecayOffset  ConfigFloat `yaml:"DiscoveryMaxDecayOffset"`  // Hard ceiling on combined offset (default 0.8; effective decay floor = Decay × (1 - this))
```

Fallback defaults in `internal/configs/config.balance.spells.go`
(or a new `config.balance.discovery.go` — whichever matches existing
placement; Spell* knobs are currently in `spells.go`, Recipe* in
`shops.go`, so a third split is appropriate).

### Config YAML

`_datafiles/config.yaml` — add new keys under the Balance section,
near existing `SpellDiscoveryBaseChance` / `RecipeDiscoveryBaseChance`:

```yaml
  DiscoveryPerceptionScale: 200.0
  DiscoverySkillScale: 100.0
  DiscoveryMaxDecayOffset: 0.8
```

### Call sites to update

All 5 occurrences of the inline formula become calls to
`configs.DiscoveryChance`:

| File | Line | Content | Skill |
|------|------|---------|-------|
| `internal/hooks/NewRound_DoCombat_helpers.go` | 304 | Player spell — traditional | Spellcasting |
| `internal/hooks/NewRound_DoCombat_helpers.go` | 323 | Player spell — manifestation | Manifestation |
| `internal/hooks/NewRound_DoCombat_helpers.go` | 423 | Mob spell — traditional | Spellcasting |
| `internal/hooks/NewRound_DoCombat_helpers.go` | 436 | Mob spell — manifestation | Manifestation |
| `internal/hooks/NewRound_UserRoundTick.go` | 391 | Player recipe | `recipe.Skill` |

Traditional+manifestation spell discovery currently share the same
`discoveryChance` variable. In the new world they MUST be computed
independently because the relevant skill differs. Each roll gets its
own call to `DiscoveryChance`.

Perception source:
- Players: `user.Character.Stats.Perception.ValueAdj` (or whatever
  the canonical adjusted-stat getter is — verify in the characters
  package)
- Mobs: same `Character.Stats.Perception.ValueAdj` — mobs use the
  shared Character struct

## Testing

### Unit tests (`internal/configs/discovery_test.go`)

Table-driven test covering the shape-verification scenarios above:

```go
func TestDiscoveryChance(t *testing.T) {
    // Seed the balance config with the documented defaults
    cases := []struct {
        name       string
        base, decay float64
        known, per, skill int
        wantChance float64 // ±0.05% tolerance
    }{
        {"newbie spells",   5.0, 0.1,  3, 100,   0, 3.85},
        {"early spells",    5.0, 0.1,  8, 110,  10, 2.97},
        {"mid spells",      5.0, 0.1, 12, 130,  25, 2.83},
        {"late spells",     5.0, 0.1, 18, 160,  50, 3.07},
        {"gm spells",       5.0, 0.1, 20, 200, 100, 3.57},
        {"newbie recipes", 10.0, 0.1,  3, 100,   0, 7.69},
        {"gm recipes",     10.0, 0.1, 20, 200, 100, 7.14},
        // Edge cases
        {"per below baseline clamps to 0", 5.0, 0.1, 10,  50, 50, /* compute */ },
        {"skill above scale clamps to 1",  5.0, 0.1, 10, 100, 200, /* compute */ },
        {"known=0 returns base",           5.0, 0.1,  0, 100,   0, 5.00},
        {"max offset cap at 0.8",          5.0, 0.1, 20, 300, 200, /* compute */ },
    }
    // ...
}
```

### Integration points (no new tests needed)

- Existing `TestPlayerCastSpellProgression` etc. should continue to
  pass — the chance value changes but the sampling flow is
  unchanged.
- Manual smoke: start a fresh character, cast a few spells, verify
  discovery message still fires occasionally. Then give the
  character high Per + Spellcasting rank and verify discoveries
  come more often.

## Edge Cases

- **Perception below 100** (some low-Per species): clamped to 0
  contribution. No penalty — these characters just get the baseline
  decay, same as today.
- **Skill of 0**: contribution = 0. For a fresh character who just
  cast their first spell, the formula degrades to the current
  behavior (offset = 0).
- **Perception above 300** (exotic buffs, edge mutations):
  per-contribution clamps to 1. Skill must carry the rest.
- **Known count of 0**: the decay term `known × effDecay` is 0
  regardless of offset. Chance = Base. Matches current behavior.
- **Mob Perception**: mobs share the Character struct, so
  `mob.Character.Stats.Perception.ValueAdj` works identically.
  Mobs without training (most common case) have Per ≈ species base.

## Non-Goals

- **No change to BaseChance or BaseDecay defaults.** Tuning base
  values is a separate balance pass — this work is purely about
  adding the stat-driven offset.
- **No player-visible "show me my discovery rate" command.** If
  Duard or others want visibility, that's a followup.
- **No change to the eligibility filtering or sampling logic** in
  the call sites. Only the chance computation changes.
- **No retroactive recompute** for characters mid-session — the
  formula simply produces a different chance on the next successful
  cast/craft. No migration needed.

## Risk / Rollback

- Formula is pure math; rollback = revert the config YAML +
  `DiscoveryChance` helper and restore the inline formulas.
- Defaults are conservative (offset maxes at 0.8, so the worst-case
  is ~5× current discovery chance for a maxed character — not a
  game-breaking acceleration).
- Mob spell discovery acceleration is a side effect. Caster mobs
  with high Per + Spellcasting will discover spells faster than
  today. That's fine: a battle-hardened mob casting repeatedly
  *should* expand its repertoire; the alternative is they stay
  forever bound to their seed list.
