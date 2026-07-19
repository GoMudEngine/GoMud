# Companion Conviction Economy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `min(4, manifestationSkill/19)` companion count cap with a Conviction-reservation economy: each companion reserves Conviction (per summon type), manifestation skill + a Manifester mutation reduce the cost, and the Conviction budget becomes the real limit.

**Architecture:** A live companion's reservation is stored on `CompanionInfo.ConvictionReserve` (snapshotted at summon time) and summed into the existing `GetPoolReservation("conviction", …)` machinery, which already clamps usable Conviction. The per-type base cost is a new `SpellData.SummonConvictionReserve` field; the reduction is computed by `Character.CalcCompanionReserve`; the summon/charm gate becomes `Character.CanAffordCompanion` (budget, not count). `GetMaxCompanions` degrades to a soft backstop.

**Tech Stack:** Go. Packages touched: `internal/characters`, `internal/mutations`, `internal/spells`, `internal/configs`, `internal/hooks`, plus spell YAML data files. Verify symbols with the codegraph MCP, not grep.

**Spec:** `docs/superpowers/specs/2026-07-11-companion-conviction-economy-design.md`

---

## Verified context (already confirmed via codegraph — do not re-discover)

- `internal/characters/companions.go` — holds `CompanionInfo` (struct, ~line 53), `GetMaxCompanions` (~line 130, currently `skill/19` capped 4), `CalcCompanionStatPool`, `AddCompanion`/`RemoveCompanion`. Imports `configs`, `math`, `skills`, `mutations` are available (companions.go already imports `configs`, `math`, `skills`; `mutations` is imported elsewhere in the package, e.g. validate.go).
- `internal/characters/validate.go:241` — `func (c *Character) GetPoolReservation(pool string, poolMax int) int` iterates equipment and returns **absolute reserved pool points** (sums `floor(poolMax*pct)`). Called only by `RecalculateStats` (validate.go:29), which clamps `c.Conviction` to `ConvictionMax.Value - reservation` (validate.go:162-170). `ConvictionMax.Value` is already post-`BodyConvictionScale` (pole decay applied at validate.go:117).
- `internal/spells/spells.go:19` — `SpellData` struct. Summon fields at lines 44-50: `SummonMobId`, `SummonBasePool`, `SummonScalingDivisor`, `SummonComponentId`, `SummonRequiresCorpse`, `SummonMinCorpsePool`. `Cost` (line 27) is the one-time cast cost, independent of the ongoing reservation.
- `internal/hooks/companion_summon.go:24` — `resolveCompanionSummon`. Count check at lines 37-41 (`len(ch.Companions) >= ch.GetMaxCompanions()`). Registers `CompanionInfo` at lines 150-157. `charisma := ch.Stats.Charisma.ValueAdj`, `manifestSkill := ch.GetSkillLevel(skills.Manifestation)` already computed at lines 120-121.
- `internal/mutations/mutations.go` — `sumEffects(owned, type, target)` scales `p.Value * LevelMultiplier(level)`. `LevelMultiplier` (mutations.go:315) is a **config curve** (`MutationLevel2/3/4Multiplier`), NOT linear rank — so the Manifester reducer reads rank directly, it does NOT go through `sumEffects`. `GetMutationFlags`/`HasMutationFlag` are the flag readers. `DescribeEffect` (describe.go:10) renders effect phrases.
- `internal/configs/config.balance.go:7` — `Balance` struct. Manifestation knobs already here: `ManifestStatScaleChaFactor`, `ManifestStatScaleSkillFactor`. Add the new knobs adjacent to these, matching the existing custom numeric field types (e.g. `ConfigFloat`/`ConfigInt` — copy the neighbor's type). Defaults are applied in the same file's default/validate path — grep `ManifestStatScaleChaFactor` to find where its default is set and add the new defaults there.

---

## File Structure

- `internal/characters/companions.go` — add `CompanionInfo.ConvictionReserve` field; rewrite `GetMaxCompanions` to a soft backstop; add `CalcCompanionReserve` and `CanAffordCompanion`.
- `internal/characters/validate.go` — extend `GetPoolReservation` with the companion term (conviction only).
- `internal/spells/spells.go` — add `SummonConvictionReserve` field.
- `internal/mutations/mutations.go` — add `GetCompanionReserveRank`.
- `internal/mutations/describe.go` — add a `companion_reserve_reduction` case.
- `internal/configs/config.balance.go` — add 9 knobs + defaults.
- `internal/hooks/companion_summon.go` — wire the budget gate + reservation snapshot into the summon path.
- Charm path (`resolveCharmSpell`) and btree (`actSummonCompanion`) — same wiring (find exact locations via codegraph).
- `_datafiles/world/dogmud/spells/*.yaml` — set `summon_conviction_reserve:` per tier on every summon/raise/charm spell.
- Tests colocated: `companions_test.go`, `validate_test.go` (or nearest), `mutations_test.go`, plus a calibration acceptance test.

---

### Task 1: Data model — reservation field + per-type cost field

**Files:**
- Modify: `internal/characters/companions.go` (CompanionInfo struct, ~line 72 — after `Equipment`)
- Modify: `internal/spells/spells.go:50` (after `SummonMinCorpsePool`)
- Test: `internal/characters/companions_test.go`

- [ ] **Step 1: Write the failing test**

Add to `companions_test.go`:

```go
func TestCompanionInfo_ConvictionReserveField(t *testing.T) {
	c := New()
	c.Skills[string(skills.Manifestation)] = 19
	comp := CompanionInfo{MobId: 1001, InstanceId: 5, Name: "Spirit Wolf", ConvictionReserve: 333}
	require.True(t, c.AddCompanion(comp))
	assert.Equal(t, 333, c.Companions[0].ConvictionReserve)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/characters/ -run TestCompanionInfo_ConvictionReserveField`
Expected: FAIL — `unknown field 'ConvictionReserve' in struct literal`.

- [ ] **Step 3: Add the fields**

In `companions.go`, inside `CompanionInfo`, after the `Equipment Worn` line:

```go
	// Conviction reserved to keep this companion fielded, snapshotted at summon
	// time so it doesn't drift when the summoner's skill/mutation changes mid-life.
	ConvictionReserve int `yaml:"conviction_reserve,omitempty"`
```

In `spells.go`, after `SummonMinCorpsePool  int    `yaml:"summon_min_corpse_pool,omitempty"``:

```go
	SummonConvictionReserve int    `yaml:"summon_conviction_reserve,omitempty"` // Ongoing Conviction reserved to maintain this companion (per summon type). 0 = use CompanionReserveDefault.
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/characters/ -run TestCompanionInfo_ConvictionReserveField`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/companions.go internal/characters/companions_test.go internal/spells/spells.go
git commit -m "feat(companions): add ConvictionReserve + SummonConvictionReserve fields"
```

---

### Task 2: Config knobs

**Files:**
- Modify: `internal/configs/config.balance.go` (Balance struct + defaults path)
- Test: whichever `_test.go` in `internal/configs` asserts defaults (grep `ManifestStatScaleChaFactor` in tests; if none exists, add a minimal one)

- [ ] **Step 1: Write the failing test**

Add (new file `internal/configs/config.balance.companions_test.go`):

```go
package configs

import "testing"

func TestCompanionReserveDefaults(t *testing.T) {
	b := GetBalanceConfig()
	if float64(b.CompanionReserveSkillPct) != 0.01 {
		t.Fatalf("CompanionReserveSkillPct = %v, want 0.01", float64(b.CompanionReserveSkillPct))
	}
	if float64(b.CompanionReserveSkillCap) != 0.55 {
		t.Fatalf("CompanionReserveSkillCap = %v, want 0.55", float64(b.CompanionReserveSkillCap))
	}
	if float64(b.CompanionReserveMutPctPerRank) != 0.06 {
		t.Fatalf("CompanionReserveMutPctPerRank = %v, want 0.06", float64(b.CompanionReserveMutPctPerRank))
	}
	if float64(b.CompanionReserveMutCap) != 0.24 {
		t.Fatalf("CompanionReserveMutCap = %v, want 0.24", float64(b.CompanionReserveMutCap))
	}
	if float64(b.CompanionReserveTotalCap) != 0.79 {
		t.Fatalf("CompanionReserveTotalCap = %v, want 0.79", float64(b.CompanionReserveTotalCap))
	}
	if int(b.CompanionSoftCap) != 5 {
		t.Fatalf("CompanionSoftCap = %v, want 5", int(b.CompanionSoftCap))
	}
	if int(b.CompanionSoftCapApex) != 7 {
		t.Fatalf("CompanionSoftCapApex = %v, want 7", int(b.CompanionSoftCapApex))
	}
	if int(b.CompanionReserveDefault) != 350 {
		t.Fatalf("CompanionReserveDefault = %v, want 350", int(b.CompanionReserveDefault))
	}
	if float64(b.CompanionCastingFloorPct) != 0.0 {
		t.Fatalf("CompanionCastingFloorPct = %v, want 0.0", float64(b.CompanionCastingFloorPct))
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/configs/ -run TestCompanionReserveDefaults`
Expected: FAIL — unknown fields.

- [ ] **Step 3: Add the fields + defaults**

In `config.balance.go`, in the `Balance` struct near `ManifestStatScaleSkillFactor`, matching neighbor field types (use the float type of `ManifestStatScaleSkillFactor` for the pct/cap floats, and the int type of an int neighbor for the counts/default):

```go
	CompanionReserveSkillPct      ConfigFloat `yaml:"CompanionReserveSkillPct"`      // Reduction per manifestation skill point
	CompanionReserveSkillCap      ConfigFloat `yaml:"CompanionReserveSkillCap"`      // Max reduction from skill
	CompanionReserveMutPctPerRank ConfigFloat `yaml:"CompanionReserveMutPctPerRank"` // Reduction per Manifester mutation rank
	CompanionReserveMutCap        ConfigFloat `yaml:"CompanionReserveMutCap"`        // Max reduction from the mutation
	CompanionReserveTotalCap      ConfigFloat `yaml:"CompanionReserveTotalCap"`      // Combined reduction ceiling
	CompanionSoftCap              ConfigInt   `yaml:"CompanionSoftCap"`              // Soft count backstop (real limit is the Conviction budget)
	CompanionSoftCapApex          ConfigInt   `yaml:"CompanionSoftCapApex"`          // Soft count backstop with the Manifester apex
	CompanionReserveDefault       ConfigInt   `yaml:"CompanionReserveDefault"`       // Fallback reserve when a summon spell omits summon_conviction_reserve
	CompanionCastingFloorPct      ConfigFloat `yaml:"CompanionCastingFloorPct"`      // Conviction fraction kept unreservable as casting headroom
```

> Match the ACTUAL neighbor types — if `ManifestStatScaleSkillFactor` is a bare `float64`/`int` or a `ConfigFloat`/`ConfigInt` wrapper, copy that exactly. The test's `float64(...)`/`int(...)` casts work either way.

In the defaults path (where `ManifestStatScaleChaFactor` gets its default — same file), add:

```go
	// Companion Conviction economy
	if b.CompanionReserveSkillPct == 0 { b.CompanionReserveSkillPct = 0.01 }
	if b.CompanionReserveSkillCap == 0 { b.CompanionReserveSkillCap = 0.55 }
	if b.CompanionReserveMutPctPerRank == 0 { b.CompanionReserveMutPctPerRank = 0.06 }
	if b.CompanionReserveMutCap == 0 { b.CompanionReserveMutCap = 0.24 }
	if b.CompanionReserveTotalCap == 0 { b.CompanionReserveTotalCap = 0.79 }
	if b.CompanionSoftCap == 0 { b.CompanionSoftCap = 5 }
	if b.CompanionSoftCapApex == 0 { b.CompanionSoftCapApex = 7 }
	if b.CompanionReserveDefault == 0 { b.CompanionReserveDefault = 350 }
	// CompanionCastingFloorPct intentionally defaults to 0.0 (costs self-limit).
```

> Follow the file's actual default idiom — if it uses a `validate()`/`SetDefaults()` with a different pattern (e.g. struct-literal defaults), match that instead of the `if == 0` guards.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/configs/ -run TestCompanionReserveDefaults`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/configs/
git commit -m "feat(companions): add Conviction-economy config knobs"
```

---

### Task 3: Manifester mutation reducer — `GetCompanionReserveRank`

**Files:**
- Modify: `internal/mutations/mutations.go` (near `GetMutationFlags`)
- Modify: `internal/mutations/describe.go` (DescribeEffect switch)
- Test: `internal/mutations/mutations_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestGetCompanionReserveRank(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"broodmaster": {
			MutationId: "broodmaster", Name: "Broodmaster", Rarity: 5, Pole: "belief",
			Pros: []MutationEffect{{Type: "companion_reserve_reduction"}},
		},
		"claws": {MutationId: "claws", Name: "Claws", Rarity: 3, Pole: "body"},
	})
	defer cleanup()

	if r := GetCompanionReserveRank(map[string]int{}); r != 0 {
		t.Fatalf("no mutations -> rank 0, got %d", r)
	}
	if r := GetCompanionReserveRank(map[string]int{"claws": 4}); r != 0 {
		t.Fatalf("no reducer mutation -> rank 0, got %d", r)
	}
	if r := GetCompanionReserveRank(map[string]int{"broodmaster": 3}); r != 3 {
		t.Fatalf("reducer at rank 3 -> 3, got %d", r)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/mutations/ -run TestGetCompanionReserveRank`
Expected: FAIL — undefined `GetCompanionReserveRank`.

- [ ] **Step 3: Implement**

In `mutations.go`, after `HasMutationFlag`:

```go
// GetCompanionReserveRank returns the highest owned rank among mutations that
// carry a "companion_reserve_reduction" effect (0 if none). The reduction
// magnitude is computed by the caller from config knobs (linear per rank) —
// NOT via sumEffects — because companion reservation reduction is a
// character-economy knob, not a stat effect scaled by the mutation-level curve.
func GetCompanionReserveRank(owned map[string]int) int {
	rank := 0
	for id, level := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		for _, p := range spec.Pros {
			if p.Type == "companion_reserve_reduction" && level > rank {
				rank = level
			}
		}
	}
	return rank
}
```

In `describe.go`, add a case to the `DescribeEffect` switch (before `case "flag":`):

```go
	case "companion_reserve_reduction":
		return "Eases the conviction you must devote to sustaining your companions, letting you field more of them."
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/mutations/ -run 'TestGetCompanionReserveRank|TestDescribeEffect'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/mutations/
git commit -m "feat(companions): Manifester companion_reserve_reduction reader + description"
```

---

### Task 4: Reduction formula — `Character.CalcCompanionReserve`

**Files:**
- Modify: `internal/characters/companions.go`
- Test: `internal/characters/companions_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestCalcCompanionReserve_Calibration(t *testing.T) {
	seedMut := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"broodmaster": {
			MutationId: "broodmaster", Name: "Broodmaster", Rarity: 5, Pole: "belief",
			Pros: []mutations.MutationEffect{{Type: "companion_reserve_reduction"}},
		},
	})
	defer seedMut()

	newbie := New()
	newbie.Skills[string(skills.Manifestation)] = 5
	assert.Equal(t, 333, newbie.CalcCompanionReserve(350)) // 350*(1-0.05)=332.5 -> 333

	meirok := New()
	meirok.Skills[string(skills.Manifestation)] = 48
	assert.Equal(t, 229, meirok.CalcCompanionReserve(440)) // 440*(1-0.48)=228.8 -> 229

	archety := New()
	archety.Skills[string(skills.Manifestation)] = 55
	archety.Mutations = map[string]int{"broodmaster": 4}
	assert.Equal(t, 92, archety.CalcCompanionReserve(440))  // 440*0.21 = 92.4 -> 92
	assert.Equal(t, 154, archety.CalcCompanionReserve(735)) // 735*0.21 = 154.35 -> 154

	unit := New()
	unit.Skills[string(skills.Manifestation)] = 65
	unit.Mutations = map[string]int{"broodmaster": 4}
	assert.Equal(t, 154, unit.CalcCompanionReserve(735)) // reduction caps at 0.79 -> same as archetyped
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/characters/ -run TestCalcCompanionReserve_Calibration`
Expected: FAIL — undefined `CalcCompanionReserve`.

- [ ] **Step 3: Implement**

In `companions.go`, add (`math`, `configs`, `skills`, `mutations` are available in the package):

```go
// CalcCompanionReserve returns the Conviction a companion of the given base
// cost reserves for THIS summoner, after manifestation-skill and Manifester-
// mutation reductions. reservation = round(base * (1 - reduction)).
func (c *Character) CalcCompanionReserve(baseCost int) int {
	cfg := configs.GetBalanceConfig()
	manif := c.GetSkillLevel(skills.Manifestation)
	manifRed := math.Min(float64(cfg.CompanionReserveSkillCap), float64(manif)*float64(cfg.CompanionReserveSkillPct))
	mutRank := mutations.GetCompanionReserveRank(c.Mutations)
	mutRed := math.Min(float64(cfg.CompanionReserveMutCap), float64(mutRank)*float64(cfg.CompanionReserveMutPctPerRank))
	reduction := math.Min(float64(cfg.CompanionReserveTotalCap), manifRed+mutRed)
	return int(math.Round(float64(baseCost) * (1.0 - reduction)))
}
```

> If `internal/mutations` is not yet imported in `companions.go`, add it to the import block. It cannot create a cycle — `mutations` does not import `characters`.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/characters/ -run TestCalcCompanionReserve_Calibration`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/companions.go internal/characters/companions_test.go
git commit -m "feat(companions): CalcCompanionReserve reduction formula"
```

---

### Task 5: Sum companion reserves into `GetPoolReservation`

**Files:**
- Modify: `internal/characters/validate.go` (GetPoolReservation, ~line 241-268)
- Test: `internal/characters/validate_test.go` (create if absent)

- [ ] **Step 1: Write the failing test**

```go
func TestGetPoolReservation_IncludesCompanions(t *testing.T) {
	c := New()
	c.Skills[string(skills.Manifestation)] = 19
	c.RecalculateStats()
	base := c.GetPoolReservation("conviction", c.ConvictionMax.Value)

	require.True(t, c.AddCompanion(CompanionInfo{InstanceId: 1, Name: "A", ConvictionReserve: 100}))
	require.True(t, c.AddCompanion(CompanionInfo{InstanceId: 2, Name: "B", ConvictionReserve: 60}))

	got := c.GetPoolReservation("conviction", c.ConvictionMax.Value)
	assert.Equal(t, base+160, got)

	// Companions never affect non-conviction pools.
	assert.Equal(t, 0, c.GetPoolReservation("stamina", c.StaminaMax.Value))
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/characters/ -run TestGetPoolReservation_IncludesCompanions`
Expected: FAIL — companions not counted (got == base).

- [ ] **Step 3: Implement**

In `GetPoolReservation`, immediately before `return total`:

```go
	// Companions reserve Conviction while fielded (snapshotted at summon time).
	if pool == "conviction" {
		for i := range c.Companions {
			total += c.Companions[i].ConvictionReserve
		}
	}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/characters/ -run TestGetPoolReservation_IncludesCompanions`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/validate.go internal/characters/validate_test.go
git commit -m "feat(companions): sum companion reserves into conviction pool reservation"
```

---

### Task 6: `GetMaxCompanions` → soft backstop

**Files:**
- Modify: `internal/characters/companions.go` (GetMaxCompanions ~line 130; `hasManifestationSpell` may become unused — remove it if so)
- Modify: `internal/characters/companions_test.go` (rewrite `TestGetMaxCompanions_Ranks`)

- [ ] **Step 1: Rewrite the test**

Replace `TestGetMaxCompanions_Ranks` with:

```go
func TestGetMaxCompanions_SoftBackstop(t *testing.T) {
	seedMut := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"broodqueen": {
			MutationId: "broodqueen", Name: "Brood Queen", Rarity: 8, Pole: "belief",
			Pros: []mutations.MutationEffect{{Type: "flag", Target: "companion-cap-raise"}},
		},
	})
	defer seedMut()

	// No manifestation investment still returns the soft cap — the real gate is
	// the Conviction budget (CanAffordCompanion), not this count.
	c := New()
	assert.Equal(t, 5, c.GetMaxCompanions())

	// The apex flag raises the backstop.
	c.Mutations = map[string]int{"broodqueen": 1}
	assert.Equal(t, 7, c.GetMaxCompanions())
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/characters/ -run TestGetMaxCompanions_SoftBackstop`
Expected: FAIL — old formula returns 0.

- [ ] **Step 3: Implement**

Replace `GetMaxCompanions` with:

```go
// GetMaxCompanions returns the SOFT count backstop on simultaneous companions.
// It is a safety limit only — the real constraint is the Conviction budget
// (see CanAffordCompanion). The Manifester apex ("companion-cap-raise" flag)
// raises the backstop.
func (c *Character) GetMaxCompanions() int {
	cfg := configs.GetBalanceConfig()
	cap := int(cfg.CompanionSoftCap)
	if cap < 1 {
		cap = 5
	}
	if mutations.HasMutationFlag(c.Mutations, "companion-cap-raise") {
		if apex := int(cfg.CompanionSoftCapApex); apex > cap {
			cap = apex
		}
	}
	return cap
}
```

Delete `hasManifestationSpell` if it is now unreferenced (compiler will flag it). Verify with `go build ./...` after this step.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/characters/ -run TestGetMaxCompanions_SoftBackstop` then `go build ./...`
Expected: PASS; build clean.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/companions.go internal/characters/companions_test.go
git commit -m "refactor(companions): GetMaxCompanions becomes soft backstop (budget is the real cap)"
```

---

### Task 7: Budget gate — `Character.CanAffordCompanion`

**Files:**
- Modify: `internal/characters/companions.go`
- Test: `internal/characters/companions_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestCanAffordCompanion_Budget(t *testing.T) {
	// Newbie: ConvictionMax ~440, wolf reserves 333 -> affords 1, not a 2nd.
	c := New()
	c.ConvictionMax.Value = 440
	assert.True(t, c.CanAffordCompanion(333))
	c.AddCompanion(CompanionInfo{InstanceId: 1, Name: "Wolf", ConvictionReserve: 333})
	assert.False(t, c.CanAffordCompanion(333)) // 333+333=666 > 440

	// Meirok: ConvictionMax ~547, golem reserves 229 -> affords 2, not 3.
	m := New()
	m.ConvictionMax.Value = 547
	m.AddCompanion(CompanionInfo{InstanceId: 1, Name: "G1", ConvictionReserve: 229})
	assert.True(t, m.CanAffordCompanion(229))  // 229+229=458 <= 547
	m.AddCompanion(CompanionInfo{InstanceId: 2, Name: "G2", ConvictionReserve: 229})
	assert.False(t, m.CanAffordCompanion(229)) // 458+229=687 > 547

	// Soft backstop still bites even if budget allows (cheap pets).
	s := New()
	s.ConvictionMax.Value = 100000
	for i := 0; i < 5; i++ {
		s.AddCompanion(CompanionInfo{InstanceId: i + 1, Name: "x", ConvictionReserve: 1})
	}
	assert.False(t, s.CanAffordCompanion(1)) // at soft cap 5
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/characters/ -run TestCanAffordCompanion_Budget`
Expected: FAIL — undefined `CanAffordCompanion`.

- [ ] **Step 3: Implement**

In `companions.go`:

```go
// CanAffordCompanion reports whether the character can field one more companion
// reserving `reserveCost` Conviction: the new total reservation (plus any
// casting floor) must fit within ConvictionMax, and the soft count backstop
// must not be exceeded.
func (c *Character) CanAffordCompanion(reserveCost int) bool {
	if len(c.Companions) >= c.GetMaxCompanions() {
		return false
	}
	cfg := configs.GetBalanceConfig()
	current := c.GetPoolReservation("conviction", c.ConvictionMax.Value)
	floor := int(math.Round(float64(c.ConvictionMax.Value) * float64(cfg.CompanionCastingFloorPct)))
	return current+reserveCost+floor <= c.ConvictionMax.Value
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/characters/ -run TestCanAffordCompanion_Budget`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/companions.go internal/characters/companions_test.go
git commit -m "feat(companions): CanAffordCompanion budget gate"
```

---

### Task 8: Wire the summon path

**Files:**
- Modify: `internal/hooks/companion_summon.go` (`resolveCompanionSummon`)
- Test: manual smoke (below) — this path needs a live room/mob; keep automated coverage on the helpers (Tasks 4/7) and verify integration at boot.

- [ ] **Step 1: Replace the count gate with a budget gate + compute the reserve**

Replace lines 37-41 (the `── 1. Companion cap` block, at the TOP of the function before component/corpse consumption so we fail early) with a reserve computation and a budget check. The base cost comes from the spell, so no pool/stat info is needed here:

```go
	// ── 1. Reservation + budget gate ────────────────────────────────────
	baseReserve := spellData.SummonConvictionReserve
	if baseReserve <= 0 {
		baseReserve = int(configs.GetBalanceConfig().CompanionReserveDefault)
	}
	reserve := ch.CalcCompanionReserve(baseReserve)
	if !ch.CanAffordCompanion(reserve) {
		user.SendText(messaging.CategorySpellManifestation,
			"You cannot spare the conviction to bind another companion.")
		return false
	}
```

> Add `configs` to the file's import block if absent.

- [ ] **Step 2: Snapshot the reserve on the CompanionInfo**

At the `info := characters.CompanionInfo{…}` literal (lines 150-157), add the field:

```go
	info := characters.CompanionInfo{
		MobId:             int(mob.MobId),
		InstanceId:        mob.InstanceId,
		SourceType:        sourceType,
		Name:              mob.Character.Name,
		BaseName:          mob.Character.Name,
		AutoAssist:        true,
		ConvictionReserve: reserve,
	}
```

- [ ] **Step 3: Recalculate after adding so the reservation clamps usable Conviction now**

Immediately after the successful `ch.AddCompanion(info)` block, add:

```go
	ch.RecalculateStats()
```

- [ ] **Step 4: Build + boot smoke**

Run: `go build ./...`

Then nuke instance saves (per CLAUDE.md SOP) and boot to confirm no panic:

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Boot the server (`go run .`), watch for clean `spells.LoadDataFiles()` / no panic, then quit.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/companion_summon.go
git commit -m "feat(companions): gate summoning on the Conviction budget, snapshot reservation"
```

---

### Task 9: Wire the charm + btree summon paths

**Files:**
- Modify: charm path — find with codegraph (`codegraph_node resolveCharmSpell`); it calls `GetMaxCompanions` + `AddCompanion`.
- Modify: btree path — find with codegraph (`codegraph_node actSummonCompanion`); it calls `AddCompanion`.
- Test: build + smoke.

- [ ] **Step 1: Charm — gate + snapshot**

In `resolveCharmSpell`, mirror Task 8: compute `baseReserve` from the charm spell's `SummonConvictionReserve` (fallback `CompanionReserveDefault`), `reserve := ch.CalcCompanionReserve(baseReserve)`, replace the `GetMaxCompanions` count check with `if !ch.CanAffordCompanion(reserve) { …refuse… }`, set `ConvictionReserve: reserve` on the `CompanionInfo`, and `ch.RecalculateStats()` after add. Keep the existing charm-duration/reroll logic untouched.

- [ ] **Step 2: Btree `actSummonCompanion` — gate + snapshot**

Same pattern for the mob-driven summon. The actor is a mob (`*mobs.Mob`); use `mob.Character.CalcCompanionReserve` / `mob.Character.CanAffordCompanion` / `mob.Character.RecalculateStats()`. If the mob has no Manifester investment the reduction is simply 0 — the math is identical. If the surrounding btree code has no clean spot for the refuse-message, just skip the summon (return failure) when `!CanAffordCompanion`.

- [ ] **Step 3: Build + smoke**

Run: `go build ./...` then boot-smoke as in Task 8 Step 4.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat(companions): apply Conviction budget gate to charm + btree summon paths"
```

---

### Task 10: Content — set per-type reserves on summon/raise/charm spells

**Files:**
- Modify: every summon/raise/charm spell YAML under `_datafiles/world/dogmud/spells/`

- [ ] **Step 1: Enumerate the spells**

Grep the spell YAML dir for `summon_mob_id` and for charm effect spells (`effect_type: charm`). List each with its summoned creature.

- [ ] **Step 2: Assign a tier per spell** using the spec §3 table:
  - **Lesser (350):** spirit wolf, conjured imp, and any low-end summon.
  - **Greater (440):** magma/elemental summons, standard golem raises.
  - **Elder (735):** elite/high-end summons.
  - Charm spells: start at **440** (Greater) unless the charm is clearly weak (then 350); tune in playtest.

Add `summon_conviction_reserve: <N>` to each spell file. Where a summon is genuinely trivial and should be near-free, you may set a low value — but never omit it silently expecting free (0 → falls back to `CompanionReserveDefault` 350, which is intentional).

- [ ] **Step 3: Boot smoke**

Nuke instance saves, boot, confirm `spells.LoadDataFiles()` loads clean (no YAML panic), quit.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/spells/
git commit -m "content(companions): assign summon_conviction_reserve per summon tier"
```

---

### Task 11: Calibration acceptance test

**Files:**
- Test: `internal/characters/companion_calibration_test.go` (new)

- [ ] **Step 1: Write the acceptance test** — encodes the spec §5 rows end-to-end (reserve → budget), so a future knob change that breaks the intended progression fails loudly.

```go
package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/stretchr/testify/assert"
)

func TestCompanionEconomy_Calibration(t *testing.T) {
	seedMut := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"broodmaster": {
			MutationId: "broodmaster", Name: "Broodmaster", Rarity: 5, Pole: "belief",
			Pros: []mutations.MutationEffect{{Type: "companion_reserve_reduction"}},
		},
	})
	defer seedMut()

	fieldN := func(c *Character, base, convMax int) int {
		c.ConvictionMax.Value = convMax
		c.Companions = nil
		n := 0
		for {
			r := c.CalcCompanionReserve(base)
			if !c.CanAffordCompanion(r) {
				break
			}
			n++
			c.AddCompanion(CompanionInfo{InstanceId: n, Name: "p", ConvictionReserve: r})
		}
		return n
	}

	// Newbie: manif 5, Lesser 350, ConvMax 440 -> exactly 1 wolf.
	nb := New()
	nb.Skills[string(skills.Manifestation)] = 5
	assert.Equal(t, 1, fieldN(nb, 350, 440), "newbie fields exactly 1 spirit wolf")

	// Meirok: manif 48, no mutation, Greater 440, ConvMax 547 -> 2 golems.
	me := New()
	me.Skills[string(skills.Manifestation)] = 48
	assert.Equal(t, 2, fieldN(me, 440, 547), "Meirok fields 2 greater golems, not 3")

	// Fully archetyped: manif 55 + rank-4 mutation, ConvMax 600.
	ar := New()
	ar.Skills[string(skills.Manifestation)] = 55
	ar.Mutations = map[string]int{"broodmaster": 4}
	assert.Equal(t, 5, fieldN(ar, 440, 600), "archetyped fields 5 greater golems")
	assert.Equal(t, 3, fieldN(ar, 735, 600), "archetyped fields 3 elder golems")

	// Absolute unit: manif 65 + rank-4, ConvMax 850 -> 5 elder golems.
	un := New()
	un.Skills[string(skills.Manifestation)] = 65
	un.Mutations = map[string]int{"broodmaster": 4}
	assert.Equal(t, 5, fieldN(un, 735, 850), "absolute unit fields 5 elder golems")
}
```

> The archetyped "5 greater" row depends on `CompanionSoftCap = 5` (budget alone would allow 6 at 92 each). That's intended — the soft cap is doing its backstop job exactly here. If a reviewer objects that the budget should be the sole limiter, note it: at rank-4 reduction the greater-golem cost is low enough that the count cap binds first, which is the designed safety.

- [ ] **Step 2: Run**

Run: `go test ./internal/characters/ -run TestCompanionEconomy_Calibration -v`
Expected: PASS. If a row is off by one, adjust the spec's base costs / knobs FIRST (tuning surface), then re-run — do not fudge the test.

- [ ] **Step 3: Commit**

```bash
git add internal/characters/companion_calibration_test.go
git commit -m "test(companions): calibration acceptance test for the Conviction economy"
```

---

### Task 12: Patch notes + full boot verification

**Files:**
- Modify: `PATCH_NOTES.md`

- [ ] **Step 1: Add a dated PATCH_NOTES entry** describing the change in player terms: manifestation companions now reserve Conviction (powerful pets reserve more; manifestation skill and the Manifester mutation reduce the cost), so fielding pets trades against casting. Note the count cap is gone in favor of the budget.

- [ ] **Step 2: Full suite**

Run: `go test ./...`
Expected: PASS (watch `internal/characters`, `internal/mutations`, `internal/configs`, `internal/hooks`).

- [ ] **Step 3: Full boot smoke**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Boot `go run .`, confirm clean load through `spells.LoadDataFiles()` / `mobs.LoadDataFiles()` with no panic, quit.

- [ ] **Step 4: Commit**

```bash
git add PATCH_NOTES.md
git commit -m "docs(companions): patch notes for the Conviction economy"
```

---

## Out of scope (follow-on)

- **Manifester cluster content** (the actual `broodmaster` / apex mutations with the `companion_reserve_reduction` and `companion-cap-raise` flag) lands in mutation **Wave 6** — this plan only builds the engine seams they plug into. Until that content exists the reducer/apex are inert (rank 0, flag absent), which is correct and backward-compatible.
- **Tier 4+ companions** (ghost/lich/undead dragon) — reserved, per spec §10.
- **Per-pool reservation** (undead→Health) — rejected, per spec §10b.
- **`companion` command display** of reservation (number-free) — nice-to-have, not required for the economy to function.
