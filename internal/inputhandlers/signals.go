package inputhandlers

import (
	"github.com/GoMudEngine/GoMud/internal/connections"
)

//
// The purpose of this handler is to convert ctrl codes into text commands, when applicable
// Basically shortcuts for common commands
//

const (
	CtrlA       = 0x01
	CtrlB       = 0x02
	CtrlC       = 0x03
	CtrlD       = 0x04
	CtrlE       = 0x05
	CtrlF       = 0x06
	CtrlN       = 0x0E
	CtrlP       = 0x10 // paste whatever is in the clipboard - This is kinda special and the clipboard should usually be empty unless you're
	CtrlQ       = 0x11 // shortcut for /quit
	CtrlR       = 0x12
	CtrlS       = 0x13
	CtrlT       = 0x14
	CtrlU       = 0x15
	CtrlV       = 0x16
	CtrlW       = 0x17
	CtrlX       = 0x18
	CtrlY       = 0x19
	CtrlZ       = 0x1A
	CtrlBkSlash = 0x1C
)

// This should be the first handler in the chain
// It will convert any special signals like CTRL-C into a /quit command
func SignalHandler(clientInput *connections.ClientInput, sharedState map[string]any) (nextHandler bool) {

	if len(clientInput.DataIn) < 1 {
		return true
	}

	if clientInput.DataIn[len(clientInput.DataIn)-1] == CtrlQ {
		clientInput.DataIn = []byte("/quit")
		clientInput.Buffer = []byte{}
		clientInput.Cursor = 0
		clientInput.EnterPressed = true
		return true
	}

	if clientInput.DataIn[len(clientInput.DataIn)-1] == CtrlW {
		clientInput.DataIn = []byte("/who")
		clientInput.Buffer = []byte{}
		clientInput.Cursor = 0
		clientInput.EnterPressed = true
		return true
	}

	if clientInput.DataIn[len(clientInput.DataIn)-1] == CtrlX {
		clientInput.DataIn = []byte("/shutdown 0")
		clientInput.Buffer = []byte{}
		clientInput.Cursor = 0
		clientInput.EnterPressed = true
		return true
	}

	if clientInput.DataIn[len(clientInput.DataIn)-1] == CtrlP {

		clientInput.DataIn = make([]byte, len(clientInput.Clipboard))
		copy(clientInput.DataIn, clientInput.Clipboard)

		clientInput.Buffer = []byte{}
		clientInput.Cursor = 0
		clientInput.EnterPressed = false
		return true
	}

	return true
}
