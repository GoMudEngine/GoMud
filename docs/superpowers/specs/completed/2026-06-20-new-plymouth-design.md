# New Plymouth — Master Build Plan (Design Spec)

*Date: 2026-06-20 · Status: design, plan-only · The capital city build.*

> **What this document is.** The master plan for building **New Plymouth**,
> the capital — ~170 rooms across 7 sub-zones, designed as a **living city**.
> This is a **plan-only** artifact: it defines geometry, the resident model,
> civic infrastructure, supply chains, ID allocation, the build sequence, and
> per-district specs. **It does not build content.** Each district (and the
> engine prerequisite) becomes its own spec → plan → build session afterward.
>
> **Canon source:** `docs/new_plymouth_canon.md` (extracted from
> *What the Moons Keep*). The MUD is set **after** the novel.
>
> **Supersedes/expands:** `docs/ZONE_EXPANSION.md` Phase 4 (which sketched the
> 7 districts). This plan keeps that district structure but reorganizes the
> *method* around residents-first and corrects the geometry to novel canon.

---

## 0. Decisions locked (this session)

| Decision | Choice |
|----------|--------|
| Session scope | **Plan-only.** Build in later sessions. |
| Canon fidelity | **Novel-grounded, freely expanded.** |
| Build order | **Canon arrival order — Docks first.** |
| Supply/maintenance engine (chunk 3.5) | **Build a scoped 3.5 FIRST**, as an engine prerequisite. |
| Novel timing | **After the novel.** Protagonists have left east; the city bears their traces. |
| Population depth | **Maximal — ~45 anchors + ~80 ambient** (~125+ living NPCs). |

---

## 1. The organizing principle — a living city, designed resident-first

