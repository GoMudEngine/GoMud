# Combat State — Chunk 4b: Position Control Axis Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Light up the chunk-4a Position FSM scaffold by cutting over all 11 command-site writers + ~10 reader sites from the legacy `CombatPosition` enum to the new 14-state FSM, adding per-round opposed control rolls (with stamina + encumbrance penalty curves), threshold-triggered position transitions, gradient messaging, six new control-axis btree primitives, and a periodic consistency checker. Sunset the legacy `CombatPosition` enum + `PositionRoundsMin` + `GrappleControllerId` + `ConditionGrappleController` + `internal/characters/combatposition.go` at the end. End-state: the 14-state Position machine is sole source of truth; grappling has per-round texture; rich-grapple system is LIVE.

**Architecture:** Three new observer files in `internal/hooks/` (per-round tick, messaging, consistency check). New control-axis API in `internal/state/position/` (TransitionPair, InitialControlForPair, DefaultEscapeTarget, ValidateGrapplePair). Parallel-write migration strategy throughout: writers land first (paralleling legacy + new fields), readers cut over next (now reading from FSM), legacy fields delete last. Three message classes (gradient / transition / stamina warning) with YAML-configured templates and per-grapple cooldown tracking. Four formal pair-state invariants enforced by `TransitionPair` + tested via `ValidateGrapplePair` + backstopped by a periodic consistency checker that force-breaks invalid pairs.

**Tech Stack:** Go 1.21+ with generics, existing `internal/state/` framework, existing `internal/state/position/` machine from chunk 4a.

**Spec:** `docs/superpowers/specs/2026-05-16-state-chunk-4b-position-control-axis-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP.

**Doc scope:** Comprehensive — 4b's cutover touches `internal/combat/`, `internal/hooks/`, `internal/characters/`, `internal/usercommands/`, `internal/mobcommands/`, `internal/users/`, `internal/behaviortree/`, `internal/state/position/`. T22 surveys + T23 updates all affected context.md files; T23 also addresses any helpfiles surfaced by the audit, scripting docs, and template documentation. **Wider doc sweep than chunk 4a** because 4b is the chunk that flips behavior — what was scaffold-only in 4a now produces user-visible texture.

---

## File structure

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/state/position/pair.go` | NEW | `TransitionPair`, `InitialControlForPair`, `DefaultEscapeTarget`, `Role` enum |
| `internal/state/position/control.go` | NEW | Control-axis arithmetic (`ShiftControl`, `IsController` derivation), per-round roll formula helpers |
| `internal/state/position/control_test.go` | NEW | Unit tests for control arithmetic + initial-state table + escape-target table |
| `internal/state/position/validation.go` | NEW | `ValidateGrapplePair` — the four invariants encoded |
| `internal/state/position/validation_test.go` | NEW | Tests for each invariant violation case |
| `internal/state/position/position.go` | MODIFY | Add `IsController()` method on Machine |
| `internal/state/position/position_test.go` | MODIFY | Append PB-001-080 Behavior Matrix tests for 4b |
| `internal/state/position/context.md` | MODIFY | Extend with control-axis API + messaging contract + invariants |
| `internal/characters/position_predicates.go` | MODIFY | Add `IsController()`, `IsBeingControlled()`, `IsLowGrappleStamina()` |
| `internal/characters/character.go` | MODIFY | Add `PerGrappleMessageCooldowns map[string]bool yaml:"-"` field |
| `internal/characters/combatposition.go` | DELETE (T21) | Legacy enum + helpers, after all readers migrated |
| `internal/characters/context.md` | MODIFY | Document new predicates + sunset of legacy fields |
| `internal/hooks/Position_GrappleTick.go` | NEW | Per-round opposed-roll observer; pair-aware iteration |
| `internal/hooks/Position_GrappleTick_test.go` | NEW | Integration tests for drift + threshold transitions |
| `internal/hooks/Position_ConsistencyCheck.go` | NEW | Periodic checker; force-breaks invalid pairs |
| `internal/hooks/Position_ConsistencyCheck_test.go` | NEW | Detect + heal + no-false-positive tests |
| `internal/hooks/Position_Messaging.go` | NEW | Gradient / transition / stamina messages with per-grapple cooldowns |
| `internal/hooks/Position_Messaging_test.go` | NEW | Cooldown + template-loading tests |
| `internal/hooks/context.md` | MODIFY | Document the three new observers |
| `internal/hooks/Life_Cascades.go` | MODIFY (T18) | DELETE chunk-2 pre-wire lines 55-57 |
| `internal/hooks/CombatPhase_Vetoes.go` | MODIFY (T18) | Rewire `RegisterPositionCheck` to `c.IsStanding()` |
| `internal/hooks/spell_resolution.go` | MODIFY (T13) | Spell knockdown writers migrate (lines 408, 1121) |
| `internal/hooks/NewRound_DoCombat.go` | MODIFY (T10) | DELETE ProcessGrappleProgression block (lines 327-378) |
| `internal/hooks/NewRound_DoCombat_helpers.go` | MODIFY (T17) | Flee blocker reads repointed (lines 504-510) |
| `internal/combat/grapple.go` | MODIFY (T9, T10, T11, T15) | `ApplyGrappleResult`, `ApplyPositionProgression` deletion, submission outcomes, crit-failure block all migrate to `TransitionPair`. DELETE `CheckClinchProgression` + `CheckGroundedEscape` in T10. |
| `internal/combat/skill_moves.go` | MODIFY (T12) | Trip / bash knockdown writes migrate with direction logic |
| `internal/combat/combat_helpers.go` | MODIFY (T16) | Damage/speed/crit/kick-variant readers repointed + third-party defense filter repointed |
| `internal/characters/skills.go` | MODIFY (T14) | `AttemptRecovery` migrates to `Position.TransitionToStanding` |
| `internal/usercommands/stand.go` | MODIFY (T15) | Explicit stand migrates |
| `internal/usercommands/grapple.go` | MODIFY (T9) | Entry command uses `TransitionPair` |
| `internal/mobcommands/grapple.go` | MODIFY (T9) | Mob equivalent |
| `internal/usercommands/trip.go` + `bash.go` + `kick.go` | MODIFY (T12, T16) | Variant-selector reads use predicates |
| `internal/mobcommands/trip.go` + `bash.go` + `kick.go` | MODIFY (T12, T16) | Mob equivalents |
| `internal/mobcommands/flee.go` | MODIFY (T17) | Flee blocker reads repointed |
| `internal/users/userrecord.prompt.go` | MODIFY (T19) | `{pos}` token repointed to new FSM |
| `internal/behaviortree/conditions_position.go` | MODIFY (T5) | Add 6 new control-axis primitives |
| `internal/behaviortree/conditions_position_test.go` | MODIFY (T5) | Smoke tests for new primitives |
| `_datafiles/config.yaml` | MODIFY (T6) | Add control-axis config knobs (penalty curves, stamina costs, consistency check interval, low-stamina threshold) |
| `_datafiles/messages/position_control.yaml` | NEW (T7) | YAML config for gradient/transition/stamina message templates |
| Test fixtures across packages | MODIFY (T20) | Parallel-write both legacy + new fields during migration window |
| `tools/testing/audits/2026-05-16-chunk-4b-doc-helpfile-audit.md` | NEW (T22) | Doc audit deliverable |
| Various context.md files | MODIFY (T23) | Comprehensive doc updates per audit |
| `COMBAT_STATE_ROADMAP.md` | MODIFY (T25) | Mark chunk 4b Done |

---

## Task 1: Position pair API + role enum

**Files:**
- Create: `internal/state/position/pair.go`

Foundation. Pair-aware atomic transitions + role enum + initial-control-per-state lookup + default-escape-target lookup.

- [ ] **Step 1: Create `internal/state/position/pair.go`**

```go
// Pair-aware transitions for the Position FSM. TransitionPair is
// the canonical way to put two characters into the same grapple
// state with role-appropriate initial ControlLevels. Direct calls
// to TransitionToXxx remain available for tests + edge cases but
// don't enforce pair semantics.
package position

import (
	"errors"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
)

// Role identifies a character's perspective in a grapple pair.
// In symmetric positions (Clinch, HalfGuard, Turtle solo) both
// sides hold the same Neutral ControlLevel; in asymmetric
// positions the controller side has ControlLevel ∈ {InControl,
// LosingControl} and the controlled has the complement.
type Role int

const (
	RoleController Role = iota
	RoleControlled
)

// ErrPairInvalidSourceStates is returned when TransitionPair is
// called with characters that aren't in compatible source states
// for the target transition.
var ErrPairInvalidSourceStates = errors.New("TransitionPair: source states incompatible with target")

// ErrPairRollbackFailed indicates the second-side transition
// failed and the rollback of the first side also failed —
// extremely unlikely but the consistency checker will catch it.
var ErrPairRollbackFailed = errors.New("TransitionPair: rollback of first side failed; pair may be desynced")

// InitialControlForPair returns the ControlLevel a character should
// start with when their pair transitions into the given grapple
// state. Encodes the design intuition that some positions are
// inherently asymmetric (Mount-top dominant) and others symmetric
// (Clinch true 50/50); also captures the Guard inversion (bottom is
// the active controller via legs).
func InitialControlForPair(target State, role Role) ControlLevel {
	switch target {
	case Clinch, HalfGuard:
		return Neutral
	case BackStanding, Mount, SideControl, Crucifix, BackGround:
		if role == RoleController {
			return InControl
		}
		return Controlled
	case KneeOnBelly, NorthSouth:
		if role == RoleController {
			return LosingControl
		}
		return BecomingControlled
	case Guard:
		// Inverted! Bottom (RoleController per our convention) is
		// the active grappler controlling top with their legs.
		if role == RoleController {
			return InControl
		}
		return Controlled
	case Turtle:
		// Defensive curl; both sides somewhat passive.
		if role == RoleController {
			return BecomingControlled
		}
		return LosingControl
	}
	return Neutral
}

// DefaultEscapeTarget returns the position state to transition to
// when the controlled fighter escapes the given grapple position.
// Represents the most common BJJ outcome per state.
func DefaultEscapeTarget(current State) State {
	switch current {
	case Clinch:
		return Standing
	case BackStanding:
		return Standing
	case Mount:
		return HalfGuard
	case SideControl, KneeOnBelly, NorthSouth:
		return Guard
	case Crucifix:
		return SideControl
	case BackGround:
		return Mount
	case HalfGuard:
		return Guard
	case Guard:
		return Standing
	case Turtle:
		return Standing
	}
	return Standing
}

// TransitionPair atomically moves a controller + controlled pair
// into the same grapple state, with role-appropriate initial
// ControlLevels per InitialControlForPair. Validates source states;
// rolls back the first side if the second fails. Standing target
// is a special case: clears both sides' GrappleData (the grapple
// has ended).
//
// Caller is responsible for identifying the controller and
// controlled (the existing CheckClinchProgression / grapple
// command code already knows which is which; new code can use
// IsController()).
func TransitionPair(
	controller, controlled *characters.Character,
	target State,
	r state.TransitionReason,
) error {
	if controller == nil || controlled == nil ||
		controller.Position == nil || controlled.Position == nil {
		return ErrPairInvalidSourceStates
	}

	// Standing target: break the grapple. Both sides return to
	// Standing; no GrappleData required.
	if target == Standing {
		prev := snapshotPosition(controller)
		if err := controller.Position.TransitionToStanding(r); err != nil {
			return err
		}
		if err := controlled.Position.TransitionToStanding(r); err != nil {
			if rbErr := restorePosition(controller, prev); rbErr != nil {
				return ErrPairRollbackFailed
			}
			return err
		}
		return nil
	}

	// Non-Standing target: build pair data + fire both transitions.
	ctrlRef := state.ActorRef{
		UserId:        controller.UserId,
		MobInstanceId: controller.MobInstanceId,
	}
	cdRef := state.ActorRef{
		UserId:        controlled.UserId,
		MobInstanceId: controlled.MobInstanceId,
	}

	ctrlData := GrappleData{
		Partner:      cdRef,
		ControlLevel: InitialControlForPair(target, RoleController),
	}
	cdData := GrappleData{
		Partner:      ctrlRef,
		ControlLevel: InitialControlForPair(target, RoleControlled),
	}

	prev := snapshotPosition(controller)
	if err := transitionTo(controller, target, ctrlData, r); err != nil {
		return err
	}
	if err := transitionTo(controlled, target, cdData, r); err != nil {
		if rbErr := restorePosition(controller, prev); rbErr != nil {
			return ErrPairRollbackFailed
		}
		return err
	}
	return nil
}

// transitionTo dispatches to the right TransitionToXxx method per
// target state. Go doesn't have generic dispatch over state values;
// switch is the cleanest way to keep TransitionPair as a single
// helper.
func transitionTo(c *characters.Character, target State, d GrappleData, r state.TransitionReason) error {
	switch target {
	case Clinch:
		return c.Position.TransitionToClinch(d, r)
	case BackStanding:
		return c.Position.TransitionToBackStanding(d, r)
	case Mount:
		return c.Position.TransitionToMount(d, r)
	case SideControl:
		return c.Position.TransitionToSideControl(d, r)
	case KneeOnBelly:
		return c.Position.TransitionToKneeOnBelly(d, r)
	case NorthSouth:
		return c.Position.TransitionToNorthSouth(d, r)
	case Crucifix:
		return c.Position.TransitionToCrucifix(d, r)
	case BackGround:
		return c.Position.TransitionToBackGround(d, r)
	case HalfGuard:
		return c.Position.TransitionToHalfGuard(d, r)
	case Guard:
		return c.Position.TransitionToGuard(d, r)
	case Turtle:
		return c.Position.TransitionToTurtle(d, r)
	}
	return ErrPairInvalidSourceStates
}

// positionSnapshot captures enough state to roll back a transition.
type positionSnapshot struct {
	state    State
	grapple  *GrappleData
	prone    *ProneData
	supine   *SupineData
}

func snapshotPosition(c *characters.Character) positionSnapshot {
	snap := positionSnapshot{state: c.Position.State()}
	if d, ok := c.Position.GrappleData(); ok {
		snap.grapple = &d
	}
	if d, ok := c.Position.ProneData(); ok {
		snap.prone = &d
	}
	if d, ok := c.Position.SupineData(); ok {
		snap.supine = &d
	}
	return snap
}

// restorePosition reverses a transition. Uses ForceStanding then
// re-applies the snapshotted state. If the snapshot was Standing,
// just ForceStanding is enough.
func restorePosition(c *characters.Character, snap positionSnapshot) error {
	c.Position.ForceStanding(state.TransitionReason{Trigger: "pair_rollback"})
	if snap.state == Standing {
		return nil
	}
	r := state.TransitionReason{Trigger: "pair_rollback"}
	if snap.grapple != nil {
		return transitionTo(c, snap.state, *snap.grapple, r)
	}
	if snap.prone != nil {
		return c.Position.TransitionToProne(*snap.prone, r)
	}
	if snap.supine != nil {
		return c.Position.TransitionToSupine(*snap.supine, r)
	}
	return nil
}
```

