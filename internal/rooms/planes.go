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

// NextFreeAuthoredPlane returns an unused authored plane id — the maximum
// authored plane currently in use (below the instance-plane base) plus one.
// The web builder stamps this on a newly created zone so its rooms occupy a
// fresh coordinate space and never collide with the overworld (plane 0) or
// another zone in the shared (plane,x,y,z) placement check.
func NextFreeAuthoredPlane() int {
	maxPlane := 0
	for _, roomId := range GetAllRoomIds() {
		if room := LoadRoom(roomId); room != nil && room.Plane < instancePlaneBase && room.Plane > maxPlane {
			maxPlane = room.Plane
		}
	}
	return maxPlane + 1
}

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
