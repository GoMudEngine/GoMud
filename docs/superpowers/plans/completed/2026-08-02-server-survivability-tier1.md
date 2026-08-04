# Server Survivability (Tier 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close every defect from the 2026-08-02 adversarial review that can kill the server process, freeze the world, or destroy player data.

**Architecture:** The engine has one authoritative game-loop goroutine (`worldManager.MainWorker`, `world.go:732`) that holds `util.LockMud()` for every tick body. Everything else — one goroutine per telnet/websocket connection, plus HTTP handlers — runs genuinely concurrently with it. Three of the four defects here are the same root cause: **connection-goroutine code reading or writing game state without taking the mud lock.** Go's runtime throws an uncatchable `fatal error` on concurrent map access, so these are process kills, not recoverable panics. The fourth is a data-loss bug in the user loader. Fixes use two patterns that already exist in the codebase: `util.RLockMud()` for read-only access, and deferring mutations to `MainWorker` via `events.AddToQueue`.

**Tech Stack:** Go 1.x, `gopkg.in/yaml.v2`, gorilla/websocket, testify. No new dependencies.

---

## Triage: the full review, and what this plan covers

The adversarial review produced 16 verified findings. They do not belong in one plan — they span unrelated subsystems with different risk profiles. This document specifies **Tier 1 only**; the rest are listed so nothing is lost.

### Tier 1 — in this plan (process death, world freeze, data loss)

| # | Severity | Defect | Task |
|---|---|---|---|
| 1 | CRITICAL | `LoadUser` overwrites a partly-corrupt save with defaults | 1 |
| 2 | CRITICAL | Prompt redraw reads room manager from the connection goroutine, unlocked | 2 |
| 3 | CRITICAL | GMCP `Char.Action.Try` / `Char.Quests.Focus` mutate world state unlocked | 3 |
| 4 | HIGH | LLM `onUnavailable` callbacks touch mob + dialogue state off-lock | 4 |
| 5 | HIGH | Global connection lock held across deadline-less `net.Conn.Write` | 5 |
| 6 | HIGH | Pre-login websocket disconnects leak connection slots forever | 6 |

### Tier 2 — exploitable game logic (separate plan)

- `actReturnItem` mints a fresh item and leaves the original in the mob's inventory, so give-then-kill duplicates items (11 shipped behavior files, incl. the combat-capable `guard_captain`).
- Mob spawn shares the template's `Character.Mutations` map; intrinsic mutations compound across respawns.
- `GiveItem` / dialogue `givesItem` ignore `StoreItem`'s failure, so an overloaded player is told they received a quest item that does not exist — unrecoverable soft-lock.

### Tier 3 — network security surface (separate plan)

- `authCache` is an unguarded map, and `/build` + `/build-help` sit outside `RunWithMUDLocked`; also an unthrottled bcrypt oracle against real player credentials.
- IP bans are inert for web-client players — `IsLocal()` reads the proxy's address, which is loopback behind Caddy.
- Websocket has no `SetReadLimit` and no connection cap; `CheckOrigin` always returns true.
- Player text is interpolated into `<ansi>` markup without escaping `<`, so chat can spoof server/admin lines.

### Tier 4 — combat correctness (separate plan)

- Melee crit damage rolls with a stale, ~half-sized standard deviation.
- Pet/companion damage bypasses `ApplyMitigation` entirely.
- `spellDefenseValue("physical")` sums the legacy `DamageReduction` field, which 35 of 77 mitigation-bearing items never set.

### Tier 5 — persistence robustness (separate plan)

- `rooms` and `shops` saves ignore `CarefulSaveFiles`, so a crash mid-write truncates a YAML that panics at next boot.
- Defusing an exit trap never persists (`Room.Exits` is `instance:"skip"`), while the container branch of the same command does.

---

## File Structure

