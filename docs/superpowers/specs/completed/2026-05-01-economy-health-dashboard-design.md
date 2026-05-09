# Economy Health Dashboard — Design

**Date:** 2026-05-01
**Status:** Draft (awaiting user review)
**Surface:** Web admin portal (`/admin/economy/`)

## Purpose

Give the admin a one-screen answer to systemic questions about the
NPC supply chain:

1. Are shopkeepers getting their materials topped off enough?
2. Are foragers delivering goods in the right proportions to support
   player crafting?
3. Does the NPC economy churn while no players are online (e.g.
   overnight)?
4. Did the changes I just made improve or harm overall economy health?

The dashboard is a **balance/regression monitoring tool**, not a live
debugging tool. It surfaces drift over time so we can catch regressions
before they ship to prod.

## Scope

**In scope:**
- New web admin dashboard at `/admin/economy/` matching the existing
  Combat Stats / Progression dashboard pattern.
- Hourly wall-clock snapshot capture of shop, caravan, and forager
  state to disk.
- Manual "snapshot now" capture for ad-hoc before/after comparisons.
- Health scoring per shop, per shop archetype, per caravan, per
  forager, plus an overall economy score.
- Delta columns at 1h / 6h / 1d / 3d / 1w from the closest historical
  snapshot.

**Out of scope (deferred):**
- Event-stream instrumentation (per-transfer event logs). Snapshots
  alone answer the questions above; event tracking is a future
  upgrade if/when sub-hour granularity becomes important.
- Sparkline / time-series charts. Tables + bars cover MVP needs.
- In-game admin command. The user has zero interest in a text
  surface.

## Architecture

Three pieces, mirroring the Combat Stats / Progression pattern.

### `internal/economy/health/` (new package)

Owns snapshot capture, persistence, and scoring. Public surface:

- `CaptureSnapshot() Snapshot` — walks `shops.shopCache`, all caravan
  leaders (mobs whose `BTreeState.GetString("caravan_state") != ""`),
  and all foragers (`forager_state` analog). Returns a populated
  `Snapshot` struct.
- `WriteSnapshot(s Snapshot) error` — serializes to YAML at
  `_datafiles/economy/snapshots/{unix-ts}.yaml`.
- `LoadSnapshot(ts int64) (*Snapshot, error)` — read from disk.
- `ListSnapshots() []SnapshotMeta` — directory listing, sorted by
  timestamp descending. Lightweight (filename + `manual` flag only,
  no full parse). `SnapshotMeta` is `{UnixTs int64, Manual bool,
  ManualLabel string}`.
- `Score(s *Snapshot, history []*Snapshot) Scores` — computes per-
  shop, per-archetype, per-caravan, per-forager, and overall scores.
  `history` is the caller-loaded set of recent snapshots needed for
  cycle-counting (typically the last 168 hourly snapshots). The
  caller decides how many to load; `Score` walks them to count
  state-transitions.
- `PruneSnapshots(retentionDays int)` — deletes auto-snapshots older
  than retention. Manual snapshots (`manual: true`) never pruned.

A wall-clock ticker started from `main.go` — next to the other server
tickers — calls `WriteSnapshot(CaptureSnapshot())` every hour and
runs `PruneSnapshots()` daily.

### `internal/web/admin.economyhealth.go` (new)

Three handlers, same shape as `admin.combatstats.go`:

- `combatStatsIndex` analog: `economyIndex` — renders header +
  `economy/index.html` + footer.
- `economyAPI` (GET `/admin/api/economy/`) — returns JSON containing:
  - The current live snapshot (capture-on-demand, not the most recent
    on disk — gives the admin freshest data).
  - The five comparison snapshots (closest to now-1h, -6h, -1d, -3d,
    -1w) loaded from disk.
  - Computed `Scores` for the live snapshot.
  - Computed deltas per (shop, comparison-snapshot) and per (caravan,
    comparison-snapshot) and per (forager, comparison-snapshot).
