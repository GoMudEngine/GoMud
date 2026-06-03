# Mob Aliveness 5.3 — Equipment-Aware Shopping Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give gold-using humanoid mobs a standing "upgrade my gear" drive: they
score every item in stock at zone shops against their current loadout via the
2.2 `itemvalue` primitive, travel to the vendor, sell junk to fund it if short,
buy the best affordable upgrade, and equip it.

**Architecture:** One new planner-local shop-stock evaluator (`scanZoneUpgrades`)
composes pure `itemvalue.ItemValueDelta` scoring with `shops` dynamic pricing.
A new perpetual `upgrade-gear` goal type (catalog) governs activation via a
cheap context-score heuristic with a non-zero floor (so 4.6 pruning never
abandons it). A new `upgrade-gear` planner runs the buy/sell/equip state
machine. Two balance knobs and two archetype wirings. No behavior-tree edits —
`try_goal_planner` is already in every non-boss tree.

**Tech Stack:** Go. Packages: `internal/planners`, `internal/goals/catalog`,
`internal/configs`, `internal/shops`, `internal/itemvalue`. YAML archetype data.

**Spec:** `docs/superpowers/specs/2026-06-01-mob-aliveness-5.3-equipment-aware-shopping-design.md`

---

## Key verified facts (read before starting)

- **Price direction (do not get this backwards):** the mob *buys from* the shop,
  so the price it pays is `shops.CalcSellPrice(baseValue, current, restockQty,
  cfg)` ("what the NPC charges a player to buy an item"). `shops.CalcBuyPrice`
  is the *opposite* direction (what the shop pays when buying from the player) —
  do NOT use it here.
- `shops.AllShops() []*shops.ShopInventory`; each has `.Zone string`,
  `.RoomId int`, `.Gold int`, `.Stock []shops.StockEntry`. `StockEntry` has
  `ItemId, RestockQty, MaxStock, Current int`.
- `shops.PricingConfigFromBalance() shops.PricingConfig`.
- `itemvalue.ProfileFor(mob.Archetype, mob.BehaviorArchetype) itemvalue.WeightProfile`
  (exact call used in `gearup.go`, `crafter.go`, `mob_equip_best_floor_item.go`).
- `itemvalue.ItemValueDelta(char *characters.Character, profile, candidate
  items.Item) itemvalue.SwapDelta`; `SwapDelta.Score float64`, `.Slot SlotName`
  ("" if not equippable).
- `items.GetItemSpec(id int) *items.ItemSpec` (nil if unknown); `items.New(id)
  items.Item`; `item.Name() string`; `spec.Value int`.
- `mob.Character` is a value of type `characters.Character`; pass
  `&mob.Character` to `ItemValueDelta`. Fields used: `.Zone`, `.RoomId`,
  `.Gold`, `.Items`, `.Aggro` (nil = out of combat), `.MiscData`.
- Planner contract (`internal/planners/planners.go`): `PlanFn func(mob
  *mobs.Mob, goal *goals.Goal) PlanResult`; `PlanResult{Command string, Status
  BTreeStatus}`; statuses `StatusFailure`, `StatusSuccess`, `StatusRunning`.
  Register in `init()` via `RegisterPlanner(type, fn)`. `LookupPlanner(type)`
  for tests.
- Planner MiscData keys MUST start with `PlanKeyPrefix` (`"plan:"`) so
  `ClearPlanState` wipes them on goal switch. Helpers exist: `mobMiscIntOr`,
  `mobSetMisc`, `mobHasSellableItems`, `mobInVendorRoom`, `findShopInZoneBuying`.
- Goal catalog pattern (`internal/goals/catalog/survival.go`): `init()` calls
  `goals.RegisterGoalType(type, goals.GoalTypeMeta{Predicate, ContextScore,
  AllowMultiple, DedupKey, Params})`. `Predicate func(*goals.Goal,*mobs.Mob)
  bool`; `ContextScore func(*goals.Goal,*mobs.Mob) float64`. Catalog-local
  `paramIntOr(g, key, def)` helper exists in the package.
