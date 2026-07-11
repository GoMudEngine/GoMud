# Affixed Gear — Stage 1 (Value Primitive) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give affix-scaled instance loot a gold value that reflects its affix
power. Add a pure `items.AffixValue` function + a `Balance.GoldPerAffixPoint`
knob, and stamp the computed value onto the per-instance spec in
`GenerateAffixedItem`.

**Architecture:** `AffixValue` is a pure function of (affixed spec, base spec,
goldPerPoint) — no config/registry deps — so it's trivially unit-tested. It
reconstructs the affix cost-point budget from the spec diff (same cost weights
as `affixgen.go`'s `allBonusTypes`) and returns `baseValue + points ×
goldPerPoint`. `GenerateAffixedItem` gains a `goldPerPoint` param (mirroring the
existing `scalar` param — the `items` package stays config-free; callers inject
config values) and stamps `specCopy.Value`.

**Tech Stack:** Go. Pure functions + table tests. No new deps.

**Spec:** `docs/superpowers/specs/2026-07-08-shops-trade-affixed-gear-design.md`

**Behavioral safety:** This stage stamps `Value` but changes **no** sell/shop
behavior — affixed items remain `IsSpecial`-blocked from selling (Stage 2 lifts
that). So the stamped value is inert until Stage 2/3. `go build ./...` + full
suite must stay green.

---

## Task 1: Add the `GoldPerAffixPoint` config knob

**Files:**
- Modify: `internal/configs/config.balance.go` (field, near `LootBudgetScalar` ~L508)
- Modify: `internal/configs/config.balance.misc.go` (default, near the `LootBudgetScalar` default ~L275)

- [ ] **Step 1: Add the field**

In `internal/configs/config.balance.go`, right after the `LootBudgetScalar`
line:

```go
	GoldPerAffixPoint ConfigFloat `yaml:"GoldPerAffixPoint"` // Gold value per affix cost-point on instance/affixed loot (default 3.0)
```

- [ ] **Step 2: Add the default**

In `internal/configs/config.balance.misc.go`, near the `LootBudgetScalar`
default block:

```go
	if b.GoldPerAffixPoint <= 0 {
		b.GoldPerAffixPoint = 3.0
	}
```

- [ ] **Step 3: Verify it builds**

Run: `go build ./internal/configs/`
Expected: builds clean.

- [ ] **Step 4: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.misc.go
git commit -m "feat(config): add Balance.GoldPerAffixPoint (default 3.0)"
```

---

## Task 2: `items.AffixValue` + `affixPoints` (pure) + tests

**Files:**
- Modify: `internal/items/affixgen.go` (add the two functions; `math` is already imported)
- Test: `internal/items/affixgen_test.go` (add table test; import `statmods`)

- [ ] **Step 1: Write the failing test**

Append to `internal/items/affixgen_test.go` (add
`"github.com/GoMudEngine/GoMud/internal/statmods"` to its imports):

```go
func TestAffixValue(t *testing.T) {
	base := ItemSpec{Value: 100}

	tests := []struct {
		name    string
		affixed ItemSpec
		gpp     float64
		want    int
	}{
		{"no affixes", ItemSpec{Value: 100}, 3.0, 100},
		// 3 strength: 3 points each? No — stat affix costs 3 per +1, so +3 str = 9 points; 9*3 = 27.
		{"stat +3 strength", ItemSpec{Value: 100, StatMods: statmods.StatMods{"strength": 3}}, 3.0, 100 + 9*3},
		// skill +1 costs 12 points; 12*3 = 36.
		{"skill +1 weapon-combat", ItemSpec{Value: 100, StatMods: statmods.StatMods{"weapon-combat": 1}}, 3.0, 100 + 12*3},
		// mitigation +5 costs 5 each = 25 points; 25*3 = 75.
		{"phys mit +5", ItemSpec{Value: 100, PhysicalMitigation: 5}, 3.0, 100 + 25*3},
		// damage_mult_phys: +0.10 = 2 ranks * 8 = 16 points; 16*3 = 48.
		{"damage phys +0.10", ItemSpec{Value: 100, DamageMultiplier: 0.10}, 3.0, 100 + 16*3},
		// damage_mult_both: +0.05 phys (8) + 0.05 spell (4) = 12 points; 12*3 = 36.
		{"damage both +0.05", ItemSpec{Value: 100, DamageMultiplier: 0.05, SpellDamageMultiplier: 0.05}, 3.0, 100 + 12*3},
		// goldPerPoint scaling: same +3 str at gpp=2 → 9*2 = 18.
		{"gpp scales linearly", ItemSpec{Value: 100, StatMods: statmods.StatMods{"strength": 3}}, 2.0, 100 + 9*2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AffixValue(tt.affixed, base, tt.gpp)
			if got != tt.want {
				t.Errorf("AffixValue(%s) = %d; want %d", tt.name, got, tt.want)
			}
		})
	}
}

// A base with its own damage/stats must only count the POSITIVE delta above base.
func TestAffixValue_OnlyCountsDeltaAboveBase(t *testing.T) {
	base := ItemSpec{Value: 80, DamageMultiplier: 1.0, StatMods: statmods.StatMods{"strength": 2}}
	affixed := ItemSpec{Value: 80, DamageMultiplier: 1.10, StatMods: statmods.StatMods{"strength": 5}}
	// damage +0.10 = 2 ranks*8 = 16; strength +3 = 9; total 25 points; *3 = 75.
	want := 80 + 25*3
	if got := AffixValue(affixed, base, 3.0); got != want {
		t.Errorf("AffixValue delta = %d; want %d", got, want)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/items/ -run TestAffixValue -v`
Expected: FAIL — `undefined: AffixValue`.

- [ ] **Step 3: Implement**

Append to `internal/items/affixgen.go`:

```go
// AffixValue computes the gold value of an affixed item instance: the base
// template value plus a premium for each affix cost-point of power added above
// the base, priced at goldPerPoint gold per point. affixedSpec is the item's
// per-instance spec; baseSpec is the unmodified template. Only positive deltas
// count (affixes never subtract). See the design spec's Calibration section.
func AffixValue(affixedSpec, baseSpec ItemSpec, goldPerPoint float64) int {
	pts := affixPoints(affixedSpec, baseSpec)
	return baseSpec.Value + int(math.Round(float64(pts)*goldPerPoint))
}

// affixPoints reconstructs the affix cost-point budget spent on affixedSpec vs
// baseSpec, using the same per-affix cost weights as allBonusTypes:
// damage_mult_phys 8 (+0.05/rank), the spell half of damage_mult_both 4,
// mitigations 5, stats 3, skills 12.
func affixPoints(affixedSpec, baseSpec ItemSpec) int {
	pts := 0

	if d := affixedSpec.DamageMultiplier - baseSpec.DamageMultiplier; d > 0 {
		pts += int(math.Round(d/0.05)) * 8
	}
	if d := affixedSpec.SpellDamageMultiplier - baseSpec.SpellDamageMultiplier; d > 0 {
		pts += int(math.Round(d/0.05)) * 4
	}
	if d := affixedSpec.PhysicalMitigation - baseSpec.PhysicalMitigation; d > 0 {
		pts += d * 5
	}
	if d := affixedSpec.MagicalMitigation - baseSpec.MagicalMitigation; d > 0 {
		pts += d * 5
	}
	if d := affixedSpec.ConvictionMitigation - baseSpec.ConvictionMitigation; d > 0 {
		pts += d * 5
	}
	for k, v := range affixedSpec.StatMods {
		base := 0
		if baseSpec.StatMods != nil {
			base = baseSpec.StatMods[k]
		}
		if d := v - base; d > 0 {
			if isSkillMod(k) {
				pts += d * 12
			} else {
				pts += d * 3
			}
		}
	}
	return pts
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/items/ -run TestAffixValue -v`
Expected: PASS (both tests, all subcases).

- [ ] **Step 5: Commit**

```bash
git add internal/items/affixgen.go internal/items/affixgen_test.go
git commit -m "feat(items): AffixValue — gold value from affix cost-point diff"
```

---

## Task 3: Stamp `Value` in `GenerateAffixedItem` (+ caller updates)

Add a `goldPerPoint` param to `GenerateAffixedItem`, stamp
`specCopy.Value = AffixValue(specCopy, baseSpec, goldPerPoint)` after bonuses are
applied, and update the two callers to inject the config value.

**Files:**
- Modify: `internal/items/affixgen.go` (signature + stamp)
- Modify: `internal/rooms/rooms.go` (~L855–857 caller)
- Modify: `internal/bountyhunter/bountyhunter.go` (~L86–88 caller)
- Test: `internal/items/affixgen_test.go`

- [ ] **Step 1: Write the failing test (stamping invariant)**

Append to `internal/items/affixgen_test.go`:

```go
// TestGenerateAffixedItem_StampsValue proves the generated instance's Value is
// self-consistent with its rolled affixes: Value == AffixValue(spec, base, gpp).
// RNG-independent — whatever affixes roll, the stamped value must match them.
func TestGenerateAffixedItem_StampsValue(t *testing.T) {
	cleanup := SeedItemsForTest(map[int]*ItemSpec{
		9100: {ItemId: 9100, Name: "Test Torc", Type: Neck, Value: 85},
	})
	defer cleanup()

	base := New(9100).GetSpec() // base template spec (Value 85, no affixes)

	// goldPaid 200, scalar 7 → budget ~98 points, so Value should exceed base.
	item := GenerateAffixedItem(9100, 200, 7.0, 3.0)

	if item.Spec == nil {
		t.Fatal("expected a per-instance spec on an affixed item")
	}
	stamped := item.Spec.Value
	recomputed := AffixValue(*item.Spec, base, 3.0)
	if stamped != recomputed {
		t.Errorf("stamped Value %d != AffixValue(spec) %d", stamped, recomputed)
	}
	if stamped <= base.Value {
		t.Errorf("stamped Value %d should exceed base %d for a budgeted item", stamped, base.Value)
	}
}
```

(If `SeedItemsForTest` needs a companion registry seeded — mirror an existing
`items` test that calls `New(...)`; if none exists in-package, the
`SeedItemsForTest` helper used by `internal/usercommands/usercommands_test.go`
is the reference for the exact map shape.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/items/ -run TestGenerateAffixedItem_StampsValue -v`
Expected: FAIL — `GenerateAffixedItem` currently takes 3 args, not 4 (compile
error), and doesn't stamp Value.

- [ ] **Step 3: Add the param + stamp**

In `internal/items/affixgen.go`, change the signature:

```go
func GenerateAffixedItem(baseItemId int, goldPaid int, scalar float64, goldPerPoint float64) Item {
```

Then, just before `item.Spec = &specCopy` (near the end), stamp the value:

```go
	// Value scales with the affix power added (Stage 1: shops-trade-affixed-gear).
	specCopy.Value = AffixValue(specCopy, baseSpec, goldPerPoint)

	item.Spec = &specCopy
```

Note: the two early-return paths (`rawBudget <= 0`, `len(eligible) == 0`) return
the plain unmodified `item` (no per-instance spec) — leave those as-is; an item
with no affixes correctly keeps its base value and no `Spec`.

- [ ] **Step 4: Update the callers**

In `internal/rooms/rooms.go` (the instance-loot block ~L855), the `scalar` is
already read from config; add the goldPerPoint and pass it:

```go
	scalar := float64(configs.GetBalanceConfig().LootBudgetScalar)
	goldPerPoint := float64(configs.GetBalanceConfig().GoldPerAffixPoint)
	for _, baseItemId := range mob.LootPool {
		affixedItem := items.GenerateAffixedItem(baseItemId, goldPaid, scalar, goldPerPoint)
```

In `internal/bountyhunter/bountyhunter.go` (~L86), `bal` is already in scope:

```go
	scalar := float64(bal.LootBudgetScalar)
	goldPerPoint := float64(bal.GoldPerAffixPoint)
	for _, baseId := range hunter.LootPool {
		affixed := items.GenerateAffixedItem(baseId, gg, scalar, goldPerPoint)
```

- [ ] **Step 5: Run tests + full build**

Run: `go test ./internal/items/ -run 'TestAffixValue|TestGenerateAffixedItem' -v && go build ./...`
Expected: PASS + build clean (both callers updated; no other callers exist —
verify with `grep -rn "GenerateAffixedItem(" internal --include=*.go | grep -v _test`).

- [ ] **Step 6: Full suite (no behavior regression)**

Run: `go test ./internal/items/ ./internal/rooms/ ./internal/bountyhunter/`
Expected: PASS. Affixed items now carry a scaled `Value` but remain unsellable
(`IsSpecial`), so nothing else changes.

- [ ] **Step 7: Commit**

```bash
git add internal/items/affixgen.go internal/items/affixgen_test.go internal/rooms/rooms.go internal/bountyhunter/bountyhunter.go
git commit -m "feat(items): stamp affix-scaled Value in GenerateAffixedItem (goldPerPoint param)"
```

---

## Definition of Done (Stage 1)

- `items.AffixValue`/`affixPoints` exist, pure, table-tested across stat / skill /
  mitigation / damage-phys / damage-both / gpp-scaling / delta-above-base cases.
- `GenerateAffixedItem` stamps `specCopy.Value = AffixValue(...)`; both callers
  inject `GoldPerAffixPoint`.
- `Balance.GoldPerAffixPoint` defaults to 3.0.
- `go build ./...` clean; full suite green; **no sell/shop behavior changed**
  (affixed items still `IsSpecial`-blocked — Stage 2 lifts that).

## Divergences From Spec (this stage)

- `GenerateAffixedItem` gains a `goldPerPoint` param (spec implied config-read;
  the param keeps the `items` package config-free, matching the existing
  `scalar` param, at the cost of a one-line change to each of the 2 callers).
- Stamping test asserts the RNG-independent invariant `Value ==
  AffixValue(spec, base, gpp)` rather than an exact number (budget has gaussian
  variance + random affix selection).
