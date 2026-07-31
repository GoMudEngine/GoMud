# Achievements Context

## Purpose

`internal/achievements` holds authored achievement definitions and evaluates
whether a character has earned them. It is a **pure evaluator over a fixed
trigger vocabulary** — it does not listen to events, does not write to
characters, and does not award anything. The caller checks and the caller
awards.

Unlocked ids live on the character, not here.

## Files

- **achievements.go** — `Definition`, `Trigger`, the category constants, and
  the registry.
- **evaluate.go** — `Evaluate` and `Progress`, plus the per-trigger readers.
- **loader.go** — YAML load and definition validation.

## Types

```go
type Definition struct {
    Id          string
    Name        string
    Description string
    Category    string   // combat | exploration | wealth | progression | quests
    Points      int
    Trigger     Trigger
}

type Trigger struct {
    Type      string
    Threshold int    // count/value trigger types
    Stat      string // stat_reached — a primary stat name, or "any"
    Skill     string // skill_reached — a skill name, or "any"
    Token     string // quest_completed — a quest token
}
```

Categories are a **closed set** validated at load: `combat`, `exploration`,
`wealth`, `progression`, `quests`. A file using anything else fails validation.

`Stat` and `Skill` both accept the literal `"any"`, which is how "reach 150 in
*any* stat" is expressed without one definition per stat.

## Public API

```go
func LoadDataFiles()
func All() []Definition                 // load order, stable
func Get(id string) (Definition, bool)
func PointsFor(ids map[string]uint64) int

func Evaluate(t Trigger, c *characters.Character, earnedPoints int) bool
func Progress(t Trigger, c *characters.Character) (current, target int, numeric bool)
```

`Evaluate` answers "is this earned right now?". `Progress` powers the progress
bar in the achievements UI and returns `numeric = false` for triggers that have
no meaningful partial state (a quest token is earned or it is not).

`Evaluate` takes `earnedPoints` so that meta-achievements — "earn N achievement
points" — can be expressed in the same vocabulary without recursion.

## Gotchas

- **`All()` is ordered by `registryOrder`, `Get`/`PointsFor` read the map.**
  Only `All()` is deterministic. Anything that renders a list must use it.
- **`PointsFor` silently skips unknown ids.** A character carrying an
  achievement id that has since been deleted from the data files loses those
  points with no warning — intentional, so removing an achievement does not
  break saves, but it means point totals can drop after a content change.
- **Nothing here awards.** `Evaluate` returning true has no side effect; the
  calling module is responsible for recording the unlock and announcing it.
- **`roomsExplored` reads `Character.VisitedRooms`**, the same fog-of-war map
  the mapper uses. An exploration achievement is therefore affected by anything
  that resets visited rooms.

## Dependencies

`characters`, `configs`, `mudlog`, plus YAML. No dependency on `events` or
`users` — evaluation is a pure function of a character.

## Consumers

`modules/achievements` (the event wiring and award announcements),
`internal/usercommands` (`achievements`), and the leaderboard.
