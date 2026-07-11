# New Plymouth — Supply Runner (caravan engine generalization) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generalize the single-crew caravan engine to support a second,
independent New-Plymouth supply runner — Dobb (mob 9304) — who loops a Docks→
Crafting circuit, loading sea-imported feedstock at Dunmar's warehouse and
delivering it into the Crafting vendors' stock.

**Architecture:** Additive, isolation-tested engine code (a per-circuit registry,
a generalized mob lookup, a bounded import-loader, and a pure arrival-event router
hung off the existing `CaravanArrivalListener`) plus a self-perpetuating **looping**
patrol on Dobb (no traveling wagon, so no wagon-arrival trigger). The legacy
Thornwall/Stillwater oneshot path is left untouched. Wiring (patrol YAML + config +
vendor stock) depends on the Crafting district content existing (Plan 1).

**Tech Stack:** Go (`internal/caravan`, `internal/economy`), `go test`, the patrol
system (`patrol_id:` always-on patrols), DOGMud world YAML, local server boot.

**Spec:** `docs/superpowers/specs/2026-06-22-np-crafting-district-design.md` §5
(verified architecture + symbols). **Depends on:** Plan 1
(`2026-06-22-np-crafting-district.md`) — the Crafting vendor rooms/shops must exist
before Tasks 6–7.

---

## Verified facts (codegraph 2026-06-22 — do not re-derive)

- `RunnerMobId=359`, `WagonMobId=374`, `LeaderMobId=357` are single consts in
  `internal/caravan/wagon.go`. `FindRunnerInRoom(roomId)` (wagon.go:65) and
  `FindWagonInRoom(roomId)` (wagon.go:45) hardcode those template IDs.
- `bucketsForRunnerPatrol` (arrival_listener.go:199) + `runnerCircuitPatrols`
  (runner_completion_listener.go:10) drive the legacy oneshot circuit. **Leave
  them untouched.**
- `CaravanArrivalListener` (arrival_listener.go:30) is the single
  `PatrolWaypointArrival` consumer; it early-returns for unrecognized
  `ArrivalEvent`/patrol — so a new branch with new tags is non-regressing.
- `VisitVendorsInRoom(roomId, carrier, deliveryBuckets, pickupBuckets)`
  (visit.go:42): delivery moves carrier→vendor for items whose
  `economy.BucketFor(id)` ∈ deliveryBuckets and vendor `StockEntry.Current <
  MaxStock`. Passing `pickupBuckets=nil` ⇒ delivery-only.
- `applyPatrolPlan` (NewRound_IdleMobs_patrol.go:100) emits
  `events.PatrolWaypointArrival{MobInstanceId,PatrolId,WaypointIdx,RoomId,ArrivalEvent}`
  on **every** waypoint of **any** patrol, including always-on looping patrols.
- Test pattern: `newCargoTestMob(instId, mobId, name)` and `addItemToTestMob`
  (cargo_handoff_test.go:12/28) build bare mobs and `t.Skip` if `items.New` returns
  invalid (registry not loaded in unit context). New tests follow this.
- Dobb = mob **9304** (Docks zone, non-combatant, `maxwander:0`, no patrol yet).
  Dunmar = mob **9303** (warehouse master, has a shop). Both already built.

## Conventions

- Branch `feature/np-supply-runner` off `master` (Task 0). Conventional commits,
  trailer `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- Engine tests: `go test ./internal/caravan/...` (+ `./internal/economy/...`).
- Boot test (Conventions in Plan 1): wipe instances, `go run .`, grep
  `ERROR:.*PANIC|fatal error:` (never bare `panic`).

---

## Task 0: Branch + green baseline

- [ ] **Step 1: Branch** (from master, after Plan 1 is merged so Crafting vendors exist)

```bash
git checkout master && git checkout -b feature/np-supply-runner
```

- [ ] **Step 2: Baseline tests green**

```bash
go build ./... && go test ./internal/caravan/... ./internal/economy/...
```
Expected: build OK; existing caravan/economy tests PASS. Record the pass as the
baseline (we must not regress them).

---

## Task 1: The import-circuit registry

**Files:**
- Create: `internal/caravan/import_circuits.go`
- Test: `internal/caravan/import_circuits_test.go`

- [ ] **Step 1: Write the failing test**

```go
package caravan

