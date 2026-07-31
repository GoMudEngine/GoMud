# Code Smells & Refinement Candidates — context.md audit sweep

Recorded while auditing and rewriting every `internal/` and `modules/`
`context.md` on 2026-07-31. **Nothing here has been acted on.** Each entry is
something noticed while reading the real source to document it; this file is
the review queue, not a changelog.

Severity key: 🔴 correctness risk · 🟠 maintenance hazard · 🟡 tidy-up

---

## Findings

### 🟠 `gametime.AddPeriod` — real-day constant is `84600`, not `86400`

`internal/gametime/gametime.go` (~line 317) computes
`roundsPerRealDay = 84600 / RoundSeconds`. A real day is 86400 seconds; this is
an upstream typo that makes every `x irl days` period **0.9% short**.

Not safe to silently fix: authored `irl` periods and any persisted timer
derived from one would shift. Needs a decision — correct it and accept the
one-time shift, or leave it and add a comment saying it is deliberate.

### 🟠 `plugins` — exported-function ids are a flat global namespace

`pluginRegistry.GetExportedFunction` walks every plugin and returns the first
match for a string id. Two modules exporting the same id silently resolve to
whichever registered first, with no warning. A duplicate check at registration
time would turn a silent mis-wire into a boot error.

### 🟡 `plugins` — `Requires()` records dependencies but nothing enforces them

`(*Plugin).Requires(name, version)` appends to `p.dependencies` and that slice
is never read for ordering or validation. Either enforce it at load or drop the
API so it stops implying a guarantee.

### 🟡 `plugins` — plugin writes land in `os.TempDir()` until `Load()` runs

`writeFolderPath` defaults to `os.TempDir()` and is only repointed by
`Load(dataFilesPath)`. Anything persisted before `Load` is written to temp and
lost. Worth either failing loudly on a pre-`Load` write or setting the path at
package init.

### 🟡 `mapper` — `ClearCache` is one of several identically-named symbols

`mapper.ClearCache` collides by name with `ClearCache` in other packages. Not a
bug, but it makes grep-based navigation unreliable and has already caused
confusion (noted in the project's codegraph guidance). A rename to
`ClearMapperCache` would be a cheap readability win.

### 🔴 `dialogue` — a `nil` `*PlayerState` silently disables every quest gate

`checkQuestGate` treats a nil `PlayerState` as "no checks apply," so a caller
that forgets to pass one gets an NPC that hands out every quest, ignores every
exclusion, and skips masterwork/item/flag requirements — with no error. The
comment calls it backward compatibility. Consider either requiring a non-nil
state on the production paths or splitting the permissive behaviour into an
explicit `NoGating()` sentinel so the intent is visible at the call site.

### 🟠 `dialogue` — `memoryCache` is unbounded and never evicted

`memory.go` holds `map[uint64]*PlayerMemory` keyed by
`mobInstanceId<<32 | userId`, with entries created on demand and only ever
removed by an explicit `ResetMemory`. Every mob instance a player has ever
spoken to accumulates for the life of the process. Small per entry, but it
grows monotonically with uptime and player count and nothing sweeps it when a
mob instance despawns. A despawn hook or a periodic sweep keyed on
`LastVisitRound` would bound it.

### 🟠 `dialogue.IsExpired` — magic baseline round to derive a duration

```go
baseline := gametime.GetDate(1000000)
expiryRound := baseline.AddPeriod(expiryPeriod)
delta := expiryRound - 1000000
```

This computes a *duration* by asking for an absolute round from an arbitrary
fixed origin and subtracting it back out. It works, but it is a workaround for
`AddPeriod` only returning absolute rounds. A `gametime.PeriodLength(periodStr)
uint64` helper would remove the magic number here and probably at other call
sites.

### 🟡 `dialogue` — ten gating fields duplicated across three structs

`Pattern`, `TreeNode`, and `QuestGreeting` each declare the same ten
quest-gating fields. `validate.go` already has a `gateFields` type that
acknowledges the duplication. Extracting an embedded `QuestGate` struct would
remove ~30 duplicated field declarations — but it changes YAML nesting unless
the embed is inlined, so it needs care.

### 🟠 `questengine` — per-action `recover()` lets a quest half-apply

`executeActions` wraps each action in its own `recover()` and continues to the
next action on panic *or* error, logging only. A quest step can therefore give
the item but never set the flag, leaving the player in a state the content
author never designed. Consider aborting the remaining actions of a trigger
once one fails, or marking the trigger's effects as a unit.

### 🟡 `questengine` — zero-valued trigger fields are wildcards

`matchTriggerFields` skips any filter whose value is `0`/`""`. Authoring
`mob: 0` therefore matches every mob rather than none. A distinct
"unset" sentinel (or pointer fields) would make over-broad triggers impossible
to write by accident.

### 🟡 `questengine.AllQuests` returns map iteration order

Callers that print or diff results need determinism. Either sort inside
`AllQuests` or document the requirement at every call site.

### 🟠 `internal/hooks` is 116 files with one registration function

`RegisterListeners()` in `hooks.go` wires every listener in a package of 116
non-test files. It works, but it means the only way to discover what the engine
reacts to is to read one long function, and any merge touching two hooks
conflicts there. Grouping registration per subsystem (combat, quests, economy,
NPC life) into a few `registerXxxListeners()` calls would cut conflicts and make
the surface readable. No behaviour change.

### 🟡 `util.Save` defaults to the unsafe path

`Save(path, data, doSafe ...bool)` writes directly unless the caller opts in,
while `SafeSave` does the temp-file-and-rename dance. The safe behaviour should
arguably be the default, with an explicit opt-out for the few hot paths that
need it — the current shape means a new caller gets the risky version by
omission.

### 🟡 Detector note: deprecation tables read as fabricated APIs

The ghost-symbol scan used in this audit flags `internal/characters/context.md`
for `IsGrapplePosition()` / `IsGroundPosition()` / `GetPositionColor()`. Those
are **correct** — they appear in a migration table mapping the retired
`CombatPosition` API to its replacements, and the file explicitly states that
no `IsBlinded()` predicate ships. Any future automated check needs to skip
old→new mapping tables, or it will "fix" accurate history.

### 🟡 Upstream `context.md` boilerplate is being deleted wholesale

The upstream-generated docs pad every package with "Future Enhancements,"
"Security Considerations," "Scalability," and "Administrative Features"
sections describing capabilities that do not exist (SIMD UUID generation,
distributed pathfinding, AR navigation). These are being removed as each file
is rewritten. Listed here so the deletions are not mistaken for lost content.
