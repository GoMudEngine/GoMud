# Dice Package — Statistical Distribution Roller

A statistical dice rolling system for DOGMud based on normal (Gaussian)
distributions. Every attack, defense, spell check, grapple, and skill roll
in the engine flows through this package.

---

## The Master Randomness Knob: `RollSpread`

**This is the single most impactful balance lever in the game.**

All stat-based rolls compute their standard deviation as:

```
stdDev = stat_value × RollSpread
```

`RollSpread` defaults to **0.15** (15%) and is loaded from
`_datafiles/config.yaml` at server startup via `SetRollSpread`.  Changing it
re-scales the spread of *every single dice roll in the engine simultaneously*
— no individual files need to be touched.

### Effect on Gameplay

At the human baseline of `stat = 100`:

| RollSpread | stdDev | 120 vs 100 win% | Feel |
|------------|--------|-----------------|------|
| 0.05       | 5.0    | ~99%            | Near-deterministic. High stats dominate completely. |
| 0.10       | 10.0   | ~92%            | Low variance. Skill matters most. |
| **0.15**   | **15.0** | **~78%**      | **DEFAULT — balanced, recommended.** |
| 0.20       | 20.0   | ~67%            | Higher variance. Upsets are common. |
| 0.30       | 30.0   | ~57%            | Chaotic. Luck rivals skill. |

### Crit / Fumble Rates

**Not affected by `RollSpread`.** Criticals trigger when a roll's z-score
is ≥ +2.0 (or ≤ −2.0), which is always ~2.3% per roll regardless of the
absolute standard deviation — the z-score distribution is invariant to
scaling.

### Wiring

```go
// main.go — called once at startup after configs.ReloadConfig()
dice.SetRollSpread(float64(configs.GetGamePlayConfig().RollSpread))
```

Changing `config.yaml` requires a server restart.

---

## Stat-Based Roll API (use these for all game mechanics)

### `RollStat(mean float64) RollResult`

Single stat-based roll. Standard deviation is derived automatically from
`StdDevFor(mean)`. Use this for skill checks, damage/heal rolls, and
knockdown-chance rolls.

```go
result := dice.RollStat(attackerStat)
// result.Value    — the actual roll outcome
// result.ZScore   — >= 2.0 crit, <= -2.0 fumble
```

### `OpposedRollStat(atk, def float64) (bool, float64, RollResult, RollResult)`

Contested check between two stat-based scores. Both sides roll with the
attacker's standard deviation (`StdDevFor(atk)`). Use this for every
attack-vs-defense, spell-vs-resist, and grapple check.

```go
success, margin, atkRoll, defRoll := dice.OpposedRollStat(attackScore, defenseScore)
if !success { /* fizzle / miss */ }
if atkRoll.ZScore >= 2.0 { /* critical hit */ }
if atkRoll.ZScore <= -2.0 { /* fumble / backfire */ }
```

---

## Low-Level API (explicit stdDev — use sparingly)

These exist for cases where variance is **not** proportional to a stat, such
as weapon damage variance derived from dice-notation item specs.

### `Roll(mean, stdDev float64) RollResult`
### `OpposedRoll(atk, def, stdDev float64) (bool, float64, RollResult, RollResult)`
### `StdDevFor(mean float64) float64`

Returns `mean × RollSpread` (floor 1.0).  Prefer calling `RollStat` /
`OpposedRollStat` directly rather than computing this yourself.

---

## RollResult Fields

```go
type RollResult struct {
    Value       float64 // The actual roll outcome
    Mean        float64 // The distribution mean (stat value)
    StdDev      float64 // Standard deviation used
    ZScore      float64 // Standard deviations from mean (crit/fumble detection)
    Percentile  float64 // 0–100 percentile of this roll
    Success     bool    // True if the roll beat its check target
    Margin      float64 // Margin of success (positive) or failure (negative)
    Description string  // Human-readable description
}
```

---

## Z-Score Reference

| Z-Score | Percentile | Interpretation |
|---------|------------|----------------|
| −3.0    | 0.1%       | Extremely low  |
| −2.0    | 2.3%       | **Fumble threshold** |
| −1.0    | 15.9%      | Below average  |
|  0.0    | 50.0%      | Average        |
| +1.0    | 84.1%      | Above average  |
| +2.0    | 97.7%      | **Crit threshold** |
| +3.0    | 99.9%      | Extremely high |

---

## Utility Functions

```go
dice.RollDamage(base, variance, min float64) float64   // explicit variance damage
dice.DiceToDistribution(dCount, dSides, bonus int)     // NdM+bonus → (mean, stdDev)
dice.Percentile(chance float64) (bool, float64)        // simple % check
dice.RollBetween(min, max float64) float64
dice.RollBetweenInt(min, max int) int
dice.RollTable(weights []int) int                      // weighted loot table
```

---

## Testing

```bash
go test ./internal/dice/... -v
go test ./internal/dice/... -bench=.
```
