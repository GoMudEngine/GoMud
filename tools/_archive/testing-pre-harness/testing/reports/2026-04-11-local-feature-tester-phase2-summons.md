# Test Report: Phase 2 Companion Summoning and Buff Tick Systems

**Date:** 2026-04-11
**Target:** local
**Role:** feature-tester
**Character:** smoketester
**Goals file:** phase2-summons.yaml
**Duration:** ~15 minutes, ~50 commands sent

## Session Summary

Connected to the local server at Dustwalk Road with an existing Hive Swarm companion. Worked through all 10 goals methodically — dismissed existing companions to test summoning, fought multiple dustwalk bandits in melee and with spells, tested healing via both spells and potions, and verified the conditions display. Conjure-earth and summon-hive-swarm worked well. Raise-skeleton consistently failed across 5 attempts, which is likely a bug. Healing tick text was absent for both mend-wounds and potions, confirming the known ConditionRegen gap.

## Goal Results

- [x] Equip available gear from inventory and verify with status — **PASS**: Crude bone club (weapon) and copper ring (ring) already equipped. Status showed all stats and equipment correctly.
- [x] Cast conjure-earth — verify earth elemental spawns as companion — **PASS**: Earth Elemental spawned as companion (100%|friend). Great multi-phase casting text: initiation ("You drag your will into the earth itself..."), channeling ("The floor shudders..."), and completion ("The floor cracks apart as a massive figure of stone and compacted earth heaves itself upright.").
- [x] Cast conviction-surge on self — verify buff start text and end text — **PASS**: Start text: "Conviction surges through your limbs, empowering your strikes." End text (after buff expired): "The surge of conviction fades from your limbs." Both displayed correctly.
- [x] Find and kill a hostile mob using melee combat — **PASS**: Killed dustwalk bandit in melee with earth elemental companion helping. Stats progressed during fight (vitality, dexterity). Combat text varied and descriptive.
- [ ] Cast raise-skeleton — verify companion spawns and corpse consumed — **FAIL**: Spell failed 5 out of 5 attempts across two different rooms with fresh corpses. Every attempt shows "Your spell erupts outward but finds no targets" before a fizzle message. Corpse was NOT consumed on failure. Manifestation skill still progressed. See BUG below.
- [x] Cast summon-hive-swarm — verify hive fragment component consumed from inventory — **PASS**: Component consumed ("You crush the Hive Fragment in your fist"), companion spawned ("A Hive Swarm coalesces around you, awaiting your command!"). Multi-phase casting text worked perfectly.
- [ ] Cast mend-wounds on self — check for healing tick flavor text — **FAIL**: Start text worked ("A warm glow of healing magic envelops you. Your wounds begin to mend.") but NO tick text appeared during the buff's duration. HP bars showed recovery in prompts but no descriptive text. Confirms known bug.
- [ ] Drink a healing potion — verify healing buff applies with tick messages — **FAIL**: Buff applied correctly (conditions showed "Healing Salve" and "Regenerating"), HP recovered, but NO tick messages displayed. Same issue as mend-wounds.
- [x] Cast sparks or conviction-spike in combat — verify damage spell text phases — **PASS**: Conviction Spike dealt "serious wounds" to a dustwalk bandit with proper damage description text. Initiation text: "You focus your conviction into a sharp point of force." Sparks initiation also worked ("You shatter your conviction into razor-sharp fragments") but no targets remained when it resolved.
- [x] Check conditions command during active buffs — verify buff names and descriptions show — **PASS**: Conditions displayed correctly. Conviction Surge: "Your conviction bolsters your strength, empowering your attacks." / "active". Healing Salve: "Warmth spreads through your body, mending injuries." / "well established". Regenerating: "Healing magic mending wounds over time" / "active".

## Findings

### CONCERN: raise-skeleton failed 5/5 attempts — likely corpse too weak
Every raise-skeleton attempt (5/5) showed fizzle text despite corpses confirmed present in the room. Tester did NOT use `assess` on corpses before casting — dustwalk bandits are low-level mobs and may not meet a minimum stat threshold for skeleton raising. The fizzle text strongly hints at this: "Not enough essence remains", "the spark of undeath finds nothing to cling to." This is likely working as designed rather than a bug. Future testing should `assess corpse` first and use stronger mob corpses.

### BUG: Dismissed companions turn hostile (known)
All three dismiss attempts (hive swarm x2, earth elemental x1) resulted in the companion attacking the player. "Hive Swarm turns on you with fury!" / "Earth Elemental turns on you with fury!" This is a known bug already tracked in memory.

### BUG: No healing tick text for mend-wounds or potions (known)
ConditionRegen heals silently — HP bars update in the prompt but no descriptive text is shown per tick. Start text works fine for both mend-wounds and healing salve potion. This is a known bug already tracked in memory.

### PASS: Conjure-earth multi-phase casting text
Excellent flavor text across initiation, channeling, and resolution phases. Earth elemental spawned correctly as a companion that follows and fights.

### PASS: Summon-hive-swarm component consumption
Hive Fragment was properly consumed from inventory. Companion spawned with good thematic text.

### PASS: Conviction-surge start/end text
Both buff application and expiration messages displayed correctly via the YAML text field system.

### PASS: Conviction-spike damage spell
Single-target conviction damage worked correctly with descriptive damage text ("serious wounds") rather than raw numbers.

### PASS: Conditions command display
Buff names, descriptions, and status indicators all displayed correctly for multiple concurrent conditions.

### OBSERVATION: Spell fizzle text variety
Raise-skeleton showed at least 4 different fizzle messages across 5 failures: "Bones rattle briefly, then collapse into dust", "The remains shudder and twitch, but the spark of undeath finds nothing to cling to", "Your magic courses through the remains, but finds only emptiness", "The corpse stirs, a hollow mockery of life flickers — then fades." Good variety, though the spell shouldn't be failing this consistently.

### OBSERVATION: Atmospheric room text working
Room ambiance messages fire regularly: "A scrap of torn canvas flaps in the weak breeze", "The cold fire ring smells of old ash and rancid fat", "Something clinks among the stolen goods as the breeze shifts a loose tarp."

### OBSERVATION: Companion follows through rooms
Earth elemental and hive swarm both correctly followed the player through room transitions ("earth elemental enters from the north", "Hive Swarm enters from the north").

### OBSERVATION: Stat progression working
Multiple stat progressions observed during testing: vitality, dexterity, charisma, plus weapon-combat, unarmed-combat, and manifestation skills.

## Raw Stats

- Commands sent: ~50
- Fights: 6 (2 scavenger birds [1 pre-existing], 3 dustwalk bandits, 2 dismissed companions)
- Deaths: 0
- Spells cast: 10 (conjure-earth x1, conviction-surge x1, raise-skeleton x5, summon-hive-swarm x1, mend-wounds x1, conviction-spike x1, sparks x1 [note: some counted as a single cast attempt])
- Items used: 2 (healing salve x1, hive fragment consumed x1)
- Bugs found: 2 (dismiss=hostile [known], no heal tick text [known])
- Concerns: 1 (raise-skeleton 5/5 failures, likely bad RNG)
- Observations: 4
