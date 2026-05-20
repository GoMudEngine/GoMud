# State Chunk 4f — Position Balance + Smoke Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the deterministic Prone/Supine/Grapple spell-disruption gates in `processFoldRound` with a chance-based, Willpower-mediated per-position check, then run a comprehensive two-pass smoke across the whole chunk-4 position system and react to critical findings only. Sweep all chunk 4a-4f context.md files + helpfiles for accuracy and SOP compliance.

**Architecture:** Add a new lookup `internal/state/position/disruption.go` that returns a damage%-equivalent integer per (position, control-role) pair. `processFoldRound` calls the lookup, feeds it through the existing `characters.CalcConcentrationChance(Wil, dmgPctEquiv)` curve, and rolls. Standing returns 0 (skip the check). Existing damage-path `checkConcentrationBreak` is unchanged — both paths can break a cast in the same round, layered.

**Tech Stack:** Go, existing `internal/state/position` + `internal/state/control` + `internal/characters` packages, `_datafiles/world/dogmud/templates/help/*.template` helpfiles, AI testing harness (`/test-mud local feature-tester|feel-tester`).

**Spec reference:** `docs/superpowers/specs/2026-05-19-state-chunk-4f-balance-smoke-design.md`

---

## File Structure

### New files

| Path | Responsibility |
|---|---|
| `internal/state/position/disruption.go` | `PositionDisruptionDmgEquiv(pos, role) int` lookup + the per-position damage%-equivalent table. Same package as `modifiers.go`; sibling lookup. |
| `internal/state/position/disruption_test.go` | Unit tests covering every cell of the table, Standing-returns-0, controlled-role >= controller-role invariant, Guard inversion. |
| `tools/testing/goals/chunk-4f-position-system-smoke.yaml` | Comprehensive smoke goals file covering every chunk-4 deliverable. |

### Modified files

| Path | Change |
|---|---|
| `internal/hooks/combat_shared_helpers.go` | Replace the three deterministic 100% gates in `processFoldRound` (Prone/Supine block, Grapple block) with a single chance-based check that calls the new disruption lookup. |
| `_datafiles/world/dogmud/templates/help/grapple.template` | Soften the disruption language ("disrupted just as if knocked prone" → Willpower-mediated framing). |
| `internal/state/position/context.md` | Document the new disruption lookup. |
| `internal/hooks/context.md` | Document the rewrite of `processFoldRound`'s disruption gates. |
| `internal/characters/context.md` | Cross-link the position-disruption path next to the existing damage-path concentration documentation. |
| `internal/state/control/context.md` | Note that ControlLevel role feeds the new disruption lookup. |
| `internal/combat/context.md` | If it documents anything chunk-4-related, ensure it's still accurate. |
| `COMBAT_STATE_ROADMAP.md` | Add the chunk 4f row as Done. |
| (Helpfiles surfaced by the audit in Task 5) | Per-file edits to remove numerical-value SOP violations and stale chunk-4 wording. Scope list defined in Task 5; specific files edited depend on audit findings. |

---

## Tasks

### Task 1: New `disruption.go` lookup + unit tests

**Files:**
- Create: `internal/state/position/disruption.go`
- Test: `internal/state/position/disruption_test.go`

- [ ] **Step 1: Write the failing unit-test file**

Create `internal/state/position/disruption_test.go` with the full test set:

```go
package position

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state/control"
)

func TestPositionDisruptionDmgEquiv_StandingReturnsZero(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(Standing, control.Neutral); got != 0 {
		t.Errorf("Standing → %d, want 0", got)
	}
	if got := PositionDisruptionDmgEquiv(Standing, control.Controlling); got != 0 {
		t.Errorf("Standing controller → %d, want 0", got)
	}
	if got := PositionDisruptionDmgEquiv(Standing, control.Controlled); got != 0 {
		t.Errorf("Standing controlled → %d, want 0", got)
	}
}

func TestPositionDisruptionDmgEquiv_ProneSupine(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(Prone, control.Neutral); got != 30 {
		t.Errorf("Prone → %d, want 30", got)
	}
	if got := PositionDisruptionDmgEquiv(Supine, control.Neutral); got != 25 {
		t.Errorf("Supine → %d, want 25", got)
	}
}

func TestPositionDisruptionDmgEquiv_Clinch(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(Clinch, control.Controlling); got != 40 {
		t.Errorf("Clinch controller → %d, want 40", got)
	}
	if got := PositionDisruptionDmgEquiv(Clinch, control.Controlled); got != 40 {
		t.Errorf("Clinch controlled → %d, want 40", got)
	}
}

func TestPositionDisruptionDmgEquiv_BackStanding(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(BackStanding, control.Controlling); got != 35 {
		t.Errorf("BackStanding controller → %d, want 35", got)
	}
	if got := PositionDisruptionDmgEquiv(BackStanding, control.Controlled); got != 50 {
		t.Errorf("BackStanding controlled → %d, want 50", got)
	}
}

func TestPositionDisruptionDmgEquiv_Mount(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(Mount, control.Controlling); got != 35 {
		t.Errorf("Mount controller → %d, want 35", got)
	}
	if got := PositionDisruptionDmgEquiv(Mount, control.Controlled); got != 60 {
		t.Errorf("Mount controlled → %d, want 60", got)
	}
}

func TestPositionDisruptionDmgEquiv_SideControl(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(SideControl, control.Controlling); got != 30 {
		t.Errorf("SideControl controller → %d, want 30", got)
	}
	if got := PositionDisruptionDmgEquiv(SideControl, control.Controlled); got != 55 {
		t.Errorf("SideControl controlled → %d, want 55", got)
	}
}

func TestPositionDisruptionDmgEquiv_KneeOnBelly(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(KneeOnBelly, control.Controlling); got != 30 {
		t.Errorf("KneeOnBelly controller → %d, want 30", got)
	}
	if got := PositionDisruptionDmgEquiv(KneeOnBelly, control.Controlled); got != 50 {
		t.Errorf("KneeOnBelly controlled → %d, want 50", got)
	}
}

func TestPositionDisruptionDmgEquiv_NorthSouth(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(NorthSouth, control.Controlling); got != 30 {
		t.Errorf("NorthSouth controller → %d, want 30", got)
	}
	if got := PositionDisruptionDmgEquiv(NorthSouth, control.Controlled); got != 45 {
		t.Errorf("NorthSouth controlled → %d, want 45", got)
	}
}

func TestPositionDisruptionDmgEquiv_Crucifix(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(Crucifix, control.Controlling); got != 35 {
		t.Errorf("Crucifix controller → %d, want 35", got)
	}
	if got := PositionDisruptionDmgEquiv(Crucifix, control.Controlled); got != 70 {
		t.Errorf("Crucifix controlled → %d, want 70", got)
	}
}

func TestPositionDisruptionDmgEquiv_BackGround(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(BackGround, control.Controlling); got != 35 {
		t.Errorf("BackGround controller → %d, want 35", got)
	}
	if got := PositionDisruptionDmgEquiv(BackGround, control.Controlled); got != 65 {
		t.Errorf("BackGround controlled → %d, want 65", got)
	}
}

func TestPositionDisruptionDmgEquiv_HalfGuard(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(HalfGuard, control.Controlling); got != 30 {
		t.Errorf("HalfGuard controller → %d, want 30", got)
	}
	if got := PositionDisruptionDmgEquiv(HalfGuard, control.Controlled); got != 40 {
		t.Errorf("HalfGuard controlled → %d, want 40", got)
	}
}

// Guard is inverted: bottom (Controlling) has free hands, lower disruption
// than top (Controlled) who is stuck on top of someone's hips.
func TestPositionDisruptionDmgEquiv_GuardInverted(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(Guard, control.Controlling); got != 25 {
		t.Errorf("Guard bottom (controller) → %d, want 25", got)
	}
	if got := PositionDisruptionDmgEquiv(Guard, control.Controlled); got != 40 {
		t.Errorf("Guard top (controlled) → %d, want 40", got)
	}
}

func TestPositionDisruptionDmgEquiv_Turtle(t *testing.T) {
	if got := PositionDisruptionDmgEquiv(Turtle, control.Controlling); got != 35 {
		t.Errorf("Turtle controller → %d, want 35", got)
	}
	if got := PositionDisruptionDmgEquiv(Turtle, control.Controlled); got != 45 {
		t.Errorf("Turtle controlled → %d, want 45", got)
	}
}

// Sanity invariant: for every NON-Guard ground grapple, controlled-role
// disruption is >= controller-role disruption (the bottom guy has it worse).
func TestPositionDisruptionDmgEquiv_ControlledWorseExceptGuard(t *testing.T) {
	positions := []State{Mount, SideControl, KneeOnBelly, NorthSouth,
		Crucifix, BackGround, HalfGuard, BackStanding, Turtle}
	for _, pos := range positions {
		ctrl := PositionDisruptionDmgEquiv(pos, control.Controlling)
		cd := PositionDisruptionDmgEquiv(pos, control.Controlled)
		if cd < ctrl {
			t.Errorf("position %v: controlled (%d) < controller (%d); expected controlled >= controller", pos, cd, ctrl)
		}
	}
}

// Guard inversion sanity: controller-role (bottom) < controlled-role (top).
func TestPositionDisruptionDmgEquiv_GuardControllerLowerThanControlled(t *testing.T) {
	ctrl := PositionDisruptionDmgEquiv(Guard, control.Controlling)
	cd := PositionDisruptionDmgEquiv(Guard, control.Controlled)
	if ctrl >= cd {
		t.Errorf("Guard inversion broken: controller %d >= controlled %d; expected controller < controlled", ctrl, cd)
	}
}
```

