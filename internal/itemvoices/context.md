# Item Voices Context

## Purpose

`internal/itemvoices` gives an item a personality: a named pool of lines keyed
by event, so a sentient or storied item can speak when it is wielded, when it
strikes, when its owner is hurt, and so on.

It is a small authored-data package — a registry of `VoiceSpec` files and one
lookup that returns a random line for an event.

## API

```go
type VoiceSpec struct { /* VoiceId + event → lines */ }

func LoadDataFiles()
func GetVoice(id string) *VoiceSpec
func AllVoiceIds() []string
func SeedVoicesForTest(m map[string]*VoiceSpec) func()

func (v *VoiceSpec) Id() string
func (v *VoiceSpec) Filepath() string
func (v *VoiceSpec) Validate() error
func (v *VoiceSpec) Line(event string) string
```

An item references a voice by id; `Line(event)` picks one of the authored lines
for that event.

## Gotchas

- **`Line` returns an empty string for an unknown event or an empty pool.** The
  caller must treat empty as "say nothing" rather than sending a blank line.
- **`GetVoice` returns nil for an unknown id** — a typo in an item YAML gives a
  silent, voiceless item rather than a load error.
- **Voices are shared, not per-instance.** Two copies of the same item draw
  from the same pool and have no memory of what the other said.
- **Lines are player-facing text**: 80-column wrapping and no raw numbers apply.

## Dependencies

`util`, `mudlog`, and the fileloader.

## Consumers

`internal/items` and the combat/equip messaging paths.
