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
// userId is appended to r.players (after dedupe).
func incrementZonePlayerCount(zone string) {
	zonePlayerCountMu.Lock()
	zonePlayerCount[zone]++
	zonePlayerCountMu.Unlock()
}

// decrementZonePlayerCount is called from Room.RemovePlayer when a
// userId is actually removed from r.players. Clamps at zero defensively —
// drift is still logged via VerifyZonePlayerCount diagnostics.
func decrementZonePlayerCount(zone string) {
	zonePlayerCountMu.Lock()
	if zonePlayerCount[zone] > 0 {
		zonePlayerCount[zone]--
	}
	zonePlayerCountMu.Unlock()
}

// ResetZonePlayerCount is a test helper — clears all state.
func ResetZonePlayerCount() {
	zonePlayerCountMu.Lock()
	zonePlayerCount = map[string]int{}
	zonePlayerCountMu.Unlock()
}

// RebuildZonePlayerCount recomputes the counter from the authoritative
// room.players slices. Called at server startup and from the admin
// diagnostics page.
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