- [ ] **Step 2: Run the tests and verify they fail**

Run: `go test ./internal/state/position/... -run TestPositionDisruptionDmgEquiv -v`
Expected: FAIL with "undefined: PositionDisruptionDmgEquiv".

- [ ] **Step 3: Write the implementation**

Create `internal/state/position/disruption.go`:

```go
// Per-position spell-disruption damage%-equivalents (chunk 4f). Fed into
// the existing characters.CalcConcentrationChance(Wil, dmgPctEquiv) curve
// in internal/hooks/combat_shared_helpers.go:processFoldRound to produce a
// per-round, Willpower-mediated disruption chance. Replaces the three
// deterministic 100% gates that chunks 4e and earlier shipped.
//
// Returns 0 for Standing (no check). Higher = more disruption. Controlled-role
// values are generally >= controller-role values because the controlled side
// has hands and movement suppressed. Guard inverts: bottom (Controlling) has
// free hands and lower disruption than top (Controlled).
//
// See docs/superpowers/specs/2026-05-19-state-chunk-4f-balance-smoke-design.md
// §3.1 for the table + rationale.
package position

import (
	"github.com/GoMudEngine/GoMud/internal/state/control"
)

// PositionDisruptionDmgEquiv returns the damage%-equivalent for a caster in
// the given position + control role. Returns 0 for Standing (skip the
// check). Returned value is fed to CalcConcentrationChance(Wil, dmgPctEquiv).
func PositionDisruptionDmgEquiv(pos State, role control.State) int {
	switch pos {
	case Standing:
		return 0
	case Prone:
		return 30
	case Supine:
		return 25
	case Clinch:
		return 40
	case BackStanding:
		if role == control.Controlling {
			return 35
		}
		return 50
	case Mount:
		if role == control.Controlling {
			return 35
		}
		return 60
	case SideControl:
		if role == control.Controlling {
			return 30
		}
		return 55
	case KneeOnBelly:
		if role == control.Controlling {
			return 30
		}
		return 50
	case NorthSouth:
		if role == control.Controlling {
			return 30
		}
		return 45
	case Crucifix:
		if role == control.Controlling {
			return 35
		}
		return 70
	case BackGround:
		if role == control.Controlling {
			return 35
		}
		return 65
	case HalfGuard:
		if role == control.Controlling {
			return 30
		}
		return 40
	case Guard:
		// Inverted: bottom (Controlling) has free hands; top (Controlled)
		// is stuck on someone's hips.
		if role == control.Controlling {
			return 25
		}
		return 40
	case Turtle:
		if role == control.Controlling {
			return 35
		}
		return 45
	}
	return 0
}
```

- [ ] **Step 4: Run the tests and verify they pass**

Run: `go test ./internal/state/position/... -run TestPositionDisruptionDmgEquiv -v`
Expected: PASS for all 14 tests.

- [ ] **Step 5: Run the full position-package test suite**

Run: `go test ./internal/state/position/...`
Expected: PASS — no regression in the existing modifiers/outcomes/submissions/validation tests.

- [ ] **Step 6: Commit**

