# DOGMud Codebase Tech-Debt Audit — 2026-07-20

**Scope:** all Go source outside `vendor/`, `.claude/`, `_datafiles/`.
~69,500 LOC non-test across 123 packages, plus ~19,500 LOC of tests in 675 files.

**Method:** six parallel specialist audits (error handling, concurrency, duplication,
modern-Go idioms, architecture/readability, tests/tooling), each required to read the
code behind every claim. Findings below were then de-duplicated and cross-referenced.
Every claim is tagged with its verification status:

| Tag | Meaning |
|-----|---------|
| ✅ **VERIFIED** | Confirmed directly by reading the code / running the tool during this audit |
| 🔍 **REPORTED** | Audited and read by a specialist agent, not independently re-confirmed |
| ⚠️ **NEEDS PROOF** | Plausible from static reading, but requires a runtime check to confirm |

> A note on tone: this codebase is in **good** shape for its size and age. The sections
> below are a debt register, not an indictment. Read "Baseline health" first so the rest
> lands in proportion.

**Companion document:** [`TEST_COVERAGE_AUDIT.md`](TEST_COVERAGE_AUDIT.md), regenerated the
same day. Repo coverage is **42.3%**; Tier-1 gameplay packages average **60.4%**, up from 24%
in February. Tier 6 below summarizes CI/test tooling; the coverage doc holds the per-package
table, the skipped-test analysis, and the ranked list of tests to write. Note that **four of
the eight Tier 0 bugs below (0.1, 0.2, 0.6, 0.7) sit in code with no test coverage at all** —
that correlation is the argument for the coverage work, not a coincidence.

---

## Execution status — 2026-07-21

This audit was executed over 2026-07-20 – 07-21. **This ledger is the source of truth for
what actually happened**; the per-item sections and the "Recommended sequence" at the bottom
are tagged to match it. When they disagree, this table wins.

Legend: ✅ done · ◐ partial (intentionally scoped down) · ⏸️ withdrawn/descoped by decision
(rationale in the linked section) · ▶️ remaining.

**Phase 1 — Stop the bleeding — ✅ complete**
- ✅ 1.1 panic recovery in the listener dispatch loop (`c4ab3f50e`).
- ✅ 0.1–0.8 all eight Tier-0 correctness bugs (`52a766f50`), plus a bonus statmods nil-map
  fix (`446d211ab`).
- ✅ 6.4 #1 `go vet` + `gofmt` gates in CI (`e044a5424`, `2dfbf5d49`, `c508d5933`).
- ✅ Mechanical cleanups: lumberjack dedup (5.2); `sliceContains`/`max1` removal, Discord
  client timeout, web.go mudlog bypass (5.4) (`733d691c7`).

**Phase 2 — Make the invariants enforceable — ✅ mostly complete**
- ✅ 6.2 data-file boot smoke test in CI (`c6f19296e`).
- ✅ 2.1 / 2.2 / 2.3 concurrency locking — copyover, autocomplete, per-session user state
  (`4799200f7`, `b61bf45cf`).
- ✅ 2.4 `RLockMud` — **resolved as part of 2.2** (see §2.4). `GetAutoComplete` (`world.go:319`)
  takes `RLockMud` — a genuine read-only path — so the pair is live code and `mudLock` is a
  real `RWMutex`, exactly the "autocomplete is the first read-lock customer" outcome the plan
  called for. (An earlier draft of this ledger wrongly marked it open — the verifying grep was
  scoped to `internal/` and missed the caller in root-level `world.go`.)
- ✅ 1.3 quest-sequence goroutine hardening (`b92692859`).
- ⏸️ 5.1 yaml.v2 → v3 — **withdrawn after measurement** (see §5.1). Replaced by a dialogue
  parse gate (`685953f80`) + a silently-ignored-key drift gate (`769ab4fcc`).

**Phase 3 — Pay down structure — partially done; several items descoped by decision**
- ✅ 3.1 `AcquireMeleeTarget` melee-preamble extraction (`ef93d2d17`).
- ⏸️ 3.4 `attack` → `actions/` full migration — **descoped.** The aggro-type drift (0.5) was
  fixed and engagement-aggro typing centralised (`b23d34a15`); the ~480-line mechanical
  migration was not judged worth the churn.
- ✅ 3.3 combat-text darkness gating — **already resolved; premise disproven 2026-07-21**
  (see §3.3). The whole combat-text surface (usercommand special moves — 40 gated calls, zero
  raw broadcasts — plus the auto-attack round) already routes room narration through the
  visibility-gated `SendTextVisual` pipeline, with a dark-room "you hear fighting" fallback.
  The proposed `SendCombatExchange` helper is obsolete. No code needed.
- ⏸️ 3.6 hook combinators — withdrawn; premise disproven (see §1.2; `01e985be9`).
- ⏸️ 4.2 `internal/util` split — **descoped.** The cheap, clearly-right part was done:
  `GetMyIP` / `net/http` + dead code dropped (`25132e6e5`). The larger dice /
  `GetLockSequence` / fileutil / term split was judged not worth the blast radius.
- ✅ 4.3 wiring-seam documentation (`c713cdede`).
- ⏸️ 4.1 break up `Go()` / `Get()` — **descoped.** The highest-value `unlockExit` extraction
  landed (`dec87613b`); the full god-function breakup was left alone to avoid churning
  heavily-trafficked code for modest gain.
- ✅ 6.4 #5 `.golangci.yml` gate (`ee4d435d5`; see §6.4).

**Beyond the original plan:** generic quest reward-key fix (`3f390ca0e`); grapple crit-failure
flake + sibling knockdown nil-guards (2026-07-21).

**Still open / carried forward:** only the "Explicitly deferred" list at the end of this
document (4.6 DI, 4.5 file-naming, doc-comment backfill, 5.3 `%w` backfill, `t.Parallel()`).
With 3.3 disproven and 2.4 resolved, **no scheduled Tier 0–4 or Phase 1–3 work remains** — the
audit is fully worked through.

---

## 0. Baseline health — what is already right

Establishing this matters, because several items below are "keep it that way" rather than
"fix it."

- ✅ `go build ./...` clean; `go vet ./...` clean **with default analyzers** (CI only runs
  the weaker `-composites=false` variant, so the codebase is cleaner than CI demands).
- ✅ Full test suite passes (`go test ./...`, exit 0).
- ✅ CI **does** run the race detector and a coverage gate — `.github/actions/codegen-and-test`
  runs `go test -timeout 300s -race -coverprofile=coverage.out ./...`, on both PRs and
  pushes to master. (I initially assumed otherwise; the tooling audit corrected me.)
- ✅ Dependabot is configured (`.github/dependabot.yml`, gomod, weekly).
- ✅ TODO/FIXME/HACK density is **29 total** across the whole tree — genuinely low.
- ✅ No `io/ioutil`, no `os.SEEK_*`, no `rand.Seed`, no `*_v2.go`/`*_deprecated.go`
  scaffolding. Several "modern Go" migrations are already finished.
- ✅ `interface{}` → `any` is ~98% done (1611 `any` vs 41 `interface{}`, and 26 of those
  41 are deliberately-frozen schema snapshots in `internal/migration/`).
- 🔍 Structured logging is real and consistent: `internal/mudlog` wraps `log/slog`, called
  798 times across 218 files, with only **one** genuine bypass.
- 🔍 Test quality in the core game-math packages (`internal/dice`, `internal/combat`,
  `internal/characters`) is high — Monte Carlo distribution tests, table-driven subtests,
  named regression tests that cite the bug they guard. This is the bar for everything else.
- 🔍 The `internal/actions/` actor-parity refactor genuinely worked for melee specials:
  11 of ~19 combat verbs are fully unified behind the `Actor` interface.

---

## Tier 0 — Correctness bugs found during the audit

These are defects, not style. They are small and surgical; they should not wait on any
refactor.

### 0.1 Mobs create gold from nothing via `put` — and bypass container locks
✅ **VERIFIED** (read both files side by side)

- `internal/mobcommands/put.go:74-77` — `container.Gold += goldAmt` with **no**
  `mob.Character.Gold -= goldAmt` and **no** affordability check.
