# New Plymouth — City-Wide Design Layer (Design Spec)

*Date: 2026-06-20 · The shared skeleton every district build pulls from.*

> **What this is.** The city-wide design layer called for by the New Plymouth
> master plan (`docs/superpowers/specs/2026-06-20-new-plymouth-design.md` §7-C):
> the **coordinate/zone geometry**, the **6 faction definitions**, the **civic
> infrastructure**, the **Docks-warehouse supply map**, and a **~45-anchor
> roster sketch** with its cross-district relationship + supply web. It is
> designed resident-first: rooms and streets exist because residents need them.
>
> **Scope:** structural skeleton + roster *sketch* (one line per anchor). Full
> anchor life-sheets (schedules, dialogue), the ~80 ambient/transient templates,
> and exact per-room coordinates are authored **per district at build time**,
> pulling from this layer. Canon bible: `docs/new_plymouth_canon.md`.
> Build order: districts **Docks-first** (§8).

---

## 1. The organizing principle (restated)

The city is a **population, not a floorplan**. Geometry is shaped by how people
live and move — daily routines (home → work → meals → pub → sleep) that cross
districts, supply that flows, civic hearts where routines intersect. Two
arteries carry the city's life and cross at its heart; neighborhoods hang off
them; civic infrastructure sits *on* them where lives converge. Every primitive
this needs already ships (aliveness roadmap 45/45: schedules, sleep, patrols,
NPC↔NPC conversation, relationships, factions, goals, market participation).

---

## 2. Geometry — arteries first, neighborhoods hung on them

North of the existing outskirts (`y > 78`). Canon-oriented: Docks west on the
water, Temple east on the ridge, Noble/Palace the northern climb, Merchant the
central hub.

```
 y
135  ░ PALACE ░  (stub; far north end of the Processional)
130  ┌── NOBLE QUARTER ──┐            elite homes, watched streets
116  ├── MERCHANT ──┬── TEMPLE ──┐    Merchant = THE HEART · Temple = E ridge
     │  (CENTRAL    │  DISTRICT   │
     │   SQUARE)    │ (Grand Temple,
100  ┌── CRAFTING ──┤  Archive)   │    Inkwalk + cooperage (river edge)
 98  ├── DOCKS ──┬──┴ COMMON ─────┘    Common = most homes; Carter's Rise
 82  │ (water W) │  (gate entry)        well + taverns
 80  └────┬──────┴──────┬────────┘
      River Rd 5480     East Gate 5471   ← attach: ADD north exits
 78   (-17,78)          (-18,77)
            ░ NP OUTSKIRTS (existing) ░
        x: −42 ←———————— 0 ————————→ +24
   OLD QUARTER / canal (Lintel St): z = −1, beneath Docks+Crafting+Common
```

### The two arteries (first-class design objects; district builds must honor)

- **The Long Market (E–W spine):** Docks → Crafting → **Central Square** →
  Temple. The working/commerce axis **and the mini-caravan supply route** (§5).
  Goods land at the Docks and flow east to every vendor; NPCs commute along it.
- **The Processional (N–S spine):** East Gate → Common → **Central Square** →
  Noble → Palace. The civic/ceremonial axis — the gate-to-palace climb.
- They **cross at the Central Square** (Merchant) — the great market, the one
  room everyone passes through. That crossing is *why* Merchant is the hub.

### Zone regions (non-overlapping; exact per-room coords assigned + `cartcheck`-validated per build)

