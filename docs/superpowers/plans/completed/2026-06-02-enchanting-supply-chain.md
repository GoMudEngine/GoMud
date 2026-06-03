# Enchanting Supply Chain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Salvaging potions yields enchanting mats scaled to potion difficulty — both on-demand (players salvage spoiled potions) and passively (alchemy-vendor potion decay feeds a global reserve that enchanters draw from, neediest-gap-first).

**Architecture:** One shared difficulty-tiered mapping (potion → enchanting mats), reused by the player spoiled-potion salvage and an NPC global loop. The NPC loop mirrors the 5.4 globalized chest backfill: decay feeds a global mat reserve; enchanters draw via the (lifted, shared) neediest-gap allocator. Mostly Go in `internal/crafting` + `internal/shops` + the idle hook; config knobs for the band chances.

**Tech Stack:** Go (`testing` + `testify/assert`). Build `go build ./...`; test `go test ./internal/<pkg>/...`. Boot smoke per CLAUDE.md SOP.

**Spec:** `docs/superpowers/specs/2026-06-02-enchanting-supply-chain-design.md`

---

## Verified facts (confirmed before writing — trust over memory)

- **Player salvage spoiled branch** (`internal/actions/salvage.go:164` `salvageItem(actor Actor, uuid string, spoiledPotion bool, chance float64)`): lines 200-208 set `var recovered []crafting.RecipeIngredient`; the `spoiledPotion` branch hardcodes `recovered = []crafting.RecipeIngredient{{ItemTag:"binding-paste", Quantity: qty}}` (qty=1, or 2 if `chance>0.5`). The non-spoiled branch uses `crafting.GetRecipeByOutputItemId` + `RollSalvageReturns`/`RollSalvageReturnsFromSpec`. We replace ONLY the spoiled branch.
- **`crafting.RecipeIngredient`** (`crafting.go:20`): `{ItemTag string; Quantity int}`.
- **`crafting.GetRecipeByOutputItemId(itemId) *RecipeSpec`** (`crafting.go:110`). `RecipeSpec` has the recipe's skill gate field (YAML `skill_minimum`; confirm the Go field name — likely `SkillMinimum int`). Use it as the potion difficulty key (no potion has `rarity_tier`).
- **`items.FindSpecByComponentTag(tag) *ItemSpec`** (`itemspec.go:593`) — resolve a mat `ItemTag` → item id for `AddStockAtRound`.
- **`selectBackfillTransfers(si *shops.ShopInventory, pool map[int]int) map[int]int`** (`internal/forager/chest_backfill.go:17`) — the generic neediest-gap allocator (gap = MaxStock-Current, only items the shop stocks, pool-capped). Currently unexported in `forager`. We lift it to `internal/shops` and have forager call the lifted one.
- **`shops.TickOverstockDecay(si *ShopInventory, round uint64)`** (`overstock_decay.go:16`) — currently returns nothing; decrements `Current` for non-component overstock above RestockQty baseline. We make it return what it removed.
- **Enchanter identification:** `mobs.Mob.ShopCraftSupport string` (`mobs.go:142`, YAML `craft_support`) + `(*Mob).GetShopCraftSupport()` (`mobs.go:804`). Enchanter = `ShopCraftSupport == "enchanting"` (Vael 109 Thornwall). Alchemy vendor = `"alchemy"` (Voss 98 Thornwall, Ilsa 338 Stillwater — both crafters).
- **Hook tick reality:** the existing market block in `internal/hooks/MobIdle_HandleIdleMobs.go` (decay + chest backfill) is gated on `restocked` (set by `TickMobShopRestock` for non-crafters OR `TickMobCraft` for crafters). Voss/Ilsa are alchemy CRAFTERS → `TickMobCraft` fires → `restocked` → the market block runs → decay runs (this is where the FEED hooks). **Vael is a NON-crafter in caravan-served Thornwall** → `TickMobShopRestock` is skipped (caravan-zone guard) and `TickMobCraft` returns nil → `restocked` is never true → the market block never runs for Vael. So the enchanter **DRAW must be its own idle-tick block**, NOT inside the `restocked`-gated market block.
- **Vael's shop is registered** (he's a shopkeeper with a shop list); his mat entries exist (binding-paste 40028, chrysalis-shard 40027, mutation-catalyst 40029, chrysalis-setting 40030, chrysalis-core 40010, hive-fragment 40011) at Current 0 — so the draw has real gaps to fill.
- **Potion type:** `items.GetItemSpec(id).Type == items.Potion` identifies a potion.
- **RNG:** no confirmed `[0,1)` helper; `util.Rand(n)` returns `[0,n)`. Use `func() float64 { return float64(util.Rand(10000)) / 10000.0 }` as the roll source.

