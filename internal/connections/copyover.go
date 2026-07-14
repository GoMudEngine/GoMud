package connections

import "github.com/GoMudEngine/GoMud/internal/copyover"

// IssueWebSocketReconnectToken is set by main to avoid an import cycle. It is
// called for each WebSocket connection during CopyoverSave (Unix only),
// returning a one-time token the client uses to skip the login prompt after
// reconnecting. Declared here (not the Unix-only fd file) so main can reference
// it on every platform.
var IssueWebSocketReconnectToken func(connectionId ConnectionId) (string, error)

type connectionRecord struct {
	ConnectionId   ConnectionId   `json:"connection_id"`
	State          ConnectState   `json:"state"`
	Fd             int            `json:"fd"`
	ClientSettings ClientSettings `json:"client_settings"`
}

type connectionsState struct {
	ConnectCounter uint64             `json:"connect_counter"`
	Records        []connectionRecord `json:"records"`
}

type connectionsCopyoverContributor struct{}

func (c *connectionsCopyoverContributor) CopyoverName() string {
	return "connections"
}

// CopyoverContributor returns the connections contributor for registration.
func CopyoverContributor() copyover.Contributor {
	return &connectionsCopyoverContributor{}
}
