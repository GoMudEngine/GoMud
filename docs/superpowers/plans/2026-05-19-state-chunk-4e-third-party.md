# State Chunk 4e — Position Third-Party + Defense Degradation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Spec:** `docs/superpowers/specs/2026-05-19-state-chunk-4e-third-party-design.md`

**Goal:** Implement the five chunk 4e deliverables — position-tiered hit modifiers, eat/drink offense restrictions, outside-damage control degradation, mob AI tiebreaker bias, sub-interrupt risk — so position dominance translates to combat math and grapples interact realistically with multi-combatant scenes.

**Architecture:** New `internal/state/position/modifiers.go` exposes two pure lookup functions consumed by the combat hit-roll path (`internal/combat/combat.go` at the `calcAttackScore`→`runBestOfAllDefense` site). New per-Character per-round trackers (`OutsideHitDisruptedRound`, `SubInterruptDamageThisRound`) drive damage-pipeline hooks that feed chunk 4b-fixup-2's `applyControlShift` and chunk 4d's `Position_SubmissionTick`. Eat/drink commands gain a single pre-check. Mob target picker gains a tie-band tiebreaker for `IsBeingControlled` targets. Spell disruption is audited and fixed only if the audit shows gaps.

**Tech Stack:** Go 1.21+, existing `dice.OpposedRollStat`, existing `position.IsThirdPartyAttack` (combat_helpers.go:426), existing `applyControlShift` (Position_GrappleTick.go), existing `Position_SubmissionTick` (hooks/). No new dependencies.

---

## File Structure

### Created
| Path | Responsibility |
|---|---|
| `internal/state/position/modifiers.go` | Two pure lookup functions: `TargetSideHitModifier(state, role)` + `AttackerSelfHitModifier(state, role)`; modifier tables as code constants |
| `internal/state/position/modifiers_test.go` | Unit tests for every (position, role) cell + composition examples |
| `internal/combat/position_hit_integration_test.go` (optional, see T2) | Integration test verifying calcAttackScore applies the modifiers correctly |

### Modified
| Path | Change |
|---|---|
| `internal/characters/character.go` | Add `OutsideHitDisruptedRound int64` + `SubInterruptDamageThisRound float64` fields; reset in `ResetForMobInstance` |
| `internal/configs/config.balance.go` | Add `ControlDegradeOnOutsideHit` bool + `SubInterruptDamageThresholdPct` float64 |
| `_datafiles/config.yaml` | Default values for the two new knobs |
| `internal/combat/combat.go` | Multiply `attackScore` by position modifiers before `runBestOfAllDefense` |
| `internal/combat/damage_pipeline.go` (or wherever damage application happens — verify in T6) | Fire §5 control degrade + §7 sub-interrupt tracking hooks after damage application |
| `internal/hooks/Position_SubmissionTick.go` | Check `SubInterruptDamageThisRound` before resolving sub outcome; force Bad tier if > 0 |
| `internal/usercommands/eat.go`, `internal/usercommands/drink.go` | Pre-check `IsGrappling()` and reject |
| `internal/mobcommands/eat.go`, `internal/mobcommands/drink.go` (if they exist; verify in T3) | Same pre-check |
| `internal/combat/ai.go` or wherever mob target picker lives (verify in T9) | Tie-band tiebreaker for grappled-controlled targets |
| `internal/state/position/context.md` | Document modifier tables |
| `internal/combat/context.md` | Document hit-roll modifier integration |
| `internal/hooks/context.md` | Document damage-pipeline hooks for §5/§7 |
| `_datafiles/world/dogmud/templates/help/grapple.template` | Brief note on outside-hit disruption + eat/drink restriction (descriptive, no hard numbers) |
| `COMBAT_STATE_ROADMAP.md` | Chunk 4e row as Done |

### Audit-only (may or may not be modified depending on findings)
| Path | T4 (spell disruption audit) |
|---|---|
| `internal/hooks/Activity_Cascades.go` or wherever cast-cancel happens | If audit finds gaps, add catch-all "grappled implies disruption-equivalent-to-prone" path. Otherwise comment-only. |

---

## Task 1: Position modifier tables + unit tests

**Files:**
- Create: `internal/state/position/modifiers.go`
- Create: `internal/state/position/modifiers_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/state/position/modifiers_test.go`:

```go
package position

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state/control"
)

func TestTargetSideHitModifier_Standing(t *testing.T) {
	got := TargetSideHitModifier(Standing, control.Neutral)
	if got != 1.0 {
		t.Errorf("Standing → %v, want 1.0", got)
	}
}

func TestTargetSideHitModifier_Mount(t *testing.T) {
	if got := TargetSideHitModifier(Mount, control.Controlling); got != 1.12 {
		t.Errorf("Mount controller → %v, want 1.12", got)
	}
	if got := TargetSideHitModifier(Mount, control.Controlled); got != 1.20 {
		t.Errorf("Mount controlled → %v, want 1.20", got)
	}
}

func TestTargetSideHitModifier_Crucifix(t *testing.T) {
	if got := TargetSideHitModifier(Crucifix, control.Controlled); got != 1.25 {
		t.Errorf("Crucifix controlled → %v, want 1.25", got)
	}
}

func TestTargetSideHitModifier_GuardInverted(t *testing.T) {
	if got := TargetSideHitModifier(Guard, control.Controlling); got != 1.08 {
		t.Errorf("Guard controller (bottom) → %v, want 1.08", got)
	}
	if got := TargetSideHitModifier(Guard, control.Controlled); got != 1.10 {
		t.Errorf("Guard controlled (top) → %v, want 1.10", got)
	}
}

func TestAttackerSelfHitModifier_Standing(t *testing.T) {
	got := AttackerSelfHitModifier(Standing, control.Neutral)
	if got != 1.0 {
		t.Errorf("Standing → %v, want 1.0", got)
	}
}

func TestAttackerSelfHitModifier_MountControlling(t *testing.T) {
	got := AttackerSelfHitModifier(Mount, control.Controlling)
	if got != 1.10 {
		t.Errorf("Mount controller (apex) → %v, want 1.10", got)
	}
}

func TestAttackerSelfHitModifier_MountControlled(t *testing.T) {
	got := AttackerSelfHitModifier(Mount, control.Controlled)
	if got != 0.70 {
		t.Errorf("Mount controlled (under) → %v, want 0.70", got)
	}
}

func TestAttackerSelfHitModifier_Crucifix(t *testing.T) {
	if got := AttackerSelfHitModifier(Crucifix, control.Controlled); got != 0.50 {
		t.Errorf("Crucifix controlled (arm trapped) → %v, want 0.50", got)
	}
}

func TestComposition_MountControllerSwingingAtControlled(t *testing.T) {
	// Bug-report scenario: Mount controller's hit modifier composes to ~1.32
	atkSelf := AttackerSelfHitModifier(Mount, control.Controlling)
	tgtSide := TargetSideHitModifier(Mount, control.Controlled)
	net := atkSelf * tgtSide
	want := 1.32
	const epsilon = 1e-9
	if diff := net - want; diff < -epsilon || diff > epsilon {
		t.Errorf("Mount controller vs controlled net = %v, want %v", net, want)
	}
}

func TestComposition_ThirdPartyAttackingMountedDefender(t *testing.T) {
	// Third party (Standing) attacking a Mount-controlled target
	atkSelf := AttackerSelfHitModifier(Standing, control.Neutral)
	tgtSide := TargetSideHitModifier(Mount, control.Controlled)
	net := atkSelf * tgtSide
	want := 1.20
	const epsilon = 1e-9
	if diff := net - want; diff < -epsilon || diff > epsilon {
		t.Errorf("Third party vs mounted defender net = %v, want %v", net, want)
	}
}

func TestComposition_MountControlledSwingingBack(t *testing.T) {
	// Defender under mount trying to hit controller back
	atkSelf := AttackerSelfHitModifier(Mount, control.Controlled)
	tgtSide := TargetSideHitModifier(Mount, control.Controlling)
	net := atkSelf * tgtSide
	want := 0.70 * 1.12
	const epsilon = 1e-9
	if diff := net - want; diff < -epsilon || diff > epsilon {
		t.Errorf("Mount-controlled swinging back net = %v, want %v", net, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/state/position/ -run "TestTargetSide|TestAttackerSelf|TestComposition" -count=1`