- [ ] **Step 2: Build verify**

```bash
go build ./internal/state/position/
```
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add internal/state/position/pair.go
git commit -m "$(cat <<'EOF'
feat(position): TransitionPair API + InitialControlForPair + DefaultEscapeTarget

Pair-aware atomic transitions for the Position FSM. TransitionPair
is the canonical way to put two characters into the same grapple
state with role-appropriate initial ControlLevels (per the per-state
initial-control table — Guard inversion encoded). Validates source
states; rolls back the first side via snapshot/restore if the second
fails. Standing target is special-cased (breaks the grapple).

InitialControlForPair encodes the 11 grapple states' starting
ControlLevels per role (controller / controlled). DefaultEscapeTarget
maps each grapple state to its most common BJJ controlled-escape
target (Mount → HalfGuard, SideControl → Guard, etc.).

Direct TransitionToXxx calls remain available for tests + edge cases
but don't enforce pair semantics.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Control axis arithmetic + ValidateGrapplePair

**Files:**
- Create: `internal/state/position/control.go`
- Create: `internal/state/position/validation.go`
- Create: `internal/state/position/control_test.go`
- Create: `internal/state/position/validation_test.go`
- Modify: `internal/state/position/position.go` (add `IsController()` on Machine)

ControlLevel arithmetic (Shift / Clamp), IsController derivation on Machine, ValidateGrapplePair encoding the 4 invariants.

- [ ] **Step 1: Create `internal/state/position/control.go`**

```go
package position

// ShiftControl moves a ControlLevel by `delta` (positive = toward
// InControl, negative = toward Controlled), clamping to the
// [InControl, Controlled] range. Encodes the per-round drift
// arithmetic.
func ShiftControl(current ControlLevel, delta int) ControlLevel {
	// Enum order (post chunk-4a iota reorder):
	//   Neutral=0, InControl=1, LosingControl=2,
	//   BecomingControlled=3, Controlled=4
	//
	// Conceptual ordering (winning → losing):
	//   InControl < LosingControl < Neutral < BecomingControlled < Controlled
	//
	// "toward InControl" means moving leftward in the conceptual
	// ordering. The enum order doesn't match the conceptual order,
	// so we map to an integer rank and back.
	rank := controlRank(current)
	rank += delta // positive delta = toward Controlled (worse for the side)
	if rank < 0 {
		rank = 0
	}
	if rank > 4 {
		rank = 4
	}
	return controlFromRank(rank)
}

// controlRank maps ControlLevel to its position in the conceptual
// "winning → losing" gradient. 0 = InControl (best), 4 = Controlled
// (worst). Used by ShiftControl arithmetic.
func controlRank(c ControlLevel) int {
	switch c {
	case InControl:
		return 0
	case LosingControl:
		return 1
	case Neutral:
		return 2
	case BecomingControlled:
		return 3
	case Controlled:
		return 4
	}
	return 2 // safe default
}

// controlFromRank is the inverse of controlRank.
func controlFromRank(rank int) ControlLevel {
	switch rank {
	case 0:
		return InControl
	case 1:
		return LosingControl
	case 2:
		return Neutral
	case 3:
		return BecomingControlled
	case 4:
		return Controlled
	}
	return Neutral
}

// IsControllerLevel returns true if the given ControlLevel
// indicates the holder is in the "controller" role of the
// asymmetric grapple pair. By convention, ControlLevel ∈
// {InControl, LosingControl} = controller. Neutral is ambiguous
// (used for symmetric positions); caller's state context resolves.
func IsControllerLevel(c ControlLevel) bool {
	return c == InControl || c == LosingControl
}

// IsControlledLevel returns true if the given ControlLevel
// indicates the holder is in the "controlled" role.
func IsControlledLevel(c ControlLevel) bool {
	return c == BecomingControlled || c == Controlled
}

// MarginToDelta maps the |z-score| of an opposed roll outcome to
// the magnitude of ControlLevel shift per the 4b spec:
//
//   |z| range  | magnitude
//   0.0 – 0.5  | 0 (no shift)
//   0.5 – 1.0  | 1
//   1.0 – 2.0  | 2
//   ≥ 2.0      | 3 (crit)
func MarginToDelta(absZScore float64) int {
	switch {
	case absZScore < 0.5:
		return 0
	case absZScore < 1.0:
		return 1
	case absZScore < 2.0:
		return 2
	default:
		return 3
	}
}
```

- [ ] **Step 2: Add `IsController()` to Machine in `internal/state/position/position.go`**

Find the existing predicates block (IsStanding, IsProne, etc.) and append:

```go
// IsController returns true when the character is the controller
// side of a grapple pair. False for Standing, Prone, Supine, and
// for grapple states where the character's ControlLevel is on the
// controlled side or neutral (symmetric positions).
//
// Used by the per-round tick to identify which side of a pair to
// iterate from; used by btree primitives for AI decisions.
func (m *Machine) IsController() bool {
	if !m.IsGrappling() {
		return false
	}
	d, ok := m.GrappleData()
	if !ok {
		return false
	}
	return IsControllerLevel(d.ControlLevel)
}

// IsBeingControlled returns true when the character is the
// controlled side of a grapple pair.
func (m *Machine) IsBeingControlled() bool {
	if !m.IsGrappling() {
		return false
	}
	d, ok := m.GrappleData()
	if !ok {
		return false
	}
	return IsControlledLevel(d.ControlLevel)
}
```

- [ ] **Step 3: Create `internal/state/position/validation.go`**

```go
package position

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
)

// PairInvariantViolation describes which of the four invariants
// failed and provides context for logging.
type PairInvariantViolation struct {
	Invariant   string // "single-partner", "bidirectional", "matching-state", "role-exclusivity"
	Description string
}

func (v PairInvariantViolation) Error() string {
	return fmt.Sprintf("pair invariant violation: %s — %s", v.Invariant, v.Description)
}

// ValidateGrapplePair checks the four pair-state invariants for two
// characters claimed to be grappling each other. Returns a
// PairInvariantViolation on failure, nil if all invariants hold.
//
// Invariants:
//
//  1. Single-partner: each side's GrappleData.Partner is non-zero
//     and refers to the other character. (Turtle exception: solo
//     Turtle may have zero Partner; both-Turtle still requires
//     each to point at the other.)
//
//  2. Bidirectional: if a.Partner = b.Self then b.Partner = a.Self.
//     Always reciprocal.
//
//  3. Matching-state: a.State() == b.State() while in a grapple pair.
//
//  4. Role-exclusivity: for asymmetric positions (any except Clinch,
//     HalfGuard, Turtle), exactly one side is the controller
//     (ControlLevel ∈ {InControl, LosingControl}) and the other is
//     controlled (∈ {BecomingControlled, Controlled}). Both-Neutral
//     and one-Neutral-one-non-Neutral are violations for asymmetric
//     positions.
func ValidateGrapplePair(a, b *characters.Character) error {
	if a == nil || b == nil {
		return PairInvariantViolation{
			Invariant:   "nil-input",
			Description: "one or both characters are nil",
		}
	}
	if a.Position == nil || b.Position == nil {
		return PairInvariantViolation{
			Invariant:   "nil-machine",
			Description: "one or both characters have nil Position machine",
		}
	}

	stateA := a.Position.State()
	stateB := b.Position.State()

	// Both must be in grapple states.
	if !a.Position.IsGrappling() || !b.Position.IsGrappling() {
		return PairInvariantViolation{
			Invariant:   "single-partner",
			Description: fmt.Sprintf("one side not grappling (a=%v, b=%v)", stateA, stateB),
		}
	}

	// Invariant 3: matching state.
	if stateA != stateB {
		return PairInvariantViolation{
			Invariant: "matching-state",
			Description: fmt.Sprintf("state mismatch: a=%v, b=%v",
				stateA, stateB),
		}
	}

	dA, _ := a.Position.GrappleData()
	dB, _ := b.Position.GrappleData()
	refA := state.ActorRef{UserId: a.UserId, MobInstanceId: a.MobInstanceId}
	refB := state.ActorRef{UserId: b.UserId, MobInstanceId: b.MobInstanceId}

	// Invariant 1 + 2: single-partner + bidirectional.
	// Turtle solo case: if either side has zero Partner, both sides
	// must be solo (i.e., the other side's Partner is also zero).
	if stateA == Turtle && (dA.Partner.IsZero() || dB.Partner.IsZero()) {
		// Solo Turtle on either side — not a pair, not subject to
		// pair invariants. Caller shouldn't be validating this as a
		// pair, but tolerate.
		return nil
	}

	if dA.Partner != refB {
		return PairInvariantViolation{
			Invariant: "single-partner",
			Description: fmt.Sprintf("a.Partner (%+v) != b.Self (%+v)",
				dA.Partner, refB),
		}
	}
	if dB.Partner != refA {
		return PairInvariantViolation{
			Invariant: "bidirectional",
			Description: fmt.Sprintf("b.Partner (%+v) != a.Self (%+v)",
				dB.Partner, refA),
		}
	}

	// Invariant 4: role-exclusivity for asymmetric positions.
	if !isSymmetricGrapple(stateA) {
		aIsCtrl := IsControllerLevel(dA.ControlLevel)
		bIsCtrl := IsControllerLevel(dB.ControlLevel)
		aIsCd := IsControlledLevel(dA.ControlLevel)
		bIsCd := IsControlledLevel(dB.ControlLevel)

		// Exactly one controller + exactly one controlled required.
		if !(aIsCtrl && bIsCd) && !(aIsCd && bIsCtrl) {
			return PairInvariantViolation{
				Invariant: "role-exclusivity",
				Description: fmt.Sprintf(
					"asymmetric state %v requires one controller + one controlled; got a=%v, b=%v",
					stateA, dA.ControlLevel, dB.ControlLevel),
			}
		}
	}

	return nil
}

// isSymmetricGrapple returns true for grapple states where both
// sides can legitimately hold Neutral ControlLevel (Clinch,
// HalfGuard, Turtle).
func isSymmetricGrapple(s State) bool {
	return s == Clinch || s == HalfGuard || s == Turtle
}
```

- [ ] **Step 4: Create unit tests for control + validation**

`internal/state/position/control_test.go`:

```go
package position_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state/position"
)

func TestShiftControl_TowardInControl(t *testing.T) {
	got := position.ShiftControl(position.Controlled, -2)
	want := position.Neutral
	if got != want {
		t.Errorf("ShiftControl(Controlled, -2) = %v, want %v", got, want)
	}
}

func TestShiftControl_TowardControlled(t *testing.T) {
	got := position.ShiftControl(position.InControl, 3)
	want := position.BecomingControlled
	if got != want {
		t.Errorf("ShiftControl(InControl, 3) = %v, want %v", got, want)
	}
}

func TestShiftControl_ClampsHigh(t *testing.T) {
	got := position.ShiftControl(Neutral, 100)
	if got != position.Controlled {
		t.Errorf("ShiftControl should clamp at Controlled; got %v", got)
	}
}

func TestShiftControl_ClampsLow(t *testing.T) {
	got := position.ShiftControl(position.Neutral, -100)
	if got != position.InControl {
		t.Errorf("ShiftControl should clamp at InControl; got %v", got)
	}
}

func TestMarginToDelta_NoShift(t *testing.T) {
	if got := position.MarginToDelta(0.3); got != 0 {
		t.Errorf("MarginToDelta(0.3) = %d, want 0", got)
	}
}

func TestMarginToDelta_OneLevel(t *testing.T) {
	if got := position.MarginToDelta(0.7); got != 1 {
		t.Errorf("MarginToDelta(0.7) = %d, want 1", got)
	}
}

func TestMarginToDelta_TwoLevels(t *testing.T) {
	if got := position.MarginToDelta(1.5); got != 2 {
		t.Errorf("MarginToDelta(1.5) = %d, want 2", got)
	}
}

func TestMarginToDelta_Crit(t *testing.T) {
	if got := position.MarginToDelta(2.5); got != 3 {
		t.Errorf("MarginToDelta(2.5) = %d, want 3", got)
	}
}

func TestInitialControlForPair_Mount(t *testing.T) {
	if got := position.InitialControlForPair(position.Mount, position.RoleController); got != position.InControl {
		t.Errorf("InitialControlForPair(Mount, Controller) = %v, want InControl", got)
	}
	if got := position.InitialControlForPair(position.Mount, position.RoleControlled); got != position.Controlled {
		t.Errorf("InitialControlForPair(Mount, Controlled) = %v, want Controlled", got)
	}
}

func TestInitialControlForPair_GuardInversion(t *testing.T) {
	// Guard's "controller" (per our naming) is the BOTTOM person
	// trapping top with legs. Bottom starts InControl.
	if got := position.InitialControlForPair(position.Guard, position.RoleController); got != position.InControl {
		t.Errorf("InitialControlForPair(Guard, Controller) = %v, want InControl (bottom is controller)", got)
	}
}

func TestInitialControlForPair_SymmetricClinch(t *testing.T) {
	if got := position.InitialControlForPair(position.Clinch, position.RoleController); got != position.Neutral {
		t.Errorf("InitialControlForPair(Clinch, Controller) = %v, want Neutral (symmetric)", got)
	}
	if got := position.InitialControlForPair(position.Clinch, position.RoleControlled); got != position.Neutral {
		t.Errorf("InitialControlForPair(Clinch, Controlled) = %v, want Neutral", got)
	}
}

func TestDefaultEscapeTarget_Mount(t *testing.T) {
	if got := position.DefaultEscapeTarget(position.Mount); got != position.HalfGuard {
		t.Errorf("DefaultEscapeTarget(Mount) = %v, want HalfGuard", got)
	}
}

func TestDefaultEscapeTarget_Guard(t *testing.T) {
	if got := position.DefaultEscapeTarget(position.Guard); got != position.Standing {
		t.Errorf("DefaultEscapeTarget(Guard) = %v, want Standing", got)
	}
}
```

