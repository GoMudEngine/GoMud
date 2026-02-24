package connections

import (
	"errors"
	"net"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/gorilla/websocket"
)

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

type ConnectState uint32

const (
	Login ConnectState = iota
	LoggedIn
	Zombie
	MaxHistory = 10
)

type ConnType uint32

const (
	ConnHuman ConnType = 0
	ConnAI    ConnType = 1
)

type InputHistory struct {
	inhistory bool
	position  int
	history   [][]byte
}

func (ih *InputHistory) Get() []byte {
	if len(ih.history) < 1 {
		return nil
	}

	return ih.history[ih.position]
}

func (ih *InputHistory) Add(input []byte) {

	if len(ih.history) >= MaxHistory {
		ih.history = ih.history[1:]
	}

	ih.history = append(ih.history, make([]byte, len(input)))
	ih.position = len(ih.history) - 1
	copy(ih.history[ih.position], input)
	ih.inhistory = false
}

func (ih *InputHistory) Previous() {
	if !ih.inhistory {
		ih.inhistory = true
		return
	}
	if ih.position <= 0 {
		return
	}

	ih.position--
}

func (ih *InputHistory) Next() {
	if !ih.inhistory {
		ih.inhistory = true
		return
	}
	if ih.position >= len(ih.history)-1 {
		return
	}

	ih.position++
}

// returns position and whether position is not the last item
func (ih *InputHistory) Position() int {
	return ih.position
}

func (ih *InputHistory) ResetPosition() {
	ih.inhistory = false
	ih.position = len(ih.history) - 1
	if ih.position < 0 {
		ih.position = 0
	}
}

func (ih *InputHistory) InHistory() bool {
	return ih.inhistory
}

// A structure to package up everything we need to know about this input.
type ClientInput struct {
	ConnectionId  ConnectionId // Who does this belong to?
	DataIn        []byte       // What was the last thing they typed?
	Buffer        []byte       // What is the current buffer
	Clipboard     []byte       // Text that can be easily pasted with ctrl-v
	LastSubmitted []byte       // The last thing submitted
	EnterPressed  bool         // Did they hit enter? It's stripped from the buffer/input FYI
	BSPressed     bool         // Did they hit backspace?
	TabPressed    bool         // Did they hit tab?
	History       InputHistory // A list of the last 10 things they typed
}

// Reset the client input to essentially "No current input"
func (ci *ClientInput) Reset() {
	ci.DataIn = ci.DataIn[:0]
	ci.Buffer = ci.Buffer[:0]
	ci.EnterPressed = false
}

type InputHandler func(ci *ClientInput, handlerState map[string]any) (doNextHandler bool)

type ConnectionDetails struct {
	connectionId      ConnectionId
	connType          ConnType
	state             ConnectState
	lastInputTime     time.Time
	conn              net.Conn
	wsConn            *websocket.Conn
	wsLock            sync.Mutex
	handlerMutex      sync.Mutex
	inputHandlerNames []string
	inputHandlers     []InputHandler
	inputDisabled     bool
	clientSettings    ClientSettings
	heartbeat         *heartbeatManager
	stripAnsi         bool
	aiCommandCount    int
	aiCommandRound    int64
}

