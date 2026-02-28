package inputhandlers

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/templates"
)

func SystemCommandInputHandler(clientInput *connections.ClientInput, sharedState map[string]any) (nextHandler bool) {

	// if they didn't hit enter, just keep buffering, go next.
	if !clientInput.EnterPressed {
		return true
	}

	// Handle text input

	// grab whatever was typed in so far and trim out leading/trailing whitespace and null byte
	message := strings.TrimSpace(string(clientInput.Buffer))

	//mudlog.Info("SystemCommandInputHandler Received", "type", "TXT", "size", len(clientInput.Buffer), "data", string(clientInput.Buffer), "message", message)

	// If all they ever sent was white space, we won't have anything to work with. We can just ignore the input...
	// ALternatively, they may have only hit ENTER, and we may do something with that...
	if len(message) == 0 {
		return true
	}

	// If successful, we're done.
	if trySystemCommand(message, clientInput.ConnectionId) {
		// zero out the current buffer
		clientInput.Buffer = clientInput.Buffer[:0]
		return false
	}

	return true
}

// TODO: Move into own handler?

const SystemCommandPrefix = "/"

type SystemCommandHelp struct {
	Description  string
	Details      string
	ExampleInput string
}

var (
	systemCommandList = map[string]SystemCommandHelp{
		"quit": SystemCommandHelp{
			Description:  "Disconnect self from the server",
			ExampleInput: "quit",
		},
		"reload": SystemCommandHelp{
			Description:  "Reload datafiles for various packages (items, mobs, buffs, etc.)",
			ExampleInput: "reload",
		},
		"shutdown": SystemCommandHelp{
			Description:  "Shutdown the server",
			ExampleInput: "shutdown [15/seconds]",
		},
	}
)

func systemCommandParts(cmd string) (systemCmd string, cmdArg string) {

	cmd = strings.TrimSpace(cmd)

	// If there's a space split it into cmd vs arg
	if index := strings.Index(cmd, " "); index != -1 {

		systemCmd, cmdArg = strings.ToLower(cmd[0:index]), cmd[index+1:]

		if cmdArg[0:1] == " " {
			cmdArg = strings.TrimSpace(cmdArg)
		}

		return systemCmd, cmdArg
	}

	systemCmd, cmdArg = strings.ToLower(cmd), ""

	return systemCmd, cmdArg
}

func trySystemCommand(cmd string, connectionId connections.ConnectionId) bool {

	if len(cmd) < 1 {
		return false
	}

	if cmd[0:1] != SystemCommandPrefix {
		return false
	}

	cmd, arg := systemCommandParts(strings.TrimSpace(cmd[1:]))

	// look for cmd in the command list
	if _, ok := systemCommandList[cmd]; !ok {
		return false
	}

	mudlog.Info("System Command", "cmd", cmd, "arg", arg)
	//fmt.Printf("cmd:[%s] arg:[%s]\n", cmd, arg)

	if cmd == "quit" {

		// Not building complex output, so just preparse the ansi in the template and cache that
		tplTxt, _ := templates.Process("goodbye", nil)

		connections.SendTo([]byte(templates.AnsiParse(tplTxt)), connectionId)

		connections.Kick(connectionId, `/quit`)
		return true
	}

	if cmd == "reload" {
		events.AddToQueue(events.System{
			Command: "reload",
		})
	}

	if cmd == "shutdown" {
		var timeToShutdown uint64 = 15

		if len(arg) > 0 {
			timeToShutdown, _ = strconv.ParseUint(arg, 10, 64)
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					mudlog.Error("PANIC", "error", r)
					s := string(debug.Stack())
					for _, str := range strings.Split(s, "\n") {
						mudlog.Error("PANIC", "stack", str)
					}
				}
			}()

			// Not building complex output, so just preparse the ansi in the template and cache that
			tplTxt, _ := templates.Process("admincommands/shutdown-countdown", nil)

			for i := timeToShutdown; i > 0; i-- {

				writeOut := false
				if i == timeToShutdown {
					writeOut = true
				} else if i > 60 {
					if i%30 == 0 {
						writeOut = true
					}
				} else if i > 15 {
					if i%15 == 0 {
						writeOut = true
					}
				} else if i%5 == 0 {
					writeOut = true
				}

				if writeOut {

					events.AddToQueue(events.Broadcast{Text: fmt.Sprintf(tplTxt, i)})

				}

				time.Sleep(time.Second)
			}
			connections.SignalShutdown(syscall.SIGTERM)

		}()
		return true
	}

	mudlog.Error("valid command unhandled", "cmd", cmd, "arg", arg)

	return true
}
