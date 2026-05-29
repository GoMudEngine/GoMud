# Mob Aliveness 4.6 — Goal Satisfaction & Pruning

**Date:** 2026-05-29
**Chunk:** 4.6 (Strategic layer) • **Size:** S • **Depends on:** 4.1–4.5

---

## Overview

Chunks 4.1–4.5 give mobs a persistent, typed goal list that is seeded,
selected each round, translated to behavior, and reactively generated. What's
missing is **cleanup**: nothing removes goals once they're done, impossible, or
stale, so mobs accumulate "ghost desires" that waste evaluation and can leave
an NPC fixated on an unreachable target.

The primitives already exist but are unused in production:
- `goals.IsSatisfied(g, mob)` — per-type satisfaction predicate (all 13
  catalog types already register one).
- `goals.IsExpired(g, now)` — pure `ExpiresAt` time check.

4.6 wires these into a **throttled prune sweep** and adds **dormancy-based
abandonment** for goals that are neither satisfied nor expired but have become
permanently unreachable (their context score sits at ~0).

---

## Removal triggers (per goal, per sweep)

1. **Satisfied** — `IsSatisfied(g, mob)` true. Remove, reason `satisfied`.
   (e.g. `revenge-mob` target dead, `wealth-gold` at target, `mastery-skill`
   rank reached.)
2. **Expired** — `IsExpired(g, now)` true (`ExpiresAt` in the past; zero =
   never). Remove, reason `expired`.
3. **Abandoned-stale** — context score has been ~0 for
   `GoalAbandonDormantRounds`. Remove, reason `abandoned-stale`. This retires
   ongoing goals (`protection-mob`/`protection-faction`, whose predicate is
   permanently `false`) once their target is dead/gone — matching the existing
   `// 4.6's pruning sweep removes the goal when the target has been dead for
   >= N rounds` note in `protection_mob.go`.

---

## Dormancy tracking

Add one persisted field to `Goal`:

```go
DormantSinceRound uint64 `yaml:"dormant_since_round,omitempty"`
```

Backward compatible (omitempty, zero default on existing files).

At each sweep, for each goal, compute its context score via the registered
`ContextScore` (using the existing panic-recovered helper that `Select` uses;
nil ContextScore defaults to 1.0):

- score **> 0** → goal is live: set `DormantSinceRound = 0` (clear).
- score **== 0**:
  - if `DormantSinceRound == 0`, stamp it to `nowRound`.
  - if `nowRound - DormantSinceRound >= GoalAbandonDormantRounds`, abandon.

Keying on **selectability** (score > 0), not on **being selected**, means a
legitimate lower-priority goal waiting behind a higher-priority one (score > 0)
is never abandoned — only genuinely unreachable goals accumulate dormancy.
Goal types with no registered `ContextScore` (default 1.0) never go dormant and
are pruned only by satisfied/expired.

The file is written only on dormancy **transitions** (live↔dormant) or on
removal — never every tick, preserving the no-churn property `Recompute`
already maintains.

---

## Components

### `internal/goals/prune.go` (new)

```go
// PruneReason categorizes why a goal was removed.
type PruneReason string
const (
    ReasonSatisfied PruneReason = "satisfied"
    ReasonExpired   PruneReason = "expired"
    ReasonAbandoned PruneReason = "abandoned-stale"
)

// PruneRecord reports one removed goal (for logging).
type PruneRecord struct {
    GoalId string
    Type   string
    Reason PruneReason
}

// Prune runs the full sweep for one mob template under a single write
// lock: it evaluates every goal, updates dormancy stamps, removes dead
// goals in one batch, persists once, and triggers one Recompute. Returns
// the removed records (may be empty). Safe with a nil mob (ContextScore
// funcs defend themselves; satisfied/expired still evaluated).
func Prune(mobId int, namesimple string, mob *mobs.Mob, now time.Time, nowRound uint64) []PruneRecord
```

Rationale: a single batch sweep (one lock → one save → one `Recompute`)
instead of N× `Remove` (each of which saves + recomputes) avoids file and
selection churn.

### `Goal` struct (`internal/goals/types.go`)
Add `DormantSinceRound` field (above).

### Tick integration (`internal/hooks/NewRound_MobRoundTick.go`)
`tickMobRecomputeGoals` calls `goals.Prune(...)` **before** `goals.Recompute`,
gated to the throttled cadence and staggered per mob:

```go
if (nowRound+uint64(templateId)) % uint64(pruneInterval) == 0 {
    if recs := goals.Prune(templateId, name, mob, time.Now().UTC(), nowRound); len(recs) > 0 {
        // debug log per record (below)
    }
}
goals.Recompute(templateId, name, mob, nowRound)
```

### Config (balance)
- `GoalPruneIntervalRounds` (sweep cadence) — default **50**.
- `GoalAbandonDormantRounds` (dormancy threshold) — default **600** (same order
  as the existing forager watchdog).

### Logging
Per removed goal, a debug line mirroring `goals.switch`:
```
goals.prune mob_id=<id> goal=<gid>(<type>,<prio>) reason=<reason> round=<n>
```

---

## Testing

Unit tests (`internal/goals/prune_test.go`):
- Satisfied goal removed (reason `satisfied`).
- Expired goal removed (reason `expired`); zero `ExpiresAt` never expires.
- Goal dormant for `>= GoalAbandonDormantRounds` removed (`abandoned-stale`);
  dormant but under threshold kept.
- Dormancy cleared when a goal's score returns to > 0 (no abandonment).
- Active/backseat goal (score > 0, not selected) kept.
- Batch: multiple dead goals removed in a single sweep/save.
- nil-mob safety: satisfied/expired still evaluated; ContextScore defended.

Tick test (`NewRound_MobRoundTick` area): `Prune` fires only on the throttled
cadence, not every round.

Boot smoke: server boots clean (new `Goal` field round-trips through existing
goal files; no panic).

---

## Out of Scope

- Per-type predicate changes — all 13 already register predicates.
- Admin-command surfacing of prune reasons (log-only this chunk).
- Instance-vs-template goal-ownership rework.
- Retuning context-score curves.
