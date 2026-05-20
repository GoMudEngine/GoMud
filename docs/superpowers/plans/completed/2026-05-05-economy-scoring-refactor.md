# Economy Health Scoring Refactor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the single per-shop fill-percentage score with five
independent axes (stock, throughput, input rate, logistics health,
shop gold), per-rarity-tier restock cadence, and tighter caravan/
forager failure-mode signals.

**Architecture:** Phase 1 changes engine restock cadence (per-tier
clocks). Phase 2 adds persistent counters on `ShopInventory` /
`Caravan` / `Forager` that record sales, buys, restocks, depletion
events, and pounds delivered — counters accumulate without consumers.
Phase 3 implements the five score functions and the new overall
blend. Phase 4 rewires the dashboard. Phase 5 sanity-checks against
known-broken entities (Halix despawn, Kessa stuck).

**Tech Stack:** Go (engine + scoring), YAML (config + persistence),
HTML/JS (dashboard).

**Spec:** `docs/superpowers/specs/2026-05-05-economy-scoring-refactor-design.md`

---

## Phase 1 — Per-rarity-tier restock cadence

Replaces the single global `CrafterMaterialRestockRate` knob with
five per-tier cadences. Tier 50 commons refill hourly; tier 10 rares
refill every 5 game-days as a slow backstop.

### Task 1: Add per-tier cadence config knobs

**Files:**
- Modify: `internal/configs/balance.go`

- [ ] **Step 1: Add the new fields to the `Balance` struct**

Find the existing `CrafterMaterialRestockRate` field and add directly
below it (preserve the comment style of nearby fields):

```go
// Per-rarity-tier restock cadences (game-time hours). Replaces the
// single CrafterMaterialRestockRate. Higher rarity tiers (= more
// common) fire faster; lower tiers (= rarer) fire slowly as a
// backstop on top of forager / player-sale input.
RestockCadenceTier50Hours ConfigInt    `yaml:"RestockCadenceTier50Hours"`
RestockCadenceTier40Hours ConfigInt    `yaml:"RestockCadenceTier40Hours"`
RestockCadenceTier30Hours ConfigInt    `yaml:"RestockCadenceTier30Hours"`
RestockCadenceTier20Hours ConfigInt    `yaml:"RestockCadenceTier20Hours"`
RestockCadenceTier10Days  ConfigInt    `yaml:"RestockCadenceTier10Days"`
```

If the codebase uses a different config-int wrapper (`ConfigUint`,
`ConfigSecondsString`, etc.), match the pattern of nearby fields.

- [ ] **Step 2: Add a defaults-setter for these fields**

Find `func (c *Balance) SetDefaults()` (or the equivalent) and add:

```go
if c.RestockCadenceTier50Hours == 0 {
    c.RestockCadenceTier50Hours = 1
}
if c.RestockCadenceTier40Hours == 0 {
    c.RestockCadenceTier40Hours = 2
}
if c.RestockCadenceTier30Hours == 0 {
    c.RestockCadenceTier30Hours = 6
}
if c.RestockCadenceTier20Hours == 0 {
    c.RestockCadenceTier20Hours = 24
}
if c.RestockCadenceTier10Days == 0 {
    c.RestockCadenceTier10Days = 5
}
```

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add internal/configs/balance.go
git commit -m "feat(config): per-rarity-tier restock cadence knobs

Adds RestockCadenceTier50Hours/Tier40Hours/Tier30Hours/Tier20Hours/
Tier10Days. Defaults: 1h / 2h / 6h / 24h / 5d. Replaces the single
CrafterMaterialRestockRate knob (kept for back-compat in this
commit; removed in a later phase)."
```

### Task 2: Set defaults in config.yaml

**Files:**
- Modify: `_datafiles/config.yaml`

- [ ] **Step 1: Add the new YAML keys**

Find the line `CrafterMaterialRestockRate: 75` (around line 674) and
add directly above:

```yaml
  # Per-rarity-tier restock cadences (game-time). Replaces the single
  # CrafterMaterialRestockRate. Common items refill quickly so
  # crafting grinders have stock; rare items refill slowly as a
  # backstop above forager/player-sale input.
  RestockCadenceTier50Hours: 1   # game-hours
  RestockCadenceTier40Hours: 2
  RestockCadenceTier30Hours: 6
  RestockCadenceTier20Hours: 24
  RestockCadenceTier10Days:  5   # game-days
```

- [ ] **Step 2: Commit**

```bash
git add _datafiles/config.yaml
git commit -m "feat(config): default per-tier restock cadences (1h/2h/6h/24h/5d)"
```

### Task 3: Convert per-tier hours to round counts

**Files:**
- Create: `internal/shops/restock_cadence.go`
- Create: `internal/shops/restock_cadence_test.go`

- [ ] **Step 1: Write the failing test**

```go
package shops

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestRestockCadenceRounds_PerTier(t *testing.T) {
	// configs.GetServerConfig().Timing.RoundsPerSecond exposes the
	// rounds-per-real-second; game-time uses the configured GameDay
	// multiplier. The test isolates the math by stubbing those
	// values and asserting the conversion.
	b := &configs.Balance{
		RestockCadenceTier50Hours: 1,
		RestockCadenceTier40Hours: 2,
		RestockCadenceTier30Hours: 6,
		RestockCadenceTier20Hours: 24,
		RestockCadenceTier10Days:  5,
	}
	cases := []struct {
		tier int
		hrs  int
	}{
		{50, 1},
		{40, 2},
		{30, 6},
		{20, 24},
		{10, 5 * 24}, // tier-10 expressed in days
	}
	for _, c := range cases {
		got := RestockCadenceHours(b, c.tier)
		if got != c.hrs {
			t.Errorf("tier %d: got %d hours, want %d", c.tier, got, c.hrs)
		}
	}
}

func TestRestockCadenceHours_UnknownTier(t *testing.T) {
	b := &configs.Balance{RestockCadenceTier50Hours: 1}
	got := RestockCadenceHours(b, 999)
	if got != 0 {
		t.Errorf("unknown tier: got %d, want 0 (sentinel for skip)", got)
	}
}
```

- [ ] **Step 2: Run test — expected to fail**

```
go test ./internal/shops/... -run TestRestockCadence
```
Expected: FAIL with "RestockCadenceHours not defined".

- [ ] **Step 3: Implement**

```go
// Package shops — restock_cadence.go
package shops

import "github.com/GoMudEngine/GoMud/internal/configs"

// RestockCadenceHours returns the configured restock period for the
// given rarity tier, in game-time hours. Returns 0 for unrecognized
// tiers — callers treat 0 as "no scheduled restock".
func RestockCadenceHours(b *configs.Balance, rarityTier int) int {
	switch rarityTier {
	case 50:
		return int(b.RestockCadenceTier50Hours)
	case 40:
		return int(b.RestockCadenceTier40Hours)
	case 30:
		return int(b.RestockCadenceTier30Hours)
	case 20:
		return int(b.RestockCadenceTier20Hours)
	case 10:
		return int(b.RestockCadenceTier10Days) * 24
	default:
		return 0
	}
}
```

- [ ] **Step 4: Run test — should pass**

`go test ./internal/shops/... -run TestRestockCadence`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shops/restock_cadence.go internal/shops/restock_cadence_test.go
git commit -m "feat(shops): RestockCadenceHours lookup by rarity tier"
```

### Task 4: RestockTier method on ShopInventory

**Files:**
- Modify: `internal/shops/shopinventory.go`
- Modify: `internal/shops/shopinventory_test.go`

- [ ] **Step 1: Write the failing test**

In `shopinventory_test.go`:

```go
func TestRestockTier_OnlyTopsUpMatchingTier(t *testing.T) {
	// Arrange: shop has one tier-50 item half-empty, one tier-30
	// item half-empty, one tier-50 item already full.
	// Both tier-50s have RestockQty=5, MaxStock=10.
	// Tier-30 has RestockQty=5, MaxStock=10.
	si := &ShopInventory{
		Stock: []StockEntry{
			{ItemId: testTier50ItemA, RestockQty: 5, MaxStock: 10, Current: 5},
			{ItemId: testTier30Item,  RestockQty: 5, MaxStock: 10, Current: 5},
			{ItemId: testTier50ItemB, RestockQty: 5, MaxStock: 10, Current: 10},
		},
	}
	// Act: restock tier 50 only.
	added := si.RestockTier(50)
	// Assert: tier-50 half-empty filled, tier-30 untouched, tier-50
	// already-full untouched. Returns true (something added).
	if !added {
		t.Errorf("expected RestockTier to return true")
	}
	if si.Stock[0].Current != 10 {
		t.Errorf("tier-50 half-empty: got %d, want 10", si.Stock[0].Current)
	}
	if si.Stock[1].Current != 5 {
		t.Errorf("tier-30 should be untouched: got %d", si.Stock[1].Current)
	}
	if si.Stock[2].Current != 10 {
		t.Errorf("tier-50 full should be untouched: got %d", si.Stock[2].Current)
	}
}
```

