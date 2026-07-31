# Code Smells & Refinement Candidates — context.md audit sweep

Recorded while auditing and rewriting every `internal/` and `modules/`
`context.md` on 2026-07-31, then worked through the same day.

Severity key: 🔴 correctness risk · 🟠 maintenance hazard · 🟡 tidy-up
Status key: ✅ fixed · ✏️ entry was wrong, corrected · ⏸️ deliberately not done

---

## ✅ Fixed

### 🔴 `dialogue` — four of ten `PlayerState` callbacks were called unguarded

`checkQuestGate`/`applyQuestEffects` nil-checked `GiveItem`, `SetQuestFlag`,
`BumpRep`, `GiveGold`, `HasOwnMasterwork` and `GetQuestFlag`, but invoked
`HasQuest`, `HasItem`, `RemoveItem` and `GiveQuest` directly — so a
partially-populated state panicked on those four and only those.

All ten are now guarded consistently. A missing *interrogative* callback fails
the gate **closed** (we cannot confirm the player holds a token, so we must not
conclude they do); the exception is `questExcluded`, where the same reasoning
runs the other way and the node stays available rather than silently vanishing.

Scope note: the original entry framed nil-`ps` as the danger. Both production
callers (`talk.go`, `ask.go`) build a fully-populated state via
`buildPlayerState`, so nil was only reachable from tests — the unguarded fields
were the real defect. Tests: `playerstate_guards_test.go`.

### 🟠 `questengine` — per-action `recover()` let a quest half-apply

Each action ran in its own `recover()` and the loop continued past a failure, so
a step could give the item but never set the flag — a state no author designed,
visible only as a log line.

A trigger's actions are now applied as a unit: a failed or panicking action
abandons the remaining actions of that trigger and logs how many were dropped.
There is still no rollback of what already ran; aborting stops it compounding.
A duplicate grant is still a skip, not a failure. Tests:
`action_abort_test.go`.

### 🟠 `gametime.AddPeriod` — real-day constant was `84600`, not `86400`

A digit transposition from the upstream import made every `x irl days` period
run 0.9% short (~13 minutes per day). Replaced both literals with a named
`secondsPerRealDay = 86400` carrying the history. In-flight real-time periods
now expire slightly later; nothing else depended on the old value.

### 🟠 `dialogue` — `memoryCache` was unbounded and never evicted

One entry per (mob instance × player) that had ever spoken, removed only by an
explicit `ResetMemory`, so it grew for the life of the process and held entries
for mob instances that despawned days earlier.

Added `SweepMemories()` (drops entries idle beyond `memorySweepIdleRounds`) and
`ForgetMobInstance(id)` (for despawn), wired into the existing turn maintenance
as `hooks.SweepDialogueMemory` on a 500-turn interval. Tests:
`memory_sweep_test.go`, including one that a large user id cannot bleed into
the mob-instance half of the packed key.

### 🟠 `dialogue.IsExpired` — magic baseline round to derive a duration

It computed a *length* by calling `AddPeriod` from an arbitrary origin
(`1000000`) and subtracting the origin back out. Added
`gametime.PeriodLength(periodStr) uint64`, which does that once and correctly,
and `IsExpired` now calls it.

### 🟠 `plugins` — exported-function ids are a flat global namespace

`GetExportedFunction` walks the registry and returns the first match, so two
modules exporting the same id resolved by registration order with no warning.
`ExportFunction` now panics on a duplicate id, naming both plugins — consistent
with the existing panic on a non-func argument, and a registration-time failure
rather than a runtime mystery.

### 🟡 `plugins` — writes before `Load()` land in `os.TempDir()`

`writeFolderPath` defaults to the OS temp dir until `Load` repoints it, so
anything persisted earlier is written where it will never be read back. Added
`writeFolderReady`; `WriteBytes` now warns loudly instead of failing silently.

### 🟡 `util.Save` defaulted to the unsafe write path

`Save(path, data, doSafe ...bool)` wrote directly unless the caller opted in.
Flipped: safe is now the default, `false` is an explicit opt-out. There is
exactly one call site (`configs.go:354`) and it already passed explicitly, so
behaviour is unchanged for it — only the default a new caller inherits.

