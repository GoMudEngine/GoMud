# General-Store Restock Fix — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give non-crafter vendors in caravan-served zones a cadence-gated baseline common-tier (50/40) self-refill so general-store basics replenish, while rare goods stay caravan-gated.

**Architecture:** New `mobs.TickMobShopBaselineRestock` mirrors the existing `TickMobShopRestock` cadence pattern but calls `shopInv.RestockBaselineTiers()` (tier 50/40, `RestockQty>0` only). `MobIdle_HandleIdleMobs.go`'s caravan-served branch — which currently *skips* non-crafter restock entirely — calls it instead.

**Tech Stack:** Go; `internal/mobs`, `internal/shops`, `internal/hooks`.

**Spec:** `docs/superpowers/specs/2026-06-05-general-store-restock-design.md`

---

## Reference: verified code

- `TickMobShopRestock(mob)` (`mobs/crafter.go`): no-op if `mob.Crafter`; fetches
  `shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)`; per-tier
  cadence via `shopInv.LastRestockByTier` + `shops.RestockCadenceHours(b, tier)` ×
  `roundsPerHour` (`= GetTimingConfig().RoundsPerDay / 24`); calls `RestockTier`.
- `shopInv.RestockBaselineTiers() bool` (`shops/shopinventory.go`): tops up
  `RestockQty>0` entries whose `items.GetItemSpec(itemId).RarityTier` is 50 or 40,
  by `RestockQty` (capped at `MaxStock`). Already unit-tested.
- `MobIdle_HandleIdleMobs.go` ~lines 64-79: `if !IsCaravanServedZone(mob.Zone) {
  didRestock := mobs.TickMobShopRestock(mob); if didRestock { <supply-cart msg> } }`
  — caravan zones currently get nothing for non-crafters.
- Test fixtures: `shops.RegisterShop(zone, mobId, roomId, ShopInventory{...})`
  (seeds `Current` from `RestockQty` for supply-cart items), `shops.GetShopInventory(...)`,
  `items.RegisterTestItemSpec(&items.ItemSpec{ItemId, RarityTier})`,
  `util.SetRoundCountForTest(n)`, `inv.RemoveStock(itemId, qty)`.

---

## Task 1: `mobs.TickMobShopBaselineRestock` (TDD)

**Files:**
- Modify: `internal/mobs/crafter.go`
- Test: `internal/mobs/crafter_test.go`

- [ ] **Step 1: Write the failing tests** (append to `crafter_test.go`):
```go
func TestTickMobShopBaselineRestock_NoOpForCrafter(t *testing.T) {
	mob := &Mob{Crafter: true}
	if TickMobShopBaselineRestock(mob) {
		t.Fatal("crafter should no-op (uses TickMobCraft path)")
	}
}

func TestTickMobShopBaselineRestock_RefillsCommonTiersOnCadence(t *testing.T) {
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 7001, RarityTier: 50})
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 7002, RarityTier: 30})
	shops.RegisterShop("gsr_zone", 7700, 1, shops.ShopInventory{
		Zone:       "gsr_zone",
		StartingGold: 500,
		Stock: []shops.StockEntry{
			{ItemId: 7001, MaxStock: 20, RestockQty: 5}, // common — should refill
			{ItemId: 7002, MaxStock: 20, RestockQty: 5}, // rare tier 30 — caravan-gated
		},
	})
	// RegisterShop seeds Current from RestockQty; deplete both to 0 to test refill.
	inv := shops.GetShopInventory("gsr_zone", 7700, 1)
	inv.RemoveStock(7001, inv.GetStock(7001).Current)
	inv.RemoveStock(7002, inv.GetStock(7002).Current)

	mob := &Mob{Zone: "gsr_zone", HomeRoomId: 1}
	mob.MobId = 7700

	b := configs.GetBalanceConfig()
	roundsPerHour := uint64(configs.GetTimingConfig().RoundsPerDay) / 24
	cadence := uint64(shops.RestockCadenceHours(b, 50)) * roundsPerHour

	// First call stamps LastRestockByTier[50], does not restock.
	util.SetRoundCountForTest(100000)
	if TickMobShopBaselineRestock(mob) {
		t.Fatal("first call should stamp, not restock")
	}
	// Before cadence elapses: no restock.
	util.SetRoundCountForTest(100000 + cadence - 1)
	if TickMobShopBaselineRestock(mob) {
		t.Fatal("before cadence should not restock")
	}
	// After cadence: tier-50 refills by RestockQty; tier-30 untouched.
	util.SetRoundCountForTest(100000 + cadence + 1)
	if !TickMobShopBaselineRestock(mob) {
		t.Fatal("after cadence should restock the common tier")
	}
	inv = shops.GetShopInventory("gsr_zone", 7700, 1)
	if got := inv.GetStock(7001).Current; got != 5 {
		t.Fatalf("tier-50 entry: Current = %d, want 5", got)
	}
	if got := inv.GetStock(7002).Current; got != 0 {
		t.Fatalf("tier-30 entry should stay caravan-gated: Current = %d, want 0", got)
	}
}
```
(Confirm `crafter_test.go` already imports `items`, `shops`, `configs`, `util` — the existing tests use them; add any missing import.)