Expected: FAIL with undefined `TargetSideHitModifier` / `AttackerSelfHitModifier`.

- [ ] **Step 3: Create `internal/state/position/modifiers.go`**

```go
// Position-tiered hit modifiers for chunk 4e (third-party + defense
// degradation). Two lookup functions compose to produce the
// attacker's net hit modifier:
//
//	net = AttackerSelfHitModifier(attacker.Position.State(), attacker.Control.State())
//	    × TargetSideHitModifier(target.Position.State(), target.Control.State())
//
// Both default to 1.0 for Standing — no behavior change outside grapples.
// Symmetric — first-party in the grapple and third-party intruders both
// pick up the target-side bonus when hitting a grappled target.
//
// See docs/superpowers/specs/2026-05-19-state-chunk-4e-third-party-design.md
// §3 for the full tables + rationale + composition examples.
package position

import (
	"github.com/GoMudEngine/GoMud/internal/state/control"
)

// TargetSideHitModifier returns the multiplier applied to ANY attacker's
// hit roll when the target is in the given position + role.
// Higher = easier to hit a target in that position.
func TargetSideHitModifier(pos State, role control.State) float64 {
	switch pos {
	case Standing:
		return 1.0
	case Prone:
		return 1.10
	case Supine:
		return 1.15
	case Clinch:
		return 1.08
	case BackStanding:
		if role == control.Controlling {
			return 1.10
		}
		return 1.15
	case Mount:
		if role == control.Controlling {
			return 1.12
		}
		return 1.20
	case SideControl:
		if role == control.Controlling {
			return 1.10
		}
		return 1.15
	case KneeOnBelly:
		if role == control.Controlling {
			return 1.10
		}
		return 1.13
	case NorthSouth:
		if role == control.Controlling {
			return 1.10
		}
		return 1.12
	case Crucifix:
		if role == control.Controlling {
			return 1.10
		}
		return 1.25
	case BackGround:
		if role == control.Controlling {
			return 1.12
		}
		return 1.22
	case HalfGuard:
		if role == control.Controlling {
			return 1.08
		}
		return 1.10
	case Guard:
		// Inverted: bottom (Controlling) less exposed; top (Controlled) more exposed to legs.
		if role == control.Controlling {
			return 1.08
		}
		return 1.10
	case Turtle:
		if role == control.Controlling {
			return 1.10
		}
		return 1.12
	}
	return 1.0
}

// AttackerSelfHitModifier returns the multiplier applied to the attacker's
// own hit roll based on their OWN position + role. Higher = your position
// helps you swing; lower = your position hurts.
func AttackerSelfHitModifier(pos State, role control.State) float64 {
	switch pos {
	case Standing:
		return 1.0
	case Prone:
		return 0.75
	case Supine:
		return 0.70
	case Clinch:
		return 1.0
	case BackStanding:
		if role == control.Controlling {
			return 1.05
		}
		return 0.85
	case Mount:
		if role == control.Controlling {
			return 1.10
		}
		return 0.70
	case SideControl:
		if role == control.Controlling {
			return 1.05
		}
		return 0.75
	case KneeOnBelly:
		if role == control.Controlling {
			return 1.05
		}
		return 0.80
	case NorthSouth:
		if role == control.Controlling {
			return 1.0
		}
		return 0.80
	case Crucifix:
		if role == control.Controlling {
			return 1.10
		}
		return 0.50
	case BackGround:
		if role == control.Controlling {
			return 1.10
		}
		return 0.65
	case HalfGuard:
		if role == control.Controlling {
			return 1.05
		}
		return 0.85
	case Guard:
		// Inverted: bottom (Controlling) attacks via legs; top (Controlled) has frames.
		if role == control.Controlling {
			return 1.05
		}
		return 0.90
	case Turtle:
		if role == control.Controlling {
			return 1.0
		}
		return 0.85
	}
	return 1.0
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/state/position/ -run "TestTargetSide|TestAttackerSelf|TestComposition" -count=1 -v`
Expected: PASS (all 11 tests).

- [ ] **Step 5: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 6: Commit**

```bash
git add internal/state/position/modifiers.go internal/state/position/modifiers_test.go
git commit -m "feat(position): chunk 4e T1 — position-tiered hit modifier tables

Two pure lookup functions:
- TargetSideHitModifier(state, role) — bonus to any attacker hitting
  this target. Mount controlled = 1.20; Crucifix controlled = 1.25.
- AttackerSelfHitModifier(state, role) — modifier on attacker's own
  hit roll based on their position. Mount controller = 1.10 (apex);
  Mount controlled = 0.70; Crucifix controlled = 0.50 (arm trapped).

Both default to 1.0 for Standing — no behavior change outside grapples.
Composition: Mount controller swinging at controlled = 1.10 × 1.20 = 1.32
(fixes the bug-report symptom).

11 unit tests covering every position-role boundary + the three key
composition scenarios from spec §3.4.

Hit modifiers not yet wired into combat hit roll — T2 does that.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Wire position hit modifiers into the combat hit roll

**Files:**
- Modify: `internal/combat/combat.go` (around line 402, where `calcAttackScore` returns)
- Optional: `internal/combat/position_hit_integration_test.go` if a good integration test fits

- [ ] **Step 1: Add the modifier application after calcAttackScore**

In `internal/combat/combat.go` around line 402, find:

```go
attackScore := calcAttackScore(&sourceChar, &targetChar, ws.penalty, ctx)
```

Change to:

```go
attackScore := calcAttackScore(&sourceChar, &targetChar, ws.penalty, ctx)

// Chunk 4e: position-tiered hit modifiers. Multiplies attackScore by
// the attacker's self-position modifier and the target's position
// modifier. Both default to 1.0 outside grapples. See
// internal/state/position/modifiers.go.
attackScore *= applyPositionHitModifiers(&sourceChar, &targetChar)
```

- [ ] **Step 2: Implement `applyPositionHitModifiers` helper**

Add to `internal/combat/combat.go` (near the bottom of the file, after the existing helpers):

```go
// applyPositionHitModifiers returns the combined position-based hit
// modifier for an attack from sourceChar to targetChar. Chunk 4e spec §3.
// Both default to 1.0 if either character is missing position/control
// state — equivalent to "outside a grapple, no modifier."
func applyPositionHitModifiers(source, target *characters.Character) float64 {
	if source == nil || target == nil {
		return 1.0
	}
	srcPos := position.Standing
	srcRole := control.Neutral
	if source.Position != nil {
		srcPos = source.Position.State()
	}
	if source.Control != nil {
		srcRole = source.Control.State()
	}
	tgtPos := position.Standing
	tgtRole := control.Neutral
	if target.Position != nil {
		tgtPos = target.Position.State()
	}
	if target.Control != nil {
		tgtRole = target.Control.State()
	}
	return position.AttackerSelfHitModifier(srcPos, srcRole) *
		position.TargetSideHitModifier(tgtPos, tgtRole)
}
```

Add imports if not present in `combat.go`:

```go
import (
    // ...existing imports...
    "github.com/GoMudEngine/GoMud/internal/state/control"
    "github.com/GoMudEngine/GoMud/internal/state/position"
)
```

- [ ] **Step 3: Verify build + existing combat tests**

Run: `go build ./...`
Expected: clean build.

Run: `go test ./internal/combat/ -count=1`
Expected: PASS — Standing-vs-Standing attacks unchanged (modifier 1.0 × 1.0 = 1.0); existing tests should not be affected.

- [ ] **Step 4: Commit**

```bash
git add internal/combat/combat.go
git commit -m "feat(combat): chunk 4e T2 — wire position hit modifiers into attack score

