# DOGMud Patch Notes

## 2026-03-29 — Combat, Stealth & Spell Balance

### Kick Rework
- **Kick** is now a powerful unarmed strike (damage doubled from 0.40 to
  0.80). Three automatic variants based on combat position:
  - **Kick** (standing target): 35% knockdown chance.
  - **Stomp** (prone target): 1.20x damage, bypasses half armor,
    extends prone duration. The payoff for knocking someone down.
  - **Knee** (grapple, in control): 1.0x damage, works in clinch/ground.
- `stomp` and `knee` are command aliases for `kick`.

### Opening Fights with Special Moves
- Kick, bash, trip, grapple, and taunt can now initiate combat by
  naming a target (e.g., `kick bandit`). No longer requires attacking
  first.

### Stealth System
- Players now detect hidden mobs when entering a room via opposed
  Perception+Search vs Dex+Skullduggery roll.
- Rogue NPCs added: Blind Stalker, Pale Lurker, Warren Scout, Tunnel
  Lookout, and Goblin Scout spawn hidden and ambush on entry.
- Thornwall Highwayman, Smuggler Runner, and Torvan Cresk use tactical
  combat stealth.

### Caster NPCs
- Elder Saris, Priest Olen, Geomancer Rhett, and Windwarden Sylara now
  have spellbooks and cast buff spells while idle. Attack them and
  they fight back with appropriate magic.

### Buff/Ward Spell Rework
- **Shield spells** now scale by spell magnitude. Conviction Ward is
  75% strength (quick/cheap). Chrysalis Cocoon is 125% strength and
  grants magical/conviction mitigation. Both last 2.5x longer.
- **Iron Will** now provides magical and conviction damage mitigation
  alongside the willpower boost. Lasts 50 rounds (was 8). Costs more.
- **Chrysalis Haste** costs more but lasts twice as long.
- **Vital Surge** regen lasts 3x longer.
- **Empathic Shroud** no longer cancels on entering combat.
- **Veil Sight** now grants see-hidden (was incorrectly giving light).
- **Skill Attunement** and **Mutation Catalyst** last 10x longer but
  cost 3x more conviction.
- All debuffs (Nerve Disruption, Mind Fog, Sensory Overload, Psychic
  Anchor) last 50% longer.

### New Commands
- **reply** — Whisper back to the last person who whispered to you.
- **rep/report** — Broadcast your vital bars to the room, party, or
  a specific player.
- **setdesc** — Set your own character description.

### Stat Progression
- Taking a critical hit now triggers stat progression: physical crits
  improve vitality, magical crits improve willpower, rhetoric crits
  improve charisma.

### Balance
- Taunt damage +50% (RhetoricDamageScale 2.0 → 3.0).
- Spell damage -14% (SpellDamageScale 1.85 → 1.6).
- Subcomponent recipe discovery thresholds lowered: Steel Ingot 10→4,
  Chain Links 15→7, Chrysalis Setting 15→7.

### Other
- Spells list now sorted by fold count (simplest first).
- Leaderboard expanded from 10 to 20 entries.
- 4 new tailoring recipes: Leather Backpack, Reinforced Travel Pack,
  Artisan's Satchel, Master's Component Case.
- Component bag capacities increased (20/40/75).
- Apothecary Voss now sells alchemy ingredients.

---

## 2026-03-29 — Equipment Slot Expansion + Component Bags

### New Equipment Slots
- **Wrist** (x2) — Bracelets and bracers now have their own slots
  instead of using the ring slot. Existing bracelet items have been
  updated.
- **Shoulders** — Pauldrons, mantles, and shoulder armor.
- **Back** — Cloaks for stats, or backpacks that reduce the effective
  weight of your carried items. Choose wisely.
- **Second Ring** — Two ring slots instead of one.
- **Component Bag** — A dedicated bag for crafting materials. Materials
  auto-sort into it on pickup. Use `sort` to migrate existing
  materials. Buy a starter pouch from Weaver Maren in Thornwall.

### Extra Arms Mutation — Levels 3-4
- The Extra Arms mutation can now reach levels 3 and 4, granting up
  to four additional arms (six total weapon slots).
- Each extra arm comes with an extra wrist slot for bracelets.
- Higher levels impose steeper charisma penalties and aggro, with
  diminishing dexterity returns.
- Combat hit penalties scale: +20 per arm beyond your offhand.

### Component Bag System
- Crafting materials with the `is_component` flag auto-route to your
  component bag on pickup and purchase.
- The `sort` command moves existing materials from your backpack into
  the bag.
- Crafting consumes from the component bag first, then your backpack.
- Component bags reduce the effective weight of their contents.

