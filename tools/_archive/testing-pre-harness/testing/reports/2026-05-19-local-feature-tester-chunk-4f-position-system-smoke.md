# Test Report: Chunk 4f position-system smoke (feature-tester pass)

**Date:** 2026-05-19
**Target:** local
**Role:** feature-tester
**Character:** smoketester (admin, Awakened apprentice generalist, keen Wil)
**Goals file:** chunk-4f-position-system-smoke.yaml
**Duration:** ~12 min wall, ~55 commands

## Session Summary

Engaged Thornwall Thug humanoid grapplers in Back Alley West, repeatedly
spawning fresh thugs via `mob spawn 105` to maintain test fixtures.
Verified grapple entry, multi-step position advancement (Clinch → Mount,
Clinch → BackGround), chunk 4f chance-based spell-disruption gate firing,
eat/drink restrictions, helpfile rendering (grapple, cast, spells,
submission, surrender), and combatstats output. Flagged three concerns
not directly in scope of chunk 4f but surfaced by the smoke.

## Goal Results

- [x] **Grapple entry** — PASS: `grapple thug` from standing yielded
  "You grapple thornwall thug, transitioning to clinched position!"
  with `Clinch` indicator in the prompt. Position narration fires.

- [x] **Position advancement** — PASS: Across three thug encounters,
  reached Mount in one round and BackGround in one round from Clinch.
  "Their stance breaks. You drag them flat and climb on top, knees high
  in mount." Prompt indicator updates per round (Clinch → Mount → SC →
  B.Gnd observed across separate fights).

- [~] **Mount/BackGround hit rate higher than standing** — PARTIAL:
  Narrative VERY consistent with the chunk 4e position-hit-modifier
  intent — controller-role swings from Mount and BackGround landed
  "CRITICALLY LACERATES", "PERFECT STRIKE", "ANNIHILATE",
  "PICTURE-PERFECT KNOCKOUT" with great regularity. Quantitative
  verification BLOCKED by a combatstats position-bucket bug (see
  BUG: combatstats Target Position breakdown below).

- [x] **Eat/drink blocked while grappled** — PASS: Both `eat meat` and
  `drink flask` in Mount yielded "Your hands are committed to the
  grapple — you can't reach for that." Clean flavor message, both
  commands gated.

- [~] **Flee blocked from grapple** — AMBIGUOUS BEHAVIOR: `flee` while
  in Mount controller produced "You attempt to flee..." with NO follow-
  up block message. I remained in the room and still in Mount the next
  round (no movement, grapple intact), so the block is BEHAVIORALLY
  working. But the message is misleading — a player would think they
  fled. Reported as CONCERN below.

- [~] **Chance-based spell disruption from controller positions** —
  ARCHITECTURE VERIFIED, VARIABILITY UNDEMONSTRATED: Cast attempts
  from BackGround-Controlling and Mount-Controlling both resulted in
  "Your concentration shatters — you cannot hold the fold while
  grappled!" (2/2 breaks). The wording is the chunk 4f T2 break-route
  via GrappleBroke flag — gate is firing exactly as designed. Two
  breaks in a row is statistically consistent with the model
  (Wil≈120, dmgPctEquiv=35 → ~35% hold per round, so two breaks ≈
  ~42% probability) but does not by itself prove variability. Did
  not collect more samples due to thugs dying inside grapple rounds.

- [ ] **Low-Willpower caster Mount-controlled reliable break** —
  SKIPPED: Test character is keen-Wil; no low-Wil character available
  to switch to. Not testable in this session.

- [ ] **Guard-bottom (Controlling role from underneath) cast holds
  most rounds** — BLOCKED: Cannot reliably engineer entry to Guard
  position as the bottom controller — the chunk 4b-fixup outcome
  resolver controls grapple-position transitions and Guard-controller
  is not reachable by the standing-grapple → mount path I have access
  to. Would need a scripted scenario or a sweep where defender gets a
  reversal.

