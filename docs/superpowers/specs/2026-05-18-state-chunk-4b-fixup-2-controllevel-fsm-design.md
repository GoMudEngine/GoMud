# State Chunk 4b-fixup-2 — ControlLevel FSM (Design)

**Status:** Draft 2026-05-18 — awaiting user review before writing-plans handoff
**Branch:** `feature/mob-aliveness-1.3-crimes`
**Predecessor chunks:** 4a (Position FSM), 4b (drift needle — sunset),
4c (reach utility), 4d (submissions), 4b-fixup (outcome resolver +
~280 templates)
**Successor chunks:** 4e (third-party), 4f (balance + full-stack smoke)

---

## 1. Problem Statement

Chunk 4b-fixup (26 tasks, shipped 2026-05-18) replaced the chunk-4b
`ControlLevel` 5-step drift needle with a `GrappleData.IsControllerRole
bool` and a direct z-bucket → outcome resolver. The 2026-05-18 AI smoke
test surfaced a critical regression:

**Bug:** In symmetric grapple states (Clinch, HalfGuard, Turtle),
`TransitionPair` sets `IsControllerRole = false` on BOTH sides (the
chunk-4b-fixup convention for "no asymmetric role"). The drift tick
filters characters by `IsController() == true` before running
`processGrapplePair`. Result: neither side passes the filter in
symmetric positions, the per-round drift roll never fires, and Clinch
(the default grapple-entry position) is a dead state — no advancement,
no Hold flavor, no escape, no submission windows.

Player smoke (35 minutes, 65 commands, 55+ rounds in Clinch across two
sessions): zero position changes, zero gradient/outcome messaging, zero
sub windows. `combatstats position` confirmed `Grapple Controller: 0.0%`
across all combat events including ~40 active Clinch rounds.

**Root design error:** simplifying ControlLevel from a 5-state enum to
a bool collapsed the "Neutral" case to "both false" — an implicit-state
invariant the tick filter then violated. The boolean has no way to
express "neither side is dominant *yet*."

**Original chunk-4b-fixup escape gate problem (not regressed):** the
chunk-4b "Controlled for 2 consecutive rounds → escape" gate misfired
in asymmetric Mount because defender STARTS at Controlled by design.
Chunk 4b-fixup correctly removed that gate; escape now fires on
`|z| ≤ -2.0` directly. This chunk does NOT re-introduce the broken
gate. Position outcomes continue to come from the z-bucket resolver.

**Fix scope:** restore ControlLevel as a proper FSM (matching the
codebase pattern at `internal/state/`), make it the source of truth
for "who is the controller side," and parallel-consume drift z for
gradient messaging. Keep the chunk-4b-fixup outcome resolver and
~280 templates untouched.

---

## 2. Design Goals

1. **Match the codebase FSM pattern.** ControlLevel becomes
   `internal/state/control/` alongside Activity, Awareness, CombatPhase,
   Life, Position — same framework, same shape, same testing model.
2. **5 FSM states, 2 transient.** Stable: Controlling, Neutral,
   Controlled. Transient: LosingControl, BecomingControlled — entered
   same-tick during boundary crossings, mirroring Awareness `Revealing`.
3. **Per-character state.** Each character has their own
   `Character.Control` machine. Symmetric positions start both at
   Neutral; asymmetric positions start at the endpoints.
4. **Two parallel consumers of drift z** (per user direction, alongside-
   resolver model):
   - Existing outcome resolver (chunk 4b-fixup, unchanged) consumes
     z-bucket for position changes.
   - New ControlLevel shift logic consumes z magnitude for state
     transitions; transient states fire gradient messaging.
5. **Pair iteration in the tick.** `processGrappleTick` iterates
   grapple pairs (deduped), not individual characters with a bool
   filter. Fixes the Clinch tick-skipping bug at the iteration layer
   independent of ControlLevel.
6. **Restore gradient messaging.** ~36 new templates for the four
   boundary-crossing events (Controlling↔Neutral up/down, Neutral↔Controlled
   up/down), authored per the chunk-4b-fixup tone rubric and realism-
   reviewed before ship.