- `internal/usercommands/put.go:98-114` — the player path correctly checks
  `goldAmt > user.Character.Gold`, then debits, then credits.
- Additionally, the player path checks `container.Lock.IsLocked()` at
  `internal/usercommands/put.go:60-65` and refuses; **the mob path has no lock check at all.**

**Impact:** any mob that executes `put <n> gold` mints currency into a container. `"put"` is
registered in the mob command table (`internal/mobcommands/mobcommands.go:69`), so it is
reachable from behavior trees / idle command pools. Whether it is *currently* reached by
authored content is a separate question — a content grep found only 4 candidate files, so
live exploitation is likely low, but the code path is open.

**Fix:** extract `actions.PutItem` / `actions.PutGold` mirroring the existing
`DropItem`/`GetItemFromFloor` pattern, with the lock check and gold debit inside the shared
function. (This is also duplication item 3.2.)

### 0.2 Players get no crafting-skill progression from instant recipes
🔍 **REPORTED**

`internal/mobcommands/craft.go:49-55` calls `OnSkillUseScaled` on the `ImmediateComplete`
path. The player-side equivalents — `internal/usercommands/craft.go:121-123` and
`completeCraft()` at `internal/usercommands/craft.go:534-539` — do not. Multi-round crafts
are symmetric; only the `TimeRounds <= 0` (instant) path diverges.

**Impact:** silent under-reward. Players building crafting skill via instant recipes gain
nothing; NPCs do. No error, no log line.

**Fix:** move the `OnSkillUse` call into `actions.InitiateCraft` for the `ImmediateComplete`
case so both callers get it, rather than patching the two player sites.

### 0.3 Mob shouts don't propagate through mutator/temp exits
🔍 **REPORTED**

`internal/usercommands/shout.go:51-79` propagates through `room.Exits`, `room.ExitsTemp`,
**and** `room.ActiveMutators`-added exits. `internal/mobcommands/shout.go:29-35` walks only
`room.Exits`.

**Impact:** a mob alarm or boss shout in a room with mutator- or temp-added exits fails to
alert adjacent rooms that an equivalent player shout would reach.

### 0.4 Mobs never build Rhetoric from `warcry`/`rally`
🔍 **REPORTED**

`internal/usercommands/warcry.go:93-100` and `rally.go:92-97` call `OnSkillUse(Rhetoric)`.
`internal/mobcommands/warcry.go` and `rally.go` never do. These two verbs are the only
migrated verbs that leave `OnSkillUse` to the caller instead of putting it inside
`actions/combat_*.go`, and the mob caller simply never implemented it.

**Fix:** move the call into `ExecuteWarcry`/`ExecuteRally`, matching the other 11 verbs.

### 0.5 Stealth-opening attacks tag aggro differently for players vs mobs
🔍 **REPORTED**

`internal/usercommands/attack.go:196-209` always sets `Aggro.Type = DefaultAttack`, even on
the `SurpriseAttack` path. `internal/mobcommands/attack.go:68-80` sets
`aggroType = SurpriseAttack` when the mob is hidden. Downstream combat code that keys off
`Aggro.Type` (crit bonuses etc.) therefore treats player and mob stealth openers differently.

### 0.6 ⚠️ Players can loot locked containers — `get` is the only command that skips the lock check
✅ **VERIFIED** (grep + read; confirmed against every sibling command)

`internal/usercommands/get.go` contains **no lock check of any kind** — `grep -n "Lock\|IsLocked"`
returns nothing for the whole file. The container withdrawal path at `get.go:451` does:

```go
container := room.Containers[containerName]
...
goldAmt := container.Gold
user.Character.Gold += goldAmt
container.Gold -= goldAmt
```

Every other command in the family gates correctly on `container.Lock.IsLocked()`:
`look.go:174,670` · `put.go:60` · `lock.go:43` · `unlock.go:38` · `picklock.go:75`.

**Impact:** a player cannot *look inside* a locked chest, but can `get gold from <chest>` and
empty it. Picking the lock is entirely optional. Note the near-miss that makes this easy to
overlook: `get.go:79` **does** check `c.Hidden`, so hidden containers are gated — only the
*lock* is ignored. This makes `picklock` skill progression and every locked-chest reward
bypassable.

**Fix:** add the `put.go:60-65` lock check to `get.go`'s container branch. Then fold both into
the shared `actions` helper (see 3.2) so the two paths cannot drift again — this bug and 0.1
are the same root cause in opposite directions.

### 0.7 `auctions.GetAuctionHistory` panics on any partial request
✅ **VERIFIED** (read `modules/auctions/auctions.go:1176-1187`)

```go
return am.PastAuctions[len(am.PastAuctions)-totalItems : totalItems]
```

The high bound should be the slice end, not `totalItems`. With 10 past auctions and
`totalItems=3` this evaluates to `PastAuctions[7:3]` — low > high — a guaranteed
`slice bounds out of range` panic. It fires whenever `totalItems < len(PastAuctions)/2`.

**Currently dormant:** the sole caller (`auctions.go:242`) passes `0`, which returns early on
the `totalItems < 1` guard. But this is an exported method on a public manager, so any future
"show last N auctions" feature detonates it. Given Tier 1's finding that panics are unrecovered,
this would take down the server.

**Fix:** `return am.PastAuctions[len(am.PastAuctions)-totalItems:]` — one character class of
change, plus the regression test.

### 0.8 The release workflow tags every master push with a tag literally named `master`
✅ **VERIFIED** (read `.github/workflows/build-and-release.yml`)

`RELEASE_VERSION: ${{ github.ref_name }}` (line 13) resolves to `master` on a push to master,
and is passed to `softprops/action-gh-release` as `tag_name` (line 81).

**This is the root cause of the recurring stray `refs/tags/master` that has to be deleted
after every push.** It is not a git quirk — CI recreates it every time.

**Fix:** gate the release job on `startsWith(github.ref, 'refs/tags/')`, or derive the version
from a real tag / commit SHA rather than `ref_name`.

---

## Tier 1 — Crash safety (highest structural risk)

### 1.1 The main game loop and the per-connection loop have zero panic recovery
✅ **VERIFIED** (read `internal/events/listeners.go`, `world.go`, `main.go`)

- `main.go:496` spawns `go worldManager.MainWorker(...)`. `MainWorker`'s only defer
  (`world.go:719-722`) is a log line + `wg.Done()` — no `recover()`.
- The chain `MainWorker → EventLoop → events.ProcessEvents → DoListeners` reaches
  `lw.listener(e)` (`internal/events/listeners.go:166,177`) with **no recover anywhere**.
  That single dispatch point executes every combat round, quest event, command execution,
  and mob AI tick — a surface spanning ~150 files.
- `main.go:1480` spawns `go handleTelnetConnection(...)`; its only defer is `wg.Done()`
  (`main.go:628-630`). This goroutine processes **untrusted network input** through the
  telnet/login/ANSI handler chain.

**Impact:** any nil deref, index-out-of-range, or failed type assertion anywhere in gameplay
code or connection handling takes down the entire process and disconnects every player.

**This is not a stylistic gap — the team already knows the pattern.** Targeted recover
wrappers exist at `internal/behaviortree/actions_goal.go:102` (`invokePlannerSafely`),
`internal/goals/select.go:193`, `internal/goals/store.go:555`, and
`internal/behaviortree/engine.go:282`, each commented about not crashing the engine round
tick. The discipline was applied to individual callbacks but never to the dispatch loop
they all run inside. Of 18 non-test `recover()` sites, the ones that exist protect the
*least* trafficked goroutines (web listeners, Discord, LLM client) while the two that
execute gameplay and handle network input have none.

**Fix (small, high leverage):** wrap the per-listener call in `DoListeners` in a
`defer/recover` that logs the panic + event type and continues to the next listener; add
the same to `handleTelnetConnection` so a bad connection disconnects one player instead of
killing the server. This one change converts most of Tier 2 and 3's latent panics from
"outage" to "one failed round."

