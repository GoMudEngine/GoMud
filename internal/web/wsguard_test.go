package web

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
)

func originRequest(host, origin string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/ws", nil)
	r.Host = host
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	return r
}

// The defect: CheckOrigin returned true unconditionally, so any third-party
// page could open sockets from a visitor's browser.
func TestWsOriginRejectsCrossOrigin(t *testing.T) {
	rejected := []struct {
		name   string
		host   string
		origin string
	}{
		{"unrelated site", "dogmud.org", "https://evil.example"},
		{"lookalike subdomain", "dogmud.org", "https://dogmud.org.evil.example"},
		{"suffix trick", "dogmud.org", "https://notdogmud.org"},
		{"prefix trick", "dogmud.org", "https://evildogmud.org"},
		{"attacker on localhost name", "dogmud.org", "https://evil.localhost.example"},
		{"opaque origin", "dogmud.org", "null"},
		{"unparseable", "dogmud.org", "://///"},
	}

	for _, tt := range rejected {
		t.Run(tt.name, func(t *testing.T) {
			if wsOriginAllowed(originRequest(tt.host, tt.origin)) {
				t.Fatalf("origin %q was allowed against host %q", tt.origin, tt.host)
			}
		})
	}
}

// Legitimate traffic must keep working, or the fix gets reverted rather than
// tuned.
func TestWsOriginAllowsLegitimate(t *testing.T) {
	allowed := []struct {
		name   string
		host   string
		origin string
	}{
		{"same origin https", "dogmud.org", "https://dogmud.org"},
		{"same origin http", "dogmud.org", "http://dogmud.org"},
		{"same origin with port on origin", "dogmud.org", "https://dogmud.org:443"},
		{"same origin with port on host", "dogmud.org:80", "http://dogmud.org"},
		{"case-insensitive host", "DogMud.org", "https://dogmud.ORG"},
		{"configured WebDomain (localhost)", "192.168.1.20", "http://localhost:8080"},
		{"loopback ip", "192.168.1.20", "http://127.0.0.1"},
		{"no Origin header (non-browser client)", "dogmud.org", ""},
	}

	for _, tt := range allowed {
		t.Run(tt.name, func(t *testing.T) {
			if !wsOriginAllowed(originRequest(tt.host, tt.origin)) {
				t.Fatalf("origin %q was rejected against host %q", tt.origin, tt.host)
			}
		})
	}
}

// Hostnames the operator has already configured elsewhere are honoured, so a
// deployment whose public domain differs from the Host header it sees (or that
// serves the client from a second domain) works without extra setup.
func TestWsOriginHonoursConfiguredHostnames(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.FilePaths.WebDomain = "dogmud.org"
	cfg.Server.MSSP.Hostname = "mud.dogmud.org"
	cfg.Network.AllowedWebSocketOrigins = configs.ConfigSliceString{"https://client.example"}
	configs.SetConfigForTest(t, cfg)

	// Host is something else entirely (e.g. an internal address), so these can
	// only pass via the configured allow-list, not the same-origin rule.
	for _, origin := range []string{
		"https://dogmud.org",     // FilePaths.WebDomain
		"https://mud.dogmud.org", // Server.MSSP.Hostname
		"https://client.example", // Network.AllowedWebSocketOrigins
	} {
		if !wsOriginAllowed(originRequest("10.0.0.9", origin)) {
			t.Errorf("configured origin %q was rejected", origin)
		}
	}

	// Widening the list must not turn into a wildcard.
	if wsOriginAllowed(originRequest("10.0.0.9", "https://attacker.test")) {
		t.Error("an unlisted origin was allowed once the allow-list was populated")
	}
}

// The shipped default must not be permissive. If the operator never touches
// AllowedWebSocketOrigins, an arbitrary third-party page must still be
// refused.
func TestWsOriginDefaultIsRestrictive(t *testing.T) {
	if wsOriginAllowed(originRequest("dogmud.org", "https://attacker.test")) {
		t.Fatal("default configuration allowed a cross-origin upgrade")
	}
}

func TestHostOnlyNormalisation(t *testing.T) {
	cases := map[string]string{
		"example.com":      "example.com",
		"example.com:8080": "example.com",
		"EXAMPLE.com":      "example.com",
		"[::1]:9000":       "::1",
		"[::1]":            "::1",
		"":                 "",
	}

	for in, want := range cases {
		if got := hostOnly(in); got != want {
			t.Errorf("hostOnly(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHostnameOfNormalisation(t *testing.T) {
	cases := map[string]string{
		"https://example.com":        "example.com",
		"http://example.com:8080":    "example.com",
		"HTTPS://EXAMPLE.COM":        "example.com",
		"ws://[::1]:9000":            "::1",
		"example.com":                "example.com",
		"null":                       "",
		"":                           "",
		"https://example.com/a/path": "example.com",
	}

	for in, want := range cases {
		if got := hostnameOf(in); got != want {
			t.Errorf("hostnameOf(%q) = %q, want %q", in, got, want)
		}
	}
}

// The telnet listener enforces a connection limit; /ws did not, so the cap
// could be bypassed entirely by connecting over the web client — and, because
// the telnet check counts websockets too, an unbounded /ws could starve telnet
// out of the shared pool.
func TestWsCapacityUsesTheSharedConnectionPool(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Network.MaxTelnetConnections = configs.ConfigInt(connections.ActiveConnectionCount() + 1)
	configs.SetConfigForTest(t, cfg)

	if full, active, max := wsCapacityExceeded(); full {
		t.Fatalf("precondition: should have room (active=%d max=%d)", active, max)
	}

	// Registering a connection through the same registry the telnet listener
	// counts must consume the remaining slot.
	clientSide, peerSide := net.Pipe()
	defer clientSide.Close()
	defer peerSide.Close()

	cd := connections.Add(peerSide, nil)
	t.Cleanup(func() { _ = connections.Remove(cd.ConnectionId()) })

	full, active, max := wsCapacityExceeded()
	if !full {
		t.Fatalf("websocket upgrade was admitted past the cap (active=%d max=%d)", active, max)
	}

	// Freeing the slot re-admits.
	if err := connections.Remove(cd.ConnectionId()); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if full, active, max := wsCapacityExceeded(); full {
		t.Fatalf("still full after the connection was removed (active=%d max=%d)", active, max)
	}
}

// A non-positive limit means "unlimited", matching the telnet listener's
// treatment of the localhost admin port.
func TestWsCapacityUnlimitedWhenMaxIsZero(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Network.MaxTelnetConnections = 0
	configs.SetConfigForTest(t, cfg)

	if full, _, _ := wsCapacityExceeded(); full {
		t.Fatal("a max of 0 should mean unlimited, not zero-capacity")
	}
}