---

## File structure

| File | Responsibility |
|------|----------------|
| `internal/crafting/enchant_salvage_map.go` (+ test) | tiered potion→mat mapping (`EnchantSalvageYield` + testable core) |
| `internal/configs/config.balance.go` (+ validation + config.yaml) | band thresholds + per-mat chance knobs |
| `internal/actions/salvage.go` | spoiled branch calls the mapping |
| `internal/shops/stock_transfers.go` (+ test) | lifted `SelectStockTransfers` (was forager `selectBackfillTransfers`) |
| `internal/forager/chest_backfill.go` (+ test) | call `shops.SelectStockTransfers` |
| `internal/shops/enchant_reserve.go` (+ test) | global mat reserve (`AddToReserve`/`ReservePool`/`DrainReserve`) |
| `internal/shops/overstock_decay.go` (+ test) | `TickOverstockDecay` returns `[]DecayedUnit` |
| `internal/hooks/MobIdle_HandleIdleMobs.go` | feed decayed potions → reserve; enchanter draw block |
| context.md (crafting, shops) + roadmap/memory | docs |

---

## Task 1: Tiered potion → enchanting-mat mapping

**Files:** Create `internal/crafting/enchant_salvage_map.go` (+ test); Modify config (Step 1).

- [ ] **Step 1: Add config knobs**

In `internal/configs/config.balance.go` (near the other shop/economy knobs), add:
```go
	EnchantSalvageBand2Min      ConfigInt `yaml:"EnchantSalvageBand2Min,omitempty"`      // potion skill_min >= this → band 2 (default 10)
	EnchantSalvageBand3Min      ConfigInt `yaml:"EnchantSalvageBand3Min,omitempty"`      // band 3 (default 18)
	EnchantSalvageBand4Min      ConfigInt `yaml:"EnchantSalvageBand4Min,omitempty"`      // band 4 (default 28)
	EnchantSalvageBand2SettingPct  ConfigInt `yaml:"EnchantSalvageBand2SettingPct,omitempty"`  // default 25
	EnchantSalvageBand3SettingPct  ConfigInt `yaml:"EnchantSalvageBand3SettingPct,omitempty"`  // default 35
	EnchantSalvageBand3CatalystPct ConfigInt `yaml:"EnchantSalvageBand3CatalystPct,omitempty"` // default 12
	EnchantSalvageBand4CatalystPct ConfigInt `yaml:"EnchantSalvageBand4CatalystPct,omitempty"` // default 40
	EnchantSalvageBand4SettingPct  ConfigInt `yaml:"EnchantSalvageBand4SettingPct,omitempty"`  // default 30
	EnchantSalvageBand4CorePct     ConfigInt `yaml:"EnchantSalvageBand4CorePct,omitempty"`     // default 8
```
In the balance defaults validation file (the one that defaults `ShopOverstockDecayRounds` — `internal/configs/config.balance.shops.go`), add the `if b.X <= 0 { b.X = default }` lines for each (10/18/28/25/35/12/40/30/8). Add the same keys+defaults to the `Balance:` block of `_datafiles/config.yaml`.

- [ ] **Step 2: Write the failing test**

