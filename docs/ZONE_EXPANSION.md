# Zone Expansion Plan — The Windward Marches

*Geography aligned to "What the Moons Keep" novel canon.*
*Zones built in mini-stages of 10 rooms each.*

---

## Current World State — Reconciled 2026-06-19

The novel-canon geography, zone designs, and build order below are still the
guiding target. This banner reconciles them with what is actually on disk
after the newbie-area rework (Pothole Coulee) and the wilderness build-out —
work that happened outside this document's original numbering.

**Built and live (on-plan main-road progression):**
- Marches Spur Road (4000–4037), Ashwick (4015–4034), North Road — Southern
  (4038–4062), The Fernway interlude (4147–4156), **Stillwater (4100–4146) —
  now fully populated** (25 rooms with mob spawns, 6 shop files; the old
  "NPCs not placed" note is stale → status promoted to ✅ Built).

**Built and live (off-plan — not in the numbered table, but real content):**
- **Pothole Coulee (5200–5371, 169 rooms)** — the current newbie start zone
  (StartRoom 5200). **Replaces the retired Sanctum Basin** as the tutorial
  region. Connects to Ironwind Steppe.
- Ironwind Steppe (3000–3122, 123 rooms), A Dark Forest (1002–1082),
  Stillwater Marsh (4177–4198), The Fernway South (4157–4197) — wilderness.
- Thornwall City (460+) ↔ Thornwall Outskirts (440–447) ↔ Watchers Crossing
  (420–427) — the original backwater hub.
- Dustwalk Road (400–409), Labyrinth of Low Tunnels (300–319), World Road
  (single room 2001, now a Dustwalk↔Labyrinth connector), Endless Trashheap.

**Connectivity backbone (verified from cross-zone exits):**
Thornwall City ↔ Thornwall Outskirts ↔ Watchers Crossing ↔ {Dustwalk Road,
Marches Spur Road}; Marches Spur Road ↔ {Ashwick, The Fernway}; The Fernway ↔
{North Road, The Fernway South}; North Road → **Stillwater** (north_road 4062
"North Road End" → stillwater 4100); Stillwater ↔ Stillwater Marsh; Pothole
Coulee ↔ Ironwind Steppe.

**Next-build attach points (frontiers for unbuilt zones):**
- **#5 North Road — Northern** attaches at **Stillwater's north gate (~room
  4111 / Travelers' Camp 4142)**, which already references New Plymouth, and
  runs north toward NP Outskirts.
- **#14 South Road** attaches at **Ashwick Crossroads (4014)**, whose south
  signpost already reads "Amber Valley, the Confluence."

**Known data nits to fix in passing (not blockers):**
- `north_road/4062.yaml` north exit → roomid 4100 is **missing its
  `zone: Stillwater`** annotation (movement works since roomids are global,
  but the mapper/weather-crawler want the hint). Add it.
- 3 rooms still name the retired **Sanctum Basin** as flavor:
  `dustwalk_road/405`, `thornwall_outskirts/445`, `watchers_crossing/427`.
  Re-point or genericize when convenient.

---

## Quality Standards

Every zone, room, NPC, and quest in this expansion must meet the following
standards before it ships. These are not aspirational — they are the minimum
bar.