(Establish `testTier50ItemA`, `testTier50ItemB`, `testTier30Item` as
test fixtures in `test_main_test.go` if not already present — register
test ItemSpecs with `RarityTier: 50` and `RarityTier: 30`. Use the
existing fixture pattern in the `_test.go` files.)

- [ ] **Step 2: Run — expected fail (method doesn't exist)**

`go test ./internal/shops/... -run TestRestockTier`
Expected: FAIL.

- [ ] **Step 3: Implement on `ShopInventory`**

In `shopinventory.go`, add below `RestockBaselineTiers`:

```go
// RestockTier tops up StockEntries whose item carries the given
// rarity_tier, by RestockQty per call (capped at MaxStock). Skips
// entries with RestockQty <= 0 (NPC-crafted items don't restock).
// Returns true if any stock was added.
//
// Replaces the all-or-nothing Restock() / RestockBaselineTiers()
// approach with per-tier granularity, enabling per-rarity-tier
// cadences (commons restock often, rares rarely).
func (si *ShopInventory) RestockTier(rarityTier int) bool {
	restocked := false
	for i := range si.Stock {
		e := &si.Stock[i]
		if e.RestockQty <= 0 {
			continue
		}
		spec := items.GetItemSpec(e.ItemId)
		if spec == nil || spec.RarityTier != rarityTier {
			continue
		}
		room := e.MaxStock - e.Current
		if room <= 0 {
			continue
		}
		add := e.RestockQty
		if add > room {
			add = room
		}
		e.Current += add
		restocked = true
	}
	return restocked
}
```

- [ ] **Step 4: Run — pass**

`go test ./internal/shops/... -run TestRestockTier`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shops/shopinventory.go internal/shops/shopinventory_test.go
git commit -m "feat(shops): RestockTier method tops up only matching rarity tier"
```

### Task 5: Per-tier last-restock tracking on ShopInventory

**Files:**
- Modify: `internal/shops/shopinventory.go`

- [ ] **Step 1: Add LastRestockByTier field**

In `ShopInventory` struct:

```go
// LastRestockByTier records the round of the most recent restock
// per rarity tier. Replaces the single LastRestock for the
// per-tier cadence model. Persisted; zero value for a tier means
// "never restocked at this tier yet" — callers initialize to
// currentRound on first encounter.
LastRestockByTier map[int]uint64 `yaml:"last_restock_by_tier,omitempty"`
```

Keep `LastRestock` field as-is for now — Phase 1 supplements rather
than replaces, so older snapshots load fine.

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add internal/shops/shopinventory.go
git commit -m "feat(shops): LastRestockByTier map on ShopInventory persistence"
```

### Task 6: Per-tier ticker firing in TickMobShopRestock

**Files:**
- Modify: `internal/mobs/crafter.go` (lines around 137-164)

- [ ] **Step 1: Replace TickMobShopRestock body**

Replace the existing function with:

```go
// TickMobShopRestock fires per-tier restock cycles based on each
// rarity tier's configured cadence. A shop with stock entries across
// multiple tiers sees fast cycles for commons (tier 50) and slow
// cycles for rares (tier 10), matching the per-tier cadence config.
//
// Returns true if any tier fired this tick.
func TickMobShopRestock(mob *Mob) bool {
	if mob.Crafter {
		return false // crafters use TickMobCraft path
	}
	shopInv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)
	if shopInv == nil {
		return false
	}
	if shopInv.LastRestockByTier == nil {
		shopInv.LastRestockByTier = map[int]uint64{}
	}

	b := configs.GetBalanceConfig()
	roundCount := util.GetRoundCount()
	roundsPerHour := uint64(util.RoundsPerGameHour())

	anyFired := false
	for _, tier := range []int{50, 40, 30, 20, 10} {
		hours := shops.RestockCadenceHours(b, tier)
		if hours <= 0 {
			continue
		}
		cadence := uint64(hours) * roundsPerHour
		last := shopInv.LastRestockByTier[tier]
		if last == 0 {
			shopInv.LastRestockByTier[tier] = roundCount
			continue
		}
		if roundCount-last < cadence {
			continue
		}
		shopInv.LastRestockByTier[tier] = roundCount
		if shopInv.RestockTier(tier) {
			anyFired = true
		}
	}
	return anyFired
}
```

If `util.RoundsPerGameHour()` doesn't exist, use the existing
`configs.GetTimingConfig().RoundsPerGameDay() / 24` path or grep
for how `gametime` package converts round counts to hours.

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 3: Quick smoke test**

If `crafter_test.go` exists, run `go test ./internal/mobs/...`
Expected: PASS or no related failures.

- [ ] **Step 4: Commit**

```bash
git add internal/mobs/crafter.go
git commit -m "refactor(crafter): per-tier restock cadence in TickMobShopRestock

Each rarity tier ticks on its own cadence — commons every 1h,
rares every 5d, etc. Replaces the single CrafterMaterialRestockRate
gate. Last-restock per tier persists in LastRestockByTier."
```

### Task 7: Same per-tier model in TickMobCraft (crafter shops)

**Files:**
- Modify: `internal/mobs/crafter.go` (TickMobCraft, around line 166)

- [ ] **Step 1: Replace the restock-cadence gate inside TickMobCraft**

Find the section (around line 180-194) that reads:

```go
restockRate := uint64(b.CrafterMaterialRestockRate)
if restockRate == 0 {
    return nil
}
if mob.crafterLastRestockRound == 0 {
    mob.crafterLastRestockRound = roundCount
    return nil
}
if roundCount-mob.crafterLastRestockRound < restockRate {
    return nil
}
mob.crafterLastRestockRound = roundCount
```

Replace with: take the per-tier-aware path (mirror Task 6), then
continue into the crafting decision logic. The function signature
returns `*CraftResult` so the inner loop sets `restocked = true`
when any tier fired.

```go
shopInv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)
if shopInv == nil {
    return nil
}
if shopInv.LastRestockByTier == nil {
    shopInv.LastRestockByTier = map[int]uint64{}
}

roundsPerHour := uint64(util.RoundsPerGameHour())
restocked := false
for _, tier := range []int{50, 40, 30, 20, 10} {
    hours := shops.RestockCadenceHours(b, tier)
    if hours <= 0 {
        continue
    }
    cadence := uint64(hours) * roundsPerHour
    last := shopInv.LastRestockByTier[tier]
    if last == 0 {
        shopInv.LastRestockByTier[tier] = roundCount
        continue
    }
    if roundCount-last < cadence {
        continue
    }
    shopInv.LastRestockByTier[tier] = roundCount
    if b.IsCaravanServedZone(mob.Zone) && (tier == 50 || tier == 40) {
        // Caravan-served: foragers/caravans handle 30/20/10. Per-tier
        // ticker only fires on baseline tiers in those zones, leaving
        // rarer tiers to their natural input paths.
        if shopInv.RestockTier(tier) {
            restocked = true
        }
    } else if !b.IsCaravanServedZone(mob.Zone) {
        if shopInv.RestockTier(tier) {
            restocked = true
        }
    }
}
```

- [ ] **Step 2: Continue with the existing crafting decision logic**

Keep the recipe selection / craft attempt logic below intact. It now
runs on every tick rather than gated on the single restock cadence —
that's intentional: crafting decisions should fire on the smallest
cadence (commons hourly) so the crafter can react to depleted
ingredients quickly.

- [ ] **Step 3: Build + test**

```
go build ./... && go test ./internal/mobs/...
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/mobs/crafter.go
git commit -m "refactor(crafter): per-tier cadence in TickMobCraft

Crafter shops use the same per-tier restock model. In caravan-served
zones the ticker still only fires baseline tiers (50, 40); the rarer
tiers depend on caravan/forager flow as before."
```

### Task 8: Remove old CrafterMaterialRestockRate dependence

**Files:**
- Modify: `internal/configs/balance.go`
- Modify: `_datafiles/config.yaml`

- [ ] **Step 1: Mark CrafterMaterialRestockRate as deprecated**

Add a comment above the field and leave the value in place (don't
remove until external consumers verified). At top of `balance.go`:

```go
// Deprecated: replaced by RestockCadenceTier{50,40,30,20,10}*. Kept
// only so old config.yaml files load without error. Remove after
// one deploy cycle.
CrafterMaterialRestockRate ConfigInt `yaml:"CrafterMaterialRestockRate"`
```

- [ ] **Step 2: Verify no remaining consumers**

Run:
```
grep -rn "CrafterMaterialRestockRate" internal/
```
Expected: only the field declaration in `balance.go`. No code-path
reads it after Tasks 6-7.

- [ ] **Step 3: Commit**

```bash
git add internal/configs/balance.go
git commit -m "chore(config): mark CrafterMaterialRestockRate deprecated"
```

---

## Phase 2 — Persistent counters

Adds the data-layer fields that all the new scoring functions read.
Counters accumulate without consumers in this phase; Phase 3 wires
them into scoring.

### Task 9: New StockEvent type and ShopInventory fields

**Files:**
- Modify: `internal/shops/shopinventory.go`

- [ ] **Step 1: Add the type and fields**

Above the `ShopInventory` struct definition:

```go
// StockEvent records one depletion→refill cycle for a single item.
// DepletedRound is the round Current dropped to 0; RefilledRound is
// the round Current returned > 0. RefilledRound = 0 means the
// event is still ongoing (item currently depleted).
type StockEvent struct {
	DepletedRound uint64 `yaml:"depleted_round"`
	RefilledRound uint64 `yaml:"refilled_round"`
}
```

In `ShopInventory`, add after `KnownRecipes`:

```go
// SalesCount is the cumulative number of items the shop has sold
// to players. Drives the throughput score's "items moved" axis.
SalesCount int `yaml:"sales_count,omitempty"`

// BuysCount is the cumulative number of items the shop has bought
// FROM players.
BuysCount int `yaml:"buys_count,omitempty"`

// RestockCount is the cumulative number of items added to stock
// via fired restock cycles (not foragers/caravans/player sales).
// Drives input rate scoring.
RestockCount int `yaml:"restock_count,omitempty"`

// StockEvents holds completed depletion→refill events per item,
// rolling 7-game-day window. Drives Time-to-Refill (TtR) scoring.
StockEvents map[int][]StockEvent `yaml:"stock_events,omitempty"`

// CurrentDepletion records the round each item went to 0 and is
// still depleted. Cleared when the item refills (event pushed to
// StockEvents). Items not present here are not currently depleted.
CurrentDepletion map[int]uint64 `yaml:"current_depletion,omitempty"`
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add internal/shops/shopinventory.go
git commit -m "feat(shops): persistent counters and StockEvent tracking on ShopInventory"
```

### Task 10: Hook AddStock to record refills

**Files:**
- Modify: `internal/shops/shopinventory.go` (AddStock method)
- Modify: `internal/shops/shopinventory_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestAddStock_PushesStockEventOnRefill(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 0}},
		CurrentDepletion: map[int]uint64{100: 1000},
	}
	si.AddStockAtRound(100, 5, 1500)
	if len(si.StockEvents[100]) != 1 {
		t.Fatalf("expected 1 stock event, got %d", len(si.StockEvents[100]))
	}
	ev := si.StockEvents[100][0]
	if ev.DepletedRound != 1000 || ev.RefilledRound != 1500 {
		t.Errorf("event = %+v, want {1000, 1500}", ev)
	}
	if _, still := si.CurrentDepletion[100]; still {
		t.Errorf("CurrentDepletion[100] should be cleared")
	}
}

func TestAddStock_NoEventIfNotDepleted(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 3}},
	}
	si.AddStockAtRound(100, 5, 1500)
	if len(si.StockEvents[100]) != 0 {
		t.Errorf("no event expected when item wasn't fully depleted, got %d", len(si.StockEvents[100]))
	}
}
```

- [ ] **Step 2: Run — fail**

`go test ./internal/shops/... -run TestAddStock_Pushes`
Expected: FAIL — `AddStockAtRound` undefined.

- [ ] **Step 3: Implement**

Replace `AddStock` with this and add `AddStockAtRound`:

```go
// AddStock increases current stock for an item, capped at MaxStock.
// Use AddStockAtRound when you want depletion-event tracking
// (Phase-2 throughput scoring).
func (si *ShopInventory) AddStock(itemId int, qty int) {
	si.AddStockAtRound(itemId, qty, 0)
}

// AddStockAtRound is the round-aware variant: when round > 0 and the
// item is currently depleted (Current was 0 before this call), push
// a completed StockEvent into history and clear CurrentDepletion.
// round = 0 means "don't track" (used by Phase-2-naive callers
// that don't have a round handy).
func (si *ShopInventory) AddStockAtRound(itemId int, qty int, round uint64) {
	entry := si.GetStock(itemId)
	if entry == nil {
		si.Stock = append(si.Stock, StockEntry{
			ItemId:     itemId,
			RestockQty: 0,
			MaxStock:   20,
			Current:    0,
		})
		entry = &si.Stock[len(si.Stock)-1]
	}
	wasDepleted := entry.Current == 0
	entry.Current += qty
	if entry.Current > entry.MaxStock {
		entry.Current = entry.MaxStock
	}
	if !wasDepleted || qty <= 0 || round == 0 {
		return
	}
	if si.CurrentDepletion == nil {
		return
	}
	depRound, ok := si.CurrentDepletion[itemId]
	if !ok {
		return
	}
	if si.StockEvents == nil {
		si.StockEvents = map[int][]StockEvent{}
	}
	si.StockEvents[itemId] = append(si.StockEvents[itemId], StockEvent{
		DepletedRound: depRound,
		RefilledRound: round,
	})
	delete(si.CurrentDepletion, itemId)
}
```

- [ ] **Step 4: Run — pass**

`go test ./internal/shops/... -run TestAddStock`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shops/shopinventory.go internal/shops/shopinventory_test.go
git commit -m "feat(shops): AddStockAtRound pushes StockEvent on refill from depleted"
```

### Task 11: Hook RemoveStock to mark depletion

**Files:**
- Modify: `internal/shops/shopinventory.go`
- Modify: `internal/shops/shopinventory_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestRemoveStock_MarksDepletionWhenHittingZero(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 3}},
	}
	si.RemoveStockAtRound(100, 3, 2000)
	got, ok := si.CurrentDepletion[100]
	if !ok {
		t.Fatalf("expected CurrentDepletion[100] to be set")
	}
	if got != 2000 {
		t.Errorf("CurrentDepletion[100] = %d, want 2000", got)
	}
}

