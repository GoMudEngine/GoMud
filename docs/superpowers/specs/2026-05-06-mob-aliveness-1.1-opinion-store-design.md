# Mob Aliveness 1.1 — Persistent NPC Opinion Store

**Status:** Design • **Roadmap chunk:** 1.1 (Phase 1, Substrate) •
**Size:** M • **Depends on:** —

## Goal

Per-NPC × per-player disposition score that persists across spawns,
deaths, and server restarts. Foundational substrate for Phase 4
(strategic goals) and Phase 5 (cross-cutting features like town
justice and bounty hunting).

Without this, "the merchant remembers you cheated him last week" is
impossible. With it, every downstream aliveness feature has a place
to read and write per-NPC feelings about a player.

## Non-goals

- **Player-facing visibility.** The score is internal substrate. Players
  feel the relationship through dialogue, NPC behavior, and fiction —
  never through a number on a status sheet. (CLAUDE.md: "no hard
  numbers in player-facing text".)
- **Faction rollup.** Per-faction reputation is chunk 1.2.
- **Knowledge model.** "What this NPC saw you do" is chunk 1.4. Opinion
  stores the *feeling*, not the deeds.
- **Dialogue mood migration.** The existing `internal/dialogue/mood.go`
  in-memory cache stays as is. Migration to opinion-backed mood is a
  follow-on chunk.
- **Quest YAML `disposition_reward` field.** Own chunk when quest hooks
  are wired.
- **Bulk queries** like "every NPC who hates this player." Defer until
  a consumer needs it. Directory walk is the escape hatch.
- **Mob-attacks-player bump.** Asymmetric on purpose: the mob is
  already disposing badly, that's why it's swinging. Bumping in this
  direction would create a feedback loop. Revisit if a real use case
  appears.
- **Kill-bump.** Killing is the natural conclusion of aggression — the
  player has already paid the aggression bump for starting the fight.

## Decisions

### Identity granularity

Opinions are keyed by **mob template ID (`mobId`)**, not mob instance
ID. All instances of a template share one opinion. Persists naturally
across respawn/death because the template never goes away.

DOGMud is moving away from generic spawned mobs except for ambient
animals; nearly all NPCs that matter for opinion are unique named
NPCs whose template == identity. For the rare generic case (e.g.
bandits), template-shared opinion is the desired fiction: "the
bandits remember you."

### Score shape

**Single signed scalar** in `[-100, +100]`, clamped on every write.
Stored as `int`.

A tier helper bands the scalar for consumer convenience:

| Tier        | Score range  |
|-------------|-------------|
| Hostile     | ≤ -50       |
| Cold        | -49 .. -15  |
| Neutral     | -14 .. +14  |
| Warm        | +15 .. +49  |
| Friendly    | ≥ +50       |

Multi-axis (trust/respect/fear) was considered and rejected.
Aliveness texture in DOGMud comes from combining the score with
chunk 1.4 (knowledge), 1.2 (factions), 1.6 (NPC-to-NPC relations),
and dialogue/mood, not from internal axis decomposition. Multi-axis
risks duplicating responsibilities of those chunks and turns 1.1
into "player reputation seen by NPC," which is category creep into
player-state.

### Player identity

Keyed by `userId`. In DOGMud, character-and-account identity have
been merged (a 1:1 relationship), so `userId` effectively *is* the
character key. Matches the existing `internal/dialogue/memory.go`
convention.

### Default disposition & decay

Each mob YAML may declare a `default_disposition` field (top-level,
sibling to `archetype`, `non_combatant`). Optional, defaults to `0`
(neutral) when absent.

Score decays toward the per-NPC default over **game-time** (round
count), via a half-life model. Decay is **lazy** — computed on read,
no background tick:

```
elapsed = currentRound - last_updated_round
steps   = elapsed / Balance.DispositionDecayHalfLifeRounds
score   = pull(score, default, steps)
```

Where `pull(s, d, n)` moves `s` toward `d` by `n` integer steps,
clamped so it never overshoots the default. Whenever non-zero decay
is applied, `last_updated_round` is updated and the row is marked
dirty.

`Balance.DispositionDecayHalfLifeRounds` defaults to **100,000**
rounds (~4-5 real-world days at typical round cadence). Tunable via
config.

### Storage layout

One YAML file per mob template:

```
_datafiles/world/dogmud/opinions/{mobId}-{namesimple}.yaml
```

`namesimple` derived via `util.ConvertForFilename(mob.Character.Name)`,
matching the mob YAML filename convention.

File schema:

```yaml
mob_id: 41
default_disposition: 5
opinions:
  17:
    score: -42
    last_updated_round: 1843201
  92:
    score: 28
    last_updated_round: 1846020
```

`default_disposition` is mirrored from the mob template into the
opinion file at first write. After that, the opinion file is the
source of truth — if the mob template's default later changes,
existing opinion files won't auto-update (acceptable; the score is
anchored to authoring intent at the time the relationship started).

