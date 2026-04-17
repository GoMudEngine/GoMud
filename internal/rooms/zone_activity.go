package rooms

import "sync"

var (
	zonePlayerCount   = map[string]int{}
	zonePlayerCountMu sync.RWMutex
)

// ZoneHasPlayers returns true if the named zone has at least one player.
// Use SnapshotActiveZones inside loops to avoid re-locking.
func ZoneHasPlayers(zone string) bool {
	zonePlayerCountMu.RLock()
	defer zonePlayerCountMu.RUnlock()
	return zonePlayerCount[zone] > 0
}

// SnapshotActiveZones returns a set of zones currently containing at
// least one player. Intended for per-round hot loops.
func SnapshotActiveZones() map[string]bool {
	zonePlayerCountMu.RLock()
	defer zonePlayerCountMu.RUnlock()
	out := make(map[string]bool, len(zonePlayerCount))
	for zone, n := range zonePlayerCount {
		if n > 0 {
			out[zone] = true
		}
	}
	return out
}

// incrementZonePlayerCount is called from Room.AddPlayer when a new
// userId is appended to r.players (after dedupe). Unconditional +1;
// does not create the key if absent (Go maps handle that automatically).
func incrementZonePlayerCount(zone string) {
	zonePlayerCountMu.Lock()
	zonePlayerCount[zone]++
	zonePlayerCountMu.Unlock()
}

// decrementZonePlayerCount is called from Room.RemovePlayer when a
// userId is actually removed from r.players. Defensively clamps at
// zero — a call with an already-zero (or missing) key is a no-op, not
// an underflow. Drift from missed increments is still surfaced by
// VerifyZonePlayerCount diagnostics.
func decrementZonePlayerCount(zone string) {
	zonePlayerCountMu.Lock()
	if zonePlayerCount[zone] > 0 {
		zonePlayerCount[zone]--
	}
	zonePlayerCountMu.Unlock()
}

// resetZonePlayerCount clears all state. Test-only helper.
func resetZonePlayerCount() {
	zonePlayerCountMu.Lock()
	zonePlayerCount = map[string]int{}
	zonePlayerCountMu.Unlock()
}

// RebuildZonePlayerCount recomputes the counter from the authoritative
// room.players slices. Intended for server startup (before any player
// is connected) and admin diagnostics (manual trigger during a lull).
//
// NOT safe to call concurrently with live AddPlayer/RemovePlayer — a
// player who enters between the wipe and the scan's read of that room
// will be missed. Startup is single-threaded so the counter is correct.
// Admin use on a live server is best-effort and should be followed by a
// VerifyZonePlayerCount to confirm drift.
func RebuildZonePlayerCount() {
	zonePlayerCountMu.Lock()
	zonePlayerCount = map[string]int{}
	for _, r := range roomManager.rooms {
		if len(r.players) > 0 {
			zonePlayerCount[r.Zone] += len(r.players)
		}
	}
	zonePlayerCountMu.Unlock()
}

// VerifyZonePlayerCount returns the delta (ground truth − incremental
// counter) for every zone where they disagree. An empty map means the
// counter is correct.
func VerifyZonePlayerCount() map[string]int {
	groundTruth := map[string]int{}
	for _, r := range roomManager.rooms {
		if len(r.players) > 0 {
			groundTruth[r.Zone] += len(r.players)
		}
	}

	drift := map[string]int{}
	zonePlayerCountMu.RLock()
	defer zonePlayerCountMu.RUnlock()

	// Every zone from either side must match.
	for zone, truth := range groundTruth {
		if diff := truth - zonePlayerCount[zone]; diff != 0 {
			drift[zone] = diff
		}
	}
	for zone, counter := range zonePlayerCount {
		if _, seen := groundTruth[zone]; !seen && counter != 0 {
			drift[zone] = -counter
		}
	}
	return drift
}