| File | Change |
|---|---|
| `internal/users/users.go` | `LoadUser` must return unmarshal errors |
| `internal/characters/validate.go` | `Validate` must be able to fail |
| `internal/users/loaduser_test.go` | New — corrupt-save handling |
| `main.go` | Lock the prompt-redraw reads; fix websocket cleanup |
| `modules/gmcp/gmcp.go` | Defer state-touching handlers to MainWorker |
| `internal/llm/client.go` | Take the mud lock around `onUnavailable` |
| `internal/connections/connections.go` | Release the lock before writing; set write deadlines |
| `internal/connections/connectiondetails.go` | Write deadline plumbing |

---

## Task 1: `LoadUser` must not destroy a partly-corrupt save

**The defect.** `internal/users/users.go:481-489`:

```go
loadedUser := &UserRecord{}
if err := yaml.Unmarshal([]byte(userFileTxt), loadedUser); err != nil {
    mudlog.Error("LoadUser", "error", err.Error())
}

if len(skipValidation) == 0 || !skipValidation[0] {
    if err := loadedUser.Character.Validate(true); err == nil {
        SaveUser(loadedUser)
    }
}
```

The unmarshal error is logged and execution continues. `Character.Validate` (`internal/characters/validate.go:567-694`) contains **exactly one `return` statement — `return nil`** — so the `err == nil` gate is always true and `SaveUser` always fires. A save with one bad field is silently rewritten with defaults, destroying items, stats and progression. A fully torn file leaves `Character` nil and `Validate` dereferences it, panicking the server on that player's login.

`internal/shops/persistence.go:340` and `internal/guilds/persistence.go:82` both log-and-skip correctly. `users` is the outlier and the only one that writes back.

- [ ] **Step 1: Write the failing tests**

Create `internal/users/loaduser_test.go`. Follow the chdir + `configs.ReloadConfig()` convention now standardised in `internal/configs/testing_support.go` (`SetConfigForTest`) — see `internal/characters/poolmax_test.go`'s `withRepoRoot` for a current example.

```go
package users

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A save that is valid YAML but has one bad field must NOT be silently
// rewritten with defaults. Before 2026-08-02 LoadUser logged the unmarshal
// error, continued, and re-saved the defaulted record over the original.
func TestLoadUserDoesNotOverwriteCorruptSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "1.yaml")
	// `training` is an int; a string here makes yaml.v2 return a TypeError
	// while still populating the rest of the document.
	original := "userid: 1\nusername: corrupt\ncharacter:\n  name: Corrupt\n  stats:\n    strength:\n      base: 120\n      training: notanumber\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadUserFromPath(path, false)
	if err == nil {
		t.Fatal("LoadUser accepted a save with a malformed field; want an error")
	}

	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != original {
		t.Errorf("save file was rewritten:\n--- before ---\n%s\n--- after ---\n%s", original, string(after))
	}
	if !strings.Contains(string(after), "training: notanumber") {
		t.Error("the offending field was rewritten — the original must be preserved for diagnosis")
	}
}

// A completely unparseable file must return an error, not panic on a nil
// Character.
func TestLoadUserTornFileReturnsErrorNotPanic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2.yaml")
	if err := os.WriteFile(path, []byte("userid: 2\nusername: torn\ncharacter:\n  stats:\n   \x00\x00 broken"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadUserFromPath(path, false); err == nil {
		t.Fatal("torn file was accepted; want an error")
	}
}
```

> `loadUserFromPath` does not exist yet. Step 3 extracts it from `LoadUser` so the parsing and validation logic is testable without the username index. If you find a cleaner seam while reading `LoadUser`, use it and adjust the tests — but the behaviour asserted above must hold.

- [ ] **Step 2: Run and confirm both tests fail**

Run: `go test ./internal/users/ -run TestLoadUser -v`
Expected: compile failure on the missing `loadUserFromPath`, then — once Step 3 lands — real assertion failures if the behaviour is wrong.

- [ ] **Step 3: Make `LoadUser` return the unmarshal error**

Extract the read-parse-validate portion of `LoadUser` into:

```go
func loadUserFromPath(userFilePath string, skipValidation bool) (*UserRecord, error) {
	userFileTxt, err := os.ReadFile(userFilePath)
	if err != nil {
		return nil, err
	}

	loadedUser := &UserRecord{}
	if err := yaml.Unmarshal([]byte(userFileTxt), loadedUser); err != nil {
		// Do NOT fall through. Returning here is what stops a partly-parsed
		// record being "repaired" with defaults and written back over the
		// player's real save. Matches shops/guilds, which log and skip.
		mudlog.Error("LoadUser", "path", userFilePath, "error", err.Error())
		return nil, fmt.Errorf("could not parse user file %s: %w", userFilePath, err)
	}

	if loadedUser.Character == nil {
		return nil, fmt.Errorf("user file %s parsed with no character data", userFilePath)
	}

	if !skipValidation {
		if err := loadedUser.Character.Validate(true); err != nil {
			return nil, fmt.Errorf("user file %s failed validation: %w", userFilePath, err)
		}
		SaveUser(loadedUser)
	}

	return loadedUser, nil
}
```

Then rewrite `LoadUser` to resolve the path via the username index as it does today and delegate to `loadUserFromPath`, propagating the error.

- [ ] **Step 4: Give `Validate` a real failure mode**

`Character.Validate` currently cannot fail. At minimum it must reject a receiver it cannot safely operate on. At the top of `internal/characters/validate.go:567`:

```go
func (c *Character) Validate(recalcPermaBuffs ...bool) error {
	if c == nil {
		return errors.New("cannot validate a nil character")
	}
```

Do **not** attempt to make `Validate` reject every malformed field in this task — its backfilling behaviour is relied upon widely. The contract change is narrow: it may now return an error, and callers must check it. `go build` will enumerate every caller; audit each one and make sure none silently discards the new error.

- [ ] **Step 5: Verify**

```
go test ./internal/users/ ./internal/characters/ -count=1
go build ./...
go vet ./...
```
All must pass. Then confirm the audit: `grep -rn "\.Validate(" --include=*.go internal/ | grep -v _test` and check no call site drops the error.

- [ ] **Step 6: Commit**

```bash
git add internal/users/users.go internal/users/loaduser_test.go internal/characters/validate.go
git commit -F /tmp/msg1.txt
```
with the message body: `fix(users): stop LoadUser rewriting a corrupt save with defaults` plus the explanation that `Validate` could never return an error so the re-save gate was always open, and the `Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>` trailer.

---

## Task 2: Lock the prompt-redraw path

**The defect.** `main.go:889` (and the sibling redraw sites at `:1024`, `:1026`, `:1182`, `:1201`) calls `userObject.GetCommandPrompt()` on the **connection goroutine** with no lock. That walks into `ProcessPromptString` → `canSeeTargetForPrompt` → the hook registered at `main.go:323` → `rooms.LoadRoom(...)` → `getRoomFromMemory` / `addRoomToMemory`, which read and write `roomManager`'s five plain, mutex-free maps (`internal/rooms/roommanager.go:50-56`, writes at `:563`, `:567`, `:586`). `MainWorker` writes those same maps under the lock during room maintenance (`world.go:824-825`).

A player pressing Tab or Backspace during room maintenance produces `fatal error: concurrent map read and map write` — uncatchable, process dies.

- [ ] **Step 1: Confirm the read-only lock primitive**

`util.RLockMud()` / `util.RUnlockMud()` exist (`internal/util/util.go:88-94`). The only current caller is `GetAutoComplete` (`world.go:321`), which is precedent for exactly this situation: a connection-goroutine read of game state. Read that call site before editing.

- [ ] **Step 2: Wrap each redraw site**

At `main.go:888-895` and each sibling site:

```go
if redrawPrompt {
    util.RLockMud()
    pTxt := userObject.GetCommandPrompt()
    util.RUnlockMud()

    if connections.IsWebsocket(clientInput.ConnectionId) {
        connections.SendTo([]byte(pTxt), clientInput.ConnectionId)
    } else {
        connections.SendTo([]byte(templates.AnsiParse(pTxt)), clientInput.ConnectionId)
    }
}
```