### Bug Fixes
- Extra arm weapons now correctly count toward carried weight.
- Bracelet items correctly equip to wrist slots instead of ring.
- Cloaks moved from neck slot to back slot (automatic migration
  on login for existing characters).

---

## 2026-03-29 — Combat, Spell & Crafting Fixes

### Spell Fixes
- **Sparks** now correctly hits all enemies in the room (was only
  hitting one target despite being an area spell).
- **Mend All** now actually heals (was missing effect type data).
- **Hemorrhagic Wave** rebalanced: folds 10→20, magnitude 300→225.
  Still powerful AoE but no longer trivializes encounters.
- **Healing spells can now target downed players**, enabling
  revive-style healing like Chrysalis Aid. Harm spells skip
  downed players.
- **Self-targeting blocked** for harm spells — Conviction Spike
  and Nerve Disruption can no longer be cast on yourself.
- **Player-targeted harm spells** now deal damage and trigger
  reciprocal aggro (previously did nothing).
- Pet damage messages no longer duplicate in same-room combat.

### Crafting Fixes
- **Skill level-up messages** no longer repeat on every successful
  craft. The "Your X skill reaches Y!" message only appears when
  your skill tier actually increases.
- **Recipe discovery** now filters by the skill you're currently
  crafting. No more learning enchanting recipes while blacksmithing.
- **Casting and sneaking blocked while crafting.** Moving to another
  room cancels the active craft.

### Other Fixes
- **Title command** no longer shows raw numbers. Mutation load and
  skill progress use descriptive labels (fledgling/seasoned/etc).
- **Apothecary Voss** now sells alchemy ingredients instead of
  enchanting binding paste.

---

## 2026-03-29 — Critical Fixes + Inventory Rework

### Critical Bug Fixes
- **Death loop fix**: Players can no longer get permanently stuck
  in the Shadow Realm with stale combat state. Root cause fixed
  (mobs could re-aggro dead players), plus safety net and escape
  hatch so the portal always works.
- **Spell scripts now work for players**: Fold Anchor, Chrysalis
  Aid, Summon Steppe Spirit, and other script-based spells were
  silently broken — onMagic/onCast/onWait callbacks never fired
  for player casts. All three hooks are now wired into the cast
  pipeline.
- **Fold Anchor split**: Now two spells — `fold-anchor` (set) and
  `fold-recall` (teleport back). Players who knew fold-anchor
  automatically receive fold-recall on login.
- **Quest spell rewards**: Quests can now teach spells on
  completion. The Warden's Covenant (quest 12) now properly
  grants Summon Steppe Spirit.
- **Fetish gating**: Windwarden Sylara no longer gives unlimited
  spirit fetishes. If you already have one, she refuses.

### Inventory Rework
- **Diku-style disambiguation**: Use `3.dagger` or `dagger#3` to
  target a specific item when you have duplicates. Use `all.item`
  with get/drop to affect all matching items (e.g., `drop all.potion`).
- **Inventory stacking**: Identical items now group together with a
  count, e.g., `iron ingot (x5)`. Items with different enchantments
  remain separate.
- **Equipped item targeting**: `look` and `identify` now search
  your backpack and equipment as a single pool. You can examine a
  wielded weapon without unequipping it — use `look 2.dagger` to
  reach the equipped one when a duplicate is in your pack.
- **Encumbrance display**: Carrying capacity has been rebalanced.
  The inventory command now shows a colored encumbrance tier
  (light / moderate / heavy / overburdened / crushed) instead of
  raw weight numbers. Add `{enc}` to your prompt to track it at
  a glance (`help set prompt`).
- **Multi-buy**: `buy 5 iron ingot` purchases multiple copies in
  one command. Stops early if you run out of gold or can't carry
  any more.
- **Enchanting targeting**: `craft <recipe> <item-name>` lets you
  choose which item to enchant. Works on equipped items too. Shows
  a numbered list when multiple targets match.
- **Look direction fix**: `look n` no longer matches inventory
  items when no north exit exists.

### Balance
- Carry capacity reduced ~78% (now Strength × 0.65, configurable).
  Being overweight costs more stamina to move and reduces combat
  swings.

## 2026-03-18 — Skullduggery Skill + Tutorial Fix

### New Skill: Skullduggery
The old `stealth` skill has been consolidated into **skullduggery**,
a full rogue toolkit with seven sub-commands:

- **sneak** — hide using opposed rolls (Dex+skill vs observers'
  Perception+search). Empty rooms auto-succeed. Party excluded.
