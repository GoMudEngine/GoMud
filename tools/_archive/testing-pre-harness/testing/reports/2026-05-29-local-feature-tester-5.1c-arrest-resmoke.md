# Feature Test Report: Town Justice 5.1c Arrest Re-Smoke (Post-Fix)

**Date:** 2026-05-29
**Session type:** local feature-tester (re-smoke)
**Goal:** Verify 4 specific bug fixes from the prior 5.1c arrest smoke
**Server:** localhost:55555 (AIPort)
**Character:** smoketester (AI-flagged)
**Zone tested:** Thornwall City -- Guard Barracks, Holding Cell, Gate Ward, Main Street, Market Square

---

## Session Summary

Re-smoke after 4 bugs were fixed from the prior 5.1c arrest smoke. The session covered the
full arrest->jail->payfine->release loop, the resist path, and all three help topics. The
character spawned already in the holding cell (leftover from a prior session), was released,
then committed fresh crimes to test the full cycle again.

Commands sent (approx): ~85

---

## Goal Results

### STEP 1 -- Help files and `set arrest`

PASS. All three help topics exist and read cleanly:
- `help arrest` -- covers surrender/resist distinction, usage with `set arrest [surrender|resist]`
- `help justice` -- explains wanted/guard/arrest/redemption cycle
- `help fine` -- explains decaying fine and `payfine`

Both `set arrest surrender` and `set arrest resist` confirmed clean:
- "Arrest policy set to surrender."
- "Arrest policy set to resist."

No raw numbers, no 3rd-person voice issues, no typos observed.

### STEP 2 -- `fine` command while jailed (decaying amount)

PASS. While jailed, `fine` showed a decaying amount:
- First check: 265 gold
- After ~15 seconds: 250 gold
- After ~30 seconds: 210 gold

Display text: "Your fine to walk free now is 265 gold. It drops the longer you sit. Pay it with payfine."

### STEP 3 -- Cell confinement (`go up`, other directions)

PASS. While jailed:
- `go up` -> "You can't do that!" (blocked)
- Other directions -> "You're bumping into walls."
No escape possible from cell while jailed.

### STEP 4 -- `flee` from cell while in combat (BUG-02 test)

PASS -- BUG-02 CONFIRMED FIXED. While in combat inside the holding cell, `flee` returned:

  "You're locked in -- there's nowhere to flee to."

Previously flee was incorrectly allowed. Now correctly blocked with the appropriate message.

### STEP 5 -- `payfine` release message count (BUG-01 test)

LIKELY STILL BROKEN. When `payfine` was executed with 200g, the output was:

  "You count out 200 gold. The cell door opens."
  [prompt]
  "The cell door swings open. You are free to go."

Two distinct door-related release phrases appeared in immediate succession. Whether "The cell
door opens." is intentional padding in the payfine command response and "The cell door swings
open. You are free to go." is the canonical release event requires code review to confirm.
From the player perspective, two door-opening messages appear -- consistent with BUG-01.

### STEP 6 -- Guards enter cell and attack jailed player (BUG-04 test)

STILL BROKEN. After being arrested (surrender policy active), two city guards followed the
player into the holding cell and attacked:

  "A guard seizes you and hauls you to the holding cell. You have been placed under
  arrest by the thornwall_guards."
  >>> city guard enters from the up.
  >>> city guard enters from the up.
  City guard bellows a thunderous challenge at smoketester!
  [combat initiates inside cell]

Room `look` confirmed: "Also here: City Guard #1 (72%) and City Guard #2 (100%)"

Both guards actively attacked. Player was forced to fight and killed both guards inside the
cell. The expected behavior (guards deposit prisoner and leave) is not happening.

### STEP 7 -- Re-arrest after release (BUG-03 test)

CONFIRMED FIXED. After paying the fine with `payfine`, the player walked through Thornwall
for 20+ rounds:
- Gate Ward -- no arrest
- Main Street West / Central -- no arrest
- Market Square -- City Guard present, did NOT arrest or attack
- Guard Barracks -- no guards present, no arrest

Guards left the released player alone. BUG-03 is fixed.

Nuance: After NEW crimes were committed later (killing guards in a subsequent resist-path
fight), Velk respawned and issued an arrest declaration before combat -- this is new-crime
behavior, not re-arrest for the cleared original crime. The record-clearing on payfine works.

### STEP 8 -- Resist path: guards fight for real