- `economySnapshotAPI` (POST `/admin/api/economy/snapshot`) — manual
  snapshot trigger. Accepts optional `?label=...` query param;
  written into the snapshot's `manual_label` field.

Wired into `internal/web/web.go` next to the Combat Stats /
Progression routes.

### `_datafiles/html/admin/economy/index.html` (new)

Bootstrap 4 + vanilla JS poll, identical scaffolding to
`admin/combatstats/index.html`. Layout described in §6.

A new sidebar entry is added to `_datafiles/html/admin/_header.html`:
`<a ... href="/admin/economy/">Economy</a>`.

## Snapshot Storage

**Path:** `_datafiles/economy/snapshots/{unix-ts}.yaml`

**Gitignored.** Per "no instance saves to prod" feedback — runtime
state, not content. Daily prod state would diverge from dev state
immediately.

**Retention:** auto-snapshots pruned past `EconomySnapshotRetentionDays`
(default 30 = ~720 hourly files, ~25 MB). Manual snapshots kept
forever.

**Size budget:** ~30-50 KB per file given current zone count
(~15 shops, 1 caravan, 3 foragers). 720 files ≈ 25 MB.

**Config knobs** added to `_datafiles/config.yaml` under `Balance`:

```yaml
EconomySnapshotIntervalHours: 1
EconomySnapshotRetentionDays: 30
EconomyScoreWeightShop: 0.6
EconomyScoreWeightCaravan: 0.2
EconomyScoreWeightForager: 0.2
```

## Snapshot Schema

Flat YAML, capturing only what the dashboard needs (no full item
state). Fields mirror existing engine types but trimmed.

```yaml
timestamp: 2026-05-01T13:00:00Z
unix_ts: 1746104400
round: 12345
manual: false                  # true for "Snapshot Now" captures
manual_label: ""               # optional label, e.g. "pre stage-3.4"

shops:
  - zone: stillwater
    mob_id: 341
    room_id: 4105
    name: "Storekeeper Wulf"
    archetype: general          # general | smith | inn | alchemist | ...
    gold: 487
    starting_gold: 500
    last_restock_round: 12000
    stock:
      - item_id: 40001
        bucket: base             # from economy.BucketFor()
        current: 8
        max: 20
        restock_qty: 5

caravans:
  - inst_id: 42
    name: "Caravan Master Borric"
    state: outbound_transit
    state_entered_round: 12100
    room_id: 1500
    cargo_count: 14
    cargo_capacity: 30
    cargo_by_bucket: {base: 5, stillwater: 9}

foragers:
  - inst_id: 88
    name: "Storekeeper Wulf"
    territory: stillwater_marsh
    state: foraging
    state_entered_round: 12200
    room_id: 4520
    cargo_count: 6
    cargo_capacity: 12
    cargo_by_bucket: {stillwater: 6}
```

### Schema dependencies

Two pre-existing fields need to be added before snapshots can carry
the data described above:

1. **Shop `archetype` field.** Add `Archetype string` to
   `shops.ShopInventory` (YAML key `archetype`). Backfill all existing
   shop YAMLs in `_datafiles/world/dogmud/shops/` (~15 files) and
   ensure new shops set it on registration. Allowed values:
   `general`, `smith`, `inn`, `alchemist` (extend as new vendor types
   appear). The field is stored on the persisted ShopInventory because
   it's a property of the shop (not the mob template).

2. **State-entry round tracking.** Caravan and forager btree actions
   already write `caravan_state` / `forager_state` strings on every
   transition. Add a sibling write of `caravan_state_round` /
   `forager_state_round` (uint64, current round) on transition.
   ~5 LoC each. Reads default to 0 when absent (interpreted as
   "unknown — not stuck").

Both changes are tiny and ship as part of this work.

## Health Scoring

All scores 0-100. Color thresholds: red <40, yellow 40-70, green >70.

### Per-shop score

Weighted average across stocked items. Weight by `restock_qty` so
high-throughput items dominate (rare items shouldn't drag the score
down disproportionately).