```bash
git add internal/state/position/disruption.go internal/state/position/disruption_test.go
git commit -m "$(cat <<'EOF'
feat(state): T1 — per-position spell-disruption damage%-equivalent lookup

New PositionDisruptionDmgEquiv(pos, role) returns the damage%-equivalent
fed into the existing CalcConcentrationChance(Wil, dmgPctEquiv) curve.
Standing returns 0 (skip check); 25-70 range across the 13 grapple/downed
positions. Used by chunk 4f's processFoldRound rewrite (T2).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Refactor `processFoldRound` to call the new lookup + roll

**Files:**
- Modify: `internal/hooks/combat_shared_helpers.go:420-449` (the three deterministic 100% gates)
- Test: `internal/hooks/combat_shared_helpers_test.go` (new tests for chance-based behavior)

- [ ] **Step 1: Write the failing integration tests**

First, search for the existing test file to see if it exists:

```bash
ls internal/hooks/combat_shared_helpers_test.go 2>/dev/null
```

If it doesn't exist, create it. Otherwise append. Add these tests:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state/control"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// makeCastingChar returns a Character set up for a fold-round test. Caller
// configures Position/Control/Willpower as needed.
func makeCastingChar(t *testing.T, willpower int) *characters.Character {
	t.Helper()
	c := characters.New()
	c.Stats.Willpower.ValueAdj = willpower
	// Start a minimal casting activity so processFoldRound's "IsCasting"
	// gate passes. Use any spell id that exists in spells data. If no
	// real spell is required for the test path, the caller may stub via
	// the Activity API.
	return c
}

func TestProcessFoldRound_StandingNoCheck(t *testing.T) {
	// Standing should never hit the disruption gate; the function should
	// proceed past it. We can't easily run the full fold without a real
	// spell, so this test only asserts that PositionDisruptionDmgEquiv
	// returns 0 for Standing and that the gate code path skips when 0.
	if got := position.PositionDisruptionDmgEquiv(position.Standing, control.Neutral); got != 0 {
		t.Errorf("Standing should be 0, got %d", got)
	}
}

// TestProcessFoldRound_MountControlled_LowWil_BreaksOften is a smoke test
// over many rolls. A 50-Wil caster in Mount controlled (dmgPctEquiv 60)
// should break most rounds. This is a probabilistic check; allow a wide
// band but verify the direction of the curve.
func TestProcessFoldRound_MountControlled_LowWil_BreaksOften(t *testing.T) {
	// CalcConcentrationChance(50, 60) → base + 50/divisor - 60. Default
	// config has base=40, divisor=4: 40 + 12 - 60 = -8 → clamped to 5.
	// So hold rate is roughly 5%. Across 1000 rolls we should see <= ~10%
	// holds.
	chance := characters.CalcConcentrationChance(50, 60)
	if chance >= 50 {
		t.Errorf("CalcConcentrationChance(50,60) = %d; expected low (<50)", chance)
	}
}

// TestProcessFoldRound_GuardBottom_HighWil_HoldsOften verifies the
// opposite end: a 150-Wil caster in Guard-controller-role (dmgPctEquiv 25,
// the lowest non-Standing value) should hold most rounds.
func TestProcessFoldRound_GuardBottom_HighWil_HoldsOften(t *testing.T) {
	// CalcConcentrationChance(150, 25) → 40 + 150/4 - 25 = 40 + 37 - 25 = 52.
	// Clamped within 5-95. So hold rate ~52%.
	chance := characters.CalcConcentrationChance(150, 25)
	if chance < 40 || chance > 70 {
		t.Errorf("CalcConcentrationChance(150,25) = %d; expected middle band 40-70", chance)
	}
}
```

- [ ] **Step 2: Run the tests and verify they pass on the helpers**

Run: `go test ./internal/hooks/... -run TestProcessFoldRound -v`

These three tests verify the inputs (lookup + curve) without exercising the
full fold-round body, because that requires a live spells data load. The
full per-round behavior is verified in the smoke task (T7-T8). Expected: PASS.

- [ ] **Step 3: Refactor `processFoldRound`**

Open `internal/hooks/combat_shared_helpers.go` and replace lines 426-449 (the
Prone/Supine block + the Grapple catch-all) with a single chance-based check.

**Find this block:**

```go
	// Downed (prone or supine) → immediate concentration break.
	if char.IsProne() || char.IsSupine() {
		clearCastingActivity(char, activity.TriggerConcentrationBreak)
		return FoldRoundResult{ProneBroke: true, CastingData: cs}
	}

	// Chunk 4e T4 — spell disruption audit: grappled casters cannot maintain
	// concentration. All 11 grapple positions (Clinch, BackStanding, Mount,
	// SideControl, KneeOnBelly, NorthSouth, Crucifix, BackGround, HalfGuard,
	// Guard, Turtle) break concentration at the start of every fold round.
	// Per spec §4.2 simplification: grapple implies disruption equivalent to
	// Prone. Per-position disruption curves are chunk 4f territory.
	//
	// Spell disruption audit (chunk 4e T4, 2026-05-19):
	// GAP FOUND: processFoldRound only checked IsProne()/IsSupine(); all 11
	// grapple states fell through to the fold-advance path, letting grappled
	// casters complete spells unimpeded. Added catch-all below treating any
	// grapple state as Prone-equivalent immediate concentration break.
	// checkConcentrationBreak (damage-hit path) has no position filter and
	// fires correctly for all positions — no gap there.
	if char.IsGrappling() {
		clearCastingActivity(char, activity.TriggerConcentrationBreak)
		return FoldRoundResult{GrappleBroke: true, CastingData: cs}
	}
```

**Replace with:**

```go
	// Position-based concentration disruption (chunk 4f). Replaces the
	// three deterministic 100% gates (Prone/Supine/Grapple) that chunks
	// pre-4e shipped. Now: damage%-equivalent per (position, role) →
	// existing CalcConcentrationChance(Wil, dmgPctEquiv) curve → roll.
	// Standing returns 0 and skips the check entirely.
	//
	// The damage-path checkConcentrationBreak still fires independently
	// when damage lands during a round — both paths can break a single
	// cast (layered disruption).
	if char.Position != nil {
		posState := char.Position.State()
		var ctrlState control.State
		if char.Control != nil {
			ctrlState = char.Control.State()
		}
		dmgPctEquiv := position.PositionDisruptionDmgEquiv(posState, ctrlState)
		if dmgPctEquiv > 0 {
			chance := characters.CalcConcentrationChance(
				char.Stats.Willpower.ValueAdj, dmgPctEquiv)
			roll := util.Rand(100)
			util.LogRoll(`PositionConcentration`, roll, chance)
			if roll >= chance {
				// Concentration broke. Route messaging by which break
				// flag the caller expects for this position.
				clearCastingActivity(char, activity.TriggerConcentrationBreak)
				result := FoldRoundResult{CastingData: cs}
				switch {
				case char.IsProne(), char.IsSupine():
					result.ProneBroke = true
				case char.IsGrappling():
					result.GrappleBroke = true
				}
				return result
			}
			// Roll passed — concentration held this round; fold continues.
		}
	}
```

- [ ] **Step 4: Add the new imports**

Open `internal/hooks/combat_shared_helpers.go` and verify the imports
block at the top of the file includes `internal/state/control` and
`internal/state/position`. Look for the existing imports block (lines
3-22 in the current file). Add these two import lines if missing:

```go
	"github.com/GoMudEngine/GoMud/internal/state/control"
	"github.com/GoMudEngine/GoMud/internal/state/position"
```