PASS. With resist arrest policy, guard combat fires with real mechanics:
- Guards used warcry (conviction damage)
- Guards used shield bash, longsword strikes
- Town NPCs and additional guards reinforced (faction backup working)
- Guard Captain Velk and city guards dealt real damage
- No crash, no stuck state

The resist path is working. Guards are legitimately dangerous.

---

## Findings

### BUG (STILL BROKEN) -- BUG-04: Guards enter holding cell and attack

**Severity:** High
**Repro:** Commit crime with surrender arrest policy -> get arrested -> observe.

Exact output observed:
```
A guard seizes you and hauls you to the holding cell. You have been placed under
arrest by the thornwall_guards.
[prompt]:
>>> city guard enters from the up.
>>> city guard enters from the up.
City guard's thunderous challenge rattles your nerve! (a crushing blow to their confidence)
City guard bellows a thunderous challenge at smoketester!
```

`look` after arrest:
```
.: [*] Holding Cell [Thornwall City]
...
Also here: City Guard #1 (72%) and City Guard #2 (100%)
```

Both guards engaged in active combat. Player killed both guards in the cell. The fix to
prevent guards from entering the cell after depositing a prisoner is NOT working.

### BUG (LIKELY STILL BROKEN) -- BUG-01: Double release message on payfine

**Severity:** Low-Medium
**Repro:** Pay fine with `payfine` while jailed.

Observed:
```
You count out 200 gold. The cell door opens.
[prompt]:The cell door swings open. You are free to go.
```

Two door messages in immediate sequence. Could be intentional split (payment confirmation +
release event) but reads as a duplicate to the player. Matches the prior BUG-01 pattern.

### CONCERN -- Velk respawn gear pickup delays arrest declaration

On respawn, Guard Captain Velk picked up equipment from the floor before initiating the
arrest sequence. When the player fled to Gate Ward, Velk followed and THEN issued the arrest
declaration "Move along is past -- you're under arrest. Come quietly." before "prepares to
fight you." -- correct order, but delayed by the gear pickup. Minor timing issue, not a
functional break.

### OBSERVATION -- `fine` while NOT jailed produces no output

Typing `fine` when not in jail produces silence. A "You have no outstanding fines" message
would improve UX.

### OBSERVATION -- Caravan picks up dead guards' weapons from floor

Ketil's caravan (Lars, Bran, Marta, Hob) passed through Gate Ward and picked up weapons from
dead guards on the ground. Lars wielded an iron short sword, Bran took a steel buckler. Item
pickup from crime-scene loot is functionally harmless but visually odd -- caravan members
arming themselves with guard gear.

### PASS -- help arrest / help justice / help fine all exist and read correctly
### PASS -- `set arrest surrender` and `set arrest resist` both work clean
### PASS -- BUG-02: `flee` from cell returns "You're locked in -- there's nowhere to flee to."
### PASS -- BUG-03: No re-arrest after `payfine` (walked past city guard freely in Market Square)
### PASS -- `fine` shows decaying amount while jailed (265 -> 250 -> 210 over ~30s)
### PASS -- Cell confinement blocks all exits while jailed
### PASS -- Resist path produces real combat, guards are dangerous
### PASS -- Arrest declaration fires before combat in resist mode
### PASS -- No crashes, panics, or disconnects throughout session

---

## Per-Bug Verdict Summary

| Bug    | Status               | Evidence                                           |
|--------|----------------------|----------------------------------------------------|
| BUG-01 | LIKELY STILL BROKEN  | Two "cell door" messages immediately on payfine    |
| BUG-02 | CONFIRMED FIXED      | flee in cell -> "You're locked in..."              |
| BUG-03 | CONFIRMED FIXED      | 20+ rounds post-payfine, city guard in room, no arrest |
| BUG-04 | STILL BROKEN         | Two guards entered cell and attacked immediately after arrest |

---

## Raw Stats

| Metric                          | Value              |
|---------------------------------|--------------------|
| Commands sent (approx)          | ~85                |
| Crashes / panics                | 0                  |
| Unexpected disconnects          | 0                  |
| Times arrested (surrender)      | 1                  |
| Times released via payfine      | 1 (200 gold)       |
| Guards killed in session        | 5+ (Velk x1, city guards x4) |
| Civilians killed                | 3 (thug x2, beggar x1)     |
| Civilian NPCs turning hostile   | 5+ (aliveness revenge goals working) |
| Session end                     | Bridge terminated (quit command sent) |
