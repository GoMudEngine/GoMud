# Mob Aliveness 4.2 — Goal Selection Design

**Date:** 2026-05-27
**Status:** Approved (design)
**Roadmap position:** Phase 4 (strategic layer). After 4.2: 24/42.
**Depends on:** 4.1 (goal substrate — shipped).
**Next chunks:** 4.3 Goal types catalog → 4.4 Strategic→tactical translation.

---

## Summary

Add a pure selection function on top of the 4.1 goal substrate that picks one
"current goal" per mob from its goal list, weighted by priority, per-archetype
multipliers, and an optional per-goal-type context-score hook. Selection runs
on every mob round tick (cheap-path when goal list is empty) and eagerly on
goal-list mutations. Hysteresis (margin + min-hold) prevents thrash.
Selection state persists in the existing MobGoals YAML.

4.2 ships the pure function, the per-tick recompute hook, the schema delta,
new admin subcommands (`goal current` / `goal scores`), a debug-level log
line on every switch, and config knobs for tuning. **It ships with an empty
`ContextScore` registry** — 4.3 fills it per concrete goal type. **No
behavior-tree integration yet** — 4.4 wires the selected goal into tactical
execution.

The chunk is "no player-facing change" by design. NPCs select goals; nothing
reads the selection yet. Observable surface: admin command output + a log
line per switch.

---

## Goals

- Provide one cached "current goal" per mob, derivable on demand and updated
  on tick + on mutation.
- Selection is a **pure function** of `(goals, archetype weights, mob
  context, prev state, now)`. All side effects (persistence, log emission)
  live in the caller.
- Anti-thrash via two-gate hysteresis: a new goal must outscore the current
  one by a margin AND the current goal must have been held for a minimum
  number of rounds.
- Per-archetype goal-type weighting via a declarative `goal_weights:` map
  in archetype YAML.
- Per-goal-type context-score hook (matching the existing `Predicate`
  pattern in `GoalTypeMeta`) so 4.3 can author context-aware goal types
  (e.g. survival when low HP) without 4.5 reactive plumbing.
- Persist selection state (`current_goal_id` + round timers) in the
  existing per-template MobGoals YAML.
- Admin observability: `goal current <mob>` and `goal scores <mob>`
  subcommands; structured debug log line on every switch.
- Stay decoupled: 4.2 produces "the current goal" as cached state. 4.4's
  job is to make tactical execution read it.

## Non-goals

- Multi-goal pursuit — single-goal-at-a-time, per the roadmap.
- Concrete goal types and their predicates / context-scores — 4.3.
- Behavior-tree integration / tactical execution — 4.4.
- Reactive goal generation from events — 4.5.
- Goal satisfaction sweeping / pruning — 4.6.
- Per-instance current goal (per-template storage is the substrate's
  contract; shared-template selection state is an accepted limitation).
- Mutation-triggered archetype/goal-weight shifts (captured as a future
  followup in MEMORY.md → `project_mutation_triggered_archetype_shift`;
  revisit after 4.4).

---

## 1. Architecture & data flow

A new pure selection function lives in the existing `internal/goals/`
package. It runs in two places:

1. **Per-mob round tick hook** (`internal/hooks/MobRoundTick_RecomputeGoals.go`,
   mirroring the naming pattern of existing per-mob round hooks). Fires
   every mob round tick for every loaded mob instance. Early-returns when
   the goal list is empty (the common case at 4.2 ship — 4.3/4.5 populate it).
2. **Eagerly on goal-list mutations** — `Add`, `Remove`, `Clear` — these
   call `Recompute(mob, currentRound)` after mutating the cache (best
   effort; if no instances are loaded, the mutation-time call no-ops and
   the next tick handles it).

`Select(goals, weights, mob, prev, currentSinceRound, lastSwitchRound,
nowRound) (current *Goal, switched bool, reason SelectReason)` is the
pure function. Same inputs → same output. All side effects (cache writes,
log emission, persistence) happen in `Recompute()`, not inside `Select`.

`CurrentGoalOf(mobId, namesimple) *Goal` is a cheap accessor. Future btree
consumers (4.4) read this.

**Key invariant:** `Select` is pure. `Recompute` orchestrates side
effects. Mutation paths call `Recompute` eagerly; the tick hook calls
`Recompute` per round.