7. **Aggressor-as-tiebreaker.** New `GrappleData.IsAggressor` bool
   tracks who initiated the grapple. Used only as a tiebreaker for
   the drift roll's attacker-arg when both sides are at the same
   ControlLevel state (typical Clinch round 1). Does not determine
   initial ControlLevel.

---

## 3. Concept Model

| Concept | Type | Lives on |
|---|---|---|
| Position state | FSM (14 states, chunk 4a) | `Character.Position` |
| ControlLevel state | NEW FSM, 5 states (2 transient) | NEW `Character.Control` |
| Aggressor marker | bool field | NEW `GrappleData.IsAggressor` |
| Outcome resolution | Pure function | `position.ResolveOutcome` (chunk 4b-fixup, unchanged) |

### 3.1 ControlLevel state semantics

| State | Stability | Semantics |
|---|---|---|
| Controlling | Stable | This side has positional dominance |
| LosingControl | **Transient** (auto-resolves same-tick) | Traversing Controlling↔Neutral boundary; fires gradient flavor |
| Neutral | Stable | Neither side dominant; symmetric or in-flux |
| BecomingControlled | **Transient** (auto-resolves same-tick) | Traversing Neutral↔Controlled boundary; fires gradient flavor |
| Controlled | Stable | This side is dominated |

Transient state semantics mirror the Awareness `Hidden → Revealing →
Visible` pattern: a state direct-jump (e.g., `Controlling →
LosingControl → Neutral`) executes all transitions in one tick;
subscribers observing on the transient state fire flavor during the
brief in-state moment.

The transient states fire in BOTH directions across their boundary:
- LosingControl fires on Controlling→Neutral AND Neutral→Controlling.
- BecomingControlled fires on Neutral→Controlled AND Controlled→Neutral.

Messaging templates differentiate by traversal direction (from, to).

### 3.2 Role definitions

- **Controller (per-position semantics):** caller of `TransitionPair`
  specifies. For Mount/SC/etc., the dominant side. For Guard (inverted),
  the bottom side (active via legs).
- **Aggressor (per-history):** whoever fired the action that created
  the grapple. Set by `ApplyGrappleResult` from the original
  attacker/defender. Persists for the lifetime of the grapple.
- **POV side for the drift roll attacker-arg:** whichever side has the
  more controller-leaning ControlLevel state; tiebreaker = aggressor.

These three concepts are orthogonal. Aggressor ≠ controller for Guard
(top fighter attacks, gets pulled into guard, is the aggressor but
the controlled side).

---

## 4. Per-Round Resolution Flow

```
For each grapple pair (deduped per round):
  1. Determine drift roll attacker-arg side:
     - whichever side has more controller-leaning ControlLevel state;
     - tiebreaker: IsAggressor.
  2. Compute scores (unchanged from chunk 4b-fixup):
     attacker = (Str + WeaponCombat) × stamina × encumbrance
     defender = (Str + WeaponCombat + 0.5·Dex + EscapeModifier) × stamina × encumbrance
  3. Roll: dice.OpposedRollStat(attacker_score, defender_score)
     → (success, margin, atkRoll, defRoll)
  4. Snapshot LastDriftRoll on both sides (for chunk 4d sub gate).
  5. z = margin / atkRoll.StdDev (signed)
  6. Outcome resolver (chunk 4b-fixup, unchanged):
     - source = current position; defender posture for Clinch dispatch
     - outcome = position.ResolveOutcome(source, z, defenderPosture)
     - apply via TransitionPair if position change
  7. ControlLevel shift (NEW):
     - per the §5 table, shift both sides' ControlLevel state
     - transient states fire same-tick; observers emit gradient messaging
  8. Stamina cost (unchanged); fireStaminaWarningIfLow (unchanged)
  9. Outcome messaging (unchanged ~280 templates)
```

Steps 6 and 7 are independent consumers of the same z. Position changes
fire on z-magnitude per the existing outcome resolver; ControlLevel
shifts fire on a separate (simpler) bucketing.