(The `internal/state/activity` and `internal/util` imports are already
present from existing code.)

- [ ] **Step 5: Build and verify compilation**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 6: Run the hooks package tests**

Run: `go test ./internal/hooks/...`
Expected: PASS — no regression. If a test fails because it relied on the
old deterministic break behavior, investigate before continuing — it may
indicate a real semantic change in caller messaging that needs the
caller's break-flag handling re-examined.

- [ ] **Step 7: Run the full test suite to catch broader regressions**

Run: `go test ./...`
Expected: PASS, or a clear list of breaks that all trace to the
deterministic-break removal (which is intentional). Investigate any test
failure that is NOT about deterministic disruption.

- [ ] **Step 8: Commit**

```bash
git add internal/hooks/combat_shared_helpers.go internal/hooks/combat_shared_helpers_test.go
git commit -m "$(cat <<'EOF'
feat(combat): T2 — chance-based per-position spell disruption

Replace the three deterministic 100% concentration-break gates in
processFoldRound (Prone/Supine/Grapple) with a single chance-based check
that calls PositionDisruptionDmgEquiv + CalcConcentrationChance and rolls.
Standing skips the check. High-Willpower casters get meaningful protection
even in dominant positions; Crucifix-controlled remains brutal but no
longer mathematically impossible.

Damage-path checkConcentrationBreak is unchanged — both paths can break a
single cast in the same round.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Soften helpfile disruption language

**Files:**
- Modify: `_datafiles/world/dogmud/templates/help/grapple.template:113-116`

- [ ] **Step 1: Open the helpfile and locate the disruption line**

Open `_datafiles/world/dogmud/templates/help/grapple.template`. Find
lines 113-116, which currently read:

```
  - Eating and drinking are unavailable until you're free.
  - Spellcasting and other concentration-heavy actions are
    disrupted just as if you were knocked prone.
  - You cannot move from the room until the grapple ends.
```

- [ ] **Step 2: Replace the disruption bullet with Willpower-mediated phrasing**

Use the Edit tool to change the middle bullet only. Replace:

```
  - Spellcasting and other concentration-heavy actions are
    disrupted just as if you were knocked prone.
```

With:

```
  - Spellcasting and other concentration-heavy actions become
    harder when you're on the ground or pinned in a grapple. Your
    Willpower decides how often your concentration holds — a
    strong-willed caster can sometimes finish a spell from
    underneath, while a distracted one rarely manages it.
```

Verify the surrounding bullets are unchanged. Wrap remains 80 chars.

- [ ] **Step 3: Boot the server locally and verify the helpfile reads cleanly**

Run: `go run main.go --config=_datafiles/config.yaml` (or whatever boot
command the project uses) in the background, log in as a test character,
type `help grapple`, and confirm the new wording renders without
formatting damage. Look for any orphaned ANSI tags or wrapping breaks.

Kill the server after verification.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/templates/help/grapple.template
git commit -m "$(cat <<'EOF'
docs(help): T3 — soften grapple-disruption language to Willpower framing

The deterministic prone-equivalent break shipped in chunk 4e is now
chance-based per position with Willpower as the mediator (chunk 4f T2).
Updated the grapple helpfile bullet to reflect the new framing without
exposing numeric values.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Chunk 4a-4f context.md sweep

Walk through every context.md touched by chunks 4a-4f and verify it
accurately reflects the current code. Update where stale, add where
missing, cross-link where useful.

**Files:**
- Modify: `internal/state/position/context.md` (chunks 4a, 4b, 4b-fixup, 4b-fixup-2, 4c, 4d, 4e, 4f)
- Modify: `internal/state/control/context.md` (chunks 4b-fixup-2, 4e, 4f)
- Modify: `internal/state/activity/context.md` (chunk 4e T4 + chunk 4f disruption integration)
- Modify: `internal/hooks/context.md` (chunks 4d, 4e, 4f)
- Modify: `internal/combat/context.md` (chunks 4c, 4e §3 hit modifiers)
- Modify: `internal/characters/context.md` (chunks 4a position predicates, 4b-fixup-2 control predicates)

- [ ] **Step 1: Read every target context.md and inventory drift**

Read each of the six files. For each, write a short note in scratch (or
TodoWrite) describing:
- What chunk-4 work is documented today.
- What is missing or stale (e.g., 4d submission system not yet listed,
  or chunk 4e T4 spell-disruption catch-all described as 100% break
  when it's now chance-based).
- Whether the "Next chunks" or "Status" header section needs updating.

Read in parallel via four Read tool calls:
- `internal/state/position/context.md`
- `internal/state/control/context.md`
- `internal/state/activity/context.md`
- `internal/hooks/context.md`

Then two more:
- `internal/combat/context.md`
- `internal/characters/context.md`

- [ ] **Step 2: Update `internal/state/position/context.md`**

Specifically check:
1. The status header lists chunk 4d as shipped (it is — confirm). Add a
   chunk 4e bullet describing position-tiered hit modifiers
   (TargetSideHitModifier + AttackerSelfHitModifier), eat/drink restrictions,
   outside-damage control degradation, AI tiebreaker, sub interrupt — if
   not already present.
2. Add a chunk 4f bullet for the new `disruption.go` lookup +
   chance-based concentration check.
3. Update the "Next chunks" section: chunk 4 (Position) is now CLOSED;
   next is chunk 5 (Presence).

Use Edit to make targeted changes. Do NOT rewrite the whole file —
only the sections that drift. Keep prose tight.

- [ ] **Step 3: Update `internal/state/control/context.md`**

Specifically check:
1. The chunk 4e relationship: the ControlLevel role now feeds (a) the
   position hit-modifier tables (`modifiers.go`) AND (b) the position
   disruption lookup (`disruption.go`, chunk 4f). Cross-link both.
2. The chunk 4e outside-damage degradation pushes ControlLevel one step
   toward Neutral — confirm this is mentioned, or add it.

- [ ] **Step 4: Update `internal/state/activity/context.md`**

Specifically check:
1. The chunk 4e T4 spell-disruption audit noted that `processFoldRound`
   had a Mount-pinned-caster gap. If the activity context.md documents
   the casting state interactions, update or cross-link to note that
   position-based concentration breaks are now chance-based (chunk 4f).
2. If no chunk-4 content exists in this file today, add a short
   cross-reference paragraph pointing to `internal/hooks/context.md`
   and `internal/state/position/disruption.go`.

- [ ] **Step 5: Update `internal/hooks/context.md`**

Specifically check:
1. The `processFoldRound` documentation must reflect the chance-based
   chunk 4f rewrite, NOT the deterministic chunk 4e T4 catch-all.
2. Cross-link to `internal/state/position/disruption.go` and to
   `characters.CalcConcentrationChance`.
3. Note that damage-path `checkConcentrationBreak` still fires
   independently (layered disruption).
4. If chunk 4e §5 outside-damage hooks and chunk 4e §7 sub interrupt
   are described, leave them alone — those are unchanged in 4f.

- [ ] **Step 6: Update `internal/combat/context.md`**

Specifically check:
1. The chunk 4c reach utility is documented (confirm).
2. Chunk 4e §3 position hit modifiers: `applyPositionHitModifiers` helper
   in `combat/combat.go` composes `AttackerSelfHitModifier` ×
   `TargetSideHitModifier`. Confirm documented, add if missing.

- [ ] **Step 7: Update `internal/characters/context.md`**

Specifically check:
1. Position predicates (chunk 4a): `IsProne`, `IsSupine`, `IsGrappling`,
   `IsTopDominant`, etc. — should be listed.
2. Control predicates (chunk 4b-fixup-2): `IsController`,
   `IsBeingControlled` — should be listed.
3. `CalcConcentrationChance` documentation should mention it is consumed
   by BOTH the damage-path (`checkConcentrationBreak`) AND the
   position-path (chunk 4f `processFoldRound` disruption gate).

- [ ] **Step 8: Build to catch any broken markdown references**

Run: `go build ./...`
Expected: clean — context.md edits do not affect Go compilation, but
this confirms no accidental file rename or broken doc-comment.

- [ ] **Step 9: Commit**

```bash
git add internal/state/position/context.md internal/state/control/context.md \
        internal/state/activity/context.md internal/hooks/context.md \
        internal/combat/context.md internal/characters/context.md