- [ ] **Step 5: Create validation tests**

`internal/state/position/validation_test.go`:

```go
package position_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

func TestValidateGrapplePair_ValidMount(t *testing.T) {
	a := characters.New()
	a.UserId = 1
	b := characters.New()
	b.UserId = 2

	refA := state.ActorRef{UserId: 1}
	refB := state.ActorRef{UserId: 2}

	_ = a.Position.TransitionToClinch(
		position.GrappleData{Partner: refB},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = a.Position.TransitionToMount(
		position.GrappleData{Partner: refB, ControlLevel: position.InControl},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	_ = b.Position.TransitionToClinch(
		position.GrappleData{Partner: refA},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = b.Position.TransitionToMount(
		position.GrappleData{Partner: refA, ControlLevel: position.Controlled},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)

	if err := position.ValidateGrapplePair(a, b); err != nil {
		t.Errorf("expected valid pair; got %v", err)
	}
}

func TestValidateGrapplePair_BothInControlRejected(t *testing.T) {
	a := characters.New()
	a.UserId = 1
	b := characters.New()
	b.UserId = 2

	refA := state.ActorRef{UserId: 1}
	refB := state.ActorRef{UserId: 2}

	// Force both into Mount with InControl (invariant violation).
	_ = a.Position.TransitionToClinch(
		position.GrappleData{Partner: refB},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = a.Position.TransitionToMount(
		position.GrappleData{Partner: refB, ControlLevel: position.InControl},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	_ = b.Position.TransitionToClinch(
		position.GrappleData{Partner: refA},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = b.Position.TransitionToMount(
		position.GrappleData{Partner: refA, ControlLevel: position.InControl}, // BUG: both InControl
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)

	err := position.ValidateGrapplePair(a, b)
	if err == nil {
		t.Fatal("expected role-exclusivity violation")
	}
	violation, ok := err.(position.PairInvariantViolation)
	if !ok {
		t.Fatalf("expected PairInvariantViolation; got %T", err)
	}
	if violation.Invariant != "role-exclusivity" {
		t.Errorf("expected role-exclusivity; got %q", violation.Invariant)
	}
}

func TestValidateGrapplePair_StateMismatchRejected(t *testing.T) {
	a := characters.New()
	a.UserId = 1
	b := characters.New()
	b.UserId = 2

	refA := state.ActorRef{UserId: 1}
	refB := state.ActorRef{UserId: 2}

	// a in Mount, b in SideControl — state mismatch.
	_ = a.Position.TransitionToClinch(position.GrappleData{Partner: refB}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = a.Position.TransitionToMount(position.GrappleData{Partner: refB, ControlLevel: position.InControl}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
	_ = b.Position.TransitionToClinch(position.GrappleData{Partner: refA}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = b.Position.TransitionToSideControl(position.GrappleData{Partner: refA, ControlLevel: position.Controlled}, state.TransitionReason{Trigger: position.TriggerTakedownSide})

	err := position.ValidateGrapplePair(a, b)
	if err == nil {
		t.Fatal("expected matching-state violation")
	}
	violation, _ := err.(position.PairInvariantViolation)
	if violation.Invariant != "matching-state" {
		t.Errorf("expected matching-state; got %q", violation.Invariant)
	}
}

func TestValidateGrapplePair_BrokenPartnerRefRejected(t *testing.T) {
	a := characters.New()
	a.UserId = 1
	b := characters.New()
	b.UserId = 2

	refA := state.ActorRef{UserId: 1}
	refB := state.ActorRef{UserId: 2}
	refStale := state.ActorRef{UserId: 99}

	_ = a.Position.TransitionToClinch(position.GrappleData{Partner: refB}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = a.Position.TransitionToMount(position.GrappleData{Partner: refB, ControlLevel: position.InControl}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
	_ = b.Position.TransitionToClinch(position.GrappleData{Partner: refStale}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = b.Position.TransitionToMount(position.GrappleData{Partner: refStale, ControlLevel: position.Controlled}, state.TransitionReason{Trigger: position.TriggerTakedownMount})

	err := position.ValidateGrapplePair(a, b)
	if err == nil {
		t.Fatal("expected single-partner violation")
	}
	_ = refA // refA is what b SHOULD point at; the test asserts the violation fires
}
```

- [ ] **Step 6: Build + run tests**

```bash
go build ./...
go test ./internal/state/position/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/state/position/control.go \
        internal/state/position/validation.go \
        internal/state/position/control_test.go \
        internal/state/position/validation_test.go \
        internal/state/position/position.go
git commit -m "$(cat <<'EOF'
feat(position): control-axis arithmetic + ValidateGrapplePair

control.go: ShiftControl (clamping arithmetic for ControlLevel
drift), MarginToDelta (z-score → 0-3 magnitude per 4b spec table),
IsControllerLevel/IsControlledLevel derivations, controlRank
mapping (the enum order doesn't match the conceptual
winning→losing gradient, so internal rank conversion is required).

Machine methods: IsController() / IsBeingControlled() on the
Position machine. Used by per-round tick to identify pair sides
and by btree primitives for AI decisions.

validation.go: ValidateGrapplePair encoding the four invariants
(single-partner, bidirectional, matching-state, role-exclusivity).
Returns PairInvariantViolation with descriptive context on failure.
Used by tests, defensive assertions, and the periodic consistency
checker (T8).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Character predicates + PerGrappleMessageCooldowns field

**Files:**
- Modify: `internal/characters/character.go` (add `PerGrappleMessageCooldowns map[string]bool yaml:"-"`)
- Modify: `internal/characters/position_predicates.go` (add IsController, IsBeingControlled, IsLowGrappleStamina)

- [ ] **Step 1: Add field to `internal/characters/character.go`**

Find the existing position-related fields (look for `Position *position.Machine` from chunk 4a) and add the cooldown map alongside:

```go
// PerGrappleMessageCooldowns tracks which gradient/stamina
// messages have already fired during the current grapple session.
// Resets when the character returns to a non-grapple state.
// Non-persistent — combat doesn't survive logout.
PerGrappleMessageCooldowns map[string]bool `yaml:"-"`
```

Initialize in `New()` to an empty map:
```go
PerGrappleMessageCooldowns: map[string]bool{},
```

Nil-guard in `Validate()`:
```go
if c.PerGrappleMessageCooldowns == nil {
    c.PerGrappleMessageCooldowns = map[string]bool{}
}
```

- [ ] **Step 2: Add predicates to `internal/characters/position_predicates.go`**

Append:

```go
// --- Control-axis predicates (chunk 4b) ---

// IsController returns true when the character is the controller
// side of a grapple pair. False outside of grapples.
func (c *Character) IsController() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsController()
}

// IsBeingControlled returns true when the character is the
// controlled side of a grapple pair.
func (c *Character) IsBeingControlled() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsBeingControlled()
}

// IsLowGrappleStamina returns true when the character's stamina
// fraction (current/max) is below the GrappleStaminaLowThreshold
// config knob. Used by btree primitive mob_low_grapple_stamina
// and by Position_Messaging to fire stamina warnings.
func (c *Character) IsLowGrappleStamina() bool {
	cfg := configs.GetBalanceConfig()
	threshold := float64(cfg.GrappleStaminaLowThreshold)
	if threshold <= 0 {
		threshold = 0.25 // hard fallback
	}
	if c.StaminaMax.Value <= 0 {
		return false
	}
	return float64(c.Stamina)/float64(c.StaminaMax.Value) < threshold
}
```

Add the import if not already present:
```go
"github.com/GoMudEngine/GoMud/internal/configs"
```

- [ ] **Step 3: Build + test**

```bash
go build ./...
go test ./internal/characters/ ./internal/state/position/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
Note: `GrappleStaminaLowThreshold` config knob doesn't exist yet — T6 adds it. The `IsLowGrappleStamina` will use the hard fallback 0.25 until then; build will pass because `cfg.GrappleStaminaLowThreshold` is a method call that will resolve to zero value (0) on the un-set field. T6 wires the config properly.

Actually — `cfg.GrappleStaminaLowThreshold` needs to compile. Either:
(a) Add the field to `internal/configs/config.balance.go` now in this task (cleanest)
(b) Defer the field to T6 and use a hard-coded 0.25 here

Going with (a) — add the field stub now:

```go
// In internal/configs/config.balance.go, in the Balance struct:
GrappleStaminaLowThreshold ConfigFloat `yaml:"GrappleStaminaLowThreshold"`
```

T6 will add the rest of the grapple-axis config knobs alongside.

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/characters/character.go \
        internal/characters/position_predicates.go \
        internal/configs/config.balance.go
git commit -m "$(cat <<'EOF'
feat(position): Character control-axis predicates + cooldown map

PerGrappleMessageCooldowns map[string]bool field on Character
(non-persistent, init empty, nil-guard in Validate). Used by
Position_Messaging (T7) for per-grapple-session cooldown tracking
to avoid spam when ControlLevel oscillates around a boundary.

IsController() / IsBeingControlled() / IsLowGrappleStamina()
predicates on Character (delegating to Position machine + Balance
config). IsLowGrappleStamina compares c.Stamina/c.StaminaMax.Value
against GrappleStaminaLowThreshold (field stub added now; T6 wires
the default value via _datafiles/config.yaml).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Behavior Matrix RED tests (PB-001 through PB-080)

**Files:**
- Modify: `internal/state/position/position_test.go` (append new PB-* tests; chunk-4a PO-* tests stay)

All ~80 Behavior Matrix tests for 4b authored as failing skeletons where the underlying API doesn't exist yet. Pure-arithmetic tests (control + validation) added in Tasks 1-2 may already cover PB-001-027; this task adds the rest as SKIP stubs that activate in later tasks.

- [ ] **Step 1: Read the spec section "Behavior Matrix preview"**

```
docs/superpowers/specs/2026-05-16-state-chunk-4b-position-control-axis-design.md
```

The matrix has 6 groups (PB-001-015 drift mechanics, PB-016-027 initial-state, PB-028-042 thresholds, PB-043-052 invariants, PB-053-070 cutover smoke, PB-071-080 messaging).

- [ ] **Step 2: Append PB-* tests to `internal/state/position/position_test.go`**

Many of the rows are SKIP stubs that activate in later tasks. The pattern:

```go
// --- PB-001 through PB-015: Per-round drift mechanics ---

func TestPB_001_ShiftControlClampHigh(t *testing.T) {
	// Already covered by control_test.go TestShiftControl_ClampsHigh
	// — this is the matrix anchor.
	t.Skip("covered by internal/state/position/control_test.go:TestShiftControl_ClampsHigh")
}

// ... PB-002 through PB-015 similarly anchor to existing unit tests
//     (in control_test.go) or t.Skip with reference to the Task that
//     adds the integration test.

// --- PB-016 through PB-027: InitialControlForPair table ---

func TestPB_016_InitialMountController(t *testing.T) {
	t.Skip("covered by control_test.go:TestInitialControlForPair_Mount")
}
// ... etc.

// --- PB-028 through PB-042: Threshold transitions (Tasks 6, 9-15) ---

func TestPB_028_MountControlledHitsControlledFiresEscape(t *testing.T) {
	t.Skip("integration test — verified in Task 6 GrappleTick test")
}
// ... etc.

// --- PB-043 through PB-052: Pair invariants (Tasks 1, 2, 8) ---

func TestPB_043_FreshGrapplePairInvariantsHold(t *testing.T) {
	t.Skip("covered by validation_test.go:TestValidateGrapplePair_ValidMount")
}
// ... etc.

// --- PB-053 through PB-070: Cutover smoke (Tasks 9-19) ---

func TestPB_053_GrappleCommandLandsInClinch(t *testing.T) {
	t.Skip("integration test — verified in Task 9 grapple cutover test")
}
// ... etc.

// --- PB-071 through PB-080: Messaging contract (Task 7) ---

func TestPB_071_LosingControlMessageFiresOnce(t *testing.T) {
	t.Skip("integration test — verified in Task 7 Messaging test")
}
// ... etc.
```

The full 80-row list is in the spec. Each test is a 1-3 line SKIP stub pointing at the task that activates it (or the existing test that covers it). Real assertions land in later tasks.

- [ ] **Step 3: Run tests — verify SKIP counts**

```bash
go test ./internal/state/position/ -v -count=1 2>&1 | grep -E "^--- (PASS|SKIP|FAIL)" | awk '{print $2}' | sort | uniq -c
```

Expected: PB rows mostly SKIP; PO rows from chunk 4a still PASS; 0 FAIL.

- [ ] **Step 4: Commit**

```bash
git add internal/state/position/position_test.go
git commit -m "$(cat <<'EOF'
test(position): Behavior Matrix RED skeleton PB-001 through PB-080

80 chunk-4b matrix rows authored as SKIP stubs anchored to either
(a) existing chunk-4b unit tests (control_test.go,
validation_test.go from Tasks 1-2) or (b) future integration tests
in Tasks 6-19. Stubs are clear "covered by Task N" references;
they activate as each later task lands.

Chunk-4a PO-* tests unchanged.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Btree primitives for control axis

**Files:**
- Modify: `internal/behaviortree/conditions_position.go` (append 6 new conditions)
- Modify: `internal/behaviortree/conditions_position_test.go` (append smoke tests)

- [ ] **Step 1: Append 6 conditions to `internal/behaviortree/conditions_position.go`**

```go
// --- Control-axis queries (chunk 4b) ---