import "testing"

func TestImportCircuits_NPDocksRegistered(t *testing.T) {
	c, ok := ImportCircuitFor("np_docks_runner_circuit")
	if !ok {
		t.Fatal("np_docks_runner_circuit not registered")
	}
	if c.RunnerMobId != 9304 {
		t.Errorf("runner mob = %d, want 9304 (Dobb)", c.RunnerMobId)
	}
	if c.DepotEvent != "np_runner_depot" || c.VendorEvent != "np_runner_vendor" {
		t.Errorf("event tags = %q/%q", c.DepotEvent, c.VendorEvent)
	}
	if len(c.DeliveryBuckets) == 0 || len(c.ImportItems) == 0 || c.LoadCap <= 0 {
		t.Errorf("circuit under-specified: %+v", c)
	}
}

func TestImportCircuits_UnknownNotFound(t *testing.T) {
	if _, ok := ImportCircuitFor("nope"); ok {
		t.Error("unknown circuit should not resolve")
	}
}
```

- [ ] **Step 2: Run — expect FAIL** (`ImportCircuitFor` undefined)

```bash
go test ./internal/caravan/ -run TestImportCircuits -v
```

- [ ] **Step 3: Implement**

```go
package caravan

// ImportCircuit describes a self-perpetuating, looping supply runner whose
// cargo originates from a stationary import source (sea import → warehouse),
// with NO traveling inter-zone wagon. Distinct from the legacy
// wagon-triggered oneshot circuits in runner_completion_listener.go.
type ImportCircuit struct {
	PatrolId        string   // the runner's looping (strict) patrol id
	RunnerMobId     int      // who runs it (Dobb = 9304)
	DepotEvent      string   // ArrivalEvent tag at the load stop
	VendorEvent     string   // ArrivalEvent tag at delivery stops
	DeliveryBuckets []string // economy buckets delivered to vendors
	ImportItems     []int    // the sea-import manifest (item IDs loaded at the depot)
	LoadCap         int      // max items carried per loop (bounds inventory growth)
}

// importCircuits is the registry of looping import circuits, keyed by patrol id.
var importCircuits = map[string]ImportCircuit{
	"np_docks_runner_circuit": {
		PatrolId:        "np_docks_runner_circuit",
		RunnerMobId:     9304, // Dobb
		DepotEvent:      "np_runner_depot",
		VendorEvent:     "np_runner_vendor",
		DeliveryBuckets: []string{"base", "overlap"},
		// Crafter feedstock imported by sea (base/overlap bucket items):
		ImportItems: []int{
			40001, // iron ingot (Halvard)
			40018, // steel ingot (Halvard)
			40020, // coal dust (Halvard)
			40006, // glass vial (Vesna/Edda)
			40004, // healer's root (Vesna)
			40012, // thread spool (Nessa)
			40007, // cloth strip (Nessa)
			40002, // leather strip (Corwin)
		},
		LoadCap: 24,
	},
}

// ImportCircuitFor returns the import circuit for a patrol id, if registered.
func ImportCircuitFor(patrolId string) (ImportCircuit, bool) {
	c, ok := importCircuits[patrolId]
	return c, ok
}
```

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/caravan/ -run TestImportCircuits -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/caravan/import_circuits.go internal/caravan/import_circuits_test.go
git commit -m "feat(caravan): import-circuit registry for looping supply runners"
```

---

## Task 2: Generalize the in-room mob lookup

**Files:**
- Modify: `internal/caravan/wagon.go`
- Test: `internal/caravan/wagon_test.go` (create if absent)

- [ ] **Step 1: Write the failing test**

```go
package caravan

import "testing"

func TestFindMobByTemplateInRoom_NilRoomReturnsNil(t *testing.T) {
	// Room 0 does not exist; loader returns nil; lookup must be nil-safe.
	if m := FindMobByTemplateInRoom(0, 9304); m != nil {
		t.Errorf("expected nil for missing room, got %v", m)
	}
}
```

