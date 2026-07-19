# Copyover / Hot-Reboot — Design

**Date:** 2026-07-14
**Status:** Approved (brainstorm), ready for implementation plan
**Goal:** Restart the live server (to deploy updates) **without disconnecting players** — a
brief pause and a "Copyover complete." instead of a full drop/reconnect. Critical during a
launch push when we iterate constantly. A path-to-1.0 advertising-critical item.

---

## 1. Approach: port upstream, hand-ported

Upstream GoMud already has a complete, tested copyover system on `upstream/master` (merge
`7fd805da0`, "Copyover support (#474)"). We **port it**, not rebuild it. A clean cherry-pick is
impossible — the upstream merge drags in 400+ unrelated files, and our `main.go`/`connections`
have diverged — so it's a **hand-port of the copyover-specific files + re-wiring into our
`main.go`**, plus one DOGMud-specific addition (the economy flush, §5).

Upstream ships a design doc we inherit: `internal/copyover/context.md` (read it — it's the
authoritative mechanism reference).

---

## 2. Architecture (from upstream)

- **Trigger** — an admin `copyover` command or `SIGUSR1` → `triggerCopyover()` (package `main`):
  `serverAlive.Store(false)` (stop accepting new connections) → flush state to disk → `copyover.Execute()`.
- **Execute** (`internal/copyover/copyover.go`) — open an `os.Pipe`, call `copyover.Save(writeFd)`
  (iterates registered `Contributor`s, each writing a named JSON section), close writeFd, then
  `exec` the same binary with the read pipe as `ExtraFiles[0]` (fd 3) and `--copyover-fd=3`
  appended, then `os.Exit(0)`.
- **Connection hand-off** — the `connections` contributor clears `FD_CLOEXEC` on each raw TCP
  socket so telnet FDs survive the `exec` (`internal/connections/fd_unix.go`). **WebSockets can't
  transfer** → they get a disconnect notice and the web client **auto-relogs** (upstream JS in
  `webclient-pure.html`).
- **Restore** — the child detects `flags.CopyoverFd() >= 0`, reads the sections, calls each
  `Contributor.CopyoverRestore`, then resumes each restored connection's I/O loop
  (`resumeRestoredConnection`). Normal-startup steps that would corrupt restored state
  (**user-index rebuild, round-count reload from disk, data migrations**) are **skipped** when the
  copyover fd is set.
- **Contributor interface** (`CopyoverName/CopyoverSave/CopyoverRestore`, JSON sections) — the seam
  for transient in-memory state. **On-disk state is NOT a contributor**: it's saved before the exec
  and reloaded by normal startup.

### Why most DOGMud state "just works"
Copyover saves everything to disk, re-execs, and reloads. So our added systems — mutations, channel
toggles, pinnacle-item MiscData, warehouses (already `SaveDirty` every round) — are preserved via
the normal save/load. Contributors only carry what is **not** derivable from disk: the live
connections, the global round counter, and in-flight combat state.

---

## 3. Files to port

**Port ~as-is (new files, platform-aware):**
- `internal/copyover/` — the whole package: `copyover.go` (Execute/Save/Restore/Register + Encoder/
  Decoder), `tokens.go`, `sysproc_unix.go`, `sysproc_windows.go`, `context.md`, `copyover_test.go`.
- `internal/connections/copyover.go` (connections contributor), `fd_unix.go` (FD_CLOEXEC clearing +
  raw-fd extraction), `fd_windows.go` (stub).
- `copyover.go` (root — `triggerCopyover`), `copyover_signal_unix.go` (SIGUSR1 handler),
  `copyover_signal_windows.go` (stub).
- `internal/gametime/copyover.go` (round-counter contributor).
- `internal/usercommands/admin.copyover.go` (the `copyover` command + `SetCopyoverFunc`).

**Small edits to existing files:**
- `internal/flags/flags.go` — add the `--copyover-fd` flag (+`CopyoverFd()` accessor).
- `internal/connections/connections.go` — the small hook the port needs.

**Adapt to our divergence (the real work):**
- `main.go` — §4.
- `_datafiles/html/public/webclient-pure.html` — the websocket auto-relog JS (port the upstream
  diff; our webclient has diverged, so re-apply by hand).

---

## 4. `main.go` integration

Our `main.go` already has the hooks copyover needs — `serverAlive atomic.Bool`, `allServerListeners`,
`signal.Notify(...)`, and an accept-gate on `serverAlive.Load()`. The wiring:

1. **Boot-time branch** — early in `main()`, after config load and **after registering all
   contributors** (`copyover.Register(connections.CopyoverContributor())`, `gametime...`, combat, …),
   check `flags.CopyoverFd()`. If `>= 0`: call `copyover.Restore(fd)`, then **skip** the normal
   user-index rebuild, `.roundcount` load, and migration passes; after the world is up, spawn
   `resumeRestoredConnection` per restored connection.
