package web

import "net/http"

// serveBuildPage renders the admin room-builder page (admin web-building 1b).
//
// Admin gating is applied by the doBasicAuth wrapper at registration time
// (web.go), so by the time this runs the request is an authenticated RoleAdmin
// user. It serves build.html through the standard template pipeline
// (serveTemplate maps the extension-less /build path to build.html), so the
// page can resolve {{ .CONFIG }} values — notably the static-asset base path —
// exactly like the other web pages.
//
// The page is otherwise self-contained: it opens its own WebSocket → GMCP
// session and drives the world through the RoleAdmin-gated Build.* packages.
func serveBuildPage(w http.ResponseWriter, r *http.Request) {
	serveTemplate(w, r)
}
