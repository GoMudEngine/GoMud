# New Plymouth Sewers (#13.5) — Zone Design

**Date:** 2026-07-02
**Status:** Draft for review
**Plan slot:** ZONE_EXPANSION.md priority 13.5 (chunk 4.7) — the LAST unbuilt
zone in the plan. On completion the Build Priority Summary is 29/29.

## Purpose

Every city has a second city below it. The sewers connect the capital's
districts underground, give the Bloom trade and the criminal economy a
physical route the surface never sees, and hold the oldest stonework in
New Plymouth — older than the city, in places. The zone is a
lore-discovery fill-in: **no quest at launch**, but every criminal/Bloom
hook is built as a furnished, empty seam so later questlines (Bloom
epilogue, cooperage, espionage, a tosher quest-giver) can be wired in
without re-authoring rooms.

## Decisions (user-confirmed 2026-07-02)

1. **Scope: 20 rooms, 2 stages** (the original chunk-4.7 scope). Not
   scaled up — it is a connective fill-in, not a destination city.
2. **Lore-only, no quest** — but explicitly constructed for later
   criminal-activity / Bloom tie-ins (user's directive).
3. **Three live entrances + the Old Quarter seam** (user-confirmed with
   topology preview). Noble Quarter drain stays welded shut (stub).
4. **Difficulty: mid tunnels / high deep** (recommended default — user
   was AFK; flag at review). 4.7a ~150–250 statpool, 4.7b ~350–500 with
   one ~700 mini-boss.

## Zone identity

- **Display name:** `New Plymouth Sewers` → folder **`new_plymouth_sewers`**
  (ConvertForFilename — the recurring boot-panic gotcha).
- **zone-config.yaml:** `roomid: 6403` (entry), `defaultbiome: city`,
  `region: Windward Marches`. NOT instanced, NOT non_cartesian —
  normal Cartesian zone, panic-mode consistency checks apply.
- **Biome/lighting:** `city` biome throughout (the Old Quarter precedent —
  underground rooms stay mechanically lit; darkness, drips, and
  lamp-niches live in prose). No cave biome, no light mutators. This
  avoids the #22-B1 pitch-dark failure mode entirely.
- **IDs:** rooms **6403–6422**, mobs from **9563**, items from **40177**
  (verified free via id_inventory 2026-07-02; re-verify at build time).

## Topology & coordinates (verified against live room coords)

Surface anchors (all already have the entrance prop described in prose —
no surface prose rewrites needed beyond adding the `down` exit +
reciprocal, and un-sealing language where needed):

| Entrance | Surface room | Coord | Prop |
|----------|-------------|-------|------|
| Docks | 5515 Cutter's Lane | (−17, 83, 0) | loose grate, rope already tied |
| Common | 5618 Sweeper's Corner | (−15, 87, 0) | gutter-grate |
| Crafting | 5718 The Tanning Yard | (−15, 90, 0) | capped drain "in slightly cleaner mortar" |

The Old Quarter occupies **x=−21..−27, y=78..82, z=−1/−2**. The Common
Quarter's Old Foundation (5624) sits at (−12, 87, −1). The sewer band
**x=−15..−19, y=82..90** is verified clear at z−1/z−2. Run the
full-world coord collision scan (memory: feedback_zone_coord_planning)
before writing rooms.

**Stage 4.7a — Main Tunnels (10 rm, z=−1):** a trunk under the streets
linking the three entrance drops: Crafting drop (−15, 90, −1) → south
under the Common (drop at −15, 87, −1) → bending south-west to the Docks
drop (−17, 83, −1). ~9 trunk/junction rooms + 1 side chamber. Working
infrastructure: outfall channels, a junction vault where three flows
meet, lamp niches (toshers' candles), maintenance walkways, and — as the
tunnels run west/deeper — stonework that stops matching the surface
architecture. **The old stair down** to 4.7b sits near the Docks end.

**Stage 4.7b — the Pre-Founding Deep (10 rm, z=−2):** from the stair
(~−17, 83, −2) west along y≈82–83 to **(−24, 82, −2)**, whose west exit
opens into **The Deep Canal (6038, Old Quarter z−2)** — the canal drains
into the sewers; the Bloom's unseen logistics route becomes legible.
Rooms: older galleries, gray-material **fragments in the walls**
(threshold-only, see Lore boundary), the **sealed chamber** (echoing the
Confluence undercroft — it does not open), the **cooperage hidden
meeting room** (concealed door, furnished, recently used, EMPTY — the
Q68-epilogue seam), the mini-boss lair guarding the sealed-chamber
approach, and the three chunk-4.7 expansion stubs as described props:
- **Collapsed tunnel** at the deepest point — worked stone of a different
  character beyond, faint dry air (future pre-Founding complex /
  secondary debris field).
- **Flooded passage** — dark, deep, something large occasionally
  surfaces (future underwater section). Described only; no swim exit.
- **Welded grate** beneath the Noble Quarter — shut from the OTHER side,
  voices above (future espionage route).

Terminus-stub rule applies: none of the three stubs may invite
`go <direction>` — frame as not-passable-yet.

**Cross-zone exits to wire (both sides annotated `zone:` BEFORE first
boot — the Cascade lesson):**
1. 5515 `down` ↔ 4.7a Docks drop `up` (grate/rope)
2. 5618 `down` ↔ 4.7a Common drop `up` (gutter-grate)
3. 5718 `down` ↔ 4.7a Crafting drop `up` (capped drain — prose says
   capped; the room's `down` exit can carry an `exitmessage` of prying
   it up, or reword to pried-open)
4. Deep room (−24, 82, −2) `west` ↔ 6038 The Deep Canal `east`

6038's prose should gain a line acknowledging the outflow eastward
(small edit, in keeping with its existing "the water moves" character).

## Mobs (~10, ids 9563+)

**4.7a (statpool ~150–250):**
- Sewer rats (1–2 spawns, pack feel; correct animal speciesid — verify
  species table at build time, NOT speciesid 1)
- A slime/ooze (the wet-tunnel signature; verify a species with
  non-zero basedamage — the orb/basedamage-0 lesson)
- A feral mutated dog or cat (city cast-off, distinct Chrysalis mutation)
- **The tosher** (non_combatant, living NPC): a sewer-scavenger who
  knows every tunnel, sells nothing yet, speaks in hints — the future
  quest-giver anchor for criminal content. Small dialogue tree (who/
  tunnels/deep/finds), NO quest, no symbol content. Dialogue rules:
  first-person text, `|` blocks for long text, no semicolons, no ": "
  in prose, hint keywords must route.
- 1–2 ambient-only spawns (more rats) for respawn texture.

**4.7b (statpool ~350–500 + boss ~700):**
- Pale deep-fauna (blind, white — 2 variants, e.g. a giant blind
  salamander and a leech-swarm analog)
- A stronger slime variant (fed on what seeps from the old stone)
- **Mini-boss ~700** guarding the sealed-chamber approach — a massive
  albino alligator-analog or equivalent apex fauna. NOT a construct
  (constructs are #21/#22 vocabulary; the deep here is old, wet, and
  ANIMAL). Drops: a guaranteed trophy + modest gold-value loot via
  `character.items`.

All combat mobs pair `behavior_archetype` + `aiprofile` (the overworld
lesson) and carry loot via **`character.items`** (loot_pool is
instance-only). Mob names must pass casing.Title(); filenames
`mobid-ConvertForFilename(name).yaml`.

## Items (small set, ids 40177+)

- 1–2 mob-drop materials (e.g. rat pelts / slime residue) with
  `component_tag` fitting existing recipes where possible — reuse before
  minting.
- Mini-boss trophy (1 item).
- Optionally ONE pre-Founding curio as a lore item (`not_salable`) found
  near the sealed chamber — NO `grey-relic` tag (that tag is the
  #21/#22 pinnacle-craft economy; the sewers must not leak endgame
  reagents at mid-tier difficulty).
- No forageables (underground; no biome wiring needed).

## Lore boundary

Same restraint as the whole Eastern Arc, and it does NOT relax here even
though #22 has shipped: the revelation is a per-player quest (77), so
zone prose stays **threshold-only**. Allowed: older-than-the-city,
worked stone that doesn't match, gray material that is neither stone nor
metal, a sealed chamber nobody opened. Forbidden: ship/sky/moons/
machine/vessel, any named connection to the crash, any explanation.
The nested-rings orbital symbol may appear **at most once**, understated
(e.g. faint on the sealed chamber), or not at all — world-critic call.

## Criminal/Bloom expansion seams (built now, wired later)

1. **The Deep Canal passage** — the physical Bloom logistics route
   between 215 Lintel St's production room and the city above.
2. **The cooperage hidden room** — furnished, recently used, empty.
3. **The tosher** — the quest-giver anchor who has seen who passes.
4. **A smuggler's cache alcove** near the Docks drop (tide-marks, empty
   crates, a cold lantern).
5. The three expansion stubs (collapsed tunnel / flooded passage /
   welded Noble grate).

## Build process & acceptance

Standard pipeline: this spec → writing-plans → subagent-driven build
(rooms → mobs/dialogue → wiring/stubs), world-critic pass, mudagent
feel-test, boot-verify. Prompts must carry the standing author rules
(80-col, no ": " in prose, no semicolons in dialogue, `|` blocks,
hyphenated hinted nouns, Title() mob names, explicit git pathspecs —
never `git add -A`).

Acceptance:
- Boots clean: ValidateZoneConsistency errors=0 mode=panic, casing OK,
  0 panics; instance saves wiped before smoke tests.
- All four cross-zone exits walkable both ways in-game.
- Feel-test STRONG: entrance discovery from all three grates, the
  z−1→z−2 escalation reads, the Old Quarter emergence lands ("so THIS
  is how it moves"), lore boundary holds (0 leaks), stubs don't invite
  dead `go`.
- ZONE_EXPANSION.md updated: row 13.5 → ✅ Built (+ the stale row 22 →
  ✅ Built and the TOTAL row corrected — plan bookkeeping finishes with
  the zone).

## Non-goals

- No quest, no faction, no vendor economy (tosher sells nothing).
- No swim/boat mechanic (flooded passage is prose only).
- No Noble Quarter infiltration (welded shut).
- No new engine code expected; data-only build.
