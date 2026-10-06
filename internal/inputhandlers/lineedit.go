package inputhandlers

import (
	"fmt"
	"unicode/utf8"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/mattn/go-runewidth"
)

// runeDisplayWidth returns the number of terminal columns a rune occupies when
// rendered. East-Asian wide/fullwidth characters (e.g. CJK) occupy 2 columns;
// everything else occupies at least 1 so erasure never under-shoots.
func runeDisplayWidth(r rune) int {
	if w := runewidth.RuneWidth(r); w > 0 {
		return w
	}
	return 1
}

// bufferDisplayWidth returns the total terminal-column width of a UTF-8 byte
// slice. Invalid byte sequences are skipped (counted as 0 columns).
func bufferDisplayWidth(b []byte) int {
	w := 0
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		if r == utf8.RuneError && size == 1 {
			// Skip the invalid byte rather than treating it as a printable column.
			b = b[1:]
			continue
		}
		w += runeDisplayWidth(r)
		b = b[size:]
	}
	return w
}

// isMaskedPrompt reports whether the connection is currently on a masked
// (password-style) prompt step. When true, line editing stays linear and the
// server must not echo the real characters or move the cursor within the line.
func isMaskedPrompt(sharedState map[string]any) bool {
	if sharedState == nil {
		return false
	}
	v, ok := sharedState[promptHandlerStateKey]
	if !ok {
		return false
	}
	s, ok := v.(*PromptHandlerState)
	if !ok || s == nil {
		return false
	}
	if s.CurrentStepIndex < 0 || s.CurrentStepIndex >= len(s.Steps) {
		return false
	}
	return s.Steps[s.CurrentStepIndex].MaskInput
}

// isLocalEchoConn reports whether the client echoes input itself (Mudlet and the
// websocket/web client). For such clients the server must not emit per-keystroke
// echo or cursor-move sequences — they manage their own input line.
func isLocalEchoConn(connectionId connections.ConnectionId) bool {
	cs := connections.GetClientSettings(connectionId)
	return cs.IsMudlet || connections.IsWebsocket(connectionId)
}

// clampCursor keeps Cursor within [0, len(Buffer)]. It must be called by any
// handler that is about to read/mutate the buffer relative to the cursor, since
// other handlers (history recall, signal shortcuts) can resize the buffer.
func clampCursor(clientInput *connections.ClientInput) {
	if clientInput.Cursor < 0 {
		clientInput.Cursor = 0
	} else if clientInput.Cursor > len(clientInput.Buffer) {
		clientInput.Cursor = len(clientInput.Buffer)
	}
}

// cursorBackwardN returns the ANSI sequence to move the terminal cursor left n
// columns. Returns nil for n <= 0 so callers can pass it straight to SendTo.
func cursorBackwardN(n int) []byte {
	if n <= 0 {
		return nil
	}
	return []byte(term.AnsiMoveCursorBackward.StringWithPayload(fmt.Sprintf("%d", n)))
}

// cursorForwardN returns the ANSI sequence to move the terminal cursor right n
// columns. Returns nil for n <= 0.
func cursorForwardN(n int) []byte {
	if n <= 0 {
		return nil
	}
	return []byte(term.AnsiMoveCursorForward.StringWithPayload(fmt.Sprintf("%d", n)))
}

// matchesKey is a bool-only wrapper around term.Matches for use in compound
// conditions (term.Matches returns two values, which can't appear in ||).
func matchesKey(data []byte, cmd term.TerminalCommand) bool {
	ok, _ := term.Matches(data, cmd)
	return ok
}

// isHomeKey reports whether data is any of the common Home-key sequences.
func isHomeKey(data []byte) bool {
	return matchesKey(data, term.AnsiKeyHomeCSI) ||
		matchesKey(data, term.AnsiKeyHomeTilde) ||
		matchesKey(data, term.AnsiKeyHomeApp)
}

// isEndKey reports whether data is any of the common End-key sequences.
func isEndKey(data []byte) bool {
	return matchesKey(data, term.AnsiKeyEndCSI) ||
		matchesKey(data, term.AnsiKeyEndTilde) ||
		matchesKey(data, term.AnsiKeyEndApp)
}

// clearInputLineDisplay erases the current input from the terminal screen for
// server-side-echo clients and homes the logical cursor, in preparation for
// replacing the buffer contents (e.g. history recall). It does not touch the
// buffer itself; callers reset Buffer/Cursor/DataIn afterwards.
func clearInputLineDisplay(clientInput *connections.ClientInput) {
	if isLocalEchoConn(clientInput.ConnectionId) {
		return
	}
	// Move the terminal cursor back to the start of the input field (just after
	// the prompt), then erase to the end of the line. Input is contiguous from
	// the field start to its end, so this clears the whole field.
	connections.SendTo(cursorBackwardN(bufferDisplayWidth(clientInput.Buffer[:clientInput.Cursor])), clientInput.ConnectionId)
	connections.SendTo([]byte(term.AnsiEraseLineForward.String()), clientInput.ConnectionId)
}

// redrawEraseAtCursor is called after a rune has been removed from the buffer
// just before the cursor (backspace). The terminal cursor is still at the old
// (pre-removal) position; this moves it back onto the gap, redraws the tail so
// remaining characters slide left, clears the now-stale trailing cell, and
// leaves the terminal cursor aligned with the logical cursor.
func redrawEraseAtCursor(clientInput *connections.ClientInput, erasedWidth int) {
	connections.SendTo(cursorBackwardN(erasedWidth), clientInput.ConnectionId)

	tail := clientInput.Buffer[clientInput.Cursor:]
	connections.SendTo(tail, clientInput.ConnectionId)

	// The buffer shrank by one rune, so erase any leftover at the end of the line.
	connections.SendTo([]byte(term.AnsiEraseLineForward.String()), clientInput.ConnectionId)

	// Writing the tail advanced the cursor; move it back to the logical cursor.
	connections.SendTo(cursorBackwardN(bufferDisplayWidth(tail)), clientInput.ConnectionId)
}

// redrawInsertAtCursor is called after DataIn has been inserted into the buffer
// at oldCursor. It re-renders from oldCursor to the end (inserted text + old
// tail) and then moves the terminal cursor back to the logical cursor, which
// sits just after the inserted text. For an append-at-end edit the tail is empty
// so this reduces to a plain echo.
func redrawInsertAtCursor(clientInput *connections.ClientInput, oldCursor int) {
	connections.SendTo(clientInput.Buffer[oldCursor:], clientInput.ConnectionId)
	connections.SendTo(cursorBackwardN(bufferDisplayWidth(clientInput.Buffer[clientInput.Cursor:])), clientInput.ConnectionId)
}