- [ ] **Step 2: Run — fails (undefined: TickMobShopBaselineRestock)**
Run: `go test ./internal/mobs/ -run TestTickMobShopBaselineRestock -v` → FAIL (undefined).

- [ ] **Step 3: Add `TickMobShopBaselineRestock` to `internal/mobs/crafter.go`** (next to `TickMobShopRestock`):
```go
// TickMobShopBaselineRestock gives a non-crafter vendor in a caravan-served
// zone the baseline common-tier (50/40) self-refill that crafters already get
// via TickMobCraft. Cadence-gated on the tier-50 restock cadence (keyed in
// LastRestockByTier[50]); tops up only RestockQty>0 tier-50/40 items via
// RestockBaselineTiers — rare goods (tier 30/20/10) stay caravan-gated. No-op
// for crafters (they use the TickMobCraft path) and for mobs without a shop.
// Returns true if any stock was added.
func TickMobShopBaselineRestock(mob *Mob) bool {
	if mob.Crafter {
		return false
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
	roundsPerHour := uint64(configs.GetTimingConfig().RoundsPerDay) / 24
	hours := shops.RestockCadenceHours(b, 50)
	if hours <= 0 {
		return false
	}
	cadence := uint64(hours) * roundsPerHour
	last := shopInv.LastRestockByTier[50]
	if last == 0 {
		shopInv.LastRestockByTier[50] = roundCount
		return false
	}
	if roundCount-last < cadence {
		return false
	}
	shopInv.LastRestockByTier[50] = roundCount
	return shopInv.RestockBaselineTiers()
}
```

- [ ] **Step 4: Run — passes**
Run: `go test ./internal/mobs/ -run TestTickMobShopBaselineRestock -v` → PASS (both tests).

- [ ] **Step 5: Commit**
```bash
go build ./... && go test ./internal/mobs/
git add internal/mobs/crafter.go internal/mobs/crafter_test.go
git commit -m "feat(shops): TickMobShopBaselineRestock — baseline common-tier refill for non-crafter vendors"
```

---

## Task 2: Wire it into the MobIdle caravan-zone branch

**Files:**
- Modify: `internal/hooks/MobIdle_HandleIdleMobs.go`

