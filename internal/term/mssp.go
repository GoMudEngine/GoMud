package term

// MUD Server Status Protocol — telnet option 70.
// https://mudhalla.net/tintin/protocols/mssp/
//
// Handshake: server sends IAC WILL MSSP; a crawler replies IAC DO MSSP; the
// server then sends the data block once:
//   IAC SB MSSP (MSSP_VAR name MSSP_VAL value ...) IAC SE
//
// Mirrors msp.go — a sibling telnet option.

const (
	MSSP IACByte = 70

	MSSP_VAR byte = 1 // Precedes a variable (field) name.
	MSSP_VAL byte = 2 // Precedes a value; a field may have multiple values.
)

var (
	MsspEnable = TerminalCommand{[]byte{TELNET_IAC, TELNET_WILL, MSSP}, []byte{}} // Server offers MSSP.
	MsspAccept = TerminalCommand{[]byte{TELNET_IAC, TELNET_DO, MSSP}, []byte{}}   // Client accepts (asks for the data).
	MsspRefuse = TerminalCommand{[]byte{TELNET_IAC, TELNET_DONT, MSSP}, []byte{}} // Client refuses.

	MsspCommand = TerminalCommand{[]byte{TELNET_IAC, TELNET_SB, MSSP}, []byte{TELNET_IAC, TELNET_SE}} // Wraps the data block.
)

// MSSPField is one MSSP variable and its one-or-more values.
type MSSPField struct {
	Name   string
	Values []string
}

func IsMSSPCommand(b []byte) bool {
	return len(b) > 2 && b[0] == TELNET_IAC && b[2] == MSSP
}

// EncodeMSSPPayload builds the body that goes between IAC SB MSSP and IAC SE.
// Any IAC (0xFF) byte in a name or value is doubled so it can't terminate the
// sub-negotiation early.
func EncodeMSSPPayload(fields []MSSPField) []byte {
	out := []byte{}
	for _, f := range fields {
		out = append(out, MSSP_VAR)
		out = appendEscapedIAC(out, f.Name)
		for _, v := range f.Values {
			out = append(out, MSSP_VAL)
			out = appendEscapedIAC(out, v)
		}
	}
	return out
}

func appendEscapedIAC(dst []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		if s[i] == TELNET_IAC {
			dst = append(dst, TELNET_IAC, TELNET_IAC)
		} else {
			dst = append(dst, s[i])
		}
	}
	return dst
}
