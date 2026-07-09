# Upstream Tier-1 Security/Stability Hand-Port — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Hand-port the security/stability fixes from upstream GoMudEngine/GoMud that DOGMud genuinely lacks, adapting each to DOGMud's diverged code (no cherry-pick — files have drifted).

**Architecture:** Five independent Go fixes, ordered low-risk-first. Each is verified with `go build ./...` plus a targeted test (port the upstream test, adapted to DOGMud's `users.SeedUsersForTest` helper, where one exists). Triage source: `docs/upstream-cherry-pick-triage-2026-06-08.md`. Per-PR applicability was verified against current DOGMud code 2026-06-08.

**Tech Stack:** Go. DOGMud specifics: roles `users.RoleAdmin = "admin"` / `users.RoleUser = "user"` (`internal/users/userrecord.go`); test seeding via `users.SeedUsersForTest(map[int]*UserRecord) (cleanup func())`; categorized messaging `user.SendText(messaging.Category, string)` (T9 refactor — upstream uses a single-arg legacy call).

**Already done (not in this plan):** PR #462 (SHA256→bcrypt + 0600 perms) was ported 2026-04-15 (`c2239674`). Optional leftover: port upstream's standalone `internal/users/password_test.go` (8 isolated cases) to complement DOGMud's inline tests — captured as Task 6 (optional).

---

## File Structure

- **edit** `internal/web/auth.go` — basic-auth admin-role gate (#565)
- **new** `internal/web/auth_test.go` — port upstream's table test (#565)
- **edit** `internal/inputhandlers/systemcommands.go` — system-command authz gate (#463)
- **new/port** `internal/inputhandlers/systemcommands_test.go` — adapted authz test (#463)
- **edit** `main.go` — comma-ok type-assert guards at two connection handlers (#461)
- **edit** `internal/rooms/roommanager.go` — nil-user guard in `MoveToRoom` (#460)
- **edit** `internal/rooms/save_and_load.go` — nil-room guard in `SaveRoomTemplate` (#460)
- **edit** `internal/users/userrecord.go` — `HasPlaintextPassword()` + plaintext match (#469)
- **edit** `internal/hooks/PlayerSpawn_HandleJoin.go` — gate `OnLoginCommands` on plaintext pw (#469)
- **edit** `internal/usercommands/usercommands.go` — block commands until pw changed (#469)
- **edit** `internal/usercommands/password.go` — skip current-pw prompt for plaintext users (#469)

---

## Task 1: #565 — Require admin role for web basic auth

DOGMud's `auth.go:49` admits *any* non-`user` role (builder/helper/custom) to all `/admin/*` pages. Change to admit only `admin`.

**Files:**
- Modify: `internal/web/auth.go:49`
- Test: `internal/web/auth_test.go` (new)

- [ ] **Step 1: Write the failing test**

Create `internal/web/auth_test.go`. Port the table-test idea from upstream commit `122f4070` (`internal/web/auth_test.go`) adapted to DOGMud: seed users with roles `admin`/`builder`/`user`, call the basic-auth check, assert only `admin` is authorized. Read the upstream file first (`git show 122f4070 -- internal/web/auth_test.go`) and adapt its setup to DOGMud's `users.SeedUsersForTest` and `configs` fixtures. The key assertion: a `builder`-role credential is REJECTED (the bug), `admin` accepted.

- [ ] **Step 2: Run it — verify it fails**

Run: `go test ./internal/web/ -run TestBasicAuth -v`
Expected: FAIL (builder currently authorized).

- [ ] **Step 3: Apply the one-line fix**

In `internal/web/auth.go` line 49:
```go
// BEFORE
if uRecord.Role != users.RoleUser {
// AFTER
if uRecord.Role == users.RoleAdmin {
```

- [ ] **Step 4: Run the test — verify it passes**

Run: `go test ./internal/web/ -run TestBasicAuth -v`
Expected: PASS.

- [ ] **Step 5: Build + commit**

```bash
go build ./...
git add internal/web/auth.go internal/web/auth_test.go
git commit -m "security(web): require admin role for basic auth (port upstream #565)"
```

---

## Task 2: #463 — Authorization check inside system commands

`trySystemCommand` (`/shutdown`, `/reload`) has no internal role gate — only the registration-time guard in `main.go` protects it. Add defense-in-depth.

**Files:**
- Modify: `internal/inputhandlers/systemcommands.go` (insert after the command-list lookup, ~line 110; add `users` import)
- Test: `internal/inputhandlers/systemcommands_test.go` (port, adapted)

- [ ] **Step 1: Write the failing test**

Create `internal/inputhandlers/systemcommands_test.go` adapting upstream `e014dc62`'s test to DOGMud's `users.SeedUsersForTest`: seed a non-admin user on a connection, invoke `trySystemCommand` with `cmd="reload"`, assert it returns `false` (refused) and does not execute. Also assert `cmd="quit"` is allowed for the non-admin. Read upstream first: `git show e014dc62 -- internal/inputhandlers/systemcommands_test.go`.

- [ ] **Step 2: Run it — verify it fails**

Run: `go test ./internal/inputhandlers/ -run TestTrySystemCommand -v`
Expected: FAIL (reload currently allowed for non-admin).

- [ ] **Step 3: Apply the gate**

Add import `"github.com/GoMudEngine/GoMud/internal/users"` to `systemcommands.go`. Insert immediately after the `if _, ok := systemCommandList[cmd]; !ok { return false }` block (~line 110), before the `mudlog.Info("System Command", ...)` line:
```go
// /quit is allowed for all users. Privileged commands require admin role.
if cmd != "quit" {
    user := users.GetByConnectionId(connectionId)
    if user == nil || user.Role != users.RoleAdmin {
        mudlog.Warn("Unauthorized system command attempt", "cmd", cmd, "connectionId", connectionId)
        return false
    }
}
```
(`users.GetByConnectionId`, `users.RoleAdmin`, `mudlog.Warn` all resolve in DOGMud.)

- [ ] **Step 4: Run the test — verify it passes**

Run: `go test ./internal/inputhandlers/ -run TestTrySystemCommand -v`
Expected: PASS.

- [ ] **Step 5: Build + commit**

```bash
go build ./...
git add internal/inputhandlers/systemcommands.go internal/inputhandlers/systemcommands_test.go
git commit -m "security: internal admin authz check on system commands (port upstream #463)"
```

---

## Task 3: #461 — Comma-ok type assertion in connection handlers

`main.go` does an unguarded `userObject = uo.(*users.UserRecord)` at two connection handlers; a wrong-typed `sharedState["UserObject"]` panics the whole server. Guard both.

**Files:**
- Modify: `main.go` — `handleTelnetConnection` (~line 812) and `HandleWebSocketConnection` (~line 1087)

- [ ] **Step 1: Locate both sites**

Run: `grep -n 'userObject = uo.(\*users.UserRecord)' main.go`
Expected: two lines (~812, ~1087), each inside an `if uo, exists := sharedState["UserObject"]; exists { ... }` block. Read ~10 lines around each to confirm context.

- [ ] **Step 2: Apply the guard at BOTH sites**

Replace each `userObject = uo.(*users.UserRecord)` with:
```go
var ok bool
userObject, ok = uo.(*users.UserRecord)
if !ok {
    mudlog.Error("UserObject type assertion failed", "connectionId", clientInput.ConnectionId)
    connections.Remove(clientInput.ConnectionId)
    break
}
```
(`mudlog` and `connections` are already imported/used in both functions. If `var ok bool` shadows an existing `ok` in scope, reuse the existing one instead — check the surrounding lines.)

- [ ] **Step 3: Build — verify it compiles**

Run: `go build ./...`
Expected: exit 0. (No unit test — this is a panic-guard in the top-level connection loop; verify by compilation + a boot smoke in Task 7.)

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "fix: comma-ok UserObject assertion in connection handlers (port upstream #461)"
```

---

## Task 4: #460 — Nil-pointer guards (server crash fixes)

Two of upstream's four nil-deref fixes still apply to DOGMud (the other two are already handled / the code path is gone).

**Files:**
- Modify: `internal/rooms/roommanager.go` (~line 270, `MoveToRoom`)
- Modify: `internal/rooms/save_and_load.go` (~line 218, `SaveRoomTemplate`)

- [ ] **Step 1: Guard `MoveToRoom` against a nil user**

In `internal/rooms/roommanager.go`, immediately after `user := users.GetByUserId(userId)` (~line 269) and before `currentRoom := LoadRoom(user.Character.RoomId)`:
```go
if user == nil {
    return fmt.Errorf("user %d not found", userId)
}
```
(Confirm `fmt` is imported; it almost certainly is. Verify the exact var name `user` and the following deref line.)

- [ ] **Step 2: Guard `SaveRoomTemplate` against a nil room**

In `internal/rooms/save_and_load.go`, immediately after `roomBeingReplaced := roomManager.rooms[roomTpl.RoomId]` (~line 216) and before the first use (`range roomBeingReplaced.Containers`):
```go
if roomBeingReplaced == nil {
    // New room not yet in memory — no live state to copy.
    addRoomToMemory(&roomTpl, true)
    return nil
}
```
Use `addRoomToMemory(&roomTpl, true)` (DOGMud's helper, already called at the function's normal end ~line 262) rather than a raw map assignment, so any locking/cache bookkeeping is respected. Verify the helper name/signature by reading the end of the function.

- [ ] **Step 3: Build + run room tests**

Run: `go build ./... && go test ./internal/rooms/ -v`
Expected: build exit 0; room tests pass (DOGMud has `roommanager_test.go`).

- [ ] **Step 4: Commit**

```bash
git add internal/rooms/roommanager.go internal/rooms/save_and_load.go
git commit -m "fix: nil guards in MoveToRoom + SaveRoomTemplate (port upstream #460, 2/4 applicable)"
```

---

## Task 5: #469 — Force password change for plaintext-stored passwords

DOGMud's strict bcrypt port removed the plaintext match entirely, so the default admin (`_datafiles/world/default/users/1.yaml`, `password: password`) **cannot log in on a fresh install**. Port upstream #469: detect a plaintext-stored password, allow login but lock the account to the `password` command until changed.

**Files:**
- Modify: `internal/users/userrecord.go` (`PasswordMatches` ~line 200; add `HasPlaintextPassword()`)
- Modify: `internal/hooks/PlayerSpawn_HandleJoin.go` (~line 205, `OnLoginCommands` block)
- Modify: `internal/usercommands/usercommands.go` (`TryCommand`, before the `userCommands[cmd]` lookup ~line 461)
- Modify: `internal/usercommands/password.go` (~line 15, current-pw prompt)
- Test: add `TestHasPlaintextPassword` to `internal/users/users_test.go`

- [ ] **Step 1: Write the failing test for `HasPlaintextPassword`**

In `internal/users/users_test.go`, add `TestHasPlaintextPassword`: a `UserRecord` with `Password: "password"` (plaintext) → `true`; with a bcrypt hash (`$2a$...`, generate via `SetPassword`) → `false`; with a 64-char hex SHA-256 string → `false`.

Run: `go test ./internal/users/ -run TestHasPlaintextPassword -v` → FAIL (method undefined).

- [ ] **Step 2: Add `HasPlaintextPassword()` + restore plaintext match**

In `internal/users/userrecord.go`, add (port from upstream `8a1bb436`):
```go
// HasPlaintextPassword reports whether the stored password is neither a
// bcrypt hash nor a 64-char SHA-256 hex digest — i.e. a legacy/default
// plaintext credential the user must replace.
func (u *UserRecord) HasPlaintextPassword() bool {
    p := u.Password
    if strings.HasPrefix(p, "$2a$") || strings.HasPrefix(p, "$2b$") || strings.HasPrefix(p, "$2y$") {
        return false
    }
    if len(p) == 64 {
        if _, err := hex.DecodeString(p); err == nil {
            return false
        }
    }
    return true
}
```
(Add imports `strings`, `encoding/hex` if absent.) In `PasswordMatches`, before the final `return false`, add:
```go
if u.HasPlaintextPassword() && u.Password == input {
    return true
}
```

Run: `go test ./internal/users/ -run TestHasPlaintextPassword -v` → PASS.

- [ ] **Step 3: Gate login commands on plaintext pw**

In `internal/hooks/PlayerSpawn_HandleJoin.go` (~line 205), wrap the `OnLoginCommands` execution in `if !user.HasPlaintextPassword() { ... }`, and in the `if user.HasPlaintextPassword()` branch send a "you must change your password" notice. **DOGMud adaptation:** use `user.SendText(messaging.CategorySystem, "...")` (NOT the upstream single-arg call); confirm `messaging` is imported.

- [ ] **Step 4: Block other commands until pw changed**

In `internal/usercommands/usercommands.go` `TryCommand`, immediately before `if cmdInfo, ok := userCommands[cmd]; ok {` (~line 461), insert:
```go
if user.HasPlaintextPassword() && cmd != "password" {
    user.SendText(messaging.CategorySystem, "You must change your password before doing anything else. Type: password")
    return true, nil
}
```
Match the exact return signature of `TryCommand` (read the function — it returns `(bool, error)` or similar) and DOGMud's `messaging.CategorySystem`.

- [ ] **Step 5: Skip current-pw prompt for plaintext users**

In `internal/usercommands/password.go` (~line 15), wrap the "What is your current password?" prompt + verification in `if !user.HasPlaintextPassword() { ... }` so a user who doesn't know the plaintext value can set a new one. Mirror upstream's variable-scoping (`question` declared once).

- [ ] **Step 6: Build + verify the fresh-install login path**

Run: `go build ./... && go test ./internal/users/ ./internal/usercommands/ -v`
Expected: build exit 0; tests pass. Manual confirmation happens in Task 7 (boot + log in as default admin with `password`, confirm forced change).

- [ ] **Step 7: Commit**

```bash
git add internal/users/userrecord.go internal/users/users_test.go internal/hooks/PlayerSpawn_HandleJoin.go internal/usercommands/usercommands.go internal/usercommands/password.go
git commit -m "fix(auth): force password change for plaintext-stored passwords (port upstream #469)

Also fixes the default admin (1.yaml, password: password) being unable to
log in on a fresh install after the strict bcrypt port."
```

---

## Task 6 (optional): #462 leftover — standalone password test file

DOGMud already has bcrypt (`c2239674`); only the isolated test file is missing.

- [ ] **Step 1:** `git show e3087bee -- internal/users/password_test.go` to read upstream's 8-case file.
- [ ] **Step 2:** Create `internal/users/password_test.go`, adapting only as needed (the `UserRecord` API + bcrypt import are identical). Drop any case already covered inline in `users_test.go` to avoid duplication, or keep all 8 if non-overlapping.
- [ ] **Step 3:** `go test ./internal/users/ -v` → PASS. Commit: `test(users): port upstream standalone bcrypt password tests (#462 leftover)`.

---

## Task 7: Boot smoke + full build/test gate

**Files:** none (verification)

- [ ] **Step 1:** `go build ./...` → exit 0.
- [ ] **Step 2:** `go test ./internal/web/ ./internal/inputhandlers/ ./internal/rooms/ ./internal/users/ ./internal/usercommands/ 2>&1 | tail` → all pass.
- [ ] **Step 3:** Nuke instance saves per SOP, boot the server, watch for clean data-file load (no panics). Confirm `Server Ready`.
- [ ] **Step 4 (manual):** With a fresh/default `users/1.yaml` (`password: password`), connect and log in as the default admin — confirm login succeeds and the account is forced to `password` before any other command works (validates #469). Confirm a non-admin can't reach `/admin/*` over basic auth (validates #565).

---

## Self-Review notes (addressed)

- **Triage coverage:** #565 (T1), #463 (T2), #461 (T3), #460 ×2 applicable (T4), #469 (T5); #462 already done → optional test (T6). All Tier-1 candidates accounted for.
- **DOGMud adaptations called out explicitly:** `users.SeedUsersForTest` for tests; `messaging.CategorySystem` first-arg on all `SendText` in #469 (the easy-to-miss T9 divergence); `addRoomToMemory` (not raw map assignment) in the #460 SaveRoomTemplate guard; reuse existing `ok` if shadowed in #461.
- **Type consistency:** roles `users.RoleAdmin`/`users.RoleUser`; `users.GetByConnectionId` / `GetByUserId`; `HasPlaintextPassword()` referenced identically across userrecord/hooks/usercommands/password.