### 1.2 Unchecked type assertions across hooks and combat
⚠️ **SEVERITY DISPROVEN — 2026-07-20.** The panic mechanism described below does not exist.

> The claim was that ~14 unchecked `evt := e.(events.X)` assertions are "one uncomment away
> from firing", because `internal/events/listeners.go` has a wildcard-listener facility and
> `hooks.go` carries a commented-out `RegisterListener(nil, ...)` debug hook.
>
> **Tested empirically: enabling it would not affect them.** A nil/`*` registration lands in its
> own `eventListeners["*"]` bucket, which `DoListeners` walks *separately* from
> `eventListeners[e.Type()]`. A wildcard listener receives every event; type-specific listeners
> keep receiving only their own type. Verified by registering both and dispatching a foreign
> event: the wildcard saw it, the typed listener saw nothing. Pinned by
> `TestDispatchRoutesOnlyMatchingTypes`.
>
> Independently, finding 1.1's fix means any panic that *did* occur is now caught and logged by
> `invokeListenerSafely` rather than killing the server. The class is doubly mitigated.
>
> **The `Cancel` vs `Continue` sub-finding was also mis-stated.** It described
> `RoomChange_LocationMusicChange.go` as diverging "from every sibling". It is not one outlier —
> the split is roughly **18 `Cancel` vs 23 `Continue`** across 41 type-assertion branches. There
> is no convention, rather than one file breaking it. Since the branch is unreachable, the
> convention is now documented at the `ListenerReturn` declaration (use `Continue`) instead of
> churning 32 files for a branch that cannot execute.
>
> **The 3.6 combinator refactor is therefore not carried out.** With no bug behind it, it reduces
> to ~110 lines of boilerplate across 14 self-registering files — real, but not worth the churn
> ahead of the items that fix actual defects.

🔍 **REPORTED** (original text follows, retained for the file/line inventory)

- 14 hook files use the unchecked `evt := e.(events.X)` form vs ~55 that use the checked
  `evt, ok := ...; if !ok { return }` form. Samples: `NewRound_AutoHeal.go:30`,
  `NewRound_DoCombat.go:25`, `NewRound_PresenceTick.go:24`, `MobIdle_HandleIdleMobs.go:36`,
  `RoomChange_CleanupEphemeralRooms.go:15`.
- These are safe *only* because dispatch is keyed by exact event type. But
  `internal/events/listeners.go:136-144` contains a live **wildcard-listener facility**
  (`RegisterListener(nil, ...)`), currently commented out at `hooks.go:136-143`.
  **Re-enabling it for debugging turns all 14 into guaranteed panics** on the first
  non-matching event.
- `internal/inputhandlers/login_prompt_handler.go:183` —
  `state = stateVal.(*PromptHandlerState) // We assume it's the correct type`, a
  single-value assertion on a `map[string]any` value, running inside the unrecovered
  connection goroutine from 1.1.
- `asUser`/`asMob` (`internal/hooks/NewRound_DoCombat_unified.go:913-927`) are called 14
  times in the core per-round combat resolver, relying on a manual `IsPlayer()`/`IsMob()`
  gate rather than the type system.
- `RoomChange_LocationMusicChange.go:17-21` checks the type but returns `events.Cancel`
  instead of `events.Continue` like every sibling — aborting the rest of the listener chain
  on a type mismatch. A behavioral inconsistency and a landmine for anyone cloning it as a
  template.

**Fix:** standardize on the checked form via `onX`-style combinators (see 3.6), and fix the
`Cancel`/`Continue` split. Lower urgency once 1.1 lands, but 1.1 does not make these correct
— it makes them survivable.

### 1.3 Quest dialogue-sequence goroutines: unrecovered *and* unlocked
🔍 **REPORTED** (flagged independently by both the error and concurrency audits)

`internal/questengine/bridge.go:401-406` and `:420-437` (inside `GameBridge.QueueSequence`)
schedule delayed work with bare `go func(){ time.Sleep(...); ... }()`, then call
`u.SetTempData(...)` and `ExecuteAction(action, bridge)` — arbitrary quest actions that
mutate `Character`, grant items, move players.

Three problems in one place:
1. **No `recover()`** — a panic here (e.g. nil room because the player logged out during the
   delay) crashes the server.
2. **No `util.LockMud()`** — mutates live game state from a goroutine that never takes the
   world lock, while `MainWorker` may be mid-tick on the same objects.
3. **No shutdown check** — a sequence queued shortly before shutdown fires after
   `users.SaveAllUsers()` has run, silently losing the mutation.

**The correct pattern already exists in-repo:** `internal/llm/client.go:51-53,161-163`
explicitly documents "`onResponse` is called under `util.LockMud()` — safe to call mob/room
functions directly" and does exactly that, with a recover.

**Why this ranks high:** this path is driven by **content** (quest YAML), not code. Given the
project's documented history of content-authoring footguns, a malformed quest action reaching
here is a far more likely trigger than a code bug.

**Fix:** wrap in `LockMud`/`recover` following `internal/llm/client.go`, or better — move
sequence delivery onto the event queue with a `ReadyTurn` so it participates in the tick loop
naturally instead of a bare timer goroutine.

---

## Tier 2 — Concurrency

### The model as it stands
🔍 **REPORTED**

DOGMud is a *single-writer tick engine wrapped in a big lock*. `MainWorker` (`world.go:714`)
drives a `select` loop and wraps essentially every case body in `util.LockMud()`/`UnlockMud()`
(a package-level `sync.RWMutex`, `internal/util/util.go:70`). Player commands funnel into the
same goroutine: each connection has its own read-loop goroutine, but it only *sends* input over
a channel; execution happens synchronously inside `MainWorker` under the lock.

The three core entity packages (`users`, `rooms`, `mobs`) each guard their top-level map with a
proper `RWMutex`, used correctly. **That tier is solid.**

**The gap is the middle tier:** once a caller holds a live `*UserRecord` / `*Room` / `*Mob`
pointer out of one of those maps, the struct itself has **zero internal synchronization** — and
several paths read/write those pointers from connection goroutines that never take
`util.LockMud()`.

### 2.1 `UserRecord` prompt/tempdata fields raced between tick loop and connection goroutines
⚠️ **NEEDS PROOF** (structurally verified; needs a `-race` run against a live session to confirm)

`UserRecord.unsentText`, `.suggestText`, `.activePrompt`, and `.tempDataStore` are plain
fields/maps with no mutex (`internal/users/userrecord.go:470,483,590,596,681,686`).

- Written/read **unlocked** from the telnet loop at `main.go:823,835,842,858,864,869,971,994,998,1005,1027`
  and mirrored WebSocket paths.
- Written/read **under `util.LockMud()`** from `world.go:926` (`user.ClearPrompt()` inside
  `processInput`) and `internal/hooks/RedrawPrompt_SendRedraw.go:22,34,40,48`.

`tempDataStore` is a `map[string]any` — concurrent map read+write is a hard runtime panic in Go,
not a benign race.

**Fix:** give `UserRecord` its own mutex covering these specific accessors. Cheaper and safer
than rerouting prompt interaction through the input channel.

### 2.2 `GetAutoComplete` reads world state unlocked from the connection goroutine
✅ **VERIFIED** (read `main.go:820-840`; confirmed no lock in the call chain)

`world.go:304` `GetAutoComplete` is **410 lines** and walks `room.Exits`, `room.Containers`,
`room.GetMobs()`, `mob.Character.Name`, `user.Character.GetSpells()` — all with no lock. It is
called from `main.go:830` and `main.go:1119` directly on the per-connection goroutine, on every
Tab keypress, while `MainWorker` concurrently mutates those same objects under the lock.

`for exitName := range room.Exits` iterating while `MainWorker` mutates `room.Exits` during
ephemeral-room cleanup is a `concurrent map iteration and map write` fatal error.

**Fix:** cheapest correct fix is to take `util.LockMud()` (or `RLockMud`, see 2.4) around the
call — it's tab-press-driven and infrequent, so the contention cost is negligible.

