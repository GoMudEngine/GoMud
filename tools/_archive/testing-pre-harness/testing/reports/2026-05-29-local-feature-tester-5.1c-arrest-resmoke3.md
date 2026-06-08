# Test Report: Town Justice 5.1c Arrest Re-Smoke #3

**Date:** 2026-05-29
**Tester:** AI (smoketester / feature-tester role)
**Server:** Local, fresh build with BUG-04 fix applied (AIPort 55555)
**Goal file:** tools/testing/goals/5.1c-arrest.yaml
**Commands sent:** ~90

---

## Executive Summary

| Bug | Verdict | Evidence |
|-----|---------|----------|
| BUG-04: Guards enter cell and attack jailed player | **STILL BROKEN** | Guard captain Velk entered cell and declared "prepares to fight you!" on 8+ separate occasions across ~15 rounds of idle. Pattern was perfectly consistent. |
| BUG-01: Duplicate "cell door" line on payfine | **CONFIRMED FIXED** | payfine produced exactly one payment line and exactly one door line with no duplication. |
| BUG-02: Flee blocked in cell | CONFIRMED FIXED (consistent with prior resmoke) | "You're locked in -- there's nowhere to flee to." |
| BUG-03: No re-arrest after release | PASS (with caveat) | City guards did not attack after release. The guard captain who was in active combat pre-arrest continued combat post-release (pre-existing combat session, not a re-arrest). |
| Resist path: guards fight for real | PASS | Attacking a city guard with `set arrest resist` engaged real combat immediately -- no arrest-announcement phase, just full combat rounds. |

---

## Detailed Findings

### BUG-04: Guard Captain Enters Cell and Attacks — STILL BROKEN

**Exact sequence observed (repeated 8+ times over ~15 rounds):**

```
>>> guard captain Velk enters from the up.
Guard captain Velk prepares to fight you!
[player idle, no commands]
>>> guard captain Velk leaves towards the up exit.
```

**Timing:** Every 2-3 idle rounds, the captain would descend into the cell, declare
"prepares to fight you!", then leave without delivering actual combat swings.

**No damage was dealt** during these cell intrusions — the captain declared intent but
did not sustain combat. However, the declaration itself is the bug: the guard is
entering the holding cell and recognizing the player as a combat target.

**Observed 8+ distinct enter-declare-leave cycles** before the fine dropped enough
to payfine. The CP bar shows conviction damage from warcry (which the captain used
during prior barracks combat), but no HP damage from the cell intrusions themselves.

One idle flavor message also fired during a cell intrusion:
```
guard captain Velk drums fingers on the table, studying an incident report
```
This suggests the combat was being suppressed by some mechanism, but the aggro
detection that causes the enter-and-declare loop is not suppressed.

**Root cause hypothesis (for developer):** The fix that makes jailed players
"invisible to mob aggro" may not be covering the `prepares to fight` / aggro-declare
path at room-entry time. The captain's aggro fires on room entry, declares combat,
but something (jailed flag?) prevents actual swings. The fix needs to suppress the
aggro check at entry, not just the swing authorization.

---

### BUG-01: payfine Door Message — CONFIRMED FIXED

When the fine reached 30g (within the 57g bank balance), `payfine` produced:

```
You count out 30 gold and settle your fine with the guards.
The cell door swings open. You are free to go.
```

This is exactly one payment line followed by exactly one door line. **No duplicate.**
BUG-01 is fixed.

---

### BUG-02: Flee Blocked in Cell — CONFIRMED FIXED

Observed exact output:
```
flee
You're locked in -- there's nowhere to flee to.
```

Movement (`up`, `go up`) also blocked:
```
You can't do that!
```

BUG-02 remains fixed.

---

### BUG-03: No Re-Arrest After Release

After payfine released me from the cell, I walked to Gate Ward and Main Street.
City guards in those rooms did NOT attack or arrest me. Observed:

```
city guard nods to a passing merchant.
```