---

## 2. API surface

```go
package goals

// Extends 4.1's GoalTypeMeta. ContextScore is optional; nil = 1.0.
type GoalTypeMeta struct {
    Predicate     PredicateFn
    ConflictsWith []string
    ContextScore  ContextScoreFn // NEW — multiplier from current mob state
}

// ContextScoreFn returns a non-negative multiplier. 0 effectively
// suppresses the goal from selection this tick (e.g. "revenge target
// not in zone"). Must be pure(ish): same goal + same mob state →
// same answer. Side effects forbidden — may be called from any context.
type ContextScoreFn func(g *Goal, mob *mobs.Mob) float64

// SelectReason explains why the selector picked what it did. Used by
// the log line + admin command.
type SelectReason struct {
    Kind   string // "no_goals", "kept_no_candidates", "kept_top_unchanged",
                  // "kept_hysteresis_margin", "kept_hysteresis_min_hold",
                  // "switched", "switched_prev_invalid"
    Detail string // free-form, e.g. "g3(80) beat current g1(70) by 10pts after min-hold"
}

// Select is pure. nil prev = mob has no current goal yet.
// Returns (newCurrent, switched, reason). newCurrent may equal prev.
func Select(
    goals []*Goal,
    weights map[string]float64,
    mob *mobs.Mob,
    prev *Goal,
    currentSinceRound, lastSwitchRound, nowRound uint64,
) (current *Goal, switched bool, reason SelectReason)

// CurrentGoalOf returns the cached current goal, or nil if there is
// none or the cached id is stale (goal removed). Cheap accessor; lazy-
// loads MobGoals on first access (matches GoalsOf semantics).
func CurrentGoalOf(mobId int, namesimple string) *Goal

// Recompute is called by the tick hook + Add/Remove/Clear. Reads goals,
// calls Select, persists current_goal_id / current_since_round /
// last_switch_round on switch, emits debug log line on switch.
// Recovers panics from ContextScore functions (logs warning, scores 0).
func Recompute(mob *mobs.Mob, nowRound uint64)
```

Archetype side (existing archetype package):

```go
// Loaded from `goal_weights:` map in archetype YAML. Returns empty map
// (selection treats as default 1.0 per type) if archetype declares none
// or the archetype id is unknown.
func WeightsFor(archetypeId string) map[string]float64
```

**Effective score formula** (single line, inside `Select`):

```
effectiveScore(g) = float64(g.Priority) * weights[g.Type] * contextMod(g, mob)
                    (default weight = 1.0 for unlisted types)
                    (default contextMod = 1.0 when ContextScore is nil)
```

Selection picks the goal with the highest effective score, then applies
hysteresis gates against `prev` before deciding to switch.

Concurrency: existing 4.1 `sync.RWMutex` covers the cache. `Recompute`
takes write lock for the duration of any cache update + persistence.
`CurrentGoalOf` takes read lock. `Select` itself is lock-free (operates
on caller-supplied snapshots).

---

## 3. Selection logic

```
Select(goals, weights, mob, prev, currentSince, lastSwitch, now):

1. Filter goals: drop satisfied (IsSatisfied true), expired (IsExpired
   now true), and contextMod==0 candidates. Keep filtered list as
   `candidates`.

2. If candidates is empty AND prev still valid (in goals list, not
   satisfied, not expired):
     → keep prev. reason = "kept_no_candidates".

3. If candidates is empty AND prev invalid:
     → return nil. reason = "no_goals".

4. Pick top scorer by effectiveScore. Tie-break: priority desc, then
   id asc (matches GoalsOf).

5. If prev is nil OR prev invalid (removed / satisfied / expired):
     → switch to top. reason = "switched" or "switched_prev_invalid".

6. If top == prev:
     → keep prev. reason = "kept_top_unchanged".

7. Otherwise compute hysteresis gates:
     a. heldRounds = max(0, nowRound - currentSinceRound)
        (clamp guards against stale currentSinceRound > nowRound — see
         edge case 10. In normal operation, currentSinceRound ≤ nowRound.)
     b. scoreGap  = effectiveScore(top) - effectiveScore(prev)
     c. If heldRounds < GoalSelectMinHoldRounds:
          → keep prev. reason = "kept_hysteresis_min_hold"
            (detail: "heldRounds/min")
     d. If scoreGap < GoalSelectSwitchMargin:
          → keep prev. reason = "kept_hysteresis_margin"
            (detail: "gap/required")
     e. Else: switch. reason = "switched"
            (detail: "top.id(score) beat prev.id(score) by gap, after held rounds")
```

