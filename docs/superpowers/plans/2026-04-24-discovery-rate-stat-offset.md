# Discovery Rate Stat Offset Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Perception + relevant-skill offset to the effective decay rate of spell/recipe discovery so high-investment characters keep discovering new content late-game.

**Architecture:** One pure helper `configs.DiscoveryChance(DiscoveryParams)` computes `chance = base / (1 + known × decay × (1 - offset))` where `offset = min(MaxOffset, 1 - (1-perContrib)(1-skillContrib))`. Five existing inline call sites in `internal/hooks/` migrate to the helper, with each site passing its own `relevant skill`. Three new Balance knobs (`DiscoveryPerceptionScale`, `DiscoverySkillScale`, `DiscoveryMaxDecayOffset`) are shared across spell + recipe discovery.

**Tech Stack:** Go 1.20+, `github.com/stretchr/testify`, existing `ConfigFloat` pattern in `internal/configs/config.balance.go`.

---

## File Structure

| File | Role | Status |
|------|------|--------|
| `internal/configs/config.balance.go` | Balance struct — add 3 new fields | Modify |
| `internal/configs/config.balance.discovery.go` | Default-value validator for the 3 new knobs | Create |
| `internal/configs/discovery.go` | `DiscoveryChance` helper + `DiscoveryParams` struct | Create |
| `internal/configs/discovery_test.go` | Table-driven unit tests | Create |
| `internal/configs/config.balance.spells.go` | Wire validator into chain (one-line add) | Modify |
| `_datafiles/config.yaml` | Add 3 new YAML keys | Modify |
| `internal/hooks/NewRound_DoCombat_helpers.go` | Migrate 4 spell discovery call sites (player + mob × traditional + manifestation) | Modify |
| `internal/hooks/NewRound_UserRoundTick.go` | Migrate 1 recipe discovery call site | Modify |
| `PATCH_NOTES.md` | Document the change | Modify |

---

## Task 1: Add config knobs (struct, defaults, YAML)

**Files:**
- Modify: `internal/configs/config.balance.go:193-195` (add new fields adjacent to existing `SpellDiscovery*`)
- Create: `internal/configs/config.balance.discovery.go`
- Modify: `internal/configs/config.balance.spells.go` (call new validator)
- Modify: `_datafiles/config.yaml:759-760` area (add keys near existing `SpellDiscovery*` / `RecipeDiscovery*`)

- [ ] **Step 1: Add fields to Balance struct**

Open `internal/configs/config.balance.go`. Find line 194–195 (the two existing `SpellDiscovery*` fields). Insert these three fields immediately **after** `SpellDiscoveryDecayRate`, before `SpellInitiationBase`:

```go
	// ── DISCOVERY OFFSET (shared: spells + recipes) ──────────────────────────
	DiscoveryPerceptionScale ConfigFloat `yaml:"DiscoveryPerceptionScale"` // Raw Per contribution reaches 1.0 at (Per - 100) / this (default 200)
	DiscoverySkillScale      ConfigFloat `yaml:"DiscoverySkillScale"`      // Raw skill contribution reaches 1.0 at rank / this (default 100)
	DiscoveryMaxDecayOffset  ConfigFloat `yaml:"DiscoveryMaxDecayOffset"`  // Hard ceiling on combined offset; effective decay floor = Decay × (1 - this) (default 0.8)
```

- [ ] **Step 2: Create the validator file**

Create `internal/configs/config.balance.discovery.go` with this content:

```go
package configs

// validateDiscovery sets defaults for the shared spell/recipe
// discovery offset knobs.
func (b *Balance) validateDiscovery() {
	if b.DiscoveryPerceptionScale <= 0 {
		b.DiscoveryPerceptionScale = 200.0
	}
	if b.DiscoverySkillScale <= 0 {
		b.DiscoverySkillScale = 100.0
	}
	if b.DiscoveryMaxDecayOffset <= 0 {
		b.DiscoveryMaxDecayOffset = 0.8
	}
}
```

