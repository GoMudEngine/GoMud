# Newbie Area Rework — Design Spec

**Status:** Draft for review
**Date:** 2026-05-27
**Branch:** `feature/newbie-area-rework-spec`
**Scope:** Architectural brushstrokes for a from-scratch newbie zone replacing
Sanctum Basin. Detailed authoring lives in per-chunk sub-specs.
**Comparable in size to:** the Mob Aliveness roadmap (multi-week, multi-chunk).

---

## 1. Overview

Sanctum Basin (the current tutorial) ships as 24 rooms with a single linear
6-trial quest. The experience is too short, too rigid, fails to showcase
DOGMud's breadth, and offers no replayability or discovery.

This rework replaces Sanctum Basin with a **150-room minimum** newbie zone
built on a hub-and-spoke topology with concentric difficulty within each
spoke. The new zone is geographically isolated — only a long overland hike
or an earned hub portal lets the player leave — and it is structured for
**incremental authoring**: each spoke is a self-contained module that can
be built, tested, and connected independently of the others. Future systems
(ranged weapons, etc.) drop in as new spokes without touching what already
ships.

Cutover is no-risk: the new zone goes live spoke-by-spoke alongside the old
Sanctum Basin. Sanctum Basin is only deleted in the final chunk, after the
new zone is fully polished and connected to the wider world.

---

## 2. Goals (user-given)

1. **Dual audience.** Fun, draw-them-in experience for new players; an
   in-world portal escape hatch for veterans so they can skip at any time.
2. **Teach + reward.** Teach core mechanics; give the player unambiguous
   triumphs, interesting starter gear, and a few skill ranks.
3. **Simple, fun, progressive.** Clear sense of forward motion at every
   stage.
4. **Soft first hour.** No willy-nilly deaths. Failure is a setback, not
   a wipe.
5. **Don't overwhelm.** Concepts unfold one at a time, basic → advanced.
6. **Showcase the breadth.** By the time the player chooses to leave, they
   have at least *seen* every interesting system DOGMud offers.

---

## 3. Universal authoring tenets

These are non-negotiable rules that bind every chunk:

1. **No hard numbers in player-facing text, ever.** All damage, healing,
   durations, cooldowns, stat values, and skill levels are taught
   in-character with descriptive language. The newbie area is the highest-
   stakes place to honor this — first impressions set immersion. The
   `status` command exists as a player self-check tool, but the *teaching*
   never uses raw numbers.
2. **Extensibility.** Each spoke is a self-contained module — its own
   rooms, NPCs, quests, drops, and certification rewards. Adding a new
   system later (ranged, mounted combat, whatever ships next) means
   authoring a new spoke. No part of this design assumes a fixed roster.
3. **Soft early game.** Inner-ring rooms cannot kill an attentive new
   player. Middle ring can knock you out, but death-recovery returns you
   to the hub. Outer ring is where real challenge lives.
4. **Geographically isolated.** Pothole Coulee. The only fast way out is
   the hub portal (always available after the Awakening). The only slow
   way out is a hike through the badlands. There is no caravan, no
   teleport-to-friends, no shortcut.
5. **Every visible humanoid NPC is Opened (mutated).** Non-mutated humans
   are actively hunted in this world per canon, so they would not stand
   openly in a Chrysalis-Church-sponsored school. Every named townsperson,
   instructor, merchant, bandit, etc. is Opened — i.e. visibly mutated.
   This rule does **not** apply to non-humanoid creatures (wild animals,
   predators, aberrants, monsters); those exist as their own thing. The
   player walks into a town where mutation is the human norm —
   reframes the typical "you're a freak" trope from the start.
6. **Concepts unfold one at a time.** Every lesson beat introduces *one*
   new idea. Stacking is for the wider world.
7. **In-character lore wherever possible.** Folding, mutations, the
   Chrysalis, factions — all delivered through NPC dialogue, environmental
   storytelling, and quest framing.

---

## 4. Lore foundations

