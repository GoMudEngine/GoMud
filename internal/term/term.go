package term

import "fmt"

type TerminalCommandPayloadParser func(b []byte) []byte

const ASCII_NULL = 0
const ASCII_BACKSPACE = 8
const ASCII_SPACE = 32
const ASCII_DELETE = 127
const ASCII_TAB = 9
const ASCII_CR = 13
const ASCII_LF = 10

var (
	CRLF    = []byte{13, 10}
	CRLFStr = string(CRLF)

	BELL    = []byte{13, 7} // may beep, may flash the window, may bounce from the bar on macos
	BELLStr = string(BELL)

	// alternative mode (No scrollback)
	// Hide Cursor
	// UTF8 mode
	// Request resolution
	/*
		c.Conn.Write([]byte(util.ANSI_ALTMODE_START + util.ANSI_CUSOR_HIDE + util.ANSI_UTF8 + util.ANSI_REQ_RESOLUTION + util.ANSI_REPORT_MOUSE_CLICK))
	*/

	///////////////////////////
	// Useful sequences
	///////////////////////////
	// Move cursor back, print a space, move cursor back again.
	BACKSPACE_SEQUENCE = []byte{ASCII_BACKSPACE, ASCII_SPACE, ASCII_BACKSPACE}

	///////////////////////////
	// TELNET COMMANDS
	///////////////////////////
	//
	// SCREEN RESOLUTION
	//

	// // Request resolution from the client
	TelnetScreenSizeRequest = TerminalCommand{[]byte{TELNET_IAC, TELNET_DO, TELNET_OPT_NAWS}, []byte{}}
	// // Client response with their resolution
	TelnetScreenSizeResponse = TerminalCommand{[]byte{TELNET_IAC, TELNET_SB, TELNET_OPT_NAWS}, []byte{}}

	//
	// GENERAL SETUP
	//
	// // Do/Don't Suppress Go Ahead
	TelnetSuppressGoAhead     = TerminalCommand{[]byte{TELNET_IAC, TELNET_WILL, TELNET_OPT_SUP_GO_AHD}, []byte{}}
	TelnetDontSuppressGoAhead = TerminalCommand{[]byte{TELNET_IAC, TELNET_DONT, TELNET_OPT_SUP_GO_AHD}, []byte{}}
	// // Echo On
	TelnetEchoOn = TerminalCommand{[]byte{TELNET_IAC, TELNET_WILL, TELNET_OPT_ECHO}, []byte{}}
	// // Echo Off
	TelnetEchoOff = TerminalCommand{[]byte{TELNET_IAC, TELNET_WONT, TELNET_OPT_ECHO}, []byte{}}
	// // Line Mode Off
	TelnetLineModeOff = TerminalCommand{[]byte{TELNET_IAC, TELNET_WONT, TELNET_OPT_LINE_MODE}, []byte{}}

	//
	// Handshake example:
	// Server (TelnetRequestChangeCharset)	-> Client
	// Server 								<- (TelnetAgreeChangeCharset) Client
	// Server (TelnetCharset)				-> Client
	// Server								<- (TelnetAcceptedChangeCharset) Client
	//
	// Indicate wish to change charset
	TelnetRequestChangeCharset = TerminalCommand{[]byte{TELNET_IAC, TELNET_WILL, TELNET_OPT_CHARSET}, []byte{}}
	// Client agreed to accept a change
	TelnetAgreeChangeCharset = TerminalCommand{[]byte{TELNET_IAC, TELNET_DO, TELNET_OPT_CHARSET}, []byte{}}
	// Send actual charset change
	// Can separate with a space multiple charsets:
	// " UTF-8 ISO-8859-1"
	TelnetCharset = TerminalCommand{[]byte{TELNET_IAC, TELNET_SB, TELNET_OPT_CHARSET, 1}, []byte{TELNET_IAC, TELNET_SE}}
	// Client accepted change
	TelnetAcceptedChangeCharset = TerminalCommand{[]byte{TELNET_IAC, TELNET_SB, 2}, []byte{TELNET_IAC, TELNET_SE}}
	// Client rejectected change
	TelnetRejectedChangeCharset = TerminalCommand{[]byte{TELNET_IAC, TELNET_SB, 3, TELNET_IAC, TELNET_SE}, []byte{}}
	// Go Ahead
	TelnetGoAhead = TerminalCommand{[]byte{TELNET_IAC, TELNET_GA}, []byte{}}

	//
	// NEW-ENVIRON / MNES (used to detect the client at connect time)
	//
	// Ask the client to enable NEW-ENVIRON. A compliant client replies with
	// TelnetWillNewEnviron (or TelnetWontNewEnviron to refuse).
	TelnetRequestNewEnviron = TerminalCommand{[]byte{TELNET_IAC, TELNET_DO, TELNET_OPT_NEW_ENV}, []byte{}}
	// Client agrees / refuses to use NEW-ENVIRON.
	TelnetWillNewEnviron = TerminalCommand{[]byte{TELNET_IAC, TELNET_WILL, TELNET_OPT_NEW_ENV}, []byte{}}
	TelnetWontNewEnviron = TerminalCommand{[]byte{TELNET_IAC, TELNET_WONT, TELNET_OPT_NEW_ENV}, []byte{}}
	// Server -> Client request for variables. Sent with an empty payload to
	// request ALL of the client's environment variables (broadly compatible;
	// Mudlet replies with CLIENT_NAME among them).
	TelnetNewEnvironSendRequest = TerminalCommand{[]byte{TELNET_IAC, TELNET_SB, TELNET_OPT_NEW_ENV, TELNET_NEWENV_SEND}, []byte{TELNET_IAC, TELNET_SE}}
	// Client -> Server response carrying the variable values. EndChars are left
	// empty so the match succeeds on the `IAC SB NEW-ENVIRON IS` prefix alone -
	// the trailing IAC SE is tolerated inside the payload and ignored by the
	// parser. This is more robust to TCP segmentation and trailing bytes than
	// requiring an exact terminator.
	TelnetNewEnvironResponse = TerminalCommand{[]byte{TELNET_IAC, TELNET_SB, TELNET_OPT_NEW_ENV, TELNET_NEWENV_IS}, []byte{}}

	///////////////////////////
	// ANSI COMMANDS
	///////////////////////////

	// Did they hit the escape key?
	AnsiEscapeKey = TerminalCommand{[]byte{ANSI_ESC}, []byte{}}

	// 4-bit color - Can contain fg, bg, bold, etc.
	AnsiColor4Bit = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'m'}}
	// 8 Bit color
	AnsiColor8BitFG = TerminalCommand{[]byte{ANSI_ESC, '[', '3', '8', ';', '5', ';'}, []byte{'m'}}
	AnsiColor8BitBG = TerminalCommand{[]byte{ANSI_ESC, '[', '4', '8', ';', '5', ';'}, []byte{'m'}}
	// 24 Bit color - RGB
	AnsiColor24BitFG = TerminalCommand{[]byte{ANSI_ESC, '[', '3', '8', ';', '2', ';'}, []byte{'m'}}
	AnsiColor24BitBG = TerminalCommand{[]byte{ANSI_ESC, '[', '4', '8', ';', '2', ';'}, []byte{'m'}}
	// Reset colors back to client default
	AnsiColorReset = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'0', 'm'}}
	// Enable alternative screen buffer
	AnsiAltModeStart = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'?', '1', '0', '4', '9', 'h'}}
	// Disable alternative screen buffer
	AnsiAltModeEnd = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'?', '1', '0', '4', '9', 'l'}}
	// Hide Cursor
	AnsiCursorHide = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'?', '2', '5', 'l'}} // DECTCEM
	// Show Cursor
	AnsiCursorShow = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'?', '2', '5', 'h'}} // DECTCEM
	// Clear from cursor to end of screen
	AnsiClearForward = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'0', 'J'}}
	// Clear from cursor to beginning of screen
	AnsiClearBack = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'1', 'J'}}
	// Clear Scrollback
	AnsiClearScreen = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'2', 'J'}}
	// Clear Screen and scrollback Buffer
	AnsiScreenAndScrollbackClear = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'3', 'J'}}
	// Clear from the cursor to the end of the line
	AnsiEraseLineForward = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'0', 'K'}}
	// Clear from the cursor to the start of the line
	AnsiEraseLineBackward = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'1', 'K'}}
	// Clear enter line
	AnsiEraseLine = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'2', 'K'}}
	// Hide Text
	AnsiTextHide = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'8', 'm'}}
	// Show Text
	AnsiTextShow = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'2', '8', 'm'}}
	// Request client report mouse clicks

	AnsiReportMouseClick      = TerminalCommand{[]byte{ANSI_ESC, '[', '?', '1', '0', '0', '2', 'h'}, []byte{ANSI_ESC, '[', '?', '1', '0', '0', '6', 'h'}}
	AnsiSaveCursorPosition    = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'s'}}
	AnsiRestoreCursorPosition = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'u'}}
	AnsiRequestCursorPosition = TerminalCommand{[]byte{ANSI_ESC, '[', '6'}, []byte{'n'}}
	AnsiCharSetUTF8           = TerminalCommand{[]byte{ANSI_ESC, '%'}, []byte{'G'}}
	// Client is reporting a mouse click
	// ESC [  <  0  ;  1  ;  1  m
	// [27 91 60 48 59 49 59 49 109]
	// ESC [  <  0  ;  1  4  0  ;  4  8  m
	// [27 91 60 48 59 49 52 48 59 52 56 109]
	AnsiClientMouseDown = TerminalCommand{[]byte{ANSI_ESC, '[', '<', '0', ';'}, []byte{'M'}}
	AnsiClientMouseUp   = TerminalCommand{[]byte{ANSI_ESC, '[', '<', '0', ';'}, []byte{'m'}}
	// To request the client terminal dimensions (hacky way):
	// 1. Save the cursor position
	// 2. Move the cursor to the bottom right corner
	// 3. Request the cursor position
	// 4. Restore the cursor position
	AnsiRequestResolution = TerminalCommand{[]byte(AnsiSaveCursorPosition.String() + AnsiMoveCursorBottomRight.String() + AnsiRequestCursorPosition.String() + AnsiRestoreCursorPosition.String()), []byte{}}
	// Client is reporting screen size
	// Looks like: ESC [  40 ; 80  R
	AnsiClientScreenSize = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'R'}}
	// Move the cursor - any number after '[' is how many spaces to move
	AnsiMoveCursor         = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'H'}}
	AnsiMoveCursorUp       = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'A'}}
	AnsiMoveCursorDown     = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'B'}}
	AnsiMoveCursorForward  = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'C'}}
	AnsiMoveCursorBackward = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'D'}}
	// Move the cursor to a specific column - any number after '[' is column. Default 1.
	AnsiMoveCursorColumn = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'G'}}

	AnsiMoveCursorBottomRight = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'9', '9', '9', ';', '9', '9', '9', 'H'}}
	AnsiMoveCursorTopLeft     = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'1', ';', '1', 'H'}}
	// \033[<65;9(xPos);19(yPos)M
	AnsiMouseWheelUp   = TerminalCommand{[]byte{ANSI_ESC, '[', '<', '6', '5', ';'}, []byte{'M'}}
	AnsiMouseWheelDown = TerminalCommand{[]byte{ANSI_ESC, '[', '<', '6', '4', ';'}, []byte{'M'}}
	// F keys

	AnsiF1  = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'1', '1', '~'}} // Putty sends this
	AnsiF2  = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'1', '2', '~'}} // Putty sends this
	AnsiF3  = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'1', '3', '~'}} // Putty sends this
	AnsiF4  = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'1', '4', '~'}} // Putty sends this
	AnsiF5  = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'1', '5', '~'}} // Putty sends this
	AnsiF6  = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'1', '7', '~'}} // Putty sends this
	AnsiF7  = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'1', '8', '~'}} // Putty sends this
	AnsiF8  = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'1', '9', '~'}} // Putty sends this
	AnsiF9  = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'2', '0', '~'}} // Putty sends this
	AnsiF10 = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'2', '1', '~'}} // Putty sends this
	AnsiF11 = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'2', '3', '~'}} // Putty sends this
	AnsiF12 = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'2', '4', '~'}} // Putty sends this
	AnsiF1b = TerminalCommand{[]byte{ANSI_ESC, 'O'}, []byte{'P'}}           // macos terminal telnet
	AnsiF2b = TerminalCommand{[]byte{ANSI_ESC, 'O'}, []byte{'Q'}}           // macos terminal telnet
	AnsiF3b = TerminalCommand{[]byte{ANSI_ESC, 'O'}, []byte{'R'}}           // macos terminal telnet
	AnsiF4b = TerminalCommand{[]byte{ANSI_ESC, 'O'}, []byte{'S'}}           // macos terminal telnet

	///////////////////////////
	// EDITING KEYS
	///////////////////////////
	// These arrive from the client when the user presses cursor-editing keys.
	// Left/Right reuse the cursor-motion sequences (ESC [ C / ESC [ D).
	AnsiKeyLeftApp   = TerminalCommand{[]byte{ANSI_ESC, 'O'}, []byte{'D'}}      // ESC O D (application cursor mode)
	AnsiKeyRightApp  = TerminalCommand{[]byte{ANSI_ESC, 'O'}, []byte{'C'}}      // ESC O C (application cursor mode)
	AnsiKeyHomeCSI   = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'H'}}      // ESC [ H (same bytes as cursor-home)
	AnsiKeyEndCSI    = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'F'}}      // ESC [ F
	AnsiKeyHomeTilde = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'1', '~'}} // ESC [ 1 ~
	AnsiKeyEndTilde  = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'4', '~'}} // ESC [ 4 ~
	AnsiKeyDelete    = TerminalCommand{[]byte{ANSI_ESC, '['}, []byte{'3', '~'}} // ESC [ 3 ~
	AnsiKeyHomeApp   = TerminalCommand{[]byte{ANSI_ESC, 'O'}, []byte{'H'}}      // ESC O H (xterm app mode)
	AnsiKeyEndApp    = TerminalCommand{[]byte{ANSI_ESC, 'O'}, []byte{'F'}}      // ESC O F (xterm app mode)

	// Payload is the window title to set it to
	AnsiSetWindowTitle = TerminalCommand{[]byte{ANSI_ESC, ']', '2', ';'}, []byte{'S', 'T'}}

	// Payload is frequency in Hz
	AnsiSetBellFrequency = TerminalCommand{[]byte{ANSI_ESC, '[', '1', '0', ';'}, []byte{']'}}
	// Payload is bell duration in msec
	AnsiSetBellDuration = TerminalCommand{[]byte{ANSI_ESC, '[', '1', '1', ';'}, []byte{']'}}
)

