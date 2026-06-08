# Test Report: 5.2 Bounty Hunting + 5.1 Stillwater Justice

**Date:** 2026-05-30
**Target:** local
**Role:** feature-tester
**Character:** smoketester
**Goals file:** none (inline goals: 5.2 bounty hunting + 5.1 Stillwater justice)
**Duration:** ~6 minutes, ~11 commands sent

## Session Summary

On login, the smoketester (already carrying 724g murder bounties from prior
crime testing) was **immediately pursued by a dispatched bounty hunter** — the
5.2 Half A loop fired live within the first round. The session became an
opportunistic verification of the full dispatch → pursuit → engage → kill →
reprieve chain, plus confirmation of the Half B standing-bounty board. The
smoketester (a strong character) defeated the hunter; the bounty correctly
persisted afterward (reprieve, not cleared). Both halves of 5.2 verified;
several lifecycle paths confirmed in passing.

## Goal Results

- [x] **5.2: hunter dispatched when bounty high enough** — PASS. smoketester
  has open 724g murder bounties (≥ 500 threshold) from `thornwall_guards` and
  `thornwall_citizens`; a bounty hunter was dispatched and pursued.
- [x] **5.2: hunter pursues / closes in** — PASS. "bounty hunter enters from the
  east" — it traveled (from the Thornwall barracks seat) and reached the
  smoketester at the city-gate threshold room.
- [x] **5.2: hunter engages (planner same-room attack)** — PASS. On arrival it
  immediately entered combat (clinch, weapon swings).
- [x] **5.2: affix-scaled gear used** — PASS. Hunter wielded a **"bounty
  hunter's blade (Keen)"** — the Task-6 base blade (10037) with a spawn-time
  affix ("Keen") from `GenerateAffixedItem`.
- [x] **5.2: statpool scaling** — PASS (inferred). 724g → `clamp(250 +
  724×0.25, 300, 500)` ≈ **431 statpool** — a tough but beatable elite; the
  strong smoketester won, matching the design's "pursuit pressure, not
  guaranteed execution" balance note.
- [x] **5.2: kill the hunter → reprieve (bounty stands)** — PASS. After the
  smoketester killed the hunter, bounties 12/13 remained **open** (724g) — the
  kill did NOT clear the record. Correct: only justice clears it.
- [x] **5.2: no instant re-dispatch (cooldown)** — PASS. No new hunter arrived
  in the rounds after the kill (redispatch cooldown holding).
- [x] **5.2 Half B: standing player-claimable NPC bounties** — PASS. `bounty
  list` shows the 3 seeded standing bounties with correctly auto-computed
  rewards: **Chrysalis Phantom 150g**, **Soren 87g**, **goblin shaman 65g**.
- [ ] **5.2: hunter kills player → record clears (death pays the debt)** —
  NOT TESTED (smoketester won the fight; would need to lose to a hunter).
- [ ] **5.2: jailed-target safety (no kill in cell) + serving calls off hunt**
  — NOT TESTED (requires getting arrested mid-pursuit; deferred).
- [ ] **5.2 Half B: actually claim a standing bounty (kill Chrysalis Phantom →
  150g)** — NOT TESTED (would require traveling + a boss fight).
- [ ] **5.1 Stillwater justice (Drunn warn/arrest, cell, pay fine/serve)** —
  NOT TESTED this session (the live hunter encounter took priority; the
  smoketester's bounties are Thornwall-faction, so Drunn/Stillwater wouldn't
  enforce them without a Stillwater crime).

## Findings

### PASS: 5.2 dispatch → pursuit → engage works end-to-end, live on login
A hunter was dispatched for a 724g-bounty player, traveled to the player's room
("enters from the east"), and immediately engaged. The whole Half A front half
of the feature worked with zero setup.

### PASS: Affix-scaled gear at spawn
The hunter fought with a "bounty hunter's blade (Keen)" — confirming the
instance affix-gear path (base loot_pool item + `GenerateAffixedItem` "Keen"
affix scaled off the hunter's statpool/divisor). Gear is real and named.

### PASS: Kill → reprieve semantics
Killing the hunter left the triggering bounties open (724g, status open). The
"beating a hunter buys a reprieve, not absolution" design is correct — the
bounty stands and (per cooldown) another will eventually come.

### PASS: Half B standing bounties seeded with correct rewards
The 3 standing bounties exist and their gold matches statpool-derived auto-
computation (Phantom 300→150, Soren 175→87, shaman 130→65). Players can see
them via `bounty list`.

### OBSERVATION: Hunter panic-flee fires at low HP
At "near death" the hunter "tries to flee but is blocked" — the `bounty_hunter`
archetype's shared panic-flee branch is active (it was blocked by the clinch
state). Behaves as designed.

### OBSERVATION: No gear dropped from the killed hunter
The hunter corpse dropped no gear this kill — consistent with the intended
~3%-per-piece drop rate (a single kill usually yields nothing; not a farm).

### CONCERN (balance, pre-flagged): hunter beatable by a strong wanted player
The ~431-statpool hunter was defeated by the smoketester without the smoketester
dropping below ~40% HP. This matches the final-review balance note — for very
strong, very wanted players, hunters may read as surmountable rather than
terrifying. Intended as "pressure," but worth a playtest-tuning look (e.g. raise
`BountyHunterStatpoolPerGold` or the max clamp) if hunters feel like free kills.

## Raw Stats

- Commands sent: ~11
- Fights: 1 (vs the dispatched bounty hunter)
- Deaths: 0
- Spells cast: 0
- Items used: 0
- Bugs found: 0
- Concerns: 1 (balance, pre-flagged)
- Observations: 2
- Passes confirmed: 8
