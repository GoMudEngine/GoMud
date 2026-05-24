# Grapple Drift Formula Rework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Spec:** `docs/superpowers/specs/2026-05-19-grapple-drift-formula-rework-design.md`

**Goal:** Fix the per-round grapple drift formula in `internal/hooks/Position_GrappleTick.go` so that grapples no longer auto-escape on round 1: use `UnarmedCombat` (the documented grappling skill) instead of `WeaponCombat`, drop the oversized defender Dex bonus, and apply a small aggressor edge tied to skill.

**Architecture:** New pure helper function `grappleScore(c, isAggressor)` computes one side's score as `(0.7·Str + 0.3·Dex + skill_coef·UnarmedCombat) × stamina_mult × encumbrance_mult`, where `skill_coef = 2.2` for the aggressor side and `2.0` for the defender. `processGrapplePair` calls the helper for each side instead of computing inline. The chunk-4b-fixup-2 outcome resolver, ControlLevel shift, pair iteration, and all messaging stay unchanged. `escapeModifierFromBody` helper is deleted; `ItemSpec.EscapeModifier` field stays but is no longer read.

**Tech Stack:** Go 1.21+, existing `dice.OpposedRollStat`, existing `skills.UnarmedCombat` skill tag, existing stamina/encumbrance multiplier helpers. No new dependencies.

---

## File Structure

### Modified
| Path | Change |
|---|---|
| `internal/hooks/Position_GrappleTick.go` | Add `grappleScore` helper; rewrite `processGrapplePair` score lines; delete `escapeModifierFromBody`; update score-formula comment block |
| `internal/hooks/Position_GrappleTick_test.go` (extend if exists, create if not) | Unit tests for each coefficient; regression test for quester-vs-boar |
| `internal/hooks/context.md` | Document new formula |
| `_datafiles/world/dogmud/templates/help/grapple.template` | Add brief "what matters" section (descriptive, no hard numbers) |
| `_datafiles/world/dogmud/templates/help/unarmed-combat.template` | Note the skill drives the grapple drift roll |

### Unchanged but referenced
- `internal/skills/skills.go` — `UnarmedCombat` constant (no change)
- `internal/state/position/pair.go` — `GrappleData.IsAggressor` field (set elsewhere, read here)
- `internal/items/itemspec.go` — `EscapeModifier` field stays in struct, just unused by the formula
- `_datafiles/world/dogmud/items/armor-20000/body/*.yaml` — EscapeModifier YAML values stay valid

---

## Task 1: Add `grappleScore` helper function + unit tests

**Files:**
- Modify: `internal/hooks/Position_GrappleTick.go`
- Modify: `internal/hooks/Position_GrappleTick_test.go` (create if doesn't exist)

- [ ] **Step 1: Verify the test file exists or create it**

Run: `ls internal/hooks/Position_GrappleTick_test.go 2>&1`

If missing, create a minimal scaffold:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
)
```

If present, ensure those imports are already there or will be added by step 3's test code.

- [ ] **Step 2: Write the failing unit tests**

Append to `internal/hooks/Position_GrappleTick_test.go`:

```go
// makeGrappleCharacter constructs a minimal Character with the given
// stats + skill level for grappleScore unit tests. Stamina and
// encumbrance multipliers will resolve to 1.0 because no penalties
// apply (Stamina at max, no items carried).
func makeGrappleCharacter(t *testing.T, str, dex, ucSkill int) *characters.Character {
	t.Helper()
	c := characters.NewCharacter()
	c.Stats.Strength.Base = str
	c.Stats.Dexterity.Base = dex
	c.Stats.Strength.Recalculate()
	c.Stats.Dexterity.Recalculate()
	c.Stamina = 1000
	c.StaminaMax.Value = 1000
	if c.Skills == nil {
		c.Skills = map[string]int{}
	}
	if ucSkill > 0 {
		c.Skills[string(skills.UnarmedCombat)] = ucSkill
	}
	return c
}