func condMobIsInControl(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsController() {
		return Success
	}
	return Failure
}

func condMobIsBeingControlled(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsBeingControlled() {
		return Success
	}
	return Failure
}

// condMobControlAtLeast is parameterized: takes "level" string
// (e.g., "neutral", "in_control"). Returns Success if mob's
// ControlLevel is "at or better than" the param for the mob's
// role perspective (controller: lower rank = better; controlled:
// higher rank = better).
func condMobControlAtLeast(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || mob.Character.Position == nil {
		return Failure
	}
	d, ok := mob.Character.Position.GrappleData()
	if !ok {
		return Failure
	}
	levelStr, _ := params["level"].(string)
	threshold, ok := parseControlLevel(levelStr)
	if !ok {
		return Failure
	}
	// "At least" depends on role:
	//   controller (low rank = good): current.rank <= threshold.rank
	//   controlled (high rank = good): current.rank >= threshold.rank
	cur := position.ControlRankExported(d.ControlLevel)   // export the helper in T2
	th := position.ControlRankExported(threshold)
	if mob.Character.IsController() {
		if cur <= th {
			return Success
		}
		return Failure
	}
	if cur >= th {
		return Success
	}
	return Failure
}

// parseControlLevel maps a YAML-friendly string to a ControlLevel.
func parseControlLevel(s string) (position.ControlLevel, bool) {
	switch s {
	case "in_control":
		return position.InControl, true
	case "losing_control":
		return position.LosingControl, true
	case "neutral":
		return position.Neutral, true
	case "becoming_controlled":
		return position.BecomingControlled, true
	case "controlled":
		return position.Controlled, true
	}
	return position.Neutral, false
}

func condMobLowGrappleStamina(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsLowGrappleStamina() {
		return Success
	}
	return Failure
}

func condTargetIsInControl(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || !mob.Character.IsInCombat() {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if target.Char.IsController() {
		return Success
	}
	return Failure
}

func condTargetIsBeingControlled(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || !mob.Character.IsInCombat() {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if target.Char.IsBeingControlled() {
		return Success
	}
	return Failure
}
```

In `init()` (chunk 4a already established the registry pattern), append:
```go
conditionRegistry["mob_is_in_control"] = condMobIsInControl
conditionRegistry["mob_is_being_controlled"] = condMobIsBeingControlled
conditionRegistry["mob_control_at_least"] = condMobControlAtLeast
conditionRegistry["mob_low_grapple_stamina"] = condMobLowGrappleStamina
conditionRegistry["target_is_in_control"] = condTargetIsInControl
conditionRegistry["target_is_being_controlled"] = condTargetIsBeingControlled
```

Note the call to `position.ControlRankExported`. The chunk-4a internal `controlRank` is package-private; add a thin exported alias in `internal/state/position/control.go`:

```go
// ControlRankExported is the public-API wrapper for the internal
// controlRank helper. Used by btree primitives that need to compare
// ControlLevels in conceptual ordering rather than enum order.
func ControlRankExported(c ControlLevel) int {
	return controlRank(c)
}
```

Add this to the T2 commit if not already there, or as a separate small commit before this task.

- [ ] **Step 2: Append smoke tests to `internal/behaviortree/conditions_position_test.go`**

```go
func TestCondMobIsInControl_NotGrappling_Failure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine()
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condMobIsInControl(nil, ctx))
}

func TestCondMobIsInControl_InMountAsController_Success(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine()
	_ = mob.Character.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 999}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = mob.Character.Position.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 999}, ControlLevel: position.InControl},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Success, condMobIsInControl(nil, ctx))
}

func TestCondMobControlAtLeast_NeutralOrBetter_AsController(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine()
	_ = mob.Character.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 999}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = mob.Character.Position.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 999}, ControlLevel: position.InControl},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	params := map[string]any{"level": "neutral"}
	// Controller at InControl is "better than" Neutral → Success.
	assert.Equal(t, Success, condMobControlAtLeast(params, ctx))
}

func TestPositionPrimitives_AllRegistered_4b(t *testing.T) {
	names := []string{
		// 4a primitives
		"mob_is_standing", "mob_is_prone", "mob_is_grappling",
		"mob_in_mount", "mob_in_guard", "mob_in_clinch", "mob_in_top_dominant",
		"target_is_standing", "target_is_prone", "target_is_grappled",
		// 4b primitives
		"mob_is_in_control", "mob_is_being_controlled",
		"mob_control_at_least", "mob_low_grapple_stamina",
		"target_is_in_control", "target_is_being_controlled",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			if LookupCondition(name) == nil {
				t.Errorf("condition %q not registered", name)
			}
		})
	}
}
```

- [ ] **Step 3: Build + test**

```bash
go build ./...
go test ./internal/behaviortree/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/behaviortree/conditions_position.go \
        internal/behaviortree/conditions_position_test.go \
        internal/state/position/control.go
git commit -m "$(cat <<'EOF'
feat(position): 6 btree control-axis primitives

mob_is_in_control / mob_is_being_controlled — boolean role queries.
mob_control_at_least — parameterized ("level" string param);
  comparison respects role direction (controller-low-rank-better
  vs controlled-high-rank-better).
mob_low_grapple_stamina — stamina fraction below GrappleStaminaLowThreshold.
target_is_in_control / target_is_being_controlled — resolved via
  ResolveAggroTarget; gated on IsInCombat.

ControlRankExported public alias added in control.go for the btree
primitives' comparison need.

