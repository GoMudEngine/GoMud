package inputhandlers

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/term"
)

// CleanserInputHandler's job is to remove any bad characters from the input stream
// before passing it down the chain.
// For this reason, it's important it happen before other text processing handlers
func CleanserInputHandler(clientInput *connections.ClientInput, sharedState map[string]any) (nextHandler bool) {

	if len(clientInput.DataIn) < 1 {
		return true
	}

	// backspace
	dIn := clientInput.DataIn[len(clientInput.DataIn)-1]

	if dIn == term.ASCII_DELETE || dIn == term.ASCII_BACKSPACE {

		clientInput.BSPressed = true

		//connections.SendTo([]byte(term.AnsiMoveCursorBackward.String()+" "+term.AnsiMoveCursorBackward.String()), connDetails.UniqueId())
		// send backspace, space, backspace
		if len(clientInput.Buffer) > 0 {
			connections.SendTo([]byte{term.ASCII_BACKSPACE, term.ASCII_SPACE, term.ASCII_BACKSPACE}, clientInput.ConnectionId)

			// Handle UTF-8 properly by removing the last complete character (rune)
			bufferStr := string(clientInput.Buffer)
			if len(bufferStr) > 0 {
				// Find the start of the last rune
				_, size := utf8.DecodeLastRune(clientInput.Buffer)
				if size > 0 {
					clientInput.Buffer = clientInput.Buffer[:len(clientInput.Buffer)-size]
				}
			}
		}
		clientInput.DataIn = clientInput.DataIn[:len(clientInput.DataIn)-1]
		return true
	}

	if dIn == term.ASCII_TAB {
		clientInput.TabPressed = true
	} else {
		// Check if the last byte is a CR or LF or NULL
		if dIn <= term.ASCII_CR {
			if clientInput.DataIn[len(clientInput.DataIn)-1] == term.ASCII_NULL || clientInput.DataIn[len(clientInput.DataIn)-1] == term.ASCII_LF || clientInput.DataIn[len(clientInput.DataIn)-1] == term.ASCII_CR {
				clientInput.EnterPressed = true
			}
		}
	}

	// Remove non printing chars
	//clientInput.DataIn = trimNonPrintingBytes(clientInput.DataIn)

	clientInput.DataIn = []byte(strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, string(clientInput.DataIn)))

	// Add all input to the currentBuffer
	clientInput.Buffer = append(clientInput.Buffer, clientInput.DataIn...)

	return true
}

// Trims non printing bytes and SPACE from front/back of a byte slice
func trimNonPrintingBytes(b []byte) []byte {
	start := 0
	for ; start < len(b); start++ {
		c := b[start]
		if c > 31 && c < 127 {
			break
		}
	}

	stop := len(b)
	for ; stop > start; stop-- {
		c := b[stop-1]
		if c > 31 && c < 127 {
			break
		}
	}

	return b[start:stop]
}
