# Test Report: New Player Experience Session

**Date:** 2026-05-15
**Target:** prod (dogmud.org:55555)
**Role:** feel-tester
**Character:** aitester (freshly recreated after `deletecharacter` wipe)
**Goals file:** none (general NPE evaluation)
**Duration:** ~55 minutes, ~265 commands sent

## Session Summary

After wiping the existing `aitester` account via `deletecharacter` and walking
through fresh account creation, I role-played a true first-time player from
character birth in the Awakening Rite through tutorial completion and into
the wider world. The Sanctum Trials (6-step intro quest) was clear,
discoverable, and well-paced; I completed it in ~25 minutes. I then took
on the Warden's Report (bandit camp quest) without enough preparation, got
overwhelmed by 2-bandit ambush, died, respawned, recovered gear, and
explored more of the basin. Overall the NPE is strong on writing,
atmosphere, and onboarding scaffolding (hint/quests/help systems are
excellent), but suffers from one significant fairness issue (post-tutorial
difficulty cliff combined with unreliable flee) and a handful of polish
bugs.

## New Player Experience Timeline

### First 5 Minutes (~16:53-16:58)
- Awakening Rite plays out cinematically — the Chrysalis Priest narrates
  the lore while granting 10 starting gold AND the first quest step in one
  motion. This was excellent — no menu, no dump of info, just a scene.
- Quest immediately gives a `Type hint for directions.` prompt, and `hint`
  delivers a precise routing instruction (`From the Academy Hall, go south
  (1 south).`). I felt oriented within 30 seconds.
- The `mosaic map` on the floor of Academy Hall gives an in-fiction world
  map — beautifully done.
- `status` reveals character is "Awakened novice generalist" with a mutation
  already (`fast-reflexes`). Nice immediate hook of the "you're already
  changing" theme.

### First 15 Minutes (~16:58-17:08)
- Bought basic gear with starter gold (sharp stick, cotton shirt, wooden
  shield, rusty pot, small red potion) — Merchant Adela gives in-character
  prompts every couple of rounds. Felt guided, not bossy.
- `wear all` equipped everything in one command — small but huge for new
  player friction.
- Combat trainer + training dummy fight took noticeably long (~12 rounds)
  but during the fight the trainer NPC narrated tactical advice
  ("try kick, grapple, or trip", "shield bash available next", "try taunt").
  This was masterful — it converted dead time into a tutorial.
- By minute 12 I had killed the dummy and was crafting (iron dagger, then
  healing salve at the alchemist — salve fizzled, but Yenna immediately
  explained `salvage` for failed brews). Failure was educational.

### First 30 Minutes (~17:08-17:23)
- Tracking the meadow lizard took 3-4 tries because of skill checks
  ("Your tracking skills aren't sharp enough right now."). My search skill
  reached apprentice during these attempts. Eventually got the track.
- Cast spell on Chrysalis Echo at the Observatory — first cast fizzled,
  second worked. Elder Saris narrated through it.
- The cave system (Cave Path → Cave Depths → Boss Cave) was a satisfying
  micro-dungeon with 3 fights of escalating difficulty. The aberrant
  Chrysalis boss had a great descriptive intro paragraph that broke from
  the normal room template ("The air in this chamber is wrong...").
- Quest complete at ~minute 30. I felt rewarded — title bumped me to
  #8 on the power leaderboard automatically.

### First 60 Minutes (~17:23-end)
- South of the gate into Dustwalk Road, the difficulty cliff hit hard.
  Open-world mobs (scrubland dog, scavenger bird, dustwalk bandits)
  spawn 2-up in many rooms, regularly grapple, and hit much harder than
  tutorial mobs. New character with starter gear got to 10% HP fast.
- Picked up the Warden's Report quest from Road Warden Tessara — clear
  ask (scout bandit camp, collect evidence). Found the bandit camp.
- Got blocked from fleeing by bandits multiple times; `flee` had no
  obvious direction respect (typed `flee north` and was dropped SOUTH
  into the Bandit Hollow with 2 bandits anyway).
- Died once at Dustwalk Scrubland Stretch. Respawn was forgiving —
  full inventory kept, modest stat penalty (lost some Dex training,
  some bartering rank), woke up at Academy Hall under sanctuary regen.
  The "Darkness swallows you" death flavor is great.
- Spent last 10 minutes regenerating, exploring east meadow (valley rats,
  cave bats — much more manageable), foraging for healer's root (got
  one, plus a bitter thistle from West Meadow), discovering the
  Labyrinth of Low Tunnels (Blind Tunnel Rat → Tunnel Shaman — clearly
  a non-starter zone for a fresh character).

## Findings

### PASS: Onboarding tutorial design
The Sanctum Trials quest chain is exemplary. Each step (visit X, do Y) is
geographically contained, fed by `hint`, and the destination NPC narrates
the next mechanic IN-FICTION while you do it. By the end of 6 steps a new
player has bought, equipped, fought, crafted (smithing AND alchemy),
tracked, cast a spell, and killed a boss. That's a complete genre
introduction without a single info dump.