### 2.3 `triggerCopyover()` skips the world lock on the signal path
🔍 **REPORTED** — **relevant to the already-flagged "hot-reboot UNTESTED" risk**

`copyover.go:20-39` calls `rooms.SaveAllRooms()`, `users.SaveAllUsers()`, `shops.SaveAllShops()`
with **no** `util.LockMud()`. Compare `world.go:754-767`, which wraps the identical sequence in
the lock during graceful shutdown.

When triggered by `SIGUSR1` (`copyover_signal_unix.go:18-25`) it runs on its own goroutine,
concurrent with a mid-tick `MainWorker` — risking a fatal map-iteration panic during YAML
marshal, or a torn save that corrupts the copyover handoff.

**Subtlety that makes this non-trivial:** when triggered by the `copyover` *admin command*, it
runs already inside the locked `EventLoop()`. `sync.RWMutex` is not reentrant, so naively adding
`LockMud()` inside `triggerCopyover()` **deadlocks that path.** The two call sites need different
treatment — e.g. a `LockMud`-wrapping entry point for the signal handler, and the bare function
for the command path.

Also: `copyover.go:28` discards the room-save error (`_ = rooms.SaveAllRooms()`) right before the
process re-execs, so a failed save is invisible.

**This should be resolved before the pending droplet hot-reboot validation**, since it's exactly
the failure mode that test would hit.

### 2.4 `RLockMud`/`RUnlockMud` are dead code — the RWMutex is really a Mutex
✅ **RESOLVED — and the original finding was factually wrong, 2026-07-21.** The "zero non-test
call sites" claim was a **grep-scope error**: the search only covered `internal/`, but the
caller is in root-level `world.go`. `GetAutoComplete` (`world.go:319`) takes `util.RLockMud()`
on a genuinely read-only path (it walks room/mob/character state to build tab-completions and
mutates nothing), which is precisely the "make it live by using it on the autocomplete read
path" resolution the fix below and the Phase-2 sequence both recommended. `RLockMud`/`RUnlockMud`
are live code and `mudLock` is a real `RWMutex`. Nothing to do; downgrading to `sync.Mutex`
would be a regression (it was attempted 2026-07-21 and the build immediately flagged the
`world.go` caller).

> Original finding below is retained but **superseded** — its `✅ VERIFIED` tag was the grep-scope
> mistake, a caution that "verified" is only as good as the search that backed it.

🔍 **Original text:** `internal/util/util.go:90-96` defines `RLockMud`/`RUnlockMud`; the only
references anywhere are the definitions themselves and `internal/util/util_test.go:37-38`. Every
real caller uses the exclusive `LockMud()`.

**Impact:** read-only operations (admin dashboard GETs, autocomplete) needlessly serialize
against each other, and the type signals a read-concurrency capability the codebase doesn't have.

**Fix:** pick one — either delete them and simplify to `sync.Mutex`, or start using them on
genuinely read-only paths (2.2 is the obvious first customer).

### 2.5 Economy-snapshot goroutine isn't registered with the WaitGroup
🔍 **REPORTED**

`main.go:520-551` selects on `workerShutdownChan` correctly but is never `wg.Add(1)`-registered,
so `wg.Wait()` doesn't wait for it. Low practical risk (a trailing `time.Sleep(1s)` likely covers
it) but inconsistent with the pattern used by every other worker.

---

## Tier 3 — Duplication and drift

The `internal/actions/` unification is real and worked well for melee specials. The debt is in
what it *didn't* reach. **Note that Tier 0 items 0.1–0.5 are all drift bugs — they are the
symptom this section describes.**

### 3.1 Melee special-move target-acquisition preamble copy-pasted 11×
🔍 **REPORTED** — ~440 lines, largest single consolidation

`bash.go:16-56`, `kick.go:18-58`, `grapple.go:15-55`, `trip.go:17-57`, `taunt.go:18-62`, plus
`gore`, `maul`, `pounce`, `rake`, `throttle`, `drain` (11 files in `internal/usercommands/`).

Each repeats: crafting guard → empty-target check → `actions.ResolveTargetActor` → on error, a
**second redundant `room.FindByName` scan** purely to distinguish "can't target yourself" from
"not here" → non-combatant/PvP-immune branch → `SetAggro`. Verified byte-identical across five
side-by-side reads apart from verb strings.

No drift *yet* — but the pattern is still actively growing as new mutation-kick variants clone it.

**Fix:** `actions.AcquireMeleeTarget(user, room, rest, verb) (Actor, bool, error)`. Each file
collapses to 3–4 lines. Also removes a redundant room scan per invocation.

### 3.2 `put` — full duplication, no `actions/put.go` at all
See **0.1**. ~230 lines duplicated; contains the gold-dupe and lock-bypass.

### 3.3 Combat messaging duplicated in `usercommands`, missing darkness gating
✅ **RESOLVED — premise disproven on re-verification, 2026-07-21.** The "information leak" half
of this finding does not hold, and likely didn't when the audit was written (a 🔍 REPORTED item
the specialist never runtime-verified). The messaging-pipeline refactor already closed it:

> - Every combat/special-move room broadcast in `internal/usercommands/` — **40 calls across 15
>   files** (`attack`, `trip`, `taunt`, `grapple`, `bash`, `kick`, `gore`, `maul`, `pounce`,
>   `rake`, `throttle`, `drain`, `shoot`, `warcry`, `rally`) — goes through the visibility-gated
>   `room.SendTextVisual`. A full sweep found **zero** raw `room.SendText` calls in any combat
>   file (the only raw calls in the whole package are non-combat: merchant speech, room redesc).
> - `SendTextVisual` → `messaging.RenderForRecipient` with `ChannelVisual` returns `""` for a
>   `SightNone` recipient (`pipeline.go:62-63`), so a sightless observer is fully suppressed —
>   the opposite of over-shared.
> - The auto-attack round is gated identically: `AttackResult` room messages flush via
>   `sendVisualRoomText` → `SendTextVisual` and `drainSpectatorLines` → `SendTextVisualToUser`,
>   plus a deliberate `sendDarkRoomCombatFallback` that sends "You hear the sounds of fighting
>   nearby" to sightless observers instead of the combat detail.
>
> The proposed fix (`messaging.SendCombatExchange` porting `mobcommands/darkness.go`) is
> therefore obsolete — the darkness logic now lives in the messaging pipeline, not in
> `mobcommands`. The *duplication* half (each command hand-rolls the three-way
> attacker/defender/room send) is real but minor, shares 3.1's spirit, and isn't worth a
> dedicated helper on its own. **No action.**

🔍 **REPORTED** (original text follows, retained for context) — ~150–200 lines, **and a real information leak**

Every hit/miss/crit branch across ~10 usercommand files hand-rolls "tell attacker / tell
defender / tell room" as three separate `SendText` calls (`trip.go:95-157`, `taunt.go:130-217`, etc.).

`internal/mobcommands/darkness.go` already extracted this into `sendRoomText`/`sendAudioRoomText`
— used **56 times across 21 mobcommand files** — and that helper respects
`room.GetVisibility()`/nightvision. **`usercommands` never adopted it.**

**Impact:** player melee combat text bypasses darkness gating entirely, over-sharing combat
information to sightless observers in dark rooms.

**Fix:** `messaging.SendCombatExchange(...)` porting the mobcommands darkness logic.

### 3.4 `attack` never migrated to `actions/`
⏸️ **DESCOPED BY DECISION — 2026-07-21.** The drift bug behind this (0.5, the stealth-opener
aggro type) was fixed and engagement-aggro typing was centralised in one place
(`b23d34a15`), which removes the correctness argument. The remaining ~480-line mechanical
migration is pure structure with no behavior change, and was judged not worth the churn now.
Reopen if a future combat change needs `attack` to sit behind the `Actor` seam.

See **0.5**. ~480 lines across the two callers; only target resolution (~90 lines) was extracted.
This is the last large architectural gap in combat.

### 3.5 `shout` and `sayto`/`replyto` never migrated
See **0.3**. Additionally `internal/mobcommands/sayto.go:15-81, 83-119, 121-163` triplicates
SayTo/SayToOnly/ReplyTo (~120 lines) differing only in wording — while `actions/say.go` already
exists as the extraction point.