func IsTelnetCommand(b []byte) bool {
	return len(b) > 0 && b[0] == TELNET_IAC
}

func IsAnsiCommand(b []byte) bool {
	return len(b) > 0 && b[0] == ANSI_ESC
}

func ExtractPayload(b []byte, template TerminalCommand) []byte {
	return []byte{}
}

type TerminalCommand struct {
	Chars    []byte
	EndChars []byte
}

func (cmd *TerminalCommand) BytesWithPayload(payload []byte) []byte {
	result := []byte{}
	result = append(result, cmd.Chars...)
	if len(payload) > 0 {
		result = append(result, payload...)
	}
	result = append(result, cmd.EndChars...)
	return result
}

func (cmd *TerminalCommand) ExtractBody(input []byte) []byte {
	if len(cmd.Chars) == 0 {
		return input
	}
	return input[len(cmd.Chars) : len(input)-len(cmd.EndChars)]
}

func Matches(input []byte, cmd TerminalCommand) (ok bool, payload []byte) {
	// The length of the testBytes should at least be the same as the chars plus endchars
	if len(input) < len(cmd.Chars)+len(cmd.EndChars) {
		return false, nil
	}
	// Check the start chars
	for i, b := range cmd.Chars {
		if b != input[i] {
			return false, nil
		}
	}

	if len(cmd.EndChars) == 0 {
		if len(input) == len(cmd.Chars) {
			return true, nil
		}
		// Return any remaining ("payload") bytes
		return true, input[len(cmd.Chars):]
	}

	// Check the end chars
	for i, b := range cmd.EndChars {
		if b != input[len(input)-len(cmd.EndChars)+i] {
			return false, nil
		}
	}

	if len(input) == len(cmd.Chars)+len(cmd.EndChars) {
		return true, nil
	}
	// Return any "payload" bytes
	return true, input[len(cmd.Chars) : len(input)-len(cmd.EndChars)]
}