`internal/crafting/enchant_salvage_map_test.go`:
```go
package crafting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func bandsFixture() EnchantSalvageBands {
	return EnchantSalvageBands{
		Band2Min: 10, Band3Min: 18, Band4Min: 28,
		Band2SettingPct: 25, Band3SettingPct: 35, Band3CatalystPct: 12,
		Band4CatalystPct: 40, Band4SettingPct: 30, Band4CorePct: 8,
	}
}

func tags(out []RecipeIngredient) map[string]int {
	m := map[string]int{}
	for _, r := range out {
		m[r.ItemTag] += r.Quantity
	}
	return m
}

func TestEnchantSalvage_Band1_BindingPasteOnly(t *testing.T) {
	out := EnchantSalvageYieldWith(0, func() float64 { return 0.0 }, 0, bandsFixture())
	assert.Equal(t, 1, tags(out)["binding-paste"])
	assert.Len(t, out, 1)
}

func TestEnchantSalvage_Band1_QtyBonus(t *testing.T) {
	out := EnchantSalvageYieldWith(5, func() float64 { return 0.0 }, 1, bandsFixture())
	assert.Equal(t, 2, tags(out)["binding-paste"])
}

func TestEnchantSalvage_Band2_RollsSetting(t *testing.T) {
	// roll 0.0 < 25% → setting drops; binding-paste floor present
	out := EnchantSalvageYieldWith(12, func() float64 { return 0.0 }, 0, bandsFixture())
	tg := tags(out)
	assert.GreaterOrEqual(t, tg["binding-paste"], 1)
	assert.Equal(t, 1, tg["chrysalis-setting"])
}

func TestEnchantSalvage_Band2_MissSetting(t *testing.T) {
	// roll 0.99 > 25% → no setting, just binding-paste
	out := EnchantSalvageYieldWith(12, func() float64 { return 0.99 }, 0, bandsFixture())
	tg := tags(out)
	assert.GreaterOrEqual(t, tg["binding-paste"], 1)
	assert.Equal(t, 0, tg["chrysalis-setting"])
}

func TestEnchantSalvage_Band4_RollsRareMats(t *testing.T) {
	out := EnchantSalvageYieldWith(40, func() float64 { return 0.0 }, 0, bandsFixture())
	tg := tags(out)
	assert.Equal(t, 1, tg["mutation-catalyst"])
	assert.Equal(t, 1, tg["chrysalis-setting"])
	assert.Equal(t, 1, tg["chrysalis-core"])
}

func TestEnchantSalvage_Band4_AllMiss_FloorBindingPaste(t *testing.T) {
	out := EnchantSalvageYieldWith(40, func() float64 { return 0.99 }, 0, bandsFixture())
	assert.Equal(t, 1, tags(out)["binding-paste"], "band 4 with all rolls missing falls back to binding-paste")
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/crafting/ -run TestEnchantSalvage -v`
Expected: FAIL (undefined).

- [ ] **Step 4: Implement the mapping**