git commit -m "$(cat <<'EOF'
docs(context): T4 — chunk 4a-4f context.md sweep

Walk every context.md touched by chunks 4a through 4f. Update Position
package to list chunks 4e (hit modifiers, eat/drink, third-party hooks,
sub interrupt, AI tiebreaker) and 4f (chance-based concentration
disruption). Update Control package to cross-link the modifier + disruption
consumers. Update Activity + hooks + combat + characters with the same
cross-links and any stale chunk-4 references.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Chunk 4a-4f helpfile coverage audit

Sweep every helpfile touched (or affected) by chunks 4a-4f, verifying
that (a) coverage is current and (b) numerical values are not exposed
except where they satisfy the common-sense exception below.

**SOP applied in this task — the "common-sense exception":**

- **OK to expose numerically:** values a person living in the world
  would naturally know — gold prices on a merchant's sign, item weights
  on a label, round counts when phrased as "lasts a few rounds" or "for
  a moment", physical distances ("five paces", "across the river"),
  the COUNT of something visible ("two daggers"), time-of-day, IDs
  visible on world objects (mob ids in `look`, etc.).
- **NOT OK to expose:** internal mechanical values that an in-world
  person wouldn't know — hit-chance percentages, damage multipliers,
  damage scales, control-shift increments, dmgPctEquiv values, HP /
  stamina / conviction percentages, modifier ranges (`0.50-1.25`),
  drift z-scores, randomness knobs, `RollSpread`, `MitigationCap`,
  config knob names, stat soft-cap formulas, `chance = base + Wil/4 - dmg`.

**Files in scope (chunks 4a-4f surface):**
- `_datafiles/world/dogmud/templates/help/grapple.template` (chunks 4a-4f)
- `_datafiles/world/dogmud/templates/help/submission.template` (chunk 4d) — if it exists
- `_datafiles/world/dogmud/templates/help/surrender.template` (chunk 4d)
- `_datafiles/world/dogmud/templates/help/reach.template` (chunk 4c)
- `_datafiles/world/dogmud/templates/help/prone.template` (chunks 4a, 4f)
- `_datafiles/world/dogmud/templates/help/stand.template` (chunks 4a-4b)
- `_datafiles/world/dogmud/templates/help/trip.template` (chunk 4b writer)
- `_datafiles/world/dogmud/templates/help/bash.template` (chunk 4b writer)
- `_datafiles/world/dogmud/templates/help/cast.template` (chunks 4e T4, 4f T2 disruption)
- `_datafiles/world/dogmud/templates/help/spell.template` (chunk 4f disruption)
- `_datafiles/world/dogmud/templates/help/spells.template` (chunk 4f disruption)
- `_datafiles/world/dogmud/templates/help/attack.template` (chunk 4e §3 hit modifiers)
- `_datafiles/world/dogmud/templates/help/drink.template` (chunk 4e §4 eat/drink)
- `_datafiles/world/dogmud/templates/help/flee.template` (chunk 4a flee block from grapple)

- [ ] **Step 1: Confirm which helpfiles in the scope list exist**

Run: `ls _datafiles/world/dogmud/templates/help/` and verify each file
in the scope list. Note any that don't exist — those go on a "missing
coverage" memory at the end of the task (NOT created here per spec §7
out-of-scope: full helpfile rewrite).

- [ ] **Step 2: Read each existing helpfile and inventory issues**

Use parallel Read calls (4-6 at a time). For each file, log to a
scratch note:
- Any line that exposes a numerical value forbidden by the SOP above
  (cite line number and the offending text).
- Any chunk-4 mechanic that should be covered but isn't (e.g.,
  `attack.template` not mentioning position-tiered hit modifiers at
  all, `drink.template` not mentioning the grapple block).
- Any wording that is stale because of chunk 4f's chance-based
  disruption (e.g., references to deterministic prone-disruption).

- [ ] **Step 3: Sort findings into three buckets**

For each finding:
- **In-scope fix (chunk 4f):** numerical SOP violation that's a one-line
  edit OR stale wording that the chunk 4f rewrite just invalidated.
  Examples: `cast.template` describes spell disruption as a hard break,
  `attack.template` exposes the literal `1.32` Mount net multiplier.
- **Out-of-scope memory:** missing coverage of an entire chunk 4
  mechanic (e.g., `attack.template` has zero position-modifier
  coverage). Per spec §7, "Full helpfile rewrite" is out of scope; log
  a memory and skip.
- **No change needed:** the helpfile is accurate and SOP-compliant.

- [ ] **Step 4: Apply in-scope edits**

For each in-scope fix, use Edit to make a targeted change. Keep prose
tight, wrap at 80 chars, follow the chunk 4f softened-language pattern
from Task 3. Examples:

- If `cast.template` says "Casting in a grapple is interrupted
  automatically", rewrite to "Casting while grappled or downed is
  harder — your Willpower determines how often your concentration
  holds."
- If `attack.template` says "Mount controllers gain a 32% hit-rate
  bonus", rewrite to "Striking down from a dominant position like
  mount or side control is markedly easier than standing combat;
  swinging back from underneath is markedly harder."

Do NOT add NEW sections — only fix what's wrong. Coverage gaps go to
memory.

