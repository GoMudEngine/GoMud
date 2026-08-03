package web

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const (
	// authCacheTTL is the hard lifetime of a successful admin authentication.
	// Past this the full bcrypt path runs again regardless of revalidation.
	authCacheTTL = 30 * time.Minute

	// authRevalidateEvery bounds how stale a cached grant may be with respect
	// to the *user record*. The old cache keyed on the raw Authorization header
	// and never looked at the account again for 30 minutes, so a password
	// change, a demotion out of RoleAdmin, or an outright account deletion left
	// a working admin session behind it. Re-reading the record is far cheaper
	// than bcrypt but is still a disk read + YAML parse, and /admin/static/*
	// runs this middleware once per asset, so it is rate-limited rather than
	// done on every request.
	authRevalidateEvery = 30 * time.Second

	// Failed-attempt throttling. doBasicAuth runs bcrypt against real player
	// credentials, which is deliberately expensive; unthrottled it is both a
	// password-guessing oracle and a cheap CPU-exhaustion vector.
	authMaxFailures   = 5
	authFailureWindow = 5 * time.Minute
	authLockoutDur    = 1 * time.Minute

	// authTrackerMaxEntries caps the failure map. Its key is derived from
	// attacker-controlled connection data, so it must never grow without
	// bound: entries are swept on every write and, if the cap is still hit,
	// the least-recently-failed entry is evicted to make room.
	authTrackerMaxEntries = 4096
)

// authCacheEntry records everything needed to re-check a cached grant without
// paying for bcrypt again. username/role/pwHash are the identity the grant was
// issued against; if any of them no longer matches the stored record, the grant
// is void.
type authCacheEntry struct {
	expires        time.Time
	nextRevalidate time.Time
	username       string
	role           string
	pwHash         string
}

type authFailureRecord struct {
	count       int
	windowStart time.Time
	lastFail    time.Time
	lockedUntil time.Time
}

var (
	// authMu guards BOTH maps below.
	//
	// These were previously unguarded. Every /admin/* route happens to be
	// wrapped in RunWithMUDLocked, which serialized them by accident, but
	// GET /build and GET /build-help are deliberately not (the build page must
	// not hold the world lock). An unauthenticated flood of /build therefore
	// raced any admin authenticating elsewhere in the admin UI and killed the
	// process with "concurrent map read and map write".
	authMu       sync.Mutex
	authCache    = map[string]authCacheEntry{}
	authFailures = map[string]*authFailureRecord{}
)

func handlerToHandlerFunc(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}
}

// authCacheKey hashes the Authorization header rather than using it directly.
// The header is base64(user:password), so using it as a map key kept plaintext
// credentials alive in a long-lived map. The hash still uniquely binds the
// grant to the exact credential that was presented.
func authCacheKey(authHeader string) string {
	sum := sha256.Sum256([]byte(authHeader))
	return hex.EncodeToString(sum[:])
}

// authSourceIP is the throttling key. It is the direct peer for now; see
// clientIPFromRequest, which replaces this with the proxy-aware resolution so
// that a reverse-proxied deployment does not collapse every admin onto one
// bucket.
func authSourceIP(r *http.Request) string {
	return clientIPFromRequest(r)
}

// authThrottled reports whether this source is currently locked out. Called
// before any bcrypt work so that a lockout actually saves the CPU.
func authThrottled(ip string, now time.Time) (bool, time.Duration) {
	authMu.Lock()
	defer authMu.Unlock()

	rec, ok := authFailures[ip]
	if !ok {
		return false, 0
	}
	if now.Before(rec.lockedUntil) {
		return true, rec.lockedUntil.Sub(now)
	}
	return false, 0
}

// noteAuthFailure records a failed attempt and locks the source out once it
// exceeds authMaxFailures inside authFailureWindow.
func noteAuthFailure(ip string, now time.Time) {
	authMu.Lock()
	defer authMu.Unlock()

	sweepAuthFailuresLocked(now)

	rec, ok := authFailures[ip]
	if !ok {
		if len(authFailures) >= authTrackerMaxEntries {
			evictOldestAuthFailureLocked()
		}
		rec = &authFailureRecord{windowStart: now}
		authFailures[ip] = rec
	}

	// A quiet period longer than the window starts a fresh count.
	if now.Sub(rec.windowStart) > authFailureWindow {
		rec.count = 0
		rec.windowStart = now
	}

	rec.count++
	rec.lastFail = now

	if rec.count >= authMaxFailures {
		rec.lockedUntil = now.Add(authLockoutDur)
		mudlog.Warn("ADMIN AUTH THROTTLE", "ip", ip, "failures", rec.count, "lockedForSeconds", int(authLockoutDur.Seconds()))
	}
}

// clearAuthFailures drops the failure record for a source that just
// authenticated successfully.
func clearAuthFailures(ip string) {
	authMu.Lock()
	defer authMu.Unlock()
	delete(authFailures, ip)
}

// sweepAuthFailuresLocked drops records that are outside the failure window and
// not currently locked out. Caller must hold authMu.
func sweepAuthFailuresLocked(now time.Time) {
	for ip, rec := range authFailures {
		if now.Before(rec.lockedUntil) {
			continue
		}
		if now.Sub(rec.lastFail) > authFailureWindow {
			delete(authFailures, ip)
		}
	}
}

