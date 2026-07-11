# Ferry Stage 4 — Warehouse Drawdown — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Warehouse reserves flow back out during shortages: carriers loading in a warehouse city pull manifest cargo from the warehouse first (shortage-prioritized), and vendor stops in warehouse cities get their gaps topped up from the local warehouse — completing the "reserves backfill a big run on materials" loop.

**Architecture:** Two release mechanisms behind one kill switch (`GamePlay.WarehouseDrawdownEnabled`): (1) **load-time drawdown** — `ensureFactorLoaded` (ferry) and `LoadRunnerFromImport` (Dobb) withdraw manifest items from the loading city's warehouse before minting from the infinite trickle, with ferry loads prioritized by downstream vendor demand; (2) **delivery-time local release** — after a carrier's delivery pass at a vendor stop in a warehouse city, remaining vendor stock gaps are topped up from the local warehouse, bounded per visit (slow release by design). Empty warehouse → behavior byte-identical to Stage 3.

**Tech Stack:** Go; builds solely on Stage 2/3 seams (`warehouse.Deposit` gains a `Withdraw` sibling; the two load sites and the two deliver sites gain calls).

**Spec:** Stage 4 of `docs/superpowers/specs/2026-07-03-ferry-system-design.md`, with one amendment locked below.

---

## Verified engine facts + spec amendment

- **Spec amendment (world-graph reality):** the spec's "legacy Thornwall-bound caravan loading at the NP end" hook does not exist — Lars's runner circuits (`thornwall_runner_circuit`, `stillwater_runner_circuit`, see `internal/caravan/arrival_listener.go`) never enter NP; Dobb's import circuit owns the NP depot. Thornwall backfill already flows via Lars's pickup passes at Stillwater vendors (`pickupQualifies` accepts any `IsComponent` raw material), which ferry deliveries keep stocked. Stage 4 therefore hooks **Dobb's depot load + vendor deliveries** for NP and drops the phantom legacy-caravan step. Amend the spec's Stage 4 section with one parenthetical when committing Task 1.
- Load sites: `internal/ferry/factor.go` `ensureFactorLoaded(factor, c, portIdx)` (manifest round-robin via `items.New`+`StoreItem` up to `c.LoadCap`, counting existing items); `internal/caravan/import_load.go` `LoadRunnerFromImport(runner, c ImportCircuit)` (same pattern, `c.ImportItems`, `c.LoadCap`).
- Deliver sites: ferry `applyFactorAction` `ActDeliverHere` (calls `caravan.VisitVendorsInRoomOpts`); Dobb `handleImportArrival` `importDeliver` case (calls `VisitVendorsInRoom`).
- Warehouse package (Stage 3): `Deposit/accrue → deposit(zone,item,qty,isAccrual)`, `dirty` map + `markDirty`, `itemCapFn`, `ResetForTest`, `CityFor`, `WarehouseFor`, `StockOf`, counters incl. the so-far-unwritten `DrawnCount`. Snapshot already carries `drawn_count` (nothing to add there).
- Import direction rules: `warehouse` may ADD an import on `shops` (nothing in `shops` imports `warehouse` — verify with `go list -deps ./internal/shops` before relying on it). `ferry`/`caravan` already import `warehouse`.
- Vendor enumeration in a room: `room.GetMobs(rooms.FindMerchant)` → `mobs.GetInstance` → `shops.GetShopInventory(vendor.Zone, int(vendor.MobId), vendor.HomeRoomId)` (the `VisitVendorsInRoomOpts` pattern in `internal/caravan/visit.go` — mirror its nil-guards). Persist with `shops.SaveShop(zone, mobId, roomId)` only when mutated.
- Rooms hosting shopkeepers are essential-pinned once spawned; at boot before first spawn a cross-city demand probe may see no vendors → demand 0 → clean fallback to manifest order.
- Ferry circuits: `TradeCircuit{PortExports, PortStops, PortDeliveryBuckets, LoadCap, NewSlotMaxStock}`; loading at port i ships `PortExports[i]` to `PortStops[1-i]`.
- Stage 3 live state currently on disk: Confluence warehouse holds 12 captured stillwater items + accrued river goods — ready-made fixture for live verification.

