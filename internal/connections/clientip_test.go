package connections

import (
	"net"
	"testing"
)

// fakeConn is a net.Conn that only needs to answer RemoteAddr(); ClientIP and
// IsLocal touch nothing else.
type fakeConn struct {
	net.Conn
	remote net.Addr
}

func (f *fakeConn) RemoteAddr() net.Addr { return f.remote }

func tcpAddr(t *testing.T, s string) net.Addr {
	t.Helper()
	a, err := net.ResolveTCPAddr("tcp", s)
	if err != nil {
		t.Fatalf("ResolveTCPAddr(%q): %v", s, err)
	}
	return a
}

// Telnet has no proxy in front of it. Nothing sets clientIP on a telnet
// connection, so ClientIP() must still be the socket peer and IsLocal() must
// behave exactly as it did before proxy resolution existed.
func TestClientIPTelnetPathUnchanged(t *testing.T) {
	tests := []struct {
		name      string
		remote    string
		wantIP    string
		wantLocal bool
	}{
		{"remote telnet client", "203.0.113.50:41000", "203.0.113.50", false},
		{"loopback telnet client", "127.0.0.1:41000", "127.0.0.1", true},
		{"ipv6 loopback telnet client", "[::1]:41000", "::1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd := &ConnectionDetails{conn: &fakeConn{remote: tcpAddr(t, tt.remote)}}

			if got := cd.ClientIP(); got != tt.wantIP {
				t.Fatalf("ClientIP() = %q, want %q", got, tt.wantIP)
			}
			if got := cd.IsLocal(); got != tt.wantLocal {
				t.Fatalf("IsLocal() = %t, want %t", got, tt.wantLocal)
			}
		})
	}
}

// The defect: a proxied websocket's socket peer is the reverse proxy, so
// IsLocal() was true for every web-client player and the IP-ban check was
// skipped. Once the resolved client IP is recorded, both must reflect the real
// origin while RemoteAddr() keeps reporting the actual socket peer for logs.
func TestClientIPOverridesProxyPeer(t *testing.T) {
	cd := &ConnectionDetails{conn: &fakeConn{remote: tcpAddr(t, "127.0.0.1:52000")}}

	if !cd.IsLocal() {
		t.Fatalf("precondition: a loopback peer with no resolved client IP should read as local")
	}

	cd.SetClientIP("203.0.113.77")

	if got := cd.ClientIP(); got != "203.0.113.77" {
		t.Fatalf("ClientIP() = %q, want 203.0.113.77", got)
	}
	if cd.IsLocal() {
		t.Fatalf("IsLocal() = true; a proxied remote player must not read as loopback")
	}
	if got := cd.RemoteAddr().String(); got != "127.0.0.1:52000" {
		t.Fatalf("RemoteAddr() = %q; the socket peer must still be reported for logging", got)
	}
}

// A genuinely local websocket (browser on the server box, no proxy) resolves to
// loopback and must keep its exemption.
func TestClientIPLocalWebsocketStaysLocal(t *testing.T) {
	cd := &ConnectionDetails{conn: &fakeConn{remote: tcpAddr(t, "127.0.0.1:52000")}}
	cd.SetClientIP("127.0.0.1")

	if !cd.IsLocal() {
		t.Fatalf("IsLocal() = false, want true for a genuinely loopback client")
	}
}

// Unix sockets have no host:port and are always local.
func TestClientIPUnixSocketIsLocal(t *testing.T) {
	cd := &ConnectionDetails{conn: &net.UnixConn{}}

	if !cd.IsLocal() {
		t.Fatalf("IsLocal() = false, want true for a unix socket")
	}
	if got := cd.ClientIP(); got != "127.0.0.1" {
		t.Fatalf("ClientIP() = %q, want 127.0.0.1 for a unix socket", got)
	}
}