### 3.6 Life-cascade hook wiring repeated 14× / RoomChange preamble repeated
🔍 **REPORTED** — ~110–130 lines

14 files repeat `AfterTransition("name", func(from, to, r){ if from != Alive || to != Dead { return } ... })`
plus a byte-identical mob-or-player lookup-or-bail: 6 mob-fork files (`Death_MobBroadcast.go:26-39`,
`Death_InboundAggroCleanup.go:21-33`, …) and 5 player-fork files (`Death_PlayerCleanup.go:24-37`, …).

No missing nil-checks — this family is disciplined. The cost is volume plus 14 self-registering
`init()`s invisible from the centralized `hooks.go` list.

**Fix:** `wireOnMobDeath(name, fn)` / `wireOnPlayerDeath(name, fn)` combinators. Precedent already
exists: `hooks/combat_shared_helpers.go` proved the pattern. Same combinators fix 1.2's unchecked
asserts and the `Cancel`/`Continue` inconsistency.

### 3.7 Duplicated tuning formulas — future-drift risk
🔍 **REPORTED**

`internal/actions/combat_rally.go:82` and `combat_warcry.go:82` each independently write
`bonus := 0.05 + 0.15*math.Sqrt((rhetoric/75.0)*(charisma/175.0))`. The `"special-move"` cooldown
key is hardcoded identically at 15 sites.

No drift yet, but two independently-typed copies of a *tuning constant* are one balance pass away
from silently diverging.

**Fix:** `combat.RhetoricBonus(rhetoric, charisma)` (mirroring the existing `combat.SkillMultiplier`)
and `actions.CheckSpecialMoveCooldown(char)`.

---

## Tier 4 — Architecture and readability

### 4.1 God functions
◐ **PARTIAL — full breakup DESCOPED BY DECISION, 2026-07-21.** The highest-value extraction
called out below (the `Go()` exit-lock ladder → `unlockExit`) was done (`dec87613b`). The
broader breakup of `Go()`/`Get()`/etc. was consciously left alone: these are heavily-trafficked
paths, the gain is readability rather than correctness, and characterization-test coverage
would have to come first. Pick individual extractions up opportunistically, not as a project.

✅ **VERIFIED** (measured across the tree)

Longest functions in the codebase:

| Lines | Location |
|-------|----------|
| **736** | `internal/usercommands/go.go:30` `Go()` |
| **657** | `internal/usercommands/get.go:19` `Get()` |
| 544 | `internal/rooms/roomdetails.go:42` `GetDetails()` |
| 516 | `internal/hooks/NewRound_UserRoundTick.go:36` `UserRoundTick()` |
| 505 | `main.go:122` `main()` |
| 484 | `internal/usercommands/look.go:25` `Look()` |
| 459 | `internal/configs/config.balance.misc.go:4` `validateMisc()` |
| 417 | `main.go:627` `handleTelnetConnection()` |
| 410 | `world.go:304` `GetAutoComplete()` |
| 395 | `internal/hooks/NewRound_AutoHeal.go:28` `AutoHeal()` |
| 341 | `internal/mobs/mobs.go:340` `newMobByIdInternal()` |

`Go()` alone handles combat/death gating, quest locks, activity interruption, sneak
reconciliation, encumbrance cost, a 4-level-deep exit-lock ladder, GMCP dispatch, the room
transition, and arrival effects. No sub-step is independently testable.

**Encouraging precedent:** `internal/characters/godfunc_refactor_test.go` shows the team has
already run one god-function extraction with characterization tests pinning behavior. **Read that
file before starting any of these** — the approach is established, not novel.

**Highest-value single extraction:** `go.go:175-260`, the exit-lock ladder, where three
successful-unlock branches each repeat an identical
`SendTextVisual` + `PlaySound` + `SetUnlocked` + `SetExitLock` sequence differing only in message
text. One `unlockExit(...)` helper collapses all three.

### 4.2 `internal/util` is a genuine junk drawer
◐ **PARTIAL — full split DESCOPED BY DECISION, 2026-07-21.** The clearly-right, zero-judgment
part was done: `GetMyIP` and the `net/http` dependency it dragged in were removed along with
other dead code (`25132e6e5`). The larger relocation — dice functions → `internal/dice`,
`GetLockSequence` → lock/container code, `SafeSave`/`FilePath` → `internal/fileutil`,
display helpers → `internal/term` — was judged not worth the blast radius (touches many
importers for a mental-model gain, no behavior change). The dice/`GetLockSequence` symbols
still live in `util` today.

🔍 **REPORTED** — 1076 lines, 99 symbols, ~8 unrelated concerns

