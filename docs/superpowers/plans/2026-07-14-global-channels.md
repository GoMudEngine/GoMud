# Global Chat Channels Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add three tunable global channels (chat/newbie/trade) with per-player toggles, built on the existing broadcast fan-out, delivered to both terminal and web clients.

**Architecture:** A dependency-free `internal/channels` registry (channel defs + the default-on toggle rule) shared by every layer. A new `events.ChannelMessage` event + a toggle-filtered fan-out handler for terminal text. Thin talk commands + a `channels` manager, with `broadcast` re-pointed to the `chat` channel. The web path extends `gmcp.Comm.go`'s recipient routing and adds Comms tabs in the web client.

**Tech Stack:** Go; the existing `events`/`hooks` listener pattern, `userCommands` registry, `GetConfigOption`/`SetConfigOption` per-user store, GMCP `Comm.Channel`, and the `webclient-pure.html` Comms panel.

**Spec:** `docs/superpowers/specs/2026-07-14-global-channels-design.md`

---

## File map

- **Create** `internal/channels/channels.go` — registry + `Enabled`/`ShouldReceive` (pure).
- **Create** `internal/channels/channels_test.go`.
- **Modify** `internal/events/eventtypes.go` — `ChannelMessage` event type.
- **Create** `internal/hooks/ChannelMessage_SendToAll.go` — terminal fan-out (toggle-filtered).
- **Modify** `internal/hooks/hooks.go` — register the listener.
- **Create** `internal/usercommands/channel.go` — `sendChannel` helper + `Chat`/`Newbie`/`Trade`/`Channels` commands.
- **Modify** `internal/usercommands/broadcast.go` — re-point `Broadcast` to the chat channel.
- **Modify** `internal/usercommands/usercommands.go` — register `chat`/`newbie`/`trade`/`channels`.
- **Modify** `modules/gmcp/gmcp.Comm.go` — route the new CommTypes to recipients (web).
- **Modify** `_datafiles/html/public/webclient-pure.html` — Comms tabs, content divs, placeholders.
- **Modify** `PATCH_NOTES.md` — dated entry.

---

## Task 1: `internal/channels` package (registry + rules)

**Files:**
- Create: `internal/channels/channels.go`
- Test: `internal/channels/channels_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/channels/channels_test.go`:

```go
package channels

import "testing"

func TestEnabled_DefaultOn(t *testing.T) {
	if !Enabled(nil) {
		t.Error("nil (unset) should be on")
	}
	if !Enabled(true) {
		t.Error("true should be on")
	}
	if Enabled(false) {
		t.Error("false should be off")
	}
	if !Enabled("garbage") {
		t.Error("non-bool should default on")
	}
}

func TestGetAndAll(t *testing.T) {
	if len(All()) != 3 {
		t.Fatalf("expected 3 channels, got %d", len(All()))
	}
	for _, name := range []string{"chat", "newbie", "trade"} {
		if _, ok := Get(name); !ok {
			t.Errorf("channel %q should resolve", name)
		}
	}
	if _, ok := Get("nope"); ok {
		t.Error("unknown channel must not resolve")
	}
}

func TestShouldReceive(t *testing.T) {
	// Sender always sees their own echo, even toggled off / deafened.
	if !ShouldReceive(true, true, false) {
		t.Error("sender should always receive")
	}
	// Non-sender, enabled, not deafened -> receives.
	if !ShouldReceive(false, false, nil) {
		t.Error("enabled non-sender should receive")
	}
	// Non-sender, toggled off -> not.
	if ShouldReceive(false, false, false) {
		t.Error("toggled-off non-sender must not receive")
	}
	// Non-sender, deafened -> not.
	if ShouldReceive(false, true, nil) {
		t.Error("deafened non-sender must not receive")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/channels/ -v`
Expected: FAIL — package/functions undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/channels/channels.go`:

```go
// Package channels defines DOGMud's fixed set of global chat channels and the
// per-user toggle rules. It is deliberately dependency-free so the events,
// hooks, usercommands, and gmcp layers can all share it without import cycles.
package channels

// Channel is one global chat channel.
type Channel struct {
	Name      string // command + CommType, e.g. "newbie"
	ConfigKey string // per-user config-option key, e.g. "channel.newbie"
	Prefix    string // display prefix, e.g. "(newbie)"
	Color     string // ansi fg name for the prefix
}

var registry = []Channel{
	{Name: "chat", ConfigKey: "channel.chat", Prefix: "(chat)", Color: "cyan"},
	{Name: "newbie", ConfigKey: "channel.newbie", Prefix: "(newbie)", Color: "green"},
	{Name: "trade", ConfigKey: "channel.trade", Prefix: "(trade)", Color: "yellow"},
}