---

## 5. Drift z → ControlLevel Shift Mapping

Per round, both sides shift in opposite directions based on z magnitude:

| \|z\| range | Shift size | Effect |
|---|---|---|
| < 0.5 | 0 | No shift; both stay at current state |
| 0.5 ≤ \|z\| < 1.5 | 1 step | Winner shifts one state toward Controlling; loser shifts one toward Controlled |
| ≥ 1.5 | 2 steps | Winner shifts two; loser shifts two |

Shifts cap at the endpoints (you can't go past Controlling or
Controlled in either direction).

Each shift may cross one or two boundaries; transient states fire
in sequence same-tick. Example: a side at Controlling shifted 2
steps downward executes `Controlling → LosingControl → Neutral →
BecomingControlled → Controlled`, with both transient states firing
gradient messaging in order.

**Note:** the outcome resolver (§4 step 6) uses a different
z-bucketing (`< 0.5`, `[0.5, 1.0)`, `[1.0, 2.0)`, `≥ 2.0`) for
position changes. The two buckets don't need to align — they consume
the same z for different purposes.

---

## 6. Per-Position Initial ControlLevel State

Set by `TransitionPair` based on the target position's symmetry class:

| Position type | `controller` arg starts at | `controlled` arg starts at |
|---|---|---|
| **Symmetric:** Clinch, HalfGuard, Turtle | Neutral | Neutral |
| **Asymmetric:** Mount, SideControl, Crucifix, BackGround, BackStanding, KneeOnBelly, NorthSouth | Controlling | Controlled |
| **Asymmetric inverted (Guard):** controller is bottom, controlled is top | Controlling | Controlled |

The caller of `TransitionPair` is responsible for passing characters
in the right role slots per position semantics. For Guard:
- Top-fighter-driven-in-guard: `controller=bottom`, `controlled=top`.
  Top fighter (aggressor) → Controlled.
- Bottom-fighter-pulls-guard-from-sweep: same arg ordering.
  Bottom fighter (aggressor) → Controlling.

Aggressor identification is orthogonal — set on `GrappleData.IsAggressor`
by `ApplyGrappleResult`.

---

## 7. Gradient Messaging Library Extension

New `gradients:` section in
`_datafiles/world/dogmud/messaging/grapple_outcomes.yaml`:

```yaml
gradients:
  upper_boundary_down:    # Controlling → Neutral
    self: [...]           # ≥3 templates, second-person; e.g. "your grip slips"
    partner: [...]        # ≥3 templates; e.g. "{controllerName}'s grip slips"
    observers: [...]      # ≥3 templates third-person
  upper_boundary_up:      # Neutral → Controlling
    self: [...]           # e.g. "you find your dominance"
    partner: [...]
    observers: [...]
  lower_boundary_down:    # Neutral → Controlled
    self: [...]           # e.g. "you're being overwhelmed"
    partner: [...]
    observers: [...]
  lower_boundary_up:      # Controlled → Neutral
    self: [...]           # e.g. "you create space"
    partner: [...]
    observers: [...]
```

**4 keys × 3 speakers × ≥3 templates = ≥36 templates minimum.** Tone
follows the chunk-4b-fixup §7.2 rubric (MMA/BJJ vocabulary, visceral
grounded language, no hard numbers). Realism review (fresh-subagent
pass) required before ship per §7.6 of the chunk-4b-fixup spec.

**Speaker variants:** `self / partner / observers` (not
controller/controlled) because the transient fires per-character; the
gradient describes that specific character's state crossing.

**Implementation note:** define a NEW `GradientTriad` struct (parallel
to the existing `TemplateTriad`) with `Self`, `Partner`, `Observers`
string-slice fields, to keep YAML keys + Go field names unambiguous.
Don't reuse `TemplateTriad` and reinterpret its fields — the semantic
mismatch (controller vs self) will confuse future readers and template
authors. The loader's `Library.Gradients` field is therefore
`map[string]GradientTriad`, not `map[string]TemplateTriad`.