2. **`triggerCopyover()`** — add the DOGMud version (see §5 for the flush additions).
3. **Signal** — extend `signal.Notify` to also catch `SIGUSR1` (Unix only, via
   `copyover_signal_unix.go`) → `triggerCopyover()`.
4. **`SetCopyoverFunc`** — pass `triggerCopyover` to the usercommands package so the `copyover`
   command can call it.
5. **Register the `copyover` command** in the `userCommands` map (admin-appropriate access) + a help
   file (the help-completeness test requires one).

---

## 5. The economy flush (DOGMud-specific, folds into `triggerCopyover`)

Several living-economy subsystems persist their in-memory changes **only on graceful shutdown**, so
a copyover that doesn't flush them would silently rewind that state (foraged-node depletion, caravan
restock progress, shop stock/prices) — the one visible seam in an otherwise-invisible reboot. The
fix is the same flush graceful shutdown already does, called before `copyover.Execute()`:

```go
func triggerCopyover() error {
	binaryPath, err := os.Executable()
	if err != nil {
		return err
	}
	serverAlive.Store(false)

	rooms.SaveAllRooms()
	users.SaveAllUsers()
	// Living-economy dirty-state (else a copyover rewinds it — keep the reboot seamless).
	shops.SaveAllShops()
	warehouse.SaveAll()
	forager.SaveAllThroughputs()
	caravan.SaveAllThroughputs()
	opinions.SaveAllOpinions()
	plugins.Save()

	return copyover.Execute(binaryPath, os.Args[1:])
}
```

All five functions exist and match graceful-shutdown behavior. This is purely additive and cheap.

---

## 6. Contributors

Port upstream's contributors and adapt to our types:
- **connections** — the live sockets (telnet FDs transferred; websockets flagged for relog).
- **gametime** — the global round counter (so combat/aging timers don't jump; the copyover'd value
  is used instead of the skipped `.roundcount` disk load).
- **combat/in-flight state** — upstream added a contributor for combat state + buff countdowns. Our
  combat is heavily diverged (unified damage pipeline, mutations); the port adapts this contributor
  to our `Aggro`/round fields, or — if our combat state is fully reconstructable from the saved
  character (buffs are on the character, saved) + round counter — we keep it minimal. **The plan
  audits this and adds only what can't reload from disk.**

**Audit task (spec-mandated):** sweep DOGMud subsystems for transient in-memory state not on disk
and not covered above (candidates: ferry vessel state — derivable from round counter + data, so no;
weather — plugin persistence, so no; conversation/schedule/patrol — MiscData, saved). Expected
result: connections + gametime + (maybe) combat. Add a contributor only where genuinely needed.

---

## 7. Windows & testing (per the validation decision)

- **Windows**: the `_windows.go` stubs make copyover a **compile-time no-op** — the build and boot
  are unaffected on dev. The `copyover` command / SIGUSR1 simply do nothing on Windows.
- **On Windows dev I verify**: `go build ./...` clean, the ported **`internal/copyover/copyover_test.go`
  passes** (platform-agnostic Save/Restore round-trip via `FuncContributor`), the full suite passes,
  and boot-smoke is clean (normal startup unaffected; `copyover` command registered).
- **Real validation is on the droplet** (agreed): after deploy, run `copyover` (or `kill -SIGUSR1`)
  with a test telnet connection + a web-client tab connected, and confirm the telnet session survives
  the reboot and the web client auto-relogs. Suggested first run: low-traffic window, test char only.

---

## 8. Edge cases & risks

- **Restore error → fatal**: if any `CopyoverRestore` fails, `main` exits(1). The operator sees the
  failed restart (players dropped) rather than a half-restored server. Acceptable and correct.
- **Binary path**: `os.Executable()` must resolve; on the droplet the container path is stable.
- **In-flight combat rounds**: preserved via the round counter + on-disk buffs; a mid-swing tick is
  effectively paused and resumed. Verify no double-tick on resume during droplet testing.
- **go.mod**: copyover uses stdlib (`os/exec`, `syscall`, `os.Pipe`) — no new deps expected; the plan
  confirms and drops any unrelated go.mod churn from the upstream diff.
- **AI port / non-telnet connections**: treated like telnet (raw TCP) for FD transfer, or dropped
  gracefully; verify the AI port behaves on the droplet.

---

## 9. Out of scope / future

- Preserving WebSocket connections without a relog (not feasible; auto-relog is the accepted answer).
- A scheduled/automatic copyover on deploy (manual `copyover` command is enough for 1.0).
- Copyover of plugin runtime state beyond the existing `onSave`/`onLoad` disk persistence.

---

## 10. Success criteria

- `go build ./...` clean on Windows; `internal/copyover` unit tests + full suite pass; boot-smoke clean.
- On the droplet: a `copyover` keeps a telnet player connected through the restart ("Copyover
  complete." not a disconnect); the web client auto-relogs.
- No economic rewind after a copyover (forage/caravan/shop state intact).
- Normal (non-copyover) startup and shutdown are unchanged.