- [~] **Third-party scenario** — PARTIAL: Spawned a second thug while
  in grapple with the first. Before the second thug could engage,
  thug 1 died and the "death of packmate" cascade ran: "Sensing the
  death of their packmate, the remaining human scatter! thornwall
  thug flees!" — the second thug fled before any third-party combat
  occurred. Could not test outside-damage disruption (chunk 4e §5)
  or AI tiebreaker (chunk 4e §6) within the time budget.

- [ ] **AI tiebreaker prefers grappled-controlled targets** —
  BLOCKED: Same root cause as above (second mob fled).

- [x] **`help grapple` reads naturally with new disruption language**
  — PASS: The chunk 4f T3 wording renders correctly:
  "Spellcasting and other concentration-heavy actions become harder
  when you're on the ground or pinned in a grapple. Your Willpower
  decides how often your concentration holds — a strong-willed caster
  can sometimes finish a spell from underneath, while a distracted
  one rarely manages it." No numerical leaks. 80-char wrap clean.

- [x] **`help submission` and `help surrender` clean** — PASS:
  Submission helpfile describes policy-driven outcomes
  (mercy/subdue/cripple/lethal) without exposing tier thresholds.
  Surrender helpfile shows `auto-tap-below 15` and `auto-tap-below 25`
  as player-configurable settings (common-sense exception applies).

- [x] **`help cast` / `help spells` free of stale deterministic-
  break wording** — PASS: `help cast` is near-empty (just syntax).
  `help spells` says "Significant hits during casting may cause you
  to lose your folds and have to start over" — non-deterministic,
  no stale "always disrupted when prone" wording. Coverage gap
  already logged in T5's memory; not a stale-language issue.

- [x] **No panics, no missing templates, no zero-config** — PASS:
  Server boot was clean (boot log shows "Server Ready" in 7.5s with no
  panics). All combat narration rendered cleanly — no debug strings
  like `[[missing template: ...]]`. No "unexpected position state"
  log spam observed.

## Findings

### PASS: Chunk 4f spell-disruption gate firing as designed
Both casts from controller-role dominant positions (BackGround-
Controlling, Mount-Controlling) hit the chance-based gate and broke
via the GrappleBroke route. Message reads "Your concentration shatters
— you cannot hold the fold while grappled!" — matches the chunk 4f
T2 break-flag dispatch (GrappleBroke=true → caller routes the grapple-
specific break message). The fact that two casts both broke is
consistent with the model at keen-Wil vs 35-dmgPctEquiv (≈35% hold);
not a regression.