func (c *TerminalCommand) String() string {
	return string(c.Chars) + string(c.EndChars)
}

func (c *TerminalCommand) StringWithPayload(payload string) string {
	return string(c.Chars) + payload + string(c.EndChars)
}

func (c *TerminalCommand) DebugString() string {
	if c.Chars[0] == TELNET_IAC {
		return TelnetCommandToString(c.Chars) + TelnetCommandToString(c.EndChars)
	} else if c.Chars[0] == ANSI_ESC {
		return AnsiCommandToString(c.Chars) + AnsiCommandToString(c.EndChars)
	}
	return "???"
}

func TelnetCommandToString(cmdBytes []byte) string {
	var retStr string = ""
	for _, b := range cmdBytes {
		retStr += fmt.Sprintf("[%v %s]", b, iacByteString(b))
	}
	return retStr
}

func AnsiCommandToString(cmdBytes []byte) string {
	var retStr string = ""
	for _, b := range cmdBytes {
		retStr += fmt.Sprintf("[%v %s]", b, string(b))
	}
	return retStr
}

func BytesString(cmdBytes []byte) string {
	var retStr string = "[]byte{ "
	for _, b := range cmdBytes {
		retStr += fmt.Sprintf("%v, ", b)
	}
	return retStr + " }"
}