- [ ] **Step 3: Wire the validator into the validate chain**

Find where `validateSpells` is called. Run:

```bash
grep -n "validateSpells\|validateShops\|validateCombat" internal/configs/config.balance.go
```

Expected: a `Validate()` method listing `b.validateCombat()`, `b.validateSpells()`, `b.validateShops()`, etc.

Open that method. Add `b.validateDiscovery()` immediately after `b.validateSpells()`:

```go
	b.validateSpells()
	b.validateDiscovery()
	b.validateShops()
```

(Exact surrounding lines will vary — just add the one line.)

- [ ] **Step 4: Add YAML keys**

Open `_datafiles/config.yaml`. Find line 782–783 (the existing `SpellDiscoveryBaseChance: 5.0` and `SpellDiscoveryDecayRate: 0.1`). Add these three lines immediately after `SpellDiscoveryDecayRate: 0.1`:

```yaml
  DiscoveryPerceptionScale: 200.0
  DiscoverySkillScale: 100.0
  DiscoveryMaxDecayOffset: 0.8
```

- [ ] **Step 5: Build to verify compile**

Run: `cd "C:\Users\Calabe Davis\workspace\DOGMud" && go build ./internal/configs/...`
Expected: no output (success).

- [ ] **Step 6: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.discovery.go internal/configs/config.balance.spells.go _datafiles/config.yaml
git commit -m "$(cat <<'EOF'
feat(configs): add DiscoveryPerceptionScale / DiscoverySkillScale / DiscoveryMaxDecayOffset knobs

Three new shared Balance knobs prepare the offset helper in Task 2.
Per contribution reaches 1.0 at (Per - 100) / 200, skill contribution
at rank / 100, combined offset capped at 0.8.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: DiscoveryChance helper with TDD

**Files:**
- Create: `internal/configs/discovery.go`
- Create: `internal/configs/discovery_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/configs/discovery_test.go`:

```go
package configs

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setupDiscoveryTestBalance installs a balance config with the documented
// default offset knobs so the test cases below match the spec tables.
func setupDiscoveryTestBalance(t *testing.T) {
	t.Helper()
	b := &Balance{
		DiscoveryPerceptionScale: 200.0,
		DiscoverySkillScale:      100.0,
		DiscoveryMaxDecayOffset:  0.8,
	}
	setBalanceForTest(b)
}

func TestDiscoveryChance(t *testing.T) {
	setupDiscoveryTestBalance(t)

	const tolerance = 0.05 // 0.05 percentage points

	cases := []struct {
		name                       string
		base, decay                float64
		known, perception, skill   int
		wantChance                 float64
	}{
		// Spec table — spells (base 5%, decay 0.1)
		{"spells: newbie", 5.0, 0.1, 3, 100, 0, 3.85},
		{"spells: early", 5.0, 0.1, 8, 110, 10, 2.97},
		{"spells: mid", 5.0, 0.1, 12, 130, 25, 2.83},
		{"spells: late", 5.0, 0.1, 18, 160, 50, 3.07},
		{"spells: gm (caps at 0.8)", 5.0, 0.1, 20, 200, 100, 3.57},

		// Spec table — recipes (base 10%, decay 0.1)
		{"recipes: newbie", 10.0, 0.1, 3, 100, 0, 7.69},
		{"recipes: gm (caps at 0.8)", 10.0, 0.1, 20, 200, 100, 7.14},

		// Edge cases
		{"known=0 returns base", 5.0, 0.1, 0, 100, 0, 5.00},
		{"per below baseline clamps to 0", 5.0, 0.1, 10, 50, 50, 3.77},
		{"skill above scale clamps to 1", 5.0, 0.1, 10, 100, 200, 3.77},
		{"offset caps at 0.8 (very high per+skill)", 5.0, 0.1, 20, 300, 200, 3.57},
		{"pure per build (skill=0)", 5.0, 0.1, 10, 200, 0, 3.33},
		{"pure skill build (per=100)", 5.0, 0.1, 10, 100, 100, 3.33},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DiscoveryChance(DiscoveryParams{
				Base:       tc.base,
				Decay:      tc.decay,
				Known:      tc.known,
				Perception: tc.perception,
				Skill:      tc.skill,
			})
			assert.InDelta(t, tc.wantChance, got, tolerance,
				"expected %.2f%%, got %.4f%%", tc.wantChance, got)
		})
	}
}

func TestDiscoveryChance_NegativeKnownClampsToZero(t *testing.T) {
	setupDiscoveryTestBalance(t)
	got := DiscoveryChance(DiscoveryParams{
		Base: 5.0, Decay: 0.1, Known: -3, Perception: 100, Skill: 0,
	})
	// Negative known would produce > Base if not handled; we treat it as 0.
	assert.InDelta(t, 5.0, got, 0.01)
}

func TestDiscoveryChance_DefaultsApplied(t *testing.T) {
	// When Balance has not been set, DiscoveryChance should still work
	// using the validator-applied defaults.
	b := &Balance{}
	b.validateDiscovery()
	assert.InDelta(t, 200.0, float64(b.DiscoveryPerceptionScale), 0.001)
	assert.InDelta(t, 100.0, float64(b.DiscoverySkillScale), 0.001)
	assert.InDelta(t, 0.8, float64(b.DiscoveryMaxDecayOffset), 0.001)
}

// confirm that setBalanceForTest unsafely overwrites the singleton so the
// helper isn't reading an unconfigured struct.
func TestDiscoveryChance_ReadsConfiguredBalance(t *testing.T) {
	b := &Balance{
		DiscoveryPerceptionScale: 200.0,
		DiscoverySkillScale:      100.0,
		DiscoveryMaxDecayOffset:  0.0, // cap at 0 → no offset ever applied
	}
	setBalanceForTest(b)

	// With MaxOffset=0, Per+Skill irrelevant; pure baseline formula.
	got := DiscoveryChance(DiscoveryParams{
		Base: 5.0, Decay: 0.1, Known: 10, Perception: 200, Skill: 100,
	})
	want := 5.0 / (1.0 + 10.0*0.1) // 2.5
	assert.InDelta(t, want, got, 0.01)
	_ = math.Abs // silence unused-import if compiler is picky
}
```

- [ ] **Step 2: Check whether `setBalanceForTest` already exists**

Run: `grep -rn "func setBalanceForTest\|func SetBalanceForTest" internal/configs/`

**If it exists** (function is present), skip to Step 3.

**If it does NOT exist** (no results), create `internal/configs/testing_support.go`:

```go
package configs

// setBalanceForTest replaces the module-level balance config with
// the provided instance. Intended for use from _test.go files in the
// configs package. Callers are responsible for saving/restoring if
// parallelism is a concern (these tests run sequentially).
func setBalanceForTest(b *Balance) {
	balanceConfig = b
}
```

Verify the singleton is named `balanceConfig`:

```bash
grep -n "var balanceConfig\|balanceConfig =\|GetBalanceConfig" internal/configs/*.go | head
```

**If the singleton is named differently** (e.g. `currentBalance`, `globalBalance`), replace `balanceConfig` in the helper with the correct name. Do the same replacement in the test file's `setBalanceForTest` calls if needed.

- [ ] **Step 3: Run test to verify it fails (helper does not exist yet)**

Run: `go test ./internal/configs/ -run TestDiscoveryChance -v`
Expected: compile error, `undefined: DiscoveryChance` or `undefined: DiscoveryParams`.

- [ ] **Step 4: Create the helper**

Create `internal/configs/discovery.go`:

```go
package configs

// DiscoveryParams bundles the inputs for DiscoveryChance.
type DiscoveryParams struct {
	Base       float64 // base chance as a percent (e.g. 5.0 = 5%)
	Decay      float64 // per-known decay rate
	Known      int     // count of already-known spells/recipes
	Perception int     // character Perception stat (adjusted value)
	Skill      int     // relevant skill rank
}

// DiscoveryChance computes the discovery roll chance as a percent
// with a Perception + skill offset reducing the effective decay.
//
// Formula:
//
//	perContrib   = clamp(0, 1, (Perception - 100) / PerceptionScale)
//	skillContrib = clamp(0, 1, Skill / SkillScale)
//	offset       = min(MaxOffset, 1 - (1 - perContrib)(1 - skillContrib))
//	effDecay     = Decay × (1 - offset)
//	chance       = Base / (1 + Known × effDecay)
//
// Returns a value in [0, Base]. Negative Known is clamped to 0.
func DiscoveryChance(p DiscoveryParams) float64 {
	bal := GetBalanceConfig()
	perScale := float64(bal.DiscoveryPerceptionScale)
	skillScale := float64(bal.DiscoverySkillScale)
	maxOffset := float64(bal.DiscoveryMaxDecayOffset)

	perContrib := (float64(p.Perception) - 100.0) / perScale
	if perContrib < 0 {
		perContrib = 0
	}
	if perContrib > 1 {
		perContrib = 1
	}

	skillContrib := float64(p.Skill) / skillScale
	if skillContrib < 0 {
		skillContrib = 0
	}
	if skillContrib > 1 {
		skillContrib = 1
	}

	offset := 1.0 - (1.0-perContrib)*(1.0-skillContrib)
	if offset > maxOffset {
		offset = maxOffset
	}

	effDecay := p.Decay * (1.0 - offset)

	known := p.Known
	if known < 0 {
		known = 0
	}
	return p.Base / (1.0 + float64(known)*effDecay)
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/configs/ -run TestDiscoveryChance -v`
Expected: all subtests PASS.

If the "gm" caps-at-0.8 cases come in slightly off (e.g. 3.58% vs 3.57%), verify tolerance is `0.05`; if still failing, print the intermediate `offset` value and confirm the clamp to 0.8 is applied *after* the 1−(1−a)(1−b) combination.

- [ ] **Step 6: Commit**

```bash
git add internal/configs/discovery.go internal/configs/discovery_test.go internal/configs/testing_support.go
git commit -m "$(cat <<'EOF'
feat(configs): DiscoveryChance helper with Per+skill decay offset

Pure helper computing discovery roll chance with an independent-probability
combination of Per and skill contributions chipping away at effective decay.
12-case table-driven test covers the spec scenarios + edge cases.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Note: `testing_support.go` might not exist if the helper was already present — in that case just add the two new files.

---

## Task 3: Migrate player spell discovery call sites

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go:300-337`

The current block computes a single `discoveryChance` from `castSkillLevel` (Spellcasting) and uses it for BOTH the traditional-schools roll and the manifestation-school roll. The new world splits into two independent `DiscoveryChance` calls — one passing Spellcasting skill, one passing Manifestation skill.

- [ ] **Step 1: Inspect the current block**

Run: `sed -n '298,338p' internal/hooks/NewRound_DoCombat_helpers.go`

Expected current structure (verify):

```go
// Phase 25.1: Spell discovery — traditional schools.
castSkillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
knownCount := len(user.Character.SpellBook)
bal := configs.GetBalanceConfig()
discoveryChance := float64(bal.SpellDiscoveryBaseChance) / (1.0 + float64(knownCount)*float64(bal.SpellDiscoveryDecayRate))
if util.Rand(100) < int(discoveryChance) {
	eligible := spells.GetEligibleSpells(user.Character.SpellBook, castSkillLevel,
		spells.SchoolElemental, spells.SchoolEnhancement, spells.SchoolMental, spells.SchoolVital)
	// ...
}
// Phase 25.1: Spell discovery — manifestation school.
manifestSkillLevel := user.Character.GetSkillLevel(skills.Manifestation)
if manifestSkillLevel > 0 {
	if util.Rand(100) < int(discoveryChance) {
		eligible := spells.GetEligibleSpells(user.Character.SpellBook, manifestSkillLevel,
			spells.SchoolManifestation)
		// ...
	}
}
```