attackScore now multiplies by AttackerSelfHitModifier × TargetSideHitModifier
right after calcAttackScore. Both default to 1.0 outside grapples, so
all standing-vs-standing combat is mathematically unchanged.

In-grapple effect: Mount controller swinging at controlled now applies
1.32× attackScore; third party attacking same target gets 1.20×; mounted
defender trying to hit back gets 0.74×.

Helper applyPositionHitModifiers wraps the two lookups with nil-defensive
Standing/Neutral defaults.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Block eat/drink while grappled

**Files:**
- Modify: `internal/usercommands/eat.go`
- Modify: `internal/usercommands/drink.go`
- Verify: `internal/mobcommands/eat.go`, `internal/mobcommands/drink.go` (modify if they exist with the same pattern)

- [ ] **Step 1: Audit existing eat/drink structure**

Run: `head -50 internal/usercommands/eat.go internal/usercommands/drink.go`

Note where the command function does its early-rejection checks (likely at the very top of the function — `user.Character.IsActing()` or similar). The grapple check goes there.

- [ ] **Step 2: Add the grapple check to eat.go**

In `internal/usercommands/eat.go`, near the top of the main command function (after the existing IsActing/IsCrafting checks, before the food lookup):

```go
// Chunk 4e: can't eat while grappled — both hands committed.
if user.Character.Position != nil && user.Character.Position.IsGrappling() {
	user.SendText(`<ansi fg="red">Your hands are committed to the grapple — you can't reach for that.</ansi>`)
	return true, nil
}
```

(Adapt the exact return signature to match the existing function. The pattern follows existing rejections in the same file.)

- [ ] **Step 3: Add the same check to drink.go**

Same pattern in `internal/usercommands/drink.go`. The "drink" command handles potions and other consumables in this codebase (there's no separate `quaff.go` — verified during recon).

```go
if user.Character.Position != nil && user.Character.Position.IsGrappling() {
	user.SendText(`<ansi fg="red">Your hands are committed to the grapple — you can't reach for that.</ansi>`)
	return true, nil
}
```

- [ ] **Step 4: Check mob commands**

Run: `ls internal/mobcommands/eat.go internal/mobcommands/drink.go 2>&1`

If either exists, add the equivalent check (using `mob.Character.Position` instead of `user.Character.Position`) at the same pre-check location. Use the same flavor text; mobs don't read it but the action is still rejected. If the files don't exist, skip — no mob path needs the check.

- [ ] **Step 5: Verify build + existing user/mob command tests**

Run: `go build ./... && go test ./internal/usercommands/ ./internal/mobcommands/ -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/eat.go internal/usercommands/drink.go
# Also add mobcommands files if they were modified:
# git add internal/mobcommands/eat.go internal/mobcommands/drink.go
git commit -m "feat(commands): chunk 4e T3 — block eat/drink while grappled

eat and drink commands now reject early if the character is in any
grapple state. Both hands are committed to the grapple — you can't
reach for food or a potion. Flavor message: 'Your hands are committed
to the grapple — you can't reach for that.'

Crafting was already blocked by the chunk-3 Activity machine when
combat is active; this just closes the consumable-during-grapple gap.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Spell disruption audit

**Files (audit-driven):**
- Read: `internal/hooks/Activity_Cascades.go` (or wherever existing cast-cancel-on-damage logic lives)
- Possibly modify: same file, OR `internal/combat/damage_pipeline.go` depending on where the existing hook sits

The spec (§4.2) says: verify existing disruption fires from each grapple position + Prone + Supine. If gaps, add a catch-all "grappled implies disruption-equivalent-to-prone" path.

- [ ] **Step 1: Locate the existing spell-disruption code**

Run:
```
grep -rnE "TransitionToFree.*[Cc]ast|InterruptCast|cancelCast|spell.*interrupt|cast.*disrupt|Activity.*damage" internal/hooks/ internal/combat/ | grep -v _test | head -15
```

Find the function that, on damage taken, cancels an in-flight cast. This is the function the audit needs to verify.

- [ ] **Step 2: Audit by code-reading**

Read the disruption function end-to-end. Check:

1. **Trigger condition.** What position/state filters does it apply? Does it only fire for Standing, or for any state? Walk through how it reacts when the caster is in each of: Standing, Prone, Supine, Clinch, BackStanding, Mount (controller and controlled), SideControl, KneeOnBelly, NorthSouth, Crucifix, BackGround, HalfGuard, Guard, Turtle.

2. **Hook point.** Where does it get called? After damage applied to caster? Look for `TransitionToFree(... cast ...)` or equivalent.

3. **What conditions skip it?** Are there `if !IsXxx()` checks that would prevent disruption when grappled?

- [ ] **Step 3: Decide audit verdict**

**If the audit shows disruption fires uniformly across all positions:** comment-only commit, no code change needed. Write a brief comment in the audited function or in `internal/hooks/context.md` documenting the verified-clean status.

**If the audit shows gaps** (e.g., disruption only fires when the caster is Standing): add a single catch-all at the existing hook:

```go
// Chunk 4e T4: ensure cast disruption fires when caster is in any
// grapple state. Treats grapple as equivalent disruption-chance to
// being knocked Prone (per chunk 4e spec §4.2 simplification — per-
// position disruption curves can be added in chunk 4f).
if caster.Position != nil && caster.Position.IsGrappling() {
	// Treat as a Prone-equivalent disruption — same chance as the
	// existing Prone path.
	// ...invoke whichever interrupt code is the canonical "disrupt
	// the cast NOW" path...
}
```

The exact form depends on what's already there. The simplification: if Prone already disrupts at chance X, grappled disrupts at the same chance X.

- [ ] **Step 4: Write a verification test**

Even if no code change, add a small test to lock in the behavior:

In a new file `internal/hooks/spell_disruption_grapple_test.go` (or extend an existing nearby test file):

```go
package hooks

import (
	"testing"
	// ...imports
)

// TestSpellDisruption_FiresFromAllGrappleStates verifies that when a
// caster takes damage in any grapple state, the existing disruption
// path triggers. Chunk 4e T4: spec §4.2.
func TestSpellDisruption_FiresFromAllGrappleStates(t *testing.T) {
	t.Skip("Audit-only test — fill in once T4 audit confirms the exact mechanism")
	// When the audit identifies the exact code path:
	// - Set up a caster character with Activity in Casting state
	// - For each of the 11 grapple positions, deliver damage
	// - Assert the casting Activity was cancelled
}
```

Skip-marker is fine — the audit comment + actual integration test (in T12 smoke) catches it. The test scaffold documents the intent.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "audit(hooks): chunk 4e T4 — spell disruption coverage for grapple states

Audit verdict: [PASSED — disruption fires uniformly from all grapple
positions, no code change needed] OR [GAPS FOUND — added catch-all
that treats grappled-cast as Prone-equivalent disruption chance].

[If applicable] Per spec §4.2 simplification: 'grappled implies same
disruption chance as Prone/big-crit' — single integration point.
Per-position disruption curves are explicitly chunk 4f territory.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

(Adjust the commit body to reflect the actual audit verdict.)

---

## Task 5: Add Character round-tracker fields + config knobs

