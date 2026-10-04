package term

//
// Telnet Protocol Helper
//

import (
	"errors"
)

/****************************************
*
* Telnet Command bytes
*	Either end of a telnet dialogue can enable or disable an option either locally or remotely.
*	The initiator sends a 3 byte command of the form:
*	TELNET_IAC,<type of operation>,<option>
*
*	The response is of the same form.
*	TELNET_WILL	Sender wants to do something.
*	TELNET_WONT	Sender doesn't want to do something.
*	TELNET_DO		Sender wants the other end to do something.
*	TELNET_DONT	Sender wants the other not to do something
*
*	Associated with each of the these there are various possible responses:
*	Sender Sent	|	Receiver Responds	|	Implication
*	TELNET_WILL		TELNET_DO				The sender would like to use a certain facility if the receiver can handle it. Option is now in effect
*	TELNET_WILL		TELNET_DONT				Receiver says it cannot support the option. Option is not in effect.
*	TELNET_DO		TELNET_WILL				The sender says it can handle traffic from the sender if the sender wishes to use a certain option. Option is now in effect.
*	TELNET_DO		TELNET_WONT				Receiver says it cannot support the option. Option is not in effect.
*	TELNET_WONT		TELNET_DONT				Option disabled. DONT is only valid response.
*	TELNET_DONT		TELNET_WONT				Option disabled. WONT is only valid response.
*
*	For example if the sender wants the other end to suppress go-ahead it would send the byte sequence
*	255(TELNET_IAC),251(TELNET_WILL),3
*	The final byte of the three byte sequence identifies the required action.
*
*	For some of the negotiable options values need to be communicated once support of the option has been agreed. This is done using sub-option negotiation. Values are communicated via an exchange of value query commands and responses in the following form.
*
*	TELNET_IAC,TELNET_SB,<option code number>,1,TELNET_IAC,TELNET_SE
*	and
*
*	TELNET_IAC,TELNET_SB,<option code>,0,<value>,TELNET_IAC,TELNET_SE
*	For example if the client wishes to identify the terminal type to the server the following exchange might take place
*
*	Client   255(TELNET_IAC),251(TELNET_WILL),24
*	Server   255(TELNET_IAC),253(TELNET_DO),24
*	Server   255(TELNET_IAC),250(TELNET_SB),24,1,255(TELNET_IAC),240(TELNET_SE)
*	Client   255(TELNET_IAC),250(TELNET_SB),24,0,'V','T','2','2','0',255(TELNET_IAC),240(TELNET_SE)
*	The first exchange establishes that terminal type (option number 24) will be handled, the server then enquires of the client what value it wishes to associate with the terminal type. The sequence SB,24,1 implies sub-option negotiation for option type 24, value required (1). The TELNET_IAC,SE sequence indicates the end of this request. The repsonse TELNET_IAC,SB,24,0,'V'... implies sub-option negotiation for option type 24, value supplied (0), the TELNET_IAC,SE sequence indicates the end of the response (and the supplied value).
*	The encoding of the value is specific to the option but a sequence of characters, as shown above, is common.
*
*	Lots more info at:
*	http://pcmicro.com/netfoss/telnet.html
*
****************************************/

type IACByte = byte