**Both gates must pass to switch.** `prev_invalid` bypasses both gates
(you can't be "stuck" on a goal that no longer exists).

---

## 4. Hysteresis config knobs

Three new fields in `Balance` (existing `configs/config.balance.go`):

| Knob | Default | Meaning |
|---|---|---|
| `GoalSelectSwitchMargin` | `5.0` | New goal's effective score must exceed current goal's by ≥ this margin to switch. Float so weights/contextMod can produce fractional scores. |
| `GoalSelectMinHoldRounds` | `100` | Current goal must have been current for at least this many rounds before any switch is allowed (≈ 5 minutes at default round cadence). |
| `GoalSelectTickEnabled` | `true` | Master kill-switch for the tick-driven recompute path. Eager recompute on Add/Remove/Clear still fires when false (substrate stays consistent). |

Defaults are deliberately conservative — strategic shifts should feel
deliberate. 4.3/4.4 smoke testing will likely re-tune.

---

## 5. Tick + event integration

### 5.1 Tick hook

`internal/hooks/MobRoundTick_RecomputeGoals.go`, registered with the
round-tick dispatcher next to the existing conversation tick / mob-idle
hook.

Per loaded mob instance per round tick:

```
1. If GoalSelectTickEnabled == false → return early.
2. Resolve mob template id + namesimple.
3. If GoalsOf(templateId, namesimple) is empty → return early
   (cheap path; most mobs at 4.2 ship).
4. Call goals.Recompute(mob, currentRoundNumber).
```

**Throttling.** None at 4.2. `Recompute` cost is O(n) goals × O(1)
per goal (one map lookup + one multiplication + one optional function
call). With ≤ ~10 goals per mob in practice, negligible. If profiling
later flags this, add a "recompute only every N rounds when goal list
unchanged" pass — not premature.

### 5.2 Eager recompute on mutation

Modify the existing `Add`, `Remove`, `Clear`:

```
After persistence write succeeds (or fails with logged warning):
  for each currently-loaded instance of this template:
    goals.Recompute(instance, currentRound)
```

**Instance lookup:** there is no direct `GetInstancesByTemplate` helper
on the `mobs` package today. 4.2 iterates `mobs.GetAllMobInstanceIds()`
and filters by `mobs.GetInstance(id).MobId == templateId`. With
instance counts in the low thousands and mutations being rare admin
or future-4.5 events (not per-tick hot path), this is fine. If
profiling later shows it as a hotspot, add a maintained
`GetInstancesByTemplate` index in `mobs/`.

Best-effort: if no instances are loaded (e.g. admin adds a goal for an
offline mob), the recompute happens naturally on next tick when an
instance loads.

### 5.3 Per-template vs per-instance recompute

MobGoals file is keyed by template id (4.1 contract), so `current_goal_id`
is also per template. When multiple instances of the same template are
loaded (rare — most named NPCs are 1:1), the first one to tick this
round triggers `Recompute`; subsequent instances see the same cached
state. Accepted limitation: shared-template instances share their current
goal. Per-instance current goals are a Phase 5+ concern if it ever
matters.

### 5.4 Mob context passed to ContextScore

The hook hands `Select` the live `*mobs.Mob` instance — the one currently
being ticked. Predicates and ContextScore functions can read HP%,
position, room id, in-combat status, buff list, etc. from it. For the
per-template case above, "the mob" passed in is the first-ticking
instance.

---

## 6. Persistence schema delta

Three new fields on `MobGoals` (top-level — these track the mob's
selection state, not per-goal):

```yaml
mob_id: 371
next_goal_id: 4
current_goal_id: g2        # NEW — id of current goal, "" if none
current_since_round: 12450 # NEW — round when current became current
last_switch_round: 12450   # NEW — round of most recent switch
goals:
  - id: g1
    ...
```

```go
type MobGoals struct {
    MobId             int     `yaml:"mob_id"`
    NextGoalId        int     `yaml:"next_goal_id"`
    CurrentGoalId     string  `yaml:"current_goal_id,omitempty"`     // NEW
    CurrentSinceRound uint64  `yaml:"current_since_round,omitempty"` // NEW
    LastSwitchRound   uint64  `yaml:"last_switch_round,omitempty"`   // NEW
    Goals             []*Goal `yaml:"goals"`
}
```

**Boot / lazy-load:** existing files load cleanly — missing fields
default to zero values. First post-boot `Recompute` selects a goal
afresh and writes the fields. No migration needed.

**On `Remove(goalId)`:** if the removed goal was `CurrentGoalId`, clear
`CurrentGoalId` and zero the round fields under the same write lock.
The eager `Recompute` fired by `Remove` (§5.2) then runs immediately
with `prev == nil` and selects fresh from whatever remains.

**On `Clear()`:** zeros all three fields plus the goal list.

**On `Add()`:** does not touch these fields directly. The eager
`Recompute` after `Add` does, if it picks the new goal.

**On `Recompute` with no switch:** the file is NOT rewritten — the cache
still matches disk. Writing on every tick would be wasteful. Disk
catches up on the next actual switch.

**Atomic write:** existing `.tmp` + `os.Rename` covers the three new
fields with no change.

---

## 7. Archetype `goal_weights:` integration

Each archetype YAML in `_datafiles/world/dogmud/behaviors/archetypes/*.yaml`
can optionally declare:

```yaml
goal_weights:
  revenge: 1.5
  wealth-target: 1.2
  protection: 0.7
```

Default weight for any unlisted type is `1.0`. Loader behavior:

- Missing field → empty map (everything scores at 1.0). No warning.
- Malformed map (not a `map[string]float64`) → log single-line warning,
  treat as empty map. Other archetype fields load normally. Soft check.
- Unknown goal type referenced (no `GoalTypeMeta` registered for that
  key) → log single-line warning at boot, drop the key. Selection
  ignores it. Not a panic — content drift shouldn't crash the server.
- Negative or zero values are allowed (weight = 0 effectively filters
  the type for this archetype).

`WeightsFor(archetypeId)` accessor returns the loaded map (or empty map
if the archetype id is unknown). Lives in the archetype package next to
existing accessors.

---

## 8. Admin command extension

Extend the existing `goal` admin command (4.1 shipped `list / show /
add / remove / clear`). Add two read-only subcommands:

```
goal current <mob-ident>
goal scores  <mob-ident>
```

Both reuse the existing `<mob-ident>` resolution (numeric template id
OR namesimple).

### 8.1 `goal current <mob-ident>` — single-line status

```
Current goal for Tova (mob 371): g2 wealth-target priority=30
  current since round 12450 (235 rounds, ~12s ago)
  last switch round 12450, reason: switched
  archetype: forager, weights: (default)
```

If no current goal:

```
Current goal for Tova (mob 371): none
  (0 goals on file)

OR

Current goal for Tova (mob 371): none
  (last selection: kept_no_candidates — 3 goals filtered out as
   satisfied/expired)
```

### 8.2 `goal scores <mob-ident>` — full score breakdown table

```
Score breakdown for Tova (mob 371):
  Archetype: forager  Weights: revenge=1.2, wealth=0.8, (others=1.0)
  Hysteresis: margin=5.0  min-hold=100 rounds  held=235 rounds

  ID   Type            Pri   Weight  CtxMod  Effective  Status
  g1   revenge          70    1.2    1.0      84.00     candidate
  g2   wealth-target    30    0.8    1.0      24.00     CURRENT
  g3   protection       20    1.0    0.0       0.00     filtered (contextMod=0)
  g4   debt             40    1.0    1.0      40.00     filtered (satisfied)

  Selection: g1 would win (84.00 > 24.00 by 60.00pts, margin=5.0 ✓
             min-hold satisfied 235/100 ✓)
             SWITCH WOULD FIRE on next Recompute.
```

**Status values:** `CURRENT` / `candidate` / `filtered (satisfied)` /
`filtered (expired)` / `filtered (contextMod=0)`.

**Hysteresis explanation footer** is computed off the same `Select`
invocation that drives the table — table and footer always agree.

**Wire-up:** the existing dispatch switch in
`internal/usercommands/admin.goal.go` gets two new cases. Both
read-only — no persistence, no goal mutation. `goal scores` calls
`Select` itself (the pure function) to produce the breakdown without
touching the cache or persistence.

### 8.3 Helpfile

`_datafiles/world/dogmud/templates/admincommands/help/command.goal.template`
already exists from 4.1. **Update it** to document the two new
subcommands. The plan must call this out explicitly — the helpfile is
not a new file, and a subagent must not no-op on it. Per the
"admin-command-wiring-checklist" feedback in MEMORY.md, every wiring
step must be its own task.

The `goal` command itself is already registered in
`internal/usercommands/usercommands.go`'s command table from 4.1 (no
new registration needed — same handler resolves all `goal *`
subcommands).