- [ ] **Step 1: Replace the caravan-zone skip** (the `restocked := false` / `if !IsCaravanServedZone {...}` block, ~lines 64-79) so both zone types share the supply-cart message:
```go
	restocked := false
	var didRestock bool
	if !configs.GetBalanceConfig().IsCaravanServedZone(mob.Zone) {
		didRestock = mobs.TickMobShopRestock(mob)
	} else {
		// Caravan-served zones: non-crafter vendors no longer skip restock —
		// they get the baseline common-tier (50/40) self-refill. Rare goods
		// still arrive only via the caravan. (TickMobShopBaselineRestock
		// no-ops for crafters and for mobs without a shop.)
		didRestock = mobs.TickMobShopBaselineRestock(mob)
	}
	if didRestock {
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			msgs := []string{
				`A supply cart pulls up outside. <ansi fg="mobname">%s</ansi> sorts through a fresh delivery.`,
				`<ansi fg="mobname">%s</ansi> unpacks a crate of supplies and restocks the shelves.`,
				`A runner drops off a bundle of goods. <ansi fg="mobname">%s</ansi> checks the contents and nods.`,
			}
			msg := fmt.Sprintf(msgs[util.Rand(len(msgs))], mob.Character.Name)
			sendVisualRoomText(room, messaging.CategoryMobIdle, msg)
		}
		restocked = true
	}
```
(This preserves the existing message block + the `restocked` flag used downstream; it just routes caravan-served non-crafters to the new baseline tick instead of skipping. Keep the surrounding code — the `restocked` variable's later use, the crafter `TickMobCraft` block — unchanged.)

- [ ] **Step 2: Build**
Run: `go build ./internal/hooks/` → clean. Confirm `mobs`, `configs`, `rooms`, `messaging`, `fmt`, `util` already imported (they are — the block already used them).

- [ ] **Step 3: Commit**
```bash
go build ./...
git add internal/hooks/MobIdle_HandleIdleMobs.go
git commit -m "feat(shops): caravan-zone non-crafter vendors run baseline restock (general-store supply)"
```

---

## Task 3: Verify Wulf's stock tiers (audit-only)

**Files:** (read-only audit; data edit only if a mis-tagged basic is found)
- Inspect: `_datafiles/world/dogmud/mobs/stillwater/341-storekeeper_wulf.yaml`

- [ ] **Step 1: Confirm Wulf's basics are tier 50/40 (or untagged → fallback 50)**
For each itemid in Wulf's `shop:` list, check the item's `rarity_tier` in its item YAML (`grep -rl "itemid: <id>"` or the items dir). Items meant to be common self-restocking basics (oil lantern 40038, water flask 40016, salt pouch 40017, wild vegetables 40015, wooden plank 40003, etc.) should resolve to tier 50 or 40 (untagged falls back to 50 — that's fine). 
- If a basic is deliberately tagged a rarer tier (30/20/10) but is meant to be a common self-restocking good, re-tag it to 40 or 50 in its item YAML.
- If all basics are 50/40/untagged, NO data change — note that in the report.
Expected: most/all are tier-50 (fallback) → no change. This step just guards against a basic being silently caravan-gated.

- [ ] **Step 2: Commit (only if a re-tag was needed)**
```bash
git add _datafiles/world/dogmud/items/...
git commit -m "content(items): re-tag <item> to common tier so the general store self-restocks it"
```
(If no change: skip the commit; note "no re-tag needed" in the report.)

---

## Task 4: Build, test, boot smoke

- [ ] **Step 1: Full build + test**
Run: `go build ./... && go test ./...` → build clean; all packages pass.

- [ ] **Step 2: Boot smoke** (wipe instances first; NOTE: a perf-capture server may be using port 33333 — only run this when the port is free):
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Boot; confirm clean load past data files. Then (admin): visit Wulf in Stillwater, buy down a common basic (e.g. water flask) to deplete it, idle past the tier-50 restock cadence, and confirm it climbs back (the "supply cart pulls up" message fires). Confirm a rare good (if any) does NOT self-refill. (In-game smoke MAY be deferred to user per precedent.)

No commit (verification only).

---

## Task 5: Docs + memory

**Files:**
- Modify: `internal/mobs/context.md` (or `internal/shops/context.md`)
- Memory: `project_store_restock_considered_fix` (general-store half → done)

- [ ] **Step 1: context.md note**
Note `TickMobShopBaselineRestock`: non-crafter vendors in caravan-served zones get the baseline common-tier (50/40) self-refill on their idle tick (the crafter analogue is `TickMobCraft` → `RestockBaselineTiers`); rare goods stay caravan-gated.

- [ ] **Step 2: Commit docs**
```bash
git add internal/mobs/context.md
git commit -m "docs(context): non-crafter caravan-zone baseline restock"
```

- [ ] **Step 3: Update memory**
`project_store_restock_considered_fix`: change "STILL OPEN: general-store restock" to RESOLVED (this chunk) — all three store types (cooking, enchanting, general) now supplied; note the new mechanism. Leave the Fernway `deliveries_by_tier` caravan-content gap as still-open. Update the MEMORY.md index line accordingly.

---

## Self-review notes

**Spec coverage:**
- Spec "new TickMobShopBaselineRestock (cadence-gated RestockBaselineTiers)" → Task 1.
- Spec "wire MobIdle caravan-zone branch" → Task 2.
- Spec "verify Wulf's stock tiers" → Task 3.
- Spec "scope: all non-crafter caravan-zone vendors" → Task 2 (the branch applies to all; the function self-gates crafters/no-shop).
- Spec testing → Task 1 (unit) + Task 4 (build/smoke).
- Spec memory update → Task 5.

**Placeholder check:** function + both unit tests + the MobIdle replacement block given in full. Task 3 is an explicit read-audit with a conditional data edit. No TBDs.

**Consistency:** `TickMobShopBaselineRestock` signature/behavior identical across Tasks 1-2 + tests; cadence math mirrors `TickMobShopRestock` exactly (tier-50 key, `RoundsPerDay/24`). RegisterShop-seeds-Current nuance handled in the test by depleting before asserting.

**TDD:** real unit tests via `RegisterTestItemSpec` + `RegisterShop` + `SetRoundCountForTest` (the established shops/mobs test pattern); MobIdle wiring + in-game behavior verified by build + boot smoke.