**Cooldown:** reuses existing `Character.PerGrappleMessageCooldowns`
map. Same per-grapple variety mechanism as outcome messaging.

Two transient state crossings in one tick (e.g., max-z shift through
both boundaries) fire BOTH gradient lines in sequence. Coalescing is
not required — the rapid sequence reads as a decisive moment to the
player.

---

## 8. Sub Eligibility Refactor

Current (chunk 4b-fixup T18):

```go
func IsTopSubEligible(s position.State, isControllerRole bool) bool
func IsBottomSubEligible(s position.State, isControllerRole bool) bool
```

New:

```go
func IsTopSubEligible(posState position.State, ctrlState control.State) bool {
    if ctrlState != control.Controlling {
        return false
    }
    // existing position-based eligibility (unchanged):
    switch posState {
    case position.Mount, position.SideControl, position.KneeOnBelly,
         position.NorthSouth, position.BackGround, position.Crucifix:
        return true
    }
    return false
}

func IsBottomSubEligible(posState position.State, ctrlState control.State) bool {
    if ctrlState != control.Controlled {
        return false
    }
    switch posState {
    case position.Guard, position.HalfGuard:
        return true
    }
    return false
}
```

**Behavior changes:**
- Sub windows now require the appropriate ControlLevel state, not just
  the position. A controller in Mount whose ControlLevel has drifted
  to Neutral or below can NOT initiate a top sub — they've lost their
  positional dominance enough that sub setups aren't viable. They'd
  need to drift back to Controlling first.
- Bottom subs from Guard/HalfGuard similarly require Controlled state.
  This is intentional: it ties sub setups to actual positional
  dominance rather than just position-state.

Callers (chunk 4d `Position_SubmissionTick`, btree
`conditions_submission.go`) update to pass `Character.Control.State()`
instead of `gd.IsControllerRole`.

---

## 9. Pair Iteration in processGrappleTick

Replace per-character + bool filter with deduped pair iteration:

```go
func processGrappleTick(e events.Event) events.ListenerReturn {
    seen := map[state.ActorRef]bool{}
    for _, u := range users.GetAllActiveUsers() {
        if u == nil || u.Character == nil { continue }
        myRef := selfRef(u.Character)
        if seen[myRef] { continue }
        partner := resolvePartner(u.Character)
        if partner == nil { continue }
        seen[partner.SelfRef] = true
        processGrapplePair(u.Character, partner)
    }
    for _, mobInstId := range mobs.GetAllMobInstanceIds() {
        m := mobs.GetInstance(mobInstId)
        if m == nil { continue }
        myRef := selfRef(&m.Character)
        if seen[myRef] { continue }
        partner := resolvePartner(&m.Character)
        if partner == nil { continue }
        seen[partner.SelfRef] = true
        processGrapplePair(&m.Character, partner)
    }
    return events.Continue
}
```

Key points:
- Mark BOTH sides as seen when processing a pair so we don't
  double-process when iterating the partner.
- No filter — every character in a grapple gets its pair processed
  exactly once per round.
- The drift roll's attacker-arg determination (which side is the
  "controller" for the roll's perspective) moves inside
  `processGrapplePair`.

---

## 10. Validation Invariants

Update `internal/state/position/validation.go`:

| Invariant | Position type | Allowed ControlLevel combinations |
|---|---|---|
| 1 | Symmetric (Clinch, HalfGuard, Turtle) | Both Neutral; OR one Controlling and the other Controlled; OR one Neutral and the other Controlling/Controlled. **NOT** both at the same non-Neutral state. |
| 2 | Asymmetric | One side must have a more controller-leaning state than the other. Equal non-Neutral states are forbidden. |

In practice the invariants reduce to: "the two sides' ControlLevel
states must not both be Controlling, must not both be Controlled."

Transient states are not held between ticks — they should never appear
in a `ConsistencyCheck` snapshot. If they do, that's a bug.

---

## 11. What Survives Unchanged (from chunk 4b-fixup)

