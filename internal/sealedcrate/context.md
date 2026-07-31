# Sealed Crate Context

## Purpose

`internal/sealedcrate` is a small, deliberately dumb container: a fixed-capacity
box of items bound to a room, which can be filled, snapshotted, and drained
whole. It is the hand-off buffer between an actor that produces goods and one
that collects them — most visibly the forager→caravan handoff at a meeting
point, where the two NPCs are rarely in the room at the same time.

It exists as its own package precisely so it has **no** dependency on mobs,
rooms, or the economy. It is a data structure with a file format.

## Files

- **sealedcrate.go** — the `Crate` type.
- **persistence.go** — `SaveTo` / `LoadFrom` and the on-disk payload shape.

## API

```go
func New(roomId, capacity int) *Crate

func (c *Crate) RoomId() int
func (c *Crate) Capacity() int
func (c *Crate) Len() int
func (c *Crate) Add(it items.Item) bool      // false when full
func (c *Crate) DrainAll() []items.Item      // returns everything, empties the crate
func (c *Crate) Snapshot() []items.Item      // read-only copy, crate untouched
func (c *Crate) SetItemsForLoad(its []items.Item)

func SaveTo(path string, c *Crate) error
func LoadFrom(path string) (*Crate, error)
```

## Gotchas

- **`Add` returns false when the crate is full and the item is *not* stored.**
  Every caller must handle the rejected remainder — `caravan`'s
  `drainCrateIntoWagonCaravan` returns `(stored, rejected)` for exactly this
  reason.
- **`DrainAll` empties.** `Snapshot` does not. Reaching for the wrong one either
  duplicates goods or destroys them.
- **`SetItemsForLoad` bypasses the capacity check.** It is the deserialisation
  path only; using it as a bulk setter can produce an over-full crate that then
  rejects everything.
- **The crate does not know when its room unloads.** Persistence is the
  caller's job — `caravan.persistCaravanCrate` is the reference.

## Dependencies

`items`, plus YAML. Nothing else, by design.

## Consumers

`internal/caravan` and `internal/forager`.