The guard captain was still in active combat with me (from the pre-arrest barracks
fight) and resumed attacking after release — but this is a pre-existing combat
session, not a new arrest trigger. This is expected behavior: payfine clears the
crime/wanted flag but does not end combat already in progress.

**BUG-03 PASS** — fresh guards not in prior combat leave the released player alone.

---

### `set arrest` Commands

Both policies confirmed working:

```
set arrest resist
→ Arrest policy set to resist.

set arrest surrender
→ Arrest policy set to surrender.

set arrest   (show only)
→ Your arrest policy: surrender.
   Set with: set arrest <surrender|resist>.
   See help arrest for details.
```

---

### Resist Path: Guards Fight for Real — PASS

With `set arrest resist`, attacking a city guard in Main Street West triggered
immediate full combat:

```
You prepare to enter into mortal combat with City Guard.
[combat swings begin immediately]
City guard prepares to fight you!
[real combat rounds follow, guard swings back]
```

No arrest-announcement phase. Guards pursued across rooms. Real damage exchanged.
Guard used warcry, blocked attacks, swung back. Combat functioned correctly.

---

### Help Files — All Present, Read Sensibly

All four help files verified:

- `help arrest` — clear, first-person voice, describes surrender/resist paths, correct `set arrest` usage
- `help justice` — explains the crime/bounty/redemption cycle, good prose
- `help fine` — brief, explains decay mechanic, points to payfine
- `help payfine` — explains gold-then-bank draw order, points to fine

No typos, no 3rd-person NPC voice issues, no raw numbers leaked.

---

### `fine` Command

Showed decaying amount each round. Confirmed decay rate: ~5 gold per round.

Sample sequence:
```
fine → 505 gold
fine → 430 gold  (after ~15 rounds)
fine → 285 gold  (after ~30 rounds)
fine → 95 gold   (after ~50 rounds)
fine → 60 gold   (after ~55 rounds)
fine → 40 gold   (after ~57 rounds)
fine → 30 gold   (paid via payfine)
```

Decay is smooth and predictable. `payfine` correctly pulled from bank
(had 0 gold on person, 57 in bank, paid 30g).

---

## BUGS Found

### BUG-04 (Primary Regression — Already Known)
**Guard captain enters holding cell and declares combat on jailed player.**
Described above. The fix is incomplete — aggro suppression must cover room-entry
detection, not just swing authorization.

### NEW BUG: Format String Leak in Combat Message

During post-release combat with the guard captain, this line appeared:

```
guard captain Velk attempts a %s, but misses!
```

The `%s` format placeholder was not replaced. This is a pre-existing bug in a
combat message template, not introduced by the 5.1c work. Observed once.

---

## CONCERNS

### Captain Continues Combat Post-Release
After payfine, the guard captain (who was in active combat from before arrest)
immediately re-engaged. Players will likely find it jarring to be released and
immediately attacked. Design question: should payfine also clear the captain's
combat target? This is not a bug per the current spec but feels bad UX.

### Arrest Without Arrest-Declaration Observed
In the very first session, I was arrested and hauled to the cell without observing
the full declare → grace → haul sequence. This may be because I was already in
active combat with the captain when the arrest fired — the grace period may be
skipped when combat is already underway. Worth confirming this is intended.

---

## Test Coverage

| Goal | Result |
|------|--------|
| `set arrest` / policies work | PASS |
| help arrest / justice / fine / payfine | PASS |
| Surrender path: arrested and hauled to cell | PASS |
| Cell: flee blocked | PASS |
| Cell: go up / movement blocked | PASS |
| Cell: `fine` shows decaying amount | PASS |
| Cell: `payfine` door message clean (BUG-01) | CONFIRMED FIXED |
| Cell: guards leave you alone in cell (BUG-04) | STILL BROKEN |
| Post-release: no re-arrest from fresh guards | PASS |
| Resist path: guards fight for real | PASS |
| Server stable throughout (no crash/panic/disconnect) | PASS |

---

*Report generated by AI feature-tester session, 2026-05-29.*
*Server was NOT stopped after testing (per SOP).*