- **steal** — take gold/items from NPCs or containers. Being hidden
  improves your chances. Replaces the old pickpocket command.
- **plant** — slip items onto NPCs or into containers unnoticed.
- **shadow** — tail a target between rooms while hidden (rank 2+).
  Room-entry detection checks on each transition.
- **surprise attack** — automatic extra crit hit when attacking from
  stealth. Swings all weapons with stacking hit penalties. Party
  auto-assist triggers coordinated ambushes.
- **picklock** — existing minigame, now gated behind skullduggery
  rank 1.
- **defuse** — trap disabling stub for future content (rank 3+).

### Stealth Detection Rework
- Hidden players are now rolled against when entering rooms
- New arrivals roll to spot hidden occupants in the room
- Party members excluded from all detection checks

### Stealth & Movement Improvements
- Hidden players skip room onEnter scripts (NPCs no longer greet
  you when they can't see you)
- Sneak buff now applies immediately (no tick delay before moving)
- Per-weapon surprise attack messaging shows each weapon's hit
  and damage individually

### Quality of Life
- MOTD now displays in a visible bordered box on login
- Skill-gated commands show "You aren't advanced enough at
  skullduggery for that" instead of "command not found"
- Updated help files for steal and plant with clear syntax and
  examples
- Added missing alchemy_bench station to Apothecary Lane (room 471)
- Added hidden buff (ID 9) to dogmud world buffs (was missing)

### Bug Fixes
- Tutorial casting step now accepts spell ID shortcuts and aliases
  (e.g., `conviction-spike echo` works, not just
  `cast conviction-spike echo`)
- Existing characters auto-migrate stealth skill to skullduggery
  on login

## 2026-03-17 — Bug Fixes, Hidden Object Discovery, Identify Spell

### Legacy Stat Scaling Fixes
- Map command perception thresholds rescaled for 100-baseline
  stats (was 25/50/75, now 110/135/175). New characters start
  at tier 1 zoom instead of getting max zoom immediately.

### New Spell: Identify
The old `inspect` command has been removed and replaced with
the **Identify** spell (Mental school).

- Cast `identify <item>` to reveal an item's properties using
  descriptive language (no raw numbers shown to players)
- Works on items in your backpack or currently equipped
- Costs 20 conviction, 3 folds to cast, 30-round cooldown
- Merchants still offer `appraise` as a non-magical alternative

### Appraise Rework
- Appraise now shows full item details (previously capped at
  tier 3). All output uses descriptive language instead of raw
  numbers.

## 2026-03-17 — Bug Fix Day + Hidden Object Discovery System

### Bug Fixes (9 issues from playtesting)
- Conviction regen bumped to 2% per tick (matches health/stamina)
- Removed legacy tame-on-kill messages (taming now uses spells)
- Fixed disarm crit triggering on unarmed/disabled-slot targets
- Fixed misspelled commands showing "can't do that in combat"
  instead of "command not recognized"
- Fixed 2H weapon + extra arms exploit (extra arm slots now
  cleared when equipping a two-handed weapon)
- Fixed fold-anchor recall failing due to type mismatch
- Fixed gossip system — NPCs now report mob kills and player
  mutations (event buffer was starving for events)
- Fixed bleedout test to match current rate (2 per tick)
- Added `{attack}` token for defense messages (resolves to
  "strike" when attacker is unarmed)

### New Feature: Hidden Object Discovery
Rooms can now contain hidden nouns and hidden containers that
players must actively discover using the search command.

- **Hidden nouns** — invisible until found via search. Once
  discovered, they appear in the room description and respond
  to `look <noun>` permanently for that character.
- **Hidden containers** — function like normal containers but
  are invisible until discovered. Locks still apply after
  discovery.
- Discoveries persist permanently per-character.

### Skill Consolidation: Search
The tracking and foraging skills have been merged into a single
**Search** skill governed by Perception.

- `search`, `track`, and `forage` all progress the Search skill
- All three commands now use gaussian dice rolls (Perception +
  Search skill bonus) instead of hard stat thresholds
- Forage difficulty varies by biome (farmland is easiest,
  cliffs are hardest)
- Existing players: Search rank = max(tracking, foraging).
  No progression is lost.

### Balance
- Extra-arms mutation restricted to species with arm slots
  (no more wolves with extra arms)
- Search skill progression only fires when there's something
  undiscovered to roll against (prevents AFK botting)

## 2026-03-14 — Zone Expansion, Spell Merge, Coordinate System

### New Zone: Marches Spur Road
A new 15-room zone connecting Watchers Crossing south to the Ashwick
Crossroads — the first road into the wider Windward Marches.

- **15 rooms** from scrubland through farmland to a waypoint inn and
  crossroads junction
- **The Broken Yoke Inn** — social hub with gossiper NPCs relaying
  world events
- **Peddler Malk** — road merchant and quest giver at a camp along
  the spur
- **Quest: The Peddler's Overdue Freight** — find a stolen freight
  crate. Solve it through combat (clear the bandit barn) or diplomacy
  (negotiate a toll with the bandit leader). Multiple breadcrumbs and
  elephant-path coverage.
- **Bandit Leader encounter** — non-hostile with a 5-round dialogue
  window before she attacks. Talk fast or fight.
- **Wildlife**: scrub coyotes, feral hogs, field sparrows, farm cats

### New Zone: Ashwick
Maren's home hamlet from the novel, 20 rooms east of the Ashwick
Crossroads. A quiet farming village with secrets beneath the surface.

- **20 rooms** — hamlet proper (central green, chapel, farmstead,
  ritual circle, well, Delia's cottage) plus forest outskirts
  leading into deep woods
- **Delia the herbalist** — quest giver and alchemy crafting station
- **Deacon Ferris** — lore NPC with quest-gated deeper dialogue
- **The Forager (Sev)** — a hollow person hiding in the woods,
  mirroring the novel's themes of identity and concealment
- **Quest: The Herbalist's Shortage** — someone is harvesting
  Delia's herbs. Negotiate with the forager or find an alternate
  source in a hidden Chrysalis-touched grove.
- **Quest: The Empty Cottage** — explore Maren's abandoned family
  home. Push a loose hearthstone to find a hidden letter mentioning
  "the hill" and "Voss in New Plymouth." Study a recipe page from
  the bedside table to advance your foraging skill.
- **Novel breadcrumbs** throughout — scorch mark on the ritual
  circle, inner orbit symbol at the well and chapel, the cottage's
  empty shelves and cold hearth. Layered discovery rewards
  attentive players without frontloading spoilers.
- **Wildlife**: timber wolves, briar hawks, forest foxes, village
  chickens, a cottage mouse

### Spell Changes
- **Fold Anchor + Fold Recall merged** into a single toggle spell.
  Cast once to set an anchor, cast again from elsewhere to teleport
  back. No more needing to learn two spells for one mechanic. Existing
  players with Fold Anchor gain recall automatically.
- New dedicated help template for Fold Anchor explaining both modes.

### Cartesian Coordinate System
- All 224 existing rooms now have hidden `coord` fields (x, y, z)
  for spatial validation
- Full coordinate map at `docs/coordinate_map.md`
- **3 geometry overlaps fixed** in Sanctum Basin and Ironwind Steppe
  where rooms occupied the same coordinate
- Cartesian consistency rules added to zone expansion standards

### Bug Fixes
- Fixed steppe rooms 3032/3082: replaced JS quest item scripts with
  native container-based spawns
- Removed extra mob spawn from goblin camp room 3070
- Deleted stale instance saves that were overriding template edits

### Infrastructure
- Zone expansion plan (ZONE_EXPANSION.md): 22 zones, ~600 rooms
  planned across the Windward Marches
- Geography aligned to novel canon (What the Moons Keep)
- AI player default host updated to dogmud.org

---

## 2026-03-05 — Ironwind Steppe Rebuild, Quests & Ecosystem AI

### Ironwind Steppe Zone Rebuild
The entire Ironwind Steppe zone was rebuilt from scratch on a clean
cardinal grid with proper reciprocal exits throughout.

- **Rebuilt entry area** (rooms 3000-3009) on a clean cardinal grid
- **Sagebrush Flats** expansion (3010-3015, 3018) with ambient wildlife
- **Northern wolf/hawk territory** (3019-3023) with predator encounters
- **Hollow and boar/viper area** (3024-3028) with burrowing wildlife
- **Ironwind Ridge column** and northern steppe edge (3029-3033)
- **Upper ridge** — nesting ledge to summit (3034-3038)
- **East ridge descent** — alcove to overlook (3039-3043)
- **Dry creek system** and ridge descent (3044-3048)
- **Creek basin depths** — undercut bank to boar wallows (3049-3053)
- **Lower creek basin** — junction to basin south end (3054-3058)
- **Basalt coulee system** east of creek basin (3059-3063)
- **Deep coulee goblin territory** (3064-3068)
- **Goblin camp interior** and coulee south exit (3069-3073)
- **Wolf Run** and eastern coulee edge (3074-3078)
- **Deep Wolf Run** and wolf ravine east column (3079-3083)
- **Lower wolf territory** (3084-3088)
- **Boar wallow column** and eastern grassland (3089-3093)
- **Mudflat/boar territory** and drinking pool (3094-3098)
- **Cave system** entrance through deepest chambers (3099-3114)

### Quests
- **Quest 12 audit** — Sylara now grants quest start directly,
  removed unnecessary Kael checkpoint that could brick progression
- **Quest 14: The Undertow** — new smuggler tunnel quest beneath
  the Drowning Post tavern in Thornwall City

### Ecosystem AI
- **Species-based pack behavior** — mobs now ally and flee based on
  shared species (SpeciesId) instead of broad group tags. Wolves pack
  with wolves, not with squirrels that happen to share a group tag.
- **Predator-prey combat** — `HatesMob()` rewritten to use species
  names. Added natural predator-prey hatred across the ecosystem:
  - Canines (wolves, foxes, dogs) hunt rodents and boars
  - Raptors (hawks, eagles) hunt rodents and serpents
  - Felines hunt rodents and insects
  - Serpents hunt rodents and insects
  - Arachnids (spiders, scorpions) hunt insects
  - Boars defensively attack canines
  - Trolls attack most wildlife species

### Balance
- Bumped player conviction regen from 1% to 1.5% per tick

### Bug Fixes
- Fixed broken ANSI tag in Old Citadel Plaza board noun
- Fixed scrubland dog species (was reptile, now canine)

---

## 2026-03-04 — Quest Fixes, Balance Tuning & Zone Repairs

### Quest Fixes
- **Velk/Elara quest** — made dialogue discoverable and unbrickable
- **Harn/Pell delivery quest (Quest 6)** — unbricked progression
- **Removed `requires` + `expiryPeriod` quest brick** from all
  dialogue files across the game. These combinations could silently
  brick quests when player memory expired.

### Balance Tuning
- Reduced `GlobalDamageMultiplier` from 1.75 to 1.25 for less swingy
  combat
- Potion buff improvements and helpfile additions
- Temple regen and hint system improvements
- Faster bleedout timer for downed players

### Zone Fixes
- Resolved 74 broken reciprocal exits across the Ironwind Steppe zone
- Fixed spatial inconsistency in Watchers Crossing river lurker loop
- Disconnected Ironwind Steppe from Thornwall temporarily until zone
  rebuild was complete

### Features
- **Player PvE death gossip** — tavern gossip system now broadcasts
  player deaths to the gossip channel (global, not just local)
- **setmotd admin command** — admins can now set the message of the
  day in-game

### Bug Fixes
- Fixed `FindRecipeByName` to prefer exact match, preventing wrong
  recipe selection when names overlap
- Removed Area field from status template to prevent zone name
  misalignment
- Aligned status template columns with consistent 12+13 char widths
- Added missing admin commands to help list with helpfiles

---

## 2026-03-03 — Launch Day Fixes

### Major Fixes
- **ANSI tag crash fix** — prevented nested ANSI tags in noun
  highlighting that caused client crashes. Root cause fix in
  `roomdetails.go` to skip nouns already inside ANSI tags.
- **Tinymap panic fix** — used `VisibleWidth()` instead of `len()`
  for tinymap padding, preventing panics from ANSI escape sequences
  in map rendering.
- **Instance save override fix** — added `instance:"skip"` tag to
  structural room fields (exits, nouns, signs) so instance saves
  can no longer silently override template data for these fields.

### Quest & Item Fixes
- Scholar quest now accepts totem and spore sac in either order
- Added `givesItem` field to dialogue engine for NPC item handoffs
- Fixed Watchers Crossing quest items using new `givesItem` system
- Replaced removed skulduggery quest reward
- Fixed `get all <container>` command support
- Fixed leaderboard stale stats and scholar `onGive` handler

### Combat & Mob Fixes
- Made mob commands darkness-aware (mobs no longer act normally in
  pitch-dark rooms)
- Enabled wolf vs boar predator/prey combat on the steppe
- Fixed web client auto-scroll behavior

### UI & Display Fixes
- Reorganized status template into logical sections
- Fixed per-player buy/equip tracking with purchase debug logging
- Renamed 'back corner' room noun to 'alcove' to fix room 472 crash
- Removed ANSI tags from descriptions where the word is also a noun
  key
- Removed long-range exits from Thornwall City templates
- Fixed Thornwall cardinality, Brecca shop inventory, copper ring
  naming
- Web portal "Who's Online" now uses Title instead of removed
  Profession field

### Documentation
- Added deployment troubleshooting guide for git sync and Docker
  cache issues
- Added compose file warnings and port conflict troubleshooting