const (
	TELNET_IAC  IACByte = 255 // Interpret as command
	TELNET_DONT IACByte = 254 // Indicates the demand that the other party stop performing, or confirmation that you are no longer expecting the other party to perform, the indicated option.
	TELNET_DO   IACByte = 253 // Indicates the request that the other party perform, or confirmation that you are expecting the other party to perform, the indicated option.
	TELNET_WONT IACByte = 252 // Indicates the refusal to perform, or continue performing, the indicated option.
	TELNET_WILL IACByte = 251 // Indicates the desire to begin performing, or confirmation that you are now performing, the indicated option.
	TELNET_SB   IACByte = 250 // Subnegotiation of the indicated option follows.
	TELNET_GA   IACByte = 249 // Go ahead. Used, under certain circumstances, to tell the other end that it can transmit.
	TELNET_EL   IACByte = 248 // Erase line. Delete characters from the data stream back to but not including the previous CRLF.
	TELNET_EC   IACByte = 247 // Erase character. The receiver should delete the last preceding undeleted character from the data stream.
	TELNET_AYT  IACByte = 246 // Are you there. Send back to the NVT some visible evidence that the AYT was received.
	TELNET_AO   IACByte = 245 // Abort output. Allows the current process to run to completion but do not send its output to the user.
	TELNET_IP   IACByte = 244 // Suspend, interrupt or abort the process to which the NVT is connected.
	TELNET_BRK  IACByte = 243 // Break. Indicates that the "break" or "attention" key was hit.
	TELNET_DM   IACByte = 242 // Data mark. Indicates the position of a Synch event within the data stream. This should always be accompanied by a TCP urgent notification.
	TELNET_NOP  IACByte = 241 // No operation
	TELNET_SE   IACByte = 240 // End of subnegotiation parameters

	// Common...
	TELNET_OPT_TXBIN      IACByte = 0  // Transmit Binary													RFC: http://pcmicro.com/netfoss/RFC856.html
	TELNET_OPT_ECHO       IACByte = 1  // Echo																RFC: http://pcmicro.com/netfoss/RFC857.html
	TELNET_OPT_SUP_GO_AHD IACByte = 3  // Suppress Go Ahead													RFC: http://pcmicro.com/netfoss/RFC858.html
	TELNET_OPT_STAT       IACByte = 5  // Status															RFC: http://pcmicro.com/netfoss/RFC859.html
	TELNET_OPT_TMARK      IACByte = 6  // Timing Mark														RFC: http://pcmicro.com/netfoss/RFC860.html
	TELNET_OPT_TERM_TYPE  IACByte = 24 // Terminal Type														RFC: https://www.ietf.org/rfc/rfc1091.txt
	TELNET_OPT_NAWS       IACByte = 31 // NAWS, Negotiate About Window Size.								RFC: https://www.ietf.org/rfc/rfc1073.txt
	TELNET_OPT_TERM_SPD   IACByte = 32 // Terminal Speed													RFC: https://www.ietf.org/rfc/rfc1079.txt
	TELNET_OPT_RMT_FC     IACByte = 33 // Remote Flow Control												RFC: https://www.ietf.org/rfc/rfc1372.txt
	TELNET_OPT_LINE_MODE  IACByte = 34 // Linemode															RFC: https://www.ietf.org/rfc/rfc1184.txt
	TELNET_OPT_ENV        IACByte = 36 // Environment														RFC: https://www.ietf.org/rfc/rfc1408.txt
	// Uncommon...
	TELNET_OPT_RECONN            IACByte = 2  // Reconnection												RFC:
	TELNET_OPT_APRXMSGSZ         IACByte = 4  // Approx Message Size Negotiation.							RFC:
	TELNET_OPT_RMT_TRANSECHO     IACByte = 7  // Remote Controlled Trans and Echo							RFC: https://www.ietf.org/rfc/rfc563.txt : https://www.ietf.org/rfc/rfc726.txt
	TELNET_OPT_OUTPUT_LW         IACByte = 8  // Output Line Width											RFC:
	TELNET_OPT_OUTPUT_PGSZ       IACByte = 9  // Output Page Size											RFC:
	TELNET_OPT_NEG_CR            IACByte = 10 // Negotiate About Output Carriage-Return Disposition			RFC: https://www.ietf.org/rfc/rfc652.txt
	TELNET_OPT_NEG_HTAB_STOP     IACByte = 11 // Negotiate About Output Horizontal Tabstops					RFC: https://www.ietf.org/rfc/rfc653.txt
	TELNET_OPT_NEG_HTAB_DISP     IACByte = 12 // NAOHTD, Negotiate About Output Horizontal Tab Disposition	RFC: https://www.ietf.org/rfc/rfc654.txt
	TELNET_OPT_NEG_FF_DISP       IACByte = 13 // Negotiate About Output Formfeed Disposition				RFC: https://www.ietf.org/rfc/rfc655.txt
	TELNET_OPT_NEG_VTAB_STOP     IACByte = 14 // Negotiate About Vertical Tabstops							RFC: https://www.ietf.org/rfc/rfc656.txt
	TELNET_OPT_NEG_VTAB_DISP     IACByte = 15 // Negotiate About Output Vertcial Tab Disposition			RFC: https://www.ietf.org/rfc/rfc657.txt
	TELNET_OPT_NEG_LF_DISP       IACByte = 16 // Negotiate About Output Linefeed Disposition				RFC: https://www.ietf.org/rfc/rfc658.txt
	TELNET_OPT_EXT_ASCII         IACByte = 17 // Extended ASCII.											RFC: https://www.ietf.org/rfc/rfc698.txt
	TELNET_OPT_LOGOUT            IACByte = 18 // Logout.													RFC: https://www.ietf.org/rfc/rfc727.txt
	TELNET_OPT_BYTE_MACRO        IACByte = 19 // Byte Macro													RFC: https://www.ietf.org/rfc/rfc735.txt
	TELNET_OPT_DATAENTRY_TERM    IACByte = 20 // Data Entry Terminal										RFC: https://www.ietf.org/rfc/rfc732.txt : https://www.ietf.org/rfc/rfc1043.txt
	TELNET_OPT_SUPDUP            IACByte = 21 // SUPDUP														RFC: https://www.ietf.org/rfc/rfc734.txt : https://www.ietf.org/rfc/rfc736.txt
	TELNET_OPT_SUPDUP_OUT        IACByte = 22 // SUPDUP Output												RFC: https://www.ietf.org/rfc/rfc749.txt
	TELNET_OPT_SEND_LOC          IACByte = 23 // Send Location												RFC: https://www.ietf.org/rfc/rfc779.txt
	TELNET_OPT_EOR               IACByte = 25 // End of Record												RFC: https://www.ietf.org/rfc/rfc885.txt
	TELNET_OPT_TACACS_USER_ID    IACByte = 26 // TACACS User Identification									RFC: https://www.ietf.org/rfc/rfc927.txt
	TELNET_OPT_OUTPUT_MARK       IACByte = 27 // Output Marking												RFC: https://www.ietf.org/rfc/rfc933.txt
	TELNET_OPT_TTYLOC            IACByte = 28 // TTYLOC, Terminal Location Number.							RFC: https://www.ietf.org/rfc/rfc946.txt
	TELNET_OPT_3270_REGIME       IACByte = 29 // Telnet 3270 Regime											RFC: https://www.ietf.org/rfc/rfc1041.txt
	TELNET_OPT_X3_PAD            IACByte = 30 // X.3 PAD.													RFC: https://www.ietf.org/rfc/rfc1053.txt
	TELNET_OPT_X_DISP_LOC        IACByte = 35 // X Display Location.										RFC: https://www.ietf.org/rfc/rfc1096.txt
	TELNET_OPT_AUTH              IACByte = 37 // Authentication												RFC: https://www.ietf.org/rfc/rfc1416.txt | https://www.ietf.org/rfc/rfc2941.txt | https://www.ietf.org/rfc/rfc2942.txt | https://www.ietf.org/rfc/rfc2943.txt | https://www.ietf.org/rfc/rfc2951.txt
	TELNET_OPT_CRYPT             IACByte = 38 // Encryption Option											RFC: https://www.ietf.org/rfc/rfc2946.txt
	TELNET_OPT_NEW_ENV           IACByte = 39 // New Environment											RFC: https://www.ietf.org/rfc/rfc1572.txt
	TELNET_OPT_TN3270E           IACByte = 40 // TN3270E													RFC: https://www.ietf.org/rfc/rfc2355.txt
	TELNET_OPT_XAUTH             IACByte = 41 // XAUTH														RFC:
	TELNET_OPT_CHARSET           IACByte = 42 // CHARSET													RFC: https://www.ietf.org/rfc/rfc2066.txt
	TELNET_OPT_RSP               IACByte = 43 // RSP, Telnet Remote Serial Port								RFC:
	TELNET_OPT_COM_PORT          IACByte = 44 // Com Port Control											RFC: https://www.ietf.org/rfc/rfc2217.txt
	TELNET_OPT_SUPPRESS_LOC_ECHO IACByte = 45 // Telnet Suppress Local Echo									RFC:
	TELNET_OPT_START_TLS         IACByte = 46 // Telnet Start TLS											RFC:
	TELNET_OPT_KERMIT            IACByte = 47 // KERMIT														RFC: https://www.ietf.org/rfc/rfc2840.txt
	TELNET_OPT_SEND_URL          IACByte = 48 // SEND-URL													RFC:
	TELNET_OPT_FORWARD_X         IACByte = 49 // FORWARD_X													RFC:
	TELNET_OPT_137               IACByte = 137
	TELNET_OPT_PRAGMA_LOGON      IACByte = 138 // TELOPT PRAGMA LOGON										RFC:
	TELNET_OPT_SSPI_LOGON        IACByte = 139 // TELOPT SSPI LOGON											RFC:
	TELNET_OPT_PRAGMA_HEARTBEAT  IACByte = 140 // TELOPT PRAGMA HEARTBEAT									RFC:
	TELNET_OPT_254               IACByte = 254
	TELNET_OPT_EXTENDED_OPT      IACByte = 255 // Extended-Options-List										RFC: https://www.ietf.org/rfc/rfc861.txt

	// NEW-ENVIRON (option 39) sub-negotiation codes.								RFC: https://www.ietf.org/rfc/rfc1572.txt
	// Used by the MNES (Mud New-Environ Standard) handshake that we use to detect
	// Mudlet (and other clients that advertise CLIENT_NAME). Their byte values
	// coincide with telnet option codes already defined above, so they are aliased
	// to those constants instead of being re-declared as literals.
	TELNET_NEWENV_IS      = TELNET_OPT_TXBIN      // 0: Client -> Server: here are my variables
	TELNET_NEWENV_SEND    = TELNET_OPT_ECHO       // 1: Server -> Client: please send these variables
	TELNET_NEWENV_INFO    = TELNET_OPT_RECONN     // 2: Client -> Server: a variable changed
	TELNET_NEWENV_VAR     = TELNET_OPT_TXBIN      // 0: A well-known variable name follows
	TELNET_NEWENV_VALUE   = TELNET_NEWENV_SEND    // 1: A variable value follows
	TELNET_NEWENV_ESC     = TELNET_NEWENV_INFO    // 2: Escape the next byte (it is data, not a code)
	TELNET_NEWENV_USERVAR = TELNET_OPT_SUP_GO_AHD // 3: A user-defined variable name follows
)