**Hold the read lock across `GetCommandPrompt` only — never across `connections.SendTo`.** `SendTo` performs blocking network writes (see Task 5); holding a world lock across it would couple a stalled client to the game loop, which is the exact failure Task 5 exists to remove.

`mudLock` is a `sync.RWMutex` and is **not reentrant** (pinned by `internal/util/mudlock_test.go:31`). Before wrapping a site, confirm it is not already inside a locked region — if it is, do not double-lock; note it and move on.

- [ ] **Step 3: Verify no site was missed**

`grep -n "GetCommandPrompt" main.go` and confirm every connection-goroutine call is wrapped. Calls from `world.go:1028`, `world.go:1044` and `internal/hooks/RedrawPrompt_SendRedraw.go:22` are already on `MainWorker` under the exclusive lock and must **not** be wrapped.

- [ ] **Step 4: Verify**

`go build ./... && go vet ./...`, then `go test -race ./internal/rooms/ ./internal/users/ -count=1` if the race detector is usable in this environment. Record whether `-race` ran; if Windows Defender blocks the instrumented binary, say so rather than claiming a clean race run.

- [ ] **Step 5: Commit** — `fix(prompt): take the mud read lock around prompt redraw`

---

## Task 3: Defer state-touching GMCP handlers to MainWorker

**The defect.** `modules/gmcp/gmcp.go:421-449` (`Char.Action.Try`), `:451-475` (`Char.Quests.Focus`) and `:372-419` (`Char.Automation.*`) run inside `HandleIAC` on the **connection goroutine** and call `rooms.LoadRoom`, `u.Command(...)` and `u.Character` mutators with no lock.

The fix already exists thirty lines below, for the `Build.*` commands (`gmcp.go:477-496`), with a comment stating the exact reason: *"HandleIAC runs on the per-connection goroutine, so doing that work here would race the world tick (concurrent map writes -> fatal). Defer to MainWorker via an event."* Those commands post a `GMCPBuildOp` event handled by `handleBuildOp`, registered at `gmcp.go:70`. The neighbouring handlers never got the same treatment.

- [ ] **Step 1: Read the existing pattern**

Read `gmcp.go:477-496` and `handleBuildOp` end to end, plus the `events.RegisterListener(GMCPBuildOp{}, ...)` registration at `gmcp.go:70`. **Mirror it.** Note the payload-copy comment — the IAC read buffer is reused after `HandleIAC` returns, so any payload carried onto the queue must be copied.

- [ ] **Step 2: Add an event type and handler**

Define a `GMCPCharOp` event carrying the connection id, the command name, and a **copied** payload. Register a listener alongside the existing one at `gmcp.go:70`. Move the bodies of `Char.Action.Try`, `Char.Quests.Focus` and `Char.Automation.Set`/`Remove` into that handler unchanged — it runs on `MainWorker` under the lock, so the existing logic is correct there as-is.

In `HandleIAC`, each of those cases reduces to: parse the payload, copy it, `events.AddToQueue(...)`, return.

- [ ] **Step 3: Preserve `Char.Action.Try`'s reply**

That handler currently calls `sendActionResult(uid, req.Id, "fired"|"deferred"|"rejected", reason)` synchronously. The reply must still be sent, now from the deferred handler. Confirm `sendActionResult` is safe to call from `MainWorker` — it queues or writes output like any other game-loop code path.

- [ ] **Step 4: Verify**

`go build ./... && go vet ./... && go test ./modules/... ./internal/... -count=1`. Then grep `modules/gmcp/` for any remaining handler that touches `rooms.`, `users.`, `mobs.` or `u.Character` directly inside `HandleIAC`, and report anything still unconverted.

- [ ] **Step 5: Commit** — `fix(gmcp): defer state-touching Char.* handlers to MainWorker`

---

## Task 4: Take the mud lock around LLM `onUnavailable`