- **Unit-test reality:** in `go test` context, data files are NOT loaded —
  `items.GetItemSpec` returns nil and `shops.AllShops()` is empty. Existing
  planner tests therefore cover pure helpers + branch shapes only; live-data
  scoring is validated by in-game smoke. This plan does the same. Do NOT write
  unit tests that assume loaded item/shop data.
- Config knob pattern: declare `ConfigInt`/`ConfigFloat` field in
  `internal/configs/config.balance.go`; default it in
  `internal/configs/config.balance.misc.go` with a `if b.X <= 0 { b.X = ... }`
  block; add a line to `_datafiles/config.yaml`.

---

## File structure

- **Create:** `internal/planners/shop_upgrade.go` — evaluator (`scanZoneUpgrades`,
  `stockRestockNorm`) + tests `shop_upgrade_test.go`
- **Create:** `internal/planners/upgrade_gear.go` — planner + tests
  `upgrade_gear_test.go`
- **Create:** `internal/goals/catalog/upgrade_gear.go` — goal type + tests
  `upgrade_gear_test.go`
- **Modify:** `internal/configs/config.balance.go` — 2 knob fields
- **Modify:** `internal/configs/config.balance.misc.go` — 2 defaults
- **Modify:** `_datafiles/config.yaml` — 2 knob lines
- **Modify:** `_datafiles/world/dogmud/behaviors/archetypes/thief.yaml`
- **Modify:** `_datafiles/world/dogmud/behaviors/archetypes/guard_captain.yaml`
- **Modify:** `internal/planners/context.md`,
  `internal/goals/catalog/context.md` — doc updates
- **Modify:** `MOB_ALIVENESS_ROADMAP.md` — flip 5.3 status; note 5.1d

---

## Task 1: Config knobs

**Files:**
- Modify: `internal/configs/config.balance.go:596-602` (add after the bounty
  hunter block)
- Modify: `internal/configs/config.balance.misc.go:384` (add after the bounty
  hunter defaults)
- Modify: `_datafiles/config.yaml:711` (after `ShopGoldReserveRatio`)
- Test: `internal/configs/config.balance_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/configs/config.balance_test.go`:

```go
func TestBalance_MobUpgradeDefaults(t *testing.T) {
	b := BalanceConfig{}
	b.Validate()
	if int(b.MobUpgradeGoldReserve) != 50 {
		t.Fatalf("MobUpgradeGoldReserve = %d, want 50", int(b.MobUpgradeGoldReserve))
	}
	if float64(b.MobUpgradeMinDelta) != 1.0 {
		t.Fatalf("MobUpgradeMinDelta = %v, want 1.0", float64(b.MobUpgradeMinDelta))
	}
}
```