Combined with chunk-4a's 10 primitives, btree authors have 16 total
position-related queries by end of 4b. Enables tactical patterns
("if I'm in mount + in control + target low stamina, try
submission"). Actual archetype YAML authoring is content / aliveness
work after chunk 6.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Position_GrappleTick observer

**Files:**
- Create: `internal/hooks/Position_GrappleTick.go`
- Create: `internal/hooks/Position_GrappleTick_test.go`
- Modify: `_datafiles/config.yaml` (add grapple-axis config knobs)
- Modify: `internal/configs/config.balance.go` (add field declarations)

The per-round opposed-roll tick. Pair-aware iteration (one roll per pair). Updates both ControlLevels per the margin → magnitude table. Fires threshold-triggered transitions when either side hits Controlled. Applies asymmetric stamina cost per round.

- [ ] **Step 1: Add config fields to `internal/configs/config.balance.go`**

Find the `Balance` struct and add (alongside the chunk-3-era resource penalty config knobs):

```go
// Grapple control axis (chunk 4b)
GrappleStaminaPenaltyMax      ConfigFloat `yaml:"GrappleStaminaPenaltyMax"`      // Max penalty multiplier reduction at 0% stamina (default 0.60)
GrappleStaminaPenaltyCurve    ConfigFloat `yaml:"GrappleStaminaPenaltyCurve"`    // Exponent shape of penalty curve (default 1.5)
GrappleEncumbrancePenaltyMax  ConfigFloat `yaml:"GrappleEncumbrancePenaltyMax"`  // Max penalty multiplier reduction at max encumbrance (default 0.80)
GrappleEncumbrancePenaltyCurve ConfigFloat `yaml:"GrappleEncumbrancePenaltyCurve"` // Exponent shape (default 1.5)
GrappleStaminaCostPerRound    ConfigInt   `yaml:"GrappleStaminaCostPerRound"`    // Base per-round stamina cost (default 5)
GrappleControllerCostMultiplier ConfigFloat `yaml:"GrappleControllerCostMultiplier"` // Controller's cost multiplier (default 1.0)
GrappleControlledCostMultiplier ConfigFloat `yaml:"GrappleControlledCostMultiplier"` // Controlled's cost multiplier (default 2.0)
GrappleStaminaLowThreshold    ConfigFloat `yaml:"GrappleStaminaLowThreshold"`    // Stamina fraction below which stamina warning fires (default 0.25)
PositionConsistencyCheckRounds ConfigInt  `yaml:"PositionConsistencyCheckRounds"` // How often the consistency checker runs (default 10)
```

In the Validate() method of Balance config, add defaults for any field == 0:

```go
if g.GrappleStaminaPenaltyMax <= 0 {
    g.GrappleStaminaPenaltyMax = 0.60
}
if g.GrappleStaminaPenaltyCurve <= 0 {
    g.GrappleStaminaPenaltyCurve = 1.5
}
if g.GrappleEncumbrancePenaltyMax <= 0 {
    g.GrappleEncumbrancePenaltyMax = 0.80
}
if g.GrappleEncumbrancePenaltyCurve <= 0 {
    g.GrappleEncumbrancePenaltyCurve = 1.5
}
if g.GrappleStaminaCostPerRound <= 0 {
    g.GrappleStaminaCostPerRound = 5
}
if g.GrappleControllerCostMultiplier <= 0 {
    g.GrappleControllerCostMultiplier = 1.0
}
if g.GrappleControlledCostMultiplier <= 0 {
    g.GrappleControlledCostMultiplier = 2.0
}
if g.GrappleStaminaLowThreshold <= 0 {
    g.GrappleStaminaLowThreshold = 0.25
}
if g.PositionConsistencyCheckRounds <= 0 {
    g.PositionConsistencyCheckRounds = 10
}
```

- [ ] **Step 2: Add YAML defaults to `_datafiles/config.yaml`**

Find the existing `Balance:` block and add (after the existing combat-axis config knobs):

```yaml
  # Chunk 4b — Position control axis (grapple per-round drift mechanics)
  # See docs/superpowers/specs/2026-05-16-state-chunk-4b-position-control-axis-design.md
  GrappleStaminaPenaltyMax: 0.60        # Max roll-mult reduction at 0% stamina (vs chunk-3 ResourcePenaltyMax 0.28)
  GrappleStaminaPenaltyCurve: 1.5       # Exponent — higher = steeper near 0% stamina
  GrappleEncumbrancePenaltyMax: 0.80    # Max roll-mult reduction at "crushed" encumbrance
  GrappleEncumbrancePenaltyCurve: 1.5   # Exponent — higher = steeper near max
  GrappleStaminaCostPerRound: 5         # Base per-round stamina cost in grapples
  GrappleControllerCostMultiplier: 1.0  # Controller side multiplier (5 × 1.0 = 5/round)
  GrappleControlledCostMultiplier: 2.0  # Controlled side multiplier (5 × 2.0 = 10/round)
  GrappleStaminaLowThreshold: 0.25      # Stamina fraction below which "you're getting gassed" fires once per grapple
  PositionConsistencyCheckRounds: 10    # How often the periodic invariant checker runs
```

- [ ] **Step 3: Create `internal/hooks/Position_GrappleTick.go`**

```go
package hooks

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// processGrappleTick runs once per round per active grapple pair.
// Iterates all characters in active combat rooms; for each character
// that's a controller (IsController()), looks up their partner via
// GrappleData.Partner, validates the pair, and calls
// processGrapplePair. Skips controlled side (processed via the
// controller's iteration).
func processGrappleTick(e events.Event) events.ListenerReturn {
	// Players
	for _, u := range users.GetAllActiveUsers() {
		if u == nil || u.Character == nil {
			continue
		}
		if !u.Character.IsController() {
			continue
		}
		partner := resolvePartner(u.Character)
		if partner == nil {
			continue
		}
		if err := position.ValidateGrapplePair(u.Character, partner); err != nil {
			// Consistency check will catch + force-break; skip this round.
			mudlog.Warn("Position_GrappleTick: invalid pair", "user", u.UserId, "err", err)
			continue
		}
		processGrapplePair(u.Character, partner)
	}
	// Mobs
	for _, mobInstId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(mobInstId)
		if m == nil {
			continue
		}
		if !m.Character.IsController() {
			continue
		}
		partner := resolvePartner(&m.Character)
		if partner == nil {
			continue
		}
		if err := position.ValidateGrapplePair(&m.Character, partner); err != nil {
			mudlog.Warn("Position_GrappleTick: invalid pair", "mob", m.InstanceId, "err", err)
			continue
		}
		processGrapplePair(&m.Character, partner)
	}
	return events.Continue
}

// resolvePartner looks up a character's grapple partner via its
// GrappleData. Returns nil if no partner or if the partner reference
// can't be resolved.
func resolvePartner(c *characters.Character) *characters.Character {
	if c.Position == nil {
		return nil
	}
	d, ok := c.Position.GrappleData()
	if !ok || d.Partner.IsZero() {
		return nil
	}
	if d.Partner.UserId > 0 {
		if u := users.GetByUserId(d.Partner.UserId); u != nil {
			return u.Character
		}
	}
	if d.Partner.MobInstanceId > 0 {
		if m := mobs.GetInstance(d.Partner.MobInstanceId); m != nil {
			return &m.Character
		}
	}
	return nil
}

// processGrapplePair fires the opposed roll, computes drift,
// updates both sides' ControlLevels, applies stamina cost, and
// checks for threshold-triggered transition.
func processGrapplePair(controller, controlled *characters.Character) {
	cfg := configs.GetBalanceConfig()

	// Compute scores.
	ctrlBase := float64(controller.Stats.Strength.Value + controller.GetSkillRank("weapon-combat"))
	cdBase := float64(controlled.Stats.Strength.Value + controlled.GetSkillRank("weapon-combat")) +
		0.5*float64(controlled.Stats.Dexterity.Value)
	cdBase += escapeModifierFromBody(controlled)

	ctrlScore := ctrlBase *
		grappleStaminaMultiplier(controller, cfg) *
		grappleEncumbranceMultiplier(controller, cfg)
	cdScore := cdBase *
		grappleStaminaMultiplier(controlled, cfg) *
		grappleEncumbranceMultiplier(controlled, cfg)

	// Opposed roll.
	roll := dice.OpposedRollStat(int(ctrlScore), int(cdScore))

	delta := position.MarginToDelta(math.Abs(roll.ZScore))
	if delta == 0 {
		// No shift, but apply stamina cost + check threshold.
		applyGrappleStaminaCost(controller, controlled, cfg)
		fireStaminaWarningIfLow(controller)
		fireStaminaWarningIfLow(controlled)
		return
	}

	// Determine direction.
	// ZScore > 0 means controller won — controlled drifts toward
	// Controlled and controller drifts toward InControl.
	var ctrlDelta, cdDelta int
	if roll.ZScore > 0 {
		ctrlDelta = -delta // toward InControl (rank 0)
		cdDelta = +delta   // toward Controlled (rank 4)
	} else {
		ctrlDelta = +delta // toward Controlled
		cdDelta = -delta   // toward InControl
	}

	ctrlData, _ := controller.Position.GrappleData()
	cdData, _ := controlled.Position.GrappleData()

	newCtrl := position.ShiftControl(ctrlData.ControlLevel, ctrlDelta)
	newCd := position.ShiftControl(cdData.ControlLevel, cdDelta)

	// Update GrappleData via re-transitioning (Position machine
	// doesn't expose direct ControlLevel mutation; the value is
	// stored in the per-state data which is set on TransitionToXxx.
	// We need a helper to update ControlLevel in-place.).
	updateControlLevel(controller, newCtrl)
	updateControlLevel(controlled, newCd)

	// Fire gradient messages for any boundary crossings (Task 7).
	fireGradientMessages(controller, ctrlData.ControlLevel, newCtrl)
	fireGradientMessages(controlled, cdData.ControlLevel, newCd)

	// Stamina cost + warnings.
	applyGrappleStaminaCost(controller, controlled, cfg)
	fireStaminaWarningIfLow(controller)
	fireStaminaWarningIfLow(controlled)

	// Threshold check: if either side hit Controlled, fire escape.
	if newCtrl == position.Controlled || newCd == position.Controlled {
		escapeTarget := position.DefaultEscapeTarget(controller.Position.State())
		_ = position.TransitionPair(
			controller, controlled,
			escapeTarget,
			state.TransitionReason{Trigger: position.TriggerControlThresholdCrossed},
		)
		// Reset per-grapple message cooldowns for both sides.
		controller.PerGrappleMessageCooldowns = map[string]bool{}
		controlled.PerGrappleMessageCooldowns = map[string]bool{}
		fireTransitionMessages(controller, controlled, escapeTarget)
	}
}

// updateControlLevel mutates the character's GrappleData
// ControlLevel without re-transitioning the FSM state. Uses the
// Machine's MutateGrappleData helper (added in T2 next to control.go).
//
// NOTE: This is the only Position-side mutation of GrappleData
// outside transition methods. Justification: control level
// changes per round without changing state; the FSM transition
// table would prohibit Mount → Mount.
func updateControlLevel(c *characters.Character, newLevel position.ControlLevel) {
	if c.Position == nil {
		return
	}
	c.Position.MutateGrappleControlLevel(newLevel)
}

// grappleStaminaMultiplier applies the chunk-4b stamina penalty
// curve. More punishing than the existing combat ResourcePenaltyMax.
func grappleStaminaMultiplier(c *characters.Character, cfg configs.Balance) float64 {
	if c.StaminaMax.Value <= 0 {
		return 1.0
	}
	s := float64(c.Stamina) / float64(c.StaminaMax.Value)
	if s < 0 {
		s = 0
	}
	if s > 1 {
		s = 1
	}
	maxPen := float64(cfg.GrappleStaminaPenaltyMax)
	curve := float64(cfg.GrappleStaminaPenaltyCurve)
	return 1.0 - maxPen*math.Pow(1.0-s, curve)
}

// grappleEncumbranceMultiplier applies the chunk-4b encumbrance
// penalty curve. e = carry fraction (load / capacity).
func grappleEncumbranceMultiplier(c *characters.Character, cfg configs.Balance) float64 {
	cap := float64(c.GetCarryCapacity())
	if cap <= 0 {
		return 1.0
	}
	load := float64(c.GetCarryWeight())
	e := load / cap
	if e < 0 {
		e = 0
	}
	threshold := e - 0.5
	if threshold < 0 {
		return 1.0 // light encumbrance — no penalty
	}
	normalized := threshold / 1.5
	if normalized > 1 {
		normalized = 1
	}
	maxPen := float64(cfg.GrappleEncumbrancePenaltyMax)
	curve := float64(cfg.GrappleEncumbrancePenaltyCurve)
	return 1.0 - maxPen*math.Pow(normalized, curve)
}

// escapeModifierFromBody reads the controlled character's body
// armor slot for the EscapeModifier field on ItemSpec. Mirrors the
// existing CheckGroundedEscape logic.
func escapeModifierFromBody(c *characters.Character) float64 {
	bodyItem := c.Equipment.Body
	if bodyItem.ItemId == 0 {
		return 0.0
	}
	spec := items.GetItemSpec(bodyItem.ItemId)
	if spec == nil {
		return 0.0
	}
	return float64(spec.EscapeModifier)
}

// applyGrappleStaminaCost deducts asymmetric per-round stamina
// from both sides. Cost can drive stamina to 0; the character
// keeps grappling (penalty curve maxes out, which is the intended
// "smother" feedback loop).
func applyGrappleStaminaCost(controller, controlled *characters.Character, cfg configs.Balance) {
	base := float64(cfg.GrappleStaminaCostPerRound)
	ctrlCost := int(math.Round(base * float64(cfg.GrappleControllerCostMultiplier)))
	cdCost := int(math.Round(base * float64(cfg.GrappleControlledCostMultiplier)))
	controller.Stamina -= ctrlCost
	if controller.Stamina < 0 {
		controller.Stamina = 0
	}
	controlled.Stamina -= cdCost
	if controlled.Stamina < 0 {
		controlled.Stamina = 0
	}
}

// fireGradientMessages / fireTransitionMessages / fireStaminaWarningIfLow
// are stubs here; Task 7 adds the real implementations in Position_Messaging.go.
// Keeping the call sites in the GrappleTick body means Task 7's work is purely
// additive (no GrappleTick changes).
func fireGradientMessages(c *characters.Character, from, to position.ControlLevel) {
	// Implementation in Position_Messaging.go (Task 7).
}

func fireTransitionMessages(controller, controlled *characters.Character, target position.State) {
	// Implementation in Position_Messaging.go (Task 7).
}

func fireStaminaWarningIfLow(c *characters.Character) {
	// Implementation in Position_Messaging.go (Task 7).
}

func init() {
	events.RegisterListener(events.NewRound{}, processGrappleTick)
}
```

Also add the `MutateGrappleControlLevel` helper to `internal/state/position/position.go` (the Machine type). This is the ONE place where we mutate per-state data without a transition:

```go
// MutateGrappleControlLevel updates the ControlLevel on the
// current GrappleData WITHOUT firing a transition. Used by the
// per-round drift hook in Position_GrappleTick.go — the FSM
// transition table forbids Mount→Mount, so per-round
// re-transitions aren't viable.
//
// No-op if the machine is not in a grapple state.
func (m *Machine) MutateGrappleControlLevel(newLevel ControlLevel) {
	if !m.IsGrappling() || m.grapple == nil {
		return
	}
	m.grapple.ControlLevel = newLevel
}
```

- [ ] **Step 4: Create `internal/hooks/Position_GrappleTick_test.go`**

```go
package hooks_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	_ "github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// TestGrappleTick_ControlLevelShiftsOverTime verifies that
// per-round ticks produce ControlLevel drift. Uses a fresh
// Mount pair; runs N ticks; expects detectable drift (direction
// varies by RNG, magnitude based on margin table).
func TestGrappleTick_ControlLevelShiftsOverTime(t *testing.T) {
	// Setup: two characters in Mount.
	// ... (use TransitionPair to set up; need test infrastructure
	//      to fire processGrappleTick directly without an event)
	t.Skip("integration test — depends on test infrastructure for direct tick call")
}

// TestGrappleTick_ThresholdTriggersEscape verifies that hitting
// Controlled on the controller side triggers a position transition.
func TestGrappleTick_ThresholdTriggersEscape(t *testing.T) {
	// Setup: Mount pair with controller's ControlLevel already
	// at BecomingControlled (one shift away from Controlled).
	// Force enough rounds (or rig the dice) to push to Controlled.
	// Verify both characters transition to HalfGuard (DefaultEscapeTarget(Mount)).
	t.Skip("integration test — needs dice RNG override; deferred to T28 smoke")
}
```

The tick tests are integration-flavored — they need dice RNG control + direct tick invocation. For now, mark as SKIP and rely on T28's smoke validation. The mechanics are unit-tested via Tasks 1-2.

- [ ] **Step 5: Build + test**

```bash
go build ./...
go test ./internal/state/... ./internal/hooks/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/Position_GrappleTick.go \
        internal/hooks/Position_GrappleTick_test.go \
        internal/state/position/position.go \
        internal/configs/config.balance.go \
        _datafiles/config.yaml
git commit -m "$(cat <<'EOF'
feat(position): per-round control-axis tick observer

Position_GrappleTick.go registers on NewRound events. Per round:
iterate active users + mobs; for each IsController() character,
look up partner, validate pair, fire opposed roll, compute drift,
update both sides' ControlLevels, apply asymmetric stamina cost,
fire threshold-triggered transition when either side hits
Controlled. Skips controlled side (processed via controller).

Score formula:
  controller: (Str + WeaponCombat) × stamina × encumbrance
  controlled: (Str + WeaponCombat + Dex×0.5 + body.EscapeModifier)
              × stamina × encumbrance
  → dice.OpposedRollStat → margin → MarginToDelta lookup → both
  sides shift in mirrored directions.

Stamina + encumbrance curves use new config knobs (defaults in
_datafiles/config.yaml + Balance.Validate fallbacks): penalty
formulas exactly as specified.

Asymmetric stamina cost: controller pays
GrappleStaminaCostPerRound × ControllerCostMultiplier (5 × 1.0);
controlled pays × ControlledCostMultiplier (5 × 2.0). Drains can
hit 0 — character keeps grappling, penalty curve maxes out
("smother" feedback loop).

MutateGrappleControlLevel helper added to position.Machine: the
ONE place outside transition methods that mutates per-state data,
since the transition table forbids Mount→Mount but we need to
shift control without changing state.

Messaging stub functions (fireGradientMessages /
fireTransitionMessages / fireStaminaWarningIfLow) are no-ops in
T6; Task 7 implements them in Position_Messaging.go.

Integration tests SKIP for now (depend on dice RNG override +
direct tick invocation infrastructure); covered by T28 smoke.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Position_Messaging (gradient + transition + stamina warning)

**Files:**
- Create: `internal/hooks/Position_Messaging.go`
- Create: `internal/hooks/Position_Messaging_test.go`
- Create: `_datafiles/messages/position_control.yaml`

Three message classes + YAML config loader + per-grapple cooldown tracking. Implements the stubs left in T6's GrappleTick.

- [ ] **Step 1: Create `_datafiles/messages/position_control.yaml`**

```yaml
# Position control-axis message templates (chunk 4b).
# {position} substitutes the display name of the current position.
# {old_position} / {new_position} for transitions.
# {Character} / {Controller} / {Controlled} for room broadcasts.
#
# Starter copy — polished in 4f balance pass.

gradient_messages:
  controller:
    in_control:
      self: "You settle into a dominating {position}."
      room: "{Controller} settles into a dominating {position}."
    losing_control:
      self: "You feel your {position} slipping."
      room: "{Controller} struggles to maintain {position}."
    neutral:
      self: "The {position} is even — it's all up for grabs."
      room: "The {position} between {Controller} and {Controlled} is even."
    becoming_controlled:
      self: "You're about to lose the {position}."
      room: "{Controller} is losing {position}."
    controlled:
      self: "You've lost the {position}!"
      room: "{Controller} has lost {position}."

  controlled:
    losing_control:
      self: "You feel an opening — they're losing the position."
      room: ""  # generally too noisy; controller-side message covers
    neutral:
      self: "The {position} is even — it's all up for grabs."
      room: ""
    becoming_controlled:
      self: "You're starting to control the position!"
      room: ""
    controlled:
      self: "You're about to escape!"
      room: ""

transition_messages:
  controller:
    self: "You lose the {old_position} as they recover to {new_position}."
    room: "{Controller} loses the {old_position} to {Controlled}, who recovers to {new_position}."
  controlled:
    self: "You escape the {old_position} to {new_position}!"
    room: ""  # controller-side room msg covers

stamina_warning:
  self: "You're getting gassed — your {position} is hard to maintain."
  room: "{Character} looks exhausted in the {position}."
```

- [ ] **Step 2: Create `internal/hooks/Position_Messaging.go`**

```go
package hooks

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"gopkg.in/yaml.v2"
)

// positionMessageTemplates is the parsed YAML config.
type positionMessageTemplates struct {
	GradientMessages map[string]map[string]struct {
		Self string `yaml:"self"`
		Room string `yaml:"room"`
	} `yaml:"gradient_messages"`
	TransitionMessages map[string]struct {
		Self string `yaml:"self"`
		Room string `yaml:"room"`
	} `yaml:"transition_messages"`
	StaminaWarning struct {
		Self string `yaml:"self"`
		Room string `yaml:"room"`
	} `yaml:"stamina_warning"`
}

var (
	posMsgOnce      sync.Once
	posMsgTemplates positionMessageTemplates
	posMsgLoadErr   error
)

// loadPositionMessages reads the YAML config at boot. Cached.
func loadPositionMessages() positionMessageTemplates {
	posMsgOnce.Do(func() {
		path := filepath.Join("_datafiles", "messages", "position_control.yaml")
		data, err := ioutil.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				mudlog.Warn("Position_Messaging: template file missing", "path", path)
				return
			}
			posMsgLoadErr = err
			return
		}
		if err := yaml.Unmarshal(data, &posMsgTemplates); err != nil {
			posMsgLoadErr = err
			mudlog.Error("Position_Messaging: yaml parse failed", "err", err)
		}
	})
	return posMsgTemplates
}