### 🟡 `questengine.AllQuests` returned map iteration order

Now sorted by quest id. Callers print and diff these results; the order was
shuffling between runs for no reason.

### 🟡 `hooks.calcSpellDamageForCharacter` — redundant nil check

`caster != nil` re-tested inside a branch that already requires it. Removed.

---

## ✏️ Entries that were wrong

### `plugins.Requires()` — it **is** enforced

The original entry claimed nothing read the recorded dependencies. It was
written from the registration side only. `Load()` (`plugins.go:400-422`) checks
every dependency against the registry and drops the plugin with a
`mudlog.Error` if unmet.

The real sharp edge is narrower: the match is exact-string on both name and
version (`regCheckPlugin.version == dep.version`), so requiring `"1.0"` fails
against a plugin declaring `"1.0.0"`. The source already carries a
`// Later improve version matching.` TODO. Left alone.

### `mapper.ClearCache` — the name is fine

Flagged as colliding with identically-named symbols elsewhere. Six packages
define `ClearCache()` (`factions`, `crimes`, `opinions`, `goals`, `shops`,
`mapper`) and that is idiomatic Go — the package name is the namespace. The
difficulty is grep-based navigation, which `codegraph_node` already solves.
Renaming one of six would make things less consistent, not more. Not doing it.

### `characters/context.md` deprecation table

The ghost-symbol scan flagged `IsGrapplePosition()` / `IsGroundPosition()` /
`GetPositionColor()`. Those are correct: they appear in a migration table
mapping the retired `CombatPosition` API to its replacements, and the file
explicitly states no `IsBlinded()` predicate ships. Any future automated check
must skip old→new mapping tables or it will "fix" accurate history.

---

## ⏸️ Deliberately not done

### 🟡 `dialogue` — ten gating fields duplicated across three structs

`Pattern`, `TreeNode` and `QuestGreeting` each declare the same ten quest-gating
fields; `validate.go` already has a `gateFields` type acknowledging it.
Extracting an embedded `QuestGate` would remove ~30 declarations **but changes
YAML nesting** unless the embed is inlined, and these structs are the on-disk
schema for 186 authored dialogue files. Worth doing deliberately, with a
round-trip test over every dialogue file — not as a drive-by.

### 🟡 `questengine` — zero-valued trigger fields are wildcards

`matchTriggerFields` skips any filter that is `0`/`""`, so `mob: 0` matches
every mob rather than none. Fixing it properly means pointer fields or an
explicit unset sentinel, which changes the quest YAML schema and every trigger
in 79 quest files. Documented as a gotcha instead; revisit with the schema.

### 🟠 `internal/hooks` — 116 files behind one `RegisterListeners()`

Splitting registration per subsystem would cut merge conflicts and make the
surface readable. It touches the single most load-bearing function in the
engine and deserves its own change with a boot test, not a tail-end commit in a
docs sweep.

### 🟡 Codebase-wide lint backlog

The LSP surfaced a large set of pre-existing findings across `usercommands`,
`actions` and `hooks`: unused parameters, `simplifyrange`, `QF1003` tagged
switches, `S1003` `strings.Contains`, `S1039` unnecessary `Sprintf`,
`minmax`/`stringscutprefix` modernizations, and two unused symbols
(`logFollowConnectionIds`, `getShadowTargetUserId`). This is the existing
"lint modernization sweep" backlog item and wants one mechanical pass with
`golangci-lint --fix`, reviewed as a unit.

A second `tautological condition` at `NewRound_DoCombat.go:309` (`mobRoom !=
nil`) is genuine redundancy but sits in the combat hot path where the
provably-non-nil claim depends on flow the analyzer can see and a reader
cannot. Left alone.

### 🟡 Upstream `context.md` boilerplate

Deleted during the audit — 541 lines of "Future Enhancements", "Security
Considerations", "Scalability" and "Administrative Features" describing
capabilities that do not exist (SIMD UUID generation, distributed pathfinding,
AR navigation). Recorded here so the deletions are not mistaken for lost
content.