**The city is a population, not a floorplan.** The recurring failure mode we are
explicitly designing against: building rooms first, then *bolting* schedules and
homes onto NPCs afterward (the chunk-3.2 pilot literally invented "a new
above-shop home room" for Blacksmith Kerra to satisfy her schedule). That makes
a schedule *work* but it doesn't make a *person*.

**Inversion:** author the people and their daily lives first; **rooms exist
because residents need them.** A blacksmith is a person with a home in a named
residential cluster, who walks to the forge at dawn (`activity: craft`), buys
iron from a dock importer (his supply chain literally stitches the Docks to the
Crafting Quarter), eats at a cookshop, and drinks at the Salt Cellar — every one
of those a real room serving a real routine, several of them *crossing district
boundaries*.

### Why this is now achievable (it wasn't an engine gap)

The **mob-aliveness roadmap is 45/45 complete.** Every primitive a living city
needs already shipped to production:

| Life-sheet element | Engine primitive (shipped) |
|---|---|
| Where they sleep | NPC schedules (3.2) route via `pathto`; `activity: sleeping` (3.3) |
| Work, and visibly working | schedule day segment; `activity: craft` gates `TickMobCraft` |
| Where they eat / drink | schedule meal + evening segments → cookshop / tavern |
| Who they live/work/drink with | NPC↔NPC relationships (1.6) |
| What faction they serve | factions (1.2) + crime/wanted (1.3) |
| What they *want* | strategic goals (4.1–4.6) |
| Where supplies come from | NPC market participation (5.4), mob `buy` (2.1) |
| Two NPCs chatting in the square | NPC↔NPC conversation (3.6) |

**The one real gap** — reusable supply/restock activity (chunk **3.5**) — was
deferred precisely until "content authoring hits real duplication pain across
multiple smiths/farmers/librarians in multiple zones." A 170-room capital full
of crafters and vendors **is** that trigger. We build a scoped 3.5 first (§6).

---

## 2. City geometry & zone structure

Canon-corrected layout (north = up; **sea/harbor to the west**; the river enters
from the northeast; **Temple east on the ridge**, **Palace/Noble on the northern
rise** — the far end from the docks, the endgame climb):

```
                         [ PALACE ]  (northern rise — endgame stub)
                              |
                       [ NOBLE QUARTER ]
                        bloodline addresses
                              |
   [ DOCKS ]——[ CRAFTING ]——[ MERCHANT ]——[ TEMPLE DISTRICT ]
   western      Inkwalk,      central        eastern ridge,
   waterfront   cooperage      square        Grand Temple, Archive
      |            |             |
      |        [ COMMON / TANNERY — Carter's Rise ]
      |            |
   (the Ford)  (Old Quarter / Canal beneath Crafting+Docks: z = -1)
      |         Lintel St, footbridge, Deren's basement
      |            |
      +———[ NP OUTSKIRTS (existing) ]———(East Gate)
         River Road 5480              East Gate 5471
         → unwatched → DOCKS          → watched → COMMON
```

**Two entries (canon-exact):**
- **East Gate** (Outskirts 5471, watched) → **Common Quarter** (overland,
  working-class first impression; Horst's agents monitor it).
- **The Ford** (Outskirts River Road 5480, unwatched, "two miles west of the
  east gate, comes out on the river road") → **Docks** (the quiet way in).

**The "Sewers" is canon-reframed as The Old Quarter / Canal District** — buried
pre-colonial stonework, the Lintel Street canal, the stone footbridge, sealed
cellars, and Deren's Bloom basement. Access from Docks, Crafting (the
cooperage's concealed hatch), and Common. Far richer than "sewer + rats," and
it hosts the Bloom Trail climax.

### Zone & room-ID blocks

Each district is its own zone folder, 100-wide room block (initial build + stub
expansion headroom). Verified clear: room next-free is **5482**; `5500–6199` is
unused.

| Zone (4.x) | Folder | Room block | Initial | z |
|---|---|---|---|---|
| Docks District | `new_plymouth_docks` | 5500–5599 | 30 | 0 |
| Common Quarter | `new_plymouth_common` | 5600–5699 | 25 | 0 |
| Crafting Quarter | `new_plymouth_crafting` | 5700–5799 | 25 | 0 |
| Merchant Quarter | `new_plymouth_merchant` | 5800–5899 | 25 | 0 |
| Temple District | `new_plymouth_temple` | 5900–5999 | 25 | 0 |
| Noble Quarter | `new_plymouth_noble` | 6000–6099 | 20 | 0 |
| Old Quarter (canal/sewers) | `new_plymouth_oldquarter` | 6100–6199 | 20 | −1 |

> Folder names must round-trip through `ConvertForFilename(zoneDisplayName)`.
> Confirm each display name (e.g. "New Plymouth Docks") converts to the folder
> above before authoring rooms — a mismatch panics at load.

### Other ID blocks (verified next-free in parens)

| Type | Block | Notes |
|---|---|---|
| Mobs (9217) | **9300–9499** | ~125 NPCs; per-district sub-blocks of ~25 |
| Items (40079) | **40100–40299** | zone-flavor goods + quest items |
| Quests (63) | **63–90** | 7 district lines + Bloom Trail spine + side |
| Dialogue (9209) | **9209–9399** | ~45 anchor trees |
| Buffs (90) | **90–95** | Bloom high / withdrawal, if modeled |
| Factions | new slugs | `bloodline_domestic`, `bloom_trade`, `cooperage_circle` (lore), `temple_np`, `np_dockfolk`, `np_commonfolk` |

Per-district mob/item sub-blocks get allocated at each district's build session
via `python tools/id_inventory.py --alloc`.

---

## 3. The resident model

Three tiers keep "maximal" tractable.

### Tier A — Anchors (~45, ~6–7 per district)
Named NPCs with a **full life sheet**. Canon seeds them where it exists; we
invent the rest. The life-sheet template (authored *before* the NPC's rooms):

```
ANCHOR: <name>
  district:       <which zone>
  mutation:       <unique visible mutation — no two share a description>
  home:           <room — where they sleep (activity: sleeping at night)>
  workplace:      <room — where they work by day (activity: craft|work|...)>
  meals:          <room(s) — cookshop/tavern/market, midday + evening>
  social:         <room — pub/temple/well, evening or rest>
  supply_source:  <upstream room/NPC their materials come from (crafters)>
  relationships:  <1.6 edges: family/friend/rival/lover/employer/employee>
  faction:        <1.2 membership>
  motivation:     <4.x strategic goal coloring behavior>
  schedule_id:    <derived 24h routine: home→work→meal→social→home/sleep>
  dialogue:       <≥3 topics beyond quest function; voice notes>
```

A district's rooms are then derived as: **homes + workplaces + the civic rooms
their routines require + the streets that connect them.** No orphan rooms.

### Tier B — Ambient residents (~80)
Background populace on **pooled cluster schedules** — e.g. a "Common Quarter
laborer" template routing home-cluster → work-district → tavern → home. They
make crowds and fill streets but don't need bespoke life sheets. Authored as a
handful of reusable schedule templates + name/appearance variation.

### Tier C — Transients (rotating pool)
Travelers, pilgrims, drovers, dockhands with **no city home** — they cycle
through inns, the docks, and the gates (reuse caravan/forager-style movement or
simple wander+inn schedules). They give the gates and inns turnover.

---

## 4. The post-novel setting state

The player arrives in the **aftermath**. This is a feature: the departed cast
becomes a lore/quest layer, and several canon threads are *live problems*.

- **The cooperage stands abandoned** — basement intact, lamps cold, the
  group's traces discoverable. Reconstructing what happened (and the
  fourth-moon secret) is a lore questline. Faction `cooperage_circle` exists as
  recoverable knowledge, not living membership.
- **Edvar's shop is shuttered** (or under a new owner); the Inkwalk remembers
  him. His maps are sought-after.
- **Tova still rents rooms** near the old seminary and tells stories of "the
  group that left." The room Vane & Maren used is now empty.
- **Horst is still in the Merchant Quarter, still hunting** — a live antagonist
  presence; his agents went east. Players can run afoul of the domestic office.
- **The Bloom trade is disrupted-but-regrowing** — Cade reported, Deren's
  basement exposed; a successor is filling the vacuum. The Bloom Trail is the
  city's spine questline and is *current*, not historical.
- **Junie was freed and went south** — referenced; a possible future thread.

---

## 5. Shared civic infrastructure (designed once, serves everyone)

These are the **intersection rooms** where routines cross and 3.6 conversations
fire. Designed at the city-wide layer so every district's residents have
somewhere to eat, drink, draw water, worship, shop, and gather.

| Civic space | District | Canon anchor | Routine role |
|---|---|---|---|
| The Salt Cellar (inn/tavern) | Docks | canon | dockfolk evenings, transient lodging |
| A dockside cookshop | Docks | — | dockfolk midday meals |
| Aldric's public house | Docks | canon | rougher lodging, no questions |
| The Gilt Threshold (high inn) | Merchant | canon | merchant-class social |
| The central market | Merchant | canon | everyone shops; surveillance beat |
| Carter's Rise well + flower market | Common | canon | water; the hollow-accusation scene |
| A neighborhood tavern | Common | — | commonfolk evenings |
| A cookshop / street-food row | Common | — | commonfolk meals |
| The Grand Temple | Temple | canon | rest, respawn, worship; pilgrims |
| A bathhouse | Common/Merchant | — | cross-class mixing |
| The fighting pit | Common | (expansion stub) | gambling/combat social |

Respawn point: the **Grand Temple** (Temple District), per the original design.

---

## 6. Supply model & engine prerequisites

### 6.1 The centralized Docks-warehouse model (Phase 1 — now)

The whole city's supply originates in **one** place: **everything is imported by
sea, landed at the Docks, and stored in massive Docks warehouses.** From there,
goods are **distributed into the districts by mini-caravans** — a wagon parked at
a warehouse depot, a runner walking a circuit out to district vendors and
crafters. Every anchor crafter's `supply_source` therefore resolves, directly or
through a delivery, to a **Docks warehouse**. This makes the Docks the visible
economic heart and stitches all seven zones to one origin.

**Engine mapping (no new movement engine needed):**
- **Mini-caravans = the caravan runner-delivery pattern (chunk 3.8, shipped).** A
  wagon parks at the Docks depot; a runner (`loop_shape: oneshot` sub-patrol)
  walks goods out to each district's vendors on a loop, then returns. Routes are
  authored caravan/patrol YAML, not code. Inter-district runs ride on the
  cross-zone patrol layer (chunk 3.7 — **verify, see §6.3**).
- **Barge/boat traffic = ambient + transient flavor.** "Fake in a great deal" of
  it: scheduled barge-arrival/departure room emotes at the quays, transient
  stevedore/longshore mobs that spawn, unload, and despawn, named barges as
  scenery nouns, the dock hiring board turning over. The goal is a waterfront
  that *visibly* feeds the city.

**Phase 2 (future — after the city is fleshed out):** layer in diversified
supply routes — food from the surrounding farms (Kingsbarrow Vale already exists
as the granary belt), timber and ore from Kilnreach Works, foraging operations,
overland caravans through the East Gate. The centralized model is deliberately a
**starting simplification**; these routes attach later without reworking the
Docks origin.

### 6.2 Engine prerequisite — scoped chunk 3.5 (its own spec+build first)

**Goal:** make the **consume→produce** half of the loop mechanically real, so a
vendor/crafter's shop stock comes from delivered supply rather than from nothing.
(Distribution — getting materials *to* the vendor — is handled by the
mini-caravan layer above, already shipped.)

**Scoped in:**
- A reusable **restock activity** dispatched by a schedule segment
  (`activity: restock`, or extend `activity: craft`): the vendor/crafter consumes
  delivered input materials and produces output shop stock on a cadence, with
  believable caps.
- A lightweight **supply-source link** on the crafter (the Docks warehouse / the
  delivering caravan it draws from), so the chain warehouse → mini-caravan →
  vendor → shop list is real and inspectable.
- Close the known **"crafter ticks but the item never appears in the shop list"**
  bug (see `project_crafted_items_buyability_investigation`).

**Scoped out:** a full activity *library* (tend_crops, shelve_books, etc.) — we
build only the restock/supply activity the city needs; other flavor activities
stay per-segment `idlecommands:` for now. Phase-2 diversified supply routes.

**Deliverable:** separate `docs/superpowers/specs/<date>-mob-aliveness-3.5-*`
spec + plan + build, completed and smoke-verified **before** any district build.
Update `MOB_ALIVENESS_ROADMAP.md` chunk 3.5 status on completion.

### 6.3 Dependency to verify — cross-zone caravan (chunk 3.7)

The mini-caravan distribution assumes caravans/runners can **cross district (=
zone) boundaries** within the city. Chunk **3.7** (inter-zone patrols + caravan
unification) is marked **Done** in the roadmap progress tracker and roll-up, and
the Done-3.8 chunk builds on "the post-3.7 patrol substrate" — **but the 3.7
detail header still reads "Not started" (stale)**. **Action:** confirm in code
(`internal/caravan/`, `internal/mobs/patrol*.go`) that cross-zone caravan
movement actually works **before** the city-wide build. If it does not, fold the
gap into the engine-prerequisite phase alongside scoped 3.5. Do not author
intra-city mini-caravans against an unverified capability.

---

## 7. Build sequence & dependencies

```
(P) Engine prerequisites                ── own spec(s)+plan+build:
        │   - verify cross-zone caravan (chunk 3.7) works; build if not (§6.3)
        │   - scoped chunk 3.5 restock activity (§6.2)
        │
(C) City-wide design layer             ── one spec covering the whole city:
        │   - the ~45 anchor roster (life sheets)
        │   - ~80 ambient + transient templates
        │   - civic infrastructure rooms
        │   - the supply map: all → Docks warehouses → mini-caravan routes (§6.1)
        │   - the coordinate map (all 7 zones placed, Cartesian-clean)
        │   - faction definitions (new slugs)
        ▼
(1) Docks ─→ (2) Common ─→ (3) Crafting ─→ (4) Merchant ─→
(5) Temple ─→ (6) Noble ─→ (7) Old Quarter
        each: own spec+plan+build, pulling residents from the (C) roster
```

**Why a city-wide design layer (C) before any district:** resident-first means
the cast, relationships, factions, and supply chains are *cross-district* by
nature. Designing them once up front keeps Docks-built NPCs coherent with the
Merchant supplier they buy from and the Common cousin they visit — even though
those districts get built later. Each district build then *consumes* the roster
rather than inventing in isolation.

**The Bloom Trail** (multi-zone spine) is authored across Docks → Crafting →
Old Quarter → Noble as those districts land; its breadcrumbs and payoff are
specified in the city-wide layer so the thread is coherent from the first
district.

---

## 8. Per-district specs (plan altitude)

Each entry is a build brief, not authored content. Room budgets are initial;
every district carries **expansion stubs** (described, gated, not broken exits)
per `ZONE_EXPANSION.md` Phase 4. Quality bar = `ZONE_EXPANSION.md` "Quality
Standards" (3-layer room descriptions, ≥2 nouns/room, 80-col, discoverable
quests, idle behaviors, ≥3 dialogue topics).

### 8.1 Docks District — `new_plymouth_docks` (5500–5599, 30 rm) — **built 1st**
- **Theme:** commerce and its shadows; the western waterfront; the canon entry;
  **the city's import head and supply origin** (§6.1).
- **Key rooms:** the quays/barge landing, **massive warehouse row (the city's
  supply store)**, the **mini-caravan depot** (where distribution wagons load and
  runners set out), the fish market, the quiet dock end, the Salt Cellar,
  Aldric's public house, a flophouse, Cutter's Lane, Cade's fabric-front (outer
  edge — bridges to Crafting), a hiring board.
- **Anchors (canon-seeded):** Jesset (foreman, finger-joint mutation),
  a successor Bloom-runner filling Cade's role, the Salt Cellar keeper, a
  harbormaster, a **warehouse master / dispatch agent** (runs the depot), a
  fishmonger, a pawnbroker. Plus invented dockfolk.
- **Ambient/transient (heavy):** barge arrival/departure emotes on a cadence,
  transient stevedores that spawn-unload-despawn, named barges as scenery, a
  busy hiring board — "fake in a great deal of barge/boat traffic" so the
  waterfront visibly feeds the city.
- **Supply role:** **everything originates here.** Sea imports land at the
  warehouses; **mini-caravans (§6.1) distribute to every district.** Every
  crafter's `supply_source` resolves back to these warehouses.
- **Questlines:** **Dock Rat** (wrongly-accused dockworker) + the **Bloom Trail**
  opening breadcrumbs (overhear the back room; an addled mutterer; a constable's
  hint). Expansion stubs: chained shipyard, sealed warehouse, collapsed underdock
  tunnel.

### 8.2 Common Quarter — `new_plymouth_common` (5600–5699, 25 rm) — **2nd**
- **Theme:** where most people live; crowded, loud, real; the East-Gate entry.
- **Key rooms:** tenement rows (the city's main **residential cluster** — many
  anchors *sleep* here), Carter's Rise (well + flower market), the tannery
  streets, a neighborhood market, cookshop row, a neighborhood tavern, a
  back-alley healer, the fighting-pit gate (stub).
- **Anchors:** a landlady, a street-sweeper-who-knows-everything, a retired
  soldier, errand-running kids, a tannery master, a back-alley healer.
- **Civic role:** holds the most *homes* — many crafters/vendors who *work* in
  other districts *live* here; their schedules cross the city daily.
- **Questlines:** **The Street Sweeper's Secret** (pre-Founding fragments →
  small undercroft). The Carter's Rise hollow-accusation scene as a public
  hazard. Stubs: fighting pit, condemned tenement upper floors, chained river gate.

