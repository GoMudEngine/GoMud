package web

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
)

// Admission control for websocket upgrades.
//
// Two gaps this closes:
//
//   - The telnet listener enforces a connection limit (main.go, checked against
//     connections.ActiveConnectionCount()); /ws did not, so the cap could be
//     bypassed entirely by connecting over the web client. Note the telnet
//     check counts ALL connections including websockets, which means an
//     unbounded /ws could also starve telnet out of its own pool.
//
//   - CheckOrigin returned true unconditionally, so any third-party page could
//     open sockets from a visitor's browser and drive them against the server.

// wsCapacityExceeded reports whether admitting another websocket would exceed
// the server's connection limit.
//
// It reuses Network.MaxTelnetConnections rather than introducing a websocket
// knob, because that is what the limit already means in practice: the telnet
// listener compares it against connections.ActiveConnectionCount(), which
// counts every registered connection, websockets included. Adding a separate
// websocket cap would let the two paths disagree about a pool they share.
//
// A configured value of 0 or less means "no limit", matching the telnet
// listener's behaviour for the localhost admin port.
//
// Read from config on each call rather than from the value captured at startup,
// so a config reload takes effect without a restart.
func wsCapacityExceeded() (exceeded bool, active int, max int) {
	max = int(configs.GetNetworkConfig().MaxTelnetConnections)
	active = connections.ActiveConnectionCount()

	if max <= 0 {
		return false, active, max
	}
	return active >= max, active, max
}

// wsOriginAllowed decides whether an upgrade request may proceed.
//
// Rules, in order:
//
//  1. No Origin header — allow. Non-browser websocket clients (bots, the
//     playtest harness, curl) legitimately omit it, and its absence is not a
//     cross-origin signal: browsers always send it on a websocket handshake.
//  2. Origin host matches the Host the request was addressed to — allow. This
//     is the same-origin case and covers every normal player, including behind
//     a reverse proxy, since Caddy preserves the Host header.
//  3. Origin host matches a hostname the operator has already configured
//     (FilePaths.WebDomain, Server.MSSP.Hostname) or an explicit entry in
//     Network.AllowedWebSocketOrigins — allow.
//  4. Anything else — reject.
//
// Comparison is on hostname only; scheme and port are ignored. Requiring a port
// match would break the common dev setup (page on :80, config naming a bare
// host) without adding protection, because an attacker who controls a port on
// an allowed hostname already controls that origin.
func wsOriginAllowed(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get(`Origin`))
	if origin == `` {
		return true
	}

	originHost := hostnameOf(origin)
	if originHost == `` {
		// An Origin we cannot parse is not one we can vouch for.
		return false
	}

	if originHost == hostOnly(r.Host) {
		return true
	}

	for _, allowed := range allowedWSOriginHosts() {
		if originHost == allowed {
			return true
		}
	}

	return false
}

// allowedWSOriginHosts is the operator-configured allow-list, normalised to
// bare lowercase hostnames. Loopback names are always included so a developer
// running the client locally is never locked out of their own server.
func allowedWSOriginHosts() []string {
	hosts := []string{`localhost`, `127.0.0.1`, `::1`}

	if d := hostOnly(string(configs.GetFilePathsConfig().WebDomain)); d != `` {
		hosts = append(hosts, d)
	}
	if h := hostOnly(string(configs.GetServerConfig().MSSP.Hostname)); h != `` {
		hosts = append(hosts, h)
	}
	for _, extra := range configs.GetNetworkConfig().AllowedWebSocketOrigins {
		// Accept either a bare hostname or a full origin URL.
		if h := hostnameOf(extra); h != `` {
			hosts = append(hosts, h)
		} else if h := hostOnly(extra); h != `` {
			hosts = append(hosts, h)
		}
	}

	return hosts
}

// hostnameOf extracts the lowercase hostname from an origin. Accepts a full
// URL ("https://example.com:443") or a bare host ("example.com").
func hostnameOf(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == `` || origin == `null` {
		return ``
	}

	if strings.Contains(origin, `://`) {
		u, err := url.Parse(origin)
		if err != nil {
			return ``
		}
		return strings.ToLower(u.Hostname())
	}

	return hostOnly(origin)
}

// hostOnly strips a trailing port and IPv6 brackets from a host value and
// lowercases it.
func hostOnly(host string) string {
	host = strings.TrimSpace(host)
	if host == `` {
		return ``
	}

	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimPrefix(host, `[`)
	host = strings.TrimSuffix(host, `]`)

	return strings.ToLower(host)
}