The opinions package looks up the template default exactly once,
at first-row creation, via `internal/mobs.GetMobSpec(mobId)` (or
equivalent). This is the package's only outbound dependency on
mobs. For a `Get` call against a mobId with no opinion file at
all, the same lookup happens (without creating a row).

### Persistence model (mirrors shops)

- **In-memory cache:** `map[int]*MobOpinions` keyed by `mobId`,
  protected by `sync.RWMutex`.
- **Lazy load:** on first `Get`/`Set`/`Bump` for a `mobId`, try to
  read the file from disk; on miss, instantiate empty
  `MobOpinions` seeded from the mob template's default.
- **No proactive registration on mob spawn.** Unlike shops (which
  must seed inventory at spawn), opinions are pure data — nothing
  to do until someone queries or mutates.
- **Dirty-flag flush:** mutations mark the `MobOpinions` dirty. A
  periodic flusher writes dirty entries every N seconds (mirror
  the shop flush interval).
- **Save-all on shutdown:** `SaveAllOpinions()` called from the
  shutdown path that already calls `SaveAllShops()`.

### Lifecycle

- **Row creation:** lazy on first `Bump` or `Set` for a
  `(mobId, userId)` pair. `Get` for a non-existent row returns the
  template default *without* creating a row.
- **Row pruning:** never explicitly. Files are small (a 100-player
  `MobOpinions` is a few KB). If size becomes a real concern later,
  a "trim rows where `|score| < 2` and `last_updated >> threshold`"
  pass can run on shutdown. Defer.

### Concurrency

Per-package `sync.RWMutex` for the cache map; per-`MobOpinions`
internal mutex for the inner map. Same pattern as
`internal/shops/persistence.go`.

### Public API

A new package `internal/opinions/`. Public surface:

```go
package opinions

type Tier int

const (
    TierHostile  Tier = iota // <= -50
    TierCold                 // -49 .. -15
    TierNeutral              // -14 .. +14
    TierWarm                 // +15 .. +49
    TierFriendly             // >= +50
)

// Get returns the (decay-adjusted) score this NPC has of the given
// user. Returns the NPC's default disposition if no row exists.
// Side effect: applies and persists decay if elapsed rounds warrant
// it.
func Get(mobId int, userId int) int

// Set assigns an absolute score, clamped to [-100, +100], stamps
// last_updated_round, marks dirty. Used for admin overrides and
// quest rewards that snap to a value.
func Set(mobId int, userId int, score int)

// Bump adds delta to the current (decay-adjusted) score, clamped to
// [-100, +100], stamps last_updated_round, marks dirty. The
// everyday mutator — combat, dialogue, quest hooks all call this.
func Bump(mobId int, userId int, delta int)

// TierOf bucket-maps a score to its Tier.
func TierOf(score int) Tier

// TierFor is sugar for TierOf(Get(mobId, userId)).
func TierFor(mobId int, userId int) Tier

// SaveAllOpinions flushes dirty cache entries to disk. Call from
// the same shutdown path that calls SaveAllShops.
func SaveAllOpinions()
```

