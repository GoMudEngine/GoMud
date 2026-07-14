//go:build !windows

package connections

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// IssueWebSocketReconnectToken is set by main to avoid an import cycle.
// It is called for each WebSocket connection during CopyoverSave, returning a
// one-time token the client can use to skip the login prompt after reconnecting.
var IssueWebSocketReconnectToken func(connectionId ConnectionId) (string, error)

func (c *connectionsCopyoverContributor) CopyoverSave(enc *copyover.Encoder) error {
	lock.Lock()
	defer lock.Unlock()

	state := connectionsState{
		ConnectCounter: connectCounter,
	}

	for id, cd := range netConnections {
		if cd.IsWebSocket() {
			if IssueWebSocketReconnectToken != nil {
				if token, err := IssueWebSocketReconnectToken(id); err == nil {
					cd.Write([]byte("RELOGTKN:" + token))
				} else {
					mudlog.Warn("copyover: could not issue reconnect token", "connectionId", id, "error", err)
					cd.Write([]byte("\r\nServer is rebooting. Please reconnect.\r\n"))
				}
			} else {
				cd.Write([]byte("\r\nServer is rebooting. Please reconnect.\r\n"))
			}
			continue
		}

		fd, ok := connRawFd(cd.conn)
		if !ok {
			mudlog.Warn("copyover: could not get fd for connection, skipping", "connectionId", id)
			continue
		}

		if _, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_SETFD, 0); errno != 0 {
			return fmt.Errorf("connections copyover: fcntl clear cloexec fd %d: %w", fd, errno)
		}

		state.Records = append(state.Records, connectionRecord{
			ConnectionId:   id,
			State:          cd.State(),
			Fd:             int(fd),
			ClientSettings: cd.clientSettings,
		})
	}

	return enc.WriteSection(c.CopyoverName(), state)
}

func (c *connectionsCopyoverContributor) CopyoverRestore(dec *copyover.Decoder) error {
	var state connectionsState
	if err := dec.ReadSection(c.CopyoverName(), &state); err != nil {
		return err
	}

	lock.Lock()
	defer lock.Unlock()

	connectCounter = state.ConnectCounter

	for _, rec := range state.Records {
		if rec.State != LoggedIn {
			syscall.Close(rec.Fd)
			continue
		}

		f := os.NewFile(uintptr(rec.Fd), "tcp")
		conn, err := net.FileConn(f)
		f.Close()
		if err != nil {
			mudlog.Warn("copyover: could not restore connection from fd", "fd", rec.Fd, "error", err)
			continue
		}

		cd := NewConnectionDetails(rec.ConnectionId, conn, nil, nil)
		cd.SetState(rec.State)
		cd.clientSettings = rec.ClientSettings

		netConnections[rec.ConnectionId] = cd
	}

	return nil
}

func connRawFd(conn net.Conn) (uintptr, bool) {
	type syscallConner interface {
		SyscallConn() (syscall.RawConn, error)
	}

	sc, ok := conn.(syscallConner)
	if !ok {
		return 0, false
	}

	raw, err := sc.SyscallConn()
	if err != nil {
		return 0, false
	}

	var fd uintptr
	if err := raw.Control(func(f uintptr) { fd = f }); err != nil {
		return 0, false
	}

	return fd, true
}
