# Test Report: Phase 4b Behavior Tree Migrations

**Date:** 2026-04-14
**Target:** local
**Role:** feature-tester
**Character:** smoketester
**Goals file:** phase4b-mob-migration.yaml
**Duration:** ~25 minutes, ~130 commands sent

## Session Summary

Started at The Drowning Post Tavern with Barmaid Dal present. Tested Dal's gossip
dialogue (PASS) and observed the tavern for ambient behavior (partial — only idle
text seen, no full patrol cycle or patron heckling observed in ~60s of observation).
Searched Temple District for Records Clerk Pell but only found Temple Priest Olen
(Pell not in the temple or adjacent rooms). Tested item-give on Olen (kept item with
flavor text) and City Gate Guard (default "considers" emote, no custom handler).
Travelled south through Watchers Crossing to Marches Spur Road and into Ashwick
hamlet looking for The Hermit's Cottage and Old Edrin, but the Hermit's Cottage
(room 4036) was not located in the explored area — fought several mobs en route
but could not complete the boss reveal test. Warden Tessara was unreachable in the
available time (Warden's Post is on the Dustwalk Road route south of Sanctum Basin,
far from the Thornwall-area start location).

## Goal Results

### Dal patron heckling
- [x] Travel to The Drowning Post Tavern, find Barmaid Dal — PASS: started directly at room 472 with Dal present
- [ ] Observe Dal patrol + patron heckling — PARTIAL: observed ambient idle text ("barmaid Dal pushes a strand of hair from her face and glances toward the back corner", "barmaid Dal sets a fresh cup on the bar without being asked", tavern keeper Marek idle text) over ~60 seconds of quiet observation, but did not see Dal walk south to the back corner or any patron heckle from Old Fen/Gobb/Wrex
- [x] Ask Dal about gossip — PASS: Dal responded "You want gossip? Ask the old men in the back corner. They have more stories than sense." — dialogue works, voice is in-character

### Records Clerk Pell
- [ ] Travel to Temple Interior, find Pell — BLOCKED: Temple Interior (north from Temple District) contains only Temple Priest Olen. Pell not found in adjacent rooms (east leads to Riftkeeper Sable's chamber). Pell may be in a different room number than expected, or may require a different route
- [ ] Give Pell a random item — BLOCKED: Pell not found

### Road Warden Tessara
- [ ] Travel to Warden's Post, find Tessara — BLOCKED: Warden's Post is on Dustwalk Road south of Sanctum Basin. Starting location near Thornwall makes this a very long multi-zone traversal. Could not complete in available time
- [ ] Give Tessara a random item — BLOCKED: Tessara not reached

### Old Edrin boss reveal
- [ ] Travel to The Hermit's Cottage (room 4036) on Marches Spur Road — BLOCKED: traversed south through Watchers Crossing → Marches Spur Road → Ashwick → multiple rooms (Ashwick Delia's Cottage, Maren's Cottage, Forager's Camp, Herb Clearing) but no room matching "The Hermit's Cottage" or containing "Old Edrin" was found in the explored area. The cottage may be off a path not explored, or via a different route
- [ ] Attack Edrin and observe reveal sequence — BLOCKED: Edrin not found
- [ ] Note whether Edrin uses different spells by target count — BLOCKED

### General observations
- [x] Note reaction delays — PASS: NPC idle/ambient text appeared naturally between commands (Marek polishing, Dal glancing, cup-setting, etc.), not instantaneous spam
- [x] Report silent item-consumption cases — See CONCERN below

## Findings

### PASS: Dal dialogue responds correctly
`ask dal gossip` returned in-character snappy response pointing to the back corner regulars. Voice is first-person from Dal, matches tavern flavor.

### PASS: Temple Priest Olen item-give has flavor text
Giving the protection notice to Olen produced:
> "The temple accepts tithes in coin, not goods."

This is a clear rejection with flavor. However, Olen **kept the item** (not found in room or inventory afterward). This may be intentional "donation accepted silently" behavior rather than a rejection-with-return pattern. Not necessarily a bug, but worth confirming whether Olen is expected to return the item or accept it as a tithe.

### CONCERN: City Gate Guard uses default fallback emote
Giving an item to City Gate Guard produced only:
> "city gate guard considers the crude short blade for a moment."

The item was kept and no custom rejection text fired. Per CLAUDE.md guidance, this is the default emote that appears when no `onGive` script / behavior tree handler exists. City Gate Guard is not on the Phase 4b migration list specifically, but the generic-guard behavior tree migration was implied. If Tessara shares the same underlying template, she may have the same gap. **Recommend verifying Tessara directly** — she was the test target and may differ.

### OBSERVATION: Dal ambient text is present but patrol not confirmed
During ~60 seconds of quiet observation in the tavern, I saw:
- "barmaid Dal pushes a strand of hair from her face and glances toward the back corner"
- "barmaid Dal sets a fresh cup on the bar without being asked"
- "tavern keeper Marek polishes a mug with a cloth, eyes on the room"
- "tavern keeper Marek leans on the bar, listening to a conversation at a nearby table"

I did not see Dal actually traverse south to The Back Corner, nor any heckling emotes from Old Fen/Gobb/Wrex. The patrol cycle may be longer than my observation window (goals suggested 2-3 minutes, but the bridge output buffer makes long-duration passive observation difficult — ambient lines scroll past before they can be captured). **Suggest checking tick frequency on the patrol behavior** if the intended cadence is shorter.

### OBSERVATION: Hermit's Cottage not in Ashwick village
Goals said "The Hermit's Cottage (room 4036) on Marches Spur Road... south from Dustwalk Road." I travelled the length of Marches Spur Road from Watchers Crossing south to Ashwick Crossroads, then explored Ashwick hamlet including Delia's Cottage, Maren's Cottage, the Chapel Lane, East Edge, and forest paths (Forest Path → Deep Woods → Forager's Camp → Herb Clearing). No room named "Hermit's Cottage" or containing "Old Edrin" was visible from cardinal exits. Goals may need a more specific route hint, or the cottage may be reached from a direction I didn't try (maybe west from Ashwick Crossroads? I did not try that branch).

### OBSERVATION: Starting location vs. goal locations
Goals file says "Smoketester starts near Thornwall" but the listed goals span Thornwall (Dal, Pell), Dustwalk Road (Tessara — far north), and Marches Spur Road (Edrin — south). Tessara is about 10+ rooms from the Thornwall start via Sanctum Basin. Recommend either pre-positioning smoketester at Sanctum Basin for multi-zone tests, or splitting Phase 4b goals into city-only and rural subsets.

### OBSERVATION: Character start-state inconsistency
On connect, the prompt initially showed full inventory available (equipment/potions from prior session), but a stray `inventory` call earlier in the session returned "no objects" and "Carrying: no objects" before recovering. Likely a bridge output-buffer artifact rather than a character-state bug, but worth flagging.

## Raw Stats

- Commands sent: ~130
- Fights: 4 (Thornwall Highwayman, Road Bandit, Feral Hog, Bandit Leader, plus Farm Cat mid-fight)
- Deaths: 0
- Spells cast: 1 (mend-wounds)
- Items used: 0
- Bugs found: 0
- Concerns: 1 (City Gate Guard default emote — may indicate gap if shared with Tessara)
- Observations: 4
- Passes: 3 (Dal dialogue, Olen rejection has flavor text, reaction delays are natural)