- [ ] **Step 2: Run — expect FAIL** (`FindMobByTemplateInRoom` undefined)

```bash
go test ./internal/caravan/ -run TestFindMobByTemplateInRoom -v
```

- [ ] **Step 3: Implement — add the general helper, delegate the legacy func**

Add to `wagon.go`:

```go
// FindMobByTemplateInRoom returns the first co-located mob with the given
// template id in the room, or nil. Generalizes FindRunnerInRoom/FindWagonInRoom
// so multiple independent runner crews (e.g. the NP import runner) can be found.
func FindMobByTemplateInRoom(roomId, mobTemplateId int) *mobs.Mob {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return nil
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		if int(m.MobId) == mobTemplateId {
			return m
		}
	}
	return nil
}
```

Then make the existing `FindRunnerInRoom` delegate (keep its signature + callers):

```go
func FindRunnerInRoom(roomId int) *mobs.Mob {
	return FindMobByTemplateInRoom(roomId, RunnerMobId)
}
```

(Leave `FindWagonInRoom` as-is, or likewise delegate — optional, no behavior change.)

- [ ] **Step 4: Run — expect PASS + no regressions**

```bash
go test ./internal/caravan/ -run TestFind -v
go test ./internal/caravan/...
```

- [ ] **Step 5: Commit**

```bash
git add internal/caravan/wagon.go internal/caravan/wagon_test.go
git commit -m "feat(caravan): FindMobByTemplateInRoom — generalize runner lookup"
```

---

## Task 3: `LoadRunnerFromImport` — the bounded sea-import loader

**Files:**
- Create: `internal/caravan/import_load.go`
- Test: `internal/caravan/import_load_test.go`

- [ ] **Step 1: Write the failing test**

```go
package caravan

import "testing"

func TestLoadRunnerFromImport_RespectsCapAndSkipsWhenRegistryAbsent(t *testing.T) {
	runner := newCargoTestMob(80904, 9304, "Dobb")
	c := ImportCircuit{
		ImportItems: []int{40001, 40006, 40012},
		LoadCap:     5,
	}
	loaded := LoadRunnerFromImport(runner, c)
	if loaded == 0 {
		// Item registry not loaded in unit context — acceptable; smoke covers it.
		t.Skip("items.New returned invalid (registry not loaded); integrated smoke covers this")
	}
	if loaded > c.LoadCap || len(runner.Character.Items) > c.LoadCap {
		t.Errorf("load exceeded cap: loaded=%d items=%d cap=%d",
			loaded, len(runner.Character.Items), c.LoadCap)
	}
}

func TestLoadRunnerFromImport_NilOrEmptyNoOp(t *testing.T) {
	if got := LoadRunnerFromImport(nil, ImportCircuit{ImportItems: []int{40001}, LoadCap: 3}); got != 0 {
		t.Errorf("nil runner should load 0, got %d", got)
	}
	runner := newCargoTestMob(80905, 9304, "Dobb")
	if got := LoadRunnerFromImport(runner, ImportCircuit{ImportItems: nil, LoadCap: 3}); got != 0 {
		t.Errorf("empty manifest should load 0, got %d", got)
	}
	if got := LoadRunnerFromImport(runner, ImportCircuit{ImportItems: []int{40001}, LoadCap: 0}); got != 0 {
		t.Errorf("zero cap should load 0, got %d", got)
	}
}
```

- [ ] **Step 2: Run — expect FAIL** (`LoadRunnerFromImport` undefined)

```bash
go test ./internal/caravan/ -run TestLoadRunnerFromImport -v
```

- [ ] **Step 3: Implement**