## Design decisions locked here

- **Load-time drawdown draws MANIFEST items only** (`PortExports[i]` / `ImportItems`). Non-manifest captured stock (e.g. stillwater goods banked at the Confluence) is released by mechanism (2) locally, never re-shipped — delivery bucket gates would strand it anyway. This also means load-time drawdown can never change WHAT a carrier ships, only WHERE it came from.
- **Local release tops up EXISTING vendor entries only** (native or ferry-created slots), never creates slots, bounded `warehouseReleaseMaxPerItem = 2` per item per visit (spec: "deliberately slow"). Release happens after the carrier's own delivery pass so cargo lands first.
- **Ferry demand prioritization**: demand per export item = Σ over the other port's stops' vendors of `max(0, MaxStock-Current)` for existing entries, + `NewSlotMaxStock` per vendor lacking the entry. Pure sort helper unit-tested; the gathering walk is thin and live-covered. Dobb skips prioritization (LoadCap 24 ≥ his whole manifest — ordering is irrelevant).
- Kill switch `GamePlay.WarehouseDrawdownEnabled` gates BOTH mechanisms; `WarehousesEnabled` false implies no drawdown too (warehouses empty/frozen).

## File structure

```
internal/warehouse/withdraw.go          Withdraw + ReleaseToVendorsInRoom
internal/warehouse/withdraw_test.go
internal/ferry/factor.go                (modify: demand-prioritized warehouse-first load)
internal/ferry/factor_test.go           (modify: prioritizeByDemand tests)
internal/caravan/import_load.go         (modify: warehouse-first draw)
internal/caravan/import_load_test.go    (modify: drawdown test)
internal/caravan/import_arrival.go      (modify: local release at importDeliver)
internal/configs/config.gameplay.go     (modify: WarehouseDrawdownEnabled)
_datafiles/config.yaml                  (modify: knob)
docs/superpowers/specs/2026-07-03-ferry-system-design.md  (modify: 1-line amendment)
```

---

### Task 1: `Withdraw` + kill switch + spec amendment

**Files:**
- Create: `internal/warehouse/withdraw.go`, `internal/warehouse/withdraw_test.go`
- Modify: `internal/configs/config.gameplay.go`, `_datafiles/config.yaml`, the spec (one parenthetical)

- [ ] **Step 1: Failing tests.**

```go
package warehouse

import "testing"

func TestWithdraw_DecrementsAndCounts(t *testing.T) {
	ResetForTest()
	Deposit("The Confluence", 40123, 5)
	got := Withdraw("The Confluence", 40123, 3)
	if got != 3 {
		t.Fatalf("Withdraw = %d, want 3", got)
	}
	w := WarehouseFor("The Confluence")
	if w.StockOf(40123) != 2 || w.DrawnCount != 3 {
		t.Fatalf("stock=%d drawn=%d, want 2/3", w.StockOf(40123), w.DrawnCount)
	}
}

func TestWithdraw_FloorsAtZeroStock(t *testing.T) {
	ResetForTest()
	Deposit("The Confluence", 40123, 2)
	if got := Withdraw("The Confluence", 40123, 10); got != 2 {
		t.Fatalf("partial withdraw = %d, want 2", got)
	}
	if got := Withdraw("The Confluence", 40123, 1); got != 0 {
		t.Fatalf("empty withdraw = %d, want 0", got)
	}
	if got := Withdraw("Stillwater", 40123, 1); got != 0 {
		t.Fatalf("unknown-zone withdraw = %d, want 0", got)
	}
}
```

- [ ] **Step 2:** FAIL run. **Step 3: Implement** in `withdraw.go`:

```go
package warehouse

// Withdraw removes up to qty of an item from a city's pool, returning the
// amount actually withdrawn (0 for unknown zones / empty stock). Increments
// DrawnCount and marks the zone dirty. Stage 4 drawdown's only exit path —
// callers mint the physical items (items.New) for what they receive.
func Withdraw(zone string, itemId int, qty int) int {
	if qty <= 0 {
		return 0
	}
	if _, ok := cities[zone]; !ok {
		return 0
	}
	mu.Lock()
	defer mu.Unlock()
	w := getOrCreateLocked(zone)
	for i := range w.Stock {
		if w.Stock[i].ItemId != itemId {
			continue
		}
		take := qty
		if take > w.Stock[i].Current {
			take = w.Stock[i].Current
		}
		if take <= 0 {
			return 0
		}
		w.Stock[i].Current -= take
		w.DrawnCount += take
		dirty[zone] = true
		return take
	}
	return 0
}
```