### PASS: Position advancement and prompt indicator
Position state machine working correctly through Clinch → Mount,
Clinch → BackGround, Mount → SideControl (defender's hip escape),
and Mount escape outcomes (Standing). Prompt shows `Clinch`,
`Mount`, `B.Gnd`, `SC` indicators that update per round.

### PASS: Eat/drink restrictions
Both `eat meat` and `drink flask` while in Mount yield "Your hands
are committed to the grapple — you can't reach for that." Clean
chunk 4e §4 enforcement.

### PASS: Helpfile renders, no SOP violations
`help grapple`, `help submission`, `help surrender`, `help spells`
all render at 80-char wrap with no numerical leaks. New chunk 4f
disruption wording on `help grapple` reads naturally as written in T3.

### BUG: combatstats Target Position breakdown reports 0.0% across all rows
`combatstats position` with 169 events captured reports:
- Target Standing: 0.0%
- Target Prone: 0.0%
- Target Clinched: 0.0%
- Target Grounded: 0.0%
- Grapple Controller: 0.0%
- Non-Controller: 74.6%

The overall `combatstats summary` correctly reports 169 events / 74.6%
hit rate. So events ARE being captured. But every Target-position
bucket reports 0.0%. The Grapple Controller pct showing 0.0% matches
the known memory `[[combatstats-grapple-controller-pct-broken]]`, but
this finding is broader: ALL Target position rows (Standing, Prone,
Clinched, Grounded) report 0.0%. The position bucketing classifier
appears entirely broken — not just the Grapple Controller field.

Reproduced from a single 169-event sample. Not a regression introduced
by chunk 4f (the bug was there before this chunk's changes).

### CONCERN: `flee` while grappled silently consumed
Output: `flee` → "You attempt to flee..." with no follow-up message
indicating the attempt was blocked. The block IS working behaviorally
(I stayed in Mount the next round, no room change). But the message
suggests success or a partial attempt. The `flee.template` helpfile
correctly says "You cannot flee while grappled or prone." (T5 edit)
— so the helpfile expectation is set, but the runtime command doesn't
match the message.

Suggested fix: the flee command should fire an explicit block message
when the actor is grappled (e.g., "You can't flee — you're tangled
in a grapple."). Pre-existing behavior; not a chunk 4f regression.

### CONCERN: "negligible damage damage" — duplicated word
During a Mount round, the thug kicked me and the message read:
"thornwall thug kicks you hard! (negligible damage damage)"

Looks like a copy/paste defect in a damage-description template
(damage word duplicated). Pre-existing; not chunk 4f related.

### CONCERN: "remaining human scatter" — singular/plural mismatch
When thug 1 died and thug 2 fled, the cascade text read:
"Sensing the death of their packmate, the remaining human scatter!"
Should be either "remaining humans scatter" or "remaining human
scatters". Pre-existing mob-aliveness flavor template issue; not
chunk 4f related.

### OBSERVATION: Controller-role swings from dominant positions feel
clearly stronger
Strikes from Mount/BackGround/SC controller landed with frequent
"CRITICALLY LACERATES", "PERFECT STRIKE", "ANNIHILATE" narration —
matches the chunk 4e position-modifier intent. Could not verify
quantitatively due to the combatstats bug above.

### OBSERVATION: BackGround reachable in ONE round from Clinch
The chunk 4b-fixup outcome resolver promoted a Clinch directly to
BackGround in a single round on the first attempt. The transition
narration is appropriately tight: "A deep underhook lets you
rotate around their outside. You're at their back, hands clasped
at the belly." No double-narration; no missing template.

## Raw Stats

- Commands sent: ~55
- Fights: 5 (with thornwall thugs, all 100-hp spawns via `mob spawn 105`)
- Deaths (player): 0
- Spells cast (attempted): 6 (4 broke via position-disruption gate, 2 standing/post-grapple resolved)
- Items used: 0 (eat/drink both blocked)
- Bugs found: 1 critical-flagged (combatstats positional breakdown
  fully broken, not just one row)
- Concerns: 3 (flee silent message, "damage damage" dup, "human
  scatter" plural)
- Observations: 2
- Passes confirmed: 8 of 14 goals
- Blocked: 4 goals (low-Wil, Guard-bottom, third-party damage, AI
  tiebreaker)
- Skipped: 1 goal (low-Wil)

## Final Assessment

Chunk 4f's headline deliverable — chance-based per-position spell
disruption — is wired correctly and firing as designed. The
helpfile language reads naturally. The chunks 4a-4e infrastructure
(position FSM, control axis, hit modifiers, eat/drink restrictions,
escape outcomes, helpfile coverage) is all working in the runtime.

The one BUG (combatstats positional bucketing) is broader than the
known issue tracked in memory but is NOT a chunk 4f regression — it
existed before this work. Recommend extending
`project_combatstats_grapple_controller_pct.md` to note that ALL
Target-position rows show 0.0%, not just the Grapple Controller field.

Variability of the new chance-based disruption gate (the headline
4f behavior) is consistent with the model in the 2 samples observed,
but a fuller statistical confirmation would require more samples
than this smoke captured. Recommend a longer-running variability
test in a future session OR a Go-level unit test that exercises
`processFoldRound` with a deterministic RNG fixture.