| Artifact | Notes |
|---|---|
| `position.ResolveOutcome` + Outcome struct + OutcomeKind enum | Unchanged |
| Per-position transition tables (`AdvancementTarget`, `DegradeTarget`, `ReversalTarget`) | Unchanged |
| z-bucket → outcome mapping (`OutcomeTierFromAbsZ`, `SubWindowOpens`) | Unchanged |
| `position.TransitionPair` for position changes | Minor: drops `IsControllerRole` stamping; gains ControlLevel state initialization |
| `internal/grapplemessaging/` package (loader, render, cooldowns) | Extended with Gradients map + RequiredGradientKeys validator |
| ~280 outcome message templates in `grapple_outcomes.yaml` | Unchanged |
| `processGrapplePair`'s score formula | Unchanged |
| `LastDriftRoll` snapshot (chunk 4d composition) | Unchanged |
| Stamina cost (`applyGrappleStaminaCost`, `fireStaminaWarningIfLow`) | Unchanged |
| Chunk 4d `Position_SubmissionTick` (other than sub eligibility predicate sig) | Unchanged otherwise |
| Sub window gate `|z| ≥ 1.5` | Unchanged |
| Outcome messaging dispatch (`emitOutcomeMessages`, `emitHoldFlavor`, `emitStrikingApexFlavor`) | Unchanged |
| Reach utility (chunk 4c) | Unchanged |

---

## 12. What Gets Replaced (from chunk 4b-fixup)

| Artifact | Disposition |
|---|---|
| `GrappleData.IsControllerRole bool` | Delete — replaced by reading `Character.Control` state |
| `Machine.IsController()` / `Machine.IsBeingControlled()` on Position machine | Refactor to delegate to `Character.Control.State()` |
| `position.IsTopSubEligible(s, bool)` and `IsBottomSubEligible(s, bool)` | Refactor: take `control.State` instead of bool |
| `processGrappleTick` per-character iteration + `IsController()` filter | Replace with pair iteration (§9) |
| Validation invariant 4 (current: "exactly one IsControllerRole=true in asymmetric") | Replace with §10 invariants |
| Chunk-4b-fixup TransitionPair `IsControllerRole: !isSymmetricGrapple(target)` line | Replace with control machine initialization per §6 |
| Sub eligibility callers (`Position_SubmissionTick.go`, `conditions_submission.go`) | Update arg types from bool to `control.State` |

---

## 13. New Artifacts

| Path | Responsibility |
|---|---|
| `internal/state/control/control.go` | FSM State enum, Machine struct, transition methods, mirroring Awareness package shape |
| `internal/state/control/control_test.go` | Unit tests (transitions, transient resolution, idempotency, boundary firing) |
| `internal/state/control/transitions.go` | Trigger constants (`TriggerGrappleEnter`, `TriggerDriftWin`, `TriggerDriftLoss`, `TriggerGrappleExit`) |
| `internal/state/control/context.md` | Package documentation |
| `Character.Control *control.Machine` field | New field on Character, initialized in NewCharacter, reset on grapple entry/exit |
| `GrappleData.IsAggressor bool` field | Set by ApplyGrappleResult; used as drift-roll tiebreaker |
| `Library.Gradients map[string]GradientTriad` | New field on grapplemessaging.Library (uses new GradientTriad type with Self/Partner/Observers fields) |
| `grapplemessaging.GradientTriad` type | New struct: Self/Partner/Observers []string |
| `grapplemessaging.RequiredGradientKeys []string` | Exported list for validator + author reference |
| `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml` `gradients:` section | ~36 new templates across 4 boundary-direction keys |

---

## 14. Migration / Implementation Order

1. Scaffold `internal/state/control/` package + tests.
2. Add `Character.Control` field, initialize in NewCharacter.
3. Add `GrappleData.IsAggressor` field, update `ApplyGrappleResult`.
4. Refactor `TransitionPair`: drop IsControllerRole stamping, add
   ControlLevel state initialization via `Character.Control.Reset(...)`.
5. Refactor `Machine.IsController()` / `IsBeingControlled()` to
   delegate to `Character.Control.State()`.
