package forager

import (
	"sort"
	"sync"
)

// chestIndex maps a zone to the set of forager storage-lockbox room IDs known
// in that zone. Self-populated by RegisterChestRoom as foragers reach their
// storing state, so the YAML-authored storage_chest_room stays the single
// source of truth (no duplication into the static profiles registry, no
// per-tick all-instance scan). Chest rooms are fixed for a server's lifetime,
// so the set only grows.
var (
	chestIndexMu sync.RWMutex
	chestIndex   = map[string]map[int]bool{}
)

// RegisterChestRoom records that zone has a forager lockbox in chestRoom.
// Idempotent. No-op for zero values.
func RegisterChestRoom(zone string, chestRoom int) {
	if zone == "" || chestRoom == 0 {
		return
	}
	chestIndexMu.Lock()
	defer chestIndexMu.Unlock()
	set := chestIndex[zone]
	if set == nil {
		set = map[int]bool{}
		chestIndex[zone] = set
	}
	set[chestRoom] = true
}

// ChestRoomsForZone returns the chest room IDs registered for zone (stable
// order by room id for determinism).
func ChestRoomsForZone(zone string) []int {
	chestIndexMu.RLock()
	defer chestIndexMu.RUnlock()
	set := chestIndex[zone]
	if len(set) == 0 {
		return nil
	}
	out := make([]int, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}
