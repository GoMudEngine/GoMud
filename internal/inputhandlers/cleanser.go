package inputhandlers

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/term"
)

// CleanserInputHandler's job is to remove any bad characters from the input stream
// before passing it down the chain, and to maintain the editing buffer + cursor.
// For this reason, it's important it happen before other text processing handlers.
func CleanserInputHandler(clientInput *connections.ClientInput, sharedState map[string]any) (nextHandler bool) {

	if len(clientInput.DataIn) < 1 {
		return true
	}

	// Other handlers (history recall, ctrl-shortcuts) can resize the buffer, so
	// keep the cursor valid before we touch anything relative to it.
	clampCursor(clientInput)

	serverEcho := !isLocalEchoConn(clientInput.ConnectionId)
	masked := isMaskedPrompt(sharedState)

	// Examine the final byte for control keys (backspace / tab / enter). A single
	// read can deliver several keystrokes at once; we only react to the last one
	// for control-key detection, matching the historical behavior.
	dIn := clientInput.DataIn[len(clientInput.DataIn)-1]

	// Backspace / Delete (the ASCII DEL key, value 127): remove the rune
	// immediately before the cursor and erase the columns it occupied.
	if dIn == term.ASCII_DELETE || dIn == term.ASCII_BACKSPACE {

		clientInput.BSPressed = true

		// Strip the control byte itself from the input we forward downstream.
		clientInput.DataIn = clientInput.DataIn[:len(clientInput.DataIn)-1]

		if clientInput.Cursor > 0 {
			r, size := utf8.DecodeLastRune(clientInput.Buffer[:clientInput.Cursor])
			if size > 0 {
				// Remove the whole rune (not a single byte) so multibyte UTF-8
				// characters such as CJK are deleted atomically.
				clientInput.Buffer = append(clientInput.Buffer[:clientInput.Cursor-size], clientInput.Buffer[clientInput.Cursor:]...)
				clientInput.Cursor -= size

				if serverEcho {
					w := runeDisplayWidth(r)
					if masked {
						// Masked fields render one column per rune, regardless of width.
						w = 1
					}
					redrawEraseAtCursor(clientInput, w)
				}
			}
		}

		return true
	}

	if dIn == term.ASCII_TAB {
		clientInput.TabPressed = true
	} else if dIn <= term.ASCII_CR {
		// Check if the last byte is a CR or LF or NULL -> treat as Enter.
		if last := clientInput.DataIn[len(clientInput.DataIn)-1]; last == term.ASCII_NULL || last == term.ASCII_LF || last == term.ASCII_CR {
			clientInput.EnterPressed = true
		}
	}

	// Strip non-printable bytes while preserving full UTF-8 runes (e.g. CJK).
	clientInput.DataIn = []byte(strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, string(clientInput.DataIn)))

	if len(clientInput.DataIn) == 0 {
		return true
	}

	// Insert the typed text at the cursor (rather than always appending), so the
	// user can edit anywhere in the line after moving the cursor with the arrows.
	clientInput.Buffer = slices.Insert(clientInput.Buffer, clientInput.Cursor, clientInput.DataIn...)
	oldCursor := clientInput.Cursor
	clientInput.Cursor += len(clientInput.DataIn)

	// For server-side-echo clients we render the edit ourselves so that mid-line
	// inserts and wide characters display correctly, then clear DataIn so the
	// downstream echo handlers (EchoInputHandler / login prompt) don't double up.
	// Masked steps leave DataIn intact so the login handler can emit the mask,
	// and local-echo clients manage their own display.
	if serverEcho && !masked {
		redrawInsertAtCursor(clientInput, oldCursor)
		clientInput.DataIn = clientInput.DataIn[:0]
	}

	return true
}
