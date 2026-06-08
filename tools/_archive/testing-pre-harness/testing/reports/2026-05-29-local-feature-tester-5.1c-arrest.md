# Session Report: Town Justice 5.1c — Arrest / Jail / Fine + Combat-Capable Guards

**Date:** 2026-05-29  
**Branch:** feature/town-justice-5.1c (local)  
**Tester:** smoketester (AI)  
**Server:** localhost:55555  
**Session duration:** ~55 minutes  
**Commands sent:** ~75

---

## Session Summary

Tested the full Town Justice 5.1c arrest/jail/fine system and combat-capable-guards
fix in Thornwall City. The surrender→arrest→jail→fine decay→payfine→release loop
ran, and the resist path produced real combat. However, several significant bugs were
found: guards enter and fight the player inside the holding cell, `flee` from
in-cell combat bypasses the jail lockdown, and there is a double-release message on
`payfine`. The overall structure of the system is present and working at a high level,
but the cell's security boundary is broken.

**Note on session state:** smoketester had a pre-existing wanted status from a prior
session. This meant the first arrest happened automatically (without deliberate
crime-commission) when entering Market Square. Testing was adapted to work around
this, but it complicated observing a clean declare → grace → haul sequence.

---

## Goal Results

| Step | Status | Notes |
|------|--------|-------|
| Check arrest policy (`set arrest`) | PASS | Defaults to surrender correctly |
| `set arrest resist` / `set arrest surrender` | PASS | Both set cleanly with confirmation message |
| Read `help arrest` / `help justice` / `help fine` | PASS | All exist, readable, well-written |
| Become wanted, let guard arrest (surrender path) | PARTIAL | Arrest triggers correctly; declare message seen but grace window not clearly observable |
| Jailed: confirm can't walk out / recall blocked | PARTIAL | `up` blocked ("You can't do that!"), `recall` not a command in this build. But FLEE from in-cell combat bypasses the lockdown — see BUG-02 |
| `fine` command shows and decays | PASS | Works correctly; decay confirmed multiple observations |
| `payfine` releases player | PARTIAL | Releases but double-message bug (BUG-01); immediate re-arrest on first attempt (BUG-03) |
| Release clears wanted status | PARTIAL | Appears to work eventually; confirmed clean exit from Barracks without re-arrest on a later payfine; unreliable due to BUG-03 |
| Resist path: guard fights for real | PASS | Guard Captain Velk engaged in genuine combat with full rounds, hit/miss messages, HP progression; combat-capable-guards fix confirmed working |
| Any crash / disconnect / panic | PASS | No crash or panic observed throughout session |

---

## Findings

### BUG-01 — `payfine` prints release message twice

**Severity:** Medium (cosmetic but confusing)

Every call to `payfine` prints the release message twice in sequence:

```
The cell door swings open. You are free to go.
You count out 415 gold. The cell door opens.
The cell door swings open. You are free to go.
```

The first line "The cell door swings open. You are free to go." appears, then the
gold-deduction line, then the message appears again. Reproduced on every payfine call
across the entire session (4+ times). Likely a double-send in the payfine handler or
the release trigger fires twice.

---

### BUG-02 — `flee` from in-cell combat bypasses jail lockdown (jailbreak via flee)

**Severity:** High (gameplay integrity)

When guards enter the holding cell and engage the player in combat (see BUG-04),
the player can use `flee` to escape combat and land in the Guard Barracks above.
The `up` exit from the cell is normally locked ("You can't do that!") but is NOT
locked during a flee action.

Reproduction sequence:
1. Be jailed in the Holding Cell.
2. Wait for a guard to enter the cell and start combat (see BUG-04).
3. Type `flee`.
4. Player successfully flees to the Guard Barracks.

The player is then re-arrested immediately when they try to go south from the
Barracks (that exit IS locked). But the interim state — standing free in the
Barracks with a corpse on the floor and gear to loot — is incorrect. The jail's
lockdown should block flee-based exits as well as voluntary movement.

---

