# Smoke Test Report: Chunk 1.4 NPC Knowledge Model — Thornwall Crime Witnessing
**Date:** 2026-05-09
**Tester:** smoketester (AI feature-tester role)
**Target:** local (localhost:55555)
**Substrate:** Chunk 1.4 — per-NPC knowledge auto-write on crime witnessing (T15/T16/T17)

---

## Smoke Verdict

All three player-driven crime paths were reached during this session. The
T15 assault path fired (attack on Tavern Keeper Marek in room 472 with
Barmaid Dal as witness), and the subsequent murder happened while I was
absent from the room — which is an edge case for T16 Case A (I fled before
Marek died from accumulated wounds). The T16 hook should have fired for
current room witnesses when Marek's health reached zero; only Barmaid Dal
was present. The lone-act path (Goal 3) was executed cleanly in room 462
with no other faction mobs present — the Street Performer was killed solo,
which should trigger the lone-perp logic and skip knowledge writes. The T17
theft path fired definitively: Barmaid Dal caught the steal in room 472 and
entered combat, which is the canonical "caught in the act" failure branch
where T17 writes knowledge. Admin verification is required for all three
paths to confirm that knowledge records actually landed (or were correctly
skipped). One CONCERN is raised about the murder-upgrade path: Marek died
from wounds after I fled, so the question of which witnesses were "current"
at the moment of death depends on whether Dal was in the same room as the
corpse when the death event fired.

---

## Goal Results Table

| Goal | Description | Result | Notes |
|------|-------------|--------|-------|
| 0 | Login + orientation | PASS | Spawned in Gate Ward (room 460), ASCII mode confirmed, inventory shows sharp stick + iron dagger equipped, skullduggery at apprentice |
| 1 | Witnessed assault → knowledge write | PASS (action completed) | Attacked Tavern Keeper Marek (mob 96) in room 472 with Barmaid Dal (mob 117) present as witness; fled after assault; Marek survived initial exchange. Admin verification needed. |
| 2 | Murder upgrade → knowledge preserved | PARTIAL | Marek died from wounds while I was absent (I had fled to room 471). Corpse + Dal confirmed in room 472 on return. The death event should have fired T16; whether witnesses at death time were correct is unverified. Admin verification needed. |
| 3 | Lone act → no knowledge write | PASS (action completed) | Street Performer (mob 101) killed solo in room 462 (Main Street Central); no other faction mobs visible or found via search. Should trigger lone-perp path. Admin verification needed. |
| 3.5 | Rank up skullduggery to 2 | PASS | Skullduggery already at [apprentice] (rank 2+) on login. During session it progressed further (saw "skullduggery skills sharpening" messages during steal attempts). No grinding needed. |
| 4 | Failed steal → theft knowledge write | PASS (action completed) | Barmaid Dal caught the steal attempt ("barmaid Dal catches you in the act!"), entered combat. This is the T17 caught-path. Fled successfully. Admin verification needed. |
| 5 | Summary verdict | PASS | See Smoke Verdict above. |
| 6 | Admin commands listed | PASS | See section below. |

---

## Findings

### OBSERVATION: Steal succeeds vs. empty-inventory mobs — no crime path

All of the following mobs had no gold and no items, causing steals to
"succeed" the dice roll but return "nothing worth taking" — which does NOT
hit the T17 knowledge-write branch:
- City Beggar (mob 100, room 461)
- Records Clerk Pell (mob 99, room 463)
- Old Fen (mob 114, room 484) — x3 attempts
- Old Gobb (mob 115, room 484)

The T17 knowledge write only fires on the caught-in-the-act path (dice roll
failure). It does NOT fire on successful-steal-but-nothing-found. This means
T17 cannot be exercised against mobs with empty inventories, regardless of
how many attempts are made. Barmaid Dal (mob 117) had sufficient inventory
or gold to cause a dice-roll failure, triggering the catch.

**Implication for test setup:** Future smoke runs should pre-populate target
mobs with gold or items if the goal is to guarantee a caught-steal. The
lucky catch against Dal is the only T17 test data point this session.

### CONCERN: Murder upgrade (T16) — perp was not in the room at death

Tavern Keeper Marek (mob 96) died from accumulated wounds after I fled room
472. When I returned to the tavern, the corpse and 400 gold were on the
ground, and only Barmaid Dal remained at 100% health. This means:

- The MobDeath hook (T16) fired while I was in room 471 (Craftsmen's
  Quarter West), not room 472.
- The "current witnesses" at the moment of death would have been whoever
  was in room 472 at that tick — presumably only Barmaid Dal.
- The assault crime row should have been upgraded to murder in-place.
- Whether the T16 knowledge write for Dal fires in this scenario (perp not
  present) needs admin confirmation.

This is not a bug per se — the spec says "Case A: external witness present
at the kill" — but the perp's absence from the room at the moment of the
kill may affect which case applies. If T16 checks `perp.Type == PerpPlayer`
independently of perp presence in the room, the write should still fire.

