# Mutators Context

## Purpose

A **mutator** is a named, time-bounded modifier attached to a *room*. It can
rewrite the room's name and description, inject an alert line, grant buffs to
whoever is standing there, open extra exits, change the light level, scale
regen, and override PvP rules — all while it is "live," and all of it reverts
when it decays.

Mutators are the mechanism behind environmental state that comes and goes on a
schedule: weather effects on a room, a seasonal bloom, a fire that burns down
to embers and then to ash. They are authored as YAML specs and attached to
rooms by id; the room stores only the id plus two round stamps.

Scope note: despite the name, mutators are **room-scoped only** and are
unrelated to `internal/mutations` (the player Chrysalis mutation system).

## Files

- **mutators.go** — the whole package: spec type, instance type, lifecycle, and
  the YAML loader.
- **test_helpers.go** — `SeedSpecsForTest(specs ...MutatorSpec) func()` installs
  specs into the package-global registry and returns a restore func.

## Types

### `MutatorSpec` — the authored definition

```go
type MutatorSpec struct {
    MutatorId           string
    NameModifier        *TextModifier
    DescriptionModifier *TextModifier
    AlertModifier       *TextModifier            // append-only in practice
    DecayIntoId         string                   // becomes this mutator on decay
    PlayerBuffIds       []int                    // applied to players + their followers
    MobBuffIds          []int                    // applied to mobs
    NativeBuffIds       []int                    // applied only to mobs that spawned here
    DecayRate           string                   // gametime period string
    RespawnRate         string                   // gametime period string
    LightMod            int                      // -2..2
    RegenMultiplier     float64                  // 1.0 / 0 = no change
    Exits               map[string]exit.RoomExit // only reachable while live
    Pvp                 PvpOverride
    OutdoorOnly         bool                     // skipped in indoor biomes
}
```

### `Mutator` — the per-room instance

```go
type Mutator struct {
    MutatorId      string
    SpawnedRound   uint64
    DespawnedRound uint64 // 0 == live
}
```

Three fields, because this is what gets serialised into every room instance
save. All the heavy data lives in the spec, looked up by id.

### `TextModifier`

```go
type TextModifier struct {
    Behavior     TextBehavior // prepend | append | replace (default replace)
    Text         string
    ColorPattern string       // optional colorpatterns name
}
```

## Lifecycle

`(*Mutator).Update(currentRound)` is the state machine, and it is the only
place spawn/despawn transitions happen:

1. **Uninitialised** (both stamps 0) → normally spawns immediately. *Exception:*
   if `RespawnRate` ends in `noon`, `midnight`, `sunrise`, or `sunset`, the
   mutator starts **despawned** and waits for that moment to come round, rather
   than firing at an arbitrary point in the day.
2. **Despawned with a `RespawnRate`** → respawns once
   `gametime.GetDate(despawnedRound).AddPeriod(respawnRate)` is reached.
3. **Live with a `DecayRate`** → on expiry, either becomes `DecayIntoId` (fresh
   spawn stamp, still live) or simply despawns. Decay chains may be circular —
   that is the supported way to build a repeating cycle.

`Live()` is `DespawnedRound == 0`. `Removable()` reports whether a despawned
mutator can be dropped from the room entirely — true only when it will neither
respawn nor decay into anything.

## Public API

```go
func LoadDataFiles()
func GetAllMutatorSpecs() []MutatorSpec
func GetMutatorSpec(mutatorId string) *MutatorSpec
func GetAllMutatorIds() []string
func IsMutator(mutName string) bool

func (ml *MutatorList) Has(mutName string) bool
func (ml *MutatorList) Add(mutName string) bool     // false if id unknown
func (ml *MutatorList) Remove(mutName string) bool
func (ml *MutatorList) Update(roundNow uint64)
func (ml *MutatorList) GetActive() MutatorList

func (m *Mutator) Live() bool
func (m *Mutator) Removable() bool
func (m *Mutator) GetSpec() *MutatorSpec           // nil if id unknown
func (m *Mutator) Update(currentRound uint64)

func (m *MutatorSpec) Id() string
func (m *MutatorSpec) Filename() string            // ConvertForFilename(id) + ".yaml"
func (m *MutatorSpec) Filepath() string
func (m *MutatorSpec) Validate() error
func (m *MutatorSpec) Save() error
```

## Gotchas

- **`GetSpec()` returns nil for an unknown id, and both `Update` and
  `Removable` guard for it deliberately.** A content typo yields a mutator with
  no spec; the guards exist so that (a) the server does not panic on a nil
  dereference and (b) `Removable` does not decide the orphan is disposable and
  silently delete authored content from a room. Do not "simplify" those nil
  checks away — the comments in the source say the same thing.
- **`Add` returns `false` for an unregistered id** rather than adding a
  placeholder. Check the return value.
- **Re-adding a despawned mutator resets it** (both stamps cleared, then
  `Update`) instead of appending a duplicate entry.
- **`Filepath()` returns just the filename**, so specs live flat in the mutators
  data directory — no zone subfoldering.
- **`OutdoorOnly` is honoured by the caller, not here.** The spec field is
  inert inside this package; room/weather code checks it before applying.

## Dependencies

`configs`, `exit`, `fileloader`, `gametime`, `mudlog`, `util`, plus
`gopkg.in/yaml.v2` and `github.com/pkg/errors`. The `gametime` dependency is
load-bearing: every decay and respawn interval is a gametime period string, not
a round count.

## Consumers

`rooms`, `hooks`, `usercommands`, `internal/web`, `modules/gmcp`, and
`modules/weather/engine` — the weather simulation is the heaviest user, driving
room mutators as fronts move across zones.