### 8.3 Crafting Quarter — `new_plymouth_crafting` (5700–5799, 25 rm) — **3rd**
- **Theme:** honest work; the Inkwalk; the cooperage; quiet rebellion (now
  departed — its traces remain).
- **Key rooms:** the Inkwalk (Edvar's shuttered shop, a chandler, a bookseller),
  a blacksmith (+forge), an alchemist, a leatherworker, a tailor, Asha's glass
  workshop, the **cooperage** (front shop + yard + abandoned basement), Cade's
  back room (from the Docks side), crafting stations for players.
- **Anchors:** a blacksmith, an alchemist, a glassblower (Asha's successor or a
  peer), a leatherworker, a bookseller, the cooperage apprentice still minding
  the front. Each crafter gets a **supply_source** (Docks warehouse / dock
  importer) — the §6 engine makes this real.
- **Questlines:** **The Apprentice's Commission** + the **cooperage lore
  reconstruction** ("the group that left") + Bloom Trail middle (Cade → the
  Old Quarter). Stubs: barred guild hall, fenced kiln complex, the cooperage's
  deeper locked cellar (→ Old Quarter).

### 8.4 Merchant Quarter — `new_plymouth_merchant` (5800–5899, 25 rm) — **4th**
- **Theme:** wealth on display and hidden; the legitimate face of power; Horst's
  base.
- **Key rooms:** the central market square, permanent shops (weapons, armor,
  general goods, moneylender), the auction house, the Gilt Threshold inn,
  trading houses, a bank, a currency exchange, a notary, **Horst's rented
  house** (live antagonist), a bloodline permit-clerk's office.
- **Anchors:** an auctioneer, a moneylender, a weapon dealer, an armorer, a
  bloodline clerk, **Horst** (antagonist, `bloodline_domestic`), the Gilt
  Threshold keeper.
- **Supply role:** the *distribution hub* — finished goods sold on; the lawyer/
  records thread links to the Bloom Trail (Noble Quarter property ownership).
- **Questlines:** **Market Manipulation** (unfair competitor → smuggling →
  Docks). Stubs: walled consortium garden, Gilt Threshold private upper floors,
  sealed inter-city counting house.

### 8.5 Temple District — `new_plymouth_temple` (5900–5999, 25 rm) — **5th**
- **Theme:** institutional power in spiritual dress; the eastern ridge; faith
  built on a story that is wrong.
- **Key rooms:** the Grand Temple (entrance hall, nave, meditation garden,
  pilgrim hostel — **respawn point**), the Keeper's House (chapel of Saint Imret;
  Yelin, Thane), a seminary, a religious bookshop, a healer's chapel, the
  **Archive / Restricted Collection** entrance (gated stub), an old cemetery and
  cloister where the orbital symbol survives in pre-Chrysalis stonework.
