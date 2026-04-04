# Rogue Lair Zone Sketch — Chrysalis Undercity

## Zone Metadata

- **Zone:** Thornwall City (extends existing zone)
- **Folder:** `_datafiles/world/dogmud/rooms/thornwall_city/`
- **Biome:** cave
- **Room IDs:** 500–509 (10 rooms)
- **Depth:** z:-3 (one level below smuggler tunnels at z:-2)
- **Entry:** Hidden locked grate in room 487 (Collapsed Passage)

## Design Notes

### Atmosphere
Every room should feel cluttered, cramped, and dangerous. Random
flavor objects (bric-a-brac) with `look` descriptions throughout:
chrysalis resin drippings, crude scratched symbols, discarded
lockpick fragments, stained rags, bone dice, stolen merchant
ledgers, cracked vials. Nouns with look descriptions in every room.
Players should feel there's a trap around every corner.

### Mob Behavior — Hit and Run
ALL mobs in this zone fight dirty:
- Sneak → surprise strike → fight 2-3 rounds → flee → sneak → repeat
- All mobs have high skullduggery (8-15) and high dexterity
- All mobs should have `sneak` and `flee` in idle/combat commands
- Mobs re-sneak between engagements (idle sneak)
- Players with high perception/skullduggery counter this

### Boss — First Real Boss Encounter
The Chrysalis Phantom should be a genuine challenge for endgame
players. Not a gear check — a skill check. Players need stealth
detection, good timing on interrupts, and poison resistance.

### Loot Philosophy
- Boss weapon drops at 20% chance (rare, aspirational)
- Other zone items have rarity but at least 2 guaranteed items
  are always present in the zone via chest spawns
- Equipment fills underrepresented slots: shoulders, fist,
  wrist, gloves, feet
- All items chrysalis-themed even if effects are mechanical

## Room List

### Room 500 — "Sealed Drain Grate"
Coord: (2, -2, -3). Entry below Collapsed Passage (room 487).

Rusted iron grate half-hidden beneath rubble and chrysalis
residue. Requires `search` to discover the exit. Locked grate
(difficulty 15). On failed pick: minor poison trap (buff).
Scratch marks show it's been pried open before. Chrysalis
resin beads along the edges like dried amber.

Nouns: grate, rubble, scratch marks, resin beads

### Room 501 — "Resin-Slicked Shaft"
Coord: (2, -3, -3). Near-vertical shaft.

Shaft coated in hardened chrysalis resin shimmering faintly.
Hand/footholds carved into tacky resin. Air thick and warm.
TRAPPED: chrysalis-tipped dart trap (poison debuff, difficulty
12). Discarded rope fragments hang from a rusted piton above.

Nouns: resin, piton, rope fragments, footholds

### Room 502 — "Warded Corridor"
Coord: (2, -4, -3). Branches east. Key trap room.

Low corridor lined with pulsing chrysalis growths. Thin
filaments stretch across passage — some natural mycelium, some
deliberately placed tripwires. TRAPPED: alarm filaments
(difficulty 18). If triggered without defusing, spawns 1-2
resin hounds in this room. Crude ward symbols scratched into
walls in what looks like dried blood.

Nouns: filaments, ward symbols, chrysalis growths, blood marks

### Room 503 — "Fungal Nook"
Coord: (2, -1, -3). North of entry. Side chamber.

Cramped chamber where luminescent fungi grow in dense clusters
on chrysalis-laced stone. A battered chest sits half-buried in
fungal growth, its surface etched with crude wards. Locked
(difficulty 20) + trapped (chrysalis spore burst — blindness
debuff). Contains: chitin spaulders (shoulders).

Nouns: fungi, chest, wards, spore clusters, stone shelf

### Room 504 — "The Crawl"
Coord: (3, -4, -3). East branch.

Passage so low you crouch. Walls scratched with fingernail
marks and dried chrysalis residue. Something has been dragged
through here recently. Stealth detection check on entry — mobs
in room 505 detect non-sneaking players. Scattered bone dice
and a torn playing card lie in the dust.

Nouns: scratches, drag marks, bone dice, playing card, residue

### Room 505 — "Chrysalis Den"
Coord: (2, -5, -3). Hub room. Main mob camp.

Roughly circular chamber, chrysalis growth on every surface in
overlapping layers. Hammocks of woven resin hang from ceiling.
Crude weapon racks, a cookfire pit with cold ashes, stolen
goods piled against walls — merchant sacks, a broken loom, a
barrel of questionable wine. This is home to the skulkers.

Mobs: 2-3 Chrysalis Skulkers
Nouns: hammocks, weapon racks, cookfire, merchant sacks, wine
barrel, broken loom, stolen goods pile

