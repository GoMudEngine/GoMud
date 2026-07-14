package inputhandlers

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/term"
)

var (
	iacHandlers = []IACHandler{}
)

type IACHandler interface {
	HandleIAC(uint64, []byte) bool
}

func AddIACHandler(h IACHandler) {
	iacHandlers = append(iacHandlers, h)
}

func TelnetIACHandler(clientInput *connections.ClientInput, sharedState map[string]any) (nextHandler bool) {

	// Check for Telnet IAC commands
	// If not, pass it on to next handler
	if !term.IsTelnetCommand(clientInput.DataIn) {
		return true
	}

	// Multiple Telnet IAC's can be stacked into one send, so useful to split them out
	iacCmds := [][]byte{}

	var lastIAC int = 0
	for i, b := range clientInput.DataIn {
		if i != 0 && b == term.TELNET_IAC {
			if i < len(clientInput.DataIn)-1 && clientInput.DataIn[i+1] != term.TELNET_SE {
				iacCmds = append(iacCmds, clientInput.DataIn[lastIAC:i])
				lastIAC = i
			}
		}
	}

	//mudlog.Debug("Received", "type", "IAC (TEST)", "data", term.BytesString(clientInput.DataIn))

	if lastIAC < len(clientInput.DataIn) {
		iacCmds = append(iacCmds, clientInput.DataIn[lastIAC:])
	}

	for _, iacCmd := range iacCmds {
		// Check incoming Telnet IAC commands for anything useful...

		if len(iacHandlers) > 0 {

			handlerFound := false
			for _, h := range iacHandlers {
				if h.HandleIAC(clientInput.ConnectionId, iacCmd) {
					handlerFound = true
					break
				}
			}

			if handlerFound {
				continue
			}

		}

		if term.IsMSSPCommand(iacCmd) {
			// A crawler accepted our WILL MSSP (IAC DO MSSP) — reply once with
			// the status block. A DONT/refuse is a silent no-op.
			if ok, _ := term.Matches(iacCmd, term.MsspAccept); ok {
				if fields := gatherMSSPFields(); len(fields) > 0 {
					connections.SendTo(
						term.MsspCommand.BytesWithPayload(term.EncodeMSSPPayload(fields)),
						clientInput.ConnectionId,
					)
				}
			}
			continue
		}

		if term.IsMSPCommand(iacCmd) {

			if ok, payload := term.Matches(iacCmd, term.MspAccept); ok {
				mudlog.Debug("Received", "type", "IAC (Client-MSP Accept)", "data", term.BytesString(payload))

				cs := connections.GetClientSettings(clientInput.ConnectionId)
				cs.MSPEnabled = true
				connections.OverwriteClientSettings(clientInput.ConnectionId, cs)

				connections.SendTo(
					term.MspCommand.BytesWithPayload([]byte("!!SOUND(Off U="+configs.GetFilePathsConfig().WebCDNLocation.String()+")")),
					clientInput.ConnectionId,
				)

				continue
			}

			if ok, payload := term.Matches(iacCmd, term.MspRefuse); ok {
				mudlog.Debug("Received", "type", "IAC (Client-MSP Refuse)", "data", term.BytesString(payload))

				cs := connections.GetClientSettings(clientInput.ConnectionId)
				cs.MSPEnabled = false
				connections.OverwriteClientSettings(clientInput.ConnectionId, cs)

				continue
			}

			continue
		}

		if ok, payload := term.Matches(iacCmd, term.TelnetAcceptedChangeCharset); ok {
			mudlog.Debug("Received", "type", "IAC (TelnetAcceptedChangeCharset)", "data", term.BytesString(payload))
			continue
		}

		if ok, _ := term.Matches(iacCmd, term.TelnetRejectedChangeCharset); ok {
			mudlog.Debug("Received", "type", "IAC (TelnetRejectedChangeCharset)")
			continue
		}

		if ok, _ := term.Matches(iacCmd, term.TelnetAgreeChangeCharset); ok {
			mudlog.Debug("Received", "type", "IAC (TelnetAgreeChangeCharset)")
			connections.SendTo(
				term.TelnetCharset.BytesWithPayload([]byte(" UTF-8")),
				clientInput.ConnectionId,
			)
			continue
		}

		if ok, _ := term.Matches(iacCmd, term.TelnetDontSuppressGoAhead); ok {
			mudlog.Debug("Received", "type", "IAC (TelnetDontSuppressGoAhead)")

			cs := connections.GetClientSettings(clientInput.ConnectionId)
			cs.SendTelnetGoAhead = true
			connections.OverwriteClientSettings(clientInput.ConnectionId, cs)

			continue
		}

		// Is it a screen size report?
		if ok, payload := term.Matches(iacCmd, term.TelnetScreenSizeResponse); ok {

			w, h, err := term.TelnetParseScreenSizePayload(payload)
			if err != nil {
				mudlog.Debug("Received", "type", "IAC (Screensize)", "data", term.BytesString(payload), "error", err)
			} else {
				mudlog.Debug("Received", "type", "IAC (Screensize)", "width", w, "height", h)

				if err == nil {

					cs := connections.GetClientSettings(clientInput.ConnectionId)
					cs.Display.ScreenWidth = uint32(w)
					cs.Display.ScreenHeight = uint32(h)
					connections.OverwriteClientSettings(clientInput.ConnectionId, cs)

				}

			}

			continue
		}

		// Unhanlded IAC command, log it
		mudlog.Debug("Received", "type", "IAC (Unhandled)", "size", len(clientInput.DataIn), "data", term.TelnetCommandToString(iacCmd))

	}

	// We handled it, so don't pass it on
	return false
}