### Room Descriptions
- **80-character line width.** MUD clients render fixed-width. No exceptions.
- **Three layers per room:** A short atmospheric sentence (what hits you first),
  a paragraph of grounded physical detail (what you see when you look around),
  and at least one sentence hinting at interaction ("A weathered signpost leans
  at the fork" — tells the player to `look signpost`).
- **Sensory variety.** Not every room leads with sight. Use sound, smell,
  temperature, texture, the feel of the ground underfoot. Vary which sense
  leads across adjacent rooms so traversal feels like movement, not reading.
- **No generic filler rooms.** "You are on a road" is not a room. Every room
  must have at least one distinguishing detail that a player could use to
  identify it without seeing the room name.
- **Season and weather awareness.** Where relevant, descriptions should
  acknowledge the biome. A steppe room and a forest room feel different even
  if both are "on a path."

### Nouns and Interactables
- Every room must have **at least 2 examinable nouns** beyond exits.
- Nouns that contain items or trigger interactions must be subtly highlighted
  with `<ansi fg="itemname">noun</ansi>` in the room description.
- Nouns should reward examination. `look signpost` should return something
  worth reading — a direction, a warning, a piece of world flavor, a clue.
- **Container nouns** (crates, hollow logs, loose stones, saddlebags) should
  exist in ~20% of non-city rooms. Some contain items. Most contain flavor.
  The uncertainty is the point.

### NPC Behavior
- Every NPC must have **idle behaviors** — things they do when no player is
  interacting with them. A blacksmith hammers. A drunk sways. A scholar
  mutters. A guard shifts weight. These fire on tick timers and make the
  world feel alive.
- NPCs must have **at least 3 dialogue topics** beyond their quest function.
  Ask them about the weather, the town, rumors, their trade. They should
  feel like people, not quest dispensers.
- NPCs in cities should have **schedules** where the engine supports it —
  different locations at different times, shops that close, taverns that
  fill up at night.
- **Mutation variety.** Every NPC with a visible mutation should have a unique
  one described in their appearance. No two NPCs in a zone should share the
  same mutation description.

### Quest Design — The Breadcrumb Rule
Every quest must be **discoverable through play alone**, with no out-of-game
knowledge required. This means:

**Multiple Entry Points:**
- Every quest must have at least **3 independent breadcrumbs** that can lead
  a player to discover it exists. Examples: an NPC mentions it in passing
  dialogue, a room description hints at something odd, an item description
  references it, a rumor at a tavern points toward it.
- `ask <npc> quest` and `ask <npc> task` must always work for quest-giving
  NPCs (per existing SOP).

**Multiple Resolution Paths:**
- Every quest stage must have at least **2 valid approaches** a player could
  reasonably attempt. If the "intended" path is to `give letter to Brennan`,
  a player who tries `ask Brennan about letter` while holding it should also
  work.
- **Elephant path testing:** Before finalizing any quest, walk through it as
  a player who is trying obvious things. What would you try first? Does that
  work? What's the second thing you'd try? Does THAT work?
- NPCs who receive quest items must have `onGive` scripts (per give.go SOP).
  NPCs who should NOT receive items need rejection scripts.
- Quest items that can be lost must have recovery paths (dialogue nodes on
  the quest giver, or respawning sources).

**Quest Gating:**
- Use `questRequired` over `requires` for quest-gated dialogue nodes.
- Never use `expiryPeriod` unless timed urgency is the deliberate design.
- Every `grantsQuest` node must have `questExcluded` to prevent double
  completion.

**No Dead Ends:**
- If a player has a quest active, there must always be a discoverable next
  step. If they're stuck, an NPC in the zone should have a hint dialogue
  that nudges without solving.
- Every quest item must have a verified source (mob drop, room spawn, NPC
  gift, container). If the item exists but can't be obtained, the quest
  is broken.

### City Design — Making Them Feel Alive
Cities of 40+ rooms must have:
- **Districts** with distinct character (architecture, smells, NPC types,
  wealth level).
- **Ambient NPCs** who wander between rooms, creating the sense of a
  populated space. Guards on patrol routes, merchants moving between
  shop and home, children running through streets.
- **Services** appropriate to the city's size: shops, inns, banks, temples,
  trainers, crafting stations.
- **Overheard conversations** — room scripts that fire ambient dialogue
  between NPCs, giving players world lore and quest hints passively.
- **At least one questline per district** for major cities.
- **Layered discovery** — a player's first visit reveals the surface; return
  visits with more knowledge or quest progress reveal hidden areas, new
  dialogue, and deeper lore.

### Cartesian Consistency
All rooms in the world must be placeable on a 2D coordinate grid without
overlaps. If you walk north then east then south then west, you must
arrive back where you started. No "impossible geometry" — the world is
a consistent physical space.

**Rules:**
- Every room has an implicit (x, y) coordinate. North = +y, south = -y,
  east = +x, west = -x. Diagonal exits (NE, NW, SE, SW) move both axes.
- **No overlaps.** Two rooms cannot occupy the same coordinate. Before
  placing a new room, verify that its coordinate is unoccupied by any
  existing room in any zone.
- **No wormholes.** If room A has a north exit to room B, then room B
  must have a south exit back to room A (or no exit, for one-way passages
  like cliffs or falls — but the coordinate relationship must still hold).
- **Cross-zone boundaries must be consistent.** When a road crosses from
  one zone folder to another, the coordinates must be continuous.
- Up/down exits are z-axis and don't affect x,y position.
- "Enter"/"leave" exits (e.g., entering a building from a street) occupy
  an interior coordinate offset — typically the same x,y as the exterior
  room but on a separate interior layer.

**Hidden Coordinates:**
Every room YAML should include a `coord` field for reference:
```yaml
coord:
  x: 0
  y: 0
  z: 0
```
These are not shown to players but allow us to validate spatial consistency,
generate automaps, and catch overlap errors. When building new rooms, always
assign coordinates relative to the existing grid.

**Existing World Origin:**
Thornwall City Gate Ward (room 460) is designated as the coordinate origin
(0, 0, 0). All existing and new rooms are placed relative to this point.

**Coordinate Map of Existing Zones:**
See `docs/coordinate_map.md` for the current room coordinate assignments.
This file must be updated whenever rooms are added.

---

## World Geography — The Windward Marches

```
Geography aligned to Washington State (Pacific Northwest).
Novel directions corrected: Aldric travels WEST from Greenford to NP.

                   ┌─────────────────────────┐
                   │                         │
                   │    NEW PLYMOUTH          │
                   │    (Seattle)             │
                   │    ~150-200 rooms        │
                   │    Coastal, docks,       │
                   │    political capital     │
                   │         |        \       │
                   │         |     [Cascade   │
                   │         |      Pass Rd]  │
                   │         |          \     │
                   │    NP OUTSKIRTS    EASTERN│
                   │    ~20 rooms    HIGHLANDS │
                   │         |       ~40 rooms│
                   │         |          \     │
                   │    STILLWATER    THE HILL │
                   │    (Olympia)   (crash     │
                   │    ~30 rooms   site,      │
                   │         |      endgame)   │
                   │    [North Road]           │
                   │         |                │
                   │    CROSSROADS             │
                   │    VILLAGES               │
                   │    ~20 rooms              │
                   │         |                │
                   │    ASHWICK /              │
                   │    RETREAT SPUR           │
                   │    ~20 rooms              │
                   │      /     \              │
                   │  [spur]  [South Road]     │
                   │   /           \           │
                   │ POTHOLE       AMBER       │
                   │ COULEE        VALLEY      │
                   │ (newbie)      (Yakima)    │
                   │ + THORNWALL   ~40 rooms   │
                   │                \          │
                   │            THE CONFLUENCE │
                   │            (Tri-Cities)   │
                   │            ~70 rooms      │
                   │                 \         │
                   │              GREENFORD    │
                   │              (Walla Walla)│
                   │              ~45 rooms    │
                   └─────────────────────────┘
```

### Travel Relationships (Novel Canon)
- Maren: Ashwick → north → crossroads → Stillwater → New Plymouth
- Davan: Amber Valley → north → the Confluence → river barge north → NP
- Aldric: The Confluence (temple) → east → Greenford → west/NW → NP
- All four: New Plymouth → east through Cascade Pass → Eastern Highlands
- Pothole Coulee (newbie start) / Thornwall: backwater spur off the main road
  near Ashwick. Pothole Coulee replaced the retired Sanctum Basin as the
  tutorial region; it connects to Ironwind Steppe.

---

## Zone Breakdown — Build Order

Zones are ordered by player progression and narrative importance. Each zone
is broken into mini-stages of 10 rooms. A mini-stage is a self-contained
buildable unit: it has its own rooms, NPCs, items, and (where applicable)
quest content that functions independently.

---

### PHASE 1 — The Connection (Existing Content → Main Road)

**Purpose:** Bridge the existing backwater (Pothole Coulee newbie region /
Thornwall) to the novel's main geography. Players who have outgrown the
tutorial region need a path into the wider world.

#### Zone 1.1: Marches Spur Road
*The road from Thornwall to the main north-south highway.*

- **Biome:** Transitioning scrubland to mixed farmland
- **Size:** 15 rooms (2 mini-stages)
- **Connects:** Thornwall City (east) ↔ Ashwick Crossroads (west)
- **Theme:** Leaving the backwater. The road widens, the signs point to
  places you've only heard of. The world gets bigger.

| Stage | Rooms | Content |
|-------|-------|---------|
| 1.1a | 10 | Road rooms transitioning from steppe scrub to farmland. A waypoint shrine. A peddler NPC with rumors about New Plymouth. 2-3 ambient wildlife mobs. One roadside encounter (bandits or a broken wagon — player choice how to resolve). |
| 1.1b | 5 | Approach to the crossroads. A small waypoint inn (2-3 interior rooms). An NPC who gives directions. The road forks: north toward Stillwater, south toward Amber Valley, west toward the retreat. |

**Quest: The Peddler's Debt**
A traveling merchant on the spur road asks for help with a problem that has
two breadcrumbs (the merchant himself + a warning note at the waypoint
shrine) and two resolution paths (pay off the debt at the inn OR convince
the creditor through dialogue). Rewards: a small item + the merchant
becomes a recurring vendor on the road.

---

#### Zone 1.2: Ashwick
*Maren's home hamlet. A small rural community near the Sanctum Basin region.*

- **Biome:** Rural farmland, forest edge
- **Size:** 20 rooms (2 mini-stages)
- **Connects:** Marches Spur Road (east) ↔ North Road (north)
- **Theme:** A quiet place with a secret. The hamlet where Maren grew up
  pretending to be something she wasn't. The Autumnal Rite, the deacon,
  the community that didn't know one of its own was hollow.

| Stage | Rooms | Content |
|-------|-------|---------|
| 1.2a | 10 | The hamlet proper: a central green with a ritual circle, Deacon Ferris's chapel, Delia the herbalist's cottage (with garden), a general store, 3-4 houses, the hamlet well, surrounding farmland. Ferris has dialogue about the Rite and recent strange events. Delia teaches basic herbalism. |
| 1.2b | 10 | The outskirts: forest paths, Maren's family cottage (abandoned, examinable), the road north out of town, a woodcutter's shelter (from Ch. 9), farmsteads. Betta's farmstead is north on the crossroads road. NPCs reference "the girl who left" without naming her. |

**Quest: The Herbalist's Shortage**
Delia needs ingredients from the forest edge. Three breadcrumbs: Delia
mentions it directly, a sign on her door says "closed — no stock," and a
farmer at the well complains about the shortage. Two resolution paths:
gather the herbs yourself (foraging skill) OR arrange a trade with the
peddler on the spur road. A secondary thread hints at WHY the forest herbs
are scarce (something deeper in the woods — seeds a future quest).

**Quest: The Empty Cottage**
Maren's abandoned family home can be explored. Examining objects reveals
fragments of the family's story. Three breadcrumbs: a neighbor mentions
"the family that left," Delia speaks obliquely about "a girl I trained,"
and the cottage itself has a `for sale` sign. Resolution is discovery-only
(lore quest, no combat), but finding a specific hidden item in the cottage
unlocks a dialogue option with an NPC in New Plymouth later.

---

### PHASE 2 — The North Road (Ashwick → Stillwater)

**Purpose:** The main travel corridor north. This is the road Maren walked.
It should feel like a real journey — changing terrain, waypoint stops,
the sense of approaching something larger.

#### Zone 2.1: North Road — Southern Stretch
*Open farmland giving way to river country.*

- **Biome:** Farmland, scattered woodland, river crossings
- **Size:** 20 rooms (2 mini-stages)
- **Connects:** Ashwick (south) ↔ Stillwater (north)
- **Theme:** The road between. Wagon traffic, itinerant travelers, the
  slow accumulation of signs that you're approaching civilization.

| Stage | Rooms | Content |
|-------|-------|---------|
| 2.1a | 10 | Southern farmland road. Rolling terrain, hedgerows, farm gates. A crossroads village (3-4 rooms — the one from Ch. 8 where Vane finds Maren's trail). A roadside shrine. Betta's farmstead as a waypoint. Ambient farmer NPCs, a traveling merchant caravan (2-3 NPCs with dialogue about trade and rumors). |
| 2.1b | 10 | River approach. Terrain shifts — more water, bridges, the road following a river valley. A ford crossing. A woodcutter's camp (2 rooms). The landscape opening up as Stillwater approaches. A lone traveler NPC who warns about bandits or mentions bloodline agents on the road (quest hook). |

**Quest: The Caravan Guard**
A merchant caravan needs an escort through a stretch where bandits have been
active. Breadcrumbs: the caravan master asks directly, a farmer at the
crossroads mentions the attacks, a posted notice at the roadside shrine.
Resolution: travel with the caravan and fight off the ambush (combat) OR
scout ahead and find the bandit camp and negotiate/intimidate them into
leaving (dialogue/stealth). The caravan master becomes a contact who
offers discounted goods in Stillwater.

---

#### Zone 2.2: Stillwater
*A lake town on the north road. Maren's uncle lived here. Ulla still does.*

- **Biome:** Lakeside town, temperate
- **Size:** 30 rooms (3 mini-stages)
- **Connects:** North Road South (south) ↔ North Road North (north)
- **Theme:** A town that exists because of the lake and the road. Fishing,
  trade, a waypoint for travelers heading to New Plymouth. Comfortable
  but not exciting — the kind of place where people settle rather than
  arrive.

| Stage | Rooms | Content |
|-------|-------|---------|
| 2.2a | 10 | Town center: the main square, a large inn (The Pike & Lantern, 3 interior rooms), a general store, a blacksmith, the town well. The lake is visible from the square. Ambient NPCs: fishmongers, a town crier, a drunk telling stories about New Plymouth. |
| 2.2b | 10 | Lakeside and residences: the lakeshore (3-4 rooms along the water), fishing docks, Ulla's house (Maren's uncle's widow — has dialogue about the family, the letter, the man who went east). Boat rental for fishing. A healer's shop. |
| 2.2c | 10 | Outskirts and north road departure: the north gate, a travelers' camp outside town, the Stillwater constabulary (small, 2 rooms), lakeside caves (3 rooms, minor dungeon with low-level mobs). The road north begins to show more traffic — wagon ruts deeper, the smell of a city somewhere ahead. |

**Quest: Ulla's Silence**
Ulla knows things about Maren's family she hasn't told anyone. Three
breadcrumbs: Ulla herself (if asked about "the letter" or "the family"),
a neighbor who mentions Ulla's been "odd since the uncle died," and an
item in the uncle's old workshop (a tool with the inner orbit symbol
scratched into the handle). Two resolution paths: earn Ulla's trust
through dialogue (multiple visits, correct topics) OR find the uncle's
hidden journal in the workshop first, which unlocks a direct conversation.
Rewards: lore about Maren's father's journey east + a map fragment
useful for the Eastern Highlands later.

**Quest: The Lake Caves**
Something has been raiding the fishing nets at night. Breadcrumbs: a
fisherman complains at the docks, torn nets are visible as a room noun,
and the innkeeper mentions strange sounds from the caves. Resolution:
clear the caves of the creatures (combat) OR discover the creatures are
displaced by something deeper and seal the passage (exploration +
problem-solving). Both paths resolve the fishing problem but reveal
different lore.

---

### PHASE 3 — The Long Road to New Plymouth (Expanded Corridor)

> **2026-06-19 redesign.** The original plan hopped Stillwater → New Plymouth
> in two zones (~35 rooms). Canon makes that far too short: the Ashwick
> signpost reads *"Stillwater 4 days, New Plymouth 7 days"* and the novel puts
> Stillwater↔NP at *"four or five days"* (ln 1462), with long empty stretches
> (*"no farms, no smoke, no one on the road"* ln 460) before the capital's
> support belt. So the corridor is now **8 legs / ~145 rooms** of road, open
> reach, a waypoint town, and the capital's farm/industry hinterland — with
> open zones deliberately left sparse and **expandable** for future outdoor
> content. Stillwater is also a **ferry hub** (see "Water Routes & Ferries"):
> players can pay for a boat to skip the overland slog. Visual reference:
> `docs/world_atlas_mock.html`. Priority-table ids: 5, 5.1–5.5, 6.

#### Zone 5 — North Road North
*Day 1 out of the lake country. The road widens; the wilds thin behind you.*
- **Biome:** farmland → scrub. **Size:** 18 rooms (2 mini-stages).
- **Connects:** Stillwater north gate (~4111) ↔ The Empty Reach.
- Heavy wagon traffic, a toll station (pay/argue/bypass), seedy roadside inns,
  a bloodline checkpoint (bluff/credentials/ford-bypass). Mileposts begin the
  count toward the capital.

#### Zone 5.1 — The Empty Reach  *(NEW · open · expandable)*
*Dry open scrub and bare basalt; sage, bunchgrass, and a wind that smells of
dust and cold stone. No farms. No smoke.*
- **Biome:** dry scrub/steppe. **Size:** 12 rooms (sparse by design; reserved
  outdoor-expansion slot). A lone waypoint shrine (novel ln 125). Ambient
  predators/scavengers; the loneliness is the content.

#### Zone 5.2 — Hartcharn  *(NEW · waypoint town)*
*A coaching town: the overnight stop on the long road north.*
- **Biome:** roadside settlement. **Size:** 20 rooms (2 mini-stages).
- Hostels (novel — travelers bound for Bloomings, merchants, drovers), stables,
  a smithy, a market, a ferry-agent's office (Stillwater↔NP packet sells here).

#### Zone 5.3 — Greywater Flats  *(NEW · open · expandable)*
*Open river country — wide water-meadows, a ford or bridge, big sky.*
- **Biome:** river flats/wetland edge. **Size:** 12 rooms (expandable). A river
  crossing; fishing; the first hint of the capital's reach (toll markers, a
  bloodline survey post).

