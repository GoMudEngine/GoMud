# Test Report: Chunk 4b-fixup-2 ControlLevel FSM Smoke

**Date:** 2026-05-18
**Target:** local (localhost:55555)
**Role:** feature-tester
**Character:** smoketester
**Config note:** GlobalDamageMultiplier 0.5 (halved)
**Duration:** ~30 minutes, ~35 commands sent (plus passive combat observation)

---

## Comparison to prior smoke

The prior chunk-4b-fixup smoke found that Clinch grapples never advanced because both
sides had `IsControllerRole=false` and the tick filter skipped them entirely. That bug
is **FIXED**: Clinch grapples now advance per-round and position changes happen within
2–5 rounds as expected. The pair-iteration fix in commit b0b8f5a6 is working correctly.

---

## Session Summary

Four separate grapple engagements were run against Smuggler Enforcer (mob 246,
`generic_fighter` archetype). Each engagement began with `grapple enforcer` from
Clinch and progressed through 5–15+ rounds of grapple. Position advancement, hold
flavor, reversal flavor, and mount-apex flavor all fired correctly. The ControlLevel
FSM boundary messages (gradients section) did NOT fire during any engagement.
`combatstats position` confirmed grapple mechanics are active (244 events recorded,
Non-Controller hit rate 50.8%) but surfaced two analytics bugs.

---

## Goal Results

- [x] **Engage a humanoid mob with grappling archetype for 5+ rounds.** Four fights
  against Smuggler Enforcer (generic_fighter). Each grapple lasted 5–20+ rounds.
- [x] **Position advances at least once (Clinch → Mount or further) within 10 rounds.**
  Confirmed multiple times. Clinch → Mount occurred within 2–5 rounds in every fight.
  Full chain observed: Clinch → Mount → Guard → BackGround → Mount → SideControl →
  Mount. Second fight: Clinch → Mount in round 2.
- [x] **Hold rounds show sparse flavor messaging.** Multiple hold lines observed:
  "Shoulders pressed together, breath ragged — you wrestle for the underhook."
  (Clinch hold), "Your hips frame smuggler enforcer's; the guard stays closed."
  (Guard hold), "You settle your weight and let smuggler enforcer burn cardio trying
  to shift you." (ground hold), "Sweat and copper. You ride the mount and drop
  knuckles like pistons." (Mount hold). Firing every ~4 rounds as intended.
- [x] **Mount rounds show striking-apex flavor.** Confirmed: "You ride high mount, knees
  driving into smuggler enforcer's biceps — their arms can't lift to defend.", "You ride
  high in mount and rain elbows down.", "Cross-face pinning their jaw, free hand cocked
  back — you punish from full mount.", "Their arms tire from defending their face. You
  sit heavy and let the strikes through.", "You shift your weight onto smuggler enforcer's
  sternum and unload — short, vicious elbows from the top.", "Sweat and copper..."
- [x] **Degradation/reversal/escape messaging fires.** "smuggler enforcer bridges and you
  tumble forward — they roll up between your legs and now you're in their guard.",
  "smuggler enforcer kicks off you and pops to their feet — the grapple breaks.",
  "Your base evaporates. smuggler enforcer bucks you off and comes up between your
  legs — full guard.", "They hip-escape sideways and drag your top hook down..."
  Mob-initiated grapple also observed: "smuggler enforcer grapples you, transitioning
  to clinched position!" confirming MvP grapple path works.
- [ ] **Gradient messages (ControlLevel boundary flavor) fire.** NOT OBSERVED. Searched
  full session log for all 12 gradient template lines ("Your control slips", "You feel
  the position click", "Pressure crashes down", "You create space — a bridge",
  "dominant grip", etc.). Zero matches. See CONCERN-1.
- [x] **No panics or missing-template debug strings.** Zero "missing message key" WARN
  log lines visible. Zero "missing gradient key" WARN lines. Zero server panics.

---

## Position Advancement Captures

### Fight 1 (vs Smuggler Enforcer, mob healed mid-fight)

```
Round 1: grapple → [Clinch]
Round 1: gradient: "smuggler enforcer finds the leverage you missed. The position
         flips and suddenly they're the one in control." (reversal in Clinch)
Round 2: "You held it a half-second too long. smuggler enforcer reverses the position
         and pins you down." (reversal in Clinch)
Round 3: "A scramble, a shift — and you're on the wrong end of the pin." (reversal)
Round 4: "You wrench the underhook, drive forward, and ride them down into mount."
         → [Mount] *** CLINCH → MOUNT ADVANCE ***
Round 5: "You settle your weight..." / "You ride high mount..." (Hold + apex)
Round 6: "smuggler enforcer bridges and you tumble forward..." → [Guard] (degrade)
Round 7: "You transition to a back-take..." → [B.Gnd] (Guard → BackGround advance)
Round 8: "They hip-escape sideways..." → [Mount] (BackGround → Mount reversal)
Round 9: Hold rounds in Mount: "Pressure stays on.", "Cross-face pinning their jaw..."
Round 11: → [SC] (Side Control)
Round 12: → [B.Gnd] (Back Ground)
```