### 4.1 Geography — Pothole Coulee

The newbie area sits in **Pothole Coulee**, the channeled scablands east of
the Columbia River, north of the Feathers. Glacial-flood geology: deep
basalt plunge pools (each a small lake in a stone bowl), dry coulees (channel
washes) connecting them, scrubby steppe plateau in between, scattered talus
slopes and cave systems carved into basalt cliffs.

The geography is the architecture. The hub town sits in the central plunge
pool. Each spoke radiates outward along a dry coulee to its own basalt bowl
or scabland landmark. Lateral coulees connect adjacent spokes at their
outer rings, creating a roamable network rather than a fan of dead-end
roads.

### 4.2 The Chrysalis Church school

The hub centerpiece is a small school for the newly-Opened, sponsored by
(but not dominated by) the Chrysalis Church. It is not a grand academy or a
cathedral — it is a frontier waystation, run by a single Opened cleric and
a small staff. Its purpose is to help newly-mutated arrivals understand what
has happened to them, learn the basic skills of survival in Gaius, and find
their footing before joining the wider world.

The school's framing is *practical* more than *theological*. The cleric
performs the Awakening Rite (lore beat) but the actual teaching is done by
the various Opened residents of the town — a smith, a herbalist, a forager,
a folder (mage), and so on. The Church-sponsorship gives the school its
funding and protection; the day-to-day character is folksy and earthy.

### 4.3 The Awakening (random mutation, upfront)

The Awakening Rite is the player's first beat. The Church cleric performs
it at the central plunge pool. The pool is the awakening site — geologically
unusual, possibly an ancient Chrysalis-touched place where the infection
takes most readily. The player receives **one random mutation** drawn from
the species mutation pool (current engine behavior). Every mutation has
upside and downside; a powerful one is rare. The mutation is the player's
identity going forward; there is no reroll.

The Awakening also unlocks the hub **portal**, which becomes the veteran
escape route. From this moment, leaving the zone is a single action away.

### 4.4 The Folding (canon lore for spellcasting)

Spellcasting in Gaius is the discipline of **Folding**: the mage holds a
single clear mental image of the reality they intend to manifest — a flame,
a shielded body, a wound knit shut. They then **bifurcate** that image into
two identical copies of that intended reality, then four, then eight, then
sixteen — each fold a doubling, the way a sheet folded against itself
doubles its thickness. Each fold sinks the imagined reality more firmly
into the world.

Three properties scale with **willpower** (innate) and **spellcasting
skill** (trained):

1. **Maximum folds** achievable before the stack collapses
2. **Speed** of folding (how quickly the cascade reaches its peak)
3. **Concentration** — the ability to hold the stack against interruption
   (damage taken, being knocked prone, grappled, etc.)

A weak-willed novice may only manage one or two folds before the image
dissolves. A master under pressure can sustain dozens.

This is the in-fiction explanation of `calcSpellDuration(baseFolds, skill,
willpower)`. `baseFolds` is the spell's intrinsic fold depth; willpower and
skill multiply on top. The Folding instructor in Spoke E demonstrates the
1 → 2 → 4 → 8 cascade visibly with a small flame.

### 4.5 The Opened, the Set-Apart, the orbital truth

- **The Opened.** Per canon ("First Light", *What the Moons Keep*): the
  Chrysalis touched the soil of Gaius and the people. The strong opened
  and became Gaius's true people, mutated and named. Everyone in the hub
  town is Opened. Everyone the player meets is Opened.
- **The Set-Apart.** The few who did not open. Canon says they "tend the
  old things, carry the old burdens, serve." In Gaius proper they are
  actively hunted (per user). They are therefore **not visible** in the
  newbie area. No Set-Apart NPC stands in the school. The concept is
  preserved as a deeper-world tease for later content, not surfaced here.
