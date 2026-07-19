# Global Chat Channels — Design

**Date:** 2026-07-14
**Status:** Approved (brainstorm), ready for implementation plan
**Goal:** Give players a small set of tunable, topical global channels — general chat,
a newbie/help channel, and trade — so a newcomer *feels* other players and can ask for
help world-wide, without drowning in system spam. A path-to-1.0 community/retention item.

---

## 1. Current state & the gap

DOGMud already has **one** global player channel: the `broadcast` command (its code comment
says "Global chat room"). It fans out to all online users via `events.Broadcast` and shows
in the web client's "Broadcasts" Comms tab.

The gaps:
1. **No tunability.** There is no per-player channel toggle infrastructure — players can't
   turn a channel off to cut the noise.
2. **The broadcast stream is a firehose.** `events.Broadcast` carries player chat *and* every
   system announcement (new-character, sunrise/sunset, moon phase, death announcements,
   autosave). Player chat drowns.
3. **No newbie/help channel.**

The reusable plumbing that already exists:
- Per-character persistent settings: `UserRecord.GetConfigOption(key) any` /
  `SetConfigOption(key, value any)`. This is where toggles live — no new persistence.
- The web Comms tabs are driven by `events.Communication{CommType: ...}` — adding a channel
  = a new `CommType` + a tab.
- The `Deafened` flag + `Muted` flag are already honored by the comm commands.

---

## 2. Channels

Three fixed channels, each **on by default** and individually toggleable:

| Channel | CommType | Purpose | Default |
|---|---|---|---|
| `chat`   | `chat`   | General OOC (replaces player use of `broadcast`) | on |
| `newbie` | `newbie` | Help for newcomers (everyone on by default = more helpers) | on |
| `trade`  | `trade`  | Buying/selling chatter (fits the living economy) | on |

Each channel has a colored prefix, e.g. `(chat)`, `(newbie)`, `(trade)`, styled via the
existing ansi color-pattern system (mirroring `broadcast-prefix`).

---

## 3. Architecture

Additive; mirrors the existing `broadcast` → `events.Broadcast` → `Broadcast_SendToAll`
pattern, but with a **per-recipient toggle filter** — which is the entire point.

### 3.1 `internal/channels/channels.go` (new package) — the registry (pure, testable)
```go
type Channel struct {
    Name      string // "newbie"
    ConfigKey string // "channel.newbie"
    Prefix    string // "(newbie)"
    Color     string // ansi color name for the prefix
}
```
- A fixed slice/map of the three `Channel`s + `Get(name string) (Channel, bool)` and `All() []Channel`.
- `Enabled(cfgValue any) bool` — the **default-on** rule: returns `false` only when the stored
  value is explicitly the boolean `false`; `nil` (unset) and `true` both mean on. This is the
  single source of truth for "is this channel on for a user", unit-tested in isolation.

Kept dependency-free (no `users`/`events` imports) so `hooks`, `usercommands`, and the web
layer can all reference it without cycles.

### 3.2 `events.ChannelMessage` (new event)
```go
type ChannelMessage struct {
    Channel      string // "newbie"
    SourceUserId int
    Name         string // sender display name
    Text         string // fully-formatted, ansi-tagged line (prefix + name + body)
}
```
Distinct from `events.Broadcast` so player chat and system announcements stay separate.

### 3.3 `internal/hooks/ChannelMessage_SendToAll.go` (new handler)
Registered like `Broadcast_SendToAll`. For each `users.GetAllActiveUsers()`:
- skip if `u.Deafened` (globally deaf) — consistent with broadcast,
- skip if `!channels.Enabled(u.GetConfigOption(ch.ConfigKey))` (the toggle filter),
- otherwise `u.SendText(<comm category>, text)` + queue a prompt redraw. (The exact
  `messaging.Category` is the one the existing comm path uses; the plan pins it.)

The **sender always sees their own line** regardless of their toggle (echo), so talking never
looks like it did nothing.

### 3.4 `internal/usercommands/` — the commands
- **Shared helper** `sendChannel(user, channelName, msg)`:
  1. `Muted` gate (same message `broadcast` uses).
  2. Reject empty message with a usage line.
  3. Format: `(<chan>) <name>: <body>` with the channel's color.
  4. Emit `events.ChannelMessage{...}` (fan-out) **and** `events.Communication{CommType: chan, ...}` (web tab).
