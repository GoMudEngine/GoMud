package web

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// Real-client-IP resolution behind a reverse proxy.
//
// Production fronts the web client with Caddy, so the TCP peer of every
// websocket upgrade and every admin request is the proxy (127.0.0.1), not the
// player. That made IP bans inert for anyone using /webclient — IsLocal()
// returned true for every web-client connection and the ban check in
// FinalizeLoginOrCreate was skipped entirely — and it collapsed admin auth
// throttling onto a single bucket.
//
// X-Forwarded-For is attacker-controlled. Believing it unconditionally would
// be strictly worse than the bug: anyone could send
// `X-Forwarded-For: 127.0.0.1` and be treated as loopback, which is exempt
// from ban checks for everyone. Two rules keep that from happening:
//
//  1. The header is read ONLY when the immediate TCP peer is in
//     Network.TrustedProxies (loopback by default). A remote attacker cannot
//     forge their socket peer address — that would require completing a TCP
//     handshake and an HTTP upgrade from a spoofed source.
//  2. Within the header, the RIGHTMOST hop that is not itself a trusted proxy
//     wins. Proxies append; so anything a client pre-seeds sits to the LEFT of
//     the address the proxy observed and is never reached.
//
// This assumes the proxy APPENDS to X-Forwarded-For rather than replacing it,
// which is Caddy's `reverse_proxy` default. A proxy configured to overwrite the
// header with the client's own value defeats any XFF scheme, not just this one.

var (
	trustedProxyMu    sync.Mutex
	trustedProxyKey   string
	trustedProxyNets  []*net.IPNet
	trustedProxyExact []net.IP
)

// trustedProxyMatchers returns the parsed Network.TrustedProxies entries,
// re-parsing only when the configured list changes.
func trustedProxyMatchers() ([]*net.IPNet, []net.IP) {
	cfg := configs.GetNetworkConfig().TrustedProxies
	key := strings.Join(cfg, "\x00")

	trustedProxyMu.Lock()
	defer trustedProxyMu.Unlock()

	if key == trustedProxyKey && (trustedProxyNets != nil || trustedProxyExact != nil) {
		return trustedProxyNets, trustedProxyExact
	}

	nets := []*net.IPNet{}
	exact := []net.IP{}

	for _, entry := range cfg {
		entry = strings.TrimSpace(entry)
		if entry == `` {
			continue
		}
		if _, ipNet, err := net.ParseCIDR(entry); err == nil {
			nets = append(nets, ipNet)
			continue
		}
		if ip := net.ParseIP(entry); ip != nil {
			exact = append(exact, ip)
		}
		// Anything unparseable is dropped rather than treated as a wildcard.
	}

	trustedProxyKey = key
	trustedProxyNets = nets
	trustedProxyExact = exact

	return nets, exact
}

// isTrustedProxy reports whether an address may have its X-Forwarded-For
// header believed.
func isTrustedProxy(ip net.IP) bool {
	if ip == nil {
		return false
	}

	nets, exact := trustedProxyMatchers()

	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	for _, e := range exact {
		if e.Equal(ip) {
			return true
		}
	}
	return false
}

// parseHopIP accepts the forms a proxy might write into X-Forwarded-For: a bare
// IP, a bracketed IPv6 literal, or either with a port appended. Returns nil if
// the value is not an IP at all.
func parseHopIP(raw string) net.IP {
	raw = strings.TrimSpace(raw)
	if raw == `` {
		return nil
	}

	if ip := net.ParseIP(raw); ip != nil {
		return ip
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip
		}
	}
	// "[::1]" with no port.
	if strings.HasPrefix(raw, `[`) && strings.HasSuffix(raw, `]`) {
		if ip := net.ParseIP(raw[1 : len(raw)-1]); ip != nil {
			return ip
		}
	}
	return nil
}

// peerIP is the address of the machine actually holding the socket. This is the
// only address in the request that cannot be forged.
func peerIP(remoteAddr string) net.IP {
	if ip := parseHopIP(remoteAddr); ip != nil {
		return ip
	}
	return nil
}

// forwardedHops flattens X-Forwarded-For into hop order (leftmost = claimed
// original client, rightmost = the hop the nearest proxy observed). Multiple
// header lines are concatenated in the order Go received them.
func forwardedHops(h http.Header) []string {
	var hops []string
	for _, v := range h.Values(`X-Forwarded-For`) {
		for _, part := range strings.Split(v, `,`) {
			part = strings.TrimSpace(part)
			if part != `` {
				hops = append(hops, part)
			}
		}
	}
	return hops
}

// ResolveClientIP returns the best-supported source address for a request: the
// socket peer unless that peer is a trusted proxy, in which case the rightmost
// non-proxy hop from X-Forwarded-For.
//
// Returns a bare IP string (no port), or the raw RemoteAddr if it could not be
// parsed at all.
func ResolveClientIP(r *http.Request) string {
	peer := peerIP(r.RemoteAddr)
	if peer == nil {
		// Nothing parseable — hand back what we were given rather than
		// inventing an address.
		return r.RemoteAddr
	}

	// The header is only meaningful when the machine that sent it is one we
	// trust to have written it honestly.
	if !isTrustedProxy(peer) {
		return peer.String()
	}

	hops := forwardedHops(r.Header)
	for i := len(hops) - 1; i >= 0; i-- {
		ip := parseHopIP(hops[i])
		if ip == nil {
			// A hop we cannot parse means we cannot attribute the request.
			// Fall back to the one address we actually observed rather than
			// walking further left into client-supplied territory.
			break
		}
		if isTrustedProxy(ip) {
			continue
		}
		return ip.String()
	}

	return peer.String()
}