Contains: global mutex primitives, a turn/round counter singleton, a generic time-tracking
`Accumulator`, string fuzzy-matching (`FindMatchIn` — core target-resolution logic),
SHA256/MD5 hashing, **game-domain lockpicking sequence generation** (`GetLockSequence`), gzip,
base64, **an HTTP call to an external IP-lookup API** (`GetMyIP`), ASCII progress bars,
**dice rolling** (`RollDice`/`ParseDiceRoll` — overlapping the dedicated `internal/dice` package
the project's own conventions say to prefer), safe-file-save helpers, health-percent
classification, preposition stripping, ANSI conversion, `ConvertForFilename`, wildcard matching,
world-file validation, round-count persistence, comma formatting, and Unicode→ASCII box drawing.

**Consequence:** every package importing `util` for `ConvertForFilename` transitively pulls in an
HTTP client and gzip.

**Fix (mechanical, compiler-checked):** `internal/dice` absorbs the dice functions;
`GetLockSequence` moves to the lock/container code; `GetMyIP` is deleted or isolated;
`SafeSave`/`FilePath` → `internal/fileutil`; ANSI/display helpers → `internal/term`. What remains
(`Hash`, `FindMatchIn`, `ConvertForFilename`, `BoolYN`) is a legitimate small string-util package.

### 4.3 Import-cycle-breaking seams are undocumented
🔍 **REPORTED**

Beyond the documented `internal/conversationadapter`, the same inversion is repeated ad hoc via
package-level function-pointer vars:

- `internal/actions/actor.go:62-68` (`FunctionExporter`, avoids `actions→plugins`)
- `internal/goals/lookup.go:13-18,84-89` (`WeightsLookupFn`, `SetPlanStateClear`)
- `internal/rooms/roommanager.go:24-28` (`companionTransport`, avoids `rooms→hooks`)
- `connections.IssueWebSocketReconnectToken`, wired in `main.go:157-163`

Each is well-commented *locally*, but there's no single place saying "main.go is the composition
root and these N seams exist." Finding them requires grepping for `func(` package vars.

**Fix:** zero behavior change — a `docs/architecture/wiring.md` listing every `Set*`/func-var seam
and the cycle it breaks. Cheapest high-value item in this tier.

### 4.4 Half-migrated combat state — dual-write scaffolding
🔍 **REPORTED**

`internal/characters/combat_state_compat.go:10-35` — the file's own doc comment says the old
`Aggro` struct is "kept for backward compatibility" and that `SetAggro`/`EndAggro` "dual-write to
both this struct and CombatPhase," while "direct field reads remain valid across the codebase."

Two live representations of the same state, one deprecated in comment only. **This is exactly the
shape of debt that produced Tier 0's drift bugs.** Audit remaining `.Aggro.*` readers and schedule
the struct's removal.

### 4.5 Smaller readability items
🔍 **REPORTED**

- **Magic numbers violating the project's own config convention:** `go.go:117,120` hardcode
  `actionCost := 10` / `= 50` (encumbered) inline; `Position_GrappleTick.go:450` has a
  function-local `holdEmitEveryRounds`. Project convention is that combat/movement tuning lives in
  `configs.GetBalanceConfig()` so it's tunable without a rebuild+redeploy.
- **Opaque boolean params:** `newMobByIdInternal(mobId, homeRoomId, skipInstanceLoad bool, ...)`;
  `Room.SendTextToExits(txt, isQuiet bool, ...)` — `room.SendTextToExits(msg, true)` is unreadable
  at the call site. Note `NewMobById`/`NewMobByIdFresh` already solve this correctly for one case;
  extend the same treatment.
- **Wide parameter lists:** `buildAttackMessages` (`combat_helpers.go:1025`) takes 12 params, four
  of which are an obvious cohesive group. The right pattern (`weaponSetup`, `swingDamageParams`
  structs) already exists in the same package — it just wasn't applied here.
- **`rooms.RoomManager`** stutters *and* is barely a type: it has exactly one method
  (`GetFilePath`); everything else in the file operates on package globals as free functions.
  Either delete the type or finish the conversion (see 4.6).
- **Doc comments** on exported symbols are sparse on older code (most of `*Room`'s ~80 exported
  methods) and good on newer code. Recommend enforcing on new symbols via review habit rather
  than a mechanical backfill — backfilled comments restating the method name add noise.

### 4.6 Global world-state registries have no injection seam
🔍 **REPORTED** — highest effort in this document; **scope it, don't do it wholesale**

The entire in-memory world lives in package-level vars: `internal/mobs/mobs.go:37-51`,
`internal/users/users.go:26-31`, `internal/rooms/roommanager.go`. Defensible for a single-process
MUD, but it means no two worlds can coexist in one test binary and every state-touching test must
reset shared globals.

A tell that this already causes friction: `util.go:132-140` has retrofitted
`SetRoundCountForTest`/`ResetRoundCountForTest` helpers.

**Cheapest first step:** `rooms.RoomManager` already exists as a name — finish converting its free
functions into real methods and thread the instance through the ~10 lowest-fan-in callers. Do
`mobs`/`users` **only** if a concrete testing pain point demands it.

---

## Tier 5 — Modern Go idioms

`go.mod` declares `go 1.25.0`. Most migrations are already done (see Baseline health). What
remains:

### 5.1 Two YAML libraries in one binary — tied to a real production incident
✅ **VERIFIED** (79 files on v2, 13 on v3)

`go.mod` depends on **both** `gopkg.in/yaml.v2` and `gopkg.in/yaml.v3`. v2 is the dominant loader
for game content (everything `internal/fileloader` touches — mobs, rooms, quests, dialogue, items);
v3 is used in a scattered minority (`goals`, `facts`, `bounties`, `knowledge`, `sealedcrate`,
`grapplemessaging` persistence).

**This is not a style nit.** Two documented production incidents live in this exact code path:
- "yaml tag on unexported field = silent no-op" — cost the `hostile:` field two months on prod
  with zero errors.
- "dialogue bare-scalar list field mutes whole NPC" — `questRequired: "X"` instead of `["X"]`
  causes yaml.v2 to nil the **entire file**, and lazy-loading hides it from the boot test.
  `internal/dialogue/loader.go` is a **yaml.v2** consumer.

> ## ⚠️ RECOMMENDATION WITHDRAWN — 2026-07-20, after measurement
>
> The original recommendation ("standardize content loading on yaml.v3 and add a
> `KnownFields(true)` pass") **should not be carried out.** Three claims behind it were tested
> against the real content tree and did not hold:
>
> **1. "v2 nils the entire file, v3 would catch it" — false.** Both decoders produce a *byte
> identical* error on the bare-scalar case, and both still populate the rest of the struct.
> Discarding the whole file is not a library behaviour: `internal/dialogue/loader.go:48-53`
> throws away the partial result and permanently caches `nilSentinel[key] = true` on any error.
> **The bug is in the loader, not in yaml.v2.** Migrating would not have fixed the incident.
>
> **2. "v2 has no clean equivalent to `KnownFields`" — false.** `yaml.v2.UnmarshalStrict`
> produces the same "field X not found in type Y" error. Strict decoding needs no migration.
>
> **3. "v2 and v3 differ on some edge cases" — true but misleading.** Decoding all 6,111 data
> files with both libraries showed **1,302 files differing** — which sounds alarming, and is why
> this needed measuring rather than assuming. The dominant cause (2,460 occurrences across 1,230
> files) is YAML 1.1 implicit typing on *map keys*: v2 reads the key `y` in `coord: {x:, y:}` as
> boolean `true`, v3 reads it as the string `"y"`.
>
> **But that difference vanishes under typed decoding.** Decoding into structs, `map[string]string`
> and `[]string` — which is what all content loaders actually do — v2 and v3 agree exactly,
> including on `n`/`y`/`yes`/`no`. The 1,302 differences are an artifact of decoding into
> `interface{}`.
>
> So the migration is *mostly* safe and *entirely* pointless — high blast radius across 79 files,
> no benefit. Worse, the places that genuinely would change behaviour are the untyped decode sites:
> `internal/migration/0.1x.0.go` does raw `map[string]interface{}` surgery on **user save files**.
> That is the one place where key retyping could corrupt player data, and it is the last place a
> "mechanical" migration should touch.
>
> **What was done instead**, closing the actual incident class at a fraction of the risk:
> `TestSmoke_AllDialogueFilesParse` eagerly parses all 302 dialogue files in CI. Dialogue is the
> one major content type `loadAllDataFiles` does not cover — it is lazy-loaded per mob, which is
> exactly why a malformed file could mute an NPC in production unnoticed. Verified to fail on the
> documented bare-scalar case.
>
> **The useful half — DONE, 2026-07-20, as a drift gate rather than a hard flip.**
> Measuring first showed a blanket `UnmarshalStrict` would fail the boot with **3,213 violations
> across 32 distinct field/type pairs**, the vast bulk benign: `coord` (2,459) is authored room
> data the engine abandoned for crawled exit-delta positions, `level` (612) is legacy from the
> level/XP removal, and most quest entries come from quest YAML being parsed by *two* type systems
> (legacy `quests` and newer `questengine`), each seeing the other's fields as unknown.
>
> So instead of flipping loading to strict, `fileloader` gained a zero-cost
> `StrictDecodeProbe` hook (nil in production) and `TestSmoke_NoNewSilentlyIgnoredYAMLKeys`
> baselines today's 32 pairs. Any **new** unknown key fails CI — verified by injecting `hostiel:`
> on a mob, which reproduces the original incident shape exactly. That closes the class going
> forward without demanding the backlog be cleared first.
>
> **Content bugs the measurement surfaced, worth fixing separately** (all currently baselined, each
> an authored value that does nothing):
> - `item_id` on `quests.QuestReward` — a **live instance** of the documented "reward keys are
>   no-underscore (`itemid`); snake_case silently no-ops" footgun.
> - `scriptag` on `mobs.Mob` — almost certainly a typo for `scripttag`.
> - `cooldown` on `rooms.SpawnInfo` (118 files) and `zone` on `exit.RoomExit` (136 files).
> - `visible`, `sequential`, `expireMessage` on `buffs.BuffSpec`.
> - `allow_recall` on `rooms.Room`, `long` on `rooms.Container`, `tactics` on
>   `characters.Character`, `items` on `mobs.Mob`.
>
> Each is cleared by fixing the content (or adding the field) and deleting its line from
> `knownSilentlyIgnoredKeys`.

### 5.2 Two copies of the same logging-rotation library
✅ **VERIFIED**

`go.mod` pulls **both** `github.com/natefinch/lumberjack v2.0.0+incompatible` (the unmaintained
pre-module path) and `gopkg.in/natefinch/lumberjack.v2 v2.2.1`. The code imports the *old* path in
`internal/mudlog/mudlog.go:12` and `internal/combat/analytics.go:12`.

**Fix:** switch both imports to `gopkg.in/natefinch/lumberjack.v2`, drop the old require. Mechanical.

### 5.3 `github.com/pkg/errors` is fully redundant
✅ **VERIFIED** (19 files import it; 🔍 29 `Wrap`/`Wrapf` calls)

Pre-1.13 wrapping library, now entirely replaceable by `fmt.Errorf("...: %w", err)`.
`errors.Wrap(err, "msg")` → `fmt.Errorf("msg: %w", err)` is a same-shape swap the compiler verifies.

Related: of 448 `fmt.Errorf` sites, only **172 (38%)** use `%w`. **Good news:** a grep for the usual
compensating anti-pattern (`strings.Contains(err.Error(), ...)`) found **zero occurrences in
production code** — only in tests, which is normal. So the codebase isn't string-matching errors;
it just isn't set up for identity-based handling. That's a latent maintainability gap, not a
present bug — treat the `%w` backfill as opportunistic, not a project.

### 5.4 Smaller idiom items
🔍 **REPORTED**

- **Hand-rolled stdlib equivalents:** `sliceContains` (`internal/goals/registry.go:67`) is a literal
  `slices.Contains` body; `max1` (`internal/economy/health/scoring.go:649`) is `max(n, 1)`. Delete
  and inline — the codebase already uses `slices.*` (26 sites), `maps.Copy` (10), and builtin
  `min`/`max` (36) elsewhere.
- **Index loops:** 25 sites across 18 production files use `for i := 0; i < len(s); i++` for plain
  scans. A handful genuinely need the index (comparing `i` to `i±1`) and should stay.
  **Zero adoption of Go 1.22's `for range n`.**
- **~~One real missing timeout~~ — CORRECTED 2026-07-20, this finding was overstated.**
  The original text claimed `internal/integrations/discord/client.go:140` posts a webhook "with no
  context and no explicit client timeout — that goroutine can hang indefinitely." **That is wrong.**
  Reading the code, the client already sets per-phase transport timeouts (dial 3s, TLS handshake 3s,
  response-header 3s, expect-continue 1s) and the goroutine has a `recover()`. The request is
  bounded and cannot hang indefinitely. The only genuine residual gap was the absence of an overall
  `http.Client.Timeout` covering body transfer — a minor defence-in-depth item, since this code
  never reads the response body. Fixed by adding `Timeout: 10 * time.Second`.
  This codebase is genuinely not a "context is missing everywhere" one; most I/O is local disk where
  ctx adds nothing.
- **One real logging bypass:** `internal/web/web.go:286` uses `log.Println("WebSocket upgrade
  failed:", err)` instead of mudlog, inside the actual request path. The other `log.*` uses are
  legitimate (bootstrap before the logger exists; a `go:generate` build tool).
- **`math/rand` → `math/rand/v2`:** 11 files, zero on v2. Mechanical *except* `internal/dice/dice.go`,
  the core stat-roll engine — combat balance depends on its distribution characteristics, so that
  one needs tests, not a sed. `internal/gametime/zodiac.go:240` uses a config-derived seed for
  reproducible generation and should move deliberately to `rand.NewPCG`.
- **Embed candidates:** `internal/web/web.go:297` serves admin UI static assets from a
  configured CWD-relative path at request time. `//go:embed` + `http.FS` would remove a class of
  "wrong working directory on deploy" bugs — **but** only if the team doesn't want those assets
  live-editable on the droplet. Judgment call. The `_datafiles` YAML tree should **not** be embedded;
  hot-reloadability is deliberate design there.

---

## Tier 6 — Tests and CI

### 6.1 What CI enforces today
✅ **VERIFIED**

| Check | Status |
|---|---|
| `go test -race` + coverage | ✅ Runs on PRs **and** master pushes |
| Coverage gate (28% floor) | ✅ On PRs only |
| Dependabot | ✅ Weekly, gomod |
| `go vet` | ❌ **Not in CI** (Makefile-only via `make validate`) |
| `gofmt` check | ❌ **Not in CI** |
| golangci-lint / staticcheck | ❌ No config exists |
| `govulncheck` | ❌ Absent |
| `js-lint` (jshint) | ❌ Makefile-only; CI skips it entirely |
| Data-file boot validation | ❌ Absent — **highest-value gap** |
| Cross-OS test execution | ❌ Builds 5 targets, tests only on ubuntu-latest |

Two soft spots worth noting: the coverage gate silently `exit 0`s if `coverage.out` is missing,
and the `Makefile`'s `test` target uses `-race`, which **requires cgo** — on a machine without a C
toolchain it errors rather than running (confirmed locally: `-race requires cgo`). CI is on Linux
so this is a local-dev papercut, not a CI gap.

### 6.2 A data-file boot test is the single highest-leverage addition
🔍 **REPORTED** — feasibility confirmed against existing precedent

The project's Pre-Push SOP *requires* manually booting the server and watching for
`mobs.LoadDataFiles() loadedCount=...` without panics, because YAML content errors (filename/name
mismatch, invalid triggers, ID collisions) surface **only at boot** — never at `go build` or
`go vet` time. **No Go test anywhere calls the loaders against the real
`_datafiles/world/dogmud` tree.**

Precedent for the pattern already exists in-repo: `internal/usercommands/helpfile_completeness_test.go`
walks up from CWD to find `_datafiles/world/dogmud` and validates real content against it.

**Fix:** a `TestSmoke_ServerBootsCleanWithRealData` calling `mobs.LoadDataFiles()`,
`quests.LoadDataFiles()`, the room loader, `dialogue.LoadDataFiles()` etc. against the real tree,
asserting zero errors and `loadedCount > 0` per category. **This automates the manual SOP step
that currently depends on a human remembering to do it** — and it is the prerequisite for safely
attempting the yaml.v2→v3 consolidation in 5.1.

### 6.3 Test suite shape
✅ **VERIFIED** — full detail in [`TEST_COVERAGE_AUDIT.md`](TEST_COVERAGE_AUDIT.md)

Measured 2026-07-20: **42.3% repo total**, 4,880 test functions, 122 packages. Distribution:
31 packages at 0%, 13 under 25%, 23 at 25–50%, 26 at 50–75%, 29 above 75%. Tier-1 gameplay
packages average **60.4%** (was 24% in February — the Feb remediation plan worked).

Three findings from that audit that bear directly on this document:
- **211 skip sites**, ~130 of which are migration-checklist shells that assert nothing.
  Eight tests referenced by "*covered elsewhere*" skip messages **do not exist anywhere** —
  a verified false assurance covering eight grapple positions.
- Several tests **cannot fail** (zero assertions; assertions placed after a `recover()`;
  one test that doesn't touch the type it's named for).
- The CI floor of 28% is now 14 points below actual (42.3%), so it cannot detect a
  meaningful regression. Raising it to ~40% is a two-character change.

Additional detail below:

- **23 packages have zero test files.** Ranked by (size × centrality), the top targets:
  1. `internal/statmods` — feeds the combat-math stack via `buffs`/`enchantments`/`characters`;
     pure logic, trivially testable, currently 0%.
  2. `internal/enchantments` — boot-critical (`main.go:62`); `copyStatMods` is exactly the
     shallow-copy-shared-pointer bug class already documented as a past incident.
  3. `internal/plugins` — boot-critical module wiring, 49 symbols.
  4. `internal/connections` — every player session; concurrency-heavy, so a real `-race` beneficiary.
  5. `internal/web` admin surface — only `auth_test.go` exists; `admin.progression.go` (44 symbols)
     and 7 sibling admin files are untested.
- ✅ `t.Parallel()` appears in **exactly 1 file** — and that file is the one *documenting why
  parallelism is unsafe* here (`internal/goals/prune_test.go:10-14`). The suite is effectively
  serial. Given the package-global architecture (4.6), **this is the correct trade-off, not a gap.**
  If wall-time becomes a problem, parallelize across packages (already default), not within them.
- ✅ `t.Skip` appears in **43 files.** Most are legitimate and well-documented (e.g.
  `internal/actions/actions_test.go:238` skips on a real import-cycle constraint). But some encode
  known-stale behavior: `internal/hooks/Death_PlayerRespawn_test.go:55` skips with "Die() now
  revives userId-0 characters; the soft-lock discriminator changed — re-audit new-player death."
  That's honestly-flagged debt sitting unaudited — and cheap to pay down because it has a paper trail.
- 🔍 `time.Sleep`-based test synchronization appears in only **1 file** — flake risk is genuinely low.
- 🔍 ~20 packages carry an identical 4-line `TestMain` calling `mudlog.SetupLogger(nil, "", "", false)`.
  Go requires `TestMain` in-package so the win is small, but a `testutil.SetupTestLogger()` would at
  least centralize the arguments.

### 6.4 CI hardening checklist (ordered by value-to-effort)

1. **Add `go vet ./...` + `gofmt -l` to `.github/actions/codegen-and-test`.** Both pass clean today
   — pure regression-proofing. *Effort: trivial (2 lines).*
2. **Add the data-file boot test (6.2) and run it in CI.** Automates the manual Pre-Push SOP step.
   *Effort: medium.*
3. **Fix the `refs/tags/master` bug (0.8).** *Effort: trivial.*
4. **Add `govulncheck ./...`**, non-blocking first, then gating. A MUD server on a public port is a
   reasonable target and there's zero vuln visibility today. *Effort: small.*
5. **Introduce `.golangci.yml`** — ✅ **DONE 2026-07-20.** Shipped with `govet`, `staticcheck`
   (SA* bug checks only — ST/S/QF style excluded), `errcheck`, `ineffassign`, `unconvert`.
   `internal/migration/`, `vendor/`, `_datafiles/` excluded; errcheck excluded on `_test.go`.
   `misspell`/`revive`/`gocritic` were dropped from the original plan as style noise on a legacy
   tree; `gocyclo`/`funlen`/`lll` likewise. Measured backlog: **107 findings** (errcheck 50,
   ineffassign 32, staticcheck-SA 20, unconvert 3, govet 2) — grandfathered, not fixed. CI's
   `run-tests.yml` runs `golangci-lint-action` with `only-new-issues: true` so only what a change
   introduces gates; `make lint` (new-from-merge-base) is the local equivalent, `make lint-all`
   shows the full backlog.

   Reach caveat: the gate lives on the `pull_request` workflow, so under this project's
   direct-push-to-master habit it only bites on actual PRs. `make lint` is the direct-push
   developer's tool.

   Notable items in the backlog worth a look (surfaced by staticcheck SA*, not gating):
   - `internal/usercommands/look.go:437,445` — `mobCorpses`/`playerCorpses` built with `append`
     then never read (SA4010); dead code, no wrong behaviour.
   - `internal/usercommands/look.go:615` — `user != nil` checked after `user` is already
     dereferenced at :608 (SA5011); either dead check or a latent nil-deref.
   - `strings.Title` deprecated at `killstats.go:87`, `gmcp.Char.go:338`, `gmcp.Room.go:184` (SA1019).
6. **Wire `js-lint` into CI** (`npx jshint` directly on the runner; no Docker needed).
   *Effort: small.*
7. **Publish coverage delta as a PR comment** rather than only gating an absolute floor — a flat
   28% floor lets coverage stagnate at 28% forever. *Effort: small.*
8. **Ratchet the coverage floor** (+1%/quarter) *after* the untested packages in 6.3 are addressed.

---

## Recommended sequence

Ordered so that each step de-risks the next. **Status tags added 2026-07-21 reflect what was
actually executed — see the "Execution status" ledger near the top for the authoritative
summary and commit references.**

**Phase 1 — Stop the bleeding (days, low risk, high payoff)**
1. ✅ **Panic recovery in `DoListeners` + `handleTelnetConnection` (1.1).** Single highest-leverage
   change in this document: converts most latent panics from "server outage" to "one failed round."
2. ✅ **Tier 0 correctness bugs (0.1–0.8).** Small, surgical, independent of any refactor. Fix the
   locked-container `get` bypass (0.6) first — it is player-reachable today — then the `put`
   gold dupe (0.1) and the `refs/tags/master` CI bug (0.8), both trivial.
3. ✅ **`go vet` + `gofmt` in CI (6.4 #1).** Two lines.
4. ✅ **Lumberjack dedup (5.2), `sliceContains`/`max1` deletion (5.4), Discord timeout + web.go logging
   bypass (5.4).** Mechanical cleanups, no judgment required.

**Phase 2 — Make the invariants enforceable (weeks)**
5. ✅ **Data-file boot test in CI (6.2).** Automates the manual SOP; **prerequisite for step 8.**
6. ✅ **Concurrency fixes (2.1, 2.2, 2.3, 2.4).** Done — including 2.4: 2.2 was fixed by having
   `GetAutoComplete` take `RLockMud` (a read-only path), which resolved 2.4 by making the
   read-lock live exactly as this step intended.
7. ✅ **Quest-sequence goroutine hardening (1.3).** Content-driven, so higher real-world trigger
   probability than its line count suggests.
8. ⏸️ **yaml.v2 → v3 consolidation + `KnownFields(true)` (5.1) — WITHDRAWN after measurement.**
   Replaced with a dialogue parse gate + a silently-ignored-key drift gate (see §5.1).

**Phase 3 — Pay down structure (opportunistic, incremental)**
9. ✅ **`actions.AcquireMeleeTarget` (3.1)** — largest, lowest-risk extraction (~400 lines, no behavior
   change), and it establishes the seam that 3.4 and 3.3 both want to sit next to.
10. ⏸️ **Migrate `attack` (3.4) — DESCOPED** (see §3.4). The aggro-type drift it would have fixed
    was addressed directly (0.5); the mechanical migration wasn't judged worth the churn.
11. ✅ **`messaging.SendCombatExchange` (3.3) — RESOLVED, premise disproven** (see §3.3). The
    combat-text surface already routes room broadcasts through the gated `SendTextVisual`
    pipeline; the info leak was closed by the messaging refactor. No new helper needed.
12. ⏸️ **Hook combinators (3.6) — WITHDRAWN** (premise disproven, see §1.2).
13. ⏸️ **Split `internal/util` (4.2) — DESCOPED** (see §4.2). The `GetMyIP`/`net/http`/dead-code
    removal was done; the larger relocation was judged not worth the blast radius.
14. ✅ **Document the wiring seams (4.3)** — an afternoon, zero risk.
15. ⏸️ **Break up `Go()` and `Get()` (4.1) — full breakup DESCOPED** (see §4.1); the highest-value
    `unlockExit` extraction was done.
16. ✅ **Introduce `.golangci.yml` (6.4 #5)** last in this phase, so the first run isn't fighting code
    that's actively being restructured.

**Explicitly deferred (documented, not scheduled)**
- Global-registry dependency injection (4.6) — highest effort here; do only if a concrete testing
  pain point demands it, and start with `rooms.RoomManager`.
- Repo-wide file-naming convergence (4.5) — churns git blame for cosmetic gain.
- Mechanical doc-comment backfill — enforce on new code via review instead.
- `%w` backfill across 276 sites (5.3) — opportunistic; no active harm today.
- `t.Parallel()` adoption — unsafe until 4.6, and the serial suite is currently the right call.

---

## Appendix — audit provenance

Tooling run directly during this audit:
- `go build ./...` → clean
- `go vet ./...` (default analyzers) → clean, exit 0
- `go test -timeout 600s ./...` → all pass, exit 0
- `go test -race ./...` → **could not run locally** (no C toolchain; `-race requires cgo`).
  CI runs it on Linux. The concurrency findings in Tier 2 are therefore static-analysis-derived and
  marked accordingly — **a `-race` run against a live multi-player session is the recommended
  confirmation step for 2.1 and 2.2.**

Counts verified directly: yaml.v2 79 files / yaml.v3 13 files; `pkg/errors` 19 files; dual
lumberjack imports; ~~`RLockMud` zero non-test callers~~ (**wrong — grep was scoped to
`internal/` and missed the `world.go` caller; see §2.4**); `t.Parallel()` 1 file; `t.Skip` 43 files;
TODO/FIXME/HACK/XXX 29 total; function lengths in 4.1; `put.go` gold/lock divergence;
`DoListeners`/`MainWorker` recover absence; `GetAutoComplete` call sites unlocked.

**Known limitation:** the six specialist audits read code but could not execute it. Findings marked
🔍 REPORTED are backed by direct code reading; findings marked ⚠️ NEEDS PROOF describe a race whose
*structure* is confirmed but whose *occurrence* requires runtime observation. No finding in this
document was accepted on pattern-matching alone.