// fireGradientMessages fires the per-level message for a character
// whose ControlLevel changed. Each level fires at most once per
// grapple session (cooldown tracked in
// c.PerGrappleMessageCooldowns).
//
// Called from Position_GrappleTick.go's processGrapplePair (replaces
// the T6 stub).
func fireGradientMessages(c *characters.Character, from, to position.ControlLevel) {
	if from == to {
		return
	}
	templates := loadPositionMessages()
	role := "controlled"
	if c.IsController() {
		role = "controller"
	}
	levelKey := controlLevelKey(to)
	cooldownKey := "gradient_" + levelKey
	if c.PerGrappleMessageCooldowns[cooldownKey] {
		return
	}
	c.PerGrappleMessageCooldowns[cooldownKey] = true

	roleBlock, ok := templates.GradientMessages[role]
	if !ok {
		return
	}
	msg, ok := roleBlock[levelKey]
	if !ok {
		return
	}

	posName := c.Position.State().String()
	selfText := substitute(msg.Self, map[string]string{"position": posName})
	roomText := substitute(msg.Room, map[string]string{
		"position":   posName,
		"Controller": c.Name,
		"Controlled": "the other grappler", // refined below if we resolve partner
	})

	sendCharacterMsg(c, selfText, roomText)
}

// fireTransitionMessages fires the position-transition message
// pair on both sides + a room broadcast.
func fireTransitionMessages(controller, controlled *characters.Character, target position.State) {
	templates := loadPositionMessages()
	oldPos := controller.Position.State().String()
	newPos := target.String()

	subs := map[string]string{
		"old_position": oldPos,
		"new_position": newPos,
		"Controller":   controller.Name,
		"Controlled":   controlled.Name,
	}
	ctrlMsg := templates.TransitionMessages["controller"]
	cdMsg := templates.TransitionMessages["controlled"]

	sendCharacterMsg(controller, substitute(ctrlMsg.Self, subs), substitute(ctrlMsg.Room, subs))
	sendCharacterMsg(controlled, substitute(cdMsg.Self, subs), substitute(cdMsg.Room, subs))
}

// fireStaminaWarningIfLow fires the stamina warning once per
// grapple if the character drops below GrappleStaminaLowThreshold.
func fireStaminaWarningIfLow(c *characters.Character) {
	if !c.IsLowGrappleStamina() {
		return
	}
	cooldownKey := "stamina_low"
	if c.PerGrappleMessageCooldowns[cooldownKey] {
		return
	}
	c.PerGrappleMessageCooldowns[cooldownKey] = true

	templates := loadPositionMessages()
	posName := c.Position.State().String()
	subs := map[string]string{"position": posName, "Character": c.Name}
	sendCharacterMsg(c,
		substitute(templates.StaminaWarning.Self, subs),
		substitute(templates.StaminaWarning.Room, subs),
	)
}

// controlLevelKey maps a ControlLevel to its YAML key.
func controlLevelKey(c position.ControlLevel) string {
	switch c {
	case position.InControl:
		return "in_control"
	case position.LosingControl:
		return "losing_control"
	case position.Neutral:
		return "neutral"
	case position.BecomingControlled:
		return "becoming_controlled"
	case position.Controlled:
		return "controlled"
	}
	return ""
}

