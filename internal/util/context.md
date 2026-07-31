# Util Context

## Purpose

`internal/util` is the engine's foundation layer: the global round/turn
counters and the MUD-wide lock, plus string, file, encoding, and memory
helpers. Almost every other package imports it, so it imports almost nothing —
keep it that way.

Three files, despite the size of the surface: **util.go** (everything except
the two below), **memory.go** (memory reporting), **copyover.go** (round-count
survival across a hot restart).

## The global clock

```go
func GetRoundCount() uint64
func IncrementRoundCount() uint64
func SetRoundCount(newRoundCount uint64)
func GetTurnCount() uint64
func IncrementTurnCount() uint64
func SaveRoundCount(fpath string)
func LoadRoundCount(fpath string) uint64
func SetRoundCountForTest(r uint64)
func ResetRoundCountForTest()
```

`GetRoundCount()` is the single time source for the whole game. Schedules,
mutator decay, ferry position, buff expiry, and shop restock are all functions
of it. **Rounds and turns are not the same thing** — a round is the gameplay
tick; turns are finer-grained.

The `*ForTest` setters exist because so much behaviour is round-derived that
tests must be able to place themselves in time. They are global — restore what
you change.

## The MUD lock

```go
func LockMud()
func UnlockMud()
func RLockMud()
func RUnlockMud()
```

A process-wide RWMutex guarding world state. Take the read lock for inspection,
the write lock for mutation, and hold neither longer than necessary — this is
the engine's single biggest contention point.

## String handling

```go
func VisibleWidth(s string) int
func SplitString(input string, lineWidth int) []string
func SplitStringNL(input string, lineWidth int, nlPrefix ...string) string
func SplitButRespectQuotes(s string) []string
func BreakIntoParts(full string) []string
func StripPrepositions(input string) string
func StripANSI(str string) string
func ConvertColorShortTags(input string) string
func StripCharsForScreenReaders(s string) string
func ConvertToAscii(s string) string
func FormatNumber(n int) string
func BoolYN(b bool) string
func StringWildcardMatch(stringToSearch, patternToSearch string) bool
```

**`VisibleWidth` is not `len`.** Colour tags and multi-byte runes mean the byte
length of a MUD string bears no relation to how wide it prints. Every wrapping
and alignment decision must use `VisibleWidth`, and `SplitString` already does.

`ConvertToAscii` (backed by the `unicodeToAscii` table) and
`StripCharsForScreenReaders` serve the accessibility and
limited-client paths — a client that has not converged on UTF-8 gets readable
text rather than mojibake.

## Matching

```go
func FindMatchIn(searchName string, items ...string) (match, closeMatch string)
func GetMatchNumber(input string) (string, int)
func ConvertForFilename(input string) string
```

`FindMatchIn` returns **two** results: an exact match and a near match. Callers
that only read the first silently lose fuzzy matching — this is the routine
behind most "why doesn't `get lake iron nodule` work" questions, and it matches
whole multi-word phrases already.

`GetMatchNumber` parses the disambiguation forms (`2.sword`, `sword#2`).

**`ConvertForFilename` is load-bearing.** Data-file names are derived from
display names through it: lowercase, keep `a-z0-9`, drop apostrophes, every
other character becomes `_`. A mismatch between a `name:` field and its
filename is a startup panic. It must also be applied to *lookup input* so a
space-form query matches the underscore filename form.

## Dice (legacy)

```go
func Rand(maxInt int) int
func LogRoll(name string, rollResult, targetNumber int)
func RollDice(dice, sides int) int
func ParseDiceRoll(dRoll string) (attacks, dCount, dSides, bonus int, buffOnCrit []int)
func FormatDiceRoll(attacks, dCount, dSides, bonus int, buffOnCrit []int) string
```

**`Rand` and `LogRoll` are NOT used for hit or attack checks.** All stat-based
resolution goes through `internal/dice` (`RollStat`, `OpposedRollStat`). What
remains here is authored `2d6+3` notation parsing and non-combat randomness.

## Files, hashing, encoding

```go
func FilePath(pathParts ...string) string
func Save(path string, data []byte, doSafe ...bool) error
func SafeSave(path string, data []byte) error
func ValidateWorldFiles(exampleWorldPath, worldPath string) error

func Hash(input string) string
func Md5Bytes(input []byte) []byte
func Compress(input []byte) []byte
func Decompress(input []byte) []byte
func Encode(blobdata []byte) string
func Decode(base64str string) []byte
func GetLockSequence(lockIdentifier string, difficulty int, seed string, rotation uint64) string
```

`SafeSave` writes via a temporary file and renames, so an interrupted save
cannot truncate a player's character. Prefer it for anything that would hurt to
lose.

**`ValidateWorldFiles` hard-errors at boot when the live world is missing any
subfolder present in the example world.** This is why folders whose *contents*
are gitignored must still ship a tracked `.gitkeep` — git cannot store an empty
directory, and a fresh clone otherwise dies before loading a room.

`GetLockSequence` derives a deterministic lockpicking sequence, so the same lock
presents the same puzzle until its rotation changes.

## Display helpers

```go
func ProgressBar(complete float64, maxBarSize int, barParts ...string) (fullBar, emptyBar string)
func HealthClass(health, maxHealth int) string
func QuantizeTens(value, max int) int
```

`HealthClass` and `QuantizeTens` are how numeric state becomes a descriptive
band — the project never shows players raw values.

## Instrumentation

```go
func TrackTime(name string, timePassed float64)
func GetTimeTrackers() []Accumulator
type Accumulator struct{ /* … */ }
func (t *Accumulator) Record(nextValue float64)
func (t *Accumulator) Stats() (lowest, highest, average, count float64)
func (t *Accumulator) Average() float64

type MemReport func() map[string]MemoryResult
func AddMemoryReporter(name string, reporter MemReport)
func GetMemoryReport() (names []string, trackedResults []map[string]MemoryResult)
func ServerGetMemoryUsage() map[string]MemoryResult
func MemoryUsage(i interface{}) uint64
func FormatBytes(bytes uint64) string
```

`MemoryUsage` walks a value reflectively (`sizeOf`) — it is an approximation
for the admin memory report, not an allocator statistic, and it is **not
cheap**. Do not call it on a hot path.

## Server identity

```go
func SetServerAddress(addr string)
func GetServerAddress() string
func SetServerStart(t time.Time)
func GetServerStartUnix() int64
func CopyoverContributor() copyover.Contributor
```

The copyover contributor persists the round and turn counters across a hot
restart, which is what stops every timed thing in the world jumping when the
server reloads.

## Gotchas

- **Everything here is global.** Counters, trackers, the lock, the server
  address. Tests that mutate them must restore them.
- **`VisibleWidth`, not `len`.** Repeated because it is the most common bug
  this package causes.
- **`FindMatchIn` returns two values.** Ignoring the second disables fuzzy
  matching.
- **`Save` is safe by default** as of 2026-07-31: with no third argument it
  routes through `SafeSave`. Pass `false` to opt out and write directly. The
  default was the other way round, so a new caller inherited the risky path by
  omission.
- **Keep the import list tiny.** This package sits beneath nearly everything;
  importing a game package from here creates a cycle that is painful to unpick.

## Dependencies

`configs`, `mudlog`, `copyover`, and the standard library. Nothing else.

## Consumers

Effectively every package in `internal/` and `modules/`.
