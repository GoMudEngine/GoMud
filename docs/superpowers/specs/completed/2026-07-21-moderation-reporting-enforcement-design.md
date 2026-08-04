# Moderation: Reporting + Enforcement — Design

**Date:** 2026-07-21
**Status:** Approved design, pending implementation plan
**Roadmap:** `docs/PATH_TO_1.0.md` §4 (Trust / moderation once strangers interact)

## Problem

DOGMud has **no player-vs-player moderation path** today. A "stranger crowd"
(the point of the 1.0 advertising push) needs a way for a harassed player to
reach staff, and for staff to review and act — even when no admin was online at
the moment of the incident. Investigation (2026-07-21) found:

- The `report`/`rep` command is a **vital-bar status utility** (broadcasts the
  caller's own HP/SP/CP to allies) — a false friend, unrelated to abuse.
  (`internal/usercommands/report.go`.)
- There is **no** report queue, **no** persistence of complaints, and **no**
  notify-online-admins mechanism.
- The only persistent player submission is `bug` → `_datafiles/feedback/bugs.txt`
  (free text, no target, not surfaced to online staff).
- Enforcement is limited to `mute`/`deafen` (resolve target **in the same room**
  only; in-memory + a persisted `Muted`/`Deafened` flag on the user record) and
  `zap` (smite damage). There is **no `ban`** primitive anywhere, and **no
  admin "kick player X" command** (the `kick` command is a combat melee move;
  `connections.Kick` is the raw self-disconnect path).
- Staff are identified by role: `RoleAdmin`/`RoleUser` with any non-`RoleUser`
  role treated as staff; permission via `UserRecord.HasRolePermission`
  (`internal/users/userrecord.go`). No admin/mod broadcast channel exists.

A victim's only current recourse is a manual whisper to someone they happen to
know is staff. That is not acceptable for a public launch.

## Goals

1. A player command to contact staff about any issue (harassment, grief, stuck,
   request), free-text.
2. Durable, admin-reviewable **petition queue** with open/resolved status —
   nothing lost if no staff were online when it was filed.
3. Instant notification to online staff on a new petition.
4. Enforcement primitives staff currently lack: a global **server-disconnect**
   (`boot`), a persistent **ban** (account + optional IP), and **global-by-name
   targeting** for the existing speech controls (`mute`/`deafen`).

## Non-goals (YAGNI — explicitly deferred)

- **Structured accused-player field on petitions.** Petitions are free-text; the
  reporter names the offender in prose. Filtering/acting by structured target is
  a later enhancement if volume ever demands it.
- **Timed/temporary bans.** Bans are permanent; `unban` lifts them. Durations
  can be added later.
- **Automatic evidence capture** (recent chat log snippet attached to a
  petition). Requires a per-player output ring buffer; deferred.
- **A dedicated `RoleMod` constant.** The existing role + `HasRolePermission`
  mechanism already gates custom mod roles.
- **In-world justice integration.** The `internal/justice` package (jail/fine/
  bounty, NPC guards) is an in-game crime simulation and stays separate from
  out-of-world moderation.

## Architecture

A new **`internal/moderation`** package owns all moderation state and logic,
following the established living-state pattern (cf. `internal/guilds`,
`_datafiles/world/dogmud/shops/`). Two durable YAML stores under
`_datafiles/moderation/`:

- `petitions.yaml` — the petition queue.
- `bans.yaml` — account + IP ban lists.

**These are persistent living state, NOT instance saves.** They are
`.gitignore`d (never committed), persist on the prod droplet, and must **not**
be wiped by the instance-save smoke-test SOP (same policy as `shops/` and
`guilds/`). A `CLAUDE.md` note will record this.

Commands live in `internal/usercommands/`; the login and connection-accept
enforcement seams call the package's ban-check functions. This keeps the
sprawling `users`/`main.go` code touched only at two well-defined points, and
all moderation reasoning in one testable unit.

### Package API (sketch)

```go
package moderation

// --- Petitions ---
type Petition struct {
    Id         int
    Reporter   string    // username
    Timestamp  time.Time
    RoomId     int
    Zone       string
    Message    string
    Status     string    // "open" | "resolved"
    ResolvedBy string
    ResolvedAt time.Time
    Note       string
}

func Add(reporter string, roomId int, zone, message string) (Petition, error)
func ListOpen() []Petition
func ListAll() []Petition
func Get(id int) (Petition, bool)
func Resolve(id int, by, note string) error

// --- Bans ---
type AccountBan struct { Username, Reason, BannedBy string; Timestamp time.Time }
type IPBan       struct { IP, Reason, BannedBy string;       Timestamp time.Time }

func BanAccount(username, reason, by string) error
func BanIP(ip, reason, by string) error
func Unban(username string) error
func UnbanIP(ip string) error
func IsAccountBanned(username string) (reason string, banned bool)
func IsIPBanned(host string) (reason string, banned bool)

// Load/Save persistence; Load called once at startup.
```

Username/IP comparisons are case-insensitive on the username side (usernames are
already normalized elsewhere) and exact-host on the IP side.

## Components

### 1. Player command — `petition <message>`

- Registered non-admin (`{Petition, true, true, false}` style).
- Rejects an empty message with usage text.
- **Anti-spam:** per-player cooldown `PetitionCooldownRounds` (default 50). A
  too-soon petition is rejected with a friendly notice; no queue entry.
- On submit:
  1. `moderation.Add(user.Username, roomId, zone, message)` (persists).
  2. Confirm to the player (see Player-facing messages).
  3. **Notify every online staff member** (role ≠ `RoleUser`) with a one-line
     alert: `[PETITION #<id>] <reporter> (<room title>): <message>` plus a hint
     to type `petitions`.
- Auto-captures reporter, timestamp, room id + zone. Free-text body only.

### 2. Admin review — `petitions` (AdminOnly / mod-permitted)

- `petitions` — list **open** petitions: id, relative time, reporter, snippet.
- `petitions all` — include resolved.
- `petitions <id>` — full detail (reporter, when, room, full message, status).
- `petitions resolve <id> [note]` — set status `resolved`, record `ResolvedBy`
  and `ResolvedAt` and optional `note`.

### 3. Admin enforcement

- **`boot <name> [reason]`** (AdminOnly) — resolve any **online** player
  globally by name (not room-scoped). Optionally show the player the reason,
  then **fully disconnect** them via the immediate leave path (`SendLeaveWorld`
  + `SendLogoutConnectionId`), deliberately **bypassing** the `ZombieSeconds`
  linger so a booted player does not simply reconnect as a zombie. Records to
  the target's `EventLog`.
- **`ban <name> [reason]`** (AdminOnly) — permanent **account** ban:
  `moderation.BanAccount(username, reason, admin)`, and if the target is online,
  `boot` them immediately.
- **`ban ip <name|ip> [reason]`** (AdminOnly) — optional **IP** ban. Accepts an
  online player's name (resolve their connection's `RemoteAddr()` host) or a
  literal IP. Skips banning a local/loopback host.
