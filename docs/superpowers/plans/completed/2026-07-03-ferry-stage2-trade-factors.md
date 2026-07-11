# Ferry Stage 2 — Riding Trade Factors + Cross-Region Stock — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** NPC trade factors that visibly ride the Stage 1 ferries carrying regional export cargo, walk short vendor loops in each port city delivering by bucket, and thereby seed cross-region `RestockQty: 0` stock slots that only ferry deliveries refill — damping price spikes and opening player cargo-running.

**Architecture:** One duplex factor per route (3 total), homed at a dock, driven entirely by the ferry controller (no patrol executor entanglement): the controller teleports it aboard/ashore (TransportCompanions pattern) and walks it between vendor stops via `pathto`, delivering with the caravan `VisitVendorsInRoom` helper extended with opt-in missing-slot creation. Factors surface on the economy dashboard as CaravanSnapshot rows.

**Tech Stack:** Go (GoMud engine), existing caravan visit/throughput helpers, `internal/ferry` Stage 1 controller, YAML mob content.

**Spec:** `docs/superpowers/specs/2026-07-03-ferry-system-design.md` (Stage 2 scope). Stage 1 is merged (`86d10731b`).

---

## Verified engine facts (do not re-derive)

- **Mob room transport** (TransportCompanions pattern, `internal/hooks/companion_follow.go:55-78`): `curRoom.RemoveMob(instId)` → `destRoom.AddMob(instId)` → `mob.Character.RoomId = newRoomId`. Abort casts first if any; clear stale aggro. Ferry factors are non-combatant so the cast/aggro parts reduce to nothing, but keep the RemoveMob/AddMob/RoomId triple exactly.
- **`VisitVendorsInRoom(roomId int, wagon *mobs.Mob, deliveryBuckets, pickupBuckets []string) (delivered, pickedUp []ItemMove)`** (`internal/caravan/visit.go:42`): visits every shop mob in ONE room; DELIVER pass moves carrier items whose `economy.BucketFor(itemId)` ∈ deliveryBuckets into vendor stock, **but skips items the vendor has no StockEntry for** (`entry == nil → continue`, visit.go:85-88). Increments throughput under `(wagon.Zone, wagon.MobId)`. Pass `nil` pickupBuckets (Stage 2 does not drain vendors — pickup is a Stage 3 integration point).
- **`FormatVisitMessage(name string, delivered, pickedUp []ItemMove) string`** exists in caravan (used by `handleImportArrival`).
- **Shop persistence**: `shops.RegisterShop` ignores the template when a persisted file exists (only CraftSupport migrates) — template `shop:` edits do NOT reach existing shops. That is WHY this plan creates slots at delivery time instead of touching vendor templates.
- **StockEntry with `RestockQty: 0`** is never touched by the restock ticker (`Restock()` skips `RestockQty <= 0`) — created slots are ferry-fed only, per spec.
- **`mobs.FindLiveInstanceByHomeAndId(roomId, mobId int)`** (mobs.go:703) finds a live instance by HomeRoomId+MobId — how the controller locates its factor.
- **`m.IsEssential()`** returns true for mobs with group `caravan` (or `forager`, or a shop) — the room manager never unloads rooms containing them. Factors get `groups: [humanoid, caravan]`.
- **`systemNPCAnchorRooms`** (main.go:108) — rooms whose spawninfo fires at boot instead of first visit; consumed at main.go:1438. Factor home docks must be added.
- **`captureCaravans`** (`internal/economy/health/capture.go:165`) recognizes only caravan leaders via `SynthesizeStateForLeader` — factors need their own capture helper appending `CaravanSnapshot` rows (schema at `internal/economy/health/snapshot.go:76`; carrier cargo weight via `Character.GetCarriedWeight()` / `CarryCapacity()`; throughput via `caravan.GetThroughput(m.Zone, int(m.MobId))` — same keys visit.go writes).
- **Bucket map** (`internal/economy/buckets.go`): items not in the map return `""` and are SKIPPED by delivery. Confluence has no bucket yet — its exports need one. Drift test `TestBucketMap_AuditMatrixCoverage` ties the map to `docs/economy/mat-audit-matrix.md`.
- Stage 1 ferry facts: `ferry.AllRoutes()`, `ferry.RouteFor`, `StateAt(r, round, rpd) VesselState{Docked, PortIdx, RoundsUntilTransition}`; controller `Tick()` in `internal/ferry/controller.go` runs every round behind `FerriesEnabled`; routes: `stillwater_np_packet` (ports [4118, 5502], deck 6423), `stillwater_confluence_barge` (ports [4118, 6109], deck 6424), `confluence_np_barge` (ports [6109, 5502], deck 6425).
- Mob idle/pathing: mobs walk via `mob.Command("pathto <roomid>")` (schedules/patrols use this plumbing — **verify the exact arg form in `internal/mobcommands/pathto.go` before use**; `pathto home` definitely exists).

## ID + content assignments (this plan owns these)

| ID | What |
|----|------|
| mobs 9577 | A Lakeway Factor — rides the packet, home Stillwater pier 4118 |
| mobs 9578 | A Riverway Factor — rides the heron, home Stillwater pier 4118 |
| mobs 9579 | A Broadwater Factor — rides the broadbeam, home Confluence Barge Dock 6109 |

Re-verify with `python tools/id_inventory.py --type mobs` before authoring (9577-9579 free as of 2026-07-03).

**Vendor delivery stops** (verified shop mobs + spawn rooms, 2026-07-03):

| Port | Stops (rooms walked by arriving factors) | Vendors there |
|------|------------------------------------------|---------------|
| Stillwater (dock 4118) | 4105, 4106, 4125 | Storekeeper Wulf, Smith Brindle, Apothecary Ilsa |
| NP Docks (dock 5502) | 5505, 5508 | Dunmar Wells (Warehouse Row), Chandler Voss |
| Confluence (dock 6109) | 6110, 6128 | Pella the Fish-Trader, Varro the Importer |