### Room 506 — "Resin Armory"
Coord: (1, -5, -3). West of den. Locked (difficulty 22).

Alcove packed with chrysalis-hardened weapons and armor. Racks
hold daggers with resin-coated blades and gauntlets studded
with crystallized chrysalis nodes. A workbench shows signs of
recent use — chrysalis shavings, a pot of binding paste, molds
for knuckle weapons. Everything crafted, not looted.

Contains: Chrysalis Knuckles (fist weapon), Phantom's Wraps
(gloves). Item respawn: 30 real minutes.
Nouns: weapon racks, workbench, shavings, paste pot, molds,
dagger rack

### Room 507 — "The Listening Post"
Coord: (3, -5, -3). East of den. Fence NPC.

Narrow niche overlooking the den through a slit in the wall.
A crude speaking tube runs up through the rock. Maps of
Thornwall's sewer system are pinned to the wall with chrysalis
resin tacks. A hooded figure sits in the shadows, dealing in
whispers.

NPC: Whisper (fence). Sells: lockpick sets (3 tiers), disarm
kits, chrysalis resin vials. Buys stolen goods.
Nouns: wall slit, speaking tube, sewer maps, resin tacks

### Room 508 — "Chitin Throne"
Coord: (2, -6, -3). Boss room.

Chamber dominated by a chair carved from a single massive piece
of chrysalis chitin, its surface covered in iridescent whorls.
The air shimmers with residual mutation energy. Trophies hang
from the walls — guard badges, merchant seals, a broken sword
with a noble crest. The room smells of chrysalis and fear.

Boss: Chrysalis Phantom (statpool 300)
Nouns: throne, trophies, guard badges, merchant seals, broken
sword, noble crest, chrysalis whorls

### Room 509 — "The Stash"
Coord: (2, -7, -3). Behind throne.

Hidden compartment lined with chrysalis resin to preserve its
contents. The lock is intricate — multiple pins, each trapped
independently (difficulty 30, multi-trap: poison + alarm).
Velvet-lined shelves hold the best of the rogues' hoard.

Contains: Resin-Laced Bracers (wrist), Silkstep Boots (feet).
Item respawn: 45 real minutes.
Nouns: resin lining, lock mechanism, velvet shelves, hoard

## Adjacency Map

```
              [503]
                |
[487(z:-2)] → [500]
   (down)       |
              [501]
                |
              [502]——[504]
                |       |
              [505]←————+
             / | \
         [506] | [507]
             [508]
               |
             [509]
```

## Exit Connections

```
487 (down) → 500 [hidden + locked]
500 (up) → 487, (north) → 503, (down) → 501
501 (up) → 500, (south) → 502
502 (north) → 501, (east) → 504, (south) → 505
503 (south) → 500
504 (west) → 502, (south) → 505
505 (north) → 502, (west) → 506, (east) → 507, (south) → 508
506 (east) → 505
507 (west) → 505
508 (north) → 505, (south) → 509
509 (north) → 508
```

## Mob Details

### Chrysalis Skulker (×2-3, rooms 505/504)
- Statpool: 100-120
- Archetype: fighting
- Species: human (1)
- Skills: skullduggery 12, weapon-combat 8, unarmed-combat 6
- High Dex training, moderate Str/Per
- buffids: [9] (spawn hidden)
- Combat loop: sneak, flee, sneak — surprise strike focus
- Idle: sneak (re-hide between fights)
- Combat: flee + emotes, poisoned dagger stabs
- Item drops: chrysalis dagger (10%), stolen gold (50-100g)
- Respawn: 10 real minutes

### Resin Hound (×1-2, room 502 alarm spawn only)
- Statpool: 70
- Archetype: fighting
- Species: canine (2)
- High Per (tracks hidden), high Dex
- Only spawns if alarm trap in 502 triggers
- Aggressive, no flee — they chase
- Respawn: only on alarm trigger (not timed)

