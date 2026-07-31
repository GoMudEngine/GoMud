# Dice Context

## Purpose

`internal/dice` is the engine's randomness. Every stat contest, attack, defence,
skill check, and damage roll goes through it. It is **distribution-based, not
die-based**: a roll is a sample from a normal distribution around a mean, so
crits and fumbles live in the tails rather than on a flat d20.

There is a companion write-up with win-probability tables in
`internal/dice/README.md`.

## The rule that matters

**For anything stat-based, call `RollStat` / `OpposedRollStat`.** They take only
a mean and derive the standard deviation from the global spread factor:

```go
func RollStat(mean float64) RollResult
func OpposedRollStat(atk, def float64) (bool, float64, RollResult, RollResult)
func StdDevFor(mean float64) float64        // mean × RollSpread
func SetRollSpread(factor float64)
```

`Roll` and `OpposedRoll` take an explicit `stdDev` and are **low-level**. Use
them only where variance is genuinely not proportional to the stat — weapon
damage variance taken from an item spec, for example.

`util.Rand` and `util.LogRoll` are **not** used for hit or attack checks.

## `RollSpread`

The single master randomness knob, `GamePlay.RollSpread` in `config.yaml`
(default `0.15`). Changing it rescales every roll in the engine at once.

Z-score thresholds are unaffected by it: `ZScore >= 2.0` is a critical,
`ZScore <= -2.0` is a fumble or backfire — roughly 2.3% each, whatever the
spread.

## Public API

Rolling:

```go
func Roll(mean, stdDev float64) RollResult
func RollInt(mean, stdDev float64) int
func RollClamped(mean, stdDev, min, max float64) RollResult
func RollIntClamped(mean, stdDev, min, max float64) int
func RollBetween(min, max float64) float64
func RollBetweenInt(min, max int) int
func RollTable(weights []int) int
func RollStatArray(count int, mean, stdDev, min, max float64) []int
```

Contests:

```go
func OpposedRoll(attackerStat, defenderStat, stdDev float64) (bool, float64, RollResult, RollResult)
func DifficultyCheck(stat, difficulty, stdDev float64) RollResult
func CompareRolls(roll1, roll2 RollResult) int
func Percentile(chance float64) (bool, float64)
```

Damage:

```go
func RollDamage(baseDamage, variance, minDamage float64) float64
func RollDamageInt(baseDamage, variance, minDamage float64) int
```

Criticals:

```go
func CriticalCheck(result RollResult, criticalThreshold, fumbleThreshold float64) (bool, bool)
func RollWithCriticals(mean, stdDev, critThreshold, fumbleThreshold float64) (RollResult, bool, bool)
```

Analysis (no randomness — for tuning, tests, and documentation):

```go
func SuccessChance(stat, difficulty, stdDev float64) float64
func OpposedSuccessChance(attackerStat, defenderStat, stdDev float64) float64
func ExpectedMargin(stat, difficulty float64) float64
func AverageResult(mean, stdDev float64) float64
func GetPercentile(mean, stdDev, percentile float64) float64
func StandardDeviation(statRange, randomnessFactor float64) float64
func DiceToDistribution(dCount, dSides, bonus int) (mean, stdDev float64)
```

`DiceToDistribution` is the bridge for authored content still expressed as
`2d6+3`: it converts that notation to the equivalent mean and standard
deviation so the same distribution machinery can roll it.

## `RollResult`

Carries the rolled value plus the z-score, which is what makes crit and fumble
detection uniform across every subsystem. `String()` renders it for debug logs.

## Gotchas

- **`SetRollSpread` is global and process-wide.** Tests that change it must
  restore it, or every later test in the binary inherits the new variance.
- **Analysis functions are deterministic; rolling functions are not.** Do not
  use `SuccessChance` where you meant `Roll` — it will silently always succeed
  or always fail depending on how you compare it.
- **`RollTable` takes weights, not probabilities.** They need not sum to
  anything in particular.
- **Clamping changes the distribution.** `RollClamped` piles probability mass
  on the bounds; a clamped roll is no longer normal, and z-score-based crit
  detection on top of it will misbehave at the edges.
- **Never show a raw roll to a player.** Player-facing text uses descriptive
  language (`combat.GetDamageDescription`), not numbers.

## Dependencies

`configs` (for `RollSpread`) and the standard library `math`/`math/rand`.

## Consumers

`combat`, `hooks`, `mobs`, `characters`, `skills`, `crafting`, `spells`,
`justice` — effectively every gameplay system.
