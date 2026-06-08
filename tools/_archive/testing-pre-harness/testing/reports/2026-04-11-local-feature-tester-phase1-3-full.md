# Test Report: Full smoke test of JS Audit Phases 1-3 plus sethome command

**Date:** 2026-04-11
**Target:** local
**Role:** feature-tester
**Character:** smoketester
**Goals file:** phase1-3-full.yaml
**Duration:** ~20 minutes, ~120 commands sent

## Session Summary

Started in Bandit Hollow with two hostile bandits already engaging. Killed them, then
systematically worked through goals: tested sethome command, verified spell YAML text
fields (Phase 1), tested companion summoning and cap (Phase 2), checked buff tick
display and conditions command, and fought multiple mob types across Dustwalk Road and
Sanctum Basin caves. Earth elemental companion followed faithfully throughout.

## Goal Results

### Setup
- [x] Equip available gear — PASS: equipped wool cloak and copper ring, crude bone club already wielded
- [x] Run sethome to see current home — PASS: showed "Current home: Sanctum Basin" with options
- [x] Run sethome thornwall to change it — PASS: "Home set to Thornwall City (Temple Interior)"
- [x] Run sethome again to confirm — PASS: showed "(current)" next to Thornwall
- [x] Run help sethome — PASS: help file displays correctly with usage examples

### Phase 1 - YAML text fields (spell flavor text)
- [x] Cast conviction-spike on hostile mob — PASS: cast text "You focus your conviction into a sharp point of force.", wait text "The mental image of Conviction Spike begins to fracture into its component folds...", resolution "Your Conviction Spike strikes Dustwalk Bandit! (serious wounds)" all appeared correctly
- [x] Cast conviction-surge on self — PASS: cast "You channel conviction into empowering energy.", wait "Your consciousness splinters as the folds of Conviction Surge begin to take shape...", resolution "Your Conviction Surge takes effect on smoketester!" + buff start "Conviction surges through your limbs, empowering your strikes." + buff end "The surge of conviction fades from your limbs." all confirmed
- [x] Cast chrysalis-glow — PASS: cast "You coax the Chrysalis spores around you to luminesce.", wait text, resolution "A warm glow surrounds you." all appeared
- [x] Cast mend-wounds on self — PASS: cast "You press your will into the wounds, urging flesh to mend.", wait text, resolution "You weave restorative magic around smoketester." + "A warm glow of healing magic envelops you. Your wounds begin to mend." confirmed
- [x] Cast sparks AoE — PASS: cast "You shatter your conviction into razor-sharp fragments.", wait text, resolution "Your Conviction Sparks strikes Scrubland Dog! (serious wounds)" confirmed
- [x] Cast blood-boil — PASS: cast "You focus on the target's blood, willing it to boil.", wait text, resolution "Your Blood Boil afflicts Scrubland Dog!" confirmed

### Phase 2 - Companion summoning
- [x] Cast conjure-earth — PASS: earth elemental spawned as companion (♥friend), followed player across zones, engaged in combat automatically
- [ ] Raise skeleton from corpse — BLOCKED: all corpses in starting areas (bandits, dogs, birds, cave goblins, even Aberrant Chrysalis boss) assessed as "barely a trace of essence within. The essence is too faint to animate any form." No corpse with sufficient essence found in accessible zones
- [x] Help raise-skeleton — PASS: help file shows correctly, requires "weak remains"
- [ ] Assess corpse before raise — PASS (assess works): "You study the remains of [mob]. You sense barely a trace of essence within."
- [ ] Summon-hive-swarm — BLOCKED: no hive fragment component in inventory, not available from Merchant Adela
- [x] Try to summon past companion cap — PASS: cast conjure-water with earth elemental active, spell channeled but then showed "You cannot maintain any more companions." Spell consumed CP but did not spawn second companion

### Phase 2 - Config-driven buff ticks
- [x] Drink healing salve — PASS: "You drink the healing salve." Buff applied immediately
- [x] Check conditions during active buff — PASS: conditions command showed "Healing Salve — Warmth spreads through your body, mending injuries." with duration phases (well established → active → briefly active → expired)
- [ ] Wait for buff to expire and verify end text — FAIL: Healing Salve expired (disappeared from conditions list) but no expiry text was observed. May have been lost between output reads (bridge limitation), or end text may not be configured for potions