(If the existing tests call a differently-named validation entry point, match
it — grep `func (b *BalanceConfig)` in `config.balance.misc.go` for the method
the other tests use; the bounty defaults are applied by the same method tested
at `config.balance_test.go:29`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/configs/ -run TestBalance_MobUpgradeDefaults -count=1`
Expected: FAIL — `b.MobUpgradeGoldReserve` undefined (compile error).

- [ ] **Step 3: Add the struct fields**

In `internal/configs/config.balance.go`, immediately after line 602
(`BountyHunterGearGoldDivisor ...`):

```go

	// ── EQUIPMENT-AWARE SHOPPING (5.3) ──────────────────────────────────
	MobUpgradeGoldReserve ConfigInt   `yaml:"MobUpgradeGoldReserve"` // Gold a shopping mob keeps in reserve, won't spend below (default 50)
	MobUpgradeMinDelta    ConfigFloat `yaml:"MobUpgradeMinDelta"`    // Minimum itemvalue swap delta worth buying (default 1.0)
```

- [ ] **Step 4: Add the defaults**

In `internal/configs/config.balance.misc.go`, immediately after the bounty
hunter defaults block (after line 384's `BountyHunterMaxStatpool` default, and
its closing brace — place after the full bounty hunter group ends):

```go

	// ── EQUIPMENT-AWARE SHOPPING (5.3) ──────────────────────────────────
	if b.MobUpgradeGoldReserve <= 0 {
		b.MobUpgradeGoldReserve = 50
	}
	if b.MobUpgradeMinDelta <= 0 {
		b.MobUpgradeMinDelta = 1.0
	}
```

- [ ] **Step 5: Add the config.yaml lines**

In `_datafiles/config.yaml`, after the `ShopGoldReserveRatio: 0.50` line (711):

```yaml
  MobUpgradeGoldReserve: 50     # Gold a shopping mob keeps in reserve (5.3)
  MobUpgradeMinDelta: 1.0       # Min itemvalue delta worth buying (5.3)
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/configs/ -run TestBalance_MobUpgradeDefaults -count=1`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.misc.go internal/configs/config.balance_test.go _datafiles/config.yaml
git commit -m "feat(config): MobUpgradeGoldReserve + MobUpgradeMinDelta knobs (5.3)"
```

---

## Task 2: Shop-stock evaluator

**Files:**
- Create: `internal/planners/shop_upgrade.go`
- Test: `internal/planners/shop_upgrade_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/planners/shop_upgrade_test.go`:

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/itemvalue"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

func TestStockRestockNorm(t *testing.T) {
	cases := []struct {
		name string
		e    shops.StockEntry
		want int
	}{
		{"explicit restock qty", shops.StockEntry{RestockQty: 5, MaxStock: 20}, 5},
		{"crafted-only uses maxstock/2", shops.StockEntry{RestockQty: 0, MaxStock: 20}, 10},
		{"maxstock/2 floors at 1", shops.StockEntry{RestockQty: 0, MaxStock: 1}, 1},
		{"no info floors at 1", shops.StockEntry{}, 1},
	}
	for _, c := range cases {
		if got := stockRestockNorm(c.e); got != c.want {
			t.Errorf("%s: stockRestockNorm = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestScanZoneUpgrades_NilMob(t *testing.T) {
	if _, ok := scanZoneUpgrades(nil, itemvalue.WeightProfile{}, 100, true, 1.0); ok {
		t.Errorf("expected ok=false for nil mob")
	}
}

func TestScanZoneUpgrades_NoShopsLoaded(t *testing.T) {
	// In unit-test context shops.AllShops() is empty → no candidate.
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	if _, ok := scanZoneUpgrades(mob, itemvalue.WeightProfile{}, 1000, true, 1.0); ok {
		t.Errorf("expected ok=false when no shops are loaded")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/planners/ -run 'TestStockRestockNorm|TestScanZoneUpgrades' -count=1`
Expected: FAIL — `stockRestockNorm` / `scanZoneUpgrades` undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/planners/shop_upgrade.go`:

```go
package planners

import (
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/itemvalue"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

// upgradeCandidate is the best in-stock upgrade the evaluator found.
type upgradeCandidate struct {
	ShopRoom int
	ItemId   int
	ItemName string  // canonical item name, used for the `buy <name>` command
	Price    int     // what the mob pays (CalcSellPrice)
	Delta    float64 // itemvalue swap delta (gain)
}

// stockRestockNorm mirrors the normalizer used across the shops package
// (buyrules.go / craftdecision.go): RestockQty when > 0, else MaxStock/2
// (min 1), else 1. Normalizes the scarcity-pricing curve.
func stockRestockNorm(e shops.StockEntry) int {
	if e.RestockQty > 0 {
		return e.RestockQty
	}
	if e.MaxStock > 0 {
		n := e.MaxStock / 2
		if n < 1 {
			n = 1
		}
		return n
	}
	return 1
}

// scanZoneUpgrades walks every in-stock entry across all shops in the mob's
// zone, scores each as a swap for this mob via itemvalue, and prices it under
// dynamic pricing. It returns the best positive-delta upgrade (highest delta,
// tie-break lower price).
//
//   - onlyAffordable=true filters to price <= budget (used for "buy now").
//   - onlyAffordable=false ignores budget (used to decide whether to save up).
//   - minDelta gates out marginal upgrades.
//
// Rescans on each call (zone shop count is small); callers do not cache the
// chosen item, only optionally the sell-shop room.
func scanZoneUpgrades(mob *mobs.Mob, profile itemvalue.WeightProfile, budget int, onlyAffordable bool, minDelta float64) (upgradeCandidate, bool) {
	if mob == nil {
		return upgradeCandidate{}, false
	}
	cfg := shops.PricingConfigFromBalance()
	var best upgradeCandidate
	found := false

	for _, shop := range shops.AllShops() {
		if shop.Zone != mob.Character.Zone {
			continue
		}
		for i := range shop.Stock {
			e := shop.Stock[i]
			if e.Current <= 0 {
				continue
			}
			spec := items.GetItemSpec(e.ItemId)
			if spec == nil {
				continue
			}
			candidate := items.New(e.ItemId)
			delta := itemvalue.ItemValueDelta(&mob.Character, profile, candidate)
			if delta.Slot == "" || delta.Score <= minDelta {
				continue
			}
			price := shops.CalcSellPrice(spec.Value, e.Current, stockRestockNorm(e), cfg)
			if onlyAffordable && price > budget {
				continue
			}
			better := !found ||
				delta.Score > best.Delta ||
				(delta.Score == best.Delta && price < best.Price)
			if better {
				best = upgradeCandidate{
					ShopRoom: shop.RoomId,
					ItemId:   e.ItemId,
					ItemName: candidate.Name(),
					Price:    price,
					Delta:    delta.Score,
				}
				found = true
			}
		}
	}
	return best, found
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/planners/ -run 'TestStockRestockNorm|TestScanZoneUpgrades' -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/planners/shop_upgrade.go internal/planners/shop_upgrade_test.go
git commit -m "feat(planners): itemvalue-driven zone shop upgrade evaluator (5.3)"
```

---

## Task 3: `upgrade-gear` goal type

**Files:**
- Create: `internal/goals/catalog/upgrade_gear.go`
- Test: `internal/goals/catalog/upgrade_gear_test.go`

Design: `Predicate` always false (perpetual drive). `ContextScore` returns a
positive floor (1.0) always so the 4.6 dormancy sweep never abandons this
standing default; it rises to 2.5 when the mob is out of combat AND has a
plausible path to a purchase (spendable gold above reserve, or sellable loot to
fund saving). The actual "is anything in stock" decision lives in the planner —
ContextScore stays cheap and self-contained (no shop scan), matching the 4.3
`mastery-equip` precedent that deliberately avoided cross-package shop scans in
context scoring.

- [ ] **Step 1: Write the failing test**

Create `internal/goals/catalog/upgrade_gear_test.go`:

```go
package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestUpgradeGear_PredicateAlwaysFalse(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.Gold = 99999
	g := &goals.Goal{Type: "upgrade-gear"}
	if upgradeGearPredicate(g, mob) {
		t.Errorf("upgrade-gear predicate must always be false (perpetual drive)")
	}
}

func TestUpgradeGear_PredicateNilMob(t *testing.T) {
	if upgradeGearPredicate(&goals.Goal{Type: "upgrade-gear"}, nil) {
		t.Errorf("predicate must be false for nil mob")
	}
}

func TestUpgradeGear_ContextScore_FloorWhenBroke(t *testing.T) {
	mob := &mobs.Mob{}            // 0 gold, no items, not in combat
	g := &goals.Goal{Type: "upgrade-gear"}
	if got := upgradeGearContextScore(g, mob); got != 1.0 {
		t.Errorf("context score = %v, want floor 1.0 (broke + no loot)", got)
	}
}

func TestUpgradeGear_ContextScore_ElevatedWhenGold(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.Gold = 1000 // well above default reserve 50
	g := &goals.Goal{Type: "upgrade-gear"}
	if got := upgradeGearContextScore(g, mob); got != 2.5 {
		t.Errorf("context score = %v, want 2.5 (spendable gold, idle)", got)
	}
}

func TestUpgradeGear_ContextScore_FloorInCombat(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.Gold = 1000
	mob.Character.Aggro = &mobs.Aggro{} // in combat
	g := &goals.Goal{Type: "upgrade-gear"}
	if got := upgradeGearContextScore(g, mob); got != 1.0 {
		t.Errorf("context score = %v, want floor 1.0 while in combat", got)
	}
}

func TestUpgradeGear_ContextScore_NilMob(t *testing.T) {
	if got := upgradeGearContextScore(&goals.Goal{}, nil); got != 0 {
		t.Errorf("context score = %v, want 0 for nil mob", got)
	}
}
```

> Verify the `mobs.Aggro{}` literal compiles — grep `type Aggro` in
> `internal/mobs`. If the combat flag is set a different way (e.g. a method),
> set `mob.Character.Aggro` to whatever non-nil value the survival predicate
> test uses; `survival.go` checks `mob.Character.Aggro == nil`, so any non-nil
> pointer of the correct type works.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/goals/catalog/ -run TestUpgradeGear -count=1`
Expected: FAIL — `upgradeGearPredicate` / `upgradeGearContextScore` undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/goals/catalog/upgrade_gear.go`:

```go
package catalog

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const (
	upgradeGearFloorScore    = 1.0 // never 0 → 4.6 dormancy never abandons this standing default
	upgradeGearActiveScore   = 2.5 // idle + a plausible path to a purchase
)

func init() {
	goals.RegisterGoalType("upgrade-gear", goals.GoalTypeMeta{
		Predicate:    upgradeGearPredicate,
		ContextScore: upgradeGearContextScore,
		// AllowMultiple: false — one upgrade-gear drive per mob.
		Params: []goals.ParamSchema{
			{Key: "reserve", Required: false, GoType: "int"},
		},
	})
}

// upgradeGearPredicate always returns false: a mob can always want better
// gear, so this drive has no terminal satisfied state. Activation is governed
// entirely by ContextScore + the planner.
func upgradeGearPredicate(_ *goals.Goal, _ *mobs.Mob) bool {
	return false
}

// upgradeGearReserve returns the gold floor a shopping mob keeps, from the
// goal's optional "reserve" param or the MobUpgradeGoldReserve config default.
func upgradeGearReserve(g *goals.Goal) int {
	def := int(configs.GetBalanceConfig().MobUpgradeGoldReserve)
	return paramIntOr(g, "reserve", def)
}

// upgradeGearContextScore:
//   - 0 for nil mob.
//   - floor (1.0) while in combat (Aggro != nil) — survival/combat goals win;
//     staying at the floor keeps the drive from 4.6 dormancy abandonment.
//   - active (2.5) when idle AND there is a plausible path to a purchase:
//     spendable gold above the reserve, OR sellable loot to fund saving.
//   - floor (1.0) otherwise (idle but broke and nothing to sell).
//
// Deliberately cheap and self-contained — no shop scan. The planner is the
// authority on whether anything is actually in stock.
func upgradeGearContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	if mob.Character.Aggro != nil {
		return upgradeGearFloorScore
	}
	spendable := mob.Character.Gold - upgradeGearReserve(g)
	if spendable > 0 || mobHasAnySellableLoot(mob) {
		return upgradeGearActiveScore
	}
	return upgradeGearFloorScore
}

// mobHasAnySellableLoot reports whether the mob carries at least one
// non-quest item with positive vendor value. Mirrors the planners-package
// mobHasSellableItems; duplicated to keep catalog free of a planners import
// (the same intentional duplication noted across the goals/catalog and
// planners packages).
func mobHasAnySellableLoot(mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	for i := range mob.Character.Items {
		spec := mob.Character.Items[i].GetSpec()
		if spec.QuestToken != "" {
			continue
		}
		if spec.Value > 0 {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/goals/catalog/ -run TestUpgradeGear -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/goals/catalog/upgrade_gear.go internal/goals/catalog/upgrade_gear_test.go
git commit -m "feat(goals): perpetual upgrade-gear goal type with context floor (5.3)"
```

---

## Task 4: `upgrade-gear` planner

**Files:**
- Create: `internal/planners/upgrade_gear.go`
- Test: `internal/planners/upgrade_gear_test.go`

State machine (see spec §3). MiscData keys (all `plan:`-prefixed so
`ClearPlanState` wipes them on goal switch):
- `plan:upgrade-gear:pending_equip` — set to 1 the tick after a `buy`; the next
  tick emits `gearup` instead of re-scanning (prevents same-target double-buy).
- `plan:upgrade-gear:sell_shop_room` — sticky buying-vendor room for the
  save-up sell loop (mirrors `wealth-gold`). The buy *target* is rescanned each
  tick (cheap; avoids stale-target bugs), so no buy-shop sticky.

- [ ] **Step 1: Write the failing test**

Create `internal/planners/upgrade_gear_test.go`:

```go
package planners

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestUpgradeGear_Registered(t *testing.T) {
	if LookupPlanner("upgrade-gear") == nil {
		t.Fatalf("upgrade-gear planner not registered")
	}
}

func TestUpgradeGear_NilMob_Failure(t *testing.T) {
	fn := LookupPlanner("upgrade-gear")
	res := fn(nil, &goals.Goal{Type: "upgrade-gear"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want StatusFailure (nil mob)", res.Status)
	}
}

// No shops + no items loaded in unit-test context → the evaluator finds
// nothing and there's nothing to sell → idle (empty command, Running).
func TestUpgradeGear_NothingInStock_Idle(t *testing.T) {
	fn := LookupPlanner("upgrade-gear")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	mob.Character.Gold = 1000
	res := fn(mob, &goals.Goal{Type: "upgrade-gear"})
	if res.Command != "" {
		t.Errorf("command=%q, want empty (nothing in stock, nothing to sell)", res.Command)
	}
	if res.Status != StatusRunning {
		t.Errorf("status=%v, want StatusRunning", res.Status)
	}
}

// The pending-equip one-shot: when the flag is set, the planner emits gearup,
// clears the flag, and does not re-evaluate stock.
func TestUpgradeGear_PendingEquip_EmitsGearup(t *testing.T) {
	fn := LookupPlanner("upgrade-gear")
	mob := &mobs.Mob{}
	mob.Character.MiscData = map[string]any{
		upgradePendingEquipKey: 1,
	}
	res := fn(mob, &goals.Goal{Type: "upgrade-gear"})
	if res.Command != "gearup" {
		t.Errorf("command=%q, want gearup", res.Command)
	}
	if res.Status != StatusRunning {
		t.Errorf("status=%v, want StatusRunning", res.Status)
	}
	if mobMiscIntOr(mob, upgradePendingEquipKey, 0) != 0 {
		t.Errorf("pending-equip flag should be cleared after gearup")
	}
}

func TestUpgradeGear_MiscKeys_HavePlanPrefix(t *testing.T) {
	for _, k := range []string{upgradePendingEquipKey, upgradeSellShopRoomKey} {
		if !strings.HasPrefix(k, PlanKeyPrefix) {
			t.Errorf("key %q does not start with PlanKeyPrefix %q", k, PlanKeyPrefix)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/planners/ -run TestUpgradeGear -count=1`
Expected: FAIL — planner + key constants undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/planners/upgrade_gear.go`:

```go
package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/itemvalue"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const (
	upgradePendingEquipKey  = "plan:upgrade-gear:pending_equip"
	upgradeSellShopRoomKey  = "plan:upgrade-gear:sell_shop_room"
)

func init() {
	RegisterPlanner("upgrade-gear", upgradeGearPlanner)
}

// upgradeGearPlanner: survey-worst-slot equipment shopping. Each tick the mob
// (idle, out of combat by selection contract):
//  1. Just bought last tick → gearup to wear it, clear flag.
//  2. Affordable in-stock upgrade → at shop: buy it (+ flag equip next tick);
//     else pathto the shop.
//  3. Upgrade exists but unaffordable → save up via the wealth-gold sell loop.
//  4. Nothing in stock → idle (Running, empty command).
func upgradeGearPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}

	// (1) Equip what we bought last tick.
	if mobMiscIntOr(mob, upgradePendingEquipKey, 0) == 1 {
		mobSetMisc(mob, upgradePendingEquipKey, 0)
		return PlanResult{Command: "gearup", Status: StatusRunning}
	}

	b := configs.GetBalanceConfig()
	reserve := paramIntOr(goal, "reserve", int(b.MobUpgradeGoldReserve))
	minDelta := float64(b.MobUpgradeMinDelta)
	profile := itemvalue.ProfileFor(mob.Archetype, mob.BehaviorArchetype)
	budget := mob.Character.Gold - reserve

	// (2) Affordable upgrade?
	if budget > 0 {
		if cand, ok := scanZoneUpgrades(mob, profile, budget, true, minDelta); ok {
			if mob.Character.RoomId == cand.ShopRoom {
				mobSetMisc(mob, upgradePendingEquipKey, 1)
				return PlanResult{Command: "buy " + cand.ItemName, Status: StatusRunning}
			}
			return PlanResult{Command: "pathto " + strconv.Itoa(cand.ShopRoom), Status: StatusRunning}
		}
	}

	// (3) Upgrade exists but unaffordable → save up by selling loot.
	if _, exists := scanZoneUpgrades(mob, profile, 0, false, minDelta); exists {
		if mobHasSellableItems(mob) {
			if mobInVendorRoom(mob) {
				return PlanResult{Command: "sell all", Status: StatusRunning}
			}
			room := mobMiscIntOr(mob, upgradeSellShopRoomKey, 0)
			if room == 0 {
				if r, ok := findShopInZoneBuying(mob); ok {
					room = r
					mobSetMisc(mob, upgradeSellShopRoomKey, room)
				}
			}
			if room != 0 {
				return PlanResult{Command: "pathto " + strconv.Itoa(room), Status: StatusRunning}
			}
		}
		// Nothing to sell (or no buying vendor) → wander hoping for loot.
		return PlanResult{Command: "wander", Status: StatusRunning}
	}

	// (4) Nothing in stock anywhere in zone → idle.
	return PlanResult{Status: StatusRunning}
}
```

> Note the `paramIntOr` used here is the **planners**-package helper in
> `helpers.go` (already defined), not the catalog one.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/planners/ -run TestUpgradeGear -count=1`
Expected: PASS

- [ ] **Step 5: Run the full planners + goals package tests**

Run: `go test ./internal/planners/ ./internal/goals/... -count=1`
Expected: PASS (no regressions in existing planner/goal tests).

- [ ] **Step 6: Commit**

```bash
git add internal/planners/upgrade_gear.go internal/planners/upgrade_gear_test.go
git commit -m "feat(planners): upgrade-gear survey-worst-slot shopping planner (5.3)"
```

---

## Task 5: Archetype wiring

**Files:**
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/thief.yaml:109-115`
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/guard_captain.yaml:113-115`

No test file — validated by the boot smoke in Task 6 (archetype `default_goals`
parse + goal-type registration symmetry check run at startup).

- [ ] **Step 1: Add upgrade-gear to thief**

In `thief.yaml`, the `default_goals:` block currently ends with the
`wealth-gold` entry (priority 40, `params: target: 500`). Append:

```yaml
  - type: upgrade-gear
    priority: 30
```

Result (for reference — match existing indentation exactly):

```yaml
default_goals:
  - type: survival
    priority: 80
  - type: wealth-gold
    priority: 40
    params:
      target: 500
  - type: upgrade-gear
    priority: 30
```

- [ ] **Step 2: Add upgrade-gear to guard_captain**

In `guard_captain.yaml`, the `default_goals:` block currently has only the
`survival` entry. Append:

```yaml
  - type: upgrade-gear
    priority: 30
```

Result:

```yaml
default_goals:
  - type: survival
    priority: 80
  - type: upgrade-gear
    priority: 30
```

- [ ] **Step 3: Build to confirm YAML is structurally referenced by no code change**

Run: `go build ./...`
Expected: clean (YAML is data; this just confirms nothing else broke).

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/thief.yaml _datafiles/world/dogmud/behaviors/archetypes/guard_captain.yaml
git commit -m "content(archetypes): seed upgrade-gear drive on thief + guard_captain (5.3)"
```

---

## Task 6: Boot smoke + docs + roadmap

**Files:**
- Modify: `internal/planners/context.md`
- Modify: `internal/goals/catalog/context.md`
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Full build + test sweep**

Run: `go build ./... && go test ./internal/planners/ ./internal/goals/... ./internal/configs/ -count=1`
Expected: build clean, all PASS.

- [ ] **Step 2: Boot smoke (catches data-load panics the build can't)**

Wipe instance saves per the project SOP, then boot and watch for clean
data-file loading (no panic on archetype `default_goals` parse or goal-type
registration):

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | head -80
```

Expected: lines like `mobs.LoadDataFiles() loadedCount=...` and
`quests.LoadDataFiles() loadedCount=...` with NO panic. Stop the server once
load completes cleanly (Ctrl-C). (Do NOT wipe `_datafiles/world/dogmud/shops/`.)

- [ ] **Step 3: Update planners context.md**

Add an entry to `internal/planners/context.md` documenting the new
`upgrade-gear` planner + `scanZoneUpgrades` evaluator: it scores zone-shop stock
via `itemvalue.ItemValueDelta`, prices via `shops.CalcSellPrice`, buys the best
affordable upgrade and `gearup`s it, and falls back to the `wealth-gold` sell
loop to save up. Note the two MiscData keys and that the buy target is rescanned
each tick (only the sell-shop room is sticky).

- [ ] **Step 4: Update goals/catalog context.md**

Add an entry for the `upgrade-gear` goal type: perpetual drive (`Predicate`
always false), `ContextScore` floor 1.0 / active 2.5 (idle + spendable gold or
sellable loot), self-contained (no shop scan — the planner owns stock
decisions), optional `reserve` param.

- [ ] **Step 5: Flip roadmap status + note 5.1d**

In `MOB_ALIVENESS_ROADMAP.md`:
- In the progress tracker table (line ~112), change the 5.3 row Status from
  `Not started` to `Done` (fill the shipped date on completion of this plan).
- In the §5.3 mini-brief (line ~867), change `**Status:** Not started` to
  `**Status:** Done (2026-06-01)` and add a `**Shipped:**` paragraph summarizing
  the evaluator + goal + planner + knobs + archetype wiring, mirroring the style
  of the 5.2 entry, and listing the OUT-of-scope items (cross-respawn
  persistence, adjacent-zone shopping, mastery-equip left untouched).
- In the §5.1 mini-brief, append a note that 5.1d redemption is considered
  "done as far as we are taking it" per the 2026-06-01 decision (the user's
  call when starting 5.3), so Phase 5 forward work is 5.4 (gated on this chunk)
  and the 6.x polish phase.
- Update the roll-up count (e.g. `29 / 42 done`).

- [ ] **Step 6: Commit**

```bash
git add internal/planners/context.md internal/goals/catalog/context.md MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(5.3): planner/catalog context + roadmap status; note 5.1d closed"
```

---

## Self-review notes (author)

- **Spec coverage:** §1 evaluator → Task 2; §2 goal type → Task 3; §3 planner →
  Task 4; §4 config knobs → Task 1; §5 archetype wiring → Task 5; §6 out-of-scope
  honored (no `mastery-equip` edits, same-zone only via `shops` zone filter, no
  persistence work); §7 testing → per-task unit tests + Task 6 boot smoke; the
  in-game smoke is deferred to the user per the §7 note and 2.8/2.9/2.10
  precedent.
- **Price direction:** the spec's earlier `CalcBuyPrice` slip was corrected to
  `CalcSellPrice`; this plan uses `CalcSellPrice` throughout (the price the
  buyer pays).
- **Type consistency:** `scanZoneUpgrades` signature and `upgradeCandidate`
  fields are identical across Tasks 2 and 4; MiscData key constant names
  (`upgradePendingEquipKey`, `upgradeSellShopRoomKey`) match between the planner
  and its tests; the goal's `reserve` param key matches between catalog
  (`upgradeGearReserve`) and planner.
- **Known unit-test limitation (honest):** the live scoring/pricing path can't
  be unit-tested without a data-file load, so its correctness is validated by
  the deferred in-game smoke. Unit tests cover the pure helper
  (`stockRestockNorm`), the nil/empty-registry branches, the goal predicate +
  context-score tiers, and the planner branch shapes (nil, idle, pending-equip,
  key-prefix).