func TestRemoveStock_NoMarkIfNotZero(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 5}},
	}
	si.RemoveStockAtRound(100, 2, 2000)
	if _, ok := si.CurrentDepletion[100]; ok {
		t.Errorf("CurrentDepletion[100] should not be set when stock > 0")
	}
}
```

- [ ] **Step 2: Run — fail**

`go test ./internal/shops/... -run TestRemoveStock_Marks`

- [ ] **Step 3: Implement**

Replace `RemoveStock` and add `RemoveStockAtRound`:

```go
// RemoveStock decreases current stock by qty. Returns actual amount
// removed.
func (si *ShopInventory) RemoveStock(itemId int, qty int) int {
	return si.RemoveStockAtRound(itemId, qty, 0)
}

// RemoveStockAtRound is the round-aware variant: when round > 0 and
// the removal causes Current to hit 0, mark CurrentDepletion[itemId]
// = round so a later AddStockAtRound can produce a completed event.
func (si *ShopInventory) RemoveStockAtRound(itemId int, qty int, round uint64) int {
	entry := si.GetStock(itemId)
	if entry == nil || entry.Current <= 0 {
		return 0
	}
	removed := qty
	if removed > entry.Current {
		removed = entry.Current
	}
	entry.Current -= removed
	if entry.Current == 0 && round > 0 {
		if si.CurrentDepletion == nil {
			si.CurrentDepletion = map[int]uint64{}
		}
		if _, exists := si.CurrentDepletion[itemId]; !exists {
			si.CurrentDepletion[itemId] = round
		}
	}
	return removed
}
```

- [ ] **Step 4: Run — pass**

`go test ./internal/shops/... -run TestRemoveStock`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shops/shopinventory.go internal/shops/shopinventory_test.go
git commit -m "feat(shops): RemoveStockAtRound marks CurrentDepletion when hitting zero"
```

### Task 12: Wire round-aware Add/RemoveStock through buy/sell paths

**Files:**
- Modify: `internal/usercommands/buy.go`
- Modify: `internal/usercommands/sell.go`

- [ ] **Step 1: Update sell.go ShopInventory branch**

In `Sell()` (and `trySellOne()` if extracted by the multi-sell change),
locate the `shopInv.AddStock(item.ItemId, 1)` calls (lines around
131-145) and add `RestockCount`-aware variant. For each:

```go
shopInv.AddStockAtRound(item.ItemId, 1, util.GetRoundCount())
shopInv.BuysCount++
```

(The `BuysCount` increment goes here because *the shop bought from
the player*. From the player's POV they sold; from the shop's POV
they bought.)

- [ ] **Step 2: Update buy.go ShopInventory branch**

In `tryPurchaseFromInventory()`, around line 771 where the existing
code reads `if shopInv.RemoveStock(matched.entry.ItemId, 1) == 0`,
replace with:

```go
if shopInv.RemoveStockAtRound(matched.entry.ItemId, 1, util.GetRoundCount()) == 0 {
    ...
}
shopInv.SalesCount++
```

- [ ] **Step 3: Build + test**