Notes:
- **No `Delete`.** Opinions are conceptually append-only; "reset" via
  `Set(mobId, userId, default_disposition)`.
- **No event subscriptions.** Callers wire bumps explicitly. The
  substrate stays dependency-free.
- **`Get` is mutating** (it applies decay and may dirty the cache).
  Documented explicitly in the doc comment. Same pattern as how
  buff effect-decay self-mutates on read.

### Integration scope for chunk 1.1

API + **one** demo consumer hookup: combat first-aggression bump.

**Combat hookup.** When a player swings at a mob the player wasn't
already fighting last round, apply a one-shot bump:

```yaml
# in _datafiles/config.yaml under Balance
OpinionAttackBump:                -15
DispositionDecayHalfLifeRounds: 100000
```

The bump fires once per fight (the "first aggression" rule
naturally limits frequency). Repeated farming runs accumulate:
~7 runs saturate at -100 and create a permanent grudge. That's the
intended fiction — repeatedly killing an NPC creates an enemy.

Hook lives in `internal/hooks/NewRound_DoCombat.go` (the unified
combat round handler per CLAUDE.md "all combat logic goes in
handleCombatRound"). Exact line determined during implementation.

**Not wired in 1.1:**
- Dialogue mood migration.
- Quest reward deltas.
- Mob-attacks-player.
- Kill-bump (rejected; aggression bump is enough).

**Discipline.** Future opinion mutations (dialogue, quest, faction)
must route through `opinions.Bump` / `opinions.Set` — no parallel
state, no direct cache writes outside the package.

### Admin debug command

Command name `opinion`, registered admin-only in
`internal/usercommands/admin.opinion.go`.

Subcommands:

```
opinion show <playerName>
opinion show <mobName|mobId> <playerName>
opinion set <mobName|mobId> <playerName> <score>
opinion bump <mobName|mobId> <playerName> <delta>
opinion reset <mobName|mobId> <playerName>
```

`opinion reset` snaps the row back to the NPC's
`default_disposition`.

Output mirrors DOGMud's ANSI-table style, 80-column wrap:

```
opinion show Aliceia:

  Mob               Score  Tier      Last bump
  ----------------  -----  --------  ---------
  Lars (41)         -42    Cold      ~4hr ago (game)
  Voss (42)          28    Warm      ~22hr ago (game)
  Stillwater Smith    5    Neutral   default
```

Helpfile and admin command-list entry both ship with the command.
Helpfile path follows the existing admin help convention; verified
during implementation.

## Architecture diagram

```
                ┌──────────────────────────────────────┐
                │   _datafiles/world/dogmud/opinions/  │
                │     {mobId}-{namesimple}.yaml         │
                └─────────────┬────────────────────────┘
                              │ lazy load / dirty flush
                              ▼
   ┌──────────────────────────────────────────────────────┐
   │                internal/opinions package             │
   │  cache: map[mobId]*MobOpinions  (sync.RWMutex)        │
   │  API:   Get, Set, Bump, TierOf, TierFor, SaveAll      │
   │  Decay: lazy, half-life from Balance config           │
   └─────┬────────────────────────────────────┬──────────┘
         │ Bump / Set                         │ Get / TierFor
         ▼                                    ▼
  ┌─────────────────────────┐   ┌─────────────────────────────┐
  │  Combat hookup (1.1)    │   │  Future consumers (later    │
  │  NewRound_DoCombat.go   │   │  chunks): dialogue, quest,  │
  │  first-aggression bump  │   │  faction, btree, goals...   │
  └─────────────────────────┘   └─────────────────────────────┘

  ┌─────────────────────────┐
  │  admin.opinion.go (1.1) │
  │  show / set / bump /    │
  │  reset                  │
  └─────────────────────────┘
```

## Test plan

**Unit (`internal/opinions/opinions_test.go`):**
- `Get` returns default when no row exists, no row created.
- `Bump` clamps at `[-100, +100]`.
- `Set` overrides absolute, stamps round.
- Decay math: 0/1/2/N half-lives moves score toward default by
  integer steps, never overshoots, doesn't drift past default.
- `TierOf` boundary cases (-50, -49, -15, -14, +14, +15, +49, +50).

**Persistence (`internal/opinions/persistence_test.go`):**
- Save → load round-trip preserves all rows.
- Missing file → fallback to template default disposition.
- Corrupt YAML → log warning, treat as missing (do not crash
  startup).
- Dirty-flag flush triggers a write; clean flag does not.
- `SaveAllOpinions` writes only dirty entries.

**Concurrency:**
- Parallel `Bump` from N goroutines on the same `(mobId, userId)`
  converges (no lost writes).
- Parallel reads during a write don't race.

**Integration (combat hookup):**
- Player attacks mob who wasn't aggressed last round → opinion
  bumps by `OpinionAttackBump`.
- Subsequent swings same fight → no further bump.
- Mob dies → no kill-bump applied.
- Bump persists across server restart (save → restart → reload →
  same value, plus decay if applicable).

**Admin command (`internal/usercommands/admin.opinion_test.go`):**
- Each subcommand parses correctly.
- `show <player>` lists only non-default rows.
- `reset` snaps to default disposition.

## Non-functional requirements

1. **Single source of truth.** All opinion mutations route through
   `opinions.Bump` / `opinions.Set`. No parallel state, no direct
   cache writes outside the package.
2. **Write amplification budget.** Persistence flushes every N
   seconds (mirror shops). One combat round produces no more than
   one disk write per affected mob.
3. **Startup cost.** Lazy load — server startup must not eagerly
   read all opinion files. Memory baseline at boot stays roughly
   unchanged.
4. **Decay determinism.** Decay is a pure function of
   `(score, default, elapsed_rounds, half_life)`. Replayable in
   tests.
5. **No player-facing numbers.** All player-visible relationship
   signals come through fiction (dialogue, behavior). The score
   sheet is admin-only.

## Files touched (estimate)

**New:**
- `internal/opinions/opinions.go` — Tier, API surface
- `internal/opinions/decay.go` — pure decay math
- `internal/opinions/persistence.go` — file load/save, cache,
  flusher
- `internal/opinions/types.go` — Opinion, MobOpinions
- `internal/opinions/opinions_test.go`
- `internal/opinions/persistence_test.go`
- `internal/usercommands/admin.opinion.go`
- `internal/usercommands/admin.opinion_test.go`
- `_datafiles/world/dogmud/templates/help/admin/opinion.template`
  (path verified during implementation)

**Modified:**
- `internal/configs/config.balance.go` — add
  `OpinionAttackBump`, `DispositionDecayHalfLifeRounds`
- `_datafiles/config.yaml` — defaults for the above
- `internal/hooks/NewRound_DoCombat.go` — first-aggression hook
- `internal/mobs/mobs.go` (or schema-loader) — read
  `default_disposition` from mob YAML
- Shutdown path — call `opinions.SaveAllOpinions()`
- Admin command registration — include `opinion`
- Admin help index template — list `opinion`

## Open questions for implementation

These do not block design approval but will be resolved during
plan-writing:

- **Helpfile path.** Confirm the exact admin helpfile directory by
  reading existing admin command examples.
- **Periodic flush interval.** Match the existing shop flush
  interval directly; pull the value from the same config knob if
  one exists.
- **First-aggression detection.** Identify the cleanest signal in
  the current combat state machine (likely `user.Character.Aggro`
  or equivalent) to detect the round-zero attack.
- **`MobOpinions` write batching during periodic flush.** Verify
  that the shop flusher already batches via a single goroutine and
  re-use the pattern verbatim.