- **Anchors:** a temple canon, a scribe, Yelin (warden), Father Thane, a
  skeptical scholar, a **doubting novice**, a healer.
- **Questlines:** **The Doubting Novice** + the **Gallery/Archive cipher**
  thread (the pre-theology symbol). Stubs: the Restricted Collection itself,
  the seminary annex "under restoration for three years," a locked crypt.

### 8.6 Noble Quarter — `new_plymouth_noble` (6000–6099, 20 rm) — **6th**
- **Theme:** power without apology; the northern rise; the watched, quiet end of
  the city.
- **Key rooms:** wide boulevards, the **Palace approach** (exterior only —
  endgame stub), the Bloodline Administrative Office, a Founding-era art gallery
  (clues in the oldest pieces), a high-end clothier, the gated residential lane
  (the canon delivery address among them), private gardens over walls.
- **Anchors:** a bloodline functionary, a nervous servant, a tour guide reciting
  approved history, a liveried porter, an art-gallery keeper, ceremonial guards.
- **Questlines:** **The Gallery Cipher** + Bloom Trail payoff (the delivery
  address; property records from the Merchant Exchange). Stubs: the Royal Palace
  gates (major endgame), the gated estate lane, the Administrative Office's
  private chapel.

### 8.7 The Old Quarter — `new_plymouth_oldquarter` (6100–6199, 20 rm, z=−1) — **7th**
- **Theme:** the buried pre-Founding city; the canal; what the surface forgot.
- **Key rooms:** main canal tunnels connecting Docks/Common/Crafting; Lintel
  Street and the stone footbridge; **Deren's exposed Bloom basement** (the
  production room — rescue/crime scene); a cooperage-group secondary cache; a
  sealed pre-Founding chamber; the concealed cooperage hatch.
