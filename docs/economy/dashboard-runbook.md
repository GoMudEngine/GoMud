# Economy Health Dashboard Runbook

URL: `/admin/economy/` (basic auth — same credentials as Combat Stats).

## What it shows

- **Five score cards** at the top: Economy (overall), Shops, Caravans,
  Foragers, plus the most recent snapshot timestamp and a "Snapshot Now"
  button. Scores are 0-100, colored red <40 / yellow 40-70 / green >70.
- **Per-discipline rollup** of shops grouped by `craft_support:` tag —
  one row each for blacksmithing, alchemy, tailoring, cooking,
  jewelcrafting, enchanting, and general. Then per-shop detail with
  stock bars colored by supply bucket (blue=base, cyan=stillwater,
  orange=thornwall, green=fernway, gray=overlap). The discipline
  rollup is the answer to "is the supply chain supporting player
  crafting?" — a low score on `alchemy` means alchemists are
  starving for materials.
- **Caravan + forager tables** with state, time-in-state, cargo bar
  (weight in pounds), per-instance score, and cycle counters.

## How snapshots work

The server writes a snapshot to
`_datafiles/economy/snapshots/{unix_ts}.yaml` every hour (config:
`Balance.EconomySnapshotIntervalHours`). The dashboard's "now" data is
captured live on each fetch — disk snapshots are only used for delta
columns and cycle counting.

Auto-snapshots are pruned past `Balance.EconomySnapshotRetentionDays`
(default 30). Manual snapshots (the "Snapshot Now" button) are never
pruned and are useful for "before/after" comparisons across config
changes.

The snapshot directory is gitignored — runtime state, not content.

## Cargo metrics: pounds, not item counts

Caravan and forager `cargo_weight` / `cargo_capacity` are both in
pounds (the wagon's `CarryCapacity()` and `GetCarriedWeight()` from
`internal/characters/inventory.go`). Carry weight is what actually
limits the wagon — the dashboard's "is the wagon filling up?"
question reads honestly as a weight ratio. Per-bucket cargo sums are
also weights, not item counts.

## Score formulas

- **Per-shop:** weighted average of `Current/Max` per stock entry,
  weighted by `RestockQty` so high-throughput items dominate.
  Returns "—" (no score) if the shop has no stock entries.
- **Per-discipline:** mean of all per-shop scores in that
  `craft_support` bucket.
- **Per-caravan / per-forager:** cycle-rate score over the last 168
  snapshots (7 days at 1h cadence). 100 = expected cadence (1 cycle/
  day for caravans, ~3/day for foragers). A 30-point stuck penalty
  fires if the entity has been in its current state longer than
  5000 rounds. Returns "—" for the first 24 hours after server boot
  (insufficient history).
- **Overall:** weighted mean — shops 0.6 / caravans 0.2 / foragers
  0.2. Renormalizes when one component has insufficient history.

## Troubleshooting

- **Scores show "—":** insufficient history. Caravan/forager scores
  need at least 24 snapshot entries (one full day). Shop scores need
  no history; if shop scores are "—" check that
  `internal/economy/health/captureShops()` is finding shops at all
  (`shops.AllShops()` returns the cache).

- **Empty caravan/forager tables:** confirm the entities are alive.
  Caravan leader is identified by `BTreeState["caravan_state"] != ""`,
  forager by `BTreeState["forager_state"] != ""`. The flavor caravan
  master at North Road is correctly excluded (no btree state).

- **Shop discipline shows "(uncategorized)":** the startup validator
  should have prevented this. If it slipped through, add
  `craft_support:` to the mob YAML and restart.

- **Server panics at boot with `shops.ValidateShopMobTags failed`:**
  the panic message lists every shop-bearing mob missing its
  `craft_support:` tag. Add the tag to each listed mob YAML and
  restart. Use `general` for mixed-stock merchants, otherwise pick
  the matching crafting discipline.

- **Empty snapshot directory:** snapshots are written hourly. On a
  fresh boot it'll take an hour for the first auto-snapshot. Use
  "Snapshot Now" to seed one immediately.

## Adding a new vendor discipline

The `ValidCraftSupports` list mirrors the player crafting skills in
`internal/skills/skills.go`. If a new player-facing crafting skill is
added there:

1. Add a matching constant to `internal/shops/shopinventory.go`
   (`CraftSupport<Foo>`) and append to `ValidCraftSupports`.
2. Tag the relevant mobs with `craft_support: foo`.
3. Restart server — dashboard rolls up the new discipline
   automatically.

If you just want to subdivide an existing discipline (e.g. split
`tailoring` into `cloth` + `leather`), prefer keeping the existing
tag and using the per-shop bucket bar to read the difference.

## Known followups

Tracked in `MEMORY.md` under "Economy dashboard followups":

1. `ListSnapshotsFrom` full-parses every YAML for a 2-key peek —
   ~70MB of allocations per dashboard fetch at full retention. Add a
   `manualPeek` struct.
2. `/reload` swallows the `craft_support` validator panic via the
   existing recover. Cold boot still panics correctly.
3. `territoryFor` switch silently degrades on unknown ForagerKind.
4. Cargo capture misses `ComponentItems` and `PotionItems` (irrelevant
   today since wagons/foragers don't equip those slots).
