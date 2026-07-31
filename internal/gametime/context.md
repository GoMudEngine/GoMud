# Game Time Context

## Purpose

`internal/gametime` converts the server's monotonic round counter into an
in-world calendar — year, month, week, day, hour, minute, day/night — and
back again. Nothing here ticks: there is no goroutine and no stored clock.
Every answer is derived on demand from `util.GetRoundCount()` and the
`RoundsPerDay` / `RoundSeconds` timing config.

DOGMud extends the upstream calendar with **three moons** whose phases feed
stat modifiers and mutation pacing, and with a zodiac year name.

## Files

- **gametime.go** — `GameDate`, `RoundTimer`, day/night control, and the
  period-string arithmetic (`AddPeriod`, `GetLastPeriod`).
- **months.go** — `MonthName(month int) string`.
- **moonphase.go** — the three moons and their stat contribution.
- **zodiac.go** — `GetZodiac(year int) string`.
- **copyover.go** — copyover contributor so a hot restart does not jump the
  clock.

## `GameDate`

```go
type GameDate struct {
    RoundNumber      uint64
    RoundsPerDay     int
    NightHoursPerDay int
    Year, Month, Week, Day, Hour int
    Minute      int
    MinuteFloat float64
    AmPm        string
    Night       bool
    DayStart, NightStart int
}
```

`GetDate(forceRound ...uint64) GameDate` is the entry point — no argument means
"now," an argument means "the calendar at that round." `ReCalculate()` refreshes
a `GameDate` in place after its `RoundNumber` is changed. `String(symbolOnly
...bool)` renders it for display. `Add(adjustHours, adjustDays, adjustYears)`
returns a shifted copy.

## Period strings

`AddPeriod(periodStr string) uint64` is the most-used function in the package:
it takes a human period and returns the **absolute round number** that period
lands on, measured from the receiver's round.

Accepted forms:

- `"10 days"` — quantity + unit. Units: `years`, `months`, `weeks`, `days`,
  `hours`, `rounds`.
- bare period names such as `daily`, `weekly`, `noon`, `midnight`, `sunrise`,
  `sunset`.
- `"2 irl days"` / `"2 real days"` — real-world time, converted through
  `RoundSeconds`. `irl` and `real` are interchangeable.
- `""` — returns the receiver's round unchanged.

`GetLastPeriod(periodName string, roundNumber uint64) uint64` is the inverse:
the most recent round at which the named period boundary occurred.

Anything with an authored duration in YAML — mutator decay and respawn, shop
restock, schedule segments — routes through these two functions, which is why
they accept sloppy input rather than erroring. A malformed quantity silently
becomes `1`.

## Day / night

```go
func IsNight() bool
func SetToDay(roundAdjustment ...int)
func SetToNight(roundAdjustment ...int)
func SetTime(setToHour int, setToMinutes ...int)
```

The three setters shift the world clock and back the admin `time` command.
Day/night is a **derived** property (`GameDate.Night`), recomputed per query —
there is no transition callback and no cached state to invalidate.

## The three moons

Cycle lengths are multiples of `RoundsPerDay`, matching the lore in
`docs/world.md`:

| Moon           | Period mult | Influences                          |
|----------------|-------------|-------------------------------------|
| Swiftmoon      | 4.7         | Dexterity, Strength                 |
| The Wanderer   | 10.6        | Vitality, Willpower                 |
| The Eye        | 21.1        | Perception, Charisma; mutation rate |

```go
func GetSwiftmoonPhase() float64
func GetWandererPhase() float64
func GetEyePhase() float64
func GetAllPhases() (swiftmoon, wanderer, eye float64)
func CurrentMoonFlavorBucket() int
func MoonStatDelta(phase, maxMod float64, base int) int
```

Phase is a smooth `[0.0, 1.0]` **contribution**, not a raw cycle position: the
linear phase percent is passed through `(1 - cos(2π·p)) / 2`, so 0.0 is new
moon, 1.0 is full, and both quarters read 0.5. `MoonStatDelta` turns that into
an integer stat adjustment scaled off a base value.

**Use `GetAllPhases` when you need more than one** — each single-moon getter
recomputes all three and discards two.

## Other API

```go
type RoundTimer struct { RoundStart uint64; Period string }
func (r RoundTimer) Expired() bool

func MonthName(month int) string
func GetZodiac(year int) string
func CopyoverContributor() copyover.Contributor
```

`RoundTimer` is the small serialisable "started at round X, lasts for period P"
helper embedded in other packages' YAML.

## Gotchas

- **`RoundsPerDay == 0` short-circuits the moons to zero** rather than dividing
  by zero. A misconfigured timing block therefore reads as "all moons new," not
  as a crash — check config before hunting a phase bug.
- **`AddPeriod` returns an absolute round, not a duration.** Compare it against
  `util.GetRoundCount()`, never against a length.
- **Real-world conversion uses `84600` seconds per day**, not 86400 — an
  upstream typo now baked into every persisted `irl` period. Changing it would
  shift existing timers.
- **Nothing here is cached.** `GetDate()` recomputes from scratch on every call.
  Cheap individually; still worth hoisting out of a per-actor loop.
- **There is no `GetTimeConfig`** — timing lives in
  `configs.GetTimingConfig()`.

## Dependencies

`configs` (timing block), `util` (round count), `copyover`. No dependency on
rooms, users, or mobs — this package sits low in the graph and is safe to
import almost anywhere.

## Consumers

`mutators` (decay/respawn), `shops` (restock), `mobs` (schedules), `buffs`,
`rooms`, `usercommands`, `internal/hooks`, and `modules/weather`.