```go
package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// LoadRunnerFromImport tops the runner's inventory up to circuit.LoadCap with
// fresh items drawn round-robin from circuit.ImportItems — the "sea import"
// abstraction for a stationary warehouse source. Bounded (LoadCap) so a looping
// runner can never accumulate unbounded cargo. Returns the number loaded.
func LoadRunnerFromImport(runner *mobs.Mob, c ImportCircuit) int {
	if runner == nil || len(c.ImportItems) == 0 || c.LoadCap <= 0 {
		return 0
	}
	loaded := 0
	maxTries := c.LoadCap*2 + len(c.ImportItems) // bound: avoids spin on invalid ids
	for tries := 0; tries < maxTries && len(runner.Character.Items) < c.LoadCap; tries++ {
		it := items.New(c.ImportItems[tries%len(c.ImportItems)])
		if !it.IsValid() {
			continue
		}
		if !runner.Character.StoreItem(it) {
			break // runner at carry capacity
		}
		loaded++
	}
	return loaded
}
```

- [ ] **Step 4: Run — expect PASS** (the cap test may `t.Skip` without a registry; the nil/empty test must pass)

```bash
go test ./internal/caravan/ -run TestLoadRunnerFromImport -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/caravan/import_load.go internal/caravan/import_load_test.go
git commit -m "feat(caravan): LoadRunnerFromImport — bounded sea-import cargo loader"
```

---

## Task 4: Arrival-event router + dispatch branch

**Files:**
- Create: `internal/caravan/import_arrival.go` (pure router + handler)
- Modify: `internal/caravan/arrival_listener.go` (additive branch in `CaravanArrivalListener`)
- Test: `internal/caravan/import_arrival_test.go`

- [ ] **Step 1: Write the failing test (pure router — no rooms needed)**

```go
package caravan

import "testing"

func TestClassifyImportArrival_RoutesByEventTag(t *testing.T) {
	c, _ := ImportCircuitFor("np_docks_runner_circuit")
	if got := classifyImportArrival(c, "np_runner_depot"); got != importLoad {
		t.Errorf("depot event → %v, want importLoad", got)
	}
	if got := classifyImportArrival(c, "np_runner_vendor"); got != importDeliver {
		t.Errorf("vendor event → %v, want importDeliver", got)
	}
	if got := classifyImportArrival(c, "something_else"); got != importNone {
		t.Errorf("unknown event → %v, want importNone", got)
	}
}
```

- [ ] **Step 2: Run — expect FAIL** (`classifyImportArrival`/`importLoad`… undefined)

```bash
go test ./internal/caravan/ -run TestClassifyImportArrival -v
```

- [ ] **Step 3: Implement the router + the room-touching handler**

`internal/caravan/import_arrival.go`:

```go
package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

type importAction int

const (
	importNone importAction = iota
	importLoad
	importDeliver
)

// classifyImportArrival is the pure routing decision for an import-circuit
// waypoint arrival, keyed on the authored ArrivalEvent tag.
func classifyImportArrival(c ImportCircuit, arrivalEvent string) importAction {
	switch arrivalEvent {
	case c.DepotEvent:
		return importLoad
	case c.VendorEvent:
		return importDeliver
	}
	return importNone
}

// handleImportArrival drives a single import-runner waypoint stop: load the
// runner at the depot, or deliver to the room's vendors at a vendor stop.
func handleImportArrival(c ImportCircuit, arrival events.PatrolWaypointArrival) {
	runner := FindMobByTemplateInRoom(arrival.RoomId, c.RunnerMobId)
	if runner == nil {
		return
	}
	switch classifyImportArrival(c, arrival.ArrivalEvent) {
	case importLoad:
		LoadRunnerFromImport(runner, c)
	case importDeliver:
		delivered, _ := VisitVendorsInRoom(arrival.RoomId, runner, c.DeliveryBuckets, nil)
		if msg := FormatVisitMessage(runner.Character.Name, delivered, nil); msg != "" {
			if r := rooms.LoadRoom(arrival.RoomId); r != nil {
				r.SendText(messaging.CategoryMobEmote, msg)
			}
		}
	}
	_ = mobs.GetInstance // keep mobs import if FormatVisitMessage signature changes
}
```

(Verify `FormatVisitMessage`'s signature before writing — it is used by the legacy
`handleVendorArrival`; match its argument list. Drop the `_ = mobs.GetInstance`
line if `mobs` ends up referenced elsewhere in the file.)