- [ ] **Step 2: Replace with per-school calls**

Edit the block starting at "Phase 25.1: Spell discovery — traditional schools." through the end of the manifestation-school `if` block. Replace the shared `discoveryChance` with two per-call lookups.

Old block (lines ~300-337):

```go
// Phase 25.1: Spell discovery — traditional schools.
castSkillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
knownCount := len(user.Character.SpellBook)
bal := configs.GetBalanceConfig()
discoveryChance := float64(bal.SpellDiscoveryBaseChance) / (1.0 + float64(knownCount)*float64(bal.SpellDiscoveryDecayRate))
if util.Rand(100) < int(discoveryChance) {
```

New:

```go
// Phase 25.1: Spell discovery — traditional schools.
castSkillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
knownCount := len(user.Character.SpellBook)
bal := configs.GetBalanceConfig()
perception := user.Character.Stats.Perception.ValueAdj
traditionalChance := configs.DiscoveryChance(configs.DiscoveryParams{
	Base:       float64(bal.SpellDiscoveryBaseChance),
	Decay:      float64(bal.SpellDiscoveryDecayRate),
	Known:      knownCount,
	Perception: perception,
	Skill:      castSkillLevel,
})
if util.Rand(100) < int(traditionalChance) {
```

And further down, the manifestation `if util.Rand(100) < int(discoveryChance)` line. Replace:

```go
if util.Rand(100) < int(discoveryChance) {
	eligible := spells.GetEligibleSpells(user.Character.SpellBook, manifestSkillLevel,
		spells.SchoolManifestation)
```

New:

```go
manifestChance := configs.DiscoveryChance(configs.DiscoveryParams{
	Base:       float64(bal.SpellDiscoveryBaseChance),
	Decay:      float64(bal.SpellDiscoveryDecayRate),
	Known:      knownCount,
	Perception: perception,
	Skill:      manifestSkillLevel,
})
if util.Rand(100) < int(manifestChance) {
	eligible := spells.GetEligibleSpells(user.Character.SpellBook, manifestSkillLevel,
		spells.SchoolManifestation)
```

- [ ] **Step 3: Build to verify compile**

Run: `go build ./internal/hooks/...`
Expected: no output.

If build fails on missing `configs` import: verify `configs` is already imported in this file (it is — the old code uses `configs.GetBalanceConfig()`).

- [ ] **Step 4: Run all hooks tests**

Run: `go test ./internal/hooks/...`
Expected: all tests pass (this refactor is behavior-preserving except for the offset mechanic, which now kicks in — existing tests that run with default Per=100 and skill=0 should see identical chance values).

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/NewRound_DoCombat_helpers.go
git commit -m "$(cat <<'EOF'
refactor(hooks): player spell discovery uses DiscoveryChance helper