- **Mobs:** canal rats, slimes, feral mutated animals, Bloom-addled wanderers,
  thugs (human).
- **Questlines:** the **Bloom Trail climax** (free the captive / dismantle the
  network) + a pre-Founding lore vault. Stubs: a collapsed worked-stone tunnel,
  a flooded passage (needs a boat), a welded Noble-Quarter drain (future
  infiltration).

---

## 9. Quest map (canon-seeded)

| # (alloc 63+) | Title | District(s) | Canon seed |
|---|---|---|---|
| 63 | Dock Rat | Docks | invented (plan) |
| 64 | The Bloom Trail (spine) | Docks→Crafting→Old Q→Noble | Cade/Deren/Noble address |
| 65 | The Street Sweeper's Secret | Common | invented (plan) |
| 66 | The Apprentice's Commission | Crafting | invented (plan) |
| 67 | The Group That Left (cooperage lore) | Crafting/Old Q | cooperage circle |
| 68 | Market Manipulation | Merchant | invented (plan) |
| 69 | The Doubting Novice | Temple | invented (plan) |
| 70 | The Gallery Cipher | Noble/Temple | the pre-theology symbol |
| 71+ | reserve | — | side/transient/faction favors |

Every quest must meet the **Breadcrumb Rule** (≥3 independent discovery paths,
≥2 resolution paths, no dead ends) and the quest SOPs (questExcluded with end
tokens, quest/task triggers, give.go recovery nodes).