`internal/crafting/enchant_salvage_map.go`:
```go
package crafting

import "github.com/GoMudEngine/GoMud/internal/configs"

// EnchantSalvageBands holds the band thresholds (by potion alchemy-recipe
// skill_minimum) and per-mat drop chances for tiered potion salvage.
type EnchantSalvageBands struct {
	Band2Min, Band3Min, Band4Min                           int
	Band2SettingPct                                        int
	Band3SettingPct, Band3CatalystPct                      int
	Band4CatalystPct, Band4SettingPct, Band4CorePct        int
}

// EnchantSalvageYieldWith is the pure, testable core: maps a potion's recipe
// skill_minimum to enchanting-mat returns, rolling per-band chances via the
// injected roll() ([0,1)). qtyBonus (0+) bumps the binding-paste floor (e.g. for
// high salvage skill). Bands 1-3 always include a binding-paste floor; band 4
// rolls rare mats and falls back to binding-paste if all rolls miss.
func EnchantSalvageYieldWith(skillMin int, roll func() float64, qtyBonus int, b EnchantSalvageBands) []RecipeIngredient {
	hit := func(pct int) bool { return roll()*100.0 < float64(pct) }
	var out []RecipeIngredient
	switch {
	case skillMin >= b.Band4Min:
		if hit(b.Band4CatalystPct) {
			out = append(out, RecipeIngredient{ItemTag: "mutation-catalyst", Quantity: 1})
		}
		if hit(b.Band4SettingPct) {
			out = append(out, RecipeIngredient{ItemTag: "chrysalis-setting", Quantity: 1})
		}
		if hit(b.Band4CorePct) {
			out = append(out, RecipeIngredient{ItemTag: "chrysalis-core", Quantity: 1})
		}
	case skillMin >= b.Band3Min:
		out = append(out, RecipeIngredient{ItemTag: "binding-paste", Quantity: 1 + qtyBonus})
		if hit(b.Band3SettingPct) {
			out = append(out, RecipeIngredient{ItemTag: "chrysalis-setting", Quantity: 1})
		}
		if hit(b.Band3CatalystPct) {
			out = append(out, RecipeIngredient{ItemTag: "mutation-catalyst", Quantity: 1})
		}
	case skillMin >= b.Band2Min:
		out = append(out, RecipeIngredient{ItemTag: "binding-paste", Quantity: 1 + qtyBonus})
		if hit(b.Band2SettingPct) {
			out = append(out, RecipeIngredient{ItemTag: "chrysalis-setting", Quantity: 1})
		}
	default:
		out = append(out, RecipeIngredient{ItemTag: "binding-paste", Quantity: 1 + qtyBonus})
	}
	if len(out) == 0 {
		out = append(out, RecipeIngredient{ItemTag: "binding-paste", Quantity: 1})
	}
	return out
}

// EnchantSalvageYield resolves a potion item id to its recipe skill_minimum and
// applies the configured bands. Unknown/no-recipe potions fall to band 1.
func EnchantSalvageYield(potionItemId int, roll func() float64, qtyBonus int) []RecipeIngredient {
	skillMin := 0
	if r := GetRecipeByOutputItemId(potionItemId); r != nil {
		skillMin = r.SkillMinimum // confirm field name on RecipeSpec
	}
	c := configs.GetBalanceConfig()
	bands := EnchantSalvageBands{
		Band2Min:         int(c.EnchantSalvageBand2Min),
		Band3Min:         int(c.EnchantSalvageBand3Min),
		Band4Min:         int(c.EnchantSalvageBand4Min),
		Band2SettingPct:  int(c.EnchantSalvageBand2SettingPct),
		Band3SettingPct:  int(c.EnchantSalvageBand3SettingPct),
		Band3CatalystPct: int(c.EnchantSalvageBand3CatalystPct),
		Band4CatalystPct: int(c.EnchantSalvageBand4CatalystPct),
		Band4SettingPct:  int(c.EnchantSalvageBand4SettingPct),
		Band4CorePct:     int(c.EnchantSalvageBand4CorePct),
	}
	return EnchantSalvageYieldWith(skillMin, roll, qtyBonus, bands)
}
```
Confirm `RecipeSpec`'s skill field name (read `crafting.go` — it's the field for YAML `skill_minimum`); adjust `r.SkillMinimum` if different.

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/crafting/ -run TestEnchantSalvage -v && go build ./...`
Expected: PASS, clean.

- [ ] **Step 6: Commit**

```bash
git add internal/crafting/enchant_salvage_map.go internal/crafting/enchant_salvage_map_test.go internal/configs/config.balance.go internal/configs/config.balance.shops.go _datafiles/config.yaml
git commit -m "feat(enchanting): tiered potion->enchanting-mat salvage mapping + config knobs"
```

---

## Task 2: Player salvage uses the mapping

**Files:** Modify `internal/actions/salvage.go`.

- [ ] **Step 1: Replace the spoiled-potion branch**

In `internal/actions/salvage.go` `salvageItem`, replace the spoiled block (currently lines ~201-208):
```go
	if spoiledPotion {
		qtyBonus := 0
		if chance > 0.5 {
			qtyBonus = 1 // preserve the existing salvage-skill bump
		}
		roll := func() float64 { return float64(util.Rand(10000)) / 10000.0 }
		recovered = crafting.EnchantSalvageYield(itemId, roll, qtyBonus)
	} else {
		// ... unchanged recipe / SalvageReturns branch ...
	}