- [ ] **Step 4:** PASS. **Step 5:** Knob: `config.gameplay.go` under `WarehousesEnabled`:

```go
	WarehouseDrawdownEnabled ConfigBool `yaml:"WarehouseDrawdownEnabled"` // Stage 4 kill switch: warehouse-first loading + local vendor release
```

`config.yaml` sibling entry `WarehouseDrawdownEnabled: true` with comment. Spec amendment — in the Stage 4 section, after the load-event list, append: `(2026-07-03 build note: the "legacy Thornwall-bound caravan at the NP end" hook does not exist in the world graph — Lars's circuits never enter NP; Dobb's depot load carries the NP role, and Thornwall backfill flows via Lars's Stillwater pickups.)`

- [ ] **Step 6:** Commit `feat(warehouse): Withdraw + drawdown kill switch`.

---

### Task 2: Load-time drawdown — ferry + Dobb

**Files:**
- Modify: `internal/ferry/factor.go`, `internal/ferry/factor_test.go`, `internal/caravan/import_load.go`, `internal/caravan/import_load_test.go`

- [ ] **Step 1: Failing test — the pure prioritizer** (ferry package):

```go
func TestPrioritizeByDemand_SortsDescStable(t *testing.T) {
	items := []int{40056, 40057, 40058, 40059}
	demand := map[int]int{40056: 2, 40057: 9, 40058: 0, 40059: 9}
	got := prioritizeByDemand(items, demand)
	// Desc by demand; ties keep manifest order (40057 before 40059).
	want := []int{40057, 40059, 40056, 40058}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	// Input slice must not be mutated.
	if items[0] != 40056 {
		t.Fatal("input mutated")
	}
}
```

- [ ] **Step 2:** FAIL. **Step 3: Implement in `factor.go`:**

```go
// prioritizeByDemand returns manifest items sorted by downstream demand
// (desc), ties keeping manifest order. Pure.
func prioritizeByDemand(manifest []int, demand map[int]int) []int {
	out := append([]int(nil), manifest...)
	sort.SliceStable(out, func(a, b int) bool {
		return demand[out[a]] > demand[out[b]]
	})
	return out
}
```

CAREFUL: `sort.SliceStable`'s less receives INDICES — the comparator above must read `demand[out[a]] > demand[out[b]]` where a/b index `out`. Write it correctly (the test pins it).

```go
// exportDemand sums downstream vendor gaps for each export item of the
// given port: existing entries contribute MaxStock-Current; a vendor
// missing the entry contributes NewSlotMaxStock (CreateMissingSlots will
// open one on delivery). Unspawned vendors / unloaded rooms contribute 0
// — clean fallback to manifest order at cold boot.
func exportDemand(c TradeCircuit, portIdx int) map[int]int {
	demand := map[int]int{}
	for _, stop := range c.PortStops[1-portIdx] {
		room := rooms.LoadRoom(stop)
		if room == nil {
			continue
		}
		for _, instId := range room.GetMobs(rooms.FindMerchant) {
			vendor := mobs.GetInstance(instId)
			if vendor == nil || !vendor.HasShop() {
				continue
			}
			shop := shops.GetShopInventory(vendor.Zone, int(vendor.MobId), vendor.HomeRoomId)
			if shop == nil {
				continue
			}
			for _, itemId := range c.PortExports[portIdx] {
				if e := shop.GetStock(itemId); e != nil {
					if gap := e.MaxStock - e.Current; gap > 0 {
						demand[itemId] += gap
					}
				} else {
					demand[itemId] += c.NewSlotMaxStock
				}
			}
		}
	}
	return demand
}
```

(`ferry` importing `shops`: verify no cycle — shops must not import ferry; check.)

