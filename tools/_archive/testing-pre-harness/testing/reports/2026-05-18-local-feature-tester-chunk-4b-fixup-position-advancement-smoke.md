# Test Report: Chunk 4b-fixup Position Advancement Smoke

**Date:** 2026-05-18
**Target:** local (localhost:55555)
**Role:** feature-tester
**Character:** smoketester
**Config note:** GlobalDamageMultiplier 0.5 (halved for this smoke)
**Duration:** ~35 minutes, ~65 commands sent

---

## Session Summary

Connected via mud_bridge.py, healed smoketester to full HP using admin
buffs, then spawned Smuggler Enforcer mobs (mobid 246, generic_fighter
archetype, statpool 55) as grapple-capable humanoid targets. Used the
player-side `grapple` command to enter Clinch and observed behavior over
30+ consecutive grapple rounds per engagement. Performed two full grapple
sessions (dying in the second from stamina+HP exhaustion in the Clinch
after mob heal kept the target alive). No position advancement, Hold
flavor, or advancement messaging was observed in either session. Code
inspection confirmed the root cause.

---

## Goal Results

- [ ] **FAIL** — Goal 1: Engage a humanoid mob with a grappling archetype and stay for 5+ grapple rounds.
  The mob (Smuggler Enforcer, generic_fighter) was engaged and a grapple was
  maintained for 30+ rounds using `mob heal` to keep the target alive. The mob
  attempted grapple itself multiple times before I landed it ("tries to grapple
  you, but you slip away"). However, the mob's own grapple attempts were
  consistently deflected; I had to initiate via `grapple enforcer` to enter the
  grapple state. Once in Clinch, the state persisted indefinitely until mob death
  or my own death. Five-round minimum was met.

- [ ] **FAIL** — Goal 2: Verify that position advances at least once (Clinch → Mount or further) within a 10-round window.
  After 30+ rounds in Clinch, no position advancement occurred in either grapple
  session. Root cause identified: see BUG-1 below.

- [ ] **FAIL** — Goal 3: Verify that Hold rounds occasionally show flavor messaging (not every round; sparse).
  Zero Hold flavor lines were emitted across 30+ Hold rounds in two sessions.
  The Hold flavor system (emitHoldFlavor) depends on the processGrappleTick
  firing for a controller, which never fires for Clinch. Root cause: BUG-1.

- [ ] **FAIL** — Goal 4: Verify that Mount rounds show striking-apex flavor.
  Mount was never reached; untestable due to BUG-1.

- [ ] **FAIL** — Goal 5: Verify that if defender wins decisive rolls, you see degradation, reversal, or escape messaging.
  The defender ("tries to break free but you've got them locked down!") attempted
  escape once in the first session but used the legacy flee-block message, not
  the chunk-4b-fixup escape messaging. No degradation or reversal messaging was
  observed. Root cause: BUG-1 prevents drift rolls from firing.

- [x] **PASS** — Goal 6: Report no panics or "missing grapple message template" debug strings.
  No panics observed. No `(grapple messaging missing template)` debug strings
  appeared. The server log contains no grapple-related errors or warnings.

---

## Position Advancement Captures

### Session 1 (Grapple 1 — ~30 rounds)

Entry: `grapple enforcer` → "You grapple smuggler enforcer, transitioning to
clinched position!"

Position at start: `Clinch` (shown in prompt as `CP:##########Clinch`)

**Rounds 1–5:** Stayed at `Clinch`. No flavor. Mob hit, blocked, occasional
kick attempts.

**Rounds 6–10:** Stayed at `Clinch`. Mob "tries to break free but you've got
them locked down!" on one round — this is the legacy grapple escape messaging,
not the chunk-4b-fixup escape outcome.

**Rounds 11–30+:** Stayed at `Clinch`. No position change. No flavor of any
kind from the grapple tick. Only standard combat messages (parry, riposte, hit,
miss, etc.).

Position at end: `Clinch` (unchanged). Mob died to accumulated damage.

### Session 2 (Grapple 2 — ~25 rounds)

Same pattern. Clinch entered, held indefinitely, no advancement. Stamina
drained to 0 on both sides (grapple stamina cost IS working). I died from
HP depletion before the mob did.

Notable: `combatstats position` during Session 2 showed:
```
| Grapple Controller | 0.0%     |
| Non-Controller     | 52.0%    |
```
This confirms analytically that smoketester was never recorded as the grapple
controller even though they initiated the grapple, because IsController()
returns false for Clinch (symmetric state — both sides get
IsControllerRole=false).

---

## Findings

### BUG: Clinch position never advances — processGrappleTick skips both sides

**What was done:** Initiated grapple (`grapple enforcer`) from Standing →
Clinch. Observed for 30+ rounds.

**What happened:** Position stayed at Clinch indefinitely. No advancement, no
Hold flavor, no messaging of any kind from the grapple-tick system.

**Root cause (confirmed via code inspection):**

In `internal/state/position/pair.go`, `TransitionPair()` sets
`IsControllerRole = !isSymmetricGrapple(target)` on the controller's
GrappleData. For Clinch (a symmetric state), this evaluates to `false`. Both
sides get `IsControllerRole = false`.

In `internal/state/position/position.go`, `IsController()` returns
`d.IsControllerRole`, which is `false` for both sides in Clinch.

In `internal/hooks/Position_GrappleTick.go`, `processGrappleTick` only
processes characters where `IsController()` is true:
```go
if !u.Character.IsController() {
    continue
}
```

Since neither side of a Clinch pair is a controller, the per-round drift roll
is never fired. No position advancement, no Hold flavor, and no any other
grapple tick output ever fires from Clinch.

**This affects:** All grapples that enter via Standing → Clinch (the default
path). HalfGuard and Turtle are also symmetric and would have the same issue
if reached.

**Expected behavior:** Per spec §4/§5, Clinch should undergo per-round drift
rolls. One side (the one who initiated, or the "attacker" in a symmetric
dispute) needs a controller role assigned, OR the Clinch tick needs to use a
different dispatch path (pick-attacker, shared tick, etc.) that doesn't rely
on IsController().

**Analytics confirmation:** `combatstats position` showed `Grapple Controller:
0.0%` across 256 events, including ~40 events during active grapple rounds.

**Files involved:**
- `internal/state/position/position.go` — IsController() returns false for Clinch
- `internal/state/position/pair.go` — TransitionPair sets IsControllerRole=false for symmetric states
- `internal/hooks/Position_GrappleTick.go` — processGrappleTick skips non-controllers

---

### OBSERVATION: Grapple stamina cost fires correctly even when tick is broken

Stamina drained from full to 0 during the ~30-round Clinch session, consistent
with `GrappleStaminaCostPerRound=5` and `GrappleControlledCostMultiplier=2`.
This means the stamina cost in `applyGrappleStaminaCost` is being applied, but
NOT via `processGrapplePair` (which never fires). The cost must be applied
somewhere else. This is worth checking — if processGrapplePair isn't firing,
the stamina cost should also not be applying. Possible alternative: the legacy
grapple stamina path in `NewRound_DoCombat.go` or similar is still active.

Update: re-examining the stamina readings more carefully — my stamina showed
SP:######.... → SP:##........ → SP:.......... across ~35 rounds, but some of
that drain was from normal combat (stamina is used by attacks). The drain rate
may not prove processGrapplePair is firing. Inconclusive without isolating
sources.

### OBSERVATION: Mob grapple attempts work correctly (pre-Clinch)

Before I initiated the grapple, the Smuggler Enforcer (generic_fighter
archetype, btree cascade includes grapple for single-enemy targets) attempted
grapple multiple times against me. These showed the correct resist message:
"tries to grapple you, but you slip away!" — the grapple attempt resolution
via ExecuteGrappleMove/AttemptGrapple is working on the mob side.

### OBSERVATION: Legacy escape messaging still present

During Clinch, the mob's escape attempt showed "tries to break free but you've
got them locked down!" — this is the old pre-chunk-4b-fixup messaging from
grapple.go or the usercommand grapple path. The new chunk-4b-fixup escape
messaging from `emitOutcomeMessages(OutcomeEscape)` was never observed because
processGrapplePair never fires.

### OBSERVATION: combatstats position shows 0% for target-position categories

Target Standing, Target Prone, Target Clinched, and Target Grounded all showed
0.0% hit rate in `combatstats position`. This is suspicious — there were clearly
hits during Clinch that should be tagged as "Target Clinched." Suggests the
analytics code for position-based tagging may also be affected by the
IsController() bug, or the position tagging reads a different field. Not the
primary finding but worth a secondary check.

### PASS: Server stability

No panics, no stack traces, no disconnects. Server stayed stable throughout
35 minutes of active testing including repeated mob spawning and mid-combat
admin commands. Log showed no grapple-related WARN or ERROR entries.

### PASS: Grapple entry works

`grapple enforcer` → "transitioning to clinched position!" — entry is clean.
Position prompt updates to `Clinch` on both sides immediately. The initial
grapple command path is functional.

### PASS: Position prompt (pos token) works

`{pos}` token in the prompt correctly shows `Clinch` during grapple and
disappears on return to Standing. Multiple observations confirmed the prompt
token is wired correctly.

---

## Raw Stats

- Commands sent: ~65
- Fights initiated: 5 (2 with dustwalk bandit 80, 3 with smuggler enforcer 246)
- Grapples observed: 2 (both player-initiated via `grapple enforcer`)
- Mob grapple attempts: 4+ (all deflected by player — "you slip away")
- Position advances seen: **0**
- Reversals seen: 0
- Escapes seen: 0 (1 legacy flee-block message, not a chunk-4b-fixup escape)
- Hold flavor lines seen: **0**
- Mount strike-apex lines seen: 0 (Mount never reached)
- Sub windows fired: 0 (Mount never reached)
- "Missing template" debug strings: **0** (PASS)
- Panics: **0** (PASS)
- Rounds in Clinch (combined): ~55
- Position changes from Clinch: **0**
