# Rogue Lair Zone Sketch — Chrysalis Undercity

## Zone Metadata

- **Zone:** Thornwall City (extends existing zone)
- **Folder:** `_datafiles/world/dogmud/rooms/thornwall_city/`
- **Biome:** cave
- **Room IDs:** 500–509 (10 rooms)
- **Depth:** z:-3 (one level below smuggler tunnels at z:-2)
- **Entry:** Hidden locked grate in room 487 (Collapsed Passage)

## Room List

### Room 500 — "Sealed Drain Grate"
Coord: (2, -2, -3). Entry room below Collapsed Passage.
Rusted iron grate half-hidden beneath rubble. Requires search
to discover, locked to pick. Stale air carrying chrysalis resin
and damp stone rises from below.

### Room 501 — "Resin-Slicked Shaft"
Coord: (2, -3, -3). Near-vertical shaft coated in hardened
chrysalis resin. Hand and footholds carved into tacky resin.
TRAPPED: tripwire triggers chrysalis-tipped poison dart.

### Room 502 — "Warded Corridor"
Coord: (2, -4, -3). Low corridor lined with pulsing chrysalis
growths. Thin filaments across passage — some natural, some
deliberate tripwires. TRAPPED: alarm filaments trigger guard
mob spawn (resin hounds) if not defused.

### Room 503 — "Fungal Nook"
Coord: (2, -1, -3). Side chamber north of entry. Luminescent
fungi on chrysalis-laced stone. Locked + trapped chest with
shoulder armor loot (chitin spaulders).

### Room 504 — "The Crawl"
Coord: (3, -4, -3). East branch. Passage so low you crouch.
Scratched walls, dried chrysalis residue. Stealth check — mobs
detect non-sneaking players entering.

### Room 505 — "Chrysalis Den"
Coord: (2, -5, -3). Hub room. Circular chamber, chrysalis on
every surface. Resin hammocks, crude weapon racks. 2-3 chrysalis
skulker mobs. Exits: north, west, east, south.

### Room 506 — "Resin Armory"
Coord: (1, -5, -3). West of den. Locked alcove with chrysalis-
hardened weapons. Fist weapon (chrysalis knuckles) + gloves
(phantom's wraps) loot.

### Room 507 — "The Listening Post"
Coord: (3, -5, -3). East of den. Niche overlooking den through
wall slit. Speaking tube to tunnels above. Fence NPC "Whisper"
buys stolen goods, sells lockpicks + disarm kits.

### Room 508 — "Chitin Throne"
Coord: (2, -6, -3). Boss room. Chair carved from massive
chrysalis chitin. The Chrysalis Phantom — chameleon skin mutation,
high skullduggery, dual chrysalis knuckles with poison, flees
and surprise strikes.

### Room 509 — "The Stash"
Coord: (2, -7, -3). Behind throne. Hidden compartment lined
with preservation resin. Complex lock, multi-trapped. Best loot:
resin-laced bracers (wrist) + silkstep boots (feet).

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
487 (down) → 500, 500 (up) → 487
500 (north) → 503, 503 (south) → 500
500 (down) → 501, 501 (up) → 500
501 (south) → 502, 502 (north) → 501
502 (east) → 504, 504 (west) → 502
502 (south) → 505, 504 (south) → 505
505 (north) → 502
505 (west) → 506, 506 (east) → 505
505 (east) → 507, 507 (west) → 505
505 (south) → 508, 508 (north) → 505
508 (south) → 509, 509 (north) → 508
```

## Boundary Connections

Room 487 (Collapsed Passage) needs:
- New `down` exit to room 500
- Exit should be a hidden exit requiring `search` to discover
- Locked with difficulty ~20, trap on failure (minor poison)

## Mob Suggestions

### Chrysalis Skulker (×2-3, rooms 505/504)
Hit-and-run rogues. High skullduggery, sneak+flee+surprise
strike combat loop. Chrysalis-hardened leather, poisoned daggers.
Statpool: 80-100. Groups: [humanoid, smuggler].

### Resin Hound (×1-2, spawns in room 502 on alarm)
Mutated guard dogs with chrysalis chitin plating. Only appear
if alarm trap in room 502 is triggered without defusing.
Aggressive, high perception (tracks hidden players).
Statpool: 60-80. Species: canine. Groups: [predatory].

### Chrysalis Phantom (boss, room 508)
Lair leader. Chameleon skin mutation → spawns hidden, re-sneaks
mid-combat via combat commands. High skullduggery + dexterity.
Dual-wields chrysalis knuckles (fist subtype, poison on hit).
Flees and surprise strikes repeatedly. Statpool: 140-160.
buffids: [9] (hidden). Combat: sneak, flee, surprise strike loop.
Groups: [humanoid, smuggler].

### Whisper (fence NPC, room 507)
Shadowy fence. Buys stolen goods at better rates than Siv.
Sells: lockpick sets (3 tiers), disarm kits, chrysalis resin
vials (poison consumable?). Charm immune. Not hostile.

## Item Suggestions (Chrysalis-themed)

### Equipment Drops
| Item | Slot | Source | Effect |
|------|------|--------|--------|
| Chrysalis Knuckles | weapon (fist) | Boss / armory | Unarmed combat, poison on hit |
| Chitin Spaulders | shoulders | Fungal nook chest | Physical mitigation |
| Phantom's Wraps | gloves | Armory | Dex bonus, slight magical mitigation |
| Resin-Laced Bracers | wrist | Stash chest | Skullduggery stat bonus |
| Silkstep Boots | feet | Stash chest | Movement stamina reduction |

### Tools (sold by Whisper + Siv)
| Item | Type | Effect |
|------|------|--------|
| Iron Lockpicks | lockpicks | Basic, breaks easily |
| Steel Lockpicks | lockpicks | Standard durability |
| Master Lockpicks | lockpicks | High durability, expensive |
| Disarm Kit | tool | Single-use trap defusing |

## Mechanics to Implement

### 1. Flee Rework
Opposed roll: fleeer's Dex + skullduggery vs target's Dex +
unarmed-combat. Same formula for players and mobs. Replaces
the current flat dex-ratio calculation.

### 2. Defuse Command
Wire up the existing stub. Opposed roll: Perception +
skullduggery vs trap difficulty. Success disarms trap. Failure
triggers trap. Requires disarm kit (consumed on use).

### 3. Siv's Shop Update
Add lockpick sets (3 tiers) and disarm kits to Fence Dealer
Siv's inventory in Thornwall.

### 4. Lockpick Items
Create 3 tiers of lockpick items with the `lockpicks` type.
Higher tier = more uses before breaking.

### 5. Skullduggery Flee Bonus
The flee opposed roll naturally incorporates skullduggery via
the formula. Higher skullduggery = better flee chance.

## Tone Notes

Claustrophobic, tense, paranoid. Chrysalis growth everywhere
gives an organic alien quality — walls breathe, light pulses.
Players feel like trespassers being watched. Hit-and-run rogues
reinforce this — shadows may stab you. Boss fight is genuinely
frustrating in a fun way — the Phantom keeps disappearing.