```
Ensure `internal/util` is imported (likely already). `crafting` is already imported. Leave the non-spoiled branch untouched.

- [ ] **Step 2: Build + test**

Run: `go build ./... && go test ./internal/actions/`
Expected: clean + PASS. (Existing salvage tests should still pass — band-1 potions still yield binding-paste ×1–2. If a test asserted the exact spoiled output, it still holds for band-1 inputs; if it used a higher-skill potion, update it to the new tiered expectation.)

- [ ] **Step 3: Commit**

```bash
git add internal/actions/salvage.go
git commit -m "feat(enchanting): player spoiled-potion salvage uses tiered mapping"
```

---

## Task 3: Lift the neediest-gap allocator to shops

**Files:** Create `internal/shops/stock_transfers.go` (+ test); Modify `internal/forager/chest_backfill.go`.

- [ ] **Step 1: Write the test for the lifted allocator**

`internal/shops/stock_transfers_test.go`:
```go
package shops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectStockTransfers_NeediestGapFirst(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 10, MaxStock: 10, Current: 9}, // gap 1
		{ItemId: 20, MaxStock: 10, Current: 2}, // gap 8 (neediest)
	}}
	pool := map[int]int{10: 5, 20: 5}
	got := SelectStockTransfers(si, pool)
	assert.Equal(t, 5, got[20]) // neediest filled first, pool-capped at 5
	assert.Equal(t, 1, got[10])
}

func TestSelectStockTransfers_OnlyStockedAndCapped(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{{ItemId: 10, MaxStock: 10, Current: 10}}}
	assert.Empty(t, SelectStockTransfers(si, map[int]int{10: 5})) // no gap
	si2 := &ShopInventory{Stock: []StockEntry{{ItemId: 10, MaxStock: 10, Current: 5}}}
	assert.Empty(t, SelectStockTransfers(si2, map[int]int{99: 5})) // not stocked
}
```

- [ ] **Step 2: Run to verify fail**

Run: `go test ./internal/shops/ -run TestSelectStockTransfers -v`
Expected: FAIL (undefined).

- [ ] **Step 3: Create the lifted function**

`internal/shops/stock_transfers.go` — move the body of forager's `selectBackfillTransfers` here, exported:
```go
package shops

import "sort"

// SelectStockTransfers decides how many of each item to pull from `pool` into
// shopInv, neediest stock-gap first (gap = MaxStock-Current), capped by each
// entry's gap and by pool availability. Only items shopInv already stocks (with
// a gap) are eligible. Pure. (Lifted from forager.selectBackfillTransfers so the
// chest backfill and the enchanting-reserve draw share one allocator.)
func SelectStockTransfers(shopInv *ShopInventory, pool map[int]int) map[int]int {
	type gap struct {
		itemId int
		gap    int
	}
	var gaps []gap
	for i := range shopInv.Stock {
		e := &shopInv.Stock[i]
		g := e.MaxStock - e.Current
		if g > 0 && pool[e.ItemId] > 0 {
			gaps = append(gaps, gap{e.ItemId, g})
		}
	}
	sort.Slice(gaps, func(a, b int) bool {
		if gaps[a].gap != gaps[b].gap {
			return gaps[a].gap > gaps[b].gap
		}
		return gaps[a].itemId < gaps[b].itemId
	})
	remaining := map[int]int{}
	for id, n := range pool {
		remaining[id] = n
	}
	out := map[int]int{}
	for _, g := range gaps {
		take := g.gap
		if take > remaining[g.itemId] {
			take = remaining[g.itemId]
		}
		if take > 0 {
			out[g.itemId] = take
			remaining[g.itemId] -= take
		}
	}
	return out
}
```

- [ ] **Step 4: Point forager at the lifted version**

In `internal/forager/chest_backfill.go`: delete the local `selectBackfillTransfers` func and replace its one call site in `BackfillVendorFromChests` with `shops.SelectStockTransfers(shopInv, pool)`. Update `internal/forager/chest_backfill_test.go`'s `TestSelectBackfillTransfers_*` tests to call `shops.SelectStockTransfers` (or, since they now live in shops, delete the forager copies — the shops test covers them; keep whichever avoids losing coverage. Simplest: change the forager tests' calls to `shops.SelectStockTransfers`).

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/shops/ ./internal/forager/ && go build ./...`
Expected: PASS (forager backfill still works via the lifted allocator), clean.