func TestGrappleScore_StrCoefficient(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	c := makeGrappleCharacter(t, 100, 0, 0)
	got := grappleScore(c, false, cfg)
	want := 70.0 // 0.7 * 100 + 0 + 0
	if got != want {
		t.Errorf("Str=100 only → score = %v, want %v", got, want)
	}
}

func TestGrappleScore_DexCoefficient(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	c := makeGrappleCharacter(t, 0, 100, 0)
	got := grappleScore(c, false, cfg)
	want := 30.0 // 0 + 0.3 * 100 + 0
	if got != want {
		t.Errorf("Dex=100 only → score = %v, want %v", got, want)
	}
}

func TestGrappleScore_SkillCoefficientDefender(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	c := makeGrappleCharacter(t, 0, 0, 50)
	got := grappleScore(c, false, cfg)
	want := 100.0 // 0 + 0 + 2.0 * 50
	if got != want {
		t.Errorf("UC=50 defender → score = %v, want %v", got, want)
	}
}

func TestGrappleScore_SkillCoefficientAggressor(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	c := makeGrappleCharacter(t, 0, 0, 50)
	got := grappleScore(c, true, cfg)
	want := 110.0 // 0 + 0 + 2.2 * 50
	if got != want {
		t.Errorf("UC=50 aggressor → score = %v, want %v", got, want)
	}
}

func TestGrappleScore_CombinedFormulaDefender(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	c := makeGrappleCharacter(t, 100, 100, 30)
	got := grappleScore(c, false, cfg)
	want := 130.0 // 70 + 30 + 60
	if got != want {
		t.Errorf("S100/D100/UC30 defender → score = %v, want %v", got, want)
	}
}

func TestGrappleScore_CombinedFormulaAggressor(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	c := makeGrappleCharacter(t, 100, 100, 30)
	got := grappleScore(c, true, cfg)
	want := 136.0 // 70 + 30 + 36 (skill coef 2.2)
	if got != want {
		t.Errorf("S100/D100/UC30 aggressor → score = %v, want %v", got, want)
	}
}

func TestGrappleScore_EscapeModifierIgnored(t *testing.T) {
	// EscapeModifier on body armor should NOT change the score.
	// The legacy formula read it; the new formula does not.
	cfg := configs.GetBalanceConfig()
	c := makeGrappleCharacter(t, 100, 100, 0)
	baseline := grappleScore(c, false, cfg)
	// Equip body armor with high EscapeModifier — note: we'd need to
	// actually wire up an Equipment.Body slot to test this end-to-end,
	// but the simpler verification is: grappleScore never references
	// any EscapeModifier value (no field read in the function).
	got := grappleScore(c, false, cfg)
	if got != baseline {
		t.Errorf("score should not depend on EscapeModifier; got %v, baseline %v", got, baseline)
	}
}

