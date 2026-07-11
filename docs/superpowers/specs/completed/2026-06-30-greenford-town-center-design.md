# Greenford — District 2: Town Center — Design

*Spec date: 2026-06-30. District 2 of 5 (city-wide layer:
`docs/superpowers/specs/2026-06-30-greenford-citywide-design.md`; District 1
River District merged `6b162857`). The civic hub of the university town — small
and quiet (a deliberate contrast to the Confluence's grand plaza). Seeds ONE
Q75 breadcrumb (the bookseller's "surveyor who retired early"); the quest itself
hubs later at Brennan (District 3/4). NO orbital-symbol content here (reserved
for the University district).*

## Role

Up the Town Stair from the riverfront: the everyday heart of Greenford — a
modest market square, a bookshop, the Cartographer's Rest inn, a general store,
a civic building — with the university glimpsed higher up the hill and a stair
continuing up to it (District 3, a stub). Bookish, comfortable, a town that
"thinks for a living" but at street level is just people getting on. The
**bookseller** plants the first thread of the Surveyor's Report (Reth, "the
surveyor who retired early"); a **student/young scholar** gives a lore-light
forward-gesture to the university. Everything else is warm civic texture.

- **Folder:** `greenford` (same zone). **Rooms:** 6288–6297 (10). **Mobs/
  dialogue:** 9509–9515 (7). **Items:** 40155–40158.
- **No quest grant** (Q75 hubs in District 3/4). **No new faction** — townsfolk
  are `[humanoid]`; the Margin proper appears at the University (District 3).

## Geography & Seam

- **Seam:** District 1's **6287 "The Town Stair"** `{x:22,y:-76,z:0}` (its
  "stair rising into the town" prose; currently exits `north→6284, west→6286`,
  no onward). Add **`6287 up → 6288`** (climbing the stone treads to the upper
  town) and lightly revise 6287's prose so the way up is now open/walkable
  (the stair climbs to the market square above). 6288 sits on **z=1** (the upper
  town above the riverfront) — this carries the "town is up the hill" feel and
  keeps the riverfront and town on clean separate elevation layers.
- **Suggested layout** (builder finalizes a clean reciprocal, collision-free
  graph on **z=1** around {22,−76,1}; re-run `cartcheck`/boot consistency — the
  z=1 plane there is clear of existing content):

| Room | Title | role |
|------|-------|------|
| 6288 | The Market Square | top of the stair (down→6287); the civic heart, small + quiet |
| 6289 | The Market Cross | the square's second beat (stalls, a notice board, the town crier) |
| 6290 | The Bookshop | the **bookseller** + the Q75 breadcrumb |
| 6291 | The Cartographer's Rest | the inn common room (innkeeper, cooking vendor) |
| 6292 | The Cartographer's Rest, Upstairs | inn lodging (a z+1/`up` or side room — the rest-stop) |
| 6293 | The General Store | the general-store keeper (general-`craft_support` vendor) |
| 6294 | The Guild Lane | a quiet civic street (a scribe, a chandler — texture) |
| 6295 | The Town Hall steps | a civic building exterior (records/clerk — light) |
| 6296 | The Upper Street | the street climbing toward the university |
| 6297 | The University Stair | **stub** up to the University district (District 3) — described, NOT wired |

(Reciprocal exits; the square (6288/6289) is the hub; the shops + inn open off
it; 6296→6297 climbs toward the university. Use a `up`/z-step for the inn
upstairs and the university stair if it reads better, keeping coords clean.)

## NPCs (mobs 9509–9515: 7)

Canonical Title-Case names, `ConvertForFilename` filenames, ambient archetype
`noncombat_passive`, unique visible mutations, ≥3 dialogue topics, idle
behaviors, voice rules (NPC text 1st-person, hints 2nd-person, **every hint word
must route to its node** — cross-check; this bit District 1), NO quest fields,
`groups: [humanoid]` (no Margin here).

| mob | room | role |
|-----|------|------|
| 9509 (named) The Bookseller | 6290 | dialogue; local lore + books; **plants the Q75 breadcrumb** — when asked about surveyors / the eastern country / "the hill", mentions "the surveyor who retired early" (Reth) and that Brennan at the university "asks the same sorts of questions." Lore-light, NO crash-site answer, NO orbital symbol. |
| 9510 (named) The Innkeeper | 6291 | **cooking vendor** (Cartographer's Rest — food/drink); travelers, the road, the town |
| 9511 (named) The Store-Keeper | 6293 | **general `craft_support` vendor** (mixed goods — stock a few real-discipline items as the catch-all per the general-store gotcha); daily-trade talk |
| 9512 A Produce-Seller | 6289 | market ambient (could be a small cooking vendor or ambient); square color |
| 9513 A Town Crier | 6289 | ambient; civic notices, the day's news (light, no mystery) |
| 9514 A Student | 6296 | dialogue; a young scholar — lore-light forward-gesture to the university uphill (what's studied, why people come), NEVER mystery/crash-site; the soft pull toward District 3 |
| 9515 A Townsperson | 6294/6295 | ambient; civic daily-life |

The bookseller (9509) is the one Q75-loaded NPC; the student (9514) is the soft
university gesture. Keep the rest mundane.

## Economy & items (40155–40158)

- Inn cooking goods + general-store mixed goods + a market produce item. Each
  salable item carries a REAL discipline `vendor_categories` (never `general`);
  the general store's `craft_support: general` is the catch-all VENDOR, but its
  stocked items each carry a real discipline (tailoring/cooking/etc.) — per the
  shipped general-store pattern (see the economy-depth gotcha). Reuse existing
  goods where they fit. A book/map flavor item is optional (if non-vendor, mark
  `not_salable: true`).
- **No new forageables** (town center; reuse nothing needed).

## Mystery / lore boundary

ONLY the **Q75 breadcrumb** (the bookseller's "surveyor who retired early" +
the pointer to Brennan). **NO orbital-symbol content, no pre-Founding artifact,
no numerology** — that's the University district (3) and beyond. (District 1's
review caught an accidental symbol bleed in a folk stone — guard against it: any
"old carving"/map detail here stays mundane.)

## Terminus stub

**6297 "The University Stair"** — the street reaches a stair/gate climbing to the
university quarter (lecture halls, the library, faculty), described in prose
(the tower close now), but the onward exit is **NOT wired** (District 3 attaches
here). Softer "just up there, coming soon," like the District 1 Town Stair.

## Build conventions & validation

Carry the Greenford city-wide convention/gotcha list (folder `greenford`;
Title-Case; colon/`>`-block; no `kind:`; vendor categories never `general` on
items; node-shadowing — gated/specific nodes first + cross-check every hint word
routes; **long NPC text fields use `|` literal block scalars** — District 1's
truncation bug; stage explicit git pathspecs never `-A`). Per-district SOP:
`id_inventory` → author → wipe instances → clean boot (`ValidateZoneConsistency
errors=0 mode=panic`) → `cartcheck greenford` → **world-critic + harness
feel-test** (the seam from 6287 up, the square + all shops/inn, vendor buys, the
bookseller breadcrumb + student gesture, the University stair stub) → update
`docs/ZONE_EXPANSION.md` row 19 (2/5) + memory → merge `--no-ff`.

## Out of scope

Q75 grant/wiring (District 3/4), Margin NPCs (District 3), the symbol/university
content (District 3), Brennan & Reth (Districts 3/4).