```
itemFill   = clamp(current / max, 0, 1)
itemWeight = max(restock_qty, 1)
shopScore  = 100 × Σ(itemWeight × itemFill) / Σ(itemWeight)
```

Shops with no stock entries report `nil` (rendered as "—" in UI),
not zero.

### Per-archetype score

Straight mean of all per-shop scores within that archetype.

### Caravan score

Composite — fullness alone is a sawtooth, not a health signal:

```
cycleScore   = clamp(100 × completedCyclesLast7d / expectedCycles, 0, 100)
stuckPenalty = 30 if currentRound - state_entered_round > 2 × expectedDuration(currentState) else 0
caravanScore = max(0, cycleScore - stuckPenalty)
```

Completed cycles are derived from snapshot history: count
`ThornwallDwell → ThornwallDwell` transitions across the last 168
hourly snapshots (full week of data). `expectedCycles` = 7 days ÷
configured cycle length.

`expectedDuration(state)` is per-state, not a single global value.
Dwell states have a long expected duration (the dwell timer); transit
and route states have shorter expected durations (path length). The
expected-duration table is read from existing caravan pacing config —
we don't introduce a new tuning surface.

### Forager score

Same shape as caravan, counting `Resting → Resting` transitions.
`expectedDuration(state)` reads from the existing forager pacing
config (per-state, same convention as caravan).

### Overall economy score (headline gauge)

```
overall = 0.6 × meanShopScore
        + 0.2 × meanCaravanScore
        + 0.2 × meanForagerScore
```

Weights live in config (see §"Snapshot Storage" above). Shops
weighted heaviest because they are the player-facing terminus —
caravans and foragers exist to serve them.

### Insufficient-history handling

Cycle-counting needs a baseline of snapshots. When
`len(history) < 24` (i.e. <24h of hourly data), caravan and forager
scores render as "—" with tooltip "needs N more snapshots". The
overall score falls back to weight-renormalized mean of available
scores. Per-shop and per-archetype scores compute fine from a single
snapshot.

## Dashboard Layout

Top to bottom on `/admin/economy/`:

### Header strip — five summary cards

Mirror Combat Stats. Each card is a big colored number 0-100 with
label.

- Economy Health (overall)
- Shop Health (mean across all shops)
- Caravan Health
- Forager Health
- Last Snapshot (timestamp + "Snapshot Now" button + optional label
  input)

### Auto-refresh controls

10s / 30s / 60s / 2m, identical to Combat Stats.

### Section A — Shopkeepers

Archetype rollup table at top. One row per archetype:

| Archetype | Shops | Score | Total Gold | Δ1h | Δ6h | Δ1d | Δ3d | Δ1w |
|-----------|-------|-------|------------|-----|-----|-----|-----|-----|
| Smith | 3 | [▓▓▓▓░░] 71% | 1,420 | -12 | -45 | -120 | +200 | +800 |

Click a row → expand into per-shop subrows. Each per-shop row shows:
- Name, room, score bar, total gold, Δ-gold for each of the five
  comparison points.
- **Stacked bucket-fill bar** — current/max stock summed per bucket
  (`base`, `stillwater`, `thornwall`, `fernway`, `overlap`),
  color-coded by bucket. Width = bucket fill %, color = bucket
  identity.
- Click → modal with full per-item stock list (current/max bar per
  item, last-restock round translated to approximate real time).

### Section B — Caravans

One row per caravan-leader instance. Columns:

- Name, score bar
- State badge (`dwelling` | `transiting` | `routing`), colored by
  category
- Time-in-state (red if exceeds tolerance)
- Current room
- **Cargo bar:** stacked-bucket bar of `cargo_by_bucket`,
  `cargo_count / cargo_capacity` fill
- **Cycles in last 1h / 6h / 1d / 3d / 1w** — derived from snapshot
  history. Cargo deltas are intentionally omitted: cargo is a
  sawtooth (fills, empties, fills) so a 1h delta carries no signal.
  Cycle counts are the meaningful "is the supply chain churning?"
  metric.

### Section C — Foragers