#### Zone 5.4 — Kingsbarrow Vale  *(NEW · the granary belt · expandable)*
*The capital's breadbasket. Farms to the horizon — this is what feeds 100,000
mouths.*
- **Biome:** intensive farmland. **Size:** 25 rooms (3 mini-stages, expandable).
- Farm estates, granaries, watermills, drovers' roads, a tithe-barn, hired
  field hands. Where the abstract "capital" becomes a visible economy.

#### Zone 5.5 — Kilnreach Works  *(NEW · the industry belt · expandable)*
*Smoke and noise: the works that build and supply the city.*
- **Biome:** industrial/extractive. **Size:** 25 rooms (3 mini-stages, expandable).
- Quarries, timber yards, tanneries, foundries, brick-kilns, lime-burners,
  charcoal camps. Resource nodes + crafting-material sources; rougher NPCs,
  guild crews, the bloodline's licensing reach. The city is close now —
  you can smell it (novel ln 1236).

---

#### Zone 6 — New Plymouth Outskirts
*Where the city bleeds into the countryside. Not yet the city, no longer
the road.*

- **Biome:** Urban fringe, mixed commercial/residential
- **Size:** 20 rooms (2 mini-stages)
- **Connects:** North Road (south) ↔ New Plymouth gates (north/east)
- **Theme:** The city's edge. Cheaper housing, unlicensed workshops,
  people who can't afford to live inside the walls but can't afford
  to leave. The ford crossing enters here.

| Stage | Rooms | Content |
|-------|-------|---------|
| 3.2a | 10 | Southern approach: the main road widens to a proper avenue. Guard posts. A market that sprawls outside the walls (5-6 rooms of stalls and hawkers). An unlicensed inn. The smell of the docks carries on the wind. NPCs: a con artist, a legitimate guide offering city tours, a refugee family, guards who can be bribed or reasoned with. |
| 3.2b | 10 | The ford crossing approach (Vane's route): riverside path, the ford itself, the river road emerging west of the east gate. Less patrolled, rougher. A smuggler's contact. The path to the docks district that avoids the main checkpoints. A boatman who ferries people across for a fee. This is how Maren entered the city. |

**Quest: The Forged Papers**
A contact in the outskirts can arrange papers for entering the city without
passing the bloodline's checkpoints. Breadcrumbs: an NPC at the traveler's
waystation mentions it, graffiti on a wall gives a coded address, and a
boatman at the ford hints at "the woman who arranges things." Resolution:
find the contact and pay the fee (simple), OR do a favor for her first
(a delivery that teaches you the outskirts layout), OR find an honest way
in through the main gate (talk your way past the guards with sufficient
rhetoric or a legitimate introduction from an NPC met earlier on the road).

---

### PHASE 4 — New Plymouth (The Big City)

**This is the heart of the expansion.** New Plymouth must feel enormous,
alive, layered, and dangerous. It's the novel's central stage — where all
four protagonists converge, where the cooperage group meets, where Horst
hunts, where the Bloom trade operates, where the Restricted Collection
hides its secrets.

**Target: 170 rooms across 6 districts + the sewers (initial build).**

New Plymouth is designed to grow. The initial 170 rooms establish a
functional, explorable city, but every district includes **expansion
stubs** — visible locations the player can see but not yet enter. These
are not broken exits or placeholder rooms. They are described, locked,
guarded, or otherwise narratively gated so players understand something
is there and will be accessible later. Examples:

- A palace gate with a dozen guards and no passage (Noble Quarter)
- A locked university annex "closed for renovation" (Temple District)
- A boarded-up warehouse with sounds coming from inside (Docks)
- A gated residential lane with a porter who turns you away (Noble Quarter)
- A collapsed tunnel section marked "unstable" (Sewers)
- A shipyard with a chain across the entrance and a harbormaster who says
  "authorized personnel only" (Docks)
- A walled garden visible over a wall with no gate on this side (Merchant)
- A second floor or basement with stairs described but flagged impassable

These stubs serve three purposes: they make the city feel larger than its
room count, they give players something to anticipate, and they give us
clean expansion points that don't require reworking existing content. When
we build Phase 4 expansions later, each stub becomes a real entrance.

**Expansion target: 300+ rooms when all stubs are built out.**

Every district has its own character, its own NPCs, its own ambient life,
and at least one questline. The city should reward exploration — there is
always another alley, another conversation, another hint.

#### Zone 4.1: Docks District
*Where the river meets the sea. Trade, smuggling, Bloom, and Vane's
territory.*

- **Biome:** Urban waterfront
- **Size:** 30 rooms (3 mini-stages)
- **Theme:** Commerce and its shadows. Legal trade on the surface,
  the Bloom supply chain underneath. This is where Vane lives, where
  Pip operates, where Cade's supply house runs its back-room business.

| Stage | Rooms | Content |
|-------|-------|---------|
| 4.1a | 10 | The wharves: dock platforms (3-4 rooms along the waterfront), a harbormaster's office, warehouse row, a fish market. Ambient: dock workers loading cargo, seabirds, the smell of brine and tar and rotting wood. The barge landing where Davan arrived. A hiring board for dock work. |
| 4.1b | 10 | Dockside streets: Vane's neighborhood. A cheap inn (The Salt Cellar, 3 rooms), Pip's usual haunts, a pawnshop, a tavern where rumors flow. Cade's fabric shop (front) with the supply house behind it. NPCs with layered dialogue — ask about trade and get trade answers, ask about Bloom and get silence or deflection depending on trust. |
| 4.1c | 10 | The underdocks: below street level, where the pilings meet the water. Smuggler passages, a hidden dock, the path to the Bloom-giver's location (low stone ceiling, underground). Darker, more dangerous. Mobs here are human — thugs, desperate people, Bloom-addled wanderers. The constabulary rarely comes down here. |

**Expansion Stubs:**
- A chained-off **shipyard** at the north end of the wharves. A harbormaster
  says "authorized personnel only — naval commission business." (Future:
  player-accessible shipyard, sea travel, coastal zones.)
- A **sealed warehouse** on the waterfront with sounds of machinery inside
  and a guard who won't discuss the contents. (Future: Bloom refinery
  or bloodline logistics operation.)
- A passage in the underdocks that leads to a **collapsed tunnel** marked
  with warning signs. (Future: connection to a deeper smuggler network
  or coastal caves.)

**Quest: Dock Rat**
A dockworker has been accused of theft he didn't commit. Breadcrumbs: the
dockworker's friend pleads at the tavern, a posted notice names the accused,
and examining the "stolen" cargo at the warehouse reveals inconsistencies.
Resolution: investigate the real thief (a warehouse foreman skimming —
requires searching his office and confronting him) OR convince the
harbormaster through testimony from witnesses gathered around the docks.
A third path: the pawnshop owner knows the truth but needs a favor first.

**Quest: The Bloom Trail** (multi-zone, starts here)
Hints about the Bloom supply chain accumulate across the docks district.
This is NOT a quest a player is handed — it emerges from paying attention.
Breadcrumbs: Cade's back room can be overheard (specific room, specific
time), a Bloom-addled NPC in the underdocks mutters about "the room with
the low ceiling," and a dock constable mentions investigating "something
in the pilings." No single NPC gives you the full picture. Resolution
is discovery — finding the Bloom-giver's room is the payoff. What the
player does with the knowledge is their choice.

---

#### Zone 4.2: Crafting Quarter
*Where things are made. Forges, workshops, alchemy benches, and the
cooperage.*

- **Biome:** Urban industrial/artisan
- **Size:** 25 rooms (3 mini-stages)
- **Theme:** Honest work and quiet rebellion. The cooperage group meets
  here because craftspeople understand that making things requires
  asking how things work, and asking how things work is a short step
  from asking why things are the way they are.

| Stage | Rooms | Content |
|-------|-------|---------|
| 4.2a | 10 | Main crafting street: a blacksmith (shop + forge), an alchemist, a leatherworker, a tailor. Crafting stations for player use. NPCs discuss their trades, share recipes, and complain about material costs. Ambient: hammer sounds, chemical smells, the creak of looms. A notice board with crafting commissions. |
| 4.2b | 10 | Side streets and specialists: Edvar's map shop (2 rooms — front shop, back workroom where the face is hidden), Asha's glass workshop, a jeweler, a woodcarver. The cooperage building (front shop where you buy hoop-sets, the yard, the basement meeting room). These NPCs have deeper dialogue trees — Edvar tests you before revealing anything, Asha assesses your hands before your words. |
| 4.2c | 5 | Back alleys and storage: warehouse spaces, a hidden passage connecting the cooperage to a secondary exit (the gate Vane uses), a storage loft where Renner keeps her archive copies. Less polished, more private. The infrastructure of a district that has learned to keep some things out of sight. |

**Expansion Stubs:**
- A **guild hall** with its doors barred and a posted notice: "Closed by
  order of the Crafting Council pending restructuring." (Future: crafting
  guild questline, advanced recipe access, political faction content.)
- A **kiln complex** behind Asha's workshop, fenced off and overgrown.
  Asha mentions she'd "need help clearing it out before it's usable
  again." (Future: advanced glassblowing station, crafting expansion.)
- A **locked cellar door** beneath the cooperage that even the group
  doesn't use. Renner says it "goes deeper than we've explored."
  (Future: pre-Founding tunnels beneath the Crafting Quarter.)

**Quest: The Apprentice's Commission**
A young apprentice needs help completing a masterwork piece to earn their
journeyman rank. Breadcrumbs: the apprentice asks for help at the forge,
their master mentions the deadline in passing conversation, and a notice
on the crafting board lists the required materials. Resolution: help
gather materials (foraging/purchasing across districts) OR teach the
apprentice a technique you know (if your crafting skill is high enough)
OR find a shortcut material at the pawnshop in the docks district. Reward:
the apprentice becomes a reliable crafter who offers player discounts.

---

#### Zone 4.3: Merchant Quarter
*Money and influence. Where Horst set up shop.*

- **Biome:** Urban commercial, prosperous
- **Size:** 25 rooms (3 mini-stages)
- **Theme:** Wealth on display and wealth hidden. The legitimate face
  of power. Horst operates from a rented house here because this is
  where you go when you want to look like you belong.

| Stage | Rooms | Content |
|-------|-------|---------|
| 4.3a | 10 | The main market: a large open market square (3-4 rooms), permanent shops (a weapon dealer, an armor shop, a general goods emporium, a moneylender). The auction house. Ambient: shouting merchants, haggling, the clink of coin. Prices here are higher than the docks. The quality is better. |
| 4.3b | 10 | Merchant streets: trading houses, import/export offices, a high-end inn (The Gilt Threshold, 3 rooms). Horst's rented house is on one of these streets — unremarkable, deliberately so. NPCs: merchants with trade gossip, a tax collector, a bloodline clerk processing permits. Information is currency here. |
| 4.3c | 5 | The Exchange and banks: the money quarter. A bank (2 rooms), a currency exchange, a notary. Legal services. The kind of infrastructure that makes a city function. An NPC lawyer who knows things about property records (connects to the Bloom Trail — Vane tracked Noble Quarter addresses through property arrangements). |