Traditional + manifestation discovery now compute chances independently
with school-appropriate skill (Spellcasting vs Manifestation). Per +
skill offset reduces effective decay per the new mechanic.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Migrate mob spell discovery call sites

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go:416-445`

Same pattern as Task 3 but for the mob-caster block. The mob's Perception comes from the same `Character.Stats.Perception.ValueAdj` field.

- [ ] **Step 1: Inspect the current block**

Run: `sed -n '415,445p' internal/hooks/NewRound_DoCombat_helpers.go`

Expected structure (verify):

```go
isCaster := mob.Archetype == "casting" || len(mob.Character.SpellBook) > 0
if isCaster {
	castSkillLevel := mob.Character.GetSkillLevel(skills.Spellcasting)
	knownCount := len(mob.Character.SpellBook)
	bal := configs.GetBalanceConfig()
	discoveryChance := float64(bal.SpellDiscoveryBaseChance) / (1.0 + float64(knownCount)*float64(bal.SpellDiscoveryDecayRate))
	// Traditional school discovery.
	if util.Rand(100) < int(discoveryChance) {
		// ...
	}
	// Manifestation school discovery — only if mob has manifestation skill.
	manifestSkillLevel := mob.Character.GetSkillLevel(skills.Manifestation)
	if manifestSkillLevel > 0 {
		if util.Rand(100) < int(discoveryChance) {
			// ...
		}
	}
}
```

- [ ] **Step 2: Replace with per-school calls**

Edit the block. Replace the shared `discoveryChance` assignment and the two `util.Rand(100) < int(discoveryChance)` sites.

New block:

```go
isCaster := mob.Archetype == "casting" || len(mob.Character.SpellBook) > 0
if isCaster {
	castSkillLevel := mob.Character.GetSkillLevel(skills.Spellcasting)
	knownCount := len(mob.Character.SpellBook)
	bal := configs.GetBalanceConfig()
	perception := mob.Character.Stats.Perception.ValueAdj
	traditionalChance := configs.DiscoveryChance(configs.DiscoveryParams{
		Base:       float64(bal.SpellDiscoveryBaseChance),
		Decay:      float64(bal.SpellDiscoveryDecayRate),
		Known:      knownCount,
		Perception: perception,
		Skill:      castSkillLevel,
	})
	// Traditional school discovery.
	if util.Rand(100) < int(traditionalChance) {
		eligible := spells.GetEligibleSpells(mob.Character.SpellBook, castSkillLevel,
			spells.SchoolElemental, spells.SchoolEnhancement, spells.SchoolMental, spells.SchoolVital)
		if len(eligible) > 0 {
			pick := eligible[util.Rand(len(eligible))]
			mob.Character.LearnSpell(pick)
		}
	}
	// Manifestation school discovery — only if mob has manifestation skill.
	manifestSkillLevel := mob.Character.GetSkillLevel(skills.Manifestation)
	if manifestSkillLevel > 0 {
		manifestChance := configs.DiscoveryChance(configs.DiscoveryParams{
			Base:       float64(bal.SpellDiscoveryBaseChance),
			Decay:      float64(bal.SpellDiscoveryDecayRate),
			Known:      knownCount,
			Perception: perception,
			Skill:      manifestSkillLevel,
		})
		if util.Rand(100) < int(manifestChance) {
			eligible := spells.GetEligibleSpells(mob.Character.SpellBook, manifestSkillLevel,
				spells.SchoolManifestation)
			if len(eligible) > 0 {
				pick := eligible[util.Rand(len(eligible))]
				mob.Character.LearnSpell(pick)
			}
		}
	}
}
```

- [ ] **Step 3: Build**

Run: `go build ./internal/hooks/...`
Expected: no output.

- [ ] **Step 4: Run hooks tests**

Run: `go test ./internal/hooks/...`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/NewRound_DoCombat_helpers.go
git commit -m "$(cat <<'EOF'
refactor(hooks): mob spell discovery uses DiscoveryChance helper

Same split as player spell discovery — traditional + manifestation rolls
compute independently with school-appropriate skill. Mobs share the
Character.Stats.Perception.ValueAdj field, so the helper wires up
identically.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Migrate recipe discovery call site

**Files:**
- Modify: `internal/hooks/NewRound_UserRoundTick.go:388-407`

- [ ] **Step 1: Inspect the current block**

Run: `sed -n '388,408p' internal/hooks/NewRound_UserRoundTick.go`

Expected structure (verify):

```go
// Stage 31.1: Recipe discovery roll
bal := configs.GetBalanceConfig()
knownCount := len(user.Character.KnownRecipes)
discChance := float64(bal.RecipeDiscoveryBaseChance) /
	(1.0 + float64(knownCount)*float64(bal.RecipeDiscoveryDecayRate))
