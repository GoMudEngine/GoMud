package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// resetTrustedProxyCacheForTest forces the next resolution to re-read config.
func resetTrustedProxyCacheForTest() {
	trustedProxyMu.Lock()
	defer trustedProxyMu.Unlock()
	trustedProxyKey = "\x00sentinel-never-matches\x00"
	trustedProxyNets = nil
	trustedProxyExact = nil
}

func requestFrom(remoteAddr string, xff ...string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/ws", nil)
	r.RemoteAddr = remoteAddr
	for _, v := range xff {
		r.Header.Add("X-Forwarded-For", v)
	}
	return r
}

// The whole point of the trusted-proxy gate: a remote client's own
// X-Forwarded-For must never be believed, because it is trivially forged and
// loopback is exempt from ban checks for everyone.
func TestResolveClientIPIgnoresForgedHeaderFromUntrustedPeer(t *testing.T) {
	resetTrustedProxyCacheForTest()
	defer resetTrustedProxyCacheForTest()

	tests := []struct {
		name string
		xff  string
	}{
		{"forged loopback", "127.0.0.1"},
		{"forged ipv6 loopback", "::1"},
		{"forged chain ending in loopback", "8.8.8.8, 127.0.0.1"},
		{"forged chain ending in another victim", "10.0.0.5, 203.0.113.99"},
		{"garbage", "not-an-ip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveClientIP(requestFrom("198.51.100.4:41234", tt.xff))
			if got != "198.51.100.4" {
				t.Fatalf("ResolveClientIP = %q, want the real peer 198.51.100.4 (header must be ignored)", got)
			}
		})
	}
}

// The bug being fixed: behind Caddy the socket peer is loopback for every
// web-client player, so the forwarded address must be used.
func TestResolveClientIPUsesForwardedFromTrustedPeer(t *testing.T) {
	resetTrustedProxyCacheForTest()
	defer resetTrustedProxyCacheForTest()

	got := ResolveClientIP(requestFrom("127.0.0.1:52000", "203.0.113.77"))
	if got != "203.0.113.77" {
		t.Fatalf("ResolveClientIP = %q, want 203.0.113.77", got)
	}
}

// A client that pre-seeds X-Forwarded-For sits to the LEFT of what the proxy
// appends, so the rightmost-untrusted walk must skip past the forgery.
func TestResolveClientIPTakesRightmostUntrustedHop(t *testing.T) {
	resetTrustedProxyCacheForTest()
	defer resetTrustedProxyCacheForTest()

	tests := []struct {
		name string
		xff  []string
		want string
	}{
		{
			name: "client forged a loopback prefix, proxy appended the truth",
			xff:  []string{"127.0.0.1, 203.0.113.77"},
			want: "203.0.113.77",
		},
		{
			name: "client forged someone else, proxy appended the truth",
			xff:  []string{"8.8.8.8, 203.0.113.77"},
			want: "203.0.113.77",
		},
		{
			name: "forgery split across multiple header lines",
			xff:  []string{"8.8.8.8", "203.0.113.77"},
			want: "203.0.113.77",
		},
		{
			name: "trailing trusted hops are skipped",
			xff:  []string{"203.0.113.77, 127.0.0.1, ::1"},
			want: "203.0.113.77",
		},
		{
			name: "hop carrying a port",
			xff:  []string{"203.0.113.77:9999"},
			want: "203.0.113.77",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveClientIP(requestFrom("127.0.0.1:52000", tt.xff...))
			if got != tt.want {
				t.Fatalf("ResolveClientIP = %q, want %q", got, tt.want)
			}
		})
	}
}

// With no header, or with an unusable one, a trusted peer resolves to itself —
// the pre-existing behaviour, not a bypass.
func TestResolveClientIPFallsBackToPeer(t *testing.T) {
	resetTrustedProxyCacheForTest()
	defer resetTrustedProxyCacheForTest()

	tests := []struct {
		name string
		xff  []string
	}{
		{"no header at all", nil},
		{"empty header", []string{""}},
		{"every hop is itself a trusted proxy", []string{"127.0.0.1, ::1"}},
		{"unparseable rightmost hop", []string{"203.0.113.77, junk"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveClientIP(requestFrom("127.0.0.1:52000", tt.xff...))
			if got != "127.0.0.1" {
				t.Fatalf("ResolveClientIP = %q, want 127.0.0.1", got)
			}
		})
	}
}

// The shipped default must trust loopback only. Anything broader (a private
// range, a wildcard) would let a LAN host forge a source address.
func TestTrustedProxyDefaultIsLoopbackOnly(t *testing.T) {
	resetTrustedProxyCacheForTest()
	defer resetTrustedProxyCacheForTest()

	for _, entry := range configs.GetNetworkConfig().TrustedProxies {
		switch entry {
		case "127.0.0.1/32", "::1/128", "127.0.0.1", "::1":
		default:
			t.Fatalf("Network.TrustedProxies contains %q; the default must be loopback only", entry)
		}
	}

	for _, addr := range []string{"10.0.0.1:1", "192.168.1.5:1", "172.16.0.9:1", "203.0.113.4:1"} {
		if got := ResolveClientIP(requestFrom(addr, "127.0.0.1")); got == "127.0.0.1" {
			t.Fatalf("peer %s was treated as a trusted proxy", addr)
		}
	}
}