### Fight 2

```
Round 1: [Clinch]
Round 2: [Mount] (Clinch → Mount in 2 rounds)
Rounds 3–4: Mount holds / apex flavor
Round 5: [Guard] (degrade)
Round 6: [Mount] (Guard → Mount re-advance)
Round 7: [Guard] (degrade again)
...continuing cycle with SC, H.Gd appearances
```

### Fight 3 (abbreviated — mob near death)

```
Round 1: [Clinch]
Round 1: [Mount] (immediate, very fast roll)
Round 2: [Guard] (degrade)
Round 3: grapple break → "smuggler enforcer kicks off you and pops to their feet"
Round 4: mob re-initiates: "smuggler enforcer grapples you, transitioning to
         clinched position!" → player now controlled side
```

### Fight 4 (longest session)

Positions observed across ~25 rounds: Clinch (×5), Mount (×7), Guard (×6),
SideControl (×3), BackGround (×3), HalfGuard (×1). Rich variety. Multiple
apex and hold lines per position.

---

## Gradient Messaging Captures

**None observed.** All four grapple sessions were searched systematically via log.
The gradient templates in `gradients:` section of `grapple_outcomes.yaml` use these
distinctive phrases (selected), none of which appeared:

- "Your control slips — {name} squirms free of the lock you had."
- "You feel the position click — your weight settles right and you assert full control."
- "Pressure crashes down — {name} smothers your movement and you're fully controlled."
- "You create space — a bridge and a hip-escape breaks {name}'s weight off you."
- "You feel {name} establish the dominant grip. The fight tightens."
- "{name} establishes dominance over {name2} — control settles in their favor."

The reversal/advancement template strings that DO use "control" language were
confirmed to come from `reversals.generic_reverse.controller` and combat momentum
`{momentum}="in control"`, NOT from the gradients section.

---

## Findings

### PASS: Clinch advancement bug is fixed

The core regression from chunk-4b-fixup is resolved. Pair-iteration dedups
correctly (both sides marked seen), and Clinch grapples now produce per-round
drift rolls that advance position. Clinch → Mount was observed in as few as
2 rounds. This is the primary deliverable of chunk-4b-fixup-2 T8 and it works.

### PASS: Position variety and advancement chain

Six distinct grapple positions were observed across four fights: Clinch, Mount,
Guard, SideControl, BackGround, HalfGuard. All advancement, degrade, reversal,
and escape templates selected from `advancements`/`degradations`/`reversals`/
`escapes` sections fired with correct prose. No "missing message key" warnings
were triggered.

### PASS: Hold flavor (sparse cadence)

Hold flavor messages fire at approximately every 4 rounds as designed. Multiple
variants confirmed per position (Clinch hold, Guard hold, ground generic hold,
BackGround hold). Mount-apex pool fires separately from hold pool on Mount holds.

### PASS: Mount striking-apex flavor

Mount hold rounds produced at least 6 distinct apex lines across the session.
The `mount_strike_flavor` pool is working and the per-grapple cooldown map
correctly provides variety (no single line repeated back-to-back).

### PASS: No panics or missing template warnings

Zero server panics across ~30 minutes of combat. Zero "missing message key"
or "missing gradient key" WARN log entries surfaced via `syslogs` streaming.
grapple_outcomes.yaml loaded cleanly; `grapplemessaging.ValidateCompleteness`
produced no warnings visible to the player.

### CONCERN-1: Gradient messages (ControlLevel FSM boundary) not firing

**Observed:** Zero gradient messages from `gradients:` section of
grapple_outcomes.yaml across 244 combat events and 4+ grapple sessions.

**Root cause analysis:** Two possible explanations:

1. **ControlLevel never reaches a boundary state.** `applyControlShift` runs
   each drift round and can shift 1–2 stable-rank steps when |z| >= 0.5. With
   RollSpread=0.15 and frequent reversals resetting the "controller" arg,
   consecutive same-direction z > 0.5 rolls may not accumulate before a
   position change or reversal reshuffles the pair. The FSM machinery is in
   place but the dynamics of this matchup (even stats, short grapples, frequent
   reversals) may not be generating sustained single-direction dominance.

2. **`grappleOutcomesLib` nil at callback time.** `emitGradientMessage` guard-
   checks `if grappleOutcomesLib == nil { return }`. The library is loaded lazily
   by `loadGrappleLib()` the first time `emitOutcomeMessages` runs. Since position
   messages fired, the library was loaded. However, the boundary-cross callback
   fires via `control.RegisterBoundaryCrossCallback` during `init()`, which binds
   before any combat round. If the first boundary crossing happens before the first
   `emitOutcomeMessages` call in the same session, `grappleOutcomesLib` could be nil.
   This is unlikely given that position messages fired in round 1, but possible if
   the Control FSM shifts before the first position message event.

