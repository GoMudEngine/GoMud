# ASCII Charset Mode (zMUD Fix) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a per-user `set charset` toggle that converts UTF-8 box-drawing characters to ASCII equivalents, fixing display corruption and flashing on legacy clients like zMUD.

**Architecture:** A `ConvertToAscii()` function in `internal/util/` replaces multi-byte UTF-8 box-drawing/block characters with ASCII equivalents. An `AsciiMode` flag on `ClientSettings` gates the conversion in `ConnectionDetails.Write()` — the single output funnel for all data sent to a client. The flag is persisted on `UserRecord` and restored on login.

**Tech Stack:** Go, existing `connections`/`users`/`util` packages, existing `set` command infrastructure.

---

### Task 1: Add ConvertToAscii utility function

**Files:**
- Modify: `internal/util/util.go` (append new function)
- Modify: `internal/util/util_test.go` (or create if needed — check for existing test file)

This function maps UTF-8 box-drawing, block element, bullet, and other common Unicode decorative characters to their closest ASCII visual equivalents. It fast-paths when no bytes >= 0x80 are present (pure ASCII input returns unchanged).

- [ ] **Step 1: Write the failing test**

Add to the test file for `internal/util/`:

```go
func TestConvertToAscii(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"pure ascii passthrough", "Hello world!", "Hello world!"},
		{"box corners", "┌─┐└─┘", "+-++-+"},
		{"box sides", "│text│", "|text|"},
		{"double box", "╔═╗║x║╚═╝", "+=+|x|+=+"},
		{"mixed box", "╒═╕╘═╛", "+=++=+"},
		{"intersections", "├┤┬┴┼", "+++++"},
		{"double intersections", "╠╣╦╩╬", "+++++"},
		{"block elements", "█░▒▓", "#.:# "},
		{"half blocks", "▄▀▌▐", "-_||"},
		{"bullet", "• item", "* item"},
		{"ansi preserved", "\x1b[32mgreen\x1b[0m", "\x1b[32mgreen\x1b[0m"},
		{"mixed ansi and unicode", "\x1b[33m┌─┐\x1b[0m", "\x1b[33m+-+\x1b[0m"},
		{"status template chars", " ┌─ Attributes ─┐\n └──────────────┘", " +- Attributes -+\n +---------------+"},
		{"motd chars", "╔══╗\n║hi║\n╚══╝", "+=+=+\n|hi|\n+=+=+"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToAscii(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertToAscii(%q)\n  got:  %q\n  want: %q", tt.input, result, tt.expected)
			}
		})
	}
}
```