---

## 9. Edge cases

| # | Case | Behavior |
|---|---|---|
| 1 | All goals satisfied/expired/zero-context-mod | `Select` returns nil + `reason=no_goals`. `Recompute` clears `CurrentGoalId` and zeros round fields. |
| 2 | Mob has no archetype | `WeightsFor("")` returns empty map. All goals score at `priority × 1.0`. Selection works fine. |
| 3 | Archetype references unknown goal type in `goal_weights:` | Loader logs single-line warning at boot, drops the unknown key. Selection ignores it. Not a startup panic. |
| 4 | Archetype YAML has malformed `goal_weights:` | Loader logs warning, treats as empty map. Other archetype fields load normally. |
| 5 | `ContextScore` registered but panics on a particular mob state | Recovered by `Recompute`'s deferred panic catch (mirrors how `actBtreeStep` handles action panics). Logs `goals.context_score panic` with type + mob id; that goal scored 0 for this tick. |
| 6 | `current_goal_id` in MobGoals file points to a goal id that no longer exists (manual file edit, version drift) | `Select` sees `prev=nil` (lookup miss), branches to `switched_prev_invalid`, picks fresh. File self-heals on next switch. |
| 7 | Two instances of same mob template tick in same round | First instance's `Recompute` writes cache + file. Second instance's `Recompute` sees `top == prev` (the same goal was just selected) and short-circuits at step 6 with `kept_top_unchanged`. Idempotent regardless of hysteresis state. |
| 8 | Goal added with priority high enough to beat current goal mid-min-hold-window | Eager `Recompute` after `Add()` fires. Min-hold gate still applies — new goal must wait out the cooldown unless `prev_invalid`. Authoring escape hatch: admin can `goal remove` the current goal first, which clears prev and lets the new goal win immediately. |
| 9 | Mob in combat when `Recompute` fires | Selection runs normally; current goal can switch. Tactical (btree) decisions stay tactical — 4.4 will decide whether a goal switch *during* combat re-routes the tactical loop. 4.2 just updates state. |
| 10 | Stale `CurrentSinceRound` > `nowRound` (e.g. round-counter file got reset or restored from an older snapshot than the goals file) | uint64 subtraction would underflow. `Select` clamps `heldRounds = max(0, nowRound - currentSinceRound)` — if currentSince > now, treat held = 0 (min-hold blocks). On the next switch, the stale value is overwritten. The clamp keeps `Select` defensive against any future plumbing that lets these get out of sync. (Normal operation: `roundCount` is persisted and clamped to `RoundCountMinimum = 1,314,000` at boot, so it never goes backward from a clean shutdown.) |
| 11 | `goal scores` invoked from admin command during heavy contention | `Select` is pure and uses passed-in args; the admin command takes its own read-lock snapshot of `MobGoals` then releases before calling `Select`. No write contention from the read path. |
| 12 | Schedule activity changes mid-round (e.g. mob enters sleep segment) | Schedules and goals are orthogonal layers — selection runs whether or not the mob is sleeping. `ContextScore` for individual goal types can consult `mob.HasBuffFlag(buffs.Sleeping)` if they want to score themselves to 0 while sleeping. 4.2 doesn't filter on activity. |
| 13 | Mob dies and respawns | Goal list is per-template (4.1 contract) — survives automatically. Selection state (`current_goal_id`, round fields) also preserved; no special respawn hook. First post-respawn `Recompute` sees prev still valid in most cases, heldRounds spans the death window so min-hold gate trivially passes, same goal stays current unless a higher-scoring goal was added during the death window. If prev was removed during the death window, `switched_prev_invalid` branch picks fresh. Shared-template instances share selection state — accepted limitation. |

