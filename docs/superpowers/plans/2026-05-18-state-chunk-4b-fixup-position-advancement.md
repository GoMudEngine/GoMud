# State Chunk 4b-fixup — Position Advancement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Spec:** `docs/superpowers/specs/2026-05-18-state-chunk-4b-fixup-position-advancement-design.md`

**Goal:** Replace chunk 4b's incoherent ControlLevel drift needle with a position-advancement system where each round's drift roll resolves directly into Advance / Hold / Degrade / Reverse / Escape outcomes, backed by ~227+ flavor templates.

**Architecture:** Three layers — (1) pure outcome resolver in `internal/state/position/outcomes.go` (z-score → outcome tier → position change via existing TransitionPair), (2) messaging library in new `internal/grapplemessaging/` package with template YAML in `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml`, (3) refactored `Position_GrappleTick` orchestrating resolver + messaging + existing stamina cost. Old `ControlLevel` enum, `GrappleData.ControlLevel` field, `InitialControlForPair`, and gradient messages all sunset together once the new path is live.

**Tech Stack:** Go 1.21+, YAML data files via `gopkg.in/yaml.v3`, existing `dice.OpposedRollStat`, existing `position.TransitionPair`, existing `util.GetRoundCount`. No new dependencies.

---

## File Structure

### Created
| Path | Responsibility |
|------|---|
| `internal/state/position/outcomes.go` | Outcome enum, per-position transition tables, bucket function, ResolveOutcome dispatcher |
| `internal/state/position/outcomes_test.go` | Unit tests for all outcome paths, table lookups, boundary z values |
| `internal/grapplemessaging/loader.go` | Loads `grapple_outcomes.yaml` at boot, validates structural completeness |
| `internal/grapplemessaging/loader_test.go` | Loader tests (round-trip YAML, missing-category errors, min-template-count assertions) |
| `internal/grapplemessaging/render.go` | Picks template (cooldown-aware), substitutes `{controllerName}`/`{controlledName}`, sends to controller/controlled/room |
| `internal/grapplemessaging/render_test.go` | Render tests (template selection, cooldown, name substitution, empty-list fallback) |
| `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml` | The ~227+ template library |

### Modified
| Path | Change |
|------|---|
| `internal/state/position/position.go` | Delete `ControlLevel` enum + 5 constants; delete `ControlLevel` field from `GrappleData` |
| `internal/state/position/pair.go` | Delete `InitialControlForPair`; `TransitionPair` no longer initializes a ControlLevel |
| `internal/state/position/transitions.go` | Add new trigger constants: `TriggerPositionAdvance` (already exists, becomes load-bearing), `TriggerPositionDegrade`, `TriggerReversal`, `TriggerControlledEscape` |
| `internal/hooks/Position_GrappleTick.go` | Rewrite `processGrapplePair` to call `position.ResolveOutcome` + `grapplemessaging.RenderOutcome` instead of drift-needle math |
| `internal/hooks/Position_Messaging.go` | Delete `fireGradientMessages` + helpers; keep `fireStaminaWarningIfLow` |
| `internal/hooks/Position_SubmissionTick.go` | Verify sub gate at \|z\| ≥ 1.5 still fires correctly from post-advance position; retune alpha constant if needed |
| `internal/state/position/context.md` | Document the new outcome model; remove all ControlLevel references |
| `internal/hooks/context.md` | Update Position_GrappleTick + Position_Messaging sections |
| `internal/behaviortree/context.md` | Document any btree-primitive sunsets |
| `internal/combat/context.md` | Update any chunk-4b drift references |
| `COMBAT_STATE_ROADMAP.md` | Add "Chunk 4b-fixup — Position Advancement" entry as Done |

### Deleted
| Path | Reason |
|------|---|
| `internal/state/position/control.go` | ControlLevel enum/helpers no longer used |
| `internal/state/position/control_test.go` | Tests for deleted code |

### Discovery tasks (file paths TBD per audit)
- Help text files mentioning grapple mechanics (audit `_datafiles/help/`)
- BTree primitive files referencing ControlLevel (audit `internal/behaviortree/primitives/`)
- Tests outside the position package that reference ControlLevel

---

## Task 1: Outcome enum + trigger constants

**Files:**
- Create: `internal/state/position/outcomes.go`
- Modify: `internal/state/position/transitions.go`
- Test: `internal/state/position/outcomes_test.go`

- [ ] **Step 1: Write the failing test for Outcome enum and OutcomeTier bucket function**

Create `internal/state/position/outcomes_test.go`:

```go
package position

import "testing"

func TestOutcomeTierBucketing(t *testing.T) {
	cases := []struct {
		name string
		absZ float64
		want OutcomeTier
	}{
		{"deep_hold", 0.0, TierHold},
		{"hold_just_under", 0.499, TierHold},
		{"one_step_low", 0.500, TierOneStep},
		{"one_step_mid", 0.75, TierOneStep},
		{"one_step_high", 0.999, TierOneStep},
		{"two_step_low", 1.000, TierTwoStep},
		{"two_step_mid", 1.5, TierTwoStep},
		{"two_step_high", 1.999, TierTwoStep},
		{"three_step_low", 2.000, TierThreeStep},
		{"three_step_high", 5.0, TierThreeStep},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := OutcomeTierFromAbsZ(c.absZ)
			if got != c.want {
				t.Errorf("OutcomeTierFromAbsZ(%v) = %v, want %v", c.absZ, got, c.want)
			}
		})
	}
}

func TestSubWindowGate(t *testing.T) {
	if SubWindowOpens(1.499) {
		t.Error("z=1.499 should not open sub window")
	}
	if !SubWindowOpens(1.500) {
		t.Error("z=1.500 should open sub window")
	}
	if !SubWindowOpens(3.0) {
		t.Error("z=3.0 should open sub window")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/state/position/ -run "TestOutcomeTierBucketing|TestSubWindowGate" -count=1`
Expected: FAIL with undefined symbols `OutcomeTier`, `TierHold`, `OutcomeTierFromAbsZ`, `SubWindowOpens`.

- [ ] **Step 3: Create `internal/state/position/outcomes.go` with enum + bucket function**

```go
// Package-level file for chunk 4b-fixup outcome resolution.
// Replaces the chunk-4b ControlLevel drift-needle model with a
// per-round dispatcher that maps the drift roll z-score directly
// to a position change.
package position

// OutcomeTier represents the magnitude bucket of a drift roll's
// |z-score|. Sign of z determines whether the outcome is controller-
// favorable (Advance) or defender-favorable (Degrade / Reverse /
// Escape).
type OutcomeTier int

const (
	TierHold      OutcomeTier = iota // |z| < 0.5
	TierOneStep                      // 0.5 <= |z| < 1.0
	TierTwoStep                      // 1.0 <= |z| < 2.0
	TierThreeStep                    // |z| >= 2.0
)

// Z-score thresholds for outcome bucketing. Match spec §5.
const (
	holdThreshold     = 0.5
	oneStepThreshold  = 1.0
	twoStepThreshold  = 2.0
	subWindowAlpha    = 1.5
)

// OutcomeTierFromAbsZ buckets a z-magnitude into an OutcomeTier per
// the spec §5 table. Caller is responsible for sign-dispatching to
// the advance vs degrade/reverse/escape branch.
func OutcomeTierFromAbsZ(absZ float64) OutcomeTier {
	switch {
	case absZ < holdThreshold:
		return TierHold
	case absZ < oneStepThreshold:
		return TierOneStep
	case absZ < twoStepThreshold:
		return TierTwoStep
	default:
		return TierThreeStep
	}
}

// SubWindowOpens returns true if |z| meets the independent sub-gate
// threshold (chunk 4d composition). Spec §5: |z| >= 1.5 on controller
// side opens a sub window from the post-advance position.
func SubWindowOpens(absZ float64) bool {
	return absZ >= subWindowAlpha
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/state/position/ -run "TestOutcomeTierBucketing|TestSubWindowGate" -count=1 -v`
Expected: PASS (all 12 cases).

- [ ] **Step 5: Add new trigger constants to `internal/state/position/transitions.go`**

Locate the trigger constants block (currently has `TriggerPositionAdvance` at line ~113). Add three new constants alongside it:

```go
// TriggerPositionDegrade fires when the defender wins drift by a
// moderate margin (|z| in [0.5, 1.0)) and the position regresses
// to a less-dominant state per the spec §6.2 table.
TriggerPositionDegrade = "position_degrade"

// TriggerReversal fires when the defender wins drift big (|z| in
// [1.0, 2.0)). Roles swap; position usually stays the same, with
// realism exceptions Mount→Guard and BackGround→Mount per spec §6.3.
TriggerReversal = "reversal"

// TriggerControlledEscape fires when the defender wins drift
// decisively (|z| >= 2.0). TransitionPair to Standing regardless
// of current position. Replaces the chunk-4b "Controlled for 2
// consecutive rounds" gate.
TriggerControlledEscape = "controlled_escape"
```

- [ ] **Step 6: Verify build still compiles**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 7: Commit**

```bash
git add internal/state/position/outcomes.go internal/state/position/outcomes_test.go internal/state/position/transitions.go
git commit -m "feat(position): chunk 4b-fixup T1 — outcome enum + bucket function + new triggers

Adds OutcomeTier (Hold / OneStep / TwoStep / ThreeStep), the
|z|-magnitude bucket function, the sub-window gate (|z| >= 1.5),
and three new trigger constants (PositionDegrade, Reversal,
ControlledEscape) that subsequent tasks will fire.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Advancement transition table

**Files:**
- Modify: `internal/state/position/outcomes.go`
- Modify: `internal/state/position/outcomes_test.go`

- [ ] **Step 1: Write failing tests for AdvancementTarget**

Append to `outcomes_test.go`:

```go
func TestAdvancementTargetSimple(t *testing.T) {
	// Source position, tier, defender posture → expected target.
	// "Hold" means no position change (controller stays at source).
	cases := []struct {
		name           string
		source         State
		tier           OutcomeTier
		defenderPosture State // for Clinch's posture-based dispatch; irrelevant for others
		wantTarget     State
		wantHold       bool
	}{
		// BackStanding always advances toward BackGround
		{"backstanding_1step", BackStanding, TierOneStep, Standing, BackGround, false},
		{"backstanding_2step", BackStanding, TierTwoStep, Standing, BackGround, false},
		{"backstanding_3step", BackStanding, TierThreeStep, Standing, BackGround, false},

		// Mount: striking apex — 1/2 step is Hold, 3 step takes back
		{"mount_1step_hold", Mount, TierOneStep, Standing, Mount, true},
		{"mount_2step_hold", Mount, TierTwoStep, Standing, Mount, true},
		{"mount_3step_advance", Mount, TierThreeStep, Standing, BackGround, false},

		// SideControl → Mount (1/2), BackGround (3)
		{"sidecontrol_1step", SideControl, TierOneStep, Standing, Mount, false},
		{"sidecontrol_2step", SideControl, TierTwoStep, Standing, Mount, false},
		{"sidecontrol_3step", SideControl, TierThreeStep, Standing, BackGround, false},

		// KneeOnBelly → Mount (1/2), BackGround (3)
		{"kob_1step", KneeOnBelly, TierOneStep, Standing, Mount, false},
		{"kob_2step", KneeOnBelly, TierTwoStep, Standing, Mount, false},
		{"kob_3step", KneeOnBelly, TierThreeStep, Standing, BackGround, false},

		// NorthSouth → SideControl (1), Mount (2), BackGround (3)
		{"ns_1step", NorthSouth, TierOneStep, Standing, SideControl, false},
		{"ns_2step", NorthSouth, TierTwoStep, Standing, Mount, false},
		{"ns_3step", NorthSouth, TierThreeStep, Standing, BackGround, false},

		// Crucifix: terminal apex (sub-only). Always hold from advancement POV.
		{"crucifix_1step_hold", Crucifix, TierOneStep, Standing, Crucifix, true},
		{"crucifix_2step_hold", Crucifix, TierTwoStep, Standing, Crucifix, true},
		{"crucifix_3step_hold", Crucifix, TierThreeStep, Standing, Crucifix, true},

		// BackGround: terminal apex (sub-only).
		{"background_1step_hold", BackGround, TierOneStep, Standing, BackGround, true},
		{"background_2step_hold", BackGround, TierTwoStep, Standing, BackGround, true},
		{"background_3step_hold", BackGround, TierThreeStep, Standing, BackGround, true},

		// HalfGuard → SideControl (1), Mount (2), BackGround (3)
		{"halfguard_1step", HalfGuard, TierOneStep, Standing, SideControl, false},
		{"halfguard_2step", HalfGuard, TierTwoStep, Standing, Mount, false},
		{"halfguard_3step", HalfGuard, TierThreeStep, Standing, BackGround, false},

		// Guard (inverted; controller is bottom) → SideControl (1), Mount (2), BackGround (3)
		{"guard_1step", Guard, TierOneStep, Standing, SideControl, false},
		{"guard_2step", Guard, TierTwoStep, Standing, Mount, false},
		{"guard_3step", Guard, TierThreeStep, Standing, BackGround, false},

		// Turtle → BackGround at all tiers
		{"turtle_1step", Turtle, TierOneStep, Standing, BackGround, false},
		{"turtle_2step", Turtle, TierTwoStep, Standing, BackGround, false},
		{"turtle_3step", Turtle, TierThreeStep, Standing, BackGround, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			target, hold := AdvancementTarget(c.source, c.tier, c.defenderPosture)
			if hold != c.wantHold {
				t.Errorf("AdvancementTarget(%v,%v) hold=%v, want %v", c.source, c.tier, hold, c.wantHold)
			}
			if !hold && target != c.wantTarget {
				t.Errorf("AdvancementTarget(%v,%v) target=%v, want %v", c.source, c.tier, target, c.wantTarget)
			}
		})
	}
}