### BUG-03 — `payfine` does not reliably clear wanted status; immediate re-arrest

**Severity:** High (loop-breaking)

After paying the fine and being released to the Guard Barracks, Guard Captain Velk
(if alive) or patrol guards immediately declared arrest again:

```
The cell door swings open. You are free to go.
guard captain Velk says, "Move along is past — you're under arrest. Come quietly."
A guard seizes you and hauls you to the holding cell.
```

This happened on the first two `payfine` attempts. On the third attempt (after Velk
had been killed in resist combat), the release was clean — no immediate re-arrest
occurred and the player was able to walk through the city without guards engaging.

Hypothesis: the re-arrest may be caused by one of the following:
- The payfine handler clears the bounty but a guard already had the arrest goal
  "locked in" (seeded before the payfine resolved), and that in-flight goal fires
  before the cleared status propagates.
- There is a race between the payfine clearing the wanted flag and the guard's
  per-tick arrest-check re-evaluating.
- The initial session had TWO separate crime entries stacked, and payfine only
  cleared the most recent one.

Either way, the expected behavior (paying fine → clean release → guards treat
player as neutral) did not work consistently.

---

### BUG-04 — Guards enter the holding cell and engage jailed player in combat

**Severity:** High (gameplay integrity)

After the player was placed in the holding cell, patrol guards from the Guard
Barracks entered the cell (via the "up/down" stairs exit) and engaged in combat.
The guards announced arrest and then fought the jailed player:

```
city guard says, "Move along is past — you're under arrest. Come quietly."
City guard prepares to fight you!
[HP/SP/CP prompt] » city guard[|healthy] :
...full combat rounds firing...
```

This should not happen. A jailed player is already in custody — guards should not
re-arrest or attack someone already serving a sentence. The cell should be a safe
holding space. The guards appear to be patrolling down into the cell room and
triggering their arrest behavior because the player's wanted flag is still set
(consistent with BUG-03 — wanted status not cleared by being jailed).

Two guards died in the holding cell during this session, leaving multiple corpses
and gear on the cell floor. No crash resulted, but the experience was chaotic.

---

### BUG-05 — Pre-existing wanted status on smoketester at session start

**Severity:** Low (test-harness / data hygiene, not a game bug per se)

When smoketester connected, guards in the Gate Ward immediately announced arrest:

```
city guard says, "Move along is past — you're under arrest. Come quietly."
city guard says, "Move along is past — you're under arrest. Come quietly."
```

smoketester was arrested before any deliberate crime was committed, when entering
Market Square. This is likely a stale crime flag from a prior session where
smoketester was never fully cleared. The CLAUDE.md SOP notes that instance saves
are nuked before smoke tests, but player YAML (where crime flags live) is presumably
persistent. Recommend resetting smoketester's faction reputation before future
arrest-system smoke tests.

---

### CONCERN-01 — `up` movement from the holding cell: direction unclear

The holding cell's only exit is `up`. When the player tries to go `up` while jailed,
they get "You can't do that!" which is the engine's generic blocked-exit message.
There is no custom message explaining WHY they cannot leave — e.g., "The cell door
is locked — you're serving your sentence." or "The bars hold firm." Compare to the
`look bars` response which is well-written. A specific "you're in jail" message on
the blocked `up` exit would significantly improve clarity.

---

### CONCERN-02 — Arrest sequence: grace window not perceptible

The `help arrest` file says "After a guard announces an arrest you have a few moments
to decide." In practice, the sequence appeared to be:

1. `city guard says, "Move along is past — you're under arrest. Come quietly."`
2. (no perceptible gap — next server tick)
3. `A guard seizes you and hauls you to the holding cell.`

The grace window may exist mechanically but is not perceptible to the player. If
the design intent is to let the player draw a weapon to choose resist, there needs
to be at least 1-2 rounds of delay between the announce and the haul.

---

### CONCERN-03 — `attack guard` entered combat state but no first-round damage message

When the player typed `attack guard` (surrender policy set), the output was:

```
You prepare to enter into mortal combat with City Guard.
[prompt] » city guard[|healthy] :
```

Then after two ticks, the arrest haul triggered with no combat damage on either side.
This is correct behavior for the surrender path (the combat is aborted by the arrest),
but the single-line "enter into mortal combat" message hanging without resolution may
confuse players. Consider a message like "The guard moves to restrain you before you
can act." to explain the abort.

---

### OBSERVATION-01 — Resist path: real combat confirmed working

Guard Captain Velk fought for real when the resist policy was active:

```
Guard captain Velk prepares to fight you!
...
Guard captain Velk's thunderous challenge rattles your nerve! (a rattling verbal assault)
Guard captain Velk bellows a thunderous challenge at smoketester!
[HP prompt] » guard captain Velk[|healthy] :
...multiple rounds of hits, parries, fumbles, special moves...
[HP prompt] » guard captain Velk[|near death] :
...
Guard captain Velk has died.
Iron short sword drops to the ground.
...
```

Full combat progression confirmed: HP states advanced (healthy → bruised → wounded
→ badly wounded → near death → dead), special moves fired (RIPOSTE, SWEEP, SHIELD
SLAM, thunderous challenge), and loot dropped on death. The combat-capable-guards
fix is working. Velk is a legitimate deterrent.

---

### OBSERVATION-02 — `fine` command: clean, working

`fine` works from within the cell:

```
Your fine to walk free now is 480 gold. It drops the longer you sit. Pay it with payfine.
```

Decay observed: 480 → 425 → 340 → 315 → 300 → 275 → 265 (paid) across the session.
Decay rate appears roughly 25-30 gold per few idle rounds. Command text is clear.

---

### OBSERVATION-03 — `look bars` and `look pallet` work in cell

Both room nouns respond with appropriate flavor text:

```
look bars:
Thick iron bars set into the stone, spaced too narrow to squeeze through and too
solid to bend. The lock is a heavy iron affair, recently oiled -- these cells
see regular use.

look pallet:
A thin pallet of straw in the corner, flattened by many previous occupants. It
is the only concession to comfort in the cell.
```

`look door` returns "Look at what???" — `door` is not a defined room noun. Minor
gap; `bars` is the better target anyway.

---

### OBSERVATION-04 — Unwitnessed crimes do not generate a wanted flag

When the player killed the Street Performer in Gate Ward (no guards present), a
City Guard that later spawned in Market Square did not arrest the player for several
rounds while standing in the same room. Only when the player directly attacked
a guard in Market Square did the arrest trigger. This is correct behavior — witness
gating is working.

---

### OBSERVATION-05 — `help arrest`, `help justice`, `help fine` — quality is good

All three help files exist and are well-written:
- `help arrest`: explains the policy, both paths, and usage cleanly.
- `help justice`: good overview of the faction system, gives context for why arrest
  matters, mentions bounties and redemption.
- `help fine`: brief but correct; tells the player the fine decays and to use payfine.

No typos, no raw numbers leaking into player text, no 3rd-person voice errors found.

---

### OBSERVATION-06 — `set arrest` both directions work cleanly

```
set arrest resist    → "Arrest policy set to resist."
set arrest surrender → "Arrest policy set to surrender."
set arrest           → "Your arrest policy: surrender. Set with: set arrest <surrender|resist>."
```

All three variants work correctly.

---

### PASS — No crash, panic, or disconnect at any point

The session exercised multiple arrest cycles, guard combat, payfine, and in-cell
guard combat. No crash, panic, or unexpected disconnect occurred. The server
remained stable throughout.

---

## Raw Stats

- Commands sent: ~75
- Arrests (haul to cell): 5
- payfine calls: 4 (3 re-arrested, 1 clean release)
- Guards killed in combat: 4 (Velk + 3 city guards)
- Citizens killed: 2 (Street Performer, Thornwall Thug — collateral during flee)
- Disconnects / panics: 0
- Times jailbreak via flee succeeded: 1 (escaped cell but re-arrested from Barracks)