| Zone | Folder | x-range | y-range | z | Rooms | Role in the living city |
|------|--------|---------|---------|---|-------|--------------------------|
| Docks | `new_plymouth_docks` | −42 … −22 | 82 … 102 | 0 | 30 | import origin; dock community; rough lodging |
| Common | `new_plymouth_common` | −20 … −2 | 80 … 98 | 0 | 25 | **most homes**; the working populace; gate entry |
| Crafting | `new_plymouth_crafting` | −24 … −8 | 100 … 114 | 0 | 25 | artisan workplaces; the Inkwalk; cooperage traces |
| Merchant | `new_plymouth_merchant` | −8 … +8 | 100 … 116 | 0 | 25 | the heart; great market; banks; Horst's base |
| Temple | `new_plymouth_temple` | +8 … +24 | 100 … 116 | 0 | 25 | faith; respawn; Archive; pilgrim arrival |
| Noble | `new_plymouth_noble` | −8 … +10 | 116 … 130 | 0 | 20 | elite homes; bloodline admin; the watched streets |
| Old Quarter | `new_plymouth_oldquarter` | −28 … −8 | 86 … 102 | −1 | 20 | buried canal city; Bloom basement; pre-Founding lore |

**ID blocks** (from the master plan, verified clear): rooms 5500–6199 (100/zone);
mobs 9300–9499; items 40100–40299; quests 63–90; dialogue 9209–9399; buffs 90–95.

### Connections

- **River Road (5480)** → new north exit → runs west along the waterfront into
  the **Docks** (the discreet/ford route, per canon).
- **East Gate (5471)** → new north exit → gate plaza into the **Common Quarter**
  (the watched overland entry).
- Districts tile north along the arteries; **Old Quarter (z=−1)** has stair
  access from Docks, Common, and the cooperage (Crafting).
- The exact gate-transition rooms are resolved in the Docks/Common builds.
- **Update `docs/coordinate_map.md`** as each district lands (it is currently
  stale — predates the whole corridor).

---

## 3. Factions (the substrate of daily disposition)

Six new faction slugs. Membership colors each anchor's trust, gossip, drinking
company, and resentments. Definition YAMLs at
`_datafiles/world/dogmud/factions/{slug}.yaml`.

| Slug | Who | Default rep (outsiders) | Allies | Enemies |
|------|-----|--------------------------|--------|---------|
| `bloodline_domestic` | Horst, agents, clerks, Noble admin, palace guard, the Archive gate | −10 | `temple_np` (institutional) | `cooperage_circle` |
| `cooperage_circle` | departed dissidents; remaining sympathizers (cooper's lad, glassblower, bookseller, gallery keeper, doubting novice) | +5 | — | `bloodline_domestic` |
| `bloom_trade` | Deren, Cade's successor, dock/canal contacts — hidden criminal | −20 (when known) | — | (wary: dockfolk, commonfolk) |
| `temple_np` | Yelin, Thane, canons, healers, scholars | 0 | `bloodline_domestic` (institutional, not all members) | — |
| `np_dockfolk` | dock community: foreman, keepers, warehouse, runner, pawnbroker | +10 | `np_commonfolk` | — |
| `np_commonfolk` | the Common populace; servants working up in Noble | +10 | `np_dockfolk` | — |

**Living-city note:** the bloodline's reach is *felt* daily — gate checkpoints,
tribute, the watched Noble streets — so `np_dockfolk`/`np_commonfolk` carry a
quiet resentment (gossip topic, dialogue flavor), while `temple_np` is publicly
aligned with the bloodline yet harbors doubters (the novice, a scholar). The
`cooperage_circle` exists mostly as recoverable lore + a few sympathizers.

---

## 4. Civic infrastructure (designed once; sited ON the arteries; where routines cross)

| Civic node | District | Artery position | Routine role |
|------------|----------|-----------------|--------------|
| Central Square great market | Merchant | the crossing | everyone shops; surveillance beat; the heart |
| Carter's Rise well + flower market | Common | Processional (Common stretch) | dawn water; the hollow-accusation scene |
| A neighborhood tavern (the Brimming Bowl) | Common | Processional | commonfolk evenings; conversation hub |
| A cookshop / street-food row | Common | Processional | commonfolk midday meals |
| The Salt Cellar (inn/tavern) | Docks | Long Market, west end | dockfolk evenings; transient lodging |
| A dockside cookshop | Docks | Long Market | dock midday meals |
| Aldric's public house (canon) | Docks | off the Long Market | rough lodging, no questions |
| The Gilt Threshold (high inn) | Merchant | near the Square | merchant-class social; Horst's quiet meetings |
| The Grand Temple | Temple | Long Market, east end | rest, **respawn**, worship; pilgrim arrival |
| A bathhouse | Common/Merchant seam | Processional | cross-class mixing |
| The fighting pit (stub) | Common | off the Processional | gambling/combat social (expansion stub) |
| A well per residential cluster | each | local | morning water; small-talk node |

Each is a node where NPC schedules converge (meal segments → cookshops, evening →
taverns, dawn → wells, rest-day → temple) and 3.6 conversations fire. **Respawn
point: the Grand Temple.**

---

## 5. Supply map (spatial, visible daily)

**Single import origin → distribution by mini-caravan along the Long Market.**

- The **Docks warehouse cluster + caravan depot** (run by the warehouse master,
  Dunmar Wells) is where *all* imported goods land and are stored.
- The **mini-caravan runner** (Dobb) loads at the depot and walks the **Long
  Market** east — Crafting → Central Square → Temple — branching south to Common,
  delivering to each district's vendors (the chunk-3.8 oneshot runner-delivery
  pattern; engine already ships). The import economy is something players *watch
  move through the streets*.
