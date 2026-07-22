package rooms

import "sync"

// PlaneInfo describes one coordinate-space plane.
type PlaneInfo struct {
	NonEuclidean bool
	Label        string
}

// PlaneRegistry maps plane id -> PlaneInfo. A plane is non-Euclidean if ANY
// contributing zone/room marks it so (OR semantics). Unknown planes default to
// Euclidean, so enforcement is the safe default.
type PlaneRegistry struct {
	mu     sync.RWMutex
	planes map[int]PlaneInfo
}

func NewPlaneRegistry() *PlaneRegistry {
	return &PlaneRegistry{planes: map[int]PlaneInfo{}}
}

func (r *PlaneRegistry) Mark(plane int, nonEuclidean bool, label string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur := r.planes[plane]
	cur.NonEuclidean = cur.NonEuclidean || nonEuclidean
	if label != "" {
		cur.Label = label
	}
	r.planes[plane] = cur
}

func (r *PlaneRegistry) IsNonEuclidean(plane int) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.planes[plane].NonEuclidean
}

// planeRegistry is the process-wide registry, (re)built after rooms load.
var planeRegistry = NewPlaneRegistry()

// GetPlaneRegistry returns the process registry (consumers: mapper enforcement,
// ValidatePlacement).
func GetPlaneRegistry() *PlaneRegistry { return planeRegistry }

// RebuildPlaneRegistry walks every loaded room and marks its plane
// non-Euclidean when the room's zone is non_cartesian. Call after all rooms +
// zone configs are loaded (from the same boot pass that runs PreCacheMaps).
func RebuildPlaneRegistry() {
	reg := NewPlaneRegistry()
	for _, roomId := range GetAllRoomIds() {
		room := LoadRoom(roomId)
		if room == nil {
			continue
		}
		reg.Mark(room.Plane, IsZoneNonCartesian(room.Zone), room.Zone)
	}
	planeRegistry = reg
}
