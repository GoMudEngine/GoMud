# State Chunk 4b-fixup — Position Advancement (Design)

**Status:** Draft 2026-05-18 — awaiting user review before writing-plans handoff
**Branch:** `feature/mob-aliveness-1.3-crimes`
**Predecessor chunks:** 4a (FSM scaffold), 4b (drift loop — partially incoherent),
4c (reach utility), 4d (submissions)
**Successor chunks:** 4e (third-party), 4f (balance + full-stack smoke)

---

## 1. Problem Statement

Chunk 4b shipped a 5-level `ControlLevel` drift needle (`InControl →
LosingControl → Neutral → BecomingControlled → Controlled`) intended to
represent "how firmly each side holds their role" in a grapple. The
escape gate fires when either side hits `Controlled` for two consecutive
rounds.

This model is **coherent for symmetric positions** (Clinch, HalfGuard,
Turtle) where both sides start at `Neutral` and drift around. It is
**incoherent for asymmetric established positions** (Mount, SideControl,
Crucifix, BackGround, BackStanding):

- In Mount, `InitialControlForPair` starts the defender at `Controlled`
  by design (they're pinned).
- The escape gate "Controlled for 2 consecutive rounds" then fires on
  round 2 of Mount unless the defender drifts *off* `Controlled` —
  meaning **winning the drift in Mount accelerates the escape**.
- Net effect: dominance fires escape faster, opposite to player
  intuition and real grappling.

Additionally, chunk 4b's per-round drift produces only one observable
outcome: ControlLevel shifts (which never advance position) or threshold
escape (which pops back to Standing). The 9 ground-grapple states
(Mount, SideControl, KneeOnBelly, NorthSouth, Crucifix, BackGround,
HalfGuard, Guard, Turtle) are essentially unreachable in normal play.
`TriggerPositionAdvance` is defined but no production code fires it.

**The FSM is data without behavior for ~9 of its 14 states.** Chunks
4c and 4d built on top of this hollow foundation. The user surfaced
both issues in a single conversation: "all of this wiring is kind of
useless atm, isn't it?"

This chunk corrects both problems together — sunsets ControlLevel
entirely and makes the drift roll resolve directly into position
changes (advance / hold / degrade / reverse / escape), plus richer
flavor messaging so the resulting combat reads as varied and visceral
rather than mechanical.

---

## 2. Design Goals

1. **Engine-driven, opportunistic.** No new player commands. Each
   round in a grapple, drift roll outcome fires automatically. Mirrors
   chunk 4d's submission philosophy ("anti free will choice").
2. **Single concept axis.** Position state is the only thing that
   changes round-to-round. ControlLevel needle abolished.
3. **Outcome variety per round.** Five distinct outcomes (Advance,
   Hold, Degrade, Reverse, Escape) plus an independent sub window.
   Player sees something meaningful in most rounds — either a position
   change, a sub attempt, strikes from a striking apex, or a
   stamina-grind hold round with flavor.
4. **Realism-respecting hierarchy.** Mount is striking apex (you
   pound them); BackGround is control apex (RNC, ground-and-pound from
   the back). Hierarchy matches MMA/BJJ intuition.
5. **Rich messaging.** Every outcome × source × target combination
   has multiple flavor templates. Combat reads varied across rounds
   and across grapples. No hard numbers — pure descriptive language.
6. **Clean composition with 4c (reach) and 4d (submissions).** Reach
   utility's per-state radius continues to apply; submission gate
   composes on top of position result.

---

## 3. Concept Model

**Two concepts only:**

### Position (FSM state)
14 states from chunk 4a. Pure geometric/postural fact: where the bodies
are. `Character.Position.State()`.

### Role
Who is the controller, who is the controlled, or whether the position
is neutral (no controller). Stored via `GrappleData.Partner` pointer.
`IsController()` / `IsControlled()` predicates.

Per-position role assignment:

| Position | Default role assignment |
|---|---|
| Standing, Prone, Supine | N/A (not a grapple) |
| Clinch | Neutral (no controller — both sides started Standing) |
| BackStanding | Controller is the one behind |
| Mount, SideControl, KneeOnBelly, NorthSouth, Crucifix, BackGround | Controller is on top / has back |
| HalfGuard | Controller is on top |
| Guard | Controller is on bottom (inverted — bottom controls via legs) |
| Turtle | Controller is on top |

**ControlLevel is sunset.** No needle. No InControl / LosingControl /
etc. The drift roll resolves directly into position changes.

---

## 4. Per-Round Resolution Algorithm

Each round, for every grapple pair, `Position_GrappleTick` runs:

```
1. Compute attacker and defender scores:
     attacker_score = (Str + WeaponCombat) × stamina_mult × encumbrance_mult
     defender_score = (Str + WeaponCombat + 0.5·Dex + body.EscapeModifier)
                      × stamina_mult × encumbrance_mult
   (Unchanged from chunk 4b.)

2. Roll: dice.OpposedRollStat(attacker_score, defender_score)
   → (success, margin, atkRoll, defRoll)

3. Snapshot to LastDriftRoll on both characters (for chunk 4d reads,
   unchanged).

4. Compute z-score: marginZ = margin / atkRoll.StdDev
   (signed; positive = controller won)

5. Resolve outcome based on z-magnitude and sign:
   - If |z| < 0.5            → Hold
   - Else if z > 0           → Controller-side outcome (advance band)
   - Else                    → Defender-side outcome (degrade/reverse/escape band)

6. Apply outcome (position change via TransitionPair, or no-op for
   Hold).

7. Independent sub gate: if |z| ≥ 1.5 on controller side, open sub
   window FROM the new position (post-step-6). Chunk 4d's
   Position_SubmissionTick reads LastDriftRoll and fires.

8. Apply stamina cost to both sides (unchanged from chunk 4b).

9. Render outcome message (see §7 Messaging).
```

For Clinch (neutral position), whichever side won the drift becomes
the controller in the resulting position. If z magnitude is in the
"hold" band, both stay at Clinch.

---

## 5. Outcome Buckets

Z-magnitude bands map to outcome tier:

| \|z\| band | Outcome tier | Controller-side meaning | Defender-side meaning |
|---|---|---|---|
| < 0.5 | Hold | Status quo | Status quo |
| [0.5, 1.0) | 1-step | Advance 1 step | Degrade 1 step |
| [1.0, 2.0) | 2-step | Advance 2 steps | **Reversal** |
| ≥ 2.0 | 3-step (crit) | Advance 3 steps + sub | **Escape** to Standing |

Sub gate: \|z\| ≥ 1.5 (controller side only — chunk 4d already handles
bottom subs via its own logic from the same `LastDriftRoll` snapshot).

---

## 6. Per-Position Transition Tables

### 6.1 Advancement (controller wins drift)

| Source | 1-step → | 2-step → | 3-step → |
|---|---|---|---|
| Clinch | Mount / SideControl / BackStanding / BackGround (by defender posture) | Mount | BackGround |
| BackStanding | BackGround | BackGround | BackGround |
| **Mount** (striking apex) | Hold (strike) | Hold (strike) | BackGround |
| SideControl | Mount | Mount | BackGround |
| KneeOnBelly | Mount | Mount | BackGround |
| NorthSouth | SideControl | Mount | BackGround |
| Crucifix | Hold (sub-only — see below) | Hold | Hold |
| BackGround (APEX) | Hold (sub-only — see below) | Hold | Hold |
| HalfGuard | SideControl | Mount | BackGround |
| Guard (controller on bottom) | Pass to SideControl (controller now on top) | Mount | BackGround |
| Turtle | BackGround | BackGround | BackGround |

**Sub gate composition.** The sub window (\|z\| ≥ 1.5) is independent
of the advancement outcome and fires AFTER the position change resolves.
So:
- 1-step advances (\|z\| ∈ [0.5, 1.0)) — never open a sub window
  (below threshold).
- 2-step advances (\|z\| ∈ [1.0, 2.0)) — open a sub window iff
  \|z\| ≥ 1.5 (i.e. the top half of this band).
- 3-step advances (\|z\| ≥ 2.0) — always open a sub window.

Crucifix and BackGround as source positions: 1-step and 2-step are
Hold rounds (position doesn't change because there's nowhere more
dominant to go); 3-step is also Hold because you're already at the
apex. The sub gate fires independently — these positions still produce
sub attempts whenever \|z\| ≥ 1.5.

**Clinch 1-step target resolution by defender posture:**
- Defender Prone → SideControl
- Defender Supine → Mount
- Defender turned around (e.g. fled-mid-clinch flag) → BackStanding
- Defender turtled → BackGround
- Else → Mount (default; matches chunk-4b `ApplyGrappleResult` fallback)

**Crucifix entry:** Crucifix is intentionally NOT an advancement target.
It is reached only via chunk-4d submission-attempt flows (the
armbar/americana family transitions through Crucifix briefly during the
attempt). This keeps the advancement graph simple and respects the
real-world reality that Crucifix is a specific arm-trap setup, not a
position climbed to.

### 6.2 Degrade (defender wins moderate — z ∈ [-1.0, -0.5))

| Source | 1-step degrade → |
|---|---|
| Clinch | Hold (no degrade target — symmetric; stamina-drain round) |
| BackStanding | Clinch |
| Mount | SideControl |
| SideControl | HalfGuard |
| KneeOnBelly | SideControl |
| NorthSouth | SideControl |
| Crucifix | BackGround |
| BackGround | Mount |
| HalfGuard | Guard |
| Guard, Turtle | Hold (terminal — defender can only escape or reverse from here) |

### 6.3 Reversal (defender wins big — z ∈ [-2.0, -1.0))

Roles swap. By default the position is unchanged; two realism exceptions
land in a different position:

| Source | Reversal target position | Former controller's new role |
|---|---|---|
| Mount | Guard (former defender comes up between former controller's legs) | Controlled-top of Guard (the inverted Guard role) |
| BackGround | Mount (former defender turns into attacker) | Controlled-bottom of Mount |
| All others | Same position, roles swapped | Controlled-side of the same position |

After reversal, all subsequent drift rolls and outcomes use the new
role assignments. The reversal itself is one full round's outcome —
the next round's drift roll fires fresh against the new configuration.

Note on Guard's role convention (from §3): Guard's "controller" is
the person on bottom (active grappling with legs). So when Mount
reverses to Guard, the former Mount-controller (who was on top)
becomes the Guard-controlled (still on top, but now defending against
the active legs of the person below them).

### 6.4 Escape (defender wins decisive — z ≤ -2.0)

`TransitionPair(controller, controlled, Standing, TriggerControlledEscape)`.
Hard exit; both back on their feet. GrappleData cleared.

---

## 7. Messaging Design

**This is half the value of the chunk.** Without varied, evocative
flavor text, the combat reads as a mechanical stat dump. The mechanics
above are useless if every round of every grapple sounds the same.

### 7.1 Template library structure

A new file `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml`
holds all flavor templates. Loaded at boot, hot-reloadable.

Top-level structure:

```yaml
advancements:
  clinch_to_mount:
    controller: [...]     # 3+ templates, picked randomly
    controlled: [...]
    observers: [...]
  clinch_to_sidecontrol:
    controller: [...]
    controlled: [...]
    observers: [...]
  # ... one block per (source, target) pair in the advancement table

degradations:
  mount_to_sidecontrol:
    controller: [...]
    controlled: [...]
    observers: [...]
  # ...

reversals:
  mount_reverse:        # special: target is Guard
    controller: [...]
    controlled: [...]
    observers: [...]
  generic_reverse:      # role swap in same position
    controller: [...]
    controlled: [...]
    observers: [...]
  # ...

escapes:
  generic_escape:
    controller: [...]
    controlled: [...]
    observers: [...]

holds:
  # Hold rounds get sparse flavor — most rounds silent, occasional
  # "you grind for position" / "blood and sweat" line every ~3-4 rounds
  # to avoid noise. Tracked per-grapple via existing
  # PerGrappleMessageCooldowns map.
  clinch_hold: [...]
  mount_hold_strike: [...]      # Mount-specific — emphasizes striking
  ground_hold_generic: [...]
  # ...

striking_apex:
  # Mount-only: when controller holds Mount (1-step or 2-step), they
  # get a striking flavor line in addition to whatever combat damage
  # rolls produce that round. Reinforces "you are pounding them."
  mount_strike_flavor: [...]   # 5+ visceral options
```

### 7.2 Template tone and variety requirements

- **Minimum 3 templates per (source, target) advancement pair.**
- **Minimum 3 templates per degradation pair.**
- **Minimum 5 templates for Mount strike flavor** (the apex round
  outcome that fires most often once Mount is reached).
- **No hard numbers.** Descriptive language only per the project SOP
  (`combat.GetDamageDescription` style — never "13 damage").
- **Visceral, MMA/BJJ-flavored vocabulary.** "Posts on the
  cross-face," "frames the elbow," "rides the hooks tight,"
  "scrambles back to half," "buckle gives way," "rolls through and
  comes up on top." Not generic-fantasy "you grapple skillfully."
- **Three speaker variants per template** — `controller` (second-person:
  "You drive forward and take mount"), `controlled` (second-person:
  "She drives forward and pins you flat"), `observers` (third-person
  to the room: "She drives forward and mounts him"). Existing
  Position_Messaging templates use the same triad.
- **Cooldown:** each (outcome × pair) template fires at most once
  per grapple via the existing `PerGrappleMessageCooldowns` map.
  Forces variety within a single grapple.

### 7.3 Example templates (to seed authoring)

To make the tone concrete, here are starter templates for one
advancement pair (Clinch → Mount) and the Mount-hold-strike flavor:

**clinch_to_mount.controller:**
- "You wrench the underhook, drive forward, and ride them down into mount."
- "A snap-down sets it up — you sprawl over the shoulder and slide into mount before they can recover."
- "Their stance breaks. You drag them flat and climb on top, knees high in mount."

**clinch_to_mount.controlled:**
- "{controllerName} wrenches the underhook and rides you down. The ceiling spins overhead — they're mounted on your chest."
- "You feel the snap-down too late. Their weight crashes onto your sternum and the world goes vertical."
- "Your stance crumbles. {controllerName}'s knees hammer up under your armpits — full mount."

**clinch_to_mount.observers:**
- "{controllerName} wrenches an underhook and rides {controlledName} down into mount."
- "{controllerName} snaps {controlledName} down and climbs into mount before they can post."
- "{controlledName}'s stance breaks; {controllerName} drags them flat and mounts them."

**striking_apex.mount_strike_flavor:**
- "You ride high in mount and rain elbows down."
- "Your knees pin {controlledName}'s shoulders flat — heavy shots punch through their guard."
- "{controlledName} bridges weakly under you; you posture up and drive hammerfists into their guard."
- "Sweat and copper. You ride the mount and drop knuckles like pistons."
- "Their arms tire from defending their face. You sit heavy and let the strikes through."

### 7.4 Authoring scope

Templates needed at chunk-fixup ship (counted minimum, more is better):

| Category | Pairs | Templates per pair (min) | Total templates (min) |
|---|---|---|---|
| Advancements | ~12 (see §6.1) | 3 controller + 3 controlled + 3 observer = 9 | ~108 |
| Degradations | ~7 (see §6.2) | 9 | ~63 |
| Reversals | 3 (Mount→Guard, BackGround→Mount, generic) | 9 | ~27 |
| Escapes | 1 generic | 9 | ~9 |
| Holds | ~5 categories | 3 | ~15 |
| Mount strike apex | 1 | 5 (single-speaker) | 5 |
| **Total** | | | **~227 templates** |

This is the minimum to ship. The plan must include authoring
ALL of these — not "TBD" or "fill in later." Templates that ship as
placeholders or stubs invalidate the chunk's premise.

### 7.5 Sunset old chunk-4b messaging

`Position_Messaging.go`'s gradient messages ("losing control of the
clinch", "becoming controlled") are sunset along with the ControlLevel
concept. The transition-message and stamina-warning code paths in
Position_Messaging stay — the message *firing* hook gets new event
types.

---

## 8. What Chunk 4b Code Gets Sunset

| Artifact | Disposition |
|---|---|
| `internal/state/position/control.go` entire file | Delete |
| `ControlLevel` enum, `Neutral`/`InControl`/`LosingControl`/`BecomingControlled`/`Controlled` constants | Delete |
| `GrappleData.ControlLevel` field | Delete |
| `InitialControlForPair` | Delete (no level to initialize) |
| `IsControllerLevel`, `IsControlledLevel` | Delete |
| `ControlRankExported`, `MarginToDelta` | Delete (replaced by §5 outcome buckets) |
| `Machine.MutateGrappleControlLevel` | Delete |
| Position_Messaging.go gradient message functions | Delete |
| Position_GrappleTick.go drift-needle math (lines 145-230 area) | Replace with outcome resolver |
| Sustained-pressure escape gate ("Controlled for 2 consecutive rounds") | Delete (escape is now z ≤ -2.0 outcome) |
| BTree primitives reading ControlLevel (`control_level_at_least`, etc. — chunk 4b's 6 control-axis primitives) | Delete or convert to role checks |
| Chunk 4b's gradient message templates in `Position_Messaging.go` constants | Delete |
| `position_test.go` tests touching ControlLevel | Delete |
| `control_test.go` entire file | Delete |

---

## 9. What Survives Unchanged

| Artifact | Notes |
|---|---|
| Position FSM (14 states, ~75 edges) | All edges still valid; new outcome resolver uses them |
| `TransitionPair` / `ValidateGrapplePair` / `Position_ConsistencyCheck` | Pair invariants still hold |
| Drift roll formula (Str + WeaponCombat + stamina + encumbrance) | Same scores, same opposed roll |
| `LastDriftRoll` snapshot (chunk 4d) | Same write site, chunk 4d still reads it |
| `Position_SubmissionTick` (chunk 4d) | Sub gate at \|z\| ≥ 1.5; composes after position change |
| Reach utility (chunk 4c) | Per-state radius unchanged; advancement changes the state, reach recomputes naturally |
| Stamina cost per round | Same asymmetric formula |
| `escapeModifierFromBody` (chunk 4b) | Still applied to defender score |
| Player `grapple` command + mob `grapple` btree action | Entry points unchanged |
| `Position_Cascades.go` (life-dead cascade) | Death handling unchanged |

---

## 10. Migration Order

1. Add new outcome resolver + transition tables (new file
   `internal/state/position/outcomes.go`).
2. Add new messaging library loader + template YAML.
3. Refactor `Position_GrappleTick.go` to call resolver instead of
   shift-needle math; keep both paths behind a config flag for one
   commit so behavior diff is reviewable.
4. Author all messaging templates (~227 minimum).
5. Sunset ControlLevel concept (delete files in §8). One commit.
6. Sunset chunk-4b gradient messages. One commit.
7. Update btree primitives (delete control-axis ones or convert to
   role checks).
8. Update all helpfiles touching grapple mechanics.
9. Update all `context.md` files documenting chunk 4b's drift model.
10. Boot smoke + AI tester smoke against the local server.

The plan must enforce this ordering — sunsetting ControlLevel before
the new resolver is wired produces a broken intermediate state.

---

## 11. Testing Strategy

### Unit tests
- One test per outcome bucket × source position (advance, degrade,
  reverse, escape, hold from each of the 11 grapple positions).
- Reversal exception tests (Mount → Guard with role assertions,
  BackGround → Mount with role assertions).
- Clinch posture-based target selection (defender prone → SideControl,
  defender supine → Mount, defender turned → BackStanding, defender
  turtled → BackGround, default → Mount).
- Sub gate composition (z = 2.5 from Mount: assert advancement to
  BackGround AND sub window opened from BackGround; z = 1.7 from Mount:
  Hold + sub window opens from Mount; z = 0.8 from Mount: Hold, no sub).
- Boundary z values — assert correct bucket on each side of each
  threshold: ±0.499, ±0.500, ±0.999, ±1.000, ±1.499 (sub gate boundary
  on positive side), ±1.500, ±1.999, ±2.000.
- Hold round stamina drain matches existing 4b cost.
- Terminal-degrade positions (Clinch, Guard, Turtle) — defender
  moderate-win produces Hold, not a phantom transition.

### Integration tests
- Full grapple lifecycle: Clinch → Mount → BackGround → escape, with
  asserts on position state and role at each round.
- Reversal mid-sequence: Mount → defender reverses to Guard → Guard
  drift roll fires correctly with role swap.
- 4d interaction: sub window opens correctly from the post-advance
  position.

### Smoke tests
- Local server boot, log scan for panics in Position_GrappleTick.
- AI feature-tester role: 50 simulated combats with grappling-archetype
  mob (predator wolf or warren guard); count position-change events
  per fight; assert >= 1 advance per fight on average and >= 1 unique
  message template per outcome category.
- AI feel-tester role: read combat transcripts and report whether
  combat reads as varied and visceral.

---

## 12. Out of Scope

- **Third-party attacks on grappled pairs** — chunk 4e.
- **Balance tuning** — chunk 4f. We pick reasonable starting values
  (per §5 bucket table) but don't promise they're tuned.
- **Player-initiated advance commands** (`mount`, `pass guard`) —
  explicitly rejected during brainstorm; engine-only opportunistic.
- **Position-specific stamina cost variations** — current uniform
  cost stays.
- **Time-in-position decay or fatigue effects** — separate concern.
- **Mob policy gating of advancement** (e.g. subdue archetype prefers
  to stay in lower-control positions) — could be added later, not
  needed for shipping.
- **Submission-as-position-flavor** outside chunk 4d's existing
  system — chunk 4d's sub mechanics stay as authored.

---

## 13. Risks / Open Questions

- **Pacing.** With Hold being the most common outcome (\|z\| < 0.5
  has ~38% probability on a standard normal), grapples will average
  ~2-3 rounds per position before a change. May feel slow; tune in 4f.
- **Reach utility revaluation.** Chunk 4c's per-state radius applies
  per current position. Advancement now changes position more
  frequently, so weapon utility shifts rapidly mid-grapple. Test
  whether this produces interesting choices or just confusion.
- **Stamina death spirals.** Hold rounds drain stamina, lower stamina
  worsens score, worsens outcome → does this produce a desirable
  "exhausted defender" dynamic or just feel like inevitable doom?
  Watch in smoke.
- **Template authoring effort.** ~227 templates is significant prose
  work. Risk of burnout or inconsistency. Mitigate by authoring per
  category in batches with a tone-reference example at top of each
  section.
- **BTree primitive churn.** Chunk 4b's 6 control-axis btree
  primitives may be referenced in zone-specific behaviors. Sweep
  `_datafiles/world/dogmud/behaviors/` before deletion.

---

## 14. Files Touched (preview for plan)

### Created
- `internal/state/position/outcomes.go` (resolver + transition tables)
- `internal/state/position/outcomes_test.go`
- `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml`
- `internal/grapplemessaging/loader.go` (template loader)
- `internal/grapplemessaging/loader_test.go`

### Modified
- `internal/state/position/position.go` (delete ControlLevel enum)
- `internal/state/position/pair.go` (delete InitialControlForPair)
- `internal/hooks/Position_GrappleTick.go` (rewrite drift math)
- `internal/hooks/Position_Messaging.go` (delete gradient messages, add outcome messages)
- `internal/hooks/Position_SubmissionTick.go` (verify still composes; possibly retune alpha)
- `internal/behaviortree/primitives/*.go` (delete or convert control-axis primitives)
- All `context.md` files documenting chunk 4b's drift model:
  - `internal/state/position/context.md`
  - `internal/hooks/context.md`
  - `internal/behaviortree/context.md` (if it documents control-axis primitives)
  - `internal/combat/context.md` (if it documents 4b drift)
- All helpfiles touching grapple mechanics:
  - `_datafiles/help/grapple.txt` (or wherever it lives)
  - `_datafiles/help/clinch.txt`, `mount.txt`, `position.txt` etc. if they exist
  - `_datafiles/help/combat.txt` (if it mentions ControlLevel)
- `COMBAT_STATE_ROADMAP.md` (chunk 4b-fixup entry)
- `MEMORY.md` (if grapple references need updating)

### Deleted
- `internal/state/position/control.go`
- `internal/state/position/control_test.go`
- Tests in `position_test.go` and `pair_test.go` covering ControlLevel.

---

## 15. Success Criteria

This chunk is done when, in a local smoke test:

1. A predator-wolf grapple results in observable position advancement
   (Clinch → Mount or further) within 3-5 rounds.
2. Mount stays held for multiple rounds with varied strike flavor
   messages (no two consecutive rounds use the same message).
3. Decisive defender rolls produce reversal or escape with appropriate
   messaging — defender doesn't just disappear from the grapple.
4. Sub windows fire from the post-advancement position (e.g. RNC
   family from BackGround, not from Mount).
5. No panics, no stale ControlLevel references logged.
6. AI feel-tester report describes the combat as varied and visceral,
   not mechanical or repetitive.
7. All `context.md` files describing the position system reflect the
   new model — no surviving references to ControlLevel.
8. All helpfiles describe the new outcome model accurately —
   no surviving references to ControlLevel.