Then add the **additive branch** at the top of `CaravanArrivalListener`
(arrival_listener.go), right after the `arrival, ok := …` type assertion and
before the legacy `runnerCircuitPatrols` check:

```go
	// NP import circuits (looping runner, stationary source). Additive: new
	// ArrivalEvent tags the legacy paths don't recognize, so non-regressing.
	if c, isImport := ImportCircuitFor(arrival.PatrolId); isImport {
		handleImportArrival(c, arrival)
		return events.Continue
	}
```

- [ ] **Step 4: Run — expect PASS + whole-package green**

```bash
go test ./internal/caravan/ -run TestClassifyImportArrival -v
go build ./... && go test ./internal/caravan/...
```

- [ ] **Step 5: Commit**

```bash
git add internal/caravan/import_arrival.go internal/caravan/arrival_listener.go internal/caravan/import_arrival_test.go
git commit -m "feat(caravan): route NP import-circuit arrivals (load at depot, deliver at vendors)"
```

---

## Task 5: Engine green gate

- [ ] **Step 1: Full build + caravan/economy tests**

```bash
go build ./... && go test ./internal/caravan/... ./internal/economy/...
```
Expected: build OK; ALL tests PASS (no regression to legacy circuits). This closes
the **engine** portion — it is complete and isolation-tested independent of any
world content.

---

## Task 6: Wire the NP runner to the world (needs Plan 1 vendors)

**Files:**
- Create: `_datafiles/world/dogmud/patrols/new_plymouth/np_docks_runner_circuit.yaml`
- Modify: `_datafiles/world/dogmud/mobs/new_plymouth_docks/9304-dobb.yaml`
  (add `patrol_id: np_docks_runner_circuit`)
- Modify: `_datafiles/config.yaml` (`CaravanServedZones`)
- Verify: Crafting vendor `StockEntry`s exist (Plan 1 Task 8)

- [ ] **Step 1: Author Dobb's looping patrol YAML**

Determine the Docks **depot room** (Dunmar's warehouse spawn room — read
`9303-dunmar_wells.yaml` / its room spawn; e.g. the warehouse room in 5500–5519)
and the Crafting **vendor rooms** (5706 Orin, 5709 Halvard, 5711 Vesna, 5713 Edda,
5715 Nessa, 5717 Corwin, 5703 general stall). Then:

```yaml
id: np_docks_runner_circuit
description: "Dobb's daily delivery loop — load at Dunmar's warehouse, run imports out to the Crafting Quarter vendors."
loop_shape: strict
max_path_retries: 40
waypoints:
  - { room: <DOCKS_DEPOT_ROOM>, dwell_rounds: 12, arrival_event: np_runner_depot }
  - { room: 5703, dwell_rounds: 3, arrival_event: np_runner_vendor }
  - { room: 5706, dwell_rounds: 3, arrival_event: np_runner_vendor }
  - { room: 5709, dwell_rounds: 3, arrival_event: np_runner_vendor }
  - { room: 5711, dwell_rounds: 3, arrival_event: np_runner_vendor }
  - { room: 5713, dwell_rounds: 3, arrival_event: np_runner_vendor }
  - { room: 5715, dwell_rounds: 3, arrival_event: np_runner_vendor }
  - { room: 5717, dwell_rounds: 3, arrival_event: np_runner_vendor }
```
Every waypoint room must be `pathto`-reachable from the previous (the footbridge
seam Docks 5523 ↔ Crafting 5700 must exist — Plan 1 Task 1). `strict` loops back to
wp0, so the cycle re-loads at the depot each pass.

- [ ] **Step 2: Assign the patrol to Dobb**

Add `patrol_id: np_docks_runner_circuit` to `9304-dobb.yaml` (top-level, like other
always-on patrol mobs). Keep `maxwander: 0` (the patrol drives movement).

- [ ] **Step 3: Add the Crafting zone to `CaravanServedZones`**

In `_datafiles/config.yaml`, under `CaravanServedZones`, add:
```yaml
    - "New Plymouth Crafting"
```
(Display-name form, matching `"Stillwater"`/`"Thornwall City"`. This makes the
Crafting vendors' tier-30/20/10 slots runner-served rather than ticker-served;
base/40-50 still trickle — acceptable.)