```
go build ./... && go test ./internal/usercommands/...
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/buy.go internal/usercommands/sell.go
git commit -m "feat(commands): wire SalesCount/BuysCount and depletion tracking through buy/sell"
```

### Task 13: Increment RestockCount in RestockTier and friends

**Files:**
- Modify: `internal/shops/shopinventory.go`

- [ ] **Step 1: Update RestockTier to increment RestockCount**

In `RestockTier`, change the inner block from `e.Current += add` to:

```go
e.Current += add
si.RestockCount += add
restocked = true
```

Repeat the increment in `Restock()`, `RestockBuckets()`, and
`RestockBaselineTiers()`. Anywhere a restock cycle adds to stock
should bump `RestockCount`.

- [ ] **Step 2: Quick test**

Add to `shopinventory_test.go`:

```go
func TestRestockTier_IncrementsRestockCount(t *testing.T) {
	si := &ShopInventory{
		Stock: []StockEntry{{ItemId: testTier50ItemA, RestockQty: 5, MaxStock: 10, Current: 5}},
	}
	si.RestockTier(50)
	if si.RestockCount != 5 {
		t.Errorf("RestockCount = %d, want 5", si.RestockCount)
	}
}
```

Run and verify pass: `go test ./internal/shops/... -run TestRestockTier_Increments`

- [ ] **Step 3: Commit**

```bash
git add internal/shops/shopinventory.go internal/shops/shopinventory_test.go
git commit -m "feat(shops): increment RestockCount on every restock-driven add"
```

### Task 14: LbsDelivered counter on Caravan

**Files:**
- Modify: `internal/caravan/throughput.go`
- Modify: `internal/caravan/throughput_test.go`

- [ ] **Step 1: Inspect existing throughput.go**

Read the file. The caravan already has a delivery-event hook (this
is where caravan cargo is handed to a shop). Add a cumulative
counter. Find the function/method that performs the delivery
operation — it should be where `RestockBuckets` or `AddStock` is
called from the caravan's POV.

- [ ] **Step 2: Add LbsDelivered field**

In whichever struct represents the caravan's persistent state
(probably a `Caravan` or `CaravanState` type), add:

```go
// LbsDelivered is the cumulative pounds of cargo this caravan has
// delivered to shops over its lifetime. Drives logistics health
// scoring (cargo_score axis).
LbsDelivered uint64 `yaml:"lbs_delivered,omitempty"`
```

- [ ] **Step 3: Increment on delivery**

In the delivery code path, after the cargo transfer succeeds,
increment by `cargo.WeightLbs()` (or whatever the property is).
Use the existing cargo-weight accessor pattern from
`internal/caravan/wagon.go`.

- [ ] **Step 4: Add a test**

In `throughput_test.go`:

```go
func TestCaravan_LbsDeliveredAccumulates(t *testing.T) {
	c := newTestCaravan() // existing helper
	c.deliver(150) // deliver 150 lbs
	c.deliver(75)
	if c.LbsDelivered != 225 {
		t.Errorf("LbsDelivered = %d, want 225", c.LbsDelivered)
	}
}
```

- [ ] **Step 5: Build + run test**

```
go build ./... && go test ./internal/caravan/...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/caravan/throughput.go internal/caravan/throughput_test.go
git commit -m "feat(caravan): LbsDelivered cumulative counter for logistics scoring"
```

### Task 15: LbsDelivered counter on Forager

**Files:**
- Modify: `internal/forager/throughput.go`
- Modify: `internal/forager/throughput_test.go`

Mirror Task 14 exactly. Add `LbsDelivered uint64` to the forager's
persistent struct, increment on each delivery, add an analogous
test.

- [ ] **Step 1: Add field, increment, test**

Same pattern as Task 14. Use the matching forager delivery code
path.

- [ ] **Step 2: Build + run test**

```
go build ./... && go test ./internal/forager/...
```
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/forager/throughput.go internal/forager/throughput_test.go
git commit -m "feat(forager): LbsDelivered cumulative counter for logistics scoring"
```

### Task 16: Capture new fields in Snapshot

**Files:**
- Modify: `internal/economy/health/snapshot.go`
- Modify: `internal/economy/health/capture.go`

- [ ] **Step 1: Add to ShopSnapshot**

```go
type ShopSnapshot struct {
    // ... existing fields ...
    SalesCount       int                   `yaml:"sales_count"        json:"sales_count"`
    BuysCount        int                   `yaml:"buys_count"         json:"buys_count"`
    RestockCount     int                   `yaml:"restock_count"      json:"restock_count"`
    StockEvents      map[int][]StockEvent  `yaml:"stock_events"       json:"stock_events"`
    CurrentDepletion map[int]uint64        `yaml:"current_depletion"  json:"current_depletion"`
}

type StockEvent struct {
    DepletedRound uint64 `yaml:"depleted_round" json:"depleted_round"`
    RefilledRound uint64 `yaml:"refilled_round" json:"refilled_round"`
}
```

- [ ] **Step 2: Add to CaravanSnapshot and ForagerSnapshot**

```go
LbsDelivered uint64 `yaml:"lbs_delivered" json:"lbs_delivered"`
```

- [ ] **Step 3: Wire capture.go**

In `captureShop()`, copy the new fields from `ShopInventory` to
`ShopSnapshot`. Use a deep copy for the maps so subsequent
mutations don't bleed into the snapshot.

```go
shopSnap.SalesCount = shopInv.SalesCount
shopSnap.BuysCount = shopInv.BuysCount
shopSnap.RestockCount = shopInv.RestockCount

if len(shopInv.StockEvents) > 0 {
    shopSnap.StockEvents = make(map[int][]StockEvent, len(shopInv.StockEvents))
    for k, v := range shopInv.StockEvents {
        cp := make([]StockEvent, len(v))
        for i, e := range v {
            cp[i] = StockEvent{DepletedRound: e.DepletedRound, RefilledRound: e.RefilledRound}
        }
        shopSnap.StockEvents[k] = cp
    }
}
if len(shopInv.CurrentDepletion) > 0 {
    shopSnap.CurrentDepletion = make(map[int]uint64, len(shopInv.CurrentDepletion))
    for k, v := range shopInv.CurrentDepletion {
        shopSnap.CurrentDepletion[k] = v
    }
}
```

In `captureCaravan` and `captureForager`, copy `LbsDelivered`.

- [ ] **Step 4: Build + test**

```
go build ./... && go test ./internal/economy/health/...
```

- [ ] **Step 5: Commit**

```bash
git add internal/economy/health/snapshot.go internal/economy/health/capture.go
git commit -m "feat(health): capture sales/buys/restocks/events in snapshot"
```

---

## Phase 3 — Scoring functions

### Task 17: Config knobs for TtR targets, score weights, logistics

**Files:**
- Modify: `internal/configs/balance.go`
- Modify: `_datafiles/config.yaml`

- [ ] **Step 1: Add config fields**

```go
// TtR targets per rarity tier (game-time)
TtRTargetTier50Hours ConfigInt `yaml:"TtRTargetTier50Hours"`
TtRTargetTier40Hours ConfigInt `yaml:"TtRTargetTier40Hours"`
TtRTargetTier30Hours ConfigInt `yaml:"TtRTargetTier30Hours"`
TtRTargetTier20Days  ConfigInt `yaml:"TtRTargetTier20Days"`
TtRTargetTier10Days  ConfigInt `yaml:"TtRTargetTier10Days"`

// Throughput rolling window
TtRWindowGameDays ConfigInt `yaml:"TtRWindowGameDays"`

// Logistics
LogisticsStuckRounds      ConfigInt `yaml:"LogisticsStuckRounds"`
LogisticsStuckMultiplier  ConfigFloat64 `yaml:"LogisticsStuckMultiplier"`

