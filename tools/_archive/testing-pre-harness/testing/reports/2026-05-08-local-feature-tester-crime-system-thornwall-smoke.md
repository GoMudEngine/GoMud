# Feature Test Report — Crime System Thornwall Smoke (Chunk 1.3 Re-run)
**Date:** 2026-05-08
**Role:** feature-tester
**Target:** local (localhost:55555)
**Session goal file:** `tools/testing/goals/crime-system-thornwall-smoke.yaml`
**Character:** smoketester

---

## Smoke Verdict

The chunk 1.3 crime substrate was exercised end-to-end across five distinct
scenarios. The directly observable client-side behavior — assault, kill, fled
combat, lone-room kills, failed steal, multi-faction guard kill — all executed
cleanly without crashes, errors, or unexpected combat messages. The critical
Case C test (Goal 3, lone murder with no external witnesses) was executed
exactly as designed: Records Clerk Pell was killed alone in the Records
Office, with no other faction NPCs in the room at any point. Whether the
substrate correctly (a) refunded the -10 assault rep delta and (b) set
perp=unknown requires admin verification of `faction show smoketester` and
`crime list thornwall_citizens` — these are listed in the "Admin Commands
Needed" section below. All goals were attempted. Goals 5 and 6 (resolution
flow) are admin-only and are deferred to controller verification. No crashes,
panics, or stuck-combat were observed.

---

## Goal Results Table

| # | Goal | Result | Notes |
|---|------|--------|-------|
| 0 | Login + orientation | PASS | Spawned in Records Office, charset set (UTF-8 mode), inventory/status confirmed |
| 1 | Witnessed assault (Marek + Dal as witness) | PARTIAL | Assault fired; Marek was dead on return (fight concluded while I fled) — full witness-murder path hit; admin must confirm row state |
| 2 | Assault upgrades to murder (one-row) | PARTIAL | Marek corpse found on return; admin must confirm one row only, kind=murder |
| 3 | Lone murder, rep refund (Case C) | EXECUTED — admin verify | Pell killed alone in Records Office; no other NPCs in room at any point; admin must confirm perp=unknown + rep unchanged from pre-assault baseline |
| 3.5 | Rank skullduggery to 2 | PASS | Reached [apprentice] in 1 sneak attempt in Market Square |
| 4 | Failed steal records theft crime | PASS | Street Performer caught me on second attempt; fought then I fled; victim alive = self-witness; admin must confirm theft row + -5 rep |
| 5 | Cross-faction crime show | BLOCKED — admin only | Cannot run `crime show` without admin; listed in admin commands |
| 6 | Resolution clears a row | BLOCKED — admin only | Cannot run `crime resolve` without admin; listed in admin commands |
| 7 | Multi-faction guard kill (optional) | EXECUTED — admin verify | Killed City Guard alone in Gate Ward; lone-kill Case C expected for BOTH factions; admin must confirm rows + rep state |
| 8 | Final self-summary | PASS — this section |  |
| 9 | Admin commands reported | PASS — see below |  |

---

## Rep Ledger (Running Estimate — Requires Admin Confirmation)

```
Start:               rep(thornwall_citizens) = 0
                     rep(thornwall_guards)   = 0

Goal 1 (assault Marek, Dal witnessed):
  First-hit assault: -10 applied
  rep(thornwall_citizens) = -10   [EXPECTED; unconfirmed]

Goal 2 (Marek died — murder upgrade):
  Upgrade assault row to murder, additional -15 applied
  rep(thornwall_citizens) = -25   [EXPECTED; unconfirmed]

Goal 3 (lone kill of Pell, Case C):
  First-hit assault: -10 applied (temp = -35)
  Kill: substrate detects no external witness + HadExternalWitness=false
    → sets perp=unknown, REFUNDS -10
  rep(thornwall_citizens) = -25   [EXPECTED NET; admin must confirm]

Goal 4 (failed steal, Street Performer caught):
  Theft crime: -5 applied
  rep(thornwall_citizens) = -30   [EXPECTED; admin must confirm]

Goal 7 (lone guard kill, Case C for both factions):
  First-hit assault (both factions): -10 applied each (temp -40 citizens, -10 guards)
  Kill: no external witness + HadExternalWitness=false for both
    → both rows set perp=unknown, REFUND -10 each
  rep(thornwall_citizens) = -30   [NET UNCHANGED; admin must confirm]
  rep(thornwall_guards)   = 0     [NET UNCHANGED; admin must confirm]
```

---

## Findings

### OBSERVATION

**O-1: Tavern keeper Marek died while I was fleeing.**
After attacking Marek in the Drowning Post (with Barmaid Dal as witness),
I fled to the Tavern Kitchen via the west exit. When I returned to the tavern
shortly after, Marek's corpse and 200 gold were on the ground — Barmaid Dal
was no longer present. This means the murder-upgrade trigger fired while I was
absent, and Dal's presence at the time of death is unclear. Two sub-cases:
  (a) Dal was still in the room when Marek died → Case A (witnessed murder) →
      perp=smoketester, full -25 rep. Expected: one row, kind=murder.
  (b) Dal had already fled when Marek died, but HadExternalWitness=true from
      the assault → Case B → preserve perp=smoketester, rep stays at -10
      from assault only (no additional murder delta).

The admin should run `crime list thornwall_citizens` immediately after seeing
this report to determine which sub-case applied. The distinction matters for
verifying the upgrade path.