**Export manifests** (every item MUST be bucketed or delivery silently skips):

| Port | Manifest (loaded when the factor is at this port) | Buckets |
|------|---------------------------------------------------|---------|
| Stillwater | 40057 lake mint, 40058 freshwater clam, 40059 lake-iron nodule, 40056 marsh willow bark | `stillwater` |
| NP | 40018 steel ingot, 40019 chain link, 40006 glass vial, 40021 copper wire | `thornwall` + `base` |
| Confluence | 40123 watercress, 40124 mussels (+ 40125/40126 IF they are goods-type items — Task 1 verifies) | `confluence` (NEW bucket, Task 1) |

## File structure

```
internal/economy/buckets.go               (modify: confluence bucket + AllBuckets)
docs/economy/mat-audit-matrix.md          (modify: confluence rows)
internal/economy/buckets_test.go          (modify if drift test needs rows)
internal/caravan/visit.go                 (modify: options struct + CreateMissingSlots)
internal/caravan/visit_test.go            (add tests)
internal/ferry/tradecircuit.go            TradeCircuit registry + boot validation
internal/ferry/tradecircuit_test.go
internal/ferry/factor.go                  Factor state machine + transport + load + deliver
internal/ferry/factor_test.go             Pure decision-function tests
internal/ferry/controller.go              (modify: drive factors from Tick)
internal/economy/health/capture.go        (modify: captureFerryFactors)
internal/economy/health/capture_ferry_test.go
main.go                                   (modify: anchor rooms 4118, 6109)
_datafiles/world/dogmud/mobs/stillwater/9577-a_lakeway_factor.yaml
_datafiles/world/dogmud/mobs/stillwater/9578-a_riverway_factor.yaml
_datafiles/world/dogmud/mobs/the_confluence/9579-a_broadwater_factor.yaml
_datafiles/world/dogmud/rooms/stillwater/4118.yaml     (modify: +2 spawninfo)
_datafiles/world/dogmud/rooms/the_confluence/6109.yaml (modify: +1 spawninfo)
tools/playtest/goals/ferry-stage2.yaml
```

---

### Task 1: The `confluence` supply bucket

**Files:**
- Modify: `internal/economy/buckets.go`
- Modify: `docs/economy/mat-audit-matrix.md`
- Test: existing `internal/economy/buckets_test.go` (drift test)

- [ ] **Step 1: Verify the candidate items.** Read `_datafiles/world/dogmud/items/materials-40000/40123-*.yaml`, `40124-*.yaml`, `40125-*.yaml`, `40126-*.yaml`. 40123/40124 are the River Road water forageables (watercress, mussels). Include 40125/40126 in the bucket ONLY if they are material/provision goods (not tools or non-tradables) — check their `type`/`is_component`/`value`. Record the decision.

- [ ] **Step 2: Read the drift test** (`TestBucketMap_AuditMatrixCoverage` in `internal/economy/buckets_test.go`) to learn exactly what it cross-checks, then update BOTH sides together:

In `internal/economy/buckets.go`, add to `itemBucket` (after the fernway block, matching comment style):

```go
	// Confluence bucket (river goods — ferry Stage 2)
	40123: "confluence", // watercress
	40124: "confluence", // river mussels
	// (+ 40125/40126 per Step 1 decision)
```

And in `AllBuckets()`:

```go
func AllBuckets() []string {
	return []string{"base", "stillwater", "thornwall", "fernway", "confluence", "overlap"}
}
```

Add matching rows to `docs/economy/mat-audit-matrix.md` in its established table format.

- [ ] **Step 3: Run the economy tests.**

Run: `go test ./internal/economy/... -count=1`
Expected: PASS (drift test green with the new rows).

- [ ] **Step 4: Commit.**

```bash
git add internal/economy/buckets.go docs/economy/mat-audit-matrix.md internal/economy/buckets_test.go
git commit -m "feat(economy): confluence supply bucket for ferry trade"
```

---

### Task 2: `VisitVendorsInRoom` opt-in missing-slot creation

**Files:**
- Modify: `internal/caravan/visit.go`
- Test: `internal/caravan/visit_test.go` (or the file where existing Visit tests live — find it first)

The deliver pass currently skips items a vendor has no StockEntry for. Add an options form that, when enabled, CREATES the entry (`RestockQty: 0` — ferry-fed only, never ticker-refilled) with a small MaxStock cap. Existing callers keep exact current behavior.

- [ ] **Step 1: Write the failing test.** Locate the existing test file for visit.go (`grep -rl "VisitVendorsInRoom" internal/caravan/*_test.go`) and match its fixture style (it seeds rooms/mobs/shops — reuse its helpers). Add:

```go
func TestVisitVendorsInRoomOpts_CreatesMissingSlot(t *testing.T) {
	// Arrange (adapt to the existing fixture helpers in this file):
	// - a room with one shop vendor whose ShopInventory has NO entry for item X
	// - a carrier mob holding item X, where BucketFor(X) is in deliveryBuckets
	// Act:
	//   delivered, _ := VisitVendorsInRoomOpts(roomId, carrier, VisitOpts{
	//       DeliveryBuckets:   []string{bucketOfX},
	//       CreateMissingSlots: true,
	//       NewSlotMaxStock:    6,
	//   })
	// Assert:
	// - len(delivered) == 1
	// - vendor shop now has a StockEntry for X with Current==1, MaxStock==6, RestockQty==0
}

func TestVisitVendorsInRoomOpts_NoCreateWhenDisabled(t *testing.T) {
	// Same arrangement, CreateMissingSlots: false →
	// len(delivered) == 0 and GetStock(X) == nil (legacy behavior preserved).
}
```