// Overall blend
ScoreWeightStock      ConfigFloat64 `yaml:"ScoreWeightStock"`
ScoreWeightInput      ConfigFloat64 `yaml:"ScoreWeightInput"`
ScoreWeightThroughput ConfigFloat64 `yaml:"ScoreWeightThroughput"`
ScoreWeightShopGold   ConfigFloat64 `yaml:"ScoreWeightShopGold"`
```

- [ ] **Step 2: SetDefaults**

```go
if c.TtRTargetTier50Hours == 0 { c.TtRTargetTier50Hours = 3 }
if c.TtRTargetTier40Hours == 0 { c.TtRTargetTier40Hours = 6 }
if c.TtRTargetTier30Hours == 0 { c.TtRTargetTier30Hours = 18 }
if c.TtRTargetTier20Days  == 0 { c.TtRTargetTier20Days  = 3 }
if c.TtRTargetTier10Days  == 0 { c.TtRTargetTier10Days  = 7 }
if c.TtRWindowGameDays    == 0 { c.TtRWindowGameDays    = 7 }
if c.LogisticsStuckRounds == 0 { c.LogisticsStuckRounds = 3000 }
if c.LogisticsStuckMultiplier == 0 { c.LogisticsStuckMultiplier = 0.4 }
if c.ScoreWeightStock      == 0 { c.ScoreWeightStock      = 0.40 }
if c.ScoreWeightInput      == 0 { c.ScoreWeightInput      = 0.30 }
if c.ScoreWeightThroughput == 0 { c.ScoreWeightThroughput = 0.20 }
if c.ScoreWeightShopGold   == 0 { c.ScoreWeightShopGold   = 0.10 }
```

- [ ] **Step 3: Add YAML keys to config.yaml**

Add a block under the existing `Balance:` section:

```yaml
  # Economy scoring — TtR targets per rarity tier (game-time)
  TtRTargetTier50Hours: 3
  TtRTargetTier40Hours: 6
  TtRTargetTier30Hours: 18
  TtRTargetTier20Days:  3
  TtRTargetTier10Days:  7
  TtRWindowGameDays:    7

  # Logistics health
  LogisticsStuckRounds:     3000
  LogisticsStuckMultiplier: 0.4

  # Overall score blend (must sum to 1.0)
  ScoreWeightStock:      0.40
  ScoreWeightInput:      0.30
  ScoreWeightThroughput: 0.20
  ScoreWeightShopGold:   0.10
```

- [ ] **Step 4: Build**

`go build ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add internal/configs/balance.go _datafiles/config.yaml
git commit -m "feat(config): TtR targets, score weights, logistics knobs"
```

### Task 18: StockScore (rename + alias to PerShopScore)

**Files:**
- Modify: `internal/economy/health/scoring.go`
- Modify: `internal/economy/health/scoring_test.go`

- [ ] **Step 1: Rename PerShopScore → StockScore (keep alias)**

Add to the top of the existing `PerShopScore` definition:

```go
// StockScore returns the per-shop weighted-fill-percentage. This
// is the dashboard's "Stock Score" — answers "is there material
// on the shelves right now?" Identical to the historical
// PerShopScore (the alias is kept for callers).
func StockScore(s ShopSnapshot) float64 {
	v, _ := StockScoreOpt(s)
	return v
}

// StockScoreOpt is the option-returning variant used when callers
// need to distinguish "no signal" from "0%."
func StockScoreOpt(s ShopSnapshot) (float64, bool) {
	return PerShopScoreOpt(s)
}
```

Leave `PerShopScore` and `PerShopScoreOpt` unchanged — they delegate
to the same math.

- [ ] **Step 2: Build + test**

`go build ./... && go test ./internal/economy/health/...`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/economy/health/scoring.go
git commit -m "refactor(health): StockScore alias for PerShopScore (dashboard rename)"
```

### Task 19: ThroughputScore (TtR-based)

**Files:**
- Modify: `internal/economy/health/scoring.go`
- Modify: `internal/economy/health/scoring_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestThroughputScore_NoEventsEverIsHundred(t *testing.T) {
	// A shop with stock but no depletion history reads as "always
	// available" — score 100.
	snap := ShopSnapshot{
		Stock: []StockSnapshot{
			{ItemId: 100, Tier: 50, Current: 5, Max: 10},
		},
	}
	got := ThroughputScore(snap, 1000, &testCfg{})
	if got < 99 {
		t.Errorf("no-history shop = %.1f, want ~100", got)
	}
}

func TestThroughputScore_FastRefillIsHundred(t *testing.T) {
	// One tier-50 item, ttr = 1h, target = 3h.
	snap := ShopSnapshot{
		Stock: []StockSnapshot{{ItemId: 100, Tier: 50, Current: 5, Max: 10}},
		StockEvents: map[int][]StockEvent{
			100: {{DepletedRound: 1000, RefilledRound: 1000 + roundsForHours(1, &testCfg{})}},
		},
	}
	got := ThroughputScore(snap, 9999, &testCfg{})
	if got < 95 {
		t.Errorf("1h ttr vs 3h target = %.1f, want >=95", got)
	}
}

func TestThroughputScore_SlowRefillIsLow(t *testing.T) {
	// tier-50 item, ttr = 6h, target = 3h → score should be near 0.
	snap := ShopSnapshot{
		Stock: []StockSnapshot{{ItemId: 100, Tier: 50, Current: 0, Max: 10}},
		StockEvents: map[int][]StockEvent{
			100: {{DepletedRound: 1000, RefilledRound: 1000 + roundsForHours(6, &testCfg{})}},
		},
	}
	got := ThroughputScore(snap, 9999, &testCfg{})
	if got > 5 {
		t.Errorf("6h ttr vs 3h target = %.1f, want <=5", got)
	}
}

func TestThroughputScore_CurrentlyDepletedDragsScore(t *testing.T) {
	// item depleted, no completed events; ongoing 4h-and-counting
	// should drag below the cutoff for tier-50.
	snap := ShopSnapshot{
		Stock: []StockSnapshot{{ItemId: 100, Tier: 50, Current: 0, Max: 10}},
		CurrentDepletion: map[int]uint64{100: 1000},
	}
	now := uint64(1000) + roundsForHours(4, &testCfg{})
	got := ThroughputScore(snap, now, &testCfg{})
	if got > 35 {
		t.Errorf("ongoing 4h depletion vs 3h target tier-50 = %.1f, want <=35", got)
	}
}
```

(Define `testCfg` as a stub Balance with the spec's TtR targets and
a known `RoundsPerGameHour`. Define `roundsForHours` as
`uint64(h * cfg.RoundsPerGameHour)`.)

- [ ] **Step 2: Run — fail**

`go test ./internal/economy/health/... -run TestThroughputScore`

- [ ] **Step 3: Implement ThroughputScore**

```go
// ttrTargetForTier returns the TtR target in rounds for a rarity
// tier, reading the configured per-tier values.
func ttrTargetForTier(tier int, cfg ScoringConfig) uint64 {
	rph := cfg.RoundsPerGameHour
	switch tier {
	case 50:
		return uint64(cfg.TtRTargetTier50Hours) * rph
	case 40:
		return uint64(cfg.TtRTargetTier40Hours) * rph
	case 30:
		return uint64(cfg.TtRTargetTier30Hours) * rph
	case 20:
		return uint64(cfg.TtRTargetTier20Days) * 24 * rph
	case 10:
		return uint64(cfg.TtRTargetTier10Days) * 24 * rph
	default:
		return 0
	}
}

// itemTtRScore returns the 0..1 throughput score for one item.
// Combines completed events in window with any ongoing depletion.
// Items with no signal score 1 (always-available).
func itemTtRScore(itemId int, tier int, snap ShopSnapshot, currentRound uint64, cfg ScoringConfig) float64 {
	target := ttrTargetForTier(tier, cfg)
	if target == 0 {
		return 1.0 // unknown tier: don't penalize
	}
	windowStart := uint64(0)
	windowRounds := uint64(cfg.TtRWindowGameDays) * 24 * cfg.RoundsPerGameHour
	if currentRound > windowRounds {
		windowStart = currentRound - windowRounds
	}

	var sumTtR uint64
	var n int
	for _, ev := range snap.StockEvents[itemId] {
		if ev.RefilledRound < windowStart {
			continue
		}
		sumTtR += ev.RefilledRound - ev.DepletedRound
		n++
	}
	// Currently depleted contributes ongoing duration as a partial event.
	if depRound, ok := snap.CurrentDepletion[itemId]; ok && currentRound > depRound {
		sumTtR += currentRound - depRound
		n++
	}
	if n == 0 {
		return 1.0
	}
	mean := float64(sumTtR) / float64(n)
	score := 1.0 - mean/float64(target)
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

// ThroughputScore returns the per-shop time-to-refill (TtR) score,
// 0-100. Items weighted by rarity_tier^2 so commons (50) drive the
// score 25× harder than rares (10).
func ThroughputScore(snap ShopSnapshot, currentRound uint64, cfg ScoringConfig) float64 {
	var weighted, totalWeight float64
	for _, e := range snap.Stock {
		w := float64(e.Tier * e.Tier)
		if w <= 0 {
			continue
		}
		s := itemTtRScore(e.ItemId, e.Tier, snap, currentRound, cfg)
		weighted += w * s
		totalWeight += w
	}
	if totalWeight == 0 {
		return 100
	}
	return 100 * weighted / totalWeight
}
```

(Define `ScoringConfig` as a small struct holding the values from
`configs.GetBalanceConfig()` to keep the scoring math testable
without depending on the global config.)

- [ ] **Step 4: Run — pass**

`go test ./internal/economy/health/... -run TestThroughputScore`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/economy/health/scoring.go internal/economy/health/scoring_test.go
git commit -m "feat(health): ThroughputScore based on time-to-refill, rarity-tier^2 weighted"
```

### Task 20: InputRateScore (per-zone)

**Files:**
- Modify: `internal/economy/health/scoring.go`
- Modify: `internal/economy/health/scoring_test.go`

- [ ] **Step 1: Write tests**

```go
func TestInputRateScore_BootstrapNoHistory(t *testing.T) {
	// Empty zone with no input → score 100 (bootstrap).
	got := InputRateScore("stillwater", &Snapshot{}, []*Snapshot{}, &testCfg{})
	if got < 99 {
		t.Errorf("empty bootstrap = %.1f, want ~100", got)
	}
}