### OBSERVATION: Goal 2 complicates the Goal 1 delta check

Because Marek died before I could return for a deliberate murder (Goal 2),
Goals 1 and 2 effectively merged into a single crime sequence. The admin
delta check for Goal 1 (assault witness) and Goal 2 (murder upgrade) will
both need to be verified against the single crime row that should have gone
assault → murder.

### OBSERVATION: Multiple steal attempts consumed before finding T17 path

Approximately 6 steal attempts (City Beggar, Records Clerk, Old Fen x3,
Old Gobb) found nothing worth taking before Barmaid Dal triggered the
caught-path. Each successful-but-empty steal still consumed the skill
cooldown and advanced skullduggery progression (saw two "sharpening"
messages). This is expected behavior but means any future T17 smoke run
should go directly to Dal or another mob with gold.

### PASS: Skullduggery already at apprentice on login

No grinding required. Pre-existing state from the 1.3 smoke run had
already advanced skullduggery to apprentice tier.

### PASS: Lone-act room confirmed empty

Room 462 (Main Street Central) showed only Street Performer at time of
attack. A `search` command found no hidden mobs. The attack and kill
proceeded without any faction witness entering the room during combat.

---

## Per-Act Event Ledger

| # | Act Type | Room ID | Room Name | Victim Mob ID | Victim Name | Bystander Witness | Notes |
|---|----------|---------|-----------|---------------|-------------|-------------------|-------|
| 1 | Assault (T15) | 472 | The Drowning Post Tavern | 96 | Tavern Keeper Marek | mob 117, Barmaid Dal | I fled after assault; Marek was bruised. T15 should have written knowledge for Dal. |
| 2 | Murder upgrade (T16) | 472 | The Drowning Post Tavern | 96 | Tavern Keeper Marek | mob 117, Barmaid Dal (inferred) | Death occurred while I was absent in room 471. Dal was present in 472 on my return. T16 should have upgraded the crime row and written knowledge for Dal. |
| 3 | Lone murder (no witnesses) | 462 | Main Street Central | 101 | Street Performer | none (solo kill confirmed by search) | Lone-perp path; perp should be set to unknown; no knowledge write expected. |
| 4 | Failed theft / caught (T17) | 472 | The Drowning Post Tavern | 117 | Barmaid Dal | mob 117 (victim = only witness) | "barmaid Dal catches you in the act!" fired. Dal is the victim and only witness. T17 should have written knowledge for Dal about smoketester. |

---

## Admin Commands Needed

### Goal 1 — Witnessed Assault (T15)

```
knowledge show 117 smoketester
```
Expected: Dal has a knowledge record about smoketester with HasMet: true,
Source: witnessed, at least one crime ID in CrimesWitnessed matching an
assault row.

```
crime list thornwall_citizens
```
Expected: A KindAssault row with perp=smoketester, room=472, created
during this session (most recent rows).

### Goal 2 — Murder Upgrade (T16)

```
knowledge show 117 smoketester
```
Expected: Same crime ID from the assault row is now present (upgraded
in-place to KindMurder). The CrimesWitnessed list on Dal should NOT have
gained a duplicate crime ID; the same ID should now point to a murder row.

```
crime list thornwall_citizens
```
Expected: The most recent KindMurder row with perp=smoketester, room=472.
The assault row from Goal 1 should be gone (upgraded in-place) or the
murder row should reference the same ID.

### Goal 3 — Lone Act / No Knowledge Write

```
knowledge show smoketester
```
Expected: Lists every NPC who has knowledge about smoketester. After Goal 3,
there should be NO new knowledge records added — only the pre-existing
records from Goals 1 and 2 (Dal at minimum).

```
crime list thornwall_citizens
```
Expected: A KindMurder row with perp=unknown, victim=street_performer
(mob 101), room=462.

### Goal 4 — Failed Theft / T17 Write

```
knowledge show 117 smoketester
```
Expected: Dal's knowledge of smoketester now includes an ADDITIONAL crime ID
in CrimesWitnessed corresponding to a KindTheft row from this session.

```
crime list thornwall_citizens
```
Expected: A KindTheft row with perp=smoketester, room=472, victim=barmaid
dal (mob 117).

---

## Raw Stats

| Metric | Value |
|--------|-------|
| Commands sent | ~55 |
| Deaths | 0 |
| Kills | 2 (Tavern Keeper Marek via wound bleed-out, Street Performer) |
| Combats entered | 3 (Marek assault, Dal combat after theft catch, brief Marek fight) |
| Successful flees | 2 (from Marek, from Dal) |
| Steal attempts | ~7 total (6 "nothing worth taking", 1 caught) |
| Skullduggery rank on login | apprentice (rank 2+) |
| Skullduggery rank on logout | apprentice (progressed further, exact rank unknown) |
| Session duration | ~25 minutes |
| Player deaths | 0 |