Write these as REAL tests against the file's existing fixtures (the skeleton above states the required behavior; the arrange section must be concrete code copied from the sibling tests' setup pattern).

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/caravan/ -run TestVisitVendorsInRoomOpts -v`
Expected: FAIL — undefined: VisitVendorsInRoomOpts / VisitOpts.

- [ ] **Step 3: Implement.** In `visit.go`:

```go
// VisitOpts controls a vendor-visit pass.
type VisitOpts struct {
	DeliveryBuckets []string
	PickupBuckets   []string
	// CreateMissingSlots lets a delivery CREATE a StockEntry the vendor
	// never stocked (RestockQty stays 0: the slot is carrier-fed only —
	// the restock ticker never touches RestockQty<=0 entries). Used by
	// ferry trade factors to seed cross-region goods. Legacy caravan and
	// forager deliveries keep this off so random cargo can't invent
	// shelf space.
	CreateMissingSlots bool
	NewSlotMaxStock    int // MaxStock for created slots (required when CreateMissingSlots)
}

// VisitVendorsInRoom preserves the legacy behavior (no slot creation).
func VisitVendorsInRoom(roomId int, wagon *mobs.Mob, deliveryBuckets, pickupBuckets []string) (delivered, pickedUp []ItemMove) {
	return VisitVendorsInRoomOpts(roomId, wagon, VisitOpts{
		DeliveryBuckets: deliveryBuckets,
		PickupBuckets:   pickupBuckets,
	})
}

func VisitVendorsInRoomOpts(roomId int, wagon *mobs.Mob, opts VisitOpts) (delivered, pickedUp []ItemMove) {
	// ... body is the current VisitVendorsInRoom body, with the deliver-pass
	// entry check changed from:
	//     entry := shop.GetStock(item.ItemId)
	//     if entry == nil || entry.Current >= entry.MaxStock { continue }
	// to:
	//     entry := shop.GetStock(item.ItemId)
	//     if entry == nil {
	//         if !opts.CreateMissingSlots || opts.NewSlotMaxStock <= 0 {
	//             continue
	//         }
	//         shop.Stock = append(shop.Stock, shops.StockEntry{
	//             ItemId:   item.ItemId,
	//             MaxStock: opts.NewSlotMaxStock,
	//             // RestockQty deliberately 0: ferry-fed only.
	//         })
	//         entry = shop.GetStock(item.ItemId)
	//     }
	//     if entry.Current >= entry.MaxStock { continue }
}
```

Mechanically: rename the existing function body to `VisitVendorsInRoomOpts`, replace `deliveryBuckets`/`pickupBuckets` reads with `opts.DeliveryBuckets`/`opts.PickupBuckets`, and apply the entry-creation change above. CHECK the `StockEntry` field list in `internal/shops/shopinventory.go` before writing the literal (there may be fields like `LastGrewRound` worth setting via an existing helper — if `AddStockAtRound` or similar creates entries properly, prefer calling it; read the Kerra-regression path in `internal/mobs/crafter.go` for the precedent).

- [ ] **Step 4: Run tests.**

Run: `go test ./internal/caravan/ -count=1`
Expected: PASS — new tests + all existing caravan tests (legacy wrapper unchanged behavior).

- [ ] **Step 5: Commit.**

```bash
git add internal/caravan/visit.go internal/caravan/visit_test.go
git commit -m "feat(caravan): VisitVendorsInRoomOpts — opt-in missing-slot creation for ferry trade"
```

---

### Task 3: TradeCircuit registry + boot validation

**Files:**
- Create: `internal/ferry/tradecircuit.go`
- Test: `internal/ferry/tradecircuit_test.go`

- [ ] **Step 1: Write the failing tests.**

```go
package ferry

import "testing"

func validCircuit() TradeCircuit {
	return TradeCircuit{
		RouteId:     "test_route",
		FactorMobId: 9577,
		HomePortIdx: 0,
		PortExports: [2][]int{{40057, 40058}, {40018, 40006}},
		PortStops:   [2][]int{{4105, 4106}, {5505, 5508}},
		PortDeliveryBuckets: [2][]string{
			{"thornwall", "base"}, // delivered AT port 0 (= port 1's export buckets)
			{"stillwater"},        // delivered AT port 1
		},
		LoadCap:         12,
		NewSlotMaxStock: 6,
	}
}

func TestTradeCircuitValidate_Valid(t *testing.T) {
	if err := validCircuit().Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestTradeCircuitValidate_Rejects(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*TradeCircuit)
	}{
		{"missing route", func(c *TradeCircuit) { c.RouteId = "" }},
		{"missing factor", func(c *TradeCircuit) { c.FactorMobId = 0 }},
		{"bad home port", func(c *TradeCircuit) { c.HomePortIdx = 2 }},
		{"empty exports port0", func(c *TradeCircuit) { c.PortExports[0] = nil }},
		{"empty stops port1", func(c *TradeCircuit) { c.PortStops[1] = nil }},
		{"zero loadcap", func(c *TradeCircuit) { c.LoadCap = 0 }},
		{"zero slot max", func(c *TradeCircuit) { c.NewSlotMaxStock = 0 }},
	}
	for _, tc := range cases {
		c := validCircuit()
		tc.mutate(&c)
		if err := c.Validate(); err == nil {
			t.Errorf("%s: expected error, got nil", tc.name)
		}
	}
}
```

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/ferry/ -run TestTradeCircuit -v`
Expected: FAIL — undefined: TradeCircuit.

- [ ] **Step 3: Implement `tradecircuit.go`.**