---

## 10. Testing strategy & rollout

### 10.1 Unit tests

`internal/goals/select_test.go` — pure `Select` table-driven tests:

- Empty goal list → nil, `no_goals`.
- Single goal, no prev → switch to it, `switched`.
- Single goal, same as prev → keep, `kept_top_unchanged`.
- Two goals, top != prev, gap ≥ margin, held ≥ min → switch.
- Two goals, top != prev, gap < margin → keep, `kept_hysteresis_margin`.
- Two goals, top != prev, gap ≥ margin, held < min → keep, `kept_hysteresis_min_hold`.
- Prev satisfied → `switched_prev_invalid`.
- Prev expired → `switched_prev_invalid`.
- Prev removed (not in goals slice) → `switched_prev_invalid`, no hysteresis applied.
- All goals filtered, prev still valid → keep, `kept_no_candidates`.
- All goals filtered, prev nil → nil, `no_goals`.
- Archetype weight elevates a lower-priority goal above a higher-priority one.
- `ContextScore` of 0 fully filters a goal.
- `ContextScore` multiplier > 1 elevates appropriately.
- Tie-break stability: equal effective scores → priority desc, id asc.
- Stale `currentSinceRound > nowRound` clamps `heldRounds` to 0 (no
  uint64 underflow); min-hold gate blocks; no panic.

