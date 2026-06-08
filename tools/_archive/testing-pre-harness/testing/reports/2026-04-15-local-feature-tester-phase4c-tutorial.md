## Test Report: Phase 4C Tutorial Walkthrough

**Date:** 2026-04-15
**Target:** local
**Role:** feature-tester
**Character:** smoketester
**Goals file:** phase4c-tutorial.yaml
**Duration:** ~25 minutes, ~45 commands sent

## Session Summary

Walked the Sanctum Trials quest end-to-end on an existing test character.
The quest steps automatically chained via hint system: ceremony → Market
(buy + wield) → Training Yard (kill dummy) → Forge (craft iron dagger) →
Alchemist (craft healing salve) → West Meadow (forage + track) →
Observatory (cast spell) → Boss Cave (kill Aberrant Chrysalis) →
Basin Gate (graduation). Quest completed 100% and Warden unlocked the
south gate to World Gate. All nine tutorial rooms fired their behavior
trees correctly, all NPCs reacted on entry with appropriate delays.

## Goal Results

- [x] Character creation — BLOCKED: smoketester is an existing character,
  skipped "start" flow. Already had 25g/100 bank, Keen Eyes mutation from
  a prior ceremony. Not a fresh-character test.
- [x] Academy Hall ceremony — PASS: Chrysalis Priest performed the Awakening
  Rite on entry, forehead touch → "The air hums" → mutation granted →
  "Type mutations to see what the change has wrought." Quest started at
  15%. `look mosaic` rendered the ASCII map correctly.
- [x] Market Street buy/equip — PASS: Merchant Adela greeted on entry,
  explained `list`/`buy`, guided through purchase. `buy sharp stick` for
  3g worked, `wield sharp stick` equipped. Quest advanced to 30%.
  (Note: quest step jumped from "buy equipment" straight to Training Yard,
  skipping an explicit equip check — see OBSERVATION below.)
- [x] The Forge — PASS: Korvath auto-granted iron ingot + leather strip
  on entry, craft iron dagger completed over 3 rounds. Quest advanced to 46%.
- [x] Alchemist's Workshop — PASS: Yenna auto-granted 2× healer's root +
  glass vial on entry, craft healing salve completed. Quest advanced.
- [x] West Meadow — PASS: Wilderness Guide Fen greeted, `forage` found a
  healer's root and advanced quest; `track` showed recent visitors (meadow
  lizard) and guide acknowledged.
- [x] Training Yard — PASS: Combat Trainer reacted to combat start with
  coaching lines (kick/grapple/trip/taunt/bash advice). Killed training
  dummy after extended fight; weapon-combat hit apprentice. Quest advanced.
- [x] Observatory — PASS: Elder Saris present; `cast conviction-spike echo`
  hit the Chrysalis Echo and advanced quest. Defeated Echo for further
  progression. Elder Saris emoted ward/focus lines during fight.
- [x] Boss Cave — PASS: Aberrant Chrysalis killed, dropped Chrysalis Core
  and 10g. Quest to 84%.
- [x] Basin Gate graduation — PASS: Basin Warden recognized completion
  ("Your record is complete. Six trials, six instructors — and the cave."),
  south gate shown as `south (unlocked)`. Quest completed. South exit led
  to World Gate successfully.
- [x] NPC pacing — PASS: Every NPC had a natural delay before speaking on
  room entry. Idle emotes scattered between combat rounds (trainer
  practicing forms, Yenna making ledger notes, Fen crouching to read
  tracks). Felt paced, not robotic.
- [x] Quest step progression — PASS: Every trial step advanced the quest
  the moment the required action completed. No stuck steps.
- [x] No visible parse/btree errors during session.

## Findings

### PASS: Behavior tree driven NPCs feel alive
All nine tutorial NPCs (Priest, Adela, Korvath, Yenna, Fen, Trainer,
Saris, Aberrant, Warden) greeted on entry, interjected mid-activity
emotes, and gated their coaching lines appropriately. Delays before
speech feel like perception-scaled reaction rather than scripted snaps.

### PASS: Hint system drives the whole quest
`hint` gave exact directional breadcrumbs for every step. A fresh player
could complete the tutorial using only `quest` + `hint` + the single
action the hint calls out.

### PASS: Ceremony lock-step
Academy Hall ceremony sequence played correctly: priest approaches →
fingers on forehead → "The air hums" → ceremony completes → mutation
granted → instructional line about `mutations` command.

### OBSERVATION: Quest step labels skip some sub-steps
Goals file expects separate "buy" and "equip" checks on Market Street,
but the quest tracker jumped straight from 15% → 30% after `buy sharp
stick` and showed the next step as Training Yard without an explicit
"equip something" confirmation. `wield` clearly worked (weapon used all
fight), but if the design intent is two sub-steps, they are either
collapsed or the equip check is implicit. Worth a spec review — not
a bug unless two steps are required.

### OBSERVATION: Training Dummy is HP-sponge long
Dummy took ~15 rounds of sharp-stick combat to go from healthy → dead,
most swings landing. For a "tutorial dummy" meant to teach combat this
feels a beat too long; a fresh player with lower stats than smoketester
could easily spend 25+ rounds here. Consider reducing dummy HP by ~30%
for pedagogical pacing, or flag quest progress at "badly wounded"
rather than kill.

### OBSERVATION: Boss Cave "Aberrant" not an exact keyword for look
`attack aberrant` worked cleanly (target resolution tolerant), so this
is purely PASS — mentioning only because `Also here: Aberrant Chrysalis`
could confuse a player into typing `attack chrysalis` and hitting the
wrong thing in rooms that have chrysalis echoes. No issue on Boss Cave
specifically.

### CONCERN: Existing-character smoke tests can't verify fresh-ceremony mechanics
smoketester already had a Keen Eyes mutation from a previous session —
the ceremony re-granted the *same* mutation (or just re-showed
existing). Can't tell from this session whether the ceremony correctly
rolls a new mutation on a truly fresh character, or whether it's
idempotent. To fully validate the Awakening Rite path, need to run
this with a character that has never been through it.

## Raw Stats

- Commands sent: ~45
- Fights: 3 (training dummy, Chrysalis Echo, Aberrant Chrysalis)
- Deaths: 0
- Spells cast: 1 (conviction-spike)
- Items crafted: 2 (iron dagger, healing salve)
- Items foraged: 1 (healer's root)
- Bugs found: 0
- Concerns: 1
- Observations: 3
- Passes: 3