- **`unban <name>`** / **`unban ip <ip>`** (AdminOnly) — lift the respective ban.
- **Extend `mute`/`deafen`/`unmute`/`undeafen`** to resolve a target **globally
  by name** (online or known character) in addition to the current same-room
  resolution. This makes speech-moderation usable without walking to the
  offender.

### 4. Enforcement seams

- **Account ban** — checked at **login completion**, after the username is
  identified and before/at `users.LoginUser` (`internal/inputhandlers/
  login_prompt_handler.go` completion path → the login that calls `LoginUser` in
  `internal/users/users.go:218`). A banned account is rejected with its ban
  reason and disconnected before entering the world.
- **IP ban** — checked at **connection accept**, before the login prompt is
  presented (the telnet/websocket accept path in `main.go`). Uses
  `connDetails.RemoteAddr()` (`internal/connections/connectiondetails.go:305`)
  and skips `IsLocal()` connections. A banned IP is closed immediately.

*(Exact insertion lines pinned in the implementation plan; the injection points
above are confirmed to exist.)*

### 5. Notification

New-petition alert loops over active users filtered by `Role != RoleUser` and
sends each a system-category line. No new channel type; reuses the existing
per-user `SendText` path. (A dedicated staff channel is a possible future
enhancement but not required here.)