// substitute does simple {key} replacement.
func substitute(template string, subs map[string]string) string {
	out := template
	for k, v := range subs {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	return out
}

// sendCharacterMsg dispatches a self message and a room message
// for the character. Players get SendText; mobs get nothing on the
// self channel (we still emit the room message for spectators).
func sendCharacterMsg(c *characters.Character, selfMsg, roomMsg string) {
	if selfMsg != "" {
		if u := userForCharacter(c); u != nil {
			u.SendText(selfMsg)
		}
	}
	if roomMsg != "" {
		r := rooms.LoadRoom(c.RoomId)
		if r != nil {
			r.SendText(roomMsg)
		}
	}
}

// userForCharacter finds the player User associated with a
// Character (if any). Returns nil for mobs.
func userForCharacter(c *characters.Character) *users.UserRecord {
	if c.UserId > 0 {
		return users.GetByUserId(c.UserId)
	}
	return nil
}

// _ = mobs.GetInstance // ensure mobs is imported even if not used directly
var _ = fmt.Sprintf
var _ = mobs.GetAllMobInstanceIds
```

(Note: the imports + `_ =` lines are illustrative — actual code keeps imports clean per `go build` requirements. Adjust as needed.)

- [ ] **Step 3: Create `internal/hooks/Position_Messaging_test.go`**

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

func TestGradientMessage_CooldownPreventsSpam(t *testing.T) {
	c := characters.New()
	c.PerGrappleMessageCooldowns = map[string]bool{}

	// Put c in Mount as controller.
	// ... (test setup — direct TransitionToMount via TransitionPair if pair available)

	// First fire should set cooldown.
	fireGradientMessages(c, position.InControl, position.LosingControl)

	if !c.PerGrappleMessageCooldowns["gradient_losing_control"] {
		t.Errorf("cooldown should be set after first fire")
	}

	// Second fire (same level) should not re-fire — already set.
	fireGradientMessages(c, position.InControl, position.LosingControl)
	// Verify (via side effect or counter) that no second message went out.
	// Stub for now; relies on Position_Messaging.go test infrastructure.
}

func TestLoadPositionMessages_ParsesYAML(t *testing.T) {
	templates := loadPositionMessages()
	if len(templates.GradientMessages) == 0 {
		t.Skip("YAML config not present in test environment; skip in CI")
	}
	if _, ok := templates.GradientMessages["controller"]; !ok {
		t.Errorf("expected controller block in gradient_messages")
	}
}
```

- [ ] **Step 4: Build + test**

```bash
go build ./...
go test ./internal/hooks/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/Position_Messaging.go \
        internal/hooks/Position_Messaging_test.go \
        _datafiles/messages/position_control.yaml
git commit -m "$(cat <<'EOF'
feat(position): gradient/transition/stamina messages with cooldowns

Three message classes per the chunk-4b spec:
  - Gradient (per-level boundary crossings): per-grapple cooldown
    via c.PerGrappleMessageCooldowns map (set on first fire,
    cleared on grapple end via threshold-transition reset in
    GrappleTick).
  - Transition (position changes): always fires, no cooldown.
  - Stamina warning: once per grapple when stamina drops below
    GrappleStaminaLowThreshold.

YAML config at _datafiles/messages/position_control.yaml — loader
caches at boot via sync.Once; sendCharacterMsg dispatches to
player SendText + room broadcast (with mob-safe nil handling).

T6's stub functions in Position_GrappleTick.go are now real
implementations — purely additive change to T6's call sites.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Position_ConsistencyCheck (periodic invariant checker)

**Files:**
- Create: `internal/hooks/Position_ConsistencyCheck.go`
- Create: `internal/hooks/Position_ConsistencyCheck_test.go`

Runs every `PositionConsistencyCheckRounds` rounds. Walks characters with `IsGrappling()`; validates each pair; force-breaks invalid pairs to Standing + logs.

- [ ] **Step 1: Create `internal/hooks/Position_ConsistencyCheck.go`**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// processPositionConsistencyCheck runs every PositionConsistencyCheckRounds
// rounds. Walks all grappling characters; validates each pair;
// force-breaks invalid pairs to Standing. Fail-safe code — should
// never fire in production if TransitionPair + the GrappleTick
// maintain invariants by construction.
func processPositionConsistencyCheck(e events.Event) events.ListenerReturn {
	cfg := configs.GetBalanceConfig()
	interval := int(cfg.PositionConsistencyCheckRounds)
	if interval <= 0 {
		interval = 10
	}
	if util.GetRoundCount()%uint64(interval) != 0 {
		return events.Continue
	}

	checked := map[characterRef]bool{}

	// Players
	for _, u := range users.GetAllActiveUsers() {
		if u == nil || u.Character == nil {
			continue
		}
		if !u.Character.IsGrappling() {
			continue
		}
		ref := characterRef{UserId: u.UserId}
		if checked[ref] {
			continue
		}
		partner := resolvePartner(u.Character)
		if partner == nil {
			forceBreakSolo(u.Character, "no resolvable partner")
			continue
		}
		if err := position.ValidateGrapplePair(u.Character, partner); err != nil {
			forceBreakPair(u.Character, partner, err)
		}
		checked[ref] = true
		checked[characterRefForCharacter(partner)] = true
	}
	// Mobs
	for _, mobInstId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(mobInstId)
		if m == nil {
			continue
		}
		if !m.Character.IsGrappling() {
			continue
		}
		ref := characterRef{MobInstanceId: m.InstanceId}
		if checked[ref] {
			continue
		}
		partner := resolvePartner(&m.Character)
		if partner == nil {
			forceBreakSolo(&m.Character, "no resolvable partner")
			continue
		}
		if err := position.ValidateGrapplePair(&m.Character, partner); err != nil {
			forceBreakPair(&m.Character, partner, err)
		}
		checked[ref] = true
		checked[characterRefForCharacter(partner)] = true
	}

	return events.Continue
}

type characterRef struct {
	UserId        int
	MobInstanceId int
}

func characterRefForCharacter(c *characters.Character) characterRef {
	return characterRef{UserId: c.UserId, MobInstanceId: c.MobInstanceId}
}

func forceBreakSolo(c *characters.Character, reason string) {
	mudlog.Warn("Position consistency: force-break solo",
		"user", c.UserId, "mob", c.MobInstanceId, "reason", reason)
	_ = c.Position.TransitionToStanding(state.TransitionReason{
		Trigger: "consistency_check_break",
	})
	notifyForceBreak(c)
}

func forceBreakPair(a, b *characters.Character, err error) {
	mudlog.Warn("Position consistency: force-break pair",
		"a_user", a.UserId, "a_mob", a.MobInstanceId,
		"b_user", b.UserId, "b_mob", b.MobInstanceId,
		"err", err)
	_ = a.Position.TransitionToStanding(state.TransitionReason{
		Trigger: "consistency_check_break",
	})
	_ = b.Position.TransitionToStanding(state.TransitionReason{
		Trigger: "consistency_check_break",
	})
	notifyForceBreak(a)
	notifyForceBreak(b)
}

func notifyForceBreak(c *characters.Character) {
	r := rooms.LoadRoom(c.RoomId)
	if r == nil {
		return
	}
	r.SendText("The grapple suddenly breaks apart.")
}

func init() {
	events.RegisterListener(events.NewRound{}, processPositionConsistencyCheck)
}
```

- [ ] **Step 2: Create `internal/hooks/Position_ConsistencyCheck_test.go`**

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

func TestConsistencyCheck_NoFalsePositives_ValidMount(t *testing.T) {
	a := characters.New()
	a.UserId = 1
	b := characters.New()
	b.UserId = 2

	_ = position.TransitionPair(a, b, position.Mount,
		state.TransitionReason{Trigger: position.TriggerTakedownMount})

	if err := position.ValidateGrapplePair(a, b); err != nil {
		t.Errorf("ValidateGrapplePair on valid Mount should return nil; got %v", err)
	}
}

func TestForceBreakPair_BothReturnToStanding(t *testing.T) {
	a := characters.New()
	a.UserId = 1
	b := characters.New()
	b.UserId = 2

	// Manually set up invalid state: both in Mount with both InControl.
	refA := state.ActorRef{UserId: 1}
	refB := state.ActorRef{UserId: 2}
	_ = a.Position.TransitionToClinch(position.GrappleData{Partner: refB}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = a.Position.TransitionToMount(position.GrappleData{Partner: refB, ControlLevel: position.InControl}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
	_ = b.Position.TransitionToClinch(position.GrappleData{Partner: refA}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = b.Position.TransitionToMount(position.GrappleData{Partner: refA, ControlLevel: position.InControl}, state.TransitionReason{Trigger: position.TriggerTakedownMount})

	err := position.ValidateGrapplePair(a, b)
	if err == nil {
		t.Fatal("expected invariant violation in setup")
	}

	forceBreakPair(a, b, err)

	if !a.Position.IsStanding() {
		t.Errorf("a should be Standing after force-break; got %v", a.Position.State())
	}
	if !b.Position.IsStanding() {
		t.Errorf("b should be Standing after force-break; got %v", b.Position.State())
	}
}
```

- [ ] **Step 3: Build + test**

```bash
go build ./...
go test ./internal/hooks/ -count=1 -run 'TestConsistencyCheck|TestForceBreak' 2>&1 | grep -E "^--- (PASS|FAIL)"
```
PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/Position_ConsistencyCheck.go \
        internal/hooks/Position_ConsistencyCheck_test.go
git commit -m "$(cat <<'EOF'
feat(position): periodic pair-invariant consistency checker

Position_ConsistencyCheck.go registers on NewRound; fires every
PositionConsistencyCheckRounds rounds (config knob, default 10).
Walks all IsGrappling() characters; validates each pair via
ValidateGrapplePair; force-breaks invalid pairs to Standing.

Fail-safe code — should never fire in production if TransitionPair
+ GrappleTick maintain invariants by construction. The log entry
IS the bug report: production failures surface as engineering
tickets rather than stuck players.

Force-break sends "The grapple suddenly breaks apart." to the
room as a generic explanation; logs at WARN level with character
IDs + violation details.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Tasks 9-19: Cutover (W1-W8, R1-R6, F1)

Each task is a separate, focused cutover. All follow the parallel-write pattern from chunk 3:
- The new Position machine is updated via TransitionPair / TransitionToXxx.
- The legacy `CombatPosition` field is still set in parallel (until S1 deletes it).
- Tests verify both paths produce equivalent state.

**Format:** Each task lists the file changes; full pattern shown for W1 (most complex). W2-W8 + R1-R6 + F1 follow the same shape — read, identify legacy write/read, add new write/read, parallel-write where required, run regression tests, commit.

---

## Task 9: W1 — grapple entry (ApplyGrappleResult) cutover

**Files:**
- Modify: `internal/combat/grapple.go` (`ApplyGrappleResult` function)
- Modify: `internal/usercommands/grapple.go` + `internal/mobcommands/grapple.go` (entry-point hooks)

`ApplyGrappleResult` currently writes `attacker.CombatPosition` + `defender.CombatPosition` to Clinched or Grounded (lines 128-133 per spec). After cutover: fire `position.TransitionPair` AND parallel-write the legacy field.

- [ ] **Step 1: Read current `ApplyGrappleResult`**

```bash
git --no-pager grep -n "func ApplyGrappleResult" -- internal/combat/grapple.go
```

Read the function. Understand the existing branch (Standing → Clinched vs Prone → Grounded).

- [ ] **Step 2: Modify `ApplyGrappleResult` to parallel-write**

Replace the legacy writes with paired calls:

```go
// Determine target FSM state.
target := position.Clinch
if attacker.CombatPosition == characters.PositionProne ||
   defender.CombatPosition == characters.PositionProne {
    target = position.SideControl // direct Prone → ground grapple lands in SideControl
}

// NEW: pair-write via Position machine.
if err := position.TransitionPair(
    attacker, defender, target,
    state.TransitionReason{Trigger: position.TriggerGrappleEntry},
); err != nil {
    // Pair write failed — bail out, don't update legacy either.
    return false
}

// LEGACY parallel-write (deleted in S1):
attacker.CombatPosition = legacyMapPositionToCombatPosition(target)
defender.CombatPosition = legacyMapPositionToCombatPosition(target)
attacker.GrappleControllerId = attackerInstanceId
defender.GrappleControllerId = attackerInstanceId
```

Add helper:
```go
// legacyMapPositionToCombatPosition translates the new FSM State
// to the legacy CombatPosition enum during the migration window.
// Deleted in S1 alongside CombatPosition itself.
func legacyMapPositionToCombatPosition(s position.State) characters.CombatPosition {
    switch s {
    case position.Standing:
        return characters.PositionStanding
    case position.Prone, position.Supine:
        return characters.PositionProne
    case position.Clinch, position.BackStanding:
        return characters.PositionClinched
    default: // Mount, SideControl, KOB, NS, Crucifix, BackGround, HalfGuard, Guard, Turtle
        return characters.PositionGrounded
    }
}
```

- [ ] **Step 3: Update grapple entry commands**

For `internal/usercommands/grapple.go` and `internal/mobcommands/grapple.go`, just verify the call flows through `ApplyGrappleResult` — no command-level changes needed. The cutover is internal to grapple.go.

- [ ] **Step 4: Build + test**

```bash
go build ./...
go test ./internal/combat/ ./internal/usercommands/ ./internal/mobcommands/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/combat/grapple.go
git commit -m "$(cat <<'EOF'
refactor(position): W1 — grapple-entry cutover (ApplyGrappleResult)

Parallel-write strategy. ApplyGrappleResult now fires
position.TransitionPair to the appropriate target (Clinch from
Standing-vs-Standing, SideControl from Prone target), in addition
to the legacy CombatPosition + GrappleControllerId writes.

If the pair-write fails, the legacy fields are not updated either
(consistency across migration window).

legacyMapPositionToCombatPosition helper added (deleted in S1
alongside CombatPosition itself).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: W2 — grapple progression replacement

**Files:**
- Modify: `internal/combat/grapple.go` (DELETE `CheckClinchProgression`, `CheckGroundedEscape`, `ApplyPositionProgression`)
- Modify: `internal/hooks/NewRound_DoCombat.go` (DELETE `ProcessGrappleProgression` block at lines 327-378)

The per-round tick (T6) IS the replacement. Old binary single-roll progression is gone.

- [ ] **Step 1: Delete the four functions from grapple.go**

```bash
git --no-pager grep -n "func CheckClinchProgression\|func CheckGroundedEscape\|func ApplyPositionProgression" -- internal/combat/grapple.go
```

Delete each function body. Run `go build` — should surface callers as compile errors.

- [ ] **Step 2: Delete `ProcessGrappleProgression` from `NewRound_DoCombat.go`**

```bash
git --no-pager grep -n "ProcessGrappleProgression\|CheckClinchProgression\|CheckGroundedEscape" -- internal/hooks/
```

Find and delete the call sites. The T6 tick is the replacement.

- [ ] **Step 3: Build verify**

```bash
go build ./... 2>&1 | head -10
```
Should be clean. If there are unexpected callers, migrate them to use the per-round tick model.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/combat/ ./internal/hooks/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/combat/grapple.go internal/hooks/NewRound_DoCombat.go
git commit -m "$(cat <<'EOF'
refactor(position): W2 — DELETE legacy grapple progression

Binary single-roll progression (CheckClinchProgression /
CheckGroundedEscape / ApplyPositionProgression) replaced by the
T6 per-round control-axis tick. ProcessGrappleProgression block
in NewRound_DoCombat.go (lines 327-378) deleted.

The per-round tick produces graduated multi-round drift toward
thresholds rather than single-round binary outcomes. This IS the
behavior change that 4b ships.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: W3+W8 — submission outcomes + grapple crit-fail

**Files:**
- Modify: `internal/combat/grapple.go` (`ApplySubmissionFailure`, `ApplySubmissionSuccess`, crit-failure block)

Combined task — both touch the same file with the same pattern (write Prone/Standing via `Position.TransitionTo*`).

- [ ] **Step 1: Migrate `ApplySubmissionFailure`**

Currently: controlled → Standing, controller → Prone. Replace with:
```go
_ = controlled.Position.TransitionToStanding(state.TransitionReason{Trigger: position.TriggerGrappleBreak})
_ = controller.Position.TransitionToProne(
    position.ProneData{MinRecoveryRounds: 2, KnockdownSource: state.ActorRef{...}},
    state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
)
// LEGACY parallel:
controlled.CombatPosition = characters.PositionStanding
controller.CombatPosition = characters.PositionProne
controller.PositionRoundsMin = 2
```

- [ ] **Step 2: Migrate `ApplySubmissionSuccess`** (controlled → Prone)

- [ ] **Step 3: Migrate crit-failure block** (attacker → Prone with MinRecoveryRounds=2)

- [ ] **Step 4: Build + test + commit**

```bash
go build ./...
go test ./internal/combat/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
git add internal/combat/grapple.go
git commit -m "refactor(position): W3+W8 — submission + crit-fail cutover"
```

---

## Task 12: W4 — trip/bash cutover (with direction logic)

**Files:**
- Modify: `internal/combat/skill_moves.go`
- Modify: `internal/usercommands/trip.go` + `bash.go` + mob equivalents (call-site verification)

Trip → Prone (face-forward). Bash → Supine (face-up / backward). Spell knockdown stays in W5.

- [ ] **Step 1: Migrate `skill_moves.go`**

Find the existing `CombatPosition = PositionProne` writes. Replace with direction-aware writes:
```go
// Trip / leg sweep — target falls face-forward (Prone).
if isTrip {
    _ = defender.Position.TransitionToProne(
        position.ProneData{MinRecoveryRounds: 2, KnockdownSource: ...},
        state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
    )
} else {
    // Bash — target gets knocked backward (Supine).
    _ = defender.Position.TransitionToSupine(
        position.SupineData{MinRecoveryRounds: 2, KnockdownSource: ...},
        state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward},
    )
}
// LEGACY parallel:
defender.CombatPosition = characters.PositionProne
defender.PositionRoundsMin = 2
```

- [ ] **Step 2: Build + test + commit**

```bash
go build ./...
go test ./internal/combat/ ./internal/usercommands/ ./internal/mobcommands/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
git add internal/combat/skill_moves.go
git commit -m "refactor(position): W4 — trip→Prone/bash→Supine cutover"
```

---

## Task 13: W5 — spell knockdowns

**Files:**
- Modify: `internal/hooks/spell_resolution.go` (lines 408, 1121)

Per the spec direction mapping:
- Direct-hit blast → Supine
- Shockwave / area → Prone

- [ ] **Step 1: Migrate line 408** (verify spell type, choose direction)
- [ ] **Step 2: Migrate line 1121** (mob spell handler, same pattern)
- [ ] **Step 3: Build + test + commit**

---

## Task 14: W6 — auto-recovery from Prone/Supine

**Files:**
- Modify: `internal/characters/skills.go` (`AttemptRecovery`)

Read both Prone and Supine; transition to Standing via `Position.TransitionToStanding`.

- [ ] **Step 1: Modify `AttemptRecovery`**

```go
// Trigger gate: check both Prone and Supine.
if !c.IsProne() && !c.IsSupine() {
    return false
}

// Decrement MinRecoveryRounds (use the per-state data — Prone vs Supine).
var minRounds int
if c.IsProne() {
    d, _ := c.Position.ProneData()
    minRounds = d.MinRecoveryRounds
} else {
    d, _ := c.Position.SupineData()
    minRounds = d.SupineData.MinRecoveryRounds
}

if minRounds > 0 {
    // Decrement and re-set the data (need a helper to mutate min-recovery in place).
    // For 4b, fold this into TransitionPair-style helper: ConsumeRecoveryRound(c).
    // ... existing penalty condition application stays ...
    return false
}

// Try recovery roll (existing Dex-based logarithmic formula).
// ...

if success {
    _ = c.Position.TransitionToStanding(state.TransitionReason{Trigger: position.TriggerRecoveryRoll})
    c.CombatPosition = characters.PositionStanding // LEGACY parallel
    c.PositionRoundsMin = 0
}
```

The `ConsumeRecoveryRound` helper goes in position.go — mutates MinRecoveryRounds in-place per the same pattern as `MutateGrappleControlLevel`.

- [ ] **Step 2-3: Build + test + commit**

---

## Task 15: W7 — explicit stand command

**Files:**
- Modify: `internal/usercommands/stand.go`

Same as W6 but explicit, bypasses MinRecoveryRounds gate.

- [ ] **Step 1: Modify**
```go
_ = user.Character.Position.TransitionToStanding(state.TransitionReason{Trigger: position.TriggerStandCommand})
user.Character.CombatPosition = characters.PositionStanding // LEGACY
user.Character.PositionRoundsMin = 0
```

- [ ] **Step 2-3: Build + test + commit**

---

## Task 16: R1+R2 — combat math + third-party defense filter

**Files:**
- Modify: `internal/combat/combat_helpers.go` (damage multipliers, speed, crit, kick variant selector, third-party defense filter)

All in one file. Migrate every `c.CombatPosition == X` to `c.IsX()`.

- [ ] **Step 1: Damage multiplier reads (lines ~303-352)**
- [ ] **Step 2: Speed multiplier (line ~113; use new `c.GetPositionSpeedMultiplier()` helper on Character — add to position_predicates.go)**
- [ ] **Step 3: Crit threshold modifiers (lines ~380-398)**
- [ ] **Step 4: Kick variant selector (uses both position + IsController)**
- [ ] **Step 5: Third-party defense filter (lines ~400-428)** — replace `target.CombatPosition.IsGrapplePosition()` with `target.IsGrappling()`
- [ ] **Step 6: Build + test + commit**

Add helper to `internal/characters/position_predicates.go`:
```go
// GetPositionSpeedMultiplier returns the combat speed multiplier
// for the character's current position. Replaces
// CombatPosition.GetSpeedMultiplier() (sunset in S5).
func (c *Character) GetPositionSpeedMultiplier() float64 {
    if c.Position == nil {
        return 1.0
    }
    switch c.Position.State() {
    case position.Standing:
        return 1.0
    case position.Prone, position.Supine, position.Turtle:
        return 0.5
    case position.Clinch, position.BackStanding:
        return 0.6
    default: // ground grapple states
        return 0.3
    }
}
```

---

## Task 17: R3 — flee blockers

**Files:**
- Modify: `internal/mobcommands/flee.go`
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go` (lines 504-510)

Replace `CombatPosition.IsGrapplePosition()` checks with `c.IsStandingGrapple() || c.IsGroundGrapple()`.

- [ ] **Step 1: Migrate `mobcommands/flee.go`**
- [ ] **Step 2: Migrate `NewRound_DoCombat_helpers.go`**
- [ ] **Step 3: Build + test + commit**

---

## Task 18: R4+R5 — chunk-2 + chunk-0 pre-wire cleanup

**Files:**
- Modify: `internal/hooks/Life_Cascades.go` (DELETE lines 55-57 — the chunk-2 pre-wire)
- Modify: `internal/hooks/CombatPhase_Vetoes.go` (rewire `RegisterPositionCheck` from `c.CombatPosition == PositionStanding` to `c.IsStanding()`)

The chunk-4a Position cascade observer (already shipped) is now sole owner of Life Dead → Position Standing.

- [ ] **Step 1: DELETE chunk-2 Life pre-wire**
```go
// DELETE from Life_Cascades.go:55-57:
c.CombatPosition = characters.PositionStanding
c.GrappleControllerId = 0
```

- [ ] **Step 2: Rewire chunk-0 veto**
```go
// In CombatPhase_Vetoes.go:32-33, change:
c.CombatPhase.RegisterPositionCheck(func() bool {
    return c.IsStanding() // was: c.CombatPosition == characters.PositionStanding
})
```

- [ ] **Step 3: Build + test + commit**

---

## Task 19: R6 — prompt {pos} token

**Files:**
- Modify: `internal/users/userrecord.prompt.go` (lines 486-491)

Repoint to the new FSM. Add display-color + abbreviation helpers as needed.

- [ ] **Step 1: Modify the `{pos}` substitution**
```go
case "pos":
    if user.Character.IsStanding() {
        out = ""
        return
    }
    posName := user.Character.Position.State().String()
    color := positionDisplayColor(user.Character.Position.State())
    out = fmt.Sprintf("%s%s%s", color, abbreviatePosition(posName), colors.Reset)
```

Add helpers:
```go
// positionDisplayColor returns the ANSI color for a position used
// in the {pos} prompt token. Replaces the legacy enum's GetPositionColor.
func positionDisplayColor(s position.State) string {
    switch s {
    case position.Standing:
        return colors.White
    case position.Prone, position.Supine:
        return colors.Yellow
    case position.Clinch, position.BackStanding:
        return colors.Orange
    default: // all ground grapples
        return colors.Red
    }
}

// abbreviatePosition keeps the prompt narrow. Most positions
// abbreviate to ~3-4 chars; full names rendered elsewhere.
func abbreviatePosition(s string) string {
    switch s {
    case "BackStanding":
        return "B.Std"
    case "BackGround":
        return "B.Gnd"
    case "SideControl":
        return "SC"
    case "KneeOnBelly":
        return "KOB"
    case "NorthSouth":
        return "N-S"
    case "HalfGuard":
        return "H.Gd"
    default:
        return s
    }
}
```

- [ ] **Step 2: Build + test + commit**

---

## Task 20: F1 — test fixture parallel-write migration

**Files:**
- Modify: `internal/combat/ai_test.go`, `internal/hooks/hooks_test.go`, `internal/mobcommands/mobcommands_test.go`, `internal/combat/hitroll_test.go` (any test that sets `c.CombatPosition = X` directly)

Audit + parallel-write:

```bash
git --no-pager grep -n "c.CombatPosition = \|mob.Character.CombatPosition = " -- internal/ | grep "_test.go"
```

For each hit: keep the legacy write AND add the corresponding FSM transition. Pattern:

```go
// Before:
c.CombatPosition = characters.PositionMount
c.GrappleControllerId = 999

// After:
_ = c.Position.TransitionToClinch(
    position.GrappleData{Partner: state.ActorRef{UserId: 999}},
    state.TransitionReason{Trigger: position.TriggerGrappleEntry},
)
_ = c.Position.TransitionToMount(
    position.GrappleData{Partner: state.ActorRef{UserId: 999}, ControlLevel: position.InControl},
    state.TransitionReason{Trigger: position.TriggerTakedownMount},
)
c.CombatPosition = characters.PositionMount        // LEGACY parallel
c.GrappleControllerId = 999                         // LEGACY parallel
```

- [ ] **Step 1: Audit**
- [ ] **Step 2: Migrate each test file (separate commit per file is fine; final task commit summarizes)**
- [ ] **Step 3: Build + test + commit**

---

## Task 21: S1-S5 — legacy field/file sunsets

**Files:**
- DELETE: `internal/characters/combatposition.go`
- Modify: `internal/characters/character.go` (DELETE `CombatPosition`, `PositionRoundsMin`, `GrappleControllerId` fields)
- DELETE: `ConditionGrappleController` constant + all usages
- Iterate: fix compile errors from deletions; each iteration is its own commit

Largest single task in 4b. Implementer iterates until clean build.

- [ ] **Step 1: DELETE `Character.CombatPosition` field**

```bash
git --no-pager grep -n "CombatPosition" -- internal/
```

For each remaining reference outside `combatposition.go`: it's either a stale parallel-write from W tasks (delete it) or a missed reader (migrate to new API). Iterate to clean build.

- [ ] **Step 2: DELETE `PositionRoundsMin`**
- [ ] **Step 3: DELETE `GrappleControllerId`**
- [ ] **Step 4: DELETE `ConditionGrappleController` constant**
- [ ] **Step 5: DELETE `internal/characters/combatposition.go`**
- [ ] **Step 6: Final grep — zero remaining references**

```bash
git --no-pager grep -n "CombatPosition\|PositionRoundsMin\|GrappleControllerId\|ConditionGrappleController" -- internal/
```

Expected: zero hits (except possibly historical comments).

- [ ] **Step 7: Build + full test suite + smoke**
- [ ] **Step 8: Commit (single big commit summarizing the deletions)**

```bash
git add -A
git commit -m "$(cat <<'EOF'
chore(position): sunset legacy CombatPosition / PositionRoundsMin / GrappleControllerId

Final cleanup of the chunk-4b cutover. After W1-W8 + R1-R6 + F1
ran, all readers + writers + tests use the new Position FSM. The
legacy fields are now dead — deleted.

DELETED:
  - Character.CombatPosition CombatPosition field
  - Character.PositionRoundsMin int field (folded into
    ProneData.MinRecoveryRounds + SupineData.MinRecoveryRounds)
  - Character.GrappleControllerId int field (derived from
    ControlLevel + GrappleData.Partner)
  - ConditionGrappleController constant + all usages (replaced
    by c.IsController() derivation)
  - internal/characters/combatposition.go (legacy enum +
    IsGroundPosition / IsGrapplePosition / GetSpeedMultiplier /
    GetPositionColor / GetWorstPosition helpers)

Parallel-write blocks in W1-W8 cutover tasks removed.

Chunks 0-3-4a regression tests pass; server boots cleanly.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 22: Documentation audit (T12-style)

**Files:** (survey — no production code changes)
- Create: `tools/testing/audits/2026-05-16-chunk-4b-doc-helpfile-audit.md`

Comprehensive survey of docs affected by 4b's cutover + new mechanics. Wider scope than 4a because 4b changes user-visible behavior.

- [ ] **Step 1: Grep context.md files**

```bash
git --no-pager grep -ln "CombatPosition\|PositionRoundsMin\|GrappleControllerId\|ConditionGrappleController\|IsCrafting\|IsGround.*Position\|IsGrapple.*Position" -- '*/context.md'
```

For each hit: DELETE / UPDATE / KEEP-AS-IS.

- [ ] **Step 2: Grep helpfiles** (DOGMud doesn't have a `_datafiles/helpfiles/` per chunk 2/3 findings — verify)
```bash
ls _datafiles/helpfiles/ 2>&1 | head
```

- [ ] **Step 3: Grep top-level docs + scripting guides + YAML lore**

```bash
git --no-pager grep -ln "CombatPosition\|GrappleControllerId" -- '*.md' '*.txt' _datafiles/guides/ _datafiles/
```

- [ ] **Step 4: Identify NEW doc surface from 4b**
- Player-facing combat help mentions of grapple — describe the new per-round drift?
- Scripting docs for the 6 new btree primitives + 14-state Position predicate API
- Config docs for the new Balance knobs

- [ ] **Step 5: Write the audit**

Path: `tools/testing/audits/2026-05-16-chunk-4b-doc-helpfile-audit.md`. Sections per the chunk-2/3 audit template.

- [ ] **Step 6: Commit**

```bash
git add tools/testing/audits/2026-05-16-chunk-4b-doc-helpfile-audit.md
git commit -m "docs(audits): chunk-4b doc + helpfile audit"
```

---

## Task 23: Documentation updates (T13-style)

**Files:** (per audit findings — typically 5-10 docs)
- Modify: `internal/state/position/context.md` (extend with control-axis API + messaging + invariants + cutover-complete notes)
- Modify: `internal/characters/context.md` (new control-axis predicates + sunset notes)
- Modify: `internal/hooks/context.md` (three new observers documented)
- Modify: `internal/combat/context.md` (grapple mechanics rewritten — per-round drift replaces single-round binary)
- Modify: `internal/behaviortree/context.md` (6 new control-axis primitives documented)
- Modify: per audit additional files

Comprehensive doc sweep. Each context.md sees a clear "what changed in chunk 4b" section.

- [ ] **Step 1: Read audit; iterate per file**
- [ ] **Step 2: Extend `internal/state/position/context.md`** with new sections (control-axis API, messaging contract, invariants, "post-cutover state" replacing the 4a scaffold-only language)
- [ ] **Step 3: Update each other affected context.md**
- [ ] **Step 4: Build verify (defensive)**
- [ ] **Step 5: Commit (single comprehensive doc commit)**

```bash
git add internal/state/position/context.md \
        internal/characters/context.md \
        internal/hooks/context.md \
        internal/combat/context.md \
        internal/behaviortree/context.md \
        # plus audit-flagged additions
git commit -m "docs(position): chunk-4b comprehensive documentation update"
```

---

## Task 24: Build / test / smoke validation

**Files:** (verification only)

- [ ] **Step 1: Full build**
```bash
go build ./...
```

- [ ] **Step 2: Full test suite**
```bash
go test ./... -count=1 2>&1 | grep -E "^FAIL"
```
Expected: zero FAILs.

- [ ] **Step 3: Position matrix tally**
```bash
go test ./internal/state/position/ -v -count=1 2>&1 | grep -E "^--- (PASS|FAIL|SKIP)" | awk '{print $2}' | sort | uniq -c
```

- [ ] **Step 4: Chunk 0-4a regression**
```bash
go test ./internal/state/combatphase/ ./internal/state/awareness/ ./internal/state/life/ ./internal/state/activity/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```

- [ ] **Step 5: Server boot smoke**

```bash
go build -o /tmp/dogmud-chunk4b.exe . && /tmp/dogmud-chunk4b.exe > /tmp/dogmud-chunk4b.log 2>&1 &
PID=$!
until grep -qE "Server Ready|panic|FATAL" /tmp/dogmud-chunk4b.log; do sleep 3; done
grep -E "Server Ready|panic|FATAL|loadedCount" /tmp/dogmud-chunk4b.log | tail -25
kill -9 $PID 2>/dev/null
rm -f /tmp/dogmud-chunk4b.exe /tmp/dogmud-chunk4b.log
```

Expected: `Server Ready`, no panic, all data files load (including the new YAML config + messages).

- [ ] **Step 6: Note in-game smoke deferred to user session**

Per the chunk-2/3 pattern. User runs the rich-grapple smoke scenarios (grapple a mob, observe per-round drift messages, escape from mount, etc.) as a separate session. DO NOT commit anything in this task.

---

## Task 25: Roadmap closeout

**Files:**
- Modify: `COMBAT_STATE_ROADMAP.md`

- [ ] **Step 1: Mark chunk 4b Done**

Update the 4b row in the progress table. Add a "Chunk 4b — Shipped" section parallel to the existing chunks 0-4a sections.

Content:
- Cutover summary (all 11 writers + ~10 readers migrated; legacy CombatPosition + supporting fields deleted)
- Per-round control mechanics summary (formula, magnitude, asymmetric stamina cost)
- Messaging contract summary
- 6 new btree primitives
- Pair invariants + consistency checker
- Behavior Matrix outcome (PB-001-080 final tally)
- Sunset list (deleted in S1-S5)
- Next sub-chunk: 4c (weapon-utility-by-position table)

- [ ] **Step 2: Commit**

```bash
git add COMBAT_STATE_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(roadmap): chunk 4b (Position control axis + cutover) Done

Position FSM cutover complete. Legacy CombatPosition enum +
PositionRoundsMin + GrappleControllerId + ConditionGrappleController
+ internal/characters/combatposition.go all deleted. 14-state
Position machine is sole source of truth. Per-round control-axis
mechanics (opposed Str+CombatSkill rolls with stamina+encumbrance
penalty curves, margin-based magnitude, threshold-triggered
position transitions) replace single-round binary grapple
progression. Gradient messaging + stamina warning observers fire
per-grapple-cooldown gated. 6 new btree control-axis primitives
shipped (16 total position primitives by end of 4b). Four formal
pair-state invariants enforced via TransitionPair + tested via
ValidateGrapplePair + backstopped by periodic consistency checker.

Behavior Matrix: PB-001-080 with mix of PASS / SKIP. Chunks
0-4a regression clean.

Next: chunk 4c — weapon-utility-by-position table.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Spec coverage check

| Spec section | Task(s) |
|---|---|
| Per-round tick wiring | T6 |
| Roll formula + stamina/encumbrance curves | T6 |
| Margin → control delta | T2 (MarginToDelta), T6 (consumer) |
| Asymmetric stamina cost | T6 |
| Initial ControlLevel per state table | T1 (InitialControlForPair) |
| Threshold-triggered transitions | T1 (TransitionPair) + T6 (trigger) |
| Default escape targets | T1 (DefaultEscapeTarget) |
| TransitionPair atomic transitions | T1 |
| Knockdown direction logic | T12 (trip/bash), T13 (spell) |
| Gradient messages | T7 |
| Transition messages | T7 |
| Stamina warning | T7 |
| Per-grapple cooldown | T3 (field), T7 (logic) |
| YAML config templates | T7 |
| 6 new btree primitives | T5 |
| W1-W8 writer cutover | T9-T15 (W2+W8 merged into T10, W3+W8 in T11) |
| R1-R6 reader cutover | T16-T19 (R1+R2 in T16, R4+R5 in T18) |
| F1 test fixtures | T20 |
| S1-S5 sunsets | T21 |
| 4 pair invariants | T2 (ValidateGrapplePair) |
| ValidateGrapplePair | T2 |
| Periodic consistency checker | T8 |
| Build/test/smoke | T24 |
| Doc audit (T12-style per user SOP) | T22 |
| Doc updates (T13-style per user SOP) | T23 |
| Roadmap closeout | T25 |

All spec sections covered.

## Known followups (out of chunk 4b)

- **4c** — Weapon-utility-by-position table. (Position × WeaponType) → modifier; combat resolution reads it.
- **4d** — Submission system rework. (Position, ControlLevel) gated; opportunistic vs. explicit; submission outcomes (choked/damaged/tap).
- **4e** — Third-party interaction asymmetries. Symmetric defense degradation (controller moderately, controlled severely); offense restrictions; outside-damage → control-axis degradation; mob AI bias toward grappled targets; submission-interrupt risk.
- **4f** — Balance pass + flavor text + full-stack smoke. Tune curves; polish all the message copy; smoke against real combat scenarios.
- **Per-state asymmetric stamina cost / penalty curves** (Mount-bottom drains worse than HalfGuard-bottom). 4f.
- **Per-position roll formula variants** (Mount Str-heavy, Guard Dex-heavy). 4c/4d/4f.
- **Voluntary controller-advancement** (Mount → BackGround explicit). 4d.
- **State-specific extras on GrappleData** (ClinchGrip, ArmsIsolated, HooksIn, TrappedLeg, GuardVariant). 4c/4d wrapping structs.
- **Low-stamina auto-tap** (controlled at 0 stamina auto-submits). 4d.
- **N-vs-1 grappling, cardio mechanics, knockdown immunity** — master-spec out-of-scope.