func TestInputRateScore_LowInputIsLow(t *testing.T) {
	// Snapshot history shows 1 day passing with very few items in.
	cur := &Snapshot{
		Round: roundsForDays(2, &testCfg{}),
		Shops: []ShopSnapshot{
			{Zone: "stillwater", RestockCount: 5}, // only 5 items in 1 day
		},
	}
	prev := &Snapshot{
		Round: roundsForDays(1, &testCfg{}),
		Shops: []ShopSnapshot{
			{Zone: "stillwater", RestockCount: 0},
		},
	}
	got := InputRateScore("stillwater", cur, []*Snapshot{prev, cur}, &testCfg{})
	if got > 30 {
		t.Errorf("low input = %.1f, want <=30", got)
	}
}
```

- [ ] **Step 2: Run — fail**

- [ ] **Step 3: Implement**

```go
// InputRateScore measures items entering a zone's supply per
// game-day, weighted linearly by rarity_tier (commons drive volume).
// Compares against an auto-derived zone target.
func InputRateScore(zone string, cur *Snapshot, history []*Snapshot, cfg ScoringConfig) float64 {
	if cur == nil || len(history) < 2 {
		return 100
	}
	// Find the snapshot ~24h back.
	roundsPerDay := uint64(24) * cfg.RoundsPerGameHour
	if cur.Round < roundsPerDay {
		return 100 // not enough history
	}
	cutoff := cur.Round - roundsPerDay
	var prev *Snapshot
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Round <= cutoff {
			prev = history[i]
			break
		}
	}
	if prev == nil {
		return 100
	}

	// Sum input over the window. Forager deliveries by tier from
	// DeliveriesByTier; restock counts from per-shop RestockCount
	// delta.
	var inputWeighted float64
	for _, s := range cur.Shops {
		if s.Zone != zone {
			continue
		}
		var prevRestock int
		for _, p := range prev.Shops {
			if p.Zone == s.Zone && p.MobId == s.MobId {
				prevRestock = p.RestockCount
				break
			}
		}
		delta := s.RestockCount - prevRestock
		if delta < 0 {
			delta = 0
		}
		// Weighted by mean tier of stock (proxy for "what got restocked").
		meanTier := meanRarityTier(s.Stock)
		inputWeighted += float64(delta) * float64(meanTier)
	}
	for _, f := range cur.Foragers {
		if f.Zone != zone {
			continue
		}
		var prevDeliv map[int]int
		for _, p := range prev.Foragers {
			if p.Zone == f.Zone && p.MobId == f.MobId {
				prevDeliv = p.DeliveriesByTier
				break
			}
		}
		for tier, n := range f.DeliveriesByTier {
			delta := n - prevDeliv[tier]
			if delta < 0 {
				delta = 0
			}
			inputWeighted += float64(delta) * float64(tier)
		}
	}

	target := zoneInputTarget(zone, cur, cfg)
	if target <= 0 {
		return 100
	}
	score := 100 * inputWeighted / target
	if score > 100 {
		return 100
	}
	if score < 0 {
		return 0
	}
	return score
}

func zoneInputTarget(zone string, cur *Snapshot, cfg ScoringConfig) float64 {
	var target float64
	for _, s := range cur.Shops {
		if s.Zone != zone {
			continue
		}
		for _, e := range s.Stock {
			cadenceHours := cadenceHoursForTier(e.Tier, cfg)
			if cadenceHours == 0 {
				continue
			}
			target += (24.0 / float64(cadenceHours)) * float64(e.Tier)
		}
	}
	// Forager: 3 deliveries/day at typical capacity, mean tier 30.
	for _, f := range cur.Foragers {
		if f.Zone == zone && f.State != "(not active)" {
			target += 3.0 * float64(f.CargoCapacity) * 30.0 / 100.0
		}
	}
	return target
}

func meanRarityTier(stock []StockSnapshot) int {
	if len(stock) == 0 {
		return 30
	}
	sum := 0
	for _, e := range stock {
		sum += e.Tier
	}
	return sum / len(stock)
}

func cadenceHoursForTier(tier int, cfg ScoringConfig) int {
	switch tier {
	case 50:
		return cfg.RestockCadenceTier50Hours
	case 40:
		return cfg.RestockCadenceTier40Hours
	case 30:
		return cfg.RestockCadenceTier30Hours
	case 20:
		return cfg.RestockCadenceTier20Hours
	case 10:
		return cfg.RestockCadenceTier10Days * 24
	default:
		return 0
	}
}
```

If `ForagerSnapshot` doesn't already include `Zone`, add it in
`snapshot.go` capture.

- [ ] **Step 4: Run — pass**

`go test ./internal/economy/health/... -run TestInputRateScore`

- [ ] **Step 5: Commit**

```bash
git add internal/economy/health/scoring.go internal/economy/health/scoring_test.go
git commit -m "feat(health): InputRateScore — items/day weighted by tier vs zone target"
```

### Task 21: LogisticsHealth (caravans + foragers)

**Files:**
- Modify: `internal/economy/health/scoring.go`
- Modify: `internal/economy/health/scoring_test.go`

- [ ] **Step 1: Write tests**

```go
func TestLogisticsHealth_DespawnedIsZero(t *testing.T) {
	got := LogisticsHealth(LogisticsArgs{Despawned: true}, &testCfg{})
	if got != 0 {
		t.Errorf("despawned = %.1f, want 0", got)
	}
}

func TestLogisticsHealth_StuckMultiplier(t *testing.T) {
	args := LogisticsArgs{
		Cycles: 1, ExpectedCycles: 1,
		LbsDelivered: 5000, TargetLbs: 5000,
		Stuck: true,
	}
	got := LogisticsHealth(args, &testCfg{})
	// base = 100; stuck multiplier 0.4 → 40
	if got < 38 || got > 42 {
		t.Errorf("stuck-with-perfect-flow = %.1f, want ~40", got)
	}
}

func TestLogisticsHealth_HealthyComposite(t *testing.T) {
	args := LogisticsArgs{
		Cycles: 1, ExpectedCycles: 1,
		LbsDelivered: 5000, TargetLbs: 5000,
	}
	got := LogisticsHealth(args, &testCfg{})
	if got < 99 {
		t.Errorf("perfect = %.1f, want ~100", got)
	}
}
```

- [ ] **Step 2: Run — fail**

- [ ] **Step 3: Implement**

```go
// LogisticsArgs is the input bundle for LogisticsHealth, decoupled
// from caravan/forager structs so the same function scores both.
type LogisticsArgs struct {
	Cycles         int
	ExpectedCycles int
	LbsDelivered   uint64
	TargetLbs      uint64
	Stuck          bool
	Despawned      bool
}

