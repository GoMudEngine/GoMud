# Cascade Pass Road (#20) — design

*Date: 2026-07-01. Phase 7 — The Eastern Road (Endgame Approach), leg 1.*
*Canonical plan: `docs/ZONE_EXPANSION.md` §Zone 7.1.*

## Purpose

The first leg of the endgame arc: a hidden road that leaves the settled
country just south of New Plymouth and climbs east — forest, then a
mountain pass — toward the broken highland country where the buried hull
lies. It is a **lore-and-ambient connector** (no quest), doing three jobs:

1. **Gate the endgame subtly.** The turn-off is easy to miss; only a
   player who is *looking* (or who carries Reth's marked map from
   Greenford's Quest 75) finds it. This keeps newbies out of endgame
   country by design.
2. **Foreshadow the trapped, lethal ship** without ever naming it —
   through environment and one memorable returned-survivor NPC.
3. **Commit to solo-endgame combat** from just past the gate, doubling as
   a newbie-repellent: anyone who stumbles through gets wrecked and reads
   the message *"turn back if you're not ready."*

Only the **ship interior (#22)** is true party endgame. Cascade Pass and
the Eastern Highlands approach are tuned as **solo endgame**.

## Geography & the hidden branch

New Plymouth is a walled capital with exactly one open road out, running
**south** through Kingsbarrow Vale (farmland) → Kilnreach → Greywater.
Cascade Pass branches **east off Kingsbarrow Vale room 5441 "North Vale
Road"** (`farmland`, x−18 / y61 / z0), into currently-unclaimed grid to
the east.

- **The turn-off is a `secret: true` east exit** on 5441. Room 5441
  already describes an open **farm-gate cut into a head-high thorn hedge**
  (for cart traffic). We keep that mundane reading on `look`, add a
  `hidden_noun` (worn game-trail / old cart-ruts running the *wrong* way
  past the gate / disturbed ground) that a `search` reveals with the
  passage hint, and hang the secret east exit behind it. Newbies read
  "farm access" and move on; a searcher — or a player following Reth's
  directions — finds the road.
- **Reciprocity:** 5441 `east` (secret) ↔ 6323 `west`, both carrying the
  cross-zone `zone:` annotation (Kingsbarrow Vale ↔ Cascade Pass Road).
  5441 keeps all its existing exits; we only add the hidden east.
- The new zone occupies the eastern grid (x ≥ −17 at y ≈ 61), climbing
  east and then upward. Cartesian-clean; east = increasing x, the pass
  section climbs via `up`/z-steps (the z convention proven in Greenford).

## Layout — 20 rooms (6323–6342), two mini-stages

### Stage 7.1a — the forest road (10 rm, 6323–6332)
`farmland` edge → `forest`, the road narrowing and climbing east.

- The hedge-gap trail past the gate; the fields falling away behind.
- The road entering old timber — trees larger and older the further east.
- **A lumber camp** — still worked, but the crews only cut *west* now;
  nobody takes the good timber east anymore, and they won't say why in so
  many words. (Ambient NPC + the first oblique "east is wrong" note.)
- **A ruined waypoint** — a travellers' rest abandoned years ago; the
  road east of it barely used, reclaimed by the forest.
- The sense of entering country that does not want visitors: predator
  sign, a game-trail that avoids the eastern slope, the quiet getting
  wrong.

### Stage 7.1b — the pass (10 rm, 6333–6342)
`forest` → `mountains` → `cliffs`, climbing via `up`/z-steps.

- The **tree line** — the timber thinning, then stopping.
- Exposed rock, switchbacks, **views in both directions** (the settled
  country behind and below; broken highland ahead).
- **A crumbling watchtower** — old, pre-anyone's-memory, its sightline
  fixed east on the highlands, not back toward the road.
- **The returned survivor's shelter** near the high point — the zone's
  emotional and warning centerpiece (see NPCs).
- **The survey marker** — Reth's territory begins; the understated
  orbital-symbol beat (see below).
- The plateau edge: the terrain breaking into the highland country
  beyond. Thinner, colder air; "something old underneath it." **Terminus
  stub east** — framed as *the country ahead, not yet crossable* (must
  NOT invite `go east` as a wall-bump), toward the unbuilt **Eastern
  Highlands (#21)**.

## Difficulty — solo endgame, base + tough

`statpool` on an overworld mob is applied **directly** (the instance
gold-scaling does not apply here). Calibrated to the Elemental Oasis
yardstick (base = ~275-gold difficulty, tough = 2×):

| Tier | `statpool` | Examples |
|------|-----------|----------|
| base predator | **275** | a lone hunter-wolf, a territorial highland cat, a rangy boar — `archetype: fighting` |
| tough / apex | **550** | a pack alpha; the thing that came down from the pass — the road's hardest mob |

No true boss on the road (bosses reserved for Highlands #21 and interior
#22). The first forest rooms carry the **base** tier so a prepared solo
player can gauge the threat and retreat before the pass commits to
tough-tier apex predators. Ambient fauna and NPCs are non-hostile flavor.
HP/fight-length gets a tuning check at feel-test (high `statpool` fighting
mobs accrue Vitality → large HP pools).

## NPCs (mobs 9535+)

Each mob carries a distinct Chrysalis mutation (world convention); names
in canonical `internal/casing.Title()` form; filenames
`mobid-ConvertForFilename(name).yaml`.

- **The returned survivor** (non-combatant hermit near the pass) — the
  centerpiece. Came back from the east maimed (one hand; won't speak of
  the rest) and does not go down again. Warns **obliquely**, never naming
  what is out there: the ground opening, walls that move, a place that
  *"doesn't want to be opened."* This is where the trapped/lethal-ship
  foreshadowing lands hardest. Dialogue is threshold-only — dread, not
  disclosure. No quest, no reward; a person, not a mechanism.
- **Lumber-camp ambient** — a foreman / last logger who works the western
  cut and won't take the eastern timber; delivers the first "east is
  wrong" note in working-man's terms. Optional day/night anchor schedule.
- **Forest predators** (combat, base 275) — 2–3 territorial hunters.
- **Pack alpha / pass apex** (combat, tough 550) — the road's hardest
  fight; carries loot.
- **Highland fauna** (flavor / light combat) at the pass — birds that do
  not nest past the tree line, etc.; reinforces "the landscape knows."

## The symbol beat

One understated orbital-symbol surface: **an old survey marker or cairn**
where Reth's survey territory begins, scored with the recurring **nested
rings**. `look`able noun; **UNEXPLAINED, threshold-only** — never
interpreted, no numerology, no NPC decodes it ("no one knows why it's
here" at most). Ramps symbol density as we near the hull, consistent with
East Road's Old Waystone. Guard against symbol bleed elsewhere in the zone
(the D1-Greenford lesson): exactly one surface carries it.

## Items (40163+)

Kept light; reuse the existing palette where possible.

- Predator loot (a pelt / a fang) on the combat mobs, itemdropchance set
  so kills actually yield.
- Optional forest/mountain forageables (mountain herb, high moss) — only
  if we add them, wiring `forest`/`mountains` ForageYields TDD-style
  (`internal/forager/forage_core.go`) as East Road did for dry country.
  Default: skip unless it adds felt value; reuse existing forageables.
- No quest items. The survivor and the symbol are lookable nouns /
  dialogue, not deliverables.

## What this zone deliberately is NOT

- **No quest.** (Warning rides on the survivor + environment.)
- **No disc-door / no hull.** Those are Highlands #21c and interior #22.
- **No mystery revelation.** Symbol and survivor stay at the threshold.
- **No easy access.** The secret exit + solo-endgame combat are the gate.

## Build method & known gotchas (banked from East Road / Greenford)

- Full **spec → plan → subagent-driven-development**; two-stage review per
  task; a world-critic integration pass + a mudagent feel-test at the end.
- Zone folder MUST equal `ConvertForFilename("Cascade Pass Road")` =
  `cascade_pass_road`; boot panics otherwise. Each zone needs a
  `zone-config.yaml` (name/roomid/defaultbiome/region — region "The
  Eastern Reach" or similar; defaultbiome `forest` or `land`).
- **Stage explicit git pathspecs, never `git add -A`** (dirty repo).
- Dialogue: **no colon-space `": "` in prose** (YAML key → panic; use
  em-dash); **no semicolons in NPC `text:`/`hints:`** (command separator —
  drops the rest); **`|` literal block scalars for all long NPC text**
  (≥~120 chars — double-quoted flow scalars truncate in-game); **place
  gated grant/lore nodes first** (short triggers substring-match topics);
  **`room_interact` fires on `examine`/`look`, not `take`/`get`** — write
  hints as "examine the X"; **hinted multi-word nouns need a hyphenated
  token+key matching the hint**.
- **`biome: wilderness` is invalid** — use `land`/`forest`/`mountains`/
  `cliffs` (all confirmed valid in-world).
- Cross-check mutations against the WHOLE zone roster (agents only see
  their own batch).
- **Terminus stubs must not invite `go <onward>`** — frame as not-yet.
- Boot test: `ValidateZoneConsistency errors=0` (mode=panic),
  `loadAllRoomZones`, mobs/quests loadedCount clean, 0 panics. To
  feel-test, point the admin smoketester (user 17.yaml) at 5441 (or 6323)
  before boot; nuke `rooms.instances/*` + `mobs.instances/*` first.

## Attach summary (the one edit outside the new zone)

`_datafiles/world/dogmud/rooms/kingsbarrow_vale/5441.yaml`: add a
`secret: true` east exit → 6323 (`zone: Cascade Pass Road`) + a
`hidden_noun` revealing the eastward trail. No other Kingsbarrow change.