Same shape as Section B. Per-row: name, territory, state badge,
time-in-state, room, cargo bar, cycle-counter columns at 1h / 6h /
1d / 3d / 1w.

### Visual conventions

- Bucket colors fixed across the dashboard (e.g. base=blue,
  stillwater=cyan, thornwall=orange, fernway=green, overlap=gray)
  so a glance at any bar reveals composition consistently.
- Δ cells use ↑/↓ arrow + magnitude. Color: green if change is
  favorable for that metric (gold up = good, stock up = good,
  time-in-state up past tolerance = bad), red otherwise.
- "Insufficient history" rendered as "—" with tooltip. Never zero.
- No charts in MVP — tables + bars only. Chart.js is loaded by
  `_header.html` so future sparklines are a drop-in upgrade.

## Edge Cases

**Server clock vs in-game round.** Snapshots timestamped by wall
clock; state-entered values use round numbers. The dashboard
converts round → approximate-real-time using a rolling round-rate
estimate from recent snapshots (Δrounds ÷ Δseconds). Good enough for
"stuck for ~3 hours" labels.

**Snapshot capture during shop mutation.** `shopCache` is
RWMutex-protected; `CaptureSnapshot()` takes the read lock for the
duration of iteration. `mobs.GetAllMobInstanceIds()` is similarly
safe. Hourly cadence makes lock-hold time a non-issue.

**First-boot empty state.** No snapshot history → no deltas, no
cycle counts. Render "—" with explanatory tooltips. Per-shop scores
still compute and remain useful.

**Missing-archetype shops.** A shop with `archetype: ""` rolls into
a synthetic "uncategorized" archetype row. Loud yellow tooltip:
"Set archetype: in this shop's YAML to categorize." Backfill task
captures all known shops at design-time, so this should never trip
in practice — but it's better to surface than silently hide.

**Caravan/forager identification correctness.** Reuses the existing
discriminator `BTreeState.GetString("caravan_state") != ""` /
`forager_state != ""`. The flavor-only "caravan master" NPC on the
north road is correctly excluded because he has no btree state.
Same convention as the existing `caravan reset` admin command.

**Snapshot file directory creation.** Server creates
`_datafiles/economy/snapshots/` on first capture if absent. Path
is gitignored so it never appears in the repo.

**Cycle-counting approximation.** Hourly samples can miss complete
cycles that finish in <1h (the state machine moves through 8 phases
between snapshots). Current pacing makes a full caravan cycle take
many hours, so this is fine. If pacing tightens later, we'd need
event-stream tracking — a flagged future upgrade.

## Testing

Unit tests in `internal/economy/health/`:

- `health_test.go` — score formulas (per-shop weighting,
  archetype mean, overall weighting, insufficient-history fallback).
- `snapshot_test.go` — capture from a fixture `shopCache` + mob
  state, round-trip through YAML write/read, prune correctness.
- `delta_test.go` — comparison-snapshot picker (closest-to-target
  with tolerance), delta computation per metric.

No integration tests for the web handler — matches the existing
Combat Stats / Progression precedent of relying on unit tests for
the underlying package and manual smoke for the HTTP surface.

## Migration / Backfill

One small content task as a prerequisite:
- Add `archetype:` field to all ~15 existing shop YAMLs in
  `_datafiles/world/dogmud/shops/`. Inspect each shop's role and
  pick from the allowed list (`general`, `smith`, `inn`,
  `alchemist`). Done as part of the implementation, not a
  separate ticket.

No database migration; no save-file format changes beyond the
shop YAML addition.

## Open Questions

None — all forks resolved during brainstorming:
- Surface: web dashboard (decided)
- Time-series mechanism: hourly disk snapshots (decided)
- Comparison points: 1h / 6h / 1d / 3d / 1w (decided)
- Health visualization: colored fill bars + 0-100 scores (decided)
- Archetype field: explicit YAML field (decided)
- Caravan/forager identification: existing btree-state convention
  (decided)
- Snapshots gitignored (decided)