// LogisticsHealth returns a 0-100 composite of cycle rate and cargo
// flow, with a hard multiplier for stuck (×0.4 default) and 0 for
// despawned. Used identically for caravans and foragers.
func LogisticsHealth(a LogisticsArgs, cfg ScoringConfig) float64 {
	if a.Despawned {
		return 0
	}
	cycleScore := 0.0
	if a.ExpectedCycles > 0 {
		cycleScore = 100.0 * float64(a.Cycles) / float64(a.ExpectedCycles)
		if cycleScore > 100 {
			cycleScore = 100
		}
	}
	cargoScore := 0.0
	if a.TargetLbs > 0 {
		cargoScore = 100.0 * float64(a.LbsDelivered) / float64(a.TargetLbs)
		if cargoScore > 100 {
			cargoScore = 100
		}
	}
	base := 0.5*cycleScore + 0.5*cargoScore
	if a.Stuck {
		base *= cfg.LogisticsStuckMultiplier
	}
	if base < 0 {
		return 0
	}
	if base > 100 {
		return 100
	}
	return base
}
```

- [ ] **Step 4: Run — pass**

- [ ] **Step 5: Commit**

```bash
git add internal/economy/health/scoring.go internal/economy/health/scoring_test.go
git commit -m "feat(health): LogisticsHealth composite for caravans+foragers"
```

### Task 22: ShopGoldScore

**Files:**
- Modify: `internal/economy/health/scoring.go`
- Modify: `internal/economy/health/scoring_test.go`

- [ ] **Step 1: Write tests**

```go
func TestShopGoldScore(t *testing.T) {
	cases := []struct {
		gold, starting int
		want           float64
	}{
		{0, 500, 0},
		{250, 500, 33.3},
		{500, 500, 66.6},
		{750, 500, 100},
		{1000, 500, 100}, // capped at 1.5×
	}
	for _, c := range cases {
		got := ShopGoldScore(ShopSnapshot{Gold: c.gold, StartingGold: c.starting})
		if !floatNear(got, c.want, 1.0) {
			t.Errorf("gold=%d/start=%d: got %.1f, want %.1f", c.gold, c.starting, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run — fail**

- [ ] **Step 3: Implement**

```go
// ShopGoldScore returns 0-100 based on merchant liquidity.
// gold/starting ratio clamped to [0, 1.5], then mapped linearly so
// 1.5× starting = 100, 1.0× = 67, 0 = 0. Captures the "merchant
// can't buy from players" failure mode.
func ShopGoldScore(s ShopSnapshot) float64 {
	if s.StartingGold <= 0 {
		return 100 // no baseline → don't penalize
	}
	ratio := float64(s.Gold) / float64(s.StartingGold)
	if ratio > 1.5 {
		ratio = 1.5
	}
	if ratio < 0 {
		ratio = 0
	}
	return ratio / 1.5 * 100
}
```

- [ ] **Step 4: Run — pass**

- [ ] **Step 5: Commit**

```bash
git add internal/economy/health/scoring.go internal/economy/health/scoring_test.go
git commit -m "feat(health): ShopGoldScore — merchant liquidity, gold/starting ratio"
```

### Task 23: New OverallScore blend + integrate into Score()

**Files:**
- Modify: `internal/economy/health/scoring.go`
- Modify: `internal/economy/health/scoring_test.go`

- [ ] **Step 1: Update Scores struct**

```go
type Scores struct {
    OverallScore float64

    PerShop         []ShopScoreRow
    PerCraftSupport map[string]float64
    PerCaravan      []EntityScoreRow
    PerForager      []EntityScoreRow

    // Component aggregates surfaced as the four cards.
    MeanStock      float64
    MeanThroughput float64
    MeanInput      float64
    MeanShopGold   float64
    // Logistics is per-entity only; not blended into Overall.

    // Back-compat alias surfaced for legacy dashboard JS.
    MeanShop     float64 // == MeanStock
    MeanCaravan  float64 // remains for legacy display
    MeanForager  float64 // remains for legacy display
}

type ShopScoreRow struct {
    Zone, MobId, RoomId int   // (use existing types)
    Name, CraftSupport  string
    StockScore          float64
    ThroughputScore     float64
    GoldScore           float64
    HasScore            bool
}
```

- [ ] **Step 2: Update Score() function**

In the existing `Score()` body, add per-shop throughput and gold,
plus per-zone input rate aggregation. Build the new MeanStock /
MeanThroughput / MeanInput / MeanShopGold means. Compute Overall as:

```go
const (
    wStock     = cfg.ScoreWeightStock
    wInput     = cfg.ScoreWeightInput
    wThru      = cfg.ScoreWeightThroughput
    wGold      = cfg.ScoreWeightShopGold
)
var weighted, weightSum float64
if shopCount > 0 {
    weighted += wStock * out.MeanStock
    weightSum += wStock
    weighted += wThru * out.MeanThroughput
    weightSum += wThru
    weighted += wGold * out.MeanShopGold
    weightSum += wGold
}
if inputZones > 0 {
    weighted += wInput * out.MeanInput
    weightSum += wInput
}
if weightSum > 0 {
    out.OverallScore = weighted / weightSum
}
out.MeanShop = out.MeanStock // back-compat
```

- [ ] **Step 3: Wire LogisticsHealth into PerCaravan/PerForager**

Replace the existing `PerCaravanScore` / `PerForagerScore` calls
inside `Score()` with `LogisticsHealth` invocations, building the
`LogisticsArgs` from snapshot data:

```go
// caravan: cycles from CountCaravanCycles; lbs delivered = current - earliest-in-window
args := LogisticsArgs{
    Cycles: cycles, ExpectedCycles: expected,
    LbsDelivered: deliveredInWindow, TargetLbs: target,
    Stuck: cur.Round - c.StateEnteredRound > stuckThresholdRounds,
    Despawned: c.State == "(not active)",
}
```

- [ ] **Step 4: Update existing scoring tests** to assert against the
new fields. Keep `PerShopScore` test cases untouched (they still
call the alias).

- [ ] **Step 5: Run all tests**

`go test ./internal/economy/health/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/economy/health/scoring.go internal/economy/health/scoring_test.go
git commit -m "feat(health): five-axis scoring — stock + throughput + input + gold + logistics

Score() now produces MeanStock, MeanThroughput, MeanInput,
MeanShopGold component means and a renormalized OverallScore using
the configured weights. PerCaravan and PerForager use LogisticsHealth
with stuck/despawned multipliers. Legacy MeanShop kept as alias."
```

---

## Phase 4 — Dashboard updates

### Task 24: API delta fields for new counters

**Files:**
- Modify: `internal/economy/health/delta.go`
- Modify: `internal/economy/health/delta_test.go`

- [ ] **Step 1: Add delta fields**

```go
type ShopDelta struct {
    GoldDelta       int
    BucketDeltas    map[string]int
    StockScoreDelta int
    SalesDelta      int
    BuysDelta       int
    RestocksDelta   int
    ThroughputScore float64 // computed at delta time using window
}

type CaravanDelta struct {
    DeliveriesByTierDelta map[int]int
    LbsDeliveredDelta     uint64
}

type ForagerDelta struct {
    DeliveriesByTierDelta map[int]int
    StuckRoundsDelta      int64
    LbsDeliveredDelta     uint64
}
```

- [ ] **Step 2: Update ComputeShopDelta**

Add SalesDelta = `now.SalesCount - old.SalesCount`, etc.

- [ ] **Step 3: Run delta tests**

`go test ./internal/economy/health/... -run TestComputeShop`

- [ ] **Step 4: Commit**

```bash
git add internal/economy/health/delta.go internal/economy/health/delta_test.go
git commit -m "feat(health): delta fields for sales/buys/restocks/lbs"
```

### Task 25: Top-card layout (5 cards)

**Files:**
- Modify: `_datafiles/html/admin/economy/index.html`

- [ ] **Step 1: Replace the top-card row**

Find the `<!-- Five summary cards -->` block. Replace the cards with:

```html
<div class="row mb-3">
    <div class="col-md-2"><div class="card text-center"><div class="card-body p-2">
        <h6 class="card-subtitle text-muted">Overall</h6>
        <h4 id="score-overall" class="card-title mb-0">—</h4>
    </div></div></div>
    <div class="col-md-2"><div class="card text-center"><div class="card-body p-2">
        <h6 class="card-subtitle text-muted">Stock</h6>
        <h4 id="score-stock" class="card-title mb-0">—</h4>
    </div></div></div>
    <div class="col-md-2"><div class="card text-center"><div class="card-body p-2">
        <h6 class="card-subtitle text-muted">Input</h6>
        <h4 id="score-input" class="card-title mb-0">—</h4>
    </div></div></div>
    <div class="col-md-2"><div class="card text-center"><div class="card-body p-2">
        <h6 class="card-subtitle text-muted">Throughput</h6>
        <h4 id="score-throughput" class="card-title mb-0">—</h4>
    </div></div></div>
    <div class="col-md-2"><div class="card text-center"><div class="card-body p-2">
        <h6 class="card-subtitle text-muted">Shop Gold</h6>
        <h4 id="score-gold" class="card-title mb-0">—</h4>
    </div></div></div>
</div>
```

(Drop the "Last Snapshot" card from this row; move it into a new row
below or omit if not actively used.)

- [ ] **Step 2: Update the render() function**

Replace the existing top-card setters:

```js
document.getElementById('score-overall').innerHTML = fmtScore(d.scores.OverallScore, true);
document.getElementById('score-stock').innerHTML = fmtScore(d.scores.MeanStock, d.scores.MeanStock > 0);
document.getElementById('score-input').innerHTML = fmtScore(d.scores.MeanInput, d.scores.MeanInput > 0);
document.getElementById('score-throughput').innerHTML = fmtScore(d.scores.MeanThroughput, d.scores.MeanThroughput > 0);
document.getElementById('score-gold').innerHTML = fmtScore(d.scores.MeanShopGold, d.scores.MeanShopGold > 0);
```

- [ ] **Step 3: Visual smoke**

Open the dashboard locally; verify all 5 cards render values.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/html/admin/economy/index.html
git commit -m "feat(dashboard): five-card top layout (overall/stock/input/throughput/gold)"
```

### Task 26: Stock table relabel + drop noisy delta cells + add gold column

**Files:**
- Modify: `_datafiles/html/admin/economy/index.html`

- [ ] **Step 1: Update Per-Shop Detail table header**

```html
<thead><tr>
    <th>Shop</th><th>Discipline</th><th>Stock Score</th>
    <th>Gold Score</th><th>Gold</th><th>Stock by Tier</th>
</tr></thead>
```

(Drop the per-window Δ stock pp cells — they were noise.)

- [ ] **Step 2: Update render loop for per-shop rows**

Replace the row-build to produce the new columns:

```js
shopHtml += '<tr><td>' + (row.Name || '#'+row.MobId) + '</td>' +
            '<td>' + (row.CraftSupport || '—') + '</td>' +
            '<td>' + fmtScore(row.StockScore, row.HasScore) + '</td>' +
            '<td>' + fmtScore(row.GoldScore, true) + '</td>' +
            '<td>' + snap.gold + '</td>' +
            '<td>' + tierBar(snap.stock, totalCap(snap.stock)) + '</td></tr>';
```

- [ ] **Step 3: Update Discipline rollup table similarly**

Drop the Δ columns; rename "Score" → "Stock Score". Optionally add a
Mean-Gold-Score column to the discipline rollup.

- [ ] **Step 4: Visual smoke**

- [ ] **Step 5: Commit**

```bash
git add _datafiles/html/admin/economy/index.html
git commit -m "feat(dashboard): stock table relabel, gold column, drop noisy delta cells"
```

### Task 27: Throughput table

**Files:**
- Modify: `_datafiles/html/admin/economy/index.html`

- [ ] **Step 1: Add the table HTML**

After the Per-Shop Detail table:

```html
<h4 class="mt-3">Throughput — Time-to-Refill</h4>
<table class="table table-sm table-striped">
    <thead><tr>
        <th>Shop</th><th>Score</th>
        <th>TtR (commons)</th><th>TtR (rares)</th><th>Currently depleted</th>
        <th>1h</th><th>6h</th><th>1d</th><th>3d</th><th>1w</th>
    </tr></thead>
    <tbody id="tbl-throughput"></tbody>
</table>
```

- [ ] **Step 2: Add the render code**

```js
function fmtTtR(rounds) {
    if (rounds == null || rounds <= 0) return '—';
    var hours = rounds / 600; // assume 600 rounds/game-hour; replace with API-served value
    if (hours < 1) return Math.round(hours * 60) + 'min';
    if (hours < 24) return hours.toFixed(1) + 'h';
    return (hours / 24).toFixed(1) + 'd';
}

var thruHtml = '';
for (var i = 0; i < shopPairs.length; i++) {
    var row = shopPairs[i].row;
    var snap = shopPairs[i].snap;
    var key = shopKey(snap);
    thruHtml += '<tr><td>' + (row.Name || '#'+row.MobId) + '</td>' +
                '<td>' + fmtScore(row.ThroughputScore, true) + '</td>' +
                '<td>' + fmtTtR(snap.median_ttr_commons) + '</td>' +
                '<td>' + fmtTtR(snap.median_ttr_rares) + '</td>' +
                '<td>' + (snap.currently_depleted_count || 0) + '</td>';
    ['1h','6h','1d','3d','1w'].forEach(function(label) {
        var sd = d.deltas[label] && d.deltas[label].shops[key];
        var ttr = sd ? sd.MedianTtR : 0;
        thruHtml += '<td>' + fmtTtR(ttr) + '</td>';
    });
    thruHtml += '</tr>';
}
document.getElementById('tbl-throughput').innerHTML = thruHtml;
```

The API needs to expose `median_ttr_commons`, `median_ttr_rares`,
`currently_depleted_count` per shop snapshot, and `MedianTtR` per
window-delta.

- [ ] **Step 3: Update API to expose those fields**

In `internal/economy/health/snapshot.go`, add to ShopSnapshot:

```go
MedianTtRCommons        uint64 `json:"median_ttr_commons"`
MedianTtRRares          uint64 `json:"median_ttr_rares"`
CurrentlyDepletedCount  int    `json:"currently_depleted_count"`
```

Compute these in `captureShop` from `StockEvents` + `CurrentDepletion`.
For the per-window MedianTtR in delta.go, compute from `StockEvents`
where `RefilledRound` falls inside the window.

- [ ] **Step 4: Visual smoke**

- [ ] **Step 5: Commit**

```bash
git add _datafiles/html/admin/economy/index.html internal/economy/health/snapshot.go internal/economy/health/capture.go internal/economy/health/delta.go
git commit -m "feat(dashboard): throughput table with TtR per-window cells"
```

### Task 28: Input rate table

**Files:**
- Modify: `_datafiles/html/admin/economy/index.html`

- [ ] **Step 1: Add the table HTML**

After the throughput table:

```html
<h4 class="mt-3">Input Rate — items/day per zone</h4>
<table class="table table-sm table-striped">
    <thead><tr>
        <th>Zone</th><th>Score</th>
        <th>Items/day (in)</th><th>From foragers</th><th>From restock</th>
        <th>Tier mix</th>
    </tr></thead>
    <tbody id="tbl-input"></tbody>
</table>
```

- [ ] **Step 2: Render**

API exposes `d.scores.InputRateByZone = map[zone]InputRateRow` where
`InputRateRow` has the per-source breakdown. Iterate that map.

- [ ] **Step 3: Add InputRateByZone to the API**

In `Scores`:

```go
type InputRateRow struct {
    Score          float64
    ItemsPerDay    float64
    FromForagers   float64
    FromRestock    float64
    TierMix        map[int]int
}
InputRateByZone map[string]InputRateRow
```

Populate in `Score()`.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/html/admin/economy/index.html internal/economy/health/scoring.go
git commit -m "feat(dashboard): input rate table per zone with source breakdown"
```

### Task 29: Logistics panels (caravans + foragers, with multipliers)

**Files:**
- Modify: `_datafiles/html/admin/economy/index.html`

- [ ] **Step 1: Update existing caravan and forager tables**

Add columns showing the multiplier currently active and lifetime
LbsDelivered. Header changes:

```html
<th>Multiplier</th><th>Lbs Delivered</th>
```

- [ ] **Step 2: Render the multiplier text**

```js
function multText(score, snap) {
    if (snap.state === '(not active)') return '<span class="text-danger">despawned</span>';
    if (snap.stuck_rounds && snap.stuck_rounds > stuckThreshold) {
        return '<span class="text-warning">stuck (×0.4)</span>';
    }
    return '—';
}
```

- [ ] **Step 3: Commit**

```bash
git add _datafiles/html/admin/economy/index.html
git commit -m "feat(dashboard): logistics panels show stuck/despawn multiplier active"
```

---

## Phase 5 — Sanity / smoke

### Task 30: Manual smoke verification

**Files:** none (verification only).

- [ ] **Step 1: Restart server, watch a tier-50 item cycle**

Watch for `RestockTier 50` firing on the 1h cadence. Verify the
console shows the tick and the StockEvents map gains entries when
items deplete and refill.

- [ ] **Step 2: Force a forager despawn**

(Document the dev command if one exists, or use admin tools.)
Verify `LogisticsHealth` for that forager goes to 0 within one
snapshot tick. Forager appears in the dashboard logistics panel
with "despawned" text.

- [ ] **Step 3: Hold a caravan in one state past LogisticsStuckRounds (3000 rounds)**

Verify the multiplier kicks in and the dashboard reflects "stuck
(×0.4)".

- [ ] **Step 4: If any of the smoke checks fail**, file fix
commits scoped to the specific issue. Common likely fixes: the
stuck threshold being too low for slow caravan routes (raise from
3000 → 5000), or the despawn detection key (forager `(not active)`
vs `MobId == 0`).

- [ ] **Step 5: Commit doc note**

Add to `PATCH_NOTES.md`:

```
## 2026-05-05 Economy scoring refactor
- Five-axis scoring: stock + throughput + input + gold + logistics
- Per-rarity-tier restock cadence (1h / 2h / 6h / 24h / 5d)
- TtR-based throughput score, weighted toward commons
- Logistics health surfaces stuck/despawned entities prominently
```

```bash
git add PATCH_NOTES.md
git commit -m "docs: PATCH_NOTES entry for economy scoring refactor"
```

---

## Self-review notes

- **Spec coverage:** Each of the 5 axes (stock, throughput, input,
  logistics, shop gold) has an implementation task. Per-tier restock
  cadence covered in Phase 1. Persistence migration is implicit (zero
  defaults; see Task 9 + spec migration section).
- **Type consistency:** `StockEvent` defined once in shopinventory.go
  and re-declared in snapshot.go (different package); JSON tags
  match. `LogisticsArgs` is the single input bundle for caravans
  and foragers — same function, two callers.
- **Phasing:** Phase 1 ships independently (engine cadence change,
  no scoring/UI). Phase 2 ships counters with no consumers (safe).
  Phase 3 turns on scoring. Phase 4 turns on UI. Phase 5 verifies.
  Each phase is mergeable on its own.
- **Open implementation choices for the executor:**
  - The exact form of `util.RoundsPerGameHour()` — there may be a
    different idiom in the codebase. Match what `gametime` package
    uses.
  - Test fixture style for items with explicit `RarityTier` —
    follow the existing pattern in `test_main_test.go`.