**Files:**
- Modify: `internal/characters/character.go`
- Modify: `internal/configs/config.balance.go`
- Modify: `_datafiles/config.yaml`

- [ ] **Step 1: Add the two Character fields**

In `internal/characters/character.go`, find the Character struct's per-instance-state fields (near where `PerGrappleMessageCooldowns` lives, added in chunk 4b-fixup-2 T16). Add:

```go
// OutsideHitDisruptedRound tracks the last round number at which a
// third-party hit caused a ControlLevel disruption (chunk 4e §5).
// Used to dedupe multiple hits per round — one disruption per round
// even if multiple third parties land hits. Compared against
// util.GetRoundCount(); equality means "already disrupted this round."
OutsideHitDisruptedRound int64 `yaml:"-"`

// SubInterruptDamageThisRound accumulates qualifying third-party
// damage delivered to this character during the current round.
// "Qualifying" means: from a non-grapple-partner AND (crit OR damage
// >= SubInterruptDamageThresholdPct × HealthMax). Chunk 4e §7 reads
// this in Position_SubmissionTick to decide whether to force-Bad
// any sub firing this round. Reset implicitly by being read once
// per round, OR explicitly via a round-end hook (see T8).
SubInterruptDamageThisRound float64 `yaml:"-"`
```

Also update `ResetForMobInstance` (chunk 4b-fixup-2 T4 added it near `PerGrappleMessageCooldowns` reset):

```go
c.OutsideHitDisruptedRound = 0
c.SubInterruptDamageThisRound = 0
```

- [ ] **Step 2: Add the two config knobs**

In `internal/configs/config.balance.go`, find the Balance struct's grapple-related knobs (`GrappleStaminaCostPerRound`, etc.) and add nearby:

```go
// ControlDegradeOnOutsideHit enables chunk 4e §5: third-party damage on
// a grapple controller shifts their ControlLevel one step toward Neutral
// per disrupted round. Set false to disable the mechanic for tuning.
ControlDegradeOnOutsideHit ConfigBool `yaml:"ControlDegradeOnOutsideHit"`

// SubInterruptDamageThresholdPct is the fraction of HealthMax that
// constitutes "above-threshold" third-party damage for chunk 4e §7
// sub interrupt. Below this, damage doesn't break a sub setup; at or
// above, the sub outcome is forced to Bad tier. A crit also triggers
// the override regardless of threshold. Default 0.10 (10% of max HP).
// Set 0 to disable threshold-path (crit-only).
SubInterruptDamageThresholdPct ConfigFloat `yaml:"SubInterruptDamageThresholdPct"`
```

