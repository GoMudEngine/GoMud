# Ferry Stage 3 — Warehouse Buffer Pools — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persistent, backend-only warehouse inventory pools in New Plymouth and The Confluence that soak up surplus (end-of-circuit carrier leftovers + a slow ambient accrual trickle) under high per-item caps, with counters and stock captured into the hourly economy snapshot — reserves that Stage 4's drawdown will spend.

**Architecture:** New `internal/warehouse` package (shops-persistence pattern: living-economy YAML state at `_datafiles/world/dogmud/warehouses/`, gitignored like `shops/`). Two fill feeds: (1) overflow capture at the **end of a delivery circuit** — ferry factors bank leftover bucketed cargo when they return to a warehouse-city dock, and the NP import runner banks leftovers at its depot-load stop (mid-circuit leftovers are NOT captured — they're still deliverable at later stops); (2) ambient accrual — a seeded per-city item list trickles upward on a game-hour cadence. All bounded by a flat per-item cap. Backend forever: no player access, no rooms, no new NPCs.

**Tech Stack:** Go (GoMud engine), YAML persistence, existing economy buckets, ferry/caravan integration points built in Stage 2.

**Spec:** `docs/superpowers/specs/2026-07-03-ferry-system-design.md` (Stage 3). Stages 1+2 merged (`86d10731b`, `f8608f402`).

---

## Verified engine facts (do not re-derive)

- Shutdown save hooks: `main.go:518-519` (`forager.SaveAllThroughputs()`, `caravan.SaveAllThroughputs()`) — `warehouse.SaveAll()` goes beside them.
- `.gitignore:9` has `_datafiles/**/shops` — warehouses need the identical line (never committed, never deployed over).
- Shops persistence pattern to mirror: `internal/shops/persistence.go` (path building via `configs.GetFilePathsConfig().DataFiles.String()`, `os.MkdirAll` + `yaml.Marshal` + `os.WriteFile` on save, load-on-miss, in-memory cache + mutex, `ClearCache()` for tests).
- Balance config: `internal/configs/config.balance.go` uses `ConfigInt`/`ConfigFloat`/`ConfigBool` with `yaml:"...,omitempty"` tags; defaults are applied in the section's `Validate()` (read it for the exact idiom before adding fields). GamePlay toggles live in `config.gameplay.go` (`FerriesEnabled` precedent at ~line 23).
- Ferry Stage 2 integration points (all in `internal/ferry/factor.go`): the Returning→Waiting transition calls `ensureFactorLoaded(factor, c, st.PortIdx)` — leftover banking goes immediately BEFORE that call. Dock zone: `rooms.LoadRoom(dockRoomId).Zone`.
- Caravan integration point: `internal/caravan/import_arrival.go` `handleImportArrival` `importLoad` case calls `LoadRunnerFromImport(runner, c)` — leftover banking goes immediately before it. Dobb's depot 5506 is in zone "New Plymouth Docks".
- `economy.BucketFor(itemId) string` ("" = unbucketed → never bank). Import direction: `warehouse` may import `economy`/`items`/`configs`/`mudlog`/`util` ONLY — never `ferry`/`caravan`/`shops` (those import warehouse).
- `mob.Character.Items []items.Item`; `mob.Character.RemoveItem(item) `— walk in REVERSE when removing (index-safety, same as visit.go's deliver pass).
- `util.GetRoundCount()`; `configs.GetTimingConfig().RoundsPerDay` (900 live; 1 game-hour = 37.5 rounds).
- NewRound hook registration: `internal/hooks/hooks.go` (~line 46, beside FerryTick).
- Snapshot shape: `internal/economy/health/snapshot.go` (`Snapshot` struct, yaml+json tags both required — dashboard JS wants lowercase json names); capture walk in `capture.go` `CaptureSnapshot()`.
- Zone display names: `New Plymouth Docks`, `The Confluence` (ConvertForFilename → `new_plymouth_docks`, `the_confluence`).

## Design decisions locked here

- **Capture point = end of circuit only.** Mid-circuit leftovers stay aboard (later stops still need them — Dobb's manifest is the union of all his vendors' needs). Ferry factors bank at Returning→Waiting; the NP import runner banks at its depot-load event. Stillwater has no warehouse (spec: NP + Confluence) — Stillwater-docked leftovers stay aboard and count against the reload cap, exactly as today. Foragers are untouched (out of scope).
- **Flat per-item cap** `Balance.WarehouseItemCap` (default **400**) instead of the spec's sketched "multiplier of vendor MaxStock" — vendor MaxStock varies per slot, a flat cap is the same magnitude (20–60× typical vendor caps) and much simpler. Deviation noted per spec's "config default" latitude.
- **Accrual**: every `Balance.WarehouseAccrualHours` game-hours (default **2** → 75 rounds), +1 of each seeded item per city, capped. Seeds are per-city lists in the Go registry (importCircuits precedent).
- Persistence: save-on-mutation (capture/accrual batch) + `SaveAll()` at shutdown; load at boot after ferry loading; missing dir/files → fresh zero-value warehouses.

## File structure

```
internal/warehouse/warehouse.go        Types, city registry, Deposit/cap logic, queries
internal/warehouse/warehouse_test.go
internal/warehouse/persistence.go      LoadAll/SaveAll/save-one (shops pattern)
internal/warehouse/persistence_test.go
internal/warehouse/tick.go             Ambient accrual on cadence
internal/warehouse/tick_test.go
internal/hooks/NewRound_WarehouseTick.go
internal/hooks/hooks.go                (modify: register)
internal/ferry/factor.go               (modify: bank leftovers at Returning→Waiting)
internal/caravan/import_arrival.go     (modify: bank leftovers at depot load)
internal/economy/health/snapshot.go    (modify: WarehouseSnapshot + Snapshot.Warehouses)
internal/economy/health/capture.go     (modify: captureWarehouses)
internal/economy/health/capture_warehouse_test.go
internal/configs/config.gameplay.go    (modify: WarehousesEnabled)
internal/configs/config.balance.go     (modify: WarehouseItemCap, WarehouseAccrualHours)
_datafiles/config.yaml                 (modify: 3 knobs)
.gitignore                             (modify: warehouses line)
main.go                                (modify: LoadAll at boot, SaveAll at shutdown)
```

---

### Task 1: Config knobs + gitignore

**Files:**
- Modify: `internal/configs/config.gameplay.go`, `internal/configs/config.balance.go`, `_datafiles/config.yaml`, `.gitignore`

- [ ] **Step 1:** `config.gameplay.go` — under `FerriesEnabled` (~line 23):

```go
	WarehousesEnabled ConfigBool `yaml:"WarehousesEnabled"` // Master toggle for warehouse buffer pools (Stage 3 ferry system)
```

- [ ] **Step 2:** `config.balance.go` — READ the Balance struct + its `Validate()` first to match the default-setting idiom exactly, then add near the Shop* knobs:

```go
	WarehouseItemCap      ConfigInt `yaml:"WarehouseItemCap,omitempty"`      // Per-item stock cap in city warehouses (default 400)
	WarehouseAccrualHours ConfigInt `yaml:"WarehouseAccrualHours,omitempty"` // Game-hours between ambient accrual ticks (default 2)
```

…and in `Validate()`, following the file's existing zero→default idiom:

```go
	if b.WarehouseItemCap <= 0 {
		b.WarehouseItemCap = 400
	}
	if b.WarehouseAccrualHours <= 0 {
		b.WarehouseAccrualHours = 2
	}
```

- [ ] **Step 3:** `_datafiles/config.yaml` — `FerriesEnabled` sibling comment style:

```yaml
  # - WarehousesEnabled -
  # Master toggle for the city warehouse buffer pools (Stage 3 ferry system).
  WarehousesEnabled: true
```

…and in the Balance section near the Shop* knobs: `WarehouseItemCap: 400` and `WarehouseAccrualHours: 2` with one-line comments.

- [ ] **Step 4:** `.gitignore` — directly under the `_datafiles/**/shops` line:

```
_datafiles/**/warehouses
```

- [ ] **Step 5:** `go build ./...` clean. Commit:

```bash
git add internal/configs/ _datafiles/config.yaml .gitignore
git commit -m "feat(warehouse): config knobs + gitignored living-state dir"
```

---

### Task 2: Warehouse core — types, registry, Deposit

**Files:**
- Create: `internal/warehouse/warehouse.go`
- Test: `internal/warehouse/warehouse_test.go`

- [ ] **Step 1: Failing tests.**

```go
package warehouse

import "testing"

func TestCityFor_RegisteredZones(t *testing.T) {
	if _, ok := CityFor("New Plymouth Docks"); !ok {
		t.Fatal("NP Docks should be a warehouse city")
	}
	if _, ok := CityFor("The Confluence"); !ok {
		t.Fatal("The Confluence should be a warehouse city")
	}
	if _, ok := CityFor("Stillwater"); ok {
		t.Fatal("Stillwater must NOT be a warehouse city (spec: NP + Confluence)")
	}
}

func TestDeposit_AddsAndCaps(t *testing.T) {
	ResetForTest()
	// 40001 (iron ingot) is bucketed "base".
	if !Deposit("New Plymouth Docks", 40001, 3) {
		t.Fatal("deposit of a bucketed item should be accepted")
	}
	w := WarehouseFor("New Plymouth Docks")
	if w == nil || w.StockOf(40001) != 3 {
		t.Fatalf("expected 3 stocked, got %+v", w)
	}
	if w.CapturedCount != 3 {
		t.Fatalf("CapturedCount = %d, want 3", w.CapturedCount)
	}
	// Cap clamp: deposit beyond cap accepts only up to cap.
	Deposit("New Plymouth Docks", 40001, 1000)
	if got := w.StockOf(40001); got != testItemCap {
		t.Fatalf("stock = %d, want cap %d", got, testItemCap)
	}
}

func TestDeposit_RejectsUnbucketedAndUnknownZone(t *testing.T) {
	ResetForTest()
	if Deposit("New Plymouth Docks", 99999, 1) {
		t.Fatal("unbucketed item must be rejected")
	}
	if Deposit("Stillwater", 40001, 1) {
		t.Fatal("non-warehouse zone must be rejected")
	}
}
```

(`testItemCap`: the tests must not read live config — give `Deposit` an internal cap resolver variable `itemCapFn func() int` defaulting to the Balance read, overridden in tests via `ResetForTest()` setting it to a constant `testItemCap = 400`. Keep the hook unexported + a `ResetForTest` that clears state AND restores/sets the test cap.)

- [ ] **Step 2:** Run `go test ./internal/warehouse/ -v` → FAIL (package missing).

- [ ] **Step 3: Implement `warehouse.go`.**

```go
// Package warehouse implements the Stage 3 city warehouse buffer pools:
// persistent, backend-only inventory that soaks up end-of-circuit carrier
// surplus and a slow ambient accrual, to be spent by Stage 4 drawdown.
// Spec: docs/superpowers/specs/2026-07-03-ferry-system-design.md.
// Player access: NONE, forever. Only caravans/runners/factors touch this.
package warehouse

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/economy"
)

// City is a warehouse city's static config.
type City struct {
	Zone         string // display zone name (rooms' Zone field)
	AccrualItems []int  // ambient-accrual seed list (all must be bucketed)
}

// cities is the registry (Go-map precedent: caravan importCircuits).
var cities = map[string]City{
	"New Plymouth Docks": {
		Zone: "New Plymouth Docks",
		// base/thornwall staples the capital "produces" off-screen.
		AccrualItems: []int{40001, 40003, 40006, 40012, 40018, 40019, 40021},
	},
	"The Confluence": {
		Zone: "The Confluence",
		// river goods (confluence bucket, Stage 2).
		AccrualItems: []int{40123, 40124, 40125, 40126},
	},
}

// Entry is one stocked item.
type Entry struct {
	ItemId  int `yaml:"item_id"`
	Current int `yaml:"current"`
}

// Warehouse is one city's persistent pool + counters.
type Warehouse struct {
	Zone          string  `yaml:"zone"`
	Stock         []Entry `yaml:"stock,omitempty"`
	CapturedCount int     `yaml:"captured_count,omitempty"` // via overflow capture
	AccruedCount  int     `yaml:"accrued_count,omitempty"`  // via ambient accrual
	DrawnCount    int     `yaml:"drawn_count,omitempty"`    // Stage 4 (0 until then)
}

var (
	mu         sync.Mutex
	warehouses = map[string]*Warehouse{}
	dirty      = map[string]bool{}

	// itemCapFn resolves the per-item cap; swapped in tests so unit tests
	// never read live config.
	itemCapFn = func() int { return int(configs.GetBalanceConfig().WarehouseItemCap) }
)

// CityFor returns the city config for a zone, if it hosts a warehouse.
func CityFor(zone string) (City, bool) {
	c, ok := cities[zone]
	return c, ok
}

// WarehouseFor returns the live pool for a zone (creating a zero-value one
// for registered cities), or nil for non-warehouse zones.
func WarehouseFor(zone string) *Warehouse {
	if _, ok := cities[zone]; !ok {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	return getOrCreateLocked(zone)
}

func getOrCreateLocked(zone string) *Warehouse {
	if w, ok := warehouses[zone]; ok {
		return w
	}
	w := &Warehouse{Zone: zone}
	warehouses[zone] = w
	return w
}

// StockOf returns the current count of an item.
func (w *Warehouse) StockOf(itemId int) int {
	for i := range w.Stock {
		if w.Stock[i].ItemId == itemId {
			return w.Stock[i].Current
		}
	}
	return 0
}

// Deposit banks qty of an item into a city's warehouse via CAPTURE.
// Returns false (nothing banked) for non-warehouse zones or unbucketed
// items. Accepts partially when the cap truncates. Counter: CapturedCount.
func Deposit(zone string, itemId int, qty int) bool {
	return deposit(zone, itemId, qty, false)
}

// accrue is the ambient-accrual variant (counter: AccruedCount).
func accrue(zone string, itemId int, qty int) bool {
	return deposit(zone, itemId, qty, true)
}

func deposit(zone string, itemId int, qty int, isAccrual bool) bool {
	if qty <= 0 {
		return false
	}
	if _, ok := cities[zone]; !ok {
		return false
	}
	if economy.BucketFor(itemId) == `` {
		return false
	}
	mu.Lock()
	defer mu.Unlock()
	w := getOrCreateLocked(zone)

	cap := itemCapFn()
	cur := 0
	idx := -1
	for i := range w.Stock {
		if w.Stock[i].ItemId == itemId {
			cur, idx = w.Stock[i].Current, i
			break
		}
	}
	room := cap - cur
	if room <= 0 {
		return false
	}
	add := qty
	if add > room {
		add = room
	}
	if idx < 0 {
		w.Stock = append(w.Stock, Entry{ItemId: itemId, Current: add})
	} else {
		w.Stock[idx].Current += add
	}
	if isAccrual {
		w.AccruedCount += add
	} else {
		w.CapturedCount += add
	}
	dirty[zone] = true
	return true
}

// AllWarehouses returns the live pools for every registered city (creating
// zero-value ones as needed). Order is not guaranteed.
func AllWarehouses() []*Warehouse {
	mu.Lock()
	defer mu.Unlock()
	out := make([]*Warehouse, 0, len(cities))
	for zone := range cities {
		out = append(out, getOrCreateLocked(zone))
	}
	return out
}

// ResetForTest clears all state and pins the item cap to 400 so unit
// tests never read live config.
func ResetForTest() {
	mu.Lock()
	defer mu.Unlock()
	warehouses = map[string]*Warehouse{}
	dirty = map[string]bool{}
	itemCapFn = func() int { return 400 }
}
```

(Also define `const testItemCap = 400` in the TEST file, not production code.)

- [ ] **Step 4:** `go test ./internal/warehouse/ -v` → PASS. **Step 5:** Commit `feat(warehouse): core pool types, city registry, capped deposits`.

---

### Task 3: Persistence + boot/shutdown wiring

**Files:**
- Create: `internal/warehouse/persistence.go` + `persistence_test.go`
- Modify: `main.go`

- [ ] **Step 1: Failing test** (mirror `internal/shops/persistence_test.go`'s YAML round-trip style — read it first; it redirects the data dir via `configs.AddOverlayOverrides` or writes to temp — copy that isolation approach exactly):

```go
func TestSaveLoad_RoundTrip(t *testing.T) {
	// (temp-dir isolation per the shops persistence_test pattern)
	ResetForTest()
	Deposit("The Confluence", 40123, 5)
	if err := saveOne("The Confluence"); err != nil {
		t.Fatalf("save: %v", err)
	}
	ResetForTest()
	LoadAll()
	w := WarehouseFor("The Confluence")
	if w.StockOf(40123) != 5 || w.CapturedCount != 5 {
		t.Fatalf("round trip lost state: %+v", w)
	}
}
```

- [ ] **Step 2:** FAIL run. **Step 3: Implement `persistence.go`** — mirror `internal/shops/persistence.go` exactly: `warehousePath(zone) = DataFiles + "/warehouses/" + ConvertForFilename(zone) + ".yaml"`; `saveOne(zone)` (marshal under `mu`-safe snapshot, MkdirAll, WriteFile); `SaveAll()` saves every dirty city then clears dirty (also called at shutdown — save all registered regardless of dirty is fine and simpler: pick one, document); `SaveDirty()` used by the tick; `LoadAll()` reads each registered city's file if present (missing file → keep zero-value), sets `warehouses[zone]`, clears dirty. Log one `mudlog.Info("warehouse.LoadAll()", "loadedCount", n)` line (Pre-Push-SOP-greppable format).

- [ ] **Step 4:** PASS + `go build ./...`. **Step 5:** `main.go`: add `warehouse.LoadAll()` right after `ferry.LoadDataFiles()` (~line 1424) and `warehouse.SaveAll()` beside `caravan.SaveAllThroughputs()` (line ~519); import the package. **Step 6:** Commit `feat(warehouse): YAML persistence + boot/shutdown wiring`.

---

### Task 4: Ambient accrual tick + hook

**Files:**
- Create: `internal/warehouse/tick.go` + `tick_test.go`, `internal/hooks/NewRound_WarehouseTick.go`
- Modify: `internal/hooks/hooks.go`

- [ ] **Step 1: Failing tests** (pure cadence math + accrual application):

```go
func TestAccrualDue(t *testing.T) {
	// 2 game-hours at rpd 900 → every 75 rounds.
	if !accrualDue(150, 2, 900) || accrualDue(151, 2, 900) {
		t.Fatal("cadence math wrong")
	}
}

func TestRunAccrual_SeedsAndCaps(t *testing.T) {
	ResetForTest()
	runAccrual()
	w := WarehouseFor("The Confluence")
	if w.StockOf(40123) != 1 || w.AccruedCount < 1 {
		t.Fatalf("accrual didn't seed: %+v", w)
	}
	for i := 0; i < 1000; i++ {
		runAccrual()
	}
	if got := w.StockOf(40123); got != 400 {
		t.Fatalf("cap not enforced: %d", got)
	}
}
```

- [ ] **Step 2:** FAIL. **Step 3: Implement `tick.go`:**

```go
package warehouse

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// accrualDue: pure cadence check (testable without globals).
func accrualDue(round uint64, accrualHours int, roundsPerDay int) bool {
	interval := uint64(accrualHours * roundsPerDay / 24)
	if interval == 0 {
		return false
	}
	return round%interval == 0
}

// runAccrual adds +1 of each seeded item to each city, capped.
func runAccrual() {
	for zone, c := range cities {
		for _, itemId := range c.AccrualItems {
			accrue(zone, itemId, 1)
		}
	}
}

// Tick runs once per round from the NewRound hook.
func Tick() {
	if !bool(configs.GetGamePlayConfig().WarehousesEnabled) {
		return
	}
	now := util.GetRoundCount()
	if accrualDue(now, int(configs.GetBalanceConfig().WarehouseAccrualHours), int(configs.GetTimingConfig().RoundsPerDay)) {
		runAccrual()
	}
	SaveDirty() // no-op when nothing changed this round
}
```

Hook file (FerryTick pattern) + registration in hooks.go beside FerryTick:

```go
	events.RegisterListener(events.NewRound{}, WarehouseTick) // Warehouse pools: accrual + dirty save
```

- [ ] **Step 4:** PASS + build. **Step 5:** Commit `feat(warehouse): ambient accrual tick + NewRound hook`.

---

### Task 5: Overflow capture — ferry + caravan integration

**Files:**
- Modify: `internal/ferry/factor.go`, `internal/caravan/import_arrival.go`
- Test: extend `internal/ferry/factor_test.go` IF a pure seam exists; otherwise the warehouse.Deposit unit tests + Task 7 live checks cover the glue (document the choice in the commit message).

- [ ] **Step 1: Ferry banking.** In `internal/ferry/factor.go`, add:

```go
// bankLeftoverCargo consigns a factor's undelivered bucketed cargo to the
// dock city's warehouse (Stage 3 overflow capture). Called ONLY at the
// end of a circuit (Returning→Waiting), never mid-circuit — later stops
// still need mid-circuit cargo. Non-warehouse cities (Stillwater): no-op,
// leftovers stay aboard and count against the reload cap as before.
func bankLeftoverCargo(factor *mobs.Mob, dockRoomId int) {
	dock := rooms.LoadRoom(dockRoomId)
	if dock == nil {
		return
	}
	if _, ok := warehouse.CityFor(dock.Zone); !ok {
		return
	}
	banked := 0
	for i := len(factor.Character.Items) - 1; i >= 0; i-- {
		it := factor.Character.Items[i]
		if warehouse.Deposit(dock.Zone, it.ItemId, 1) {
			factor.Character.RemoveItem(it)
			banked++
		}
	}
	if banked > 0 {
		dock.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(`%s consigns the unsold remainder of the cargo to the warehouse.`, factor.Character.Name))
	}
}
```

Call it at the Returning→Waiting transition, immediately BEFORE `ensureFactorLoaded(...)` (find the exact site — the arrival branch that flips Phase to Waiting and reloads; there is exactly one). Also call it in the stuck-reset path before ITS reload IF that path reloads (read it; keep behavior symmetric).

- [ ] **Step 2: Caravan banking.** In `internal/caravan/import_arrival.go`, `handleImportArrival` `importLoad` case — before `LoadRunnerFromImport(runner, c)`:

```go
		// Stage 3 overflow capture: leftovers surviving a full vendor loop
		// mean the city's vendors were full — bank them rather than letting
		// them ride forever. No-op outside warehouse cities.
		if room := rooms.LoadRoom(arrival.RoomId); room != nil {
			if _, ok := warehouse.CityFor(room.Zone); ok {
				for i := len(runner.Character.Items) - 1; i >= 0; i-- {
					it := runner.Character.Items[i]
					if warehouse.Deposit(room.Zone, it.ItemId, 1) {
						runner.Character.RemoveItem(it)
					}
				}
			}
		}
```

(Match the file's existing import style; `rooms` is already imported there.)

- [ ] **Step 3:** `go build ./... && go test ./internal/ferry/ ./internal/caravan/ ./internal/warehouse/ -count=1` → green (no behavior change unit-visible; existing tests must not break). **Step 4:** Commit `feat(warehouse): end-of-circuit overflow capture from ferry factors + NP import runner`.

---

### Task 6: Snapshot capture

**Files:**
- Modify: `internal/economy/health/snapshot.go`, `internal/economy/health/capture.go`
- Test: `internal/economy/health/capture_warehouse_test.go`

- [ ] **Step 1: Failing test** (black-box `package health_test`, sibling style of `capture_ferry_test.go`):

```go
func TestCaptureSnapshot_Warehouses(t *testing.T) {
	warehouse.ResetForTest()
	warehouse.Deposit("The Confluence", 40123, 7)
	snap := health.CaptureSnapshot()
	var found *health.WarehouseSnapshot
	for i := range snap.Warehouses {
		if snap.Warehouses[i].Zone == "The Confluence" {
			found = &snap.Warehouses[i]
		}
	}
	if found == nil {
		t.Fatal("no Confluence warehouse row in snapshot")
	}
	if found.CapturedCount != 7 {
		t.Fatalf("CapturedCount = %d, want 7", found.CapturedCount)
	}
	var stock *health.WarehouseStockSnapshot
	for i := range found.Stock {
		if found.Stock[i].ItemId == 40123 {
			stock = &found.Stock[i]
		}
	}
	if stock == nil || stock.Current != 7 || stock.Bucket != "confluence" {
		t.Fatalf("bad stock row: %+v", stock)
	}
}
```

- [ ] **Step 2:** FAIL. **Step 3: Implement.** `snapshot.go` additions (yaml+json tags, lowercase json, matching sibling style):

```go
// WarehouseSnapshot captures one city warehouse pool (Stage 3).
type WarehouseSnapshot struct {
	Zone          string                   `yaml:"zone"           json:"zone"`
	Stock         []WarehouseStockSnapshot `yaml:"stock"          json:"stock"`
	CapturedCount int                      `yaml:"captured_count" json:"captured_count"`
	AccruedCount  int                      `yaml:"accrued_count"  json:"accrued_count"`
	DrawnCount    int                      `yaml:"drawn_count"    json:"drawn_count"`
}

type WarehouseStockSnapshot struct {
	ItemId  int    `yaml:"item_id" json:"item_id"`
	Bucket  string `yaml:"bucket"  json:"bucket"`
	Current int    `yaml:"current" json:"current"`
}
```

+ `Warehouses []WarehouseSnapshot \`yaml:"warehouses" json:"warehouses"\`` on `Snapshot`. In `capture.go`: `captureWarehouses()` walking `warehouse.AllWarehouses()` (Bucket via `economy.BucketFor`), called from `CaptureSnapshot()` (`snap.Warehouses = captureWarehouses()`). Import-cycle: health already imports ferry+caravan; warehouse is lower still — safe (verify).

- [ ] **Step 4:** PASS + full build/test. **Step 5:** Commit `feat(warehouse): pools in the hourly economy snapshot`.

---

### Task 7: Boot test + live verification (main loop + light harness)

- [ ] **Step 1:** Instance wipe SOP, `go run .` — expect `warehouse.LoadAll() loadedCount=0` (fresh), `ferry.LoadDataFiles() loadedCount=3`, no panics.
- [ ] **Step 2: Accrual check (fast):** leave the server running ~7 real minutes (≥2 accrual intervals at 75 rounds each). Then inspect `_datafiles/world/dogmud/warehouses/*.yaml` — both files exist, seeded items present with Current ≥ 2, `accrued_count` matching. (No client needed.)
- [ ] **Step 3: Capture check (slower, needs full circuits):** let the server run ~35–45 min (2-3 vessel cycles). The Stage 2 playtest showed destination slots hit their caps by circuit ~2; circuit 3's undeliverable cargo must appear in the warehouses: check `captured_count > 0` in `the_confluence.yaml` and/or `new_plymouth_docks.yaml` and the factor's dock emote in the server log ("consigns the unsold remainder"). If schedule timing makes this flaky to observe, drive it deterministically with a short harness session (smoketester: buy NOTHING, just wait and watch the dock at circuit 3 — or pre-fill vendor slots by admin-observing until caps are hit).
- [ ] **Step 4:** One manual snapshot (admin economy endpoint or wait for the hourly tick) — verify the `warehouses:` block appears in the newest `_datafiles/world/dogmud/economy/snapshots/*.yaml`.
- [ ] **Step 5:** Kill all GoMud/go processes. Commit any fixes: `fix(warehouse): live-verification findings`.

---

## Post-plan notes

- Stage 4 plug points established: `Warehouse.DrawnCount` + the `deposit(..., isAccrual)` split make the withdraw path symmetrical; drawdown will add `Withdraw(zone, itemId, qty)` and hook `ensureFactorLoaded`/`LoadRunnerFromImport`.
- The admin dashboard warehouse PANEL stays a follow-up (snapshot data now accumulates for it).
- Pre-push SOP before any prod push covers the whole ferry bundle (Stages 1-3).