### 6. Config & roles

- New config knobs under the appropriate block: `PetitionCooldownRounds`
  (default 50), `PetitionMaxLen` (default 500; an overlong message is
  **rejected** with a "keep it under N characters" notice, not silently
  truncated).
- All admin commands are `AdminOnly`; custom mod roles reach them through the
  existing `HasRolePermission` + `configs.GetRolesConfig()` mechanism. No new
  role constant.

## Data flow

```
petition <msg>
  → moderation.Add (persist petitions.yaml)
  → confirm to reporter
  → fan-out alert to online staff (Role != RoleUser)

admin: petitions / petitions <id> / petitions resolve <id>
  → moderation.ListOpen/Get/Resolve (persist)

admin: ban <name> <reason>
  → moderation.BanAccount (persist bans.yaml)
  → if online: boot (immediate disconnect, no zombie)

login attempt (username known)
  → moderation.IsAccountBanned → reject + disconnect if banned

connection accept (RemoteAddr known)
  → moderation.IsIPBanned → close if banned (skip IsLocal)
```

## Player-facing messages (voice + no hard numbers)

- Petition confirm (narrator, warm): *"Your petition has been sent to the staff.
  They'll review it as soon as they can."*
- Cooldown reject: *"You've just sent a petition — give the staff a moment before
  sending another."* (Describe, don't print the round count.)
- Ban-at-login: *"This account has been banned. Reason: <reason>"* (reason is the
  admin's free text).
- Boot notice to target: *"You have been disconnected by staff."* (+ reason if
  given).
- Staff alert: `[PETITION #<id>] <reporter> (<room>): <message>` — this is a
  staff-facing operational line, so the id and room are appropriate here (not a
  player-immersion context).

## Error handling

- Corrupt/missing store files: log + start empty (do not panic — these are
  runtime state, mirroring the guilds loader's log-and-skip policy, unlike
  authored content which panics).
- `boot`/`ban` on an unknown or offline name: clear "no such player online"
  (boot) / still records the account ban (ban, since the account may be offline).
- `petitions resolve` on an unknown or already-resolved id: clear error.
- Concurrency: package guards its maps with a mutex; persistence writes are
  serialized.

## Testing

- **Unit (`internal/moderation`):** petition Add → ListOpen → Get → Resolve
  round-trip incl. persistence load/save; account ban add/check/unban; IP ban
  add/check/unban; case-insensitive username match; loopback skip; cooldown
  helper.
- **Command handlers:** light tests for `boot`/`ban` argument parsing and the
  global name-resolution helper where feasible.
- **Boot-clean smoke** with the new package loading + a fresh
  `_datafiles/moderation/` created on demand.
- **Player-facing strings** reviewed for voice and the no-hard-numbers rule.
- (Enforcement seams verified in the implementation plan via a scripted
  login-rejection check; a full adversarial content playtest is not required —
  this is systems/command work, not authored content.)

## Files touched / created

**New:**
- `internal/moderation/moderation.go` (petitions + bans stores, persistence)
- `internal/moderation/moderation_test.go`
- `internal/usercommands/petition.go`, `petitions.go`, `boot.go`, `ban.go`,
  `unban.go`

**Edited:**
- `internal/usercommands/usercommands.go` — register the 5 new commands
- `internal/usercommands/admin.mute.go`, `admin.deafen.go` — global-by-name
  targeting
- `internal/inputhandlers/login_prompt_handler.go` (and/or the login completion)
  — account-ban reject
- `main.go` — connection-accept IP-ban reject
- config (network or gameplay block) — `PetitionCooldownRounds`, `PetitionMaxLen`
- `.gitignore` — `_datafiles/moderation/`
- `CLAUDE.md` — moderation-persistence note (living state, not wiped by the
  instance SOP)
- `docs/PATH_TO_1.0.md` — §4 status update

## Future enhancements (out of scope now)

- Structured accused-player field + act-by-accused from the queue.
- Temporary/timed bans with auto-expiry.
- Attach recent-chat evidence to a petition.
- A dedicated staff channel (vs. per-user fan-out).
- Warn/strike escalation tracking per player.