---

## 10. Risks & open items

- **Schedule/path load at scale.** ~125 NPCs with schedules + patrols across a
  170-room city is the largest routing load yet. Validate with `cartcheck`,
  schedule coverage validators, and a perf datapoint (see
  `reference_prod_perf_baseline`). Stagger by district; watch tick cost.
- **Mini-caravan + barge load.** Intra-city distribution adds continuous
  inter-zone caravan/runner traffic (§6.1) on top of NPC schedules, and the Docks
  carry heavy ambient barge emotes + transient spawns. Budget this in the perf
  pass; keep caravan routes and barge cadences tunable. The single-origin model
  also means a Docks supply stall would starve the whole city — acceptable for
  Phase 1, but note it before Phase-2 diversification.
- **Cartesian consistency across 7 adjoining zones + a z=−1 layer.** The
  coordinate map (city-wide design layer) must place all zones cleanly before
  any room is authored; the Old Quarter sits at z=−1 under multiple districts.
- **Folder-name `ConvertForFilename` round-trip** for all 7 zones — verify
  before authoring (panics at load otherwise).
- **Instance-save shadowing** during smoke tests — follow the nuke-instances SOP.
- **Bloom mechanics scope.** Whether Bloom is a real consumable buff/withdrawal
  system (buffs 90–95) or purely narrative is deferred to the Docks/Bloom-Trail
  build spec — flagged here, decided there.
- **Protagonist references only.** Keep Maren/Davan/Vane/Aldric out as NPCs
  (they left east); they appear only as lore/dialogue references.

---

## 11. Next step

Per the brainstorming → planning flow, the immediate next artifact is **not**
content — it is the writing-plans pass that turns this master plan into the
**first executable plan**: the scoped **chunk 3.5** engine prerequisite (§6),
followed by the **city-wide design layer** (§7-C). District builds follow in
arrival order.