### Phase 3 - Charm spell
- [ ] Cast charm on non-immune hostile mob — BLOCKED: character has not learned the Charm spell. "You haven't learned the spell 'Charm'."

### Phase 2 - Antidote cure
- [ ] Drink antidote when poisoned — SKIPPED: never got poisoned during testing

### General
- [x] Run spells command — PASS: spell list displays correctly with SpellId, Name, Target, Cost, Cast time, Difficulty, and Reliability columns. 17 spells listed
- [x] Run conditions during various active buffs — PASS: conditions showed Illumination, Healing Salve, Conviction Surge, and Regenerating buffs at various points with descriptive text and duration phases
- [x] Fight at least 2 different mobs — PASS: fought Dustwalk Bandits, Scrubland Dogs, Scavenger Birds, Cave Bat, Cave Goblin Guard, Aberrant Chrysalis (boss) — 6 distinct mob types
- [x] Check status periodically — PASS: HP/SP/CP bars tracked correctly, status command showed full stat sheet

## Findings

### PASS: Spell YAML text fields (Phase 1)
All tested spells (conviction-spike, conviction-surge, chrysalis-glow, mend-wounds, sparks, blood-boil) displayed proper cast text, wait text, and resolution text. Buff start and end text also confirmed for conviction-surge. The Phase 1 JS-to-YAML migration for spell flavor text is working correctly.

### PASS: Companion summoning (Phase 2)
Earth elemental spawns correctly, follows player across rooms, auto-engages hostile mobs, displays (♥friend) tag. Companion cap enforcement works — attempting to summon a second companion fails with clear error message after the spell channels.

### PASS: Sethome command
Works as designed — shows current home and options, changes home correctly, persists across checks. Help file is complete.

### PASS: Conditions display
Conditions command shows buff names, descriptive text, and duration phases (well established → active → briefly active → fading fast → expired). Clean formatting with ASCII box drawing.

### CONCERN: No corpse with sufficient essence in starting zones
Every corpse tested (6 different mob types including the Aberrant Chrysalis boss) assessed as "barely a trace of essence within. The essence is too faint to animate any form." This means raise-skeleton cannot be tested without traveling to higher-level zones. The test character may need to be placed in a zone with stronger mobs, or the test goals should specify which zone to travel to.

### CONCERN: Potion buff end text not observed
Healing Salve buff expired without visible end text to the player. The conviction-surge spell had clear end text ("The surge of conviction fades from your limbs"), but no similar message appeared for the potion. This could be:
1. Bridge output buffer overwritten between reads (bridge limitation)
2. No end text configured for potion buffs
This aligns with known bug "ConditionRegen has no tick text" in project memory.

### CONCERN: Mend-wounds Regenerating buff has no tick or end text
The Regenerating buff (from mend-wounds) appeared in conditions and expired, but no periodic "you feel healing" tick messages or end text were observed. This is the known "ConditionRegen has no tick text" bug.

### PASS: Spell fizzle messages vary
Multiple fizzle messages observed: "You reach for the folds of [spell] but the image shatters before you can grasp it" and "Your grip on [spell] slips; the folds scatter before the pattern takes hold." Good variety.

### PASS: Earth elemental combat behavior
The elemental auto-targeted hostile mobs, attempted grapples, charged, and threw haymakers. When knocked prone, it showed "attempts to stand" and "clambers to their feet" messages. Good combat text variety.

### OBSERVATION: Charset toggle behavior
The `set charset` command toggles between ASCII and UTF-8. When starting a session, the initial state may vary. Had to toggle twice to reach ASCII mode. Not a bug, just the toggle nature of the command.

### OBSERVATION: Stat progression working
Multiple stat increases observed during testing: "Your dexterity grows stronger!", "Your strength grows stronger!", "Your willpower grows stronger!", "You feel your combat skills sharpening!", "You feel your manifestation skills sharpening!", "You feel your weapon-combat skills sharpening!" Use-based progression is firing correctly.

## Raw Stats

- Commands sent: ~120
- Fights: 10 (2 bandits, 3 dogs, 2 birds, 1 bat, 1 goblin, 1 boss)
- Deaths: 0
- Spells cast: ~10 (conviction-spike x2, conviction-surge x3, chrysalis-glow x1, mend-wounds x2, sparks x1, blood-boil x1, conjure-earth x1, conjure-water x1)
- Items used: 1 (healing salve)
- Bugs found: 0 (known bugs confirmed)
- Concerns: 3
- Observations: 2
- Passes: 8