- **Talk commands** `chat.go`, `newbie.go`, `trade.go` — thin wrappers over `sendChannel`.
- **`channels.go`** — the manager:
  - `channels` (no args): lists all three channels with on/off state and usage.
  - `channels <name>`: toggles that channel (writes `SetConfigOption(ConfigKey, newBool)`).
  - `channels <name> on|off`: sets explicitly.
- **`broadcast` becomes an alias for `chat`** — existing muscle memory keeps working, and player
  chat moves off the `events.Broadcast` system stream onto the `chat` channel.

### 3.5 Web client (`_datafiles/html/public/static/js/gmcp.js`)
Add Comms tabs for the new `CommType`s (`chat`/`newbie`/`trade`). Terminal clients work
immediately via the colored prefixes; the tabs are additive frontend wiring. The "Broadcasts"
tab now naturally shows system announcements only.

### Data flow
```
player: "newbie how do I equip?"
  └─ Newbie cmd → sendChannel(u, "newbie", msg)
        ├─ Muted? refuse
        ├─ format "(newbie) Bob: how do I equip?"
        ├─ emit events.ChannelMessage{Channel:"newbie", ...}
        │     └─ ChannelMessage_SendToAll: for each online user,
        │          skip if Deafened or channel toggled off → else SendText
        └─ emit events.Communication{CommType:"newbie", ...}  → web "Newbie" tab
```

---

## 4. Toggle semantics

- Storage key: `channel.<name>` in the per-user config-option store (persists across sessions).
- Default **on**: a channel is off for a user only if they explicitly turned it off
  (`channels.Enabled` treats `nil`/`true` as on, `false` as off).
- Toggling off means you neither **hear** the channel nor see others' messages on it; you can
  still send (you'll see your own echo) — but the manager listing tells you it's off so it's
  not surprising.
- `Muted` (moderation) blocks **sending** on all channels (existing behavior). `Deafened`
  blocks **receiving** all global comms (existing behavior).

---

## 5. Edge cases

- **Empty message** (`newbie` with no text): print a one-line usage, don't broadcast.
- **Unknown channel** in `channels <name>`: friendly error listing valid names.
- **Muted sender**: refused with the standard muted message.
- **No other players online**: sender still sees their own echo; no error.
- **Config value round-trip**: a stored `false` must come back as boolean `false` from
  `GetConfigOption` for the filter to work (verified in the plan; the existing toggle settings
  like combatverbosity prove the pattern).

---

## 6. Testing

**Unit:**
- `channels.Enabled`: `nil`→true, `true`→true, `false`→false (the default-on rule).
- `channels.Get`/`All`: the three channels resolve; unknown name fails.
- `sendChannel` mute gate: a muted user's send is refused and emits no event (test via a fake
  user + draining the event queue, mirroring existing comm-command tests).
- Recipient filter (handler-level or an extracted pure predicate): a user with `channel.newbie=false`
  is filtered out; `nil`/`true` users are included; a `Deafened` user is excluded.

**Manual:**
- Two connections. `newbie hi` from A appears for B. `channels newbie off` on B, then A's next
  `newbie` line does **not** reach B but still reaches a third client. Confirm `chat`/`trade`
  independently. Confirm `broadcast hi` now shows on the chat channel and system announcements
  (e.g. an autosave notice) show on Broadcasts only.

---

## 7. Out of scope / future

- Arbitrary config-defined channels, per-channel roles/permissions, join/leave, who's-listening
  (the "full framework" option — deferred; three fixed channels serve 1.0).
- Channel history/scrollback beyond the web client's existing tab buffering.
- Cross-server / Discord bridging of the new channels (the existing Discord bridge is untouched).
- Auto-muting the newbie channel for veterans (kept simple: everyone on by default, toggle at will).

---

## 8. Success criteria

- A player can `chat`/`newbie`/`trade` and every online player with that channel on sees it.
- `channels` lists the three with state; toggling one off stops that player receiving it while
  others keep receiving; the setting persists across logout.
- Player chat no longer mixes into the system-announcement stream.
- `broadcast` still works (as `chat`). Muted/Deafened behavior unchanged.
- Web client shows the new channels as Comms tabs; terminal shows colored prefixes.
