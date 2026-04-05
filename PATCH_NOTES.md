# DOGMud Patch Notes

## 2026-04-04 — Living Economy, Gear Upgrades, Spell & PvP Fixes

### Bug Fixes (Round 2)
- **Tail slot no longer shows** when the character lacks the tail
  mutation. Was caused by EnableAll() resetting the disabled state
  before the tail check ran.
- **Companions no longer despawn** from the idle boredom timer.
  Wolf spirits and other charmed companions now persist properly,
  fixing missing vitals bars in the web client.
- **Mobs targeting your companions now show red** in the look
  command, same as mobs targeting you directly.
- **Duplicate companion vitals fixed** — same-name companions
  (e.g., two skeletons) now show separate health bars.
- **Gossip quality improved** — NPCs now use different phrasing
  for local vs. distant events, and each gossiper tracks recently
  mentioned events to avoid repetition.

### Living Economy
- Merchants now track finite stock and gold. Prices rise when
  stock runs low and drop when overstocked.
- Crafter NPCs restock materials periodically (with flavor
  text) and craft items to sell — prioritizing self-gear
  upgrades, then profitable crafts, then salvage.
- Merchants will buy craft materials matching their trade,
  potions (unless they craft that potion themselves), and gear
  that upgrades their own equipment — including paired slots
  like rings and wrists. Specialists won't buy materials from
  other professions.
- Shopkeepers are now non-combatant — they cannot be attacked,
  stolen from, or targeted by harmful spells.
- Bartering skill now affects buy and sell prices at shops.

### Under the Hood
- Mobs now advance stats and skills from basic attacks, special
  moves, and spellcasting — same progression system as players.
- Combat commands (bash, kick, trip, grapple, cast) now handle
  skill progression in the shared action layer rather than
  separately for players and mobs.
- Mob howl and player taunt now share the same underlying
  conviction-damage mechanics. (Also fixed howl not applying
  the skill-weight multiplier to rhetoric.)
- Bite and hamstring are now shared actions, ready for future
  player species-gated abilities.

### Bug Fixes
- **Area harm spells no longer damage the caster's companions.**
  This was caused by the spell resolution step overwriting the
  companion-exclusion filter from cast initiation.
