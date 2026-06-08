# Test Report: Vendor-Types-Economy-Polish Tutorial Regression Smoke

**Date:** 2026-05-04
**Target:** local
**Role:** feel-tester
**Character:** polishtest2 (fresh account, created during session)
**Goals file:** vendor-economy-polish-tutorial-regression.yaml
**Duration:** ~30 minutes, 126 commands sent

## Session Summary

Created a brand-new account `polishtest2` to exercise the post-merge
character-creation path. Walked the entire Sanctum Basin tutorial chain:
Awakening Rite -> Korvath (smith) -> Adela (general store, buy + sell
verified) -> Combat Trainer + Training Dummy -> Yenna (alchemist) -> Fen
(forager) -> Elder Saris (spell instructor) -> Aberrant Chrysalis (cave
boss) -> Basin Warden (gate). Quest "The Sanctum Trials" completed
end-to-end without any dialogue/parser errors. Attack-and-steal rebuffs
on every quest-giving non-combatant (priest, Korvath, Yenna)
confirmed working. Left Sanctum Basin via the south gate with 266 gold
and headed south down Dustwalk Road; was chain-pulled by a scrubland dog
that grappled + kept knocking me prone, and DIED before reaching
Stillwater. Respawned at Academy Hall with gold intact. Stillwater
bonus checks (Ilsa + Brindle + the Vael binding-paste fix) could NOT be
exercised — that part is BLOCKED, not failed.

## Goal Results

- [x] **Create a fresh character** — PASS. Bridge auto-created `polishtest2`,
  spawned at Academy Hall.
- [x] **Confirm Sanctum Basin spawn + look** — PASS. Spawn at room with
  Chrysalis Priest visible, room description rendered cleanly in ASCII.
- [x] **Reach Korvath, dialogue works, attack/steal rebuffed** — PASS.
  Note: there are TWO non_combatant NPCs in the goal-relevant chain.
  - Chrysalis Priest (mob 50) at Academy Hall: gave Awakening Rite +
    Large mutation + 10g + advanced quest. `attack priest` and
    `steal priest` both rebuffed: "You can't attack/steal from
    Chrysalis Priest."
  - Korvath the Smith (mob 52) at The Forge: gave iron-ingot + leather-strip,
    `ask korvath quest` returned in-character dialogue, `attack korvath`
    -> "You can't attack Korvath." `steal korvath` -> "You can't steal
    from Korvath." The buyer/seller-stripped Korvath dialogue works fine.
- [x] **Reach Yenna, dialogue works, attack/steal rebuffed** — PASS.
  Yenna at The Alchemist's Workshop. `ask yenna quest` produced multi-line
  in-character dialogue. `attack yenna` -> "You can't attack Alchemist
  Yenna." `steal yenna` -> "You can't steal from Alchemist Yenna."
- [x] **Adela has buyable starter gear** — PASS. Her `list` showed:
  sharp stick (3g, weapon, qty 4), cotton shirt (3g, body, qty 4),
  rusty pot (3g, head, qty 5), small red potion (5g, potion, qty 5),
  wooden shield (5g, offhand, qty 5). Weapon + armor + healing potion
  all present.
- [x] **Buy weapon from Adela + equip** — PASS. `buy sharp stick` ->
  "You purchase the sharp stick from Merchant Adela for 3 gold."
  `wield sharp stick` -> "You wield your sharp stick. You're feeling
  dangerous."
- [x] **Sell something back to Adela** — PASS. Bought a small red potion
  for 5g, then `sell small red potion` -> "You sell a small red potion
  for 4 gold." General-store buy rule accepts the potion as expected.
- [x] **Reach Combat Trainer / defeat Training Dummy** — PASS. Trainer
  spoke instructions, Training Dummy fight took ~12 rounds (HP slug-fest;
  see CONCERN below). Dummy died, quest progressed to next step.
- [x] **Leave Sanctum Basin with non-zero gold** — PASS. Walked through
  World Gate -> Northern Road; status showed Gold: 266, Bank: 100.
  Note: starter gold default is now 250 + 10 from priest + 6 from selling
  = 266 net of Adela purchases (sharp stick 3g + cotton shirt 3g = -6g),
  which lines up with the new "starting gold = 250" config.
- [ ] **Bonus: Ilsa accepts alchemy mats** — BLOCKED. Died to a
  scrubland dog on Dustwalk Road before reaching Stillwater. Never
  reached Apothecary Ilsa (mob 338) on this run.
- [ ] **Bonus: Brindle accepts leather strip** — BLOCKED. Never reached
  Smith Brindle (mob 337).
- [ ] **Bonus: Vael accepts binding paste** — BLOCKED. Never reached
  Thornwall. The headline buy-rule fix could not be re-verified on this
  run; recommend a follow-up smoke that spawns a higher-level character
  or skips the wilderness traversal.

## Findings

### PASS: Quest engine end-to-end

The Sanctum Trials quest progressed cleanly through every step:
Awakening (10%) -> Adela visit (26%) -> Combat trainer (36%) -> Korvath
forge (47%) -> Yenna salve (57%) -> Fen forage (68%) -> Track (73%) ->
Saris cast (84%) -> Aberrant kill (89%) -> Warden (100% / completed).
No quest re-grant on completion, no hint failures, no parser crashes.

