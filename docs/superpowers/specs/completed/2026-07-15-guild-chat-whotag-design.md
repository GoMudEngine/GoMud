# Guild chat + who-tag

**Date:** 2026-07-15
**Status:** Design (approved; pending review)
**Arc:** Social guilds — sub-project #2 (after membership core). Small slice.
**Depends on:** `internal/guilds` (membership core, shipped).

---

## 1. Goal

Make guilds *feel* like guilds: a members-only chat, and visible `[TAG]` membership in the
room. Both are thin additions over the shipped membership core.

---

## 2. Approved decisions

1. **Guild chat = a targeted broadcast** (party-chat pattern), NOT a global channel — it goes
   only to online guild members. `guild chat <msg>` subcommand **plus a top-level `gc <msg>`
   alias**. No deafen toggle in v1 (you're in the guild, like party chat).
2. **who-tag**: a guilded player renders as `[TAG] Name` in the room "also here" line.

---

## 3. Guild chat — `internal/usercommands/guild.go` (+ a `gc` command)

Shared helper `guildChatSend(user, msg)`:
- `g, ok := guilds.GetByUser(user.UserId)`; not in a guild → hint; empty msg → usage.
- To each **online** member except the sender: `(guild) <name>: <msg>` (ANSI, guild-cyan
  prefix, 80-col wrapped) via `users.GetByUserId(m.UserId).SendText`.
- Self-echo: `(guild) You: <msg>`.
- Emit `events.Communication{SourceUserId, CommType: "guild", Name, Message: msg}` for the
  web/GMCP comm tab (mirrors party chat).

Wiring:
- `guild chat <msg>` → the `"chat"` case in the `Guild` dispatcher calls `guildChatSend`.
- New top-level command `Gc(rest, user, room, flags)` → `guildChatSend(user, rest)`.
  Registered `` `gc`: {Gc, true, true, false} ``; allowlisted `"gc": "player-mechanic"` in
  `internal/actions/divergences.go`; a brief `gc` help template (points to `guild chat`).
- Update `guild.template` to list `guild chat <msg>`.

Muted players: respect `user.Muted` (refuse, like `sendChannel`) — a muted player can't guild-chat.

---

## 4. who-tag — `internal/rooms/roomdetails.go`

In `GetDetails`, where each visible player entry is built (`playerEntry := pName.String()`,
~line 263), prepend the guild tag:

```go
playerEntry := pName.String()
if tag := guilds.TagForUser(player.UserId); tag != "" {
    playerEntry = fmt.Sprintf(`<ansi fg="cyan">[%s]</ansi> %s`, tag, playerEntry)
}
```

`rooms` importing `guilds` is import-cycle-safe (`guilds` imports neither `rooms` nor
`characters`). Empty tag → no prefix (unchanged for guildless players). Confirm the loop
variable exposes `player.UserId` (it's a `*users.UserRecord`).

The ASCII `map`/other displays are unaffected; this is only the room-occupant list.

---

## 5. Testing

- **`guildChatSend` recipient logic** — factor the "who receives" into a pure helper
  `guildChatRecipients(g, senderId) []int` (online members except sender) and unit-test it
  (returns the right member userIds; excludes sender; skips offline). The formatting/SendText
  is integration (boot/manual).
- **who-tag** — a small helper `guildTagPrefix(tag, name) string` (or test `TagForUser`
  already covered); assert `[TAG] Name` when tagged, unchanged when not. (Registry `TagForUser`
  is already tested; the prepend is trivial — a focused unit test on the format helper.)
- Full suite green + boot clean (no CommandParity warning for `gc`).

---

## 6. Out of scope

- Deafen/toggle for guild chat, guild-chat history, cross-server. (Later if wanted.)
- Deeper name/prompt integration beyond the room "also here" line.

---

## 7. Files touched

- `internal/usercommands/guild.go` — `guildChatSend` + `guildChatRecipients` + `"chat"` case.
- `internal/usercommands/guild_chat.go` (or same file) — `Gc` command.
- `internal/usercommands/usercommands.go` — register `gc`.
- `internal/actions/divergences.go` — allowlist `gc`.
- `internal/rooms/roomdetails.go` — `[TAG]` prefix.
- `_datafiles/world/dogmud/templates/help/gc.template` (+ default) + update `guild.template`.
- `PATCH_NOTES.md`, `docs/PATH_TO_1.0.md` (note guild chat + who-tag done).