### PASS: Discoverable NPC dialogue
Almost every NPC offers a bracketed `[Try: ask priest chrysalis, ask priest
awakening, ...]` hint after `talk`. This solves the classic MUD problem
of "what are the keywords?" Players never have to guess. The keyword set
is also wide enough to invite roleplay exploration.

### PASS: Player-facing language quality
No raw numbers in combat output ("light wounds", "negligible damage",
"badly wounded"). Stats shown as "modest / average / keen" rather than
numeric. Encumbrance as a colored tier. The fiction stays intact. This
is a strong design discipline and clearly enforced.

### PASS: Death penalty calibration
Death is forgiving enough that exploration feels worth the risk: full
inventory retained, modest training loss, sanctuary regen on respawn,
respawn-immunity window described by the Warden. This makes the death
educational instead of punitive.

### PASS: Helpful in-game tip system
Periodic `[Tip] ...` messages surface advanced features as you play
(companions, dismiss, elementals). These are well-paced — frequent
enough to inform, infrequent enough not to spam.

### PASS: Atmospheric writing
Room descriptions, NPC speech, and combat text consistently land. The
Aberrant Chrysalis boss room break-from-template ("The air in this
chamber is wrong...") was the strongest moment. Elder Saris's lore
("The Fold came like a silence...") evokes setting without exposition
dumping.

### CONCERN: Post-tutorial difficulty cliff
Walking south from Basin Gate into Dustwalk Road, the first mob I met
(scrubland dog) is roughly even with a fresh post-tutorial character;
but rooms spawn 2 hostile mobs simultaneously, and bandits in the camp
spawn 2 per room as well. With starter gear (iron dagger, wooden shield,
cotton shirt, rusty pot) a player rapidly drops to 10% HP. There's no
clear intermediate-difficulty zone between the Sanctum tutorial and the
wider world. Recommendation: either (a) lone-mob spawns in the first 2-3
rooms south of the gate, or (b) a more explicit warning on the Basin
Gate sign that "you are not ready to go south alone yet — find a party."

### CONCERN: Flee direction unpredictable
`flee` (with or without a direction) sent me randomly between exits
multiple times in the bandit camp. I typed `flee north` while in the
Bandit Hollow and ended up moving... still in Bandit Hollow / back to it
on the next attempt. With only one exit (north) this was especially
confusing. Recommendation: when only one exit exists, flee should go
that direction deterministically. Also `flee <direction>` arguably
should honor the direction when blocked-multiple, retry-then-fail rather
than choose another exit.

### CONCERN: Hint-promised dialogue branches that don't deliver
Multiple NPCs offer follow-up hints in brackets that don't trigger.
Examples:
- Korvath: `[Ask about swords or axes specifically, or about what he
  charges.]` → `ask korvath swords` and `ask korvath axes` both repeat
  the parent line verbatim. `ask korvath charges` returns "Got work to
  do. Make it quick."
- Elder Saris: `ask saris fold` says "The Fold came like a silence. One
  moment the world was familiar" — sentence ends with no terminating
  punctuation. Likely truncated.
- Elder Saris: `ask saris companions` ends with "The schools differ" —
  also no terminator.
This breaks the discoverability contract — the bracketed prompt makes a
promise the dialogue tree can't keep.

### CONCERN: `ask priest quest` does not surface quest info
Per the project's quest-NPC dialogue SOP, every quest-granting node
should include `"quest"` and `"task"` in triggers. The Chrysalis Priest
holds the active Sanctum Trials quest but `ask priest quest` and `ask
priest task` both fall through to a generic "Speak your question clearly"
response. New players who try `ask <npc> quest` to find work would not
discover the priest is questing them.

### CONCERN: Yenna's dialogue claims she's a shop but isn't
`ask yenna potions` → "You can buy reagents, simple potions, and an empty
bottle or two..." but `list` at her room says "Visit a merchant to list
and buy objects." She's not a shop. The dialogue lies, which is more
frustrating than no info at all. (She did auto-give materials on the
first salve attempt; perhaps the design is "if you fail, you have to
forage for more" — fine, but the dialogue should say so.)

### CONCERN: Tracking quest has stale-information UI noise
After tracking the meadow lizard, every subsequent room print included
the line `You lost the trail of meadow lizard` for the rest of the
session — across rooms, after deaths, after quest completion. This
visual clutter looks like an error and should expire once tracking is
no longer active.

### CONCERN: Combat in the open is very chatty / verbose
Tutorial fights took 10+ rounds against a "novice" dummy. Every round
prints multiple lines per swing including riposte / parry flavor text,
companion notes from the trainer, and (occasional) skill-up lines. For
a new player this is fine because the trainer fills the time with
tutorial chatter, but in the open it becomes spammy and HARD to track
what's actually happening. Especially with multiple mobs, the screen
fills before you can react.

### OBSERVATION: Health bars use 10 # characters but mobs show only word descriptors
Player HP bar is `[HP:########## ...]` — a 10-segment ASCII bar that
ticks down clearly. But mob health is `[|healthy]` → `[|bruised]` →
`[|wounded]` etc. with no proportional bar. After 15+ rounds against
the training dummy still seeing "healthy" was disorienting — I thought
I was dealing no damage. A visible progress hint (even just a
20-character bar shown via `consider`) would let players gauge combat
flow more naturally.

### OBSERVATION: `wear all` works beautifully across multiple item types
Mixed-type bulk equip (`wear all` equipped weapon + shield + body +
head from a single command) was a quality-of-life win. Cross-system
parsing seems to know which slot each item wants.

### OBSERVATION: Title system reacts to playstyle silently
Sometime between buying a shield+weapon and killing the boss, my title
went from "novice generalist" to "novice guardian" without notification.
I only saw it on the leaderboard. A small `*** Your title is now
Awakened novice guardian! ***` line would make it feel earned.

### OBSERVATION: Salvage RNG harsh at novice rank
Salvaged 2 corpses (scavenger bird → 1 leather strip; valley rat →
nothing). Salvage gating felt right (skill check makes sense) but
returning literally zero materials feels bad. Maybe guarantee a
"trash" recovery (bone scrap, gristle) so the time investment isn't
zero.

### BUG: Audio MUSIC/SOUND directives bleed into text stream
Every shop purchase prints raw escape codes:
`��Z!!SOUND(static/audio/sound/other/buy.mp3 T=other V=100)��You purchase...`
Same for combat hits (`hit-other.mp3`, `hit-self.mp3`, `miss1.mp3`),
movement (`room-exit.mp3`), and the initial MUSIC directive. These are
GMCP/MSDP audio packets intended for graphical clients. A plain telnet
client (or any client that doesn't strip them) sees garbage characters.
Common bridge/client filters strip them but `mud_bridge.py` does not.
Either the server should not emit them on plain telnet, or the bridge
needs a filter pass. (Note: the bridge is a tester tool, but a real
new player on a plain telnet client would see the same garbage.)

### BUG: `exits` command listed in help but unrecognized
`help` lists `exits` under Information. `exits` returns "exits not
recognized. Type help for commands."

### BUG: `species` command listed in help but unrecognized
Same pattern: shown in `help`, returns "species not recognized" when
typed.

### BUG: `stash` with no argument prints "You don't have a  to stash"
Empty noun placeholder leaks into the error message — should either
list stashable items or say "Stash what?"

### BUG: Quest hint phrasing math is off by one
For the Alchemist step, hint said: `go west, west, north, west (3 west,
1 north)`. The path is actually 2 west + 1 north + 1 west. The summary
in parens disagrees with the explicit step list.

### BUG: Town Square signpost mentioned in description but not interactable
Town Square description: "A carved sign post stands at the center, hung
with a painted map of the basin and its facilities." All of `look sign`,
`look signpost`, `look post`, `look map` return "Look at what???". The
signpost is a great affordance promise the room can't deliver.

### BUG: Cross-room admin notification bleed
While crafting at Korvath, line `*** Megalomania has DIED! ***` printed
in my room scroll. Admin player death broadcasting to non-party,
non-witness players seems wrong — at minimum it broke immersion in a
peaceful scene.

### BUG: East Meadow description claims south exit that doesn't exist
"The land drops toward the cliff edge to the south and rises toward
denser scrub to the north." Exits: north, west. There IS a southern
land feature (the south cliffs / cliffs overlook reached via Basin
Gate → west), so the prose isn't lying, but a new player reading the
description WILL try `south` and bump into walls.

## Raw Stats

- Commands sent: ~265
- Fights: ~12 (training dummy x1, chrysalis echo x1, cave bat x2,
  cave goblin guard x1, aberrant chrysalis x1, scrubland dog x2,
  scavenger bird x2, valley rat x2, dustwalk bandit x1 partial)
- Deaths: 1 (in Dustwalk Scrubland Stretch, blocked-flee + dog)
- Spells cast: 3 (conviction-spike x2, chrysalis-glow x1)
- Items used: 4 (1 starter potion drunk in combat, 3 potions bought
  for next attempt, 1 wear-all equip)
- Quests started: 2 (The Sanctum Trials COMPLETE, The Warden's Report
  in progress)
- Skill ranks gained: bartering→apprentice, search→apprentice,
  salvage→apprentice, weapon-combat→journeyman
- Stats gained: Dexterity x3, Strength x3, Willpower x1, Perception x1
- Bugs found: 7
- Concerns: 7
- Observations: 4
- Passes: 6

## Headline NPE Takeaway

The Sanctum Basin tutorial is one of the most thoughtful MUD onboarding
experiences I've encountered — discoverable, in-fiction, well-paced, and
ends with the player genuinely competent in 6 systems. The cliff at the
basin gate is the single biggest gap: a fresh tutorial graduate is one
two-mob ambush away from a respawn, and `flee` is unreliable when it
matters most. Add an intermediate buffer zone and tighten flee, and the
NPE goes from "good" to "great."