- Single-target harm spells now prevent targeting your own
  companion ("You can't target your own companion with a
  harmful spell.").
- Charmed mob casters no longer hit their owner or the owner's
  other companions with area spells.
- Casting an area spell with no valid targets now gives feedback
  ("Your spell erupts outward but finds no targets.") instead
  of silently consuming conviction.
- PvP is now properly blocked across all combat entry points
  (attack, bash, kick, trip, grapple, taunt, shoot, spells).
- Fixed enchanting craft command parsing for hyphenated recipe
  names (e.g. "craft honed-edge knuckles" no longer fails).
- Shop listing now shows correct finite stock and dynamic
  prices instead of infinite legacy quantities.

### Cleanup
- Removed deprecated mob commands: roar, throw, backstab.
- Renamed mob `alchemy` command to `selljunk` (converts
  inventory to gold — not related to player alchemy).

---

## 2026-04-03 — Manifestation, Companions, Necromancy, Elementals, New Zones

### New Content
- New hidden areas have been added to the world. Sharp-eyed
  adventurers may discover passages others have overlooked.
- A reclusive figure lives off the beaten path. Not everything
  is as it appears — tread carefully.
- Lockpicks and disarm kits now available from certain merchants.
- Crafters can forge superior tools at high skill levels.
- Powerful caster equipment can be found by those who earn it.

### New Mechanics
- **Defuse command** — disarm traps on locks before picking.
  Requires a disarm kit. Higher tier kits improve success.
- **Flee rework** — flee is now an opposed roll (Dex+skullduggery
  vs Dex+unarmed-combat). Rogues are better at escaping.
  Can't flee while grappled. Prone halves flee chance.
- **Fist weapons** — new weapon subtype using unarmed-combat skill.

### Quality of Life
- `idea` is now an alias for `suggest`.
- `disarm` is now an alias for `defuse`.
- `lockpick` and `pick` are aliases for `picklock`.
- Companions prevent sneaking — dismiss before stealth.
- Companion corpses cannot be raised by necromancy.

### Bug Fixes
- AOE harm spells no longer damage the caster or their companions.

### New System: Manifestation Skill
A new charisma-based skill governing summoning, conjuring, charming,
and raising undead companions. Manifestation spells use Charisma
instead of Willpower for fold rate and discovery.

### New System: Unified Companions
Pets, summoned creatures, conjured elementals, charmed mobs, and
raised undead all share a unified companion system. Summoned,
conjured, and raised companions persist across restarts. Charmed
companions are temporary — they don't survive server restarts.
All companions show in the vitals panel, respond to autoassist,
and can be buffed with help spells.

- `companion` / `companions` — view companion vitals and stats
- `dismiss` — release a companion (warning: full betrayal)
- `assess` — study a corpse for necromantic potential
- `{pet_hp}`, `{pet_sp}`, `{pet_cp}` prompt tokens

### Necromancy (6 undead types)
Raise undead from corpses. Stronger corpses support more powerful
types. Power scales 50/50 from caster stats and corpse strength.
- Skeleton, Zombie, Wraith (caster), Spectre (conviction caster),
  Vampire (life drain bite), Flesh Golem (absorbs corpses)

### Conjure Elementals (5 types)
Conjure elemental companions from nothing. Very high conviction
cost — conjuring a magma elemental drains nearly your entire pool.
- Water (tank), Earth (bash), Air (evasive), Fire (return damage),
  Magma (bash + return damage, skill gate 60)

### Charm Spell
Opposed roll (Charisma+manifestation vs Willpower+statpool%) to
convert hostile mobs into companions. Harder against targets in
combat. Duration-based with diminishing re-rolls — your hold
gradually weakens until it breaks or you reassert control.

### New Combat Mechanics
- **Return damage** — fire/magma elementals reflect melee damage.
  Also available via equipment and buffs (battlerager armor, etc.)
- **Natural bash** — earth/magma elementals bash without shields
  ("crushing slam" instead of "shield bash")
- **Grapple immunity** — wraiths, spectres, air and fire elementals
  can't be grappled or grapple others
- **Vampire bite** — life drain special attack

### Aggro Rework
Centralized aggro state management fixes multiple companion combat
bugs. Players now properly retarget when companions kill their
target, when targets flee, and when new threats appear.

### Bug Fixes
- Enchanting target search broken by multi-word recipe names
- Conditions display showed total duration instead of remaining
- Infinite gold exploit — merchants pay from own gold pool
- Companion duplication on browser refresh
- Summon species corrections (were using rodent stats)
- Pack flee excludes companions
- Stale aggro from companion kills
- Web client vitals panel resizes with companion rows

### Balance
- Melee skill progression 0.20 → 0.15 (auto-attack now works)
- Spell damage scale 1.6 → 1.2 (progression provides natural scaling)
- Merchant stats buffed (85-150 statpool, 50-300g gold)
- Corpse decay 1 hour → 4 hours (for necromancy)

## 2026-04-02 — Command Unification + Bug Fixes

### Command Unification (feature/command-unification)
Major architectural rework unifying player and mob command systems
through shared core logic. Both sides now call the same underlying
actions for all major game commands.

**Shared Actor System:**
- Actor interface in `internal/actions/` abstracts over players
  and mobs. Shared actions operate on either actor type.
- Atomic transfer primitives (TransferItem, TransferGold) with
  rollback prevent item duplication and loss.
- Registry audit at startup warns about unintentional command gaps.

**Unified Commands:** say, emote, drop, remove, equip, get, give,
go, bash, kick, trip, grapple, shoot, attack, cast, sneak, craft.

**Combat Parity:**
- Kick now selects stomp/knee variants for mobs (position-aware).
- Trip uses tailsweep for mobs with tail mutation.
- Shared combat helpers (target resolution, cooldowns, analytics).

**Progression Parity:**
- Mobs now advance stats and skills from combat, casting, crafting.
- Player auto-attack melee progression was broken (never fired) —
  now works correctly.
- Caster mobs discover new spells as spellcasting skill increases.
- Mob sneak uses opposed rolls instead of auto-succeeding.

**Mob Crafting:**
- Mobs can now craft items via the shared craft system.
- Crafting completion fires skill progression for mobs.

### Fixes
- **Hidden mob perma-stealth:** Root cause found — permabuff system
  re-added Hidden buff after every Validate(). Fixed with
  RemovePermaBuff + proper combat loop integration.
- **Hidden mob surprise attacks:** Mobs properly get [SURPRISE ATTACK]
  when ambushing from stealth. Hidden buff clears after first strike.
- **Duplicate "prepares to fight" message:** Suppressed when mob
  re-attacks the same target.
- **Sneak in combat:** Blocked for both players and mobs — sneaking
  mid-combat doesn't make sense and caused perma-hidden bug.
- **Conditions duration display:** Was showing total duration instead
  of remaining rounds (swapped return values).
- **Infinite gold exploit:** Merchants now pay from their own gold
  pool when buying items. Refuse if they can't afford it.
- **Defense hint:** Now points to `help defense` instead of a
  nonexistent `defense` command.

### Balance
- Melee skill progression reduced from 0.20 to 0.15 — the bump
  was compensating for broken auto-attack progression (now fixed).
- Spell damage scale reduced 25% (1.6 → 1.2) — progression now
  provides natural scaling.
- Merchants buffed: higher stats (85-150 statpool), gold reserves
  (50-300g), Siv armed with a dagger.

## 2026-04-02 — Bug Fixes & Polish

### Fixes
- **Enchantment idle bug:** Chrysalis enchantments (honed edge, etc.)
  no longer progress while idle. They now only tick during combat.
- **Web client side panels:** Map, Communications, and Vitals
  windows now resize and reposition dynamically when the browser
  window is resized (both horizontally and vertically). Vitals
  no longer gets cut off on smaller screens like laptops.
- **Small screen support:** Side panels are hidden entirely on
  very small screens (phones/small tablets under 768px) to keep
  the terminal usable.

### Content
- **help equipment:** New help file covering all equipment slots,
  back slot trade-off (cloaks vs backpacks), belt slot trade-off
  (belts vs bandoliers), and the component bag system.

## 2026-04-01 — Quest Engine

### New System: Quest Engine
A complete YAML-driven quest engine that replaces JavaScript
scripts for all quest logic. Quests are now defined entirely in
data files with declarative triggers and conditions.

- **9 event types:** room_enter, room_interact, item_gain,
  item_give, mob_death, skill_use, command, dialogue,
  quest_granted (for chaining steps automatically).
- **Trigger actions:** grant quest tokens, give/consume items,
  send text, NPC dialogue sequences, teleport, spawn mobs,
  teach spells, apply buffs, set quest flags.
- **Quest flags:** branching quests track which path the player
  chose. Flag-gated dialogue shows different content per path.
  Undeclared flags panic at startup to catch typos early.
- **hint command:** type `hint` for guidance on your current
  quest step. Hints give explicit directions and next actions.
- **Verbose quest debugging:** admins can enable per-player
  quest debug logging with `questdebug <player>`.

### All Quests Ported (1-16)
Every quest in the game now runs through the quest engine:
- **Quest 1 (Sanctum Trials)** — full tutorial with ceremony
  sequences, mutation grant, shopping/equip/combat/magic steps.
- **Quest 2 (Warren Compact)** — salve delivery to tunnel shaman.
  Mobs become peaceful after quest completion.
- **Quest 3 (Scholar's Collection)** — dual-item delivery with
  flag tracking for partial completion.
- **Quests 4-7** — item delivery and combat quests across
  Dustwalk Road, Watchers Crossing, and Thornwall Outskirts.
- **Quest 8 (Missing Person)** — investigation quest in Thornwall.
- **Quest 9 (Tithe Audit)** — ledger delivery to Priest Olen.
- **Quest 10 (Drowning Post)** — protection notice to Velk.
- **Quest 11 (Windwarden's Dilemma)** — opposed branching quest
  with quest flags. Choose Sylara or Rhett; the other dismisses
  you. Flag-gated followup quests (12 or 13).
- **Quests 12-13** — path-exclusive followup quests (Covenant
  vs Extraction) gated by Q11 branch flag.
- **Quest 14 (The Undertow)** — 6-step dungeon crawl with cellar
  gate, tally stick discovery, strongbox key/open interaction,
  and bribe ledger delivery. Full room_interact support.
- **Quest 15 (Peddler's Freight)** — crate delivery with combat
  or diplomacy paths.
- **Quest 16 (Herbalist's Shortage)** — dual-path herb delivery
  with bypass for players who explore first.
- **Quest 17 (Empty Cottage)** — converted to lore discovery
  (no longer a tracked quest).

### Bug Fixes
- **Quest re-grant prevention** — fixed 18 dialogue nodes across
  15 files where completed quests could be re-offered. Added
  runtime validation that warns if a quest-granting node is
  missing its end-token exclusion.
- **Quest hints improved** — all quests now give explicit
  step-by-step directions with cardinal directions and counts.
- **Dialogue hints** now display as narrator text, not NPC speech.
- **Branching quest dismissals** — wrong-path players get clear
  rejection instead of confusing keyword matches.
- **Shadow Realm combat trap** — fixed a bug where players could
  get stuck in the Shadow Realm with stale combat state after
  the warden-bandit alliance fight.
- **False skill-up messages** — skill progression messages no
  longer fire on critical failures or first mob kills when no
  real skill gain occurred.
- **Alchemy recipe cleanup** — removed legacy duplicate starter
  recipes that confused new players in the tutorial. Tutorial
  now uses healing salve instead of removed healing poultice.

### Balance
- **Moon phase effects doubled** — full/new moon bonuses and
  penalties are now more noticeable.

### Migration
- Players on removed quest steps are automatically reset to
  "start" on server startup. Quest 17 progress removed entirely.
- Quest 11 branch flags inferred from Q12/Q13 progress for
  existing players.
- Legacy healing poultice and stamina draught auto-converted to
  new alchemy equivalents.

---

## 2026-03-31 — Salvage System

### New Feature: Salvage
Break down crafted items and tagged salvageable items to recover
crafting materials with the new `salvage` command.

- **New skill: Salvage** — standalone Perception-based skill in
  the "scavenger" profession alongside Search. Recovery chance
  scales with skill via a sqrt curve (15% at novice, up to 85%
  at master). Each ingredient is rolled independently.
- **Recipe reverse-lookup** — any item produced by a crafting
  recipe can be salvaged at the matching station for free.
- **Salvage kit** — sold by Fence Dealer Siv in Thornwall's back
  alleys for 1g. Allows salvaging anywhere without a station.
  Consumed on each use.
- **Tagged items** — non-crafted items can be marked salvageable
  with `salvage_returns` on their item spec. Always requires a
  salvage kit.
- **Multi-round activity** — salvage duration scales with
  ingredient value (1-5 rounds). Interrupted by combat.
- **Item always consumed** — even if no materials are recovered,
  the item is destroyed.
- Type `help salvage` in-game for full details.

---

## 2026-03-31 — Bug Fixes & QoL

### Features
- **ASCII Charset Mode:** `set charset` toggles between UTF-8 and ASCII
  display. Legacy clients (zMUD etc.) that show garbled box-drawing
  characters can switch to clean ASCII mode. Persists across sessions.
- **Mutation Help Files:** All 40 mutations now have individual help
  pages (`help healing-gel`, `help extra-arms`, etc.).

### Bug Fixes
- **Skill progression messages fixed:** Critical hit "technique improves"
  messages were firing on every crit regardless of whether the skill
  actually advanced. Now only shows when a real gain occurs.
- **Harm spell exploit closed:** Casting harm spells with no target no
  longer grants free spellcasting progression.
- **Harmful buffs trigger aggro:** Spells like Nerve Disruption that
  apply debuffs now properly start combat, matching damage/dot/knockdown
  behavior.
- **Tutorial directions corrected:** Directions to the Training Yard
  now correctly say north-then-east (was "northeast").
- **Removed misleading combat-end message:** The generic "rage subsides"
  text no longer appears after every kill.

### Balance
- **Combat skill progression bumped:** Weapon-combat and unarmed-combat
  progression rate increased from 0.12 to 0.20. These skills were
  advancing too slowly relative to other skills.

---

## 2026-03-30 — Alchemy Rework (Phase 1-3)

### Alchemy Overhaul
- **Potion Aging:** Potions now age through five phases (Fresh →
  Fermented → Peak → Declining → Spoiled). Peak potions are 30% more
  potent. Spoiled potions cause nausea and triple toxicity.
- **Bottle Tiers:** Four bottle types control aging speed. Clay flask
  (ages 3x faster, cheap), glass vial (baseline), sealed phial (half
  speed, jewelcrafting), crystalline decanter (quarter speed, advanced
  jewelcrafting).
- **Toxicity System:** Every potion adds toxicity. Exceed your limit
  and your body rejects the brew. High toxicity causes stat penalties.
  Toxicity decays naturally over time.
- **Craft Skill Matters:** Higher alchemy skill at brew time means
  stronger, longer-lasting potions that age slower.
- **Skill-Based Detection:** Examining potions reveals aging info
  based on your alchemy skill. Novices can't tell fresh from spoiled.

### New Potions (21 recipes)
- **Pool Regen (7):** Healing salve, stamina tonic, conviction
  draught, warrior's brew, preacher's tincture, windrunner draught,
  elixir of renewal.
- **Combat/Utility (10):** Ironhide brew, mindshield elixir,
  veilguard tonic, stone stomach, cat's eye draught, swiftfoot
  essence, berserker elixir, silver tongue oil, battle trance,
  purging draught.
- **Progression (4):** Essence of growth, savant's infusion, mutagen
  brew, chrysalis catalyst. These accelerate character development
  but reserve portions of your resource pools.

### Potion Bandolier
- New belt-slot item that auto-routes potions and reduces their
  weight. Two tiers: leather (6 slots, 30% weight reduction) and
  reinforced (12 slots, 40% weight reduction). Craft via tailoring.

### New Materials
- **Moonpetal** — rare forage, night only.
- **Veilbloom Petal** — very rare forage on the steppe.
- **Serpent Venom Sac** — drops from river lurkers and blind stalkers.
- **Ironbark Shaving** — uncommon forest forage.
- Clay flask sold by Apothecary Voss.

### Consumption Rework
- Drinking a potion now checks toxicity before consuming. If you'd
  exceed your maximum, the potion is rejected.
- Aging phase affects potency: peak potions last 30% longer, declining
  potions are weaker, spoiled potions cause nausea + 3x toxicity.
- Craft skill at brew time scales potion duration (skill 20 = +20%).
- Bottle type is stamped on the potion at craft time, determining its
  aging speed for its entire lifecycle.

### Maker's Mark
- Skilled crafters (skill 30+) now leave their name on items they
  craft. Examine a crafted weapon, potion, or piece of armor to see
  "Made by {name}." Purely cosmetic — does not affect stacking.

### QoL
- Spoiled potions display as "(turned)" in inventory for alchemists.
- Potions in bandolier show in a dedicated "Potions:" section.
- Drink command pulls from bandolier first (oldest potion).
- Five new alchemy-related gameplay tips in the hints rotation.
- Old potions and recipe knowledge auto-migrate on login.

### Bug Fixes
- **Velk bribe ledger quest:** Fixed quest getting stuck at 83%.
  The dialogue was still asking for the ledger after it had been
  given. Players with the stuck quest should now be able to complete
  it by talking to Velk.
- **Sylara spirit fetish spell:** Fixed "You need a spirit fetish"
  error when the fetish was in the component bag. Spirit fetishes
  now stay in the regular backpack where the spell can find them.
- **Text wrapping:** Say, shout, whisper, emote, and party chat
  now wrap the full message (including speaker name) at 80 chars
  instead of wrapping text alone at 65 then prepending the name.
- **zMUD compatibility:** Fixed display flashing for legacy MUD
  clients that don't support GMCP. The server no longer sends GMCP
  data to clients that haven't completed the GMCP handshake.
- **Description wrapping:** Player and NPC descriptions no longer
  double-wrap with orphaned words. Descriptions are stored raw and
  wrapped once at display time. Existing player descriptions are
  auto-migrated on login.
- **Floor item stacking:** Identical items on the ground now display
  with (xN) count instead of separate lines.
- **Vendor room clutter:** Removed crafting materials baked into
  7 vendor/crafter room templates that respawned every restart.
- **Drop all:** No longer drops your gold. Use "drop N gold" to
  drop gold explicitly.

---

## 2026-03-30 — Mutations, Balance, Documentation & QoL

### New Mutations
- **Chameleon Skin** (rarity 7) — +30 stealth bonus, +10 dodge.
  Costs charisma and natural armor. Conflicts with thick-hide.
- **Tail** (rarity 8) — Adds Tail equipment slot, disables Legs
  slot. Reskins trip to tailsweep (better damage and knockdown).
  Three tail attachments: weighted cap, spiked band, bladed sheath.

### Stealth Improvements
- Characters emitting light have their sneak score halved.
- Moving while sneaking costs 50% more stamina.
- Hidden mobs now get surprise attack on their first strike.

### Spell Duration Scaling
- All spell durations now scale with fold count, spellcasting
  skill, and willpower via universal formula.
- Higher-fold spells naturally last longer. Investing in willpower
  and spellcasting extends everything.

### PowerScore Rework
- Skills are now a major factor (sqrt of total ranks × 25).
- All three resource pools count (HP + SP×0.5 + CP×0.5).
- Mutations contribute 20 points per level.
- KD ratio replaces raw kill count (kills/deaths × 10, cap 50).
- Magic/conviction offense normalized against physical.
- Defense weighted 3× more heavily.

### Defense Balance
- Dodge effectiveness 0.97→0.95, Parry 1.0→0.97, Block 1.02→1.05.
- New clinch defense penalties: dodge 0.80, parry 0.83, block 0.85.
- New grounded defense penalties: dodge 0.75, parry 0.77, block 0.80.
- Prone dodge/parry penalties 0.95→0.93.

### New Commands
- **afk** — Manual AFK toggle with optional message. Shows (AFK)
  next to your name in the room. Auto-clears on any input.
- **setdesc** — Set your own character description.

### Crafting
- Craft list now shows recipe completion tier per skill and overall.
- Subcomponent recipe thresholds lowered (steel ingot, chain links,
  chrysalis setting).

### Documentation
- Help files for all 39 spells, 47 recipes, and 4 combat skills.
- Completeness tests ensure new content always has help files.
- 15 new gameplay tips added to the hint rotation.

---

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