`internal/goals/store_test.go` — `Recompute` integration tests (FS-spy fake):

- First call writes `current_goal_id` + round fields to disk.
- Subsequent no-switch call does NOT rewrite disk (compare mtime / write-call count).
- Switch call rewrites disk with new id + round.
- Emits `mudlog.Debug("goals.switch", ...)` only on switch (test hook on logger).
- `ContextScore` panic recovered; that goal scored 0; others continue.
- Tick-disabled config knob skips Recompute entirely (tested at the hook level).

`internal/goals/store_test.go` — Add/Remove/Clear integration:

- `Add` triggers eager `Recompute`; new high-pri goal becomes current
  immediately when prev was nil.
- `Remove(currentGoalId)` clears selection state under the same write-lock.
- `Clear()` zeros all three fields.

Archetype loader tests (existing archetype-package test file):

- `goal_weights:` parses to expected map.
- Missing `goal_weights:` field → empty map, no error.
- Malformed map → warning logged, empty map.
- `WeightsFor("nonexistent_archetype")` → empty map.

### 10.2 Admin command tests

`internal/usercommands/admin_goal_test.go` (extend existing):

- `goal current <mob>` happy path (with current), no-current path, no-goals path.
- `goal scores <mob>` table format with mixed candidate/filtered/CURRENT statuses.
- `goal scores <mob>` matches the actual `Select` result (no drift between display and engine).

### 10.3 Persistence round-trip

`internal/goals/persistence_test.go` (extend existing):

- File with the three new fields loads correctly.
- Old-format file (without new fields) loads with zero defaults.
- Round-trip preserves new fields.

### 10.4 Boot smoke (per CLAUDE.md pre-push SOP)

- Server starts; no panics from the new tick hook on mobs with empty goal lists.
- Existing 4.1 admin command exercise still passes.
- New `goal current` / `goal scores` round-trip on a test mob with seeded goals.
- Structured log line appears on engineered switch.

### 10.5 Out of scope for 4.2 tests

- Concrete `ContextScore` implementations — that's 4.3.
- Behavior-tree integration / selected-goal-driving-action tests — 4.4.
- Reactive event-driven goal generation tests — 4.5.

### 10.6 Rollout — feature branch decomposition

Single chunk on `feature/aliveness-4.2-goal-selection`, dispatched via
subagent-driven-development. Suggested task ordering for the plan:

1. **Schema delta** (`types.go` + `persistence_test.go`): add 3 new
   fields to MobGoals, round-trip test, legacy-file load test.
2. **Pure Select function** (`select.go` + `select_test.go`): no
   integration, just the function and exhaustive table tests.