**The defect.** `internal/llm/client.go:18-19` documents the contract:

```go
// onResponse is called under util.LockMud() — safe to call mob/room functions directly.
// onUnavailable is called (without the mud lock) when the LLM is down, timed out, or busy.
```

Both real callbacks then do exactly what that says is unsafe. `internal/usercommands/talk.go:83-107` calls `mobs.GetInstance`, `dialogue.Load`, `m.Command(...)` and `dialogue.ShiftMood`; `ask.go:187-194` calls `dialogue.Load` + `deliverDialogue`. The `internal/dialogue` package globals (`dialogueCache`, `nilSentinel`, `moodCache`, `memoryCache`) are unguarded maps.

This is the **default** path whenever the LLM backend is down — with `LLM.Enabled: true` and Ollama not running, every `talk`/`ask` at an LLM mob takes it.

- [ ] **Step 1: Wrap all seven call sites**

`internal/llm/client.go` lines 44, 104, 118, 127, 134, 142, 149. Each becomes:

```go
util.LockMud()
onUnavailable()
util.UnlockMud()
```

matching how `onResponse` is already wrapped at `client.go:51` and `:161`.

- [ ] **Step 2: Correct the contract comment**

Update `client.go:18-19` to state that **both** callbacks are invoked under `util.LockMud()`, so a future callback author is not misled into thinking `onUnavailable` may safely block or touch unsynchronised state.

- [ ] **Step 3: Confirm no reentrancy**

`mudLock` is not reentrant. Verify that no `onUnavailable` implementation is itself reached from a path that already holds the lock. Both current implementations are invoked only from the async goroutine in `AskAsync`, so this should hold — confirm it rather than assuming.

- [ ] **Step 4: Verify** — `go build ./... && go vet ./... && go test ./internal/llm/ ./internal/usercommands/ -count=1`

- [ ] **Step 5: Commit** — `fix(llm): take the mud lock around onUnavailable callbacks`

---

## Task 5: Stop holding the connection lock across blocking writes

**The defect.** `internal/connections/connections.go:151-232`. `Broadcast` and `SendTo` hold the package-level `lock` while calling `cd.Write(...)`, which reaches a plain `net.Conn.Write`. **`SetWriteDeadline` appears nowhere in the repository** — the only deadlines anywhere are the websocket read deadlines in `heartbeat.go:52,55`.

One client with a full TCP receive window blocks that write for as long as the OS retransmission timeout. Because message dispatch runs inside `ProcessEvents` on the tick goroutine (`world.go:1146`), the entire game freezes for every player — and nobody can even be kicked, since `Remove` wants the same lock.

- [ ] **Step 1: Add a write deadline**

In `internal/connections/connectiondetails.go`, in `Write`, set a bounded deadline before each write on the TCP path:

```go
if cd.conn != nil {
    _ = cd.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
}
```

Define `writeTimeout` as a named package constant with a comment explaining that an unbounded write couples a stalled client to the game loop. Start at 5 seconds. For the websocket path use `cd.wsConn.SetWriteDeadline(...)` equivalently, taking care to keep the existing `wsLock` discipline.

- [ ] **Step 2: Snapshot targets, then write outside the lock**

Restructure `SendTo` and `Broadcast` so the package `lock` is held only long enough to collect the target `*ConnectionDetails` values into a local slice, then released **before** any `Write` call. Collect failures into `removeIds` as today and call `Remove` after, exactly as the current code already does post-unlock.

Be careful: a connection could be removed between snapshot and write. `Write` must tolerate that — verify `ConnectionDetails.Write` handles a closed/nil conn without panicking, and add a guard if it does not.

- [ ] **Step 3: Test**

Add `internal/connections/write_deadline_test.go` proving a slow reader cannot block indefinitely: create a `net.Pipe()` or a listener whose peer never reads, wrap it in a `ConnectionDetails`, write more than the socket buffer, and assert the call returns a timeout error within a bounded time rather than hanging. Use `t.Deadline()`/a timer so the test itself cannot hang the suite.