**Note:** The exact expected values for block elements may need adjusting once you see the mapping table in step 3. The key invariants are: box corners→`+`, horizontal lines→`-`, vertical lines→`|`, double horizontal→`=`, bullets→`*`, ANSI escapes pass through unchanged.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/util/ -run TestConvertToAscii -v`
Expected: FAIL — `ConvertToAscii` undefined.

- [ ] **Step 3: Implement ConvertToAscii**

Add to `internal/util/util.go`:

```go
// ConvertToAscii replaces UTF-8 box-drawing, block element, and other
// decorative Unicode characters with ASCII visual equivalents.
// ANSI escape sequences pass through unchanged.
// Fast-paths when input contains no bytes >= 0x80.
func ConvertToAscii(s string) string {
	// Fast path: if no high bytes, nothing to convert
	hasHighByte := false
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			hasHighByte = true
			break
		}
	}
	if !hasHighByte {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		if ascii, ok := unicodeToAscii[r]; ok {
			b.WriteByte(ascii)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// unicodeToAscii maps decorative Unicode runes to ASCII byte equivalents.
var unicodeToAscii = map[rune]byte{
	// Box-drawing: light
	'─': '-', '│': '|',
	'┌': '+', '┐': '+', '└': '+', '┘': '+',
	'├': '+', '┤': '+', '┬': '+', '┴': '+', '┼': '+',
	// Box-drawing: double
	'═': '=', '║': '|',
	'╔': '+', '╗': '+', '╚': '+', '╝': '+',
	'╠': '+', '╣': '+', '╦': '+', '╩': '+', '╬': '+',
	// Box-drawing: mixed single/double
	'╒': '+', '╕': '+', '╘': '+', '╛': '+',
	'╞': '+', '╡': '+', '╤': '+', '╧': '+', '╪': '+',
	'╓': '+', '╖': '+', '╙': '+', '╜': '+',
	'╟': '+', '╢': '+', '╥': '+', '╨': '+', '╫': '+',
	// Block elements
	'█': '#', '▓': '#', '▒': ':', '░': '.',
	'▄': '-', '▀': '_', '▌': '|', '▐': '|',
	// Bullet / misc
	'•': '*',
	// Diagonal lines
	'╲': '\\', '╱': '/',
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/util/ -run TestConvertToAscii -v`
Expected: PASS. Adjust any test expectations to match the mapping table if needed.

- [ ] **Step 5: Commit**

```bash
git add internal/util/util.go internal/util/util_test.go
git commit -m "feat: add ConvertToAscii utility for legacy client support"
```

---

### Task 2: Add AsciiMode to ClientSettings and ConnectionDetails.Write()

**Files:**
- Modify: `internal/connections/clientsettings.go` — add `AsciiMode` field
- Modify: `internal/connections/connectiondetails.go` — add conversion in `Write()`

The conversion hooks into `Write()` — the single output funnel for ALL data sent to any client. This catches messages, broadcasts, prompts, MOTD, login splash, and everything else without modifying individual handlers.

- [ ] **Step 1: Add AsciiMode to ClientSettings**

In `internal/connections/clientsettings.go`, add the field to `ClientSettings`:

```go
type ClientSettings struct {
	Display           DisplaySettings
	MSPEnabled        bool // Do they accept sound in their client?
	SendTelnetGoAhead bool // Defaults false, should we send a IAC GA after prompts?
	AsciiMode         bool // Convert UTF-8 decorative chars to ASCII equivalents?
}
```

- [ ] **Step 2: Add conversion in ConnectionDetails.Write()**

In `internal/connections/connectiondetails.go`, add the import for `util` and modify the `Write()` method. Find the existing `Write` method and add the ASCII conversion after the `stripAnsi` check but before the websocket/conn write. The conversion should skip telnet IAC commands (same guard as stripAnsi):

```go
import (
	// ... existing imports ...
	"github.com/GoMudEngine/GoMud/internal/util"
)
```

Inside `Write()`, after the existing `stripAnsi` block and before the websocket check:

```go
// Convert UTF-8 decorative chars to ASCII for legacy clients
if cd.clientSettings.AsciiMode && p[0] != term.TELNET_IAC {
	p = []byte(util.ConvertToAscii(string(p)))
	if len(p) == 0 {
		return 0, nil
	}
}
```

- [ ] **Step 3: Build to verify compilation**

Run: `go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/connections/clientsettings.go internal/connections/connectiondetails.go
git commit -m "feat: ASCII mode conversion in connection Write path"
```

---

### Task 3: Add AsciiMode to UserRecord and set command

**Files:**
- Modify: `internal/users/userrecord.go` — add `AsciiMode` field
- Modify: `internal/usercommands/set.go` — add `charset` case
- Modify: `_datafiles/world/dogmud/templates/help/set.template` — document the option

- [ ] **Step 1: Add AsciiMode field to UserRecord**

In `internal/users/userrecord.go`, add the field after `ScreenReader`:

```go
ScreenReader   bool                  `yaml:"screenreader,omitempty"` // Are they using a screen reader? (We should remove excess symbols)
AsciiMode      bool                  `yaml:"asciimode,omitempty"`    // Convert UTF-8 decorative chars to ASCII for legacy clients
```

- [ ] **Step 2: Add charset case to set command**

In `internal/usercommands/set.go`, add to the `switch setTarget` block (after the `screenreader` case):

```go
case `charset`:
	return cmdSetCharset(user)
```

Then add the handler function at the bottom of the file:

```go
func cmdSetCharset(user *users.UserRecord) (bool, error) {
	user.AsciiMode = !user.AsciiMode

	// Apply to active connection immediately
	cs := connections.GetClientSettings(user.ConnectionId())
	cs.AsciiMode = user.AsciiMode
	connections.OverwriteClientSettings(user.ConnectionId(), cs)

	if user.AsciiMode {
		user.SendText("Charset mode set to <ansi fg=\"red\">ASCII</ansi>.")
		user.SendText("Box-drawing characters will be converted to ASCII equivalents.")
		user.SendText("Use <ansi fg=\"command\">set charset</ansi> again to switch back to UTF-8.")
	} else {
		user.SendText("Charset mode set to <ansi fg=\"green\">UTF-8</ansi>.")
		user.SendText("Full Unicode box-drawing characters will be displayed.")
	}

	events.AddToQueue(events.UserSettingChanged{
		UserId: user.UserId,
		Name:   "charset",
	})

	return true, nil
}
```

- [ ] **Step 3: Update displaySetStatus to show charset**

In `internal/usercommands/set.go`, in the `displaySetStatus` function, add after the ScreenReader block:

```go
user.SendText(`<ansi fg="yellow-bold">charset:</ansi> `)
if user.AsciiMode {
	user.SendText(`<ansi fg="red">ASCII</ansi> (legacy client mode)`)
} else {
	user.SendText(`<ansi fg="green">UTF-8</ansi>`)
}
user.SendText(``)
```

- [ ] **Step 4: Update help template**

In `_datafiles/world/dogmud/templates/help/set.template`, add before the "See also" line:

```
  <ansi fg="command">set charset</ansi>
  Toggles between UTF-8 and ASCII display modes. If your client
  shows garbled characters in tables and borders, switch to ASCII.
```

- [ ] **Step 5: Build to verify compilation**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 6: Commit**

```bash
git add internal/users/userrecord.go internal/usercommands/set.go _datafiles/world/dogmud/templates/help/set.template
git commit -m "feat: add 'set charset' command for ASCII/UTF-8 toggle"
```

---

### Task 4: Apply user AsciiMode on login

**Files:**
- Modify: `internal/users/users.go` — apply AsciiMode to ClientSettings after login

When a user logs in (or reconnects from zombie), their `AsciiMode` preference needs to be copied to the connection's `ClientSettings` so that `Write()` knows to convert.

- [ ] **Step 1: Add AsciiMode propagation in LoginUser**

In `internal/users/users.go`, find the `LoginUser` function. There are two paths: normal login (around line 212) and zombie reconnect (around line 180). After `user.connectionId = connectionId` in **both** paths, add:

```go
// Apply persisted charset preference to connection
if user.AsciiMode {
	cs := connections.GetClientSettings(connectionId)
	cs.AsciiMode = true
	connections.OverwriteClientSettings(connectionId, cs)
}
```

For the zombie reconnect path (around line 180, after `user.connectionId = connectionId`), add the same block.

For the normal login path (around line 212, after `user.connectionId = connectionId`), add the same block.

- [ ] **Step 2: Build to verify compilation**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 3: Run full test suite**

Run: `go test ./... 2>&1 | tail -20`
Expected: All tests pass. No regressions.

- [ ] **Step 4: Commit**

```bash
git add internal/users/users.go
git commit -m "feat: restore ASCII mode preference on login/reconnect"
```

---

### Task 5: Integration test — set charset round-trip

**Files:**
- Modify: `internal/usercommands/usercommands_test.go` — add test for the set charset command

- [ ] **Step 1: Write integration test**

Add a test that verifies `set charset` toggles `user.AsciiMode` and that the connection's `ClientSettings.AsciiMode` is updated:

```go
func TestSetCharset(t *testing.T) {
	user, room := getTestUserAndRoom(t)

	// Default should be false (UTF-8)
	assert.False(t, user.AsciiMode)

	// Toggle to ASCII
	handled, err := Set("charset", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)
	assert.True(t, user.AsciiMode)

	// Verify connection setting was updated
	cs := connections.GetClientSettings(user.ConnectionId())
	assert.True(t, cs.AsciiMode)

	// Toggle back to UTF-8
	handled, err = Set("charset", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)
	assert.False(t, user.AsciiMode)

	cs = connections.GetClientSettings(user.ConnectionId())
	assert.False(t, cs.AsciiMode)
}
```

- [ ] **Step 2: Run the test**

Run: `go test ./internal/usercommands/ -run TestSetCharset -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/usercommands_test.go
git commit -m "test: add integration test for set charset command"
```

---

### Task 6: Update empty/default world templates

**Files:**
- Modify: `_datafiles/world/empty/templates/help/set.template` (if it has a set help file — may not exist or may mirror dogmud's)
- Modify: `_datafiles/world/default/templates/help/set.template` (same)

Check whether `_datafiles/world/empty/` and `_datafiles/world/default/` have their own `help/set.template`. If they do, add the same `set charset` help text. If they don't exist, skip this task.

- [ ] **Step 1: Check for template files**

```bash
ls _datafiles/world/empty/templates/help/set.template _datafiles/world/default/templates/help/set.template 2>/dev/null
```

- [ ] **Step 2: Update any that exist with the same charset help text**

Add the same block as Task 3 Step 4.

- [ ] **Step 3: Commit (if changes were made)**

```bash
git add _datafiles/world/empty/templates/help/set.template _datafiles/world/default/templates/help/set.template
git commit -m "docs: add charset help to empty/default world templates"
```

---

## Manual Testing Checklist

After all tasks are complete, verify on a running server:

1. `set` — shows charset status (UTF-8 by default)
2. `set charset` — toggles to ASCII, confirmation message appears
3. `status` — box-drawing chars show as `+`, `-`, `|` instead of `┌`, `─`, `│`
4. `motd` — MOTD border shows `+==+` instead of `╔══╗`
5. `who` — table borders are ASCII
6. `help set` — shows charset documentation
7. `set charset` again — toggles back to UTF-8, Unicode chars return
8. Log out and back in — ASCII mode persists
9. Reconnect (zombie resume) — ASCII mode persists
