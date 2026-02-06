# Dice Package - Statistical Distribution Roller

A statistical dice rolling system for DOGMud based on normal (Gaussian) distributions. This provides smooth, predictable probabilities for combat, skill checks, and stat generation.

## Philosophy

Unlike traditional tabletop dice (2d6, 1d20, etc.), this system uses **normal distributions** for more nuanced probability control:

- **Predictable bell curves** around character stats
- **Configurable randomness** via standard deviation
- **Statistical analysis** built into every roll (z-scores, percentiles)
- **Smooth probability curves** rather than discrete outcomes

## Core Concepts

### Normal Distribution Rolls

Every roll uses a character's stat as the **mean** and a configurable **standard deviation** for variance:

```go
mean := characterStat    // 50 (character's effective stat)
stdDev := 10.0           // How much randomness (typically 10-15)

result := dice.Roll(mean, stdDev)
// Result: 47.75 (mean: 50.00, σ: 10.00, z: -0.23)
```

The **standard deviation** controls how random results are:
- **Low (5-10)**: Consistent, skill-based gameplay
- **Medium (10-15)**: Balanced randomness and skill
- **High (15-20)**: Very swingy, luck-based gameplay

### Standard Deviation Helper

```go
// Calculate stdDev based on your stat range and desired randomness
statRange := 100.0        // Stats range from 0-100
randomnessFactor := 0.15  // 15% randomness (moderate)

stdDev := dice.StandardDeviation(statRange, randomnessFactor)
// Result: 15.0
```

**Recommended randomness factors:**
- **0.10** (10%): Low randomness, skill matters most
- **0.15** (15%): Moderate, good balance
- **0.20** (20%): High randomness, upsets likely

## Basic Usage

### Simple Roll

```go
result := dice.Roll(50.0, 10.0)

fmt.Printf("Result: %.2f\n", result.Value)        // 47.75
fmt.Printf("Z-score: %.2f\n", result.ZScore)      // -0.23 (slightly below average)
fmt.Printf("Percentile: %.1f%%\n", result.Percentile)  // 40.9% (below average roll)
```

### Difficulty Check

```go
playerStat := 60.0
difficulty := 50.0
stdDev := 10.0

result := dice.DifficultyCheck(playerStat, difficulty, stdDev)

if result.Success {
    fmt.Printf("Success! Margin: %.2f\n", result.Margin)
    // Higher margin = better success
} else {
    fmt.Printf("Failed by %.2f\n", -result.Margin)
}
```

### Opposed Roll (PvP, Stealth vs Perception, etc.)

```go
attackerStat := 60.0
defenderStat := 50.0
stdDev := 10.0

success, margin, atkRoll, defRoll := dice.OpposedRoll(attackerStat, defenderStat, stdDev)

if success {
    fmt.Printf("Attacker wins by %.2f!\n", margin)
} else {
    fmt.Printf("Defender wins by %.2f\n", -margin)
}
```

## Probability Analysis

### Success Chance Calculation

Before rolling, calculate the probability of success:

```go
playerStat := 60.0
difficulty := 50.0
stdDev := 10.0

chance := dice.SuccessChance(playerStat, difficulty, stdDev)
fmt.Printf("Success probability: %.1f%%\n", chance*100)
// Result: 84.1%

// For opposed rolls
attackerStat := 60.0
defenderStat := 50.0
winChance := dice.OpposedSuccessChance(attackerStat, defenderStat, stdDev)
fmt.Printf("Win probability: %.1f%%\n", winChance*100)
// Result: 76.0%
```

### Percentile Analysis

Find what value represents a given percentile:

```go
mean := 50.0
stdDev := 10.0

median := dice.GetPercentile(mean, stdDev, 50)  // 50.00
p75 := dice.GetPercentile(mean, stdDev, 75)     // 56.74
p95 := dice.GetPercentile(mean, stdDev, 95)     // 66.45
p99 := dice.GetPercentile(mean, stdDev, 99)     // 73.26
```

## Critical Hits & Fumbles

Criticals are based on **z-scores** (standard deviations from mean):

```go
result := dice.Roll(50.0, 10.0)

// Check if critical or fumble
isCrit, isFumble := dice.CriticalCheck(result, 2.0, -2.0)

// z-score ≥ 2.0 = critical (2.28% chance)
// z-score ≤ -2.0 = fumble (2.28% chance)
```

**Common thresholds:**
- **±2.0**: ~2.3% chance (rigorous, D&D-like)
- **±1.5**: ~6.7% chance (moderate)
- **±1.0**: ~15.9% chance (frequent)

### Roll with Criticals

```go
result, isCrit, isFumble := dice.RollWithCriticals(50.0, 10.0, 2.0, -2.0)

if isCrit {
    // Exceptional success!
} else if isFumble {
    // Catastrophic failure!
}
```