3. **GoalTypeMeta.ContextScore hook** (`registry.go`): add field,
   default-nil handling, panic-recovery wrapper, tests.
4. **Archetype `goal_weights:` loader** (archetype package + its test
   file): YAML field, `WeightsFor` accessor, malformed-map handling.
5. **Recompute + log emission** (`store.go`): wires Select → cache +
   persistence + log, panic recovery.
6. **Eager recompute on Add/Remove/Clear** (`store.go`): wire into
   existing mutation paths.
7. **Tick hook**
   (`internal/hooks/MobRoundTick_RecomputeGoals.go`): registered with
   round-tick dispatcher, cheap-path early return, instance-aware
   Recompute call.
8. **Admin command extension** (`admin.goal.go`): `current` + `scores`
   dispatch cases. **Per `feedback-admin-command-wiring-checklist` in
   MEMORY.md, the plan must enumerate each wiring step explicitly:**
   - 8a. Add dispatch cases in `admin.goal.go`.
   - 8b. Update `command.goal.template` helpfile (already exists from
        4.1 — do not skip this thinking it's a no-op).
   - 8c. Confirm existing `goal` registration in `usercommands.go`
        resolves the new subcommands correctly (no new registration
        needed — same handler).
   - 8d. Unit test for each new subcommand.
   - 8e. Pre-push smoke: invoke `help goal`, `goal current <mob>`,
        `goal scores <mob>` interactively.
9. **Config knobs** (`config.balance.go`): `GoalSelectSwitchMargin`,
   `GoalSelectMinHoldRounds`, `GoalSelectTickEnabled` with defaults.
10. **Smoke checklist** (per pre-push SOP): boot test, admin command
    exercise on seeded test goals, `goal current`/`goal scores` output
    sanity, structured log line appears on engineered switch.

**Push to prod is safe** — selection runs but nothing reads its output
yet (4.4 wires btree). Observable change: admin command output + a
debug log line per switch. Zero player-facing impact.

### 10.7 Roadmap rollup after 4.2 ships

24/42. Next: 4.3 Goal types catalog (concrete revenge, wealth-target,
protection, etc. types with predicates and ContextScore functions) →
4.4 Strategic→tactical translation (where btree finally reads
`CurrentGoalOf`).

---

## File touch list

**New:**
- `internal/goals/select.go` — pure `Select` function + `SelectReason`,
  `ContextScoreFn` type.
- `internal/goals/select_test.go` — exhaustive table-driven tests.
- `internal/hooks/MobRoundTick_RecomputeGoals.go` — per-round-tick
  hook, cheap-path early return.

**Modified:**
- `internal/goals/types.go` — add `CurrentGoalId`, `CurrentSinceRound`,
  `LastSwitchRound` to `MobGoals`. Add `ContextScore` to `GoalTypeMeta`.
- `internal/goals/store.go` — add `CurrentGoalOf`, `Recompute`; wire
  eager `Recompute` into existing `Add`/`Remove`/`Clear`.
- `internal/goals/store_test.go` — add `Recompute` integration tests +
  Add/Remove/Clear eager-recompute tests.
- `internal/goals/persistence.go` + `persistence_test.go` — round-trip
  the three new fields; legacy-file load test.
- `internal/goals/registry.go` — add `ContextScore` field handling +
  panic-recovery wrapper.
- The archetype loader package (the one that parses
  `_datafiles/world/dogmud/behaviors/archetypes/*.yaml`) — add
  `goal_weights:` parsing + `WeightsFor` accessor + tests.
- `configs/config.balance.go` — `GoalSelectSwitchMargin`,
  `GoalSelectMinHoldRounds`, `GoalSelectTickEnabled`.
- `internal/usercommands/admin.goal.go` — `current` + `scores`
  dispatch cases.
- `internal/usercommands/admin_goal_test.go` — extend with new
  subcommand tests.
- `_datafiles/world/dogmud/templates/admincommands/help/command.goal.template`
  — update with `current` + `scores` documentation.
- `MOB_ALIVENESS_ROADMAP.md` — mark 4.2 done at ship time, rollup 24/42.

**Not touched in 4.2:** behavior-tree, schedules, conversations,
patrols, mob struct (selection passes the live `*mobs.Mob` instance
through but does not mutate it).
