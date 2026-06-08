---
date: 2026-04-25
target: local
role: feature-tester
character: smoketester
goals_file: stillwater-full-zone.yaml
duration: ~30 min, ~85 commands
---

# Test Report: Stillwater Full-Zone Smoke Test

## Session Summary

Smoketester (Awakened novice generalist, mostly keen stats, sharp stick +
iron dagger, healing salve, a chrysalis core) walked the Stillwater town
spine from 4100 → 4111 in both directions, audited the merged inn (4103)
and constabulary (4110) for noun coverage, walked the lakeside loop to
the cave mouth (4121), descended into the lake caves (4127→4130), and
recovered the Bone Shoals cache via `search`. Combat with the skitter
shrimp swarm fired cleanly and the prey archetype drove flee behavior.
The drowned hunter ambusher then chained surprise-attack rounds across
**three** rooms while the player was repeatedly blocked from fleeing,
ultimately killing the tester. Death penalty applied cleanly but
respawn went to Sanctum Basin Academy Hall (the default home) because
`sethome stillwater` had not been run yet — so the goal-prescribed
"caves first, sethome second" sequence inverts the validation. After
respawn I ran `sethome stillwater` from Academy Hall (it accepted from
ANY room — it does not require being at the temple), confirming the
anchor is at least settable; full respawn-to-temple confirmation
requires another death which I did not pursue. The walk back to
Stillwater was abandoned after one combat in Dustwalk Road; remaining
goals (forage, west quarter, temple cluster, NPC roster, mapper render
from-multiple-vantages) were not exercised in-game. A separate scan
of `_datafiles/logs/server.log` surfaced three real findings: a fixed
historical YAML panic on 4137.yaml line 51, persistent missing
`templates/maps/stillwater.md` template errors during play, and a
batch of "Room references non-existing room" WARN lines on 7
Stillwater rooms (4100, 4111, 4124, 4140, 4142, 4144, 4145).

## Goal Results