**Expansion Stubs:**
- A **walled garden** visible over a high stone wall along Merchant Row.
  No gate on this side — a gardener working inside can be spoken to
  through the wall, mentions it belongs to "the consortium." (Future:
  merchant faction HQ, trade guild politics, economic questlines.)
- The **upper floors** of the Gilt Threshold inn. A staircase with a
  velvet rope and a concierge who says "private suites — by arrangement
  only." (Future: high-end NPC encounters, espionage content, Horst's
  actual operations.)
- A **sealed counting house** in the Exchange. A clerk explains it handles
  "inter-city transfers only" and won't process anything for the player.
  (Future: banking system, cross-city trade, Tidemark connection.)

**Quest: Market Manipulation**
A merchant is being squeezed out by a competitor using unfair practices.
Breadcrumbs: the merchant asks for help, their empty stall is noticeable
in the market, and a customer at the inn complains about the competitor's
suddenly low prices. Resolution: investigate the competitor's supply
(they're getting goods from a smuggling operation — connects to docks) OR
help the merchant find a new supplier (travel to outskirts or use road
contacts) OR expose the scheme to the market authority through evidence
gathered from the Exchange's records.

---

#### Zone 4.4: Temple District
*Faith and its architecture. Where Aldric would find lodging.*

- **Biome:** Urban religious/institutional
- **Size:** 25 rooms (3 mini-stages)
- **Theme:** Institutional power dressed in spiritual language. The
  Grand Temple of the Chrysalis is beautiful and sincere and built on
  a story that is wrong. The smaller chapels and pilgrim hostels serve
  real needs. The tension between genuine faith and managed truth lives
  in the stonework.

| Stage | Rooms | Content |
|-------|-------|---------|
| 4.4a | 10 | The Grand Temple: entrance hall, nave (large, high-ceilinged, the Recitation of First Light inscribed on the walls), a meditation garden, a pilgrim hostel (3 rooms — cheap lodging). Temple functionaries, a canon who gives blessings, a scribe maintaining records. The architecture includes the inner orbit symbol worked into older stonework — not hidden, just unexamined. |
| 4.4b | 10 | Temple surrounds: smaller chapels, a seminary building, a religious bookshop, a healer's chapel (temple healing services), the archive entrance (restricted — the Restricted Collection is referenced but not accessible). A courtyard where scholars debate theology. NPCs: pilgrims, a skeptical scholar, a temple guard, a novice with doubts. |
| 4.4c | 5 | The quieter reaches: an old cemetery, a meditation cloister, the oldest section of the temple complex where the stonework predates the current building. The symbol is here too, in its original form — not the Chrysalis spiral, the orbital geometry. A lore-rich area for observant players. |

**Expansion Stubs:**
- The **Restricted Collection archive entrance**: a heavy door with two
  temple guards and a bloodline seal. A posted sign: "Access by written
  authorization of the High Keeper and the Bloodline Historical
  Commission." (Future: the archive itself — deep lore content, the
  novel's central institutional secret.)
- A **seminary annex** with scaffolding and a sign: "Closed for
  restoration — completion date pending." A novice mentions it's been
  "pending" for three years. (Future: seminary questline, deeper temple
  politics, Crane's investigation.)
- A **locked crypt** beneath the old cemetery. A groundskeeper says the
  key was "lost two Keepers ago." (Future: pre-Founding burial site,
  sealed containers, connections to the Confluence undercroft.)

**Quest: The Doubting Novice**
A temple novice is struggling with questions about the official account of
the Founding. Breadcrumbs: the novice approaches you in the courtyard, a
book in the religious shop has a margin note questioning the standard
theology, and a scholar in the courtyard debates the age of certain temple
inscriptions. Resolution: help the novice research their question (visit
the archive, talk to scholars) OR encourage them to speak with their
superior (which may go well or poorly depending on player choices) OR
find them a contact outside the temple who studies these things (connects
to the cooperage group). Reward: an ally in the temple district who
provides access to minor archive materials.

---

#### Zone 4.5: Noble Quarter
*Where the bloodline lives. Where people are delivered to Noble Quarter
addresses and not seen again.*

- **Biome:** Urban elite, controlled
- **Size:** 20 rooms (2 mini-stages)
- **Theme:** Power without apology. Beautiful streets, manicured grounds,
  and the quiet menace of a district where everyone is watched and
  the watching is considered a service. The addresses where Vane
  delivered contracts are here.

| Stage | Rooms | Content |
|-------|-------|---------|
| 4.5a | 10 | The public face: wide boulevards, the Palace approach (exterior only — the Palace is visible but not enterable except for endgame content), the Bloodline Administrative Office, a high-end clothier, a gallery of Founding-era art (contains subtle clues in the oldest pieces). Guards are more numerous and more attentive here. NPCs: bloodline functionaries, a nervous servant, a tour guide who recites approved history. |
| 4.5b | 10 | The residential streets: the Noble Quarter addresses. Elegant townhouses, private gardens visible over walls, the particular quiet of a district where noise is considered vulgar. One of these houses is the address from Vane's contract. A player exploring carefully might notice that several houses share the same property arrangement (connects to the Exchange records). Fewer NPCs, more guards. The sense of being watched. |

**Expansion Stubs:**
- The **Royal Palace gates**: a massive iron-and-stone gatehouse with
  a dozen guards in ceremonial armor. The palace grounds are visible
  through the bars — manicured gardens, stone walkways, the palace
  itself gleaming in the distance. No amount of persuasion moves the
  guards. "The Palace is not open to visitors." (Future: major endgame
  content — the bloodline's seat of power, the Wound-maker vault,
  political confrontation.)
- A **gated residential lane** leading to the most exclusive addresses.
  A liveried porter stands at the gate and politely but firmly redirects
  all traffic. "This lane is for residents and their invited guests."
  (Future: bloodline family estates, deep espionage content.)
- A **private chapel** attached to the Administrative Office. Locked,
  with light visible through stained-glass windows. A functionary says
  it's "for bloodline observances only." (Future: the bloodline's
  private theology, evidence of what they actually know vs. what they
  teach.)

**Quest: The Gallery Cipher**
The Founding-era art gallery contains pieces that predate the current
theology. Breadcrumbs: an artist in the Crafting Quarter mentions the
gallery's oldest pieces "don't match the story," a scholar in the Temple
District references pre-Chrysalis art, and examining the oldest painting
in the gallery reveals a symbol that doesn't match the Chrysalis iconography.
Resolution: research the symbol through dialogue (cooperage contacts, temple
scholars, Edvar) OR find a reference in the archive materials OR compare it
to the disc's markings. Reward: lore + a key piece of evidence for the
larger mystery.

---

#### Zone 4.6: Common Quarter
*Where most people actually live. Crowded, loud, real.*

- **Biome:** Urban residential, mixed income
- **Size:** 25 rooms (3 mini-stages)
- **Theme:** The city as experienced by the people who keep it running.
  Tenement houses, street markets, neighborhood taverns, the particular
  vitality of a district where people have to be resourceful. This is
  where Maren and Vane's rented room would be.

| Stage | Rooms | Content |
|-------|-------|---------|
| 4.6a | 10 | Main residential streets: tenement rows, a neighborhood market (smaller and cheaper than the Merchant Quarter), a laundry, a barber, a street-food vendor. Ambient: children playing, laundry hanging between buildings, the sound of too many people in too little space. NPCs: a landlady, a street sweeper who knows everything, a retired soldier, neighborhood kids who run errands for coin. |
| 4.6b | 10 | The east side: closer to the docks, rougher. A fighting pit (underground entertainment), a flophouse, a back-alley healer who doesn't ask questions. The room Maren and Vane rented — a nondescript building in a nondescript street, chosen for its sight lines and exit options. Mobs: pickpockets, drunks, the occasional desperate person. |
| 4.6c | 5 | The river road: the eastern edge of the Common Quarter, where the district meets the river. A small park. The path that leads east toward the ford crossing and eventually the eastern road. The moons are visible over the eastern roofline from here. A quiet bench where someone might sit and count three lights in the sky. |

**Expansion Stubs:**
- A **fighting pit** with a locked iron gate and a bouncer who says
  "fights are by invitation. Come back when someone vouches for you."
  Cheering is audible from inside. (Future: underground fighting ring,
  combat reputation system, gambling, faction contacts.)
- A **tenement block** with a barricaded stairwell. A landlady says the
  upper floors are "condemned — structural damage." Sounds of habitation
  come from above anyway. (Future: squatter community, hidden NPC
  network, alternate quest paths through the city's underclass.)
- A **river gate** in the city wall, chained shut. A plaque reads
  "Flood gate — authorized access only." The river is visible through
  the bars. (Future: river travel within the city, connection to
  upstream zones, waterborne trade routes.)

**Quest: The Street Sweeper's Secret**
The street sweeper has been finding odd objects in the gutters — fragments
of old material, things that don't belong. Breadcrumbs: the sweeper
mentions it casually, a child found one and is playing with it in the
street, and the back-alley healer has one displayed as a curiosity.
Resolution: collect the fragments and bring them to someone who can
identify them (Edvar, Renner, or the temple scholar) OR follow the trail
to their source (a collapsed section of old foundation beneath the
Common Quarter that predates the current city). Reward: access to a
small underground area with pre-Founding materials + lore.

---

#### Zone 4.7: New Plymouth Sewers
*Beneath the city. Connecting districts, hiding secrets.*

- **Biome:** Underground, wet, dark
- **Size:** 20 rooms (2 mini-stages)
- **Theme:** What's underneath. Every city has a second city below it.
  The sewers connect districts, allow covert movement, and contain
  things the surface has forgotten about.

| Stage | Rooms | Content |
|-------|-------|---------|
| 4.7a | 10 | Main sewer tunnels: connecting passages beneath the major districts. Entrances from the docks, the Common Quarter, and the Crafting Quarter (the cooperage has a concealed access). Mobs: rats, slimes, feral mutated animals. The infrastructure is old — older than the current city in places, with stonework that doesn't match the surface architecture. |
| 4.7b | 10 | The deep section: older tunnels, pre-Founding construction. A sealed chamber (similar to the one at the Confluence). Fragments of the gray material in the walls. A hidden room used by the cooperage group as a secondary meeting place. The path that connects to the Bloom-giver's underground room. This area rewards high-level exploration and connects multiple questlines. |

**Expansion Stubs:**
- A **collapsed tunnel** at the deepest point, partially cleared rubble
  revealing worked stone of a different character than the sewers. A
  faint current of dry air comes from beyond. (Future: pre-Founding
  underground complex, possibly part of a secondary debris field from
  the crash.)
- A **flooded passage** requiring a boat or swimming ability to cross.
  The water is dark and deep and something large occasionally surfaces.
  (Future: underwater section, aquatic mobs, a submerged pre-Founding
  structure.)
- A **grated drain** beneath the Noble Quarter that is welded shut from
  the other side. Voices and footsteps are occasionally audible above.
  (Future: infiltration route into the Noble Quarter, espionage content.)

---

### PHASE 5 — The Southern Road (Amber Valley & The Confluence)

**Purpose:** The novel's secondary geography. Amber Valley is Davan's home,
the Confluence is Aldric's temple. These zones flesh out the south and
provide alternative leveling paths.

#### Zone 5.1: South Road
*The road south from the Ashwick crossroads toward Amber Valley.*

- **Biome:** Transitioning farmland to dry valley
- **Size:** 15 rooms (2 mini-stages)
- **Connects:** Ashwick Crossroads (north) ↔ Amber Valley (south)
- **Theme:** The land warming and drying. Orchards giving way to drier
  scrub. The Yakima Valley approaching.

| Stage | Rooms | Content |
|-------|-------|---------|
| 5.1a | 10 | The descent from the crossroads. Road winding through increasingly dry terrain. A waypoint inn at the midpoint. Traveling merchants heading north. A shepherd NPC with local knowledge. Views of the valley opening below. |
| 5.1b | 5 | Valley approach. Orchards and irrigated farms. The first signs of Amber Valley — the warm air, the particular smell of sun-baked earth and ripening fruit. A farmstead with a water dispute (quest hook). |

---

#### Zone 5.2: Amber Valley
*Davan's home. A warm farming community in the rain shadow of the mountains.*

- **Biome:** Dry valley, irrigated farmland, warm
- **Size:** 35 rooms (4 mini-stages)
- **Connects:** South Road (north) ↔ River Road to Confluence (south)
- **Theme:** A place of growth — both agricultural and personal. This
  is where the Chrysalis Rite happens, where young people discover
  what they're becoming. Davan left because his discovery pointed him
  elsewhere.

| Stage | Rooms | Content |
|-------|-------|---------|
| 5.2a | 10 | Town center: the market square, the Rite pavilion (where Blooming ceremonies happen), a general store, an inn (The Golden Bough), a woodworker's shop (Davan's father's trade). Warm, dry air. Fruit stalls. NPCs who talk about their changes with casual pride. |
| 5.2b | 10 | Residential and farms: Davan's family home (his father still works there), irrigated orchards, a vineyard, the river that feeds the valley's irrigation. NPCs: Davan's father (dialogue about his son leaving, the carving talent, worry), neighboring farmers, a traveling Rite deacon. |
| 5.2c | 10 | The valley edges: foothills, dry scrub, the old paths up toward the ridge. A cave system (minor dungeon, 4-5 rooms). The road south toward the Confluence begins here, following the river. Wildlife appropriate to the dry valley — lizards, hawks, sun-adapted mutated fauna. |
| 5.2d | 5 | The Chrysalis grove: a sacred site outside town where the most dramatic Bloomings are commemorated. Old stone markers for particularly notable mutations. A quiet, reverential place with deep lore about how the community understands the Chrysalis. A hidden marker that predates the theology — the inner orbit symbol, weathered almost flat. |

**Quest: The Water Dispute**
Two neighboring farms are fighting over irrigation rights. Breadcrumbs: both
farmers complain at the market, the innkeeper mentions the tension, and the
dried-up irrigation channel is visible in the farmland rooms. Resolution:
mediate between the farmers (dialogue with both, finding a compromise) OR
fix the upstream water source (a collapsed section of the old irrigation
system needs clearing — exploration/combat) OR find the original water
agreement in the town records (research at the town hall). Each resolution
favors a different farmer — consequences persist.

**Quest: The Rite Deacon's Concern**
The traveling deacon has noticed something unusual about this year's
Bloomings — the mutations are more dramatic than expected. Breadcrumbs: the
deacon mentions it to the player, a young person in town is visibly
struggling with an accelerating change, and the Chrysalis grove's markers
show a pattern (recent markers are more extreme). Resolution: investigate
the grove's hidden marker (the old symbol — connects to the larger mystery)
OR help the struggling youth manage their change (herbalism + dialogue) OR
report to the temple (which may or may not be what the deacon actually
wanted). The deacon has their own theory they'll share if trust is earned.

---

#### Zone 5.3: River Road to the Confluence  *(✅ Built 2026-06-26 — rooms 6090–6105, folder river_road)*
*Following the river south and east toward the meeting of three waters.*

- **Biome:** River valley, increasingly lush
- **Size:** 15 rooms (2 mini-stages)
- **Connects:** Amber Valley (north) ↔ The Confluence (south)
- **Theme:** Water shaping land. The river widening as tributaries join.
  The sense of convergence — rivers coming together, as the story's
  characters will.

| Stage | Rooms | Content |
|-------|-------|---------|
| 5.3a | 10 | River road rooms. The water gets bigger. A barge dock where river traffic is visible. A fishing village (3 rooms). The second river joining — the meeting visible from a bluff. Ambient: river birds, the sound of water over stones, the smell of wet earth. |
| 5.3b | 5 | Confluence approach. The three rivers visible ahead. The city's spires/towers emerging. Pilgrim traffic increasing — the Temple of Confluence is a destination. A pilgrim camp outside the city. |

---

#### Zone 5.4: The Confluence  *(🔧 Building — SCALED UP to ~150rm/10 districts; see `docs/superpowers/specs/2026-06-26-confluence-citywide-design.md`. District 1 "The Landings" ✅ built, rooms 6106–6121.)*
*A major temple city where three rivers meet. Aldric's home.*

- **Biome:** River city, temperate, institutional
- **Size:** 70 rooms (7 mini-stages)
- **Connects:** River Road (north) ↔ East Road to Greenford (east) ↔
  River barge to New Plymouth (north, via river)
- **Theme:** Where faith meets scholarship meets water. A city built
  around a temple built on top of something it doesn't understand.
  The three rivers meeting is both geographic fact and theological
  metaphor, and the city has made the most of both.

| Stage | Rooms | Content |
|-------|-------|---------|
| 5.4a | 10 | River district: the barge docks (where Davan departed for NP), warehouses, a riverside market, fish traders. The three rivers are visible from the docks — the meeting of waters is impressive, the currents complex. A barge master who sells passage north. Ambient: water sounds, trading shouts, the creak of moored vessels. |
| 5.4b | 10 | City center: the main square, a large inn (The Three Waters), shops, a municipal building. The city is prosperous — river trade keeps it fed. NPCs discuss theology, trade, and the latest pilgrimage numbers. The architecture shows the inner orbit symbol worked into older buildings, reinterpreted as Chrysalis motifs in newer ones. |
| 5.4c | 10 | Temple of Confluence — exterior and public areas: the grand entrance (the symbol above the door that Davan noticed), the public nave, a meditation garden, a pilgrim reception hall. Temple functionaries, a guide, a historian NPC. The eastern wing's architecture is visibly older than the rest. |
| 5.4d | 10 | Temple of Confluence — interior: Aldric's quarters (or his replacement's, if timeline permits), the archive, the eastern corridor with the stairs, the records room. Brother Cael as an NPC. Prioress Crane may or may not still be present. The undercroft entrance is here — sealed or accessible depending on quest progress. |
| 5.4e | 10 | The undercroft and sealed chamber: the stairs down, the older construction, the sealed room with the remaining objects (the face has been found by Aldric — what remains are containers). This is deep lore content, accessible only through significant quest progress. The face's orbital display is the payoff. |
| 5.4f | 10 | Residential quarters: the city's living areas. A scholars' district near the temple. A craftsmen's row. A small market for daily needs. NPCs with lives — a baker, a riverman's wife, a retired temple functionary who hints at things the official history doesn't cover. |
| 5.4g | 10 | East gate and surrounds: the road east toward Greenford begins here. A travelers' inn. A stable. The east gate guards. The road out of the city follows the river before diverging. The landscape begins to dry — the transition toward the eastern interior. |

**Quest: The Margin Notation** (multi-stage, lore-heavy)
A scholar in the Confluence has been studying old maps with unusual margin
notations. Breadcrumbs: the scholar mentions it at the inn, a map in the
municipal building has a visible notation, and a bookseller has a damaged
map with the symbol for sale. Resolution unfolds over multiple stages:
find maps with the notation (multiple sources), compare the numbers
(three vs four), and eventually connect this to the temple's oldest
architecture. This quest is the player's path into the cooperage group's
question — what changed in the sky?

**Quest: The Undercroft**
Access to the sealed chamber beneath the temple requires earning trust
from temple NPCs and following a chain of clues about the temple's
construction history. This is endgame-tier lore content — the full
payoff for players who have been following the mystery across multiple
zones.

---

### PHASE 6 — Greenford

#### Zone 6.1: East Road to Greenford
*The road east from the Confluence into drier country.*

- **Biome:** Transitioning river valley to dry plateau
- **Size:** 15 rooms (2 mini-stages)
- **Connects:** The Confluence (west) ↔ Greenford (east)

| Stage | Rooms | Content |
|-------|-------|---------|
| 6.1a | 10 | Road rooms through drying terrain. Wheat fields. The river falling behind. A waypoint village. Views of the eastern plateau. |
| 6.1b | 5 | Greenford approach. The university's profile visible. The river Greenford sits above. A bridge crossing. |

---

#### Zone 6.2: Greenford
*A university town on a river. Brennan's home. Where old questions are
studied quietly.*

- **Biome:** River town, temperate, scholarly
- **Size:** 45 rooms (5 mini-stages)
- **Connects:** East Road (west) ↔ West Road toward New Plymouth (west/NW)
- **Theme:** Knowledge and its caretakers. A smaller city than the
  Confluence or New Plymouth, but richer in questions. The university's
  archive is a backwater that has escaped the attention of the people who
  manage what gets remembered.

| Stage | Rooms | Content |
|-------|-------|---------|
| 6.2a | 10 | Town center: the market square (smaller, quieter than Confluence), a bookshop, an inn (The Cartographer's Rest), a general store. The university's buildings visible up the hill. Ambient: scholars walking with books, the quiet industry of a town that thinks for a living. |
| 6.2b | 10 | University district: lecture halls (exteriors + one accessible), the library (3 rooms — public stacks, reading room, restricted section), Brennan's office. Faculty NPCs who discuss pre-Chrysalis history, old languages, material culture. The archive's geological records (where Reth's survey is filed). |
| 6.2c | 10 | Brennan's neighborhood: residential streets, his narrow house with the blue door (interior rooms — cluttered, maps on walls). Reth's cottage in the north end (sparse, orderly). Smaller shops, a tea house, a garden. The specific quiet of a town that values discretion. |
| 6.2d | 10 | River district: the river below the town, a small dock, a bridge, a watermill. Fishing. A path along the river that connects to the road west toward New Plymouth. The landscape here is greener than expected — the river sustains it. |
| 6.2e | 5 | Outskirts: the road west. A stable. The west road out of town toward New Plymouth (this is the route Aldric takes). Views back toward the town and forward toward the lowlands. A farewell waypoint shrine. |

**Quest: The Surveyor's Report**
Brennan knows Reth has information but can't get it himself. Breadcrumbs:
Brennan mentions a "difficult source" if the player earns his trust, the
university's geological records contain Reth's anomalous survey entry (the
"mineral deposit" language that says nothing), and a bookseller mentions
"the surveyor who retired early." Resolution: approach Reth directly
(requires a specific reputation or introduction) OR find Reth's original
field notes in the archive (research quest) OR earn Brennan's trust first
and get his introduction (dialogue chain). Reth's testimony about the hill
is the payoff — verbal directions to the crash site.

---

### PHASE 7 — The Eastern Road (Endgame Approach)

#### Zone 7.1: Cascade Pass Road
*East from New Plymouth into the mountains. The road to the hill.*

- **Biome:** Forest → mountain pass → highland plateau
- **Size:** 20 rooms (2 mini-stages)
- **Connects:** New Plymouth East Gate (west) ↔ Eastern Highlands (east)
- **Theme:** The world getting wilder and older. The road narrows. The
  trees thicken. The air changes. You are leaving civilization behind.

| Stage | Rooms | Content |
|-------|-------|---------|
| 7.1a | 10 | The forest road: dense timber, the road climbing. A lumber camp. A ruined waypoint (abandoned years ago). The sense of entering country that doesn't want visitors. Mobs: forest predators, territorial wildlife. The trees are larger and older the further east you go. |
| 7.1b | 10 | The pass itself: high altitude, exposed rock, views in both directions. A crumbling watchtower. The tree line. The terrain breaking into the highland plateau beyond. The air is different — thinner, colder, something underneath it that Davan's empathic sense would read as "old." |

---

#### Zone 7.2: Eastern Highlands
*The broken country east of the tree line. Reth's survey territory.*

- **Biome:** Highland plateau, erosion gullies, exposed rock, heavy scrub
- **Size:** 30 rooms (3 mini-stages)
- **Connects:** Cascade Pass (west) ↔ The Hill (east)
- **Theme:** Desolation with a secret underneath. The terrain is rough,
  deceptive, inhospitable. But there is something here that is not
  geological, and the landscape knows it.

| Stage | Rooms | Content |
|-------|-------|---------|
| 7.2a | 10 | The approach: highland scrub, erosion gullies, basalt formations. The previous survey markers (old, weathered). A camp site. The landscape is genuinely difficult — high-level mobs, environmental hazards, limited shelter. Reth's "lightning-split cairn" is a landmark room. |
| 7.2b | 10 | The formation: the first sign of the hull. A line where vegetation stops — too clean, too precise. The exposed surface: smooth, metallic, no grain, no seam. Rooms that describe walking along a buried object four hundred meters long. The light catches it at certain angles. The descriptions should feel wrong — this is not landscape. |
| 7.2c | 10 | The approach to the entrance: the southeastern curve where the hull goes into the ground. The cleared section (someone was here before — Maren's father). The recesses in the hull surface. The disc-shaped depression with the symbol. This is the endgame threshold — the door that requires Maren's disc (or its equivalent) to open. |

---

#### Zone 7.3: The Crash Site (Interior)
*Inside the fourth ship. The truth.*

- **Biome:** Alien interior, preserved technology
- **Size:** 20 rooms (2 mini-stages)
- **Connects:** Eastern Highlands (exterior)
- **Theme:** Ten thousand years of waiting. The ship is not dead — it's
  dormant. The records are intact. The truth is here: a native organism,
  a crash landing, an immune minority who built a religion on a
  misunderstanding that served their interests.

| Stage | Rooms | Content |
|-------|-------|---------|
| 7.3a | 10 | The breached section: the entry point, corridors of the gray material, sealed compartments, emergency lighting that still functions (dim, cold, blue-white). The air is different inside — the Chrysalis cannot reach here, the hull resists it. Storage rooms with sealed containers. A navigation alcove with a face (the orbital display — four shapes, one damaged). |
| 7.3b | 10 | The deep interior: the ship's log access point, the command section, the records archive. This is where the truth lives. Examining the records reveals the novel's central revelation: the Chrysalis is native, the colonists were infected, the bloodline's immunity was genetic chance, not divine selection. The signal array — activated by the disc, it sends the signal the three orbiting ships have been waiting for. |

---

## Build Priority Summary

**Status legend:** ✅ Built · 🟡 Sketched (Phase 1 plan exists, not built) · 🔧 Building · ⬜ Not started

Update the Status column whenever a zone advances. After each
mini-stage build pass, also note the roomid range used in the Notes
column so the next zone-builder knows what's free.

| Priority | Zone | Rooms | Mini-stages | Status | Notes |
|----------|------|-------|-------------|--------|-------|
| 1 | Marches Spur Road | 15 | 2 | ✅ Built | rooms 4000–4014 |
| 2 | Ashwick | 20 | 2 | ✅ Built | rooms 4015–4034 |
| 3 | North Road — Southern | 20 | 2 | ✅ Built | rooms 4038–4062 (25 used incl. inn interior) |
| 4 | Stillwater | 47 | 3 | ✅ Built | **Live & populated as of 2026-06-19** (25 spawn rooms, 6 shop files, connected via north_road 4062→4100). roomid range 4100–4146; 7-station crafting hub (forge, alchemy_bench ×2, loom, cooking_fire ×2, jeweler_bench, enchanting_circle); designed as showcase for NPC AI features (daily routines, forager-driven shop restock, Stillwater↔Thornwall caravan, Stillwater↔Ironwind material trade); **2026-04-25: ALL stillwater rooms shifted west by 7 to resolve 4 coord collisions with Dustwalk Road and Labyrinth (mapper now renders correctly)**; Temple of Stillwater (4123) needs sethome already wired (sethome stillwater); NPCs and mob spawns not yet placed |
| 3.5 | The Fernway (interlude zone) | 10 | 1 | ✅ Built | roomid range 4147–4156; inserted east-west between Ashwick Crossroads (4014) and North Road Road Fork (4038) to push Stillwater + north_road westward away from Dustwalk Road / Labyrinth coord overlap. Outdoorsy bracken-and-fern wilderness with foragable plants (wild thyme, watercress, wood sorrel, foxglove, wild garlic, marsh chamomile, alder cones, elderberry, marsh willow). 7-room east-west spine + 3 side rooms (Old Weddell Farmstead, Heron Pond, Fox Den). No mobs/quests yet — pure flavor + foraging connector. |
| 5 | North Road — Northern | 18 | 2 | 🔧 rooms done | Day 1 out of Stillwater; attaches at Stillwater north gate 4111. **All 18 rooms built 2026-06-19: 5372–5389** — 5a (verge→tollgate+mire-cutoff→Lake & Ladle inn→hedgerow/drovers) + 5b (farmland→scrub transition→lonely frontier at Edge of the Empty Reach, 5389, gated stub to zone 5.1). Boots clean, ValidateZoneConsistency errors=0/warnings=0 (mode=panic). NEXT: mobs/items/spawns pass, then quests. |
| 5.1 | The Empty Reach | 12 | 1+ | ✅ Built | NEW. rooms 5390–5401 (road's-end, wayside shrine, dry wash, coyote ground, abandoned camp, badlands, lookout, alkali pan, distant smoke). Feel-test PASS ("solitude works"). Attaches NRN 5389↔5390, frontier 5401→Hartcharn. |
| 5.2 | Hartcharn | 20 | 2 | ✅ Built | NEW. Full coaching town 5402–5421 (2D grid). Core: gates/square/Coachman's Rest inn/posting yard/**Ferry & Coach Agent**/smithy/store. Residential: North Market St, Temple Lane, Weavers' Row, Tap & Trough tavern, Back Lane, Cottage Row, Town Common, Well Yard, Wagon Camp, North Gate. Attaches Empty Reach 5401↔5402; frontier 5419→Greywater Flats. Boots clean, consistency=0. Feel-test pending. |
| 5.3 | Greywater Flats | 12 | 1+ | ✅ Built | NEW. rooms 5422–5433 (fen road, greywater meadows, causeway, Tollford Crossing ford + **the capital's first survey/jurisdiction post**, heron marsh, survey post + drainage works, drying flats → Kingsbarrow March frontier). The capital's administrative reach first appears here. Attaches Hartcharn 5419↔5422; frontier 5433→Kingsbarrow Vale. Boots clean, consistency=0. Feel-test pending. |
| 5.4 | Kingsbarrow Vale | 25 | 3 | ✅ Built (core 16) | NEW. Granary belt 5434–5449 (expandable core): vale road, Barleywick estate + yard, the tithe road + Great Tithe Barn, Kingsbarrow village (green, granary, church, Mill Street, watermill, market cross, Sheaf & Sickle inn), upper fields → Vale's End frontier. Capital's tithe-reach felt throughout. Attaches Greywater 5433↔5434; frontier 5443→Kilnreach. Boots clean, consistency=0. Feel-test pending. |
| 5.5 | Kilnreach Works | 25 | 3 | ✅ Built (core 16) | NEW. Industry belt 5450–5465 (expandable core): works road, quarry road + the Great Quarry, Kiln Row + lime kilns, Kilnreach settlement, the Foundry, the Tannery, Foundry Lane, the Brickfields + brickworks, the Timber Yard, Slag Lane (workers' housing), the Furnace & Flagon tavern, north works road → Works' End (New Plymouth's smoke now in view). Attaches Kingsbarrow 5443↔5450; frontier 5457→NP Outskirts. Boots clean, consistency=0. Feel-test pending. |
| 6 | NP Outskirts | 25 | 2 | ✅ Built (core 16) | NEW. Urban fringe 5466–5481 (folder new_plymouth_outskirts): city road (walls in view), the unlicensed Outer Market + stall rows + hawkers' yard + Broken Gate doss-inn, the Shambles + refugee camp + workshop warren, Gate Approach + doss-house, the **East Gate** (checkpoint into NP proper — stub, city not yet built), and Vane's ford/river-road bypass (Riverside Path → the Ford → River Road, stub toward the docks). Attaches Kilnreach 5457↔5466. Boots clean, consistency=0. **COMPLETES the Stillwater→NP corridor.** Feel-test pending. |
| 7 | NP Docks District | 30 | 3 | ✅ Built | **LIVE ON PROD** (the whole capital pushed `ec2384ce` 2026-06-25). rooms 5500–5529; `np_dockfolk` faction; **Quest 63 Dock Rat** + the **Bloom Trail (Q66/Q67)**; Marn's Bloom front, the Dock Constable's Post, the Underdocks→Old Quarter seam. Own spec/plan, not this table. |
| 8 | NP Crafting Quarter | 25 | 3 | ✅ Built | rooms 5700–5724; `cooperage_circle` faction; **Quest 68 The Cooperage Circle** (branching); Long Market, footbridge entry from Common, the supply runner (Dobb). |
| 9 | NP Merchant Quarter | 25 | 3 | ✅ Built | rooms 5800–5824; `bloodline_domestic` faction; **Quest 71 The Tribute**; the Central Square, the Gilt Threshold, financial/arms rows. |
| 10 | NP Temple District | 25 | 3 | ✅ Built | rooms 5900–5924; `temple_np` faction; **Quest 69 The Gallery Cipher** (Temple→Noble); the Grand Temple + Archive + an opt-in respawn anchor (5901). |
| 11 | NP Noble Quarter | 20 | 2 | ✅ Built | rooms 6000–6019; the gallery-cipher payoff (Q69), the bloodline apparatus, the Palace gate stub. |
| 12 | NP Common Quarter | 25 | 3 | ✅ Built | rooms 5600–5624; `np_commonfolk`; **Quest 65 The Street Sweeper's Secret**; Carter's Rise, the market, tenements/tannery/undercroft. |
| 13 | NP Old Quarter | 20 | 2 | ✅ Built | rooms 6020–6039 (folder `new_plymouth_old_quarter`; z−1/z−2 buried canal city). The **Bloom-Trail climax** (215 Lintel St) + pre-Founding lore (Gritta's cellar, the Buried Lintel); **Quest 70 The Pre-Founding Web**. Seam: Docks Underdocks 5520→west→6020. Completes the 7-district capital. |
| 13.5 | NP Sewers | 20 | 2 | ⬜ Not started | The chunk 4.7 underground network beneath the districts (rats/slimes/feral fauna; a pre-Founding deep section; a sealed chamber echoing the Confluence undercroft; the Bloom-giver's underground room). Genuinely unbuilt. |
| 14 | South Road | 15 | 2 | ✅ Built | NEW 2026-06-25. rooms 6040–6054 (straight N–S spine); attaches at Marches Spur Road crossroads **4014** (new south exit). Inn (Lake & Ladle), shepherd, merchant; the dried-channel breadcrumb at the Dryside Farmstead. Boots clean, ValidateZoneConsistency errors=0 mode=panic. |
| 15 | Amber Valley | 35 | 4 | ✅ Built | NEW 2026-06-25. rooms 6055–6089 (folder amber_valley): town center + residential/farms + valley-edges & a 5-room cave dungeon (z-descent; sun/cave fauna 9407–9409) + the Chrysalis grove (seeded near-flat orbital marker). 13 NPCs 9394–9406 (Hesper/Golden Bough giver, Davan's father Corwin, the feuding farmers Fenn + Ayres, Rite deacon Pember [seeded], Blooming youth), each a unique mutation. **Quest 72 The Water Dispute** (3 paths: mediate / restore-source / record; outcome flag) — harness-verified (attach, giver hook, grant, restore-path completion + flag). Deferred: NPC schedules (anchors stay put — better for the giver's findability) + forageables. Bloom-mutation link kept latent. Attaches South Road 6054↔6055; south road (6071) now opens to River Road (leg 2, built 2026-06-26). Boots clean, errors=0. |
| 16 | River Road to Confluence | 16 | 2 | ✅ Built | NEW 2026-06-26. rooms 6090–6105 (folder river_road; zone-config defaultbiome water, region Windward Marches). Lore-and-ambient CONNECTOR, **no quest**. Stage A: the mended road dropping from the dry valley lip to the river, a barge landing, a 3-room fishing village, the Confluence bluff (second river joining), an Old Waystone Rise carrying a pre-Founding nested-rings/orbital marker (ties the symbol web). Stage B: pilgrim approach bending SE — Pilgrims' Way, pilgrim camp, sight of the spires, the Confluence Gates (intentional barred stub toward the unbuilt 5.4). 6 ambient NPCs 9410–9415 (Carew warden / Tarro dock-hand / Birrel netmender-**fishmonger** cooking vendor / Sedge old-fisher / Wess+Nara pilgrims) + 3 river fauna 9416–9418 (heron, otter, gar). Economy: river forageables 40123 watercress + 40124 mussels wired into water-biome ForageYields; fishmonger food goods 40125/40126. Lore: the lone "four-waters" aside lives only in Sedge's dialogue. North seam opens Amber Valley 6071 (the washed-out road, now mended). Boots clean, ValidateZoneConsistency errors=0 mode=panic. |
| 17 | The Confluence | 153 | 10 | ✅ Built | **COMPLETE — all 10 districts, rooms 6106–6257, pushed `524357df` (2026-06-27); Q73 Margin Notation + Q74 The Undercroft (the pre-Founding climax, both allegiance endings). The per-district build notes below predate completion and are kept as history.** Originally **SCALED UP** 2026-06-26 from 70→~150rm / 10 districts (≈ half NP's planned size) per the city-wide design layer (`docs/superpowers/specs/2026-06-26-confluence-citywide-design.md`). Tri-city temple climax; rooms 6106–6257. **District 1 — The Landings ✅ built**: 16rm 6106–6121 (folder `the_confluence`), the river-ward waterfront opening the River Road 6105 seam; new `quayfolk` faction; 8 NPCs 9419–9426 (Holt warden / Osse barge-master **w/ Davan tie-back** / Pella fish-cooking-vendor / Wult dockmaster / Ferrick chandler-general-vendor / Sybba tavern / Goran+Clout ambient); items 40127–40129; the Old Mole orbital-mark seed + Three-Rivers Overlook (temple island). No quest. Boots clean, errors=0 mode=panic; harness-walk verified STRONG. **District 2 — The Long Quay ✅ built**: 16rm 6122–6137, the commercial waterfront (river market, guild halls, counting houses, trade wharves); debuts the **`margin` faction** via Tallis the Scrivener's four-waters chart aside (seeds Q73, points to the Scholars' Quarter); 7 NPCs 9427–9433 (Edric factor / Varro importer-cooking-vendor / Lenne provisioner-general-vendor / Gett weighmaster / **Tallis scrivener-MARGIN** / Oslen guild-steward / porter); items 40130–40132; the Old Customs House orbital-mark seed. No quest. Boots clean errors=0; harness-walk verified STRONG. **District 3 — Tri-Cross Square ✅ built**: 16rm 6138–6153 (**`city` biome**), the civic heart — the central plaza, **The Three Waters inn** (w/ a proper `up`/z+1 upstairs lodging), the **Municipal Hall** (the **notation map** = the literal margin-notation Q73 breadcrumb, a fourth channel struck through). The orbital-symbol motif **escalates** to "it's everywhere" (Bell-Tower pre-Founding base + Chrysalis cocoon-ring ornament on new facades, plainly the same shape). 7 civic NPCs 9434–9440 (Orvin hall-clerk / **Savel Margin-scholar** → points to Scholars' / Bremm innkeeper-remark / Corliss general-vendor / Pellan notary / crier + citizen); item 40133 broadsheet. No quest (Q73 seeded as lore). Boots clean errors=0 mode=panic; harness-walk verified STRONG. **District 4 — The Scholars' Quarter ✅ built + Quest 73 ✅**: 14rm 6218–6231 (`city` biome), the Margin's home, attaches off the Hall of Records (6149 west). **Q73 The Margin Notation** (giver Quist the Elder 9441 @ Margin Hall) — a linear investigation: examine 3 disagreeing records via quest-gated room_interact `look` triggers (bookseller's `damaged-chart` 6226 + Tallis's `old-charts` 6131 + the hall `marginal-note` 6143), synthesize "there was a fourth," reward 30g + survey item 40138 + `margin` rep, pointing to the temple/undercroft (Q74 hook). **Harness-verified end-to-end.** 8 NPCs 9441–9448 (Quist giver / Fenn archivist / Riss cartographer / Drunn bookseller-vendor / scholars + copyist + porter + student); items 40137 book + 40138 reward (40134–40136 went to Sybba's tavern). **GOTCHAS: room_interact nouns must be ansi-highlighted HYPHENATED tokens + hyphenated noun keys; dialogue `questRequired` is a list; grant nodes need `grantsQuest`; non-vendor items need `not_salable: true`; item filenames keep the leading article ("A Fair Copy"→`a_fair_copy`).** Boots clean errors=0 mode=panic. **Next: Processional+Temple → Cloisters+Undercroft (Q74) → outer quarters.** |
| 18 | East Road to Greenford | 15 | 2 | ✅ Built | NEW 2026-06-30. rooms **6263–6277** (folder `east_road_to_greenford`; defaultbiome farmland, region The Tri-Rivers). Lore-and-ambient CONNECTOR, **no quest, no faction** — the first leg of the Eastern Arc. Attaches at Confluence **6250** (The Greenford Road, east exit opened). Dry wheat plateau → waypoint village (Wheatside Hamlet 6268, victualler Mistress Odell cooking-vendor) → descent to the Greenford river → **barred Greenford Bridge 6277** (terminus stub toward unbuilt Greenford). Symbol-web seed: the **Old Waystone 6266** (orbital nested-rings, understated, no NPC tie — eastern echo of River Road's waystone). 9 NPCs **9492–9500** (6 ambient + 3 dry-plateau fauna, each a unique mutation); items **40147–40151** (3 vendor foods + 2 forageables); dry-country forageables 40150/40151 wired into farmland/land ForageYields (TDD). 1 anchor schedule (er_victualler). Boots clean errors=0 mode=panic; **feel-tested STRONG end-to-end** (seam, all 15 rooms, waystone, vendor buy, forage, barred terminus; zero bugs; report `tools/playtest/reports/2026-06-30-local-feel-tester-east-road.md`). **GOTCHA (cost a boot cycle): zone FOLDER must = `ConvertForFilename(zone display name)`** — "East Road to Greenford" → folder `east_road_to_greenford`, NOT `east_road` (CLAUDE.md naming rule; panics at boot otherwise). |
| 19 | Greenford | 45 | 5 | ✅ Built | **COMPLETE (5/5, 45rm 6278–6322) 2026-06-30.** University town, east bank across the Greenford bridge — the Eastern Arc crux (Reth's crash-site directions), **Q75 The Surveyor's Report playable end-to-end.** City-wide design layer DONE (`docs/superpowers/specs/2026-06-30-greenford-citywide-design.md`): folder `greenford`, extends the `margin` faction, spine = **Q75 The Surveyor's Report** (directions + "it's not natural", mystery boundary locked to threshold-only). Built district-by-district (Confluence pattern). **District 1 — River District & Bridge Landing ✅ built**: 10rm **6278–6287**, the entrance — **opens the East Road bridge** (6277 south→6278; approach rooms 6273/6276 now read open). Riverfront (bridge-warden Oswin + miller/fishmonger cooking-vendors + fisherfolk/barge-hand + 2 river fauna, 9 NPCs **9501–9508**); items **40152–40154**; reuses River Road water forageables (no new Go code); 2 anchor schedules. No quest, no symbol beat (deliberately mundane; the mystery is uphill). Boots clean errors=0 mode=panic; world-critic + feel-tested **STRONG** (both; 14 findings fixed incl. dead hint keywords, a river-name unify, the washerwoman text-truncation). **District 2 — Town Center ✅ built**: 10rm **6288–6297** on **z=1** (the upper town, climbed `up` from the 6287 Town Stair); a small quiet civic hub — market square, **bookshop** (seeds the Q75 breadcrumb: the bookseller names "the surveyor who retired early"=Reth + points to Brennan uphill), **Cartographer's Rest inn**, **general store** (mixed-discipline catch-all vendor), civic lane; **University-stair stub 6297** up to District 3. 7 NPCs **9509–9515** (unique mutations); items **40155–40159** (incl. lamp oil; reuses 40077/40154). No quest grant, **NO symbol content** (guarded — mystery is District 3). Boots clean errors=0; world-critic + feel **STRONG** (lore boundary 100% clean; fixes: a mutation clone, a lamp-oil shop mismatch, more `|`-block long-text conversions). **District 3 — University District ✅ built + Q75 front half**: 11rm **6298–6308** on **z=2** (the grounds, highest; climbed `up` from the 6297 University Stair); quad, lecture hall, **library suite** (stacks→reading room→**archive**→restricted-collection stub), **Brennan's office**, common room, cloister, **college-lane stub 6308** to District 4. 8 NPCs **9516–9523** (**Brennan + Tess the Archivist = `margin`**); **the FIRST orbital-symbol beat** (Brennan's old-maps + the archive — recurring, **unexplained**, threshold-only). **Quest 75 FRONT HALF wired** (split-quest like Q74): Brennan grants `75-start` → examine the archive **`filed-survey`** room_interact (`75-survey`) → Brennan's intro to Reth (`75-intro`) → in-progress "go to Reth"; **testimony/end declared-not-granted (D4 completes)**. Boots clean errors=0, quests 65, ValidateAllFlags OK; world-critic + feel **STRONG** (Q75 front-half COMPLETE PASS in-game; symbol threshold-clean everywhere; fixes: a mis-spawned scholar, a stub `go south` prime, a dead "doubts" trigger). **District 4 — Brennan's & Reth's Neighborhood ✅ built + Q75 COMPLETE**: 8rm **6309–6316** (z=2, the residential quarter off the 6308 back gate); College Row, **Brennan's blue-door house** (lore), a tea house, gardens, **Reth's Cottage 6315** at the quiet end (the overgrown yew matches Brennan's "first right, third house" directions). 6 NPCs **9524–9529** (Reth + tea-keeper + residents). **Quest 75 COMPLETES here:** with `75-intro`, Reth gives the testimony (directions east + the lightning-split cairn + **"it's not natural"** — never *what*) and points to his **`field-notes`** (6315 room_interact, gated → grants `75-testimony`+`75-end`, `give_item 40160` Reth's marked map [not_salable], `bump_rep margin +20`); the onward reward fires on `end`. Boots clean errors=0, ValidateAllFlags OK; world-critic + feel **STRONG** — **Q75 FULL END-TO-END (D3→D4) a COMPLETE PASS in-game** (Brennan grant → archive → intro → follow directions to Reth → testimony → map + rep + done); lore boundary clean. Fixes: compass/verb directions (north→quiet, take→examine, left→right), a **semicolon-in-dialogue command-split BUG** (6 instances), mutation dedups. **District 5 — West Outskirts ✅ built (the mundane close)**: 6rm **6317–6322** (z=1, west of the town center off **6295 Town Hall Steps**); West Gate → West Road → **Coaching Stable** (ostler vendor) → **Wayfarer's Shrine** (mundane traveler's blessing — NO symbol) → Milepost → **Plymouth Road terminus stub 6322** (the road toward NP, another journey — the loop deferred; the closing beat for the whole city). 5 NPCs **9530–9534**; item **40162** trail rations. No quest, no symbol. Boots clean errors=0 warnings=0; world-critic + feel **STRONG** ("the right way to end a district — the town gestures the player outward through people"). Fixes: the invalid **`wilderness` biome → `land`** (invented biome caused ref-warnings), a `go west` stub-prime, hinted-noun key alignment (distance-board/stable-gate/road-ahead), a dead "work" trigger. **✅ GREENFORD COMPLETE — the Eastern Arc's full approach city (45rm, Q75 end-to-end). Next Eastern Arc: Cascade Pass (#20) → Eastern Highlands (#21) → Crash Site (#22, the moon-crash payoff).** |
| 20 | Cascade Pass Road | 20 | 2 | ⬜ Not started | |
| 21 | Eastern Highlands | 30 | 3 | ⬜ Not started | |
| 22 | Crash Site Interior | 20 | 2 | ⬜ Not started | |
| **TOTAL** | **29 zones** | **~770 rooms** | **~70 mini-stages** | **~27 / 29 built** — the full **New Plymouth capital (all 7 districts) is LIVE ON PROD**; the Stillwater→NP corridor is done; Southern Road legs 1–2 done; **The Confluence COMPLETE (live on prod)**; **the Eastern Arc's approach is built + UNPUSHED (2026-06-30): East Road to Greenford (#18, 15rm) ✅ + Greenford (#19, 45rm, 5 districts, Q75 end-to-end) ✅ COMPLETE**. Remaining (3 zones): **NP Sewers (#13.5)** standalone; and the endgame — **Cascade Pass (#20) → Eastern Highlands (#21) → Crash Site (#22)** (the moon-crash payoff, reached from NP East Gate via Cascade Pass). | 2026-06-19: Stillwater→NP corridor **expanded** (the 5.1–5.5 zones, +~94 rooms) so the capital is ~4–5 travel-days north with open reaches reserved for outdoor expansion and a farm/industry hinterland — see the World Atlas (`docs/world_atlas_mock.html`) and the "Water Routes & Ferries" section. New Plymouth itself is scoped to grow well beyond its initial 170 (→300+ via stubs, and likely larger). PLUS large off-plan built footprint not counted here (Pothole Coulee newbie 169rm, Ironwind 123rm, A Dark Forest 81rm, Thornwall, Watchers Crossing, Dustwalk, Labyrinth, Stillwater Marsh, Fernway South). **Next on-plan: the endgame arc — Cascade Pass Road (#20, NP East Gate → the mountains) → Eastern Highlands (#21, the buried hull) → Crash Site Interior (#22, the moon-crash truth). Greenford (#19) is done and holds Reth's directions to it. NP Sewers (#13.5) is a standalone fill-in.** |

*New Plymouth's 170 initial rooms include 21 expansion stubs (3 per
district) that are visible but inaccessible. When all stubs are built
out in future phases, NP grows to 300+ rooms. The stub count is not
included in the room totals above — they are described elements of
existing rooms, not separate rooms.*

---

## Water Routes & Ferries

*Planned fast-travel network (added 2026-06-19). Routes are reserved here so
the geography supports them; the actual ferry vessels + travel mechanic are a
later build. Ferries are intended to be **paid and faster than walking** — the
long overland corridors stay relevant for low-coin players, exploration, and
leveling; boats are the premium shortcut.*

| Route | Type | Connects | Notes |
|-------|------|----------|-------|
| Stillwater → New Plymouth | lake-and-river packet | Stillwater ↔ NP Docks | The headline shortcut — skips the ~4–5 day overland corridor (zones 5–5.5). Stillwater is a lake town with working docks; the packet runs the water route to the coastal capital. |
| Stillwater → The Confluence | downriver barge | Stillwater ↔ The Confluence | Ties the northern hub into the southern river-city network. |
| The Confluence → New Plymouth | river barge | Confluence ↔ NP Docks | **Novel canon** — Davan rides this barge north (ln 1116, 1236). |

This makes **Stillwater a ferry hub** and forms a water triangle
(Stillwater ⇄ Confluence ⇄ New Plymouth, plus Stillwater ⇄ NP direct). New
Plymouth is coastal, so its docks are also the seed for **future coastal
ferries** (e.g., to Tidemark and other seaboard cities) when those zones exist
— leave a described dock/harbormaster stub in the NP Docks build.

Design seams for the future ferry mechanic: fares (gold sink), schedules or
on-demand departure, a short in-transit scene, and gating (a route may require
discovering the dock or an NPC contact first). Boats should never be *strictly*
better than walking for a first-timer who hasn't earned the fare.

---

## Cross-Zone Questlines

These quests span multiple zones and provide the connective tissue that
makes the world feel like one place rather than a series of disconnected
areas.

### The Inner Orbit Mystery
The central mystery of the novel, told through discoverable fragments.
Players encounter the symbol, the margin notation, the counting rhyme,
and the faces across multiple zones. No single NPC explains it. The
player assembles the picture the way the novel's characters do — piece
by piece, from different angles, with the full truth waiting at the
crash site.

**Touchpoints:** Ashwick (a carving in the old cottage), Amber Valley
(the Chrysalis grove marker), the Confluence (temple architecture + the
undercroft), Greenford (Brennan's maps + Reth's testimony), New Plymouth
(the cooperage group + Edvar's face + the gallery), the Eastern Highlands
(the hull), the Crash Site (the answer).

### The Bloom Trail
The supply chain from captive hollow people to refined wafers to users.
Starts in NP's docks district, connects to the Noble Quarter property
records, touches the Common Quarter, and resolves in the underdocks.

### The Bloodline's Reach
Hints that the bloodline's authority extends further than the official
account suggests. Property records, temple cooperation, the Restricted
Collection, Horst's operation, Crane's review — fragments across the
Temple District, Noble Quarter, Merchant Quarter, and the Confluence.

### The Hollow Question
What can unchanged hands actually do? This question accumulates across
zones: Ashwick (Maren's cottage), the Confluence (the sealed containers),
NP Crafting Quarter (the cooperage debate), NP Temple District (archive
materials), and the Crash Site (the answer — unchanged hands operate
everything).

---

## Notes

- **Room IDs:** Assign new ID ranges per zone to avoid conflicts with
  existing content. Suggested: 4000+ for the expansion zones.
- **Instance saves:** After building each mini-stage, verify no stale
  instance saves exist in `rooms.instances/`.
- **Aldric direction fix:** Chapters 13 and 17 of the novel need "east"
  changed to "west/northwest" for Aldric's travel from Confluence/Greenford
  to New Plymouth. Flag for manuscript revision.
- **Existing zone connections (verified 2026-06-19):** see the reconciliation
  banner at the top of this file for the full cross-zone backbone. In short:
  Thornwall City ↔ Outskirts ↔ Watchers Crossing ↔ Marches Spur Road ↔
  {Ashwick, The Fernway} ↔ North Road → Stillwater. World Road is no longer a
  deprecation candidate — it survives as a single-room (2001) connector
  between Dustwalk Road and the Labyrinth of Low Tunnels.
- **Mob scaling:** Road zones should scale from the existing Ironwind
  Steppe difficulty (mid-level) through to endgame at the Eastern
  Highlands and Crash Site. New Plymouth city zones should have minimal
  combat (guards, pickpockets) with the sewers and underdocks as the
  exception.