```go
package ferry

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/economy"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// TradeCircuit describes one duplex trade factor riding one ferry route:
// at each port it loads that port's export manifest, rides to the other
// port, walks that port's vendor stops delivering by bucket, returns to
// the dock, and waits for the boat home.
type TradeCircuit struct {
	RouteId             string
	FactorMobId         int
	HomePortIdx         int         // where the factor spawns/recovers (0 or 1)
	PortExports         [2][]int    // manifest loaded AT port i
	PortStops           [2][]int    // vendor rooms walked AT port i
	PortDeliveryBuckets [2][]string // buckets deliverable AT port i (the OTHER port's export buckets)
	LoadCap             int
	NewSlotMaxStock     int
}

func (c TradeCircuit) Validate() error {
	if c.RouteId == `` {
		return fmt.Errorf(`trade circuit missing RouteId`)
	}
	if c.FactorMobId <= 0 {
		return fmt.Errorf(`trade circuit %s missing FactorMobId`, c.RouteId)
	}
	if c.HomePortIdx < 0 || c.HomePortIdx > 1 {
		return fmt.Errorf(`trade circuit %s HomePortIdx must be 0 or 1`, c.RouteId)
	}
	for i := 0; i < 2; i++ {
		if len(c.PortExports[i]) == 0 {
			return fmt.Errorf(`trade circuit %s port %d has no exports`, c.RouteId, i)
		}
		if len(c.PortStops[i]) == 0 {
			return fmt.Errorf(`trade circuit %s port %d has no stops`, c.RouteId, i)
		}
	}
	if c.LoadCap <= 0 {
		return fmt.Errorf(`trade circuit %s needs positive LoadCap`, c.RouteId)
	}
	if c.NewSlotMaxStock <= 0 {
		return fmt.Errorf(`trade circuit %s needs positive NewSlotMaxStock`, c.RouteId)
	}
	return nil
}

// tradeCircuits is the registry, keyed by route id. Registered in Go (the
// importCircuits precedent in internal/caravan/import_circuits.go).
var tradeCircuits = map[string]TradeCircuit{
	"stillwater_np_packet": {
		RouteId:     "stillwater_np_packet",
		FactorMobId: 9577, // A Lakeway Factor
		HomePortIdx: 0,    // Stillwater pier 4118
		PortExports: [2][]int{
			{40057, 40058, 40059, 40056}, // Stillwater exports (stillwater bucket)
			{40018, 40019, 40006, 40021}, // NP exports (thornwall+base)
		},
		PortStops: [2][]int{
			{4105, 4106, 4125}, // Stillwater: Wulf, Brindle, Ilsa
			{5505, 5508},       // NP: Dunmar Wells, Chandler Voss
		},
		PortDeliveryBuckets: [2][]string{
			{"thornwall", "base"}, // NP goods arriving in Stillwater
			{"stillwater"},        // lake goods arriving in NP
		},
		LoadCap:         12,
		NewSlotMaxStock: 6,
	},
	"stillwater_confluence_barge": {
		RouteId:     "stillwater_confluence_barge",
		FactorMobId: 9578, // A Riverway Factor
		HomePortIdx: 0,    // Stillwater pier 4118
		PortExports: [2][]int{
			{40057, 40058, 40059, 40056}, // Stillwater exports
			{40123, 40124},               // Confluence exports (confluence bucket; extend per Task 1)
		},
		PortStops: [2][]int{
			{4105, 4106, 4125}, // Stillwater
			{6110, 6128},       // Confluence: Pella, Varro
		},
		PortDeliveryBuckets: [2][]string{
			{"confluence"},
			{"stillwater"},
		},
		LoadCap:         12,
		NewSlotMaxStock: 6,
	},
	"confluence_np_barge": {
		RouteId:     "confluence_np_barge",
		FactorMobId: 9579, // A Broadwater Factor
		HomePortIdx: 0,    // Confluence Barge Dock 6109
		PortExports: [2][]int{
			{40123, 40124},               // Confluence exports
			{40018, 40019, 40006, 40021}, // NP exports
		},
		PortStops: [2][]int{
			{6110, 6128}, // Confluence
			{5505, 5508}, // NP
		},
		PortDeliveryBuckets: [2][]string{
			{"thornwall", "base"},
			{"confluence"},
		},
		LoadCap:         12,
		NewSlotMaxStock: 6,
	},
}

// TradeCircuitFor returns the circuit for a route id, if registered.
func TradeCircuitFor(routeId string) (TradeCircuit, bool) {
	c, ok := tradeCircuits[routeId]
	return c, ok
}

// validateTradeCircuits runs boot-time integrity checks. Called from
// LoadDataFiles AFTER routes are loaded. Panics on failures (startup
// validator doctrine). Circuits referencing unloaded routes are an error;
// routes without circuits are fine (passenger-only lines).
func validateTradeCircuits() {
	for id, c := range tradeCircuits {
		if err := c.Validate(); err != nil {
			panic(fmt.Sprintf(`ferry trade circuit %s: %v`, id, err))
		}
		r, ok := RouteFor(c.RouteId)
		if !ok {
			// A circuit for a route with no YAML: tolerate ONLY if no routes
			// loaded at all (the data-optional skip path); otherwise panic.
			if len(routes) == 0 {
				continue
			}
			panic(fmt.Sprintf(`ferry trade circuit %s references unknown route`, id))
		}
		if mobs.GetMobSpec(mobs.MobId(c.FactorMobId)) == nil {
			panic(fmt.Sprintf(`ferry trade circuit %s: factor mob %d does not exist`, id, c.FactorMobId))
		}
		for i := 0; i < 2; i++ {
			for _, itemId := range c.PortExports[i] {
				if economy.BucketFor(itemId) == `` {
					panic(fmt.Sprintf(`ferry trade circuit %s: export item %d is unbucketed — delivery would silently skip it`, id, itemId))
				}
			}
			for _, stop := range c.PortStops[i] {
				if roomsLoad(stop) == nil {
					panic(fmt.Sprintf(`ferry trade circuit %s: stop room %d does not exist`, id, stop))
				}
			}
		}
		_ = r
	}
}
```