func (cd *ConnectionDetails) IsLocal() bool {

	remoteAddrStr := ``

	if cd.wsConn == nil {
		// Unix sockets are always local
		if _, ok := cd.conn.(*net.UnixConn); ok {
			return true
		}
		remoteAddrStr = cd.conn.RemoteAddr().String()
	} else {
		remoteAddrStr = cd.wsConn.RemoteAddr().String()
	}

	host, _, err := net.SplitHostPort(remoteAddrStr)
	if err != nil {
		// e.g. not “host:port” syntax
		return false
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

func (cd *ConnectionDetails) IsWebSocket() bool {
	return cd.wsConn != nil
}

// If HandleInput receives an error, we shouldn't pass input to the game logic
func (cd *ConnectionDetails) HandleInput(ci *ClientInput, handlerState map[string]any) (doNextHandler bool, lastHandler string, err error) {
	cd.handlerMutex.Lock()
	defer cd.handlerMutex.Unlock()

	cd.lastInputTime = time.Now()

	handlerCt := len(cd.inputHandlers)
	if handlerCt < 1 {
		return false, lastHandler, errors.New("no input handlers")
	}

	for i, inputHandler := range cd.inputHandlers {
		lastHandler = cd.inputHandlerNames[i]
		if runNextHandler := inputHandler(ci, handlerState); !runNextHandler {
			// If it's the last one in the chain, ignore any aborts
			// if i == handlerCt-1 {
			// 	return false, lastHandler, nil
			// }
			return false, lastHandler, nil
		}
	}
	return true, lastHandler, nil
}

func (cd *ConnectionDetails) RemoveInputHandler(name string) {
	cd.handlerMutex.Lock()
	defer cd.handlerMutex.Unlock()

	for i := len(cd.inputHandlerNames) - 1; i >= 0; i-- {
		if cd.inputHandlerNames[i] == name {
			cd.inputHandlerNames = append(cd.inputHandlerNames[:i], cd.inputHandlerNames[i+1:]...)
			cd.inputHandlers = append(cd.inputHandlers[:i], cd.inputHandlers[i+1:]...)
		}
	}

}

func (cd *ConnectionDetails) AddInputHandler(name string, newInputHandler InputHandler, after ...string) {
	cd.handlerMutex.Lock()
	defer cd.handlerMutex.Unlock()

	if len(after) > 0 {
		for i, handlerName := range cd.inputHandlerNames {
			if handlerName == after[0] {
				cd.inputHandlerNames = append(cd.inputHandlerNames[:i+1], append([]string{name}, cd.inputHandlerNames[i+1:]...)...)
				cd.inputHandlers = append(cd.inputHandlers[:i+1], append([]InputHandler{newInputHandler}, cd.inputHandlers[i+1:]...)...)
				return
			}
		}
	}

	cd.inputHandlerNames = append(cd.inputHandlerNames, name)
	cd.inputHandlers = append(cd.inputHandlers, newInputHandler)
}

func (cd *ConnectionDetails) Write(p []byte) (n int, err error) {

	p = []byte(strings.ReplaceAll(string(p), "\n", "\r\n"))

	if len(p) == 0 {
		return 0, nil
	}

	// Strip ANSI escape codes for AI connections (not telnet IAC commands)
	if cd.stripAnsi && p[0] != term.TELNET_IAC {
		p = ansiRegexp.ReplaceAll(p, nil)
		if len(p) == 0 {
			return 0, nil
		}
	}

	if cd.wsConn != nil {
		cd.wsLock.Lock()
		defer cd.wsLock.Unlock()

		// If this isn't caught and avoided, lots of stuff goes wrong.
		// Websocket client complains, disconnects, error is rasised: close 1002 (protocol error): Invalid UTF-8 in text frame
		// Then a panic ensues as the server tries to write to a socket that's nil.
		// TODO: Investigate cleaning up this condition better.
		if p[0] == term.TELNET_IAC {
			mudlog.Error("conn.Write", "error", "Trying to send telnet command to websocket!", "bytes", p, "string", string(p))
			return 0, nil
		}

		err := cd.wsConn.WriteMessage(websocket.TextMessage, p)
		if err != nil {
			return 0, err
		}
		return len(p), nil
	}

	return cd.conn.Write(p)
}

func (cd *ConnectionDetails) Read(p []byte) (n int, err error) {

	if cd.wsConn != nil {
		// read the bytes and then copy them into p
		_, message, err := cd.wsConn.ReadMessage()
		if err != nil {
			return 0, err
		}
		copy(p, message)
		return len(message), nil
	}

	return cd.conn.Read(p)
}

func (cd *ConnectionDetails) Close() {
	if cd.heartbeat != nil {
		cd.heartbeat.stop()
	}

	if cd.wsConn != nil {
		cd.wsConn.Close()
		return
	}
	cd.conn.Close()
}

func (cd *ConnectionDetails) RemoteAddr() net.Addr {
	if cd.wsConn != nil {
		return cd.wsConn.RemoteAddr()
	}
	return cd.conn.RemoteAddr()
}

// get for uniqueId
func (cd *ConnectionDetails) ConnectionId() ConnectionId {
	return ConnectionId(atomic.LoadUint64((*uint64)(&cd.connectionId)))
}

// set and get for state
func (cd *ConnectionDetails) State() ConnectState {
	return ConnectState(atomic.LoadUint32((*uint32)(&cd.state)))
}

func (cd *ConnectionDetails) SetState(state ConnectState) {
	atomic.StoreUint32((*uint32)(&cd.state), uint32(state))
}

func (cd *ConnectionDetails) InputDisabled(setTo ...bool) bool {
	if len(setTo) > 0 {
		cd.inputDisabled = setTo[0]
	}
	return cd.inputDisabled
}

func (cd *ConnectionDetails) ConnType() ConnType {
	return ConnType(atomic.LoadUint32((*uint32)(&cd.connType)))
}

func (cd *ConnectionDetails) SetConnType(t ConnType) {
	atomic.StoreUint32((*uint32)(&cd.connType), uint32(t))
}

func (cd *ConnectionDetails) SetStripAnsi(strip bool) {
	cd.stripAnsi = strip
}

// AICommandAllowed checks whether an AI connection is allowed to send another command this round.
// Returns true if the command is allowed. Resets the counter when a new round begins.
func (cd *ConnectionDetails) AICommandAllowed(currentRound int64, maxPerRound int) bool {
	if currentRound != cd.aiCommandRound {
		cd.aiCommandRound = currentRound
		cd.aiCommandCount = 0
	}
	cd.aiCommandCount++
	return cd.aiCommandCount <= maxPerRound
}

func NewConnectionDetails(connId ConnectionId, c net.Conn, wsC *websocket.Conn, config *HeartbeatConfig) *ConnectionDetails {
	if config == nil {
		config = &DefaultHeartbeatConfig
	}
	cd := &ConnectionDetails{
		state:         Login,
		connectionId:  connId,
		inputDisabled: false,
		conn:          c,
		wsConn:        wsC,
		wsLock:        sync.Mutex{},
		// Track client settings
		clientSettings: ClientSettings{
			Display: DisplaySettings{ScreenWidth: 80, ScreenHeight: 40}, // Default to 80x40
		},
	}

	if wsC != nil {
		if err := cd.StartHeartbeat(*config); err != nil {
			mudlog.Error("Heartbeat",
				"connectionId", connId,
				"error", err)
		}
	}

	return cd
}