- [ ] **Step 4: Verify** — `go build ./... && go vet ./... && go test ./internal/connections/ -count=1`

- [ ] **Step 5: Commit** — `fix(connections): set write deadlines and release the lock before writing`

---

## Task 6: Clean up websocket connections that die before login

**The defect.** `main.go:1286-1312`. On a read error the handler only cleans up `if userObject != nil`, then `break`s out of a function with **no deferred cleanup**. The telnet path (`main.go:628-646`, `:787-813`) has both a `defer recover(){... connections.Remove(...)}` and a direct `connections.Remove` on read error; the websocket path has neither.

The entry never self-heals, because `Broadcast` skips connections still in `Login` state (`connections.go:160`) so no later write failure can reap it. Every visitor who closes the tab mid-signup — bounce traffic, health-check bots hitting the upgrade endpoint — permanently consumes a slot counted against `MaxHumanConnections`.

- [ ] **Step 1: Add unconditional cleanup**

Immediately after `connDetails := connections.Add(nil, conn)` at `main.go:1222`, register cleanup that runs on every exit path:

```go
defer func() {
    if r := recover(); r != nil {
        mudlog.Error("PANIC", "where", "HandleWebSocketConnection", "error", r)
    }
    // Runs on every exit path. A connection that dies during login never
    // leaves Login state, and Broadcast skips Login connections, so nothing
    // else will ever reap it.
    connections.Remove(connDetails.ConnectionId())
}()
```

Verify `connections.Remove` is idempotent — the logged-in paths already call it via `SendLogoutConnectionId`/zombie handling, so this defer will frequently be a second call. Read `Remove` and confirm; if it is not idempotent, make it so rather than conditionalising the defer.

- [ ] **Step 2: Bound incoming frames**

`SetReadLimit` appears nowhere in the repo, so gorilla's default is unlimited and `ReadMessage` buffers a whole frame before returning. After the upgrade in `internal/web/web.go:280-290`, call `conn.SetReadLimit(maxWSMessageBytes)` with a named constant (start at 64 KiB — comfortably above any legitimate GMCP frame) and a comment explaining that an unbounded frame lets one pre-login client allocate arbitrary memory.

- [ ] **Step 3: Test**

Add a test asserting that a connection added and then closed without ever logging in is not present in the registry afterwards. `connections.ActiveConnectionCount()` is the natural observable.

- [ ] **Step 4: Verify** — `go build ./... && go vet ./... && go test ./internal/connections/ ./internal/web/ -count=1`

- [ ] **Step 5: Commit** — `fix(web): reap websocket connections that die before login`

---

## Final verification

- [ ] `go build ./...`, `go vet ./...`, `gofmt -l internal/ modules/ main.go` — all clean.
- [ ] `go test ./... ` — green apart from the known Windows Defender false positive on `internal/relationships` test binaries. Do not attempt to work around antivirus; record it.
- [ ] Boot test in an **isolated worktree** on alternate ports. Never start a server in the main working directory — the user runs theirs there. Confirm the server reaches `Server Ready` with no panic, then remove the worktree.
- [ ] Exercise the two crash paths by hand against that isolated server: connect a client, press Tab repeatedly while moving between rooms (Task 2), and send `Char.Action.Try` frames in a loop (Task 3). Neither should kill the process.
- [ ] Update `docs/PATCH_NOTES.md` — these are invisible to players except as "the server stops falling over", so keep the entry short and factual.

## Notes for whoever executes this

- Tasks 2, 3 and 4 are the same bug three times. Do them together and review them as a unit; a fix to one that does not generalise is a sign the pattern was misunderstood.
- `mudLock` is **not reentrant**. Double-locking deadlocks the whole server. Before adding a lock, establish which goroutine the code runs on.
- `_datafiles/config.yaml` has git skip-worktree set. Unset before staging, re-set after committing.
- Do not start, stop, or kill the game server in the main working directory.