- [ ] **Step 5: Log out-of-scope coverage gaps as a single followup memory**

If any helpfiles are missing coverage of chunk-4 mechanics (e.g.,
`attack.template` has no position-modifier section at all), write ONE
memory file consolidating the findings. Use the `Write` tool to create:

`C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_chunk_4_helpfile_coverage_gaps.md`

Frontmatter `type: project`, name `chunk-4-helpfile-coverage-gaps`,
short description. Body lists per-file what's missing. Cross-link
`[[combat-text-color-coding]]` if any visual-clarity overlap exists.

Then add a line to `MEMORY.md` (in the Loose Followups table)
pointing at the new file.

- [ ] **Step 6: Boot server and spot-check 2-3 edited helpfiles**

Run: `go run main.go --config=_datafiles/config.yaml` in background.
Log in, type `help grapple`, `help cast`, and one other edited file.
Confirm wrapping and ANSI tags render cleanly. Kill the server.

- [ ] **Step 7: Commit**

```bash
git add _datafiles/world/dogmud/templates/help/*.template
git commit -m "$(cat <<'EOF'
docs(help): T5 — chunk 4a-4f helpfile audit, SOP fixes

Sweep every helpfile touched by chunks 4a through 4f. Fix numerical SOP
violations (no internal mechanical values exposed; common-sense exception
for things a person in the world would know — gold, weights, distances,
round-count flavor). Update stale wording where chunk 4f's chance-based
disruption invalidated deterministic-break phrasing. Out-of-scope coverage
gaps logged as project_chunk_4_helpfile_coverage_gaps.md memory.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Write the comprehensive smoke goals file

**Files:**
- Create: `tools/testing/goals/chunk-4f-position-system-smoke.yaml`

- [ ] **Step 1: Write the goals file**

Create `tools/testing/goals/chunk-4f-position-system-smoke.yaml`:

```yaml
goals:
  - Engage a humanoid grappling-archetype mob and initiate a grapple
    from standing. Verify the engine moves you into a grapple state
    (Clinch or BackStanding) and a position narration appears.

  - Push the grapple to advance: keep attacking and grappling. Look
    for messages indicating you advanced to a more dominant position
    (Mount, SideControl, BackGround, etc.). The roadmap calls Mount
    the "striking apex"; if you reach it, note the round count.

  - Once you reach a dominant position, verify the round-by-round hit
    rate against the controlled defender is meaningfully higher than
    your hit rate before the grapple began. Watch the swing narration:
    a controller in Mount, SideControl, or BackGround should land
    more often than a standing attacker would. If you have
    combatstats, sample it; otherwise observe.

  - Try to eat any food item while grappled (use `eat <food>`). Verify
    the command is rejected with a flavor message about hands being
    committed to the grapple. Do the same for `drink <potion>`.

  - Try to flee from a grappled position. Verify it remains blocked
    with appropriate messaging.

  - Cast a damaging or buff spell while grappled in a controller
    position (e.g., Mount-Controlling, BackGround-Controlling) and
    note whether the spell completes or whether your concentration
    breaks before completion. Try multiple casts — chunk 4f makes
    this a per-round chance roll mediated by Willpower, so behavior
    should be VARIABLE: sometimes you complete, sometimes you break.
    If your character has high Willpower, completion is more likely.

  - Have a low-Willpower character (if available, or create one) be
    pinned in Mount-Controlled and attempt to cast. Verify concentration
    breaks reliably (high disruption + low Wil → very low hold chance).
    Sometimes still holds — chance roll, not deterministic.

  - Try casting from a Guard-bottom position (you should be in
    Controlling role, even though you're underneath geometrically).
    Verify casting is the LEAST disrupted of any grapple position —
    you should complete spells more often here than from any
    Controlled-role position.

  - If you can engineer a third-party scenario (spawn a second hostile
    in the room), observe:
      (a) Does the third party hit the grappled target more easily
          than they'd hit a standing target? (target-side bonus)
      (b) Does the controller's grip seem to loosen when outside
          damage lands? (chunk 4e §5 outside-damage drift — look for
          flavor like "your control slips" or a ControlLevel gradient
          message)
      (c) If you reach a submission attempt and outside damage hits
          during the sub-firing round, does the sub fail / land in
          the Bad tier? (chunk 4e §7 sub interrupt)

  - With a second mob in the room and you grappled with the first,
    observe whether nearby mobs gain interest in attacking the
    controlled defender or stay on their original target. The AI
    tiebreaker (chunk 4e §6) prefers grappled-controlled targets
    when priorities are close, but does NOT override clear primary
    preferences.

  - Read `help grapple` and confirm the disruption language (chunk
    4f T3) reads naturally: "harder when you're on the ground or
    pinned … Willpower decides how often your concentration holds".
    No mention of percentages, multipliers, or other internal
    values.

  - Read `help submission` and `help surrender` (if they exist).
    Verify no numerical leaks from chunks 4d-4f.

  - Read `help cast` and `help spells`. Verify no stale references
    to "spells fail when grappled" or "always disrupted by prone"
    or other deterministic-break wording.

  - Report: no panics, no missing-template debug strings (e.g.,
    `[[missing template: grapple_outcome_advance_mount]]`), no
    "unexpected position state" log spam, no zero-config or nil-
    pointer issues, no double-message bugs (same outcome narrated
    twice in one round).
```

- [ ] **Step 2: Commit**

```bash
git add tools/testing/goals/chunk-4f-position-system-smoke.yaml
git commit -m "$(cat <<'EOF'
test(goals): T6 — chunk 4f comprehensive position-system smoke goals

Goals file for the two-pass AI smoke (feature-tester + feel-tester) that
verifies every chunk-4 deliverable end-to-end: grapple entry + position
advancement + hit modifiers + eat/drink restrictions + Willpower-mediated
spell disruption (4f) + third-party hooks + AI tiebreaker + sub interrupt
+ helpfile wording sanity.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: Feature-tester smoke pass

**Files:**
- Read: `tools/testing/goals/chunk-4f-position-system-smoke.yaml`
- Output: `tools/testing/reports/YYYY-MM-DD-local-feature-tester-chunk-4f-position-system-smoke.md`

- [ ] **Step 1: Boot the server locally and confirm clean startup**

Run the boot command the project uses (typically `go build` then
`./dogmud.exe` on Windows or `go run main.go`). Watch for:
- `mobs.LoadDataFiles() loadedCount=...` (no panic)
- `quests.LoadDataFiles() loadedCount=...`
- `items.LoadDataFiles() loadedCount=...`
- `rooms.LoadDataFiles() loadedCount=...`

If any panic fires, STOP, fix the panic, and re-boot. Do not proceed
to AI smoke against a broken server.

- [ ] **Step 2: Run the feature-tester pass**

Invoke `/test-mud local feature-tester chunk-4f-position-system-smoke.yaml`.

The harness will write a report to `tools/testing/reports/` when done.

Hard caps to avoid runaway sessions:
- Max 60 commands.
- Max 15 wall minutes.

If the report doesn't appear within 20 minutes, kill the bridge process
(`taskkill /F /IM python.exe` or `pkill -f mud_bridge.py`) and inspect
`tools/mud_output.txt` for what happened. Re-dispatch with a tighter
scope if needed.

- [ ] **Step 3: Kill any lingering test-server processes**

Per the SOP in MEMORY.md ([[kill-test-servers]]): after each chunk's
smoke pass, clean up `dogmud*.exe` and `go run` processes.

```bash
taskkill /F /IM dogmud.exe 2>nul
taskkill /F /IM go.exe 2>nul
```

(On Linux: `pkill -f dogmud`; `pkill -f mud_bridge`.)

- [ ] **Step 4: Read the report and classify findings**

Read the report file from `tools/testing/reports/`. Apply the spec §4.3
classification:

- **Critical bug (regression from a prior chunk):** add a task to Task 9
  (react pass). Track in TodoWrite as `chunk4f-critical-<short-id>`.
- **Tuning-want:** log as a followup memory in Task 9.
- **New feature suggestion:** log as a followup memory in Task 9.
- **Helpfile gap:** log as a followup memory unless trivial.
- **PASS:** record in TodoWrite as "feature-tester verified <feature>".

Do NOT yet act on findings — collect both passes first.

- [ ] **Step 5: Commit the report**

```bash
git add tools/testing/reports/*chunk-4f-position-system-smoke*.md
git commit -m "$(cat <<'EOF'
test(reports): T7 — chunk 4f feature-tester smoke report

Feature-tester pass against the comprehensive position-system smoke
goals. See report for findings classification.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 8: Feel-tester smoke pass

**Files:**
- Read: `tools/testing/goals/chunk-4f-position-system-smoke.yaml`
- Output: `tools/testing/reports/YYYY-MM-DD-local-feel-tester-chunk-4f-position-system-smoke.md`

- [ ] **Step 1: Re-boot the server cleanly**

Same as Task 7 Step 1. Confirm no panics.

- [ ] **Step 2: Run the feel-tester pass**

Invoke `/test-mud local feel-tester chunk-4f-position-system-smoke.yaml`.

Same hard caps: 60 commands, 15 wall minutes. Feel-tester output is
about variety, viscerality, coherence, and pacing — qualitative.

- [ ] **Step 3: Kill any lingering test-server processes**

Same cleanup as Task 7 Step 3.

- [ ] **Step 4: Read the report and classify findings**

Apply the spec §4.3 classification, same as Task 7 Step 4. Add to the
TodoWrite collection.

- [ ] **Step 5: Commit the report**

```bash
git add tools/testing/reports/*chunk-4f-position-system-smoke*.md
git commit -m "$(cat <<'EOF'
test(reports): T8 — chunk 4f feel-tester smoke report

Feel-tester pass against the comprehensive position-system smoke goals.
See report for variety/viscerality/coherence/pacing notes.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 9: React to smoke findings — critical bugs only

This task's scope is determined by Task 7 + Task 8's output. The shape
of this task is fixed (the steps below). The CONTENT (which specific
fixes get made) is determined by the smoke reports.

**Files:** TBD by smoke findings, but expected modifications:
- Possibly: `internal/state/position/modifiers.go` (if a hit-modifier
  value smoke surfaces as a clear regression).
- Possibly: `internal/state/position/disruption.go` (if the chunk 4f
  disruption table itself needs an adjustment from smoke).
- Possibly: `internal/configs/config.balance.go` knob defaults.
- New memory files in `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\` for tuning-wants + feature ideas + non-critical findings.

- [ ] **Step 1: Build the findings classification table**

Open both reports (Task 7 + Task 8). For each finding, put it in one
bucket:

| Bucket | Action |
|---|---|
| Critical bug (regression) | Fix in this task |
| Critical bug (new) | Fix in this task |
| Tuning-want (single signal) | Memory only — do NOT fix |
| Tuning-want (multiple findings on same value) | Discuss — may be elevated |
| New feature idea | Memory only |
| Helpfile gap (already audited in T5) | Memory only |
| Already-known issue (already in MEMORY.md) | No action |
| PASS | Note in roadmap update (T10) |

- [ ] **Step 2: Fix each critical-bug finding**

For each critical-bug finding, follow the standard TDD-style mini-flow:
1. Write a failing test that reproduces the bug.
2. Fix the underlying code.
3. Verify the test now passes.
4. Run `go test ./...` for the affected package.
5. Commit with a `fix(...)` prefix referencing the smoke report.

The plan can't specify the test code in advance because the bug shape
is unknown. Use the test conventions visible in existing test files in
the affected package as a template.

If a critical-bug fix is non-trivial (touches > 3 files OR > 50 LoC),
STOP. Do not roll it into chunk 4f. Instead, write a memory describing
the bug and the fix scope and ESCALATE to the human partner. Chunk 4f's
scope is "react to critical findings only with bounded fixes" per the
spec — large fixes get their own chunk.

- [ ] **Step 3: Log non-critical findings as followup memories**

For each tuning-want / feature idea / helpfile gap, write a memory file
in `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\`.

Naming convention: `project_chunk_4_smoke_<short_id>.md`. Body uses
the standard project-memory shape (Why / How to apply / Where it
surfaced).

Then add a single line to `MEMORY.md`'s "Loose Followups" table
pointing at the new file.

Use the `Write` tool. Multiple small memories are better than one
sprawling one — each tuning-want stands alone.

- [ ] **Step 4: Re-run the relevant package tests after fixes**

If Step 2 fixes touched Go code:

```bash
go build ./...
go test ./...
```

Both must pass before moving to Task 10.

- [ ] **Step 5: Commit the followup-memory index changes**

```bash
git add C:/Users/Calabe\ Davis/.claude/projects/.../memory/MEMORY.md \
        C:/Users/Calabe\ Davis/.claude/projects/.../memory/project_chunk_4_smoke_*.md
```

(Adjust path quoting for your shell. PowerShell uses different quoting
than bash.)

Commit message:

```bash
git commit -m "$(cat <<'EOF'
docs(memory): T9 — chunk 4f smoke followups logged

Non-critical findings from chunk 4f's two-pass smoke logged as project
memories for future polish. Critical bugs (if any) fixed in this chunk
and tracked via separate fix(...) commits.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 10: COMBAT_STATE_ROADMAP row + chunk 4f close-out

**Files:**
- Modify: `COMBAT_STATE_ROADMAP.md:47` (the chunk 4f row)
- Modify: `PATCH_NOTES.md` (top of file — dated entry)

- [ ] **Step 1: Update the chunk 4f row in `COMBAT_STATE_ROADMAP.md`**

Open `COMBAT_STATE_ROADMAP.md` and find line 47:

```markdown
| 4f | Position — balance + smoke | Not started | Tune modifiers, write position-flavor text, full-stack combat smoke. |
```

Replace with a "Done" row. The text should mention: the chance-based
position-spell-disruption rewrite, the two-pass smoke, the context.md
+ helpfile audit, and the count of followup memories generated. Use
the same prose style as chunks 4d and 4e on the same table.

Example (count of memories is determined by Task 9 output — fill in
the actual number from your TodoWrite):

```markdown
| 4f | Position — balance + smoke | Done (2026-05-19) | Replaced the deterministic Prone/Supine/Grapple 100% spell-disruption gates in `processFoldRound` with a chance-based per-position check fed through the existing `CalcConcentrationChance(Wil, dmgPctEquiv)` curve; new `internal/state/position/disruption.go` lookup with damage%-equivalents 25-70 across the 13 grapple/downed positions (Standing returns 0). Guard inverts (bottom-controller lowest at 25). Damage-path `checkConcentrationBreak` unchanged — both paths fire layered. Comprehensive two-pass smoke (feature-tester + feel-tester) verified the position-system end-to-end; helpfile language softened from "disrupted just as if knocked prone" to a Willpower-mediated framing. Context.md sweep across position/control/activity/hooks/combat/characters packages. Helpfile audit across the 14 chunk-4-relevant templates fixed N numerical-leak violations and logged M coverage-gap memories. Chunk 4 (Position) closed. |
```

Adjust N + M to actual counts from Tasks 5 and 9. If no critical bugs
were fixed, the row reads as above. If critical bugs WERE fixed, append
a clause: "Smoke surfaced and fixed K critical regressions (see commits)."

- [ ] **Step 2: Update `PATCH_NOTES.md`**

Per CLAUDE.md's "Pre-Push SOP", patch notes get a dated entry before
pushing to prod. This task isn't pushing yet, but the entry should be
added now so the entire chunk 4f surface is captured in one place.

Open `PATCH_NOTES.md`, add a top entry under today's date:

```markdown
## 2026-05-19 — Chunk 4f: Position balance + smoke

**Spell disruption in grapples is now Willpower-mediated.** Previously,
being knocked prone or grappled automatically broke any spell you were
casting. Now, your Willpower mediates a per-round concentration check
— a strong-willed caster can sometimes complete a spell from
underneath, while a distracted one rarely will. The hardest positions
(crucifix, back mount) remain brutal disruptors; the most lenient
(guard from underneath, where your hands are free) gives high-Wil
casters a real fighting chance.

**Comprehensive position-system smoke** across grapple entry,
advancement, dominant-position striking, eat/drink restrictions,
third-party hooks, AI tiebreaker, submission interrupt, and helpfile
language. Followup polish items logged for future chunks.

**Helpfile coverage audit** across grapple, cast, attack, submission,
and related help topics. Removed mechanical-value leaks; tightened
language wherever chunk 4f's chance-based disruption invalidated older
wording.
```

Keep prose tight, no internal mechanical numbers.

- [ ] **Step 3: Verify build + tests one final time**

```bash
go build ./...
go test ./...
```

Expected: PASS across the board.

- [ ] **Step 4: Boot the server one last time to verify clean startup**

Per CLAUDE.md "Pre-Push SOP" — boot the server locally, watch for
`mobs.LoadDataFiles() loadedCount=...`,
`quests.LoadDataFiles() loadedCount=...`, etc., confirm no panics
during data-file loading.

Kill the server after verification.

- [ ] **Step 5: Commit roadmap + patch notes**

```bash
git add COMBAT_STATE_ROADMAP.md PATCH_NOTES.md
git commit -m "$(cat <<'EOF'
docs(roadmap): T10 — chunk 4f done, position chunk closed

Update COMBAT_STATE_ROADMAP to mark chunk 4f as Done. Patch note dated
entry describing the Willpower-mediated spell disruption, the full
position-system smoke, and the helpfile audit. Chunk 4 (Position) is
now closed; next is chunk 5 (Presence).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Verification Checklist

Before declaring chunk 4f complete:

- [ ] `go build ./...` clean.
- [ ] `go test ./...` clean.
- [ ] `internal/state/position/disruption.go` exists with 14-cell table.
- [ ] `internal/state/position/disruption_test.go` covers every cell + invariants.
- [ ] `internal/hooks/combat_shared_helpers.go:processFoldRound` no longer has the three deterministic 100% gates; uses `position.PositionDisruptionDmgEquiv` + `characters.CalcConcentrationChance` + `util.Rand` + `util.LogRoll`.
- [ ] `_datafiles/world/dogmud/templates/help/grapple.template` reads with the Willpower-mediated framing.
- [ ] All six context.md files updated (Task 4).
- [ ] All chunk 4a-4f-relevant helpfiles audited (Task 5) and SOP-compliant.
- [ ] Two smoke reports exist in `tools/testing/reports/`.
- [ ] Critical-bug fixes (if any) committed with `fix(...)` prefix.
- [ ] Followup memories written for non-critical findings.
- [ ] `MEMORY.md` updated with pointers to new followup memories.
- [ ] `COMBAT_STATE_ROADMAP.md` chunk 4f row marked Done.
- [ ] `PATCH_NOTES.md` dated entry added.
- [ ] Test-server processes killed.

---

## Estimated effort

- T1 (lookup + tests): 30-45 min
- T2 (processFoldRound refactor): 30-45 min
- T3 (helpfile softening): 10 min
- T4 (context.md sweep): 45-60 min
- T5 (helpfile audit): 60-90 min (the bulkiest non-smoke task)
- T6 (goals file): 15 min
- T7 (feature-tester smoke): 20-30 min wall time (most spent waiting on the AI run)
- T8 (feel-tester smoke): 20-30 min wall time
- T9 (react to findings): 30 min to 3 hours depending on critical-bug volume
- T10 (roadmap + patch notes): 15 min

Total: ~4-7 hours, dominated by T5 audit and T9 react.

---

## Out of Scope (Reminders)

From spec §7:
- Modifier value tuning for chunk 4e tables, grapple-drift coefficients, ControlLevel thresholds — UNLESS smoke surfaces critical issues.
- Per-archetype AI bias variation.
- Full helpfile rewrite (only fix what's wrong + log gaps as memories).
- Position-flavor template authoring beyond the disruption-language softening.
- Sub-tier alpha tweaks.
- `CalcConcentrationChance` rewrite — reuse as-is.
- Combat damage formula changes.

If a smoke finding falls into one of the above categories, it becomes a
followup memory in Task 9, NOT a chunk 4f task — no exceptions.
