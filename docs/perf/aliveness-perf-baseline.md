# Aliveness Performance Baseline

Fine-grained baseline for the mob-aliveness substrate + per-tick work, captured
after chunk 6.4 instrumentation landed. **Re-run the procedure below verbatim for
chunk 6.6** (post-content-pass re-review) and append a new dated section to
compare against these floors.

See also: the coarse prod pull/restart + idle-CPU log in the
`reference_prod_perf_baseline` memory (this doc is its fine-grained companion).

## Instrumentation (always-on, no toggle)

Both kinds of instrumentation are always-on by design (decision 2026-06-05):
- **Memory reporters** (8 substrate packages) are **pull-only** — executed only
  when an admin runs `server stats`, so zero per-tick cost.
- **Timer seams** extend the engine's existing always-on `util.TrackTime` set at
  sub-microsecond cost (measured below). Gating them would diverge from the
  existing timers and break apples-to-apples comparison for 6.6.

Always-on also enables a future admin-dashboard section to read this data live.

## Capture procedure (re-run exactly for 6.6)

The server must be built from a tree with the 6.4 instrumentation. Drive it over
telnet with a **single, strictly-sequential admin connection** — overlapping
connects can trip a pre-existing concurrent-map-write crash in the login path
(see `project_templates_configcache_concurrent_write_panic`; fixed separately on
`fix/templates-configcache-race`).

**Idle floor:**
1. `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`
2. Boot locally, **no players connected**.
3. Wait until `server stats` shows the `IdleMobs()` accumulator `Ct` = ~500
   (Ct = ticks/rounds elapsed; ~4s/round → ~33 min).
4. Snapshot the full `server stats` output (Timer Stats table + memory report).

**Under load:**
1. Same wipe + boot.
2. Connect one admin player, `teleport 468` (Temple Interior, Thornwall), and
   issue `look` every ~30s to defeat the AFK timer (a single persistent player
   present is the load — per the 6.4 design).
3. Snapshot `server stats` at the same `IdleMobs()` Ct ≈ 500 mark.

**Timer denominators:** `IdleMobs::{schedule,patrol,conversation}` are per-tick
totals (recorded once/tick → sum ≤ `IdleMobs()`); `MobIdle::goalplanner` and
`Enforcement` are per-invocation (per mob), so their `Ct` is much higher.

---

## 2026-06-05 — 6.4 baseline (master-track @ f9c775cd, ~228 mobs / ~261 items / ~200 rooms loaded)

Captured locally (Windows, `GoMud.exe` built from the 6.4 branch). Idle snapshot
at IdleMobs() Ct=524; load snapshot at Ct=536 with Users=1 (one admin player
sitting in Temple Interior 468, looking every 30s).

### Substrate memory footprint

| Section | Store | Idle count | Idle size | Load count | Load size |
|---------|-------|-----------|-----------|-----------|-----------|
| Relationships | graph | 34 | 0.7 KB | 34 | 0.7 KB |
| Facts | awarenessCache | 22 | 2.1 KB | 22 | 2.1 KB |
| Facts | registry | 10 | ~0 KB | 10 | ~0 KB |
| Goals | cache | 31 | 1.2 KB | 31 | 1.2 KB |
| Goals | nameByMobId | 31 | 0.6 KB | 31 | 0.6 KB |
| Opinions | opinionCache | 0 | ~0 KB | 0 | ~0 KB |
| Opinions | nameByMobId | 0 | ~0 KB | 0 | ~0 KB |
| Factions | definitions | 5 | 1.1 KB | 5 | 1.1 KB |
| Factions | repCache | 3 | 0.2 KB | 2 | 0.1 KB |
| Crimes | crimeCache | 2 | 0.1 KB | 0 | ~0 KB |
| Knowledge | knowledgeCache | 22 | 0.6 KB | 22 | 0.6 KB |
| Bounties | registry | 35 | ~0 KB | 35 | ~0 KB |
| **Aliveness substrate subtotal** | | | **~7 KB** | | **~7 KB** |
| **Total (Non-Go, all sections)** | | | **2.1 MB** | | **2.1 MB** |
| Go HeapAlloc | | | 15.1 MB | | 11.3 MB |
| Go Sys (everything) | | | 45.5 MB | | 41.2 MB |

The aliveness substrate is **~7 KB** — a rounding error against the 2.1 MB non-Go
total, which is dominated by Mobs (~1.3 MB), Rooms (~0.3 MB), and Items (~0.14 MB).

### Tick budget (avg / low / high ms, count, per-sec)

| Seam | Idle avg | Idle high | Idle Ct | Load avg | Load high | Notes |
|------|----------|-----------|---------|----------|-----------|-------|
| IdleMobs() (roll-up) | 0.002 | 0.532 | 524 | 0.000 | 0.024 | per-tick |
| IdleMobs::schedule | 0.000 | 0.000 | 524 | 0.000 | 0.024 | per-tick |
| IdleMobs::patrol | 0.000 | 0.010 | 524 | 0.000 | 0.000 | per-tick |
| IdleMobs::conversation | 0.001 | 0.509 | 524 | 0.000 | 0.000 | per-tick; largest aliveness seam |
| MobIdle::goalplanner | 0.000 | 1.007 | 11205 | 0.000 | 1.001 | per-invocation |
| Enforcement | 0.001 | 1.047 | 756 | 0.000 | 0.000 | per-invocation (guards) |
| events.ProcessEvents() (roll-up) | 0.005 | 67.697 | 14147 | 0.005 | 67.312 | per-batch |
| DoCombat::handlePlayerCombat() | — | — | — | 0.000 | 0.000 | load only (no combat) |

### Reading

- **Memory is a non-issue.** The full aliveness substrate is ~7 KB; even at many
  multiples after the 6.5 content pass it stays negligible against the mob/room
  footprint. The per-tick seams confirm the parent/child relationship holds
  (`IdleMobs::*` sum ≤ `IdleMobs()`), with **conversations the largest aliveness
  seam** (0.001 ms avg idle) — expected, since the conversation trigger rolls
  every idle tick.
- **Tick budget has enormous headroom.** Every aliveness seam averages
  sub-microsecond; the worst-case highs are ~1 ms (a single goal-planner or
  enforcement invocation). The one large outlier — `events.ProcessEvents()` high
  of ~67 ms — is a periodic spike (autosave/GC coinciding with an event batch),
  **not** an aliveness seam, and its *average* is 0.005 ms.
- **Idle vs load is essentially identical.** One player sitting in a room adds no
  measurable per-tick cost (combat seam present but 0.000 ms with no fighting;
  goal-planner invocation count rises slightly). The lower load-snapshot HeapAlloc
  (11.3 vs 15.1 MB) is GC-timing variance, not a real reduction.
- **Conclusion:** the substrate and per-tick aliveness work leave comfortable
  headroom before the 6.5 content pass scales mob/zone counts up. 6.6 should watch
  whether `IdleMobs::conversation` and `MobIdle::goalplanner` averages climb as
  many more scheduled/relationship-bearing NPCs come online across zones — those
  are the most likely growth vectors — and whether the substrate subtotal moves
  off the KB scale.