6. Refactor `processGrappleTick` for pair iteration (the bug fix —
   testable standalone).
7. Add ControlLevel shift logic in `processGrapplePair` (z → shift).
8. Extend grapplemessaging Library with Gradients + validator.
9. Author ~36 gradient templates.
10. Realism review pass on gradient templates (fresh subagent, per
    chunk-4b-fixup §7.6 pattern).
11. Wire gradient messaging observers (init in grapplemessaging
    package; subscribe to transient state transitions).
12. Refactor sub eligibility predicates + update callers.
13. Update validation invariants.
14. Delete IsControllerRole bool field + cleanup.
15. Update context.md (control, position, hooks) + COMBAT_STATE_ROADMAP.
16. Boot smoke + re-run AI feature-tester smoke (chunk-4b-fixup goal
    file is reusable; expected to pass now).

The ordering allows step 6 (pair iteration fix) to ship standalone if
the rest of the chunk is delayed — that single fix resolves the Clinch
tick-skipping bug. Steps 7+ add the ControlLevel state machine and
gradient messaging on top.

---

## 15. Testing Strategy

### Unit tests
- ControlLevel FSM: all transitions (direct + via transient), idempotency,
  endpoint clamping, transient-state same-tick resolution.
- Pair iteration: dedupe correctness, both-sides-marked-seen, partners
  resolved correctly.
- z → ControlLevel shift: boundary z values (0.499, 0.500, 1.499, 1.500),
  step capping at endpoints, two-step shifts crossing both boundaries.
- Sub eligibility: position state × ControlLevel state matrix.
- Validation invariants: pass legitimate configurations, reject
  forbidden ones (both Controlling, both Controlled).

### Integration tests
- Full Clinch lifecycle: enter → drift over N rounds → either side
  reaches Controlling → assert position advancement fires (via the
  unchanged outcome resolver).
- Mount lifecycle: enter at Controlling/Controlled → defender drifts
  back to Neutral → drifts further to Controlling → assert reversal
  fires.
- Pair iteration: spawn two grappling mobs against player, ensure
  each pair processed exactly once per round (no double-firing).

### Smoke tests
- Boot: server starts, control package loads, grapple_outcomes.yaml
  loads with gradient templates, ValidateCompleteness passes.
- AI feature-tester: rerun the chunk-4b-fixup smoke goal file. Should
  now PASS all goals (position advances, hold flavor fires, gradient
  messages observed, mount strike apex reached).
- AI feel-tester: rerun the feel-tester pass. Confirm gradient messages
  read naturally and don't spam during decisive rounds.

---

## 16. Out of Scope

- **Mount escape-gate problem.** Chunk 4b-fixup already fixed this
  by removing the chunk-4b "Controlled for 2 rounds" gate. Escape
  now fires on `|z| ≤ -2.0` directly. This chunk does not re-introduce
  any sustained-state escape gating.
- **Per-position ControlLevel initialization variations** (e.g.,
  KneeOnBelly starting at "mildly Controlling" rather than full
  Controlling). The 3-stable-state model collapses chunk-4b's 5-level
  initial-state nuances. If needed later, drift-roll modifiers can
  reintroduce position-specific bias without state variants.
- **Position advance gating on sustained ControlLevel state.** Per
  user direction (option 2 from brainstorm), position outcomes stay
  z-bucket-based. ControlLevel is for tracking + messaging + AI, not
  outcome gating.
- **Mob policy / archetype-driven ControlLevel behavior.** Could be
  added later (e.g., subdue archetype prefers to stay in Controlling
  without advancing). Not needed for shipping.
- **Hot-reload of gradient templates.** Same loader pattern as
  outcome templates; reload-by-restart is acceptable.