// https://users.cs.cf.ac.uk/Dave.Marshall/Internet/node142.html
// TelnetWILL(TELNET_OPT_SUP_GO_AHD) + TelnetWill(TELNET_OPT_ECHO) + TelnetWONT(TELNET_OPT_LINE_MODE)
func TelnetWILL(what IACByte) []IACByte {
	return []IACByte{TELNET_IAC, TELNET_WILL, what}
}

func TelnetWONT(what IACByte) []IACByte {
	return []IACByte{TELNET_IAC, TELNET_WONT, what}
}

func TelnetDO(what IACByte) []IACByte {
	return []IACByte{TELNET_IAC, TELNET_DO, what}
}

func TelnetDONT(what IACByte) []IACByte {
	return []IACByte{TELNET_IAC, TELNET_DONT, what}
}

func TelnetParseScreenSizePayload(info []byte) (width int, height int, err error) {

	// Width and height are two bytes each, so a report shorter than four
	// bytes cannot be read. Any client can send one.
	if len(info) >= 4 {

		width = (int(info[0]) << 8) | int(info[1])
		height = (int(info[2]) << 8) | int(info[3])

	} else {
		err = errors.New("not enough IAC commands to properly parse")
	}

	return

}

func iacByteString(b byte) string {

	switch b {
	case TELNET_IAC:
		return "IAC"
	case TELNET_DONT:
		return "DONT"
	case TELNET_DO:
		return "DO"
	case TELNET_WONT:
		return "WONT"
	case TELNET_WILL:
		return "WILL"
	case TELNET_SB:
		return "SB"
	case TELNET_GA:
		return "_GA"
	case TELNET_EL:
		return "EL"
	case TELNET_EC:
		return "EC"
	case TELNET_AYT:
		return "AYT"
	case TELNET_AO:
		return "AO"
	case TELNET_IP:
		return "IP"
	case TELNET_BRK:
		return "BRK"
	case TELNET_DM:
		return "DM"
	case TELNET_NOP:
		return "NOP"
	case TELNET_SE:
		return "SE"
	// options
	case TELNET_OPT_TXBIN:
		return "OPT_TXBIN"
	case TELNET_OPT_ECHO:
		return "OPT_ECHO"
	case TELNET_OPT_SUP_GO_AHD:
		return "OPT_SUP_GO_AHD"
	case TELNET_OPT_STAT:
		return "OPT_STAT"
	case TELNET_OPT_TMARK:
		return "OPT_TMARK"
	case TELNET_OPT_TERM_TYPE:
		return "OPT_TERM_TYPE"
	case TELNET_OPT_NAWS:
		return "OPT_NAWS"
	case TELNET_OPT_TERM_SPD:
		return "OPT_TERM_SPD"
	case TELNET_OPT_RMT_FC:
		return "OPT_RMT_FC"
	case TELNET_OPT_LINE_MODE:
		return "OPT_LINE_MODE"
	case TELNET_OPT_ENV:
		return "OPT_ENV"
	// Random have come up
	case TELNET_OPT_NEW_ENV: // 39
		return "OPT_NEW_ENV"
	}

	return "?"
}
