# Guild chat + who-tag Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** A members-only `guild chat` (+ `gc` alias) and a `[TAG]` prefix on guilded players in the room "also here" line.

**Architecture:** Guild chat is a targeted broadcast (party-chat pattern) over `internal/guilds` members; the recipient set is a pure, tested helper. The who-tag is a one-line prepend in `rooms.GetDetails` (rooms→guilds is cycle-safe).

**Spec:** `docs/superpowers/specs/2026-07-15-guild-chat-whotag-design.md`

---

## Task 1: Guild chat (`guild chat` + `gc`)

**Files:** `internal/usercommands/guild.go` (+ `Gc`), `usercommands.go`, `internal/actions/divergences.go`, help templates. Test: `internal/usercommands/guild_test.go` (create).

- [ ] **Step 1: Write the failing test**

`internal/usercommands/guild_test.go`:

```go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/guilds"
)

func TestGuildChatRecipients(t *testing.T) {
	g := &guilds.Guild{Tag: "QC", Members: []guilds.GuildMember{
		{UserId: 1}, {UserId: 2}, {UserId: 3},
	}}
	got := guildChatRecipients(g, 2)
	if len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Errorf("recipients = %v, want [1 3] (members except sender)", got)
	}
	if len(guildChatRecipients(&guilds.Guild{}, 1)) != 0 {
		t.Error("empty guild -> no recipients")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/usercommands/ -run TestGuildChatRecipients -v`
Expected: FAIL — `guildChatRecipients` undefined.

- [ ] **Step 3: Implement chat**

Add to `internal/usercommands/guild.go`:

```go
// guildChatRecipients returns the member userIds to deliver a guild-chat line to
// (all members except the sender). The caller filters offline members.
func guildChatRecipients(g *guilds.Guild, senderId int) []int {
	out := []int{}
	for _, m := range g.Members {
		if m.UserId != senderId {
			out = append(out, m.UserId)
		}
	}
	return out
}

// guildChatSend broadcasts a guild-chat line to online members + echoes to the
// sender + emits a Communication event for the web/GMCP comm tab.
func guildChatSend(user *users.UserRecord, msg string) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	if user.Muted {
		user.SendText(messaging.CategoryWarning, `You are <ansi fg="alert-5">MUTED</ansi>.`)
		return
	}
	msg = strings.TrimSpace(msg)
	if msg == "" {
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">guild chat &lt;message&gt;</ansi>  (or <ansi fg="command">gc &lt;message&gt;</ansi>)`)
		return
	}
	for _, uid := range guildChatRecipients(g, user.UserId) {
		if u := users.GetByUserId(uid); u != nil {
			line := fmt.Sprintf(`<ansi fg="cyan">(guild)</ansi> <ansi fg="username">%s</ansi>: <ansi fg="white">%s</ansi>`, user.Character.Name, msg)
			u.SendText(messaging.CategorySystem, util.SplitStringNL(line, 80))
		}
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="cyan">(guild)</ansi> You: <ansi fg="white">%s</ansi>`, msg))
	events.AddToQueue(events.Communication{
		SourceUserId: user.UserId,
		CommType:     `guild`,
		Name:         user.Character.Name,
		Message:      msg,
	})
}
```

Add `"github.com/GoMudEngine/GoMud/internal/util"` to guild.go imports (for `SplitStringNL` —
confirm the exact name/signature against `party.go`'s usage).

Wire `guild chat` into the dispatcher (in `Guild`'s switch):

```go
	case "chat", "say", "gc":
		guildChatSend(user, remainder)
```

Add the top-level `Gc` command (same file or `guild_chat.go`):

```go
func Gc(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	guildChatSend(user, rest)
	return true, nil
}
```

- [ ] **Step 4: Register + allowlist + help**

- `usercommands.go`: `` `gc`: {Gc, true, true, false}, `` (near `guild`).
- `divergences.go`: `"gc": "player-mechanic",` (alphabetical).
- Create `_datafiles/world/dogmud/templates/help/gc.template` (brief: "Speak to your guild.
  Same as `guild chat`.") + copy to `default/`.
- Add a `guild chat <msg>` line to `guild.template` (both copies).

- [ ] **Step 5: Run test + build + full usercommands test**

Run: `go test ./internal/usercommands/ -run TestGuildChatRecipients -v` → PASS
Run: `go build ./... && go test ./internal/usercommands/` → PASS (help completeness for `gc`).

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/ internal/actions/divergences.go _datafiles/world/dogmud/templates/help/gc.template _datafiles/world/default/templates/help/gc.template _datafiles/world/dogmud/templates/help/guild.template _datafiles/world/default/templates/help/guild.template
git commit -m "feat(guilds): guild chat (guild chat + gc alias)"
```

---

## Task 2: who-tag in the room "also here" line

**Files:** `internal/rooms/roomdetails.go`

- [ ] **Step 1: Prepend the tag**

In `GetDetails`, right after `playerEntry := pName.String()` (~line 263):

```go
playerEntry := pName.String()
if tag := guilds.TagForUser(playerId); tag != "" {
	playerEntry = fmt.Sprintf(`<ansi fg="cyan">[%s]</ansi> %s`, tag, playerEntry)
}
```

`playerId` is the loop variable (`for _, playerId := range r.players`). Add the `guilds` import
to `roomdetails.go` (`fmt` is already imported). Verify no import cycle at build (guilds imports
neither rooms nor characters — safe).

- [ ] **Step 2: Build + commit**

Run: `go build ./...` → success.
```bash
git add internal/rooms/roomdetails.go
git commit -m "feat(guilds): [TAG] prefix for guilded players in the room list"
```

---

## Task 3: Boot smoke test + docs

- [ ] **Step 1: Build + touched tests**

Run: `go build ./... && go test ./internal/usercommands/ ./internal/rooms/ ./internal/guilds/`
Expected: all `ok`.

- [ ] **Step 2: Boot smoke test**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Run `go run .`; confirm `Server Ready`, no panic, and **no CommandParity warning for `gc`**; stop.

- [ ] **Step 3: Docs**

- `PATCH_NOTES.md`: brief entry (guild chat + `gc`; guild tag now shows by your name in a room).
- `docs/PATH_TO_1.0.md` §3 guilds line: note guild chat + who-tag done; treasury + ranks-polish
  remain (of the user's picked set).

- [ ] **Step 4: Commit**

```bash
git add PATCH_NOTES.md docs/PATH_TO_1.0.md
git commit -m "docs(guilds): patch notes + roadmap for guild chat + who-tag"
```

---

## Notes for the implementer

- **Verify before coding:** `util.SplitStringNL` (name/sig, per `party.go`),
  `events.Communication` fields (per `channel.go`/`party.go`), `user.Muted`, the `r.players`
  loop var in `roomdetails.go`.
- **No emoji**; ANSI only. 80-col wrap guild-chat lines.
- Guild chat is a **targeted** send (members only) — do NOT route it through the global
  `events.ChannelMessage` fan-out.
```