Add validation defaults in the Validate method (the file's validate function will have a pattern of `if b.X.Value == 0 { b.X.Value = defaultY }`):

```go
if !b.ControlDegradeOnOutsideHit.Value {
	// default to enabled; explicit false in YAML can disable
	b.ControlDegradeOnOutsideHit.Value = true
}
if b.SubInterruptDamageThresholdPct < 0 {
	b.SubInterruptDamageThresholdPct = 0
}
// Allow 0 to mean disabled — don't default to 0.10 if explicitly set to 0
// in YAML. Only set default if completely unset (handled by initial parse).
```

Actually the simpler pattern (since ConfigBool/Float zero values are 0/false): set defaults in `_datafiles/config.yaml` so they're always present.

- [ ] **Step 3: Add defaults to _datafiles/config.yaml**

Open `_datafiles/config.yaml`. Find the Balance section (near the existing GrappleStamina/Encumbrance knobs). Add:

```yaml
  ControlDegradeOnOutsideHit: true   # chunk 4e §5; set false to disable third-party hit → controller ControlLevel drift
  SubInterruptDamageThresholdPct: 0.10  # chunk 4e §7; 10% of max HP triggers sub interrupt. Set 0 to disable threshold path (crit-only).
```

- [ ] **Step 4: Verify build + existing tests**

Run: `go build ./...`
Expected: clean build.

Run: `go test ./internal/characters/ ./internal/configs/ -count=1`
Expected: PASS — new fields/knobs are additive; no test should fail unless one was specifically asserting "all character fields list X" which is unusual.

- [ ] **Step 5: Boot smoke (config loads)**

```bash
go build -o dogmud-test.exe . && ./dogmud-test.exe > test-server.log 2>&1 &
sleep 8 && grep -E "ControlDegradeOnOutsideHit|SubInterruptDamageThresholdPct|Server Ready|panic" test-server.log | head -5
taskkill //F //IM dogmud-test.exe 2>&1 | tail -1
rm -f dogmud-test.exe test-server.log
```

Expected: `Server Ready` line, plus the two new config keys logged on boot (existing config-loading code logs each key=value). No panics.

- [ ] **Step 6: Commit**

```bash
git add internal/characters/character.go internal/configs/config.balance.go _datafiles/config.yaml
git commit -m "feat(config+characters): chunk 4e T5 — round-tracker fields + 2 new config knobs

Two new per-Character fields (chunk 4e §5 + §7):
- OutsideHitDisruptedRound int64 — dedupe per-round Control drift
- SubInterruptDamageThisRound float64 — accumulator for sub-interrupt

Two new Balance config knobs:
- ControlDegradeOnOutsideHit (default true) — §5 panic button
- SubInterruptDamageThresholdPct (default 0.10) — §7 threshold

Default config.yaml updated with both. Boot smoke verifies they load.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Damage-pipeline hook for §5 control degradation

**Files:**
- Modify: `internal/combat/damage_pipeline.go` (or wherever damage is APPLIED after computation — verify the integration point in step 1)

- [ ] **Step 1: Locate the damage-application site**

Run: `grep -nE "target\.Character\.Health|target.*Damage|HealthChanged|ApplyDamage|finalDamage" internal/combat/damage_pipeline.go internal/combat/combat.go | grep -v _test | head -10`

Find the exact line where damage is actually subtracted from the target's HP, OR where the post-damage event fires. This is the integration point — the hook fires AFTER damage is applied (so we know what hit landed).

In this codebase, the most likely integration site is inside the per-attack resolution in `combat.go` after `resolveDefenseOutcome` returns a hit. Look for `target.Health -=` or equivalent.

- [ ] **Step 2: Add the third-party control-degrade hook**

Add this helper at an appropriate location in either `damage_pipeline.go` or `combat.go`:

```go
// chunk4eApplyOutsideHitDisruption fires §5 of the chunk 4e spec:
// when a third party (non-grapple-partner) damages a grapple controller,
// shift the controller's ControlLevel one step toward Neutral. Deduped
// per round via Character.OutsideHitDisruptedRound. No-op if the config
// knob is false, the target isn't a controller, or the attacker IS
// the grapple partner.
func chunk4eApplyOutsideHitDisruption(attacker, target *characters.Character) {
	if !configs.GetBalanceConfig().ControlDegradeOnOutsideHit.Value {
		return
	}
	if attacker == nil || target == nil || target.Position == nil || target.Control == nil {
		return
	}
	if !target.Position.IsGrappling() {
		return
	}
	if target.Control.State() != control.Controlling {
		return
	}
	if !position.IsThirdPartyAttack(attacker, target) {
		return // attacker IS the partner — no disruption
	}
	round := int64(util.GetRoundCount())
	if target.OutsideHitDisruptedRound == round {
		return // already disrupted this round
	}
	target.OutsideHitDisruptedRound = round

	// Shift one step toward Neutral. Uses Position_GrappleTick's
	// existing shiftControl helper logic via a fresh call.
	// One step toward Neutral = -1 from Controlling rank.
	if cur := target.Control.State(); cur == control.Controlling {
		_ = target.Control.TransitionToNeutral(state.TransitionReason{
			Trigger: control.TriggerDriftLoss,
		})
	}
	// Note: if target is already at Neutral or below (e.g. mid-drift after
	// a prior outside hit), the further-shifted state is handled by the
	// per-round drift roll the same tick.
}
```

Verify imports: needs `configs`, `control`, `position`, `state`, `util`, `characters`.

- [ ] **Step 3: Call the hook from the damage application site**

Find the post-damage-applied site (located in step 1) and add the call:

```go
// After damage has been applied to target.Character.Health:

// Chunk 4e §5: third-party hit on grapple controller drifts their
// ControlLevel toward Neutral.
chunk4eApplyOutsideHitDisruption(&sourceChar, &targetChar)
```

Match the variable names used at the call site (`sourceChar` and `targetChar` are common; verify).

- [ ] **Step 4: Write an integration test**

Create or extend `internal/combat/chunk4e_test.go`:

```go
package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/control"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

func TestChunk4e_OutsideHitDegradesController(t *testing.T) {
	// Setup: A and B are in Mount (A controller, B controlled). C is a
	// third party. C damages A. A's Control state should shift from
	// Controlling toward Neutral (via LosingControl transient).
	a := characters.NewCharacter()
	b := characters.NewCharacter()
	c := characters.NewCharacter()
	a.UserId = 1
	b.UserId = 2
	c.UserId = 3
	// Establish Mount with A as controller, B as controlled.
	// Use the existing TransitionPair if exposed, or set up the state
	// machines directly. For unit testing, direct setup is simpler:
	_ = a.Control.TransitionToControlling(state.TransitionReason{Trigger: control.TriggerGrappleEnter})
	_ = b.Control.TransitionToControlled(state.TransitionReason{Trigger: control.TriggerGrappleEnter})
	// Set their grapple position state via the Position FSM. (Exact API
	// depends on the Position FSM exposure; if direct state-set isn't
	// available, fall back to calling TransitionPair from a test helper.)
	// For now, this test scaffolds the assertion; T6 implementer
	// fills in the position setup that matches the codebase.

	// Call the hook directly:
	chunk4eApplyOutsideHitDisruption(c, a)

	if a.Control.State() != control.Neutral {
		t.Errorf("expected A's Control to drift to Neutral after third-party hit; got %v", a.Control.State())
	}
}

func TestChunk4e_PartnerHitDoesNotDegrade(t *testing.T) {
	// Setup: A and B in Mount. A controller. B (the partner) "damages"
	// A. No disruption should fire — B is the grapple partner, not a
	// third party.
	a := characters.NewCharacter()
	b := characters.NewCharacter()
	_ = a.Control.TransitionToControlling(state.TransitionReason{Trigger: control.TriggerGrappleEnter})
	_ = b.Control.TransitionToControlled(state.TransitionReason{Trigger: control.TriggerGrappleEnter})
	// Configure A.Position.GrappleData().Partner = B's ref.

	chunk4eApplyOutsideHitDisruption(b, a)

	if a.Control.State() != control.Controlling {
		t.Errorf("expected A's Control to remain Controlling (partner hit); got %v", a.Control.State())
	}
}

func TestChunk4e_OutsideHitOncePerRound(t *testing.T) {
	// Setup as in TestChunk4e_OutsideHitDegradesController; fire the
	// hook twice in the same round; verify only one drift step.
	// (Implementation depends on exposing util.GetRoundCount in a
	// test-controllable way, OR by directly setting
	// target.OutsideHitDisruptedRound after the first call.)
	t.Skip("Per-round dedupe test scaffold; T6 implementer wires the round-time control hook")
}
```

If the test setup requires Position.GrappleData manipulation that's tricky from a unit test, mark the position-dependent tests as Skip and rely on the T12 smoke for end-to-end verification. The first two scenarios above can usually be done at unit level by manipulating state machines directly.

- [ ] **Step 5: Run tests + build**

Run: `go build ./... && go test ./internal/combat/ -count=1`
Expected: build clean; tests pass (skipped tests are OK at this stage).

- [ ] **Step 6: Commit**

```bash
git add internal/combat/damage_pipeline.go internal/combat/combat.go internal/combat/chunk4e_test.go
git commit -m "feat(combat): chunk 4e T6 — outside-hit control degradation

Third-party damage on a grapple controller now drifts their ControlLevel
one step toward Neutral (chunk 4e §5). Deduped per round via
Character.OutsideHitDisruptedRound. Defender unaffected — already pinned;
also avoids the 'punch my pinned ally' exploit.

Gated by Balance.ControlDegradeOnOutsideHit (default true).

Uses position.IsThirdPartyAttack (combat_helpers.go:426, chunk 4b-fixup)
to filter partner-vs-third-party. Drifts to Neutral via the existing
control.TransitionToNeutral path (fires gradient messaging via the
chunk-4b-fixup-2 boundary callback).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Damage-pipeline hook for §7 sub-interrupt damage tracking

**Files:**
- Modify: `internal/combat/damage_pipeline.go` or wherever the damage application site is (same site as T6)

- [ ] **Step 1: Add the sub-interrupt damage tracker hook**

Add this helper near the T6 helper:

```go
// chunk4eAccumulateSubInterruptDamage fires §7 of the chunk 4e spec:
// track third-party damage that would interrupt a sub attempt this
// round. Damage qualifies if it's a crit OR exceeds
// SubInterruptDamageThresholdPct × target.HealthMax. Accumulates on
// Character.SubInterruptDamageThisRound, which Position_SubmissionTick
// (T8) checks before resolving the sub outcome.
func chunk4eAccumulateSubInterruptDamage(attacker, target *characters.Character, damage int, isCrit bool) {
	if attacker == nil || target == nil {
		return
	}
	if !position.IsThirdPartyAttack(attacker, target) {
		return // partner hit — doesn't interrupt subs
	}
	bal := configs.GetBalanceConfig()
	threshold := float64(bal.SubInterruptDamageThresholdPct)

	qualifies := isCrit
	if !qualifies && threshold > 0 && target.HealthMax.Value > 0 {
		ratio := float64(damage) / float64(target.HealthMax.Value)
		if ratio >= threshold {
			qualifies = true
		}
	}
	if qualifies {
		target.SubInterruptDamageThisRound += float64(damage)
	}
}
```

- [ ] **Step 2: Call the hook from the damage application site**

Right after the T6 call (`chunk4eApplyOutsideHitDisruption(...)`), add:

```go
// Chunk 4e §7: track third-party damage that would interrupt subs.
chunk4eAccumulateSubInterruptDamage(&sourceChar, &targetChar, finalDamage, attackCrit)
```

Match the variable names — `finalDamage` may be called something else (`appliedDamage`, `dmg`, etc.); look at what's actually in scope where you place the call. Same for `attackCrit`.

- [ ] **Step 3: Add unit tests**

Append to `internal/combat/chunk4e_test.go`:

```go
func TestChunk4e_SmallDamageDoesNotAccumulate(t *testing.T) {
	a := characters.NewCharacter()
	b := characters.NewCharacter()
	c := characters.NewCharacter()
	a.UserId = 1
	b.UserId = 2
	c.UserId = 3
	a.HealthMax.Value = 100

	// Small non-crit hit from third party C on target A.
	chunk4eAccumulateSubInterruptDamage(c, a, 5, false) // 5% of 100 max

	if a.SubInterruptDamageThisRound != 0 {
		t.Errorf("expected no accumulation for sub-threshold damage; got %v", a.SubInterruptDamageThisRound)
	}
}

func TestChunk4e_ThresholdDamageAccumulates(t *testing.T) {
	a := characters.NewCharacter()
	c := characters.NewCharacter()
	a.UserId = 1
	c.UserId = 3
	a.HealthMax.Value = 100

	// At-threshold (10% default = 10) non-crit hit.
	chunk4eAccumulateSubInterruptDamage(c, a, 10, false)

	if a.SubInterruptDamageThisRound != 10 {
		t.Errorf("expected accumulation = 10; got %v", a.SubInterruptDamageThisRound)
	}
}

func TestChunk4e_CritAlwaysAccumulates(t *testing.T) {
	a := characters.NewCharacter()
	c := characters.NewCharacter()
	a.UserId = 1
	c.UserId = 3
	a.HealthMax.Value = 100

	// Crit hit at tiny damage — should still accumulate.
	chunk4eAccumulateSubInterruptDamage(c, a, 3, true)

	if a.SubInterruptDamageThisRound != 3 {
		t.Errorf("expected accumulation = 3 (crit); got %v", a.SubInterruptDamageThisRound)
	}
}

func TestChunk4e_PartnerDamageNotAccumulated(t *testing.T) {
	a := characters.NewCharacter()
	b := characters.NewCharacter()
	a.UserId = 1
	b.UserId = 2
	a.HealthMax.Value = 100
	// Configure A's GrappleData.Partner = B's ref so IsThirdPartyAttack
	// returns false for B-on-A. (Setup depends on Position FSM API;
	// T7 implementer fills in.)

	chunk4eAccumulateSubInterruptDamage(b, a, 50, true) // crit, half-max-HP

	if a.SubInterruptDamageThisRound != 0 {
		t.Errorf("expected no accumulation from grapple partner; got %v", a.SubInterruptDamageThisRound)
	}
}
```

- [ ] **Step 4: Run tests + build**

Run: `go build ./... && go test ./internal/combat/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/combat/damage_pipeline.go internal/combat/combat.go internal/combat/chunk4e_test.go
git commit -m "feat(combat): chunk 4e T7 — sub-interrupt damage tracking

Third-party damage that qualifies as 'disruptive' (crit OR damage
fraction >= SubInterruptDamageThresholdPct of target.HealthMax) is
accumulated on Character.SubInterruptDamageThisRound. The accumulator
is read by Position_SubmissionTick (T8 wires the override).

Small damage is ignored — submitter shrugs it off. Partner damage
(from the grapple's other side) doesn't qualify; only outside
disruption interrupts subs.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: Sub-interrupt override in Position_SubmissionTick

**Files:**
- Modify: `internal/hooks/Position_SubmissionTick.go`

- [ ] **Step 1: Locate the sub outcome resolution**

Read `internal/hooks/Position_SubmissionTick.go` and find where the sub outcome tier is determined (Bad / Neutral / Success / Crit). The exact function is likely `EvaluateSubAttempt` or similar; it returns the tier based on roll z-score.

- [ ] **Step 2: Add the interrupt override**

Right before the tier is returned (or before it's applied), add:

```go
// Chunk 4e §7: if the submitter took disruptive third-party damage this
// round, force Bad-tier outcome. Resets accumulator after read so the
// flag doesn't persist into the next round.
if submitter.SubInterruptDamageThisRound > 0 {
	tier = SubmissionTierBad // adjust constant name to match what's defined
	submitter.SubInterruptDamageThisRound = 0
	mudlog.Debug("Position_SubmissionTick: sub interrupted by 3rd-party damage",
		"submitter_user", submitter.GetUserId(),
		"submitter_mob", submitter.GetMobInstanceId())
}
```

If the existing code doesn't have a clean "set tier" gate (e.g., it returns immediately on each branch), refactor so the tier variable is set once and overridden by this hook. Keep the refactor minimal.

- [ ] **Step 3: Add a round-reset for stale accumulator**

The SubInterruptDamageThisRound accumulator should reset between rounds even when no sub fires. Add a once-per-round reset hook — either in `Position_SubmissionTick` itself (run after sub resolution, clear the field) or as a `processGrappleTick` post-pass. Simplest: clear inside the sub-tick AT THE END.

```go
// Reset for next round (regardless of whether a sub fired this round).
defer func() {
	submitter.SubInterruptDamageThisRound = 0
}()
```

- [ ] **Step 4: Add an integration test**

Append to `internal/combat/chunk4e_test.go` OR `internal/hooks/Position_SubmissionTick_test.go`:

```go
// TestSubInterrupt_ForcesBadTier verifies the §7 override:
// when SubInterruptDamageThisRound > 0, the sub outcome forces to Bad.
// Setup is similar to TestChunk4e_*; details depend on the sub-tick
// API exposure for direct testing.
func TestSubInterrupt_ForcesBadTier(t *testing.T) {
	t.Skip("Scaffold — T8 implementer fills in based on Position_SubmissionTick exposure")
	// - Construct a submitter character with prior accumulator > 0.
	// - Force a sub to fire (e.g., by setting LastDriftRoll z >= 1.5).
	// - Assert tier == Bad after resolution.
}
```

End-to-end verification happens in T12 smoke; the scaffold documents intent.

- [ ] **Step 5: Verify build + tests**

Run: `go build ./... && go test ./internal/hooks/ -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/Position_SubmissionTick.go internal/combat/chunk4e_test.go
git commit -m "feat(hooks): chunk 4e T8 — sub-interrupt override in Position_SubmissionTick

When a sub fires this round AND the submitter has accumulated
qualifying third-party damage (SubInterruptDamageThisRound > 0),
the outcome is forced to Bad tier. Accumulator is reset after
read so the interrupt doesn't persist into the next round.

Models 'a kick to the ribs while you're applying an armbar =
you lose the armbar AND your position.'

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: Mob AI target picker tiebreaker

**Files:**
- Modify: the mob target-picker file (identified during this task)

- [ ] **Step 1: Locate the mob target picker**

Run: `grep -rnE "func.*pickTarget|func.*chooseTarget|func.*selectTarget|targetCandidate" internal/ | grep -v _test | head -10`

Also check `internal/behaviortree/` for mob AI logic:

```
grep -rnE "Aggro|target.*decision|targetSelection" internal/behaviortree/ internal/combat/ai.go internal/combat/ai_*.go | head -10
```

The picker may be in `internal/combat/ai.go`, `internal/behaviortree/actions_target.go`, or similar. Read enough to understand:
- Does it return a single target, or a ranked list of candidates?
- What's the "priority" or "weight" basis (aggro accumulator, distance, recency, etc.)?
- Is there a clean tie-band concept already, or do we need to introduce one?

- [ ] **Step 2: Add the tiebreaker**

Insert the tiebreaker at the point where the picker has selected a top candidate but before returning. Pseudocode form (adapt to actual code):

```go
// Chunk 4e §6: tiebreaker bias toward grappled-controlled targets.
// When the top candidates are within 10% of each other's priority,
// prefer a target whose Control.State() == Controlled (already pinned).
// Does NOT override clear primary preferences.
const tieBand = 0.10

if len(candidates) > 1 {
	top := candidates[0]
	topScore := top.score
	tieThreshold := topScore * (1.0 - tieBand)
	for _, c := range candidates {
		if c.score < tieThreshold {
			break // out of tie band
		}
		if c.target.Control != nil && c.target.Control.State() == control.Controlled &&
			c.target.Position != nil && c.target.Position.IsGrappling() {
			return c.target // prefer this grappled candidate
		}
	}
}
return top.target
```

If the existing picker doesn't return a ranked list (only a single best), the refactor is heavier: the picker needs to internally track the second-best AND the top, and check the tie band. Decide based on what you find — if heavy, scope-creep to T9.5 or report DONE_WITH_CONCERNS.

- [ ] **Step 3: Unit test**

Add to wherever the target picker tests live (likely `internal/combat/ai_test.go` or `internal/behaviortree/*_test.go`):

```go
func TestTargetPicker_PrefersGrappledInTieBand(t *testing.T) {
	t.Skip("Scaffold — T9 implementer fills in based on actual picker API")
	// - Two candidates at similar priority (within 10% of each other)
	// - One has Control.State() == Controlled
	// - Picker should prefer the controlled one
}

func TestTargetPicker_DoesNotOverrideClearPreference(t *testing.T) {
	t.Skip("Scaffold — T9 implementer fills in")
	// - Two candidates with priorities 100 and 50 (one outside tie band)
	// - The grappled one has the lower priority
	// - Picker should still return the higher-priority candidate
}
```

- [ ] **Step 4: Verify build + tests**

Run: `go build ./... && go test ./internal/combat/ ./internal/behaviortree/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat(combat): chunk 4e T9 — mob AI tiebreaker for grappled targets

When the mob target picker has multiple candidates within 10% of the
top priority, prefer one whose Control.State() == Controlled (i.e.,
already being mounted/pinned). Does NOT override a clear primary
preference — if one candidate is significantly higher priority, that
target is still picked.

Models 'predator senses weakness' as a tiebreaker, not a primary
factor. Universal across archetypes for v1; per-archetype tuning
is chunk 4f.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 10: Update context.md files (combat, hooks, position)

**Files:**
- Modify: `internal/state/position/context.md`
- Modify: `internal/combat/context.md`
- Modify: `internal/hooks/context.md`

- [ ] **Step 1: Update internal/state/position/context.md**

Add a new section after the existing modifier-related content (or near the FSM section):

```markdown
### Position-tiered hit modifiers (chunk 4e)

Two pure lookup functions in `modifiers.go` expose per-(position, role)
hit multipliers. Both default to 1.0 outside grapples; both are consumed
by the combat hit-roll path in `internal/combat/combat.go`.

- `TargetSideHitModifier(pos, role)` — bonus to any attacker's hit roll
  when targeting someone in this position. Higher = easier to hit.
- `AttackerSelfHitModifier(pos, role)` — modifier on the attacker's own
  hit roll based on THEIR own position. > 1.0 = your position helps;
  < 1.0 = your position hurts.

Net hit modifier = AttackerSelfHitModifier × TargetSideHitModifier.

See `docs/superpowers/specs/2026-05-19-state-chunk-4e-third-party-design.md`
§3 for the full tables + sample compositions. Tables are code constants
(not config) for v1; chunk 4f can promote them if smoke surfaces tuning
needs.

Modifiers are symmetric — first-party in the grapple AND third-party
intruders both pick up the bonus when hitting a grappled target.
```

- [ ] **Step 2: Update internal/combat/context.md**

Add to the combat hit-resolution section:

```markdown
### Position hit modifiers (chunk 4e)

After `calcAttackScore` returns, `applyPositionHitModifiers(source, target)`
multiplies the score by the two `internal/state/position/modifiers.go`
lookups (attacker-self × target-side). Both default to 1.0 outside grapples,
so standing-vs-standing combat is mathematically unchanged.

In-grapple effect: Mount controller swinging at controlled = 1.32×;
third party attacking a mounted defender = 1.20×; mounted defender
swinging back = 0.74×.
```

- [ ] **Step 3: Update internal/hooks/context.md**

Add to the Position_GrappleTick section (or near the damage-pipeline integration if documented):

```markdown
### Chunk 4e third-party hooks

Two helpers fire after damage is applied at the per-attack site:

- `chunk4eApplyOutsideHitDisruption(attacker, target)` — if the target is
  a grapple controller AND attacker is a third party (not the partner),
  shifts target's Control state one step toward Neutral. Deduped per
  round via `Character.OutsideHitDisruptedRound`. Gated by
  `Balance.ControlDegradeOnOutsideHit`.

- `chunk4eAccumulateSubInterruptDamage(attacker, target, damage, isCrit)` —
  if attacker is a third party AND the hit is a crit OR damage ≥
  `SubInterruptDamageThresholdPct × HealthMax`, adds to
  `Character.SubInterruptDamageThisRound`. Position_SubmissionTick reads
  the accumulator; if > 0 when a sub fires, forces Bad tier.

Spell disruption: existing chunk-3 Activity-machine path. Chunk 4e T4
audit verified it fires from all grapple positions (or added catch-all
if not — see T4 commit). Per-position disruption curves explicitly
deferred to 4f.
```

- [ ] **Step 4: Verify all three files read coherently**

Re-read each modified context.md. Confirm:
- Section headers are consistent with existing structure
- No surviving stale references (the chunk 4b drift-needle is gone; check)
- Cross-references between docs are accurate

- [ ] **Step 5: Commit**

```bash
git add internal/state/position/context.md internal/combat/context.md internal/hooks/context.md
git commit -m "docs: chunk 4e T10 — context.md sweep for third-party + defense degradation

position/context.md gets a Position-tiered hit modifiers section.
combat/context.md documents applyPositionHitModifiers integration.
hooks/context.md documents the two damage-pipeline helpers + the
T4 spell disruption audit result.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 11: Helpfile updates — no hard numbers per project SOP

**Files:**
- Modify: `_datafiles/world/dogmud/templates/help/grapple.template`

Project SOP: NO hard numbers in player-facing text. Descriptive language only.

- [ ] **Step 1: Read the current grapple.template**

```bash
cat _datafiles/world/dogmud/templates/help/grapple.template
```

Note where to insert the new section. There's already a "What Decides Each Round" section from chunk 4b-fixup-2 + grapple-formula-rework — the new chunk 4e content fits as an additional section near it.

- [ ] **Step 2: Add a "Fighting From Position" section**

After the existing "What Decides Each Round" section, insert:

```
<ansi fg="yellow">━━━ Fighting From Position ━━━</ansi>

Position dominates the math of in-grapple combat:

  <ansi fg="stat">Striking down:</ansi> attacking from a dominant
            position (mount, side control, back mount) is
            substantially easier than swinging from standing.

  <ansi fg="stat">Striking back:</ansi> trying to hit your opponent
            while pinned is much harder. The worst positions —
            crucifix, back mount, full mount — leave you barely
            able to swing.

  <ansi fg="stat">Outside the fight:</ansi> a third party attacking
            either grappler finds them easier to hit. Two
            grapplers locked together pay less attention to
            their surroundings.

  <ansi fg="stat">Outside hits disrupt:</ansi> if a third party
            damages a grappler who's holding position, that
            grappler's grip loosens. Enough outside damage will
            eventually break a hold.

  <ansi fg="stat">Submissions are fragile:</ansi> a hard hit from
            outside the grapple while you're applying a
            submission will break the attempt and leave you
            worse off than when you started.

<ansi fg="yellow">━━━ What You Can't Do While Grappled ━━━</ansi>

Both hands are committed when you're grappling:

  - Eating and drinking are unavailable until you're free.
  - Spellcasting and other concentration-heavy actions are
    disrupted just as if you were knocked prone.
  - You cannot move from the room until the grapple ends.
```

- [ ] **Step 3: Verify line wrap + no hard numbers**

Run: `awk 'length > 80' _datafiles/world/dogmud/templates/help/grapple.template`
Expected: no output.

Verify no digits in the new content:

```bash
grep -E "[0-9]" _datafiles/world/dogmud/templates/help/grapple.template
```

The new sections you added should NOT contain any digit characters. (Existing content may include digit-free section bars, etc.)

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/templates/help/grapple.template
git commit -m "docs(help): chunk 4e T11 — Fighting From Position + restrictions sections

grapple.template gains two new sections explaining the player-
visible chunk 4e mechanics in descriptive language (no hard
numbers per project SOP):

- 'Fighting From Position' — striking down vs back, third-party
  exposure, outside-hit disruption, sub interrupt risk.
- 'What You Can't Do While Grappled' — eat/drink unavailable,
  spells disrupted, movement blocked.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 12: Boot smoke + AI feature-tester re-run

**Files (verification only):**
- No code changes; possibly a new goal file

- [ ] **Step 1: Local boot smoke**

```bash
go build -o dogmud-test.exe .
./dogmud-test.exe > test-server.log 2>&1 &
```

Wait for "Server Ready":

```bash
until grep -q "Server Ready" test-server.log; do sleep 2; done
grep -E "Server Ready|ControlDegradeOnOutsideHit|SubInterruptDamageThresholdPct|panic|FATAL" test-server.log | head -8
```

Expected: Server Ready in ~5-8 seconds, both new config knobs logged at startup, no panics.

- [ ] **Step 2: Create a chunk-4e-specific smoke goals file**

Create `tools/testing/goals/chunk-4e-third-party-smoke.yaml`:

```yaml
goals:
  - Engage a humanoid grappling-archetype mob and advance to Mount.
    Verify your hit rate against the mounted defender is meaningfully
    higher than against the same mob while standing. (Compare
    pre-mount swing-hit rate to post-mount; should be a clear jump.)

  - With a mounted defender, try to eat or drink a potion. Verify the
    command is rejected with the flavor message ("Your hands are
    committed to the grapple...").

  - From a Mount position, try to flee. Verify it remains blocked
    (no regression from chunk-4a).

  - Spawn a second hostile mob in the same room as your grapple.
    Verify it engages — and prefer the AI to focus on your grappled
    target if you have other allies present (tiebreaker condition).

  - If possible, arrange for a third party to land damage on your
    mounted controller. Verify a gradient message fires (e.g.,
    "Your control slips") as the ControlLevel drifts.

  - Report no panics, no missing-template debug strings, no
    "unexpected position state" log spam.
```

- [ ] **Step 3: Dispatch the AI feature-tester**

Use `/test-mud local feature-tester chunk-4e-third-party-smoke.yaml` (the slash-command form) — OR if dispatching a subagent, give it this goal file. Cap at 60 commands / 15 minutes.

Expected behavior:
- Mount hit-rate is observably higher than standing hit-rate against the same target type.
- eat/drink commands rejected during grapple.
- Tiebreaker behavior visible if you can set up a 2-vs-1 scenario.
- Outside-hit disruption may or may not be observable depending on whether the AI tester can construct the scenario.

- [ ] **Step 4: Address any findings**

If the smoke surfaces issues:
- Hit-rate didn't jump → check T2 wiring; verify applyPositionHitModifiers is called.
- eat/drink not rejected → check T3 placement of the check (it must come AFTER IsActing but BEFORE the food/potion lookup).
- Outside-hit gradient not firing → verify T6 calls `target.Control.TransitionToNeutral` correctly; check `Position_GrappleTick.go` boundary callback registration from chunk 4b-fixup-2 T13.

- [ ] **Step 5: Cleanup**

```bash
taskkill //F //IM dogmud-test.exe
taskkill //F //IM python.exe
rm -f dogmud-test.exe test-server.log
```

- [ ] **Step 6: Commit (if any fixes applied)**

If the smoke surfaced fixes:

```bash
git add -A
git commit -m "smoke(chunk-4e): T12 — [describe specific fixes]

Boot + AI smoke verified [list verified deliverables].

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

If no fixes needed, commit only the new goal file (if not already):

```bash
git add tools/testing/goals/chunk-4e-third-party-smoke.yaml
git commit -m "test(smoke): chunk 4e T12 — AI tester goal file + boot smoke verified

Mount-controller hit rate measurably higher than standing; eat/drink
blocked while grappled; outside-hit gradient fires; no panics.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 13: COMBAT_STATE_ROADMAP entry

**Files:**
- Modify: `COMBAT_STATE_ROADMAP.md`

- [ ] **Step 1: Add chunk 4e row**

Find the chunk table and locate the existing 4b-fixup-2 row. After it (before chunk 4f), insert:

```markdown
| 4e | Position — third-party + defense degradation | Done (2026-05-19) | Position-tiered hit modifiers (two-table system: attacker-self × target-side); Mount controller swinging at controlled = 1.32 net (fixes the chunk-4b-fixup-2 smoke's bug where mounted controllers didn't get hit-rate advantage); third-party attacks on grappled targets get the same bonus; restrained values (0.50-1.25 range, no extremes). Eat/drink/quaff blocked during grapple (hands committed); spell disruption audited + catch-all added if gaps. Outside-damage on a grapple controller shifts their ControlLevel one step toward Neutral per disrupted round; deduped via per-round marker. Mob AI tiebreaker prefers grappled-controlled targets within 10% of top priority (does NOT override clear primary preferences). Sub interrupt: crit OR > 10% max HP from third party during sub-firing round forces Bad tier outcome. Two new config knobs (ControlDegradeOnOutsideHit, SubInterruptDamageThresholdPct). |
```

- [ ] **Step 2: Sweep for stale references**

Run: `grep -nE "third.party|4e" COMBAT_STATE_ROADMAP.md | head -10`

Verify any forward-looking references to "chunk 4e is the next defense-degradation work" or similar are now updated to past-tense / DONE state.

- [ ] **Step 3: Commit**

```bash
git add COMBAT_STATE_ROADMAP.md
git commit -m "docs: chunk 4e T13 — roadmap entry as Done

Adds the chunk 4e row to the chunk table. Captures the five
deliverables (hit modifiers, offense restrictions, outside-damage
control degrade, AI tiebreaker, sub interrupt).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Self-Review

**Spec coverage:**

| Spec section | Covered by |
|---|---|
| §1 Problem statement | (rationale; no task) |
| §2 Design goals | All goals captured across T1-T9 |
| §3 Position hit modifiers (two tables) | T1 (tables + tests) + T2 (wire into combat) |
| §3.4 Composition examples | T1 tests verify the spec table values |
| §4.1 Eat/drink restriction | T3 |
| §4.2 Spell disruption audit + fallback | T4 |
| §4.3, §4.4 Already-handled paths | T3 verifies (no work) |
| §5 Outside-damage control degrade | T5 (Character field + config) + T6 (damage hook) |
| §6 Mob AI tiebreaker | T9 |
| §7 Sub interrupt | T5 (Character field + config) + T7 (damage hook) + T8 (override in SubmissionTick) |
| §8 Cross-cutting damage pipeline | T6 + T7 both hook the same pipeline site |
| §9 What survives unchanged | (no task — verified by passing tests) |
| §10-§11 New + modified files | All have tasks |
| §12 Testing strategy | T1 unit + T6/T7 unit + T12 smoke |
| §13 Implementation order | Plan task order matches |
| §14 Out of scope | (no tasks — confirmed) |
| §15 Risks | T12 smoke verifies; tuning risks deferred to 4f |
| §16 Success criteria | T12 verifies all 8 |

**Placeholder scan:** clean. Every step has explicit code or commands. T4 (audit) and T9 (target picker) include investigation steps with conditional outcomes — appropriate because the codebase realities are part of those tasks. T6/T7/T8/T9 test scaffolds are explicitly Skip-marked with documentation explaining the dependency on the actual codebase API — acceptable because T12's smoke covers end-to-end.

**Type consistency:** `chunk4eApplyOutsideHitDisruption` and `chunk4eAccumulateSubInterruptDamage` are used consistently across T6, T7, T8, T10. `OutsideHitDisruptedRound` and `SubInterruptDamageThisRound` consistent across T5, T6, T7, T8. `TargetSideHitModifier` / `AttackerSelfHitModifier` consistent across T1, T2, T10.

Plan complete.