- **The orbital truth.** Canon hints (per the novel) that the old
  iconography of the Chrysalis Church — concentric arcs interpreted as a
  spiral — is actually a set of *orbital paths*. The Witnesses (three
  moons) and the First Light have a sci-fi underpinning beneath the
  religion. The newbie area plants exactly one tease of this: an Orbital
  Stone in Spoke F (Lore), discoverable as a quiet moment with no fight
  attached. The player who finds and reads it will not understand it. The
  player who comes back later (after Stage X content lands) may, if they 
  have really explored the world in complete depth through to the end game.

---

## 5. Systems & commands a graduate should know

Organized into tiers. Each item is a **touch point** — the new zone must
include at least one moment where the player uses or sees it. Tier 1 is
mandatory; Tier 2 is strongly desired; Tier 3 is "tease only"; Tier 4 is
out of scope.

### Tier 1 — Must teach

**Awakening & identity**
- The Awakening Rite (random mutation granted)
- `mutations` command — see what you got and what it does
- Mention (in-character only) that mutations evolve with use and that
  new mutations can unlock naturally over time. **Concept only** —
  the underlying mechanics (catalysts, evolution stages) remain Tier 4
  out-of-scope for this zone. An NPC line, not a demonstration.
- Stats centered around a human baseline — taught in-character ("you are
  slightly stronger than most" never "Str 112")
- The 9 skills concept, use-based growth
- No XP, no levels — change comes from use

**Movement & exploration**
- Compass directions and `exits`
- `look`, `look <target>`, `examine`
- `map` (visited-room map)
- `pathto <room>` for long-distance routing
- `hint` (the quest/dialogue hint system)
- `who`, `time`, `weather`, `help`, `help <topic>`

**Inventory & equipment**
- `inventory`/`i`, `get`, `drop`, `give`
- `wield`, `wear`, `remove`, `unwield`
- Item disambiguation (`2.dagger` / `dagger#2` / `all.coin`)
- Encumbrance tiers (light → crushed) — descriptive, never raw weight
- Equipment-slot overview (which slots exist, what goes where)
- Component-bag concept (auto-route on pickup, `sort` for migration)

**Combat basics**
- `attack <target>`, `flee`, `surrender`
- Three resource pools — HP / SP / CP — taught in-character ("a deep
  ache in your bones / a burning in your lungs / your resolve falters")
- Defense is automatic (dodge/parry/block) — no command, no input
- Damage descriptors ("light wounds," "serious wounds") — never raw
- Death and recovery flow

**Health & rest**
- `status` exists as a self-check tool (the teaching itself stays
  descriptive)
- `eat`, `drink`, `sleep`/`stand`
- Sleep mechanics: 5× regen with the auto-crit-first-round tradeoff

**Communication**
- `say`, `tell <player>`, `ooc`, `shout`
- `emote`
- `ask <npc> <topic>` — keyword dialogue
- `help`, `help <topic>`

**Shopping & money**
- `list`, `buy`, `sell`, multi-buy (`buy 5 …`)
- Bartering basics
- Bank deposit / withdraw (if gold should survive death)

**Quests**
- `quest` / `quests`, `ask <npc> quest`
- Hint-reading
- Reward + turn-in flow

**Helpfiles and self-help literacy**
- The `help` command and `help <topic>` syntax
- How the helpfiles are structured and cross-referenced
- Encouragement to read more — the in-game help system is the player's
  long-term reference, not just a beginner crutch
- **Placement:** primary teaching in the hub (Chunk 1 — school cleric
  walks the player through `help` as part of orientation); reinforcement
  in Spoke F (Lore), where reading and discovery are already thematic

### Tier 2 — Should teach

**The shared cooldown** (key teaching beat, often missed by new players)
- All special moves (kick, bash, trip, rally, hamstring, bite), **thrown
  alchemical grenades**, and **spellcasting** share a single cooldown
  timer (`special-move`, currently `SpecialMoveCooldown` rounds —
  verified at `internal/usercommands/throw.go:35`)
- Taught in-character as "you can only attempt one feat of will or
  technique per few breaths" — the concept is pacing, not the number
- A failed special on cooldown still triggers combat (no risk-free
  probing) — surface that lesson explicitly
- Spoke C (Alchemy) is responsible for the grenades-share-this-cooldown
  beat, since grenades originate there; throwing a grenade locks out
  spellcasting and special moves for the same window

**Crafting & gathering**
- `recipes`, `craft <recipe>`
- `forage` (biome-gated)
- `track`
- `salvage <item>`

**Magic & the Folding**
- `cast <spell>` syntax — no memorization needed
- Three damage channels: physical / magical / conviction (descriptive,
  not numbers)
- Spellcasting skill + willpower scaling, framed as the Folding lore
  in §4.4
- Concentration loss under damage / position change

**Consumables**
- Potions, bottle tiers (clay → glass → sealed → crystalline) — in-
  fiction as glass-craft quality
- Toxicity ("your gut churns")
- Potion bandolier (belt slot)
- Healing salves vs. heal-spell buffs
- Grenades (alchemical throwables; see the shared-cooldown beat above)

**Advanced combat**
- `kick` (auto-routes to stomp/knee based on position)
- `taunt`
- Grapple basics: `takedown`, `escape`, `reversal`
- Positions: standing / prone / grappled

### Tier 3 — Tease only (player sees, does not master)

- **Living NPCs.** Schedules (NPCs come and go by time of day), sleeping
  at night, NPC↔NPC conversations, caravans passing through the area
  (not boardable — just visible)
- **Factions & reputation.** The concept is flagged. Mutations cause
  reactions. No deep faction quests here.
- **Companions.** The player sees one in the world. The
  gear-loss-on-dismiss gotcha is hinted at in lore (e.g., an Opened
  veteran muttering about a lost companion's gear).
- **Stealth / steal / locked containers.** A sneak NPC is visible; a
  locked chest sits behind a door the player cannot yet open.
- **Long-distance travel.** Roads exist; the long-hike-out is real but
  arduous. No fast-travel beyond the hub portal.
- **Sanctuary mutator.** Safe-haven rooms — the school grounds are
  flagged sanctuary, as is the hub square.

### Tier 4 — Out of scope (do not touch in newbie area)

- Mutation evolution / chrysalis catalysts
- Aging-optimized alchemy
- Combat-position dominance math (never explained, no hard numbers)
- Faction-specific quest chains
- Anything that doesn't already ship in DOGMud (no inventing systems here)

---

## 6. Architecture — hub-and-spoke with concentric spokes

### 6.1 Topology

```
            [Spoke F outer]══════[Spoke A outer]
              |                     |
            [F middle]          [A middle]
              |                     |
            [F inner]           [A inner]
              |                     |
[E outer]──[E middle]──[E inner]──[HUB]──[B inner]──[B middle]──[B outer]
              |                     |
                                [D inner]      ▒ future-system slot ▒
                                  |              (slot pre-reserved
                                [D middle]        but unbuilt)
                                  |
                                [D outer]══════[C outer]
                                                 |
                                              [C middle]
                                                 |
                                              [C inner]
```

- `|` and `──` are spoke paths (rooms connect in sequence)
- `══` are **lateral outer-ring connectors** between adjacent spokes
- `▒ slot ▒` is reserved but unbuilt — a future spoke (ranged weapons,
  mounted combat, whatever) drops in here without disturbing what exists.
  The slot's diagrammatic position is **illustrative only**; final
  geographic placement is deferred to the future-system author (see §11.6)

### 6.2 The hub town (~15–20 rooms)

The central plunge pool basin. Stilted houses on the lake edge, a small
Chrysalis-Church school on a basalt shelf, a folk-tradition counterpoint
(an old woman who keeps the older fire, e.g. a folk healer with pre-Church
remedies and stories), an inn, a healer, a general store, a bank, the
awakening site (the central plunge pool itself), the portal room, and
several side streets / alleys / shore paths to provide texture.

The hub is **sanctuary-flagged** (mutator). The player cannot be attacked
here. NPCs cannot drag combat into the hub. The `sanctuary` mutator
(`_datafiles/world/dogmud/mutators/sanctuary.yaml`) also bakes in a 5×
regen multiplier for HP / SP / CP, so first-hour respite — wounds close,
breath deepens, resolve returns — is automatic. Sanctum Basin used the
same mutator on Academy Hall; the behavior carries over.

### 6.3 Spoke roster — 6 active + 1 reserved

| Spoke | System(s) | Geographic flavor | Outer-ring landmark |
|---|---|---|---|
| **A — Martial** | Combat basics, defense, kick/trip/special-cooldown, taunt | Dry training canyon, abandoned watch-post | Bandit captain at a ruined watchtower |
| **B — Forge & Forge-Craft** | Smithing crafting, salvage, slot gear, component bag | Smithy → talus slope → mine shaft | Cave-in survivor / stone-blooded beast deep in the mine |
| **C — Herbalism & Alchemy** | Forage, recipes, brewing, toxicity, bandolier | Sheltered plunge pool → reedy marsh → poison swamp | Spirit-of-the-swamp aberrant |
| **D — Wilderness & Tracking** | Forage, track, hunt, sleep mechanics in field | Scrub steppe → predator territory | Apex pack-leader / scabland raptor |
| **E — The Folding (magic)** | Cast, folding lore, willpower, concentration, channels | Observatory ruin → meditation grove → reality-thin scabland | "Unfolded" aberrant — a long-dead caster's escaped folds |
| **F — Lore & Folk Tradition** | Faction tease, dialogue depth, social, schedules | Outlying farmstead → standing stones → old shrine | The Orbital Stone (no fight — quiet discovery) |
| **G — *Future slot*** | TBD (ranged weapons / mounted / etc.) | TBD | TBD |

Spoke F is the "soft" spoke — no boss, all social and discovery. It
exists to ensure the player who only plays combat-y content still sees
the world's texture.

### 6.4 Concentric rings within each spoke

Every active spoke has the same internal structure:

| Ring | Rooms | Difficulty | Function |
|---|---|---|---|
| **Inner** | 4–6 | Trivial. Cannot kill an attentive newbie. | Lesson + introductory NPC + small repeatable quest |
| **Middle** | 5–8 | Modest. Can knock you out → death-recovery → hub. | Skill-rank quest + slot-filler gear drop + repeatable |
| **Outer** | 5–8 | Real challenge. Group fights, environmental hazards, mini-bosses. | Cert quest culminating in the spoke's landmark/boss |

Lateral outer-ring connectors (the `══` paths) link adjacent spokes' outer
rings, letting an explorer roam laterally without returning to the hub.

### 6.5 Approximate room budget

| Region | Rooms |
|---|---|
| Hub town | 15–20 |
| Each active spoke (6 × ~20) | ~120 |
| Lateral outer-ring connectors | ~10–15 |
| Reserved future-spoke slot | 0 (placeholder rooms allowed at slot boundary) |
| **Total** | **~150** |

Floor is 150; ceiling is whatever feels right per spoke during authoring.
Some spokes (E — Folding, F — Lore) may end up larger; others (A —
Martial, the most archetypal) may be tighter.

---

## 7. Reward structure — per-spoke certification

There is **no graduation event**. There is no "you have finished the
tutorial" moment. Each ring within each spoke grants its own
**certification reward** when completed. The reward gradient is the
motivation; the portal is always available, so a player can leave any
time and walk away with whatever they've earned.

### 7.1 Per-ring reward tiers

| Ring | Reward weight | Examples |
|---|---|---|
| **Inner** | Small | A few skill uses worth of progress, one minor gear piece, a small purse, a hint of lore |
| **Middle** | Medium | Skill rank bump, a slot-filler gear piece, a granted recipe |
| **Outer** | Large | Stat bump, a granted spell **or** a notable gear piece, a better recipe, a faction nod |

### 7.2 Reward type per spoke (rough sketch — refined in chunk specs)

| Spoke | Inner | Middle | Outer |
|---|---|---|---|
| **A — Martial** | Combat-skill seeds, a basic weapon | Weapon-combat rank bump, slot-filler armor piece | Strength or Dexterity bump, a notable weapon, granted special-move proficiency |
| **B — Forge** | Smithing seeds, ingot starter | Smithing rank bump, component bag, a recipe | Strength bump, a forged weapon, advanced recipe |
| **C — Alchemy** | Herbalism seeds, vials | Alchemy rank bump, potion bandolier, a recipe | Vitality bump, a granted potion stockpile, advanced recipe |
| **D — Wilderness** | Tracking seeds, scout gear | Forage rank bump, slot-filler stamina gear, recipe | Perception bump, a wilderness garment, hunting kit |
| **E — Folding** | Spellcasting seeds, focus item | Spellcasting rank bump, a basic spell granted | Willpower bump, a notable spell granted (e.g. a heal or a ward) |
| **F — Lore** | Charisma seeds, a folk charm | Rhetoric rank bump, dialogue keywords unlocked, lore item | Charisma bump, a faction nod, a discovery moment (Orbital Stone) |

A player who completes 3 spokes walks out with a functional kit. A player
who completes all 6 walks out fully outfitted across most slots, with
several skill ranks, several stat bumps, a couple of granted spells, and
a handful of recipes. Both are valid outcomes.

### 7.3 Repeatable quests (bankroll layer)

Each spoke also hosts at least one **repeatable quest** — a smaller loop
that grants gold and minor items. Repeatable quests are how a deep-
diving player builds a bankroll before leaving. Cadence (daily, per-
cooldown, always-available with diminishing returns) is a per-chunk
decision; the spec mandates only that *every* spoke has at least one
repeatable.

---

## 8. Veteran skip path

**The portal is always available the moment a character is Awakened.**
That is the only gating condition — no cert required, no quests required,
no graduation required. A veteran player rolling a new character:

1. Logs in, lands in the hub
2. Walks to the awakening site (central plunge pool)
3. Receives a random mutation (the rite is short and unskippable, but
   measured in seconds, not minutes)
4. Walks to the portal room
5. Steps through, lands in the wider world

End-to-end: a few minutes. The veteran has accepted the bare-bones
outcome — no stat bumps, no granted spells, no slot-filler gear, no
recipes — but they're out, and they have the mutation that defines this
character.

A new player who lingers will, over an hour or several, work through the
spokes and walk out fully kitted.

The portal's destination is an **open question** to be resolved in the
chunk-1 spec (see §11.1).

---

## 9. Critical path & replayability

### 9.1 Critical path

There is no critical path in the traditional sense. The Awakening is the
only mandatory beat. Everything else is optional reward-gated content. A
player who never enters a spoke can still leave the zone.

A *recommended* path is implied by the inner-ring quests and dialogue
hints — but it is never enforced.

### 9.2 Replayability sources

1. **Six spokes worth of cert chain.** Players who want full kit return
   for the spokes they skipped.
2. **Repeatable quests in every spoke.** Bankroll-building loops.
3. **Lateral outer-ring connectors.** Once the player reaches an outer
   ring, lateral exits to neighboring spokes' outer rings create
   non-obvious shortcuts and exploration routes.
4. **Spoke F discovery layer.** The Orbital Stone, folk-tradition NPC
   conversations, and standing stones provide a quieter "I never
   noticed that before" replay layer.
5. **Mutation variety.** Each new character gets a different random
   mutation; spokes may have small mutation-flavored dialogue branches
   (e.g., an NPC reacts differently to a player with the Tail mutation).

---

## 10. Chunked work plan

Modeled on the Mob Aliveness chunk structure. Each chunk is a self-
contained piece of work with its own sub-spec, plan, and verification
gate. After Chunk 1 ships, the new area is **playable** end-to-end —
spokes 2–7 each *replace a stubbed spoke* with the real authored content.
The old Sanctum Basin only disappears in the final chunk.

| Chunk | Scope | Approx new rooms | Sub-spec needed? |
|---|---|---|---|
| **0** | This spec; per-chunk specs sketched at outline level | 0 | (this doc) |
| **1** | **Hub town authored end-to-end.** Awakening site + rite. Portal mechanic. Veteran skip path works end-to-end. **All six active spokes stubbed** as single placeholder rooms with "under construction" flavor — playable, walkable, but no real content. (Spoke G — the reserved future slot — has *no exit* from the hub until a future-system chunk claims it; it does not get a stub.) School cleric, innkeeper, healer, banker, general-store merchant, folk-tradition NPC, hub-square crier all authored. | ~20 + 6 stubs | Yes |
| **2** | **Spoke A — Martial.** Full inner/middle/outer + boss + repeatable + cert rewards. Combat-system teaching beats. Replaces the Spoke A stub. | ~20 | Yes |
| **3** | **Spoke B — Forge & Forge-Craft.** | ~20 | Yes |
| **4** | **Spoke C — Herbalism & Alchemy.** | ~20 | Yes |
| **5** | **Spoke D — Wilderness & Tracking.** | ~20 | Yes |
| **6** | **Spoke E — The Folding.** Authoring of in-fiction folding lessons + Spoke-E reward chain. | ~20 | Yes |
| **7** | **Spoke F — Lore & Folk Tradition.** Includes Orbital Stone discovery beat. Plus authoring of all lateral outer-ring connectors. | ~25 | Yes |
| **8** | **Polish.** Cross-spoke balance pass on rewards. Repeatable-quest tuning. Hint-coverage audit (every triggerable thing is hinted). In-character no-numbers audit. Schedule integration for hub/outlying NPCs. NPC↔NPC conversations seeded. | 0 (tuning only) | Light |
| **9** | **Cutover.** Connection to wider world (exit road from outer-ring exit point + portal destination). Sanctum Basin retired. Config `StartRoom` and `DeathRecoveryRoom` updated. Old Sanctum Basin rooms/mobs/quest deleted. PATCH_NOTES update. | ~5 (connection rooms) | Yes |

### 10.1 Per-chunk acceptance criteria (sketch)

Each chunk's sub-spec must define:

- **Room manifest** (ids, names, biome, exits, sanctuary flags)
- **Mob manifest** (ids, archetype, schedules if applicable, dialogue
  tree references)
- **Item manifest** (drops, quest items, slot-filler gear, recipes
  granted, granted spells)
- **Quest manifest** (one cert quest per ring + one repeatable; all
  hint coverage)
- **Lesson coverage** (which Tier 1 / Tier 2 items from §5 are touched
  by this spoke and where)
- **No-hard-numbers audit** (every player-facing string sweeps clean)
- **Smoke-test plan** (the specific things a tester verifies before
  declaring the chunk done)

---

## 11. Open questions

These are flagged for resolution **during chunk authoring**, not as
blockers to this spec.

### 11.1 Exit destination (resolve in Chunk 1)

The hub **portal** lands at **Thornwall City — Temple Interior
(room 468)**, where Temple Priest Olen tends the altar. The temple is
non-denominational ("a place for contemplation, community, and the
management of tithes"), which gives the new arrival a tonal shift away
from the Chrysalis-Church-sponsored hub and into the wider world's
communal mode. Olen becomes the player's first wider-world NPC —
calm, welcoming, sanctuary-feeling — without re-firehosing tutorial
content.

The **long-hike-out** destination is still open. Likely the
`thornwall_outskirts` zone or the existing road network feeding into
Thornwall City, to be confirmed in the Chunk 9 cutover sub-spec.

(Splitting portal and hike-out landings is acceptable — the portal
delivers a controlled "safe spot" arrival, the hike delivers a more
adventurous "you walked here" arrival.)

### 11.2 Death-recovery room (resolve in Chunk 1)

The current global `DeathRecoveryRoom: 75`. Options:

- **Reuse global.** Simplest; consistent with rest of game.
- **Newbie-specific recovery room in the hub.** Softer first-hour
  experience — newbie deaths return to a sanctuary-flagged room in the
  hub town instead of a distant recovery shrine.

Default: **newbie-specific recovery in the hub** while the character is
still in the zone, falling back to the global recovery room once they've
left. Implementation might be a flag on the player or a per-zone config.

### 11.3 Repeatable-quest cadence (resolve per spoke)

Options: daily (24-hour real cooldown), per-N-rounds, always-available
with diminishing rewards on repeat. Different spokes may use different
cadences (e.g., Forge is always-available because materials are
continually consumed; Wilderness is daily because hunts feel weighty).

### 11.4 Gear-economy stance (resolve in Chunk 8 polish)

Two philosophies:

- **Slot-fillers.** Newbie gear is intended to be replaced quickly once
  the player reaches mid-tier content. Sets early expectations.
- **Sticky early gear.** Some newbie cert rewards are designed to remain
  competitive for hours of wider-world play. Encourages exploration.

Default: **mixed** — most newbie gear is slot-filler, but a few cert
rewards (e.g., the Spoke E granted spell, the Spoke C bandolier) are
genuinely sticky.

### 11.5 Mutation-flavored dialogue (resolve per spoke)

Whether NPCs should react differently based on the player's specific
mutation (tail, extra arms, etc.). Doing this thoroughly is a lot of
authoring work. Doing it lightly is high-impact flavor.

Default: **lightly**. One or two NPCs per spoke notice the player's
mutation. The rest treat the player as a generic Opened.

### 11.6 Future-slot placeholder

The reserved Spoke G slot — should the spec lock in *where* it attaches
geographically (between which existing spokes)? Knowing the slot in
advance helps Chunk 1 stub it; not knowing lets the future-system author
choose.

Default: **leave un-located**. Future-system authoring decides geography.

---

## 12. Out of scope

- New game systems invented for this rework (we teach what ships, not
  what's planned)
- Stat-reroll mechanic (removed from game design)
- PvP, guilds (do not exist in DOGMud)
- Caravan boarding for fast travel (not a system that exists)
- Anything in Tier 4 of §5
- Refactoring of any production code outside the newbie area (each
  chunk's smoke test stays inside the new zone until Chunk 9 cutover)

---

## 13. Migration plan (summary; details in Chunk 9 sub-spec)

1. Through Chunks 1–8, the new zone is built and tested alongside the
   live Sanctum Basin. `StartRoom` continues to point at Sanctum Basin
   (113) until Chunk 9.
2. In Chunk 9:
   1. New zone's hub Awakening room becomes the new `StartRoom`
   2. Death-recovery routing updated for the new zone
   3. Wider-world exit road / portal destination wired in
   4. Sanctum Basin rooms (`_datafiles/world/dogmud/rooms/sanctum_basin/`),
      mobs (`_datafiles/world/dogmud/mobs/sanctum_basin/`), and the
      tutorial quest (`_datafiles/world/dogmud/quests/1-the_sanctum_trials.yaml`)
      are deleted
   5. Any references to Sanctum Basin in `world.md` are updated
   6. PATCH_NOTES.md entry documents the cutover
   7. Pre-push SOP boot-test verifies clean load
3. Existing characters who have already passed Sanctum Basin are
   unaffected — they're already in the wider world. Existing characters
   *mid-tutorial* in Sanctum Basin (unlikely in prod given low traffic,
   but possible) need a migration: a one-time relocation to the new
   zone's hub on next login. Implementation TBD in Chunk 9 sub-spec.

---

## 14. Approvals

- [ ] Spec reviewed by user
- [ ] Chunk 1 sub-spec authored (separate doc)
- [ ] Implementation plan (writing-plans) generated