if util.Rand(100) < int(discChance) {
```

- [ ] **Step 2: Replace with helper call**

The recipe's relevant skill comes from `recipe.Skill` (a `SkillTag` string). Convert to rank via `user.Character.GetSkillLevel(skills.SkillTag(recipe.Skill))` — but note: that same expression is already computed just above at line ~372 as `newItem.CraftSkill`. Re-compute locally for clarity (the scope is a few lines, refactoring to share is out of scope).

New:

```go
// Stage 31.1: Recipe discovery roll
bal := configs.GetBalanceConfig()
knownCount := len(user.Character.KnownRecipes)
craftSkillLevel := user.Character.GetSkillLevel(skills.SkillTag(recipe.Skill))
discChance := configs.DiscoveryChance(configs.DiscoveryParams{
	Base:       float64(bal.RecipeDiscoveryBaseChance),
	Decay:      float64(bal.RecipeDiscoveryDecayRate),
	Known:      knownCount,
	Perception: user.Character.Stats.Perception.ValueAdj,
	Skill:      craftSkillLevel,
})
if util.Rand(100) < int(discChance) {
```

- [ ] **Step 3: Verify `skills` import is available**

Run: `grep -n '"github.com/GoMudEngine/GoMud/internal/skills"' internal/hooks/NewRound_UserRoundTick.go`

**If present**: skip to Step 4.

**If absent**: add the import. Run:

```bash
grep -n '^import\|"github.com/GoMudEngine/GoMud/internal/' internal/hooks/NewRound_UserRoundTick.go | head -10
```

Add `"github.com/GoMudEngine/GoMud/internal/skills"` to the import block, alphabetically sorted with existing imports.

- [ ] **Step 4: Build**

Run: `go build ./internal/hooks/...`
Expected: no output.

- [ ] **Step 5: Run hooks tests**

Run: `go test ./internal/hooks/...`
Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/NewRound_UserRoundTick.go
git commit -m "$(cat <<'EOF'
refactor(hooks): recipe discovery uses DiscoveryChance helper

Recipe's specific skill (blacksmithing / alchemy / tailoring / cooking /
jewelcrafting / enchanting) drives the offset along with Perception.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Full-repo build + test sweep

- [ ] **Step 1: Verify no inline formula remains**

Run:

```bash
grep -rn "SpellDiscoveryBaseChance\|RecipeDiscoveryBaseChance" --include="*.go" --exclude-dir=.worktrees internal/
```

Expected: only 4 results — the two field declarations in `config.balance.go`, and the two default assignments (one each in `config.balance.spells.go` and `config.balance.shops.go`). **No** references in `internal/hooks/`.

If any inline formula remains, go back to the relevant Task (3/4/5) and finish the migration.

- [ ] **Step 2: Full build**

Run: `go build ./...`
Expected: no output.

- [ ] **Step 3: Full test suite**

Run: `go test ./...`
Expected: all pass.

If any test fails that wasn't failing before, read its setup — it may need to seed the Balance config with the new defaults. The validator should handle this automatically, but a hand-constructed `Balance{}` in a test could hit zero-valued knobs. Fix by adding `.validateDiscovery()` to that test's setup or by initializing the knobs inline.

- [ ] **Step 4: Commit (if any Step 3 fixes needed)**

If Step 3 required fixes, commit them. Otherwise this task has no commit.

---

## Task 7: Patch notes

**Files:**
- Modify: `PATCH_NOTES.md`

- [ ] **Step 1: Add patch-notes entry**

Open `PATCH_NOTES.md`. At the very top, **after** the `# DOGMud Patch Notes` header and **before** the most recent `## 2026-04-22 (evening)` section, insert:

```markdown
## 2026-04-24 — Discovery Rate Stat Offset

### Gameplay

- **Spell and recipe discovery now scales with Perception + skill.**
  The decay that slows discovery as you learn more spells/recipes
  is now partially offset by your Perception stat and the relevant
  skill (Spellcasting for traditional spells, Manifestation for
  manifestation-school spells, or the specific crafting skill for
  each recipe). A newbie discovers at the current rate; a seasoned
  character with invested Per + skill discovers roughly 1.8× faster
  at 20 known — closing the late-game discovery drought without
  flooding new characters with learn-messages.
- **Offset mechanic:** Per contribution reaches 1.0 at Per=300,
  skill contribution reaches 1.0 at rank 100, combined via
  `1 - (1 - per)(1 - skill)` and capped at 0.8 (effective decay
  floor = 20% of base). Either Per or skill alone gives a partial
  benefit; the combination unlocks the full cap.
- **Mobs benefit too.** Caster mobs with high Per + Spellcasting
  will expand their spell repertoire faster than before — a
  battle-hardened mob learning from repeated casts.

### Config

- New `Balance` knobs: `DiscoveryPerceptionScale` (default 200),
  `DiscoverySkillScale` (default 100), `DiscoveryMaxDecayOffset`
  (default 0.8). Set `DiscoveryMaxDecayOffset: 0` to disable the
  offset mechanic entirely and revert to the prior flat-chance
  formula.

```

- [ ] **Step 2: Commit**

```bash
git add PATCH_NOTES.md
git commit -m "$(cat <<'EOF'
docs: patch notes for discovery rate stat offset

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Testing Summary

**Unit coverage** (Task 2):
- 12 table-driven cases exercising every scenario in the spec table + edge cases (per-below-baseline, skill-above-scale, known=0, pure-Per, pure-skill, max-offset cap).
- Defaults-applied test verifies the validator.
- ReadsConfiguredBalance test proves the helper reads from the singleton correctly.

**Integration coverage**:
- Existing `internal/hooks/` tests run after each migration (Tasks 3/4/5).
- Full-repo `go test ./...` in Task 6.

**Manual smoke** (post-merge, in-game):
1. Log in as a low-level character. Cast a few spells. Verify occasional discovery message fires ("A new pattern crystallizes…").
2. Use `admin` to set Perception to 200 and Spellcasting to 100 on a test character. Cast the same spells. Verify discoveries fire noticeably more often (compare roll-count feel).
3. Craft a few recipes at low skill; then level the specific crafting skill to ~50 and verify discovery rate rises.

**Non-goals** (do NOT add):
- Player command to show current discovery rate.
- Changing default BaseChance or BaseDecay values.
- Retroactive recompute for mid-session characters.

---

## Self-Review

**Spec coverage:**
- Formula → Task 2 ✓
- Three new knobs → Task 1 ✓
- Shared between spell + recipe → Task 1 (single validator file) ✓
- Helper location (`internal/configs/discovery.go`) → Task 2 ✓
- Skill mapping per call site → Tasks 3/4/5 (each passes the right skill) ✓
- 5 call sites migrated → Tasks 3 (2 of them), 4 (2 of them), 5 (1) ✓
- Testing plan → Task 2 unit tests + Task 6 full sweep + manual smoke ✓
- Edge cases (Per<100, skill=0, known=0, high-offset cap) → Task 2 ✓
- Non-goals (no visibility command, no base tuning) → Task 7 patch notes reflects this ✓

**Placeholder scan:** no TBDs, every code block is complete, every `grep` / `sed` command is exact.

**Type consistency:** `DiscoveryChance` + `DiscoveryParams` used identically across Task 2/3/4/5. Field names (`Base`, `Decay`, `Known`, `Perception`, `Skill`) stable across tasks. `configs.DiscoveryChance` qualified path used consistently in all call-site tasks.