- [ ] **Step 4: Confirm vendor StockEntries exist** — verify Plan 1 Task 8 declared
  the deliverable feedstock entries on each Crafting vendor (`Current:0`,
  `MaxStock:8`, `RestockQty:0`). If missing, add them now (delivery silently skips
  undeclared items).

- [ ] **Step 5: Boot-verify**

Boot-test recipe (wipe instances first). Expected: no panic; the patrol loads
(grep for patrol-load / `np_docks_runner_circuit`); no schedule/patrol validator
errors. Dobb's existing schedule (if any) must not conflict with the patrol — if it
does, prefer `patrol_id` (always-on) per the patrol system, or gate via a schedule
segment `activity: patrol` (Dobb currently has no schedule, so direct `patrol_id`
is correct).

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/patrols/new_plymouth/np_docks_runner_circuit.yaml \
        _datafiles/world/dogmud/mobs/new_plymouth_docks/9304-dobb.yaml _datafiles/config.yaml
git commit -m "feat(np-supply): Dobb's looping import circuit (Docks depot → Crafting vendors)"
```

---

## Task 7: In-game delivery smoke test

- [ ] **Step 1: Observe a full delivery cycle**

Boot the server (instances wiped). As admin, watch Dobb: confirm he loads at the
depot (`np_runner_depot` dwell), walks the footbridge into Crafting, and stops at
each vendor (`np_runner_vendor`). Before/after, inspect a Crafting vendor's stock
(buy menu or shop file under `_datafiles/world/dogmud/shops/new_plymouth_crafting/`)
and confirm `Current` for the delivered feedstock **climbs** after his stop. Watch
the room for the delivery emote (`FormatVisitMessage`).

- [ ] **Step 2: Confirm no leak into the legacy economy**

Verify Lars / the Thornwall↔Stillwater caravan still behaves (no Dobb interference):
grep the boot log for the legacy caravan lines; optionally observe a Lars stop.

- [ ] **Step 3: Record evidence** — note the stock deltas + the cycle in the build
  memory / a short report. Fix any path-retry/fallback issues (if Dobb falls back to
  `pathto home`, a waypoint is unreachable — fix the seam/coords).

---

## Task 8: Merge to master (hold push)

- [ ] **Step 1: Merge `--no-ff`**

```bash
git checkout master
git merge --no-ff feature/np-supply-runner -m "Merge: New Plymouth supply runner (caravan engine generalization)"
```

- [ ] **Step 2: Update memory** — record the engine generalization + the working
  Docks→Crafting delivery leg in `project_new_plymouth_build.md`; note that
  Merchant/Temple/Common vendors join Dobb's loop as those districts land (extend
  the patrol YAML + their StockEntries). **Do NOT push** (capital push policy).

---

## Self-Review (completed during planning)

- **Spec coverage:** §5.3.1 registry → Task 1; §5.3.2 generalized lookup → Task 2;
  §5.3.4 `LoadRunnerFromImport` → Task 3; §5.3.3 dispatch → Task 4; §5.3.5 patrol
  → Task 6.1–2; §5.3.7 config → Task 6.3; §5.3.6 vendor stock → Task 6.4 (+ Plan 1
  Task 8); §5.3.8 smoke → Task 7; §5.4 TDD seams → Tasks 1/3/4 are unit-tested in
  isolation.
- **Placeholder scan:** `<DOCKS_DEPOT_ROOM>` in Task 6.1 is a deliberate
  resolve-at-build value with an explicit instruction to read it from Dunmar's
  spawn — not a hidden TODO. `FormatVisitMessage` signature is flagged for
  verification before use. No other placeholders.
- **Type consistency:** `ImportCircuit`, `ImportCircuitFor`, `FindMobByTemplateInRoom`,
  `LoadRunnerFromImport`, `classifyImportArrival`/`importLoad`/`importDeliver`/
  `importNone`, and `handleImportArrival` are named identically wherever referenced
  across Tasks 1–4. `RunnerMobId=9304` (Dobb) is consistent with the registry and
  the patrol/mob wiring.