- **Every crafter anchor's `supply_source`** resolves to Dunmar's warehouse or
  the delivering caravan (e.g., Halvard's iron, Vesna's reagents, Corwin's hides,
  Nessa's cloth). Finished goods then move *between* artisans/vendors (Nessa's
  cloth → Aurel's Noble atelier; Halvard's blades → Ostry's Merchant shop).
- **Food/water:** cookshops source from the Central market; wells give water.
- **Engine wiring** (built during the Docks build, per the pricing-fix spec §2):
  add the NP zones to `CaravanServedZones`; register `np_docks_runner_circuit` +
  its delivery buckets (~6 lines); Dunmar = a Dock-Master merchant (not in
  `CaravanServedZones`) whose overstock the runner drains. Phase-2 routes (farm/
  timber/ore/forage via Kingsbarrow Vale / Kilnreach Works) layer in later.

---

## 6. The roster sketch (~45 anchors)

Format: **name · mutation (unique) · trade · home → workplace · faction ·
key relationships · [supply] · motivation.** Canon names marked *(canon)*. Full
life-sheets (24h schedule, dialogue, exact home/work rooms) authored per district
at build time. **The cross-district relationships and supply links are the point.**

### Docks (`np_dockfolk` unless noted) — import origin + dock community
1. **Jesset** *(canon)* · extra knuckle-joint per finger (canon) · dock foreman · home above the hiring office → the wharves · employs day-laborers; old friend of Bressa (Salt Cellar) · keep his crew fed and working.
2. **Bressa Toll** · skin flushes faint amber when she laughs · Salt Cellar keeper · above the Salt Cellar → the taproom · serves dockfolk; **buys ale from Renn Bowl's Common brewery** · keep the one warm room on the waterfront.
3. **Harbormaster Greer** · salt-crystal growth crusting his knuckles · harbormaster · harbor office → the quays · bloodline-licensed (tension); oversees Jesset; deals with Dunmar · keep the manifests clean enough to stay licensed.
4. **Dunmar Wells** · a third eye that never closes (always inventorying) · warehouse master / caravan dispatcher — **the supply hub** · above the warehouse → depot · **dispatches Dobb; supplies every district crafter** · never let the shelves run empty.
5. **Dobb** · tireless legs, never winded · mini-caravan runner · a depot bunk → the Long Market circuit · works for Dunmar; **visits every district vendor** (the moving thread) · finish the round before dark.
6. **Marn the Draper** · unnaturally clean, scentless · fabric-remnants front (Bloom successor) · back room → the shop · **`bloom_trade`**; supplied by Deren (Old Quarter); a dock contact · fill Cade's vacuum without being noticed. *(Bloom Trail breadcrumb.)*
7. **Old Sable** · silver coin-like scales on her palms · pawnbroker · above the shop → the counter · fences for the underdocks; the docks gossip node · know everyone's business, owe nothing.

### Common (`np_commonfolk`) — most homes, the working populace
8. **Tova** *(canon)* · faintly luminous freckles · landlady, rooming house by the old seminary · the rooming house · rents to transients; **holds "the group that left" lore** · keep her lodgers safe and quiet. *(Lore node.)*
9. **Coll the Sweeper** · can hear faint sound through stone · street-sweeper who knows everything · a tenement room → the streets · finds pre-Founding fragments (**Gritta** feeds him); supplies curiosities to Ysolde · miss nothing that passes. *(Quest: The Street Sweeper's Secret.)*
10. **Marda** · hands that never burn · cookshop keeper · above the cookshop → the cookpot · **sister of Halvard (Crafting smith)** — they eat midday together; sources from the Central market · feed the quarter cheap and well.
11. **Renn Bowl** · a voice that carries unnaturally · tavern keeper (the Brimming Bowl) · above the tavern · brews ale **sold on to Bressa's Salt Cellar**; the Common conversation hub · keep the room full and the talk flowing.
12. **Ysolde** · feels others' pain (empathic) · back-alley healer who doesn't ask questions · a back room · `bloom_trade`-adjacent (treats addicts); gets curiosities from Coll; buys salves from Vesna · mend who the temple won't.
13. **Garrick One-Hand** · a stone-hard right forearm · retired soldier, fighting-pit fixture · a flophouse → the pit gate · **old comrade of Guard-Captain Doryn (Palace)**; vouches for pit entry · one more good fight in him.
14. **Mother Brisk** · counts and remembers perfectly · tenement matron · her block (the condemned-upper-floors stub) · houses many ambient residents; wary of Clerk Vell's tax collection · keep a roof over her people.

### Crafting — artisan workplaces, the Inkwalk, cooperage traces
15. **Master Halvard** · skin that won't scorch · blacksmith · above the forge → the forge · **brother of Marda (Common)**; **supplies Ostry (Merchant weapons)** · [iron from Dunmar's warehouse] · forge the best steel on the coast.
16. **Toby the cooper's lad** *(canon role)* · steam-reddened permanent hands · tends the abandoned cooperage front · a cot in the cooperage · **`cooperage_circle` sympathizer — knows the basement** · keep the door shut and the memory alive. *(Cooperage-lore key.)*
17. **Vesna** · iridescent oil-sheen skin · alchemist · above the lab → the bench · **supplies Ysolde (Common) and, unknowingly, `bloom_trade`** · [reagents from warehouse + forager deliveries] · brew something no one else can.
18. **Edda Glass** · heat-scarred forearms, glass-clear fingernails · glassblower · home in Common → the workshop · `cooperage_circle` sympathizer; knew Asha; the kiln-complex stub · raise the kiln from the dead.
19. **Orin the bookseller** · ink-stained eyes that read in the dark · bookseller by Edvar's shuttered shop · above the shop · sells Edvar's last maps; `cooperage_circle`; **deals with Temple scholars (Dross, Novice Ept)** · find what Edvar was right about.
20. **Corwin the tanner** · leather-tough skin · leatherworker (tannery streets, Common seam) · above the tannery · [hides from warehouse] · the smell of honest work.
21. **Nessa the tailor** · uncannily nimble fingers · tailor · above the shop → the bench · **supplies Aurel (Noble clothier)** · [cloth from warehouse] · dress the city, top to bottom.

### Merchant — the heart, commerce, Horst's base
22. **Horst** *(canon, ANTAGONIST)* · none visible (uninfected) · bloodline domestic handler · rented house · **`bloodline_domestic`**; runs agents; deals with Clerk Vell · find what the bloodline sent him for. *(Live threat.)*
23. **Falk the auctioneer** · never forgets a face or a price · auction house · above the house → the block · fences high-end goods; knows property records (**Bloom Trail link**) · the gavel always falls his way.
24. **Goss the moneylender** · weighs gold true by touch · moneylender / the Exchange · the counting house · **holds debts on Common folk** (cross-district tension) · every ledger balanced in his favor.
25. **Dame Ostry** · an Extra Arm (mutation) · weapon dealer · above the shop → the floor · [blades from Halvard + warehouse] · handle every blade in the city.
26. **Brun the armorer** · plated, callused skin · armorer · above the shop → the floor · [plate from warehouse + Crafting] · no client of his ever falls.
27. **Clerk Vell** · a seal-shaped birthmark on his palm · bloodline permit-clerk · a Merchant office · `bloodline_domestic`; Horst's contact; **collects tribute (resented by commonfolk/Mother Brisk)** · process every permit by the letter.
28. **Madam Sephe** · a subtle charming glamour · the Gilt Threshold keeper · the inn · hosts merchants and **Horst's quiet meetings** · know every guest's secret.

### Temple (`temple_np`) — faith, respawn, the Archive
29. **Yelin** *(canon)* · hands worn smooth like prayer-stones · lay brother / warden, Keeper's House · the Keeper's House · 19+ years; gatekeeps visiting keepers · keep the house in order.
30. **Father Thane** *(canon)* · a voice that soothes the anxious · desk intake, Keeper's House · the desk · processes keepers; **the Crane pre-filed-cover node** · everything in its proper register.
31. **Canon Merid** · a faint dawn-prayer halo-glow · temple canon (blessings/regen) · the canon's cell · oversees Archive access; bloodline-aligned · keep the faith's machinery turning.
32. **Novice Ept** · sees the old orbital symbol everywhere · doubting novice · the seminary dormitory · **drawn to Orin's cooperage lore (cross-district)** · learn whether the story is true. *(Quest: The Doubting Novice.)*
33. **Scholar Dross** · an overlarge, veined cranium · skeptical scholar · the courtyard · argues the inscriptions' age; links to the gallery cipher · be right, and be heard.
34. **Sister Alms** · warm hands that mend · temple healer · the healer's chapel · the legit counterpart to Ysolde (Common) · heal the deserving and the not.
35. **Archivist Holt** · eyes that catalog at a glance · Restricted Collection gate · the archive door · **`bloodline_domestic` + `temple_np`** (the institutional intertwine) · let nothing out that shouldn't be. *(Deep-lore gate stub.)*

### Noble (`bloodline_domestic` unless noted) — elite, watched
36. **Steward Caldwin** · none (uninfected pride) · bloodline functionary · the Administrative Office · Horst's superior-side contact; Ferrol reports to him · the bloodline's will, executed cleanly.
37. **Wenna the servant** · flinches with a faint fear-light · nervous Noble servant · a servants' garret · **`np_commonfolk` working up in Noble**; **knows the Noble delivery-house secret (Bloom Trail)**; terrified of Caldwin · survive another day unnoticed.
38. **Guide Ferrol** · a rehearsed smile that never reaches the eyes · tour guide (approved history) · a Noble flat · recites Founding history; **his script vs. the old gallery art (cipher)** · never deviate from the text.
39. **Keeper Lysha** · color-true eyes (reads old pigments) · art-gallery keeper · above the gallery · the gallery cipher; quietly a `cooperage_circle` sympathizer · protect what the paintings really say. *(Quest: The Gallery Cipher.)*
40. **Porter Skell** · an immovable stance · liveried porter, gated estate lane · the lane gatehouse · turns all away (the gated-lane stub) · this lane is for residents only.
41. **Modiste Aurel** · fingers that read fabric quality blind · high-end clothier · the atelier · [cloth from Nessa, Crafting] · dress the bloodline and be remembered.
42. **Guard-Captain Doryn** · a parade-perfect physique · ceremonial palace guard · the gatehouse barracks · **old comrade of Garrick One-Hand (Common)**; the Palace gate stub · the Palace is not open to visitors. *(Endgame gate.)*

### Old Quarter (`bloom_trade` / poorest) — the buried canal city
43. **Deren** *(canon)* · veins glowing faint copper (Bloom exposure) · Bloom supplier, 215 Lintel St · the basement · **`bloom_trade`**; supplies Marn (Docks); **holds the captive** · keep the operation quiet and the product flowing. *(Bloom Trail climax.)*
44. **Quill the lamplighter** · night-adapted eyes · Old-Quarter lamplighter · a canal-side hovel · `np_commonfolk` (poorest); **witnessed Deren's traffic (Bloom Trail breadcrumb)** · light the lamps, see nothing, say less.
45. **Gritta** · senses the buried gray material · pre-Founding relic-scavenger · a flooded cellar · **feeds fragments to Coll (Common) and Orin (Crafting)** — the lore web · find what the city was built on.

---

## 7. The cross-district web (coherence check)

The roster is woven, not a flat list. The load-bearing threads:

- **Family/meals:** Marda (Common cookshop) ↔ Halvard (Crafting smith) — siblings who eat midday together (a daily cross-district routine).
- **The supply spine:** Dunmar's warehouse (Docks) → Dobb's caravan → Halvard/Vesna/Corwin/Nessa (Crafting) → onward (Nessa → Aurel/Noble; Halvard → Ostry/Merchant). One economy, visibly moving.
- **Drink supply:** Renn Bowl's Common brewery → Bressa's Salt Cellar (Docks).
- **The lore web (pre-Founding + cooperage):** Gritta (Old Q) → Coll (Common) + Orin (Crafting); Orin ↔ Novice Ept + Dross (Temple); cooperage sympathizers Toby, Edda, Lysha (Crafting/Noble).
- **The Bloom Trail:** Deren (Old Q) → Marn (Docks); Quill (Old Q) witness; Wenna (Noble) knows the delivery house; Ysolde (Common) treats the addicted.
- **The bloodline's reach:** Horst (Merchant) → Clerk Vell → Steward Caldwin (Noble) → Skell, Doryn, Holt — felt as tribute/checkpoints, resented by Mother Brisk / dockfolk.
- **Cross-class comrades:** Garrick One-Hand (Common pit) ↔ Guard-Captain Doryn (Palace).
- **Healers, mirrored:** Sister Alms (Temple, legit) vs. Ysolde (Common, no-questions).

These become 1.6 relationship edges + supply links + faction reps when each
district is built; they are why a smith lives in one quarter and works in another.

---

## 8. How district builds consume this layer

Each district (Docks-first: Docks → Common → Crafting → Merchant → Temple →
Noble → Old Quarter) gets its own spec → plan → build that:
1. Pulls its **anchors** from §6 and authors full life-sheets (home/work rooms,
   24h `schedule_id`, dialogue ≥3 topics, the unique mutation).
2. Lays out rooms within its **coordinate region** (§2), honoring the arteries +
   civic nodes (§4), `cartcheck`-clean; updates `docs/coordinate_map.md`.
3. Wires **relationships** (§7), **faction** membership (§3), and **supply
   links** (§5).
4. Adds **ambient + transient** residents (pooled cluster schedules) to fill the
   streets.
5. Authors its **questline** (master plan §8–9) + expansion stubs.
6. The **Docks build additionally** does the §5 engine wiring (CaravanServedZones
   + `np_docks_runner_circuit` + Dunmar as Dock-Master) and the Bloom Trail
   opening.

---

## 9. Open items / next step

- **Name-collision check:** verify the invented anchor names in §6 against the
  existing world mob roster before any are authored (no name recycling rule).
- **Bloom-as-mechanic vs. narrative** decision deferred to the Docks build.
- writing-plans → the **Docks district** plan is the first build artifact (it
  carries the supply engine wiring + the Bloom Trail opening).
