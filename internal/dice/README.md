# Dice Package

A comprehensive dice rolling system for DOGMud that provides various rolling mechanics for stats, combat, skill checks, and other game systems.

## Features

- **Basic Rolling**: Standard XdY+modifier rolls with critical/fumble detection
- **Advantage/Disadvantage**: Roll twice and take best/worst result
- **Opposed Rolls**: Contested checks between two stats
- **Difficulty Checks**: Roll against target numbers with success/failure
- **Stat Array Generation**: Multiple methods for rolling character stats
- **Specialized Systems**: Success pools, exploding dice, weighted tables, and more

## Basic Usage

### Simple Rolls

```go
import "github.com/GoMudEngine/GoMud/internal/dice"

// Roll 2d6+3
result := dice.Roll(2, 6, 3)
fmt.Printf("Result: %d\n", result.Total)
fmt.Printf("Raw rolls: %v\n", result.RawRolls)

// Check for criticals (automatic on d20 and d100)
if result.Critical {
    fmt.Println("Critical success!")
}
if result.Fumble {
    fmt.Println("Fumble!")
}
```

### Advantage/Disadvantage

```go
// Roll with advantage (roll twice, take higher)
result := dice.RollWithType(1, 20, 5, dice.Advantage)
fmt.Printf("Rolled %v, selected: %d\n", result.Rolls, result.Total)

// Roll with disadvantage (roll twice, take lower)
result := dice.RollWithType(1, 20, 5, dice.Disadvantage)
```

### Difficulty Checks

```go
// Check if character stat beats difficulty
stat := 60  // Character's effective stat (base + modifiers)
difficulty := 50  // Target number to beat

result := dice.DifficultyCheck(stat, difficulty, dice.Normal)
if result.Success {
    fmt.Printf("Success! Margin: %d\n", result.Margin)
}

// With advantage
result := dice.DifficultyCheck(stat, difficulty, dice.Advantage)
```

### Opposed Rolls

```go
// Two characters competing (e.g., stealth vs perception)
attackerStat := 50
defenderStat := 40

success, margin, atkRoll, defRoll := dice.OpposedRoll(attackerStat, defenderStat)
if success {
    fmt.Printf("Attacker wins by %d\n", margin)
}
```

### Contested Rolls

```go
// Similar to opposed but with advantage/disadvantage support
stat1Wins, margin, roll1, roll2 := dice.ContestedRoll(60, 50, dice.Normal, dice.Disadvantage)
```

## Character Creation

### Rolling Stat Arrays

```go
// Standard 3d6 (range: 3-18, average: 10-11)
stats := dice.RollStatArray(3, 6, 0, 6, false)

// 4d6 drop lowest (range: 4-24, average: 12-13) - more generous
stats := dice.RollStatArray(4, 6, 0, 6, true)

// 2d6+6 (range: 8-18, average: 13) - bounded higher
stats := dice.RollStatArray(2, 6, 6, 6, false)

fmt.Printf("Stats: %v\n", stats)  // e.g., [12, 14, 10, 15, 8, 13]
```

### Bell Curve Rolling

```go
// Roll 3d6 for more predictable results (bell curve distribution)
result := dice.BellCurve(0)  // modifier optional
```

## Advanced Systems

### Percentile Checks

```go
// Simple percentage chance (1-100)
success, roll := dice.Percentile(75)  // 75% chance
if success {
    fmt.Printf("Success! Rolled %d\n", roll)
}
```

### Weighted Random Tables

```go
// Roll on a weighted loot table
weights := []int{10, 20, 30, 40}  // Common, Uncommon, Rare, Epic
index := dice.RollTable(weights)

loot := []string{"Common", "Uncommon", "Rare", "Epic"}
fmt.Printf("Found %s loot!\n", loot[index])
```

### Exploding Dice

```go
// Dice that "explode" on max value (roll again and add)
result := dice.ExplodingDice(3, 6, 0, 2)  // 3d6 with max 2 explosions per die
fmt.Printf("Total: %d from rolls: %v\n", result.Total, result.RawRolls)
```

### Success Pool

```go
// Count successes (used in World of Darkness, Shadowrun, etc.)
result := dice.SuccessPool(5, 10, 7)  // Roll 5d10, target 7+
fmt.Printf("%d successes from %v\n", result.Total, result.RawRolls)
```

## Utility Functions

### Roll Range

```go
// Random number between min and max (inclusive)
value := dice.RollBetween(10, 20)
```

### Average/Min/Max

```go
// Calculate expected results
avg := dice.AverageRoll(2, 6, 3)   // 10.0
min := dice.MinRoll(2, 6, 3)       // 5
max := dice.MaxRoll(2, 6, 3)       // 15
```

### Critical Detection

```go
// Check if a roll is critical or fumble with custom thresholds
naturalRoll := 18
diceSides := 20
critThreshold := 18  // 18-20 is critical
fumbleThreshold := 3  // 1-3 is fumble

isCrit, isFumble := dice.CheckCritical(naturalRoll, diceSides, critThreshold, fumbleThreshold)
```

## RollResult Structure

All roll functions return a `RollResult` struct containing:

```go
type RollResult struct {
    Total       int      // Final result after modifiers
    RawRolls    []int    // Individual die results
    Modifier    int      // Total modifier applied
    Success     bool     // Whether the roll succeeded (for checks)
    Margin      int      // Margin of success/failure
    Critical    bool     // Whether this was a critical success
    Fumble      bool     // Whether this was a critical failure
    Description string   // Human-readable description
    Rolls       []int    // All rolls made (for advantage/disadvantage)
}
```

## Critical Success/Failure Rules

### Automatic Detection

- **d20 rolls**: Natural 20 is critical, natural 1 is fumble
- **d100 rolls**: 95+ is critical, 5 or less is fumble
- **Difficulty checks**: Criticals auto-succeed, fumbles auto-fail

### Custom Thresholds

Use `CheckCritical()` or `ApplyCriticalToResult()` for custom thresholds.

## Examples in DOGMud Context

### Skill Check

```go
// Player trying to pick a lock
skillLevel := user.Character.GetSkillLevel(skills.Skulduggery)
stat := user.Character.Stats.Dexterity.ValueAdj
effectiveStat := stat + (skillLevel * 10)
difficulty := lockDifficulty

result := dice.DifficultyCheck(effectiveStat, difficulty, dice.Normal)
if result.Critical {
    // Lock picked perfectly, no time wasted
} else if result.Success {
    // Lock picked with margin determining quality
} else if result.Fumble {
    // Lock jammed, alert nearby enemies
} else {
    // Failed, can try again
}
```

### Combat Hit Check

```go
// Determine if attack hits
attackerSpeed := attacker.Stats.Dexterity.ValueAdj
defenderSpeed := defender.Stats.Dexterity.ValueAdj

success, margin, _, _ := dice.OpposedRoll(attackerSpeed, defenderSpeed)
if success {
    // Hit! Margin could affect damage
}
```

### Random Loot Drop

```go
// Mob drops random quality item
rarities := []int{50, 30, 15, 5}  // Common, Uncommon, Rare, Legendary
rarity := dice.RollTable(rarities)
```

## Performance

All functions are optimized for frequent calls during game loops. Benchmark results available in `dice_test.go`.

## Testing

Run comprehensive tests:

```bash
go test ./internal/dice -v
```

Run benchmarks:

```bash
go test ./internal/dice -bench=.
```