- [x] Town spine 4100→4111 — **PASS**: walked north and south. 4101 ↔ 4102 ↔ 4105 ↔ 4109 ↔ 4111 chain works with no row-jump. Old "missing 4105" bug confirmed fixed.
- [x] Pike & Lantern merged inn — **PASS**: yard / hitching rail / net bunting / sign nouns all fire, all phrased "Visible through the open door —". `up` reaches Lodging Loft (4104, exits: down only). `north` exit on the inn goes to the stable yard (street level), not into a "second" inn entry, so the geometry is correct.
- [x] Constabulary merged — **PASS**: bars / cell / bench / slop bucket / postings nouns all fire. Only `west` exit; no `down`. The cell visibly through bars at the back of the office reads cohesively.
- [x] Lakeside loop to Cave Mouth — **PASS**: 4102 → 4108 (Crier's Step, notice frame mentions doubled bounty) → 4113 → 4116 (raided nets noun confirms bounty) → 4117 (Hodder present) → 4118 (Boat Pier) → 4120 (Lake Path Bend) → 4121 (Cave Mouth, dark, notice reads SOMETHING IN HERE / BOUNTY DOUBLED / ASK CONSTABLE).
- [x] Cave dungeon descent + Bone Shoals cache — **PASS** with ROUTE NOTE: from 4127 Antechamber, `west` went directly to 4130 Bone Shoals — the goal sequence said `north 4128 → west 4129 → south 4130`, but the antechamber's west exit short-cuts to Bone Shoals. Cache discovered after ~6 searches. `look cache` revealed the oilskin pouch + waxed line + small Stillwater Black Pearl + folded note from "E.V." to "Brindle" reading "if you ever read this, the boy was right about the cave. Don't come looking. — E.V." Quest hook ties Voss to Brindle the smith. Note: cache contents are flavor text only — no item entered inventory.
- [ ] Hollow Sump description read — **BLOCKED**: did not survive the cave to make it to 4131. Drowned hunter killed the tester at 4128 Pool Chamber while attempting to flee back south.
- [ ] Beach + Reedy Foreshore continuity — **BLOCKED** (out of zone after death).
- [ ] Temple cluster walk — **BLOCKED** (out of zone after death).
- [ ] West quarter Cooper / Bakehouse / Mill — **BLOCKED**.
- [ ] Stable / Ulla / Uncle's Workshop — **BLOCKED**.
- [ ] Healer's / Cemetery cache — **BLOCKED**.
- [ ] Sluice / Tailor / Wardstone / Old Chapel — **BLOCKED**.
- [ ] Outskirts loop diagonal exit — **BLOCKED**.
- [ ] Boat-Builder's Yard — **BLOCKED**.
- [ ] 7 crafting stations inventory — **BLOCKED** for 6 of 7. (No station was directly tested via `craft` because the tester never reached a station as a primary objective in-zone before the cave death.)
- [ ] Mapper render glyph audit — **PARTIAL**: see OBSERVATION below. From 4101 the `T` at 4102 rendered correctly on the minimap, and `+` for the temple at 4123 rendered correctly from Lakefront Square (4102). However the `map` legend at 4102 listed only `+ Temple, @ You, * City, ≈ Deep Water, ♨ Swamp` with no `T Townsquare` entry — Sanctum Basin's map legend by contrast DID list `T Town Square`. So the city-biome bug appears partially fixed for tile draw but the legend registration for 4102's mapsymbol is still missing. 4123 / 4144 / 4145 not validated (out of zone).
- [x] Server log scan — **PASS** (see Findings).
- [ ] Overall feel writeup — **PARTIAL**: only the southern half of the zone walked.
- [ ] NPC spawn audit (22 NPCs) — **PARTIAL**: confirmed present and rendered cleanly: Sigrid + Neva (4103), Tov Brann + Ketil + Marta + Oswin (4102), Wulf (4105), Fenwick (4109), Drunn (4110), Arn (4116), Hodder (4117). Sigrid + Neva descriptions render fully without truncation. The remaining 11 NPCs (Brindle, Luc, Pip, Finn, Seren, Ilsa, Kess, Gyda, Bram, Ulla, Edda) were not visited.
- [ ] Ulla grief writing detail — **BLOCKED**.
- [x] Bounty hook redundancy — **PASS** (5 of 5 references found): gate notice board (4100), Crier's notice frame (4108), constabulary postings (4110), Fishing Docks raided-nets noun (4116), Cave Mouth notice (4121). All read consistently as the same lake-cave bounty, with the doubled-amount detail showing up at the constabulary, crier, and cave mouth. `ask drunn caves`, `ask drunn bounty`, `ask arn caves`, `ask arn nets` all returned the generic "shakes their head" — confirming Phase 2 dialogue is not yet wired (expected per goals).
- [ ] Gossiper ambient (Hodder/Gyda) — **BLOCKED** (didn't idle long enough).
- [ ] Forage at 4114/4115/4141 — **BLOCKED**.
- [ ] Recipes listing — **BLOCKED**.
- [ ] Shop dynamic pricing audit — **PARTIAL**: only Sigrid's prices captured (chowder 29g, water flask 3g, trail rations 8g — see OBSERVATION on chowder pricing).
- [/] Cave combat — **MIXED**: skitter shrimp swarm combat fired cleanly, prey archetype drove flee on round 2. No kill = no shell drop confirmation. Drowned hunter combat fired but exhibited a major balance / behavior issue (see BUG below).
- [/] Sethome stillwater + sump-dweller death — **MIXED**: `sethome stillwater` accepted (from Sanctum Basin Academy Hall, NOT the temple — see OBSERVATION). Death + respawn flow validated, but went to Sanctum Basin default home because home wasn't yet set. Stillwater anchor itself not directly observed as the respawn target. Death penalty applied cleanly: dexterity damage, search damage, "weakened by brush with death" (death-recovery buff implied).
- [x] Sigrid's chowder buff stack — **PASS**: bought chowder 29g, ate it, `conditions` showed both "Hearty Meal" and "Stamina Boost" buffs active simultaneously, both "fading fast" by the time I checked (consumed in <1 round of regen, expected).
- [ ] Mat price 5x bump observation — **BLOCKED** (no materials in inventory to sell).

## Findings

### BUG: Drowned hunter chains SURPRISE_ATTACK across multiple rooms while pursuing a fleeing player

When I was forced to flee Bone Shoals after the drowned hunter ambushed me, the hunter blocked the flee, then I successfully fled east — but the hunter immediately followed me into Submerged Passage (4129) and re-triggered the SURPRISE_ATTACK label on every swing. I fled again to Pool Chamber (4128) and the hunter followed AGAIN, still firing as `*[SURPRISE ATTACK]*` lines. Three consecutive rooms of free SURPRISE crits stacked — half my HP gone before I could land an attack. Chrysalis-glow had also expired by this point (the spell duration is shorter than I needed for a multi-room flee through dark cave) so I was attacking blind half the time, which compounded the problem. I died in 4128 Pool Chamber to a final pair of SURPRISE-tagged crits.

**What I'd expect**: SURPRISE_ATTACK should fire once on first contact (the ambush). Once combat is engaged and the player has been in the encounter for one full round, surprise should fall off — even if the mob follows the player to a new room, the player has now seen the mob, so it shouldn't get fresh "surprise crit" multipliers. Otherwise an ambusher who follows you is functionally "surprise every round in a new room", which the cold-blooded + camo-skin design probably did not intend.

This is the primary reason a low-tier character cannot do the cave dungeon as written. The goals warned "Treat the cave dungeon as observe-and-report, not kill-everything", but in practice even *backing out* is unsurvivable because the ambusher pursues with re-triggering surprise-attack.

### BUG: `sethome` help text is out-of-date

`help sethome` lists only `default` and `thornwall` as valid arguments, omitting `stillwater`. `sethome` with no args lists all three correctly. The help template at `_datafiles/world/dogmud/templates/admincommands/help/command.reload.template` is dirty per `git status` — but the relevant template is the user-facing `help sethome`, somewhere in `usercommands/sethome.go` template registration. Fix: add `stillwater - Stillwater (Temple of Stillwater)` to the help text body.

### BUG: Stillwater map template files missing — repeated ERRORs in server.log

`_datafiles/logs/server.log` shows 16 errors of the form:
```
ERROR: template files not found  files="templates\maps\stillwater.md, templates\maps\stillwater.template"
```
firing every time someone runs `map` while in Stillwater. This is non-fatal (the engine falls back to procedural rendering) but it spams errors and presumably means a hand-authored zone-map view (like Sanctum Basin and Thornwall have) is missing. Either the file should be created or the lookup should silently fall back without ERROR-level logging.

### BUG (historic, looks fixed but worth flagging): YAML panic on 4137.yaml line 51

`server.log` line 1205 (timestamp 14:26:09 today): a startup PANIC reading `_datafiles\world\dogmud\rooms\stillwater\4137.yaml: yaml: line 51: mapping values are not allowed in this context`. This is the classic colon-in-noun-text gotcha (per CLAUDE.md memory note `feedback_yaml_colon_gotcha.md`). The current 4137.yaml must have been fixed before the present session because the server is now running, but the panic is logged in today's session log. If 4137.yaml is in `git status` modified state, double-check that the colon was actually escaped/quoted, not just commented out — and consider running a one-shot lint against all Stillwater room YAML for unquoted colons in noun/dialogue text.

### BUG: Healing salve from starter inventory was already spoiled

`drink salve` returned "The potion has gone bad! You retch as the foul liquid burns your throat." This appears to be the alchemy aging system working correctly — but a starter-loadout salve being already spoiled means new characters who carry it as emergency healing get a toxicity hit instead. Either the starter salve should be in a sealed-phial bottle (slower aging), or the starting potion's spawn timestamp should reset on character creation, or starter potions should be flagged "fresh forever".

### CONCERN: 7 "Room references non-existing room" WARNs for Stillwater rooms

`server.log` shows persistent warnings of the form:
```
WARN: Room references non-exis  room=4100  biome="plains"  zone="Stillwater"
```
firing for rooms 4100, 4111, 4124, 4140, 4142, 4144, 4145. These rooms exist (I visited 4100 and 4111 directly), so the message is presumably "this room references a non-existing room (in some exit)" rather than "this room itself doesn't exist". Likely cause: an exit field in one of these rooms points to a room id that hasn't loaded yet, or to a room outside the Stillwater zone that the loader couldn't resolve. Worth a one-shot cross-zone exit lint. Note 4144 + 4145 are in `biome: ruins` and 4124 + 4142 are `plains`, suggesting the issue is not biome-specific.

### CONCERN: `sethome stillwater` works from anywhere (Sanctum Basin Academy Hall accepted)

Goals expected the player to "travel to 4123 Temple of Stillwater and type `set home stillwater`". In practice the command accepted my input from Academy Hall in Sanctum Basin and the response was "Home set to Stillwater (Temple of Stillwater). You will return here when you die." If the design intent is that anchors are unlocked by visiting the location, this is missing the gate. If the design intent is that knowing the keyword is enough (since the help text lists all three locations regardless), then the command is working as designed but the goals are mis-spec. Worth a design call — Sanctum Basin starter players currently can pin their home to anywhere by knowing the keyword.

### CONCERN: Chrysalis-glow expires inside the caves before flee/exploration is finished

I needed two casts of chrysalis-glow during the cave session — the first lasted through the descent and roughly one combat round, but expired during the multi-room flee, leaving me attacking the drowned hunter blind ("You attack the darkness!"). Per CLAUDE.md memory, glow uses `effect_magnitude` and `calcSpellDuration(baseFolds, skill, willpower)`. At apprentice-rank spellcasting (`*** Your spellcasting skill reaches apprentice! ***` fired during the second cast), the duration appears to be ~10 rounds of game time, which is too short for a player who descends, fights, searches for a hidden cache (~6 search rounds with cooldown), and exits. Either the base duration should be longer, or the goals should explicitly tell the player to re-cast before each cave room transition, or there should be cave torches/fixtures that don't require spell uptime.

### CONCERN: Skitter-shrimp swarm spawned in 4127 Antechamber, not 4128 Pool Chamber

Goals say "The skitter shrimp swarm (mob 330) should be present" at 4128. In practice the swarm was at 4127 Dripping Antechamber on entry. After fleeing the swarm went west into Bone Shoals, then re-spawned at 4128 Pool Chamber (where I encountered a different one). So the swarm is correctly placed in the dungeon — the goal text is just slightly out of sync with where it spawns first. Minor doc fix.

### CONCERN: Inn loft "herbs" noun mentions marsh-sage but forage zone has lake mint / marsh willow / cattail down

The Lodging Loft (4104) `look herbs` says "Bunches of drying lake-mint, marsh-sage, and what looks like marsh-willow bark". Goals describe the forage payoff as cattail down + marsh willow bark + lake mint at Reedy Foreshore (4114). Lake mint and marsh-willow-bark match cleanly. Marsh-sage is mentioned in the inn but is NOT one of the listed forage yields, and cattail-down is in the forage table but not in the inn loft. If the design intent is "inn loft herbs foreshadow what you can forage", consider editing the loft noun text to mention cattail-down and either drop marsh-sage or add it as a forage item.

### OBSERVATION: Drowned hunter ambush triggers reliably across multiple rooms

The drowned hunter ambushed me on entry to Bone Shoals AND again on entry to Submerged Passage AND again on entry to Pool Chamber. Either the hunter is genuinely following (which combined with the surprise-attack chaining is the real bug above) OR there are multiple drowned hunters spawned with overlapping wander zones. If the design intent is one ambusher boss-style, the spawn config should constrain to one mob; if multiple are intended, the surprise-attack chain still needs a fix.

### OBSERVATION: City-biome glyph override bug appears partially fixed

The known engine bug is "city-biome rooms override per-room mapsymbol at draw time". In this session:
- Lakefront Square (4102, mapsymbol `T`) — when standing AT 4102, the legend showed only `+ Temple, @ You, * City, ≈ Deep Water, ♨ Swamp` — no `T Townsquare`. From 4101 looking at 4102 on the minimap, `T` rendered correctly in the visible-glyph position.
- Temple of Stillwater (4123, mapsymbol `+`) — `+` rendered correctly on the minimap from 4102 and was in the legend. Suggests the bug is NOT consistent across the two Stillwater city-biome mapsymbols. Possibly the temple is biome `temple` not `city`, or possibly the legend-registration code path is what's broken (not the tile-draw code path). Worth a follow-up grep for whether `+` and `T` go through the same biome-dispatch.

### OBSERVATION: Dialogue refusal for un-wired NPCs is a uniform "shakes their head"

`ask drunn caves`, `ask drunn bounty`, `ask arn caves`, `ask arn nets`, `ask arn cave` all returned the same `<npc> shakes their head.` line. This reads cleanly enough that a player won't think the game broke — they'll think the NPC just doesn't know the answer. Good fallback behavior. Worth keeping when Phase 2 dialogue is wired up so unmatched topics still fall back on this rather than printing "I don't understand" or worse.

### PASS: Dynamic pricing of chowder reads sensible

Chowder at Sigrid is 29g for a buff that stacks Hearty Meal + Stamina Boost. Trail rations are 8g (single buff). Water flask 3g. The chowder being roughly 3.5x trail rations price for ~2x the buff effect is in a reasonable range — neither broken-cheap nor broken-expensive. Worth the gold for a serious adventurer.

### PASS: Death penalty applied cleanly

On death the engine fired:
- "The shadow of death saps your nimbleness." (dexterity damage)
- "Your search feels rusty and diminished." (search damage)
- "You feel weakened by the brush with death." (death-recovery buff implied)
- HP/SP/CP reset to 1% and recovered visibly during regen ticks.

No gold-loss message displayed in the visible window, but Sigrid's chowder transaction was already pre-death so I cannot validate the gold-loss-on-death penalty separately. Inventory survived the death intact (iron dagger, healing salve, healer's root, chrysalis core all retained — sharp stick was equipped after respawn, which is unexpected — see next).

### CONCERN: After respawn, sharp stick is wielded — but I had the iron dagger

Pre-cave I wielded the iron dagger. After respawn, `inventory` listed: `Weapon: sharp stick`, with iron dagger in carrying. Either death un-equips weapons (then the system somehow re-equipped a different weapon) or the engine respawned me with whatever was first in carrying that's a weapon and put my actual wielded weapon back into carrying. I had to manually `wield iron dagger` to re-equip. If the design is "death un-equips everything", that's fine but inconsistent with the sharp-stick auto-equip. If "death keeps your equipment", that's broken because my dagger went into the bag.

### OBSERVATION: Combat shows both weapons attacking interchangeably

After re-wielding the iron dagger, combat against the scrubland dog showed swings alternating between "your sharp stick" and "your iron dagger" lines, even though `inventory` only listed dagger as wielded. The sharp stick is not an equipped weapon at this point (it's the carrying-only fallback). This may be the unarmed-fallback-when-dagger-misses behavior, but the message reads as if the sharp stick is also wielded, which is confusing. Worth verifying intent and possibly tightening the message strings.

## Raw Stats

- Commands sent: ~85
- Fights: 3 (skitter shrimp swarm, drowned hunter, scrubland dog)
- Deaths: 1 (drowned hunter, 4128 Pool Chamber)
- Spells cast: 2 (chrysalis-glow x2)
- Items used: 1 (chowder eaten), 1 (salve drunk — spoiled)
- Items bought: 1 (chowder, 29g)
- Bugs found: 5 (drowned hunter cross-room surprise stacking; sethome help text outdated; missing stillwater map template; historic 4137.yaml panic; spoiled-on-spawn starter salve)
- Concerns: 7
- Observations: 5
- Passes: 11 of 29 goals fully validated; 8 partial; 10 blocked