## Character Creation

### Rolling Stat Arrays

```go
// Generate 6 stats with mean 12, stdDev 2, clamped to 3-18
stats := dice.RollStatArray(6, 12.0, 2.0, 3.0, 18.0)
// Result: [12, 13, 12, 14, 7, 13]

// More generous (mean 14, higher variance)
stats := dice.RollStatArray(6, 14.0, 3.0, 3.0, 18.0)

// Tight distribution (elite character)
stats := dice.RollStatArray(6, 15.0, 1.5, 10.0, 18.0)
```

### Clamped Rolls

Ensure rolls stay within logical bounds:

```go
// Roll with bounds
result := dice.RollClamped(50.0, 20.0, 30.0, 70.0)
// Never goes below 30 or above 70

// Integer version
value := dice.RollIntClamped(50.0, 20.0, 30.0, 70.0)
```

## Combat & Damage

### Damage Rolls

```go
baseDamage := 50.0
variance := 10.0
minDamage := 10.0

damage := dice.RollDamage(baseDamage, variance, minDamage)
// Result: 47.32 (can't go below 10)

// Integer version for discrete damage
damage := dice.RollDamageInt(baseDamage, variance, minDamage)
// Result: 52
```

### Combat Example

```go
// Attacker stats
attackStat := attacker.Stats.Dexterity.ValueAdj
attackDamage := attacker.GetAttackDamage()

// Defender stats
defenseStat := defender.Stats.Dexterity.ValueAdj

stdDev := dice.StandardDeviation(100.0, 0.15)  // 15% randomness

// Check if attack hits
hits, margin, _, _ := dice.OpposedRoll(attackStat, defenseStat, stdDev)

if hits {
    // Roll damage with variance based on margin
    damageVariance := stdDev * 0.5
    damage := dice.RollDamageInt(attackDamage, damageVariance, 1.0)

    // Check for critical
    result := dice.Roll(attackStat, stdDev)
    isCrit, _ := dice.CriticalCheck(result, 2.0, -2.0)
    if isCrit {
        damage *= 2
    }
}
```

## Utility Functions

### Simple Percentile Check

```go
success, roll := dice.Percentile(75.0)  // 75% chance
// Returns: (true, 68.3) or (false, 82.1)
```

### Random Range

```go
value := dice.RollBetween(10.0, 50.0)    // Float
value := dice.RollBetweenInt(10, 50)     // Integer
```

### Weighted Table

```go
// Loot rarity table
rarities := []int{50, 30, 15, 5}  // Common, Uncommon, Rare, Legendary
rarity := dice.RollTable(rarities)

items := []string{"Common", "Uncommon", "Rare", "Legendary"}
fmt.Printf("Found %s item!\n", items[rarity])
```

## RollResult Structure

All roll functions return detailed information:

```go
type RollResult struct {
    Value       float64 // The actual roll result
    Mean        float64 // The distribution mean (stat value)
    StdDev      float64 // Standard deviation used
    Success     bool    // Whether the check succeeded
    Margin      float64 // Margin of success/failure
    ZScore      float64 // Standard deviations from mean
    Percentile  float64 // What percentile this roll represents
    Description string  // Human-readable description
}
```

## Statistical Reference

### Z-Score Interpretation

| Z-Score | Percentile | Interpretation |
|---------|------------|----------------|
| -3.0    | 0.1%       | Extremely low |
| -2.0    | 2.3%       | Very low |
| -1.0    | 15.9%      | Below average |
| 0.0     | 50.0%      | Average |
| +1.0    | 84.1%      | Above average |
| +2.0    | 97.7%      | Very high |
| +3.0    | 99.9%      | Extremely high |

### Win Probability (Opposed Rolls)

For two characters with stats A and B, using stdDev σ:

| Stat Diff | Win Chance (A vs B) |
|-----------|---------------------|
| +0σ       | 50.0% (even match) |
| +0.5σ     | 61.8% |
| +1.0σ     | 76.0% (significant advantage) |
| +1.5σ     | 86.6% |
| +2.0σ     | 92.9% (overwhelming advantage) |

Example: If A has stat 60, B has stat 50, and σ=10, then A has a +1σ advantage = 76% win chance.

## Performance

All functions are optimized for high-frequency calls:

```bash
BenchmarkRoll-8                  5000000    240 ns/op
BenchmarkOpposedRoll-8           2000000    478 ns/op
BenchmarkDifficultyCheck-8       2000000    483 ns/op
BenchmarkSuccessChance-8        10000000    112 ns/op
```

## Testing

Run comprehensive tests:

```bash
go test ./internal/dice -v
go test ./internal/dice -bench=.
```

Tests include:
- Distribution correctness (10,000 roll statistical validation)
- Opposed roll win rate verification
- Success probability accuracy
- Critical/fumble rate verification
- Percentile calculation accuracy