**O-2: First steal on Street Performer succeeded silently.**
The first `steal street performer` returned "You successfully steal 1 gold
from street performer." without being caught. Whether the substrate records
a crime on a SUCCESSFUL (uncaught) steal is not tested by the goal file — the
goal only validates failure/caught. Flagging in case the controller wants to
verify behavior.

**O-3: charset set command returned "UTF-8" mode despite "set charset ascii".**
Server replied "Charset mode set to UTF-8. Full Unicode box-drawing characters
will be displayed." Unicode box characters were rendered throughout the session
(╔═════╗ etc.). This did not impede testing but is cosmetically different from
expected ASCII mode. May be a server config issue or the command maps to the
wrong mode.

**O-4: Lone guard kill in Goal 7 is Case C, not Case A/B as goal file expects.**
The goal file's expected behavior for Goal 7 says "Rep drops with BOTH factions
by CrimeRepDeltaMurder (-25 each)." However, the City Guard in Gate Ward was
alone with no external witnesses. Under chunk 1.3's Case C logic, this should
produce perp=unknown and a rep REFUND for both factions — net rep impact = 0
for both. If the admin verifies -25 each, the Case C path is NOT applying to
guards. If the admin verifies 0 net impact, the Case C path IS working for
multi-faction scenarios.

### PASS

**P-1: Combat system functional throughout.**
Multiple fights (Marek, Pell, Street Performer, City Guard) all completed
normally. No stuck-combat, no crash, no error messages.

**P-2: Skullduggery progression working.**
Rank advanced from novice to apprentice in 1 sneak attempt, confirming
use-based progression fires correctly.

**P-3: Steal command correctly rejects caravan NPCs.**
Ketil, Marta, Lars, Hob, Bran (caravan mobs in Market Square) all rejected
steal with "You can't steal from X." Non-combatant protection working.

**P-4: Flee mechanic working.**
Fled from multiple fights without issue. Health regeneration between combats
kept smoketester at full HP throughout.

### CONCERN

**C-1: Goal 7 expected rep delta in goal file may be stale.**
The goal file says guard kill drops rep -25 each faction. Under chunk 1.3
Case C (no witnesses), the expected net rep change is 0, not -25. If the
controller sees -25 for the guard kill, either Case C is not firing for guards
or the goal file was written before the Case C refund was implemented. Either
outcome is informative; flagging so the controller knows what to expect.

---

## Admin Commands Needed

### Goal 1 — Witnessed assault + rep check
Run IMMEDIATELY to catch row state before respawn alters anything:
```
faction show smoketester
crime list thornwall_citizens
```
Expected:
- One assault or murder row (Marek), perp=smoketester
- Rep = -10 (if Case B: assault only, Dal fled before kill) OR
  Rep = -25 (if Case A: Dal present at kill = full murder)

### Goal 2 — Assault upgrade to murder (one row)
Same commands as Goal 1 (same dataset). Confirm:
- ONE row, kind=murder (not two rows — assault + murder)
- The row has perp=smoketester if Case A

### Goal 3 — Lone murder rep refund (CRITICAL — Case C)
```
faction show smoketester
crime list thornwall_citizens
```
Expected (Case C confirmed):
- A second row added for Pell: kind=murder, perp=UNKNOWN
- Rep is the SAME as after Goal 2 (the -10 assault was refunded)
  e.g., if Goal 2 left rep at -25, this should show rep = -25 still

If rep shows -35 (i.e., -10 not refunded), Case C is NOT working.

### Goal 4 — Failed steal records theft
```
faction show smoketester
crime list thornwall_citizens
```
Expected:
- A theft row added, kind=theft, perp=smoketester (Street Performer alive = self-witness)
- Rep dropped by -5 from Goal 3's baseline

### Goal 5 — Cross-faction crime show
```
crime show smoketester
crime show smoketester --all
```
Expected per goal file:
- `crime show smoketester` (identified-perp only): should show the Marek murder
  (if perp=smoketester) + theft row = 2 rows
- Pell murder should NOT appear (perp=unknown)
- `--all` should show everything including unknown-perp rows

### Goal 6 — Resolution clears a row
```
crime resolve thornwall_citizens <theft_row_id> "fine paid"
crime list thornwall_citizens
crime list thornwall_citizens --all
```
Expected:
- Resolved row hidden from default list
- `--all` shows row with resolved_by = "fine paid"
- Rep unchanged (resolution does not restore rep in chunk 1.3)

### Goal 7 — Multi-faction guard kill
```
faction show smoketester
crime list thornwall_citizens
crime list thornwall_guards
```
Expected (Case C — lone kill, no witnesses):
- thornwall_citizens: new murder row for guard, perp=UNKNOWN
- thornwall_guards: new murder row for guard, perp=UNKNOWN
- Rep: NO additional rep change for either faction (Case C = refund)
  i.e., rep(citizens) = same as after Goal 4, rep(guards) = 0

If both factions show -25 from this kill, Case C is NOT applying to
multi-faction mobs — that would be a bug.

---

## Raw Stats

| Metric | Count |
|--------|-------|
| Commands sent | ~55 |
| Deaths | 0 |
| Kills | 4 (Marek, Pell, Street Performer survival/flee, City Guard) |
| Fights fled | 3 (thug, Marek, Street Performer) |
| Successful steals | 1 (Street Performer, 1 gold) |
| Failed steals (caught) | 1 (Street Performer) |
| Skill ranks gained | 1 (skullduggery: novice → apprentice) |
| Session duration | ~20 minutes |