- **Species-gated grappling** (wolves shouldn't BJJ). Tracked
  separately as `project_species_gated_grappling.md`.

---

## 17. Risks / Open Questions

- **Tick fix is the load-bearing change.** Step 6 (pair iteration)
  alone resolves the Clinch-skipping bug. The full ControlLevel FSM
  + gradient messaging is additive. If implementation hits unexpected
  scope, the pair iteration alone is shippable.
- **ControlLevel + outcome resolver dual-consumer model.** Outcomes
  fire on z-bucket regardless of ControlLevel state. This means a
  controller still at Neutral (Clinch round 1) can theoretically
  trigger position advance via z ≥ 2.0 on the first roll, before
  ControlLevel has caught up. Whether this feels natural or
  inconsistent will surface in smoke. Acceptable starting position;
  tune in 4f if needed.
- **Sub eligibility gating tightens.** Subs now require Controlling
  (top) or Controlled (bottom) ControlLevel state, not just position.
  This could reduce sub-window frequency relative to chunk-4b-fixup
  shipping behavior. Mitigation: smoke before shipping; tune sub
  alpha or eligibility threshold if frequency feels off.
- **Transient state same-tick semantics.** Mirrors Awareness Revealing
  — well-tested framework pattern. Risk is low but the boundary-cross
  observer firing during a same-tick chain (e.g., 2-step shift through
  both boundaries) is novel surface; unit tests must cover it.
- **Gradient messaging volume.** Adding ~36 templates on top of the
  existing ~280. Potential for messaging spam if ControlLevel shifts
  frequently. Cooldown logic from chunk 4b-fixup reused — should keep
  in check. Watch in smoke.

---

## 18. Files Touched (preview for plan)

### Created
- `internal/state/control/control.go`
- `internal/state/control/control_test.go`
- `internal/state/control/transitions.go`
- `internal/state/control/context.md`

### Modified
- `internal/characters/character.go` — add `Control *control.Machine` field
- `internal/state/position/position.go` — delete `GrappleData.IsControllerRole`; refactor `IsController()`/`IsBeingControlled()`
- `internal/state/position/pair.go` — add `IsAggressor` field; refactor `TransitionPair` for ControlLevel init
- `internal/state/position/validation.go` — replace invariant 4 with §10 invariants
- `internal/state/position/submissions.go` — refactor `IsTopSubEligible`/`IsBottomSubEligible` sigs
- `internal/state/position/context.md` — document ControlLevel FSM integration
- `internal/hooks/Position_GrappleTick.go` — pair iteration; ControlLevel shift logic
- `internal/hooks/Position_SubmissionTick.go` — update sub eligibility callers
- `internal/hooks/context.md` — update for pair iteration + ControlLevel shift
- `internal/behaviortree/conditions_submission.go` — update sub eligibility callers
- `internal/grapplemessaging/loader.go` — add Gradients map + RequiredGradientKeys validator
- `internal/grapplemessaging/loader_test.go` — gradient validator tests + production-library guard tests
- `internal/combat/grapple.go` (line ~105) — set `IsAggressor=true` for `attacker`, `false` for `defender` in `ApplyGrappleResult` when calling `TransitionPair`
- `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml` — new `gradients:` section
- `COMBAT_STATE_ROADMAP.md` — add chunk 4b-fixup-2 row

### Deleted
- (No standalone file deletions — `IsControllerRole` field removal is in-place.)

---

## 19. Success Criteria

This chunk is done when, in a local AI feature-tester smoke run:

1. A Clinch grapple's drift roll fires every round (verified via
   `combatstats position` showing nonzero Grapple Controller %).
2. Position advances from Clinch to Mount/SC/BackGround within 5-10
   rounds of sustained controller dominance.
3. Gradient messaging fires when ControlLevel crosses boundaries —
   both directions, both boundaries.
4. Mount strike apex flavor fires once Mount is reached (testing
   chunk-4b-fixup wiring end-to-end).
5. Sub windows fire from positions with controller at Controlling
   state (chunk 4d composition intact).
6. No panics, no stale `IsControllerRole` references, no missing
   gradient template debug strings.
7. AI feel-tester report describes combat as varied, paced, and
   coherent — gradient messages don't spam, decisive rounds feel
   decisive.
8. All `context.md` files describe the ControlLevel FSM accurately;
   no surviving references to the `IsControllerRole bool` field.