- [ ] **Step 6: Commit**

```bash
git add internal/shops/stock_transfers.go internal/shops/stock_transfers_test.go internal/forager/chest_backfill.go internal/forager/chest_backfill_test.go
git commit -m "refactor(shops): lift neediest-gap allocator to shops.SelectStockTransfers"
```

---

## Task 4: Global enchanting-mat reserve

**Files:** Create `internal/shops/enchant_reserve.go` (+ test).

- [ ] **Step 1: Write the failing test**

`internal/shops/enchant_reserve_test.go`:
```go
package shops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnchantReserve_AddDrainPool(t *testing.T) {
	ResetEnchantReserveForTest()
	AddToReserve(40028, 3)
	AddToReserve(40028, 2)
	AddToReserve(40030, 1)
	assert.Equal(t, map[int]int{40028: 5, 40030: 1}, ReservePool())

	DrainReserve(40028, 4)
	assert.Equal(t, 1, ReservePool()[40028])

	// Pool() is a copy — mutating it doesn't affect the reserve.
	p := ReservePool()
	p[40028] = 999
	assert.Equal(t, 1, ReservePool()[40028])

	AddToReserve(40028, 0)   // no-op
	AddToReserve(0, 5)       // ignored (zero id)
	assert.Equal(t, 1, ReservePool()[40028])
}
```

- [ ] **Step 2: Run to verify fail**

Run: `go test ./internal/shops/ -run TestEnchantReserve -v`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement the reserve**

`internal/shops/enchant_reserve.go`:
```go
package shops

import "sync"

// enchantReserve is the global pool of enchanting mats produced by alchemy-
// vendor potion decay (the analog of the aggregated forager chests, but
// virtual). In-memory v1 — it refills from ongoing decay, so a restart just
// re-accumulates. Enchanters draw from it neediest-gap-first on their idle tick.
var (
	enchantReserveMu sync.Mutex
	enchantReserve   = map[int]int{}
)

// AddToReserve adds qty of matItemId to the global reserve. No-op for zero id/qty.
func AddToReserve(matItemId, qty int) {
	if matItemId <= 0 || qty <= 0 {
		return
	}
	enchantReserveMu.Lock()
	defer enchantReserveMu.Unlock()
	enchantReserve[matItemId] += qty
}

// ReservePool returns a copy of the current reserve (safe to mutate / iterate).
func ReservePool() map[int]int {
	enchantReserveMu.Lock()
	defer enchantReserveMu.Unlock()
	out := make(map[int]int, len(enchantReserve))
	for id, n := range enchantReserve {
		out[id] = n
	}
	return out
}

// DrainReserve removes up to qty of matItemId. Safe if qty exceeds the balance.
func DrainReserve(matItemId, qty int) {
	if matItemId <= 0 || qty <= 0 {
		return
	}
	enchantReserveMu.Lock()
	defer enchantReserveMu.Unlock()
	if enchantReserve[matItemId] <= qty {
		delete(enchantReserve, matItemId)
	} else {
		enchantReserve[matItemId] -= qty
	}
}

// ResetEnchantReserveForTest clears the reserve (test helper).
func ResetEnchantReserveForTest() {
	enchantReserveMu.Lock()
	defer enchantReserveMu.Unlock()
	enchantReserve = map[int]int{}
}
```

- [ ] **Step 4: Run + build + commit**

Run: `go test ./internal/shops/ -run TestEnchantReserve -v && go build ./...`
```bash
git add internal/shops/enchant_reserve.go internal/shops/enchant_reserve_test.go
git commit -m "feat(enchanting): global enchanting-mat reserve store"
```

---

## Task 5: TickOverstockDecay reports what it removed

**Files:** Modify `internal/shops/overstock_decay.go` (+ test).

- [ ] **Step 1: Write the failing test**