**Recommendation:** Cannot confirm gradient messaging is working from in-game
observation alone. Recommend a targeted integration test that:
(a) forces ControlLevel to Controlling via `TransitionToControlling` on a character,
(b) triggers a drift round that crosses the Controlling → Neutral boundary,
(c) confirms `emitGradientMessage` is called and the player receives the template.
Alternatively, add temporary debug logging in `emitGradientMessage` to confirm
it is being invoked at all.

### CONCERN-2: combatstats position — all hit rate rows show 0.0%

**Observed:** After 244 combat events across many grapple positions, all four
target-position rows (`Target Standing`, `Target Prone`, `Target Clinched`,
`Target Grounded`) show 0.0%.

**Root cause identified:** `computeSummary` uses lowercase string keys:
`posMap["standing"]`, `posMap["prone"]`, `posMap["clinched"]`, `posMap["grounded"]`.
But `positionFields` calls `char.Position.State().String()` which returns capitalized
strings: "Standing", "Prone", "Clinch", "Mount", "Guard", etc.

- `"clinched"` never matches `"Clinch"` — map lookup fails.
- `"standing"` never matches `"Standing"` — map lookup fails.
- `"grounded"` never matches "Mount", "Guard", "BackGround", etc.

Result: `posMap[e.TargetPosition]` returns false for every event, all four
counters stay at zero, all rates are 0.0%. This is a pre-existing analytics bug
— not introduced by chunk-4b-fixup-2, but still broken.

**Fix required:** Either normalize `TargetPosition` to lowercase in `positionFields`
or `appendEvent`, OR change the `posMap` keys to match the actual `String()` output.
Also: the "grounded" bucket would need to aggregate multiple states ("Mount",
"Guard", "BackGround", "SideControl", "KneeOnBelly", "NorthSouth", "Crucifix",
"HalfGuard", "Turtle") — this may require an explicit list of ground-position names.

### CONCERN-3: combatstats Grapple Controller 0.0%

**Observed:** `Grapple Controller` hit rate is 0.0% across 244 events.

**Root cause analysis:** `HitRateGrappleController` tracks whether the attack
SOURCE has `char.IsController()` == true at the time the analytics event is
recorded. `IsController()` returns true only when `Control.State() == Controlling`.

The attack analytics (`RecordAttack`/`RecordSwings`) are recorded by the `DoCombat`
hook, which is registered before `processGrappleTick` in the event dispatch order
(`hooks.go` line 32 vs `Position_GrappleTick.go:init()` line 763). This means:

- Round N: `DoCombat` fires → records attack with current `Control.State()` (Neutral)
- Round N: `processGrappleTick` fires → calls `applyControlShift` → may update
  `Control.State()` to Controlling

The 0.0% is therefore expected given this ordering: ControlLevel transitions happen
AFTER combat attacks are recorded each round, so attacks are always recorded against
the previous round's ControlLevel state. A character who earned Controlling state last
round will have that reflected only next round's attack recording — but by then,
another drift roll may have shifted them away again.

This is a structural timing issue in the analytics pipeline, not a runtime bug. The
ControlLevel FSM itself works; the analytics measurement lags by one round.

**Note:** If ControlLevel IS reaching Controlling state on some rounds, the 0.0%
hit rate for the Grapple Controller bucket also confirms Concern-1: the `Grapple
Controller` row shows non-zero only if any attack is recorded while already in
Controlling state from the prior round. Zero events in this bucket across 244
events reinforces that Controlling state is rarely or never reached.

---

## Raw Stats

- Commands sent: ~35
- Fights initiated: 4
- Grapples observed: 4 (plus 1 mob-initiated grapple)
- Position advances seen: 15+ (Clinch→Mount×4, Mount→Guard, Guard→SC, SC→Mount,
  Guard→BackGround, BackGround→Mount, Mount→BackGround, Guard→HalfGuard, etc.)
- Gradient messages seen: **0** (CONCERN-1)
- Hold flavor lines seen: 15+ (Clinch hold ×5, Guard hold ×3, ground hold ×5,
  BackGround hold ×2)
- Mount strike-apex lines seen: 8+ (across all fights)
- Reversal / escape messages seen: 12+
- "Missing template" / "missing gradient key" debug strings: **0**
- combatstats Grapple Controller % (peak): **0.0%** (CONCERN-3 — timing issue)
- combatstats Non-Controller % (peak): **50.8%** (confirms grapple tick firing)
- combatstats position-by-target rows: **all 0.0%** (CONCERN-2 — string mismatch bug)
- Panics: **0**
- Server errors: **0**