NOTE: `roomsLoad` = `rooms.LoadRoom` (write it directly; the placeholder name above only avoids an import line in this excerpt). Wire `validateTradeCircuits()` as the LAST line of the existing `LoadDataFiles()` in `route.go` (after `routes = loaded` and the log line; also call it on the missing-dir skip path ONLY if you keep the `len(routes)==0` tolerance — simplest: call it unconditionally at the end of both paths, the tolerance handles the empty case). Verify `mobs.GetMobSpec(mobs.MobId(N))` signature via codegraph before use (seen in `buildMercRows`, `internal/usercommands/list.go:236`).

- [ ] **Step 4: Run tests + build.**

Run: `go test ./internal/ferry/ -count=1 && go build ./...`
Expected: PASS + clean. (validateTradeCircuits isn't exercised by unit tests — boot-covered.)

- [ ] **Step 5: Commit.**

```bash
git add internal/ferry/tradecircuit.go internal/ferry/tradecircuit_test.go internal/ferry/route.go
git commit -m "feat(ferry): trade circuit registry + boot validation"
```

---

### Task 4: Factor state machine + controller integration

**Files:**
- Create: `internal/ferry/factor.go`
- Test: `internal/ferry/factor_test.go`
- Modify: `internal/ferry/controller.go` (drive factors from `Tick()`)

The factor lifecycle is a small explicit state machine driven once per round by the controller. States: `FactorWaiting` (at a dock, loaded, waiting for its vessel), `FactorAboard`, `FactorDelivering` (walking stops at a port), `FactorReturning` (walking back to the dock after the last stop). All transitions are decided by a **pure function** so the core is unit-testable; the impure shell does the room moves / commands.

- [ ] **Step 1: Write the failing tests for the pure decision core.**

```go
package ferry

import "testing"

// factorDecide is pure: given the circuit, the vessel state for its route,
// which port the factor is at (or -1 if elsewhere), whether it's on the
// deck, its current machine state, and its delivery progress, return the
// action to take this round.
func TestFactorDecide_BoardsWhenDockedAtItsPort(t *testing.T) {
	c := validCircuit()
	d := factorDecide(c, VesselState{Docked: true, PortIdx: 0}, factorPos{AtPortIdx: 0, OnDeck: false},
		factorState{Phase: FactorWaiting})
	if d.Kind != ActBoard {
		t.Fatalf("expected ActBoard, got %+v", d)
	}
}

func TestFactorDecide_DisembarksOnArrival(t *testing.T) {
	c := validCircuit()
	d := factorDecide(c, VesselState{Docked: true, PortIdx: 1}, factorPos{AtPortIdx: -1, OnDeck: true},
		factorState{Phase: FactorAboard})
	if d.Kind != ActDisembark || d.PortIdx != 1 {
		t.Fatalf("expected ActDisembark at port 1, got %+v", d)
	}
}

func TestFactorDecide_StaysAboardAtSea(t *testing.T) {
	c := validCircuit()
	d := factorDecide(c, VesselState{Docked: false, PortIdx: 1}, factorPos{AtPortIdx: -1, OnDeck: true},
		factorState{Phase: FactorAboard})
	if d.Kind != ActNone {
		t.Fatalf("expected ActNone at sea, got %+v", d)
	}
}

func TestFactorDecide_WalksStopsInOrderThenReturns(t *testing.T) {
	c := validCircuit()
	// Delivering at port 1, currently AT stop 0 (5505) → deliver + advance.
	d := factorDecide(c, VesselState{Docked: false, PortIdx: 0},
		factorPos{InRoom: 5505}, factorState{Phase: FactorDelivering, PortIdx: 1, StopIdx: 0})
	if d.Kind != ActDeliverHere || d.NextStop != 5508 {
		t.Fatalf("expected ActDeliverHere then next stop 5508, got %+v", d)
	}
	// Past the last stop → return to dock.
	d = factorDecide(c, VesselState{Docked: false, PortIdx: 0},
		factorPos{InRoom: 5508}, factorState{Phase: FactorDelivering, PortIdx: 1, StopIdx: 1})
	if d.Kind != ActDeliverHere || !d.LastStop {
		t.Fatalf("expected final ActDeliverHere with LastStop, got %+v", d)
	}
}

func TestFactorDecide_MissedBoatKeepsWaiting(t *testing.T) {
	c := validCircuit()
	// Vessel docked at the OTHER port → keep waiting, no action.
	d := factorDecide(c, VesselState{Docked: true, PortIdx: 1}, factorPos{AtPortIdx: 0},
		factorState{Phase: FactorWaiting})
	if d.Kind != ActNone {
		t.Fatalf("expected ActNone, got %+v", d)
	}
}
```

Define the exact types alongside (the test file pins the contract):

```go
type FactorPhase int

const (
	FactorWaiting FactorPhase = iota
	FactorAboard
	FactorDelivering
	FactorReturning
)

type factorState struct {
	Phase   FactorPhase
	PortIdx int // port being serviced (Delivering/Returning) or waited at (Waiting)
	StopIdx int // next/current index into PortStops[PortIdx]
}

type factorPos struct {
	AtPortIdx int // 0/1 when standing in that port's dock room, else -1
	OnDeck    bool
	InRoom    int // current room id (used during Delivering)
}

type FactorActionKind int

const (
	ActNone FactorActionKind = iota
	ActBoard
	ActDisembark
	ActWalkTo      // issue pathto d.NextStop
	ActDeliverHere // deliver at current room, then walk to d.NextStop (or dock if LastStop)
)

type factorAction struct {
	Kind     FactorActionKind
	PortIdx  int
	NextStop int
	LastStop bool
}
```

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/ferry/ -run TestFactorDecide -v`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Implement `factor.go`.** Two parts:

**(a) The pure core** — `factorDecide(c TradeCircuit, vs VesselState, pos factorPos, st factorState) factorAction`:

```go
func factorDecide(c TradeCircuit, vs VesselState, pos factorPos, st factorState) factorAction {
	switch st.Phase {
	case FactorWaiting:
		if vs.Docked && vs.PortIdx == st.PortIdx && pos.AtPortIdx == st.PortIdx {
			return factorAction{Kind: ActBoard, PortIdx: st.PortIdx}
		}
	case FactorAboard:
		if vs.Docked && pos.OnDeck {
			return factorAction{Kind: ActDisembark, PortIdx: vs.PortIdx}
		}
	case FactorDelivering:
		stops := c.PortStops[st.PortIdx]
		if st.StopIdx >= len(stops) {
			return factorAction{Kind: ActWalkTo, PortIdx: st.PortIdx, LastStop: true} // NextStop filled by caller with dock room
		}
		cur := stops[st.StopIdx]
		if pos.InRoom == cur {
			next := 0
			last := st.StopIdx == len(stops)-1
			if !last {
				next = stops[st.StopIdx+1]
			}
			return factorAction{Kind: ActDeliverHere, PortIdx: st.PortIdx, NextStop: next, LastStop: last}
		}
		return factorAction{Kind: ActWalkTo, PortIdx: st.PortIdx, NextStop: cur}
	case FactorReturning:
		if pos.AtPortIdx == st.PortIdx {
			return factorAction{Kind: ActNone} // shell transitions to Waiting + reload
		}
	}
	return factorAction{Kind: ActNone}
}
```

(Adjust exactly to make the Step 1 tests pass — the tests are the contract; e.g. the walk-vs-deliver distinction keyed on `pos.InRoom`.)

**(b) The impure shell** — `tickFactors()` called from `Tick()` after the vessel loop:

```go
// factorStates is in-memory only. On restart, factors are re-discovered in
// their spawn rooms (anchored spawninfo) and re-enter the loop as Waiting.
var factorStates = map[int]*factorState{} // mob instance id → state

func tickFactors(rpd int, now uint64) {
	for routeId, c := range tradeCircuits {
		r, ok := RouteFor(routeId)
		if !ok {
			continue
		}
		homeDock := r.Ports[c.HomePortIdx].DockRoom
		factor := mobs.FindLiveInstanceByHomeAndId(homeDock, c.FactorMobId)
		if factor == nil {
			continue // not spawned (yet); anchored spawninfo will bring it back
		}
		st, ok := factorStates[factor.InstanceId]
		if !ok {
			st = &factorState{Phase: FactorWaiting, PortIdx: portIdxOfRoom(r, factor.Character.RoomId)}
			if st.PortIdx < 0 { // mid-world after restart: walk home
				st.Phase = FactorReturning
				st.PortIdx = c.HomePortIdx
			}
			factorStates[factor.InstanceId] = st
			ensureFactorLoaded(factor, c, st.PortIdx)
		}
		vs := StateAt(r, now, rpd)
		pos := factorPos{
			AtPortIdx: portIdxOfRoom(r, factor.Character.RoomId),
			OnDeck:    factor.Character.RoomId == r.DeckRoom,
			InRoom:    factor.Character.RoomId,
		}
		act := factorDecide(c, vs, pos, *st)
		applyFactorAction(factor, r, c, st, act)
	}
}
```

`applyFactorAction` does the world mutations:
- **ActBoard**: transport factor dock→deck (RemoveMob/AddMob/RoomId; TransportCompanions pattern), emotes in both rooms (`"%s leads a laden handcart up the gangplank."` on the dock; a boarding line on deck), `st.Phase = FactorAboard`.
- **ActDisembark**: transport deck→dock of `act.PortIdx`, arrival emote, `st.Phase = FactorDelivering; st.PortIdx = act.PortIdx; st.StopIdx = 0`.
- **ActWalkTo**: if the factor is idle (not mid-path — track `lastWalkIssuedRound` on the state and reissue at most every N rounds, N=5), `factor.Command(fmt.Sprintf("pathto %d", act.NextStop))` — FIRST verify the pathto arg form in `internal/mobcommands/pathto.go`; if it doesn't take a room id, use whatever the schedule executor issues (read `applySchedulePlan` in `internal/hooks/NewRound_IdleMobs.go` for the exact command). When `act.LastStop` on the Delivering branch with exhausted stops, `NextStop` = the port's dock room and `st.Phase = FactorReturning`.
- **ActDeliverHere**: `delivered, _ := caravan.VisitVendorsInRoomOpts(pos.InRoom, factor, caravan.VisitOpts{DeliveryBuckets: c.PortDeliveryBuckets[st.PortIdx], CreateMissingSlots: true, NewSlotMaxStock: c.NewSlotMaxStock})`; emote via `caravan.FormatVisitMessage(...)` if non-empty; advance `st.StopIdx++`; if `act.LastStop` → `st.Phase = FactorReturning` and walk to dock.
- **FactorReturning + arrived at dock** (decide returns ActNone but shell checks pos): `st.Phase = FactorWaiting; st.PortIdx = current port; ensureFactorLoaded(...)` — reload THIS port's export manifest.

`ensureFactorLoaded(factor, c, portIdx)`: fill `factor.Character.Items` from `c.PortExports[portIdx]` round-robin via `items.New` + `StoreItem` up to `c.LoadCap` items total (the `LoadRunnerFromImport` pattern, `internal/caravan/import_load.go:12` — copy its maxTries bound).

Stuck safety: if Phase is Delivering/Returning and the factor's room hasn't changed for 60 rounds (track `lastRoom`, `lastRoomChangedRound` on factorState), reset: transport it directly to the current port's dock room with a quiet log (`mudlog.Warn("ferry.factor stuck-reset", ...)`) and set Waiting + reload. Blunt but self-healing, mirrors pathto-home recovery doctrine.

Add to `Tick()` in controller.go, after the route loop:

```go
	tickFactors(rpd, now)
```

- [ ] **Step 4: Run tests + build.**

Run: `go test ./internal/ferry/ -count=1 -v && go build ./...`
Expected: PASS (decide tests) + clean build.

- [ ] **Step 5: Commit.**

```bash
git add internal/ferry/factor.go internal/ferry/factor_test.go internal/ferry/controller.go
git commit -m "feat(ferry): trade factor state machine driven by the vessel controller"
```

---

### Task 5: Factors on the economy dashboard

**Files:**
- Modify: `internal/economy/health/capture.go`
- Modify: `internal/ferry` (small exported query)
- Test: `internal/economy/health/capture_ferry_test.go`

- [ ] **Step 1:** Export a snapshot query from ferry — add to `factor.go`:

```go
// FactorPhaseName returns a human state string for the dashboard, and ok
// false when the instance isn't a registered trade factor.
func FactorPhaseName(instanceId int) (string, bool) {
	st, ok := factorStates[instanceId]
	if !ok {
		return ``, false
	}
	switch st.Phase {
	case FactorWaiting:
		return `waiting_at_dock`, true
	case FactorAboard:
		return `aboard`, true
	case FactorDelivering:
		return `delivering`, true
	case FactorReturning:
		return `returning_to_dock`, true
	}
	return `unknown`, true
}
```

- [ ] **Step 2: Write the failing test.** In `capture_ferry_test.go`, seed a mob instance registered in `factorStates`-equivalent (call the ferry package's test hook — add `func SetFactorStateForTest(instanceId int, phase FactorPhase)` guarded to _test usage) and assert `captureFerryFactors()` emits a `CaravanSnapshot` with `Name`, `State`, `RoomId`, and cargo fields populated from the mob. Match the fixture style of existing capture tests (find them: `grep -l captureCaravans internal/economy/health/*_test.go`); if capture tests require heavy world scaffolding, write the test at the smallest seam that stays honest (e.g. assert on a helper `ferryFactorSnapshotFor(m *mobs.Mob) CaravanSnapshot` that the walker calls per factor).

- [ ] **Step 3: Implement.** In `capture.go`:

```go
// captureFerryFactors appends ferry trade factors as CaravanSnapshot rows.
// Factors are carrier mobs (cargo in their own inventory, no wagon) whose
// state comes from the ferry package's factor machine. Throughput accrues
// under (m.Zone, m.MobId) — the same keys VisitVendorsInRoomOpts writes.
func captureFerryFactors() []CaravanSnapshot {
	out := []CaravanSnapshot{}
	for _, instId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		stateName, ok := ferry.FactorPhaseName(instId)
		if !ok {
			continue
		}
		cs := CaravanSnapshot{
			InstId:        instId,
			Name:          m.Character.Name,
			State:         stateName,
			RoomId:        m.Character.RoomId,
			CargoWeight:   int(m.Character.GetCarriedWeight()),
			CargoCapacity: int(m.Character.CarryCapacity()),
			CargoByBucket: map[string]int{},
		}
		for _, it := range m.Character.Items {
			bucket := economy.BucketFor(it.ItemId)
			if bucket == `` {
				continue
			}
			if w := int(it.GetSpec().GetWeight()); w > 0 {
				cs.CargoByBucket[bucket] += w
			}
		}
		if tp := caravan.GetThroughput(m.Zone, int(m.MobId)); tp != nil {
			cs.LbsDelivered = tp.LbsDelivered
			if tp.DeliveriesByTier != nil {
				cs.DeliveriesByTier = map[int]int{}
				for tier, count := range tp.DeliveriesByTier {
					cs.DeliveriesByTier[tier] = count
				}
			}
		}
		out = append(out, cs)
	}
	return out
}
```

Append in `CaptureSnapshot()`: `snap.Caravans = append(snap.Caravans, captureFerryFactors()...)`. CHECK the import graph first: does `internal/economy/health` importing `internal/ferry` create a cycle? (`ferry` imports `economy` (buckets) but NOT `economy/health` — verify with `go list -deps` or codegraph; if a cycle exists, invert with a registered callback the way `rooms.SetBTreeStateEvictor` does.)

- [ ] **Step 4: Run tests + build.**

Run: `go test ./internal/economy/... ./internal/ferry/ -count=1 && go build ./...`
Expected: PASS + clean.

- [ ] **Step 5: Commit.**

```bash
git add internal/economy/health/ internal/ferry/factor.go
git commit -m "feat(ferry): trade factors surface as caravan rows in economy snapshots"
```

---

### Task 6: Content — factor mobs, spawninfo, anchor rooms

**Files:**
- Create: `_datafiles/world/dogmud/mobs/stillwater/9577-a_lakeway_factor.yaml`
- Create: `_datafiles/world/dogmud/mobs/stillwater/9578-a_riverway_factor.yaml`
- Create: `_datafiles/world/dogmud/mobs/the_confluence/9579-a_broadwater_factor.yaml`
- Modify: `_datafiles/world/dogmud/rooms/stillwater/4118.yaml` (append 2 spawninfo entries)
- Modify: `_datafiles/world/dogmud/rooms/the_confluence/6109.yaml` (append 1 spawninfo entry)
- Modify: `main.go:108` (`systemNPCAnchorRooms`)

- [ ] **Step 1: ID + name check.** `python tools/id_inventory.py --type mobs` (9577-9579 must still be free) and `grep -ri "lakeway\|riverway\|broadwater" _datafiles/world/dogmud/mobs/ --include=*.yaml` (no display-name collisions; also confirm no other mob in rooms 4118/6109 shares the token "factor").

- [ ] **Step 2: Author the three factor mobs.** Schema per the Town Crier / deckhand pattern (verified in Stage 1). Template for 9577 (write ORIGINAL prose per factor; 9578 = river-trade variant, 9579 = Confluence variant):

```yaml
mobid: 9577
zone: Stillwater
behavior_archetype: noncombat_passive
non_combatant: true
charm_immune: true
hostile: false
statpool: 30
maxwander: 0
groups:
  - humanoid
  - caravan   # IsEssential: rooms holding factors are never unloaded
idlecommands:
  - 'emote checks the lashings on a laden handcart and re-counts the
    crates against a tally-slip.'
  - ''
  - 'say Lake goods out, city goods back. The boat does the walking.'
  - ''
activitylevel: 10
character:
  name: A Lakeway Factor
  description: |
    A Lakeway Factor waits with a laden handcart, tally-slip tucked
    into a coat pocket gone shiny with wear. He rides the packet both
    ways every day it sails, buying where things are cheap and selling
    where they are wanted, and can tell you the price of lake mint in
    two cities without looking at the slip.
  speciesid: 1
  level: 1
  gold: 10
  stats:
    strength:
      training: 5
    charisma:
      training: 5
```

Filename check: `A Lakeway Factor` → `9577-a_lakeway_factor.yaml`; same rule for the others. NO semicolons in speech. Factor carry capacity note: LoadCap is 12 items of light goods — if the smoke test shows `StoreItem` failing on carry weight, bump `strength.training` rather than LoadCap.

- [ ] **Step 3: Spawninfo (surgical appends).** 4118 gains:

```yaml
  - mobid: 9577
    respawnrate: "15 real minutes"
  - mobid: 9578
    respawnrate: "15 real minutes"
```

6109 gains the 9579 entry. Preserve every existing line.

- [ ] **Step 4: Anchor the factor home docks.** In `main.go`, extend `systemNPCAnchorRooms`:

```go
	4118, // Stillwater Boat Rental Pier — ferry factors 9577 (packet) + 9578 (heron)
	6109, // Confluence Barge Dock — ferry factor 9579 (broadbeam)
```

- [ ] **Step 5: Verify.** `go build ./...`; python yaml parse over the 5 touched YAML files; 80-col check on added lines.

- [ ] **Step 6: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/ _datafiles/world/dogmud/rooms/stillwater/4118.yaml _datafiles/world/dogmud/rooms/the_confluence/6109.yaml main.go
git commit -m "content(ferry): three trade factors, dock spawninfo, anchored home docks"
```

---

### Task 7: Boot test

- [ ] **Step 1:** `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*` (4118/6109 room templates changed — mandatory).
- [ ] **Step 2:** `go run .` (background, from repo root). Expect: `mobs.LoadDataFiles() loadedCount=606`, `ferry.LoadDataFiles() loadedCount=3` with NO trade-circuit panic (validates circuits: routes, factor specs, bucketed exports, stop rooms), `ValidateZoneConsistency errors=0`, factors spawned at boot in 4118/6109 (anchored). Kill all GoMud/go processes after.
- [ ] **Step 3:** Commit any fixes.

---

### Task 8: Harness verification — a full factor circuit

**Files:**
- Create: `tools/playtest/goals/ferry-stage2.yaml`

- [ ] **Step 1: Goals file** (feature-tester; smoketester admin char; format per `tools/playtest/goals/ferry-stage1.yaml`):

1. `goto 4118`; observe both factors present with cargo flavor; confirm they do NOT respond to `ask factor passage` as if they were agents (no boarding).
2. Wait for the packet to dock; watch the factor board (gangplank emote) — the player boards too (pay the agent) and rides WITH the factor.
3. On arrival at NP: factor disembarks, walks off toward Warehouse Row; follow it; watch a delivery emote at 5505 and/or 5508 (`FormatVisitMessage` output).
4. Verify stock: `list` at Dunmar Wells (5505) — stillwater goods (lake mint / freshwater clam / lake-iron nodule / marsh willow bark) now appear in his listing with small quantities. Record before/after if possible.
5. Watch the factor return to the North Quay and wait; confirm it boards the next departure home.
6. Economy dashboard row: hit the admin economy endpoint (or run a manual snapshot capture) and confirm a caravan row named "A Lakeway Factor" with state + cargo fields.
7. Stuck-safety spot check: none expected; note any factor standing mid-street > 3 minutes as a finding.
8. Feel: does the factor read as alive (boarding/delivery emotes), do the cross-region goods in shops feel natural, any message weirdness.

- [ ] **Step 2:** Boot fresh (instance wipe), run `/playtest local feature-tester ferry-stage2.yaml`. Fix mechanical findings; log feel notes.

- [ ] **Step 3:** Commit goals + fixes.

```bash
git add tools/playtest/goals/ferry-stage2.yaml
git commit -m "test(ferry): stage-2 factor circuit harness goals + findings fixes"
```

---

## Post-plan notes

- **Fallback (pre-approved in spec):** if the riding-factor seam proves flaky in practice, replace ActBoard/ActDisembark with direct port-call delivery (controller calls `VisitVendorsInRoomOpts` for each stop room on vessel arrival, no factor movement) — the circuits/manifests/slots machinery is identical, only the visible mob ride is lost.
- Stage 3 integration seams left ready: `VisitOpts.PickupBuckets` (overflow capture inverse), the factor loading site (`ensureFactorLoaded`) is where warehouse drawdown (Stage 4) plugs in.
- Pre-push SOP applies before any prod push — not part of this plan.