### Chrysalis Phantom (boss, room 508)
- **Statpool: 300** — first real boss encounter
- Archetype: fighting
- Species: human (1)
- Skills: skullduggery 20, unarmed-combat 15, weapon-combat 10
- Mutations: chameleon-skin (flavor — implemented via buffids: [9])
- Extremely high Dex (training 60+) and Per (training 40+)
- Dual-wields chrysalis knuckles (fist subtype, poison on hit)
- buffids: [9] (spawn hidden)
- Combat commands: sneak, flee, emotes about phasing/shimmering
- Combat AI: flee after 3-4 rounds, re-sneak, surprise strike
- activitylevel: 70 (aggressive special moves)
- Item drops: chrysalis knuckles (20%), phantom's cowl (15%),
  stolen hoard key (100% — opens stash if player doesn't pick)
- Respawn: 30 real minutes (boss respawn)
- charm_immune: true

### Whisper (fence NPC, room 507)
- Statpool: 80
- Not hostile, charm_immune
- Shop: lockpick sets × 3 tiers, disarm kits, chrysalis vials
- Buys stolen goods at good rates
- Idle: whispered emotes, examining goods
- High perception (notices everything)

## Item Details

### Equipment (Chrysalis-themed)

| Item | Slot | Subtype | Drop Source | Drop % | Key Stats |
|------|------|---------|------------|--------|-----------|
| Chrysalis Knuckles | weapon | fist | Boss (508) | 20% | Unarmed scaling, poison on hit debuff |
| Chitin Spaulders | shoulders | wearable | Chest (503) | guaranteed | Physical mitigation, Str bonus |
| Phantom's Wraps | gloves | wearable | Armory (506) | guaranteed | Dex bonus, magical mitigation |
| Resin-Laced Bracers | wrist | wearable | Stash (509) | guaranteed | Skullduggery stat mod bonus |
| Silkstep Boots | feet | wearable | Stash (509) | guaranteed | Dex bonus, movement stamina reduction |
| Phantom's Cowl | head | wearable | Boss (508) | 15% | Per + Dex bonus, sneak-themed |

### Tools — Lockpicks

Lockpicks have a `uses` field. Each failed pin in the lockpick
minigame consumes 1 use. Successful pins don't consume. When
uses hit 0, they break. Tier 1 bought, Tiers 2-3 crafted.

| Item | Uses | Source | Skill Gate | Price/Mats |
|------|------|--------|-----------|------------|
| Iron Lockpicks | 3 | Buy from Siv/Whisper | — | 5g |
| Steel Lockpicks | 8 | Craft (blacksmithing) | Blacksmithing 12 | TBD — must cost > 5g in mats |
| Master Lockpicks | 20 | Craft (jewelcrafting) | Jewelcrafting 20 | TBD — must include rare ingredient |

### Tools — Disarm Kits

All kits are single-use (consumed on attempt, success or fail).
Higher tier kits give a stat mod bonus to the defuse opposed
roll, making harder traps viable.

| Item | Bonus | Source | Skill Gate | Price/Mats |
|------|-------|--------|-----------|------------|
| Basic Disarm Kit | +0 | Buy from Siv/Whisper | — | 30g |
| Reinforced Disarm Kit | +15 | Craft (blacksmithing) | Blacksmithing 15 | TBD — must cost > 30g in mats |
| Precision Disarm Kit | +30 | Craft (jewelcrafting) | Jewelcrafting 22 | TBD — must include rare ingredient |

### Crafted Tool Material Philosophy

Crafted tools must NOT be cheaper than buying the base version.
The value of crafting is better quality, not gold savings.
Higher tier recipes should require at least one ingredient that
is either:
- A rare mob drop (not sold by merchants)
- A zone-specific forage item (must search it out)
- Expensive enough that total mat cost > base tool shop price

Exact material lists finalized during item creation. The goal:
iron lockpicks are pocket change, steel lockpicks take effort,
master lockpicks take real investment in both skill and mats.

## Respawn Rates

| Thing | Rate | Notes |
|-------|------|-------|
| Skulker mobs | 10 real minutes | Fast — zone should feel alive |
| Boss (Phantom) | 30 real minutes | Boss timer, worth camping |
| Resin Hounds | alarm-triggered only | No timer, spawn on trap |
| Chest items (503) | 30 real minutes | Guaranteed shoulders |
| Armory items (506) | 30 real minutes | Guaranteed gloves + fist |
| Stash items (509) | 45 real minutes | Best loot, slower respawn |
| Locks | relock 15 real minutes | Locks re-engage periodically |
| Traps | re-arm with lock | Traps reset when lock relocks |

## Mechanics to Implement

### 1. Flee Rework (global)
Opposed roll: fleeer's Dex + skullduggery vs blocker's Dex +
unarmed-combat. Same formula for players and mobs. Replaces
the current flat dex-ratio calculation. Applied everywhere.

### 2. Defuse Command (global)
Wire up existing stub. Opposed: Perception + skullduggery vs
trap difficulty. Requires disarm kit (consumed). Success
disarms trap. Failure triggers trap effects.

### 3. Siv's Shop Update
Add lockpick sets (3 tiers) and disarm kits to Fence Dealer
Siv's shop in Thornwall.

### 4. Lockpick Items
Create 3 tiers with `lockpicks` item type. Uses field controls
durability.

### 5. PvP Taunt Loophole (pinned for later)
Taunt command missing CanPvp check — players deal conviction
damage to each other in non-PvP rooms.
