# Warehouse Context

## Purpose

`internal/warehouse` is the per-city goods buffer that sits between production
and retail. Foragers, caravans, and ferry factors deposit into it; vendors draw
from it. It also **accrues** stock on a timer, representing off-screen
production the player never sees.

Without it, a shop that sold out would stay empty until a caravan physically
arrived. The warehouse is what gives the economy inertia.

## Files

- **warehouse.go** — `City`, `Warehouse`, `Entry`, deposit and lookup.
- **withdraw.go** — `Withdraw` and `ReleaseToVendorsInRoom`.
- **tick.go** — the accrual timer.
- **persistence.go** — per-zone YAML, dirty tracking, load/save.

## Types

```go
type City struct      { /* the zone's warehouse configuration */ }
type Entry struct     { /* one item id and its quantity */ }
type Warehouse struct { /* zone + entries */ }
```

One warehouse per zone, created on demand.

## Public API

```go
func CityFor(zone string) (City, bool)
func WarehouseFor(zone string) *Warehouse
func AllWarehouses() []*Warehouse
func (w *Warehouse) StockOf(itemId int) int

func Deposit(zone string, itemId, qty int) bool
func Withdraw(zone string, itemId, qty int) int
func ReleaseToVendorsInRoom(zone string, roomId int, maxPerItem int) int

func Tick()
func LoadAll()
func SaveAll()
func SaveDirty()
func SetBaseDirForTest(dir string)
func ResetForTest()
```

`Deposit` returns a bool (accepted / rejected — capacity), `Withdraw` returns
the quantity actually taken, which may be less than asked. **Check both.**

`ReleaseToVendorsInRoom` is the retail hand-off: it pushes up to `maxPerItem`
of each stocked item into the vendors standing in that room.

## Accrual

`Tick()` calls `runAccrual()` when `accrualDue(round, accrualHours,
roundsPerDay)` says a cycle has elapsed. Accrual goes through the same
`deposit()` path as a real delivery but with `isAccrual = true`, so the two can
be distinguished for economy reporting — accrued goods are not evidence that
logistics is working.

## Persistence

State is written per zone and is **living state, not instance data**. Writes are
deferred: mutations call `markDirty(zone)` and `SaveDirty()` flushes only what
changed. `SaveAll()` forces everything.

## Gotchas

- **`Deposit` returning false means the goods were rejected, not stored.** A
  caller that ignores it silently destroys items.
- **`Withdraw` is partial-fill.** Requesting 10 and receiving 3 is normal.
- **`WarehouseFor` creates on demand** — it never returns nil, so it cannot be
  used to test whether a zone *has* a warehouse. Use `CityFor` for that.
- **Accrual is what keeps a badly-served zone alive.** If you are testing
  whether caravans are actually delivering, look at the accrual/delivery split
  in the economy dashboard, not at raw stock.
- **`SaveDirty` only writes zones that were marked.** Anything mutating a
  `*Warehouse` directly rather than through `Deposit`/`Withdraw` will not be
  persisted.

## Dependencies

`items`, `rooms`, `shops`, `configs`, `util`, `mudlog`.

## Consumers

`caravan`, `ferry` (factor loading), `forager`, `shops`,
`internal/economy/health`, and admin warehouse commands.