Add to `internal/shops/overstock_decay_test.go`:
```go
func TestOverstockDecay_ReturnsDecayedUnits(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 100, RestockQty: 2, MaxStock: 10, Current: 6, LastGrewRound: 0},
	}}
	got := TickOverstockDecayWith(si, 100000, func(int) bool { return false }, 21600, 1)
	assert.Equal(t, []DecayedUnit{{ItemId: 100, Qty: 1}}, got)
	assert.Equal(t, 5, si.Stock[0].Current)
}
```

- [ ] **Step 2: Run to verify fail**

Run: `go test ./internal/shops/ -run TestOverstockDecay_ReturnsDecayedUnits -v`
Expected: FAIL (TickOverstockDecayWith returns nothing / DecayedUnit undefined).

- [ ] **Step 3: Add the return type + return values**

In `internal/shops/overstock_decay.go`:
- Add `type DecayedUnit struct{ ItemId int; Qty int }`.
- Change `TickOverstockDecayWith(...)` to `... []DecayedUnit`; inside the loop, where it does `e.Current -= drop`, append `DecayedUnit{ItemId: e.ItemId, Qty: drop}` to a result slice; return it.
- Change `TickOverstockDecay(si, round)` to `... []DecayedUnit` and `return TickOverstockDecayWith(...)`.

- [ ] **Step 4: Run + build**

Run: `go test ./internal/shops/ -run TestOverstockDecay -v && go build ./...`
Expected: PASS (existing decay tests still pass — they ignore the new return). Note: the existing MobIdle caller (`shops.TickOverstockDecay(...)`) now returns a value it didn't before — Go allows ignoring it, so the build stays clean until Task 6 wires it.

- [ ] **Step 5: Commit**

```bash
git add internal/shops/overstock_decay.go internal/shops/overstock_decay_test.go
git commit -m "feat(enchanting): TickOverstockDecay returns the units it removed"
```

---

## Task 6: Wire the feed + the enchanter draw (MobIdle)

**Files:** Modify `internal/hooks/MobIdle_HandleIdleMobs.go`. No unit test (idle-hook integration; verified by boot + in-game smoke).

- [ ] **Step 1: Feed — capture decayed potions into the reserve**

In the existing `restocked`-gated market block (where `shops.TickOverstockDecay` is called), capture the return and feed potions:
```go
		decayed := shops.TickOverstockDecay(shopInv, util.GetRoundCount())
		for _, du := range decayed {
			if spec := items.GetItemSpec(du.ItemId); spec != nil && spec.Type == items.Potion {
				roll := func() float64 { return float64(util.Rand(10000)) / 10000.0 }
				for i := 0; i < du.Qty; i++ {
					for _, mat := range crafting.EnchantSalvageYield(du.ItemId, roll, 0) {
						if ms := items.FindSpecByComponentTag(mat.ItemTag); ms != nil {
							shops.AddToReserve(ms.ItemId, mat.Quantity)
						}
					}
				}
			}
		}
		forager.BackfillVendorFromChests(mob, shopInv)   // existing
		// ... existing SaveShop ...
```
Add imports `internal/crafting` and `internal/items` if not present. (This runs for alchemy crafters Voss/Ilsa, whose `TickMobCraft` sets `restocked`.)

- [ ] **Step 2: Draw — enchanter pulls from the reserve (its own ungated block)**

After the market block, add a SEPARATE block (NOT gated on `restocked`, because Vael is a non-crafter in a caravan-served zone and never sets `restocked`):
```go
	// 5.4-followup: enchanters draw enchanting mats from the global reserve
	// (fed by alchemy-vendor potion decay), neediest stock-gap first. Ungated
	// by `restocked` — Vael is a non-crafter in a caravan-served zone, so its
	// restock tick is skipped; this fires on the enchanter's idle tick.
	if !isCharmed && mob.GetShopCraftSupport() == "enchanting" {
		if eShop := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId); eShop != nil {
			transfers := shops.SelectStockTransfers(eShop, shops.ReservePool())
			mutated := false
			for matId, qty := range transfers {
				shops.DrainReserve(matId, qty)
				eShop.AddStockAtRound(matId, qty, util.GetRoundCount())
				mutated = true
			}
			if mutated {
				if err := shops.SaveShop(mob.Zone, int(mob.MobId), mob.HomeRoomId); err != nil {
					mudlog.Error("MobIdle.enchantDraw", "error", err)
				}
			}
		}
	}
```
Place it where `mob`, `isCharmed`, and the imports are in scope (alongside the other per-mob blocks).

