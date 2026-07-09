package inputhandlers

import (
	"unicode/utf8"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/term"
)

func AnsiHandler(clientInput *connections.ClientInput, sharedState map[string]any) (nextHandler bool) {

	// Check for ANSI commands
	if !term.IsAnsiCommand(clientInput.DataIn) {
		return true
	}

	serverEcho := !isLocalEchoConn(clientInput.ConnectionId)
	masked := isMaskedPrompt(sharedState)
	// Cursor editing (arrows / home / end / delete) is only meaningful for
	// server-side-echo, non-masked input. Masked (password) fields stay linear.
	cursorEditAllowed := serverEcho && !masked

	// Multiple Ansi Commands's can be stacked into one send, so useful to split them out
	ansiCmds := [][]byte{}

	var lastAnsiEsc int = 0
	for i, b := range clientInput.DataIn {
		if i != 0 && b == term.ANSI_ESC {
			ansiCmds = append(ansiCmds, clientInput.DataIn[lastAnsiEsc:i])
			lastAnsiEsc = i
		}
	}
	if lastAnsiEsc < len(clientInput.DataIn) {
		ansiCmds = append(ansiCmds, clientInput.DataIn[lastAnsiEsc:])
	}

	for _, ansiCmds := range ansiCmds {
		// Check incoming ANSI commands for anything useful...

		// Is it a screen size report?
		if ok, payload := term.Matches(ansiCmds, term.AnsiClientScreenSize); ok {

			w, h, err := term.AnsiParseScreenSizePayload(payload)
			if err != nil {
				mudlog.Debug("Received", "type", "ANSI (Screensize)", "data", term.BytesString(payload), "error", err)
			} else {
				mudlog.Debug("Received", "type", "ANSI (Screensize)", "width", w, "height", h)

				if err != nil {

					cs := connections.GetClientSettings(clientInput.ConnectionId)
					cs.Display.ScreenWidth = uint32(w)
					cs.Display.ScreenHeight = uint32(h)
					connections.OverwriteClientSettings(clientInput.ConnectionId, cs)

				}
			}

			continue
		}

		// Is it a mouse click report?
		if ok, _ := term.Matches(ansiCmds, term.AnsiClientMouseDown); ok {
			// Ignore the down click, wait for the up before processing the click
			continue
		}

		if ok, payload := term.Matches(ansiCmds, term.AnsiClientMouseUp); ok {

			x, y, err := term.AnsiParseMouseClickPayload(payload)
			if err != nil {
				mudlog.Debug("Received", "type", "ANSI (MouseClick)", "data", term.BytesString(payload), "error", err)
			} else {
				mudlog.Debug("Received", "type", "ANSI (MouseClick)", "x", x, "y", y)
			}

			continue
		}

		if ok, payload := term.Matches(ansiCmds, term.AnsiMouseWheelUp); ok {

			x, y, err := term.AnsiParseMouseWheelScroll(payload)
			if err != nil {
				mudlog.Debug("Received", "type", "ANSI (MouseWheelUp)", "data", term.BytesString(payload), "error", err)
			} else {
				mudlog.Debug("Received", "type", "ANSI (MouseWheelUp)", "x", x, "y", y)
			}

			continue
		}

		if ok, payload := term.Matches(ansiCmds, term.AnsiMouseWheelDown); ok {
			x, y, err := term.AnsiParseMouseWheelScroll(payload)
			if err != nil {
				mudlog.Debug("Received", "type", "ANSI (MouseWheelDown)", "data", term.BytesString(payload), "error", err)
			} else {
				mudlog.Debug("Received", "type", "ANSI (MouseWheelDown)", "x", x, "y", y)
			}
			continue
		}

		if ok, _ := term.Matches(ansiCmds, term.AnsiMoveCursorUp); ok {
			mudlog.Debug("Received", "type", "ANSI (MoveCursorUp)", "currentInput", string(clientInput.Buffer), "LastSubmitted", string(clientInput.LastSubmitted))

			// Replace the current input line with the previous history entry.
			// Clear by display width (not byte count) so wide characters erase
			// correctly, then feed the recalled entry back through CleanserInputHandler
			// so it is re-inserted and echoed.
			clientInput.DataIn = []byte{}

			clearInputLineDisplay(clientInput)

			clientInput.History.Previous()
			historicInput := clientInput.History.Get()
			clientInput.DataIn = make([]byte, len(historicInput))
			copy(clientInput.DataIn, historicInput)

			clientInput.Buffer = []byte{}
			clientInput.Cursor = 0
			clientInput.EnterPressed = false
			nextHandler = true
			continue
		}

		if ok, _ := term.Matches(ansiCmds, term.AnsiMoveCursorDown); ok {
			mudlog.Debug("Received", "type", "ANSI (MoveCursorDown)", "currentInput", string(clientInput.Buffer), "LastSubmitted", string(clientInput.LastSubmitted))

			// Replace the current input line with the next history entry.
			clientInput.DataIn = []byte{}

			clearInputLineDisplay(clientInput)

			clientInput.History.Next()
			historicInput := clientInput.History.Get()
			clientInput.DataIn = make([]byte, len(historicInput))
			copy(clientInput.DataIn, historicInput)

			clientInput.Buffer = []byte{}
			clientInput.Cursor = 0
			clientInput.EnterPressed = false
			nextHandler = true
			continue
		}

		// --- Cursor-editing keys (Left / Right / Home / End / Delete) ---
		// These move or edit at the logical cursor within the current input line.
		// They are no-ops for masked prompts and for local-echo clients.
		if cursorEditAllowed {

			// Left arrow (ESC [ D, or ESC O D in application cursor mode)
			if matchesKey(ansiCmds, term.AnsiMoveCursorBackward) || matchesKey(ansiCmds, term.AnsiKeyLeftApp) {
				clientInput.DataIn = []byte{}
				if clientInput.Cursor > 0 {
					if r, size := utf8.DecodeLastRune(clientInput.Buffer[:clientInput.Cursor]); size > 0 {
						clientInput.Cursor -= size
						connections.SendTo(cursorBackwardN(runeDisplayWidth(r)), clientInput.ConnectionId)
					}
				}
				nextHandler = true
				continue
			}

			// Right arrow (ESC [ C, or ESC O C)
			if matchesKey(ansiCmds, term.AnsiMoveCursorForward) || matchesKey(ansiCmds, term.AnsiKeyRightApp) {
				clientInput.DataIn = []byte{}
				if clientInput.Cursor < len(clientInput.Buffer) {
					if r, size := utf8.DecodeRune(clientInput.Buffer[clientInput.Cursor:]); size > 0 {
						clientInput.Cursor += size
						connections.SendTo(cursorForwardN(runeDisplayWidth(r)), clientInput.ConnectionId)
					}
				}
				nextHandler = true
				continue
			}

			// Home: move cursor to the start of the input.
			if isHomeKey(ansiCmds) {
				clientInput.DataIn = []byte{}
				connections.SendTo(cursorBackwardN(bufferDisplayWidth(clientInput.Buffer[:clientInput.Cursor])), clientInput.ConnectionId)
				clientInput.Cursor = 0
				nextHandler = true
				continue
			}

			// End: move cursor to the end of the input.
			if isEndKey(ansiCmds) {
				clientInput.DataIn = []byte{}
				connections.SendTo(cursorForwardN(bufferDisplayWidth(clientInput.Buffer[clientInput.Cursor:])), clientInput.ConnectionId)
				clientInput.Cursor = len(clientInput.Buffer)
				nextHandler = true
				continue
			}

			// Delete (forward): remove the rune at the cursor, redraw the tail.
			if ok, _ := term.Matches(ansiCmds, term.AnsiKeyDelete); ok {
				clientInput.DataIn = []byte{}
				if clientInput.Cursor < len(clientInput.Buffer) {
					if _, size := utf8.DecodeRune(clientInput.Buffer[clientInput.Cursor:]); size > 0 {
						clientInput.Buffer = append(clientInput.Buffer[:clientInput.Cursor], clientInput.Buffer[clientInput.Cursor+size:]...)
						// Cursor position is unchanged; redraw from it to the end.
						tail := clientInput.Buffer[clientInput.Cursor:]
						connections.SendTo(tail, clientInput.ConnectionId)
						connections.SendTo([]byte(term.AnsiEraseLineForward.String()), clientInput.ConnectionId)
						connections.SendTo(cursorBackwardN(bufferDisplayWidth(tail)), clientInput.ConnectionId)
					}
				}
				nextHandler = true
				continue
			}
		}

		isF1, _ := term.Matches(ansiCmds, term.AnsiF1)
		if !isF1 { // check for Alternate F1
			isF1, _ = term.Matches(ansiCmds, term.AnsiF1b)
		}
		if isF1 {
			clientInput.DataIn = []byte("=1")
			clientInput.Buffer = []byte{}
			clientInput.EnterPressed = true
			// Since we are transforming this, pass it on
			nextHandler = true
			continue
		}

		// Unhanlded ANSI command, log it
		mudlog.Debug("Received", "type", "ANSI", "size", len(ansiCmds), "data", term.AnsiCommandToString(ansiCmds))
	}

	return nextHandler
}