// evictOldestAuthFailureLocked makes room when the tracker is at capacity.
// Caller must hold authMu.
func evictOldestAuthFailureLocked() {
	var oldestIP string
	var oldest time.Time
	for ip, rec := range authFailures {
		if oldestIP == `` || rec.lastFail.Before(oldest) {
			oldestIP = ip
			oldest = rec.lastFail
		}
	}
	if oldestIP != `` {
		delete(authFailures, oldestIP)
	}
}

// authCacheLookup returns true when a cached grant is still valid. It also
// re-reads the user record at most once per authRevalidateEvery so that a
// password change, a role demotion, or a deleted account invalidates an
// outstanding grant instead of surviving for the rest of the TTL.
func authCacheLookup(key string, now time.Time) bool {
	authMu.Lock()
	entry, ok := authCache[key]
	if !ok {
		authMu.Unlock()
		return false
	}
	if !now.Before(entry.expires) {
		delete(authCache, key)
		authMu.Unlock()
		return false
	}
	if now.Before(entry.nextRevalidate) {
		authMu.Unlock()
		return true
	}
	authMu.Unlock()

	// Revalidate outside the lock — LoadUser touches the filesystem and must
	// not be run while holding a mutex that every request contends on.
	uRecord, err := users.LoadUser(entry.username, true)
	if err != nil || uRecord.Role != users.RoleAdmin || uRecord.Password != entry.pwHash {
		authMu.Lock()
		// Only drop the entry if it is still the one we validated; a
		// concurrent successful auth may have replaced it.
		if cur, still := authCache[key]; still && cur.username == entry.username && cur.pwHash == entry.pwHash {
			delete(authCache, key)
		}
		authMu.Unlock()

		reason := `credentials changed`
		if err != nil {
			reason = `user record unreadable`
		} else if uRecord.Role != users.RoleAdmin {
			reason = `role is no longer admin`
		}
		mudlog.Warn("ADMIN AUTH REVOKED", "username", entry.username, "reason", reason)
		return false
	}

	authMu.Lock()
	if cur, still := authCache[key]; still {
		cur.nextRevalidate = now.Add(authRevalidateEvery)
		authCache[key] = cur
	}
	authMu.Unlock()

	return true
}

// authCacheStore records a fresh grant, sweeping expired entries first. The map
// only ever gains an entry on a *successful* admin authentication, so it is
// naturally bounded by the number of admin credentials, but it is swept anyway
// so that rotated credentials do not accumulate.
func authCacheStore(key string, uRecord *users.UserRecord, now time.Time) {
	authMu.Lock()
	defer authMu.Unlock()

	for k, e := range authCache {
		if !now.Before(e.expires) {
			delete(authCache, k)
		}
	}

	authCache[key] = authCacheEntry{
		expires:        now.Add(authCacheTTL),
		nextRevalidate: now.Add(authRevalidateEvery),
		username:       uRecord.Username,
		role:           uRecord.Role,
		pwHash:         uRecord.Password,
	}
}

func doBasicAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		now := time.Now()
		authHeader := r.Header.Get("Authorization")

		// An already-granted credential short-circuits before the throttle so
		// that an authenticated admin is not punished for someone else's
		// guessing. Producing a cache hit requires having already passed the
		// full check with this exact credential.
		if authHeader != `` && authCacheLookup(authCacheKey(authHeader), now) {
			next.ServeHTTP(w, r)
			return
		}

		// Everything past here is expensive (bcrypt), so throttle first.
		sourceIP := authSourceIP(r)
		if locked, retryIn := authThrottled(sourceIP, now); locked {
			secs := int(retryIn.Seconds()) + 1
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			http.Error(w, "Too many failed authentication attempts", http.StatusTooManyRequests)
			return
		}

		// Extract the username and password from the request
		// Authorization header. If no Authentication header is present
		// or the header value is invalid, then the 'ok' return value
		// will be false.
		username, password, ok := r.BasicAuth()
		if ok {

			// Authorize against actual user record
			uRecord, err := users.LoadUser(username, true)
			if err == nil {

				if uRecord.PasswordMatches(password) {

					if uRecord.Role == users.RoleAdmin {

						mudlog.Warn("ADMIN LOGIN", "username", username, "success", true)

						// Cache auth for 30 minutes to avoid re-auth every load
						authCacheStore(authCacheKey(authHeader), uRecord, now)
						clearAuthFailures(sourceIP)

						next.ServeHTTP(w, r)
						return

					} else {

						mudlog.Error("ADMIN LOGIN", "username", username, "success", false, "error", `Role=`+uRecord.Role)

					}
				}

			} else {
				mudlog.Error("ADMIN LOGIN", "username", username, "success", false, "error", err)
			}
		}

		// Anything that reaches here presented no credential, a malformed one,
		// a wrong password, or a non-admin account. All of those count toward
		// the per-source lockout.
		noteAuthFailure(sourceIP, now)

		// If the Authentication header is not present, is invalid, or the
		// username or password is wrong, then set a WWW-Authenticate
		// header to inform the client that we expect them to use basic
		// authentication and send a 401 Unauthorized response.
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// clientIPFromRequest resolves the source address used for auth throttling.
//
// This is the direct socket peer. Defect 2 (trusted-proxy resolution) replaces
// the body of this function so that a reverse-proxied deployment throttles on
// the real client rather than collapsing every admin onto the proxy's address.
func clientIPFromRequest(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