- [ ] **Step 3: Build + boot smoke**

Run: `go build ./...`
Then (SOP): `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*` and boot; confirm `InputWorker ... Started`, no panic. (Controller may run this.)

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/MobIdle_HandleIdleMobs.go
git commit -m "feat(enchanting): feed decayed potions to reserve + enchanter draws from it"
```

---

## Task 7: Docs, full verification, roadmap

**Files:** context.md (crafting, shops); `MOB_ALIVENESS_ROADMAP.md` / memory.

- [ ] **Step 1: Docs**

- `internal/crafting/context.md`: document `EnchantSalvageYield` (tiered potion→mat mapping; band by recipe skill_minimum) and that it's shared by player salvage + the NPC decay loop.
- `internal/shops/context.md`: document the global enchant reserve (`AddToReserve`/`ReservePool`/`DrainReserve`), `SelectStockTransfers` (the lifted shared allocator), and `TickOverstockDecay`'s new `[]DecayedUnit` return.

- [ ] **Step 2: Full build + test sweep**

Run: `go build ./... && go test ./...`
Expected: all packages PASS. Classify any failure as enchanting-chunk-related vs pre-existing.

- [ ] **Step 3: Boot smoke (clean instances)**

`rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*` then boot; `InputWorker ... Started`, no panic.

- [ ] **Step 4: Roadmap/memory**

Update [[project_store_restock_considered_fix]] / `MOB_ALIVENESS_ROADMAP.md`: enchanting half addressed (tiered potion-salvage + global decayed-potion reserve → enchanters); note remaining: general-store restock; the Vael caravan-routing gap is now mitigated by the salvage/reserve supply (rare chrysalis-core/shard still also via Old Edrin loot).

- [ ] **Step 5: Commit**

```bash
git add internal/crafting/context.md internal/shops/context.md MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(enchanting): context + roadmap for potion-salvage enchanting supply"
```

---

## In-game smoke checklist (deferred to user, per precedent)

- [ ] Salvage a **low-tier** spoiled potion (healing salve / stamina tonic) → binding paste ×1–2 (unchanged).
- [ ] Salvage a **high-tier chrysalis-themed** spoiled potion (mutagen brew skill 35 / chrysalis catalyst skill 40) → see chrysalis-setting / mutation-catalyst / chrysalis-core rolls.
- [ ] Over time (let alchemy vendors craft+overstock+decay), confirm Vael's binding-paste / chrysalis-setting stock ticks up from the global reserve (Voss AND Ilsa decay feed it; cross-zone Stillwater→Thornwall works).
- [ ] Confirm the alchemy shops are NOT starved of potions (in-demand potions stay stocked; only abandoned surplus decays).
- [ ] No `looks a little confused` / no crash.

---

## Self-review notes (flagged for the implementer)

- **`RecipeSpec` skill field name** (Task 1 Step 4): confirm it's `SkillMinimum` (YAML `skill_minimum`) when wiring `EnchantSalvageYield`; adjust if the Go field differs.
- **Enchanter draw is ungated** (Task 6 Step 2) — this is deliberate: Vael's restock tick is skipped (non-crafter, caravan-served zone), so the draw can't live in the `restocked`-gated market block. It's cheap (a map alloc + SelectStockTransfers) and self-limits (only fills real gaps from a non-empty reserve).
- **Feed runs for alchemy crafters** (Voss/Ilsa) via the `restocked` market block (their `TickMobCraft` fires). A non-crafter alchemy vendor in a caravan zone wouldn't run decay — none exists today; note if one is added later.
- **Reserve is in-memory** — a restart drops it; it re-accumulates from ongoing decay. Persistence is deferred (out of scope).
- **`util.Rand(10000)/10000.0`** is the [0,1) roll source (no dedicated helper confirmed). If a `[0,1)` util exists, prefer it.