Rework `ensureFactorLoaded`: when `WarehousesEnabled && WarehouseDrawdownEnabled` and the loading port's dock zone has a warehouse, first pass over `prioritizeByDemand(c.PortExports[portIdx], exportDemand(c, portIdx))`: for each item while `len(Items) < LoadCap`, `n := warehouse.Withdraw(zone, itemId, 1)`; if n==1 mint `items.New(itemId)` + `StoreItem` (on StoreItem failure, RE-DEPOSIT the unit — `warehouse.Deposit(zone, itemId, 1)` — never destroy stock; note Deposit counts it as captured, acceptable). Loop items round-robin (keep drawing 1 each in priority order, cycling while any warehouse stock + capacity remain, bounded by `LoadCap*2+len(manifest)` tries like the sibling loader). THEN the existing manifest round-robin tops up remaining capacity unchanged. Keep the whole drawdown block in a small helper `loadFromWarehouse(factor, c, portIdx, dockZone) int` so `ensureFactorLoaded` stays readable.

- [ ] **Step 4: Dobb.** In `internal/caravan/import_load.go`, extend `LoadRunnerFromImport`: before the existing mint loop, when the toggles are on AND the runner's current room zone has a warehouse (`warehouse.CityFor`), loop `c.ImportItems` round-robin drawing `warehouse.Withdraw(zone, itemId, 1)` + `items.New`+`StoreItem` (same re-deposit-on-store-failure rule) until capacity or warehouse dry. The signature gains the zone: change to `LoadRunnerFromImport(runner *mobs.Mob, c ImportCircuit, roomZone string)` OR resolve the zone inside from `runner.Character.RoomId` — pick resolving inside (no caller churn; `rooms.LoadRoom` nil-safe). Extend `import_load_test.go`: existing tests must pass unchanged (toggle off in test env — ConfigBool zero=false, document that); add `TestLoadRunnerFromImport_DrawsFromWarehouseFirst` — seed `warehouse.ResetForTest()` + `Deposit`, force the toggle via a test hook... ConfigBool zero-value is FALSE in unit env, so to test the drawdown path add an unexported `drawdownEnabledFn func() bool` var in caravan (defaulting to the config read) swapped in the test — mirror the warehouse `itemCapFn` pattern. Same pattern in ferry if needed for its tests (the pure prioritizer + Withdraw tests may be sufficient there — implementer's call, note it).

- [ ] **Step 5:** `go build ./... && go test ./internal/ferry/ ./internal/caravan/ ./internal/warehouse/ -count=1` green. **Step 6:** Commit `feat(warehouse): load-time drawdown — demand-prioritized ferry loads + Dobb depot draws`.

---

### Task 3: Delivery-time local release

**Files:**
- Modify: `internal/warehouse/withdraw.go` (+`withdraw_test.go`), `internal/ferry/factor.go`, `internal/caravan/import_arrival.go`

- [ ] **Step 1: Failing test** (warehouse package; shop fixtures in the `internal/caravan/visit_test.go` style — items.SeedItemsForTest + shops.ClearCache + RegisterShop; check what shops-test helpers are exported and reuse):

```go
func TestReleaseToVendorsInRoom_TopsGapsBounded(t *testing.T) {
	// vendor in room R with entry {ItemId: 40123, Current: 1, MaxStock: 6};
	// warehouse holds 10 × 40123.
	// After ReleaseToVendorsInRoom("The Confluence", R, 2):
	//  - vendor entry Current == 3 (bounded by maxPerItem 2, not the gap 5)
	//  - warehouse stock == 8, DrawnCount == 2
	// Second call: Current == 5, stock == 6.
	// Entry at MaxStock: no change, no withdraw.
	// Item NOT stocked by vendor: nothing created (release never creates slots).
}
```

Write it concretely against the fixtures (the comment states the contract; arrange section must be real code — room seeding via `rooms.SeedRoomsForTest`, vendor mob via `mobs.SetInstanceForTest`, patterns all present in `capture_ferry_test.go`/`visit_test.go`).

- [ ] **Step 2:** FAIL. **Step 3: Implement** in `withdraw.go`:

```go
// ReleaseToVendorsInRoom tops up existing vendor stock entries in one room
// from the local warehouse, at most maxPerItem per entry per call (slow
// release by design). Never creates slots; never touches non-warehouse
// zones. Returns units released. Callers gate on the drawdown toggle.
func ReleaseToVendorsInRoom(zone string, roomId int, maxPerItem int) int {
	if _, ok := cities[zone]; !ok || maxPerItem <= 0 {
		return 0
	}
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return 0
	}
	released := 0
	for _, instId := range room.GetMobs(rooms.FindMerchant) {
		vendor := mobs.GetInstance(instId)
		if vendor == nil || !vendor.HasShop() {
			continue
		}
		shop := shops.GetShopInventory(vendor.Zone, int(vendor.MobId), vendor.HomeRoomId)
		if shop == nil {
			continue
		}
		mutated := false
		for i := range shop.Stock {
			e := &shop.Stock[i]
			gap := e.MaxStock - e.Current
			if gap <= 0 {
				continue
			}
			want := gap
			if want > maxPerItem {
				want = maxPerItem
			}
			got := Withdraw(zone, e.ItemId, want)
			if got > 0 {
				e.Current += got
				released += got
				mutated = true
			}
		}
		if mutated {
			if err := shops.SaveShop(vendor.Zone, int(vendor.MobId), vendor.HomeRoomId); err != nil {
				mudlog.Error("warehouse.ReleaseToVendorsInRoom", "error", err)
			}
		}
	}
	return released
}
```

NOTE: this adds `warehouse → shops, rooms, mobs` imports. VERIFY none of those import `warehouse` (they don't today — `go list -deps` check; if one does, move this function into `internal/caravan` instead and say so).

- [ ] **Step 4: Call sites.** Ferry `applyFactorAction` `ActDeliverHere` branch, after the `VisitVendorsInRoomOpts` call + emote: 

```go
	if bool(configs.GetGamePlayConfig().WarehousesEnabled) && bool(configs.GetGamePlayConfig().WarehouseDrawdownEnabled) {
		if room := rooms.LoadRoom(pos.InRoom); room != nil {
			warehouse.ReleaseToVendorsInRoom(room.Zone, pos.InRoom, warehouseReleaseMaxPerItem)
		}
	}
```

with `const warehouseReleaseMaxPerItem = 2` near the ferry constants (adapt variable names to the actual code — read the branch first). Same gated call in Dobb's `importDeliver` case after his `VisitVendorsInRoom` (his vendor rooms are NP = warehouse city). No emote for release v1 (the vendor restock messaging already covers flavor; note as future polish).

- [ ] **Step 5:** Full test run green. **Step 6:** Commit `feat(warehouse): delivery-time local release of reserves to vendor gaps`.

---

### Task 4: Boot + live verification + wrap

- [ ] **Step 1:** Instance wipe, boot. Expect all Stage 1-3 loader lines + no panics; `warehouse.LoadAll() loadedCount=2` (files persist from Stage 3 verification).
- [ ] **Step 2: The headline scenario, live.** Harness (smoketester): `teleport 6110`; `list` at Pella — record stillwater goods; BUY OUT one item entirely (e.g. all lake mint — smoketester has gold; this is the "big run"). Wait for the next heron factor visit at Pella (≤1 cycle). Verify: (a) the factor's delivery + local release refill the bought-out slot ABOVE what cargo alone could (cargo lands ≤ manifest share; release adds up to 2 more from warehouse); (b) `the_confluence.yaml` `drawn_count` rose; (c) stock levels in the shop file. Also verify load-time draw: after a factor loads at the Confluence, warehouse river-goods stock dips (drawn for manifest) in the file.
- [ ] **Step 3:** Toggle check: set `WarehouseDrawdownEnabled: false` in config.yaml, reboot, one circuit — `drawn_count` unchanged; revert to true. (Fast: ~5 min.)
- [ ] **Step 4:** Kill processes. Commit fixes if any. Final whole-branch review (integration: toggle-off byte-identical to Stage 3; no Withdraw path can go negative; release respects MaxStock; demand walk nil-safe) → merge `--no-ff` → delete branch → memory update.