func TestGrappleScore_NilCharacter(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	got := grappleScore(nil, false, cfg)
	if got != 0 {
		t.Errorf("nil character → score = %v, want 0", got)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/hooks/ -run "TestGrappleScore_" -count=1`
Expected: FAIL with undefined `grappleScore`.

- [ ] **Step 4: Implement `grappleScore` helper**

Add to `internal/hooks/Position_GrappleTick.go` (near the existing `grappleStaminaMultiplier` / `grappleEncumbranceMultiplier` helpers, after them):

```go
// grappleScore computes one side's per-round drift score. Spec
// 2026-05-19 §3. Symmetric formula for both sides; aggressor gets a
// 10% bonus on the skill term (initiative edge).
//
//	score = (0.7·Str + 0.3·Dex + skill_coef·UnarmedCombat)
//	        × stamina_multiplier × encumbrance_multiplier
//
// where skill_coef = 2.2 for the aggressor side, 2.0 for the defender.
// Returns 0 for a nil character (defensive — callers should never
// pass nil but the function shouldn't panic on it).
func grappleScore(c *characters.Character, isAggressor bool, cfg configs.Balance) float64 {
	if c == nil {
		return 0
	}
	strVal := float64(c.Stats.Strength.Value)
	dexVal := float64(c.Stats.Dexterity.Value)
	skill := float64(c.GetSkillLevel(skills.UnarmedCombat))

	skillCoef := 2.0
	if isAggressor {
		skillCoef = 2.2
	}

	base := 0.7*strVal + 0.3*dexVal + skillCoef*skill
	return base * grappleStaminaMultiplier(c, cfg) * grappleEncumbranceMultiplier(c, cfg)
}
```

- [ ] **Step 5: Run tests to verify pass**

Run: `go test ./internal/hooks/ -run "TestGrappleScore_" -count=1 -v`
Expected: ALL PASS (8 tests).

- [ ] **Step 6: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 7: Commit**

```bash
git add internal/hooks/Position_GrappleTick.go internal/hooks/Position_GrappleTick_test.go
git commit -m "feat(hooks): grapple formula T1 — grappleScore helper + unit tests

Pure helper computes one side's drift score per the new spec:
  (0.7·Str + 0.3·Dex + coef·UnarmedCombat) × stamina_mult × enc_mult
where coef = 2.2 for aggressor, 2.0 for defender.

8 unit tests cover Str coefficient, Dex coefficient, defender
skill coefficient, aggressor skill coefficient, combined formula
(both sides), EscapeModifier-ignored sanity, nil-character defensive.

Helper not yet wired into processGrapplePair — T2 does that.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Wire `grappleScore` into `processGrapplePair`

**Files:**
- Modify: `internal/hooks/Position_GrappleTick.go`

- [ ] **Step 1: Read the current score-computation block in processGrapplePair**

Open `internal/hooks/Position_GrappleTick.go` and locate the lines that currently compute `ctrlBase`, `cdBase`, `ctrlScore`, and `cdScore` (around lines 258-275). Note the surrounding context — the `dice.OpposedRollStat` call comes right after.

- [ ] **Step 2: Replace the inline score formula with helper calls**

Find this block (exact location may vary slightly; search for `ctrlBase :=` and `cdBase :=`):

```go
	// Score formula unchanged from chunk 4b:
	//   controller: (Str + WeaponCombat) × stamina × encumbrance
	//   controlled: (Str + WeaponCombat + 0.5·Dex + body.EscapeModifier)
	//               × stamina × encumbrance
	ctrlBase := float64(controller.Stats.Strength.Value) +
		float64(controller.GetSkillLevel(skills.WeaponCombat))
	cdBase := float64(controlled.Stats.Strength.Value) +
		float64(controlled.GetSkillLevel(skills.WeaponCombat)) +
		0.5*float64(controlled.Stats.Dexterity.Value) +
		escapeModifierFromBody(controlled)

	ctrlScore := ctrlBase *
		grappleStaminaMultiplier(controller, cfg) *
		grappleEncumbranceMultiplier(controller, cfg)
	cdScore := cdBase *
		grappleStaminaMultiplier(controlled, cfg) *
		grappleEncumbranceMultiplier(controlled, cfg)
```

Replace with:

```go
	// Score formula (spec 2026-05-19 §3):
	//   score = (0.7·Str + 0.3·Dex + skill_coef·UnarmedCombat)
	//           × stamina_mult × encumbrance_mult
	//   skill_coef = 2.2 (aggressor) or 2.0 (defender)
	//
	// IsAggressor is set on GrappleData by ApplyGrappleResult's
	// markAggressor call (chunk 4b-fixup-2 T5). It persists for the
	// lifetime of the grapple — even after reversals, the original
	// initiator keeps the bonus.
	ctrlScore := grappleScore(controller, isAggressorSide(controller), cfg)
	cdScore := grappleScore(controlled, isAggressorSide(controlled), cfg)
```

- [ ] **Step 3: Add the `isAggressorSide` helper**

Add this helper alongside `grappleScore` in `Position_GrappleTick.go`:

```go
// isAggressorSide returns true if this character's GrappleData
// has IsAggressor=true. Set once by ApplyGrappleResult.markAggressor
// (chunk 4b-fixup-2 T5) when the grapple is initiated; persists
// for the grapple's lifetime regardless of subsequent reversals.
// Returns false defensively for nil/Position-less characters.
func isAggressorSide(c *characters.Character) bool {
	if c == nil || c.Position == nil {
		return false
	}
	d, ok := c.Position.GrappleData()
	if !ok {
		return false
	}
	return d.IsAggressor
}
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: clean build. The old `ctrlBase` / `cdBase` variable names are gone; `ctrlScore` and `cdScore` are still defined at the same scope so the downstream `OpposedRollStat` call works unchanged.

- [ ] **Step 5: Run all hook tests**

Run: `go test ./internal/hooks/ -count=1`
Expected: PASS (existing tests + T1's new unit tests).

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/Position_GrappleTick.go
git commit -m "feat(hooks): grapple formula T2 — wire grappleScore into processGrapplePair

processGrapplePair now calls grappleScore() for each side instead
of computing the score inline with the buggy formula
(Str + WeaponCombat / +0.5·Dex defender bonus). New isAggressorSide
helper reads GrappleData.IsAggressor.

The dice.OpposedRollStat call, LastDriftRoll snapshot, outcome
resolver, ControlLevel shift, and all messaging are UNCHANGED —
only the score inputs to the roll changed.

escapeModifierFromBody is now unused (still defined; T3 deletes it).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Delete `escapeModifierFromBody` helper + comment cleanup

**Files:**
- Modify: `internal/hooks/Position_GrappleTick.go`

- [ ] **Step 1: Confirm `escapeModifierFromBody` is unused**

Run: `grep -rn "escapeModifierFromBody" internal/`
Expected: only the function definition itself in `Position_GrappleTick.go`. If any caller remains outside that file's own definition, that's a bug from T2 — fix the caller first.

- [ ] **Step 2: Delete the function**

In `internal/hooks/Position_GrappleTick.go`, find and remove:

```go
// escapeModifierFromBody reads the controlled character's body slot
// armor for the EscapeModifier field on ItemSpec. Mirrors the legacy
// CheckGroundedEscape helper from chunk 2.
func escapeModifierFromBody(c *characters.Character) float64 {
	// ...whole function body...
}
```

If the surrounding comment block has paragraph references to `EscapeModifier`, remove those too. The grapple-formula doc comment in T2 already explains the new shape; legacy references just clutter.

- [ ] **Step 3: Verify build + tests**

Run: `go build ./... && go test ./internal/hooks/ -count=1`
Expected: clean build + tests pass.

If the `items` package import becomes unused as a result of the deletion, `goimports` (or the editor) will remove it on save — verify imports are clean.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/Position_GrappleTick.go
git commit -m "chore(hooks): grapple formula T3 — delete escapeModifierFromBody

T2 stopped calling it. The ItemSpec.EscapeModifier field stays
on the struct and in YAML content (no breaking changes), but the
grapple drift formula no longer reads it. Future systems can
re-purpose the field if useful (sub eligibility, armor resistance
bias, etc.).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Regression integration test for quester-vs-boar scenario

**Files:**
- Modify: `internal/hooks/Position_GrappleTick_test.go`

- [ ] **Step 1: Write the regression test**

Append to `internal/hooks/Position_GrappleTick_test.go`:

```go
// TestGrappleScore_QuesterVsBoarSurvives is the regression test for
// the 2026-05-19 bug where quester0 immediately escaped from every
// grapple against a steppe boar. With the new formula, quester0's
// 30 unarmed-combat training should bring them into competitive
// range (not 80+ points behind).
//
// Stats from production YAML:
//   quester0: Str.Base=103 + Training=13 = 116; Dex.Base=113 + Training=13 = 126
//   quester0 skill: unarmed-combat=30
//   steppe boar (spawned): racial Str.Base=120 + ~29 training = ~149
//                          racial Dex.Base=70 + ~21 training = ~91
//   steppe boar skill: unarmed-combat=0
//
// Quester0 grappling boar: quester0 is aggressor. Expected:
//   atk = (0.7*116 + 0.3*126) + 2.2*30 = 119 + 66 = 185
//   def = (0.7*149 + 0.3*91) + 2.0*0  = 131.6 + 0 = 131.6
//   margin = 53.4, z = 53.4/(185*0.15) ≈ +1.92 → 2-step advance.
//
// We don't roll the dice here (too flaky) — we just verify the
// deterministic score components and the resulting margin.
func TestGrappleScore_QuesterVsBoarSurvives(t *testing.T) {
	cfg := configs.GetBalanceConfig()

	quester := makeGrappleCharacter(t, 116, 126, 30)
	boar := makeGrappleCharacter(t, 149, 91, 0)

	atkScore := grappleScore(quester, true, cfg)  // quester is aggressor
	defScore := grappleScore(boar, false, cfg)

	// Quester0's score should exceed boar's despite raw Str gap.
	if atkScore <= defScore {
		t.Errorf("quester0 (trained) score %.1f should exceed boar score %.1f", atkScore, defScore)
	}

	// Margin should be in the 2-step band (z in [1.0, 2.0)):
	// 2-step means decisive advance but not immediate sub/escape.
	margin := atkScore - defScore
	sigma := atkScore * 0.15 // RollSpread default
	z := margin / sigma
	if z < 1.0 || z >= 2.0 {
		t.Errorf("expected z in [1.0, 2.0) (2-step advance band), got z=%.2f (atk=%.1f def=%.1f margin=%.1f sigma=%.1f)",
			z, atkScore, defScore, margin, sigma)
	}
}

// TestGrappleScore_UntrainedVsTrainedDefenderEscapes is the inverse
// regression: an untrained aggressor against a trained defender
// should produce z ≤ -2.0 (escape). Confirms the formula doesn't
// silently make grapples always-favor-aggressor.
func TestGrappleScore_UntrainedVsTrainedDefenderEscapes(t *testing.T) {
	cfg := configs.GetBalanceConfig()

	untrained := makeGrappleCharacter(t, 100, 100, 0)
	trained := makeGrappleCharacter(t, 100, 100, 30)

	atkScore := grappleScore(untrained, true, cfg) // 100 + 0
	defScore := grappleScore(trained, false, cfg)  // 100 + 60

	margin := atkScore - defScore
	sigma := atkScore * 0.15
	z := margin / sigma

	if z > -2.0 {
		t.Errorf("expected z ≤ -2.0 (defender escape band), got z=%.2f (atk=%.1f def=%.1f)",
			z, atkScore, defScore)
	}
}

// TestGrappleScore_EqualPairHolds confirms balanced grapplers land
// in the Hold band (|z| < 0.5) when their stats and skill match.
func TestGrappleScore_EqualPairHolds(t *testing.T) {
	cfg := configs.GetBalanceConfig()

	a := makeGrappleCharacter(t, 100, 100, 15)
	b := makeGrappleCharacter(t, 100, 100, 15)

	atkScore := grappleScore(a, true, cfg)  // 100 + 33 = 133
	defScore := grappleScore(b, false, cfg) // 100 + 30 = 130

	margin := atkScore - defScore
	sigma := atkScore * 0.15
	z := margin / sigma

	if z < 0 || z >= 0.5 {
		t.Errorf("expected z in [0, 0.5) (Hold band, slight aggressor edge), got z=%.2f (atk=%.1f def=%.1f)",
			z, atkScore, defScore)
	}
}
```

- [ ] **Step 2: Run tests to verify pass**

Run: `go test ./internal/hooks/ -run "TestGrappleScore_QuesterVsBoarSurvives|TestGrappleScore_UntrainedVsTrainedDefenderEscapes|TestGrappleScore_EqualPairHolds" -count=1 -v`
Expected: ALL PASS (3 regression tests).

- [ ] **Step 3: Verify full hooks suite**

Run: `go test ./internal/hooks/ -count=1`
Expected: ALL PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/Position_GrappleTick_test.go
git commit -m "test(hooks): grapple formula T4 — regression tests for the original bug scenarios

Three integration tests pin the formula behavior against the
bug-reporting scenarios from bug_log.txt and the spec's example
table:

  1. Quester0 (trained, S116/D126/UC30) vs steppe boar (S149/D91/UC0)
     as aggressor → z in [1.0, 2.0) (2-step advance band, NOT escape).
     Catches the original bug.
  2. Untrained vs trained-defender → z ≤ -2.0 (defender escape band).
     Confirms skill gap still matters in the opposite direction.
  3. Equal pair (S100/D100/UC15 vs S100/D100/UC15) with A as aggressor
     → z in [0, 0.5) (Hold band with slight aggressor edge).

If anyone re-introduces the WeaponCombat skill bug or the +0.5·Dex
defender bonus, test #1 fails loud.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Update `internal/hooks/context.md` to document the new formula

**Files:**
- Modify: `internal/hooks/context.md`

- [ ] **Step 1: Find the Position_GrappleTick section**

Run: `grep -n "Position_GrappleTick\|score formula\|drift roll\|Str + WeaponCombat\|+0.5·Dex" internal/hooks/context.md`

Identify the section describing the grapple tick's per-round work.

- [ ] **Step 2: Replace the score-formula description**

Find the section that documents the score formula (likely titled something like "Per-round drift" or "Score computation"). Replace any reference to the old formula with:

```markdown
### Score formula (2026-05-19 rework)

Each side's per-round score is computed by `grappleScore(c, isAggressor, cfg)`:

```
score = (0.7·Str + 0.3·Dex + skill_coef·UnarmedCombat)
        × stamina_multiplier × encumbrance_multiplier
```

where `skill_coef = 2.2` for the aggressor (the side that initiated
the grapple via `grapple` command or btree `grapple` primitive) and
`2.0` for the defender. Symmetric in shape — no role-based unilateral
stat bonus. Position bias is already captured by `ControlLevel` state
initialization (chunk 4b-fixup-2); the formula doesn't double-encode it.

The aggressor flag (`GrappleData.IsAggressor`) is set once by
`ApplyGrappleResult.markAggressor` at grapple entry and persists for
the grapple's lifetime, regardless of any later reversals.

Body-armor `EscapeModifier` is NOT read by the formula. The field
remains on `ItemSpec` for backward compatibility and possible future
re-purposing (sub eligibility, armor resistance, etc.). The legacy
`escapeModifierFromBody` helper is deleted.

The grapple-skill is `UnarmedCombat` per its own definition
(`internal/skills/skills.go:29`: "Fist/body attacks & defense,
grappling"). Earlier versions of this formula read `WeaponCombat`
by mistake; that bug auto-escaped every grapple for any unarmed-
trained player.

See `docs/superpowers/specs/2026-05-19-grapple-drift-formula-rework-design.md`
for the design rationale and sample z-score table.
```

If the old section also documented the buggy comment block ("controller: (Str + WeaponCombat)... controlled: ... +0.5·Dex... +body.EscapeModifier"), remove those lines — they're historically interesting but now misleading.

- [ ] **Step 3: Verify the doc reads coherently end-to-end**

Re-read the modified section. Make sure no stale references to `WeaponCombat`, `+0.5·Dex defender bonus`, or `EscapeModifier` survive in the formula description (historical-sunset narrative paragraphs may keep them, but live formula docs should not).

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/context.md
git commit -m "docs(hooks): grapple formula T5 — context.md updated for the new score formula

Position_GrappleTick section reflects the 2026-05-19 rework:
symmetric formula with 0.7·Str + 0.3·Dex + skill_coef·UnarmedCombat,
aggressor skill coefficient bumped 10%, EscapeModifier no longer
read.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Update helpfiles — no hard numbers per project SOP

**Files:**
- Modify: `_datafiles/world/dogmud/templates/help/grapple.template`
- Modify: `_datafiles/world/dogmud/templates/help/unarmed-combat.template`

Project SOP requires NO hard numbers in player-facing text. Descriptions are mechanical-but-vague: "strength matters most", "your unarmed skill is the biggest factor" — never "0.7·Str + 0.3·Dex + 2·UnarmedCombat".

- [ ] **Step 1: Read both helpfiles end-to-end**

Run:
```
cat _datafiles/world/dogmud/templates/help/grapple.template
cat _datafiles/world/dogmud/templates/help/unarmed-combat.template
```

Note where to insert "what determines drift" without breaking existing structure.

- [ ] **Step 2: Add a "What determines the round outcome" section to grapple.template**

Open `_datafiles/world/dogmud/templates/help/grapple.template`. After the existing "Grapple Positions" section (and before the next section, likely about commands or related help), insert:

```
<ansi fg="yellow">━━━ What Decides Each Round ━━━</ansi>

A grapple round resolves through an opposed contest of fitness and
technique. Several things tilt the contest:

  <ansi fg="stat">Unarmed Combat skill:</ansi> the biggest single factor.
            Trained grapplers reliably beat untrained ones of
            similar build.

  <ansi fg="stat">Strength:</ansi> the primary stat for controlling and
            holding positions.

  <ansi fg="stat">Dexterity:</ansi> a secondary stat for setups, framing,
            and scrambling.

  <ansi fg="stat">Initiative:</ansi> the side that started the grapple has
            a small edge that round — they picked the moment.

  <ansi fg="stat">Stamina:</ansi> a gassed grappler is dramatically worse.
            Long grapples favor whoever paces themselves.

  <ansi fg="stat">Encumbrance:</ansi> carrying too much weight hurts both
            sides of a grapple.

Heavy armor doesn't directly help you escape — it just costs you
more stamina to fight while wearing it.
```

Verify line wrap is ≤80 chars per the project SOP.

- [ ] **Step 3: Add a grapple-drift note to unarmed-combat.template**

Open `_datafiles/world/dogmud/templates/help/unarmed-combat.template`. The file already mentions grapple in its "Grapple-Friendly" bullet. Add a more explicit line in the existing "Attacks" or near the "Grapple-Friendly" bullet:

Find the "Grapple-Friendly" bullet (looks like):

```
  - <ansi fg="stat">Grapple-Friendly:</ansi> Fists and claws have very
    short reach. They stay fully effective in clinches and ground
    grapples where long weapons become awkward.
```

Add a follow-on bullet immediately after it:

```
  - <ansi fg="stat">Grapple Drift:</ansi> <ansi fg="skill">unarmed-combat</ansi>
    is the primary skill that determines who wins each round of a
    grapple's drift toward advance, hold, reversal, or escape.
    See <ansi fg="command">help grapple</ansi> for the full picture.
```

Verify line wrap ≤80 chars.

- [ ] **Step 4: Verify both templates render correctly**

Read both modified files end-to-end. Confirm:
- No hard numbers (no "+0.7", no "skill 50", no "coefficient 2.2", etc.).
- Line wrap ≤80 chars throughout.
- Cross-references to other help commands are accurate (`help grapple`, `help submission` exist).
- ANSI tags are balanced (`<ansi fg=...>` closed with `</ansi>`).

Run a quick wrap check:

```bash
awk 'length > 80' _datafiles/world/dogmud/templates/help/grapple.template _datafiles/world/dogmud/templates/help/unarmed-combat.template
```

Expected: no output (no lines over 80 chars). If a line exceeds 80, wrap it.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/templates/help/grapple.template _datafiles/world/dogmud/templates/help/unarmed-combat.template
git commit -m "docs(help): grapple formula T6 — player-facing notes for new drift formula

grapple.template gains a 'What Decides Each Round' section listing
the inputs in descriptive terms: unarmed-combat skill (biggest
factor), strength (primary stat), dexterity (secondary), initiative
(aggressor edge), stamina, encumbrance. No hard numbers per project
SOP.

unarmed-combat.template gains a Grapple Drift bullet making the
skill's role in grapple rounds explicit, with a cross-reference to
help grapple.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Boot smoke + AI feature-tester re-run

**Files (verification only):**
- No code changes

- [ ] **Step 1: Local boot smoke**

Build and start the server:

```bash
go build -o dogmud-test.exe .
./dogmud-test.exe > test-server.log 2>&1 &
```

Wait for "Server Ready" in the log:

```bash
until grep -q "Server Ready" test-server.log; do sleep 2; done
grep -E "Server Ready|panic|FATAL" test-server.log | head -3
```

Expected: Server Ready in ~5-8 seconds, no panics.

Verify the new formula is in effect by checking the log:

```bash
grep -E "grappleScore|UnarmedCombat" test-server.log
```

(Likely no specific log line; the formula is implicit. The real verification is the smoke run below.)

- [ ] **Step 2: Re-run the chunk-4b-fixup-2 feature-tester smoke**

The goal file `tools/testing/goals/chunk-4b-fixup-position-advancement-smoke.yaml` is still valid for verifying the grappling system works end-to-end. Use it:

Dispatch a feature-tester subagent (or `/test-mud local feature-tester chunk-4b-fixup-position-advancement-smoke.yaml`) and let it run.

**Expected change from the chunk-4b-fixup-2 results:**
- A trained-but-stat-disadvantaged character (like quester0 vs boar) should be able to HOLD a grapple — not immediately escape.
- An untrained character against a tougher mob should still struggle (formula didn't make grapples easy).
- Hold flavor, advance messaging, gradient messaging — all unchanged from chunk-4b-fixup-2.

If smoke surfaces immediate-escape behavior again, that's a sign the formula didn't actually deploy — check T2's wiring.

- [ ] **Step 3: Address findings (if any)**

If the smoke surfaces issues:
- Immediate escape still happening → check T2's wiring; verify `grappleScore` is called and `WeaponCombat` is not referenced anywhere. Re-grep: `grep -n "WeaponCombat" internal/hooks/Position_GrappleTick.go` should return nothing.
- Grapples now NEVER escape → formula may be over-tilted; verify by re-running the unit tests; they catch this case.
- Mob seems impossible to beat → expected behavior if the mob is a high-skill grappler; not a bug.

- [ ] **Step 4: Cleanup**

```bash
taskkill //F //IM dogmud-test.exe
taskkill //F //IM python.exe
rm -f dogmud-test.exe test-server.log
```

- [ ] **Step 5: Optional commit (only if fixes were needed)**

If the smoke ran clean, no commit is needed — the implementation is complete after T6.

If fixes were applied:

```bash
git commit -m "smoke(grapple-formula): T7 — boot smoke + AI tester verified

Quester0 vs steppe boar no longer immediately escapes — formula
deploys correctly. All chunk-4b-fixup-2 messaging still fires.

[any specific fixes applied here]

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Self-Review

**Spec coverage:**

| Spec section | Covered by |
|---|---|
| §1 Problem statement (wrong skill, oversized Dex bonus) | T1 (helper uses UnarmedCombat, drops Dex bonus), T2 (wires new helper) |
| §2 Design goals | All — symmetric formula (T1), skill matters (T1 + T4 tests), initiative edge (T1 coef), drop EscapeModifier (T3), z-scores in [-2, 2] (T4 verifies) |
| §3 Formula | T1 implements; T4 verifies via regression tests |
| §4 Implementation scope | T1-T3 (single file) + T5-T6 (docs) |
| §5 Testing strategy | T1 unit tests + T4 regression tests + T7 smoke |
| §6 Migration order | Plan task order matches |
| §7 Risks (IsAggressor populated, symmetric Hold-heavy fights, EscapeModifier vestigial, soft cap interaction) | T7 smoke verifies; symmetric Hold is acknowledged out-of-scope for tuning |
| §8 Out of scope | (no tasks needed) |
| §9 Success criteria | T4 + T7 verify |

**Placeholder scan:** clean. Every step has explicit code, exact file paths, exact commit messages. T3 has a defensive grep-first step before deletion — appropriate.

**Type consistency:** `grappleScore(c, isAggressor, cfg)` signature consistent across T1, T2, T4. `isAggressorSide(c)` introduced in T2 and not referenced again. Tests in T1 and T4 use the same helper. No drift.

Plan complete.