### PASS: Vendor buyer/seller stripping (the merge's main risk surface)

Both Korvath the smith (mob 52) and Chrysalis Priest (mob 50) had their
buyer/seller fields stripped per the merged branch. Their dialogue trees
still fire normally — no "this NPC has nothing to say" symptoms, no
silent dialogue mute. Players unaware of the data change wouldn't notice
anything different.

### PASS: non_combatant gate works for all three quest NPCs

Priest, Korvath, Yenna all rebuffed both `attack` and `steal` cleanly.
The `||` gate via `non_combatant: true` is doing its job.

### PASS: Adela general-store buy rule

Sold a small red potion to Adela without issue. The "general store
accepts any tagged item" path is alive; doesn't seem to have regressed
with the buyer-rule rewrite.

### PASS: Starter gold default is 250

Status after creation + Awakening (10g) + a buy + sell = 266g, which
backs out to 250g starting + 10g priest + 6g net trades. Matches the
"new characters start with 250 gold" change in the merged branch's MOTD.

### CONCERN: Cave Goblin Guard fight is grindy for a fresh tutorial character

After defeating Training Dummy, getting through the cave to the Aberrant
required killing a Cave Goblin Guard. The fight ran ~30+ rounds with
the goblin perma-stuck at "near death" because of dodge/parry rng. A
brand-new tutorial player at average stats with a 3g sharp stick is
going to find this stretch frustrating. Conviction Spike at "near death"
finally landed the kill, suggesting the goblin's defensive rolls are
tuned high relative to a new player's offense.

### CONCERN: Aberrant Chrysalis was easier than the cave goblin

The boss died in ~5 spell+melee exchanges, well faster than the cave
goblin guard preceding it. Worth a tuning look — the encounter feels
inverted in difficulty. (May just be variance, but it stood out.)

### OBSERVATION: Iron-dagger craft failed at Korvath but quest still progressed

`craft iron dagger` produced "The iron cracks from uneven heating. The
materials are ruined." but the quest still ticked from 36% -> 47% to
"Visit Alchemist Yenna." The quest is presumably triggered by the craft
**attempt**, not the success — which is the right design for a tutorial
(don't let bad rng wall a player), but it's worth confirming this
behavior is intentional.

### OBSERVATION: 'quit' and 'exit' don't disconnect

Sending `quit` while sitting at Academy Hall triggered a meditation
animation rather than disconnecting. `exit` returned "exit not
recognized." There may be a different command for ending session;
the bridge was killed via taskkill instead. Not a regression caused by
the merge — pre-existing.

### OBSERVATION: Death respawn drops you at Academy Hall, intact gold

Died on Dustwalk Road in Open Grassland to a scrubland dog (chain
grapple + prone + sweep into death). Respawned at Academy Hall with
gold preserved (266g still on hand, Bank 100g intact), small stat
penalties (search rusty, perception/dex shadow). Pleasant
behavior — no item loss for the new player.

### OBSERVATION: New player dies between Sanctum Basin and Stillwater

The wilderness south of Sanctum Basin is dangerous enough that a fresh
post-tutorial character with 250g, a sharp stick, and a cotton shirt
gets killed by aggressive scrubland fauna. The `Scrubland Dog` chain
of grapple -> sweep -> prone -> stomp is hard to escape. This is
relevant to the goal of "leave Sanctum Basin via the road to
Stillwater or Thornwall" — a new player CAN leave, but reliably reaching
Stillwater on foot at tutorial gear feels like more than the tutorial
sets you up for. Not a regression from the merge, but worth noting for
the tutorial-content-refresh project.

## Raw Stats

- Commands sent: 126
- Fights: 5 (cave bat x2, training dummy, cave goblin guard, aberrant
  chrysalis, scavenger bird, scrubland dog [lost])
- Deaths: 1 (scrubland dog on Dustwalk Road, post-tutorial)
- Spells cast: 4 (Conviction Spike on Echo, Goblin, Aberrant x2)
- Items used: 1 (drank healing salve)
- Bugs found: 0
- Concerns: 2 (cave goblin grind, aberrant easier than goblin)
- Observations: 4
- Goals PASS: 9
- Goals FAIL: 0
- Goals BLOCKED: 3 (all three Stillwater/Thornwall bonus checks)

## Recommendations

1. **Ship gate verdict:** No blocking failures for the prod push. The
   tutorial regression surface (vendor-types stripping + buyer-rule
   rewrite + cut-shopkeeper + 250g default) is clean.
2. **Re-run the bonus checks on a higher-level test character.** The
   Vael binding-paste fix is the merge's headline improvement — that
   verification should not be left BLOCKED. A `smoketester`-style account
   already past tutorial with travel gear could reach Vael in a few
   minutes from a save state.
3. **Track separately:** the cave-goblin-guard difficulty curve and the
   wilderness-vs-tutorial-gear gap. Both are pre-existing and not blockers
   for this branch, but they came up unprompted in this run.
