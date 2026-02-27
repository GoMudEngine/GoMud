package connections

import (
	"errors"
	"net"
	"os"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/gorilla/websocket"
)

const ReadBufferSize = 1024

type ConnectionId = uint64

var (

	//
	// Mutex
	//
	lock sync.RWMutex = sync.RWMutex{}
	//
	// Counters
	//
	connectCounter    uint64 = 0 // a counter for each time a connection is accepted
	disconnectCounter uint64 = 0 // a counter for each tim ea connection is dropped
	//
	// Track connections
	//
	netConnections map[ConnectionId]*ConnectionDetails = map[ConnectionId]*ConnectionDetails{} // a mapping of unique id's to connections
	//
	// Channel to send a shutdown signal to
	//
	shutdownChannel chan os.Signal // channel to receive shutdown signals
)

func SignalShutdown(s os.Signal) {
	if shutdownChannel != nil {
		shutdownChannel <- s
	}
}

func Add(conn net.Conn, wsConn *websocket.Conn, cType ...ConnType) *ConnectionDetails {

	lock.Lock()
	defer lock.Unlock()

	connectCounter++

	connDetails := NewConnectionDetails(
		connectCounter,
		conn,
		wsConn,
		nil, // use default settings for now TODO: add into overall config pattern?
	)

	if len(cType) > 0 {
		connDetails.connType = cType[0]
	}

	netConnections[connDetails.ConnectionId()] = connDetails

	// return the unique ID to find this connection later
	return connDetails
}

// Returns the total number of connections
func Get(id ConnectionId) *ConnectionDetails {
	lock.Lock()
	defer lock.Unlock()

	return netConnections[id]
}

func IsWebsocket(id ConnectionId) bool {
	lock.Lock()
	defer lock.Unlock()

	if cd, ok := netConnections[id]; ok {
		return cd.IsWebSocket()
	}

	return false
}

func GetAllConnectionIds() []ConnectionId {

	lock.Lock()
	defer lock.Unlock()

	ids := make([]ConnectionId, len(netConnections))

	for id := range netConnections {
		ids = append(ids, id)
	}

	return ids
}

func Cleanup() {
	for _, id := range GetAllConnectionIds() {
		Remove(id)
	}
}

func Kick(id ConnectionId, reason string) (err error) {

	lock.Lock()
	defer lock.Unlock()

	// Try to retrieve the value
	if cd, ok := netConnections[id]; ok {

		// close the connection, no longer useful.
		cd.Close()
		// keep track of the number of disconnects
		disconnectCounter++
		// remove the connection from the map
		mudlog.Info("connection kicked", "connectionId", id, "remoteAddr", cd.RemoteAddr().String(), `reason`, reason)

		return nil

	}

	return errors.New("connection not found")
}

func Remove(id ConnectionId) (err error) {

	lock.Lock()
	defer lock.Unlock()

	// Try to retrieve the value
	if cd, ok := netConnections[id]; ok {

		// close the connection, no longer useful.
		cd.Close()
		// keep track of the number of disconnects
		disconnectCounter++
		// Remove the entry
		delete(netConnections, id)

		return nil

	}

	return errors.New("connection not found")
}

func Broadcast(colorizedText []byte, skipConnectionIds ...ConnectionId) []ConnectionId {

	lock.Lock()

	removeIds := []ConnectionId{}
	sentToIds := []ConnectionId{}

	for id, cd := range netConnections {

		if cd.state == Login {
			continue
		}

		if len(skipConnectionIds) > 0 {
			skip := false
			for _, cId := range skipConnectionIds {
				if cId == id {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
		}

		// Write the message to the connection
		var err error

		_, err = cd.Write(colorizedText)

		if err != nil {
			mudlog.Warn("Broadcast()", "connectionId", id, "remoteAddr", cd.RemoteAddr().String(), "error", err)
			// Remove from the connections
			removeIds = append(removeIds, id)
		}

		sentToIds = append(sentToIds, id)
	}
	lock.Unlock()

	for _, id := range removeIds {
		Remove(id)
	}

	return sentToIds
}

func SendTo(b []byte, ids ...ConnectionId) {
	lock.Lock()

	removeIds := []ConnectionId{}

	sentCt := 0
	// iterate through all provided id's and attempt to send

	for _, id := range ids {

		if cd, ok := netConnections[id]; ok {

			if _, err := cd.Write(b); err != nil {
				mudlog.Warn("SendTo()", "connectionId", id, "remoteAddr", cd.RemoteAddr().String(), "error", err)
				// Remove from the connections
				removeIds = append(removeIds, id)
				continue
			}

		}

		sentCt++
	}

	if sentCt < 1 {
		//mudlog.Info("message sent to nobody", "message", strings.Replace(string(b), "\033", "ESC", -1))
	}

	lock.Unlock()

	for _, id := range removeIds {
		Remove(id)
	}
}

// make this more efficient later
func ActiveConnectionCount() int {
	lock.RLock()
	defer lock.RUnlock()

	return len(netConnections)
}

func ActiveHumanConnectionCount() int {
	lock.RLock()
	defer lock.RUnlock()

	ct := 0
	for _, cd := range netConnections {
		if cd.ConnType() == ConnHuman {
			ct++
		}
	}
	return ct
}

func ActiveAIConnectionCount() int {
	lock.RLock()
	defer lock.RUnlock()

	ct := 0
	for _, cd := range netConnections {
		if cd.ConnType() == ConnAI {
			ct++
		}
	}
	return ct
}

// make this more efficient later
func SetShutdownChan(osSignalChan chan os.Signal) {
	lock.Lock()
	defer lock.Unlock()

	if shutdownChannel != nil {
		panic("Can't set shutdown channel a second time!")
	}
	shutdownChannel = osSignalChan
}

func Stats() (connections uint64, disconnections uint64) {
	lock.RLock()
	defer lock.RUnlock()

	return connectCounter, disconnectCounter
}

func GetClientSettings(id ConnectionId) ClientSettings {
	lock.Lock()
	defer lock.Unlock()

	if cd, ok := netConnections[id]; ok {
		return cd.clientSettings
	}

	return ClientSettings{}
}

func OverwriteClientSettings(id ConnectionId, cs ClientSettings) {
	lock.Lock()
	defer lock.Unlock()

	if cd, ok := netConnections[id]; ok {
		cd.clientSettings = cs
	}
}