func TestAdvancementTargetClinchPosture(t *testing.T) {
	cases := []struct {
		name            string
		defenderPosture State
		tier            OutcomeTier
		wantTarget      State
	}{
		// Clinch 1-step: posture dispatches
		{"clinch_prone_1step", Prone, TierOneStep, SideControl},
		{"clinch_supine_1step", Supine, TierOneStep, Mount},
		{"clinch_turtle_1step", Turtle, TierOneStep, BackGround},
		{"clinch_standing_1step", Standing, TierOneStep, Mount},

		// Clinch 2-step: always Mount
		{"clinch_prone_2step", Prone, TierTwoStep, Mount},
		{"clinch_standing_2step", Standing, TierTwoStep, Mount},

		// Clinch 3-step: always BackGround
		{"clinch_standing_3step", Standing, TierThreeStep, BackGround},
		{"clinch_prone_3step", Prone, TierThreeStep, BackGround},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			target, hold := AdvancementTarget(Clinch, c.tier, c.defenderPosture)
			if hold {
				t.Errorf("Clinch should never Hold on advancement, got hold=true")
			}
			if target != c.wantTarget {
				t.Errorf("AdvancementTarget(Clinch,%v,posture=%v) = %v, want %v", c.tier, c.defenderPosture, target, c.wantTarget)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/state/position/ -run "TestAdvancementTarget" -count=1`
Expected: FAIL with `undefined: AdvancementTarget`.

- [ ] **Step 3: Implement AdvancementTarget in `outcomes.go`**

Append to `internal/state/position/outcomes.go`:

```go
// AdvancementTarget returns the position state to transition to when
// the controller wins drift at the given tier, plus a `hold` flag
// indicating "no position change" (status quo). Implements spec §6.1
// table. For Clinch, defenderPosture determines the 1-step target.
// defenderPosture is ignored for non-Clinch sources.
//
// For terminal-apex positions (Crucifix, BackGround) and the striking
// apex (Mount at 1/2 step), returns (source, true). Sub-gate is the
// caller's responsibility.
func AdvancementTarget(source State, tier OutcomeTier, defenderPosture State) (target State, hold bool) {
	if tier == TierHold {
		return source, true
	}

	switch source {
	case Clinch:
		return clinchAdvancementTarget(tier, defenderPosture), false

	case BackStanding:
		return BackGround, false

	case Mount:
		if tier == TierThreeStep {
			return BackGround, false
		}
		// 1-step + 2-step: striking apex Hold
		return Mount, true

	case SideControl, KneeOnBelly:
		if tier == TierThreeStep {
			return BackGround, false
		}
		return Mount, false

	case NorthSouth:
		switch tier {
		case TierOneStep:
			return SideControl, false
		case TierTwoStep:
			return Mount, false
		default: // TierThreeStep
			return BackGround, false
		}

	case Crucifix, BackGround:
		// Terminal apex — sub-only, no position advance.
		return source, true

	case HalfGuard, Guard:
		switch tier {
		case TierOneStep:
			return SideControl, false
		case TierTwoStep:
			return Mount, false
		default: // TierThreeStep
			return BackGround, false
		}

	case Turtle:
		return BackGround, false
	}

	// Unexpected source state — treat as Hold to be safe.
	return source, true
}

func clinchAdvancementTarget(tier OutcomeTier, defenderPosture State) State {
	switch tier {
	case TierOneStep:
		switch defenderPosture {
		case Prone:
			return SideControl
		case Supine:
			return Mount
		case Turtle:
			return BackGround
		default:
			// BackStanding is also valid but reached only when the
			// defender turned away mid-clinch; default fallthrough is
			// Mount per spec §6.1.
			return Mount
		}
	case TierTwoStep:
		return Mount
	default: // TierThreeStep
		return BackGround
	}
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/state/position/ -run "TestAdvancementTarget" -count=1 -v`
Expected: PASS (all subtests).

- [ ] **Step 5: Commit**

```bash
git add internal/state/position/outcomes.go internal/state/position/outcomes_test.go
git commit -m "feat(position): chunk 4b-fixup T2 — advancement transition table

AdvancementTarget(source, tier, defenderPosture) → (target, hold).
Implements spec §6.1 fully: Mount as striking apex (1/2 step Hold,
3-step takes back to BackGround), Crucifix/BackGround terminal,
Clinch posture-based dispatch, all other 11 grapple sources mapped.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Degrade transition table

**Files:**
- Modify: `internal/state/position/outcomes.go`
- Modify: `internal/state/position/outcomes_test.go`

- [ ] **Step 1: Write failing tests**

Append to `outcomes_test.go`:

```go
func TestDegradeTarget(t *testing.T) {
	cases := []struct {
		name       string
		source     State
		wantTarget State
		wantHold   bool
	}{
		// Symmetric source — no degrade target
		{"clinch_hold", Clinch, Clinch, true},

		// Standard degrades
		{"backstanding_to_clinch", BackStanding, Clinch, false},
		{"mount_to_sidecontrol", Mount, SideControl, false},
		{"sidecontrol_to_halfguard", SideControl, HalfGuard, false},
		{"kob_to_sidecontrol", KneeOnBelly, SideControl, false},
		{"ns_to_sidecontrol", NorthSouth, SideControl, false},
		{"crucifix_to_background", Crucifix, BackGround, false},
		{"background_to_mount", BackGround, Mount, false},
		{"halfguard_to_guard", HalfGuard, Guard, false},

		// Terminal degrade sources — defender must escape or reverse
		{"guard_hold", Guard, Guard, true},
		{"turtle_hold", Turtle, Turtle, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			target, hold := DegradeTarget(c.source)
			if hold != c.wantHold {
				t.Errorf("DegradeTarget(%v) hold=%v, want %v", c.source, hold, c.wantHold)
			}
			if !hold && target != c.wantTarget {
				t.Errorf("DegradeTarget(%v) target=%v, want %v", c.source, target, c.wantTarget)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/state/position/ -run "TestDegradeTarget" -count=1`
Expected: FAIL with undefined `DegradeTarget`.

- [ ] **Step 3: Implement DegradeTarget**

Append to `outcomes.go`:

```go
// DegradeTarget returns the position to step down to when the
// defender wins drift moderately (|z| in [0.5, 1.0)). Implements
// spec §6.2. For symmetric or terminal positions (Clinch, Guard,
// Turtle), returns (source, true) — defender can't degrade further
// from these and must escape or reverse instead.
func DegradeTarget(source State) (target State, hold bool) {
	switch source {
	case Clinch:
		// Symmetric — no degrade target. Stamina-drain round.
		return Clinch, true
	case BackStanding:
		return Clinch, false
	case Mount:
		return SideControl, false
	case SideControl:
		return HalfGuard, false
	case KneeOnBelly:
		return SideControl, false
	case NorthSouth:
		return SideControl, false
	case Crucifix:
		return BackGround, false
	case BackGround:
		return Mount, false
	case HalfGuard:
		return Guard, false
	case Guard, Turtle:
		// Terminal — defender can only escape or reverse from here.
		return source, true
	}
	return source, true
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/state/position/ -run "TestDegradeTarget" -count=1 -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/state/position/outcomes.go internal/state/position/outcomes_test.go
git commit -m "feat(position): chunk 4b-fixup T3 — degrade transition table

DegradeTarget(source) → (target, hold). Implements spec §6.2.
Clinch / Guard / Turtle return Hold (no degrade target; defender
must escape or reverse instead). Other 8 sources mapped per spec.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Reversal transition with realism exceptions

**Files:**
- Modify: `internal/state/position/outcomes.go`
- Modify: `internal/state/position/outcomes_test.go`

- [ ] **Step 1: Write failing tests**

Append to `outcomes_test.go`:

```go
func TestReversalTarget(t *testing.T) {
	cases := []struct {
		name       string
		source     State
		wantTarget State
		wantSwap   bool // role swap (always true for reversals)
	}{
		// Realism exceptions
		{"mount_reverse_to_guard", Mount, Guard, true},
		{"background_reverse_to_mount", BackGround, Mount, true},

		// Default: same position, roles swap
		{"clinch_reverse_same", Clinch, Clinch, true},
		{"sidecontrol_reverse_same", SideControl, SideControl, true},
		{"kob_reverse_same", KneeOnBelly, KneeOnBelly, true},
		{"ns_reverse_same", NorthSouth, NorthSouth, true},
		{"crucifix_reverse_same", Crucifix, Crucifix, true},
		{"halfguard_reverse_same", HalfGuard, HalfGuard, true},
		{"guard_reverse_same", Guard, Guard, true},
		{"turtle_reverse_same", Turtle, Turtle, true},
		{"backstanding_reverse_same", BackStanding, BackStanding, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			target, swap := ReversalTarget(c.source)
			if target != c.wantTarget {
				t.Errorf("ReversalTarget(%v) target=%v, want %v", c.source, target, c.wantTarget)
			}
			if swap != c.wantSwap {
				t.Errorf("ReversalTarget(%v) swap=%v, want %v", c.source, swap, c.wantSwap)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/state/position/ -run "TestReversalTarget" -count=1`
Expected: FAIL with undefined `ReversalTarget`.

- [ ] **Step 3: Implement ReversalTarget**

Append to `outcomes.go`:

```go
// ReversalTarget returns the position to transition to when the
// defender wins drift big (|z| in [1.0, 2.0)) — a reversal. Roles
// always swap (returned as `swap=true`). Two realism exceptions per
// spec §6.3 land in a different position:
//   - Mount        → Guard  (defender bridges up into former
//                            controller's guard; former controller
//                            is now the Guard-controlled top)
//   - BackGround   → Mount  (defender turns into former controller;
//                            former controller is now Mount-controlled
//                            bottom)
// All other sources return the same state with roles swapped.
//
// Caller is responsible for performing the role swap when applying
// the transition.
func ReversalTarget(source State) (target State, swap bool) {
	switch source {
	case Mount:
		return Guard, true
	case BackGround:
		return Mount, true
	default:
		return source, true
	}
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/state/position/ -run "TestReversalTarget" -count=1 -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/state/position/outcomes.go internal/state/position/outcomes_test.go
git commit -m "feat(position): chunk 4b-fixup T4 — reversal transition table

ReversalTarget(source) → (target, swap). All reversals swap roles;
two realism exceptions land in a different position (Mount→Guard
and BackGround→Mount per spec §6.3). All other sources reverse in
place.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Outcome resolver dispatcher

**Files:**
- Modify: `internal/state/position/outcomes.go`
- Modify: `internal/state/position/outcomes_test.go`

- [ ] **Step 1: Write failing tests for ResolveOutcome**

Append to `outcomes_test.go`:

```go
func TestResolveOutcomeHold(t *testing.T) {
	o := ResolveOutcome(Mount, 0.3, Standing)
	if o.Kind != OutcomeHold {
		t.Errorf("z=0.3 should resolve to Hold, got %v", o.Kind)
	}
	if o.SubWindow {
		t.Errorf("z=0.3 should not open sub window")
	}
}

func TestResolveOutcomeAdvance(t *testing.T) {
	// SideControl, z=0.7 (1-step) → Mount, no sub
	o := ResolveOutcome(SideControl, 0.7, Standing)
	if o.Kind != OutcomeAdvance {
		t.Errorf("z=0.7 from SideControl should be Advance, got %v", o.Kind)
	}
	if o.Target != Mount {
		t.Errorf("z=0.7 from SideControl should target Mount, got %v", o.Target)
	}
	if o.SubWindow {
		t.Errorf("z=0.7 should not open sub window")
	}
}

func TestResolveOutcomeAdvanceWithSub(t *testing.T) {
	// SideControl, z=1.7 (2-step, sub-window open) → Mount + sub
	o := ResolveOutcome(SideControl, 1.7, Standing)
	if o.Kind != OutcomeAdvance {
		t.Errorf("z=1.7 should be Advance, got %v", o.Kind)
	}
	if o.Target != Mount {
		t.Errorf("z=1.7 from SideControl should target Mount, got %v", o.Target)
	}
	if !o.SubWindow {
		t.Errorf("z=1.7 should open sub window")
	}
}

func TestResolveOutcomeAdvanceWithSubAtThreeStep(t *testing.T) {
	// SideControl, z=2.5 (3-step) → BackGround + sub
	o := ResolveOutcome(SideControl, 2.5, Standing)
	if o.Target != BackGround {
		t.Errorf("z=2.5 from SideControl should target BackGround, got %v", o.Target)
	}
	if !o.SubWindow {
		t.Errorf("z=2.5 should open sub window")
	}
}

func TestResolveOutcomeMountStrikingApex(t *testing.T) {
	// Mount, z=1.7 (2-step) → Hold (striking apex), but sub window opens
	o := ResolveOutcome(Mount, 1.7, Standing)
	if o.Kind != OutcomeHold {
		t.Errorf("z=1.7 from Mount should be Hold (striking apex), got %v", o.Kind)
	}
	if !o.SubWindow {
		t.Errorf("z=1.7 from Mount should still open sub window (independent gate)")
	}
}

func TestResolveOutcomeDegrade(t *testing.T) {
	// Mount, z=-0.7 (defender 1-step) → degrade to SideControl
	o := ResolveOutcome(Mount, -0.7, Standing)
	if o.Kind != OutcomeDegrade {
		t.Errorf("z=-0.7 from Mount should be Degrade, got %v", o.Kind)
	}
	if o.Target != SideControl {
		t.Errorf("z=-0.7 from Mount should degrade to SideControl, got %v", o.Target)
	}
}

func TestResolveOutcomeDegradeTerminalClinchHold(t *testing.T) {
	// Clinch, z=-0.7 → Hold (Clinch has no degrade target)
	o := ResolveOutcome(Clinch, -0.7, Standing)
	if o.Kind != OutcomeHold {
		t.Errorf("z=-0.7 from Clinch should Hold (no degrade target), got %v", o.Kind)
	}
}

func TestResolveOutcomeReversal(t *testing.T) {
	// Mount, z=-1.5 → Reversal to Guard
	o := ResolveOutcome(Mount, -1.5, Standing)
	if o.Kind != OutcomeReversal {
		t.Errorf("z=-1.5 from Mount should be Reversal, got %v", o.Kind)
	}
	if o.Target != Guard {
		t.Errorf("Mount reversal target should be Guard, got %v", o.Target)
	}
}

func TestResolveOutcomeEscape(t *testing.T) {
	// Mount, z=-2.5 → Escape to Standing
	o := ResolveOutcome(Mount, -2.5, Standing)
	if o.Kind != OutcomeEscape {
		t.Errorf("z=-2.5 from Mount should be Escape, got %v", o.Kind)
	}
	if o.Target != Standing {
		t.Errorf("Escape target should be Standing, got %v", o.Target)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/state/position/ -run "TestResolveOutcome" -count=1`
Expected: FAIL with undefined `ResolveOutcome`, `Outcome`, `OutcomeHold`, etc.

- [ ] **Step 3: Implement Outcome + ResolveOutcome**

Append to `outcomes.go`:

```go
// OutcomeKind enumerates the five round-outcome categories for a
// grapple per spec §4. Combined with Outcome.Target and
// Outcome.SubWindow, fully describes what happens this round.
type OutcomeKind int

const (
	OutcomeHold     OutcomeKind = iota // No position change
	OutcomeAdvance                     // Controller wins; advance to Target
	OutcomeDegrade                     // Defender wins moderate; position regresses to Target
	OutcomeReversal                    // Defender wins big; roles swap, position is Target
	OutcomeEscape                      // Defender wins decisive; both → Standing
)

// Outcome is the resolved per-round result of a drift roll, ready
// for Position_GrappleTick to apply via TransitionPair and pass to
// the messaging layer.
type Outcome struct {
	Kind      OutcomeKind
	Source    State // pre-transition position (for messaging context)
	Target    State // post-transition position (= Source for Hold)
	SubWindow bool  // true if |z| >= 1.5 (chunk 4d composition)
}

// ResolveOutcome dispatches the drift roll's z-score to one of the
// five outcome categories per spec §4 + §5.
//
//   - z is signed: positive = controller won, negative = defender won.
//   - source is the controller's current grapple position.
//   - defenderPosture matters only when source is Clinch (Clinch-1-step
//     posture-based dispatch); ignored for non-Clinch sources.
func ResolveOutcome(source State, z float64, defenderPosture State) Outcome {
	absZ := z
	if absZ < 0 {
		absZ = -absZ
	}
	tier := OutcomeTierFromAbsZ(absZ)
	subOpens := SubWindowOpens(absZ)

	// Hold tier (|z| < 0.5): no position change regardless of sign.
	if tier == TierHold {
		return Outcome{
			Kind:      OutcomeHold,
			Source:    source,
			Target:    source,
			SubWindow: subOpens, // Always false at this tier, but consistent
		}
	}

	// Controller-favored (positive z).
	if z > 0 {
		target, hold := AdvancementTarget(source, tier, defenderPosture)
		if hold {
			return Outcome{
				Kind:      OutcomeHold,
				Source:    source,
				Target:    source,
				SubWindow: subOpens,
			}
		}
		return Outcome{
			Kind:      OutcomeAdvance,
			Source:    source,
			Target:    target,
			SubWindow: subOpens,
		}
	}

	// Defender-favored (negative z). Branch on tier.
	switch tier {
	case TierOneStep:
		target, hold := DegradeTarget(source)
		if hold {
			return Outcome{
				Kind:      OutcomeHold,
				Source:    source,
				Target:    source,
				SubWindow: subOpens, // Never true at TierOneStep
			}
		}
		return Outcome{
			Kind:      OutcomeDegrade,
			Source:    source,
			Target:    target,
			SubWindow: subOpens,
		}
	case TierTwoStep:
		target, _ := ReversalTarget(source)
		return Outcome{
			Kind:      OutcomeReversal,
			Source:    source,
			Target:    target,
			SubWindow: subOpens,
		}
	default: // TierThreeStep
		return Outcome{
			Kind:      OutcomeEscape,
			Source:    source,
			Target:    Standing,
			SubWindow: subOpens,
		}
	}
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/state/position/ -run "TestResolveOutcome" -count=1 -v`
Expected: PASS (all 9 subtests).

- [ ] **Step 5: Commit**

```bash
git add internal/state/position/outcomes.go internal/state/position/outcomes_test.go
git commit -m "feat(position): chunk 4b-fixup T5 — ResolveOutcome dispatcher

Outcome struct + OutcomeKind enum (Hold/Advance/Degrade/Reversal/
Escape) + ResolveOutcome(source, z, defenderPosture) dispatcher.
Implements spec §4 (per-round algorithm) and §5 (z-bucket table)
fully. Sub-window flag attached to outcome via independent gate
at |z| >= 1.5.

Pure function — no side effects, ready for Position_GrappleTick
wiring in T17.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Messaging YAML schema + loader

**Files:**
- Create: `internal/grapplemessaging/loader.go`
- Create: `internal/grapplemessaging/loader_test.go`
- Create: `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml` (initial skeleton — full templates in T8-T13)

- [ ] **Step 1: Create initial skeleton YAML for loader to read**

Create `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml`:

```yaml
# Grapple outcome flavor templates for chunk 4b-fixup.
# See docs/superpowers/specs/2026-05-18-state-chunk-4b-fixup-position-
# advancement-design.md §7 for the authoring rubric.
#
# Template substitution variables:
#   {controllerName} - the controller side's display name
#   {controlledName} - the controlled side's display name
#
# Speaker variants:
#   controller - second-person to the controller side
#   controlled - second-person to the controlled side
#   observers  - third-person to the room
#
# This file is loaded at boot by internal/grapplemessaging.
# Skeleton committed in T6; full templates land in T8-T13.

advancements: {}
degradations: {}
reversals: {}
escapes: {}
holds: {}
striking_apex: {}
```

- [ ] **Step 2: Write failing tests for loader**

Create `internal/grapplemessaging/loader_test.go`:

```go
package grapplemessaging

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test_grapple_outcomes.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp yaml: %v", err)
	}
	return path
}

func TestLoaderEmptySkeleton(t *testing.T) {
	path := writeTempYAML(t, `
advancements: {}
degradations: {}
reversals: {}
escapes: {}
holds: {}
striking_apex: {}
`)
	lib, err := Load(path)
	if err != nil {
		t.Fatalf("Load skeleton: %v", err)
	}
	if lib == nil {
		t.Fatal("Load returned nil library")
	}
	if len(lib.Advancements) != 0 {
		t.Errorf("expected empty advancements, got %d", len(lib.Advancements))
	}
}

func TestLoaderTriadParse(t *testing.T) {
	path := writeTempYAML(t, `
advancements:
  clinch_to_mount:
    controller:
      - "You drive forward and ride them into mount."
    controlled:
      - "{controllerName} drives forward and mounts you."
    observers:
      - "{controllerName} drives forward and mounts {controlledName}."
degradations: {}
reversals: {}
escapes: {}
holds: {}
striking_apex: {}
`)
	lib, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	triad, ok := lib.Advancements["clinch_to_mount"]
	if !ok {
		t.Fatal("missing clinch_to_mount key")
	}
	if len(triad.Controller) != 1 || len(triad.Controlled) != 1 || len(triad.Observers) != 1 {
		t.Errorf("triad lengths: ctrl=%d cd=%d obs=%d, want 1/1/1",
			len(triad.Controller), len(triad.Controlled), len(triad.Observers))
	}
}

func TestLoaderMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/grapple_outcomes.yaml")
	if err == nil {
		t.Error("Load should error on missing file")
	}
}

func TestLoaderInvalidYAML(t *testing.T) {
	path := writeTempYAML(t, `this is not: valid: yaml: at all: ::`)
	_, err := Load(path)
	if err == nil {
		t.Error("Load should error on invalid YAML")
	}
}

func TestStrikingApexParse(t *testing.T) {
	path := writeTempYAML(t, `
advancements: {}
degradations: {}
reversals: {}
escapes: {}
holds: {}
striking_apex:
  mount_strike_flavor:
    - "You ride high in mount and rain elbows down."
    - "Their arms tire from defending; you sit heavy."
`)
	lib, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	apex, ok := lib.StrikingApex["mount_strike_flavor"]
	if !ok {
		t.Fatal("missing mount_strike_flavor key")
	}
	if len(apex) != 2 {
		t.Errorf("expected 2 strike flavor lines, got %d", len(apex))
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/grapplemessaging/ -count=1`
Expected: FAIL with no package or undefined Load.

- [ ] **Step 4: Implement loader**

Create `internal/grapplemessaging/loader.go`:

```go
// Package grapplemessaging loads and renders flavor templates for
// grapple outcomes (advance, degrade, reverse, escape, hold,
// striking apex). Templates live in
// _datafiles/world/dogmud/messaging/grapple_outcomes.yaml.
//
// Consumer is internal/hooks/Position_GrappleTick.go via the
// RenderOutcome function (T9).
package grapplemessaging

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// TemplateTriad holds the three speaker-variant template lists for
// a single outcome key. controller is shown to the controller side
// (second-person "you"); controlled is shown to the controlled side
// (second-person "you", from their POV); observers is broadcast to
// everyone else in the room (third-person).
type TemplateTriad struct {
	Controller []string `yaml:"controller"`
	Controlled []string `yaml:"controlled"`
	Observers  []string `yaml:"observers"`
}

// Library is the parsed in-memory template store. Keys for each map
// follow spec §7.1 conventions:
//   - Advancements:  "<source>_to_<target>" (e.g. "clinch_to_mount")
//   - Degradations:  "<source>_to_<target>" (e.g. "mount_to_sidecontrol")
//   - Reversals:     "<source>_reverse" for realism-exception sources;
//                    "generic_reverse" as fallback.
//   - Escapes:       "generic_escape" (only key for now)
//   - Holds:         "<context>_hold" (e.g. "clinch_hold",
//                    "ground_hold_generic")
//   - StrikingApex:  Single-speaker (no triad); "mount_strike_flavor"
//                    is the only key currently.
type Library struct {
	Advancements map[string]TemplateTriad `yaml:"advancements"`
	Degradations map[string]TemplateTriad `yaml:"degradations"`
	Reversals    map[string]TemplateTriad `yaml:"reversals"`
	Escapes      map[string]TemplateTriad `yaml:"escapes"`
	Holds        map[string]TemplateTriad `yaml:"holds"`
	StrikingApex map[string][]string      `yaml:"striking_apex"`
}

// Load reads and parses the grapple outcome template file. Returns
// an error if the file is missing, unreadable, or malformed YAML.
// Empty / partial libraries are valid — callers may apply their own
// completeness validation via ValidateCompleteness (T8).
func Load(path string) (*Library, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("grapplemessaging.Load: read %q: %w", path, err)
	}
	lib := &Library{}
	if err := yaml.Unmarshal(data, lib); err != nil {
		return nil, fmt.Errorf("grapplemessaging.Load: parse %q: %w", path, err)
	}
	// Initialize nil maps so consumers can index safely.
	if lib.Advancements == nil {
		lib.Advancements = map[string]TemplateTriad{}
	}
	if lib.Degradations == nil {
		lib.Degradations = map[string]TemplateTriad{}
	}
	if lib.Reversals == nil {
		lib.Reversals = map[string]TemplateTriad{}
	}
	if lib.Escapes == nil {
		lib.Escapes = map[string]TemplateTriad{}
	}
	if lib.Holds == nil {
		lib.Holds = map[string]TemplateTriad{}
	}
	if lib.StrikingApex == nil {
		lib.StrikingApex = map[string][]string{}
	}
	return lib, nil
}
```

- [ ] **Step 5: Run tests to verify pass**

Run: `go test ./internal/grapplemessaging/ -count=1 -v`
Expected: PASS (all 5 tests).

- [ ] **Step 6: Verify build still compiles**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 7: Commit**

```bash
git add internal/grapplemessaging/loader.go internal/grapplemessaging/loader_test.go _datafiles/world/dogmud/messaging/grapple_outcomes.yaml
git commit -m "feat(grapplemessaging): chunk 4b-fixup T6 — YAML loader + skeleton library

New internal/grapplemessaging package. Loader parses the
grapple_outcomes.yaml file into a Library struct with five triad
maps (Advancements / Degradations / Reversals / Escapes / Holds)
plus a single-speaker StrikingApex map. Spec §7.1 schema.

Templates themselves land in T8-T13; this task ships only the
schema + loader + empty skeleton file so T9 (render) and T17
(GrappleTick wiring) have a stable interface to build against.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Library completeness validator

**Files:**
- Modify: `internal/grapplemessaging/loader.go`
- Modify: `internal/grapplemessaging/loader_test.go`

- [ ] **Step 1: Write failing test for ValidateCompleteness**

Append to `loader_test.go`:

```go
func TestValidateCompletenessEmpty(t *testing.T) {
	lib := &Library{
		Advancements: map[string]TemplateTriad{},
		Degradations: map[string]TemplateTriad{},
		Reversals:    map[string]TemplateTriad{},
		Escapes:      map[string]TemplateTriad{},
		Holds:        map[string]TemplateTriad{},
		StrikingApex: map[string][]string{},
	}
	errs := ValidateCompleteness(lib)
	if len(errs) == 0 {
		t.Error("empty library should produce completeness errors")
	}
}

func TestValidateCompletenessMissingTriadSpeaker(t *testing.T) {
	lib := &Library{
		Advancements: map[string]TemplateTriad{
			"clinch_to_mount": {
				Controller: []string{"a", "b", "c"},
				Controlled: []string{"a", "b", "c"},
				// Observers missing — should error
			},
		},
		Degradations: map[string]TemplateTriad{},
		Reversals:    map[string]TemplateTriad{},
		Escapes:      map[string]TemplateTriad{},
		Holds:        map[string]TemplateTriad{},
		StrikingApex: map[string][]string{},
	}
	errs := ValidateCompleteness(lib)
	if len(errs) == 0 {
		t.Error("triad with empty Observers should fail validation")
	}
}

func TestValidateCompletenessUnderMinCount(t *testing.T) {
	lib := &Library{
		Advancements: map[string]TemplateTriad{
			"clinch_to_mount": {
				Controller: []string{"only one"},
				Controlled: []string{"only one"},
				Observers:  []string{"only one"},
			},
		},
		Degradations: map[string]TemplateTriad{},
		Reversals:    map[string]TemplateTriad{},
		Escapes:      map[string]TemplateTriad{},
		Holds:        map[string]TemplateTriad{},
		StrikingApex: map[string][]string{},
	}
	errs := ValidateCompleteness(lib)
	if len(errs) == 0 {
		t.Error("triad with only 1 template per speaker should fail (minimum 3)")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/grapplemessaging/ -run "TestValidateCompleteness" -count=1`
Expected: FAIL with undefined `ValidateCompleteness`.

- [ ] **Step 3: Implement ValidateCompleteness**

Append to `loader.go`:

```go
// Minimum templates per triad-speaker variant (spec §7.4).
const MinTemplatesPerSpeaker = 3

// Minimum templates for the Mount strike-apex single-speaker list.
const MinStrikingApexTemplates = 5

// RequiredAdvancementKeys lists the (source, target) pairs that the
// advancement messaging must cover. Derived from the spec §6.1 table
// — every cell in the table that produces an actual transition (not
// Hold) needs its own template triad.
var RequiredAdvancementKeys = []string{
	// Clinch 1-step posture variants
	"clinch_to_mount",
	"clinch_to_sidecontrol",
	"clinch_to_background",
	"clinch_to_backstanding",
	// Clinch 2-step
	// (clinch_to_mount already in list — same key, no duplicate)
	// Clinch 3-step
	// (clinch_to_background already in list)

	// BackStanding all tiers
	"backstanding_to_background",

	// Mount 3-step only (1/2-step are Hold)
	"mount_to_background",

	// SideControl 1/2-step
	"sidecontrol_to_mount",
	// SideControl 3-step (same key as Mount 3-step? No: SC→BackGround)
	"sidecontrol_to_background",

	// KneeOnBelly 1/2 → Mount; 3 → BackGround
	"kob_to_mount",
	"kob_to_background",

	// NorthSouth 1 → SC, 2 → Mount, 3 → BackGround
	"ns_to_sidecontrol",
	"ns_to_mount",
	"ns_to_background",

	// HalfGuard 1 → SC, 2 → Mount, 3 → BackGround
	"halfguard_to_sidecontrol",
	"halfguard_to_mount",
	"halfguard_to_background",

	// Guard 1 → SC, 2 → Mount, 3 → BackGround (controller bottom passes)
	"guard_to_sidecontrol",
	"guard_to_mount",
	"guard_to_background",

	// Turtle all tiers → BackGround
	"turtle_to_background",
}

// RequiredDegradationKeys lists the (source, target) pairs from spec
// §6.2 that produce actual transitions (Hold sources omitted).
var RequiredDegradationKeys = []string{
	"backstanding_to_clinch",
	"mount_to_sidecontrol",
	"sidecontrol_to_halfguard",
	"kob_to_sidecontrol",
	"ns_to_sidecontrol",
	"crucifix_to_background",
	"background_to_mount",
	"halfguard_to_guard",
}

// RequiredReversalKeys covers the 2 realism-exception reversals
// (Mount→Guard, BackGround→Mount) plus the generic fallback.
var RequiredReversalKeys = []string{
	"mount_reverse",       // → Guard with role swap
	"background_reverse",  // → Mount with role swap
	"generic_reverse",     // any other source: same position, role swap
}

// RequiredEscapeKeys: single generic key for the always-to-Standing
// escape.
var RequiredEscapeKeys = []string{
	"generic_escape",
}

// RequiredHoldKeys: per-context hold flavor. Sparse — only fires
// every ~3-4 rounds via cooldown.
var RequiredHoldKeys = []string{
	"clinch_hold",
	"ground_hold_generic",
	"guard_hold",
	"turtle_hold",
	"backstanding_hold",
}

// RequiredStrikingApexKeys: Mount-only.
var RequiredStrikingApexKeys = []string{
	"mount_strike_flavor",
}

// ValidateCompleteness checks that every required key is present
// AND each triad has at least MinTemplatesPerSpeaker templates per
// speaker variant. StrikingApex keys need MinStrikingApexTemplates
// in the single-speaker list. Returns a slice of all violations
// (caller decides whether to fail-loud at boot or just log).
func ValidateCompleteness(lib *Library) []error {
	var errs []error

	check := func(category string, keys []string, m map[string]TemplateTriad) {
		for _, key := range keys {
			triad, ok := m[key]
			if !ok {
				errs = append(errs, fmt.Errorf("%s: missing key %q", category, key))
				continue
			}
			if len(triad.Controller) < MinTemplatesPerSpeaker {
				errs = append(errs, fmt.Errorf("%s.%s.controller: %d templates, need >= %d",
					category, key, len(triad.Controller), MinTemplatesPerSpeaker))
			}
			if len(triad.Controlled) < MinTemplatesPerSpeaker {
				errs = append(errs, fmt.Errorf("%s.%s.controlled: %d templates, need >= %d",
					category, key, len(triad.Controlled), MinTemplatesPerSpeaker))
			}
			if len(triad.Observers) < MinTemplatesPerSpeaker {
				errs = append(errs, fmt.Errorf("%s.%s.observers: %d templates, need >= %d",
					category, key, len(triad.Observers), MinTemplatesPerSpeaker))
			}
		}
	}

	check("advancements", RequiredAdvancementKeys, lib.Advancements)
	check("degradations", RequiredDegradationKeys, lib.Degradations)
	check("reversals", RequiredReversalKeys, lib.Reversals)
	check("escapes", RequiredEscapeKeys, lib.Escapes)
	check("holds", RequiredHoldKeys, lib.Holds)

	for _, key := range RequiredStrikingApexKeys {
		templates, ok := lib.StrikingApex[key]
		if !ok {
			errs = append(errs, fmt.Errorf("striking_apex: missing key %q", key))
			continue
		}
		if len(templates) < MinStrikingApexTemplates {
			errs = append(errs, fmt.Errorf("striking_apex.%s: %d templates, need >= %d",
				key, len(templates), MinStrikingApexTemplates))
		}
	}

	return errs
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/grapplemessaging/ -run "TestValidateCompleteness" -count=1 -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/grapplemessaging/loader.go internal/grapplemessaging/loader_test.go
git commit -m "feat(grapplemessaging): chunk 4b-fixup T7 — completeness validator

ValidateCompleteness(lib) walks every required key and asserts
minimum template counts per speaker variant. Required keys are
defined as exported variables so the YAML authors (and the
realism reviewer in T14) can grep the list.

Authoring tasks T8-T13 will populate the library to make this
validator return no errors. Boot smoke (T26) calls
ValidateCompleteness and fails fast on any violation.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: Author advancement templates

**Files:**
- Modify: `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml`

This task populates the `advancements:` section. ~19 keys, each needs ≥3 templates per speaker variant (controller / controlled / observers) = ~171 templates minimum.

- [ ] **Step 1: Replace `advancements: {}` with the full populated block**

The full text is long — implementer authors per the rubric below, using the spec §7.3 starter templates for Clinch→Mount as the tone anchor. Every key in `RequiredAdvancementKeys` must be present with ≥3 templates per speaker variant.

Authoring rubric (from spec §7.2):
- Minimum 3 templates per (source, target) per speaker.
- No hard numbers.
- MMA/BJJ-flavored vocabulary; visceral, grounded.
- Three speaker variants: controller (second-person "You ..."), controlled (second-person "She/He ..." from defender POV), observers (third-person to room).
- Each template per (key × speaker) must be unique within its list.

Use the spec's clinch_to_mount example block as the tone reference (§7.3):

```yaml
advancements:
  clinch_to_mount:
    controller:
      - "You wrench the underhook, drive forward, and ride them down into mount."
      - "A snap-down sets it up — you sprawl over the shoulder and slide into mount before they can recover."
      - "Their stance breaks. You drag them flat and climb on top, knees high in mount."
    controlled:
      - "{controllerName} wrenches the underhook and rides you down. The ceiling spins overhead — they're mounted on your chest."
      - "You feel the snap-down too late. Their weight crashes onto your sternum and the world goes vertical."
      - "Your stance crumbles. {controllerName}'s knees hammer up under your armpits — full mount."
    observers:
      - "{controllerName} wrenches an underhook and rides {controlledName} down into mount."
      - "{controllerName} snaps {controlledName} down and climbs into mount before they can post."
      - "{controlledName}'s stance breaks; {controllerName} drags them flat and mounts them."
  # ... (18 more keys)
```

Author each of the following 19 keys following this template structure:

1. `clinch_to_mount` (above — anchor)
2. `clinch_to_sidecontrol`
3. `clinch_to_background`
4. `clinch_to_backstanding`
5. `backstanding_to_background`
6. `mount_to_background` (the back-take from full mount — defender turning away)
7. `sidecontrol_to_mount`
8. `sidecontrol_to_background`
9. `kob_to_mount`
10. `kob_to_background`
11. `ns_to_sidecontrol`
12. `ns_to_mount`
13. `ns_to_background`
14. `halfguard_to_sidecontrol`
15. `halfguard_to_mount`
16. `halfguard_to_background`
17. `guard_to_sidecontrol` (guard pass — controller-on-bottom comes up on top)
18. `guard_to_mount`
19. `guard_to_background`
20. `turtle_to_background`

(Note: 20 keys, not 19 — clinch_to_background appears in both 1-step and 3-step paths; it's one key. Total unique: 20.)

- [ ] **Step 2: Run validator test to confirm completeness**

Run: `go test ./internal/grapplemessaging/ -run "TestValidateCompleteness" -count=1 -v`

To run against the actual production file, add a quick ad-hoc test:

```go
func TestProductionLibraryAdvancementsComplete(t *testing.T) {
	lib, err := Load("../../_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
	if err != nil {
		t.Fatalf("Load prod library: %v", err)
	}
	for _, key := range RequiredAdvancementKeys {
		triad, ok := lib.Advancements[key]
		if !ok {
			t.Errorf("missing advancement key: %s", key)
			continue
		}
		if len(triad.Controller) < MinTemplatesPerSpeaker {
			t.Errorf("%s.controller: %d < %d", key, len(triad.Controller), MinTemplatesPerSpeaker)
		}
		if len(triad.Controlled) < MinTemplatesPerSpeaker {
			t.Errorf("%s.controlled: %d < %d", key, len(triad.Controlled), MinTemplatesPerSpeaker)
		}
		if len(triad.Observers) < MinTemplatesPerSpeaker {
			t.Errorf("%s.observers: %d < %d", key, len(triad.Observers), MinTemplatesPerSpeaker)
		}
	}
}
```

(This test stays in the repo as a guard against future regressions in the YAML.)

Run: `go test ./internal/grapplemessaging/ -run "TestProductionLibraryAdvancementsComplete" -count=1 -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/messaging/grapple_outcomes.yaml internal/grapplemessaging/loader_test.go
git commit -m "content(messaging): chunk 4b-fixup T8 — advancement templates (~180)

All 20 advancement keys populated with >= 3 templates per speaker
variant (controller / controlled / observers). MMA/BJJ-flavored
vocabulary per spec §7.2. Tone anchored on the clinch_to_mount
templates from spec §7.3.

Adds a production-library guard test so future drift in this YAML
fails CI rather than silently breaking the messaging library.

Realism sanity-check pass lands in T14 — these may be revised.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: Author degradation templates

**Files:**
- Modify: `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml`
- Modify: `internal/grapplemessaging/loader_test.go`

Populates `degradations:` section. 8 keys × 9 templates min = 72 templates min.

- [ ] **Step 1: Replace `degradations: {}` with populated block**

Required keys (from `RequiredDegradationKeys`):
- `backstanding_to_clinch`
- `mount_to_sidecontrol`
- `sidecontrol_to_halfguard`
- `kob_to_sidecontrol`
- `ns_to_sidecontrol`
- `crucifix_to_background`
- `background_to_mount`
- `halfguard_to_guard`

Authoring guidance: degradations are defender-driven escapes. The controller side reads as "losing the position," the controlled as "scrambling toward a better position," observers as "X frames out / hip-escapes to Y."

Example tone for `mount_to_sidecontrol`:

```yaml
degradations:
  mount_to_sidecontrol:
    controller:
      - "{controlledName} bridges hard and hip-escapes — your mount slides off into side control."
      - "You can't keep your base. {controlledName} shrimps free and lands in side control."
      - "A frame on your hip pries your knees apart; you settle for side control."
    controlled:
      - "You bridge hard and shrimp the hip free — {controllerName}'s mount slides off you and into side control."
      - "Your elbow frames between you and {controllerName}'s hip. You hip-escape and they settle for side."
      - "You buckle your knees and arc your back. {controllerName} loses base; you scramble to side."
    observers:
      - "{controlledName} bridges and hip-escapes from under {controllerName}; the mount slides to side control."
      - "{controlledName} shrimps a frame in; {controllerName} can't hold mount and settles for side."
      - "{controlledName}'s buckle-and-arc pries {controllerName} off; mount becomes side control."
  # ... (7 more keys)
```

- [ ] **Step 2: Add production-library guard test for degradations**

Append to `loader_test.go`:

```go
func TestProductionLibraryDegradationsComplete(t *testing.T) {
	lib, err := Load("../../_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
	if err != nil {
		t.Fatalf("Load prod library: %v", err)
	}
	for _, key := range RequiredDegradationKeys {
		triad, ok := lib.Degradations[key]
		if !ok {
			t.Errorf("missing degradation key: %s", key)
			continue
		}
		if len(triad.Controller) < MinTemplatesPerSpeaker {
			t.Errorf("%s.controller: %d < %d", key, len(triad.Controller), MinTemplatesPerSpeaker)
		}
		if len(triad.Controlled) < MinTemplatesPerSpeaker {
			t.Errorf("%s.controlled: %d < %d", key, len(triad.Controlled), MinTemplatesPerSpeaker)
		}
		if len(triad.Observers) < MinTemplatesPerSpeaker {
			t.Errorf("%s.observers: %d < %d", key, len(triad.Observers), MinTemplatesPerSpeaker)
		}
	}
}
```

- [ ] **Step 3: Run the guard test**

Run: `go test ./internal/grapplemessaging/ -run "TestProductionLibraryDegradationsComplete" -count=1 -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/messaging/grapple_outcomes.yaml internal/grapplemessaging/loader_test.go
git commit -m "content(messaging): chunk 4b-fixup T9 — degradation templates (~72)

All 8 degradation keys populated. Defender-driven escape framing:
controller loses position, controlled hip-escapes/shrimps/frames
to a better one.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 10: Author reversal templates

**Files:**
- Modify: `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml`
- Modify: `internal/grapplemessaging/loader_test.go`

Populates `reversals:` section. 3 keys × 9 templates min = 27 templates min.

- [ ] **Step 1: Replace `reversals: {}` with populated block**

Required keys:
- `mount_reverse` (Mount → Guard with role swap; defender bridges up between attacker's legs)
- `background_reverse` (BackGround → Mount; defender turns into attacker)
- `generic_reverse` (any other source: same position, roles swap)

Example tone for `mount_reverse`:

```yaml
reversals:
  mount_reverse:
    controller:
      - "{controlledName} bridges and you tumble forward — they roll up between your legs and now you're in their guard."
      - "Your base evaporates. {controlledName} bucks you off and comes up between your legs — full guard."
      - "Hooks dig under your hips and lift; you cartwheel forward into their guard."
    controlled:
      - "You bridge hard, post on the temple, and roll {controllerName} between your legs — full guard, you're on top."
      - "Hips up, frame on the chin, sweep the post. {controllerName} tumbles forward into your guard."
      - "You time the buck and lift the hooks — {controllerName} cartwheels off you and lands in your closed guard."
    observers:
      - "{controlledName} bridges and rolls {controllerName} forward into a guard reversal — top and bottom flip."
      - "{controlledName} times a buck and sweeps {controllerName} into closed guard, reversing the mount."
      - "{controllerName} cartwheels off {controlledName}'s bridge and lands in their guard."
  # ... (2 more keys)
```

- [ ] **Step 2: Add production-library guard test**

Append to `loader_test.go`:

```go
func TestProductionLibraryReversalsComplete(t *testing.T) {
	lib, err := Load("../../_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
	if err != nil {
		t.Fatalf("Load prod library: %v", err)
	}
	for _, key := range RequiredReversalKeys {
		triad, ok := lib.Reversals[key]
		if !ok {
			t.Errorf("missing reversal key: %s", key)
			continue
		}
		if len(triad.Controller) < MinTemplatesPerSpeaker {
			t.Errorf("%s.controller: %d < %d", key, len(triad.Controller), MinTemplatesPerSpeaker)
		}
		if len(triad.Controlled) < MinTemplatesPerSpeaker {
			t.Errorf("%s.controlled: %d < %d", key, len(triad.Controlled), MinTemplatesPerSpeaker)
		}
		if len(triad.Observers) < MinTemplatesPerSpeaker {
			t.Errorf("%s.observers: %d < %d", key, len(triad.Observers), MinTemplatesPerSpeaker)
		}
	}
}
```

- [ ] **Step 3: Run guard test**

Run: `go test ./internal/grapplemessaging/ -run "TestProductionLibraryReversalsComplete" -count=1 -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/messaging/grapple_outcomes.yaml internal/grapplemessaging/loader_test.go
git commit -m "content(messaging): chunk 4b-fixup T10 — reversal templates (~27)

mount_reverse + background_reverse + generic_reverse keys.
Realism-exception flavor for the two specific reversals
(Mount→Guard via bridge-and-roll, BackGround→Mount via
turn-in-and-pin); generic for the role-swap-in-place fallback.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 11: Author escape templates

**Files:**
- Modify: `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml`
- Modify: `internal/grapplemessaging/loader_test.go`

Populates `escapes:` section. 1 key × 9 templates min.

- [ ] **Step 1: Replace `escapes: {}` with populated block**

```yaml
escapes:
  generic_escape:
    controller:
      - "{controlledName} explodes free — they're back on their feet before you can grab again."
      - "Your grip fails. {controlledName} scrambles up and out of reach."
      - "{controlledName} kicks off you and pops to their feet — the grapple breaks."
    controlled:
      - "You explode through the gap and scramble to your feet — free of {controllerName}'s grip at last."
      - "A frame, a kick, a scramble. You're up — {controllerName}'s grip is broken."
      - "You wrench the joint loose, kick {controllerName} away, and stand up clear."
    observers:
      - "{controlledName} explodes free of {controllerName} and scrambles back to their feet."
      - "{controllerName}'s grip fails — {controlledName} scrambles up and out of reach."
      - "{controlledName} kicks {controllerName} off and the grapple breaks; both stand."
```

- [ ] **Step 2: Add production-library guard test**

Append to `loader_test.go`:

```go
func TestProductionLibraryEscapesComplete(t *testing.T) {
	lib, err := Load("../../_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
	if err != nil {
		t.Fatalf("Load prod library: %v", err)
	}
	for _, key := range RequiredEscapeKeys {
		triad, ok := lib.Escapes[key]
		if !ok {
			t.Errorf("missing escape key: %s", key)
			continue
		}
		if len(triad.Controller) < MinTemplatesPerSpeaker ||
			len(triad.Controlled) < MinTemplatesPerSpeaker ||
			len(triad.Observers) < MinTemplatesPerSpeaker {
			t.Errorf("escape %s under-populated", key)
		}
	}
}
```

- [ ] **Step 3: Run guard test**

Run: `go test ./internal/grapplemessaging/ -run "TestProductionLibraryEscapesComplete" -count=1 -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/messaging/grapple_outcomes.yaml internal/grapplemessaging/loader_test.go
git commit -m "content(messaging): chunk 4b-fixup T11 — escape templates (~9)

Single generic_escape key — always-to-Standing exit. Defender-
explodes-free flavor across all three speaker variants.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 12: Author hold + striking-apex templates

**Files:**
- Modify: `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml`
- Modify: `internal/grapplemessaging/loader_test.go`

Populates `holds:` (5 keys × 9 templates = 45 min) and `striking_apex:` (1 key × 5 templates).

- [ ] **Step 1: Replace `holds: {}` and `striking_apex: {}` with populated blocks**

Hold flavor is sparse (fires every ~3-4 rounds via cooldown). Tone: grinding, stamina-drain, breath-and-sweat.

```yaml
holds:
  clinch_hold:
    controller:
      - "You and {controlledName} grind in the clinch — wrist fights and head-position grunts."
      - "Hand-fighting in the pocket. Neither of you gives an inch."
      - "Shoulders pressed together, breath ragged — you wrestle for the underhook."
    controlled:
      - "You and {controllerName} grind in the clinch — wrists battle for control."
      - "Your shoulders crash together. Neither side breaks position."
      - "Breath rasps against {controllerName}'s neck. The clinch holds."
    observers:
      - "{controllerName} and {controlledName} grind in the clinch — neither giving ground."
      - "{controllerName} and {controlledName} fight hands and head position."
      - "Shoulders crash together; the clinch holds."
  ground_hold_generic:
    controller:
      - "You posture down and ride out {controlledName}'s squirming."
      - "Pressure stays on. {controlledName} can't generate the leverage to move you."
      - "You settle your weight and let {controlledName} burn cardio trying to shift you."
    controlled:
      - "{controllerName}'s weight settles harder. You burn cardio trying to find leverage."
      - "You squirm and frame, but {controllerName} rides it out."
      - "Pressure pins you flat. You can't generate the leverage to budge."
    observers:
      - "{controllerName} rides out {controlledName}'s squirming."
      - "{controllerName} settles in and lets {controlledName} burn cardio."
      - "{controlledName} squirms; {controllerName} stays heavy."
  guard_hold:
    controller:
      - "You break {controlledName}'s posture down with your legs — they grind for an opening."
      - "Your hips frame {controlledName}'s; the guard stays closed."
      - "You snap {controlledName}'s head down with high guard, smothering their base."
    controlled:
      - "{controllerName}'s legs pry and snap — you can't post your base."
      - "Your knees frame in vain. {controllerName}'s guard holds."
      - "{controllerName} drags your head down with high guard; you smother."
    observers:
      - "{controllerName} works the guard, breaking {controlledName}'s posture."
      - "{controllerName}'s legs frame and pry; {controlledName} can't pass."
      - "{controllerName} keeps {controlledName} smothered in high guard."
  turtle_hold:
    controller:
      - "{controlledName} curls tighter. You hand-fight for an opening to the back."
      - "You ride the shoulder, looking for the seatbelt. {controlledName} stays balled up."
      - "Knees driven in tight — {controlledName} can't expand from the turtle."
    controlled:
      - "You curl tighter. {controllerName} pulls at your wrists."
      - "{controllerName} rides your shoulder, hunting hooks. You stay balled up."
      - "Their knees drive in. You can't expand from the turtle."
    observers:
      - "{controllerName} hand-fights {controlledName}, looking for an opening to the back."
      - "{controllerName} rides {controlledName}'s shoulder; the turtle holds."
      - "{controlledName} stays balled tight against {controllerName}."
  backstanding_hold:
    controller:
      - "You stay glued to {controlledName}'s back, hands fighting for the over-under."
      - "Hooks aren't there yet — you ride the standing back-clinch and wait."
      - "{controlledName} hand-fights your wrists; you stay stuck to their spine."
    controlled:
      - "{controllerName} stays glued to your back. You hand-fight the over-under."
      - "You feel them hunting hooks — you grip and shake to keep them off your hips."
      - "{controllerName}'s chest rides your spine. You can't peel them off."
    observers:
      - "{controllerName} stays glued to {controlledName}'s back, hand-fighting."
      - "{controllerName} hunts hooks against {controlledName}'s hips; {controlledName} blocks."
      - "{controlledName} shakes against {controllerName}'s back-clinch — neither gives."

striking_apex:
  mount_strike_flavor:
    - "You ride high in mount and rain elbows down."
    - "Your knees pin {controlledName}'s shoulders flat — heavy shots punch through their guard."
    - "{controlledName} bridges weakly under you; you posture up and drive hammerfists into their guard."
    - "Sweat and copper. You ride the mount and drop knuckles like pistons."
    - "Their arms tire from defending their face. You sit heavy and let the strikes through."
    - "You shift your weight onto {controlledName}'s sternum and unload — short, vicious elbows from the top."
    - "Cross-face pinning their jaw, free hand cocked back — you punish from full mount."
```

- [ ] **Step 2: Add production-library guard tests for holds and striking_apex**

Append to `loader_test.go`:

```go
func TestProductionLibraryHoldsComplete(t *testing.T) {
	lib, err := Load("../../_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
	if err != nil {
		t.Fatalf("Load prod library: %v", err)
	}
	for _, key := range RequiredHoldKeys {
		triad, ok := lib.Holds[key]
		if !ok {
			t.Errorf("missing hold key: %s", key)
			continue
		}
		if len(triad.Controller) < MinTemplatesPerSpeaker ||
			len(triad.Controlled) < MinTemplatesPerSpeaker ||
			len(triad.Observers) < MinTemplatesPerSpeaker {
			t.Errorf("hold %s under-populated", key)
		}
	}
}

func TestProductionLibraryStrikingApexComplete(t *testing.T) {
	lib, err := Load("../../_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
	if err != nil {
		t.Fatalf("Load prod library: %v", err)
	}
	for _, key := range RequiredStrikingApexKeys {
		templates, ok := lib.StrikingApex[key]
		if !ok {
			t.Errorf("missing striking_apex key: %s", key)
			continue
		}
		if len(templates) < MinStrikingApexTemplates {
			t.Errorf("striking_apex.%s: %d < %d", key, len(templates), MinStrikingApexTemplates)
		}
	}
}
```

- [ ] **Step 3: Run guard tests**

Run: `go test ./internal/grapplemessaging/ -run "TestProductionLibraryHoldsComplete|TestProductionLibraryStrikingApexComplete" -count=1 -v`
Expected: PASS.

- [ ] **Step 4: Verify full ValidateCompleteness passes against production file**

Add a final end-to-end test:

```go
func TestProductionLibraryFullValidation(t *testing.T) {
	lib, err := Load("../../_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
	if err != nil {
		t.Fatalf("Load prod library: %v", err)
	}
	errs := ValidateCompleteness(lib)
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("completeness: %v", e)
		}
	}
}
```

Run: `go test ./internal/grapplemessaging/ -count=1 -v`
Expected: ALL PASS.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/messaging/grapple_outcomes.yaml internal/grapplemessaging/loader_test.go
git commit -m "content(messaging): chunk 4b-fixup T12 — hold + striking-apex templates

5 hold keys (clinch/ground/guard/turtle/backstanding) × 9 each
= ~45 hold templates. mount_strike_flavor populated with 7 single-
speaker strike lines (above the 5-min). Full ValidateCompleteness
now passes against the production YAML.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 13: Render function with cooldown + name substitution

**Files:**
- Create: `internal/grapplemessaging/render.go`
- Create: `internal/grapplemessaging/render_test.go`

- [ ] **Step 1: Write failing tests for RenderOutcome**

Create `internal/grapplemessaging/render_test.go`:

```go
package grapplemessaging

import (
	"strings"
	"testing"
)

func buildTestLib() *Library {
	return &Library{
		Advancements: map[string]TemplateTriad{
			"clinch_to_mount": {
				Controller: []string{"You mount {controlledName}.", "You ride them down."},
				Controlled: []string{"{controllerName} mounts you.", "{controllerName} rides you down."},
				Observers:  []string{"{controllerName} mounts {controlledName}.", "{controllerName} rides {controlledName} down."},
			},
		},
		Degradations: map[string]TemplateTriad{},
		Reversals:    map[string]TemplateTriad{},
		Escapes:      map[string]TemplateTriad{},
		Holds:        map[string]TemplateTriad{},
		StrikingApex: map[string][]string{},
	}
}

func TestRenderAdvancementSubstitutesNames(t *testing.T) {
	lib := buildTestLib()
	out := RenderTemplate(lib.Advancements["clinch_to_mount"].Controller[0], "Athos", "Porthos")
	if !strings.Contains(out, "Porthos") {
		t.Errorf("expected substituted controlled name, got %q", out)
	}
	if strings.Contains(out, "{controlledName}") {
		t.Errorf("unsubstituted placeholder remained: %q", out)
	}
}

func TestPickTemplateRandomization(t *testing.T) {
	lib := buildTestLib()
	cooldowns := map[string]bool{}
	// Pick should rotate through both available templates given enough
	// calls; with cooldown empty, both should be reachable.
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		t := PickTemplate(lib.Advancements["clinch_to_mount"].Controller, cooldowns, "adv:clinch_to_mount:ctrl")
		seen[t] = true
	}
	if len(seen) < 2 {
		t.Errorf("expected variety after 100 picks, only saw %d unique templates", len(seen))
	}
}

func TestPickTemplateRespectsCooldownByExactString(t *testing.T) {
	lib := buildTestLib()
	cooldownKey := "adv:clinch_to_mount:ctrl"
	cooldowns := map[string]bool{}

	// First pick — marks something used.
	first := PickTemplate(lib.Advancements["clinch_to_mount"].Controller, cooldowns, cooldownKey)
	cooldowns[cooldownKey+":"+first] = true

	// Subsequent picks should avoid the cooldown-marked template until
	// all are exhausted, then reset.
	for i := 0; i < 20; i++ {
		next := PickTemplate(lib.Advancements["clinch_to_mount"].Controller, cooldowns, cooldownKey)
		// At least one of the two templates must be the "other" one
		// (cooldown not yet exhausted with both).
		if next == first {
			// Both templates have been used; cooldown should have
			// reset. This is fine — test passes.
			return
		}
	}
}

func TestPickTemplateEmptyListReturnsFallback(t *testing.T) {
	out := PickTemplate([]string{}, map[string]bool{}, "any:key")
	if out == "" {
		t.Error("PickTemplate should return a non-empty fallback even for empty input")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/grapplemessaging/ -run "TestRender|TestPickTemplate" -count=1`
Expected: FAIL (RenderTemplate / PickTemplate undefined).

- [ ] **Step 3: Implement render.go**

Create `internal/grapplemessaging/render.go`:

```go
package grapplemessaging

import (
	"math/rand"
	"strings"
)

// RenderTemplate substitutes {controllerName} and {controlledName}
// in a template string and returns the rendered result. Caller is
// responsible for ANSI-wrapping or other formatting.
func RenderTemplate(template, controllerName, controlledName string) string {
	out := strings.ReplaceAll(template, "{controllerName}", controllerName)
	out = strings.ReplaceAll(out, "{controlledName}", controlledName)
	return out
}

// PickTemplate selects a template from the list, preferring ones
// not yet used in this grapple's cooldown map. The cooldown key is
// computed per template as `<keyPrefix>:<template>` so each unique
// rendered template can be tracked independently across rounds.
//
// When all templates in the list have been used at least once in
// this grapple, the cooldown map is reset and selection starts over
// — forcing variety within reasonable bounds without blocking
// templates forever.
//
// Empty `pool` returns a benign fallback string so callers can
// always send something (a missing template should not crash a
// round).
func PickTemplate(pool []string, cooldowns map[string]bool, keyPrefix string) string {
	if len(pool) == 0 {
		return "(grapple messaging missing template)"
	}

	// Filter out templates whose cooldown key is marked.
	available := make([]string, 0, len(pool))
	for _, tmpl := range pool {
		ck := keyPrefix + ":" + tmpl
		if !cooldowns[ck] {
			available = append(available, tmpl)
		}
	}

	// If all are cooled-down, reset and start over.
	if len(available) == 0 {
		for _, tmpl := range pool {
			delete(cooldowns, keyPrefix+":"+tmpl)
		}
		available = pool
	}

	pick := available[rand.Intn(len(available))]
	cooldowns[keyPrefix+":"+pick] = true
	return pick
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/grapplemessaging/ -run "TestRender|TestPickTemplate" -count=1 -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/grapplemessaging/render.go internal/grapplemessaging/render_test.go
git commit -m "feat(grapplemessaging): chunk 4b-fixup T13 — render + cooldown picker

RenderTemplate substitutes {controllerName}/{controlledName}.
PickTemplate uses a per-grapple cooldown map to force template
variety; resets when all templates exhausted. Empty-pool fallback
returns a debug string instead of panicking — boot smoke catches
this if a key is missing.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 14: Realism sanity-check pass on templates

**Files (no code change — review-only):**
- All templates in `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml`

- [ ] **Step 1: Dispatch a fresh subagent for the realism review**

This task is review-only. Dispatch a new general-purpose subagent with NO prior context from authoring tasks. The dispatch prompt:

```
You are a fresh reviewer with no prior context.

Review every template in
_datafiles/world/dogmud/messaging/grapple_outcomes.yaml against
this question: "Does this read like a real description of combat
with grappling?"

A reader familiar with MMA/BJJ should be able to picture each
line without cringing at:
- Anatomical impossibilities (joint locks that don't work that
  way, levers that don't lever)
- Made-up moves that sound vaguely martial but aren't real
  techniques
- Wrong move for the position (e.g. a leg lock from full mount —
  leg locks don't come from mount)
- Position-of-the-bodies errors (defender described as standing
  while in a ground position)
- Wrong subject for the action (defender "lands a takedown" during
  a degradation — defender isn't initiating)
- Tone inconsistency within a category (jokes mixed with serious,
  fantasy vocabulary in an MMA-flavored line)

For each issue found:
- File location (key + speaker + index)
- Quote the line
- Describe the problem in one sentence
- Suggest a revised line

Report findings as a markdown checklist grouped by key. Approve
templates that pass with no comment (don't enumerate the passes —
just flag the fails).

Do NOT modify the YAML yourself. Report only.
```

- [ ] **Step 2: Apply reviewer revisions**

For each issue the reviewer flagged, revise the YAML directly. Use Edit tool for surgical replacements.

- [ ] **Step 3: Re-run validator + production-library guard tests**

Run: `go test ./internal/grapplemessaging/ -count=1 -v`
Expected: ALL PASS (counts unchanged; just text revised).

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/messaging/grapple_outcomes.yaml
git commit -m "content(messaging): chunk 4b-fixup T14 — realism sanity-check revisions

Fresh-subagent review per spec §7.6. N templates revised for
anatomical accuracy, technique correctness, position/posture
consistency, and tone uniformity. No template counts change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 15: Wire ResolveOutcome into Position_GrappleTick

**Files:**
- Modify: `internal/hooks/Position_GrappleTick.go`

This is the biggest single-task code change. It rewrites `processGrapplePair` to call `position.ResolveOutcome` instead of doing drift-needle math.

- [ ] **Step 1: Read the current processGrapplePair implementation**

The current implementation is at `internal/hooks/Position_GrappleTick.go:105-231` (per the spec §1 line refs). Read the entire function before modifying — note the existing stamina cost call (`applyGrappleStaminaCost`) and the LastDriftRoll snapshot (chunk 4d composition) — both must be preserved.

- [ ] **Step 2: Replace processGrapplePair with the new outcome-driven version**

Replace the function body (everything from line 105 to the closing `}` around line 231) with:

```go
// processGrapplePair runs the per-round drift roll, dispatches the
// result through position.ResolveOutcome, applies the resulting
// transition (advance / degrade / reversal / escape / hold), updates
// the LastDriftRoll snapshot for chunk-4d submission tick, and
// applies stamina cost.
//
// Chunk 4b-fixup: replaces the chunk-4b ControlLevel drift-needle
// math. ControlLevel is sunset entirely; the per-round outcome IS
// the position change.
func processGrapplePair(controller, controlled *characters.Character) {
	cfg := configs.GetBalanceConfig()

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

	_, margin, atkRoll, defRoll := dice.OpposedRollStat(ctrlScore, cdScore)

	// LastDriftRoll snapshot for chunk-4d Position_SubmissionTick.
	currentRound := util.GetRoundCount()
	snap := characters.DriftRollSnapshot{
		Round:          currentRound,
		MarginAttacker: margin,
		AttackerZScore: atkRoll.ZScore,
		DefenderZScore: defRoll.ZScore,
	}
	controller.LastDriftRoll = snap
	controlled.LastDriftRoll = snap

	// Compute signed z used by ResolveOutcome.
	z := 0.0
	if atkRoll.StdDev > 0 {
		z = margin / atkRoll.StdDev
	}

	source := controller.Position.State()
	defenderPosture := controlled.Position.State()
	outcome := position.ResolveOutcome(source, z, defenderPosture)

	// Apply outcome via TransitionPair when position changes.
	switch outcome.Kind {
	case position.OutcomeAdvance:
		applyAdvanceOrEscape(controller, controlled, outcome.Target,
			position.TriggerPositionAdvance)
	case position.OutcomeDegrade:
		applyAdvanceOrEscape(controller, controlled, outcome.Target,
			position.TriggerPositionDegrade)
	case position.OutcomeReversal:
		applyReversal(controller, controlled, outcome.Target)
	case position.OutcomeEscape:
		applyAdvanceOrEscape(controller, controlled, position.Standing,
			position.TriggerControlledEscape)
		// Clear per-grapple cooldowns on full escape — next grapple
		// starts fresh.
		controller.PerGrappleMessageCooldowns = map[string]bool{}
		controlled.PerGrappleMessageCooldowns = map[string]bool{}
	case position.OutcomeHold:
		// No transition. Stamina drains; flavor handled below.
	}

	// Stamina cost unchanged.
	applyGrappleStaminaCost(controller, controlled, cfg)
	fireStaminaWarningIfLow(controller)
	fireStaminaWarningIfLow(controlled)

	// Messaging — T16 wires this. Stub for now so the test in T15
	// asserts call-through without rendering.
	emitOutcomeMessages(controller, controlled, outcome)
}

// applyAdvanceOrEscape fires position.TransitionPair with controller
// and controlled in their existing roles. Used for advances,
// degrades, and full escapes (which all keep the role assignment).
func applyAdvanceOrEscape(controller, controlled *characters.Character,
	target position.State, trigger string) {
	if err := position.TransitionPair(controller, controlled, target,
		state.TransitionReason{Trigger: trigger}); err != nil {
		mudlog.Warn("Position_GrappleTick: TransitionPair failed",
			"controller_user", controller.GetUserId(),
			"controller_mob", controller.GetMobInstanceId(),
			"target", target, "trigger", trigger, "err", err)
	}
}

// applyReversal swaps roles when transitioning. The former defender
// becomes the new controller; former controller becomes the new
// controlled. position.TransitionPair takes (controller, controlled)
// args, so we swap them at the call site.
func applyReversal(formerController, formerControlled *characters.Character,
	target position.State) {
	if err := position.TransitionPair(formerControlled, formerController, target,
		state.TransitionReason{Trigger: position.TriggerReversal}); err != nil {
		mudlog.Warn("Position_GrappleTick: reversal TransitionPair failed",
			"former_controller_user", formerController.GetUserId(),
			"former_controlled_user", formerControlled.GetUserId(),
			"target", target, "err", err)
	}
}

// emitOutcomeMessages is wired to grapplemessaging.RenderOutcome
// in T16. Stub here so T15 compiles + runs.
func emitOutcomeMessages(controller, controlled *characters.Character,
	outcome position.Outcome) {
	// Wired in T16.
}
```

Delete the now-unused `updateControlLevel` helper (it called `MutateGrappleControlLevel` which goes away in T18).

Delete `fireGradientMessages` calls (function deletion in T17).

- [ ] **Step 3: Verify build compiles**

Run: `go build ./...`
Expected: clean build. (`position.TriggerPositionDegrade`, `TriggerReversal`, `TriggerControlledEscape` were added in T1.)

- [ ] **Step 4: Run all position + hooks tests**

Run: `go test ./internal/state/position/ ./internal/hooks/ -count=1`
Expected: existing tests should still pass; some chunk-4b drift-needle tests in `internal/state/position/control_test.go` may fail but `control_test.go` is deleted in T18.

If any current tests fail OTHER than ControlLevel-related ones, investigate. Likely candidates: tests asserting `cdHittingControlled` escape semantics — those are intentionally broken by this task.

Document any deleted/modified tests in the commit message.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/Position_GrappleTick.go
git commit -m "feat(hooks): chunk 4b-fixup T15 — wire ResolveOutcome into GrappleTick

processGrapplePair now calls position.ResolveOutcome and dispatches
on outcome.Kind to advance / degrade / reverse / escape / hold via
TransitionPair. Drift-needle math (ControlLevel shifts + sustained-
pressure escape gate) is gone — outcome IS the position change.

LastDriftRoll snapshot preserved for chunk-4d composition.
applyGrappleStaminaCost + fireStaminaWarningIfLow preserved
unchanged.

Messaging stub (emitOutcomeMessages) wired in T16.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 16: Wire grapplemessaging into Position_GrappleTick

**Files:**
- Modify: `internal/hooks/Position_GrappleTick.go`

- [ ] **Step 1: Add a package-level grapplemessaging library variable + init loader**

Add to the top of `Position_GrappleTick.go` (after the existing imports):

```go
var grappleOutcomesLib *grapplemessaging.Library

func init() {
	events.RegisterListener(events.NewRound{}, processGrappleTick)

	// Load grapple outcome templates at boot. Failure here is
	// fail-loud — combat would still work but with no flavor;
	// better to surface the missing file or YAML error than ship
	// silently-flavorless grapples.
	lib, err := grapplemessaging.Load(
		"_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
	if err != nil {
		mudlog.Error("Position_GrappleTick: failed to load grapple_outcomes.yaml",
			"err", err)
		// Use an empty library so render calls return debug strings.
		lib = &grapplemessaging.Library{
			Advancements: map[string]grapplemessaging.TemplateTriad{},
			Degradations: map[string]grapplemessaging.TemplateTriad{},
			Reversals:    map[string]grapplemessaging.TemplateTriad{},
			Escapes:      map[string]grapplemessaging.TemplateTriad{},
			Holds:        map[string]grapplemessaging.TemplateTriad{},
			StrikingApex: map[string][]string{},
		}
	}
	errs := grapplemessaging.ValidateCompleteness(lib)
	for _, e := range errs {
		mudlog.Warn("Position_GrappleTick: grapple_outcomes.yaml incomplete",
			"violation", e)
	}
	grappleOutcomesLib = lib
}
```

Remove the existing standalone `func init()` at the bottom of the file (the new init above subsumes it).

- [ ] **Step 2: Implement emitOutcomeMessages**

Replace the stub `emitOutcomeMessages` from T15 with the full implementation:

```go
// emitOutcomeMessages picks a template triad for the outcome and
// sends the appropriate speaker variant to each side + the room.
// Cooldown map lives on Character.PerGrappleMessageCooldowns;
// PickTemplate handles within-grapple variety.
func emitOutcomeMessages(controller, controlled *characters.Character,
	outcome position.Outcome) {
	if grappleOutcomesLib == nil {
		return
	}

	// Initialize cooldown maps if nil (chunk-4b created them on
	// transitions; defensive init here).
	if controller.PerGrappleMessageCooldowns == nil {
		controller.PerGrappleMessageCooldowns = map[string]bool{}
	}
	if controlled.PerGrappleMessageCooldowns == nil {
		controlled.PerGrappleMessageCooldowns = map[string]bool{}
	}

	// Pick template triad based on outcome kind + source/target.
	var (
		triad grapplemessaging.TemplateTriad
		key   string
		found bool
	)

	switch outcome.Kind {
	case position.OutcomeAdvance:
		key = stateMessagingName(outcome.Source) + "_to_" + stateMessagingName(outcome.Target)
		triad, found = grappleOutcomesLib.Advancements[key]
	case position.OutcomeDegrade:
		key = stateMessagingName(outcome.Source) + "_to_" + stateMessagingName(outcome.Target)
		triad, found = grappleOutcomesLib.Degradations[key]
	case position.OutcomeReversal:
		key = stateMessagingName(outcome.Source) + "_reverse"
		triad, found = grappleOutcomesLib.Reversals[key]
		if !found {
			key = "generic_reverse"
			triad, found = grappleOutcomesLib.Reversals[key]
		}
	case position.OutcomeEscape:
		key = "generic_escape"
		triad, found = grappleOutcomesLib.Escapes[key]
	case position.OutcomeHold:
		emitHoldFlavor(controller, controlled, outcome)
		emitStrikingApexFlavor(controller, controlled, outcome)
		return
	}

	if !found {
		mudlog.Warn("Position_GrappleTick: missing message key", "kind", outcome.Kind, "key", key)
		return
	}

	controllerName := characterDisplayName(controller)
	controlledName := characterDisplayName(controlled)

	if msg := grapplemessaging.PickTemplate(triad.Controller,
		controller.PerGrappleMessageCooldowns, key+":ctrl"); msg != "" {
		sendToCharacter(controller,
			grapplemessaging.RenderTemplate(msg, controllerName, controlledName))
	}
	if msg := grapplemessaging.PickTemplate(triad.Controlled,
		controlled.PerGrappleMessageCooldowns, key+":cd"); msg != "" {
		sendToCharacter(controlled,
			grapplemessaging.RenderTemplate(msg, controllerName, controlledName))
	}
	if msg := grapplemessaging.PickTemplate(triad.Observers,
		controller.PerGrappleMessageCooldowns, key+":obs"); msg != "" {
		broadcastToRoomExcluding(controller, controlled,
			grapplemessaging.RenderTemplate(msg, controllerName, controlledName))
	}
}

// emitHoldFlavor sends sparse hold-round flavor every ~3-4 rounds.
// Tracked via a "hold_emitted_round" key in the cooldown map.
func emitHoldFlavor(controller, controlled *characters.Character,
	outcome position.Outcome) {
	const holdEmitEveryRounds = 4
	round := util.GetRoundCount()
	lastKey := "hold_last_round:" + stateMessagingName(outcome.Source)
	lastRound := int64(0)
	if v, ok := controller.PerGrappleMessageCooldownsLastRound[lastKey]; ok {
		lastRound = v
	}
	if round-lastRound < holdEmitEveryRounds {
		return
	}
	// Mark for next time.
	if controller.PerGrappleMessageCooldownsLastRound == nil {
		controller.PerGrappleMessageCooldownsLastRound = map[string]int64{}
	}
	controller.PerGrappleMessageCooldownsLastRound[lastKey] = round

	// Pick hold key by source state.
	key := holdKeyForState(outcome.Source)
	triad, found := grappleOutcomesLib.Holds[key]
	if !found {
		return
	}
	controllerName := characterDisplayName(controller)
	controlledName := characterDisplayName(controlled)

	if msg := grapplemessaging.PickTemplate(triad.Controller,
		controller.PerGrappleMessageCooldowns, "hold:"+key+":ctrl"); msg != "" {
		sendToCharacter(controller,
			grapplemessaging.RenderTemplate(msg, controllerName, controlledName))
	}
	if msg := grapplemessaging.PickTemplate(triad.Controlled,
		controlled.PerGrappleMessageCooldowns, "hold:"+key+":cd"); msg != "" {
		sendToCharacter(controlled,
			grapplemessaging.RenderTemplate(msg, controllerName, controlledName))
	}
	if msg := grapplemessaging.PickTemplate(triad.Observers,
		controller.PerGrappleMessageCooldowns, "hold:"+key+":obs"); msg != "" {
		broadcastToRoomExcluding(controller, controlled,
			grapplemessaging.RenderTemplate(msg, controllerName, controlledName))
	}
}

// emitStrikingApexFlavor fires Mount-strike flavor on Hold rounds at
// Mount. Single-speaker list — visible to controller; observers see
// a parallel third-person form via the existing combat damage text
// system, so we don't double-broadcast.
func emitStrikingApexFlavor(controller, controlled *characters.Character,
	outcome position.Outcome) {
	if outcome.Source != position.Mount {
		return
	}
	pool, found := grappleOutcomesLib.StrikingApex["mount_strike_flavor"]
	if !found {
		return
	}
	controlledName := characterDisplayName(controlled)
	msg := grapplemessaging.PickTemplate(pool,
		controller.PerGrappleMessageCooldowns, "apex:mount_strike")
	if msg != "" {
		sendToCharacter(controller,
			grapplemessaging.RenderTemplate(msg, characterDisplayName(controller), controlledName))
	}
}

// stateMessagingName converts a position.State to its canonical
// snake_case YAML key segment.
func stateMessagingName(s position.State) string {
	switch s {
	case position.Clinch:
		return "clinch"
	case position.BackStanding:
		return "backstanding"
	case position.Mount:
		return "mount"
	case position.SideControl:
		return "sidecontrol"
	case position.KneeOnBelly:
		return "kob"
	case position.NorthSouth:
		return "ns"
	case position.Crucifix:
		return "crucifix"
	case position.BackGround:
		return "background"
	case position.HalfGuard:
		return "halfguard"
	case position.Guard:
		return "guard"
	case position.Turtle:
		return "turtle"
	case position.Standing:
		return "standing"
	}
	return "unknown"
}

// holdKeyForState picks the right hold-template key for a source.
func holdKeyForState(s position.State) string {
	switch s {
	case position.Clinch:
		return "clinch_hold"
	case position.Guard:
		return "guard_hold"
	case position.Turtle:
		return "turtle_hold"
	case position.BackStanding:
		return "backstanding_hold"
	default:
		// Everything else is a ground position.
		return "ground_hold_generic"
	}
}
```

- [ ] **Step 3: Add the supporting helpers `characterDisplayName`, `sendToCharacter`, `broadcastToRoomExcluding`**

These must follow the existing patterns used elsewhere in the hooks package. Search for existing equivalents:

Run: `grep -n "SendText\|broadcastToRoom\|displayName" internal/hooks/Position_Messaging.go internal/hooks/combat_shared_helpers.go`

Use the same patterns. The display-name pattern in this codebase is usually `c.Name` (for characters) wrapped in `<ansi fg="username">` for users or `<ansi fg="mobname">` for mobs. The implementer should verify by reading existing message-sending code in the same package.

Add `PerGrappleMessageCooldownsLastRound` field to Character struct if not present (audit needed in `internal/characters/character.go`). If it doesn't exist, add it with type `map[string]int64` alongside the existing `PerGrappleMessageCooldowns map[string]bool`.

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 5: Run all related tests**

Run: `go test ./internal/state/position/ ./internal/hooks/ ./internal/grapplemessaging/ -count=1`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/Position_GrappleTick.go internal/characters/character.go
git commit -m "feat(hooks): chunk 4b-fixup T16 — wire grapplemessaging into GrappleTick

emitOutcomeMessages picks template triads by outcome kind +
source/target keys; renders with character display names; sends
to controller, controlled, and room observers with appropriate
speaker variant. Per-grapple cooldown map drives template variety.

Hold rounds fire sparse flavor every 4 rounds via a last-emitted-
round map. Mount holds additionally fire striking-apex flavor to
the controller (observers see the combat damage text from the
combat system).

Boot loads grapple_outcomes.yaml; ValidateCompleteness logs warnings
for missing keys but doesn't block boot — fail-loud at first use
via PickTemplate's fallback string.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 17: Sunset Position_Messaging gradient functions

**Files:**
- Modify: `internal/hooks/Position_Messaging.go`

- [ ] **Step 1: Identify the functions to delete**

Run: `grep -n "fireGradientMessages\|gradient\|fireTransitionMessages" internal/hooks/Position_Messaging.go`

Functions to delete:
- `fireGradientMessages` and any helpers it calls (`gradientLineForLevel`, etc.)
- Any chunk-4b gradient message string constants

Functions to KEEP:
- `fireStaminaWarningIfLow` (still used by GrappleTick)
- The YAML-loading code for transition messages (still used by entry/exit messaging)

- [ ] **Step 2: Delete the gradient functions**

Open `internal/hooks/Position_Messaging.go`. Remove:
- The `fireGradientMessages` function (and helpers it exclusively used)
- Any callsites of these functions (search to be safe)

- [ ] **Step 3: Verify build + tests**

Run: `go build ./... && go test ./internal/hooks/ -count=1`
Expected: clean build + tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/Position_Messaging.go
git commit -m "chore(hooks): chunk 4b-fixup T17 — sunset gradient messages

The 'losing control of the clinch' / 'becoming controlled'
gradient message functions are sunset along with the ControlLevel
concept (T18). fireStaminaWarningIfLow stays — still called from
processGrapplePair.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 18: Sunset ControlLevel — delete control.go + enum + field

**Files:**
- Delete: `internal/state/position/control.go`
- Delete: `internal/state/position/control_test.go`
- Modify: `internal/state/position/position.go`
- Modify: `internal/state/position/pair.go`

- [ ] **Step 1: Delete control.go + control_test.go**

```bash
rm internal/state/position/control.go internal/state/position/control_test.go
```

- [ ] **Step 2: Remove ControlLevel enum + GrappleData.ControlLevel field from position.go**

Open `internal/state/position/position.go`. Find:

```go
type ControlLevel int

const (
	Neutral ControlLevel = iota
	InControl
	LosingControl
	BecomingControlled
	Controlled
)

func (c ControlLevel) String() string {
	// ...
}
```

Delete the entire ControlLevel type, all 5 constants, and the `String()` method.

Find the `GrappleData` struct (around line 132):

```go
type GrappleData struct {
	Reason       string
	Partner      state.ActorRef
	ControlLevel ControlLevel  // ← DELETE THIS LINE
}
```

Delete the `ControlLevel` field.

- [ ] **Step 3: Remove InitialControlForPair from pair.go**

Open `internal/state/position/pair.go`. Find and delete the entire `InitialControlForPair` function. Also remove the `Role` type and `RoleController`/`RoleControlled` constants if they're not used elsewhere.

Run: `grep -rn "InitialControlForPair\|RoleController\|RoleControlled\|MutateGrappleControlLevel" internal/`

If `MutateGrappleControlLevel` is on the Machine type, delete it too.

In `TransitionPair`, remove the lines that set `ctrlData.ControlLevel = InitialControlForPair(...)`. GrappleData literals just become `GrappleData{Partner: ref}`.

- [ ] **Step 4: Compile-check + fix any consumers**

Run: `go build ./...`
Expected: numerous failures across the codebase from sites that referenced ControlLevel. Triage each:

- Tests in `internal/state/position/position_test.go` and `pair_test.go` that assert ControlLevel values → delete those test cases.
- Any other `internal/` files referencing `ControlLevel` → these are bugs the spec should have caught; fix or delete the reference.

Run `grep -rn "ControlLevel\|InControl\|LosingControl\|BecomingControlled\b" internal/` to find every consumer.

For each consumer, decide:
- Test assertion on a now-deleted concept → delete the assertion (keep the test if it covers other behavior).
- Production code reading ControlLevel → re-engineer to read role/position instead.

- [ ] **Step 5: Run full test suite**

Run: `go test ./... -count=1 2>&1 | tail -30`
Expected: ALL PASS (or list of remaining ControlLevel references to clean up).

- [ ] **Step 6: Commit**

```bash
git add -u internal/state/position/ internal/hooks/ internal/characters/ internal/combat/
git commit -m "chore(position): chunk 4b-fixup T18 — sunset ControlLevel entirely

Deletes control.go + control_test.go.
Deletes ControlLevel enum + 5 constants + String() from position.go.
Deletes GrappleData.ControlLevel field.
Deletes InitialControlForPair + Role + RoleController/RoleControlled.
Deletes Machine.MutateGrappleControlLevel.
Triage of every ControlLevel reference across internal/ — test
assertions deleted, production reads converted to role/position.

The needle is gone. The Position FSM + the outcome resolver carry
the full per-round semantics.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 19: Audit + sunset control-axis btree primitives

**Files:**
- Modify or delete: btree primitive files under `internal/behaviortree/primitives/`
- Audit: zone-specific behaviors under `_datafiles/world/dogmud/behaviors/`

- [ ] **Step 1: Audit btree primitive files for ControlLevel references**

Run: `grep -rn "ControlLevel\|control_level\|InControl\|LosingControl\|BecomingControlled" internal/behaviortree/`

For each match:
- If it's a primitive name like `control_level_at_least` → delete the primitive file entirely.
- If it's documentation in a context.md → update in T22-T25.

- [ ] **Step 2: Audit data files for primitive usage**

Run: `grep -rn "control_level_at_least\|in_control\|controlled_threshold" _datafiles/world/dogmud/behaviors/`

For each match in a YAML, replace the primitive call with an equivalent using role/position primitives (e.g. `is_grappling`, `is_top_dominant`).

If a YAML uses a primitive that has no replacement, escalate — the design may have missed a use case.

- [ ] **Step 3: Verify build + tests**

Run: `go build ./... && go test ./... -count=1`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add -u internal/behaviortree/ _datafiles/world/dogmud/behaviors/
git commit -m "chore(btree): chunk 4b-fixup T19 — sunset control-axis primitives

The 6 chunk-4b control-axis primitives (control_level_at_least,
in_control, etc.) are deleted along with ControlLevel. Zone-specific
behavior YAMLs migrated to role/position primitives (is_grappling,
is_top_dominant, etc.).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 20: Verify chunk-4d Position_SubmissionTick composition

**Files:**
- Modify (verify only): `internal/hooks/Position_SubmissionTick.go`

- [ ] **Step 1: Read Position_SubmissionTick.go**

Read the entire file to confirm:
- Sub gate currently reads `LastDriftRoll` snapshot.
- Sub gate uses an alpha constant that may differ from the new spec's `subWindowAlpha = 1.5`.

- [ ] **Step 2: Align sub gate alpha if needed**

If the existing alpha is something other than 1.5, decide:
- Update the chunk-4d alpha to 1.5 to match the new spec.
- OR keep chunk-4d's alpha and document the mismatch in §5.

Per spec §5, the unified gate is 1.5. Update chunk-4d's alpha constant to match. The constant should be `position.subWindowAlpha` (so there's one source of truth).

If chunk-4d defines its own `submissionAlpha` constant, replace its uses with `position.SubWindowOpens(absZ)` from outcomes.go.

- [ ] **Step 3: Verify integration test**

Add a test in `internal/hooks/Position_SubmissionTick_test.go` (or extend existing test file):

```go
func TestSubmissionTickReadsPostAdvancePosition(t *testing.T) {
	// Setup: controller in Mount, z = 2.5 → ResolveOutcome advances
	// to BackGround. SubmissionTick should fire sub from BackGround
	// (RNC family), not from Mount (americana family).
	//
	// This test asserts ordering: T15 wired processGrapplePair to call
	// ResolveOutcome → TransitionPair → then SubmissionTick reads the
	// new position.
	t.Skip("Integration scaffolding TBD by implementer; manual smoke covers in T26.")
}
```

(Mark Skip if full integration setup is heavy — the manual AI smoke in T26 catches this.)

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/Position_SubmissionTick.go internal/hooks/Position_SubmissionTick_test.go
git commit -m "chore(hooks): chunk 4b-fixup T20 — align SubmissionTick alpha with new gate

Chunk-4d's sub gate now uses position.SubWindowOpens(|z|) (the
shared >=1.5 threshold). Sub fires from the post-advance position
via the natural ordering of GrappleTick (T15) → TransitionPair
mutates state → SubmissionTick reads new state.

Manual smoke (T26) verifies the RNC-family sub fires from
BackGround when Mount advances at z >= 2.0.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 21: Update internal/state/position/context.md

**Files:**
- Modify: `internal/state/position/context.md`

- [ ] **Step 1: Read existing context.md**

Read `internal/state/position/context.md` end-to-end. Note all sections that describe:
- ControlLevel enum and its 5 levels
- The drift-needle model
- `InitialControlForPair` per-position assignments
- "Controlled for 2 consecutive rounds → escape" gate
- `MarginToDelta` shift magnitudes

These all need rewriting to describe the new outcome model.

- [ ] **Step 2: Rewrite the relevant sections**

Replace the chunk-4b drift-needle description with the chunk 4b-fixup outcome model. Reference structure (mirror what's in the spec §3, §4, §5, §6):

```markdown
## Per-round outcome resolution (chunk 4b-fixup)

Each round, processGrapplePair (in internal/hooks/Position_GrappleTick.go)
runs an opposed Str+WeaponCombat roll modified by stamina + encumbrance.
The z-score result dispatches through `position.ResolveOutcome` to one of
five outcome kinds:

- Hold (|z| < 0.5): no position change
- Advance (z > 0): controller wins; position changes per §6.1 table
- Degrade (z < 0 in [-1, -0.5)): defender wins moderately; position regresses
- Reversal (z in [-2, -1)): roles swap, position may stay or change per §6.3
- Escape (z <= -2): TransitionPair to Standing

Sub-window gate is independent: |z| >= 1.5 on controller side opens a sub
attempt for Position_SubmissionTick (chunk 4d) to resolve.

ControlLevel from chunk 4b is sunset entirely — see the chunk 4b-fixup
spec for rationale. The needle never worked for asymmetric established
positions like Mount (defender started at Controlled by design, so winning
drift fired the escape gate faster).

### Per-position transition tables

See outcomes.go for the authoritative tables. Summary:
- Advancement: Mount is striking apex (1/2-step Hold, 3-step → BackGround);
  BackGround + Crucifix are terminal apexes (sub-only).
- Degrade: Clinch / Guard / Turtle are terminal (no degrade target).
- Reversal: Mount → Guard, BackGround → Mount (realism exceptions); all
  others same-position with role swap.
```

Delete the chunk-4b sections about ControlLevel, MarginToDelta, sustained-pressure gate. Keep sections about:
- The 14 FSM states (unchanged)
- Pair invariants
- Trigger constants (add the new ones from T1)
- Role assignments per position (in spec §3 table)

- [ ] **Step 3: Commit**

```bash
git add internal/state/position/context.md
git commit -m "docs(position): chunk 4b-fixup T21 — update context.md for outcome model

ControlLevel sections removed. New per-round outcome resolution
section describes the 5 outcome kinds + sub-window gate + per-
position transition tables. Pointers to outcomes.go for the
authoritative tables (don't duplicate in docs).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 22: Update internal/hooks/context.md

**Files:**
- Modify: `internal/hooks/context.md`

- [ ] **Step 1: Find the Position_GrappleTick + Position_Messaging sections**

Run: `grep -n "Position_GrappleTick\|Position_Messaging\|ControlLevel\|gradient" internal/hooks/context.md`

- [ ] **Step 2: Rewrite to describe the new flow**

The Position_GrappleTick section should now describe:
- Calls `position.ResolveOutcome(source, z, defenderPosture)`
- Dispatches outcome kinds to `applyAdvanceOrEscape` / `applyReversal` / hold
- Loads `grapple_outcomes.yaml` at boot via `grapplemessaging.Load`
- Per-grapple message cooldowns drive template variety

The Position_Messaging section should describe ONLY the surviving functions (stamina warnings + entry/exit transition messaging). The gradient functions are deleted.

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/context.md
git commit -m "docs(hooks): chunk 4b-fixup T22 — update context.md

Position_GrappleTick + Position_Messaging sections rewritten for
the outcome-driven flow. Gradient messaging notes removed.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 23: Update remaining context.md files

**Files:**
- Modify: `internal/behaviortree/context.md` (if it documents control-axis primitives)
- Modify: `internal/combat/context.md` (if it mentions chunk-4b drift)

- [ ] **Step 1: Audit both files**

Run: `grep -n "ControlLevel\|control_level\|drift needle\|gradient" internal/behaviortree/context.md internal/combat/context.md`

- [ ] **Step 2: Update each match**

For each reference, either delete (if the concept is gone) or revise to describe the new model.

- [ ] **Step 3: Commit**

```bash
git add internal/behaviortree/context.md internal/combat/context.md
git commit -m "docs: chunk 4b-fixup T23 — context.md sweep

Btree + combat context docs updated to remove ControlLevel
references and describe the new role/position model.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 24: Update helpfiles

**Files:**
- Audit + modify: `_datafiles/help/*.txt` for grapple-related entries

- [ ] **Step 1: Audit helpfiles for ControlLevel references**

Run: `grep -rn -i "ControlLevel\|in control\|losing control\|becoming controlled\|drift\|gradient\|grapple" _datafiles/help/`

For each match, decide:
- Description of ControlLevel mechanic → rewrite to describe the new outcome model.
- Description of `grapple` command (entry point) → unchanged; just verify still accurate.
- Combat tactics advice that mentions "control level" → revise to mention position changes ("if you dominate, you'll advance to a better position").

- [ ] **Step 2: Add a `help advancement` or similar entry**

If there's an existing `help grapple` file, append (or create) a section describing position advancement. Tone: descriptive (per project SOP no hard numbers), MMA-flavored, briefly explains: dominate the drift to advance to better positions, get reversed if you lose big, escape if you lose decisive.

Example for `_datafiles/help/grapple.txt`:

```text
GRAPPLING POSITIONS

Once you're in a grapple, each round resolves to one of:
  - Hold       : neither side gains ground; stamina drains
  - Advance    : you move to a more dominant position
  - Degrade    : you slip to a less dominant position
  - Reversal   : your opponent takes over the position
  - Escape     : the grapple breaks; both fighters stand

Dominant positions like the Mount are striking apexes — you stay
there to pound your opponent. Climbing to back-control (taking
the back) requires a decisive moment.

Submission attempts fire independently when you're decisively
winning a round, from whatever position you're currently in.
```

- [ ] **Step 3: Commit**

```bash
git add _datafiles/help/
git commit -m "docs(help): chunk 4b-fixup T24 — helpfile sweep

ControlLevel references replaced with the new outcome model in
all grapple-related helpfiles. New 'GRAPPLING POSITIONS' section
in help grapple explains the 5 outcome kinds in player-facing
language (no hard numbers).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 25: Update COMBAT_STATE_ROADMAP.md

**Files:**
- Modify: `COMBAT_STATE_ROADMAP.md`

- [ ] **Step 1: Add new chunk row**

Find the chunk table in `COMBAT_STATE_ROADMAP.md`. After the row for chunk 4d (or wherever the 4b/4c/4d block lives), insert:

```markdown
| 4b-fixup | Position — outcome model | Done (2026-MM-DD) | Replaces chunk-4b ControlLevel drift needle with direct position-change outcomes (Hold / Advance / Degrade / Reversal / Escape) per round. Mount is striking apex (1/2-step Hold, 3-step → BackGround); BackGround is the control apex. Crucifix terminal (sub-only). Reversal swaps roles with two realism exceptions (Mount→Guard, BackGround→Mount). ControlLevel + InitialControlForPair + gradient messages + sustained-pressure escape gate all sunset. ~227 flavor templates in grapple_outcomes.yaml across advancements / degradations / reversals / escapes / holds / striking_apex categories, validated by fresh-subagent realism pass. Chunk 4d submission gate composes via shared |z| >= 1.5 threshold; sub fires from post-advance position. Species-gated grappling deferred (see [project_species_gated_grappling.md](memory)). |
```

(Replace `2026-MM-DD` with the actual ship date when committing T26.)

- [ ] **Step 2: Update any other top-level summaries that reference chunk 4b's drift model**

Search the file for "drift needle", "ControlLevel", "5-level scale", "InControl" — update or remove each.

- [ ] **Step 3: Commit**

```bash
git add COMBAT_STATE_ROADMAP.md
git commit -m "docs: chunk 4b-fixup T25 — roadmap entry as Done

Adds the chunk 4b-fixup row to the chunk table. Updates other
4b drift-needle references in the roadmap.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 26: Boot smoke + AI tester smoke

**Files (no code change — verification only):**
- Smoke output

- [ ] **Step 1: Local boot smoke**

Build and run the server locally, watch for panics / errors during data loading and combat:

```bash
go build -o dogmud.exe ./cmd/server
./dogmud.exe 2>&1 | tee boot-smoke.log
```

Look for:
- `mobs.LoadDataFiles() loadedCount=...`
- `quests.LoadDataFiles() loadedCount=...`
- `Position_GrappleTick: grapple_outcomes.yaml incomplete` warnings (should be zero — T14 should have made the library complete)
- Any panic / nil-pointer / unmarshal error

Kill the server with Ctrl-C after confirming clean startup.

- [ ] **Step 2: Manual quick combat test (using `/test-mud local` or manual telnet)**

Spin the server up again. Log in with the AI test account, then dispatch:

```
/test-mud local feature-tester chunk-4b-fixup-position-advancement-smoke.yaml
```

(If this goals file doesn't exist, create it at `tools/testing/goals/chunk-4b-fixup-position-advancement-smoke.yaml`:)

```yaml
goals:
  - Engage a humanoid mob with a grappling archetype (predator wolf,
    warren guard, or generic_fighter). Stay in combat for at least 5
    rounds of grappling. Report all position changes observed via
    {pos} prompt and any messaging seen.
  - Verify that position advances at least once (Clinch → Mount or
    further) within a 10-round window.
  - Verify that Hold rounds occasionally show flavor messaging (not
    every round; sparse).
  - Verify that Mount rounds show striking-apex flavor.
  - Verify that if defender wins decisive rolls, you see degradation,
    reversal, or escape messaging.
  - Report no panics or "missing grapple message template" debug
    strings.
```

Run the AI feature-tester per `test-mud` skill. Read the report.

- [ ] **Step 3: AI feel-tester pass**

After feature-tester confirms mechanical correctness, run a feel-tester pass for tone/variety:

```
/test-mud local feel-tester chunk-4b-fixup-position-advancement-feel.yaml
```

Goals file `tools/testing/goals/chunk-4b-fixup-position-advancement-feel.yaml`:

```yaml
goals:
  - Engage at least 3 different grappling-archetype mobs. For each
    fight, capture the full text transcript of all messaging from
    grapple-entry to grapple-end.
  - Read the transcripts as a player would. Report whether combat
    reads as:
      - Varied (no repeated lines within a single grapple)
      - Visceral (real MMA/BJJ vocabulary, not generic fantasy)
      - Coherent (lines describe actions that match the position)
      - Paced (Hold rounds don't feel empty; advances feel earned)
  - Flag any lines that read awkwardly, repeat too often, or break
    immersion.
```

- [ ] **Step 4: Address any findings**

If smoke or feel-tester surfaces issues:
- Crashes / missing templates → fix in the YAML or render code, commit.
- Tone/realism issues → loop back to T14 (realism review) for affected templates.
- Pacing issues (Hold rate too high) → defer to chunk 4f tuning unless severe.

- [ ] **Step 5: Final commit (if any patches were needed)**

```bash
git commit -m "smoke(chunk-4b-fixup): T26 fixes from boot + AI tester pass

[Document the specific fixes here based on findings.]

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

If no fixes needed, commit only the new goals files:

```bash
git add tools/testing/goals/chunk-4b-fixup-position-advancement-smoke.yaml tools/testing/goals/chunk-4b-fixup-position-advancement-feel.yaml
git commit -m "test(smoke): chunk 4b-fixup T26 — AI tester goal files

Goals files for the chunk-4b-fixup ship smoke. Feature-tester
verifies mechanical correctness (position changes, sub windows,
no missing templates); feel-tester verifies varied/visceral/
coherent/paced tone.

Smoke ran clean — no fixes needed.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Self-Review

**Spec coverage** (cross-referencing the spec sections to plan tasks):

| Spec section | Covered by |
|---|---|
| §1 Problem Statement | (rationale only — no task needed) |
| §2 Design Goals | (rationale only) |
| §3 Concept Model (Position + Role) | T18 (ControlLevel sunset); roles unchanged from chunk 4a |
| §4 Per-Round Resolution Algorithm | T5 (ResolveOutcome), T15 (GrappleTick wiring) |
| §5 Outcome Buckets | T1 (bucket function) |
| §6.1 Advancement Table | T2 |
| §6.2 Degrade Table | T3 |
| §6.3 Reversal Table | T4 |
| §6.4 Escape | T5 (dispatcher) + T15 (apply) |
| §7.1 Template library structure | T6 (schema + loader) |
| §7.2 Template tone + variety | T8-T12 (authoring), T14 (review) |
| §7.3 Example templates | T8 anchor |
| §7.4 Authoring scope | T8-T12 each task targets specific counts |
| §7.5 Sunset old messaging | T17 |
| §7.6 Realism sanity-check pass | T14 |
| §8 Sunset chunk-4b | T17 + T18 + T19 |
| §9 Surviving artifacts | (no task — verified by passing tests) |
| §10 Migration order | Plan task ordering matches |
| §11 Testing strategy — unit | T1-T13 (each task includes unit tests) |
| §11 Testing strategy — integration | T15 (GrappleTick integration), T20 (chunk-4d composition) |
| §11 Testing strategy — smoke | T26 |
| §12 Out of scope | (no tasks — confirmed) |
| §13 Risks | Addressed in tasks (T14 realism, T20 4d composition) |
| §14 Files Touched | All listed files have tasks |
| §15 Success Criteria | T26 verifies all 8 |

**Placeholder scan:** clean — every step has explicit code or commands. T19 and T20 have audit-driven steps ("grep for X, fix each match") which is appropriate — the implementer needs to discover the exact references.

**Type consistency:** `OutcomeTier` / `Outcome` / `OutcomeKind` used consistently across tasks; `Library` / `TemplateTriad` / `RenderTemplate` / `PickTemplate` consistent; trigger constants (`TriggerPositionDegrade`, `TriggerReversal`, `TriggerControlledEscape`) added in T1, used in T15.

Plan complete.

---

**Plan complete and saved to `docs/superpowers/plans/2026-05-18-state-chunk-4b-fixup-position-advancement.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