// All returns the channels in display order.
func All() []Channel { return registry }

// Get resolves a channel by name.
func Get(name string) (Channel, bool) {
	for _, c := range registry {
		if c.Name == name {
			return c, true
		}
	}
	return Channel{}, false
}

// Enabled applies the default-on rule: a channel is off only when the stored
// config value is explicitly the boolean false. nil (unset), true, or any
// non-bool all mean on.
func Enabled(cfgValue any) bool {
	if b, ok := cfgValue.(bool); ok {
		return b
	}
	return true
}

// ShouldReceive decides whether a user gets a channel message. The sender always
// sees their own echo; everyone else must not be deafened and must have the
// channel enabled.
func ShouldReceive(isSender, deafened bool, cfgValue any) bool {
	if isSender {
		return true
	}
	return !deafened && Enabled(cfgValue)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/channels/ -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/channels/
git commit -m "feat(channels): channel registry + toggle rules"
```

---

## Task 2: `events.ChannelMessage` event

**Files:**
- Modify: `internal/events/eventtypes.go`

- [ ] **Step 1: Add the event type**

Append to `internal/events/eventtypes.go` (near the `Broadcast` type, ~line 86):

```go
// ChannelMessage is a global chat-channel line for terminal fan-out. Recipients
// are filtered per-user by their channel toggle in ChannelMessage_SendToAll. The
// web/GMCP delivery goes through the separate Communication event.
type ChannelMessage struct {
	Channel      string // channel name, e.g. "newbie"
	SourceUserId int
	Name         string // sender display name
	Text         string // fully-formatted, ansi-tagged line (ends with CRLF)
}

func (c ChannelMessage) Type() string { return `ChannelMessage` }
```

- [ ] **Step 2: Build to verify it compiles**

Run: `go build ./internal/events/`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add internal/events/eventtypes.go
git commit -m "feat(channels): ChannelMessage event"
```

---

## Task 3: terminal fan-out handler

**Files:**
- Create: `internal/hooks/ChannelMessage_SendToAll.go`
- Modify: `internal/hooks/hooks.go`

- [ ] **Step 1: Write the handler**

Create `internal/hooks/ChannelMessage_SendToAll.go`:

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/channels"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ChannelMessage_SendToAll delivers a global chat-channel line as terminal text
// to every online user who has that channel on (the sender always sees their own
// echo). The web/GMCP copy is handled separately by gmcp.Comm.
func ChannelMessage_SendToAll(e events.Event) events.ListenerReturn {
	msg, typeOk := e.(events.ChannelMessage)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "ChannelMessage", "Actual Type", e.Type())
		return events.Continue
	}

	ch, ok := channels.Get(msg.Channel)
	if !ok {
		return events.Continue
	}

	for _, u := range users.GetAllActiveUsers() {
		if !channels.ShouldReceive(u.UserId == msg.SourceUserId, u.Deafened, u.GetConfigOption(ch.ConfigKey)) {
			continue
		}
		u.SendText(messaging.CategoryBroadcast, msg.Text)
		events.AddToQueue(events.RedrawPrompt{UserId: u.UserId}, 100)
	}

	return events.Continue
}
```

- [ ] **Step 2: Register the listener**

In `internal/hooks/hooks.go`, immediately after the existing line
`events.RegisterListener(events.Broadcast{}, Broadcast_SendToAll)` (~line 92), add:

```go
	events.RegisterListener(events.ChannelMessage{}, ChannelMessage_SendToAll)
```

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: success (no import cycle — `channels` is dependency-free).

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/ChannelMessage_SendToAll.go internal/hooks/hooks.go
git commit -m "feat(channels): terminal fan-out handler (toggle-filtered)"
```

---

## Task 4: commands (talk + manager + broadcast alias)

**Files:**
- Create: `internal/usercommands/channel.go`
- Modify: `internal/usercommands/broadcast.go`
- Modify: `internal/usercommands/usercommands.go`

- [ ] **Step 1: Write the talk helper + commands + manager**

Create `internal/usercommands/channel.go`:

```go
package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/channels"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// sendChannel formats and dispatches a global chat-channel message. It emits a
// ChannelMessage (terminal fan-out) and a Communication (web/GMCP tab).
func sendChannel(user *users.UserRecord, channelName, rest string) (bool, error) {
	ch, ok := channels.Get(channelName)
	if !ok {
		return true, nil
	}

	if user.Muted {
		user.SendText(messaging.CategoryWarning, `You are <ansi fg="alert-5">MUTED</ansi>. You can only send <ansi fg="command">whisper</ansi>'s to Admins and Moderators.`)
		return true, nil
	}

	rest = strings.TrimSpace(rest)
	if rest == `` {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`Usage: <ansi fg="command">%s</ansi> &lt;message&gt;`, ch.Name))
		return true, nil
	}

	text := fmt.Sprintf(`<ansi fg="%s">%s</ansi> <ansi fg="username">%s</ansi>: %s`,
		ch.Color, ch.Prefix, user.Character.Name, rest)

	events.AddToQueue(events.ChannelMessage{
		Channel:      ch.Name,
		SourceUserId: user.UserId,
		Name:         user.Character.Name,
		Text:         text + term.CRLFStr,
	})

	events.AddToQueue(events.Communication{
		SourceUserId: user.UserId,
		CommType:     ch.Name,
		Name:         user.Character.Name,
		Message:      rest,
	})

	return true, nil
}

func Chat(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	return sendChannel(user, "chat", rest)
}

func Newbie(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	return sendChannel(user, "newbie", rest)
}

func Trade(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	return sendChannel(user, "trade", rest)
}

