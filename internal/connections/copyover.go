package connections

import "github.com/GoMudEngine/GoMud/internal/copyover"

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