// Channels lists the channels and their on/off state, or toggles one.
//   channels             -> list all with state
//   channels <name>      -> toggle
//   channels <name> on   -> enable
//   channels <name> off  -> disable
func Channels(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)

	if len(args) == 0 {
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Channels</ansi> <ansi fg="8">(channels &lt;name&gt; to toggle):</ansi>`)
		for _, ch := range channels.All() {
			state := `<ansi fg="red">off</ansi>`
			if channels.Enabled(user.GetConfigOption(ch.ConfigKey)) {
				state = `<ansi fg="green">on</ansi>`
			}
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`  <ansi fg="%s">%s</ansi> — %s`, ch.Color, ch.Name, state))
		}
		return true, nil
	}

	ch, ok := channels.Get(strings.ToLower(args[0]))
	if !ok {
		names := make([]string, 0, len(channels.All()))
		for _, c := range channels.All() {
			names = append(names, c.Name)
		}
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`No such channel. Valid channels: %s`, strings.Join(names, ", ")))
		return true, nil
	}

	newState := !channels.Enabled(user.GetConfigOption(ch.ConfigKey))
	if len(args) >= 2 {
		switch strings.ToLower(args[1]) {
		case "on":
			newState = true
		case "off":
			newState = false
		}
	}
	user.SetConfigOption(ch.ConfigKey, newState)

	word, color := "off", "red"
	if newState {
		word, color = "on", "green"
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`Channel <ansi fg="%s">%s</ansi> is now <ansi fg="%s">%s</ansi>.`, ch.Color, ch.Name, color, word))
	return true, nil
}
```

- [ ] **Step 2: Re-point `broadcast` to the chat channel**

Replace the body of `func Broadcast(...)` in `internal/usercommands/broadcast.go` so the whole file becomes:

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Broadcast is retained as an alias for the chat channel so existing muscle
// memory (and the web "broadcast" send path) keeps working, while moving player
// chat off the system-announcement stream.
func Broadcast(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	return sendChannel(user, "chat", rest)
}
```

- [ ] **Step 3: Register the new commands**

In `internal/usercommands/usercommands.go`, inside the `userCommands` map (near the
existing `` `broadcast`: {Broadcast, true, true, false}, `` entry), add:

```go
		`chat`:            {Chat, true, true, false},
		`newbie`:          {Newbie, true, true, false},
		`trade`:           {Trade, true, true, false},
		`channels`:        {Channels, true, true, false},
```

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: success. (The old `broadcast.go` imports `fmt`/`messaging`/`term` are gone; confirm no unused-import error — the rewritten file above already drops them.)

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/channel.go internal/usercommands/broadcast.go internal/usercommands/usercommands.go
git commit -m "feat(channels): chat/newbie/trade + channels manager, broadcast->chat"
```

---

## Task 5: web recipient routing (`gmcp.Comm.go`)

**Files:**
- Modify: `modules/gmcp/gmcp.Comm.go`

- [ ] **Step 1: Route the new CommTypes**

In `modules/gmcp/gmcp.Comm.go`, add the `channels` import to the import block:

```go
	"github.com/GoMudEngine/GoMud/internal/channels"
```

Then, in `onComm`, extend the recipient `if/else` chain — add this branch immediately
after the existing `} else if evt.CommType == \`whisper\` {` block (before the chain's
closing `}`):

```go
	} else if ch, ok := channels.Get(evt.CommType); ok {

		// Global chat channels: every online user who has the channel on (the
		// sender always sees their own line), mirroring the terminal fan-out.
		for _, u := range users.GetAllActiveUsers() {
			if channels.ShouldReceive(u.UserId == evt.SourceUserId, u.Deafened, u.GetConfigOption(ch.ConfigKey)) {
				sendToUserIds = append(sendToUserIds, u.UserId)
			}
		}

	}
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: success (no cycle — `gmcp` importing `channels` is fine).

- [ ] **Step 3: Commit**

```bash
git add modules/gmcp/gmcp.Comm.go
git commit -m "feat(channels): route chat/newbie/trade to web comms (toggle-filtered)"
```

---

## Task 6: web client Comms tabs

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html`

- [ ] **Step 1: Add the tab buttons**

In `webclient-pure.html`, after the broadcast tab button (line ~316), add:

```html
                <button id="comm-tab-chat" class="tab-button" data-tab="comm-chat" data-label="Chat" data-unread="0">Chat</button>
                <button id="comm-tab-newbie" class="tab-button" data-tab="comm-newbie" data-label="Newbie" data-unread="0">Newbie</button>
                <button id="comm-tab-trade" class="tab-button" data-tab="comm-trade" data-label="Trade" data-unread="0">Trade</button>
```

- [ ] **Step 2: Add the content divs**

After the `comm-broadcast` content div (line ~322), add:

```html
                <div id="comm-chat" class="chat-window chat tab-content"></div>
                <div id="comm-newbie" class="chat-window newbie tab-content"></div>
                <div id="comm-trade" class="chat-window trade tab-content"></div>
```

- [ ] **Step 3: Add the send-input placeholders**

In the `channelPlaceholders` object (line ~2795), add entries:

```js
                'chat':      'Chat (global)…',
                'newbie':    'Newbie help…',
                'trade':     'Trade: buying/selling…',
```

> No other JS changes needed: `activeCommChannel()` derives the channel from `data-tab`
> (so the Chat tab sends `chat <msg>`), and the incoming `Comm.Channel` handler already
> routes `obj.Channel.channel` to `comm-<channel>` / `comm-tab-<channel>` — which now exist.

- [ ] **Step 4: Manual smoke of the tabs (after Task 7 boot)**

Verified in Task 7's manual test (tabs appear, messages land, toggles work).

- [ ] **Step 5: Commit**

```bash
git add _datafiles/html/public/webclient-pure.html
git commit -m "feat(channels): web client Comms tabs for chat/newbie/trade"
```

---

## Task 7: patch notes + manual verification

**Files:**
- Modify: `PATCH_NOTES.md`

- [ ] **Step 1: Add a patch-notes entry**

Prepend to `PATCH_NOTES.md`:

```markdown
## 2026-07-14 — Global chat channels

Three tunable, world-wide channels so you can talk to (and hear) everyone online:
**chat** (general), **newbie** (ask for help — everyone's on it by default so there's
always someone to answer), and **trade** (buying and selling). Use `chat`, `newbie`, or
`trade` to talk; type `channels` to see them and toggle any on or off. Player chat now
has its own channels, separate from system announcements. (`broadcast` still works — it's
now the chat channel.)
```

- [ ] **Step 2: Full suite + boot-smoke**

```bash
go test ./... 2>&1 | grep -E "^(FAIL|---)" | head        # expect none
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run .   # wait for "Server Ready", no panic
```

- [ ] **Step 3: Manual multi-client check**

With the server running, connect two telnet clients (or use a socket probe) as two
characters, plus the web client:
- `newbie hello` from A → appears for B and in the web "Newbie" tab.
- On B: `channels newbie off` → B stops receiving A's next `newbie` line; A and a third
  client still get it. `channels` shows newbie as off; relog B → still off (persisted).
- `chat hi` and `trade wts sword` land on their own tabs.
- `broadcast hey` shows on the **chat** channel/tab (not a separate broadcast).
- Trigger a system announcement (e.g. wait for an autosave notice) → it appears on the
  **Broadcasts** tab only, not the chat channel.

- [ ] **Step 4: Commit**

```bash
git add PATCH_NOTES.md
git commit -m "docs(channels): patch notes"
```

---

## Final verification

- [ ] `go build ./...` clean.
- [ ] `go test ./internal/channels/` passes; full suite `go test ./...` all ok.
- [ ] Boot-smoke clean (`Server Ready`, no panic).
- [ ] Manual: talk on all three channels; toggle one off and confirm per-recipient filtering
  (terminal AND web tab); confirm persistence across relog; confirm `broadcast`→chat and that
  system announcements stay on Broadcasts.
```
